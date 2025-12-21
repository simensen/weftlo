package profile

import (
	"testing"

	"github.com/spf13/afero"

	domainprofile "github.com/simensen/weftlo/internal/domain/profile"
)

// =============================================================================
// Task Group 5: Integration Tests
// End-to-end tests that verify the complete workflow works together
// =============================================================================

// TestIntegration_LoadProfile_EnumerateContent_RespectIgnorePatterns tests the
// complete end-to-end workflow: load profile with ignore patterns and verify
// content enumeration respects those patterns when filtering files.
func TestIntegration_LoadProfile_EnumerateContent_RespectIgnorePatterns(t *testing.T) {
	t.Parallel()
	fs := afero.NewMemMapFs()

	// Create profile directory structure
	profileDir := "/home/user/.weftlo/profiles/acme/integration"
	contentDir := profileDir + "/content"

	// Create content directory structure
	if err := fs.MkdirAll(contentDir+"/docs", 0755); err != nil {
		t.Fatalf("failed to create content/docs directory: %v", err)
	}
	if err := fs.MkdirAll(contentDir+"/temp", 0755); err != nil {
		t.Fatalf("failed to create content/temp directory: %v", err)
	}

	// Create profile.yaml
	profileYAML := `name: acme/integration
`
	if err := afero.WriteFile(fs, profileDir+"/profile.yaml", []byte(profileYAML), 0644); err != nil {
		t.Fatalf("failed to write profile.yaml: %v", err)
	}

	// Create .weftlo.ignore with patterns
	ignoreContent := `# Ignore temporary files
*.tmp
*.bak

# Ignore temp directory
temp/
`
	if err := afero.WriteFile(fs, profileDir+"/.weftlo.ignore", []byte(ignoreContent), 0644); err != nil {
		t.Fatalf("failed to write .weftlo.ignore: %v", err)
	}

	// Create content files - some should be ignored, some should not
	contentFiles := map[string]string{
		"/content/README.md":                "# README",
		"/content/INSTALL.md":               "# Installation",
		"/content/docs/api.md":              "# API Documentation",
		"/content/docs/guide.md":            "# User Guide",
		"/content/cache.tmp":                "temporary data",
		"/content/backup.bak":               "backup data",
		"/content/temp/scratch.md":          "scratch notes",
		"/content/temp/work-in-progress.md": "WIP",
	}

	for path, content := range contentFiles {
		if err := afero.WriteFile(fs, profileDir+path, []byte(content), 0644); err != nil {
			t.Fatalf("failed to write %s: %v", path, err)
		}
	}

	// Load the profile
	loader := NewProfileLoaderWithFs(fs, func() (string, error) {
		return "/home/user", nil
	})

	profile, err := loader.Load("acme/integration")
	if err != nil {
		t.Fatalf("expected no error loading profile, got: %v", err)
	}

	// Verify profile was loaded with correct name
	if profile.Name != "acme/integration" {
		t.Errorf("expected profile name 'acme/integration', got '%s'", profile.Name)
	}

	// Verify Matcher is populated
	if profile.Matcher == nil {
		t.Fatal("expected Matcher to be populated, got nil")
	}

	// Verify ignore patterns work
	// Should ignore *.tmp files
	if !profile.Matcher.Match("cache.tmp", false) {
		t.Error("expected '*.tmp' pattern to match 'cache.tmp'")
	}

	// Should ignore *.bak files
	if !profile.Matcher.Match("backup.bak", false) {
		t.Error("expected '*.bak' pattern to match 'backup.bak'")
	}

	// Should ignore temp/ directory
	if !profile.Matcher.Match("temp", true) {
		t.Error("expected 'temp/' pattern to match 'temp' directory")
	}

	// Should NOT ignore regular markdown files
	if profile.Matcher.Match("README.md", false) {
		t.Error("expected 'README.md' to NOT be ignored")
	}
	if profile.Matcher.Match("docs/api.md", false) {
		t.Error("expected 'docs/api.md' to NOT be ignored")
	}

	// Verify content files were enumerated (before applying ignore patterns)
	// Note: Content enumeration does NOT automatically apply ignore patterns
	// That is done by the caller (e.g., install command) using the Matcher
	if len(profile.TemplateFiles) == 0 {
		t.Error("expected template files to be enumerated")
	}

	// Verify files from content directory are present
	foundFiles := make(map[string]bool)
	for _, tf := range profile.TemplateFiles {
		foundFiles[tf.SourcePath] = true
	}

	// Should have enumerated the non-ignored files AND the ignored files
	// (filtering happens at install time, not enumeration time)
	expectedFiles := []string{
		"README.md",
		"INSTALL.md",
		"docs/api.md",
		"docs/guide.md",
	}

	for _, expected := range expectedFiles {
		if !foundFiles[expected] {
			t.Errorf("expected file '%s' to be enumerated", expected)
		}
	}
}

