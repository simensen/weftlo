package install

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/spf13/afero"

	"github.com/simensen/weftlo/internal/app/dryrun"
	"github.com/simensen/weftlo/internal/app/rendering"
	"github.com/simensen/weftlo/internal/app/routing"
	domainconfig "github.com/simensen/weftlo/internal/domain/config"
	domainignore "github.com/simensen/weftlo/internal/domain/ignore"
	"github.com/simensen/weftlo/internal/domain/manifest"
	infraprofile "github.com/simensen/weftlo/internal/infrastructure/profile"
)

// InstallOptions configures the behavior of the install operation.
type InstallOptions struct {
	// DryRun when true prevents any filesystem writes and instead outputs
	// what would happen to stdout. Defaults to false.
	DryRun bool

	// Stdout is the writer for dry-run output. If nil, os.Stdout is used.
	Stdout io.Writer

	// Verbose controls whether warnings are emitted during template rendering.
	// When true, warnings about missing files, suppressed targets, and self-references
	// are output. When false, warnings are suppressed. Defaults to false.
	Verbose bool
}

// DefaultInstallOptions returns InstallOptions with default values.
func DefaultInstallOptions() InstallOptions {
	return InstallOptions{
		DryRun:  false,
		Stdout:  os.Stdout,
		Verbose: false,
	}
}

// Service orchestrates the installation of profiles into projects.
// It coordinates rendering, routing, manifest generation, and file writing.
type Service struct {
	// fs is the filesystem for all file operations.
	fs afero.Fs

	// pipeline is the rendering pipeline for template processing.
	pipeline *rendering.RenderingPipeline

	// manifestGenerator creates manifest from rendered files.
	manifestGenerator *manifest.ManifestGenerator

	// operationBuilder converts render results to file operations.
	operationBuilder *OperationBuilder
}

// NewService creates a new install Service with the provided dependencies.
//
// Parameters:
//   - fs: filesystem for read/write operations
//   - pipeline: rendering pipeline for template processing
//   - manifestGenerator: generator for creating manifests
func NewService(
	fs afero.Fs,
	pipeline *rendering.RenderingPipeline,
	manifestGenerator *manifest.ManifestGenerator,
) *Service {
	return &Service{
		fs:                fs,
		pipeline:          pipeline,
		manifestGenerator: manifestGenerator,
		operationBuilder:  NewOperationBuilder(),
	}
}

// InstallResult contains the outcome of an install operation.
type InstallResult struct {
	// FilesProcessed is the number of files that were or would be written.
	FilesProcessed int

	// DryRun indicates if this was a dry-run execution.
	DryRun bool
}

// Install executes the installation workflow with the provided options.
// The workflow:
//  1. Renders all templates from the merged profile
//  2. Routes paths using the Router to determine target locations
//  3. If dry-run: formats and outputs planned operations, then returns
//  4. If not dry-run: writes files, directories, and manifest to disk
//
// Files routed to a suppressed target (RoutedFile.Suppressed == true) are
// silently skipped and not included in the manifest.
//
// The router is passed to the rendering pipeline so that templates can use
// the reference() and referenceGlob() functions for cross-file references.
//
// Parameters:
//   - mergedProfile: the merged profile containing templates to render
//   - projectDir: absolute path to the project directory
//   - router: the Router for computing target installation paths
//   - opts: installation options (dry-run, output writer, verbose, etc.)
//
// Returns:
//   - InstallResult with operation counts
//   - error if any step fails
func (s *Service) Install(
	mergedProfile *infraprofile.MergedProfile,
	projectDir string,
	router routing.Router,
	opts InstallOptions,
) (*InstallResult, error) {
	// Step 1: Render all templates with router passed for reference() support
	renderOpts := &rendering.RenderOptions{
		Router:  router,
		Verbose: opts.Verbose,
	}
	renderResults, err := s.pipeline.RenderAllWithOptions(mergedProfile, projectDir, renderOpts)
	if err != nil {
		return nil, fmt.Errorf("rendering failed: %w", err)
	}

	// Step 2: Check for dry-run mode
	if opts.DryRun {
		return s.executeDryRun(renderResults, router, mergedProfile.LeafProfileName(), projectDir, opts)
	}

	// Step 3: Execute actual installation (write files)
	return s.executeInstall(renderResults, router, mergedProfile.ProfileNames(), mergedProfile.LeafProfileName(), projectDir)
}

