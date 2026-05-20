package cli

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	infraconfig "github.com/simensen/weftlo/internal/infrastructure/config"
)

// ConfigDirResolver resolves the configuration directory.
// This interface allows for dependency injection in tests.
type ConfigDirResolver interface {
	// ResolveConfigDir resolves the config directory for reading existing config.
	// It checks if ~/.config/weftlo/ exists before falling back to ~/.weftlo/.
	ResolveConfigDir() (string, error)

	// ResolveConfigDirForCreation resolves the config directory for creating new config.
	// It checks if ~/.config/ exists (parent) before falling back to ~/.weftlo/.
	ResolveConfigDirForCreation() (string, error)
}

// homeDirResolver is a simple resolver that always returns homeDir + "/.weftlo".
// This is used by test constructors to maintain backward compatibility with tests
// that inject a mock home directory.
type homeDirResolver struct {
	homeDir HomeDirFunc
}

func (r *homeDirResolver) ResolveConfigDir() (string, error) {
	home, err := r.homeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".weftlo"), nil
}

func (r *homeDirResolver) ResolveConfigDirForCreation() (string, error) {
	// For tests, both methods return the same path
	return r.ResolveConfigDir()
}

// HomeDirFunc is a function type for resolving the user's home directory.
// This allows for dependency injection in tests.
type HomeDirFunc func() (string, error)

// InitCommand represents the init subcommand configuration.
// This type is exported for testing purposes.
type InitCommand struct {
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

	// force skips the reinitialize prompt when config already exists.
	force bool

	// quiet suppresses non-essential output.
	quiet bool

	// verbose enables detailed operational output.
	verbose bool

	// installPrefix is the default install prefix to set in the global config.
	installPrefix string

	// createdPaths tracks the paths created during initialization for output.
	createdPaths []string

	// configDirSource tracks the source of the config directory resolution for verbose output.
	configDirSource string
}

// globalConfig represents the structure of config.yaml
type globalConfig struct {
	DefaultProfile string `yaml:"default_profile"`
	InstallPrefix  string `yaml:"install_prefix,omitempty"`
}

// profileConfig represents the structure of profile.yaml.
type profileConfig struct {
	Name string `yaml:"name"`
}

// NewInitCommand creates and returns the init subcommand.
// The filesystem is injected for testability.
// This uses the OS filesystem resolver for XDG Base Directory resolution.
func NewInitCommand(fs afero.Fs) *cobra.Command {
	// Create a default resolver using the OS filesystem for XDG resolution
	resolver := infraconfig.NewResolver()
	return NewInitCommandWithResolver(fs, os.UserHomeDir, os.Stdin, os.Stdout, resolver)
}

// NewInitCommandWithHomeDir creates and returns the init subcommand with an injectable home directory function.
// This is primarily used for testing to provide a custom home directory.
// It uses a fallback resolver that always uses homeDir + "/.weftlo".
func NewInitCommandWithHomeDir(fs afero.Fs, homeDir HomeDirFunc) *cobra.Command {
	return NewInitCommandWithIO(fs, homeDir, os.Stdin, os.Stdout)
}

// NewInitCommandWithIO creates and returns the init subcommand with injectable I/O streams.
// This allows tests to inject custom stdin/stdout for testing interactive prompts.
// It uses a fallback resolver that always uses homeDir + "/.weftlo".
func NewInitCommandWithIO(fs afero.Fs, homeDir HomeDirFunc, stdin io.Reader, stdout io.Writer) *cobra.Command {
	// Create a simple resolver that uses the provided homeDir for backward compatibility with tests
	resolver := &homeDirResolver{homeDir: homeDir}
	return NewInitCommandWithResolver(fs, homeDir, stdin, stdout, resolver)
}

