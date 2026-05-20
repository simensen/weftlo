// Package cli provides the command-line interface for weftlo.
package cli

import (
	"fmt"
	"io"
	"os"
	"runtime"

	"github.com/spf13/afero"
	"github.com/spf13/cobra"

	infraconfig "github.com/simensen/weftlo/internal/infrastructure/config"
)

// NewRootCommand creates and returns the root command for weftlo CLI.
// The filesystem is injected for testability - use fs.New() for production
// or afero.NewMemMapFs() for tests.
// The version string is injected at build time via ldflags.
// This uses the OS filesystem resolver for XDG Base Directory resolution.
func NewRootCommand(fs afero.Fs, version string) *cobra.Command {
	// Production: use OS resolver for XDG support
	return newRootCommandInternal(fs, os.UserHomeDir, os.Stdin, infraconfig.NewResolver(), version)
}

// NewRootCommandWithHomeDir creates and returns the root command with an injectable home directory function.
// This is primarily used for testing to provide a custom home directory.
// It uses a fallback resolver that always uses homeDir + "/.weftlo".
func NewRootCommandWithHomeDir(fs afero.Fs, homeDir HomeDirFunc) *cobra.Command {
	return NewRootCommandWithHomeDirAndIO(fs, homeDir, os.Stdin)
}

// NewRootCommandWithHomeDirAndIO creates and returns the root command with injectable home directory and I/O.
// This allows tests to inject custom stdin for testing interactive prompts.
// It uses a fallback resolver that always uses homeDir + "/.weftlo".
func NewRootCommandWithHomeDirAndIO(fs afero.Fs, homeDir HomeDirFunc, stdin io.Reader) *cobra.Command {
	// Test mode: use homeDirResolver for backward compatibility
	resolver := &homeDirResolver{homeDir: homeDir}
	return newRootCommandInternal(fs, homeDir, stdin, resolver, "dev")
}

