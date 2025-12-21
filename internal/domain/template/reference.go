package template

import (
	"fmt"
	"path"
	"strings"

	"github.com/simensen/weftlo/internal/app/routing"
)

// createReferenceFunc creates a closure that implements the reference template function.
// The closure captures the render context and template context, allowing it to:
//   - Resolve source paths to target installation paths via the Router
//   - Prepend the @ prefix to target paths
//   - Handle suppressed targets by returning empty string
//   - Provide lenient error handling with warnings for missing files
//   - Detect and reject partial source paths
//
// Parameters:
//   - renderCtx: The render-time context with filesystem, resolver, router, and verbose flag
//   - tmplCtx: The template context data passed to all templates (unused but included for consistency)
//
// Returns a function that can be registered in a template.FuncMap under the "reference" key.
func createReferenceFunc(renderCtx *RenderContext, tmplCtx *TemplateContext) func(string) (string, error) {
	return func(sourcePath string) (string, error) {
		// Validate that router is available
		if renderCtx.Router == nil {
			return "", fmt.Errorf("reference: router not available in render context")
		}

		// Strip .tmpl suffix from input source path (consistent with pipeline behavior)
		// This allows users to reference "file.md.tmpl" as "file.md.tmpl" or "file.md"
		// and get the correct target path
		cleanSourcePath := strings.TrimSuffix(sourcePath, ".tmpl")

		// Check if the source path is a partial using type assertion
		// The Router interface may also implement PartialResolver
		if partialResolver, ok := renderCtx.Router.(routing.PartialResolver); ok {
			if partialResolver.IsPartialSource(cleanSourcePath) {
				return "", &ReferencePartialError{
					SourcePath: sourcePath,
				}
			}
		}

		// Check if source path exists in merged profile
		_, exists := renderCtx.FileResolver.Resolve(sourcePath)
		if !exists {
			// Also try with .tmpl suffix stripped in case user provided the template filename
			_, exists = renderCtx.FileResolver.Resolve(cleanSourcePath)
		}

		// Check for self-reference (current file referencing itself)
		// The current file is the last entry in the include chain (if any)
		isSelfReference := false
		if len(renderCtx.IncludeChain) > 0 {
			currentFile := renderCtx.IncludeChain[len(renderCtx.IncludeChain)-1]
			currentFileClean := strings.TrimSuffix(currentFile, ".tmpl")
			if currentFileClean == cleanSourcePath || currentFile == sourcePath {
				isSelfReference = true
			}
		}

		// Route the path to get the target
		routedFile, err := renderCtx.Router.Route(cleanSourcePath)
		if err != nil {
			return "", fmt.Errorf("reference: failed to route path %q: %w", sourcePath, err)
		}

		// Handle suppressed targets (null override)
		if routedFile.Suppressed {
			if renderCtx.Verbose {
				fmt.Printf("reference: warning: target for %q is suppressed (null override)\n", sourcePath)
			}
			return "", nil
		}

		// Strip .tmpl suffix from target path (consistent with pipeline behavior)
		targetPath := strings.TrimSuffix(routedFile.TargetPath, ".tmpl")

		// Normalize with forward slashes using path.Clean
		targetPath = path.Clean(targetPath)

		// Prepend @ prefix
		result := "@" + targetPath

		// Emit warnings for lenient cases when verbose is enabled
		if renderCtx.Verbose {
			if !exists {
				fmt.Printf("reference: warning: source path %q not found in merged profile, returning computed path\n", sourcePath)
			}
			if isSelfReference {
				fmt.Printf("reference: warning: file %q references itself\n", sourcePath)
			}
		}

		return result, nil
	}
}
