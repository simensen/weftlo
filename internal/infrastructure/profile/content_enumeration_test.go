package profile

import (
	"testing"

	"github.com/spf13/afero"
)

// =============================================================================
// Task Group 3: Content Enumeration Tests
// Tests for enumerateContentFiles walking only content root directory
// =============================================================================

// TestEnumerateContentFiles_WalksOnlyContentRoot tests that enumerateContentFiles
// walks only the content root directory (default: "content") and not the entire
// profile directory.
func TestEnumerateContentFiles_WalksOnlyContentRoot(t *testing.T) {
	t.Parallel()
	fs := afero.NewMemMapFs()

	// Create profile directory structure with files OUTSIDE and INSIDE content root
	profileDir := "/home/user/.weftlo/profiles/acme/test"
	contentDir := profileDir + "/content"

	// Create directories
	if err := fs.MkdirAll(contentDir+"/docs", 0755); err != nil {
		t.Fatalf("failed to create content/docs directory: %v", err)
	}

	// Files INSIDE content root (should be enumerated)
	if err := afero.WriteFile(fs, contentDir+"/README.md", []byte("# README"), 0644); err != nil {
		t.Fatalf("failed to write content/README.md: %v", err)
	}
	if err := afero.WriteFile(fs, contentDir+"/docs/api.md", []byte("# API"), 0644); err != nil {
		t.Fatalf("failed to write content/docs/api.md: %v", err)
	}

	// Files OUTSIDE content root (should NOT be enumerated)
	if err := afero.WriteFile(fs, profileDir+"/profile.yaml", []byte("name: acme/test"), 0644); err != nil {
		t.Fatalf("failed to write profile.yaml: %v", err)
	}
	if err := afero.WriteFile(fs, profileDir+"/README.md", []byte("# Profile README"), 0644); err != nil {
		t.Fatalf("failed to write README.md: %v", err)
	}
	if err := afero.WriteFile(fs, profileDir+"/docs.md", []byte("# Docs"), 0644); err != nil {
		t.Fatalf("failed to write docs.md: %v", err)
	}

	loader := NewProfileLoaderWithFs(fs, func() (string, error) {
		return "/home/user", nil
	})

	profile, err := loader.Load("acme/test")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// Should only have files from content root
	if len(profile.TemplateFiles) != 2 {
		t.Fatalf("expected 2 template files (from content root), got %d", len(profile.TemplateFiles))
	}

	// Verify the files are from content root
	expectedPaths := map[string]bool{
		"README.md":   true,
		"docs/api.md": true,
	}

	for _, tf := range profile.TemplateFiles {
		if !expectedPaths[tf.SourcePath] {
			t.Errorf("unexpected file: %s", tf.SourcePath)
		}
		delete(expectedPaths, tf.SourcePath)
	}

	for path := range expectedPaths {
		t.Errorf("missing expected file: %s", path)
	}
}

// TestEnumerateContentFiles_EmptyListWhenContentRootMissing tests that when
// the content root directory does not exist, an empty file list is returned
// (not an error).
func TestEnumerateContentFiles_EmptyListWhenContentRootMissing(t *testing.T) {
	t.Parallel()
	fs := afero.NewMemMapFs()

	// Create profile directory WITHOUT content subdirectory
	profileDir := "/home/user/.weftlo/profiles/acme/no-content"
	if err := fs.MkdirAll(profileDir, 0755); err != nil {
		t.Fatalf("failed to create profile directory: %v", err)
	}

	// Create profile.yaml at profile root
	if err := afero.WriteFile(fs, profileDir+"/profile.yaml", []byte("name: acme/no-content"), 0644); err != nil {
		t.Fatalf("failed to write profile.yaml: %v", err)
	}

	// Create some files at profile root (should NOT be enumerated)
	if err := afero.WriteFile(fs, profileDir+"/README.md", []byte("# README"), 0644); err != nil {
		t.Fatalf("failed to write README.md: %v", err)
	}

	loader := NewProfileLoaderWithFs(fs, func() (string, error) {
		return "/home/user", nil
	})

	profile, err := loader.Load("acme/no-content")

	// Should NOT return an error
	if err != nil {
		t.Fatalf("expected no error when content root missing, got: %v", err)
	}

	// Should return empty file list
	if len(profile.TemplateFiles) != 0 {
		t.Errorf("expected 0 template files when content root missing, got %d", len(profile.TemplateFiles))
	}
}