// executeDryRun formats and outputs the planned operations without writing any files.
// This method guarantees zero writes to the filesystem.
func (s *Service) executeDryRun(
	renderResults []rendering.RenderResult,
	router routing.Router,
	profileName string,
	projectDir string,
	opts InstallOptions,
) (*InstallResult, error) {
	// Route each render result and filter out suppressed files
	var routedResults []rendering.RenderResult
	for _, result := range renderResults {
		// Route the file using the Router
		routedFile, err := router.Route(result.TargetPath)
		if err != nil {
			return nil, fmt.Errorf("routing failed for %s: %w", result.TargetPath, err)
		}

		// Skip suppressed files silently
		if routedFile.Suppressed {
			continue
		}

		// Update the target path with the routed path
		result.TargetPath = routedFile.TargetPath
		routedResults = append(routedResults, result)
	}

	// Convert render results to file operations with absolute target paths
	// so the formatter can check for existing files correctly
	operations := s.operationBuilder.BuildFromRenderResultsWithProjectDir(
		routedResults,
		profileName,
		projectDir,
	)

	// Determine output writer
	stdout := opts.Stdout
	if stdout == nil {
		stdout = os.Stdout
	}

	// Create formatter and output results
	formatter := dryrun.NewFormatter(stdout, s.fs)
	if err := formatter.Format(operations); err != nil {
		return nil, fmt.Errorf("formatting dry-run output failed: %w", err)
	}

	return &InstallResult{
		FilesProcessed: len(operations),
		DryRun:         true,
	}, nil
}

// executeInstall writes files, directories, and manifest to disk.
func (s *Service) executeInstall(
	renderResults []rendering.RenderResult,
	router routing.Router,
	profileNames []string,
	leafProfileName string,
	projectDir string,
) (*InstallResult, error) {
	// Build rendered files for manifest generation
	renderedFiles := make([]manifest.RenderedFile, 0, len(renderResults))
	filesProcessed := 0

	for _, result := range renderResults {
		// Route the file using the Router
		routedFile, err := router.Route(result.TargetPath)
		if err != nil {
			return nil, fmt.Errorf("routing failed for %s: %w", result.TargetPath, err)
		}

		// Skip suppressed files silently (no error, no manifest entry)
		if routedFile.Suppressed {
			continue
		}

		// Compute absolute target path using the routed path
		targetPath := filepath.Join(projectDir, routedFile.TargetPath)

		// Ensure parent directory exists
		dir := filepath.Dir(targetPath)
		if err := s.fs.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create directory %s: %w", dir, err)
		}

		// Write file content (rendered output)
		outputContent := []byte(result.Content)
		if err := afero.WriteFile(s.fs, targetPath, outputContent, 0644); err != nil {
			return nil, fmt.Errorf("failed to write file %s: %w", targetPath, err)
		}

		// Build RenderedFile for manifest with proper source/output separation.
		// SourceContent is the raw template content before rendering.
		// OutputContent is the rendered content after template processing.
		// TargetCategory is populated from the RoutedFile.
		renderedFile := manifest.RenderedFile{
			TargetPath:     routedFile.TargetPath,
			SourceContent:  result.SourceContent,
			OutputContent:  outputContent,
			SourceProfile:  leafProfileName,
			TargetCategory: routedFile.TargetCategory,
		}
		renderedFiles = append(renderedFiles, renderedFile)
		filesProcessed++
	}

	// Generate manifest
	m, err := s.manifestGenerator.Generate(profileNames, renderedFiles)
	if err != nil {
		return nil, fmt.Errorf("failed to generate manifest: %w", err)
	}

	// Write manifest to project directory
	manifestPath := filepath.Join(projectDir, ".weftlo.manifest.json")

	manifestContent, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to serialize manifest: %w", err)
	}

	if err := afero.WriteFile(s.fs, manifestPath, manifestContent, 0644); err != nil {
		return nil, fmt.Errorf("failed to write manifest: %w", err)
	}

	return &InstallResult{
		FilesProcessed: filesProcessed,
		DryRun:         false,
	}, nil
}

// =============================================================================
// Global Override Integration Helpers
// =============================================================================

