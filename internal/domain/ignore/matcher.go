// Package ignore provides domain interfaces and implementations for gitignore-style
// pattern matching. It enables filtering files during profile loading and installation
// using patterns defined in .weftlo.ignore files and inline profile.yaml ignore fields.
//
// # Library Selection
//
// This package uses github.com/sabhiram/go-gitignore as the underlying pattern matching
// library. The library was selected over alternatives (like go-git's gitignore package) for:
//
//   - Minimal dependency footprint: standalone library vs entire go-git ecosystem
//   - Simple string-based API: integrates well with our existing path handling
//   - Full gitignore semantics: negation (!), rooted (/), recursive (**), directory-only (/)
//   - Well-maintained: 487 importers, active maintenance, MIT license
//
// The isDir parameter in Match() is handled by appending "/" to directory paths before
// matching, which correctly triggers directory-only pattern matching (patterns ending in /).
//
// # Interface Design
//
// The Matcher interface follows patterns established in internal/domain/template/file_resolver.go:
//   - Clean interface definition with documentation
//   - Concrete implementations in the same package
//   - Map-based or slice-based internal storage
package ignore

import (
	"strings"

	gitignore "github.com/sabhiram/go-gitignore"
)

// Matcher defines the interface for matching paths against gitignore-style patterns.
// Implementations determine whether a given file or directory should be ignored (excluded)
// during profile installation.
//
// The interface supports pattern merging to enable the profile inheritance model where
// child profiles can extend or override parent profile ignore patterns.
//
// Usage with profile hierarchy:
//
//	parentMatcher := LoadFromFile(fs, parentIgnorePath)
//	childMatcher := LoadFromFile(fs, childIgnorePath)
//	merged := parentMatcher.MergeWith(childMatcher)
//
//	// Child patterns evaluated after parent patterns (last-rule-wins)
//	if merged.Match("draft/readme.md", false) {
//	    // File should be ignored
//	}
type Matcher interface {
	// Match determines whether the given path should be ignored.
	//
	// Parameters:
	//   - path: The file or directory path to check, using forward slashes as separators.
	//     Paths should be relative to the context where patterns were defined
	//     (e.g., relative to the profile root for profile-level ignore files).
	//   - isDir: Set to true if the path represents a directory. This enables
	//     correct matching of directory-only patterns (patterns ending with /).
	//
	// Returns:
	//   - true if the path matches any ignore pattern and should be excluded
	//   - false if the path does not match any pattern or matches a negation pattern
	//
	// Pattern evaluation follows gitignore semantics:
	//   - Later patterns override earlier patterns
	//   - Negation patterns (starting with !) can re-include previously excluded paths
	//   - Directory-only patterns (ending with /) only match when isDir is true
	Match(path string, isDir bool) bool

	// MergeWith creates a new Matcher that combines patterns from this matcher
	// with patterns from another matcher.
	//
	// The resulting matcher evaluates patterns in order:
	//   1. First, patterns from this (the receiver) matcher
	//   2. Then, patterns from the other matcher
	//
	// This ordering implements the "last rule wins" semantic of gitignore, allowing
	// child profiles to override parent patterns. A child profile's negation pattern
	// can re-include files that were excluded by a parent profile's pattern.
	//
	// Example:
	//   parent has: "*.draft.md" (exclude all draft files)
	//   child has: "!important.draft.md" (re-include specific draft file)
	//   merged.Match("important.draft.md", false) returns false (not ignored)
	//
	// Returns a new Matcher; neither the receiver nor other are modified.
	MergeWith(other Matcher) Matcher
}

// EmptyMatcher returns a Matcher that matches nothing (ignores no files).
// This is useful as a base case for pattern merging or when no ignore patterns exist.
//
// An empty matcher will:
//   - Return false for all Match() calls
//   - Return the other matcher unchanged when used as receiver in MergeWith()
func EmptyMatcher() Matcher {
	return &patternMatcher{patterns: nil}
}

// patternMatcher is a concrete implementation of Matcher that stores
// raw pattern strings for merging capability.
//
// The patterns slice preserves the original pattern strings to enable
// MergeWith to concatenate patterns from multiple matchers. Pattern
// compilation to the underlying library format happens during Match().
type patternMatcher struct {
	// patterns contains the raw gitignore pattern strings.
	// Patterns are evaluated in order, with later patterns taking precedence.
	patterns []string

	// compiled caches the compiled gitignore matcher.
	// It is lazily initialized on first Match() call.
	compiled *gitignore.GitIgnore
}

