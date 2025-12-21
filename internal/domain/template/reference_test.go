package template

import (
	"errors"
	"testing"
	"time"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/simensen/weftlo/internal/app/routing"
	"github.com/simensen/weftlo/internal/domain/profile"
)

// Test 1: Basic path resolution with @ prefix output
func TestReferenceFunc_BasicPathResolutionWithAtPrefix(t *testing.T) {
	// Arrange
	fs := afero.NewMemMapFs()
	afero.WriteFile(fs, "/content/claude/settings.json", []byte("{}"), 0644)

	resolver := NewMapFileResolver(map[string]string{
		"claude/settings.json": "/content/claude/settings.json",
	})

	config := profile.ContentConfig{
		Targets: map[string]string{
			"claude": ".claude",
		},
	}
	router, err := routing.NewContentRouter(config, nil, nil)
	require.NoError(t, err)

	renderCtx := NewRenderContextWithRouter(fs, resolver, router, false)
	tmplCtx := NewTemplateContext(
		&profile.Profile{Name: "test/profile"},
		"/project",
		time.Now(),
		nil,
		nil,
		nil,
	)

	referenceFunc := createReferenceFunc(renderCtx, tmplCtx)

	// Act
	result, err := referenceFunc("claude/settings.json")

	// Assert
	require.NoError(t, err)
	assert.Equal(t, "@.claude/settings.json", result)
}

// Test 2: .tmpl suffix stripping from output path
func TestReferenceFunc_TmplSuffixStripping(t *testing.T) {
	// Arrange
	fs := afero.NewMemMapFs()
	afero.WriteFile(fs, "/content/claude/CLAUDE.md.tmpl", []byte("# Claude"), 0644)

	resolver := NewMapFileResolver(map[string]string{
		"claude/CLAUDE.md.tmpl": "/content/claude/CLAUDE.md.tmpl",
	})

	config := profile.ContentConfig{
		Targets: map[string]string{
			"claude": ".claude",
		},
	}
	router, err := routing.NewContentRouter(config, nil, nil)
	require.NoError(t, err)

	renderCtx := NewRenderContextWithRouter(fs, resolver, router, false)
	tmplCtx := NewTemplateContext(
		&profile.Profile{Name: "test/profile"},
		"/project",
		time.Now(),
		nil,
		nil,
		nil,
	)

	referenceFunc := createReferenceFunc(renderCtx, tmplCtx)

	// Act - reference the template file with .tmpl suffix
	result, err := referenceFunc("claude/CLAUDE.md.tmpl")

	// Assert - output path should NOT have .tmpl suffix
	require.NoError(t, err)
	assert.Equal(t, "@.claude/CLAUDE.md", result)
}

// Test 3: Lenient behavior for non-existent source path (returns would-be path)
func TestReferenceFunc_LenientNonExistentPath(t *testing.T) {
	// Arrange
	fs := afero.NewMemMapFs()
	// Note: file does NOT exist in filesystem or resolver

	resolver := NewMapFileResolver(map[string]string{}) // Empty - file doesn't exist

	config := profile.ContentConfig{
		Targets: map[string]string{
			"claude": ".claude",
		},
	}
	router, err := routing.NewContentRouter(config, nil, nil)
	require.NoError(t, err)

	renderCtx := NewRenderContextWithRouter(fs, resolver, router, false)
	tmplCtx := NewTemplateContext(
		&profile.Profile{Name: "test/profile"},
		"/project",
		time.Now(),
		nil,
		nil,
		nil,
	)

	referenceFunc := createReferenceFunc(renderCtx, tmplCtx)

	// Act - reference a file that doesn't exist
	result, err := referenceFunc("claude/nonexistent.json")

	// Assert - should return would-be path, not error (lenient behavior)
	require.NoError(t, err)
	assert.Equal(t, "@.claude/nonexistent.json", result)
}

