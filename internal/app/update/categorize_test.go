package update_test

import (
	"testing"

	"github.com/simensen/weftlo/internal/app/update"
	"github.com/simensen/weftlo/internal/domain/manifest"
)

// =============================================================================
// Task Group 3: Decision Categorization Logic Tests
// =============================================================================

// Test New status files go to ToCreate category
func TestCategorizeFile_NewStatus_GoesToCreate(t *testing.T) {
	t.Parallel()
	options := update.UpdateOptions{}

	category, rationale := update.CategorizeFile(manifest.FileStatusNew, options)

	if category != update.CategoryCreate {
		t.Errorf("expected category CategoryCreate, got %s", category)
	}
	expectedRationale := "File will be created from profile template"
	if rationale != expectedRationale {
		t.Errorf("expected rationale '%s', got '%s'", expectedRationale, rationale)
	}
}

// Test SourceChanged status files go to ToUpdate category
func TestCategorizeFile_SourceChangedStatus_GoesToUpdate(t *testing.T) {
	t.Parallel()
	options := update.UpdateOptions{}

	category, rationale := update.CategorizeFile(manifest.FileStatusSourceChanged, options)

	if category != update.CategoryUpdate {
		t.Errorf("expected category CategoryUpdate, got %s", category)
	}
	expectedRationale := "Template source changed, file will be updated"
	if rationale != expectedRationale {
		t.Errorf("expected rationale '%s', got '%s'", expectedRationale, rationale)
	}
}

// Test UserModified status files go to ToSkip by default
func TestCategorizeFile_UserModifiedStatus_GoesToSkipByDefault(t *testing.T) {
	t.Parallel()
	options := update.UpdateOptions{Force: false}

	category, rationale := update.CategorizeFile(manifest.FileStatusUserModified, options)

	if category != update.CategorySkip {
		t.Errorf("expected category CategorySkip, got %s", category)
	}
	expectedRationale := "File has local modifications, skipping (use --force to overwrite)"
	if rationale != expectedRationale {
		t.Errorf("expected rationale '%s', got '%s'", expectedRationale, rationale)
	}
}

// Test Conflict status files go to ToSkip by default
func TestCategorizeFile_ConflictStatus_GoesToSkipByDefault(t *testing.T) {
	t.Parallel()
	options := update.UpdateOptions{Force: false}

	category, rationale := update.CategorizeFile(manifest.FileStatusConflict, options)

	if category != update.CategorySkip {
		t.Errorf("expected category CategorySkip, got %s", category)
	}
	expectedRationale := "Both source and local file changed, skipping (use --force to overwrite)"
	if rationale != expectedRationale {
		t.Errorf("expected rationale '%s', got '%s'", expectedRationale, rationale)
	}
}

// Test Removed status files excluded from ToDelete by default
func TestCategorizeFile_RemovedStatus_ExcludedByDefault(t *testing.T) {
	t.Parallel()
	options := update.UpdateOptions{RemoveOrphans: false}

	category, rationale := update.CategorizeFile(manifest.FileStatusRemoved, options)

	if category != update.CategoryNone {
		t.Errorf("expected category CategoryNone, got %s", category)
	}
	expectedRationale := "File no longer in profile, keeping (use --remove-orphans to delete)"
	if rationale != expectedRationale {
		t.Errorf("expected rationale '%s', got '%s'", expectedRationale, rationale)
	}
}

// Test Unchanged status files excluded from all categories
func TestCategorizeFile_UnchangedStatus_ExcludedFromAllCategories(t *testing.T) {
	t.Parallel()
	options := update.UpdateOptions{}

	category, rationale := update.CategorizeFile(manifest.FileStatusUnchanged, options)

	if category != update.CategoryNone {
		t.Errorf("expected category CategoryNone, got %s", category)
	}
	expectedRationale := "File is up to date, no action needed"
	if rationale != expectedRationale {
		t.Errorf("expected rationale '%s', got '%s'", expectedRationale, rationale)
	}
}

// Test Unchanged status files are still excluded even with Force=true
func TestCategorizeFile_UnchangedStatus_ExcludedEvenWithForce(t *testing.T) {
	t.Parallel()
	options := update.UpdateOptions{Force: true, RemoveOrphans: true}

	category, rationale := update.CategorizeFile(manifest.FileStatusUnchanged, options)

	if category != update.CategoryNone {
		t.Errorf("expected category CategoryNone for unchanged even with Force=true, got %s", category)
	}
	if rationale != "File is up to date, no action needed" {
		t.Errorf("expected unchanged rationale, got '%s'", rationale)
	}
}

