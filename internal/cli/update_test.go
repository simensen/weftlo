package cli_test

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/simensen/weftlo/internal/cli"
	"github.com/simensen/weftlo/internal/domain/manifest"
)

// =============================================================================
// Helper Functions
// =============================================================================

// writeGlobalConfig creates ~/.weftlo/config.yaml
func writeGlobalConfig(fs afero.Fs, homeDir string, defaultProfile string) {
	configDir := filepath.Join(homeDir, ".weftlo")
	_ = fs.MkdirAll(configDir, 0755)
	content := fmt.Sprintf("default_profile: %s\n", defaultProfile)
	_ = afero.WriteFile(fs, filepath.Join(configDir, "config.yaml"), []byte(content), 0644)
}

// writeProfile creates a profile with optional templates
// Templates are stored in the content/ subdirectory per the new profile structure
// Files are installed to weftlo/ by default per the content routing architecture
func writeProfile(fs afero.Fs, homeDir string, vendor string, name string, inheritsFrom string, templates map[string]string) {
	profileDir := filepath.Join(homeDir, ".weftlo", "profiles", vendor, name)
	contentDir := filepath.Join(profileDir, "content")
	_ = fs.MkdirAll(contentDir, 0755)

	// Write profile.yaml with content configuration
	profileYaml := fmt.Sprintf("name: %s/%s\n", vendor, name)
	if inheritsFrom != "" {
		profileYaml += fmt.Sprintf("inherits_from: %s\n", inheritsFrom)
	}
	// Content root defaults to "content", default_target defaults to "weftlo"
	profileYaml += "content:\n  root: content\n"
	_ = afero.WriteFile(fs, filepath.Join(profileDir, "profile.yaml"), []byte(profileYaml), 0644)

	// Write templates in content directory (add .tmpl suffix to make them template files)
	for relPath, content := range templates {
		// Add .tmpl suffix if not already present to ensure they're recognized as templates
		tmplPath := relPath
		if !strings.HasSuffix(relPath, ".tmpl") {
			tmplPath = relPath + ".tmpl"
		}
		templatePath := filepath.Join(contentDir, tmplPath)
		_ = fs.MkdirAll(filepath.Dir(templatePath), 0755)
		_ = afero.WriteFile(fs, templatePath, []byte(content), 0644)
	}
}

// writeProjectConfig creates .weftlo.yaml in project directory
func writeProjectConfig(fs afero.Fs, projectDir string, profile string) {
	content := fmt.Sprintf("profiles:\n  - %s\n", profile)
	_ = afero.WriteFile(fs, filepath.Join(projectDir, ".weftlo.yaml"), []byte(content), 0644)
}

// writeManifest creates .weftlo.manifest.json in project directory
func writeManifest(fs afero.Fs, projectDir string, profile string, files map[string]manifest.ManifestFile) {
	m := manifest.Manifest{
		Version:     "1.0.0",
		Profiles:    []string{profile},
		GeneratedAt: time.Now().UTC(),
		Files:       files,
	}
	data, _ := json.MarshalIndent(m, "", "  ")
	_ = afero.WriteFile(fs, filepath.Join(projectDir, ".weftlo.manifest.json"), data, 0644)
}

// writeInstalledFile creates an installed file in the project directory
func writeInstalledFile(fs afero.Fs, projectDir string, relPath string, content string) {
	fullPath := filepath.Join(projectDir, relPath)
	_ = fs.MkdirAll(filepath.Dir(fullPath), 0755)
	_ = afero.WriteFile(fs, fullPath, []byte(content), 0644)
}

// computeChecksum calculates SHA256 checksum of content matching manifest format
func computeChecksum(content string) string {
	hash := sha256.Sum256([]byte(content))
	return "sha256:" + hex.EncodeToString(hash[:])
}

// =============================================================================
// Group 1: Error Types and Command Structure Tests
// =============================================================================

// Test 1.1: InstallationNotFoundError message for missing .weftlo.yaml
func TestInstallationNotFoundError_MissingProjectConfig(t *testing.T) {
	err := &cli.InstallationNotFoundError{MissingFile: ".weftlo.yaml"}
	assert.Contains(t, err.Error(), ".weftlo.yaml")
	assert.Contains(t, err.Error(), "weftlo install")
}

// Test 1.2: InstallationNotFoundError message for missing manifest
func TestInstallationNotFoundError_MissingManifest(t *testing.T) {
	err := &cli.InstallationNotFoundError{MissingFile: ".weftlo.manifest.json"}
	assert.Contains(t, err.Error(), "manifest.json")
}

// Test 1.3: ProfileNotFoundError message format
func TestProfileNotFoundError_Format(t *testing.T) {
	err := &cli.ProfileNotFoundError{ProfileName: "acme/backend"}
	assert.Contains(t, err.Error(), "acme/backend")
	assert.Contains(t, err.Error(), "not found")
}

// Test 1.4: UpdateCommand constructor creates valid command
func TestUpdateCommand_Constructor(t *testing.T) {
	memFs := afero.NewMemMapFs()
	cmd := cli.NewUpdateCommand(memFs)
	assert.Equal(t, "update", cmd.Name())
}

// Test 1.5: Command has all required flags
func TestUpdateCommand_Flags(t *testing.T) {
	memFs := afero.NewMemMapFs()
	cmd := cli.NewUpdateCommand(memFs)

	assert.NotNil(t, cmd.Flags().Lookup("force"))
	assert.NotNil(t, cmd.Flags().Lookup("dry-run"))
	assert.NotNil(t, cmd.Flags().Lookup("quiet"))
	assert.NotNil(t, cmd.Flags().Lookup("remove-orphans"))
}

// Test 1.6: Short flags work
func TestUpdateCommand_ShortFlags(t *testing.T) {
	memFs := afero.NewMemMapFs()
	cmd := cli.NewUpdateCommand(memFs)

	assert.NotNil(t, cmd.Flags().ShorthandLookup("f")) // --force
	assert.NotNil(t, cmd.Flags().ShorthandLookup("n")) // --dry-run
	assert.NotNil(t, cmd.Flags().ShorthandLookup("q")) // --quiet
}

// =============================================================================
// Group 2: Installation Validation Tests
// =============================================================================

// Test 2.1: Error when .weftlo.yaml is missing
func TestUpdateCommand_ErrorWhenProjectConfigMissing(t *testing.T) {
	memFs := afero.NewMemMapFs()
	homeDir := "/home/testuser"
	projectDir := "/projects/myproject"

	_ = memFs.MkdirAll(projectDir, 0755)
	// Note: NO .weftlo.yaml created

	var stdout bytes.Buffer
	cmd := cli.NewUpdateCommandForTesting(memFs, func() (string, error) {
		return homeDir, nil
	}, strings.NewReader(""), &stdout, projectDir)

	err := cmd.Execute()
	require.Error(t, err)

	var notFoundErr *cli.InstallationNotFoundError
	require.True(t, errors.As(err, &notFoundErr))
	assert.Equal(t, ".weftlo.yaml", notFoundErr.MissingFile)
}

