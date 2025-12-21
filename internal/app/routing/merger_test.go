// Package routing_test contains tests for the routing package.
package routing_test

import (
	"testing"

	"github.com/simensen/weftlo/internal/app/routing"
	"github.com/simensen/weftlo/internal/domain/profile"
)

// =============================================================================
// Task Group 2: ContentConfig Merger Tests
// =============================================================================

// Test shallow merge preserves both parent and child target keys
func TestMergeContentConfigs_ShallowMergePreservesBothParentAndChildTargetKeys(t *testing.T) {
	t.Parallel()
	parent := profile.ContentConfig{
		Targets: map[string]string{
			"claude": ".claude",
			"shared": "weftlo/shared",
		},
	}

	child := profile.ContentConfig{
		Targets: map[string]string{
			"cursor": ".cursor",
			"custom": "custom-path",
		},
	}

	result := routing.MergeContentConfigs(parent, child)

	// Verify all parent keys are preserved
	if result.Targets["claude"] != ".claude" {
		t.Errorf("expected parent key 'claude' to be preserved with value '.claude', got: %s", result.Targets["claude"])
	}
	if result.Targets["shared"] != "weftlo/shared" {
		t.Errorf("expected parent key 'shared' to be preserved with value 'weftlo/shared', got: %s", result.Targets["shared"])
	}

	// Verify all child keys are present
	if result.Targets["cursor"] != ".cursor" {
		t.Errorf("expected child key 'cursor' to be present with value '.cursor', got: %s", result.Targets["cursor"])
	}
	if result.Targets["custom"] != "custom-path" {
		t.Errorf("expected child key 'custom' to be present with value 'custom-path', got: %s", result.Targets["custom"])
	}

	// Verify total count is correct (4 unique keys)
	if len(result.Targets) != 4 {
		t.Errorf("expected 4 targets in merged result, got: %d", len(result.Targets))
	}
}

// Test child target values override parent target values for same key
func TestMergeContentConfigs_ChildTargetValuesOverrideParentForSameKey(t *testing.T) {
	t.Parallel()
	parent := profile.ContentConfig{
		Targets: map[string]string{
			"claude": ".claude",
			"shared": "weftlo/shared",
		},
	}

	child := profile.ContentConfig{
		Targets: map[string]string{
			"claude": ".ai/claude", // Override parent value
			"cursor": ".cursor",    // New key
		},
	}

	result := routing.MergeContentConfigs(parent, child)

	// Verify child value overrides parent value for the same key
	if result.Targets["claude"] != ".ai/claude" {
		t.Errorf("expected child value '.ai/claude' to override parent for key 'claude', got: %s", result.Targets["claude"])
	}

	// Verify parent key not in child is preserved
	if result.Targets["shared"] != "weftlo/shared" {
		t.Errorf("expected parent key 'shared' to be preserved, got: %s", result.Targets["shared"])
	}

	// Verify new child key is present
	if result.Targets["cursor"] != ".cursor" {
		t.Errorf("expected child key 'cursor' to be present, got: %s", result.Targets["cursor"])
	}
}

// Test child root takes precedence over parent root
func TestMergeContentConfigs_ChildRootTakesPrecedenceOverParentRoot(t *testing.T) {
	t.Parallel()
	parent := profile.ContentConfig{
		Root: "parent-content",
	}

	child := profile.ContentConfig{
		Root: "child-content",
	}

	result := routing.MergeContentConfigs(parent, child)

	if result.Root != "child-content" {
		t.Errorf("expected child root 'child-content' to take precedence, got: %s", result.Root)
	}
}

// Test child default_target takes precedence over parent default_target
func TestMergeContentConfigs_ChildDefaultTargetTakesPrecedenceOverParentDefaultTarget(t *testing.T) {
	t.Parallel()
	parent := profile.ContentConfig{
		DefaultTarget: "parent-default",
	}

	child := profile.ContentConfig{
		DefaultTarget: "child-default",
	}

	result := routing.MergeContentConfigs(parent, child)

	if result.DefaultTarget != "child-default" {
		t.Errorf("expected child default_target 'child-default' to take precedence, got: %s", result.DefaultTarget)
	}
}

