// Package rendering provides application-layer services for template rendering.
// It orchestrates template enumeration, context building, and rendering for
// merged profiles, returning rendered content with computed target paths.
package rendering

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/spf13/afero"

	"github.com/simensen/weftlo/internal/app/routing"
	"github.com/simensen/weftlo/internal/domain/config"
	"github.com/simensen/weftlo/internal/domain/profile"
	"github.com/simensen/weftlo/internal/domain/template"
	infraprofile "github.com/simensen/weftlo/internal/infrastructure/profile"
)

// RenderResult represents the outcome of rendering a single template.
// It contains the original source path, computed target path, source content,
// and rendered content.
type RenderResult struct {
	// SourcePath is the original template path in the profile (e.g., "config.md.tmpl").
	// This preserves the original path for traceability.
	SourcePath string

	// TargetPath is the computed output path with .tmpl suffix stripped (e.g., "config.md").
	// Other path transformations (install prefix, @ cross-reference) are handled by
	// Output Path Mapping (roadmap item #13).
	TargetPath string

	// SourceContent is the original template content before rendering.
	// This is used for computing source checksums in the manifest.
	SourceContent []byte

	// Content is the rendered template content as a string.
	Content string
}

// RenderCallbacks provides optional callback functions for progress reporting
// during template rendering. Both callbacks are optional and nil-safe.
type RenderCallbacks struct {
	// BeforeRender is called before each template is rendered.
	// Parameters:
	//   - index: 0-based position in the non-partial template list
	//   - total: total number of non-partial templates to render
	//   - sourcePath: the source path of the template about to be rendered
	BeforeRender func(index int, total int, sourcePath string)

	// AfterRender is called after each template render attempt.
	// Parameters:
	//   - index: 0-based position in the non-partial template list
	//   - total: total number of non-partial templates to render
	//   - sourcePath: the source path of the template that was rendered
	//   - err: nil on success, or the error that occurred during rendering
	AfterRender func(index int, total int, sourcePath string, err error)
}

// RenderOptions provides optional configuration for the rendering pipeline.
// All fields are optional and can be nil.
type RenderOptions struct {
	// GlobalConfig is the global configuration from config.yaml.
	// Variables from this config have the lowest precedence.
	GlobalConfig *config.GlobalConfig

	// ProfileConfig is the profile configuration from profile.yaml.
	// Variables from this config override global variables.
	// Deprecated: Use mergedProfile.Variables instead, which contains
	// the deep-merged variables from the entire profile inheritance chain.
	ProfileConfig *profile.ProfileConfig

	// ProjectConfig is the project configuration from .weftlo.yaml.
	// Variables from this config have the highest precedence.
	ProjectConfig *config.ProjectConfig

	// Callbacks provides optional progress reporting callbacks.
	Callbacks *RenderCallbacks

	// Router is the content router for resolving source paths to target paths.
	// This is used by reference() and referenceGlob() template functions to
	// determine where files will be installed. When nil, these functions will
	// return an error if called.
	Router routing.Router

	// Verbose controls whether warnings are emitted during template rendering.
	// When true, warnings about missing files, suppressed targets, and self-references
	// are output. When false, warnings are suppressed.
	Verbose bool

	// WarningWriter is the writer for emitting conflict warnings and other verbose output.
	// When Verbose is true and WarningWriter is set, warnings about variable conflicts
	// and other issues are written to this writer.
	// If nil, warnings are not emitted even when Verbose is true.
	WarningWriter io.Writer
}

// RenderingPipeline orchestrates template rendering for merged profiles.
// It iterates over non-partial templates, builds appropriate contexts,
// renders each template, and returns the results.
type RenderingPipeline struct {
	// engine is the template engine used for rendering operations.
	engine *template.TemplateEngine
}

