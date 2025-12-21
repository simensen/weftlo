// Package routing_test contains tests for the routing package.
package routing_test

import (
	"strings"
	"testing"

	"github.com/simensen/weftlo/internal/app/routing"
)

// =============================================================================
// Task Group 1: Error Types and Interfaces Tests
// =============================================================================

// Test InvalidTargetError provides descriptive error message with reserved value context
func TestInvalidTargetError_ProvidesDescriptiveMessage(t *testing.T) {
	t.Parallel()
	err := &routing.InvalidTargetError{
		TargetName:    "myTarget",
		TargetValue:   "..",
		ReservedValue: "..",
		Reason:        "reserved path component that would break path resolution",
	}

	msg := err.Error()

	// Verify error message contains the reserved value
	if !strings.Contains(msg, "..") {
		t.Errorf("expected error message to contain reserved value '..', got: %s", msg)
	}

	// Verify error message contains the target name
	if !strings.Contains(msg, "myTarget") {
		t.Errorf("expected error message to contain target name 'myTarget', got: %s", msg)
	}

	// Verify error message contains context about why it's invalid
	if !strings.Contains(msg, "reserved") {
		t.Errorf("expected error message to explain reserved value context, got: %s", msg)
	}
}

// Test ConflictingTargetError includes conflicting paths in error message
func TestConflictingTargetError_IncludesConflictingPaths(t *testing.T) {
	t.Parallel()
	err := &routing.ConflictingTargetError{
		TargetName:      "direct",
		TargetPath:      "",
		ConflictingName: "claude",
		ConflictingPath: ".claude",
	}

	msg := err.Error()

	// Verify error message contains both target names
	if !strings.Contains(msg, "direct") {
		t.Errorf("expected error message to contain target name 'direct', got: %s", msg)
	}
	if !strings.Contains(msg, "claude") {
		t.Errorf("expected error message to contain conflicting target 'claude', got: %s", msg)
	}

	// Verify error message mentions conflict
	if !strings.Contains(msg, "conflict") {
		t.Errorf("expected error message to mention 'conflict', got: %s", msg)
	}
}

// Test PartialSuppressionError message explains why partials cannot be suppressed
func TestPartialSuppressionError_ExplainsWhyPartialsCannotBeSuppressed(t *testing.T) {
	t.Parallel()
	err := &routing.PartialSuppressionError{
		SourcePath: "project.yaml",
	}

	msg := err.Error()

	// Verify error message mentions _partials
	if !strings.Contains(msg, "_partials") {
		t.Errorf("expected error message to mention '_partials', got: %s", msg)
	}

	// Verify error message explains suppression is not allowed
	if !strings.Contains(msg, "suppress") || !strings.Contains(msg, "cannot") {
		t.Errorf("expected error message to explain suppression is not allowed, got: %s", msg)
	}

	// Verify error message mentions template rendering dependency
	if !strings.Contains(msg, "template") {
		t.Errorf("expected error message to mention template rendering dependency, got: %s", msg)
	}
}

// Test UnknownTargetCategoryError includes source path and available targets
func TestUnknownTargetCategoryError_IncludesSourcePathAndAvailableTargets(t *testing.T) {
	t.Parallel()
	err := &routing.UnknownTargetCategoryError{
		SourcePath:       "unknown/file.md",
		Category:         "unknown",
		AvailableTargets: []string{"claude", "cursor", "shared"},
	}

	msg := err.Error()

	// Verify error message contains the source path
	if !strings.Contains(msg, "unknown/file.md") {
		t.Errorf("expected error message to contain source path 'unknown/file.md', got: %s", msg)
	}

	// Verify error message contains the category that wasn't found
	if !strings.Contains(msg, "unknown") {
		t.Errorf("expected error message to contain category 'unknown', got: %s", msg)
	}

	// Verify error message lists available targets
	if !strings.Contains(msg, "claude") || !strings.Contains(msg, "cursor") || !strings.Contains(msg, "shared") {
		t.Errorf("expected error message to list available targets, got: %s", msg)
	}
}

// Test error types implement error interface correctly
func TestErrorTypes_ImplementErrorInterface(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name string
		err  error
	}{
		{
			name: "InvalidTargetError",
			err: &routing.InvalidTargetError{
				TargetName:    "test",
				TargetValue:   ".",
				ReservedValue: ".",
				Reason:        "reserved",
			},
		},
		{
			name: "ConflictingTargetError",
			err: &routing.ConflictingTargetError{
				TargetName:      "direct",
				TargetPath:      "",
				ConflictingName: "other",
				ConflictingPath: ".other",
			},
		},
		{
			name: "PartialSuppressionError",
			err: &routing.PartialSuppressionError{
				SourcePath: "project.yaml",
			},
		},
		{
			name: "UnknownTargetCategoryError",
			err: &routing.UnknownTargetCategoryError{
				SourcePath:       "unknown/file.md",
				Category:         "unknown",
				AvailableTargets: []string{"claude"},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Verify the error implements the error interface
			var _ error = tc.err

			// Verify Error() returns a non-empty string
			msg := tc.err.Error()
			if msg == "" {
				t.Errorf("%s.Error() returned empty string", tc.name)
			}
		})
	}
}
