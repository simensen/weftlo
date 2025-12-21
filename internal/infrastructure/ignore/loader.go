// Package ignore provides infrastructure for loading ignore patterns from files
// and inline profile configurations. It wraps the domain ignore package with
// file system operations using afero for testability.
//
// This package follows patterns established in internal/infrastructure/config/resolver.go
// and internal/infrastructure/profile/loader.go:
//   - Afero filesystem abstraction for testability
//   - Silent handling of missing files (no error, no warning)
//   - Warning collection for malformed patterns without failing
//   - Functional options pattern for configuration
package ignore

import (
	"path/filepath"
	"strings"

	"github.com/spf13/afero"

	"github.com/simensen/weftlo/internal/domain/ignore"
)

// LoadResult contains the result of loading ignore patterns from a file or inline patterns.
// It includes the matcher, any warnings produced during parsing, and the base path
// for pattern relativity.
type LoadResult struct {
	// Matcher is the compiled pattern matcher for the loaded patterns.
	Matcher ignore.Matcher

	// Warnings contains any warnings produced during pattern parsing.
	// Each warning includes the pattern text and the reason for invalidity.
	// The loader continues processing valid patterns after encountering invalid ones.
	Warnings []string

	// BasePath is the directory containing the ignore file.
	// Patterns are relative to this path.
	// For inline patterns, this is empty string (patterns relative to profile root).
	BasePath string
}

// Loader provides methods for loading ignore patterns from files and inline configurations.
// It uses afero filesystem abstraction for testability.
type Loader struct {
	fs afero.Fs
}

// NewLoader creates a new Loader with the specified filesystem.
// For production use with the real filesystem:
//
//	loader := NewLoader(afero.NewOsFs())
//
// For testing with an in-memory filesystem:
//
//	loader := NewLoader(afero.NewMemMapFs())
func NewLoader(fs afero.Fs) *Loader {
	return &Loader{fs: fs}
}

// LoadFromFile loads ignore patterns from a file at the specified path.
// It implements the following behaviors:
//
//   - Missing files: Silently returns an empty matcher (no error, no warning)
//   - Malformed patterns: Produces warnings but continues processing valid patterns
//   - Pattern relativity: Sets BasePath to the directory containing the ignore file
//
// The returned LoadResult contains:
//   - Matcher: A compiled pattern matcher for all valid patterns
//   - Warnings: Any warnings for malformed patterns
//   - BasePath: The directory containing the ignore file
//
// This method follows error handling patterns from internal/infrastructure/profile/loader.go.
func (l *Loader) LoadFromFile(path string) (*LoadResult, error) {
	// Check if file exists
	exists, err := afero.Exists(l.fs, path)
	if err != nil {
		// File system error (e.g., permission denied) - return error
		return nil, err
	}

	// Missing files are silently skipped (no error, no warning)
	if !exists {
		return &LoadResult{
			Matcher:  ignore.EmptyMatcher(),
			Warnings: nil,
			BasePath: "",
		}, nil
	}

	// Read file contents
	data, err := afero.ReadFile(l.fs, path)
	if err != nil {
		return nil, err
	}

	// Parse file content into lines
	content := string(data)
	lines := strings.Split(content, "\n")

	// Parse patterns and collect warnings
	patterns := ignore.ParsePatterns(lines)
	warnings := ignore.CollectWarnings(patterns)

	// Extract raw pattern strings for matcher creation
	// We use the original lines (filtered for valid patterns) rather than Pattern.Pattern
	// because the gitignore library handles negation/rooted/etc syntax directly
	var rawPatterns []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		// Skip blank lines and comments
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		rawPatterns = append(rawPatterns, trimmed)
	}

	// Create matcher from raw patterns
	matcher := ignore.NewPatternMatcher(rawPatterns)

	// Base path is the directory containing the ignore file
	basePath := filepath.Dir(path)

	return &LoadResult{
		Matcher:  matcher,
		Warnings: warnings,
		BasePath: basePath,
	}, nil
}

// LoadFromPatterns loads ignore patterns from a slice of pattern strings.
// This is used for inline patterns defined in profile.yaml's ignore: field.
//
// The patterns are processed with the same semantics as file-based patterns:
//   - Comment lines (starting with #) are ignored
//   - Blank lines are ignored
//   - All gitignore pattern types are supported
//
// Returns an empty matcher for nil or empty pattern slices.
// The returned LoadResult has an empty BasePath since inline patterns
// are relative to the profile root by convention.
func (l *Loader) LoadFromPatterns(patterns []string) *LoadResult {
	// Return empty matcher for nil or empty input
	if len(patterns) == 0 {
		return &LoadResult{
			Matcher:  ignore.EmptyMatcher(),
			Warnings: nil,
			BasePath: "",
		}
	}

	// Parse patterns and collect warnings
	parsedPatterns := ignore.ParsePatterns(patterns)
	warnings := ignore.CollectWarnings(parsedPatterns)

	// Filter to valid pattern strings for matcher creation
	var rawPatterns []string
	for _, p := range patterns {
		trimmed := strings.TrimSpace(p)
		// Skip blank lines and comments
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		rawPatterns = append(rawPatterns, trimmed)
	}

	// Create matcher from raw patterns
	matcher := ignore.NewPatternMatcher(rawPatterns)

	return &LoadResult{
		Matcher:  matcher,
		Warnings: warnings,
		BasePath: "", // Inline patterns are relative to profile root
	}
}

// LoadFromFileWithBasePath loads ignore patterns from a file and validates
// that path matching considers the base path for pattern relativity.
// This is useful when loading ignore files from content subdirectories
// where patterns should apply relative to that subdirectory.
//
// The behavior is identical to LoadFromFile, but explicitly documents
// the base path handling for subdirectory ignore files.
func (l *Loader) LoadFromFileWithBasePath(path string) (*LoadResult, error) {
	return l.LoadFromFile(path)
}

// MergeResults merges multiple LoadResults into a single result.
// Patterns are merged in order, with later results' patterns taking precedence
// over earlier ones (last-rule-wins semantics).
//
// Warnings from all results are combined.
// The BasePath is set to empty string since merged results may come from
// multiple locations.
//
// Usage for profile hierarchy:
//
//	fileResult, _ := loader.LoadFromFile(ignoreFilePath)
//	inlineResult := loader.LoadFromPatterns(inlinePatterns)
//	merged := MergeResults(fileResult, inlineResult)
func MergeResults(results ...*LoadResult) *LoadResult {
	if len(results) == 0 {
		return &LoadResult{
			Matcher:  ignore.EmptyMatcher(),
			Warnings: nil,
			BasePath: "",
		}
	}

	// Start with first result's matcher
	merged := results[0].Matcher

	// Combine warnings
	var allWarnings []string
	if len(results[0].Warnings) > 0 {
		allWarnings = append(allWarnings, results[0].Warnings...)
	}

	// Merge subsequent results
	for i := 1; i < len(results); i++ {
		merged = merged.MergeWith(results[i].Matcher)
		if len(results[i].Warnings) > 0 {
			allWarnings = append(allWarnings, results[i].Warnings...)
		}
	}

	return &LoadResult{
		Matcher:  merged,
		Warnings: allWarnings,
		BasePath: "", // Merged results don't have a single base path
	}
}
