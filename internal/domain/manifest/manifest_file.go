package manifest

// ManifestFile represents a single file entry within the installation manifest.
// It tracks checksums for change detection and the source profile for inheritance tracking.
type ManifestFile struct {
	// SourceChecksum is the checksum of the source template file
	SourceChecksum string `json:"source_checksum"`

	// OutputChecksum is the checksum of the rendered output file
	OutputChecksum string `json:"output_checksum"`

	// SourceProfile indicates which profile the file originated from (for inheritance tracking)
	SourceProfile string `json:"source_profile"`

	// TargetCategory is the source directory name that was used to determine the
	// installation target path. This is used for grouping files by target in CLI output.
	TargetCategory string `json:"target_category"`
}
