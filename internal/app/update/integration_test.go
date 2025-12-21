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
// Task Group 7: Integration and Gap-Filling Tests
// =============================================================================

// Test UpdatePlanner with Force=true and RemoveOrphans=true combined
// This verifies that both flags work together correctly in the full pipeline
func TestUpdatePlanner_CombinedForceAndRemoveOrphans(t *testing.T) {
	t.Parallel()
	// Arrange
	fs := testutil.CreateTestFilesystem(t, map[string]string{
		"/profiles/vendor/test/content/modified.md.tmpl": "Modified content",
		"/profiles/vendor/test/content/conflict.md.tmpl": "Conflict content",
	})
	engine := template.NewTemplateEngineWithFs(fs)

	p := &profile.Profile{
		Name: "vendor/test",
		TemplateFiles: []profile.TemplateFile{
			{SourcePath: "modified.md.tmpl", IsPartial: false},
			{SourcePath: "conflict.md.tmpl", IsPartial: false},
		},
	}
	mergedProfile := testutil.CreateTestMergedProfile(t, fs, "/profiles", p)

	changeResults := map[string]manifest.FileStatus{
		"modified.md": manifest.FileStatusUserModified,
		"conflict.md": manifest.FileStatusConflict,
		"removed.md":  manifest.FileStatusRemoved,
	}

	// Both Force and RemoveOrphans enabled
	options := update.UpdateOptions{Force: true, RemoveOrphans: true}

	// Act
	plan, err := update.PlanUpdate(changeResults, mergedProfile, options, engine, "/project")

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Modified and conflict should go to ToUpdate due to Force=true
	if len(plan.ToUpdate) != 2 {
		t.Errorf("expected 2 files in ToUpdate (modified + conflict with Force=true), got %d", len(plan.ToUpdate))
	}

	// Removed should go to ToDelete due to RemoveOrphans=true
	if len(plan.ToDelete) != 1 {
		t.Errorf("expected 1 file in ToDelete (removed with RemoveOrphans=true), got %d", len(plan.ToDelete))
	}

	// No files should be skipped when Force=true
	if len(plan.ToSkip) != 0 {
		t.Errorf("expected 0 files in ToSkip with Force=true, got %d", len(plan.ToSkip))
	}
}

// Test that SourceProfile is correctly populated in FileDecisions
func TestUpdatePlanner_SourceProfilePopulation(t *testing.T) {
	t.Parallel()
	// Arrange
	fs := testutil.CreateTestFilesystem(t, map[string]string{
		"/profiles/vendor/my-profile/content/new.md.tmpl":  "New content",
		"/profiles/vendor/my-profile/content/skip.md.tmpl": "Skip content",
	})
	engine := template.NewTemplateEngineWithFs(fs)

	p := &profile.Profile{
		Name: "vendor/my-profile",
		TemplateFiles: []profile.TemplateFile{
			{SourcePath: "new.md.tmpl", IsPartial: false},
			{SourcePath: "skip.md.tmpl", IsPartial: false},
		},
	}
	mergedProfile := testutil.CreateTestMergedProfile(t, fs, "/profiles", p)

	changeResults := map[string]manifest.FileStatus{
		"new.md":  manifest.FileStatusNew,
		"skip.md": manifest.FileStatusUserModified,
	}
	options := update.UpdateOptions{}

	// Act
	plan, err := update.PlanUpdate(changeResults, mergedProfile, options, engine, "/project")

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Check SourceProfile in ToCreate
	if len(plan.ToCreate) != 1 {
		t.Fatalf("expected 1 file in ToCreate, got %d", len(plan.ToCreate))
	}
	if plan.ToCreate[0].SourceProfile != "vendor/my-profile" {
		t.Errorf("expected SourceProfile 'vendor/my-profile' in ToCreate, got '%s'", plan.ToCreate[0].SourceProfile)
	}

	// Check SourceProfile in ToSkip
	if len(plan.ToSkip) != 1 {
		t.Fatalf("expected 1 file in ToSkip, got %d", len(plan.ToSkip))
	}
	if plan.ToSkip[0].SourceProfile != "vendor/my-profile" {
		t.Errorf("expected SourceProfile 'vendor/my-profile' in ToSkip, got '%s'", plan.ToSkip[0].SourceProfile)
	}
}

