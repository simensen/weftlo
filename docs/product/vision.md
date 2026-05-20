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

### 1. Initialize Weftlo

```bash
weftlo init
```

This creates the configuration directory structure:
- `~/.config/weftlo/` (or `~/.weftlo/` as fallback)
- `config.yaml` - Global configuration
- `profiles/default/default/` - Default profile

### 2. Create a Custom Profile

```bash
weftlo profile create mycompany/backend
```

### 3. Add Content to Your Profile

Navigate to `~/.config/weftlo/profiles/mycompany/backend/content/` and add your configuration files. Files can be templates using Go template syntax with `.tmpl` extension.

### 4. Install Profile to a Project

```bash
cd /path/to/your/project
weftlo install --profile mycompany/backend
```

### 5. Check Installation Status

```bash
weftlo status
```

### 6. Update After Profile Changes

```bash
weftlo update
```

## Key Concepts

### Profiles

Profiles are collections of configuration files that can be installed into projects. Each profile:
- Has a name in `vendor/name` format (e.g., `mycompany/backend`)
- Can inherit from other profiles
- Contains template files in a `content/` directory
- Can define variables, targets, and ignore patterns

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

### Routing and Targets

Files are routed to target directories during installation:
- Default target: All files go to the install prefix (default: `.claude/`)
- Named targets: Route specific files to different directories
- Target overrides: Customize or suppress file routing at project level

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
