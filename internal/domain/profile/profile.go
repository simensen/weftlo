package profile

import (
	"github.com/simensen/weftlo/internal/domain/ignore"
)

// Profile represents a fully loaded and resolved profile object in memory.
// It contains the profile metadata and the enumerated template files.
type Profile struct {
	// Name is the profile identifier.
	Name string `yaml:"name"`

	// InheritsFrom is the optional parent profile name for inheritance.
	// When empty, the profile has no parent.
	InheritsFrom string `yaml:"inherits_from,omitempty"`

	// TemplateFiles is the slice of enumerated template files for this profile.
	TemplateFiles []TemplateFile `yaml:"template_files,omitempty"`

	// ContentConfig carries content routing configuration through profile loading.
	// It specifies where installable content is located within the profile (Root),
	// how content subdirectories map to installation targets (Targets), and the
	// fallback target for unmapped content (DefaultTarget).
	// This field is populated from ProfileConfig.Content during profile loading.
	// Default values (Root: "content", DefaultTarget: "weftlo") are applied
	// when values are not explicitly set.
	ContentConfig ContentConfig `yaml:"-"`

	// Matcher carries ignore patterns for later merging in MergedProfile.
	// It is populated during profile loading from .weftlo.ignore files
	// and inline ignore patterns from profile.yaml.
	// When no ignore patterns exist, this field holds an empty/no-op matcher
	// that matches nothing.
	Matcher ignore.Matcher `yaml:"-"`

	// Variables carries custom key-value pairs from profile.yaml for use in templates.
	// This field is copied from ProfileConfig.Variables during profile loading.
	// Variables defined here are merged with deep merge semantics through the
	// profile inheritance chain, with child profile values overriding parent values.
	// When nil or empty, no profile-specific variables are defined.
	Variables map[string]interface{} `yaml:"-"`
}

// ApplyContentConfigDefaults applies default values to ContentConfig fields
// that are not explicitly set (empty strings).
// Default values:
//   - Root: DefaultContentRoot ("content")
//   - DefaultTarget: DefaultTargetPath ("weftlo")
//
// This method modifies the Profile in place.
func (p *Profile) ApplyContentConfigDefaults() {
	if p.ContentConfig.Root == "" {
		p.ContentConfig.Root = DefaultContentRoot
	}
	if p.ContentConfig.DefaultTarget == "" {
		p.ContentConfig.DefaultTarget = DefaultTargetPath
	}
}
