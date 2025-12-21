package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"

	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	infraconfig "github.com/simensen/weftlo/internal/infrastructure/config"
)

// ProfileListCommand represents the profile list subcommand configuration.
// This type is exported for testing purposes.
type ProfileListCommand struct {
	// fs is the filesystem for all file operations.
	fs afero.Fs

	// configDirResolver resolves the configuration directory respecting XDG conventions.
	configDirResolver ConfigDirResolver

	// stdout is the output writer for prompts and messages (injectable for testing).
	stdout io.Writer

	// verbose enables detailed operational output.
	verbose bool

	// quiet suppresses non-essential output.
	quiet bool

	// jsonOutput when true outputs in JSON format.
	jsonOutput bool
}

// ProfileListResult holds all data gathered for the profile list command output.
// This struct is used for both human-readable and JSON output formats.
type ProfileListResult struct {
	// Profiles contains the list of available profile names.
	Profiles []ProfileInfo `json:"profiles"`
	// BrokenProfiles contains the list of broken profiles (only used in verbose mode).
	// Not included in JSON output.
	BrokenProfiles []BrokenProfile `json:"-"`
}

// ProfileInfo represents information about a single profile.
type ProfileInfo struct {
	// Name is the full profile name in vendor/name format.
	Name string `json:"name"`
	// InheritsFrom is the optional parent profile name for inheritance.
	// Only populated in verbose mode.
	InheritsFrom string `json:"inherits_from,omitempty"`
}

// BrokenProfile represents a profile that exists but is invalid.
type BrokenProfile struct {
	// Name is the full profile name in vendor/name format.
	Name string `json:"name"`
	// Reason describes why the profile is broken.
	// Examples: "missing profile.yaml", "invalid YAML syntax"
	Reason string `json:"reason"`
}

// ProfileEnumerator handles enumeration of profiles from the filesystem.
type ProfileEnumerator struct {
	// fs is the filesystem for all file operations.
	fs afero.Fs
	// configDir is the configuration directory path.
	configDir string
}

// profileYAMLConfig represents the structure of profile.yaml for parsing.
// This mirrors the ProfileConfig from domain/profile but is local to avoid
// unnecessary dependencies in the CLI layer.
type profileYAMLConfig struct {
	// Name is the profile identifier.
	Name string `yaml:"name"`
	// InheritsFrom is the optional parent profile name for inheritance.
	InheritsFrom string `yaml:"inherits_from,omitempty"`
}

// NewProfileEnumerator creates a new ProfileEnumerator with the given filesystem and config directory.
func NewProfileEnumerator(fs afero.Fs, configDir string) *ProfileEnumerator {
	return &ProfileEnumerator{
		fs:        fs,
		configDir: configDir,
	}
}

