package cli_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/simensen/weftlo/internal/cli"
)

// =============================================================================
// Task Group 2: CLI Error Types Extension Tests
// =============================================================================

// Test 1: GlobalConfigNotFoundError message format and suggestion text
func TestGlobalConfigNotFoundError_MessageFormat(t *testing.T) {
	err := &cli.GlobalConfigNotFoundError{}

	errMsg := err.Error()

	// Verify message contains the key information
	if !strings.Contains(errMsg, "Global configuration not found") {
		t.Errorf("expected error message to contain 'Global configuration not found', got: %s", errMsg)
	}

	// Verify message contains the suggestion to run init
	if !strings.Contains(errMsg, "weftlo init") {
		t.Errorf("expected error message to contain suggestion 'weftlo init', got: %s", errMsg)
	}

	// Verify exact message format
	expectedMsg := "Global configuration not found. Run `weftlo init` to initialize."
	if errMsg != expectedMsg {
		t.Errorf("expected error message to be '%s', got: '%s'", expectedMsg, errMsg)
	}
}

// Test 2: ProfileResolutionError message format with resolution chain details
func TestProfileResolutionError_MessageFormat(t *testing.T) {
	err := &cli.ProfileResolutionError{
		FlagChecked:          true,
		ProjectConfigChecked: true,
		GlobalConfigChecked:  true,
	}

	errMsg := err.Error()

	// Verify message indicates profile could not be resolved
	if !strings.Contains(errMsg, "profile") {
		t.Errorf("expected error message to mention 'profile', got: %s", errMsg)
	}

	// Verify message lists resolution chain sources
	if !strings.Contains(errMsg, "--profile flag") {
		t.Errorf("expected error message to mention '--profile flag', got: %s", errMsg)
	}

	if !strings.Contains(errMsg, "project config") {
		t.Errorf("expected error message to mention 'project config', got: %s", errMsg)
	}

	if !strings.Contains(errMsg, "global config") {
		t.Errorf("expected error message to mention 'global config', got: %s", errMsg)
	}
}

// Test 3: FileConflictError message format listing conflicting files
func TestFileConflictError_MessageFormat(t *testing.T) {
	conflictingFiles := []string{
		".claude/settings.json",
		".claude/commands/test.md",
		"CLAUDE.md",
	}

	err := &cli.FileConflictError{
		ConflictingFiles: conflictingFiles,
	}

	errMsg := err.Error()

	// Verify message indicates file conflicts exist
	if !strings.Contains(errMsg, "conflict") {
		t.Errorf("expected error message to mention 'conflict', got: %s", errMsg)
	}

	// Verify all conflicting files are listed
	for _, file := range conflictingFiles {
		if !strings.Contains(errMsg, file) {
			t.Errorf("expected error message to list conflicting file '%s', got: %s", file, errMsg)
		}
	}

	// Verify message suggests --force flag
	if !strings.Contains(errMsg, "--force") {
		t.Errorf("expected error message to suggest '--force' flag, got: %s", errMsg)
	}
}

// Test 4: Error() and Unwrap() methods for all new error types
func TestNewErrorTypes_ErrorAndUnwrapMethods(t *testing.T) {
	t.Run("GlobalConfigNotFoundError", func(t *testing.T) {
		err := &cli.GlobalConfigNotFoundError{}

		// Test Error() returns non-empty string
		if err.Error() == "" {
			t.Error("expected Error() to return non-empty string")
		}

		// GlobalConfigNotFoundError has no underlying error, so no Unwrap needed
		// Just verify it satisfies the error interface
		var _ error = err
	})

	t.Run("ProfileResolutionError_WithWrappedError", func(t *testing.T) {
		underlyingErr := errors.New("underlying error")
		err := &cli.ProfileResolutionError{
			FlagChecked:          true,
			ProjectConfigChecked: true,
			GlobalConfigChecked:  true,
			Err:                  underlyingErr,
		}

		// Test Error() returns non-empty string
		if err.Error() == "" {
			t.Error("expected Error() to return non-empty string")
		}

		// Test Unwrap() returns the underlying error
		unwrapped := err.Unwrap()
		if unwrapped != underlyingErr {
			t.Errorf("expected Unwrap() to return underlying error, got: %v", unwrapped)
		}
	})

	t.Run("ProfileResolutionError_WithoutWrappedError", func(t *testing.T) {
		err := &cli.ProfileResolutionError{
			FlagChecked:          true,
			ProjectConfigChecked: true,
			GlobalConfigChecked:  true,
		}

		// Test Unwrap() returns nil when no underlying error
		unwrapped := err.Unwrap()
		if unwrapped != nil {
			t.Errorf("expected Unwrap() to return nil when no underlying error, got: %v", unwrapped)
		}
	})

	t.Run("FileConflictError", func(t *testing.T) {
		err := &cli.FileConflictError{
			ConflictingFiles: []string{"file1.txt", "file2.txt"},
		}

		// Test Error() returns non-empty string
		if err.Error() == "" {
			t.Error("expected Error() to return non-empty string")
		}

		// FileConflictError has no underlying error, so no Unwrap needed
		// Just verify it satisfies the error interface
		var _ error = err
	})

	t.Run("AllErrorsSatisfyErrorInterface", func(t *testing.T) {
		// Verify all error types satisfy the error interface
		var errs []error = []error{
			&cli.GlobalConfigNotFoundError{},
			&cli.ProfileResolutionError{},
			&cli.FileConflictError{ConflictingFiles: []string{}},
		}

		for i, err := range errs {
			if err == nil {
				t.Errorf("error at index %d should not be nil", i)
			}
		}
	})
}
