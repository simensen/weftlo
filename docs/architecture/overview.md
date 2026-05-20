# Weftlo Architecture

This document describes the technical architecture, design patterns, and system components of Weftlo.

## Table of Contents

1. [System Overview](#system-overview)
2. [Layered Architecture](#layered-architecture)
3. [Domain Models](#domain-models)
4. [Configuration System](#configuration-system)
5. [Profile System](#profile-system)
6. [Template System](#template-system)
7. [Routing System](#routing-system)
8. [Manifest System](#manifest-system)
9. [Change Detection](#change-detection)
10. [Installation Workflow](#installation-workflow)

## Related Documents

- [ADR-0001: Afero Filesystem Abstraction](../decisions/0001-afero-filesystem-abstraction.md)
- [ADR-0002: Clean Architecture Layers](../decisions/0002-clean-architecture-layers.md)
- [ADR-0003: XDG Base Directory Support](../decisions/0003-xdg-base-directory-support.md)
- [ADR-0004: Manifest-Based Change Detection](../decisions/0004-manifest-based-change-detection.md)
- [ADR-0005: Deep Merge for Variables](../decisions/0005-deep-merge-for-variables.md)
- [Profile System Specification](../specs/profiles.md) — Detailed profile behavior

## System Overview

Weftlo follows a clean architecture pattern with four distinct layers:

```
┌─────────────────────────────────────────────────────────────────┐
│                         CLI Layer                               │
│  Handles user interaction, command parsing, output formatting   │
└──────────────────────────────┬──────────────────────────────────┘
                               │
┌──────────────────────────────▼──────────────────────────────────┐
│                     Application Layer                           │
│  Orchestrates business workflows, coordinates domain services   │
└──────────────────────────────┬──────────────────────────────────┘
                               │
┌──────────────────────────────▼──────────────────────────────────┐
│                       Domain Layer                              │
│  Core business logic, domain models, validation rules           │
└──────────────────────────────┬──────────────────────────────────┘
                               │
┌──────────────────────────────▼──────────────────────────────────┐
│                   Infrastructure Layer                          │
│  External concerns: filesystem, config loading, persistence     │
└─────────────────────────────────────────────────────────────────┘
```

## Layered Architecture

### CLI Layer (`internal/cli/`)

The CLI layer uses Cobra for command-line interface implementation. Key files:

| File | Purpose |
|------|---------|
| `root.go` | Root command, global flags, command registration |
| `init.go` | Initialize global configuration directory |
| `install.go` | Install profiles to projects |
| `update.go` | Update installed files from profile changes |
| `status.go` | Display installation status |
| `profile_create.go` | Create new profiles |
| `profile_list.go` | List available profiles |
| `errors.go` | Custom error types with user-friendly messages |

**Command Wrapper Pattern:**

The CLI uses a wrapper pattern for deferred stdout resolution and global flag forwarding:

```go
type installCommandWrapper struct {
    fs       afero.Fs
    homeDir  HomeDirFunc
    rootCmd  *cobra.Command
    resolver ConfigDirResolver
    verbose  *bool  // Pointer to root's persistent flag
    quiet    *bool  // Pointer to root's persistent flag
}

func (w *installCommandWrapper) run(cmd *cobra.Command, args []string) error {
    // Create actual command at execution time
    installCmd := NewInstallCommandWithResolver(...)
    // Forward flags from root command
    return installCmd.RunE(installCmd, args)
}
```

### Application Layer (`internal/app/`)

Orchestrates domain services to implement business workflows.

#### Rendering Pipeline (`internal/app/rendering/`)

```go
// RenderingPipeline orchestrates template rendering across profiles
type RenderingPipeline struct {
    engine *template.TemplateEngine
}

// RenderAllWithOptions renders all templates from a merged profile
func (p *RenderingPipeline) RenderAllWithOptions(
    mergedProfile *infraprofile.MergedProfile,
    projectDir string,
    opts *RenderOptions,
) ([]RenderResult, error)
```

Key concepts:
- Renders templates from merged profile
- Passes router context for `reference()` function support
- Supports verbose mode for warnings
- Returns `RenderResult` with source path, target path, and content

#### Routing System (`internal/app/routing/`)

```go
// Router determines target paths for files
type Router interface {
    Route(sourcePath string) (RoutedFile, error)
}

// RoutedFile contains routing results
type RoutedFile struct {
    TargetPath     string  // Final installation path
    TargetCategory string  // Category name for grouping
    Suppressed     bool    // True if file should be skipped
}
```

The `ContentRouter` implementation:
1. Strips `.tmpl` suffix from source paths
2. Matches against defined targets using path prefixes
3. Applies target overrides (redirect or suppress)
4. Evaluates template expressions in override values

#### Install Service (`internal/app/install/`)

```go
// Service orchestrates installation workflow
type Service struct {
    fs                afero.Fs
    pipeline          *rendering.RenderingPipeline
    manifestGenerator *manifest.ManifestGenerator
}

// Install executes the installation workflow
func (s *Service) Install(
    mergedProfile *infraprofile.MergedProfile,
    projectDir string,
    router routing.Router,
    opts InstallOptions,
) (*InstallResult, error)
```

Installation workflow:
1. Render all templates using RenderingPipeline
2. Route files using Router
3. Filter suppressed files
4. Write files to disk (or dry-run)
5. Generate manifest

### Domain Layer (`internal/domain/`)

Contains core business logic independent of external concerns.

#### Profile Domain (`internal/domain/profile/`)

```go
// Profile represents a single profile configuration
type Profile struct {
    Name          string
    ProfileConfig ProfileConfig
    ContentConfig ContentConfig
    ContentRoot   string  // Absolute path to content directory
}

// ProfileConfig from profile.yaml
type ProfileConfig struct {
    Name     string                 `yaml:"name"`
    Extends  string                 `yaml:"extends,omitempty"`
    Variables map[string]interface{} `yaml:"variables,omitempty"`
}

// ContentConfig from content.yaml
type ContentConfig struct {
    DefaultTarget string            `yaml:"default_target"`
    Targets       map[string]string `yaml:"targets,omitempty"`
    Ignore        []string          `yaml:"ignore,omitempty"`
}
```

#### Template Domain (`internal/domain/template/`)

```go
// TemplateEngine renders Go templates with custom functions
type TemplateEngine struct {
    fs afero.Fs
}

// Custom template functions:
// - include(path, context): Include another template
// - reference(path): Reference another file's rendered content
// - referenceGlob(pattern): Reference multiple files by glob pattern
```

#### Manifest Domain (`internal/domain/manifest/`)

```go
// Manifest tracks installed files for change detection
type Manifest struct {
    Version     int               `json:"version"`
    GeneratedAt time.Time         `json:"generated_at"`
    Profiles    []string          `json:"profiles"`
    Files       []ManifestFile    `json:"files"`
}

// ManifestFile tracks individual file state
type ManifestFile struct {
    Path           string `json:"path"`
    SourceChecksum string `json:"source_checksum"`
    OutputChecksum string `json:"output_checksum"`
    SourceProfile  string `json:"source_profile"`
    TargetCategory string `json:"target_category,omitempty"`
}
```

### Infrastructure Layer (`internal/infrastructure/`)

Handles external concerns: filesystem, config loading, profile resolution.

#### Config Loading (`internal/infrastructure/config/`)

```go
// GlobalConfigLoader loads ~/.config/weftlo/config.yaml
type GlobalConfigLoader struct {
    fs afero.Fs
}

// ProjectConfigLoader loads .weftlo.yaml from project
type ProjectConfigLoader struct {
    fs afero.Fs
}

// Resolver handles XDG Base Directory resolution
type Resolver struct{}
```

#### Profile Loading (`internal/infrastructure/profile/`)

```go
// ProfileLoader loads and merges profiles
type ProfileLoader struct {
    fs        afero.Fs
    configDir string
}

// LoadMergedMultiple loads and merges multiple profiles
func (l *ProfileLoader) LoadMergedMultiple(profileNames []string) (*MergedProfile, error)

// MergedProfile combines multiple profiles with inheritance
type MergedProfile struct {
    profiles        []*profile.Profile
    contentConfig   profile.ContentConfig
    variables       map[string]interface{}
    templateFiles   []profile.TemplateFile
    matcher         ignore.Matcher
    conflictTracker *profile.VariableConflictTracker
}
```

## Domain Models

### Profile Model

A profile consists of:

```
<config-dir>/profiles/<vendor>/<name>/
├── profile.yaml          # Profile metadata, inheritance, variables
├── content.yaml          # Optional: targets, ignore patterns
├── .weftlo.ignore        # Optional: ignore patterns file
└── content/              # Content root directory
    ├── file1.md
    ├── file2.md.tmpl     # Template file
    └── _partial.md       # Partial (not installed)
```

### Variable Merging

Variables are merged using deep merge semantics with conflict detection:

```go
// DeepMergeVariables merges two maps with deep merge semantics
func DeepMergeVariables(base, override map[string]interface{}) map[string]interface{}

// DeepMergeVariablesChain merges multiple sources with tracking
func DeepMergeVariablesChain(sources ...VariableSource) (map[string]interface{}, []VariableConflict)
```

Merge precedence (lowest to highest):
1. Global config variables
2. Profile inheritance chain (ancestor → child)
3. Project config variables

### Template File Model

```go
// TemplateFile represents a file in a profile's content directory
type TemplateFile struct {
    SourcePath string  // Path within content root
    IsPartial  bool    // True if filename starts with "_"
    IsTemplate bool    // True if filename ends with ".tmpl"
    Profile    string  // Source profile name
}
```

## Configuration System

### Global Configuration (`config.yaml`)

```yaml
# ~/.config/weftlo/config.yaml
default_profile: mycompany/standard
install_prefix: .ai

# Global variables (lowest precedence)
variables:
  company: MyCompany
  author: Development Team

# Universal target overrides
target_overrides:
  commands: ""              # Redirect to root
  deprecated: ~             # Suppress entirely

# Profile-specific overrides
profile_overrides:
  mycompany/backend:
    target_overrides:
      commands: backend/

# Global ignore patterns
ignore:
  - "*.bak"
  - ".DS_Store"
```

### Project Configuration (`.weftlo.yaml`)

```yaml
# .weftlo.yaml in project root
profiles:
  - mycompany/base
  - mycompany/backend

install_prefix: .claude

# Project variables (highest precedence)
variables:
  namespace: my-project
  environment: production

# Project target overrides
target_overrides:
  skills: .claude/skills/
  hooks: ~  # Suppress hooks for this project
```

### Profile Configuration (`profile.yaml`)

```yaml
# profiles/<vendor>/<name>/profile.yaml
name: mycompany/backend
extends: mycompany/base

variables:
  framework: express
  database: postgres
```

### Content Configuration (`content.yaml`)

```yaml
# profiles/<vendor>/<name>/content.yaml
default_target: .claude

targets:
  skills: .claude/skills/
  commands: .claude/commands/
  hooks: .claude/hooks/

ignore:
  - "*.test.md"
  - "_*"
```

## Profile System

### Profile Inheritance

Profiles can extend other profiles using single inheritance:

```yaml
# base profile
name: mycompany/base
variables:
  company: MyCompany

# child profile
name: mycompany/backend
extends: mycompany/base
variables:
  framework: express
```

**Inheritance Resolution:**

1. Load the requested profile
2. If `extends` is set, recursively load parent profile
3. Build inheritance chain (root → leaf)
4. Merge content from each profile in order

### Profile Merging

The `MergedProfile` struct combines profiles:

```go
type MergedProfile struct {
    profiles        []*profile.Profile  // Inheritance chain
    contentConfig   profile.ContentConfig
    variables       map[string]interface{}
    templateFiles   []profile.TemplateFile
    matcher         ignore.Matcher
}

// ProfileNames returns all profile names in inheritance order
func (m *MergedProfile) ProfileNames() []string

// LeafProfileName returns the most specific (last) profile
func (m *MergedProfile) LeafProfileName() string

// Resolve finds a file's absolute path by searching profiles
func (m *MergedProfile) Resolve(relativePath string) (string, bool)
```

### Template Files Discovery

Template files are discovered by walking the content directory:

```go
// EnumerateTemplateFiles walks content directory
func EnumerateTemplateFiles(contentRoot, profileName string) ([]TemplateFile, error)
```

Rules:
- Files starting with `_` are partials (not installed)
- Files ending with `.tmpl` are templates (rendered)
- Other files are copied verbatim
- Hidden files (starting with `.`) are skipped

## Template System

### Template Engine

```go
// NewTemplateEngineWithFs creates engine with custom filesystem
func NewTemplateEngineWithFs(fs afero.Fs) *TemplateEngine

// Render processes a template with context
func (e *TemplateEngine) Render(templateContent string, context interface{}) (string, error)

// RenderFile loads and renders a template file
func (e *TemplateEngine) RenderFile(path string, context interface{}) (string, error)
```

### Template Context

Templates receive a context with:

```go
type TemplateContext struct {
    Variables map[string]interface{}  // Merged variables
    Env       map[string]string       // Environment variables
    Profile   string                  // Current profile name
}
```

### Built-in Functions

#### `include(path, context)`

Include another template file, passing the current context:

```
{{ include "_header.md" . }}

Main content here.

{{ include "_footer.md" . }}
```

The `@` prefix references the profile's content root:

```
{{ include "@partials/_common.md" . }}
```

#### `reference(path)`

Reference another file's rendered content (for cross-file dependencies):

```
## Related Files

{{ reference "commands/build.md" }}
```

#### `referenceGlob(pattern)`

Reference multiple files matching a glob pattern:

```
## All Skills

{{ referenceGlob "skills/*.md" }}
```

### Template Variables

Variables are accessible via `.Variables`:

```
# {{ .Variables.project_name }}

Author: {{ .Variables.author }}
Version: {{ .Variables.version | default "1.0.0" }}
```

## Routing System

### Target Categories

Files are routed to named target directories:

```yaml
# content.yaml
default_target: .claude

targets:
  skills: .claude/skills/
  commands: .claude/commands/
```

A file `skills/coding.md` would be installed to `.claude/skills/coding.md`.

### Target Override Precedence

Four levels of precedence (lowest to highest):

1. **Profile targets**: From `content.yaml`
2. **Global universal overrides**: From `config.yaml target_overrides`
3. **Global profile-specific**: From `config.yaml profile_overrides.<name>.target_overrides`
4. **Project overrides**: From `.weftlo.yaml target_overrides`

### Override Semantics

```yaml
target_overrides:
  # Redirect to different path
  commands: backend/commands/

  # Redirect to project root
  config: ""

  # Suppress (don't install)
  deprecated: ~
```

### Template Evaluation in Overrides

Override values can contain template expressions:

```yaml
target_overrides:
  skills: "{{ .Variables.namespace }}/skills/"
```

### Router Implementation

```go
// ContentRouter implements Router interface
type ContentRouter struct {
    contentConfig ContentConfig
    overrides     map[string]*string  // nil = suppress
    variables     map[string]interface{}
}

func (r *ContentRouter) Route(sourcePath string) (RoutedFile, error) {
    // 1. Strip .tmpl suffix
    // 2. Find matching target category
    // 3. Apply overrides if present
    // 4. Return RoutedFile with target path
}
```

## Manifest System

### Manifest Structure

```json
{
  "version": 1,
  "generated_at": "2024-01-15T10:30:00Z",
  "profiles": ["mycompany/base", "mycompany/backend"],
  "files": [
    {
      "path": ".claude/skills/coding.md",
      "source_checksum": "sha256:abc123...",
      "output_checksum": "sha256:def456...",
      "source_profile": "mycompany/backend",
      "target_category": "skills"
    }
  ]
}
```

### Checksum Types

- **Source checksum**: Hash of original template content (pre-rendering)
- **Output checksum**: Hash of rendered content (post-rendering)

### Manifest Generation

```go
// ManifestGenerator creates manifests from rendered files
type ManifestGenerator struct{}

func (g *ManifestGenerator) Generate(
    profileNames []string,
    renderedFiles []RenderedFile,
) (*Manifest, error)
```

## Change Detection

### File Status Types

```go
type FileStatus string

const (
    StatusUnchanged     FileStatus = "unchanged"      // No changes
    StatusSourceChanged FileStatus = "source_changed" // Template modified
    StatusUserModified  FileStatus = "user_modified"  // User edited output
    StatusConflict      FileStatus = "conflict"       // Both changed
    StatusNew           FileStatus = "new"            // New file in profile
    StatusRemoved       FileStatus = "removed"        // Removed from profile
)
```

### Detection Algorithm

```go
func DetectChanges(
    manifest *Manifest,
    mergedProfile *MergedProfile,
    fs afero.Fs,
    projectDir string,
    router Router,
) (map[string]FileChangeInfo, error)
```

For each file:
1. If in manifest but not in profile → `StatusRemoved`
2. If in profile but not in manifest → `StatusNew`
3. If both:
   - Compare source checksums (template changed?)
   - Compare output checksums (user modified?)
   - Determine appropriate status

### Update Planning

```go
type UpdatePlan struct {
    ToCreate []FileDecision  // New files
    ToUpdate []FileDecision  // Changed templates
    ToSkip   []FileDecision  // User modified (unless --force)
    ToDelete []FileDecision  // Removed (if --remove-orphans)
}

func PlanUpdate(
    changeResults map[string]FileStatus,
    mergedProfile *MergedProfile,
    options UpdateOptions,
    engine *TemplateEngine,
    projectDir string,
) (UpdatePlan, error)
```

## Installation Workflow

### Install Command Workflow

```
1. Resolve config directory (XDG)
2. Load global config
3. Load project config (if exists)
4. Resolve profile names
5. Load and merge profiles
6. Build Router with merged overrides
7. Check for file conflicts (unless --force)
8. Render templates
9. Route files
10. Write files (or dry-run)
11. Generate manifest
12. Create .weftlo.yaml (if new)
13. Display success output
```

### Update Command Workflow

```
1. Verify installation exists (.weftlo.yaml, .weftlo.manifest.json)
2. Load configs and profiles
3. Build Router
4. Detect changes (compare with manifest)
5. Plan update (categorize files)
6. If no changes, exit early
7. If dry-run, display summary and exit
8. Execute file operations (create/update/delete)
9. Regenerate manifest
10. Display success output
```

### Dry-Run Mode

Both `install` and `update` support `--dry-run`:
- No filesystem writes occur
- Operations are formatted and displayed
- Returns successfully without changes

## Design Decisions

### Afero Filesystem Abstraction

All filesystem operations use the `afero.Fs` interface:
- Production: `afero.OsFs` (real filesystem)
- Tests: `afero.MemMapFs` (in-memory)

This enables comprehensive testing without touching the real filesystem.

### XDG Base Directory Support

Configuration directory resolution follows XDG conventions:
1. `$XDG_CONFIG_HOME/weftlo/` if set
2. `~/.config/weftlo/` if `~/.config/` exists
3. `~/.weftlo/` as fallback

### Dependency Injection

All commands receive dependencies through constructors:
- Filesystem (`afero.Fs`)
- Home directory resolver (`HomeDirFunc`)
- Config directory resolver (`ConfigDirResolver`)
- I/O streams (`stdin`, `stdout`)

This enables testing without mocking or monkey patching.

### Error Types

Custom error types provide user-friendly messages:

```go
type HomeDirError struct{ Err error }
type PermissionError struct{ Path string; Err error }
type ProfileNotFoundError struct{ ProfileName string }
type InstallationNotFoundError struct{ MissingFile string }
type FileConflictError struct{ ConflictingFiles []string }
```

Each implements `Error()` with helpful context and suggestions.

---

<!-- Migrated from weftlo-current-implementation/architecture.md -->
