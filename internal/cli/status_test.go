package cli_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/simensen/weftlo/internal/cli"
	"github.com/simensen/weftlo/internal/domain/manifest"
)

// =============================================================================
// Task Group 1: StatusCommand Foundation Tests
// =============================================================================

// Test 1.1.1: Command exists and is registered with root command
func TestStatusCommand_RegisteredWithRootCommand(t *testing.T) {
	memFs := afero.NewMemMapFs()
	rootCmd := cli.NewRootCommand(memFs, "test")

	// Find the status command
	var statusCmd *cobra.Command
	for _, cmd := range rootCmd.Commands() {
		if cmd.Use == "status" {
			statusCmd = cmd
			break
		}
	}

	require.NotNil(t, statusCmd, "expected 'status' subcommand to be registered with root command")
	assert.Equal(t, "status", statusCmd.Name())
}

// Test 1.1.2: --quiet flag suppresses non-essential output (from root persistent flag)
func TestStatusCommand_QuietFlagSuppressesOutput(t *testing.T) {
	memFs := afero.NewMemMapFs()
	homeDir := "/home/testuser"
	projectDir := "/projects/myproject"

	// Setup: create a valid installation
	_ = memFs.MkdirAll(projectDir, 0755)
	setupStatusTestGlobalConfig(t, memFs, homeDir)
	setupStatusTestProfile(t, memFs, homeDir, "default", "default", "", map[string]string{})
	setupStatusTestProjectConfig(t, memFs, projectDir, "default/default")
	setupStatusTestManifest(t, memFs, projectDir, "default/default", map[string]manifest.ManifestFile{})

	var stdout bytes.Buffer
	cmd := cli.NewStatusCommandForTesting(memFs, func() (string, error) {
		return homeDir, nil
	}, strings.NewReader(""), &stdout, projectDir)
	cmd.SetArgs([]string{"--quiet"})

	err := cmd.Execute()

	// Should succeed but suppress output
	require.NoError(t, err)
	assert.Empty(t, stdout.String(), "quiet mode should suppress all non-essential output")
}

// Test 1.1.3: --json flag produces valid JSON output
func TestStatusCommand_JsonFlagProducesValidJSON(t *testing.T) {
	memFs := afero.NewMemMapFs()
	homeDir := "/home/testuser"
	projectDir := "/projects/myproject"

	// Setup: create a valid installation
	_ = memFs.MkdirAll(projectDir, 0755)
	setupStatusTestGlobalConfig(t, memFs, homeDir)
	setupStatusTestProfile(t, memFs, homeDir, "default", "default", "", map[string]string{})
	setupStatusTestProjectConfig(t, memFs, projectDir, "default/default")
	setupStatusTestManifest(t, memFs, projectDir, "default/default", map[string]manifest.ManifestFile{})

	var stdout bytes.Buffer
	cmd := cli.NewStatusCommandForTesting(memFs, func() (string, error) {
		return homeDir, nil
	}, strings.NewReader(""), &stdout, projectDir)
	cmd.SetArgs([]string{"--json"})

	err := cmd.Execute()
	require.NoError(t, err)

	// Verify output is valid JSON
	output := stdout.String()
	var jsonOutput map[string]interface{}
	err = json.Unmarshal([]byte(output), &jsonOutput)
	require.NoError(t, err, "output should be valid JSON: %s", output)
}

// Test 1.1.4: Command returns error when .weftlo.yaml is missing
func TestStatusCommand_ErrorWhenProjectConfigMissing(t *testing.T) {
	memFs := afero.NewMemMapFs()
	homeDir := "/home/testuser"
	projectDir := "/projects/myproject"

	_ = memFs.MkdirAll(projectDir, 0755)
	// Note: NO .weftlo.yaml created

	var stdout bytes.Buffer
	cmd := cli.NewStatusCommandForTesting(memFs, func() (string, error) {
		return homeDir, nil
	}, strings.NewReader(""), &stdout, projectDir)

	err := cmd.Execute()
	require.Error(t, err)

	var notFoundErr *cli.InstallationNotFoundError
	require.True(t, errors.As(err, &notFoundErr), "error should be InstallationNotFoundError")
	assert.Equal(t, ".weftlo.yaml", notFoundErr.MissingFile)
}

// Test 1.1.5: Command returns error when manifest.json is missing
func TestStatusCommand_ErrorWhenManifestMissing(t *testing.T) {
	memFs := afero.NewMemMapFs()
	homeDir := "/home/testuser"
	projectDir := "/projects/myproject"

	_ = memFs.MkdirAll(projectDir, 0755)
	setupStatusTestProjectConfig(t, memFs, projectDir, "default/default")
	// Note: NO manifest.json created

	var stdout bytes.Buffer
	cmd := cli.NewStatusCommandForTesting(memFs, func() (string, error) {
		return homeDir, nil
	}, strings.NewReader(""), &stdout, projectDir)

	err := cmd.Execute()
	require.Error(t, err)

	var notFoundErr *cli.InstallationNotFoundError
	require.True(t, errors.As(err, &notFoundErr), "error should be InstallationNotFoundError")
	assert.Contains(t, notFoundErr.MissingFile, "manifest.json")
}

// Test 1.1.6: Error message matches InstallationNotFoundError pattern
func TestStatusCommand_InstallationNotFoundErrorMessage(t *testing.T) {
	err := &cli.InstallationNotFoundError{MissingFile: ".weftlo.yaml"}
	assert.Contains(t, err.Error(), ".weftlo.yaml")
	assert.Contains(t, err.Error(), "weftlo install")
}

// =============================================================================
// Task Group 2: Profile and Manifest Loading Tests
// =============================================================================

// Test 2.1.1: Manifest JSON is correctly parsed
func TestStatusCommand_ManifestJSONCorrectlyParsed(t *testing.T) {
	memFs := afero.NewMemMapFs()
	homeDir := "/home/testuser"
	projectDir := "/projects/myproject"

	// Setup with specific timestamp for verification
	_ = memFs.MkdirAll(projectDir, 0755)
	setupStatusTestGlobalConfig(t, memFs, homeDir)
	setupStatusTestProfile(t, memFs, homeDir, "default", "default", "", map[string]string{})
	setupStatusTestProjectConfig(t, memFs, projectDir, "default/default")

	// Create manifest with specific data
	expectedTime := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
	m := manifest.Manifest{
		Version:     "1.0.0",
		Profiles:    []string{"default/default"},
		GeneratedAt: expectedTime,
		Files: map[string]manifest.ManifestFile{
			".claude/CLAUDE.md": {
				SourceChecksum: "abc123",
				OutputChecksum: "def456",
				SourceProfile:  "default/default",
			},
		},
	}
	data, _ := json.MarshalIndent(m, "", "  ")
	_ = afero.WriteFile(memFs, filepath.Join(projectDir, ".weftlo.manifest.json"), data, 0644)

	var stdout bytes.Buffer
	cmd := cli.NewStatusCommandForTesting(memFs, func() (string, error) {
		return homeDir, nil
	}, strings.NewReader(""), &stdout, projectDir)
	cmd.SetArgs([]string{"--json"})

	err := cmd.Execute()
	require.NoError(t, err)

	// Parse the JSON output
	var result cli.StatusResult
	err = json.Unmarshal([]byte(stdout.String()), &result)
	require.NoError(t, err)

	// Verify manifest data was correctly parsed
	assert.Equal(t, []string{"default/default"}, result.Profiles)
	assert.Equal(t, expectedTime.Format(time.RFC3339), result.InstalledAt.Format(time.RFC3339))
}