// NewInitCommandWithResolver creates and returns the init subcommand with a custom config directory resolver.
// This is primarily used for testing XDG behavior.
func NewInitCommandWithResolver(fs afero.Fs, homeDir HomeDirFunc, stdin io.Reader, stdout io.Writer, resolver ConfigDirResolver) *cobra.Command {
	initCmd := &InitCommand{
		fs:                fs,
		homeDir:           homeDir,
		configDirResolver: resolver,
		stdin:             stdin,
		stdout:            stdout,
		createdPaths:      make([]string, 0),
	}

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize the global configuration directory",
		Long: `Initialize the weftlo directory structure with a default profile.

The configuration directory is determined by XDG Base Directory conventions:
  1. $XDG_CONFIG_HOME/weftlo/ if XDG_CONFIG_HOME is set
  2. ~/.config/weftlo/ if ~/.config/ exists
  3. ~/.weftlo/ as fallback

This command creates:
  - <config-dir>/                     Root configuration directory
  - <config-dir>/config.yaml          Global configuration file
  - <config-dir>/profiles/            Profile storage directory
  - <config-dir>/profiles/default/default/  Default profile
  - <config-dir>/profiles/default/default/content/  Content directory

If the configuration already exists, you will be prompted to reinitialize.
Use --force to skip the prompt and reinitialize immediately.`,
		RunE: initCmd.run,
	}

	// Add --force flag via Flags().BoolVarP()
	cmd.Flags().BoolVarP(&initCmd.force, "force", "f", false, "Skip reinitialize prompt and proceed directly")

	// Add --quiet flag via Flags().BoolVarP()
	cmd.Flags().BoolVarP(&initCmd.quiet, "quiet", "q", false, "Suppress non-essential output")

	// Add --verbose flag via Flags().BoolVarP()
	cmd.Flags().BoolVarP(&initCmd.verbose, "verbose", "v", false, "Enable verbose output")

	// Add --install-prefix flag via Flags().StringVarP()
	cmd.Flags().StringVarP(&initCmd.installPrefix, "install-prefix", "", "", "Set default install prefix")

	return cmd
}

// run is the RunE function for the init command.
func (c *InitCommand) run(cmd *cobra.Command, args []string) error {
	// Resolve config directory using XDG conventions
	configDir, err := c.configDirResolver.ResolveConfigDirForCreation()
	if err != nil {
		return &HomeDirError{Err: err}
	}

	// Determine the source of the config directory for verbose output
	c.configDirSource = c.determineConfigDirSource(configDir)

	// Display verbose config directory resolution
	c.displayVerboseConfigDir(configDir)

	// Build paths
	configPath := filepath.Join(configDir, "config.yaml")

	// Check if config.yaml already exists
	exists, err := c.configExists(configPath)
	if err != nil {
		return err
	}

	// If config exists and not forced, prompt for reinitialize
	if exists && !c.force {
		shouldProceed, err := c.promptReinitialize()
		if err != nil {
			return err
		}
		if !shouldProceed {
			return &ReinitializeAbortedError{}
		}
	}

	// Build remaining paths
	profilesDir := filepath.Join(configDir, "profiles")
	vendorDir := filepath.Join(profilesDir, "default")
	defaultProfileDir := filepath.Join(vendorDir, "default")
	contentDir := filepath.Join(defaultProfileDir, "content")

	// Create directory structure (including content directory)
	if err := c.createDirectories(configDir, profilesDir, vendorDir, defaultProfileDir, contentDir); err != nil {
		return err
	}

	// Create config.yaml
	if err := c.createConfigYaml(configPath); err != nil {
		return err
	}

	// Create profile.yaml (at profile root, not in content/)
	profilePath := filepath.Join(defaultProfileDir, "profile.yaml")
	if err := c.createProfileYaml(profilePath); err != nil {
		return err
	}

	// Create a hello-world CLAUDE.md.tmpl inside content/ so that
	// `weftlo install` produces something visible in .claude/ that the
	// user can immediately recognize as "this is where my AI rules go."
	templatePath := filepath.Join(contentDir, "CLAUDE.md.tmpl")
	if err := c.createDefaultTemplate(templatePath); err != nil {
		return err
	}

	// Display success output (unless quiet mode is enabled)
	c.displaySuccessOutput(configDir)

	return nil
}

// determineConfigDirSource determines whether the config directory is XDG or fallback.
func (c *InitCommand) determineConfigDirSource(configDir string) string {
	// Check if it's an XDG path (contains .config)
	if strings.Contains(configDir, ".config") {
		return "XDG"
	}
	return "fallback"
}

// displayVerboseConfigDir prints verbose output about config directory resolution.
func (c *InitCommand) displayVerboseConfigDir(configDir string) {
	if !c.verbose || c.quiet {
		return
	}
	_, _ = fmt.Fprintf(c.stdout, "Config directory: %s (%s)\n", configDir, c.configDirSource)
}

// configExists checks if the config.yaml file exists.
func (c *InitCommand) configExists(path string) (bool, error) {
	exists, err := afero.Exists(c.fs, path)
	if err != nil {
		return false, &PermissionError{Path: path, Err: err}
	}
	return exists, nil
}

