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

	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/simensen/weftlo/internal/app/rendering"
	"github.com/simensen/weftlo/internal/app/routing"
	"github.com/simensen/weftlo/internal/app/update"
	domainconfig "github.com/simensen/weftlo/internal/domain/config"
	"github.com/simensen/weftlo/internal/domain/manifest"
	"github.com/simensen/weftlo/internal/domain/template"
	infraconfig "github.com/simensen/weftlo/internal/infrastructure/config"
	infraprofile "github.com/simensen/weftlo/internal/infrastructure/profile"
)

// UpdateCommand represents the update subcommand configuration.
// This type is exported for testing purposes.
type UpdateCommand struct {
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

	// dryRun when true prevents any filesystem writes.
	dryRun bool

	// quiet suppresses non-essential output.
	quiet bool

	// verbose enables detailed operational output.
	verbose bool

	// force allows overwriting user-modified and conflicting files.
	force bool

	// removeOrphans when true deletes files no longer in the profile.
	removeOrphans bool

	// addProfiles is the list of profiles to add via --add-profile flag (repeatable).
	addProfiles []string

	// removeProfiles is the list of profiles to remove via --remove-profile flag (repeatable).
	removeProfiles []string

	// profilesModified indicates if the profile list was changed during this update.
	profilesModified bool

	// createdCount tracks the number of files created during update.
	createdCount int

	// updatedCount tracks the number of files updated during update.
	updatedCount int

	// skippedFiles tracks the files that were skipped with their rationales.
	skippedFiles []skippedFile

	// deletedCount tracks the number of files deleted during update.
	deletedCount int

	// createdPathsByCategory tracks created paths grouped by target category for output.
	createdPathsByCategory map[string][]string

	// updatedPathsByCategory tracks updated paths grouped by target category for output.
	updatedPathsByCategory map[string][]string
}

// skippedFile represents a file that was skipped during update with its rationale.
type skippedFile struct {
	path      string
	rationale string
}

// NewUpdateCommand creates and returns the update subcommand.
// The filesystem is injected for testability.
// This uses the OS filesystem resolver for XDG Base Directory resolution.
func NewUpdateCommand(fs afero.Fs) *cobra.Command {
	// Create a default resolver using the OS filesystem for XDG resolution
	resolver := infraconfig.NewResolver()
	return NewUpdateCommandWithResolver(fs, os.UserHomeDir, os.Stdin, os.Stdout, resolver)
}

// NewUpdateCommandWithHomeDir creates and returns the update subcommand with an injectable home directory function.
// This is primarily used for testing to provide a custom home directory.
// It uses a fallback resolver that always uses homeDir + "/.weftlo".
func NewUpdateCommandWithHomeDir(fs afero.Fs, homeDir HomeDirFunc) *cobra.Command {
	return NewUpdateCommandWithIO(fs, homeDir, os.Stdin, os.Stdout)
}

// NewUpdateCommandForTesting creates the update subcommand with injectable project directory for testing.
// This allows tests to use an in-memory filesystem by specifying where the project files are.
func NewUpdateCommandForTesting(fs afero.Fs, homeDir HomeDirFunc, stdin io.Reader, stdout io.Writer, projectDir string) *cobra.Command {
	resolver := &homeDirResolver{homeDir: homeDir}
	return NewUpdateCommandWithProjectDir(fs, homeDir, stdin, stdout, resolver, projectDir)
}

// NewUpdateCommandWithIO creates and returns the update subcommand with injectable I/O streams.
// This allows tests to inject custom stdin/stdout for testing interactive prompts.
// It uses a fallback resolver that always uses homeDir + "/.weftlo".
func NewUpdateCommandWithIO(fs afero.Fs, homeDir HomeDirFunc, stdin io.Reader, stdout io.Writer) *cobra.Command {
	// Create a simple resolver that uses the provided homeDir for backward compatibility with tests
	resolver := &homeDirResolver{homeDir: homeDir}
	return NewUpdateCommandWithResolver(fs, homeDir, stdin, stdout, resolver)
}

