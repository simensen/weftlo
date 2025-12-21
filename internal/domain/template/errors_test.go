package template

import (
	"errors"
	"strings"
	"testing"
)

func TestIncludeNotFoundError_Error(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		err     *IncludeNotFoundError
		wantMsg string
	}{
		{
			name: "formats message with path and empty include chain",
			err: &IncludeNotFoundError{
				SourcePath:   "_partials/header.md",
				IncludeChain: []string{},
			},
			wantMsg: "include file not found: _partials/header.md (include chain: [])",
		},
		{
			name: "formats message with path and single-item include chain",
			err: &IncludeNotFoundError{
				SourcePath:   "_partials/footer.md",
				IncludeChain: []string{"main.md"},
			},
			wantMsg: "include file not found: _partials/footer.md (include chain: [main.md])",
		},
		{
			name: "formats message with path and multi-item include chain",
			err: &IncludeNotFoundError{
				SourcePath:   "_partials/nested.md",
				IncludeChain: []string{"main.md", "_partials/header.md"},
			},
			wantMsg: "include file not found: _partials/nested.md (include chain: [main.md _partials/header.md])",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := tt.err.Error()
			if got != tt.wantMsg {
				t.Errorf("IncludeNotFoundError.Error() = %q, want %q", got, tt.wantMsg)
			}
		})
	}
}

func TestIncludeDepthError_Error(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		err     *IncludeDepthError
		wantMsg string
	}{
		{
			name: "formats message with depth info and include chain",
			err: &IncludeDepthError{
				MaxDepth:     10,
				CurrentDepth: 11,
				IncludeChain: []string{"main.md", "a.md", "b.md", "c.md", "d.md", "e.md", "f.md", "g.md", "h.md", "i.md"},
			},
			wantMsg: "include depth limit exceeded: current depth 11, max depth 10 (include chain: [main.md a.md b.md c.md d.md e.md f.md g.md h.md i.md])",
		},
		{
			name: "formats message with minimal chain",
			err: &IncludeDepthError{
				MaxDepth:     10,
				CurrentDepth: 10,
				IncludeChain: []string{"root.md"},
			},
			wantMsg: "include depth limit exceeded: current depth 10, max depth 10 (include chain: [root.md])",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := tt.err.Error()
			if got != tt.wantMsg {
				t.Errorf("IncludeDepthError.Error() = %q, want %q", got, tt.wantMsg)
			}
		})
	}
}

func TestIncludeSyntaxError_Error(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		err     *IncludeSyntaxError
		wantMsg string
	}{
		{
			name: "wraps underlying parse error correctly",
			err: &IncludeSyntaxError{
				SourcePath: "_partials/broken.md",
				Err:        errors.New("unexpected EOF"),
			},
			wantMsg: "syntax error in included file _partials/broken.md: unexpected EOF",
		},
		{
			name: "handles complex error message",
			err: &IncludeSyntaxError{
				SourcePath: "templates/invalid.md",
				Err:        errors.New("template: template:1: unexpected \"}\" in operand"),
			},
			wantMsg: "syntax error in included file templates/invalid.md: template: template:1: unexpected \"}\" in operand",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := tt.err.Error()
			if got != tt.wantMsg {
				t.Errorf("IncludeSyntaxError.Error() = %q, want %q", got, tt.wantMsg)
			}
		})
	}
}

func TestIncludeSyntaxError_Unwrap(t *testing.T) {
	t.Parallel()
	underlyingErr := errors.New("parse error")
	err := &IncludeSyntaxError{
		SourcePath: "test.md",
		Err:        underlyingErr,
	}

	unwrapped := err.Unwrap()
	if unwrapped != underlyingErr {
		t.Errorf("IncludeSyntaxError.Unwrap() = %v, want %v", unwrapped, underlyingErr)
	}

	// Verify errors.Is works with the wrapped error
	if !errors.Is(err, underlyingErr) {
		t.Error("errors.Is(err, underlyingErr) = false, want true")
	}
}

func TestIncludeExecutionError_Error(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		err     *IncludeExecutionError
		wantMsg string
	}{
		{
			name: "wraps underlying execution error correctly",
			err: &IncludeExecutionError{
				SourcePath: "_partials/dynamic.md",
				Err:        errors.New("nil pointer evaluating interface"),
			},
			wantMsg: "execution error in included file _partials/dynamic.md: nil pointer evaluating interface",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := tt.err.Error()
			if got != tt.wantMsg {
				t.Errorf("IncludeExecutionError.Error() = %q, want %q", got, tt.wantMsg)
			}
		})
	}
}

