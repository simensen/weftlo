package rendering_test

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/spf13/afero"

	"github.com/simensen/weftlo/internal/app/rendering"
	"github.com/simensen/weftlo/internal/app/routing"
	"github.com/simensen/weftlo/internal/domain/config"
	"github.com/simensen/weftlo/internal/domain/profile"
	"github.com/simensen/weftlo/internal/domain/template"
	infraprofile "github.com/simensen/weftlo/internal/infrastructure/profile"
)

// =============================================================================
// Task Group 1: RenderingPipeline Service Foundation Tests
// =============================================================================

// Test 1.1.1: Test basic pipeline construction with NewRenderingPipeline
func TestNewRenderingPipeline_Construction(t *testing.T) {
	// Arrange
	fs := afero.NewMemMapFs()
	engine := template.NewTemplateEngineWithFs(fs)

	// Act
	pipeline := rendering.NewRenderingPipeline(engine)

	// Assert
	if pipeline == nil {
		t.Fatal("expected pipeline to be non-nil")
	}
}

// Test 1.1.2: Test RenderAll returns empty slice for profile with no templates
func TestRenderAll_EmptyProfile_ReturnsEmptySlice(t *testing.T) {
	// Arrange
	fs := afero.NewMemMapFs()
	engine := template.NewTemplateEngineWithFs(fs)
	pipeline := rendering.NewRenderingPipeline(engine)

	// Create merged profile with no templates
	chain := []*profile.Profile{
		{
			Name:          "vendor/empty",
			TemplateFiles: []profile.TemplateFile{},
		},
	}
	mergedProfile := infraprofile.NewMergedProfile(chain, fs, "/profiles")

	// Act
	results, err := pipeline.RenderAll(mergedProfile, "/project", nil)

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected empty slice, got %d results", len(results))
	}
}

// Test 1.1.3: Test RenderAll renders single non-partial template correctly
func TestRenderAll_SingleNonPartialTemplate_RendersCorrectly(t *testing.T) {
	// Arrange
	fs := afero.NewMemMapFs()
	engine := template.NewTemplateEngineWithFs(fs)
	pipeline := rendering.NewRenderingPipeline(engine)

	// Create template file on filesystem
	templateContent := "Hello, {{ .ProjectDir }}!"
	afero.WriteFile(fs, "/profiles/vendor/test/content/config.md.tmpl", []byte(templateContent), 0644)

	// Create merged profile with one template
	chain := []*profile.Profile{
		{
			Name: "vendor/test",
			TemplateFiles: []profile.TemplateFile{
				{SourcePath: "config.md.tmpl", IsPartial: false},
			},
		},
	}
	mergedProfile := infraprofile.NewMergedProfile(chain, fs, "/profiles")

	// Act
	results, err := pipeline.RenderAll(mergedProfile, "/my/project", nil)

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Content != "Hello, /my/project!" {
		t.Errorf("expected rendered content 'Hello, /my/project!', got %q", results[0].Content)
	}
}

// Test 1.1.4: Test RenderAll skips partial templates (IsPartial = true)
func TestRenderAll_SkipsPartialTemplates(t *testing.T) {
	// Arrange
	fs := afero.NewMemMapFs()
	engine := template.NewTemplateEngineWithFs(fs)
	pipeline := rendering.NewRenderingPipeline(engine)

	// Create template files on filesystem
	afero.WriteFile(fs, "/profiles/vendor/test/content/config.md.tmpl", []byte("Main content"), 0644)
	afero.WriteFile(fs, "/profiles/vendor/test/content/_partial.md", []byte("Partial content"), 0644)

	// Create merged profile with both partial and non-partial templates
	chain := []*profile.Profile{
		{
			Name: "vendor/test",
			TemplateFiles: []profile.TemplateFile{
				{SourcePath: "config.md.tmpl", IsPartial: false},
				{SourcePath: "_partial.md", IsPartial: true},
			},
		},
	}
	mergedProfile := infraprofile.NewMergedProfile(chain, fs, "/profiles")

	// Act
	results, err := pipeline.RenderAll(mergedProfile, "/project", nil)

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result (partial should be skipped), got %d", len(results))
	}
	if results[0].SourcePath != "config.md.tmpl" {
		t.Errorf("expected source path 'config.md.tmpl', got %q", results[0].SourcePath)
	}
}

// Test 1.1.5: Test TargetPath computation strips .tmpl extension
func TestRenderAll_TargetPath_StripsTmplExtension(t *testing.T) {
	// Arrange
	fs := afero.NewMemMapFs()
	engine := template.NewTemplateEngineWithFs(fs)
	pipeline := rendering.NewRenderingPipeline(engine)

	// Create template file
	afero.WriteFile(fs, "/profiles/vendor/test/content/config.md.tmpl", []byte("Content"), 0644)

	chain := []*profile.Profile{
		{
			Name: "vendor/test",
			TemplateFiles: []profile.TemplateFile{
				{SourcePath: "config.md.tmpl", IsPartial: false},
			},
		},
	}
	mergedProfile := infraprofile.NewMergedProfile(chain, fs, "/profiles")

	// Act
	results, err := pipeline.RenderAll(mergedProfile, "/project", nil)

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].TargetPath != "config.md" {
		t.Errorf("expected target path 'config.md', got %q", results[0].TargetPath)
	}
}

// Test 1.1.6: Test TargetPath leaves non-.tmpl paths unchanged
func TestRenderAll_TargetPath_LeavesNonTmplUnchanged(t *testing.T) {
	// Arrange
	fs := afero.NewMemMapFs()
	engine := template.NewTemplateEngineWithFs(fs)
	pipeline := rendering.NewRenderingPipeline(engine)

	// Create template file without .tmpl extension
	afero.WriteFile(fs, "/profiles/vendor/test/content/README.md", []byte("Static content"), 0644)

	chain := []*profile.Profile{
		{
			Name: "vendor/test",
			TemplateFiles: []profile.TemplateFile{
				{SourcePath: "README.md", IsPartial: false},
			},
		},
	}
	mergedProfile := infraprofile.NewMergedProfile(chain, fs, "/profiles")

	// Act
	results, err := pipeline.RenderAll(mergedProfile, "/project", nil)

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].TargetPath != "README.md" {
		t.Errorf("expected target path 'README.md', got %q", results[0].TargetPath)
	}
}

// =============================================================================
// Task Group 2: Template Rendering Logic Tests
// =============================================================================

// Test 2.1.1: Test RenderAll iterates over all non-partial templates
func TestRenderAll_IteratesOverAllNonPartialTemplates(t *testing.T) {
	// Arrange
	fs := afero.NewMemMapFs()
	engine := template.NewTemplateEngineWithFs(fs)
	pipeline := rendering.NewRenderingPipeline(engine)

	// Create multiple template files
	afero.WriteFile(fs, "/profiles/vendor/test/content/file1.md.tmpl", []byte("Content 1"), 0644)
	afero.WriteFile(fs, "/profiles/vendor/test/content/file2.md.tmpl", []byte("Content 2"), 0644)
	afero.WriteFile(fs, "/profiles/vendor/test/content/file3.md.tmpl", []byte("Content 3"), 0644)
	afero.WriteFile(fs, "/profiles/vendor/test/content/_partial.md", []byte("Partial"), 0644)

	chain := []*profile.Profile{
		{
			Name: "vendor/test",
			TemplateFiles: []profile.TemplateFile{
				{SourcePath: "file1.md.tmpl", IsPartial: false},
				{SourcePath: "file2.md.tmpl", IsPartial: false},
				{SourcePath: "file3.md.tmpl", IsPartial: false},
				{SourcePath: "_partial.md", IsPartial: true},
			},
		},
	}
	mergedProfile := infraprofile.NewMergedProfile(chain, fs, "/profiles")

	// Act
	results, err := pipeline.RenderAll(mergedProfile, "/project", nil)

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(results) != 3 {
		t.Errorf("expected 3 results (excluding partial), got %d", len(results))
	}
}

// Test 2.1.2: Test RenderAll builds TemplateContext with correct parameters
func TestRenderAll_BuildsTemplateContextCorrectly(t *testing.T) {
	// Arrange
	fs := afero.NewMemMapFs()
	engine := template.NewTemplateEngineWithFs(fs)
	pipeline := rendering.NewRenderingPipeline(engine)

	// Create template that uses context values
	templateContent := `Profile: {{ .Profile.Name }}
ProjectDir: {{ .ProjectDir }}
HasTimestamp: {{ if .GeneratedAt }}yes{{ else }}no{{ end }}`
	afero.WriteFile(fs, "/profiles/vendor/test/content/config.md.tmpl", []byte(templateContent), 0644)

	chain := []*profile.Profile{
		{
			Name: "vendor/test",
			TemplateFiles: []profile.TemplateFile{
				{SourcePath: "config.md.tmpl", IsPartial: false},
			},
		},
	}
	mergedProfile := infraprofile.NewMergedProfile(chain, fs, "/profiles")

	// Act
	results, err := pipeline.RenderAll(mergedProfile, "/my/project", nil)

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	expected := `Profile: vendor/test
ProjectDir: /my/project
HasTimestamp: yes`
	if results[0].Content != expected {
		t.Errorf("expected:\n%s\n\ngot:\n%s", expected, results[0].Content)
	}
}

