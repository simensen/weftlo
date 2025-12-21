package testutil_test

import (
	"testing"
	"time"

	"github.com/spf13/afero"

	"github.com/simensen/weftlo/internal/domain/config"
	"github.com/simensen/weftlo/internal/domain/profile"
	"github.com/simensen/weftlo/internal/testutil"
)

// TestCreateTestFilesystem_CreatesFilesWithCorrectContent verifies that
// files are created with the correct content.
func TestCreateTestFilesystem_CreatesFilesWithCorrectContent(t *testing.T) {
	files := map[string]string{
		"/path/to/file1.txt": "content one",
		"/path/to/file2.txt": "content two",
	}

	fs := testutil.CreateTestFilesystem(t, files)

	// Verify each file exists with correct content
	for path, expectedContent := range files {
		content, err := afero.ReadFile(fs, path)
		if err != nil {
			t.Fatalf("failed to read file %s: %v", path, err)
		}
		if string(content) != expectedContent {
			t.Errorf("file %s: expected content %q, got %q", path, expectedContent, string(content))
		}
	}
}

// TestCreateTestFilesystem_CreatesParentDirectories verifies that
// parent directories are automatically created for nested file paths.
func TestCreateTestFilesystem_CreatesParentDirectories(t *testing.T) {
	files := map[string]string{
		"/deeply/nested/directory/structure/file.txt": "nested content",
	}

	fs := testutil.CreateTestFilesystem(t, files)

	// Verify the directory structure was created
	dirs := []string{
		"/deeply",
		"/deeply/nested",
		"/deeply/nested/directory",
		"/deeply/nested/directory/structure",
	}

	for _, dir := range dirs {
		info, err := fs.Stat(dir)
		if err != nil {
			t.Fatalf("expected directory %s to exist, got error: %v", dir, err)
		}
		if !info.IsDir() {
			t.Errorf("expected %s to be a directory", dir)
		}
	}

	// Verify the file exists
	content, err := afero.ReadFile(fs, "/deeply/nested/directory/structure/file.txt")
	if err != nil {
		t.Fatalf("failed to read nested file: %v", err)
	}
	if string(content) != "nested content" {
		t.Errorf("expected content %q, got %q", "nested content", string(content))
	}
}

// TestCreateTestFilesystem_HandlesEmptyMap verifies that an empty map
// returns a valid empty filesystem.
func TestCreateTestFilesystem_HandlesEmptyMap(t *testing.T) {
	files := map[string]string{}

	fs := testutil.CreateTestFilesystem(t, files)

	// Verify filesystem was created and is empty
	if fs == nil {
		t.Fatal("expected non-nil filesystem")
	}

	// Verify root directory is accessible
	_, err := fs.Stat("/")
	if err != nil {
		t.Fatalf("expected root directory to be accessible: %v", err)
	}
}

// TestCreateTestFilesystem_HandlesMultipleFilesInSameDirectory verifies that
// multiple files can be created in the same directory.
func TestCreateTestFilesystem_HandlesMultipleFilesInSameDirectory(t *testing.T) {
	files := map[string]string{
		"/config/file1.yaml": "content1",
		"/config/file2.yaml": "content2",
		"/config/file3.yaml": "content3",
	}

	fs := testutil.CreateTestFilesystem(t, files)

	// Verify all files exist with correct content
	for path, expectedContent := range files {
		content, err := afero.ReadFile(fs, path)
		if err != nil {
			t.Fatalf("failed to read file %s: %v", path, err)
		}
		if string(content) != expectedContent {
			t.Errorf("file %s: expected content %q, got %q", path, expectedContent, string(content))
		}
	}
}

// TestCreateTestFilesystem_HandlesEmptyFileContent verifies that
// files can be created with empty content.
func TestCreateTestFilesystem_HandlesEmptyFileContent(t *testing.T) {
	files := map[string]string{
		"/empty/file.txt": "",
	}

	fs := testutil.CreateTestFilesystem(t, files)

	content, err := afero.ReadFile(fs, "/empty/file.txt")
	if err != nil {
		t.Fatalf("failed to read empty file: %v", err)
	}
	if len(content) != 0 {
		t.Errorf("expected empty content, got %q", string(content))
	}
}

// TestCreateTestProfile_CreatesValidProfile verifies that a profile
// is created with the correct name and template files.
func TestCreateTestProfile_CreatesValidProfile(t *testing.T) {
	p := testutil.CreateTestProfile(t, "vendor/myprofile", "file1.md", "file2.md")

	if p.Name != "vendor/myprofile" {
		t.Errorf("expected name %q, got %q", "vendor/myprofile", p.Name)
	}

	if len(p.TemplateFiles) != 2 {
		t.Fatalf("expected 2 template files, got %d", len(p.TemplateFiles))
	}

	if p.TemplateFiles[0].SourcePath != "file1.md" {
		t.Errorf("expected first file source path %q, got %q", "file1.md", p.TemplateFiles[0].SourcePath)
	}
	if p.TemplateFiles[1].SourcePath != "file2.md" {
		t.Errorf("expected second file source path %q, got %q", "file2.md", p.TemplateFiles[1].SourcePath)
	}
}