// Test rationale content is properly preserved through PlanUpdate
func TestUpdatePlanner_RationaleContentPreserved(t *testing.T) {
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

	// Test without Force
	options := update.UpdateOptions{Force: false}
	plan, err := update.PlanUpdate(changeResults, mergedProfile, options, engine, "/project")

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Check skip rationale
	if len(plan.ToSkip) != 1 {
		t.Fatalf("expected 1 file in ToSkip, got %d", len(plan.ToSkip))
	}
	expectedRationale := "File has local modifications, skipping (use --force to overwrite)"
	if plan.ToSkip[0].Rationale != expectedRationale {
		t.Errorf("expected rationale '%s', got '%s'", expectedRationale, plan.ToSkip[0].Rationale)
	}

	// Test with Force
	optionsForce := update.UpdateOptions{Force: true}
	planForce, err := update.PlanUpdate(changeResults, mergedProfile, optionsForce, engine, "/project")

	if err != nil {
		t.Fatalf("expected no error with Force, got %v", err)
	}

	// Check update rationale with Force
	if len(planForce.ToUpdate) != 1 {
		t.Fatalf("expected 1 file in ToUpdate with Force, got %d", len(planForce.ToUpdate))
	}
	expectedForceRationale := "File has local modifications, overwriting due to --force"
	if planForce.ToUpdate[0].Rationale != expectedForceRationale {
		t.Errorf("expected rationale '%s', got '%s'", expectedForceRationale, planForce.ToUpdate[0].Rationale)
	}
}

// Test UpdatePlanner when only skip/delete files exist (no rendering needed)
func TestUpdatePlanner_OnlySkipAndDeleteFiles(t *testing.T) {
	t.Parallel()
	// Arrange - No template files needed since we only have skip and delete operations
	fs := testutil.CreateTestFilesystem(t, map[string]string{})
	engine := template.NewTemplateEngineWithFs(fs)

	p := &profile.Profile{
		Name:          "vendor/test",
		TemplateFiles: []profile.TemplateFile{},
	}
	mergedProfile := testutil.CreateTestMergedProfile(t, fs, "/profiles", p)

	changeResults := map[string]manifest.FileStatus{
		"modified.md":  manifest.FileStatusUserModified,
		"conflict.md":  manifest.FileStatusConflict,
		"unchanged.md": manifest.FileStatusUnchanged,
		"removed.md":   manifest.FileStatusRemoved,
	}

	options := update.UpdateOptions{Force: false, RemoveOrphans: true}

	// Act
	plan, err := update.PlanUpdate(changeResults, mergedProfile, options, engine, "/project")

	// Assert
	if err != nil {
		t.Fatalf("expected no error when only skip/delete files, got %v", err)
	}

	// No files to create or update
	if len(plan.ToCreate) != 0 {
		t.Errorf("expected 0 files in ToCreate, got %d", len(plan.ToCreate))
	}
	if len(plan.ToUpdate) != 0 {
		t.Errorf("expected 0 files in ToUpdate, got %d", len(plan.ToUpdate))
	}

	// UserModified and Conflict should be skipped
	if len(plan.ToSkip) != 2 {
		t.Errorf("expected 2 files in ToSkip, got %d", len(plan.ToSkip))
	}

	// Removed should be deleted (RemoveOrphans=true)
	if len(plan.ToDelete) != 1 {
		t.Errorf("expected 1 file in ToDelete, got %d", len(plan.ToDelete))
	}
}

