package rendering_test

import (
	"testing"

	"github.com/simensen/weftlo/internal/app/rendering"
)

// Test 1: Global variables only (no overrides)
func TestMergeVariables_GlobalVariablesOnly(t *testing.T) {
	t.Parallel()
	globalVars := map[string]interface{}{
		"company": "Acme Corp",
		"env":     "development",
	}

	merged := rendering.MergeVariables(globalVars)

	if merged["company"] != "Acme Corp" {
		t.Errorf("expected company 'Acme Corp', got %v", merged["company"])
	}
	if merged["env"] != "development" {
		t.Errorf("expected env 'development', got %v", merged["env"])
	}
}

// Test 2: Profile overrides global variables
func TestMergeVariables_ProfileOverridesGlobal(t *testing.T) {
	t.Parallel()
	globalVars := map[string]interface{}{
		"company": "Acme Corp",
		"env":     "development",
	}
	profileVars := map[string]interface{}{
		"env":     "staging",
		"version": "1.0.0",
	}

	merged := rendering.MergeVariables(globalVars, profileVars)

	if merged["company"] != "Acme Corp" {
		t.Errorf("expected company 'Acme Corp' (from global), got %v", merged["company"])
	}
	if merged["env"] != "staging" {
		t.Errorf("expected env 'staging' (overridden by profile), got %v", merged["env"])
	}
	if merged["version"] != "1.0.0" {
		t.Errorf("expected version '1.0.0' (from profile), got %v", merged["version"])
	}
}

// Test 3: Project overrides profile variables
func TestMergeVariables_ProjectOverridesProfile(t *testing.T) {
	t.Parallel()
	profileVars := map[string]interface{}{
		"env":     "staging",
		"version": "1.0.0",
	}
	projectVars := map[string]interface{}{
		"env":          "production",
		"project_name": "my-project",
	}

	merged := rendering.MergeVariables(profileVars, projectVars)

	if merged["env"] != "production" {
		t.Errorf("expected env 'production' (overridden by project), got %v", merged["env"])
	}
	if merged["version"] != "1.0.0" {
		t.Errorf("expected version '1.0.0' (from profile), got %v", merged["version"])
	}
	if merged["project_name"] != "my-project" {
		t.Errorf("expected project_name 'my-project' (from project), got %v", merged["project_name"])
	}
}

// Test 4: Full chain - global < profile < project
func TestMergeVariables_FullChain(t *testing.T) {
	t.Parallel()
	globalVars := map[string]interface{}{
		"company":   "Acme Corp",
		"env":       "development",
		"log_level": "debug",
	}
	profileVars := map[string]interface{}{
		"env":       "staging",
		"version":   "1.0.0",
		"log_level": "info",
	}
	projectVars := map[string]interface{}{
		"env":          "production",
		"project_name": "my-project",
	}

	merged := rendering.MergeVariables(globalVars, profileVars, projectVars)

	// company: from global (not overridden)
	if merged["company"] != "Acme Corp" {
		t.Errorf("expected company 'Acme Corp', got %v", merged["company"])
	}
	// env: overridden by project (final in chain)
	if merged["env"] != "production" {
		t.Errorf("expected env 'production', got %v", merged["env"])
	}
	// log_level: overridden by profile (not overridden by project)
	if merged["log_level"] != "info" {
		t.Errorf("expected log_level 'info', got %v", merged["log_level"])
	}
	// version: from profile (not overridden by project)
	if merged["version"] != "1.0.0" {
		t.Errorf("expected version '1.0.0', got %v", merged["version"])
	}
	// project_name: from project only
	if merged["project_name"] != "my-project" {
		t.Errorf("expected project_name 'my-project', got %v", merged["project_name"])
	}
}

// Test 5: Empty maps at each level
func TestMergeVariables_EmptyMaps(t *testing.T) {
	t.Parallel()
	// All empty maps
	merged := rendering.MergeVariables(
		map[string]interface{}{},
		map[string]interface{}{},
		map[string]interface{}{},
	)

	if len(merged) != 0 {
		t.Errorf("expected empty merged map, got %v", merged)
	}
}

// Test 6: Nil maps are handled gracefully
func TestMergeVariables_NilMaps(t *testing.T) {
	t.Parallel()
	globalVars := map[string]interface{}{
		"company": "Acme Corp",
	}

	// nil in the middle of the chain
	merged := rendering.MergeVariables(globalVars, nil, nil)

	if merged["company"] != "Acme Corp" {
		t.Errorf("expected company 'Acme Corp', got %v", merged["company"])
	}
	if len(merged) != 1 {
		t.Errorf("expected 1 variable, got %d", len(merged))
	}
}