// Test 2.1.2: Profile loading succeeds with valid profile
func TestStatusCommand_ProfileLoadingSucceedsWithValidProfile(t *testing.T) {
	memFs := afero.NewMemMapFs()
	homeDir := "/home/testuser"
	projectDir := "/projects/myproject"

	// Setup with valid profile
	_ = memFs.MkdirAll(projectDir, 0755)
	setupStatusTestGlobalConfig(t, memFs, homeDir)
	setupStatusTestProfile(t, memFs, homeDir, "acme", "backend", "", map[string]string{
		"CLAUDE.md": "# Test content",
	})
	setupStatusTestProjectConfig(t, memFs, projectDir, "acme/backend")
	setupStatusTestManifest(t, memFs, projectDir, "acme/backend", map[string]manifest.ManifestFile{})

	var stdout bytes.Buffer
	cmd := cli.NewStatusCommandForTesting(memFs, func() (string, error) {
		return homeDir, nil
	}, strings.NewReader(""), &stdout, projectDir)
	cmd.SetArgs([]string{"--json"})

	err := cmd.Execute()
	require.NoError(t, err)

	// Parse the JSON output
	var result cli.StatusResult
	err = json.Unmarshal([]byte(stdout.String()), &result)
	require.NoError(t, err)

	// Verify profile was loaded successfully
	assert.Equal(t, []string{"acme/backend"}, result.Profiles)
	assert.Contains(t, result.InheritanceChain, "acme/backend")
	assert.Empty(t, result.Warnings, "should have no warnings when profile loads successfully")
}

// Test 2.1.3: ProfileNotFoundError triggers warning mode (not hard error)
func TestStatusCommand_ProfileNotFoundTriggersWarning(t *testing.T) {
	memFs := afero.NewMemMapFs()
	homeDir := "/home/testuser"
	projectDir := "/projects/myproject"

	// Setup with valid installation but profile does NOT exist in config directory
	_ = memFs.MkdirAll(projectDir, 0755)
	setupStatusTestGlobalConfig(t, memFs, homeDir)
	// Note: NOT creating the "missing/profile" in the config directory
	setupStatusTestProjectConfig(t, memFs, projectDir, "missing/profile")
	setupStatusTestManifest(t, memFs, projectDir, "missing/profile", map[string]manifest.ManifestFile{})

	var stdout bytes.Buffer
	cmd := cli.NewStatusCommandForTesting(memFs, func() (string, error) {
		return homeDir, nil
	}, strings.NewReader(""), &stdout, projectDir)
	cmd.SetArgs([]string{"--json"})

	// Command should succeed (not return error)
	err := cmd.Execute()
	require.NoError(t, err, "command should succeed even with missing profile")

	// Parse the JSON output
	var result cli.StatusResult
	err = json.Unmarshal([]byte(stdout.String()), &result)
	require.NoError(t, err)

	// Verify warning was added
	require.NotEmpty(t, result.Warnings, "should have warnings when profile is missing")
	assert.Contains(t, result.Warnings[0], "missing/profile", "warning should mention the missing profile name")

	// Verify fallback to manifest profiles for inheritance chain
	assert.Contains(t, result.InheritanceChain, "missing/profile")
}

// Test 2.1.4: Inheritance chain is extracted from MergedProfile.ProfileNames()
func TestStatusCommand_InheritanceChainExtracted(t *testing.T) {
	memFs := afero.NewMemMapFs()
	homeDir := "/home/testuser"
	projectDir := "/projects/myproject"

	// Setup with profile inheritance: child -> parent
	_ = memFs.MkdirAll(projectDir, 0755)
	setupStatusTestGlobalConfig(t, memFs, homeDir)
	// Create parent profile
	setupStatusTestProfile(t, memFs, homeDir, "acme", "base", "", map[string]string{
		"base.md": "# Base content",
	})
	// Create child profile that inherits from parent
	setupStatusTestProfile(t, memFs, homeDir, "acme", "extended", "acme/base", map[string]string{
		"extended.md": "# Extended content",
	})
	setupStatusTestProjectConfig(t, memFs, projectDir, "acme/extended")
	setupStatusTestManifest(t, memFs, projectDir, "acme/extended", map[string]manifest.ManifestFile{})

	var stdout bytes.Buffer
	cmd := cli.NewStatusCommandForTesting(memFs, func() (string, error) {
		return homeDir, nil
	}, strings.NewReader(""), &stdout, projectDir)
	cmd.SetArgs([]string{"--json"})

	err := cmd.Execute()
	require.NoError(t, err)

	// Parse the JSON output
	var result cli.StatusResult
	err = json.Unmarshal([]byte(stdout.String()), &result)
	require.NoError(t, err)

	// Verify inheritance chain contains both profiles in correct order (root-to-leaf)
	require.Len(t, result.InheritanceChain, 2, "inheritance chain should have 2 profiles")
	assert.Equal(t, "acme/base", result.InheritanceChain[0], "first profile should be parent (root)")
	assert.Equal(t, "acme/extended", result.InheritanceChain[1], "second profile should be child (leaf)")
}

// Test 2.1.5: Install prefix is read from project config
func TestStatusCommand_InstallPrefixFromProjectConfig(t *testing.T) {
	memFs := afero.NewMemMapFs()
	homeDir := "/home/testuser"
	projectDir := "/projects/myproject"

	// Setup with custom install prefix
	_ = memFs.MkdirAll(projectDir, 0755)
	setupStatusTestGlobalConfig(t, memFs, homeDir)
	setupStatusTestProfile(t, memFs, homeDir, "default", "default", "", map[string]string{})

	// Create project config with custom install_prefix
	projectConfig := "profiles:\n  - default/default\ninstall_prefix: custom-prefix\n"
	_ = afero.WriteFile(memFs, filepath.Join(projectDir, ".weftlo.yaml"), []byte(projectConfig), 0644)

	setupStatusTestManifest(t, memFs, projectDir, "default/default", map[string]manifest.ManifestFile{})

	var stdout bytes.Buffer
	cmd := cli.NewStatusCommandForTesting(memFs, func() (string, error) {
		return homeDir, nil
	}, strings.NewReader(""), &stdout, projectDir)
	cmd.SetArgs([]string{"--json"})

	err := cmd.Execute()
	require.NoError(t, err)

	// Parse the JSON output
	var result cli.StatusResult
	err = json.Unmarshal([]byte(stdout.String()), &result)
	require.NoError(t, err)

	// Verify install prefix was read from project config
	assert.Equal(t, "custom-prefix", result.InstallPrefix)
}

// Test 2.1.6: Default install prefix is used when not specified
func TestStatusCommand_DefaultInstallPrefixWhenNotSpecified(t *testing.T) {
	memFs := afero.NewMemMapFs()
	homeDir := "/home/testuser"
	projectDir := "/projects/myproject"

	// Setup without specifying install_prefix - write project config directly
	_ = memFs.MkdirAll(projectDir, 0755)
	setupStatusTestGlobalConfig(t, memFs, homeDir)
	setupStatusTestProfile(t, memFs, homeDir, "default", "default", "", map[string]string{})
	// Write project config directly WITHOUT install_prefix to test default behavior
	projectConfigContent := "profiles:\n  - default/default\n"
	_ = afero.WriteFile(memFs, filepath.Join(projectDir, ".weftlo.yaml"), []byte(projectConfigContent), 0644)
	setupStatusTestManifest(t, memFs, projectDir, "default/default", map[string]manifest.ManifestFile{})

	var stdout bytes.Buffer
	cmd := cli.NewStatusCommandForTesting(memFs, func() (string, error) {
		return homeDir, nil
	}, strings.NewReader(""), &stdout, projectDir)
	cmd.SetArgs([]string{"--json"})

	err := cmd.Execute()
	require.NoError(t, err)

	// Parse the JSON output
	var result cli.StatusResult
	err = json.Unmarshal([]byte(stdout.String()), &result)
	require.NoError(t, err)

	// Verify the default install prefix (".claude") is used.
	assert.Equal(t, ".claude", result.InstallPrefix)
}

// =============================================================================
// Task Group 3: File Status Detection Tests
// =============================================================================

