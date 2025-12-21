package manifest

import (
	"path/filepath"
	"strings"

	"github.com/spf13/afero"

	"github.com/simensen/weftlo/internal/app/routing"
	"github.com/simensen/weftlo/internal/infrastructure/profile"
)

// FileChangeInfo represents the change detection result for a single file.
// It includes both the change status and the target category from routing.
type FileChangeInfo struct {
	// Status indicates the change detection status of the file.
	Status FileStatus

	// TargetCategory is the source directory name that was used to determine
	// the target installation path during routing.
	TargetCategory string
}

// DetectChanges compares current source templates and output files against stored
// manifest checksums, categorizing each file into one of six statuses.
//
// The function iterates through both the manifest files (to detect changes and removals)
// and the profile templates (to detect new files), computing current checksums and
// comparing them against the manifest's stored values.
//
// Files routed to a suppressed target (RoutedFile.Suppressed == true) are silently
// excluded from the results and not considered in change detection.
//
// Parameters:
//   - manifest: The installation manifest containing recorded checksums for installed files
//   - mergedProfile: The merged profile providing access to current source templates
//   - fs: The filesystem abstraction for reading current file contents (enables testability)
//   - projectRoot: The project root path for resolving output file locations
//   - router: The Router for computing target installation paths from source paths
//
// Returns:
//   - A map of target paths to FileChangeInfo values containing status and target category
//   - An error for filesystem read failures (missing output files are handled as status, not errors)
//
// Status Detection Logic:
//   - Unchanged: Both source and output checksums match manifest values
//   - SourceChanged: Source checksum differs, output checksum matches
//   - UserModified: Output checksum differs, source checksum matches
//   - Conflict: Both checksums differ from manifest values
//   - New: File exists in profile but not in manifest
//   - Removed: File exists in manifest but not in profile
func DetectChanges(manifest *Manifest, mergedProfile *profile.MergedProfile, fs afero.Fs, projectRoot string, router routing.Router) (map[string]FileChangeInfo, error) {
	result := make(map[string]FileChangeInfo)

	// Build a set of current profile target paths for efficient lookup
	// Also maintain a mapping from target path back to source path for checksum computation
	// And track target category for each routed path
	profileTargetPaths := make(map[string]bool)
	targetToSource := make(map[string]string)
	targetToCategory := make(map[string]string)
	templateFiles := profile.EnumerateOutputTemplates(mergedProfile)

	for _, tf := range templateFiles {
		// Strip .tmpl suffix before routing (Router expects source paths without .tmpl)
		sourcePath := stripTmplSuffix(tf.SourcePath)

		// Use Router for path computation
		routedFile, err := router.Route(sourcePath)
		if err != nil {
			return nil, &ChangeDetectionError{
				TargetPath: tf.SourcePath,
				Reason:     "failed to route source path: " + err.Error(),
			}
		}

		// Skip suppressed files silently
		if routedFile.Suppressed {
			continue
		}

		targetPath := routedFile.TargetPath
		profileTargetPaths[targetPath] = true
		targetToSource[targetPath] = tf.SourcePath
		targetToCategory[targetPath] = routedFile.TargetCategory
	}

	// Build a set of profile target paths for checking removed files
	// This allows us to identify files in manifest that are no longer in profile
	profilePathsSet := make(map[string]bool)
	for targetPath := range profileTargetPaths {
		profilePathsSet[targetPath] = true
	}

	// Iterate through manifest files to detect Unchanged, SourceChanged, UserModified, Conflict, and Removed
	for targetPath, manifestFile := range manifest.Files {
		// Check if file still exists in profile
		if !profilePathsSet[targetPath] {
			// File is in manifest but not in current profile - Removed
			// Use the TargetCategory from the manifest if available
			result[targetPath] = FileChangeInfo{
				Status:         FileStatusRemoved,
				TargetCategory: manifestFile.TargetCategory,
			}
			continue
		}

		// Get the source path for this target
		sourcePath := targetToSource[targetPath]

		// File exists in both manifest and profile - compute current checksums
		sourceChecksum, err := computeSourceChecksum(mergedProfile, fs, sourcePath)
		if err != nil {
			return nil, &ChangeDetectionError{
				TargetPath: targetPath,
				Reason:     "failed to read source template: " + err.Error(),
			}
		}

		outputChecksum, err := computeOutputChecksum(fs, projectRoot, targetPath)
		if err != nil {
			// Missing output file is treated as user modification (file was deleted)
			// This means the output checksum does not match the manifest
			outputChecksum = "" // Empty checksum won't match manifest
		}

		// Determine status based on checksum comparisons
		sourceMatches := sourceChecksum == manifestFile.SourceChecksum
		outputMatches := outputChecksum == manifestFile.OutputChecksum

		var status FileStatus
		switch {
		case sourceMatches && outputMatches:
			status = FileStatusUnchanged
		case !sourceMatches && outputMatches:
			status = FileStatusSourceChanged
		case sourceMatches && !outputMatches:
			status = FileStatusUserModified
		case !sourceMatches && !outputMatches:
			status = FileStatusConflict
		}

		result[targetPath] = FileChangeInfo{
			Status:         status,
			TargetCategory: targetToCategory[targetPath],
		}

		// Remove from profile paths set (we've processed this file)
		delete(profileTargetPaths, targetPath)
	}

	// Remaining profile target paths are New files (not in manifest)
	for targetPath := range profileTargetPaths {
		result[targetPath] = FileChangeInfo{
			Status:         FileStatusNew,
			TargetCategory: targetToCategory[targetPath],
		}
	}

	return result, nil
}

// computeSourceChecksum reads the source template file and computes its checksum.
// It uses the MergedProfile to resolve the source path to an absolute filesystem path.
func computeSourceChecksum(mergedProfile *profile.MergedProfile, fs afero.Fs, sourcePath string) (string, error) {
	// Resolve the source path to an absolute path
	absolutePath, found := mergedProfile.Resolve(sourcePath)
	if !found {
		return "", &ChangeDetectionError{
			TargetPath: sourcePath,
			Reason:     "source path not found in merged profile",
		}
	}

	// Read the source template content
	content, err := afero.ReadFile(fs, absolutePath)
	if err != nil {
		return "", err
	}

	return ComputeChecksum(content), nil
}

// computeOutputChecksum reads the output file from the project and computes its checksum.
// It combines the project root with the target path to locate the file.
func computeOutputChecksum(fs afero.Fs, projectRoot, targetPath string) (string, error) {
	outputPath := filepath.Join(projectRoot, targetPath)

	// Read the output file content
	content, err := afero.ReadFile(fs, outputPath)
	if err != nil {
		return "", err
	}

	return ComputeChecksum(content), nil
}

// stripTmplSuffix removes the .tmpl suffix from a source path if present.
// This is called before routing since Router expects source paths without .tmpl suffix.
func stripTmplSuffix(sourcePath string) string {
	const tmplSuffix = ".tmpl"
	if strings.HasSuffix(sourcePath, tmplSuffix) {
		return sourcePath[:len(sourcePath)-len(tmplSuffix)]
	}
	return sourcePath
}
