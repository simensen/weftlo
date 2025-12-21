package config_test

import (
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/simensen/weftlo/internal/domain/config"
	"github.com/simensen/weftlo/internal/testutil"
)

// Test 1: GlobalConfig struct instantiation and field access
func TestGlobalConfig_Instantiation(t *testing.T) {
	t.Parallel()
	cfg := config.GlobalConfig{
		DefaultProfile: "vendor/my-profile",
	}

	if cfg.DefaultProfile != "vendor/my-profile" {
		t.Errorf("expected DefaultProfile to be 'vendor/my-profile', got '%s'", cfg.DefaultProfile)
	}
}

// Test 2: ProjectConfig struct instantiation and field access
func TestProjectConfig_Instantiation(t *testing.T) {
	t.Parallel()
	cfg := config.ProjectConfig{
		Profiles: []string{"vendor/project-profile"},
	}

	if len(cfg.Profiles) != 1 || cfg.Profiles[0] != "vendor/project-profile" {
		t.Errorf("expected Profiles to be ['vendor/project-profile'], got %v", cfg.Profiles)
	}
}

// Test 3: YAML serialization/deserialization with snake_case tags
func TestGlobalConfig_YAMLSerialization(t *testing.T) {
	t.Parallel()
	cfg := config.GlobalConfig{
		DefaultProfile: "vendor/my-profile",
	}

	// Serialize to YAML
	data, err := yaml.Marshal(cfg)
	if err != nil {
		t.Fatalf("failed to marshal GlobalConfig to YAML: %v", err)
	}

	yamlStr := string(data)

	// Verify snake_case tags are used
	if !testutil.ContainsString(yamlStr, "default_profile: vendor/my-profile") {
		t.Errorf("expected YAML to contain 'default_profile: vendor/my-profile', got:\n%s", yamlStr)
	}

	// Deserialize back
	var cfg2 config.GlobalConfig
	err = yaml.Unmarshal(data, &cfg2)
	if err != nil {
		t.Fatalf("failed to unmarshal GlobalConfig from YAML: %v", err)
	}

	if cfg2.DefaultProfile != cfg.DefaultProfile {
		t.Errorf("round-trip failed: expected DefaultProfile '%s', got '%s'", cfg.DefaultProfile, cfg2.DefaultProfile)
	}
}

// Test 4: Omitempty behavior for optional fields
func TestGlobalConfig_OmitemptyBehavior(t *testing.T) {
	t.Parallel()
	// GlobalConfig with empty DefaultProfile
	cfg := config.GlobalConfig{
		DefaultProfile: "",
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		t.Fatalf("failed to marshal GlobalConfig to YAML: %v", err)
	}

	yamlStr := string(data)

	// Verify omitempty works - default_profile should NOT appear in output when empty
	if testutil.ContainsString(yamlStr, "default_profile") {
		t.Errorf("expected YAML to NOT contain 'default_profile' when empty, got:\n%s", yamlStr)
	}
}

// Test 5: Profile name validation (valid vendor/name format)
func TestValidateProfileName_ValidFormats(t *testing.T) {
	t.Parallel()
	validNames := []string{
		"vendor/profile",
		"my-vendor/my-profile",
		"My_Vendor/My_Profile",
		"vendor123/profile456",
		"VENDOR/PROFILE",
		"a/b",
		"vendor-name_123/profile-name_456",
	}

	for _, name := range validNames {
		err := config.ValidateProfileName(name)
		if err != nil {
			t.Errorf("expected profile name '%s' to be valid, got error: %v", name, err)
		}
	}
}

// Test 6: Profile name validation (invalid formats rejected)
func TestValidateProfileName_InvalidFormats(t *testing.T) {
	t.Parallel()
	invalidNames := []string{
		"",                     // empty string
		"noslash",              // missing slash
		"/profile",             // empty vendor
		"vendor/",              // empty name
		"vendor/profile/extra", // too many slashes
		"vendor//profile",      // double slash
		"vendor name/profile",  // space in vendor
		"vendor/profile name",  // space in name
		"vendor.name/profile",  // dot in vendor
		"vendor/profile.name",  // dot in name
		"@vendor/profile",      // special char in vendor
		"vendor/@profile",      // special char in name
	}

	for _, name := range invalidNames {
		err := config.ValidateProfileName(name)
		if err == nil {
			t.Errorf("expected profile name '%s' to be invalid, but got no error", name)
		}
	}
}

