// Package config contains domain entities for global and project configuration.
package config

// DefaultInstallPrefix is the default value for InstallPrefix when not specified
// or empty in the project configuration file.
const DefaultInstallPrefix = "weftlo"

// ProjectConfig represents the raw content of the project configuration file (.weftlo.yaml)
// as loaded from a project's root directory.
// It contains settings specific to a single project.
type ProjectConfig struct {
	// Profiles is the list of profiles to install for this project.
	// Each profile must follow the vendor/name format (e.g., "beausimensen/golang-backend").
	// Profiles are merged in order (later profiles override earlier profiles).
	// Each profile name is validated using the profile_name validator.
	// When empty, the profile resolution chain will fall back to global default.
	Profiles []string `yaml:"profiles" validate:"omitempty,dive,profile_name"`

	// InstallPrefix is the directory path where rendered profile files will be installed.
	// This path is joined with each template's SourcePath to compute the TargetPath.
	// Supports multi-level paths (e.g., ".claude/my-project"), absolute paths (e.g., "/srv/config"),
	// and relative paths (e.g., "../../shared-config").
	// Defaults to "weftlo" when empty or not specified.
	InstallPrefix string `yaml:"install_prefix,omitempty"`

	// Variables contains custom key-value pairs that can be used in templates.
	// These variables are accessible via {{ .Variables.key }} in templates.
	// Variables defined here override both global and profile variables.
	// When empty, no project-specific variables are defined.
	Variables map[string]interface{} `yaml:"variables,omitempty"`

	// TargetOverrides provides project-level control over where profile content directories
	// are installed. This allows projects to customize installation paths defined by profiles.
	//
	// Map keys are content directory names (e.g., "skills", "prompts").
	// Map values are pointers to strings with the following semantics:
	//   - nil value: Suppress this target entirely (do not install content from this directory)
	//   - Pointer to empty string (""):  Redirect content to project root instead of default target
	//   - Pointer to non-empty string: Redirect content to the specified path
	//
	// Example YAML:
	//   target_overrides:
	//     skills: custom-skills    # Redirect skills to custom-skills/
	//     prompts: null            # Suppress prompts entirely
	//     templates: ""            # Install templates at project root
	//
	// When empty or nil, no overrides are applied and profile-defined targets are used.
	TargetOverrides map[string]*string `yaml:"target_overrides,omitempty"`

	// Ignore contains project-level ignore patterns using gitignore syntax.
	// These patterns specify files and directories that should be excluded
	// from profile content installation.
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
	//   - "*.log"           # Ignore all .log files
	//   - "temp/"           # Ignore the temp directory
	//   - "!important.log"  # But keep important.log
	//
	// When empty or nil, no project-level ignore patterns are applied.
	// These patterns are combined with profile-level ignore patterns during installation.
	Ignore []string `yaml:"ignore,omitempty"`
}