// NewUpdateCommandWithResolver creates and returns the update subcommand with a custom config directory resolver.
// This is primarily used for testing XDG behavior.
func NewUpdateCommandWithResolver(fs afero.Fs, homeDir HomeDirFunc, stdin io.Reader, stdout io.Writer, resolver ConfigDirResolver) *cobra.Command {
	return NewUpdateCommandWithProjectDir(fs, homeDir, stdin, stdout, resolver, "")
}

// NewUpdateCommandWithProjectDir creates and returns the update subcommand with a custom project directory.
// This is primarily used for testing where os.Getwd() cannot be used with an in-memory filesystem.
func NewUpdateCommandWithProjectDir(fs afero.Fs, homeDir HomeDirFunc, stdin io.Reader, stdout io.Writer, resolver ConfigDirResolver, projectDir string) *cobra.Command {
	updateCmd := &UpdateCommand{
		fs:                     fs,
		homeDir:                homeDir,
		projectDir:             projectDir,
		configDirResolver:      resolver,
		stdin:                  stdin,
		stdout:                 stdout,
		skippedFiles:           make([]skippedFile, 0),
		createdPathsByCategory: make(map[string][]string),
		updatedPathsByCategory: make(map[string][]string),
	}

	cmd := &cobra.Command{
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
		RunE: updateCmd.run,
	}

	// Add --force flag
	cmd.Flags().BoolVarP(&updateCmd.force, "force", "f", false, "Overwrite user-modified and conflicting files")

	// Add --dry-run flag
	cmd.Flags().BoolVarP(&updateCmd.dryRun, "dry-run", "n", false, "Preview changes without writing files")

	// Add --quiet flag
	cmd.Flags().BoolVarP(&updateCmd.quiet, "quiet", "q", false, "Suppress non-essential output")

	// Add --verbose flag
	cmd.Flags().BoolVarP(&updateCmd.verbose, "verbose", "v", false, "Enable verbose output")

	// Add --remove-orphans flag (no short form per spec)
	cmd.Flags().BoolVar(&updateCmd.removeOrphans, "remove-orphans", false, "Delete files no longer in the profile")

	// Add --add-profile flag (repeatable for adding multiple profiles)
	cmd.Flags().StringSliceVar(&updateCmd.addProfiles, "add-profile", nil, "Add profile(s) to the installation (repeatable)")

	// Add --remove-profile flag (repeatable for removing multiple profiles)
	cmd.Flags().StringSliceVar(&updateCmd.removeProfiles, "remove-profile", nil, "Remove profile(s) from the installation (repeatable)")

	return cmd
}

