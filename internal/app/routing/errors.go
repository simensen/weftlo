// Package routing provides services for transforming content source paths to target installation paths.
// This file defines error types for content routing failures.
package routing

import "fmt"

// InvalidTargetError is returned when a target name or value contains reserved
// values that would break path resolution.
//
// Reserved values include "." and ".." which are filesystem path components
// that have special meaning and cannot be used as target names or values.
type InvalidTargetError struct {
	// TargetName is the name of the target that has an invalid value.
	// For example, "claude" when the target name itself is invalid.
	TargetName string

	// TargetValue is the value assigned to the target that is invalid.
	// For example, ".." when used as an installation path.
	TargetValue string

	// ReservedValue is the specific reserved value that was detected.
	// This will be either "." or "..".
	ReservedValue string

	// Reason provides additional context about why the value is invalid.
	Reason string
}

// Error implements the error interface, providing a message that includes
// the target name, invalid value, and reason for the error.
func (e *InvalidTargetError) Error() string {
	if e.TargetName == e.ReservedValue {
		return fmt.Sprintf("invalid target name '%s': '%s' is a reserved path component (%s)",
			e.TargetName, e.ReservedValue, e.Reason)
	}
	return fmt.Sprintf("invalid target value for '%s': '%s' contains reserved value '%s' (%s)",
		e.TargetName, e.TargetValue, e.ReservedValue, e.Reason)
}

// ConflictingTargetError is returned when a direct mapping (target with empty
// string value) creates path conflicts with other target paths.
//
// A direct mapping installs content directly to the project root, which can
// conflict with other targets that install to subdirectories of the root.
type ConflictingTargetError struct {
	// TargetName is the name of the target causing the conflict (usually the direct mapping).
	TargetName string

	// TargetPath is the path value of the conflicting target (usually empty string for direct mapping).
	TargetPath string

	// ConflictingName is the name of the other target that conflicts.
	ConflictingName string

	// ConflictingPath is the path of the other target that would conflict.
	ConflictingPath string
}

// Error implements the error interface, providing a message that includes
// both conflicting target names and their paths.
func (e *ConflictingTargetError) Error() string {
	return fmt.Sprintf("target '%s' (path: '%s') creates conflict with target '%s' (path: '%s'): "+
		"direct mapping to project root conflicts with other target paths",
		e.TargetName, e.TargetPath, e.ConflictingName, e.ConflictingPath)
}

// PartialSuppressionError is returned when attempting to suppress the _partials
// target via a project override.
//
// The _partials namespace is required for template rendering and cannot be
// suppressed. Partials are virtual and never installed, so suppression would
// only break template includes without any benefit.
type PartialSuppressionError struct {
	// SourcePath is the path of the project configuration file that attempted
	// to suppress _partials.
	SourcePath string
}

// Error implements the error interface, providing a message that explains
// why _partials cannot be suppressed.
func (e *PartialSuppressionError) Error() string {
	return fmt.Sprintf("cannot suppress '_partials' target: partials are required for template rendering "+
		"and cannot be suppressed (source: %s)", e.SourcePath)
}

// UnknownTargetCategoryError is returned when a source path's first segment
// does not match any target in the targets map and no default_target is configured.
//
// This error helps users understand that their content subdirectory is not
// mapped to any installation target.
type UnknownTargetCategoryError struct {
	// SourcePath is the full source path that could not be routed.
	SourcePath string

	// Category is the first path segment (subdirectory name) that was not found
	// in the targets map.
	Category string

	// AvailableTargets lists the target categories that are configured.
	// This helps users see what subdirectory names are valid.
	AvailableTargets []string
}

// Error implements the error interface, providing a message that includes
// the source path, unmatched category, and available targets.
func (e *UnknownTargetCategoryError) Error() string {
	if len(e.AvailableTargets) == 0 {
		return fmt.Sprintf("no target mapping for '%s': category '%s' not found and no default_target configured (no targets available)",
			e.SourcePath, e.Category)
	}
	return fmt.Sprintf("no target mapping for '%s': category '%s' not found and no default_target configured (available targets: %v)",
		e.SourcePath, e.Category, e.AvailableTargets)
}

// TargetTemplateError is returned when a target path template has a syntax error
// or cannot be parsed by the Go template engine.
//
// This error occurs during router construction when evaluating target templates
// and indicates that the template string in the profile configuration is malformed.
type TargetTemplateError struct {
	// TargetName is the name of the target whose template failed to parse or execute.
	TargetName string

	// TemplateString is the original template string that caused the error.
	TemplateString string

	// Err is the underlying error from the template engine.
	Err error
}

// Error implements the error interface, providing a message that includes
// the target name, template string, and underlying error.
func (e *TargetTemplateError) Error() string {
	return fmt.Sprintf("template error for target '%s': %v (template: %q)",
		e.TargetName, e.Err, e.TemplateString)
}

// Unwrap returns the underlying error for use with errors.Is and errors.As.
func (e *TargetTemplateError) Unwrap() error {
	return e.Err
}

// MissingVariableError is returned when a target path template references a
// variable that is not defined in the variables map.
//
// This error implements fail-fast behavior: evaluation stops at the first
// undefined variable rather than producing a path with "<no value>" placeholders.
type MissingVariableError struct {
	// TargetName is the name of the target whose template references the missing variable.
	TargetName string

	// VariableName is the name of the variable that was not found.
	// This is extracted from the template execution error.
	VariableName string

	// TemplateString is the original template string for debugging context.
	TemplateString string
}

// Error implements the error interface, providing a message that includes
// the target name, missing variable name, and template string.
func (e *MissingVariableError) Error() string {
	return fmt.Sprintf("missing variable '%s' in target '%s' (template: %q)",
		e.VariableName, e.TargetName, e.TemplateString)
}

// InvalidTargetPathError is returned when an evaluated target path fails
// validation checks after template substitution.
//
// Validation failures include:
//   - Absolute paths (starting with /)
//   - Path traversal attempts (containing ..)
//   - Reserved path components (. or .. as standalone components)
type InvalidTargetPathError struct {
	// TargetName is the name of the target whose evaluated path is invalid.
	TargetName string

	// ResultPath is the evaluated path that failed validation.
	ResultPath string

	// Reason describes why the path is invalid.
	Reason string
}

// Error implements the error interface, providing a message that includes
// the target name, invalid path, and reason for the validation failure.
func (e *InvalidTargetPathError) Error() string {
	return fmt.Sprintf("invalid evaluated path for target '%s': %q (%s)",
		e.TargetName, e.ResultPath, e.Reason)
}
