// Package profile provides infrastructure for loading and managing profiles.
package profile

import (
	"fmt"
	"strings"
)

// MaxInheritanceDepth defines the maximum allowed depth for profile inheritance chains.
// This limit prevents infinite recursion and excessive memory usage in deep inheritance hierarchies.
const MaxInheritanceDepth = 15

// ProfileNotFoundError is returned when a profile directory does not exist.
// This error provides the profile name and the expected path for troubleshooting.
type ProfileNotFoundError struct {
	// ProfileName is the requested profile identifier (e.g., "acme/backend").
	ProfileName string
	// Path is the expected path where the profile directory should exist.
	Path string
}

func (e *ProfileNotFoundError) Error() string {
	return fmt.Sprintf("profile '%s' not found at %s", e.ProfileName, e.Path)
}

// ProfileConfigParseError is returned when a profile.yaml file contains invalid YAML syntax.
// This error wraps the underlying parse error for error chaining.
type ProfileConfigParseError struct {
	// Path is the path to the profile.yaml file that failed to parse.
	Path string
	// Err is the underlying parse error from the YAML parser.
	Err error
}

func (e *ProfileConfigParseError) Error() string {
	return fmt.Sprintf("failed to parse profile configuration %s: %v", e.Path, e.Err)
}

func (e *ProfileConfigParseError) Unwrap() error {
	return e.Err
}

// FileEnumerationError is returned when the filesystem walk fails during template file enumeration.
// This error wraps the underlying filesystem error for error chaining.
type FileEnumerationError struct {
	// Path is the profile directory path where enumeration failed.
	Path string
	// Err is the underlying filesystem error.
	Err error
}

func (e *FileEnumerationError) Error() string {
	return fmt.Sprintf("failed to enumerate files in profile directory %s: %v", e.Path, e.Err)
}

func (e *FileEnumerationError) Unwrap() error {
	return e.Err
}

// CircularDependencyError is returned when a circular dependency is detected in the inheritance chain.
// This occurs when profile A inherits from profile B, which directly or indirectly inherits from A.
type CircularDependencyError struct {
	// ProfileName is the profile where the cycle was detected.
	ProfileName string
	// Chain contains the profile names visited in order, ending with the profile that creates the cycle.
	Chain []string
}

func (e *CircularDependencyError) Error() string {
	return fmt.Sprintf("circular dependency detected: profile '%s' creates a cycle in chain [%s]", e.ProfileName, strings.Join(e.Chain, ", "))
}

// InheritanceDepthExceededError is returned when the inheritance chain exceeds the maximum allowed depth.
// This prevents infinite recursion and excessive resource consumption in deeply nested inheritance hierarchies.
type InheritanceDepthExceededError struct {
	// MaxDepth is the maximum allowed inheritance depth that was exceeded.
	MaxDepth int
	// Chain contains the profile names traversed up to the point where the limit was exceeded.
	Chain []string
}

func (e *InheritanceDepthExceededError) Error() string {
	return fmt.Sprintf("inheritance depth exceeded: maximum depth of %d reached with chain [%s]", e.MaxDepth, strings.Join(e.Chain, ", "))
}

// NoProfilesSpecifiedError is returned when LoadMergedMultiple is called with an empty profile list.
type NoProfilesSpecifiedError struct{}

func (e *NoProfilesSpecifiedError) Error() string {
	return "no profiles specified"
}

// MisplacedTemplatesError is returned when *.tmpl files are found at the
// profile root rather than inside the content/ directory. The renderer only
// reads files under content/, so misplaced templates would otherwise be
// silently ignored — this error catches that mistake early.
type MisplacedTemplatesError struct {
	// ProfileName is the profile where the misplaced files were found.
	ProfileName string
	// ContentRoot is the expected content directory (typically "content").
	ContentRoot string
	// Files contains the misplaced filenames, relative to the profile root.
	Files []string
}

func (e *MisplacedTemplatesError) Error() string {
	noun := "file"
	if len(e.Files) != 1 {
		noun = "files"
	}
	var b strings.Builder
	fmt.Fprintf(&b, "profile %q has %d template %s outside %s/:\n",
		e.ProfileName, len(e.Files), noun, e.ContentRoot)
	for _, f := range e.Files {
		fmt.Fprintf(&b, "  - %s\n", f)
	}
	fmt.Fprintf(&b, "Templates must live in the profile's %s/ directory. Move the %s and retry.",
		e.ContentRoot, noun)
	return b.String()
}
