# Weftlo Documentation Guide

**Version**: 1.0.0
**Last Updated**: 2026-01-03
**Migration Note**: This documentation was migrated from `weftlo-current-implementation/` using the Layered Documentation System.

---

## Overview

This documentation follows a layered approach where each layer has a specific purpose. The goal is to prevent drift between documents by ensuring each piece of information has exactly one canonical source.

### Installed Layers

| Layer | Directory | Purpose |
|-------|-----------|---------|
| **1. Decisions** | `decisions/` | Capture *why* choices were made (ADRs) |
| **2. Product** | `product/` | Define *what* we're building and *why* |
| **3. Architecture** | `architecture/` | Show *how* components relate |
| **4. Specifications** | `specs/` | Detail *how* features work |
| **5. Reference** | `reference/` | Provide lookup information |

---

## Glossary

### Directory Terminology

#### Documentation Root (`DOCS_DIR`)

The project-specific directory where documentation is installed and maintained: `docs/`

#### Legacy Documentation Directory (`LEGACY_DOCS_DIR`)

The directory containing the original documentation that was migrated: `weftlo-current-implementation/`

### Core System Terms

#### Layer

A distinct category of documentation with a specific purpose, stability level, audience, and lifecycle.

#### Appendix

A supplementary file containing large content (code examples, detailed references, complete scripts) that exceeds thresholds. Stored in `appendices/{topic}/` directories.

#### Main Document

The primary document for a topic (e.g., `cli.md`) containing overview and key information, with links to appendices for detailed content.

#### Canonical Source

The single authoritative location where a piece of information is mastered. Other documents should reference rather than duplicate content from the canonical source.

#### Single Source of Truth (SSOT)

The principle that each piece of information should live in exactly one place to prevent drift and conflicts.

### Layer-Specific Terms

#### ADR (Architectural Decision Record)

An immutable document in Layer 1 capturing a significant technical or product decision, including context, the decision itself, and consequences.

---

## Content Thresholds (Configurable)

These thresholds control how content is classified. Edit to customize for this project.

| Threshold | Value | Purpose |
|-----------|-------|---------|
| `CODE_BLOCK_LINES` | 50 | Code blocks >= this go to appendix |
| `STEP_LIST_ITEMS` | 10 | Step lists >= this go to appendix |
| `TABLE_ROWS` | 20 | Tables >= this go to appendix |
| `EXAMPLE_FILE_ALWAYS_APPENDIX` | true | Complete file examples -> appendix |
| `ERROR_CATALOG_ALWAYS_APPENDIX` | true | Error catalogs -> appendix |
| `SHELL_SCRIPT_ALWAYS_APPENDIX` | true | Shell scripts -> appendix |

---

## Classification Decision Tree

Use this flowchart to classify new content:

```
Is this explaining WHY a choice was made?
|-- YES -> Layer 1: Decisions (ADR)
|-- NO ↓

Is this about product vision, goals, or concepts?
|-- YES -> Layer 2: Product
|-- NO ↓

Is this showing how components relate (diagrams, topology)?
|-- YES -> Layer 3: Architecture
|-- NO ↓

Is this detailing HOW a feature works (states, flows, schemas)?
|-- YES -> Layer 4: Specifications
|-- NO ↓

Is this a lookup table (commands, options, codes)?
|-- YES -> Layer 5: Reference
|-- NO -> Reconsider: may not need documentation
```

---

## Document Hierarchy and Authority

When documents conflict, higher-authority documents win. Update lower documents to match.

1. **ADRs** (highest) - Decisions are immutable once accepted
2. **Product Vision** - High-level "what" and "why"
3. **Architecture** - System structure and patterns
4. **Specifications** - Feature behavior details
5. **Reference** (lowest) - Factual lookup information

---

## Cross-Layer Linking

Use relative Markdown links to connect layers:

```markdown
<!-- In a specification -->
See [ADR-0001](../decisions/0001-afero-filesystem-abstraction.md) for rationale.

<!-- From main doc to appendix -->
For complete examples, see [Detailed Examples](./appendices/cli/detailed-examples.md).

<!-- In architecture -->
For command details, see the [CLI Reference](../reference/cli.md).
```

### Link Direction Guidelines

| From | To | When |
|------|----|------|
| Specification | ADR | Explaining *why* a design choice was made |
| Architecture | ADR | Justifying architectural patterns |
| Architecture | Specification | Deep-diving into component behavior |
| Reference | Specification | Providing conceptual context |

---

## Directory Structure

```
docs/
|-- DOCUMENTATION-GUIDE.md      # This file
|
|-- maintenance/                 # Maintenance workflows
|   |-- audit.md                 # Documentation audit
|   |-- refine.md                # Quality improvement
|   |-- sync.md                  # Implementation sync
|   |-- update.md                # Post-change updates
|
|-- decisions/                   # Layer 1: ADRs (simple files)
|   |-- README.md                # Index of all ADRs
|   |-- 0000-template.md         # Template for new ADRs
|   |-- 0001-afero-filesystem-abstraction.md
|   |-- ...
|
|-- product/                     # Layer 2: Product
|   |-- README.md                # Index
|   |-- vision.md                # Product vision
|   |-- roadmap.md               # Future considerations
|
|-- architecture/                # Layer 3: Architecture
|   |-- README.md                # Index
|   |-- overview.md              # System architecture
|
|-- specs/                       # Layer 4: Specifications
|   |-- README.md                # Index
|   |-- profiles.md              # Profile system spec
|
|-- reference/                   # Layer 5: Reference
|   |-- README.md                # Index
|   |-- cli.md                   # CLI command reference
|   |-- configuration.md         # Configuration options
|   |-- templates.md             # Template syntax reference
```

---

## Appendix Guidelines

### When to Create an Appendix

Move content to an appendix when it exceeds the thresholds above or matches these patterns:

- Complete file examples (has shebang or package declaration)
- Scaffold code (multiple related code blocks)
- Error catalogs (list of error codes/messages)
- Workflow examples (multi-step with context)

### Appendix Structure

Appendices are stored in `appendices/{topic}/` where `{topic}` matches the main document basename:

```
reference/
|-- cli.md                       # Main CLI reference
|-- appendices/
    |-- cli/                     # Appendices for cli.md
        |-- detailed-examples.md
        |-- error-messages.md
```

### Appendix Document Format

```markdown
# [Appendix Title]

> **Parent**: [Main Document](../../cli.md)
> **Purpose**: [What this appendix contains]

## Content

[The detailed content]

---

*This appendix supports [CLI Reference](../../cli.md).*
```

---

## Maintenance Workflows

| Task | Guide | When to Use |
|------|-------|-------------|
| Code changed | `maintenance/update.md` | After implementation changes |
| Periodic audit | `maintenance/sync.md` | Before releases, periodic maintenance |
| Quality improvement | `maintenance/refine.md` | Documentation review cycles |
| Compare docs vs source | `maintenance/audit.md` | After migration, or to check for drift |

---

## Quick Reference

### Creating a New ADR

1. Copy `decisions/0000-template.md`
2. Rename to `decisions/NNNN-{short-title}.md`
3. Fill in Context, Decision, and Consequences
4. Update `decisions/README.md` index

### Adding a New Specification

1. Create `specs/{feature-name}.md`
2. Follow the specification template structure
3. Link to relevant ADRs for rationale
4. Update `specs/README.md` index

### Updating Reference Docs

1. Find the relevant file in `reference/`
2. Update the content to match implementation
3. Check if content should go to appendix
4. Verify cross-links are correct
