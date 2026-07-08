---
objective: 02-edendocs-rebrand-web-ui
trd: "05"
type: standard
wave: 3
depends_on: ["02-04"]
files_modified:
  - docs/eden/TRADEMARK-MPL-CHECKLIST.md
  - scripts/eden/trademark-exceptions.txt
  - scripts/eden/verify-branding.sh
  - .github/workflows/build.yml
autonomous: true
requirements: [BRAND-05, BRAND-06]

must_haves:
  truths:
    - "A written trademark/MPL checklist exists covering every surface class from the research taxonomy, with per-surface status and evidence, and every accepted exception explicitly justified"
    - "A mechanical CI gate (verify-branding.sh) sweeps the BUILT browser/dist for Collabora trademarks against the exceptions file and fails the build on any unexplained hit"
    - "MPL/legal attribution is proven untouched: COPYING, THIRDPARTYLICENSES, CODA-THIRDPARTYLICENSES.html byte-identical to their committed upstream state"
    - "BRAND-06 is a mechanical gate, not a claim: the four phone-home flags are asserted absent from build.sh and the brandProductName canary is asserted present, in CI, on every push"
  artifacts:
    - path: "docs/eden/TRADEMARK-MPL-CHECKLIST.md"
      provides: "The written BRAND-05 checklist (taxonomy, per-surface rows, accepted exceptions, MPL preservation section)"
      contains: "Accepted exceptions"
    - path: "scripts/eden/trademark-exceptions.txt"
      provides: "Fixed-string allowlist consumed by the sweep (every line justified by a checklist row)"
      contains: "sdk.collaboraonline.com"
    - path: "scripts/eden/verify-branding.sh"
      provides: "Mechanical BRAND-01..06 verification gate (static dist checks; runtime checks live in smoke-test.sh)"
      contains: "trademark-exceptions.txt"
    - path: ".github/workflows/build.yml"
      provides: "CI step running verify-branding.sh after the smoke test"
      contains: "verify-branding.sh"
  key_links:
    - from: ".github/workflows/build.yml"
      to: "scripts/eden/verify-branding.sh"
      via: "workflow step after smoke test (thin-orchestrator pattern from 01-02)"
      pattern: "verify-branding\\.sh"
    - from: "scripts/eden/verify-branding.sh"
      to: "scripts/eden/trademark-exceptions.txt"
      via: "grep -viFf exceptions filter over the dist sweep"
      pattern: "grep -viFf"
    - from: "scripts/eden/trademark-exceptions.txt"
      to: "docs/eden/TRADEMARK-MPL-CHECKLIST.md"
      via: "every exception line has a justifying checklist row (manual invariant, asserted in checklist text)"
      pattern: "sdk.collaboraonline.com"
---

<objective>
Author the written trademark/MPL checklist that BRAND-05 requires, and turn
BRAND-05/BRAND-06 verification into a mechanical CI gate:
`scripts/eden/verify-branding.sh` sweeps the built `browser/dist` for Collabora
marks (filtered by an explicit, justified exceptions file), asserts MPL/legal
files untouched, and asserts the no-phone-home flag set — wired as a build.yml
step so every future push re-proves the rebrand.

Purpose: This is the task that actually PROVES BRAND-05 and makes BRAND-06's
proof permanent (research Implication 8+9). It extends the Objective 1 CI
pipeline (thin-orchestrator pattern), never forks it.

Output: Checklist doc, exceptions file, verify script, one new CI step, and a
green run whose logs show the gate passing.
</objective>

<file_tree>
docs/
└── eden/
    └── TRADEMARK-MPL-CHECKLIST.md   ← CREATE
scripts/eden/
├── trademark-exceptions.txt          ← CREATE
└── verify-branding.sh                ← CREATE
.github/workflows/
└── build.yml                         ← MODIFY (one added step)
</file_tree>

<execution_context>
@~/.claude/devflow/workflows/execute-trd.md
@~/.claude/devflow/templates/summary.md
</execution_context>

<embedded_context>

<codebase_examples>
build.yml's existing final step (add the new step directly AFTER it, same
thin-orchestrator style — zero inline logic in the workflow):
```yaml
      # BUILD-04: native port 9980 — port 8080 is banned project-wide.
      - name: Smoke test /hosting/discovery + /hosting/capabilities on 9980
        run: ./scripts/eden/smoke-test.sh
```

