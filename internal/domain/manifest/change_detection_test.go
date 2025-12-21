package manifest

import (
	"testing"

	"github.com/spf13/afero"

	"github.com/simensen/weftlo/internal/app/routing"
	domainprofile "github.com/simensen/weftlo/internal/domain/profile"
	"github.com/simensen/weftlo/internal/infrastructure/profile"
)

// =============================================================================
// Task Group 3: DetectChanges Core Function Tests (6-8 tests)
// =============================================================================

// Helper function to create a test MergedProfile with specified source files
func createTestMergedProfile(fs afero.Fs, profilesBasePath string, sourcePaths []string) *profile.MergedProfile {
	// Create template files
	templateFiles := make([]domainprofile.TemplateFile, len(sourcePaths))
	for i, sp := range sourcePaths {
		templateFiles[i] = domainprofile.TemplateFile{
			SourcePath: sp,
			IsPartial:  false,
		}
	}

	// Create a single profile in the chain
	testProfile := &domainprofile.Profile{
		Name:          "test/profile",
		TemplateFiles: templateFiles,
	}

	return profile.NewMergedProfile([]*domainprofile.Profile{testProfile}, fs, profilesBasePath)
}

// createDefaultRouter creates a simple Router for tests using a default_target configuration.
// This router maps all source paths to a default target path, simulating the legacy installPrefix behavior.
func createDefaultRouter(defaultTarget string) routing.Router {
	contentConfig := domainprofile.ContentConfig{
		DefaultTarget: defaultTarget,
	}
	router, _ := routing.NewContentRouter(contentConfig, nil, nil)
	return router
}

// Test 3.1a: Unchanged status - both checksums match manifest
func TestDetectChangesUnchangedStatus(t *testing.T) {
	t.Parallel()
	fs := afero.NewMemMapFs()
	projectRoot := "/project"
	profilesBasePath := "/profiles"

	// Setup source template file (note: path includes content root)
	sourceContent := []byte("template content")
	sourceChecksum := ComputeChecksum(sourceContent)
	afero.WriteFile(fs, "/profiles/test/profile/content/config.md", sourceContent, 0644)

	// Setup output file with same content (unchanged)
	outputContent := []byte("rendered content")
	outputChecksum := ComputeChecksum(outputContent)
	afero.WriteFile(fs, "/project/config.md", outputContent, 0644)

	// Create manifest with matching checksums
	manifest := &Manifest{
		Files: map[string]ManifestFile{
			"config.md": {
				SourceChecksum: sourceChecksum,
				OutputChecksum: outputChecksum,
			},
		},
	}

	// Create merged profile
	mergedProfile := createTestMergedProfile(fs, profilesBasePath, []string{"config.md"})

	// Create router with empty default target (source path == target path)
	router := createDefaultRouter("")

	// Detect changes
	changes, err := DetectChanges(manifest, mergedProfile, fs, projectRoot, router)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	info, exists := changes["config.md"]
	if !exists {
		t.Fatal("expected config.md in changes map")
	}
	if info.Status != FileStatusUnchanged {
		t.Errorf("expected FileStatusUnchanged, got %s", info.Status)
	}
}

// Test 3.1b: SourceChanged status - source differs, output matches
func TestDetectChangesSourceChangedStatus(t *testing.T) {
	t.Parallel()
	fs := afero.NewMemMapFs()
	projectRoot := "/project"
	profilesBasePath := "/profiles"

	// Setup source template file (new content)
	newSourceContent := []byte("updated template content")
	afero.WriteFile(fs, "/profiles/test/profile/content/config.md", newSourceContent, 0644)

	// Setup output file (matches manifest)
	outputContent := []byte("rendered content")
	outputChecksum := ComputeChecksum(outputContent)
	afero.WriteFile(fs, "/project/config.md", outputContent, 0644)

	// Create manifest with OLD source checksum, current output checksum
	oldSourceChecksum := ComputeChecksum([]byte("old template content"))
	manifest := &Manifest{
		Files: map[string]ManifestFile{
			"config.md": {
				SourceChecksum: oldSourceChecksum,
				OutputChecksum: outputChecksum,
			},
		},
	}

	// Create merged profile
	mergedProfile := createTestMergedProfile(fs, profilesBasePath, []string{"config.md"})

	// Create router
	router := createDefaultRouter("")

	// Detect changes
	changes, err := DetectChanges(manifest, mergedProfile, fs, projectRoot, router)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	info, exists := changes["config.md"]
	if !exists {
		t.Fatal("expected config.md in changes map")
	}
	if info.Status != FileStatusSourceChanged {
		t.Errorf("expected FileStatusSourceChanged, got %s", info.Status)
	}
}