// Test 2.2: Error when manifest.json is missing
func TestUpdateCommand_ErrorWhenManifestMissing(t *testing.T) {
	memFs := afero.NewMemMapFs()
	homeDir := "/home/testuser"
	projectDir := "/projects/myproject"

	_ = memFs.MkdirAll(projectDir, 0755)
	writeProjectConfig(memFs, projectDir, "default/default")
	// Note: NO manifest.json created

	var stdout bytes.Buffer
	cmd := cli.NewUpdateCommandForTesting(memFs, func() (string, error) {
		return homeDir, nil
	}, strings.NewReader(""), &stdout, projectDir)

	err := cmd.Execute()
	require.Error(t, err)

	var notFoundErr *cli.InstallationNotFoundError
	require.True(t, errors.As(err, &notFoundErr))
	assert.Contains(t, notFoundErr.MissingFile, "manifest.json")
}

// Test 2.3: Error when profile no longer exists
func TestUpdateCommand_ErrorWhenProfileDeleted(t *testing.T) {
	memFs := afero.NewMemMapFs()
	homeDir := "/home/testuser"
	projectDir := "/projects/myproject"

	// Set up project config and manifest referencing a profile that doesn't exist
	_ = memFs.MkdirAll(projectDir, 0755)
	writeProjectConfig(memFs, projectDir, "deleted/profile")
	writeManifest(memFs, projectDir, "deleted/profile", map[string]manifest.ManifestFile{})
	writeGlobalConfig(memFs, homeDir, "default/default")
	// Note: NO profile created for "deleted/profile"

	var stdout bytes.Buffer
	cmd := cli.NewUpdateCommandForTesting(memFs, func() (string, error) {
		return homeDir, nil
	}, strings.NewReader(""), &stdout, projectDir)

	err := cmd.Execute()
	require.Error(t, err)

	var profileErr *cli.ProfileNotFoundError
	require.True(t, errors.As(err, &profileErr))
	assert.Equal(t, "deleted/profile", profileErr.ProfileName)
}

// Test 2.4: Successful loading of all components
func TestUpdateCommand_SuccessfulLoading(t *testing.T) {
	memFs := afero.NewMemMapFs()
	homeDir := "/home/testuser"
	projectDir := "/projects/myproject"

	_ = memFs.MkdirAll(projectDir, 0755)
	writeGlobalConfig(memFs, homeDir, "default/default")
	writeProfile(memFs, homeDir, "default", "default", "", map[string]string{})
	writeProjectConfig(memFs, projectDir, "default/default")
	writeManifest(memFs, projectDir, "default/default", map[string]manifest.ManifestFile{})

	var stdout bytes.Buffer
	cmd := cli.NewUpdateCommandForTesting(memFs, func() (string, error) {
		return homeDir, nil
	}, strings.NewReader(""), &stdout, projectDir)

	err := cmd.Execute()
	require.NoError(t, err)
	// Should show "No changes needed" since profile has no templates
	assert.Contains(t, stdout.String(), "No changes needed")
}

// Test 2.5: Invalid profile name format
func TestUpdateCommand_InvalidProfileNameFormat(t *testing.T) {
	memFs := afero.NewMemMapFs()
	homeDir := "/home/testuser"
	projectDir := "/projects/myproject"

	_ = memFs.MkdirAll(projectDir, 0755)
	writeGlobalConfig(memFs, homeDir, "default/default")
	writeProjectConfig(memFs, projectDir, "invalid-no-slash") // Invalid format
	writeManifest(memFs, projectDir, "invalid-no-slash", map[string]manifest.ManifestFile{})

	var stdout bytes.Buffer
	cmd := cli.NewUpdateCommandForTesting(memFs, func() (string, error) {
		return homeDir, nil
	}, strings.NewReader(""), &stdout, projectDir)

	err := cmd.Execute()
	require.Error(t, err)
	// Should fail validation
}

// =============================================================================
// Group 3: Change Detection and Planning Tests
// =============================================================================

// Test 3.1: No changes needed when files unchanged
func TestUpdateCommand_NoChangesNeeded(t *testing.T) {
	memFs := afero.NewMemMapFs()
	homeDir := "/home/testuser"
	projectDir := "/projects/myproject"

	templateContent := "# README"
	checksum := computeChecksum(templateContent)

	_ = memFs.MkdirAll(projectDir, 0755)
	writeGlobalConfig(memFs, homeDir, "default/default")
	writeProfile(memFs, homeDir, "default", "default", "", map[string]string{
		"README.md": templateContent,
	})
	writeProjectConfig(memFs, projectDir, "default/default")
	writeManifest(memFs, projectDir, "default/default", map[string]manifest.ManifestFile{
		".claude/README.md": {
			SourceChecksum: checksum,
			OutputChecksum: checksum,
		},
	})
	writeInstalledFile(memFs, projectDir, ".claude/README.md", templateContent)

	var stdout bytes.Buffer
	cmd := cli.NewUpdateCommandForTesting(memFs, func() (string, error) {
		return homeDir, nil
	}, strings.NewReader(""), &stdout, projectDir)

	err := cmd.Execute()
	require.NoError(t, err)
	assert.Contains(t, stdout.String(), "No changes needed")
}

// Test 3.2: Source changed files are updated
func TestUpdateCommand_SourceChangedFilesUpdated(t *testing.T) {
	memFs := afero.NewMemMapFs()
	homeDir := "/home/testuser"
	projectDir := "/projects/myproject"

	oldContent := "# Old README"
	newContent := "# New README"
	oldChecksum := computeChecksum(oldContent)

	_ = memFs.MkdirAll(projectDir, 0755)
	writeGlobalConfig(memFs, homeDir, "default/default")
	writeProfile(memFs, homeDir, "default", "default", "", map[string]string{
		"README.md": newContent, // Template has new content
	})
	writeProjectConfig(memFs, projectDir, "default/default")
	writeManifest(memFs, projectDir, "default/default", map[string]manifest.ManifestFile{
		".claude/README.md": {
			SourceChecksum: oldChecksum, // Manifest has old checksum
			OutputChecksum: oldChecksum,
		},
	})
	writeInstalledFile(memFs, projectDir, ".claude/README.md", oldContent)

	var stdout bytes.Buffer
	cmd := cli.NewUpdateCommandForTesting(memFs, func() (string, error) {
		return homeDir, nil
	}, strings.NewReader(""), &stdout, projectDir)

	err := cmd.Execute()
	require.NoError(t, err)

	// Verify file was updated
	content, _ := afero.ReadFile(memFs, filepath.Join(projectDir, ".claude", "README.md"))
	assert.Equal(t, newContent, string(content))
	assert.Contains(t, stdout.String(), "updated")
}

