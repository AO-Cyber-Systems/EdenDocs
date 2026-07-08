---
objective: 02-edendocs-rebrand-web-ui
trd: "04"
type: standard
wave: 2
depends_on: ["02-01", "02-02", "02-03"]
files_modified:
  - scripts/eden/build.sh
  - scripts/eden/smoke-test.sh
autonomous: true
requirements: [BRAND-01, BRAND-02, BRAND-03, BRAND-04, BRAND-06]

must_haves:
  truths:
    - "The build configures with --with-app-branding/--with-app-name/--with-vendor/--with-help-url and the branding files land in browser/dist (branding.css, branding.js, images overwrites, toolbar-bg.svg)"
    - "The generated coolwsd.xml contains brandProductName=EdenDocs and brandProductURL=https://aocyber.ai (Pitfall-1 canary asserted loudly at build time)"
    - "The served favicon on 9980 is byte-identical to eden-branding/favicon.ico, and browser/dist/welcome/ contains ONLY the Eden replacement page"
    - "None of the four phone-home flags (with-welcome-url, with-feedback-url, with-infobar-url, with-support-public-key) is passed to configure — and CI proves the whole chain by staying green with new runtime branding checks in the smoke test"
  artifacts:
    - path: "scripts/eden/build.sh"
      provides: "Branding configure flags + welcome-url dependency comment + coolwsd.xml brand-key injection + favicon/welcome overlays + loud dist assertions"
      contains: "--with-app-branding"
    - path: "scripts/eden/smoke-test.sh"
      provides: "Runtime branding checks against coolwsd on native port 9980"
      contains: "branding.css"
  key_links:
    - from: "scripts/eden/build.sh"
      to: "browser/dist/branding.css + branding.js + images/*"
      via: "./configure --with-app-branding=$PWD/eden-branding → browser/Makefile.am build-cool copy recipe"
      pattern: "with-app-branding"
    - from: "scripts/eden/build.sh"
      to: "coolwsd.xml (generated, gitignored)"
      via: "post-configure python3 injection of brandProductName/brandProductURL/logoURL values"
      pattern: "brandProductName"
    - from: "scripts/eden/smoke-test.sh"
      to: "http://127.0.0.1:9980/favicon.ico"
      via: "curl | cmp against eden-branding/favicon.ico"
      pattern: "favicon"
---

<objective>
Wire the `eden-branding/` payload (TRD 02-03) into the Objective 1 build
pipeline: add the four branding configure flags to `scripts/eden/build.sh`,
assert the Pitfall-1 canary, inject the EdenDocs values into the generated
`coolwsd.xml`, add the favicon + welcome overlay steps that
`--with-app-branding` does NOT cover on Linux, extend `scripts/eden/smoke-test.sh`
with runtime branding checks, and drive CI back to green.

Purpose: This TRD turns the committed assets into a branded build. It modifies
ONLY Eden-owned scripts from Objective 1 (zero upstream files), extending the
existing CI pipeline rather than forking it.

Output: Updated build.sh + smoke-test.sh; a green push-triggered CI run whose
logs evidence the branding assertions.
</objective>

<execution_context>
@~/.claude/devflow/workflows/execute-trd.md
@~/.claude/devflow/templates/summary.md
</execution_context>

<embedded_context>

<codebase_examples>
Current configure invocation in `scripts/eden/build.sh:38-42` (append the new
flags to THIS call — none of the existing flags interact, research §7):
```bash
CONFIGURE_LOG="$(mktemp)"
./configure --enable-silent-rules --enable-debug \
  --with-lokit-path="$PWD/engine/include" \
  --with-lo-path="$PWD/engine/instdir" \
  LDFLAGS="-L/usr/local/lib" 2>&1 | tee "$CONFIGURE_LOG"
```

Existing loud-assertion pattern to mirror (build.sh:44-51, the BUILD-02
POCO-fallback assertion — the project's "silent set -e deaths are banned"
convention from 01-02-SUMMARY):
```bash
grep -q "POCO not found in the engine workdir, falling back to system POCO" "$CONFIGURE_LOG"
```

Existing smoke-test fail helper to reuse (scripts/eden/smoke-test.sh:35-40):
```bash
fail() {
  echo "SMOKE TEST FAILED: $1"
  echo "=== tail of /tmp/coolwsd.log ==="
  tail -n 200 /tmp/coolwsd.log 2>/dev/null || echo "(no /tmp/coolwsd.log)"
  exit 1
}
```