// Test 3.1c: UserModified status - output differs, source matches
func TestDetectChangesUserModifiedStatus(t *testing.T) {
	t.Parallel()
	fs := afero.NewMemMapFs()
	projectRoot := "/project"
	profilesBasePath := "/profiles"

	// Setup source template file (matches manifest)
	sourceContent := []byte("template content")
	sourceChecksum := ComputeChecksum(sourceContent)
	afero.WriteFile(fs, "/profiles/test/profile/content/config.md", sourceContent, 0644)

	// Setup output file (user modified)
	userModifiedContent := []byte("user modified rendered content")
	afero.WriteFile(fs, "/project/config.md", userModifiedContent, 0644)

	// Create manifest with current source checksum, OLD output checksum
	oldOutputChecksum := ComputeChecksum([]byte("original rendered content"))
	manifest := &Manifest{
		Files: map[string]ManifestFile{
			"config.md": {
				SourceChecksum: sourceChecksum,
				OutputChecksum: oldOutputChecksum,
			},
		},
	}

	// Create merged profile
	mergedProfile := createTestMergedProfile(fs, profilesBasePath, []string{"config.md"})

	// Create router
	router := createDefaultRouter("")

	// Detect changes
	changes, err := DetectChanges(manifest, mergedProfile, fs, projectRoot, router)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	info, exists := changes["config.md"]
	if !exists {
		t.Fatal("expected config.md in changes map")
	}
	if info.Status != FileStatusUserModified {
		t.Errorf("expected FileStatusUserModified, got %s", info.Status)
	}
}

// Test 3.1d: Conflict status - both checksums differ
func TestDetectChangesConflictStatus(t *testing.T) {
	t.Parallel()
	fs := afero.NewMemMapFs()
	projectRoot := "/project"
	profilesBasePath := "/profiles"

	// Setup source template file (new content)
	newSourceContent := []byte("updated template content")
	afero.WriteFile(fs, "/profiles/test/profile/content/config.md", newSourceContent, 0644)

	// Setup output file (user modified)
	userModifiedContent := []byte("user modified rendered content")
	afero.WriteFile(fs, "/project/config.md", userModifiedContent, 0644)

	// Create manifest with OLD checksums for both
	oldSourceChecksum := ComputeChecksum([]byte("old template content"))
	oldOutputChecksum := ComputeChecksum([]byte("original rendered content"))
	manifest := &Manifest{
		Files: map[string]ManifestFile{
			"config.md": {
				SourceChecksum: oldSourceChecksum,
				OutputChecksum: oldOutputChecksum,
			},
		},
	}

	// Create merged profile
	mergedProfile := createTestMergedProfile(fs, profilesBasePath, []string{"config.md"})

	// Create router
	router := createDefaultRouter("")

	// Detect changes
	changes, err := DetectChanges(manifest, mergedProfile, fs, projectRoot, router)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	info, exists := changes["config.md"]
	if !exists {
		t.Fatal("expected config.md in changes map")
	}
	if info.Status != FileStatusConflict {
		t.Errorf("expected FileStatusConflict, got %s", info.Status)
	}
}

// Test 3.1e: New status - file in profile but not in manifest
func TestDetectChangesNewStatus(t *testing.T) {
	t.Parallel()
	fs := afero.NewMemMapFs()
	projectRoot := "/project"
	profilesBasePath := "/profiles"

	// Setup source template file
	sourceContent := []byte("new template content")
	afero.WriteFile(fs, "/profiles/test/profile/content/new-file.md", sourceContent, 0644)

	// Create empty manifest (no files recorded)
	manifest := &Manifest{
		Files: map[string]ManifestFile{},
	}

	// Create merged profile with a new file
	mergedProfile := createTestMergedProfile(fs, profilesBasePath, []string{"new-file.md"})

	// Create router
	router := createDefaultRouter("")

	// Detect changes
	changes, err := DetectChanges(manifest, mergedProfile, fs, projectRoot, router)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	info, exists := changes["new-file.md"]
	if !exists {
		t.Fatal("expected new-file.md in changes map")
	}
	if info.Status != FileStatusNew {
		t.Errorf("expected FileStatusNew, got %s", info.Status)
	}
}

