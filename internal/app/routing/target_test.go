// Package routing_test contains tests for the routing package.
package routing_test

import (
	"errors"
	"testing"

	"github.com/simensen/weftlo/internal/app/routing"
)

// =============================================================================
// Task Group 3: Target Resolution Logic Tests
// =============================================================================

// Test first path segment matches against targets map keys
func TestResolveTarget_FirstPathSegmentMatchesAgainstTargetsMapKeys(t *testing.T) {
	t.Parallel()
	targets := map[string]string{
		"claude": ".claude",
		"cursor": ".cursor",
		"shared": "weftlo/shared",
	}

	testCases := []struct {
		sourcePath       string
		expectedTarget   string
		expectedCategory string
	}{
		{"claude/settings.json", ".claude/settings.json", "claude"},
		{"cursor/rules.md", ".cursor/rules.md", "cursor"},
		{"shared/config.yaml", "weftlo/shared/config.yaml", "shared"},
	}

	for _, tc := range testCases {
		t.Run(tc.sourcePath, func(t *testing.T) {
			result, err := routing.ResolveTarget(tc.sourcePath, targets, "")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if result.TargetPath != tc.expectedTarget {
				t.Errorf("expected TargetPath %q, got %q", tc.expectedTarget, result.TargetPath)
			}

			if result.Category != tc.expectedCategory {
				t.Errorf("expected Category %q, got %q", tc.expectedCategory, result.Category)
			}
		})
	}
}