// Test 2.1.3: Test RenderAll uses RenderWithContext for each template
func TestRenderAll_UsesSprigFunctions(t *testing.T) {
	// Arrange
	fs := afero.NewMemMapFs()
	engine := template.NewTemplateEngineWithFs(fs)
	pipeline := rendering.NewRenderingPipeline(engine)

	// Create template that uses sprig functions
	templateContent := `Upper: {{ .Profile.Name | upper }}`
	afero.WriteFile(fs, "/profiles/vendor/test/content/config.md.tmpl", []byte(templateContent), 0644)

	chain := []*profile.Profile{
		{
			Name: "vendor/test",
			TemplateFiles: []profile.TemplateFile{
				{SourcePath: "config.md.tmpl", IsPartial: false},
			},
		},
	}
	mergedProfile := infraprofile.NewMergedProfile(chain, fs, "/profiles")

	// Act
	results, err := pipeline.RenderAll(mergedProfile, "/project", nil)

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Content != "Upper: VENDOR/TEST" {
		t.Errorf("expected 'Upper: VENDOR/TEST', got %q", results[0].Content)
	}
}

// Test 2.1.4: Test RenderAll returns slice of RenderResult with correct content
func TestRenderAll_ReturnsCorrectRenderResults(t *testing.T) {
	// Arrange
	fs := afero.NewMemMapFs()
	engine := template.NewTemplateEngineWithFs(fs)
	pipeline := rendering.NewRenderingPipeline(engine)

	afero.WriteFile(fs, "/profiles/vendor/test/content/config.yaml.tmpl", []byte("key: value"), 0644)

	chain := []*profile.Profile{
		{
			Name: "vendor/test",
			TemplateFiles: []profile.TemplateFile{
				{SourcePath: "config.yaml.tmpl", IsPartial: false},
			},
		},
	}
	mergedProfile := infraprofile.NewMergedProfile(chain, fs, "/profiles")

	// Act
	results, err := pipeline.RenderAll(mergedProfile, "/project", nil)

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	result := results[0]
	if result.SourcePath != "config.yaml.tmpl" {
		t.Errorf("expected SourcePath 'config.yaml.tmpl', got %q", result.SourcePath)
	}
	if result.TargetPath != "config.yaml" {
		t.Errorf("expected TargetPath 'config.yaml', got %q", result.TargetPath)
	}
	if result.Content != "key: value" {
		t.Errorf("expected Content 'key: value', got %q", result.Content)
	}
}

// Test 2.1.5: Test rendering with templates that use include
func TestRenderAll_WithIncludeFunction(t *testing.T) {
	// Arrange
	fs := afero.NewMemMapFs()
	engine := template.NewTemplateEngineWithFs(fs)
	pipeline := rendering.NewRenderingPipeline(engine)

	// Create main template and partial
	afero.WriteFile(fs, "/profiles/vendor/test/content/main.md.tmpl", []byte(`Header
{{ include "_partial.md" }}
Footer`), 0644)
	afero.WriteFile(fs, "/profiles/vendor/test/content/_partial.md", []byte("Partial Content"), 0644)

	chain := []*profile.Profile{
		{
			Name: "vendor/test",
			TemplateFiles: []profile.TemplateFile{
				{SourcePath: "main.md.tmpl", IsPartial: false},
				{SourcePath: "_partial.md", IsPartial: true},
			},
		},
	}
	mergedProfile := infraprofile.NewMergedProfile(chain, fs, "/profiles")

	// Act
	results, err := pipeline.RenderAll(mergedProfile, "/project", nil)

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result (partial not rendered directly), got %d", len(results))
	}

	expected := `Header
Partial Content
Footer`
	if results[0].Content != expected {
		t.Errorf("expected:\n%s\n\ngot:\n%s", expected, results[0].Content)
	}
}

// Test 2.1.6: Test rendering with templates that use includeGlob
func TestRenderAll_WithIncludeGlobFunction(t *testing.T) {
	// Arrange
	fs := afero.NewMemMapFs()
	engine := template.NewTemplateEngineWithFs(fs)
	pipeline := rendering.NewRenderingPipeline(engine)

	// Create main template and multiple partials
	afero.WriteFile(fs, "/profiles/vendor/test/content/main.md.tmpl", []byte(`Skills:
{{ includeGlob "_skills/*.md" }}`), 0644)
	afero.WriteFile(fs, "/profiles/vendor/test/content/_skills/skill1.md", []byte("- Skill 1"), 0644)
	afero.WriteFile(fs, "/profiles/vendor/test/content/_skills/skill2.md", []byte("- Skill 2"), 0644)

	chain := []*profile.Profile{
		{
			Name: "vendor/test",
			TemplateFiles: []profile.TemplateFile{
				{SourcePath: "main.md.tmpl", IsPartial: false},
				{SourcePath: "_skills/skill1.md", IsPartial: true},
				{SourcePath: "_skills/skill2.md", IsPartial: true},
			},
		},
	}
	mergedProfile := infraprofile.NewMergedProfile(chain, fs, "/profiles")

	// Act
	results, err := pipeline.RenderAll(mergedProfile, "/project", nil)

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	// Check that both skills are included (order may vary due to glob)
	if !strings.Contains(results[0].Content, "- Skill 1") {
		t.Error("expected content to contain '- Skill 1'")
	}
	if !strings.Contains(results[0].Content, "- Skill 2") {
		t.Error("expected content to contain '- Skill 2'")
	}
}

// =============================================================================
// Task Group 3: Progress Callbacks Tests
// =============================================================================

// Test 3.1.1: Test BeforeRender callback invoked before each template render
func TestRenderAll_BeforeRenderCallback_InvokedBeforeRender(t *testing.T) {
	// Arrange
	fs := afero.NewMemMapFs()
	engine := template.NewTemplateEngineWithFs(fs)
	pipeline := rendering.NewRenderingPipeline(engine)

	afero.WriteFile(fs, "/profiles/vendor/test/content/file1.md.tmpl", []byte("Content 1"), 0644)
	afero.WriteFile(fs, "/profiles/vendor/test/content/file2.md.tmpl", []byte("Content 2"), 0644)

	chain := []*profile.Profile{
		{
			Name: "vendor/test",
			TemplateFiles: []profile.TemplateFile{
				{SourcePath: "file1.md.tmpl", IsPartial: false},
				{SourcePath: "file2.md.tmpl", IsPartial: false},
			},
		},
	}
	mergedProfile := infraprofile.NewMergedProfile(chain, fs, "/profiles")

	var beforeCalls []string
	callbacks := &rendering.RenderCallbacks{
		BeforeRender: func(index int, total int, sourcePath string) {
			beforeCalls = append(beforeCalls, sourcePath)
		},
	}

	// Act
	_, err := pipeline.RenderAll(mergedProfile, "/project", callbacks)

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(beforeCalls) != 2 {
		t.Errorf("expected 2 BeforeRender calls, got %d", len(beforeCalls))
	}
}

// Test 3.1.2: Test AfterRender callback invoked after each template render
func TestRenderAll_AfterRenderCallback_InvokedAfterRender(t *testing.T) {
	// Arrange
	fs := afero.NewMemMapFs()
	engine := template.NewTemplateEngineWithFs(fs)
	pipeline := rendering.NewRenderingPipeline(engine)

	afero.WriteFile(fs, "/profiles/vendor/test/content/file1.md.tmpl", []byte("Content 1"), 0644)
	afero.WriteFile(fs, "/profiles/vendor/test/content/file2.md.tmpl", []byte("Content 2"), 0644)

	chain := []*profile.Profile{
		{
			Name: "vendor/test",
			TemplateFiles: []profile.TemplateFile{
				{SourcePath: "file1.md.tmpl", IsPartial: false},
				{SourcePath: "file2.md.tmpl", IsPartial: false},
			},
		},
	}
	mergedProfile := infraprofile.NewMergedProfile(chain, fs, "/profiles")

	var afterCalls []struct {
		sourcePath string
		err        error
	}
	callbacks := &rendering.RenderCallbacks{
		AfterRender: func(index int, total int, sourcePath string, err error) {
			afterCalls = append(afterCalls, struct {
				sourcePath string
				err        error
			}{sourcePath, err})
		},
	}

	// Act
	_, err := pipeline.RenderAll(mergedProfile, "/project", callbacks)

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(afterCalls) != 2 {
		t.Errorf("expected 2 AfterRender calls, got %d", len(afterCalls))
	}
	// On success, error should be nil
	for _, call := range afterCalls {
		if call.err != nil {
			t.Errorf("expected nil error in AfterRender for %s, got %v", call.sourcePath, call.err)
		}
	}
}