// Test 3.1f: Removed status - file in manifest but not in profile
func TestDetectChangesRemovedStatus(t *testing.T) {
	t.Parallel()
	fs := afero.NewMemMapFs()
	projectRoot := "/project"
	profilesBasePath := "/profiles"

	// Setup output file that still exists
	outputContent := []byte("rendered content")
	afero.WriteFile(fs, "/project/old-file.md", outputContent, 0644)

	// Create manifest with a file that no longer exists in profile
	manifest := &Manifest{
		Files: map[string]ManifestFile{
			"old-file.md": {
				SourceChecksum: "sha256:abc123",
				OutputChecksum: "sha256:def456",
			},
		},
	}

	// Create merged profile with no files (empty profile)
	mergedProfile := createTestMergedProfile(fs, profilesBasePath, []string{})

	// Create router
	router := createDefaultRouter("")

	// Detect changes
	changes, err := DetectChanges(manifest, mergedProfile, fs, projectRoot, router)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	info, exists := changes["old-file.md"]
	if !exists {
		t.Fatal("expected old-file.md in changes map")
	}
	if info.Status != FileStatusRemoved {
		t.Errorf("expected FileStatusRemoved, got %s", info.Status)
	}
}

// Test 3.1g: Error handling for filesystem read failures on source template
func TestDetectChangesSourceReadError(t *testing.T) {
	t.Parallel()
	fs := afero.NewMemMapFs()
	projectRoot := "/project"
	profilesBasePath := "/profiles"

	// Setup output file but NOT source template (source is missing)
	outputContent := []byte("rendered content")
	afero.WriteFile(fs, "/project/config.md", outputContent, 0644)
	// Note: NOT creating /profiles/test/profile/content/config.md

	// Create manifest with existing file
	manifest := &Manifest{
		Files: map[string]ManifestFile{
			"config.md": {
				SourceChecksum: "sha256:abc123",
				OutputChecksum: "sha256:def456",
			},
		},
	}

	// Create merged profile expecting the file
	mergedProfile := createTestMergedProfile(fs, profilesBasePath, []string{"config.md"})

	// Create router
	router := createDefaultRouter("")

	// Detect changes - should return error for missing source template
	_, err := DetectChanges(manifest, mergedProfile, fs, projectRoot, router)
	if err == nil {
		t.Fatal("expected error for missing source template")
	}

	// Verify error type
	detectionErr, ok := err.(*ChangeDetectionError)
	if !ok {
		t.Fatalf("expected *ChangeDetectionError, got %T", err)
	}
	if detectionErr.TargetPath != "config.md" {
		t.Errorf("expected TargetPath 'config.md', got '%s'", detectionErr.TargetPath)
	}
}

// Test 3.1h: Empty manifest with populated profile - all files are New
func TestDetectChangesEmptyManifestAllNew(t *testing.T) {
	t.Parallel()
	fs := afero.NewMemMapFs()
	projectRoot := "/project"
	profilesBasePath := "/profiles"

	// Setup multiple source template files
	afero.WriteFile(fs, "/profiles/test/profile/content/file1.md", []byte("content1"), 0644)
	afero.WriteFile(fs, "/profiles/test/profile/content/file2.md", []byte("content2"), 0644)
	afero.WriteFile(fs, "/profiles/test/profile/content/file3.md", []byte("content3"), 0644)

	// Create empty manifest
	manifest := &Manifest{
		Files: map[string]ManifestFile{},
	}

	// Create merged profile with multiple files
	mergedProfile := createTestMergedProfile(fs, profilesBasePath, []string{
		"file1.md",
		"file2.md",
		"file3.md",
	})

	// Create router
	router := createDefaultRouter("")

	// Detect changes
	changes, err := DetectChanges(manifest, mergedProfile, fs, projectRoot, router)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify all files are New
	expectedFiles := []string{"file1.md", "file2.md", "file3.md"}
	if len(changes) != len(expectedFiles) {
		t.Errorf("expected %d files in changes, got %d", len(expectedFiles), len(changes))
	}

	for _, file := range expectedFiles {
		info, exists := changes[file]
		if !exists {
			t.Errorf("expected %s in changes map", file)
			continue
		}
		if info.Status != FileStatusNew {
			t.Errorf("expected FileStatusNew for %s, got %s", file, info.Status)
		}
	}
}

// =============================================================================
// Task Group 4: Strategic Gap-Filling Tests (up to 5 additional tests)
// =============================================================================