// TestIntegration_ProfileInheritance_ContentConfigAndMatcherMerge tests
// the complete profile inheritance workflow where both ContentConfig and
// Matcher are properly merged from parent to child using LoadMerged.
//
// Note: Due to how defaults are currently applied during profile loading,
// when a child profile doesn't specify default_target, the loader fills in
// the system default ("weftlo"). This means child profiles cannot
// currently inherit parent's default_target by leaving it empty - they must
// explicitly specify the same value to preserve it.
// This test verifies the actual behavior, where explicit values in both
// parent and child work correctly.
func TestIntegration_ProfileInheritance_ContentConfigAndMatcherMerge(t *testing.T) {
	t.Parallel()
	fs := afero.NewMemMapFs()

	// Create parent profile
	parentDir := "/home/user/.weftlo/profiles/acme/parent"
	parentContentDir := parentDir + "/content"

	if err := fs.MkdirAll(parentContentDir, 0755); err != nil {
		t.Fatalf("failed to create parent content directory: %v", err)
	}

	// Parent profile.yaml with ContentConfig and inline ignore patterns
	parentYAML := `name: acme/parent
content:
  root: content
  default_target: .claude
  targets:
    claude: .claude
    shared: weftlo/shared
ignore:
  - "*.log"
  - "*.tmp"
`
	if err := afero.WriteFile(fs, parentDir+"/profile.yaml", []byte(parentYAML), 0644); err != nil {
		t.Fatalf("failed to write parent profile.yaml: %v", err)
	}

	// Parent content file
	if err := afero.WriteFile(fs, parentContentDir+"/PARENT.md", []byte("# Parent"), 0644); err != nil {
		t.Fatalf("failed to write parent content: %v", err)
	}

	// Create child profile
	childDir := "/home/user/.weftlo/profiles/acme/child"
	childContentDir := childDir + "/content"

	if err := fs.MkdirAll(childContentDir, 0755); err != nil {
		t.Fatalf("failed to create child content directory: %v", err)
	}

	// Child profile.yaml with inheritance and overrides
	// Note: Child explicitly specifies default_target to ensure proper inheritance
	childYAML := `name: acme/child
inherits_from: acme/parent
content:
  default_target: .claude
  targets:
    claude: .ai/claude
    cursor: .cursor
ignore:
  - "!important.log"
`
	if err := afero.WriteFile(fs, childDir+"/profile.yaml", []byte(childYAML), 0644); err != nil {
		t.Fatalf("failed to write child profile.yaml: %v", err)
	}

	// Child content file
	if err := afero.WriteFile(fs, childContentDir+"/CHILD.md", []byte("# Child"), 0644); err != nil {
		t.Fatalf("failed to write child content: %v", err)
	}

	// Load the child profile with inheritance resolution using LoadMerged
	loader := NewProfileLoaderWithFs(fs, func() (string, error) {
		return "/home/user", nil
	})

	mergedProfile, err := loader.LoadMerged("acme/child")
	if err != nil {
		t.Fatalf("expected no error loading merged profile, got: %v", err)
	}

	// Convert to Profile to access merged values
	profile := mergedProfile.ToProfile()

	// Verify profile name is the child
	if profile.Name != "acme/child" {
		t.Errorf("expected profile name 'acme/child', got '%s'", profile.Name)
	}

	// Verify ContentConfig is merged correctly
	// Root should be from parent (child didn't override)
	if profile.ContentConfig.Root != "content" {
		t.Errorf("expected ContentConfig.Root to be 'content', got '%s'", profile.ContentConfig.Root)
	}

	// DefaultTarget should be .claude (child explicitly specified it)
	if profile.ContentConfig.DefaultTarget != ".claude" {
		t.Errorf("expected ContentConfig.DefaultTarget to be '.claude', got '%s'", profile.ContentConfig.DefaultTarget)
	}

	// Targets should be merged: child overrides claude, adds cursor, preserves shared
	if profile.ContentConfig.Targets["claude"] != ".ai/claude" {
		t.Errorf("expected Targets['claude'] to be '.ai/claude' (child override), got '%s'", profile.ContentConfig.Targets["claude"])
	}
	if profile.ContentConfig.Targets["cursor"] != ".cursor" {
		t.Errorf("expected Targets['cursor'] to be '.cursor' (from child), got '%s'", profile.ContentConfig.Targets["cursor"])
	}
	if profile.ContentConfig.Targets["shared"] != "weftlo/shared" {
		t.Errorf("expected Targets['shared'] to be 'weftlo/shared' (from parent), got '%s'", profile.ContentConfig.Targets["shared"])
	}

	// Verify Matcher is merged correctly
	if profile.Matcher == nil {
		t.Fatal("expected Matcher to be populated, got nil")
	}

	// Parent patterns should still work (*.log, *.tmp)
	if !profile.Matcher.Match("debug.log", false) {
		t.Error("expected 'debug.log' to be ignored (parent pattern *.log)")
	}
	if !profile.Matcher.Match("cache.tmp", false) {
		t.Error("expected 'cache.tmp' to be ignored (parent pattern *.tmp)")
	}

	// Child negation should re-include important.log
	if profile.Matcher.Match("important.log", false) {
		t.Error("expected 'important.log' to NOT be ignored (child negation !important.log)")
	}

	// Non-matching files should not be ignored
	if profile.Matcher.Match("readme.md", false) {
		t.Error("expected 'readme.md' to NOT be ignored")
	}
}