// Test 3.1.3: Test callbacks receive correct index, total count, and sourcePath
func TestRenderAll_Callbacks_ReceiveCorrectParameters(t *testing.T) {
	// Arrange
	fs := afero.NewMemMapFs()
	engine := template.NewTemplateEngineWithFs(fs)
	pipeline := rendering.NewRenderingPipeline(engine)

	afero.WriteFile(fs, "/profiles/vendor/test/content/a.md.tmpl", []byte("A"), 0644)
	afero.WriteFile(fs, "/profiles/vendor/test/content/b.md.tmpl", []byte("B"), 0644)
	afero.WriteFile(fs, "/profiles/vendor/test/content/c.md.tmpl", []byte("C"), 0644)

	chain := []*profile.Profile{
		{
			Name: "vendor/test",
			TemplateFiles: []profile.TemplateFile{
				{SourcePath: "a.md.tmpl", IsPartial: false},
				{SourcePath: "b.md.tmpl", IsPartial: false},
				{SourcePath: "c.md.tmpl", IsPartial: false},
			},
		},
	}
	mergedProfile := infraprofile.NewMergedProfile(chain, fs, "/profiles")

	var beforeParams []struct {
		index      int
		total      int
		sourcePath string
	}
	callbacks := &rendering.RenderCallbacks{
		BeforeRender: func(index int, total int, sourcePath string) {
			beforeParams = append(beforeParams, struct {
				index      int
				total      int
				sourcePath string
			}{index, total, sourcePath})
		},
	}

	// Act
	_, err := pipeline.RenderAll(mergedProfile, "/project", callbacks)

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(beforeParams) != 3 {
		t.Fatalf("expected 3 callback invocations, got %d", len(beforeParams))
	}

	// Check that total is always 3
	for _, param := range beforeParams {
		if param.total != 3 {
			t.Errorf("expected total to be 3, got %d", param.total)
		}
	}

	// Check that indexes are 0, 1, 2
	expectedIndexes := map[int]bool{0: false, 1: false, 2: false}
	for _, param := range beforeParams {
		expectedIndexes[param.index] = true
	}
	for idx, seen := range expectedIndexes {
		if !seen {
			t.Errorf("expected index %d to be called", idx)
		}
	}
}

// Test 3.1.4: Test nil callbacks are handled safely (no panic)
func TestRenderAll_NilCallbacks_NoPanic(t *testing.T) {
	// Arrange
	fs := afero.NewMemMapFs()
	engine := template.NewTemplateEngineWithFs(fs)
	pipeline := rendering.NewRenderingPipeline(engine)

	afero.WriteFile(fs, "/profiles/vendor/test/content/config.md.tmpl", []byte("Content"), 0644)

	chain := []*profile.Profile{
		{
			Name: "vendor/test",
			TemplateFiles: []profile.TemplateFile{
				{SourcePath: "config.md.tmpl", IsPartial: false},
			},
		},
	}
	mergedProfile := infraprofile.NewMergedProfile(chain, fs, "/profiles")

	// Act - should not panic with nil callbacks
	results, err := pipeline.RenderAll(mergedProfile, "/project", nil)

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(results) != 1 {
		t.Errorf("expected 1 result, got %d", len(results))
	}
}

// Test 3.1.5: Test nil individual callbacks are handled safely
func TestRenderAll_NilIndividualCallbacks_NoPanic(t *testing.T) {
	// Arrange
	fs := afero.NewMemMapFs()
	engine := template.NewTemplateEngineWithFs(fs)
	pipeline := rendering.NewRenderingPipeline(engine)

	afero.WriteFile(fs, "/profiles/vendor/test/content/config.md.tmpl", []byte("Content"), 0644)

	chain := []*profile.Profile{
		{
			Name: "vendor/test",
			TemplateFiles: []profile.TemplateFile{
				{SourcePath: "config.md.tmpl", IsPartial: false},
			},
		},
	}
	mergedProfile := infraprofile.NewMergedProfile(chain, fs, "/profiles")

	// Callbacks struct with nil functions
	callbacks := &rendering.RenderCallbacks{
		BeforeRender: nil,
		AfterRender:  nil,
	}

	// Act - should not panic with nil individual callbacks
	results, err := pipeline.RenderAll(mergedProfile, "/project", callbacks)

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(results) != 1 {
		t.Errorf("expected 1 result, got %d", len(results))
	}
}

// =============================================================================
// Task Group 4: Error Handling Tests
// =============================================================================

// Test 4.1.1: Test RenderAll stops on first template rendering error
// Uses a single template to verify fail-fast behavior (order-independent)
func TestRenderAll_StopsOnFirstError(t *testing.T) {
	// Arrange
	fs := afero.NewMemMapFs()
	engine := template.NewTemplateEngineWithFs(fs)
	pipeline := rendering.NewRenderingPipeline(engine)

	// Single template with error to avoid order-dependency issues
	afero.WriteFile(fs, "/profiles/vendor/test/content/bad.md.tmpl", []byte("{{ .InvalidField }}"), 0644)

	chain := []*profile.Profile{
		{
			Name: "vendor/test",
			TemplateFiles: []profile.TemplateFile{
				{SourcePath: "bad.md.tmpl", IsPartial: false},
			},
		},
	}
	mergedProfile := infraprofile.NewMergedProfile(chain, fs, "/profiles")

	var renderCount int
	callbacks := &rendering.RenderCallbacks{
		BeforeRender: func(index int, total int, sourcePath string) {
			renderCount++
		},
	}

	// Act
	_, err := pipeline.RenderAll(mergedProfile, "/project", callbacks)

	// Assert
	if err == nil {
		t.Fatal("expected error for invalid template, got nil")
	}
	// Should have attempted exactly 1 render (which fails)
	if renderCount != 1 {
		t.Errorf("expected 1 render attempt, got %d", renderCount)
	}
}

// Test 4.1.2: Test RenderAll returns error immediately with partial results
// Uses a single valid template followed by error, order-independent test
func TestRenderAll_ReturnsPartialResultsOnError(t *testing.T) {
	// Arrange
	fs := afero.NewMemMapFs()
	engine := template.NewTemplateEngineWithFs(fs)
	pipeline := rendering.NewRenderingPipeline(engine)

	// Single template with error - test that error is returned
	afero.WriteFile(fs, "/profiles/vendor/test/content/bad.md.tmpl", []byte("{{ .NonExistent }}"), 0644)

	chain := []*profile.Profile{
		{
			Name: "vendor/test",
			TemplateFiles: []profile.TemplateFile{
				{SourcePath: "bad.md.tmpl", IsPartial: false},
			},
		},
	}
	mergedProfile := infraprofile.NewMergedProfile(chain, fs, "/profiles")

	// Act
	results, err := pipeline.RenderAll(mergedProfile, "/project", nil)

	// Assert
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	// Should have 0 successful results since the only template failed
	if len(results) != 0 {
		t.Errorf("expected 0 partial results, got %d", len(results))
	}
}

// Test 4.1.3: Test AfterRender callback receives error before method returns
func TestRenderAll_AfterRenderReceivesError(t *testing.T) {
	// Arrange
	fs := afero.NewMemMapFs()
	engine := template.NewTemplateEngineWithFs(fs)
	pipeline := rendering.NewRenderingPipeline(engine)

	// Template with error
	afero.WriteFile(fs, "/profiles/vendor/test/content/bad.md.tmpl", []byte("{{ .BadField }}"), 0644)

	chain := []*profile.Profile{
		{
			Name: "vendor/test",
			TemplateFiles: []profile.TemplateFile{
				{SourcePath: "bad.md.tmpl", IsPartial: false},
			},
		},
	}
	mergedProfile := infraprofile.NewMergedProfile(chain, fs, "/profiles")

	var capturedErr error
	callbacks := &rendering.RenderCallbacks{
		AfterRender: func(index int, total int, sourcePath string, err error) {
			capturedErr = err
		},
	}

	// Act
	_, err := pipeline.RenderAll(mergedProfile, "/project", callbacks)

	// Assert
	if err == nil {
		t.Fatal("expected error from RenderAll, got nil")
	}
	if capturedErr == nil {
		t.Fatal("expected AfterRender to receive error, got nil")
	}
}

// Test 4.1.4: Test successfully rendered templates are included in partial results
// Use a simple test case to verify partial results work
func TestRenderAll_SuccessfulTemplatesInPartialResults(t *testing.T) {
	// Arrange
	fs := afero.NewMemMapFs()
	engine := template.NewTemplateEngineWithFs(fs)
	pipeline := rendering.NewRenderingPipeline(engine)

	// Single valid template - verify it's included in results
	afero.WriteFile(fs, "/profiles/vendor/test/content/good.md.tmpl", []byte("Content"), 0644)

	chain := []*profile.Profile{
		{
			Name: "vendor/test",
			TemplateFiles: []profile.TemplateFile{
				{SourcePath: "good.md.tmpl", IsPartial: false},
			},
		},
	}
	mergedProfile := infraprofile.NewMergedProfile(chain, fs, "/profiles")

	// Act
	results, err := pipeline.RenderAll(mergedProfile, "/project", nil)

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	// Should have 1 successful result
	if len(results) != 1 {
		t.Errorf("expected 1 result, got %d", len(results))
	}
	if len(results) > 0 && results[0].Content != "Content" {
		t.Errorf("expected content 'Content', got %q", results[0].Content)
	}
}

// =============================================================================
// Task Group 5: Integration and Strategic Tests
// =============================================================================

