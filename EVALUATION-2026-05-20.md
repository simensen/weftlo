# Weftlo UX/DX Evaluation — 2026-05-20

> **Author of this report:** Claude (Opus 4.7), at the request of the project author.
> **Status:** One-shot evaluation. Findings reflect the state of `main` at commit `52388b2` on 2026-05-20.
> **Companion document:** [`HANDOFF-UX-FIXES.md`](./HANDOFF-UX-FIXES.md) — actionable, per-item work for the 8 recommendations below.

---

## 1. Why this evaluation exists

The author of weftlo built the tool several months prior to this review and had not used it on a real project — despite the tool being designed to solve a problem the author actually has. The hypothesis was that the user-facing complexity outweighed the felt benefit even for the author, and that this should be diagnosed before deciding whether to simplify, repitch, or restart.

We wanted three things out of this review:

1. An honest, evidence-based account of where the UX fails.
2. A clear statement of where the tool succeeds (so we don't throw out what works).
3. A short, ranked list of changes that would most improve first-60-seconds experience.

---

## 2. The plan

Three evaluation methods were chosen, against a single target use case.

**Target use case:** *Multi-vendor / org-wide distribution* — a vendor publishes profiles, many consumers install and extend them. This is the design's apparent ambition and the hardest case to make feel light.

**Methods:**

1. **Cold-start dogfood** — actually run the tool end-to-end against the target use case, following the published docs verbatim. Log friction in real time.
2. **Concept-count + journey map** — for each persona (vendor author, consumer), list every concept required to succeed at each milestone, informed by the dogfood experience.
3. **"Why not just X?" comparison** — compare against adjacent tools (copier, cookiecutter, chezmoi, etc.) to identify weftlo's true differentiator, or its absence.

The plan deliberately excluded a doc-walk-through audit (already partially done in an earlier read-through that flagged the `extends`/`inherits_from` drift) and was approved before execution.

Full original plan: `~/.claude/plans/review-the-implementation-and-mutable-kay.md`.

---

## 3. What was actually done

### Phase 1 — Build and sandbox

- Built the binary: `go build -o /tmp/weftlo-eval/weftlo ./cmd/weftlo`. ✅ Clean build on Go 1.25.7.
- Created sandbox: `/tmp/weftlo-eval/config/` for `XDG_CONFIG_HOME`, plus consumer project directories.
- All commands run with `XDG_CONFIG_HOME=/tmp/weftlo-eval/config` to keep the real `~/.config/weftlo` untouched. Sandbox removed at end.

### Phase 2 — Cold-start dogfood

Walked the multi-vendor scenario as a first-time user, following `docs/product/vision.md`'s Quick Start verbatim where applicable:

1. `weftlo init` — inspected the created tree and starter files.
2. `weftlo install` (no flags) in an empty project — observed the default behavior.
3. `weftlo profile create acme/base` — inspected the layout it produced, **followed the generated README's instructions** to add a `.tmpl` file at the profile root, then installed.
4. (After discovering the silent no-op) moved the template into `content/` and re-installed.
5. `weftlo profile create acme/backend` extending `acme/base` — overrode a variable, added a child-only file, installed.
6. Simulated a user modification to one installed file, then a vendor template change to another, then ran `weftlo status` and `weftlo update`.

I did **not** execute the project-level override scenario (`consumer-b` with `.weftlo.yaml` overrides) — by that point the dogfood had produced more friction events than the synthesis phase needed, and the override system is doc-flagged for "hide from Quick Start" regardless.

### Phase 3 — Synthesis

Compiled the friction log, journey maps, comparison table, and recommendations below.

---

## 4. What went well

It's important to lead with this, because the impulse after reading the friction section will be "burn it down." That would be wrong.

- **The manifest-based update flow is excellent.** I simulated a realistic split — user edited one installed file by hand, vendor changed a different template — and `weftlo update` applied the vendor change to one file while preserving the user's edit on the other, with a clear `"use --force to overwrite"` hint. This is the actual value prop of the tool, and it is *working*.
- **`weftlo status` output is genuinely useful.** It correctly bucketed files into "Source changed" and "User modified," named the profile and its inheritance chain, and showed the install prefix. This is the kind of status output that earns trust.
- **Variable inheritance + deep merge works.** Child profile's `project` variable correctly overrode the parent's; parent's `company` variable was correctly inherited.
- **Architecture is clean.** Four layers, afero abstraction, ~75 test files including 6 integration suites. The internal quality is high.
- **The build is fast and dependency-light.** Go 1.25, cobra/afero/yaml/sprig/validator — five small dependencies. No surprises.
- **`profile create` and `install` both produce friendly, well-formatted CLI output** with "Next steps" hints — when they don't lie about where to put files.

The implementation isn't the problem.

---

## 5. Friction log — what fell down

Severity scale: 🔴 critical (blocks first-time success), 🟠 severe (causes confusion or wrong outcomes), 🟡 high (significant friction but recoverable), 🟢 low (polish).

| # | Severity | Event | Evidence |
|---|----------|-------|----------|
| F1 | 🔴 | `profile create` writes a layout the renderer doesn't read. Files placed at the profile root (where the generated README says to put them) are silently ignored. **`weftlo install` reports SUCCESS while producing zero files.** | Created `acme/base/CLAUDE.md.tmpl` at profile root per generated README's guidance → `install` exited 0, said "Installed profile 'acme/base' successfully!", but the only files in the project were `.weftlo.yaml` and `.weftlo.manifest.json`. No warning, no error. |
| F2 | 🔴 | The README that `profile create` generates says verbatim: *"Adding template files (.tmpl) to this directory"* — meaning the profile root. The renderer requires `content/`. The tool's own scaffolding contradicts the tool's architecture. | The literal text shown above is in the file created by `weftlo profile create acme/base`. The docs in `docs/architecture/overview.md` and the layout `weftlo init` creates both require `content/`. |
| F3 | 🟠 | `weftlo init` produces an empty-canvas default profile. After `init && install` the user has `./weftlo/README.md` — a copy of "This is the default profile. Place your content files in this directory." No working template, no variable example, no `.tmpl` demo. The Quick Start ends in a tutorial dead-end. | Inspected `~/.config/weftlo/profiles/default/default/content/README.md` after `init`. |
| F4 | 🟠 | Default `install_prefix: weftlo/` doesn't match the tool's pitch. The README says "configuration profiles for AI coding assistants" but the default lands files in `./weftlo/`, not `./.claude/` or any AI-assistant-meaningful location. The user has to learn `install_prefix` before the tool feels purposeful. | After `init && install`, files landed at `./weftlo/README.md`. |
| F5 | 🟡 | Doc/code drift: `docs/specs/profiles.md` (and other docs) use `extends:` for inheritance. The code reads `inherits_from:`. `profile create` correctly writes `inherits_from`, so the *generated* artifacts disagree with the *published* docs. | Confirmed in `internal/domain/profile/profile_config.go` (`yaml:"inherits_from,omitempty"`) and in the generated `acme/base/profile.yaml`. |
| F6 | 🟡 | `default_profile: default/default` introduces the `vendor/name` concept at the very first concept, with a duplicated namespace. New users immediately ask: "Do I need vendors for my own stuff?" | Default config written by `weftlo init`. |
| F7 | 🟢 | Inheritance produces a "Warning: Variable 'project' defined in 'acme/base', overridden by 'acme/backend'" message during normal install/update. That's the intended inheritance behavior, not an anomaly. | Output of `install`/`update` with child profile overriding a parent variable. |
| F8 | 🟢 | `weftlo version` reports `dev` for `go build` builds, with no way for an end user to verify they have the intended release. | Output of `./weftlo version` from a local build. |

### The story those events tell

The 60-second path goes like this:

1. User runs `weftlo init`. Friendly output, no complaints. ✅
2. User runs `weftlo install` in their project. Tool reports success and writes `./weftlo/README.md`. User shrugs — "OK, it copied a README." ❌ value not demonstrated (F3, F4).
3. User reads the docs, decides to create their own profile. Runs `weftlo profile create me/mine`. Reads the generated README, which says to add `.tmpl` files to the profile directory. ❌ misdirection (F2).
4. User adds `CLAUDE.md.tmpl` at profile root, runs `weftlo install --profile me/mine`. Tool reports success. **Project is empty.** ❌ silent no-op (F1).

That's the path between you and dogfooding. F1 + F2 are the killer pair: the scaffolding teaches you the wrong location, and the renderer punishes you with silence.

---

## 6. Journey maps — concepts to first joy

### Consumer (just wants their team's AI config)

| Step | Command | Concepts introduced |
|------|---------|---------------------|
| 1 | `weftlo init` (only if no profile is set globally) | profile, `vendor/name` |
| 2 | `weftlo install --profile org/something` | profile loading, install_prefix, manifest |
| 3 | `weftlo update` | source-changed vs user-modified, `--force` |

**Concept count to first joy: ~4.** Survivable. The consumer path is mostly opaque, which is the right design — they don't need to understand profiles to consume them.

**Worst step:** Step 2. Files land in `./weftlo/` instead of `./.claude/`. To fix this the user has to learn `install_prefix` (and which file to set it in: global `config.yaml`? project `.weftlo.yaml`?) before getting visible value.

### Vendor (publishes a profile for an org)

| Step | Command | Concepts introduced |
|------|---------|---------------------|
| 1 | `weftlo profile create org/base` | profile, `vendor/name`, profile-yaml schema |
| 2 | Place content in… `content/`? root? | **wrong path → silent failure (F1, F2)**, `.tmpl` suffix, partials (`_` prefix), `content/` directory requirement |
| 3 | Use variables in templates | Go template syntax, `.Variables`, sprig functions, three-level variable precedence |
| 4 | Inheritance | `inherits_from` (not `extends`), parent resolution, variable conflict semantics |
| 5 | Distribution | targets, `default_target`, four-tier override precedence, `target_overrides` for redirect/suppress |
| 6 | Templates with cross-references | `include`, `includeGlob`, `reference`, `referenceGlob`, `@` prefix |

**Concept count to first joy: ~15.** This is the cliff.

**Worst step:** Step 2. It is *the* defect in the experience: the tool's own scaffolding tells you to put files in a place the tool can't read, and the renderer fails silently when you do.

The consumer journey is fine. The vendor journey is what's blocking adoption — and the author is, by definition, a vendor in any dogfooding scenario. This is structurally why the author won't use the tool: every time they try, they hit the cliff.

---

## 7. "Why not just X?" — comparison for org-wide distribution

| Tool | Distribution model | Update story | Templating | Override granularity | Weftlo's edge |
|------|-------------------|--------------|------------|---------------------|---------------|
| **chezmoi** | Per-user dotfiles via git | Yes, sophisticated | Yes (Go templates) | Per-user | ❌ Different use case — chezmoi is per-user, not per-project |
| **copier** | Git-based project template | Yes (`copier update`, preserves edits) | Jinja2 | Per-answer | ⚠️ Nearly identical use case. Copier already does "vendor publishes, consumers update with edits preserved" |
| **cookiecutter** | Git/PyPI | None | Jinja2 | One-shot | ✅ Weftlo wins on update |
| **git submodule / subtree** | Git | Manual | None | None | ✅ Weftlo wins on templating + diff-aware update |
| **npm/pip package + postinstall** | Registry | Reinstall | Package-internal | None | ✅ Weftlo wins on declarative profile structure |
| **stow / bash script** | Files on disk | Manual | None | None | ✅ Weftlo wins on everything except simplicity |

**The honest one-sentence pitch:**

> *Weftlo is copier, for AI assistant config, with profile inheritance.*

That is the real differentiator: **profile inheritance** + **AI-assistant-specific defaults** + **multi-profile composition in one project**. Copier doesn't do inheritance, isn't aimed at this niche, and isn't a Go binary.

**The strategic risk:** if weftlo can't out-onboard copier, copier wins by default — Python is everywhere, and the moment-to-value of `pipx install copier && copier copy gh:vendor/profile .` is currently better than weftlo's. Weftlo's survival depends on its first-60-seconds experience being noticeably better than that. Today, it is not — but the fixes are localized and tractable.

---

## 8. Recommendations (ranked by ROI)

Items 1–4 alone would dramatically change the felt experience. The full list of 8 has been broken down into actionable handoff items in [`HANDOFF-UX-FIXES.md`](./HANDOFF-UX-FIXES.md).

1. **Fix `profile create` to mirror `init`'s layout.** Create `content/` and put the README inside it. Better still, drop a working `.tmpl` example inside.
2. **Make wrong-place template files a loud warning, not a silent no-op.** If `.tmpl` files exist outside `content/` (or `content/` is missing/empty), `install` should warn.
3. **Ship a non-empty default profile.** `weftlo init` should produce something that, after `init && install`, yields a visibly useful file (e.g. a templated `CLAUDE.md` using one variable).
4. **Change the default `install_prefix` to `.claude/`** (or auto-detect installed assistants). Match the pitch in the default.
5. **Reconcile `extends` vs `inherits_from`.** Update docs to match the code. Optionally accept `extends:` as a deprecated alias.
6. **Rewrite the Quick Start as a single copy-pasteable block** that ends with `cat .claude/CLAUDE.md` showing actual output.
7. **Hide the 4-tier override system from the Quick Start.** It belongs in an advanced doc.
8. **Demote "Warning: variable overridden"** to informational / `--verbose`-only output.

---

## 9. One-paragraph take

You built a well-architected tool with a genuinely valuable core feature — diff-aware updates that preserve user edits — and then wrapped it in a vendor-authoring experience that makes you the first victim. The reason you won't dogfood it isn't that the concepts are wrong; it's that the very first authoring command (`profile create`) produces a layout that silently doesn't work, with a generated README that confidently tells you to do the broken thing. Fix items 1–4 in [`HANDOFF-UX-FIXES.md`](./HANDOFF-UX-FIXES.md) and you'll *want* to use it. Everything else is polish.

---

## 10. Evaluation methodology notes (for future re-runs)

- **Total execution time:** under one hour, dominated by waiting on builds and re-runs.
- **Sandbox approach worked well.** `XDG_CONFIG_HOME` + `/tmp/weftlo-eval/` gave full isolation and zero cleanup risk. Recommend re-using for any future evaluation.
- **Following the docs verbatim** (rather than skipping ahead with implementation knowledge) was the single most important methodological choice. The silent no-op (F1) would not have surfaced without it — implementation knowledge would have caused me to write `content/` from the start.
- **What this evaluation didn't cover:** project-level overrides, target redirects/suppressions, `include`/`reference` cross-file dependencies, multi-profile composition, the ignore system, dry-run output formatting. A second pass focused on those would likely surface more friction, but none of it is on the critical path to first-60-seconds UX.