// Test 3.3: User-modified files are skipped (without --force)
func TestUpdateCommand_UserModifiedFilesSkipped(t *testing.T) {
	memFs := afero.NewMemMapFs()
	homeDir := "/home/testuser"
	projectDir := "/projects/myproject"

	templateContent := "# Template README"
	userContent := "# User modified README"
	templateChecksum := computeChecksum(templateContent)

	_ = memFs.MkdirAll(projectDir, 0755)
	writeGlobalConfig(memFs, homeDir, "default/default")
	writeProfile(memFs, homeDir, "default", "default", "", map[string]string{
		"README.md": templateContent,
	})
	writeProjectConfig(memFs, projectDir, "default/default")
	writeManifest(memFs, projectDir, "default/default", map[string]manifest.ManifestFile{
		".claude/README.md": {
			SourceChecksum: templateChecksum,
			OutputChecksum: templateChecksum, // Original installed checksum
		},
	})
	writeInstalledFile(memFs, projectDir, ".claude/README.md", userContent) // User modified!

	var stdout bytes.Buffer
	cmd := cli.NewUpdateCommandForTesting(memFs, func() (string, error) {
		return homeDir, nil
	}, strings.NewReader(""), &stdout, projectDir)

	err := cmd.Execute()
	require.NoError(t, err)

	// Verify file was NOT updated (user changes preserved)
	content, _ := afero.ReadFile(memFs, filepath.Join(projectDir, ".claude", "README.md"))
	assert.Equal(t, userContent, string(content))
	assert.Contains(t, stdout.String(), "skipped")
}

// Test 3.4: --force updates user-modified files
func TestUpdateCommand_ForceUpdatesUserModifiedFiles(t *testing.T) {
	memFs := afero.NewMemMapFs()
	homeDir := "/home/testuser"
	projectDir := "/projects/myproject"

	templateContent := "# Template README"
	userContent := "# User modified README"
	templateChecksum := computeChecksum(templateContent)

	_ = memFs.MkdirAll(projectDir, 0755)
	writeGlobalConfig(memFs, homeDir, "default/default")
	writeProfile(memFs, homeDir, "default", "default", "", map[string]string{
		"README.md": templateContent,
	})
	writeProjectConfig(memFs, projectDir, "default/default")
	writeManifest(memFs, projectDir, "default/default", map[string]manifest.ManifestFile{
		".claude/README.md": {
			SourceChecksum: templateChecksum,
			OutputChecksum: templateChecksum,
		},
	})
	writeInstalledFile(memFs, projectDir, ".claude/README.md", userContent)

	var stdout bytes.Buffer
	cmd := cli.NewUpdateCommandForTesting(memFs, func() (string, error) {
		return homeDir, nil
	}, strings.NewReader(""), &stdout, projectDir)
	cmd.SetArgs([]string{"--force"})

	err := cmd.Execute()
	require.NoError(t, err)

	// Verify file WAS updated (user changes overwritten)
	content, _ := afero.ReadFile(memFs, filepath.Join(projectDir, ".claude", "README.md"))
	assert.Equal(t, templateContent, string(content))
}

// Test 3.5: --remove-orphans deletes orphan files
func TestUpdateCommand_RemoveOrphansDeletesOrphanFiles(t *testing.T) {
	memFs := afero.NewMemMapFs()
	homeDir := "/home/testuser"
	projectDir := "/projects/myproject"

	orphanContent := "# Orphan file"
	orphanChecksum := computeChecksum(orphanContent)

	_ = memFs.MkdirAll(projectDir, 0755)
	writeGlobalConfig(memFs, homeDir, "default/default")
	writeProfile(memFs, homeDir, "default", "default", "", map[string]string{}) // No templates!
	writeProjectConfig(memFs, projectDir, "default/default")
	writeManifest(memFs, projectDir, "default/default", map[string]manifest.ManifestFile{
		".claude/orphan.md": { // File in manifest but not in profile
			SourceChecksum: orphanChecksum,
			OutputChecksum: orphanChecksum,
		},
	})
	writeInstalledFile(memFs, projectDir, ".claude/orphan.md", orphanContent)

	var stdout bytes.Buffer
	cmd := cli.NewUpdateCommandForTesting(memFs, func() (string, error) {
		return homeDir, nil
	}, strings.NewReader(""), &stdout, projectDir)
	cmd.SetArgs([]string{"--remove-orphans"})

	err := cmd.Execute()
	require.NoError(t, err)

	// Verify orphan file was deleted
	exists, _ := afero.Exists(memFs, filepath.Join(projectDir, ".claude", "orphan.md"))
	assert.False(t, exists)
	assert.Contains(t, stdout.String(), "deleted")
}

// =============================================================================
// Group 4: File Operations Tests
// =============================================================================

// Test 4.1: New files are created with correct permissions
func TestUpdateCommand_CreatesNewFiles(t *testing.T) {
	memFs := afero.NewMemMapFs()
	homeDir := "/home/testuser"
	projectDir := "/projects/myproject"

	templateContent := "# New File"

	_ = memFs.MkdirAll(projectDir, 0755)
	writeGlobalConfig(memFs, homeDir, "default/default")
	writeProfile(memFs, homeDir, "default", "default", "", map[string]string{
		"docs/guide.md": templateContent,
	})
	writeProjectConfig(memFs, projectDir, "default/default")
	writeManifest(memFs, projectDir, "default/default", map[string]manifest.ManifestFile{}) // Empty - no files yet

	var stdout bytes.Buffer
	cmd := cli.NewUpdateCommandForTesting(memFs, func() (string, error) {
		return homeDir, nil
	}, strings.NewReader(""), &stdout, projectDir)

	err := cmd.Execute()
	require.NoError(t, err)

	// Verify file was created in weftlo/ directory
	content, err := afero.ReadFile(memFs, filepath.Join(projectDir, ".claude", "docs", "guide.md"))
	require.NoError(t, err)
	assert.Equal(t, templateContent, string(content))
	assert.Contains(t, stdout.String(), "created")
}

// Test 4.2: Parent directories are created
func TestUpdateCommand_CreatesParentDirectories(t *testing.T) {
	memFs := afero.NewMemMapFs()
	homeDir := "/home/testuser"
	projectDir := "/projects/myproject"

	_ = memFs.MkdirAll(projectDir, 0755)
	writeGlobalConfig(memFs, homeDir, "default/default")
	writeProfile(memFs, homeDir, "default", "default", "", map[string]string{
		"deeply/nested/path/file.md": "content",
	})
	writeProjectConfig(memFs, projectDir, "default/default")
	writeManifest(memFs, projectDir, "default/default", map[string]manifest.ManifestFile{})

	var stdout bytes.Buffer
	cmd := cli.NewUpdateCommandForTesting(memFs, func() (string, error) {
		return homeDir, nil
	}, strings.NewReader(""), &stdout, projectDir)

	err := cmd.Execute()
	require.NoError(t, err)

	// Verify nested directories were created under weftlo/
	exists, _ := afero.DirExists(memFs, filepath.Join(projectDir, ".claude", "deeply", "nested", "path"))
	assert.True(t, exists)
}