// Test 4.3a: Populated manifest with empty profile - all files are Removed
// This is the inverse of TestDetectChangesEmptyManifestAllNew
func TestDetectChangesPopulatedManifestAllRemoved(t *testing.T) {
	t.Parallel()
	fs := afero.NewMemMapFs()
	projectRoot := "/project"
	profilesBasePath := "/profiles"

	// Setup output files that still exist in the project
	afero.WriteFile(fs, "/project/file1.md", []byte("content1"), 0644)
	afero.WriteFile(fs, "/project/file2.md", []byte("content2"), 0644)
	afero.WriteFile(fs, "/project/file3.md", []byte("content3"), 0644)

	// Create manifest with multiple files
	manifest := &Manifest{
		Files: map[string]ManifestFile{
			"file1.md": {
				SourceChecksum: "sha256:abc123",
				OutputChecksum: "sha256:def456",
			},
			"file2.md": {
				SourceChecksum: "sha256:ghi789",
				OutputChecksum: "sha256:jkl012",
			},
			"file3.md": {
				SourceChecksum: "sha256:mno345",
				OutputChecksum: "sha256:pqr678",
			},
		},
	}

	// Create empty merged profile (no files)
	mergedProfile := createTestMergedProfile(fs, profilesBasePath, []string{})

	// Create router
	router := createDefaultRouter("")

	// Detect changes
	changes, err := DetectChanges(manifest, mergedProfile, fs, projectRoot, router)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify all files are Removed
	expectedFiles := []string{"file1.md", "file2.md", "file3.md"}
	if len(changes) != len(expectedFiles) {
		t.Errorf("expected %d files in changes, got %d", len(expectedFiles), len(changes))
	}

	for _, file := range expectedFiles {
		info, exists := changes[file]
		if !exists {
			t.Errorf("expected %s in changes map", file)
			continue
		}
		if info.Status != FileStatusRemoved {
			t.Errorf("expected FileStatusRemoved for %s, got %s", file, info.Status)
		}
	}
}

// Test 4.3b: Mixed statuses in a single detection run
// Tests a realistic scenario where multiple files have different statuses simultaneously
func TestDetectChangesMixedStatuses(t *testing.T) {
	t.Parallel()
	fs := afero.NewMemMapFs()
	projectRoot := "/project"
	profilesBasePath := "/profiles"

	// --- Setup files for each status ---

	// Unchanged file: both checksums match
	unchangedSourceContent := []byte("unchanged source")
	unchangedOutputContent := []byte("unchanged output")
	afero.WriteFile(fs, "/profiles/test/profile/content/unchanged.md", unchangedSourceContent, 0644)
	afero.WriteFile(fs, "/project/unchanged.md", unchangedOutputContent, 0644)

	// SourceChanged file: source differs, output matches
	sourceChangedNewContent := []byte("source changed - new content")
	sourceChangedOutputContent := []byte("source changed output")
	afero.WriteFile(fs, "/profiles/test/profile/content/source-changed.md", sourceChangedNewContent, 0644)
	afero.WriteFile(fs, "/project/source-changed.md", sourceChangedOutputContent, 0644)

	// UserModified file: output differs, source matches
	userModifiedSourceContent := []byte("user modified source")
	userModifiedNewOutputContent := []byte("user modified - edited by user")
	afero.WriteFile(fs, "/profiles/test/profile/content/user-modified.md", userModifiedSourceContent, 0644)
	afero.WriteFile(fs, "/project/user-modified.md", userModifiedNewOutputContent, 0644)

	// Conflict file: both checksums differ
	conflictNewSourceContent := []byte("conflict - new source")
	conflictNewOutputContent := []byte("conflict - user edited")
	afero.WriteFile(fs, "/profiles/test/profile/content/conflict.md", conflictNewSourceContent, 0644)
	afero.WriteFile(fs, "/project/conflict.md", conflictNewOutputContent, 0644)

	// New file: in profile but not in manifest
	newFileContent := []byte("brand new file")
	afero.WriteFile(fs, "/profiles/test/profile/content/new-file.md", newFileContent, 0644)

	// Removed file: in manifest but not in profile (no source file needed)
	afero.WriteFile(fs, "/project/removed.md", []byte("removed file output"), 0644)

	// Create manifest with appropriate checksums
	manifest := &Manifest{
		Files: map[string]ManifestFile{
			"unchanged.md": {
				SourceChecksum: ComputeChecksum(unchangedSourceContent),
				OutputChecksum: ComputeChecksum(unchangedOutputContent),
			},
			"source-changed.md": {
				SourceChecksum: ComputeChecksum([]byte("source changed - old content")), // Old source
				OutputChecksum: ComputeChecksum(sourceChangedOutputContent),             // Current output
			},
			"user-modified.md": {
				SourceChecksum: ComputeChecksum(userModifiedSourceContent),            // Current source
				OutputChecksum: ComputeChecksum([]byte("user modified - old output")), // Old output
			},
			"conflict.md": {
				SourceChecksum: ComputeChecksum([]byte("conflict - old source")), // Old source
				OutputChecksum: ComputeChecksum([]byte("conflict - old output")), // Old output
			},
			"removed.md": {
				SourceChecksum: "sha256:removed-source",
				OutputChecksum: "sha256:removed-output",
			},
			// Note: new-file.md is NOT in manifest
		},
	}

	// Create merged profile with all files EXCEPT removed.md
	mergedProfile := createTestMergedProfile(fs, profilesBasePath, []string{
		"unchanged.md",
		"source-changed.md",
		"user-modified.md",
		"conflict.md",
		"new-file.md",
	})

	// Create router
	router := createDefaultRouter("")

	// Detect changes
	changes, err := DetectChanges(manifest, mergedProfile, fs, projectRoot, router)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify we have exactly 6 files
	if len(changes) != 6 {
		t.Errorf("expected 6 files in changes, got %d", len(changes))
	}

	// Verify each file has the correct status
	expectedStatuses := map[string]FileStatus{
		"unchanged.md":      FileStatusUnchanged,
		"source-changed.md": FileStatusSourceChanged,
		"user-modified.md":  FileStatusUserModified,
		"conflict.md":       FileStatusConflict,
		"new-file.md":       FileStatusNew,
		"removed.md":        FileStatusRemoved,
	}

	for file, expectedStatus := range expectedStatuses {
		info, exists := changes[file]
		if !exists {
			t.Errorf("expected %s in changes map", file)
			continue
		}
		if info.Status != expectedStatus {
			t.Errorf("expected %s for %s, got %s", expectedStatus, file, info.Status)
		}
	}
}

