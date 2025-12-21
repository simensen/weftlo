# Weftlo Roadmap

This document outlines the current state, potential future development directions, and known limitations for Weftlo.

## Current State (v1.0)

### Core Features

| Feature | Status | Description |
|---------|--------|-------------|
| Profile Management | Complete | Create, list, and manage profiles |
| Profile Inheritance | Complete | Single inheritance with variable merging |
| Template Rendering | Complete | Go templates with custom functions |
| Target Routing | Complete | Named targets with override support |
| Change Detection | Complete | Checksum-based file status tracking |
| Install Command | Complete | Install profiles to projects |
| Update Command | Complete | Sync changes from profiles |
| Status Command | Complete | Display installation status |
| Dry-Run Mode | Complete | Preview changes without writing |
| XDG Support | Complete | Standard config directory locations |
| Variable Merging | Complete | Deep merge with conflict detection |

### Recent Implementations

#### Profile Variables Support
- Deep merge semantics for nested variables
- Conflict detection and warnings
- Three-level precedence (global -> profile -> project)

#### Parallel Testing
- Test suite runs in parallel for faster CI
- Fixed race conditions in test utilities

#### Documentation Improvements
- Fixed `@` prefix documentation for `include()` function
- Clarified variable merge behavior

## Potential Future Enhancements

### Profile Registry

**Description:** Remote profile hosting and discovery.

**Potential Features:**
- `weftlo profile pull vendor/name` - Download from registry
- `weftlo profile push` - Publish profile
- Profile versioning and updates
- Private registries

**Considerations:**
- Authentication and access control
- Version compatibility
- Caching and offline support

### Multi-Profile Installation

**Description:** Install multiple profiles simultaneously with explicit merge rules.

**Current State:** Profiles are listed in order and merged sequentially.

**Potential Enhancements:**
- Explicit merge strategies (union, intersection, override)
- Conflict resolution options
- Per-file profile selection

### Interactive Mode

**Description:** Guided profile installation with prompts.

**Potential Features:**
- Variable value prompts during install
- Profile selection wizard
- Interactive conflict resolution

### Watch Mode

**Description:** Automatically update on profile changes.

**Potential Features:**
- `weftlo watch` - Monitor profile for changes
- Auto-update on save
- IDE integration

### Profile Testing

**Description:** Validate profiles before installation.

**Potential Features:**
- `weftlo profile test` - Validate profile structure
- Template syntax checking
- Variable completeness verification
- Output preview

### Profile Diff

**Description:** Compare profiles or show pending changes.

**Potential Features:**
- `weftlo diff` - Show differences from manifest
- `weftlo profile diff a b` - Compare two profiles
- Unified diff format output

### Hooks System

**Description:** Execute scripts during installation lifecycle.

**Potential Hooks:**
- `pre-install` - Before installation
- `post-install` - After installation
- `pre-update` - Before update
- `post-update` - After update
- `pre-file` - Before each file
- `post-file` - After each file

### Template Plugins

**Description:** Custom template functions from external sources.

**Potential Features:**
- Plugin discovery and loading
- Custom function registration
- Plugin configuration in profile

### GUI/TUI Interface

**Description:** Visual interface for profile management.

**Potential Features:**
- Terminal UI for status visualization
- File browser for profile content
- Interactive variable editing

### IDE Extensions

**Description:** IDE integration for profile development.

**Potential Features:**
- VS Code extension
- Template syntax highlighting
- Variable autocompletion
- Live preview

## Known Limitations

### Single Inheritance

Profiles can only extend one parent. Multi-inheritance would require:
- Merge order specification
- Conflict resolution rules
- Diamond dependency handling

**Workaround:** Use multi-profile installation or flatten inheritance.

### No Profile Versioning

Profiles are always current version. Versioning would require:
- Version specification in profile.yaml
- Version constraint syntax
- Migration tooling

**Workaround:** Use git branches/tags for profile versioning.

### Template-Only Processing

Only `.tmpl` files are processed. Cannot:
- Post-process non-template files
- Apply transformations to binary files
- Generate files from data sources

**Workaround:** Use templates for all generated content.

### No Selective Installation

Cannot install subset of profile files. Would require:
- File selection flags
- Interactive file picker
- Pattern-based filtering

**Workaround:** Use ignore patterns or multiple profiles.

## Contributing

### Development Setup

```bash
git clone https://github.com/simensen/weftlo.git
cd weftlo
go build ./...
go test ./...
```

### Running Tests

```bash
# All tests
make test

# With coverage
go test -cover ./...

# Specific package
go test ./internal/domain/template/...
```

### Code Style

- Follow standard Go conventions
- Use `gofmt` for formatting
- Run `go vet` before committing
- Add tests for new functionality

### Pull Request Process

1. Fork the repository
2. Create feature branch
3. Implement changes with tests
4. Update documentation
5. Submit pull request

## Changelog

### v1.0.0 (Current)

- Initial release
- Profile management (create, list)
- Profile inheritance
- Template rendering with custom functions
- Target routing with overrides
- Change detection and update
- XDG Base Directory support
- Variable deep merge with conflict detection
- Dry-run mode
- Verbose and quiet output modes

### Planned for v1.1

- Enhanced error messages
- Performance optimizations
- Additional template functions
- Extended documentation

---

**Note**: Architecture decisions that were previously in this file have been moved to individual ADRs in `decisions/`. See:
- [ADR-0001: Afero Filesystem Abstraction](../decisions/0001-afero-filesystem-abstraction.md)
- [ADR-0002: Clean Architecture Layers](../decisions/0002-clean-architecture-layers.md)
- [ADR-0003: XDG Base Directory Support](../decisions/0003-xdg-base-directory-support.md)
- [ADR-0004: Manifest-Based Change Detection](../decisions/0004-manifest-based-change-detection.md)
- [ADR-0005: Deep Merge for Variables](../decisions/0005-deep-merge-for-variables.md)

<!-- Migrated from weftlo-current-implementation/roadmap.md (excluding Architecture Decisions section) -->