Loud-assertion script style to follow (project convention from 01-02:
"silent set -e deaths are banned"):
```bash
test -f browser/dist/branding.css || { echo "ERROR: branding.css missing from dist (BRAND-01)"; exit 1; }
```

Research §6 taxonomy classes for the checklist backbone:
1. User-facing branding — MUST change (titles, Sidebar.tsx header,
   adminSettings version headers, collabora-office-white.svg content,
   favicon, welcome page)
2. Default config values pointing at live Collabora infra — MUST change
   (help_url via --with-help-url=https://aocyber.ai; brandProductURL injection)
3. Internal protocol identifiers — MUST NOT change (COOLWSD agent strings,
   cool.html/coolwsd.xml filenames, C++ names)
4. Third-party technical documentation links — accepted exception v1
   (sdk/forum/help/gerrit domains across ~9 files; revisit when Eden publishes
   integrator docs)
5. Legal attribution — MUST preserve verbatim (COPYING, THIRDPARTYLICENSES,
   CODA-THIRDPARTYLICENSES.html, per-file MPL headers)
6. Judgment calls — flagged, not auto-fixed (Gerrit provenance links in
   Control.AboutDialog.ts / AdminSocketSettings.js — kept for v1)
</codebase_examples>

<anti_patterns>
- No silent exceptions: every line in trademark-exceptions.txt MUST have a
  justifying row in the checklist. An exceptions file that grows without
  checklist rows is the failure mode this TRD exists to prevent.
- Do NOT "fix" category-3 internal identifiers or category-5 legal files to
  make the sweep pass — the sweep's include-globs and exceptions are the right
  knobs.
- Do NOT put sweep logic inline in build.yml — thin orchestrator, logic in
  scripts/eden/ (01-02 pattern).
- Do NOT weaken a failing gate to go green — every relaxation is an exceptions
  line + checklist row, reviewed and justified.
</anti_patterns>

<error_recovery>
- The dist sweep can ONLY be tuned against a real built tree, which exists in
  CI (Linux build; this Mac does not build dist). Expect 1-2 tuning cycles:
  push, read the gate's survivor output from the CI log, classify each
  survivor (fix vs exception+justification), commit, re-push. Budget ≤4 cycles
  (01-02 discipline); the script must PRINT every surviving hit so a red run
  is self-diagnosing.
- Known-expected survivors to pre-seed (research §6 + compiled fallbacks):
  sdk.collaboraonline.com, forum.collaboraonline.com, help.collaboraoffice.com,
  www.collaboraonline.com, gerrit.collaboraoffice.com, github.com/CollaboraOnline,
  "Collabora Online Development Edition (unbranded)" (unreached fallback
  literal compiled into bundle.js — never rendered once brandProductName is
  set), collabora-office-white.svg (intentional filename collision, content is
  Eden gold).
- If `grep -viFf` behaves oddly under the environment's ugrep wrapper, use
  `command grep` explicitly inside the script (01-03 lesson).
</error_recovery>

</embedded_context>

<context>
@.planning/PROJECT.md
@.planning/objectives/02-edendocs-rebrand-web-ui/OBJECTIVE.md
@.planning/objectives/02-edendocs-rebrand-web-ui/2-RESEARCH.md
@.planning/objectives/01-from-source-build-pipeline/01-02-SUMMARY.md
</context>

<research_context>
- Verified-clean surfaces to record as such (no action): cool-help.html
  ("Collaborative" false positives only), discovery.xml, docker/from-source
  Dockerfiles (zero LABELs — container LABEL work belongs to Objective 4),
  net/Socket.cpp COOLWSD strings (deferred BRAND-07, v2).
- The l10n translation catalogs (browser/dist/l10n/*.json) may carry upstream
  fallback strings in translations; the sweep's include set is
  *.html/*.js/*.css only — record l10n as a reviewed v1 exception row.
- BRAND-06 flag audit: FEEDBACK_URL/INFOBAR_URL default empty; support-key and
  welcome-url actively disable code paths when unset (research §7). The
  mechanical proof = negative greps on build.sh + the brandProductName canary
  (which only exists when welcome-url is off — one grep proves both halves).
</research_context>

<gotchas>
- HARD RULE: never use or reference port 8080 (local web: 8091; coolwsd: 9980).
  verify-branding.sh is STATIC-only (no server, no ports) — runtime checks
  already live in smoke-test.sh (TRD 02-04).
- build.yml edits: additive single step only; keep concurrency/cache blocks
  untouched; YAML-lint before pushing
  (`python3 -c "import yaml; yaml.safe_load(open('.github/workflows/build.yml'))"`).
- The negative flag greps work because 02-04 wrote build.sh comments naming
  flags WITHOUT leading dashes — do not add a literal `--with-welcome-url`
  string to build.sh in this TRD either (verify-branding.sh may contain the
  literal strings it greps FOR; it greps build.sh, not itself).
- MPL check must be a diff against git HEAD state (`git diff --quiet HEAD --`),
  not just existence — "intact" means byte-identical.
</gotchas>

<tasks>

<task type="auto">
  <name>Task 1: Author docs/eden/TRADEMARK-MPL-CHECKLIST.md + scripts/eden/trademark-exceptions.txt</name>
  <files>docs/eden/TRADEMARK-MPL-CHECKLIST.md, scripts/eden/trademark-exceptions.txt</files>
  <action>
Create `docs/eden/TRADEMARK-MPL-CHECKLIST.md` with these sections:
1. **Purpose + scope** — BRAND-05's written checklist; verified mechanically by
   scripts/eden/verify-branding.sh in CI on every push.
2. **Surface taxonomy** — the 6-class table from embedded_context, each class
   with its action rule.
3. **Per-surface checklist** — one row per concrete surface with columns
   Surface | Class | Action taken | Where fixed | Evidence. Rows: editor title
   (02-01), admin title (02-01), backstage header string (02-02), admin
   Version-tab headers (02-02), backstage logo SVG content (02-03 collision
   overwrite), favicon (02-03/02-04 overlay), welcome page (02-03/02-04 full
   replacement), color palette (02-03 branding.css), help_url
   (02-04 --with-help-url=https://aocyber.ai), brandProductName/URL keys
   (02-04 injection), About dialog product name + vendor (config-driven +
   --with-vendor), container LABELs (none exist in from-source Dockerfiles —
   verified clean; Objective 4 owns future LABELs).
4. **Accepted exceptions (v1)** — one row per trademark-exceptions.txt line
   with justification and revisit trigger: the 5 third-party doc/community
   domains (+ github.com/CollaboraOnline links), the unreached
   "(unbranded)" fallback literal in compiled bundles, the intentional
   collabora-office-white.svg FILENAME, l10n catalogs (outside sweep globs),
   Gerrit provenance links (judgment call, kept for v1). State the invariant:
   "every exceptions-file line has a row here; additions require both."
5. **MPL/legal preservation** — COPYING, THIRDPARTYLICENSES,
   CODA-THIRDPARTYLICENSES.html byte-identical (mechanically asserted);
   README.FILENOTICES.md convention followed for new eden files; MPL
   source-availability link = https://github.com/AO-Cyber-Systems/EdenDocs
   (welcome page footer + this doc).
6. **No-phone-home (BRAND-06)** — the four never-passed flags, the
   welcome-url↔brand-keys dependency, and pointer to the mechanical gates.
7. **Divergence allowlist candidates** — the 4 patch files from 02-01/02-02 +
   the favicon build-time overlay remark (for Objective 5 to harvest).

Create `scripts/eden/trademark-exceptions.txt`: pure fixed-string lines (no
comments — grep -F treats every line literally), pre-seeded per error_recovery.

Commit: `docs(02-05): trademark/MPL checklist + sweep exceptions (BRAND-05)`.
  </action>
  <verify>
test -f docs/eden/TRADEMARK-MPL-CHECKLIST.md &amp;&amp; grep -q 'Accepted exceptions' docs/eden/TRADEMARK-MPL-CHECKLIST.md &amp;&amp; grep -q 'THIRDPARTYLICENSES' docs/eden/TRADEMARK-MPL-CHECKLIST.md &amp;&amp; grep -q 'github.com/AO-Cyber-Systems/EdenDocs' docs/eden/TRADEMARK-MPL-CHECKLIST.md &amp;&amp; grep -q 'sdk.collaboraonline.com' scripts/eden/trademark-exceptions.txt &amp;&amp; grep -q 'Collabora Online Development Edition (unbranded)' scripts/eden/trademark-exceptions.txt &amp;&amp; grep -q 'collabora-office-white.svg' scripts/eden/trademark-exceptions.txt &amp;&amp; [ "$(grep -c . scripts/eden/trademark-exceptions.txt)" -ge 6 ] &amp;&amp; ! grep -q '^#' scripts/eden/trademark-exceptions.txt
  </verify>
  <done>Checklist covers taxonomy + per-surface rows + justified exceptions + MPL section + BRAND-06 section; exceptions file is comment-free fixed strings, each mirrored by a checklist row.</done>
  <recovery>Documentation-only — re-edit and re-verify.</recovery>
</task>

<task type="auto">
  <name>Task 2: scripts/eden/verify-branding.sh + build.yml gate step</name>
  <files>scripts/eden/verify-branding.sh, .github/workflows/build.yml</files>
  <action>
Create `scripts/eden/verify-branding.sh` (`set -euo pipefail`, MPL header, cd
to repo root like sibling scripts, `command grep` throughout, loud ERROR echo
on every assertion). Gates, in order:

1. BRAND-01 dist artifacts: `browser/dist/branding.css`,
   `browser/dist/branding.js`, `browser/dist/images/toolbar-bg.svg` exist;
   `command grep -qi 'efb32c' browser/dist/images/collabora-office-white.svg`
   (gold gradient proves the Eden overwrite, upstream glyph is white).
2. BRAND-02 titles/vendor in dist: `<title>EdenDocs</title>` in
   `browser/dist/cool.html`; locate the admin template under browser/dist
   (`find browser/dist -name 'admintemplate.html'` — expected at
   browser/dist/admin/admintemplate.html) and grep
   `<title>EdenDocs - Admin console</title>`; `AO Cyber Systems` present in
   browser/dist/cool.html.
3. BRAND-03: `command grep -q "window.brandProductName" browser/dist/branding.js`
   and the value EdenDocs.
4. BRAND-04: `command grep -qF -- '--color-primary: #D4A853 !important' browser/dist/branding.css`;
   welcome clean: `! command grep -rqi collabora browser/dist/welcome/`.
5. BRAND-05 sweep (the core gate):
   ```bash
   SURVIVORS=$(command grep -rni 'collabora' browser/dist \
     --include='*.html' --include='*.js' --include='*.css' \
     | command grep -viFf scripts/eden/trademark-exceptions.txt || true)
   [ -z "$SURVIVORS" ] || { echo "ERROR: unexplained Collabora trademark hits in browser/dist:"; echo "$SURVIVORS"; echo "Fix the surface or add a JUSTIFIED exception (checklist row required)."; exit 1; }
   ```
6. BRAND-05 legal: `git diff --quiet HEAD -- COPYING THIRDPARTYLICENSES CODA-THIRDPARTYLICENSES.html`
   plus `command grep -q 'Mozilla Public' COPYING`.
7. BRAND-06 mechanical gate: for each of the four forbidden flags assert
   `! command grep -q -- "--<flag>" scripts/eden/build.sh`; then the canary:
   `[ -f coolwsd.xml ] && command grep -q '<brandProductName' coolwsd.xml`
   with a loud error pointing at 2-RESEARCH.md Pitfall 1 (in CI this always
   runs post-build so the file exists; locally, skip-with-warning if absent).
Final line: `echo "BRANDING VERIFICATION PASSED (BRAND-01..06)"`.

Edit `.github/workflows/build.yml`: add ONE step after the smoke-test step:
```yaml
      # BRAND-01..06: mechanical rebrand gate — static dist sweep + MPL +
      # no-phone-home assertions (runtime checks live in smoke-test.sh).
      - name: Verify EdenDocs branding + trademark/MPL gates
        run: ./scripts/eden/verify-branding.sh
```
YAML-lint the workflow. `chmod +x scripts/eden/verify-branding.sh`.
Commit: `feat(02-05): verify-branding.sh mechanical gate + CI step (BRAND-05/06)`.
  </action>
  <verify>
bash -n scripts/eden/verify-branding.sh &amp;&amp; test -x scripts/eden/verify-branding.sh &amp;&amp; grep -q 'trademark-exceptions.txt' scripts/eden/verify-branding.sh &amp;&amp; grep -q 'git diff --quiet HEAD -- COPYING' scripts/eden/verify-branding.sh &amp;&amp; grep -qF -- '--with-welcome-url' scripts/eden/verify-branding.sh &amp;&amp; grep -q 'brandProductName' scripts/eden/verify-branding.sh &amp;&amp; python3 -c "import yaml; yaml.safe_load(open('.github/workflows/build.yml'))" &amp;&amp; grep -q 'verify-branding.sh' .github/workflows/build.yml &amp;&amp; ! grep -q '8080' scripts/eden/verify-branding.sh
  </verify>
  <done>verify-branding.sh syntax-checks, is executable, contains all 7 gate groups, no port references; build.yml parses and invokes it as a thin step after the smoke test.</done>
  <recovery>If YAML indentation breaks the workflow, restore from git and re-apply as a minimal single-step insertion; never restructure existing steps.</recovery>
</task>

<task type="auto">
  <name>Task 3: Drive the gate green in CI and record checklist evidence</name>
  <files>docs/eden/TRADEMARK-MPL-CHECKLIST.md, scripts/eden/trademark-exceptions.txt</files>
  <action>
Push and watch build.yml. Expected: 1-2 sweep-tuning cycles (the exceptions
file can only be finalized against the real Linux dist — see error_recovery).
For every survivor the gate prints: either FIX the surface (if it is a genuine
missed trademark — coordinate scope: source patches belong to new isolated
commits, flagged in SUMMARY for the allowlist) or ADD the exception line + its
justifying checklist row in the SAME commit. Never one without the other.

When green: update the checklist's per-surface Evidence column with the run
ID/URL and the `BRANDING VERIFICATION PASSED` log line; also record the
`BRANDING SMOKE PASSED` line from the same run (02-04's runtime half). Verify
locally that COPYING/THIRDPARTYLICENSES/CODA-THIRDPARTYLICENSES.html show no
diff (`git diff --stat HEAD -- COPYING THIRDPARTYLICENSES CODA-THIRDPARTYLICENSES.html`
is empty). Final commit:
`docs(02-05): checklist evidence from green CI run (BRAND-05 verified)`.
  </action>
  <verify>
gh run list --workflow=build.yml --branch eden-main --limit 1 --json conclusion --jq '.[0].conclusion' | grep -q success &amp;&amp; grep -qE 'runs/[0-9]+' docs/eden/TRADEMARK-MPL-CHECKLIST.md &amp;&amp; git diff --quiet HEAD -- COPYING THIRDPARTYLICENSES CODA-THIRDPARTYLICENSES.html
  </verify>
  <done>Latest eden-main run is green WITH the verify-branding step passing; checklist carries the run evidence; legal files are diff-clean.</done>
  <recovery>If a survivor is a genuinely new upstream surface (not in research), classify it via the taxonomy before acting; if it needs an upstream-file patch, make it an isolated one-file commit and add it to the checklist's divergence-allowlist section. If >4 CI cycles loom, STOP and surface findings to the user.</recovery>
</task>

</tasks>

<validation_gates>
<lint>bash -n scripts/eden/verify-branding.sh &amp;&amp; python3 -c "import yaml; yaml.safe_load(open('.github/workflows/build.yml'))"</lint>
<test>! grep -rn '8080' scripts/eden/verify-branding.sh docs/eden/TRADEMARK-MPL-CHECKLIST.md | grep -vi 'never\|banned\|prohibit\|do not' | grep -q .</test>
<build>gh run list --workflow=build.yml --branch eden-main --limit 1 --json conclusion --jq '.[0].conclusion' | grep -q success</build>
</validation_gates>

<verification>
- Checklist + exceptions + verify script + CI step all exist and are wired
  (key_links patterns hold).
- Latest CI run green including the "Verify EdenDocs branding" step; its log
  contains BRANDING VERIFICATION PASSED.
- Objective-level: this closes the loop on all 5 roadmap success criteria —
  SC-1/2/3 proven by 02-04's smoke + this gate's dist checks, SC-4 by the
  checklist + sweep, SC-5 by the negative-flag gate + canary.
</verification>

<success_criteria>
BRAND-05 is verified against a written checklist whose every claim maps to a
mechanical assertion in CI, and BRAND-06 is enforced (not just observed) by a
gate that fails any future push reintroducing a phone-home flag or losing the
brand keys.
</success_criteria>

<output>
After completion, create `.planning/objectives/02-edendocs-rebrand-web-ui/02-05-SUMMARY.md`
with the green run evidence and the final consolidated divergence-allowlist
candidate list (harvested by Objective 5).
</output>