Generated coolwsd.xml brand-key shapes (from configure.ac:1270-1339's
CONFIG_INTERFACE_FRAGMENT — note brandProductURL ships a Collabora default that
MUST be overridden):
```xml
<brandProductName ...></brandProductName>
<brandProductURL ...>https://www.collaboraonline.com</brandProductURL>
<logoURL ...></logoURL>
```
coolwsd.xml is GITIGNORED (verified) — injecting values into it dirties nothing.
</codebase_examples>

<anti_patterns>
- Do NOT pass `--with-welcome-url`, `--with-feedback-url`, `--with-infobar-url`,
  or `--with-support-public-key` — ever. Welcome-url additionally DELETES the
  brand keys from coolwsd.xml (research Pitfall 1).
- Do NOT edit `coolwsd.xml.in`, `configure.ac`, or `browser/Makefile.am` — all
  wiring lives in Eden-owned scripts.
- Do NOT commit the overlaid repo-root `favicon.ico` — it is a build-time
  overlay of a TRACKED upstream file; the working-tree change is expected and
  reverted with `git checkout -- favicon.ico` (document this in the script
  comment). Guard every commit in this TRD with an explicit file list.
- In script comments, write the forbidden flag names WITHOUT the leading `--`
  (e.g. "with-welcome-url") so TRD 02-05's mechanical negative gate
  (`! grep -- '--with-welcome-url' scripts/eden/build.sh`) stays a plain grep.
- Do NOT use `cp eden-branding/welcome/* browser/dist/welcome/` (partial
  overlay leaves upstream slide*.png/welcome.js in place) — full replacement:
  `rm -rf` then `cp -R`.
</anti_patterns>

<error_recovery>
- If configure fails after adding flags: check `APP_HAS_BRANDING` requires
  `eden-branding/` to EXIST at configure time (it is committed by 02-03; if
  running out of order, stop and finish 02-03 first).
- If the brandProductName canary grep fails: someone/something enabled a
  welcome URL — inspect `$CONFIGURE_LOG` for ENABLE_WELCOME_MESSAGE; do NOT
  loosen the assertion (mirror of the POCO-fallback rule).
- If the admin-console curl check proves un-curlable (auth/route differences),
  downgrade THAT ONE check to the static dist grep (browser/dist admin
  template) and record the substitution in the SUMMARY — do not delete the
  check silently.
- CI fix cycles: diagnose from `gh run view --log-failed` (raw log zip via
  `gh api` if truncated) BEFORE any fix commit — Objective 1's pattern.
  Guidance: ≤4 fix cycles.
</error_recovery>

</embedded_context>

<context>
@.planning/PROJECT.md
@.planning/objectives/02-edendocs-rebrand-web-ui/OBJECTIVE.md
@.planning/objectives/02-edendocs-rebrand-web-ui/2-RESEARCH.md
@.planning/objectives/01-from-source-build-pipeline/01-02-SUMMARY.md
</context>

<research_context>
- Exact flag set (research "Exact Configure-Flag Set", amended by the locked
  help-url decision): `--with-app-branding="$PWD/eden-branding"
  --with-app-name="EdenDocs" --with-vendor="AO Cyber Systems"
  --with-help-url="https://aocyber.ai"`.
- No `make` invocation changes needed: the existing `make -j"$(nproc)" -C browser`
  already triggers `build-cool`, where the APP_HAS_BRANDING copy recipe lives
  (research §8). The recipe re-copies on every run (not mtime-gated).
- `--with-welcome-url` unset is a HARD PREREQUISITE for the brand keys existing
  in coolwsd.xml at all (configure.ac:1270-1339 else-branch) — hence the
  loud permanent comment + canary assertion.
- favicon: `handleFaviconRequest` is a hardcoded filesystem lookup
  (application-path `favicon.ico`, else FileServerRoot + `/favicon.ico`; both
  resolve to the repo root in dev/CI builds) — no config override exists.
- welcome: `browser/dist/welcome/` is unconditionally compiled with full
  Collabora branding for non-macOS builds; the branding-dir welcome overlay is
  ENABLE_MACOS-gated — hence the unconditional build.sh replacement step.
