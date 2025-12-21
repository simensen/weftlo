package config_test

import (
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/simensen/weftlo/internal/domain/config"
	"github.com/simensen/weftlo/internal/testutil"
)

// =============================================================================
// Task Group 1: GlobalConfig Domain Model Enhancement Tests
// =============================================================================

// Test 1: GlobalConfig TargetOverrides field with nil, empty, and populated values
func TestGlobalConfig_TargetOverrides_FieldVariants(t *testing.T) {
	t.Parallel()
	strPtr := func(s string) *string { return &s }

	// Test with nil TargetOverrides
	cfgNil := config.GlobalConfig{
		DefaultProfile:  "vendor/profile",
		TargetOverrides: nil,
	}

	if cfgNil.TargetOverrides != nil {
		t.Errorf("expected TargetOverrides to be nil, got %v", cfgNil.TargetOverrides)
	}

	// Serialize and verify omitempty behavior
	dataNil, err := yaml.Marshal(cfgNil)
	if err != nil {
		t.Fatalf("failed to marshal GlobalConfig with nil TargetOverrides: %v", err)
	}
	if testutil.ContainsString(string(dataNil), "target_overrides") {
		t.Errorf("expected YAML to NOT contain 'target_overrides' when nil, got:\n%s", string(dataNil))
	}

	// Test with empty map
	cfgEmpty := config.GlobalConfig{
		DefaultProfile:  "vendor/profile",
		TargetOverrides: map[string]*string{},
	}

	dataEmpty, err := yaml.Marshal(cfgEmpty)
	if err != nil {
		t.Fatalf("failed to marshal GlobalConfig with empty TargetOverrides: %v", err)
	}
	if testutil.ContainsString(string(dataEmpty), "target_overrides") {
		t.Errorf("expected YAML to NOT contain 'target_overrides' when empty map, got:\n%s", string(dataEmpty))
	}

	// Test with populated values
	cfgPopulated := config.GlobalConfig{
		DefaultProfile: "vendor/profile",
		TargetOverrides: map[string]*string{
			"agents":   strPtr(".claude/agents"),
			"commands": strPtr(".claude/commands"),
		},
	}

	dataPopulated, err := yaml.Marshal(cfgPopulated)
	if err != nil {
		t.Fatalf("failed to marshal GlobalConfig with populated TargetOverrides: %v", err)
	}
	yamlStr := string(dataPopulated)
	if !testutil.ContainsString(yamlStr, "target_overrides:") {
		t.Errorf("expected YAML to contain 'target_overrides:', got:\n%s", yamlStr)
	}
	if !testutil.ContainsString(yamlStr, "agents: .claude/agents") {
		t.Errorf("expected YAML to contain 'agents: .claude/agents', got:\n%s", yamlStr)
	}

	// Round-trip verification
	var cfg2 config.GlobalConfig
	err = yaml.Unmarshal(dataPopulated, &cfg2)
	if err != nil {
		t.Fatalf("failed to unmarshal GlobalConfig: %v", err)
	}
	if cfg2.TargetOverrides == nil {
		t.Fatal("expected TargetOverrides to be non-nil after round-trip")
	}
	if cfg2.TargetOverrides["agents"] == nil || *cfg2.TargetOverrides["agents"] != ".claude/agents" {
		t.Errorf("round-trip failed: expected agents '.claude/agents', got '%v'", cfg2.TargetOverrides["agents"])
	}
}

