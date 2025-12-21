// Package routing_test contains tests for the routing package.
package routing_test

import (
	"errors"
	"testing"

	"github.com/simensen/weftlo/internal/app/routing"
	"github.com/simensen/weftlo/internal/domain/profile"
)

// =============================================================================
// Task Group 3: Router Integration with TargetEvaluator Tests
// =============================================================================

// Test NewContentRouter() with variables evaluates target templates
func TestNewContentRouter_WithVariablesEvaluatesTargetTemplates(t *testing.T) {
	t.Parallel()
	config := profile.ContentConfig{
		Root:          "content",
		DefaultTarget: "weftlo",
		Targets: map[string]string{
			"claude":    ".{{ .Variables.namespace }}/claude",
			"cursor":    ".{{ .Variables.namespace }}/cursor",
			"_partials": "_partials",
		},
	}

	variables := map[string]interface{}{
		"namespace": "acme",
	}

	router, err := routing.NewContentRouter(config, nil, variables)
	if err != nil {
		t.Fatalf("unexpected error creating router: %v", err)
	}

	// Route a file and verify the template was evaluated
	result, err := router.Route("claude/settings.json")
	if err != nil {
		t.Fatalf("unexpected error routing: %v", err)
	}

	expected := ".acme/claude/settings.json"
	if result.TargetPath != expected {
		t.Errorf("expected TargetPath %q (template evaluated), got %q", expected, result.TargetPath)
	}
}

// Test evaluated targets are stored on router (not raw templates)
func TestNewContentRouter_StoresEvaluatedTargetsNotRawTemplates(t *testing.T) {
	t.Parallel()
	config := profile.ContentConfig{
		Root:          "content",
		DefaultTarget: "weftlo",
		Targets: map[string]string{
			"claude":    ".{{ .Variables.namespace }}/claude",
			"_partials": "_partials",
		},
	}

	variables := map[string]interface{}{
		"namespace": "mycompany",
	}

	router, err := routing.NewContentRouter(config, nil, variables)
	if err != nil {
		t.Fatalf("unexpected error creating router: %v", err)
	}

	// Route multiple files to verify templates are pre-evaluated
	testCases := []struct {
		sourcePath   string
		expectedPath string
	}{
		{"claude/settings.json", ".mycompany/claude/settings.json"},
		{"claude/commands/build.json", ".mycompany/claude/commands/build.json"},
	}

	for _, tc := range testCases {
		t.Run(tc.sourcePath, func(t *testing.T) {
			result, err := router.Route(tc.sourcePath)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// The evaluated path should be consistent - templates evaluated once during construction
			if result.TargetPath != tc.expectedPath {
				t.Errorf("expected TargetPath %q, got %q", tc.expectedPath, result.TargetPath)
			}
		})
	}
}

// Test Route() uses pre-evaluated target paths
func TestNewContentRouter_RouteUsesPreEvaluatedTargetPaths(t *testing.T) {
	t.Parallel()
	config := profile.ContentConfig{
		Root:          "content",
		DefaultTarget: "weftlo",
		Targets: map[string]string{
			"claude": ".{{ .Variables.ns }}/ai/claude",
			"cursor": ".{{ .Variables.ns }}/ai/cursor",
		},
	}

	variables := map[string]interface{}{
		"ns": "tools",
	}

	router, err := routing.NewContentRouter(config, nil, variables)
	if err != nil {
		t.Fatalf("unexpected error creating router: %v", err)
	}

	// Both routes should use pre-evaluated paths
	claudeResult, err := router.Route("claude/settings.json")
	if err != nil {
		t.Fatalf("unexpected error routing claude: %v", err)
	}

	cursorResult, err := router.Route("cursor/rules.md")
	if err != nil {
		t.Fatalf("unexpected error routing cursor: %v", err)
	}

	// Verify evaluated paths are used
	if claudeResult.TargetPath != ".tools/ai/claude/settings.json" {
		t.Errorf("claude: expected '.tools/ai/claude/settings.json', got %q", claudeResult.TargetPath)
	}

	if cursorResult.TargetPath != ".tools/ai/cursor/rules.md" {
		t.Errorf("cursor: expected '.tools/ai/cursor/rules.md', got %q", cursorResult.TargetPath)
	}
}

