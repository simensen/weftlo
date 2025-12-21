package profile

import (
	"errors"
	"strconv"
	"testing"

	"github.com/spf13/afero"

	domainconfig "github.com/simensen/weftlo/internal/domain/config"
	domainprofile "github.com/simensen/weftlo/internal/domain/profile"
)

// TestLoadProfileWithValidProfileYAML tests that loading a profile with a valid
// profile.yaml file returns a populated Profile struct with the correct values.
func TestLoadProfileWithValidProfileYAML(t *testing.T) {
	t.Parallel()
	fs := afero.NewMemMapFs()

	// Create profile directory structure
	profileDir := "/home/user/.weftlo/profiles/acme/backend"
	if err := fs.MkdirAll(profileDir, 0755); err != nil {
		t.Fatalf("failed to create profile directory: %v", err)
	}

	// Create valid profile.yaml
	profileYAML := `name: acme/backend
inherits_from: acme/base
`
	if err := afero.WriteFile(fs, profileDir+"/profile.yaml", []byte(profileYAML), 0644); err != nil {
		t.Fatalf("failed to write profile.yaml: %v", err)
	}

	loader := NewProfileLoaderWithFs(fs, func() (string, error) {
		return "/home/user", nil
	})

	profile, err := loader.Load("acme/backend")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if profile.Name != "acme/backend" {
		t.Errorf("expected Name to be 'acme/backend', got '%s'", profile.Name)
	}

	if profile.InheritsFrom != "acme/base" {
		t.Errorf("expected InheritsFrom to be 'acme/base', got '%s'", profile.InheritsFrom)
	}
}

// TestLoadProfileWithoutProfileYAMLInfersName tests that loading a profile without
// a profile.yaml file infers the name from the directory path.
func TestLoadProfileWithoutProfileYAMLInfersName(t *testing.T) {
	t.Parallel()
	fs := afero.NewMemMapFs()

	// Create profile directory structure without profile.yaml
	profileDir := "/home/user/.weftlo/profiles/acme/backend"
	if err := fs.MkdirAll(profileDir, 0755); err != nil {
		t.Fatalf("failed to create profile directory: %v", err)
	}

	loader := NewProfileLoaderWithFs(fs, func() (string, error) {
		return "/home/user", nil
	})

	profile, err := loader.Load("acme/backend")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if profile.Name != "acme/backend" {
		t.Errorf("expected Name to be 'acme/backend', got '%s'", profile.Name)
	}

	if profile.InheritsFrom != "" {
		t.Errorf("expected InheritsFrom to be empty, got '%s'", profile.InheritsFrom)
	}
}

// TestLoadProfileWithMissingNameFieldInfersName tests that loading a profile with
// a profile.yaml that has no name field infers the name from the directory path.
func TestLoadProfileWithMissingNameFieldInfersName(t *testing.T) {
	t.Parallel()
	fs := afero.NewMemMapFs()

	// Create profile directory structure
	profileDir := "/home/user/.weftlo/profiles/acme/backend"
	if err := fs.MkdirAll(profileDir, 0755); err != nil {
		t.Fatalf("failed to create profile directory: %v", err)
	}

	// Create profile.yaml without name field
	profileYAML := `inherits_from: acme/base
`
	if err := afero.WriteFile(fs, profileDir+"/profile.yaml", []byte(profileYAML), 0644); err != nil {
		t.Fatalf("failed to write profile.yaml: %v", err)
	}

	loader := NewProfileLoaderWithFs(fs, func() (string, error) {
		return "/home/user", nil
	})

	profile, err := loader.Load("acme/backend")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if profile.Name != "acme/backend" {
		t.Errorf("expected Name to be 'acme/backend', got '%s'", profile.Name)
	}

	if profile.InheritsFrom != "acme/base" {
		t.Errorf("expected InheritsFrom to be 'acme/base', got '%s'", profile.InheritsFrom)
	}
}

// TestLoadNonExistentProfileReturnsProfileNotFoundError tests that loading a
// non-existent profile returns a ProfileNotFoundError.
func TestLoadNonExistentProfileReturnsProfileNotFoundError(t *testing.T) {
	t.Parallel()
	fs := afero.NewMemMapFs()

	loader := NewProfileLoaderWithFs(fs, func() (string, error) {
		return "/home/user", nil
	})

	_, err := loader.Load("acme/nonexistent")
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var notFoundErr *ProfileNotFoundError
	if !errors.As(err, &notFoundErr) {
		t.Fatalf("expected ProfileNotFoundError, got %T: %v", err, err)
	}

	if notFoundErr.ProfileName != "acme/nonexistent" {
		t.Errorf("expected ProfileName to be 'acme/nonexistent', got '%s'", notFoundErr.ProfileName)
	}

	expectedPath := "/home/user/.weftlo/profiles/acme/nonexistent"
	if notFoundErr.Path != expectedPath {
		t.Errorf("expected Path to be '%s', got '%s'", expectedPath, notFoundErr.Path)
	}
}

// TestLoadProfileWithInvalidYAMLReturnsProfileConfigParseError tests that loading
// a profile with invalid YAML syntax returns a ProfileConfigParseError.
func TestLoadProfileWithInvalidYAMLReturnsProfileConfigParseError(t *testing.T) {
	t.Parallel()
	fs := afero.NewMemMapFs()

	// Create profile directory structure
	profileDir := "/home/user/.weftlo/profiles/acme/backend"
	if err := fs.MkdirAll(profileDir, 0755); err != nil {
		t.Fatalf("failed to create profile directory: %v", err)
	}

	// Create invalid YAML
	invalidYAML := `name: acme/backend
inherits_from: [invalid
  - this is not valid yaml
`
	if err := afero.WriteFile(fs, profileDir+"/profile.yaml", []byte(invalidYAML), 0644); err != nil {
		t.Fatalf("failed to write profile.yaml: %v", err)
	}

	loader := NewProfileLoaderWithFs(fs, func() (string, error) {
		return "/home/user", nil
	})

	_, err := loader.Load("acme/backend")
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var parseErr *ProfileConfigParseError
	if !errors.As(err, &parseErr) {
		t.Fatalf("expected ProfileConfigParseError, got %T: %v", err, err)
	}

	expectedPath := "/home/user/.weftlo/profiles/acme/backend/profile.yaml"
	if parseErr.Path != expectedPath {
		t.Errorf("expected Path to be '%s', got '%s'", expectedPath, parseErr.Path)
	}

	if parseErr.Err == nil {
		t.Error("expected underlying error to be set")
	}
}

// TestProfileNameValidationRejectsInvalidFormat tests that loading a profile
// with an invalid vendor/name format is rejected.
func TestProfileNameValidationRejectsInvalidFormat(t *testing.T) {
	t.Parallel()
	fs := afero.NewMemMapFs()

	loader := NewProfileLoaderWithFs(fs, func() (string, error) {
		return "/home/user", nil
	})

	testCases := []struct {
		name        string
		profileName string
		errContains string
	}{
		{
			name:        "empty profile name",
			profileName: "",
			errContains: "cannot be empty",
		},
		{
			name:        "missing vendor segment",
			profileName: "backend",
			errContains: "vendor/name format",
		},
		{
			name:        "too many segments",
			profileName: "acme/backend/extra",
			errContains: "vendor/name format",
		},
		{
			name:        "empty vendor segment",
			profileName: "/backend",
			errContains: "vendor segment cannot be empty",
		},
		{
			name:        "empty name segment",
			profileName: "acme/",
			errContains: "name segment cannot be empty",
		},
		{
			name:        "invalid vendor characters",
			profileName: "ac.me/backend",
			errContains: "invalid characters",
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := loader.Load(tc.profileName)
			if err == nil {
				t.Fatal("expected error, got nil")
			}

			// Verify it's a validation error from domainconfig.ValidateProfileName
			errStr := err.Error()
			if !contains(errStr, tc.errContains) {
				t.Errorf("expected error to contain '%s', got: %s", tc.errContains, errStr)
			}
		})
	}
}

// contains checks if a string contains a substring.
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && searchSubstring(s, substr)))
}

func searchSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// Ensure the tests use the correct import
var _ = domainconfig.ValidateProfileName

// =============================================================================
// Task Group 3: Template File Enumeration Tests
// =============================================================================

// TestEnumerateFilesCreatesTemplateFileEntriesWithCorrectSourcePath tests that
// enumerating files creates TemplateFile entries with the correct SourcePath.
func TestEnumerateFilesCreatesTemplateFileEntriesWithCorrectSourcePath(t *testing.T) {
	t.Parallel()
	fs := afero.NewMemMapFs()

	// Create profile directory structure with template files
	profileDir := "/home/user/.weftlo/profiles/acme/backend"
	if err := fs.MkdirAll(profileDir+"/content/standards", 0755); err != nil {
		t.Fatalf("failed to create standards directory: %v", err)
	}

	// Create template files
	if err := afero.WriteFile(fs, profileDir+"/content/README.md", []byte("# README"), 0644); err != nil {
		t.Fatalf("failed to write README.md: %v", err)
	}
	if err := afero.WriteFile(fs, profileDir+"/content/standards/api.md", []byte("# API Standards"), 0644); err != nil {
		t.Fatalf("failed to write standards/api.md: %v", err)
	}

	loader := NewProfileLoaderWithFs(fs, func() (string, error) {
		return "/home/user", nil
	})

	profile, err := loader.Load("acme/backend")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// Verify template files were enumerated
	if len(profile.TemplateFiles) != 2 {
		t.Fatalf("expected 2 template files, got %d", len(profile.TemplateFiles))
	}

	// Check that SourcePaths are correct (order may vary, so check both exist)
	sourcePaths := make(map[string]bool)
	for _, tf := range profile.TemplateFiles {
		sourcePaths[tf.SourcePath] = true
	}

	if !sourcePaths["README.md"] {
		t.Error("expected SourcePath 'README.md' to exist")
	}
	if !sourcePaths["standards/api.md"] {
		t.Error("expected SourcePath 'standards/api.md' to exist")
	}
}

// TestProfileYAMLIsExcludedFromEnumeratedTemplateFiles tests that profile.yaml
// is excluded from the enumerated template files.
func TestProfileYAMLIsExcludedFromEnumeratedTemplateFiles(t *testing.T) {
	t.Parallel()
	fs := afero.NewMemMapFs()

	// Create profile directory structure
	profileDir := "/home/user/.weftlo/profiles/acme/backend"
	if err := fs.MkdirAll(profileDir, 0755); err != nil {
		t.Fatalf("failed to create profile directory: %v", err)
	}

	// Create profile.yaml (metadata, not a template)
	profileYAML := `name: acme/backend
`
	if err := afero.WriteFile(fs, profileDir+"/profile.yaml", []byte(profileYAML), 0644); err != nil {
		t.Fatalf("failed to write profile.yaml: %v", err)
	}

	// Create a template file
	if err := afero.WriteFile(fs, profileDir+"/content/README.md", []byte("# README"), 0644); err != nil {
		t.Fatalf("failed to write README.md: %v", err)
	}

	loader := NewProfileLoaderWithFs(fs, func() (string, error) {
		return "/home/user", nil
	})

	profile, err := loader.Load("acme/backend")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// Verify only README.md is in template files (profile.yaml excluded)
	if len(profile.TemplateFiles) != 1 {
		t.Fatalf("expected 1 template file, got %d", len(profile.TemplateFiles))
	}

	if profile.TemplateFiles[0].SourcePath != "README.md" {
		t.Errorf("expected SourcePath to be 'README.md', got '%s'", profile.TemplateFiles[0].SourcePath)
	}

	// Verify profile.yaml is not in template files
	for _, tf := range profile.TemplateFiles {
		if tf.SourcePath == "profile.yaml" {
			t.Error("profile.yaml should not be included in template files")
		}
	}
}

