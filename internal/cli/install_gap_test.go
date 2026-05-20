package cli_test

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/afero"
	"gopkg.in/yaml.v3"

	"github.com/simensen/weftlo/internal/cli"
)

// =============================================================================
// Task Group 5: Strategic Gap-Fill Tests for Install Command Feature
// Maximum 10 additional tests to cover critical user workflows
// =============================================================================

// Test 1: Install command is registered under root command and accessible
func TestInstallCommand_RegisteredUnderRootCommand(t *testing.T) {
	memFs := afero.NewMemMapFs()
	rootCmd := cli.NewRootCommand(memFs, "test")

	// Find install subcommand
	var installFound bool
	for _, cmd := range rootCmd.Commands() {
		if cmd.Use == "install" {
			installFound = true

			// Verify essential properties
			if cmd.Short == "" {
				t.Error("expected install command to have Short description")
			}
			if cmd.RunE == nil {
				t.Error("expected install command to have RunE function")
			}

			// Verify local flags are available on install command
			localFlags := []string{"profile", "install-prefix", "dry-run", "force"}
			for _, flagName := range localFlags {
				if cmd.Flags().Lookup(flagName) == nil {
					t.Errorf("expected --%s flag to be registered on install command", flagName)
				}
			}
			break
		}
	}

	if !installFound {
		t.Error("expected install subcommand to be registered under root command")
	}

	// Verify --quiet is a persistent flag on root (not local to install)
	if rootCmd.PersistentFlags().Lookup("quiet") == nil {
		t.Error("expected --quiet flag to be registered as persistent flag on root command")
	}
}

// Test 2: Install when .weftlo.yaml exists returns AlreadyInstalledError
func TestInstallCommand_FailsWhenProjectConfigExists(t *testing.T) {
	memFs := afero.NewMemMapFs()
	homeDir := "/home/testuser"

	// Setup global config with default profile
	setupGlobalConfig(t, memFs, homeDir, "global/fallback", "")
	setupProfile(t, memFs, homeDir, "global/fallback")

	// Create project directory in memFs and change to a real temp directory
	projectDir := t.TempDir()
	originalWd, _ := os.Getwd()
	defer os.Chdir(originalWd)
	if err := os.Chdir(projectDir); err != nil {
		t.Fatalf("failed to change to project directory: %v", err)
	}

	// Get the actual working directory (may differ due to symlinks on macOS)
	actualProjectDir, _ := os.Getwd()

	// Create .weftlo.yaml in memFs at the actual project path
	projectConfigContent := []byte("profiles:\n  - global/fallback\n")
	if err := afero.WriteFile(memFs, filepath.Join(actualProjectDir, ".weftlo.yaml"), projectConfigContent, 0644); err != nil {
		t.Fatalf("failed to write project config: %v", err)
	}

	var stdout bytes.Buffer
	installCmd := cli.NewInstallCommandWithIO(memFs, func() (string, error) {
		return homeDir, nil
	}, strings.NewReader(""), &stdout)

	installCmd.SetArgs([]string{})
	err := installCmd.Execute()

	// Should fail with AlreadyInstalledError because .weftlo.yaml exists
	if err == nil {
		t.Fatal("expected error when .weftlo.yaml already exists")
	}

	var alreadyInstalledErr *cli.AlreadyInstalledError
	if !errors.As(err, &alreadyInstalledErr) {
		t.Errorf("expected AlreadyInstalledError, got: %v", err)
	}

	// Verify error message mentions using update instead
	errMsg := err.Error()
	if !strings.Contains(errMsg, "weftlo update") {
		t.Errorf("expected error to mention 'weftlo update', got: %s", errMsg)
	}
}

// Test 3: Install with invalid profile name format returns validation error
func TestInstallCommand_InvalidProfileNameFormatError(t *testing.T) {
	memFs := afero.NewMemMapFs()
	homeDir := "/home/testuser"

	// Setup minimal global config
	setupGlobalConfig(t, memFs, homeDir, "default/default", "")
	setupProfile(t, memFs, homeDir, "default/default")

	// Save and restore working directory
	originalWd, _ := os.Getwd()
	defer os.Chdir(originalWd)
	tempDir := t.TempDir()
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("failed to change directory: %v", err)
	}

	var stdout bytes.Buffer
	installCmd := cli.NewInstallCommandWithIO(memFs, func() (string, error) {
		return homeDir, nil
	}, strings.NewReader(""), &stdout)

	// Use invalid profile name (no slash)
	installCmd.SetArgs([]string{"--profile", "invalid-no-slash"})
	err := installCmd.Execute()

	if err == nil {
		t.Fatal("expected error with invalid profile name format")
	}

	// Verify error message mentions invalid profile name format
	errMsg := err.Error()
	if !strings.Contains(errMsg, "vendor/name format") {
		t.Errorf("expected error about invalid profile name format, got: %s", errMsg)
	}
}

