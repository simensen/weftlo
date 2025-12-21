package profile_test

import (
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/simensen/weftlo/internal/domain/profile"
	"github.com/simensen/weftlo/internal/testutil"
)

// =============================================================================
// ContentConfig Tests (Task Group 1 - Domain Model Extensions)
// =============================================================================

// Test 1: ContentConfig YAML unmarshaling with all fields populated
func TestContentConfig_YAMLUnmarshal_AllFieldsPopulated(t *testing.T) {
	yamlInput := `
root: custom-content
targets:
  skills: .claude/skills
  workflows: .claude/workflows
  standards: weftlo/standards
default_target: weftlo
`

	var cfg profile.ContentConfig
	err := yaml.Unmarshal([]byte(yamlInput), &cfg)
	if err != nil {
		t.Fatalf("failed to unmarshal ContentConfig from YAML: %v", err)
	}

	// Verify Root field
	if cfg.Root != "custom-content" {
		t.Errorf("expected Root to be 'custom-content', got '%s'", cfg.Root)
	}

	// Verify Targets map
	if cfg.Targets == nil {
		t.Fatal("expected Targets to be non-nil")
	}
	if len(cfg.Targets) != 3 {
		t.Errorf("expected 3 targets, got %d", len(cfg.Targets))
	}
	if cfg.Targets["skills"] != ".claude/skills" {
		t.Errorf("expected Targets[skills] to be '.claude/skills', got '%s'", cfg.Targets["skills"])
	}
	if cfg.Targets["workflows"] != ".claude/workflows" {
		t.Errorf("expected Targets[workflows] to be '.claude/workflows', got '%s'", cfg.Targets["workflows"])
	}
	if cfg.Targets["standards"] != "weftlo/standards" {
		t.Errorf("expected Targets[standards] to be 'weftlo/standards', got '%s'", cfg.Targets["standards"])
	}

	// Verify DefaultTarget field
	if cfg.DefaultTarget != "weftlo" {
		t.Errorf("expected DefaultTarget to be 'weftlo', got '%s'", cfg.DefaultTarget)
	}
}

// Test 2: ContentConfig YAML unmarshaling with empty/missing fields produces zero values
func TestContentConfig_YAMLUnmarshal_EmptyFieldsProduceZeroValues(t *testing.T) {
	// Test with completely empty YAML
	yamlInput := `{}`

	var cfg profile.ContentConfig
	err := yaml.Unmarshal([]byte(yamlInput), &cfg)
	if err != nil {
		t.Fatalf("failed to unmarshal empty ContentConfig from YAML: %v", err)
	}

	// Verify all fields are zero values
	if cfg.Root != "" {
		t.Errorf("expected Root to be empty string, got '%s'", cfg.Root)
	}
	if cfg.Targets != nil {
		t.Errorf("expected Targets to be nil, got %v", cfg.Targets)
	}
	if cfg.DefaultTarget != "" {
		t.Errorf("expected DefaultTarget to be empty string, got '%s'", cfg.DefaultTarget)
	}

	// Test with partial fields (only root set)
	yamlPartial := `root: content-only`

	var cfgPartial profile.ContentConfig
	err = yaml.Unmarshal([]byte(yamlPartial), &cfgPartial)
	if err != nil {
		t.Fatalf("failed to unmarshal partial ContentConfig from YAML: %v", err)
	}

	if cfgPartial.Root != "content-only" {
		t.Errorf("expected Root to be 'content-only', got '%s'", cfgPartial.Root)
	}
	if cfgPartial.Targets != nil {
		t.Errorf("expected Targets to be nil when not specified, got %v", cfgPartial.Targets)
	}
	if cfgPartial.DefaultTarget != "" {
		t.Errorf("expected DefaultTarget to be empty when not specified, got '%s'", cfgPartial.DefaultTarget)
	}
}

