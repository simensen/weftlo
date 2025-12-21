package manifest

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

// =============================================================================
// Task Group 1: RenderedFile Struct and Checksum Utility Tests (4-6 tests)
// =============================================================================

// Test 1.1a: RenderedFile struct instantiation with all fields populated
func TestRenderedFileInstantiation(t *testing.T) {
	t.Parallel()
	sourceContent := []byte("{{ .ProjectName }}")
	outputContent := []byte("my-project")

	rf := RenderedFile{
		TargetPath:    "weftlo/standards/api.md",
		SourceContent: sourceContent,
		OutputContent: outputContent,
		SourceProfile: "vendor/base-profile",
	}

	if rf.TargetPath != "weftlo/standards/api.md" {
		t.Errorf("expected TargetPath 'weftlo/standards/api.md', got '%s'", rf.TargetPath)
	}
	if string(rf.SourceContent) != "{{ .ProjectName }}" {
		t.Errorf("expected SourceContent '{{ .ProjectName }}', got '%s'", string(rf.SourceContent))
	}
	if string(rf.OutputContent) != "my-project" {
		t.Errorf("expected OutputContent 'my-project', got '%s'", string(rf.OutputContent))
	}
	if rf.SourceProfile != "vendor/base-profile" {
		t.Errorf("expected SourceProfile 'vendor/base-profile', got '%s'", rf.SourceProfile)
	}
}

// Test 1.1b: Checksum computation produces correct SHA-256 format with "sha256:" prefix
func TestComputeChecksumFormat(t *testing.T) {
	t.Parallel()
	content := []byte("test content")
	checksum := ComputeChecksum(content)

	if !strings.HasPrefix(checksum, "sha256:") {
		t.Errorf("expected checksum to start with 'sha256:', got '%s'", checksum)
	}

	// SHA-256 produces 64 hex characters
	hexPart := strings.TrimPrefix(checksum, "sha256:")
	if len(hexPart) != 64 {
		t.Errorf("expected 64 hex characters after prefix, got %d", len(hexPart))
	}
}

// Test 1.1c: Checksum produces consistent results for same input
func TestComputeChecksumConsistency(t *testing.T) {
	t.Parallel()
	content := []byte("identical content")

	checksum1 := ComputeChecksum(content)
	checksum2 := ComputeChecksum(content)

	if checksum1 != checksum2 {
		t.Errorf("expected consistent checksums, got '%s' and '%s'", checksum1, checksum2)
	}
}

// Test 1.1d: Checksum produces different results for different inputs
func TestComputeChecksumDifferentInputs(t *testing.T) {
	t.Parallel()
	content1 := []byte("first content")
	content2 := []byte("second content")

	checksum1 := ComputeChecksum(content1)
	checksum2 := ComputeChecksum(content2)

	if checksum1 == checksum2 {
		t.Errorf("expected different checksums for different inputs, both got '%s'", checksum1)
	}
}

// Test 1.1e: Empty content produces valid checksum
func TestComputeChecksumEmptyContent(t *testing.T) {
	t.Parallel()
	checksum := ComputeChecksum([]byte{})

	if !strings.HasPrefix(checksum, "sha256:") {
		t.Errorf("expected checksum to start with 'sha256:' for empty content, got '%s'", checksum)
	}

	hexPart := strings.TrimPrefix(checksum, "sha256:")
	if len(hexPart) != 64 {
		t.Errorf("expected 64 hex characters for empty content checksum, got %d", len(hexPart))
	}

	// SHA-256 of empty string is a known value
	expectedEmpty := "sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	if checksum != expectedEmpty {
		t.Errorf("expected empty content checksum '%s', got '%s'", expectedEmpty, checksum)
	}
}

// Test 1.1f: Checksum format is lowercase hex-encoded
func TestComputeChecksumLowercaseHex(t *testing.T) {
	t.Parallel()
	content := []byte("test")
	checksum := ComputeChecksum(content)

	hexPart := strings.TrimPrefix(checksum, "sha256:")

	// Check that all characters are lowercase hex
	for _, c := range hexPart {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
			t.Errorf("expected lowercase hex character, got '%c' in checksum '%s'", c, checksum)
		}
	}
}