// Test 7: All nil maps returns empty (not nil)
func TestMergeVariables_AllNilReturnsEmptyNotNil(t *testing.T) {
	t.Parallel()
	merged := rendering.MergeVariables(nil, nil, nil)

	if merged == nil {
		t.Error("expected empty map, got nil")
	}
	if len(merged) != 0 {
		t.Errorf("expected 0 variables, got %d", len(merged))
	}
}

// Test 8: No arguments returns empty map
func TestMergeVariables_NoArgs(t *testing.T) {
	t.Parallel()
	merged := rendering.MergeVariables()

	if merged == nil {
		t.Error("expected empty map, got nil")
	}
	if len(merged) != 0 {
		t.Errorf("expected 0 variables, got %d", len(merged))
	}
}

// Test 9: Mixed types of values
func TestMergeVariables_MixedValueTypes(t *testing.T) {
	t.Parallel()
	vars := map[string]interface{}{
		"string_val": "hello",
		"int_val":    42,
		"bool_val":   true,
		"float_val":  3.14,
		"nested": map[string]interface{}{
			"key": "value",
		},
	}

	merged := rendering.MergeVariables(vars)

	if merged["string_val"] != "hello" {
		t.Errorf("expected string_val 'hello', got %v", merged["string_val"])
	}
	if merged["int_val"] != 42 {
		t.Errorf("expected int_val 42, got %v", merged["int_val"])
	}
	if merged["bool_val"] != true {
		t.Errorf("expected bool_val true, got %v", merged["bool_val"])
	}

	nested, ok := merged["nested"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected nested to be map[string]interface{}, got %T", merged["nested"])
	}
	if nested["key"] != "value" {
		t.Errorf("expected nested.key 'value', got %v", nested["key"])
	}
}

// Test 10: Deep merge preserves nested keys from base when override provides partial map
// This test verifies that when using DeepMergeVariables, nested maps are recursively merged
// rather than replaced entirely. Non-conflicting nested keys from the base are preserved.
func TestDeepMergeVariables_NestedMapPreservation(t *testing.T) {
	t.Parallel()
	globalVars := map[string]interface{}{
		"database": map[string]interface{}{
			"host": "localhost",
			"port": 5432,
		},
	}
	projectVars := map[string]interface{}{
		"database": map[string]interface{}{
			"host": "production.db",
			// Note: port is NOT included here - it should be preserved from base
		},
	}

	// Use DeepMergeVariables which recursively merges nested maps
	merged, conflicts := rendering.DeepMergeVariables(globalVars, projectVars, "global config", "project config")

	// The nested database map should be deep merged (not replaced)
	db, ok := merged["database"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected database to be map[string]interface{}, got %T", merged["database"])
	}

	// host should be overridden by project
	if db["host"] != "production.db" {
		t.Errorf("expected database.host 'production.db', got %v", db["host"])
	}

	// port should be PRESERVED from base (deep merge preserves non-conflicting nested keys)
	if db["port"] != 5432 {
		t.Errorf("expected database.port 5432 (preserved from base), got %v", db["port"])
	}

	// Should have one conflict for database.host (was overridden)
	if len(conflicts) != 1 {
		t.Errorf("expected 1 conflict, got %d", len(conflicts))
	}
	if len(conflicts) > 0 && conflicts[0].Path != "database.host" {
		t.Errorf("expected conflict path 'database.host', got %v", conflicts[0].Path)
	}
}

// ============================================================================
// DeepMergeVariables Tests
// ============================================================================

// Test DeepMergeVariables: Basic scalar override (child scalar replaces parent scalar at same key)
func TestDeepMergeVariables_BasicScalarOverride(t *testing.T) {
	t.Parallel()
	base := map[string]interface{}{
		"env":     "development",
		"version": "1.0.0",
	}
	override := map[string]interface{}{
		"env": "production",
	}

	merged, conflicts := rendering.DeepMergeVariables(base, override, "parent-profile", "child-profile")

	// env should be overridden
	if merged["env"] != "production" {
		t.Errorf("expected env 'production', got %v", merged["env"])
	}
	// version should remain from base
	if merged["version"] != "1.0.0" {
		t.Errorf("expected version '1.0.0', got %v", merged["version"])
	}
	// Should have one conflict for env
	if len(conflicts) != 1 {
		t.Errorf("expected 1 conflict, got %d", len(conflicts))
	}
	if len(conflicts) > 0 && conflicts[0].Path != "env" {
		t.Errorf("expected conflict path 'env', got %v", conflicts[0].Path)
	}
}

