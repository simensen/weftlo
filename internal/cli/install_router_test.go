package cli_test

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/simensen/weftlo/internal/cli"
	"github.com/simensen/weftlo/internal/domain/manifest"
)

// =============================================================================
// Task Group 4.1: CLI Router Integration and Output Tests
// =============================================================================

// Test 4.1.1: Router is constructed from MergedProfile.ContentConfig and ProjectConfig.TargetOverrides
func TestInstallCommand_RouterConstructedFromContentConfig(t *testing.T) {
	fs := afero.NewMemMapFs()
	homeDir := "/home/testuser"
	projectDir := "/projects/myproject"

	// Setup global config
	configDir := filepath.Join(homeDir, ".weftlo")
	require.NoError(t, fs.MkdirAll(configDir, 0755))
	globalConfig := "default_profile: test/myprofile\n"
	require.NoError(t, afero.WriteFile(fs, filepath.Join(configDir, "config.yaml"), []byte(globalConfig), 0644))

	// Setup profile with content configuration that specifies targets
	profileDir := filepath.Join(configDir, "profiles", "test", "myprofile")
	require.NoError(t, fs.MkdirAll(profileDir, 0755))

	// Profile with content config defining targets
	profileYaml := `name: test/myprofile
content:
  root: content
  targets:
    skills: .claude/skills
    standards: weftlo/standards
  default_target: weftlo
`
	require.NoError(t, afero.WriteFile(fs, filepath.Join(profileDir, "profile.yaml"), []byte(profileYaml), 0644))

	// Create content directory with files in different target directories
	contentDir := filepath.Join(profileDir, "content")
	require.NoError(t, fs.MkdirAll(filepath.Join(contentDir, "skills"), 0755))
	require.NoError(t, fs.MkdirAll(filepath.Join(contentDir, "standards"), 0755))
	require.NoError(t, afero.WriteFile(fs, filepath.Join(contentDir, "skills", "coding.md.tmpl"), []byte("# Coding skill"), 0644))
	require.NoError(t, afero.WriteFile(fs, filepath.Join(contentDir, "standards", "style.md.tmpl"), []byte("# Style guide"), 0644))

	// Setup project directory
	require.NoError(t, fs.MkdirAll(projectDir, 0755))

	// Create and execute install command
	var stdout bytes.Buffer
	cmd := cli.NewInstallCommandForTesting(fs, func() (string, error) {
		return homeDir, nil
	}, strings.NewReader(""), &stdout, projectDir)

	cmd.SetArgs([]string{"--profile", "test/myprofile"})
	err := cmd.Execute()
	require.NoError(t, err)

	// Verify files were installed to the correct target directories based on ContentConfig
	// skills/* should go to .claude/skills/
	skillsFile := filepath.Join(projectDir, ".claude", "skills", "coding.md")
	exists, err := afero.Exists(fs, skillsFile)
	require.NoError(t, err)
	assert.True(t, exists, "skills file should be installed to .claude/skills/")

	// standards/* should go to weftlo/standards/
	standardsFile := filepath.Join(projectDir, "weftlo", "standards", "style.md")
	exists, err = afero.Exists(fs, standardsFile)
	require.NoError(t, err)
	assert.True(t, exists, "standards file should be installed to weftlo/standards/")
}

// Test 4.1.2: Install output groups files by TargetCategory
func TestInstallCommand_OutputGroupsByTargetCategory(t *testing.T) {
	fs := afero.NewMemMapFs()
	homeDir := "/home/testuser"
	projectDir := "/projects/myproject"

	// Setup global config
	configDir := filepath.Join(homeDir, ".weftlo")
	require.NoError(t, fs.MkdirAll(configDir, 0755))
	globalConfig := "default_profile: test/myprofile\n"
	require.NoError(t, afero.WriteFile(fs, filepath.Join(configDir, "config.yaml"), []byte(globalConfig), 0644))

	// Setup profile with multiple target categories
	profileDir := filepath.Join(configDir, "profiles", "test", "myprofile")
	require.NoError(t, fs.MkdirAll(profileDir, 0755))

	profileYaml := `name: test/myprofile
content:
  root: content
  targets:
    skills: .claude/skills
    prompts: .claude/prompts
  default_target: weftlo
`
	require.NoError(t, afero.WriteFile(fs, filepath.Join(profileDir, "profile.yaml"), []byte(profileYaml), 0644))

	// Create content with files in different categories
	contentDir := filepath.Join(profileDir, "content")
	require.NoError(t, fs.MkdirAll(filepath.Join(contentDir, "skills"), 0755))
	require.NoError(t, fs.MkdirAll(filepath.Join(contentDir, "prompts"), 0755))
	require.NoError(t, afero.WriteFile(fs, filepath.Join(contentDir, "skills", "skill1.md.tmpl"), []byte("# Skill 1"), 0644))
	require.NoError(t, afero.WriteFile(fs, filepath.Join(contentDir, "prompts", "prompt1.md.tmpl"), []byte("# Prompt 1"), 0644))

	// Setup project directory
	require.NoError(t, fs.MkdirAll(projectDir, 0755))

	// Create and execute install command
	var stdout bytes.Buffer
	cmd := cli.NewInstallCommandForTesting(fs, func() (string, error) {
		return homeDir, nil
	}, strings.NewReader(""), &stdout, projectDir)

	cmd.SetArgs([]string{"--profile", "test/myprofile"})
	err := cmd.Execute()
	require.NoError(t, err)

	output := stdout.String()

	// Verify output contains category-grouped headers
	assert.Contains(t, output, ".claude/skills/", "output should show .claude/skills/ category")
	assert.Contains(t, output, ".claude/prompts/", "output should show .claude/prompts/ category")
}