// Test router construction fails on first evaluation error
func TestNewContentRouter_FailsOnFirstEvaluationError(t *testing.T) {
	t.Parallel()
	config := profile.ContentConfig{
		Root:          "content",
		DefaultTarget: "weftlo",
		Targets: map[string]string{
			"claude":    ".{{ .Variables.undefined_var }}/claude",
			"cursor":    ".cursor",
			"_partials": "_partials",
		},
	}

	// No variables provided - template references undefined variable
	variables := map[string]interface{}{}

	_, err := routing.NewContentRouter(config, nil, variables)
	if err == nil {
		t.Fatal("expected error when template references undefined variable, got nil")
	}

	// Should be a MissingVariableError
	var missingVarErr *routing.MissingVariableError
	if !errors.As(err, &missingVarErr) {
		t.Errorf("expected MissingVariableError, got %T: %v", err, err)
	}

	// Verify error contains useful context
	if missingVarErr.VariableName != "undefined_var" {
		t.Errorf("expected VariableName 'undefined_var', got %q", missingVarErr.VariableName)
	}
}

// Test nil/empty variables map is handled gracefully
func TestNewContentRouter_NilEmptyVariablesHandledGracefully(t *testing.T) {
	t.Parallel()
	config := profile.ContentConfig{
		Root:          "content",
		DefaultTarget: "weftlo",
		Targets: map[string]string{
			"claude":    ".claude",
			"cursor":    ".cursor",
			"_partials": "_partials",
		},
	}

	testCases := []struct {
		name      string
		variables map[string]interface{}
	}{
		{"nil variables", nil},
		{"empty variables", map[string]interface{}{}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			router, err := routing.NewContentRouter(config, nil, tc.variables)
			if err != nil {
				t.Fatalf("unexpected error creating router with %s: %v", tc.name, err)
			}

			// Router should work normally with static targets
			result, err := router.Route("claude/settings.json")
			if err != nil {
				t.Fatalf("unexpected error routing: %v", err)
			}

			if result.TargetPath != ".claude/settings.json" {
				t.Errorf("expected '.claude/settings.json', got %q", result.TargetPath)
			}
		})
	}
}

// Test target overrides bypass template evaluation (literal paths)
func TestNewContentRouter_OverridesBypassTemplateEvaluation(t *testing.T) {
	t.Parallel()
	config := profile.ContentConfig{
		Root:          "content",
		DefaultTarget: "weftlo",
		Targets: map[string]string{
			"claude":    ".{{ .Variables.ns }}/claude",
			"_partials": "_partials",
		},
	}

	variables := map[string]interface{}{
		"ns": "acme",
	}

	// Override with a literal path that contains template-like syntax
	// This should be treated as a literal string, not evaluated
	overrideValue := ".literal/{{ .Not.A.Template }}/path"
	overrides := map[string]*string{
		"claude": &overrideValue,
	}

	router, err := routing.NewContentRouter(config, overrides, variables)
	if err != nil {
		t.Fatalf("unexpected error creating router: %v", err)
	}

	// Route a file - the override should be used as-is (literal path)
	result, err := router.Route("claude/settings.json")
	if err != nil {
		t.Fatalf("unexpected error routing: %v", err)
	}

	// The override value should be used literally, not template-evaluated
	expected := ".literal/{{ .Not.A.Template }}/path/settings.json"
	if result.TargetPath != expected {
		t.Errorf("expected override to be literal path %q, got %q", expected, result.TargetPath)
	}

	if !result.OverrideApplied {
		t.Error("expected OverrideApplied to be true")
	}
}

// =============================================================================
// Task Group 4: ContentRouter Implementation Tests
// =============================================================================

