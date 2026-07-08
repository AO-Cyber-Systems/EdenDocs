---
objective: 02-edendocs-rebrand-web-ui
trd: "01"
subsystem: ui
tags: [branding, html, m4, title-tag, upstream-patch]

# Dependency graph
requires:
  - objective: 01-from-source-build-pipeline
    provides: buildable eden-main checkout with a working browser/ tree
provides:
  - "Editor page static <title> reads 'EdenDocs' (browser/html/cool.html.m4)"
  - "Admin console static <title> reads 'EdenDocs - Admin console' (browser/admin/admintemplate.html)"
  - "Two isolated single-file upstream-patch commits, flagged for the Objective 5 divergence allowlist"
affects: [02-04-build-wiring, 02-05-trademark-checklist-verification, 05-divergence-allowlist]

# Tech tracking
tech-stack:
  added: []
  patterns: ["Upstream-file edits land as isolated, single-file, single-line commits for future rebase/divergence-allowlist tracking"]

key-files:
  created: []
  modified:
    - browser/html/cool.html.m4
    - browser/admin/admintemplate.html

key-decisions:
  - "Each upstream <title> patch committed alone (not batched) to keep the future Objective 5 divergence allowlist entries independently revertable/rebasable"

patterns-established:
  - "Isolated upstream-patch commit: one file, one line, dedicated feat(02-01) message, recorded for the divergence allowlist"

requirements-completed: []  # BRAND-02 intentionally NOT marked complete here — see Post-TRD Verification note; full completion depends on TRD 02-04's runtime/build wiring.

# Verification evidence
verification:
  gates_defined: 3
  gates_passed: 3
  auto_fix_cycles: 0
  tdd_evidence: false
  test_pairing: false

# Metrics
duration: 5min
completed: 2026-07-08
---

# Objective 2 TRD 01: Upstream Title Patches Summary

**Two single-line, single-file upstream patches replace the static `<title>` strings in the editor page (`cool.html.m4`) and admin console (`admintemplate.html`) with EdenDocs branding, each landed as its own isolated commit.**

## Performance

- **Duration:** 5 min
- **Started:** 2026-07-08T15:44:00Z
- **Completed:** 2026-07-08T15:45:34Z
- **Tasks:** 2
- **Files modified:** 2

## Accomplishments
- Editor tab now shows "EdenDocs" as the static (pre-JS) page title, eliminating the "Online Editor" flash before `main.js` overwrites `document.title`
- Admin console tab now shows "EdenDocs - Admin console" as the static page title, eliminating the "Collabora Online - Admin console" flash before `adminBody.html`'s brandProductName-driven rewrite runs
- Both patches committed independently (one file per commit) so each is a self-contained, revertable/rebasable entry for the Objective 5 divergence allowlist

## Task Evidence

| Task | Verify Command | Exit Code | Status |
|---|---|---|---|
| 1: Patch editor static title in cool.html.m4 | `grep -q '<title>EdenDocs</title>' browser/html/cool.html.m4 && ! grep -q 'Online Editor' browser/html/cool.html.m4 && git show --stat HEAD \| grep -q 'cool.html.m4' && [ "$(git show --name-only --pretty=format: HEAD \| grep -c .)" -eq 1 ]` | 0 | PASS |
| 2: Patch admin console static title in admintemplate.html | `grep -q '<title>EdenDocs - Admin console</title>' browser/admin/admintemplate.html && ! grep -qi 'collabora' browser/admin/admintemplate.html && [ "$(git show --name-only --pretty=format: HEAD \| grep -c .)" -eq 1 ]` | 0 | PASS |

## Task Commits

Each task was committed atomically:

1. **Task 1: Patch editor static title in cool.html.m4** - `3aa5befe6c5` (feat)
2. **Task 2: Patch admin console static title in admintemplate.html** - `d27976178bb` (feat)

**Plan metadata:** (recorded in the follow-up `docs(02-01)` commit per state_updates step)

_Note: Each commit's diff is exactly one changed line in exactly one file, confirmed via `git show --stat` and `git show --name-only`._

## Divergence allowlist candidates

For Objective 5's upstream-divergence allowlist (isolated, deliberate deviations from upstream Collabora Online source):

| File | Commit | Change |
|---|---|---|
| `browser/html/cool.html.m4` | `3aa5befe6c5` | `<title>Online Editor</title>` → `<title>EdenDocs</title>` (1 line) |
| `browser/admin/admintemplate.html` | `d27976178bb` | `<title>Collabora Online - Admin console</title>` → `<title>EdenDocs - Admin console</title>` (1 line) |

## Validation Gate Results

| Gate | Command | Exit Code | Status |
|---|---|---|---|
| lint | `bash -n scripts/eden/build.sh && true` (placeholder gate — no script changes in this TRD) | 0 | PASS |
| test | `grep -q '<title>EdenDocs</title>' browser/html/cool.html.m4 && grep -q '<title>EdenDocs - Admin console</title>' browser/admin/admintemplate.html` | 0 | PASS |
| build | `true` (dist-level proof deferred to TRD 02-04/02-05 CI on Linux; not buildable on this Mac) | 0 | PASS |

## Post-TRD Verification

- **Auto-fix cycles used:** 0
- **Must-haves verified:** 3/3 (both artifacts contain the expected `<title>` string; both upstream edits are isolated single-file commits)
- **Gate failures:** None
- **`requirements mark-complete` intentionally skipped:** Per TRD state_updates instructions, BRAND-02 is not marked fully complete by this TRD alone — the runtime title mechanisms (`browser/src/main.js:340`, `browser/admin/adminBody.html:2-3,11`) that also carry Collabora/brandProductName strings are in scope for TRD 02-04. This TRD covers only the two named static-title patches.

### Verification section from TRD (cross-check)

```
$ git log --oneline -- browser/html/cool.html.m4 | head -1
3aa5befe6c5 feat(02-01): EdenDocs editor tab title (isolated upstream patch 1/2)

$ git log --oneline -- browser/admin/admintemplate.html | head -1
d27976178bb feat(02-01): EdenDocs admin console tab title (isolated upstream patch 2/2)
```
Two different commits, each touching exactly one file — confirmed.

## Files Created/Modified
- `browser/html/cool.html.m4` - Static editor page `<title>` changed from "Online Editor" to "EdenDocs" (one line, immediately after the MPL header comment; comment left byte-identical)
- `browser/admin/admintemplate.html` - Static admin console `<title>` changed from "Collabora Online - Admin console" to "EdenDocs - Admin console" (one line, leading indentation preserved; zero remaining case-insensitive "collabora" occurrences in the file)

## Decisions Made
- Kept the two upstream-file edits as two fully independent commits (rather than one combined commit) specifically so each can be cherry-picked, reverted, or rebased on its own when Objective 5 builds the formal divergence allowlist.

## Deviations from Plan

None - TRD executed exactly as written. Line numbers matched the TRD's `<codebase_examples>` exactly (cool.html.m4:42, admintemplate.html:12), so no recovery/error-recovery paths were needed.

## Issues Encountered
None.

## User Setup Required

None - no external service configuration required.

## Next Objective Readiness
- TRD 02-02 (upstream trademark patches) and TRD 02-04 (build wiring, which also touches `main.js`/`adminBody.html` runtime title mechanisms) can proceed independently; no blockers introduced by this TRD.
- Divergence allowlist candidates recorded above are ready for Objective 5 to consume verbatim.

---
*Objective: 02-edendocs-rebrand-web-ui*
*Completed: 2026-07-08*
