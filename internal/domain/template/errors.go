package template

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// IncludeNotFoundError is returned when an include function cannot find
// the specified file in the merged profile's file map.
type IncludeNotFoundError struct {
	// SourcePath is the path that was passed to the include function.
	SourcePath string
	// IncludeChain tracks the sequence of files that led to this include,
	// useful for debugging nested include failures.
	IncludeChain []string
}

func (e *IncludeNotFoundError) Error() string {
	return fmt.Sprintf("include file not found: %s (include chain: %v)", e.SourcePath, e.IncludeChain)
}

// IncludeDepthError is returned when the include function exceeds the
// maximum allowed recursion depth. This prevents infinite loops from
// circular includes.
type IncludeDepthError struct {
	// MaxDepth is the configured maximum recursion depth (typically 10).
	MaxDepth int
	// CurrentDepth is the depth at which the error was triggered.
	CurrentDepth int
	// IncludeChain tracks the sequence of files that led to this depth limit,
	// useful for debugging circular or deeply nested includes.
	IncludeChain []string
}

func (e *IncludeDepthError) Error() string {
	return fmt.Sprintf("include depth limit exceeded: current depth %d, max depth %d (include chain: %v)",
		e.CurrentDepth, e.MaxDepth, e.IncludeChain)
}

// IncludeSyntaxError is returned when an included file contains invalid
// Go template syntax that cannot be parsed.
type IncludeSyntaxError struct {
	// SourcePath is the path of the file that failed to parse.
	SourcePath string
	// Err is the underlying parse error from the template parser.
	Err error
}

func (e *IncludeSyntaxError) Error() string {
	return fmt.Sprintf("syntax error in included file %s: %v", e.SourcePath, e.Err)
}

func (e *IncludeSyntaxError) Unwrap() error {
	return e.Err
}

// IncludeExecutionError is returned when an included template fails during
// execution (e.g., nil pointer, missing field, function error).
type IncludeExecutionError struct {
	// SourcePath is the path of the file that failed to execute.
	SourcePath string
	// Err is the underlying execution error from the template executor.
	Err error
}

func (e *IncludeExecutionError) Error() string {
	return fmt.Sprintf("execution error in included file %s: %v", e.SourcePath, e.Err)
}

func (e *IncludeExecutionError) Unwrap() error {
	return e.Err
}

// IncludeGlobPatternError is returned when an includeGlob function receives
// an invalid glob pattern that cannot be parsed by filepath.Match.
type IncludeGlobPatternError struct {
	// Pattern is the glob pattern that failed to parse.
	Pattern string
	// Err is the underlying error from filepath.Match (typically filepath.ErrBadPattern).
	Err error
}

func (e *IncludeGlobPatternError) Error() string {
	return fmt.Sprintf("invalid glob pattern %q: %v", e.Pattern, e.Err)
}

func (e *IncludeGlobPatternError) Unwrap() error {
	return e.Err
}

// ReferencePartialError is returned when reference() is called with a partial source path.
// Partials can only be included, not referenced, because they are not installed to the
// target project.
type ReferencePartialError struct {
	// SourcePath is the path that was passed to the reference function.
	SourcePath string
}

func (e *ReferencePartialError) Error() string {
	return fmt.Sprintf("reference: cannot reference partial %q (partials can only be included, not referenced)", e.SourcePath)
}

// ReferenceGlobPatternError is returned when referenceGlob() is called with an invalid
// glob pattern that cannot be parsed by filepath.Match.
type ReferenceGlobPatternError struct {
	// Pattern is the glob pattern that failed to parse.
	Pattern string
	// Err is the underlying error from filepath.Match (typically filepath.ErrBadPattern).
	Err error
}

func (e *ReferenceGlobPatternError) Error() string {
	return fmt.Sprintf("referenceGlob: invalid glob pattern %q: %v", e.Pattern, e.Err)
}

func (e *ReferenceGlobPatternError) Unwrap() error {
	return e.Err
}

// TemplateSyntaxError is returned when a template contains invalid Go template
// syntax that cannot be parsed. It provides detailed error information including
// the file path, line/column location, surrounding context, and include chain
// for nested template errors.
type TemplateSyntaxError struct {
	// SourcePath is the file path where the error occurred, or "template" for inline content.
	SourcePath string
	// Line is the 1-based line number where the error occurred, or 0 if unknown.
	Line int
	// Column is the 1-based column number where the error occurred, or 0 if not available.
	Column int
	// Content is the original template content, used for extracting context around the error.
	Content string
	// IncludeChain tracks the sequence of files that led to this error when
	// the error occurs in a nested include.
	IncludeChain []string
	// Err is the underlying error from template.Parse().
	Err error
}

