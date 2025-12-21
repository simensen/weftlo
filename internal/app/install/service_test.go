package install_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/spf13/afero"

	"github.com/simensen/weftlo/internal/app/install"
	"github.com/simensen/weftlo/internal/app/rendering"
	"github.com/simensen/weftlo/internal/app/routing"
	domainconfig "github.com/simensen/weftlo/internal/domain/config"
	domainignore "github.com/simensen/weftlo/internal/domain/ignore"
	"github.com/simensen/weftlo/internal/domain/manifest"
	domainprofile "github.com/simensen/weftlo/internal/domain/profile"
	"github.com/simensen/weftlo/internal/domain/template"
	"github.com/simensen/weftlo/internal/testutil"
)

// =============================================================================
// Task Group 3: Install Command Integration Tests
// =============================================================================

// Test 1: InstallOptions DryRun flag defaults to false
func TestInstallOptions_DryRunDefaultsToFalse(t *testing.T) {
	t.Parallel()
	opts := install.DefaultInstallOptions()

	if opts.DryRun != false {
		t.Errorf("expected DryRun to default to false, got %v", opts.DryRun)
	}
}

// Test 2: Dry-run mode produces output without writing files
func TestService_DryRunProducesOutputWithoutWritingFiles(t *testing.T) {
	t.Parallel()
	// Setup in-memory filesystem with template
	profilesBasePath := "/home/user/.weftlo/profiles"
	templateContent := "# Test Template\nGenerated content"
	memFs := testutil.CreateTestFilesystem(t, map[string]string{
		profilesBasePath + "/vendor/test-profile/content/README.md.tmpl": templateContent,
	})

	// Create a profile using helper
	profile := testutil.CreateTestProfile(t, "vendor/test-profile", "README.md.tmpl")

	// Create merged profile with content directory
	mergedProfile := testutil.CreateTestMergedProfileWithContentDir(t, memFs, profilesBasePath, profile, "content")

	// Create dependencies
	engine := template.NewTemplateEngineWithFs(memFs)
	pipeline := rendering.NewRenderingPipeline(engine)
	manifestGenerator := manifest.NewManifestGenerator()

	// Create service
	service := install.NewService(memFs, pipeline, manifestGenerator)

	// Create router with default target for weftlo
	contentConfig := domainprofile.ContentConfig{
		DefaultTarget: "weftlo",
	}
	router, err := routing.NewContentRouter(contentConfig, nil, nil)
	if err != nil {
		t.Fatalf("failed to create router: %v", err)
	}

	// Capture output
	var outputBuffer bytes.Buffer
	opts := install.InstallOptions{
		DryRun: true,
		Stdout: &outputBuffer,
	}

	// Execute install with dry-run
	projectDir := "/project"
	err = memFs.MkdirAll(projectDir, 0755)
	if err != nil {
		t.Fatalf("failed to create project directory: %v", err)
	}

	result, err := service.Install(mergedProfile, projectDir, router, opts)
	if err != nil {
		t.Fatalf("dry-run install failed: %v", err)
	}

	// Verify dry-run result
	if !result.DryRun {
		t.Errorf("expected result.DryRun to be true")
	}

	// Verify output was produced
	output := outputBuffer.String()
	if output == "" {
		t.Errorf("expected dry-run to produce output, got empty string")
	}

	// Verify output contains expected content
	if !strings.Contains(output, "would create") {
		t.Errorf("expected output to contain 'would create', got: %s", output)
	}

	// Verify NO files were written to project directory
	exists, err := afero.Exists(memFs, "/project/weftlo/README.md")
	if err != nil {
		t.Fatalf("failed to check file existence: %v", err)
	}
	if exists {
		t.Errorf("expected no files to be written in dry-run mode, but file exists")
	}

	// Verify manifest was NOT written
	manifestExists, err := afero.Exists(memFs, "/project/.weftlo.manifest.json")
	if err != nil {
		t.Fatalf("failed to check manifest existence: %v", err)
	}
	if manifestExists {
		t.Errorf("expected manifest not to be written in dry-run mode")
	}
}

// Test 3: Dry-run mode reads existing files for diff generation
func TestService_DryRunReadsExistingFilesForDiff(t *testing.T) {
	t.Parallel()
	// Setup in-memory filesystem with template and existing file
	profilesBasePath := "/home/user/.weftlo/profiles"
	templateContent := "# New Content\nThis is new"
	existingContent := "# Old Content\nThis is old"
	projectDir := "/project"

	memFs := testutil.CreateTestFilesystem(t, map[string]string{
		profilesBasePath + "/vendor/test-profile/content/config.md.tmpl": templateContent,
		projectDir + "/weftlo/config.md":                                 existingContent,
	})

	// Create a profile using helper
	profile := testutil.CreateTestProfile(t, "vendor/test-profile", "config.md.tmpl")

	// Create merged profile with content directory
	mergedProfile := testutil.CreateTestMergedProfileWithContentDir(t, memFs, profilesBasePath, profile, "content")

	// Create dependencies
	engine := template.NewTemplateEngineWithFs(memFs)
	pipeline := rendering.NewRenderingPipeline(engine)
	manifestGenerator := manifest.NewManifestGenerator()

	// Create service
	service := install.NewService(memFs, pipeline, manifestGenerator)

	// Create router with default target for weftlo
	contentConfig := domainprofile.ContentConfig{
		DefaultTarget: "weftlo",
	}
	router, err := routing.NewContentRouter(contentConfig, nil, nil)
	if err != nil {
		t.Fatalf("failed to create router: %v", err)
	}

	// Capture output
	var outputBuffer bytes.Buffer
	opts := install.InstallOptions{
		DryRun: true,
		Stdout: &outputBuffer,
	}

	// Execute install with dry-run
	result, err := service.Install(mergedProfile, projectDir, router, opts)
	if err != nil {
		t.Fatalf("dry-run install failed: %v", err)
	}

	if !result.DryRun {
		t.Errorf("expected result.DryRun to be true")
	}

	// Verify output contains diff markers (showing existing file was read)
	output := outputBuffer.String()
	if !strings.Contains(output, "---") || !strings.Contains(output, "+++") {
		t.Errorf("expected output to contain diff markers when existing file differs, got: %s", output)
	}

	// Verify existing file was NOT modified
	content, err := afero.ReadFile(memFs, projectDir+"/weftlo/config.md")
	if err != nil {
		t.Fatalf("failed to read existing file: %v", err)
	}
	if string(content) != existingContent {
		t.Errorf("existing file content was modified in dry-run mode")
	}
}

// Test 4: Dry-run mode outputs to stdout (via provided writer)
func TestService_DryRunOutputsToProvidedWriter(t *testing.T) {
	t.Parallel()
	// Setup in-memory filesystem with template
	profilesBasePath := "/home/user/.weftlo/profiles"
	templateContent := "Test content"
	projectDir := "/project"

	memFs := testutil.CreateTestFilesystem(t, map[string]string{
		profilesBasePath + "/vendor/test-profile/content/file.md.tmpl": templateContent,
	})

	// Create project directory
	if err := memFs.MkdirAll(projectDir, 0755); err != nil {
		t.Fatalf("failed to create project directory: %v", err)
	}

	// Create a profile using helper
	profile := testutil.CreateTestProfile(t, "vendor/test-profile", "file.md.tmpl")

	// Create merged profile with content directory
	mergedProfile := testutil.CreateTestMergedProfileWithContentDir(t, memFs, profilesBasePath, profile, "content")

	// Create dependencies
	engine := template.NewTemplateEngineWithFs(memFs)
	pipeline := rendering.NewRenderingPipeline(engine)
	manifestGenerator := manifest.NewManifestGenerator()

	// Create service
	service := install.NewService(memFs, pipeline, manifestGenerator)

	// Create router with default target for weftlo
	contentConfig := domainprofile.ContentConfig{
		DefaultTarget: "weftlo",
	}
	router, err := routing.NewContentRouter(contentConfig, nil, nil)
	if err != nil {
		t.Fatalf("failed to create router: %v", err)
	}

	// Use a custom writer to verify output destination
	var customWriter bytes.Buffer
	opts := install.InstallOptions{
		DryRun: true,
		Stdout: &customWriter,
	}

	// Execute install with dry-run
	_, err = service.Install(mergedProfile, projectDir, router, opts)
	if err != nil {
		t.Fatalf("dry-run install failed: %v", err)
	}

	// Verify output was written to custom writer
	if customWriter.Len() == 0 {
		t.Errorf("expected output to be written to provided writer, but writer is empty")
	}

	// Verify output contains the target path
	output := customWriter.String()
	if !strings.Contains(output, "weftlo/file.md") {
		t.Errorf("expected output to contain target path 'weftlo/file.md', got: %s", output)
	}
}

