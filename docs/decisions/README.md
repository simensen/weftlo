# Architectural Decision Records

This directory contains all Architectural Decision Records (ADRs) for Weftlo.

## What is an ADR?

An ADR captures a significant architectural or technical decision, including the context, the decision itself, and the consequences. ADRs are immutable once accepted — to change a decision, create a new ADR that supersedes the old one.

## Index

| ADR | Title | Status |
|-----|-------|--------|
| [0000](./0000-template.md) | Template | — |
| [0001](./0001-afero-filesystem-abstraction.md) | Afero Filesystem Abstraction | Accepted |
| [0002](./0002-clean-architecture-layers.md) | Clean Architecture Layers | Accepted |
| [0003](./0003-xdg-base-directory-support.md) | XDG Base Directory Support | Accepted |
| [0004](./0004-manifest-based-change-detection.md) | Manifest-Based Change Detection | Accepted |
| [0005](./0005-deep-merge-for-variables.md) | Deep Merge for Variables | Accepted |

## When to Create an ADR

Create an ADR when:
- Choosing between multiple viable technical approaches
- Making a decision that would be expensive to reverse
- Establishing a pattern that will be followed throughout the codebase
- Deviating from common practice or industry norms

## Creating a New ADR

1. Copy `0000-template.md` to `NNNN-{short-title}.md`
2. Fill in Context, Decision, and Consequences
3. Update this README index
4. Submit for review

See [DOCUMENTATION-GUIDE.md](../DOCUMENTATION-GUIDE.md) for complete guidelines.
