package dryrun

import (
	"fmt"
	"strings"
)

// GenerateDiff generates a unified diff between existing and new content.
// It returns an empty string if the contents are identical.
//
// Parameters:
//   - existingContent: The current content of the file on disk
//   - newContent: The new content that would be written
//   - targetPath: The path used for diff header labels
//
// Returns:
//   - A string containing the unified diff, or empty string if no differences
//
// The diff format follows standard unified diff conventions:
//   - Header lines with "---" (old) and "+++" (new) labels
//   - Hunk headers with @@ markers showing line ranges
//   - Lines prefixed with "-" for removals, "+" for additions, " " for context
func GenerateDiff(existingContent, newContent []byte, targetPath string) string {
	oldStr := string(existingContent)
	newStr := string(newContent)

	// If contents are identical, return empty string
	if oldStr == newStr {
		return ""
	}

	// Split content into lines
	oldLines := splitLines(oldStr)
	newLines := splitLines(newStr)

	// Build the diff using a simple line-by-line algorithm
	diff := generateUnifiedDiff(oldLines, newLines, targetPath)

	return diff
}

// splitLines splits a string into lines, preserving empty lines.
// Each line retains its content without the trailing newline.
func splitLines(s string) []string {
	if s == "" {
		return []string{}
	}

	// Split on newline, but handle trailing newline specially
	lines := strings.Split(s, "\n")

	// If the last element is empty (due to trailing newline), remove it
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}

	return lines
}

// generateUnifiedDiff creates a unified diff from old and new line slices.
// This is a simplified implementation that shows the full context.
func generateUnifiedDiff(oldLines, newLines []string, targetPath string) string {
	var result strings.Builder

	// Write diff headers
	result.WriteString(fmt.Sprintf("--- a/%s\n", targetPath))
	result.WriteString(fmt.Sprintf("+++ b/%s\n", targetPath))

	// Use the longest common subsequence algorithm to find differences
	lcs := computeLCS(oldLines, newLines)

	// Generate hunks based on the LCS
	hunks := generateHunks(oldLines, newLines, lcs)

	for _, hunk := range hunks {
		result.WriteString(hunk)
	}

	return result.String()
}

// editOp represents a single edit operation
type editOp struct {
	op   byte // ' ' for context, '-' for delete, '+' for insert
	line string
}

// computeLCS computes the longest common subsequence indices.
// Returns a slice of (oldIndex, newIndex) pairs that are common.
func computeLCS(old, new []string) [][2]int {
	m := len(old)
	n := len(new)

	// Build LCS length table
	dp := make([][]int, m+1)
	for i := range dp {
		dp[i] = make([]int, n+1)
	}

	for i := 1; i <= m; i++ {
		for j := 1; j <= n; j++ {
			if old[i-1] == new[j-1] {
				dp[i][j] = dp[i-1][j-1] + 1
			} else {
				if dp[i-1][j] > dp[i][j-1] {
					dp[i][j] = dp[i-1][j]
				} else {
					dp[i][j] = dp[i][j-1]
				}
			}
		}
	}

	// Backtrack to find the LCS
	var lcs [][2]int
	i, j := m, n
	for i > 0 && j > 0 {
		if old[i-1] == new[j-1] {
			lcs = append([][2]int{{i - 1, j - 1}}, lcs...)
			i--
			j--
		} else if dp[i-1][j] > dp[i][j-1] {
			i--
		} else {
			j--
		}
	}

	return lcs
}

// generateHunks generates unified diff hunks from old/new lines and their LCS.
func generateHunks(old, new []string, lcs [][2]int) []string {
	var hunks []string

	// Convert LCS to edit operations
	var ops []editOp

	oldIdx := 0
	newIdx := 0
	lcsIdx := 0

	for oldIdx < len(old) || newIdx < len(new) {
		if lcsIdx < len(lcs) {
			lcsOld := lcs[lcsIdx][0]
			lcsNew := lcs[lcsIdx][1]

			// Add deletions before this LCS element
			for oldIdx < lcsOld {
				ops = append(ops, editOp{'-', old[oldIdx]})
				oldIdx++
			}

			// Add insertions before this LCS element
			for newIdx < lcsNew {
				ops = append(ops, editOp{'+', new[newIdx]})
				newIdx++
			}

			// Add the common line
			ops = append(ops, editOp{' ', old[oldIdx]})
			oldIdx++
			newIdx++
			lcsIdx++
		} else {
			// No more LCS elements, add remaining deletions and insertions
			for oldIdx < len(old) {
				ops = append(ops, editOp{'-', old[oldIdx]})
				oldIdx++
			}
			for newIdx < len(new) {
				ops = append(ops, editOp{'+', new[newIdx]})
				newIdx++
			}
		}
	}

	// Group operations into hunks with context
	const contextLines = 3
	hunks = groupIntoHunks(ops, len(old), len(new), contextLines)

	return hunks
}

// groupIntoHunks groups edit operations into hunks with context lines.
func groupIntoHunks(ops []editOp, oldLen, newLen, context int) []string {
	if len(ops) == 0 {
		return nil
	}

	var hunks []string
	var currentHunk strings.Builder

	// Find ranges of changes
	type changeRange struct {
		start, end int // indices into ops
	}

	var changes []changeRange
	inChange := false
	changeStart := 0

	for i, op := range ops {
		if op.op != ' ' {
			if !inChange {
				inChange = true
				changeStart = i
			}
		} else {
			if inChange {
				inChange = false
				changes = append(changes, changeRange{changeStart, i})
			}
		}
	}
	if inChange {
		changes = append(changes, changeRange{changeStart, len(ops)})
	}

	if len(changes) == 0 {
		return nil
	}

	// Merge adjacent change ranges that are close together
	merged := []changeRange{changes[0]}
	for i := 1; i < len(changes); i++ {
		last := &merged[len(merged)-1]
		// If the gap between changes is small enough, merge them
		if changes[i].start-last.end <= context*2 {
			last.end = changes[i].end
		} else {
			merged = append(merged, changes[i])
		}
	}

	// Generate hunks for each merged range
	for _, cr := range merged {
		// Expand range to include context
		start := cr.start - context
		if start < 0 {
			start = 0
		}
		end := cr.end + context
		if end > len(ops) {
			end = len(ops)
		}

		// Calculate line numbers
		oldStart := 1
		newStart := 1
		for i := 0; i < start; i++ {
			if ops[i].op == ' ' || ops[i].op == '-' {
				oldStart++
			}
			if ops[i].op == ' ' || ops[i].op == '+' {
				newStart++
			}
		}

		oldCount := 0
		newCount := 0
		for i := start; i < end; i++ {
			if ops[i].op == ' ' || ops[i].op == '-' {
				oldCount++
			}
			if ops[i].op == ' ' || ops[i].op == '+' {
				newCount++
			}
		}

		// Write hunk header
		currentHunk.Reset()
		currentHunk.WriteString(fmt.Sprintf("@@ -%d,%d +%d,%d @@\n", oldStart, oldCount, newStart, newCount))

		// Write hunk content
		for i := start; i < end; i++ {
			currentHunk.WriteByte(ops[i].op)
			currentHunk.WriteString(ops[i].line)
			currentHunk.WriteByte('\n')
		}

		hunks = append(hunks, currentHunk.String())
	}

	return hunks
}
