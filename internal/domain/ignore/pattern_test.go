package ignore

import "testing"

// Task Group 2 Tests: Pattern Parsing and Matching
// These tests verify gitignore-style pattern semantics.

// Test 2.1.1: Comment lines (starting with #) are ignored
func TestPatternMatcher_CommentLinesIgnored(t *testing.T) {
	t.Parallel()
	// Arrange - patterns include comments which should be skipped
	patterns := []string{
		"# This is a comment",
		"ignore-me.md",
		"#another comment",
		"  # comment with leading spaces",
	}
	matcher := NewPatternMatcher(patterns)

	// Act & Assert
	// The comment lines should not create patterns
	if !matcher.Match("ignore-me.md", false) {
		t.Error("expected 'ignore-me.md' to be ignored")
	}
	// Literal "# This is a comment" should NOT match (it was a comment)
	if matcher.Match("# This is a comment", false) {
		t.Error("comment line should not be treated as a pattern")
	}
}

// Test 2.1.2: Blank lines are ignored
func TestPatternMatcher_BlankLinesIgnored(t *testing.T) {
	t.Parallel()
	// Arrange - patterns include blank lines which should be skipped
	patterns := []string{
		"",
		"ignore-me.md",
		"   ",
		"	", // tab
		"also-ignore.md",
	}
	matcher := NewPatternMatcher(patterns)

	// Act & Assert
	if !matcher.Match("ignore-me.md", false) {
		t.Error("expected 'ignore-me.md' to be ignored")
	}
	if !matcher.Match("also-ignore.md", false) {
		t.Error("expected 'also-ignore.md' to be ignored")
	}
	// Empty string path should not match
	if matcher.Match("", false) {
		t.Error("empty path should not match")
	}
}

// Test 2.1.3: Glob patterns (*.bak, draft-*)
func TestPatternMatcher_GlobPatterns(t *testing.T) {
	t.Parallel()
	// Arrange
	patterns := []string{
		"*.bak",
		"draft-*",
		"*.tmp",
	}
	matcher := NewPatternMatcher(patterns)

	// Act & Assert
	testCases := []struct {
		path     string
		expected bool
		desc     string
	}{
		{"file.bak", true, "*.bak should match file.bak"},
		{"document.bak", true, "*.bak should match document.bak"},
		{"backup.txt", false, "*.bak should not match backup.txt"},
		{"draft-readme.md", true, "draft-* should match draft-readme.md"},
		{"draft-v2.txt", true, "draft-* should match draft-v2.txt"},
		{"mydraft-file.md", false, "draft-* should not match mydraft-file.md (prefix not matching)"},
		{"temp.tmp", true, "*.tmp should match temp.tmp"},
		{"subdir/file.bak", true, "*.bak should match in subdirectories"},
		{"deep/nested/draft-notes.md", true, "draft-* should match in deep subdirectories"},
	}

	for _, tc := range testCases {
		result := matcher.Match(tc.path, false)
		if result != tc.expected {
			t.Errorf("%s: got %v, want %v", tc.desc, result, tc.expected)
		}
	}
}

// Test 2.1.4: Negation patterns (!important.md)
func TestPatternMatcher_NegationPatterns(t *testing.T) {
	t.Parallel()
	// Arrange - exclude all .md files, but re-include important.md
	patterns := []string{
		"*.md",
		"!important.md",
	}
	matcher := NewPatternMatcher(patterns)

	// Act & Assert
	testCases := []struct {
		path     string
		expected bool
		desc     string
	}{
		{"readme.md", true, "*.md should exclude readme.md"},
		{"notes.md", true, "*.md should exclude notes.md"},
		{"important.md", false, "!important.md should re-include important.md"},
		{"file.txt", false, "file.txt should not be affected"},
	}

	for _, tc := range testCases {
		result := matcher.Match(tc.path, false)
		if result != tc.expected {
			t.Errorf("%s: got %v, want %v", tc.desc, result, tc.expected)
		}
	}
}

