package update

// UpdateOptions controls the behavior of update decision-making.
// Default values (false for all boolean fields) implement safe behavior
// that preserves user modifications and doesn't delete files.
//
// UpdateOptions is designed to be passed as a value to decision-making
// functions, following the pattern used in Docker Compose where
// --force and --remove-orphans are independent flags.
//
// Example usage:
//
//	// Safe defaults - skip user-modified files, don't delete orphans
//	safeOptions := UpdateOptions{}
//
//	// Force overwrite of user modifications
//	forceOptions := UpdateOptions{Force: true}
//
//	// Delete orphaned files
//	cleanupOptions := UpdateOptions{RemoveOrphans: true}
//
//	// Both force and cleanup
//	aggressiveOptions := UpdateOptions{Force: true, RemoveOrphans: true}
type UpdateOptions struct {
	// Force enables overwriting of files that have local modifications.
	// When true:
	//   - FileStatusUserModified files are moved from ToSkip to ToUpdate
	//   - FileStatusConflict files are moved from ToSkip to ToUpdate
	// When false (default):
	//   - FileStatusUserModified and FileStatusConflict files are skipped
	//   - A warning rationale is provided explaining how to force the update
	//
	// Force does NOT affect Removed files; use RemoveOrphans instead.
	Force bool

	// RemoveOrphans enables deletion of files that are no longer part of
	// the current profile. When true:
	//   - FileStatusRemoved files are added to ToDelete
	// When false (default):
	//   - FileStatusRemoved files are excluded from all categories
	//
	// RemoveOrphans is independent from Force, following the Docker Compose
	// model where --force and --remove-orphans are separate concerns.
	RemoveOrphans bool
}