// Test 4.3: --dry-run prevents file writes
func TestUpdateCommand_DryRunPreventsWrites(t *testing.T) {
	memFs := afero.NewMemMapFs()
	homeDir := "/home/testuser"
	projectDir := "/projects/myproject"

	_ = memFs.MkdirAll(projectDir, 0755)
	writeGlobalConfig(memFs, homeDir, "default/default")
	writeProfile(memFs, homeDir, "default", "default", "", map[string]string{
		"new-file.md": "content",
	})
	writeProjectConfig(memFs, projectDir, "default/default")
	writeManifest(memFs, projectDir, "default/default", map[string]manifest.ManifestFile{})

	var stdout bytes.Buffer
	cmd := cli.NewUpdateCommandForTesting(memFs, func() (string, error) {
		return homeDir, nil
	}, strings.NewReader(""), &stdout, projectDir)
	cmd.SetArgs([]string{"--dry-run"})

	err := cmd.Execute()
	require.NoError(t, err)

	// Verify file was NOT created (not even in weftlo/)
	exists, _ := afero.Exists(memFs, filepath.Join(projectDir, ".claude", "new-file.md"))
	assert.False(t, exists)

	// Verify dry-run indicator in output
	assert.Contains(t, stdout.String(), "dry-run")
}

// Test 4.4: File deletion works
func TestUpdateCommand_DeletesFiles(t *testing.T) {
	memFs := afero.NewMemMapFs()
	homeDir := "/home/testuser"
	projectDir := "/projects/myproject"

	orphanChecksum := computeChecksum("orphan content")

	_ = memFs.MkdirAll(projectDir, 0755)
	writeGlobalConfig(memFs, homeDir, "default/default")
	writeProfile(memFs, homeDir, "default", "default", "", map[string]string{})
	writeProjectConfig(memFs, projectDir, "default/default")
	writeManifest(memFs, projectDir, "default/default", map[string]manifest.ManifestFile{
		".claude/to-delete.md": {
			SourceChecksum: orphanChecksum,
			OutputChecksum: orphanChecksum,
		},
	})
	writeInstalledFile(memFs, projectDir, ".claude/to-delete.md", "orphan content")

	var stdout bytes.Buffer
	cmd := cli.NewUpdateCommandForTesting(memFs, func() (string, error) {
		return homeDir, nil
	}, strings.NewReader(""), &stdout, projectDir)
	cmd.SetArgs([]string{"--remove-orphans"})

	err := cmd.Execute()
	require.NoError(t, err)

	exists, _ := afero.Exists(memFs, filepath.Join(projectDir, ".claude", "to-delete.md"))
	assert.False(t, exists)
}

// Test 4.5: Multiple operations in sequence
func TestUpdateCommand_MultipleOperations(t *testing.T) {
	memFs := afero.NewMemMapFs()
	homeDir := "/home/testuser"
	projectDir := "/projects/myproject"

	oldContent := "# Old"
	newContent := "# New"
	oldChecksum := computeChecksum(oldContent)
	orphanChecksum := computeChecksum("orphan")

	_ = memFs.MkdirAll(projectDir, 0755)
	writeGlobalConfig(memFs, homeDir, "default/default")
	writeProfile(memFs, homeDir, "default", "default", "", map[string]string{
		"existing.md": newContent,  // Will be updated
		"new-file.md": "brand new", // Will be created
	})
	writeProjectConfig(memFs, projectDir, "default/default")
	writeManifest(memFs, projectDir, "default/default", map[string]manifest.ManifestFile{
		".claude/existing.md": {
			SourceChecksum: oldChecksum,
			OutputChecksum: oldChecksum,
		},
		".claude/orphan.md": {
			SourceChecksum: orphanChecksum,
			OutputChecksum: orphanChecksum,
		},
	})
	writeInstalledFile(memFs, projectDir, ".claude/existing.md", oldContent)
	writeInstalledFile(memFs, projectDir, ".claude/orphan.md", "orphan")

	var stdout bytes.Buffer
	cmd := cli.NewUpdateCommandForTesting(memFs, func() (string, error) {
		return homeDir, nil
	}, strings.NewReader(""), &stdout, projectDir)
	cmd.SetArgs([]string{"--remove-orphans"})

	err := cmd.Execute()
	require.NoError(t, err)

	// Verify all operations completed
	output := stdout.String()
	assert.Contains(t, output, "created")
	assert.Contains(t, output, "updated")
	assert.Contains(t, output, "deleted")
}

// =============================================================================
// Group 5: Manifest and Output Tests
// =============================================================================

// Test 5.1: Manifest is regenerated after update
func TestUpdateCommand_RegeneratesManifest(t *testing.T) {
	memFs := afero.NewMemMapFs()
	homeDir := "/home/testuser"
	projectDir := "/projects/myproject"

	_ = memFs.MkdirAll(projectDir, 0755)
	writeGlobalConfig(memFs, homeDir, "default/default")
	writeProfile(memFs, homeDir, "default", "default", "", map[string]string{
		"file.md": "content",
	})
	writeProjectConfig(memFs, projectDir, "default/default")
	writeManifest(memFs, projectDir, "default/default", map[string]manifest.ManifestFile{})

	var stdout bytes.Buffer
	cmd := cli.NewUpdateCommandForTesting(memFs, func() (string, error) {
		return homeDir, nil
	}, strings.NewReader(""), &stdout, projectDir)

	err := cmd.Execute()
	require.NoError(t, err)

	// Read and verify manifest
	manifestPath := filepath.Join(projectDir, ".weftlo.manifest.json")
	data, err := afero.ReadFile(memFs, manifestPath)
	require.NoError(t, err)

	var m manifest.Manifest
	err = json.Unmarshal(data, &m)
	require.NoError(t, err)

	assert.Contains(t, m.Files, ".claude/file.md")
}