// Test 7 (Gap Fill): ProjectConfig YAML serialization round-trip
func TestProjectConfig_YAMLSerialization(t *testing.T) {
	t.Parallel()
	cfg := config.ProjectConfig{
		Profiles: []string{"vendor/project-profile"},
	}

	// Serialize to YAML
	data, err := yaml.Marshal(cfg)
	if err != nil {
		t.Fatalf("failed to marshal ProjectConfig to YAML: %v", err)
	}

	yamlStr := string(data)

	// Verify profiles array is serialized
	if !testutil.ContainsString(yamlStr, "profiles:") {
		t.Errorf("expected YAML to contain 'profiles:', got:\n%s", yamlStr)
	}
	if !testutil.ContainsString(yamlStr, "vendor/project-profile") {
		t.Errorf("expected YAML to contain 'vendor/project-profile', got:\n%s", yamlStr)
	}

	// Deserialize back
	var cfg2 config.ProjectConfig
	err = yaml.Unmarshal(data, &cfg2)
	if err != nil {
		t.Fatalf("failed to unmarshal ProjectConfig from YAML: %v", err)
	}

	if len(cfg2.Profiles) != len(cfg.Profiles) || cfg2.Profiles[0] != cfg.Profiles[0] {
		t.Errorf("round-trip failed: expected Profiles %v, got %v", cfg.Profiles, cfg2.Profiles)
	}
}

// Test 8 (Gap Fill): Profile name validation boundary cases
func TestValidateProfileName_BoundaryCases(t *testing.T) {
	t.Parallel()
	// Single character segments (minimum valid case)
	singleCharCases := []string{
		"a/b", // already tested, but confirming minimum
		"1/2", // numeric single chars
		"-/-", // dash single chars
		"_/_", // underscore single chars
	}

	for _, name := range singleCharCases {
		err := config.ValidateProfileName(name)
		if err != nil {
			t.Errorf("expected profile name '%s' (boundary case) to be valid, got error: %v", name, err)
		}
	}

	// Long segments (no max length defined, but verify they work)
	longVendor := "very-long-vendor-name-with-many-characters-1234567890"
	longProfile := "very-long-profile-name-with-many-characters-1234567890"
	longName := longVendor + "/" + longProfile

	err := config.ValidateProfileName(longName)
	if err != nil {
		t.Errorf("expected long profile name to be valid, got error: %v", err)
	}

	// Unicode/non-ASCII characters should be rejected
	unicodeCases := []string{
		"vendor\u00e9/profile", // e with accent
		"vendor/profile\u00e9", // e with accent in name
	}

	for _, name := range unicodeCases {
		err := config.ValidateProfileName(name)
		if err == nil {
			t.Errorf("expected profile name '%s' with unicode to be invalid, but got no error", name)
		}
	}
}

// Task Group 1 Tests: Configuration Updates for CLI Install Command

// Test 9 (Task 1.1a): GlobalConfig with InstallPrefix field serialization/deserialization
func TestGlobalConfig_InstallPrefix_YAMLSerialization(t *testing.T) {
	t.Parallel()
	cfg := config.GlobalConfig{
		DefaultProfile: "vendor/my-profile",
		InstallPrefix:  ".claude/configs",
	}

	// Serialize to YAML
	data, err := yaml.Marshal(cfg)
	if err != nil {
		t.Fatalf("failed to marshal GlobalConfig to YAML: %v", err)
	}

	yamlStr := string(data)

	// Verify install_prefix is serialized with correct snake_case
	if !testutil.ContainsString(yamlStr, "install_prefix: .claude/configs") {
		t.Errorf("expected YAML to contain 'install_prefix: .claude/configs', got:\n%s", yamlStr)
	}

	// Deserialize back
	var cfg2 config.GlobalConfig
	err = yaml.Unmarshal(data, &cfg2)
	if err != nil {
		t.Fatalf("failed to unmarshal GlobalConfig from YAML: %v", err)
	}

	if cfg2.InstallPrefix != cfg.InstallPrefix {
		t.Errorf("round-trip failed: expected InstallPrefix '%s', got '%s'", cfg.InstallPrefix, cfg2.InstallPrefix)
	}

	// Verify omitempty works - empty InstallPrefix should NOT appear in YAML output
	cfgEmpty := config.GlobalConfig{
		DefaultProfile: "vendor/test",
		InstallPrefix:  "",
	}

	dataEmpty, err := yaml.Marshal(cfgEmpty)
	if err != nil {
		t.Fatalf("failed to marshal empty GlobalConfig to YAML: %v", err)
	}

	if testutil.ContainsString(string(dataEmpty), "install_prefix") {
		t.Errorf("expected YAML to NOT contain 'install_prefix' when empty, got:\n%s", string(dataEmpty))
	}
}