// Test 2.1.5: Directory-only patterns (trailing / like experimental/)
func TestPatternMatcher_DirectoryOnlyPatterns(t *testing.T) {
	t.Parallel()
	// Arrange - pattern ending with / should only match directories
	patterns := []string{
		"experimental/",
		"build/",
	}
	matcher := NewPatternMatcher(patterns)

	// Act & Assert
	testCases := []struct {
		path     string
		isDir    bool
		expected bool
		desc     string
	}{
		{"experimental", true, true, "experimental/ should match 'experimental' directory"},
		{"experimental", false, false, "experimental/ should NOT match 'experimental' file"},
		{"build", true, true, "build/ should match 'build' directory"},
		{"build", false, false, "build/ should NOT match 'build' file"},
		{"subdir/experimental", true, true, "experimental/ should match nested 'experimental' directory"},
		{"subdir/build", true, true, "build/ should match nested 'build' directory"},
	}

	for _, tc := range testCases {
		result := matcher.Match(tc.path, tc.isDir)
		if result != tc.expected {
			t.Errorf("%s: got %v, want %v", tc.desc, result, tc.expected)
		}
	}
}

// Test 2.1.6: Rooted patterns (leading / like /root-only.md)
func TestPatternMatcher_RootedPatterns(t *testing.T) {
	t.Parallel()
	// Arrange - patterns starting with / should only match at root level
	patterns := []string{
		"/root-only.md",
		"/config.yaml",
	}
	matcher := NewPatternMatcher(patterns)

	// Act & Assert
	testCases := []struct {
		path     string
		expected bool
		desc     string
	}{
		{"root-only.md", true, "/root-only.md should match 'root-only.md' at root"},
		{"config.yaml", true, "/config.yaml should match 'config.yaml' at root"},
		{"subdir/root-only.md", false, "/root-only.md should NOT match in subdirectory"},
		{"deep/nested/config.yaml", false, "/config.yaml should NOT match in nested directory"},
	}

	for _, tc := range testCases {
		result := matcher.Match(tc.path, false)
		if result != tc.expected {
			t.Errorf("%s: got %v, want %v", tc.desc, result, tc.expected)
		}
	}
}

// Test 2.1.7: Recursive patterns (**/test/)
func TestPatternMatcher_RecursivePatterns(t *testing.T) {
	t.Parallel()
	// Arrange - ** matches any number of directories
	patterns := []string{
		"**/test/",
		"**/node_modules/",
		"**/*.log",
	}
	matcher := NewPatternMatcher(patterns)

	// Act & Assert
	testCases := []struct {
		path     string
		isDir    bool
		expected bool
		desc     string
	}{
		{"test", true, true, "**/test/ should match 'test' at root"},
		{"src/test", true, true, "**/test/ should match 'src/test'"},
		{"deep/nested/test", true, true, "**/test/ should match 'deep/nested/test'"},
		{"test", false, false, "**/test/ should NOT match 'test' file (directory pattern)"},
		{"node_modules", true, true, "**/node_modules/ should match at root"},
		{"packages/ui/node_modules", true, true, "**/node_modules/ should match nested"},
		{"app.log", false, true, "**/*.log should match 'app.log'"},
		{"logs/error.log", false, true, "**/*.log should match 'logs/error.log'"},
		{"deep/nested/debug.log", false, true, "**/*.log should match deep nested .log file"},
	}

	for _, tc := range testCases {
		result := matcher.Match(tc.path, tc.isDir)
		if result != tc.expected {
			t.Errorf("%s: got %v, want %v", tc.desc, result, tc.expected)
		}
	}
}