// Test unknown/empty status defaults to CategoryNone with appropriate rationale
func TestCategorizeFile_UnknownStatus_DefaultsToNone(t *testing.T) {
	t.Parallel()
	options := update.UpdateOptions{}

	// Test with an unknown status value
	category, rationale := update.CategorizeFile(manifest.FileStatus("unknown"), options)

	if category != update.CategoryNone {
		t.Errorf("expected category CategoryNone for unknown status, got %s", category)
	}
	expectedRationale := "Unknown file status, no action taken"
	if rationale != expectedRationale {
		t.Errorf("expected rationale '%s', got '%s'", expectedRationale, rationale)
	}
}

// =============================================================================
// Task Group 4: Force Flag Behavior Tests
// =============================================================================

// Test Force=true moves UserModified from ToSkip to ToUpdate
func TestCategorizeFile_ForceFlag_UserModifiedGoesToUpdate(t *testing.T) {
	t.Parallel()
	options := update.UpdateOptions{Force: true}

	category, rationale := update.CategorizeFile(manifest.FileStatusUserModified, options)

	if category != update.CategoryUpdate {
		t.Errorf("expected Force=true to move UserModified to CategoryUpdate, got %s", category)
	}
	expectedRationale := "File has local modifications, overwriting due to --force"
	if rationale != expectedRationale {
		t.Errorf("expected rationale '%s', got '%s'", expectedRationale, rationale)
	}
}

// Test Force=true moves Conflict from ToSkip to ToUpdate
func TestCategorizeFile_ForceFlag_ConflictGoesToUpdate(t *testing.T) {
	t.Parallel()
	options := update.UpdateOptions{Force: true}

	category, rationale := update.CategorizeFile(manifest.FileStatusConflict, options)

	if category != update.CategoryUpdate {
		t.Errorf("expected Force=true to move Conflict to CategoryUpdate, got %s", category)
	}
	expectedRationale := "Both source and local file changed, overwriting due to --force"
	if rationale != expectedRationale {
		t.Errorf("expected rationale '%s', got '%s'", expectedRationale, rationale)
	}
}

// Test Force=true does NOT affect Removed files (RemoveOrphans=false)
func TestCategorizeFile_ForceFlag_DoesNotAffectRemovedFiles(t *testing.T) {
	t.Parallel()
	options := update.UpdateOptions{Force: true, RemoveOrphans: false}

	category, rationale := update.CategorizeFile(manifest.FileStatusRemoved, options)

	// Force should NOT move Removed files to any category; RemoveOrphans controls that
	if category != update.CategoryNone {
		t.Errorf("expected Force=true to NOT affect Removed files (category should be CategoryNone), got %s", category)
	}
	// The rationale should still indicate using --remove-orphans, not --force
	expectedRationale := "File no longer in profile, keeping (use --remove-orphans to delete)"
	if rationale != expectedRationale {
		t.Errorf("expected rationale '%s', got '%s'", expectedRationale, rationale)
	}
}

// Test Force=true combined with RemoveOrphans=true still respects each flag's behavior
func TestCategorizeFile_ForceFlag_IndependentOfRemoveOrphans(t *testing.T) {
	t.Parallel()
	options := update.UpdateOptions{Force: true, RemoveOrphans: true}

	// UserModified should go to update due to Force
	category, rationale := update.CategorizeFile(manifest.FileStatusUserModified, options)
	if category != update.CategoryUpdate {
		t.Errorf("expected Force=true to move UserModified to CategoryUpdate with RemoveOrphans=true, got %s", category)
	}
	if rationale != "File has local modifications, overwriting due to --force" {
		t.Errorf("unexpected rationale: %s", rationale)
	}

	// Removed should go to delete due to RemoveOrphans, not affected by Force
	category, rationale = update.CategorizeFile(manifest.FileStatusRemoved, options)
	if category != update.CategoryDelete {
		t.Errorf("expected RemoveOrphans=true to move Removed to CategoryDelete, got %s", category)
	}
	if rationale != "File no longer part of profile, will be deleted" {
		t.Errorf("unexpected rationale for removed: %s", rationale)
	}
}

// Test Force=false maintains default skip behavior for UserModified
func TestCategorizeFile_ForceFlagFalse_UserModifiedSkipped(t *testing.T) {
	t.Parallel()
	options := update.UpdateOptions{Force: false}

	category, rationale := update.CategorizeFile(manifest.FileStatusUserModified, options)

	if category != update.CategorySkip {
		t.Errorf("expected Force=false to keep UserModified in CategorySkip, got %s", category)
	}
	expectedRationale := "File has local modifications, skipping (use --force to overwrite)"
	if rationale != expectedRationale {
		t.Errorf("expected rationale '%s', got '%s'", expectedRationale, rationale)
	}
}

// Test Force=false maintains default skip behavior for Conflict
func TestCategorizeFile_ForceFlagFalse_ConflictSkipped(t *testing.T) {
	t.Parallel()
	options := update.UpdateOptions{Force: false}

	category, rationale := update.CategorizeFile(manifest.FileStatusConflict, options)

	if category != update.CategorySkip {
		t.Errorf("expected Force=false to keep Conflict in CategorySkip, got %s", category)
	}
	expectedRationale := "Both source and local file changed, skipping (use --force to overwrite)"
	if rationale != expectedRationale {
		t.Errorf("expected rationale '%s', got '%s'", expectedRationale, rationale)
	}
}