// Test 3: ContentConfig YAML marshaling produces expected output with snake_case tags
func TestContentConfig_YAMLMarshal_ProducesSnakeCaseTags(t *testing.T) {
	cfg := profile.ContentConfig{
		Root: "my-content",
		Targets: map[string]string{
			"skills": ".claude/skills",
		},
		DefaultTarget: "weftlo",
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		t.Fatalf("failed to marshal ContentConfig to YAML: %v", err)
	}

	yamlStr := string(data)

	// Verify snake_case tag for root
	if !testutil.ContainsString(yamlStr, "root: my-content") {
		t.Errorf("expected YAML to contain 'root: my-content', got:\n%s", yamlStr)
	}

	// Verify snake_case tag for targets
	if !testutil.ContainsString(yamlStr, "targets:") {
		t.Errorf("expected YAML to contain 'targets:', got:\n%s", yamlStr)
	}

	// Verify snake_case tag for default_target
	if !testutil.ContainsString(yamlStr, "default_target: weftlo") {
		t.Errorf("expected YAML to contain 'default_target: weftlo', got:\n%s", yamlStr)
	}

	// Verify round-trip works
	var cfg2 profile.ContentConfig
	err = yaml.Unmarshal(data, &cfg2)
	if err != nil {
		t.Fatalf("failed to unmarshal ContentConfig from YAML: %v", err)
	}

	if cfg2.Root != cfg.Root {
		t.Errorf("round-trip failed: expected Root '%s', got '%s'", cfg.Root, cfg2.Root)
	}
	if cfg2.DefaultTarget != cfg.DefaultTarget {
		t.Errorf("round-trip failed: expected DefaultTarget '%s', got '%s'", cfg.DefaultTarget, cfg2.DefaultTarget)
	}
	if cfg2.Targets["skills"] != cfg.Targets["skills"] {
		t.Errorf("round-trip failed: expected Targets[skills] '%s', got '%s'", cfg.Targets["skills"], cfg2.Targets["skills"])
	}
}

// Test 4: Empty ContentConfig struct marshals to empty YAML (omitempty behavior)
func TestContentConfig_YAMLMarshal_EmptyStructProducesEmptyYAML(t *testing.T) {
	// Test with zero-value struct
	cfg := profile.ContentConfig{}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		t.Fatalf("failed to marshal empty ContentConfig to YAML: %v", err)
	}

	yamlStr := string(data)

	// Verify omitempty works - no fields should appear in output
	if testutil.ContainsString(yamlStr, "root") {
		t.Errorf("expected YAML to NOT contain 'root' when empty, got:\n%s", yamlStr)
	}
	if testutil.ContainsString(yamlStr, "targets") {
		t.Errorf("expected YAML to NOT contain 'targets' when empty, got:\n%s", yamlStr)
	}
	if testutil.ContainsString(yamlStr, "default_target") {
		t.Errorf("expected YAML to NOT contain 'default_target' when empty, got:\n%s", yamlStr)
	}

	// Test with empty map (not nil) - should also be omitted
	cfgEmptyMap := profile.ContentConfig{
		Targets: map[string]string{},
	}

	dataEmptyMap, err := yaml.Marshal(cfgEmptyMap)
	if err != nil {
		t.Fatalf("failed to marshal ContentConfig with empty map to YAML: %v", err)
	}

	yamlStrEmptyMap := string(dataEmptyMap)

	// Empty map should be omitted due to omitempty
	if testutil.ContainsString(yamlStrEmptyMap, "targets") {
		t.Errorf("expected YAML to NOT contain 'targets' when empty map, got:\n%s", yamlStrEmptyMap)
	}
}

// =============================================================================
// ProfileConfig Extensions Tests (Task Group 2 - Domain Model Extensions)
// =============================================================================