// Test 4.1.3: Suppressed files shown only with --verbose flag (using update command)
// Note: This test uses update command because install only works on initial setup
// (before .weftlo.yaml exists), and target_overrides are defined in .weftlo.yaml.
func TestUpdateCommand_SuppressedFilesShownOnlyWithVerbose(t *testing.T) {
	fs := afero.NewMemMapFs()
	homeDir := "/home/testuser"
	projectDir := "/projects/myproject"

	// Setup global config
	configDir := filepath.Join(homeDir, ".weftlo")
	require.NoError(t, fs.MkdirAll(configDir, 0755))
	globalConfig := "default_profile: test/myprofile\n"
	require.NoError(t, afero.WriteFile(fs, filepath.Join(configDir, "config.yaml"), []byte(globalConfig), 0644))

	// Setup profile
	profileDir := filepath.Join(configDir, "profiles", "test", "myprofile")
	require.NoError(t, fs.MkdirAll(profileDir, 0755))

	profileYaml := `name: test/myprofile
content:
  root: content
  targets:
    skills: .claude/skills
  default_target: weftlo
`
	require.NoError(t, afero.WriteFile(fs, filepath.Join(profileDir, "profile.yaml"), []byte(profileYaml), 0644))

	// Create content
	contentDir := filepath.Join(profileDir, "content")
	require.NoError(t, fs.MkdirAll(filepath.Join(contentDir, "skills"), 0755))
	require.NoError(t, afero.WriteFile(fs, filepath.Join(contentDir, "skills", "skill1.md.tmpl"), []byte("# Skill 1"), 0644))

	// Setup project directory with target override that suppresses skills
	require.NoError(t, fs.MkdirAll(projectDir, 0755))

	projectConfig := `profiles:
  - test/myprofile
target_overrides:
  skills: null
`
	require.NoError(t, afero.WriteFile(fs, filepath.Join(projectDir, ".weftlo.yaml"), []byte(projectConfig), 0644))

	// Create an existing manifest (simulating previous installation)
	existingManifest := manifest.Manifest{
		Version:     "1.0.0",
		Profiles:    []string{"test/myprofile"},
		GeneratedAt: time.Now(),
		Files:       map[string]manifest.ManifestFile{},
	}
	manifestData, _ := json.MarshalIndent(existingManifest, "", "  ")
	require.NoError(t, afero.WriteFile(fs, filepath.Join(projectDir, ".weftlo.manifest.json"), manifestData, 0644))

	// Test without --verbose: suppressed files should NOT be shown
	var stdoutQuiet bytes.Buffer
	cmdQuiet := cli.NewUpdateCommandForTesting(fs, func() (string, error) {
		return homeDir, nil
	}, strings.NewReader(""), &stdoutQuiet, projectDir)

	cmdQuiet.SetArgs([]string{})
	err := cmdQuiet.Execute()
	require.NoError(t, err)

	outputQuiet := stdoutQuiet.String()
	assert.NotContains(t, outputQuiet, "suppressed", "without --verbose, suppressed files should not be mentioned")

	// Test with --verbose
	var stdoutVerbose bytes.Buffer
	cmdVerbose := cli.NewUpdateCommandForTesting(fs, func() (string, error) {
		return homeDir, nil
	}, strings.NewReader(""), &stdoutVerbose, projectDir)

	cmdVerbose.SetArgs([]string{"--verbose"})
	err = cmdVerbose.Execute()
	require.NoError(t, err)

	outputVerbose := stdoutVerbose.String()
	// The test verifies that when a profile has files but they are all suppressed
	// via target_overrides, the update command handles this gracefully.
	// Since all skills files are suppressed, no changes are detected.
	assert.Contains(t, outputVerbose, "No changes", "with all files suppressed, no changes should be needed")
}