// Test 2.1.8: isDir parameter affects directory-only pattern matching
func TestPatternMatcher_IsDirParameterAffectsMatching(t *testing.T) {
	t.Parallel()
	// Arrange - mix of directory-only and file patterns
	patterns := []string{
		"logs/",     // directory-only
		"*.log",     // file pattern (matches both files and directories)
		"temp/",     // directory-only
		"cache.dat", // exact file pattern
	}
	matcher := NewPatternMatcher(patterns)

	// Act & Assert - test same paths with different isDir values
	testCases := []struct {
		path     string
		isDir    bool
		expected bool
		desc     string
	}{
		// logs/ pattern tests
		{"logs", true, true, "logs/ matches 'logs' when isDir=true"},
		{"logs", false, false, "logs/ does NOT match 'logs' when isDir=false"},
		// *.log pattern tests (should match regardless of isDir for file patterns)
		{"error.log", false, true, "*.log matches 'error.log' file"},
		{"error.log", true, true, "*.log matches 'error.log' even as directory"},
		// temp/ pattern tests
		{"temp", true, true, "temp/ matches 'temp' when isDir=true"},
		{"temp", false, false, "temp/ does NOT match 'temp' when isDir=false"},
		// cache.dat pattern tests
		{"cache.dat", false, true, "cache.dat matches file"},
		{"cache.dat", true, true, "cache.dat matches directory with same name"},
	}

	for _, tc := range testCases {
		result := matcher.Match(tc.path, tc.isDir)
		if result != tc.expected {
			t.Errorf("%s: got %v, want %v", tc.desc, result, tc.expected)
		}
	}
}

// Test pattern merging with negation (verifies later patterns override earlier)
func TestPatternMatcher_MergeWithNegationOverride(t *testing.T) {
	t.Parallel()
	// Arrange - parent excludes all .draft files, child re-includes important one
	parentPatterns := []string{"*.draft.md"}
	childPatterns := []string{"!important.draft.md"}

	parentMatcher := NewPatternMatcher(parentPatterns)
	childMatcher := NewPatternMatcher(childPatterns)
	merged := parentMatcher.MergeWith(childMatcher)

	// Act & Assert
	testCases := []struct {
		path     string
		expected bool
		desc     string
	}{
		{"readme.draft.md", true, "merged should exclude readme.draft.md (from parent)"},
		{"notes.draft.md", true, "merged should exclude notes.draft.md (from parent)"},
		{"important.draft.md", false, "merged should NOT exclude important.draft.md (child negation)"},
	}

	for _, tc := range testCases {
		result := merged.Match(tc.path, false)
		if result != tc.expected {
			t.Errorf("%s: got %v, want %v", tc.desc, result, tc.expected)
		}
	}
}

// Test NormalizePath function
func TestNormalizePath(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		input    string
		expected string
		desc     string
	}{
		{"", "", "empty path stays empty"},
		{"file.txt", "file.txt", "simple filename unchanged"},
		{"dir/file.txt", "dir/file.txt", "forward slash path unchanged"},
		{"dir\\file.txt", "dir/file.txt", "backslash converted to forward slash"},
		{"/leading/slash", "leading/slash", "leading slash removed"},
		{"trailing/slash/", "trailing/slash", "trailing slash removed"},
		{"/both/slashes/", "both/slashes", "both leading and trailing slashes removed"},
		{"deep\\nested\\path\\file.txt", "deep/nested/path/file.txt", "multiple backslashes converted"},
		{"mixed/path\\with/slashes", "mixed/path/with/slashes", "mixed slashes normalized"},
	}

	for _, tc := range testCases {
		result := NormalizePath(tc.input)
		if result != tc.expected {
			t.Errorf("%s: got %q, want %q", tc.desc, result, tc.expected)
		}
	}
}

