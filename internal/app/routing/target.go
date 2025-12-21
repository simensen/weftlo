// Package routing provides services for transforming content source paths to target installation paths.
// This file implements target resolution logic for path transformation.
package routing

import (
	"path"
	"strings"
)

// TargetResult contains the result of resolving a source path to a target path.
// It provides both the computed path and metadata about how the resolution was performed.
type TargetResult struct {
	// TargetPath is the resolved installation path relative to the project root.
	TargetPath string

	// Category is the source directory name that matched during resolution.
	// This is the first segment of the source path that was used to look up
	// the target mapping. Empty if the path was routed via default_target.
	Category string

	// UsedDefault is true if the path was routed through the default_target
	// because no explicit target mapping was found.
	UsedDefault bool

	// IsPartial is true if the resolved path is a partial template
	// (matched the _partials target category).
	IsPartial bool
}

// ResolveTarget transforms a source path (relative to content root) into a target
// installation path based on the provided targets map and default target.
//
// The resolution process:
//  1. Extract the first path segment from the source path
//  2. Match the segment against the targets map keys
//  3. If matched, replace the first segment with the target value
//  4. If not matched, prepend default_target and preserve the full path
//
// Parameters:
//   - sourcePath: path relative to content root (e.g., "claude/settings.json")
//   - targets: map of source directory names to target paths
//   - defaultTarget: fallback target for unmapped directories
//
// Returns:
//   - TargetResult with the computed path and metadata
//   - error if validation fails (reserved values, etc.)
func ResolveTarget(sourcePath string, targets map[string]string, defaultTarget string) (TargetResult, error) {
	// Validate targets for reserved values before resolution
	if err := validateTargets(targets); err != nil {
		return TargetResult{}, err
	}

	// Extract the first path segment
	segment, remainder := extractFirstSegment(sourcePath)

	// Check if segment matches a target key
	if targetPath, found := targets[segment]; found {
		return resolveMatchedTarget(segment, targetPath, remainder)
	}

	// No match found - use default target
	return resolveDefaultTarget(sourcePath, defaultTarget)
}

// ValidateTargetsForConflicts checks if the targets map contains configurations
// that could cause path conflicts during installation.
//
// A conflict occurs when:
//   - A direct mapping (target value "") could create files that overlap with
//     other target paths. For example, if "direct" maps to "" and "claude" maps
//     to ".claude", then files in "direct/.claude/" would conflict with files
//     in "claude/".
//
// Returns nil if no conflicts are detected, or ConflictingTargetError if a
// conflict is found.
func ValidateTargetsForConflicts(targets map[string]string) error {
	// First validate for reserved values
	if err := validateTargets(targets); err != nil {
		return err
	}

	// Check for direct mapping conflicts
	return checkDirectMappingConflicts(targets)
}

// extractFirstSegment splits a path into its first segment and the remainder.
// For example, "claude/settings.json" returns ("claude", "settings.json").
// For a single segment path like "file.md", returns ("file.md", "").
func extractFirstSegment(sourcePath string) (segment string, remainder string) {
	// Clean the path to normalize separators
	cleanPath := path.Clean(sourcePath)

	// Find the first separator
	idx := strings.Index(cleanPath, "/")
	if idx == -1 {
		// No separator - the entire path is the segment
		return cleanPath, ""
	}

	return cleanPath[:idx], cleanPath[idx+1:]
}

// resolveMatchedTarget handles resolution when a target mapping is found.
// It replaces the matched prefix with the target path value.
func resolveMatchedTarget(segment string, targetPath string, remainder string) (TargetResult, error) {
	result := TargetResult{
		Category:  segment,
		IsPartial: segment == "_partials",
	}

	// Build the target path
	if targetPath == "" {
		// Direct mapping to project root - just use the remainder
		result.TargetPath = remainder
	} else if remainder == "" {
		// Source path was just the segment (e.g., "claude" with no file)
		result.TargetPath = targetPath
	} else {
		// Normal case - join target path with remainder
		result.TargetPath = path.Join(targetPath, remainder)
	}

	return result, nil
}

// resolveDefaultTarget handles resolution when no target mapping is found.
// It prepends the default target to the full source path.
func resolveDefaultTarget(sourcePath string, defaultTarget string) (TargetResult, error) {
	result := TargetResult{
		Category:    "", // No category matched
		UsedDefault: true,
	}

	if defaultTarget == "" {
		// No default target - use path as-is
		result.TargetPath = sourcePath
	} else {
		// Prepend default target, preserving full source path structure
		result.TargetPath = path.Join(defaultTarget, sourcePath)
	}

	return result, nil
}

// validateTargets checks that the targets map does not contain reserved values.
// Reserved values are "." and ".." which have special meaning in filesystem paths.
func validateTargets(targets map[string]string) error {
	for name, value := range targets {
		// Check target name for reserved values
		if name == "." || name == ".." {
			return &InvalidTargetError{
				TargetName:    name,
				TargetValue:   value,
				ReservedValue: name,
				Reason:        "cannot be used as a target name because it has special meaning in filesystem paths",
			}
		}

		// Check target value for reserved values
		if value == "." || value == ".." {
			return &InvalidTargetError{
				TargetName:    name,
				TargetValue:   value,
				ReservedValue: value,
				Reason:        "cannot be used as a target path because it has special meaning in filesystem paths",
			}
		}
	}

	return nil
}

// checkDirectMappingConflicts detects when a direct mapping (empty string target)
// could create files that conflict with other target paths.
//
// A direct mapping installs content directly to the project root. If other targets
// install to subdirectories of the root, files from the direct mapping could
// potentially overwrite or conflict with them.
//
// For example:
//   - targets: {"direct": "", "claude": ".claude"}
//   - "direct/.claude/file.md" -> ".claude/file.md"
//   - "claude/file.md" -> ".claude/file.md"
//   - These two paths could conflict!
func checkDirectMappingConflicts(targets map[string]string) error {
	// Find direct mappings (empty string targets)
	var directMappings []string
	var otherTargets []struct {
		name  string
		value string
	}

	for name, value := range targets {
		if value == "" {
			directMappings = append(directMappings, name)
		} else {
			otherTargets = append(otherTargets, struct {
				name  string
				value string
			}{name, value})
		}
	}

	// If no direct mappings, no conflicts possible
	if len(directMappings) == 0 {
		return nil
	}

	// If we have direct mappings and other targets, there's potential for conflict
	// because files in direct/<target-path>/* would map to the same location
	// as files in <target-name>/*
	for _, directName := range directMappings {
		for _, other := range otherTargets {
			// A direct mapping can potentially place files at any path including
			// the paths used by other targets
			return &ConflictingTargetError{
				TargetName:      directName,
				TargetPath:      "",
				ConflictingName: other.name,
				ConflictingPath: other.value,
			}
		}
	}

	return nil
}
