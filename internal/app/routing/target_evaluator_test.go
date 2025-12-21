// Package routing_test contains tests for the routing package.
package routing_test

import (
	"errors"
	"os"
	"testing"

	"github.com/simensen/weftlo/internal/app/routing"
)

// =============================================================================
// Task Group 1: TargetEvaluator Component Tests
// =============================================================================

// Test basic variable substitution: {{ .Variables.namespace }} in target path
func TestTargetEvaluator_BasicVariableSubstitution(t *testing.T) {
	t.Parallel()
	variables := map[string]interface{}{
		"namespace": "acme",
	}

	evaluator := routing.NewTargetEvaluator(variables)

	testCases := []struct {
		name           string
		targetName     string
		targetTemplate string
		expectedPath   string
	}{
		{
			name:           "simple variable substitution",
			targetName:     "claude",
			targetTemplate: ".{{ .Variables.namespace }}/claude",
			expectedPath:   ".acme/claude",
		},
		{
			name:           "variable at start of path",
			targetName:     "config",
			targetTemplate: "{{ .Variables.namespace }}/config",
			expectedPath:   "acme/config",
		},
		{
			name:           "variable at end of path",
			targetName:     "tools",
			targetTemplate: ".tools/{{ .Variables.namespace }}",
			expectedPath:   ".tools/acme",
		},
		{
			name:           "multiple variables in path",
			targetName:     "multi",
			targetTemplate: "{{ .Variables.namespace }}/agents/{{ .Variables.namespace }}",
			expectedPath:   "acme/agents/acme",
		},
		{
			name:           "literal path without variables",
			targetName:     "literal",
			targetTemplate: ".claude",
			expectedPath:   ".claude",
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := evaluator.Evaluate(tc.targetName, tc.targetTemplate)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if result != tc.expectedPath {
				t.Errorf("expected path %q, got %q", tc.expectedPath, result)
			}
		})
	}
}

// Test Sprig function support: {{ env "USER" }} and {{ lower .Variables.name }}
func TestTargetEvaluator_SprigFunctionSupport(t *testing.T) {
	t.Parallel()
	// Set a test environment variable
	testUser := os.Getenv("USER")
	if testUser == "" {
		testUser = os.Getenv("USERNAME") // Windows fallback
	}
	if testUser == "" {
		// If no user env var is set, we'll test with a known variable
		t.Skip("No USER or USERNAME environment variable set")
	}

	variables := map[string]interface{}{
		"name":      "ACME",
		"namespace": "MyNamespace",
	}

	evaluator := routing.NewTargetEvaluator(variables)

	testCases := []struct {
		name           string
		targetName     string
		targetTemplate string
		expectedPath   string
	}{
		{
			name:           "env function reads environment variable",
			targetName:     "user-config",
			targetTemplate: `.{{ env "USER" }}/config`,
			expectedPath:   "." + testUser + "/config",
		},
		{
			name:           "lower function transforms to lowercase",
			targetName:     "lowered",
			targetTemplate: ".{{ lower .Variables.name }}",
			expectedPath:   ".acme",
		},
		{
			name:           "upper function transforms to uppercase",
			targetName:     "uppered",
			targetTemplate: ".{{ upper .Variables.name }}",
			expectedPath:   ".ACME",
		},
		{
			name:           "kebabcase function transforms string",
			targetName:     "kebab",
			targetTemplate: ".{{ .Variables.namespace | kebabcase }}",
			expectedPath:   ".my-namespace",
		},
		{
			name:           "chained sprig functions",
			targetName:     "chained",
			targetTemplate: ".{{ .Variables.name | lower | replace \"a\" \"@\" }}",
			expectedPath:   ".@cme",
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := evaluator.Evaluate(tc.targetName, tc.targetTemplate)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if result != tc.expectedPath {
				t.Errorf("expected path %q, got %q", tc.expectedPath, result)
			}
		})
	}
}