// Test empty child config preserves all parent values
func TestMergeContentConfigs_EmptyChildConfigPreservesAllParentValues(t *testing.T) {
	t.Parallel()
	parent := profile.ContentConfig{
		Root:          "parent-content",
		DefaultTarget: "parent-default",
		Targets: map[string]string{
			"claude": ".claude",
			"shared": "weftlo/shared",
		},
	}

	child := profile.ContentConfig{
		// All fields empty/nil
	}

	result := routing.MergeContentConfigs(parent, child)

	// Verify parent root is preserved
	if result.Root != "parent-content" {
		t.Errorf("expected parent root to be preserved, got: %s", result.Root)
	}

	// Verify parent default_target is preserved
	if result.DefaultTarget != "parent-default" {
		t.Errorf("expected parent default_target to be preserved, got: %s", result.DefaultTarget)
	}

	// Verify parent targets are preserved
	if result.Targets["claude"] != ".claude" {
		t.Errorf("expected parent target 'claude' to be preserved, got: %s", result.Targets["claude"])
	}
	if result.Targets["shared"] != "weftlo/shared" {
		t.Errorf("expected parent target 'shared' to be preserved, got: %s", result.Targets["shared"])
	}
}

// Test merge with nil/empty parent returns child unchanged
func TestMergeContentConfigs_NilOrEmptyParentReturnsChildUnchanged(t *testing.T) {
	t.Parallel()
	child := profile.ContentConfig{
		Root:          "child-content",
		DefaultTarget: "child-default",
		Targets: map[string]string{
			"claude": ".claude",
		},
	}

	// Test with empty parent (all fields empty/nil)
	emptyParent := profile.ContentConfig{}

	result := routing.MergeContentConfigs(emptyParent, child)

	// Verify child values are returned unchanged
	if result.Root != "child-content" {
		t.Errorf("expected child root to be unchanged, got: %s", result.Root)
	}
	if result.DefaultTarget != "child-default" {
		t.Errorf("expected child default_target to be unchanged, got: %s", result.DefaultTarget)
	}
	if result.Targets["claude"] != ".claude" {
		t.Errorf("expected child target 'claude' to be unchanged, got: %s", result.Targets["claude"])
	}
}

// Test merge with nil/empty child returns parent unchanged
func TestMergeContentConfigs_NilOrEmptyChildReturnsParentUnchanged(t *testing.T) {
	t.Parallel()
	parent := profile.ContentConfig{
		Root:          "parent-content",
		DefaultTarget: "parent-default",
		Targets: map[string]string{
			"claude": ".claude",
		},
	}

	// Test with empty child (all fields empty/nil)
	emptyChild := profile.ContentConfig{}

	result := routing.MergeContentConfigs(parent, emptyChild)

	// Verify parent values are returned unchanged
	if result.Root != "parent-content" {
		t.Errorf("expected parent root to be unchanged, got: %s", result.Root)
	}
	if result.DefaultTarget != "parent-default" {
		t.Errorf("expected parent default_target to be unchanged, got: %s", result.DefaultTarget)
	}
	if result.Targets["claude"] != ".claude" {
		t.Errorf("expected parent target 'claude' to be unchanged, got: %s", result.Targets["claude"])
	}
}

// =============================================================================
// Task Group 3: MergeOverrides Tests
// =============================================================================

// Test two-map merge (global overrides < project overrides)
func TestMergeOverrides_TwoMapMergeGlobalLessThanProject(t *testing.T) {
	t.Parallel()
	globalPath := ".claude/global"
	projectPath := ".claude/project"

	globalOverrides := map[string]*string{
		"claude": &globalPath,
		"cursor": nil, // Global suppresses cursor
	}

	projectOverrides := map[string]*string{
		"claude": &projectPath, // Project overrides global
		"agents": &globalPath,  // Project adds new key
	}

	result := routing.MergeOverrides(globalOverrides, projectOverrides)

	// Verify project value overrides global value for same key
	if result["claude"] == nil || *result["claude"] != ".claude/project" {
		t.Errorf("expected project value '.claude/project' to override global for key 'claude'")
	}

	// Verify global key not in project is preserved
	if result["cursor"] != nil {
		t.Errorf("expected global suppression (nil) to be preserved for key 'cursor'")
	}

	// Verify project key not in global is added
	if result["agents"] == nil || *result["agents"] != ".claude/global" {
		t.Errorf("expected project key 'agents' to be added")
	}
}

