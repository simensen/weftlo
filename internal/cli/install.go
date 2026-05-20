package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/simensen/weftlo/internal/app/install"
	"github.com/simensen/weftlo/internal/app/rendering"
	"github.com/simensen/weftlo/internal/app/routing"
	domainconfig "github.com/simensen/weftlo/internal/domain/config"
	domainignore "github.com/simensen/weftlo/internal/domain/ignore"
	"github.com/simensen/weftlo/internal/domain/manifest"
	"github.com/simensen/weftlo/internal/domain/template"
	infraconfig "github.com/simensen/weftlo/internal/infrastructure/config"
	infraignore "github.com/simensen/weftlo/internal/infrastructure/ignore"
	infraprofile "github.com/simensen/weftlo/internal/infrastructure/profile"
)

// InstallCommand represents the install subcommand configuration.
// This type is exported for testing purposes.
type InstallCommand struct {
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

	// profiles is the optional profile names specified via --profile flag (repeatable).
	profiles []string

	// installPrefix is the optional install prefix specified via --install-prefix flag.
	installPrefix string

	// dryRun when true prevents any filesystem writes.
	dryRun bool

	// quiet suppresses non-essential output.
	quiet bool

	// verbose enables detailed operational output.
	verbose bool

	// force allows overwriting existing files.
	force bool

	// createdPathsByCategory tracks paths created during installation, grouped by category.
	createdPathsByCategory map[string][]string

	// overwrittenPaths tracks the paths that were overwritten.
	overwrittenPaths []string

	// suppressedPaths tracks paths that were suppressed via target overrides.
	suppressedPaths []string

	// resolvedProfile stores the resolved profile name for output.
	resolvedProfile string

	// projectConfigCreated indicates if .weftlo.yaml was created.
	projectConfigCreated bool

	// profileResolutionSource tracks where the profile was resolved from for verbose output.
	profileResolutionSource string

	// installPrefixResolutionSource tracks where the install prefix was resolved from for verbose output.
	installPrefixResolutionSource string
}

// projectConfigOutput represents the structure of .weftlo.yaml for writing.
type projectConfigOutput struct {
	Profiles      []string `yaml:"profiles"`
	InstallPrefix string   `yaml:"install_prefix,omitempty"`
}

// NewInstallCommand creates and returns the install subcommand.
// The filesystem is injected for testability.
// This uses the OS filesystem resolver for XDG Base Directory resolution.
func NewInstallCommand(fs afero.Fs) *cobra.Command {
	// Create a default resolver using the OS filesystem for XDG resolution
	resolver := infraconfig.NewResolver()
	return NewInstallCommandWithResolver(fs, os.UserHomeDir, os.Stdin, os.Stdout, resolver)
}

// NewInstallCommandWithHomeDir creates and returns the install subcommand with an injectable home directory function.
// This is primarily used for testing to provide a custom home directory.
// It uses a fallback resolver that always uses homeDir + "/.weftlo".
func NewInstallCommandWithHomeDir(fs afero.Fs, homeDir HomeDirFunc) *cobra.Command {
	return NewInstallCommandWithIO(fs, homeDir, os.Stdin, os.Stdout)
}

// NewInstallCommandWithIO creates and returns the install subcommand with injectable I/O streams.
// This allows tests to inject custom stdin/stdout for testing interactive prompts.
// It uses a fallback resolver that always uses homeDir + "/.weftlo".
func NewInstallCommandWithIO(fs afero.Fs, homeDir HomeDirFunc, stdin io.Reader, stdout io.Writer) *cobra.Command {
	// Create a simple resolver that uses the provided homeDir for backward compatibility with tests
	resolver := &homeDirResolver{homeDir: homeDir}
	return NewInstallCommandWithResolver(fs, homeDir, stdin, stdout, resolver)
}

