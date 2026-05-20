# Weftlo UX Fixes — Handoff

> **Companion to:** [`EVALUATION-2026-05-20.md`](./EVALUATION-2026-05-20.md) — the full evaluation that produced these items.
> **Purpose:** Each fix below is designed to be picked up in a fresh context (new Claude conversation, new contributor, future self) and executed independently. Items 1–4 are the high-ROI fixes; 5–8 are polish.
> **Order matters loosely:** within each tier, items are roughly independent. Cross-references are noted where they exist.

---

## Decisions confirmed 2026-05-20

All five open implementation questions have been answered by the author. Each affected fix's "Approach" section below already reflects the chosen direction — no further author input is needed before starting work.

| # | Decision | Affects |
|---|----------|---------|
| 1 | Default `install_prefix` is **`.claude/`** (hardcoded). Auto-detection is a future enhancement, not this fix. | Fix 4, Fix 3 |
| 2 | Inheritance keyword: **docs follow code, use `inherits_from:`**. No alias, no deprecation period. | Fix 5 |
| 3 | Default profile content: **neutral hello-world `CLAUDE.md.tmpl` + one variable**. Single file. No partials, no includes, no `content.yaml`. | Fix 3 |
| 4 | Misplaced `.tmpl` files outside `content/`: **error by default**. Same check on `install` and `update`. | Fix 2 |
| 5 | Variable-override messages: **suppress single-chain inheritance overrides; keep multi-profile collisions loud**. Fall back to "all messages `--verbose`-only" if the existing tracker can't cheaply distinguish. | Fix 8 |

---

## How to use this document

Each fix has the same structure:

- **Problem** — what is broken, in one sentence.
- **Evidence** — concrete proof from the evaluation, with file paths and observed output.
- **Files to change** — the modules and tests that are likely involved.
- **Approach** — the recommended implementation direction, with the key design decisions called out.
- **Verification** — how to know it's done. Includes a manual sanity check and a test suggestion.
- **Notes** — dependencies on other items, caveats, things to ask the author about before committing to a direction.

When picking up a fix, **read its full entry first** before making changes — the design decisions in the Approach section often have alternatives worth thinking about.

---

## Tier 1 — High-ROI fixes (the cliff)

### Fix 1: `profile create` must produce a layout the renderer actually reads

**Severity:** 🔴 Critical

**Problem.** `weftlo profile create vendor/name` creates `profile.yaml` and `README.md` at the profile root, with no `content/` directory. The renderer only reads from `content/`. The result is that any user who follows the standard scaffolding ends up with a profile that produces nothing on install.

**Evidence.**

- `weftlo init` creates `profiles/default/default/content/README.md` — i.e. inside `content/`.
- `weftlo profile create acme/base` creates `profiles/acme/base/README.md` — i.e. at profile root, no `content/` dir.
- Result during dogfood: placed `CLAUDE.md.tmpl` at profile root per the generated README's guidance → `install` exit 0, "Installed profile 'acme/base' successfully!" → project contained only `.weftlo.yaml` and `.weftlo.manifest.json`.

**Files to change.**

- `internal/cli/profile_create.go` — the creation logic.
- `internal/cli/profile_create_test.go` — update / add tests for the new layout.
- Any fixture-based tests that assert on the current (broken) layout.

**Approach.**

1. Create `content/` directory at the profile root.
2. Move the generated README into `content/` — *or* (better) replace it with a working starter template, e.g. `content/CLAUDE.md.tmpl`:

   ```
   # {{ .Variables.company | default "Your Company" }} Standards

   This file was rendered by weftlo from profile `{{ .Profile }}`.
   Edit the template at `profile create`'s content/ directory to customize.
   ```

3. Leave a small `README.md` *at the profile root* (not inside `content/`) explaining the profile's purpose. This file is metadata about the profile, not content to be installed. **Do not let this README claim that `.tmpl` files go at the profile root** — see Fix 2.
4. Decide whether `content.yaml` should also be scaffolded (currently optional). Recommendation: don't, to keep the scaffolding minimal. Mention it in the root README as an optional next step.

**Verification.**

- Manual: `weftlo profile create demo/x && find $(weftlo's config dir)/profiles/demo/x` should show `content/` exists. Then `cd /tmp/empty && weftlo install --profile demo/x` should produce a rendered file in the install prefix.
- Test: add a `profile_create_test.go` case that asserts `content/` exists in the created tree and asserts a fresh install produces ≥1 file.
- Run: `go test ./internal/cli/... ./internal/app/install/...` to confirm nothing else regresses.

**Notes.**