// Test pattern parser - ParsePattern function
func TestParsePattern(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		input           string
		expectedNil     bool
		isNegation      bool
		isRooted        bool
		isDirectoryOnly bool
		isRecursive     bool
		desc            string
	}{
		{"# comment", true, false, false, false, false, "comment returns nil"},
		{"  # comment with spaces", true, false, false, false, false, "comment with leading spaces returns nil"},
		{"", true, false, false, false, false, "empty line returns nil"},
		{"   ", true, false, false, false, false, "whitespace-only line returns nil"},
		{"*.md", false, false, false, false, false, "simple glob pattern"},
		{"!*.md", false, true, false, false, false, "negation pattern"},
		{"/root.txt", false, false, true, false, false, "rooted pattern"},
		{"dir/", false, false, false, true, false, "directory-only pattern"},
		{"**/test", false, false, false, false, true, "recursive pattern"},
		{"!/important.md", false, true, true, false, false, "negation + rooted"},
		{"**/build/", false, false, false, true, true, "recursive + directory-only"},
	}

	for _, tc := range testCases {
		result := ParsePattern(tc.input)
		if tc.expectedNil {
			if result != nil {
				t.Errorf("%s: expected nil, got pattern", tc.desc)
			}
			continue
		}
		if result == nil {
			t.Errorf("%s: expected pattern, got nil", tc.desc)
			continue
		}
		if result.IsNegation != tc.isNegation {
			t.Errorf("%s: IsNegation = %v, want %v", tc.desc, result.IsNegation, tc.isNegation)
		}
		if result.IsRooted != tc.isRooted {
			t.Errorf("%s: IsRooted = %v, want %v", tc.desc, result.IsRooted, tc.isRooted)
		}
		if result.IsDirectoryOnly != tc.isDirectoryOnly {
			t.Errorf("%s: IsDirectoryOnly = %v, want %v", tc.desc, result.IsDirectoryOnly, tc.isDirectoryOnly)
		}
		if result.IsRecursive != tc.isRecursive {
			t.Errorf("%s: IsRecursive = %v, want %v", tc.desc, result.IsRecursive, tc.isRecursive)
		}
	}
}

// Test pattern validation warnings
func TestPatternValidationWarnings(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		input      string
		hasWarning bool
		desc       string
	}{
		{"*.md", false, "valid pattern has no warning"},
		{"[abc", true, "unbalanced bracket has warning"},
		{"file\\", true, "trailing backslash has warning"},
		{"[a-z]", false, "balanced brackets no warning"},
		{"file\\\\", false, "escaped backslash no warning"},
	}

	for _, tc := range testCases {
		result := ParsePattern(tc.input)
		if result == nil {
			t.Errorf("%s: expected pattern, got nil", tc.desc)
			continue
		}
		hasWarning := result.Warning != ""
		if hasWarning != tc.hasWarning {
			t.Errorf("%s: hasWarning = %v, want %v (warning: %q)", tc.desc, hasWarning, tc.hasWarning, result.Warning)
		}
	}
}

// Test ParsePatterns function (multiple patterns)
func TestParsePatterns(t *testing.T) {
	t.Parallel()
	lines := []string{
		"# Header comment",
		"",
		"*.bak",
		"!important.bak",
		"  # Another comment",
		"build/",
	}

	patterns := ParsePatterns(lines)

	// Should have 3 patterns (comments and blank lines filtered out)
	if len(patterns) != 3 {
		t.Errorf("expected 3 patterns, got %d", len(patterns))
	}

	// Verify first pattern
	if patterns[0].Pattern != "*.bak" {
		t.Errorf("expected first pattern to be '*.bak', got %q", patterns[0].Pattern)
	}

	// Verify second pattern is negation
	if !patterns[1].IsNegation || patterns[1].Pattern != "important.bak" {
		t.Errorf("expected second pattern to be negation 'important.bak', got %+v", patterns[1])
	}

	// Verify third pattern is directory-only
	if !patterns[2].IsDirectoryOnly {
		t.Errorf("expected third pattern to be directory-only, got %+v", patterns[2])
	}
}

