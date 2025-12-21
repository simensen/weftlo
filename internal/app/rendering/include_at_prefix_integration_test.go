package rendering_test

import (
	"strings"
	"testing"

	"github.com/spf13/afero"

	"github.com/simensen/weftlo/internal/app/rendering"
	"github.com/simensen/weftlo/internal/domain/profile"
	"github.com/simensen/weftlo/internal/domain/template"
	infraprofile "github.com/simensen/weftlo/internal/infrastructure/profile"
)

// ============================================================================
// Task Group 5: Full Pipeline Integration Tests for @ Prefix Behavior
//
// These tests verify the include() behavior through the full rendering
// pipeline with actual MergedProfile inheritance chains.
// ============================================================================

// Test 5.1/5.2: Profile inheritance scenario - include WITHOUT @ prefix
// This is the full end-to-end test showing that plain paths work with inheritance.
func TestRenderAll_IncludeWithoutAtPrefix_InheritanceWorks(t *testing.T) {
	t.Parallel()
	// Arrange: Create parent profile with shared partial
	fs := afero.NewMemMapFs()
	engine := template.NewTemplateEngineWithFs(fs)
	pipeline := rendering.NewRenderingPipeline(engine)

	// Parent profile: has _shared/header.md partial
	parentHeaderContent := "# Header from Parent Profile"
	afero.WriteFile(fs, "/profiles/test/parent/content/_shared/header.md",
		[]byte(parentHeaderContent), 0644)

	// Child profile: has main.md.tmpl that includes parent's header
	// Using plain path - this is the CORRECT way
	childMainContent := `Document Start
{{ include "_shared/header.md" }}
Document End`
	afero.WriteFile(fs, "/profiles/test/child/content/main.md.tmpl",
		[]byte(childMainContent), 0644)

	// Create inheritance chain: parent -> child
	chain := []*profile.Profile{
		{
			Name: "test/parent",
			TemplateFiles: []profile.TemplateFile{
				{SourcePath: "_shared/header.md", IsPartial: true},
			},
		},
		{
			Name:         "test/child",
			InheritsFrom: "test/parent",
			TemplateFiles: []profile.TemplateFile{
				{SourcePath: "main.md.tmpl", IsPartial: false},
			},
		},
	}
	mergedProfile := infraprofile.NewMergedProfile(chain, fs, "/profiles")

	// Act: Render all templates
	results, err := pipeline.RenderAll(mergedProfile, "/project", nil)

	// Assert: Should succeed
	if err != nil {
		t.Fatalf("Expected RenderAll to succeed with plain path include, got: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(results))
	}

	expected := `Document Start
# Header from Parent Profile
Document End`
	if results[0].Content != expected {
		t.Errorf("Expected:\n%s\n\nGot:\n%s", expected, results[0].Content)
	}

	t.Logf("SUCCESS: Include via plain path works with inheritance")
	t.Logf("Template: {{ include \"_shared/header.md\" }}")
	t.Logf("Rendered content:\n%s", results[0].Content)
}

// Test 5.3: Profile inheritance scenario - include WITH @ prefix
// This demonstrates that @ prefix does NOT work as documented in README.
func TestRenderAll_IncludeWithAtPrefix_Fails(t *testing.T) {
	t.Parallel()
	// Arrange: Same setup as successful test
	fs := afero.NewMemMapFs()
	engine := template.NewTemplateEngineWithFs(fs)
	pipeline := rendering.NewRenderingPipeline(engine)

	parentHeaderContent := "# Header from Parent Profile"
	afero.WriteFile(fs, "/profiles/test/parent/content/_shared/header.md",
		[]byte(parentHeaderContent), 0644)

	// Child profile uses @ prefix syntax as shown in README examples
	// This is the INCORRECT way that doesn't work
	childMainContent := `Document Start
{{ include "@test/parent/_shared/header.md" }}
Document End`
	afero.WriteFile(fs, "/profiles/test/child/content/main.md.tmpl",
		[]byte(childMainContent), 0644)

	chain := []*profile.Profile{
		{
			Name: "test/parent",
			TemplateFiles: []profile.TemplateFile{
				{SourcePath: "_shared/header.md", IsPartial: true},
			},
		},
		{
			Name:         "test/child",
			InheritsFrom: "test/parent",
			TemplateFiles: []profile.TemplateFile{
				{SourcePath: "main.md.tmpl", IsPartial: false},
			},
		},
	}
	mergedProfile := infraprofile.NewMergedProfile(chain, fs, "/profiles")

	// Act: Render all templates
	_, err := pipeline.RenderAll(mergedProfile, "/project", nil)

	// Assert: Should FAIL because @ prefix is not parsed
	if err == nil {
		t.Fatal("Expected RenderAll to fail with @ prefix path, but it succeeded")
	}

	// Verify the error message contains the problematic path with @ prefix
	errorMsg := err.Error()
	if !strings.Contains(errorMsg, "@test/parent/_shared/header.md") {
		t.Errorf("Expected error to contain literal @ prefix path, got: %s", errorMsg)
	}

	t.Logf("EXPECTED FAILURE: Include via @ prefix path fails")
	t.Logf("Template: {{ include \"@test/parent/_shared/header.md\" }}")
	t.Logf("Error: %v", err)
}

// Test 5.4: Summary comparison test
func TestRenderAll_IncludeBehaviorSummary(t *testing.T) {
	t.Parallel()
	// This test creates both scenarios side by side for comparison

	fs := afero.NewMemMapFs()
	engine := template.NewTemplateEngineWithFs(fs)

	// Shared parent header
	parentHeaderContent := "Shared Header Content"
	afero.WriteFile(fs, "/profiles/org/base/content/_shared/header.md",
		[]byte(parentHeaderContent), 0644)

	// Test A: Plain path (works)
	t.Run("PlainPath_Works", func(t *testing.T) {
		afero.WriteFile(fs, "/profiles/org/frontend/content/main.md.tmpl",
			[]byte("{{ include \"_shared/header.md\" }}"), 0644)

		chain := []*profile.Profile{
			{
				Name: "org/base",
				TemplateFiles: []profile.TemplateFile{
					{SourcePath: "_shared/header.md", IsPartial: true},
				},
			},
			{
				Name:         "org/frontend",
				InheritsFrom: "org/base",
				TemplateFiles: []profile.TemplateFile{
					{SourcePath: "main.md.tmpl", IsPartial: false},
				},
			},
		}

		mergedProfile := infraprofile.NewMergedProfile(chain, fs, "/profiles")
		pipeline := rendering.NewRenderingPipeline(engine)

		results, err := pipeline.RenderAll(mergedProfile, "/project", nil)
		if err != nil {
			t.Errorf("Plain path should work: %v", err)
		} else if len(results) > 0 && results[0].Content != parentHeaderContent {
			t.Errorf("Expected %q, got %q", parentHeaderContent, results[0].Content)
		} else {
			t.Logf("Plain path include works: %q", results[0].Content)
		}
	})

	// Test B: @ prefix path (fails)
	t.Run("AtPrefixPath_Fails", func(t *testing.T) {
		afero.WriteFile(fs, "/profiles/org/mobile/content/main.md.tmpl",
			[]byte("{{ include \"@org/base/_shared/header.md\" }}"), 0644)

		chain := []*profile.Profile{
			{
				Name: "org/base",
				TemplateFiles: []profile.TemplateFile{
					{SourcePath: "_shared/header.md", IsPartial: true},
				},
			},
			{
				Name:         "org/mobile",
				InheritsFrom: "org/base",
				TemplateFiles: []profile.TemplateFile{
					{SourcePath: "main.md.tmpl", IsPartial: false},
				},
			},
		}

		mergedProfile := infraprofile.NewMergedProfile(chain, fs, "/profiles")
		pipeline := rendering.NewRenderingPipeline(engine)

		_, err := pipeline.RenderAll(mergedProfile, "/project", nil)
		if err == nil {
			t.Error("@ prefix path should fail, but it succeeded")
		} else {
			t.Logf("@ prefix path fails as expected: %v", err)
		}
	})

	t.Log("============================================================")
	t.Log("FULL PIPELINE TEST SUMMARY:")
	t.Log("")
	t.Log("WORKING approach (use this):")
	t.Log("  {{ include \"_shared/header.md\" }}")
	t.Log("")
	t.Log("BROKEN approach (do not use):")
	t.Log("  {{ include \"@org/base/_shared/header.md\" }}")
	t.Log("")
	t.Log("The @ prefix is NOT parsed by include(). The plain path works")
	t.Log("because inheritance merges parent files into the child's file map.")
	t.Log("============================================================")
}