// Test 5: ProfileConfig YAML round-trip with Content and Ignore fields populated
func TestProfileConfig_ContentAndIgnore_YAMLRoundTrip(t *testing.T) {
	cfg := profile.ProfileConfig{
		Name:         "my-profile",
		InheritsFrom: "base-profile",
		Variables: map[string]interface{}{
			"company": "Acme Corp",
		},
		Content: profile.ContentConfig{
			Root: "templates",
			Targets: map[string]string{
				"skills":    ".claude/skills",
				"workflows": ".claude/workflows",
			},
			DefaultTarget: "weftlo",
		},
		Ignore: []string{
			"*.tmp",
			"*.bak",
			"node_modules/",
			"!important.tmp",
		},
	}

	// Serialize to YAML
	data, err := yaml.Marshal(cfg)
	if err != nil {
		t.Fatalf("failed to marshal ProfileConfig to YAML: %v", err)
	}

	yamlStr := string(data)

	// Verify content section is serialized
	if !testutil.ContainsString(yamlStr, "content:") {
		t.Errorf("expected YAML to contain 'content:', got:\n%s", yamlStr)
	}
	if !testutil.ContainsString(yamlStr, "root: templates") {
		t.Errorf("expected YAML to contain 'root: templates', got:\n%s", yamlStr)
	}
	if !testutil.ContainsString(yamlStr, "targets:") {
		t.Errorf("expected YAML to contain 'targets:', got:\n%s", yamlStr)
	}
	if !testutil.ContainsString(yamlStr, "default_target: weftlo") {
		t.Errorf("expected YAML to contain 'default_target: weftlo', got:\n%s", yamlStr)
	}

	// Verify ignore section is serialized
	if !testutil.ContainsString(yamlStr, "ignore:") {
		t.Errorf("expected YAML to contain 'ignore:', got:\n%s", yamlStr)
	}
	if !testutil.ContainsString(yamlStr, "*.tmp") {
		t.Errorf("expected YAML to contain '*.tmp', got:\n%s", yamlStr)
	}
	if !testutil.ContainsString(yamlStr, "node_modules/") {
		t.Errorf("expected YAML to contain 'node_modules/', got:\n%s", yamlStr)
	}

	// Deserialize back
	var cfg2 profile.ProfileConfig
	err = yaml.Unmarshal(data, &cfg2)
	if err != nil {
		t.Fatalf("failed to unmarshal ProfileConfig from YAML: %v", err)
	}

	// Verify Content fields round-trip correctly
	if cfg2.Content.Root != cfg.Content.Root {
		t.Errorf("round-trip failed: expected Content.Root '%s', got '%s'", cfg.Content.Root, cfg2.Content.Root)
	}
	if cfg2.Content.DefaultTarget != cfg.Content.DefaultTarget {
		t.Errorf("round-trip failed: expected Content.DefaultTarget '%s', got '%s'", cfg.Content.DefaultTarget, cfg2.Content.DefaultTarget)
	}
	if len(cfg2.Content.Targets) != len(cfg.Content.Targets) {
		t.Errorf("round-trip failed: expected %d targets, got %d", len(cfg.Content.Targets), len(cfg2.Content.Targets))
	}
	if cfg2.Content.Targets["skills"] != cfg.Content.Targets["skills"] {
		t.Errorf("round-trip failed: expected Targets[skills] '%s', got '%s'", cfg.Content.Targets["skills"], cfg2.Content.Targets["skills"])
	}

	// Verify Ignore field round-trips correctly
	if len(cfg2.Ignore) != len(cfg.Ignore) {
		t.Errorf("round-trip failed: expected %d ignore patterns, got %d", len(cfg.Ignore), len(cfg2.Ignore))
	}
	for i, pattern := range cfg.Ignore {
		if cfg2.Ignore[i] != pattern {
			t.Errorf("round-trip failed: expected Ignore[%d] '%s', got '%s'", i, pattern, cfg2.Ignore[i])
		}
	}

	// Verify existing fields still work
	if cfg2.Name != cfg.Name {
		t.Errorf("round-trip failed: expected Name '%s', got '%s'", cfg.Name, cfg2.Name)
	}
	if cfg2.InheritsFrom != cfg.InheritsFrom {
		t.Errorf("round-trip failed: expected InheritsFrom '%s', got '%s'", cfg.InheritsFrom, cfg2.InheritsFrom)
	}
	if cfg2.Variables["company"] != cfg.Variables["company"] {
		t.Errorf("round-trip failed: expected Variables[company] '%v', got '%v'", cfg.Variables["company"], cfg2.Variables["company"])
	}
}

// Test 6: ProfileConfig omitempty behavior when Content is zero-value struct
func TestProfileConfig_Content_OmitemptyBehavior(t *testing.T) {
	// ProfileConfig with zero-value Content struct
	cfg := profile.ProfileConfig{
		Name:    "test-profile",
		Content: profile.ContentConfig{}, // zero-value struct
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		t.Fatalf("failed to marshal ProfileConfig to YAML: %v", err)
	}

	yamlStr := string(data)

	// Verify omitempty works - content should NOT appear in output when zero-value
	if testutil.ContainsString(yamlStr, "content") {
		t.Errorf("expected YAML to NOT contain 'content' when zero-value struct, got:\n%s", yamlStr)
	}

	// Verify the ProfileConfig with partial Content (only some fields set)
	cfgPartial := profile.ProfileConfig{
		Name: "test-profile",
		Content: profile.ContentConfig{
			Root: "custom-root",
			// Targets and DefaultTarget are empty
		},
	}

	dataPartial, err := yaml.Marshal(cfgPartial)
	if err != nil {
		t.Fatalf("failed to marshal partial ProfileConfig to YAML: %v", err)
	}

	yamlStrPartial := string(dataPartial)

	// Content should appear since Root is set
	if !testutil.ContainsString(yamlStrPartial, "content:") {
		t.Errorf("expected YAML to contain 'content:' when Root is set, got:\n%s", yamlStrPartial)
	}
	if !testutil.ContainsString(yamlStrPartial, "root: custom-root") {
		t.Errorf("expected YAML to contain 'root: custom-root', got:\n%s", yamlStrPartial)
	}
	// But targets and default_target should be omitted
	if testutil.ContainsString(yamlStrPartial, "targets") {
		t.Errorf("expected YAML to NOT contain 'targets' when nil, got:\n%s", yamlStrPartial)
	}
	if testutil.ContainsString(yamlStrPartial, "default_target") {
		t.Errorf("expected YAML to NOT contain 'default_target' when empty, got:\n%s", yamlStrPartial)
	}
}