// EnumerateProfiles walks the profiles directory and returns lists of valid and broken profiles.
// Valid profiles have a profile.yaml file, broken profiles have a directory but no profile.yaml.
// Both lists are sorted alphabetically by full name (vendor/name format).
// This method does not parse profile.yaml contents - use EnumerateProfilesVerbose for that.
func (e *ProfileEnumerator) EnumerateProfiles() ([]ProfileInfo, []BrokenProfile, error) {
	profilesDir := filepath.Join(e.configDir, "profiles")

	// Check if profiles directory exists
	exists, err := afero.DirExists(e.fs, profilesDir)
	if err != nil {
		return nil, nil, err
	}
	if !exists {
		// No profiles directory means no profiles
		return []ProfileInfo{}, []BrokenProfile{}, nil
	}

	// Read vendor directories
	vendorDirs, err := afero.ReadDir(e.fs, profilesDir)
	if err != nil {
		return nil, nil, err
	}

	var validProfiles []ProfileInfo
	var brokenProfiles []BrokenProfile

	// Walk two levels: vendor/name
	for _, vendorDir := range vendorDirs {
		if !vendorDir.IsDir() {
			continue
		}

		vendorName := vendorDir.Name()
		vendorPath := filepath.Join(profilesDir, vendorName)

		// Read profile directories within vendor
		profileDirs, err := afero.ReadDir(e.fs, vendorPath)
		if err != nil {
			return nil, nil, err
		}

		for _, profileDir := range profileDirs {
			if !profileDir.IsDir() {
				continue
			}

			profileName := profileDir.Name()
			fullProfileName := vendorName + "/" + profileName
			profilePath := filepath.Join(vendorPath, profileName)

			// Check for profile.yaml existence
			profileYAMLPath := filepath.Join(profilePath, "profile.yaml")
			profileYAMLExists, err := afero.Exists(e.fs, profileYAMLPath)
			if err != nil {
				return nil, nil, err
			}

			if profileYAMLExists {
				// Valid profile
				validProfiles = append(validProfiles, ProfileInfo{
					Name: fullProfileName,
				})
			} else {
				// Broken profile (directory exists but no profile.yaml)
				brokenProfiles = append(brokenProfiles, BrokenProfile{
					Name:   fullProfileName,
					Reason: "missing profile.yaml",
				})
			}
		}
	}

	// Sort profiles alphabetically by full name
	sort.Slice(validProfiles, func(i, j int) bool {
		return validProfiles[i].Name < validProfiles[j].Name
	})
	sort.Slice(brokenProfiles, func(i, j int) bool {
		return brokenProfiles[i].Name < brokenProfiles[j].Name
	})

	return validProfiles, brokenProfiles, nil
}

// EnumerateProfilesVerbose walks the profiles directory and returns lists of valid and broken profiles
// with full profile.yaml parsing. This is used for verbose mode output.
// Valid profiles have a valid profile.yaml file with parsed inherits_from field.
// Broken profiles include those with missing profile.yaml or invalid YAML syntax.
// Both lists are sorted alphabetically by full name (vendor/name format).
func (e *ProfileEnumerator) EnumerateProfilesVerbose() ([]ProfileInfo, []BrokenProfile, error) {
	profilesDir := filepath.Join(e.configDir, "profiles")

	// Check if profiles directory exists
	exists, err := afero.DirExists(e.fs, profilesDir)
	if err != nil {
		return nil, nil, err
	}
	if !exists {
		// No profiles directory means no profiles
		return []ProfileInfo{}, []BrokenProfile{}, nil
	}

	// Read vendor directories
	vendorDirs, err := afero.ReadDir(e.fs, profilesDir)
	if err != nil {
		return nil, nil, err
	}

	var validProfiles []ProfileInfo
	var brokenProfiles []BrokenProfile

	// Walk two levels: vendor/name
	for _, vendorDir := range vendorDirs {
		if !vendorDir.IsDir() {
			continue
		}

		vendorName := vendorDir.Name()
		vendorPath := filepath.Join(profilesDir, vendorName)

		// Read profile directories within vendor
		profileDirs, err := afero.ReadDir(e.fs, vendorPath)
		if err != nil {
			return nil, nil, err
		}

		for _, profileDir := range profileDirs {
			if !profileDir.IsDir() {
				continue
			}

			profileName := profileDir.Name()
			fullProfileName := vendorName + "/" + profileName
			profilePath := filepath.Join(vendorPath, profileName)

			// Check for profile.yaml existence
			profileYAMLPath := filepath.Join(profilePath, "profile.yaml")
			profileYAMLExists, err := afero.Exists(e.fs, profileYAMLPath)
			if err != nil {
				return nil, nil, err
			}

			if !profileYAMLExists {
				// Broken profile (directory exists but no profile.yaml)
				brokenProfiles = append(brokenProfiles, BrokenProfile{
					Name:   fullProfileName,
					Reason: "missing profile.yaml",
				})
				continue
			}

			// Parse profile.yaml to extract inherits_from
			config, parseErr := e.parseProfileYAML(profileYAMLPath)
			if parseErr != nil {
				// Broken profile (invalid YAML syntax)
				brokenProfiles = append(brokenProfiles, BrokenProfile{
					Name:   fullProfileName,
					Reason: "invalid YAML syntax",
				})
				continue
			}

			// Valid profile with parsed inheritance info
			validProfiles = append(validProfiles, ProfileInfo{
				Name:         fullProfileName,
				InheritsFrom: config.InheritsFrom,
			})
		}
	}

	// Sort profiles alphabetically by full name
	sort.Slice(validProfiles, func(i, j int) bool {
		return validProfiles[i].Name < validProfiles[j].Name
	})
	sort.Slice(brokenProfiles, func(i, j int) bool {
		return brokenProfiles[i].Name < brokenProfiles[j].Name
	})

	return validProfiles, brokenProfiles, nil
}

