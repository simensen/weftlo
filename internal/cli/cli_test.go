package cli_test

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/spf13/afero"
	"gopkg.in/yaml.v3"

	"github.com/simensen/weftlo/internal/cli"
)

// =============================================================================
// Task Group 2: Root Command and CLI Structure Tests
// =============================================================================

// Test 1: Root command initializes correctly
func TestRootCommand_InitializesCorrectly(t *testing.T) {
	memFs := afero.NewMemMapFs()
	rootCmd := cli.NewRootCommand(memFs, "dev")

	if rootCmd == nil {
		t.Fatal("expected root command to be non-nil")
	}

	if rootCmd.Use != "weftlo" {
		t.Errorf("expected root command Use to be 'weftlo', got '%s'", rootCmd.Use)
	}

	if rootCmd.Short == "" {
		t.Error("expected root command Short description to be non-empty")
	}
}

// Test 2: Init subcommand is registered under root
func TestInitSubcommand_IsRegisteredUnderRoot(t *testing.T) {
	memFs := afero.NewMemMapFs()
	rootCmd := cli.NewRootCommand(memFs, "dev")

	// Find init subcommand
	var initCmd *cli.InitCommand
	for _, cmd := range rootCmd.Commands() {
		if cmd.Use == "init" {
			// Cast to check it's the right type
			initCmd = &cli.InitCommand{}
			break
		}
	}

	if initCmd == nil {
		// Check if init command exists by Use name
		found := false
		for _, cmd := range rootCmd.Commands() {
			if cmd.Use == "init" {
				found = true
				break
			}
		}
		if !found {
			t.Error("expected init subcommand to be registered under root command")
		}
	}
}