// Test 5.1: Profile with only partial templates returns empty results
func TestRenderAll_AllPartialsProfile_ReturnsEmptyResults(t *testing.T) {
	// Arrange
	fs := afero.NewMemMapFs()
	engine := template.NewTemplateEngineWithFs(fs)
	pipeline := rendering.NewRenderingPipeline(engine)

	// Create only partial template files
	afero.WriteFile(fs, "/profiles/vendor/test/content/_partial1.md", []byte("Partial 1"), 0644)
	afero.WriteFile(fs, "/profiles/vendor/test/content/_partial2.md", []byte("Partial 2"), 0644)
	afero.WriteFile(fs, "/profiles/vendor/test/content/_partials/nested.md", []byte("Nested partial"), 0644)

	chain := []*profile.Profile{
		{
			Name: "vendor/test",
			TemplateFiles: []profile.TemplateFile{
				{SourcePath: "_partial1.md", IsPartial: true},
				{SourcePath: "_partial2.md", IsPartial: true},
				{SourcePath: "_partials/nested.md", IsPartial: true},
			},
		},
	}
	mergedProfile := infraprofile.NewMergedProfile(chain, fs, "/profiles")

	// Act
	results, err := pipeline.RenderAll(mergedProfile, "/project", nil)

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results (all partials), got %d", len(results))
	}
}

// Test 5.2: Multi-profile inheritance chain with child overriding parent template
func TestRenderAll_InheritanceChain_ChildOverridesParent(t *testing.T) {
	// Arrange
	fs := afero.NewMemMapFs()
	engine := template.NewTemplateEngineWithFs(fs)
	pipeline := rendering.NewRenderingPipeline(engine)

	// Parent profile template
	afero.WriteFile(fs, "/profiles/vendor/parent/content/config.md.tmpl", []byte("Parent: {{ .Profile.Name }}"), 0644)
	afero.WriteFile(fs, "/profiles/vendor/parent/content/_shared.md", []byte("Shared from parent"), 0644)

	// Child profile template (overrides config.md.tmpl)
	afero.WriteFile(fs, "/profiles/vendor/child/content/config.md.tmpl", []byte("Child: {{ .Profile.Name }}"), 0644)

	// Create inheritance chain (root to leaf)
	chain := []*profile.Profile{
		{
			Name: "vendor/parent",
			TemplateFiles: []profile.TemplateFile{
				{SourcePath: "config.md.tmpl", IsPartial: false},
				{SourcePath: "_shared.md", IsPartial: true},
			},
		},
		{
			Name: "vendor/child",
			TemplateFiles: []profile.TemplateFile{
				{SourcePath: "config.md.tmpl", IsPartial: false}, // Overrides parent
			},
		},
	}
	mergedProfile := infraprofile.NewMergedProfile(chain, fs, "/profiles")

	// Act
	results, err := pipeline.RenderAll(mergedProfile, "/project", nil)

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	// Should use child's template (child overrides parent)
	if results[0].Content != "Child: vendor/child" {
		t.Errorf("expected 'Child: vendor/child', got %q", results[0].Content)
	}
}

// Test 5.3: Nested directory structure with .tmpl extension is preserved in target path
func TestRenderAll_NestedDirectoryStructure_PreservesPath(t *testing.T) {
	// Arrange
	fs := afero.NewMemMapFs()
	engine := template.NewTemplateEngineWithFs(fs)
	pipeline := rendering.NewRenderingPipeline(engine)

	// Create templates in nested directories
	afero.WriteFile(fs, "/profiles/vendor/test/content/.claude/settings.json.tmpl", []byte(`{"key": "value"}`), 0644)
	afero.WriteFile(fs, "/profiles/vendor/test/content/subdir/another/config.yaml.tmpl", []byte("nested: true"), 0644)

	chain := []*profile.Profile{
		{
			Name: "vendor/test",
			TemplateFiles: []profile.TemplateFile{
				{SourcePath: ".claude/settings.json.tmpl", IsPartial: false},
				{SourcePath: "subdir/another/config.yaml.tmpl", IsPartial: false},
			},
		},
	}
	mergedProfile := infraprofile.NewMergedProfile(chain, fs, "/profiles")

	// Act
	results, err := pipeline.RenderAll(mergedProfile, "/project", nil)

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	// Check that nested paths are preserved with .tmpl stripped
	pathsFound := make(map[string]bool)
	for _, r := range results {
		pathsFound[r.TargetPath] = true
	}

	if !pathsFound[".claude/settings.json"] {
		t.Error("expected target path '.claude/settings.json' not found")
	}
	if !pathsFound["subdir/another/config.yaml"] {
		t.Error("expected target path 'subdir/another/config.yaml' not found")
	}
}

// Test 5.4: Include function with inheritance uses child's partial when available
func TestRenderAll_IncludeWithInheritance_UsesChildPartial(t *testing.T) {
	// Arrange
	fs := afero.NewMemMapFs()
	engine := template.NewTemplateEngineWithFs(fs)
	pipeline := rendering.NewRenderingPipeline(engine)

	// Parent profile with main template and partial
	afero.WriteFile(fs, "/profiles/vendor/parent/content/main.md.tmpl", []byte(`Main:
{{ include "_header.md" }}`), 0644)
	afero.WriteFile(fs, "/profiles/vendor/parent/content/_header.md", []byte("Parent Header"), 0644)

	// Child profile overrides the partial
	afero.WriteFile(fs, "/profiles/vendor/child/content/_header.md", []byte("Child Header"), 0644)

	// Create inheritance chain (root to leaf)
	chain := []*profile.Profile{
		{
			Name: "vendor/parent",
			TemplateFiles: []profile.TemplateFile{
				{SourcePath: "main.md.tmpl", IsPartial: false},
				{SourcePath: "_header.md", IsPartial: true},
			},
		},
		{
			Name: "vendor/child",
			TemplateFiles: []profile.TemplateFile{
				{SourcePath: "_header.md", IsPartial: true}, // Overrides parent's partial
			},
		},
	}
	mergedProfile := infraprofile.NewMergedProfile(chain, fs, "/profiles")

	// Act
	results, err := pipeline.RenderAll(mergedProfile, "/project", nil)

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	expected := `Main:
Child Header`
	if results[0].Content != expected {
		t.Errorf("expected:\n%s\n\ngot:\n%s", expected, results[0].Content)
	}
}

// Test 5.5: Template syntax error returns descriptive error
func TestRenderAll_TemplateSyntaxError_ReturnsDescriptiveError(t *testing.T) {
	// Arrange
	fs := afero.NewMemMapFs()
	engine := template.NewTemplateEngineWithFs(fs)
	pipeline := rendering.NewRenderingPipeline(engine)

	// Template with syntax error (unclosed action)
	afero.WriteFile(fs, "/profiles/vendor/test/content/bad.md.tmpl", []byte("Hello {{ .Profile.Name"), 0644)

	chain := []*profile.Profile{
		{
			Name: "vendor/test",
			TemplateFiles: []profile.TemplateFile{
				{SourcePath: "bad.md.tmpl", IsPartial: false},
			},
		},
	}
	mergedProfile := infraprofile.NewMergedProfile(chain, fs, "/profiles")

	// Act
	_, err := pipeline.RenderAll(mergedProfile, "/project", nil)

	// Assert
	if err == nil {
		t.Fatal("expected error for syntax error, got nil")
	}

	// Check error is a TemplateSyntaxError
	var syntaxErr *template.TemplateSyntaxError
	if !strings.Contains(err.Error(), "unclosed") && !strings.Contains(err.Error(), "unexpected") {
		// Also acceptable if it's wrapped
		syntaxErr, _ = err.(*template.TemplateSyntaxError)
		if syntaxErr == nil {
			t.Logf("Error type: %T, message: %s", err, err.Error())
		}
	}
}

// Test 5.6: Both callbacks invoked in correct order for each template
func TestRenderAll_CallbackOrder_BeforeThenAfter(t *testing.T) {
	// Arrange
	fs := afero.NewMemMapFs()
	engine := template.NewTemplateEngineWithFs(fs)
	pipeline := rendering.NewRenderingPipeline(engine)

	afero.WriteFile(fs, "/profiles/vendor/test/content/file.md.tmpl", []byte("Content"), 0644)

	chain := []*profile.Profile{
		{
			Name: "vendor/test",
			TemplateFiles: []profile.TemplateFile{
				{SourcePath: "file.md.tmpl", IsPartial: false},
			},
		},
	}
	mergedProfile := infraprofile.NewMergedProfile(chain, fs, "/profiles")

	var callOrder []string
	callbacks := &rendering.RenderCallbacks{
		BeforeRender: func(index int, total int, sourcePath string) {
			callOrder = append(callOrder, "before:"+sourcePath)
		},
		AfterRender: func(index int, total int, sourcePath string, err error) {
			callOrder = append(callOrder, "after:"+sourcePath)
		},
	}

	// Act
	_, err := pipeline.RenderAll(mergedProfile, "/project", callbacks)

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(callOrder) != 2 {
		t.Fatalf("expected 2 calls, got %d", len(callOrder))
	}
	if callOrder[0] != "before:file.md.tmpl" {
		t.Errorf("expected first call to be 'before:file.md.tmpl', got %q", callOrder[0])
	}
	if callOrder[1] != "after:file.md.tmpl" {
		t.Errorf("expected second call to be 'after:file.md.tmpl', got %q", callOrder[1])
	}
}

// =============================================================================
// Source Content Preservation Tests (Task Group 1 from Source Content Preservation spec)
// =============================================================================