- `--with-info-url` intentionally NOT passed: INFO_URL only feeds the
  mobile-app m4 path (MOBILEAPP=false for web) — record as a comment.
</research_context>

<gotchas>
- HARD RULE: never use or reference port 8080 anywhere in scripts, comments, or
  checks (except an explicit prohibition sentence). Runtime checks use
  coolwsd's native 9980; any static local web serving would use 8091 (none
  needed here).
- `--with-vendor="AO Cyber Systems"` contains spaces — keep the existing
  configure line-continuation style and quoting exactly.
- The value injected for logoURL is `images/eden-logo.svg` (relative URL —
  resolves against the served cool.html base path, and the file is guaranteed
  in dist/images by the branding glob). It is best-effort: if runtime DOM
  verification ever shows it mis-resolving, set it to empty string (the
  Menubar fallback icon is trademark-clean per research §3) and record the
  decision.
- MPL: these are Eden-authored scripts (already headed from Objective 1) — keep
  headers intact.
</gotchas>

<tasks>

<task type="auto">
  <name>Task 1: build.sh — branding configure flags, welcome-url canary, coolwsd.xml value injection</name>
  <files>scripts/eden/build.sh</files>
  <action>
1. Append to the existing `./configure` invocation (before the LDFLAGS arg,
   keeping style):
   ```
   --with-app-branding="$PWD/eden-branding" \
   --with-app-name="EdenDocs" \
   --with-vendor="AO Cyber Systems" \
   --with-help-url="https://aocyber.ai" \
   ```
2. Directly above the configure call, add a LOUD permanent comment block
   (mirroring the existing POCO comment style):
   - BRAND-06 omit list: never pass with-welcome-url / with-feedback-url /
     with-infobar-url / with-support-public-key (write names WITHOUT leading
     dashes in the comment — see anti_patterns).
   - Pitfall-1 dependency: enabling a welcome URL makes configure.ac skip
     CONFIG_INTERFACE_FRAGMENT generation entirely, silently deleting
     brandProductName/brandProductURL/logoURL from coolwsd.xml (BRAND-01 dies
     with no error). The assertion below is the canary.
   - with-info-url intentionally unset (mobile-only m4 path; not the same key
     as brandProductURL).
3. After the existing POCO assertion, add the canary + injection:
   ```bash
   # BRAND-01/BRAND-06 canary (research Pitfall 1): the brand keys only exist
   # when no welcome URL is configured. If this fails, someone enabled a
   # welcome URL — do NOT loosen this assertion.
   grep -q "<brandProductName" coolwsd.xml \
     || { echo "ERROR: brandProductName key missing from generated coolwsd.xml — welcome-url dependency broken (see 2-RESEARCH.md Pitfall 1)"; exit 1; }
   ```
   Then inject the EdenDocs values into the GENERATED (gitignored) coolwsd.xml
   with a python3 one-liner using regex replacements on the three elements:
   brandProductName inner text → `EdenDocs`; brandProductURL inner text →
   `https://aocyber.ai` (overriding the collaboraonline.com default);
   logoURL inner text → `images/eden-logo.svg`. Follow with loud assertions:
   ```bash
   grep -q ">EdenDocs</brandProductName>" coolwsd.xml || { echo "ERROR: brandProductName injection failed"; exit 1; }
   grep -q "aocyber.ai</brandProductURL>" coolwsd.xml || { echo "ERROR: brandProductURL injection failed"; exit 1; }
   ```
4. Run `bash -n scripts/eden/build.sh` locally before committing.
Commit: `feat(02-04): branding configure flags + coolwsd.xml brand keys + welcome-url canary`
(commit ONLY scripts/eden/build.sh).
  </action>
  <verify>
