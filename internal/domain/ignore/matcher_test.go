package ignore

import "testing"

// Test 1: Match returns true for ignored files (exact match)
// This tests the basic interface contract: Match returns true when a path should be ignored.
func TestMatcher_Match_ReturnsTrue_ForIgnoredFiles(t *testing.T) {
	t.Parallel()
	// Arrange
	patterns := []string{"draft.md", "backup.txt"}
	matcher := NewPatternMatcher(patterns)

	// Act
	result := matcher.Match("draft.md", false)

	// Assert
	if !result {
		t.Error("expected Match to return true for ignored file 'draft.md', got false")
	}
}

// Test 2: Match returns false for non-ignored files
// This tests the basic interface contract: Match returns false when a path should not be ignored.
func TestMatcher_Match_ReturnsFalse_ForNonIgnoredFiles(t *testing.T) {
	t.Parallel()
	// Arrange
	patterns := []string{"draft.md", "backup.txt"}
	matcher := NewPatternMatcher(patterns)

	// Act
	result := matcher.Match("readme.md", false)

	// Assert
	if result {
		t.Error("expected Match to return false for non-ignored file 'readme.md', got true")
	}
}

// Test 3: MergeWith combines patterns correctly
// This tests that merging matchers preserves patterns from both matchers.
func TestMatcher_MergeWith_CombinesPatterns(t *testing.T) {
	t.Parallel()
	// Arrange
	parentPatterns := []string{"parent-ignore.md"}
	childPatterns := []string{"child-ignore.md"}
	parentMatcher := NewPatternMatcher(parentPatterns)
	childMatcher := NewPatternMatcher(childPatterns)

	// Act
	merged := parentMatcher.MergeWith(childMatcher)

	// Assert - both parent and child patterns should match
	if !merged.Match("parent-ignore.md", false) {
		t.Error("expected merged matcher to match parent pattern 'parent-ignore.md'")
	}
	if !merged.Match("child-ignore.md", false) {
		t.Error("expected merged matcher to match child pattern 'child-ignore.md'")
	}
	// Non-ignored file should still not match
	if merged.Match("readme.md", false) {
		t.Error("expected merged matcher to not match 'readme.md'")
	}
}

// Test 4: Empty matcher behavior (no patterns = nothing ignored)
// This tests that an empty matcher ignores nothing.
func TestMatcher_EmptyMatcher_IgnoresNothing(t *testing.T) {
	t.Parallel()
	// Arrange
	matcher := EmptyMatcher()

	// Act & Assert - various paths should all not be ignored
	testCases := []struct {
		path  string
		isDir bool
	}{
		{"readme.md", false},
		{"draft.md", false},
		{"backup/", true},
		{"node_modules", true},
		{"any/path/file.txt", false},
	}

	for _, tc := range testCases {
		if matcher.Match(tc.path, tc.isDir) {
			t.Errorf("expected empty matcher to return false for path %q (isDir=%v), got true",
				tc.path, tc.isDir)
		}
	}
}

// Test 5: MergeWith empty matcher returns original patterns
// This tests that merging with an empty matcher preserves the original matcher's behavior.
func TestMatcher_MergeWith_EmptyMatcher_PreservesOriginal(t *testing.T) {
	t.Parallel()
	// Arrange
	patterns := []string{"ignore-me.md"}
	original := NewPatternMatcher(patterns)
	empty := EmptyMatcher()

	// Act
	merged := original.MergeWith(empty)

	// Assert
	if !merged.Match("ignore-me.md", false) {
		t.Error("expected merged matcher to still match 'ignore-me.md' after merging with empty")
	}
	if merged.Match("readme.md", false) {
		t.Error("expected merged matcher to not match 'readme.md'")
	}
}

// Test 6: Empty matcher MergeWith non-empty returns non-empty patterns
// This tests merging an empty matcher with a non-empty one.
func TestMatcher_EmptyMatcher_MergeWith_NonEmpty(t *testing.T) {
	t.Parallel()
	// Arrange
	empty := EmptyMatcher()
	patterns := []string{"ignore-me.md"}
	nonEmpty := NewPatternMatcher(patterns)

	// Act
	merged := empty.MergeWith(nonEmpty)

	// Assert
	if !merged.Match("ignore-me.md", false) {
		t.Error("expected merged matcher to match 'ignore-me.md'")
	}
	if merged.Match("readme.md", false) {
		t.Error("expected merged matcher to not match 'readme.md'")
	}
}

// Test 7: MergeWith does not modify original matchers
// This tests that MergeWith is non-destructive.
func TestMatcher_MergeWith_DoesNotModifyOriginals(t *testing.T) {
	t.Parallel()
	// Arrange
	parentPatterns := []string{"parent-only.md"}
	childPatterns := []string{"child-only.md"}
	parentMatcher := NewPatternMatcher(parentPatterns)
	childMatcher := NewPatternMatcher(childPatterns)

	// Act
	_ = parentMatcher.MergeWith(childMatcher)

	// Assert - original matchers should be unchanged
	// Parent should still only match its own patterns
	if !parentMatcher.Match("parent-only.md", false) {
		t.Error("parent matcher should still match 'parent-only.md'")
	}
	if parentMatcher.Match("child-only.md", false) {
		t.Error("parent matcher should NOT match 'child-only.md' after merge")
	}

	// Child should still only match its own patterns
	if !childMatcher.Match("child-only.md", false) {
		t.Error("child matcher should still match 'child-only.md'")
	}
	if childMatcher.Match("parent-only.md", false) {
		t.Error("child matcher should NOT match 'parent-only.md'")
	}
}

// Test 8: NewPatternMatcher with nil patterns behaves like empty matcher
func TestMatcher_NewPatternMatcher_NilPatterns_BehavesLikeEmpty(t *testing.T) {
	t.Parallel()
	// Arrange
	matcher := NewPatternMatcher(nil)

	// Act & Assert
	if matcher.Match("any-file.md", false) {
		t.Error("expected matcher with nil patterns to not match any file")
	}
}