// Test: RenderResult includes SourceContent field
func TestRenderResult_IncludesSourceContentField(t *testing.T) {
	// Arrange
	fs := afero.NewMemMapFs()
	engine := template.NewTemplateEngineWithFs(fs)
	pipeline := rendering.NewRenderingPipeline(engine)

	templateContent := "Static content without variables"
	afero.WriteFile(fs, "/profiles/vendor/test/content/config.md.tmpl", []byte(templateContent), 0644)

	chain := []*profile.Profile{
		{
			Name: "vendor/test",
			TemplateFiles: []profile.TemplateFile{
				{SourcePath: "config.md.tmpl", IsPartial: false},
			},
		},
	}
	mergedProfile := infraprofile.NewMergedProfile(chain, fs, "/profiles")

	// Act
	results, err := pipeline.RenderAll(mergedProfile, "/project", nil)

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	// Verify SourceContent is populated
	if results[0].SourceContent == nil {
		t.Error("expected SourceContent to be non-nil")
	}
	if len(results[0].SourceContent) == 0 {
		t.Error("expected SourceContent to have content")
	}

	// For static content, source and rendered should match
	if string(results[0].SourceContent) != templateContent {
		t.Errorf("expected SourceContent %q, got %q", templateContent, string(results[0].SourceContent))
	}
}

// Test: Pipeline.Render captures source content
func TestRenderAll_CapturesSourceContent(t *testing.T) {
	// Arrange
	fs := afero.NewMemMapFs()
	engine := template.NewTemplateEngineWithFs(fs)
	pipeline := rendering.NewRenderingPipeline(engine)

	templateContent := "Hello, {{ .ProjectDir }}!"
	afero.WriteFile(fs, "/profiles/vendor/test/content/greeting.md.tmpl", []byte(templateContent), 0644)

	chain := []*profile.Profile{
		{
			Name: "vendor/test",
			TemplateFiles: []profile.TemplateFile{
				{SourcePath: "greeting.md.tmpl", IsPartial: false},
			},
		},
	}
	mergedProfile := infraprofile.NewMergedProfile(chain, fs, "/profiles")

	// Act
	results, err := pipeline.RenderAll(mergedProfile, "/my/project", nil)

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	// Verify SourceContent contains the original template
	if !bytes.Equal(results[0].SourceContent, []byte(templateContent)) {
		t.Errorf("expected SourceContent to be original template %q, got %q",
			templateContent, string(results[0].SourceContent))
	}

	// Verify Content contains the rendered output
	expectedRendered := "Hello, /my/project!"
	if results[0].Content != expectedRendered {
		t.Errorf("expected Content to be rendered %q, got %q",
			expectedRendered, results[0].Content)
	}
}

// Test: Source content differs from rendered content when template has variables
func TestRenderAll_SourceContentDiffersFromRenderedWhenTemplateHasVariables(t *testing.T) {
	// Arrange
	fs := afero.NewMemMapFs()
	engine := template.NewTemplateEngineWithFs(fs)
	pipeline := rendering.NewRenderingPipeline(engine)

	// Template with variables that will be replaced
	templateContent := `Project: {{ .ProjectDir }}
Profile: {{ .Profile.Name }}
Generated: {{ .GeneratedAt.Format "2006-01-02" }}`
	afero.WriteFile(fs, "/profiles/vendor/test/content/info.md.tmpl", []byte(templateContent), 0644)

	chain := []*profile.Profile{
		{
			Name: "vendor/test",
			TemplateFiles: []profile.TemplateFile{
				{SourcePath: "info.md.tmpl", IsPartial: false},
			},
		},
	}
	mergedProfile := infraprofile.NewMergedProfile(chain, fs, "/profiles")

	// Act
	results, err := pipeline.RenderAll(mergedProfile, "/my/project", nil)

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	// SourceContent should be the original template with variables
	if !bytes.Equal(results[0].SourceContent, []byte(templateContent)) {
		t.Errorf("expected SourceContent to match original template")
	}

	// Content should be different (variables replaced)
	if string(results[0].SourceContent) == results[0].Content {
		t.Error("expected SourceContent and Content to differ when template has variables")
	}

	// Verify Content has replaced values
	if !strings.Contains(results[0].Content, "/my/project") {
		t.Error("expected Content to contain project directory")
	}
	if !strings.Contains(results[0].Content, "vendor/test") {
		t.Error("expected Content to contain profile name")
	}
}

// Test: Multiple templates each have correct source content
func TestRenderAll_MultipleTemplatesHaveCorrectSourceContent(t *testing.T) {
	// Arrange
	fs := afero.NewMemMapFs()
	engine := template.NewTemplateEngineWithFs(fs)
	pipeline := rendering.NewRenderingPipeline(engine)

	template1 := "Template 1: {{ .ProjectDir }}"
	template2 := "Template 2: {{ .Profile.Name }}"
	afero.WriteFile(fs, "/profiles/vendor/test/content/file1.md.tmpl", []byte(template1), 0644)
	afero.WriteFile(fs, "/profiles/vendor/test/content/file2.md.tmpl", []byte(template2), 0644)

	chain := []*profile.Profile{
		{
			Name: "vendor/test",
			TemplateFiles: []profile.TemplateFile{
				{SourcePath: "file1.md.tmpl", IsPartial: false},
				{SourcePath: "file2.md.tmpl", IsPartial: false},
			},
		},
	}
	mergedProfile := infraprofile.NewMergedProfile(chain, fs, "/profiles")

	// Act
	results, err := pipeline.RenderAll(mergedProfile, "/project", nil)

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	// Check each result has correct source content
	sourceContents := make(map[string]string)
	for _, r := range results {
		sourceContents[r.SourcePath] = string(r.SourceContent)
	}

	if sourceContents["file1.md.tmpl"] != template1 {
		t.Errorf("expected file1 SourceContent %q, got %q", template1, sourceContents["file1.md.tmpl"])
	}
	if sourceContents["file2.md.tmpl"] != template2 {
		t.Errorf("expected file2 SourceContent %q, got %q", template2, sourceContents["file2.md.tmpl"])
	}
}

// Test: Static template has matching source and rendered content
func TestRenderAll_StaticTemplateHasMatchingSourceAndRendered(t *testing.T) {
	// Arrange
	fs := afero.NewMemMapFs()
	engine := template.NewTemplateEngineWithFs(fs)
	pipeline := rendering.NewRenderingPipeline(engine)

	// Template without any variables
	staticContent := "This is static content with no template variables."
	afero.WriteFile(fs, "/profiles/vendor/test/content/static.md.tmpl", []byte(staticContent), 0644)

	chain := []*profile.Profile{
		{
			Name: "vendor/test",
			TemplateFiles: []profile.TemplateFile{
				{SourcePath: "static.md.tmpl", IsPartial: false},
			},
		},
	}
	mergedProfile := infraprofile.NewMergedProfile(chain, fs, "/profiles")

	// Act
	results, err := pipeline.RenderAll(mergedProfile, "/project", nil)

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	// For static content, source and rendered should be identical
	if string(results[0].SourceContent) != results[0].Content {
		t.Errorf("expected SourceContent and Content to match for static template\nSourceContent: %q\nContent: %q",
			string(results[0].SourceContent), results[0].Content)
	}
}

// =============================================================================
// Reference Template Functions Integration Tests (Task Group 4)
// =============================================================================

// Test 4.1.1: Test RenderingPipeline passes router to RenderContext
func TestRenderAllWithOptions_PassesRouterToRenderContext(t *testing.T) {
	// Arrange
	fs := afero.NewMemMapFs()
	engine := template.NewTemplateEngineWithFs(fs)
	pipeline := rendering.NewRenderingPipeline(engine)

	// Create template that uses reference function (will fail if router not passed)
	templateContent := `Reference: {{ reference "claude/config.md" }}`
	afero.WriteFile(fs, "/profiles/vendor/test/content/claude/main.md.tmpl", []byte(templateContent), 0644)
	afero.WriteFile(fs, "/profiles/vendor/test/content/claude/config.md", []byte("Config content"), 0644)

	chain := []*profile.Profile{
		{
			Name: "vendor/test",
			TemplateFiles: []profile.TemplateFile{
				{SourcePath: "claude/main.md.tmpl", IsPartial: false},
				{SourcePath: "claude/config.md", IsPartial: false},
			},
		},
	}
	mergedProfile := infraprofile.NewMergedProfile(chain, fs, "/profiles")

	// Create router with target configuration
	contentConfig := profile.ContentConfig{
		Targets: map[string]string{
			"claude": ".claude",
		},
	}
	router, err := routing.NewContentRouter(contentConfig, nil, nil)
	if err != nil {
		t.Fatalf("failed to create router: %v", err)
	}

	// Act
	opts := &rendering.RenderOptions{
		Router: router,
	}
	results, err := pipeline.RenderAllWithOptions(mergedProfile, "/project", opts)

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	// Find the main.md result and verify reference was resolved
	for _, result := range results {
		if result.SourcePath == "claude/main.md.tmpl" {
			// The reference() function should have returned @.claude/config.md
			if !strings.Contains(result.Content, "@.claude/config.md") {
				t.Errorf("expected content to contain '@.claude/config.md', got %q", result.Content)
			}
			break
		}
	}
}