// NewInstallCommandWithResolver creates and returns the install subcommand with a custom config directory resolver.
// This is primarily used for testing XDG behavior.
func NewInstallCommandWithResolver(fs afero.Fs, homeDir HomeDirFunc, stdin io.Reader, stdout io.Writer, resolver ConfigDirResolver) *cobra.Command {
	return NewInstallCommandWithProjectDir(fs, homeDir, stdin, stdout, resolver, "")
}

// NewInstallCommandForTesting creates the install subcommand with injectable project directory for testing.
// This allows tests to use an in-memory filesystem by specifying where the project files are.
func NewInstallCommandForTesting(fs afero.Fs, homeDir HomeDirFunc, stdin io.Reader, stdout io.Writer, projectDir string) *cobra.Command {
	resolver := &homeDirResolver{homeDir: homeDir}
	return NewInstallCommandWithProjectDir(fs, homeDir, stdin, stdout, resolver, projectDir)
}

// NewInstallCommandWithProjectDir creates and returns the install subcommand with a custom project directory.
// This is primarily used for testing where os.Getwd() cannot be used with an in-memory filesystem.
func NewInstallCommandWithProjectDir(fs afero.Fs, homeDir HomeDirFunc, stdin io.Reader, stdout io.Writer, resolver ConfigDirResolver, projectDir string) *cobra.Command {
	installCmd := &InstallCommand{
		fs:                     fs,
		homeDir:                homeDir,
		projectDir:             projectDir,
		configDirResolver:      resolver,
		stdin:                  stdin,
		stdout:                 stdout,
		createdPathsByCategory: make(map[string][]string),
		overwrittenPaths:       make([]string, 0),
		suppressedPaths:        make([]string, 0),
	}

	cmd := &cobra.Command{
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
		RunE: installCmd.run,
	}

	// Add --profile flag (repeatable for multiple profiles)
	cmd.Flags().StringSliceVarP(&installCmd.profiles, "profile", "p", nil, "Profile to install (vendor/name format, repeatable)")

	// Add --install-prefix flag
	cmd.Flags().StringVarP(&installCmd.installPrefix, "install-prefix", "", "", "Installation directory prefix")

	// Add --dry-run flag
	cmd.Flags().BoolVarP(&installCmd.dryRun, "dry-run", "n", false, "Preview changes without writing files")

	// Add --quiet flag
	cmd.Flags().BoolVarP(&installCmd.quiet, "quiet", "q", false, "Suppress non-essential output")

	// Add --verbose flag
	cmd.Flags().BoolVarP(&installCmd.verbose, "verbose", "v", false, "Enable verbose output")

	// Add --force flag
	cmd.Flags().BoolVarP(&installCmd.force, "force", "f", false, "Overwrite existing files")

	return cmd
}