// Test 10 (Task 1.1b): ProjectConfig with optional Profiles field (omitempty validation)
func TestProjectConfig_OptionalProfiles_Validation(t *testing.T) {
	t.Parallel()
	// Create validator with custom profile_name validator
	v, err := config.NewValidator()
	if err != nil {
		t.Fatalf("failed to create validator: %v", err)
	}

	// Test 1: Empty profiles should pass validation (omitempty allows empty)
	cfgEmpty := config.ProjectConfig{
		Profiles:      []string{},
		InstallPrefix: "my-prefix",
	}

	err = v.Struct(cfgEmpty)
	if err != nil {
		t.Errorf("expected empty Profiles to pass validation with omitempty, got error: %v", err)
	}

	// Test 2: Valid profiles should pass validation
	cfgValid := config.ProjectConfig{
		Profiles:      []string{"vendor/my-profile"},
		InstallPrefix: "my-prefix",
	}

	err = v.Struct(cfgValid)
	if err != nil {
		t.Errorf("expected valid Profiles to pass validation, got error: %v", err)
	}

	// Test 3: Invalid profile format should fail validation
	cfgInvalid := config.ProjectConfig{
		Profiles:      []string{"invalid-no-slash"},
		InstallPrefix: "my-prefix",
	}

	err = v.Struct(cfgInvalid)
	if err == nil {
		t.Error("expected invalid Profiles format to fail validation, but got no error")
	}
}

// Test 11 (Task 1.1c): GlobalConfigLoader reading install_prefix from YAML
// Note: This test is in the infrastructure package test file

// Test 12 (Task 1.1d): ProjectConfig validation passes with empty profiles field
func TestProjectConfig_EmptyProfiles_ValidationPasses(t *testing.T) {
	t.Parallel()
	v, err := config.NewValidator()
	if err != nil {
		t.Fatalf("failed to create validator: %v", err)
	}

	// ProjectConfig with empty profiles (allowed after omitempty change)
	cfg := config.ProjectConfig{
		Profiles:      []string{},
		InstallPrefix: config.DefaultInstallPrefix,
	}

	err = v.Struct(cfg)
	if err != nil {
		t.Errorf("expected ProjectConfig with empty Profiles to pass validation, got error: %v", err)
	}

	// Also test with only install prefix (no profiles at all)
	cfgNoProfiles := config.ProjectConfig{
		InstallPrefix: ".custom/path",
	}

	err = v.Struct(cfgNoProfiles)
	if err != nil {
		t.Errorf("expected ProjectConfig with no Profiles to pass validation, got error: %v", err)
	}
}

// =============================================================================
// Template Variables Tests (Task Group 1)
// =============================================================================