// Test MergeMatchers utility function
func TestMergeMatchers(t *testing.T) {
	t.Parallel()
	// Test empty input
	empty := MergeMatchers()
	if empty.Match("anything", false) {
		t.Error("empty merge should match nothing")
	}

	// Test single matcher
	single := NewPatternMatcher([]string{"*.md"})
	merged := MergeMatchers(single)
	if !merged.Match("file.md", false) {
		t.Error("single matcher merge should match *.md")
	}

	// Test multiple matchers with negation override
	parent := NewPatternMatcher([]string{"*.draft"})
	child := NewPatternMatcher([]string{"!important.draft"})
	merged = MergeMatchers(parent, child)
	if !merged.Match("readme.draft", false) {
		t.Error("merged should match readme.draft")
	}
	if merged.Match("important.draft", false) {
		t.Error("merged should NOT match important.draft (negation)")
	}
}

// Test MergePatternsInOrder utility function
func TestMergePatternsInOrder(t *testing.T) {
	t.Parallel()
	// Test empty input
	empty := MergePatternsInOrder()
	if empty.Match("anything", false) {
		t.Error("empty merge should match nothing")
	}

	// Test multiple pattern sets
	parentFile := []string{"*.draft"}
	parentInline := []string{"backup/"}
	childFile := []string{"!important.draft"}

	merged := MergePatternsInOrder(parentFile, parentInline, childFile)

	// Should match patterns from parent file
	if !merged.Match("readme.draft", false) {
		t.Error("merged should match readme.draft")
	}

	// Should match patterns from parent inline
	if !merged.Match("backup", true) {
		t.Error("merged should match backup directory")
	}

	// Child negation should override parent
	if merged.Match("important.draft", false) {
		t.Error("merged should NOT match important.draft (child negation)")
	}
}

// ==================== Task Group 4: Gap-Filling Tests ====================

// Test 4.3.1: Full merge chain integration (parent file -> parent inline -> child file -> child inline)
// This tests the complete merge order as specified: parent profile ignore -> profile root .weftlo.ignore -> profile inline ignore: -> child profile patterns
func TestFullMergeChainIntegration(t *testing.T) {
	t.Parallel()
	// Simulate a complete profile inheritance scenario:
	// - Grandparent file: excludes all *.draft files
	// - Parent file: excludes all *.bak files
	// - Parent inline: re-includes important.draft (overrides grandparent)
	// - Child file: excludes *.tmp files
	// - Child inline: excludes experimental/ directories
	//
	// Note: gitignore semantics do not allow re-including subdirectories of excluded directories.
	// See: "It is not possible to re-include a file if a parent directory of that file is excluded."

	grandparentFile := []string{"*.draft"}
	parentFile := []string{"*.bak"}
	parentInline := []string{"!important.draft"}
	childFile := []string{"*.tmp"}
	childInline := []string{"experimental/"}

	// Merge in order: grandparent -> parent file -> parent inline -> child file -> child inline
	merged := MergePatternsInOrder(grandparentFile, parentFile, parentInline, childFile, childInline)

	testCases := []struct {
		path     string
		isDir    bool
		expected bool
		desc     string
	}{
		// Grandparent pattern tests
		{"readme.draft", false, true, "grandparent excludes *.draft"},
		{"notes.draft", false, true, "grandparent excludes all .draft files"},
		// Parent inline overrides grandparent for files
		{"important.draft", false, false, "parent inline re-includes important.draft"},
		// Parent file patterns
		{"old.bak", false, true, "parent file excludes *.bak"},
		{"backup/data.bak", false, true, "parent file excludes nested .bak files"},
		// Child file pattern
		{"temp.tmp", false, true, "child file excludes *.tmp"},
		{"cache/data.tmp", false, true, "child file excludes nested .tmp files"},
		// Child inline directory pattern
		{"experimental", true, true, "child inline excludes experimental/ directory"},
		// Non-matched files should not be ignored
		{"readme.md", false, false, "non-matching file not ignored"},
		{"src/main.go", false, false, "source files not ignored"},
	}

	for _, tc := range testCases {
		result := merged.Match(tc.path, tc.isDir)
		if result != tc.expected {
			t.Errorf("%s: path=%q isDir=%v got %v, want %v",
				tc.desc, tc.path, tc.isDir, result, tc.expected)
		}
	}
}

