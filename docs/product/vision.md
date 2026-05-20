# Weftlo Vision

## Overview

Weftlo is a CLI tool for managing configuration profiles that can be installed into projects for AI coding assistants. It supports profile inheritance, template rendering, and configuration management across multiple projects.

## Related Documents

- [Roadmap](./roadmap.md) — Future development plans
- [Architecture Overview](../architecture/overview.md) — Technical architecture
- [Profile System Specification](../specs/profiles.md) — Core feature specification

## Problem Statement

Weftlo solves the problem of managing configuration profiles for AI coding assistants across multiple projects. Without a tool like Weftlo, developers must:
- Manually copy configuration files between projects
- Maintain separate versions of configurations
- Struggle to keep configurations consistent

## Quick Start

Copy-paste the block below. In under a minute you'll have a rendered AI-rules file
sitting in `.claude/` of a project, ready to edit.

```bash
# 1. Install
go install github.com/simensen/weftlo/cmd/weftlo@latest

# 2. Initialize (creates ~/.config/weftlo/ with a starter profile)
weftlo init

# 3. Install the starter profile into a project
cd ~/projects/my-app   # or any project directory
weftlo install

# 4. See what got rendered
cat .claude/CLAUDE.md
```

The `cat` should print a rendered `CLAUDE.md` with `{{ .Variables.company }}`
substituted in. That file is the starter profile's template, rendered into a
directory your AI assistant actually reads.

### Customize

Edit the template at `~/.config/weftlo/profiles/default/default/content/CLAUDE.md.tmpl`,
then run `weftlo update` in your project to sync changes. Add variables under
`variables:` in the profile's `profile.yaml` and reference them as
`{{ .Variables.your_var }}`. See the [template reference](../reference/templates.md)
for the full syntax.

### Inherit from another profile

A profile can build on another by setting `inherits_from:`:

```yaml
# profile.yaml
name: acme/backend
inherits_from: acme/base
variables:
  tier: backend
```

The child inherits content and variables from the parent and can override either.

### Use in multiple projects

The same profile can be installed into any number of projects:

```bash
cd ~/projects/service-a && weftlo install --profile acme/backend
cd ~/projects/service-b && weftlo install --profile acme/backend
```

Run `weftlo update` in each project after editing the profile to pull changes through.

## Key Concepts

### Profiles

Profiles are collections of configuration files that can be installed into projects. Each profile:
- Has a name in `vendor/name` format (e.g., `mycompany/backend`)
- Can inherit from other profiles
- Contains template files in a `content/` directory
- Can define variables and ignore patterns (and advanced target routing — see below)

### Configuration Hierarchy

Configuration is resolved from multiple sources with defined precedence:

1. **Global Config** (`~/.config/weftlo/config.yaml`) - Lowest precedence
2. **Profile Config** (`profile.yaml` in each profile) - Medium precedence
3. **Project Config** (`.weftlo.yaml` in project root) - Highest precedence

### Template System

Template files use Go's `text/template` syntax and support:
- Variable interpolation: `{{ .Variables.name }}`
- Include directives: `{{ include "partial.md" . }}`
- Reference functions: `{{ reference "other-file.md" }}`
- All standard Go template functions

### Routing (advanced)

By default, weftlo installs every rendered file under the project's install prefix
(default: `.claude/`). For advanced routing — sending different files to different
directories, redirecting or suppressing files at the project level, or applying
overrides across all profiles — see the
[configuration reference](../reference/configuration.md) for field-level docs and
the [architecture overview's Routing System](../architecture/overview.md#routing-system)
for the full four-tier precedence model. New users should ignore this until they
have a single profile installing cleanly.

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────────┐
│                         CLI Layer                               │
│  (root.go, install.go, update.go, status.go, init.go)           │
└────────────────────────────────┬────────────────────────────────┘
                                 │
┌────────────────────────────────▼────────────────────────────────┐
│                     Application Layer                           │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐              │
│  │  Rendering  │  │   Routing   │  │   Install   │              │
│  │  Pipeline   │  │   Router    │  │   Service   │              │
│  └─────────────┘  └─────────────┘  └─────────────┘              │
└────────────────────────────────┬────────────────────────────────┘
                                 │
┌────────────────────────────────▼────────────────────────────────┐
│                       Domain Layer                              │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐              │
│  │   Profile   │  │  Manifest   │  │  Template   │              │
│  │   Config    │  │  Generator  │  │   Engine    │              │
│  └─────────────┘  └─────────────┘  └─────────────┘              │
└────────────────────────────────┬────────────────────────────────┘
                                 │
┌────────────────────────────────▼────────────────────────────────┐
│                   Infrastructure Layer                          │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐              │
│  │   Config    │  │   Profile   │  │  Filesystem │              │
│  │   Loader    │  │   Loader    │  │   (afero)   │              │
│  └─────────────┘  └─────────────┘  └─────────────┘              │
└─────────────────────────────────────────────────────────────────┘
```

## Requirements

- Go 1.21 or later (for building from source)
- macOS, Linux, or Windows

## Installation

### From Source

```bash
git clone https://github.com/simensen/weftlo.git
cd weftlo
go build -o weftlo ./cmd/weftlo
```

### With Version

```bash
go build -ldflags "-X main.version=1.0.0" -o weftlo ./cmd/weftlo
```

---

<!-- Migrated from weftlo-current-implementation/README.md -->