// NewRenderingPipeline creates a new RenderingPipeline with the provided template engine.
// This follows the constructor injection pattern for dependency injection.
//
// Parameters:
//   - engine: The TemplateEngine to use for rendering templates. Should be created
//     with NewTemplateEngineWithFs() to enable include/includeGlob support.
func NewRenderingPipeline(engine *template.TemplateEngine) *RenderingPipeline {
	return &RenderingPipeline{
		engine: engine,
	}
}

// RenderAll renders all non-partial templates from the merged profile.
// It iterates over templates, builds contexts, renders each one, and returns
// the results with computed target paths.
//
// Parameters:
//   - mergedProfile: The merged profile containing templates to render
//   - projectDir: The absolute path to the project directory
//   - callbacks: Optional callbacks for progress reporting (can be nil)
//
// Returns:
//   - A slice of RenderResult containing rendered templates
//   - An error if any template fails to render (fail-fast behavior)
//
// On error, any successfully rendered templates are included in the returned slice.
// The AfterRender callback (if provided) receives the error before the method returns.
//
// Partial templates (IsPartial = true) are skipped as they are only used via include.
//
// Deprecated: Use RenderAllWithOptions for access to variables and configs.
func (p *RenderingPipeline) RenderAll(
	mergedProfile *infraprofile.MergedProfile,
	projectDir string,
	callbacks *RenderCallbacks,
) ([]RenderResult, error) {
	return p.RenderAllWithOptions(mergedProfile, projectDir, &RenderOptions{
		Callbacks: callbacks,
	})
}