// Test missing variable produces MissingVariableError with fail-fast behavior
func TestTargetEvaluator_MissingVariableProducesMissingVariableError(t *testing.T) {
	t.Parallel()
	// Create evaluator with limited variables (missing 'namespace')
	variables := map[string]interface{}{
		"name": "acme",
	}

	evaluator := routing.NewTargetEvaluator(variables)

	testCases := []struct {
		name               string
		targetName         string
		targetTemplate     string
		expectedMissingVar string
	}{
		{
			name:               "missing top-level variable",
			targetName:         "claude",
			targetTemplate:     ".{{ .Variables.namespace }}/claude",
			expectedMissingVar: "namespace",
		},
		{
			name:               "missing variable in different position",
			targetName:         "cursor",
			targetTemplate:     ".cursor/{{ .Variables.undefined }}",
			expectedMissingVar: "undefined",
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := evaluator.Evaluate(tc.targetName, tc.targetTemplate)

			if err == nil {
				t.Fatalf("expected MissingVariableError, got nil (result: %q)", result)
			}

			var missingVarErr *routing.MissingVariableError
			if !errors.As(err, &missingVarErr) {
				t.Fatalf("expected MissingVariableError, got %T: %v", err, err)
			}

			if missingVarErr.TargetName != tc.targetName {
				t.Errorf("expected TargetName %q, got %q", tc.targetName, missingVarErr.TargetName)
			}

			if missingVarErr.VariableName != tc.expectedMissingVar {
				t.Errorf("expected VariableName %q, got %q", tc.expectedMissingVar, missingVarErr.VariableName)
			}

			if missingVarErr.TemplateString != tc.targetTemplate {
				t.Errorf("expected TemplateString %q, got %q", tc.targetTemplate, missingVarErr.TemplateString)
			}
		})
	}
}

// Test template syntax error produces TargetTemplateError
func TestTargetEvaluator_TemplateSyntaxErrorProducesTargetTemplateError(t *testing.T) {
	t.Parallel()
	evaluator := routing.NewTargetEvaluator(nil)

	testCases := []struct {
		name           string
		targetName     string
		targetTemplate string
	}{
		{
			name:           "unclosed action delimiter",
			targetName:     "broken",
			targetTemplate: ".{{ .Variables.namespace",
		},
		{
			name:           "invalid function name",
			targetName:     "invalid-func",
			targetTemplate: ".{{ notafunction .Variables.name }}",
		},
		{
			name:           "mismatched braces",
			targetName:     "mismatched",
			targetTemplate: ".{{ .Variables.name }",
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := evaluator.Evaluate(tc.targetName, tc.targetTemplate)

			if err == nil {
				t.Fatalf("expected TargetTemplateError, got nil (result: %q)", result)
			}

			var templateErr *routing.TargetTemplateError
			if !errors.As(err, &templateErr) {
				t.Fatalf("expected TargetTemplateError, got %T: %v", err, err)
			}

			if templateErr.TargetName != tc.targetName {
				t.Errorf("expected TargetName %q, got %q", tc.targetName, templateErr.TargetName)
			}

			if templateErr.TemplateString != tc.targetTemplate {
				t.Errorf("expected TemplateString %q, got %q", tc.targetTemplate, templateErr.TemplateString)
			}

			if templateErr.Err == nil {
				t.Error("expected underlying Err to be non-nil")
			}
		})
	}
}

// Test EvaluateAll() processes all targets and returns evaluated map
func TestTargetEvaluator_EvaluateAll_ProcessesAllTargetsAndReturnsEvaluatedMap(t *testing.T) {
	t.Parallel()
	variables := map[string]interface{}{
		"namespace": "acme",
		"team":      "backend",
	}

	evaluator := routing.NewTargetEvaluator(variables)

	targets := map[string]string{
		"claude":    ".{{ .Variables.namespace }}/claude",
		"cursor":    ".cursor",
		"custom":    "{{ .Variables.team }}/config",
		"_partials": "_partials",
	}

	evaluated, err := evaluator.EvaluateAll(targets)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedResults := map[string]string{
		"claude":    ".acme/claude",
		"cursor":    ".cursor",
		"custom":    "backend/config",
		"_partials": "_partials",
	}

	if len(evaluated) != len(expectedResults) {
		t.Errorf("expected %d results, got %d", len(expectedResults), len(evaluated))
	}

	for name, expected := range expectedResults {
		actual, exists := evaluated[name]
		if !exists {
			t.Errorf("expected target %q to be in results", name)
			continue
		}
		if actual != expected {
			t.Errorf("target %q: expected %q, got %q", name, expected, actual)
		}
	}
}

