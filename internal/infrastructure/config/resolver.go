// Package config provides infrastructure for loading and resolving configuration.
package config

import (
	"os"
	"path/filepath"
)

// Filesystem abstracts filesystem operations for testability.
// This interface allows dependency injection for testing without touching the real filesystem.
type Filesystem interface {
	// UserHomeDir returns the current user's home directory.
	UserHomeDir() (string, error)

	// GetEnv returns the value of the environment variable named by the key.
	GetEnv(key string) string

	// DirExists checks if a directory exists at the given path.
	// Returns true if the directory exists, false otherwise.
	// Returns an error if the path cannot be accessed (e.g., permission denied).
	DirExists(path string) (bool, error)
}

// osFilesystem implements Filesystem using the real operating system.
type osFilesystem struct{}

func (fs *osFilesystem) UserHomeDir() (string, error) {
	return os.UserHomeDir()
}

func (fs *osFilesystem) GetEnv(key string) string {
	return os.Getenv(key)
}

func (fs *osFilesystem) DirExists(path string) (bool, error) {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		// Permission denied or other error
		return false, err
	}
	return info.IsDir(), nil
}

// Resolver resolves the configuration directory path following XDG Base Directory specification.
type Resolver struct {
	fs Filesystem
}

// ResolverOption is a functional option for configuring a Resolver.
type ResolverOption func(*Resolver)

// WithFilesystem sets a custom filesystem implementation for testing.
func WithFilesystem(fs Filesystem) ResolverOption {
	return func(r *Resolver) {
		r.fs = fs
	}
}

// NewResolver creates a new Resolver with the given options.
// If no filesystem is provided, it uses the real operating system filesystem.
func NewResolver(opts ...ResolverOption) *Resolver {
	r := &Resolver{
		fs: &osFilesystem{},
	}
	for _, opt := range opts {
		opt(r)
	}
	return r
}

// ResolveConfigDir determines the appropriate configuration directory path.
// Resolution order:
//  1. $XDG_CONFIG_HOME/weftlo/ if XDG_CONFIG_HOME environment variable is set
//  2. ~/.config/weftlo/ if that directory exists
//  3. ~/.weftlo/ as fallback
//
// Returns the resolved absolute path and any error that occurred.
func (r *Resolver) ResolveConfigDir() (string, error) {
	// Priority 1: Check XDG_CONFIG_HOME environment variable
	xdgConfigHome := r.fs.GetEnv("XDG_CONFIG_HOME")
	if xdgConfigHome != "" {
		return filepath.Join(xdgConfigHome, "weftlo"), nil
	}

	// Get user home directory for remaining checks
	homeDir, err := r.fs.UserHomeDir()
	if err != nil {
		return "", err
	}

	// Priority 2: Check if ~/.config/weftlo/ exists
	dotConfigPath := filepath.Join(homeDir, ".config", "weftlo")
	exists, err := r.fs.DirExists(dotConfigPath)
	if err != nil {
		// Permission error or other issue - gracefully fall back
		// Don't fail, just move to fallback
	} else if exists {
		return dotConfigPath, nil
	}

	// Priority 3: Fallback to ~/.weftlo/
	return filepath.Join(homeDir, ".weftlo"), nil
}

// ResolveConfigDirForCreation determines the appropriate configuration directory path
// for creating new configuration (e.g., during init).
//
// The difference from ResolveConfigDir is:
//   - ResolveConfigDir checks if ~/.config/weftlo/ EXISTS
//   - ResolveConfigDirForCreation checks if ~/.config/ EXISTS (parent directory)
//
// This allows init to create ~/.config/weftlo/ when the user has a ~/.config/
// directory, following XDG conventions.
//
// Resolution order:
//  1. $XDG_CONFIG_HOME/weftlo/ if XDG_CONFIG_HOME environment variable is set
//  2. ~/.config/weftlo/ if ~/.config/ directory exists
//  3. ~/.weftlo/ as fallback
//
// Returns the resolved absolute path and any error that occurred.
func (r *Resolver) ResolveConfigDirForCreation() (string, error) {
	// Priority 1: Check XDG_CONFIG_HOME environment variable
	xdgConfigHome := r.fs.GetEnv("XDG_CONFIG_HOME")
	if xdgConfigHome != "" {
		return filepath.Join(xdgConfigHome, "weftlo"), nil
	}

	// Get user home directory for remaining checks
	homeDir, err := r.fs.UserHomeDir()
	if err != nil {
		return "", err
	}

	// Priority 2: Check if ~/.config/ directory exists (NOTE: checking PARENT, not weftlo subdir)
	dotConfigPath := filepath.Join(homeDir, ".config")
	exists, err := r.fs.DirExists(dotConfigPath)
	if err != nil {
		// Permission error or other issue - gracefully fall back
		// Don't fail, just move to fallback
	} else if exists {
		return filepath.Join(dotConfigPath, "weftlo"), nil
	}

	// Priority 3: Fallback to ~/.weftlo/
	return filepath.Join(homeDir, ".weftlo"), nil
}

// DirExists checks if a directory exists at the given path.
// This is a helper function for path existence checks.
// Returns true if the directory exists, false otherwise.
// Handles permission errors gracefully by returning false without error.
func DirExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}

// DirExistsWithError checks if a directory exists at the given path.
// Unlike DirExists, this function returns an error if the path cannot be accessed
// (e.g., permission denied).
func DirExistsWithError(path string) (bool, error) {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		// Permission denied or other error
		return false, err
	}
	return info.IsDir(), nil
}