// Test 4.3.2: Multiple negation patterns interacting
// Tests complex scenarios where multiple negation patterns interact across merge boundaries
func TestMultipleNegationPatternsInteracting(t *testing.T) {
	t.Parallel()
	// Scenario: Complex negation chain
	// 1. Exclude all *.md files
	// 2. Re-include README.md
	// 3. Exclude README.md again (later pattern wins)
	// 4. Re-include it again from child
	patterns := []string{
		"*.md",                // Exclude all .md
		"!README.md",          // Re-include README.md
		"README.md",           // Exclude README.md again (later pattern)
		"!README.md",          // Re-include README.md again (final state: not ignored)
		"*.draft.md",          // Exclude all .draft.md
		"!important.draft.md", // Re-include specific draft
	}
	matcher := NewPatternMatcher(patterns)

	testCases := []struct {
		path     string
		expected bool
		desc     string
	}{
		{"README.md", false, "README.md should be re-included (last negation wins)"},
		{"CHANGELOG.md", true, "CHANGELOG.md should be excluded (*.md)"},
		{"notes.md", true, "notes.md should be excluded (*.md)"},
		{"spec.draft.md", true, "spec.draft.md should be excluded (*.draft.md)"},
		{"important.draft.md", false, "important.draft.md should be re-included"},
		{"config.yaml", false, "non-.md files should not be affected"},
	}

	for _, tc := range testCases {
		result := matcher.Match(tc.path, false)
		if result != tc.expected {
			t.Errorf("%s: got %v, want %v", tc.desc, result, tc.expected)
		}
	}
}

// Test 4.3.3: Deeply nested directory matching with recursive patterns
func TestDeeplyNestedDirectoryMatching(t *testing.T) {
	t.Parallel()
	patterns := []string{
		"**/node_modules/",
		"**/vendor/",
		"**/.git/",
		"**/build/output/",
		"**/*.generated.*",
	}
	matcher := NewPatternMatcher(patterns)

	testCases := []struct {
		path     string
		isDir    bool
		expected bool
		desc     string
	}{
		// Very deep nesting tests
		{"a/b/c/d/e/f/node_modules", true, true, "6 levels deep node_modules matches"},
		{"packages/core/src/lib/utils/vendor", true, true, "deeply nested vendor matches"},
		{"projects/app/frontend/src/components/.git", true, true, "deeply nested .git matches"},
		{"monorepo/packages/ui/lib/dist/build/output", true, true, "deeply nested build/output matches"},
		{"src/modules/auth/handlers/user.generated.go", false, true, "deeply nested generated file matches"},
		// Non-matching paths
		{"a/b/c/d/e/f/node_modules", false, false, "file named node_modules should not match directory pattern"},
		{"not_node_modules", true, false, "similar but different name should not match"},
		{"my/vendor_backup", true, false, "partial name match should not match"},
	}

	for _, tc := range testCases {
		result := matcher.Match(tc.path, tc.isDir)
		if result != tc.expected {
			t.Errorf("%s: path=%q isDir=%v got %v, want %v",
				tc.desc, tc.path, tc.isDir, result, tc.expected)
		}
	}
}