func TestIncludeExecutionError_Unwrap(t *testing.T) {
	t.Parallel()
	underlyingErr := errors.New("execution error")
	err := &IncludeExecutionError{
		SourcePath: "test.md",
		Err:        underlyingErr,
	}

	unwrapped := err.Unwrap()
	if unwrapped != underlyingErr {
		t.Errorf("IncludeExecutionError.Unwrap() = %v, want %v", unwrapped, underlyingErr)
	}

	// Verify errors.Is works with the wrapped error
	if !errors.Is(err, underlyingErr) {
		t.Error("errors.Is(err, underlyingErr) = false, want true")
	}
}

// Test: IncludeGlobPatternError.Error() formats correctly with pattern
func TestIncludeGlobPatternError_Error(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		err     *IncludeGlobPatternError
		wantMsg string
	}{
		{
			name: "formats message with pattern and underlying error",
			err: &IncludeGlobPatternError{
				Pattern: "_skills/[invalid",
				Err:     errors.New("syntax error in pattern"),
			},
			wantMsg: `invalid glob pattern "_skills/[invalid": syntax error in pattern`,
		},
		{
			name: "handles pattern with special characters",
			err: &IncludeGlobPatternError{
				Pattern: "_skills/[abc",
				Err:     errors.New("missing closing bracket"),
			},
			wantMsg: `invalid glob pattern "_skills/[abc": missing closing bracket`,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := tt.err.Error()
			if got != tt.wantMsg {
				t.Errorf("IncludeGlobPatternError.Error() = %q, want %q", got, tt.wantMsg)
			}
		})
	}
}

// Test: IncludeGlobPatternError.Unwrap() returns underlying error
func TestIncludeGlobPatternError_Unwrap(t *testing.T) {
	t.Parallel()
	underlyingErr := errors.New("bad pattern syntax")
	err := &IncludeGlobPatternError{
		Pattern: "_skills/*.md",
		Err:     underlyingErr,
	}

	unwrapped := err.Unwrap()
	if unwrapped != underlyingErr {
		t.Errorf("IncludeGlobPatternError.Unwrap() = %v, want %v", unwrapped, underlyingErr)
	}

	// Verify errors.Is works with the wrapped error
	if !errors.Is(err, underlyingErr) {
		t.Error("errors.Is(err, underlyingErr) = false, want true")
	}
}

// Test: parseErrorLocation extracts line, column, and message from Go template errors
func TestParseErrorLocation(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		errMsg     string
		wantLine   int
		wantColumn int
		wantMsg    string
	}{
		{
			name:       "parses parse error with line and column",
			errMsg:     "template: template:5:12: unexpected \"}\" in operand",
			wantLine:   5,
			wantColumn: 12,
			wantMsg:    "unexpected \"}\" in operand",
		},
		{
			name:       "parses execution error with line only",
			errMsg:     "template: template:3: nil pointer evaluating interface {}",
			wantLine:   3,
			wantColumn: 0,
			wantMsg:    "nil pointer evaluating interface {}",
		},
		{
			name:       "handles error without location info",
			errMsg:     "some generic error without location",
			wantLine:   0,
			wantColumn: 0,
			wantMsg:    "some generic error without location",
		},
		{
			name:       "parses error with custom template name",
			errMsg:     "template: mytemplate:10:5: function \"badFunc\" not defined",
			wantLine:   10,
			wantColumn: 5,
			wantMsg:    "function \"badFunc\" not defined",
		},
		{
			name:       "parses error with path-like template name",
			errMsg:     "template: /path/to/file.md:7: missing value for command",
			wantLine:   7,
			wantColumn: 0,
			wantMsg:    "missing value for command",
		},
		{
			name:       "handles error with only template prefix but no location",
			errMsg:     "template: unexpected EOF",
			wantLine:   0,
			wantColumn: 0,
			wantMsg:    "template: unexpected EOF",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			loc := parseErrorLocation(errors.New(tt.errMsg))
			if loc.Line != tt.wantLine {
				t.Errorf("parseErrorLocation().Line = %d, want %d", loc.Line, tt.wantLine)
			}
			if loc.Column != tt.wantColumn {
				t.Errorf("parseErrorLocation().Column = %d, want %d", loc.Column, tt.wantColumn)
			}
			if loc.Message != tt.wantMsg {
				t.Errorf("parseErrorLocation().Message = %q, want %q", loc.Message, tt.wantMsg)
			}
		})
	}
}

