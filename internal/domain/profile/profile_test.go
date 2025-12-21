package profile_test

import (
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/simensen/weftlo/internal/domain/profile"
	"github.com/simensen/weftlo/internal/testutil"
)

// Test 1: ProfileConfig struct instantiation and field access
func TestProfileConfig_Instantiation(t *testing.T) {
	t.Parallel()
	cfg := profile.ProfileConfig{
		Name:         "my-profile",
		InheritsFrom: "default",
	}

	if cfg.Name != "my-profile" {
		t.Errorf("expected Name to be 'my-profile', got '%s'", cfg.Name)
	}
	if cfg.InheritsFrom != "default" {
		t.Errorf("expected InheritsFrom to be 'default', got '%s'", cfg.InheritsFrom)
	}
}

// Test 2: Profile struct instantiation with TemplateFiles slice
func TestProfile_InstantiationWithTemplateFiles(t *testing.T) {
	t.Parallel()
	templateFiles := []profile.TemplateFile{
		{SourcePath: "standards/api.md", TargetPath: "weftlo/standards/api.md", IsPartial: false},
		{SourcePath: "_partials/header.md", TargetPath: "", IsPartial: true},
	}

	p := profile.Profile{
		Name:          "my-profile",
		InheritsFrom:  "default",
		TemplateFiles: templateFiles,
	}

	if p.Name != "my-profile" {
		t.Errorf("expected Name to be 'my-profile', got '%s'", p.Name)
	}
	if p.InheritsFrom != "default" {
		t.Errorf("expected InheritsFrom to be 'default', got '%s'", p.InheritsFrom)
	}
	if len(p.TemplateFiles) != 2 {
		t.Errorf("expected 2 template files, got %d", len(p.TemplateFiles))
	}
}

// Test 3: TemplateFile struct field population
func TestTemplateFile_FieldPopulation(t *testing.T) {
	t.Parallel()
	tf := profile.TemplateFile{
		SourcePath: "standards/backend/api.md",
		TargetPath: "weftlo/standards/backend/api.md",
		IsPartial:  false,
		Checksum:   "abc123def456",
	}

	if tf.SourcePath != "standards/backend/api.md" {
		t.Errorf("expected SourcePath to be 'standards/backend/api.md', got '%s'", tf.SourcePath)
	}
	if tf.TargetPath != "weftlo/standards/backend/api.md" {
		t.Errorf("expected TargetPath to be 'weftlo/standards/backend/api.md', got '%s'", tf.TargetPath)
	}
	if tf.IsPartial != false {
		t.Errorf("expected IsPartial to be false, got %v", tf.IsPartial)
	}
	if tf.Checksum != "abc123def456" {
		t.Errorf("expected Checksum to be 'abc123def456', got '%s'", tf.Checksum)
	}
}

// Test 4: YAML serialization/deserialization for ProfileConfig (snake_case tags)
func TestProfileConfig_YAMLSerialization(t *testing.T) {
	t.Parallel()
	cfg := profile.ProfileConfig{
		Name:         "my-profile",
		InheritsFrom: "default",
	}

	// Serialize to YAML
	data, err := yaml.Marshal(cfg)
	if err != nil {
		t.Fatalf("failed to marshal ProfileConfig to YAML: %v", err)
	}

	yamlStr := string(data)

	// Verify snake_case tags are used
	if !testutil.ContainsString(yamlStr, "name: my-profile") {
		t.Errorf("expected YAML to contain 'name: my-profile', got:\n%s", yamlStr)
	}
	if !testutil.ContainsString(yamlStr, "inherits_from: default") {
		t.Errorf("expected YAML to contain 'inherits_from: default', got:\n%s", yamlStr)
	}

	// Deserialize back
	var cfg2 profile.ProfileConfig
	err = yaml.Unmarshal(data, &cfg2)
	if err != nil {
		t.Fatalf("failed to unmarshal ProfileConfig from YAML: %v", err)
	}

	if cfg2.Name != cfg.Name {
		t.Errorf("round-trip failed: expected Name '%s', got '%s'", cfg.Name, cfg2.Name)
	}
	if cfg2.InheritsFrom != cfg.InheritsFrom {
		t.Errorf("round-trip failed: expected InheritsFrom '%s', got '%s'", cfg.InheritsFrom, cfg2.InheritsFrom)
	}
}

