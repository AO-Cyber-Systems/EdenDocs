---
objective: 02-edendocs-rebrand-web-ui
trd: "01"
type: standard
wave: 1
depends_on: []
files_modified:
  - browser/html/cool.html.m4
  - browser/admin/admintemplate.html
autonomous: true
requirements: [BRAND-02]

must_haves:
  truths:
    - "The editor page's static <title> is 'EdenDocs' — no flash of 'Online Editor' before JS runs, and 'EdenDocs' is the title if JS never runs"
    - "The admin console's static <title> is 'EdenDocs - Admin console' — no flash of 'Collabora Online - Admin console'"
    - "Each upstream-file edit is its own minimal commit touching exactly one file (divergence-allowlist discipline)"
  artifacts:
    - path: "browser/html/cool.html.m4"
      provides: "Patched editor static title"
      contains: "<title>EdenDocs</title>"
    - path: "browser/admin/admintemplate.html"
      provides: "Patched admin console static title"
      contains: "<title>EdenDocs - Admin console</title>"
  key_links:
    - from: "browser/html/cool.html.m4"
      to: "browser/dist/cool.html"
      via: "m4 build (browser/Makefile.am:1174-1197 lists cool.html.m4 as an explicit prerequisite — mtime change triggers rebuild)"
      pattern: "<title>EdenDocs</title>"
    - from: "browser/admin/admintemplate.html"
      to: "served admin console page (wsd/FileServer.cpp preprocessAdminFile)"
      via: "template load at request time"
      pattern: "<title>EdenDocs - Admin console</title>"
---

<objective>
Apply the two isolated upstream `<title>` patches named by BRAND-02: the editor
tab title in `browser/html/cool.html.m4` and the admin console tab title in
`browser/admin/admintemplate.html`. Each patch is a single-line, single-file,
own-commit change — the highest upstream-merge-risk items in this objective, so
they land first as trivially-rebasable diffs.

Purpose: The static titles are the no-JS fallback and the flash-of-wrong-title
fix. Runtime titles are already config-driven (`document.title = fileName + ' - '
+ window.brandProductName` in `browser/src/main.js:340`; adminBody.html rewrites
the admin title from `brandProductName`), so these two patches complete the
title surface once TRD 02-04 populates the config keys.

Output: Two one-line commits, each touching exactly one upstream file, both
flagged for the Objective 5 divergence allowlist.
</objective>

<execution_context>
@~/.claude/devflow/workflows/execute-trd.md
@~/.claude/devflow/templates/summary.md
</execution_context>

<embedded_context>

<codebase_examples>
Exact current state (verified 2026-07-08 on eden-main):

`browser/html/cool.html.m4:42` (immediately after the MPL header comment):
```html
<title>Online Editor</title>
```

`browser/admin/admintemplate.html:12`:
```html
    <title>Collabora Online - Admin console</title>
```

Runtime title mechanisms already in place (do NOT touch — config-driven, handled
by TRD 02-04):
- `browser/src/main.js:340` — `document.title = fileName + ' - ' + window.brandProductName`
- `browser/admin/adminBody.html:2-3,11` — `if (typeof brandProductName !== 'undefined') {l10nstrings.strProductName = brandProductName}` then `document.title = l10nstrings.strProductName + ' - ' + l10nstrings.strAdminConsole;`
</codebase_examples>

<anti_patterns>
- Do NOT batch both patches into one commit — the mergeability rule requires
  each upstream-file edit to be independently revertable/rebasable.
- Do NOT edit any other Collabora string encountered nearby (e.g. MPL header
  comments — legal attribution stays verbatim; other trademark strings belong
  to TRD 02-02/02-05).
- Do NOT rename files or touch m4 macro structure — single-line string
  replacement only.
</anti_patterns>

<error_recovery>
- If line numbers have drifted (upstream merge since research), locate by
  content: `grep -n '<title>Online Editor</title>' browser/html/cool.html.m4`
  and `grep -n '<title>Collabora Online - Admin console</title>' browser/admin/admintemplate.html`.
  If either string is absent entirely, STOP and re-verify against
  2-RESEARCH.md §3 before improvising — the surface may have moved.
- If a commit accidentally includes both files: `git reset --soft HEAD~1` and
  re-commit separately.
</error_recovery>

</embedded_context>

<context>
@.planning/PROJECT.md
@.planning/objectives/02-edendocs-rebrand-web-ui/OBJECTIVE.md
@.planning/objectives/02-edendocs-rebrand-web-ui/2-RESEARCH.md
</context>

<gotchas>
- HARD RULE: never use or reference port 8080 anywhere (local web checks: 8091;
  coolwsd native: 9980). This TRD needs no server at all — pure source edits.
- The m4 file's `<title>` line is plain literal HTML (no m4 token on that
  line) — safe single-line edit; `browser/Makefile.am`'s cool.html rule lists
  the .m4 as a prerequisite, so no cache-busting is needed (research §8).
