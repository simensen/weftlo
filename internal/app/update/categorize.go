package update

import "github.com/simensen/weftlo/internal/domain/manifest"

// Category represents the update action category for a file.
// It is used internally by CategorizeFile to indicate which UpdatePlan
// slice a file should be added to.
type Category string

const (
	// CategoryCreate indicates the file should be added to UpdatePlan.ToCreate.
	// Files with FileStatusNew are assigned this category.
	CategoryCreate Category = "create"

	// CategoryUpdate indicates the file should be added to UpdatePlan.ToUpdate.
	// Files with FileStatusSourceChanged are assigned this category.
	// When Force is true, FileStatusUserModified and FileStatusConflict
	// files are also assigned this category.
	CategoryUpdate Category = "update"

	// CategorySkip indicates the file should be added to UpdatePlan.ToSkip.
	// By default, FileStatusUserModified and FileStatusConflict files
	// are assigned this category, with warnings in their rationale.
	CategorySkip Category = "skip"

	// CategoryDelete indicates the file should be added to UpdatePlan.ToDelete.
	// FileStatusRemoved files are assigned this category only when
	// RemoveOrphans is true.
	CategoryDelete Category = "delete"

	// CategoryNone indicates the file should not be added to any category.
	// FileStatusUnchanged files are assigned this category since no
	// action is needed. FileStatusRemoved files also receive this category
	// when RemoveOrphans is false.
	CategoryNone Category = "none"
)

// CategorizeFile determines the update action category for a file based on
// its change detection status and the provided update options.
//
// The function implements the following decision logic:
//   - New: Always goes to CategoryCreate
//   - SourceChanged: Always goes to CategoryUpdate
//   - UserModified: CategorySkip by default, CategoryUpdate when Force is true
//   - Conflict: CategorySkip by default, CategoryUpdate when Force is true
//   - Removed: CategoryNone by default, CategoryDelete when RemoveOrphans is true
//   - Unchanged: Always CategoryNone (no action needed)
//
// The returned rationale provides a human-readable explanation of the decision,
// suitable for CLI output. Rationales include actionable hints when relevant
// (e.g., "use --force to overwrite").
//
// Parameters:
//   - status: The FileStatus from Change Detection indicating the file's current state
//   - options: UpdateOptions controlling Force and RemoveOrphans behavior
//
// Returns:
//   - category: The Category indicating which UpdatePlan slice to use
//   - rationale: A human-readable explanation of the categorization decision
func CategorizeFile(status manifest.FileStatus, options UpdateOptions) (Category, string) {
	switch status {
	case manifest.FileStatusNew:
		return CategoryCreate, "File will be created from profile template"

	case manifest.FileStatusSourceChanged:
		return CategoryUpdate, "Template source changed, file will be updated"

	case manifest.FileStatusUserModified:
		if options.Force {
			return CategoryUpdate, "File has local modifications, overwriting due to --force"
		}
		return CategorySkip, "File has local modifications, skipping (use --force to overwrite)"

	case manifest.FileStatusConflict:
		if options.Force {
			return CategoryUpdate, "Both source and local file changed, overwriting due to --force"
		}
		return CategorySkip, "Both source and local file changed, skipping (use --force to overwrite)"

	case manifest.FileStatusRemoved:
		if options.RemoveOrphans {
			return CategoryDelete, "File no longer part of profile, will be deleted"
		}
		return CategoryNone, "File no longer in profile, keeping (use --remove-orphans to delete)"

	case manifest.FileStatusUnchanged:
		return CategoryNone, "File is up to date, no action needed"

	default:
		return CategoryNone, "Unknown file status, no action taken"
	}
}