// TestCreateTestProfile_MarksFilesAsNonPartial verifies that all template
// files are marked as non-partial by default.
func TestCreateTestProfile_MarksFilesAsNonPartial(t *testing.T) {
	p := testutil.CreateTestProfile(t, "vendor/test", "a.md", "b.md", "c.md")

	for i, tf := range p.TemplateFiles {
		if tf.IsPartial {
			t.Errorf("template file %d (%s): expected IsPartial=false, got true", i, tf.SourcePath)
		}
	}
}

// TestCreateTestProfile_HandlesNoFiles verifies that a profile can be
// created with no template files.
func TestCreateTestProfile_HandlesNoFiles(t *testing.T) {
	p := testutil.CreateTestProfile(t, "vendor/empty")

	if p.Name != "vendor/empty" {
		t.Errorf("expected name %q, got %q", "vendor/empty", p.Name)
	}

	if len(p.TemplateFiles) != 0 {
		t.Errorf("expected 0 template files, got %d", len(p.TemplateFiles))
	}
}

// TestCreateTestMergedProfile_CreatesMergedProfile verifies that a merged
// profile is created from a chain of profiles.
func TestCreateTestMergedProfile_CreatesMergedProfile(t *testing.T) {
	fs := afero.NewMemMapFs()
	parent := testutil.CreateTestProfile(t, "vendor/parent", "shared.md")
	child := testutil.CreateTestProfile(t, "vendor/child", "custom.md")

	merged := testutil.CreateTestMergedProfile(t, fs, "/profiles", parent, child)

	// Verify leaf profile name
	if merged.LeafProfileName() != "vendor/child" {
		t.Errorf("expected leaf profile name %q, got %q", "vendor/child", merged.LeafProfileName())
	}

	// Verify both profiles are in the chain
	names := merged.ProfileNames()
	if len(names) != 2 {
		t.Fatalf("expected 2 profile names, got %d", len(names))
	}
	if names[0] != "vendor/parent" {
		t.Errorf("expected first profile name %q, got %q", "vendor/parent", names[0])
	}
	if names[1] != "vendor/child" {
		t.Errorf("expected second profile name %q, got %q", "vendor/child", names[1])
	}
}

// TestCreateTestMergedProfile_ResolvesSourcePaths verifies that source paths
// can be resolved to absolute paths.
func TestCreateTestMergedProfile_ResolvesSourcePaths(t *testing.T) {
	fs := afero.NewMemMapFs()
	p := testutil.CreateTestProfile(t, "vendor/test", "CLAUDE.md")

	merged := testutil.CreateTestMergedProfile(t, fs, "/profiles", p)

	absPath, found := merged.Resolve("CLAUDE.md")
	if !found {
		t.Fatal("expected to find CLAUDE.md in merged profile")
	}

	expectedPath := "/profiles/vendor/test/content/CLAUDE.md"
	if absPath != expectedPath {
		t.Errorf("expected resolved path %q, got %q", expectedPath, absPath)
	}
}

// TestCreateTestTemplateContext_UsesDefaults verifies that a template context
// is created with sensible default values.
func TestCreateTestTemplateContext_UsesDefaults(t *testing.T) {
	ctx := testutil.CreateTestTemplateContext(t)

	if ctx.Profile != nil {
		t.Errorf("expected default Profile to be nil")
	}

	if ctx.ProjectDir != "/test/project" {
		t.Errorf("expected default ProjectDir %q, got %q", "/test/project", ctx.ProjectDir)
	}

	expectedTime := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
	if !ctx.GeneratedAt.Equal(expectedTime) {
		t.Errorf("expected default GeneratedAt %v, got %v", expectedTime, ctx.GeneratedAt)
	}

	if ctx.ProjectConfig != nil {
		t.Errorf("expected default ProjectConfig to be nil")
	}

	if ctx.GlobalConfig != nil {
		t.Errorf("expected default GlobalConfig to be nil")
	}

	if ctx.Variables != nil {
		t.Errorf("expected default Variables to be nil")
	}
}

// TestCreateTestTemplateContext_WithProfile verifies that the WithProfile
// option correctly sets the profile.
func TestCreateTestTemplateContext_WithProfile(t *testing.T) {
	p := &profile.Profile{Name: "vendor/test"}

	ctx := testutil.CreateTestTemplateContext(t, testutil.WithProfile(p))

	if ctx.Profile != p {
		t.Errorf("expected Profile to be set via option")
	}
	if ctx.Profile.Name != "vendor/test" {
		t.Errorf("expected Profile.Name %q, got %q", "vendor/test", ctx.Profile.Name)
	}
}