// Test DeepMergeVariables: Deep merge of nested maps
func TestDeepMergeVariables_DeepMergeNestedMaps(t *testing.T) {
	t.Parallel()
	base := map[string]interface{}{
		"database": map[string]interface{}{
			"host": "localhost",
			"port": 5432,
		},
	}
	override := map[string]interface{}{
		"database": map[string]interface{}{
			"port":     5433,
			"username": "admin",
		},
	}

	merged, conflicts := rendering.DeepMergeVariables(base, override, "parent-profile", "child-profile")

	db, ok := merged["database"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected database to be map[string]interface{}, got %T", merged["database"])
	}
	// host should remain from base
	if db["host"] != "localhost" {
		t.Errorf("expected database.host 'localhost', got %v", db["host"])
	}
	// port should be overridden
	if db["port"] != 5433 {
		t.Errorf("expected database.port 5433, got %v", db["port"])
	}
	// username should be added from override
	if db["username"] != "admin" {
		t.Errorf("expected database.username 'admin', got %v", db["username"])
	}
	// Should have one conflict for database.port
	if len(conflicts) != 1 {
		t.Errorf("expected 1 conflict, got %d", len(conflicts))
	}
	if len(conflicts) > 0 && conflicts[0].Path != "database.port" {
		t.Errorf("expected conflict path 'database.port', got %v", conflicts[0].Path)
	}
}

// Test DeepMergeVariables: Type mismatch - parent has map, child has scalar
func TestDeepMergeVariables_TypeMismatchMapToScalar(t *testing.T) {
	t.Parallel()
	base := map[string]interface{}{
		"database": map[string]interface{}{
			"host": "localhost",
			"port": 5432,
		},
	}
	override := map[string]interface{}{
		"database": "sqlite://memory",
	}

	merged, _ := rendering.DeepMergeVariables(base, override, "parent-profile", "child-profile")

	// database should be completely replaced with scalar
	if merged["database"] != "sqlite://memory" {
		t.Errorf("expected database 'sqlite://memory', got %v", merged["database"])
	}
}

// Test DeepMergeVariables: Type mismatch - parent has scalar, child has map
func TestDeepMergeVariables_TypeMismatchScalarToMap(t *testing.T) {
	t.Parallel()
	base := map[string]interface{}{
		"database": "sqlite://memory",
	}
	override := map[string]interface{}{
		"database": map[string]interface{}{
			"host": "localhost",
			"port": 5432,
		},
	}

	merged, _ := rendering.DeepMergeVariables(base, override, "parent-profile", "child-profile")

	// database should be completely replaced with map
	db, ok := merged["database"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected database to be map[string]interface{}, got %T", merged["database"])
	}
	if db["host"] != "localhost" {
		t.Errorf("expected database.host 'localhost', got %v", db["host"])
	}
	if db["port"] != 5432 {
		t.Errorf("expected database.port 5432, got %v", db["port"])
	}
}

// Test DeepMergeVariables: Conflict tracking with correct dot-notation path
func TestDeepMergeVariables_ConflictTrackingDotNotation(t *testing.T) {
	t.Parallel()
	base := map[string]interface{}{
		"database": map[string]interface{}{
			"connection": map[string]interface{}{
				"host": "localhost",
			},
		},
	}
	override := map[string]interface{}{
		"database": map[string]interface{}{
			"connection": map[string]interface{}{
				"host": "production.db",
			},
		},
	}

	_, conflicts := rendering.DeepMergeVariables(base, override, "vendor/parent", "vendor/child")

	if len(conflicts) != 1 {
		t.Fatalf("expected 1 conflict, got %d", len(conflicts))
	}
	if conflicts[0].Path != "database.connection.host" {
		t.Errorf("expected conflict path 'database.connection.host', got %v", conflicts[0].Path)
	}
}

// Test DeepMergeVariables: Conflict info includes source and overriding profile names
func TestDeepMergeVariables_ConflictInfoProfileNames(t *testing.T) {
	t.Parallel()
	base := map[string]interface{}{
		"env": "development",
	}
	override := map[string]interface{}{
		"env": "production",
	}

	_, conflicts := rendering.DeepMergeVariables(base, override, "acme/base", "acme/child")

	if len(conflicts) != 1 {
		t.Fatalf("expected 1 conflict, got %d", len(conflicts))
	}
	if conflicts[0].SourceProfile != "acme/base" {
		t.Errorf("expected SourceProfile 'acme/base', got %v", conflicts[0].SourceProfile)
	}
	if conflicts[0].OverridingProfile != "acme/child" {
		t.Errorf("expected OverridingProfile 'acme/child', got %v", conflicts[0].OverridingProfile)
	}
}

