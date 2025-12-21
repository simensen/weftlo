// Package update provides application-layer services for update decision logic.
// It categorizes files into create, update, skip, or delete actions based on
// Change Detection status and user-provided flags.
package update

import "github.com/simensen/weftlo/internal/domain/manifest"

// FileDecision represents the update decision for a single file.
// It captures the target path, the original change detection status,
// a human-readable rationale for the decision, and the source profile
// for provenance tracking.
//
// FileDecision is a value type and should be passed by value in slices.
// This ensures predictable behavior when iterating over decisions and
// prevents accidental mutation through aliasing.
type FileDecision struct {
	// TargetPath is the output file location (relative to project root).
	// This is the path where the file would be written or deleted.
	TargetPath string

	// OriginalStatus is the FileStatus from Change Detection that was used
	// to determine this decision. This preserves the input status for
	// auditing and debugging purposes.
	OriginalStatus manifest.FileStatus

	// Rationale is a human-readable explanation of why this decision was made.
	// Rationales are suitable for CLI output and should explain the action
	// and any relevant flags that influenced the decision.
	//
	// Examples:
	//   - "File will be created from profile template"
	//   - "Template source changed, file will be updated"
	//   - "File has local modifications, skipping (use --force to overwrite)"
	Rationale string

	// SourceProfile is the name of the profile that provided this file.
	// This tracks file provenance through the inheritance chain and is
	// useful for understanding which profile a file originated from.
	SourceProfile string
}