// Test 13: GlobalConfig with Variables field YAML serialization
func TestGlobalConfig_Variables_YAMLSerialization(t *testing.T) {
	t.Parallel()
	cfg := config.GlobalConfig{
		DefaultProfile: "vendor/my-profile",
		Variables: map[string]interface{}{
			"company":     "Acme Corp",
			"environment": "production",
			"port":        8080,
		},
	}

	// Serialize to YAML
	data, err := yaml.Marshal(cfg)
	if err != nil {
		t.Fatalf("failed to marshal GlobalConfig to YAML: %v", err)
	}

	yamlStr := string(data)

	// Verify variables is serialized
	if !testutil.ContainsString(yamlStr, "variables:") {
		t.Errorf("expected YAML to contain 'variables:', got:\n%s", yamlStr)
	}
	if !testutil.ContainsString(yamlStr, "company: Acme Corp") {
		t.Errorf("expected YAML to contain 'company: Acme Corp', got:\n%s", yamlStr)
	}
	if !testutil.ContainsString(yamlStr, "environment: production") {
		t.Errorf("expected YAML to contain 'environment: production', got:\n%s", yamlStr)
	}

	// Deserialize back
	var cfg2 config.GlobalConfig
	err = yaml.Unmarshal(data, &cfg2)
	if err != nil {
		t.Fatalf("failed to unmarshal GlobalConfig from YAML: %v", err)
	}

	if cfg2.Variables["company"] != "Acme Corp" {
		t.Errorf("round-trip failed: expected company 'Acme Corp', got '%v'", cfg2.Variables["company"])
	}
	if cfg2.Variables["environment"] != "production" {
		t.Errorf("round-trip failed: expected environment 'production', got '%v'", cfg2.Variables["environment"])
	}
}

// Test 14: GlobalConfig with empty Variables map omitempty behavior
func TestGlobalConfig_Variables_OmitemptyBehavior(t *testing.T) {
	t.Parallel()
	// GlobalConfig with nil/empty Variables
	cfg := config.GlobalConfig{
		DefaultProfile: "vendor/test",
		Variables:      nil,
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		t.Fatalf("failed to marshal GlobalConfig to YAML: %v", err)
	}

	yamlStr := string(data)

	// Verify omitempty works - variables should NOT appear in output when nil
	if testutil.ContainsString(yamlStr, "variables") {
		t.Errorf("expected YAML to NOT contain 'variables' when nil, got:\n%s", yamlStr)
	}

	// Test with empty map
	cfgEmpty := config.GlobalConfig{
		DefaultProfile: "vendor/test",
		Variables:      map[string]interface{}{},
	}

	dataEmpty, err := yaml.Marshal(cfgEmpty)
	if err != nil {
		t.Fatalf("failed to marshal empty GlobalConfig to YAML: %v", err)
	}

	// Empty map should also be omitted
	if testutil.ContainsString(string(dataEmpty), "variables") {
		t.Errorf("expected YAML to NOT contain 'variables' when empty map, got:\n%s", string(dataEmpty))
	}
}

// Test 15: ProjectConfig with Variables field YAML serialization
func TestProjectConfig_Variables_YAMLSerialization(t *testing.T) {
	t.Parallel()
	cfg := config.ProjectConfig{
		Profiles:      []string{"vendor/my-profile"},
		InstallPrefix: "weftlo",
		Variables: map[string]interface{}{
			"project_name": "my-project",
			"debug":        true,
			"max_retries":  3,
		},
	}

	// Serialize to YAML
	data, err := yaml.Marshal(cfg)
	if err != nil {
		t.Fatalf("failed to marshal ProjectConfig to YAML: %v", err)
	}

	yamlStr := string(data)

	// Verify variables is serialized
	if !testutil.ContainsString(yamlStr, "variables:") {
		t.Errorf("expected YAML to contain 'variables:', got:\n%s", yamlStr)
	}
	if !testutil.ContainsString(yamlStr, "project_name: my-project") {
		t.Errorf("expected YAML to contain 'project_name: my-project', got:\n%s", yamlStr)
	}
	if !testutil.ContainsString(yamlStr, "debug: true") {
		t.Errorf("expected YAML to contain 'debug: true', got:\n%s", yamlStr)
	}

	// Deserialize back
	var cfg2 config.ProjectConfig
	err = yaml.Unmarshal(data, &cfg2)
	if err != nil {
		t.Fatalf("failed to unmarshal ProjectConfig from YAML: %v", err)
	}

	if cfg2.Variables["project_name"] != "my-project" {
		t.Errorf("round-trip failed: expected project_name 'my-project', got '%v'", cfg2.Variables["project_name"])
	}
	if cfg2.Variables["debug"] != true {
		t.Errorf("round-trip failed: expected debug true, got '%v'", cfg2.Variables["debug"])
	}
}

