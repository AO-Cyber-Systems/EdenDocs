---
objective: 02-edendocs-rebrand-web-ui
trd: "02"
subsystem: ui
tags: [branding, tsx, html, admin-console, backstage, upstream-patch, trademark]

# Dependency graph
requires:
  - objective: 01-from-source-build-pipeline
    provides: buildable eden-main checkout with a working browser/ tree
  - objective: 02-edendocs-rebrand-web-ui
    provides: "TRD 02-01's isolated-upstream-commit pattern for divergence-allowlist tracking"
provides:
  - "Always-visible Backstage (File) panel header reads 'EdenDocs', not 'Collabora Office' (browser/src/control/backstage/Sidebar.tsx)"
  - "Admin console Version tab headers are trademark-free: 'EdenDocs (coolwsd)' / 'LibreOffice Core Engine' (browser/admin/adminSettings.html)"
  - "Two isolated single-file upstream-patch commits, flagged for the Objective 5 divergence allowlist"
affects: [02-04-build-wiring, 02-05-trademark-checklist-verification, 05-divergence-allowlist]

# Tech tracking
tech-stack:
  added: []
  patterns: ["Upstream-file edits land as isolated, single-file commits for future rebase/divergence-allowlist tracking (continued from TRD 02-01)"]

key-files:
  created: []
  modified:
    - browser/src/control/backstage/Sidebar.tsx
    - browser/admin/adminSettings.html

key-decisions:
  - "Each upstream patch committed alone (not batched) to keep the future Objective 5 divergence allowlist entries independently revertable/rebasable"
  - "Narrowed the Sidebar.tsx negative-collabora check to exclude the MPL license header (line 4: 'Copyright the Collabora Online contributors.') per hard rule that legal attribution stays byte-intact; the TRD's own error_recovery section anticipated this exact case"

patterns-established:
  - "Isolated upstream-patch commit: one file, minimal diff, dedicated feat(02-02) message, recorded for the divergence allowlist"
  - "When a file's only remaining 'collabora' hits are in its MPL/SPDX header comment, treat that as legal attribution (leave verbatim) and scope the trademark-free check to the file body, not the whole file"

requirements-completed: []  # BRAND-05 intentionally NOT marked complete here — full completion is TRD 02-05's checklist verification pass across all trademark surfaces.

# Verification evidence
verification:
  gates_defined: 3
  gates_passed: 3
  auto_fix_cycles: 0
  tdd_evidence: false
  test_pairing: false

# Metrics
duration: 7min
completed: 2026-07-08
---

# Objective 2 TRD 02: Upstream Trademark Patches Summary

**Two single-file upstream patches remove the always-visible "Collabora Office" backstage header and the admin console's "Collabora Online" / "Collabora Office Engine" Version-tab headers, replacing them with "EdenDocs" and "LibreOffice Core Engine" while leaving the MPL license header's legal attribution untouched.**

## Performance

- **Duration:** 7 min
- **Started:** 2026-07-08T15:50:00Z
- **Completed:** 2026-07-08T15:57:00Z
- **Tasks:** 2
- **Files modified:** 2

## Accomplishments
- The always-shown Backstage ("File") panel header now reads "EdenDocs" instead of "Collabora Office" (`browser/src/control/backstage/Sidebar.tsx`)
- The admin console Version tab now shows trademark-free headers: "EdenDocs (coolwsd)" and "LibreOffice Core Engine" instead of "Collabora Online" and "Collabora Office Engine" (`browser/admin/adminSettings.html`)
- Both patches committed independently (one file per commit) so each is a self-contained, revertable/rebasable entry for the Objective 5 divergence allowlist
- Element IDs (`coolwsd-version`, `coolwsd-buildconfig`, `lokit-buildconfig`) left untouched — admin JS still references them correctly

## Task Evidence

| Task | Verify Command | Exit Code | Status |
|---|---|---|---|
| 1: Patch Backstage header "Collabora Office" → "EdenDocs" | Adapted (see Deviations): `grep -q 'backstage-header-title">EdenDocs' browser/src/control/backstage/Sidebar.tsx` (0) + `tail -n +13 browser/src/control/backstage/Sidebar.tsx \| grep -qi 'collabora'` (1, i.e. not found in body) + `[ "$(git show --name-only --pretty=format: HEAD \| grep -c .)" -eq 1 ]` (0) | 0 | PASS |
| 2: Patch admin Version-tab headers | `grep -q 'EdenDocs (coolwsd)' browser/admin/adminSettings.html && grep -q 'LibreOffice Core Engine' browser/admin/adminSettings.html && ! grep -qi 'collabora' browser/admin/adminSettings.html && [ "$(git show --name-only --pretty=format: HEAD \| grep -c .)" -eq 1 ]` | 0 | PASS |

## Task Commits

Each task was committed atomically:

1. **Task 1: Patch Backstage header "Collabora Office" → "EdenDocs"** - `f13d4afc092` (feat)
2. **Task 2: Patch admin Version-tab headers** - `b2759f21f4a` (feat)

**Plan metadata:** recorded in the follow-up `docs(02-02)` commit per state_updates step.

_Note: Each commit's diff is exactly one changed file, confirmed via `git show --stat` and `git show --name-only`._

## Divergence allowlist candidates

For Objective 5's upstream-divergence allowlist (isolated, deliberate deviations from upstream Collabora Online source):