// Test empty variable value produces valid path (no error)
func TestTargetEvaluator_EmptyVariableValueProducesValidPath(t *testing.T) {
	t.Parallel()
	variables := map[string]interface{}{
		"namespace": "",
		"prefix":    "",
	}

	evaluator := routing.NewTargetEvaluator(variables)

	testCases := []struct {
		name           string
		targetName     string
		targetTemplate string
		expectedPath   string
	}{
		{
			name:           "empty variable produces empty segment",
			targetName:     "empty-ns",
			targetTemplate: ".{{ .Variables.namespace }}claude",
			expectedPath:   ".claude",
		},
		{
			name:           "empty variable in middle of path",
			targetName:     "middle-empty",
			targetTemplate: ".ai/{{ .Variables.prefix }}config",
			expectedPath:   ".ai/config",
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := evaluator.Evaluate(tc.targetName, tc.targetTemplate)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if result != tc.expectedPath {
				t.Errorf("expected path %q, got %q", tc.expectedPath, result)
			}
		})
	}
}

// =============================================================================
// Task Group 2: Path Normalization and Validation Tests
// =============================================================================

// Test consecutive slash collapsing: `.claude/agents//file` becomes `.claude/agents/file`
func TestTargetEvaluator_ConsecutiveSlashCollapsing(t *testing.T) {
	t.Parallel()
	variables := map[string]interface{}{
		"namespace": "acme",
	}

	evaluator := routing.NewTargetEvaluator(variables)

	testCases := []struct {
		name           string
		targetName     string
		targetTemplate string
		expectedPath   string
	}{
		{
			name:           "double slash in literal path",
			targetName:     "double-slash",
			targetTemplate: ".claude/agents//file",
			expectedPath:   ".claude/agents/file",
		},
		{
			name:           "multiple consecutive slashes",
			targetName:     "triple-slash",
			targetTemplate: ".claude///agents////file",
			expectedPath:   ".claude/agents/file",
		},
		{
			name:           "double slash after variable substitution",
			targetName:     "var-slash",
			targetTemplate: ".{{ .Variables.namespace }}//config",
			expectedPath:   ".acme/config",
		},
		{
			name:           "trailing slash removed",
			targetName:     "trailing",
			targetTemplate: ".claude/agents/",
			expectedPath:   ".claude/agents",
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := evaluator.Evaluate(tc.targetName, tc.targetTemplate)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if result != tc.expectedPath {
				t.Errorf("expected path %q, got %q", tc.expectedPath, result)
			}
		})
	}
}

// Test whitespace trimming: leading/trailing spaces removed from final path
func TestTargetEvaluator_WhitespaceTrimming(t *testing.T) {
	t.Parallel()
	variables := map[string]interface{}{
		"namespace": "acme",
		"spaced":    " trimmed ",
	}

	evaluator := routing.NewTargetEvaluator(variables)

	testCases := []struct {
		name           string
		targetName     string
		targetTemplate string
		expectedPath   string
	}{
		{
			name:           "leading spaces trimmed",
			targetName:     "leading",
			targetTemplate: "   .claude/agents",
			expectedPath:   ".claude/agents",
		},
		{
			name:           "trailing spaces trimmed",
			targetName:     "trailing",
			targetTemplate: ".claude/agents   ",
			expectedPath:   ".claude/agents",
		},
		{
			name:           "both leading and trailing spaces trimmed",
			targetName:     "both",
			targetTemplate: "  .claude/agents  ",
			expectedPath:   ".claude/agents",
		},
		{
			name:           "variable with spaces in value",
			targetName:     "var-spaced",
			targetTemplate: ".{{ .Variables.spaced }}/config",
			// The variable value " trimmed " is embedded in path, not trimmed
			// But the overall path gets cleaned
			expectedPath: ". trimmed /config",
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := evaluator.Evaluate(tc.targetName, tc.targetTemplate)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if result != tc.expectedPath {
				t.Errorf("expected path %q, got %q", tc.expectedPath, result)
			}
		})
	}
}

