// Package routing_test contains integration tests for the routing package.
// These tests verify end-to-end workflows and edge cases not covered by unit tests.
package routing_test

import (
	"errors"
	"testing"

	"github.com/simensen/weftlo/internal/app/routing"
	"github.com/simensen/weftlo/internal/domain/profile"
)

// =============================================================================
// Task Group 5: Integration Testing and Gap Analysis
// =============================================================================

// Test full workflow: merged ContentConfig -> ContentRouter -> Route multiple paths
// This verifies the complete integration from config merge through path routing.
func TestIntegration_MergedConfigToRouterToMultiplePaths(t *testing.T) {
	t.Parallel()
	// Parent profile defines base targets
	parent := profile.ContentConfig{
		Root:          "content",
		DefaultTarget: "weftlo",
		Targets: map[string]string{
			"claude":    ".claude",
			"shared":    "weftlo/shared",
			"_partials": "_partials",
		},
	}

	// Child profile adds cursor and overrides claude path
	child := profile.ContentConfig{
		Targets: map[string]string{
			"cursor": ".cursor",
			"claude": ".ai/claude", // Override parent
		},
	}

	// Merge configs (simulates profile inheritance)
	merged := routing.MergeContentConfigs(parent, child)

	// Verify merge preserved parent values and applied child overrides
	if merged.DefaultTarget != "weftlo" {
		t.Errorf("expected DefaultTarget 'weftlo' from parent, got: %s", merged.DefaultTarget)
	}

	// Create router with merged config (no project overrides, no variables)
	router, err := routing.NewContentRouter(merged, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error creating router: %v", err)
	}

	// Route multiple paths and verify full workflow
	testCases := []struct {
		sourcePath       string
		expectedTarget   string
		expectedCategory string
		description      string
	}{
		{
			sourcePath:       "claude/settings.json",
			expectedTarget:   ".ai/claude/settings.json", // Child override
			expectedCategory: "claude",
			description:      "child override applied to parent target",
		},
		{
			sourcePath:       "cursor/rules.md",
			expectedTarget:   ".cursor/rules.md", // Child-only target
			expectedCategory: "cursor",
			description:      "child-only target works",
		},
		{
			sourcePath:       "shared/config.yaml",
			expectedTarget:   "weftlo/shared/config.yaml", // Parent-only target
			expectedCategory: "shared",
			description:      "parent-only target preserved",
		},
		{
			sourcePath:       "unknown/file.md",
			expectedTarget:   "weftlo/unknown/file.md", // Default target
			expectedCategory: "",
			description:      "unmapped directory uses default_target",
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.description, func(t *testing.T) {
			t.Parallel()
			result, err := router.Route(tc.sourcePath)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if result.TargetPath != tc.expectedTarget {
				t.Errorf("expected TargetPath %q, got %q", tc.expectedTarget, result.TargetPath)
			}

			if result.TargetCategory != tc.expectedCategory {
				t.Errorf("expected TargetCategory %q, got %q", tc.expectedCategory, result.TargetCategory)
			}
		})
	}
}

// Test inheritance chain scenario: grandparent -> parent -> child config merge
// This verifies multiple levels of inheritance work correctly.
func TestIntegration_ThreeLevelInheritanceChain(t *testing.T) {
	t.Parallel()
	// Grandparent: base profile with minimal config
	grandparent := profile.ContentConfig{
		Root:          "base-content",
		DefaultTarget: "base-default",
		Targets: map[string]string{
			"claude":    ".claude",
			"_partials": "_partials",
		},
	}

	// Parent: extends grandparent, adds cursor
	parent := profile.ContentConfig{
		Root: "parent-content", // Override grandparent root
		Targets: map[string]string{
			"cursor": ".cursor",
		},
	}

	// Child: extends parent, overrides claude and default_target
	child := profile.ContentConfig{
		DefaultTarget: "child-default", // Override parent (which inherited from grandparent)
		Targets: map[string]string{
			"claude": ".ai/claude", // Override grandparent's claude
			"custom": "custom-dir",
		},
	}

	// Simulate two-level merge: grandparent + parent first
	intermediate := routing.MergeContentConfigs(grandparent, parent)

	// Then merge intermediate + child
	final := routing.MergeContentConfigs(intermediate, child)

	// Verify final merged result
	if final.Root != "parent-content" {
		t.Errorf("expected Root 'parent-content' (parent overrides grandparent), got: %s", final.Root)
	}

	if final.DefaultTarget != "child-default" {
		t.Errorf("expected DefaultTarget 'child-default' (child overrides), got: %s", final.DefaultTarget)
	}

	// Verify all targets are present with correct values
	expectedTargets := map[string]string{
		"claude":    ".ai/claude", // Child override
		"cursor":    ".cursor",    // From parent
		"_partials": "_partials",  // From grandparent
		"custom":    "custom-dir", // Child-only
	}

	for name, expected := range expectedTargets {
		if actual := final.Targets[name]; actual != expected {
			t.Errorf("expected target '%s' = %q, got %q", name, expected, actual)
		}
	}

	// Create router and verify routing works
	router, err := routing.NewContentRouter(final, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error creating router: %v", err)
	}

	result, err := router.Route("claude/test.json")
	if err != nil {
		t.Fatalf("unexpected error routing: %v", err)
	}

	if result.TargetPath != ".ai/claude/test.json" {
		t.Errorf("expected final override path '.ai/claude/test.json', got: %s", result.TargetPath)
	}
}