// Test NewContentRouter correctly stores config and overrides
func TestNewContentRouter_CorrectlyStoresConfigAndOverrides(t *testing.T) {
	t.Parallel()
	config := profile.ContentConfig{
		Root:          "content",
		DefaultTarget: "weftlo",
		Targets: map[string]string{
			"claude": ".claude",
			"cursor": ".cursor",
		},
	}

	overrideValue := ".ai/claude"
	overrides := map[string]*string{
		"claude": &overrideValue,
	}

	router, err := routing.NewContentRouter(config, overrides, nil)
	if err != nil {
		t.Fatalf("unexpected error creating router: %v", err)
	}

	if router == nil {
		t.Fatal("expected router to be non-nil")
	}

	// Verify the router can be used (indicates config was stored)
	result, err := router.Route("claude/settings.json")
	if err != nil {
		t.Fatalf("unexpected error routing: %v", err)
	}

	// The override should have been applied
	if result.TargetPath != ".ai/claude/settings.json" {
		t.Errorf("expected overridden target path '.ai/claude/settings.json', got: %s", result.TargetPath)
	}
}

// Test Route() returns correct RoutedFile for standard target mapping
func TestContentRouter_Route_ReturnsCorrectRoutedFileForStandardTargetMapping(t *testing.T) {
	t.Parallel()
	config := profile.ContentConfig{
		Root:          "content",
		DefaultTarget: "weftlo",
		Targets: map[string]string{
			"claude":    ".claude",
			"cursor":    ".cursor",
			"_partials": "_partials",
		},
	}

	router, err := routing.NewContentRouter(config, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error creating router: %v", err)
	}

	testCases := []struct {
		sourcePath       string
		expectedTarget   string
		expectedCategory string
	}{
		{"claude/settings.json", ".claude/settings.json", "claude"},
		{"cursor/rules.md", ".cursor/rules.md", "cursor"},
		{"claude/commands/build.json", ".claude/commands/build.json", "claude"},
	}

	for _, tc := range testCases {
		t.Run(tc.sourcePath, func(t *testing.T) {
			result, err := router.Route(tc.sourcePath)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if result.SourcePath != tc.sourcePath {
				t.Errorf("expected SourcePath %q, got %q", tc.sourcePath, result.SourcePath)
			}

			if result.TargetPath != tc.expectedTarget {
				t.Errorf("expected TargetPath %q, got %q", tc.expectedTarget, result.TargetPath)
			}

			if result.TargetCategory != tc.expectedCategory {
				t.Errorf("expected TargetCategory %q, got %q", tc.expectedCategory, result.TargetCategory)
			}

			if result.OverrideApplied {
				t.Error("expected OverrideApplied to be false")
			}

			if result.Suppressed {
				t.Error("expected Suppressed to be false")
			}
		})
	}
}

