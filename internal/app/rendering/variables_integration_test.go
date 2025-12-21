package rendering_test

import (
	"strings"
	"testing"
	"time"

	"github.com/spf13/afero"

	"github.com/simensen/weftlo/internal/app/rendering"
	"github.com/simensen/weftlo/internal/domain/config"
	"github.com/simensen/weftlo/internal/domain/profile"
	"github.com/simensen/weftlo/internal/domain/template"
	infraprofile "github.com/simensen/weftlo/internal/infrastructure/profile"
)

// =============================================================================
// Integration Tests for Template Variables (Task Group 4)
// =============================================================================

// Integration Test: End-to-end template rendering with variables
// This tests the full workflow where a template uses variables from
// global, profile (via profile chain), and project configs with proper precedence.
//
// Note: Profile variables should be defined in the profile chain (profile.Variables),
// not via opts.ProfileConfig.Variables. The mergedProfile.Variables field contains
// the deep-merged variables from the entire profile inheritance chain.
func TestRenderAllWithOptions_VariablePrecedence(t *testing.T) {
	t.Parallel()
	// Arrange
	fs := afero.NewMemMapFs()

	// Create a template that uses variables
	templateContent := `Company: {{ .Variables.company }}
Environment: {{ .Variables.env }}
Version: {{ .Variables.version }}
Project: {{ .Variables.project_name }}
`
	afero.WriteFile(fs, "/profiles/test/test/content/config.md.tmpl", []byte(templateContent), 0644)

	// Create merged profile with one template and profile-level variables
	// Profile variables are now accessed via mergedProfile.Variables, which comes from
	// the profile chain, not from opts.ProfileConfig
	chain := []*profile.Profile{
		{
			Name: "test/test",
			Variables: map[string]interface{}{
				"env":     "staging",
				"version": "1.0.0",
			},
			TemplateFiles: []profile.TemplateFile{
				{SourcePath: "config.md.tmpl", IsPartial: false},
			},
		},
	}
	mergedProfile := infraprofile.NewMergedProfile(chain, fs, "/profiles")

	// Create configs with variables at each level
	globalConfig := &config.GlobalConfig{
		Variables: map[string]interface{}{
			"company": "Acme Corp",
			"env":     "development",
		},
	}

	projectConfig := &config.ProjectConfig{
		Profiles: []string{"test/test"},
		Variables: map[string]interface{}{
			"env":          "production",
			"project_name": "my-awesome-project",
		},
	}

	// Create pipeline
	engine := template.NewTemplateEngineWithFs(fs)
	pipeline := rendering.NewRenderingPipeline(engine)

	// Act
	results, err := pipeline.RenderAllWithOptions(mergedProfile, "/project", &rendering.RenderOptions{
		GlobalConfig:  globalConfig,
		ProjectConfig: projectConfig,
	})

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	result := results[0]
	expected := `Company: Acme Corp
Environment: production
Version: 1.0.0
Project: my-awesome-project
`

	if result.Content != expected {
		t.Errorf("expected:\n%s\ngot:\n%s", expected, result.Content)
	}
}

// Integration Test: Template rendering with nested variables
func TestRenderAllWithOptions_NestedVariables(t *testing.T) {
	t.Parallel()
	// Arrange
	fs := afero.NewMemMapFs()

	// Create a template that uses nested variables
	// Note: Go templates require index for nested map access
	templateContent := `Database Host: {{ index .Variables.database "host" }}
Database Port: {{ index .Variables.database "port" }}
`
	afero.WriteFile(fs, "/profiles/test/test/content/db.md.tmpl", []byte(templateContent), 0644)

	// Create merged profile
	chain := []*profile.Profile{
		{
			Name: "test/test",
			TemplateFiles: []profile.TemplateFile{
				{SourcePath: "db.md.tmpl", IsPartial: false},
			},
		},
	}
	mergedProfile := infraprofile.NewMergedProfile(chain, fs, "/profiles")

	// Create config with nested variables
	projectConfig := &config.ProjectConfig{
		Profiles: []string{"test/test"},
		Variables: map[string]interface{}{
			"database": map[string]interface{}{
				"host": "localhost",
				"port": 5432,
			},
		},
	}

	// Create pipeline
	engine := template.NewTemplateEngineWithFs(fs)
	pipeline := rendering.NewRenderingPipeline(engine)

	// Act
	results, err := pipeline.RenderAllWithOptions(mergedProfile, "/project", &rendering.RenderOptions{
		ProjectConfig: projectConfig,
	})

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	result := results[0]
	expected := `Database Host: localhost
Database Port: 5432
`

	if result.Content != expected {
		t.Errorf("expected:\n%s\ngot:\n%s", expected, result.Content)
	}
}

// Integration Test: Template rendering with no variables (nil/empty)
func TestRenderAllWithOptions_NoVariables(t *testing.T) {
	t.Parallel()
	// Arrange
	fs := afero.NewMemMapFs()

	// Create a template that conditionally uses variables
	// Note: Empty map evaluates to false in Go templates
	templateContent := `{{ if .Variables }}Has Variables{{ else }}No Variables{{ end }}
`
	afero.WriteFile(fs, "/profiles/test/test/content/check.md.tmpl", []byte(templateContent), 0644)

	// Create merged profile
	chain := []*profile.Profile{
		{
			Name: "test/test",
			TemplateFiles: []profile.TemplateFile{
				{SourcePath: "check.md.tmpl", IsPartial: false},
			},
		},
	}
	mergedProfile := infraprofile.NewMergedProfile(chain, fs, "/profiles")

	// Create pipeline
	engine := template.NewTemplateEngineWithFs(fs)
	pipeline := rendering.NewRenderingPipeline(engine)

	// Act - no configs provided
	results, err := pipeline.RenderAllWithOptions(mergedProfile, "/project", nil)

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	result := results[0]
	// MergeVariables returns empty map (not nil)
	// Empty map evaluates to false in Go templates
	expected := `No Variables
`

	if result.Content != expected {
		t.Errorf("expected:\n%s\ngot:\n%s", expected, result.Content)
	}
}