// Test 4.3c: Missing output file is treated as UserModified
// When a user deletes an output file, it should be detected as a user modification
func TestDetectChangesMissingOutputFileTreatedAsUserModified(t *testing.T) {
	t.Parallel()
	fs := afero.NewMemMapFs()
	projectRoot := "/project"
	profilesBasePath := "/profiles"

	// Setup source template file (matches manifest)
	sourceContent := []byte("template content")
	sourceChecksum := ComputeChecksum(sourceContent)
	afero.WriteFile(fs, "/profiles/test/profile/content/config.md", sourceContent, 0644)

	// Note: NOT creating the output file - simulating user deletion
	// The output file /project/config.md does not exist

	// Create manifest with matching source checksum and a recorded output checksum
	manifest := &Manifest{
		Files: map[string]ManifestFile{
			"config.md": {
				SourceChecksum: sourceChecksum,
				OutputChecksum: ComputeChecksum([]byte("original rendered content")),
			},
		},
	}

	// Create merged profile
	mergedProfile := createTestMergedProfile(fs, profilesBasePath, []string{"config.md"})

	// Create router
	router := createDefaultRouter("")

	// Detect changes - missing output file should be treated as user modification
	changes, err := DetectChanges(manifest, mergedProfile, fs, projectRoot, router)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	info, exists := changes["config.md"]
	if !exists {
		t.Fatal("expected config.md in changes map")
	}
	if info.Status != FileStatusUserModified {
		t.Errorf("expected FileStatusUserModified for deleted output file, got %s", info.Status)
	}
}

// =============================================================================
// Source Content Preservation Integration Tests
// These tests verify the change detection accuracy when source and output
// content differ (the key scenario for source content preservation).
// =============================================================================