// Test 16: ProjectConfig with empty Variables map omitempty behavior
func TestProjectConfig_Variables_OmitemptyBehavior(t *testing.T) {
	t.Parallel()
	cfg := config.ProjectConfig{
		Profiles:  []string{"vendor/test"},
		Variables: nil,
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		t.Fatalf("failed to marshal ProjectConfig to YAML: %v", err)
	}

	yamlStr := string(data)

	// Verify omitempty works - variables should NOT appear in output when nil
	if testutil.ContainsString(yamlStr, "variables") {
		t.Errorf("expected YAML to NOT contain 'variables' when nil, got:\n%s", yamlStr)
	}
}

// Test 17: Config with nested variables map
func TestGlobalConfig_Variables_NestedMap(t *testing.T) {
	t.Parallel()
	cfg := config.GlobalConfig{
		DefaultProfile: "vendor/my-profile",
		Variables: map[string]interface{}{
			"database": map[string]interface{}{
				"host": "localhost",
				"port": 5432,
			},
			"api": map[string]interface{}{
				"base_url": "https://api.example.com",
				"timeout":  30,
			},
		},
	}

	// Serialize to YAML
	data, err := yaml.Marshal(cfg)
	if err != nil {
		t.Fatalf("failed to marshal GlobalConfig to YAML: %v", err)
	}

	// Deserialize back
	var cfg2 config.GlobalConfig
	err = yaml.Unmarshal(data, &cfg2)
	if err != nil {
		t.Fatalf("failed to unmarshal GlobalConfig from YAML: %v", err)
	}

	// Verify nested structure is preserved
	db, ok := cfg2.Variables["database"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected database to be a map, got %T", cfg2.Variables["database"])
	}
	if db["host"] != "localhost" {
		t.Errorf("expected database.host to be 'localhost', got '%v'", db["host"])
	}

	api, ok := cfg2.Variables["api"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected api to be a map, got %T", cfg2.Variables["api"])
	}
	if api["base_url"] != "https://api.example.com" {
		t.Errorf("expected api.base_url to be 'https://api.example.com', got '%v'", api["base_url"])
	}
}

// =============================================================================
// Domain Model Extensions - Task Group 3: ProjectConfig Field Extensions
// =============================================================================

// Test 18 (Task 3.1a): ProjectConfig YAML round-trip with TargetOverrides and Ignore fields
func TestProjectConfig_TargetOverridesAndIgnore_YAMLRoundTrip(t *testing.T) {
	t.Parallel()
	// Helper to create string pointer
	strPtr := func(s string) *string { return &s }

	cfg := config.ProjectConfig{
		Profiles:      []string{"vendor/my-profile"},
		InstallPrefix: "weftlo",
		Variables: map[string]interface{}{
			"project_name": "test-project",
		},
		TargetOverrides: map[string]*string{
			"skills":  strPtr("custom-skills"),
			"prompts": strPtr("custom-prompts"),
		},
		Ignore: []string{
			"*.log",
			"temp/",
			"!important.log",
		},
	}

	// Serialize to YAML
	data, err := yaml.Marshal(cfg)
	if err != nil {
		t.Fatalf("failed to marshal ProjectConfig to YAML: %v", err)
	}

	yamlStr := string(data)

	// Verify target_overrides is serialized with correct snake_case
	if !testutil.ContainsString(yamlStr, "target_overrides:") {
		t.Errorf("expected YAML to contain 'target_overrides:', got:\n%s", yamlStr)
	}
	if !testutil.ContainsString(yamlStr, "skills: custom-skills") {
		t.Errorf("expected YAML to contain 'skills: custom-skills', got:\n%s", yamlStr)
	}

	// Verify ignore is serialized
	if !testutil.ContainsString(yamlStr, "ignore:") {
		t.Errorf("expected YAML to contain 'ignore:', got:\n%s", yamlStr)
	}
	if !testutil.ContainsString(yamlStr, "*.log") {
		t.Errorf("expected YAML to contain '*.log', got:\n%s", yamlStr)
	}

	// Deserialize back
	var cfg2 config.ProjectConfig
	err = yaml.Unmarshal(data, &cfg2)
	if err != nil {
		t.Fatalf("failed to unmarshal ProjectConfig from YAML: %v", err)
	}

	// Verify TargetOverrides round-trip
	if cfg2.TargetOverrides == nil {
		t.Fatal("expected TargetOverrides to be non-nil after round-trip")
	}
	if cfg2.TargetOverrides["skills"] == nil || *cfg2.TargetOverrides["skills"] != "custom-skills" {
		t.Errorf("round-trip failed: expected skills 'custom-skills', got '%v'", cfg2.TargetOverrides["skills"])
	}
	if cfg2.TargetOverrides["prompts"] == nil || *cfg2.TargetOverrides["prompts"] != "custom-prompts" {
		t.Errorf("round-trip failed: expected prompts 'custom-prompts', got '%v'", cfg2.TargetOverrides["prompts"])
	}

	// Verify Ignore round-trip
	if len(cfg2.Ignore) != 3 {
		t.Errorf("round-trip failed: expected 3 ignore patterns, got %d", len(cfg2.Ignore))
	}
	if cfg2.Ignore[0] != "*.log" {
		t.Errorf("round-trip failed: expected first ignore pattern '*.log', got '%s'", cfg2.Ignore[0])
	}

	// Verify existing fields still work
	if len(cfg2.Profiles) != 1 || cfg2.Profiles[0] != "vendor/my-profile" {
		t.Errorf("existing Profiles field corrupted after round-trip: %v", cfg2.Profiles)
	}
	if cfg2.InstallPrefix != "weftlo" {
		t.Errorf("existing InstallPrefix field corrupted after round-trip: %s", cfg2.InstallPrefix)
	}
}

