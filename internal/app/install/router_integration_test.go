package install_test

import (
	"encoding/json"
	"testing"

	"github.com/spf13/afero"

	"github.com/simensen/weftlo/internal/app/install"
	"github.com/simensen/weftlo/internal/app/rendering"
	"github.com/simensen/weftlo/internal/app/routing"
	"github.com/simensen/weftlo/internal/domain/manifest"
	domainprofile "github.com/simensen/weftlo/internal/domain/profile"
	"github.com/simensen/weftlo/internal/domain/template"
	"github.com/simensen/weftlo/internal/testutil"
)

// =============================================================================
// Task Group 2: Install Service Router Integration Tests
// =============================================================================

// Test 1: Install() correctly routes files using Router.Route()
func TestService_InstallCorrectlyRoutesFilesUsingRouter(t *testing.T) {
	t.Parallel()
	// Setup in-memory filesystem with template in claude directory
	profilesBasePath := "/home/user/.weftlo/profiles"
	templateContent := "# Claude Settings"
	projectDir := "/project"

	memFs := testutil.CreateTestFilesystem(t, map[string]string{
		profilesBasePath + "/vendor/test-profile/content/claude/settings.md.tmpl": templateContent,
	})

	// Create project directory
	if err := memFs.MkdirAll(projectDir, 0755); err != nil {
		t.Fatalf("failed to create project directory: %v", err)
	}

	// Create a profile with template in claude subdirectory
	profile := &domainprofile.Profile{
		Name: "vendor/test-profile",
		TemplateFiles: []domainprofile.TemplateFile{
			{SourcePath: "claude/settings.md.tmpl", IsPartial: false},
		},
	}

	// Create merged profile with content subdirectory
	mergedProfile := testutil.CreateTestMergedProfileWithContentDir(t, memFs, profilesBasePath, profile, "content")

	// Create dependencies
	engine := template.NewTemplateEngineWithFs(memFs)
	pipeline := rendering.NewRenderingPipeline(engine)
	manifestGenerator := manifest.NewManifestGenerator()

	// Create service (without PathMapper)
	service := install.NewService(memFs, pipeline, manifestGenerator)

	// Create Router with target configuration
	contentConfig := domainprofile.ContentConfig{
		Targets: map[string]string{
			"claude": ".claude",
		},
	}
	router, err := routing.NewContentRouter(contentConfig, nil, nil)
	if err != nil {
		t.Fatalf("failed to create router: %v", err)
	}

	// Execute install with Router
	opts := install.InstallOptions{
		DryRun: false,
	}

	result, err := service.Install(mergedProfile, projectDir, router, opts)
	if err != nil {
		t.Fatalf("install failed: %v", err)
	}

	// Verify file was installed to routed path
	if result.FilesProcessed != 1 {
		t.Errorf("expected 1 file processed, got %d", result.FilesProcessed)
	}

	// Verify file exists at correct routed location (.claude/settings.md)
	exists, err := afero.Exists(memFs, projectDir+"/.claude/settings.md")
	if err != nil {
		t.Fatalf("failed to check file existence: %v", err)
	}
	if !exists {
		t.Errorf("expected file at .claude/settings.md, but it does not exist")
	}

	// Verify file content
	content, err := afero.ReadFile(memFs, projectDir+"/.claude/settings.md")
	if err != nil {
		t.Fatalf("failed to read installed file: %v", err)
	}
	if string(content) != templateContent {
		t.Errorf("expected content '%s', got '%s'", templateContent, string(content))
	}
}

