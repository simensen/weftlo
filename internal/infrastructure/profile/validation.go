// Package profile provides infrastructure for loading and managing profiles.
package profile

import (
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/afero"

	domainprofile "github.com/simensen/weftlo/internal/domain/profile"
)

// ValidateProfileLayout performs a pre-flight check on a single profile
// directory, returning a MisplacedTemplatesError if any *.tmpl files exist at
// the profile root (one level deep, non-recursive).
//
// The renderer walks only the content/ subdirectory, so templates placed at
// the profile root are silently ignored. This validation catches that mistake
// before install/update operations report a misleading "success".
//
// contentRoot is the configured content directory name (defaults to "content").
// profileName is used for error messages only.
func ValidateProfileLayout(fs afero.Fs, profileDir, profileName, contentRoot string) error {
	if contentRoot == "" {
		contentRoot = domainprofile.DefaultContentRoot
	}

	entries, err := afero.ReadDir(fs, profileDir)
	if err != nil {
		// If the directory cannot be read here, the loader would have already
		// surfaced a more specific error. Skip silently.
		return nil
	}

	var misplaced []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasSuffix(name, ".tmpl") {
			misplaced = append(misplaced, name)
		}
	}

	if len(misplaced) == 0 {
		return nil
	}

	sort.Strings(misplaced)
	return &MisplacedTemplatesError{
		ProfileName: profileName,
		ContentRoot: contentRoot,
		Files:       misplaced,
	}
}

// ValidateChainLayout runs ValidateProfileLayout for every profile in a chain.
// It returns the first error encountered (root-to-leaf order), so the user
// sees the upstream-most violation first.
//
// profileNames must be in the same order as MergedProfile.ProfileNames()
// (i.e. merge order, root ancestor first).
func ValidateChainLayout(fs afero.Fs, profilesBasePath string, profileNames []string, contentRoot string) error {
	for _, name := range profileNames {
		parts := strings.SplitN(name, "/", 2)
		if len(parts) != 2 {
			continue
		}
		profileDir := filepath.Join(profilesBasePath, parts[0], parts[1])
		if err := ValidateProfileLayout(fs, profileDir, name, contentRoot); err != nil {
			return err
		}
	}
	return nil
}
