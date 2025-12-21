package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	domainconfig "github.com/simensen/weftlo/internal/domain/config"
	infraconfig "github.com/simensen/weftlo/internal/infrastructure/config"
)

// profileYamlConfig represents the structure of profile.yaml for the profile create command.
// This struct does not use omitempty on InheritsFrom to ensure the field is always present.
type profileYamlConfig struct {
	Name         string `yaml:"name"`
	InheritsFrom string `yaml:"inherits_from"`
}

// ProfileCreateCommand represents the profile create subcommand configuration.
// This type is exported for testing purposes.
type ProfileCreateCommand struct {
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

	// force overwrites existing profile when profile directory already exists.
	force bool

	// quiet suppresses non-essential output.
	quiet bool

	// verbose enables detailed operational output.
	verbose bool

	// createdPaths tracks the paths created during profile creation for output.
	createdPaths []string

	// profileName stores the profile name for use in displaySuccessOutput.
	profileName string

	// profileDir stores the profile directory path for use in displaySuccessOutput.
	profileDir string
}

// NewProfileCreateCommand creates and returns the profile create subcommand.
// The filesystem is injected for testability.
// This uses the OS filesystem resolver for XDG Base Directory resolution.
func NewProfileCreateCommand(fs afero.Fs) *cobra.Command {
	// Create a default resolver using the OS filesystem for XDG resolution
	resolver := infraconfig.NewResolver()
	return NewProfileCreateCommandWithResolver(fs, os.UserHomeDir, os.Stdin, os.Stdout, resolver)
}

// NewProfileCreateCommandWithHomeDir creates and returns the profile create subcommand
// with an injectable home directory function.
// This is primarily used for testing to provide a custom home directory.
// It uses a fallback resolver that always uses homeDir + "/.weftlo".
func NewProfileCreateCommandWithHomeDir(fs afero.Fs, homeDir HomeDirFunc) *cobra.Command {
	// Create a simple resolver that uses the provided homeDir for backward compatibility with tests
	resolver := &homeDirResolver{homeDir: homeDir}
	return NewProfileCreateCommandWithResolver(fs, homeDir, os.Stdin, os.Stdout, resolver)
}

// NewProfileCreateCommandWithResolver creates and returns the profile create subcommand
// with a custom config directory resolver.
// This is primarily used for testing XDG behavior.
func NewProfileCreateCommandWithResolver(fs afero.Fs, homeDir HomeDirFunc, stdin io.Reader, stdout io.Writer, resolver ConfigDirResolver) *cobra.Command {
	createCmd := &ProfileCreateCommand{
		fs:                fs,
		homeDir:           homeDir,
		configDirResolver: resolver,
		stdin:             stdin,
		stdout:            stdout,
		createdPaths:      make([]string, 0),
	}

	cmd := &cobra.Command{
		Use:   "create <vendor/name>",
		Short: "Create a new configuration profile",
		Long: `Create a new configuration profile with the specified name.

The profile name must be in vendor/name format, where both vendor and name
consist of alphanumeric characters, underscores, and hyphens.

This command creates:
  - <config-dir>/profiles/{vendor}/{name}/          Profile directory
  - <config-dir>/profiles/{vendor}/{name}/profile.yaml  Profile configuration
  - <config-dir>/profiles/{vendor}/{name}/README.md     Profile documentation

Examples:
  weftlo profile create mycompany/backend
  weftlo profile create personal/dotfiles
  weftlo profile create --force existing/profile`,
		Args: cobra.ExactArgs(1),
		RunE: createCmd.run,
	}

	// Add --force flag via Flags().BoolVarP()
	cmd.Flags().BoolVarP(&createCmd.force, "force", "f", false, "Overwrite existing profile")

	// Add --quiet flag via Flags().BoolVarP()
	cmd.Flags().BoolVarP(&createCmd.quiet, "quiet", "q", false, "Suppress non-essential output")

	// Add --verbose flag via Flags().BoolVarP()
	cmd.Flags().BoolVarP(&createCmd.verbose, "verbose", "v", false, "Enable verbose output")

	return cmd
}

// run is the RunE function for the profile create command.
func (c *ProfileCreateCommand) run(cmd *cobra.Command, args []string) error {
	// Extract profile name from the first argument
	c.profileName = args[0]

	// Validate profile name format using domain validation
	if err := domainconfig.ValidateProfileName(c.profileName); err != nil {
		return err
	}

	// Resolve the profile directory path
	profileDir, err := c.resolveProfileDirectory(c.profileName)
	if err != nil {
		return err
	}
	c.profileDir = profileDir

	// Check if profile directory already exists
	exists, err := c.profileExists(profileDir)
	if err != nil {
		return err
	}

	// If profile exists and --force not set, return ProfileExistsError
	if exists && !c.force {
		return &ProfileExistsError{ProfileName: c.profileName}
	}

	// Create the profile directory structure
	if err := c.createDirectories(profileDir); err != nil {
		return err
	}

	// Create profile.yaml
	profileYamlPath := filepath.Join(profileDir, "profile.yaml")
	if err := c.createProfileYaml(profileYamlPath, c.profileName); err != nil {
		return err
	}

	// Create README.md
	readmePath := filepath.Join(profileDir, "README.md")
	if err := c.createReadme(readmePath, c.profileName); err != nil {
		return err
	}

	// Display success output (unless quiet mode is enabled)
	c.displaySuccessOutput()

	return nil
}