// Integration Test: Variables work with include function
func TestRenderAllWithOptions_VariablesWithInclude(t *testing.T) {
	t.Parallel()
	// Arrange
	fs := afero.NewMemMapFs()

	// Create a partial that uses variables
	partialContent := `Welcome to {{ .Variables.company }}!`
	afero.WriteFile(fs, "/profiles/test/test/content/_partials/header.md", []byte(partialContent), 0644)

	// Create main template that includes the partial
	mainContent := `{{ include "_partials/header.md" }}
Environment: {{ .Variables.env }}
`
	afero.WriteFile(fs, "/profiles/test/test/content/main.md.tmpl", []byte(mainContent), 0644)

	// Create merged profile
	chain := []*profile.Profile{
		{
			Name: "test/test",
			TemplateFiles: []profile.TemplateFile{
				{SourcePath: "main.md.tmpl", IsPartial: false},
				{SourcePath: "_partials/header.md", IsPartial: true},
			},
		},
	}
	mergedProfile := infraprofile.NewMergedProfile(chain, fs, "/profiles")

	// Create config with variables
	projectConfig := &config.ProjectConfig{
		Profiles: []string{"test/test"},
		Variables: map[string]interface{}{
			"company": "Acme Corp",
			"env":     "production",
		},
	}

	// Create pipeline
	engine := template.NewTemplateEngineWithFs(fs)
	pipeline := rendering.NewRenderingPipeline(engine)

	// Act
	results, err := pipeline.RenderAllWithOptions(mergedProfile, "/project", &rendering.RenderOptions{
		ProjectConfig: projectConfig,
	})

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	result := results[0]
	expected := `Welcome to Acme Corp!
Environment: production
`

	if result.Content != expected {
		t.Errorf("expected:\n%s\ngot:\n%s", expected, result.Content)
	}
}

// Integration Test: Template rendering preserves backward compatibility
// RenderAll (deprecated) should still work without variables
func TestRenderAll_BackwardCompatibility(t *testing.T) {
	t.Parallel()
	// Arrange
	fs := afero.NewMemMapFs()

	// Create a simple template that doesn't use variables
	templateContent := `Profile: {{ .Profile.Name }}
Generated: {{ .GeneratedAt.Format "2006-01-02" }}
`
	afero.WriteFile(fs, "/profiles/test/test/content/simple.md.tmpl", []byte(templateContent), 0644)

	// Create merged profile
	chain := []*profile.Profile{
		{
			Name: "test/test",
			TemplateFiles: []profile.TemplateFile{
				{SourcePath: "simple.md.tmpl", IsPartial: false},
			},
		},
	}
	mergedProfile := infraprofile.NewMergedProfile(chain, fs, "/profiles")

	// Create pipeline
	engine := template.NewTemplateEngineWithFs(fs)
	pipeline := rendering.NewRenderingPipeline(engine)

	// Act - use the deprecated RenderAll method
	results, err := pipeline.RenderAll(mergedProfile, "/project", nil)

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	result := results[0]

	// Check that the profile name is rendered correctly
	if !strings.Contains(result.Content, "test/test") {
		t.Errorf("expected content to contain 'test/test', got:\n%s", result.Content)
	}

	// Check that the date is rendered (format: YYYY-MM-DD)
	today := time.Now().Format("2006-01-02")
	if !strings.Contains(result.Content, today) {
		t.Errorf("expected content to contain today's date '%s', got:\n%s", today, result.Content)
	}
}

// Integration Test: Variables with special characters in keys
func TestRenderAllWithOptions_VariablesWithSpecialKeys(t *testing.T) {
	t.Parallel()
	// Arrange
	fs := afero.NewMemMapFs()

	// Create a template that uses variables with special keys
	templateContent := `API URL: {{ index .Variables "api_url" }}
DB Name: {{ index .Variables "database-name" }}
`
	afero.WriteFile(fs, "/profiles/test/test/content/special.md.tmpl", []byte(templateContent), 0644)

	// Create merged profile
	chain := []*profile.Profile{
		{
			Name: "test/test",
			TemplateFiles: []profile.TemplateFile{
				{SourcePath: "special.md.tmpl", IsPartial: false},
			},
		},
	}
	mergedProfile := infraprofile.NewMergedProfile(chain, fs, "/profiles")

	// Create config with variables that have special keys
	projectConfig := &config.ProjectConfig{
		Profiles: []string{"test/test"},
		Variables: map[string]interface{}{
			"api_url":       "https://api.example.com",
			"database-name": "my_database",
		},
	}

	// Create pipeline
	engine := template.NewTemplateEngineWithFs(fs)
	pipeline := rendering.NewRenderingPipeline(engine)

	// Act
	results, err := pipeline.RenderAllWithOptions(mergedProfile, "/project", &rendering.RenderOptions{
		ProjectConfig: projectConfig,
	})

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	result := results[0]
	expected := `API URL: https://api.example.com
DB Name: my_database
`

	if result.Content != expected {
		t.Errorf("expected:\n%s\ngot:\n%s", expected, result.Content)
	}
}