// Test 4: Install when global config exists but default_profile is not set returns ProfileResolutionError
func TestInstallCommand_NoDefaultProfileReturnsProfileResolutionError(t *testing.T) {
	memFs := afero.NewMemMapFs()
	homeDir := "/home/testuser"

	// Setup global config WITHOUT default_profile (empty)
	configDir := filepath.Join(homeDir, ".weftlo")
	if err := memFs.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("failed to create config directory: %v", err)
	}
	// Write config with empty default_profile
	configContent := []byte("install_prefix: some-prefix\n")
	if err := afero.WriteFile(memFs, filepath.Join(configDir, "config.yaml"), configContent, 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	// Save and restore working directory
	originalWd, _ := os.Getwd()
	defer os.Chdir(originalWd)
	tempDir := t.TempDir()
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("failed to change directory: %v", err)
	}

	var stdout bytes.Buffer
	installCmd := cli.NewInstallCommandWithIO(memFs, func() (string, error) {
		return homeDir, nil
	}, strings.NewReader(""), &stdout)

	// No --profile flag, no project config, no global default_profile
	installCmd.SetArgs([]string{})
	err := installCmd.Execute()

	if err == nil {
		t.Fatal("expected ProfileResolutionError when no profile can be resolved")
	}

	var profileResErr *cli.ProfileResolutionError
	if !errors.As(err, &profileResErr) {
		t.Errorf("expected ProfileResolutionError, got: %v", err)
	}

	if profileResErr != nil {
		errMsg := profileResErr.Error()
		// Verify error mentions the resolution sources that were checked
		if !strings.Contains(errMsg, "--profile flag") {
			t.Errorf("expected error to mention '--profile flag', got: %s", errMsg)
		}
		if !strings.Contains(errMsg, "global config") {
			t.Errorf("expected error to mention 'global config', got: %s", errMsg)
		}
	}
}

// Test 5: Install when profile does not exist in profiles directory
func TestInstallCommand_ProfileNotFoundError(t *testing.T) {
	memFs := afero.NewMemMapFs()
	homeDir := "/home/testuser"

	// Setup global config with a profile name that doesn't have a matching directory
	setupGlobalConfig(t, memFs, homeDir, "nonexistent/profile", "")
	// DO NOT create the profile directory

	// Save and restore working directory
	originalWd, _ := os.Getwd()
	defer os.Chdir(originalWd)
	tempDir := t.TempDir()
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("failed to change directory: %v", err)
	}

	var stdout bytes.Buffer
	installCmd := cli.NewInstallCommandWithIO(memFs, func() (string, error) {
		return homeDir, nil
	}, strings.NewReader(""), &stdout)

	installCmd.SetArgs([]string{})
	err := installCmd.Execute()

	if err == nil {
		t.Fatal("expected error when profile does not exist")
	}

	// Verify error mentions failed to load profile
	errMsg := err.Error()
	if !strings.Contains(errMsg, "failed to load profile") {
		t.Errorf("expected error about failed profile loading, got: %s", errMsg)
	}
}