// Test 5: Zero writes guarantee - no files, directories, or manifest written in dry-run
func TestService_ZeroWritesGuarantee(t *testing.T) {
	t.Parallel()
	// Setup in-memory filesystem with multiple templates
	profilesBasePath := "/home/user/.weftlo/profiles"
	projectDir := "/project"

	memFs := testutil.CreateTestFilesystem(t, map[string]string{
		profilesBasePath + "/vendor/test-profile/content/file1.md.tmpl":            "Content 1",
		profilesBasePath + "/vendor/test-profile/content/nested/file2.md.tmpl":     "Content 2",
		profilesBasePath + "/vendor/test-profile/content/nested/dir/file3.md.tmpl": "Content 3",
	})

	// Create project directory
	if err := memFs.MkdirAll(projectDir, 0755); err != nil {
		t.Fatalf("failed to create project directory: %v", err)
	}

	// Create a profile with multiple template files
	profile := &domainprofile.Profile{
		Name: "vendor/test-profile",
		TemplateFiles: []domainprofile.TemplateFile{
			{SourcePath: "file1.md.tmpl", IsPartial: false},
			{SourcePath: "nested/file2.md.tmpl", IsPartial: false},
			{SourcePath: "nested/dir/file3.md.tmpl", IsPartial: false},
		},
	}

	// Create merged profile with content directory
	mergedProfile := testutil.CreateTestMergedProfileWithContentDir(t, memFs, profilesBasePath, profile, "content")

	// Create dependencies
	engine := template.NewTemplateEngineWithFs(memFs)
	pipeline := rendering.NewRenderingPipeline(engine)
	manifestGenerator := manifest.NewManifestGenerator()

	// Create service
	service := install.NewService(memFs, pipeline, manifestGenerator)

	// Create router with default target for weftlo
	contentConfig := domainprofile.ContentConfig{
		DefaultTarget: "weftlo",
	}
	router, err := routing.NewContentRouter(contentConfig, nil, nil)
	if err != nil {
		t.Fatalf("failed to create router: %v", err)
	}

	// Execute install with dry-run
	var outputBuffer bytes.Buffer
	opts := install.InstallOptions{
		DryRun: true,
		Stdout: &outputBuffer,
	}

	result, err := service.Install(mergedProfile, projectDir, router, opts)
	if err != nil {
		t.Fatalf("dry-run install failed: %v", err)
	}

	// Verify result indicates dry-run
	if !result.DryRun {
		t.Errorf("expected result.DryRun to be true")
	}
	if result.FilesProcessed != 3 {
		t.Errorf("expected FilesProcessed to be 3, got %d", result.FilesProcessed)
	}

	// Verify NO files were created in project directory
	pathsToCheck := []string{
		"/project/weftlo/file1.md",
		"/project/weftlo/nested/file2.md",
		"/project/weftlo/nested/dir/file3.md",
		"/project/weftlo",
		"/project/weftlo/nested",
		"/project/weftlo/nested/dir",
		"/project/.weftlo.manifest.json",
	}

	for _, path := range pathsToCheck {
		exists, err := afero.Exists(memFs, path)
		if err != nil {
			t.Fatalf("failed to check existence of %s: %v", path, err)
		}
		if exists {
			t.Errorf("zero writes guarantee violated: %s exists after dry-run", path)
		}
	}
}

// Test 6: Normal install (non-dry-run) does write files
func TestService_NormalInstallWritesFiles(t *testing.T) {
	t.Parallel()
	// Setup in-memory filesystem with template
	profilesBasePath := "/home/user/.weftlo/profiles"
	templateContent := "# Generated File\nTest content"
	projectDir := "/project"

	memFs := testutil.CreateTestFilesystem(t, map[string]string{
		profilesBasePath + "/vendor/test-profile/content/output.md.tmpl": templateContent,
	})

	// Create project directory
	if err := memFs.MkdirAll(projectDir, 0755); err != nil {
		t.Fatalf("failed to create project directory: %v", err)
	}

	// Create a profile using helper
	profile := testutil.CreateTestProfile(t, "vendor/test-profile", "output.md.tmpl")

	// Create merged profile with content directory
	mergedProfile := testutil.CreateTestMergedProfileWithContentDir(t, memFs, profilesBasePath, profile, "content")

	// Create dependencies
	engine := template.NewTemplateEngineWithFs(memFs)
	pipeline := rendering.NewRenderingPipeline(engine)
	manifestGenerator := manifest.NewManifestGenerator()

	// Create service
	service := install.NewService(memFs, pipeline, manifestGenerator)

	// Create router with default target for weftlo
	contentConfig := domainprofile.ContentConfig{
		DefaultTarget: "weftlo",
	}
	router, err := routing.NewContentRouter(contentConfig, nil, nil)
	if err != nil {
		t.Fatalf("failed to create router: %v", err)
	}

	// Execute install WITHOUT dry-run
	opts := install.InstallOptions{
		DryRun: false,
	}

	result, err := service.Install(mergedProfile, projectDir, router, opts)
	if err != nil {
		t.Fatalf("install failed: %v", err)
	}

	// Verify result indicates not dry-run
	if result.DryRun {
		t.Errorf("expected result.DryRun to be false")
	}

	// Verify file WAS created
	exists, err := afero.Exists(memFs, "/project/weftlo/output.md")
	if err != nil {
		t.Fatalf("failed to check file existence: %v", err)
	}
	if !exists {
		t.Errorf("expected file to be created in normal install mode")
	}

	// Verify file content
	content, err := afero.ReadFile(memFs, "/project/weftlo/output.md")
	if err != nil {
		t.Fatalf("failed to read installed file: %v", err)
	}
	if string(content) != templateContent {
		t.Errorf("expected file content '%s', got '%s'", templateContent, string(content))
	}

	// Verify manifest WAS created
	manifestExists, err := afero.Exists(memFs, "/project/.weftlo.manifest.json")
	if err != nil {
		t.Fatalf("failed to check manifest existence: %v", err)
	}
	if !manifestExists {
		t.Errorf("expected manifest to be created in normal install mode")
	}
}

// =============================================================================
// Task Group 2: Source Content Preservation Tests
// =============================================================================