// Test 3.1.1: Files are categorized by status (unchanged, source_changed, user_modified, conflict, new, removed)
func TestStatusCommand_FilesCategorizedByStatus(t *testing.T) {
	memFs := afero.NewMemMapFs()
	homeDir := "/home/testuser"
	projectDir := "/projects/myproject"

	// Setup
	_ = memFs.MkdirAll(projectDir, 0755)
	setupStatusTestGlobalConfig(t, memFs, homeDir)

	// Create profile with template files
	setupStatusTestProfile(t, memFs, homeDir, "default", "default", "", map[string]string{
		"unchanged.md":      "# Unchanged content",
		"source_changed.md": "# New source content", // Different from manifest
		"user_modified.md":  "# Original content",   // Same as manifest source
		"conflict.md":       "# New source content", // Different from manifest
		"new.md":            "# New file content",   // Not in manifest
		// "removed.md" is in manifest but NOT in profile templates
	})

	setupStatusTestProjectConfig(t, memFs, projectDir, "default/default")

	// Compute checksums for the test files
	unchangedContent := "# Unchanged content"
	originalSourceChangedContent := "# Original source content"
	userModifiedContent := "# Original content"
	originalConflictContent := "# Original conflict content"

	unchangedChecksum := manifest.ComputeChecksum([]byte(unchangedContent))
	originalSourceChangedChecksum := manifest.ComputeChecksum([]byte(originalSourceChangedContent))
	userModifiedChecksum := manifest.ComputeChecksum([]byte(userModifiedContent))
	originalConflictChecksum := manifest.ComputeChecksum([]byte(originalConflictContent))
	removedChecksum := manifest.ComputeChecksum([]byte("# Removed content"))

	// Create manifest with files in various states (paths include weftlo/ prefix)
	setupStatusTestManifest(t, memFs, projectDir, "default/default", map[string]manifest.ManifestFile{
		".claude/unchanged.md": {
			SourceChecksum: unchangedChecksum,
			OutputChecksum: unchangedChecksum,
			SourceProfile:  "default/default",
		},
		".claude/source_changed.md": {
			SourceChecksum: originalSourceChangedChecksum, // Source will differ
			OutputChecksum: originalSourceChangedChecksum, // Output still matches manifest
			SourceProfile:  "default/default",
		},
		".claude/user_modified.md": {
			SourceChecksum: userModifiedChecksum,                                   // Source matches
			OutputChecksum: manifest.ComputeChecksum([]byte("# Different output")), // Output differs
			SourceProfile:  "default/default",
		},
		".claude/conflict.md": {
			SourceChecksum: originalConflictChecksum,                                   // Source differs
			OutputChecksum: manifest.ComputeChecksum([]byte("# Old conflict content")), // Output differs
			SourceProfile:  "default/default",
		},
		".claude/removed.md": {
			SourceChecksum: removedChecksum,
			OutputChecksum: removedChecksum,
			SourceProfile:  "default/default",
		},
		// "new.md" is NOT in manifest (will be detected as new)
	})

	// Create output files in weftlo/ directory
	_ = memFs.MkdirAll(filepath.Join(projectDir, ".claude"), 0755)
	_ = afero.WriteFile(memFs, filepath.Join(projectDir, ".claude", "unchanged.md"), []byte(unchangedContent), 0644)
	_ = afero.WriteFile(memFs, filepath.Join(projectDir, ".claude", "source_changed.md"), []byte(originalSourceChangedContent), 0644)
	_ = afero.WriteFile(memFs, filepath.Join(projectDir, ".claude", "user_modified.md"), []byte("# User modified content"), 0644)
	_ = afero.WriteFile(memFs, filepath.Join(projectDir, ".claude", "conflict.md"), []byte("# User modified conflict"), 0644)
	// new.md does not exist in project yet
	// removed.md is in manifest but not in profile, so we create it to see it's detected
	_ = afero.WriteFile(memFs, filepath.Join(projectDir, ".claude", "removed.md"), []byte("# Removed content"), 0644)

	var stdout bytes.Buffer
	cmd := cli.NewStatusCommandForTesting(memFs, func() (string, error) {
		return homeDir, nil
	}, strings.NewReader(""), &stdout, projectDir)
	cmd.SetArgs([]string{"--json"})

	err := cmd.Execute()
	require.NoError(t, err)

	// Parse the JSON output
	var result cli.StatusResult
	err = json.Unmarshal([]byte(stdout.String()), &result)
	require.NoError(t, err)

	// Verify files are categorized correctly (paths include weftlo/ prefix)
	assert.Contains(t, result.Files["unchanged"], ".claude/unchanged.md", "unchanged.md should be in unchanged category")
	assert.Contains(t, result.Files["source_changed"], ".claude/source_changed.md", "source_changed.md should be in source_changed category")
	assert.Contains(t, result.Files["user_modified"], ".claude/user_modified.md", "user_modified.md should be in user_modified category")
	assert.Contains(t, result.Files["conflict"], ".claude/conflict.md", "conflict.md should be in conflict category")
	assert.Contains(t, result.Files["new"], ".claude/new.md", "new.md should be in new category")
	assert.Contains(t, result.Files["removed"], ".claude/removed.md", "removed.md should be in removed category")
}

// Test 3.1.2: All file paths are collected per category
func TestStatusCommand_AllFilePathsCollectedPerCategory(t *testing.T) {
	memFs := afero.NewMemMapFs()
	homeDir := "/home/testuser"
	projectDir := "/projects/myproject"

	// Setup
	_ = memFs.MkdirAll(projectDir, 0755)
	setupStatusTestGlobalConfig(t, memFs, homeDir)

	// Create profile with multiple files that will all be unchanged
	content1 := "# File 1 content"
	content2 := "# File 2 content"
	content3 := "# File 3 content"

	setupStatusTestProfile(t, memFs, homeDir, "default", "default", "", map[string]string{
		"file1.md": content1,
		"file2.md": content2,
		"file3.md": content3,
	})

	setupStatusTestProjectConfig(t, memFs, projectDir, "default/default")

	checksum1 := manifest.ComputeChecksum([]byte(content1))
	checksum2 := manifest.ComputeChecksum([]byte(content2))
	checksum3 := manifest.ComputeChecksum([]byte(content3))

	// Create manifest with all files unchanged (paths include weftlo/ prefix)
	setupStatusTestManifest(t, memFs, projectDir, "default/default", map[string]manifest.ManifestFile{
		".claude/file1.md": {SourceChecksum: checksum1, OutputChecksum: checksum1, SourceProfile: "default/default"},
		".claude/file2.md": {SourceChecksum: checksum2, OutputChecksum: checksum2, SourceProfile: "default/default"},
		".claude/file3.md": {SourceChecksum: checksum3, OutputChecksum: checksum3, SourceProfile: "default/default"},
	})

	// Create output files in weftlo/ directory matching the source
	_ = memFs.MkdirAll(filepath.Join(projectDir, ".claude"), 0755)
	_ = afero.WriteFile(memFs, filepath.Join(projectDir, ".claude", "file1.md"), []byte(content1), 0644)
	_ = afero.WriteFile(memFs, filepath.Join(projectDir, ".claude", "file2.md"), []byte(content2), 0644)
	_ = afero.WriteFile(memFs, filepath.Join(projectDir, ".claude", "file3.md"), []byte(content3), 0644)

	var stdout bytes.Buffer
	cmd := cli.NewStatusCommandForTesting(memFs, func() (string, error) {
		return homeDir, nil
	}, strings.NewReader(""), &stdout, projectDir)
	cmd.SetArgs([]string{"--json"})

	err := cmd.Execute()
	require.NoError(t, err)

	// Parse the JSON output
	var result cli.StatusResult
	err = json.Unmarshal([]byte(stdout.String()), &result)
	require.NoError(t, err)

	// Verify all files are collected in unchanged category (paths include weftlo/ prefix)
	require.Len(t, result.Files["unchanged"], 3, "should have 3 unchanged files")
	assert.Contains(t, result.Files["unchanged"], ".claude/file1.md")
	assert.Contains(t, result.Files["unchanged"], ".claude/file2.md")
	assert.Contains(t, result.Files["unchanged"], ".claude/file3.md")
}