// TestFilesPrefixedWithUnderscoreHaveIsPartialTrue tests that files prefixed
// with underscore have IsPartial set to true.
func TestFilesPrefixedWithUnderscoreHaveIsPartialTrue(t *testing.T) {
	t.Parallel()
	fs := afero.NewMemMapFs()

	// Create profile directory structure
	profileDir := "/home/user/.weftlo/profiles/acme/backend"
	if err := fs.MkdirAll(profileDir, 0755); err != nil {
		t.Fatalf("failed to create profile directory: %v", err)
	}

	// Create partial file (prefixed with underscore)
	if err := afero.WriteFile(fs, profileDir+"/content/_partial.md", []byte("# Partial"), 0644); err != nil {
		t.Fatalf("failed to write _partial.md: %v", err)
	}

	// Create regular file
	if err := afero.WriteFile(fs, profileDir+"/content/regular.md", []byte("# Regular"), 0644); err != nil {
		t.Fatalf("failed to write regular.md: %v", err)
	}

	loader := NewProfileLoaderWithFs(fs, func() (string, error) {
		return "/home/user", nil
	})

	profile, err := loader.Load("acme/backend")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// Verify template files were enumerated
	if len(profile.TemplateFiles) != 2 {
		t.Fatalf("expected 2 template files, got %d", len(profile.TemplateFiles))
	}

	// Check IsPartial flag for each file
	for _, tf := range profile.TemplateFiles {
		if tf.SourcePath == "_partial.md" {
			if !tf.IsPartial {
				t.Errorf("expected '_partial.md' to have IsPartial=true, got false")
			}
		} else if tf.SourcePath == "regular.md" {
			if tf.IsPartial {
				t.Errorf("expected 'regular.md' to have IsPartial=false, got true")
			}
		}
	}
}

// TestDirectoriesPrefixedWithUnderscoreMarkAllContainedFilesAsPartial tests that
// directories prefixed with underscore mark all contained files as partial.
func TestDirectoriesPrefixedWithUnderscoreMarkAllContainedFilesAsPartial(t *testing.T) {
	t.Parallel()
	fs := afero.NewMemMapFs()

	// Create profile directory structure with underscore-prefixed directory
	profileDir := "/home/user/.weftlo/profiles/acme/backend"
	if err := fs.MkdirAll(profileDir+"/content/_partials", 0755); err != nil {
		t.Fatalf("failed to create _partials directory: %v", err)
	}
	if err := fs.MkdirAll(profileDir+"/content/_partials/nested", 0755); err != nil {
		t.Fatalf("failed to create _partials/nested directory: %v", err)
	}

	// Create files inside underscore-prefixed directory
	if err := afero.WriteFile(fs, profileDir+"/content/_partials/header.md", []byte("# Header"), 0644); err != nil {
		t.Fatalf("failed to write _partials/header.md: %v", err)
	}
	if err := afero.WriteFile(fs, profileDir+"/content/_partials/nested/footer.md", []byte("# Footer"), 0644); err != nil {
		t.Fatalf("failed to write _partials/nested/footer.md: %v", err)
	}

	// Create a regular file outside the underscore directory
	if err := afero.WriteFile(fs, profileDir+"/content/regular.md", []byte("# Regular"), 0644); err != nil {
		t.Fatalf("failed to write regular.md: %v", err)
	}

	loader := NewProfileLoaderWithFs(fs, func() (string, error) {
		return "/home/user", nil
	})

	profile, err := loader.Load("acme/backend")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// Verify template files were enumerated
	if len(profile.TemplateFiles) != 3 {
		t.Fatalf("expected 3 template files, got %d", len(profile.TemplateFiles))
	}

	// Check IsPartial flag for each file
	for _, tf := range profile.TemplateFiles {
		switch tf.SourcePath {
		case "_partials/header.md":
			if !tf.IsPartial {
				t.Errorf("expected '_partials/header.md' to have IsPartial=true, got false")
			}
		case "_partials/nested/footer.md":
			if !tf.IsPartial {
				t.Errorf("expected '_partials/nested/footer.md' to have IsPartial=true, got false")
			}
		case "regular.md":
			if tf.IsPartial {
				t.Errorf("expected 'regular.md' to have IsPartial=false, got true")
			}
		default:
			t.Errorf("unexpected SourcePath: %s", tf.SourcePath)
		}
	}
}

// TestSourcePathUsesForwardSlashesForCrossPlatformConsistency tests that
// SourcePath uses forward slashes for cross-platform consistency.
func TestSourcePathUsesForwardSlashesForCrossPlatformConsistency(t *testing.T) {
	t.Parallel()
	fs := afero.NewMemMapFs()

	// Create profile directory structure with nested directories
	profileDir := "/home/user/.weftlo/profiles/acme/backend"
	if err := fs.MkdirAll(profileDir+"/content/standards/api", 0755); err != nil {
		t.Fatalf("failed to create nested directories: %v", err)
	}

	// Create a deeply nested file
	if err := afero.WriteFile(fs, profileDir+"/content/standards/api/endpoints.md", []byte("# Endpoints"), 0644); err != nil {
		t.Fatalf("failed to write deeply nested file: %v", err)
	}

	loader := NewProfileLoaderWithFs(fs, func() (string, error) {
		return "/home/user", nil
	})

	profile, err := loader.Load("acme/backend")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// Verify template files were enumerated
	if len(profile.TemplateFiles) != 1 {
		t.Fatalf("expected 1 template file, got %d", len(profile.TemplateFiles))
	}

	// Verify forward slashes are used
	expectedPath := "standards/api/endpoints.md"
	if profile.TemplateFiles[0].SourcePath != expectedPath {
		t.Errorf("expected SourcePath '%s', got '%s'", expectedPath, profile.TemplateFiles[0].SourcePath)
	}

	// Verify no backslashes in path
	for _, tf := range profile.TemplateFiles {
		for _, c := range tf.SourcePath {
			if c == '\\' {
				t.Errorf("SourcePath should not contain backslashes: %s", tf.SourcePath)
				break
			}
		}
	}
}

// TestTargetPathAndChecksumAreLeftAsEmptyStrings tests that TargetPath and
// Checksum are left as empty strings (stubs for downstream computation).
func TestTargetPathAndChecksumAreLeftAsEmptyStrings(t *testing.T) {
	t.Parallel()
	fs := afero.NewMemMapFs()

	// Create profile directory structure
	profileDir := "/home/user/.weftlo/profiles/acme/backend"
	if err := fs.MkdirAll(profileDir, 0755); err != nil {
		t.Fatalf("failed to create profile directory: %v", err)
	}

	// Create a template file
	if err := afero.WriteFile(fs, profileDir+"/content/README.md", []byte("# README"), 0644); err != nil {
		t.Fatalf("failed to write README.md: %v", err)
	}

	loader := NewProfileLoaderWithFs(fs, func() (string, error) {
		return "/home/user", nil
	})

	profile, err := loader.Load("acme/backend")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// Verify template files were enumerated
	if len(profile.TemplateFiles) != 1 {
		t.Fatalf("expected 1 template file, got %d", len(profile.TemplateFiles))
	}

	// Verify TargetPath is empty (computed in Rendering Pipeline phase)
	if profile.TemplateFiles[0].TargetPath != "" {
		t.Errorf("expected TargetPath to be empty string, got '%s'", profile.TemplateFiles[0].TargetPath)
	}

	// Verify Checksum is empty (computed in Manifest Generation phase)
	if profile.TemplateFiles[0].Checksum != "" {
		t.Errorf("expected Checksum to be empty string, got '%s'", profile.TemplateFiles[0].Checksum)
	}
}

// =============================================================================
// Task Group 4: Integration Tests and Edge Cases
// =============================================================================

// TestIntegration_LoadCompleteProfileWithNestedDirectories tests loading a complete
// profile with nested directories, profile.yaml, and multiple template files.
// This is an end-to-end integration test for the profile loading workflow.
func TestIntegration_LoadCompleteProfileWithNestedDirectories(t *testing.T) {
	t.Parallel()
	fs := afero.NewMemMapFs()

	// Create profile directory structure with nested directories
	profileDir := "/home/user/.weftlo/profiles/acme/backend"
	if err := fs.MkdirAll(profileDir+"/content/docs/api/v1", 0755); err != nil {
		t.Fatalf("failed to create nested directories: %v", err)
	}
	if err := fs.MkdirAll(profileDir+"/content/standards/code", 0755); err != nil {
		t.Fatalf("failed to create standards directories: %v", err)
	}

	// Create profile.yaml with inheritance
	profileYAML := `name: acme/backend
inherits_from: acme/base
`
	if err := afero.WriteFile(fs, profileDir+"/profile.yaml", []byte(profileYAML), 0644); err != nil {
		t.Fatalf("failed to write profile.yaml: %v", err)
	}

	// Create template files in various directories
	files := map[string]string{
		"/content/README.md":            "# README",
		"/content/docs/index.md":        "# Docs Index",
		"/content/docs/api/overview.md": "# API Overview",
		"/content/docs/api/v1/spec.md":  "# API v1 Spec",
		"/content/standards/code/go.md": "# Go Standards",
		"/content/standards/code/js.md": "# JS Standards",
	}
	for path, content := range files {
		if err := afero.WriteFile(fs, profileDir+path, []byte(content), 0644); err != nil {
			t.Fatalf("failed to write %s: %v", path, err)
		}
	}

	loader := NewProfileLoaderWithFs(fs, func() (string, error) {
		return "/home/user", nil
	})

	profile, err := loader.Load("acme/backend")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// Verify profile metadata
	if profile.Name != "acme/backend" {
		t.Errorf("expected Name 'acme/backend', got '%s'", profile.Name)
	}
	if profile.InheritsFrom != "acme/base" {
		t.Errorf("expected InheritsFrom 'acme/base', got '%s'", profile.InheritsFrom)
	}

	// Verify all 6 template files were enumerated
	if len(profile.TemplateFiles) != 6 {
		t.Fatalf("expected 6 template files, got %d", len(profile.TemplateFiles))
	}

	// Verify all expected SourcePaths are present
	expectedPaths := map[string]bool{
		"README.md":            true,
		"docs/index.md":        true,
		"docs/api/overview.md": true,
		"docs/api/v1/spec.md":  true,
		"standards/code/go.md": true,
		"standards/code/js.md": true,
	}
	for _, tf := range profile.TemplateFiles {
		if !expectedPaths[tf.SourcePath] {
			t.Errorf("unexpected SourcePath: %s", tf.SourcePath)
		}
		delete(expectedPaths, tf.SourcePath)
	}
	for path := range expectedPaths {
		t.Errorf("missing expected SourcePath: %s", path)
	}
}

