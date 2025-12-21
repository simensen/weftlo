package ignore

import "strings"

// Pattern represents a parsed gitignore-style pattern with its modifiers detected.
// This struct captures the semantic meaning of a pattern line from an ignore file.
type Pattern struct {
	// Raw is the original pattern string as it appeared in the ignore file.
	Raw string

	// Pattern is the processed pattern string with modifiers removed.
	// For example: "!*.md" becomes "*.md", "/root.txt" becomes "root.txt"
	Pattern string

	// IsNegation indicates the pattern starts with "!" which re-includes
	// previously excluded files.
	IsNegation bool

	// IsRooted indicates the pattern starts with "/" which means it only
	// matches at the top level relative to the ignore file location.
	IsRooted bool

	// IsDirectoryOnly indicates the pattern ends with "/" which means it
	// only matches directories (when isDir is true in Match).
	IsDirectoryOnly bool

	// IsRecursive indicates the pattern contains "**" which matches
	// any number of directory levels.
	IsRecursive bool

	// Warning contains any parse warning for malformed patterns.
	// Empty string means the pattern parsed successfully.
	Warning string
}

// ParsePatterns parses a slice of pattern strings into structured Pattern objects.
// It handles gitignore syntax including:
//   - Comment lines (starting with #) - skipped
//   - Blank lines - skipped
//   - Negation patterns (starting with !)
//   - Rooted patterns (starting with /)
//   - Directory-only patterns (ending with /)
//   - Recursive patterns (containing **)
//
// Returns a slice of Pattern objects for valid patterns only.
// Patterns with warnings are still returned but with Warning field populated.
func ParsePatterns(lines []string) []Pattern {
	var patterns []Pattern

	for _, line := range lines {
		pattern := ParsePattern(line)
		if pattern == nil {
			// Line was a comment or blank - skip
			continue
		}
		patterns = append(patterns, *pattern)
	}

	return patterns
}

// ParsePattern parses a single pattern line from an ignore file.
// Returns nil if the line is a comment or blank line.
// Returns a Pattern struct for valid pattern lines.
func ParsePattern(line string) *Pattern {
	// Skip blank lines (including whitespace-only lines)
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return nil
	}

	// Skip comment lines (starting with #)
	if strings.HasPrefix(trimmed, "#") {
		return nil
	}

	pattern := &Pattern{
		Raw: line,
	}

	// Work with trimmed version for processing
	processed := trimmed

	// Detect negation (starts with !)
	if strings.HasPrefix(processed, "!") {
		pattern.IsNegation = true
		processed = strings.TrimPrefix(processed, "!")
	}

	// Detect directory-only (ends with /)
	if strings.HasSuffix(processed, "/") {
		pattern.IsDirectoryOnly = true
		// Keep the trailing slash for gitignore library compatibility
	}

	// Detect rooted (starts with /)
	if strings.HasPrefix(processed, "/") {
		pattern.IsRooted = true
		// Remove leading slash for pattern matching
		processed = strings.TrimPrefix(processed, "/")
	}

	// Detect recursive (contains **)
	if strings.Contains(processed, "**") {
		pattern.IsRecursive = true
	}

	// Check for malformed patterns
	warning := validatePattern(processed)
	if warning != "" {
		pattern.Warning = warning
	}

	pattern.Pattern = processed

	return pattern
}

// validatePattern checks for common pattern errors and returns a warning message.
// Returns empty string if pattern is valid.
func validatePattern(pattern string) string {
	// Check for unbalanced brackets
	if hasUnbalancedBrackets(pattern) {
		return "pattern has unbalanced brackets"
	}

	// Check for invalid escape sequences (backslash at end)
	if strings.HasSuffix(pattern, "\\") && !strings.HasSuffix(pattern, "\\\\") {
		return "pattern has trailing backslash"
	}

	return ""
}

// hasUnbalancedBrackets checks if a pattern has unbalanced [ ] brackets.
func hasUnbalancedBrackets(pattern string) bool {
	depth := 0
	escaped := false

	for _, ch := range pattern {
		if escaped {
			escaped = false
			continue
		}
		if ch == '\\' {
			escaped = true
			continue
		}
		switch ch {
		case '[':
			depth++
		case ']':
			if depth > 0 {
				depth--
			}
		}
	}

	return depth != 0
}

// ParsePatternsFromString parses patterns from a multi-line string (typical file content).
// Lines are split by newlines and each line is parsed as a pattern.
func ParsePatternsFromString(content string) []Pattern {
	lines := strings.Split(content, "\n")
	return ParsePatterns(lines)
}

// FilterValidPatterns filters a slice of patterns to only those without warnings.
// Patterns with non-empty Warning field are excluded.
func FilterValidPatterns(patterns []Pattern) []Pattern {
	var valid []Pattern
	for _, p := range patterns {
		if p.Warning == "" {
			valid = append(valid, p)
		}
	}
	return valid
}

// CollectWarnings extracts all warnings from a slice of patterns.
// Returns a slice of warning strings for patterns that had parse issues.
func CollectWarnings(patterns []Pattern) []string {
	var warnings []string
	for _, p := range patterns {
		if p.Warning != "" {
			warnings = append(warnings, p.Raw+": "+p.Warning)
		}
	}
	return warnings
}
