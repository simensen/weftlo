package update_test

import (
	"testing"

	"github.com/simensen/weftlo/internal/app/update"
	"github.com/simensen/weftlo/internal/domain/manifest"
)

// =============================================================================
// Task Group 1: Core Data Structure Tests
// =============================================================================

// Test FileDecision field initialization with all required fields
func TestFileDecision_FieldInitialization(t *testing.T) {
	t.Parallel()
	decision := update.FileDecision{
		TargetPath:     ".gitconfig",
		OriginalStatus: manifest.FileStatusNew,
		Rationale:      "File will be created from profile template",
		SourceProfile:  "vendor/my-profile",
	}

	// Verify all fields are properly initialized
	if decision.TargetPath != ".gitconfig" {
		t.Errorf("expected TargetPath '.gitconfig', got '%s'", decision.TargetPath)
	}
	if decision.OriginalStatus != manifest.FileStatusNew {
		t.Errorf("expected OriginalStatus FileStatusNew, got '%s'", decision.OriginalStatus)
	}
	if decision.Rationale != "File will be created from profile template" {
		t.Errorf("expected Rationale 'File will be created from profile template', got '%s'", decision.Rationale)
	}
	if decision.SourceProfile != "vendor/my-profile" {
		t.Errorf("expected SourceProfile 'vendor/my-profile', got '%s'", decision.SourceProfile)
	}
}

// Test UpdatePlan categorical slices are properly initialized
func TestUpdatePlan_CategoricalSlicesInitialization(t *testing.T) {
	t.Parallel()
	plan := update.UpdatePlan{
		ToCreate: []update.FileDecision{
			{TargetPath: "new-file.md", OriginalStatus: manifest.FileStatusNew},
		},
		ToUpdate: []update.FileDecision{
			{TargetPath: "changed-file.md", OriginalStatus: manifest.FileStatusSourceChanged},
		},
		ToSkip: []update.FileDecision{
			{TargetPath: "modified-file.md", OriginalStatus: manifest.FileStatusUserModified},
			{TargetPath: "conflict-file.md", OriginalStatus: manifest.FileStatusConflict},
		},
		ToDelete: []update.FileDecision{
			{TargetPath: "removed-file.md", OriginalStatus: manifest.FileStatusRemoved},
		},
	}

	// Verify each slice has the expected number of elements
	if len(plan.ToCreate) != 1 {
		t.Errorf("expected ToCreate length 1, got %d", len(plan.ToCreate))
	}
	if len(plan.ToUpdate) != 1 {
		t.Errorf("expected ToUpdate length 1, got %d", len(plan.ToUpdate))
	}
	if len(plan.ToSkip) != 2 {
		t.Errorf("expected ToSkip length 2, got %d", len(plan.ToSkip))
	}
	if len(plan.ToDelete) != 1 {
		t.Errorf("expected ToDelete length 1, got %d", len(plan.ToDelete))
	}

	// Verify the correct files are in each category
	if plan.ToCreate[0].TargetPath != "new-file.md" {
		t.Errorf("expected ToCreate[0].TargetPath 'new-file.md', got '%s'", plan.ToCreate[0].TargetPath)
	}
	if plan.ToUpdate[0].OriginalStatus != manifest.FileStatusSourceChanged {
		t.Errorf("expected ToUpdate[0].OriginalStatus FileStatusSourceChanged, got '%s'", plan.ToUpdate[0].OriginalStatus)
	}
}

// Test that value types (not pointers) are used correctly for FileDecision
func TestFileDecision_ValueType(t *testing.T) {
	t.Parallel()
	// Create a FileDecision and copy it
	original := update.FileDecision{
		TargetPath:     "original.md",
		OriginalStatus: manifest.FileStatusNew,
		Rationale:      "Original rationale",
		SourceProfile:  "profile1",
	}

	// Copy by value
	copied := original

	// Modify the copy
	copied.TargetPath = "copied.md"
	copied.Rationale = "Modified rationale"

	// Original should be unchanged (proving value semantics)
	if original.TargetPath != "original.md" {
		t.Errorf("expected original.TargetPath to remain 'original.md', got '%s'", original.TargetPath)
	}
	if original.Rationale != "Original rationale" {
		t.Errorf("expected original.Rationale to remain 'Original rationale', got '%s'", original.Rationale)
	}
}

// Test that UpdatePlan slices use value types (not pointers) for FileDecision
func TestUpdatePlan_ValueTypeSlices(t *testing.T) {
	t.Parallel()
	plan := update.UpdatePlan{
		ToCreate: []update.FileDecision{
			{TargetPath: "file1.md"},
		},
	}

	// Modify an element in the slice
	plan.ToCreate[0].TargetPath = "modified.md"

	// The modification should be reflected (slice elements are values, but slice itself is a reference)
	if plan.ToCreate[0].TargetPath != "modified.md" {
		t.Errorf("expected slice modification to persist, got '%s'", plan.ToCreate[0].TargetPath)
	}

	// Create a new slice entry and append
	newDecision := update.FileDecision{TargetPath: "new.md"}
	plan.ToCreate = append(plan.ToCreate, newDecision)

	// Modifying newDecision after append should NOT affect the slice
	newDecision.TargetPath = "changed-after-append.md"
	if plan.ToCreate[1].TargetPath != "new.md" {
		t.Errorf("expected appended value to be independent, got '%s'", plan.ToCreate[1].TargetPath)
	}
}

// Test UpdateOptions default values implement safe behavior
func TestUpdateOptions_DefaultValues(t *testing.T) {
	t.Parallel()
	// Zero-value initialization should have safe defaults
	options := update.UpdateOptions{}

	// Force should default to false (safe: don't overwrite user modifications)
	if options.Force != false {
		t.Error("expected Force to default to false")
	}

	// RemoveOrphans should default to false (safe: don't delete files)
	if options.RemoveOrphans != false {
		t.Error("expected RemoveOrphans to default to false")
	}
}

// Test UpdateOptions can be explicitly configured
func TestUpdateOptions_ExplicitConfiguration(t *testing.T) {
	t.Parallel()
	options := update.UpdateOptions{
		Force:         true,
		RemoveOrphans: true,
	}

	if options.Force != true {
		t.Error("expected Force to be true when explicitly set")
	}
	if options.RemoveOrphans != true {
		t.Error("expected RemoveOrphans to be true when explicitly set")
	}
}
