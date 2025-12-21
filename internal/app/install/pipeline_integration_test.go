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
// Task Group 5.5: Strategic Integration Tests
// Focus on Router integration with install/update pipelines
// =============================================================================

// Test 1: All files suppressed results in empty manifest
// Edge case: When all target categories are suppressed via overrides
func TestService_AllFilesSuppressedResultsInEmptyInstall(t *testing.T) {
	t.Parallel()
	profilesBasePath := "/home/user/.weftlo/profiles"
	projectDir := "/project"

	memFs := testutil.CreateTestFilesystem(t, map[string]string{
		profilesBasePath + "/vendor/test-profile/content/claude/settings.md.tmpl": "Claude settings",
		profilesBasePath + "/vendor/test-profile/content/cursor/rules.md.tmpl":    "Cursor rules",
		profilesBasePath + "/vendor/test-profile/content/weftlo/config.md.tmpl":   "Agent config",
	})

	if err := memFs.MkdirAll(projectDir, 0755); err != nil {
		t.Fatalf("failed to create project directory: %v", err)
	}

	profile := &domainprofile.Profile{
		Name: "vendor/test-profile",
		TemplateFiles: []domainprofile.TemplateFile{
			{SourcePath: "claude/settings.md.tmpl", IsPartial: false},
			{SourcePath: "cursor/rules.md.tmpl", IsPartial: false},
			{SourcePath: "weftlo/config.md.tmpl", IsPartial: false},
		},
	}

	mergedProfile := testutil.CreateTestMergedProfileWithContentDir(t, memFs, profilesBasePath, profile, "content")

	engine := template.NewTemplateEngineWithFs(memFs)
	pipeline := rendering.NewRenderingPipeline(engine)
	manifestGenerator := manifest.NewManifestGenerator()
	service := install.NewService(memFs, pipeline, manifestGenerator)

	// Create Router with ALL categories suppressed
	contentConfig := domainprofile.ContentConfig{
		Targets: map[string]string{
			"claude": ".claude",
			"cursor": ".cursor",
			"weftlo": "weftlo",
		},
	}
	overrides := map[string]*string{
		"claude": nil, // Suppress
		"cursor": nil, // Suppress
		"weftlo": nil, // Suppress
	}
	router, err := routing.NewContentRouter(contentConfig, overrides, nil)
	if err != nil {
		t.Fatalf("failed to create router: %v", err)
	}

	opts := install.InstallOptions{DryRun: false}
	result, err := service.Install(mergedProfile, projectDir, router, opts)
	if err != nil {
		t.Fatalf("install failed: %v", err)
	}

	// Verify no files were processed (all suppressed)
	if result.FilesProcessed != 0 {
		t.Errorf("expected 0 files processed when all suppressed, got %d", result.FilesProcessed)
	}

	// Verify manifest exists but has no files
	manifestPath := projectDir + "/.weftlo.manifest.json"
	manifestContent, err := afero.ReadFile(memFs, manifestPath)
	if err != nil {
		t.Fatalf("failed to read manifest: %v", err)
	}

	var m manifest.Manifest
	if err := json.Unmarshal(manifestContent, &m); err != nil {
		t.Fatalf("failed to parse manifest: %v", err)
	}

	if len(m.Files) != 0 {
		t.Errorf("expected 0 files in manifest when all suppressed, got %d", len(m.Files))
	}
}

