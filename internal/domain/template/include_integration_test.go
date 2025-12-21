package template

import (
	"errors"
	"strconv"
	"testing"
	"time"

	"github.com/spf13/afero"

	"github.com/simensen/weftlo/internal/domain/config"
	"github.com/simensen/weftlo/internal/domain/profile"
)

// ============================================================================
// Strategic Gap-Filling Tests for Include Function Feature (Task Group 5)
// ============================================================================

// Integration Test: Template including partial that uses sprig functions
// This tests the full workflow where a parent template includes a partial
// and that partial uses sprig functions to transform the context data.
func TestIncludeIntegration_PartialWithSprigFunctions(t *testing.T) {
	// Arrange
	fs := afero.NewMemMapFs()

	// Partial that uses sprig functions to format profile data
	partialContent := `## Profile: {{ .Profile.Name | upper }}
- Directory: {{ .ProjectDir | base }}
- Generated: {{ .GeneratedAt.Format "2006-01-02" | quote }}`
	afero.WriteFile(fs, "/templates/_partials/profile-info.md", []byte(partialContent), 0644)

	resolver := NewMapFileResolver(map[string]string{
		"_partials/profile-info.md": "/templates/_partials/profile-info.md",
	})

	renderCtx := NewRenderContext(fs, resolver)
	tmplCtx := NewTemplateContext(
		&profile.Profile{Name: "vendor/my-profile"},
		"/home/user/projects/my-app",
		time.Date(2025, 12, 21, 10, 30, 0, 0, time.UTC),
		&config.ProjectConfig{Profiles: []string{"vendor/my-profile"}},
		nil,
		nil,
	)

	includeFunc := createIncludeFunc(renderCtx, tmplCtx)

	// Act
	result, err := includeFunc("_partials/profile-info.md")

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	expected := `## Profile: VENDOR/MY-PROFILE
- Directory: my-app
- Generated: "2025-12-21"`
	if result != expected {
		t.Errorf("expected:\n%s\n\ngot:\n%s", expected, result)
	}
}

// Integration Test: Multi-level inheritance simulation (child -> parent -> grandparent)
// This simulates a real-world scenario where templates are organized in an inheritance chain.
func TestIncludeIntegration_MultiLevelInheritanceChain(t *testing.T) {
	// Arrange
	fs := afero.NewMemMapFs()

	// Grandparent provides base header
	afero.WriteFile(fs, "/profiles/grandparent/templates/_partials/base-header.md",
		[]byte("=== Base Header from Grandparent ==="), 0644)

	// Parent provides footer (inherits from grandparent)
	afero.WriteFile(fs, "/profiles/parent/templates/_partials/footer.md",
		[]byte("--- Footer from Parent ---"), 0644)

	// Child provides custom sidebar (overrides nothing, adds new)
	afero.WriteFile(fs, "/profiles/child/templates/_partials/sidebar.md",
		[]byte("| Sidebar from Child |"), 0644)

	// Main template includes all three levels
	mainTemplate := `{{ include "_partials/base-header.md" }}

Content for {{ .Profile.Name }}

{{ include "_partials/sidebar.md" }}

{{ include "_partials/footer.md" }}`
	afero.WriteFile(fs, "/profiles/child/templates/main.md", []byte(mainTemplate), 0644)

	// Resolver simulates merged profile where child has access to all inherited partials
	resolver := NewMapFileResolver(map[string]string{
		"_partials/base-header.md": "/profiles/grandparent/templates/_partials/base-header.md",
		"_partials/footer.md":      "/profiles/parent/templates/_partials/footer.md",
		"_partials/sidebar.md":     "/profiles/child/templates/_partials/sidebar.md",
		"main.md":                  "/profiles/child/templates/main.md",
	})

	renderCtx := NewRenderContext(fs, resolver)
	tmplCtx := NewTemplateContext(
		&profile.Profile{
			Name:         "child/profile",
			InheritsFrom: "parent/profile",
		},
		"/project",
		time.Now(),
		nil,
		nil,
		nil,
	)

	includeFunc := createIncludeFunc(renderCtx, tmplCtx)

	// Act
	result, err := includeFunc("main.md")

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	expected := `=== Base Header from Grandparent ===

Content for child/profile

| Sidebar from Child |

--- Footer from Parent ---`
	if result != expected {
		t.Errorf("expected:\n%s\n\ngot:\n%s", expected, result)
	}
}