// Test DeepMergeVariables: Nil and empty map handling
func TestDeepMergeVariables_NilEmptyMapHandling(t *testing.T) {
	t.Parallel()
	// Test nil base
	merged1, conflicts1 := rendering.DeepMergeVariables(nil, map[string]interface{}{"key": "value"}, "base", "override")
	if merged1["key"] != "value" {
		t.Errorf("expected key 'value' with nil base, got %v", merged1["key"])
	}
	if len(conflicts1) != 0 {
		t.Errorf("expected 0 conflicts with nil base, got %d", len(conflicts1))
	}

	// Test nil override
	merged2, conflicts2 := rendering.DeepMergeVariables(map[string]interface{}{"key": "value"}, nil, "base", "override")
	if merged2["key"] != "value" {
		t.Errorf("expected key 'value' with nil override, got %v", merged2["key"])
	}
	if len(conflicts2) != 0 {
		t.Errorf("expected 0 conflicts with nil override, got %d", len(conflicts2))
	}

	// Test both nil
	merged3, conflicts3 := rendering.DeepMergeVariables(nil, nil, "base", "override")
	if merged3 == nil {
		t.Error("expected empty map, got nil")
	}
	if len(merged3) != 0 {
		t.Errorf("expected 0 variables with both nil, got %d", len(merged3))
	}
	if len(conflicts3) != 0 {
		t.Errorf("expected 0 conflicts with both nil, got %d", len(conflicts3))
	}

	// Test empty maps
	merged4, conflicts4 := rendering.DeepMergeVariables(map[string]interface{}{}, map[string]interface{}{}, "base", "override")
	if len(merged4) != 0 {
		t.Errorf("expected 0 variables with empty maps, got %d", len(merged4))
	}
	if len(conflicts4) != 0 {
		t.Errorf("expected 0 conflicts with empty maps, got %d", len(conflicts4))
	}
}

// Test DeepMergeVariables: Deeply nested merge (3+ levels deep)
func TestDeepMergeVariables_DeeplyNestedMerge(t *testing.T) {
	t.Parallel()
	base := map[string]interface{}{
		"level1": map[string]interface{}{
			"level2": map[string]interface{}{
				"level3": map[string]interface{}{
					"level4": map[string]interface{}{
						"base_value":   "from_base",
						"shared_value": "base_version",
					},
				},
			},
		},
	}
	override := map[string]interface{}{
		"level1": map[string]interface{}{
			"level2": map[string]interface{}{
				"level3": map[string]interface{}{
					"level4": map[string]interface{}{
						"override_value": "from_override",
						"shared_value":   "override_version",
					},
				},
			},
		},
	}

	merged, conflicts := rendering.DeepMergeVariables(base, override, "parent", "child")

	// Navigate to level4
	l1, _ := merged["level1"].(map[string]interface{})
	l2, _ := l1["level2"].(map[string]interface{})
	l3, _ := l2["level3"].(map[string]interface{})
	l4, _ := l3["level4"].(map[string]interface{})

	// base_value should remain from base
	if l4["base_value"] != "from_base" {
		t.Errorf("expected base_value 'from_base', got %v", l4["base_value"])
	}
	// override_value should be added from override
	if l4["override_value"] != "from_override" {
		t.Errorf("expected override_value 'from_override', got %v", l4["override_value"])
	}
	// shared_value should be overridden
	if l4["shared_value"] != "override_version" {
		t.Errorf("expected shared_value 'override_version', got %v", l4["shared_value"])
	}

	// Should have one conflict for the deeply nested shared_value
	if len(conflicts) != 1 {
		t.Fatalf("expected 1 conflict, got %d", len(conflicts))
	}
	expectedPath := "level1.level2.level3.level4.shared_value"
	if conflicts[0].Path != expectedPath {
		t.Errorf("expected conflict path '%s', got %v", expectedPath, conflicts[0].Path)
	}
}

// ============================================================================
// DeepMergeVariablesChain Tests
// ============================================================================

