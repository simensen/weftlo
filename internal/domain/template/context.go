// Package template provides domain entities for template rendering.
package template

import (
	"time"

	"github.com/simensen/weftlo/internal/domain/config"
	"github.com/simensen/weftlo/internal/domain/profile"
)

// TemplateContext provides all data sources available to templates during rendering.
// It aggregates profile information, project metadata, and configuration data
// into a single struct that can be bound to Go templates.
//
// All fields are exported to enable Go template access using dot notation,
// e.g., {{ .Profile.Name }} or {{ .ProjectDir }}.
//
// Note: The Profile field exposes the full profile.Profile struct. This decision
// should be reevaluated whenever the Profile struct changes meaningfully to ensure
// template authors only have access to appropriate data.
type TemplateContext struct {
	// Profile contains the full profile information including name, inheritance,
	// and template files. May be nil if no profile is available.
	Profile *profile.Profile

	// ProjectDir is the absolute path to the project directory.
	ProjectDir string

	// GeneratedAt is the timestamp when template rendering was initiated.
	// This is passed in as a parameter (not generated internally) to enable
	// deterministic testing.
	GeneratedAt time.Time

	// ProjectConfig contains project-specific configuration from .weftlo.yaml.
	// May be nil if no project configuration is available.
	ProjectConfig *config.ProjectConfig

	// GlobalConfig contains global configuration from config.yaml.
	// May be nil if no global configuration is available.
	GlobalConfig *config.GlobalConfig

	// Variables contains merged custom variables from global, profile, and project configs.
	// These variables are accessible in templates via {{ .Variables.key }}.
	// Variables are merged with precedence: global < profile < project.
	// Nested maps are supported (e.g., {{ .Variables.database.host }}).
	// May be nil if no variables are defined.
	Variables map[string]interface{}
}

// NewTemplateContext creates a new TemplateContext with all required data sources.
// This is a pure function with no side effects.
//
// Parameters:
//   - p: The profile to use for template rendering (may be nil)
//   - projectDir: The absolute path to the project directory
//   - generatedAt: The timestamp for template rendering (passed in for deterministic testing)
//   - projectConfig: Project configuration from .weftlo.yaml (may be nil)
//   - globalConfig: Global configuration from config.yaml (may be nil)
//   - variables: Merged variables from all sources (may be nil)
//
// Returns a pointer to a new TemplateContext with all fields populated.
func NewTemplateContext(
	p *profile.Profile,
	projectDir string,
	generatedAt time.Time,
	projectConfig *config.ProjectConfig,
	globalConfig *config.GlobalConfig,
	variables map[string]interface{},
) *TemplateContext {
	return &TemplateContext{
		Profile:       p,
		ProjectDir:    projectDir,
		GeneratedAt:   generatedAt,
		ProjectConfig: projectConfig,
		GlobalConfig:  globalConfig,
		Variables:     variables,
	}
}
