package manifest

import (
	"encoding/json"
	"testing"
	"time"
)

// Test Manifest struct instantiation and field access
func TestManifestInstantiation(t *testing.T) {
	t.Parallel()
	generatedAt := time.Date(2024, 12, 20, 10, 30, 0, 0, time.UTC)
	files := map[string]ManifestFile{
		"weftlo/standards/api.md": {
			SourceChecksum: "abc123",
			OutputChecksum: "def456",
			SourceProfile:  "default",
		},
	}

	manifest := Manifest{
		Version:     "1.0.0",
		Profiles:    []string{"my-profile"},
		GeneratedAt: generatedAt,
		Files:       files,
	}

	if manifest.Version != "1.0.0" {
		t.Errorf("expected Version '1.0.0', got '%s'", manifest.Version)
	}
	if len(manifest.Profiles) != 1 || manifest.Profiles[0] != "my-profile" {
		t.Errorf("expected Profiles ['my-profile'], got %v", manifest.Profiles)
	}
	if !manifest.GeneratedAt.Equal(generatedAt) {
		t.Errorf("expected GeneratedAt '%v', got '%v'", generatedAt, manifest.GeneratedAt)
	}
	if len(manifest.Files) != 1 {
		t.Errorf("expected 1 file, got %d", len(manifest.Files))
	}
}

// Test ManifestFile struct field population
func TestManifestFileFieldPopulation(t *testing.T) {
	t.Parallel()
	manifestFile := ManifestFile{
		SourceChecksum: "sha256-source-hash",
		OutputChecksum: "sha256-output-hash",
		SourceProfile:  "custom-profile",
	}

	if manifestFile.SourceChecksum != "sha256-source-hash" {
		t.Errorf("expected SourceChecksum 'sha256-source-hash', got '%s'", manifestFile.SourceChecksum)
	}
	if manifestFile.OutputChecksum != "sha256-output-hash" {
		t.Errorf("expected OutputChecksum 'sha256-output-hash', got '%s'", manifestFile.OutputChecksum)
	}
	if manifestFile.SourceProfile != "custom-profile" {
		t.Errorf("expected SourceProfile 'custom-profile', got '%s'", manifestFile.SourceProfile)
	}
}

// Test JSON serialization/deserialization for Manifest (snake_case tags)
func TestManifestJSONSerialization(t *testing.T) {
	t.Parallel()
	generatedAt := time.Date(2024, 12, 20, 10, 30, 0, 0, time.UTC)
	manifest := Manifest{
		Version:     "1.0.0",
		Profiles:    []string{"default"},
		GeneratedAt: generatedAt,
		Files: map[string]ManifestFile{
			"weftlo/standards/api.md": {
				SourceChecksum: "abc123",
				OutputChecksum: "def456",
				SourceProfile:  "default",
			},
		},
	}

	// Serialize to JSON
	data, err := json.Marshal(manifest)
	if err != nil {
		t.Fatalf("failed to marshal Manifest: %v", err)
	}

	// Verify snake_case keys are present
	jsonStr := string(data)
	if !contains(jsonStr, `"version"`) {
		t.Error("expected JSON to contain 'version' key")
	}
	if !contains(jsonStr, `"profiles"`) {
		t.Error("expected JSON to contain 'profiles' key")
	}
	if !contains(jsonStr, `"generated_at"`) {
		t.Error("expected JSON to contain 'generated_at' key")
	}
	if !contains(jsonStr, `"files"`) {
		t.Error("expected JSON to contain 'files' key")
	}

	// Deserialize back
	var unmarshaled Manifest
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("failed to unmarshal Manifest: %v", err)
	}

	if unmarshaled.Version != manifest.Version {
		t.Errorf("expected Version '%s', got '%s'", manifest.Version, unmarshaled.Version)
	}
	if len(unmarshaled.Profiles) != len(manifest.Profiles) || unmarshaled.Profiles[0] != manifest.Profiles[0] {
		t.Errorf("expected Profiles %v, got %v", manifest.Profiles, unmarshaled.Profiles)
	}
	if !unmarshaled.GeneratedAt.Equal(manifest.GeneratedAt) {
		t.Errorf("expected GeneratedAt '%v', got '%v'", manifest.GeneratedAt, unmarshaled.GeneratedAt)
	}
}

