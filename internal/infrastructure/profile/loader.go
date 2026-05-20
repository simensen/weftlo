// Package profile provides infrastructure for loading and managing profiles.
package profile

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/afero"
	"gopkg.in/yaml.v3"

	domainconfig "github.com/simensen/weftlo/internal/domain/config"
	domainignore "github.com/simensen/weftlo/internal/domain/ignore"
	domainprofile "github.com/simensen/weftlo/internal/domain/profile"
	infraignore "github.com/simensen/weftlo/internal/infrastructure/ignore"
)

// ProfileLoader defines the interface for loading profile configurations.
// This interface enables dependency injection for testing and allows
// different implementations for various profile sources.
type ProfileLoader interface {
	// Load loads a profile by its name (vendor/name format).
	// Returns a Profile with Name, InheritsFrom, and TemplateFiles populated.
	// Returns ProfileNotFoundError if the profile directory does not exist.
	// Returns ProfileConfigParseError for invalid YAML syntax in profile.yaml.
	Load(profileName string) (*domainprofile.Profile, error)

	// LoadWithChain loads a profile and its entire inheritance chain.
	// Starting from the requested profile, it recursively loads each parent
	// by following the InheritsFrom field until reaching a profile with no parent.
	//
	// Returns an ordered slice of Profile pointers from root ancestor (index 0)
	// to leaf child (last index). For example, if profile C inherits from B,
	// which inherits from A, calling LoadWithChain("vendor/C") returns [A, B, C].
	//
	// Returns CircularDependencyError if a cycle is detected in the inheritance chain.
	// Returns InheritanceDepthExceededError if the chain exceeds MaxInheritanceDepth.
	// Returns ProfileNotFoundError if any profile in the chain does not exist.
	// Returns ProfileConfigParseError for invalid YAML syntax in any profile.yaml.
	LoadWithChain(profileName string) ([]*domainprofile.Profile, error)

	// LoadMerged loads a profile and its entire inheritance chain, returning
	// a MergedProfile that provides a virtual filesystem view with child-overrides-parent
	// semantics.
	//
	// The returned MergedProfile implements the FileResolver interface, allowing
	// template include functions to resolve source paths to absolute filesystem paths.
	//
	// Returns CircularDependencyError if a cycle is detected in the inheritance chain.
	// Returns InheritanceDepthExceededError if the chain exceeds MaxInheritanceDepth.
	// Returns ProfileNotFoundError if any profile in the chain does not exist.
	// Returns ProfileConfigParseError for invalid YAML syntax in any profile.yaml.
	LoadMerged(profileName string) (*MergedProfile, error)

	// LoadMergedMultiple loads multiple profiles and merges them in order.
	// Each profile is loaded with its full inheritance chain, then profiles
	// are merged sequentially with later profiles overriding earlier ones.
	//
	// This is used for multi-profile support where the project specifies
	// a list of profiles to combine.
	//
	// Returns an error if profileNames is empty.
	// Returns ProfileNotFoundError if any profile does not exist.
	LoadMergedMultiple(profileNames []string) (*MergedProfile, error)
}

// HomeDirFunc is a function type for resolving the user's home directory.
// This allows for dependency injection in tests.
type HomeDirFunc func() (string, error)

// DefaultProfileLoader is the standard implementation of ProfileLoader.
// It loads profiles from the <configDir>/profiles/ directory.
// The configDir is typically resolved via XDG Base Directory conventions.
type DefaultProfileLoader struct {
	// fs is the filesystem used for reading profile files.
	fs afero.Fs
	// homeDir is the function used to resolve the user's home directory.
	// This is kept for backward compatibility with existing constructors.
	homeDir HomeDirFunc
	// configDir is the resolved configuration directory path.
	// If set, it overrides the homeDir-based path construction.
	configDir string
}

// NewProfileLoader creates a new DefaultProfileLoader with default settings.
// It uses the real OS filesystem and os.UserHomeDir for production use.
// For testing with an in-memory filesystem, use NewProfileLoaderWithFs instead.
func NewProfileLoader() *DefaultProfileLoader {
	return &DefaultProfileLoader{
		fs:      afero.NewOsFs(),
		homeDir: os.UserHomeDir,
	}
}