// Test 2: Suppressed files are skipped (no error, no manifest entry)
func TestService_SuppressedFilesAreSkipped(t *testing.T) {
	t.Parallel()
	// Setup in-memory filesystem with templates in two directories
	profilesBasePath := "/home/user/.weftlo/profiles"
	projectDir := "/project"

	memFs := testutil.CreateTestFilesystem(t, map[string]string{
		profilesBasePath + "/vendor/test-profile/content/claude/settings.md.tmpl": "Claude content",
		profilesBasePath + "/vendor/test-profile/content/weftlo/config.md.tmpl":   "Agent config content",
	})

	// Create project directory
	if err := memFs.MkdirAll(projectDir, 0755); err != nil {
		t.Fatalf("failed to create project directory: %v", err)
	}

	// Create a profile with templates in both directories
	profile := &domainprofile.Profile{
		Name: "vendor/test-profile",
		TemplateFiles: []domainprofile.TemplateFile{
			{SourcePath: "claude/settings.md.tmpl", IsPartial: false},
			{SourcePath: "weftlo/config.md.tmpl", IsPartial: false},
		},
	}

	// Create merged profile
	mergedProfile := testutil.CreateTestMergedProfileWithContentDir(t, memFs, profilesBasePath, profile, "content")

	// Create dependencies
	engine := template.NewTemplateEngineWithFs(memFs)
	pipeline := rendering.NewRenderingPipeline(engine)
	manifestGenerator := manifest.NewManifestGenerator()

	// Create service
	service := install.NewService(memFs, pipeline, manifestGenerator)

	// Create Router with suppression for claude target (nil pointer = suppressed)
	contentConfig := domainprofile.ContentConfig{
		Targets: map[string]string{
			"claude": ".claude",
			"weftlo": "weftlo",
		},
	}
	overrides := map[string]*string{
		"claude": nil, // Suppress claude target
	}
	router, err := routing.NewContentRouter(contentConfig, overrides, nil)
	if err != nil {
		t.Fatalf("failed to create router: %v", err)
	}

	// Execute install
	opts := install.InstallOptions{
		DryRun: false,
	}

	result, err := service.Install(mergedProfile, projectDir, router, opts)
	if err != nil {
		t.Fatalf("install failed: %v", err)
	}

	// Verify only 1 file was processed (claude file was suppressed)
	if result.FilesProcessed != 1 {
		t.Errorf("expected 1 file processed (suppressed file skipped), got %d", result.FilesProcessed)
	}

	// Verify suppressed file does NOT exist
	exists, err := afero.Exists(memFs, projectDir+"/.claude/settings.md")
	if err != nil {
		t.Fatalf("failed to check file existence: %v", err)
	}
	if exists {
		t.Errorf("expected suppressed file to NOT exist, but it does")
	}

	// Verify non-suppressed file DOES exist
	exists, err = afero.Exists(memFs, projectDir+"/weftlo/config.md")
	if err != nil {
		t.Fatalf("failed to check file existence: %v", err)
	}
	if !exists {
		t.Errorf("expected non-suppressed file to exist")
	}

	// Verify manifest does NOT contain suppressed file
	manifestPath := projectDir + "/.weftlo.manifest.json"
	manifestContent, err := afero.ReadFile(memFs, manifestPath)
	if err != nil {
		t.Fatalf("failed to read manifest: %v", err)
	}

	var m manifest.Manifest
	if err := json.Unmarshal(manifestContent, &m); err != nil {
		t.Fatalf("failed to parse manifest: %v", err)
	}

	// Check manifest entries
	if _, exists := m.Files[".claude/settings.md"]; exists {
		t.Errorf("expected suppressed file to NOT be in manifest, but it is")
	}

	if _, exists := m.Files["weftlo/config.md"]; !exists {
		t.Errorf("expected non-suppressed file to be in manifest, but it is not")
	}
}

// Test 3: TargetCategory is populated from RoutedFile
func TestService_TargetCategoryPopulatedFromRoutedFile(t *testing.T) {
	t.Parallel()
	// Setup in-memory filesystem with template in claude directory
	profilesBasePath := "/home/user/.weftlo/profiles"
	templateContent := "# Claude Settings"
	projectDir := "/project"

	memFs := testutil.CreateTestFilesystem(t, map[string]string{
		profilesBasePath + "/vendor/test-profile/content/claude/settings.md.tmpl": templateContent,
	})

	// Create project directory
	if err := memFs.MkdirAll(projectDir, 0755); err != nil {
		t.Fatalf("failed to create project directory: %v", err)
	}

	// Create a profile with template in claude subdirectory
	profile := &domainprofile.Profile{
		Name: "vendor/test-profile",
		TemplateFiles: []domainprofile.TemplateFile{
			{SourcePath: "claude/settings.md.tmpl", IsPartial: false},
		},
	}

	// Create merged profile with content subdirectory
	mergedProfile := testutil.CreateTestMergedProfileWithContentDir(t, memFs, profilesBasePath, profile, "content")

	// Create dependencies
	engine := template.NewTemplateEngineWithFs(memFs)
	pipeline := rendering.NewRenderingPipeline(engine)
	manifestGenerator := manifest.NewManifestGenerator()

	// Create service
	service := install.NewService(memFs, pipeline, manifestGenerator)

	// Create Router with target configuration
	contentConfig := domainprofile.ContentConfig{
		Targets: map[string]string{
			"claude": ".claude",
		},
	}
	router, err := routing.NewContentRouter(contentConfig, nil, nil)
	if err != nil {
		t.Fatalf("failed to create router: %v", err)
	}

	// Execute install
	opts := install.InstallOptions{
		DryRun: false,
	}

	_, err = service.Install(mergedProfile, projectDir, router, opts)
	if err != nil {
		t.Fatalf("install failed: %v", err)
	}

	// Read manifest and verify TargetCategory is populated
	manifestPath := projectDir + "/.weftlo.manifest.json"
	manifestContent, err := afero.ReadFile(memFs, manifestPath)
	if err != nil {
		t.Fatalf("failed to read manifest: %v", err)
	}

	var m manifest.Manifest
	if err := json.Unmarshal(manifestContent, &m); err != nil {
		t.Fatalf("failed to parse manifest: %v", err)
	}

	// Verify manifest entry has correct TargetCategory
	fileEntry, exists := m.Files[".claude/settings.md"]
	if !exists {
		t.Fatalf("expected settings.md in manifest files, got keys: %v", getMapKeys(m.Files))
	}

	// TargetCategory should be "claude" (the source directory category)
	if fileEntry.TargetCategory != "claude" {
		t.Errorf("expected TargetCategory 'claude', got '%s'", fileEntry.TargetCategory)
	}
}