- Pair with Fix 2 (the warning system) — they share the same goal of "the user can't accidentally put things in the wrong place." If both ship together, the fix is bulletproof.
- The root-README's *text* is fixed in Fix 5 area; the *location* (root vs `content/`) is fixed here.

---

### Fix 2: Silent no-op on misplaced templates must become a loud warning

**Severity:** 🔴 Critical

**Problem.** When a user places `.tmpl` files outside `content/`, `weftlo install` reports success and installs nothing, with no warning. This is the single worst failure mode in the tool.

**Evidence.** See Fix 1 evidence — `install` exits 0, prints "Installed profile 'acme/base' successfully!", and writes no rendered files.

**Files to change.**

- `internal/infrastructure/profile/enumeration.go` (or wherever content discovery walks the directory) — the walker currently scopes to `content/` only.
- `internal/app/install/service.go` — the place where "0 files rendered" is currently a silent success.
- `internal/cli/install.go` — surfacing the warning to the user.

**Approach.** *(direction confirmed 2026-05-20: error by default)*

Two complementary checks, both should ship:

1. **Pre-flight check at content enumeration — hard error.** When loading a profile, scan the profile root (one level deep, non-recursive) for `*.tmpl` files. If any are found, **abort the operation with a non-zero exit** and a clear message naming the misplaced files and the expected location, e.g.:

   ```
   error: profile 'acme/base' has 1 template file outside content/:
     - CLAUDE.md.tmpl
   Templates must live in profile's content/ directory. Move the file(s) and retry.
   ```

   Apply this check to **both** `install` and `update` code paths.

2. **Zero-output notice at install/update — informational.** If the operation would write zero files (excluding `.weftlo.yaml` and `.weftlo.manifest.json`), print a clear notice rather than silently succeeding:

   ```
   notice: no files were rendered from profile 'acme/base'. Check that templates exist in content/.
   ```

   This is a non-fatal notice (not an error) — an intentionally-empty profile is conceivable. The notice exists so the user is never confused about why their project is empty after a "successful" install.

**Verification.**

- Manual: create a profile with `CLAUDE.md.tmpl` at profile root, no `content/`. Run `install`. Expect: non-zero exit, clear error message naming the misplaced file and where it belongs. Repeat for `update`.
- Manual: create a profile with empty `content/` (no `.tmpl` files anywhere). Run `install`. Expect: exit 0 with a visible zero-output notice (not an error).
- Test: add integration tests under `internal/app/install/` (and equivalent for update) covering: (a) `.tmpl` at root → exits non-zero with the expected error text; (b) profile with zero renderable content → exits 0 with the zero-output notice; (c) valid profile with content in `content/` → behaves as before, no extra noise.

**Notes.**

- Land *with* Fix 1, ideally in the same PR. Fix 1 stops users from making the mistake; Fix 2 catches it when they make it anyway.

---

### Fix 3: Ship a non-empty, value-demonstrating default profile

**Severity:** 🟠 Severe

**Problem.** `weftlo init` ships a default profile containing only a README that says "Place your content files in this directory." After `init && install`, the user has `./weftlo/README.md` — a copy of that meaningless README. The Quick Start ends in a tutorial dead-end and the value proposition is never demonstrated.

**Evidence.** Inspected `~/.config/weftlo/profiles/default/default/content/README.md` after `weftlo init`. Its full contents:

```
# Default Profile

This is the default profile for weftlo.

This profile was created during initialization and can be customized to fit your needs.

Place your content files in this directory. They will be installed to your project
when you run 'weftlo install'.
```

After `weftlo install` in a fresh project, the user sees only `weftlo/README.md` containing the same text. No template demo, no variable demo, no rendered output.

**Files to change.**

- `internal/cli/init.go` — the function that materializes the default profile during `init`.
- Whatever fixture/embedded-content mechanism `init` uses to write the default profile (likely embedded strings, possibly `embed.FS`).
- `internal/cli/init_task4_test.go` and other init tests — update expectations.

**Approach.** *(direction confirmed 2026-05-20: neutral hello-world, single file)*

Replace the empty README with a **single-file** neutral hello-world that teaches the user where their AI rules live and that variables work. Resist the urge to demonstrate more concepts here — partials, includes, targets, `content.yaml` belong in advanced docs, not the default profile.

