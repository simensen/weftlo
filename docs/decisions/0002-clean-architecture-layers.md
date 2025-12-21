# ADR-0002: Clean Architecture Layers

**Status**: Accepted
**Date**: 2026-01-03
**Decision-Makers**: Weftlo Team

---

## Context

Need clear separation of concerns for maintainability as the codebase grows.

## Decision

Implement four-layer architecture:
1. CLI Layer - User interaction
2. Application Layer - Business workflows
3. Domain Layer - Core business logic
4. Infrastructure Layer - External concerns

## Consequences

### Positive

- Domain layer has no external dependencies
- Infrastructure implements domain interfaces
- Easy to swap implementations
- Clear boundaries between concerns

### Negative

- More files and packages to navigate
- Potential for over-engineering simple features

---

<!-- Migrated from weftlo-current-implementation/roadmap.md:58-72 -->
