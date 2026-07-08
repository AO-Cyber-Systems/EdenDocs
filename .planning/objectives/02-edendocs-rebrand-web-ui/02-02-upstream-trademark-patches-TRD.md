---
objective: 02-edendocs-rebrand-web-ui
trd: "02"
type: standard
wave: 1
depends_on: []
files_modified:
  - browser/src/control/backstage/Sidebar.tsx
  - browser/admin/adminSettings.html
autonomous: true
requirements: [BRAND-05]

must_haves:
  truths:
    - "The always-visible Backstage (File) panel header reads 'EdenDocs', not 'Collabora Office'"
    - "The admin console Version tab headers carry no Collabora trademark ('EdenDocs (coolwsd)' / 'LibreOffice Core Engine')"
    - "Each upstream-file edit is its own minimal commit touching exactly one file (divergence-allowlist discipline)"
  artifacts:
    - path: "browser/src/control/backstage/Sidebar.tsx"
      provides: "Trademark-free backstage header title"
      contains: "EdenDocs"
    - path: "browser/admin/adminSettings.html"
      provides: "Trademark-free Version-tab headers"
      contains: "LibreOffice Core Engine"
  key_links:
    - from: "browser/src/control/backstage/Sidebar.tsx"
      to: "browser/dist/bundle.js"
      via: "TypeScript compile in browser/ Make pipeline"
      pattern: "backstage-header-title.*EdenDocs"
    - from: "browser/admin/adminSettings.html"
      to: "served admin Settings/Version tab"
      via: "admin dist copy"
      pattern: "EdenDocs (coolwsd)"
---

<objective>
Remove the two hardcoded Collabora trademark strings that research discovered
BEYOND the objective brief's two named `<title>` patches (2-RESEARCH.md §6,
Pitfall 4): the always-shown Backstage panel header "Collabora Office"
(`Sidebar.tsx`) and the admin console Version-tab headers "Collabora Online" /
"Collabora Office Engine" (`adminSettings.html`). Filed under BRAND-05
(trademark removal) per research Open Question 4's resolution.

Purpose: Without these two patches, the BRAND-05 dist sweep in TRD 02-05 will
find "Collabora" in `bundle.js` (compiled Sidebar.tsx) and in the served admin
settings page even after the title patches and branding wiring are complete.

Output: Two isolated single-file commits, flagged for the Objective 5
divergence allowlist.
</objective>

<execution_context>
@~/.claude/devflow/workflows/execute-trd.md
@~/.claude/devflow/templates/summary.md
</execution_context>

<embedded_context>

<codebase_examples>
Exact current state (verified 2026-07-08 on eden-main):

`browser/src/control/backstage/Sidebar.tsx:69-76` — the `header()` function in
`namespace BackstageTemplates`:
```tsx
export function header(props: HeaderProps): HTMLElement {
  return (
    <div class="backstage-header">
      <span class="backstage-header-title">Collabora Office</span>
      {props.isStarterMode ? null : closeButton(props.onClose)}
    </div>
  );
}
```
This is the always-shown "File" panel header (Info/Save/SaveAs/Print/Share/
Templates) — not gated behind any build flag (confirmed via
Control.BackstageView.ts, research §6.1). The small logo icon next to this text
comes from CSS (`backstage.css:56-58` → `url('images/collabora-office-white.svg')`)
and is replaced by TRD 02-03's same-filename image overwrite — do NOT touch the
CSS here.

`browser/admin/adminSettings.html:135-145` — the Version info tab:
```html
<div id="versionview" class="mtabs">
      <p>
        <h5><b>Collabora Online</b></h5>
        <div id="coolwsd-version"></div>
        <div id="coolwsd-buildconfig"></div>
      </p>
      <p>
        <h5><b>Collabora Office Engine</b></h5>
        <div id="lokit-buildconfig"></div>
      </p>
 </div>
```
Static markup — NOT `l10nstrings`-templated, does not respond to
`brandProductName` (research §4).
</codebase_examples>

<anti_patterns>
- Do NOT change element ids (`coolwsd-version`, `lokit-buildconfig`, etc.) —
  admin JS references them; internal identifiers stay upstream-compatible.
- Do NOT rename `collabora-office-white.svg` or edit `backstage.css` — the
  filename collision is the overwrite mechanism (TRD 02-03; research Pitfall 5).
- Do NOT claim the engine is an AO product — the engine is LibreOffice
  technology built by Collabora; "LibreOffice Core Engine" is technically
  accurate without using the Collabora mark (research §6.2 recommendation).
- Do NOT batch the two patches into one commit.
</anti_patterns>

<error_recovery>
- If line numbers drifted, locate by content:
  `grep -n 'Collabora Office' browser/src/control/backstage/Sidebar.tsx` and
  `grep -n 'Collabora' browser/admin/adminSettings.html`.
- If the Sidebar.tsx string has been refactored into a template/l10n call since
  research, adapt minimally: replace only the literal display string, never the
  JSX structure; note the drift in the SUMMARY.
</error_recovery>

</embedded_context>