// Test 7: ProfileConfig omitempty behavior when Ignore is nil or empty slice
func TestProfileConfig_Ignore_OmitemptyBehavior(t *testing.T) {
	// ProfileConfig with nil Ignore
	cfgNil := profile.ProfileConfig{
		Name:   "test-profile",
		Ignore: nil,
	}

	dataNil, err := yaml.Marshal(cfgNil)
	if err != nil {
		t.Fatalf("failed to marshal ProfileConfig to YAML: %v", err)
	}

	yamlStrNil := string(dataNil)

	// Verify omitempty works - ignore should NOT appear in output when nil
	if testutil.ContainsString(yamlStrNil, "ignore") {
		t.Errorf("expected YAML to NOT contain 'ignore' when nil, got:\n%s", yamlStrNil)
	}

	// ProfileConfig with empty slice
	cfgEmpty := profile.ProfileConfig{
		Name:   "test-profile",
		Ignore: []string{},
	}

	dataEmpty, err := yaml.Marshal(cfgEmpty)
	if err != nil {
		t.Fatalf("failed to marshal ProfileConfig to YAML: %v", err)
	}

	yamlStrEmpty := string(dataEmpty)

	// Verify omitempty works - ignore should NOT appear in output when empty slice
	if testutil.ContainsString(yamlStrEmpty, "ignore") {
		t.Errorf("expected YAML to NOT contain 'ignore' when empty slice, got:\n%s", yamlStrEmpty)
	}
}

// Test 8: ProfileConfig existing fields (Name, InheritsFrom, Variables) continue to work
func TestProfileConfig_ExistingFields_ContinueToWork(t *testing.T) {
	// Create ProfileConfig with all existing fields populated
	cfg := profile.ProfileConfig{
		Name:         "full-profile",
		InheritsFrom: "parent-profile",
		Variables: map[string]interface{}{
			"key1": "value1",
			"key2": 42,
			"key3": true,
			"nested": map[string]interface{}{
				"inner": "data",
			},
		},
	}

	// Serialize to YAML
	data, err := yaml.Marshal(cfg)
	if err != nil {
		t.Fatalf("failed to marshal ProfileConfig to YAML: %v", err)
	}

	yamlStr := string(data)

	// Verify existing fields are serialized correctly
	if !testutil.ContainsString(yamlStr, "name: full-profile") {
		t.Errorf("expected YAML to contain 'name: full-profile', got:\n%s", yamlStr)
	}
	if !testutil.ContainsString(yamlStr, "inherits_from: parent-profile") {
		t.Errorf("expected YAML to contain 'inherits_from: parent-profile', got:\n%s", yamlStr)
	}
	if !testutil.ContainsString(yamlStr, "variables:") {
		t.Errorf("expected YAML to contain 'variables:', got:\n%s", yamlStr)
	}
	if !testutil.ContainsString(yamlStr, "key1: value1") {
		t.Errorf("expected YAML to contain 'key1: value1', got:\n%s", yamlStr)
	}

	// Deserialize back
	var cfg2 profile.ProfileConfig
	err = yaml.Unmarshal(data, &cfg2)
	if err != nil {
		t.Fatalf("failed to unmarshal ProfileConfig from YAML: %v", err)
	}

	// Verify all existing fields round-trip correctly
	if cfg2.Name != cfg.Name {
		t.Errorf("round-trip failed: expected Name '%s', got '%s'", cfg.Name, cfg2.Name)
	}
	if cfg2.InheritsFrom != cfg.InheritsFrom {
		t.Errorf("round-trip failed: expected InheritsFrom '%s', got '%s'", cfg.InheritsFrom, cfg2.InheritsFrom)
	}
	if cfg2.Variables["key1"] != cfg.Variables["key1"] {
		t.Errorf("round-trip failed: expected Variables[key1] '%v', got '%v'", cfg.Variables["key1"], cfg2.Variables["key1"])
	}
	if cfg2.Variables["key2"] != cfg.Variables["key2"] {
		t.Errorf("round-trip failed: expected Variables[key2] '%v', got '%v'", cfg.Variables["key2"], cfg2.Variables["key2"])
	}
	if cfg2.Variables["key3"] != cfg.Variables["key3"] {
		t.Errorf("round-trip failed: expected Variables[key3] '%v', got '%v'", cfg.Variables["key3"], cfg2.Variables["key3"])
	}

	// Verify nested map in Variables
	nested, ok := cfg2.Variables["nested"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected nested to be a map, got %T", cfg2.Variables["nested"])
	}
	if nested["inner"] != "data" {
		t.Errorf("round-trip failed: expected nested.inner 'data', got '%v'", nested["inner"])
	}

	// Verify new fields are zero-values (not set in this test)
	if cfg2.Content.Root != "" {
		t.Errorf("expected Content.Root to be empty, got '%s'", cfg2.Content.Root)
	}
	if cfg2.Ignore != nil {
		t.Errorf("expected Ignore to be nil, got %v", cfg2.Ignore)
	}
}