// Test project override applied to inherited target mapping
// This verifies project overrides work correctly with inherited configs.
func TestIntegration_ProjectOverrideOnInheritedTarget(t *testing.T) {
	t.Parallel()
	// Parent profile
	parent := profile.ContentConfig{
		Targets: map[string]string{
			"claude": ".claude",
			"cursor": ".cursor",
		},
	}

	// Child profile inherits and adds
	child := profile.ContentConfig{
		Targets: map[string]string{
			"shared": "weftlo/shared",
		},
	}

	// Merge
	merged := routing.MergeContentConfigs(parent, child)

	// Project override redirects claude to a custom location
	projectOverride := ".project/ai/claude"
	overrides := map[string]*string{
		"claude": &projectOverride,
	}

	router, err := routing.NewContentRouter(merged, overrides, nil)
	if err != nil {
		t.Fatalf("unexpected error creating router: %v", err)
	}

	// Route claude path - should use project override
	result, err := router.Route("claude/settings.json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.TargetPath != ".project/ai/claude/settings.json" {
		t.Errorf("expected project override path '.project/ai/claude/settings.json', got: %s", result.TargetPath)
	}

	if !result.OverrideApplied {
		t.Error("expected OverrideApplied to be true")
	}

	// Route cursor - should use profile config (no override)
	cursorResult, err := router.Route("cursor/rules.md")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cursorResult.TargetPath != ".cursor/rules.md" {
		t.Errorf("expected profile path '.cursor/rules.md', got: %s", cursorResult.TargetPath)
	}

	if cursorResult.OverrideApplied {
		t.Error("expected OverrideApplied to be false for non-overridden target")
	}
}

// Test error propagation from validation through Route method
// Verifies that validation errors during construction are properly returned.
func TestIntegration_ErrorPropagationFromValidation(t *testing.T) {
	t.Parallel()
	// Config with reserved value in target
	config := profile.ContentConfig{
		Targets: map[string]string{
			"claude":  ".claude",
			"invalid": "..", // Reserved value should cause error
		},
	}

	_, err := routing.NewContentRouter(config, nil, nil)
	if err == nil {
		t.Fatal("expected error for reserved value '..' in target, got nil")
	}

	// With TargetEvaluator, path validation returns InvalidTargetPathError
	var invalidTargetPathErr *routing.InvalidTargetPathError
	if !errors.As(err, &invalidTargetPathErr) {
		t.Errorf("expected InvalidTargetPathError, got %T: %v", err, err)
	}

	// Also test that PartialSuppressionError propagates correctly
	validConfig := profile.ContentConfig{
		Targets: map[string]string{
			"claude":    ".claude",
			"_partials": "_partials",
		},
	}

	_, err = routing.NewContentRouter(validConfig, map[string]*string{
		"_partials": nil, // Attempt to suppress _partials
	}, nil)
	if err == nil {
		t.Fatal("expected PartialSuppressionError, got nil")
	}

	var partialSuppressionErr *routing.PartialSuppressionError
	if !errors.As(err, &partialSuppressionErr) {
		t.Errorf("expected PartialSuppressionError, got %T: %v", err, err)
	}
}

// Test edge case: all standard targets suppressed, only _partials remains
// Verifies routing still works when only partials are available.
func TestIntegration_AllStandardTargetsSuppressedOnlyPartialsRemains(t *testing.T) {
	t.Parallel()
	config := profile.ContentConfig{
		DefaultTarget: "fallback",
		Targets: map[string]string{
			"claude":    ".claude",
			"cursor":    ".cursor",
			"_partials": "_partials",
		},
	}

	// Suppress all standard targets
	overrides := map[string]*string{
		"claude": nil, // Suppress
		"cursor": nil, // Suppress
	}

	router, err := routing.NewContentRouter(config, overrides, nil)
	if err != nil {
		t.Fatalf("unexpected error creating router: %v", err)
	}

	// Claude paths should be suppressed
	claudeResult, err := router.Route("claude/settings.json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !claudeResult.Suppressed {
		t.Error("expected claude to be suppressed")
	}

	if claudeResult.TargetPath != "" {
		t.Errorf("expected empty TargetPath for suppressed target, got: %s", claudeResult.TargetPath)
	}

	// Partials should still work (virtual namespace, suppressed by design)
	partialsResult, err := router.Route("_partials/header.tmpl")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !partialsResult.Suppressed {
		t.Error("expected _partials to be suppressed (virtual namespace)")
	}

	// IsPartialSource should still identify partials
	if !router.IsPartialSource("_partials/header.tmpl") {
		t.Error("expected IsPartialSource to return true for _partials path")
	}

	// Unknown paths should go to default_target
	unknownResult, err := router.Route("unknown/file.md")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if unknownResult.TargetPath != "fallback/unknown/file.md" {
		t.Errorf("expected fallback path 'fallback/unknown/file.md', got: %s", unknownResult.TargetPath)
	}
}