// run is the RunE function for the update command.
func (c *UpdateCommand) run(cmd *cobra.Command, args []string) error {
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

	// Task 2.2: Check for .weftlo.yaml existence
	projectConfigPath := filepath.Join(projectDir, ".weftlo.yaml")
	projectConfigExists, err := afero.Exists(c.fs, projectConfigPath)
	if err != nil {
		return &PermissionError{Path: projectConfigPath, Err: err}
	}
	if !projectConfigExists {
		return &InstallationNotFoundError{MissingFile: ".weftlo.yaml"}
	}

	// Task 2.2: Check for .weftlo.manifest.json existence
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

	// Extract profile names from project config and apply modifications
	profileNames := projectConfig.Profiles

	// Apply profile modifications if --add-profile or --remove-profile flags were used
	if len(c.removeProfiles) > 0 || len(c.addProfiles) > 0 {
		var err error
		profileNames, err = c.computeModifiedProfiles(profileNames)
		if err != nil {
			return err
		}
		c.profilesModified = true

		// Update .weftlo.yaml with the new profile list (unless dry-run)
		if !c.dryRun {
			if err := c.updateProjectConfigProfiles(projectDir, projectConfig, profileNames); err != nil {
				return err
			}
		}
	}

	// Handle empty profile list - nothing to update
	if len(profileNames) == 0 {
		if !c.quiet {
			_, _ = fmt.Fprintln(c.stdout, "No profiles configured. Nothing to update.")
			if c.profilesModified {
				_, _ = fmt.Fprintln(c.stdout, "Previously managed files are now orphans.")
				_, _ = fmt.Fprintln(c.stdout, "Run 'weftlo update --remove-orphans' with at least one profile to clean them up,")
				_, _ = fmt.Fprintln(c.stdout, "or delete them manually.")
			}
		}
		return nil
	}

	// Task 2.4: Validate profile name format before loading (validation is done by loader)

	// Task 2.3: Load manifest from .weftlo.manifest.json
	manifestData, err := afero.ReadFile(c.fs, manifestPath)
	if err != nil {
		return fmt.Errorf("failed to read manifest: %w", err)
	}

	var existingManifest manifest.Manifest
	if err := json.Unmarshal(manifestData, &existingManifest); err != nil {
		return fmt.Errorf("failed to parse manifest: %w", err)
	}

	// Task 2.4: Load global config (for potential future use)
	globalConfigLoader := infraconfig.NewGlobalConfigLoader(c.fs)
	globalConfig, err := globalConfigLoader.Load(configDir)
	if err != nil {
		return fmt.Errorf("failed to load global config: %w", err)
	}

	// Task 2.4: Load merged profiles using ProfileLoaderWithConfigDir
	profileLoader := infraprofile.NewProfileLoaderWithConfigDir(c.fs, configDir)
	mergedProfile, err := profileLoader.LoadMergedMultiple(profileNames)
	if err != nil {
		// Check if it's a profile not found error
		var profileNotFoundErr *infraprofile.ProfileNotFoundError
		if errors.As(err, &profileNotFoundErr) {
			return &ProfileNotFoundError{
				ProfileName: profileNotFoundErr.ProfileName,
				Err:         err,
			}
		}
		return fmt.Errorf("failed to load profiles: %w", err)
	}

	// Pre-flight: misplaced template files at the profile root are silently
	// ignored by the renderer. Fail loudly here rather than report a misleading
	// "no changes" update.
	if profilesBase, perr := profileLoader.ProfilesBasePath(); perr == nil {
		if verr := infraprofile.ValidateChainLayout(c.fs, profilesBase, mergedProfile.ProfileNames(), mergedProfile.ContentConfig.Root); verr != nil {
			return verr
		}
	}

	// Display variable conflict warnings before update begins
	if !c.quiet {
		displayVariableConflictWarnings(c.stdout, mergedProfile.VariableConflicts(), c.verbose)
	}

	// Task 4.6: Construct Router using same pattern as install command
	// Get the content config from the merged profile
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
		return fmt.Errorf("failed to create content router: %w", err)
	}

	// Task 4.6: Pass Router to DetectChanges() instead of installPrefix string
	changeResults, err := manifest.DetectChanges(&existingManifest, mergedProfile, c.fs, projectDir, router)
	if err != nil {
		return fmt.Errorf("failed to detect changes: %w", err)
	}

	// Display verbose change detection output
	c.displayVerboseChangeDetection(changeResults)

	// Task 3.3: Create UpdateOptions from CLI flags
	updateOptions := update.UpdateOptions{
		Force:         c.force,
		RemoveOrphans: c.removeOrphans,
	}

	// Task 3.3: Create template engine for rendering
	engine := template.NewTemplateEngineWithFs(c.fs)

	// Task 3.3: Call update.PlanUpdate with change results
	// Convert FileChangeInfo map to FileStatus map for compatibility
	statusResults := make(map[string]manifest.FileStatus)
	for path, info := range changeResults {
		statusResults[path] = info.Status
	}
	plan, err := update.PlanUpdate(statusResults, mergedProfile, updateOptions, engine, projectDir)
	if err != nil {
		return fmt.Errorf("failed to plan update: %w", err)
	}

	// Task 3.4: Implement early exit for no changes
	if c.isPlanEmpty(plan) {
		if !c.quiet {
			_, _ = fmt.Fprintln(c.stdout, "No changes needed.")
		}
		return nil
	}

	// Task 4.5 / 5.5: Display dry-run summary and skip file operations if dry-run
	if c.dryRun {
		if !c.quiet {
			c.displayDryRunSummary(plan, changeResults)
		}
		return nil
	}

	// Task 4.7: Execute file operations using Router-based paths
	if err := c.executeFileOperationsWithRouter(plan, mergedProfile, projectDir, router, changeResults); err != nil {
		return err
	}

	// Display verbose manifest regeneration message
	c.displayVerboseManifestRegeneration()

	// Task 4.7: Regenerate manifest with TargetCategory populated
	if err := c.regenerateManifestWithRouter(mergedProfile, projectDir, router); err != nil {
		return err
	}

	// Task 5.3 / 5.4: Display success output (unless quiet mode)
	c.displaySuccessOutput()

	return nil
}