// Test JSON serialization/deserialization for ManifestFile
func TestManifestFileJSONSerialization(t *testing.T) {
	t.Parallel()
	manifestFile := ManifestFile{
		SourceChecksum: "abc123",
		OutputChecksum: "def456",
		SourceProfile:  "my-profile",
	}

	// Serialize to JSON
	data, err := json.Marshal(manifestFile)
	if err != nil {
		t.Fatalf("failed to marshal ManifestFile: %v", err)
	}

	// Verify snake_case keys are present
	jsonStr := string(data)
	if !contains(jsonStr, `"source_checksum"`) {
		t.Error("expected JSON to contain 'source_checksum' key")
	}
	if !contains(jsonStr, `"output_checksum"`) {
		t.Error("expected JSON to contain 'output_checksum' key")
	}
	if !contains(jsonStr, `"source_profile"`) {
		t.Error("expected JSON to contain 'source_profile' key")
	}

	// Deserialize back
	var unmarshaled ManifestFile
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("failed to unmarshal ManifestFile: %v", err)
	}

	if unmarshaled.SourceChecksum != manifestFile.SourceChecksum {
		t.Errorf("expected SourceChecksum '%s', got '%s'", manifestFile.SourceChecksum, unmarshaled.SourceChecksum)
	}
	if unmarshaled.OutputChecksum != manifestFile.OutputChecksum {
		t.Errorf("expected OutputChecksum '%s', got '%s'", manifestFile.OutputChecksum, unmarshaled.OutputChecksum)
	}
	if unmarshaled.SourceProfile != manifestFile.SourceProfile {
		t.Errorf("expected SourceProfile '%s', got '%s'", manifestFile.SourceProfile, unmarshaled.SourceProfile)
	}
}

// Test Files map keyed by target path
func TestManifestFilesMapKeyedByTargetPath(t *testing.T) {
	t.Parallel()
	manifest := Manifest{
		Version:     "1.0.0",
		Profiles:    []string{"default"},
		GeneratedAt: time.Now(),
		Files: map[string]ManifestFile{
			"weftlo/standards/api.md": {
				SourceChecksum: "hash1",
				OutputChecksum: "hash2",
				SourceProfile:  "default",
			},
			"weftlo/commands/build.md": {
				SourceChecksum: "hash3",
				OutputChecksum: "hash4",
				SourceProfile:  "custom",
			},
		},
	}

	// Test that we can access files by target path
	apiFile, exists := manifest.Files["weftlo/standards/api.md"]
	if !exists {
		t.Error("expected file at 'weftlo/standards/api.md' to exist")
	}
	if apiFile.SourceProfile != "default" {
		t.Errorf("expected SourceProfile 'default', got '%s'", apiFile.SourceProfile)
	}

	buildFile, exists := manifest.Files["weftlo/commands/build.md"]
	if !exists {
		t.Error("expected file at 'weftlo/commands/build.md' to exist")
	}
	if buildFile.SourceProfile != "custom" {
		t.Errorf("expected SourceProfile 'custom', got '%s'", buildFile.SourceProfile)
	}

	// Test that a non-existent key returns zero value
	_, exists = manifest.Files["nonexistent/path.md"]
	if exists {
		t.Error("expected file at 'nonexistent/path.md' to not exist")
	}
}

// Test 6: Manifest with zero time.Time value (gap coverage for time.Time edge case)
func TestManifestZeroTimeValue(t *testing.T) {
	t.Parallel()
	// Manifest with zero time value
	manifest := Manifest{
		Version:     "1.0.0",
		Profiles:    []string{"default"},
		GeneratedAt: time.Time{}, // Zero time
		Files:       map[string]ManifestFile{},
	}

	// Serialize to JSON
	data, err := json.Marshal(manifest)
	if err != nil {
		t.Fatalf("failed to marshal Manifest with zero time: %v", err)
	}

	// Verify it serializes (Go's time.Time zero value serializes to a specific ISO8601 string)
	jsonStr := string(data)
	if !contains(jsonStr, `"generated_at"`) {
		t.Error("expected JSON to contain 'generated_at' key even with zero time")
	}

	// Deserialize back
	var unmarshaled Manifest
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("failed to unmarshal Manifest with zero time: %v", err)
	}

	// Verify round-trip preserves zero time
	if !unmarshaled.GeneratedAt.IsZero() {
		t.Errorf("expected GeneratedAt to be zero time after round-trip, got '%v'", unmarshaled.GeneratedAt)
	}

	// Test with actual time for comparison
	actualTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	manifest2 := Manifest{
		Version:     "1.0.0",
		Profiles:    []string{"default"},
		GeneratedAt: actualTime,
		Files:       map[string]ManifestFile{},
	}

	data2, err := json.Marshal(manifest2)
	if err != nil {
		t.Fatalf("failed to marshal Manifest with actual time: %v", err)
	}

	var unmarshaled2 Manifest
	if err := json.Unmarshal(data2, &unmarshaled2); err != nil {
		t.Fatalf("failed to unmarshal Manifest with actual time: %v", err)
	}

	if !unmarshaled2.GeneratedAt.Equal(actualTime) {
		t.Errorf("expected GeneratedAt '%v', got '%v'", actualTime, unmarshaled2.GeneratedAt)
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