<context>
@.planning/PROJECT.md
@.planning/objectives/02-edendocs-rebrand-web-ui/OBJECTIVE.md
@.planning/objectives/02-edendocs-rebrand-web-ui/2-RESEARCH.md
</context>

<gotchas>
- HARD RULE: never use or reference port 8080 (local web: 8091; coolwsd: 9980).
  No server needed for this TRD.
- `_('...')` l10n wrapping: the upstream string is NOT localized (plain JSX
  text). Keep the replacement plain too — introducing an l10n call would be a
  bigger diff and a new msgid, hurting mergeability.
- Record both commits as divergence-allowlist candidates in the SUMMARY.
</gotchas>

<tasks>

<task type="auto">
  <name>Task 1: Patch Backstage header "Collabora Office" → "EdenDocs" (own commit)</name>
  <files>browser/src/control/backstage/Sidebar.tsx</files>
  <action>
In `browser/src/control/backstage/Sidebar.tsx`, `header()` function (line ~72),
replace the JSX text `Collabora Office` with `EdenDocs`:

```tsx
<span class="backstage-header-title">EdenDocs</span>
```

One line, nothing else. The `::before` logo icon on this span is supplied by
CSS and becomes the gold AO emblem via TRD 02-03's image overwrite — together
they render emblem + "EdenDocs" wordmark in the backstage header, satisfying
the locked emblem+wordmark decision for this surface.

Commit this file ALONE with message
`feat(02-02): EdenDocs backstage header (isolated upstream patch, BRAND-05)`.
  </action>
  <verify>
grep -q 'backstage-header-title">EdenDocs' browser/src/control/backstage/Sidebar.tsx && ! grep -qi 'collabora' browser/src/control/backstage/Sidebar.tsx && [ "$(git show --name-only --pretty=format: HEAD | grep -c .)" -eq 1 ]
  </verify>
  <done>Sidebar.tsx renders "EdenDocs" in the backstage header, contains zero case-insensitive "collabora" occurrences, committed alone.</done>
  <recovery>If other "collabora" hits exist in the file (e.g. MPL/license comments), leave legal attribution verbatim and narrow the negative grep to non-comment lines; document in SUMMARY.</recovery>
</task>

<task type="auto">
  <name>Task 2: Patch admin Version-tab headers (own commit)</name>
  <files>browser/admin/adminSettings.html</files>
  <action>
In `browser/admin/adminSettings.html` Version tab (`#versionview`, lines
~137/142), replace exactly two strings:
- `<h5><b>Collabora Online</b></h5>` → `<h5><b>EdenDocs (coolwsd)</b></h5>`
- `<h5><b>Collabora Office Engine</b></h5>` → `<h5><b>LibreOffice Core Engine</b></h5>`

Keep `(coolwsd)` in the first header: the version/buildconfig divs below it
report the coolwsd daemon's version, and naming the internal daemon is
upstream-compatible and technically accurate (COOLWSD is a protocol/internal
identifier we intentionally keep, per the research §6 taxonomy).

Do not touch any element id, div, or the rest of the file.

Commit this file ALONE with message
`feat(02-02): trademark-free admin Version-tab headers (isolated upstream patch, BRAND-05)`.
  </action>
  <verify>
grep -q 'EdenDocs (coolwsd)' browser/admin/adminSettings.html && grep -q 'LibreOffice Core Engine' browser/admin/adminSettings.html && ! grep -qi 'collabora' browser/admin/adminSettings.html && [ "$(git show --name-only --pretty=format: HEAD | grep -c .)" -eq 1 ]
  </verify>
  <done>Both Version-tab headers are trademark-free, the file has zero case-insensitive "collabora" occurrences, committed alone.</done>
  <recovery>If other "collabora" occurrences exist elsewhere in adminSettings.html (research found only these two), classify them: legal attribution → leave + narrow check; user-facing trademark → patch in the SAME commit only if in this same file (still one-file commit), and record the extra surface in the SUMMARY for the 02-05 checklist.</recovery>
</task>

</tasks>

<validation_gates>
<lint>npx --prefix browser tsc --noEmit -p browser 2>/dev/null || true # best-effort local TS check; authoritative compile is the CI browser build</lint>
<test>grep -q 'EdenDocs' browser/src/control/backstage/Sidebar.tsx && grep -q 'LibreOffice Core Engine' browser/admin/adminSettings.html</test>
<build>true # compiled proof (bundle.js) lands in CI via TRD 02-04/02-05</build>
</validation_gates>

<verification>
- Both files contain the new strings and zero case-insensitive "collabora".
- Two separate one-file commits exist (`git show --name-only` on each).
- Downstream: TRD 02-05's dist sweep must find no "Collabora Office" in
  `browser/dist/bundle.js` attributable to Sidebar.tsx (research Pitfall 4's
  warning sign).
</verification>

<success_criteria>
The two research-discovered hardcoded trademark surfaces are patched as two
isolated single-file commits on eden-main, recorded as divergence-allowlist
candidates in the SUMMARY.
</success_criteria>

<output>
After completion, create `.planning/objectives/02-edendocs-rebrand-web-ui/02-02-SUMMARY.md`
including a "Divergence allowlist candidates" section listing both files + commit hashes.
</output>