// Test 3.1.3: Change detection is skipped when profile is missing
func TestStatusCommand_ChangeDetectionSkippedWhenProfileMissing(t *testing.T) {
	memFs := afero.NewMemMapFs()
	homeDir := "/home/testuser"
	projectDir := "/projects/myproject"

	// Setup with valid installation but profile does NOT exist
	_ = memFs.MkdirAll(projectDir, 0755)
	setupStatusTestGlobalConfig(t, memFs, homeDir)
	// Note: NOT creating the "missing/profile" in the config directory
	setupStatusTestProjectConfig(t, memFs, projectDir, "missing/profile")

	// Create manifest with some files
	setupStatusTestManifest(t, memFs, projectDir, "missing/profile", map[string]manifest.ManifestFile{
		"some-file.md": {
			SourceChecksum: "sha256:abc123",
			OutputChecksum: "sha256:def456",
			SourceProfile:  "missing/profile",
		},
	})

	var stdout bytes.Buffer
	cmd := cli.NewStatusCommandForTesting(memFs, func() (string, error) {
		return homeDir, nil
	}, strings.NewReader(""), &stdout, projectDir)
	cmd.SetArgs([]string{"--json"})

	// Command should succeed
	err := cmd.Execute()
	require.NoError(t, err)

	// Parse the JSON output
	var result cli.StatusResult
	err = json.Unmarshal([]byte(stdout.String()), &result)
	require.NoError(t, err)

	// Verify warnings indicate profile is missing
	require.NotEmpty(t, result.Warnings)

	// Verify files map still has all categories (but empty since detection was skipped)
	_, hasUnchanged := result.Files["unchanged"]
	_, hasSourceChanged := result.Files["source_changed"]
	_, hasUserModified := result.Files["user_modified"]
	_, hasConflict := result.Files["conflict"]
	_, hasNew := result.Files["new"]
	_, hasRemoved := result.Files["removed"]

	assert.True(t, hasUnchanged, "should have unchanged category")
	assert.True(t, hasSourceChanged, "should have source_changed category")
	assert.True(t, hasUserModified, "should have user_modified category")
	assert.True(t, hasConflict, "should have conflict category")
	assert.True(t, hasNew, "should have new category")
	assert.True(t, hasRemoved, "should have removed category")

	// All should be empty since change detection was skipped
	totalFiles := len(result.Files["unchanged"]) +
		len(result.Files["source_changed"]) +
		len(result.Files["user_modified"]) +
		len(result.Files["conflict"]) +
		len(result.Files["new"]) +
		len(result.Files["removed"])
	assert.Equal(t, 0, totalFiles, "all file categories should be empty when profile is missing")
}

// Test 3.1.4: Empty categories are handled correctly
func TestStatusCommand_EmptyCategoriesHandled(t *testing.T) {
	memFs := afero.NewMemMapFs()
	homeDir := "/home/testuser"
	projectDir := "/projects/myproject"

	// Setup with profile with no templates and empty manifest
	_ = memFs.MkdirAll(projectDir, 0755)
	setupStatusTestGlobalConfig(t, memFs, homeDir)
	setupStatusTestProfile(t, memFs, homeDir, "default", "empty", "", map[string]string{})
	setupStatusTestProjectConfig(t, memFs, projectDir, "default/empty")
	setupStatusTestManifest(t, memFs, projectDir, "default/empty", map[string]manifest.ManifestFile{})

	var stdout bytes.Buffer
	cmd := cli.NewStatusCommandForTesting(memFs, func() (string, error) {
		return homeDir, nil
	}, strings.NewReader(""), &stdout, projectDir)
	cmd.SetArgs([]string{"--json"})

	err := cmd.Execute()
	require.NoError(t, err)

	// Parse the JSON output
	var result cli.StatusResult
	err = json.Unmarshal([]byte(stdout.String()), &result)
	require.NoError(t, err)

	// Verify all categories exist as empty slices, not nil
	assert.NotNil(t, result.Files["unchanged"], "unchanged should not be nil")
	assert.NotNil(t, result.Files["source_changed"], "source_changed should not be nil")
	assert.NotNil(t, result.Files["user_modified"], "user_modified should not be nil")
	assert.NotNil(t, result.Files["conflict"], "conflict should not be nil")
	assert.NotNil(t, result.Files["new"], "new should not be nil")
	assert.NotNil(t, result.Files["removed"], "removed should not be nil")

	// All should be empty
	assert.Empty(t, result.Files["unchanged"])
	assert.Empty(t, result.Files["source_changed"])
	assert.Empty(t, result.Files["user_modified"])
	assert.Empty(t, result.Files["conflict"])
	assert.Empty(t, result.Files["new"])
	assert.Empty(t, result.Files["removed"])
}

// Test 3.1.5: StatusResult struct has correct fields for output
func TestStatusCommand_StatusResultHasAllFields(t *testing.T) {
	memFs := afero.NewMemMapFs()
	homeDir := "/home/testuser"
	projectDir := "/projects/myproject"

	// Setup
	_ = memFs.MkdirAll(projectDir, 0755)
	setupStatusTestGlobalConfig(t, memFs, homeDir)
	setupStatusTestProfile(t, memFs, homeDir, "default", "default", "", map[string]string{
		"test.md": "# Test content",
	})

	projectConfig := "profiles:\n  - default/default\ninstall_prefix: my-prefix\n"
	_ = afero.WriteFile(memFs, filepath.Join(projectDir, ".weftlo.yaml"), []byte(projectConfig), 0644)

	expectedTime := time.Date(2024, 6, 15, 14, 30, 0, 0, time.UTC)
	content := "# Test content"
	checksum := manifest.ComputeChecksum([]byte(content))
	m := manifest.Manifest{
		Version:     "1.0.0",
		Profiles:    []string{"default/default"},
		GeneratedAt: expectedTime,
		Files: map[string]manifest.ManifestFile{
			".claude/test.md": {
				SourceChecksum: checksum,
				OutputChecksum: checksum,
				SourceProfile:  "default/default",
			},
		},
	}
	data, _ := json.MarshalIndent(m, "", "  ")
	_ = afero.WriteFile(memFs, filepath.Join(projectDir, ".weftlo.manifest.json"), data, 0644)

	// Create output file in weftlo/ directory
	_ = memFs.MkdirAll(filepath.Join(projectDir, ".claude"), 0755)
	_ = afero.WriteFile(memFs, filepath.Join(projectDir, ".claude", "test.md"), []byte(content), 0644)

	var stdout bytes.Buffer
	cmd := cli.NewStatusCommandForTesting(memFs, func() (string, error) {
		return homeDir, nil
	}, strings.NewReader(""), &stdout, projectDir)
	cmd.SetArgs([]string{"--json"})

	err := cmd.Execute()
	require.NoError(t, err)

	// Parse the JSON output
	var result cli.StatusResult
	err = json.Unmarshal([]byte(stdout.String()), &result)
	require.NoError(t, err)

	// Verify all required fields are present
	assert.Equal(t, []string{"default/default"}, result.Profiles, "Profiles field should be set")
	assert.Equal(t, []string{"default/default"}, result.InheritanceChain, "InheritanceChain field should be set")
	assert.Equal(t, expectedTime.Format(time.RFC3339), result.InstalledAt.Format(time.RFC3339), "InstalledAt field should be set")
	assert.Equal(t, "my-prefix", result.InstallPrefix, "InstallPrefix field should be set")
	assert.NotNil(t, result.Files, "Files field should not be nil")
	assert.NotNil(t, result.Warnings, "Warnings field should not be nil")
}

// =============================================================================
// Task Group 4: Human-Readable and JSON Output Formatting Tests
// =============================================================================