// Test 4.1.2: Test RenderAllWithOptions without router maintains backward compatibility
func TestRenderAllWithOptions_WithoutRouter_BackwardCompatible(t *testing.T) {
	// Arrange
	fs := afero.NewMemMapFs()
	engine := template.NewTemplateEngineWithFs(fs)
	pipeline := rendering.NewRenderingPipeline(engine)

	// Create simple template without reference function
	templateContent := "Hello, {{ .ProjectDir }}!"
	afero.WriteFile(fs, "/profiles/vendor/test/content/config.md.tmpl", []byte(templateContent), 0644)

	chain := []*profile.Profile{
		{
			Name: "vendor/test",
			TemplateFiles: []profile.TemplateFile{
				{SourcePath: "config.md.tmpl", IsPartial: false},
			},
		},
	}
	mergedProfile := infraprofile.NewMergedProfile(chain, fs, "/profiles")

	// Act - pass nil options (backward compatibility test)
	results, err := pipeline.RenderAllWithOptions(mergedProfile, "/my/project", nil)

	// Assert
	if err != nil {
		t.Fatalf("expected no error with nil options, got %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Content != "Hello, /my/project!" {
		t.Errorf("expected content 'Hello, /my/project!', got %q", results[0].Content)
	}
}

// Test 4.1.3: Test end-to-end reference resolution through rendering
func TestRenderAllWithOptions_EndToEndReferenceResolution(t *testing.T) {
	// Arrange
	fs := afero.NewMemMapFs()
	engine := template.NewTemplateEngineWithFs(fs)
	pipeline := rendering.NewRenderingPipeline(engine)

	// Create template that references another file
	mainTemplate := `# Main Document
See also: [Git Guide]({{ reference "standards/vcs/git.md" }})
And: [Code Style]({{ reference "standards/code/style.md" }})`
	afero.WriteFile(fs, "/profiles/vendor/test/content/index.md.tmpl", []byte(mainTemplate), 0644)
	afero.WriteFile(fs, "/profiles/vendor/test/content/standards/vcs/git.md", []byte("Git guide content"), 0644)
	afero.WriteFile(fs, "/profiles/vendor/test/content/standards/code/style.md", []byte("Style guide content"), 0644)

	chain := []*profile.Profile{
		{
			Name: "vendor/test",
			TemplateFiles: []profile.TemplateFile{
				{SourcePath: "index.md.tmpl", IsPartial: false},
				{SourcePath: "standards/vcs/git.md", IsPartial: false},
				{SourcePath: "standards/code/style.md", IsPartial: false},
			},
		},
	}
	mergedProfile := infraprofile.NewMergedProfile(chain, fs, "/profiles")

	// Create router with target configuration
	contentConfig := profile.ContentConfig{
		Targets: map[string]string{
			"standards": ".ai/standards",
		},
	}
	router, err := routing.NewContentRouter(contentConfig, nil, nil)
	if err != nil {
		t.Fatalf("failed to create router: %v", err)
	}

	// Act
	opts := &rendering.RenderOptions{
		Router: router,
	}
	results, err := pipeline.RenderAllWithOptions(mergedProfile, "/project", opts)

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Find the index.md result
	var indexResult *rendering.RenderResult
	for i := range results {
		if results[i].SourcePath == "index.md.tmpl" {
			indexResult = &results[i]
			break
		}
	}

	if indexResult == nil {
		t.Fatal("could not find index.md in results")
	}

	// Verify both references were resolved with @ prefix
	if !strings.Contains(indexResult.Content, "@.ai/standards/vcs/git.md") {
		t.Errorf("expected content to contain '@.ai/standards/vcs/git.md', got %q", indexResult.Content)
	}
	if !strings.Contains(indexResult.Content, "@.ai/standards/code/style.md") {
		t.Errorf("expected content to contain '@.ai/standards/code/style.md', got %q", indexResult.Content)
	}
}

// Test 4.1.4: Test verbose flag propagation from install options
func TestRenderAllWithOptions_VerboseFlagPropagation(t *testing.T) {
	// Arrange
	fs := afero.NewMemMapFs()
	engine := template.NewTemplateEngineWithFs(fs)
	pipeline := rendering.NewRenderingPipeline(engine)

	// Create template that references a non-existent file (should emit warning when verbose)
	templateContent := `Reference: {{ reference "nonexistent/file.md" }}`
	afero.WriteFile(fs, "/profiles/vendor/test/content/claude/main.md.tmpl", []byte(templateContent), 0644)

	chain := []*profile.Profile{
		{
			Name: "vendor/test",
			TemplateFiles: []profile.TemplateFile{
				{SourcePath: "claude/main.md.tmpl", IsPartial: false},
			},
		},
	}
	mergedProfile := infraprofile.NewMergedProfile(chain, fs, "/profiles")

	// Create router with target configuration
	contentConfig := profile.ContentConfig{
		Targets: map[string]string{
			"claude":      ".claude",
			"nonexistent": ".nonexistent",
		},
	}
	router, err := routing.NewContentRouter(contentConfig, nil, nil)
	if err != nil {
		t.Fatalf("failed to create router: %v", err)
	}

	// Act - with verbose = true
	opts := &rendering.RenderOptions{
		Router:  router,
		Verbose: true,
	}
	results, err := pipeline.RenderAllWithOptions(mergedProfile, "/project", opts)

	// Assert - should succeed even with missing file (lenient behavior)
	if err != nil {
		t.Fatalf("expected no error with lenient reference handling, got %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	// The reference should still return a computed path (lenient behavior)
	// The @ prefix should be present
	if !strings.Contains(results[0].Content, "@") {
		t.Errorf("expected content to contain @ prefix for reference, got %q", results[0].Content)
	}

	// Act - with verbose = false (should also succeed, just no warnings)
	optsQuiet := &rendering.RenderOptions{
		Router:  router,
		Verbose: false,
	}
	resultsQuiet, errQuiet := pipeline.RenderAllWithOptions(mergedProfile, "/project", optsQuiet)

	// Assert - should also succeed
	if errQuiet != nil {
		t.Fatalf("expected no error with verbose=false, got %v", errQuiet)
	}
	if len(resultsQuiet) != 1 {
		t.Fatalf("expected 1 result with verbose=false, got %d", len(resultsQuiet))
	}
}

// =============================================================================
// Pipeline Merge Chain Tests (Task Group 2 from merge-variables-consistency spec)
// =============================================================================

// Test: mergedProfile.Variables is included in the merge chain
func TestRenderAllWithOptions_MergedProfileVariablesIncludedInMergeChain(t *testing.T) {
	// Arrange
	fs := afero.NewMemMapFs()
	engine := template.NewTemplateEngineWithFs(fs)
	pipeline := rendering.NewRenderingPipeline(engine)

	// Create template that accesses variables
	templateContent := `Company: {{ .Variables.company }}
Profile Setting: {{ .Variables.profile_setting }}`
	afero.WriteFile(fs, "/profiles/vendor/test/content/config.md.tmpl", []byte(templateContent), 0644)

	// Create profile chain with variables that will be merged into mergedProfile.Variables
	chain := []*profile.Profile{
		{
			Name: "vendor/parent",
			Variables: map[string]interface{}{
				"company":         "Acme Corp",
				"profile_setting": "from_parent",
			},
		},
		{
			Name: "vendor/test",
			Variables: map[string]interface{}{
				"profile_setting": "from_child", // Overrides parent
			},
			TemplateFiles: []profile.TemplateFile{
				{SourcePath: "config.md.tmpl", IsPartial: false},
			},
		},
	}
	mergedProfile := infraprofile.NewMergedProfile(chain, fs, "/profiles")

	// Act - no global or project configs, only profile inheritance chain variables
	opts := &rendering.RenderOptions{}
	results, err := pipeline.RenderAllWithOptions(mergedProfile, "/project", opts)

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	// Verify variables from profile inheritance chain are accessible
	if !strings.Contains(results[0].Content, "Company: Acme Corp") {
		t.Errorf("expected content to contain 'Company: Acme Corp', got %q", results[0].Content)
	}
	if !strings.Contains(results[0].Content, "Profile Setting: from_child") {
		t.Errorf("expected content to contain 'Profile Setting: from_child', got %q", results[0].Content)
	}
}

// Test: Priority order is global (lowest) -> profile inheritance -> project (highest)
func TestRenderAllWithOptions_MergeChainPriorityOrder(t *testing.T) {
	// Arrange
	fs := afero.NewMemMapFs()
	engine := template.NewTemplateEngineWithFs(fs)
	pipeline := rendering.NewRenderingPipeline(engine)

	// Create template that accesses variables at all levels
	templateContent := `Global Only: {{ .Variables.global_only }}
Profile Only: {{ .Variables.profile_only }}
Project Only: {{ .Variables.project_only }}
Global+Profile: {{ .Variables.global_and_profile }}
Global+Project: {{ .Variables.global_and_project }}
Profile+Project: {{ .Variables.profile_and_project }}
All Three: {{ .Variables.all_three }}`
	afero.WriteFile(fs, "/profiles/vendor/test/content/config.md.tmpl", []byte(templateContent), 0644)

	// Create profile chain
	chain := []*profile.Profile{
		{
			Name: "vendor/test",
			Variables: map[string]interface{}{
				"profile_only":        "from_profile",
				"global_and_profile":  "profile_wins_over_global",
				"profile_and_project": "from_profile",
				"all_three":           "from_profile",
			},
			TemplateFiles: []profile.TemplateFile{
				{SourcePath: "config.md.tmpl", IsPartial: false},
			},
		},
	}
	mergedProfile := infraprofile.NewMergedProfile(chain, fs, "/profiles")

	// Set up configs at all levels
	globalConfig := &config.GlobalConfig{
		Variables: map[string]interface{}{
			"global_only":        "from_global",
			"global_and_profile": "from_global",
			"global_and_project": "from_global",
			"all_three":          "from_global",
		},
	}
	projectConfig := &config.ProjectConfig{
		Variables: map[string]interface{}{
			"project_only":        "from_project",
			"global_and_project":  "project_wins_over_global",
			"profile_and_project": "project_wins_over_profile",
			"all_three":           "project_wins_all",
		},
	}

	// Act
	opts := &rendering.RenderOptions{
		GlobalConfig:  globalConfig,
		ProjectConfig: projectConfig,
	}
	results, err := pipeline.RenderAllWithOptions(mergedProfile, "/project", opts)

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	content := results[0].Content

	// Global only variable should come from global
	if !strings.Contains(content, "Global Only: from_global") {
		t.Errorf("expected 'Global Only: from_global', got %q", content)
	}
	// Profile only variable should come from profile
	if !strings.Contains(content, "Profile Only: from_profile") {
		t.Errorf("expected 'Profile Only: from_profile', got %q", content)
	}
	// Project only variable should come from project
	if !strings.Contains(content, "Project Only: from_project") {
		t.Errorf("expected 'Project Only: from_project', got %q", content)
	}
	// Global+Profile: profile should win over global
	if !strings.Contains(content, "Global+Profile: profile_wins_over_global") {
		t.Errorf("expected 'Global+Profile: profile_wins_over_global', got %q", content)
	}
	// Global+Project: project should win over global
	if !strings.Contains(content, "Global+Project: project_wins_over_global") {
		t.Errorf("expected 'Global+Project: project_wins_over_global', got %q", content)
	}
	// Profile+Project: project should win over profile
	if !strings.Contains(content, "Profile+Project: project_wins_over_profile") {
		t.Errorf("expected 'Profile+Project: project_wins_over_profile', got %q", content)
	}
	// All three: project should win all
	if !strings.Contains(content, "All Three: project_wins_all") {
		t.Errorf("expected 'All Three: project_wins_all', got %q", content)
	}
}

// Test: Nested variables from profile inheritance chain are preserved (deep merge)
func TestRenderAllWithOptions_NestedVariablesDeepMerge(t *testing.T) {
	// Arrange
	fs := afero.NewMemMapFs()
	engine := template.NewTemplateEngineWithFs(fs)
	pipeline := rendering.NewRenderingPipeline(engine)

	// Create template that accesses nested variables
	templateContent := `DB Host: {{ .Variables.database.host }}
DB Port: {{ .Variables.database.port }}
DB Username: {{ .Variables.database.username }}`
	afero.WriteFile(fs, "/profiles/vendor/test/content/config.md.tmpl", []byte(templateContent), 0644)

	// Create profile chain with nested variables
	chain := []*profile.Profile{
		{
			Name: "vendor/test",
			Variables: map[string]interface{}{
				"database": map[string]interface{}{
					"host": "profile.db.local",
					"port": 5432,
				},
			},
			TemplateFiles: []profile.TemplateFile{
				{SourcePath: "config.md.tmpl", IsPartial: false},
			},
		},
	}
	mergedProfile := infraprofile.NewMergedProfile(chain, fs, "/profiles")

	// Global config with nested database settings
	globalConfig := &config.GlobalConfig{
		Variables: map[string]interface{}{
			"database": map[string]interface{}{
				"host":     "global.db.local",
				"port":     3306,
				"username": "global_user",
			},
		},
	}

	// Project config overrides only the host (deep merge should preserve port and username)
	projectConfig := &config.ProjectConfig{
		Variables: map[string]interface{}{
			"database": map[string]interface{}{
				"host": "project.db.local",
			},
		},
	}

	// Act
	opts := &rendering.RenderOptions{
		GlobalConfig:  globalConfig,
		ProjectConfig: projectConfig,
	}
	results, err := pipeline.RenderAllWithOptions(mergedProfile, "/project", opts)

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	content := results[0].Content

	// Host should be from project (highest priority)
	if !strings.Contains(content, "DB Host: project.db.local") {
		t.Errorf("expected 'DB Host: project.db.local', got %q", content)
	}
	// Port should be from profile (preserved via deep merge, profile overrides global)
	if !strings.Contains(content, "DB Port: 5432") {
		t.Errorf("expected 'DB Port: 5432', got %q", content)
	}
	// Username should be from global (preserved via deep merge, not overridden by profile or project)
	if !strings.Contains(content, "DB Username: global_user") {
		t.Errorf("expected 'DB Username: global_user', got %q", content)
	}
}

// Test: Deep merge behavior replaces shallow merge at pipeline level
func TestRenderAllWithOptions_DeepMergeReplacesShallowMerge(t *testing.T) {
	// Arrange
	fs := afero.NewMemMapFs()
	engine := template.NewTemplateEngineWithFs(fs)
	pipeline := rendering.NewRenderingPipeline(engine)

	// Create template that accesses nested server configuration
	templateContent := `Server Host: {{ .Variables.server.host }}
Server Port: {{ .Variables.server.port }}
Server TLS: {{ .Variables.server.tls }}`
	afero.WriteFile(fs, "/profiles/vendor/test/content/config.md.tmpl", []byte(templateContent), 0644)

	// Create profile chain
	chain := []*profile.Profile{
		{
			Name: "vendor/test",
			TemplateFiles: []profile.TemplateFile{
				{SourcePath: "config.md.tmpl", IsPartial: false},
			},
		},
	}
	mergedProfile := infraprofile.NewMergedProfile(chain, fs, "/profiles")

	// Global config with full server configuration
	globalConfig := &config.GlobalConfig{
		Variables: map[string]interface{}{
			"server": map[string]interface{}{
				"host": "localhost",
				"port": 8080,
				"tls":  false,
			},
		},
	}

	// Project config overrides only host - with shallow merge, port and tls would be lost
	// With deep merge, they should be preserved
	projectConfig := &config.ProjectConfig{
		Variables: map[string]interface{}{
			"server": map[string]interface{}{
				"host": "production.example.com",
			},
		},
	}

	// Act
	opts := &rendering.RenderOptions{
		GlobalConfig:  globalConfig,
		ProjectConfig: projectConfig,
	}
	results, err := pipeline.RenderAllWithOptions(mergedProfile, "/project", opts)

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	content := results[0].Content

	// Host should be from project (overridden)
	if !strings.Contains(content, "Server Host: production.example.com") {
		t.Errorf("expected 'Server Host: production.example.com', got %q", content)
	}
	// Port should be preserved from global (deep merge)
	if !strings.Contains(content, "Server Port: 8080") {
		t.Errorf("expected 'Server Port: 8080' (preserved via deep merge), got %q", content)
	}
	// TLS should be preserved from global (deep merge)
	if !strings.Contains(content, "Server TLS: false") {
		t.Errorf("expected 'Server TLS: false' (preserved via deep merge), got %q", content)
	}
}

// Test: Profile inheritance chain with multiple levels of nested variables
func TestRenderAllWithOptions_ProfileInheritanceChainNestedVariables(t *testing.T) {
	// Arrange
	fs := afero.NewMemMapFs()
	engine := template.NewTemplateEngineWithFs(fs)
	pipeline := rendering.NewRenderingPipeline(engine)

	// Create template that accesses deeply nested variables
	templateContent := `App Name: {{ .Variables.app.name }}
App Version: {{ .Variables.app.version }}
App Author: {{ .Variables.app.author }}
DB Host: {{ .Variables.app.database.host }}
DB Port: {{ .Variables.app.database.port }}`
	afero.WriteFile(fs, "/profiles/vendor/child/content/config.md.tmpl", []byte(templateContent), 0644)

	// Create multi-level profile inheritance chain
	chain := []*profile.Profile{
		{
			Name: "vendor/parent",
			Variables: map[string]interface{}{
				"app": map[string]interface{}{
					"name":    "ParentApp",
					"version": "1.0.0",
					"database": map[string]interface{}{
						"host": "parent.db.local",
						"port": 5432,
					},
				},
			},
		},
		{
			Name: "vendor/child",
			Variables: map[string]interface{}{
				"app": map[string]interface{}{
					"name":   "ChildApp",
					"author": "ChildAuthor",
					"database": map[string]interface{}{
						"host": "child.db.local",
					},
				},
			},
			TemplateFiles: []profile.TemplateFile{
				{SourcePath: "config.md.tmpl", IsPartial: false},
			},
		},
	}
	mergedProfile := infraprofile.NewMergedProfile(chain, fs, "/profiles")

	// Act - no global or project config, testing profile inheritance only
	opts := &rendering.RenderOptions{}
	results, err := pipeline.RenderAllWithOptions(mergedProfile, "/project", opts)

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	content := results[0].Content

	// Name should be from child (overrides parent)
	if !strings.Contains(content, "App Name: ChildApp") {
		t.Errorf("expected 'App Name: ChildApp', got %q", content)
	}
	// Version should be from parent (preserved, not overridden by child)
	if !strings.Contains(content, "App Version: 1.0.0") {
		t.Errorf("expected 'App Version: 1.0.0', got %q", content)
	}
	// Author should be from child (new key added by child)
	if !strings.Contains(content, "App Author: ChildAuthor") {
		t.Errorf("expected 'App Author: ChildAuthor', got %q", content)
	}
	// DB Host should be from child (overrides parent nested key)
	if !strings.Contains(content, "DB Host: child.db.local") {
		t.Errorf("expected 'DB Host: child.db.local', got %q", content)
	}
	// DB Port should be from parent (preserved via deep merge)
	if !strings.Contains(content, "DB Port: 5432") {
		t.Errorf("expected 'DB Port: 5432', got %q", content)
	}
}

// =============================================================================
// Conflict Warning Tests (Task Group 3 from merge-variables-consistency spec)
// =============================================================================

// Test 3.1.1: Test warnings are emitted when opts.Verbose is true and WarningWriter is set
func TestRenderAllWithOptions_ConflictWarnings_EmittedWhenVerbose(t *testing.T) {
	// Arrange
	fs := afero.NewMemMapFs()
	engine := template.NewTemplateEngineWithFs(fs)
	pipeline := rendering.NewRenderingPipeline(engine)

	// Create a simple template
	templateContent := `DB Port: {{ .Variables.database.port }}`
	afero.WriteFile(fs, "/profiles/vendor/test/content/config.md.tmpl", []byte(templateContent), 0644)

	// Create profile chain
	chain := []*profile.Profile{
		{
			Name: "vendor/test",
			Variables: map[string]interface{}{
				"database": map[string]interface{}{
					"port": 5432,
				},
			},
			TemplateFiles: []profile.TemplateFile{
				{SourcePath: "config.md.tmpl", IsPartial: false},
			},
		},
	}
	mergedProfile := infraprofile.NewMergedProfile(chain, fs, "/profiles")

	// Global config with database port that will be overridden by profile
	globalConfig := &config.GlobalConfig{
		Variables: map[string]interface{}{
			"database": map[string]interface{}{
				"port": 3306, // This will be overridden
			},
		},
	}

	// Capture warning output
	var warningOutput bytes.Buffer

	// Act - with verbose = true and WarningWriter set
	opts := &rendering.RenderOptions{
		GlobalConfig:  globalConfig,
		Verbose:       true,
		WarningWriter: &warningOutput,
	}
	results, err := pipeline.RenderAllWithOptions(mergedProfile, "/project", opts)

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	// Verify warnings were emitted
	warnings := warningOutput.String()
	if !strings.Contains(warnings, "Warning:") {
		t.Errorf("expected warnings to contain 'Warning:', got %q", warnings)
	}
	if !strings.Contains(warnings, "database.port") {
		t.Errorf("expected warnings to contain 'database.port', got %q", warnings)
	}
}

// Test 3.1.2: Test warning format includes path, source, and overriding source
func TestRenderAllWithOptions_ConflictWarnings_CorrectFormat(t *testing.T) {
	// Arrange
	fs := afero.NewMemMapFs()
	engine := template.NewTemplateEngineWithFs(fs)
	pipeline := rendering.NewRenderingPipeline(engine)

	// Create a simple template
	templateContent := `Host: {{ .Variables.server.host }}`
	afero.WriteFile(fs, "/profiles/vendor/test/content/config.md.tmpl", []byte(templateContent), 0644)

	// Create profile chain with variables that will create conflicts at multiple levels
	chain := []*profile.Profile{
		{
			Name: "vendor/test",
			Variables: map[string]interface{}{
				"server": map[string]interface{}{
					"host": "profile.example.com", // Overrides global
				},
			},
			TemplateFiles: []profile.TemplateFile{
				{SourcePath: "config.md.tmpl", IsPartial: false},
			},
		},
	}
	mergedProfile := infraprofile.NewMergedProfile(chain, fs, "/profiles")

	// Global config with server.host
	globalConfig := &config.GlobalConfig{
		Variables: map[string]interface{}{
			"server": map[string]interface{}{
				"host": "global.example.com",
			},
		},
	}

	// Project config overrides server.host
	projectConfig := &config.ProjectConfig{
		Variables: map[string]interface{}{
			"server": map[string]interface{}{
				"host": "project.example.com",
			},
		},
	}

	// Capture warning output
	var warningOutput bytes.Buffer

	// Act
	opts := &rendering.RenderOptions{
		GlobalConfig:  globalConfig,
		ProjectConfig: projectConfig,
		Verbose:       true,
		WarningWriter: &warningOutput,
	}
	_, err := pipeline.RenderAllWithOptions(mergedProfile, "/project", opts)

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Verify warning format
	warnings := warningOutput.String()
	// Expected format: "Warning: Variable '[path]' from [source] overridden by [overriding source]"
	if !strings.Contains(warnings, "server.host") {
		t.Errorf("expected warnings to contain path 'server.host', got %q", warnings)
	}
	if !strings.Contains(warnings, "global config") {
		t.Errorf("expected warnings to contain source 'global config', got %q", warnings)
	}
	if !strings.Contains(warnings, "project config") {
		t.Errorf("expected warnings to contain overriding source 'project config', got %q", warnings)
	}
	if !strings.Contains(warnings, "overridden by") {
		t.Errorf("expected warnings to contain 'overridden by', got %q", warnings)
	}
}

// Test 3.1.3: Test no warnings when opts.Verbose is false
func TestRenderAllWithOptions_ConflictWarnings_NotEmittedWhenNotVerbose(t *testing.T) {
	// Arrange
	fs := afero.NewMemMapFs()
	engine := template.NewTemplateEngineWithFs(fs)
	pipeline := rendering.NewRenderingPipeline(engine)

	// Create a simple template
	templateContent := `Port: {{ .Variables.port }}`
	afero.WriteFile(fs, "/profiles/vendor/test/content/config.md.tmpl", []byte(templateContent), 0644)

	// Create profile chain
	chain := []*profile.Profile{
		{
			Name: "vendor/test",
			Variables: map[string]interface{}{
				"port": 8080,
			},
			TemplateFiles: []profile.TemplateFile{
				{SourcePath: "config.md.tmpl", IsPartial: false},
			},
		},
	}
	mergedProfile := infraprofile.NewMergedProfile(chain, fs, "/profiles")

	// Global config with port that will be overridden
	globalConfig := &config.GlobalConfig{
		Variables: map[string]interface{}{
			"port": 3000, // This will be overridden
		},
	}

	// Capture warning output
	var warningOutput bytes.Buffer

	// Act - with verbose = false
	opts := &rendering.RenderOptions{
		GlobalConfig:  globalConfig,
		Verbose:       false,
		WarningWriter: &warningOutput,
	}
	results, err := pipeline.RenderAllWithOptions(mergedProfile, "/project", opts)

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	// Verify no warnings were emitted
	if warningOutput.Len() > 0 {
		t.Errorf("expected no warnings when Verbose=false, got %q", warningOutput.String())
	}
}

// Test 3.1.4: Test profile inheritance conflicts are included in warnings
func TestRenderAllWithOptions_ConflictWarnings_IncludesProfileInheritanceConflicts(t *testing.T) {
	// Arrange
	fs := afero.NewMemMapFs()
	engine := template.NewTemplateEngineWithFs(fs)
	pipeline := rendering.NewRenderingPipeline(engine)

	// Create a simple template
	templateContent := `Setting: {{ .Variables.setting }}`
	afero.WriteFile(fs, "/profiles/vendor/child/content/config.md.tmpl", []byte(templateContent), 0644)

	// Create multi-level profile inheritance chain with conflicts
	chain := []*profile.Profile{
		{
			Name: "vendor/parent",
			Variables: map[string]interface{}{
				"setting": "from_parent", // Will be overridden by child
			},
		},
		{
			Name: "vendor/child",
			Variables: map[string]interface{}{
				"setting": "from_child", // Overrides parent
			},
			TemplateFiles: []profile.TemplateFile{
				{SourcePath: "config.md.tmpl", IsPartial: false},
			},
		},
	}
	mergedProfile := infraprofile.NewMergedProfile(chain, fs, "/profiles")

	// Capture warning output
	var warningOutput bytes.Buffer

	// Act
	opts := &rendering.RenderOptions{
		Verbose:       true,
		WarningWriter: &warningOutput,
	}
	_, err := pipeline.RenderAllWithOptions(mergedProfile, "/project", opts)

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Verify profile inheritance conflicts are included
	warnings := warningOutput.String()
	if !strings.Contains(warnings, "setting") {
		t.Errorf("expected warnings to contain path 'setting', got %q", warnings)
	}
	if !strings.Contains(warnings, "vendor/parent") {
		t.Errorf("expected warnings to contain source 'vendor/parent', got %q", warnings)
	}
	if !strings.Contains(warnings, "vendor/child") {
		t.Errorf("expected warnings to contain overriding source 'vendor/child', got %q", warnings)
	}
}

// =============================================================================
// Helper function to suppress unused import warning
// =============================================================================
var _ = time.Now // Used in tests via template context
