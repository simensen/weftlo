package dryrun_test

import (
	"testing"

	"github.com/simensen/weftlo/internal/app/dryrun"
)

// =============================================================================
// Task Group 1: FileOperation Data Types Tests
// =============================================================================

// Test 1: FileOperation struct field access
func TestFileOperation_FieldAccess(t *testing.T) {
	t.Parallel()
	content := []byte("template content")
	op := dryrun.FileOperation{
		SourceProfile: "base-profile",
		TargetPath:    "weftlo/settings.md",
		Content:       content,
		OperationType: dryrun.OperationTypeCreate,
	}

	if op.SourceProfile != "base-profile" {
		t.Errorf("expected SourceProfile 'base-profile', got '%s'", op.SourceProfile)
	}
	if op.TargetPath != "weftlo/settings.md" {
		t.Errorf("expected TargetPath 'weftlo/settings.md', got '%s'", op.TargetPath)
	}
	if string(op.Content) != "template content" {
		t.Errorf("expected Content 'template content', got '%s'", string(op.Content))
	}
	if op.OperationType != dryrun.OperationTypeCreate {
		t.Errorf("expected OperationType OperationTypeCreate, got '%s'", op.OperationType)
	}
}

// Test 2: OperationType enum values and string representation
func TestOperationType_StringRepresentation(t *testing.T) {
	t.Parallel()
	tests := []struct {
		opType   dryrun.OperationType
		expected string
	}{
		{dryrun.OperationTypeCreate, "would create"},
		{dryrun.OperationTypeUpdate, "would update"},
		{dryrun.OperationTypeDelete, "would delete"},
		{dryrun.OperationType("unknown"), "unknown operation"},
	}

	for _, tc := range tests {
		result := tc.opType.String()
		if result != tc.expected {
			t.Errorf("OperationType(%s).String() = '%s', expected '%s'",
				string(tc.opType), result, tc.expected)
		}
	}
}

// Test 3: Human-readable size formatting
func TestFormatSize_HumanReadable(t *testing.T) {
	t.Parallel()
	tests := []struct {
		bytes    int64
		expected string
	}{
		{0, "0 B"},
		{256, "256 B"},
		{1023, "1023 B"},
		{1024, "1.0 KB"},
		{1536, "1.5 KB"},
		{1048576, "1.0 MB"},
		{1572864, "1.5 MB"},
		{1073741824, "1.0 GB"},
		{1610612736, "1.5 GB"},
	}

	for _, tc := range tests {
		result := dryrun.FormatSize(tc.bytes)
		if result != tc.expected {
			t.Errorf("FormatSize(%d) = '%s', expected '%s'", tc.bytes, result, tc.expected)
		}
	}
}

// Test 4: FormatSize handles edge cases (0 bytes, negative values)
func TestFormatSize_EdgeCases(t *testing.T) {
	t.Parallel()
	// Negative values should be treated as 0
	if result := dryrun.FormatSize(-1); result != "0 B" {
		t.Errorf("FormatSize(-1) = '%s', expected '0 B'", result)
	}
	if result := dryrun.FormatSize(-100); result != "0 B" {
		t.Errorf("FormatSize(-100) = '%s', expected '0 B'", result)
	}

	// Zero bytes
	if result := dryrun.FormatSize(0); result != "0 B" {
		t.Errorf("FormatSize(0) = '%s', expected '0 B'", result)
	}
}

// Test 5: FileOperation creation with various content sizes
func TestFileOperation_ContentSize(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name         string
		content      []byte
		expectedSize int64
	}{
		{"empty content", []byte(""), 0},
		{"small content", []byte("hello"), 5},
		{"larger content", []byte("this is a longer piece of content"), 33},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			op := dryrun.NewFileOperation("profile", "path.md", tc.content, dryrun.OperationTypeCreate)
			if op.ContentSize() != tc.expectedSize {
				t.Errorf("ContentSize() = %d, expected %d", op.ContentSize(), tc.expectedSize)
			}
		})
	}
}

// Test 6: NewFileOperation constructor creates correct struct
func TestNewFileOperation_Constructor(t *testing.T) {
	t.Parallel()
	content := []byte("rendered content")
	op := dryrun.NewFileOperation(
		"my-profile",
		".claude/settings.md",
		content,
		dryrun.OperationTypeCreate,
	)

	if op.SourceProfile != "my-profile" {
		t.Errorf("expected SourceProfile 'my-profile', got '%s'", op.SourceProfile)
	}
	if op.TargetPath != ".claude/settings.md" {
		t.Errorf("expected TargetPath '.claude/settings.md', got '%s'", op.TargetPath)
	}
	if string(op.Content) != "rendered content" {
		t.Errorf("expected Content 'rendered content', got '%s'", string(op.Content))
	}
	if op.OperationType != dryrun.OperationTypeCreate {
		t.Errorf("expected OperationType OperationTypeCreate, got '%s'", op.OperationType)
	}
}
