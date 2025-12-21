package template_test

import (
	"testing"
	"time"

	"github.com/simensen/weftlo/internal/domain/config"
	"github.com/simensen/weftlo/internal/domain/profile"
	"github.com/simensen/weftlo/internal/domain/template"
)

// Test 1: NewTemplateContext creates context with all required fields populated
func TestNewTemplateContext_CreatesContextWithAllFields(t *testing.T) {
	t.Parallel()
	// Arrange
	testProfile := &profile.Profile{
		Name:         "vendor/test-profile",
		InheritsFrom: "vendor/base-profile",
		TemplateFiles: []profile.TemplateFile{
			{SourcePath: "template.md", TargetPath: "output.md", IsPartial: false},
		},
	}
	projectDir := "/path/to/project"
	generatedAt := time.Date(2025, 12, 20, 10, 30, 0, 0, time.UTC)
	projectConfig := &config.ProjectConfig{Profiles: []string{"vendor/test-profile"}}
	globalConfig := &config.GlobalConfig{DefaultProfile: "vendor/default"}
	variables := map[string]interface{}{"key": "value"}

	// Act
	ctx := template.NewTemplateContext(testProfile, projectDir, generatedAt, projectConfig, globalConfig, variables)

	// Assert
	if ctx == nil {
		t.Fatal("expected NewTemplateContext to return non-nil context")
	}
	if ctx.Profile != testProfile {
		t.Errorf("expected Profile to be %v, got %v", testProfile, ctx.Profile)
	}
	if ctx.ProjectDir != projectDir {
		t.Errorf("expected ProjectDir to be %q, got %q", projectDir, ctx.ProjectDir)
	}
	if !ctx.GeneratedAt.Equal(generatedAt) {
		t.Errorf("expected GeneratedAt to be %v, got %v", generatedAt, ctx.GeneratedAt)
	}
	if ctx.ProjectConfig != projectConfig {
		t.Errorf("expected ProjectConfig to be %v, got %v", projectConfig, ctx.ProjectConfig)
	}
	if ctx.GlobalConfig != globalConfig {
		t.Errorf("expected GlobalConfig to be %v, got %v", globalConfig, ctx.GlobalConfig)
	}
	if ctx.Variables["key"] != "value" {
		t.Errorf("expected Variables[key] to be 'value', got %v", ctx.Variables["key"])
	}
}

// Test 2: All fields are exported and accessible for template binding
func TestTemplateContext_FieldsAreExported(t *testing.T) {
	t.Parallel()
	// Arrange
	testProfile := &profile.Profile{Name: "vendor/test"}
	projectDir := "/project"
	generatedAt := time.Now()
	projectConfig := &config.ProjectConfig{Profiles: []string{"vendor/test"}}
	globalConfig := &config.GlobalConfig{DefaultProfile: "vendor/default"}
	variables := map[string]interface{}{"key": "value"}

	ctx := template.NewTemplateContext(testProfile, projectDir, generatedAt, projectConfig, globalConfig, variables)

	// Assert - verify all fields are accessible (would fail at compile time if not exported)
	_ = ctx.Profile
	_ = ctx.ProjectDir
	_ = ctx.GeneratedAt
	_ = ctx.ProjectConfig
	_ = ctx.GlobalConfig
	_ = ctx.Variables

	// Also verify we can access nested fields (important for template binding)
	_ = ctx.Profile.Name
	_ = ctx.Profile.InheritsFrom
	_ = ctx.Profile.TemplateFiles
	_ = ctx.ProjectConfig.Profiles[0]
	_ = ctx.GlobalConfig.DefaultProfile
	_ = ctx.Variables["key"]

	// If we got here without compile errors, the test passes
}

// Test 3: Generation timestamp is correctly assigned from parameter (not generated internally)
func TestNewTemplateContext_TimestampFromParameter(t *testing.T) {
	t.Parallel()
	// Arrange - use a specific timestamp in the past to verify it's not generated
	specificTime := time.Date(2020, 1, 1, 12, 0, 0, 0, time.UTC)
	testProfile := &profile.Profile{Name: "vendor/test"}
	projectConfig := &config.ProjectConfig{Profiles: []string{"vendor/test"}}
	globalConfig := &config.GlobalConfig{}

	// Act
	ctx := template.NewTemplateContext(testProfile, "/project", specificTime, projectConfig, globalConfig, nil)

	// Assert - timestamp should be exactly what we passed, not current time
	if !ctx.GeneratedAt.Equal(specificTime) {
		t.Errorf("expected GeneratedAt to be exactly %v (parameter value), got %v", specificTime, ctx.GeneratedAt)
	}

	// Verify it's definitely not close to current time (would indicate internal generation)
	now := time.Now()
	diff := now.Sub(ctx.GeneratedAt)
	if diff < time.Hour*24*365 { // Should be years in the past
		t.Error("GeneratedAt appears to be close to current time, suggesting it may be generated internally")
	}
}

// Test 4: Nil Profile is handled appropriately (stored as nil)
func TestNewTemplateContext_NilProfile(t *testing.T) {
	t.Parallel()
	// Arrange
	generatedAt := time.Now()
	projectConfig := &config.ProjectConfig{Profiles: []string{"vendor/test"}}
	globalConfig := &config.GlobalConfig{}

	// Act
	ctx := template.NewTemplateContext(nil, "/project", generatedAt, projectConfig, globalConfig, nil)

	// Assert - nil profile should be stored as nil
	if ctx.Profile != nil {
		t.Errorf("expected Profile to be nil when nil is passed, got %v", ctx.Profile)
	}
}

