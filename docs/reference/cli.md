# Weftlo CLI Reference

Complete reference for all Weftlo commands, options, and flags.

## Table of Contents

1. [Global Options](#global-options)
2. [weftlo init](#weftlo-init)
3. [weftlo install](#weftlo-install)
4. [weftlo update](#weftlo-update)
5. [weftlo status](#weftlo-status)
6. [weftlo profile create](#weftlo-profile-create)
7. [weftlo profile list](#weftlo-profile-list)
8. [weftlo version](#weftlo-version)
9. [weftlo completion](#weftlo-completion)
10. [Exit Codes](#exit-codes)
11. [Environment Variables](#environment-variables)

## Global Options

These flags can be used with any command:

| Flag | Short | Description |
|------|-------|-------------|
| `--verbose` | `-v` | Enable verbose output showing detailed operations |
| `--quiet` | `-q` | Suppress non-essential output |
| `--help` | `-h` | Show help for any command |

**Note:** `--quiet` takes precedence over `--verbose`.

## weftlo init

Initialize the global Weftlo configuration directory.

### Synopsis

```bash
weftlo init [flags]
```

### Description

Creates the Weftlo configuration directory structure with a default profile. The configuration directory is determined by XDG Base Directory conventions:

1. `$XDG_CONFIG_HOME/weftlo/` if `XDG_CONFIG_HOME` is set
2. `~/.config/weftlo/` if `~/.config/` exists
3. `~/.weftlo/` as fallback

### Created Structure

```
<config-dir>/
├── config.yaml                    # Global configuration
└── profiles/
    └── default/
        └── default/
            ├── profile.yaml       # Profile metadata
            └── content/
                └── README.md      # Default content
```

### Flags

| Flag | Short | Description | Default |
|------|-------|-------------|---------|
| `--force` | `-f` | Skip reinitialize prompt and proceed directly | `false` |
| `--install-prefix` | | Set default install prefix in global config | `""` |

### Examples

```bash
# Initialize with defaults
weftlo init

# Initialize with custom install prefix
weftlo init --install-prefix .claude

# Force reinitialize (skip confirmation)
weftlo init --force
```

### Output

```
Initialized weftlo successfully!

Created:
  ./
  profiles/
  profiles/default/
  profiles/default/default/
  profiles/default/default/content/
  config.yaml
  profiles/default/default/profile.yaml
  profiles/default/default/content/README.md

Next steps:
  Run `weftlo install` in a project to install your profile

Configuration directory: /home/user/.config/weftlo
```

---

## weftlo install

Install a profile to the current project directory.

### Synopsis

```bash
weftlo install [flags]
```

### Description

Renders and installs profile templates into the current project directory. Creates a `.weftlo.yaml` project configuration file and `.weftlo.manifest.json` manifest for change tracking.

**Note:** This command is for initial setup only. If `.weftlo.yaml` already exists, use `weftlo update --add-profile` to add profiles.

### Profile Resolution

The profile is resolved in this order:
1. `--profile` flag (if specified, can be repeated for multiple profiles)
2. `~/.config/weftlo/config.yaml` default_profile

### Install Prefix Resolution

The install prefix (target directory) is resolved in this order:
1. `--install-prefix` flag (if specified)
2. `~/.config/weftlo/config.yaml` install_prefix (if set)
3. Default: `.claude`

### Flags

| Flag | Short | Description | Default |
|------|-------|-------------|---------|
| `--profile` | `-p` | Profile to install (vendor/name format, repeatable) | `[]` |
| `--install-prefix` | | Installation directory prefix | `""` |
| `--dry-run` | `-n` | Preview changes without writing files | `false` |
| `--force` | `-f` | Overwrite existing files | `false` |

### Examples

```bash
# Install using default profile
weftlo install

# Install specific profile
weftlo install --profile mycompany/backend

# Install multiple profiles (merged in order)
weftlo install --profile company/base --profile company/backend

# Preview what would be installed
weftlo install --dry-run

# Install with custom prefix
weftlo install --install-prefix .ai

# Overwrite existing files
weftlo install --force

# Combined flags
weftlo install --profile team/standard --install-prefix .claude --verbose
```

### Output (Normal)

```
Installed profile 'mycompany/backend' successfully!

Files installed to .claude/skills/ (3):
  .claude/skills/coding.md
  .claude/skills/testing.md
  .claude/skills/debugging.md

Files installed to .claude/commands/ (2):
  .claude/commands/build.md
  .claude/commands/deploy.md

Created .weftlo.yaml with profile configuration.

Next steps:
  Run `weftlo status` to see installation state
```

### Output (Dry Run)

```
[dry-run] Would install profile 'mycompany/backend':

New files to create:
  .claude/skills/coding.md
  .claude/skills/testing.md
  .claude/commands/build.md

Existing files (would skip without --force):
  .claude/skills/debugging.md

No changes were made (dry-run mode).
```

### Output (Verbose)

```
Using profile from: --profile flag
Using install prefix from: project config (.claude)
Rendering: skills/coding.md -> .claude/skills/coding.md
Rendering: skills/testing.md -> .claude/skills/testing.md
Suppressed (target override): hooks/pre-commit.md (category: hooks)
...
```

---

## weftlo update

Update installed configuration files from profile templates.

### Synopsis

```bash
weftlo update [flags]
```

### Description

Synchronizes installed files with the latest profile templates. Detects changes using manifest checksums and updates files appropriately.

Can also be used to add or remove profiles from the installation using `--add-profile` and `--remove-profile` flags.

### Prerequisites

- `.weftlo.yaml` must exist (created by `weftlo install`)
- `.weftlo.manifest.json` must exist (created by `weftlo install`)

### File Handling

| Status | Default Behavior | With `--force` | With `--remove-orphans` |
|--------|-----------------|----------------|------------------------|
| New | Create file | Create file | Create file |
| Source Changed | Update file | Update file | Update file |
| User Modified | Skip | Overwrite | Skip |
| Conflict | Skip | Overwrite | Skip |
| Removed | Keep | Keep | Delete |
| Unchanged | No action | No action | No action |

### Flags

| Flag | Short | Description | Default |
|------|-------|-------------|---------|
| `--force` | `-f` | Overwrite user-modified and conflicting files | `false` |
| `--dry-run` | `-n` | Preview changes without writing files | `false` |
| `--remove-orphans` | | Delete files no longer in the profile | `false` |
| `--add-profile` | | Add profile(s) to the installation (repeatable) | `[]` |
| `--remove-profile` | | Remove profile(s) from the installation (repeatable) | `[]` |

### Profile Modification

The `--add-profile` and `--remove-profile` flags allow modifying the installed profile list:

- **Adding profiles**: New profiles are appended to the list. If a profile is already installed, a warning is shown and it's skipped.
- **Removing profiles**: Profiles are removed from the list. An error occurs if the profile is not currently installed.
- **Order of operations**: Removes are processed first, then adds.
- **Config update**: The `.weftlo.yaml` file is automatically updated with the new profile list.

### Examples

```bash
# Update with default behavior
weftlo update

# Preview changes before updating
weftlo update --dry-run

# Force overwrite user modifications
weftlo update --force

# Remove files deleted from profile
weftlo update --remove-orphans

# Combined: force update and remove orphans
weftlo update --force --remove-orphans

# Add a new profile to the installation
weftlo update --add-profile company/monitoring

# Remove a profile from the installation
weftlo update --remove-profile default/default

# Switch profiles (remove one, add another)
weftlo update --remove-profile company/backend --add-profile company/frontend

# Add multiple profiles at once
weftlo update --add-profile company/base --add-profile company/backend
```

### Output (No Changes)

```
No changes needed.
```

### Output (With Changes)

```
Update completed successfully!

Files created in .claude/skills/ (1):
  .claude/skills/new-skill.md

Files updated in .claude/commands/ (2):
  .claude/commands/build.md
  .claude/commands/deploy.md

Files skipped: 1
  - .claude/skills/coding.md: user_modified
```

### Output (Dry Run)

```
[dry-run] Changes that would be made:

Files to create: 1
  [.claude/skills]
    - .claude/skills/new-skill.md

Files to update: 2
  - .claude/commands/build.md
  - .claude/commands/deploy.md

Files to skip: 1
  - .claude/skills/coding.md: user_modified

No changes were made (dry-run mode).
```

---

## weftlo status

Display installation status and file changes.

### Synopsis

```bash
weftlo status [flags]
```

### Description

Shows current installation status including:
- Installed profile(s) and inheritance chain
- Installation timestamp
- Install prefix
- File statuses (unchanged, changed, modified, etc.)

### Prerequisites

- `.weftlo.yaml` must exist
- `.weftlo.manifest.json` must exist

### Flags

| Flag | Short | Description | Default |
|------|-------|-------------|---------|
| `--json` | | Output in JSON format | `false` |

### Examples

```bash
# Human-readable status
weftlo status

# JSON output for scripting
weftlo status --json
```

### Output (Human-Readable)

```
Installation Status

Profile: mycompany/backend
  Inherits from: mycompany/base
Installed: 2024-01-15T10:30:00Z
Install prefix: .claude

Files:

Unchanged (3):
  - .claude/skills/coding.md
  - .claude/skills/testing.md
  - .claude/commands/build.md

Source changed (1):
  - .claude/commands/deploy.md

User modified (1):
  - .claude/skills/debugging.md
```

### Output (JSON)

```json
{
  "profiles": ["mycompany/backend"],
  "inheritance_chain": ["mycompany/base", "mycompany/backend"],
  "installed_at": "2024-01-15T10:30:00Z",
  "install_prefix": ".claude",
  "files": {
    "unchanged": [
      ".claude/skills/coding.md",
      ".claude/skills/testing.md",
      ".claude/commands/build.md"
    ],
    "source_changed": [
      ".claude/commands/deploy.md"
    ],
    "user_modified": [
      ".claude/skills/debugging.md"
    ],
    "conflict": [],
    "new": [],
    "removed": []
  },
  "warnings": []
}
```

### File Status Meanings

| Status | Description |
|--------|-------------|
| `unchanged` | File matches manifest, no changes needed |
| `source_changed` | Template was modified, output should update |
| `user_modified` | User edited the installed file |
| `conflict` | Both template and user file changed |
| `new` | New file in profile, not yet installed |
| `removed` | File removed from profile but still on disk |

---

## weftlo profile create

Create a new configuration profile.

### Synopsis

```bash
weftlo profile create <vendor/name> [flags]
```

### Description

Creates a new profile with the specified name. The name must be in `vendor/name` format.

### Name Requirements

- Format: `vendor/name`
- Characters: alphanumeric, underscores (`_`), hyphens (`-`)
- Examples: `mycompany/backend`, `personal/dotfiles`, `team-a/api_service`

### Created Structure

```
<config-dir>/profiles/<vendor>/<name>/
├── profile.yaml       # Profile configuration
├── content/           # Content directory
└── README.md          # Profile documentation
```

### Flags

| Flag | Short | Description | Default |
|------|-------|-------------|---------|
| `--force` | `-f` | Overwrite existing profile | `false` |

### Examples

```bash
# Create a new profile
weftlo profile create mycompany/backend

# Create with parent vendor that doesn't exist
weftlo profile create newvendor/newprofile

# Overwrite existing profile
weftlo profile create existing/profile --force
```

### Output

```
Created profile 'mycompany/backend' successfully!

Profile directory: /home/user/.config/weftlo/profiles/mycompany/backend

Next steps:
  1. Edit profile.yaml to configure inheritance and variables
  2. Add content files to the content/ directory
  3. Run `weftlo install --profile mycompany/backend` to use this profile
```

---

## weftlo profile list

List available configuration profiles.

### Synopsis

```bash
weftlo profile list [flags]
weftlo profile ls [flags]
```

### Description

Lists all available profiles from the configuration directory.

### Flags

| Flag | Short | Description | Default |
|------|-------|-------------|---------|
| `--json` | | Output in JSON format | `false` |

### Examples

```bash
# List profiles
weftlo profile list

# Use short alias
weftlo profile ls

# JSON output
weftlo profile list --json
```

### Output (Human-Readable)

```
Available profiles:

default/default
mycompany/base
mycompany/backend
mycompany/frontend
personal/dotfiles
```

### Output (JSON)

```json
{
  "profiles": [
    "default/default",
    "mycompany/base",
    "mycompany/backend",
    "mycompany/frontend",
    "personal/dotfiles"
  ]
}
```

---

## weftlo version

Display version information.

### Synopsis

```bash
weftlo version
```

### Description

Shows Weftlo version, Go version, and platform information.

### Examples

```bash
weftlo version
```

### Output

```
weftlo version 1.0.0
Go version: go1.21.0
Platform: darwin/arm64
```

---

## weftlo completion

Generate shell completion scripts.

### Synopsis

```bash
weftlo completion <shell>
```

### Description

Generates shell completion scripts for bash, zsh, fish, or powershell.

### Examples

```bash
# Bash
weftlo completion bash > /etc/bash_completion.d/weftlo

# Zsh
weftlo completion zsh > "${fpath[1]}/_weftlo"

# Fish
weftlo completion fish > ~/.config/fish/completions/weftlo.fish

# PowerShell
weftlo completion powershell > weftlo.ps1
```

---

## Exit Codes

| Code | Meaning |
|------|---------|
| `0` | Success |
| `1` | General error |

### Error Messages

Weftlo provides descriptive error messages with context:

```
Error: Installation not found

No installation exists in the current directory.
Missing: .weftlo.yaml

To install a profile, run: weftlo install
```

```
Error: Profile not found

Profile 'mycompany/invalid' was not found.

Available profiles:
  - default/default
  - mycompany/backend

Run `weftlo profile list` to see all available profiles.
```

```
Error: File conflict

The following files already exist:
  - .claude/skills/coding.md
  - .claude/skills/testing.md

Use --force to overwrite existing files.
```

```
Error: already initialized: .weftlo.yaml exists. Use 'weftlo update --add-profile <profile>' to add profiles.
```

```
Error: profile 'company/old' is not in the current profile list
```

---

## Environment Variables

| Variable | Description |
|----------|-------------|
| `XDG_CONFIG_HOME` | Override default config location |
| `HOME` | User home directory (fallback for config) |

### XDG Base Directory

Weftlo follows XDG Base Directory conventions:

```bash
# Use custom config location
export XDG_CONFIG_HOME=/custom/config
weftlo init  # Creates /custom/config/weftlo/
```

### Template Environment Access

Templates can access environment variables:

```
# In a template file
Database: {{ .Env.DATABASE_URL }}
Environment: {{ .Env.NODE_ENV | default "development" }}
```

---

## Related Documents

- [Profile System Specification](../specs/profiles.md) — How profiles work
- [Configuration Reference](./configuration.md) — Config file formats
- [Template Reference](./templates.md) — Template syntax for profiles

---

<!-- Migrated from weftlo-current-implementation/cli-reference.md -->