// =============================================================================
// Task Group 2: ManifestGenerator Tests (5-8 tests)
// =============================================================================

// Test 2.1a: Generating manifest from single RenderedFile
func TestGenerateSingleFile(t *testing.T) {
	t.Parallel()
	generator := NewManifestGenerator()

	files := []RenderedFile{
		{
			TargetPath:    "weftlo/standards/api.md",
			SourceContent: []byte("source template"),
			OutputContent: []byte("rendered output"),
			SourceProfile: "vendor/profile",
		},
	}

	manifest, err := generator.Generate([]string{"vendor/my-profile"}, files)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(manifest.Files) != 1 {
		t.Errorf("expected 1 file, got %d", len(manifest.Files))
	}

	file, exists := manifest.Files["weftlo/standards/api.md"]
	if !exists {
		t.Error("expected file at 'weftlo/standards/api.md' to exist")
	}
	if file.SourceProfile != "vendor/profile" {
		t.Errorf("expected SourceProfile 'vendor/profile', got '%s'", file.SourceProfile)
	}
}

// Test 2.1b: Generating manifest from multiple RenderedFiles
func TestGenerateMultipleFiles(t *testing.T) {
	t.Parallel()
	generator := NewManifestGenerator()

	files := []RenderedFile{
		{
			TargetPath:    "weftlo/standards/api.md",
			SourceContent: []byte("api template"),
			OutputContent: []byte("api output"),
			SourceProfile: "vendor/base",
		},
		{
			TargetPath:    "weftlo/commands/build.md",
			SourceContent: []byte("build template"),
			OutputContent: []byte("build output"),
			SourceProfile: "vendor/extended",
		},
		{
			TargetPath:    "CLAUDE.md",
			SourceContent: []byte("claude template"),
			OutputContent: []byte("claude output"),
			SourceProfile: "vendor/base",
		},
	}

	manifest, err := generator.Generate([]string{"vendor/my-profile"}, files)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(manifest.Files) != 3 {
		t.Errorf("expected 3 files, got %d", len(manifest.Files))
	}
}

// Test 2.1c: Manifest Version is set to "1.0.0"
func TestGenerateManifestVersion(t *testing.T) {
	t.Parallel()
	generator := NewManifestGenerator()

	files := []RenderedFile{
		{
			TargetPath:    "test.md",
			SourceContent: []byte("source"),
			OutputContent: []byte("output"),
			SourceProfile: "vendor/profile",
		},
	}

	manifest, err := generator.Generate([]string{"vendor/profile"}, files)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if manifest.Version != "1.0.0" {
		t.Errorf("expected Version '1.0.0', got '%s'", manifest.Version)
	}
}

// Test 2.1d: Manifest Profile is set to the provided profile name
func TestGenerateManifestProfile(t *testing.T) {
	t.Parallel()
	generator := NewManifestGenerator()

	files := []RenderedFile{
		{
			TargetPath:    "test.md",
			SourceContent: []byte("source"),
			OutputContent: []byte("output"),
			SourceProfile: "vendor/base",
		},
	}

	manifest, err := generator.Generate([]string{"vendor/leaf-profile"}, files)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if manifest.Profiles[0] != "vendor/leaf-profile" {
		t.Errorf("expected Profile 'vendor/leaf-profile', got '%s'", manifest.Profiles[0])
	}
}

// Test 2.1e: Manifest GeneratedAt is set to completion timestamp
func TestGenerateManifestGeneratedAt(t *testing.T) {
	t.Parallel()
	generator := NewManifestGenerator()

	files := []RenderedFile{
		{
			TargetPath:    "test.md",
			SourceContent: []byte("source"),
			OutputContent: []byte("output"),
			SourceProfile: "vendor/profile",
		},
	}

	beforeGenerate := time.Now()
	manifest, err := generator.Generate([]string{"vendor/profile"}, files)
	afterGenerate := time.Now()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if manifest.GeneratedAt.Before(beforeGenerate) {
		t.Errorf("expected GeneratedAt to be after %v, got %v", beforeGenerate, manifest.GeneratedAt)
	}
	if manifest.GeneratedAt.After(afterGenerate) {
		t.Errorf("expected GeneratedAt to be before %v, got %v", afterGenerate, manifest.GeneratedAt)
	}
}