// Test 5.2: Manifest NOT regenerated during dry-run
func TestUpdateCommand_DryRunSkipsManifestRegeneration(t *testing.T) {
	memFs := afero.NewMemMapFs()
	homeDir := "/home/testuser"
	projectDir := "/projects/myproject"

	_ = memFs.MkdirAll(projectDir, 0755)
	writeGlobalConfig(memFs, homeDir, "default/default")
	writeProfile(memFs, homeDir, "default", "default", "", map[string]string{
		"file.md": "content",
	})
	writeProjectConfig(memFs, projectDir, "default/default")
	writeManifest(memFs, projectDir, "default/default", map[string]manifest.ManifestFile{})

	// Get original manifest content
	manifestPath := filepath.Join(projectDir, ".weftlo.manifest.json")
	originalData, _ := afero.ReadFile(memFs, manifestPath)

	var stdout bytes.Buffer
	cmd := cli.NewUpdateCommandForTesting(memFs, func() (string, error) {
		return homeDir, nil
	}, strings.NewReader(""), &stdout, projectDir)
	cmd.SetArgs([]string{"--dry-run"})

	err := cmd.Execute()
	require.NoError(t, err)

	// Verify manifest unchanged
	newData, _ := afero.ReadFile(memFs, manifestPath)
	assert.Equal(t, originalData, newData)
}

// Test 5.3: Quiet mode suppresses output
func TestUpdateCommand_QuietModeSuppressesOutput(t *testing.T) {
	memFs := afero.NewMemMapFs()
	homeDir := "/home/testuser"
	projectDir := "/projects/myproject"

	_ = memFs.MkdirAll(projectDir, 0755)
	writeGlobalConfig(memFs, homeDir, "default/default")
	writeProfile(memFs, homeDir, "default", "default", "", map[string]string{
		"file.md": "content",
	})
	writeProjectConfig(memFs, projectDir, "default/default")
	writeManifest(memFs, projectDir, "default/default", map[string]manifest.ManifestFile{})

	var stdout bytes.Buffer
	cmd := cli.NewUpdateCommandForTesting(memFs, func() (string, error) {
		return homeDir, nil
	}, strings.NewReader(""), &stdout, projectDir)
	cmd.SetArgs([]string{"--quiet"})

	err := cmd.Execute()
	require.NoError(t, err)

	// Verify minimal output
	assert.Empty(t, stdout.String())
}

// Test 5.4: Skipped files show rationale
func TestUpdateCommand_SkippedFilesShowRationale(t *testing.T) {
	memFs := afero.NewMemMapFs()
	homeDir := "/home/testuser"
	projectDir := "/projects/myproject"

	templateContent := "# Template"
	templateChecksum := computeChecksum(templateContent)

	_ = memFs.MkdirAll(projectDir, 0755)
	writeGlobalConfig(memFs, homeDir, "default/default")
	writeProfile(memFs, homeDir, "default", "default", "", map[string]string{
		"README.md": templateContent,
	})
	writeProjectConfig(memFs, projectDir, "default/default")
	writeManifest(memFs, projectDir, "default/default", map[string]manifest.ManifestFile{
		".claude/README.md": {
			SourceChecksum: templateChecksum,
			OutputChecksum: templateChecksum,
		},
	})
	writeInstalledFile(memFs, projectDir, ".claude/README.md", "user modified content")

	var stdout bytes.Buffer
	cmd := cli.NewUpdateCommandForTesting(memFs, func() (string, error) {
		return homeDir, nil
	}, strings.NewReader(""), &stdout, projectDir)

	err := cmd.Execute()
	require.NoError(t, err)

	output := stdout.String()
	assert.Contains(t, output, "README.md")
	assert.Contains(t, output, "skipped")
}

// =============================================================================
// Group 6: Integration Tests
// =============================================================================

// Test 6.1: Full update workflow with mixed statuses
func TestUpdateCommand_FullWorkflowMixedStatuses(t *testing.T) {
	memFs := afero.NewMemMapFs()
	homeDir := "/home/testuser"
	projectDir := "/projects/myproject"

	// Set up scenario with:
	// - 1 unchanged file
	// - 1 source-changed file
	// - 1 user-modified file
	// - 1 new file
	// - 1 orphan file

	unchangedContent := "# Unchanged"
	unchangedChecksum := computeChecksum(unchangedContent)

	oldSourceContent := "# Old Source"
	newSourceContent := "# New Source"
	oldSourceChecksum := computeChecksum(oldSourceContent)

	templateForUserMod := "# Template"
	templateForUserModChecksum := computeChecksum(templateForUserMod)

	orphanContent := "# Orphan"
	orphanChecksum := computeChecksum(orphanContent)

	_ = memFs.MkdirAll(projectDir, 0755)
	writeGlobalConfig(memFs, homeDir, "default/default")
	writeProfile(memFs, homeDir, "default", "default", "", map[string]string{
		"unchanged.md":     unchangedContent,
		"source-change.md": newSourceContent,
		"user-modified.md": templateForUserMod,
		"new-file.md":      "Brand new file content",
		// Note: orphan.md NOT in profile
	})
	writeProjectConfig(memFs, projectDir, "default/default")
	writeManifest(memFs, projectDir, "default/default", map[string]manifest.ManifestFile{
		".claude/unchanged.md": {
			SourceChecksum: unchangedChecksum,
			OutputChecksum: unchangedChecksum,
		},
		".claude/source-change.md": {
			SourceChecksum: oldSourceChecksum,
			OutputChecksum: oldSourceChecksum,
		},
		".claude/user-modified.md": {
			SourceChecksum: templateForUserModChecksum,
			OutputChecksum: templateForUserModChecksum,
		},
		".claude/orphan.md": {
			SourceChecksum: orphanChecksum,
			OutputChecksum: orphanChecksum,
		},
	})
	writeInstalledFile(memFs, projectDir, ".claude/unchanged.md", unchangedContent)
	writeInstalledFile(memFs, projectDir, ".claude/source-change.md", oldSourceContent)
	writeInstalledFile(memFs, projectDir, ".claude/user-modified.md", "# User Modified This")
	writeInstalledFile(memFs, projectDir, ".claude/orphan.md", orphanContent)
	// Note: new-file.md doesn't exist yet

	var stdout bytes.Buffer
	cmd := cli.NewUpdateCommandForTesting(memFs, func() (string, error) {
		return homeDir, nil
	}, strings.NewReader(""), &stdout, projectDir)
	cmd.SetArgs([]string{"--remove-orphans"})

	err := cmd.Execute()
	require.NoError(t, err)

	// Verify unchanged file remains unchanged
	content, _ := afero.ReadFile(memFs, filepath.Join(projectDir, ".claude", "unchanged.md"))
	assert.Equal(t, unchangedContent, string(content))

	// Verify source-changed file was updated
	content, _ = afero.ReadFile(memFs, filepath.Join(projectDir, ".claude", "source-change.md"))
	assert.Equal(t, newSourceContent, string(content))

	// Verify user-modified file was skipped (not overwritten)
	content, _ = afero.ReadFile(memFs, filepath.Join(projectDir, ".claude", "user-modified.md"))
	assert.Equal(t, "# User Modified This", string(content))

	// Verify new file was created
	content, _ = afero.ReadFile(memFs, filepath.Join(projectDir, ".claude", "new-file.md"))
	assert.Equal(t, "Brand new file content", string(content))

	// Verify orphan file was deleted
	exists, _ := afero.Exists(memFs, filepath.Join(projectDir, ".claude", "orphan.md"))
	assert.False(t, exists)
}

