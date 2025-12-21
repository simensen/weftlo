package manifest

import "fmt"

// ManifestGenerationError is returned when manifest generation fails
// due to invalid input data (e.g., empty target path, nil content).
type ManifestGenerationError struct {
	// TargetPath identifies the file that caused the error.
	// Empty string if the error is about an empty target path.
	TargetPath string

	// Reason provides a descriptive explanation of why generation failed.
	Reason string
}

// Error implements the error interface, providing a formatted message
// that includes the file path (if available) and reason for failure.
func (e *ManifestGenerationError) Error() string {
	if e.TargetPath == "" {
		return fmt.Sprintf("manifest generation error: %s", e.Reason)
	}
	return fmt.Sprintf("manifest generation error for '%s': %s", e.TargetPath, e.Reason)
}

// ChangeDetectionError is returned when change detection fails
// due to filesystem read failures or other errors during comparison.
type ChangeDetectionError struct {
	// TargetPath identifies the file that caused the error.
	// Empty string if the error is not specific to a particular file.
	TargetPath string

	// Reason provides a descriptive explanation of why change detection failed.
	Reason string
}

// Error implements the error interface, providing a formatted message
// that includes the file path (if available) and reason for failure.
func (e *ChangeDetectionError) Error() string {
	if e.TargetPath == "" {
		return fmt.Sprintf("change detection error: %s", e.Reason)
	}
	return fmt.Sprintf("change detection error for '%s': %s", e.TargetPath, e.Reason)
}