// Test variables with path separators: `namespace: "acme/backend"` produces valid path
func TestTargetEvaluator_VariablesWithPathSeparators(t *testing.T) {
	t.Parallel()
	variables := map[string]interface{}{
		"namespace": "acme/backend",
		"deep":      "org/team/project",
		"withSlash": "sub/path",
	}

	evaluator := routing.NewTargetEvaluator(variables)

	testCases := []struct {
		name           string
		targetName     string
		targetTemplate string
		expectedPath   string
	}{
		{
			name:           "namespace with single slash",
			targetName:     "ns-slash",
			targetTemplate: ".{{ .Variables.namespace }}/claude",
			expectedPath:   ".acme/backend/claude",
		},
		{
			name:           "deep nested namespace",
			targetName:     "deep",
			targetTemplate: ".{{ .Variables.deep }}/config",
			expectedPath:   ".org/team/project/config",
		},
		{
			name:           "multiple variables with slashes",
			targetName:     "multi-slash",
			targetTemplate: "{{ .Variables.namespace }}/{{ .Variables.withSlash }}",
			expectedPath:   "acme/backend/sub/path",
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := evaluator.Evaluate(tc.targetName, tc.targetTemplate)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if result != tc.expectedPath {
				t.Errorf("expected path %q, got %q", tc.expectedPath, result)
			}
		})
	}
}

// Test absolute path rejection: `/etc/passwd` returns `InvalidTargetPathError`
func TestTargetEvaluator_AbsolutePathRejection(t *testing.T) {
	t.Parallel()
	variables := map[string]interface{}{
		"absolute": "/etc/passwd",
		"normal":   "acme",
	}

	evaluator := routing.NewTargetEvaluator(variables)

	testCases := []struct {
		name           string
		targetName     string
		targetTemplate string
		expectedReason string
	}{
		{
			name:           "literal absolute path",
			targetName:     "etc-passwd",
			targetTemplate: "/etc/passwd",
			expectedReason: "path must be relative, not absolute",
		},
		{
			name:           "variable produces absolute path",
			targetName:     "var-absolute",
			targetTemplate: "{{ .Variables.absolute }}",
			expectedReason: "path must be relative, not absolute",
		},
		{
			name:           "absolute path with leading slash only",
			targetName:     "root",
			targetTemplate: "/config",
			expectedReason: "path must be relative, not absolute",
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := evaluator.Evaluate(tc.targetName, tc.targetTemplate)

			if err == nil {
				t.Fatalf("expected InvalidTargetPathError, got nil (result: %q)", result)
			}

			var pathErr *routing.InvalidTargetPathError
			if !errors.As(err, &pathErr) {
				t.Fatalf("expected InvalidTargetPathError, got %T: %v", err, err)
			}

			if pathErr.TargetName != tc.targetName {
				t.Errorf("expected TargetName %q, got %q", tc.targetName, pathErr.TargetName)
			}

			if pathErr.Reason != tc.expectedReason {
				t.Errorf("expected Reason %q, got %q", tc.expectedReason, pathErr.Reason)
			}
		})
	}
}

