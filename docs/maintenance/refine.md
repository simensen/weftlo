# Weftlo Documentation — Refine

**For AI Agents**: This document contains instructions for improving documentation quality without implementation changes. Use this for documentation review cycles, clarity improvements, and structural enhancements.

**Terminology**: See the [Glossary](../DOCUMENTATION-GUIDE.md#glossary) for definitions of key terms.

---

## When to Use This Guide

Use this guide for:
- Documentation quality reviews
- Improving clarity and readability
- Fixing structural issues (wrong layer, duplication)
- Enhancing cross-linking
- Filling gaps in existing documentation
- Consolidating scattered information

**Do NOT use this guide for**:
- Updating after code changes (use `update.md`)
- Auditing against implementation (use `sync.md`)

---

## Refinement Categories

1. **Layer Placement** — Is content in the correct layer?
2. **Appendix Structure** — Is large content properly in appendices?
3. **Duplication** — Is information mastered in one place only?
4. **Cross-Linking** — Are related documents properly connected?
5. **ADR Coverage** — Are major decisions documented?
6. **Completeness** — Are all features/options documented?
7. **Template Compliance** — Do documents follow templates?
8. **Clarity** — Is content clear and unambiguous?

---

## Refinement Process

### Step 1: Select Refinement Scope

Determine the scope of this refinement session:

> **Refinement Scope**
>
> What would you like to refine?
> 1. **Full audit** — Review all documentation
> 2. **Single layer** — Focus on one layer (specify which)
> 3. **Single document** — Focus on one document (specify which)
> 4. **Specific category** — Focus on one refinement category
>
> Selection: {1/2/3/4}

---

### Step 2: Layer Placement Review

For each document, verify content is in the correct layer using the classification decision tree in `DOCUMENTATION-GUIDE.md`.

Record misplacements:

> **Layer Placement Issues**
>
> | Document | Section | Current Layer | Should Be |
> |----------|---------|---------------|-----------|
> | `specs/foo.md` | "Why we chose X" | Specifications | Decisions (ADR) |

For each misplacement:
1. Extract the content from current location
2. Create or update document in correct layer
3. Replace original content with a link to new location

---

### Step 3: Appendix Structure Review

Check that large content is properly placed in appendices per `DOCUMENTATION-GUIDE.md` thresholds.

For each document, check:
- [ ] Code blocks < threshold lines (or in appendix)
- [ ] Step lists < threshold items (or in appendix)
- [ ] Tables < threshold rows (or in appendix)
- [ ] Complete file examples in appendix
- [ ] Error catalogs in appendix
- [ ] Shell scripts in appendix

---

### Step 4: Duplication Review

Check for information that appears in multiple places (SSOT violations).

Common patterns:
- Decision rationale in specs -> Move to ADR
- Config details in specs -> Move to reference
- Repeated glossary -> Centralize in vision

For each duplicate:
1. Determine canonical source (use document hierarchy)
2. Keep full content in canonical source
3. Replace duplicate with link

---

### Step 5: Cross-Linking Review

Ensure documents are properly interconnected.

Expected links:
| From | To | Link Purpose |
|------|----|--------------|
| Specification | ADR | Explain "why" for design choices |
| Specification | Reference | Point to detailed syntax/options |
| Architecture | ADR | Justify architectural patterns |
| Architecture | Specification | Deep-dive into component behavior |

Check for:
- [ ] Links to related documents exist
- [ ] No orphan documents (nothing links to them)
- [ ] No dead-end documents (they link to nothing)

---

### Step 6: ADR Coverage Review

Scan specifications and architecture docs for decision signals:
- "We chose X..."
- "We decided to..."
- "The pattern is X because..."
- Trade-off discussions

For each potential decision, ask:
- Is this a significant architectural decision?
- Would it be expensive to reverse?
- Is it already documented as an ADR?

---

### Step 7: Clarity Review

Look for:
- Vague terms ("should", "might", "usually")
- Missing specifics ("the appropriate value")
- Undefined terms (jargon without definition)
- Long paragraphs without structure

For each issue:
- Replace vague terms with specific values
- Add examples
- Break long paragraphs into lists or subsections
- Define technical terms on first use

---

### Step 8: Final Report

> **Documentation Refinement Report**
>
> **Date**: {date}
> **Scope**: {full/layer/document/category}
>
> ## Summary
>
> | Category | Issues Found | Resolved |
> |----------|--------------|----------|
> | Layer Placement | 3 | 3 |
> | Appendix Structure | 4 | 4 |
> | Duplication | 5 | 5 |
> | Cross-Links | 8 | 8 |
> | ADR Coverage | 2 | 2 |
> | Clarity | 6 | 6 |
>
> ## Documents Modified
>
> | Document | Changes |
> |----------|---------|
> | `specs/ports.md` | Moved decision to ADR, added link |

---

## Quick Reference

### Refinement Priority Order

1. **Layer placement** — Foundation for other fixes
2. **Appendix structure** — Ensures thresholds respected
3. **Duplication** — Prevents conflicting information
4. **ADR coverage** — Preserves decision rationale
5. **Cross-links** — Enables navigation
6. **Completeness** — Fills gaps
7. **Clarity** — Polish

### Common Quick Fixes

| Issue | Quick Fix |
|-------|-----------|
| Decision in spec | Extract to ADR, link back |
| Repeated content | Keep in higher-authority doc, link from others |
| Large code block | Move to appendix, link from main doc |
| Missing link | Add "Related Documents" section |
| Long paragraph | Convert to bullet list |
| Vague term | Replace with specific value |