// Test DeepMergeVariablesChain: Merging 2 sources (simple case)
func TestDeepMergeVariablesChain_TwoSources(t *testing.T) {
	t.Parallel()
	globalVars := map[string]interface{}{
		"company": "Acme Corp",
		"env":     "development",
		"database": map[string]interface{}{
			"host": "localhost",
			"port": 5432,
		},
	}
	projectVars := map[string]interface{}{
		"env": "production",
		"database": map[string]interface{}{
			"host": "prod.db.example.com",
		},
	}

	sources := []rendering.VariableSource{
		{Name: "global config", Vars: globalVars},
		{Name: "project config", Vars: projectVars},
	}

	merged, conflicts := rendering.DeepMergeVariablesChain(sources...)

	// company should remain from global
	if merged["company"] != "Acme Corp" {
		t.Errorf("expected company 'Acme Corp', got %v", merged["company"])
	}
	// env should be overridden by project
	if merged["env"] != "production" {
		t.Errorf("expected env 'production', got %v", merged["env"])
	}
	// database should be deep merged
	db, ok := merged["database"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected database to be map[string]interface{}, got %T", merged["database"])
	}
	// host should be overridden
	if db["host"] != "prod.db.example.com" {
		t.Errorf("expected database.host 'prod.db.example.com', got %v", db["host"])
	}
	// port should be preserved from global (deep merge)
	if db["port"] != 5432 {
		t.Errorf("expected database.port 5432, got %v", db["port"])
	}

	// Should have 2 conflicts: env and database.host
	if len(conflicts) != 2 {
		t.Errorf("expected 2 conflicts, got %d", len(conflicts))
	}
}

// Test DeepMergeVariablesChain: Merging 3+ sources (global, profile, project chain)
func TestDeepMergeVariablesChain_ThreeSources(t *testing.T) {
	t.Parallel()
	globalVars := map[string]interface{}{
		"company":   "Acme Corp",
		"env":       "development",
		"log_level": "debug",
		"database": map[string]interface{}{
			"host": "localhost",
			"port": 5432,
		},
	}
	profileVars := map[string]interface{}{
		"env":       "staging",
		"log_level": "info",
		"database": map[string]interface{}{
			"username": "staging_user",
		},
	}
	projectVars := map[string]interface{}{
		"env":          "production",
		"project_name": "my-project",
		"database": map[string]interface{}{
			"host": "prod.db.example.com",
		},
	}

	sources := []rendering.VariableSource{
		{Name: "global config", Vars: globalVars},
		{Name: "profile inheritance", Vars: profileVars},
		{Name: "project config", Vars: projectVars},
	}

	merged, conflicts := rendering.DeepMergeVariablesChain(sources...)

	// company should remain from global (not overridden)
	if merged["company"] != "Acme Corp" {
		t.Errorf("expected company 'Acme Corp', got %v", merged["company"])
	}
	// env should be overridden by project (final in chain)
	if merged["env"] != "production" {
		t.Errorf("expected env 'production', got %v", merged["env"])
	}
	// log_level should be overridden by profile (not overridden by project)
	if merged["log_level"] != "info" {
		t.Errorf("expected log_level 'info', got %v", merged["log_level"])
	}
	// project_name should be added from project
	if merged["project_name"] != "my-project" {
		t.Errorf("expected project_name 'my-project', got %v", merged["project_name"])
	}

	// database should be deep merged across all three levels
	db, ok := merged["database"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected database to be map[string]interface{}, got %T", merged["database"])
	}
	// host should be overridden by project
	if db["host"] != "prod.db.example.com" {
		t.Errorf("expected database.host 'prod.db.example.com', got %v", db["host"])
	}
	// port should be preserved from global
	if db["port"] != 5432 {
		t.Errorf("expected database.port 5432, got %v", db["port"])
	}
	// username should be added from profile
	if db["username"] != "staging_user" {
		t.Errorf("expected database.username 'staging_user', got %v", db["username"])
	}

	// Should have conflicts accumulated from both merge operations
	// Global -> Profile: env, log_level (2 conflicts)
	// Profile -> Project: env, database.host (2 conflicts)
	// Total: 4 conflicts
	if len(conflicts) != 4 {
		t.Errorf("expected 4 conflicts, got %d", len(conflicts))
	}
}