// TestIntegration_LoadProfileWithMixedPartialAndNonPartialFiles tests loading a
// profile that contains both partial and non-partial files in various locations.
func TestIntegration_LoadProfileWithMixedPartialAndNonPartialFiles(t *testing.T) {
	t.Parallel()
	fs := afero.NewMemMapFs()

	// Create profile directory structure
	profileDir := "/home/user/.weftlo/profiles/acme/backend"
	if err := fs.MkdirAll(profileDir+"/content/_partials/helpers", 0755); err != nil {
		t.Fatalf("failed to create _partials/helpers: %v", err)
	}
	if err := fs.MkdirAll(profileDir+"/content/docs", 0755); err != nil {
		t.Fatalf("failed to create docs: %v", err)
	}

	// Create mix of partial and non-partial files
	files := map[string]struct {
		content   string
		isPartial bool
	}{
		"/content/_header.md":                 {"# Header", true},
		"/content/_partials/helpers/utils.md": {"# Utils", true},
		"/content/docs/index.md":              {"# Docs", false},
		"/content/docs/_footer.md":            {"# Footer", true},
		"/content/README.md":                  {"# README", false},
	}

	for path, info := range files {
		if err := afero.WriteFile(fs, profileDir+path, []byte(info.content), 0644); err != nil {
			t.Fatalf("failed to write %s: %v", path, err)
		}
	}

	loader := NewProfileLoaderWithFs(fs, func() (string, error) {
		return "/home/user", nil
	})

	profile, err := loader.Load("acme/backend")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// Verify correct number of files
	if len(profile.TemplateFiles) != 5 {
		t.Fatalf("expected 5 template files, got %d", len(profile.TemplateFiles))
	}

	// Verify IsPartial flag for each file
	expectedPartials := map[string]bool{
		"_header.md":                 true,
		"_partials/helpers/utils.md": true,
		"docs/index.md":              false,
		"docs/_footer.md":            true,
		"README.md":                  false,
	}

	for _, tf := range profile.TemplateFiles {
		expectedIsPartial, exists := expectedPartials[tf.SourcePath]
		if !exists {
			t.Errorf("unexpected SourcePath: %s", tf.SourcePath)
			continue
		}
		if tf.IsPartial != expectedIsPartial {
			t.Errorf("expected IsPartial=%v for '%s', got %v", expectedIsPartial, tf.SourcePath, tf.IsPartial)
		}
	}
}

// TestEdgeCase_ProfileDirectoryWithOnlyPartialFiles tests loading a profile
// directory that contains only partial files (all start with underscore).
func TestEdgeCase_ProfileDirectoryWithOnlyPartialFiles(t *testing.T) {
	t.Parallel()
	fs := afero.NewMemMapFs()

	// Create profile directory with only partial files
	profileDir := "/home/user/.weftlo/profiles/acme/partials-only"
	if err := fs.MkdirAll(profileDir+"/content/_components", 0755); err != nil {
		t.Fatalf("failed to create directory: %v", err)
	}

	// Create only partial files
	partialFiles := []string{"/content/_header.md", "/content/_footer.md", "/content/_components/button.md"}
	for _, path := range partialFiles {
		if err := afero.WriteFile(fs, profileDir+path, []byte("partial content"), 0644); err != nil {
			t.Fatalf("failed to write %s: %v", path, err)
		}
	}

	loader := NewProfileLoaderWithFs(fs, func() (string, error) {
		return "/home/user", nil
	})

	profile, err := loader.Load("acme/partials-only")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// Verify all files were enumerated
	if len(profile.TemplateFiles) != 3 {
		t.Fatalf("expected 3 template files, got %d", len(profile.TemplateFiles))
	}

	// Verify all files are marked as partial
	for _, tf := range profile.TemplateFiles {
		if !tf.IsPartial {
			t.Errorf("expected '%s' to be partial, but IsPartial=false", tf.SourcePath)
		}
	}
}

// TestEdgeCase_ProfileDirectoryWithDeeplyNestedStructure tests loading a profile
// with a deeply nested directory structure (more than 5 levels deep).
func TestEdgeCase_ProfileDirectoryWithDeeplyNestedStructure(t *testing.T) {
	t.Parallel()
	fs := afero.NewMemMapFs()

	// Create profile directory with deeply nested structure (6 levels)
	profileDir := "/home/user/.weftlo/profiles/acme/deep"
	deepPath := "/level1/level2/level3/level4/level5/level6"
	if err := fs.MkdirAll(profileDir+"/content"+deepPath, 0755); err != nil {
		t.Fatalf("failed to create deep directory: %v", err)
	}

	// Create files at various depths
	files := []string{
		"/content/level1/file1.md",
		"/content/level1/level2/file2.md",
		"/content/level1/level2/level3/file3.md",
		"/content/level1/level2/level3/level4/file4.md",
		"/content/level1/level2/level3/level4/level5/file5.md",
		"/content/level1/level2/level3/level4/level5/level6/file6.md",
	}
	for _, path := range files {
		if err := afero.WriteFile(fs, profileDir+path, []byte("content"), 0644); err != nil {
			t.Fatalf("failed to write %s: %v", path, err)
		}
	}

	loader := NewProfileLoaderWithFs(fs, func() (string, error) {
		return "/home/user", nil
	})

	profile, err := loader.Load("acme/deep")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// Verify all 6 files were enumerated
	if len(profile.TemplateFiles) != 6 {
		t.Fatalf("expected 6 template files, got %d", len(profile.TemplateFiles))
	}

	// Verify the deepest file path uses forward slashes
	deepestPath := "level1/level2/level3/level4/level5/level6/file6.md"
	found := false
	for _, tf := range profile.TemplateFiles {
		if tf.SourcePath == deepestPath {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected to find deeply nested file at '%s'", deepestPath)
	}
}

// TestBoundary_ProfileNameWithValidSpecialCharacters tests loading profiles
// with valid special characters (hyphens and underscores) in the vendor/name.
func TestBoundary_ProfileNameWithValidSpecialCharacters(t *testing.T) {
	t.Parallel()
	fs := afero.NewMemMapFs()

	testCases := []struct {
		name        string
		profileName string
	}{
		{"hyphen in vendor", "my-company/backend"},
		{"hyphen in name", "acme/my-backend"},
		{"underscore in vendor", "my_company/backend"},
		{"underscore in name", "acme/my_backend"},
		{"mixed special chars", "my-company/my_backend"},
		{"numbers", "acme123/backend456"},
	}

	loader := NewProfileLoaderWithFs(fs, func() (string, error) {
		return "/home/user", nil
	})

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create the profile directory
			parts := splitProfileName(tc.profileName)
			profileDir := "/home/user/.weftlo/profiles/" + parts[0] + "/" + parts[1]
			if err := fs.MkdirAll(profileDir, 0755); err != nil {
				t.Fatalf("failed to create directory: %v", err)
			}
			defer fs.RemoveAll("/home/user/.weftlo/profiles/" + parts[0])

			// Create a template file
			if err := afero.WriteFile(fs, profileDir+"/content/README.md", []byte("# README"), 0644); err != nil {
				t.Fatalf("failed to write README.md: %v", err)
			}

			profile, err := loader.Load(tc.profileName)
			if err != nil {
				t.Fatalf("expected no error for profile '%s', got: %v", tc.profileName, err)
			}

			if profile.Name != tc.profileName {
				t.Errorf("expected Name '%s', got '%s'", tc.profileName, profile.Name)
			}
		})
	}
}

// splitProfileName splits a profile name into vendor and name parts.
func splitProfileName(profileName string) []string {
	for i, c := range profileName {
		if c == '/' {
			return []string{profileName[:i], profileName[i+1:]}
		}
	}
	return []string{profileName}
}

// TestBoundary_EmptyProfileDirectory tests loading a profile directory that
// exists but contains no template files (only profile.yaml or completely empty).
func TestBoundary_EmptyProfileDirectory(t *testing.T) {
	t.Parallel()
	fs := afero.NewMemMapFs()

	t.Run("empty directory with profile.yaml", func(t *testing.T) {
		profileDir := "/home/user/.weftlo/profiles/acme/empty-with-yaml"
		if err := fs.MkdirAll(profileDir, 0755); err != nil {
			t.Fatalf("failed to create directory: %v", err)
		}

		// Create only profile.yaml, no template files
		profileYAML := `name: acme/empty-with-yaml
inherits_from: acme/base
`
		if err := afero.WriteFile(fs, profileDir+"/profile.yaml", []byte(profileYAML), 0644); err != nil {
			t.Fatalf("failed to write profile.yaml: %v", err)
		}

		loader := NewProfileLoaderWithFs(fs, func() (string, error) {
			return "/home/user", nil
		})

		profile, err := loader.Load("acme/empty-with-yaml")
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}

		// Profile should load successfully with no template files
		if len(profile.TemplateFiles) != 0 {
			t.Errorf("expected 0 template files, got %d", len(profile.TemplateFiles))
		}

		// Metadata should still be populated
		if profile.Name != "acme/empty-with-yaml" {
			t.Errorf("expected Name 'acme/empty-with-yaml', got '%s'", profile.Name)
		}
		if profile.InheritsFrom != "acme/base" {
			t.Errorf("expected InheritsFrom 'acme/base', got '%s'", profile.InheritsFrom)
		}
	})

	t.Run("completely empty directory", func(t *testing.T) {
		profileDir := "/home/user/.weftlo/profiles/acme/completely-empty"
		if err := fs.MkdirAll(profileDir, 0755); err != nil {
			t.Fatalf("failed to create directory: %v", err)
		}

		loader := NewProfileLoaderWithFs(fs, func() (string, error) {
			return "/home/user", nil
		})

		profile, err := loader.Load("acme/completely-empty")
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}

		// Profile should load successfully with no template files
		if len(profile.TemplateFiles) != 0 {
			t.Errorf("expected 0 template files, got %d", len(profile.TemplateFiles))
		}

		// Name should be inferred from directory path
		if profile.Name != "acme/completely-empty" {
			t.Errorf("expected Name 'acme/completely-empty', got '%s'", profile.Name)
		}
	})
}

// TestEdgeCase_ProfileWithInheritsFromPreserved tests that InheritsFrom value
// is preserved in the Profile struct without resolution or validation.
func TestEdgeCase_ProfileWithInheritsFromPreserved(t *testing.T) {
	t.Parallel()
	fs := afero.NewMemMapFs()

	// Create profile directory
	profileDir := "/home/user/.weftlo/profiles/acme/child"
	if err := fs.MkdirAll(profileDir, 0755); err != nil {
		t.Fatalf("failed to create directory: %v", err)
	}

	// Create profile.yaml with various InheritsFrom values to test preservation
	testCases := []struct {
		name         string
		inheritsFrom string
	}{
		{"standard inheritance", "acme/parent"},
		{"deeply nested parent", "my-org/nested/deep-parent"},
		{"same vendor", "acme/sibling"},
		{"different vendor", "other-org/shared"},
	}

	loader := NewProfileLoaderWithFs(fs, func() (string, error) {
		return "/home/user", nil
	})

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Write profile.yaml with specified InheritsFrom
			profileYAML := "name: acme/child\ninherits_from: " + tc.inheritsFrom + "\n"
			if err := afero.WriteFile(fs, profileDir+"/profile.yaml", []byte(profileYAML), 0644); err != nil {
				t.Fatalf("failed to write profile.yaml: %v", err)
			}

			profile, err := loader.Load("acme/child")
			if err != nil {
				t.Fatalf("expected no error, got: %v", err)
			}

			// Verify InheritsFrom is preserved exactly as written
			if profile.InheritsFrom != tc.inheritsFrom {
				t.Errorf("expected InheritsFrom '%s', got '%s'", tc.inheritsFrom, profile.InheritsFrom)
			}
		})
	}
}

