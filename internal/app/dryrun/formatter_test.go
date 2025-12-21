package dryrun_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/afero"

	"github.com/simensen/weftlo/internal/app/dryrun"
)

// =============================================================================
// Task Group 2: DryRunFormatter Tests
// =============================================================================

// Test 1: Formatter outputs correct header for each file operation
func TestFormatter_OutputsHeaderForEachOperation(t *testing.T) {
	t.Parallel()
	fs := afero.NewMemMapFs()
	var buf bytes.Buffer
	formatter := dryrun.NewFormatter(&buf, fs)

	operations := []dryrun.FileOperation{
		dryrun.NewFileOperation("base-profile", "weftlo/settings.md", []byte("content"), dryrun.OperationTypeCreate),
	}

	err := formatter.Format(operations)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	output := buf.String()

	// Verify operation type is shown
	if !strings.Contains(output, "would create") {
		t.Errorf("expected output to contain 'would create', got: %s", output)
	}

	// Verify target path is shown
	if !strings.Contains(output, "weftlo/settings.md") {
		t.Errorf("expected output to contain target path, got: %s", output)
	}
}

// Test 2: Formatter displays source profile, target path, and content size
func TestFormatter_DisplaysSourceProfileTargetPathAndSize(t *testing.T) {
	t.Parallel()
	fs := afero.NewMemMapFs()
	var buf bytes.Buffer
	formatter := dryrun.NewFormatter(&buf, fs)

	// Create content and use len() to verify size in test
	content := []byte("this is some test content for formatter")
	operations := []dryrun.FileOperation{
		dryrun.NewFileOperation("my-custom-profile", ".claude/commands/test.md", content, dryrun.OperationTypeCreate),
	}

	err := formatter.Format(operations)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	output := buf.String()

	// Verify source profile is displayed
	if !strings.Contains(output, "my-custom-profile") {
		t.Errorf("expected output to contain source profile 'my-custom-profile', got: %s", output)
	}

	// Verify target path is displayed
	if !strings.Contains(output, ".claude/commands/test.md") {
		t.Errorf("expected output to contain target path, got: %s", output)
	}

	// Verify content size is displayed (content length is 39 bytes)
	if !strings.Contains(output, "39 B") {
		t.Errorf("expected output to contain size '39 B', got: %s", output)
	}
}

// Test 3: Formatter generates unified diff when existing file differs
func TestFormatter_GeneratesDiffWhenExistingFileDiffers(t *testing.T) {
	t.Parallel()
	fs := afero.NewMemMapFs()

	// Create an existing file with different content
	existingContent := "old content\nline two\n"
	err := afero.WriteFile(fs, "/project/weftlo/config.md", []byte(existingContent), 0644)
	if err != nil {
		t.Fatalf("failed to create existing file: %v", err)
	}

	var buf bytes.Buffer
	formatter := dryrun.NewFormatter(&buf, fs)

	newContent := "new content\nline two\nline three\n"
	operations := []dryrun.FileOperation{
		dryrun.NewFileOperation("base-profile", "/project/weftlo/config.md", []byte(newContent), dryrun.OperationTypeCreate),
	}

	err = formatter.Format(operations)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	output := buf.String()

	// Verify diff header is present
	if !strings.Contains(output, "---") || !strings.Contains(output, "+++") {
		t.Errorf("expected output to contain unified diff markers (--- and +++), got: %s", output)
	}

	// Verify diff shows content changes
	if !strings.Contains(output, "-old content") {
		t.Errorf("expected diff to show removed line '-old content', got: %s", output)
	}
	if !strings.Contains(output, "+new content") {
		t.Errorf("expected diff to show added line '+new content', got: %s", output)
	}
}

