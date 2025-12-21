package profile

// TemplateFile represents a single template file within a profile.
// It tracks the source location, computed output path, partial status, and checksum.
type TemplateFile struct {
	// SourcePath is the relative path within the profile directory.
	SourcePath string `yaml:"source_path"`

	// TargetPath is the computed output path with Install Prefix handling.
	TargetPath string `yaml:"target_path"`

	// IsPartial indicates whether the file is a partial (derived from underscore prefix convention).
	// Partial files are included in other templates but not output directly.
	IsPartial bool `yaml:"is_partial"`

	// Checksum is the file checksum for change detection.
	// Empty when the checksum has not been computed.
	Checksum string `yaml:"checksum,omitempty"`
}