// run is the RunE function for the install command.
func (c *InstallCommand) run(cmd *cobra.Command, args []string) error {
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
			return fmt.Errorf("failed to get current directory: %w", err)
		}
	}

	// Check if global config directory exists
	exists, err := afero.DirExists(c.fs, configDir)
	if err != nil {
		return &PermissionError{Path: configDir, Err: err}
	}
	if !exists {
		return &GlobalConfigNotFoundError{}
	}

	// Load global config
	globalConfigLoader := infraconfig.NewGlobalConfigLoader(c.fs)
	globalConfig, err := globalConfigLoader.Load(configDir)
	if err != nil {
		return fmt.Errorf("failed to load global config: %w", err)
	}

	// Load project config (may not exist)
	projectConfigLoader := infraconfig.NewProjectConfigLoader(c.fs)
	projectConfigData, projectConfigErr := projectConfigLoader.Load(projectDir)

	// Track whether project config exists
	projectConfigExists := true
	if projectConfigErr != nil {
		var configNotFoundErr *infraconfig.ConfigNotFoundError
		if errors.As(projectConfigErr, &configNotFoundErr) {
			projectConfigExists = false
		} else {
			// Some other error occurred
			return fmt.Errorf("failed to load project config: %w", projectConfigErr)
		}
	}

	// Install is for initial setup only. If .weftlo.yaml exists, fail with guidance.
	if projectConfigExists {
		return &AlreadyInstalledError{}
	}

	// Resolve profiles using resolution chain
	resolvedProfiles, err := c.resolveProfiles(projectConfigData, globalConfig, projectConfigExists)
	if err != nil {
		return err
	}
	// Store the leaf profile name (last in the list) for display
	c.resolvedProfile = resolvedProfiles[len(resolvedProfiles)-1]

	// Display verbose profile resolution
	c.displayVerboseProfileResolution()

	// Load the merged profiles using the resolved configDir for XDG support
	profileLoader := infraprofile.NewProfileLoaderWithConfigDir(c.fs, configDir)
	mergedProfile, err := profileLoader.LoadMergedMultiple(resolvedProfiles)
	if err != nil {
		return fmt.Errorf("failed to load profiles: %w", err)
	}

	// Pre-flight: catch *.tmpl files left at the profile root (outside content/).
	// The renderer ignores them, so without this check a misconfigured profile
	// would install successfully but produce zero rendered files.
	if profilesBase, perr := profileLoader.ProfilesBasePath(); perr == nil {
		if verr := infraprofile.ValidateChainLayout(c.fs, profilesBase, mergedProfile.ProfileNames(), mergedProfile.ContentConfig.Root); verr != nil {
			return verr
		}
	}

	// Display variable conflict warnings before installation begins
	if !c.quiet {
		displayVariableConflictWarnings(c.stdout, mergedProfile.VariableConflicts())
	}

	// Task 4.2: Load project-level ignore patterns and merge with profile patterns
	mergedMatcher := c.loadAndMergeIgnorePatterns(mergedProfile, projectDir)

	// Resolve install prefix using resolution chain
	resolvedInstallPrefix := c.resolveInstallPrefix(projectConfigData, globalConfig, projectConfigExists)

	// Display verbose install prefix resolution
	c.displayVerboseInstallPrefixResolution(resolvedInstallPrefix)

	// Create install service dependencies
	templateEngine := template.NewTemplateEngineWithFs(c.fs)
	pipeline := rendering.NewRenderingPipeline(templateEngine)
	manifestGenerator := manifest.NewManifestGenerator()

	// Create install service
	installService := install.NewService(c.fs, pipeline, manifestGenerator)

	// Task 4.3: Construct Router using MergedProfile.ContentConfig and ProjectConfig.TargetOverrides
	// Get the content config from the merged profile
	contentConfig := mergedProfile.ContentConfig

	// Apply default target from install prefix resolution if not set in profile
	if contentConfig.DefaultTarget == "" {
		contentConfig.DefaultTarget = resolvedInstallPrefix
	}

	// Get target overrides from project config (if available)
	var targetOverrides map[string]*string
	if projectConfigExists && projectConfigData != nil {
		targetOverrides = projectConfigData.TargetOverrides
	}

	// Merge variables using DeepMergeVariablesChain with precedence:
	// global (lowest) -> profile inheritance -> project (highest)
	mergedVariables, _ := rendering.DeepMergeVariablesChain(
		rendering.VariableSource{Name: "global config", Vars: globalConfig.Variables},
		rendering.VariableSource{Name: "profile inheritance", Vars: mergedProfile.Variables},
		rendering.VariableSource{Name: "project config", Vars: func() map[string]interface{} {
			if projectConfigExists && projectConfigData != nil {
				return projectConfigData.Variables
			}
			return nil
		}()},
	)

	router, err := routing.NewContentRouter(contentConfig, targetOverrides, mergedVariables)
	if err != nil {
		return fmt.Errorf("failed to create content router: %w", err)
	}

	// Before installation, check for file conflicts (unless --force or --dry-run)
	if !c.force && !c.dryRun {
		conflicts, err := c.checkForConflictsWithRouter(mergedProfile, projectDir, router, mergedMatcher)
		if err != nil {
			return err
		}
		if len(conflicts) > 0 {
			return &FileConflictError{ConflictingFiles: conflicts}
		}
	}

	// Track files that will be overwritten if --force is set
	if c.force && !c.dryRun {
		conflicts, err := c.checkForConflictsWithRouter(mergedProfile, projectDir, router, mergedMatcher)
		if err != nil {
			return err
		}
		c.overwrittenPaths = conflicts
	}

	// Configure install options
	opts := install.InstallOptions{
		DryRun: c.dryRun,
		Stdout: c.stdout,
	}

	// Display verbose file rendering details
	c.displayVerboseFileRenderingWithRouter(mergedProfile, router, mergedMatcher)

	// Execute installation
	result, err := installService.Install(mergedProfile, projectDir, router, opts)
	if err != nil {
		return fmt.Errorf("installation failed: %w", err)
	}

	// Track created paths for output (if not dry-run), grouped by category
	if !c.dryRun {
		c.collectCreatedPathsWithRouter(mergedProfile, projectDir, router, mergedMatcher)
	}

	// Create .weftlo.yaml if it doesn't exist (and not dry-run)
	if !projectConfigExists && !c.dryRun {
		if err := c.createProjectConfig(projectDir, resolvedProfiles); err != nil {
			return err
		}
		c.projectConfigCreated = true
	}

	// Display success output (unless quiet mode is enabled or dry-run)
	if !c.dryRun {
		c.displaySuccessOutput(projectDir, result)
	}

	// Zero-output notice: if the install produced no rendered files, the
	// .weftlo.yaml/.weftlo.manifest.json artifacts alone don't tell the user
	// why their project is empty. An intentionally-empty profile is valid, so
	// this is informational, not fatal.
	if !c.quiet && result.FilesProcessed == 0 {
		_, _ = fmt.Fprintf(c.stdout,
			"notice: no files were rendered from profile %q. Check that templates exist in %s/.\n",
			c.resolvedProfile, mergedProfile.ContentConfig.Root)
	}

	return nil
}