// Test: extractContext extracts lines of context around an error line
func TestExtractContext(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		content   string
		line      int
		wantLines []string // Expected substrings in output (flexible matching)
	}{
		{
			name: "extracts context from middle of multi-line template",
			content: `line 1 content
line 2 content
line 3 content
line 4 content
{{ .Invalid }
line 6 content
line 7 content`,
			line: 5,
			wantLines: []string{
				"3 | line 3 content",
				"4 | line 4 content",
				">>> ", // Error line marker
				"5 | {{ .Invalid }",
				"6 | line 6 content",
			},
		},
		{
			name: "handles error on first line of template",
			content: `{{ .Invalid }
line 2 content
line 3 content
line 4 content`,
			line: 1,
			wantLines: []string{
				">>> ",
				"1 | {{ .Invalid }",
				"2 | line 2 content",
			},
		},
		{
			name: "handles error on last line of template",
			content: `line 1 content
line 2 content
line 3 content
{{ .Invalid }`,
			line: 4,
			wantLines: []string{
				"2 | line 2 content",
				"3 | line 3 content",
				">>> ",
				"4 | {{ .Invalid }",
			},
		},
		{
			name:    "handles single-line template",
			content: `{{ .Invalid }`,
			line:    1,
			wantLines: []string{
				">>> ",
				"1 | {{ .Invalid }",
			},
		},
		{
			name: "handles template with very long lines without truncation",
			content: `short line
` + strings.Repeat("a", 500) + `
another short line`,
			line: 2,
			wantLines: []string{
				"1 | short line",
				">>> ",
				"2 | " + strings.Repeat("a", 500), // Verify long line is not truncated
				"3 | another short line",
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := extractContext(tt.content, tt.line)
			for _, wantLine := range tt.wantLines {
				if !strings.Contains(got, wantLine) {
					t.Errorf("extractContext() missing expected content %q\ngot:\n%s", wantLine, got)
				}
			}
		})
	}
}

// Additional test: verify extractContext returns empty string for invalid line numbers
func TestExtractContext_InvalidLineNumbers(t *testing.T) {
	t.Parallel()
	content := `line 1
line 2
line 3`

	tests := []struct {
		name string
		line int
	}{
		{"zero line number", 0},
		{"negative line number", -1},
		{"line number beyond content", 10},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := extractContext(content, tt.line)
			if got != "" {
				t.Errorf("extractContext() with invalid line %d = %q, want empty string", tt.line, got)
			}
		})
	}
}

// Test: formatIncludeChain formats include chain hierarchy for error display
func TestFormatIncludeChain(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		chain   []string
		want    string
		wantNil bool // true if we expect empty string
	}{
		{
			name:    "returns empty string for empty chain",
			chain:   []string{},
			wantNil: true,
		},
		{
			name:    "returns empty string for nil chain",
			chain:   nil,
			wantNil: true,
		},
		{
			name:  "formats single-item chain",
			chain: []string{"main.md"},
			want: `Include trace:
    main.md (error here)
`,
		},
		{
			name:  "formats multi-level nested chain",
			chain: []string{"main.md", "_partials/header.md", "_partials/broken.md"},
			want: `Include trace:
    main.md
    -> _partials/header.md
    -> _partials/broken.md (error here)
`,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := formatIncludeChain(tt.chain)
			if tt.wantNil {
				if got != "" {
					t.Errorf("formatIncludeChain() = %q, want empty string", got)
				}
				return
			}
			if got != tt.want {
				t.Errorf("formatIncludeChain() = %q, want %q", got, tt.want)
			}
		})
	}
}

// Test: TemplateSyntaxError.Error() output format with line and column
func TestTemplateSyntaxError_Error_WithLineAndColumn(t *testing.T) {
	t.Parallel()
	err := &TemplateSyntaxError{
		SourcePath: "config/main.md",
		Line:       5,
		Column:     12,
		Content: `line 1 content
line 2 content
line 3 content
line 4 content
{{ .Invalid }
line 6 content`,
		IncludeChain: nil,
		Err:          errors.New("unexpected \"}\" in operand"),
	}

	got := err.Error()

	// Verify the error message contains expected components
	expectedParts := []string{
		"config/main.md",     // Source path
		"line 5",             // Line number
		"column 12",          // Column number
		"unexpected \"}\"",   // Error message
		">>> ",               // Context marker
		"5 | {{ .Invalid }",  // Error line content
		"4 | line 4 content", // Context before
		"6 | line 6 content", // Context after
	}

	for _, part := range expectedParts {
		if !strings.Contains(got, part) {
			t.Errorf("TemplateSyntaxError.Error() missing expected part %q\ngot:\n%s", part, got)
		}
	}
}