// =============================================================================
// Task Group 2: ProfileLoader Interface Compliance Tests (Inheritance Chain)
// =============================================================================

// TestDefaultProfileLoaderImplementsProfileLoaderInterface tests that
// DefaultProfileLoader implements the updated ProfileLoader interface.
// This is a compile-time interface compliance check.
func TestDefaultProfileLoaderImplementsProfileLoaderInterface(t *testing.T) {
	fs := afero.NewMemMapFs()

	// Create a DefaultProfileLoader
	loader := NewProfileLoaderWithFs(fs, func() (string, error) {
		return "/home/user", nil
	})

	// Verify that DefaultProfileLoader can be assigned to ProfileLoader interface.
	// This is a compile-time check - if DefaultProfileLoader doesn't implement
	// ProfileLoader, this assignment will cause a compilation error.
	var _ ProfileLoader = loader

	// The test passes if the code compiles successfully.
	// No runtime assertion needed - the type assertion above is sufficient.
}

// TestLoadWithChainMethodSignatureIsCorrect tests that the LoadWithChain method
// has the correct signature: LoadWithChain(profileName string) ([]*domainprofile.Profile, error)
func TestLoadWithChainMethodSignatureIsCorrect(t *testing.T) {
	t.Parallel()
	fs := afero.NewMemMapFs()

	// Create a DefaultProfileLoader
	loader := NewProfileLoaderWithFs(fs, func() (string, error) {
		return "/home/user", nil
	})

	// Call LoadWithChain and verify the return types match the expected signature.
	// The method should accept a string and return ([]*domainprofile.Profile, error).
	profileName := "acme/test"
	chain, err := loader.LoadWithChain(profileName)

	// Verify return types by using them (compile-time type checking)
	// chain should be []*domainprofile.Profile
	var _ []*domainprofile.Profile = chain
	// err should be error
	var _ error = err

	// Note: The actual behavior of LoadWithChain will be tested in Task Group 3 tests
}

// =============================================================================
// Task Group 3: LoadWithChain Implementation Tests (Inheritance Chain)
// =============================================================================

// TestLoadWithChain_SingleProfileWithNoParent tests that loading a profile with
// no parent (empty InheritsFrom) returns a slice of length 1 containing only that profile.
func TestLoadWithChain_SingleProfileWithNoParent(t *testing.T) {
	t.Parallel()
	fs := afero.NewMemMapFs()

	// Create a profile with no parent
	profileDir := "/home/user/.weftlo/profiles/acme/base"
	if err := fs.MkdirAll(profileDir, 0755); err != nil {
		t.Fatalf("failed to create profile directory: %v", err)
	}

	// Create profile.yaml with no inherits_from
	profileYAML := `name: acme/base
`
	if err := afero.WriteFile(fs, profileDir+"/profile.yaml", []byte(profileYAML), 0644); err != nil {
		t.Fatalf("failed to write profile.yaml: %v", err)
	}

	loader := NewProfileLoaderWithFs(fs, func() (string, error) {
		return "/home/user", nil
	})

	chain, err := loader.LoadWithChain("acme/base")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// Verify the chain has exactly 1 profile
	if len(chain) != 1 {
		t.Fatalf("expected chain of length 1, got %d", len(chain))
	}

	// Verify it's the correct profile
	if chain[0].Name != "acme/base" {
		t.Errorf("expected profile name 'acme/base', got '%s'", chain[0].Name)
	}
}

// TestLoadWithChain_ProfileWithOneParent tests that loading a profile with one parent
// returns a slice of length 2 in correct order: [parent, child].
func TestLoadWithChain_ProfileWithOneParent(t *testing.T) {
	t.Parallel()
	fs := afero.NewMemMapFs()

	// Create parent profile (no parent)
	parentDir := "/home/user/.weftlo/profiles/acme/parent"
	if err := fs.MkdirAll(parentDir, 0755); err != nil {
		t.Fatalf("failed to create parent directory: %v", err)
	}
	parentYAML := `name: acme/parent
`
	if err := afero.WriteFile(fs, parentDir+"/profile.yaml", []byte(parentYAML), 0644); err != nil {
		t.Fatalf("failed to write parent profile.yaml: %v", err)
	}

	// Create child profile (inherits from parent)
	childDir := "/home/user/.weftlo/profiles/acme/child"
	if err := fs.MkdirAll(childDir, 0755); err != nil {
		t.Fatalf("failed to create child directory: %v", err)
	}
	childYAML := `name: acme/child
inherits_from: acme/parent
`
	if err := afero.WriteFile(fs, childDir+"/profile.yaml", []byte(childYAML), 0644); err != nil {
		t.Fatalf("failed to write child profile.yaml: %v", err)
	}

	loader := NewProfileLoaderWithFs(fs, func() (string, error) {
		return "/home/user", nil
	})

	chain, err := loader.LoadWithChain("acme/child")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// Verify the chain has exactly 2 profiles
	if len(chain) != 2 {
		t.Fatalf("expected chain of length 2, got %d", len(chain))
	}

	// Verify order: parent first, child last
	if chain[0].Name != "acme/parent" {
		t.Errorf("expected first profile to be 'acme/parent', got '%s'", chain[0].Name)
	}
	if chain[1].Name != "acme/child" {
		t.Errorf("expected second profile to be 'acme/child', got '%s'", chain[1].Name)
	}
}

// TestLoadWithChain_ThreeLevelChain tests that loading a 3-level chain returns
// the slice ordered root-to-leaf: [grandparent, parent, child].
func TestLoadWithChain_ThreeLevelChain(t *testing.T) {
	t.Parallel()
	fs := afero.NewMemMapFs()

	// Create grandparent profile (root)
	grandparentDir := "/home/user/.weftlo/profiles/acme/grandparent"
	if err := fs.MkdirAll(grandparentDir, 0755); err != nil {
		t.Fatalf("failed to create grandparent directory: %v", err)
	}
	grandparentYAML := `name: acme/grandparent
`
	if err := afero.WriteFile(fs, grandparentDir+"/profile.yaml", []byte(grandparentYAML), 0644); err != nil {
		t.Fatalf("failed to write grandparent profile.yaml: %v", err)
	}

	// Create parent profile (inherits from grandparent)
	parentDir := "/home/user/.weftlo/profiles/acme/parent"
	if err := fs.MkdirAll(parentDir, 0755); err != nil {
		t.Fatalf("failed to create parent directory: %v", err)
	}
	parentYAML := `name: acme/parent
inherits_from: acme/grandparent
`
	if err := afero.WriteFile(fs, parentDir+"/profile.yaml", []byte(parentYAML), 0644); err != nil {
		t.Fatalf("failed to write parent profile.yaml: %v", err)
	}

	// Create child profile (inherits from parent)
	childDir := "/home/user/.weftlo/profiles/acme/child"
	if err := fs.MkdirAll(childDir, 0755); err != nil {
		t.Fatalf("failed to create child directory: %v", err)
	}
	childYAML := `name: acme/child
inherits_from: acme/parent
`
	if err := afero.WriteFile(fs, childDir+"/profile.yaml", []byte(childYAML), 0644); err != nil {
		t.Fatalf("failed to write child profile.yaml: %v", err)
	}

	loader := NewProfileLoaderWithFs(fs, func() (string, error) {
		return "/home/user", nil
	})

	chain, err := loader.LoadWithChain("acme/child")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// Verify the chain has exactly 3 profiles
	if len(chain) != 3 {
		t.Fatalf("expected chain of length 3, got %d", len(chain))
	}

	// Verify order: grandparent, parent, child
	if chain[0].Name != "acme/grandparent" {
		t.Errorf("expected first profile to be 'acme/grandparent', got '%s'", chain[0].Name)
	}
	if chain[1].Name != "acme/parent" {
		t.Errorf("expected second profile to be 'acme/parent', got '%s'", chain[1].Name)
	}
	if chain[2].Name != "acme/child" {
		t.Errorf("expected third profile to be 'acme/child', got '%s'", chain[2].Name)
	}
}

// TestLoadWithChain_CircularDependency tests that a circular dependency (A -> B -> A)
// returns a CircularDependencyError.
func TestLoadWithChain_CircularDependency(t *testing.T) {
	t.Parallel()
	fs := afero.NewMemMapFs()

	// Create profile A (inherits from B)
	profileADir := "/home/user/.weftlo/profiles/acme/a"
	if err := fs.MkdirAll(profileADir, 0755); err != nil {
		t.Fatalf("failed to create profile A directory: %v", err)
	}
	profileAYAML := `name: acme/a
inherits_from: acme/b
`
	if err := afero.WriteFile(fs, profileADir+"/profile.yaml", []byte(profileAYAML), 0644); err != nil {
		t.Fatalf("failed to write profile A profile.yaml: %v", err)
	}

	// Create profile B (inherits from A - creates cycle)
	profileBDir := "/home/user/.weftlo/profiles/acme/b"
	if err := fs.MkdirAll(profileBDir, 0755); err != nil {
		t.Fatalf("failed to create profile B directory: %v", err)
	}
	profileBYAML := `name: acme/b
inherits_from: acme/a
`
	if err := afero.WriteFile(fs, profileBDir+"/profile.yaml", []byte(profileBYAML), 0644); err != nil {
		t.Fatalf("failed to write profile B profile.yaml: %v", err)
	}

	loader := NewProfileLoaderWithFs(fs, func() (string, error) {
		return "/home/user", nil
	})

	_, err := loader.LoadWithChain("acme/a")
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var circularErr *CircularDependencyError
	if !errors.As(err, &circularErr) {
		t.Fatalf("expected CircularDependencyError, got %T: %v", err, err)
	}

	// Verify the error contains the profile that created the cycle
	if circularErr.ProfileName != "acme/a" {
		t.Errorf("expected ProfileName 'acme/a', got '%s'", circularErr.ProfileName)
	}

	// Verify the chain is populated
	if len(circularErr.Chain) == 0 {
		t.Error("expected Chain to be populated")
	}
}

// TestLoadWithChain_SelfReferencingProfile tests that a self-referencing profile (A -> A)
// returns a CircularDependencyError.
func TestLoadWithChain_SelfReferencingProfile(t *testing.T) {
	t.Parallel()
	fs := afero.NewMemMapFs()

	// Create profile that inherits from itself
	profileDir := "/home/user/.weftlo/profiles/acme/self"
	if err := fs.MkdirAll(profileDir, 0755); err != nil {
		t.Fatalf("failed to create profile directory: %v", err)
	}
	profileYAML := `name: acme/self
inherits_from: acme/self
`
	if err := afero.WriteFile(fs, profileDir+"/profile.yaml", []byte(profileYAML), 0644); err != nil {
		t.Fatalf("failed to write profile.yaml: %v", err)
	}

	loader := NewProfileLoaderWithFs(fs, func() (string, error) {
		return "/home/user", nil
	})

	_, err := loader.LoadWithChain("acme/self")
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var circularErr *CircularDependencyError
	if !errors.As(err, &circularErr) {
		t.Fatalf("expected CircularDependencyError, got %T: %v", err, err)
	}

	// Verify the error identifies the self-referencing profile
	if circularErr.ProfileName != "acme/self" {
		t.Errorf("expected ProfileName 'acme/self', got '%s'", circularErr.ProfileName)
	}
}