// Test 5: YAML serialization/deserialization for Profile
func TestProfile_YAMLSerialization(t *testing.T) {
	t.Parallel()
	p := profile.Profile{
		Name:         "my-profile",
		InheritsFrom: "default",
		TemplateFiles: []profile.TemplateFile{
			{SourcePath: "standards/api.md", TargetPath: "weftlo/standards/api.md", IsPartial: false, Checksum: "abc123"},
		},
	}

	// Serialize to YAML
	data, err := yaml.Marshal(p)
	if err != nil {
		t.Fatalf("failed to marshal Profile to YAML: %v", err)
	}

	yamlStr := string(data)

	// Verify snake_case tags are used
	if !testutil.ContainsString(yamlStr, "name: my-profile") {
		t.Errorf("expected YAML to contain 'name: my-profile', got:\n%s", yamlStr)
	}
	if !testutil.ContainsString(yamlStr, "template_files:") {
		t.Errorf("expected YAML to contain 'template_files:', got:\n%s", yamlStr)
	}
	if !testutil.ContainsString(yamlStr, "source_path:") {
		t.Errorf("expected YAML to contain 'source_path:', got:\n%s", yamlStr)
	}
	if !testutil.ContainsString(yamlStr, "target_path:") {
		t.Errorf("expected YAML to contain 'target_path:', got:\n%s", yamlStr)
	}
	if !testutil.ContainsString(yamlStr, "is_partial:") {
		t.Errorf("expected YAML to contain 'is_partial:', got:\n%s", yamlStr)
	}

	// Deserialize back
	var p2 profile.Profile
	err = yaml.Unmarshal(data, &p2)
	if err != nil {
		t.Fatalf("failed to unmarshal Profile from YAML: %v", err)
	}

	if p2.Name != p.Name {
		t.Errorf("round-trip failed: expected Name '%s', got '%s'", p.Name, p2.Name)
	}
	if len(p2.TemplateFiles) != len(p.TemplateFiles) {
		t.Errorf("round-trip failed: expected %d template files, got %d", len(p.TemplateFiles), len(p2.TemplateFiles))
	}
}

// Test 6: Omitempty behavior for optional fields in ProfileConfig
func TestProfileConfig_OmitemptyBehavior(t *testing.T) {
	t.Parallel()
	// ProfileConfig without InheritsFrom (empty string)
	cfg := profile.ProfileConfig{
		Name:         "standalone-profile",
		InheritsFrom: "",
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		t.Fatalf("failed to marshal ProfileConfig to YAML: %v", err)
	}

	yamlStr := string(data)

	// Verify omitempty works - inherits_from should NOT appear in output
	if testutil.ContainsString(yamlStr, "inherits_from") {
		t.Errorf("expected YAML to NOT contain 'inherits_from' when empty, got:\n%s", yamlStr)
	}
}

// Test 7: Omitempty behavior for TemplateFiles in Profile (gap coverage)
func TestProfile_OmitemptyTemplateFiles(t *testing.T) {
	t.Parallel()
	// Profile without TemplateFiles (nil slice)
	p := profile.Profile{
		Name:          "empty-profile",
		InheritsFrom:  "",
		TemplateFiles: nil,
	}

	data, err := yaml.Marshal(p)
	if err != nil {
		t.Fatalf("failed to marshal Profile to YAML: %v", err)
	}

	yamlStr := string(data)

	// Verify omitempty works - template_files should NOT appear in output when nil
	if testutil.ContainsString(yamlStr, "template_files") {
		t.Errorf("expected YAML to NOT contain 'template_files' when nil, got:\n%s", yamlStr)
	}

	// Test with empty slice as well
	p2 := profile.Profile{
		Name:          "empty-profile",
		TemplateFiles: []profile.TemplateFile{},
	}

	data2, err := yaml.Marshal(p2)
	if err != nil {
		t.Fatalf("failed to marshal Profile to YAML: %v", err)
	}

	yamlStr2 := string(data2)

	// Verify omitempty works - template_files should NOT appear in output when empty slice
	if testutil.ContainsString(yamlStr2, "template_files") {
		t.Errorf("expected YAML to NOT contain 'template_files' when empty slice, got:\n%s", yamlStr2)
	}
}

// Test 8: Omitempty behavior for Checksum in TemplateFile (gap coverage)
func TestTemplateFile_OmitemptyChecksum(t *testing.T) {
	t.Parallel()
	// TemplateFile without Checksum (empty string)
	tf := profile.TemplateFile{
		SourcePath: "standards/api.md",
		TargetPath: "weftlo/standards/api.md",
		IsPartial:  false,
		Checksum:   "",
	}

	data, err := yaml.Marshal(tf)
	if err != nil {
		t.Fatalf("failed to marshal TemplateFile to YAML: %v", err)
	}

	yamlStr := string(data)

	// Verify omitempty works - checksum should NOT appear in output when empty
	if testutil.ContainsString(yamlStr, "checksum") {
		t.Errorf("expected YAML to NOT contain 'checksum' when empty, got:\n%s", yamlStr)
	}

	// Verify other fields are still present
	if !testutil.ContainsString(yamlStr, "source_path:") {
		t.Errorf("expected YAML to contain 'source_path:', got:\n%s", yamlStr)
	}
	if !testutil.ContainsString(yamlStr, "target_path:") {
		t.Errorf("expected YAML to contain 'target_path:', got:\n%s", yamlStr)
	}
	if !testutil.ContainsString(yamlStr, "is_partial:") {
		t.Errorf("expected YAML to contain 'is_partial:', got:\n%s", yamlStr)
	}
}