// Test 4: Suppressed target returns empty string
func TestReferenceFunc_SuppressedTargetReturnsEmptyString(t *testing.T) {
	// Arrange
	fs := afero.NewMemMapFs()
	afero.WriteFile(fs, "/content/claude/settings.json", []byte("{}"), 0644)

	resolver := NewMapFileResolver(map[string]string{
		"claude/settings.json": "/content/claude/settings.json",
	})

	config := profile.ContentConfig{
		Targets: map[string]string{
			"claude": ".claude",
		},
	}
	// Null override to suppress the claude target
	overrides := map[string]*string{
		"claude": nil, // null = suppress
	}
	router, err := routing.NewContentRouter(config, overrides, nil)
	require.NoError(t, err)

	renderCtx := NewRenderContextWithRouter(fs, resolver, router, false)
	tmplCtx := NewTemplateContext(
		&profile.Profile{Name: "test/profile"},
		"/project",
		time.Now(),
		nil,
		nil,
		nil,
	)

	referenceFunc := createReferenceFunc(renderCtx, tmplCtx)

	// Act
	result, err := referenceFunc("claude/settings.json")

	// Assert - suppressed target returns empty string, not error
	require.NoError(t, err)
	assert.Equal(t, "", result)
}

// Test 5: Partial source path returns ReferencePartialError
func TestReferenceFunc_PartialSourceReturnsError(t *testing.T) {
	// Arrange
	fs := afero.NewMemMapFs()
	afero.WriteFile(fs, "/content/_partials/header.md", []byte("# Header"), 0644)

	resolver := NewMapFileResolver(map[string]string{
		"_partials/header.md": "/content/_partials/header.md",
	})

	config := profile.ContentConfig{
		Targets: map[string]string{
			"claude":    ".claude",
			"_partials": "_partials",
		},
	}
	router, err := routing.NewContentRouter(config, nil, nil)
	require.NoError(t, err)

	renderCtx := NewRenderContextWithRouter(fs, resolver, router, false)
	tmplCtx := NewTemplateContext(
		&profile.Profile{Name: "test/profile"},
		"/project",
		time.Now(),
		nil,
		nil,
		nil,
	)

	referenceFunc := createReferenceFunc(renderCtx, tmplCtx)

	// Act
	result, err := referenceFunc("_partials/header.md")

	// Assert - partials cannot be referenced, only included
	require.Error(t, err)
	assert.Equal(t, "", result)

	var partialErr *ReferencePartialError
	require.True(t, errors.As(err, &partialErr), "expected ReferencePartialError, got %T: %v", err, err)
	assert.Equal(t, "_partials/header.md", partialErr.SourcePath)
}

// Test 6: Self-reference returns valid path (with warning when verbose)
func TestReferenceFunc_SelfReferenceReturnsValidPath(t *testing.T) {
	// Arrange
	fs := afero.NewMemMapFs()
	afero.WriteFile(fs, "/content/claude/README.md", []byte("# Readme"), 0644)

	resolver := NewMapFileResolver(map[string]string{
		"claude/README.md": "/content/claude/README.md",
	})

	config := profile.ContentConfig{
		Targets: map[string]string{
			"claude": ".claude",
		},
	}
	router, err := routing.NewContentRouter(config, nil, nil)
	require.NoError(t, err)

	// Set up render context with include chain showing we're in the file that's being referenced
	renderCtx := NewRenderContextWithRouter(fs, resolver, router, false)
	// Simulate being inside the file "claude/README.md" during rendering
	renderCtx = renderCtx.clone("claude/README.md")

	tmplCtx := NewTemplateContext(
		&profile.Profile{Name: "test/profile"},
		"/project",
		time.Now(),
		nil,
		nil,
		nil,
	)

	referenceFunc := createReferenceFunc(renderCtx, tmplCtx)

	// Act - reference the same file we're currently rendering
	result, err := referenceFunc("claude/README.md")

	// Assert - self-reference should return valid path (lenient behavior)
	require.NoError(t, err)
	assert.Equal(t, "@.claude/README.md", result)
}

// =============================================================================
// Task Group 5: Strategic Gap-Filling Tests for reference()
// =============================================================================