// Error returns a formatted error message with file path, location, context,
// and include chain (if present).
func (e *TemplateSyntaxError) Error() string {
	var builder strings.Builder

	// Header with source path and location
	builder.WriteString(fmt.Sprintf("template syntax error in %s", e.SourcePath))
	if e.Line > 0 {
		builder.WriteString(fmt.Sprintf(" at line %d", e.Line))
		if e.Column > 0 {
			builder.WriteString(fmt.Sprintf(", column %d", e.Column))
		}
	}
	builder.WriteString(":\n")

	// Error message
	if e.Err != nil {
		builder.WriteString(fmt.Sprintf("    %s\n", e.Err.Error()))
	}

	// Context display
	if e.Line > 0 && e.Content != "" {
		context := extractContext(e.Content, e.Line)
		if context != "" {
			builder.WriteString("\n")
			builder.WriteString(context)
		}
	}

	// Include chain display
	if len(e.IncludeChain) > 0 {
		builder.WriteString("\n")
		builder.WriteString(formatIncludeChain(e.IncludeChain))
	}

	return builder.String()
}

// Unwrap returns the underlying error from template.Parse(), enabling
// error chaining with errors.Is() and errors.As().
func (e *TemplateSyntaxError) Unwrap() error {
	return e.Err
}

// TemplateExecutionError is returned when a template fails during execution
// (e.g., nil pointer dereference, missing field, function error). It provides
// detailed error information including the file path, line number, surrounding
// context, and include chain for nested template errors.
type TemplateExecutionError struct {
	// SourcePath is the file path where the error occurred, or "template" for inline content.
	SourcePath string
	// Line is the 1-based line number where the error occurred, or 0 if unknown.
	// Note: Column is typically not available for execution errors.
	Line int
	// Content is the original template content, used for extracting context around the error.
	Content string
	// IncludeChain tracks the sequence of files that led to this error when
	// the error occurs in a nested include.
	IncludeChain []string
	// Err is the underlying error from template.Execute().
	Err error
}

// Error returns a formatted error message with file path, line number, context,
// and include chain (if present).
func (e *TemplateExecutionError) Error() string {
	var builder strings.Builder

	// Header with source path and location
	builder.WriteString(fmt.Sprintf("template execution error in %s", e.SourcePath))
	if e.Line > 0 {
		builder.WriteString(fmt.Sprintf(" at line %d", e.Line))
	}
	builder.WriteString(":\n")

	// Error message
	if e.Err != nil {
		builder.WriteString(fmt.Sprintf("    %s\n", e.Err.Error()))
	}

	// Context display
	if e.Line > 0 && e.Content != "" {
		context := extractContext(e.Content, e.Line)
		if context != "" {
			builder.WriteString("\n")
			builder.WriteString(context)
		}
	}

	// Include chain display
	if len(e.IncludeChain) > 0 {
		builder.WriteString("\n")
		builder.WriteString(formatIncludeChain(e.IncludeChain))
	}

	return builder.String()
}

// Unwrap returns the underlying error from template.Execute(), enabling
// error chaining with errors.Is() and errors.As().
func (e *TemplateExecutionError) Unwrap() error {
	return e.Err
}

// ErrorLocation contains the parsed location information from a Go template error.
// Line and Column are 0 when the location could not be extracted from the error.
type ErrorLocation struct {
	// Line is the 1-based line number where the error occurred, or 0 if unknown.
	Line int
	// Column is the 1-based column number where the error occurred, or 0 if unknown.
	Column int
	// Message is the error message with location prefix removed.
	Message string
}

// parseErrorLocation extracts line, column, and message from Go template errors.
// Go template errors follow these patterns:
//   - Parse errors: "template: name:line:col: message"
//   - Execution errors: "template: name:line: message"
//
// The function handles edge cases gracefully:
//   - If location info is missing, Line and Column are 0
//   - The Message field contains the original error message when parsing fails
//
// Parameters:
//   - err: An error from template.Parse() or template.Execute()
//
// Returns:
//   - ErrorLocation with extracted line, column (if available), and message
func parseErrorLocation(err error) ErrorLocation {
	if err == nil {
		return ErrorLocation{}
	}

	errStr := err.Error()

	// Pattern to match Go template error format:
	// "template: name:line:col: message" or "template: name:line: message"
	// The template name can contain slashes (paths) but cannot contain colons in the name portion.
	// We look for patterns like :number:number: or :number: at the end of the prefix.
	//
	// Regex explanation:
	// ^template:\s*     - starts with "template: " (with optional spaces)
	// (.+?)             - non-greedy capture of template name (can include paths with slashes)
	// :(\d+)            - colon followed by line number (required)
	// (?::(\d+))?       - optional colon followed by column number
	// :\s*              - colon and optional whitespace before message
	// (.+)$             - the rest is the message
	pattern := regexp.MustCompile(`^template:\s*(.+?):(\d+)(?::(\d+))?:\s*(.+)$`)

	matches := pattern.FindStringSubmatch(errStr)
	if matches == nil {
		// Could not parse location info, return original message
		return ErrorLocation{
			Line:    0,
			Column:  0,
			Message: errStr,
		}
	}

	// matches[0] = full match
	// matches[1] = template name
	// matches[2] = line number
	// matches[3] = column number (may be empty)
	// matches[4] = message

	line, _ := strconv.Atoi(matches[2])
	column := 0
	if matches[3] != "" {
		column, _ = strconv.Atoi(matches[3])
	}

	return ErrorLocation{
		Line:    line,
		Column:  column,
		Message: matches[4],
	}
}

