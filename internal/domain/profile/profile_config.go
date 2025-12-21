// Package profile contains domain entities for profile configuration and template management.
package profile

// ProfileConfig represents the raw content of profile.yaml files as loaded from disk.
// It contains the profile identifier and optional inheritance configuration.
type ProfileConfig struct {
	// Name is the profile identifier.
	Name string `yaml:"name"`

	// InheritsFrom is the optional parent profile name for inheritance.
	// When empty, the profile has no parent.
	InheritsFrom string `yaml:"inherits_from,omitempty"`

	// Variables contains custom key-value pairs that can be used in templates.
	// These variables are accessible via {{ .Variables.key }} in templates.
	// Variables defined here override global variables but are overridden by project variables.
	// When empty, no profile-specific variables are defined.
	Variables map[string]interface{} `yaml:"variables,omitempty"`

	// Content defines content routing configuration for this profile.
	// It specifies where installable content is located within the profile (Root),
	// how content subdirectories map to installation targets (Targets), and the
	// fallback target for unmapped content (DefaultTarget).
	// When empty (zero-value struct), the profile uses default content routing behavior.
	// See ContentConfig for detailed field documentation and merge semantics.
	Content ContentConfig `yaml:"content,omitempty"`

	// Ignore contains inline ignore patterns using gitignore syntax.
	// These patterns specify files and directories that should be excluded during
	// profile installation. Patterns follow standard gitignore rules:
	//   - Lines starting with # are comments
	//   - Lines starting with ! negate a previous pattern
	//   - * matches anything except /
	//   - ** matches anything including /
	//   - Trailing / matches directories only
	//
	// Merge Behavior:
	//   - Additive: child profile patterns are appended to parent profile patterns.
	//   - Parent patterns are evaluated first, then child patterns.
	//   - A child pattern can negate (!) a parent pattern.
	//
	// Note: The actual merge logic is deferred to the Profile Loader Updates spec.
	// When nil or empty, no profile-specific ignore patterns are defined.
	Ignore []string `yaml:"ignore,omitempty"`
}