// Test: Distinct source and output checksums enable accurate change detection
// This verifies that when source template content differs from rendered output,
// change detection can accurately distinguish between source changes and user modifications.
func TestDetectChanges_DistinctSourceAndOutputChecksums(t *testing.T) {
	t.Parallel()
	fs := afero.NewMemMapFs()
	projectRoot := "/project"
	profilesBasePath := "/profiles"

	// Source template content includes a variable placeholder (simulating pre-render state)
	sourceContent := []byte("Hello, {{ .Variables.name }}!")
	sourceChecksum := ComputeChecksum(sourceContent)

	// Rendered output content has the variable replaced
	outputContent := []byte("Hello, World!")
	outputChecksum := ComputeChecksum(outputContent)

	// Verify source and output checksums are different
	if sourceChecksum == outputChecksum {
		t.Fatal("test setup error: source and output checksums should differ")
	}

	afero.WriteFile(fs, "/profiles/test/profile/content/greeting.md", sourceContent, 0644)
	afero.WriteFile(fs, "/project/greeting.md", outputContent, 0644)

	// Create manifest with distinct checksums
	manifest := &Manifest{
		Files: map[string]ManifestFile{
			"greeting.md": {
				SourceChecksum: sourceChecksum,
				OutputChecksum: outputChecksum,
			},
		},
	}

	mergedProfile := createTestMergedProfile(fs, profilesBasePath, []string{"greeting.md"})

	// Create router
	router := createDefaultRouter("")

	// Detect changes - should be unchanged since both match manifest
	changes, err := DetectChanges(manifest, mergedProfile, fs, projectRoot, router)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	info, exists := changes["greeting.md"]
	if !exists {
		t.Fatal("expected greeting.md in changes map")
	}
	if info.Status != FileStatusUnchanged {
		t.Errorf("expected FileStatusUnchanged when both checksums match, got %s", info.Status)
	}
}

// Test: Source changed but output untouched detects SourceChanged
// Simulates updating a template with a new variable value while user hasn't modified output
func TestDetectChanges_SourceChangedWithDistinctContent(t *testing.T) {
	t.Parallel()
	fs := afero.NewMemMapFs()
	projectRoot := "/project"
	profilesBasePath := "/profiles"

	// NEW source template content (template was updated)
	newSourceContent := []byte("Goodbye, {{ .Variables.name }}!")
	afero.WriteFile(fs, "/profiles/test/profile/content/greeting.md", newSourceContent, 0644)

	// Output file still has the OLD rendered content (user hasn't modified it)
	outputContent := []byte("Hello, World!")
	outputChecksum := ComputeChecksum(outputContent)
	afero.WriteFile(fs, "/project/greeting.md", outputContent, 0644)

	// Manifest has OLD source checksum and current output checksum
	oldSourceContent := []byte("Hello, {{ .Variables.name }}!")
	manifest := &Manifest{
		Files: map[string]ManifestFile{
			"greeting.md": {
				SourceChecksum: ComputeChecksum(oldSourceContent),
				OutputChecksum: outputChecksum,
			},
		},
	}

	mergedProfile := createTestMergedProfile(fs, profilesBasePath, []string{"greeting.md"})

	// Create router
	router := createDefaultRouter("")

	// Detect changes - should be SourceChanged
	changes, err := DetectChanges(manifest, mergedProfile, fs, projectRoot, router)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	info, exists := changes["greeting.md"]
	if !exists {
		t.Fatal("expected greeting.md in changes map")
	}
	if info.Status != FileStatusSourceChanged {
		t.Errorf("expected FileStatusSourceChanged when source differs, got %s", info.Status)
	}
}

// Test: User modified output while source unchanged detects UserModified
// Simulates user editing the rendered file without any profile template changes
func TestDetectChanges_UserModifiedWithDistinctContent(t *testing.T) {
	t.Parallel()
	fs := afero.NewMemMapFs()
	projectRoot := "/project"
	profilesBasePath := "/profiles"

	// Source template unchanged
	sourceContent := []byte("Hello, {{ .Variables.name }}!")
	sourceChecksum := ComputeChecksum(sourceContent)
	afero.WriteFile(fs, "/profiles/test/profile/content/greeting.md", sourceContent, 0644)

	// Output file was modified by user
	userModifiedOutput := []byte("Hello, World! (with user notes)")
	afero.WriteFile(fs, "/project/greeting.md", userModifiedOutput, 0644)

	// Manifest has current source checksum but OLD output checksum
	originalOutput := []byte("Hello, World!")
	manifest := &Manifest{
		Files: map[string]ManifestFile{
			"greeting.md": {
				SourceChecksum: sourceChecksum,
				OutputChecksum: ComputeChecksum(originalOutput),
			},
		},
	}

	mergedProfile := createTestMergedProfile(fs, profilesBasePath, []string{"greeting.md"})

	// Create router
	router := createDefaultRouter("")

	// Detect changes - should be UserModified
	changes, err := DetectChanges(manifest, mergedProfile, fs, projectRoot, router)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	info, exists := changes["greeting.md"]
	if !exists {
		t.Fatal("expected greeting.md in changes map")
	}
	if info.Status != FileStatusUserModified {
		t.Errorf("expected FileStatusUserModified when output differs, got %s", info.Status)
	}
}