// Test 2.1f: Files map is keyed by TargetPath
func TestGenerateFilesKeyedByTargetPath(t *testing.T) {
	t.Parallel()
	generator := NewManifestGenerator()

	files := []RenderedFile{
		{
			TargetPath:    "path/to/first.md",
			SourceContent: []byte("first source"),
			OutputContent: []byte("first output"),
			SourceProfile: "vendor/profile",
		},
		{
			TargetPath:    "path/to/second.md",
			SourceContent: []byte("second source"),
			OutputContent: []byte("second output"),
			SourceProfile: "vendor/profile",
		},
	}

	manifest, err := generator.Generate([]string{"vendor/profile"}, files)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, exists1 := manifest.Files["path/to/first.md"]
	if !exists1 {
		t.Error("expected file at 'path/to/first.md' to exist")
	}

	_, exists2 := manifest.Files["path/to/second.md"]
	if !exists2 {
		t.Error("expected file at 'path/to/second.md' to exist")
	}
}

// Test 2.1g: ManifestFile entries have correct source and output checksums
func TestGenerateCorrectChecksums(t *testing.T) {
	t.Parallel()
	generator := NewManifestGenerator()

	sourceContent := []byte("template with {{ .Variable }}")
	outputContent := []byte("template with rendered value")

	files := []RenderedFile{
		{
			TargetPath:    "test.md",
			SourceContent: sourceContent,
			OutputContent: outputContent,
			SourceProfile: "vendor/profile",
		},
	}

	manifest, err := generator.Generate([]string{"vendor/profile"}, files)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	file := manifest.Files["test.md"]

	expectedSourceChecksum := ComputeChecksum(sourceContent)
	expectedOutputChecksum := ComputeChecksum(outputContent)

	if file.SourceChecksum != expectedSourceChecksum {
		t.Errorf("expected SourceChecksum '%s', got '%s'", expectedSourceChecksum, file.SourceChecksum)
	}
	if file.OutputChecksum != expectedOutputChecksum {
		t.Errorf("expected OutputChecksum '%s', got '%s'", expectedOutputChecksum, file.OutputChecksum)
	}
}

// Test 2.1h: ManifestFile entries have correct SourceProfile values
func TestGenerateCorrectSourceProfile(t *testing.T) {
	t.Parallel()
	generator := NewManifestGenerator()

	files := []RenderedFile{
		{
			TargetPath:    "from-base.md",
			SourceContent: []byte("base"),
			OutputContent: []byte("base"),
			SourceProfile: "vendor/base-profile",
		},
		{
			TargetPath:    "from-child.md",
			SourceContent: []byte("child"),
			OutputContent: []byte("child"),
			SourceProfile: "vendor/child-profile",
		},
	}

	manifest, err := generator.Generate([]string{"vendor/child-profile"}, files)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	baseFile := manifest.Files["from-base.md"]
	if baseFile.SourceProfile != "vendor/base-profile" {
		t.Errorf("expected SourceProfile 'vendor/base-profile', got '%s'", baseFile.SourceProfile)
	}

	childFile := manifest.Files["from-child.md"]
	if childFile.SourceProfile != "vendor/child-profile" {
		t.Errorf("expected SourceProfile 'vendor/child-profile', got '%s'", childFile.SourceProfile)
	}
}

// =============================================================================
// Task Group 3: Error Handling Tests (3-5 tests)
// =============================================================================

// Test 3.1a: Error when RenderedFile has empty TargetPath
func TestGenerateErrorEmptyTargetPath(t *testing.T) {
	t.Parallel()
	generator := NewManifestGenerator()

	files := []RenderedFile{
		{
			TargetPath:    "", // Empty target path
			SourceContent: []byte("source"),
			OutputContent: []byte("output"),
			SourceProfile: "vendor/profile",
		},
	}

	_, err := generator.Generate([]string{"vendor/profile"}, files)
	if err == nil {
		t.Fatal("expected error for empty TargetPath, got nil")
	}

	genErr, ok := err.(*ManifestGenerationError)
	if !ok {
		t.Fatalf("expected ManifestGenerationError, got %T", err)
	}

	if genErr.TargetPath != "" {
		t.Errorf("expected TargetPath '' in error, got '%s'", genErr.TargetPath)
	}
}

