// Package profile provides infrastructure for loading and managing profiles.
package profile

import (
	"path/filepath"
	"strings"

	"github.com/spf13/afero"

	"github.com/simensen/weftlo/internal/app/routing"
	domainignore "github.com/simensen/weftlo/internal/domain/ignore"
	domainprofile "github.com/simensen/weftlo/internal/domain/profile"
)

// MergedProfile represents a virtual filesystem view that combines an inheritance
// chain of profiles with child-overrides-parent semantics. It implements the
// FileResolver interface for use with template include functions.
//
// When source paths exist in both parent and child profiles, the child version
// takes precedence. The merged view includes all template files from the entire
// inheritance chain, with overlapping paths resolved to the child-most definition.
type MergedProfile struct {
	// profileNames contains all profile names that were merged (in merge order).
	// The last entry is the most specific (highest priority) profile.
	profileNames []string

	// sourceToAbsolutePath maps source paths to their absolute filesystem paths.
	// This enables O(1) path resolution for the FileResolver interface.
	sourceToAbsolutePath map[string]string

	// templateFiles contains the merged TemplateFile entries preserving IsPartial flags.
	// Each entry represents a unique source path with its winning (child-most) attributes.
	templateFiles []domainprofile.TemplateFile

	// fs is the filesystem reference for any path operations.
	fs afero.Fs

	// ContentConfig carries the merged content configuration from the profile chain.
	// It is deep merged with the following semantics:
	//   - Root: child overrides parent only if child explicitly sets it (non-empty)
	//   - Targets: shallow merge - child keys override parent keys, parent keys preserved
	//   - DefaultTarget: child overrides parent only if child explicitly sets it (non-empty)
	ContentConfig domainprofile.ContentConfig

	// Matcher carries the merged ignore patterns from the profile chain.
	// Patterns are merged with parent patterns first, then child patterns,
	// allowing child profiles to override parent patterns via negation.
	// This field represents profile-only ignore patterns; project-level ignores
	// are added separately in the installation pipeline (Spec 5).
	Matcher domainignore.Matcher

	// Variables holds the merged variables from the profile inheritance chain.
	// Variables are merged with deep merge semantics, where child profile values
	// override parent profile values. Nested maps are recursively merged.
	// This field is populated during NewMergedProfile from each profile's Variables.
	Variables map[string]interface{}

	// variableConflicts holds the accumulated variable conflicts from merging
	// the profile inheritance chain. Each conflict represents a scalar value
	// at a specific path that was overridden by a child profile.
	variableConflicts []domainprofile.VariableConflict
}

// VariableConflicts returns the accumulated variable conflicts from merging
// the profile inheritance chain. Each conflict represents a scalar value
// at a specific path that was overridden by a child profile.
func (m *MergedProfile) VariableConflicts() []domainprofile.VariableConflict {
	// Return a copy to prevent external modifications
	if m.variableConflicts == nil {
		return nil
	}
	result := make([]domainprofile.VariableConflict, len(m.variableConflicts))
	copy(result, m.variableConflicts)
	return result
}