// Test: Both source and output changed detects Conflict
// Simulates the worst case: profile updated AND user modified the output
func TestDetectChanges_ConflictWithDistinctContent(t *testing.T) {
	t.Parallel()
	fs := afero.NewMemMapFs()
	projectRoot := "/project"
	profilesBasePath := "/profiles"

	// NEW source template content
	newSourceContent := []byte("Goodbye, {{ .Variables.name }}!")
	afero.WriteFile(fs, "/profiles/test/profile/content/greeting.md", newSourceContent, 0644)

	// User also modified the output file
	userModifiedOutput := []byte("Hello, World! (user modified)")
	afero.WriteFile(fs, "/project/greeting.md", userModifiedOutput, 0644)

	// Manifest has OLD checksums for both
	oldSourceContent := []byte("Hello, {{ .Variables.name }}!")
	originalOutput := []byte("Hello, World!")
	manifest := &Manifest{
		Files: map[string]ManifestFile{
			"greeting.md": {
				SourceChecksum: ComputeChecksum(oldSourceContent),
				OutputChecksum: ComputeChecksum(originalOutput),
			},
		},
	}

	mergedProfile := createTestMergedProfile(fs, profilesBasePath, []string{"greeting.md"})

	// Create router
	router := createDefaultRouter("")

	// Detect changes - should be Conflict
	changes, err := DetectChanges(manifest, mergedProfile, fs, projectRoot, router)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	info, exists := changes["greeting.md"]
	if !exists {
		t.Fatal("expected greeting.md in changes map")
	}
	if info.Status != FileStatusConflict {
		t.Errorf("expected FileStatusConflict when both differ, got %s", info.Status)
	}
}

// Test: Missing output file with source changed is Conflict
// When source template changed AND user deleted the output, this is a conflict
func TestDetectChanges_MissingOutputWithSourceChanged(t *testing.T) {
	t.Parallel()
	fs := afero.NewMemMapFs()
	projectRoot := "/project"
	profilesBasePath := "/profiles"

	// NEW source template content
	newSourceContent := []byte("Goodbye, {{ .Variables.name }}!")
	afero.WriteFile(fs, "/profiles/test/profile/content/greeting.md", newSourceContent, 0644)

	// User DELETED the output file (not creating it)
	// The empty checksum won't match the manifest

	// Manifest has OLD checksums for both
	oldSourceContent := []byte("Hello, {{ .Variables.name }}!")
	originalOutput := []byte("Hello, World!")
	manifest := &Manifest{
		Files: map[string]ManifestFile{
			"greeting.md": {
				SourceChecksum: ComputeChecksum(oldSourceContent),
				OutputChecksum: ComputeChecksum(originalOutput),
			},
		},
	}

	mergedProfile := createTestMergedProfile(fs, profilesBasePath, []string{"greeting.md"})

	// Create router
	router := createDefaultRouter("")

	// Detect changes - should be Conflict (source changed + output "modified" by deletion)
	changes, err := DetectChanges(manifest, mergedProfile, fs, projectRoot, router)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	info, exists := changes["greeting.md"]
	if !exists {
		t.Fatal("expected greeting.md in changes map")
	}
	if info.Status != FileStatusConflict {
		t.Errorf("expected FileStatusConflict when source changed and output deleted, got %s", info.Status)
	}
}

// Test: Install prefix is correctly applied in change detection
// Verifies that target paths with install prefix are correctly computed
func TestDetectChanges_WithInstallPrefix(t *testing.T) {
	t.Parallel()
	fs := afero.NewMemMapFs()
	projectRoot := "/project"
	profilesBasePath := "/profiles"

	// Source template file
	sourceContent := []byte("config content")
	sourceChecksum := ComputeChecksum(sourceContent)
	afero.WriteFile(fs, "/profiles/test/profile/content/settings.md", sourceContent, 0644)

	// Output file at prefixed path
	outputContent := []byte("config content")
	outputChecksum := ComputeChecksum(outputContent)
	afero.WriteFile(fs, "/project/weftlo/settings.md", outputContent, 0644)

	// Manifest tracks the prefixed path
	manifest := &Manifest{
		Files: map[string]ManifestFile{
			"weftlo/settings.md": {
				SourceChecksum: sourceChecksum,
				OutputChecksum: outputChecksum,
			},
		},
	}

	mergedProfile := createTestMergedProfile(fs, profilesBasePath, []string{"settings.md"})

	// Create router with "weftlo" as default target (simulates installPrefix)
	router := createDefaultRouter("weftlo")

	// Detect changes with install prefix
	changes, err := DetectChanges(manifest, mergedProfile, fs, projectRoot, router)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should find the file at the prefixed path
	info, exists := changes["weftlo/settings.md"]
	if !exists {
		t.Fatal("expected weftlo/settings.md in changes map")
	}
	if info.Status != FileStatusUnchanged {
		t.Errorf("expected FileStatusUnchanged, got %s", info.Status)
	}
}

