// Package rendering provides application-layer services for template rendering.
package rendering

import (
	domainprofile "github.com/simensen/weftlo/internal/domain/profile"
)

// VariableConflict is an alias for the domain type.
// This alias is provided for backwards compatibility and convenience.
type VariableConflict = domainprofile.VariableConflict

// VariableSource pairs a source name with its variable map for use in
// DeepMergeVariablesChain. The Name field is used for conflict reporting
// to identify where variables originated from.
type VariableSource struct {
	// Name identifies the source for conflict reporting (e.g., "global config", "profile inheritance", "project config")
	Name string
	// Vars contains the variable map from this source
	Vars map[string]interface{}
}

// Deprecated: MergeVariables performs shallow merge. Use DeepMergeVariables() or DeepMergeVariablesChain() instead.
// MergeVariables merges multiple variable maps with later maps overriding earlier ones.
// This implements the variable precedence chain: global < profile < project.
//
// Parameters:
//   - maps: Variadic list of variable maps to merge. Later maps override earlier ones.
//
// Returns:
//   - A new merged map containing all variables from all input maps.
//   - If all input maps are nil or empty, returns an empty map (not nil).
//
// Example:
//
//	globalVars := map[string]interface{}{"company": "Acme", "env": "dev"}
//	profileVars := map[string]interface{}{"env": "staging"}
//	projectVars := map[string]interface{}{"env": "production"}
//	merged := MergeVariables(globalVars, profileVars, projectVars)
//	// merged["company"] = "Acme" (from global)
//	// merged["env"] = "production" (overridden by project)
func MergeVariables(maps ...map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})

	for _, m := range maps {
		if m == nil {
			continue
		}
		for k, v := range m {
			result[k] = v
		}
	}

	return result
}

// DeepMergeVariables delegates to the domain layer implementation.
// This function is provided for backwards compatibility.
//
// See domainprofile.DeepMergeVariables for full documentation.
func DeepMergeVariables(base, override map[string]interface{}, baseProfile, overrideProfile string) (map[string]interface{}, []VariableConflict) {
	return domainprofile.DeepMergeVariables(base, override, baseProfile, overrideProfile)
}

// DeepMergeVariablesChain performs a sequential deep merge of multiple variable sources,
// accumulating conflicts from each merge operation. Sources are merged left-to-right,
// with later sources having higher priority (overriding earlier ones).
//
// This function is designed for the complete variable precedence chain:
// global config -> profile inheritance -> project config
//
// Parameters:
//   - sources: Variadic list of VariableSource structs to merge. Later sources override earlier ones.
//
// Returns:
//   - A new merged map containing all deep-merged variables from all input sources.
//   - A slice of VariableConflict describing all scalar value conflicts across all merge operations.
//
// Example:
//
//	sources := []rendering.VariableSource{
//	    {Name: "global config", Vars: globalVars},
//	    {Name: "profile inheritance", Vars: profileVars},
//	    {Name: "project config", Vars: projectVars},
//	}
//	merged, conflicts := rendering.DeepMergeVariablesChain(sources...)
func DeepMergeVariablesChain(sources ...VariableSource) (map[string]interface{}, []VariableConflict) {
	// Handle edge cases
	if len(sources) == 0 {
		return make(map[string]interface{}), nil
	}

	// Initialize with the first source
	result := copyVariableMap(sources[0].Vars)
	var allConflicts []VariableConflict
	previousSourceName := sources[0].Name

	// Iterate through remaining sources, merging each one
	for i := 1; i < len(sources); i++ {
		source := sources[i]
		merged, conflicts := domainprofile.DeepMergeVariables(result, source.Vars, previousSourceName, source.Name)
		result = merged
		allConflicts = append(allConflicts, conflicts...)
		previousSourceName = source.Name
	}

	return result, allConflicts
}

// copyVariableMap creates a deep copy of a variable map.
// This is a local helper to avoid depending on unexported functions from the domain layer.
func copyVariableMap(m map[string]interface{}) map[string]interface{} {
	if m == nil {
		return make(map[string]interface{})
	}
	result := make(map[string]interface{}, len(m))
	for k, v := range m {
		result[k] = deepCopyVariableValue(v)
	}
	return result
}

// deepCopyVariableValue creates a deep copy of a value, recursively copying nested maps.
func deepCopyVariableValue(v interface{}) interface{} {
	if m, ok := v.(map[string]interface{}); ok {
		return copyVariableMap(m)
	}
	// For non-map values (scalars), return as-is since they're immutable
	return v
}