// Test 4: Multiple files with different categories are routed correctly
func TestService_MultipleFilesWithDifferentCategories(t *testing.T) {
	t.Parallel()
	// Setup in-memory filesystem with templates in different directories
	profilesBasePath := "/home/user/.weftlo/profiles"
	projectDir := "/project"

	memFs := testutil.CreateTestFilesystem(t, map[string]string{
		profilesBasePath + "/vendor/test-profile/content/claude/settings.md.tmpl": "Claude content",
		profilesBasePath + "/vendor/test-profile/content/weftlo/config.md.tmpl":   "Agent config content",
		profilesBasePath + "/vendor/test-profile/content/cursor/rules.md.tmpl":    "Cursor rules",
	})

	// Create project directory
	if err := memFs.MkdirAll(projectDir, 0755); err != nil {
		t.Fatalf("failed to create project directory: %v", err)
	}

	// Create a profile with templates in multiple directories
	profile := &domainprofile.Profile{
		Name: "vendor/test-profile",
		TemplateFiles: []domainprofile.TemplateFile{
			{SourcePath: "claude/settings.md.tmpl", IsPartial: false},
			{SourcePath: "weftlo/config.md.tmpl", IsPartial: false},
			{SourcePath: "cursor/rules.md.tmpl", IsPartial: false},
		},
	}

	// Create merged profile
	mergedProfile := testutil.CreateTestMergedProfileWithContentDir(t, memFs, profilesBasePath, profile, "content")

	// Create dependencies
	engine := template.NewTemplateEngineWithFs(memFs)
	pipeline := rendering.NewRenderingPipeline(engine)
	manifestGenerator := manifest.NewManifestGenerator()

	// Create service
	service := install.NewService(memFs, pipeline, manifestGenerator)

	// Create Router with multiple target configurations
	contentConfig := domainprofile.ContentConfig{
		Targets: map[string]string{
			"claude": ".claude",
			"weftlo": "weftlo",
			"cursor": ".cursor",
		},
	}
	router, err := routing.NewContentRouter(contentConfig, nil, nil)
	if err != nil {
		t.Fatalf("failed to create router: %v", err)
	}

	// Execute install
	opts := install.InstallOptions{
		DryRun: false,
	}

	result, err := service.Install(mergedProfile, projectDir, router, opts)
	if err != nil {
		t.Fatalf("install failed: %v", err)
	}

	// Verify all 3 files were processed
	if result.FilesProcessed != 3 {
		t.Errorf("expected 3 files processed, got %d", result.FilesProcessed)
	}

	// Verify each file exists at correct routed location
	expectedPaths := map[string]string{
		".claude/settings.md": "Claude content",
		"weftlo/config.md":    "Agent config content",
		".cursor/rules.md":    "Cursor rules",
	}

	for path, expectedContent := range expectedPaths {
		fullPath := projectDir + "/" + path
		exists, err := afero.Exists(memFs, fullPath)
		if err != nil {
			t.Fatalf("failed to check file existence for %s: %v", path, err)
		}
		if !exists {
			t.Errorf("expected file at %s, but it does not exist", path)
			continue
		}

		content, err := afero.ReadFile(memFs, fullPath)
		if err != nil {
			t.Fatalf("failed to read file %s: %v", path, err)
		}
		if string(content) != expectedContent {
			t.Errorf("for %s: expected content '%s', got '%s'", path, expectedContent, string(content))
		}
	}

	// Verify manifest has correct TargetCategory for each file
	manifestPath := projectDir + "/.weftlo.manifest.json"
	manifestContent, err := afero.ReadFile(memFs, manifestPath)
	if err != nil {
		t.Fatalf("failed to read manifest: %v", err)
	}

	var m manifest.Manifest
	if err := json.Unmarshal(manifestContent, &m); err != nil {
		t.Fatalf("failed to parse manifest: %v", err)
	}

	expectedCategories := map[string]string{
		".claude/settings.md": "claude",
		"weftlo/config.md":    "weftlo",
		".cursor/rules.md":    "cursor",
	}

	for path, expectedCategory := range expectedCategories {
		entry, exists := m.Files[path]
		if !exists {
			t.Errorf("expected %s in manifest, but not found", path)
			continue
		}
		if entry.TargetCategory != expectedCategory {
			t.Errorf("for %s: expected TargetCategory '%s', got '%s'", path, expectedCategory, entry.TargetCategory)
		}
	}
}

// Helper function to get map keys for debugging
func getMapKeys(m map[string]manifest.ManifestFile) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
