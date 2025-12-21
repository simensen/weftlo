package template

import (
	"bytes"
	"path/filepath"
	"sort"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig/v3"
	"github.com/spf13/afero"
)

// createIncludeGlobFunc creates a closure that implements the includeGlob template function.
// The closure captures the render context and template context, allowing it to:
//   - Match source paths against glob patterns
//   - Sort matched files alphabetically for deterministic output
//   - Track and enforce recursion depth limits per matched file
//   - Render each matched file and concatenate results with newline separators
//   - Provide descriptive error messages with pattern and file context
//
// Parameters:
//   - renderCtx: The render-time context with filesystem, resolver, and depth tracking
//   - tmplCtx: The template context data passed to all templates
//
// Returns a function that can be registered in a template.FuncMap under the "includeGlob" key.
func createIncludeGlobFunc(renderCtx *RenderContext, tmplCtx *TemplateContext) func(string) (string, error) {
	return func(pattern string) (string, error) {
		// Validate the pattern first by testing against an empty string
		// This catches invalid patterns like "[invalid" even when there are no source paths
		_, err := filepath.Match(pattern, "")
		if err != nil {
			return "", &IncludeGlobPatternError{
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
				return "", &IncludeGlobPatternError{
					Pattern: pattern,
					Err:     err,
				}
			}
			if matched {
				matchedPaths = append(matchedPaths, sourcePath)
			}
		}

		// Return empty string if no files match (not an error)
		if len(matchedPaths) == 0 {
			return "", nil
		}

		// Sort matched paths alphabetically (case-sensitive: A-Z before a-z)
		sort.Strings(matchedPaths)

		// Process each matched file sequentially
		var results []string
		for _, sourcePath := range matchedPaths {
			// Check depth limit before proceeding
			if renderCtx.CurrentDepth >= renderCtx.MaxDepth {
				return "", &IncludeDepthError{
					MaxDepth:     renderCtx.MaxDepth,
					CurrentDepth: renderCtx.CurrentDepth + 1,
					IncludeChain: append(renderCtx.IncludeChain, sourcePath),
				}
			}

			// Resolve the source path to an absolute path
			absolutePath, found := renderCtx.FileResolver.Resolve(sourcePath)
			if !found {
				return "", &IncludeNotFoundError{
					SourcePath:   sourcePath,
					IncludeChain: renderCtx.IncludeChain,
				}
			}

			// Read the file content
			content, err := afero.ReadFile(renderCtx.Fs, absolutePath)
			if err != nil {
				return "", &IncludeNotFoundError{
					SourcePath:   sourcePath,
					IncludeChain: renderCtx.IncludeChain,
				}
			}

			// Create a new render context for the nested include
			nestedCtx := renderCtx.clone(sourcePath)

			// Create function map with sprig functions and nested include/includeGlob/reference/referenceGlob functions
			funcMap := sprig.TxtFuncMap()
			funcMap["include"] = createIncludeFunc(nestedCtx, tmplCtx)
			funcMap["includeGlob"] = createIncludeGlobFunc(nestedCtx, tmplCtx)
			funcMap["reference"] = createReferenceFunc(nestedCtx, tmplCtx)
			funcMap["referenceGlob"] = createReferenceGlobFunc(nestedCtx, tmplCtx)

			// Parse the included template
			tmpl, err := template.New(sourcePath).Funcs(funcMap).Parse(string(content))
			if err != nil {
				return "", &IncludeSyntaxError{
					SourcePath: sourcePath,
					Err:        err,
				}
			}

			// Execute the template with the same context
			var buf bytes.Buffer
			if err := tmpl.Execute(&buf, tmplCtx); err != nil {
				return "", &IncludeExecutionError{
					SourcePath: sourcePath,
					Err:        err,
				}
			}

			results = append(results, buf.String())
		}

		// Concatenate results with single newline separator (no trailing newline)
		return strings.Join(results, "\n"), nil
	}
}
