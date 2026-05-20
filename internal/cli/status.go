package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/spf13/afero"
	"github.com/spf13/cobra"

	"github.com/simensen/weftlo/internal/app/rendering"
	"github.com/simensen/weftlo/internal/app/routing"
	domainconfig "github.com/simensen/weftlo/internal/domain/config"
	"github.com/simensen/weftlo/internal/domain/manifest"
	infraconfig "github.com/simensen/weftlo/internal/infrastructure/config"
	infraprofile "github.com/simensen/weftlo/internal/infrastructure/profile"
)

// StatusCommand represents the status subcommand configuration.
// This type is exported for testing purposes.
type StatusCommand struct {
	// fs is the filesystem for all file operations.
	fs afero.Fs

	// homeDir is the function used to resolve the user's home directory.
	homeDir HomeDirFunc

	// configDirResolver resolves the configuration directory respecting XDG conventions.
	configDirResolver ConfigDirResolver

	// stdin is the input reader for interactive prompts (injectable for testing).
	stdin io.Reader

	// stdout is the output writer for prompts and messages (injectable for testing).
	stdout io.Writer

	// projectDir overrides os.Getwd() for testing purposes.
	// If empty, os.Getwd() is used.
	projectDir string

	// quiet suppresses non-essential output.
	quiet bool

	// verbose enables detailed operational output (reserved for future use).
	verbose bool

	// jsonOutput when true outputs in JSON format.
	jsonOutput bool
}

// StatusResult holds all data gathered for the status command output.
// This struct is populated during the run() method and used for both
// human-readable and JSON output formats.
type StatusResult struct {
	// Profiles contains the profile names from the manifest's Profiles field.
	Profiles []string `json:"profiles"`

	// InheritanceChain contains the full inheritance chain from ProfileNames().
	// This includes all ancestor profiles in merge order (root-to-leaf).
	InheritanceChain []string `json:"inheritance_chain"`

	// InstalledAt is the timestamp from the manifest's GeneratedAt field.
	InstalledAt time.Time `json:"installed_at"`

	// InstallPrefix is the directory prefix from project config's InstallPrefix field.
	InstallPrefix string `json:"install_prefix"`

	// Files contains file paths categorized by their status.
	// Keys match FileStatus string values: unchanged, source_changed, user_modified, conflict, new, removed.
	Files map[string][]string `json:"files"`

	// Warnings contains warning messages (e.g., missing profile).
	Warnings []string `json:"warnings"`
}

// NewStatusCommand creates and returns the status subcommand.
// The filesystem is injected for testability.
// This uses the OS filesystem resolver for XDG Base Directory resolution.
func NewStatusCommand(fs afero.Fs) *cobra.Command {
	// Create a default resolver using the OS filesystem for XDG resolution
	resolver := infraconfig.NewResolver()
	return NewStatusCommandWithResolver(fs, os.UserHomeDir, os.Stdin, os.Stdout, resolver)
}

// NewStatusCommandWithHomeDir creates and returns the status subcommand with an injectable home directory function.
// This is primarily used for testing to provide a custom home directory.
// It uses a fallback resolver that always uses homeDir + "/.weftlo".
func NewStatusCommandWithHomeDir(fs afero.Fs, homeDir HomeDirFunc) *cobra.Command {
	return NewStatusCommandWithIO(fs, homeDir, os.Stdin, os.Stdout)
}

// NewStatusCommandWithIO creates and returns the status subcommand with injectable I/O streams.
// This allows tests to inject custom stdin/stdout for testing interactive prompts.
// It uses a fallback resolver that always uses homeDir + "/.weftlo".
func NewStatusCommandWithIO(fs afero.Fs, homeDir HomeDirFunc, stdin io.Reader, stdout io.Writer) *cobra.Command {
	// Create a simple resolver that uses the provided homeDir for backward compatibility with tests
	resolver := &homeDirResolver{homeDir: homeDir}
	return NewStatusCommandWithResolver(fs, homeDir, stdin, stdout, resolver)
}

// NewStatusCommandWithResolver creates and returns the status subcommand with a custom config directory resolver.
// This is primarily used for testing XDG behavior.
func NewStatusCommandWithResolver(fs afero.Fs, homeDir HomeDirFunc, stdin io.Reader, stdout io.Writer, resolver ConfigDirResolver) *cobra.Command {
	return NewStatusCommandWithProjectDir(fs, homeDir, stdin, stdout, resolver, "")
}