// Test 6: Install with --install-prefix flag creates .weftlo.yaml with install_prefix included
func TestInstallCommand_CreatesProjectConfigWithInstallPrefix(t *testing.T) {
	memFs := afero.NewMemMapFs()
	homeDir := "/home/testuser"

	// Setup global config and profile
	setupGlobalConfig(t, memFs, homeDir, "default/default", "")
	setupProfile(t, memFs, homeDir, "default/default")

	// Create project directory (no .weftlo.yaml)
	projectDir := t.TempDir()

	// Save and restore working directory
	originalWd, _ := os.Getwd()
	defer os.Chdir(originalWd)
	if err := os.Chdir(projectDir); err != nil {
		t.Fatalf("failed to change directory: %v", err)
	}

	// Get the actual working directory (handles symlinks on macOS)
	actualProjectDir, _ := os.Getwd()

	var stdout bytes.Buffer
	installCmd := cli.NewInstallCommandWithIO(memFs, func() (string, error) {
		return homeDir, nil
	}, strings.NewReader(""), &stdout)

	// Specify --install-prefix flag
	installCmd.SetArgs([]string{"--install-prefix", "custom-install-dir"})
	err := installCmd.Execute()

	if err != nil {
		t.Fatalf("expected install to succeed, got error: %v", err)
	}

	// Verify .weftlo.yaml was created with install_prefix (using actual path)
	projectConfigPath := filepath.Join(actualProjectDir, ".weftlo.yaml")
	content, err := afero.ReadFile(memFs, projectConfigPath)
	if err != nil {
		t.Fatalf("failed to read created project config at %s: %v", projectConfigPath, err)
	}

	var config struct {
		Profiles      []string `yaml:"profiles"`
		InstallPrefix string   `yaml:"install_prefix"`
	}
	if err := yaml.Unmarshal(content, &config); err != nil {
		t.Fatalf("failed to parse project config: %v", err)
	}

	if len(config.Profiles) == 0 || config.Profiles[0] != "default/default" {
		t.Errorf("expected profiles to be ['default/default'], got %v", config.Profiles)
	}
	if config.InstallPrefix != "custom-install-dir" {
		t.Errorf("expected install_prefix to be 'custom-install-dir', got '%s'", config.InstallPrefix)
	}
}

// Test 7: Install creates .weftlo.yaml WITHOUT install_prefix when flag not specified
func TestInstallCommand_CreatesProjectConfigWithoutInstallPrefixWhenNotSpecified(t *testing.T) {
	memFs := afero.NewMemMapFs()
	homeDir := "/home/testuser"

	// Setup global config and profile
	setupGlobalConfig(t, memFs, homeDir, "default/default", "")
	setupProfile(t, memFs, homeDir, "default/default")

	// Create project directory (no .weftlo.yaml)
	projectDir := t.TempDir()

	// Save and restore working directory
	originalWd, _ := os.Getwd()
	defer os.Chdir(originalWd)
	if err := os.Chdir(projectDir); err != nil {
		t.Fatalf("failed to change directory: %v", err)
	}

	// Get the actual working directory (handles symlinks on macOS)
	actualProjectDir, _ := os.Getwd()

	var stdout bytes.Buffer
	installCmd := cli.NewInstallCommandWithIO(memFs, func() (string, error) {
		return homeDir, nil
	}, strings.NewReader(""), &stdout)

	// NO --install-prefix flag
	installCmd.SetArgs([]string{})
	err := installCmd.Execute()

	if err != nil {
		t.Fatalf("expected install to succeed, got error: %v", err)
	}

	// Verify .weftlo.yaml was created WITHOUT install_prefix (using actual path)
	projectConfigPath := filepath.Join(actualProjectDir, ".weftlo.yaml")
	content, err := afero.ReadFile(memFs, projectConfigPath)
	if err != nil {
		t.Fatalf("failed to read created project config at %s: %v", projectConfigPath, err)
	}

	// Check that install_prefix is NOT in the raw YAML
	if strings.Contains(string(content), "install_prefix") {
		t.Errorf("expected install_prefix to be omitted from project config when flag not specified, got: %s", string(content))
	}

	// Still verify profiles is present
	var config struct {
		Profiles []string `yaml:"profiles"`
	}
	if err := yaml.Unmarshal(content, &config); err != nil {
		t.Fatalf("failed to parse project config: %v", err)
	}
	if len(config.Profiles) == 0 || config.Profiles[0] != "default/default" {
		t.Errorf("expected profiles to be ['default/default'], got %v", config.Profiles)
	}
}

