package manifest

import "time"

// ManifestVersion is the current manifest format version.
const ManifestVersion = "1.0.0"

// ManifestGenerator creates Manifest structs from rendered template results.
// It computes checksums for each file and populates all manifest fields.
type ManifestGenerator struct{}

// NewManifestGenerator creates a new ManifestGenerator instance.
func NewManifestGenerator() *ManifestGenerator {
	return &ManifestGenerator{}
}

// Generate creates a fully-populated Manifest from the provided profile names
// and slice of rendered files.
//
// The method validates each RenderedFile before processing:
//   - TargetPath must not be empty
//   - SourceContent must not be nil
//   - OutputContent must not be nil
//
// Empty or nil files slice is handled gracefully, returning a valid manifest
// with an empty Files map.
//
// Parameters:
//   - profileNames: The list of installed profiles (in merge order)
//   - files: Slice of RenderedFile structs from the rendering pipeline
//
// Returns:
//   - A fully-populated Manifest struct (not written to disk)
//   - ManifestGenerationError if any RenderedFile has invalid data
func (g *ManifestGenerator) Generate(profileNames []string, files []RenderedFile) (*Manifest, error) {
	// Validate all files first
	for _, file := range files {
		if err := g.validateRenderedFile(file); err != nil {
			return nil, err
		}
	}

	// Create the manifest with initialized Files map
	manifest := &Manifest{
		Version:  ManifestVersion,
		Profiles: profileNames,
		Files:    make(map[string]ManifestFile),
	}

	// Populate Files map from rendered files
	for _, file := range files {
		manifestFile := ManifestFile{
			SourceChecksum: ComputeChecksum(file.SourceContent),
			OutputChecksum: ComputeChecksum(file.OutputContent),
			SourceProfile:  file.SourceProfile,
			TargetCategory: file.TargetCategory,
		}
		manifest.Files[file.TargetPath] = manifestFile
	}

	// Set GeneratedAt to completion timestamp
	manifest.GeneratedAt = time.Now()

	return manifest, nil
}

// validateRenderedFile checks that a RenderedFile has valid data for manifest generation.
func (g *ManifestGenerator) validateRenderedFile(file RenderedFile) error {
	if file.TargetPath == "" {
		return &ManifestGenerationError{
			TargetPath: "",
			Reason:     "target path cannot be empty",
		}
	}

	if file.SourceContent == nil {
		return &ManifestGenerationError{
			TargetPath: file.TargetPath,
			Reason:     "source content cannot be nil",
		}
	}

	if file.OutputContent == nil {
		return &ManifestGenerationError{
			TargetPath: file.TargetPath,
			Reason:     "output content cannot be nil",
		}
	}

	return nil
}