// Test 3.1b: Error when RenderedFile has nil SourceContent
func TestGenerateErrorNilSourceContent(t *testing.T) {
	t.Parallel()
	generator := NewManifestGenerator()

	files := []RenderedFile{
		{
			TargetPath:    "test.md",
			SourceContent: nil, // Nil source content
			OutputContent: []byte("output"),
			SourceProfile: "vendor/profile",
		},
	}

	_, err := generator.Generate([]string{"vendor/profile"}, files)
	if err == nil {
		t.Fatal("expected error for nil SourceContent, got nil")
	}

	genErr, ok := err.(*ManifestGenerationError)
	if !ok {
		t.Fatalf("expected ManifestGenerationError, got %T", err)
	}

	if genErr.TargetPath != "test.md" {
		t.Errorf("expected TargetPath 'test.md' in error, got '%s'", genErr.TargetPath)
	}
}

// Test 3.1c: Error when RenderedFile has nil OutputContent
func TestGenerateErrorNilOutputContent(t *testing.T) {
	t.Parallel()
	generator := NewManifestGenerator()

	files := []RenderedFile{
		{
			TargetPath:    "test.md",
			SourceContent: []byte("source"),
			OutputContent: nil, // Nil output content
			SourceProfile: "vendor/profile",
		},
	}

	_, err := generator.Generate([]string{"vendor/profile"}, files)
	if err == nil {
		t.Fatal("expected error for nil OutputContent, got nil")
	}

	genErr, ok := err.(*ManifestGenerationError)
	if !ok {
		t.Fatalf("expected ManifestGenerationError, got %T", err)
	}

	if genErr.TargetPath != "test.md" {
		t.Errorf("expected TargetPath 'test.md' in error, got '%s'", genErr.TargetPath)
	}
}

// Test 3.1d: Error message includes context about which file caused the issue
func TestGenerateErrorIncludesFileContext(t *testing.T) {
	t.Parallel()
	generator := NewManifestGenerator()

	files := []RenderedFile{
		{
			TargetPath:    "valid.md",
			SourceContent: []byte("source"),
			OutputContent: []byte("output"),
			SourceProfile: "vendor/profile",
		},
		{
			TargetPath:    "problematic/file.md",
			SourceContent: nil, // This causes the error
			OutputContent: []byte("output"),
			SourceProfile: "vendor/profile",
		},
	}

	_, err := generator.Generate([]string{"vendor/profile"}, files)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	errMsg := err.Error()
	if !strings.Contains(errMsg, "problematic/file.md") {
		t.Errorf("expected error message to contain 'problematic/file.md', got '%s'", errMsg)
	}
}

// Test 3.1e: Empty files slice returns valid manifest with empty Files map
func TestGenerateEmptyFilesSlice(t *testing.T) {
	t.Parallel()
	generator := NewManifestGenerator()

	files := []RenderedFile{}

	manifest, err := generator.Generate([]string{"vendor/profile"}, files)
	if err != nil {
		t.Fatalf("unexpected error for empty files slice: %v", err)
	}

	if manifest == nil {
		t.Fatal("expected non-nil manifest for empty files slice")
	}

	if manifest.Files == nil {
		t.Error("expected Files map to be non-nil (initialized empty map)")
	}

	if len(manifest.Files) != 0 {
		t.Errorf("expected 0 files, got %d", len(manifest.Files))
	}

	if manifest.Version != "1.0.0" {
		t.Errorf("expected Version '1.0.0', got '%s'", manifest.Version)
	}

	if manifest.Profiles[0] != "vendor/profile" {
		t.Errorf("expected Profile 'vendor/profile', got '%s'", manifest.Profiles[0])
	}
}

// =============================================================================
// Task Group 4: Integration and Gap-filling Tests (up to 6 additional tests)
// =============================================================================