// Test 4: Formatter skips diff when no existing file present
func TestFormatter_SkipsDiffWhenNoExistingFile(t *testing.T) {
	t.Parallel()
	fs := afero.NewMemMapFs()
	var buf bytes.Buffer
	formatter := dryrun.NewFormatter(&buf, fs)

	operations := []dryrun.FileOperation{
		dryrun.NewFileOperation("base-profile", "/nonexistent/path/file.md", []byte("new content"), dryrun.OperationTypeCreate),
	}

	err := formatter.Format(operations)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	output := buf.String()

	// Should not contain diff markers when file doesn't exist
	// The output should show the operation but not a diff
	if strings.Contains(output, "@@") {
		t.Errorf("expected no diff hunk markers when file doesn't exist, got: %s", output)
	}
}

// Test 5: Summary line output shows correct count
func TestFormatter_SummaryLineOutput(t *testing.T) {
	t.Parallel()
	fs := afero.NewMemMapFs()
	var buf bytes.Buffer
	formatter := dryrun.NewFormatter(&buf, fs)

	operations := []dryrun.FileOperation{
		dryrun.NewFileOperation("profile1", "file1.md", []byte("content1"), dryrun.OperationTypeCreate),
		dryrun.NewFileOperation("profile2", "file2.md", []byte("content2"), dryrun.OperationTypeCreate),
		dryrun.NewFileOperation("profile3", "file3.md", []byte("content3"), dryrun.OperationTypeCreate),
	}

	err := formatter.Format(operations)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	output := buf.String()

	// Verify summary line is present with correct count
	if !strings.Contains(output, "Would create 3 files") {
		t.Errorf("expected summary 'Would create 3 files', got: %s", output)
	}
}

// Test 6: Formatter handles single file with correct grammar
func TestFormatter_SingleFileSummaryGrammar(t *testing.T) {
	t.Parallel()
	fs := afero.NewMemMapFs()
	var buf bytes.Buffer
	formatter := dryrun.NewFormatter(&buf, fs)

	operations := []dryrun.FileOperation{
		dryrun.NewFileOperation("profile", "single-file.md", []byte("content"), dryrun.OperationTypeCreate),
	}

	err := formatter.Format(operations)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	output := buf.String()

	// Verify singular grammar for single file
	if !strings.Contains(output, "Would create 1 file") {
		t.Errorf("expected summary 'Would create 1 file' (singular), got: %s", output)
	}
}

// =============================================================================
// Task Group 4: Additional Strategic Tests for Formatter
// =============================================================================

// Test 7: Formatter handles empty operations slice
func TestFormatter_EmptyOperationsSlice(t *testing.T) {
	t.Parallel()
	fs := afero.NewMemMapFs()
	var buf bytes.Buffer
	formatter := dryrun.NewFormatter(&buf, fs)

	operations := []dryrun.FileOperation{}

	err := formatter.Format(operations)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	output := buf.String()

	// Should show "Would create 0 files"
	if !strings.Contains(output, "Would create 0 files") {
		t.Errorf("expected summary 'Would create 0 files', got: %s", output)
	}
}

// Test 8: Formatter skips diff when existing file has identical content
func TestFormatter_SkipsDiffWhenIdenticalContent(t *testing.T) {
	t.Parallel()
	fs := afero.NewMemMapFs()

	// Create an existing file with identical content
	content := "same content\n"
	err := afero.WriteFile(fs, "/project/file.md", []byte(content), 0644)
	if err != nil {
		t.Fatalf("failed to create existing file: %v", err)
	}

	var buf bytes.Buffer
	formatter := dryrun.NewFormatter(&buf, fs)

	operations := []dryrun.FileOperation{
		dryrun.NewFileOperation("profile", "/project/file.md", []byte(content), dryrun.OperationTypeCreate),
	}

	err = formatter.Format(operations)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	output := buf.String()

	// Should not contain diff markers when content is identical
	if strings.Contains(output, "@@") {
		t.Errorf("expected no diff hunk markers when content is identical, got: %s", output)
	}

	// Should still show the operation header
	if !strings.Contains(output, "would create") {
		t.Errorf("expected output to contain operation header, got: %s", output)
	}
}