// extractContext extracts lines of template content surrounding an error line.
// It displays 2 lines before and 1 line after the error line when available,
// with line numbers and a >>> marker to indicate the error line.
//
// The output format follows this pattern:
//
//	    3 |   some content
//	    4 |   more content
//	>>> 5 |   {{ .Invalid }
//	    6 |   trailing content
//
// Parameters:
//   - content: The full template content as a string
//   - line: The 1-based line number where the error occurred
//
// Returns:
//   - A formatted string with context lines, or empty string if line is invalid
//
// Edge cases handled:
//   - First line errors: shows lines 1-3 (or fewer if template is shorter)
//   - Last line errors: shows last 3 lines (or fewer if template is shorter)
//   - Single-line templates: shows just the one line
//   - Invalid line numbers (0, negative, or beyond content): returns empty string
func extractContext(content string, line int) string {
	// Handle invalid line numbers
	if line <= 0 {
		return ""
	}

	// Split content into lines
	lines := strings.Split(content, "\n")
	totalLines := len(lines)

	// Handle line number beyond content
	if line > totalLines {
		return ""
	}

	// Calculate context window (2 lines before, 1 line after when possible)
	// Line numbers are 1-based, but slice indices are 0-based
	startLine := line - 2 // 2 lines before
	if startLine < 1 {
		startLine = 1
	}

	endLine := line + 1 // 1 line after
	if endLine > totalLines {
		endLine = totalLines
	}

	// Build the context output
	var builder strings.Builder

	// Calculate the width needed for line numbers (for alignment)
	lineNumWidth := len(fmt.Sprintf("%d", endLine))
	if lineNumWidth < 4 {
		lineNumWidth = 4 // Minimum width of 4 for consistent formatting
	}

	for i := startLine; i <= endLine; i++ {
		lineContent := lines[i-1] // Convert to 0-based index

		if i == line {
			// Error line - mark with >>>
			builder.WriteString(fmt.Sprintf(">>> %*d | %s\n", lineNumWidth, i, lineContent))
		} else {
			// Context line - indent with spaces
			builder.WriteString(fmt.Sprintf("    %*d | %s\n", lineNumWidth, i, lineContent))
		}
	}

	return builder.String()
}

// formatIncludeChain formats an include chain for display in error messages.
// It shows the hierarchy of files from the root template to the file where
// the error occurred, using arrows to indicate the include relationships.
//
// The output format follows this pattern:
//
//	Include trace:
//	    main.md
//	    -> _partials/header.md
//	    -> _partials/broken.md (error here)
//
// Parameters:
//   - chain: A slice of file paths representing the include hierarchy,
//     from root (index 0) to the file with the error (last index)
//
// Returns:
//   - A formatted string showing the include hierarchy, or empty string if chain is empty
//
// Edge cases handled:
//   - Empty or nil chain: returns empty string
//   - Single-item chain: shows the file with "(error here)" marker
//   - Multi-level chain: shows hierarchy with arrows and "(error here)" on last file
func formatIncludeChain(chain []string) string {
	if len(chain) == 0 {
		return ""
	}

	var builder strings.Builder
	builder.WriteString("Include trace:\n")

	for i, file := range chain {
		if i == 0 {
			// First file (root) - no arrow prefix
			if len(chain) == 1 {
				// Single-item chain - mark as error location
				builder.WriteString(fmt.Sprintf("    %s (error here)\n", file))
			} else {
				builder.WriteString(fmt.Sprintf("    %s\n", file))
			}
		} else if i == len(chain)-1 {
			// Last file - mark as error location
			builder.WriteString(fmt.Sprintf("    -> %s (error here)\n", file))
		} else {
			// Middle files - show with arrow
			builder.WriteString(fmt.Sprintf("    -> %s\n", file))
		}
	}

	return builder.String()
}
