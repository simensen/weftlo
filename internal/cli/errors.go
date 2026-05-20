package cli

import (
	"fmt"
	"strings"
)

// HomeDirError is returned when the home directory cannot be resolved.
type HomeDirError struct {
	Err error
}

func (e *HomeDirError) Error() string {
	return fmt.Sprintf("failed to resolve home directory: %v", e.Err)
}

func (e *HomeDirError) Unwrap() error {
	return e.Err
}

// PermissionError is returned when a file or directory operation fails due to permissions.
type PermissionError struct {
	Path string
	Err  error
}

func (e *PermissionError) Error() string {
	return fmt.Sprintf("permission denied: %s: %v", e.Path, e.Err)
}

func (e *PermissionError) Unwrap() error {
	return e.Err
}

// ReinitializeAbortedError is returned when the user declines to reinitialize.
type ReinitializeAbortedError struct{}

func (e *ReinitializeAbortedError) Error() string {
	return "reinitialization aborted by user"
}

// GlobalConfigNotFoundError is returned when the global configuration directory
// does not exist and needs to be initialized.
type GlobalConfigNotFoundError struct{}

func (e *GlobalConfigNotFoundError) Error() string {
	return "Global configuration not found. Run `weftlo init` to initialize."
}

// ProfileResolutionError is returned when a profile cannot be resolved from
// any source in the resolution chain (flag, project config, global config).
type ProfileResolutionError struct {
	FlagChecked          bool
	ProjectConfigChecked bool
	GlobalConfigChecked  bool
	Err                  error
}

func (e *ProfileResolutionError) Error() string {
	var sources []string

	if e.FlagChecked {
		sources = append(sources, "--profile flag")
	}
	if e.ProjectConfigChecked {
		sources = append(sources, "project config (.weftlo.yaml)")
	}
	if e.GlobalConfigChecked {
		sources = append(sources, "global config (default_profile)")
	}

	if len(sources) == 0 {
		return "could not resolve profile: no sources checked"
	}

	return fmt.Sprintf("could not resolve profile. Checked: %s", strings.Join(sources, ", "))
}

func (e *ProfileResolutionError) Unwrap() error {
	return e.Err
}

// FileConflictError is returned when installation would overwrite existing files
// and the --force flag was not specified.
type FileConflictError struct {
	ConflictingFiles []string
}

func (e *FileConflictError) Error() string {
	if len(e.ConflictingFiles) == 0 {
		return "file conflict detected. Use --force to overwrite existing files."
	}

	var msg strings.Builder
	msg.WriteString("file conflict: the following files already exist:\n")

	for _, file := range e.ConflictingFiles {
		fmt.Fprintf(&msg, "  - %s\n", file)
	}

	msg.WriteString("Use --force to overwrite existing files.")

	return msg.String()
}

// InstallationNotFoundError is returned when the update command is run
// but no installation exists (missing .weftlo.yaml or manifest).
type InstallationNotFoundError struct {
	// MissingFile indicates which file is missing (.weftlo.yaml or .weftlo.manifest.json)
	MissingFile string
	// Err is the underlying error, if any
	Err error
}

func (e *InstallationNotFoundError) Error() string {
	return fmt.Sprintf("installation not found: %s is missing. Run `weftlo install` first.", e.MissingFile)
}

func (e *InstallationNotFoundError) Unwrap() error {
	return e.Err
}

// ProfileNotFoundError is returned when a profile referenced in the project config
// no longer exists in the config directory.
type ProfileNotFoundError struct {
	// ProfileName is the name of the profile that was not found
	ProfileName string
	// Err is the underlying error, if any
	Err error
}

func (e *ProfileNotFoundError) Error() string {
	return fmt.Sprintf("profile not found: '%s' does not exist in the config directory", e.ProfileName)
}

func (e *ProfileNotFoundError) Unwrap() error {
	return e.Err
}

// ProfileExistsError is returned when attempting to create a profile that already exists
// and the --force flag was not specified.
type ProfileExistsError struct {
	// ProfileName is the name of the profile that already exists
	ProfileName string
	// Err is the underlying error, if any
	Err error
}

func (e *ProfileExistsError) Error() string {
	return fmt.Sprintf("profile '%s' already exists. Use --force to overwrite.", e.ProfileName)
}

func (e *ProfileExistsError) Unwrap() error {
	return e.Err
}

// AlreadyInstalledError is returned when attempting to run install
// but .weftlo.yaml already exists.
type AlreadyInstalledError struct{}

func (e *AlreadyInstalledError) Error() string {
	return "already initialized: .weftlo.yaml exists. Use 'weftlo update --add-profile <profile>' to add profiles."
}

// ProfileNotInListError is returned when attempting to remove a profile
// that is not in the current profile list.
type ProfileNotInListError struct {
	ProfileName string
}

func (e *ProfileNotInListError) Error() string {
	return fmt.Sprintf("profile '%s' is not in the current profile list", e.ProfileName)
}
