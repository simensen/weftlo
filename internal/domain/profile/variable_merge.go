package profile

// DeepMergeVariables performs a deep merge of two variable maps, recursively merging
// nested maps while tracking conflicts where scalar values are overridden.
//
// This function is used for merging variables within a profile inheritance chain,
// where child profiles should extend and override parent profile variables with
// deep merge semantics.
//
// Merge behavior:
//   - When both base and override have a map at the same key, the maps are merged recursively
//   - When there's a type mismatch (one has map, other has scalar), the override completely replaces the base
//   - When both have scalar values at the same key, the override replaces the base (conflict tracked)
//   - Keys only in base are preserved in the result
//   - Keys only in override are added to the result
//
// Parameters:
//   - base: The base variable map (e.g., from parent profile)
//   - override: The override variable map (e.g., from child profile)
//   - baseProfile: Name of the base profile for conflict tracking (e.g., "vendor/parent")
//   - overrideProfile: Name of the override profile for conflict tracking (e.g., "vendor/child")
//
// Returns:
//   - A new merged map containing the deep-merged variables
//   - A slice of VariableConflict describing all scalar value conflicts
//
// Example:
//
//	base := map[string]interface{}{"database": map[string]interface{}{"host": "localhost", "port": 5432}}
//	override := map[string]interface{}{"database": map[string]interface{}{"port": 5433, "user": "admin"}}
//	merged, conflicts := DeepMergeVariables(base, override, "parent", "child")
//	// merged["database"]["host"] = "localhost" (from base)
//	// merged["database"]["port"] = 5433 (from override, conflict recorded)
//	// merged["database"]["user"] = "admin" (from override)
func DeepMergeVariables(base, override map[string]interface{}, baseProfile, overrideProfile string) (map[string]interface{}, []VariableConflict) {
	// Handle nil/empty cases
	if base == nil && override == nil {
		return make(map[string]interface{}), nil
	}
	if base == nil {
		return copyVariableMap(override), nil
	}
	if override == nil {
		return copyVariableMap(base), nil
	}

	result := make(map[string]interface{})
	var conflicts []VariableConflict

	// Start recursive merge with empty path prefix
	deepMergeRecursive(base, override, "", baseProfile, overrideProfile, result, &conflicts)

	return result, conflicts
}

// deepMergeRecursive is the internal recursive helper for DeepMergeVariables.
// It handles the actual deep merge logic while building dot-notation paths for conflict tracking.
//
// Parameters:
//   - base: The base map at current recursion level
//   - override: The override map at current recursion level
//   - pathPrefix: The dot-notation path built during recursion (e.g., "database.connection")
//   - baseProfile: Name of the base profile for conflict tracking
//   - overrideProfile: Name of the override profile for conflict tracking
//   - result: The result map to populate (passed by reference for efficiency)
//   - conflicts: Slice of conflicts to accumulate (passed by pointer for mutation)
func deepMergeRecursive(
	base, override map[string]interface{},
	pathPrefix string,
	baseProfile, overrideProfile string,
	result map[string]interface{},
	conflicts *[]VariableConflict,
) {
	// First, copy all keys from base to result
	for key, baseValue := range base {
		currentPath := buildVariablePath(pathPrefix, key)

		overrideValue, existsInOverride := override[key]

		if !existsInOverride {
			// Key only exists in base, copy it (deep copy if map)
			result[key] = deepCopyVariableValue(baseValue)
			continue
		}

		// Key exists in both base and override
		baseMap, baseIsMap := baseValue.(map[string]interface{})
		overrideMap, overrideIsMap := overrideValue.(map[string]interface{})

		switch {
		case baseIsMap && overrideIsMap:
			// Both are maps: recursively merge
			nestedResult := make(map[string]interface{})
			deepMergeRecursive(baseMap, overrideMap, currentPath, baseProfile, overrideProfile, nestedResult, conflicts)
			result[key] = nestedResult

		case baseIsMap && !overrideIsMap:
			// Type mismatch: base has map, override has scalar - override wins completely
			result[key] = deepCopyVariableValue(overrideValue)

		case !baseIsMap && overrideIsMap:
			// Type mismatch: base has scalar, override has map - override wins completely
			result[key] = deepCopyVariableValue(overrideValue)

		default:
			// Both are scalars: override wins, track conflict if values differ
			result[key] = deepCopyVariableValue(overrideValue)
			if !variableValuesEqual(baseValue, overrideValue) {
				*conflicts = append(*conflicts, VariableConflict{
					Path:              currentPath,
					SourceProfile:     baseProfile,
					OverridingProfile: overrideProfile,
				})
			}
		}
	}

	// Add keys that only exist in override
	for key, overrideValue := range override {
		if _, existsInBase := base[key]; !existsInBase {
			result[key] = deepCopyVariableValue(overrideValue)
		}
	}
}

// buildVariablePath constructs the dot-notation path for conflict tracking.
func buildVariablePath(prefix, key string) string {
	if prefix == "" {
		return key
	}
	return prefix + "." + key
}

// copyVariableMap creates a deep copy of a variable map.
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

// variableValuesEqual checks if two values are equal for conflict detection purposes.
func variableValuesEqual(a, b interface{}) bool {
	// Simple equality check - works for comparable types
	return a == b
}
