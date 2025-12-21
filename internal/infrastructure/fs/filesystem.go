// Package fs provides filesystem abstraction using afero for testable file operations.
package fs

import (
	"github.com/spf13/afero"
)

// New creates a new filesystem instance for production use.
// It returns afero.NewOsFs() which provides access to the real operating system filesystem.
// For testing, use afero.NewMemMapFs() directly which is interchangeable with this return type.
func New() afero.Fs {
	return afero.NewOsFs()
}
