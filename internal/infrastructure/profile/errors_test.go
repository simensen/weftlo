package profile_test

import (
	"errors"
	"testing"

	infraprofile "github.com/simensen/weftlo/internal/infrastructure/profile"
)

// =============================================================================
// Error Type: ProfileNotFoundError
// =============================================================================

// Test ProfileNotFoundError message format includes profile name and path
func TestProfileNotFoundError_MessageFormat(t *testing.T) {
	t.Parallel()
	err := &infraprofile.ProfileNotFoundError{
		ProfileName: "acme/backend",
		Path:        "~/.weftlo/profiles/acme/backend/",
	}

	expectedMsg := "profile 'acme/backend' not found at ~/.weftlo/profiles/acme/backend/"
	actualMsg := err.Error()

	if actualMsg != expectedMsg {
		t.Errorf("expected error message '%s', got '%s'", expectedMsg, actualMsg)
	}
}

// =============================================================================
// Error Type: ProfileConfigParseError
// =============================================================================

// Test ProfileConfigParseError message format includes path and underlying error
func TestProfileConfigParseError_MessageFormat(t *testing.T) {
	t.Parallel()
	underlyingErr := errors.New("yaml: line 3: mapping values are not allowed in this context")
	err := &infraprofile.ProfileConfigParseError{
		Path: "/home/user/.weftlo/profiles/acme/backend/profile.yaml",
		Err:  underlyingErr,
	}

	expectedMsg := "failed to parse profile configuration /home/user/.weftlo/profiles/acme/backend/profile.yaml: yaml: line 3: mapping values are not allowed in this context"
	actualMsg := err.Error()

	if actualMsg != expectedMsg {
		t.Errorf("expected error message '%s', got '%s'", expectedMsg, actualMsg)
	}
}

// =============================================================================
// Error Type: FileEnumerationError
// =============================================================================

// Test FileEnumerationError message format includes path and error context
func TestFileEnumerationError_MessageFormat(t *testing.T) {
	t.Parallel()
	underlyingErr := errors.New("permission denied")
	err := &infraprofile.FileEnumerationError{
		Path: "/home/user/.weftlo/profiles/acme/backend/",
		Err:  underlyingErr,
	}

	expectedMsg := "failed to enumerate files in profile directory /home/user/.weftlo/profiles/acme/backend/: permission denied"
	actualMsg := err.Error()

	if actualMsg != expectedMsg {
		t.Errorf("expected error message '%s', got '%s'", expectedMsg, actualMsg)
	}
}

// =============================================================================
// Error Type: Wrapped Errors (Unwrap behavior)
// =============================================================================

// Test Unwrap() method returns underlying error for wrapped error types
func TestWrappedErrors_UnwrapReturnsUnderlyingError(t *testing.T) {
	t.Parallel()
	underlyingErr := errors.New("underlying error")

	// Test ProfileConfigParseError.Unwrap()
	parseErr := &infraprofile.ProfileConfigParseError{
		Path: "/some/path/profile.yaml",
		Err:  underlyingErr,
	}

	if parseErr.Unwrap() != underlyingErr {
		t.Errorf("ProfileConfigParseError.Unwrap() should return underlying error")
	}

	// Verify errors.Is works with wrapped error
	if !errors.Is(parseErr, underlyingErr) {
		t.Errorf("errors.Is should find underlying error in ProfileConfigParseError")
	}

	// Test FileEnumerationError.Unwrap()
	enumErr := &infraprofile.FileEnumerationError{
		Path: "/some/path/",
		Err:  underlyingErr,
	}

	if enumErr.Unwrap() != underlyingErr {
		t.Errorf("FileEnumerationError.Unwrap() should return underlying error")
	}

	// Verify errors.Is works with wrapped error
	if !errors.Is(enumErr, underlyingErr) {
		t.Errorf("errors.Is should find underlying error in FileEnumerationError")
	}
}

// =============================================================================
// Error Type: CircularDependencyError
// =============================================================================

// Test CircularDependencyError.Error() returns descriptive message with cycle information
func TestCircularDependencyError_MessageFormat(t *testing.T) {
	t.Parallel()
	err := &infraprofile.CircularDependencyError{
		ProfileName: "acme/child",
		Chain:       []string{"acme/grandparent", "acme/parent", "acme/child"},
	}

	expectedMsg := "circular dependency detected: profile 'acme/child' creates a cycle in chain [acme/grandparent, acme/parent, acme/child]"
	actualMsg := err.Error()

	if actualMsg != expectedMsg {
		t.Errorf("expected error message '%s', got '%s'", expectedMsg, actualMsg)
	}
}

// Test CircularDependencyError includes ProfileName and Chain fields
func TestCircularDependencyError_FieldsPopulated(t *testing.T) {
	t.Parallel()
	expectedProfileName := "acme/child"
	expectedChain := []string{"acme/grandparent", "acme/parent", "acme/child"}

	err := &infraprofile.CircularDependencyError{
		ProfileName: expectedProfileName,
		Chain:       expectedChain,
	}

	// Verify ProfileName field
	if err.ProfileName != expectedProfileName {
		t.Errorf("expected ProfileName '%s', got '%s'", expectedProfileName, err.ProfileName)
	}

	// Verify Chain field
	if len(err.Chain) != len(expectedChain) {
		t.Fatalf("expected Chain length %d, got %d", len(expectedChain), len(err.Chain))
	}

	for i, name := range expectedChain {
		if err.Chain[i] != name {
			t.Errorf("expected Chain[%d] to be '%s', got '%s'", i, name, err.Chain[i])
		}
	}
}

// =============================================================================
// Error Type: InheritanceDepthExceededError
// =============================================================================

// Test InheritanceDepthExceededError.Error() returns descriptive message with depth and chain
func TestInheritanceDepthExceededError_MessageFormat(t *testing.T) {
	t.Parallel()
	err := &infraprofile.InheritanceDepthExceededError{
		MaxDepth: 15,
		Chain:    []string{"level1/profile", "level2/profile", "level3/profile"},
	}

	expectedMsg := "inheritance depth exceeded: maximum depth of 15 reached with chain [level1/profile, level2/profile, level3/profile]"
	actualMsg := err.Error()

	if actualMsg != expectedMsg {
		t.Errorf("expected error message '%s', got '%s'", expectedMsg, actualMsg)
	}
}

// Test InheritanceDepthExceededError includes MaxDepth and Chain fields
func TestInheritanceDepthExceededError_FieldsPopulated(t *testing.T) {
	t.Parallel()
	expectedMaxDepth := 15
	expectedChain := []string{"level1/profile", "level2/profile", "level3/profile"}

	err := &infraprofile.InheritanceDepthExceededError{
		MaxDepth: expectedMaxDepth,
		Chain:    expectedChain,
	}

	// Verify MaxDepth field
	if err.MaxDepth != expectedMaxDepth {
		t.Errorf("expected MaxDepth %d, got %d", expectedMaxDepth, err.MaxDepth)
	}

	// Verify Chain field
	if len(err.Chain) != len(expectedChain) {
		t.Fatalf("expected Chain length %d, got %d", len(expectedChain), len(err.Chain))
	}

	for i, name := range expectedChain {
		if err.Chain[i] != name {
			t.Errorf("expected Chain[%d] to be '%s', got '%s'", i, name, err.Chain[i])
		}
	}
}