// displayVerboseChangeDetection prints verbose output about change detection.
func (c *UpdateCommand) displayVerboseChangeDetection(changeResults map[string]manifest.FileChangeInfo) {
	if !c.verbose || c.quiet {
		return
	}

	// Sort the file paths for consistent output
	filePaths := make([]string, 0, len(changeResults))
	for path := range changeResults {
		filePaths = append(filePaths, path)
	}
	sort.Strings(filePaths)

	for _, path := range filePaths {
		info := changeResults[path]
		_, _ = fmt.Fprintf(c.stdout, "Checking file: %s (status: %s, category: %s)\n", path, info.Status, info.TargetCategory)
	}
}

// displayVerboseManifestRegeneration prints verbose output about manifest regeneration.
func (c *UpdateCommand) displayVerboseManifestRegeneration() {
	if !c.verbose || c.quiet {
		return
	}
	_, _ = fmt.Fprintln(c.stdout, "Regenerating manifest...")
}

// executeFileOperationsWithRouter performs all file create, update, and delete operations
// using Router-based paths.
// Task 4.7: Update executeFileOperations() to use Router-based paths.
func (c *UpdateCommand) executeFileOperationsWithRouter(
	plan update.UpdatePlan,
	mergedProfile *infraprofile.MergedProfile,
	projectDir string,
	router routing.Router,
	changeResults map[string]manifest.FileChangeInfo,
) error {
	// Re-render templates using the rendering pipeline
	engine := template.NewTemplateEngineWithFs(c.fs)
	pipeline := rendering.NewRenderingPipeline(engine)
	renderResults, err := pipeline.RenderAllWithOptions(mergedProfile, projectDir, nil)
	if err != nil {
		return fmt.Errorf("failed to render templates: %w", err)
	}

	// Build a map of rendered content by routed target path for efficient lookup
	renderedByPath := make(map[string]string)
	for _, result := range renderResults {
		// Route the file to get the actual target path
		routedFile, err := router.Route(result.TargetPath)
		if err != nil {
			continue
		}
		if routedFile.Suppressed {
			continue
		}
		renderedByPath[routedFile.TargetPath] = result.Content
	}

	// Task 4.2: Implement file creation operations
	for _, decision := range plan.ToCreate {
		// Display verbose output for file creation
		if c.verbose && !c.quiet {
			_, _ = fmt.Fprintf(c.stdout, "Creating: %s\n", decision.TargetPath)
		}
		if err := c.createFile(decision, renderedByPath, projectDir); err != nil {
			return err
		}
		c.createdCount++

		// Track by category for grouped output
		if info, ok := changeResults[decision.TargetPath]; ok {
			category := getCategoryDisplayPath(decision.TargetPath, info.TargetCategory)
			c.createdPathsByCategory[category] = append(c.createdPathsByCategory[category], decision.TargetPath)
		}
	}

	// Task 4.3: Implement file update operations
	for _, decision := range plan.ToUpdate {
		// Display verbose output for file update
		if c.verbose && !c.quiet {
			_, _ = fmt.Fprintf(c.stdout, "Updating: %s\n", decision.TargetPath)
		}
		if err := c.updateFile(decision, renderedByPath, projectDir); err != nil {
			return err
		}
		c.updatedCount++

		// Track by category for grouped output
		if info, ok := changeResults[decision.TargetPath]; ok {
			category := getCategoryDisplayPath(decision.TargetPath, info.TargetCategory)
			c.updatedPathsByCategory[category] = append(c.updatedPathsByCategory[category], decision.TargetPath)
		}
	}

	// Task 4.4: Implement file deletion operations
	for _, decision := range plan.ToDelete {
		// Display verbose output for file deletion
		if c.verbose && !c.quiet {
			_, _ = fmt.Fprintf(c.stdout, "Deleting: %s\n", decision.TargetPath)
		}
		if err := c.deleteFile(decision, projectDir); err != nil {
			return err
		}
		c.deletedCount++
	}

	// Track skipped files
	for _, decision := range plan.ToSkip {
		c.skippedFiles = append(c.skippedFiles, skippedFile{
			path:      decision.TargetPath,
			rationale: decision.Rationale,
		})
	}

	return nil
}