// Test DeepMergeVariablesChain: Conflict accumulation across multiple merge operations
func TestDeepMergeVariablesChain_ConflictAccumulation(t *testing.T) {
	t.Parallel()
	source1 := map[string]interface{}{
		"var1": "value1_from_s1",
		"var2": "value2_from_s1",
	}
	source2 := map[string]interface{}{
		"var1": "value1_from_s2",
		"var3": "value3_from_s2",
	}
	source3 := map[string]interface{}{
		"var1": "value1_from_s3",
		"var2": "value2_from_s3",
	}

	sources := []rendering.VariableSource{
		{Name: "source1", Vars: source1},
		{Name: "source2", Vars: source2},
		{Name: "source3", Vars: source3},
	}

	merged, conflicts := rendering.DeepMergeVariablesChain(sources...)

	// Final values should come from the last source that defined them
	if merged["var1"] != "value1_from_s3" {
		t.Errorf("expected var1 'value1_from_s3', got %v", merged["var1"])
	}
	if merged["var2"] != "value2_from_s3" {
		t.Errorf("expected var2 'value2_from_s3', got %v", merged["var2"])
	}
	if merged["var3"] != "value3_from_s2" {
		t.Errorf("expected var3 'value3_from_s2', got %v", merged["var3"])
	}

	// Verify conflict accumulation
	// source1 -> source2: var1 conflict (source1 -> source2)
	// source2 -> source3: var1 conflict (source2 -> source3), var2 conflict (source1 -> source3)
	// Total: 3 conflicts
	if len(conflicts) != 3 {
		t.Errorf("expected 3 conflicts, got %d", len(conflicts))
	}

	// Verify conflict source attribution
	var foundS1ToS2, foundS2ToS3Var1, foundS2ToS3Var2 bool
	for _, c := range conflicts {
		if c.Path == "var1" && c.SourceProfile == "source1" && c.OverridingProfile == "source2" {
			foundS1ToS2 = true
		}
		if c.Path == "var1" && c.SourceProfile == "source2" && c.OverridingProfile == "source3" {
			foundS2ToS3Var1 = true
		}
		if c.Path == "var2" && c.SourceProfile == "source2" && c.OverridingProfile == "source3" {
			foundS2ToS3Var2 = true
		}
	}
	if !foundS1ToS2 {
		t.Error("expected conflict for var1 from source1 to source2")
	}
	if !foundS2ToS3Var1 {
		t.Error("expected conflict for var1 from source2 to source3")
	}
	if !foundS2ToS3Var2 {
		t.Error("expected conflict for var2 from source2 to source3")
	}
}

// Test DeepMergeVariablesChain: Empty/nil variable handling
func TestDeepMergeVariablesChain_EmptyNilHandling(t *testing.T) {
	t.Parallel()
	// Test with no sources
	merged1, conflicts1 := rendering.DeepMergeVariablesChain()
	if merged1 == nil {
		t.Error("expected empty map with no sources, got nil")
	}
	if len(merged1) != 0 {
		t.Errorf("expected 0 variables with no sources, got %d", len(merged1))
	}
	if len(conflicts1) != 0 {
		t.Errorf("expected 0 conflicts with no sources, got %d", len(conflicts1))
	}

	// Test with nil vars in sources
	merged2, conflicts2 := rendering.DeepMergeVariablesChain(
		rendering.VariableSource{Name: "empty", Vars: nil},
		rendering.VariableSource{Name: "with-data", Vars: map[string]interface{}{"key": "value"}},
	)
	if merged2["key"] != "value" {
		t.Errorf("expected key 'value' with nil first source, got %v", merged2["key"])
	}
	if len(conflicts2) != 0 {
		t.Errorf("expected 0 conflicts with nil first source, got %d", len(conflicts2))
	}

	// Test with empty maps
	merged3, conflicts3 := rendering.DeepMergeVariablesChain(
		rendering.VariableSource{Name: "empty1", Vars: map[string]interface{}{}},
		rendering.VariableSource{Name: "empty2", Vars: map[string]interface{}{}},
	)
	if len(merged3) != 0 {
		t.Errorf("expected 0 variables with empty sources, got %d", len(merged3))
	}
	if len(conflicts3) != 0 {
		t.Errorf("expected 0 conflicts with empty sources, got %d", len(conflicts3))
	}

	// Test with single source (no merge needed)
	merged4, conflicts4 := rendering.DeepMergeVariablesChain(
		rendering.VariableSource{Name: "only", Vars: map[string]interface{}{"key": "value"}},
	)
	if merged4["key"] != "value" {
		t.Errorf("expected key 'value' with single source, got %v", merged4["key"])
	}
	if len(conflicts4) != 0 {
		t.Errorf("expected 0 conflicts with single source, got %d", len(conflicts4))
	}
}

// ============================================================================
// Task Group 6: Additional Strategic Tests for Gap Coverage
// ============================================================================