// TestEnumerateContentFiles_PathsRelativeToContentRoot tests that paths returned
// by enumerateContentFiles are relative to the content root, not the profile directory.
func TestEnumerateContentFiles_PathsRelativeToContentRoot(t *testing.T) {
	t.Parallel()
	fs := afero.NewMemMapFs()

	// Create profile with content root and nested files
	profileDir := "/home/user/.weftlo/profiles/acme/nested"
	contentDir := profileDir + "/content"

	// Create nested directory structure inside content
	if err := fs.MkdirAll(contentDir+"/docs/api/v1", 0755); err != nil {
		t.Fatalf("failed to create nested directories: %v", err)
	}

	// Create files at various depths
	files := map[string]string{
		"/content/README.md":                "# README",
		"/content/docs/index.md":            "# Docs",
		"/content/docs/api/overview.md":     "# API Overview",
		"/content/docs/api/v1/endpoints.md": "# Endpoints",
	}

	for path, content := range files {
		if err := afero.WriteFile(fs, profileDir+path, []byte(content), 0644); err != nil {
			t.Fatalf("failed to write %s: %v", path, err)
		}
	}

	loader := NewProfileLoaderWithFs(fs, func() (string, error) {
		return "/home/user", nil
	})

	profile, err := loader.Load("acme/nested")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// Paths should be relative to content root (not include "content/" prefix)
	expectedPaths := map[string]bool{
		"README.md":                true,
		"docs/index.md":            true,
		"docs/api/overview.md":     true,
		"docs/api/v1/endpoints.md": true,
	}

	if len(profile.TemplateFiles) != len(expectedPaths) {
		t.Fatalf("expected %d template files, got %d", len(expectedPaths), len(profile.TemplateFiles))
	}

	for _, tf := range profile.TemplateFiles {
		if !expectedPaths[tf.SourcePath] {
			t.Errorf("unexpected path (possibly includes 'content/' prefix): %s", tf.SourcePath)
		}
		delete(expectedPaths, tf.SourcePath)
	}

	for path := range expectedPaths {
		t.Errorf("missing expected path: %s", path)
	}
}

// TestEnumerateContentFiles_DotfilesInsideContentRootEnumerated tests that dotfiles
// inside the content root ARE enumerated (e.g., .claude/CLAUDE.md, .env.template).
func TestEnumerateContentFiles_DotfilesInsideContentRootEnumerated(t *testing.T) {
	t.Parallel()
	fs := afero.NewMemMapFs()

	// Create profile with content root containing dotfiles
	profileDir := "/home/user/.weftlo/profiles/acme/with-dotfiles"
	contentDir := profileDir + "/content"

	// Create .claude directory inside content
	if err := fs.MkdirAll(contentDir+"/.claude", 0755); err != nil {
		t.Fatalf("failed to create .claude directory: %v", err)
	}

	// Create dotfiles inside content root (should be enumerated)
	dotfiles := map[string]string{
		"/content/.gitignore":        "node_modules/",
		"/content/.env.template":     "API_KEY=",
		"/content/.claude/CLAUDE.md": "# Claude Configuration",
	}

	for path, content := range dotfiles {
		if err := afero.WriteFile(fs, profileDir+path, []byte(content), 0644); err != nil {
			t.Fatalf("failed to write %s: %v", path, err)
		}
	}

	// Also create a regular file for completeness
	if err := afero.WriteFile(fs, contentDir+"/README.md", []byte("# README"), 0644); err != nil {
		t.Fatalf("failed to write README.md: %v", err)
	}

	loader := NewProfileLoaderWithFs(fs, func() (string, error) {
		return "/home/user", nil
	})

	profile, err := loader.Load("acme/with-dotfiles")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// All files including dotfiles should be enumerated
	expectedPaths := map[string]bool{
		".gitignore":        true,
		".env.template":     true,
		".claude/CLAUDE.md": true,
		"README.md":         true,
	}

	if len(profile.TemplateFiles) != len(expectedPaths) {
		t.Fatalf("expected %d template files (including dotfiles), got %d", len(expectedPaths), len(profile.TemplateFiles))
	}

	for _, tf := range profile.TemplateFiles {
		if !expectedPaths[tf.SourcePath] {
			t.Errorf("unexpected file: %s", tf.SourcePath)
		}
		delete(expectedPaths, tf.SourcePath)
	}

	for path := range expectedPaths {
		t.Errorf("missing expected dotfile: %s", path)
	}
}