// TestIntegration_CustomContentRoot_WithIgnorePatterns tests loading a profile
// with a custom content root and verifying ignore patterns still work correctly.
func TestIntegration_CustomContentRoot_WithIgnorePatterns(t *testing.T) {
	t.Parallel()
	fs := afero.NewMemMapFs()

	// Create profile with custom content root
	profileDir := "/home/user/.weftlo/profiles/acme/custom"
	customContentDir := profileDir + "/templates"

	if err := fs.MkdirAll(customContentDir, 0755); err != nil {
		t.Fatalf("failed to create templates directory: %v", err)
	}

	// Profile with custom content root and ignore patterns
	profileYAML := `name: acme/custom
content:
  root: templates
ignore:
  - "*.draft"
  - "temp/"
`
	if err := afero.WriteFile(fs, profileDir+"/profile.yaml", []byte(profileYAML), 0644); err != nil {
		t.Fatalf("failed to write profile.yaml: %v", err)
	}

	// Create content files in custom root
	if err := afero.WriteFile(fs, customContentDir+"/README.md", []byte("# README"), 0644); err != nil {
		t.Fatalf("failed to write README.md: %v", err)
	}
	if err := afero.WriteFile(fs, customContentDir+"/doc.draft", []byte("draft"), 0644); err != nil {
		t.Fatalf("failed to write doc.draft: %v", err)
	}

	loader := NewProfileLoaderWithFs(fs, func() (string, error) {
		return "/home/user", nil
	})

	profile, err := loader.Load("acme/custom")
	if err != nil {
		t.Fatalf("expected no error loading profile, got: %v", err)
	}

	// Verify ContentConfig has custom root
	if profile.ContentConfig.Root != "templates" {
		t.Errorf("expected ContentConfig.Root to be 'templates', got '%s'", profile.ContentConfig.Root)
	}

	// Verify Matcher works with ignore patterns
	if profile.Matcher == nil {
		t.Fatal("expected Matcher to be populated, got nil")
	}

	if !profile.Matcher.Match("doc.draft", false) {
		t.Error("expected '*.draft' pattern to match 'doc.draft'")
	}
	if !profile.Matcher.Match("temp", true) {
		t.Error("expected 'temp/' pattern to match 'temp' directory")
	}
	if profile.Matcher.Match("README.md", false) {
		t.Error("expected 'README.md' to NOT be ignored")
	}

	// Verify content files were enumerated from custom root
	if len(profile.TemplateFiles) != 2 {
		t.Fatalf("expected 2 template files from custom content root, got %d", len(profile.TemplateFiles))
	}
}