// parseProfileYAML reads and parses a profile.yaml file to extract configuration.
func (e *ProfileEnumerator) parseProfileYAML(path string) (*profileYAMLConfig, error) {
	data, err := afero.ReadFile(e.fs, path)
	if err != nil {
		return nil, err
	}

	var config profileYAMLConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	return &config, nil
}

// NewProfileListCommand creates and returns the profile list subcommand.
// The filesystem is injected for testability.
// This uses the OS filesystem resolver for XDG Base Directory resolution.
func NewProfileListCommand(fs afero.Fs) *cobra.Command {
	// Create a default resolver using the OS filesystem for XDG resolution
	resolver := infraconfig.NewResolver()
	return NewProfileListCommandWithResolver(fs, os.UserHomeDir, os.Stdin, os.Stdout, resolver)
}

// NewProfileListCommandWithHomeDir creates and returns the profile list subcommand
// with an injectable home directory function.
// This is primarily used for testing to provide a custom home directory.
// It uses a fallback resolver that always uses homeDir + "/.weftlo".
func NewProfileListCommandWithHomeDir(fs afero.Fs, homeDir HomeDirFunc) *cobra.Command {
	// Create a simple resolver that uses the provided homeDir for backward compatibility with tests
	resolver := &homeDirResolver{homeDir: homeDir}
	return NewProfileListCommandWithResolver(fs, homeDir, os.Stdin, os.Stdout, resolver)
}

// NewProfileListCommandWithResolver creates and returns the profile list subcommand
// with a custom config directory resolver.
// This is primarily used for testing XDG behavior.
func NewProfileListCommandWithResolver(fs afero.Fs, homeDir HomeDirFunc, stdin io.Reader, stdout io.Writer, resolver ConfigDirResolver) *cobra.Command {
	listCmd := &ProfileListCommand{
		fs:                fs,
		configDirResolver: resolver,
		stdout:            stdout,
	}

	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List available configuration profiles",
		Long: `List all available configuration profiles from the config directory.

This command enumerates profiles from:
  <config-dir>/profiles/{vendor}/{name}/

Output modes:
  - Default: Human-readable list of profile names
  - JSON (--json): Machine-readable JSON format

Examples:
  weftlo profile list
  weftlo profile ls
  weftlo profile list --json`,
		RunE: listCmd.run,
	}

	// Add --json flag for JSON output
	cmd.Flags().BoolVar(&listCmd.jsonOutput, "json", false, "Output in JSON format")

	// Add --quiet flag (also inherited from root persistent flags when registered there)
	cmd.Flags().BoolVarP(&listCmd.quiet, "quiet", "q", false, "Suppress non-essential output")

	// Add --verbose flag (also inherited from root persistent flags when registered there)
	cmd.Flags().BoolVarP(&listCmd.verbose, "verbose", "v", false, "Enable verbose output")

	return cmd
}