// BuildMergedOverrides merges target overrides from all four levels of the
// override precedence chain and evaluates any templates in the merged result.
//
// The four-level precedence (lowest to highest):
//  1. Profile targets from profile.yaml content.targets
//  2. Global universal overrides from ~/.config/weftlo/config.yaml target_overrides
//  3. Global profile-specific overrides from ~/.config/weftlo/config.yaml profile_overrides.<name>.target_overrides
//  4. Project overrides from .weftlo.yaml target_overrides
//
// After merging, any template expressions in the override values are evaluated
// using the provided merged variables.
//
// Parameters:
//   - profileTargets: Profile's content.targets converted to override format (level 1)
//   - globalConfig: Global configuration containing universal and profile-specific overrides
//   - profileName: The current profile name for selecting profile-specific overrides
//   - projectOverrides: Project-level target overrides from .weftlo.yaml (level 4)
//   - mergedVariables: Variables merged from global + profile + project for template evaluation
//
// Returns:
//   - Merged and evaluated overrides ready for router construction
//   - Error if template evaluation fails
func BuildMergedOverrides(
	profileTargets map[string]string,
	globalConfig *domainconfig.GlobalConfig,
	profileName string,
	projectOverrides map[string]*string,
	mergedVariables map[string]interface{},
) (map[string]*string, error) {
	// Level 1: Convert profile targets to override format
	level1 := routing.ConvertTargetsToOverrides(profileTargets)

	// Level 2: Global universal overrides
	var level2 map[string]*string
	if globalConfig != nil {
		level2 = globalConfig.TargetOverrides
	}

	// Level 3: Global profile-specific overrides
	var level3 map[string]*string
	if globalConfig != nil && profileName != "" {
		if profileOverride, exists := globalConfig.ProfileOverrides[profileName]; exists {
			level3 = profileOverride.TargetOverrides
		}
	}

	// Level 4: Project overrides
	level4 := projectOverrides

	// Merge all four levels
	merged := routing.MergeOverrides(level1, level2, level3, level4)

	// Evaluate templates in merged overrides
	evaluated, err := EvaluateOverrideTemplates(merged, mergedVariables)
	if err != nil {
		return nil, err
	}

	return evaluated, nil
}

// EvaluateOverrideTemplates evaluates Go templates in override values.
// This uses the TargetEvaluator to process template expressions like
// {{ .Variables.namespace }} in override path values.
//
// Parameters:
//   - overrides: Map of override names to path values (may contain templates)
//   - variables: Merged variables for template evaluation
//
// Returns:
//   - New map with evaluated override values
//   - Error if any template evaluation fails
//
// Nil pointer values (suppression) are preserved unchanged.
// Empty string values are passed through unchanged.
func EvaluateOverrideTemplates(
	overrides map[string]*string,
	variables map[string]interface{},
) (map[string]*string, error) {
	if overrides == nil {
		return make(map[string]*string), nil
	}

	evaluator := routing.NewTargetEvaluator(variables)
	result := make(map[string]*string, len(overrides))

	for name, value := range overrides {
		if value == nil {
			// Preserve nil (suppression semantics)
			result[name] = nil
			continue
		}

		if *value == "" {
			// Preserve empty string (redirect to root semantics)
			empty := ""
			result[name] = &empty
			continue
		}

		// Evaluate template
		evaluated, err := evaluator.Evaluate(name, *value)
		if err != nil {
			return nil, fmt.Errorf("failed to evaluate override template for '%s': %w", name, err)
		}

		result[name] = &evaluated
	}

	return result, nil
}

// MergeIgnorePatterns merges ignore patterns from profile, global, and project sources.
// The merge follows the order specified in the Global Target Overrides spec:
//  1. Profile patterns (first)
//  2. Global patterns (second)
//  3. Project patterns (last, highest precedence for negation)
//
// Parameters:
//   - profileMatcher: Matcher from the merged profile (may be nil)
//   - globalPatterns: Ignore patterns from global config (may be nil/empty)
//   - projectMatcher: Matcher from project .weftlo.ignore or config (may be nil)
//
// Returns:
//   - A merged Matcher combining all pattern sources
func MergeIgnorePatterns(
	profileMatcher domainignore.Matcher,
	globalPatterns []string,
	projectMatcher domainignore.Matcher,
) domainignore.Matcher {
	// Start with profile patterns (or empty if nil)
	result := profileMatcher
	if result == nil {
		result = domainignore.EmptyMatcher()
	}

	// Merge global patterns (level 2)
	if len(globalPatterns) > 0 {
		globalMatcher := domainignore.NewPatternMatcher(globalPatterns)
		result = result.MergeWith(globalMatcher)
	}

	// Merge project patterns (level 3, highest precedence)
	if projectMatcher != nil {
		result = result.MergeWith(projectMatcher)
	}

	return result
}

// ExtractProfileName extracts the leaf profile name from a MergedProfile.
// This is used to select profile-specific overrides from GlobalConfig.ProfileOverrides.
//
// Parameters:
//   - mergedProfile: The merged profile from which to extract the name
//
// Returns:
//   - The leaf (most specific) profile name, or empty string if profile is nil
func ExtractProfileName(mergedProfile *infraprofile.MergedProfile) string {
	if mergedProfile == nil {
		return ""
	}
	return mergedProfile.LeafProfileName()
}