// Test 2: Unmapped directory uses default target with empty TargetCategory
// When a source path doesn't match any target mapping, it uses default_target
// and the TargetCategory is set to empty string (by design)
func TestService_UnmappedDirectoryUsesDefaultTarget(t *testing.T) {
	t.Parallel()
	profilesBasePath := "/home/user/.weftlo/profiles"
	projectDir := "/project"

	// File in unmapped directory (no specific target category)
	memFs := testutil.CreateTestFilesystem(t, map[string]string{
		profilesBasePath + "/vendor/test-profile/content/misc/readme.md.tmpl": "Misc readme",
	})

	if err := memFs.MkdirAll(projectDir, 0755); err != nil {
		t.Fatalf("failed to create project directory: %v", err)
	}

	profile := &domainprofile.Profile{
		Name: "vendor/test-profile",
		TemplateFiles: []domainprofile.TemplateFile{
			{SourcePath: "misc/readme.md.tmpl", IsPartial: false},
		},
	}

	mergedProfile := testutil.CreateTestMergedProfileWithContentDir(t, memFs, profilesBasePath, profile, "content")

	engine := template.NewTemplateEngineWithFs(memFs)
	pipeline := rendering.NewRenderingPipeline(engine)
	manifestGenerator := manifest.NewManifestGenerator()
	service := install.NewService(memFs, pipeline, manifestGenerator)

	// Create Router with only default target (no explicit target for "misc")
	contentConfig := domainprofile.ContentConfig{
		Targets: map[string]string{
			"claude": ".claude", // Only claude is mapped
		},
		DefaultTarget: "weftlo",
	}
	router, err := routing.NewContentRouter(contentConfig, nil, nil)
	if err != nil {
		t.Fatalf("failed to create router: %v", err)
	}

	opts := install.InstallOptions{DryRun: false}
	result, err := service.Install(mergedProfile, projectDir, router, opts)
	if err != nil {
		t.Fatalf("install failed: %v", err)
	}

	if result.FilesProcessed != 1 {
		t.Errorf("expected 1 file processed, got %d", result.FilesProcessed)
	}

	// Verify file was installed to default target path (preserving full source structure)
	exists, err := afero.Exists(memFs, projectDir+"/weftlo/misc/readme.md")
	if err != nil {
		t.Fatalf("failed to check file existence: %v", err)
	}
	if !exists {
		t.Error("expected file to be installed to default target path weftlo/misc/readme.md")
	}

	// Verify manifest entry exists
	manifestPath := projectDir + "/.weftlo.manifest.json"
	manifestContent, err := afero.ReadFile(memFs, manifestPath)
	if err != nil {
		t.Fatalf("failed to read manifest: %v", err)
	}

	var m manifest.Manifest
	if err := json.Unmarshal(manifestContent, &m); err != nil {
		t.Fatalf("failed to parse manifest: %v", err)
	}

	fileEntry, exists := m.Files["weftlo/misc/readme.md"]
	if !exists {
		t.Fatal("expected weftlo/misc/readme.md in manifest")
	}

	// TargetCategory should be empty when using default target (by design)
	// This is because no explicit target mapping matched
	if fileEntry.TargetCategory != "" {
		t.Errorf("expected empty TargetCategory when using default target, got '%s'", fileEntry.TargetCategory)
	}
}

// Test 3: Mixed suppressed and non-suppressed files in same install
func TestService_MixedSuppressedAndNonSuppressedFiles(t *testing.T) {
	t.Parallel()
	profilesBasePath := "/home/user/.weftlo/profiles"
	projectDir := "/project"

	memFs := testutil.CreateTestFilesystem(t, map[string]string{
		profilesBasePath + "/vendor/test-profile/content/claude/settings.md.tmpl":  "Claude settings",
		profilesBasePath + "/vendor/test-profile/content/cursor/rules.md.tmpl":     "Cursor rules",
		profilesBasePath + "/vendor/test-profile/content/weftlo/standards.md.tmpl": "Standards",
	})

	if err := memFs.MkdirAll(projectDir, 0755); err != nil {
		t.Fatalf("failed to create project directory: %v", err)
	}

	profile := &domainprofile.Profile{
		Name: "vendor/test-profile",
		TemplateFiles: []domainprofile.TemplateFile{
			{SourcePath: "claude/settings.md.tmpl", IsPartial: false},
			{SourcePath: "cursor/rules.md.tmpl", IsPartial: false},
			{SourcePath: "weftlo/standards.md.tmpl", IsPartial: false},
		},
	}

	mergedProfile := testutil.CreateTestMergedProfileWithContentDir(t, memFs, profilesBasePath, profile, "content")

	engine := template.NewTemplateEngineWithFs(memFs)
	pipeline := rendering.NewRenderingPipeline(engine)
	manifestGenerator := manifest.NewManifestGenerator()
	service := install.NewService(memFs, pipeline, manifestGenerator)

	// Suppress only cursor, keep claude and weftlo
	contentConfig := domainprofile.ContentConfig{
		Targets: map[string]string{
			"claude": ".claude",
			"cursor": ".cursor",
			"weftlo": "weftlo",
		},
	}
	overrides := map[string]*string{
		"cursor": nil, // Only suppress cursor
	}
	router, err := routing.NewContentRouter(contentConfig, overrides, nil)
	if err != nil {
		t.Fatalf("failed to create router: %v", err)
	}

	opts := install.InstallOptions{DryRun: false}
	result, err := service.Install(mergedProfile, projectDir, router, opts)
	if err != nil {
		t.Fatalf("install failed: %v", err)
	}

	// Only 2 files should be processed (cursor suppressed)
	if result.FilesProcessed != 2 {
		t.Errorf("expected 2 files processed, got %d", result.FilesProcessed)
	}

	// Verify claude file exists
	exists, err := afero.Exists(memFs, projectDir+"/.claude/settings.md")
	if err != nil {
		t.Fatalf("failed to check claude file: %v", err)
	}
	if !exists {
		t.Error("expected claude/settings.md to be installed")
	}

	// Verify weftlo file exists
	exists, err = afero.Exists(memFs, projectDir+"/weftlo/standards.md")
	if err != nil {
		t.Fatalf("failed to check weftlo file: %v", err)
	}
	if !exists {
		t.Error("expected weftlo/standards.md to be installed")
	}

	// Verify cursor file does NOT exist (suppressed)
	exists, err = afero.Exists(memFs, projectDir+"/.cursor/rules.md")
	if err != nil {
		t.Fatalf("failed to check cursor file: %v", err)
	}
	if exists {
		t.Error("expected cursor/rules.md to NOT be installed (suppressed)")
	}

	// Verify manifest only contains non-suppressed files
	manifestPath := projectDir + "/.weftlo.manifest.json"
	manifestContent, err := afero.ReadFile(memFs, manifestPath)
	if err != nil {
		t.Fatalf("failed to read manifest: %v", err)
	}

	var m manifest.Manifest
	if err := json.Unmarshal(manifestContent, &m); err != nil {
		t.Fatalf("failed to parse manifest: %v", err)
	}

	if len(m.Files) != 2 {
		t.Errorf("expected 2 files in manifest, got %d", len(m.Files))
	}

	if _, exists := m.Files[".cursor/rules.md"]; exists {
		t.Error("expected suppressed file to NOT be in manifest")
	}
}