// createFile creates a new file with parent directory creation.
// Parent directories are created with 0755 permissions.
// Files are created with 0644 permissions.
func (c *UpdateCommand) createFile(decision update.FileDecision, renderedByPath map[string]string, projectDir string) error {
	targetPath := decision.TargetPath
	absolutePath := filepath.Join(projectDir, targetPath)

	// Get rendered content
	content, ok := renderedByPath[targetPath]
	if !ok {
		return fmt.Errorf("no rendered content found for %s", targetPath)
	}

	// Create parent directories with 0755 permissions
	parentDir := filepath.Dir(absolutePath)
	if err := c.fs.MkdirAll(parentDir, 0755); err != nil {
		return &PermissionError{Path: parentDir, Err: err}
	}

	// Write file content with 0644 permissions
	if err := afero.WriteFile(c.fs, absolutePath, []byte(content), 0644); err != nil {
		return &PermissionError{Path: absolutePath, Err: err}
	}

	return nil
}

// updateFile updates an existing file with new content.
// Files are written with 0644 permissions (directories should already exist).
func (c *UpdateCommand) updateFile(decision update.FileDecision, renderedByPath map[string]string, projectDir string) error {
	targetPath := decision.TargetPath
	absolutePath := filepath.Join(projectDir, targetPath)

	// Get rendered content
	content, ok := renderedByPath[targetPath]
	if !ok {
		return fmt.Errorf("no rendered content found for %s", targetPath)
	}

	// Write file content with 0644 permissions (directories should already exist)
	if err := afero.WriteFile(c.fs, absolutePath, []byte(content), 0644); err != nil {
		return &PermissionError{Path: absolutePath, Err: err}
	}

	return nil
}

// deleteFile removes a file from the filesystem.
// Does NOT remove empty parent directories (per spec).
func (c *UpdateCommand) deleteFile(decision update.FileDecision, projectDir string) error {
	targetPath := decision.TargetPath
	absolutePath := filepath.Join(projectDir, targetPath)

	// Remove the file using fs.Remove()
	if err := c.fs.Remove(absolutePath); err != nil {
		return &PermissionError{Path: absolutePath, Err: err}
	}

	return nil
}

// isPlanEmpty checks if the update plan has no files in any category.
func (c *UpdateCommand) isPlanEmpty(plan update.UpdatePlan) bool {
	return len(plan.ToCreate) == 0 &&
		len(plan.ToUpdate) == 0 &&
		len(plan.ToSkip) == 0 &&
		len(plan.ToDelete) == 0
}