// run is the RunE function for the profile list command.
func (c *ProfileListCommand) run(cmd *cobra.Command, args []string) error {
	// Resolve the config directory
	configDir, err := c.configDirResolver.ResolveConfigDir()
	if err != nil {
		return err
	}

	// Create profile enumerator
	enumerator := NewProfileEnumerator(c.fs, configDir)

	var validProfiles []ProfileInfo
	var brokenProfiles []BrokenProfile

	// Use verbose enumeration when verbose mode is enabled
	if c.verbose {
		validProfiles, brokenProfiles, err = enumerator.EnumerateProfilesVerbose()
	} else {
		validProfiles, brokenProfiles, err = enumerator.EnumerateProfiles()
	}
	if err != nil {
		return err
	}

	// Build result with enumerated profiles
	result := &ProfileListResult{
		Profiles:       validProfiles,
		BrokenProfiles: brokenProfiles,
	}

	// Output based on mode
	if c.jsonOutput {
		return c.writeJSONOutput(result)
	}

	// Use verbose output when verbose mode is enabled
	if c.verbose {
		return c.writeVerboseHumanReadableOutput(result)
	}

	return c.writeHumanReadableOutput(result)
}

// writeJSONOutput writes the profile list result as formatted JSON to stdout.
// Task 4.4: Implement JSON output mode
func (c *ProfileListCommand) writeJSONOutput(result *ProfileListResult) error {
	jsonData, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to serialize JSON output: %w", err)
	}
	_, _ = c.stdout.Write(jsonData)
	_, _ = io.WriteString(c.stdout, "\n")
	return nil
}

// writeHumanReadableOutput writes the profile list result in human-readable format.
// Task 4.2: Implement default human-readable output
func (c *ProfileListCommand) writeHumanReadableOutput(result *ProfileListResult) error {
	// Task 4.5: Handle empty state
	if len(result.Profiles) == 0 {
		_, _ = io.WriteString(c.stdout, "No profiles found.\n")
		return nil
	}

	// Display "Available profiles:" header
	_, _ = io.WriteString(c.stdout, "Available profiles:\n")

	// List each profile name with two-space indent
	// Skip broken profiles silently (they are not shown in default mode)
	for _, profile := range result.Profiles {
		_, _ = io.WriteString(c.stdout, "  "+profile.Name+"\n")
	}

	return nil
}

// writeVerboseHumanReadableOutput writes the profile list result in verbose human-readable format.
// Task 4.3: Implement verbose human-readable output
func (c *ProfileListCommand) writeVerboseHumanReadableOutput(result *ProfileListResult) error {
	// Task 4.5: Handle empty state (no valid profiles and no broken profiles)
	if len(result.Profiles) == 0 && len(result.BrokenProfiles) == 0 {
		_, _ = io.WriteString(c.stdout, "No profiles found.\n")
		return nil
	}

	// Display valid profiles section
	if len(result.Profiles) > 0 {
		_, _ = io.WriteString(c.stdout, "Available profiles:\n")

		for _, profile := range result.Profiles {
			// Include "(inherits from {parent})" after profile names with inheritance
			if profile.InheritsFrom != "" {
				_, _ = io.WriteString(c.stdout, fmt.Sprintf("  %s (inherits from %s)\n", profile.Name, profile.InheritsFrom))
			} else {
				_, _ = io.WriteString(c.stdout, "  "+profile.Name+"\n")
			}
		}
	}

	// Add "Broken profiles:" section at bottom if there are any
	if len(result.BrokenProfiles) > 0 {
		// Add a newline separator if we had valid profiles
		if len(result.Profiles) > 0 {
			_, _ = io.WriteString(c.stdout, "\n")
		}

		_, _ = io.WriteString(c.stdout, "Broken profiles:\n")

		// Show error reason for each broken profile
		for _, broken := range result.BrokenProfiles {
			_, _ = io.WriteString(c.stdout, fmt.Sprintf("  %s (%s)\n", broken.Name, broken.Reason))
		}
	}

	return nil
}