// newRootCommandInternal creates the root command with the specified resolver.
//
// # Command Wrapper Pattern Architecture
//
// This function uses a "wrapper pattern" where thin wrapper structs bridge between
// the root command (where global flags are defined) and the actual command implementations
// (in init.go, install.go, update.go, status.go). This pattern exists for three reasons:
//
//  1. Deferred stdout resolution: Tests call rootCmd.SetOut(&stdout) after command
//     creation but before execution to capture output. The wrappers defer calling
//     rootCmd.OutOrStdout() until execution time, allowing tests to redirect output.
//
//  2. Global flag forwarding: The --verbose and --quiet flags are persistent flags
//     on the root command. The wrappers hold pointers to these flag variables because
//     at wrapper creation time the flags have default values, but at execution time
//     they have the parsed user-provided values. Pointers enable reading final values.
//
//  3. Two-layer command structure: Each command file defines standalone cobra.Command
//     instances with their own flags, which supports independent testing of each command.
//     The wrappers compose these standalone commands with the root's global flags.
//
// Simplification Analysis (evaluated 2024):
//   - Cannot resolve stdout earlier: Tests depend on SetOut() being called post-creation
//   - Cannot use value flags: Flag values aren't parsed until Execute() is called
//   - Cannot eliminate wrappers: Would require tighter coupling or breaking test patterns
//
// Future simplification opportunities:
//   - If Cobra adds a pre-execution hook, stdout could be resolved there
//   - If command tests are refactored to not need standalone commands, wrappers could
//     be eliminated by having command implementations access root persistent flags directly
func newRootCommandInternal(fs afero.Fs, homeDir HomeDirFunc, stdin io.Reader, resolver ConfigDirResolver, version string) *cobra.Command {
	// Global flag variables bound to persistent flags.
	// These are local variables whose addresses are passed to wrapper structs.
	// The wrappers use pointers because flag values are not populated until
	// Execute() is called, so we need to read them at execution time.
	var verbose bool
	var quiet bool

	rootCmd := &cobra.Command{
		Use:   "weftlo",
		Short: "Manage agent configuration profiles",
		Long: `Weftlo is a CLI tool for managing configuration profiles
that can be installed into projects for AI coding assistants.

It supports profile inheritance, template rendering, and
configuration management across multiple projects.`,
	}

	// Add global persistent flags
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose output")
	rootCmd.PersistentFlags().BoolVarP(&quiet, "quiet", "q", false, "Suppress non-essential output")

	// Create the init command with a wrapper that defers stdout resolution.
	// The wrapper enables: (1) calling rootCmd.OutOrStdout() at execution time,
	// (2) reading final values of --verbose and --quiet flags via pointers.
	initCmdWrapper := &initCommandWrapper{
		fs:       fs,
		homeDir:  homeDir,
		stdin:    stdin,
		rootCmd:  rootCmd,
		resolver: resolver,
		verbose:  &verbose,
		quiet:    &quiet,
	}

	initCmd := &cobra.Command{
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

If the configuration already exists, you will be prompted to reinitialize.
Use --force to skip the prompt and reinitialize immediately.`,
		Example: `  weftlo init
  weftlo init --install-prefix .claude`,
		RunE: initCmdWrapper.run,
	}

	// Add flags (--quiet and --verbose removed, now come from root persistent flags)
	initCmd.Flags().BoolVarP(&initCmdWrapper.force, "force", "f", false, "Skip reinitialize prompt and proceed directly")
	initCmd.Flags().StringVarP(&initCmdWrapper.installPrefix, "install-prefix", "", "", "Set default install prefix")

	rootCmd.AddCommand(initCmd)

	// Create the install command with a wrapper that defers stdout resolution.
	// See initCmdWrapper comment above for explanation of the wrapper pattern.
	installCmdWrapper := &installCommandWrapper{
		fs:       fs,
		homeDir:  homeDir,
		stdin:    stdin,
		rootCmd:  rootCmd,
		resolver: resolver,
		verbose:  &verbose,
		quiet:    &quiet,
	}

	installCmd := &cobra.Command{
		Use:   "install",
		Short: "Install a profile to the current project",
		Long: `Install a profile to the current project directory.

The profile is resolved in the following order:
  1. --profile flag (if specified)
  2. .weftlo.yaml profile field (if exists)
  3. ~/.weftlo/config.yaml default_profile

The install prefix is resolved in the following order:
  1. --install-prefix flag (if specified)
  2. .weftlo.yaml install_prefix (if exists)
  3. ~/.weftlo/config.yaml install_prefix (if set)
  4. Default: ".claude"

If .weftlo.yaml does not exist, it will be created after successful installation.`,
		Example: `  weftlo install
  weftlo install --profile mycompany/backend
  weftlo install --dry-run`,
		RunE: installCmdWrapper.run,
	}

	// Add flags (--quiet and --verbose removed, now come from root persistent flags)
	installCmd.Flags().StringVarP(&installCmdWrapper.profile, "profile", "p", "", "Profile to install (vendor/name format)")
	installCmd.Flags().StringVarP(&installCmdWrapper.installPrefix, "install-prefix", "", "", "Installation directory prefix")
	installCmd.Flags().BoolVarP(&installCmdWrapper.dryRun, "dry-run", "n", false, "Preview changes without writing files")
	installCmd.Flags().BoolVarP(&installCmdWrapper.force, "force", "f", false, "Overwrite existing files")

	rootCmd.AddCommand(installCmd)

	// Create the update command with a wrapper that defers stdout resolution.
	// See initCmdWrapper comment above for explanation of the wrapper pattern.
	updateCmdWrapper := &updateCommandWrapper{
		fs:       fs,
		homeDir:  homeDir,
		stdin:    stdin,
		rootCmd:  rootCmd,
		resolver: resolver,
		verbose:  &verbose,
		quiet:    &quiet,
	}

	updateCmd := &cobra.Command{
		Use:   "update",
		Short: "Update installed configuration files from profile templates",
		Long: `Update installed configuration files by synchronizing with the latest profile templates.

This command detects changes between your installed files and the profile templates,
and updates files as needed based on manifest checksums.

The update command requires an existing installation:
  - .weftlo.yaml must exist (created by 'weftlo install')
  - .weftlo.manifest.json must exist (created by 'weftlo install')

File handling:
  - New files in the profile are created
  - Changed template files update the output if the user hasn't modified it
  - User-modified files are skipped (use --force to overwrite)
  - Files removed from the profile are kept (use --remove-orphans to delete)`,
		Example: `  weftlo update
  weftlo update --force
  weftlo update --dry-run`,
		RunE: updateCmdWrapper.run,
	}

	// Add flags (--quiet and --verbose removed, now come from root persistent flags)
	updateCmd.Flags().BoolVarP(&updateCmdWrapper.force, "force", "f", false, "Overwrite user-modified and conflicting files")
	updateCmd.Flags().BoolVarP(&updateCmdWrapper.dryRun, "dry-run", "n", false, "Preview changes without writing files")
	updateCmd.Flags().BoolVar(&updateCmdWrapper.removeOrphans, "remove-orphans", false, "Delete files no longer in the profile")

	rootCmd.AddCommand(updateCmd)

	// Create the status command with a wrapper that defers stdout resolution.
	// See initCmdWrapper comment above for explanation of the wrapper pattern.
	statusCmdWrapper := &statusCommandWrapper{
		fs:       fs,
		homeDir:  homeDir,
		stdin:    stdin,
		rootCmd:  rootCmd,
		resolver: resolver,
		verbose:  &verbose,
		quiet:    &quiet,
	}

	statusCmd := &cobra.Command{
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
		RunE: statusCmdWrapper.run,
	}

	// Add --json flag for JSON output (local to status command)
	statusCmd.Flags().BoolVar(&statusCmdWrapper.jsonOutput, "json", false, "Output in JSON format")

	rootCmd.AddCommand(statusCmd)

	// Create the version command
	versionCmd := &cobra.Command{
		Use:     "version",
		Short:   "Display version information",
		Example: "  weftlo version",
		RunE: func(cmd *cobra.Command, args []string) error {
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "weftlo version %s\n", version)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Go version: %s\n", runtime.Version())
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Platform: %s/%s\n", runtime.GOOS, runtime.GOARCH)
			return nil
		},
	}

	rootCmd.AddCommand(versionCmd)

	// Create the profile parent command (container command only, no RunE).
	// This groups profile-related subcommands under "weftlo profile".
	profileCmd := &cobra.Command{
		Use:   "profile",
		Short: "Manage configuration profiles",
		Long: `Manage configuration profiles for Weftlo.

This command group provides subcommands for creating, listing, and managing profiles.`,
	}

	// Create the profile create command with a wrapper that defers stdout resolution.
	// See initCmdWrapper comment above for explanation of the wrapper pattern.
	profileCreateCmdWrapper := &profileCreateCommandWrapper{
		fs:       fs,
		homeDir:  homeDir,
		stdin:    stdin,
		rootCmd:  rootCmd,
		resolver: resolver,
		verbose:  &verbose,
		quiet:    &quiet,
	}

	profileCreateCmd := &cobra.Command{
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
		RunE: profileCreateCmdWrapper.run,
	}

	// Add --force flag for overwriting existing profiles
	profileCreateCmd.Flags().BoolVarP(&profileCreateCmdWrapper.force, "force", "f", false, "Overwrite existing profile")

	// Add profile create command to profile parent command
	profileCmd.AddCommand(profileCreateCmd)

	// Create the profile list command with a wrapper that defers stdout resolution.
	// See initCmdWrapper comment above for explanation of the wrapper pattern.
	profileListCmdWrapper := &profileListCommandWrapper{
		fs:       fs,
		homeDir:  homeDir,
		rootCmd:  rootCmd,
		resolver: resolver,
		verbose:  &verbose,
		quiet:    &quiet,
	}

	profileListCmd := &cobra.Command{
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
		RunE: profileListCmdWrapper.run,
	}

	// Add --json flag for JSON output (local to profile list command)
	profileListCmd.Flags().BoolVar(&profileListCmdWrapper.jsonOutput, "json", false, "Output in JSON format")

	// Add profile list command to profile parent command
	profileCmd.AddCommand(profileListCmd)

	// Add profile parent command to root command
	rootCmd.AddCommand(profileCmd)

	// Add Cobra's built-in shell completion command
	// This provides `weftlo completion bash|zsh|fish|powershell` subcommands
	rootCmd.InitDefaultCompletionCmd()

	return rootCmd
}

// initCommandWrapper wraps the init command to support the command wrapper pattern.
//
// This wrapper exists to:
//  1. Defer stdout resolution: rootCmd.OutOrStdout() is called in run(), not at creation
//  2. Forward global flags: verbose and quiet are pointers to root's persistent flag variables
//
// The wrapper creates the actual InitCommand at execution time, sets its flags from both
// local flag values and root persistent flag values, then executes it.
type initCommandWrapper struct {
	fs            afero.Fs
	homeDir       HomeDirFunc
	stdin         io.Reader
	rootCmd       *cobra.Command
	resolver      ConfigDirResolver
	force         bool
	verbose       *bool // Pointer to root command's verbose flag (read at execution time)
	quiet         *bool // Pointer to root command's quiet flag (read at execution time)
	installPrefix string
}

func (w *initCommandWrapper) run(cmd *cobra.Command, args []string) error {
	// Create the actual InitCommand at execution time with the root command's
	// current stdout. This deferred resolution allows tests to call SetOut()
	// after command creation but before execution.
	initCmd := NewInitCommandWithResolver(w.fs, w.homeDir, w.stdin, w.rootCmd.OutOrStdout(), w.resolver)

	// Set the flags on the underlying command
	if err := initCmd.Flags().Set("force", boolToString(w.force)); err != nil {
		return err
	}

	// Get quiet value from root persistent flag (quiet takes precedence over verbose)
	quietValue := false
	if w.quiet != nil {
		quietValue = *w.quiet
	}
	if err := initCmd.Flags().Set("quiet", boolToString(quietValue)); err != nil {
		return err
	}

	// Get verbose value from root persistent flag
	verboseValue := false
	if w.verbose != nil {
		verboseValue = *w.verbose
	}
	if err := initCmd.Flags().Set("verbose", boolToString(verboseValue)); err != nil {
		return err
	}

	if w.installPrefix != "" {
		if err := initCmd.Flags().Set("install-prefix", w.installPrefix); err != nil {
			return err
		}
	}

	// Execute the command's RunE function directly
	return initCmd.RunE(initCmd, args)
}

// installCommandWrapper wraps the install command to support the command wrapper pattern.
// See initCommandWrapper for detailed explanation of why this pattern is necessary.
type installCommandWrapper struct {
	fs            afero.Fs
	homeDir       HomeDirFunc
	stdin         io.Reader
	rootCmd       *cobra.Command
	resolver      ConfigDirResolver
	profile       string
	installPrefix string
	dryRun        bool
	verbose       *bool // Pointer to root command's verbose flag (read at execution time)
	quiet         *bool // Pointer to root command's quiet flag (read at execution time)
	force         bool
}

func (w *installCommandWrapper) run(cmd *cobra.Command, args []string) error {
	// Create the actual InstallCommand at execution time with the root command's
	// current stdout. This deferred resolution allows tests to call SetOut()
	// after command creation but before execution.
	installCmd := NewInstallCommandWithResolver(w.fs, w.homeDir, w.stdin, w.rootCmd.OutOrStdout(), w.resolver)

	// Set the flags on the underlying command
	if w.profile != "" {
		if err := installCmd.Flags().Set("profile", w.profile); err != nil {
			return err
		}
	}
	if w.installPrefix != "" {
		if err := installCmd.Flags().Set("install-prefix", w.installPrefix); err != nil {
			return err
		}
	}
	if err := installCmd.Flags().Set("dry-run", boolToString(w.dryRun)); err != nil {
		return err
	}

	// Get quiet value from root persistent flag (quiet takes precedence over verbose)
	quietValue := false
	if w.quiet != nil {
		quietValue = *w.quiet
	}
	if err := installCmd.Flags().Set("quiet", boolToString(quietValue)); err != nil {
		return err
	}

	// Get verbose value from root persistent flag
	verboseValue := false
	if w.verbose != nil {
		verboseValue = *w.verbose
	}
	if err := installCmd.Flags().Set("verbose", boolToString(verboseValue)); err != nil {
		return err
	}

	if err := installCmd.Flags().Set("force", boolToString(w.force)); err != nil {
		return err
	}

	// Execute the command's RunE function directly
	return installCmd.RunE(installCmd, args)
}

// updateCommandWrapper wraps the update command to support the command wrapper pattern.
// See initCommandWrapper for detailed explanation of why this pattern is necessary.
type updateCommandWrapper struct {
	fs            afero.Fs
	homeDir       HomeDirFunc
	stdin         io.Reader
	rootCmd       *cobra.Command
	resolver      ConfigDirResolver
	force         bool
	dryRun        bool
	verbose       *bool // Pointer to root command's verbose flag (read at execution time)
	quiet         *bool // Pointer to root command's quiet flag (read at execution time)
	removeOrphans bool
}

func (w *updateCommandWrapper) run(cmd *cobra.Command, args []string) error {
	// Create the actual UpdateCommand at execution time with the root command's
	// current stdout. This deferred resolution allows tests to call SetOut()
	// after command creation but before execution.
	updateCmd := NewUpdateCommandWithResolver(w.fs, w.homeDir, w.stdin, w.rootCmd.OutOrStdout(), w.resolver)

	// Set the flags on the underlying command
	if err := updateCmd.Flags().Set("force", boolToString(w.force)); err != nil {
		return err
	}
	if err := updateCmd.Flags().Set("dry-run", boolToString(w.dryRun)); err != nil {
		return err
	}

	// Get quiet value from root persistent flag (quiet takes precedence over verbose)
	quietValue := false
	if w.quiet != nil {
		quietValue = *w.quiet
	}
	if err := updateCmd.Flags().Set("quiet", boolToString(quietValue)); err != nil {
		return err
	}

	// Get verbose value from root persistent flag
	verboseValue := false
	if w.verbose != nil {
		verboseValue = *w.verbose
	}
	if err := updateCmd.Flags().Set("verbose", boolToString(verboseValue)); err != nil {
		return err
	}

	if err := updateCmd.Flags().Set("remove-orphans", boolToString(w.removeOrphans)); err != nil {
		return err
	}

	// Execute the command's RunE function directly
	return updateCmd.RunE(updateCmd, args)
}

// statusCommandWrapper wraps the status command to support the command wrapper pattern.
// See initCommandWrapper for detailed explanation of why this pattern is necessary.
type statusCommandWrapper struct {
	fs         afero.Fs
	homeDir    HomeDirFunc
	stdin      io.Reader
	rootCmd    *cobra.Command
	resolver   ConfigDirResolver
	verbose    *bool // Pointer to root command's verbose flag (read at execution time)
	quiet      *bool // Pointer to root command's quiet flag (read at execution time)
	jsonOutput bool
}

func (w *statusCommandWrapper) run(cmd *cobra.Command, args []string) error {
	// Create the actual StatusCommand at execution time with the root command's
	// current stdout. This deferred resolution allows tests to call SetOut()
	// after command creation but before execution.
	statusCmd := NewStatusCommandWithResolver(w.fs, w.homeDir, w.stdin, w.rootCmd.OutOrStdout(), w.resolver)

	// Get quiet value from root persistent flag (quiet takes precedence over verbose)
	quietValue := false
	if w.quiet != nil {
		quietValue = *w.quiet
	}
	if err := statusCmd.Flags().Set("quiet", boolToString(quietValue)); err != nil {
		return err
	}

	// Get verbose value from root persistent flag
	verboseValue := false
	if w.verbose != nil {
		verboseValue = *w.verbose
	}
	if err := statusCmd.Flags().Set("verbose", boolToString(verboseValue)); err != nil {
		return err
	}

	// Set the --json flag
	if err := statusCmd.Flags().Set("json", boolToString(w.jsonOutput)); err != nil {
		return err
	}

	// Execute the command's RunE function directly
	return statusCmd.RunE(statusCmd, args)
}

// profileCreateCommandWrapper wraps the profile create command to support the command wrapper pattern.
// See initCommandWrapper for detailed explanation of why this pattern is necessary.
type profileCreateCommandWrapper struct {
	fs       afero.Fs
	homeDir  HomeDirFunc
	stdin    io.Reader
	rootCmd  *cobra.Command
	resolver ConfigDirResolver
	force    bool
	verbose  *bool // Pointer to root command's verbose flag (read at execution time)
	quiet    *bool // Pointer to root command's quiet flag (read at execution time)
}

func (w *profileCreateCommandWrapper) run(cmd *cobra.Command, args []string) error {
	// Create the actual ProfileCreateCommand at execution time with the root command's
	// current stdout. This deferred resolution allows tests to call SetOut()
	// after command creation but before execution.
	profileCreateCmd := NewProfileCreateCommandWithResolver(w.fs, w.homeDir, w.stdin, w.rootCmd.OutOrStdout(), w.resolver)

	// Set the flags on the underlying command
	if err := profileCreateCmd.Flags().Set("force", boolToString(w.force)); err != nil {
		return err
	}

	// Get quiet value from root persistent flag (quiet takes precedence over verbose)
	quietValue := false
	if w.quiet != nil {
		quietValue = *w.quiet
	}
	if err := profileCreateCmd.Flags().Set("quiet", boolToString(quietValue)); err != nil {
		return err
	}

	// Get verbose value from root persistent flag
	verboseValue := false
	if w.verbose != nil {
		verboseValue = *w.verbose
	}
	if err := profileCreateCmd.Flags().Set("verbose", boolToString(verboseValue)); err != nil {
		return err
	}

	// Execute the command's RunE function directly
	return profileCreateCmd.RunE(profileCreateCmd, args)
}

// profileListCommandWrapper wraps the profile list command to support the command wrapper pattern.
// See initCommandWrapper for detailed explanation of why this pattern is necessary.
type profileListCommandWrapper struct {
	fs         afero.Fs
	homeDir    HomeDirFunc
	rootCmd    *cobra.Command
	resolver   ConfigDirResolver
	verbose    *bool // Pointer to root command's verbose flag (read at execution time)
	quiet      *bool // Pointer to root command's quiet flag (read at execution time)
	jsonOutput bool
}

func (w *profileListCommandWrapper) run(cmd *cobra.Command, args []string) error {
	// Create the actual ProfileListCommand at execution time with the root command's
	// current stdout. This deferred resolution allows tests to call SetOut()
	// after command creation but before execution.
	profileListCmd := NewProfileListCommandWithResolver(w.fs, w.homeDir, os.Stdin, w.rootCmd.OutOrStdout(), w.resolver)

	// Get quiet value from root persistent flag (quiet takes precedence over verbose)
	quietValue := false
	if w.quiet != nil {
		quietValue = *w.quiet
	}
	if err := profileListCmd.Flags().Set("quiet", boolToString(quietValue)); err != nil {
		return err
	}

	// Get verbose value from root persistent flag
	verboseValue := false
	if w.verbose != nil {
		verboseValue = *w.verbose
	}
	if err := profileListCmd.Flags().Set("verbose", boolToString(verboseValue)); err != nil {
		return err
	}

	// Set the --json flag
	if err := profileListCmd.Flags().Set("json", boolToString(w.jsonOutput)); err != nil {
		return err
	}

	// Execute the command's RunE function directly
	return profileListCmd.RunE(profileListCmd, args)
}

func boolToString(b bool) string {
	if b {
		return "true"
	}
	return "false"
}