// Test 4.1.1: Human-readable output format matches spec
func TestStatusCommand_HumanReadableOutputFormatMatchesSpec(t *testing.T) {
	memFs := afero.NewMemMapFs()
	homeDir := "/home/testuser"
	projectDir := "/projects/myproject"

	// Setup
	_ = memFs.MkdirAll(projectDir, 0755)
	setupStatusTestGlobalConfig(t, memFs, homeDir)
	setupStatusTestProfile(t, memFs, homeDir, "default", "default", "", map[string]string{
		"CLAUDE.md":          "# Claude content",
		"commands/commit.md": "# Commit command",
		"commands/review.md": "# Review command",
	})

	projectConfig := "profiles:\n  - default/default\ninstall_prefix: weftlo\n"
	_ = afero.WriteFile(memFs, filepath.Join(projectDir, ".weftlo.yaml"), []byte(projectConfig), 0644)

	// Create manifest with specific timestamp
	// Manifest paths include the install prefix to match how real installation works
	expectedTime := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
	content1 := "# Claude content"
	content2 := "# Commit command"
	checksum1 := manifest.ComputeChecksum([]byte(content1))
	checksum2 := manifest.ComputeChecksum([]byte(content2))
	m := manifest.Manifest{
		Version:     "1.0.0",
		Profiles:    []string{"default/default"},
		GeneratedAt: expectedTime,
		Files: map[string]manifest.ManifestFile{
			".claude/CLAUDE.md":          {SourceChecksum: checksum1, OutputChecksum: checksum1, SourceProfile: "default/default"},
			".claude/commands/commit.md": {SourceChecksum: checksum2, OutputChecksum: checksum2, SourceProfile: "default/default"},
		},
	}
	data, _ := json.MarshalIndent(m, "", "  ")
	_ = afero.WriteFile(memFs, filepath.Join(projectDir, ".weftlo.manifest.json"), data, 0644)

	// Create output files at the install prefix location
	_ = memFs.MkdirAll(filepath.Join(projectDir, ".claude/commands"), 0755)
	_ = afero.WriteFile(memFs, filepath.Join(projectDir, ".claude/CLAUDE.md"), []byte(content1), 0644)
	_ = afero.WriteFile(memFs, filepath.Join(projectDir, ".claude/commands/commit.md"), []byte(content2), 0644)

	var stdout bytes.Buffer
	cmd := cli.NewStatusCommandForTesting(memFs, func() (string, error) {
		return homeDir, nil
	}, strings.NewReader(""), &stdout, projectDir)
	// No --json flag = human-readable output

	err := cmd.Execute()
	require.NoError(t, err)

	output := stdout.String()

	// Verify human-readable format contains expected elements
	assert.Contains(t, output, "Installation Status", "should have header")
	assert.Contains(t, output, "Profile:", "should show profile section")
	assert.Contains(t, output, "default/default", "should show profile name")
	assert.Contains(t, output, "Installed:", "should show installed timestamp")
	assert.Contains(t, output, "2024-01-15T10:30:00Z", "should show RFC3339 timestamp")
	assert.Contains(t, output, "Install prefix:", "should show install prefix")
	assert.Contains(t, output, "weftlo", "should show install prefix value")
	assert.Contains(t, output, "Files:", "should have files section")
	assert.Contains(t, output, "Unchanged", "should show unchanged category")
}

// Test 4.1.2: JSON output is valid and matches spec structure
func TestStatusCommand_JSONOutputMatchesSpecStructure(t *testing.T) {
	memFs := afero.NewMemMapFs()
	homeDir := "/home/testuser"
	projectDir := "/projects/myproject"

	// Setup
	_ = memFs.MkdirAll(projectDir, 0755)
	setupStatusTestGlobalConfig(t, memFs, homeDir)
	setupStatusTestProfile(t, memFs, homeDir, "default", "default", "", map[string]string{
		"test.md": "# Test content",
	})
	setupStatusTestProjectConfig(t, memFs, projectDir, "default/default")

	expectedTime := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
	content := "# Test content"
	checksum := manifest.ComputeChecksum([]byte(content))
	m := manifest.Manifest{
		Version:     "1.0.0",
		Profiles:    []string{"default/default"},
		GeneratedAt: expectedTime,
		Files: map[string]manifest.ManifestFile{
			".claude/test.md": {SourceChecksum: checksum, OutputChecksum: checksum, SourceProfile: "default/default"},
		},
	}
	data, _ := json.MarshalIndent(m, "", "  ")
	_ = afero.WriteFile(memFs, filepath.Join(projectDir, ".weftlo.manifest.json"), data, 0644)

	_ = memFs.MkdirAll(filepath.Join(projectDir, ".claude"), 0755)
	_ = afero.WriteFile(memFs, filepath.Join(projectDir, ".claude", "test.md"), []byte(content), 0644)

	var stdout bytes.Buffer
	cmd := cli.NewStatusCommandForTesting(memFs, func() (string, error) {
		return homeDir, nil
	}, strings.NewReader(""), &stdout, projectDir)
	cmd.SetArgs([]string{"--json"})

	err := cmd.Execute()
	require.NoError(t, err)

	// Parse JSON and verify structure
	var jsonOutput map[string]interface{}
	err = json.Unmarshal([]byte(stdout.String()), &jsonOutput)
	require.NoError(t, err)

	// Verify all required fields are present as per spec (lines 117-133)
	assert.Contains(t, jsonOutput, "profiles", "should have profiles field")
	assert.Contains(t, jsonOutput, "inheritance_chain", "should have inheritance_chain field")
	assert.Contains(t, jsonOutput, "installed_at", "should have installed_at field")
	assert.Contains(t, jsonOutput, "install_prefix", "should have install_prefix field")
	assert.Contains(t, jsonOutput, "files", "should have files field")
	assert.Contains(t, jsonOutput, "warnings", "should have warnings field")

	// Verify files object has all required keys
	files, ok := jsonOutput["files"].(map[string]interface{})
	require.True(t, ok, "files should be an object")
	assert.Contains(t, files, "unchanged", "files should have unchanged key")
	assert.Contains(t, files, "source_changed", "files should have source_changed key")
	assert.Contains(t, files, "user_modified", "files should have user_modified key")
	assert.Contains(t, files, "conflict", "files should have conflict key")
	assert.Contains(t, files, "new", "files should have new key")
	assert.Contains(t, files, "removed", "files should have removed key")
}

// Test 4.1.3: Quiet mode suppresses non-essential output
func TestStatusCommand_QuietModeSuppressesAllOutput(t *testing.T) {
	memFs := afero.NewMemMapFs()
	homeDir := "/home/testuser"
	projectDir := "/projects/myproject"

	// Setup with files to ensure there would be output
	_ = memFs.MkdirAll(projectDir, 0755)
	setupStatusTestGlobalConfig(t, memFs, homeDir)
	setupStatusTestProfile(t, memFs, homeDir, "default", "default", "", map[string]string{
		"test.md": "# Test content",
	})
	setupStatusTestProjectConfig(t, memFs, projectDir, "default/default")

	content := "# Test content"
	checksum := manifest.ComputeChecksum([]byte(content))
	m := manifest.Manifest{
		Version:     "1.0.0",
		Profiles:    []string{"default/default"},
		GeneratedAt: time.Now().UTC(),
		Files: map[string]manifest.ManifestFile{
			".claude/test.md": {SourceChecksum: checksum, OutputChecksum: checksum, SourceProfile: "default/default"},
		},
	}
	data, _ := json.MarshalIndent(m, "", "  ")
	_ = afero.WriteFile(memFs, filepath.Join(projectDir, ".weftlo.manifest.json"), data, 0644)
	_ = memFs.MkdirAll(filepath.Join(projectDir, ".claude"), 0755)
	_ = afero.WriteFile(memFs, filepath.Join(projectDir, ".claude", "test.md"), []byte(content), 0644)

	var stdout bytes.Buffer
	cmd := cli.NewStatusCommandForTesting(memFs, func() (string, error) {
		return homeDir, nil
	}, strings.NewReader(""), &stdout, projectDir)
	cmd.SetArgs([]string{"--quiet"})

	err := cmd.Execute()
	require.NoError(t, err)

	// Quiet mode should produce no output
	assert.Empty(t, stdout.String(), "quiet mode should suppress all output")
}