// TestLoadWithChain_DepthLimitExceeded tests that exceeding the depth limit
// returns an InheritanceDepthExceededError with the correct chain.
func TestLoadWithChain_DepthLimitExceeded(t *testing.T) {
	t.Parallel()
	fs := afero.NewMemMapFs()

	// Create a chain that exceeds MaxInheritanceDepth (15)
	// We need 16 profiles in the chain to exceed the limit
	for i := 0; i <= MaxInheritanceDepth; i++ {
		profileDir := "/home/user/.weftlo/profiles/acme/level" + strconv.Itoa(i)
		if err := fs.MkdirAll(profileDir, 0755); err != nil {
			t.Fatalf("failed to create profile directory for level %d: %v", i, err)
		}

		var profileYAML string
		if i == 0 {
			// Root profile (no parent)
			profileYAML = "name: acme/level0\n"
		} else {
			// Profile inheriting from previous level
			profileYAML = "name: acme/level" + strconv.Itoa(i) + "\ninherits_from: acme/level" + strconv.Itoa(i-1) + "\n"
		}

		if err := afero.WriteFile(fs, profileDir+"/profile.yaml", []byte(profileYAML), 0644); err != nil {
			t.Fatalf("failed to write profile.yaml for level %d: %v", i, err)
		}
	}

	loader := NewProfileLoaderWithFs(fs, func() (string, error) {
		return "/home/user", nil
	})

	// Load the deepest level, which should exceed the limit
	_, err := loader.LoadWithChain("acme/level" + strconv.Itoa(MaxInheritanceDepth))
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var depthErr *InheritanceDepthExceededError
	if !errors.As(err, &depthErr) {
		t.Fatalf("expected InheritanceDepthExceededError, got %T: %v", err, err)
	}

	// Verify the max depth is set correctly
	if depthErr.MaxDepth != MaxInheritanceDepth {
		t.Errorf("expected MaxDepth %d, got %d", MaxInheritanceDepth, depthErr.MaxDepth)
	}

	// Verify the chain is populated
	if len(depthErr.Chain) == 0 {
		t.Error("expected Chain to be populated")
	}
}

// TestLoadWithChain_MissingParentProfile tests that a missing parent profile
// propagates ProfileNotFoundError from the Load method.
func TestLoadWithChain_MissingParentProfile(t *testing.T) {
	t.Parallel()
	fs := afero.NewMemMapFs()

	// Create child profile that references a non-existent parent
	childDir := "/home/user/.weftlo/profiles/acme/child"
	if err := fs.MkdirAll(childDir, 0755); err != nil {
		t.Fatalf("failed to create child directory: %v", err)
	}
	childYAML := `name: acme/child
inherits_from: acme/nonexistent
`
	if err := afero.WriteFile(fs, childDir+"/profile.yaml", []byte(childYAML), 0644); err != nil {
		t.Fatalf("failed to write child profile.yaml: %v", err)
	}

	loader := NewProfileLoaderWithFs(fs, func() (string, error) {
		return "/home/user", nil
	})

	_, err := loader.LoadWithChain("acme/child")
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var notFoundErr *ProfileNotFoundError
	if !errors.As(err, &notFoundErr) {
		t.Fatalf("expected ProfileNotFoundError, got %T: %v", err, err)
	}

	// Verify the error identifies the missing parent
	if notFoundErr.ProfileName != "acme/nonexistent" {
		t.Errorf("expected ProfileName 'acme/nonexistent', got '%s'", notFoundErr.ProfileName)
	}
}

// TestLoadWithChain_InvalidParentProfileYAML tests that invalid YAML in a parent profile
// propagates ProfileConfigParseError from the Load method.
func TestLoadWithChain_InvalidParentProfileYAML(t *testing.T) {
	t.Parallel()
	fs := afero.NewMemMapFs()

	// Create parent profile with invalid YAML
	parentDir := "/home/user/.weftlo/profiles/acme/parent"
	if err := fs.MkdirAll(parentDir, 0755); err != nil {
		t.Fatalf("failed to create parent directory: %v", err)
	}
	invalidYAML := `name: acme/parent
inherits_from: [invalid
  - this is not valid yaml
`
	if err := afero.WriteFile(fs, parentDir+"/profile.yaml", []byte(invalidYAML), 0644); err != nil {
		t.Fatalf("failed to write parent profile.yaml: %v", err)
	}

	// Create child profile that references the invalid parent
	childDir := "/home/user/.weftlo/profiles/acme/child"
	if err := fs.MkdirAll(childDir, 0755); err != nil {
		t.Fatalf("failed to create child directory: %v", err)
	}
	childYAML := `name: acme/child
inherits_from: acme/parent
`
	if err := afero.WriteFile(fs, childDir+"/profile.yaml", []byte(childYAML), 0644); err != nil {
		t.Fatalf("failed to write child profile.yaml: %v", err)
	}

	loader := NewProfileLoaderWithFs(fs, func() (string, error) {
		return "/home/user", nil
	})

	_, err := loader.LoadWithChain("acme/child")
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var parseErr *ProfileConfigParseError
	if !errors.As(err, &parseErr) {
		t.Fatalf("expected ProfileConfigParseError, got %T: %v", err, err)
	}

	// Verify the error refers to the parent profile
	expectedPath := "/home/user/.weftlo/profiles/acme/parent/profile.yaml"
	if parseErr.Path != expectedPath {
		t.Errorf("expected Path '%s', got '%s'", expectedPath, parseErr.Path)
	}
}

// =============================================================================
// Task Group 4: Additional Strategic Tests for Inheritance Chain Feature
// =============================================================================

// TestLoadWithChain_FiveLevelChainWithinDepthLimit tests that a 5-level chain
// (within the 15 level limit) loads successfully and maintains correct ordering.
func TestLoadWithChain_FiveLevelChainWithinDepthLimit(t *testing.T) {
	t.Parallel()
	fs := afero.NewMemMapFs()

	// Create a 5-level chain: level0 -> level1 -> level2 -> level3 -> level4
	expectedNames := []string{
		"acme/level0",
		"acme/level1",
		"acme/level2",
		"acme/level3",
		"acme/level4",
	}

	for i := 0; i < 5; i++ {
		profileDir := "/home/user/.weftlo/profiles/acme/level" + strconv.Itoa(i)
		if err := fs.MkdirAll(profileDir, 0755); err != nil {
			t.Fatalf("failed to create profile directory for level %d: %v", i, err)
		}

		var profileYAML string
		if i == 0 {
			// Root profile (no parent)
			profileYAML = "name: acme/level0\n"
		} else {
			// Profile inheriting from previous level
			profileYAML = "name: acme/level" + strconv.Itoa(i) + "\ninherits_from: acme/level" + strconv.Itoa(i-1) + "\n"
		}

		if err := afero.WriteFile(fs, profileDir+"/profile.yaml", []byte(profileYAML), 0644); err != nil {
			t.Fatalf("failed to write profile.yaml for level %d: %v", i, err)
		}
	}

	loader := NewProfileLoaderWithFs(fs, func() (string, error) {
		return "/home/user", nil
	})

	chain, err := loader.LoadWithChain("acme/level4")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// Verify the chain has exactly 5 profiles
	if len(chain) != 5 {
		t.Fatalf("expected chain of length 5, got %d", len(chain))
	}

	// Verify ordering: root to leaf
	for i, expected := range expectedNames {
		if chain[i].Name != expected {
			t.Errorf("expected chain[%d] to be '%s', got '%s'", i, expected, chain[i].Name)
		}
	}
}

// TestLoadWithChain_CrossVendorInheritance tests that a chain can span
// multiple vendors (e.g., "acme/base" -> "other/child").
func TestLoadWithChain_CrossVendorInheritance(t *testing.T) {
	t.Parallel()
	fs := afero.NewMemMapFs()

	// Create root profile in "shared" vendor
	sharedDir := "/home/user/.weftlo/profiles/shared/base"
	if err := fs.MkdirAll(sharedDir, 0755); err != nil {
		t.Fatalf("failed to create shared base directory: %v", err)
	}
	sharedYAML := `name: shared/base
`
	if err := afero.WriteFile(fs, sharedDir+"/profile.yaml", []byte(sharedYAML), 0644); err != nil {
		t.Fatalf("failed to write shared base profile.yaml: %v", err)
	}

	// Create intermediate profile in "acme" vendor inheriting from "shared"
	acmeDir := "/home/user/.weftlo/profiles/acme/common"
	if err := fs.MkdirAll(acmeDir, 0755); err != nil {
		t.Fatalf("failed to create acme common directory: %v", err)
	}
	acmeYAML := `name: acme/common
inherits_from: shared/base
`
	if err := afero.WriteFile(fs, acmeDir+"/profile.yaml", []byte(acmeYAML), 0644); err != nil {
		t.Fatalf("failed to write acme common profile.yaml: %v", err)
	}

	// Create leaf profile in "myorg" vendor inheriting from "acme"
	myorgDir := "/home/user/.weftlo/profiles/myorg/app"
	if err := fs.MkdirAll(myorgDir, 0755); err != nil {
		t.Fatalf("failed to create myorg app directory: %v", err)
	}
	myorgYAML := `name: myorg/app
inherits_from: acme/common
`
	if err := afero.WriteFile(fs, myorgDir+"/profile.yaml", []byte(myorgYAML), 0644); err != nil {
		t.Fatalf("failed to write myorg app profile.yaml: %v", err)
	}

	loader := NewProfileLoaderWithFs(fs, func() (string, error) {
		return "/home/user", nil
	})

	chain, err := loader.LoadWithChain("myorg/app")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// Verify the chain has 3 profiles from 3 different vendors
	if len(chain) != 3 {
		t.Fatalf("expected chain of length 3, got %d", len(chain))
	}

	// Verify ordering: shared/base -> acme/common -> myorg/app
	if chain[0].Name != "shared/base" {
		t.Errorf("expected first profile to be 'shared/base', got '%s'", chain[0].Name)
	}
	if chain[1].Name != "acme/common" {
		t.Errorf("expected second profile to be 'acme/common', got '%s'", chain[1].Name)
	}
	if chain[2].Name != "myorg/app" {
		t.Errorf("expected third profile to be 'myorg/app', got '%s'", chain[2].Name)
	}
}

