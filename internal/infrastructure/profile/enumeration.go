// Package profile provides infrastructure for loading and managing profiles.
package profile

import (
	"sort"
	"strings"

	domainprofile "github.com/simensen/weftlo/internal/domain/profile"
	domaintemplate "github.com/simensen/weftlo/internal/domain/template"
)

// EnumerateOutputTemplates walks the FileResolver virtual filesystem to enumerate
// all output templates that should be rendered to the target project.
//
// This function filters out:
//   - Partial files (files/directories starting with underscore _)
//
// Note: Dotfiles and profile.yaml are no longer filtered here because:
//   - Content enumeration now walks only the content root directory
//   - Dotfiles inside content root are valid installable content (e.g., .claude/CLAUDE.md)
//   - profile.yaml is outside the content root and not encountered during content enumeration
//
// Returns a slice of TemplateFile sorted alphabetically by SourcePath.
// All returned TemplateFile structs have IsPartial=false.
// Returns an empty slice (not nil) if no output templates are found.
//
// Integration with MergedProfile:
//
// The FileResolver interface is implemented by MergedProfile, which provides
// a virtual filesystem view combining parent and child profiles with
// child-overrides-parent semantics. To enumerate templates from a merged profile:
//
//	mergedProfile, err := loader.LoadMerged("vendor/name")
//	if err != nil {
//	    return nil, err
//	}
//	outputTemplates := EnumerateOutputTemplates(mergedProfile)
//
// The returned TemplateFile structs have:
//   - SourcePath: populated for downstream TargetPath calculation
//   - TargetPath: empty (computed in Rendering Pipeline phase)
//   - Checksum: empty (computed in Manifest Generation phase)
//   - IsPartial: always false (partials are filtered out)
func EnumerateOutputTemplates(resolver domaintemplate.FileResolver) []domainprofile.TemplateFile {
	// Get all source paths from the resolver
	sourcePaths := resolver.ListSourcePaths()

	// Filter and collect output templates
	var outputTemplates []domainprofile.TemplateFile

	for _, sourcePath := range sourcePaths {
		// Skip partial files (underscore convention)
		if isPartialFile(sourcePath) {
			continue
		}

		// Create TemplateFile entry for output template
		templateFile := domainprofile.TemplateFile{
			SourcePath: sourcePath,
			TargetPath: "", // Computed downstream in Rendering Pipeline phase
			IsPartial:  false,
			Checksum:   "", // Computed downstream in Manifest Generation phase
		}

		outputTemplates = append(outputTemplates, templateFile)
	}

	// Ensure we return an empty slice (not nil) if no templates found
	if outputTemplates == nil {
		outputTemplates = []domainprofile.TemplateFile{}
	}

	// Sort results alphabetically by SourcePath for deterministic ordering
	sort.Slice(outputTemplates, func(i, j int) bool {
		return outputTemplates[i].SourcePath < outputTemplates[j].SourcePath
	})

	return outputTemplates
}

// isPartialFile determines if a file is a partial based on the underscore convention.
// A file is considered a partial if:
//   - The filename starts with underscore (_)
//   - Any parent directory in the path starts with underscore (_)
//
// Partial files are included in other templates but not output directly.
// The sourcePath should use forward slashes (normalized cross-platform path).
func isPartialFile(sourcePath string) bool {
	// Split the path into components using forward slashes
	// (sourcePath is already normalized to use forward slashes)
	parts := strings.Split(sourcePath, "/")

	// Check each path component (directories and filename)
	for _, part := range parts {
		if strings.HasPrefix(part, "_") {
			return true
		}
	}

	return false
}
