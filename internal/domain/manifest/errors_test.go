package manifest

import (
	"errors"
	"testing"
)

// =============================================================================
// Error Type: ChangeDetectionError
// =============================================================================

// Test Error() method returns formatted message with target path
func TestChangeDetectionError_WithTargetPath(t *testing.T) {
	t.Parallel()
	err := &ChangeDetectionError{
		TargetPath: ".gitconfig",
		Reason:     "failed to read file",
	}

	expected := "change detection error for '.gitconfig': failed to read file"
	if err.Error() != expected {
		t.Errorf("expected error message '%s', got '%s'", expected, err.Error())
	}
}

// Test Error() method handles empty target path gracefully
func TestChangeDetectionError_WithEmptyTargetPath(t *testing.T) {
	t.Parallel()
	err := &ChangeDetectionError{
		TargetPath: "",
		Reason:     "invalid manifest state",
	}

	expected := "change detection error: invalid manifest state"
	if err.Error() != expected {
		t.Errorf("expected error message '%s', got '%s'", expected, err.Error())
	}
}

// Test error implements error interface
func TestChangeDetectionError_ImplementsErrorInterface(t *testing.T) {
	t.Parallel()
	err := &ChangeDetectionError{
		TargetPath: "config/settings.json",
		Reason:     "filesystem error",
	}

	// Verify the error can be assigned to the error interface
	var e error = err
	if e == nil {
		t.Error("expected non-nil error")
	}

	// Verify it works with errors.As for type assertion
	var detectionErr *ChangeDetectionError
	if !errors.As(e, &detectionErr) {
		t.Error("expected errors.As to succeed with ChangeDetectionError")
	}

	if detectionErr.TargetPath != "config/settings.json" {
		t.Errorf("expected TargetPath 'config/settings.json', got '%s'", detectionErr.TargetPath)
	}

	if detectionErr.Reason != "filesystem error" {
		t.Errorf("expected Reason 'filesystem error', got '%s'", detectionErr.Reason)
	}
}
