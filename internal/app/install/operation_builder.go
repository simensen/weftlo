// Package install provides application-layer services for installing profiles
// into projects. It orchestrates rendering, routing, and file operations.
package install

import (
	"path/filepath"

	"github.com/simensen/weftlo/internal/app/dryrun"
	"github.com/simensen/weftlo/internal/app/rendering"
)

// OperationBuilder converts rendering pipeline results into FileOperation slices
// for use in dry-run output or actual installation.
type OperationBuilder struct{}

// NewOperationBuilder creates a new OperationBuilder instance.
func NewOperationBuilder() *OperationBuilder {
	return &OperationBuilder{}
}

// BuildFromRenderResults converts a slice of RenderResult to a slice of FileOperation.
// Each RenderResult is transformed into a FileOperation with:
//   - SourceProfile: extracted from the provided sourceProfile parameter
//   - TargetPath: taken from RenderResult.TargetPath (already routed via Router)
//   - Content: converted from RenderResult.Content string to []byte
//   - OperationType: set to OperationTypeCreate (MVP only supports create operations)
//
// The sourceProfile parameter should be the leaf profile name from the merged profile,
// as the current rendering pipeline does not track per-file source profiles in RenderResult.
//
// Parameters:
//   - results: slice of RenderResult from the rendering pipeline
//   - sourceProfile: the profile name to use as SourceProfile for all operations
//
// Returns a slice of FileOperation ready for formatting or execution.
func (b *OperationBuilder) BuildFromRenderResults(results []rendering.RenderResult, sourceProfile string) []dryrun.FileOperation {
	operations := make([]dryrun.FileOperation, 0, len(results))

	for _, result := range results {
		op := dryrun.NewFileOperation(
			sourceProfile,
			result.TargetPath,
			[]byte(result.Content),
			dryrun.OperationTypeCreate,
		)
		operations = append(operations, op)
	}

	return operations
}

// BuildFromRenderResultsWithProjectDir converts a slice of RenderResult to a slice of FileOperation
// with absolute target paths. This is used for dry-run mode where the formatter needs
// absolute paths to check for existing files and generate diffs.
//
// Parameters:
//   - results: slice of RenderResult from the rendering pipeline
//   - sourceProfile: the profile name to use as SourceProfile for all operations
//   - projectDir: the absolute path to the project directory
//
// Returns a slice of FileOperation with absolute TargetPaths.
func (b *OperationBuilder) BuildFromRenderResultsWithProjectDir(
	results []rendering.RenderResult,
	sourceProfile string,
	projectDir string,
) []dryrun.FileOperation {
	operations := make([]dryrun.FileOperation, 0, len(results))

	for _, result := range results {
		// Compute absolute target path by joining project directory with relative target path
		absoluteTargetPath := filepath.Join(projectDir, result.TargetPath)
		// Normalize to forward slashes for cross-platform consistency
		absoluteTargetPath = filepath.ToSlash(absoluteTargetPath)

		op := dryrun.NewFileOperation(
			sourceProfile,
			absoluteTargetPath,
			[]byte(result.Content),
			dryrun.OperationTypeCreate,
		)
		operations = append(operations, op)
	}

	return operations
}