// Test prefix replacement
func TestResolveTarget_PrefixReplacement(t *testing.T) {
	t.Parallel()
	targets := map[string]string{
		"claude": ".claude",
	}

	// Test that source path "claude/settings.json" becomes ".claude/settings.json"
	result, err := routing.ResolveTarget("claude/settings.json", targets, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := ".claude/settings.json"
	if result.TargetPath != expected {
		t.Errorf("expected TargetPath %q, got %q", expected, result.TargetPath)
	}

	// Test deeper paths
	result, err = routing.ResolveTarget("claude/commands/test.json", targets, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected = ".claude/commands/test.json"
	if result.TargetPath != expected {
		t.Errorf("expected TargetPath %q for nested path, got %q", expected, result.TargetPath)
	}
}

// Test empty string target maps to project root
func TestResolveTarget_EmptyStringTargetMapsToProjectRoot(t *testing.T) {
	t.Parallel()
	targets := map[string]string{
		"direct": "", // Maps to project root
	}

	// Test that "direct/file.md" becomes "file.md"
	result, err := routing.ResolveTarget("direct/file.md", targets, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := "file.md"
	if result.TargetPath != expected {
		t.Errorf("expected TargetPath %q (project root), got %q", expected, result.TargetPath)
	}

	// Test nested path under direct mapping
	result, err = routing.ResolveTarget("direct/sub/nested.md", targets, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected = "sub/nested.md"
	if result.TargetPath != expected {
		t.Errorf("expected TargetPath %q for nested direct path, got %q", expected, result.TargetPath)
	}
}

// Test unmapped directories use default_target preserving full subdirectory structure
func TestResolveTarget_UnmappedDirectoriesUseDefaultTargetPreservingStructure(t *testing.T) {
	t.Parallel()
	targets := map[string]string{
		"claude": ".claude",
	}
	defaultTarget := "weftlo"

	// Test unmapped directory uses default_target
	result, err := routing.ResolveTarget("unknown/file.md", targets, defaultTarget)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should preserve full path under default_target
	expected := "weftlo/unknown/file.md"
	if result.TargetPath != expected {
		t.Errorf("expected TargetPath %q, got %q", expected, result.TargetPath)
	}

	if result.Category != "" {
		t.Errorf("expected empty Category for unmapped directory, got %q", result.Category)
	}

	if !result.UsedDefault {
		t.Error("expected UsedDefault to be true")
	}

	// Test nested unmapped directory
	result, err = routing.ResolveTarget("unmapped/sub/deep/file.txt", targets, defaultTarget)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected = "weftlo/unmapped/sub/deep/file.txt"
	if result.TargetPath != expected {
		t.Errorf("expected TargetPath %q for nested unmapped, got %q", expected, result.TargetPath)
	}
}

// Test reserved value validation rejects "." and ".." as target names or values
func TestResolveTarget_ReservedValueValidationRejectsDotAndDoubleDot(t *testing.T) {
	t.Parallel()
	// Test "." as target name
	targetsWithDotName := map[string]string{
		".": "somepath",
	}

	_, err := routing.ResolveTarget("./file.md", targetsWithDotName, "")
	if err == nil {
		t.Error("expected error for '.' as target name, got nil")
	}

	var invalidTargetErr *routing.InvalidTargetError
	if !errors.As(err, &invalidTargetErr) {
		t.Errorf("expected InvalidTargetError, got %T: %v", err, err)
	}

	// Test ".." as target name
	targetsWithDoubleDotName := map[string]string{
		"..": "somepath",
	}

	_, err = routing.ResolveTarget("../file.md", targetsWithDoubleDotName, "")
	if err == nil {
		t.Error("expected error for '..' as target name, got nil")
	}

	if !errors.As(err, &invalidTargetErr) {
		t.Errorf("expected InvalidTargetError, got %T: %v", err, err)
	}

	// Test "." as target value
	targetsWithDotValue := map[string]string{
		"current": ".",
	}

	_, err = routing.ResolveTarget("current/file.md", targetsWithDotValue, "")
	if err == nil {
		t.Error("expected error for '.' as target value, got nil")
	}

	if !errors.As(err, &invalidTargetErr) {
		t.Errorf("expected InvalidTargetError, got %T: %v", err, err)
	}

	// Test ".." as target value
	targetsWithDoubleDotValue := map[string]string{
		"parent": "..",
	}

	_, err = routing.ResolveTarget("parent/file.md", targetsWithDoubleDotValue, "")
	if err == nil {
		t.Error("expected error for '..' as target value, got nil")
	}

	if !errors.As(err, &invalidTargetErr) {
		t.Errorf("expected InvalidTargetError, got %T: %v", err, err)
	}
}

// Test direct mapping conflict detection when paths could overlap
func TestValidateTargetsForConflicts_DirectMappingConflictDetection(t *testing.T) {
	t.Parallel()
	// Direct mapping ("") could conflict with other target paths that start at root
	// For example, if "direct" maps to "" and "claude" maps to ".claude",
	// then "direct/.claude/file.md" would conflict with "claude/file.md"
	targets := map[string]string{
		"direct": "",        // Maps to project root
		"claude": ".claude", // Maps to .claude
	}

	// When using direct mapping, check for potential conflicts
	// The path "direct/.claude/something" would install to ".claude/something"
	// which conflicts with what "claude/something" would install to
	err := routing.ValidateTargetsForConflicts(targets)
	if err == nil {
		t.Error("expected conflict error for direct mapping overlapping with other targets, got nil")
	}

	var conflictErr *routing.ConflictingTargetError
	if !errors.As(err, &conflictErr) {
		t.Errorf("expected ConflictingTargetError, got %T: %v", err, err)
	}
}

// Test _partials target resolution to virtual namespace
func TestResolveTarget_PartialsTargetResolutionToVirtualNamespace(t *testing.T) {
	t.Parallel()
	targets := map[string]string{
		"claude":    ".claude",
		"_partials": "_partials", // Virtual namespace
	}

	// Test that _partials source paths resolve correctly
	result, err := routing.ResolveTarget("_partials/header.tmpl", targets, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := "_partials/header.tmpl"
	if result.TargetPath != expected {
		t.Errorf("expected TargetPath %q for partials, got %q", expected, result.TargetPath)
	}

	if result.Category != "_partials" {
		t.Errorf("expected Category '_partials', got %q", result.Category)
	}

	if !result.IsPartial {
		t.Error("expected IsPartial to be true for _partials path")
	}
}

// Test nested paths preserve subdirectory structure after prefix replacement
func TestResolveTarget_NestedPathsPreserveSubdirectoryStructure(t *testing.T) {
	t.Parallel()
	targets := map[string]string{
		"claude":    ".claude",
		"_partials": "_partials",
	}

	testCases := []struct {
		sourcePath     string
		expectedTarget string
	}{
		{"claude/commands/build.json", ".claude/commands/build.json"},
		{"claude/a/b/c/d/file.md", ".claude/a/b/c/d/file.md"},
		{"_partials/sub/footer.tmpl", "_partials/sub/footer.tmpl"},
		{"_partials/deep/nested/partial.tmpl", "_partials/deep/nested/partial.tmpl"},
	}

	for _, tc := range testCases {
		t.Run(tc.sourcePath, func(t *testing.T) {
			result, err := routing.ResolveTarget(tc.sourcePath, targets, "")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if result.TargetPath != tc.expectedTarget {
				t.Errorf("expected TargetPath %q, got %q", tc.expectedTarget, result.TargetPath)
			}
		})
	}
}