// loadAndMergeIgnorePatterns loads project-level ignore patterns and merges with profile patterns.
// Task 4.2: Implement project-level ignore loading in install command.
func (c *InstallCommand) loadAndMergeIgnorePatterns(mergedProfile *infraprofile.MergedProfile, projectDir string) domainignore.Matcher {
	// Start with profile's merged matcher
	profileMatcher := mergedProfile.Matcher
	if profileMatcher == nil {
		profileMatcher = domainignore.EmptyMatcher()
	}

	// Load project-level .weftlo.ignore file
	ignoreFilePath := filepath.Join(projectDir, ".weftlo.ignore")
	ignoreLoader := infraignore.NewLoader(c.fs)
	projectIgnoreResult, err := ignoreLoader.LoadFromFile(ignoreFilePath)

	// Handle nil gracefully when file does not exist (LoadFromFile handles this)
	if err != nil {
		// Log warning in verbose mode but continue without project ignores
		if c.verbose && !c.quiet {
			_, _ = fmt.Fprintf(c.stdout, "Warning: failed to load .weftlo.ignore: %v\n", err)
		}
		return profileMatcher
	}

	// Merge profile patterns first, then project patterns (profile patterns first, project patterns appended)
	if projectIgnoreResult.Matcher != nil {
		return profileMatcher.MergeWith(projectIgnoreResult.Matcher)
	}

	return profileMatcher
}

