package template

import (
	"errors"
	"testing"
	"time"

	"github.com/spf13/afero"

	"github.com/simensen/weftlo/internal/domain/profile"
)

// ============================================================================
// Task Group 5: Behavioral Test Verification for @ Prefix in include()
//
// These tests document the current behavior of the include() function
// regarding @ prefix handling. They are part of the research spike to
// determine if this is a bug, documentation error, or non-issue.
// ============================================================================

// Test 5.2: Test include() WITHOUT @ prefix
// This test verifies that include with plain paths works correctly with inheritance.
// When a child profile inherits from a parent profile, the child should be able
// to include files from the parent using plain source paths.
func TestIncludeBehavior_WithoutAtPrefix_InheritanceWorks(t *testing.T) {
	// Arrange: Create a parent profile with a shared header file
	fs := afero.NewMemMapFs()

	// Parent profile has a shared header at _shared/header.md
	parentHeaderContent := "# Header from Parent Profile"
	afero.WriteFile(fs, "/profiles/test/parent/content/_shared/header.md",
		[]byte(parentHeaderContent), 0644)

	// Child profile has a main template that includes the parent's header
	// Using plain path WITHOUT @ prefix
	childMainTemplate := `Document Start
{{ include "_shared/header.md" }}
Document End`
	afero.WriteFile(fs, "/profiles/test/child/content/main.md.tmpl",
		[]byte(childMainTemplate), 0644)

	// Create a file resolver that simulates inheritance:
	// The child profile has access to parent's files via merged profile
	resolver := NewMapFileResolver(map[string]string{
		// Parent file available to child via inheritance
		"_shared/header.md": "/profiles/test/parent/content/_shared/header.md",
		// Child's own template
		"main.md.tmpl": "/profiles/test/child/content/main.md.tmpl",
	})

	renderCtx := NewRenderContext(fs, resolver)
	tmplCtx := NewTemplateContext(
		&profile.Profile{
			Name:         "test/child",
			InheritsFrom: "test/parent",
		},
		"/project",
		time.Now(),
		nil,
		nil,
		nil,
	)

	includeFunc := createIncludeFunc(renderCtx, tmplCtx)

	// Act: Include the parent's header using plain path
	result, err := includeFunc("_shared/header.md")

	// Assert: Should work - plain path resolves via inheritance
	if err != nil {
		t.Fatalf("Expected include without @ prefix to work with inheritance, got error: %v", err)
	}
	if result != parentHeaderContent {
		t.Errorf("Expected parent header content %q, got %q", parentHeaderContent, result)
	}

	// Document the result for Task 5.2
	t.Logf("SUCCESS: include(\"_shared/header.md\") works via inheritance")
	t.Logf("Result: %q", result)
}

// Test 5.3: Test include() WITH @ prefix
// This test verifies the behavior when using @ prefix in include().
// Based on code analysis, the @ prefix is NOT parsed - it's passed directly
// to the resolver which does a simple map lookup.
func TestIncludeBehavior_WithAtPrefix_Fails(t *testing.T) {
	// Arrange: Same setup as test without @ prefix
	fs := afero.NewMemMapFs()

	parentHeaderContent := "# Header from Parent Profile"
	afero.WriteFile(fs, "/profiles/test/parent/content/_shared/header.md",
		[]byte(parentHeaderContent), 0644)

	// File resolver maps PLAIN paths, not @ prefixed paths
	// This simulates how MergedProfile.Resolve() works
	resolver := NewMapFileResolver(map[string]string{
		// Map keys are plain paths like "_shared/header.md"
		// NOT prefixed paths like "@test/parent/_shared/header.md"
		"_shared/header.md": "/profiles/test/parent/content/_shared/header.md",
	})

	renderCtx := NewRenderContext(fs, resolver)
	tmplCtx := NewTemplateContext(
		&profile.Profile{
			Name:         "test/child",
			InheritsFrom: "test/parent",
		},
		"/project",
		time.Now(),
		nil,
		nil,
		nil,
	)

	includeFunc := createIncludeFunc(renderCtx, tmplCtx)

	// Act: Try to include using @ prefix syntax (as shown in README examples)
	_, err := includeFunc("@test/parent/_shared/header.md")

	// Assert: Should FAIL because @ prefix is not parsed
	if err == nil {
		t.Fatal("Expected include with @ prefix to fail (@ prefix not parsed), but it succeeded")
	}

	// Verify it's an IncludeNotFoundError
	var notFoundErr *IncludeNotFoundError
	if !errors.As(err, &notFoundErr) {
		t.Fatalf("Expected IncludeNotFoundError, got %T: %v", err, err)
	}

	// Document the exact error for Task 5.3
	t.Logf("EXPECTED FAILURE: include(\"@test/parent/_shared/header.md\") fails")
	t.Logf("Error type: IncludeNotFoundError")
	t.Logf("Error message: %v", err)
	t.Logf("Source path in error: %q", notFoundErr.SourcePath)

	// Verify the error contains the literal @ prefix (confirming it wasn't parsed)
	if notFoundErr.SourcePath != "@test/parent/_shared/header.md" {
		t.Errorf("Expected error to contain literal @ prefix path, got: %q", notFoundErr.SourcePath)
	}
}