// regenerateManifestWithRouter creates a new manifest reflecting the post-update state.
// Task 4.7: Update regenerateManifest() to populate TargetCategory in new entries.
func (c *UpdateCommand) regenerateManifestWithRouter(mergedProfile *infraprofile.MergedProfile, projectDir string, router routing.Router) error {
	// Re-render templates to get current state for manifest
	engine := template.NewTemplateEngineWithFs(c.fs)
	pipeline := rendering.NewRenderingPipeline(engine)
	renderResults, err := pipeline.RenderAllWithOptions(mergedProfile, projectDir, nil)
	if err != nil {
		return fmt.Errorf("failed to render templates for manifest: %w", err)
	}

	// Build RenderedFile slice for manifest generation
	renderedFiles := make([]manifest.RenderedFile, 0, len(renderResults))
	profileName := mergedProfile.LeafProfileName()

	for _, result := range renderResults {
		// Route the file to get the actual target path and category
		routedFile, err := router.Route(result.TargetPath)
		if err != nil {
			continue
		}

		// Skip suppressed files
		if routedFile.Suppressed {
			continue
		}

		// Read the source template content for accurate checksum
		absPath, found := mergedProfile.Resolve(result.SourcePath)
		var sourceContent []byte
		if found {
			sourceContent, _ = afero.ReadFile(c.fs, absPath)
		}
		if sourceContent == nil {
			// Fallback: use output content for source if source not readable
			sourceContent = []byte(result.Content)
		}

		// Task 4.7: Populate TargetCategory from RoutedFile
		renderedFile := manifest.RenderedFile{
			TargetPath:     routedFile.TargetPath,
			SourceContent:  sourceContent,
			OutputContent:  []byte(result.Content),
			SourceProfile:  profileName,
			TargetCategory: routedFile.TargetCategory,
		}
		renderedFiles = append(renderedFiles, renderedFile)
	}

	// Create manifest generator and generate new manifest
	manifestGenerator := manifest.NewManifestGenerator()
	newManifest, err := manifestGenerator.Generate(mergedProfile.ProfileNames(), renderedFiles)
	if err != nil {
		return fmt.Errorf("failed to generate manifest: %w", err)
	}

	// Write manifest to .weftlo.manifest.json with JSON indentation
	manifestPath := filepath.Join(projectDir, ".weftlo.manifest.json")
	manifestContent, err := json.MarshalIndent(newManifest, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to serialize manifest: %w", err)
	}

	if err := afero.WriteFile(c.fs, manifestPath, manifestContent, 0644); err != nil {
		return &PermissionError{Path: manifestPath, Err: err}
	}

	return nil
}