// Test: TemplateSyntaxError.Error() output format with line only (no column)
func TestTemplateSyntaxError_Error_WithLineOnly(t *testing.T) {
	t.Parallel()
	err := &TemplateSyntaxError{
		SourcePath: "template",
		Line:       3,
		Column:     0, // No column available
		Content: `line 1
line 2
{{ .Missing }
line 4`,
		IncludeChain: nil,
		Err:          errors.New("missing value for command"),
	}

	got := err.Error()

	// Should contain line but not "column 0"
	if !strings.Contains(got, "line 3") {
		t.Errorf("TemplateSyntaxError.Error() missing line number\ngot:\n%s", got)
	}
	if strings.Contains(got, "column 0") {
		t.Errorf("TemplateSyntaxError.Error() should not show column 0\ngot:\n%s", got)
	}
	if !strings.Contains(got, ">>> ") {
		t.Errorf("TemplateSyntaxError.Error() missing context marker\ngot:\n%s", got)
	}
}

// Test: TemplateSyntaxError.Error() output with context display
func TestTemplateSyntaxError_Error_WithContextDisplay(t *testing.T) {
	t.Parallel()
	err := &TemplateSyntaxError{
		SourcePath: "test.md",
		Line:       1,
		Column:     5,
		Content:    `{{ .Bad }`,
		Err:        errors.New("unclosed action"),
	}

	got := err.Error()

	// Verify context is displayed
	if !strings.Contains(got, ">>> ") {
		t.Errorf("TemplateSyntaxError.Error() missing context marker\ngot:\n%s", got)
	}
	if !strings.Contains(got, "1 | {{ .Bad }") {
		t.Errorf("TemplateSyntaxError.Error() missing error line content\ngot:\n%s", got)
	}
}

// Test: TemplateSyntaxError.Error() output with include chain
func TestTemplateSyntaxError_Error_WithIncludeChain(t *testing.T) {
	t.Parallel()
	err := &TemplateSyntaxError{
		SourcePath:   "_partials/broken.md",
		Line:         2,
		Column:       8,
		Content:      "line 1\n{{ .Bad }",
		IncludeChain: []string{"main.md", "_partials/header.md", "_partials/broken.md"},
		Err:          errors.New("unexpected EOF"),
	}

	got := err.Error()

	// Verify include chain is displayed
	expectedChainParts := []string{
		"Include trace:",
		"main.md",
		"-> _partials/header.md",
		"-> _partials/broken.md (error here)",
	}

	for _, part := range expectedChainParts {
		if !strings.Contains(got, part) {
			t.Errorf("TemplateSyntaxError.Error() missing include chain part %q\ngot:\n%s", part, got)
		}
	}
}

// Test: TemplateSyntaxError.Unwrap() returns underlying error
func TestTemplateSyntaxError_Unwrap(t *testing.T) {
	t.Parallel()
	underlyingErr := errors.New("parse error from template engine")
	err := &TemplateSyntaxError{
		SourcePath: "test.md",
		Line:       1,
		Column:     1,
		Content:    "{{ .Bad }",
		Err:        underlyingErr,
	}

	unwrapped := err.Unwrap()
	if unwrapped != underlyingErr {
		t.Errorf("TemplateSyntaxError.Unwrap() = %v, want %v", unwrapped, underlyingErr)
	}
}

// Test: TemplateSyntaxError errors.Is() compatibility
func TestTemplateSyntaxError_ErrorsIs(t *testing.T) {
	t.Parallel()
	underlyingErr := errors.New("template parse failure")
	err := &TemplateSyntaxError{
		SourcePath: "config.md",
		Line:       5,
		Column:     10,
		Content:    "content",
		Err:        underlyingErr,
	}

	// Verify errors.Is works with the wrapped error
	if !errors.Is(err, underlyingErr) {
		t.Error("errors.Is(err, underlyingErr) = false, want true")
	}

	// Verify errors.Is returns false for unrelated errors
	unrelatedErr := errors.New("different error")
	if errors.Is(err, unrelatedErr) {
		t.Error("errors.Is(err, unrelatedErr) = true, want false")
	}
}

