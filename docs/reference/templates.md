# Template Reference

Complete reference for Weftlo's template system, including syntax, functions, and variables.

## Table of Contents

1. [Overview](#overview)
2. [Template Files](#template-files)
3. [Template Syntax](#template-syntax)
4. [Template Context](#template-context)
5. [Built-in Functions](#built-in-functions)
6. [Standard Go Template Functions](#standard-go-template-functions)
7. [Partials](#partials)
8. [Examples](#examples)
9. [Troubleshooting](#troubleshooting)

## Overview

Weftlo uses Go's `text/template` package with custom extensions. Templates enable dynamic content generation based on variables, environment, and other files.

### Key Concepts

- Templates use `{{ }}` delimiters for actions
- Variables are accessed via `.Variables.name`
- Custom functions extend standard Go template functionality
- Partials allow reusable template fragments

## Template Files

### File Naming

| Pattern | Type | Behavior |
|---------|------|----------|
| `file.md` | Static | Copied without processing |
| `file.md.tmpl` | Template | Processed, `.tmpl` suffix removed |
| `_partial.md` | Partial | Not installed, available for `include()` |
| `_partial.md.tmpl` | Partial template | Not installed, processed when included |

### Template Discovery

Files in the profile's `content/` directory are discovered and processed:

```
content/
├── README.md              # Static → README.md
├── config.md.tmpl         # Template → config.md
├── skills/
│   ├── coding.md.tmpl     # Template → skills/coding.md
│   └── _common.md         # Partial (not installed)
└── _header.md             # Partial (not installed)
```

## Template Syntax

### Actions

Actions are enclosed in `{{ }}`:

```
{{ .Variables.name }}
{{ if condition }}...{{ end }}
{{ range .Variables.items }}...{{ end }}
```

### Variables

Access variables using dot notation:

```
{{ .Variables.project_name }}
{{ .Variables.config.database.host }}
{{ .Variables.features.logging }}
```

### Text Output

Plain text is output verbatim:

```
# My Project

This is plain text.
Version: {{ .Variables.version }}
```

### Whitespace Control

Use `-` to trim whitespace:

```
{{- .Variables.name -}}     # Trim both sides
{{- .Variables.name }}      # Trim left only
{{ .Variables.name -}}      # Trim right only
```

Example:
```
Items:
{{- range .Variables.items }}
  - {{ . }}
{{- end }}
```

### Comments

```
{{/* This is a comment */}}
{{- /* Comment with whitespace trimming */ -}}
```

## Template Context

Templates receive a context object with these fields:

### .Variables

Merged variables from all sources:

```
{{ .Variables.project_name }}
{{ .Variables.author }}
{{ .Variables.config.setting }}
```

### .Env

Environment variables:

```
{{ .Env.HOME }}
{{ .Env.USER }}
{{ .Env.DATABASE_URL }}
```

### .Profile

Current profile name:

```
Profile: {{ .Profile }}
```

## Built-in Functions

### include

Include another template file and render it with the provided context.

**Syntax:**
```
{{ include "path" context }}
```

**Parameters:**
- `path`: Relative path to the file to include
- `context`: Data to pass to the included template (usually `.`)

**Examples:**

```
{{/* Include from same directory */}}
{{ include "_header.md" . }}

{{/* Include from subdirectory */}}
{{ include "partials/_common.md" . }}

{{/* Include with @ prefix (profile content root) */}}
{{ include "@_shared/header.md" . }}

{{/* Include from parent directory */}}
{{ include "../_shared/footer.md" . }}
```

**Behavior:**
- Relative paths are resolved from the including file's directory
- `@` prefix references the profile's content root
- Included files can themselves include other files
- Circular includes cause an error

### reference

Reference another file's rendered content for cross-file dependencies.

**Syntax:**
```
{{ reference "path" }}
```

**Parameters:**
- `path`: Path to the file to reference (relative to content root)

**Examples:**

```
## Related Commands

{{ reference "commands/build.md" }}

## See Also

{{ reference "skills/coding.md" }}
```

**Behavior:**
- Returns the rendered content of the referenced file
- Useful for creating indexes or aggregate documentation
- The referenced file is rendered with the same context

### referenceGlob

Reference multiple files matching a glob pattern.

**Syntax:**
```
{{ referenceGlob "pattern" }}
```

**Parameters:**
- `pattern`: Glob pattern to match files

**Examples:**

```
## All Skills

{{ referenceGlob "skills/*.md" }}

## All Commands

{{ referenceGlob "commands/**/*.md" }}
```

**Supported Patterns:**
- `*` - Match any characters except `/`
- `**` - Match any characters including `/`
- `?` - Match single character
- `[abc]` - Match character set

### default

Provide a default value if the variable is empty or undefined.

**Syntax:**
```
{{ value | default "fallback" }}
```

**Examples:**

```
Version: {{ .Variables.version | default "1.0.0" }}
Author: {{ .Variables.author | default "Unknown" }}
Debug: {{ .Variables.debug | default "false" }}
```

## Standard Go Template Functions

All standard Go template functions are available:

### Comparison

```
{{ if eq .Variables.env "production" }}Production{{ end }}
{{ if ne .Variables.env "development" }}Not development{{ end }}
{{ if lt .Variables.count 10 }}Less than 10{{ end }}
{{ if le .Variables.count 10 }}Less than or equal to 10{{ end }}
{{ if gt .Variables.count 5 }}Greater than 5{{ end }}
{{ if ge .Variables.count 5 }}Greater than or equal to 5{{ end }}
```

### Boolean Logic

```
{{ if and .Variables.a .Variables.b }}Both true{{ end }}
{{ if or .Variables.a .Variables.b }}At least one true{{ end }}
{{ if not .Variables.disabled }}Enabled{{ end }}
```

### Conditionals

```
{{/* if-else */}}
{{ if .Variables.enabled }}
Enabled
{{ else }}
Disabled
{{ end }}

{{/* if-else if-else */}}
{{ if eq .Variables.tier "premium" }}
Premium tier
{{ else if eq .Variables.tier "standard" }}
Standard tier
{{ else }}
Free tier
{{ end }}
```

### Iteration

```
{{/* Range over slice */}}
{{ range .Variables.items }}
- {{ . }}
{{ end }}

{{/* Range with index */}}
{{ range $index, $item := .Variables.items }}
{{ $index }}: {{ $item }}
{{ end }}

{{/* Range over map */}}
{{ range $key, $value := .Variables.config }}
{{ $key }}: {{ $value }}
{{ end }}

{{/* Range with else (empty case) */}}
{{ range .Variables.items }}
- {{ . }}
{{ else }}
No items
{{ end }}
```

### Variables

```
{{/* Define variable */}}
{{ $name := .Variables.project_name }}
{{ $name }}

{{/* Reassign variable */}}
{{ $count := 0 }}
{{ range .Variables.items }}
{{ $count = add $count 1 }}
{{ end }}
Total: {{ $count }}
```

### String Functions

```
{{ printf "%s-%s" .Variables.prefix .Variables.name }}
{{ print .Variables.a .Variables.b }}
{{ println "Line with newline" }}
```

### Type Functions

```
{{ len .Variables.items }}
{{ index .Variables.items 0 }}
{{ slice .Variables.items 1 3 }}
```

### HTML (escaped)

```
{{ html .Variables.content }}
{{ js .Variables.script }}
{{ urlquery .Variables.param }}
```

## Partials

### Creating Partials

Files starting with `_` are partials:

```
content/
├── _header.md         # Partial
├── _footer.md         # Partial
├── _partials/
│   ├── _nav.md        # Partial in subdirectory
│   └── _sidebar.md    # Partial in subdirectory
└── index.md.tmpl      # Template that uses partials
```

### Using Partials

**Include a partial:**
```
{{ include "_header.md" . }}

Main content here.

{{ include "_footer.md" . }}
```

**Include from subdirectory:**
```
{{ include "_partials/_nav.md" . }}
```

**Include with @ prefix:**
```
{{ include "@_shared/_common.md" . }}
```

### Partial with Parameters

Partials receive the context you pass:

**_header.md:**
```
# {{ .Variables.title }}

Author: {{ .Variables.author }}
```

**Using the partial:**
```
{{ include "_header.md" . }}
```

### Nested Partials

Partials can include other partials:

**_layout.md:**
```
{{ include "_header.md" . }}

{{ .content }}

{{ include "_footer.md" . }}
```

## Examples

### Example 1: Basic Template

**skills/coding.md.tmpl:**
```markdown
# {{ .Variables.project_name }} Coding Guidelines

Author: {{ .Variables.author | default "Development Team" }}
Version: {{ .Variables.version }}

## Language: {{ .Variables.language }}

Follow {{ .Variables.company }} coding standards.

{{ if .Variables.linting_enabled }}
## Linting

Run the linter before committing:
```bash
npm run lint
```
{{ end }}
```

### Example 2: Iteration

**skills/features.md.tmpl:**
```markdown
# Features

{{- range .Variables.features }}
## {{ .name }}

{{ .description }}

Status: {{ .status | default "planned" }}

{{ end }}
```

### Example 3: Conditional Content

**config.md.tmpl:**
```markdown
# Configuration

Environment: {{ .Variables.environment }}

{{ if eq .Variables.environment "production" }}
## Production Settings

- Debug: disabled
- Log level: error
- Cache: enabled
{{ else if eq .Variables.environment "staging" }}
## Staging Settings

- Debug: enabled
- Log level: info
- Cache: disabled
{{ else }}
## Development Settings

- Debug: enabled
- Log level: debug
- Cache: disabled
{{ end }}
```

### Example 4: Nested Variables

**database.md.tmpl:**
```markdown
# Database Configuration

{{ with .Variables.database }}
Host: {{ .host }}
Port: {{ .port }}
Name: {{ .name }}
{{ if .ssl }}
SSL: enabled
{{ end }}
{{ end }}
```

### Example 5: Using Environment Variables

**deploy.md.tmpl:**
```markdown
# Deployment

## Environment

- User: {{ .Env.USER }}
- Home: {{ .Env.HOME }}
{{ if .Env.CI }}
- Running in CI: yes
{{ else }}
- Running in CI: no
{{ end }}

## Secrets

Database URL: {{ .Env.DATABASE_URL | default "not set" }}
```

### Example 6: Index Page with References

**index.md.tmpl:**
```markdown
# Documentation Index

## Skills

{{ referenceGlob "skills/*.md" }}

## Commands

{{ referenceGlob "commands/*.md" }}
```

### Example 7: Partials for Reuse

**_partials/_code-review.md:**
```markdown
## Code Review Guidelines

1. Review for correctness
2. Check for security issues
3. Verify test coverage
4. Ensure documentation
```

**skills/coding.md.tmpl:**
```markdown
# Coding Skill

{{ include "_partials/_code-review.md" . }}

## Additional Guidelines

Project-specific guidelines here.
```

### Example 8: Complex Data Structures

**team.md.tmpl:**
```markdown
# Team Information

{{ range .Variables.teams }}
## {{ .name }}

Lead: {{ .lead }}

### Members
{{ range .members }}
- {{ .name }} ({{ .role }})
{{ end }}

{{ end }}
```

**Variables:**
```yaml
teams:
  - name: Platform
    lead: Alice
    members:
      - name: Bob
        role: Backend
      - name: Carol
        role: Frontend
  - name: Mobile
    lead: Dave
    members:
      - name: Eve
        role: iOS
```

## Troubleshooting

### Common Errors

#### "undefined variable"

```
Error: template: file.md.tmpl:5: undefined variable "$.name"
```

**Cause:** Accessing a variable that doesn't exist.

**Fix:** Use `default` function or check with `if`:
```
{{ .Variables.name | default "Unknown" }}
{{ if .Variables.name }}{{ .Variables.name }}{{ end }}
```

#### "can't evaluate field"

```
Error: template: file.md.tmpl:3: can't evaluate field X in type Y
```

**Cause:** Accessing a field on wrong type (e.g., string instead of map).

**Fix:** Verify variable structure matches expected type.

#### "include: file not found"

```
Error: include: file "_missing.md" not found
```

**Cause:** The included file doesn't exist.

**Fix:** Check file path and ensure file exists in content directory.

#### "circular include detected"

```
Error: circular include detected: a.md -> b.md -> a.md
```

**Cause:** File A includes B, which includes A.

**Fix:** Restructure partials to avoid circular dependencies.

### Debugging Tips

1. **Use verbose mode:**
   ```bash
   weftlo install --dry-run --verbose
   ```

2. **Test with simple values:**
   ```yaml
   variables:
     test: "Hello World"
   ```

3. **Check variable structure:**
   ```
   {{/* Debug: print variables */}}
   {{ printf "%#v" .Variables }}
   ```

4. **Validate YAML syntax:**
   Ensure `profile.yaml` and `content.yaml` are valid YAML.

5. **Check for typos:**
   Variable names are case-sensitive.

### Best Practices

1. **Use default values:**
   ```
   {{ .Variables.name | default "Unknown" }}
   ```

2. **Check before accessing:**
   ```
   {{ if .Variables.optional }}{{ .Variables.optional }}{{ end }}
   ```

3. **Use `with` for nested structures:**
   ```
   {{ with .Variables.config }}
   Host: {{ .host }}
   {{ end }}
   ```

4. **Keep templates readable:**
   Use partials for reusable content.

5. **Document expected variables:**
   Add comments or README explaining required variables.

---

## Related Documents

- [Profile System Specification](../specs/profiles.md) — How templates are used in profiles
- [CLI Reference](./cli.md) — Commands that process templates

---

<!-- Migrated from weftlo-current-implementation/template-reference.md -->
