package template

import (
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/simensen/weftlo/internal/app/routing"
	"github.com/simensen/weftlo/internal/domain/profile"
)

// Test 1: Glob pattern matching returns multiple @ prefixed paths
func TestReferenceGlobFunc_GlobPatternMatchingReturnsAtPrefixedPaths(t *testing.T) {
	// Arrange
	fs := afero.NewMemMapFs()
	afero.WriteFile(fs, "/content/claude/file1.md", []byte("# File 1"), 0644)
	afero.WriteFile(fs, "/content/claude/file2.md", []byte("# File 2"), 0644)
	afero.WriteFile(fs, "/content/claude/file3.md", []byte("# File 3"), 0644)

	resolver := NewMapFileResolver(map[string]string{
		"claude/file1.md": "/content/claude/file1.md",
		"claude/file2.md": "/content/claude/file2.md",
		"claude/file3.md": "/content/claude/file3.md",
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

	referenceGlobFunc := createReferenceGlobFunc(renderCtx, tmplCtx)

	// Act
	results, err := referenceGlobFunc("claude/*.md")

	// Assert
	require.NoError(t, err)
	assert.Len(t, results, 3)
	// All results should have @ prefix
	for _, result := range results {
		assert.True(t, len(result) > 0 && result[0] == '@', "result should have @ prefix: %s", result)
	}
	assert.Contains(t, results, "@.claude/file1.md")
	assert.Contains(t, results, "@.claude/file2.md")
	assert.Contains(t, results, "@.claude/file3.md")
}

// Test 2: Results sorted alphabetically
func TestReferenceGlobFunc_ResultsSortedAlphabetically(t *testing.T) {
	// Arrange
	fs := afero.NewMemMapFs()
	afero.WriteFile(fs, "/content/docs/Zebra.md", []byte("Zebra"), 0644)
	afero.WriteFile(fs, "/content/docs/Alpha.md", []byte("Alpha"), 0644)
	afero.WriteFile(fs, "/content/docs/beta.md", []byte("beta"), 0644)
	afero.WriteFile(fs, "/content/docs/Charlie.md", []byte("Charlie"), 0644)

	resolver := NewMapFileResolver(map[string]string{
		"docs/Zebra.md":   "/content/docs/Zebra.md",
		"docs/Alpha.md":   "/content/docs/Alpha.md",
		"docs/beta.md":    "/content/docs/beta.md",
		"docs/Charlie.md": "/content/docs/Charlie.md",
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

	referenceGlobFunc := createReferenceGlobFunc(renderCtx, tmplCtx)

	// Act
	results, err := referenceGlobFunc("docs/*.md")

	// Assert
	require.NoError(t, err)
	assert.Len(t, results, 4)
	// Case-sensitive alphabetical order: A-Z (65-90) comes before a-z (97-122)
	// So: Alpha, Charlie, Zebra, beta
	assert.Equal(t, "@.docs/Alpha.md", results[0])
	assert.Equal(t, "@.docs/Charlie.md", results[1])
	assert.Equal(t, "@.docs/Zebra.md", results[2])
	assert.Equal(t, "@.docs/beta.md", results[3])
}

// Test 3: Empty slice returned for no matches (not error)
func TestReferenceGlobFunc_EmptySliceForNoMatches(t *testing.T) {
	// Arrange
	fs := afero.NewMemMapFs()
	afero.WriteFile(fs, "/content/other/file.txt", []byte("content"), 0644)

	resolver := NewMapFileResolver(map[string]string{
		"other/file.txt": "/content/other/file.txt",
	})

	config := profile.ContentConfig{
		Targets: map[string]string{
			"claude": ".claude",
			"other":  ".other",
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

	referenceGlobFunc := createReferenceGlobFunc(renderCtx, tmplCtx)

	// Act - pattern that doesn't match any files
	results, err := referenceGlobFunc("nonexistent/*.md")

	// Assert - should return empty slice, not error
	require.NoError(t, err)
	assert.NotNil(t, results)
	assert.Len(t, results, 0)
}

// Test 4: Invalid glob pattern returns error
func TestReferenceGlobFunc_InvalidPatternReturnsError(t *testing.T) {
	// Arrange
	fs := afero.NewMemMapFs()
	resolver := NewMapFileResolver(map[string]string{})

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

	referenceGlobFunc := createReferenceGlobFunc(renderCtx, tmplCtx)

	// Act - use a malformed character class which is an invalid pattern
	results, err := referenceGlobFunc("[invalid")

	// Assert
	require.Error(t, err)
	assert.Nil(t, results)

	var patternErr *ReferenceGlobPatternError
	require.True(t, errors.As(err, &patternErr), "expected ReferenceGlobPatternError, got %T: %v", err, err)
	assert.Equal(t, "[invalid", patternErr.Pattern)

	// Verify it wraps filepath.ErrBadPattern
	assert.True(t, errors.Is(patternErr.Err, filepath.ErrBadPattern), "expected wrapped error to be filepath.ErrBadPattern, got %v", patternErr.Err)
}

// Test 5: Partials filtered from results
func TestReferenceGlobFunc_PartialsFilteredFromResults(t *testing.T) {
	// Arrange
	fs := afero.NewMemMapFs()
	afero.WriteFile(fs, "/content/_partials/header.md", []byte("# Header"), 0644)
	afero.WriteFile(fs, "/content/_partials/footer.md", []byte("# Footer"), 0644)
	afero.WriteFile(fs, "/content/claude/file.md", []byte("# File"), 0644)

	resolver := NewMapFileResolver(map[string]string{
		"_partials/header.md": "/content/_partials/header.md",
		"_partials/footer.md": "/content/_partials/footer.md",
		"claude/file.md":      "/content/claude/file.md",
	})

	config := profile.ContentConfig{
		Targets: map[string]string{
			"_partials": "_partials",
			"claude":    ".claude",
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

	referenceGlobFunc := createReferenceGlobFunc(renderCtx, tmplCtx)

	// Act - pattern that would match partials
	results, err := referenceGlobFunc("_partials/*.md")

	// Assert - partials should be filtered out, resulting in empty slice
	require.NoError(t, err)
	assert.NotNil(t, results)
	assert.Len(t, results, 0, "partials should be filtered from results")
}

// Test 6: Suppressed targets excluded from results
func TestReferenceGlobFunc_SuppressedTargetsExcludedFromResults(t *testing.T) {
	// Arrange
	fs := afero.NewMemMapFs()
	afero.WriteFile(fs, "/content/claude/file1.md", []byte("# File 1"), 0644)
	afero.WriteFile(fs, "/content/claude/file2.md", []byte("# File 2"), 0644)
	afero.WriteFile(fs, "/content/docs/readme.md", []byte("# Readme"), 0644)

	resolver := NewMapFileResolver(map[string]string{
		"claude/file1.md": "/content/claude/file1.md",
		"claude/file2.md": "/content/claude/file2.md",
		"docs/readme.md":  "/content/docs/readme.md",
	})

	config := profile.ContentConfig{
		Targets: map[string]string{
			"claude": ".claude",
			"docs":   ".docs",
		},
	}
	// Suppress the claude target via null override
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

	referenceGlobFunc := createReferenceGlobFunc(renderCtx, tmplCtx)

	// Act - pattern that matches suppressed claude files
	results, err := referenceGlobFunc("claude/*.md")

	// Assert - suppressed targets should be excluded, resulting in empty slice
	require.NoError(t, err)
	assert.NotNil(t, results)
	assert.Len(t, results, 0, "suppressed targets should be excluded from results")

	// Verify that non-suppressed targets still work
	docsResults, err := referenceGlobFunc("docs/*.md")
	require.NoError(t, err)
	assert.Len(t, docsResults, 1)
	assert.Equal(t, "@.docs/readme.md", docsResults[0])
}

// =============================================================================
// Task Group 5: Strategic Gap-Filling Tests for referenceGlob()
// =============================================================================

// Test 7: referenceGlob() with nil router returns descriptive error
func TestReferenceGlobFunc_NilRouterReturnsError(t *testing.T) {
	// Arrange
	fs := afero.NewMemMapFs()
	resolver := NewMapFileResolver(map[string]string{
		"docs/file.md": "/content/docs/file.md",
	})

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

	referenceGlobFunc := createReferenceGlobFunc(renderCtx, tmplCtx)

	// Act
	results, err := referenceGlobFunc("docs/*.md")

	// Assert - should return error when router is nil
	require.Error(t, err)
	assert.Nil(t, results)
	assert.Contains(t, err.Error(), "router not available")
}

// Test 8: referenceGlob() with .tmpl suffix files returns paths without .tmpl suffix
func TestReferenceGlobFunc_TmplSuffixStripping(t *testing.T) {
	// Arrange
	fs := afero.NewMemMapFs()
	afero.WriteFile(fs, "/content/docs/guide.md.tmpl", []byte("Guide"), 0644)
	afero.WriteFile(fs, "/content/docs/readme.md.tmpl", []byte("Readme"), 0644)
	afero.WriteFile(fs, "/content/docs/static.md", []byte("Static"), 0644)

	resolver := NewMapFileResolver(map[string]string{
		"docs/guide.md.tmpl":  "/content/docs/guide.md.tmpl",
		"docs/readme.md.tmpl": "/content/docs/readme.md.tmpl",
		"docs/static.md":      "/content/docs/static.md",
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

	referenceGlobFunc := createReferenceGlobFunc(renderCtx, tmplCtx)

	// Act - pattern that matches .tmpl files
	results, err := referenceGlobFunc("docs/*.tmpl")

	// Assert - .tmpl suffix should be stripped from output paths
	require.NoError(t, err)
	assert.Len(t, results, 2)
	// Output paths should NOT have .tmpl suffix
	assert.Contains(t, results, "@.docs/guide.md")
	assert.Contains(t, results, "@.docs/readme.md")
	// Ensure no .tmpl in results
	for _, result := range results {
		assert.NotContains(t, result, ".tmpl")
	}
}

// Test 9: referenceGlob() accessible in nested templates via include()
// This tests the critical workflow: main template -> include partial -> referenceGlob() works
func TestReferenceGlobFunc_WorksInNestedInclude(t *testing.T) {
	// Arrange
	fs := afero.NewMemMapFs()

	// Main template includes a partial that uses referenceGlob()
	mainTemplate := `# Main
{{ include "_partials/doc-list.md" }}`
	partialTemplate := `Documents:
{{ range referenceGlob "docs/*.md" }}- {{ . }}
{{ end }}`

	afero.WriteFile(fs, "/content/main.md.tmpl", []byte(mainTemplate), 0644)
	afero.WriteFile(fs, "/content/_partials/doc-list.md", []byte(partialTemplate), 0644)
	afero.WriteFile(fs, "/content/docs/alpha.md", []byte("Alpha"), 0644)
	afero.WriteFile(fs, "/content/docs/beta.md", []byte("Beta"), 0644)

	resolver := NewMapFileResolver(map[string]string{
		"main.md.tmpl":          "/content/main.md.tmpl",
		"_partials/doc-list.md": "/content/_partials/doc-list.md",
		"docs/alpha.md":         "/content/docs/alpha.md",
		"docs/beta.md":          "/content/docs/beta.md",
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
	assert.Contains(t, result, "Documents:")
	assert.Contains(t, result, "- @.docs/alpha.md")
	assert.Contains(t, result, "- @.docs/beta.md")
}

// Test 10: referenceGlob() across multiple target directories
// This tests that glob matching works correctly across files from different targets
func TestReferenceGlobFunc_AcrossMultipleTargetDirectories(t *testing.T) {
	// Arrange
	fs := afero.NewMemMapFs()
	afero.WriteFile(fs, "/content/claude/file.md", []byte("Claude"), 0644)
	afero.WriteFile(fs, "/content/cursor/file.md", []byte("Cursor"), 0644)
	afero.WriteFile(fs, "/content/agents/file.md", []byte("Agents"), 0644)

	resolver := NewMapFileResolver(map[string]string{
		"claude/file.md": "/content/claude/file.md",
		"cursor/file.md": "/content/cursor/file.md",
		"agents/file.md": "/content/agents/file.md",
	})

	config := profile.ContentConfig{
		Targets: map[string]string{
			"claude": ".claude",
			"cursor": ".cursor",
			"agents": ".agents",
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

	referenceGlobFunc := createReferenceGlobFunc(renderCtx, tmplCtx)

	// Act - pattern that matches files in multiple target directories
	results, err := referenceGlobFunc("*/file.md")

	// Assert - should get all files from different targets
	require.NoError(t, err)
	assert.Len(t, results, 3)
	assert.Contains(t, results, "@.agents/file.md")
	assert.Contains(t, results, "@.claude/file.md")
	assert.Contains(t, results, "@.cursor/file.md")
}