// NewProfileLoaderWithFs creates a new DefaultProfileLoader with the specified filesystem
// and home directory function.
// The filesystem parameter allows for dependency injection, enabling tests to use
// an in-memory filesystem (afero.NewMemMapFs()) instead of the real filesystem.
// The homeDir parameter allows tests to provide a custom home directory path.
// Note: This constructor uses homeDir to construct ~/.weftlo/profiles paths.
// For XDG support, use NewProfileLoaderWithConfigDir instead.
func NewProfileLoaderWithFs(fs afero.Fs, homeDir HomeDirFunc) *DefaultProfileLoader {
	return &DefaultProfileLoader{
		fs:      fs,
		homeDir: homeDir,
	}
}

// NewProfileLoaderWithConfigDir creates a new DefaultProfileLoader with the specified filesystem
// and pre-resolved configuration directory.
// The configDir should be the resolved config directory (e.g., ~/.config/weftlo or ~/.weftlo)
// as determined by XDG Base Directory conventions.
// This is the preferred constructor for production use with XDG support.
func NewProfileLoaderWithConfigDir(fs afero.Fs, configDir string) *DefaultProfileLoader {
	return &DefaultProfileLoader{
		fs:        fs,
		configDir: configDir,
	}
}

// Load loads a profile by its name (vendor/name format).
// It resolves the profile directory, parses profile.yaml if present,
// enumerates content files from the content root, and returns a populated Profile struct.
func (l *DefaultProfileLoader) Load(profileName string) (*domainprofile.Profile, error) {
	// Validate profile name format
	if err := domainconfig.ValidateProfileName(profileName); err != nil {
		return nil, err
	}

	// Resolve profile directory path
	profileDir, err := l.resolveProfileDir(profileName)
	if err != nil {
		return nil, err
	}

	// Check if profile directory exists
	exists, err := afero.DirExists(l.fs, profileDir)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, &ProfileNotFoundError{
			ProfileName: profileName,
			Path:        profileDir,
		}
	}

	// Parse profile.yaml (or infer config if missing)
	config, err := l.parseProfileConfig(profileDir, profileName)
	if err != nil {
		return nil, err
	}

	// Build content config with defaults applied early so we can use it for enumeration
	contentConfig := config.Content
	if contentConfig.Root == "" {
		contentConfig.Root = domainprofile.DefaultContentRoot
	}
	if contentConfig.DefaultTarget == "" {
		contentConfig.DefaultTarget = domainprofile.DefaultTargetPath
	}

	// Enumerate content files from the content root directory
	templateFiles, err := l.enumerateContentFiles(profileDir, contentConfig.Root)
	if err != nil {
		return nil, err
	}

	// Load ignore patterns from .weftlo.ignore file
	matcher, err := l.loadIgnorePatterns(profileDir, config.Ignore)
	if err != nil {
		return nil, err
	}

	// Create and return the Profile struct with all fields populated
	profile := &domainprofile.Profile{
		Name:          config.Name,
		InheritsFrom:  config.InheritsFrom,
		TemplateFiles: templateFiles,
		ContentConfig: contentConfig,
		Matcher:       matcher,
		Variables:     config.Variables,
	}

	return profile, nil
}

// loadIgnorePatterns loads ignore patterns from .weftlo.ignore file and inline patterns.
// It returns a combined Matcher with file-based patterns first, then inline patterns.
// If .weftlo.ignore does not exist, an empty matcher is used for file patterns.
func (l *DefaultProfileLoader) loadIgnorePatterns(profileDir string, inlinePatterns []string) (domainignore.Matcher, error) {
	ignoreLoader := infraignore.NewLoader(l.fs)

	// Load file-based patterns from .weftlo.ignore
	ignoreFilePath := filepath.Join(profileDir, ".weftlo.ignore")
	fileResult, err := ignoreLoader.LoadFromFile(ignoreFilePath)
	if err != nil {
		return nil, &ProfileConfigParseError{
			Path: ignoreFilePath,
			Err:  err,
		}
	}

	// Load inline patterns from profile.yaml
	inlineResult := ignoreLoader.LoadFromPatterns(inlinePatterns)

	// Merge: file patterns first, inline patterns second (inline can override)
	merged := infraignore.MergeResults(fileResult, inlineResult)

	return merged.Matcher, nil
}

// LoadWithChain loads a profile and its entire inheritance chain.
// It recursively loads parent profiles following the InheritsFrom field,
// detects circular dependencies, and enforces the maximum inheritance depth.
// Returns an ordered slice from root ancestor (index 0) to leaf child (last index).
func (l *DefaultProfileLoader) LoadWithChain(profileName string) ([]*domainprofile.Profile, error) {
	// Initialize visited map for circular dependency detection
	visited := make(map[string]bool)

	// Initialize chain slice to track profile names for error reporting
	var chainNames []string

	// Build the chain recursively (in leaf-to-root order)
	chain, err := l.loadChainRecursive(profileName, visited, chainNames, 0)
	if err != nil {
		return nil, err
	}

	// Reverse the chain to achieve root-to-leaf ordering
	reversed := make([]*domainprofile.Profile, len(chain))
	for i, profile := range chain {
		reversed[len(chain)-1-i] = profile
	}

	return reversed, nil
}

