package update_test

import (
	"testing"

	"github.com/simensen/weftlo/internal/app/update"
	"github.com/simensen/weftlo/internal/domain/manifest"
	"github.com/simensen/weftlo/internal/domain/profile"
	"github.com/simensen/weftlo/internal/domain/template"
	"github.com/simensen/weftlo/internal/testutil"
)

// =============================================================================
// Task Group 6: UpdatePlanner Function Tests
// =============================================================================

// Test UpdatePlanner accepts Change Detection results (map[string]FileStatus)
func TestUpdatePlanner_AcceptsChangeDetectionResults(t *testing.T) {
	t.Parallel()
	// Arrange
	fs := testutil.CreateTestFilesystem(t, map[string]string{
		"/profiles/vendor/test/content/config.md.tmpl": "Content",
	})
	engine := template.NewTemplateEngineWithFs(fs)

	// Create a profile using helper
	p := testutil.CreateTestProfile(t, "vendor/test", "config.md.tmpl")
	mergedProfile := testutil.CreateTestMergedProfile(t, fs, "/profiles", p)

	// Change detection results with various statuses
	changeResults := map[string]manifest.FileStatus{
		"config.md": manifest.FileStatusNew,
	}
	options := update.UpdateOptions{}

	// Act
	plan, err := update.PlanUpdate(changeResults, mergedProfile, options, engine, "/project")

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if plan.ToCreate == nil {
		t.Fatal("expected ToCreate slice to be initialized")
	}
}

// Test UpdatePlanner accepts MergedProfile for template re-rendering
func TestUpdatePlanner_AcceptsMergedProfile(t *testing.T) {
	t.Parallel()
	// Arrange
	fs := testutil.CreateTestFilesystem(t, map[string]string{
		"/profiles/vendor/test/content/file1.md.tmpl": "File 1: {{ .Profile.Name }}",
		"/profiles/vendor/test/content/file2.md.tmpl": "File 2",
	})
	engine := template.NewTemplateEngineWithFs(fs)

	// Create profile with multiple template files
	p := &profile.Profile{
		Name: "vendor/test",
		TemplateFiles: []profile.TemplateFile{
			{SourcePath: "file1.md.tmpl", IsPartial: false},
			{SourcePath: "file2.md.tmpl", IsPartial: false},
		},
	}
	mergedProfile := testutil.CreateTestMergedProfile(t, fs, "/profiles", p)

	changeResults := map[string]manifest.FileStatus{
		"file1.md": manifest.FileStatusNew,
		"file2.md": manifest.FileStatusSourceChanged,
	}
	options := update.UpdateOptions{}

	// Act
	plan, err := update.PlanUpdate(changeResults, mergedProfile, options, engine, "/project")

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	// Should have one create and one update
	if len(plan.ToCreate) != 1 {
		t.Errorf("expected 1 file in ToCreate, got %d", len(plan.ToCreate))
	}
	if len(plan.ToUpdate) != 1 {
		t.Errorf("expected 1 file in ToUpdate, got %d", len(plan.ToUpdate))
	}
}

// Test UpdatePlanner accepts UpdateOptions struct
func TestUpdatePlanner_AcceptsUpdateOptions(t *testing.T) {
	t.Parallel()
	// Arrange
	fs := testutil.CreateTestFilesystem(t, map[string]string{
		"/profiles/vendor/test/content/modified.md.tmpl": "Content",
	})
	engine := template.NewTemplateEngineWithFs(fs)

	p := testutil.CreateTestProfile(t, "vendor/test", "modified.md.tmpl")
	mergedProfile := testutil.CreateTestMergedProfile(t, fs, "/profiles", p)

	changeResults := map[string]manifest.FileStatus{
		"modified.md": manifest.FileStatusUserModified,
	}

	// Test with Force=true
	options := update.UpdateOptions{Force: true}

	// Act
	plan, err := update.PlanUpdate(changeResults, mergedProfile, options, engine, "/project")

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	// With Force=true, UserModified should go to ToUpdate, not ToSkip
	if len(plan.ToUpdate) != 1 {
		t.Errorf("expected 1 file in ToUpdate with Force=true, got %d", len(plan.ToUpdate))
	}
	if len(plan.ToSkip) != 0 {
		t.Errorf("expected 0 files in ToSkip with Force=true, got %d", len(plan.ToSkip))
	}
}

