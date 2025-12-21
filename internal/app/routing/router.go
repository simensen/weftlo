// Package routing provides services for transforming content source paths to target installation paths.
//
// The routing package implements a multi-target path routing engine that transforms
// source paths (relative to content root) into target installation paths based on
// configurable target mappings from profile.yaml. It supports project-level overrides
// and a reserved _partials virtual namespace for template includes.
//
// Target paths support Go template syntax with variable substitution. Templates are
// evaluated once during router construction using the provided variables map, and the
// evaluated paths are stored on the router. This allows dynamic path components like
// configurable namespaces (e.g., ".{{ .Variables.namespace }}/claude") while ensuring
// consistent routing behavior after construction.
//
// Key components:
//   - Router: Interface for transforming source paths to installation paths
//   - PartialResolver: Interface for identifying and canonicalizing partial template sources
//   - ContentRouter: Implementation of both Router and PartialResolver interfaces
//   - RoutedFile: Result structure containing routing metadata
//   - TargetEvaluator: Evaluates target path templates with variable substitution
package routing

import (
	"path"

	"github.com/simensen/weftlo/internal/domain/profile"
)

// Router defines the interface for transforming content source paths to target
// installation paths.
//
// Router implementations are responsible for:
//   - Matching source path prefixes against configured target mappings
//   - Replacing matched prefixes with target installation paths
//   - Applying project-level overrides that redirect or suppress targets
//   - Tracking metadata about whether overrides were applied or targets suppressed
//
// Router implementations should be stateless and constructed fresh per routing
// operation with the appropriate configuration and overrides.
type Router interface {
	// Route transforms a source path (relative to content root) into a RoutedFile
	// containing the target installation path and routing metadata.
	//
	// The source path should be relative to the content root directory (e.g.,
	// "claude/settings.json" not "content/claude/settings.json").
	//
	// Route returns an error if:
	//   - The source path's category is not found and no default_target is configured
	//   - A project override attempts to suppress the _partials target
	//   - The target configuration contains invalid values
	//
	// For suppressed targets (null override), Route returns a RoutedFile with
	// Suppressed=true and TargetPath="", not an error.
	Route(sourcePath string) (RoutedFile, error)
}

// PartialResolver defines the interface for identifying and canonicalizing
// partial template source paths.
//
// Partials are template fragments that can be included by other templates.
// They are stored in a designated _partials namespace and are never installed
// to the target project. Instead, they are resolved during template rendering.
//
// The PartialResolver interface allows:
//   - Identifying which source paths are partial templates
//   - Converting source paths to canonical _partials/* reference format
//
// This interface is used by the template rendering system to resolve include
// references across profiles.
type PartialResolver interface {
	// IsPartialSource returns true if the given source path represents a partial
	// template source file.
	//
	// A path is considered a partial source if its first segment matches the
	// _partials target key in the configuration (either the literal "_partials"
	// or a custom directory mapped to _partials).
	IsPartialSource(sourcePath string) bool

	// Canonicalize converts a partial source path to its canonical _partials/*
	// reference format.
	//
	// For example, if "templates/partials" is mapped to _partials:
	//   - Input:  "templates/partials/header.tmpl"
	//   - Output: "_partials/header.tmpl"
	//
	// For nested partials, subdirectory structure is preserved:
	//   - Input:  "templates/partials/sub/footer.tmpl"
	//   - Output: "_partials/sub/footer.tmpl"
	//
	// Returns an empty string if the source path is not a partial source.
	Canonicalize(sourcePath string) string
}

// RoutedFile represents the result of routing a source path through the
// content routing engine.
//
// It contains both the computed target path and metadata about how the
// routing was performed, including whether overrides were applied and
// whether the target was suppressed.
type RoutedFile struct {
	// SourcePath is the original path relative to content root that was routed.
	// This is the input path passed to Router.Route().
	SourcePath string

	// TargetPath is the computed installation path relative to the project root.
	// This is where the file should be installed in the target project.
	//
	// TargetPath will be empty if:
	//   - The target was suppressed via a null override (Suppressed=true)
	//   - The source is a _partials path (virtual namespace, never installed)
	TargetPath string

	// TargetCategory is the source directory name that matched during routing.
	// This is the first segment of the source path that was used to look up
	// the target mapping.
	//
	// For example, for source path "claude/settings.json", TargetCategory would
	// be "claude" (assuming "claude" is a configured target).
	TargetCategory string

	// OverrideApplied is true if a project override modified the target path.
	// This indicates that the final TargetPath differs from what the profile
	// configuration alone would have produced.
	OverrideApplied bool

	// Suppressed is true if the target was set to null in the project override.
	// When Suppressed is true, TargetPath will be empty and the file should
	// not be installed to the target project.
	Suppressed bool
}

