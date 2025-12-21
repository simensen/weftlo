// Package routing provides services for transforming content source paths to target installation paths.
// This file implements ContentConfig merge functionality for profile inheritance
// and target override merging for the four-level override precedence chain.
package routing

import (
	"github.com/simensen/weftlo/internal/domain/profile"
)

// MergeContentConfigs merges two ContentConfig structs with child-overrides-parent semantics.
//
// This function performs a shallow merge of ContentConfig fields:
//   - Root: child value takes precedence if non-empty, otherwise parent value is used
//   - DefaultTarget: child value takes precedence if non-empty, otherwise parent value is used
//   - Targets: shallow map merge where child keys override parent keys, but both are preserved
//
// The merge follows the same pattern as MergedProfile.MergeWith in
// internal/infrastructure/profile/merged_profile.go.
//
// Example target merge:
//
//	parent: {a: "x", b: "y"}
//	child:  {b: "z", c: "w"}
//	result: {a: "x", b: "z", c: "w"}
//
// Parameters:
//   - parent: the base ContentConfig (from parent profile)
//   - child: the overriding ContentConfig (from child profile)
//
// Returns:
//   - A new ContentConfig with merged values
func MergeContentConfigs(parent, child profile.ContentConfig) profile.ContentConfig {
	result := profile.ContentConfig{}

	// Merge Root: child takes precedence if non-empty
	if child.Root != "" {
		result.Root = child.Root
	} else {
		result.Root = parent.Root
	}

	// Merge DefaultTarget: child takes precedence if non-empty
	if child.DefaultTarget != "" {
		result.DefaultTarget = child.DefaultTarget
	} else {
		result.DefaultTarget = parent.DefaultTarget
	}

	// Merge Targets: shallow map merge with child keys overriding parent keys
	result.Targets = mergeTargetsMaps(parent.Targets, child.Targets)

	return result
}

// mergeTargetsMaps performs a shallow merge of two target maps.
// Child keys override parent keys, but keys present only in parent are preserved.
//
// Returns a new map; neither input map is modified.
func mergeTargetsMaps(parent, child map[string]string) map[string]string {
	// Handle nil/empty cases
	if len(parent) == 0 && len(child) == 0 {
		return nil
	}

	// Calculate capacity for the merged map (sum of both for potential unique keys)
	capacity := len(parent) + len(child)

	result := make(map[string]string, capacity)

	// Copy parent values first
	for key, value := range parent {
		result[key] = value
	}

	// Override with child values (child keys win)
	for key, value := range child {
		result[key] = value
	}

	return result
}

// =============================================================================
// Target Override Merging
// =============================================================================

// MergeOverrides merges multiple target override maps with later maps overriding earlier ones.
// This implements the four-level target override precedence chain:
//
//	Level 1 (lowest): Profile targets converted via ConvertTargetsToOverrides()
//	Level 2: Global universal overrides (target_overrides from ~/.config/weftlo/config.yaml)
//	Level 3: Global profile-specific overrides (profile_overrides.<name>.target_overrides)
//	Level 4 (highest): Project overrides (target_overrides from .weftlo.yaml)
//
// Later maps in the variadic parameter list override values from earlier maps.
// Keys present only in earlier maps are preserved in the result.
//
// The function preserves nil pointer values (suppression semantics):
//   - A nil pointer value means "suppress this target entirely"
//   - A later level can unsuppress a target by providing a non-nil pointer value
//   - A later level can suppress a target by providing a nil pointer value
//
// Parameters:
//   - maps: Variadic list of override maps to merge. Later maps override earlier ones.
//
// Returns:
//   - A new merged map containing all overrides from all input maps.
//   - If all input maps are nil or empty, returns an empty map (not nil).
//
// Example:
//
//	globalOverrides := map[string]*string{"cursor": nil}  // Suppress cursor globally
//	projectOverrides := map[string]*string{"cursor": ptr(".cursor-config")}  // Unsuppress
//	merged := MergeOverrides(globalOverrides, projectOverrides)
//	// merged["cursor"] = ptr(".cursor-config") (unsuppressed by project)
func MergeOverrides(maps ...map[string]*string) map[string]*string {
	result := make(map[string]*string)

	for _, m := range maps {
		if m == nil {
			continue
		}
		for k, v := range m {
			// Note: We explicitly set the value even if v is nil
			// This preserves nil pointer semantics (suppression) from later maps
			result[k] = v
		}
	}

	return result
}

// ConvertTargetsToOverrides converts a profile's target map to override format.
// This is used to create the Level 1 (base) layer in the four-level override chain.
//
// Profile targets use map[string]string where each value is a literal target path.
// Override maps use map[string]*string where nil values indicate suppression.
//
// The conversion rules are:
//   - Non-empty string values become pointer-to-string
//   - Empty string values become pointer-to-empty-string (not nil)
//     This preserves the "redirect to project root" semantic from profile targets
//
// Note: Profile targets cannot express suppression (nil) since map[string]string
// cannot contain nil values. Suppression can only be introduced at higher override levels.
//
// Parameters:
//   - targets: Map of target names to target paths from profile.yaml content.targets
//
// Returns:
//   - A new map in override format (map[string]*string)
//   - If input is nil or empty, returns an empty map (not nil)
//
// Example:
//
//	targets := map[string]string{"claude": ".claude", "root": ""}
//	overrides := ConvertTargetsToOverrides(targets)
//	// overrides["claude"] = ptr(".claude")
//	// overrides["root"] = ptr("") (pointer to empty string, not nil)
func ConvertTargetsToOverrides(targets map[string]string) map[string]*string {
	result := make(map[string]*string)

	if targets == nil {
		return result
	}

	for k, v := range targets {
		// Create a copy of the value to get a unique pointer for each entry
		valueCopy := v
		result[k] = &valueCopy
	}

	return result
}