// Test UpdatePlanner returns populated UpdatePlan with all categories
func TestUpdatePlanner_ReturnsPopulatedUpdatePlan(t *testing.T) {
	t.Parallel()
	// Arrange
	fs := testutil.CreateTestFilesystem(t, map[string]string{
		"/profiles/vendor/test/content/new.md.tmpl":      "New file",
		"/profiles/vendor/test/content/changed.md.tmpl":  "Changed file",
		"/profiles/vendor/test/content/modified.md.tmpl": "Modified file",
		"/profiles/vendor/test/content/conflict.md.tmpl": "Conflict file",
	})
	engine := template.NewTemplateEngineWithFs(fs)

	p := &profile.Profile{
		Name: "vendor/test",
		TemplateFiles: []profile.TemplateFile{
			{SourcePath: "new.md.tmpl", IsPartial: false},
			{SourcePath: "changed.md.tmpl", IsPartial: false},
			{SourcePath: "modified.md.tmpl", IsPartial: false},
			{SourcePath: "conflict.md.tmpl", IsPartial: false},
		},
	}
	mergedProfile := testutil.CreateTestMergedProfile(t, fs, "/profiles", p)

	changeResults := map[string]manifest.FileStatus{
		"new.md":       manifest.FileStatusNew,
		"changed.md":   manifest.FileStatusSourceChanged,
		"modified.md":  manifest.FileStatusUserModified,
		"conflict.md":  manifest.FileStatusConflict,
		"removed.md":   manifest.FileStatusRemoved,
		"unchanged.md": manifest.FileStatusUnchanged,
	}

	// Default options - Force=false, RemoveOrphans=false
	options := update.UpdateOptions{}

	// Act
	plan, err := update.PlanUpdate(changeResults, mergedProfile, options, engine, "/project")

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Verify each category is populated correctly
	if len(plan.ToCreate) != 1 {
		t.Errorf("expected 1 file in ToCreate, got %d", len(plan.ToCreate))
	}
	if len(plan.ToUpdate) != 1 {
		t.Errorf("expected 1 file in ToUpdate, got %d", len(plan.ToUpdate))
	}
	if len(plan.ToSkip) != 2 {
		t.Errorf("expected 2 files in ToSkip (UserModified + Conflict), got %d", len(plan.ToSkip))
	}
	// RemoveOrphans=false, so ToDelete should be empty
	if len(plan.ToDelete) != 0 {
		t.Errorf("expected 0 files in ToDelete (RemoveOrphans=false), got %d", len(plan.ToDelete))
	}
}

// Test UpdatePlanner returns error for rendering failures
func TestUpdatePlanner_ReturnsErrorForRenderingFailures(t *testing.T) {
	t.Parallel()
	// Arrange
	fs := testutil.CreateTestFilesystem(t, map[string]string{
		"/profiles/vendor/test/content/bad.md.tmpl": "{{ .InvalidField }}",
	})
	engine := template.NewTemplateEngineWithFs(fs)

	p := testutil.CreateTestProfile(t, "vendor/test", "bad.md.tmpl")
	mergedProfile := testutil.CreateTestMergedProfile(t, fs, "/profiles", p)

	changeResults := map[string]manifest.FileStatus{
		"bad.md": manifest.FileStatusNew,
	}
	options := update.UpdateOptions{}

	// Act
	_, err := update.PlanUpdate(changeResults, mergedProfile, options, engine, "/project")

	// Assert
	if err == nil {
		t.Fatal("expected error for invalid template, got nil")
	}
}

// Test UpdatePlanner handles empty change results gracefully
func TestUpdatePlanner_HandlesEmptyChangeResults(t *testing.T) {
	t.Parallel()
	// Arrange
	fs := testutil.CreateTestFilesystem(t, map[string]string{})
	engine := template.NewTemplateEngineWithFs(fs)

	p := &profile.Profile{
		Name:          "vendor/test",
		TemplateFiles: []profile.TemplateFile{},
	}
	mergedProfile := testutil.CreateTestMergedProfile(t, fs, "/profiles", p)

	// Empty change results
	changeResults := map[string]manifest.FileStatus{}
	options := update.UpdateOptions{}

	// Act
	plan, err := update.PlanUpdate(changeResults, mergedProfile, options, engine, "/project")

	// Assert
	if err != nil {
		t.Fatalf("expected no error for empty results, got %v", err)
	}
	if len(plan.ToCreate) != 0 {
		t.Errorf("expected empty ToCreate, got %d", len(plan.ToCreate))
	}
	if len(plan.ToUpdate) != 0 {
		t.Errorf("expected empty ToUpdate, got %d", len(plan.ToUpdate))
	}
	if len(plan.ToSkip) != 0 {
		t.Errorf("expected empty ToSkip, got %d", len(plan.ToSkip))
	}
	if len(plan.ToDelete) != 0 {
		t.Errorf("expected empty ToDelete, got %d", len(plan.ToDelete))
	}
}
