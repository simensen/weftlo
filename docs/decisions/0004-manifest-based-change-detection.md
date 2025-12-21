# ADR-0004: Manifest-Based Change Detection

**Status**: Accepted
**Date**: 2026-01-03
**Decision-Makers**: Weftlo Team

---

## Context

Need to detect changes for the update command to know which files need updating without re-rendering everything.

## Decision

Track two checksums per file in `.weftlo.manifest.json`:
- **Source checksum**: Original template content
- **Output checksum**: Rendered content

## Consequences

### Positive

- Can distinguish template changes vs user modifications
- Enables conflict detection
- Efficient update operations

### Negative

- Manifest file required in project
- Additional file to manage

---

<!-- Migrated from weftlo-current-implementation/roadmap.md:90-103 -->
