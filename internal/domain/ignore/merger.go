package ignore

// MergeMatchers merges multiple matchers into a single matcher.
// Patterns are combined in order, with later matchers' patterns
// taking precedence over earlier ones (last-rule-wins semantics).
//
// This function is useful for merging patterns from the profile hierarchy:
//
//	parent -> child -> grandchild
//
// Usage:
//
//	merged := MergeMatchers(parentMatcher, childMatcher, grandchildMatcher)
//
// A negation pattern in a later matcher can override an exclusion
// from an earlier matcher.
//
// Returns an empty matcher if no matchers are provided.
// Returns the single matcher if only one is provided.
func MergeMatchers(matchers ...Matcher) Matcher {
	if len(matchers) == 0 {
		return EmptyMatcher()
	}

	if len(matchers) == 1 {
		return matchers[0]
	}

	// Start with the first matcher and merge subsequent ones
	result := matchers[0]
	for i := 1; i < len(matchers); i++ {
		result = result.MergeWith(matchers[i])
	}

	return result
}

// MergePatternsInOrder creates a new matcher from multiple pattern sets,
// merging them in order. This is equivalent to creating individual matchers
// and then using MergeMatchers, but more efficient.
//
// Pattern sets are merged in order:
//   - patternSets[0] patterns are evaluated first
//   - patternSets[1] patterns are evaluated after patternSets[0]
//   - And so on...
//
// Later patterns override earlier patterns (last-rule-wins).
//
// Usage for profile hierarchy:
//
//	merged := MergePatternsInOrder(
//	    parentFilePatterns,    // from parent .weftlo.ignore
//	    parentInlinePatterns,  // from parent profile.yaml ignore: field
//	    childFilePatterns,     // from child .weftlo.ignore
//	    childInlinePatterns,   // from child profile.yaml ignore: field
//	)
//
// Returns an empty matcher if no pattern sets are provided.
func MergePatternsInOrder(patternSets ...[]string) Matcher {
	if len(patternSets) == 0 {
		return EmptyMatcher()
	}

	// Calculate total capacity
	totalLen := 0
	for _, patterns := range patternSets {
		totalLen += len(patterns)
	}

	// Combine all patterns into a single slice
	combined := make([]string, 0, totalLen)
	for _, patterns := range patternSets {
		combined = append(combined, patterns...)
	}

	return NewPatternMatcher(combined)
}
