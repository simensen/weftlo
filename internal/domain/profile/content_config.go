// Package profile contains domain entities for profile configuration and template management.
package profile

// DefaultContentRoot is the default value for ContentConfig.Root when not specified
// or empty in the profile configuration file. This specifies the default subdirectory
// within a profile that contains installable content.
const DefaultContentRoot = "content"

// DefaultTargetPath is the default installation target path for content that does not
// have a specific target mapping. Content without an explicit target in the Targets map
// will be installed to this path relative to the project root.
const DefaultTargetPath = "weftlo"

// ContentConfig defines content routing configuration for a profile.
// It specifies where installable content is located within the profile (Root),
// how content subdirectories map to installation targets (Targets), and the
// fallback target for unmapped content (DefaultTarget).
//
// ContentConfig supports inheritance through profile hierarchies. When profiles
// inherit from parent profiles, ContentConfig values are merged according to the
// following semantics (actual merge logic is implemented in the Profile Loader):
//
// Merge Behavior:
//   - Root: child value overrides parent value only if child value is non-empty.
//     When child Root is empty string, parent Root is preserved.
//   - Targets: deep merge where child keys override parent keys for the same key.
//     Parent keys that are not present in child are preserved.
//     Example: parent {a: "x", b: "y"} + child {b: "z", c: "w"} = {a: "x", b: "z", c: "w"}
//   - DefaultTarget: child value overrides parent value only if child value is non-empty.
//     When child DefaultTarget is empty string, parent DefaultTarget is preserved.
//
// Note: The actual merge logic is deferred to the Profile Loader Updates spec.
// This type only defines the structure and documents the intended merge semantics.
type ContentConfig struct {
	// Root specifies the content root directory within the profile.
	// This is the subdirectory containing installable content files.
	// When empty, defaults to DefaultContentRoot ("content") during loading.
	// Example values: "content", "templates", "configs"
	Root string `yaml:"root,omitempty"`

	// Targets maps content subdirectory names to installation target paths.
	// Each key is a subdirectory name within the content root, and each value
	// is the path (relative to project root) where that subdirectory's content
	// should be installed.
	//
	// Target values support Go template syntax with variable substitution:
	//
	//   Template Syntax:
	//     - Use {{ .Variables.key }} to reference custom variables
	//     - All Sprig functions are available (100+ template functions)
	//     - Templates are evaluated once during router construction
	//
	//   Variable Sources (in order of precedence, lowest to highest):
	//     1. Global variables from ~/.weftlo.yaml
	//     2. Profile variables from profile.yaml
	//     3. Project variables from .weftlo.yaml
	//
	//   Examples:
	//     # Static path (no template)
	//     "claude": ".claude"
	//
	//     # Dynamic path using namespace variable
	//     "claude": ".{{ .Variables.namespace }}/claude"
	//
	//     # Using Sprig functions
	//     "skills": ".{{ .Variables.name | lower | kebabcase }}/skills"
	//
	//     # Using environment variables
	//     "config": ".{{ env \"USER\" }}/config"
	//
	//     # Conditional paths
	//     "logs": ".{{ ternary \"prod\" \"dev\" .Variables.isProd }}/logs"
	//
	//   Path Validation:
	//     - Evaluated paths must be relative (no leading /)
	//     - Path traversal (..) is forbidden
	//     - Missing variables cause immediate error (fail-fast)
	//     - Variables may contain path separators (e.g., "org/team")
	//
	// Example: {"skills": ".claude/skills", "standards": "weftlo/standards"}
	// When nil or empty, all content uses the DefaultTarget path.
	Targets map[string]string `yaml:"targets,omitempty"`

	// DefaultTarget specifies the fallback installation target path for content
	// subdirectories that are not explicitly mapped in Targets.
	// When empty, defaults to DefaultTargetPath ("weftlo") during loading.
	// Example values: "weftlo", ".claude", "configs"
	//
	// Note: DefaultTarget does not support template syntax. Use target_overrides
	// in project configuration to customize paths for unmapped content.
	DefaultTarget string `yaml:"default_target,omitempty"`
}