// Test 19 (Task 3.1b): TargetOverrides with nil pointer values (suppression case)
func TestProjectConfig_TargetOverrides_NilPointerSuppression(t *testing.T) {
	t.Parallel()
	// Create config with nil pointer value (suppresses target entirely)
	cfg := config.ProjectConfig{
		Profiles: []string{"vendor/my-profile"},
		TargetOverrides: map[string]*string{
			"skills": nil, // Suppress skills target entirely
		},
	}

	// Serialize to YAML
	data, err := yaml.Marshal(cfg)
	if err != nil {
		t.Fatalf("failed to marshal ProjectConfig to YAML: %v", err)
	}

	yamlStr := string(data)

	// Verify target_overrides contains the skills key with null value
	if !testutil.ContainsString(yamlStr, "target_overrides:") {
		t.Errorf("expected YAML to contain 'target_overrides:', got:\n%s", yamlStr)
	}
	// YAML should serialize nil pointer as null
	if !testutil.ContainsString(yamlStr, "skills: null") {
		t.Errorf("expected YAML to contain 'skills: null' for nil pointer, got:\n%s", yamlStr)
	}

	// Deserialize back
	var cfg2 config.ProjectConfig
	err = yaml.Unmarshal(data, &cfg2)
	if err != nil {
		t.Fatalf("failed to unmarshal ProjectConfig from YAML: %v", err)
	}

	// Verify nil pointer is preserved after round-trip
	if cfg2.TargetOverrides == nil {
		t.Fatal("expected TargetOverrides to be non-nil after round-trip")
	}

	// The key should exist with a nil value
	val, exists := cfg2.TargetOverrides["skills"]
	if !exists {
		t.Error("expected 'skills' key to exist in TargetOverrides")
	}
	if val != nil {
		t.Errorf("expected 'skills' value to be nil (suppressed), got '%v'", *val)
	}
}