// TestLoadWithChain_IntermediateProfileWithoutProfileYAML tests that the chain
// correctly handles an intermediate profile that has no profile.yaml file
// (name inferred from directory path).
func TestLoadWithChain_IntermediateProfileWithoutProfileYAML(t *testing.T) {
	t.Parallel()
	fs := afero.NewMemMapFs()

	// Create root profile with profile.yaml
	rootDir := "/home/user/.weftlo/profiles/acme/root"
	if err := fs.MkdirAll(rootDir, 0755); err != nil {
		t.Fatalf("failed to create root directory: %v", err)
	}
	rootYAML := `name: acme/root
`
	if err := afero.WriteFile(fs, rootDir+"/profile.yaml", []byte(rootYAML), 0644); err != nil {
		t.Fatalf("failed to write root profile.yaml: %v", err)
	}

	// Create intermediate profile WITHOUT profile.yaml (only inherits_from via a minimal yaml)
	// The profile directory exists but has profile.yaml with only inherits_from (name will be inferred)
	middleDir := "/home/user/.weftlo/profiles/acme/middle"
	if err := fs.MkdirAll(middleDir, 0755); err != nil {
		t.Fatalf("failed to create middle directory: %v", err)
	}
	// Only inherits_from, no name field - name should be inferred
	middleYAML := `inherits_from: acme/root
`
	if err := afero.WriteFile(fs, middleDir+"/profile.yaml", []byte(middleYAML), 0644); err != nil {
		t.Fatalf("failed to write middle profile.yaml: %v", err)
	}

	// Create leaf profile with profile.yaml
	leafDir := "/home/user/.weftlo/profiles/acme/leaf"
	if err := fs.MkdirAll(leafDir, 0755); err != nil {
		t.Fatalf("failed to create leaf directory: %v", err)
	}
	leafYAML := `name: acme/leaf
inherits_from: acme/middle
`
	if err := afero.WriteFile(fs, leafDir+"/profile.yaml", []byte(leafYAML), 0644); err != nil {
		t.Fatalf("failed to write leaf profile.yaml: %v", err)
	}

	loader := NewProfileLoaderWithFs(fs, func() (string, error) {
		return "/home/user", nil
	})

	chain, err := loader.LoadWithChain("acme/leaf")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// Verify the chain has 3 profiles
	if len(chain) != 3 {
		t.Fatalf("expected chain of length 3, got %d", len(chain))
	}

	// Verify the intermediate profile has its name inferred
	if chain[1].Name != "acme/middle" {
		t.Errorf("expected middle profile name to be 'acme/middle' (inferred), got '%s'", chain[1].Name)
	}

	// Verify full ordering
	if chain[0].Name != "acme/root" {
		t.Errorf("expected first profile to be 'acme/root', got '%s'", chain[0].Name)
	}
	if chain[2].Name != "acme/leaf" {
		t.Errorf("expected third profile to be 'acme/leaf', got '%s'", chain[2].Name)
	}
}

// TestLoadWithChain_DepthExactlyAtLimitSucceeds tests that a chain with exactly
// MaxInheritanceDepth (15) levels succeeds without error.
func TestLoadWithChain_DepthExactlyAtLimitSucceeds(t *testing.T) {
	t.Parallel()
	fs := afero.NewMemMapFs()

	// Create a chain with exactly MaxInheritanceDepth levels (15 profiles, indices 0-14)
	// This should succeed because depth is checked BEFORE loading, so:
	// - depth 0: load level0 (check passes: 0 < 15)
	// - depth 1: load level1 (check passes: 1 < 15)
	// - ...
	// - depth 14: load level14 (check passes: 14 < 15)
	// Total: 15 profiles, all loaded successfully
	for i := 0; i < MaxInheritanceDepth; i++ {
		profileDir := "/home/user/.weftlo/profiles/acme/exact" + strconv.Itoa(i)
		if err := fs.MkdirAll(profileDir, 0755); err != nil {
			t.Fatalf("failed to create profile directory for level %d: %v", i, err)
		}

		var profileYAML string
		if i == 0 {
			// Root profile (no parent)
			profileYAML = "name: acme/exact0\n"
		} else {
			// Profile inheriting from previous level
			profileYAML = "name: acme/exact" + strconv.Itoa(i) + "\ninherits_from: acme/exact" + strconv.Itoa(i-1) + "\n"
		}

		if err := afero.WriteFile(fs, profileDir+"/profile.yaml", []byte(profileYAML), 0644); err != nil {
			t.Fatalf("failed to write profile.yaml for level %d: %v", i, err)
		}
	}

	loader := NewProfileLoaderWithFs(fs, func() (string, error) {
		return "/home/user", nil
	})

	// Load the deepest level at exactly the limit - this should succeed
	chain, err := loader.LoadWithChain("acme/exact" + strconv.Itoa(MaxInheritanceDepth-1))
	if err != nil {
		t.Fatalf("expected no error for chain at exactly depth limit, got: %v", err)
	}

	// Verify we got all 15 profiles
	if len(chain) != MaxInheritanceDepth {
		t.Errorf("expected chain of length %d, got %d", MaxInheritanceDepth, len(chain))
	}

	// Verify ordering is correct
	for i := 0; i < MaxInheritanceDepth; i++ {
		expectedName := "acme/exact" + strconv.Itoa(i)
		if chain[i].Name != expectedName {
			t.Errorf("expected chain[%d] to be '%s', got '%s'", i, expectedName, chain[i].Name)
		}
	}
}

// TestLoadWithChain_DepthExceededErrorContainsAllVisitedProfiles tests that the
// InheritanceDepthExceededError's Chain field contains all profile names visited
// before the depth limit was reached.
func TestLoadWithChain_DepthExceededErrorContainsAllVisitedProfiles(t *testing.T) {
	t.Parallel()
	fs := afero.NewMemMapFs()

	// Create a chain that exceeds MaxInheritanceDepth (needs 16 profiles)
	for i := 0; i <= MaxInheritanceDepth; i++ {
		profileDir := "/home/user/.weftlo/profiles/acme/deep" + strconv.Itoa(i)
		if err := fs.MkdirAll(profileDir, 0755); err != nil {
			t.Fatalf("failed to create profile directory for level %d: %v", i, err)
		}

		var profileYAML string
		if i == 0 {
			profileYAML = "name: acme/deep0\n"
		} else {
			profileYAML = "name: acme/deep" + strconv.Itoa(i) + "\ninherits_from: acme/deep" + strconv.Itoa(i-1) + "\n"
		}

		if err := afero.WriteFile(fs, profileDir+"/profile.yaml", []byte(profileYAML), 0644); err != nil {
			t.Fatalf("failed to write profile.yaml for level %d: %v", i, err)
		}
	}

	loader := NewProfileLoaderWithFs(fs, func() (string, error) {
		return "/home/user", nil
	})

	_, err := loader.LoadWithChain("acme/deep" + strconv.Itoa(MaxInheritanceDepth))
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var depthErr *InheritanceDepthExceededError
	if !errors.As(err, &depthErr) {
		t.Fatalf("expected InheritanceDepthExceededError, got %T: %v", err, err)
	}

	// Verify the chain contains the expected number of visited profiles
	// The chain should contain all profiles visited before the limit was hit
	if len(depthErr.Chain) != MaxInheritanceDepth {
		t.Errorf("expected Chain to contain %d profiles visited before limit, got %d: %v",
			MaxInheritanceDepth, len(depthErr.Chain), depthErr.Chain)
	}

	// Verify the chain starts with the original requested profile
	if len(depthErr.Chain) > 0 && depthErr.Chain[0] != "acme/deep"+strconv.Itoa(MaxInheritanceDepth) {
		t.Errorf("expected first profile in chain to be 'acme/deep%d', got '%s'",
			MaxInheritanceDepth, depthErr.Chain[0])
	}
}

// TestLoadWithChain_CircularDependencyErrorContainsFullChain tests that the
// CircularDependencyError's Chain field contains all profile names visited
// including the one that creates the cycle.
func TestLoadWithChain_CircularDependencyErrorContainsFullChain(t *testing.T) {
	t.Parallel()
	fs := afero.NewMemMapFs()

	// Create a longer circular chain: A -> B -> C -> A
	profileADir := "/home/user/.weftlo/profiles/acme/cycleA"
	if err := fs.MkdirAll(profileADir, 0755); err != nil {
		t.Fatalf("failed to create profile A directory: %v", err)
	}
	if err := afero.WriteFile(fs, profileADir+"/profile.yaml", []byte("name: acme/cycleA\ninherits_from: acme/cycleB\n"), 0644); err != nil {
		t.Fatalf("failed to write profile A: %v", err)
	}

	profileBDir := "/home/user/.weftlo/profiles/acme/cycleB"
	if err := fs.MkdirAll(profileBDir, 0755); err != nil {
		t.Fatalf("failed to create profile B directory: %v", err)
	}
	if err := afero.WriteFile(fs, profileBDir+"/profile.yaml", []byte("name: acme/cycleB\ninherits_from: acme/cycleC\n"), 0644); err != nil {
		t.Fatalf("failed to write profile B: %v", err)
	}

	profileCDir := "/home/user/.weftlo/profiles/acme/cycleC"
	if err := fs.MkdirAll(profileCDir, 0755); err != nil {
		t.Fatalf("failed to create profile C directory: %v", err)
	}
	if err := afero.WriteFile(fs, profileCDir+"/profile.yaml", []byte("name: acme/cycleC\ninherits_from: acme/cycleA\n"), 0644); err != nil {
		t.Fatalf("failed to write profile C: %v", err)
	}

	loader := NewProfileLoaderWithFs(fs, func() (string, error) {
		return "/home/user", nil
	})

	_, err := loader.LoadWithChain("acme/cycleA")
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var circularErr *CircularDependencyError
	if !errors.As(err, &circularErr) {
		t.Fatalf("expected CircularDependencyError, got %T: %v", err, err)
	}

	// Verify ProfileName is the one that created the cycle
	if circularErr.ProfileName != "acme/cycleA" {
		t.Errorf("expected ProfileName 'acme/cycleA', got '%s'", circularErr.ProfileName)
	}

	// Verify Chain contains all visited profiles plus the cycle-creating profile
	// Chain should be: [acme/cycleA, acme/cycleB, acme/cycleC, acme/cycleA]
	if len(circularErr.Chain) != 4 {
		t.Errorf("expected Chain to contain 4 profiles, got %d: %v", len(circularErr.Chain), circularErr.Chain)
	}

	// Verify the chain shows the full path to the cycle
	expectedChain := []string{"acme/cycleA", "acme/cycleB", "acme/cycleC", "acme/cycleA"}
	for i, expected := range expectedChain {
		if i < len(circularErr.Chain) && circularErr.Chain[i] != expected {
			t.Errorf("expected Chain[%d] to be '%s', got '%s'", i, expected, circularErr.Chain[i])
		}
	}
}

// =============================================================================
// Task Group 2 (Profile Loader Updates): Loader Functionality Tests
// =============================================================================