// Test UpdatePlanner verifies TargetPath is correctly set in FileDecisions
func TestUpdatePlanner_TargetPathInDecisions(t *testing.T) {
	t.Parallel()
	// Arrange
	fs := testutil.CreateTestFilesystem(t, map[string]string{
		"/profiles/vendor/test/content/config/settings.json.tmpl": "{}",
	})
	engine := template.NewTemplateEngineWithFs(fs)

	p := &profile.Profile{
		Name: "vendor/test",
		TemplateFiles: []profile.TemplateFile{
			{SourcePath: "config/settings.json.tmpl", IsPartial: false},
		},
	}
	mergedProfile := testutil.CreateTestMergedProfile(t, fs, "/profiles", p)

	changeResults := map[string]manifest.FileStatus{
		"config/settings.json": manifest.FileStatusNew,
	}
	options := update.UpdateOptions{}

	// Act
	plan, err := update.PlanUpdate(changeResults, mergedProfile, options, engine, "/project")

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(plan.ToCreate) != 1 {
		t.Fatalf("expected 1 file in ToCreate, got %d", len(plan.ToCreate))
	}

	// Verify the target path includes the directory structure
	if plan.ToCreate[0].TargetPath != "config/settings.json" {
		t.Errorf("expected TargetPath 'config/settings.json', got '%s'", plan.ToCreate[0].TargetPath)
	}
}

// Test UpdatePlanner preserves OriginalStatus in FileDecisions
func TestUpdatePlanner_OriginalStatusPreserved(t *testing.T) {
	t.Parallel()
	// Arrange
	fs := testutil.CreateTestFilesystem(t, map[string]string{
		"/profiles/vendor/test/content/new.md.tmpl":     "New",
		"/profiles/vendor/test/content/changed.md.tmpl": "Changed",
	})
	engine := template.NewTemplateEngineWithFs(fs)

	p := &profile.Profile{
		Name: "vendor/test",
		TemplateFiles: []profile.TemplateFile{
			{SourcePath: "new.md.tmpl", IsPartial: false},
			{SourcePath: "changed.md.tmpl", IsPartial: false},
		},
	}
	mergedProfile := testutil.CreateTestMergedProfile(t, fs, "/profiles", p)

	changeResults := map[string]manifest.FileStatus{
		"new.md":     manifest.FileStatusNew,
		"changed.md": manifest.FileStatusSourceChanged,
	}
	options := update.UpdateOptions{}

	// Act
	plan, err := update.PlanUpdate(changeResults, mergedProfile, options, engine, "/project")

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Check ToCreate has New status
	if len(plan.ToCreate) != 1 {
		t.Fatalf("expected 1 file in ToCreate, got %d", len(plan.ToCreate))
	}
	if plan.ToCreate[0].OriginalStatus != manifest.FileStatusNew {
		t.Errorf("expected OriginalStatus FileStatusNew in ToCreate, got '%s'", plan.ToCreate[0].OriginalStatus)
	}

	// Check ToUpdate has SourceChanged status
	if len(plan.ToUpdate) != 1 {
		t.Fatalf("expected 1 file in ToUpdate, got %d", len(plan.ToUpdate))
	}
	if plan.ToUpdate[0].OriginalStatus != manifest.FileStatusSourceChanged {
		t.Errorf("expected OriginalStatus FileStatusSourceChanged in ToUpdate, got '%s'", plan.ToUpdate[0].OriginalStatus)
	}
}