// TestEnumerateContentFiles_IsPartialFilterExcludesUnderscoreFiles tests that
// the isPartialFile filter still excludes underscore-prefixed files from enumeration.
func TestEnumerateContentFiles_IsPartialFilterExcludesUnderscoreFiles(t *testing.T) {
	t.Parallel()
	fs := afero.NewMemMapFs()

	// Create profile with content root containing partial and non-partial files
	profileDir := "/home/user/.weftlo/profiles/acme/with-partials"
	contentDir := profileDir + "/content"

	// Create directories including partial directories
	if err := fs.MkdirAll(contentDir+"/_partials", 0755); err != nil {
		t.Fatalf("failed to create _partials directory: %v", err)
	}
	if err := fs.MkdirAll(contentDir+"/docs", 0755); err != nil {
		t.Fatalf("failed to create docs directory: %v", err)
	}

	// Create partial files (should be marked as IsPartial=true)
	partialFiles := map[string]string{
		"/content/_header.md":          "# Header",
		"/content/_partials/footer.md": "# Footer",
		"/content/docs/_internal.md":   "# Internal",
	}

	for path, content := range partialFiles {
		if err := afero.WriteFile(fs, profileDir+path, []byte(content), 0644); err != nil {
			t.Fatalf("failed to write %s: %v", path, err)
		}
	}

	// Create non-partial files
	nonPartialFiles := map[string]string{
		"/content/README.md":     "# README",
		"/content/docs/index.md": "# Index",
	}

	for path, content := range nonPartialFiles {
		if err := afero.WriteFile(fs, profileDir+path, []byte(content), 0644); err != nil {
			t.Fatalf("failed to write %s: %v", path, err)
		}
	}

	loader := NewProfileLoaderWithFs(fs, func() (string, error) {
		return "/home/user", nil
	})

	profile, err := loader.Load("acme/with-partials")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// All 5 files should be enumerated
	if len(profile.TemplateFiles) != 5 {
		t.Fatalf("expected 5 template files, got %d", len(profile.TemplateFiles))
	}

	// Check IsPartial flag for each file
	expectedPartials := map[string]bool{
		"_header.md":          true,
		"_partials/footer.md": true,
		"docs/_internal.md":   true,
		"README.md":           false,
		"docs/index.md":       false,
	}

	for _, tf := range profile.TemplateFiles {
		expectedIsPartial, exists := expectedPartials[tf.SourcePath]
		if !exists {
			t.Errorf("unexpected file: %s", tf.SourcePath)
			continue
		}
		if tf.IsPartial != expectedIsPartial {
			t.Errorf("expected IsPartial=%v for '%s', got %v", expectedIsPartial, tf.SourcePath, tf.IsPartial)
		}
	}
}