// =============================================================================
// Task Group 5: RemoveOrphans Flag Behavior Tests
// =============================================================================

// Test RemoveOrphans=false excludes Removed files from ToDelete
func TestCategorizeFile_RemoveOrphansFlag_FalseExcludesRemovedFromDelete(t *testing.T) {
	t.Parallel()
	options := update.UpdateOptions{RemoveOrphans: false}

	category, rationale := update.CategorizeFile(manifest.FileStatusRemoved, options)

	if category != update.CategoryNone {
		t.Errorf("expected RemoveOrphans=false to exclude Removed files (CategoryNone), got %s", category)
	}
	expectedRationale := "File no longer in profile, keeping (use --remove-orphans to delete)"
	if rationale != expectedRationale {
		t.Errorf("expected rationale '%s', got '%s'", expectedRationale, rationale)
	}
}

// Test RemoveOrphans=true adds Removed files to ToDelete
func TestCategorizeFile_RemoveOrphansFlag_TrueAddsRemovedToDelete(t *testing.T) {
	t.Parallel()
	options := update.UpdateOptions{RemoveOrphans: true}

	category, rationale := update.CategorizeFile(manifest.FileStatusRemoved, options)

	if category != update.CategoryDelete {
		t.Errorf("expected RemoveOrphans=true to add Removed files to CategoryDelete, got %s", category)
	}
	expectedRationale := "File no longer part of profile, will be deleted"
	if rationale != expectedRationale {
		t.Errorf("expected rationale '%s', got '%s'", expectedRationale, rationale)
	}
}

// Test RemoveOrphans is independent from Force flag - Force does not affect Removed files
func TestCategorizeFile_RemoveOrphansFlag_IndependentFromForce(t *testing.T) {
	t.Parallel()
	// With Force=true but RemoveOrphans=false, Removed files should NOT go to delete
	optionsForceOnly := update.UpdateOptions{Force: true, RemoveOrphans: false}
	category, rationale := update.CategorizeFile(manifest.FileStatusRemoved, optionsForceOnly)

	if category != update.CategoryNone {
		t.Errorf("expected Force=true without RemoveOrphans to keep Removed as CategoryNone, got %s", category)
	}
	expectedRationale := "File no longer in profile, keeping (use --remove-orphans to delete)"
	if rationale != expectedRationale {
		t.Errorf("expected rationale '%s', got '%s'", expectedRationale, rationale)
	}

	// With Force=false but RemoveOrphans=true, Removed files should go to delete
	optionsRemoveOrphansOnly := update.UpdateOptions{Force: false, RemoveOrphans: true}
	category, rationale = update.CategorizeFile(manifest.FileStatusRemoved, optionsRemoveOrphansOnly)

	if category != update.CategoryDelete {
		t.Errorf("expected RemoveOrphans=true without Force to add Removed to CategoryDelete, got %s", category)
	}
	expectedRationale = "File no longer part of profile, will be deleted"
	if rationale != expectedRationale {
		t.Errorf("expected rationale '%s', got '%s'", expectedRationale, rationale)
	}
}

// Test RemoveOrphans does not affect other file statuses
func TestCategorizeFile_RemoveOrphansFlag_DoesNotAffectOtherStatuses(t *testing.T) {
	t.Parallel()
	options := update.UpdateOptions{RemoveOrphans: true}

	// New files should still go to create
	category, _ := update.CategorizeFile(manifest.FileStatusNew, options)
	if category != update.CategoryCreate {
		t.Errorf("expected RemoveOrphans=true to not affect New files, got %s", category)
	}

	// SourceChanged files should still go to update
	category, _ = update.CategorizeFile(manifest.FileStatusSourceChanged, options)
	if category != update.CategoryUpdate {
		t.Errorf("expected RemoveOrphans=true to not affect SourceChanged files, got %s", category)
	}

	// UserModified files should still go to skip (Force=false)
	category, _ = update.CategorizeFile(manifest.FileStatusUserModified, options)
	if category != update.CategorySkip {
		t.Errorf("expected RemoveOrphans=true to not affect UserModified files, got %s", category)
	}

	// Conflict files should still go to skip (Force=false)
	category, _ = update.CategorizeFile(manifest.FileStatusConflict, options)
	if category != update.CategorySkip {
		t.Errorf("expected RemoveOrphans=true to not affect Conflict files, got %s", category)
	}

	// Unchanged files should still be excluded
	category, _ = update.CategorizeFile(manifest.FileStatusUnchanged, options)
	if category != update.CategoryNone {
		t.Errorf("expected RemoveOrphans=true to not affect Unchanged files, got %s", category)
	}
}
