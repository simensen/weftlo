# ADR-0003: XDG Base Directory Support

**Status**: Accepted
**Date**: 2026-01-03
**Decision-Makers**: Weftlo Team

---

## Context

Users expect standard config locations on Unix systems. Need to determine where to store global configuration.

## Decision

Follow XDG Base Directory specification:
1. `$XDG_CONFIG_HOME/weftlo/` if set
2. `~/.config/weftlo/` if `~/.config/` exists
3. `~/.weftlo/` as fallback

## Consequences

### Positive

- Consistent with other Unix tools
- Backward compatible with `~/.weftlo/`
- Respects user's XDG configuration

### Negative

- Requires resolver abstraction for testing
- Slightly more complex path resolution

---

<!-- Migrated from weftlo-current-implementation/roadmap.md:74-88 -->