// Test 4.3a: Manifest generation with files from multiple source profiles (inheritance)
func TestGenerateMultipleSourceProfilesInheritance(t *testing.T) {
	t.Parallel()
	generator := NewManifestGenerator()

	// Simulate a 3-level inheritance chain: grandparent -> parent -> child
	files := []RenderedFile{
		{
			TargetPath:    "from-grandparent.md",
			SourceContent: []byte("grandparent template"),
			OutputContent: []byte("grandparent output"),
			SourceProfile: "vendor/grandparent",
		},
		{
			TargetPath:    "from-parent.md",
			SourceContent: []byte("parent template"),
			OutputContent: []byte("parent output"),
			SourceProfile: "vendor/parent",
		},
		{
			TargetPath:    "from-child.md",
			SourceContent: []byte("child template"),
			OutputContent: []byte("child output"),
			SourceProfile: "vendor/child",
		},
		{
			TargetPath:    "overridden-by-child.md",
			SourceContent: []byte("child override"),
			OutputContent: []byte("child override output"),
			SourceProfile: "vendor/child", // Child overrode parent's file
		},
	}

	manifest, err := generator.Generate([]string{"vendor/child"}, files)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify each file tracks its correct source profile
	testCases := []struct {
		targetPath     string
		expectedSource string
	}{
		{"from-grandparent.md", "vendor/grandparent"},
		{"from-parent.md", "vendor/parent"},
		{"from-child.md", "vendor/child"},
		{"overridden-by-child.md", "vendor/child"},
	}

	for _, tc := range testCases {
		file, exists := manifest.Files[tc.targetPath]
		if !exists {
			t.Errorf("expected file '%s' to exist", tc.targetPath)
			continue
		}
		if file.SourceProfile != tc.expectedSource {
			t.Errorf("file '%s': expected SourceProfile '%s', got '%s'",
				tc.targetPath, tc.expectedSource, file.SourceProfile)
		}
	}

	// Verify the manifest profile is the leaf profile
	if manifest.Profiles[0] != "vendor/child" {
		t.Errorf("expected Profile 'vendor/child', got '%s'", manifest.Profiles[0])
	}
}

// Test 4.3b: Manifest generation with files containing template variables rendered
func TestGenerateWithRenderedTemplateVariables(t *testing.T) {
	t.Parallel()
	generator := NewManifestGenerator()

	// Source has template syntax, output has rendered values
	sourceContent := []byte(`# Project: {{ .ProjectName }}
Version: {{ .Version }}
{{ if .EnableFeature }}Feature is enabled{{ end }}`)

	outputContent := []byte(`# Project: my-awesome-project
Version: 1.0.0
Feature is enabled`)

	files := []RenderedFile{
		{
			TargetPath:    "README.md",
			SourceContent: sourceContent,
			OutputContent: outputContent,
			SourceProfile: "vendor/template-profile",
		},
	}

	manifest, err := generator.Generate([]string{"vendor/template-profile"}, files)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	file := manifest.Files["README.md"]

	// Checksums should be different because source and output content differ
	if file.SourceChecksum == file.OutputChecksum {
		t.Error("expected different checksums for source and output content")
	}

	// Verify checksums match what we compute independently
	if file.SourceChecksum != ComputeChecksum(sourceContent) {
		t.Errorf("source checksum mismatch")
	}
	if file.OutputChecksum != ComputeChecksum(outputContent) {
		t.Errorf("output checksum mismatch")
	}
}

// Test 4.3c: Checksum consistency across multiple Generate calls
func TestGenerateChecksumConsistencyAcrossCalls(t *testing.T) {
	t.Parallel()
	generator := NewManifestGenerator()

	files := []RenderedFile{
		{
			TargetPath:    "test.md",
			SourceContent: []byte("source content"),
			OutputContent: []byte("output content"),
			SourceProfile: "vendor/profile",
		},
	}

	manifest1, err := generator.Generate([]string{"vendor/profile"}, files)
	if err != nil {
		t.Fatalf("first generate error: %v", err)
	}

	manifest2, err := generator.Generate([]string{"vendor/profile"}, files)
	if err != nil {
		t.Fatalf("second generate error: %v", err)
	}

	file1 := manifest1.Files["test.md"]
	file2 := manifest2.Files["test.md"]

	if file1.SourceChecksum != file2.SourceChecksum {
		t.Errorf("source checksum inconsistent: '%s' vs '%s'",
			file1.SourceChecksum, file2.SourceChecksum)
	}
	if file1.OutputChecksum != file2.OutputChecksum {
		t.Errorf("output checksum inconsistent: '%s' vs '%s'",
			file1.OutputChecksum, file2.OutputChecksum)
	}
}