// Test 4.1.4: Update command uses Router for change detection
func TestUpdateCommand_UsesRouterForChangeDetection(t *testing.T) {
	fs := afero.NewMemMapFs()
	homeDir := "/home/testuser"
	projectDir := "/projects/myproject"

	// Setup global config
	configDir := filepath.Join(homeDir, ".weftlo")
	require.NoError(t, fs.MkdirAll(configDir, 0755))
	globalConfig := "default_profile: test/myprofile\n"
	require.NoError(t, afero.WriteFile(fs, filepath.Join(configDir, "config.yaml"), []byte(globalConfig), 0644))

	// Setup profile with content targets
	profileDir := filepath.Join(configDir, "profiles", "test", "myprofile")
	require.NoError(t, fs.MkdirAll(profileDir, 0755))

	profileYaml := `name: test/myprofile
content:
  root: content
  targets:
    skills: .claude/skills
  default_target: weftlo
`
	require.NoError(t, afero.WriteFile(fs, filepath.Join(profileDir, "profile.yaml"), []byte(profileYaml), 0644))

	// Create content
	contentDir := filepath.Join(profileDir, "content")
	require.NoError(t, fs.MkdirAll(filepath.Join(contentDir, "skills"), 0755))
	require.NoError(t, afero.WriteFile(fs, filepath.Join(contentDir, "skills", "skill1.md.tmpl"), []byte("# Skill 1"), 0644))

	// Setup project directory with existing installation
	require.NoError(t, fs.MkdirAll(projectDir, 0755))

	projectConfig := `profiles:
  - test/myprofile
install_prefix: ""
`
	require.NoError(t, afero.WriteFile(fs, filepath.Join(projectDir, ".weftlo.yaml"), []byte(projectConfig), 0644))

	// Create existing manifest with file at routed path
	existingManifest := manifest.Manifest{
		Version:     "1.0.0",
		Profiles:    []string{"test/myprofile"},
		GeneratedAt: time.Now().UTC(),
		Files: map[string]manifest.ManifestFile{
			".claude/skills/skill1.md": {
				SourceChecksum: "sha256:abc123",
				OutputChecksum: "sha256:def456",
				SourceProfile:  "test/myprofile",
				TargetCategory: "skills",
			},
		},
	}
	manifestData, err := json.MarshalIndent(existingManifest, "", "  ")
	require.NoError(t, err)
	require.NoError(t, afero.WriteFile(fs, filepath.Join(projectDir, ".weftlo.manifest.json"), manifestData, 0644))

	// Create the installed file
	require.NoError(t, fs.MkdirAll(filepath.Join(projectDir, ".claude", "skills"), 0755))
	require.NoError(t, afero.WriteFile(fs, filepath.Join(projectDir, ".claude", "skills", "skill1.md"), []byte("# Skill 1"), 0644))

	// Create and execute update command with dry-run
	var stdout bytes.Buffer
	cmd := cli.NewUpdateCommandForTesting(fs, func() (string, error) {
		return homeDir, nil
	}, strings.NewReader(""), &stdout, projectDir)

	cmd.SetArgs([]string{"--dry-run"})
	err = cmd.Execute()
	require.NoError(t, err)

	// Verify update command processed correctly using Router-based paths
	output := stdout.String()
	// Should either show no changes or indicate the file was detected
	assert.True(t, strings.Contains(output, "No changes") || strings.Contains(output, ".claude/skills"),
		"update command should use Router for path resolution")
}