// Edge Case Test: Include with empty file content
// The include function should handle empty files gracefully, returning empty string.
func TestIncludeEdgeCase_EmptyFileContent(t *testing.T) {
	// Arrange
	fs := afero.NewMemMapFs()
	afero.WriteFile(fs, "/templates/empty.md", []byte(""), 0644)

	resolver := NewMapFileResolver(map[string]string{
		"empty.md": "/templates/empty.md",
	})

	renderCtx := NewRenderContext(fs, resolver)
	tmplCtx := NewTemplateContext(
		&profile.Profile{Name: "test/profile"},
		"/project",
		time.Now(),
		nil,
		nil,
		nil,
	)

	includeFunc := createIncludeFunc(renderCtx, tmplCtx)

	// Act
	result, err := includeFunc("empty.md")

	// Assert
	if err != nil {
		t.Fatalf("expected no error for empty file, got %v", err)
	}
	if result != "" {
		t.Errorf("expected empty string for empty file, got %q", result)
	}
}

// Edge Case Test: Include path with special characters
// Paths with spaces, dashes, underscores, and dots should be handled correctly.
func TestIncludeEdgeCase_PathWithSpecialCharacters(t *testing.T) {
	// Arrange
	fs := afero.NewMemMapFs()
	afero.WriteFile(fs, "/templates/_partials/my-special_file.2024.md",
		[]byte("Special file content"), 0644)
	afero.WriteFile(fs, "/templates/path with spaces/file.md",
		[]byte("File in path with spaces"), 0644)

	resolver := NewMapFileResolver(map[string]string{
		"_partials/my-special_file.2024.md": "/templates/_partials/my-special_file.2024.md",
		"path with spaces/file.md":          "/templates/path with spaces/file.md",
	})

	renderCtx := NewRenderContext(fs, resolver)
	tmplCtx := NewTemplateContext(
		&profile.Profile{Name: "test/profile"},
		"/project",
		time.Now(),
		nil,
		nil,
		nil,
	)

	includeFunc := createIncludeFunc(renderCtx, tmplCtx)

	// Test 1: Path with dashes, underscores, and dots
	result1, err1 := includeFunc("_partials/my-special_file.2024.md")
	if err1 != nil {
		t.Errorf("expected no error for special chars in filename, got %v", err1)
	}
	if result1 != "Special file content" {
		t.Errorf("expected 'Special file content', got %q", result1)
	}

	// Test 2: Path with spaces
	result2, err2 := includeFunc("path with spaces/file.md")
	if err2 != nil {
		t.Errorf("expected no error for path with spaces, got %v", err2)
	}
	if result2 != "File in path with spaces" {
		t.Errorf("expected 'File in path with spaces', got %q", result2)
	}
}

// Edge Case Test: Large include chain at depth limit boundary (depth 9 vs 10)
// This verifies the exact boundary behavior at MaxDepth.
func TestIncludeEdgeCase_DepthLimitBoundary(t *testing.T) {
	// Arrange
	fs := afero.NewMemMapFs()

	// Create exactly 10 levels of includes (0-9), which should succeed
	// Then create level 10 which would exceed the limit
	for i := 0; i <= 10; i++ {
		var content string
		if i < 10 {
			content = "L" + strconv.Itoa(i) + ":{{ include \"level" + strconv.Itoa(i+1) + ".md\" }}"
		} else {
			content = "L" + strconv.Itoa(i) + ":end"
		}
		afero.WriteFile(fs, "/templates/level"+strconv.Itoa(i)+".md", []byte(content), 0644)
	}

	fileMap := make(map[string]string)
	for i := 0; i <= 10; i++ {
		fileMap["level"+strconv.Itoa(i)+".md"] = "/templates/level" + strconv.Itoa(i) + ".md"
	}

	resolver := NewMapFileResolver(fileMap)
	tmplCtx := NewTemplateContext(
		&profile.Profile{Name: "test/profile"},
		"/project",
		time.Now(),
		nil,
		nil,
		nil,
	)

	// Test at depth 9 (should succeed - exactly at the limit)
	t.Run("depth_9_succeeds", func(t *testing.T) {
		// Start at depth 0, chain goes 0->1->2->3->4->5->6->7->8->9 (10 levels = MaxDepth)
		renderCtx := NewRenderContext(fs, resolver)
		// Manually set to depth 0 to start a fresh chain
		renderCtx.CurrentDepth = 0

		// Create a 9-level chain that ends without exceeding depth
		// The 10th level (level9.md including level10.md) will be at depth 9
		// which equals MaxDepth-1, so it should succeed
		fs9 := afero.NewMemMapFs()
		for i := 0; i <= 9; i++ {
			var content string
			if i < 9 {
				content = "L" + strconv.Itoa(i) + ":{{ include \"level" + strconv.Itoa(i+1) + ".md\" }}"
			} else {
				content = "L" + strconv.Itoa(i) + ":end"
			}
			afero.WriteFile(fs9, "/templates/level"+strconv.Itoa(i)+".md", []byte(content), 0644)
		}

		fileMap9 := make(map[string]string)
		for i := 0; i <= 9; i++ {
			fileMap9["level"+strconv.Itoa(i)+".md"] = "/templates/level" + strconv.Itoa(i) + ".md"
		}

		resolver9 := NewMapFileResolver(fileMap9)
		renderCtx9 := NewRenderContext(fs9, resolver9)
		includeFunc := createIncludeFunc(renderCtx9, tmplCtx)

		result, err := includeFunc("level0.md")
		if err != nil {
			t.Fatalf("depth 9 should succeed, got error: %v", err)
		}
		expected := "L0:L1:L2:L3:L4:L5:L6:L7:L8:L9:end"
		if result != expected {
			t.Errorf("expected %q, got %q", expected, result)
		}
	})

	// Test at depth 10 (should fail - exceeds the limit)
	t.Run("depth_10_fails", func(t *testing.T) {
		renderCtx := NewRenderContext(fs, resolver)
		includeFunc := createIncludeFunc(renderCtx, tmplCtx)

		_, err := includeFunc("level0.md")
		if err == nil {
			t.Fatal("depth 10 should fail with IncludeDepthError")
		}

		var depthErr *IncludeDepthError
		if !errors.As(err, &depthErr) {
			t.Fatalf("expected IncludeDepthError, got %T: %v", err, err)
		}

		if depthErr.MaxDepth != 10 {
			t.Errorf("expected MaxDepth 10, got %d", depthErr.MaxDepth)
		}
	})
}

