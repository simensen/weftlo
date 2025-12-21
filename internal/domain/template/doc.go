// Package template provides domain entities for Go template rendering with sprig
// function library integration. It enables dynamic configuration file generation
// by combining profile data, project metadata, and configuration settings into
// a template context that can be used with Go's text/template package.
//
// # Overview
//
// This package provides two main components:
//
//   - TemplateContext: A struct that aggregates all data sources available to templates
//   - TemplateEngine: A rendering engine that processes templates with sprig functions
//
// # TemplateContext
//
// TemplateContext aggregates profile information, project paths, timestamps, and
// configuration data into a single struct that can be bound to Go templates.
// All fields are exported for direct template access using dot notation.
//
// Example usage:
//
//	ctx := template.NewTemplateContext(
//		&profile.Profile{Name: "acme/backend"},
//		"/path/to/project",
//		time.Now(),
//		&config.ProjectConfig{Profile: "acme/backend"},
//		&config.GlobalConfig{DefaultProfile: "acme/default"},
//	)
//
// Templates can access context fields directly:
//
//	{{ .Profile.Name }}           // "acme/backend"
//	{{ .ProjectDir }}             // "/path/to/project"
//	{{ .GeneratedAt }}            // time.Time value
//	{{ .ProjectConfig.Profile }}  // "acme/backend"
//	{{ .GlobalConfig.DefaultProfile }} // "acme/default"
//
// # TemplateEngine
//
// TemplateEngine provides template rendering with the full sprig function library.
// It uses text/template (not html/template) for non-HTML output and includes
// all 100+ sprig functions without any security restrictions.
//
// Example usage:
//
//	engine := template.NewTemplateEngine()
//	output, err := engine.Render("Hello {{ .Profile.Name | upper }}", ctx)
//	// output: "Hello ACME/BACKEND"
//
// # Sprig Functions
//
// The engine includes the complete sprig function library:
//
//   - String functions: upper, lower, trim, quote, replace, title, etc.
//   - Environment functions: env, expandenv
//   - Date functions: now, date, dateInZone, etc.
//   - Math functions: add, sub, mul, div, etc.
//   - List functions: list, first, last, append, etc.
//   - Dict functions: dict, get, set, keys, values, etc.
//   - Flow control: default, empty, coalesce, ternary, etc.
//
// For complete sprig documentation, see: https://masterminds.github.io/sprig/
//
// # Template Examples
//
// Basic variable access:
//
//	Profile: {{ .Profile.Name }}
//	Project: {{ .ProjectDir }}
//
// Using sprig functions:
//
//	{{ .Profile.Name | upper }}                    // Uppercase
//	{{ .Profile.Name | replace "/" "_" }}          // Replace characters
//	{{ .Profile.InheritsFrom | default "none" }}   // Default value
//	{{ env "HOME" }}                               // Environment variable
//
// Accessing nested data:
//
//	{{ (index .Profile.TemplateFiles 0).SourcePath }}
//	{{ len .Profile.TemplateFiles }}
//	{{ (first .Profile.TemplateFiles).TargetPath }}
//
// Conditionals and loops:
//
//	{{- if .Profile.InheritsFrom }}
//	Inherits from: {{ .Profile.InheritsFrom }}
//	{{- end }}
//
//	{{- range .Profile.TemplateFiles }}
//	- {{ .SourcePath }} -> {{ .TargetPath }}
//	{{- end }}
//
// # Stability Notes
//
// The TemplateContext exposes the full profile.Profile struct to templates. This
// design decision should be reevaluated whenever the Profile struct changes
// meaningfully to ensure template authors only have access to appropriate data.
// Future versions may introduce a more restricted view of profile data.
//
// # Error Handling
//
// The Render method returns Go template errors directly with additional context.
// Custom error types with enhanced line/column reporting are planned for a
// future specification.
//
// # Future Extensions
//
// The following features are planned for future specifications:
//
//   - include() function for including other template files
//   - includeGlob() function for glob-based includes
//   - Custom error types with line/column parsing
//   - Filesystem integration for template inclusion
package template