// LoadMerged loads a profile and its entire inheritance chain, returning
// a MergedProfile that provides a virtual filesystem view with child-overrides-parent
// semantics.
//
// This method is a convenience wrapper that combines LoadWithChain with
// NewMergedProfile construction. It handles resolving the profiles base path
// from the configuration directory.
func (l *DefaultProfileLoader) LoadMerged(profileName string) (*MergedProfile, error) {
	// Load the complete inheritance chain
	chain, err := l.LoadWithChain(profileName)
	if err != nil {
		return nil, err
	}

	// Resolve the profiles base path
	profilesBasePath, err := l.getProfilesBasePath()
	if err != nil {
		return nil, err
	}

	// Construct and return the MergedProfile
	return NewMergedProfile(chain, l.fs, profilesBasePath), nil
}

// LoadMergedMultiple loads multiple profiles and merges them in order.
// Each profile is loaded with its full inheritance chain, then profiles
// are merged sequentially with later profiles overriding earlier ones.
//
// This is used for multi-profile support where the project specifies
// a list of profiles to combine. The profiles are applied in order,
// so the last profile in the list has the highest priority.
func (l *DefaultProfileLoader) LoadMergedMultiple(profileNames []string) (*MergedProfile, error) {
	if len(profileNames) == 0 {
		return nil, &NoProfilesSpecifiedError{}
	}

	// Start with the first profile
	result, err := l.LoadMerged(profileNames[0])
	if err != nil {
		return nil, err
	}

	// Merge subsequent profiles
	for _, name := range profileNames[1:] {
		next, err := l.LoadMerged(name)
		if err != nil {
			return nil, err
		}
		result = result.MergeWith(next)
	}

	return result, nil
}

// ProfilesBasePath exposes the resolved profiles directory path for callers
// that need to perform filesystem operations alongside the loader (e.g.,
// pre-flight validation). It mirrors the private getProfilesBasePath.
func (l *DefaultProfileLoader) ProfilesBasePath() (string, error) {
	return l.getProfilesBasePath()
}

// getProfilesBasePath returns the profiles directory path.
// If configDir is set, uses <configDir>/profiles.
// Otherwise, falls back to <homeDir>/.weftlo/profiles for backward compatibility.
func (l *DefaultProfileLoader) getProfilesBasePath() (string, error) {
	if l.configDir != "" {
		return filepath.Join(l.configDir, "profiles"), nil
	}

	// Fallback for backward compatibility
	home, err := l.homeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".weftlo", "profiles"), nil
}

// loadChainRecursive is the internal recursive helper for LoadWithChain.
// It builds the chain in leaf-to-root order (child appended first).
// Parameters:
//   - profileName: the current profile to load
//   - visited: map tracking visited profile names for cycle detection
//   - chainNames: slice of profile names visited so far (for error reporting)
//   - depth: current recursion depth (for limit enforcement)
//
// Returns the chain in leaf-to-root order.
func (l *DefaultProfileLoader) loadChainRecursive(
	profileName string,
	visited map[string]bool,
	chainNames []string,
	depth int,
) ([]*domainprofile.Profile, error) {
	// Check depth limit before loading
	if depth >= MaxInheritanceDepth {
		return nil, &InheritanceDepthExceededError{
			MaxDepth: MaxInheritanceDepth,
			Chain:    chainNames,
		}
	}

	// Check for circular dependency
	if visited[profileName] {
		return nil, &CircularDependencyError{
			ProfileName: profileName,
			Chain:       append(chainNames, profileName),
		}
	}

	// Mark this profile as visited
	visited[profileName] = true

	// Add to chain names for error reporting
	chainNames = append(chainNames, profileName)

	// Load the current profile using the existing Load method
	profile, err := l.Load(profileName)
	if err != nil {
		return nil, err
	}

	// Base case: profile has no parent (empty InheritsFrom)
	if profile.InheritsFrom == "" {
		return []*domainprofile.Profile{profile}, nil
	}

	// Recursive case: load the parent chain first
	parentChain, err := l.loadChainRecursive(profile.InheritsFrom, visited, chainNames, depth+1)
	if err != nil {
		return nil, err
	}

	// Append current profile to the chain (child after parent in leaf-to-root order)
	// Note: We're building leaf-to-root, so we append the child first
	return append([]*domainprofile.Profile{profile}, parentChain...), nil
}

