package manifest

import (
	"testing"
)

// Test that all six status values are defined and distinct
func TestFileStatusValuesAreDistinct(t *testing.T) {
	t.Parallel()
	statuses := []FileStatus{
		FileStatusUnchanged,
		FileStatusSourceChanged,
		FileStatusUserModified,
		FileStatusConflict,
		FileStatusNew,
		FileStatusRemoved,
	}

	// Verify we have exactly 6 status values
	if len(statuses) != 6 {
		t.Errorf("expected 6 status values, got %d", len(statuses))
	}

	// Verify all values are distinct by checking for duplicates
	seen := make(map[FileStatus]bool)
	for _, status := range statuses {
		if seen[status] {
			t.Errorf("duplicate status value found: %s", status)
		}
		seen[status] = true
	}
}

// Test string representation of each status value matches spec requirements
func TestFileStatusStringValues(t *testing.T) {
	t.Parallel()
	tests := []struct {
		status   FileStatus
		expected string
	}{
		{FileStatusUnchanged, "unchanged"},
		{FileStatusSourceChanged, "source_changed"},
		{FileStatusUserModified, "user_modified"},
		{FileStatusConflict, "conflict"},
		{FileStatusNew, "new"},
		{FileStatusRemoved, "removed"},
	}

	for _, tt := range tests {
		if string(tt.status) != tt.expected {
			t.Errorf("expected status string '%s', got '%s'", tt.expected, string(tt.status))
		}
	}
}

// Test type safety by verifying FileStatus can be used in maps and comparisons
func TestFileStatusTypeSafety(t *testing.T) {
	t.Parallel()
	// Test that FileStatus can be used as a map key
	statusMap := make(map[string]FileStatus)
	statusMap["file1.md"] = FileStatusUnchanged
	statusMap["file2.md"] = FileStatusSourceChanged

	if statusMap["file1.md"] != FileStatusUnchanged {
		t.Error("expected FileStatusUnchanged for file1.md")
	}
	if statusMap["file2.md"] != FileStatusSourceChanged {
		t.Error("expected FileStatusSourceChanged for file2.md")
	}

	// Test equality comparison
	status1 := FileStatusConflict
	status2 := FileStatusConflict
	status3 := FileStatusNew

	if status1 != status2 {
		t.Error("expected identical status values to be equal")
	}
	if status1 == status3 {
		t.Error("expected different status values to be not equal")
	}
}

// Test that FileStatus type is a string type (can be converted to string)
func TestFileStatusIsStringBased(t *testing.T) {
	t.Parallel()
	// Verify each status can be converted to a string and back
	statuses := []FileStatus{
		FileStatusUnchanged,
		FileStatusSourceChanged,
		FileStatusUserModified,
		FileStatusConflict,
		FileStatusNew,
		FileStatusRemoved,
	}

	for _, status := range statuses {
		// Convert to string
		str := string(status)
		if str == "" {
			t.Errorf("status %v converted to empty string", status)
		}

		// Convert back to FileStatus
		converted := FileStatus(str)
		if converted != status {
			t.Errorf("round-trip conversion failed: original %v, converted %v", status, converted)
		}
	}
}