// resolveProfiles resolves the profile names using the resolution chain.
// Returns a slice of profile names to be merged in order.
func (c *InstallCommand) resolveProfiles(
	projectConfig *domainconfig.ProjectConfig,
	globalConfig *domainconfig.GlobalConfig,
	projectConfigExists bool,
) ([]string, error) {
	resolutionErr := &ProfileResolutionError{}

	// 1. Check --profile flag (takes precedence)
	if len(c.profiles) > 0 {
		resolutionErr.FlagChecked = true
		c.profileResolutionSource = "--profile flag"
		return c.profiles, nil
	}
	resolutionErr.FlagChecked = true

	// 2. Check project config profiles array
	if projectConfigExists && projectConfig != nil && len(projectConfig.Profiles) > 0 {
		resolutionErr.ProjectConfigChecked = true
		c.profileResolutionSource = "project config"
		return projectConfig.Profiles, nil
	}
	resolutionErr.ProjectConfigChecked = true

	// 3. Check global config default_profile (as single-element list)
	if globalConfig != nil && globalConfig.DefaultProfile != "" {
		resolutionErr.GlobalConfigChecked = true
		c.profileResolutionSource = "global config"
		return []string{globalConfig.DefaultProfile}, nil
	}
	resolutionErr.GlobalConfigChecked = true

	// No profile resolved
	return nil, resolutionErr
}

// resolveInstallPrefix resolves the install prefix using the resolution chain.
func (c *InstallCommand) resolveInstallPrefix(
	projectConfig *domainconfig.ProjectConfig,
	globalConfig *domainconfig.GlobalConfig,
	projectConfigExists bool,
) string {
	// 1. Check --install-prefix flag
	if c.installPrefix != "" {
		c.installPrefixResolutionSource = "--install-prefix flag"
		return c.installPrefix
	}

	// 2. Check project config
	if projectConfigExists && projectConfig != nil && projectConfig.InstallPrefix != "" {
		// Note: ProjectConfigLoader applies default, but we check if it was explicitly set
		// by comparing to the default value
		if projectConfig.InstallPrefix != domainconfig.DefaultInstallPrefix {
			c.installPrefixResolutionSource = "project config"
			return projectConfig.InstallPrefix
		}
	}

	// 3. Check global config
	if globalConfig != nil && globalConfig.InstallPrefix != "" {
		c.installPrefixResolutionSource = "global config"
		return globalConfig.InstallPrefix
	}

	// 4. Use default
	c.installPrefixResolutionSource = "default"
	return domainconfig.DefaultInstallPrefix
}

// displayVerboseProfileResolution prints verbose output about profile resolution.
func (c *InstallCommand) displayVerboseProfileResolution() {
	if !c.verbose || c.quiet {
		return
	}
	_, _ = fmt.Fprintf(c.stdout, "Using profile from: %s\n", c.profileResolutionSource)
}

// displayVerboseInstallPrefixResolution prints verbose output about install prefix resolution.
func (c *InstallCommand) displayVerboseInstallPrefixResolution(resolvedPrefix string) {
	if !c.verbose || c.quiet {
		return
	}
	_, _ = fmt.Fprintf(c.stdout, "Using install prefix from: %s (%s)\n", c.installPrefixResolutionSource, resolvedPrefix)
}

// displayVerboseFileRenderingWithRouter prints verbose output about file rendering using Router.
func (c *InstallCommand) displayVerboseFileRenderingWithRouter(mergedProfile *infraprofile.MergedProfile, router routing.Router, matcher domainignore.Matcher) {
	if !c.verbose || c.quiet {
		return
	}

	templateFiles := mergedProfile.TemplateFiles()
	for _, tf := range templateFiles {
		if tf.IsPartial {
			continue
		}

		// Strip .tmpl suffix for routing
		sourcePath := stripTmplSuffix(tf.SourcePath)

		// Check if ignored by matcher
		if matcher != nil && matcher.Match(sourcePath, false) {
			_, _ = fmt.Fprintf(c.stdout, "Skipping (ignored): %s\n", tf.SourcePath)
			continue
		}

		// Route the file
		routedFile, err := router.Route(sourcePath)
		if err != nil {
			_, _ = fmt.Fprintf(c.stdout, "Routing error for %s: %v\n", tf.SourcePath, err)
			continue
		}

		if routedFile.Suppressed {
			_, _ = fmt.Fprintf(c.stdout, "Suppressed (target override): %s (category: %s)\n", tf.SourcePath, routedFile.TargetCategory)
			continue
		}

		_, _ = fmt.Fprintf(c.stdout, "Rendering: %s -> %s\n", tf.SourcePath, routedFile.TargetPath)
	}
}