// Test 6.2: Profile inheritance works during update
func TestUpdateCommand_ProfileInheritance(t *testing.T) {
	memFs := afero.NewMemMapFs()
	homeDir := "/home/testuser"
	projectDir := "/projects/myproject"

	// Create parent profile with a template
	_ = memFs.MkdirAll(projectDir, 0755)
	writeGlobalConfig(memFs, homeDir, "child/profile")
	writeProfile(memFs, homeDir, "parent", "base", "", map[string]string{
		"inherited.md": "# From Parent",
	})
	// Create child profile that inherits from parent
	writeProfile(memFs, homeDir, "child", "profile", "parent/base", map[string]string{
		"child-only.md": "# From Child",
	})
	writeProjectConfig(memFs, projectDir, "child/profile")
	writeManifest(memFs, projectDir, "child/profile", map[string]manifest.ManifestFile{})

	var stdout bytes.Buffer
	cmd := cli.NewUpdateCommandForTesting(memFs, func() (string, error) {
		return homeDir, nil
	}, strings.NewReader(""), &stdout, projectDir)

	err := cmd.Execute()
	require.NoError(t, err)

	// Verify inherited file was created in weftlo/
	content, err := afero.ReadFile(memFs, filepath.Join(projectDir, ".claude", "inherited.md"))
	require.NoError(t, err)
	assert.Equal(t, "# From Parent", string(content))

	// Verify child-specific file was created in weftlo/
	content, err = afero.ReadFile(memFs, filepath.Join(projectDir, ".claude", "child-only.md"))
	require.NoError(t, err)
	assert.Equal(t, "# From Child", string(content))
}

// Test 6.3: Edge case - all files unchanged
func TestUpdateCommand_AllFilesUnchanged(t *testing.T) {
	memFs := afero.NewMemMapFs()
	homeDir := "/home/testuser"
	projectDir := "/projects/myproject"

	file1Content := "# File 1"
	file2Content := "# File 2"
	file1Checksum := computeChecksum(file1Content)
	file2Checksum := computeChecksum(file2Content)

	_ = memFs.MkdirAll(projectDir, 0755)
	writeGlobalConfig(memFs, homeDir, "default/default")
	writeProfile(memFs, homeDir, "default", "default", "", map[string]string{
		"file1.md": file1Content,
		"file2.md": file2Content,
	})
	writeProjectConfig(memFs, projectDir, "default/default")
	writeManifest(memFs, projectDir, "default/default", map[string]manifest.ManifestFile{
		".claude/file1.md": {
			SourceChecksum: file1Checksum,
			OutputChecksum: file1Checksum,
		},
		".claude/file2.md": {
			SourceChecksum: file2Checksum,
			OutputChecksum: file2Checksum,
		},
	})
	writeInstalledFile(memFs, projectDir, ".claude/file1.md", file1Content)
	writeInstalledFile(memFs, projectDir, ".claude/file2.md", file2Content)

	var stdout bytes.Buffer
	cmd := cli.NewUpdateCommandForTesting(memFs, func() (string, error) {
		return homeDir, nil
	}, strings.NewReader(""), &stdout, projectDir)

	err := cmd.Execute()
	require.NoError(t, err)

	// Verify clean exit with "No changes needed"
	assert.Contains(t, stdout.String(), "No changes needed")
}

// Test 6.4: Edge case - empty profile (no templates)
func TestUpdateCommand_EmptyProfile(t *testing.T) {
	memFs := afero.NewMemMapFs()
	homeDir := "/home/testuser"
	projectDir := "/projects/myproject"

	_ = memFs.MkdirAll(projectDir, 0755)
	writeGlobalConfig(memFs, homeDir, "empty/profile")
	writeProfile(memFs, homeDir, "empty", "profile", "", map[string]string{}) // No templates
	writeProjectConfig(memFs, projectDir, "empty/profile")
	writeManifest(memFs, projectDir, "empty/profile", map[string]manifest.ManifestFile{})

	var stdout bytes.Buffer
	cmd := cli.NewUpdateCommandForTesting(memFs, func() (string, error) {
		return homeDir, nil
	}, strings.NewReader(""), &stdout, projectDir)

	err := cmd.Execute()
	require.NoError(t, err)

	// Verify it handles gracefully
	assert.Contains(t, stdout.String(), "No changes needed")
}

// Test 6.5: Templates with Go template syntax are rendered
func TestUpdateCommand_TemplatesRenderedWithGoSyntax(t *testing.T) {
	memFs := afero.NewMemMapFs()
	homeDir := "/home/testuser"
	projectDir := "/projects/myproject"

	// Create a profile with a template file using Go template syntax
	templateContent := "# {{ .Profile.Name }}"

	_ = memFs.MkdirAll(projectDir, 0755)
	writeGlobalConfig(memFs, homeDir, "default/default")
	writeProfile(memFs, homeDir, "default", "default", "", map[string]string{
		"output.md": templateContent,
	})
	writeProjectConfig(memFs, projectDir, "default/default")
	writeManifest(memFs, projectDir, "default/default", map[string]manifest.ManifestFile{})

	var stdout bytes.Buffer
	cmd := cli.NewUpdateCommandForTesting(memFs, func() (string, error) {
		return homeDir, nil
	}, strings.NewReader(""), &stdout, projectDir)

	err := cmd.Execute()
	require.NoError(t, err)

	// Verify file was created in weftlo/ with template rendered
	content, err := afero.ReadFile(memFs, filepath.Join(projectDir, ".claude", "output.md"))
	require.NoError(t, err)
	assert.Equal(t, "# default/default", string(content))
}

// =============================================================================
// Group 7: Profile Modification Tests
// =============================================================================

// projectConfigYAML is used to parse .weftlo.yaml in tests
type projectConfigYAML struct {
	Profiles []string `yaml:"profiles"`
}