// NewStatusCommandForTesting creates the status subcommand with injectable project directory for testing.
// This allows tests to use an in-memory filesystem by specifying where the project files are.
func NewStatusCommandForTesting(fs afero.Fs, homeDir HomeDirFunc, stdin io.Reader, stdout io.Writer, projectDir string) *cobra.Command {
	resolver := &homeDirResolver{homeDir: homeDir}
	return NewStatusCommandWithProjectDir(fs, homeDir, stdin, stdout, resolver, projectDir)
}

// NewStatusCommandWithProjectDir creates and returns the status subcommand with a custom project directory.
// This is primarily used for testing where os.Getwd() cannot be used with an in-memory filesystem.
func NewStatusCommandWithProjectDir(fs afero.Fs, homeDir HomeDirFunc, stdin io.Reader, stdout io.Writer, resolver ConfigDirResolver, projectDir string) *cobra.Command {
	statusCmd := &StatusCommand{
		fs:                fs,
		homeDir:           homeDir,
		projectDir:        projectDir,
		configDirResolver: resolver,
		stdin:             stdin,
		stdout:            stdout,
	}

	cmd := &cobra.Command{
		Use:   "status",
		Short: "Display installation status and file changes",
		Long: `Display the current installation status, profile information, and file changes.

This command shows:
  - Currently installed profile(s) and inheritance chain
  - Installation timestamp
  - Install prefix
  - File status (unchanged, source_changed, user_modified, conflict, new, removed)

The status command requires an existing installation:
  - .weftlo.yaml must exist (created by 'weftlo install')
  - .weftlo.manifest.json must exist (created by 'weftlo install')`,
		RunE: statusCmd.run,
	}

	// Add --json flag for JSON output
	cmd.Flags().BoolVar(&statusCmd.jsonOutput, "json", false, "Output in JSON format")

	// Add --quiet flag (also inherited from root persistent flags when registered there)
	cmd.Flags().BoolVarP(&statusCmd.quiet, "quiet", "q", false, "Suppress non-essential output")

	// Add --verbose flag (reserved for future use, but defined for flag consistency)
	cmd.Flags().BoolVarP(&statusCmd.verbose, "verbose", "v", false, "Enable verbose output")

	return cmd
}

