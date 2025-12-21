// Package testutil provides shared test helper functions and utilities.
// This package is intended for use in test files only.
package testutil

import (
	"path/filepath"
	"testing"

	"github.com/spf13/afero"
)

// ContainsString checks if string s contains the substring substr.
// This is a simple helper to avoid string manipulation in test assertions.
func ContainsString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// TestFile represents a file to be created in the test filesystem.
type TestFile struct {
	Path    string
	Content string
}

// SetupMemFs creates an in-memory filesystem and populates it with the provided files.
// This helper enables fast, isolated tests without touching the real filesystem.
//
// Usage:
//
//	memFs := testutil.SetupMemFs(t,
//	    testutil.TestFile{Path: "/config/config.yaml", Content: "default_profile: vendor/test"},
//	    testutil.TestFile{Path: "/project/.weftlo.yaml", Content: "profile: vendor/proj"},
//	)
func SetupMemFs(t *testing.T, files ...TestFile) afero.Fs {
	t.Helper()
	memFs := afero.NewMemMapFs()

	for _, f := range files {
		// Create parent directory
		dir := filepath.Dir(f.Path)
		if err := memFs.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("failed to create directory %s: %v", dir, err)
		}

		// Write the file
		if err := afero.WriteFile(memFs, f.Path, []byte(f.Content), 0644); err != nil {
			t.Fatalf("failed to write file %s: %v", f.Path, err)
		}
	}

	return memFs
}

// SetupMemFsWithDirs creates an in-memory filesystem with the specified directories.
// Use this when you only need directories without files.
func SetupMemFsWithDirs(t *testing.T, dirs ...string) afero.Fs {
	t.Helper()
	memFs := afero.NewMemMapFs()

	for _, dir := range dirs {
		if err := memFs.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("failed to create directory %s: %v", dir, err)
		}
	}

	return memFs
}