- These commits will appear in the Objective 5 divergence allowlist. Record
  both file paths + commit hashes in this TRD's SUMMARY under a "Divergence
  allowlist candidates" heading so Objective 5 can harvest them.
</gotchas>

<tasks>

<task type="auto">
  <name>Task 1: Patch editor static title in cool.html.m4 (own commit)</name>
  <files>browser/html/cool.html.m4</files>
  <action>
Replace the single line `<title>Online Editor</title>` (line 42 at research
time) with `<title>EdenDocs</title>`. Change nothing else in the file — the
MPL header comment directly above stays byte-identical.

Commit this file ALONE with message
`feat(02-01): EdenDocs editor tab title (isolated upstream patch 1/2)`.
The commit diff must be exactly one changed line in exactly one file.
  </action>
  <verify>
grep -q '&lt;title&gt;EdenDocs&lt;/title&gt;' browser/html/cool.html.m4 &amp;&amp; ! grep -q 'Online Editor' browser/html/cool.html.m4 &amp;&amp; git show --stat HEAD | grep -q 'cool.html.m4' &amp;&amp; [ "$(git show --name-only --pretty=format: HEAD | grep -c .)" -eq 1 ]
  </verify>
  <done>cool.html.m4 contains `<title>EdenDocs</title>`, the old string is gone, and HEAD is a one-file commit touching only cool.html.m4.</done>
  <recovery>If the verify count shows extra files in the commit, `git reset --soft HEAD~1`, unstage extras, re-commit only cool.html.m4.</recovery>
</task>

<task type="auto">
  <name>Task 2: Patch admin console static title in admintemplate.html (own commit)</name>
  <files>browser/admin/admintemplate.html</files>
  <action>
Replace the single line `    <title>Collabora Online - Admin console</title>`
(line 12 at research time) with `    <title>EdenDocs - Admin console</title>`
(preserve the existing leading indentation exactly). Change nothing else.

Note: this is the pre-JS/no-JS title only. Once TRD 02-03's `branding.js` sets
`window.brandProductName = 'EdenDocs'`, adminBody.html rewrites the title at
runtime — this patch removes the flash of "Collabora Online" and covers the
JS-disabled case (research §3, Patch 2 nuance).

Commit this file ALONE with message
`feat(02-01): EdenDocs admin console tab title (isolated upstream patch 2/2)`.
  </action>
  <verify>
grep -q '&lt;title&gt;EdenDocs - Admin console&lt;/title&gt;' browser/admin/admintemplate.html &amp;&amp; ! grep -qi 'collabora' browser/admin/admintemplate.html &amp;&amp; [ "$(git show --name-only --pretty=format: HEAD | grep -c .)" -eq 1 ]
  </verify>
  <done>admintemplate.html contains `<title>EdenDocs - Admin console</title>` with zero remaining case-insensitive "collabora" occurrences in that file, committed alone.</done>
  <recovery>If the `! grep -qi collabora` leg fails, inspect the other hits: if they are legal/MPL attribution, leave them and narrow the check to the title line only, documenting why in the SUMMARY; if they are user-facing strings, they belong to TRD 02-02/02-05 scope — note them there, do not fix here.</recovery>
</task>

</tasks>

<validation_gates>
<lint>bash -n scripts/eden/build.sh &amp;&amp; true # no script changes in this TRD; placeholder gate</lint>
<test>grep -q '&lt;title&gt;EdenDocs&lt;/title&gt;' browser/html/cool.html.m4 &amp;&amp; grep -q '&lt;title&gt;EdenDocs - Admin console&lt;/title&gt;' browser/admin/admintemplate.html</test>
<build>true # dist-level proof runs in CI via TRD 02-04/02-05 (Linux build; not buildable on this Mac)</build>
</validation_gates>

<verification>
- Both patched strings present in source; old strings absent.
- `git log --oneline -- browser/html/cool.html.m4 | head -1` and
  `git log --oneline -- browser/admin/admintemplate.html | head -1` show two
  DIFFERENT commits, each touching exactly one file
  (`git show --name-only --pretty=format: <hash>` lists 1 file).
- Downstream (owned by later TRDs, not re-verified here): TRD 02-05's CI gate
  greps `browser/dist/cool.html` and the dist admin template for the EdenDocs
  titles, proving the m4/copy pipeline picked the patches up.
</verification>

<success_criteria>
BRAND-02's two named `<title>` patches exist as two isolated single-file
commits on eden-main, recorded in the SUMMARY as divergence-allowlist
candidates with commit hashes.
</success_criteria>

<output>
After completion, create `.planning/objectives/02-edendocs-rebrand-web-ui/02-01-SUMMARY.md`
including a "Divergence allowlist candidates" section listing both files + commit hashes.
</output>