// Test 8: Integration test - full install workflow writes files to filesystem
func TestInstallCommand_FullWorkflowWritesFilesToFilesystem(t *testing.T) {
	memFs := afero.NewMemMapFs()
	homeDir := "/home/testuser"

	// Setup global config and profile
	setupGlobalConfig(t, memFs, homeDir, "test/profile", "")
	setupProfile(t, memFs, homeDir, "test/profile")

	// Create project directory
	projectDir := t.TempDir()

	// Save and restore working directory
	originalWd, _ := os.Getwd()
	defer os.Chdir(originalWd)
	if err := os.Chdir(projectDir); err != nil {
		t.Fatalf("failed to change directory: %v", err)
	}

	// Get the actual working directory (handles symlinks on macOS)
	actualProjectDir, _ := os.Getwd()

	var stdout bytes.Buffer
	installCmd := cli.NewInstallCommandWithIO(memFs, func() (string, error) {
		return homeDir, nil
	}, strings.NewReader(""), &stdout)

	installCmd.SetArgs([]string{})
	err := installCmd.Execute()

	if err != nil {
		t.Fatalf("expected install to succeed, got error: %v", err)
	}

	// Verify output indicates success
	output := stdout.String()
	if !strings.Contains(output, "test/profile") {
		t.Errorf("expected output to mention installed profile, got: %s", output)
	}
	if !strings.Contains(output, "successfully") {
		t.Errorf("expected success message in output, got: %s", output)
	}

	// Verify .weftlo.yaml was created (using actual path)
	projectConfigPath := filepath.Join(actualProjectDir, ".weftlo.yaml")
	exists, err := afero.Exists(memFs, projectConfigPath)
	if err != nil {
		t.Fatalf("error checking project config: %v", err)
	}
	if !exists {
		t.Error("expected .weftlo.yaml to be created")
	}

	// Verify template file was written (check for test.md in weftlo/)
	templateOutputPath := filepath.Join(actualProjectDir, ".claude", "test.md")
	exists, err = afero.Exists(memFs, templateOutputPath)
	if err != nil {
		t.Fatalf("error checking template output: %v", err)
	}
	if !exists {
		t.Errorf("expected template output file at %s to exist", templateOutputPath)
	}
}

// Test 9: Install via root command works end-to-end
func TestInstallCommand_ViaRootCommandEndToEnd(t *testing.T) {
	memFs := afero.NewMemMapFs()
	homeDir := "/home/testuser"

	// Setup global config and profile
	setupGlobalConfig(t, memFs, homeDir, "root/test", "")
	setupProfile(t, memFs, homeDir, "root/test")

	// Create and change to project directory
	projectDir := t.TempDir()
	originalWd, _ := os.Getwd()
	defer os.Chdir(originalWd)
	if err := os.Chdir(projectDir); err != nil {
		t.Fatalf("failed to change directory: %v", err)
	}

	rootCmd := cli.NewRootCommandWithHomeDir(memFs, func() (string, error) {
		return homeDir, nil
	})

	var stdout bytes.Buffer
	rootCmd.SetOut(&stdout)
	rootCmd.SetArgs([]string{"install"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("expected install via root command to succeed, got: %v", err)
	}

	// Verify output
	output := stdout.String()
	if !strings.Contains(output, "root/test") {
		t.Errorf("expected output to mention installed profile 'root/test', got: %s", output)
	}
}

// Test 10: Install with --dry-run outputs planned changes and does NOT write files
func TestInstallCommand_DryRunOutputsPlanWithoutWriting(t *testing.T) {
	memFs := afero.NewMemMapFs()
	homeDir := "/home/testuser"

	// Setup global config and profile
	setupGlobalConfig(t, memFs, homeDir, "dryrun/test", "")
	setupProfile(t, memFs, homeDir, "dryrun/test")

	// Create project directory
	projectDir := t.TempDir()

	// Save and restore working directory
	originalWd, _ := os.Getwd()
	defer os.Chdir(originalWd)
	if err := os.Chdir(projectDir); err != nil {
		t.Fatalf("failed to change directory: %v", err)
	}

	// Get the actual working directory (handles symlinks on macOS)
	actualProjectDir, _ := os.Getwd()

	var stdout bytes.Buffer
	installCmd := cli.NewInstallCommandWithIO(memFs, func() (string, error) {
		return homeDir, nil
	}, strings.NewReader(""), &stdout)

	installCmd.SetArgs([]string{"--dry-run"})
	err := installCmd.Execute()

	if err != nil {
		t.Fatalf("expected dry-run install to succeed, got error: %v", err)
	}

	// Verify NO files were written to project directory (using actual path)
	projectConfigPath := filepath.Join(actualProjectDir, ".weftlo.yaml")
	exists, err := afero.Exists(memFs, projectConfigPath)
	if err != nil {
		t.Fatalf("error checking project config: %v", err)
	}
	if exists {
		t.Error("expected .weftlo.yaml to NOT be created in dry-run mode")
	}

	// Verify template output was NOT written
	templateOutputPath := filepath.Join(actualProjectDir, ".claude", "test.md")
	exists, err = afero.Exists(memFs, templateOutputPath)
	if err != nil {
		t.Fatalf("error checking template output: %v", err)
	}
	if exists {
		t.Error("expected template files to NOT be written in dry-run mode")
	}
}