// Test 2: GlobalConfig ProfileOverrides field with nested map structure
func TestGlobalConfig_ProfileOverrides_NestedStructure(t *testing.T) {
	t.Parallel()
	strPtr := func(s string) *string { return &s }

	// Test with nil ProfileOverrides
	cfgNil := config.GlobalConfig{
		DefaultProfile:   "vendor/profile",
		ProfileOverrides: nil,
	}

	dataNil, err := yaml.Marshal(cfgNil)
	if err != nil {
		t.Fatalf("failed to marshal GlobalConfig with nil ProfileOverrides: %v", err)
	}
	if testutil.ContainsString(string(dataNil), "profile_overrides") {
		t.Errorf("expected YAML to NOT contain 'profile_overrides' when nil, got:\n%s", string(dataNil))
	}

	// Test with populated ProfileOverrides
	cfgPopulated := config.GlobalConfig{
		DefaultProfile: "vendor/profile",
		ProfileOverrides: map[string]config.ProfileOverrideConfig{
			"simensen/git-profile": {
				TargetOverrides: map[string]*string{
					"agents": strPtr(".claude/agents/simensen"),
				},
			},
			"other-vendor/other-profile": {
				TargetOverrides: map[string]*string{
					"standards": strPtr(".project-standards"),
				},
			},
		},
	}

	dataPopulated, err := yaml.Marshal(cfgPopulated)
	if err != nil {
		t.Fatalf("failed to marshal GlobalConfig with ProfileOverrides: %v", err)
	}
	yamlStr := string(dataPopulated)
	if !testutil.ContainsString(yamlStr, "profile_overrides:") {
		t.Errorf("expected YAML to contain 'profile_overrides:', got:\n%s", yamlStr)
	}

	// Round-trip verification
	var cfg2 config.GlobalConfig
	err = yaml.Unmarshal(dataPopulated, &cfg2)
	if err != nil {
		t.Fatalf("failed to unmarshal GlobalConfig: %v", err)
	}
	if cfg2.ProfileOverrides == nil {
		t.Fatal("expected ProfileOverrides to be non-nil after round-trip")
	}

	simensenProfile, exists := cfg2.ProfileOverrides["simensen/git-profile"]
	if !exists {
		t.Error("expected 'simensen/git-profile' key to exist in ProfileOverrides")
	}
	if simensenProfile.TargetOverrides == nil {
		t.Error("expected TargetOverrides in simensen/git-profile to be non-nil")
	}
	if simensenProfile.TargetOverrides["agents"] == nil || *simensenProfile.TargetOverrides["agents"] != ".claude/agents/simensen" {
		t.Errorf("expected agents '.claude/agents/simensen', got '%v'", simensenProfile.TargetOverrides["agents"])
	}
}

// Test 3: GlobalConfig Ignore field with empty and populated patterns
func TestGlobalConfig_Ignore_FieldVariants(t *testing.T) {
	t.Parallel()
	// Test with nil Ignore
	cfgNil := config.GlobalConfig{
		DefaultProfile: "vendor/profile",
		Ignore:         nil,
	}

	dataNil, err := yaml.Marshal(cfgNil)
	if err != nil {
		t.Fatalf("failed to marshal GlobalConfig with nil Ignore: %v", err)
	}
	if testutil.ContainsString(string(dataNil), "ignore") {
		t.Errorf("expected YAML to NOT contain 'ignore' when nil, got:\n%s", string(dataNil))
	}

	// Test with empty slice
	cfgEmpty := config.GlobalConfig{
		DefaultProfile: "vendor/profile",
		Ignore:         []string{},
	}

	dataEmpty, err := yaml.Marshal(cfgEmpty)
	if err != nil {
		t.Fatalf("failed to marshal GlobalConfig with empty Ignore: %v", err)
	}
	if testutil.ContainsString(string(dataEmpty), "ignore") {
		t.Errorf("expected YAML to NOT contain 'ignore' when empty slice, got:\n%s", string(dataEmpty))
	}

	// Test with populated patterns
	cfgPopulated := config.GlobalConfig{
		DefaultProfile: "vendor/profile",
		Ignore: []string{
			"**/*.bak",
			"**/temp/**",
			"!important.bak",
		},
	}

	dataPopulated, err := yaml.Marshal(cfgPopulated)
	if err != nil {
		t.Fatalf("failed to marshal GlobalConfig with Ignore patterns: %v", err)
	}
	yamlStr := string(dataPopulated)
	if !testutil.ContainsString(yamlStr, "ignore:") {
		t.Errorf("expected YAML to contain 'ignore:', got:\n%s", yamlStr)
	}
	if !testutil.ContainsString(yamlStr, "**/*.bak") {
		t.Errorf("expected YAML to contain '**/*.bak', got:\n%s", yamlStr)
	}

	// Round-trip verification
	var cfg2 config.GlobalConfig
	err = yaml.Unmarshal(dataPopulated, &cfg2)
	if err != nil {
		t.Fatalf("failed to unmarshal GlobalConfig: %v", err)
	}
	if len(cfg2.Ignore) != 3 {
		t.Errorf("expected 3 ignore patterns, got %d", len(cfg2.Ignore))
	}
	if cfg2.Ignore[0] != "**/*.bak" {
		t.Errorf("expected first pattern '**/*.bak', got '%s'", cfg2.Ignore[0])
	}
}