// RenderAllWithOptions renders all non-partial templates from the merged profile
// with full configuration and variable support.
//
// Parameters:
//   - mergedProfile: The merged profile containing templates to render
//   - projectDir: The absolute path to the project directory
//   - opts: Optional render options including configs, router, and callbacks (can be nil)
//
// Returns:
//   - A slice of RenderResult containing rendered templates
//   - An error if any template fails to render (fail-fast behavior)
//
// Variables are merged with precedence: global < profile inheritance < project.
// The merge uses deep merge semantics, where nested maps are recursively merged
// rather than replaced entirely. Templates can access variables via {{ .Variables.key }}.
//
// When Router is provided in opts, the reference() and referenceGlob() template
// functions become available for cross-file reference resolution.
//
// When Verbose is true and WarningWriter is set, conflict warnings are emitted
// for any variable overrides that occurred during merging.
func (p *RenderingPipeline) RenderAllWithOptions(
	mergedProfile *infraprofile.MergedProfile,
	projectDir string,
	opts *RenderOptions,
) ([]RenderResult, error) {
	// Handle nil options
	if opts == nil {
		opts = &RenderOptions{}
	}

	// Get all template files from the merged profile
	templateFiles := mergedProfile.TemplateFiles()

	// Get the filesystem from merged profile
	fs := mergedProfile.Fs()

	// Filter to non-partial templates
	var nonPartials []struct {
		sourcePath string
		absPath    string
	}
	for _, tf := range templateFiles {
		if !tf.IsPartial {
			absPath, found := mergedProfile.Resolve(tf.SourcePath)
			if found {
				nonPartials = append(nonPartials, struct {
					sourcePath string
					absPath    string
				}{tf.SourcePath, absPath})
			}
		}
	}

	total := len(nonPartials)
	results := make([]RenderResult, 0, total)

	// Build the profile for template context
	prof := mergedProfile.ToProfile()

	// Collect variables from each config level for deep merge
	var globalVars, profileVars, projectVars map[string]interface{}

	if opts.GlobalConfig != nil {
		globalVars = opts.GlobalConfig.Variables
	}

	// Use mergedProfile.Variables which contains the deep-merged variables
	// from the entire profile inheritance chain (parent -> child)
	profileVars = mergedProfile.Variables

	if opts.ProjectConfig != nil {
		projectVars = opts.ProjectConfig.Variables
	}

	// Merge variables using DeepMergeVariablesChain with precedence:
	// global config (lowest) -> profile inheritance -> project config (highest)
	// This replaces the previous shallow MergeVariables() call
	mergedVars, mergeConflicts := DeepMergeVariablesChain(
		VariableSource{Name: "global config", Vars: globalVars},
		VariableSource{Name: "profile inheritance", Vars: profileVars},
		VariableSource{Name: "project config", Vars: projectVars},
	)

	// Combine profile inheritance conflicts with global/project level conflicts
	// Profile inheritance conflicts come first for chronological order
	profileConflicts := mergedProfile.VariableConflicts()
	allConflicts := make([]VariableConflict, 0, len(profileConflicts)+len(mergeConflicts))
	allConflicts = append(allConflicts, profileConflicts...)
	allConflicts = append(allConflicts, mergeConflicts...)

	// Emit conflict warnings if verbose mode is enabled and we have a writer
	if opts.Verbose && opts.WarningWriter != nil && len(allConflicts) > 0 {
		emitConflictWarnings(opts.WarningWriter, allConflicts)
	}

	// Build template context
	tmplCtx := template.NewTemplateContext(
		prof,
		projectDir,
		time.Now(),
		opts.ProjectConfig,
		opts.GlobalConfig,
		mergedVars,
	)

	// Build render context with filesystem, file resolver, and optionally router
	var renderCtx *template.RenderContext
	if opts.Router != nil {
		// Use the new constructor that accepts router and verbose flag
		renderCtx = template.NewRenderContextWithRouter(fs, mergedProfile, opts.Router, opts.Verbose)
	} else {
		// Fall back to the original constructor for backward compatibility
		renderCtx = template.NewRenderContext(fs, mergedProfile)
	}

	// Get callbacks from options
	callbacks := opts.Callbacks

	// Iterate and render each non-partial template
	for i, tmpl := range nonPartials {
		// Invoke BeforeRender callback if provided
		if callbacks != nil && callbacks.BeforeRender != nil {
			callbacks.BeforeRender(i, total, tmpl.sourcePath)
		}

		// Read template content from filesystem
		sourceContent, err := afero.ReadFile(fs, tmpl.absPath)
		if err != nil {
			// Invoke AfterRender callback with error
			if callbacks != nil && callbacks.AfterRender != nil {
				callbacks.AfterRender(i, total, tmpl.sourcePath, err)
			}
			return results, err
		}

		// Render the template
		rendered, err := p.engine.RenderWithContext(string(sourceContent), tmplCtx, renderCtx)
		if err != nil {
			// Invoke AfterRender callback with error
			if callbacks != nil && callbacks.AfterRender != nil {
				callbacks.AfterRender(i, total, tmpl.sourcePath, err)
			}
			return results, err
		}

		// Compute target path by stripping .tmpl suffix
		targetPath := computeTargetPath(tmpl.sourcePath)

		// Create result and append
		result := RenderResult{
			SourcePath:    tmpl.sourcePath,
			TargetPath:    targetPath,
			SourceContent: sourceContent,
			Content:       rendered,
		}
		results = append(results, result)

		// Invoke AfterRender callback with success
		if callbacks != nil && callbacks.AfterRender != nil {
			callbacks.AfterRender(i, total, tmpl.sourcePath, nil)
		}
	}

	return results, nil
}

// emitConflictWarnings writes formatted conflict warnings to the provided writer.
// Format: "Warning: Variable '[path]' from [source] overridden by [overriding source]"
func emitConflictWarnings(w io.Writer, conflicts []VariableConflict) {
	for _, conflict := range conflicts {
		_, _ = fmt.Fprintf(w, "Warning: Variable '%s' from %s overridden by %s\n",
			conflict.Path,
			conflict.SourceProfile,
			conflict.OverridingProfile,
		)
	}
}

// computeTargetPath strips the .tmpl suffix from a source path to compute
// the target path. If the source path does not have a .tmpl suffix, it is
// returned unchanged.
func computeTargetPath(sourcePath string) string {
	return strings.TrimSuffix(sourcePath, ".tmpl")
}