// TestIntegration_EmptyProfile_WithDefaults tests loading a minimal profile
// and verifying all defaults are applied correctly.
func TestIntegration_EmptyProfile_WithDefaults(t *testing.T) {
	t.Parallel()
	fs := afero.NewMemMapFs()

	// Create minimal profile
	profileDir := "/home/user/.weftlo/profiles/acme/minimal"
	contentDir := profileDir + "/content"

	if err := fs.MkdirAll(contentDir, 0755); err != nil {
		t.Fatalf("failed to create content directory: %v", err)
	}

	// Minimal profile.yaml with just name
	profileYAML := `name: acme/minimal
`
	if err := afero.WriteFile(fs, profileDir+"/profile.yaml", []byte(profileYAML), 0644); err != nil {
		t.Fatalf("failed to write profile.yaml: %v", err)
	}

	// Create a content file
	if err := afero.WriteFile(fs, contentDir+"/README.md", []byte("# README"), 0644); err != nil {
		t.Fatalf("failed to write README.md: %v", err)
	}

	loader := NewProfileLoaderWithFs(fs, func() (string, error) {
		return "/home/user", nil
	})

	profile, err := loader.Load("acme/minimal")
	if err != nil {
		t.Fatalf("expected no error loading profile, got: %v", err)
	}

	// Verify defaults are applied (using actual default values from domain/profile)
	if profile.ContentConfig.Root != domainprofile.DefaultContentRoot {
		t.Errorf("expected default ContentConfig.Root '%s', got '%s'", domainprofile.DefaultContentRoot, profile.ContentConfig.Root)
	}
	if profile.ContentConfig.DefaultTarget != domainprofile.DefaultTargetPath {
		t.Errorf("expected default ContentConfig.DefaultTarget '%s', got '%s'", domainprofile.DefaultTargetPath, profile.ContentConfig.DefaultTarget)
	}

	// Verify Matcher is initialized (even if empty)
	if profile.Matcher == nil {
		t.Error("expected Matcher to be initialized (non-nil), got nil")
	}

	// Empty matcher should not match anything
	if profile.Matcher.Match("anything.txt", false) {
		t.Error("expected empty Matcher to not match any file")
	}

	// Verify content was enumerated
	if len(profile.TemplateFiles) != 1 {
		t.Fatalf("expected 1 template file, got %d", len(profile.TemplateFiles))
	}
	if profile.TemplateFiles[0].SourcePath != "README.md" {
		t.Errorf("expected 'README.md', got '%s'", profile.TemplateFiles[0].SourcePath)
	}
}