// Edge Case Test: Circular reference detection
// While we don't have explicit circular detection, the depth limit prevents infinite loops.
// This test verifies that a circular reference is caught by the depth limit.
func TestIncludeEdgeCase_CircularReferenceDetectedByDepthLimit(t *testing.T) {
	// Arrange
	fs := afero.NewMemMapFs()

	// Create a circular reference: A includes B, B includes A
	afero.WriteFile(fs, "/templates/a.md", []byte("A:{{ include \"b.md\" }}"), 0644)
	afero.WriteFile(fs, "/templates/b.md", []byte("B:{{ include \"a.md\" }}"), 0644)

	resolver := NewMapFileResolver(map[string]string{
		"a.md": "/templates/a.md",
		"b.md": "/templates/b.md",
	})

	renderCtx := NewRenderContext(fs, resolver)
	tmplCtx := NewTemplateContext(
		&profile.Profile{Name: "test/profile"},
		"/project",
		time.Now(),
		nil,
		nil,
		nil,
	)

	includeFunc := createIncludeFunc(renderCtx, tmplCtx)

	// Act
	_, err := includeFunc("a.md")

	// Assert
	if err == nil {
		t.Fatal("expected IncludeDepthError for circular reference, got nil")
	}

	var depthErr *IncludeDepthError
	if !errors.As(err, &depthErr) {
		t.Fatalf("expected IncludeDepthError for circular reference, got %T: %v", err, err)
	}

	// Verify the include chain shows the circular pattern
	chain := depthErr.IncludeChain
	if len(chain) < 2 {
		t.Errorf("expected include chain to show circular pattern, got %v", chain)
	}
}

// Edge Case Test: Include with whitespace-only file content
// Whitespace should be preserved, similar to empty template rendering.
func TestIncludeEdgeCase_WhitespaceOnlyFileContent(t *testing.T) {
	// Arrange
	fs := afero.NewMemMapFs()
	whitespaceContent := "   \n\t\n   "
	afero.WriteFile(fs, "/templates/whitespace.md", []byte(whitespaceContent), 0644)

	resolver := NewMapFileResolver(map[string]string{
		"whitespace.md": "/templates/whitespace.md",
	})

	renderCtx := NewRenderContext(fs, resolver)
	tmplCtx := NewTemplateContext(
		&profile.Profile{Name: "test/profile"},
		"/project",
		time.Now(),
		nil,
		nil,
		nil,
	)

	includeFunc := createIncludeFunc(renderCtx, tmplCtx)

	// Act
	result, err := includeFunc("whitespace.md")

	// Assert
	if err != nil {
		t.Fatalf("expected no error for whitespace file, got %v", err)
	}
	if result != whitespaceContent {
		t.Errorf("expected whitespace to be preserved %q, got %q", whitespaceContent, result)
	}
}