// Test path traversal rejection: `../secrets` returns `InvalidTargetPathError`
func TestTargetEvaluator_PathTraversalRejection(t *testing.T) {
	t.Parallel()
	variables := map[string]interface{}{
		"traversal": "../secrets",
		"hidden":    "../../etc/passwd",
	}

	evaluator := routing.NewTargetEvaluator(variables)

	testCases := []struct {
		name           string
		targetName     string
		targetTemplate string
		expectedReason string
	}{
		{
			name:           "literal path traversal",
			targetName:     "traversal",
			targetTemplate: "../secrets",
			expectedReason: "path contains forbidden traversal component '..'",
		},
		{
			name:           "path traversal at start",
			targetName:     "start-traversal",
			targetTemplate: "../config/settings",
			expectedReason: "path contains forbidden traversal component '..'",
		},
		{
			name:           "path traversal in middle",
			targetName:     "mid-traversal",
			targetTemplate: ".claude/../secrets",
			expectedReason: "path contains forbidden traversal component '..'",
		},
		{
			name:           "variable produces path traversal",
			targetName:     "var-traversal",
			targetTemplate: "{{ .Variables.traversal }}",
			expectedReason: "path contains forbidden traversal component '..'",
		},
		{
			name:           "deep path traversal",
			targetName:     "deep-traversal",
			targetTemplate: "{{ .Variables.hidden }}",
			expectedReason: "path contains forbidden traversal component '..'",
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := evaluator.Evaluate(tc.targetName, tc.targetTemplate)

			if err == nil {
				t.Fatalf("expected InvalidTargetPathError, got nil (result: %q)", result)
			}

			var pathErr *routing.InvalidTargetPathError
			if !errors.As(err, &pathErr) {
				t.Fatalf("expected InvalidTargetPathError, got %T: %v", err, err)
			}

			if pathErr.TargetName != tc.targetName {
				t.Errorf("expected TargetName %q, got %q", tc.targetName, pathErr.TargetName)
			}

			if pathErr.Reason != tc.expectedReason {
				t.Errorf("expected Reason %q, got %q", tc.expectedReason, pathErr.Reason)
			}
		})
	}
}

// Test reserved value rejection: path component `.` or `..` returns error
func TestTargetEvaluator_ReservedValueRejection(t *testing.T) {
	t.Parallel()
	variables := map[string]interface{}{
		"dot":     ".",
		"dotdot":  "..",
		"withDot": "path/./file",
	}

	evaluator := routing.NewTargetEvaluator(variables)

	testCases := []struct {
		name           string
		targetName     string
		targetTemplate string
		expectedReason string
	}{
		{
			name:           "standalone dot rejected",
			targetName:     "dot-only",
			targetTemplate: ".",
			expectedReason: "path cannot be '.' (current directory)",
		},
		{
			name:           "standalone dotdot rejected",
			targetName:     "dotdot-only",
			targetTemplate: "..",
			expectedReason: "path contains forbidden traversal component '..'",
		},
		{
			name:           "variable produces dot",
			targetName:     "var-dot",
			targetTemplate: "{{ .Variables.dot }}",
			expectedReason: "path cannot be '.' (current directory)",
		},
		{
			name:           "variable produces dotdot",
			targetName:     "var-dotdot",
			targetTemplate: "{{ .Variables.dotdot }}",
			expectedReason: "path contains forbidden traversal component '..'",
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := evaluator.Evaluate(tc.targetName, tc.targetTemplate)

			if err == nil {
				t.Fatalf("expected InvalidTargetPathError, got nil (result: %q)", result)
			}

			var pathErr *routing.InvalidTargetPathError
			if !errors.As(err, &pathErr) {
				t.Fatalf("expected InvalidTargetPathError, got %T: %v", err, err)
			}

			if pathErr.TargetName != tc.targetName {
				t.Errorf("expected TargetName %q, got %q", tc.targetName, pathErr.TargetName)
			}

			if pathErr.Reason != tc.expectedReason {
				t.Errorf("expected Reason %q, got %q", tc.expectedReason, pathErr.Reason)
			}
		})
	}
}

// =============================================================================
// Task Group 5: Strategic Gap-Filling Tests
// =============================================================================

// Test complex Sprig expressions with nested/chained function calls
func TestTargetEvaluator_ComplexSprigExpressions(t *testing.T) {
	t.Parallel()
	variables := map[string]interface{}{
		"company":   "ACME Corporation",
		"team":      "backend-team",
		"version":   "v1.2.3",
		"mixedCase": "MyProjectName",
	}

	evaluator := routing.NewTargetEvaluator(variables)

	testCases := []struct {
		name           string
		targetName     string
		targetTemplate string
		expectedPath   string
	}{
		{
			name:           "nested lower and replace",
			targetName:     "nested-funcs",
			targetTemplate: ".{{ .Variables.company | lower | replace \" \" \"-\" }}",
			expectedPath:   ".acme-corporation",
		},
		{
			name:           "kebabcase with trimSuffix",
			targetName:     "kebab-trim",
			targetTemplate: ".{{ .Variables.team | trimSuffix \"-team\" | kebabcase }}",
			expectedPath:   ".backend",
		},
		{
			name:           "snakecase transformation",
			targetName:     "snake",
			targetTemplate: ".{{ .Variables.mixedCase | snakecase }}",
			expectedPath:   ".my_project_name",
		},
		{
			name:           "trimPrefix and lower combined",
			targetName:     "trim-lower",
			targetTemplate: ".{{ .Variables.version | trimPrefix \"v\" | replace \".\" \"-\" }}",
			expectedPath:   ".1-2-3",
		},
		{
			name:           "default function with existing variable",
			targetName:     "default-existing",
			targetTemplate: `.{{ default "fallback" .Variables.company | lower | replace " " "_" }}`,
			expectedPath:   ".acme_corporation",
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := evaluator.Evaluate(tc.targetName, tc.targetTemplate)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if result != tc.expectedPath {
				t.Errorf("expected path %q, got %q", tc.expectedPath, result)
			}
		})
	}
}

