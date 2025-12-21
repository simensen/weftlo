package update

import "fmt"

// UpdatePlanError is returned when update planning fails due to filesystem
// errors, rendering failures, or invalid input during the planning phase.
//
// This error type follows the pattern established by ManifestGenerationError
// and ChangeDetectionError in internal/domain/manifest/errors.go.
type UpdatePlanError struct {
	// TargetPath identifies the file that caused the error.
	// Empty string if the error is not specific to a particular file.
	TargetPath string

	// Reason provides a descriptive explanation of why planning failed.
	Reason string
}

// Error implements the error interface, providing a formatted message
// that includes the file path (if available) and reason for failure.
func (e *UpdatePlanError) Error() string {
	if e.TargetPath == "" {
		return fmt.Sprintf("update plan error: %s", e.Reason)
	}
	return fmt.Sprintf("update plan error for '%s': %s", e.TargetPath, e.Reason)
}
