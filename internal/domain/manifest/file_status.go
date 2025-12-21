package manifest

// FileStatus represents the change detection status of a file when comparing
// current source templates and output files against stored manifest checksums.
//
// The status is determined by comparing the current checksums of both the source
// template and the output file against the checksums stored in the installation
// manifest. This enables the CLI to identify which files need updating and which
// have potential conflicts requiring user intervention.
//
// Example usage:
//
//	changes, err := DetectChanges(manifest, profile, fs, projectRoot)
//	if err != nil {
//	    return err
//	}
//	for path, status := range changes {
//	    switch status {
//	    case FileStatusUnchanged:
//	        // File is up-to-date, no action needed
//	    case FileStatusSourceChanged:
//	        // Template updated, safe to re-render
//	    case FileStatusUserModified:
//	        // User made local changes, preserve or review
//	    case FileStatusConflict:
//	        // Both changed, requires user decision
//	    case FileStatusNew:
//	        // New file to install
//	    case FileStatusRemoved:
//	        // File no longer in profile
//	    }
//	}
type FileStatus string

const (
	// FileStatusUnchanged indicates both the source template and output file
	// checksums match their manifest values. The file is up-to-date and requires
	// no action.
	FileStatusUnchanged FileStatus = "unchanged"

	// FileStatusSourceChanged indicates the source template checksum differs from
	// the manifest's SourceChecksum while the output file checksum matches the
	// manifest's OutputChecksum. This means the profile templates have been updated
	// and the file can be safely re-rendered without losing user changes.
	FileStatusSourceChanged FileStatus = "source_changed"

	// FileStatusUserModified indicates the output file checksum differs from the
	// manifest's OutputChecksum while the source template checksum matches the
	// manifest's SourceChecksum. This means the user has made local changes to the
	// output file that should be preserved or reviewed before any update.
	FileStatusUserModified FileStatus = "user_modified"

	// FileStatusConflict indicates both the source template checksum and the output
	// file checksum differ from their manifest values. Both the template and the
	// local file have diverged from the installed state, requiring user decision
	// on how to proceed (e.g., merge, overwrite, or keep local).
	FileStatusConflict FileStatus = "conflict"

	// FileStatusNew indicates the file exists in the current profile's template
	// enumeration but not in the manifest's Files map. This file should be
	// installed but was not part of the original installation.
	FileStatusNew FileStatus = "new"

	// FileStatusRemoved indicates the file exists in the manifest's Files map but
	// not in the current profile's template enumeration. This file was installed
	// previously but is no longer part of the profile and may need to be cleaned up.
	FileStatusRemoved FileStatus = "removed"
)
