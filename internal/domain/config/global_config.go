// Package config contains domain entities for global and project configuration.
package config

// ProfileOverrideConfig represents per-profile override settings that can be configured
// at the global level. This allows users to set profile-specific target overrides that
// apply whenever a particular profile is used.
//
// Example YAML structure within global config:
//
//	profile_overrides:
//	  simensen/git-profile:
//	    target_overrides:
//	      agents: .claude/agents/simensen
//	  other-vendor/other-profile:
//	    target_overrides:
//	      standards: .project-standards
type ProfileOverrideConfig struct {
	// TargetOverrides provides profile-specific control over where content directories
	// are installed when this profile is used.
	//
	// Map keys are content directory names (e.g., "skills", "prompts", "agents").
	// Map values are pointers to strings with the following semantics:
	//   - nil value: Suppress this target entirely (do not install content from this directory)
	//   - Pointer to empty string (""):  Redirect content to project root instead of default target
	//   - Pointer to non-empty string: Redirect content to the specified path
	//
	// Override values support Go template syntax with full Sprig function support.
	// Template evaluation is deferred until all variables are merged (global + profile + project).
	//
	// When empty or nil, no profile-specific overrides are applied for this profile.
	TargetOverrides map[string]*string `yaml:"target_overrides,omitempty"`
}

// GlobalConfig represents the raw content of the global configuration file (config.yaml)
// as loaded from the user's configuration directory.
// It contains settings that apply across all projects.
type GlobalConfig struct {
	// DefaultProfile is the profile to use when a project does not specify one.
	// This follows the vendor/name format (e.g., "beausimensen/default").
	// When empty, no default profile is configured.
	DefaultProfile string `yaml:"default_profile,omitempty"`

	// InstallPrefix is the default directory path where rendered profile files will be installed.
	// This can be overridden by project configuration or command line flags.
	// When empty, no global default install prefix is configured.
	InstallPrefix string `yaml:"install_prefix,omitempty"`

	// Variables contains custom key-value pairs that can be used in templates.
	// These variables are accessible via {{ .Variables.key }} in templates.
	// Variables defined here are overridden by profile and project variables.
	// When empty, no global variables are defined.
	Variables map[string]interface{} `yaml:"variables,omitempty"`

	// TargetOverrides provides global-level control over where profile content directories
	// are installed. These overrides apply to all projects unless overridden by project config.
	//
	// Map keys are content directory names (e.g., "skills", "prompts", "agents").
	// Map values are pointers to strings with the following semantics:
	//   - nil value: Suppress this target entirely (do not install content from this directory)
	//   - Pointer to empty string (""):  Redirect content to project root instead of default target
	//   - Pointer to non-empty string: Redirect content to the specified path
	//
	// Override values support Go template syntax with full Sprig function support.
	// Template evaluation is deferred until all variables are merged (global + profile + project).
	//
	// Example YAML:
	//   target_overrides:
	//     agents: .claude/agents/{{ .Variables.namespace }}  # Universal default with variable
	//     commands: .claude/commands/{{ .Variables.namespace }}
	//     cursor: null  # Suppress cursor content globally
	//
	// When empty or nil, no global overrides are applied and profile-defined targets are used.
	TargetOverrides map[string]*string `yaml:"target_overrides,omitempty"`

	// ProfileOverrides provides per-profile override settings at the global level.
	// This allows users to configure target overrides that only apply when using specific profiles.
	//
	// Map keys are profile names in vendor/name format (e.g., "simensen/git-profile").
	// Map values are ProfileOverrideConfig structs containing the overrides for that profile.
	//
	// Override precedence (lowest to highest):
	//   1. Profile targets from profile.yaml content.targets
	//   2. Global universal overrides (this TargetOverrides field)
	//   3. Global profile-specific overrides (this ProfileOverrides field)
	//   4. Project overrides from .weftlo.yaml target_overrides
	//
	// Example YAML:
	//   profile_overrides:
	//     simensen/git-profile:
	//       target_overrides:
	//         agents: .claude/agents/simensen  # Override for this specific profile
	//     other-vendor/other-profile:
	//       target_overrides:
	//         standards: .project-standards
	//
	// When empty or nil, no profile-specific global overrides are applied.
	ProfileOverrides map[string]ProfileOverrideConfig `yaml:"profile_overrides,omitempty"`

	// Ignore contains global-level ignore patterns using gitignore syntax.
	// These patterns specify files and directories that should be excluded
	// from profile content installation across all projects.
	//
	// Patterns follow standard gitignore syntax:
	//   - "*" matches any sequence of characters except "/"
	//   - "?" matches any single character except "/"
	//   - "**" matches any sequence of characters including "/"
	//   - "!" prefix negates a pattern (re-includes a previously ignored file)
	//   - "/" suffix matches directories only
	//   - "#" prefix indicates a comment line
	//
	// Example patterns:
	//   - "**/*.bak"     # Ignore all .bak files in any directory
	//   - "**/temp/**"   # Ignore all temp directories
	//
	// Global ignore patterns merge with profile and project patterns:
	//   - Merge order: profile patterns first, then global patterns, then project patterns appended
	//   - Patterns only add to the ignore list; they cannot remove patterns from other sources
	//
	// When empty or nil, no global ignore patterns are applied.
	Ignore []string `yaml:"ignore,omitempty"`
}