// Test: TemplateExecutionError.Error() output format with line number
func TestTemplateExecutionError_Error_WithLineNumber(t *testing.T) {
	t.Parallel()
	err := &TemplateExecutionError{
		SourcePath: "config/main.md",
		Line:       5,
		Content: `line 1 content
line 2 content
line 3 content
line 4 content
{{ .Missing.Field }}
line 6 content`,
		IncludeChain: nil,
		Err:          errors.New("nil pointer evaluating interface {}"),
	}

	got := err.Error()

	// Verify the error message contains expected components
	expectedParts := []string{
		"config/main.md",           // Source path
		"line 5",                   // Line number
		"nil pointer evaluating",   // Error message
		">>> ",                     // Context marker
		"5 | {{ .Missing.Field }}", // Error line content
	}

	for _, part := range expectedParts {
		if !strings.Contains(got, part) {
			t.Errorf("TemplateExecutionError.Error() missing expected part %q\ngot:\n%s", part, got)
		}
	}
}

// Test: TemplateExecutionError.Error() output with context display
func TestTemplateExecutionError_Error_WithContextDisplay(t *testing.T) {
	t.Parallel()
	err := &TemplateExecutionError{
		SourcePath: "template",
		Line:       3,
		Content: `line 1
line 2
{{ call .UndefinedFunc }}
line 4`,
		IncludeChain: nil,
		Err:          errors.New("error calling UndefinedFunc"),
	}

	got := err.Error()

	// Verify context display with surrounding lines
	expectedParts := []string{
		">>> ",                          // Error line marker
		"3 | {{ call .UndefinedFunc }}", // Error line content
		"2 | line 2",                    // Context before
		"4 | line 4",                    // Context after
	}

	for _, part := range expectedParts {
		if !strings.Contains(got, part) {
			t.Errorf("TemplateExecutionError.Error() missing expected part %q\ngot:\n%s", part, got)
		}
	}
}

// Test: TemplateExecutionError.Error() output with include chain (nested include scenario)
func TestTemplateExecutionError_Error_WithIncludeChain(t *testing.T) {
	t.Parallel()
	err := &TemplateExecutionError{
		SourcePath:   "_partials/dynamic.md",
		Line:         2,
		Content:      "line 1\n{{ .Missing.Value }}",
		IncludeChain: []string{"main.md", "_partials/header.md", "_partials/dynamic.md"},
		Err:          errors.New("nil pointer evaluating interface {}"),
	}

	got := err.Error()

	// Verify include chain is displayed
	expectedChainParts := []string{
		"Include trace:",
		"main.md",
		"-> _partials/header.md",
		"-> _partials/dynamic.md (error here)",
	}

	for _, part := range expectedChainParts {
		if !strings.Contains(got, part) {
			t.Errorf("TemplateExecutionError.Error() missing include chain part %q\ngot:\n%s", part, got)
		}
	}
}

// Test: TemplateExecutionError.Unwrap() returns underlying error
func TestTemplateExecutionError_Unwrap(t *testing.T) {
	t.Parallel()
	underlyingErr := errors.New("execution error from template engine")
	err := &TemplateExecutionError{
		SourcePath: "test.md",
		Line:       1,
		Content:    "{{ .Missing }}",
		Err:        underlyingErr,
	}

	unwrapped := err.Unwrap()
	if unwrapped != underlyingErr {
		t.Errorf("TemplateExecutionError.Unwrap() = %v, want %v", unwrapped, underlyingErr)
	}
}

// Test: TemplateExecutionError errors.Is() compatibility
func TestTemplateExecutionError_ErrorsIs(t *testing.T) {
	t.Parallel()
	underlyingErr := errors.New("template execution failure")
	err := &TemplateExecutionError{
		SourcePath: "config.md",
		Line:       5,
		Content:    "content",
		Err:        underlyingErr,
	}

	// Verify errors.Is works with the wrapped error
	if !errors.Is(err, underlyingErr) {
		t.Error("errors.Is(err, underlyingErr) = false, want true")
	}

	// Verify errors.Is returns false for unrelated errors
	unrelatedErr := errors.New("different error")
	if errors.Is(err, unrelatedErr) {
		t.Error("errors.Is(err, unrelatedErr) = true, want false")
	}
}