// resolveProfileDir constructs the full path to the profile directory.
// The path format is: <configDir>/profiles/{vendor}/{name}/
func (l *DefaultProfileLoader) resolveProfileDir(profileName string) (string, error) {
	profilesBasePath, err := l.getProfilesBasePath()
	if err != nil {
		return "", err
	}

	// Split profile name into vendor and name segments
	parts := strings.Split(profileName, "/")
	vendor := parts[0]
	name := parts[1]

	return filepath.Join(profilesBasePath, vendor, name), nil
}

// parseProfileConfig parses the profile.yaml file if it exists,
// or creates a ProfileConfig with the name inferred from the directory path.
func (l *DefaultProfileLoader) parseProfileConfig(profileDir, profileName string) (*domainprofile.ProfileConfig, error) {
	profileYAMLPath := filepath.Join(profileDir, "profile.yaml")

	// Check if profile.yaml exists
	exists, err := afero.Exists(l.fs, profileYAMLPath)
	if err != nil {
		return nil, err
	}

	// If profile.yaml doesn't exist, infer config from directory path
	if !exists {
		return &domainprofile.ProfileConfig{
			Name: profileName,
		}, nil
	}

	// Read and parse profile.yaml
	data, err := afero.ReadFile(l.fs, profileYAMLPath)
	if err != nil {
		return nil, &ProfileConfigParseError{
			Path: profileYAMLPath,
			Err:  err,
		}
	}

	var config domainprofile.ProfileConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, &ProfileConfigParseError{
			Path: profileYAMLPath,
			Err:  err,
		}
	}

	// If name field is empty, infer from directory path
	if config.Name == "" {
		config.Name = profileName
	}

	return &config, nil
}

// enumerateContentFiles recursively walks the content root directory to enumerate
// all content files. It creates a TemplateFile entry for each file.
//
// The contentRoot parameter specifies the subdirectory within the profile that
// contains installable content (default: "content").
//
// If the content root directory does not exist, an empty file list is returned
// (not an error condition). This allows profiles to exist without a content
// directory when they only provide configuration or inherit from other profiles.
//
// Paths returned are relative to the content root (not the profile directory).
// Dotfiles inside the content root ARE enumerated (e.g., .claude/CLAUDE.md).
// Files starting with underscore are marked as partials (IsPartial=true).
func (l *DefaultProfileLoader) enumerateContentFiles(profileDir, contentRoot string) ([]domainprofile.TemplateFile, error) {
	// Construct the full path to the content root directory
	contentRootPath := filepath.Join(profileDir, contentRoot)

	// Check if content root directory exists
	exists, err := afero.DirExists(l.fs, contentRootPath)
	if err != nil {
		return nil, &FileEnumerationError{
			Path: contentRootPath,
			Err:  err,
		}
	}

	// If content root does not exist, return empty list (not an error)
	if !exists {
		return []domainprofile.TemplateFile{}, nil
	}

	var templateFiles []domainprofile.TemplateFile

	// Use afero.Walk() to recursively traverse the content root directory
	err = afero.Walk(l.fs, contentRootPath, func(path string, info os.FileInfo, walkErr error) error {
		// Handle walk errors
		if walkErr != nil {
			return walkErr
		}

		// Skip directories (we only enumerate files)
		if info.IsDir() {
			return nil
		}

		// Compute relative path from content root directory
		relPath, err := filepath.Rel(contentRootPath, path)
		if err != nil {
			return err
		}

		// Convert path separators to forward slashes for cross-platform consistency
		sourcePath := filepath.ToSlash(relPath)

		// Determine if this file is a partial (underscore convention)
		// Uses the shared isPartialFile function from enumeration.go
		partial := isPartialFile(sourcePath)

		// Create TemplateFile entry
		// Note: TargetPath is computed in Rendering Pipeline phase (left as empty string)
		// Note: Checksum is computed in Manifest Generation phase (left as empty string)
		templateFile := domainprofile.TemplateFile{
			SourcePath: sourcePath,
			TargetPath: "", // Computed downstream in Rendering Pipeline phase
			IsPartial:  partial,
			Checksum:   "", // Computed downstream in Manifest Generation phase
		}

		templateFiles = append(templateFiles, templateFile)
		return nil
	})

	if err != nil {
		return nil, &FileEnumerationError{
			Path: contentRootPath,
			Err:  err,
		}
	}

	return templateFiles, nil
}