bash -n scripts/eden/build.sh &amp;&amp; grep -q -- '--with-app-branding="$PWD/eden-branding"' scripts/eden/build.sh &amp;&amp; grep -q -- '--with-app-name="EdenDocs"' scripts/eden/build.sh &amp;&amp; grep -q -- '--with-vendor="AO Cyber Systems"' scripts/eden/build.sh &amp;&amp; grep -q -- '--with-help-url="https://aocyber.ai"' scripts/eden/build.sh &amp;&amp; grep -q '&lt;brandProductName' scripts/eden/build.sh &amp;&amp; ! grep -q -- '--with-welcome-url' scripts/eden/build.sh &amp;&amp; ! grep -q -- '--with-feedback-url' scripts/eden/build.sh &amp;&amp; ! grep -q -- '--with-infobar-url' scripts/eden/build.sh &amp;&amp; ! grep -q -- '--with-support-public-key' scripts/eden/build.sh &amp;&amp; ! grep -q '8080' scripts/eden/build.sh
  </verify>
  <done>build.sh syntax-checks; carries all four branding flags, the canary + injection assertions, and contains NO occurrence of the four forbidden flags (even in comments) nor of 8080.</done>
  <recovery>If a negative grep trips on a comment, reword the comment to name flags without leading dashes. Never weaken the verify command.</recovery>
</task>

<task type="auto">
  <name>Task 2: build.sh overlays (favicon + welcome) and smoke-test.sh runtime branding checks</name>
  <files>scripts/eden/build.sh, scripts/eden/smoke-test.sh</files>
  <action>
A) In `build.sh`, after the existing `make -j"$(nproc)" -C browser` step, add an
"EdenDocs branding overlays" block (research Pitfall 3 — neither surface is
covered by --with-app-branding on Linux):
   ```bash
   # favicon has NO config override (hardcoded filesystem lookup in
   # wsd/ClientRequestDispatcher.cpp) — overlay the served path. NOTE: this
   # overwrites a TRACKED upstream file in the working tree ON PURPOSE and it
   # must NEVER be committed; restore locally with: git checkout -- favicon.ico
   cp eden-branding/favicon.ico ./favicon.ico

   # browser/dist/welcome is unconditionally compiled with Collabora branding
   # (the branding-dir welcome overlay is macOS-gated). Full replacement so no
   # upstream slide/js assets survive:
   rm -rf browser/dist/welcome
   cp -R eden-branding/welcome browser/dist/welcome
   ```
   Then extend the existing loud output assertions with branding assertions:
   dist files exist (`browser/dist/branding.css`, `browser/dist/branding.js`,
   `browser/dist/images/toolbar-bg.svg`); the images overwrite really happened
   (`grep -qi 'efb32c' browser/dist/images/collabora-office-white.svg` — the
   upstream white glyph has no gold gradient stop); welcome is clean
   (`! grep -rqi collabora browser/dist/welcome/`). Each with an `|| { echo
   "ERROR: ..."; exit 1; }` tail.
B) In `smoke-test.sh`, after the two existing /hosting checks, add runtime
branding checks on native port 9980 using the existing `fail()` helper:
   ```bash
   curl -fsS http://127.0.0.1:9980/browser/dist/cool.html | grep -q 'branding.css' \
     || fail "served cool.html does not reference branding.css (BRAND-01 hook missing)"
   curl -fsS http://127.0.0.1:9980/browser/dist/cool.html | grep -q '<title>EdenDocs</title>' \
     || fail "served cool.html title is not EdenDocs (BRAND-02)"
   curl -fsS http://127.0.0.1:9980/browser/dist/cool.html | grep -q 'AO Cyber Systems' \
     || fail "served cool.html missing vendor 'AO Cyber Systems' (BRAND-02 --with-vendor)"
   curl -fsS http://127.0.0.1:9980/favicon.ico | cmp -s - eden-branding/favicon.ico \
     || fail "served favicon is not the EdenDocs favicon (BRAND-04)"
   curl -fsS http://127.0.0.1:9980/browser/dist/images/eden-logo.svg | grep -qi 'efb32c' \
     || fail "eden-logo.svg not served from dist images (BRAND-01 logoURL target)"
   echo "BRANDING SMOKE PASSED: EdenDocs branding served on 9980"
   ```
   Keep the final success echo of the script intact; add the branding echo
   before it.
Commit both files together:
`feat(02-04): favicon/welcome overlays + runtime branding smoke checks`
(explicit file list: scripts/eden/build.sh scripts/eden/smoke-test.sh — NEVER
include a modified root favicon.ico if present locally).
  </action>
  <verify>
