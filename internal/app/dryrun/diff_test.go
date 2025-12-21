package dryrun_test

import (
	"strings"
	"testing"

	"github.com/simensen/weftlo/internal/app/dryrun"
)

// =============================================================================
// Task Group 4: Additional Strategic Tests for diff.go
// =============================================================================

// Test 1: GenerateDiff returns empty string for identical content
func TestGenerateDiff_IdenticalContent(t *testing.T) {
	t.Parallel()
	content := []byte("line one\nline two\nline three\n")
	result := dryrun.GenerateDiff(content, content, "test/path.md")

	if result != "" {
		t.Errorf("expected empty string for identical content, got: %s", result)
	}
}

// Test 2: GenerateDiff handles empty existing content (new file scenario)
func TestGenerateDiff_EmptyExistingContent(t *testing.T) {
	t.Parallel()
	existing := []byte("")
	newContent := []byte("new content\n")
	result := dryrun.GenerateDiff(existing, newContent, "test/path.md")

	// Should show added lines
	if !strings.Contains(result, "+new content") {
		t.Errorf("expected diff to show added line, got: %s", result)
	}

	// Should have diff headers
	if !strings.Contains(result, "---") || !strings.Contains(result, "+++") {
		t.Errorf("expected diff headers, got: %s", result)
	}
}

// Test 3: GenerateDiff handles empty new content (delete scenario)
func TestGenerateDiff_EmptyNewContent(t *testing.T) {
	t.Parallel()
	existing := []byte("existing content\n")
	newContent := []byte("")
	result := dryrun.GenerateDiff(existing, newContent, "test/path.md")

	// Should show removed lines
	if !strings.Contains(result, "-existing content") {
		t.Errorf("expected diff to show removed line, got: %s", result)
	}
}

// Test 4: GenerateDiff produces valid unified diff format with context
func TestGenerateDiff_UnifiedDiffFormat(t *testing.T) {
	t.Parallel()
	existing := []byte("line 1\nline 2\nline 3\nline 4\nline 5\n")
	newContent := []byte("line 1\nline 2 modified\nline 3\nline 4\nline 5\n")
	result := dryrun.GenerateDiff(existing, newContent, "config/settings.md")

	// Verify unified diff format elements
	if !strings.Contains(result, "--- a/config/settings.md") {
		t.Errorf("expected '--- a/config/settings.md' header, got: %s", result)
	}

	if !strings.Contains(result, "+++ b/config/settings.md") {
		t.Errorf("expected '+++ b/config/settings.md' header, got: %s", result)
	}

	// Verify hunk header exists
	if !strings.Contains(result, "@@") {
		t.Errorf("expected hunk header with @@, got: %s", result)
	}

	// Verify changes are marked
	if !strings.Contains(result, "-line 2") {
		t.Errorf("expected removed line marker, got: %s", result)
	}
	if !strings.Contains(result, "+line 2 modified") {
		t.Errorf("expected added line marker, got: %s", result)
	}
}
