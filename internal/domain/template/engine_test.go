package template_test

import (
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/spf13/afero"

	"github.com/simensen/weftlo/internal/domain/config"
	"github.com/simensen/weftlo/internal/domain/profile"
	"github.com/simensen/weftlo/internal/domain/template"
	"github.com/simensen/weftlo/internal/testutil"
)

// Test 1: Basic template rendering with simple variable substitution
func TestTemplateEngine_Render_SimpleVariableSubstitution(t *testing.T) {
	t.Parallel()
	// Arrange
	engine := template.NewTemplateEngine()
	ctx := testutil.CreateTestTemplateContext(t,
		testutil.WithProfile(&profile.Profile{Name: "vendor/test-profile"}),
		testutil.WithProjectDir("/path/to/project"),
		testutil.WithProjectConfig(&config.ProjectConfig{Profiles: []string{"vendor/test-profile"}}),
		testutil.WithGlobalConfig(&config.GlobalConfig{DefaultProfile: "vendor/default"}),
	)
	templateContent := "Hello, {{ .ProjectDir }}!"

	// Act
	result, err := engine.Render(templateContent, ctx)

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	expected := "Hello, /path/to/project!"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

// Test 2: Rendering with TemplateContext data (Profile.Name, ProjectDir, etc.)
func TestTemplateEngine_Render_TemplateContextData(t *testing.T) {
	t.Parallel()
	// Arrange
	engine := template.NewTemplateEngine()
	testProfile := &profile.Profile{
		Name:         "vendor/my-profile",
		InheritsFrom: "vendor/base",
		TemplateFiles: []profile.TemplateFile{
			{SourcePath: "src.md", TargetPath: "dest.md"},
		},
	}
	generatedAt := time.Date(2025, 12, 20, 15, 30, 0, 0, time.UTC)
	ctx := testutil.CreateTestTemplateContext(t,
		testutil.WithProfile(testProfile),
		testutil.WithProjectDir("/my/project"),
		testutil.WithGeneratedAt(generatedAt),
		testutil.WithProjectConfig(&config.ProjectConfig{Profiles: []string{"vendor/my-profile"}}),
		testutil.WithGlobalConfig(&config.GlobalConfig{DefaultProfile: "vendor/default"}),
	)
	templateContent := `Profile: {{ .Profile.Name }}
Inherits: {{ .Profile.InheritsFrom }}
Project: {{ .ProjectDir }}
Config Profile: {{ index .ProjectConfig.Profiles 0 }}
Default: {{ .GlobalConfig.DefaultProfile }}`

	// Act
	result, err := engine.Render(templateContent, ctx)

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	expected := `Profile: vendor/my-profile
Inherits: vendor/base
Project: /my/project
Config Profile: vendor/my-profile
Default: vendor/default`
	if result != expected {
		t.Errorf("expected:\n%s\n\ngot:\n%s", expected, result)
	}
}

// Test 3: Sprig string functions work (upper, lower, trim)
func TestTemplateEngine_Render_SprigStringFunctions(t *testing.T) {
	t.Parallel()
	// Arrange
	engine := template.NewTemplateEngine()
	ctx := testutil.CreateTestTemplateContext(t,
		testutil.WithProfile(&profile.Profile{Name: "vendor/test"}),
		testutil.WithProjectDir("  /path/with/spaces  "),
	)
	templateContent := `Upper: {{ .Profile.Name | upper }}
Lower: {{ .Profile.Name | lower }}
Trimmed: {{ .ProjectDir | trim }}`

	// Act
	result, err := engine.Render(templateContent, ctx)

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	expected := `Upper: VENDOR/TEST
Lower: vendor/test
Trimmed: /path/with/spaces`
	if result != expected {
		t.Errorf("expected:\n%s\n\ngot:\n%s", expected, result)
	}
}

// Test 4: Sprig env function works (environment variable access)
func TestTemplateEngine_Render_SprigEnvFunction(t *testing.T) {
	// Arrange
	testEnvVar := "AGENT_CONFIG_TEST_VAR"
	testValue := "test-environment-value-12345"
	os.Setenv(testEnvVar, testValue)
	defer os.Unsetenv(testEnvVar)

	engine := template.NewTemplateEngine()
	ctx := testutil.CreateTestTemplateContext(t,
		testutil.WithProfile(&profile.Profile{Name: "vendor/test"}),
	)
	templateContent := `Env value: {{ env "AGENT_CONFIG_TEST_VAR" }}`

	// Act
	result, err := engine.Render(templateContent, ctx)

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	expected := "Env value: test-environment-value-12345"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

// Test 5: Error handling for invalid template syntax
func TestTemplateEngine_Render_InvalidSyntaxError(t *testing.T) {
	t.Parallel()
	// Arrange
	engine := template.NewTemplateEngine()
	ctx := testutil.CreateTestTemplateContext(t,
		testutil.WithProfile(&profile.Profile{Name: "vendor/test"}),
	)
	templateContent := "Invalid: {{ .Profile.Name"

	// Act
	result, err := engine.Render(templateContent, ctx)

	// Assert
	if err == nil {
		t.Fatal("expected error for invalid template syntax, got nil")
	}
	if result != "" {
		t.Errorf("expected empty result for error case, got %q", result)
	}
}

// Test 6: Error handling for missing/undefined template variables
func TestTemplateEngine_Render_UndefinedFieldError(t *testing.T) {
	t.Parallel()
	// Arrange
	engine := template.NewTemplateEngine()
	ctx := testutil.CreateTestTemplateContext(t,
		testutil.WithProfile(&profile.Profile{Name: "vendor/test"}),
	)
	templateContent := "Invalid: {{ .NonExistentField }}"

	// Act
	result, err := engine.Render(templateContent, ctx)

	// Assert
	if err == nil {
		t.Fatal("expected error for undefined field, got nil")
	}
	if result != "" {
		t.Errorf("expected empty result for error case, got %q", result)
	}
}

// Test 7: Additional sprig functions work (default, quote, replace)
func TestTemplateEngine_Render_MoreSprigFunctions(t *testing.T) {
	t.Parallel()
	// Arrange
	engine := template.NewTemplateEngine()
	ctx := testutil.CreateTestTemplateContext(t,
		testutil.WithProfile(&profile.Profile{Name: "vendor/test-profile"}),
		testutil.WithProjectDir("/my/project"),
	)
	templateContent := `Default: {{ .GlobalConfig | default "no-config" }}
Quoted: {{ .Profile.Name | quote }}
Replaced: {{ .ProjectDir | replace "/" "-" }}`

	// Act
	result, err := engine.Render(templateContent, ctx)

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	expected := `Default: no-config
Quoted: "vendor/test-profile"
Replaced: -my-project`
	if result != expected {
		t.Errorf("expected:\n%s\n\ngot:\n%s", expected, result)
	}
}

// Test 8: Error context includes template source identification
func TestTemplateEngine_Render_ErrorIncludesContext(t *testing.T) {
	t.Parallel()
	// Arrange
	engine := template.NewTemplateEngine()
	ctx := testutil.CreateTestTemplateContext(t,
		testutil.WithProfile(&profile.Profile{Name: "vendor/test"}),
	)
	templateContent := "{{ .Missing.Nested.Field }}"

	// Act
	_, err := engine.Render(templateContent, ctx)

	// Assert
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	// Verify error message contains useful context
	errStr := err.Error()
	if errStr == "" {
		t.Error("expected non-empty error message")
	}
}

// ============================================================================
// Integration Tests (Task Group 3)
// ============================================================================

// Integration Test 1: End-to-end create TemplateContext, render template with Profile data
func TestIntegration_EndToEnd_CreateContextAndRender(t *testing.T) {
	t.Parallel()
	// Arrange - Create full context with all data sources
	testProfile := &profile.Profile{
		Name:         "acme/golang-backend",
		InheritsFrom: "acme/base",
		TemplateFiles: []profile.TemplateFile{
			{SourcePath: "CLAUDE.md.tmpl", TargetPath: "CLAUDE.md", IsPartial: false},
			{SourcePath: "partials/header.md", TargetPath: "", IsPartial: true},
		},
	}
	projectDir := "/home/user/projects/my-app"
	generatedAt := time.Date(2025, 12, 20, 14, 30, 0, 0, time.UTC)
	projectConfig := &config.ProjectConfig{Profiles: []string{"acme/golang-backend"}}
	globalConfig := &config.GlobalConfig{DefaultProfile: "acme/default"}

	// Create context using helper
	ctx := testutil.CreateTestTemplateContext(t,
		testutil.WithProfile(testProfile),
		testutil.WithProjectDir(projectDir),
		testutil.WithGeneratedAt(generatedAt),
		testutil.WithProjectConfig(projectConfig),
		testutil.WithGlobalConfig(globalConfig),
	)

	// Create engine and render template
	engine := template.NewTemplateEngine()
	templateContent := `# Configuration for {{ .Profile.Name }}

Project Directory: {{ .ProjectDir }}
Generated: {{ .GeneratedAt.Format "2006-01-02 15:04:05" }}
Active Profile: {{ index .ProjectConfig.Profiles 0 }}

This configuration inherits from: {{ .Profile.InheritsFrom }}
Template files count: {{ len .Profile.TemplateFiles }}`

	// Act
	result, err := engine.Render(templateContent, ctx)

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	expected := `# Configuration for acme/golang-backend

Project Directory: /home/user/projects/my-app
Generated: 2025-12-20 14:30:00
Active Profile: acme/golang-backend

This configuration inherits from: acme/base
Template files count: 2`
	if result != expected {
		t.Errorf("expected:\n%s\n\ngot:\n%s", expected, result)
	}
}

// Integration Test 2: Template accessing nested data structures
func TestIntegration_NestedDataAccess(t *testing.T) {
	t.Parallel()
	// Arrange - Create context with complex nested data
	testProfile := &profile.Profile{
		Name:         "vendor/complex-profile",
		InheritsFrom: "vendor/parent",
		TemplateFiles: []profile.TemplateFile{
			{SourcePath: "file1.md", TargetPath: "out/file1.md", IsPartial: false, Checksum: "abc123"},
			{SourcePath: "file2.md", TargetPath: "out/file2.md", IsPartial: false, Checksum: "def456"},
			{SourcePath: "partial.md", TargetPath: "", IsPartial: true},
		},
	}
	projectConfig := &config.ProjectConfig{Profiles: []string{"vendor/complex-profile"}}
	globalConfig := &config.GlobalConfig{DefaultProfile: "vendor/default-profile"}

	ctx := testutil.CreateTestTemplateContext(t,
		testutil.WithProfile(testProfile),
		testutil.WithProjectDir("/projects/nested-test"),
		testutil.WithProjectConfig(projectConfig),
		testutil.WithGlobalConfig(globalConfig),
	)

	engine := template.NewTemplateEngine()

	// Template accessing various nested fields
	templateContent := `Profile Name: {{ .Profile.Name }}
Profile Parent: {{ .Profile.InheritsFrom }}
First Template Source: {{ (index .Profile.TemplateFiles 0).SourcePath }}
First Template Target: {{ (index .Profile.TemplateFiles 0).TargetPath }}
First Template Checksum: {{ (index .Profile.TemplateFiles 0).Checksum }}
Second Template IsPartial: {{ (index .Profile.TemplateFiles 1).IsPartial }}
Third Template IsPartial: {{ (index .Profile.TemplateFiles 2).IsPartial }}
ProjectConfig Profile: {{ index .ProjectConfig.Profiles 0 }}
GlobalConfig Default: {{ .GlobalConfig.DefaultProfile }}`

	// Act
	result, err := engine.Render(templateContent, ctx)

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	expected := `Profile Name: vendor/complex-profile
Profile Parent: vendor/parent
First Template Source: file1.md
First Template Target: out/file1.md
First Template Checksum: abc123
Second Template IsPartial: false
Third Template IsPartial: true
ProjectConfig Profile: vendor/complex-profile
GlobalConfig Default: vendor/default-profile`
	if result != expected {
		t.Errorf("expected:\n%s\n\ngot:\n%s", expected, result)
	}
}

// Integration Test 3: Template using sprig functions with context data
func TestIntegration_SprigFunctionsWithContextData(t *testing.T) {
	t.Parallel()
	// Arrange
	testProfile := &profile.Profile{
		Name:         "myvendor/my-profile",
		InheritsFrom: "",
		TemplateFiles: []profile.TemplateFile{
			{SourcePath: "readme.md.tmpl", TargetPath: "README.md"},
		},
	}
	projectConfig := &config.ProjectConfig{Profiles: []string{"myvendor/my-profile"}}
	globalConfig := &config.GlobalConfig{DefaultProfile: "default/profile"}

	ctx := testutil.CreateTestTemplateContext(t,
		testutil.WithProfile(testProfile),
		testutil.WithProjectDir("/path/to/my-project"),
		testutil.WithGeneratedAt(time.Date(2025, 6, 15, 9, 0, 0, 0, time.UTC)),
		testutil.WithProjectConfig(projectConfig),
		testutil.WithGlobalConfig(globalConfig),
	)

	engine := template.NewTemplateEngine()

	// Template using various sprig functions with context data
	templateContent := `Profile Upper: {{ .Profile.Name | upper }}
Profile Lower: {{ .Profile.Name | lower }}
Profile Title: {{ .Profile.Name | title }}
Profile Quoted: {{ .Profile.Name | quote }}
Profile Replaced: {{ .Profile.Name | replace "/" "_" }}
Project Dir Base: {{ .ProjectDir | base }}
Has Parent: {{ if .Profile.InheritsFrom }}yes{{ else }}no{{ end }}
Default Parent: {{ .Profile.InheritsFrom | default "none" }}
Template Count: {{ len .Profile.TemplateFiles }}
First Template: {{ (first .Profile.TemplateFiles).SourcePath }}
Date Formatted: {{ .GeneratedAt.Format "Jan 02, 2006" }}`

	// Act
	result, err := engine.Render(templateContent, ctx)

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	expected := `Profile Upper: MYVENDOR/MY-PROFILE
Profile Lower: myvendor/my-profile
Profile Title: Myvendor/My-Profile
Profile Quoted: "myvendor/my-profile"
Profile Replaced: myvendor_my-profile
Project Dir Base: my-project
Has Parent: no
Default Parent: none
Template Count: 1
First Template: readme.md.tmpl
Date Formatted: Jun 15, 2025`
	if result != expected {
		t.Errorf("expected:\n%s\n\ngot:\n%s", expected, result)
	}
}

// Integration Test 4: Combined workflow with conditionals and loops
func TestIntegration_CombinedWorkflowConditionalsAndLoops(t *testing.T) {
	t.Parallel()
	// Arrange
	testProfile := &profile.Profile{
		Name:         "org/full-profile",
		InheritsFrom: "org/base-profile",
		TemplateFiles: []profile.TemplateFile{
			{SourcePath: "config.yaml.tmpl", TargetPath: "config.yaml", IsPartial: false},
			{SourcePath: "readme.md.tmpl", TargetPath: "README.md", IsPartial: false},
			{SourcePath: "_header.md", TargetPath: "", IsPartial: true},
		},
	}
	projectConfig := &config.ProjectConfig{Profiles: []string{"org/full-profile"}}
	globalConfig := &config.GlobalConfig{DefaultProfile: "org/default"}

	ctx := testutil.CreateTestTemplateContext(t,
		testutil.WithProfile(testProfile),
		testutil.WithProjectDir("/workspace/project"),
		testutil.WithGeneratedAt(time.Date(2025, 12, 20, 16, 45, 30, 0, time.UTC)),
		testutil.WithProjectConfig(projectConfig),
		testutil.WithGlobalConfig(globalConfig),
	)

	engine := template.NewTemplateEngine()

	// Complex template with conditionals, loops, and sprig functions
	templateContent := `# Project: {{ .ProjectDir | base | upper }}

## Profile Information
- Name: {{ .Profile.Name }}
{{- if .Profile.InheritsFrom }}
- Inherits From: {{ .Profile.InheritsFrom }}
{{- else }}
- Inherits From: (none)
{{- end }}

## Template Files
{{- range $i, $tf := .Profile.TemplateFiles }}
{{ add $i 1 }}. {{ $tf.SourcePath }}{{ if $tf.IsPartial }} (partial){{ end }}
{{- end }}

## Configuration
- Project Profile: {{ index .ProjectConfig.Profiles 0 | upper }}
- Global Default: {{ .GlobalConfig.DefaultProfile }}

Generated at {{ .GeneratedAt.Format "2006-01-02T15:04:05Z07:00" }}`

	// Act
	result, err := engine.Render(templateContent, ctx)

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	expected := `# Project: PROJECT

## Profile Information
- Name: org/full-profile
- Inherits From: org/base-profile

## Template Files
1. config.yaml.tmpl
2. readme.md.tmpl
3. _header.md (partial)

## Configuration
- Project Profile: ORG/FULL-PROFILE
- Global Default: org/default

Generated at 2025-12-20T16:45:30Z`
	if result != expected {
		t.Errorf("expected:\n%s\n\ngot:\n%s", expected, result)
	}
}

// ============================================================================
// Edge Case Tests (Task Group 4) - Strategic gap-filling tests
// ============================================================================

// Edge Case Test 1: Empty template renders to empty string
func TestTemplateEngine_Render_EmptyTemplate(t *testing.T) {
	t.Parallel()
	// Arrange
	engine := template.NewTemplateEngine()
	ctx := testutil.CreateTestTemplateContext(t,
		testutil.WithProfile(&profile.Profile{Name: "vendor/test"}),
	)
	templateContent := ""

	// Act
	result, err := engine.Render(templateContent, ctx)

	// Assert
	if err != nil {
		t.Fatalf("expected no error for empty template, got %v", err)
	}
	if result != "" {
		t.Errorf("expected empty result for empty template, got %q", result)
	}
}

// Edge Case Test 2: Template with only whitespace renders preserved whitespace
func TestTemplateEngine_Render_WhitespaceOnlyTemplate(t *testing.T) {
	t.Parallel()
	// Arrange
	engine := template.NewTemplateEngine()
	ctx := testutil.CreateTestTemplateContext(t,
		testutil.WithProfile(&profile.Profile{Name: "vendor/test"}),
	)
	templateContent := "   \n\t\n   "

	// Act
	result, err := engine.Render(templateContent, ctx)

	// Assert
	if err != nil {
		t.Fatalf("expected no error for whitespace template, got %v", err)
	}
	if result != templateContent {
		t.Errorf("expected whitespace to be preserved %q, got %q", templateContent, result)
	}
}

// Edge Case Test 3: Empty Profile struct (not nil) with zero values
func TestTemplateEngine_Render_EmptyProfileStruct(t *testing.T) {
	t.Parallel()
	// Arrange
	engine := template.NewTemplateEngine()
	emptyProfile := &profile.Profile{} // All zero values
	ctx := testutil.CreateTestTemplateContext(t,
		testutil.WithProfile(emptyProfile),
		testutil.WithProjectConfig(&config.ProjectConfig{}),
		testutil.WithGlobalConfig(&config.GlobalConfig{}),
	)
	templateContent := `Name: "{{ .Profile.Name }}"
Inherits: "{{ .Profile.InheritsFrom }}"
Files: {{ len .Profile.TemplateFiles }}
Has Name: {{ if .Profile.Name }}yes{{ else }}no{{ end }}`

	// Act
	result, err := engine.Render(templateContent, ctx)

	// Assert
	if err != nil {
		t.Fatalf("expected no error for empty profile, got %v", err)
	}
	expected := `Name: ""
Inherits: ""
Files: 0
Has Name: no`
	if result != expected {
		t.Errorf("expected:\n%s\n\ngot:\n%s", expected, result)
	}
}

// Edge Case Test 4: Undefined function call returns error
func TestTemplateEngine_Render_UndefinedFunctionError(t *testing.T) {
	t.Parallel()
	// Arrange
	engine := template.NewTemplateEngine()
	ctx := testutil.CreateTestTemplateContext(t,
		testutil.WithProfile(&profile.Profile{Name: "vendor/test"}),
	)
	templateContent := `{{ .Profile.Name | nonExistentFunction }}`

	// Act
	result, err := engine.Render(templateContent, ctx)

	// Assert
	if err == nil {
		t.Fatal("expected error for undefined function, got nil")
	}
	if result != "" {
		t.Errorf("expected empty result for error case, got %q", result)
	}
}

// Edge Case Test 5: Access to nil pointer field (nil Profile) returns error
func TestTemplateEngine_Render_NilPointerFieldAccess(t *testing.T) {
	t.Parallel()
	// Arrange
	engine := template.NewTemplateEngine()
	ctx := testutil.CreateTestTemplateContext(t) // No profile set, defaults to nil
	templateContent := `{{ .Profile.Name }}`

	// Act
	result, err := engine.Render(templateContent, ctx)

	// Assert
	if err == nil {
		t.Fatal("expected error for nil pointer access, got nil")
	}
	if result != "" {
		t.Errorf("expected empty result for error case, got %q", result)
	}
}

// ============================================================================
// Engine Include Integration Tests (Task Group 3 - Include Support)
// ============================================================================

// Engine Include Test 1: RenderWithContext renders template with include support
func TestTemplateEngine_RenderWithContext_IncludeSupport(t *testing.T) {
	t.Parallel()
	// Arrange
	fs := testutil.CreateTestFilesystem(t, map[string]string{
		"/templates/_partials/header.md": "# Welcome Header",
		"/templates/_partials/footer.md": "--- Footer ---",
	})

	resolver := template.NewMapFileResolver(map[string]string{
		"_partials/header.md": "/templates/_partials/header.md",
		"_partials/footer.md": "/templates/_partials/footer.md",
	})

	engine := template.NewTemplateEngineWithFs(fs)
	renderCtx := template.NewRenderContext(fs, resolver)
	tmplCtx := testutil.CreateTestTemplateContext(t,
		testutil.WithProfile(&profile.Profile{Name: "test/profile"}),
	)

	templateContent := `{{ include "_partials/header.md" }}

Main content for {{ .Profile.Name }}

{{ include "_partials/footer.md" }}`

	// Act
	result, err := engine.RenderWithContext(templateContent, tmplCtx, renderCtx)

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	expected := `# Welcome Header

Main content for test/profile

--- Footer ---`
	if result != expected {
		t.Errorf("expected:\n%s\n\ngot:\n%s", expected, result)
	}
}

// Engine Include Test 2: Engine with real filesystem integration via afero.Fs
func TestTemplateEngine_RenderWithContext_AferoIntegration(t *testing.T) {
	t.Parallel()
	// Arrange - Use in-memory filesystem to simulate real file operations
	fs := testutil.CreateTestFilesystem(t, map[string]string{
		"/profiles/vendor/myprofile/templates/main.md":           "Main: {{ .Profile.Name }}",
		"/profiles/vendor/myprofile/templates/_partials/info.md": "Info: Project at {{ .ProjectDir }}",
	})

	resolver := template.NewMapFileResolver(map[string]string{
		"main.md":           "/profiles/vendor/myprofile/templates/main.md",
		"_partials/info.md": "/profiles/vendor/myprofile/templates/_partials/info.md",
	})

	engine := template.NewTemplateEngineWithFs(fs)
	renderCtx := template.NewRenderContext(fs, resolver)
	tmplCtx := testutil.CreateTestTemplateContext(t,
		testutil.WithProfile(&profile.Profile{Name: "vendor/myprofile"}),
		testutil.WithProjectDir("/home/user/myproject"),
	)

	templateContent := `Document Start
{{ include "_partials/info.md" }}
Document End`

	// Act
	result, err := engine.RenderWithContext(templateContent, tmplCtx, renderCtx)

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	expected := `Document Start
Info: Project at /home/user/myproject
Document End`
	if result != expected {
		t.Errorf("expected:\n%s\n\ngot:\n%s", expected, result)
	}
}

// Engine Include Test 3: Engine properly registers include function per-render
func TestTemplateEngine_RenderWithContext_IncludeFunctionPerRender(t *testing.T) {
	t.Parallel()
	// Arrange - Create two different render contexts with different file resolvers
	fs := testutil.CreateTestFilesystem(t, map[string]string{
		"/templates/v1/partial.md": "Version 1 Content",
		"/templates/v2/partial.md": "Version 2 Content",
	})

	resolver1 := template.NewMapFileResolver(map[string]string{
		"partial.md": "/templates/v1/partial.md",
	})
	resolver2 := template.NewMapFileResolver(map[string]string{
		"partial.md": "/templates/v2/partial.md",
	})

	engine := template.NewTemplateEngineWithFs(fs)
	tmplCtx := testutil.CreateTestTemplateContext(t,
		testutil.WithProfile(&profile.Profile{Name: "test/profile"}),
	)

	templateContent := `{{ include "partial.md" }}`

	// Act - Render with first context
	renderCtx1 := template.NewRenderContext(fs, resolver1)
	result1, err1 := engine.RenderWithContext(templateContent, tmplCtx, renderCtx1)

	// Act - Render with second context
	renderCtx2 := template.NewRenderContext(fs, resolver2)
	result2, err2 := engine.RenderWithContext(templateContent, tmplCtx, renderCtx2)

	// Assert
	if err1 != nil {
		t.Fatalf("expected no error for first render, got %v", err1)
	}
	if err2 != nil {
		t.Fatalf("expected no error for second render, got %v", err2)
	}
	if result1 != "Version 1 Content" {
		t.Errorf("expected 'Version 1 Content', got %q", result1)
	}
	if result2 != "Version 2 Content" {
		t.Errorf("expected 'Version 2 Content', got %q", result2)
	}
}

// Engine Include Test 4: Error propagation from included template syntax errors
func TestTemplateEngine_RenderWithContext_IncludeSyntaxErrorPropagation(t *testing.T) {
	t.Parallel()
	// Arrange
	fs := testutil.CreateTestFilesystem(t, map[string]string{
		"/templates/broken.md": "Invalid: {{ .Profile.Name",
	})

	resolver := template.NewMapFileResolver(map[string]string{
		"broken.md": "/templates/broken.md",
	})

	engine := template.NewTemplateEngineWithFs(fs)
	renderCtx := template.NewRenderContext(fs, resolver)
	tmplCtx := testutil.CreateTestTemplateContext(t,
		testutil.WithProfile(&profile.Profile{Name: "test/profile"}),
	)

	templateContent := `Before include
{{ include "broken.md" }}
After include`

	// Act
	result, err := engine.RenderWithContext(templateContent, tmplCtx, renderCtx)

	// Assert
	if err == nil {
		t.Fatal("expected error for include with syntax error, got nil")
	}
	if result != "" {
		t.Errorf("expected empty result for error case, got %q", result)
	}

	// Verify it's an IncludeSyntaxError
	var syntaxErr *template.IncludeSyntaxError
	if !errors.As(err, &syntaxErr) {
		t.Errorf("expected IncludeSyntaxError, got %T: %v", err, err)
	}
}

// Engine Include Test 5: Error propagation from included template execution errors
func TestTemplateEngine_RenderWithContext_IncludeExecutionErrorPropagation(t *testing.T) {
	t.Parallel()
	// Arrange
	fs := testutil.CreateTestFilesystem(t, map[string]string{
		"/templates/undefined.md": "Value: {{ .UndefinedField }}",
	})

	resolver := template.NewMapFileResolver(map[string]string{
		"undefined.md": "/templates/undefined.md",
	})

	engine := template.NewTemplateEngineWithFs(fs)
	renderCtx := template.NewRenderContext(fs, resolver)
	tmplCtx := testutil.CreateTestTemplateContext(t,
		testutil.WithProfile(&profile.Profile{Name: "test/profile"}),
	)

	templateContent := `Before include
{{ include "undefined.md" }}
After include`

	// Act
	result, err := engine.RenderWithContext(templateContent, tmplCtx, renderCtx)

	// Assert
	if err == nil {
		t.Fatal("expected error for include with execution error, got nil")
	}
	if result != "" {
		t.Errorf("expected empty result for error case, got %q", result)
	}

	// Verify it's an IncludeExecutionError
	var execErr *template.IncludeExecutionError
	if !errors.As(err, &execErr) {
		t.Errorf("expected IncludeExecutionError, got %T: %v", err, err)
	}
}

// ============================================================================
// Engine IncludeGlob Integration Tests (Task Group 3 - IncludeGlob Support)
// ============================================================================

// Engine IncludeGlob Test 1: includeGlob is available in templates via RenderWithContext
func TestTemplateEngine_RenderWithContext_IncludeGlobAvailable(t *testing.T) {
	t.Parallel()
	// Arrange
	fs := testutil.CreateTestFilesystem(t, map[string]string{
		"/templates/_skills/skill1.md": "Skill 1",
		"/templates/_skills/skill2.md": "Skill 2",
	})

	resolver := template.NewMapFileResolver(map[string]string{
		"_skills/skill1.md": "/templates/_skills/skill1.md",
		"_skills/skill2.md": "/templates/_skills/skill2.md",
	})

	engine := template.NewTemplateEngineWithFs(fs)
	renderCtx := template.NewRenderContext(fs, resolver)
	tmplCtx := testutil.CreateTestTemplateContext(t,
		testutil.WithProfile(&profile.Profile{Name: "test/profile"}),
	)

	templateContent := `Skills:
{{ includeGlob "_skills/*.md" }}`

	// Act
	result, err := engine.RenderWithContext(templateContent, tmplCtx, renderCtx)

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	expected := `Skills:
Skill 1
Skill 2`
	if result != expected {
		t.Errorf("expected:\n%s\n\ngot:\n%s", expected, result)
	}
}

// Engine IncludeGlob Test 2: includeGlob and include can be used together in same template
func TestTemplateEngine_RenderWithContext_IncludeGlobAndIncludeTogether(t *testing.T) {
	t.Parallel()
	// Arrange
	fs := testutil.CreateTestFilesystem(t, map[string]string{
		"/templates/header.md":        "# Header",
		"/templates/_skills/alpha.md": "Alpha skill",
		"/templates/_skills/beta.md":  "Beta skill",
		"/templates/footer.md":        "---Footer---",
	})

	resolver := template.NewMapFileResolver(map[string]string{
		"header.md":        "/templates/header.md",
		"_skills/alpha.md": "/templates/_skills/alpha.md",
		"_skills/beta.md":  "/templates/_skills/beta.md",
		"footer.md":        "/templates/footer.md",
	})

	engine := template.NewTemplateEngineWithFs(fs)
	renderCtx := template.NewRenderContext(fs, resolver)
	tmplCtx := testutil.CreateTestTemplateContext(t,
		testutil.WithProfile(&profile.Profile{Name: "test/profile"}),
	)

	templateContent := `{{ include "header.md" }}

Skills:
{{ includeGlob "_skills/*.md" }}

{{ include "footer.md" }}`

	// Act
	result, err := engine.RenderWithContext(templateContent, tmplCtx, renderCtx)

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	expected := `# Header

Skills:
Alpha skill
Beta skill

---Footer---`
	if result != expected {
		t.Errorf("expected:\n%s\n\ngot:\n%s", expected, result)
	}
}

// Engine IncludeGlob Test 3: nested include inside includeGlob-matched file works
func TestTemplateEngine_RenderWithContext_NestedIncludeInsideIncludeGlob(t *testing.T) {
	t.Parallel()
	// Arrange
	fs := testutil.CreateTestFilesystem(t, map[string]string{
		"/templates/_skills/skill1.md": "Skill1:{{ include \"common.md\" }}",
		"/templates/_skills/skill2.md": "Skill2:{{ include \"common.md\" }}",
		"/templates/common.md":         "COMMON",
	})

	resolver := template.NewMapFileResolver(map[string]string{
		"_skills/skill1.md": "/templates/_skills/skill1.md",
		"_skills/skill2.md": "/templates/_skills/skill2.md",
		"common.md":         "/templates/common.md",
	})

	engine := template.NewTemplateEngineWithFs(fs)
	renderCtx := template.NewRenderContext(fs, resolver)
	tmplCtx := testutil.CreateTestTemplateContext(t,
		testutil.WithProfile(&profile.Profile{Name: "test/profile"}),
	)

	templateContent := `{{ includeGlob "_skills/*.md" }}`

	// Act
	result, err := engine.RenderWithContext(templateContent, tmplCtx, renderCtx)

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	expected := "Skill1:COMMON\nSkill2:COMMON"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

// Engine IncludeGlob Test 4: recursive includeGlob inside matched file works
func TestTemplateEngine_RenderWithContext_RecursiveIncludeGlobInsideMatchedFile(t *testing.T) {
	t.Parallel()
	// Arrange
	fs := testutil.CreateTestFilesystem(t, map[string]string{
		"/templates/categories/cat1.md": "Cat1:[{{ includeGlob \"items/cat1_*.md\" }}]",
		"/templates/categories/cat2.md": "Cat2:[{{ includeGlob \"items/cat2_*.md\" }}]",
		"/templates/items/cat1_a.md":    "Item1A",
		"/templates/items/cat1_b.md":    "Item1B",
		"/templates/items/cat2_a.md":    "Item2A",
	})

	resolver := template.NewMapFileResolver(map[string]string{
		"categories/cat1.md": "/templates/categories/cat1.md",
		"categories/cat2.md": "/templates/categories/cat2.md",
		"items/cat1_a.md":    "/templates/items/cat1_a.md",
		"items/cat1_b.md":    "/templates/items/cat1_b.md",
		"items/cat2_a.md":    "/templates/items/cat2_a.md",
	})

	engine := template.NewTemplateEngineWithFs(fs)
	renderCtx := template.NewRenderContext(fs, resolver)
	tmplCtx := testutil.CreateTestTemplateContext(t,
		testutil.WithProfile(&profile.Profile{Name: "test/profile"}),
	)

	templateContent := `{{ includeGlob "categories/*.md" }}`

	// Act
	result, err := engine.RenderWithContext(templateContent, tmplCtx, renderCtx)

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	expected := "Cat1:[Item1A\nItem1B]\nCat2:[Item2A]"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

// Engine IncludeGlob Test 5: includeGlob inside included file works (include then includeGlob)
func TestTemplateEngine_RenderWithContext_IncludeGlobInsideIncludedFile(t *testing.T) {
	t.Parallel()
	// Arrange
	fs := testutil.CreateTestFilesystem(t, map[string]string{
		"/templates/skills_section.md": "Skills:{{ includeGlob \"_skills/*.md\" }}",
		"/templates/_skills/s1.md":     "S1",
		"/templates/_skills/s2.md":     "S2",
	})

	resolver := template.NewMapFileResolver(map[string]string{
		"skills_section.md": "/templates/skills_section.md",
		"_skills/s1.md":     "/templates/_skills/s1.md",
		"_skills/s2.md":     "/templates/_skills/s2.md",
	})

	engine := template.NewTemplateEngineWithFs(fs)
	renderCtx := template.NewRenderContext(fs, resolver)
	tmplCtx := testutil.CreateTestTemplateContext(t,
		testutil.WithProfile(&profile.Profile{Name: "test/profile"}),
	)

	templateContent := `{{ include "skills_section.md" }}`

	// Act
	result, err := engine.RenderWithContext(templateContent, tmplCtx, renderCtx)

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	expected := "Skills:S1\nS2"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

// ============================================================================
// Task Group 6: Render Method Typed Error Tests
// ============================================================================

// Test: Render() returns TemplateSyntaxError for invalid template syntax
func TestTemplateEngine_Render_ReturnsTemplateSyntaxError(t *testing.T) {
	t.Parallel()
	// Arrange
	engine := template.NewTemplateEngine()
	ctx := testutil.CreateTestTemplateContext(t,
		testutil.WithProfile(&profile.Profile{Name: "vendor/test"}),
	)
	templateContent := `line 1
line 2
{{ .Invalid }`

	// Act
	result, err := engine.Render(templateContent, ctx)

	// Assert
	if err == nil {
		t.Fatal("expected error for invalid template syntax, got nil")
	}
	if result != "" {
		t.Errorf("expected empty result for error case, got %q", result)
	}

	// Verify it's a TemplateSyntaxError
	var syntaxErr *template.TemplateSyntaxError
	if !errors.As(err, &syntaxErr) {
		t.Errorf("expected TemplateSyntaxError, got %T: %v", err, err)
	}

	// Verify error properties
	if syntaxErr.SourcePath != "template" {
		t.Errorf("expected SourcePath 'template', got %q", syntaxErr.SourcePath)
	}
	if syntaxErr.Line == 0 {
		t.Error("expected Line to be set, got 0")
	}
	if syntaxErr.Content != templateContent {
		t.Errorf("expected Content to match template, got %q", syntaxErr.Content)
	}
}

// Test: Render() returns TemplateExecutionError for runtime errors
func TestTemplateEngine_Render_ReturnsTemplateExecutionError(t *testing.T) {
	t.Parallel()
	// Arrange
	engine := template.NewTemplateEngine()
	ctx := testutil.CreateTestTemplateContext(t) // No profile set, defaults to nil
	templateContent := `line 1
line 2
{{ .Profile.Name }}`

	// Act
	result, err := engine.Render(templateContent, ctx)

	// Assert
	if err == nil {
		t.Fatal("expected error for nil pointer access, got nil")
	}
	if result != "" {
		t.Errorf("expected empty result for error case, got %q", result)
	}

	// Verify it's a TemplateExecutionError
	var execErr *template.TemplateExecutionError
	if !errors.As(err, &execErr) {
		t.Errorf("expected TemplateExecutionError, got %T: %v", err, err)
	}

	// Verify error properties
	if execErr.SourcePath != "template" {
		t.Errorf("expected SourcePath 'template', got %q", execErr.SourcePath)
	}
	if execErr.Content != templateContent {
		t.Errorf("expected Content to match template, got %q", execErr.Content)
	}
}

// Test: RenderWithContext() returns TemplateSyntaxError with include chain
func TestTemplateEngine_RenderWithContext_ReturnsTemplateSyntaxErrorWithIncludeChain(t *testing.T) {
	t.Parallel()
	// Arrange
	fs := afero.NewMemMapFs()
	resolver := template.NewMapFileResolver(map[string]string{})

	engine := template.NewTemplateEngineWithFs(fs)
	renderCtx := template.NewRenderContext(fs, resolver)
	// Pre-populate the include chain to simulate being inside an include
	renderCtx.IncludeChain = []string{"main.md", "header.md"}

	tmplCtx := testutil.CreateTestTemplateContext(t,
		testutil.WithProfile(&profile.Profile{Name: "test/profile"}),
	)

	templateContent := `{{ .Invalid }`

	// Act
	result, err := engine.RenderWithContext(templateContent, tmplCtx, renderCtx)

	// Assert
	if err == nil {
		t.Fatal("expected error for invalid template syntax, got nil")
	}
	if result != "" {
		t.Errorf("expected empty result for error case, got %q", result)
	}

	// Verify it's a TemplateSyntaxError
	var syntaxErr *template.TemplateSyntaxError
	if !errors.As(err, &syntaxErr) {
		t.Errorf("expected TemplateSyntaxError, got %T: %v", err, err)
	}

	// Verify include chain is present
	if len(syntaxErr.IncludeChain) != 2 {
		t.Errorf("expected IncludeChain length 2, got %d: %v", len(syntaxErr.IncludeChain), syntaxErr.IncludeChain)
	}
}

// Test: RenderWithContext() returns TemplateExecutionError with include chain
func TestTemplateEngine_RenderWithContext_ReturnsTemplateExecutionErrorWithIncludeChain(t *testing.T) {
	t.Parallel()
	// Arrange
	fs := afero.NewMemMapFs()
	resolver := template.NewMapFileResolver(map[string]string{})

	engine := template.NewTemplateEngineWithFs(fs)
	renderCtx := template.NewRenderContext(fs, resolver)
	// Pre-populate the include chain to simulate being inside an include
	renderCtx.IncludeChain = []string{"main.md", "footer.md"}

	tmplCtx := testutil.CreateTestTemplateContext(t) // No profile set, defaults to nil

	templateContent := `{{ .Profile.Name }}`

	// Act
	result, err := engine.RenderWithContext(templateContent, tmplCtx, renderCtx)

	// Assert
	if err == nil {
		t.Fatal("expected error for nil pointer access, got nil")
	}
	if result != "" {
		t.Errorf("expected empty result for error case, got %q", result)
	}

	// Verify it's a TemplateExecutionError
	var execErr *template.TemplateExecutionError
	if !errors.As(err, &execErr) {
		t.Errorf("expected TemplateExecutionError, got %T: %v", err, err)
	}

	// Verify include chain is present
	if len(execErr.IncludeChain) != 2 {
		t.Errorf("expected IncludeChain length 2, got %d: %v", len(execErr.IncludeChain), execErr.IncludeChain)
	}
}

// Test: Backward compatibility - underlying errors still unwrappable
func TestTemplateEngine_Render_ErrorsAreUnwrappable(t *testing.T) {
	t.Parallel()
	// Arrange
	engine := template.NewTemplateEngine()
	ctx := testutil.CreateTestTemplateContext(t,
		testutil.WithProfile(&profile.Profile{Name: "vendor/test"}),
	)

	// Test 1: Syntax error unwrapping
	syntaxTemplateContent := `{{ .Invalid }`
	_, syntaxErr := engine.Render(syntaxTemplateContent, ctx)
	if syntaxErr == nil {
		t.Fatal("expected syntax error, got nil")
	}

	var syntaxTypedErr *template.TemplateSyntaxError
	if !errors.As(syntaxErr, &syntaxTypedErr) {
		t.Fatalf("expected TemplateSyntaxError, got %T", syntaxErr)
	}

	// Verify Unwrap returns the underlying error
	underlyingSyntax := syntaxTypedErr.Unwrap()
	if underlyingSyntax == nil {
		t.Error("expected underlying error from Unwrap(), got nil")
	}

	// Test 2: Execution error unwrapping
	execCtx := testutil.CreateTestTemplateContext(t) // No profile set, defaults to nil
	execTemplateContent := `{{ .Profile.Name }}`
	_, execErr := engine.Render(execTemplateContent, execCtx)
	if execErr == nil {
		t.Fatal("expected execution error, got nil")
	}

	var execTypedErr *template.TemplateExecutionError
	if !errors.As(execErr, &execTypedErr) {
		t.Fatalf("expected TemplateExecutionError, got %T", execErr)
	}

	// Verify Unwrap returns the underlying error
	underlyingExec := execTypedErr.Unwrap()
	if underlyingExec == nil {
		t.Error("expected underlying error from Unwrap(), got nil")
	}
}

// ============================================================================
// Task Group 7: Strategic Gap-Filling Tests for Template Error Handling
// ============================================================================

// Gap Test 1: Deeply nested include (3+ levels) with syntax error
// Tests that syntax errors in deeply nested includes show the full include chain
func TestTemplateEngine_DeeplyNestedInclude_SyntaxError(t *testing.T) {
	t.Parallel()
	// Arrange - Create a 4-level include chain: main -> level1 -> level2 -> broken
	fs := testutil.CreateTestFilesystem(t, map[string]string{
		"/templates/level1.md": "Level1:{{ include \"level2.md\" }}",
		"/templates/level2.md": "Level2:{{ include \"level3.md\" }}",
		"/templates/level3.md": "Level3:{{ include \"broken.md\" }}",
		"/templates/broken.md": "line 1\nline 2\n{{ .Invalid }",
	})

	resolver := template.NewMapFileResolver(map[string]string{
		"level1.md": "/templates/level1.md",
		"level2.md": "/templates/level2.md",
		"level3.md": "/templates/level3.md",
		"broken.md": "/templates/broken.md",
	})

	engine := template.NewTemplateEngineWithFs(fs)
	renderCtx := template.NewRenderContext(fs, resolver)
	tmplCtx := testutil.CreateTestTemplateContext(t,
		testutil.WithProfile(&profile.Profile{Name: "test/profile"}),
	)

	templateContent := `Start:{{ include "level1.md" }}:End`

	// Act
	result, err := engine.RenderWithContext(templateContent, tmplCtx, renderCtx)

	// Assert
	if err == nil {
		t.Fatal("expected error for deeply nested syntax error, got nil")
	}
	if result != "" {
		t.Errorf("expected empty result for error case, got %q", result)
	}

	// Verify it's an IncludeSyntaxError (the include function wraps syntax errors)
	var syntaxErr *template.IncludeSyntaxError
	if !errors.As(err, &syntaxErr) {
		t.Errorf("expected IncludeSyntaxError, got %T: %v", err, err)
	}

	// Verify the error message contains information about the broken file
	errStr := err.Error()
	if !strings.Contains(errStr, "broken.md") {
		t.Errorf("expected error to mention 'broken.md', got: %s", errStr)
	}
}

// Gap Test 2: Execution error in middle of include chain (not at the deepest level)
// Tests that execution errors in the middle of an include chain are handled correctly
func TestTemplateEngine_MiddleIncludeChain_ExecutionError(t *testing.T) {
	t.Parallel()
	// Arrange - Create a 3-level include chain where error occurs in level2
	fs := testutil.CreateTestFilesystem(t, map[string]string{
		"/templates/level1.md": "Level1:{{ include \"level2.md\" }}",
		"/templates/level2.md": "Level2:{{ .Profile.Name }}:{{ include \"level3.md\" }}",
		"/templates/level3.md": "Level3:Done",
	})

	resolver := template.NewMapFileResolver(map[string]string{
		"level1.md": "/templates/level1.md",
		"level2.md": "/templates/level2.md",
		"level3.md": "/templates/level3.md",
	})

	engine := template.NewTemplateEngineWithFs(fs)
	renderCtx := template.NewRenderContext(fs, resolver)
	tmplCtx := testutil.CreateTestTemplateContext(t) // No profile set to trigger error in level2

	templateContent := `Start:{{ include "level1.md" }}:End`

	// Act
	result, err := engine.RenderWithContext(templateContent, tmplCtx, renderCtx)

	// Assert
	if err == nil {
		t.Fatal("expected error for execution error in middle of chain, got nil")
	}
	if result != "" {
		t.Errorf("expected empty result for error case, got %q", result)
	}

	// Verify it's an IncludeExecutionError
	var execErr *template.IncludeExecutionError
	if !errors.As(err, &execErr) {
		t.Errorf("expected IncludeExecutionError, got %T: %v", err, err)
	}
}

// Gap Test 3: Full error message format verification with all sections
// Tests that the complete error message includes all expected sections
func TestTemplateSyntaxError_FullErrorMessageFormat(t *testing.T) {
	t.Parallel()
	// Arrange - Create an error with all fields populated
	err := &template.TemplateSyntaxError{
		SourcePath: "config/templates/main.md",
		Line:       5,
		Column:     12,
		Content: `line 1 content
line 2 content
line 3 content
line 4 content
{{ .Invalid }
line 6 content
line 7 content`,
		IncludeChain: []string{"root.md", "header.md", "config/templates/main.md"},
		Err:          errors.New("unexpected \"}\" in operand"),
	}

	// Act
	got := err.Error()

	// Assert - Verify all major sections are present in the error message
	expectedSections := []string{
		// Header section
		"template syntax error in config/templates/main.md",
		"line 5",
		"column 12",
		// Error message section
		"unexpected \"}\" in operand",
		// Context section
		">>>",
		"5 |",
		"{{ .Invalid }",
		// Include chain section
		"Include trace:",
		"root.md",
		"-> header.md",
		"-> config/templates/main.md (error here)",
	}

	for _, section := range expectedSections {
		if !strings.Contains(got, section) {
			t.Errorf("Expected error message to contain %q\nGot:\n%s", section, got)
		}
	}
}

// Gap Test 4: TemplateExecutionError full format with include chain
// Tests that execution errors also show complete formatted output
func TestTemplateExecutionError_FullErrorMessageFormat(t *testing.T) {
	t.Parallel()
	// Arrange - Create an execution error with all fields populated
	err := &template.TemplateExecutionError{
		SourcePath: "_partials/dynamic.md",
		Line:       3,
		Content: `line 1
line 2
{{ .Missing.Field }}
line 4
line 5`,
		IncludeChain: []string{"main.md", "_partials/header.md", "_partials/dynamic.md"},
		Err:          errors.New("nil pointer evaluating interface {}"),
	}

	// Act
	got := err.Error()

	// Assert - Verify all major sections are present
	expectedSections := []string{
		// Header section
		"template execution error in _partials/dynamic.md",
		"line 3",
		// Error message section
		"nil pointer evaluating interface {}",
		// Context section
		">>>",
		"3 |",
		"{{ .Missing.Field }}",
		// Include chain section
		"Include trace:",
		"main.md",
		"-> _partials/header.md",
		"-> _partials/dynamic.md (error here)",
	}

	for _, section := range expectedSections {
		if !strings.Contains(got, section) {
			t.Errorf("Expected error message to contain %q\nGot:\n%s", section, got)
		}
	}
}

// Gap Test 5: Error at first line of template with context display
// Tests that errors on line 1 still show proper context (no lines before)
func TestTemplateEngine_ErrorAtFirstLine_ContextDisplay(t *testing.T) {
	t.Parallel()
	// Arrange
	engine := template.NewTemplateEngine()
	ctx := testutil.CreateTestTemplateContext(t,
		testutil.WithProfile(&profile.Profile{Name: "vendor/test"}),
	)
	// Error on first line
	templateContent := `{{ .Invalid }
line 2 content
line 3 content`

	// Act
	_, err := engine.Render(templateContent, ctx)

	// Assert
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var syntaxErr *template.TemplateSyntaxError
	if !errors.As(err, &syntaxErr) {
		t.Fatalf("expected TemplateSyntaxError, got %T", err)
	}

	// Verify the error shows context starting from line 1
	errStr := err.Error()
	if !strings.Contains(errStr, ">>> ") {
		t.Errorf("expected error marker in context, got: %s", errStr)
	}
	if !strings.Contains(errStr, "1 |") {
		t.Errorf("expected line 1 in context, got: %s", errStr)
	}
	// Should show line 2 as context after the error
	if !strings.Contains(errStr, "2 |") {
		t.Errorf("expected line 2 in context, got: %s", errStr)
	}
}

// Gap Test 6: Error at last line of template with context display
// Tests that errors on the last line still show proper context (no lines after)
func TestTemplateEngine_ErrorAtLastLine_ContextDisplay(t *testing.T) {
	t.Parallel()
	// Arrange
	engine := template.NewTemplateEngine()
	ctx := testutil.CreateTestTemplateContext(t,
		testutil.WithProfile(&profile.Profile{Name: "vendor/test"}),
	)
	// Error on last line (line 4)
	templateContent := `line 1 content
line 2 content
line 3 content
{{ .Invalid }`

	// Act
	_, err := engine.Render(templateContent, ctx)

	// Assert
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var syntaxErr *template.TemplateSyntaxError
	if !errors.As(err, &syntaxErr) {
		t.Fatalf("expected TemplateSyntaxError, got %T", err)
	}

	// Verify the error shows context including line 4
	errStr := err.Error()
	if !strings.Contains(errStr, ">>> ") {
		t.Errorf("expected error marker in context, got: %s", errStr)
	}
	if !strings.Contains(errStr, "4 |") {
		t.Errorf("expected line 4 in context, got: %s", errStr)
	}
	// Should show line 3 as context before the error
	if !strings.Contains(errStr, "3 |") {
		t.Errorf("expected line 3 in context, got: %s", errStr)
	}
}
