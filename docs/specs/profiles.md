# Profile System Specification

This specification describes how to create, configure, and extend custom profiles in Weftlo.

## Table of Contents

1. [Profile Basics](#profile-basics)
2. [Creating a Profile](#creating-a-profile)
3. [Profile Structure](#profile-structure)
4. [Profile Configuration](#profile-configuration)
5. [Content Configuration](#content-configuration)
6. [Profile Inheritance](#profile-inheritance)
7. [Variables](#variables)
8. [Templates](#templates)
9. [Ignore Patterns](#ignore-patterns)
10. [Target Routing](#target-routing)
11. [Best Practices](#best-practices)
12. [Examples](#examples)

## Related Documents

- [Architecture Overview](../architecture/overview.md) — System architecture and component relationships
- [CLI Reference](../reference/cli.md) — Command-line interface for profile operations
- [Configuration Reference](../reference/configuration.md) — Configuration file formats
- [Template Reference](../reference/templates.md) — Template syntax used in profiles

## Profile Basics

A profile is a collection of configuration files that can be installed into projects. Profiles support:

- **Template rendering**: Use Go templates for dynamic content
- **Inheritance**: Build specialized profiles from base profiles
- **Variables**: Customize content with variable substitution
- **Target routing**: Control where files are installed
- **Ignore patterns**: Exclude files from installation

### Profile Naming

Profiles use a `vendor/name` format:
- **vendor**: Organization or user namespace (e.g., `mycompany`, `personal`)
- **name**: Profile name (e.g., `backend`, `frontend`, `dotfiles`)

Valid characters: alphanumeric, underscores (`_`), hyphens (`-`)

Examples:
- `mycompany/standard`
- `team-a/api-service`
- `personal/dev_tools`

## Creating a Profile

### Using the CLI

```bash
weftlo profile create mycompany/backend
```

This creates:
```
~/.config/weftlo/profiles/mycompany/backend/
├── profile.yaml
├── content/
└── README.md
```

### Manual Creation

1. Create the profile directory:
   ```bash
   mkdir -p ~/.config/weftlo/profiles/mycompany/backend/content
   ```

2. Create `profile.yaml`:
   ```yaml
   name: mycompany/backend
   ```

3. Add content files to `content/`

## Profile Structure

```
profiles/<vendor>/<name>/
├── profile.yaml          # Required: Profile metadata
├── content.yaml          # Optional: Routing and ignore config
├── .weftlo.ignore        # Optional: Ignore patterns file
├── content/              # Required: Content root directory
│   ├── file.md           # Static file
│   ├── template.md.tmpl  # Template file
│   ├── _partial.md       # Partial (not installed)
│   └── subdir/
│       └── nested.md
└── README.md             # Optional: Documentation
```

### File Types

| Pattern | Type | Behavior |
|---------|------|----------|
| `*.md` | Static | Copied verbatim |
| `*.md.tmpl` | Template | Rendered, `.tmpl` suffix removed |
| `_*.md` | Partial | Not installed, available for `include()` |
| `.*` | Hidden | Ignored during discovery |

## Profile Configuration

### profile.yaml

```yaml
# Required: Profile name (must match directory path)
name: mycompany/backend

# Optional: Parent profile for inheritance
extends: mycompany/base

# Optional: Variables for templates
variables:
  company: MyCompany
  framework: express
  database: postgres
  features:
    logging: true
    metrics: false
```

### Fields

| Field | Type | Description |
|-------|------|-------------|
| `name` | string | Profile identifier in `vendor/name` format |
| `extends` | string | Parent profile for inheritance |
| `variables` | map | Key-value pairs for template substitution |

## Content Configuration

### content.yaml

```yaml
# Default target directory for files without explicit routing
default_target: .claude

# Named target directories
targets:
  skills: .claude/skills/
  commands: .claude/commands/
  hooks: .claude/hooks/
  config: ""  # Empty string = project root

# Ignore patterns (gitignore-style)
ignore:
  - "*.test.md"
  - "drafts/"
  - "!important.md"  # Negation
```

### Fields

| Field | Type | Description |
|-------|------|-------------|
| `default_target` | string | Fallback target for unmatched files |
| `targets` | map | Named target directories |
| `ignore` | list | Patterns for files to skip |

### Target Matching

Files are matched to targets by path prefix:

```yaml
targets:
  skills: .claude/skills/
  commands: .claude/commands/
```

- `content/skills/coding.md` → `.claude/skills/coding.md`
- `content/commands/build.md` → `.claude/commands/build.md`
- `content/other.md` → `.claude/other.md` (default target)

## Profile Inheritance

### Single Inheritance

Profiles can extend one parent profile:

```yaml
# base profile: mycompany/base
name: mycompany/base
variables:
  company: MyCompany
  author: Development Team
```

```yaml
# child profile: mycompany/backend
name: mycompany/backend
extends: mycompany/base
variables:
  framework: express  # Added
  author: Backend Team  # Overrides parent
```

### Inheritance Chain

When loading `mycompany/backend`:
1. Load `mycompany/base` (parent)
2. Load `mycompany/backend` (child)
3. Merge in order: base → backend

### What Gets Inherited

| Component | Inheritance Behavior |
|-----------|---------------------|
| Variables | Deep merged (child overrides parent) |
| Content files | Child files override parent files at same path |
| Targets | Child targets override parent targets |
| Ignore patterns | Combined (both apply) |

### Multi-Level Inheritance

```yaml
# Level 1: company/base
name: company/base
variables:
  company: ACME Corp

# Level 2: company/backend
name: company/backend
extends: company/base
variables:
  tier: backend

# Level 3: company/backend-api
name: company/backend-api
extends: company/backend
variables:
  service_type: api
```

## Variables

### Defining Variables

**In profile.yaml:**
```yaml
variables:
  project_name: my-project
  author: Jane Doe
  version: 1.0.0

  # Nested variables
  database:
    host: localhost
    port: 5432
    name: mydb

  # Lists
  features:
    - authentication
    - logging
    - metrics
```

**In global config.yaml:**
```yaml
variables:
  company: MyCompany
  default_author: Team
```

**In project .weftlo.yaml:**
```yaml
variables:
  environment: production
  namespace: my-namespace
```

### Variable Precedence

Lowest to highest priority:
1. Global config (`config.yaml`)
2. Profile inheritance chain (ancestor → child)
3. Project config (`.weftlo.yaml`)

### Deep Merge Semantics

Nested maps are merged recursively:

```yaml
# Parent profile
variables:
  database:
    host: localhost
    port: 5432

# Child profile
variables:
  database:
    port: 3306  # Override
    name: mydb  # Add

# Result
variables:
  database:
    host: localhost  # From parent
    port: 3306       # Overridden
    name: mydb       # Added
```

### Variable Conflict Warnings

When variables conflict, Weftlo warns during installation:

```
Warning: Variable 'author' has conflicting values across profiles:
  - "Development Team" (from global config)
  - "Backend Team" (from mycompany/backend)
Using value from: mycompany/backend
```

### Using Variables in Templates

```markdown
# {{ .Variables.project_name }}

Author: {{ .Variables.author }}
Version: {{ .Variables.version }}

## Database Configuration

Host: {{ .Variables.database.host }}
Port: {{ .Variables.database.port }}
Database: {{ .Variables.database.name }}

## Features

{{- range .Variables.features }}
- {{ . }}
{{- end }}
```

## Templates

### Template Syntax

Templates use Go's `text/template` syntax:

```
{{ .Variables.name }}              # Variable access
{{ if .Variables.enabled }}...{{ end }}  # Conditionals
{{ range .Variables.items }}...{{ end }} # Iteration
{{ .Variables.name | default "value" }}  # Default values
```

### Template Files

Files ending with `.tmpl` are processed as templates:

```
content/
├── README.md           # Static file
├── config.md.tmpl      # Template → config.md
└── skills/
    └── coding.md.tmpl  # Template → skills/coding.md
```

### Template Context

Templates receive a context object:

```go
type TemplateContext struct {
    Variables map[string]interface{}  // Merged variables
    Env       map[string]string       // Environment variables
    Profile   string                  // Current profile name
}
```

### Built-in Functions

#### include(path, context)

Include another template file:

```
{{ include "_header.md" . }}

Main content here.

{{ include "_footer.md" . }}
```

The `@` prefix references profile content root:
```
{{ include "@partials/_common.md" . }}
```

#### reference(path)

Reference another file's rendered content:

```
## Related Commands

{{ reference "commands/build.md" }}
```

#### referenceGlob(pattern)

Reference multiple files by glob pattern:

```
## All Skills

{{ referenceGlob "skills/*.md" }}
```

#### default(value)

Provide default values:

```
Version: {{ .Variables.version | default "1.0.0" }}
```

### Partials

Files starting with `_` are partials:
- Not installed to the project
- Available for `include()` in other templates

```
content/
├── _header.md         # Partial
├── _footer.md         # Partial
├── _shared/
│   └── _common.md     # Partial
└── README.md.tmpl     # Uses partials via include()
```

## Ignore Patterns

### Pattern Syntax

Uses gitignore-style patterns:

```yaml
# content.yaml
ignore:
  # Glob patterns
  - "*.test.md"
  - "*.bak"

  # Directories
  - "drafts/"
  - "examples/"

  # Specific files
  - "TODO.md"
  - "NOTES.md"

  # Negation (include despite earlier rules)
  - "!important.md"

  # Double star (any depth)
  - "**/temp/**"
```

### .weftlo.ignore File

Alternative to `content.yaml` ignore section:

```
# .weftlo.ignore
*.test.md
*.bak
drafts/
!important.md
```

### Ignore Precedence

Patterns are combined from:
1. Profile ignore patterns (inherited)
2. Project `.weftlo.ignore` file
3. Project `.weftlo.yaml` ignore patterns

Later patterns can negate earlier patterns using `!`.

## Target Routing

### Defining Targets

```yaml
# content.yaml
default_target: .claude

targets:
  skills: .claude/skills/
  commands: .claude/commands/
  hooks: .claude/hooks/
```

### Path Matching

Files are matched by path prefix to targets:

| Source Path | Target | Installed Path |
|-------------|--------|----------------|
| `skills/coding.md` | `skills` | `.claude/skills/coding.md` |
| `commands/build.md` | `commands` | `.claude/commands/build.md` |
| `README.md` | (default) | `.claude/README.md` |

### Target Overrides

Override targets at project level:

```yaml
# .weftlo.yaml
target_overrides:
  # Redirect to different path
  commands: custom/commands/

  # Redirect to project root
  config: ""

  # Suppress (don't install)
  hooks: ~
```

### Template Expressions in Overrides

```yaml
# .weftlo.yaml
variables:
  namespace: my-project

target_overrides:
  skills: "{{ .Variables.namespace }}/skills/"
```

## Best Practices

### 1. Use Meaningful Profile Names

```yaml
# Good
name: mycompany/backend-api
name: team-a/web-frontend

# Avoid
name: my/profile
name: test/test
```

### 2. Document Your Profiles

Include a README.md in each profile:

```markdown
# MyCompany Backend Profile

This profile provides configuration for backend services.

## Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `framework` | Backend framework | `express` |
| `database` | Database type | `postgres` |

## Usage

```bash
weftlo install --profile mycompany/backend
```
```

### 3. Use Inheritance for Common Patterns

```yaml
# Base profile with common settings
name: mycompany/base
variables:
  company: MyCompany
  author: Engineering Team

# Specialized profiles
name: mycompany/backend
extends: mycompany/base

name: mycompany/frontend
extends: mycompany/base
```

### 4. Keep Variables Organized

```yaml
variables:
  # Project metadata
  project_name: my-service
  version: 1.0.0

  # Team information
  team:
    name: Platform Team
    contact: platform@example.com

  # Feature flags
  features:
    logging: true
    metrics: true
    tracing: false
```

### 5. Use Partials for Reusable Content

```
content/
├── _partials/
│   ├── _header.md
│   ├── _footer.md
│   └── _code-style.md
├── skills/
│   └── coding.md.tmpl  # {{ include "@_partials/_code-style.md" . }}
└── commands/
    └── review.md.tmpl  # {{ include "@_partials/_code-style.md" . }}
```

### 6. Test with Dry Run

```bash
# Preview installation before writing files
weftlo install --dry-run --profile mycompany/backend
```

## Examples

### Example 1: Claude Code Profile

Profile for configuring Claude Code with skills and commands.

**profile.yaml:**
```yaml
name: team/claude-code
variables:
  project_type: backend
  language: typescript
  test_framework: jest
```

**content.yaml:**
```yaml
default_target: .claude

targets:
  skills: .claude/skills/
  commands: .claude/commands/
  hooks: .claude/hooks/
```

**content/skills/coding.md.tmpl:**
```markdown
# Coding Skill

Language: {{ .Variables.language }}
Project Type: {{ .Variables.project_type }}

## Guidelines

Write clean, maintainable {{ .Variables.language }} code following
team conventions.

## Testing

Use {{ .Variables.test_framework }} for all tests.
```

### Example 2: Multi-Environment Profile

**profile.yaml:**
```yaml
name: mycompany/api
extends: mycompany/base
variables:
  service_name: api-gateway

  environments:
    development:
      log_level: debug
      replicas: 1
    production:
      log_level: info
      replicas: 3
```

**content/config.md.tmpl:**
```markdown
# {{ .Variables.service_name }} Configuration

## Development
- Log Level: {{ .Variables.environments.development.log_level }}
- Replicas: {{ .Variables.environments.development.replicas }}

## Production
- Log Level: {{ .Variables.environments.production.log_level }}
- Replicas: {{ .Variables.environments.production.replicas }}
```

### Example 3: Profile with Ignore Patterns

**content.yaml:**
```yaml
default_target: docs

ignore:
  - "*.draft.md"
  - "wip/"
  - "internal/"
  - "!internal/public.md"  # Include this specific file
```

### Example 4: Profile Inheritance Chain

```yaml
# Level 1: company/base
name: company/base
variables:
  company: ACME Corp
  coding_style: standard

# Level 2: company/backend
name: company/backend
extends: company/base
variables:
  tier: backend
  languages:
    - go
    - python

# Level 3: company/backend-api
name: company/backend-api
extends: company/backend
variables:
  service_type: api
  frameworks:
    - gin
    - fastapi
```

**Usage:**
```bash
# Install the most specific profile
weftlo install --profile company/backend-api

# Inherits: company/base → company/backend → company/backend-api
```

### Installing Multiple Profiles

You can install multiple profiles at once, which are merged in order:

```bash
# Install multiple profiles (merged left to right)
weftlo install --profile company/base --profile company/backend-api

# Later, add another profile
weftlo update --add-profile company/monitoring

# Or switch profiles
weftlo update --remove-profile company/backend-api --add-profile company/frontend
```

**Note:** The `weftlo install` command is for initial setup only. To modify profiles after installation, use `weftlo update` with `--add-profile` or `--remove-profile` flags.

---

<!-- Migrated from weftlo-current-implementation/profile-guide.md -->