// Test 4.3d: Manifest JSON serialization round-trip with generated data
func TestGenerateJSONSerializationRoundTrip(t *testing.T) {
	t.Parallel()
	generator := NewManifestGenerator()

	files := []RenderedFile{
		{
			TargetPath:    "weftlo/standards/api.md",
			SourceContent: []byte("api source"),
			OutputContent: []byte("api output"),
			SourceProfile: "vendor/base",
		},
		{
			TargetPath:    "CLAUDE.md",
			SourceContent: []byte("claude source"),
			OutputContent: []byte("claude output"),
			SourceProfile: "vendor/child",
		},
	}

	manifest, err := generator.Generate([]string{"vendor/child"}, files)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Serialize to JSON
	jsonData, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		t.Fatalf("failed to marshal manifest: %v", err)
	}

	// Deserialize back
	var unmarshaled Manifest
	if err := json.Unmarshal(jsonData, &unmarshaled); err != nil {
		t.Fatalf("failed to unmarshal manifest: %v", err)
	}

	// Verify round-trip preserves all data
	if unmarshaled.Version != manifest.Version {
		t.Errorf("version mismatch: expected '%s', got '%s'",
			manifest.Version, unmarshaled.Version)
	}
	if unmarshaled.Profiles[0] != manifest.Profiles[0] {
		t.Errorf("profile mismatch: expected '%s', got '%s'",
			manifest.Profiles[0], unmarshaled.Profiles[0])
	}
	if len(unmarshaled.Files) != len(manifest.Files) {
		t.Errorf("files count mismatch: expected %d, got %d",
			len(manifest.Files), len(unmarshaled.Files))
	}

	// Verify individual file entries
	for path, originalFile := range manifest.Files {
		unmarshaledFile, exists := unmarshaled.Files[path]
		if !exists {
			t.Errorf("missing file '%s' after round-trip", path)
			continue
		}
		if unmarshaledFile.SourceChecksum != originalFile.SourceChecksum {
			t.Errorf("file '%s' source checksum mismatch", path)
		}
		if unmarshaledFile.OutputChecksum != originalFile.OutputChecksum {
			t.Errorf("file '%s' output checksum mismatch", path)
		}
		if unmarshaledFile.SourceProfile != originalFile.SourceProfile {
			t.Errorf("file '%s' source profile mismatch", path)
		}
	}
}

// Test 4.3e: Generate with large number of files
func TestGenerateLargeFileCount(t *testing.T) {
	t.Parallel()
	generator := NewManifestGenerator()

	// Create 100 files to test performance and correctness at scale
	fileCount := 100
	files := make([]RenderedFile, fileCount)
	for i := 0; i < fileCount; i++ {
		files[i] = RenderedFile{
			TargetPath:    "path/to/file" + string(rune('a'+i%26)) + string(rune('0'+i/26)) + ".md",
			SourceContent: []byte("source content " + string(rune(i))),
			OutputContent: []byte("output content " + string(rune(i))),
			SourceProfile: "vendor/profile",
		}
	}

	manifest, err := generator.Generate([]string{"vendor/profile"}, files)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(manifest.Files) != fileCount {
		t.Errorf("expected %d files, got %d", fileCount, len(manifest.Files))
	}
}

// Test 4.3f: Nil files slice returns valid manifest with empty Files map
func TestGenerateNilFilesSlice(t *testing.T) {
	t.Parallel()
	generator := NewManifestGenerator()

	manifest, err := generator.Generate([]string{"vendor/profile"}, nil)
	if err != nil {
		t.Fatalf("unexpected error for nil files slice: %v", err)
	}

	if manifest == nil {
		t.Fatal("expected non-nil manifest for nil files slice")
	}

	if manifest.Files == nil {
		t.Error("expected Files map to be non-nil (initialized empty map)")
	}

	if len(manifest.Files) != 0 {
		t.Errorf("expected 0 files, got %d", len(manifest.Files))
	}
}
