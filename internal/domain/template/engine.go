package template

import (
	"bytes"
	"text/template"

	"github.com/Masterminds/sprig/v3"
	"github.com/spf13/afero"
)

// TemplateEngine provides template rendering capabilities with sprig function support.
// It uses Go's text/template package for non-HTML output and integrates the full
// sprig function library (100+ functions) including the env function for
// environment variable access.
type TemplateEngine struct {
	// funcMap contains the template functions available during rendering.
	// This is initialized with the full sprig function library.
	funcMap template.FuncMap

	// fs is an optional filesystem for reading included files.
	// When nil, the include function is not available via the Render method.
	// Use NewTemplateEngineWithFs to enable include support.
	fs afero.Fs
}

// NewTemplateEngine creates a new TemplateEngine configured with the full sprig
// function library. The engine is ready for use immediately after construction.
//
// The sprig library provides 100+ template functions including:
//   - String functions: upper, lower, trim, quote, replace, etc.
//   - Environment functions: env, expandenv
//   - Date functions: now, date, dateInZone, etc.
//   - Math functions: add, sub, mul, div, etc.
//   - List functions: list, first, last, append, etc.
//   - Dict functions: dict, get, set, keys, values, etc.
//
// No security restrictions are applied to any sprig functions.
//
// Note: The include function is NOT available when using this constructor.
// Use NewTemplateEngineWithFs and RenderWithContext for include support.
func NewTemplateEngine() *TemplateEngine {
	return &TemplateEngine{
		funcMap: sprig.TxtFuncMap(),
		fs:      nil,
	}
}

// NewTemplateEngineWithFs creates a new TemplateEngine with filesystem support
// for the include function. The filesystem is used to read included files during
// template rendering via RenderWithContext.
//
// Parameters:
//   - fs: The filesystem interface for reading included files.
//     Use afero.NewOsFs() for production and afero.NewMemMapFs() for testing.
//
// The include function is only available when using RenderWithContext with
// a properly configured RenderContext.
func NewTemplateEngineWithFs(fs afero.Fs) *TemplateEngine {
	return &TemplateEngine{
		funcMap: sprig.TxtFuncMap(),
		fs:      fs,
	}
}

// Render executes a template with the provided context data and returns the
// rendered output as a string.
//
// Parameters:
//   - templateContent: The template content as a string (Go template syntax)
//   - ctx: The TemplateContext providing data for template variable binding
//
// Returns:
//   - The rendered output as a string
//   - An error if template parsing or execution fails
//
// The template has access to all fields in the TemplateContext using dot notation:
//   - {{ .Profile.Name }} - Access profile name
//   - {{ .ProjectDir }} - Access project directory
//   - {{ .GeneratedAt }} - Access generation timestamp
//   - {{ .ProjectConfig.Profile }} - Access project config
//   - {{ .GlobalConfig.DefaultProfile }} - Access global config
//
// All sprig functions are available for use in templates:
//   - {{ .Profile.Name | upper }} - Convert to uppercase
//   - {{ env "HOME" }} - Read environment variable
//   - {{ .ProjectDir | trim }} - Trim whitespace
//
// Note: The include function is NOT available via this method.
// Use RenderWithContext for include support.
func (e *TemplateEngine) Render(templateContent string, ctx *TemplateContext) (string, error) {
	// Parse the template with sprig functions
	tmpl, err := template.New("template").Funcs(e.funcMap).Parse(templateContent)
	if err != nil {
		// Extract location information from the error
		loc := parseErrorLocation(err)
		return "", &TemplateSyntaxError{
			SourcePath:   "template",
			Line:         loc.Line,
			Column:       loc.Column,
			Content:      templateContent,
			IncludeChain: nil,
			Err:          err,
		}
	}

	// Execute the template with the provided context
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, ctx); err != nil {
		// Extract location information from the error
		loc := parseErrorLocation(err)
		return "", &TemplateExecutionError{
			SourcePath:   "template",
			Line:         loc.Line,
			Content:      templateContent,
			IncludeChain: nil,
			Err:          err,
		}
	}

	return buf.String(), nil
}

// RenderWithContext executes a template with include, includeGlob, reference, and
// referenceGlob function support. This method enables these template functions for
// modular template composition and cross-file references.
//
// Parameters:
//   - templateContent: The template content as a string (Go template syntax)
//   - tmplCtx: The TemplateContext providing data for template variable binding
//   - renderCtx: The RenderContext providing filesystem, file resolver, router, and depth tracking
//
// Returns:
//   - The rendered output as a string
//   - An error if template parsing, execution, or include resolution fails
//
// The include function is registered per-render and captures:
//   - The filesystem for reading included files
//   - The file resolver for path resolution
//   - The current recursion depth for limit enforcement
//   - The template context for passing to included templates
//
// The includeGlob function matches multiple files using glob patterns and returns
// their concatenated rendered content, enabling modular composition of partials.
//
// The reference function resolves source paths to target installation paths,
// returning the path with an @ prefix for use in cross-file references.
//
// The referenceGlob function resolves multiple source paths matching a glob pattern
// to their target installation paths, returning a sorted slice of @ prefixed paths.
//
// Example template using include, includeGlob, reference, and referenceGlob:
//
//	{{ include "_partials/header.md" }}
//	# Main Content
//	See also: [Git Guide]({{ reference "standards/vcs/git.md" }})
//	Related documents:
//	{{ range referenceGlob "standards/**/*.md" }}
//	- [{{ . }}]({{ . }})
//	{{ end }}
//	{{ includeGlob "_skills/*.md" }}
//	{{ include "_partials/footer.md" }}
func (e *TemplateEngine) RenderWithContext(templateContent string, tmplCtx *TemplateContext, renderCtx *RenderContext) (string, error) {
	// Create function map with sprig functions
	funcMap := sprig.TxtFuncMap()

	// Register include function using the closure factory
	funcMap["include"] = createIncludeFunc(renderCtx, tmplCtx)

	// Register includeGlob function using the closure factory
	funcMap["includeGlob"] = createIncludeGlobFunc(renderCtx, tmplCtx)

	// Register reference function using the closure factory
	funcMap["reference"] = createReferenceFunc(renderCtx, tmplCtx)

	// Register referenceGlob function using the closure factory
	funcMap["referenceGlob"] = createReferenceGlobFunc(renderCtx, tmplCtx)

	// Parse the template with the enhanced function map
	tmpl, err := template.New("template").Funcs(funcMap).Parse(templateContent)
	if err != nil {
		// Extract location information from the error
		loc := parseErrorLocation(err)
		return "", &TemplateSyntaxError{
			SourcePath:   "template",
			Line:         loc.Line,
			Column:       loc.Column,
			Content:      templateContent,
			IncludeChain: renderCtx.IncludeChain,
			Err:          err,
		}
	}

	// Execute the template with the provided context
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, tmplCtx); err != nil {
		// Extract location information from the error
		loc := parseErrorLocation(err)
		return "", &TemplateExecutionError{
			SourcePath:   "template",
			Line:         loc.Line,
			Content:      templateContent,
			IncludeChain: renderCtx.IncludeChain,
			Err:          err,
		}
	}

	return buf.String(), nil
}