// displaySuccessOutput shows a summary of the update operation.
// Task 4.7: Update displaySuccessOutput() for grouped output.
// This is suppressed when --quiet flag is set.
func (c *UpdateCommand) displaySuccessOutput() {
	// Task 5.4: Check quiet flag before output
	if c.quiet {
		return
	}

	// Task 5.3: Display summary header
	_, _ = fmt.Fprintln(c.stdout, "Update completed successfully!")
	_, _ = fmt.Fprintln(c.stdout)

	// Display created files grouped by category
	if c.createdCount > 0 {
		if len(c.createdPathsByCategory) > 0 {
			categories := make([]string, 0, len(c.createdPathsByCategory))
			for category := range c.createdPathsByCategory {
				categories = append(categories, category)
			}
			sort.Strings(categories)

			for _, category := range categories {
				paths := c.createdPathsByCategory[category]
				_, _ = fmt.Fprintf(c.stdout, "Files created in %s (%d):\n", category, len(paths))
				sort.Strings(paths)
				for _, path := range paths {
					_, _ = fmt.Fprintf(c.stdout, "  %s\n", path)
				}
			}
		} else {
			_, _ = fmt.Fprintf(c.stdout, "Files created: %d\n", c.createdCount)
		}
	}

	// Display updated files grouped by category
	if c.updatedCount > 0 {
		if len(c.updatedPathsByCategory) > 0 {
			categories := make([]string, 0, len(c.updatedPathsByCategory))
			for category := range c.updatedPathsByCategory {
				categories = append(categories, category)
			}
			sort.Strings(categories)

			for _, category := range categories {
				paths := c.updatedPathsByCategory[category]
				_, _ = fmt.Fprintf(c.stdout, "Files updated in %s (%d):\n", category, len(paths))
				sort.Strings(paths)
				for _, path := range paths {
					_, _ = fmt.Fprintf(c.stdout, "  %s\n", path)
				}
			}
		} else {
			_, _ = fmt.Fprintf(c.stdout, "Files updated: %d\n", c.updatedCount)
		}
	}

	// Task 5.3: Display count of files deleted (if any)
	if c.deletedCount > 0 {
		_, _ = fmt.Fprintf(c.stdout, "Files deleted: %d\n", c.deletedCount)
	}

	// Task 5.3: Display count of files skipped with rationale for each
	if len(c.skippedFiles) > 0 {
		_, _ = fmt.Fprintf(c.stdout, "Files skipped: %d\n", len(c.skippedFiles))
		for _, sf := range c.skippedFiles {
			_, _ = fmt.Fprintf(c.stdout, "  - %s: %s\n", sf.path, sf.rationale)
		}
	}

	// Display total if any changes were made
	total := c.createdCount + c.updatedCount + c.deletedCount
	if total == 0 && len(c.skippedFiles) == 0 {
		_, _ = fmt.Fprintln(c.stdout, "No changes were made.")
	}
}

// displayDryRunSummary shows what changes would be made without making them.
// Task 4.7 / 5.5: Dry-run output formatting for grouped output
func (c *UpdateCommand) displayDryRunSummary(plan update.UpdatePlan, changeResults map[string]manifest.FileChangeInfo) {
	_, _ = fmt.Fprintln(c.stdout, "[dry-run] Changes that would be made:")
	_, _ = fmt.Fprintln(c.stdout)

	if len(plan.ToCreate) > 0 {
		// Group by category for display
		createByCategory := make(map[string][]string)
		for _, decision := range plan.ToCreate {
			category := ""
			if info, ok := changeResults[decision.TargetPath]; ok {
				category = getCategoryDisplayPath(decision.TargetPath, info.TargetCategory)
			}
			if category == "" {
				category = getDefaultCategory(decision.TargetPath)
			}
			createByCategory[category] = append(createByCategory[category], decision.TargetPath)
		}

		categories := make([]string, 0, len(createByCategory))
		for cat := range createByCategory {
			categories = append(categories, cat)
		}
		sort.Strings(categories)

		_, _ = fmt.Fprintf(c.stdout, "Files to create: %d\n", len(plan.ToCreate))
		for _, category := range categories {
			paths := createByCategory[category]
			_, _ = fmt.Fprintf(c.stdout, "  [%s]\n", strings.TrimSuffix(category, "/"))
			sort.Strings(paths)
			for _, path := range paths {
				_, _ = fmt.Fprintf(c.stdout, "    - %s\n", path)
			}
		}
	}

	if len(plan.ToUpdate) > 0 {
		_, _ = fmt.Fprintf(c.stdout, "Files to update: %d\n", len(plan.ToUpdate))
		for _, decision := range plan.ToUpdate {
			_, _ = fmt.Fprintf(c.stdout, "  - %s\n", decision.TargetPath)
		}
	}

	if len(plan.ToSkip) > 0 {
		_, _ = fmt.Fprintf(c.stdout, "Files to skip: %d\n", len(plan.ToSkip))
		for _, decision := range plan.ToSkip {
			_, _ = fmt.Fprintf(c.stdout, "  - %s: %s\n", decision.TargetPath, decision.Rationale)
		}
	}

	if len(plan.ToDelete) > 0 {
		_, _ = fmt.Fprintf(c.stdout, "Files to delete: %d\n", len(plan.ToDelete))
		for _, decision := range plan.ToDelete {
			_, _ = fmt.Fprintf(c.stdout, "  - %s\n", decision.TargetPath)
		}
	}

	_, _ = fmt.Fprintln(c.stdout)
	_, _ = fmt.Fprintln(c.stdout, "No changes were made (dry-run mode).")
}

