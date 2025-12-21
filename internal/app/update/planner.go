package update

import (
	"github.com/simensen/weftlo/internal/app/rendering"
	"github.com/simensen/weftlo/internal/domain/manifest"
	"github.com/simensen/weftlo/internal/domain/template"
	infraprofile "github.com/simensen/weftlo/internal/infrastructure/profile"
)

// PlanUpdate creates an UpdatePlan by analyzing change detection results and
// categorizing files into create, update, skip, or delete actions.
//
// The function iterates through all FileStatus entries from change detection,
// applies the categorization logic based on UpdateOptions, and re-renders
// templates for files that need to be created or updated.
//
// Parameters:
//   - changeResults: Map of target paths to FileStatus from Change Detection
//   - mergedProfile: The merged profile for accessing templates and re-rendering
//   - options: UpdateOptions controlling Force and RemoveOrphans behavior
//   - engine: Template engine for re-rendering templates
//   - projectDir: The project directory path for template context
//
// Returns:
//   - UpdatePlan with categorized file decisions
//   - Error if any filesystem or rendering operation fails
//
// The function follows a plan-then-execute pattern where decisions are computed
// before any file writes occur. This allows for dry-run preview and user
// confirmation before making changes.
//
// Categorization logic:
//   - New: Goes to ToCreate (file will be created)
//   - SourceChanged: Goes to ToUpdate (file will be re-rendered)
//   - UserModified: Goes to ToSkip by default, ToUpdate when Force=true
//   - Conflict: Goes to ToSkip by default, ToUpdate when Force=true
//   - Removed: Goes to ToDelete only when RemoveOrphans=true
//   - Unchanged: Excluded from all categories (no action needed)
func PlanUpdate(
	changeResults map[string]manifest.FileStatus,
	mergedProfile *infraprofile.MergedProfile,
	options UpdateOptions,
	engine *template.TemplateEngine,
	projectDir string,
) (UpdatePlan, error) {
	plan := UpdatePlan{
		ToCreate: []FileDecision{},
		ToUpdate: []FileDecision{},
		ToSkip:   []FileDecision{},
		ToDelete: []FileDecision{},
	}

	// Early return for empty change results
	if len(changeResults) == 0 {
		return plan, nil
	}

	// Collect files that need rendering (create or update)
	filesToRender := make(map[string]manifest.FileStatus)

	// First pass: categorize all files and collect those needing rendering
	for targetPath, status := range changeResults {
		category, rationale := CategorizeFile(status, options)

		switch category {
		case CategoryCreate, CategoryUpdate:
			// These need rendering - collect for batch rendering
			filesToRender[targetPath] = status
		case CategorySkip:
			decision := FileDecision{
				TargetPath:     targetPath,
				OriginalStatus: status,
				Rationale:      rationale,
				SourceProfile:  mergedProfile.LeafProfileName(),
			}
			plan.ToSkip = append(plan.ToSkip, decision)
		case CategoryDelete:
			decision := FileDecision{
				TargetPath:     targetPath,
				OriginalStatus: status,
				Rationale:      rationale,
				SourceProfile:  mergedProfile.LeafProfileName(),
			}
			plan.ToDelete = append(plan.ToDelete, decision)
		case CategoryNone:
			// Unchanged files or Removed files without RemoveOrphans are excluded
			continue
		}
	}

	// If no files need rendering, return early
	if len(filesToRender) == 0 {
		return plan, nil
	}

	// Render templates using the RenderingPipeline
	pipeline := rendering.NewRenderingPipeline(engine)
	renderResults, err := pipeline.RenderAllWithOptions(mergedProfile, projectDir, nil)
	if err != nil {
		return plan, &UpdatePlanError{
			TargetPath: "",
			Reason:     "failed to render templates: " + err.Error(),
		}
	}

	// Build a map of rendered results by target path for efficient lookup
	renderedByPath := make(map[string]rendering.RenderResult)
	for _, result := range renderResults {
		renderedByPath[result.TargetPath] = result
	}

	// Second pass: create decisions for files that needed rendering
	for targetPath, status := range filesToRender {
		category, rationale := CategorizeFile(status, options)

		decision := FileDecision{
			TargetPath:     targetPath,
			OriginalStatus: status,
			Rationale:      rationale,
			SourceProfile:  mergedProfile.LeafProfileName(),
		}

		switch category {
		case CategoryCreate:
			plan.ToCreate = append(plan.ToCreate, decision)
		case CategoryUpdate:
			plan.ToUpdate = append(plan.ToUpdate, decision)
		}
	}

	return plan, nil
}
