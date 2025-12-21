package manifest

import (
	"encoding/json"
	"testing"
)

// =============================================================================
// Task Group 1.1: TargetCategory Field Handling Tests (2-4 tests)
// =============================================================================

// Test 1.1a: ManifestFile JSON serialization with TargetCategory field
// Verifies that TargetCategory is included in JSON output with correct snake_case key
func TestManifestFileJSONSerializationWithTargetCategory(t *testing.T) {
	t.Parallel()
	manifestFile := ManifestFile{
		SourceChecksum: "sha256:abc123",
		OutputChecksum: "sha256:def456",
		SourceProfile:  "vendor/my-profile",
		TargetCategory: "weftlo",
	}

	// Serialize to JSON
	data, err := json.Marshal(manifestFile)
	if err != nil {
		t.Fatalf("failed to marshal ManifestFile: %v", err)
	}

	// Verify snake_case key is present
	jsonStr := string(data)
	if !containsSubstring(jsonStr, `"target_category"`) {
		t.Error("expected JSON to contain 'target_category' key")
	}
	if !containsSubstring(jsonStr, `"weftlo"`) {
		t.Error("expected JSON to contain 'weftlo' value for target_category")
	}

	// Deserialize back and verify round-trip
	var unmarshaled ManifestFile
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("failed to unmarshal ManifestFile: %v", err)
	}

	if unmarshaled.TargetCategory != manifestFile.TargetCategory {
		t.Errorf("expected TargetCategory '%s', got '%s'",
			manifestFile.TargetCategory, unmarshaled.TargetCategory)
	}
}

// Test 1.1b: RenderedFile struct accepts and stores TargetCategory
// Verifies that RenderedFile can hold TargetCategory field correctly
func TestRenderedFileAcceptsTargetCategory(t *testing.T) {
	t.Parallel()
	rf := RenderedFile{
		TargetPath:     "weftlo/standards/api.md",
		SourceContent:  []byte("source template"),
		OutputContent:  []byte("rendered output"),
		SourceProfile:  "vendor/base-profile",
		TargetCategory: "weftlo",
	}

	if rf.TargetCategory != "weftlo" {
		t.Errorf("expected TargetCategory 'weftlo', got '%s'", rf.TargetCategory)
	}

	// Test with empty TargetCategory (valid case)
	rfEmpty := RenderedFile{
		TargetPath:     "some/path.md",
		SourceContent:  []byte("source"),
		OutputContent:  []byte("output"),
		SourceProfile:  "vendor/profile",
		TargetCategory: "",
	}

	if rfEmpty.TargetCategory != "" {
		t.Errorf("expected empty TargetCategory, got '%s'", rfEmpty.TargetCategory)
	}
}

// Test 1.1c: ManifestGenerator copies TargetCategory from RenderedFile to ManifestFile
// Verifies that Generate() correctly transfers TargetCategory field
func TestManifestGeneratorCopiesTargetCategory(t *testing.T) {
	t.Parallel()
	generator := NewManifestGenerator()

	files := []RenderedFile{
		{
			TargetPath:     "weftlo/standards/api.md",
			SourceContent:  []byte("api source"),
			OutputContent:  []byte("api output"),
			SourceProfile:  "vendor/base",
			TargetCategory: "weftlo",
		},
		{
			TargetPath:     ".claude/settings.json",
			SourceContent:  []byte("claude source"),
			OutputContent:  []byte("claude output"),
			SourceProfile:  "vendor/extended",
			TargetCategory: "claude",
		},
	}

	manifest, err := generator.Generate([]string{"vendor/extended"}, files)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify first file's TargetCategory
	apiFile, exists := manifest.Files["weftlo/standards/api.md"]
	if !exists {
		t.Fatal("expected api.md file to exist")
	}
	if apiFile.TargetCategory != "weftlo" {
		t.Errorf("expected TargetCategory 'weftlo', got '%s'", apiFile.TargetCategory)
	}

	// Verify second file's TargetCategory
	claudeFile, exists := manifest.Files[".claude/settings.json"]
	if !exists {
		t.Fatal("expected settings.json file to exist")
	}
	if claudeFile.TargetCategory != "claude" {
		t.Errorf("expected TargetCategory 'claude', got '%s'", claudeFile.TargetCategory)
	}
}

// Test 1.1d: ManifestFile with TargetCategory in full manifest JSON serialization
// Verifies that TargetCategory is preserved in complete manifest round-trip
func TestManifestWithTargetCategoryJSONRoundTrip(t *testing.T) {
	t.Parallel()
	generator := NewManifestGenerator()

	files := []RenderedFile{
		{
			TargetPath:     "weftlo/file.md",
			SourceContent:  []byte("source"),
			OutputContent:  []byte("output"),
			SourceProfile:  "vendor/profile",
			TargetCategory: "weftlo",
		},
	}

	manifest, err := generator.Generate([]string{"vendor/profile"}, files)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Serialize to JSON
	jsonData, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		t.Fatalf("failed to marshal manifest: %v", err)
	}

	// Verify JSON contains target_category
	jsonStr := string(jsonData)
	if !containsSubstring(jsonStr, `"target_category"`) {
		t.Error("expected manifest JSON to contain 'target_category' key")
	}

	// Deserialize back
	var unmarshaled Manifest
	if err := json.Unmarshal(jsonData, &unmarshaled); err != nil {
		t.Fatalf("failed to unmarshal manifest: %v", err)
	}

	// Verify round-trip preserves TargetCategory
	file, exists := unmarshaled.Files["weftlo/file.md"]
	if !exists {
		t.Fatal("expected file to exist after round-trip")
	}
	if file.TargetCategory != "weftlo" {
		t.Errorf("expected TargetCategory 'weftlo' after round-trip, got '%s'",
			file.TargetCategory)
	}
}

// Helper function to check if a string contains a substring
func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