// checkForConflictsWithRouter checks if any target files already exist using Router-based paths.
// Task 4.5: Replace computeInstallTargetPath() usage with Router-based path resolution.
func (c *InstallCommand) checkForConflictsWithRouter(
	mergedProfile *infraprofile.MergedProfile,
	projectDir string,
	router routing.Router,
	matcher domainignore.Matcher,
) ([]string, error) {
	var conflicts []string

	templateFiles := mergedProfile.TemplateFiles()
	for _, tf := range templateFiles {
		if tf.IsPartial {
			continue
		}

		// Strip .tmpl suffix for routing
		sourcePath := stripTmplSuffix(tf.SourcePath)

		// Check if ignored by matcher
		if matcher != nil && matcher.Match(sourcePath, false) {
			continue
		}

		// Route the file
		routedFile, err := router.Route(sourcePath)
		if err != nil {
			return nil, fmt.Errorf("routing failed for %s: %w", tf.SourcePath, err)
		}

		// Skip suppressed files
		if routedFile.Suppressed {
			continue
		}

		absolutePath := filepath.Join(projectDir, routedFile.TargetPath)
		exists, err := afero.Exists(c.fs, absolutePath)
		if err != nil {
			return nil, &PermissionError{Path: absolutePath, Err: err}
		}
		if exists {
			conflicts = append(conflicts, routedFile.TargetPath)
		}
	}

	return conflicts, nil
}

// collectCreatedPathsWithRouter collects all paths that were created during installation,
// grouped by TargetCategory.
// Task 4.5: Replace computeInstallTargetPath() usage with Router-based path resolution.
func (c *InstallCommand) collectCreatedPathsWithRouter(
	mergedProfile *infraprofile.MergedProfile,
	projectDir string,
	router routing.Router,
	matcher domainignore.Matcher,
) {
	templateFiles := mergedProfile.TemplateFiles()
	for _, tf := range templateFiles {
		if tf.IsPartial {
			continue
		}

		// Strip .tmpl suffix for routing
		sourcePath := stripTmplSuffix(tf.SourcePath)

		// Check if ignored by matcher
		if matcher != nil && matcher.Match(sourcePath, false) {
			continue
		}

		// Route the file
		routedFile, err := router.Route(sourcePath)
		if err != nil {
			continue
		}

		// Track suppressed files for verbose output
		if routedFile.Suppressed {
			c.suppressedPaths = append(c.suppressedPaths, sourcePath+" ("+routedFile.TargetCategory+")")
			continue
		}

		// Group by target category
		category := routedFile.TargetCategory
		if category == "" {
			// Use the first directory segment of the target path as category
			parts := strings.Split(routedFile.TargetPath, "/")
			if len(parts) > 0 {
				category = parts[0]
			}
		}

		// Build category display name from target path prefix
		categoryDisplay := getCategoryDisplayPath(routedFile.TargetPath, category)

		if c.createdPathsByCategory[categoryDisplay] == nil {
			c.createdPathsByCategory[categoryDisplay] = make([]string, 0)
		}
		c.createdPathsByCategory[categoryDisplay] = append(c.createdPathsByCategory[categoryDisplay], routedFile.TargetPath)
	}
}