// Match implements Matcher.Match by compiling patterns and checking the path.
func (m *patternMatcher) Match(path string, isDir bool) bool {
	if len(m.patterns) == 0 {
		return false
	}

	// Normalize the path
	normalizedPath := NormalizePath(path)
	if normalizedPath == "" {
		return false
	}

	// Lazily compile patterns on first use
	if m.compiled == nil {
		m.compiled = compilePatterns(m.patterns)
	}

	// For directory-only pattern matching, we need to test both with and without trailing slash.
	// The go-gitignore library handles directory patterns (ending with /) by checking if the
	// path ends with /. We append / to directory paths to trigger this behavior.
	if isDir {
		// Check if path matches as a directory
		dirPath := normalizedPath
		if !strings.HasSuffix(dirPath, "/") {
			dirPath = normalizedPath + "/"
		}
		// Try matching with trailing slash first (for directory-only patterns)
		if m.compiled.MatchesPath(dirPath) {
			return true
		}
	}

	// Check if path matches normally
	return m.compiled.MatchesPath(normalizedPath)
}

// MergeWith implements Matcher.MergeWith by concatenating pattern slices.
func (m *patternMatcher) MergeWith(other Matcher) Matcher {
	otherMatcher, ok := other.(*patternMatcher)
	if !ok {
		// If other is not a patternMatcher, return a copy of this matcher
		return &patternMatcher{patterns: copyPatterns(m.patterns)}
	}

	// Concatenate patterns: this matcher's patterns first, then other's
	merged := make([]string, 0, len(m.patterns)+len(otherMatcher.patterns))
	merged = append(merged, m.patterns...)
	merged = append(merged, otherMatcher.patterns...)

	return &patternMatcher{patterns: merged}
}

// copyPatterns creates a copy of a pattern slice.
func copyPatterns(patterns []string) []string {
	if patterns == nil {
		return nil
	}
	result := make([]string, len(patterns))
	copy(result, patterns)
	return result
}

// compilePatterns compiles pattern strings into a gitignore matcher.
// Uses the go-gitignore library for full gitignore semantics.
func compilePatterns(patterns []string) *gitignore.GitIgnore {
	// Filter out comments and blank lines before compiling
	var validPatterns []string
	for _, p := range patterns {
		trimmed := strings.TrimSpace(p)
		// Skip blank lines
		if trimmed == "" {
			continue
		}
		// Skip comment lines
		if strings.HasPrefix(trimmed, "#") {
			continue
		}
		validPatterns = append(validPatterns, trimmed)
	}

	// CompileIgnoreLines handles empty slices gracefully
	gi := gitignore.CompileIgnoreLines(validPatterns...)
	return gi
}

// NewPatternMatcher creates a new Matcher from the given pattern strings.
// Patterns follow gitignore syntax as documented in the package overview.
//
// This function is primarily used by the infrastructure layer (loader.go)
// to create matchers from parsed ignore file contents or inline patterns.
func NewPatternMatcher(patterns []string) Matcher {
	return &patternMatcher{patterns: copyPatterns(patterns)}
}

// NormalizePath normalizes a file path for consistent pattern matching.
// It ensures:
//   - Forward slashes are used as separators (consistent with enumeration.go)
//   - Leading/trailing slashes are handled appropriately
//   - Empty paths return empty string
//
// This function should be used on all paths before matching against patterns.
func NormalizePath(path string) string {
	if path == "" {
		return ""
	}

	// Convert backslashes to forward slashes for cross-platform consistency
	normalized := strings.ReplaceAll(path, "\\", "/")

	// Remove leading slash (paths should be relative)
	normalized = strings.TrimPrefix(normalized, "/")

	// Remove trailing slash (unless it's just "/" which becomes "")
	// Trailing slashes are handled by the isDir parameter, not in the path itself
	if len(normalized) > 0 && normalized != "/" {
		normalized = strings.TrimSuffix(normalized, "/")
	}

	return normalized
}