// Test 20 (Task 3.1c): TargetOverrides with pointer to empty string (redirect to project root)
func TestProjectConfig_TargetOverrides_EmptyStringPointerRedirectToRoot(t *testing.T) {
	t.Parallel()
	// Helper to create string pointer
	emptyStr := ""

	cfg := config.ProjectConfig{
		Profiles: []string{"vendor/my-profile"},
		TargetOverrides: map[string]*string{
			"skills": &emptyStr, // Redirect skills to project root
		},
	}

	// Serialize to YAML
	data, err := yaml.Marshal(cfg)
	if err != nil {
		t.Fatalf("failed to marshal ProjectConfig to YAML: %v", err)
	}

	yamlStr := string(data)

	// Verify target_overrides is serialized
	if !testutil.ContainsString(yamlStr, "target_overrides:") {
		t.Errorf("expected YAML to contain 'target_overrides:', got:\n%s", yamlStr)
	}
	// Empty string pointer should serialize as empty string, not null
	if !testutil.ContainsString(yamlStr, "skills: \"\"") {
		t.Errorf("expected YAML to contain 'skills: \"\"' for empty string pointer, got:\n%s", yamlStr)
	}

	// Deserialize back
	var cfg2 config.ProjectConfig
	err = yaml.Unmarshal(data, &cfg2)
	if err != nil {
		t.Fatalf("failed to unmarshal ProjectConfig from YAML: %v", err)
	}

	// Verify empty string pointer is preserved after round-trip
	if cfg2.TargetOverrides == nil {
		t.Fatal("expected TargetOverrides to be non-nil after round-trip")
	}

	val, exists := cfg2.TargetOverrides["skills"]
	if !exists {
		t.Error("expected 'skills' key to exist in TargetOverrides")
	}
	if val == nil {
		t.Error("expected 'skills' value to be non-nil (empty string pointer), got nil")
	} else if *val != "" {
		t.Errorf("expected 'skills' value to be empty string (redirect to root), got '%s'", *val)
	}
}

// Test 21 (Task 3.1d): Omitempty behavior for TargetOverrides and Ignore fields
func TestProjectConfig_TargetOverridesAndIgnore_OmitemptyBehavior(t *testing.T) {
	t.Parallel()
	// Test with nil TargetOverrides
	cfgNilOverrides := config.ProjectConfig{
		Profiles:        []string{"vendor/test"},
		TargetOverrides: nil,
		Ignore:          nil,
	}

	data, err := yaml.Marshal(cfgNilOverrides)
	if err != nil {
		t.Fatalf("failed to marshal ProjectConfig to YAML: %v", err)
	}

	yamlStr := string(data)

	// Verify omitempty works - target_overrides should NOT appear when nil
	if testutil.ContainsString(yamlStr, "target_overrides") {
		t.Errorf("expected YAML to NOT contain 'target_overrides' when nil, got:\n%s", yamlStr)
	}

	// Verify omitempty works - ignore should NOT appear when nil
	if testutil.ContainsString(yamlStr, "ignore") {
		t.Errorf("expected YAML to NOT contain 'ignore' when nil, got:\n%s", yamlStr)
	}

	// Test with empty map/slice (should also be omitted)
	cfgEmptyOverrides := config.ProjectConfig{
		Profiles:        []string{"vendor/test"},
		TargetOverrides: map[string]*string{},
		Ignore:          []string{},
	}

	dataEmpty, err := yaml.Marshal(cfgEmptyOverrides)
	if err != nil {
		t.Fatalf("failed to marshal ProjectConfig with empty map/slice to YAML: %v", err)
	}

	yamlStrEmpty := string(dataEmpty)

	// Empty map should also be omitted
	if testutil.ContainsString(yamlStrEmpty, "target_overrides") {
		t.Errorf("expected YAML to NOT contain 'target_overrides' when empty map, got:\n%s", yamlStrEmpty)
	}

	// Empty slice should also be omitted
	if testutil.ContainsString(yamlStrEmpty, "ignore") {
		t.Errorf("expected YAML to NOT contain 'ignore' when empty slice, got:\n%s", yamlStrEmpty)
	}
}

// =============================================================================
// Domain Model Extensions - Task Group 4: Integration Tests
// =============================================================================