bash -n scripts/eden/build.sh &amp;&amp; bash -n scripts/eden/smoke-test.sh &amp;&amp; grep -q 'rm -rf browser/dist/welcome' scripts/eden/build.sh &amp;&amp; grep -q 'cp eden-branding/favicon.ico ./favicon.ico' scripts/eden/build.sh &amp;&amp; grep -q 'efb32c' scripts/eden/build.sh &amp;&amp; grep -q 'branding.css' scripts/eden/smoke-test.sh &amp;&amp; grep -q 'cmp -s - eden-branding/favicon.ico' scripts/eden/smoke-test.sh &amp;&amp; ! grep -c '8080' scripts/eden/smoke-test.sh | grep -qv '^1$' &amp;&amp; git show --name-only --pretty=format: HEAD | grep -vq 'favicon.ico'
  </verify>
  <done>Both scripts syntax-check; overlays are full-replacement style; smoke test carries the 5 runtime branding checks; the commit does not include favicon.ico; smoke-test.sh's only 8080 mention remains its pre-existing prohibition comment.</done>
  <recovery>If the smoke additions reference a wrong served path, adjust ONLY the path (never the port — 9980) after checking research §8's verification-command candidates; if favicon.ico shows in `git status`, run `git checkout -- favicon.ico` before committing.</recovery>
</task>

<task type="auto">
  <name>Task 3: Push and drive CI green with branding evidence</name>
  <files>scripts/eden/build.sh, scripts/eden/smoke-test.sh</files>
  <action>
Push eden-main (wave-1 patch commits + 02-03 assets + this TRD's commits ride
together if not already pushed). Watch the run:
`gh run watch $(gh run list --workflow=build.yml --branch eden-main --limit 1 --json databaseId --jq '.[0].databaseId')`

On failure: diagnose from `gh run view --log-failed` (raw log archive via
`gh api` when truncated) BEFORE any fix commit; each fix is its own commit in
the owning file (script fix goes in scripts/eden/, asset fix rides as a
follow-up to 02-03's files with a `fix(02-04):` message). Max 4 fix cycles —
if still red, STOP and record the diagnosis in the SUMMARY.

On success: capture into the SUMMARY (a) run ID + URL, (b) the log lines for
the POCO assertion, the brandProductName canary, `build.sh complete`,
`SMOKE TEST PASSED`, and `BRANDING SMOKE PASSED`.
  </action>
  <verify>
gh run list --workflow=build.yml --branch eden-main --limit 1 --json conclusion --jq '.[0].conclusion' | grep -q success
  </verify>
  <done>Latest build.yml run on eden-main is green and its logs contain the BRANDING SMOKE PASSED line (evidence recorded in SUMMARY).</done>
  <recovery>If >4 fix cycles loom, stop, summarize the failing step + exact log evidence in the SUMMARY, and surface to the user rather than thrashing CI.</recovery>
</task>

</tasks>

<validation_gates>
<lint>bash -n scripts/eden/build.sh &amp;&amp; bash -n scripts/eden/smoke-test.sh</lint>
<test>! grep -rn '8080' scripts/eden/build.sh scripts/eden/smoke-test.sh | grep -vi 'never\|banned\|prohibit\|do not' | grep -q .</test>
<build>gh run list --workflow=build.yml --branch eden-main --limit 1 --json conclusion --jq '.[0].conclusion' | grep -q success</build>
</validation_gates>

<verification>
- build.sh: 4 branding flags present; 4 forbidden flags absent (as literal
  `--flag` strings anywhere in the file); canary + injection assertions loud.
- smoke-test.sh: 5 runtime branding checks on 9980 using fail().
- CI green with BRANDING SMOKE PASSED in logs.
- Repo-root favicon.ico NOT committed (git log -- favicon.ico shows no new
  commit; the overlay is build-time only).
</verification>

<success_criteria>
A push-triggered CI run builds the fully-branded tree from the committed
scripts alone and proves at runtime — on coolwsd's native 9980 — that the
editor page references branding.css, titles/vendor read EdenDocs / AO Cyber
Systems, the favicon is byte-identical to the Eden asset, and the brand keys
survived configure (BRAND-06 canary intact).
</success_criteria>

<output>
After completion, create `.planning/objectives/02-edendocs-rebrand-web-ui/02-04-SUMMARY.md`
including CI run IDs/URLs and the captured assertion log lines. Note the
repo-root favicon.ico build-time overlay as a divergence-allowlist REMARK
(working-tree-only, never committed).
</output>
