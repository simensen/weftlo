// Package routing provides services for transforming content source paths to target installation paths.
// This file implements template evaluation for target paths with variable substitution.
package routing

import (
	"bytes"
	"path"
	"regexp"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig/v3"
)

// TargetEvaluator evaluates target path templates using Go template syntax.
// It supports variable substitution via {{ .Variables.key }} and all Sprig
// template functions (e.g., {{ env "USER" }}, {{ lower .Variables.name }}).
//
// The evaluator is designed for use during router construction to pre-evaluate
// all target templates once, storing the resolved paths on the router.
//
// Template evaluation follows these rules:
//   - Templates use standard Go template syntax with {{ }} delimiters
//   - Variables are accessed via .Variables map (e.g., {{ .Variables.namespace }})
//   - All Sprig functions are available (100+ functions)
//   - Missing variables cause immediate failure with MissingVariableError
//   - Template syntax errors cause failure with TargetTemplateError
//
// Path normalization and validation:
//   - Leading/trailing whitespace is trimmed from the final path
//   - Consecutive slashes are collapsed (e.g., ".claude//agents" -> ".claude/agents")
//   - Absolute paths (starting with /) are rejected
//   - Path traversal attempts (containing ..) are rejected
//   - Reserved path values (. or ..) are rejected
type TargetEvaluator struct {
	// variables contains the merged variables from global, profile, and project configs.
	// These are accessible in templates via {{ .Variables.key }}.
	variables map[string]interface{}

	// funcMap contains the template functions available during evaluation.
	funcMap template.FuncMap
}

// targetTemplateContext provides the data structure accessible to target templates.
// This is a minimal context with only the Variables field populated.
type targetTemplateContext struct {
	// Variables contains the merged custom variables from all configuration sources.
	Variables map[string]interface{}
}

// NewTargetEvaluator creates a new TargetEvaluator with the given variables.
//
// Parameters:
//   - variables: Merged variables from global, profile, and project configs.
//     These are accessible in templates via {{ .Variables.key }}.
//     May be nil if no variables are defined.
//
// Returns a pointer to a new TargetEvaluator ready for template evaluation.
func NewTargetEvaluator(variables map[string]interface{}) *TargetEvaluator {
	// Ensure variables map is never nil to simplify template access
	if variables == nil {
		variables = make(map[string]interface{})
	}

	return &TargetEvaluator{
		variables: variables,
		funcMap:   sprig.TxtFuncMap(),
	}
}

// Evaluate evaluates a single target path template and returns the resolved path.
//
// Parameters:
//   - targetName: The name of the target (used in error messages)
//   - targetTemplate: The template string to evaluate (e.g., ".{{ .Variables.namespace }}")
//
// Returns:
//   - The evaluated path string
//   - An error if evaluation fails:
//   - TargetTemplateError for syntax errors
//   - MissingVariableError for undefined variables (fail-fast behavior)
//   - InvalidTargetPathError for paths that fail validation
//
// The evaluation process:
//  1. Parse the template string with Sprig functions
//  2. Execute the template with the variables context
//  3. Detect missing variables via missingkey=error option
//  4. Normalize the path (trim whitespace, clean path)
//  5. Validate the path (reject absolute paths, traversal, reserved values)
//  6. Return the evaluated and validated string result
//
// Example:
//
//	evaluator := NewTargetEvaluator(map[string]interface{}{"namespace": "acme"})
//	path, err := evaluator.Evaluate("claude", ".{{ .Variables.namespace }}/claude")
//	// path = ".acme/claude"
func (e *TargetEvaluator) Evaluate(targetName, targetTemplate string) (string, error) {
	var result string

	// Check if the template contains any template delimiters
	// If not, it's a literal path - skip template processing
	if !strings.Contains(targetTemplate, "{{") {
		result = targetTemplate
	} else {
		// Parse the template with Sprig functions and strict variable checking
		tmpl, err := template.New("target").
			Funcs(e.funcMap).
			Option("missingkey=error").
			Parse(targetTemplate)
		if err != nil {
			return "", &TargetTemplateError{
				TargetName:     targetName,
				TemplateString: targetTemplate,
				Err:            err,
			}
		}

		// Create the context with variables
		ctx := &targetTemplateContext{
			Variables: e.variables,
		}

		// Execute the template
		var buf bytes.Buffer
		if err := tmpl.Execute(&buf, ctx); err != nil {
			// Check if this is a missing variable error
			if varName := extractMissingVariableName(err); varName != "" {
				return "", &MissingVariableError{
					TargetName:     targetName,
					VariableName:   varName,
					TemplateString: targetTemplate,
				}
			}

			// Other execution errors are wrapped as template errors
			return "", &TargetTemplateError{
				TargetName:     targetName,
				TemplateString: targetTemplate,
				Err:            err,
			}
		}

		result = buf.String()
	}

	// Apply path normalization and validation
	normalized, err := normalizeAndValidatePath(targetName, result)
	if err != nil {
		return "", err
	}

	return normalized, nil
}