// =============================================================================
// Template Variables Tests (Task Group 1)
// =============================================================================

// Test 9: ProfileConfig with Variables field YAML serialization
func TestProfileConfig_Variables_YAMLSerialization(t *testing.T) {
	t.Parallel()
	cfg := profile.ProfileConfig{
		Name:         "my-profile",
		InheritsFrom: "default",
		Variables: map[string]interface{}{
			"author":  "John Doe",
			"version": "1.0.0",
			"enabled": true,
		},
	}

	// Serialize to YAML
	data, err := yaml.Marshal(cfg)
	if err != nil {
		t.Fatalf("failed to marshal ProfileConfig to YAML: %v", err)
	}

	yamlStr := string(data)

	// Verify variables is serialized
	if !testutil.ContainsString(yamlStr, "variables:") {
		t.Errorf("expected YAML to contain 'variables:', got:\n%s", yamlStr)
	}
	if !testutil.ContainsString(yamlStr, "author: John Doe") {
		t.Errorf("expected YAML to contain 'author: John Doe', got:\n%s", yamlStr)
	}
	if !testutil.ContainsString(yamlStr, "version: 1.0.0") {
		t.Errorf("expected YAML to contain 'version: 1.0.0', got:\n%s", yamlStr)
	}

	// Deserialize back
	var cfg2 profile.ProfileConfig
	err = yaml.Unmarshal(data, &cfg2)
	if err != nil {
		t.Fatalf("failed to unmarshal ProfileConfig from YAML: %v", err)
	}

	if cfg2.Variables["author"] != "John Doe" {
		t.Errorf("round-trip failed: expected author 'John Doe', got '%v'", cfg2.Variables["author"])
	}
	if cfg2.Variables["version"] != "1.0.0" {
		t.Errorf("round-trip failed: expected version '1.0.0', got '%v'", cfg2.Variables["version"])
	}
	if cfg2.Variables["enabled"] != true {
		t.Errorf("round-trip failed: expected enabled true, got '%v'", cfg2.Variables["enabled"])
	}
}

// Test 10: ProfileConfig with empty Variables map omitempty behavior
func TestProfileConfig_Variables_OmitemptyBehavior(t *testing.T) {
	t.Parallel()
	// ProfileConfig with nil/empty Variables
	cfg := profile.ProfileConfig{
		Name:      "test-profile",
		Variables: nil,
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		t.Fatalf("failed to marshal ProfileConfig to YAML: %v", err)
	}

	yamlStr := string(data)

	// Verify omitempty works - variables should NOT appear in output when nil
	if testutil.ContainsString(yamlStr, "variables") {
		t.Errorf("expected YAML to NOT contain 'variables' when nil, got:\n%s", yamlStr)
	}

	// Test with empty map
	cfgEmpty := profile.ProfileConfig{
		Name:      "test-profile",
		Variables: map[string]interface{}{},
	}

	dataEmpty, err := yaml.Marshal(cfgEmpty)
	if err != nil {
		t.Fatalf("failed to marshal empty ProfileConfig to YAML: %v", err)
	}

	// Empty map should also be omitted
	if testutil.ContainsString(string(dataEmpty), "variables") {
		t.Errorf("expected YAML to NOT contain 'variables' when empty map, got:\n%s", string(dataEmpty))
	}
}

// Test 11: ProfileConfig with nested Variables map
func TestProfileConfig_Variables_NestedMap(t *testing.T) {
	t.Parallel()
	cfg := profile.ProfileConfig{
		Name: "my-profile",
		Variables: map[string]interface{}{
			"settings": map[string]interface{}{
				"theme":    "dark",
				"language": "en",
			},
			"features": map[string]interface{}{
				"logging": true,
				"metrics": false,
			},
		},
	}

	// Serialize to YAML
	data, err := yaml.Marshal(cfg)
	if err != nil {
		t.Fatalf("failed to marshal ProfileConfig to YAML: %v", err)
	}

	// Deserialize back
	var cfg2 profile.ProfileConfig
	err = yaml.Unmarshal(data, &cfg2)
	if err != nil {
		t.Fatalf("failed to unmarshal ProfileConfig from YAML: %v", err)
	}

	// Verify nested structure is preserved
	settings, ok := cfg2.Variables["settings"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected settings to be a map, got %T", cfg2.Variables["settings"])
	}
	if settings["theme"] != "dark" {
		t.Errorf("expected settings.theme to be 'dark', got '%v'", settings["theme"])
	}

	features, ok := cfg2.Variables["features"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected features to be a map, got %T", cfg2.Variables["features"])
	}
	if features["logging"] != true {
		t.Errorf("expected features.logging to be true, got '%v'", features["logging"])
	}
}