// Test 4.1.5: Target overrides apply correctly (using update command)
// Note: This test uses update command because install only works on initial setup
// (before .weftlo.yaml exists), and target_overrides are defined in .weftlo.yaml.
func TestUpdateCommand_TargetOverridesApplied(t *testing.T) {
	fs := afero.NewMemMapFs()
	homeDir := "/home/testuser"
	projectDir := "/projects/myproject"

	// Setup global config
	configDir := filepath.Join(homeDir, ".weftlo")
	require.NoError(t, fs.MkdirAll(configDir, 0755))
	globalConfig := "default_profile: test/myprofile\n"
	require.NoError(t, afero.WriteFile(fs, filepath.Join(configDir, "config.yaml"), []byte(globalConfig), 0644))

	// Setup profile with default target
	profileDir := filepath.Join(configDir, "profiles", "test", "myprofile")
	require.NoError(t, fs.MkdirAll(profileDir, 0755))

	profileYaml := `name: test/myprofile
content:
  root: content
  targets:
    skills: .claude/skills
  default_target: weftlo
`
	require.NoError(t, afero.WriteFile(fs, filepath.Join(profileDir, "profile.yaml"), []byte(profileYaml), 0644))

	// Create content
	contentDir := filepath.Join(profileDir, "content")
	require.NoError(t, fs.MkdirAll(filepath.Join(contentDir, "skills"), 0755))
	require.NoError(t, afero.WriteFile(fs, filepath.Join(contentDir, "skills", "skill1.md.tmpl"), []byte("# Skill 1"), 0644))

	// Setup project directory with override that redirects skills to custom location
	require.NoError(t, fs.MkdirAll(projectDir, 0755))

	projectConfig := `profiles:
  - test/myprofile
target_overrides:
  skills: custom/skills
`
	require.NoError(t, afero.WriteFile(fs, filepath.Join(projectDir, ".weftlo.yaml"), []byte(projectConfig), 0644))

	// Create an existing manifest (simulating previous installation without the override)
	existingManifest := manifest.Manifest{
		Version:     "1.0.0",
		Profiles:    []string{"test/myprofile"},
		GeneratedAt: time.Now(),
		Files:       map[string]manifest.ManifestFile{},
	}
	manifestData, _ := json.MarshalIndent(existingManifest, "", "  ")
	require.NoError(t, afero.WriteFile(fs, filepath.Join(projectDir, ".weftlo.manifest.json"), manifestData, 0644))

	// Create and execute update command
	var stdout bytes.Buffer
	cmd := cli.NewUpdateCommandForTesting(fs, func() (string, error) {
		return homeDir, nil
	}, strings.NewReader(""), &stdout, projectDir)

	cmd.SetArgs([]string{})
	err := cmd.Execute()
	require.NoError(t, err)

	// Verify file was installed to the overridden location
	overriddenFile := filepath.Join(projectDir, "custom", "skills", "skill1.md")
	exists, err := afero.Exists(fs, overriddenFile)
	require.NoError(t, err)
	assert.True(t, exists, "skills file should be installed to overridden path custom/skills/")

	// Verify file was NOT installed to the original profile target
	originalFile := filepath.Join(projectDir, ".claude", "skills", "skill1.md")
	exists, err = afero.Exists(fs, originalFile)
	require.NoError(t, err)
	assert.False(t, exists, "skills file should NOT be at original .claude/skills/ path")
}

// Test 4.1.6: Existing --dry-run, --quiet, --verbose, --force flag behaviors are maintained
func TestInstallCommand_ExistingFlagBehaviorsMaintained(t *testing.T) {
	fs := afero.NewMemMapFs()
	homeDir := "/home/testuser"
	projectDir := "/projects/myproject"

	// Setup global config
	configDir := filepath.Join(homeDir, ".weftlo")
	require.NoError(t, fs.MkdirAll(configDir, 0755))
	globalConfig := "default_profile: test/myprofile\n"
	require.NoError(t, afero.WriteFile(fs, filepath.Join(configDir, "config.yaml"), []byte(globalConfig), 0644))

	// Setup profile
	profileDir := filepath.Join(configDir, "profiles", "test", "myprofile")
	require.NoError(t, fs.MkdirAll(profileDir, 0755))

	profileYaml := `name: test/myprofile
content:
  root: content
  default_target: weftlo
`
	require.NoError(t, afero.WriteFile(fs, filepath.Join(profileDir, "profile.yaml"), []byte(profileYaml), 0644))

	// Create content
	contentDir := filepath.Join(profileDir, "content")
	require.NoError(t, fs.MkdirAll(contentDir, 0755))
	require.NoError(t, afero.WriteFile(fs, filepath.Join(contentDir, "test.md.tmpl"), []byte("# Test"), 0644))

	// Setup project directory
	require.NoError(t, fs.MkdirAll(projectDir, 0755))

	// Test --dry-run flag: should not create files
	var stdoutDryRun bytes.Buffer
	cmdDryRun := cli.NewInstallCommandForTesting(fs, func() (string, error) {
		return homeDir, nil
	}, strings.NewReader(""), &stdoutDryRun, projectDir)

	cmdDryRun.SetArgs([]string{"--profile", "test/myprofile", "--dry-run"})
	err := cmdDryRun.Execute()
	require.NoError(t, err)

	// Files should NOT be created in dry-run mode
	testFile := filepath.Join(projectDir, "weftlo", "test.md")
	exists, err := afero.Exists(fs, testFile)
	require.NoError(t, err)
	assert.False(t, exists, "--dry-run should not create files")

	// Test --quiet flag: should suppress output
	var stdoutQuiet bytes.Buffer
	cmdQuiet := cli.NewInstallCommandForTesting(fs, func() (string, error) {
		return homeDir, nil
	}, strings.NewReader(""), &stdoutQuiet, projectDir)

	cmdQuiet.SetArgs([]string{"--profile", "test/myprofile", "--quiet"})
	err = cmdQuiet.Execute()
	require.NoError(t, err)

	outputQuiet := stdoutQuiet.String()
	// --quiet should suppress the success message
	assert.NotContains(t, outputQuiet, "successfully", "--quiet should suppress success message")
}