// run is the RunE function for the status command.
func (c *StatusCommand) run(cmd *cobra.Command, args []string) error {
	// Resolve config directory using XDG conventions
	configDir, err := c.configDirResolver.ResolveConfigDir()
	if err != nil {
		return &HomeDirError{Err: err}
	}

	// Get project directory - use injected value if set, otherwise use current working directory
	projectDir := c.projectDir
	if projectDir == "" {
		var err error
		projectDir, err = os.Getwd()
		if err != nil {
			return &HomeDirError{Err: err}
		}
	}

	// Task 1.5: Check for .weftlo.yaml existence
	projectConfigPath := filepath.Join(projectDir, ".weftlo.yaml")
	projectConfigExists, err := afero.Exists(c.fs, projectConfigPath)
	if err != nil {
		return &PermissionError{Path: projectConfigPath, Err: err}
	}
	if !projectConfigExists {
		return &InstallationNotFoundError{MissingFile: ".weftlo.yaml"}
	}

	// Task 1.5: Check for .weftlo.manifest.json existence
	manifestPath := filepath.Join(projectDir, ".weftlo.manifest.json")
	manifestExists, err := afero.Exists(c.fs, manifestPath)
	if err != nil {
		return &PermissionError{Path: manifestPath, Err: err}
	}
	if !manifestExists {
		return &InstallationNotFoundError{MissingFile: ".weftlo.manifest.json"}
	}

	// Task 2.3: Load project config
	projectConfigLoader := infraconfig.NewProjectConfigLoader(c.fs)
	projectConfig, err := projectConfigLoader.Load(projectDir)
	if err != nil {
		return fmt.Errorf("failed to load project config: %w", err)
	}

	// Task 2.4: Load global config (for install prefix resolution)
	globalConfigLoader := infraconfig.NewGlobalConfigLoader(c.fs)
	globalConfig, _ := globalConfigLoader.Load(configDir) // Ignore error, may not exist

	// Task 2.2: Load manifest from .weftlo.manifest.json
	manifestData, err := afero.ReadFile(c.fs, manifestPath)
	if err != nil {
		return fmt.Errorf("failed to read manifest: %w", err)
	}

	var existingManifest manifest.Manifest
	if err := json.Unmarshal(manifestData, &existingManifest); err != nil {
		return fmt.Errorf("failed to parse manifest: %w", err)
	}

	// Task 2.5: Initialize StatusResult with data from manifest and project config
	result := &StatusResult{
		Profiles:         existingManifest.Profiles,
		InheritanceChain: []string{}, // Will be populated from profile loading
		InstalledAt:      existingManifest.GeneratedAt,
		InstallPrefix:    projectConfig.InstallPrefix,
		Files:            make(map[string][]string),
		Warnings:         []string{},
	}

	// Initialize all file status categories with empty slices
	result.Files["unchanged"] = []string{}
	result.Files["source_changed"] = []string{}
	result.Files["user_modified"] = []string{}
	result.Files["conflict"] = []string{}
	result.Files["new"] = []string{}
	result.Files["removed"] = []string{}

	// Task 2.4: Load profile with graceful degradation
	// Extract profile names from project config (or fall back to manifest profiles)
	profileNames := projectConfig.Profiles
	if len(profileNames) == 0 {
		// Fall back to manifest profiles if project config doesn't specify any
		profileNames = existingManifest.Profiles
	}

	// Track whether profile loading succeeded for change detection
	var mergedProfile *infraprofile.MergedProfile
	profileLoadFailed := false

	if len(profileNames) > 0 {
		// Use infraprofile.NewProfileLoaderWithConfigDir to load profiles
		profileLoader := infraprofile.NewProfileLoaderWithConfigDir(c.fs, configDir)
		var err error
		mergedProfile, err = profileLoader.LoadMergedMultiple(profileNames)
		if err != nil {
			// Check if it's a profile not found error
			var profileNotFoundErr *infraprofile.ProfileNotFoundError
			if errors.As(err, &profileNotFoundErr) {
				// Task 2.4: On ProfileNotFoundError, set warning flag and skip change detection
				warningMsg := fmt.Sprintf("Profile '%s' not found in config directory", profileNotFoundErr.ProfileName)
				result.Warnings = append(result.Warnings, warningMsg)
				// Use manifest profiles as inheritance chain since we can't load the profile
				result.InheritanceChain = existingManifest.Profiles
				profileLoadFailed = true
			} else {
				// For other errors, also add a warning but continue
				warningMsg := fmt.Sprintf("Failed to load profiles: %v", err)
				result.Warnings = append(result.Warnings, warningMsg)
				result.InheritanceChain = existingManifest.Profiles
				profileLoadFailed = true
			}
		} else {
			// Task 2.4: On success, extract inheritance chain via mergedProfile.ProfileNames()
			result.InheritanceChain = mergedProfile.ProfileNames()

			// Display variable conflict warnings (after profile loaded, before output)
			if !c.quiet && !c.jsonOutput {
				displayVariableConflictWarnings(c.stdout, mergedProfile.VariableConflicts())
			}
		}
	} else {
		// No profiles specified - use manifest profiles as fallback
		result.InheritanceChain = existingManifest.Profiles
		profileLoadFailed = true
	}

	// Task 3.2: Integrate change detection
	// Skip change detection if profile loading failed (warning mode)
	if !profileLoadFailed && mergedProfile != nil {
		// Construct Router using same pattern as install/update commands
		contentConfig := mergedProfile.ContentConfig

		// Resolve install prefix - use project config or fall back to global/default
		resolvedInstallPrefix := projectConfig.InstallPrefix
		if resolvedInstallPrefix == "" {
			if globalConfig != nil && globalConfig.InstallPrefix != "" {
				resolvedInstallPrefix = globalConfig.InstallPrefix
			} else {
				resolvedInstallPrefix = domainconfig.DefaultInstallPrefix
			}
		}

		// Apply default target from install prefix resolution if not set in profile
		if contentConfig.DefaultTarget == "" {
			contentConfig.DefaultTarget = resolvedInstallPrefix
		}

		// Get target overrides from project config
		targetOverrides := projectConfig.TargetOverrides

		// Merge variables using DeepMergeVariablesChain with precedence:
		// global (lowest) -> profile inheritance -> project (highest)
		var globalVars map[string]interface{}
		if globalConfig != nil {
			globalVars = globalConfig.Variables
		}
		mergedVariables, _ := rendering.DeepMergeVariablesChain(
			rendering.VariableSource{Name: "global config", Vars: globalVars},
			rendering.VariableSource{Name: "profile inheritance", Vars: mergedProfile.Variables},
			rendering.VariableSource{Name: "project config", Vars: projectConfig.Variables},
		)

		router, err := routing.NewContentRouter(contentConfig, targetOverrides, mergedVariables)
		if err != nil {
			// Add warning but continue
			warningMsg := fmt.Sprintf("Failed to create content router: %v", err)
			result.Warnings = append(result.Warnings, warningMsg)
		} else {
			// Call manifest.DetectChanges with Router
			changeResults, err := manifest.DetectChanges(&existingManifest, mergedProfile, c.fs, projectDir, router)
			if err != nil {
				// Handle error case appropriately - add warning but continue
				warningMsg := fmt.Sprintf("Failed to detect changes: %v", err)
				result.Warnings = append(result.Warnings, warningMsg)
			} else {
				// Task 3.3: Categorize files by status
				// Iterate through DetectChanges() results and group files by their FileStatus value
				for targetPath, info := range changeResults {
					// Convert FileStatus to string key and append to appropriate slice
					statusKey := string(info.Status)
					result.Files[statusKey] = append(result.Files[statusKey], targetPath)
				}
			}
		}
	}

	// Sort file lists for consistent output
	for _, files := range result.Files {
		sort.Strings(files)
	}

	// Task 4.4: Quiet mode handling - suppress all output when quiet is set
	if c.quiet {
		return nil
	}

	// Task 4.3: JSON output
	if c.jsonOutput {
		return c.writeJSONOutput(result)
	}

	// Task 4.2: Human-readable output
	return c.writeHumanReadableOutput(result)
}