// Test 4.3.4: Patterns with special characters
func TestPatternsWithSpecialCharacters(t *testing.T) {
	t.Parallel()
	patterns := []string{
		"*.min.js",        // Multiple dots
		"[Tt]est*",        // Character class
		"file[0-9].txt",   // Character range
		"data-*.json",     // Hyphen in pattern
		"log_[abc]_*.log", // Multiple special constructs
		"*.[jJ][sS]",      // Case-insensitive extension via char class
	}
	matcher := NewPatternMatcher(patterns)

	testCases := []struct {
		path     string
		expected bool
		desc     string
	}{
		{"app.min.js", true, "multiple dots pattern matches"},
		{"vendor.min.js", true, "another multiple dots match"},
		{"Test_suite.py", true, "[Tt]est* matches Test"},
		{"test_file.go", true, "[Tt]est* matches test"},
		{"contest.txt", false, "[Tt]est* should not match partial"},
		{"file1.txt", true, "file[0-9].txt matches file1.txt"},
		{"file9.txt", true, "file[0-9].txt matches file9.txt"},
		{"fileA.txt", false, "file[0-9].txt should not match fileA.txt"},
		{"data-users.json", true, "hyphen pattern matches"},
		{"data-config.json", true, "another hyphen pattern match"},
		{"log_a_debug.log", true, "multiple special constructs match"},
		{"log_b_error.log", true, "another multiple special match"},
		{"log_d_info.log", false, "log_[abc] should not match d"},
		{"app.js", true, "*.[jJ][sS] matches .js"},
		{"app.JS", true, "*.[jJ][sS] matches .JS"},
		{"app.Js", true, "*.[jJ][sS] matches .Js"},
	}

	for _, tc := range testCases {
		result := matcher.Match(tc.path, false)
		if result != tc.expected {
			t.Errorf("%s: path=%q got %v, want %v",
				tc.desc, tc.path, result, tc.expected)
		}
	}
}

// Test 4.3.5: Complex pattern combinations (mixed pattern types in single set)
// Note: gitignore semantics do not allow re-including subdirectories of excluded directories.
func TestComplexPatternCombinations(t *testing.T) {
	t.Parallel()
	// Real-world scenario: A comprehensive ignore file
	// Note: We cannot re-include build/release/ if build/ is excluded by directory pattern.
	// Instead we use file patterns that allow more flexibility with negation.
	patterns := []string{
		"# Build outputs - using file patterns for flexibility",
		"build/**", // Exclude contents of build
		"dist/**",  // Exclude contents of dist
		"*.o",
		"*.a",
		"",
		"# Keep important build artifacts",
		"!build/release/**", // Re-include release directory contents
		"!dist/final/**",    // Re-include final directory contents
		"",
		"# Logs but keep error logs",
		"**/*.log",
		"!**/*.error.log",
		"",
		"# Temp files",
		"**/tmp/",
		"*.tmp",
		"*.swp",
		"",
		"# IDE files",
		".idea/",
		".vscode/",
		"*.iml",
	}
	matcher := NewPatternMatcher(patterns)

	testCases := []struct {
		path     string
		isDir    bool
		expected bool
		desc     string
	}{
		// Build outputs (using ** pattern to match contents)
		{"build/debug/app", false, true, "build/** excludes build contents"},
		{"dist/staging/bundle.js", false, true, "dist/** excludes dist contents"},
		{"main.o", false, true, "*.o excluded"},
		{"lib.a", false, true, "*.a excluded"},
		// Negation for important artifacts
		{"build/release/app", false, false, "build/release/** re-included"},
		{"dist/final/bundle.js", false, false, "dist/final/** re-included"},
		// Logs with selective inclusion
		{"app.log", false, true, "*.log excluded"},
		{"logs/debug.log", false, true, "nested log excluded"},
		{"app.error.log", false, false, "*.error.log re-included"},
		{"logs/server.error.log", false, false, "nested error log re-included"},
		// Temp files
		{"tmp", true, true, "**/tmp/ directory excluded"},
		{"cache/tmp", true, true, "nested tmp directory excluded"},
		{"session.tmp", false, true, "*.tmp file excluded"},
		{"file.swp", false, true, "*.swp file excluded"},
		// IDE files
		{".idea", true, true, ".idea/ excluded"},
		{".vscode", true, true, ".vscode/ excluded"},
		{"project.iml", false, true, "*.iml excluded"},
		// Non-excluded
		{"src/main.go", false, false, "source files not excluded"},
		{"README.md", false, false, "docs not excluded"},
	}

	for _, tc := range testCases {
		result := matcher.Match(tc.path, tc.isDir)
		if result != tc.expected {
			t.Errorf("%s: path=%q isDir=%v got %v, want %v",
				tc.desc, tc.path, tc.isDir, result, tc.expected)
		}
	}
}