// TestCreateTestTemplateContext_WithProjectDir verifies that the WithProjectDir
// option correctly sets the project directory.
func TestCreateTestTemplateContext_WithProjectDir(t *testing.T) {
	ctx := testutil.CreateTestTemplateContext(t, testutil.WithProjectDir("/custom/path"))

	if ctx.ProjectDir != "/custom/path" {
		t.Errorf("expected ProjectDir %q, got %q", "/custom/path", ctx.ProjectDir)
	}
}

// TestCreateTestTemplateContext_WithVariables verifies that the WithVariables
// option correctly sets the variables.
func TestCreateTestTemplateContext_WithVariables(t *testing.T) {
	vars := map[string]interface{}{
		"key1": "value1",
		"key2": 42,
	}

	ctx := testutil.CreateTestTemplateContext(t, testutil.WithVariables(vars))

	if ctx.Variables == nil {
		t.Fatal("expected Variables to be set")
	}
	if ctx.Variables["key1"] != "value1" {
		t.Errorf("expected key1=%q, got %v", "value1", ctx.Variables["key1"])
	}
	if ctx.Variables["key2"] != 42 {
		t.Errorf("expected key2=%d, got %v", 42, ctx.Variables["key2"])
	}
}

// TestCreateTestTemplateContext_WithGeneratedAt verifies that the WithGeneratedAt
// option correctly sets the timestamp.
func TestCreateTestTemplateContext_WithGeneratedAt(t *testing.T) {
	customTime := time.Date(2025, 6, 15, 14, 0, 0, 0, time.UTC)

	ctx := testutil.CreateTestTemplateContext(t, testutil.WithGeneratedAt(customTime))

	if !ctx.GeneratedAt.Equal(customTime) {
		t.Errorf("expected GeneratedAt %v, got %v", customTime, ctx.GeneratedAt)
	}
}

// TestCreateTestTemplateContext_WithProjectConfig verifies that the WithProjectConfig
// option correctly sets the project configuration.
func TestCreateTestTemplateContext_WithProjectConfig(t *testing.T) {
	cfg := &config.ProjectConfig{
		Profiles:      []string{"vendor/test"},
		InstallPrefix: "custom-prefix",
	}

	ctx := testutil.CreateTestTemplateContext(t, testutil.WithProjectConfig(cfg))

	if ctx.ProjectConfig != cfg {
		t.Error("expected ProjectConfig to be set via option")
	}
	if ctx.ProjectConfig.InstallPrefix != "custom-prefix" {
		t.Errorf("expected InstallPrefix %q, got %q", "custom-prefix", ctx.ProjectConfig.InstallPrefix)
	}
}

// TestCreateTestTemplateContext_WithGlobalConfig verifies that the WithGlobalConfig
// option correctly sets the global configuration.
func TestCreateTestTemplateContext_WithGlobalConfig(t *testing.T) {
	cfg := &config.GlobalConfig{
		DefaultProfile: "vendor/default",
	}

	ctx := testutil.CreateTestTemplateContext(t, testutil.WithGlobalConfig(cfg))

	if ctx.GlobalConfig != cfg {
		t.Error("expected GlobalConfig to be set via option")
	}
	if ctx.GlobalConfig.DefaultProfile != "vendor/default" {
		t.Errorf("expected DefaultProfile %q, got %q", "vendor/default", ctx.GlobalConfig.DefaultProfile)
	}
}

// TestCreateTestTemplateContext_MultipleOptions verifies that multiple options
// can be combined to configure the context.
func TestCreateTestTemplateContext_MultipleOptions(t *testing.T) {
	p := &profile.Profile{Name: "vendor/combined"}
	vars := map[string]interface{}{"env": "test"}
	customTime := time.Date(2025, 12, 25, 8, 0, 0, 0, time.UTC)

	ctx := testutil.CreateTestTemplateContext(t,
		testutil.WithProfile(p),
		testutil.WithProjectDir("/combined/project"),
		testutil.WithVariables(vars),
		testutil.WithGeneratedAt(customTime),
	)

	if ctx.Profile != p {
		t.Error("expected Profile to be set")
	}
	if ctx.ProjectDir != "/combined/project" {
		t.Errorf("expected ProjectDir %q, got %q", "/combined/project", ctx.ProjectDir)
	}
	if ctx.Variables["env"] != "test" {
		t.Errorf("expected Variables[env]=%q, got %v", "test", ctx.Variables["env"])
	}
	if !ctx.GeneratedAt.Equal(customTime) {
		t.Errorf("expected GeneratedAt %v, got %v", customTime, ctx.GeneratedAt)
	}
}

// TestCreateTestTemplateContext_LaterOptionsOverrideEarlier verifies that
// when the same option is provided multiple times, the last one wins.
func TestCreateTestTemplateContext_LaterOptionsOverrideEarlier(t *testing.T) {
	ctx := testutil.CreateTestTemplateContext(t,
		testutil.WithProjectDir("/first"),
		testutil.WithProjectDir("/second"),
		testutil.WithProjectDir("/third"),
	)

	if ctx.ProjectDir != "/third" {
		t.Errorf("expected ProjectDir %q (last option should win), got %q", "/third", ctx.ProjectDir)
	}
}