// getCategoryDisplayPath extracts the category directory path from a target path.
// For example, ".claude/skills/coding.md" returns ".claude/skills/".
func getCategoryDisplayPath(targetPath, category string) string {
	// Find the category in the path and extract the directory portion
	parts := strings.Split(targetPath, "/")
	if len(parts) <= 1 {
		return category + "/"
	}

	// Find where the file starts (last segment)
	dirPath := strings.Join(parts[:len(parts)-1], "/")
	return dirPath + "/"
}

// stripTmplSuffix removes the .tmpl suffix from a source path if present.
func stripTmplSuffix(sourcePath string) string {
	const tmplSuffix = ".tmpl"
	if strings.HasSuffix(sourcePath, tmplSuffix) {
		return sourcePath[:len(sourcePath)-len(tmplSuffix)]
	}
	return sourcePath
}

// createProjectConfig creates the .weftlo.yaml file.
func (c *InstallCommand) createProjectConfig(projectDir string, profileNames []string) error {
	configPath := filepath.Join(projectDir, ".weftlo.yaml")

	config := projectConfigOutput{
		Profiles: profileNames,
	}

	// Include install_prefix only if --install-prefix flag was specified
	if c.installPrefix != "" {
		config.InstallPrefix = c.installPrefix
	}

	data, err := yaml.Marshal(&config)
	if err != nil {
		return fmt.Errorf("failed to marshal project config: %w", err)
	}

	if err := afero.WriteFile(c.fs, configPath, data, 0644); err != nil {
		return &PermissionError{Path: configPath, Err: err}
	}

	return nil
}

// displaySuccessOutput displays the success message with created files grouped by TargetCategory.
// Task 4.4: Update displaySuccessOutput() to group files by TargetCategory.
// This output is suppressed when --quiet flag is set.
func (c *InstallCommand) displaySuccessOutput(projectDir string, result *install.InstallResult) {
	if c.quiet {
		return
	}

	_, _ = fmt.Fprintf(c.stdout, "Installed profile '%s' successfully!\n", c.resolvedProfile)
	_, _ = fmt.Fprintln(c.stdout)

	// Display files grouped by category
	if len(c.createdPathsByCategory) > 0 {
		// Get sorted category keys for deterministic output
		categories := make([]string, 0, len(c.createdPathsByCategory))
		for category := range c.createdPathsByCategory {
			categories = append(categories, category)
		}
		sort.Strings(categories)

		for _, category := range categories {
			paths := c.createdPathsByCategory[category]
			_, _ = fmt.Fprintf(c.stdout, "Files installed to %s (%d):\n", category, len(paths))

			// Sort paths within category for deterministic output
			sort.Strings(paths)
			for _, path := range paths {
				// Check if this file was overwritten
				overwritten := false
				for _, op := range c.overwrittenPaths {
					if op == path {
						overwritten = true
						break
					}
				}
				if overwritten {
					_, _ = fmt.Fprintf(c.stdout, "  %s (overwritten)\n", path)
				} else {
					_, _ = fmt.Fprintf(c.stdout, "  %s\n", path)
				}
			}
		}
		_, _ = fmt.Fprintln(c.stdout)
	}

	// Show suppressed files only with --verbose flag
	// Task 4.4: Show suppressed/skipped files only when --verbose flag enabled.
	if c.verbose && len(c.suppressedPaths) > 0 {
		_, _ = fmt.Fprintf(c.stdout, "Files skipped (target suppressed) (%d):\n", len(c.suppressedPaths))
		for _, path := range c.suppressedPaths {
			_, _ = fmt.Fprintf(c.stdout, "  %s\n", path)
		}
		_, _ = fmt.Fprintln(c.stdout)
	}

	if c.projectConfigCreated {
		_, _ = fmt.Fprintln(c.stdout, "Created .weftlo.yaml with profile configuration.")
		_, _ = fmt.Fprintln(c.stdout)
	}

	_, _ = fmt.Fprintln(c.stdout, "Next steps:")
	_, _ = fmt.Fprintln(c.stdout, "  Run `weftlo status` to see installation state")
}