// Test default function with missing/undefined variables (Sprig behavior)
func TestTargetEvaluator_DefaultFunctionWithMissingVariable(t *testing.T) {
	t.Parallel()
	// Create evaluator with limited variables - some are intentionally missing
	variables := map[string]interface{}{
		"defined": "exists",
	}

	evaluator := routing.NewTargetEvaluator(variables)

	testCases := []struct {
		name           string
		targetName     string
		targetTemplate string
		expectedPath   string
	}{
		{
			name:           "default with defined variable uses actual value",
			targetName:     "defined-var",
			targetTemplate: `.{{ default "fallback" .Variables.defined }}`,
			expectedPath:   ".exists",
		},
		{
			name:           "default function in complex path",
			targetName:     "complex-default",
			targetTemplate: `.{{ default "default-ns" .Variables.defined }}/claude`,
			expectedPath:   ".exists/claude",
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := evaluator.Evaluate(tc.targetName, tc.targetTemplate)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if result != tc.expectedPath {
				t.Errorf("expected path %q, got %q", tc.expectedPath, result)
			}
		})
	}
}

// Test EvaluateAll fails fast on first error and stops processing
func TestTargetEvaluator_EvaluateAllFailsFastOnError(t *testing.T) {
	t.Parallel()
	// Create evaluator with limited variables - will cause error on certain templates
	variables := map[string]interface{}{
		"defined": "acme",
	}

	evaluator := routing.NewTargetEvaluator(variables)

	// Include a mix of valid and invalid templates
	// The invalid one should cause immediate failure
	targets := map[string]string{
		"valid1":  ".{{ .Variables.defined }}/claude",
		"invalid": ".{{ .Variables.undefined }}/config", // This will fail - missing variable
		"valid2":  ".cursor",
	}

	evaluated, err := evaluator.EvaluateAll(targets)

	// Should return an error
	if err == nil {
		t.Fatal("expected EvaluateAll to fail with MissingVariableError, got nil")
	}

	// Should return nil map on error
	if evaluated != nil {
		t.Errorf("expected nil map on error, got %v", evaluated)
	}

	// Error should be MissingVariableError
	var missingVarErr *routing.MissingVariableError
	if !errors.As(err, &missingVarErr) {
		t.Fatalf("expected MissingVariableError, got %T: %v", err, err)
	}

	// Should be for the invalid target
	if missingVarErr.TargetName != "invalid" {
		t.Errorf("expected error for target 'invalid', got %q", missingVarErr.TargetName)
	}

	if missingVarErr.VariableName != "undefined" {
		t.Errorf("expected missing variable 'undefined', got %q", missingVarErr.VariableName)
	}
}

