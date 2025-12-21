package update_test

import (
	"errors"
	"testing"

	"github.com/simensen/weftlo/internal/app/update"
)

// =============================================================================
// Task Group 2: Error Handling Tests
// =============================================================================

// Test Error() method returns formatted message with target path
func TestUpdatePlanError_WithTargetPath(t *testing.T) {
	t.Parallel()
	err := &update.UpdatePlanError{
		TargetPath: ".gitconfig",
		Reason:     "failed to render template",
	}

	expected := "update plan error for '.gitconfig': failed to render template"
	if err.Error() != expected {
		t.Errorf("expected error message '%s', got '%s'", expected, err.Error())
	}
}

// Test Error() method handles empty target path gracefully
func TestUpdatePlanError_WithEmptyTargetPath(t *testing.T) {
	t.Parallel()
	err := &update.UpdatePlanError{
		TargetPath: "",
		Reason:     "invalid update options",
	}

	expected := "update plan error: invalid update options"
	if err.Error() != expected {
		t.Errorf("expected error message '%s', got '%s'", expected, err.Error())
	}
}

// Test error implements error interface and works with errors.As
func TestUpdatePlanError_ImplementsErrorInterface(t *testing.T) {
	t.Parallel()
	err := &update.UpdatePlanError{
		TargetPath: "config/settings.json",
		Reason:     "filesystem error during planning",
	}

	// Verify the error can be assigned to the error interface
	var e error = err
	if e == nil {
		t.Error("expected non-nil error")
	}

	// Verify it works with errors.As for type assertion
	var planErr *update.UpdatePlanError
	if !errors.As(e, &planErr) {
		t.Error("expected errors.As to succeed with UpdatePlanError")
	}

	if planErr.TargetPath != "config/settings.json" {
		t.Errorf("expected TargetPath 'config/settings.json', got '%s'", planErr.TargetPath)
	}

	if planErr.Reason != "filesystem error during planning" {
		t.Errorf("expected Reason 'filesystem error during planning', got '%s'", planErr.Reason)
	}
}