// Test 4: Target override redirects file to different location
func TestService_TargetOverrideRedirectsFile(t *testing.T) {
	t.Parallel()
	profilesBasePath := "/home/user/.weftlo/profiles"
	projectDir := "/project"

	memFs := testutil.CreateTestFilesystem(t, map[string]string{
		profilesBasePath + "/vendor/test-profile/content/claude/settings.md.tmpl": "Claude settings",
	})

	if err := memFs.MkdirAll(projectDir, 0755); err != nil {
		t.Fatalf("failed to create project directory: %v", err)
	}

	profile := &domainprofile.Profile{
		Name: "vendor/test-profile",
		TemplateFiles: []domainprofile.TemplateFile{
			{SourcePath: "claude/settings.md.tmpl", IsPartial: false},
		},
	}

	mergedProfile := testutil.CreateTestMergedProfileWithContentDir(t, memFs, profilesBasePath, profile, "content")

	engine := template.NewTemplateEngineWithFs(memFs)
	pipeline := rendering.NewRenderingPipeline(engine)
	manifestGenerator := manifest.NewManifestGenerator()
	service := install.NewService(memFs, pipeline, manifestGenerator)

	// Profile defines claude -> .claude, but override redirects to custom/ai
	contentConfig := domainprofile.ContentConfig{
		Targets: map[string]string{
			"claude": ".claude",
		},
	}
	customPath := "custom/ai"
	overrides := map[string]*string{
		"claude": &customPath, // Redirect claude to custom/ai
	}
	router, err := routing.NewContentRouter(contentConfig, overrides, nil)
	if err != nil {
		t.Fatalf("failed to create router: %v", err)
	}

	opts := install.InstallOptions{DryRun: false}
	result, err := service.Install(mergedProfile, projectDir, router, opts)
	if err != nil {
		t.Fatalf("install failed: %v", err)
	}

	if result.FilesProcessed != 1 {
		t.Errorf("expected 1 file processed, got %d", result.FilesProcessed)
	}

	// Verify file is at overridden location
	exists, err := afero.Exists(memFs, projectDir+"/custom/ai/settings.md")
	if err != nil {
		t.Fatalf("failed to check file: %v", err)
	}
	if !exists {
		t.Error("expected file at overridden path custom/ai/settings.md")
	}

	// Verify file is NOT at original profile target
	exists, err = afero.Exists(memFs, projectDir+"/.claude/settings.md")
	if err != nil {
		t.Fatalf("failed to check original path: %v", err)
	}
	if exists {
		t.Error("expected file NOT at original path .claude/settings.md")
	}

	// Verify manifest has correct path and category
	manifestPath := projectDir + "/.weftlo.manifest.json"
	manifestContent, err := afero.ReadFile(memFs, manifestPath)
	if err != nil {
		t.Fatalf("failed to read manifest: %v", err)
	}

	var m manifest.Manifest
	if err := json.Unmarshal(manifestContent, &m); err != nil {
		t.Fatalf("failed to parse manifest: %v", err)
	}

	fileEntry, exists := m.Files["custom/ai/settings.md"]
	if !exists {
		t.Fatal("expected custom/ai/settings.md in manifest")
	}

	// TargetCategory should still be "claude" (the source directory category)
	if fileEntry.TargetCategory != "claude" {
		t.Errorf("expected TargetCategory 'claude', got '%s'", fileEntry.TargetCategory)
	}
}