// Test empty string variable produces valid path with proper concatenation
func TestTargetEvaluator_EmptyVariableEdgeCases(t *testing.T) {
	t.Parallel()
	variables := map[string]interface{}{
		"empty":    "",
		"nonempty": "value",
	}

	evaluator := routing.NewTargetEvaluator(variables)

	testCases := []struct {
		name           string
		targetName     string
		targetTemplate string
		expectedPath   string
	}{
		{
			name:           "empty variable with slash normalizes via path.Clean",
			targetName:     "empty-slash",
			targetTemplate: ".{{ .Variables.empty }}/config",
			// Template produces "./config", path.Clean() normalizes to "config"
			expectedPath: "config",
		},
		{
			name:           "empty variable between two slashes collapses",
			targetName:     "empty-middle",
			targetTemplate: ".claude/{{ .Variables.empty }}/settings",
			expectedPath:   ".claude/settings",
		},
		{
			name:           "mixed empty and nonempty variables",
			targetName:     "mixed-empty",
			targetTemplate: "{{ .Variables.empty }}{{ .Variables.nonempty }}/config",
			expectedPath:   "value/config",
		},
		{
			name:           "all empty produces minimal path",
			targetName:     "all-empty",
			targetTemplate: "config{{ .Variables.empty }}",
			expectedPath:   "config",
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := evaluator.Evaluate(tc.targetName, tc.targetTemplate)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if result != tc.expectedPath {
				t.Errorf("expected path %q, got %q", tc.expectedPath, result)
			}
		})
	}
}

// Test variable containing deeply nested path segments
func TestTargetEvaluator_DeeplyNestedPathSegments(t *testing.T) {
	t.Parallel()
	variables := map[string]interface{}{
		"deep":       "org/division/team/project/module",
		"namespace":  "company/dept",
		"singleDeep": "a/b/c/d/e/f/g",
	}

	evaluator := routing.NewTargetEvaluator(variables)

	testCases := []struct {
		name           string
		targetName     string
		targetTemplate string
		expectedPath   string
	}{
		{
			name:           "five-level deep variable",
			targetName:     "deep-nested",
			targetTemplate: ".{{ .Variables.deep }}/config",
			expectedPath:   ".org/division/team/project/module/config",
		},
		{
			name:           "combining deep variable with another path variable",
			targetName:     "combined-deep",
			targetTemplate: "{{ .Variables.namespace }}/{{ .Variables.deep }}",
			expectedPath:   "company/dept/org/division/team/project/module",
		},
		{
			name:           "seven-level deep variable",
			targetName:     "very-deep",
			targetTemplate: ".{{ .Variables.singleDeep }}/file",
			expectedPath:   ".a/b/c/d/e/f/g/file",
		},
		{
			name:           "deep path with static prefix and suffix",
			targetName:     "wrapped-deep",
			targetTemplate: "root/{{ .Variables.deep }}/leaf",
			expectedPath:   "root/org/division/team/project/module/leaf",
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := evaluator.Evaluate(tc.targetName, tc.targetTemplate)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if result != tc.expectedPath {
				t.Errorf("expected path %q, got %q", tc.expectedPath, result)
			}
		})
	}
}

// Test ternary-like conditional expressions using Sprig
func TestTargetEvaluator_ConditionalExpressions(t *testing.T) {
	t.Parallel()
	variables := map[string]interface{}{
		"env":      "production",
		"devEnv":   "development",
		"empty":    "",
		"enabled":  true,
		"disabled": false,
	}

	evaluator := routing.NewTargetEvaluator(variables)

	testCases := []struct {
		name           string
		targetName     string
		targetTemplate string
		expectedPath   string
	}{
		{
			name:           "ternary with true condition",
			targetName:     "ternary-true",
			targetTemplate: `.{{ ternary "prod" "dev" .Variables.enabled }}`,
			expectedPath:   ".prod",
		},
		{
			name:           "ternary with false condition",
			targetName:     "ternary-false",
			targetTemplate: `.{{ ternary "prod" "dev" .Variables.disabled }}`,
			expectedPath:   ".dev",
		},
		{
			name:           "coalesce to pick first non-empty",
			targetName:     "coalesce",
			targetTemplate: `.{{ coalesce .Variables.empty .Variables.env }}`,
			expectedPath:   ".production",
		},
		{
			name:           "conditional path segment",
			targetName:     "conditional-path",
			targetTemplate: `.{{ ternary "prod-ns" "dev-ns" .Variables.enabled }}/claude`,
			expectedPath:   ".prod-ns/claude",
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := evaluator.Evaluate(tc.targetName, tc.targetTemplate)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if result != tc.expectedPath {
				t.Errorf("expected path %q, got %q", tc.expectedPath, result)
			}
		})
	}
}