// Test 4.1.4: Warnings appear in both output formats
func TestStatusCommand_WarningsAppearInBothFormats(t *testing.T) {
	memFs := afero.NewMemMapFs()
	homeDir := "/home/testuser"
	projectDir := "/projects/myproject"

	// Setup with missing profile to trigger warning
	_ = memFs.MkdirAll(projectDir, 0755)
	setupStatusTestGlobalConfig(t, memFs, homeDir)
	// Note: NOT creating the "missing/profile" profile
	setupStatusTestProjectConfig(t, memFs, projectDir, "missing/profile")
	setupStatusTestManifest(t, memFs, projectDir, "missing/profile", map[string]manifest.ManifestFile{})

	// Test JSON output contains warnings
	t.Run("JSON output contains warnings", func(t *testing.T) {
		var stdout bytes.Buffer
		cmd := cli.NewStatusCommandForTesting(memFs, func() (string, error) {
			return homeDir, nil
		}, strings.NewReader(""), &stdout, projectDir)
		cmd.SetArgs([]string{"--json"})

		err := cmd.Execute()
		require.NoError(t, err)

		var result cli.StatusResult
		err = json.Unmarshal([]byte(stdout.String()), &result)
		require.NoError(t, err)

		require.NotEmpty(t, result.Warnings, "JSON output should contain warnings")
		assert.Contains(t, result.Warnings[0], "missing/profile", "warning should mention missing profile")
	})

	// Test human-readable output contains warnings
	t.Run("Human-readable output contains warnings", func(t *testing.T) {
		var stdout bytes.Buffer
		cmd := cli.NewStatusCommandForTesting(memFs, func() (string, error) {
			return homeDir, nil
		}, strings.NewReader(""), &stdout, projectDir)
		// No --json flag

		err := cmd.Execute()
		require.NoError(t, err)

		output := stdout.String()
		assert.Contains(t, output, "Warnings:", "human-readable output should have Warnings section")
		assert.Contains(t, output, "missing/profile", "warning should mention missing profile")
	})
}

// Test 4.1.5: Empty file categories are handled gracefully in both formats
func TestStatusCommand_EmptyFileCategoriesHandledGracefully(t *testing.T) {
	memFs := afero.NewMemMapFs()
	homeDir := "/home/testuser"
	projectDir := "/projects/myproject"

	// Setup with empty manifest (no files)
	_ = memFs.MkdirAll(projectDir, 0755)
	setupStatusTestGlobalConfig(t, memFs, homeDir)
	setupStatusTestProfile(t, memFs, homeDir, "default", "empty", "", map[string]string{})
	setupStatusTestProjectConfig(t, memFs, projectDir, "default/empty")
	setupStatusTestManifest(t, memFs, projectDir, "default/empty", map[string]manifest.ManifestFile{})

	// Test JSON output
	t.Run("JSON output handles empty categories", func(t *testing.T) {
		var stdout bytes.Buffer
		cmd := cli.NewStatusCommandForTesting(memFs, func() (string, error) {
			return homeDir, nil
		}, strings.NewReader(""), &stdout, projectDir)
		cmd.SetArgs([]string{"--json"})

		err := cmd.Execute()
		require.NoError(t, err)

		var result cli.StatusResult
		err = json.Unmarshal([]byte(stdout.String()), &result)
		require.NoError(t, err)

		// All categories should be empty arrays, not nil
		assert.NotNil(t, result.Files["unchanged"])
		assert.NotNil(t, result.Files["source_changed"])
		assert.NotNil(t, result.Files["user_modified"])
		assert.NotNil(t, result.Files["conflict"])
		assert.NotNil(t, result.Files["new"])
		assert.NotNil(t, result.Files["removed"])
	})

	// Test human-readable output
	t.Run("Human-readable output handles empty categories", func(t *testing.T) {
		var stdout bytes.Buffer
		cmd := cli.NewStatusCommandForTesting(memFs, func() (string, error) {
			return homeDir, nil
		}, strings.NewReader(""), &stdout, projectDir)
		// No --json flag

		err := cmd.Execute()
		require.NoError(t, err)

		output := stdout.String()
		// Human-readable output should not crash with empty categories
		assert.Contains(t, output, "Installation Status", "should have header even with no files")
		assert.Contains(t, output, "Files:", "should have Files section even if empty")
	})
}

// Test 4.1.6: Human-readable output shows profile inheritance chain
func TestStatusCommand_HumanReadableShowsInheritanceChain(t *testing.T) {
	memFs := afero.NewMemMapFs()
	homeDir := "/home/testuser"
	projectDir := "/projects/myproject"

	// Setup with profile inheritance
	_ = memFs.MkdirAll(projectDir, 0755)
	setupStatusTestGlobalConfig(t, memFs, homeDir)
	// Create parent profile
	setupStatusTestProfile(t, memFs, homeDir, "acme", "base", "", map[string]string{})
	// Create child profile that inherits from parent
	setupStatusTestProfile(t, memFs, homeDir, "acme", "extended", "acme/base", map[string]string{})
	setupStatusTestProjectConfig(t, memFs, projectDir, "acme/extended")
	setupStatusTestManifest(t, memFs, projectDir, "acme/extended", map[string]manifest.ManifestFile{})

	var stdout bytes.Buffer
	cmd := cli.NewStatusCommandForTesting(memFs, func() (string, error) {
		return homeDir, nil
	}, strings.NewReader(""), &stdout, projectDir)
	// No --json flag

	err := cmd.Execute()
	require.NoError(t, err)

	output := stdout.String()
	// Should show inheritance chain
	assert.Contains(t, output, "acme/extended", "should show profile name")
	assert.Contains(t, output, "Inherits from:", "should show inheritance label")
	assert.Contains(t, output, "acme/base", "should show parent profile in inheritance chain")
}

// =============================================================================
// Task Group 5: Strategic Tests for Gap Analysis
// =============================================================================

// Test 5.3.1: End-to-end test with multiple profiles in installation
func TestStatusCommand_MultipleProfilesInstallation(t *testing.T) {
	memFs := afero.NewMemMapFs()
	homeDir := "/home/testuser"
	projectDir := "/projects/myproject"

	// Setup with multiple profiles
	_ = memFs.MkdirAll(projectDir, 0755)
	setupStatusTestGlobalConfig(t, memFs, homeDir)

	// Create two independent profiles
	setupStatusTestProfile(t, memFs, homeDir, "acme", "frontend", "", map[string]string{
		"frontend.md": "# Frontend config",
	})
	setupStatusTestProfile(t, memFs, homeDir, "acme", "backend", "", map[string]string{
		"backend.md": "# Backend config",
	})

	// Create project config with multiple profiles
	projectConfig := "profiles:\n  - acme/frontend\n  - acme/backend\ninstall_prefix: weftlo\n"
	_ = afero.WriteFile(memFs, filepath.Join(projectDir, ".weftlo.yaml"), []byte(projectConfig), 0644)

	// Create manifest with multiple profiles
	// Manifest paths include the install prefix to match how real installation works
	frontendContent := "# Frontend config"
	backendContent := "# Backend config"
	frontendChecksum := manifest.ComputeChecksum([]byte(frontendContent))
	backendChecksum := manifest.ComputeChecksum([]byte(backendContent))

	m := manifest.Manifest{
		Version:     "1.0.0",
		Profiles:    []string{"acme/frontend", "acme/backend"},
		GeneratedAt: time.Date(2024, 3, 20, 14, 0, 0, 0, time.UTC),
		Files: map[string]manifest.ManifestFile{
			".claude/frontend.md": {SourceChecksum: frontendChecksum, OutputChecksum: frontendChecksum, SourceProfile: "acme/frontend"},
			".claude/backend.md":  {SourceChecksum: backendChecksum, OutputChecksum: backendChecksum, SourceProfile: "acme/backend"},
		},
	}
	data, _ := json.MarshalIndent(m, "", "  ")
	_ = afero.WriteFile(memFs, filepath.Join(projectDir, ".weftlo.manifest.json"), data, 0644)

	// Create output files at the install prefix location
	_ = memFs.MkdirAll(filepath.Join(projectDir, ".claude"), 0755)
	_ = afero.WriteFile(memFs, filepath.Join(projectDir, ".claude/frontend.md"), []byte(frontendContent), 0644)
	_ = afero.WriteFile(memFs, filepath.Join(projectDir, ".claude/backend.md"), []byte(backendContent), 0644)

	var stdout bytes.Buffer
	cmd := cli.NewStatusCommandForTesting(memFs, func() (string, error) {
		return homeDir, nil
	}, strings.NewReader(""), &stdout, projectDir)
	cmd.SetArgs([]string{"--json"})

	err := cmd.Execute()
	require.NoError(t, err)

	// Parse the JSON output
	var result cli.StatusResult
	err = json.Unmarshal([]byte(stdout.String()), &result)
	require.NoError(t, err)

	// Verify multiple profiles are reported
	assert.Len(t, result.Profiles, 2, "should have 2 profiles")
	assert.Contains(t, result.Profiles, "acme/frontend")
	assert.Contains(t, result.Profiles, "acme/backend")

	// Verify files from both profiles are tracked (paths include install prefix)
	assert.Len(t, result.Files["unchanged"], 2, "should have 2 unchanged files")
	assert.Contains(t, result.Files["unchanged"], ".claude/frontend.md")
	assert.Contains(t, result.Files["unchanged"], ".claude/backend.md")

	// No warnings expected
	assert.Empty(t, result.Warnings, "should have no warnings")
}

