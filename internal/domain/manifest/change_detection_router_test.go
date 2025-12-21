package manifest

import (
	"testing"

	"github.com/spf13/afero"

	"github.com/simensen/weftlo/internal/app/routing"
	domainprofile "github.com/simensen/weftlo/internal/domain/profile"
	"github.com/simensen/weftlo/internal/infrastructure/profile"
)

// =============================================================================
// Task Group 3.1: Router-based Change Detection Tests (4 tests)
// =============================================================================

// Test 3.1a: DetectChanges uses Router for path resolution
// Verifies that the Router is called and paths are correctly computed
func TestDetectChangesUsesRouterForPathResolution(t *testing.T) {
	t.Parallel()
	fs := afero.NewMemMapFs()
	projectRoot := "/project"
	profilesBasePath := "/profiles"

	// Setup source template file in content/claude directory structure
	sourceContent := []byte("template content")
	sourceChecksum := ComputeChecksum(sourceContent)
	afero.WriteFile(fs, "/profiles/test/profile/content/claude/settings.json", sourceContent, 0644)

	// Setup output file at the routed target path (.claude/settings.json)
	outputContent := []byte("rendered content")
	outputChecksum := ComputeChecksum(outputContent)
	afero.WriteFile(fs, "/project/.claude/settings.json", outputContent, 0644)

	// Create manifest with the routed path (Router maps claude -> .claude)
	manifest := &Manifest{
		Files: map[string]ManifestFile{
			".claude/settings.json": {
				SourceChecksum: sourceChecksum,
				OutputChecksum: outputChecksum,
				TargetCategory: "claude",
			},
		},
	}

	// Create merged profile with source file in claude directory
	mergedProfile := createTestMergedProfileWithContent(fs, profilesBasePath, []string{"claude/settings.json"})

	// Create Router that maps "claude" -> ".claude"
	contentConfig := domainprofile.ContentConfig{
		Targets: map[string]string{
			"claude": ".claude",
		},
		DefaultTarget: "weftlo",
	}
	router, err := routing.NewContentRouter(contentConfig, nil, nil)
	if err != nil {
		t.Fatalf("failed to create router: %v", err)
	}

	// Detect changes using Router
	changes, err := DetectChanges(manifest, mergedProfile, fs, projectRoot, router)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify the file is found at the routed path
	info, exists := changes[".claude/settings.json"]
	if !exists {
		t.Fatal("expected .claude/settings.json in changes map")
	}
	if info.Status != FileStatusUnchanged {
		t.Errorf("expected FileStatusUnchanged, got %s", info.Status)
	}
	if info.TargetCategory != "claude" {
		t.Errorf("expected TargetCategory 'claude', got '%s'", info.TargetCategory)
	}
}

// Test 3.1b: Suppressed files are handled correctly in change detection
// Verifies that files routed to a suppressed target are excluded from results
func TestDetectChangesSuppressedFilesExcluded(t *testing.T) {
	t.Parallel()
	fs := afero.NewMemMapFs()
	projectRoot := "/project"
	profilesBasePath := "/profiles"

	// Setup source files - one normal, one that will be suppressed
	normalContent := []byte("normal content")
	normalChecksum := ComputeChecksum(normalContent)
	afero.WriteFile(fs, "/profiles/test/profile/content/standards/api.md", normalContent, 0644)

	suppressedContent := []byte("suppressed content")
	afero.WriteFile(fs, "/profiles/test/profile/content/deprecated/old.md", suppressedContent, 0644)

	// Setup output file only for the normal file
	outputContent := []byte("rendered content")
	outputChecksum := ComputeChecksum(outputContent)
	afero.WriteFile(fs, "/project/weftlo/standards/api.md", outputContent, 0644)

	// Create manifest with only the normal file (suppressed file should not be tracked)
	manifest := &Manifest{
		Files: map[string]ManifestFile{
			"weftlo/standards/api.md": {
				SourceChecksum: normalChecksum,
				OutputChecksum: outputChecksum,
				TargetCategory: "standards",
			},
		},
	}

	// Create merged profile with both files
	mergedProfile := createTestMergedProfileWithContent(fs, profilesBasePath, []string{
		"standards/api.md",
		"deprecated/old.md",
	})

	// Create Router with "deprecated" category suppressed (null override)
	contentConfig := domainprofile.ContentConfig{
		Targets: map[string]string{
			"standards":  "weftlo/standards",
			"deprecated": "weftlo/deprecated",
		},
		DefaultTarget: "weftlo",
	}
	// nil pointer means suppression
	overrides := map[string]*string{
		"deprecated": nil, // Suppress the deprecated category
	}
	router, err := routing.NewContentRouter(contentConfig, overrides, nil)
	if err != nil {
		t.Fatalf("failed to create router: %v", err)
	}

	// Detect changes using Router
	changes, err := DetectChanges(manifest, mergedProfile, fs, projectRoot, router)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify only the normal file is in results (suppressed file excluded)
	if len(changes) != 1 {
		t.Errorf("expected 1 file in changes, got %d", len(changes))
	}

	_, normalExists := changes["weftlo/standards/api.md"]
	if !normalExists {
		t.Error("expected weftlo/standards/api.md in changes map")
	}

	// Verify suppressed file is NOT in results
	for path := range changes {
		if path == "weftlo/deprecated/old.md" || path == "deprecated/old.md" {
			t.Errorf("expected suppressed file to be excluded, but found: %s", path)
		}
	}
}