// Test: Template suffix stripping in target path computation
// Verifies .tmpl suffix is correctly stripped when computing target paths
func TestDetectChanges_TemplateFileSuffixStripping(t *testing.T) {
	t.Parallel()
	fs := afero.NewMemMapFs()
	projectRoot := "/project"
	profilesBasePath := "/profiles"

	// Source template file WITH .tmpl suffix
	sourceContent := []byte("template content")
	sourceChecksum := ComputeChecksum(sourceContent)
	afero.WriteFile(fs, "/profiles/test/profile/content/config.md.tmpl", sourceContent, 0644)

	// Output file WITHOUT .tmpl suffix
	outputContent := []byte("rendered content")
	outputChecksum := ComputeChecksum(outputContent)
	afero.WriteFile(fs, "/project/config.md", outputContent, 0644)

	// Manifest tracks the path without .tmpl suffix
	manifest := &Manifest{
		Files: map[string]ManifestFile{
			"config.md": {
				SourceChecksum: sourceChecksum,
				OutputChecksum: outputChecksum,
			},
		},
	}

	mergedProfile := createTestMergedProfile(fs, profilesBasePath, []string{"config.md.tmpl"})

	// Create router
	router := createDefaultRouter("")

	// Detect changes - should correctly map .tmpl source to non-.tmpl target
	changes, err := DetectChanges(manifest, mergedProfile, fs, projectRoot, router)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	info, exists := changes["config.md"]
	if !exists {
		t.Fatal("expected config.md in changes map")
	}
	if info.Status != FileStatusUnchanged {
		t.Errorf("expected FileStatusUnchanged, got %s", info.Status)
	}
}

// Test: Multiple files with mixed .tmpl extensions
// Verifies mixed extension handling works correctly
func TestDetectChanges_MixedTemplateExtensions(t *testing.T) {
	t.Parallel()
	fs := afero.NewMemMapFs()
	projectRoot := "/project"
	profilesBasePath := "/profiles"

	// File WITH .tmpl suffix
	tmplSourceContent := []byte("template file")
	tmplSourceChecksum := ComputeChecksum(tmplSourceContent)
	afero.WriteFile(fs, "/profiles/test/profile/content/config.md.tmpl", tmplSourceContent, 0644)
	tmplOutputContent := []byte("template file rendered")
	tmplOutputChecksum := ComputeChecksum(tmplOutputContent)
	afero.WriteFile(fs, "/project/config.md", tmplOutputContent, 0644)

	// File WITHOUT .tmpl suffix (static file)
	staticSourceContent := []byte("static file")
	staticSourceChecksum := ComputeChecksum(staticSourceContent)
	afero.WriteFile(fs, "/profiles/test/profile/content/readme.md", staticSourceContent, 0644)
	afero.WriteFile(fs, "/project/readme.md", staticSourceContent, 0644) // same content

	manifest := &Manifest{
		Files: map[string]ManifestFile{
			"config.md": {
				SourceChecksum: tmplSourceChecksum,
				OutputChecksum: tmplOutputChecksum,
			},
			"readme.md": {
				SourceChecksum: staticSourceChecksum,
				OutputChecksum: staticSourceChecksum, // same for static file
			},
		},
	}

	mergedProfile := createTestMergedProfile(fs, profilesBasePath, []string{
		"config.md.tmpl",
		"readme.md",
	})

	// Create router
	router := createDefaultRouter("")

	changes, err := DetectChanges(manifest, mergedProfile, fs, projectRoot, router)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Both files should be unchanged
	if len(changes) != 2 {
		t.Errorf("expected 2 files in changes, got %d", len(changes))
	}

	for _, file := range []string{"config.md", "readme.md"} {
		info, exists := changes[file]
		if !exists {
			t.Errorf("expected %s in changes map", file)
			continue
		}
		if info.Status != FileStatusUnchanged {
			t.Errorf("expected FileStatusUnchanged for %s, got %s", file, info.Status)
		}
	}
}