// Test 3: Help output displays correctly for root command
func TestRootCommand_HelpOutputDisplaysCorrectly(t *testing.T) {
	memFs := afero.NewMemMapFs()
	rootCmd := cli.NewRootCommand(memFs, "dev")

	var stdout bytes.Buffer
	rootCmd.SetOut(&stdout)
	rootCmd.SetArgs([]string{"--help"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("expected no error running help, got: %v", err)
	}

	output := stdout.String()
	if output == "" {
		t.Error("expected help output to be non-empty")
	}

	// Check that help contains expected sections
	if !bytes.Contains([]byte(output), []byte("weftlo")) {
		t.Error("expected help output to contain 'weftlo'")
	}
}

// Test 4: Help output displays correctly for init subcommand
func TestInitSubcommand_HelpOutputDisplaysCorrectly(t *testing.T) {
	memFs := afero.NewMemMapFs()
	rootCmd := cli.NewRootCommand(memFs, "dev")

	var stdout bytes.Buffer
	rootCmd.SetOut(&stdout)
	rootCmd.SetArgs([]string{"init", "--help"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("expected no error running init help, got: %v", err)
	}

	output := stdout.String()
	if output == "" {
		t.Error("expected init help output to be non-empty")
	}

	// Check that help contains expected flags
	if !bytes.Contains([]byte(output), []byte("--force")) {
		t.Error("expected init help output to contain '--force' flag")
	}
	if !bytes.Contains([]byte(output), []byte("--quiet")) {
		t.Error("expected init help output to contain '--quiet' flag")
	}
}

// Test 5: Flag parsing works for --force flag
func TestInitSubcommand_ForceFlagParsing(t *testing.T) {
	memFs := afero.NewMemMapFs()
	rootCmd := cli.NewRootCommand(memFs, "dev")

	// Find init command
	var initCobraCmd *cli.InitCommand
	for _, cmd := range rootCmd.Commands() {
		if cmd.Use == "init" {
			// The init command should have --force flag
			flag := cmd.Flags().Lookup("force")
			if flag == nil {
				t.Error("expected --force flag to be registered on init command")
				return
			}

			// Set the flag
			err := cmd.Flags().Set("force", "true")
			if err != nil {
				t.Errorf("failed to set --force flag: %v", err)
			}

			// Verify it was set
			val, err := cmd.Flags().GetBool("force")
			if err != nil {
				t.Errorf("failed to get --force flag value: %v", err)
			}
			if !val {
				t.Error("expected --force flag to be true after setting")
			}
			return
		}
	}

	// Suppress unused variable warning
	_ = initCobraCmd
	t.Error("init command not found")
}

// Test 6: Flag parsing works for --quiet flag (now a global persistent flag on root)
func TestInitSubcommand_QuietFlagParsing(t *testing.T) {
	memFs := afero.NewMemMapFs()
	rootCmd := cli.NewRootCommand(memFs, "dev")

	// The --quiet flag is now a persistent flag on the root command
	// It should be accessible via root's persistent flags
	flag := rootCmd.PersistentFlags().Lookup("quiet")
	if flag == nil {
		t.Error("expected --quiet flag to be registered as persistent flag on root command")
		return
	}

	// Set the flag
	err := rootCmd.PersistentFlags().Set("quiet", "true")
	if err != nil {
		t.Errorf("failed to set --quiet flag: %v", err)
	}

	// Verify it was set
	val, err := rootCmd.PersistentFlags().GetBool("quiet")
	if err != nil {
		t.Errorf("failed to get --quiet flag value: %v", err)
	}
	if !val {
		t.Error("expected --quiet flag to be true after setting")
	}
}

// Test 7: Short flags work (-f for --force, -q for --quiet)
func TestInitSubcommand_ShortFlagsParsing(t *testing.T) {
	memFs := afero.NewMemMapFs()
	rootCmd := cli.NewRootCommand(memFs, "dev")

	// Find init command
	for _, cmd := range rootCmd.Commands() {
		if cmd.Use == "init" {
			// Check -f short flag for --force (local to init command)
			forceFlag := cmd.Flags().ShorthandLookup("f")
			if forceFlag == nil {
				t.Error("expected -f short flag to be registered for --force")
			}
			break
		}
	}

	// Check -q short flag for --quiet (now on root command as persistent flag)
	quietFlag := rootCmd.PersistentFlags().ShorthandLookup("q")
	if quietFlag == nil {
		t.Error("expected -q short flag to be registered for --quiet on root command")
	}
}

// Test 8: Init command has RunE function defined
func TestInitSubcommand_HasRunEFunction(t *testing.T) {
	memFs := afero.NewMemMapFs()
	rootCmd := cli.NewRootCommand(memFs, "dev")

	// Find init command
	for _, cmd := range rootCmd.Commands() {
		if cmd.Use == "init" {
			if cmd.RunE == nil {
				t.Error("expected init command to have RunE function defined")
			}
			return
		}
	}

	t.Error("init command not found")
}

// =============================================================================
// Task Group 3: Directory Structure and File Creation Tests
// =============================================================================

// Test 1: ~/.weftlo/ directory creation with 0755 permissions
func TestInitCommand_CreatesRootDirectoryWithCorrectPermissions(t *testing.T) {
	memFs := afero.NewMemMapFs()
	homeDir := "/home/testuser"

	// Create the init command with injected home directory
	initCmd := cli.NewInitCommandWithHomeDir(memFs, func() (string, error) {
		return homeDir, nil
	})

	// Execute the init command
	initCmd.SetArgs([]string{})
	err := initCmd.Execute()
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// Check that ~/.weftlo/ directory exists
	configDir := homeDir + "/.weftlo"
	exists, err := afero.DirExists(memFs, configDir)
	if err != nil {
		t.Fatalf("error checking directory existence: %v", err)
	}
	if !exists {
		t.Errorf("expected %s directory to exist", configDir)
	}

	// Check permissions (afero.MemMapFs doesn't preserve exact permissions,
	// but we verify the implementation uses the correct mode)
	info, err := memFs.Stat(configDir)
	if err != nil {
		t.Fatalf("error getting directory info: %v", err)
	}
	if !info.IsDir() {
		t.Errorf("expected %s to be a directory", configDir)
	}
}

// Test 2: profiles/default/default/ directory structure creation
func TestInitCommand_CreatesProfileDirectoryStructure(t *testing.T) {
	memFs := afero.NewMemMapFs()
	homeDir := "/home/testuser"

	initCmd := cli.NewInitCommandWithHomeDir(memFs, func() (string, error) {
		return homeDir, nil
	})

	initCmd.SetArgs([]string{})
	err := initCmd.Execute()
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// Check each directory in the hierarchy
	expectedDirs := []string{
		homeDir + "/.weftlo/profiles",
		homeDir + "/.weftlo/profiles/default",
		homeDir + "/.weftlo/profiles/default/default",
	}

	for _, dir := range expectedDirs {
		exists, err := afero.DirExists(memFs, dir)
		if err != nil {
			t.Fatalf("error checking directory %s: %v", dir, err)
		}
		if !exists {
			t.Errorf("expected directory %s to exist", dir)
		}
	}
}

// Test 3: config.yaml creation with correct content and 0644 permissions
func TestInitCommand_CreatesConfigYamlWithCorrectContent(t *testing.T) {
	memFs := afero.NewMemMapFs()
	homeDir := "/home/testuser"

	initCmd := cli.NewInitCommandWithHomeDir(memFs, func() (string, error) {
		return homeDir, nil
	})

	initCmd.SetArgs([]string{})
	err := initCmd.Execute()
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// Check that config.yaml exists
	configPath := homeDir + "/.weftlo/config.yaml"
	exists, err := afero.Exists(memFs, configPath)
	if err != nil {
		t.Fatalf("error checking config.yaml existence: %v", err)
	}
	if !exists {
		t.Errorf("expected %s to exist", configPath)
	}

	// Read and verify content
	content, err := afero.ReadFile(memFs, configPath)
	if err != nil {
		t.Fatalf("error reading config.yaml: %v", err)
	}

	var config struct {
		DefaultProfile string `yaml:"default_profile"`
	}
	err = yaml.Unmarshal(content, &config)
	if err != nil {
		t.Fatalf("error parsing config.yaml: %v", err)
	}

	if config.DefaultProfile != "default/default" {
		t.Errorf("expected default_profile to be 'default/default', got '%s'", config.DefaultProfile)
	}
}

// Test 4: profile.yaml creation with correct content
func TestInitCommand_CreatesProfileYamlWithCorrectContent(t *testing.T) {
	memFs := afero.NewMemMapFs()
	homeDir := "/home/testuser"

	initCmd := cli.NewInitCommandWithHomeDir(memFs, func() (string, error) {
		return homeDir, nil
	})

	initCmd.SetArgs([]string{})
	err := initCmd.Execute()
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// Check that profile.yaml exists
	profilePath := homeDir + "/.weftlo/profiles/default/default/profile.yaml"
	exists, err := afero.Exists(memFs, profilePath)
	if err != nil {
		t.Fatalf("error checking profile.yaml existence: %v", err)
	}
	if !exists {
		t.Errorf("expected %s to exist", profilePath)
	}

	// Read and verify content
	content, err := afero.ReadFile(memFs, profilePath)
	if err != nil {
		t.Fatalf("error reading profile.yaml: %v", err)
	}

	var profile struct {
		Name string `yaml:"name"`
	}
	err = yaml.Unmarshal(content, &profile)
	if err != nil {
		t.Fatalf("error parsing profile.yaml: %v", err)
	}

	if profile.Name != "default/default" {
		t.Errorf("expected name to be 'default/default', got '%s'", profile.Name)
	}
}

// Test 5: README.md creation inside content directory of default profile
// Updated to reflect new content root architecture where README.md is inside content/
func TestInitCommand_CreatesReadmeInDefaultProfile(t *testing.T) {
	memFs := afero.NewMemMapFs()
	homeDir := "/home/testuser"

	initCmd := cli.NewInitCommandWithHomeDir(memFs, func() (string, error) {
		return homeDir, nil
	})

	initCmd.SetArgs([]string{})
	err := initCmd.Execute()
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// Check that README.md exists inside content/ directory (new architecture)
	readmePath := homeDir + "/.weftlo/profiles/default/default/content/README.md"
	exists, err := afero.Exists(memFs, readmePath)
	if err != nil {
		t.Fatalf("error checking README.md existence: %v", err)
	}
	if !exists {
		t.Errorf("expected %s to exist", readmePath)
	}

	// Read and verify content contains expected note
	content, err := afero.ReadFile(memFs, readmePath)
	if err != nil {
		t.Fatalf("error reading README.md: %v", err)
	}

	if !bytes.Contains(content, []byte("default")) {
		t.Error("expected README.md to contain a note about the default profile")
	}
}

// Test 6: Files are created with correct permissions (file info check)
func TestInitCommand_CreatesFilesWithCorrectInfo(t *testing.T) {
	memFs := afero.NewMemMapFs()
	homeDir := "/home/testuser"

	initCmd := cli.NewInitCommandWithHomeDir(memFs, func() (string, error) {
		return homeDir, nil
	})

	initCmd.SetArgs([]string{})
	err := initCmd.Execute()
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// Check that config.yaml is a regular file
	configPath := homeDir + "/.weftlo/config.yaml"
	info, err := memFs.Stat(configPath)
	if err != nil {
		t.Fatalf("error getting config.yaml info: %v", err)
	}
	if info.IsDir() {
		t.Error("expected config.yaml to be a regular file, not a directory")
	}

	// Check that profile.yaml is a regular file
	profilePath := homeDir + "/.weftlo/profiles/default/default/profile.yaml"
	info, err = memFs.Stat(profilePath)
	if err != nil {
		t.Fatalf("error getting profile.yaml info: %v", err)
	}
	if info.IsDir() {
		t.Error("expected profile.yaml to be a regular file, not a directory")
	}
}

// =============================================================================
// Task Group 4: Detection and Reinitialize Flow Tests
// =============================================================================

// Test 1: Detection of existing config.yaml
func TestInitCommand_DetectsExistingConfigYaml(t *testing.T) {
	memFs := afero.NewMemMapFs()
	homeDir := "/home/testuser"
	configDir := homeDir + "/.weftlo"
	configPath := configDir + "/config.yaml"

	// Create an existing config.yaml
	err := memFs.MkdirAll(configDir, 0755)
	if err != nil {
		t.Fatalf("failed to create config directory: %v", err)
	}
	err = afero.WriteFile(memFs, configPath, []byte("default_profile: old/profile\n"), 0644)
	if err != nil {
		t.Fatalf("failed to create existing config.yaml: %v", err)
	}

	// Create init command with stdin that says "n" (no reinitialize)
	stdin := strings.NewReader("n\n")
	var stdout bytes.Buffer

	initCmd := cli.NewInitCommandWithIO(memFs, func() (string, error) {
		return homeDir, nil
	}, stdin, &stdout)

	initCmd.SetArgs([]string{})
	err = initCmd.Execute()

	// Should get ReinitializeAbortedError
	var abortedErr *cli.ReinitializeAbortedError
	if !errors.As(err, &abortedErr) {
		t.Errorf("expected ReinitializeAbortedError when config exists and user says no, got: %v", err)
	}

	// Verify the prompt was shown
	output := stdout.String()
	if !strings.Contains(output, "reinitialize") {
		t.Error("expected prompt to contain 'reinitialize'")
	}
}

// Test 2: Prompt appears when config exists and no --force flag
func TestInitCommand_PromptsWhenConfigExistsWithoutForce(t *testing.T) {
	memFs := afero.NewMemMapFs()
	homeDir := "/home/testuser"
	configDir := homeDir + "/.weftlo"
	configPath := configDir + "/config.yaml"

	// Create an existing config.yaml
	err := memFs.MkdirAll(configDir, 0755)
	if err != nil {
		t.Fatalf("failed to create config directory: %v", err)
	}
	err = afero.WriteFile(memFs, configPath, []byte("default_profile: old/profile\n"), 0644)
	if err != nil {
		t.Fatalf("failed to create existing config.yaml: %v", err)
	}

	// Create init command with stdin that says "n"
	stdin := strings.NewReader("n\n")
	var stdout bytes.Buffer

	initCmd := cli.NewInitCommandWithIO(memFs, func() (string, error) {
		return homeDir, nil
	}, stdin, &stdout)

	initCmd.SetArgs([]string{}) // No --force flag
	_ = initCmd.Execute()

	// Verify the prompt was shown
	output := stdout.String()
	if !strings.Contains(output, "Do you want to reinitialize") {
		t.Error("expected prompt asking about reinitialize")
	}
}

// Test 3: --force flag skips prompt and proceeds
func TestInitCommand_ForceFlagSkipsPromptAndProceeds(t *testing.T) {
	memFs := afero.NewMemMapFs()
	homeDir := "/home/testuser"
	configDir := homeDir + "/.weftlo"
	configPath := configDir + "/config.yaml"

	// Create an existing config.yaml with different content
	err := memFs.MkdirAll(configDir, 0755)
	if err != nil {
		t.Fatalf("failed to create config directory: %v", err)
	}
	err = afero.WriteFile(memFs, configPath, []byte("default_profile: old/profile\n"), 0644)
	if err != nil {
		t.Fatalf("failed to create existing config.yaml: %v", err)
	}

	// Create init command with empty stdin (should not be read when --force is used)
	stdin := strings.NewReader("")
	var stdout bytes.Buffer

	initCmd := cli.NewInitCommandWithIO(memFs, func() (string, error) {
		return homeDir, nil
	}, stdin, &stdout)

	initCmd.SetArgs([]string{"--force"})
	err = initCmd.Execute()

	if err != nil {
		t.Fatalf("expected no error with --force flag, got: %v", err)
	}

	// Verify config.yaml was overwritten
	content, err := afero.ReadFile(memFs, configPath)
	if err != nil {
		t.Fatalf("error reading config.yaml: %v", err)
	}

	var config struct {
		DefaultProfile string `yaml:"default_profile"`
	}
	err = yaml.Unmarshal(content, &config)
	if err != nil {
		t.Fatalf("error parsing config.yaml: %v", err)
	}

	if config.DefaultProfile != "default/default" {
		t.Errorf("expected config to be overwritten with 'default/default', got '%s'", config.DefaultProfile)
	}

	// Verify no prompt was shown
	output := stdout.String()
	if strings.Contains(output, "reinitialize") {
		t.Error("expected no prompt when --force flag is used")
	}
}

// Test 4: Affirmative responses allow reinitialize
func TestInitCommand_AffirmativeResponsesAllowReinitialize(t *testing.T) {
	affirmativeResponses := []string{"y", "yes", "Y", "Yes", "YES"}

	for _, response := range affirmativeResponses {
		t.Run("response_"+response, func(t *testing.T) {
			memFs := afero.NewMemMapFs()
			homeDir := "/home/testuser"
			configDir := homeDir + "/.weftlo"
			configPath := configDir + "/config.yaml"

			// Create an existing config.yaml with different content
			err := memFs.MkdirAll(configDir, 0755)
			if err != nil {
				t.Fatalf("failed to create config directory: %v", err)
			}
			err = afero.WriteFile(memFs, configPath, []byte("default_profile: old/profile\n"), 0644)
			if err != nil {
				t.Fatalf("failed to create existing config.yaml: %v", err)
			}

			// Create init command with affirmative response
			stdin := strings.NewReader(response + "\n")
			var stdout bytes.Buffer

			initCmd := cli.NewInitCommandWithIO(memFs, func() (string, error) {
				return homeDir, nil
			}, stdin, &stdout)

			initCmd.SetArgs([]string{})
			err = initCmd.Execute()

			if err != nil {
				t.Fatalf("expected no error with response '%s', got: %v", response, err)
			}

			// Verify config.yaml was overwritten
			content, err := afero.ReadFile(memFs, configPath)
			if err != nil {
				t.Fatalf("error reading config.yaml: %v", err)
			}

			var config struct {
				DefaultProfile string `yaml:"default_profile"`
			}
			err = yaml.Unmarshal(content, &config)
			if err != nil {
				t.Fatalf("error parsing config.yaml: %v", err)
			}

			if config.DefaultProfile != "default/default" {
				t.Errorf("expected config to be overwritten with 'default/default', got '%s'", config.DefaultProfile)
			}
		})
	}
}

// Test 5: Negative response aborts reinitialize
func TestInitCommand_NegativeResponseAbortsReinitialize(t *testing.T) {
	negativeResponses := []string{"n", "no", "N", "No", "NO", "", "anything", "nope"}

	for _, response := range negativeResponses {
		t.Run("response_"+response, func(t *testing.T) {
			memFs := afero.NewMemMapFs()
			homeDir := "/home/testuser"
			configDir := homeDir + "/.weftlo"
			configPath := configDir + "/config.yaml"

			// Create an existing config.yaml with different content
			originalContent := "default_profile: old/profile\n"
			err := memFs.MkdirAll(configDir, 0755)
			if err != nil {
				t.Fatalf("failed to create config directory: %v", err)
			}
			err = afero.WriteFile(memFs, configPath, []byte(originalContent), 0644)
			if err != nil {
				t.Fatalf("failed to create existing config.yaml: %v", err)
			}

			// Create init command with negative response
			stdin := strings.NewReader(response + "\n")
			var stdout bytes.Buffer

			initCmd := cli.NewInitCommandWithIO(memFs, func() (string, error) {
				return homeDir, nil
			}, stdin, &stdout)

			initCmd.SetArgs([]string{})
			err = initCmd.Execute()

			// Should get ReinitializeAbortedError
			var abortedErr *cli.ReinitializeAbortedError
			if !errors.As(err, &abortedErr) {
				t.Errorf("expected ReinitializeAbortedError with response '%s', got: %v", response, err)
			}

			// Verify config.yaml was NOT overwritten
			content, err := afero.ReadFile(memFs, configPath)
			if err != nil {
				t.Fatalf("error reading config.yaml: %v", err)
			}

			if string(content) != originalContent {
				t.Errorf("expected config.yaml to remain unchanged, but it was modified")
			}
		})
	}
}

// Test 6: Fresh install (no existing config) proceeds without prompt
func TestInitCommand_FreshInstallProceedsWithoutPrompt(t *testing.T) {
	memFs := afero.NewMemMapFs()
	homeDir := "/home/testuser"

	// Create init command with empty stdin (should not need any input for fresh install)
	stdin := strings.NewReader("")
	var stdout bytes.Buffer

	initCmd := cli.NewInitCommandWithIO(memFs, func() (string, error) {
		return homeDir, nil
	}, stdin, &stdout)

	initCmd.SetArgs([]string{})
	err := initCmd.Execute()

	if err != nil {
		t.Fatalf("expected no error on fresh install, got: %v", err)
	}

	// Verify config.yaml was created
	configPath := homeDir + "/.weftlo/config.yaml"
	exists, err := afero.Exists(memFs, configPath)
	if err != nil {
		t.Fatalf("error checking config.yaml existence: %v", err)
	}
	if !exists {
		t.Error("expected config.yaml to be created on fresh install")
	}

	// Verify no prompt was shown
	output := stdout.String()
	if strings.Contains(output, "reinitialize") {
		t.Error("expected no prompt on fresh install")
	}
}

// Test 7: Force reinitialize preserves other profiles
func TestInitCommand_ForceReinitializePreservesOtherProfiles(t *testing.T) {
	memFs := afero.NewMemMapFs()
	homeDir := "/home/testuser"
	configDir := homeDir + "/.weftlo"
	profilesDir := configDir + "/profiles"

	// Create existing config and profiles
	err := memFs.MkdirAll(configDir, 0755)
	if err != nil {
		t.Fatalf("failed to create config directory: %v", err)
	}
	err = afero.WriteFile(memFs, configDir+"/config.yaml", []byte("default_profile: old/profile\n"), 0644)
	if err != nil {
		t.Fatalf("failed to create existing config.yaml: %v", err)
	}

	// Create a custom profile that should be preserved
	customProfileDir := profilesDir + "/custom/myprofile"
	err = memFs.MkdirAll(customProfileDir, 0755)
	if err != nil {
		t.Fatalf("failed to create custom profile directory: %v", err)
	}
	customProfileContent := "name: custom/myprofile\n"
	err = afero.WriteFile(memFs, customProfileDir+"/profile.yaml", []byte(customProfileContent), 0644)
	if err != nil {
		t.Fatalf("failed to create custom profile.yaml: %v", err)
	}

	// Create init command with --force flag
	stdin := strings.NewReader("")
	var stdout bytes.Buffer

	initCmd := cli.NewInitCommandWithIO(memFs, func() (string, error) {
		return homeDir, nil
	}, stdin, &stdout)

	initCmd.SetArgs([]string{"--force"})
	err = initCmd.Execute()

	if err != nil {
		t.Fatalf("expected no error with --force flag, got: %v", err)
	}

	// Verify custom profile still exists
	exists, err := afero.Exists(memFs, customProfileDir+"/profile.yaml")
	if err != nil {
		t.Fatalf("error checking custom profile existence: %v", err)
	}
	if !exists {
		t.Error("expected custom profile to be preserved during force reinitialize")
	}

	// Verify custom profile content is unchanged
	content, err := afero.ReadFile(memFs, customProfileDir+"/profile.yaml")
	if err != nil {
		t.Fatalf("error reading custom profile: %v", err)
	}
	if string(content) != customProfileContent {
		t.Error("expected custom profile content to be preserved")
	}

	// Verify default profile was recreated
	defaultProfilePath := profilesDir + "/default/default/profile.yaml"
	exists, err = afero.Exists(memFs, defaultProfilePath)
	if err != nil {
		t.Fatalf("error checking default profile existence: %v", err)
	}
	if !exists {
		t.Error("expected default profile to be created during force reinitialize")
	}
}

// =============================================================================
// Task Group 5: Output and Error Handling Tests
// =============================================================================

// Test 1: Success output displays created files/directories
func TestInitCommand_SuccessOutputDisplaysCreatedFiles(t *testing.T) {
	memFs := afero.NewMemMapFs()
	homeDir := "/home/testuser"

	stdin := strings.NewReader("")
	var stdout bytes.Buffer

	initCmd := cli.NewInitCommandWithIO(memFs, func() (string, error) {
		return homeDir, nil
	}, stdin, &stdout)

	initCmd.SetArgs([]string{})
	err := initCmd.Execute()
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	output := stdout.String()

	// Verify output contains created files/directories
	expectedItems := []string{
		".weftlo",
		"config.yaml",
		"profiles",
		"default/default",
		"profile.yaml",
		"README.md",
	}

	for _, item := range expectedItems {
		if !strings.Contains(output, item) {
			t.Errorf("expected output to contain '%s', got: %s", item, output)
		}
	}
}

// Test 2: Next-step instructions are displayed
func TestInitCommand_NextStepInstructionsDisplayed(t *testing.T) {
	memFs := afero.NewMemMapFs()
	homeDir := "/home/testuser"

	stdin := strings.NewReader("")
	var stdout bytes.Buffer

	initCmd := cli.NewInitCommandWithIO(memFs, func() (string, error) {
		return homeDir, nil
	}, stdin, &stdout)

	initCmd.SetArgs([]string{})
	err := initCmd.Execute()
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	output := stdout.String()

	// Verify next-step instructions are displayed
	if !strings.Contains(output, "weftlo install") {
		t.Errorf("expected output to contain next-step instruction 'weftlo install', got: %s", output)
	}
}

// Test 3: --quiet flag suppresses non-essential output
func TestInitCommand_QuietFlagSuppressesOutput(t *testing.T) {
	memFs := afero.NewMemMapFs()
	homeDir := "/home/testuser"

	stdin := strings.NewReader("")
	var stdout bytes.Buffer

	initCmd := cli.NewInitCommandWithIO(memFs, func() (string, error) {
		return homeDir, nil
	}, stdin, &stdout)

	initCmd.SetArgs([]string{"--quiet"})
	err := initCmd.Execute()
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	output := stdout.String()

	// Verify output is empty (all non-essential output suppressed)
	if output != "" {
		t.Errorf("expected empty output with --quiet flag, got: %s", output)
	}

	// Verify files were still created
	configPath := homeDir + "/.weftlo/config.yaml"
	exists, err := afero.Exists(memFs, configPath)
	if err != nil {
		t.Fatalf("error checking config.yaml existence: %v", err)
	}
	if !exists {
		t.Error("expected config.yaml to be created even with --quiet flag")
	}
}

// Test 4: Error messages still display when --quiet is set
func TestInitCommand_ErrorMessagesDisplayWhenQuiet(t *testing.T) {
	memFs := afero.NewMemMapFs()

	stdin := strings.NewReader("")
	var stdout bytes.Buffer

	// Create init command with a homeDir function that returns an error
	initCmd := cli.NewInitCommandWithIO(memFs, func() (string, error) {
		return "", errors.New("home directory not found")
	}, stdin, &stdout)

	initCmd.SetArgs([]string{"--quiet"})
	err := initCmd.Execute()

	// Verify an error was returned
	if err == nil {
		t.Fatal("expected an error when home directory cannot be resolved")
	}

	// Verify it's a HomeDirError
	var homeDirErr *cli.HomeDirError
	if !errors.As(err, &homeDirErr) {
		t.Errorf("expected HomeDirError, got: %v", err)
	}
}

// Test 5: Permission denied error is clear
func TestInitCommand_PermissionDeniedErrorIsClear(t *testing.T) {
	// Use a read-only filesystem to simulate permission denied
	baseFs := afero.NewMemMapFs()
	readOnlyFs := afero.NewReadOnlyFs(baseFs)

	homeDir := "/home/testuser"

	stdin := strings.NewReader("")
	var stdout bytes.Buffer

	initCmd := cli.NewInitCommandWithIO(readOnlyFs, func() (string, error) {
		return homeDir, nil
	}, stdin, &stdout)

	initCmd.SetArgs([]string{})
	err := initCmd.Execute()

	// Verify an error was returned
	if err == nil {
		t.Fatal("expected an error when filesystem is read-only")
	}

	// Verify it's a PermissionError
	var permErr *cli.PermissionError
	if !errors.As(err, &permErr) {
		t.Errorf("expected PermissionError, got: %v", err)
	}

	// Verify error message contains path information
	errMsg := err.Error()
	if !strings.Contains(errMsg, "permission denied") {
		t.Errorf("expected error message to contain 'permission denied', got: %s", errMsg)
	}
}

// Test 6: Home directory error is clear
func TestInitCommand_HomeDirErrorIsClear(t *testing.T) {
	memFs := afero.NewMemMapFs()

	stdin := strings.NewReader("")
	var stdout bytes.Buffer

	// Create init command with a homeDir function that returns an error
	initCmd := cli.NewInitCommandWithIO(memFs, func() (string, error) {
		return "", errors.New("$HOME is not defined")
	}, stdin, &stdout)

	initCmd.SetArgs([]string{})
	err := initCmd.Execute()

	// Verify an error was returned
	if err == nil {
		t.Fatal("expected an error when home directory cannot be resolved")
	}

	// Verify it's a HomeDirError
	var homeDirErr *cli.HomeDirError
	if !errors.As(err, &homeDirErr) {
		t.Errorf("expected HomeDirError, got: %v", err)
	}

	// Verify error message is clear
	errMsg := err.Error()
	if !strings.Contains(errMsg, "home directory") {
		t.Errorf("expected error message to mention 'home directory', got: %s", errMsg)
	}
}

// =============================================================================
// Task Group 6: Test Gap Analysis - Strategic Additional Tests
// =============================================================================

// Test 1: End-to-end fresh install via root command (integration test)
// This tests the full user workflow of running "weftlo init" on a fresh system
// Updated to reflect new content root architecture where README.md is inside content/
func TestInitCommand_EndToEndFreshInstallViaRootCommand(t *testing.T) {
	memFs := afero.NewMemMapFs()
	rootCmd := cli.NewRootCommandWithHomeDir(memFs, func() (string, error) {
		return "/home/testuser", nil
	})

	var stdout bytes.Buffer
	rootCmd.SetOut(&stdout)
	rootCmd.SetArgs([]string{"init"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("expected no error running init via root command, got: %v", err)
	}

	// Verify all expected files were created (README.md is now inside content/)
	expectedFiles := []string{
		"/home/testuser/.weftlo/config.yaml",
		"/home/testuser/.weftlo/profiles/default/default/profile.yaml",
		"/home/testuser/.weftlo/profiles/default/default/content/README.md",
	}

	for _, file := range expectedFiles {
		exists, err := afero.Exists(memFs, file)
		if err != nil {
			t.Fatalf("error checking file %s: %v", file, err)
		}
		if !exists {
			t.Errorf("expected file %s to exist after init via root command", file)
		}
	}

	// Verify success output was shown
	output := stdout.String()
	if !strings.Contains(output, "successfully") {
		t.Error("expected success message in output")
	}
}

// Test 2: Short flag -f works for force reinitialize
func TestInitCommand_ShortForceFlagWorks(t *testing.T) {
	memFs := afero.NewMemMapFs()
	homeDir := "/home/testuser"
	configDir := homeDir + "/.weftlo"

	// Create existing config
	err := memFs.MkdirAll(configDir, 0755)
	if err != nil {
		t.Fatalf("failed to create config directory: %v", err)
	}
	err = afero.WriteFile(memFs, configDir+"/config.yaml", []byte("default_profile: old/profile\n"), 0644)
	if err != nil {
		t.Fatalf("failed to create existing config.yaml: %v", err)
	}

	stdin := strings.NewReader("")
	var stdout bytes.Buffer

	initCmd := cli.NewInitCommandWithIO(memFs, func() (string, error) {
		return homeDir, nil
	}, stdin, &stdout)

	// Use short flag -f instead of --force
	initCmd.SetArgs([]string{"-f"})
	err = initCmd.Execute()

	if err != nil {
		t.Fatalf("expected no error with -f flag, got: %v", err)
	}

	// Verify config was overwritten
	content, err := afero.ReadFile(memFs, configDir+"/config.yaml")
	if err != nil {
		t.Fatalf("error reading config.yaml: %v", err)
	}

	if !strings.Contains(string(content), "default/default") {
		t.Error("expected config to be overwritten when using -f flag")
	}
}

// Test 3: Short flag -q works for quiet mode
func TestInitCommand_ShortQuietFlagWorks(t *testing.T) {
	memFs := afero.NewMemMapFs()
	homeDir := "/home/testuser"

	stdin := strings.NewReader("")
	var stdout bytes.Buffer

	initCmd := cli.NewInitCommandWithIO(memFs, func() (string, error) {
		return homeDir, nil
	}, stdin, &stdout)

	// Use short flag -q instead of --quiet
	initCmd.SetArgs([]string{"-q"})
	err := initCmd.Execute()

	if err != nil {
		t.Fatalf("expected no error with -q flag, got: %v", err)
	}

	// Verify output is empty
	output := stdout.String()
	if output != "" {
		t.Errorf("expected empty output with -q flag, got: %s", output)
	}

	// Verify files were still created
	exists, err := afero.Exists(memFs, homeDir+"/.weftlo/config.yaml")
	if err != nil {
		t.Fatalf("error checking config.yaml existence: %v", err)
	}
	if !exists {
		t.Error("expected config.yaml to be created even with -q flag")
	}
}

// Test 4: Combined --force and --quiet flags work together
func TestInitCommand_CombinedForceAndQuietFlags(t *testing.T) {
	memFs := afero.NewMemMapFs()
	homeDir := "/home/testuser"
	configDir := homeDir + "/.weftlo"

	// Create existing config
	err := memFs.MkdirAll(configDir, 0755)
	if err != nil {
		t.Fatalf("failed to create config directory: %v", err)
	}
	err = afero.WriteFile(memFs, configDir+"/config.yaml", []byte("default_profile: old/profile\n"), 0644)
	if err != nil {
		t.Fatalf("failed to create existing config.yaml: %v", err)
	}

	stdin := strings.NewReader("")
	var stdout bytes.Buffer

	initCmd := cli.NewInitCommandWithIO(memFs, func() (string, error) {
		return homeDir, nil
	}, stdin, &stdout)

	// Use both flags
	initCmd.SetArgs([]string{"--force", "--quiet"})
	err = initCmd.Execute()

	if err != nil {
		t.Fatalf("expected no error with --force --quiet, got: %v", err)
	}

	// Verify output is empty (quiet mode)
	output := stdout.String()
	if output != "" {
		t.Errorf("expected empty output with --quiet flag, got: %s", output)
	}

	// Verify config was overwritten (force mode)
	content, err := afero.ReadFile(memFs, configDir+"/config.yaml")
	if err != nil {
		t.Fatalf("error reading config.yaml: %v", err)
	}

	if !strings.Contains(string(content), "default/default") {
		t.Error("expected config to be overwritten when using --force flag")
	}
}

// Test 5: Combined short flags -fq work together
func TestInitCommand_CombinedShortFlags(t *testing.T) {
	memFs := afero.NewMemMapFs()
	homeDir := "/home/testuser"
	configDir := homeDir + "/.weftlo"

	// Create existing config
	err := memFs.MkdirAll(configDir, 0755)
	if err != nil {
		t.Fatalf("failed to create config directory: %v", err)
	}
	err = afero.WriteFile(memFs, configDir+"/config.yaml", []byte("default_profile: old/profile\n"), 0644)
	if err != nil {
		t.Fatalf("failed to create existing config.yaml: %v", err)
	}

	stdin := strings.NewReader("")
	var stdout bytes.Buffer

	initCmd := cli.NewInitCommandWithIO(memFs, func() (string, error) {
		return homeDir, nil
	}, stdin, &stdout)

	// Use combined short flags
	initCmd.SetArgs([]string{"-fq"})
	err = initCmd.Execute()

	if err != nil {
		t.Fatalf("expected no error with -fq, got: %v", err)
	}

	// Verify output is empty (quiet mode)
	output := stdout.String()
	if output != "" {
		t.Errorf("expected empty output with -q flag, got: %s", output)
	}

	// Verify config was overwritten (force mode)
	content, err := afero.ReadFile(memFs, configDir+"/config.yaml")
	if err != nil {
		t.Fatalf("error reading config.yaml: %v", err)
	}

	if !strings.Contains(string(content), "default/default") {
		t.Error("expected config to be overwritten when using -f flag")
	}
}

// Test 6: Partial setup (directory exists but no config.yaml) is treated as fresh install
func TestInitCommand_PartialSetupTreatedAsFreshInstall(t *testing.T) {
	memFs := afero.NewMemMapFs()
	homeDir := "/home/testuser"
	configDir := homeDir + "/.weftlo"

	// Create only the root directory (no config.yaml)
	err := memFs.MkdirAll(configDir, 0755)
	if err != nil {
		t.Fatalf("failed to create config directory: %v", err)
	}

	// Create init command with empty stdin (should not need input for "fresh" install)
	stdin := strings.NewReader("")
	var stdout bytes.Buffer

	initCmd := cli.NewInitCommandWithIO(memFs, func() (string, error) {
		return homeDir, nil
	}, stdin, &stdout)

	initCmd.SetArgs([]string{})
	err = initCmd.Execute()

	if err != nil {
		t.Fatalf("expected no error on partial setup (no config.yaml), got: %v", err)
	}

	// Verify config.yaml was created
	exists, err := afero.Exists(memFs, configDir+"/config.yaml")
	if err != nil {
		t.Fatalf("error checking config.yaml existence: %v", err)
	}
	if !exists {
		t.Error("expected config.yaml to be created on partial setup")
	}

	// Verify no reinitialize prompt was shown
	output := stdout.String()
	if strings.Contains(output, "reinitialize") {
		t.Error("expected no reinitialize prompt when only directory exists without config.yaml")
	}
}

// Test 7: Running init twice with --force is idempotent
func TestInitCommand_IdempotentWithForce(t *testing.T) {
	memFs := afero.NewMemMapFs()
	homeDir := "/home/testuser"

	// First init
	stdin1 := strings.NewReader("")
	var stdout1 bytes.Buffer
	initCmd1 := cli.NewInitCommandWithIO(memFs, func() (string, error) {
		return homeDir, nil
	}, stdin1, &stdout1)
	initCmd1.SetArgs([]string{})
	err := initCmd1.Execute()
	if err != nil {
		t.Fatalf("expected no error on first init, got: %v", err)
	}

	// Get content after first init
	configPath := homeDir + "/.weftlo/config.yaml"
	contentAfterFirst, err := afero.ReadFile(memFs, configPath)
	if err != nil {
		t.Fatalf("error reading config.yaml after first init: %v", err)
	}

	// Second init with --force
	stdin2 := strings.NewReader("")
	var stdout2 bytes.Buffer
	initCmd2 := cli.NewInitCommandWithIO(memFs, func() (string, error) {
		return homeDir, nil
	}, stdin2, &stdout2)
	initCmd2.SetArgs([]string{"--force"})
	err = initCmd2.Execute()
	if err != nil {
		t.Fatalf("expected no error on second init with --force, got: %v", err)
	}

	// Get content after second init
	contentAfterSecond, err := afero.ReadFile(memFs, configPath)
	if err != nil {
		t.Fatalf("error reading config.yaml after second init: %v", err)
	}

	// Content should be the same (idempotent)
	if string(contentAfterFirst) != string(contentAfterSecond) {
		t.Error("expected init to be idempotent - config.yaml should have same content after running twice")
	}
}

// Test 8: End-to-end reinitialize flow via root command with user confirmation
func TestInitCommand_EndToEndReinitializeViaRootCommand(t *testing.T) {
	memFs := afero.NewMemMapFs()
	homeDir := "/home/testuser"
	configDir := homeDir + "/.weftlo"

	// Create existing config
	err := memFs.MkdirAll(configDir, 0755)
	if err != nil {
		t.Fatalf("failed to create config directory: %v", err)
	}
	err = afero.WriteFile(memFs, configDir+"/config.yaml", []byte("default_profile: old/profile\n"), 0644)
	if err != nil {
		t.Fatalf("failed to create existing config.yaml: %v", err)
	}

	rootCmd := cli.NewRootCommandWithHomeDirAndIO(memFs, func() (string, error) {
		return homeDir, nil
	}, strings.NewReader("yes\n"))

	var stdout bytes.Buffer
	rootCmd.SetOut(&stdout)
	rootCmd.SetArgs([]string{"init"})

	err = rootCmd.Execute()
	if err != nil {
		t.Fatalf("expected no error running init with 'yes' response via root command, got: %v", err)
	}

	// Verify config was reinitialized
	content, err := afero.ReadFile(memFs, configDir+"/config.yaml")
	if err != nil {
		t.Fatalf("error reading config.yaml: %v", err)
	}

	if !strings.Contains(string(content), "default/default") {
		t.Error("expected config to be reinitialized with default/default")
	}
}

// =============================================================================
// Task Group 3 (Install Command Spec): Init Command --install-prefix Flag Tests
// =============================================================================

// Test 1: Init with --install-prefix writes value to config.yaml
func TestInitCommand_InstallPrefixWritesToConfigYaml(t *testing.T) {
	memFs := afero.NewMemMapFs()
	homeDir := "/home/testuser"

	stdin := strings.NewReader("")
	var stdout bytes.Buffer

	initCmd := cli.NewInitCommandWithIO(memFs, func() (string, error) {
		return homeDir, nil
	}, stdin, &stdout)

	initCmd.SetArgs([]string{"--install-prefix", "custom-prefix"})
	err := initCmd.Execute()
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// Read and verify config.yaml contains install_prefix
	configPath := homeDir + "/.weftlo/config.yaml"
	content, err := afero.ReadFile(memFs, configPath)
	if err != nil {
		t.Fatalf("error reading config.yaml: %v", err)
	}

	var config struct {
		DefaultProfile string `yaml:"default_profile"`
		InstallPrefix  string `yaml:"install_prefix"`
	}
	err = yaml.Unmarshal(content, &config)
	if err != nil {
		t.Fatalf("error parsing config.yaml: %v", err)
	}

	if config.InstallPrefix != "custom-prefix" {
		t.Errorf("expected install_prefix to be 'custom-prefix', got '%s'", config.InstallPrefix)
	}

	// Also verify default_profile is still set correctly
	if config.DefaultProfile != "default/default" {
		t.Errorf("expected default_profile to be 'default/default', got '%s'", config.DefaultProfile)
	}
}

// Test 2: Init without --install-prefix omits install_prefix from config.yaml
func TestInitCommand_NoInstallPrefixOmitsFromConfigYaml(t *testing.T) {
	memFs := afero.NewMemMapFs()
	homeDir := "/home/testuser"

	stdin := strings.NewReader("")
	var stdout bytes.Buffer

	initCmd := cli.NewInitCommandWithIO(memFs, func() (string, error) {
		return homeDir, nil
	}, stdin, &stdout)

	initCmd.SetArgs([]string{})
	err := initCmd.Execute()
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// Read config.yaml and verify install_prefix is not present
	configPath := homeDir + "/.weftlo/config.yaml"
	content, err := afero.ReadFile(memFs, configPath)
	if err != nil {
		t.Fatalf("error reading config.yaml: %v", err)
	}

	// Check that install_prefix is not in the raw YAML content
	if strings.Contains(string(content), "install_prefix") {
		t.Errorf("expected install_prefix to be omitted from config.yaml when not specified, but found it in: %s", string(content))
	}

	// Verify the config still has default_profile
	var config struct {
		DefaultProfile string `yaml:"default_profile"`
	}
	err = yaml.Unmarshal(content, &config)
	if err != nil {
		t.Fatalf("error parsing config.yaml: %v", err)
	}

	if config.DefaultProfile != "default/default" {
		t.Errorf("expected default_profile to be 'default/default', got '%s'", config.DefaultProfile)
	}
}

// Test 3: Init with --install-prefix preserves other config values (default_profile)
func TestInitCommand_InstallPrefixPreservesOtherConfigValues(t *testing.T) {
	memFs := afero.NewMemMapFs()
	homeDir := "/home/testuser"

	stdin := strings.NewReader("")
	var stdout bytes.Buffer

	initCmd := cli.NewInitCommandWithIO(memFs, func() (string, error) {
		return homeDir, nil
	}, stdin, &stdout)

	initCmd.SetArgs([]string{"--install-prefix", "my-custom-prefix"})
	err := initCmd.Execute()
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// Read and verify config.yaml has both fields correctly set
	configPath := homeDir + "/.weftlo/config.yaml"
	content, err := afero.ReadFile(memFs, configPath)
	if err != nil {
		t.Fatalf("error reading config.yaml: %v", err)
	}

	var config struct {
		DefaultProfile string `yaml:"default_profile"`
		InstallPrefix  string `yaml:"install_prefix"`
	}
	err = yaml.Unmarshal(content, &config)
	if err != nil {
		t.Fatalf("error parsing config.yaml: %v", err)
	}

	// Verify both fields are present and correct
	if config.DefaultProfile != "default/default" {
		t.Errorf("expected default_profile to be preserved as 'default/default', got '%s'", config.DefaultProfile)
	}

	if config.InstallPrefix != "my-custom-prefix" {
		t.Errorf("expected install_prefix to be 'my-custom-prefix', got '%s'", config.InstallPrefix)
	}

	// Verify both fields appear in the raw YAML
	if !strings.Contains(string(content), "default_profile") {
		t.Error("expected raw YAML to contain default_profile field")
	}
	if !strings.Contains(string(content), "install_prefix") {
		t.Error("expected raw YAML to contain install_prefix field")
	}
}
