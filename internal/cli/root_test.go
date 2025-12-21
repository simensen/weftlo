package cli_test

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/spf13/afero"
	"github.com/spf13/cobra"

	"github.com/simensen/weftlo/internal/cli"
	"github.com/simensen/weftlo/internal/domain/manifest"
)

// =============================================================================
// Task Group 1: Global Flags Infrastructure Tests
// =============================================================================

// Test 1: --verbose flag is accessible from root command
func TestRootCommand_VerboseFlagAccessible(t *testing.T) {
	memFs := afero.NewMemMapFs()
	rootCmd := cli.NewRootCommand(memFs, "dev")

	// Check that --verbose flag exists on root command as persistent flag
	verboseFlag := rootCmd.PersistentFlags().Lookup("verbose")
	if verboseFlag == nil {
		t.Fatal("expected --verbose flag to be registered as persistent flag on root command")
	}

	// Check short flag -v exists
	shortFlag := rootCmd.PersistentFlags().ShorthandLookup("v")
	if shortFlag == nil {
		t.Fatal("expected -v short flag to be registered for --verbose")
	}

	// Verify default value is false
	if verboseFlag.DefValue != "false" {
		t.Errorf("expected --verbose default value to be 'false', got '%s'", verboseFlag.DefValue)
	}
}

// Test 2: --quiet flag is accessible from root command (as persistent flag)
func TestRootCommand_QuietFlagAccessible(t *testing.T) {
	memFs := afero.NewMemMapFs()
	rootCmd := cli.NewRootCommand(memFs, "dev")

	// Check that --quiet flag exists on root command as persistent flag
	quietFlag := rootCmd.PersistentFlags().Lookup("quiet")
	if quietFlag == nil {
		t.Fatal("expected --quiet flag to be registered as persistent flag on root command")
	}

	// Check short flag -q exists
	shortFlag := rootCmd.PersistentFlags().ShorthandLookup("q")
	if shortFlag == nil {
		t.Fatal("expected -q short flag to be registered for --quiet")
	}

	// Verify default value is false
	if quietFlag.DefValue != "false" {
		t.Errorf("expected --quiet default value to be 'false', got '%s'", quietFlag.DefValue)
	}
}