// Test 5: Nested files within target category preserve subdirectory structure
func TestService_NestedFilesPreserveSubdirectoryStructure(t *testing.T) {
	t.Parallel()
	profilesBasePath := "/home/user/.weftlo/profiles"
	projectDir := "/project"

	memFs := testutil.CreateTestFilesystem(t, map[string]string{
		profilesBasePath + "/vendor/test-profile/content/claude/commands/build.json.tmpl": "build command",
		profilesBasePath + "/vendor/test-profile/content/claude/commands/test.json.tmpl":  "test command",
		profilesBasePath + "/vendor/test-profile/content/claude/settings/config.md.tmpl":  "settings config",
	})

	if err := memFs.MkdirAll(projectDir, 0755); err != nil {
		t.Fatalf("failed to create project directory: %v", err)
	}

	profile := &domainprofile.Profile{
		Name: "vendor/test-profile",
		TemplateFiles: []domainprofile.TemplateFile{
			{SourcePath: "claude/commands/build.json.tmpl", IsPartial: false},
			{SourcePath: "claude/commands/test.json.tmpl", IsPartial: false},
			{SourcePath: "claude/settings/config.md.tmpl", IsPartial: false},
		},
	}

	mergedProfile := testutil.CreateTestMergedProfileWithContentDir(t, memFs, profilesBasePath, profile, "content")

	engine := template.NewTemplateEngineWithFs(memFs)
	pipeline := rendering.NewRenderingPipeline(engine)
	manifestGenerator := manifest.NewManifestGenerator()
	service := install.NewService(memFs, pipeline, manifestGenerator)

	contentConfig := domainprofile.ContentConfig{
		Targets: map[string]string{
			"claude": ".claude",
		},
	}
	router, err := routing.NewContentRouter(contentConfig, nil, nil)
	if err != nil {
		t.Fatalf("failed to create router: %v", err)
	}

	opts := install.InstallOptions{DryRun: false}
	result, err := service.Install(mergedProfile, projectDir, router, opts)
	if err != nil {
		t.Fatalf("install failed: %v", err)
	}

	if result.FilesProcessed != 3 {
		t.Errorf("expected 3 files processed, got %d", result.FilesProcessed)
	}

	// Verify all nested files exist with preserved structure
	expectedPaths := []string{
		projectDir + "/.claude/commands/build.json",
		projectDir + "/.claude/commands/test.json",
		projectDir + "/.claude/settings/config.md",
	}

	for _, path := range expectedPaths {
		exists, err := afero.Exists(memFs, path)
		if err != nil {
			t.Fatalf("failed to check %s: %v", path, err)
		}
		if !exists {
			t.Errorf("expected file at %s", path)
		}
	}

	// Verify manifest entries all have "claude" as TargetCategory
	manifestPath := projectDir + "/.weftlo.manifest.json"
	manifestContent, err := afero.ReadFile(memFs, manifestPath)
	if err != nil {
		t.Fatalf("failed to read manifest: %v", err)
	}

	var m manifest.Manifest
	if err := json.Unmarshal(manifestContent, &m); err != nil {
		t.Fatalf("failed to parse manifest: %v", err)
	}

	for path, entry := range m.Files {
		if entry.TargetCategory != "claude" {
			t.Errorf("expected TargetCategory 'claude' for %s, got '%s'", path, entry.TargetCategory)
		}
	}
}