// NewMergedProfile creates a new MergedProfile by merging an inheritance chain
// of profiles with child-overrides-parent semantics.
//
// The chain parameter should be ordered root-to-leaf (as returned by LoadWithChain),
// where index 0 is the root ancestor and the last index is the leaf child.
// Iterating forward through the chain ensures that later profiles (children)
// override earlier ones (parents).
//
// The profilesBasePath is the base directory containing all profiles, typically
// {home}/.weftlo/profiles. Absolute paths are computed by combining:
// profilesBasePath + vendor + name + contentRoot + sourcePath.
//
// This function does not return errors since it receives already-loaded profiles.
func NewMergedProfile(chain []*domainprofile.Profile, fs afero.Fs, profilesBasePath string) *MergedProfile {
	sourceToAbsPath := make(map[string]string)
	templateFileMap := make(map[string]domainprofile.TemplateFile)

	var profileNames []string

	// Initialize merged ContentConfig and Matcher
	var mergedContentConfig domainprofile.ContentConfig
	mergedMatcher := domainignore.EmptyMatcher()

	// Initialize merged Variables and conflicts
	mergedVariables := make(map[string]interface{})
	var allConflicts []domainprofile.VariableConflict
	var previousProfileName string

	// Iterate forward through the chain (root-to-leaf) so children override parents
	for _, profile := range chain {
		// Track the leaf profile name (last in chain)
		profileNames = append(profileNames, profile.Name)

		// Merge ContentConfig using deep merge semantics
		mergedContentConfig = routing.MergeContentConfigs(mergedContentConfig, profile.ContentConfig)

		// Merge Matcher: parent patterns first, then child patterns
		if profile.Matcher != nil {
			mergedMatcher = mergedMatcher.MergeWith(profile.Matcher)
		}

		// Merge Variables using DeepMergeVariables from domain layer
		if profile.Variables != nil {
			if previousProfileName == "" {
				// First profile in chain - just copy variables
				mergedVariables, _ = domainprofile.DeepMergeVariables(nil, profile.Variables, "", profile.Name)
			} else {
				// Subsequent profiles - deep merge with conflict tracking
				var conflicts []domainprofile.VariableConflict
				mergedVariables, conflicts = domainprofile.DeepMergeVariables(
					mergedVariables,
					profile.Variables,
					previousProfileName,
					profile.Name,
				)
				allConflicts = append(allConflicts, conflicts...)
			}
		}

		// Update previous profile name for next iteration
		previousProfileName = profile.Name

		// Split profile name into vendor and name for path construction
		vendor, name := splitProfileNameParts(profile.Name)

		// Get the content root from the profile's ContentConfig
		// Default to "content" if not specified
		contentRoot := profile.ContentConfig.Root
		if contentRoot == "" {
			contentRoot = domainprofile.DefaultContentRoot
		}

		// Process each template file in this profile
		for _, tf := range profile.TemplateFiles {
			// Compute absolute path: profilesBasePath/vendor/name/contentRoot/sourcePath
			absPath := filepath.Join(profilesBasePath, vendor, name, contentRoot, tf.SourcePath)

			// Store/override in maps (later profiles override earlier ones)
			sourceToAbsPath[tf.SourcePath] = absPath
			templateFileMap[tf.SourcePath] = tf
		}
	}

	// Convert template file map to slice
	templateFiles := make([]domainprofile.TemplateFile, 0, len(templateFileMap))
	for _, tf := range templateFileMap {
		templateFiles = append(templateFiles, tf)
	}

	return &MergedProfile{
		profileNames:         profileNames,
		sourceToAbsolutePath: sourceToAbsPath,
		templateFiles:        templateFiles,
		fs:                   fs,
		ContentConfig:        mergedContentConfig,
		Matcher:              mergedMatcher,
		Variables:            mergedVariables,
		variableConflicts:    allConflicts,
	}
}

// splitProfileNameParts splits a profile name (vendor/name format) into its
// vendor and name components. Returns empty strings if the format is invalid.
func splitProfileNameParts(profileName string) (vendor, name string) {
	parts := strings.SplitN(profileName, "/", 2)
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return "", ""
}

// LeafProfileName returns the name of the leaf (most specific) profile in the chain.
// This is the profile that was originally requested for loading.
// For multi-profile merges, this returns the last (highest priority) profile.
func (m *MergedProfile) LeafProfileName() string {
	if len(m.profileNames) == 0 {
		return ""
	}
	return m.profileNames[len(m.profileNames)-1]
}

// ProfileNames returns all profile names that were merged (in merge order).
// The last entry is the most specific (highest priority) profile.
func (m *MergedProfile) ProfileNames() []string {
	result := make([]string, len(m.profileNames))
	copy(result, m.profileNames)
	return result
}

