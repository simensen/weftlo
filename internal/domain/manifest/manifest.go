package manifest

import (
	"time"
)

// Manifest represents the installation manifest (manifest.json) stored in projects.
// It tracks the installed profiles and all files with their checksums for change detection.
type Manifest struct {
	// Version is the manifest format version
	Version string `json:"version"`

	// Profiles is the list of installed profiles (in merge order)
	Profiles []string `json:"profiles"`

	// GeneratedAt is the timestamp of manifest generation
	GeneratedAt time.Time `json:"generated_at"`

	// Files is a map keyed by target path to file metadata
	Files map[string]ManifestFile `json:"files"`
}