// Test 5.3.2: JSON output timestamp is in RFC3339 format
func TestStatusCommand_JSONTimestampRFC3339Format(t *testing.T) {
	memFs := afero.NewMemMapFs()
	homeDir := "/home/testuser"
	projectDir := "/projects/myproject"

	// Setup
	_ = memFs.MkdirAll(projectDir, 0755)
	setupStatusTestGlobalConfig(t, memFs, homeDir)
	setupStatusTestProfile(t, memFs, homeDir, "default", "default", "", map[string]string{})
	setupStatusTestProjectConfig(t, memFs, projectDir, "default/default")

	// Use a specific timestamp with timezone
	expectedTime := time.Date(2024, 7, 4, 12, 30, 45, 0, time.UTC)
	m := manifest.Manifest{
		Version:     "1.0.0",
		Profiles:    []string{"default/default"},
		GeneratedAt: expectedTime,
		Files:       map[string]manifest.ManifestFile{},
	}
	data, _ := json.MarshalIndent(m, "", "  ")
	_ = afero.WriteFile(memFs, filepath.Join(projectDir, ".weftlo.manifest.json"), data, 0644)

	var stdout bytes.Buffer
	cmd := cli.NewStatusCommandForTesting(memFs, func() (string, error) {
		return homeDir, nil
	}, strings.NewReader(""), &stdout, projectDir)
	cmd.SetArgs([]string{"--json"})

	err := cmd.Execute()
	require.NoError(t, err)

	// Parse raw JSON to verify timestamp format
	var rawOutput map[string]interface{}
	err = json.Unmarshal([]byte(stdout.String()), &rawOutput)
	require.NoError(t, err)

	// Get the installed_at field as a string
	installedAtStr, ok := rawOutput["installed_at"].(string)
	require.True(t, ok, "installed_at should be a string")

	// Verify it's in RFC3339 format by parsing it
	parsedTime, err := time.Parse(time.RFC3339, installedAtStr)
	require.NoError(t, err, "installed_at should be parseable as RFC3339")

	// Verify the parsed time matches expected
	assert.True(t, expectedTime.Equal(parsedTime), "parsed time should match expected time")
}

// Test 5.3.3: All files in conflict edge case
func TestStatusCommand_AllFilesInConflict(t *testing.T) {
	memFs := afero.NewMemMapFs()
	homeDir := "/home/testuser"
	projectDir := "/projects/myproject"

	// Setup
	_ = memFs.MkdirAll(projectDir, 0755)
	setupStatusTestGlobalConfig(t, memFs, homeDir)

	// Create profile with current source content
	setupStatusTestProfile(t, memFs, homeDir, "default", "default", "", map[string]string{
		"file1.md": "# New source 1",
		"file2.md": "# New source 2",
		"file3.md": "# New source 3",
	})

	setupStatusTestProjectConfig(t, memFs, projectDir, "default/default")

	// Create manifest with OLD checksums (both source and output differ from current)
	oldChecksum1 := manifest.ComputeChecksum([]byte("# Old source 1"))
	oldChecksum2 := manifest.ComputeChecksum([]byte("# Old source 2"))
	oldChecksum3 := manifest.ComputeChecksum([]byte("# Old source 3"))

	setupStatusTestManifest(t, memFs, projectDir, "default/default", map[string]manifest.ManifestFile{
		".claude/file1.md": {SourceChecksum: oldChecksum1, OutputChecksum: oldChecksum1, SourceProfile: "default/default"},
		".claude/file2.md": {SourceChecksum: oldChecksum2, OutputChecksum: oldChecksum2, SourceProfile: "default/default"},
		".claude/file3.md": {SourceChecksum: oldChecksum3, OutputChecksum: oldChecksum3, SourceProfile: "default/default"},
	})

	// Create output files in weftlo/ with USER modifications (different from both old and new source)
	_ = memFs.MkdirAll(filepath.Join(projectDir, ".claude"), 0755)
	_ = afero.WriteFile(memFs, filepath.Join(projectDir, ".claude", "file1.md"), []byte("# User edit 1"), 0644)
	_ = afero.WriteFile(memFs, filepath.Join(projectDir, ".claude", "file2.md"), []byte("# User edit 2"), 0644)
	_ = afero.WriteFile(memFs, filepath.Join(projectDir, ".claude", "file3.md"), []byte("# User edit 3"), 0644)

	var stdout bytes.Buffer
	cmd := cli.NewStatusCommandForTesting(memFs, func() (string, error) {
		return homeDir, nil
	}, strings.NewReader(""), &stdout, projectDir)
	cmd.SetArgs([]string{"--json"})

	err := cmd.Execute()
	require.NoError(t, err)

	// Parse the JSON output
	var result cli.StatusResult
	err = json.Unmarshal([]byte(stdout.String()), &result)
	require.NoError(t, err)

	// All 3 files should be in conflict (source changed AND output modified)
	assert.Len(t, result.Files["conflict"], 3, "all 3 files should be in conflict")
	assert.Contains(t, result.Files["conflict"], ".claude/file1.md")
	assert.Contains(t, result.Files["conflict"], ".claude/file2.md")
	assert.Contains(t, result.Files["conflict"], ".claude/file3.md")

	// No files in other categories
	assert.Empty(t, result.Files["unchanged"], "should have no unchanged files")
	assert.Empty(t, result.Files["source_changed"], "should have no source_changed files")
	assert.Empty(t, result.Files["user_modified"], "should have no user_modified files")
	assert.Empty(t, result.Files["new"], "should have no new files")
	assert.Empty(t, result.Files["removed"], "should have no removed files")
}