// MergeWith combines this MergedProfile with another, returning a new MergedProfile.
// The other profile's templates override this profile's templates (other wins).
// This is used for multi-profile support where profiles are merged in sequence.
//
// ContentConfig is merged with deep merge semantics:
//   - Root: other overrides this only if other's Root is non-empty
//   - Targets: shallow merge - other keys override this keys, this keys preserved
//   - DefaultTarget: other overrides this only if other's DefaultTarget is non-empty
//
// Matcher is merged with this profile's patterns first, then other's patterns,
// allowing the other profile to override this profile's patterns via negation.
//
// Variables are merged with deep merge semantics using DeepMergeVariables.
// Conflicts from both profiles are combined and new conflicts from the merge are added.
func (m *MergedProfile) MergeWith(other *MergedProfile) *MergedProfile {
	// Start with this profile's data
	sourceToAbsPath := make(map[string]string)
	templateFileMap := make(map[string]domainprofile.TemplateFile)

	// Copy from this profile
	for path, absPath := range m.sourceToAbsolutePath {
		sourceToAbsPath[path] = absPath
	}
	for _, tf := range m.templateFiles {
		templateFileMap[tf.SourcePath] = tf
	}

	// Override with other profile's data
	for path, absPath := range other.sourceToAbsolutePath {
		sourceToAbsPath[path] = absPath
	}
	for _, tf := range other.templateFiles {
		templateFileMap[tf.SourcePath] = tf
	}

	// Convert template file map to slice
	templateFiles := make([]domainprofile.TemplateFile, 0, len(templateFileMap))
	for _, tf := range templateFileMap {
		templateFiles = append(templateFiles, tf)
	}

	// Combine profile names (this profile's names + other profile's names)
	// Use unique names to avoid duplicates from inheritance chains
	seen := make(map[string]bool)
	var combinedNames []string
	for _, name := range m.profileNames {
		if !seen[name] {
			seen[name] = true
			combinedNames = append(combinedNames, name)
		}
	}
	for _, name := range other.profileNames {
		if !seen[name] {
			seen[name] = true
			combinedNames = append(combinedNames, name)
		}
	}

	// Merge ContentConfig using deep merge semantics
	mergedContentConfig := routing.MergeContentConfigs(m.ContentConfig, other.ContentConfig)

	// Merge Matcher: this profile's patterns first, then other's patterns
	var mergedMatcher domainignore.Matcher
	if m.Matcher != nil && other.Matcher != nil {
		mergedMatcher = m.Matcher.MergeWith(other.Matcher)
	} else if m.Matcher != nil {
		mergedMatcher = m.Matcher
	} else if other.Matcher != nil {
		mergedMatcher = other.Matcher
	} else {
		mergedMatcher = domainignore.EmptyMatcher()
	}

	// Merge Variables using DeepMergeVariables from domain layer
	// Use "merged" as the base profile name since this is a combination of multiple profiles
	mergedVariables, newConflicts := domainprofile.DeepMergeVariables(
		m.Variables,
		other.Variables,
		"merged",
		other.LeafProfileName(),
	)

	// Combine conflict lists from both profiles plus new conflicts from this merge
	var combinedConflicts []domainprofile.VariableConflict
	combinedConflicts = append(combinedConflicts, m.variableConflicts...)
	combinedConflicts = append(combinedConflicts, other.variableConflicts...)
	combinedConflicts = append(combinedConflicts, newConflicts...)

	return &MergedProfile{
		profileNames:         combinedNames,
		sourceToAbsolutePath: sourceToAbsPath,
		templateFiles:        templateFiles,
		fs:                   m.fs, // Use this profile's filesystem
		ContentConfig:        mergedContentConfig,
		Matcher:              mergedMatcher,
		Variables:            mergedVariables,
		variableConflicts:    combinedConflicts,
	}
}

// Resolve implements the FileResolver interface.
// It takes a source path and returns the absolute filesystem path where the file
// can be found in the merged profile hierarchy.
//
// Returns:
//   - absolutePath: The resolved absolute path to the file
//   - found: true if the source path was found, false otherwise
//
// When found is false, absolutePath will be an empty string.
// This method provides O(1) lookup using the internal sourceToAbsolutePath map.
func (m *MergedProfile) Resolve(sourcePath string) (absolutePath string, found bool) {
	absolutePath, found = m.sourceToAbsolutePath[sourcePath]
	if !found {
		return "", false
	}
	return absolutePath, true
}

// ListSourcePaths implements the FileResolver interface.
// It returns all source paths available in this merged profile.
//
// Returns:
//   - A slice containing all source paths (keys) in the merged profile
//   - The order of paths is not guaranteed (map iteration order)
//   - Returns an empty slice (not nil) if no paths are registered
func (m *MergedProfile) ListSourcePaths() []string {
	paths := make([]string, 0, len(m.sourceToAbsolutePath))
	for sourcePath := range m.sourceToAbsolutePath {
		paths = append(paths, sourcePath)
	}
	return paths
}

// TemplateFiles returns the merged template file entries with their winning
// (child-most) attributes preserved, including the IsPartial flag.
// This method is useful for downstream template enumeration.
func (m *MergedProfile) TemplateFiles() []domainprofile.TemplateFile {
	// Return a copy to prevent external modifications
	result := make([]domainprofile.TemplateFile, len(m.templateFiles))
	copy(result, m.templateFiles)
	return result
}

// Fs returns the filesystem reference associated with this merged profile.
// This can be used for file operations on the resolved absolute paths.
func (m *MergedProfile) Fs() afero.Fs {
	return m.fs
}

// ToProfile converts the merged profile to a domain Profile struct.
// This is useful for constructing TemplateContext during rendering.
// The returned Profile contains the leaf profile name, merged template files,
// merged ContentConfig, merged Matcher, and merged Variables.
func (m *MergedProfile) ToProfile() *domainprofile.Profile {
	return &domainprofile.Profile{
		Name:          m.LeafProfileName(),
		TemplateFiles: m.TemplateFiles(),
		ContentConfig: m.ContentConfig,
		Matcher:       m.Matcher,
		Variables:     m.Variables,
	}
}