// Test 4: GlobalConfig pointer semantics for suppression (nil vs pointer-to-empty vs pointer-to-string)
func TestGlobalConfig_TargetOverrides_PointerSemantics(t *testing.T) {
	t.Parallel()
	strPtr := func(s string) *string { return &s }
	emptyStr := ""

	cfg := config.GlobalConfig{
		DefaultProfile: "vendor/profile",
		TargetOverrides: map[string]*string{
			"cursor":    nil,                      // nil = suppress target entirely
			"templates": &emptyStr,                // empty string = redirect to root
			"agents":    strPtr(".claude/agents"), // non-empty = redirect to path
		},
	}

	// Serialize to YAML
	data, err := yaml.Marshal(cfg)
	if err != nil {
		t.Fatalf("failed to marshal GlobalConfig: %v", err)
	}
	yamlStr := string(data)

	// Verify nil pointer serializes as null
	if !testutil.ContainsString(yamlStr, "cursor: null") {
		t.Errorf("expected YAML to contain 'cursor: null' for nil pointer (suppression), got:\n%s", yamlStr)
	}

	// Verify empty string pointer serializes as empty string
	if !testutil.ContainsString(yamlStr, "templates: \"\"") {
		t.Errorf("expected YAML to contain 'templates: \"\"' for empty string pointer, got:\n%s", yamlStr)
	}

	// Verify non-empty string pointer serializes as value
	if !testutil.ContainsString(yamlStr, "agents: .claude/agents") {
		t.Errorf("expected YAML to contain 'agents: .claude/agents', got:\n%s", yamlStr)
	}

	// Round-trip verification
	var cfg2 config.GlobalConfig
	err = yaml.Unmarshal(data, &cfg2)
	if err != nil {
		t.Fatalf("failed to unmarshal GlobalConfig: %v", err)
	}

	// Verify nil pointer (suppression) is preserved
	cursorVal, cursorExists := cfg2.TargetOverrides["cursor"]
	if !cursorExists {
		t.Error("expected 'cursor' key to exist in TargetOverrides")
	}
	if cursorVal != nil {
		t.Errorf("expected 'cursor' to be nil (suppressed), got '%v'", *cursorVal)
	}

	// Verify empty string pointer (redirect to root) is preserved
	templatesVal, templatesExists := cfg2.TargetOverrides["templates"]
	if !templatesExists {
		t.Error("expected 'templates' key to exist in TargetOverrides")
	}
	if templatesVal == nil {
		t.Error("expected 'templates' to be non-nil (empty string pointer), got nil")
	} else if *templatesVal != "" {
		t.Errorf("expected 'templates' to be empty string (redirect to root), got '%s'", *templatesVal)
	}

	// Verify non-empty string pointer (redirect to path) is preserved
	agentsVal, agentsExists := cfg2.TargetOverrides["agents"]
	if !agentsExists {
		t.Error("expected 'agents' key to exist in TargetOverrides")
	}
	if agentsVal == nil || *agentsVal != ".claude/agents" {
		t.Errorf("expected 'agents' to be '.claude/agents', got '%v'", agentsVal)
	}
}

