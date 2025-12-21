# ADR-0005: Deep Merge for Variables

**Status**: Accepted
**Date**: 2026-01-03
**Decision-Makers**: Weftlo Team

---

## Context

Need intuitive variable inheritance behavior when profiles extend other profiles or when project overrides profile variables.

## Decision

Deep merge nested maps, override scalars. Child profiles extend parent maps rather than replacing them entirely.

## Consequences

### Positive

- Child profiles can extend parent maps
- Can override specific nested values without losing siblings
- Intuitive behavior for configuration inheritance

### Negative

- Must detect and warn on conflicts
- Slightly more complex merge logic

---

<!-- Migrated from weftlo-current-implementation/roadmap.md:105-117 -->