// Test four-level merge (profile < global universal < global profile-specific < project)
func TestMergeOverrides_FourLevelMergePrecedence(t *testing.T) {
	t.Parallel()
	profilePath := ".profile/path"
	globalUniversalPath := ".global/universal"
	globalProfilePath := ".global/profile-specific"
	projectPath := ".project/path"

	// Level 1: Profile targets (converted to overrides)
	profileOverrides := map[string]*string{
		"claude":   &profilePath,
		"cursor":   &profilePath,
		"agents":   &profilePath,
		"commands": &profilePath,
	}

	// Level 2: Global universal overrides
	globalUniversalOverrides := map[string]*string{
		"cursor":   &globalUniversalPath, // Override profile
		"agents":   &globalUniversalPath, // Override profile
		"commands": &globalUniversalPath, // Override profile
	}

	// Level 3: Global profile-specific overrides
	globalProfileOverrides := map[string]*string{
		"agents":   &globalProfilePath, // Override global universal
		"commands": &globalProfilePath, // Override global universal
	}

	// Level 4: Project overrides
	projectOverrides := map[string]*string{
		"commands": &projectPath, // Override global profile-specific
	}

	result := routing.MergeOverrides(profileOverrides, globalUniversalOverrides, globalProfileOverrides, projectOverrides)

	// claude: should be from profile (not overridden by any later level)
	if result["claude"] == nil || *result["claude"] != ".profile/path" {
		t.Errorf("expected 'claude' to be '.profile/path' from profile level")
	}

	// cursor: should be from global universal (overrides profile)
	if result["cursor"] == nil || *result["cursor"] != ".global/universal" {
		t.Errorf("expected 'cursor' to be '.global/universal' from global universal level")
	}

	// agents: should be from global profile-specific (overrides global universal)
	if result["agents"] == nil || *result["agents"] != ".global/profile-specific" {
		t.Errorf("expected 'agents' to be '.global/profile-specific' from global profile-specific level")
	}

	// commands: should be from project (overrides all others)
	if result["commands"] == nil || *result["commands"] != ".project/path" {
		t.Errorf("expected 'commands' to be '.project/path' from project level")
	}
}

// Test nil pointer handling (suppression semantics preserved through merge)
func TestMergeOverrides_NilPointerSuppressionPreserved(t *testing.T) {
	t.Parallel()
	activePath := ".active/path"

	// First map has active value
	firstMap := map[string]*string{
		"claude": &activePath,
	}

	// Second map suppresses (nil pointer)
	secondMap := map[string]*string{
		"claude": nil,
	}

	result := routing.MergeOverrides(firstMap, secondMap)

	// Verify nil (suppression) is preserved from second map
	if result["claude"] != nil {
		t.Errorf("expected 'claude' to be nil (suppressed) from second map")
	}
}

// Test later levels unsuppressing earlier suppressed targets
func TestMergeOverrides_LaterLevelUnsuppressesTarget(t *testing.T) {
	t.Parallel()
	cursorPath := ".cursor-config"

	// Global suppresses cursor
	globalOverrides := map[string]*string{
		"cursor": nil,
	}

	// Project unsuppresses cursor by providing a value
	projectOverrides := map[string]*string{
		"cursor": &cursorPath,
	}

	result := routing.MergeOverrides(globalOverrides, projectOverrides)

	// Verify project value unsuppresses the target
	if result["cursor"] == nil {
		t.Errorf("expected project to unsuppress 'cursor' target")
	}
	if *result["cursor"] != ".cursor-config" {
		t.Errorf("expected 'cursor' to be '.cursor-config', got: %s", *result["cursor"])
	}
}