// Test edge case: empty targets map with default_target handling all routing
// Verifies routing works when no explicit targets are defined.
func TestIntegration_EmptyTargetsMapWithDefaultTarget(t *testing.T) {
	t.Parallel()
	config := profile.ContentConfig{
		DefaultTarget: "all-content",
		Targets:       map[string]string{}, // No explicit targets
	}

	router, err := routing.NewContentRouter(config, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error creating router: %v", err)
	}

	// All paths should go to default_target
	testCases := []struct {
		sourcePath     string
		expectedTarget string
	}{
		{"claude/settings.json", "all-content/claude/settings.json"},
		{"cursor/rules.md", "all-content/cursor/rules.md"},
		{"any/deep/nested/file.txt", "all-content/any/deep/nested/file.txt"},
		{"single-file.md", "all-content/single-file.md"},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.sourcePath, func(t *testing.T) {
			t.Parallel()
			result, err := router.Route(tc.sourcePath)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if result.TargetPath != tc.expectedTarget {
				t.Errorf("expected TargetPath %q, got %q", tc.expectedTarget, result.TargetPath)
			}

			// Category should be empty (no target matched)
			if result.TargetCategory != "" {
				t.Errorf("expected empty TargetCategory, got %q", result.TargetCategory)
			}
		})
	}
}

// Test edge case: deeply nested paths with various target mappings
// Verifies subdirectory structure is preserved at multiple depths.
func TestIntegration_DeeplyNestedPathsWithVariousTargets(t *testing.T) {
	t.Parallel()
	config := profile.ContentConfig{
		DefaultTarget: "default-location",
		Targets: map[string]string{
			"claude":    ".claude",
			"cursor":    ".cursor",
			"direct":    "", // Maps to project root
			"_partials": "_partials",
		},
	}

	router, err := routing.NewContentRouter(config, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error creating router: %v", err)
	}

	testCases := []struct {
		sourcePath     string
		expectedTarget string
		description    string
	}{
		{
			sourcePath:     "claude/a/b/c/d/e/deep-file.json",
			expectedTarget: ".claude/a/b/c/d/e/deep-file.json",
			description:    "6-level nested path under claude",
		},
		{
			sourcePath:     "cursor/level1/level2/level3/file.md",
			expectedTarget: ".cursor/level1/level2/level3/file.md",
			description:    "4-level nested path under cursor",
		},
		{
			sourcePath:     "unmapped/deep/path/structure/file.txt",
			expectedTarget: "default-location/unmapped/deep/path/structure/file.txt",
			description:    "deeply nested unmapped path",
		},
		{
			sourcePath:     "_partials/components/ui/buttons/primary.tmpl",
			expectedTarget: "",
			description:    "deeply nested partial (virtual namespace)",
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.description, func(t *testing.T) {
			t.Parallel()
			result, err := router.Route(tc.sourcePath)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if result.TargetPath != tc.expectedTarget {
				t.Errorf("expected TargetPath %q, got %q", tc.expectedTarget, result.TargetPath)
			}
		})
	}
}

// Test direct mapping with no other targets (special edge case)
// Verifies direct mapping works correctly in isolation.
func TestIntegration_DirectMappingOnly(t *testing.T) {
	t.Parallel()
	// Config with only direct mapping (no conflict possible)
	config := profile.ContentConfig{
		Targets: map[string]string{
			"root-files": "", // Direct mapping to project root
		},
	}

	router, err := routing.NewContentRouter(config, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error creating router: %v", err)
	}

	result, err := router.Route("root-files/README.md")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should map directly to project root
	if result.TargetPath != "README.md" {
		t.Errorf("expected TargetPath 'README.md', got: %s", result.TargetPath)
	}

	if result.TargetCategory != "root-files" {
		t.Errorf("expected TargetCategory 'root-files', got: %s", result.TargetCategory)
	}

	// Nested path under direct mapping
	nestedResult, err := router.Route("root-files/docs/guide.md")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if nestedResult.TargetPath != "docs/guide.md" {
		t.Errorf("expected TargetPath 'docs/guide.md', got: %s", nestedResult.TargetPath)
	}
}
