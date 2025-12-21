# Weftlo Documentation — Sync

**For AI Agents**: This document contains instructions for auditing documentation against the current implementation and synchronizing any drift. Use this for periodic maintenance or before releases.

**Terminology**: See the [Glossary](../DOCUMENTATION-GUIDE.md#glossary) for definitions of key terms.

---

## When to Use This Guide

Use this guide for:
- Periodic documentation audits (monthly, quarterly)
- Pre-release documentation verification
- After significant refactoring
- When documentation accuracy is questioned
- Onboarding new team members (verify docs are current)

**Do NOT use this guide for**:
- Updating after known code changes (use `update.md`)
- Quality improvements (use `refine.md`)

---

## Sync Process Overview

```
1. Audit Reference Docs  -> Are CLI/config references current?
2. Verify Cross-Links    -> Are all links valid?
3. Check Specifications  -> Do specs match implementation?
4. Review ADRs           -> Are decisions still current?
5. Resolve Drift         -> Fix discrepancies
6. Report Findings       -> Document audit results
```

---

## Sync Process

### Step 1: Audit Reference Docs

Compare `docs/reference/cli.md` against actual CLI:

```bash
# Get actual CLI help
weftlo --help
weftlo {subcommand} --help
```

For each command documented:
- [ ] Command exists in CLI
- [ ] All options are documented
- [ ] All documented options exist
- [ ] Defaults match actual defaults
- [ ] Examples work as shown

Record discrepancies:

> **CLI Reference Drift**
>
> | Issue | Document Says | CLI Actually |
> |-------|---------------|--------------|
> | Missing option | -- | `--new-flag` exists |
> | Wrong default | `--port=8080` | `--port=3000` |

Similarly audit `docs/reference/configuration.md` against actual config handling.

---

### Step 2: Verify Cross-Links

Check that all internal documentation links are valid:

#### Collect All Links

Scan all Markdown files for internal links:
```markdown
[Link text](../path/to/file.md)
[Link text](./file.md#section)
```

#### Validate Each Link

For each link:
- [ ] Target file exists
- [ ] Target section exists (if anchor specified)
- [ ] Link text is still accurate

Record broken links:

> **Broken Links Found**
>
> | Source File | Link | Issue |
> |-------------|------|-------|
> | `specs/foo.md` | `../decisions/0005-*.md` | File not found |

---

### Step 3: Check Specifications Against Implementation

For each specification in `docs/specs/`:

1. Read the specification
2. Identify key behavioral claims
3. Verify against code implementation
4. Record discrepancies

> **Specification Drift**
>
> | Specification | Claim | Actual Behavior |
> |---------------|-------|-----------------|
> | `profiles.md` | "Variables are shallow merged" | Deep merge with conflict detection |

---

### Step 4: Review ADR Currency

For each ADR in `docs/decisions/`:

- Is the decision still "Accepted"?
- Has it been superseded without marking?
- Is the decision actually implemented?

If implementation differs from ADR:
1. **If intentional**: Create new ADR to supersede
2. **If unintentional**: Flag as implementation bug

---

### Step 5: Resolve Drift

For each discrepancy found:

```
Is the documentation correct and code wrong?
|-- YES -> File implementation bug
|-- NO ↓

Is the code correct and documentation wrong?
|-- YES -> Update documentation (use update.md)
|-- NO ↓

Is this an intentional undocumented change?
|-- YES -> Document the change, possibly new ADR
|-- NO -> Investigate further
```

---

### Step 6: Document Audit Results

Create an audit report:

> **Documentation Sync Audit Report**
>
> **Date**: {date}
>
> ## Summary
>
> | Category | Issues Found | Resolved | Remaining |
> |----------|--------------|----------|-----------|
> | CLI Reference | 5 | 5 | 0 |
> | Config Reference | 2 | 2 | 0 |
> | Specifications | 3 | 2 | 1 (bug filed) |
> | Cross-Links | 4 | 4 | 0 |
> | ADRs | 1 | 1 | 0 |
>
> ## Documents Updated
>
> | Document | Changes |
> |----------|---------|
> | `docs/reference/cli.md` | Added 3 options, fixed 2 defaults |

---

## Quick Reference

### Audit Frequency Recommendations

| Project Phase | Audit Frequency |
|---------------|-----------------|
| Active development | Monthly |
| Maintenance mode | Quarterly |
| Pre-release | Always |
| Post-major-refactor | Always |

### Priority Order for Fixes

1. **Reference doc drift** — Users rely on these
2. **Broken links** — Breaks navigation
3. **Spec drift** — Important for developers
4. **ADR currency** — Historical accuracy