// writeJSONOutput writes the status result as formatted JSON to stdout.
// Task 4.3: Implement JSON output
func (c *StatusCommand) writeJSONOutput(result *StatusResult) error {
	jsonData, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to serialize JSON output: %w", err)
	}
	_, _ = c.stdout.Write(jsonData)
	_, _ = io.WriteString(c.stdout, "\n")
	return nil
}

// writeHumanReadableOutput writes the status result in human-readable format.
// Task 4.2: Implement human-readable output
func (c *StatusCommand) writeHumanReadableOutput(result *StatusResult) error {
	// Display "Installation Status" header
	_, _ = io.WriteString(c.stdout, "Installation Status\n")
	_, _ = io.WriteString(c.stdout, "\n")

	// Show profile names with inheritance chain indented below
	// If there are multiple profiles, show the leaf profile first
	if len(result.Profiles) > 0 {
		leafProfile := result.Profiles[len(result.Profiles)-1]
		_, _ = io.WriteString(c.stdout, fmt.Sprintf("Profile: %s\n", leafProfile))

		// Show inheritance chain
		if len(result.InheritanceChain) > 1 {
			// Get ancestors (all but the last one)
			ancestors := result.InheritanceChain[:len(result.InheritanceChain)-1]
			_, _ = io.WriteString(c.stdout, fmt.Sprintf("  Inherits from: %s\n", strings.Join(ancestors, " -> ")))
		} else {
			_, _ = io.WriteString(c.stdout, "  Inherits from: (none)\n")
		}
	}

	// Format timestamp in ISO 8601 / RFC3339 format
	_, _ = io.WriteString(c.stdout, fmt.Sprintf("Installed: %s\n", result.InstalledAt.Format(time.RFC3339)))

	// Show install prefix
	_, _ = io.WriteString(c.stdout, fmt.Sprintf("Install prefix: %s\n", result.InstallPrefix))

	_, _ = io.WriteString(c.stdout, "\n")

	// Display files section
	_, _ = io.WriteString(c.stdout, "Files:\n")
	_, _ = io.WriteString(c.stdout, "\n")

	// Define the order of status categories to display
	statusOrder := []struct {
		key   string
		label string
	}{
		{"unchanged", "Unchanged"},
		{"source_changed", "Source changed"},
		{"user_modified", "User modified"},
		{"conflict", "Conflict"},
		{"new", "New"},
		{"removed", "Removed"},
	}

	// Track if any files were displayed
	hasFiles := false

	// Group files by status with count in parentheses
	for _, status := range statusOrder {
		files := result.Files[status.key]
		if len(files) > 0 {
			hasFiles = true
			_, _ = io.WriteString(c.stdout, fmt.Sprintf("%s (%d):\n", status.label, len(files)))
			// List each file path with dash prefix
			for _, filePath := range files {
				_, _ = io.WriteString(c.stdout, fmt.Sprintf("  - %s\n", filePath))
			}
			_, _ = io.WriteString(c.stdout, "\n")
		}
	}

	if !hasFiles {
		_, _ = io.WriteString(c.stdout, "(No files)\n")
		_, _ = io.WriteString(c.stdout, "\n")
	}

	// Task 4.5: Handle warnings in output
	if len(result.Warnings) > 0 {
		_, _ = io.WriteString(c.stdout, "Warnings:\n")
		for _, warning := range result.Warnings {
			_, _ = io.WriteString(c.stdout, fmt.Sprintf("  - %s\n", warning))
		}
		_, _ = io.WriteString(c.stdout, "\n")
	}

	return nil
}