// getDefaultCategory extracts a default category from a target path when no category info is available.
func getDefaultCategory(targetPath string) string {
	parts := strings.Split(targetPath, "/")
	if len(parts) <= 1 {
		return ""
	}
	return strings.Join(parts[:len(parts)-1], "/") + "/"
}

// computeModifiedProfiles computes the new profile list after applying --add-profile and --remove-profile.
// Order of operations: removes are applied first, then adds.
func (c *UpdateCommand) computeModifiedProfiles(currentProfiles []string) ([]string, error) {
	// Start with a copy of current profiles
	profiles := make([]string, len(currentProfiles))
	copy(profiles, currentProfiles)

	// Apply removes first
	for _, toRemove := range c.removeProfiles {
		found := false
		newProfiles := make([]string, 0, len(profiles))
		for _, p := range profiles {
			if p == toRemove {
				found = true
			} else {
				newProfiles = append(newProfiles, p)
			}
		}
		if !found {
			return nil, &ProfileNotInListError{ProfileName: toRemove}
		}
		profiles = newProfiles
	}

	// Apply adds (only if not already present)
	for _, toAdd := range c.addProfiles {
		alreadyPresent := false
		for _, p := range profiles {
			if p == toAdd {
				alreadyPresent = true
				break
			}
		}
		if !alreadyPresent {
			profiles = append(profiles, toAdd)
		} else if !c.quiet {
			// Warn if profile is already installed
			_, _ = fmt.Fprintf(c.stdout, "Warning: profile '%s' is already installed, skipping\n", toAdd)
		}
	}

	return profiles, nil
}

// updateProjectConfigProfiles updates the .weftlo.yaml file with the new profile list,
// preserving all other fields.
func (c *UpdateCommand) updateProjectConfigProfiles(projectDir string, currentConfig *domainconfig.ProjectConfig, newProfiles []string) error {
	configPath := filepath.Join(projectDir, ".weftlo.yaml")

	// Create a new config with updated profiles but preserve other fields
	updatedConfig := &projectConfigYAML{
		Profiles:        newProfiles,
		InstallPrefix:   currentConfig.InstallPrefix,
		Variables:       currentConfig.Variables,
		TargetOverrides: currentConfig.TargetOverrides,
		Ignore:          currentConfig.Ignore,
	}

	// Don't write install_prefix if it's the default value
	if updatedConfig.InstallPrefix == domainconfig.DefaultInstallPrefix {
		updatedConfig.InstallPrefix = ""
	}

	data, err := yaml.Marshal(updatedConfig)
	if err != nil {
		return fmt.Errorf("failed to marshal project config: %w", err)
	}

	if err := afero.WriteFile(c.fs, configPath, data, 0644); err != nil {
		return &PermissionError{Path: configPath, Err: err}
	}

	return nil
}

// projectConfigYAML is the structure for writing .weftlo.yaml
type projectConfigYAML struct {
	Profiles        []string               `yaml:"profiles"`
	InstallPrefix   string                 `yaml:"install_prefix,omitempty"`
	Variables       map[string]interface{} `yaml:"variables,omitempty"`
	TargetOverrides map[string]*string     `yaml:"target_overrides,omitempty"`
	Ignore          []string               `yaml:"ignore,omitempty"`
}