// Test 3.1c: TargetCategory is included in change detection results
// Verifies that FileChangeInfo contains the correct TargetCategory from Router
func TestDetectChangesIncludesTargetCategory(t *testing.T) {
	t.Parallel()
	fs := afero.NewMemMapFs()
	projectRoot := "/project"
	profilesBasePath := "/profiles"

	// Setup source files in different content directories
	claudeContent := []byte("claude settings")
	claudeChecksum := ComputeChecksum(claudeContent)
	afero.WriteFile(fs, "/profiles/test/profile/content/claude/settings.json", claudeContent, 0644)

	standardsContent := []byte("standards content")
	standardsChecksum := ComputeChecksum(standardsContent)
	afero.WriteFile(fs, "/profiles/test/profile/content/standards/api.md", standardsContent, 0644)

	// Setup output files at routed paths
	claudeOutput := []byte("claude rendered")
	claudeOutputChecksum := ComputeChecksum(claudeOutput)
	afero.WriteFile(fs, "/project/.claude/settings.json", claudeOutput, 0644)

	standardsOutput := []byte("standards rendered")
	standardsOutputChecksum := ComputeChecksum(standardsOutput)
	afero.WriteFile(fs, "/project/weftlo/standards/api.md", standardsOutput, 0644)

	// Create manifest with both files
	manifest := &Manifest{
		Files: map[string]ManifestFile{
			".claude/settings.json": {
				SourceChecksum: claudeChecksum,
				OutputChecksum: claudeOutputChecksum,
				TargetCategory: "claude",
			},
			"weftlo/standards/api.md": {
				SourceChecksum: standardsChecksum,
				OutputChecksum: standardsOutputChecksum,
				TargetCategory: "standards",
			},
		},
	}

	// Create merged profile
	mergedProfile := createTestMergedProfileWithContent(fs, profilesBasePath, []string{
		"claude/settings.json",
		"standards/api.md",
	})

	// Create Router with multiple target mappings
	contentConfig := domainprofile.ContentConfig{
		Targets: map[string]string{
			"claude":    ".claude",
			"standards": "weftlo/standards",
		},
		DefaultTarget: "weftlo",
	}
	router, err := routing.NewContentRouter(contentConfig, nil, nil)
	if err != nil {
		t.Fatalf("failed to create router: %v", err)
	}

	// Detect changes using Router
	changes, err := DetectChanges(manifest, mergedProfile, fs, projectRoot, router)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify TargetCategory is set correctly for claude file
	claudeInfo, claudeExists := changes[".claude/settings.json"]
	if !claudeExists {
		t.Fatal("expected .claude/settings.json in changes map")
	}
	if claudeInfo.TargetCategory != "claude" {
		t.Errorf("expected TargetCategory 'claude', got '%s'", claudeInfo.TargetCategory)
	}

	// Verify TargetCategory is set correctly for standards file
	standardsInfo, standardsExists := changes["weftlo/standards/api.md"]
	if !standardsExists {
		t.Fatal("expected weftlo/standards/api.md in changes map")
	}
	if standardsInfo.TargetCategory != "standards" {
		t.Errorf("expected TargetCategory 'standards', got '%s'", standardsInfo.TargetCategory)
	}
}

// Test 3.1d: New files from profile include TargetCategory in results
// Verifies that new files (not in manifest) get correct TargetCategory
func TestDetectChangesNewFilesIncludeTargetCategory(t *testing.T) {
	t.Parallel()
	fs := afero.NewMemMapFs()
	projectRoot := "/project"
	profilesBasePath := "/profiles"

	// Setup new source file (not in manifest)
	newContent := []byte("new file content")
	afero.WriteFile(fs, "/profiles/test/profile/content/skills/new-skill.md", newContent, 0644)

	// Empty manifest (no existing files)
	manifest := &Manifest{
		Files: map[string]ManifestFile{},
	}

	// Create merged profile with the new file
	mergedProfile := createTestMergedProfileWithContent(fs, profilesBasePath, []string{
		"skills/new-skill.md",
	})

	// Create Router
	contentConfig := domainprofile.ContentConfig{
		Targets: map[string]string{
			"skills": ".claude/skills",
		},
		DefaultTarget: "weftlo",
	}
	router, err := routing.NewContentRouter(contentConfig, nil, nil)
	if err != nil {
		t.Fatalf("failed to create router: %v", err)
	}

	// Detect changes using Router
	changes, err := DetectChanges(manifest, mergedProfile, fs, projectRoot, router)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify new file has correct status and TargetCategory
	newInfo, exists := changes[".claude/skills/new-skill.md"]
	if !exists {
		t.Fatal("expected .claude/skills/new-skill.md in changes map")
	}
	if newInfo.Status != FileStatusNew {
		t.Errorf("expected FileStatusNew, got %s", newInfo.Status)
	}
	if newInfo.TargetCategory != "skills" {
		t.Errorf("expected TargetCategory 'skills', got '%s'", newInfo.TargetCategory)
	}
}

// Helper function to create a test MergedProfile with content directory structure
func createTestMergedProfileWithContent(fs afero.Fs, profilesBasePath string, sourcePaths []string) *profile.MergedProfile {
	// Create template files
	templateFiles := make([]domainprofile.TemplateFile, len(sourcePaths))
	for i, sp := range sourcePaths {
		templateFiles[i] = domainprofile.TemplateFile{
			SourcePath: sp,
			IsPartial:  false,
		}
	}

	// Create a single profile in the chain with content directory structure
	testProfile := &domainprofile.Profile{
		Name:          "test/profile",
		TemplateFiles: templateFiles,
		ContentConfig: domainprofile.ContentConfig{
			Root: "content",
		},
	}

	return profile.NewMergedProfile([]*domainprofile.Profile{testProfile}, fs, profilesBasePath)
}
