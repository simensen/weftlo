# Configuration Reference

Complete reference for all Weftlo configuration files and their options.

## Table of Contents

1. [Configuration Files Overview](#configuration-files-overview)
2. [Global Configuration](#global-configuration)
3. [Project Configuration](#project-configuration)
4. [Profile Configuration](#profile-configuration)
5. [Content Configuration](#content-configuration)
6. [Ignore Patterns](#ignore-patterns)
7. [Manifest File](#manifest-file)
8. [Configuration Precedence](#configuration-precedence)

## Configuration Files Overview

| File | Location | Purpose |
|------|----------|---------|
| `config.yaml` | `~/.config/weftlo/` | Global configuration |
| `.weftlo.yaml` | Project root | Project-specific configuration |
| `profile.yaml` | Profile directory | Profile metadata |
| `content.yaml` | Profile directory | Content routing configuration |
| `.weftlo.ignore` | Profile or project | Ignore patterns |
| `.weftlo.manifest.json` | Project root | Installation tracking |

## Global Configuration

Location: `~/.config/weftlo/config.yaml` (or `~/.weftlo/config.yaml`)

### Full Schema

```yaml
# Default profile when none specified
default_profile: vendor/name

# Default installation directory prefix
install_prefix: weftlo

# Global variables (lowest precedence)
variables:
  key: value
  nested:
    key: value

# Universal target overrides (apply to all profiles)
target_overrides:
  target_name: override_path
  suppressed_target: ~

# Profile-specific configuration
profile_overrides:
  vendor/name:
    target_overrides:
      target_name: override_path

# Global ignore patterns
ignore:
  - "*.pattern"
```

### Fields

#### default_profile

**Type:** `string`
**Required:** No
**Default:** `default/default`

The profile to use when no `--profile` flag is provided and no project config exists.

```yaml
default_profile: mycompany/standard
```

#### install_prefix

**Type:** `string`
**Required:** No
**Default:** `weftlo`

Default target directory for installed files.

```yaml
install_prefix: .claude
```

#### variables

**Type:** `map[string]interface{}`
**Required:** No
**Default:** `{}`

Global variables available to all templates. These have the lowest precedence and can be overridden by profile or project variables.

```yaml
variables:
  company: MyCompany
  author: Development Team
  settings:
    debug: false
    log_level: info
```

#### target_overrides

**Type:** `map[string]*string`
**Required:** No
**Default:** `{}`

Universal target overrides applied to all profiles. Values can be:
- A path string (redirect target)
- Empty string `""` (redirect to project root)
- Null `~` (suppress target)

```yaml
target_overrides:
  # Redirect to different path
  commands: global/commands/

  # Redirect to project root
  config: ""

  # Suppress entirely
  deprecated: ~
```

#### profile_overrides

**Type:** `map[string]ProfileOverride`
**Required:** No
**Default:** `{}`

Profile-specific overrides. Each key is a profile name.

```yaml
profile_overrides:
  mycompany/backend:
    target_overrides:
      commands: backend/commands/

  mycompany/frontend:
    target_overrides:
      commands: frontend/commands/
```

#### ignore

**Type:** `[]string`
**Required:** No
**Default:** `[]`

Global ignore patterns applied to all installations.

```yaml
ignore:
  - "*.bak"
  - ".DS_Store"
  - "Thumbs.db"
```

### Example

```yaml
# ~/.config/weftlo/config.yaml
default_profile: mycompany/standard
install_prefix: .ai

variables:
  company: MyCompany Inc.
  organization_id: org-12345
  defaults:
    language: typescript
    framework: react

target_overrides:
  legacy: ~

profile_overrides:
  mycompany/backend:
    target_overrides:
      api: backend/api/

ignore:
  - "*.bak"
  - ".DS_Store"
```

## Project Configuration

Location: `.weftlo.yaml` in project root

### Full Schema

```yaml
# Profiles to install (in merge order)
profiles:
  - vendor/base
  - vendor/specialized

# Installation directory prefix
install_prefix: directory

# Project-specific variables (highest precedence)
variables:
  key: value

# Project-specific target overrides
target_overrides:
  target_name: override_path
```

### Fields

#### profiles

**Type:** `[]string`
**Required:** Yes
**Default:** None

Profiles to install, listed in merge order (first = base, last = most specific).

```yaml
profiles:
  - mycompany/base
  - mycompany/backend
```

For a single profile:
```yaml
profiles:
  - mycompany/standard
```

#### install_prefix

**Type:** `string`
**Required:** No
**Default:** From global config or `weftlo`

Override the installation directory for this project.

```yaml
install_prefix: .claude
```

#### variables

**Type:** `map[string]interface{}`
**Required:** No
**Default:** `{}`

Project-specific variables. These have the highest precedence and override global and profile variables.

```yaml
variables:
  project_name: my-awesome-project
  environment: production
  features:
    analytics: true
    experiments: false
```

#### target_overrides

**Type:** `map[string]*string`
**Required:** No
**Default:** `{}`

Project-specific target overrides. These have the highest precedence.

```yaml
target_overrides:
  # Custom path for this project
  skills: custom/skills/

  # Disable hooks for this project
  hooks: ~
```

### Example

```yaml
# .weftlo.yaml
profiles:
  - mycompany/base
  - mycompany/backend-api

install_prefix: .claude

variables:
  project_name: order-service
  namespace: orders
  environment: production

target_overrides:
  experimental: ~
```

## Profile Configuration

Location: `<profile-dir>/profile.yaml`

### Full Schema

```yaml
# Profile identifier
name: vendor/name

# Parent profile for inheritance
extends: vendor/parent

# Profile-specific variables
variables:
  key: value
```

### Fields

#### name

**Type:** `string`
**Required:** Yes
**Format:** `vendor/name`

Profile identifier. Must match the directory path.

```yaml
name: mycompany/backend
```

#### extends

**Type:** `string`
**Required:** No
**Default:** None

Parent profile for inheritance. The child profile inherits all content, variables, and configuration from the parent.

```yaml
extends: mycompany/base
```

#### variables

**Type:** `map[string]interface{}`
**Required:** No
**Default:** `{}`

Profile-specific variables. These override parent profile variables and are overridden by project variables.

```yaml
variables:
  framework: express
  database:
    type: postgres
    version: "15"
```

### Example

```yaml
# profiles/mycompany/backend/profile.yaml
name: mycompany/backend
extends: mycompany/base

variables:
  tier: backend
  framework: express
  database:
    type: postgres
    port: 5432
  features:
    - authentication
    - logging
    - metrics
```

## Content Configuration

Location: `<profile-dir>/content.yaml`

### Full Schema

```yaml
# Default target for unmatched files
default_target: directory

# Named target directories
targets:
  name: path/

# Ignore patterns
ignore:
  - "pattern"
```

### Fields

#### default_target

**Type:** `string`
**Required:** No
**Default:** From install prefix

The target directory for files that don't match any named target.

```yaml
default_target: weftlo
```

#### targets

**Type:** `map[string]string`
**Required:** No
**Default:** `{}`

Named target directories. Files are matched by path prefix.

```yaml
targets:
  skills: .claude/skills/
  commands: .claude/commands/
  hooks: .claude/hooks/
  config: ""  # Project root
```

**Target Matching:**

Files are matched to targets by their source path prefix:

| Source | Target Match | Result |
|--------|-------------|--------|
| `skills/coding.md` | `skills` | `.claude/skills/coding.md` |
| `commands/build.md` | `commands` | `.claude/commands/build.md` |
| `README.md` | (default) | `weftlo/README.md` |

#### ignore

**Type:** `[]string`
**Required:** No
**Default:** `[]`

Gitignore-style patterns for files to exclude.

```yaml
ignore:
  - "*.test.md"
  - "drafts/"
  - "internal/"
  - "!internal/public.md"
```

### Example

```yaml
# profiles/mycompany/backend/content.yaml
default_target: docs

targets:
  skills: .claude/skills/
  commands: .claude/commands/
  hooks: .claude/hooks/
  config: ""

ignore:
  - "*.draft.md"
  - "*.test.md"
  - "wip/"
  - "!wip/important.md"
```

## Ignore Patterns

### Pattern Syntax

Weftlo uses gitignore-style patterns:

| Pattern | Meaning |
|---------|---------|
| `*.md` | All `.md` files in any directory |
| `dir/` | Directory and all contents |
| `dir/*.md` | `.md` files directly in `dir/` |
| `**/temp/` | `temp/` directory at any depth |
| `!file.md` | Negate previous pattern (include file) |
| `#comment` | Comment line |

### Pattern Locations

1. **content.yaml ignore section:**
   ```yaml
   ignore:
     - "*.test.md"
   ```

2. **.weftlo.ignore file:**
   ```
   # Profile ignore patterns
   *.test.md
   drafts/
   ```

3. **Project .weftlo.ignore:**
   ```
   # Project-specific ignores
   local/
   ```

### Pattern Precedence

Patterns are combined in order:
1. Profile ignore patterns (inherited)
2. Profile `.weftlo.ignore` file
3. Profile `content.yaml` ignore section
4. Project `.weftlo.ignore` file

Later patterns can negate earlier ones with `!`.

### Examples

```
# Ignore test files
*.test.md
*.spec.md

# Ignore directories
drafts/
wip/
internal/

# Include specific files despite earlier rules
!internal/public.md
!wip/urgent.md

# Ignore at any depth
**/temp/
**/*.bak

# Comments
# This is a comment
```

## Manifest File

Location: `.weftlo.manifest.json` in project root

The manifest tracks installed files for change detection. This file is automatically generated and should not be manually edited.

### Schema

```json
{
  "version": 1,
  "generated_at": "2024-01-15T10:30:00Z",
  "profiles": ["vendor/base", "vendor/specialized"],
  "files": [
    {
      "path": ".claude/skills/coding.md",
      "source_checksum": "sha256:abc123...",
      "output_checksum": "sha256:def456...",
      "source_profile": "vendor/specialized",
      "target_category": "skills"
    }
  ]
}
```

### Fields

#### version

Schema version for forward compatibility.

#### generated_at

ISO 8601 timestamp of when the manifest was generated.

#### profiles

List of profile names that were installed, in inheritance order.

#### files

Array of file entries, each containing:

| Field | Description |
|-------|-------------|
| `path` | Installed file path relative to project |
| `source_checksum` | SHA-256 hash of source template content |
| `output_checksum` | SHA-256 hash of rendered output |
| `source_profile` | Profile that provided this file |
| `target_category` | Target category name (if any) |

### Checksums

Two checksums enable change detection:

- **source_checksum**: Detects template changes
- **output_checksum**: Detects user modifications

| Source Changed | Output Changed | Status |
|----------------|----------------|--------|
| No | No | `unchanged` |
| Yes | No | `source_changed` |
| No | Yes | `user_modified` |
| Yes | Yes | `conflict` |

## Configuration Precedence

### Variable Precedence

From lowest to highest priority:

1. **Global config** (`config.yaml` variables)
2. **Profile inheritance chain** (ancestor → child)
3. **Project config** (`.weftlo.yaml` variables)

```yaml
# Global: author = "Team"
# Profile base: author = "Backend Team"
# Profile child: (not set)
# Project: author = "Project Owner"

# Result: author = "Project Owner"
```

### Target Override Precedence

From lowest to highest priority:

1. **Profile targets** (content.yaml)
2. **Global universal overrides** (config.yaml target_overrides)
3. **Global profile-specific overrides** (config.yaml profile_overrides)
4. **Project overrides** (.weftlo.yaml target_overrides)

### Install Prefix Precedence

1. `--install-prefix` CLI flag
2. Project config `.weftlo.yaml`
3. Global config `config.yaml`
4. Default: `weftlo`

### Profile Resolution Precedence

1. `--profile` CLI flag
2. Project config `.weftlo.yaml` profiles
3. Global config `config.yaml` default_profile

---

## Related Documents

- [CLI Reference](./cli.md) — Commands that use these configurations
- [Profile System Specification](../specs/profiles.md) — How configurations are used
- [ADR-0005: Deep Merge for Variables](../decisions/0005-deep-merge-for-variables.md) — Variable merging rationale

---

<!-- Migrated from weftlo-current-implementation/configuration-reference.md -->