// Test 7.1: --add-profile adds a profile to .weftlo.yaml and installs files
func TestUpdateCommand_AddProfileFlag(t *testing.T) {
	memFs := afero.NewMemMapFs()
	homeDir := "/home/testuser"
	projectDir := "/projects/myproject"

	_ = memFs.MkdirAll(projectDir, 0755)
	writeGlobalConfig(memFs, homeDir, "default/default")
	writeProfile(memFs, homeDir, "default", "default", "", map[string]string{
		"default.md": "# Default profile content",
	})
	writeProfile(memFs, homeDir, "new", "profile", "", map[string]string{
		"newfile.md": "# New profile content",
	})
	writeProjectConfig(memFs, projectDir, "default/default")
	writeManifest(memFs, projectDir, "default/default", map[string]manifest.ManifestFile{})

	var stdout bytes.Buffer
	cmd := cli.NewUpdateCommandForTesting(memFs, func() (string, error) {
		return homeDir, nil
	}, strings.NewReader(""), &stdout, projectDir)
	cmd.SetArgs([]string{"--add-profile", "new/profile"})

	err := cmd.Execute()
	require.NoError(t, err)

	// Verify .weftlo.yaml was updated with both profiles
	configData, err := afero.ReadFile(memFs, filepath.Join(projectDir, ".weftlo.yaml"))
	require.NoError(t, err)

	var config projectConfigYAML
	err = yaml.Unmarshal(configData, &config)
	require.NoError(t, err)

	assert.Equal(t, []string{"default/default", "new/profile"}, config.Profiles)

	// Verify file from new profile was installed
	exists, _ := afero.Exists(memFs, filepath.Join(projectDir, ".claude", "newfile.md"))
	assert.True(t, exists, "file from added profile should be installed")
}

// Test 7.2: --remove-profile removes a profile from .weftlo.yaml
func TestUpdateCommand_RemoveProfileFlag(t *testing.T) {
	memFs := afero.NewMemMapFs()
	homeDir := "/home/testuser"
	projectDir := "/projects/myproject"

	_ = memFs.MkdirAll(projectDir, 0755)
	writeGlobalConfig(memFs, homeDir, "default/default")
	writeProfile(memFs, homeDir, "default", "default", "", map[string]string{
		"default.md": "# Default content",
	})
	writeProfile(memFs, homeDir, "other", "profile", "", map[string]string{
		"other.md": "# Other content",
	})

	// Create project config with two profiles
	multiProfileConfig := "profiles:\n  - default/default\n  - other/profile\n"
	_ = afero.WriteFile(memFs, filepath.Join(projectDir, ".weftlo.yaml"), []byte(multiProfileConfig), 0644)
	writeManifest(memFs, projectDir, "default/default", map[string]manifest.ManifestFile{})

	var stdout bytes.Buffer
	cmd := cli.NewUpdateCommandForTesting(memFs, func() (string, error) {
		return homeDir, nil
	}, strings.NewReader(""), &stdout, projectDir)
	cmd.SetArgs([]string{"--remove-profile", "other/profile"})

	err := cmd.Execute()
	require.NoError(t, err)

	// Verify .weftlo.yaml only has one profile now
	configData, err := afero.ReadFile(memFs, filepath.Join(projectDir, ".weftlo.yaml"))
	require.NoError(t, err)

	var config projectConfigYAML
	err = yaml.Unmarshal(configData, &config)
	require.NoError(t, err)

	assert.Equal(t, []string{"default/default"}, config.Profiles)
}

// Test 7.3: Combined --add-profile and --remove-profile in one operation
func TestUpdateCommand_AddAndRemoveProfileFlags(t *testing.T) {
	memFs := afero.NewMemMapFs()
	homeDir := "/home/testuser"
	projectDir := "/projects/myproject"

	_ = memFs.MkdirAll(projectDir, 0755)
	writeGlobalConfig(memFs, homeDir, "default/default")
	writeProfile(memFs, homeDir, "default", "default", "", map[string]string{})
	writeProfile(memFs, homeDir, "old", "profile", "", map[string]string{})
	writeProfile(memFs, homeDir, "new", "profile", "", map[string]string{})

	// Create project config with default and old profiles
	multiProfileConfig := "profiles:\n  - default/default\n  - old/profile\n"
	_ = afero.WriteFile(memFs, filepath.Join(projectDir, ".weftlo.yaml"), []byte(multiProfileConfig), 0644)
	writeManifest(memFs, projectDir, "default/default", map[string]manifest.ManifestFile{})

	var stdout bytes.Buffer
	cmd := cli.NewUpdateCommandForTesting(memFs, func() (string, error) {
		return homeDir, nil
	}, strings.NewReader(""), &stdout, projectDir)
	// Remove old/profile and add new/profile
	cmd.SetArgs([]string{"--remove-profile", "old/profile", "--add-profile", "new/profile"})

	err := cmd.Execute()
	require.NoError(t, err)

	// Verify .weftlo.yaml has default/default and new/profile (old/profile removed)
	configData, err := afero.ReadFile(memFs, filepath.Join(projectDir, ".weftlo.yaml"))
	require.NoError(t, err)

	var config projectConfigYAML
	err = yaml.Unmarshal(configData, &config)
	require.NoError(t, err)

	assert.Equal(t, []string{"default/default", "new/profile"}, config.Profiles)
}

// Test 7.4: Error when --remove-profile specifies profile not in list
func TestUpdateCommand_RemoveNonExistentProfileReturnsError(t *testing.T) {
	memFs := afero.NewMemMapFs()
	homeDir := "/home/testuser"
	projectDir := "/projects/myproject"

	_ = memFs.MkdirAll(projectDir, 0755)
	writeGlobalConfig(memFs, homeDir, "default/default")
	writeProfile(memFs, homeDir, "default", "default", "", map[string]string{})
	writeProjectConfig(memFs, projectDir, "default/default")
	writeManifest(memFs, projectDir, "default/default", map[string]manifest.ManifestFile{})

	var stdout bytes.Buffer
	cmd := cli.NewUpdateCommandForTesting(memFs, func() (string, error) {
		return homeDir, nil
	}, strings.NewReader(""), &stdout, projectDir)
	cmd.SetArgs([]string{"--remove-profile", "nonexistent/profile"})

	err := cmd.Execute()
	require.Error(t, err)

	var profileNotInListErr *cli.ProfileNotInListError
	require.True(t, errors.As(err, &profileNotInListErr))
	assert.Equal(t, "nonexistent/profile", profileNotInListErr.ProfileName)
}

// Test 7.5: Warning when --add-profile specifies already-installed profile
func TestUpdateCommand_AddAlreadyInstalledProfileWarns(t *testing.T) {
	memFs := afero.NewMemMapFs()
	homeDir := "/home/testuser"
	projectDir := "/projects/myproject"

	_ = memFs.MkdirAll(projectDir, 0755)
	writeGlobalConfig(memFs, homeDir, "default/default")
	writeProfile(memFs, homeDir, "default", "default", "", map[string]string{})
	writeProjectConfig(memFs, projectDir, "default/default")
	writeManifest(memFs, projectDir, "default/default", map[string]manifest.ManifestFile{})

	var stdout bytes.Buffer
	cmd := cli.NewUpdateCommandForTesting(memFs, func() (string, error) {
		return homeDir, nil
	}, strings.NewReader(""), &stdout, projectDir)
	// Try to add a profile that's already installed
	cmd.SetArgs([]string{"--add-profile", "default/default"})

	err := cmd.Execute()
	require.NoError(t, err) // Should not error, just warn

	// Verify warning was shown
	output := stdout.String()
	assert.Contains(t, output, "already installed")
	assert.Contains(t, output, "default/default")
}

