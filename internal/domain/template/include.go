package template

import (
	"bytes"
	"text/template"

	"github.com/Masterminds/sprig/v3"
	"github.com/spf13/afero"
)

// createIncludeFunc creates a closure that implements the include template function.
// The closure captures the render context and template context, allowing it to:
//   - Track and enforce recursion depth limits
//   - Resolve source paths via the FileResolver
//   - Read file content via the afero filesystem
//   - Render included content with the same TemplateContext
//   - Provide descriptive error messages with include chain context
//
// Parameters:
//   - renderCtx: The render-time context with filesystem, resolver, and depth tracking
//   - tmplCtx: The template context data passed to all templates
//
// Returns a function that can be registered in a template.FuncMap under the "include" key.
func createIncludeFunc(renderCtx *RenderContext, tmplCtx *TemplateContext) func(string) (string, error) {
	return func(sourcePath string) (string, error) {
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

		return buf.String(), nil
	}
}