// Test DeepMergeVariablesChain: Type mismatch in chain context (map to scalar to map)
// This test verifies that type mismatches are handled correctly when they occur
// in the middle of a multi-source merge chain.
// NOTE: By design, type mismatches do NOT generate conflicts - only scalar-to-scalar
// overrides generate conflicts. This is documented in domain/profile/variable_merge.go.
func TestDeepMergeVariablesChain_TypeMismatchInChain(t *testing.T) {
	t.Parallel()
	// Source 1: database is a map
	source1 := map[string]interface{}{
		"database": map[string]interface{}{
			"host": "localhost",
			"port": 5432,
		},
	}
	// Source 2: database becomes a string (type mismatch - map replaced by scalar)
	source2 := map[string]interface{}{
		"database": "sqlite://memory",
	}
	// Source 3: database becomes a map again (type mismatch - scalar replaced by map)
	source3 := map[string]interface{}{
		"database": map[string]interface{}{
			"driver": "postgres",
		},
	}

	sources := []rendering.VariableSource{
		{Name: "source1", Vars: source1},
		{Name: "source2", Vars: source2},
		{Name: "source3", Vars: source3},
	}

	merged, conflicts := rendering.DeepMergeVariablesChain(sources...)

	// Final value should be the map from source3
	db, ok := merged["database"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected database to be map[string]interface{}, got %T", merged["database"])
	}
	if db["driver"] != "postgres" {
		t.Errorf("expected database.driver 'postgres', got %v", db["driver"])
	}

	// By design, type mismatches do NOT generate conflicts.
	// Only scalar-to-scalar overrides generate conflicts.
	// This test verifies the merge still works correctly even with type changes.
	if len(conflicts) != 0 {
		t.Errorf("expected 0 conflicts for type mismatches (by design), got %d", len(conflicts))
	}
}

// Test DeepMergeVariablesChain: Multiple nested conflicts in single merge operation
// This verifies that multiple conflicts within the same nested structure are all tracked.
func TestDeepMergeVariablesChain_MultipleNestedConflictsInSingleMerge(t *testing.T) {
	t.Parallel()
	source1 := map[string]interface{}{
		"config": map[string]interface{}{
			"setting1": "value1_s1",
			"setting2": "value2_s1",
			"setting3": "value3_s1",
			"nested": map[string]interface{}{
				"a": "a_s1",
				"b": "b_s1",
			},
		},
	}
	source2 := map[string]interface{}{
		"config": map[string]interface{}{
			"setting1": "value1_s2",
			"setting2": "value2_s2",
			"setting3": "value3_s2",
			"nested": map[string]interface{}{
				"a": "a_s2",
				"b": "b_s2",
			},
		},
	}

	sources := []rendering.VariableSource{
		{Name: "source1", Vars: source1},
		{Name: "source2", Vars: source2},
	}

	merged, conflicts := rendering.DeepMergeVariablesChain(sources...)

	// Verify all values are from source2
	config, _ := merged["config"].(map[string]interface{})
	if config["setting1"] != "value1_s2" {
		t.Errorf("expected setting1 'value1_s2', got %v", config["setting1"])
	}
	nested, _ := config["nested"].(map[string]interface{})
	if nested["a"] != "a_s2" {
		t.Errorf("expected nested.a 'a_s2', got %v", nested["a"])
	}

	// Should have 5 conflicts: setting1, setting2, setting3, nested.a, nested.b
	if len(conflicts) != 5 {
		t.Errorf("expected 5 conflicts for all overridden keys, got %d", len(conflicts))
	}

	// Verify all conflict paths are present
	conflictPaths := make(map[string]bool)
	for _, c := range conflicts {
		conflictPaths[c.Path] = true
	}
	expectedPaths := []string{"config.setting1", "config.setting2", "config.setting3", "config.nested.a", "config.nested.b"}
	for _, p := range expectedPaths {
		if !conflictPaths[p] {
			t.Errorf("expected conflict path '%s' not found", p)
		}
	}
}

// Test DeepMergeVariablesChain: Four sources to verify extended chain handling
// This tests a longer chain to ensure conflict accumulation works correctly.
func TestDeepMergeVariablesChain_FourSources(t *testing.T) {
	t.Parallel()
	source1 := map[string]interface{}{"key": "s1"}
	source2 := map[string]interface{}{"key": "s2"}
	source3 := map[string]interface{}{"key": "s3"}
	source4 := map[string]interface{}{"key": "s4"}

	sources := []rendering.VariableSource{
		{Name: "s1", Vars: source1},
		{Name: "s2", Vars: source2},
		{Name: "s3", Vars: source3},
		{Name: "s4", Vars: source4},
	}

	merged, conflicts := rendering.DeepMergeVariablesChain(sources...)

	// Final value should be from s4
	if merged["key"] != "s4" {
		t.Errorf("expected key 's4', got %v", merged["key"])
	}

	// Should have 3 conflicts: s1->s2, s2->s3, s3->s4
	if len(conflicts) != 3 {
		t.Errorf("expected 3 conflicts for 4-source chain, got %d", len(conflicts))
	}

	// Verify conflict chain
	var foundS1ToS2, foundS2ToS3, foundS3ToS4 bool
	for _, c := range conflicts {
		if c.SourceProfile == "s1" && c.OverridingProfile == "s2" {
			foundS1ToS2 = true
		}
		if c.SourceProfile == "s2" && c.OverridingProfile == "s3" {
			foundS2ToS3 = true
		}
		if c.SourceProfile == "s3" && c.OverridingProfile == "s4" {
			foundS3ToS4 = true
		}
	}
	if !foundS1ToS2 || !foundS2ToS3 || !foundS3ToS4 {
		t.Error("expected complete conflict chain s1->s2->s3->s4")
	}
}

