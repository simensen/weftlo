package ignore

import (
	"path/filepath"
	"testing"

	"github.com/spf13/afero"
)

// Task Group 3 Tests: File Loading and Inline Pattern Support
// These tests verify ignore file loading and inline pattern processing.

// Test 3.1.1: Loading patterns from .weftlo.ignore file
func TestLoader_LoadFromFile_ParsesPatterns(t *testing.T) {
	t.Parallel()
	// Arrange
	fs := afero.NewMemMapFs()
	ignoreContent := `# Comment line
*.bak
draft-*
!important.bak
`
	ignoreFilePath := "/profiles/vendor/test/.weftlo.ignore"
	if err := afero.WriteFile(fs, ignoreFilePath, []byte(ignoreContent), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	loader := NewLoader(fs)

	// Act
	result, err := loader.LoadFromFile(ignoreFilePath)

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify patterns are loaded correctly
	matcher := result.Matcher
	if !matcher.Match("backup.bak", false) {
		t.Error("expected '*.bak' pattern to match 'backup.bak'")
	}
	if !matcher.Match("draft-readme.md", false) {
		t.Error("expected 'draft-*' pattern to match 'draft-readme.md'")
	}
	if matcher.Match("important.bak", false) {
		t.Error("expected '!important.bak' negation to NOT match 'important.bak'")
	}
	if matcher.Match("readme.md", false) {
		t.Error("expected non-matching file 'readme.md' to not be ignored")
	}
}

// Test 3.1.2: Silently skipping missing ignore files (no error)
func TestLoader_LoadFromFile_MissingFile_ReturnsSilently(t *testing.T) {
	t.Parallel()
	// Arrange
	fs := afero.NewMemMapFs()
	loader := NewLoader(fs)

	// Act - load from non-existent file
	result, err := loader.LoadFromFile("/nonexistent/.weftlo.ignore")

	// Assert - should NOT return error, should return empty matcher
	if err != nil {
		t.Fatalf("expected no error for missing file, got: %v", err)
	}

	// Empty matcher should not match anything
	if result.Matcher.Match("anything.txt", false) {
		t.Error("empty matcher from missing file should not match anything")
	}

	// No warnings should be present for missing file
	if len(result.Warnings) > 0 {
		t.Errorf("expected no warnings for missing file, got: %v", result.Warnings)
	}
}

// Test 3.1.3: Malformed patterns produce warnings but don't fail
func TestLoader_LoadFromFile_MalformedPatterns_ProducesWarnings(t *testing.T) {
	t.Parallel()
	// Arrange
	fs := afero.NewMemMapFs()
	ignoreContent := `# Comment
*.md
[unclosed
valid-pattern.txt
trailing\
`
	ignoreFilePath := "/profiles/vendor/test/.weftlo.ignore"
	if err := afero.WriteFile(fs, ignoreFilePath, []byte(ignoreContent), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	loader := NewLoader(fs)

	// Act
	result, err := loader.LoadFromFile(ignoreFilePath)

	// Assert - should NOT fail
	if err != nil {
		t.Fatalf("expected no error despite malformed patterns, got: %v", err)
	}

	// Valid patterns should still work
	if !result.Matcher.Match("readme.md", false) {
		t.Error("expected '*.md' pattern to match 'readme.md'")
	}
	if !result.Matcher.Match("valid-pattern.txt", false) {
		t.Error("expected 'valid-pattern.txt' pattern to match")
	}

	// Should have warnings for malformed patterns
	if len(result.Warnings) == 0 {
		t.Error("expected warnings for malformed patterns, got none")
	}
}

// Test 3.1.4: Loading inline patterns from ProfileConfig
func TestLoader_LoadFromPatterns_ProcessesInlinePatterns(t *testing.T) {
	t.Parallel()
	// Arrange - simulate inline patterns from profile.yaml
	inlinePatterns := []string{
		"*.draft",
		"!important.draft",
		"temp/",
	}

	loader := NewLoader(afero.NewMemMapFs())

	// Act
	result := loader.LoadFromPatterns(inlinePatterns)

	// Assert
	matcher := result.Matcher
	if !matcher.Match("readme.draft", false) {
		t.Error("expected '*.draft' pattern to match 'readme.draft'")
	}
	if matcher.Match("important.draft", false) {
		t.Error("expected '!important.draft' negation to NOT match 'important.draft'")
	}
	if !matcher.Match("temp", true) {
		t.Error("expected 'temp/' pattern to match 'temp' directory")
	}
	if matcher.Match("temp", false) {
		t.Error("expected 'temp/' pattern to NOT match 'temp' file")
	}
}

// Test 3.1.5: Inline patterns merge after file-based patterns
func TestLoader_InlinePatternsAfterFilePatterns(t *testing.T) {
	t.Parallel()
	// Arrange
	fs := afero.NewMemMapFs()
	// File-based patterns exclude all .draft files
	fileContent := "*.draft\n"
	ignoreFilePath := "/profiles/vendor/test/.weftlo.ignore"
	if err := afero.WriteFile(fs, ignoreFilePath, []byte(fileContent), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Inline patterns re-include important.draft
	inlinePatterns := []string{"!important.draft"}

	loader := NewLoader(fs)

	// Act - load file patterns then inline patterns
	fileResult, err := loader.LoadFromFile(ignoreFilePath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	inlineResult := loader.LoadFromPatterns(inlinePatterns)

	// Merge: file patterns first, inline patterns second (inline can override file)
	merged := fileResult.Matcher.MergeWith(inlineResult.Matcher)

	// Assert - inline negation should override file pattern
	if !merged.Match("readme.draft", false) {
		t.Error("merged should exclude 'readme.draft' (from file patterns)")
	}
	if merged.Match("important.draft", false) {
		t.Error("merged should NOT exclude 'important.draft' (inline negation overrides)")
	}
}

// Test 3.1.6: Patterns are relative to ignore file location (base path handling)
func TestLoader_LoadFromFile_PatternsRelativeToFileLocation(t *testing.T) {
	t.Parallel()
	// Arrange
	fs := afero.NewMemMapFs()
	// Pattern with rooted path should be relative to ignore file's directory
	ignoreContent := `/local-only.md
*.bak
`
	// Ignore file is in a content subdirectory
	ignoreFilePath := "/profiles/vendor/test/content/subdir/.weftlo.ignore"
	if err := afero.WriteFile(fs, ignoreFilePath, []byte(ignoreContent), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	loader := NewLoader(fs)

	// Act
	result, err := loader.LoadFromFile(ignoreFilePath)

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// The base path should be stored with the result
	expectedBasePath := filepath.Dir(ignoreFilePath)
	if result.BasePath != expectedBasePath {
		t.Errorf("expected base path %q, got %q", expectedBasePath, result.BasePath)
	}
}

// Test: LoadFromPatterns with nil or empty patterns returns empty matcher
func TestLoader_LoadFromPatterns_EmptyInput_ReturnsEmptyMatcher(t *testing.T) {
	t.Parallel()
	loader := NewLoader(afero.NewMemMapFs())

	// Test with nil
	nilResult := loader.LoadFromPatterns(nil)
	if nilResult.Matcher.Match("anything", false) {
		t.Error("nil patterns should produce empty matcher")
	}

	// Test with empty slice
	emptyResult := loader.LoadFromPatterns([]string{})
	if emptyResult.Matcher.Match("anything", false) {
		t.Error("empty patterns should produce empty matcher")
	}
}

// Test: LoadFromPatterns collects warnings for malformed inline patterns
func TestLoader_LoadFromPatterns_MalformedPatterns_CollectsWarnings(t *testing.T) {
	t.Parallel()
	loader := NewLoader(afero.NewMemMapFs())

	patterns := []string{
		"valid.md",
		"[unclosed",
		"also-valid.txt",
	}

	result := loader.LoadFromPatterns(patterns)

	// Should have warnings
	if len(result.Warnings) == 0 {
		t.Error("expected warnings for malformed patterns")
	}

	// Valid patterns should still work
	if !result.Matcher.Match("valid.md", false) {
		t.Error("valid.md should match")
	}
	if !result.Matcher.Match("also-valid.txt", false) {
		t.Error("also-valid.txt should match")
	}
}

// ==================== Task Group 4: Gap-Filling Tests ====================

// Test 4.3.6: MergeResults function - combining multiple LoadResults
func TestMergeResults_CombinesMatchersAndWarnings(t *testing.T) {
	t.Parallel()
	fs := afero.NewMemMapFs()

	// Create first ignore file
	file1Content := `*.draft
[unclosed1
`
	file1Path := "/profile1/.weftlo.ignore"
	if err := afero.WriteFile(fs, file1Path, []byte(file1Content), 0644); err != nil {
		t.Fatalf("failed to create file1: %v", err)
	}

	// Create second ignore file
	file2Content := `!important.draft
backup/
[unclosed2
`
	file2Path := "/profile2/.weftlo.ignore"
	if err := afero.WriteFile(fs, file2Path, []byte(file2Content), 0644); err != nil {
		t.Fatalf("failed to create file2: %v", err)
	}

	loader := NewLoader(fs)

	// Load both files
	result1, err := loader.LoadFromFile(file1Path)
	if err != nil {
		t.Fatalf("unexpected error loading file1: %v", err)
	}

	result2, err := loader.LoadFromFile(file2Path)
	if err != nil {
		t.Fatalf("unexpected error loading file2: %v", err)
	}

	// Also create inline patterns
	inlineResult := loader.LoadFromPatterns([]string{"*.tmp", "[unclosed3"})

	// Merge all results
	merged := MergeResults(result1, result2, inlineResult)

	// Assert - matcher should combine all patterns with later-wins semantics
	if !merged.Matcher.Match("readme.draft", false) {
		t.Error("merged should match *.draft from file1")
	}
	if merged.Matcher.Match("important.draft", false) {
		t.Error("merged should NOT match important.draft (negation from file2)")
	}
	if !merged.Matcher.Match("backup", true) {
		t.Error("merged should match backup/ directory from file2")
	}
	if !merged.Matcher.Match("cache.tmp", false) {
		t.Error("merged should match *.tmp from inline")
	}

	// Assert - warnings should be combined from all results
	if len(merged.Warnings) != 3 {
		t.Errorf("expected 3 warnings (one from each source), got %d: %v",
			len(merged.Warnings), merged.Warnings)
	}

	// Assert - BasePath should be empty for merged results
	if merged.BasePath != "" {
		t.Errorf("expected empty BasePath for merged results, got %q", merged.BasePath)
	}
}

// Test 4.3.7: MergeResults with empty results
func TestMergeResults_EmptyInputs(t *testing.T) {
	t.Parallel()
	// Test with no arguments
	emptyMerge := MergeResults()
	if emptyMerge.Matcher.Match("anything", false) {
		t.Error("empty MergeResults should produce empty matcher")
	}
	if len(emptyMerge.Warnings) != 0 {
		t.Error("empty MergeResults should have no warnings")
	}

	// Test with single empty result
	loader := NewLoader(afero.NewMemMapFs())
	emptyResult := loader.LoadFromPatterns(nil)

	singleMerge := MergeResults(emptyResult)
	if singleMerge.Matcher.Match("anything", false) {
		t.Error("MergeResults with single empty result should produce empty matcher")
	}
}

// Test 4.3.8: Full infrastructure integration - simulating complete profile hierarchy loading
// Note: gitignore semantics do not allow re-including subdirectories of excluded directories.
func TestFullInfrastructureIntegration_ProfileHierarchy(t *testing.T) {
	t.Parallel()
	fs := afero.NewMemMapFs()

	// Parent profile ignore file
	parentIgnore := `# Parent profile ignores
*.draft
*.bak
`
	parentPath := "/profiles/vendor/parent/.weftlo.ignore"
	if err := afero.WriteFile(fs, parentPath, []byte(parentIgnore), 0644); err != nil {
		t.Fatalf("failed to create parent ignore: %v", err)
	}

	// Child profile ignore file (extends parent)
	childIgnore := `# Child profile ignores - override some parent patterns
!important.draft
*.tmp
experimental/
`
	childPath := "/profiles/vendor/child/.weftlo.ignore"
	if err := afero.WriteFile(fs, childPath, []byte(childIgnore), 0644); err != nil {
		t.Fatalf("failed to create child ignore: %v", err)
	}

	loader := NewLoader(fs)

	// Load parent patterns
	parentResult, err := loader.LoadFromFile(parentPath)
	if err != nil {
		t.Fatalf("failed to load parent: %v", err)
	}

	// Simulate parent inline patterns
	parentInline := loader.LoadFromPatterns([]string{"logs/"})

	// Load child patterns
	childResult, err := loader.LoadFromFile(childPath)
	if err != nil {
		t.Fatalf("failed to load child: %v", err)
	}

	// Simulate child inline patterns - add new pattern, not trying to re-include excluded subdirectory
	childInline := loader.LoadFromPatterns([]string{"cache/"})

	// Merge in order: parent file -> parent inline -> child file -> child inline
	merged := MergeResults(parentResult, parentInline, childResult, childInline)

	testCases := []struct {
		path     string
		isDir    bool
		expected bool
		desc     string
	}{
		// Parent file patterns
		{"readme.draft", false, true, "parent file *.draft still applies to non-negated"},
		{"old.bak", false, true, "parent file *.bak applies"},
		// Parent inline patterns
		{"logs", true, true, "parent inline logs/ applies"},
		// Child file overrides
		{"important.draft", false, false, "child file !important.draft overrides parent"},
		{"cache.tmp", false, true, "child file *.tmp applies"},
		{"experimental", true, true, "child file experimental/ applies"},
		// Child inline patterns
		{"cache", true, true, "child inline cache/ applies"},
		// Non-ignored files
		{"src/main.go", false, false, "source files not ignored"},
		{"README.md", false, false, "docs not ignored"},
	}

	for _, tc := range testCases {
		result := merged.Matcher.Match(tc.path, tc.isDir)
		if result != tc.expected {
			t.Errorf("%s: path=%q isDir=%v got %v, want %v",
				tc.desc, tc.path, tc.isDir, result, tc.expected)
		}
	}
}