// TestLoader_AgentConfigIgnoreFileIsLoadedFromProfileRoot tests that
// .weftlo.ignore file is loaded from profile root when it exists.
func TestLoader_AgentConfigIgnoreFileIsLoadedFromProfileRoot(t *testing.T) {
	t.Parallel()
	fs := afero.NewMemMapFs()

	// Create profile directory structure
	profileDir := "/home/user/.weftlo/profiles/acme/backend"
	if err := fs.MkdirAll(profileDir, 0755); err != nil {
		t.Fatalf("failed to create profile directory: %v", err)
	}

	// Create profile.yaml
	profileYAML := `name: acme/backend
`
	if err := afero.WriteFile(fs, profileDir+"/profile.yaml", []byte(profileYAML), 0644); err != nil {
		t.Fatalf("failed to write profile.yaml: %v", err)
	}

	// Create .weftlo.ignore with patterns
	ignoreContent := `# Ignore draft files
*.draft
*.bak
`
	if err := afero.WriteFile(fs, profileDir+"/.weftlo.ignore", []byte(ignoreContent), 0644); err != nil {
		t.Fatalf("failed to write .weftlo.ignore: %v", err)
	}

	loader := NewProfileLoaderWithFs(fs, func() (string, error) {
		return "/home/user", nil
	})

	profile, err := loader.Load("acme/backend")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// Verify Matcher is populated and matches patterns from ignore file
	if profile.Matcher == nil {
		t.Fatal("expected Matcher to be populated, got nil")
	}

	// Test that patterns from ignore file work
	if !profile.Matcher.Match("readme.draft", false) {
		t.Error("expected '*.draft' pattern to match 'readme.draft'")
	}
	if !profile.Matcher.Match("old.bak", false) {
		t.Error("expected '*.bak' pattern to match 'old.bak'")
	}
	if profile.Matcher.Match("readme.md", false) {
		t.Error("expected non-matching file 'readme.md' to not be ignored")
	}
}

// TestLoader_EmptyMatcherCreatedWhenIgnoreFileDoesNotExist tests that
// empty/no-op Matcher is created when .weftlo.ignore does not exist (no error).
func TestLoader_EmptyMatcherCreatedWhenIgnoreFileDoesNotExist(t *testing.T) {
	t.Parallel()
	fs := afero.NewMemMapFs()

	// Create profile directory structure without .weftlo.ignore
	profileDir := "/home/user/.weftlo/profiles/acme/backend"
	if err := fs.MkdirAll(profileDir, 0755); err != nil {
		t.Fatalf("failed to create profile directory: %v", err)
	}

	// Create profile.yaml
	profileYAML := `name: acme/backend
`
	if err := afero.WriteFile(fs, profileDir+"/profile.yaml", []byte(profileYAML), 0644); err != nil {
		t.Fatalf("failed to write profile.yaml: %v", err)
	}

	loader := NewProfileLoaderWithFs(fs, func() (string, error) {
		return "/home/user", nil
	})

	profile, err := loader.Load("acme/backend")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// Verify Matcher is populated with empty/no-op matcher
	if profile.Matcher == nil {
		t.Fatal("expected Matcher to be populated with empty matcher, got nil")
	}

	// Empty matcher should not match anything
	if profile.Matcher.Match("anything.txt", false) {
		t.Error("expected empty matcher to not match any files")
	}
	if profile.Matcher.Match("readme.md", false) {
		t.Error("expected empty matcher to not match 'readme.md'")
	}
}

// TestLoader_InlineIgnorePatternsFromProfileYamlAreLoaded tests that
// inline ignore: patterns from profile.yaml are loaded.
func TestLoader_InlineIgnorePatternsFromProfileYamlAreLoaded(t *testing.T) {
	t.Parallel()
	fs := afero.NewMemMapFs()

	// Create profile directory structure
	profileDir := "/home/user/.weftlo/profiles/acme/backend"
	if err := fs.MkdirAll(profileDir, 0755); err != nil {
		t.Fatalf("failed to create profile directory: %v", err)
	}

	// Create profile.yaml with inline ignore patterns
	profileYAML := `name: acme/backend
ignore:
  - "*.tmp"
  - "*.log"
  - "cache/"
`
	if err := afero.WriteFile(fs, profileDir+"/profile.yaml", []byte(profileYAML), 0644); err != nil {
		t.Fatalf("failed to write profile.yaml: %v", err)
	}

	loader := NewProfileLoaderWithFs(fs, func() (string, error) {
		return "/home/user", nil
	})

	profile, err := loader.Load("acme/backend")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// Verify Matcher is populated with inline patterns
	if profile.Matcher == nil {
		t.Fatal("expected Matcher to be populated, got nil")
	}

	// Test that inline patterns work
	if !profile.Matcher.Match("temp.tmp", false) {
		t.Error("expected '*.tmp' pattern to match 'temp.tmp'")
	}
	if !profile.Matcher.Match("app.log", false) {
		t.Error("expected '*.log' pattern to match 'app.log'")
	}
	if !profile.Matcher.Match("cache", true) {
		t.Error("expected 'cache/' pattern to match 'cache' directory")
	}
	if profile.Matcher.Match("readme.md", false) {
		t.Error("expected non-matching file 'readme.md' to not be ignored")
	}
}

// TestLoader_InlinePatternsAreMergedAfterFileBasedPatterns tests that
// inline patterns are merged after file-based patterns.
func TestLoader_InlinePatternsAreMergedAfterFileBasedPatterns(t *testing.T) {
	t.Parallel()
	fs := afero.NewMemMapFs()

	// Create profile directory structure
	profileDir := "/home/user/.weftlo/profiles/acme/backend"
	if err := fs.MkdirAll(profileDir, 0755); err != nil {
		t.Fatalf("failed to create profile directory: %v", err)
	}

	// Create .weftlo.ignore with patterns that exclude *.draft files
	ignoreContent := `# Exclude all draft files
*.draft
`
	if err := afero.WriteFile(fs, profileDir+"/.weftlo.ignore", []byte(ignoreContent), 0644); err != nil {
		t.Fatalf("failed to write .weftlo.ignore: %v", err)
	}

	// Create profile.yaml with inline patterns that re-include important.draft
	profileYAML := `name: acme/backend
ignore:
  - "!important.draft"
`
	if err := afero.WriteFile(fs, profileDir+"/profile.yaml", []byte(profileYAML), 0644); err != nil {
		t.Fatalf("failed to write profile.yaml: %v", err)
	}

	loader := NewProfileLoaderWithFs(fs, func() (string, error) {
		return "/home/user", nil
	})

	profile, err := loader.Load("acme/backend")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// Verify Matcher is populated
	if profile.Matcher == nil {
		t.Fatal("expected Matcher to be populated, got nil")
	}

	// File-based pattern should exclude *.draft (except important.draft)
	if !profile.Matcher.Match("readme.draft", false) {
		t.Error("expected '*.draft' pattern to match 'readme.draft'")
	}

	// Inline negation pattern should re-include important.draft
	if profile.Matcher.Match("important.draft", false) {
		t.Error("expected '!important.draft' negation to NOT match 'important.draft'")
	}
}

// TestLoader_ContentConfigIsPopulatedFromProfileConfigContent tests that
// ContentConfig is populated from ProfileConfig.Content.
func TestLoader_ContentConfigIsPopulatedFromProfileConfigContent(t *testing.T) {
	t.Parallel()
	fs := afero.NewMemMapFs()

	// Create profile directory structure
	profileDir := "/home/user/.weftlo/profiles/acme/backend"
	if err := fs.MkdirAll(profileDir, 0755); err != nil {
		t.Fatalf("failed to create profile directory: %v", err)
	}

	// Create profile.yaml with content configuration
	profileYAML := `name: acme/backend
content:
  root: templates
  default_target: .claude
  targets:
    skills: .claude/skills
    standards: weftlo/standards
`
	if err := afero.WriteFile(fs, profileDir+"/profile.yaml", []byte(profileYAML), 0644); err != nil {
		t.Fatalf("failed to write profile.yaml: %v", err)
	}

	loader := NewProfileLoaderWithFs(fs, func() (string, error) {
		return "/home/user", nil
	})

	profile, err := loader.Load("acme/backend")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// Verify ContentConfig is populated
	if profile.ContentConfig.Root != "templates" {
		t.Errorf("expected ContentConfig.Root to be 'templates', got '%s'", profile.ContentConfig.Root)
	}
	if profile.ContentConfig.DefaultTarget != ".claude" {
		t.Errorf("expected ContentConfig.DefaultTarget to be '.claude', got '%s'", profile.ContentConfig.DefaultTarget)
	}
	if len(profile.ContentConfig.Targets) != 2 {
		t.Errorf("expected 2 targets, got %d", len(profile.ContentConfig.Targets))
	}
	if profile.ContentConfig.Targets["skills"] != ".claude/skills" {
		t.Errorf("expected Targets['skills'] to be '.claude/skills', got '%s'", profile.ContentConfig.Targets["skills"])
	}
	if profile.ContentConfig.Targets["standards"] != "weftlo/standards" {
		t.Errorf("expected Targets['standards'] to be 'weftlo/standards', got '%s'", profile.ContentConfig.Targets["standards"])
	}
}

// TestLoader_ContentConfigDefaultsAreAppliedWhenContentNotSpecified tests that
// ContentConfig defaults are applied when Content is not specified.
func TestLoader_ContentConfigDefaultsAreAppliedWhenContentNotSpecified(t *testing.T) {
	t.Parallel()
	fs := afero.NewMemMapFs()

	// Create profile directory structure
	profileDir := "/home/user/.weftlo/profiles/acme/backend"
	if err := fs.MkdirAll(profileDir, 0755); err != nil {
		t.Fatalf("failed to create profile directory: %v", err)
	}

	// Create profile.yaml without content configuration
	profileYAML := `name: acme/backend
`
	if err := afero.WriteFile(fs, profileDir+"/profile.yaml", []byte(profileYAML), 0644); err != nil {
		t.Fatalf("failed to write profile.yaml: %v", err)
	}

	loader := NewProfileLoaderWithFs(fs, func() (string, error) {
		return "/home/user", nil
	})

	profile, err := loader.Load("acme/backend")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// Verify ContentConfig defaults are applied
	if profile.ContentConfig.Root != domainprofile.DefaultContentRoot {
		t.Errorf("expected ContentConfig.Root to be '%s', got '%s'", domainprofile.DefaultContentRoot, profile.ContentConfig.Root)
	}
	if profile.ContentConfig.DefaultTarget != domainprofile.DefaultTargetPath {
		t.Errorf("expected ContentConfig.DefaultTarget to be '%s', got '%s'", domainprofile.DefaultTargetPath, profile.ContentConfig.DefaultTarget)
	}
}