// Test DeepMergeVariablesChain: Mixed nil sources in chain
// This tests that nil sources in the middle of the chain don't break the merge.
func TestDeepMergeVariablesChain_MixedNilSourcesInChain(t *testing.T) {
	t.Parallel()
	source1 := map[string]interface{}{
		"key1": "from_s1",
		"key2": "from_s1",
	}
	source3 := map[string]interface{}{
		"key2": "from_s3",
		"key3": "from_s3",
	}

	sources := []rendering.VariableSource{
		{Name: "s1", Vars: source1},
		{Name: "s2", Vars: nil}, // nil in the middle
		{Name: "s3", Vars: source3},
	}

	merged, conflicts := rendering.DeepMergeVariablesChain(sources...)

	// key1 should be from s1 (not overridden)
	if merged["key1"] != "from_s1" {
		t.Errorf("expected key1 'from_s1', got %v", merged["key1"])
	}
	// key2 should be from s3 (overrides s1, skipping nil s2)
	if merged["key2"] != "from_s3" {
		t.Errorf("expected key2 'from_s3', got %v", merged["key2"])
	}
	// key3 should be from s3
	if merged["key3"] != "from_s3" {
		t.Errorf("expected key3 'from_s3', got %v", merged["key3"])
	}

	// Should have 1 conflict: key2 from s1 overridden by s3
	// (nil s2 shouldn't create any conflicts)
	if len(conflicts) != 1 {
		t.Errorf("expected 1 conflict, got %d", len(conflicts))
	}
}

// Test DeepMergeVariablesChain: Deep merge preserves unrelated branches
// This ensures that deep merge only affects overlapping keys, not siblings.
func TestDeepMergeVariablesChain_PreservesUnrelatedBranches(t *testing.T) {
	t.Parallel()
	source1 := map[string]interface{}{
		"branch1": map[string]interface{}{
			"leaf1": "b1_l1",
			"leaf2": "b1_l2",
		},
		"branch2": map[string]interface{}{
			"leaf1": "b2_l1",
		},
	}
	source2 := map[string]interface{}{
		"branch1": map[string]interface{}{
			"leaf1": "b1_l1_override",
			// leaf2 not present - should be preserved from source1
		},
		// branch2 not present - should be preserved entirely from source1
	}

	sources := []rendering.VariableSource{
		{Name: "s1", Vars: source1},
		{Name: "s2", Vars: source2},
	}

	merged, conflicts := rendering.DeepMergeVariablesChain(sources...)

	// branch1.leaf1 should be overridden
	branch1, _ := merged["branch1"].(map[string]interface{})
	if branch1["leaf1"] != "b1_l1_override" {
		t.Errorf("expected branch1.leaf1 'b1_l1_override', got %v", branch1["leaf1"])
	}

	// branch1.leaf2 should be preserved from source1
	if branch1["leaf2"] != "b1_l2" {
		t.Errorf("expected branch1.leaf2 'b1_l2' (preserved), got %v", branch1["leaf2"])
	}

	// branch2 should be entirely preserved from source1
	branch2, _ := merged["branch2"].(map[string]interface{})
	if branch2["leaf1"] != "b2_l1" {
		t.Errorf("expected branch2.leaf1 'b2_l1' (preserved), got %v", branch2["leaf1"])
	}

	// Should only have 1 conflict for branch1.leaf1
	if len(conflicts) != 1 {
		t.Errorf("expected 1 conflict, got %d", len(conflicts))
	}
	if len(conflicts) > 0 && conflicts[0].Path != "branch1.leaf1" {
		t.Errorf("expected conflict path 'branch1.leaf1', got %v", conflicts[0].Path)
	}
}
