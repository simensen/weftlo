# ADR-0001: Afero Filesystem Abstraction

**Status**: Accepted
**Date**: 2026-01-03
**Decision-Makers**: Weftlo Team

---

## Context

Need to test filesystem operations without touching real files. The application performs extensive file operations for profile loading, template rendering, and file installation.

## Decision

Use `afero.Fs` interface throughout codebase for all filesystem operations.

## Consequences

### Positive

- All filesystem operations are injectable
- Tests use in-memory filesystem (`afero.MemMapFs`)
- Production uses OS filesystem (`afero.OsFs`)
- Comprehensive testing without touching real filesystem

### Negative

- Slight indirection in code
- Dependency on afero library

---

<!-- Migrated from weftlo-current-implementation/roadmap.md:40-56 -->