// Test 22 (Task 4.3): Integration test for ProjectConfig with mixed TargetOverrides semantics
// This test verifies all three TargetOverrides semantic cases work together in a single config:
// - nil pointer (suppression)
// - pointer to empty string (redirect to root)
// - pointer to non-empty string (redirect to custom path)
func TestProjectConfig_TargetOverrides_MixedSemantics_Integration(t *testing.T) {
	t.Parallel()
	// Helper to create string pointer
	strPtr := func(s string) *string { return &s }
	emptyStr := ""

	cfg := config.ProjectConfig{
		Profiles:      []string{"vendor/my-profile"},
		InstallPrefix: "weftlo",
		Variables: map[string]interface{}{
			"project": "integration-test",
		},
		TargetOverrides: map[string]*string{
			"skills":    strPtr("custom-skills-dir"), // Redirect to custom path
			"prompts":   nil,                         // Suppress entirely
			"templates": &emptyStr,                   // Redirect to project root
			"workflows": strPtr(".claude/workflows"), // Redirect to nested path
		},
		Ignore: []string{
			"*.bak",
			"*.tmp",
			"!critical.tmp",
		},
	}

	// Serialize to YAML
	data, err := yaml.Marshal(cfg)
	if err != nil {
		t.Fatalf("failed to marshal ProjectConfig to YAML: %v", err)
	}

	yamlStr := string(data)

	// Verify all TargetOverrides are serialized correctly
	if !testutil.ContainsString(yamlStr, "skills: custom-skills-dir") {
		t.Errorf("expected YAML to contain 'skills: custom-skills-dir', got:\n%s", yamlStr)
	}
	if !testutil.ContainsString(yamlStr, "prompts: null") {
		t.Errorf("expected YAML to contain 'prompts: null', got:\n%s", yamlStr)
	}
	if !testutil.ContainsString(yamlStr, "templates: \"\"") {
		t.Errorf("expected YAML to contain 'templates: \"\"', got:\n%s", yamlStr)
	}
	if !testutil.ContainsString(yamlStr, "workflows: .claude/workflows") {
		t.Errorf("expected YAML to contain 'workflows: .claude/workflows', got:\n%s", yamlStr)
	}

	// Deserialize back
	var cfg2 config.ProjectConfig
	err = yaml.Unmarshal(data, &cfg2)
	if err != nil {
		t.Fatalf("failed to unmarshal ProjectConfig from YAML: %v", err)
	}

	// Verify all semantic cases round-trip correctly
	if cfg2.TargetOverrides == nil {
		t.Fatal("expected TargetOverrides to be non-nil after round-trip")
	}

	// Case 1: Non-empty string pointer (redirect to custom path)
	skillsVal, skillsExists := cfg2.TargetOverrides["skills"]
	if !skillsExists {
		t.Error("expected 'skills' key to exist in TargetOverrides")
	}
	if skillsVal == nil || *skillsVal != "custom-skills-dir" {
		t.Errorf("expected 'skills' to be 'custom-skills-dir', got '%v'", skillsVal)
	}

	// Case 2: Nil pointer (suppression)
	promptsVal, promptsExists := cfg2.TargetOverrides["prompts"]
	if !promptsExists {
		t.Error("expected 'prompts' key to exist in TargetOverrides")
	}
	if promptsVal != nil {
		t.Errorf("expected 'prompts' to be nil (suppressed), got '%v'", *promptsVal)
	}

	// Case 3: Empty string pointer (redirect to root)
	templatesVal, templatesExists := cfg2.TargetOverrides["templates"]
	if !templatesExists {
		t.Error("expected 'templates' key to exist in TargetOverrides")
	}
	if templatesVal == nil {
		t.Error("expected 'templates' to be non-nil (empty string pointer), got nil")
	} else if *templatesVal != "" {
		t.Errorf("expected 'templates' to be empty string (redirect to root), got '%s'", *templatesVal)
	}

	// Case 4: Another non-empty string (nested path)
	workflowsVal, workflowsExists := cfg2.TargetOverrides["workflows"]
	if !workflowsExists {
		t.Error("expected 'workflows' key to exist in TargetOverrides")
	}
	if workflowsVal == nil || *workflowsVal != ".claude/workflows" {
		t.Errorf("expected 'workflows' to be '.claude/workflows', got '%v'", workflowsVal)
	}

	// Verify Ignore patterns round-trip
	if len(cfg2.Ignore) != 3 {
		t.Errorf("expected 3 ignore patterns, got %d", len(cfg2.Ignore))
	}

	// Verify existing fields still work
	if cfg2.InstallPrefix != "weftlo" {
		t.Errorf("expected InstallPrefix 'weftlo', got '%s'", cfg2.InstallPrefix)
	}
	if cfg2.Variables["project"] != "integration-test" {
		t.Errorf("expected Variables[project] 'integration-test', got '%v'", cfg2.Variables["project"])
	}
}