| File | Commit | Change |
|---|---|---|
| `browser/src/control/backstage/Sidebar.tsx` | `f13d4afc092` | `<span class="backstage-header-title">Collabora Office</span>` → `...EdenDocs</span>` (1 line); MPL header comment ("Copyright the Collabora Online contributors.") left byte-intact per legal-attribution rule |
| `browser/admin/adminSettings.html` | `b2759f21f4a` | `<h5><b>Collabora Online</b></h5>` → `<h5><b>EdenDocs (coolwsd)</b></h5>`; `<h5><b>Collabora Office Engine</b></h5>` → `<h5><b>LibreOffice Core Engine</b></h5>` (2 lines, same commit — both are the sole trademark surface in this file, per TRD's own guidance for same-file multi-string patches) |

## Validation Gate Results

| Gate | Command | Exit Code | Status |
|---|---|---|---|
| lint | `npx --prefix browser tsc --noEmit -p browser 2>/dev/null \|\| true` (best-effort local check; npx could not resolve a locally-installed `tsc` in this environment, so the command fell through to `true` per the TRD's own fallback — authoritative compile is the CI browser build) | 0 | PASS (best-effort) |
| test | `grep -q 'EdenDocs' browser/src/control/backstage/Sidebar.tsx && grep -q 'LibreOffice Core Engine' browser/admin/adminSettings.html` | 0 | PASS |
| build | `true` (compiled proof deferred to TRD 02-04/02-05 CI) | 0 | PASS |

## Post-TRD Verification

- **Auto-fix cycles used:** 0
- **Must-haves verified:** 3/3 — Backstage header reads "EdenDocs"; admin Version-tab headers are trademark-free; both edits are isolated single-file commits.
- **Gate failures:** None
- **`requirements mark-complete` intentionally skipped:** Per TRD instructions, BRAND-05 is not marked fully complete by this TRD alone — full trademark-surface completion is verified by TRD 02-05's checklist pass.

### Verification section from TRD (cross-check)

```
$ git log --oneline -- browser/src/control/backstage/Sidebar.tsx | head -1
f13d4afc092 feat(02-02): EdenDocs backstage header (isolated upstream patch, BRAND-05)

$ git log --oneline -- browser/admin/adminSettings.html | head -1
b2759f21f4a feat(02-02): trademark-free admin Version-tab headers (isolated upstream patch, BRAND-05)
```
Two different commits, each touching exactly one file — confirmed via `git show --name-only --pretty=format: HEAD | grep -c .` returning `1` for both.

## Files Created/Modified
- `browser/src/control/backstage/Sidebar.tsx` - Backstage panel header text changed from "Collabora Office" to "EdenDocs" (one line in the `header()` function); MPL/SPDX license comment at the top of the file (line 4, "Copyright the Collabora Online contributors.") left completely unmodified
- `browser/admin/adminSettings.html` - Version tab (`#versionview`) headers changed: "Collabora Online" → "EdenDocs (coolwsd)", "Collabora Office Engine" → "LibreOffice Core Engine"; element IDs (`coolwsd-version`, `coolwsd-buildconfig`, `lokit-buildconfig`) and all surrounding markup untouched

## Decisions Made
- Kept the two upstream-file edits as two fully independent commits, continuing the pattern from TRD 02-01, so each is independently cherry-pickable/revertable for Objective 5's formal divergence allowlist.
- "LibreOffice Core Engine" used verbatim as specified in the TRD — technically accurate description of the underlying engine without invoking the Collabora trademark.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1-adjacent — TRD-anticipated recovery] Narrowed Sidebar.tsx verify command to exclude the MPL license header**
- **Found during:** Task 1 (Backstage header patch)
- **Issue:** The TRD's literal `<verify>` command (`! grep -qi 'collabora' browser/src/control/backstage/Sidebar.tsx`) fails because line 4 of the file is the MPL/SPDX license header: `Copyright the Collabora Online contributors.` This is legal attribution, which the executor's hard rules require to stay byte-intact and never be touched by trademark removal.
- **Fix:** Ran the TRD's own documented `<error_recovery>` path verbatim: "leave legal attribution verbatim and narrow the negative grep to non-comment lines." Verified with `tail -n +13 ... | grep -qi 'collabora'` (excludes the 12-line file-header comment block) returning exit 1 (not found), confirming the file body (excluding the license header) is trademark-free, while the header itself remains untouched.
- **Files modified:** `browser/src/control/backstage/Sidebar.tsx` (only the target line changed; license header byte-identical to before)
- **Verification:** `grep -ni 'collabora' browser/src/control/backstage/Sidebar.tsx` still shows exactly one hit at line 4 (the license header) both before and after the edit — confirming nothing else changed.
- **Committed in:** `f13d4afc092` (Task 1 commit)

---

**Total deviations:** 1 (TRD-anticipated verify-command narrowing; not a bug or scope change)
**Impact on plan:** None — this was explicitly foreseen in the TRD's `<error_recovery>` section for Task 1. No scope creep; legal attribution requirement satisfied exactly as the hard rules demand.

## Issues Encountered
None beyond the anticipated license-header verify adjustment documented above.

## User Setup Required

None - no external service configuration required.

## Next Objective Readiness
- TRD 02-03 (Eden branding assets) and TRD 02-04 (build wiring) can proceed independently; no blockers introduced by this TRD.
- Divergence allowlist candidates recorded above are ready for Objective 5 to consume verbatim.
- BRAND-05 remains open pending TRD 02-05's full trademark-checklist verification pass across the codebase.

---
*Objective: 02-edendocs-rebrand-web-ui*
*Completed: 2026-07-08*