// Test 5.3.4: Human-readable output shows accurate file counts
func TestStatusCommand_HumanReadableFileCountsAccurate(t *testing.T) {
	memFs := afero.NewMemMapFs()
	homeDir := "/home/testuser"
	projectDir := "/projects/myproject"

	// Setup
	_ = memFs.MkdirAll(projectDir, 0755)
	setupStatusTestGlobalConfig(t, memFs, homeDir)

	// Create profile with 5 files
	content1 := "# File 1"
	content2 := "# File 2"
	content3 := "# File 3"
	content4 := "# File 4"
	content5 := "# File 5"

	setupStatusTestProfile(t, memFs, homeDir, "default", "default", "", map[string]string{
		"file1.md": content1,
		"file2.md": content2,
		"file3.md": content3,
		"file4.md": content4,
		"file5.md": content5,
	})

	setupStatusTestProjectConfig(t, memFs, projectDir, "default/default")

	// Create manifest with checksums for 5 files (paths include weftlo/ prefix)
	checksum1 := manifest.ComputeChecksum([]byte(content1))
	checksum2 := manifest.ComputeChecksum([]byte(content2))
	checksum3 := manifest.ComputeChecksum([]byte(content3))
	checksum4 := manifest.ComputeChecksum([]byte(content4))
	checksum5 := manifest.ComputeChecksum([]byte(content5))

	setupStatusTestManifest(t, memFs, projectDir, "default/default", map[string]manifest.ManifestFile{
		".claude/file1.md": {SourceChecksum: checksum1, OutputChecksum: checksum1, SourceProfile: "default/default"},
		".claude/file2.md": {SourceChecksum: checksum2, OutputChecksum: checksum2, SourceProfile: "default/default"},
		".claude/file3.md": {SourceChecksum: checksum3, OutputChecksum: checksum3, SourceProfile: "default/default"},
		".claude/file4.md": {SourceChecksum: checksum4, OutputChecksum: checksum4, SourceProfile: "default/default"},
		".claude/file5.md": {SourceChecksum: checksum5, OutputChecksum: checksum5, SourceProfile: "default/default"},
	})

	// Create all 5 output files unchanged in weftlo/ directory
	_ = memFs.MkdirAll(filepath.Join(projectDir, ".claude"), 0755)
	_ = afero.WriteFile(memFs, filepath.Join(projectDir, ".claude", "file1.md"), []byte(content1), 0644)
	_ = afero.WriteFile(memFs, filepath.Join(projectDir, ".claude", "file2.md"), []byte(content2), 0644)
	_ = afero.WriteFile(memFs, filepath.Join(projectDir, ".claude", "file3.md"), []byte(content3), 0644)
	_ = afero.WriteFile(memFs, filepath.Join(projectDir, ".claude", "file4.md"), []byte(content4), 0644)
	_ = afero.WriteFile(memFs, filepath.Join(projectDir, ".claude", "file5.md"), []byte(content5), 0644)

	var stdout bytes.Buffer
	cmd := cli.NewStatusCommandForTesting(memFs, func() (string, error) {
		return homeDir, nil
	}, strings.NewReader(""), &stdout, projectDir)
	// No --json flag for human-readable output

	err := cmd.Execute()
	require.NoError(t, err)

	output := stdout.String()

	// Verify the count shows 5 unchanged files (paths include weftlo/ prefix)
	assert.Contains(t, output, "Unchanged (5):", "should show count of 5 unchanged files")
	assert.Contains(t, output, ".claude/file1.md", "should list file1.md")
	assert.Contains(t, output, ".claude/file2.md", "should list file2.md")
	assert.Contains(t, output, ".claude/file3.md", "should list file3.md")
	assert.Contains(t, output, ".claude/file4.md", "should list file4.md")
	assert.Contains(t, output, ".claude/file5.md", "should list file5.md")
}

// Test 5.3.5: Deep inheritance chain (3 levels)
func TestStatusCommand_DeepInheritanceChain(t *testing.T) {
	memFs := afero.NewMemMapFs()
	homeDir := "/home/testuser"
	projectDir := "/projects/myproject"

	// Setup with 3-level inheritance: grandparent -> parent -> child
	_ = memFs.MkdirAll(projectDir, 0755)
	setupStatusTestGlobalConfig(t, memFs, homeDir)

	// Create grandparent profile
	setupStatusTestProfile(t, memFs, homeDir, "acme", "grandparent", "", map[string]string{
		"grandparent.md": "# Grandparent content",
	})
	// Create parent profile that inherits from grandparent
	setupStatusTestProfile(t, memFs, homeDir, "acme", "parent", "acme/grandparent", map[string]string{
		"parent.md": "# Parent content",
	})
	// Create child profile that inherits from parent
	setupStatusTestProfile(t, memFs, homeDir, "acme", "child", "acme/parent", map[string]string{
		"child.md": "# Child content",
	})

	setupStatusTestProjectConfig(t, memFs, projectDir, "acme/child")
	setupStatusTestManifest(t, memFs, projectDir, "acme/child", map[string]manifest.ManifestFile{})

	var stdout bytes.Buffer
	cmd := cli.NewStatusCommandForTesting(memFs, func() (string, error) {
		return homeDir, nil
	}, strings.NewReader(""), &stdout, projectDir)
	cmd.SetArgs([]string{"--json"})

	err := cmd.Execute()
	require.NoError(t, err)

	// Parse the JSON output
	var result cli.StatusResult
	err = json.Unmarshal([]byte(stdout.String()), &result)
	require.NoError(t, err)

	// Verify inheritance chain contains all 3 profiles in correct order (root-to-leaf)
	require.Len(t, result.InheritanceChain, 3, "inheritance chain should have 3 profiles")
	assert.Equal(t, "acme/grandparent", result.InheritanceChain[0], "first profile should be grandparent (root)")
	assert.Equal(t, "acme/parent", result.InheritanceChain[1], "second profile should be parent")
	assert.Equal(t, "acme/child", result.InheritanceChain[2], "third profile should be child (leaf)")
}

// =============================================================================
// StatusCommand Test Helper Functions
// =============================================================================

// setupStatusTestGlobalConfig creates ~/.weftlo/config.yaml
func setupStatusTestGlobalConfig(t *testing.T, fs afero.Fs, homeDir string) {
	t.Helper()
	configDir := filepath.Join(homeDir, ".weftlo")
	_ = fs.MkdirAll(configDir, 0755)
	content := "default_profile: default/default\n"
	_ = afero.WriteFile(fs, filepath.Join(configDir, "config.yaml"), []byte(content), 0644)
}

// setupStatusTestProfile creates a profile with optional templates
// Files are installed to weftlo/ by default per the content routing architecture
func setupStatusTestProfile(t *testing.T, fs afero.Fs, homeDir string, vendor string, name string, inheritsFrom string, templates map[string]string) {
	t.Helper()
	profileDir := filepath.Join(homeDir, ".weftlo", "profiles", vendor, name)
	contentDir := filepath.Join(profileDir, "content")
	_ = fs.MkdirAll(contentDir, 0755)

	// Write profile.yaml with content configuration
	profileYaml := "name: " + vendor + "/" + name + "\n"
	if inheritsFrom != "" {
		profileYaml += "inherits_from: " + inheritsFrom + "\n"
	}
	// Content root defaults to "content", default_target defaults to "weftlo"
	profileYaml += "content:\n  root: content\n"
	_ = afero.WriteFile(fs, filepath.Join(profileDir, "profile.yaml"), []byte(profileYaml), 0644)

	// Write templates in content directory
	for relPath, content := range templates {
		templatePath := filepath.Join(contentDir, relPath)
		_ = fs.MkdirAll(filepath.Dir(templatePath), 0755)
		_ = afero.WriteFile(fs, templatePath, []byte(content), 0644)
	}
}

// setupStatusTestProjectConfig creates .weftlo.yaml in project directory
func setupStatusTestProjectConfig(t *testing.T, fs afero.Fs, projectDir string, profile string) {
	t.Helper()
	content := "profiles:\n  - " + profile + "\n"
	_ = afero.WriteFile(fs, filepath.Join(projectDir, ".weftlo.yaml"), []byte(content), 0644)
}

// setupStatusTestManifest creates .weftlo.manifest.json in project directory
func setupStatusTestManifest(t *testing.T, fs afero.Fs, projectDir string, profile string, files map[string]manifest.ManifestFile) {
	t.Helper()
	m := manifest.Manifest{
		Version:     "1.0.0",
		Profiles:    []string{profile},
		GeneratedAt: time.Now().UTC(),
		Files:       files,
	}
	data, _ := json.MarshalIndent(m, "", "  ")
	_ = afero.WriteFile(fs, filepath.Join(projectDir, ".weftlo.manifest.json"), data, 0644)
}