// Test 3: quiet takes precedence over verbose when both set
func TestRootCommand_QuietTakesPrecedenceOverVerbose(t *testing.T) {
	memFs := afero.NewMemMapFs()
	homeDir := "/home/testuser"

	// Setup minimal config structure for install command
	setupMinimalConfig(t, memFs, homeDir)

	rootCmd := cli.NewRootCommandWithHomeDir(memFs, func() (string, error) {
		return homeDir, nil
	})

	var stdout bytes.Buffer
	rootCmd.SetOut(&stdout)
	// Set both --verbose and --quiet, quiet should win (suppress output)
	rootCmd.SetArgs([]string{"--verbose", "--quiet", "init", "--force"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// Quiet mode should suppress output even with verbose set
	output := stdout.String()
	if output != "" {
		t.Errorf("expected empty output when --quiet is set (even with --verbose), got: %s", output)
	}
}

// Test 4: verbose and quiet flags propagate to subcommands
// This test verifies that global flags can be used with subcommands successfully.
func TestRootCommand_FlagsPropagateToSubcommands(t *testing.T) {
	memFs := afero.NewMemMapFs()
	homeDir := "/home/testuser"

	// Test --verbose flag works with init subcommand
	rootCmd := cli.NewRootCommandWithHomeDir(memFs, func() (string, error) {
		return homeDir, nil
	})

	var stdout bytes.Buffer
	rootCmd.SetOut(&stdout)
	// Use --verbose with init subcommand - it should not cause an "unknown flag" error
	rootCmd.SetArgs([]string{"--verbose", "init"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("expected --verbose flag to propagate to init subcommand, got error: %v", err)
	}

	// Also test --quiet flag works with install subcommand
	memFs2 := afero.NewMemMapFs()
	setupMinimalConfig(t, memFs2, homeDir)

	rootCmd2 := cli.NewRootCommandWithHomeDir(memFs2, func() (string, error) {
		return homeDir, nil
	})

	var stdout2 bytes.Buffer
	rootCmd2.SetOut(&stdout2)
	rootCmd2.SetArgs([]string{"--quiet", "init", "--force"})

	err = rootCmd2.Execute()
	if err != nil {
		t.Fatalf("expected --quiet flag to propagate to init subcommand, got error: %v", err)
	}

	// Quiet should suppress output
	if stdout2.String() != "" {
		t.Errorf("expected --quiet to suppress init output, got: %s", stdout2.String())
	}
}

// Test 5: Verify --quiet flag is NOT defined on individual commands (moved to root)
func TestSubcommands_QuietFlagNotDefinedLocally(t *testing.T) {
	memFs := afero.NewMemMapFs()
	rootCmd := cli.NewRootCommand(memFs, "dev")

	// Check each subcommand doesn't have its own --quiet flag definition
	// (it should come from root's persistent flags only)
	subcommands := []string{"init", "install", "update"}

	for _, subcmdName := range subcommands {
		for _, cmd := range rootCmd.Commands() {
			if cmd.Use == subcmdName {
				// The local flags should NOT include --quiet
				localQuietFlag := cmd.LocalFlags().Lookup("quiet")
				if localQuietFlag != nil {
					t.Errorf("expected %s command to NOT have local --quiet flag (should come from root persistent flags)", subcmdName)
				}
				break
			}
		}
	}
}

// Test 6: Global flags work when specified before subcommand
func TestRootCommand_GlobalFlagsBeforeSubcommand(t *testing.T) {
	memFs := afero.NewMemMapFs()
	homeDir := "/home/testuser"

	rootCmd := cli.NewRootCommandWithHomeDir(memFs, func() (string, error) {
		return homeDir, nil
	})

	var stdout bytes.Buffer
	rootCmd.SetOut(&stdout)
	// Specify --quiet before the subcommand
	rootCmd.SetArgs([]string{"--quiet", "init"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// Verify quiet mode worked (no output)
	output := stdout.String()
	if output != "" {
		t.Errorf("expected empty output with --quiet flag before subcommand, got: %s", output)
	}
}

// =============================================================================
// Task Group 2: Version Command Tests
// =============================================================================

// Test 1: `weftlo version` outputs version string
func TestVersionCommand_OutputsVersionString(t *testing.T) {
	memFs := afero.NewMemMapFs()
	rootCmd := cli.NewRootCommand(memFs, "dev")

	var stdout bytes.Buffer
	rootCmd.SetOut(&stdout)
	rootCmd.SetArgs([]string{"version"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	output := stdout.String()
	// Should contain "weftlo version" line
	if !strings.Contains(output, "weftlo version") {
		t.Errorf("expected output to contain 'weftlo version', got: %s", output)
	}
}

// Test 2: Version output includes Go version
func TestVersionCommand_IncludesGoVersion(t *testing.T) {
	memFs := afero.NewMemMapFs()
	rootCmd := cli.NewRootCommand(memFs, "dev")

	var stdout bytes.Buffer
	rootCmd.SetOut(&stdout)
	rootCmd.SetArgs([]string{"version"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	output := stdout.String()
	// Should contain "Go version:" line with the actual Go version
	if !strings.Contains(output, "Go version:") {
		t.Errorf("expected output to contain 'Go version:', got: %s", output)
	}
	// Verify it includes the actual runtime Go version
	if !strings.Contains(output, runtime.Version()) {
		t.Errorf("expected output to contain Go runtime version '%s', got: %s", runtime.Version(), output)
	}
}

// Test 3: Version output includes platform (GOOS/GOARCH)
func TestVersionCommand_IncludesPlatform(t *testing.T) {
	memFs := afero.NewMemMapFs()
	rootCmd := cli.NewRootCommand(memFs, "dev")

	var stdout bytes.Buffer
	rootCmd.SetOut(&stdout)
	rootCmd.SetArgs([]string{"version"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	output := stdout.String()
	// Should contain "Platform:" line
	if !strings.Contains(output, "Platform:") {
		t.Errorf("expected output to contain 'Platform:', got: %s", output)
	}
	// Verify it includes GOOS/GOARCH format
	expectedPlatform := runtime.GOOS + "/" + runtime.GOARCH
	if !strings.Contains(output, expectedPlatform) {
		t.Errorf("expected output to contain platform '%s', got: %s", expectedPlatform, output)
	}
}

// Test 4: Version output format matches expected pattern
func TestVersionCommand_OutputFormatMatchesPattern(t *testing.T) {
	memFs := afero.NewMemMapFs()
	rootCmd := cli.NewRootCommand(memFs, "dev")

	var stdout bytes.Buffer
	rootCmd.SetOut(&stdout)
	rootCmd.SetArgs([]string{"version"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	output := stdout.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")

	// Should have exactly 3 lines
	if len(lines) != 3 {
		t.Errorf("expected 3 lines of output, got %d: %s", len(lines), output)
	}

	// Line 1: "weftlo version X.Y.Z"
	if !strings.HasPrefix(lines[0], "weftlo version ") {
		t.Errorf("expected first line to start with 'weftlo version ', got: %s", lines[0])
	}

	// Line 2: "Go version: go1.XX.X"
	if !strings.HasPrefix(lines[1], "Go version: ") {
		t.Errorf("expected second line to start with 'Go version: ', got: %s", lines[1])
	}

	// Line 3: "Platform: darwin/arm64" (or similar)
	if !strings.HasPrefix(lines[2], "Platform: ") {
		t.Errorf("expected third line to start with 'Platform: ', got: %s", lines[2])
	}
}

// Test 5: Version command shows default "dev" version when not injected via ldflags
func TestVersionCommand_ShowsDefaultDevVersion(t *testing.T) {
	memFs := afero.NewMemMapFs()
	rootCmd := cli.NewRootCommand(memFs, "dev")

	var stdout bytes.Buffer
	rootCmd.SetOut(&stdout)
	rootCmd.SetArgs([]string{"version"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	output := stdout.String()
	// Default version should be "dev" when not injected at build time
	if !strings.Contains(output, "weftlo version dev") {
		t.Errorf("expected output to contain 'weftlo version dev', got: %s", output)
	}
}

// Test 6: Version command has correct help description and example
func TestVersionCommand_HasHelpDescriptionAndExample(t *testing.T) {
	memFs := afero.NewMemMapFs()
	rootCmd := cli.NewRootCommand(memFs, "dev")

	// Find the version subcommand
	var foundVersionCmd bool
	for _, cmd := range rootCmd.Commands() {
		if cmd.Use == "version" {
			foundVersionCmd = true
			// Check the Short description
			if cmd.Short == "" {
				t.Error("expected version command to have a Short description")
			}
			if !strings.Contains(strings.ToLower(cmd.Short), "version") {
				t.Errorf("expected version command Short description to mention 'version', got: %s", cmd.Short)
			}
			// Check the Example field
			if cmd.Example == "" {
				t.Error("expected version command to have Example field set")
			}
			if !strings.Contains(cmd.Example, "weftlo version") {
				t.Errorf("expected version command Example to include 'weftlo version', got: %s", cmd.Example)
			}
			break
		}
	}
	if !foundVersionCmd {
		t.Fatal("expected to find 'version' subcommand")
	}
}

// =============================================================================
// Task Group 3: Shell Completion Command Tests
// =============================================================================

// Test 1: `weftlo completion bash` executes successfully
// Note: Cobra's default completion commands write directly to os.Stdout,
// so we verify successful execution rather than capturing output.
func TestRootCommand_CompletionBashExecutesSuccessfully(t *testing.T) {
	memFs := afero.NewMemMapFs()
	rootCmd := cli.NewRootCommand(memFs, "dev")

	rootCmd.SetArgs([]string{"completion", "bash"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("expected no error running 'completion bash', got: %v", err)
	}
	// Command executed successfully - bash completion script is generated
}

// Test 2: `weftlo completion zsh` executes successfully
func TestRootCommand_CompletionZshExecutesSuccessfully(t *testing.T) {
	memFs := afero.NewMemMapFs()
	rootCmd := cli.NewRootCommand(memFs, "dev")

	rootCmd.SetArgs([]string{"completion", "zsh"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("expected no error running 'completion zsh', got: %v", err)
	}
	// Command executed successfully - zsh completion script is generated
}

// Test 3: `weftlo completion` shows available shells (completion subcommand exists with subcommands)
func TestRootCommand_CompletionCommandExists(t *testing.T) {
	memFs := afero.NewMemMapFs()
	rootCmd := cli.NewRootCommand(memFs, "dev")

	// Find the completion command
	var completionCmd *cobra.Command
	for _, cmd := range rootCmd.Commands() {
		if cmd.Use == "completion" {
			completionCmd = cmd
			break
		}
	}

	if completionCmd == nil {
		t.Fatal("expected 'completion' subcommand to exist on root command")
	}

	// Verify completion has subcommands for different shells
	subcommands := completionCmd.Commands()
	shellNames := make(map[string]bool)
	for _, cmd := range subcommands {
		shellNames[cmd.Use] = true
	}

	// Check that all expected shell completions are available
	expectedShells := []string{"bash", "zsh", "fish", "powershell"}
	for _, shell := range expectedShells {
		if !shellNames[shell] {
			t.Errorf("expected completion subcommand '%s' to exist", shell)
		}
	}
}

// Test 4: `weftlo completion fish` executes successfully
func TestRootCommand_CompletionFishExecutesSuccessfully(t *testing.T) {
	memFs := afero.NewMemMapFs()
	rootCmd := cli.NewRootCommand(memFs, "dev")

	rootCmd.SetArgs([]string{"completion", "fish"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("expected no error running 'completion fish', got: %v", err)
	}
	// Command executed successfully - fish completion script is generated
}

// =============================================================================
// Task Group 4: Help Text Examples Tests
// =============================================================================

// Test 1: Init command --help includes examples
func TestInitCommand_HelpIncludesExamples(t *testing.T) {
	memFs := afero.NewMemMapFs()
	rootCmd := cli.NewRootCommand(memFs, "dev")

	// Find the init subcommand
	var initCmd *cobra.Command
	for _, cmd := range rootCmd.Commands() {
		if cmd.Use == "init" {
			initCmd = cmd
			break
		}
	}

	if initCmd == nil {
		t.Fatal("expected 'init' subcommand to exist on root command")
	}

	// Check that Example field is set
	if initCmd.Example == "" {
		t.Error("expected init command to have Example field set")
	}

	// Verify examples contain expected usage patterns
	if !strings.Contains(initCmd.Example, "weftlo init") {
		t.Errorf("expected init command Example to include 'weftlo init', got: %s", initCmd.Example)
	}

	if !strings.Contains(initCmd.Example, "--install-prefix") {
		t.Errorf("expected init command Example to include '--install-prefix', got: %s", initCmd.Example)
	}

	// Verify 2-space indentation
	if !strings.HasPrefix(initCmd.Example, "  ") {
		t.Errorf("expected init command Example to use 2-space indentation, got: %s", initCmd.Example)
	}
}

// Test 2: Install command --help includes examples
func TestInstallCommand_HelpIncludesExamples(t *testing.T) {
	memFs := afero.NewMemMapFs()
	rootCmd := cli.NewRootCommand(memFs, "dev")

	// Find the install subcommand
	var installCmd *cobra.Command
	for _, cmd := range rootCmd.Commands() {
		if cmd.Use == "install" {
			installCmd = cmd
			break
		}
	}

	if installCmd == nil {
		t.Fatal("expected 'install' subcommand to exist on root command")
	}

	// Check that Example field is set
	if installCmd.Example == "" {
		t.Error("expected install command to have Example field set")
	}

	// Verify examples contain expected usage patterns
	if !strings.Contains(installCmd.Example, "weftlo install") {
		t.Errorf("expected install command Example to include 'weftlo install', got: %s", installCmd.Example)
	}

	if !strings.Contains(installCmd.Example, "--profile") {
		t.Errorf("expected install command Example to include '--profile', got: %s", installCmd.Example)
	}

	if !strings.Contains(installCmd.Example, "--dry-run") {
		t.Errorf("expected install command Example to include '--dry-run', got: %s", installCmd.Example)
	}

	// Verify 2-space indentation
	if !strings.HasPrefix(installCmd.Example, "  ") {
		t.Errorf("expected install command Example to use 2-space indentation, got: %s", installCmd.Example)
	}
}

// Test 3: Update command --help includes examples
func TestUpdateCommand_HelpIncludesExamples(t *testing.T) {
	memFs := afero.NewMemMapFs()
	rootCmd := cli.NewRootCommand(memFs, "dev")

	// Find the update subcommand
	var updateCmd *cobra.Command
	for _, cmd := range rootCmd.Commands() {
		if cmd.Use == "update" {
			updateCmd = cmd
			break
		}
	}

	if updateCmd == nil {
		t.Fatal("expected 'update' subcommand to exist on root command")
	}

	// Check that Example field is set
	if updateCmd.Example == "" {
		t.Error("expected update command to have Example field set")
	}

	// Verify examples contain expected usage patterns
	if !strings.Contains(updateCmd.Example, "weftlo update") {
		t.Errorf("expected update command Example to include 'weftlo update', got: %s", updateCmd.Example)
	}

	if !strings.Contains(updateCmd.Example, "--force") {
		t.Errorf("expected update command Example to include '--force', got: %s", updateCmd.Example)
	}

	if !strings.Contains(updateCmd.Example, "--dry-run") {
		t.Errorf("expected update command Example to include '--dry-run', got: %s", updateCmd.Example)
	}

	// Verify 2-space indentation
	if !strings.HasPrefix(updateCmd.Example, "  ") {
		t.Errorf("expected update command Example to use 2-space indentation, got: %s", updateCmd.Example)
	}
}

// Test 4: Version command --help includes example
func TestVersionCommand_HelpIncludesExample(t *testing.T) {
	memFs := afero.NewMemMapFs()
	rootCmd := cli.NewRootCommand(memFs, "dev")

	// Find the version subcommand
	var versionCmd *cobra.Command
	for _, cmd := range rootCmd.Commands() {
		if cmd.Use == "version" {
			versionCmd = cmd
			break
		}
	}

	if versionCmd == nil {
		t.Fatal("expected 'version' subcommand to exist on root command")
	}

	// Check that Example field is set
	if versionCmd.Example == "" {
		t.Error("expected version command to have Example field set")
	}

	// Verify example contains expected usage pattern
	if !strings.Contains(versionCmd.Example, "weftlo version") {
		t.Errorf("expected version command Example to include 'weftlo version', got: %s", versionCmd.Example)
	}

	// Verify 2-space indentation
	if !strings.HasPrefix(versionCmd.Example, "  ") {
		t.Errorf("expected version command Example to use 2-space indentation, got: %s", versionCmd.Example)
	}
}

// Test 5: Help output shows examples section for init command
func TestInitCommand_HelpOutputShowsExamples(t *testing.T) {
	memFs := afero.NewMemMapFs()
	rootCmd := cli.NewRootCommand(memFs, "dev")

	var stdout bytes.Buffer
	rootCmd.SetOut(&stdout)
	rootCmd.SetArgs([]string{"init", "--help"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	output := stdout.String()
	// Cobra includes "Examples:" section when Example field is set
	if !strings.Contains(output, "Examples:") {
		t.Errorf("expected help output to contain 'Examples:' section, got: %s", output)
	}

	// Verify the actual example appears in help output
	if !strings.Contains(output, "weftlo init") {
		t.Errorf("expected help output to contain example 'weftlo init', got: %s", output)
	}
}

// Test 6: Help output shows examples section for install command
func TestInstallCommand_HelpOutputShowsExamples(t *testing.T) {
	memFs := afero.NewMemMapFs()
	rootCmd := cli.NewRootCommand(memFs, "dev")

	var stdout bytes.Buffer
	rootCmd.SetOut(&stdout)
	rootCmd.SetArgs([]string{"install", "--help"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	output := stdout.String()
	// Cobra includes "Examples:" section when Example field is set
	if !strings.Contains(output, "Examples:") {
		t.Errorf("expected help output to contain 'Examples:' section, got: %s", output)
	}

	// Verify the actual examples appear in help output
	if !strings.Contains(output, "weftlo install") {
		t.Errorf("expected help output to contain example 'weftlo install', got: %s", output)
	}
}

// =============================================================================
// Task Group 5: Verbose Output Implementation Tests
// =============================================================================

// Test 1: Init command verbose output shows config directory resolution
func TestInitCommand_VerboseShowsConfigDirResolution(t *testing.T) {
	memFs := afero.NewMemMapFs()
	homeDir := "/home/testuser"

	rootCmd := cli.NewRootCommandWithHomeDir(memFs, func() (string, error) {
		return homeDir, nil
	})

	var stdout bytes.Buffer
	rootCmd.SetOut(&stdout)
	rootCmd.SetArgs([]string{"--verbose", "init"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	output := stdout.String()
	// Verbose output should show config directory resolution
	if !strings.Contains(output, "Config directory:") {
		t.Errorf("expected verbose output to contain 'Config directory:', got: %s", output)
	}
	// Should include the config path
	if !strings.Contains(output, ".weftlo") {
		t.Errorf("expected verbose output to contain config directory path, got: %s", output)
	}
}

// Test 2: Install command verbose output shows profile resolution source
func TestInstallCommand_VerboseShowsProfileResolution(t *testing.T) {
	memFs := afero.NewMemMapFs()
	homeDir := "/home/testuser"

	// Setup config and profile
	setupMinimalConfigWithProfile(t, memFs, homeDir, "default/default")

	// Save and restore working directory
	originalWd, _ := os.Getwd()
	defer func() { _ = os.Chdir(originalWd) }()

	tempDir := t.TempDir()
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("failed to change to temp directory: %v", err)
	}

	rootCmd := cli.NewRootCommandWithHomeDir(memFs, func() (string, error) {
		return homeDir, nil
	})

	var stdout bytes.Buffer
	rootCmd.SetOut(&stdout)
	rootCmd.SetArgs([]string{"--verbose", "install", "--profile", "default/default"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	output := stdout.String()
	// Verbose output should show profile resolution source
	if !strings.Contains(output, "Using profile from:") {
		t.Errorf("expected verbose output to contain 'Using profile from:', got: %s", output)
	}
	// When using --profile flag, should indicate flag
	if !strings.Contains(output, "--profile flag") {
		t.Errorf("expected verbose output to indicate '--profile flag' as source, got: %s", output)
	}
}

// Test 3: Install command verbose output shows file rendering details
func TestInstallCommand_VerboseShowsFileRendering(t *testing.T) {
	memFs := afero.NewMemMapFs()
	homeDir := "/home/testuser"

	// Setup config and profile with a template
	setupMinimalConfigWithProfile(t, memFs, homeDir, "default/default")
	// Add a template file to the profile's content directory
	contentDir := filepath.Join(homeDir, ".weftlo", "profiles", "default", "default", "content")
	if err := memFs.MkdirAll(contentDir, 0755); err != nil {
		t.Fatalf("failed to create content directory: %v", err)
	}
	templatePath := filepath.Join(contentDir, "README.md.tmpl")
	if err := afero.WriteFile(memFs, templatePath, []byte("# Test"), 0644); err != nil {
		t.Fatalf("failed to write template: %v", err)
	}

	// Save and restore working directory
	originalWd, _ := os.Getwd()
	defer func() { _ = os.Chdir(originalWd) }()

	tempDir := t.TempDir()
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("failed to change to temp directory: %v", err)
	}

	rootCmd := cli.NewRootCommandWithHomeDir(memFs, func() (string, error) {
		return homeDir, nil
	})

	var stdout bytes.Buffer
	rootCmd.SetOut(&stdout)
	rootCmd.SetArgs([]string{"--verbose", "install"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	output := stdout.String()
	// Verbose output should show file rendering details
	if !strings.Contains(output, "Rendering:") {
		t.Errorf("expected verbose output to contain 'Rendering:', got: %s", output)
	}
}

// Test 4: Update command verbose output shows change detection details
func TestUpdateCommand_VerboseShowsChangeDetection(t *testing.T) {
	memFs := afero.NewMemMapFs()
	homeDir := "/home/testuser"
	projectDir := "/projects/myproject"

	// Setup full environment for update command
	setupUpdateTestEnvironment(t, memFs, homeDir, projectDir)

	var stdout bytes.Buffer
	cmd := cli.NewUpdateCommandForTesting(memFs, func() (string, error) {
		return homeDir, nil
	}, strings.NewReader(""), &stdout, projectDir)
	cmd.SetArgs([]string{"--verbose"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	output := stdout.String()
	// Verbose output should show change detection details
	if !strings.Contains(output, "Checking file:") {
		t.Errorf("expected verbose output to contain 'Checking file:', got: %s", output)
	}
}

// Test 5: Verbose output is suppressed when --quiet is set
func TestVerboseOutput_SuppressedByQuiet(t *testing.T) {
	memFs := afero.NewMemMapFs()
	homeDir := "/home/testuser"

	// Setup minimal config
	setupMinimalConfig(t, memFs, homeDir)

	rootCmd := cli.NewRootCommandWithHomeDir(memFs, func() (string, error) {
		return homeDir, nil
	})

	var stdout bytes.Buffer
	rootCmd.SetOut(&stdout)
	// Set both --verbose and --quiet - quiet should win
	rootCmd.SetArgs([]string{"--verbose", "--quiet", "init", "--force"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	output := stdout.String()
	// With quiet mode, even verbose output should be suppressed
	if output != "" {
		t.Errorf("expected empty output when --quiet is set (even with --verbose), got: %s", output)
	}
}

// Test 6: Verbose output does not appear without --verbose flag
func TestVerboseOutput_NotShownWithoutFlag(t *testing.T) {
	memFs := afero.NewMemMapFs()
	homeDir := "/home/testuser"

	rootCmd := cli.NewRootCommandWithHomeDir(memFs, func() (string, error) {
		return homeDir, nil
	})

	var stdout bytes.Buffer
	rootCmd.SetOut(&stdout)
	// Run init without --verbose
	rootCmd.SetArgs([]string{"init"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	output := stdout.String()
	// Without --verbose, should NOT contain verbose-specific output
	if strings.Contains(output, "Config directory:") {
		t.Errorf("expected output without --verbose to NOT contain 'Config directory:', got: %s", output)
	}
}

// Test 7: Install command verbose output shows install prefix resolution
func TestInstallCommand_VerboseShowsInstallPrefixResolution(t *testing.T) {
	memFs := afero.NewMemMapFs()
	homeDir := "/home/testuser"

	// Setup config and profile
	setupMinimalConfigWithProfile(t, memFs, homeDir, "default/default")

	// Save and restore working directory
	originalWd, _ := os.Getwd()
	defer func() { _ = os.Chdir(originalWd) }()

	tempDir := t.TempDir()
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("failed to change to temp directory: %v", err)
	}

	rootCmd := cli.NewRootCommandWithHomeDir(memFs, func() (string, error) {
		return homeDir, nil
	})

	var stdout bytes.Buffer
	rootCmd.SetOut(&stdout)
	rootCmd.SetArgs([]string{"--verbose", "install", "--install-prefix", "custom-prefix"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	output := stdout.String()
	// Verbose output should show install prefix resolution
	if !strings.Contains(output, "Using install prefix from:") {
		t.Errorf("expected verbose output to contain 'Using install prefix from:', got: %s", output)
	}
}

// Test 8: Update command verbose output shows manifest regeneration
func TestUpdateCommand_VerboseShowsManifestRegeneration(t *testing.T) {
	memFs := afero.NewMemMapFs()
	homeDir := "/home/testuser"
	projectDir := "/projects/myproject"

	// Setup environment with a file that needs update
	setupUpdateTestEnvironmentWithChange(t, memFs, homeDir, projectDir)

	var stdout bytes.Buffer
	cmd := cli.NewUpdateCommandForTesting(memFs, func() (string, error) {
		return homeDir, nil
	}, strings.NewReader(""), &stdout, projectDir)
	cmd.SetArgs([]string{"--verbose"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	output := stdout.String()
	// Verbose output should show manifest regeneration
	if !strings.Contains(output, "Regenerating manifest") {
		t.Errorf("expected verbose output to contain 'Regenerating manifest', got: %s", output)
	}
}

// =============================================================================
// Task Group 6: Test Review and Gap Analysis - Strategic Tests
// =============================================================================

// Strategic Test 1: Version command with custom injected version (not just "dev")
func TestVersionCommand_WithCustomInjectedVersion(t *testing.T) {
	memFs := afero.NewMemMapFs()
	// Simulate a version injected via ldflags
	customVersion := "1.2.3"
	rootCmd := cli.NewRootCommand(memFs, customVersion)

	var stdout bytes.Buffer
	rootCmd.SetOut(&stdout)
	rootCmd.SetArgs([]string{"version"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	output := stdout.String()
	// Should contain the custom version
	expectedOutput := "weftlo version " + customVersion
	if !strings.Contains(output, expectedOutput) {
		t.Errorf("expected output to contain '%s', got: %s", expectedOutput, output)
	}
}

// Strategic Test 2: PowerShell completion executes successfully
func TestRootCommand_CompletionPowerShellExecutesSuccessfully(t *testing.T) {
	memFs := afero.NewMemMapFs()
	rootCmd := cli.NewRootCommand(memFs, "dev")

	rootCmd.SetArgs([]string{"completion", "powershell"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("expected no error running 'completion powershell', got: %v", err)
	}
	// Command executed successfully - PowerShell completion script is generated
}

// Strategic Test 3: Short flags -v and -q work as global flags
func TestRootCommand_ShortFlagsWorkAsGlobalFlags(t *testing.T) {
	memFs := afero.NewMemMapFs()
	homeDir := "/home/testuser"

	// Test -v (verbose) short flag
	rootCmd := cli.NewRootCommandWithHomeDir(memFs, func() (string, error) {
		return homeDir, nil
	})

	var stdout bytes.Buffer
	rootCmd.SetOut(&stdout)
	rootCmd.SetArgs([]string{"-v", "init"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("expected -v short flag to work, got error: %v", err)
	}

	output := stdout.String()
	// -v should enable verbose output
	if !strings.Contains(output, "Config directory:") {
		t.Errorf("expected -v to enable verbose output with 'Config directory:', got: %s", output)
	}

	// Test -q (quiet) short flag
	memFs2 := afero.NewMemMapFs()
	setupMinimalConfig(t, memFs2, homeDir)

	rootCmd2 := cli.NewRootCommandWithHomeDir(memFs2, func() (string, error) {
		return homeDir, nil
	})

	var stdout2 bytes.Buffer
	rootCmd2.SetOut(&stdout2)
	rootCmd2.SetArgs([]string{"-q", "init", "--force"})

	err = rootCmd2.Execute()
	if err != nil {
		t.Fatalf("expected -q short flag to work, got error: %v", err)
	}

	// -q should suppress output
	if stdout2.String() != "" {
		t.Errorf("expected -q to suppress output, got: %s", stdout2.String())
	}
}

// Strategic Test 4: Update command help output shows examples section
func TestUpdateCommand_HelpOutputShowsExamples(t *testing.T) {
	memFs := afero.NewMemMapFs()
	rootCmd := cli.NewRootCommand(memFs, "dev")

	var stdout bytes.Buffer
	rootCmd.SetOut(&stdout)
	rootCmd.SetArgs([]string{"update", "--help"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	output := stdout.String()
	// Cobra includes "Examples:" section when Example field is set
	if !strings.Contains(output, "Examples:") {
		t.Errorf("expected help output to contain 'Examples:' section, got: %s", output)
	}

	// Verify the actual examples appear in help output
	if !strings.Contains(output, "weftlo update") {
		t.Errorf("expected help output to contain example 'weftlo update', got: %s", output)
	}
}

// Strategic Test 5: Version command help output shows examples section
func TestVersionCommand_HelpOutputShowsExamples(t *testing.T) {
	memFs := afero.NewMemMapFs()
	rootCmd := cli.NewRootCommand(memFs, "dev")

	var stdout bytes.Buffer
	rootCmd.SetOut(&stdout)
	rootCmd.SetArgs([]string{"version", "--help"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	output := stdout.String()
	// Cobra includes "Examples:" section when Example field is set
	if !strings.Contains(output, "Examples:") {
		t.Errorf("expected help output to contain 'Examples:' section, got: %s", output)
	}

	// Verify the actual example appears in help output
	if !strings.Contains(output, "weftlo version") {
		t.Errorf("expected help output to contain example 'weftlo version', got: %s", output)
	}
}

// Strategic Test 6: Conflicting flags order - quiet before verbose still works
func TestRootCommand_QuietBeforeVerboseStillSuppresses(t *testing.T) {
	memFs := afero.NewMemMapFs()
	homeDir := "/home/testuser"

	setupMinimalConfig(t, memFs, homeDir)

	rootCmd := cli.NewRootCommandWithHomeDir(memFs, func() (string, error) {
		return homeDir, nil
	})

	var stdout bytes.Buffer
	rootCmd.SetOut(&stdout)
	// Quiet specified BEFORE verbose - quiet should still take precedence
	rootCmd.SetArgs([]string{"--quiet", "--verbose", "init", "--force"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	output := stdout.String()
	// Quiet mode should suppress output regardless of order
	if output != "" {
		t.Errorf("expected empty output when --quiet is set before --verbose, got: %s", output)
	}
}

// Strategic Test 7: Global flags work with status command
func TestRootCommand_GlobalFlagsWorkWithStatusCommand(t *testing.T) {
	memFs := afero.NewMemMapFs()
	homeDir := "/home/testuser"
	projectDir := "/projects/myproject"

	// Setup environment for status command
	setupUpdateTestEnvironment(t, memFs, homeDir, projectDir)

	var stdout bytes.Buffer
	cmd := cli.NewStatusCommandForTesting(memFs, func() (string, error) {
		return homeDir, nil
	}, strings.NewReader(""), &stdout, projectDir)
	cmd.SetArgs([]string{"--quiet"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// Quiet should suppress status output
	if stdout.String() != "" {
		t.Errorf("expected --quiet to suppress status output, got: %s", stdout.String())
	}
}

// Strategic Test 8: Verbose flag NOT defined locally on subcommands
func TestSubcommands_VerboseFlagNotDefinedLocally(t *testing.T) {
	memFs := afero.NewMemMapFs()
	rootCmd := cli.NewRootCommand(memFs, "dev")

	// Check each subcommand doesn't have its own --verbose flag definition
	// (it should come from root's persistent flags only)
	subcommands := []string{"init", "install", "update", "status"}

	for _, subcmdName := range subcommands {
		for _, cmd := range rootCmd.Commands() {
			if cmd.Use == subcmdName {
				// The local flags should NOT include --verbose
				localVerboseFlag := cmd.LocalFlags().Lookup("verbose")
				if localVerboseFlag != nil {
					t.Errorf("expected %s command to NOT have local --verbose flag (should come from root persistent flags)", subcmdName)
				}
				break
			}
		}
	}
}

// Strategic Test 9: Verbose with dry-run shows verbose output without making changes
func TestUpdateCommand_VerboseWithDryRunShowsDetailsWithoutChanges(t *testing.T) {
	memFs := afero.NewMemMapFs()
	homeDir := "/home/testuser"
	projectDir := "/projects/myproject"

	// Setup environment with a file that needs update
	setupUpdateTestEnvironmentWithChange(t, memFs, homeDir, projectDir)

	var stdout bytes.Buffer
	cmd := cli.NewUpdateCommandForTesting(memFs, func() (string, error) {
		return homeDir, nil
	}, strings.NewReader(""), &stdout, projectDir)
	cmd.SetArgs([]string{"--verbose", "--dry-run"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	output := stdout.String()
	// Should show verbose change detection
	if !strings.Contains(output, "Checking file:") {
		t.Errorf("expected verbose output to contain 'Checking file:', got: %s", output)
	}
	// Should indicate dry-run mode
	if !strings.Contains(output, "dry-run") {
		t.Errorf("expected output to indicate dry-run mode, got: %s", output)
	}
}

// Strategic Test 10: Version with semantic version format
func TestVersionCommand_WithSemanticVersionFormat(t *testing.T) {
	memFs := afero.NewMemMapFs()
	// Test with a full semantic version including pre-release
	semVer := "2.0.0-beta.1"
	rootCmd := cli.NewRootCommand(memFs, semVer)

	var stdout bytes.Buffer
	rootCmd.SetOut(&stdout)
	rootCmd.SetArgs([]string{"version"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	output := stdout.String()
	// Should contain the full semantic version
	if !strings.Contains(output, "weftlo version "+semVer) {
		t.Errorf("expected output to contain semantic version '%s', got: %s", semVer, output)
	}
}

// =============================================================================
// Helper Functions
// =============================================================================

// setupMinimalConfig creates a minimal global configuration for testing global flags.
func setupMinimalConfig(t *testing.T, fs afero.Fs, homeDir string) {
	t.Helper()

	configDir := homeDir + "/.weftlo"
	if err := fs.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("failed to create config directory: %v", err)
	}

	// Create config.yaml
	configContent := "default_profile: default/default\n"
	if err := afero.WriteFile(fs, configDir+"/config.yaml", []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config.yaml: %v", err)
	}
}

// setupMinimalConfigWithProfile creates a config and a minimal profile for testing.
func setupMinimalConfigWithProfile(t *testing.T, fs afero.Fs, homeDir string, profileName string) {
	t.Helper()

	// Create config directory
	configDir := homeDir + "/.weftlo"
	if err := fs.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("failed to create config directory: %v", err)
	}

	// Create config.yaml
	configContent := "default_profile: " + profileName + "\n"
	if err := afero.WriteFile(fs, configDir+"/config.yaml", []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config.yaml: %v", err)
	}

	// Parse profile name and create profile directory with content subdirectory
	parts := strings.Split(profileName, "/")
	if len(parts) != 2 {
		t.Fatalf("invalid profile name: %s", profileName)
	}
	profileDir := filepath.Join(configDir, "profiles", parts[0], parts[1])
	contentDir := filepath.Join(profileDir, "content")
	if err := fs.MkdirAll(contentDir, 0755); err != nil {
		t.Fatalf("failed to create profile content directory: %v", err)
	}

	// Create profile.yaml with content configuration
	profileContent := "name: " + profileName + "\ncontent:\n  root: content\n  default_target: \"\"\n"
	if err := afero.WriteFile(fs, filepath.Join(profileDir, "profile.yaml"), []byte(profileContent), 0644); err != nil {
		t.Fatalf("failed to write profile.yaml: %v", err)
	}
}

// setupUpdateTestEnvironment creates a complete environment for testing the update command.
func setupUpdateTestEnvironment(t *testing.T, fs afero.Fs, homeDir, projectDir string) {
	t.Helper()

	// Create project directory
	if err := fs.MkdirAll(projectDir, 0755); err != nil {
		t.Fatalf("failed to create project directory: %v", err)
	}

	// Setup config and profile
	setupMinimalConfigWithProfile(t, fs, homeDir, "default/default")

	// Add a template file to the profile's content directory
	contentDir := filepath.Join(homeDir, ".weftlo", "profiles", "default", "default", "content")
	if err := fs.MkdirAll(contentDir, 0755); err != nil {
		t.Fatalf("failed to create content directory: %v", err)
	}
	templatePath := filepath.Join(contentDir, "README.md")
	templateContent := "# Test README"
	if err := afero.WriteFile(fs, templatePath, []byte(templateContent), 0644); err != nil {
		t.Fatalf("failed to write template: %v", err)
	}

	// Create project config
	projectConfigContent := "profiles:\n  - default/default\ninstall_prefix: \"\"\n"
	if err := afero.WriteFile(fs, filepath.Join(projectDir, ".weftlo.yaml"), []byte(projectConfigContent), 0644); err != nil {
		t.Fatalf("failed to write project config: %v", err)
	}

	// Create manifest with matching checksum
	checksum := computeTestChecksum(templateContent)
	m := manifest.Manifest{
		Version:     "1.0.0",
		Profiles:    []string{"default/default"},
		GeneratedAt: time.Now().UTC(),
		Files: map[string]manifest.ManifestFile{
			"README.md": {
				SourceChecksum: checksum,
				OutputChecksum: checksum,
			},
		},
	}
	manifestData, _ := json.MarshalIndent(m, "", "  ")
	if err := afero.WriteFile(fs, filepath.Join(projectDir, ".weftlo.manifest.json"), manifestData, 0644); err != nil {
		t.Fatalf("failed to write manifest: %v", err)
	}

	// Create the installed file
	if err := afero.WriteFile(fs, filepath.Join(projectDir, "README.md"), []byte(templateContent), 0644); err != nil {
		t.Fatalf("failed to write installed file: %v", err)
	}
}

// setupUpdateTestEnvironmentWithChange creates an environment where an update is needed.
func setupUpdateTestEnvironmentWithChange(t *testing.T, fs afero.Fs, homeDir, projectDir string) {
	t.Helper()

	// Create project directory
	if err := fs.MkdirAll(projectDir, 0755); err != nil {
		t.Fatalf("failed to create project directory: %v", err)
	}

	// Setup config and profile
	setupMinimalConfigWithProfile(t, fs, homeDir, "default/default")

	// Add a template file to the profile's content directory with new content
	contentDir := filepath.Join(homeDir, ".weftlo", "profiles", "default", "default", "content")
	if err := fs.MkdirAll(contentDir, 0755); err != nil {
		t.Fatalf("failed to create content directory: %v", err)
	}
	templatePath := filepath.Join(contentDir, "README.md")
	newTemplateContent := "# Updated README"
	if err := afero.WriteFile(fs, templatePath, []byte(newTemplateContent), 0644); err != nil {
		t.Fatalf("failed to write template: %v", err)
	}

	// Create project config
	projectConfigContent := "profiles:\n  - default/default\ninstall_prefix: \"\"\n"
	if err := afero.WriteFile(fs, filepath.Join(projectDir, ".weftlo.yaml"), []byte(projectConfigContent), 0644); err != nil {
		t.Fatalf("failed to write project config: %v", err)
	}

	// Create manifest with OLD checksum (different from current template)
	oldContent := "# Old README"
	oldChecksum := computeTestChecksum(oldContent)
	m := manifest.Manifest{
		Version:     "1.0.0",
		Profiles:    []string{"default/default"},
		GeneratedAt: time.Now().UTC(),
		Files: map[string]manifest.ManifestFile{
			"README.md": {
				SourceChecksum: oldChecksum,
				OutputChecksum: oldChecksum,
			},
		},
	}
	manifestData, _ := json.MarshalIndent(m, "", "  ")
	if err := afero.WriteFile(fs, filepath.Join(projectDir, ".weftlo.manifest.json"), manifestData, 0644); err != nil {
		t.Fatalf("failed to write manifest: %v", err)
	}

	// Create the installed file with old content
	if err := afero.WriteFile(fs, filepath.Join(projectDir, "README.md"), []byte(oldContent), 0644); err != nil {
		t.Fatalf("failed to write installed file: %v", err)
	}
}

// computeTestChecksum calculates SHA256 checksum of content matching manifest format.
func computeTestChecksum(content string) string {
	hash := sha256.Sum256([]byte(content))
	return "sha256:" + hex.EncodeToString(hash[:])
}