// Test: RenderedFile created with distinct SourceContent from OutputContent
func TestService_RenderedFileHasDistinctSourceContent(t *testing.T) {
	t.Parallel()
	// Setup in-memory filesystem with template using variables
	profilesBasePath := "/home/user/.weftlo/profiles"
	templateContent := "Hello {{ .Variables.name }}!"
	projectDir := "/project"

	memFs := testutil.CreateTestFilesystem(t, map[string]string{
		profilesBasePath + "/vendor/test-profile/content/greeting.md.tmpl": templateContent,
	})

	// Create project directory
	if err := memFs.MkdirAll(projectDir, 0755); err != nil {
		t.Fatalf("failed to create project directory: %v", err)
	}

	// Create a profile using helper
	profile := testutil.CreateTestProfile(t, "vendor/test-profile", "greeting.md.tmpl")

	// Create merged profile with content directory
	mergedProfile := testutil.CreateTestMergedProfileWithContentDir(t, memFs, profilesBasePath, profile, "content")

	// Create dependencies with variable support
	engine := template.NewTemplateEngineWithFs(memFs)
	pipeline := rendering.NewRenderingPipeline(engine)
	manifestGenerator := manifest.NewManifestGenerator()

	// Create service
	service := install.NewService(memFs, pipeline, manifestGenerator)

	// Create router with default target for weftlo
	contentConfig := domainprofile.ContentConfig{
		DefaultTarget: "weftlo",
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

	// Read manifest and verify source checksum differs from output checksum
	manifestPath := projectDir + "/.weftlo.manifest.json"
	manifestContent, err := afero.ReadFile(memFs, manifestPath)
	if err != nil {
		t.Fatalf("failed to read manifest: %v", err)
	}

	var m manifest.Manifest
	if err := json.Unmarshal(manifestContent, &m); err != nil {
		t.Fatalf("failed to parse manifest: %v", err)
	}

	// The template with variable should have different source and output checksums
	// because source is "Hello {{ .Variables.name }}!" and output is "Hello !"
	// (empty variable resolves to empty string)
	fileEntry, exists := m.Files["weftlo/greeting.md"]
	if !exists {
		t.Fatalf("expected greeting.md in manifest files")
	}

	// Source checksum should be computed from the raw template
	// Output checksum should be computed from the rendered content
	// They should differ when template has variable placeholders
	if fileEntry.SourceChecksum == fileEntry.OutputChecksum {
		t.Errorf("expected source checksum (%s) to differ from output checksum (%s) when template has variables",
			fileEntry.SourceChecksum, fileEntry.OutputChecksum)
	}
}

// Test: RenderedFile created with correct OutputContent (rendered content)
func TestService_RenderedFileHasCorrectOutputContent(t *testing.T) {
	t.Parallel()
	// Setup in-memory filesystem with template (static content)
	profilesBasePath := "/home/user/.weftlo/profiles"
	templateContent := "Static content without variables"
	projectDir := "/project"

	memFs := testutil.CreateTestFilesystem(t, map[string]string{
		profilesBasePath + "/vendor/test-profile/content/static.md.tmpl": templateContent,
	})

	// Create project directory
	if err := memFs.MkdirAll(projectDir, 0755); err != nil {
		t.Fatalf("failed to create project directory: %v", err)
	}

	// Create a profile using helper
	profile := testutil.CreateTestProfile(t, "vendor/test-profile", "static.md.tmpl")

	// Create merged profile with content directory
	mergedProfile := testutil.CreateTestMergedProfileWithContentDir(t, memFs, profilesBasePath, profile, "content")

	// Create dependencies
	engine := template.NewTemplateEngineWithFs(memFs)
	pipeline := rendering.NewRenderingPipeline(engine)
	manifestGenerator := manifest.NewManifestGenerator()

	// Create service
	service := install.NewService(memFs, pipeline, manifestGenerator)

	// Create router with default target for weftlo
	contentConfig := domainprofile.ContentConfig{
		DefaultTarget: "weftlo",
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

	// Verify the installed file content matches the rendered output
	installedContent, err := afero.ReadFile(memFs, projectDir+"/weftlo/static.md")
	if err != nil {
		t.Fatalf("failed to read installed file: %v", err)
	}

	if string(installedContent) != templateContent {
		t.Errorf("expected installed content '%s', got '%s'", templateContent, string(installedContent))
	}

	// Read manifest and verify output checksum matches the installed file
	manifestPath := projectDir + "/.weftlo.manifest.json"
	manifestFileContent, err := afero.ReadFile(memFs, manifestPath)
	if err != nil {
		t.Fatalf("failed to read manifest: %v", err)
	}

	var m manifest.Manifest
	if err := json.Unmarshal(manifestFileContent, &m); err != nil {
		t.Fatalf("failed to parse manifest: %v", err)
	}

	fileEntry, exists := m.Files["weftlo/static.md"]
	if !exists {
		t.Fatalf("expected static.md in manifest files")
	}

	// For static content without variables, source and output should be same
	expectedChecksum := manifest.ComputeChecksum([]byte(templateContent))
	if fileEntry.OutputChecksum != expectedChecksum {
		t.Errorf("expected output checksum %s, got %s", expectedChecksum, fileEntry.OutputChecksum)
	}
}

// Test: Checksums differ when template has variables that get substituted
func TestService_ChecksumsDifferWhenTemplateHasVariables(t *testing.T) {
	t.Parallel()
	// Setup in-memory filesystem with template using built-in variables
	profilesBasePath := "/home/user/.weftlo/profiles"
	templateContent := "Profile: {{ .Profile.Name }}"
	projectDir := "/project"

	memFs := testutil.CreateTestFilesystem(t, map[string]string{
		profilesBasePath + "/vendor/test-profile/content/info.md.tmpl": templateContent,
	})

	// Create project directory
	if err := memFs.MkdirAll(projectDir, 0755); err != nil {
		t.Fatalf("failed to create project directory: %v", err)
	}

	// Create a profile using helper
	profile := testutil.CreateTestProfile(t, "vendor/test-profile", "info.md.tmpl")

	// Create merged profile with content directory
	mergedProfile := testutil.CreateTestMergedProfileWithContentDir(t, memFs, profilesBasePath, profile, "content")

	// Create dependencies
	engine := template.NewTemplateEngineWithFs(memFs)
	pipeline := rendering.NewRenderingPipeline(engine)
	manifestGenerator := manifest.NewManifestGenerator()

	// Create service
	service := install.NewService(memFs, pipeline, manifestGenerator)

	// Create router with default target for weftlo
	contentConfig := domainprofile.ContentConfig{
		DefaultTarget: "weftlo",
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

	// Read manifest
	manifestPath := projectDir + "/.weftlo.manifest.json"
	manifestFileContent, err := afero.ReadFile(memFs, manifestPath)
	if err != nil {
		t.Fatalf("failed to read manifest: %v", err)
	}

	var m manifest.Manifest
	if err := json.Unmarshal(manifestFileContent, &m); err != nil {
		t.Fatalf("failed to parse manifest: %v", err)
	}

	fileEntry, exists := m.Files["weftlo/info.md"]
	if !exists {
		t.Fatalf("expected info.md in manifest files")
	}

	// Source checksum should be from raw template
	sourceChecksum := manifest.ComputeChecksum([]byte(templateContent))
	if fileEntry.SourceChecksum != sourceChecksum {
		t.Errorf("expected source checksum %s, got %s", sourceChecksum, fileEntry.SourceChecksum)
	}

	// Output checksum should be different (template was rendered)
	// The template "Profile: {{ .Profile.Name }}" becomes "Profile: vendor/test-profile"
	expectedOutput := "Profile: vendor/test-profile"
	outputChecksum := manifest.ComputeChecksum([]byte(expectedOutput))
	if fileEntry.OutputChecksum != outputChecksum {
		t.Errorf("expected output checksum %s, got %s", outputChecksum, fileEntry.OutputChecksum)
	}

	// Source and output checksums should differ
	if fileEntry.SourceChecksum == fileEntry.OutputChecksum {
		t.Errorf("expected source and output checksums to differ when template has variables")
	}
}

// =============================================================================
// Task Group 4: Variable Support in Target Paths - Install Service Integration
// =============================================================================

// Test 1: Merged variables are passed to NewContentRouter()
func TestService_MergedVariablesPassedToContentRouter(t *testing.T) {
	t.Parallel()
	// Setup in-memory filesystem with template in a directory that uses variable
	profilesBasePath := "/home/user/.weftlo/profiles"
	templateContent := "# Test content"
	projectDir := "/project"

	memFs := testutil.CreateTestFilesystem(t, map[string]string{
		profilesBasePath + "/vendor/test-profile/content/claude/config.md.tmpl": templateContent,
	})

	// Create project directory
	if err := memFs.MkdirAll(projectDir, 0755); err != nil {
		t.Fatalf("failed to create project directory: %v", err)
	}

	// Create a profile with template in claude subdirectory
	profile := &domainprofile.Profile{
		Name: "vendor/test-profile",
		TemplateFiles: []domainprofile.TemplateFile{
			{SourcePath: "claude/config.md.tmpl", IsPartial: false},
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

	// Create variables to pass to router
	variables := map[string]interface{}{
		"namespace": "acme",
	}

	// Create router with templated target and variables
	contentConfig := domainprofile.ContentConfig{
		Targets: map[string]string{
			"claude": ".{{ .Variables.namespace }}/claude",
		},
	}

	// Create router with merged variables - this tests that variables are passed correctly
	router, err := routing.NewContentRouter(contentConfig, nil, variables)
	if err != nil {
		t.Fatalf("failed to create router with variables: %v", err)
	}

	// Execute install
	opts := install.InstallOptions{
		DryRun: false,
	}

	result, err := service.Install(mergedProfile, projectDir, router, opts)
	if err != nil {
		t.Fatalf("install failed: %v", err)
	}

	// Verify file was processed
	if result.FilesProcessed != 1 {
		t.Errorf("expected 1 file processed, got %d", result.FilesProcessed)
	}

	// Verify file was installed at the evaluated target path (.acme/claude/config.md)
	expectedPath := projectDir + "/.acme/claude/config.md"
	exists, err := afero.Exists(memFs, expectedPath)
	if err != nil {
		t.Fatalf("failed to check file existence: %v", err)
	}
	if !exists {
		t.Errorf("expected file at %s, but it does not exist - variables were not passed to router", expectedPath)
	}
}

// Test 2: Variable precedence: global < profile < project
func TestService_VariablePrecedenceCorrectlyApplied(t *testing.T) {
	t.Parallel()
	// This test verifies that variables are merged with correct precedence
	// by using the MergeVariables function and passing result to router

	profilesBasePath := "/home/user/.weftlo/profiles"
	templateContent := "# Test content"
	projectDir := "/project"

	memFs := testutil.CreateTestFilesystem(t, map[string]string{
		profilesBasePath + "/vendor/test-profile/content/skills/skill.md.tmpl": templateContent,
	})

	// Create project directory
	if err := memFs.MkdirAll(projectDir, 0755); err != nil {
		t.Fatalf("failed to create project directory: %v", err)
	}

	// Create a profile
	profile := &domainprofile.Profile{
		Name: "vendor/test-profile",
		TemplateFiles: []domainprofile.TemplateFile{
			{SourcePath: "skills/skill.md.tmpl", IsPartial: false},
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

	// Simulate variable precedence: global < profile < project
	// Using DeepMergeVariablesChain from the rendering package
	globalVars := map[string]interface{}{"namespace": "global-ns", "env": "dev"}
	profileVars := map[string]interface{}{"namespace": "profile-ns"}
	projectVars := map[string]interface{}{"namespace": "project-ns"} // This should win

	// Merge variables using the same function the install service would use
	mergedVars, _ := rendering.DeepMergeVariablesChain(
		rendering.VariableSource{Name: "global", Vars: globalVars},
		rendering.VariableSource{Name: "profile", Vars: profileVars},
		rendering.VariableSource{Name: "project", Vars: projectVars},
	)

	// Create router with templated target using merged variables
	contentConfig := domainprofile.ContentConfig{
		Targets: map[string]string{
			"skills": ".{{ .Variables.namespace }}/skills",
		},
	}

	router, err := routing.NewContentRouter(contentConfig, nil, mergedVars)
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

	// Verify file was installed at path using project variable (highest precedence)
	expectedPath := projectDir + "/.project-ns/skills/skill.md"
	exists, err := afero.Exists(memFs, expectedPath)
	if err != nil {
		t.Fatalf("failed to check file existence: %v", err)
	}
	if !exists {
		t.Errorf("expected file at %s - project variable should have highest precedence", expectedPath)

		// Check if it was installed with wrong namespace for better error message
		globalPath := projectDir + "/.global-ns/skills/skill.md"
		profilePath := projectDir + "/.profile-ns/skills/skill.md"
		if existsGlobal, _ := afero.Exists(memFs, globalPath); existsGlobal {
			t.Errorf("file was at global path %s instead - precedence is wrong", globalPath)
		}
		if existsProfile, _ := afero.Exists(memFs, profilePath); existsProfile {
			t.Errorf("file was at profile path %s instead - precedence is wrong", profilePath)
		}
	}
}

// Test 3: Installation succeeds with templated targets
func TestService_InstallationSucceedsWithTemplatedTargets(t *testing.T) {
	t.Parallel()
	profilesBasePath := "/home/user/.weftlo/profiles"
	projectDir := "/project"

	// Multiple files going to different templated targets
	memFs := testutil.CreateTestFilesystem(t, map[string]string{
		profilesBasePath + "/vendor/test-profile/content/claude/settings.md.tmpl": "Claude settings",
		profilesBasePath + "/vendor/test-profile/content/skills/skill1.md.tmpl":   "Skill 1",
		profilesBasePath + "/vendor/test-profile/content/prompts/prompt1.md.tmpl": "Prompt 1",
	})

	// Create project directory
	if err := memFs.MkdirAll(projectDir, 0755); err != nil {
		t.Fatalf("failed to create project directory: %v", err)
	}

	// Create a profile with templates in different directories
	profile := &domainprofile.Profile{
		Name: "vendor/test-profile",
		TemplateFiles: []domainprofile.TemplateFile{
			{SourcePath: "claude/settings.md.tmpl", IsPartial: false},
			{SourcePath: "skills/skill1.md.tmpl", IsPartial: false},
			{SourcePath: "prompts/prompt1.md.tmpl", IsPartial: false},
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

	// Create variables for templated targets
	variables := map[string]interface{}{
		"namespace": "mycompany",
	}

	// Create router with templated targets
	contentConfig := domainprofile.ContentConfig{
		Targets: map[string]string{
			"claude":  ".{{ .Variables.namespace }}/claude",
			"skills":  ".{{ .Variables.namespace }}/skills",
			"prompts": ".{{ .Variables.namespace }}/prompts",
		},
	}

	router, err := routing.NewContentRouter(contentConfig, nil, variables)
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

	// Verify files exist at their templated target paths
	expectedPaths := []string{
		projectDir + "/.mycompany/claude/settings.md",
		projectDir + "/.mycompany/skills/skill1.md",
		projectDir + "/.mycompany/prompts/prompt1.md",
	}

	for _, path := range expectedPaths {
		exists, err := afero.Exists(memFs, path)
		if err != nil {
			t.Fatalf("failed to check file existence for %s: %v", path, err)
		}
		if !exists {
			t.Errorf("expected file at %s, but it does not exist", path)
		}
	}

	// Verify manifest was created
	manifestExists, err := afero.Exists(memFs, projectDir+"/.weftlo.manifest.json")
	if err != nil {
		t.Fatalf("failed to check manifest existence: %v", err)
	}
	if !manifestExists {
		t.Errorf("expected manifest to be created")
	}
}

// Test 4: Installation fails gracefully on evaluation error
func TestService_InstallationFailsGracefullyOnEvaluationError(t *testing.T) {
	t.Parallel()
	// This test verifies that when router creation fails due to missing variable,
	// the error is properly reported (fail-fast behavior)

	// Create router with templated target but missing variable
	contentConfig := domainprofile.ContentConfig{
		Targets: map[string]string{
			"claude": ".{{ .Variables.namespace }}/claude", // namespace not defined
		},
	}

	// Empty variables map - will cause MissingVariableError
	emptyVariables := map[string]interface{}{}

	// Attempt to create router - should fail with MissingVariableError
	_, err := routing.NewContentRouter(contentConfig, nil, emptyVariables)

	// Verify error is returned
	if err == nil {
		t.Fatalf("expected error when variable is missing, but got nil")
	}

	// Verify error is of expected type (MissingVariableError)
	var missingVarErr *routing.MissingVariableError
	if !errors.As(err, &missingVarErr) {
		t.Errorf("expected MissingVariableError, got %T: %v", err, err)
	}

	// Verify error contains useful information
	errMsg := err.Error()
	if !strings.Contains(errMsg, "namespace") {
		t.Errorf("error message should contain the missing variable name 'namespace', got: %s", errMsg)
	}
	if !strings.Contains(errMsg, "claude") {
		t.Errorf("error message should contain the target name 'claude', got: %s", errMsg)
	}
}

// =============================================================================
// Task Group 4 (Global Target Overrides): Install Service Integration Tests
// =============================================================================

// Test 1: BuildMergedOverrides correctly merges four levels of overrides
func TestBuildMergedOverrides_FourLevelMerge(t *testing.T) {
	t.Parallel()
	// Level 1: Profile targets
	profileTargets := map[string]string{
		"claude": ".claude",
		"skills": ".skills",
		"cursor": ".cursor",
	}

	// Level 2: Global universal overrides - override claude, suppress cursor
	cursorNil := (*string)(nil)
	claudeGlobal := ".global-claude"
	globalConfig := &domainconfig.GlobalConfig{
		TargetOverrides: map[string]*string{
			"claude": &claudeGlobal,
			"cursor": cursorNil, // suppress cursor globally
		},
	}

	// Level 3: Global profile-specific overrides - override skills for this profile
	skillsProfileSpecific := ".my-profile-skills"
	globalConfig.ProfileOverrides = map[string]domainconfig.ProfileOverrideConfig{
		"vendor/test-profile": {
			TargetOverrides: map[string]*string{
				"skills": &skillsProfileSpecific,
			},
		},
	}

	// Level 4: Project overrides - unsuppress cursor
	cursorProject := ".project-cursor"
	projectOverrides := map[string]*string{
		"cursor": &cursorProject, // unsuppress cursor at project level
	}

	// No templates in overrides for this test
	mergedVars := map[string]interface{}{}

	// Execute merge
	result, err := install.BuildMergedOverrides(
		profileTargets,
		globalConfig,
		"vendor/test-profile",
		projectOverrides,
		mergedVars,
	)
	if err != nil {
		t.Fatalf("BuildMergedOverrides failed: %v", err)
	}

	// Verify claude was overridden by global universal (level 2)
	if result["claude"] == nil || *result["claude"] != ".global-claude" {
		t.Errorf("expected claude to be '.global-claude' from global override, got %v", result["claude"])
	}

	// Verify skills was overridden by global profile-specific (level 3)
	if result["skills"] == nil || *result["skills"] != ".my-profile-skills" {
		t.Errorf("expected skills to be '.my-profile-skills' from profile-specific override, got %v", result["skills"])
	}

	// Verify cursor was unsuppressed by project (level 4)
	if result["cursor"] == nil || *result["cursor"] != ".project-cursor" {
		t.Errorf("expected cursor to be '.project-cursor' (unsuppressed by project), got %v", result["cursor"])
	}
}

// Test 2: BuildMergedOverrides selects profile-specific overrides based on profile name
func TestBuildMergedOverrides_ProfileSpecificSelection(t *testing.T) {
	t.Parallel()
	profileTargets := map[string]string{
		"agents": ".agents",
	}

	// Setup global config with overrides for multiple profiles
	agentsProfile1 := ".profile1-agents"
	agentsProfile2 := ".profile2-agents"
	globalConfig := &domainconfig.GlobalConfig{
		ProfileOverrides: map[string]domainconfig.ProfileOverrideConfig{
			"vendor/profile-one": {
				TargetOverrides: map[string]*string{
					"agents": &agentsProfile1,
				},
			},
			"vendor/profile-two": {
				TargetOverrides: map[string]*string{
					"agents": &agentsProfile2,
				},
			},
		},
	}

	// Test with profile-one - should get profile1's override
	result1, err := install.BuildMergedOverrides(
		profileTargets,
		globalConfig,
		"vendor/profile-one",
		nil,
		nil,
	)
	if err != nil {
		t.Fatalf("BuildMergedOverrides failed for profile-one: %v", err)
	}

	if result1["agents"] == nil || *result1["agents"] != ".profile1-agents" {
		t.Errorf("expected agents to be '.profile1-agents' for profile-one, got %v", result1["agents"])
	}

	// Test with profile-two - should get profile2's override
	result2, err := install.BuildMergedOverrides(
		profileTargets,
		globalConfig,
		"vendor/profile-two",
		nil,
		nil,
	)
	if err != nil {
		t.Fatalf("BuildMergedOverrides failed for profile-two: %v", err)
	}

	if result2["agents"] == nil || *result2["agents"] != ".profile2-agents" {
		t.Errorf("expected agents to be '.profile2-agents' for profile-two, got %v", result2["agents"])
	}

	// Test with unknown profile - should fall back to profile targets (level 1)
	result3, err := install.BuildMergedOverrides(
		profileTargets,
		globalConfig,
		"vendor/unknown-profile",
		nil,
		nil,
	)
	if err != nil {
		t.Fatalf("BuildMergedOverrides failed for unknown-profile: %v", err)
	}

	if result3["agents"] == nil || *result3["agents"] != ".agents" {
		t.Errorf("expected agents to be '.agents' (profile default) for unknown-profile, got %v", result3["agents"])
	}
}

// Test 3: EvaluateOverrideTemplates evaluates templates with merged variables
func TestEvaluateOverrideTemplates_TemplateEvaluation(t *testing.T) {
	t.Parallel()
	// Setup overrides with template expressions
	claudeTemplate := ".{{ .Variables.namespace }}/claude"
	skillsTemplate := ".{{ .Variables.org }}/skills"
	emptyPath := ""
	overrides := map[string]*string{
		"claude":     &claudeTemplate,
		"skills":     &skillsTemplate,
		"cursor":     nil,        // suppressed - should stay nil
		"root-redir": &emptyPath, // redirect to root - should stay empty
	}

	// Variables for evaluation
	variables := map[string]interface{}{
		"namespace": "acme",
		"org":       "myorg",
	}

	// Execute evaluation
	result, err := install.EvaluateOverrideTemplates(overrides, variables)
	if err != nil {
		t.Fatalf("EvaluateOverrideTemplates failed: %v", err)
	}

	// Verify claude template was evaluated
	if result["claude"] == nil || *result["claude"] != ".acme/claude" {
		t.Errorf("expected claude to be '.acme/claude', got %v", result["claude"])
	}

	// Verify skills template was evaluated
	if result["skills"] == nil || *result["skills"] != ".myorg/skills" {
		t.Errorf("expected skills to be '.myorg/skills', got %v", result["skills"])
	}

	// Verify nil (suppression) is preserved
	if result["cursor"] != nil {
		t.Errorf("expected cursor to remain nil (suppressed), got %v", *result["cursor"])
	}

	// Verify empty string (root redirect) is preserved
	if result["root-redir"] == nil || *result["root-redir"] != "" {
		t.Errorf("expected root-redir to remain empty string, got %v", result["root-redir"])
	}
}

// Test 4: MergeIgnorePatterns merges patterns in correct order
func TestMergeIgnorePatterns_CorrectOrder(t *testing.T) {
	t.Parallel()
	// Create profile matcher with some patterns
	profileMatcher := domainignore.NewPatternMatcher([]string{
		"*.draft.md",
		"temp/",
	})

	// Global patterns
	globalPatterns := []string{
		"**/*.bak",
		"**/cache/**",
	}

	// Create project matcher with patterns including negation
	projectMatcher := domainignore.NewPatternMatcher([]string{
		"!important.draft.md", // re-include specific draft file
		"*.log",
	})

	// Execute merge
	result := install.MergeIgnorePatterns(profileMatcher, globalPatterns, projectMatcher)

	// The merged matcher should:
	// 1. Match *.draft.md from profile (but important.draft.md is re-included by project)
	// 2. Match **/*.bak from global
	// 3. Match *.log from project
	// 4. NOT match important.draft.md (negated by project)

	// Test that regular draft files are still ignored
	if !result.Match("regular.draft.md", false) {
		t.Errorf("expected regular.draft.md to be ignored")
	}

	// Test that important.draft.md is NOT ignored (project negation)
	if result.Match("important.draft.md", false) {
		t.Errorf("expected important.draft.md to NOT be ignored (project negation)")
	}

	// Test that .bak files are ignored (from global)
	if !result.Match("some/path/file.bak", false) {
		t.Errorf("expected .bak files to be ignored from global patterns")
	}

	// Test that .log files are ignored (from project)
	if !result.Match("debug.log", false) {
		t.Errorf("expected .log files to be ignored from project patterns")
	}
}

// Test 5: ExtractProfileName returns correct profile name from MergedProfile
func TestExtractProfileName_ReturnsLeafName(t *testing.T) {
	t.Parallel()
	// Setup filesystem
	profilesBasePath := "/home/user/.weftlo/profiles"
	memFs := testutil.CreateTestFilesystem(t, map[string]string{
		profilesBasePath + "/vendor/child-profile/content/file.md.tmpl": "content",
	})

	// Create a profile
	profile := testutil.CreateTestProfile(t, "vendor/child-profile", "file.md.tmpl")

	// Create merged profile
	mergedProfile := testutil.CreateTestMergedProfileWithContentDir(t, memFs, profilesBasePath, profile, "content")

	// Extract profile name
	profileName := install.ExtractProfileName(mergedProfile)

	// Verify
	if profileName != "vendor/child-profile" {
		t.Errorf("expected profile name 'vendor/child-profile', got '%s'", profileName)
	}

	// Test nil case
	nilProfileName := install.ExtractProfileName(nil)
	if nilProfileName != "" {
		t.Errorf("expected empty string for nil profile, got '%s'", nilProfileName)
	}
}

// =============================================================================
// Task Group 5: Strategic Gap-Filling Tests for Global Target Overrides
// =============================================================================

// Test 1: End-to-end install with global config override applied to routing
// This tests the complete workflow: global config -> merged overrides -> router -> installed files
func TestService_EndToEndInstallWithGlobalOverrides(t *testing.T) {
	t.Parallel()
	profilesBasePath := "/home/user/.weftlo/profiles"
	projectDir := "/project"
	templateContent := "# Claude Config\nContent here"

	// Setup filesystem with template file in "claude" directory
	memFs := testutil.CreateTestFilesystem(t, map[string]string{
		profilesBasePath + "/vendor/test-profile/content/claude/config.md.tmpl": templateContent,
	})

	// Create project directory
	if err := memFs.MkdirAll(projectDir, 0755); err != nil {
		t.Fatalf("failed to create project directory: %v", err)
	}

	// Create profile with content.targets
	profile := &domainprofile.Profile{
		Name: "vendor/test-profile",
		ContentConfig: domainprofile.ContentConfig{
			Targets: map[string]string{
				"claude": ".claude", // Profile says install to .claude
			},
		},
		TemplateFiles: []domainprofile.TemplateFile{
			{SourcePath: "claude/config.md.tmpl", IsPartial: false},
		},
	}

	// Create merged profile
	mergedProfile := testutil.CreateTestMergedProfileWithContentDir(t, memFs, profilesBasePath, profile, "content")

	// Create global config that overrides the target to a custom location
	customPath := ".custom-claude-location"
	globalConfig := &domainconfig.GlobalConfig{
		TargetOverrides: map[string]*string{
			"claude": &customPath, // Global override redirects claude to custom location
		},
	}

	// Build merged overrides using the helper (simulating what install command would do)
	mergedOverrides, err := install.BuildMergedOverrides(
		profile.ContentConfig.Targets,
		globalConfig,
		"vendor/test-profile",
		nil, // no project overrides
		nil, // no variables
	)
	if err != nil {
		t.Fatalf("BuildMergedOverrides failed: %v", err)
	}

	// Create router with merged overrides
	contentConfig := domainprofile.ContentConfig{
		Targets: map[string]string{
			"claude": ".claude", // Base target from profile
		},
	}
	router, err := routing.NewContentRouter(contentConfig, mergedOverrides, nil)
	if err != nil {
		t.Fatalf("failed to create router with overrides: %v", err)
	}

	// Create dependencies
	engine := template.NewTemplateEngineWithFs(memFs)
	pipeline := rendering.NewRenderingPipeline(engine)
	manifestGenerator := manifest.NewManifestGenerator()

	// Create service and execute install
	service := install.NewService(memFs, pipeline, manifestGenerator)
	opts := install.InstallOptions{DryRun: false}

	result, err := service.Install(mergedProfile, projectDir, router, opts)
	if err != nil {
		t.Fatalf("install failed: %v", err)
	}

	// Verify file was processed
	if result.FilesProcessed != 1 {
		t.Errorf("expected 1 file processed, got %d", result.FilesProcessed)
	}

	// Verify file was installed at the GLOBAL OVERRIDE location, not the profile's original location
	expectedPath := projectDir + "/.custom-claude-location/config.md"
	exists, err := afero.Exists(memFs, expectedPath)
	if err != nil {
		t.Fatalf("failed to check file existence: %v", err)
	}
	if !exists {
		t.Errorf("expected file at %s (from global override), but it does not exist", expectedPath)

		// Check if it was installed at original location for better error message
		originalPath := projectDir + "/.claude/config.md"
		if existsOriginal, _ := afero.Exists(memFs, originalPath); existsOriginal {
			t.Errorf("file was at original path %s instead - global override was not applied", originalPath)
		}
	}

	// Verify content is correct
	if exists {
		content, err := afero.ReadFile(memFs, expectedPath)
		if err != nil {
			t.Fatalf("failed to read installed file: %v", err)
		}
		if string(content) != templateContent {
			t.Errorf("expected content '%s', got '%s'", templateContent, string(content))
		}
	}
}

// Test 2: Complete four-level override scenario in full install workflow
// Tests all four levels: profile targets < global universal < global profile-specific < project
func TestService_FourLevelOverrideFullInstall(t *testing.T) {
	t.Parallel()
	profilesBasePath := "/home/user/.weftlo/profiles"
	projectDir := "/project"

	// Setup filesystem with templates in different target directories
	memFs := testutil.CreateTestFilesystem(t, map[string]string{
		profilesBasePath + "/vendor/test-profile/content/claude/file.md.tmpl":   "claude content",
		profilesBasePath + "/vendor/test-profile/content/skills/file.md.tmpl":   "skills content",
		profilesBasePath + "/vendor/test-profile/content/agents/file.md.tmpl":   "agents content",
		profilesBasePath + "/vendor/test-profile/content/commands/file.md.tmpl": "commands content",
	})

	// Create project directory
	if err := memFs.MkdirAll(projectDir, 0755); err != nil {
		t.Fatalf("failed to create project directory: %v", err)
	}

	// Level 1: Profile with original targets
	profile := &domainprofile.Profile{
		Name: "vendor/test-profile",
		ContentConfig: domainprofile.ContentConfig{
			Targets: map[string]string{
				"claude":   ".level1-claude",   // Will be unchanged (no override at any level)
				"skills":   ".level1-skills",   // Will be overridden at level 2 (global universal)
				"agents":   ".level1-agents",   // Will be overridden at level 3 (global profile-specific)
				"commands": ".level1-commands", // Will be overridden at level 4 (project)
			},
		},
		TemplateFiles: []domainprofile.TemplateFile{
			{SourcePath: "claude/file.md.tmpl", IsPartial: false},
			{SourcePath: "skills/file.md.tmpl", IsPartial: false},
			{SourcePath: "agents/file.md.tmpl", IsPartial: false},
			{SourcePath: "commands/file.md.tmpl", IsPartial: false},
		},
	}

	mergedProfile := testutil.CreateTestMergedProfileWithContentDir(t, memFs, profilesBasePath, profile, "content")

	// Level 2 & 3: Global config
	skillsLevel2 := ".level2-skills"
	agentsLevel2 := ".level2-agents"     // Will be further overridden at level 3
	commandsLevel2 := ".level2-commands" // Will be further overridden at level 4
	agentsLevel3 := ".level3-agents"
	commandsLevel3 := ".level3-commands" // Will be further overridden at level 4

	globalConfig := &domainconfig.GlobalConfig{
		TargetOverrides: map[string]*string{
			"skills":   &skillsLevel2,
			"agents":   &agentsLevel2,
			"commands": &commandsLevel2,
		},
		ProfileOverrides: map[string]domainconfig.ProfileOverrideConfig{
			"vendor/test-profile": {
				TargetOverrides: map[string]*string{
					"agents":   &agentsLevel3,
					"commands": &commandsLevel3,
				},
			},
		},
	}

	// Level 4: Project overrides
	commandsLevel4 := ".level4-commands"
	projectOverrides := map[string]*string{
		"commands": &commandsLevel4,
	}

	// Build merged overrides
	mergedOverrides, err := install.BuildMergedOverrides(
		profile.ContentConfig.Targets,
		globalConfig,
		"vendor/test-profile",
		projectOverrides,
		nil,
	)
	if err != nil {
		t.Fatalf("BuildMergedOverrides failed: %v", err)
	}

	// Create router with merged overrides
	router, err := routing.NewContentRouter(profile.ContentConfig, mergedOverrides, nil)
	if err != nil {
		t.Fatalf("failed to create router: %v", err)
	}

	// Create service and execute install
	engine := template.NewTemplateEngineWithFs(memFs)
	pipeline := rendering.NewRenderingPipeline(engine)
	manifestGenerator := manifest.NewManifestGenerator()
	service := install.NewService(memFs, pipeline, manifestGenerator)

	result, err := service.Install(mergedProfile, projectDir, router, install.InstallOptions{DryRun: false})
	if err != nil {
		t.Fatalf("install failed: %v", err)
	}

	if result.FilesProcessed != 4 {
		t.Errorf("expected 4 files processed, got %d", result.FilesProcessed)
	}

	// Verify each file was installed at the correct level's path
	expectedPaths := map[string]string{
		"claude":   projectDir + "/.level1-claude/file.md",   // Level 1 (profile, no override)
		"skills":   projectDir + "/.level2-skills/file.md",   // Level 2 (global universal)
		"agents":   projectDir + "/.level3-agents/file.md",   // Level 3 (global profile-specific)
		"commands": projectDir + "/.level4-commands/file.md", // Level 4 (project)
	}

	for target, expectedPath := range expectedPaths {
		exists, err := afero.Exists(memFs, expectedPath)
		if err != nil {
			t.Fatalf("failed to check file existence for %s: %v", target, err)
		}
		if !exists {
			t.Errorf("expected %s file at %s, but it does not exist", target, expectedPath)
		}
	}
}

// Test 3: Template variable in global override with variable precedence
// Tests that global overrides with templates use the correctly merged variables
func TestService_GlobalOverrideTemplateWithVariablePrecedence(t *testing.T) {
	t.Parallel()
	profilesBasePath := "/home/user/.weftlo/profiles"
	projectDir := "/project"

	memFs := testutil.CreateTestFilesystem(t, map[string]string{
		profilesBasePath + "/vendor/test-profile/content/agents/file.md.tmpl": "agents content",
	})

	if err := memFs.MkdirAll(projectDir, 0755); err != nil {
		t.Fatalf("failed to create project directory: %v", err)
	}

	profile := &domainprofile.Profile{
		Name: "vendor/test-profile",
		ContentConfig: domainprofile.ContentConfig{
			Targets: map[string]string{
				"agents": ".agents", // Base target
			},
		},
		TemplateFiles: []domainprofile.TemplateFile{
			{SourcePath: "agents/file.md.tmpl", IsPartial: false},
		},
	}

	mergedProfile := testutil.CreateTestMergedProfileWithContentDir(t, memFs, profilesBasePath, profile, "content")

	// Global config with templated override
	agentsTemplate := ".{{ .Variables.namespace }}/agents"
	globalConfig := &domainconfig.GlobalConfig{
		TargetOverrides: map[string]*string{
			"agents": &agentsTemplate,
		},
	}

	// Simulate variable precedence: global < profile < project
	// The "namespace" variable should use the project value
	globalVars := map[string]interface{}{"namespace": "global-ns"}
	profileVars := map[string]interface{}{"namespace": "profile-ns"}
	projectVars := map[string]interface{}{"namespace": "project-ns"} // Should win

	mergedVars, _ := rendering.DeepMergeVariablesChain(
		rendering.VariableSource{Name: "global", Vars: globalVars},
		rendering.VariableSource{Name: "profile", Vars: profileVars},
		rendering.VariableSource{Name: "project", Vars: projectVars},
	)

	// Build merged overrides with template evaluation
	mergedOverrides, err := install.BuildMergedOverrides(
		profile.ContentConfig.Targets,
		globalConfig,
		"vendor/test-profile",
		nil,
		mergedVars,
	)
	if err != nil {
		t.Fatalf("BuildMergedOverrides failed: %v", err)
	}

	// Verify the template was evaluated with the correct (project) variable
	if mergedOverrides["agents"] == nil || *mergedOverrides["agents"] != ".project-ns/agents" {
		t.Errorf("expected agents to be '.project-ns/agents' (project variable), got %v", mergedOverrides["agents"])
	}

	// Create router and execute install to verify end-to-end
	router, err := routing.NewContentRouter(profile.ContentConfig, mergedOverrides, nil)
	if err != nil {
		t.Fatalf("failed to create router: %v", err)
	}

	engine := template.NewTemplateEngineWithFs(memFs)
	pipeline := rendering.NewRenderingPipeline(engine)
	manifestGenerator := manifest.NewManifestGenerator()
	service := install.NewService(memFs, pipeline, manifestGenerator)

	_, err = service.Install(mergedProfile, projectDir, router, install.InstallOptions{DryRun: false})
	if err != nil {
		t.Fatalf("install failed: %v", err)
	}

	// Verify file was installed at path using project variable
	expectedPath := projectDir + "/.project-ns/agents/file.md"
	exists, err := afero.Exists(memFs, expectedPath)
	if err != nil {
		t.Fatalf("failed to check file existence: %v", err)
	}
	if !exists {
		t.Errorf("expected file at %s, but it does not exist - template variable precedence may be wrong", expectedPath)
	}
}

// Test 4: Global suppress then project unsuppress in full install workflow
// Tests that a target suppressed at global level can be unsuppressed at project level
func TestService_GlobalSuppressThenProjectUnsuppress(t *testing.T) {
	t.Parallel()
	profilesBasePath := "/home/user/.weftlo/profiles"
	projectDir := "/project"

	memFs := testutil.CreateTestFilesystem(t, map[string]string{
		profilesBasePath + "/vendor/test-profile/content/cursor/config.md.tmpl": "cursor content",
	})

	if err := memFs.MkdirAll(projectDir, 0755); err != nil {
		t.Fatalf("failed to create project directory: %v", err)
	}

	profile := &domainprofile.Profile{
		Name: "vendor/test-profile",
		ContentConfig: domainprofile.ContentConfig{
			Targets: map[string]string{
				"cursor": ".cursor", // Profile defines cursor target
			},
		},
		TemplateFiles: []domainprofile.TemplateFile{
			{SourcePath: "cursor/config.md.tmpl", IsPartial: false},
		},
	}

	mergedProfile := testutil.CreateTestMergedProfileWithContentDir(t, memFs, profilesBasePath, profile, "content")

	// Global config SUPPRESSES cursor globally
	globalConfig := &domainconfig.GlobalConfig{
		TargetOverrides: map[string]*string{
			"cursor": nil, // nil = suppress
		},
	}

	// Project override UNSUPPRESSES cursor by providing a path
	cursorPath := ".project-cursor"
	projectOverrides := map[string]*string{
		"cursor": &cursorPath,
	}

	// Build merged overrides
	mergedOverrides, err := install.BuildMergedOverrides(
		profile.ContentConfig.Targets,
		globalConfig,
		"vendor/test-profile",
		projectOverrides,
		nil,
	)
	if err != nil {
		t.Fatalf("BuildMergedOverrides failed: %v", err)
	}

	// Verify cursor is unsuppressed with project path
	if mergedOverrides["cursor"] == nil {
		t.Fatalf("expected cursor to be unsuppressed by project, but it's still nil")
	}
	if *mergedOverrides["cursor"] != ".project-cursor" {
		t.Errorf("expected cursor to be '.project-cursor', got '%s'", *mergedOverrides["cursor"])
	}

	// Create router and execute install
	router, err := routing.NewContentRouter(profile.ContentConfig, mergedOverrides, nil)
	if err != nil {
		t.Fatalf("failed to create router: %v", err)
	}

	engine := template.NewTemplateEngineWithFs(memFs)
	pipeline := rendering.NewRenderingPipeline(engine)
	manifestGenerator := manifest.NewManifestGenerator()
	service := install.NewService(memFs, pipeline, manifestGenerator)

	result, err := service.Install(mergedProfile, projectDir, router, install.InstallOptions{DryRun: false})
	if err != nil {
		t.Fatalf("install failed: %v", err)
	}

	// Verify file was processed (not suppressed)
	if result.FilesProcessed != 1 {
		t.Errorf("expected 1 file processed (unsuppressed), got %d", result.FilesProcessed)
	}

	// Verify file was installed at the project override path
	expectedPath := projectDir + "/.project-cursor/config.md"
	exists, err := afero.Exists(memFs, expectedPath)
	if err != nil {
		t.Fatalf("failed to check file existence: %v", err)
	}
	if !exists {
		t.Errorf("expected file at %s (unsuppressed by project), but it does not exist", expectedPath)
	}
}

// Test 5: Verify suppressed target does NOT create files
// Complementary test to unsuppress - ensures suppression actually works
func TestService_SuppressedTargetDoesNotCreateFiles(t *testing.T) {
	t.Parallel()
	profilesBasePath := "/home/user/.weftlo/profiles"
	projectDir := "/project"

	memFs := testutil.CreateTestFilesystem(t, map[string]string{
		profilesBasePath + "/vendor/test-profile/content/cursor/config.md.tmpl": "cursor content",
		profilesBasePath + "/vendor/test-profile/content/claude/config.md.tmpl": "claude content",
	})

	if err := memFs.MkdirAll(projectDir, 0755); err != nil {
		t.Fatalf("failed to create project directory: %v", err)
	}

	profile := &domainprofile.Profile{
		Name: "vendor/test-profile",
		ContentConfig: domainprofile.ContentConfig{
			Targets: map[string]string{
				"cursor": ".cursor",
				"claude": ".claude",
			},
		},
		TemplateFiles: []domainprofile.TemplateFile{
			{SourcePath: "cursor/config.md.tmpl", IsPartial: false},
			{SourcePath: "claude/config.md.tmpl", IsPartial: false},
		},
	}

	mergedProfile := testutil.CreateTestMergedProfileWithContentDir(t, memFs, profilesBasePath, profile, "content")

	// Global config suppresses cursor but NOT claude
	globalConfig := &domainconfig.GlobalConfig{
		TargetOverrides: map[string]*string{
			"cursor": nil, // Suppress cursor
			// claude not overridden - will use profile target
		},
	}

	mergedOverrides, err := install.BuildMergedOverrides(
		profile.ContentConfig.Targets,
		globalConfig,
		"vendor/test-profile",
		nil, // No project overrides
		nil,
	)
	if err != nil {
		t.Fatalf("BuildMergedOverrides failed: %v", err)
	}

	router, err := routing.NewContentRouter(profile.ContentConfig, mergedOverrides, nil)
	if err != nil {
		t.Fatalf("failed to create router: %v", err)
	}

	engine := template.NewTemplateEngineWithFs(memFs)
	pipeline := rendering.NewRenderingPipeline(engine)
	manifestGenerator := manifest.NewManifestGenerator()
	service := install.NewService(memFs, pipeline, manifestGenerator)

	result, err := service.Install(mergedProfile, projectDir, router, install.InstallOptions{DryRun: false})
	if err != nil {
		t.Fatalf("install failed: %v", err)
	}

	// Only 1 file should be processed (claude), cursor should be suppressed
	if result.FilesProcessed != 1 {
		t.Errorf("expected 1 file processed (cursor suppressed), got %d", result.FilesProcessed)
	}

	// Verify claude WAS installed
	claudePath := projectDir + "/.claude/config.md"
	claudeExists, err := afero.Exists(memFs, claudePath)
	if err != nil {
		t.Fatalf("failed to check claude file existence: %v", err)
	}
	if !claudeExists {
		t.Errorf("expected claude file at %s, but it does not exist", claudePath)
	}

	// Verify cursor was NOT installed (suppressed)
	cursorPath := projectDir + "/.cursor/config.md"
	cursorExists, err := afero.Exists(memFs, cursorPath)
	if err != nil {
		t.Fatalf("failed to check cursor file existence: %v", err)
	}
	if cursorExists {
		t.Errorf("cursor file at %s should NOT exist (suppressed by global config)", cursorPath)
	}
}

// Test 6: BuildMergedOverrides with nil GlobalConfig
// Tests graceful handling when no global config exists
func TestBuildMergedOverrides_NilGlobalConfig(t *testing.T) {
	t.Parallel()
	profileTargets := map[string]string{
		"claude": ".claude",
		"cursor": ".cursor",
	}

	// No global config at all
	result, err := install.BuildMergedOverrides(
		profileTargets,
		nil, // No global config
		"vendor/test-profile",
		nil, // No project overrides
		nil,
	)
	if err != nil {
		t.Fatalf("BuildMergedOverrides with nil global config failed: %v", err)
	}

	// Should just have profile targets converted to overrides
	if result["claude"] == nil || *result["claude"] != ".claude" {
		t.Errorf("expected claude to be '.claude' from profile, got %v", result["claude"])
	}
	if result["cursor"] == nil || *result["cursor"] != ".cursor" {
		t.Errorf("expected cursor to be '.cursor' from profile, got %v", result["cursor"])
	}
}

// Test 7: Complex Sprig function in global override template
// Tests that Sprig functions work in global override templates
func TestEvaluateOverrideTemplates_SprigFunctionSupport(t *testing.T) {
	t.Parallel()
	// Use Sprig functions in template
	templateWithSprig := ".{{ .Variables.namespace | lower }}/agents"
	overrides := map[string]*string{
		"agents": &templateWithSprig,
	}

	variables := map[string]interface{}{
		"namespace": "MY-COMPANY",
	}

	result, err := install.EvaluateOverrideTemplates(overrides, variables)
	if err != nil {
		t.Fatalf("EvaluateOverrideTemplates with Sprig function failed: %v", err)
	}

	// Verify the Sprig "lower" function was applied
	if result["agents"] == nil || *result["agents"] != ".my-company/agents" {
		t.Errorf("expected agents to be '.my-company/agents' (lowercased), got %v", result["agents"])
	}
}

// =============================================================================
// Reference Template Functions Integration Tests (Task Group 4)
// =============================================================================

// Test: Install service creates router with merged variables and passes to rendering
func TestService_InstallCreatesRouterWithMergedVariablesAndPassesToRendering(t *testing.T) {
	t.Parallel()
	// This test verifies that the router is passed from install service to rendering pipeline
	// so that templates can use reference() function during rendering

	profilesBasePath := "/home/user/.weftlo/profiles"
	projectDir := "/project"

	// Template that uses reference() function
	mainTemplate := `# Main Document
See: {{ reference "claude/config.md" }}`
	configContent := "Config content"

	memFs := testutil.CreateTestFilesystem(t, map[string]string{
		profilesBasePath + "/vendor/test-profile/content/claude/main.md.tmpl": mainTemplate,
		profilesBasePath + "/vendor/test-profile/content/claude/config.md":    configContent,
	})

	if err := memFs.MkdirAll(projectDir, 0755); err != nil {
		t.Fatalf("failed to create project directory: %v", err)
	}

	profile := &domainprofile.Profile{
		Name: "vendor/test-profile",
		ContentConfig: domainprofile.ContentConfig{
			Targets: map[string]string{
				"claude": ".claude",
			},
		},
		TemplateFiles: []domainprofile.TemplateFile{
			{SourcePath: "claude/main.md.tmpl", IsPartial: false},
			{SourcePath: "claude/config.md", IsPartial: false},
		},
	}

	mergedProfile := testutil.CreateTestMergedProfileWithContentDir(t, memFs, profilesBasePath, profile, "content")

	// Create router
	router, err := routing.NewContentRouter(profile.ContentConfig, nil, nil)
	if err != nil {
		t.Fatalf("failed to create router: %v", err)
	}

	// Create service and install
	engine := template.NewTemplateEngineWithFs(memFs)
	pipeline := rendering.NewRenderingPipeline(engine)
	manifestGenerator := manifest.NewManifestGenerator()
	service := install.NewService(memFs, pipeline, manifestGenerator)

	opts := install.InstallOptions{
		DryRun:  false,
		Verbose: true,
	}

	result, err := service.Install(mergedProfile, projectDir, router, opts)
	if err != nil {
		t.Fatalf("install failed: %v", err)
	}

	// Verify both files were processed
	if result.FilesProcessed != 2 {
		t.Errorf("expected 2 files processed, got %d", result.FilesProcessed)
	}

	// Verify main.md was installed with reference() resolved
	mainPath := projectDir + "/.claude/main.md"
	mainContent, err := afero.ReadFile(memFs, mainPath)
	if err != nil {
		t.Fatalf("failed to read main.md: %v", err)
	}

	// The reference() function should have resolved to @.claude/config.md
	if !strings.Contains(string(mainContent), "@.claude/config.md") {
		t.Errorf("expected main.md to contain '@.claude/config.md', got %q", string(mainContent))
	}
}
