package template

import (
	"fmt"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/simensen/weftlo/internal/app/routing"
)

// createReferenceGlobFunc creates a closure that implements the referenceGlob template function.
// The closure captures the render context and template context, allowing it to:
//   - Match source paths against glob patterns
//   - Filter out partial source files (partials cannot be referenced)
//   - Route each matched path through the router to get target paths
//   - Filter out suppressed targets (null overrides)
//   - Return sorted @ prefixed target paths for deterministic output
//
// Parameters:
//   - renderCtx: The render-time context with filesystem, resolver, router, and verbose flag
//   - tmplCtx: The template context data passed to all templates (unused but included for consistency)
//
// Returns a function that can be registered in a template.FuncMap under the "referenceGlob" key.
func createReferenceGlobFunc(renderCtx *RenderContext, tmplCtx *TemplateContext) func(string) ([]string, error) {
	return func(pattern string) ([]string, error) {
		// Validate that router is available
		if renderCtx.Router == nil {
			return nil, fmt.Errorf("referenceGlob: router not available in render context")
		}

		// Validate the pattern first by testing against an empty string
		// This catches invalid patterns like "[invalid" even when there are no source paths
		_, err := filepath.Match(pattern, "")
		if err != nil {
			return nil, &ReferenceGlobPatternError{
				Pattern: pattern,
				Err:     err,
			}
		}

		// Get all available source paths from the file resolver
		sourcePaths := renderCtx.FileResolver.ListSourcePaths()

		// Match pattern against each source path
		var matchedPaths []string
		for _, sourcePath := range sourcePaths {
			matched, err := filepath.Match(pattern, sourcePath)
			if err != nil {
				// Invalid glob pattern (e.g., malformed character class)
				// This should not happen since we validated above, but handle defensively
				return nil, &ReferenceGlobPatternError{
					Pattern: pattern,
					Err:     err,
				}
			}
			if matched {
				matchedPaths = append(matchedPaths, sourcePath)
			}
		}

		// Return empty slice if no files match (not an error)
		if len(matchedPaths) == 0 {
			return []string{}, nil
		}

		// Sort matched paths alphabetically for deterministic processing
		sort.Strings(matchedPaths)

		// Process each matched file
		var results []string
		for _, sourcePath := range matchedPaths {
			// Strip .tmpl suffix from source path (consistent with pipeline behavior)
			cleanSourcePath := strings.TrimSuffix(sourcePath, ".tmpl")

			// Check if the source path is a partial using type assertion
			// Partials are filtered from results (not an error like reference())
			if partialResolver, ok := renderCtx.Router.(routing.PartialResolver); ok {
				if partialResolver.IsPartialSource(cleanSourcePath) {
					// Skip partials - they cannot be referenced
					continue
				}
			}

			// Route the path to get the target
			routedFile, err := renderCtx.Router.Route(cleanSourcePath)
			if err != nil {
				// Skip paths that fail to route (lenient behavior)
				if renderCtx.Verbose {
					fmt.Printf("referenceGlob: warning: failed to route path %q: %v\n", sourcePath, err)
				}
				continue
			}

			// Filter out suppressed targets (null override)
			if routedFile.Suppressed {
				if renderCtx.Verbose {
					fmt.Printf("referenceGlob: warning: target for %q is suppressed (null override)\n", sourcePath)
				}
				continue
			}

			// Strip .tmpl suffix from target path (consistent with pipeline behavior)
			targetPath := strings.TrimSuffix(routedFile.TargetPath, ".tmpl")

			// Normalize with forward slashes using path.Clean
			targetPath = path.Clean(targetPath)

			// Prepend @ prefix
			result := "@" + targetPath

			results = append(results, result)
		}

		// Sort results alphabetically for deterministic output
		sort.Strings(results)

		// Return empty slice if no valid results (not an error)
		if results == nil {
			return []string{}, nil
		}

		return results, nil
	}
}