// TestEnumerateContentFiles_ProfileYAMLNotEnumerated tests that profile.yaml is
// not enumerated (implicitly - since profile.yaml is outside the content root).
func TestEnumerateContentFiles_ProfileYAMLNotEnumerated(t *testing.T) {
	t.Parallel()
	fs := afero.NewMemMapFs()

	// Create profile with profile.yaml at root and content in content/
	profileDir := "/home/user/.weftlo/profiles/acme/standard"
	contentDir := profileDir + "/content"

	if err := fs.MkdirAll(contentDir, 0755); err != nil {
		t.Fatalf("failed to create content directory: %v", err)
	}

	// Create profile.yaml at profile root (outside content root)
	if err := afero.WriteFile(fs, profileDir+"/profile.yaml", []byte("name: acme/standard"), 0644); err != nil {
		t.Fatalf("failed to write profile.yaml: %v", err)
	}

	// Create a file inside content root
	if err := afero.WriteFile(fs, contentDir+"/README.md", []byte("# README"), 0644); err != nil {
		t.Fatalf("failed to write README.md: %v", err)
	}

	// Also create a profile.yaml INSIDE content to test it IS enumerated there
	// (this would be unusual but should work since we're not filtering profile.yaml anymore)
	if err := afero.WriteFile(fs, contentDir+"/profile.yaml", []byte("# Not a real profile"), 0644); err != nil {
		t.Fatalf("failed to write content/profile.yaml: %v", err)
	}

	loader := NewProfileLoaderWithFs(fs, func() (string, error) {
		return "/home/user", nil
	})

	profile, err := loader.Load("acme/standard")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// Should have 2 files: README.md and profile.yaml (inside content)
	// The profile.yaml at profile root is not enumerated (outside content root)
	if len(profile.TemplateFiles) != 2 {
		t.Fatalf("expected 2 template files, got %d", len(profile.TemplateFiles))
	}

	// Verify the files
	foundREADME := false
	foundProfileYAML := false
	for _, tf := range profile.TemplateFiles {
		switch tf.SourcePath {
		case "README.md":
			foundREADME = true
		case "profile.yaml":
			foundProfileYAML = true
		default:
			t.Errorf("unexpected file: %s", tf.SourcePath)
		}
	}

	if !foundREADME {
		t.Error("expected README.md to be enumerated")
	}
	if !foundProfileYAML {
		t.Error("expected profile.yaml (inside content root) to be enumerated")
	}
}

// TestEnumerateContentFiles_CustomContentRoot tests that enumerateContentFiles
// respects a custom content root specified in the profile configuration.
func TestEnumerateContentFiles_CustomContentRoot(t *testing.T) {
	t.Parallel()
	fs := afero.NewMemMapFs()

	// Create profile with custom content root "templates" instead of "content"
	profileDir := "/home/user/.weftlo/profiles/acme/custom-root"
	customContentDir := profileDir + "/templates"
	defaultContentDir := profileDir + "/content"

	if err := fs.MkdirAll(customContentDir, 0755); err != nil {
		t.Fatalf("failed to create templates directory: %v", err)
	}
	if err := fs.MkdirAll(defaultContentDir, 0755); err != nil {
		t.Fatalf("failed to create content directory: %v", err)
	}

	// Create profile.yaml with custom content root
	profileYAML := `name: acme/custom-root
content:
  root: templates
`
	if err := afero.WriteFile(fs, profileDir+"/profile.yaml", []byte(profileYAML), 0644); err != nil {
		t.Fatalf("failed to write profile.yaml: %v", err)
	}

	// Create file in custom root (should be enumerated)
	if err := afero.WriteFile(fs, customContentDir+"/README.md", []byte("# README"), 0644); err != nil {
		t.Fatalf("failed to write templates/README.md: %v", err)
	}

	// Create file in default root (should NOT be enumerated)
	if err := afero.WriteFile(fs, defaultContentDir+"/OTHER.md", []byte("# Other"), 0644); err != nil {
		t.Fatalf("failed to write content/OTHER.md: %v", err)
	}

	loader := NewProfileLoaderWithFs(fs, func() (string, error) {
		return "/home/user", nil
	})

	profile, err := loader.Load("acme/custom-root")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// Should only have the file from custom content root
	if len(profile.TemplateFiles) != 1 {
		t.Fatalf("expected 1 template file from custom content root, got %d", len(profile.TemplateFiles))
	}

	if profile.TemplateFiles[0].SourcePath != "README.md" {
		t.Errorf("expected README.md from templates/, got %s", profile.TemplateFiles[0].SourcePath)
	}
}