// Test nil/empty map inputs handled gracefully
func TestMergeOverrides_NilAndEmptyMapsHandledGracefully(t *testing.T) {
	t.Parallel()
	activePath := ".active/path"

	activeMap := map[string]*string{
		"claude": &activePath,
	}

	// Mix of nil, empty, and populated maps
	result := routing.MergeOverrides(nil, map[string]*string{}, activeMap, nil)

	// Verify active map values are present
	if result["claude"] == nil || *result["claude"] != ".active/path" {
		t.Errorf("expected 'claude' to be '.active/path' from active map")
	}

	// Verify result is not nil
	if result == nil {
		t.Errorf("expected non-nil result map")
	}
}

// Test empty result when all inputs nil
func TestMergeOverrides_EmptyResultWhenAllInputsNil(t *testing.T) {
	t.Parallel()
	result := routing.MergeOverrides(nil, nil, nil)

	// Verify result is empty map (not nil)
	if result == nil {
		t.Errorf("expected empty map, got nil")
	}

	if len(result) != 0 {
		t.Errorf("expected empty map with 0 keys, got %d keys", len(result))
	}
}

// Test empty result when all inputs are empty maps
func TestMergeOverrides_EmptyResultWhenAllInputsEmpty(t *testing.T) {
	t.Parallel()
	result := routing.MergeOverrides(map[string]*string{}, map[string]*string{})

	// Verify result is empty map (not nil)
	if result == nil {
		t.Errorf("expected empty map, got nil")
	}

	if len(result) != 0 {
		t.Errorf("expected empty map with 0 keys, got %d keys", len(result))
	}
}

// =============================================================================
// Task Group 3: ConvertTargetsToOverrides Tests
// =============================================================================

// Test ConvertTargetsToOverrides converts profile targets to override format
func TestConvertTargetsToOverrides_ConvertsNonEmptyStringsToPointers(t *testing.T) {
	t.Parallel()
	targets := map[string]string{
		"claude": ".claude",
		"cursor": ".cursor/config",
	}

	result := routing.ConvertTargetsToOverrides(targets)

	// Verify each non-empty string becomes pointer-to-string
	if result["claude"] == nil || *result["claude"] != ".claude" {
		t.Errorf("expected 'claude' to be pointer to '.claude'")
	}
	if result["cursor"] == nil || *result["cursor"] != ".cursor/config" {
		t.Errorf("expected 'cursor' to be pointer to '.cursor/config'")
	}
}

// Test ConvertTargetsToOverrides handles empty string values
func TestConvertTargetsToOverrides_EmptyStringBecomesPointerToEmptyString(t *testing.T) {
	t.Parallel()
	targets := map[string]string{
		"root-content": "", // Empty string means project root
	}

	result := routing.ConvertTargetsToOverrides(targets)

	// Verify empty string becomes pointer-to-empty-string (not nil)
	if result["root-content"] == nil {
		t.Errorf("expected 'root-content' to be pointer to empty string, got nil")
	}
	if *result["root-content"] != "" {
		t.Errorf("expected 'root-content' to be pointer to empty string, got: %s", *result["root-content"])
	}
}

// Test ConvertTargetsToOverrides handles nil input
func TestConvertTargetsToOverrides_NilInputReturnsEmptyMap(t *testing.T) {
	t.Parallel()
	result := routing.ConvertTargetsToOverrides(nil)

	// Verify result is empty map (not nil)
	if result == nil {
		t.Errorf("expected empty map, got nil")
	}

	if len(result) != 0 {
		t.Errorf("expected empty map with 0 keys, got %d keys", len(result))
	}
}

// Test ConvertTargetsToOverrides handles empty input
func TestConvertTargetsToOverrides_EmptyInputReturnsEmptyMap(t *testing.T) {
	t.Parallel()
	result := routing.ConvertTargetsToOverrides(map[string]string{})

	// Verify result is empty map (not nil)
	if result == nil {
		t.Errorf("expected empty map, got nil")
	}

	if len(result) != 0 {
		t.Errorf("expected empty map with 0 keys, got %d keys", len(result))
	}
}