// Test 5.4: Full E2E test documenting both approaches for comparison
// This provides a clear before/after comparison of both approaches.
func TestIncludeBehavior_ComparisonSummary(t *testing.T) {
	// Arrange: Setup profile inheritance scenario
	fs := afero.NewMemMapFs()

	sharedContent := "Shared content from parent profile"
	afero.WriteFile(fs, "/profiles/org/base/content/_shared/header.md",
		[]byte(sharedContent), 0644)

	resolver := NewMapFileResolver(map[string]string{
		"_shared/header.md": "/profiles/org/base/content/_shared/header.md",
	})

	renderCtx := NewRenderContext(fs, resolver)
	tmplCtx := NewTemplateContext(
		&profile.Profile{
			Name:         "org/frontend",
			InheritsFrom: "org/base",
		},
		"/project",
		time.Now(),
		nil,
		nil,
		nil,
	)

	includeFunc := createIncludeFunc(renderCtx, tmplCtx)

	// Test case A: WITHOUT @ prefix (should work)
	t.Run("without_at_prefix_works", func(t *testing.T) {
		result, err := includeFunc("_shared/header.md")
		if err != nil {
			t.Errorf("Plain path FAILED: %v", err)
		} else {
			t.Logf("Plain path SUCCESS: %q", result)
		}
	})

	// Test case B: WITH @ prefix (should fail)
	t.Run("with_at_prefix_fails", func(t *testing.T) {
		result, err := includeFunc("@org/base/_shared/header.md")
		if err == nil {
			t.Errorf("@ prefix path unexpectedly SUCCEEDED: %q", result)
		} else {
			t.Logf("@ prefix path FAILED as expected: %v", err)
		}
	})

	// Summary
	t.Log("============================================================")
	t.Log("BEHAVIOR SUMMARY:")
	t.Log("  include(\"_shared/header.md\")           -> WORKS (via inheritance)")
	t.Log("  include(\"@org/base/_shared/header.md\") -> FAILS (@ not parsed)")
	t.Log("============================================================")
	t.Log("ROOT CAUSE: include.go line 36 passes sourcePath directly to")
	t.Log("FileResolver.Resolve() with NO @ prefix parsing or stripping.")
	t.Log("Map keys in MergedProfile are plain paths like \"_shared/header.md\"")
	t.Log("not prefixed paths like \"@org/base/_shared/header.md\"")
	t.Log("============================================================")
}

// Test to verify that the @ prefix error message is clear and actionable
func TestIncludeBehavior_AtPrefixErrorMessage(t *testing.T) {
	fs := afero.NewMemMapFs()
	afero.WriteFile(fs, "/profiles/org/base/content/_shared/header.md",
		[]byte("header content"), 0644)

	resolver := NewMapFileResolver(map[string]string{
		"_shared/header.md": "/profiles/org/base/content/_shared/header.md",
	})

	renderCtx := NewRenderContext(fs, resolver)
	tmplCtx := NewTemplateContext(
		&profile.Profile{Name: "org/frontend"},
		"/project",
		time.Now(),
		nil,
		nil,
		nil,
	)

	includeFunc := createIncludeFunc(renderCtx, tmplCtx)

	// Test with @ prefix
	_, err := includeFunc("@org/base/_shared/header.md")

	if err == nil {
		t.Fatal("Expected error for @ prefix, got nil")
	}

	errorMsg := err.Error()

	// Document the exact error message for the findings document
	t.Logf("Full error message: %s", errorMsg)

	// Verify error message contains the problematic path
	if notFoundErr, ok := err.(*IncludeNotFoundError); ok {
		t.Logf("SourcePath in error: %q", notFoundErr.SourcePath)
		t.Logf("IncludeChain: %v", notFoundErr.IncludeChain)
	}
}