1. **One templated file: `content/CLAUDE.md.tmpl`.** Teaches structure (this is where AI rules go) and demonstrates one variable. Suggested content:

   ```markdown
   # Coding Standards for {{ .Variables.company | default "this project" }}

   <!--
   This file is rendered from your weftlo profile at:
     {{ .Profile }}

   Edit the template at that profile's content/CLAUDE.md.tmpl and
   re-run `weftlo update` to sync changes into your project.
   -->

   <!-- Add your coding standards, conventions, and AI-assistant instructions here. -->
   ```

   (Exact wording is the author's call; the key properties are: visible structure cue, visible variable interpolation, visible edit/update loop hint.)

2. **A `profile.yaml` with one variable + a comment showing where to add more:**

   ```yaml
   name: default/default
   variables:
     company: "Your Company"
     # Add your own variables here, then reference them in templates
     # as {{ .Variables.your_var_name }}.
   ```

3. **Do NOT scaffold:** partials, includes, `content.yaml`, additional `.tmpl` files, targets, or override examples. The default profile must only exercise concepts that appear in the Quick Start (post-Fix 6): profile, template file, variable, install prefix.

**Verification.**

- Manual: after `weftlo init && cd /tmp/empty && weftlo install`, `cat .claude/CLAUDE.md` shows the rendered file with `Your Company` substituted in. The user can immediately see "this is where my AI rules go" and where to edit the template to change it.
- Test: assert the default profile produces exactly one rendered file under the default install prefix, that the variable substitution worked, and that no `content.yaml` or partial files were scaffolded.

**Notes.**

- Pair with Fix 4 (default install prefix) so the rendered output lands somewhere the user recognizes as AI-config-related.
- Pair with Fix 6 (Quick Start rewrite) so the docs end with the now-visible payoff.

---

### Fix 4: Change the default `install_prefix` to `.claude/`

**Severity:** 🟠 Severe

**Problem.** The default `install_prefix` is `weftlo/`. The tool is pitched as managing config for AI coding assistants, but the default puts files in a directory none of those assistants read.

**Evidence.** After `weftlo init && weftlo install` in a fresh project, files land at `./weftlo/README.md`. The README.md at the repo root advertises the tool as for "AI coding assistants" — there is no AI assistant that reads `./weftlo/`.

**Files to change.**

- Wherever the default `install_prefix` value is set — search for `weftlo/` and the const/default in `internal/domain/config/` and `internal/infrastructure/config/`.
- Default config generated by `weftlo init` — `internal/cli/init.go`.
- Tests that assert on the default — likely several across `internal/app/install/` and `internal/cli/`.
- Docs: `docs/product/vision.md` Quick Start, `docs/architecture/overview.md`, `docs/reference/configuration.md`.

**Approach.** *(direction confirmed 2026-05-20: hardcode `.claude/`)*

1. Change the hardcoded default from `weftlo/` to `.claude/` wherever the constant lives.
2. Update the default `config.yaml` written by `init` to omit `install_prefix` (relying on the new default). If you'd rather make the default explicit, include it commented out with `.claude/` and a one-line note — author's call.
3. Mass-update tests. Most are straightforward search-and-replace.
4. Update docs (`docs/product/vision.md`, `docs/architecture/overview.md`, `docs/reference/configuration.md`) to reflect the new default.

**Out of scope (deliberately).** Auto-detection of installed assistants (`.cursor/`, `.aider.conf.yml`, `.github/copilot-instructions.md`, etc.) is a real future feature, but it adds magic, non-determinism, and design questions about precedence and multi-assistant conflicts that aren't worth resolving in this fix. If you want to file it for later, add a one-line note in the roadmap. **Do not implement it here.**

**Backward compatibility note for the changelog.** Existing projects with `install_prefix: weftlo/` explicitly set in `.weftlo.yaml` are unaffected — this changes the *default* only. No migration is required.

**Verification.**

- Manual: `weftlo init` then `weftlo install` in a fresh project produces files under `./.claude/`, not `./weftlo/`.
- Test: existing install tests should be updated; add at least one test asserting the new default explicitly.
- Run full suite: `go test ./...` to find any test that relied on the old default and needs updating.

**Notes.**

- Pair with Fix 3 (default profile content) — if the default profile produces a `CLAUDE.md` that lands in `.claude/CLAUDE.md`, the value prop is visible in the first 60 seconds.

---

## Tier 2 — Polish and consistency

### Fix 5: Reconcile `extends` (docs) vs `inherits_from` (code)

**Severity:** 🟡 High

**Problem.** Multiple user-facing docs show `extends:` as the inheritance key in `profile.yaml`. The code reads `inherits_from:`. Anyone copy-pasting from the docs will write a key the loader silently ignores.

**Evidence.**

- Code: `internal/domain/profile/profile_config.go` line 12 — `InheritsFrom string \`yaml:"inherits_from,omitempty"\``.
- Generated artifact: `weftlo profile create` writes `inherits_from:`.
- Drift: `docs/specs/profiles.md` and `docs/product/vision.md` show `extends:` in example YAML.

**Files to change.**

- `docs/specs/profiles.md`
- `docs/product/vision.md` (Key Concepts → Profiles section)
- `docs/architecture/overview.md`
- `docs/reference/configuration.md`

No code changes. No tests change.

**Approach.** *(direction confirmed 2026-05-20: docs follow code; no alias)*

Update every doc occurrence of `extends:` to `inherits_from:`. That's the whole fix. No code change, no UnmarshalYAML alias, no deprecation period — the tool is explicitly early-development per the README, the docs were the only place suggesting `extends:`, and `profile create` already writes `inherits_from:` correctly.

After the edit, grep the repo for `extends:` — it should appear zero times outside historical/changelog notes.

**Verification.**

- Grep the repo for `extends:` — should appear zero times except in changelog/historical notes.
- Manual: copy any example YAML from any doc into a real profile.yaml and confirm `weftlo install` honors the inheritance.

**Notes.**

- Independent of other fixes. Can be done in a single small PR.

---

### Fix 6: Rewrite the Quick Start as a single copy-pasteable block

**Severity:** 🟡 High

**Problem.** The Quick Start in `docs/product/vision.md` ends mid-thought ("Navigate to `~/.config/weftlo/profiles/mycompany/backend/content/` and add your configuration files") — i.e. it tells the user to invent something rather than handing them a working example.

**Evidence.** The current Quick Start (lines ~21–61 of `docs/product/vision.md`) has six numbered steps; step 3 requires the user to author content from scratch with no guidance on what to write or what the result should look like.

**Files to change.**

- `docs/product/vision.md` — Quick Start section.
- Possibly: `README.md` — sync the Quick Start excerpt.

**Approach.**

Replace the current six-step Quick Start with a single block the reader can paste:

```bash
# 1. Install
go install github.com/simensen/weftlo/cmd/weftlo@latest

# 2. Initialize (creates ~/.config/weftlo/ with a starter profile)
weftlo init

# 3. Install the starter profile into a project
cd ~/projects/my-app
weftlo install

# 4. See what got rendered
cat .claude/CLAUDE.md
```

The block ends with a `cat` showing the rendered output. This is the value moment — the user sees their template rendered with substituted variables, in a directory their AI assistant actually reads.

After the block, three short follow-on subsections:

- **Customize** — one paragraph + link to the templating reference.
- **Inherit from another profile** — one example block showing `inherits_from:` (post-Fix 5).
- **Use in multiple projects** — one paragraph showing `weftlo install --profile vendor/name` in a different project.

Cut everything else from the Quick Start. Concepts like targets, overrides, multi-profile composition, partial files, and the four-tier override system should be linked to advanced docs but **must not appear in the Quick Start** (see Fix 7).

**Verification.**

- A friend (or the author, or a fresh-context Claude) can follow the block end-to-end and arrive at a visible rendered file in under 60 seconds, no doc lookups required.
- Concept count required to follow the block: ideally ≤ 3 (profile, template, install prefix).

**Notes.**

- This fix is only meaningful after Fix 3 (default profile produces something to `cat`) and Fix 4 (default prefix is `.claude/`). Do those first.

---

### Fix 7: Move the 4-tier override system out of the Quick Start

**Severity:** 🟡 Medium-high

**Problem.** The override system has four precedence tiers (profile → global universal → global per-profile → project). It is genuinely powerful — and is also pure cognitive load for new users. Currently it surfaces in the Quick Start and overview docs at a point where users haven't yet succeeded with a single template.

**Evidence.** `docs/architecture/overview.md` (Routing System section) and `docs/product/vision.md` (Routing and Targets section) both introduce target overrides during the user's first encounter with the tool.

**Files to change.**

- `docs/product/vision.md` — remove or radically condense the "Routing and Targets" subsection.
- `docs/architecture/overview.md` — Routing System section can stay (it's architecture, not onboarding), but ensure no cross-link from Quick Start points to it.
- Possibly: create `docs/reference/overrides.md` or extend an existing reference doc as the canonical place for the full override semantics.

**Approach.**

1. In `vision.md`, replace "Routing and Targets" with a single short paragraph: *"By default, weftlo installs files to your project's install prefix (default: `.claude/`). For advanced routing — sending different files to different directories, redirecting at the project level, or suppressing files — see [the routing reference](../reference/overrides.md)."*
2. Move the four-tier precedence table, the `~`-suppress syntax, and the template-expressions-in-overrides feature into the reference doc.
3. Ensure the Quick Start (post-Fix 6) does not reference targets, target_overrides, default_target, or profile_overrides at all.

**Design decisions to flag.**

- This is a documentation reorganization, not a feature change. No code touched.
- Consider whether the *feature itself* should be made less prominent in `content.yaml` and `config.yaml` — e.g. should the default config not even mention `target_overrides:` in commented-out examples? Recommendation: **don't show it by default**. Comments in config files teach concepts; the default configs should only teach Quick-Start concepts.

**Verification.**

- After the change, no doc reachable in ≤ 1 hop from the README/Quick Start mentions "target override" or "default_target" or "profile_overrides".
- The reference doc has the full feature documented and is reachable from the configuration reference and the architecture overview.

**Notes.**

- Can ship independently of code changes.

---

### Fix 8: Demote "Warning: variable overridden" to informational output

**Severity:** 🟢 Low

**Problem.** When a child profile overrides a parent profile's variable — the *intended, normal* behavior of inheritance — `install` and `update` print "Warning: Variable 'X' defined in 'parent', overridden by 'child'". This trains users to ignore warnings, and frames working-as-designed as a problem.

**Evidence.** Captured during dogfood: setting `project: backend-service` in `acme/backend` (extending `acme/base` which set `project: default-project`) produced "Warning: Variable 'project' defined in 'acme/base', overridden by 'acme/backend'" on every install and update.

**Files to change.**

- `internal/infrastructure/profile/variable_merge.go` (or wherever conflict tracking emits to output) — search for "Warning" or `variableConflicts`.
- `internal/cli/variable_warnings.go` — likely the formatting/output code.
- `internal/cli/install.go`, `internal/cli/update.go`, `internal/cli/status.go` — wherever the warning is emitted.

**Approach.** *(direction confirmed 2026-05-20: distinguish inheritance from collision; fall back to `--verbose`-only if costly)*

Distinguish two cases and treat them differently:

1. **Single-chain override** (child overrides parent within the same inheritance chain): this is normal, intended behavior. **Suppress in default output.** Show only under `--verbose`.
2. **Multi-profile collision** (two unrelated profiles installed in the same project both define the same variable, via `LoadMergedMultiple`): genuinely ambiguous. **Keep the warning loud** in default output — this is real signal the user wants.

**First step before writing code:** inspect the existing tracker at `internal/domain/profile/variable_conflict.go` and `internal/infrastructure/profile/variable_merge.go` to see whether it already distinguishes these two cases (e.g. via the source profile path: same chain vs. different roots).

- **If yes:** the fix is just an output-formatter change — filter out single-chain conflicts from the default message stream; keep multi-profile collisions visible.
- **If no, and adding the distinction looks expensive:** fall back to the pragmatic alternative — move *all* override messages to `--verbose`-only output. This loses the multi-profile-collision signal in the default case, but eliminates the noise problem and is a smaller change. Multi-profile composition isn't yet a heavily-used flow, so this fallback is acceptable.

Pair with Fix 5 or Fix 6 (the doc PRs) for a small, low-risk PR if landing alone feels too thin.

**Verification.**

- Manual: `install` and `update` on a single inheritance chain produce no "Warning:" lines (unless `--verbose`).
- Test: existing tests that assert on the warning text need to be split — one for the now-suppressed inheritance case, one preserving the genuine-conflict warning.

**Notes.**

- Lowest priority. Skip if time-constrained.

---

## Suggested execution order

If working through these in fresh contexts, the recommended sequencing is:

1. **PR 1 — "Fix the cliff"** — Fixes 1 and 2 together. They share files and the same goal. After this PR, `profile create` produces a working layout *and* misplaced files produce a clear warning.
2. **PR 2 — "Make first-60-seconds visible"** — Fixes 3 and 4 together. New default profile content + new default install prefix. After this PR, `init && install` produces a visibly useful rendered file in `.claude/`.
3. **PR 3 — "Reconcile docs and quickstart"** — Fixes 5, 6, and 7. All doc work, no code (except Fix 5 if accepting `extends:` as an alias is chosen). After this PR, the docs match the code and the Quick Start ends with the user seeing a real rendered file.
4. **PR 4 — "Polish"** — Fix 8, plus any cleanup. Optional.

Items within a PR are independent enough that they can also ship as separate small PRs if the author prefers smaller review surfaces.

---

*All implementation directions were finalized 2026-05-20. See the "Decisions confirmed" table at the top of this document for the summary, or each fix's Approach section for the detail. No further author input is needed before starting work on any of these fixes.*