// Test Route() applies project override when present (OverrideApplied=true)
func TestContentRouter_Route_AppliesProjectOverrideWhenPresent(t *testing.T) {
	t.Parallel()
	config := profile.ContentConfig{
		Root:          "content",
		DefaultTarget: "weftlo",
		Targets: map[string]string{
			"claude": ".claude",
			"cursor": ".cursor",
		},
	}

	// Override claude target to a different path
	overrideValue := ".ai/claude"
	overrides := map[string]*string{
		"claude": &overrideValue,
	}

	router, err := routing.NewContentRouter(config, overrides, nil)
	if err != nil {
		t.Fatalf("unexpected error creating router: %v", err)
	}

	result, err := router.Route("claude/settings.json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify override was applied
	expectedTarget := ".ai/claude/settings.json"
	if result.TargetPath != expectedTarget {
		t.Errorf("expected TargetPath %q after override, got %q", expectedTarget, result.TargetPath)
	}

	if !result.OverrideApplied {
		t.Error("expected OverrideApplied to be true when override modified target")
	}

	if result.Suppressed {
		t.Error("expected Suppressed to be false when override redirects target")
	}

	// Verify cursor (not overridden) works normally
	cursorResult, err := router.Route("cursor/rules.md")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cursorResult.TargetPath != ".cursor/rules.md" {
		t.Errorf("expected non-overridden target '.cursor/rules.md', got %q", cursorResult.TargetPath)
	}

	if cursorResult.OverrideApplied {
		t.Error("expected OverrideApplied to be false for non-overridden target")
	}
}

// Test Route() handles null override by setting Suppressed=true
func TestContentRouter_Route_HandlesNullOverrideBySuppressing(t *testing.T) {
	t.Parallel()
	config := profile.ContentConfig{
		Root:          "content",
		DefaultTarget: "weftlo",
		Targets: map[string]string{
			"claude":    ".claude",
			"cursor":    ".cursor",
			"_partials": "_partials",
		},
	}

	// Null override (nil pointer) for cursor target
	overrides := map[string]*string{
		"cursor": nil, // Suppress cursor target
	}

	router, err := routing.NewContentRouter(config, overrides, nil)
	if err != nil {
		t.Fatalf("unexpected error creating router: %v", err)
	}

	result, err := router.Route("cursor/rules.md")
	if err != nil {
		t.Fatalf("unexpected error for suppressed target: %v", err)
	}

	if !result.Suppressed {
		t.Error("expected Suppressed to be true for null override")
	}

	if result.TargetPath != "" {
		t.Errorf("expected empty TargetPath for suppressed target, got %q", result.TargetPath)
	}

	// OverrideApplied should be true since the override caused suppression
	if !result.OverrideApplied {
		t.Error("expected OverrideApplied to be true for null override")
	}

	// Verify other targets still work
	claudeResult, err := router.Route("claude/settings.json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if claudeResult.Suppressed {
		t.Error("expected claude target not to be suppressed")
	}
}

// Test Route() returns PartialSuppressionError when _partials override is null
func TestContentRouter_Route_ReturnsPartialSuppressionErrorWhenPartialsOverrideIsNull(t *testing.T) {
	t.Parallel()
	config := profile.ContentConfig{
		Root:          "content",
		DefaultTarget: "weftlo",
		Targets: map[string]string{
			"claude":    ".claude",
			"_partials": "_partials",
		},
	}

	// Attempt to suppress _partials (should fail at construction time)
	overrides := map[string]*string{
		"_partials": nil, // Attempt to suppress _partials
	}

	_, err := routing.NewContentRouter(config, overrides, nil)
	if err == nil {
		t.Fatal("expected error when suppressing _partials, got nil")
	}

	var partialSuppressionErr *routing.PartialSuppressionError
	if !errors.As(err, &partialSuppressionErr) {
		t.Errorf("expected PartialSuppressionError, got %T: %v", err, err)
	}
}

// Test IsPartialSource() correctly identifies partial source paths
func TestContentRouter_IsPartialSource_CorrectlyIdentifiesPartialSourcePaths(t *testing.T) {
	t.Parallel()
	config := profile.ContentConfig{
		Root:          "content",
		DefaultTarget: "weftlo",
		Targets: map[string]string{
			"claude":    ".claude",
			"_partials": "_partials",
		},
	}

	router, err := routing.NewContentRouter(config, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error creating router: %v", err)
	}

	testCases := []struct {
		sourcePath string
		expected   bool
	}{
		{"_partials/header.tmpl", true},
		{"_partials/sub/footer.tmpl", true},
		{"_partials/deep/nested/partial.tmpl", true},
		{"claude/settings.json", false},
		{"cursor/rules.md", false},
		{"unknown/file.md", false},
	}

	for _, tc := range testCases {
		t.Run(tc.sourcePath, func(t *testing.T) {
			result := router.IsPartialSource(tc.sourcePath)
			if result != tc.expected {
				t.Errorf("IsPartialSource(%q) = %v, expected %v", tc.sourcePath, result, tc.expected)
			}
		})
	}
}

// Test Canonicalize() normalizes custom partial directories to _partials/* format
func TestContentRouter_Canonicalize_NormalizesCustomPartialDirectoriesToPartialsFormat(t *testing.T) {
	t.Parallel()
	// Test with a custom partial directory mapped to _partials
	config := profile.ContentConfig{
		Root:          "content",
		DefaultTarget: "weftlo",
		Targets: map[string]string{
			"claude":           ".claude",
			"templates":        "_partials", // Custom directory mapped to _partials
			"template-helpers": "_partials", // Another custom mapping
		},
	}

	router, err := routing.NewContentRouter(config, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error creating router: %v", err)
	}

	testCases := []struct {
		sourcePath string
		expected   string
	}{
		{"templates/header.tmpl", "_partials/header.tmpl"},
		{"templates/sub/footer.tmpl", "_partials/sub/footer.tmpl"},
		{"template-helpers/util.tmpl", "_partials/util.tmpl"},
		{"claude/settings.json", ""}, // Not a partial source
		{"unknown/file.md", ""},      // Not a partial source
	}

	for _, tc := range testCases {
		t.Run(tc.sourcePath, func(t *testing.T) {
			result := router.Canonicalize(tc.sourcePath)
			if result != tc.expected {
				t.Errorf("Canonicalize(%q) = %q, expected %q", tc.sourcePath, result, tc.expected)
			}
		})
	}
}

// Test nested partials preserve subdirectory structure in canonical form
func TestContentRouter_Canonicalize_NestedPartialsPreserveSubdirectoryStructure(t *testing.T) {
	t.Parallel()
	config := profile.ContentConfig{
		Root:          "content",
		DefaultTarget: "weftlo",
		Targets: map[string]string{
			"_partials": "_partials",
		},
	}

	router, err := routing.NewContentRouter(config, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error creating router: %v", err)
	}

	testCases := []struct {
		sourcePath string
		expected   string
	}{
		{"_partials/header.tmpl", "_partials/header.tmpl"},
		{"_partials/sub/footer.tmpl", "_partials/sub/footer.tmpl"},
		{"_partials/deep/nested/partial.tmpl", "_partials/deep/nested/partial.tmpl"},
		{"_partials/a/b/c/d/file.tmpl", "_partials/a/b/c/d/file.tmpl"},
	}

	for _, tc := range testCases {
		t.Run(tc.sourcePath, func(t *testing.T) {
			result := router.Canonicalize(tc.sourcePath)
			if result != tc.expected {
				t.Errorf("Canonicalize(%q) = %q, expected %q", tc.sourcePath, result, tc.expected)
			}
		})
	}
}

// Test Route() for _partials paths indicates virtual namespace (not for installation)
func TestContentRouter_Route_PartialsPathsIndicateVirtualNamespace(t *testing.T) {
	t.Parallel()
	config := profile.ContentConfig{
		Root:          "content",
		DefaultTarget: "weftlo",
		Targets: map[string]string{
			"claude":    ".claude",
			"_partials": "_partials",
		},
	}

	router, err := routing.NewContentRouter(config, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error creating router: %v", err)
	}

	result, err := router.Route("_partials/header.tmpl")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Partials should have empty TargetPath (virtual, not for installation)
	if result.TargetPath != "" {
		t.Errorf("expected empty TargetPath for _partials (virtual namespace), got %q", result.TargetPath)
	}

	// Category should still be set
	if result.TargetCategory != "_partials" {
		t.Errorf("expected TargetCategory '_partials', got %q", result.TargetCategory)
	}

	// Suppressed should be true to indicate not for installation
	if !result.Suppressed {
		t.Error("expected Suppressed to be true for _partials (virtual namespace)")
	}

	// OverrideApplied should be false (this is inherent behavior, not an override)
	if result.OverrideApplied {
		t.Error("expected OverrideApplied to be false for _partials virtual namespace")
	}
}