// ContentRouter implements both Router and PartialResolver interfaces.
// It provides multi-target path routing with support for project-level overrides
// and a reserved _partials virtual namespace for template includes.
//
// ContentRouter is stateless and should be constructed fresh per routing operation
// with the appropriate configuration and overrides.
//
// The targets field contains evaluated (resolved) target paths, not raw templates.
// Template evaluation occurs once during construction via TargetEvaluator, ensuring
// consistent routing behavior after the router is created. This design means that
// variable values are fixed at construction time.
//
// The _partials virtual namespace is special:
//   - Paths in _partials are never installed to the target project
//   - They are used for template includes during rendering
//   - Attempting to suppress _partials via override returns PartialSuppressionError
type ContentRouter struct {
	// targets is the map of source directory names to evaluated target installation paths.
	// These are the final resolved paths after template evaluation, not raw template strings.
	// Templates are evaluated once during construction and stored here for use by Route().
	targets map[string]string

	// defaultTarget is the fallback path for unmapped directories
	defaultTarget string

	// overrides is the map of target category names to override values
	// A nil pointer value indicates suppression (null in YAML)
	// Override values are treated as literal paths (no template evaluation)
	overrides map[string]*string

	// partialSources maps source directory names that map to _partials
	// This includes both "_partials" itself and any custom directories
	partialSources map[string]bool
}

// Ensure ContentRouter implements both interfaces
var _ Router = (*ContentRouter)(nil)
var _ PartialResolver = (*ContentRouter)(nil)

// NewContentRouter creates a new ContentRouter with the given configuration, overrides,
// and variables for target path template evaluation.
//
// Parameters:
//   - config: The merged ContentConfig containing targets and default_target.
//     Target values may contain Go template syntax (e.g., ".{{ .Variables.namespace }}/claude")
//     which will be evaluated using the provided variables.
//   - overrides: Map of target category names to override values. A nil pointer
//     value indicates suppression (the target should be completely disabled).
//     Override values are treated as literal paths and are NOT template-evaluated.
//   - variables: Map of variable names to values for template evaluation.
//     These are accessible in target templates via {{ .Variables.key }}.
//     May be nil if no variables are defined.
//
// Returns an error if:
//   - The overrides map contains a nil value for "_partials" (PartialSuppressionError)
//   - A target template references an undefined variable (MissingVariableError)
//   - A target template has a syntax error (TargetTemplateError)
//   - An evaluated target path is invalid (InvalidTargetPathError)
//   - The targets configuration contains invalid values (InvalidTargetError)
//
// Template evaluation is performed once during construction. The evaluated paths are
// stored on the router and used for all subsequent Route() calls. This ensures
// consistent routing behavior and fail-fast error handling for template issues.
//
// The router is stateless and safe for concurrent use after construction.
func NewContentRouter(config profile.ContentConfig, overrides map[string]*string, variables map[string]interface{}) (*ContentRouter, error) {
	// Validate that _partials is not being suppressed
	if overrides != nil {
		if overrideValue, exists := overrides["_partials"]; exists && overrideValue == nil {
			return nil, &PartialSuppressionError{
				SourcePath: "project configuration",
			}
		}
	}

	// Evaluate target templates using the TargetEvaluator
	evaluator := NewTargetEvaluator(variables)
	evaluatedTargets, err := evaluator.EvaluateAll(config.Targets)
	if err != nil {
		// Fail fast: return error immediately if evaluation fails
		return nil, err
	}

	// Validate evaluated targets for reserved values
	if err := validateTargets(evaluatedTargets); err != nil {
		return nil, err
	}

	// Build the partialSources map - includes all source directories that map to _partials
	// Note: we check the evaluated targets since that's what determines if something is a partial
	partialSources := make(map[string]bool)
	for name, targetPath := range evaluatedTargets {
		if targetPath == "_partials" {
			partialSources[name] = true
		}
	}

	return &ContentRouter{
		targets:        evaluatedTargets,
		defaultTarget:  config.DefaultTarget,
		overrides:      overrides,
		partialSources: partialSources,
	}, nil
}

