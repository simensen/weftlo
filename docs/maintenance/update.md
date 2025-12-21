# Weftlo Documentation — Update

**For AI Agents**: This document contains instructions for updating documentation after implementation changes. Follow this process when code changes require documentation updates.

**Terminology**: See the [Glossary](../DOCUMENTATION-GUIDE.md#glossary) for definitions of key terms.

---

## When to Use This Guide

Use this guide when:
- Code implementation has changed
- A bug fix affects documented behavior
- A feature has been modified or extended
- Configuration options have changed
- CLI commands have been added or modified

**Do NOT use this guide for**:
- Quality improvements without code changes (use `refine.md`)
- Periodic audits (use `sync.md`)

---

## Update Process

### Step 1: Identify the Change Scope

Determine what changed in the implementation:

> **Implementation Change Summary**
>
> What changed?
> - [ ] New feature added
> - [ ] Existing feature modified
> - [ ] Feature removed
> - [ ] Bug fix that changes behavior
> - [ ] Configuration options changed
> - [ ] CLI commands changed
> - [ ] Dependencies or tech stack changed
>
> Describe the change: {brief description}

Store as `CHANGE_DESCRIPTION`.

---

### Step 2: Identify Affected Documents

Based on the change type, identify which documents need updating:

#### Change Type -> Document Mapping

| Change Type | Documents to Check |
|-------------|-------------------|
| New feature | Specs, Reference (+ appendices), possibly Architecture |
| Feature modified | Specs, Reference (+ appendices) |
| Feature removed | All layers (remove references from main docs and appendices) |
| Bug fix (behavior change) | Specs |
| Config changed | Reference (config + appendices), Specs if behavior affected |
| CLI changed | Reference (CLI + appendices), Specs if behavior affected |
| Dependencies changed | Implementation (tech-stack + appendices), possibly ADR |

List all potentially affected documents:

> **Potentially Affected Documents**
>
> Based on the change, these documents may need updating:
> - `docs/specs/{feature}.md`
> - `docs/reference/cli.md`
> - ...

---

### Step 3: Check Document Hierarchy

Determine which document is the **authoritative source** for the changed information.

Hierarchy (highest to lowest authority):
1. ADRs — If the change contradicts an ADR, the ADR needs updating first (or superseding)
2. Vision — Rarely needs updating for implementation changes
3. Specifications — Primary target for behavior changes
4. Reference — Primary target for CLI/config changes
5. Architecture — Update if structural change

---

### Step 4: Check for Decision Changes

Ask: Does this implementation change imply a new architectural decision?

Signals that an ADR is needed:
- "We changed from X to Y because..."
- "We're now using a different pattern..."
- "This breaks backward compatibility because..."

If a new decision is implied, create a new ADR in `decisions/` before proceeding.

---

### Step 5: Update Authoritative Document

Update the authoritative source document first.

#### For Specification Updates

1. Read the current specification (`docs/specs/{feature}.md`)
2. Locate the section(s) that describe the changed behavior
3. Update the behavior description to match new implementation
4. Update any examples that are now incorrect
5. Update edge cases if affected

#### For Reference Updates

1. Read the current reference document (`docs/reference/{topic}.md`)
2. Locate the section(s) for the changed command/option
3. Update syntax, options, defaults as needed
4. Update examples in main doc (brief) and appendices (detailed)

---

### Step 6: Cascade Updates

After updating the authoritative source, update derived/referencing documents.

#### Update Flow

```
ADR (if new decision)
    |
Specification (behavior details)
    |
Reference (command/config details)
    |
Architecture (if structural change)
```

For each document in the cascade:
1. Check if it references the changed content
2. Update references if needed
3. Remove any duplicated information that's now stale
4. Add links to updated authoritative sources

---

### Step 7: Verify Cross-Links

Ensure all cross-links between documents are still valid:
- Check links from specifications -> ADRs
- Check links from specifications -> reference docs
- Check links from architecture -> specifications

---

### Step 8: Final Report

Present a summary of all updates:

> **Documentation Update Complete**
>
> **Change**: {CHANGE_DESCRIPTION}
>
> **Documents Updated**:
> | Document | Changes Made |
> |----------|--------------|
> | `docs/specs/X.md` | Updated Y behavior |
> | `docs/reference/cli.md` | Added Z option |
>
> **Cross-Links Verified**: Yes
>
> **New ADR Created**: {Yes: ADR-NNNN | No}

---

## Quick Reference

### Change Type -> Primary Document

| Change | Update First |
|--------|--------------|
| Behavior change | Specification |
| New CLI option | Reference (CLI) |
| Config change | Reference (Config) |
| Architecture change | Architecture -> new ADR if significant |
| Decision change | New ADR (supersedes old) |