// Test 7.6: --dry-run with profile flags prevents .weftlo.yaml modification
func TestUpdateCommand_ProfileModificationDryRun(t *testing.T) {
	memFs := afero.NewMemMapFs()
	homeDir := "/home/testuser"
	projectDir := "/projects/myproject"

	_ = memFs.MkdirAll(projectDir, 0755)
	writeGlobalConfig(memFs, homeDir, "default/default")
	writeProfile(memFs, homeDir, "default", "default", "", map[string]string{})
	writeProfile(memFs, homeDir, "new", "profile", "", map[string]string{})
	writeProjectConfig(memFs, projectDir, "default/default")
	writeManifest(memFs, projectDir, "default/default", map[string]manifest.ManifestFile{})

	// Store original config content
	originalConfig, _ := afero.ReadFile(memFs, filepath.Join(projectDir, ".weftlo.yaml"))

	var stdout bytes.Buffer
	cmd := cli.NewUpdateCommandForTesting(memFs, func() (string, error) {
		return homeDir, nil
	}, strings.NewReader(""), &stdout, projectDir)
	cmd.SetArgs([]string{"--add-profile", "new/profile", "--dry-run"})

	err := cmd.Execute()
	require.NoError(t, err)

	// Verify .weftlo.yaml was NOT modified
	newConfig, _ := afero.ReadFile(memFs, filepath.Join(projectDir, ".weftlo.yaml"))
	assert.Equal(t, originalConfig, newConfig, ".weftlo.yaml should not be modified in dry-run mode")
}

// Test 7.7: Graceful exit when profile list becomes empty after remove
func TestUpdateCommand_EmptyProfileListAfterRemove(t *testing.T) {
	memFs := afero.NewMemMapFs()
	homeDir := "/home/testuser"
	projectDir := "/projects/myproject"

	_ = memFs.MkdirAll(projectDir, 0755)
	writeGlobalConfig(memFs, homeDir, "default/default")
	writeProfile(memFs, homeDir, "default", "default", "", map[string]string{})
	writeProjectConfig(memFs, projectDir, "default/default")
	writeManifest(memFs, projectDir, "default/default", map[string]manifest.ManifestFile{})

	var stdout bytes.Buffer
	cmd := cli.NewUpdateCommandForTesting(memFs, func() (string, error) {
		return homeDir, nil
	}, strings.NewReader(""), &stdout, projectDir)
	// Remove the only profile
	cmd.SetArgs([]string{"--remove-profile", "default/default"})

	err := cmd.Execute()
	require.NoError(t, err) // Should succeed gracefully

	// Verify .weftlo.yaml has empty profiles
	configData, err := afero.ReadFile(memFs, filepath.Join(projectDir, ".weftlo.yaml"))
	require.NoError(t, err)

	var config projectConfigYAML
	err = yaml.Unmarshal(configData, &config)
	require.NoError(t, err)

	assert.Empty(t, config.Profiles)

	// Verify appropriate message was shown
	output := stdout.String()
	assert.Contains(t, output, "No profiles")
}

// Test 7.8: Multiple --add-profile flags work (repeatable)
func TestUpdateCommand_AddProfileFlagRepeatable(t *testing.T) {
	memFs := afero.NewMemMapFs()
	homeDir := "/home/testuser"
	projectDir := "/projects/myproject"

	_ = memFs.MkdirAll(projectDir, 0755)
	writeGlobalConfig(memFs, homeDir, "default/default")
	writeProfile(memFs, homeDir, "default", "default", "", map[string]string{})
	writeProfile(memFs, homeDir, "new", "one", "", map[string]string{})
	writeProfile(memFs, homeDir, "new", "two", "", map[string]string{})
	writeProjectConfig(memFs, projectDir, "default/default")
	writeManifest(memFs, projectDir, "default/default", map[string]manifest.ManifestFile{})

	var stdout bytes.Buffer
	cmd := cli.NewUpdateCommandForTesting(memFs, func() (string, error) {
		return homeDir, nil
	}, strings.NewReader(""), &stdout, projectDir)
	// Add two profiles using repeated flags
	cmd.SetArgs([]string{"--add-profile", "new/one", "--add-profile", "new/two"})

	err := cmd.Execute()
	require.NoError(t, err)

	// Verify all three profiles are in .weftlo.yaml
	configData, err := afero.ReadFile(memFs, filepath.Join(projectDir, ".weftlo.yaml"))
	require.NoError(t, err)

	var config projectConfigYAML
	err = yaml.Unmarshal(configData, &config)
	require.NoError(t, err)

	assert.Equal(t, []string{"default/default", "new/one", "new/two"}, config.Profiles)
}

// Test 7.9: Multiple --remove-profile flags work (repeatable)
func TestUpdateCommand_RemoveProfileFlagRepeatable(t *testing.T) {
	memFs := afero.NewMemMapFs()
	homeDir := "/home/testuser"
	projectDir := "/projects/myproject"

	_ = memFs.MkdirAll(projectDir, 0755)
	writeGlobalConfig(memFs, homeDir, "default/default")
	writeProfile(memFs, homeDir, "default", "default", "", map[string]string{})
	writeProfile(memFs, homeDir, "remove", "one", "", map[string]string{})
	writeProfile(memFs, homeDir, "remove", "two", "", map[string]string{})

	// Create project config with three profiles
	multiProfileConfig := "profiles:\n  - default/default\n  - remove/one\n  - remove/two\n"
	_ = afero.WriteFile(memFs, filepath.Join(projectDir, ".weftlo.yaml"), []byte(multiProfileConfig), 0644)
	writeManifest(memFs, projectDir, "default/default", map[string]manifest.ManifestFile{})

	var stdout bytes.Buffer
	cmd := cli.NewUpdateCommandForTesting(memFs, func() (string, error) {
		return homeDir, nil
	}, strings.NewReader(""), &stdout, projectDir)
	// Remove two profiles using repeated flags
	cmd.SetArgs([]string{"--remove-profile", "remove/one", "--remove-profile", "remove/two"})

	err := cmd.Execute()
	require.NoError(t, err)

	// Verify only default/default remains
	configData, err := afero.ReadFile(memFs, filepath.Join(projectDir, ".weftlo.yaml"))
	require.NoError(t, err)

	var config projectConfigYAML
	err = yaml.Unmarshal(configData, &config)
	require.NoError(t, err)

	assert.Equal(t, []string{"default/default"}, config.Profiles)
}