// Route transforms a source path (relative to content root) into a RoutedFile
// containing the target installation path and routing metadata.
//
// The routing process:
//  1. Extract the first path segment to determine the target category
//  2. Check if the category maps to _partials (virtual namespace)
//  3. Resolve the target path using the targets map or default_target
//  4. Apply any project override for this target category
//  5. Return the RoutedFile with appropriate metadata
//
// For _partials paths:
//   - Returns Suppressed=true and TargetPath="" (virtual, not for installation)
//   - OverrideApplied will be false (this is inherent behavior)
//
// For suppressed targets (null override):
//   - Returns Suppressed=true and TargetPath=""
//   - OverrideApplied will be true
//
// Note: Target paths are pre-evaluated during router construction. The Route()
// method uses the already-evaluated paths stored on the router, not raw templates.
func (r *ContentRouter) Route(sourcePath string) (RoutedFile, error) {
	result := RoutedFile{
		SourcePath: sourcePath,
	}

	// Use target resolution from target.go
	targetResult, err := ResolveTarget(sourcePath, r.targets, r.defaultTarget)
	if err != nil {
		return result, err
	}

	// Set the category based on target resolution
	result.TargetCategory = targetResult.Category

	// Determine the category to use for override lookup
	// If no category matched (used default), we use the first segment
	overrideCategory := targetResult.Category
	if overrideCategory == "" {
		// Extract first segment for override lookup when using default target
		segment, _ := extractFirstSegment(sourcePath)
		overrideCategory = segment
	}

	// Check if this is a _partials path (virtual namespace)
	if targetResult.IsPartial {
		// _partials is a virtual namespace - never installed
		result.TargetPath = ""
		result.Suppressed = true
		result.OverrideApplied = false
		return result, nil
	}

	// Check if the category maps to _partials (custom partial directory)
	if r.partialSources[overrideCategory] {
		result.TargetPath = ""
		result.Suppressed = true
		result.OverrideApplied = false
		return result, nil
	}

	// Apply project override if present
	// Note: Override values are treated as literal paths (no template evaluation)
	if r.overrides != nil {
		if overrideValue, exists := r.overrides[overrideCategory]; exists {
			if overrideValue == nil {
				// Null override - suppress this target
				result.TargetPath = ""
				result.Suppressed = true
				result.OverrideApplied = true
				return result, nil
			}

			// Non-null override - replace target path
			_, remainder := extractFirstSegment(sourcePath)
			if *overrideValue == "" {
				// Override to project root
				result.TargetPath = remainder
			} else if remainder == "" {
				result.TargetPath = *overrideValue
			} else {
				result.TargetPath = path.Join(*overrideValue, remainder)
			}
			result.OverrideApplied = true
			return result, nil
		}
	}

	// No override - use resolved target path (already evaluated during construction)
	result.TargetPath = targetResult.TargetPath
	return result, nil
}

// IsPartialSource returns true if the given source path represents a partial
// template source file.
//
// A path is considered a partial source if its first segment matches:
//   - The literal "_partials" target key
//   - Any custom directory that maps to "_partials" in the targets config
func (r *ContentRouter) IsPartialSource(sourcePath string) bool {
	segment, _ := extractFirstSegment(sourcePath)

	// Check if the segment is "_partials" or maps to "_partials"
	if segment == "_partials" {
		return true
	}

	return r.partialSources[segment]
}

// Canonicalize converts a partial source path to its canonical _partials/*
// reference format.
//
// For custom partial directories (e.g., "templates" -> "_partials"), this
// converts the source path to use the canonical "_partials" prefix while
// preserving the subdirectory structure.
//
// Examples:
//   - "templates/header.tmpl" -> "_partials/header.tmpl" (if templates -> _partials)
//   - "_partials/sub/footer.tmpl" -> "_partials/sub/footer.tmpl"
//   - "claude/settings.json" -> "" (not a partial source)
//
// Returns an empty string if the source path is not a partial source.
func (r *ContentRouter) Canonicalize(sourcePath string) string {
	if !r.IsPartialSource(sourcePath) {
		return ""
	}

	segment, remainder := extractFirstSegment(sourcePath)

	// If already using _partials prefix, return as-is (after path.Join for normalization)
	if segment == "_partials" {
		if remainder == "" {
			return "_partials"
		}
		return path.Join("_partials", remainder)
	}

	// Convert custom partial directory to _partials prefix
	if remainder == "" {
		return "_partials"
	}
	return path.Join("_partials", remainder)
}