// TestLoader_MatcherFieldContainsCombinedFileAndInlinePatterns tests that
// Matcher field contains combined file + inline patterns.
func TestLoader_MatcherFieldContainsCombinedFileAndInlinePatterns(t *testing.T) {
	t.Parallel()
	fs := afero.NewMemMapFs()

	// Create profile directory structure
	profileDir := "/home/user/.weftlo/profiles/acme/backend"
	if err := fs.MkdirAll(profileDir, 0755); err != nil {
		t.Fatalf("failed to create profile directory: %v", err)
	}

	// Create .weftlo.ignore with file-based patterns
	ignoreContent := `# File-based patterns
*.bak
backup/
`
	if err := afero.WriteFile(fs, profileDir+"/.weftlo.ignore", []byte(ignoreContent), 0644); err != nil {
		t.Fatalf("failed to write .weftlo.ignore: %v", err)
	}

	// Create profile.yaml with inline patterns
	profileYAML := `name: acme/backend
ignore:
  - "*.tmp"
  - "logs/"
`
	if err := afero.WriteFile(fs, profileDir+"/profile.yaml", []byte(profileYAML), 0644); err != nil {
		t.Fatalf("failed to write profile.yaml: %v", err)
	}

	loader := NewProfileLoaderWithFs(fs, func() (string, error) {
		return "/home/user", nil
	})

	profile, err := loader.Load("acme/backend")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// Verify Matcher is populated
	if profile.Matcher == nil {
		t.Fatal("expected Matcher to be populated, got nil")
	}

	// Test file-based patterns
	if !profile.Matcher.Match("old.bak", false) {
		t.Error("expected file-based '*.bak' pattern to match 'old.bak'")
	}
	if !profile.Matcher.Match("backup", true) {
		t.Error("expected file-based 'backup/' pattern to match 'backup' directory")
	}

	// Test inline patterns
	if !profile.Matcher.Match("temp.tmp", false) {
		t.Error("expected inline '*.tmp' pattern to match 'temp.tmp'")
	}
	if !profile.Matcher.Match("logs", true) {
		t.Error("expected inline 'logs/' pattern to match 'logs' directory")
	}

	// Test non-matching files
	if profile.Matcher.Match("readme.md", false) {
		t.Error("expected non-matching file 'readme.md' to not be ignored")
	}
}

// TestLoader_PartialContentConfigDefaultsAreApplied tests that partial ContentConfig
// defaults are applied (e.g., Root specified but DefaultTarget not).
func TestLoader_PartialContentConfigDefaultsAreApplied(t *testing.T) {
	t.Parallel()
	fs := afero.NewMemMapFs()

	// Create profile directory structure
	profileDir := "/home/user/.weftlo/profiles/acme/backend"
	if err := fs.MkdirAll(profileDir, 0755); err != nil {
		t.Fatalf("failed to create profile directory: %v", err)
	}

	// Create profile.yaml with partial content configuration (only Root specified)
	profileYAML := `name: acme/backend
content:
  root: templates
`
	if err := afero.WriteFile(fs, profileDir+"/profile.yaml", []byte(profileYAML), 0644); err != nil {
		t.Fatalf("failed to write profile.yaml: %v", err)
	}

	loader := NewProfileLoaderWithFs(fs, func() (string, error) {
		return "/home/user", nil
	})

	profile, err := loader.Load("acme/backend")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// Verify specified value is preserved
	if profile.ContentConfig.Root != "templates" {
		t.Errorf("expected ContentConfig.Root to be 'templates', got '%s'", profile.ContentConfig.Root)
	}

	// Verify default is applied for unspecified field
	if profile.ContentConfig.DefaultTarget != domainprofile.DefaultTargetPath {
		t.Errorf("expected ContentConfig.DefaultTarget to be '%s', got '%s'", domainprofile.DefaultTargetPath, profile.ContentConfig.DefaultTarget)
	}
}

// =============================================================================
// Profile Variables Support Tests (Task Group 1)
// =============================================================================

// TestLoader_ProfileVariablesFieldIsPopulatedFromProfileConfig tests that the
// Profile struct Variables field is populated from ProfileConfig during Load().
func TestLoader_ProfileVariablesFieldIsPopulatedFromProfileConfig(t *testing.T) {
	t.Parallel()
	fs := afero.NewMemMapFs()

	// Create profile directory structure
	profileDir := "/home/user/.weftlo/profiles/acme/backend"
	if err := fs.MkdirAll(profileDir, 0755); err != nil {
		t.Fatalf("failed to create profile directory: %v", err)
	}

	// Create profile.yaml with variables
	profileYAML := `name: acme/backend
variables:
  project_name: MyProject
  version: "1.0.0"
  database:
    host: localhost
    port: 5432
`
	if err := afero.WriteFile(fs, profileDir+"/profile.yaml", []byte(profileYAML), 0644); err != nil {
		t.Fatalf("failed to write profile.yaml: %v", err)
	}

	loader := NewProfileLoaderWithFs(fs, func() (string, error) {
		return "/home/user", nil
	})

	profile, err := loader.Load("acme/backend")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// Verify Variables field is populated
	if profile.Variables == nil {
		t.Fatal("expected Variables to be populated, got nil")
	}

	// Verify scalar values
	if profile.Variables["project_name"] != "MyProject" {
		t.Errorf("expected project_name to be 'MyProject', got '%v'", profile.Variables["project_name"])
	}
	if profile.Variables["version"] != "1.0.0" {
		t.Errorf("expected version to be '1.0.0', got '%v'", profile.Variables["version"])
	}

	// Verify nested map
	database, ok := profile.Variables["database"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected database to be a map, got %T", profile.Variables["database"])
	}
	if database["host"] != "localhost" {
		t.Errorf("expected database.host to be 'localhost', got '%v'", database["host"])
	}
	if database["port"] != 5432 {
		t.Errorf("expected database.port to be 5432, got '%v'", database["port"])
	}
}

// TestLoader_ProfileVariablesFieldIsNilWhenNoVariables tests that the Profile
// Variables field is nil when profile.yaml has no variables defined.
func TestLoader_ProfileVariablesFieldIsNilWhenNoVariables(t *testing.T) {
	t.Parallel()
	fs := afero.NewMemMapFs()

	// Create profile directory structure
	profileDir := "/home/user/.weftlo/profiles/acme/backend"
	if err := fs.MkdirAll(profileDir, 0755); err != nil {
		t.Fatalf("failed to create profile directory: %v", err)
	}

	// Create profile.yaml without variables
	profileYAML := `name: acme/backend
inherits_from: acme/base
`
	if err := afero.WriteFile(fs, profileDir+"/profile.yaml", []byte(profileYAML), 0644); err != nil {
		t.Fatalf("failed to write profile.yaml: %v", err)
	}

	loader := NewProfileLoaderWithFs(fs, func() (string, error) {
		return "/home/user", nil
	})

	profile, err := loader.Load("acme/backend")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// Verify Variables field is nil
	if profile.Variables != nil {
		t.Errorf("expected Variables to be nil, got %v", profile.Variables)
	}
}

// TestLoader_ProfileVariablesFieldIsNilWhenNoProfileYAML tests that the Profile
// Variables field is nil when profile.yaml does not exist.
func TestLoader_ProfileVariablesFieldIsNilWhenNoProfileYAML(t *testing.T) {
	t.Parallel()
	fs := afero.NewMemMapFs()

	// Create profile directory structure without profile.yaml
	profileDir := "/home/user/.weftlo/profiles/acme/backend"
	if err := fs.MkdirAll(profileDir, 0755); err != nil {
		t.Fatalf("failed to create profile directory: %v", err)
	}

	loader := NewProfileLoaderWithFs(fs, func() (string, error) {
		return "/home/user", nil
	})

	profile, err := loader.Load("acme/backend")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// Verify Variables field is nil
	if profile.Variables != nil {
		t.Errorf("expected Variables to be nil, got %v", profile.Variables)
	}
}

// TestMergedProfile_VariablesFieldIsPopulatedAfterNewMergedProfile tests that
// MergedProfile Variables field is populated after NewMergedProfile().
func TestMergedProfile_VariablesFieldIsPopulatedAfterNewMergedProfile(t *testing.T) {
	t.Parallel()
	fs := afero.NewMemMapFs()

	// Create a profile with variables
	profile := &domainprofile.Profile{
		Name:         "acme/base",
		InheritsFrom: "",
		Variables: map[string]interface{}{
			"project_name": "TestProject",
			"version":      "1.0.0",
		},
	}

	chain := []*domainprofile.Profile{profile}
	profilesBasePath := "/home/user/.weftlo/profiles"

	merged := NewMergedProfile(chain, fs, profilesBasePath)

	// Verify Variables field is populated
	if merged.Variables == nil {
		t.Fatal("expected MergedProfile.Variables to be populated, got nil")
	}

	if merged.Variables["project_name"] != "TestProject" {
		t.Errorf("expected project_name to be 'TestProject', got '%v'", merged.Variables["project_name"])
	}
	if merged.Variables["version"] != "1.0.0" {
		t.Errorf("expected version to be '1.0.0', got '%v'", merged.Variables["version"])
	}
}

// TestMergedProfile_ToProfileIncludesVariables tests that MergedProfile.ToProfile()
// includes Variables in the returned Profile struct.
func TestMergedProfile_ToProfileIncludesVariables(t *testing.T) {
	t.Parallel()
	fs := afero.NewMemMapFs()

	// Create a profile with variables
	profile := &domainprofile.Profile{
		Name:         "acme/base",
		InheritsFrom: "",
		Variables: map[string]interface{}{
			"project_name": "TestProject",
			"database": map[string]interface{}{
				"host": "localhost",
				"port": 5432,
			},
		},
	}

	chain := []*domainprofile.Profile{profile}
	profilesBasePath := "/home/user/.weftlo/profiles"

	merged := NewMergedProfile(chain, fs, profilesBasePath)
	result := merged.ToProfile()

	// Verify Variables are included in returned Profile
	if result.Variables == nil {
		t.Fatal("expected ToProfile().Variables to be populated, got nil")
	}

	if result.Variables["project_name"] != "TestProject" {
		t.Errorf("expected project_name to be 'TestProject', got '%v'", result.Variables["project_name"])
	}

	database, ok := result.Variables["database"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected database to be a map, got %T", result.Variables["database"])
	}
	if database["host"] != "localhost" {
		t.Errorf("expected database.host to be 'localhost', got '%v'", database["host"])
	}
}

// TestMergedProfile_VariablesAreMergedFromChain tests that variables from the
// profile inheritance chain are merged (child overrides parent at top level).
func TestMergedProfile_VariablesAreMergedFromChain(t *testing.T) {
	t.Parallel()
	fs := afero.NewMemMapFs()

	// Create parent profile with variables
	parent := &domainprofile.Profile{
		Name:         "acme/parent",
		InheritsFrom: "",
		Variables: map[string]interface{}{
			"from_parent":     "parent_value",
			"overridden":      "parent_original",
			"parent_only_key": "only_in_parent",
		},
	}

	// Create child profile with variables that override some parent values
	child := &domainprofile.Profile{
		Name:         "acme/child",
		InheritsFrom: "acme/parent",
		Variables: map[string]interface{}{
			"from_child": "child_value",
			"overridden": "child_override",
		},
	}

	// Chain is ordered root-to-leaf: [parent, child]
	chain := []*domainprofile.Profile{parent, child}
	profilesBasePath := "/home/user/.weftlo/profiles"

	merged := NewMergedProfile(chain, fs, profilesBasePath)

	// Verify Variables from parent are present
	if merged.Variables["from_parent"] != "parent_value" {
		t.Errorf("expected from_parent to be 'parent_value', got '%v'", merged.Variables["from_parent"])
	}
	if merged.Variables["parent_only_key"] != "only_in_parent" {
		t.Errorf("expected parent_only_key to be 'only_in_parent', got '%v'", merged.Variables["parent_only_key"])
	}

	// Verify Variables from child are present
	if merged.Variables["from_child"] != "child_value" {
		t.Errorf("expected from_child to be 'child_value', got '%v'", merged.Variables["from_child"])
	}

	// Verify child overrides parent
	if merged.Variables["overridden"] != "child_override" {
		t.Errorf("expected overridden to be 'child_override', got '%v'", merged.Variables["overridden"])
	}
}