// Test UpdatePlanner handles delete category with correct rationale
func TestUpdatePlanner_DeleteCategoryRationale(t *testing.T) {
	t.Parallel()
	// Arrange
	fs := testutil.CreateTestFilesystem(t, map[string]string{})
	engine := template.NewTemplateEngineWithFs(fs)

	p := &profile.Profile{
		Name:          "vendor/test",
		TemplateFiles: []profile.TemplateFile{},
	}
	mergedProfile := testutil.CreateTestMergedProfile(t, fs, "/profiles", p)

	changeResults := map[string]manifest.FileStatus{
		"orphan.md": manifest.FileStatusRemoved,
	}

	options := update.UpdateOptions{RemoveOrphans: true}

	// Act
	plan, err := update.PlanUpdate(changeResults, mergedProfile, options, engine, "/project")

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(plan.ToDelete) != 1 {
		t.Fatalf("expected 1 file in ToDelete, got %d", len(plan.ToDelete))
	}

	expectedRationale := "File no longer part of profile, will be deleted"
	if plan.ToDelete[0].Rationale != expectedRationale {
		t.Errorf("expected rationale '%s', got '%s'", expectedRationale, plan.ToDelete[0].Rationale)
	}

	if plan.ToDelete[0].OriginalStatus != manifest.FileStatusRemoved {
		t.Errorf("expected OriginalStatus FileStatusRemoved, got '%s'", plan.ToDelete[0].OriginalStatus)
	}
}

// Test UpdatePlanner handles all six statuses in a single call
func TestUpdatePlanner_AllSixStatusesInSingleCall(t *testing.T) {
	t.Parallel()
	// Arrange - Create templates only for files that need rendering
	fs := testutil.CreateTestFilesystem(t, map[string]string{
		"/profiles/vendor/test/content/new.md.tmpl":            "New file content",
		"/profiles/vendor/test/content/source-changed.md.tmpl": "Changed content",
		"/profiles/vendor/test/content/user-modified.md.tmpl":  "Modified content",
		"/profiles/vendor/test/content/conflict.md.tmpl":       "Conflict content",
	})
	engine := template.NewTemplateEngineWithFs(fs)

	p := &profile.Profile{
		Name: "vendor/test",
		TemplateFiles: []profile.TemplateFile{
			{SourcePath: "new.md.tmpl", IsPartial: false},
			{SourcePath: "source-changed.md.tmpl", IsPartial: false},
			{SourcePath: "user-modified.md.tmpl", IsPartial: false},
			{SourcePath: "conflict.md.tmpl", IsPartial: false},
		},
	}
	mergedProfile := testutil.CreateTestMergedProfile(t, fs, "/profiles", p)

	// All six statuses
	changeResults := map[string]manifest.FileStatus{
		"new.md":            manifest.FileStatusNew,
		"source-changed.md": manifest.FileStatusSourceChanged,
		"user-modified.md":  manifest.FileStatusUserModified,
		"conflict.md":       manifest.FileStatusConflict,
		"removed.md":        manifest.FileStatusRemoved,
		"unchanged.md":      manifest.FileStatusUnchanged,
	}

	// Default options
	options := update.UpdateOptions{}

	// Act
	plan, err := update.PlanUpdate(changeResults, mergedProfile, options, engine, "/project")

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// New -> ToCreate
	if len(plan.ToCreate) != 1 {
		t.Errorf("expected 1 file in ToCreate (New), got %d", len(plan.ToCreate))
	}

	// SourceChanged -> ToUpdate
	if len(plan.ToUpdate) != 1 {
		t.Errorf("expected 1 file in ToUpdate (SourceChanged), got %d", len(plan.ToUpdate))
	}

	// UserModified + Conflict -> ToSkip (Force=false)
	if len(plan.ToSkip) != 2 {
		t.Errorf("expected 2 files in ToSkip (UserModified + Conflict), got %d", len(plan.ToSkip))
	}

	// Removed -> excluded (RemoveOrphans=false)
	// Unchanged -> excluded
	// So ToDelete should be empty
	if len(plan.ToDelete) != 0 {
		t.Errorf("expected 0 files in ToDelete (RemoveOrphans=false), got %d", len(plan.ToDelete))
	}

	// Verify total files processed is 4 (New, SourceChanged, UserModified, Conflict)
	// Unchanged and Removed with RemoveOrphans=false are excluded
	totalProcessed := len(plan.ToCreate) + len(plan.ToUpdate) + len(plan.ToSkip) + len(plan.ToDelete)
	if totalProcessed != 4 {
		t.Errorf("expected 4 files in plan (excluding Unchanged and Removed), got %d", totalProcessed)
	}
}