// Test 5: GlobalConfig full configuration with all new fields
func TestGlobalConfig_AllNewFields_Integration(t *testing.T) {
	t.Parallel()
	strPtr := func(s string) *string { return &s }

	cfg := config.GlobalConfig{
		DefaultProfile: "simensen/base-profile",
		InstallPrefix:  ".weftlo",
		Variables: map[string]interface{}{
			"namespace": "my-namespace",
		},
		TargetOverrides: map[string]*string{
			"agents":   strPtr(".claude/agents/{{ .Variables.namespace }}"),
			"commands": strPtr(".claude/commands/{{ .Variables.namespace }}"),
			"cursor":   nil, // Suppress cursor content globally
		},
		ProfileOverrides: map[string]config.ProfileOverrideConfig{
			"simensen/git-profile": {
				TargetOverrides: map[string]*string{
					"agents": strPtr(".claude/agents/simensen"),
				},
			},
		},
		Ignore: []string{
			"**/*.bak",
			"**/temp/**",
		},
	}

	// Serialize to YAML
	data, err := yaml.Marshal(cfg)
	if err != nil {
		t.Fatalf("failed to marshal GlobalConfig: %v", err)
	}

	// Round-trip verification
	var cfg2 config.GlobalConfig
	err = yaml.Unmarshal(data, &cfg2)
	if err != nil {
		t.Fatalf("failed to unmarshal GlobalConfig: %v", err)
	}

	// Verify all fields round-trip correctly
	if cfg2.DefaultProfile != "simensen/base-profile" {
		t.Errorf("expected DefaultProfile 'simensen/base-profile', got '%s'", cfg2.DefaultProfile)
	}
	if cfg2.InstallPrefix != ".weftlo" {
		t.Errorf("expected InstallPrefix '.weftlo', got '%s'", cfg2.InstallPrefix)
	}
	if cfg2.Variables["namespace"] != "my-namespace" {
		t.Errorf("expected Variables[namespace] 'my-namespace', got '%v'", cfg2.Variables["namespace"])
	}
	if cfg2.TargetOverrides == nil || len(cfg2.TargetOverrides) != 3 {
		t.Errorf("expected 3 TargetOverrides, got %d", len(cfg2.TargetOverrides))
	}
	if cfg2.ProfileOverrides == nil || len(cfg2.ProfileOverrides) != 1 {
		t.Errorf("expected 1 ProfileOverrides, got %d", len(cfg2.ProfileOverrides))
	}
	if len(cfg2.Ignore) != 2 {
		t.Errorf("expected 2 Ignore patterns, got %d", len(cfg2.Ignore))
	}
}

// Test 6: ProfileOverrideConfig struct YAML serialization
func TestProfileOverrideConfig_YAMLSerialization(t *testing.T) {
	t.Parallel()
	strPtr := func(s string) *string { return &s }

	poc := config.ProfileOverrideConfig{
		TargetOverrides: map[string]*string{
			"agents":  strPtr(".custom/agents"),
			"prompts": nil, // Suppress
		},
	}

	// Serialize to YAML
	data, err := yaml.Marshal(poc)
	if err != nil {
		t.Fatalf("failed to marshal ProfileOverrideConfig: %v", err)
	}
	yamlStr := string(data)

	if !testutil.ContainsString(yamlStr, "target_overrides:") {
		t.Errorf("expected YAML to contain 'target_overrides:', got:\n%s", yamlStr)
	}
	if !testutil.ContainsString(yamlStr, "agents: .custom/agents") {
		t.Errorf("expected YAML to contain 'agents: .custom/agents', got:\n%s", yamlStr)
	}
	if !testutil.ContainsString(yamlStr, "prompts: null") {
		t.Errorf("expected YAML to contain 'prompts: null', got:\n%s", yamlStr)
	}

	// Round-trip verification
	var poc2 config.ProfileOverrideConfig
	err = yaml.Unmarshal(data, &poc2)
	if err != nil {
		t.Fatalf("failed to unmarshal ProfileOverrideConfig: %v", err)
	}
	if poc2.TargetOverrides["agents"] == nil || *poc2.TargetOverrides["agents"] != ".custom/agents" {
		t.Errorf("round-trip failed for agents")
	}
	if poc2.TargetOverrides["prompts"] != nil {
		t.Error("round-trip failed: expected prompts to be nil (suppressed)")
	}
}