// profileExists checks if the profile directory already exists.
// Returns (bool, error) tuple following init.go configExists() pattern.
func (c *ProfileCreateCommand) profileExists(profileDir string) (bool, error) {
	exists, err := afero.Exists(c.fs, profileDir)
	if err != nil {
		return false, &PermissionError{Path: profileDir, Err: err}
	}
	return exists, nil
}

// resolveProfileDirectory resolves the full path to the profile directory.
// It uses configDirResolver.ResolveConfigDir() and appends "/profiles/{vendor}/{name}".
func (c *ProfileCreateCommand) resolveProfileDirectory(profileName string) (string, error) {
	// Resolve the config directory
	configDir, err := c.configDirResolver.ResolveConfigDir()
	if err != nil {
		return "", &HomeDirError{Err: err}
	}

	// Parse vendor and name from the profile name (format is validated, so we know it has exactly one /)
	parts := strings.Split(profileName, "/")
	vendor := parts[0]
	name := parts[1]

	// Build the full profile directory path
	profileDir := filepath.Join(configDir, "profiles", vendor, name)

	return profileDir, nil
}

// createDirectories creates the profile directory structure with 0755 permissions.
// It tracks created directories via trackCreatedPath().
func (c *ProfileCreateCommand) createDirectories(profileDir string) error {
	if err := c.fs.MkdirAll(profileDir, 0755); err != nil {
		return &PermissionError{
			Path: profileDir,
			Err:  err,
		}
	}
	c.trackCreatedPath(profileDir)

	// Verbose output for directory creation
	if c.verbose && !c.quiet {
		_, _ = fmt.Fprintf(c.stdout, "Creating directory: %s\n", profileDir)
	}

	return nil
}

// createProfileYaml creates the profile.yaml file with the profile configuration.
// It includes the profile name and an explicit empty inherits_from field.
func (c *ProfileCreateCommand) createProfileYaml(path string, profileName string) error {
	config := profileYamlConfig{
		Name:         profileName,
		InheritsFrom: "", // Explicitly empty, not omitted
	}

	data, err := yaml.Marshal(&config)
	if err != nil {
		return err
	}

	if err := afero.WriteFile(c.fs, path, data, 0644); err != nil {
		return &PermissionError{
			Path: path,
			Err:  err,
		}
	}

	c.trackCreatedPath(path)

	// Verbose output for file creation
	if c.verbose && !c.quiet {
		_, _ = fmt.Fprintf(c.stdout, "Creating file: %s\n", path)
	}

	return nil
}

// createReadme creates the README.md file with a heading containing the profile name
// and placeholder description encouraging customization.
func (c *ProfileCreateCommand) createReadme(path string, profileName string) error {
	content := fmt.Sprintf(`# %s Profile

This profile was created with weftlo.

## Description

Add a description of this profile and its purpose here.

## Usage

This profile can be customized by:
- Adding template files (.tmpl) to this directory
- Setting the inherits_from field in profile.yaml to inherit from another profile
- Adding variables to profile.yaml for use in templates

## Customization

Edit the profile.yaml file to configure this profile's behavior.
`, profileName)

	if err := afero.WriteFile(c.fs, path, []byte(content), 0644); err != nil {
		return &PermissionError{
			Path: path,
			Err:  err,
		}
	}

	c.trackCreatedPath(path)

	// Verbose output for file creation
	if c.verbose && !c.quiet {
		_, _ = fmt.Fprintf(c.stdout, "Creating file: %s\n", path)
	}

	return nil
}

// trackCreatedPath adds a path to the list of created paths for output.
func (c *ProfileCreateCommand) trackCreatedPath(path string) {
	c.createdPaths = append(c.createdPaths, path)
}

// displaySuccessOutput displays the success message with created files.
// This output is suppressed when --quiet flag is set.
// Follows init.go displaySuccessOutput() pattern.
func (c *ProfileCreateCommand) displaySuccessOutput() {
	if c.quiet {
		return
	}

	_, _ = fmt.Fprintf(c.stdout, "Created profile: %s\n", c.profileName)
	_, _ = fmt.Fprintln(c.stdout)
	_, _ = fmt.Fprintln(c.stdout, "Files created:")

	for _, path := range c.createdPaths {
		// Make the path relative to profile directory for cleaner output
		relPath, err := filepath.Rel(c.profileDir, path)
		if err != nil {
			relPath = path
		}

		// Determine if it's a directory or file
		info, err := c.fs.Stat(path)
		if err != nil {
			continue
		}

		if info.IsDir() {
			_, _ = fmt.Fprintf(c.stdout, "  %s/\n", relPath)
		} else {
			_, _ = fmt.Fprintf(c.stdout, "  %s\n", relPath)
		}
	}

	_, _ = fmt.Fprintln(c.stdout)
	_, _ = fmt.Fprintf(c.stdout, "Profile directory: %s\n", c.profileDir)
}