// Test 5: Nil configs are handled appropriately (stored as nil)
func TestNewTemplateContext_NilConfigs(t *testing.T) {
	t.Parallel()
	// Arrange
	testProfile := &profile.Profile{Name: "vendor/test"}
	generatedAt := time.Now()

	// Act
	ctx := template.NewTemplateContext(testProfile, "/project", generatedAt, nil, nil, nil)

	// Assert - nil configs should be stored as nil
	if ctx.ProjectConfig != nil {
		t.Errorf("expected ProjectConfig to be nil when nil is passed, got %v", ctx.ProjectConfig)
	}
	if ctx.GlobalConfig != nil {
		t.Errorf("expected GlobalConfig to be nil when nil is passed, got %v", ctx.GlobalConfig)
	}
	if ctx.Variables != nil {
		t.Errorf("expected Variables to be nil when nil is passed, got %v", ctx.Variables)
	}
}

// Test 6: Empty project directory is handled appropriately
func TestNewTemplateContext_EmptyProjectDir(t *testing.T) {
	t.Parallel()
	// Arrange
	testProfile := &profile.Profile{Name: "vendor/test"}
	generatedAt := time.Now()
	projectConfig := &config.ProjectConfig{Profiles: []string{"vendor/test"}}
	globalConfig := &config.GlobalConfig{}

	// Act
	ctx := template.NewTemplateContext(testProfile, "", generatedAt, projectConfig, globalConfig, nil)

	// Assert - empty string should be stored as empty string
	if ctx.ProjectDir != "" {
		t.Errorf("expected ProjectDir to be empty string when empty is passed, got %q", ctx.ProjectDir)
	}
}

// =============================================================================
// Template Variables Tests (Task Group 2)
// =============================================================================

// Test 7: Variables field is accessible in TemplateContext
func TestTemplateContext_VariablesFieldAccessible(t *testing.T) {
	t.Parallel()
	// Arrange
	variables := map[string]interface{}{
		"company":     "Acme Corp",
		"environment": "production",
	}

	// Act
	ctx := template.NewTemplateContext(nil, "/project", time.Now(), nil, nil, variables)

	// Assert
	if ctx.Variables == nil {
		t.Fatal("expected Variables to be non-nil")
	}
	if ctx.Variables["company"] != "Acme Corp" {
		t.Errorf("expected company to be 'Acme Corp', got %v", ctx.Variables["company"])
	}
	if ctx.Variables["environment"] != "production" {
		t.Errorf("expected environment to be 'production', got %v", ctx.Variables["environment"])
	}
}

// Test 8: Nested variables are accessible
func TestTemplateContext_NestedVariablesAccessible(t *testing.T) {
	t.Parallel()
	// Arrange
	variables := map[string]interface{}{
		"database": map[string]interface{}{
			"host": "localhost",
			"port": 5432,
		},
	}

	// Act
	ctx := template.NewTemplateContext(nil, "/project", time.Now(), nil, nil, variables)

	// Assert
	if ctx.Variables == nil {
		t.Fatal("expected Variables to be non-nil")
	}

	db, ok := ctx.Variables["database"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected database to be a map, got %T", ctx.Variables["database"])
	}
	if db["host"] != "localhost" {
		t.Errorf("expected database.host to be 'localhost', got %v", db["host"])
	}
	if db["port"] != 5432 {
		t.Errorf("expected database.port to be 5432, got %v", db["port"])
	}
}

// Test 9: Template rendering can access Variables
func TestTemplateContext_TemplateRenderingAccessesVariables(t *testing.T) {
	t.Parallel()
	// Arrange
	variables := map[string]interface{}{
		"greeting": "Hello",
		"name":     "World",
	}
	ctx := template.NewTemplateContext(nil, "/project", time.Now(), nil, nil, variables)
	engine := template.NewTemplateEngine()

	// Act
	templateContent := "{{ .Variables.greeting }}, {{ .Variables.name }}!"
	result, err := engine.Render(templateContent, ctx)

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result != "Hello, World!" {
		t.Errorf("expected 'Hello, World!', got %q", result)
	}
}

// Test 10: Template rendering can access nested Variables
func TestTemplateContext_TemplateRenderingAccessesNestedVariables(t *testing.T) {
	t.Parallel()
	// Arrange
	variables := map[string]interface{}{
		"database": map[string]interface{}{
			"host": "localhost",
			"port": 5432,
		},
	}
	ctx := template.NewTemplateContext(nil, "/project", time.Now(), nil, nil, variables)
	engine := template.NewTemplateEngine()

	// Act
	templateContent := "Host: {{ index .Variables.database \"host\" }}, Port: {{ index .Variables.database \"port\" }}"
	result, err := engine.Render(templateContent, ctx)

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result != "Host: localhost, Port: 5432" {
		t.Errorf("expected 'Host: localhost, Port: 5432', got %q", result)
	}
}

// Test 11: Template rendering with nil Variables does not panic
func TestTemplateContext_TemplateRenderingWithNilVariables(t *testing.T) {
	t.Parallel()
	// Arrange
	ctx := template.NewTemplateContext(nil, "/project", time.Now(), nil, nil, nil)
	engine := template.NewTemplateEngine()

	// Act - accessing nil Variables should be handled gracefully
	templateContent := "{{ if .Variables }}Has variables{{ else }}No variables{{ end }}"
	result, err := engine.Render(templateContent, ctx)

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result != "No variables" {
		t.Errorf("expected 'No variables', got %q", result)
	}
}