// promptReinitialize prompts the user asking if they want to reinitialize.
// Returns true if the user confirms, false if they decline.
func (c *InitCommand) promptReinitialize() (bool, error) {
	_, _ = fmt.Fprint(c.stdout, "Configuration already exists. Do you want to reinitialize? [y/N]: ")

	reader := bufio.NewReader(c.stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		// If we hit EOF without a newline, use what we have
		if err == io.EOF && response != "" {
			// Fall through to process the response
		} else if err == io.EOF {
			// Empty input, treat as "N"
			return false, nil
		} else {
			return false, err
		}
	}

	response = strings.TrimSpace(response)

	// Accept "y", "yes", "Y", "Yes" as affirmative responses
	switch strings.ToLower(response) {
	case "y", "yes":
		return true, nil
	default:
		return false, nil
	}
}

// createDirectories creates the required directory structure.
func (c *InitCommand) createDirectories(dirs ...string) error {
	for _, dir := range dirs {
		if err := c.fs.MkdirAll(dir, 0755); err != nil {
			return &PermissionError{
				Path: dir,
				Err:  err,
			}
		}
		c.trackCreatedPath(dir)

		// Verbose output for directory creation
		if c.verbose && !c.quiet {
			_, _ = fmt.Fprintf(c.stdout, "Creating directory: %s\n", dir)
		}
	}
	return nil
}

// createConfigYaml creates the config.yaml file with default content.
func (c *InitCommand) createConfigYaml(path string) error {
	config := globalConfig{
		DefaultProfile: "default/default",
		InstallPrefix:  c.installPrefix, // Will be omitted if empty due to omitempty tag
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

// createProfileYaml creates the profile.yaml file with default content.
// It declares one variable (`company`) so the default template has something
// to interpolate, and includes an inline comment showing the user where and
// how to add more variables. This is intentionally hand-formatted rather than
// going through yaml.Marshal so that the comments are preserved verbatim.
func (c *InitCommand) createProfileYaml(path string) error {
	// Marshal the name field through the YAML library so any future field
	// renames stay in sync with the struct tags.
	header, err := yaml.Marshal(&profileConfig{Name: "default/default"})
	if err != nil {
		return err
	}

	content := string(header) + `variables:
  company: "Your Company"
  # Add your own variables here, then reference them in templates
  # as {{ .Variables.your_var_name }}.
`

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

// createDefaultTemplate creates content/CLAUDE.md.tmpl — a neutral hello-world
// template that, after `weftlo install`, lands at `.claude/CLAUDE.md` with the
// `company` variable substituted in. The goal is for the user to see, in the
// first 60 seconds, where their AI rules live and how variables work.
func (c *InitCommand) createDefaultTemplate(path string) error {
	content := `# Coding Standards for {{ .Variables.company | default "this project" }}

<!--
This file is rendered from your weftlo profile.

Edit the template at the default profile's content/CLAUDE.md.tmpl and
re-run ` + "`weftlo update`" + ` to sync changes into your project.

Variables come from the profile.yaml ` + "`variables:`" + ` block, your
project's .weftlo.yaml, or the global config. See the weftlo docs for
the full precedence order.
-->

<!-- Add your coding standards, conventions, and AI-assistant instructions here. -->
`

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
func (c *InitCommand) trackCreatedPath(path string) {
	c.createdPaths = append(c.createdPaths, path)
}

// displaySuccessOutput displays the success message with created files/directories.
// This output is suppressed when --quiet flag is set.
func (c *InitCommand) displaySuccessOutput(configDir string) {
	if c.quiet {
		return
	}

	_, _ = fmt.Fprintln(c.stdout, "Initialized weftlo successfully!")
	_, _ = fmt.Fprintln(c.stdout)
	_, _ = fmt.Fprintln(c.stdout, "Created:")

	for _, path := range c.createdPaths {
		// Make the path relative to config directory for cleaner output
		relPath, err := filepath.Rel(configDir, path)
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
	_, _ = fmt.Fprintln(c.stdout, "Next steps:")
	_, _ = fmt.Fprintf(c.stdout, "  Run `weftlo install` in a project to install your profile\n")
	_, _ = fmt.Fprintln(c.stdout)
	_, _ = fmt.Fprintf(c.stdout, "Configuration directory: %s\n", configDir)
}