// EvaluateAll evaluates all target templates in the provided map and returns
// a new map with evaluated paths.
//
// Parameters:
//   - targets: Map of target names to template strings
//
// Returns:
//   - A new map with the same keys but evaluated path values
//   - An error if any template evaluation fails (fail-fast on first error)
//
// The method iterates through all targets and evaluates each one. If any
// evaluation fails, the method returns immediately with that error.
//
// Example:
//
//	evaluator := NewTargetEvaluator(map[string]interface{}{"ns": "acme"})
//	targets := map[string]string{
//	    "claude": ".{{ .Variables.ns }}/claude",
//	    "cursor": ".cursor",
//	}
//	evaluated, err := evaluator.EvaluateAll(targets)
//	// evaluated = {"claude": ".acme/claude", "cursor": ".cursor"}
func (e *TargetEvaluator) EvaluateAll(targets map[string]string) (map[string]string, error) {
	if targets == nil {
		return make(map[string]string), nil
	}

	evaluated := make(map[string]string, len(targets))

	for name, templateStr := range targets {
		result, err := e.Evaluate(name, templateStr)
		if err != nil {
			return nil, err
		}
		evaluated[name] = result
	}

	return evaluated, nil
}

// normalizeAndValidatePath applies path normalization and validation to an evaluated path.
//
// The process:
//  1. Trim leading and trailing whitespace
//  2. Validate path security (before path.Clean which might hide attacks)
//  3. Apply path.Clean() for normalization
//
// This order is important: validation BEFORE path.Clean() ensures we detect
// path traversal attempts like ".claude/../secrets" which path.Clean() would
// normalize to "secrets", hiding the attack.
func normalizeAndValidatePath(targetName, result string) (string, error) {
	// Step 1: Trim leading/trailing whitespace
	result = strings.TrimSpace(result)

	// Handle empty result after trimming
	if result == "" {
		return result, nil
	}

	// Step 2: Validate path security BEFORE path.Clean()
	// This ensures we catch traversal attacks that path.Clean() would hide
	if err := validatePathSecurity(targetName, result); err != nil {
		return "", err
	}

	// Step 3: Apply path.Clean() for path normalization
	// path.Clean() handles consecutive slashes, trailing slashes, and . components
	result = path.Clean(result)

	// Step 4: Final validation after normalization
	// Re-check for issues that might appear after normalization
	if err := validatePathFinal(targetName, result); err != nil {
		return "", err
	}

	return result, nil
}

// validatePathSecurity checks for security issues in a path BEFORE normalization.
// This catches attacks that path.Clean() would hide.
//
// Validation rules:
//  1. Path must not start with "/" (must be relative)
//  2. Path must not contain ".." as a component (no traversal)
func validatePathSecurity(targetName, result string) error {
	// Rule 1: Check for absolute paths
	if strings.HasPrefix(result, "/") {
		return &InvalidTargetPathError{
			TargetName: targetName,
			ResultPath: result,
			Reason:     "path must be relative, not absolute",
		}
	}

	// Rule 2: Check for path traversal (..)
	// We check BEFORE path.Clean() to catch attempts like ".claude/../secrets"
	if containsTraversalComponent(result) {
		return &InvalidTargetPathError{
			TargetName: targetName,
			ResultPath: result,
			Reason:     "path contains forbidden traversal component '..'",
		}
	}

	return nil
}

// validatePathFinal performs final validation after path normalization.
//
// Validation rules:
//  1. Path must not be "." (current directory reference)
//  2. Re-check for ".." in case normalization produced it
func validatePathFinal(targetName, result string) error {
	// Check for reserved value "."
	if result == "." {
		return &InvalidTargetPathError{
			TargetName: targetName,
			ResultPath: result,
			Reason:     "path cannot be '.' (current directory)",
		}
	}

	// Double-check for ".." after normalization (shouldn't happen but be safe)
	if result == ".." || strings.HasPrefix(result, "../") {
		return &InvalidTargetPathError{
			TargetName: targetName,
			ResultPath: result,
			Reason:     "path contains forbidden traversal component '..'",
		}
	}

	return nil
}

// containsTraversalComponent checks if a path contains ".." as a path component.
// This is different from a simple substring check because we want to allow
// filenames like "file..txt" but reject path traversal attempts.
func containsTraversalComponent(p string) bool {
	// Check if the path is exactly ".."
	if p == ".." {
		return true
	}

	// Check if path starts with "../"
	if strings.HasPrefix(p, "../") {
		return true
	}

	// Check if path ends with "/.."
	if strings.HasSuffix(p, "/..") {
		return true
	}

	// Check if path contains "/../" as a component
	if strings.Contains(p, "/../") {
		return true
	}

	return false
}

// extractMissingVariableName attempts to extract the variable name from a
// template execution error caused by a missing map key.
//
// When using missingkey=error, Go templates produce errors like:
//
//	"template: target:1:4: executing "target" at <.Variables.namespace>: map has no entry for key "namespace""
//
// This function extracts the variable name from such error messages.
func extractMissingVariableName(err error) string {
	if err == nil {
		return ""
	}

	errMsg := err.Error()

	// Pattern 1: map has no entry for key "varname"
	// This is produced when accessing a missing key with missingkey=error
	mapKeyPattern := regexp.MustCompile(`map has no entry for key "([^"]+)"`)
	if matches := mapKeyPattern.FindStringSubmatch(errMsg); len(matches) > 1 {
		return matches[1]
	}

	// Pattern 2: nil pointer evaluating interface {}.fieldname
	// This is produced when Variables map itself is nil or accessing nested nil
	nilPointerPattern := regexp.MustCompile(`nil pointer evaluating [^.]+\.(\w+)`)
	if matches := nilPointerPattern.FindStringSubmatch(errMsg); len(matches) > 1 {
		return matches[1]
	}

	return ""
}