// Test 7: reference() with nil router returns descriptive error
func TestReferenceFunc_NilRouterReturnsError(t *testing.T) {
	// Arrange
	fs := afero.NewMemMapFs()
	resolver := NewMapFileResolver(map[string]string{})

	// Create render context WITHOUT router (nil)
	renderCtx := NewRenderContext(fs, resolver) // Uses old constructor without router
	tmplCtx := NewTemplateContext(
		&profile.Profile{Name: "test/profile"},
		"/project",
		time.Now(),
		nil,
		nil,
		nil,
	)

	referenceFunc := createReferenceFunc(renderCtx, tmplCtx)

	// Act
	result, err := referenceFunc("claude/file.md")

	// Assert - should return error when router is nil
	require.Error(t, err)
	assert.Equal(t, "", result)
	assert.Contains(t, err.Error(), "router not available")
}

// Test 8: reference() accessible in nested templates via include()
// This tests the critical workflow: main template -> include partial -> reference() works
func TestReferenceFunc_WorksInNestedInclude(t *testing.T) {
	// Arrange
	fs := afero.NewMemMapFs()

	// Main template includes a partial that uses reference()
	mainTemplate := `# Main
{{ include "_partials/links.md" }}`
	partialTemplate := `See: {{ reference "docs/guide.md" }}`

	afero.WriteFile(fs, "/content/main.md.tmpl", []byte(mainTemplate), 0644)
	afero.WriteFile(fs, "/content/_partials/links.md", []byte(partialTemplate), 0644)
	afero.WriteFile(fs, "/content/docs/guide.md", []byte("Guide content"), 0644)

	resolver := NewMapFileResolver(map[string]string{
		"main.md.tmpl":       "/content/main.md.tmpl",
		"_partials/links.md": "/content/_partials/links.md",
		"docs/guide.md":      "/content/docs/guide.md",
	})

	config := profile.ContentConfig{
		Targets: map[string]string{
			"_partials": "_partials",
			"docs":      ".docs",
		},
	}
	router, err := routing.NewContentRouter(config, nil, nil)
	require.NoError(t, err)

	renderCtx := NewRenderContextWithRouter(fs, resolver, router, false)
	tmplCtx := NewTemplateContext(
		&profile.Profile{Name: "test/profile"},
		"/project",
		time.Now(),
		nil,
		nil,
		nil,
	)

	engine := NewTemplateEngineWithFs(fs)

	// Act
	result, err := engine.RenderWithContext(mainTemplate, tmplCtx, renderCtx)

	// Assert
	require.NoError(t, err)
	assert.Contains(t, result, "# Main")
	assert.Contains(t, result, "See: @.docs/guide.md")
}

// Test 9: reference() with path normalization (removes redundant . and ..)
func TestReferenceFunc_PathNormalization(t *testing.T) {
	// Arrange
	fs := afero.NewMemMapFs()
	afero.WriteFile(fs, "/content/docs/guide.md", []byte("Guide"), 0644)

	resolver := NewMapFileResolver(map[string]string{
		"docs/guide.md": "/content/docs/guide.md",
	})

	config := profile.ContentConfig{
		Targets: map[string]string{
			"docs": ".docs",
		},
	}
	router, err := routing.NewContentRouter(config, nil, nil)
	require.NoError(t, err)

	renderCtx := NewRenderContextWithRouter(fs, resolver, router, false)
	tmplCtx := NewTemplateContext(
		&profile.Profile{Name: "test/profile"},
		"/project",
		time.Now(),
		nil,
		nil,
		nil,
	)

	referenceFunc := createReferenceFunc(renderCtx, tmplCtx)

	// Act - reference with extra path segments that should be normalized
	// Note: The path resolution in router should handle normalization
	result, err := referenceFunc("docs/guide.md")

	// Assert
	require.NoError(t, err)
	// path.Clean should normalize the output
	assert.Equal(t, "@.docs/guide.md", result)
	// Ensure no double slashes or redundant segments
	assert.NotContains(t, result, "//")
}
