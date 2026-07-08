# EdenDocs Trademark / MPL Checklist (BRAND-05)

## 1. Purpose + scope

This is the written trademark/MPL checklist that requirement **BRAND-05**
demands: every Collabora trademark surface in the shipped product is
inventoried below with its class, the action taken, where it was fixed, and
the evidence. Every claim in this document maps to a **mechanical assertion**
in `scripts/eden/verify-branding.sh`, which CI (`.github/workflows/build.yml`)
runs against the **built** `browser/dist` tree on every push to `eden-main` —
this document and that gate are maintained together.

Scope: the EdenDocs web/admin UI built by `scripts/eden/build.sh`
(Objective 2). Out of scope: `COOLWSD` wire/agent identifiers (deferred
BRAND-07, v2), container image LABELs (Objective 4 — see row in §3).

## 2. Surface taxonomy

| # | Class | Definition | Action rule |
|---|-------|------------|-------------|
| 1 | User-facing branding | Visible product identity: names, logos, titles, welcome page, palette | **MUST change** (patches / config / branding dir) |
| 2 | Default config values pointing at live Collabora infra | `help_url`, `brandProductURL`, `HELP_URL` configure default | **MUST change** (configure flags / config injection) |
| 3 | Internal protocol identifiers | Non-user-visible identifiers whose value matters for wire compatibility / upstream mergeability (`COOLWSD`, `cool.html`/`coolwsd.xml` filenames, C++ names, functional string literals) | **MUST NOT change** |
| 4 | Third-party technical documentation links | Genuine reference links to still-accurate third-party docs/community | **Accepted exception v1** — revisit when Eden publishes integrator/admin docs |
| 5 | Legal attribution | MPL notices, license files, per-file copyright headers | **MUST preserve verbatim** |
| 6 | Judgment calls | Provenance links / edge-case message strings not in BRAND-05's explicit surface list | **Flagged, not auto-fixed** — kept for v1, reviewed here |

## 3. Per-surface checklist

| Surface | Class | Action taken | Where fixed | Evidence |
|---------|-------|--------------|-------------|----------|
| Editor tab title (`<title>Collabora Online...` → `EdenDocs`) | 1 | Isolated upstream patch to `browser/html/cool.html.m4` | 02-01 commit `3aa5befe6c5` | verify-branding.sh gate 2 (`<title>EdenDocs</title>` in dist cool.html); runtime: `BRANDING SMOKE PASSED` |
| Admin console tab title | 1 | Isolated upstream patch to `browser/admin/admintemplate.html` | 02-01 commit `d27976178bb` | verify-branding.sh gate 2 (`<title>EdenDocs - Admin console</title>` in dist admin template) |
| Backstage header string ("Collabora Office") | 1 | Isolated upstream patch to `browser/src/control/backstage/Sidebar.tsx` | 02-02 commit `f13d4afc092` | Sweep (gate 5) finds no `Collabora Office` header string in dist bundles |
| Admin Version-tab headers ("Collabora Online" / "Collabora Office Engine") | 1 | Isolated upstream patch to `browser/admin/adminSettings.html` | 02-02 commit `b2759f21f4a` | Sweep (gate 5); admin HTML verified hit-free at source |
| Integrator-settings tab title (`<title>Collabora Online - Settings</title>` → `EdenDocs - Settings`) — **caught by the gate-5 sweep in CI** (run 28964158981), the sweep's first real find | 1 | Isolated upstream patch to `browser/admin/adminIntegratorSettings.html.m4` | 02-05 commit `6f9e184b4ec` | Sweep (gate 5) — the run 28964158981 survivor printout is the detection evidence |
| Backstage logo SVG content (`collabora-office-white.svg`) | 1 | Same-filename collision overwrite: `eden-branding/images/collabora-office-white.svg` ships the AO gold emblem via `--with-app-branding`'s `images/*.svg` glob (zero upstream edits) | 02-03 commit `938f8bf3714` | verify-branding.sh gate 1 (`efb32c` gold gradient present in dist SVG) |
| Favicon | 1 | Build-time overlay `cp eden-branding/favicon.ico ./favicon.ico` (no config override exists upstream — hardcoded lookup in `wsd/ClientRequestDispatcher.cpp`); NEVER committed at repo root | 02-03 asset + 02-04 commit `60daa2727fc` | Runtime byte-compare in smoke-test.sh (`BRANDING SMOKE PASSED`, run 28960901457) |
| Welcome page (fully Collabora-branded upstream asset) | 1 | Full replacement (`rm -rf` + `cp -R eden-branding/welcome`) — partial overlay would leave upstream slides/js | 02-03 commit `25ce2fbc786` + 02-04 commit `60daa2727fc` | verify-branding.sh gate 4 (`browser/dist/welcome/` contains no `collabora`, asserted case-insensitively) |
| Color palette (gold `#D4A853` primary) | 1 | `eden-branding/branding.css` `!important` overrides; upstream `color-palette.css` never edited | 02-03 commit `160335432f3` | verify-branding.sh gate 4 (`--color-primary: #D4A853 !important` in dist branding.css) |
| `help_url` (upstream default = live `help.collaboraoffice.com`) | 2 | `--with-help-url=https://aocyber.ai` configure flag | 02-04 commit `252f98ece5a` (`scripts/eden/build.sh`) | Flag present in build.sh; generated coolwsd.xml carries it |
| `brandProductName` / `brandProductURL` / `logoURL` config keys | 2 | Post-configure injection into generated (gitignored) `coolwsd.xml`: `EdenDocs` / `https://aocyber.ai` / `images/eden-logo.svg`; canary asserts keys exist (welcome-url dependency, §6) | 02-04 commit `252f98ece5a` | build.sh injection assertions + verify-branding.sh gate 7 canary |
| About dialog product name + vendor | 1/2 | Config-driven: `#product-name` filled at runtime from `brandProductName` (= EdenDocs); vendor via `--with-vendor="AO Cyber Systems"` (AC_SUBST → m4 `VENDOR` expansion in cool.html) | 02-04 commit `252f98ece5a` | verify-branding.sh gate 2 (`AO Cyber Systems` in dist cool.html); static fallback text is a reviewed exception (§4 row 13) |
| Admin console product name (`window.brandProductName`) | 1 | `eden-branding/branding.js` sets `window.brandProductName = "EdenDocs"` (admin console's only path to the value) | 02-03 commit `160335432f3` | verify-branding.sh gate 3; runtime admin-page check in smoke-test.sh |
| Container image LABELs | — | **None exist** in `docker/from-source/{Ubuntu,Debian}` (verified clean — zero LABEL directives); Objective 4 owns any future Eden LABEL scheme | n/a (verified, no action) | Research §6 verified-clean list |

## 4. Accepted exceptions (v1)

**Invariant: every line in `scripts/eden/trademark-exceptions.txt` has a row
here; additions require BOTH (an exceptions line AND its justifying row) in
the same commit.** The exceptions file is pure fixed strings (`grep -F`),
matched case-insensitively against per-occurrence ±40-char snippets from the
built dist — never whole lines (minification-safe).

| # | Exception line(s) | Where it survives | Class | Justification | Revisit trigger |
|---|-------------------|-------------------|-------|---------------|-----------------|
| 1 | `sdk.collaboraonline.com` | bundle.js / admin-bundle.js (errormessages.js, Control.ServerAuditDialog.ts ×14, AdminSocketServerAudit.js ×4) | 4 | Genuine third-party technical documentation links; EdenDocs does not fork the wsd/browser protocol so the docs remain accurate; ~9 upstream files = high merge cost | Eden publishes integrator docs |
| 2 | `forum.collaboraonline.com` | bundle.js (Control.Menubar.ts help menu) | 4 | Upstream community forum link | Eden publishes community/support channel |
| 3 | `help.collaboraoffice.com` | bundle.js (FormulaErrorHelpSection.ts) | 4 | Third-party help reference (note: the *default config* `help_url` IS fixed — §3) | Eden publishes help content |
| 4 | `www.collaboraonline.com` | Pre-seeded (research §6 third-party domain set) | 4 | Same third-party documentation class | Same as #1 |
| 5 | `www.collaboraoffice.org` | bundle.js (`Util.ts` FAQ link `/post/faq/`) | 4 | Third-party FAQ reference link | Same as #1 |
| 6 | `gerrit.collaboraoffice.com` | bundle.js / admin-bundle.js (Control.AboutDialog.ts, AdminSocketSettings.js git-hash provenance links) | 6 | Upstream-provenance links tying the running build's git hash to its actual source history — removal not clearly beneficial; kept for v1 | v2 branding review |
| 7 | `github.com/CollaboraOnline` | bundle.js (Menubar bug-report link, ServerAudit user-profile link, CSS issue-reference comments) | 4 | Links to the true upstream project (bug reports/provenance); accurate and honest | Eden opens its own tracker for browser-layer bugs |
| 8 | `//collaboraonline.com` | bundle.js (PresenterConsole.js `window.open('https://collaboraonline.com')`) | 4 | Bare-domain variant of #4 (protocol-anchored so it cannot mask subdomain leaks) | Same as #1 |
| 9 | `Collabora Online Development Edition (unbranded)` + `Collabora Online Development Edition` (split-string guard for esbuild `--line-limit`) | bundle.js (Socket.ts ×3, Clipboard.js, ProgressOverlay.js, Toolbar.js, Control.AboutDialog.ts) | 3 | Unreached fallback literal — only rendered when `brandProductName` is unset, and 02-04's canary + injection guarantee it is always set to EdenDocs | If the gate-7 canary ever fails |
| 10 | `:"Collabora Online"` | bundle.js (Map.WOPI.js bare ternary fallback, minified form) | 3 | Same unreached-fallback class as #9 (`brandProductName` always set) | Same as #9 |
| 11 | `collabora-office-white.svg` | bundle.css / backstage.css `url()` reference | 3 | Intentional filename collision — the FILE CONTENT is the AO gold emblem (gate 1 proves `efb32c`); the name must stay so upstream CSS resolves it with zero edits | Upstream renames the asset |
| 12 | `collaborat` | cool-help.html, bundle.js/css ("collaborative", "collaboration", "collaboratively", `coda-collab-collaborative-editing-*` asset names) | — | Dictionary-word false positives: `collabora` is a substring of ordinary English words. Cannot mask a real mark: brand usage is always `Collabora ` or a `collabora(online\|office)` domain, never `collaborat` | n/a (linguistic, permanent) |
| 13 | `id="product-name">Collabora Online` | dist cool.html (About dialog static markup from cool.html.m4:247) | 3 | Runtime-replaced fallback: `Control.AboutDialog.ts` fills `#product-name` from `brandProductName` (= EdenDocs) before display | If About dialog ever renders Collabora |
| 14 | `title="Collabora Online"` | dist wasm.html (iframe accessibility title; WASM harness page, dormant — no wasm build) | 6 | Dormant harness page shipped by the unconditional `html/*.html` copy rule; not product UI in this build | WASM support enabled |
| 15 | `replace("Collabora Office"` | bundle.js (Control.LokDialog.js — the replace SOURCE string that *applies* EdenDocs branding to engine-produced dialog titles) | 3 | Functional literal: this is the mechanism substituting `brandProductName` — removing it would break rebranding | Upstream changes the mechanism |
| 16 | `collabora-online-mobile` | bundle.js (Clipboard.js internal clipboard-URL identifier) | 3 | Internal protocol identifier (mobile-app clipboard path), never user-visible | BRAND-07 (v2) scope review |
| 17 | `prevent Collabora Online from updating` | bundle.js (Control.Zotero.js citation-unlink warning, l10n-keyed) | 6 | Edge-case dialog string not in BRAND-05's surface list; the English literal is the translation KEY — editing it breaks every existing l10n catalog entry | v2 branding review (patch + retranslate) |
| 18 | `Collabora Online server needs updating` | bundle.js (Map.VersionBar.js version-mismatch snackbar, l10n-keyed) | 6 | Same class as #17 — rarely-surfaced admin-facing message; l10n-key cost | Same as #17 |
| 19 | `Collabora Online not loaded yet` | bundle.js (Map.WOPI.js `console.error` developer diagnostic) | 6 | Dev-console diagnostic, not UI; l10n-free but harmless | Same as #17 |
| 20 | `Copyright the Collabora Online contributors` | Legal comment headers surviving in dist HTML/CSS (and esbuild-preserved SPDX legal comments in bundles) | 5 | **MUST preserve** — MPL attribution (see §5) | Never (legal) |
| 21 | `Copyright Collabora Productivity` | dist cool.html header comment (cool.html.m4:36) | 5 | **MUST preserve** — MPL attribution | Never (legal) |

### Sweep scope (include-globs and excludes — the sanctioned knobs)

The sweep covers `browser/dist` files matching `*.html`, `*.js`, `*.css`,
`*.svg` and **mirrors upstream's own production ship-set** by excluding
exactly what upstream's non-debug `install-data-hook` excludes
(`browser/Makefile.am` EXCLUDES list):

| Excluded from sweep | Why |
|---------------------|-----|
| `src/` directories (`browser/dist/src` tsc mirror) | Debug-only source mirror for devtools/sourcemaps; excluded from production installs by upstream; its string literals are all swept anyway inside `bundle.js`/`admin-bundle.js` |
| `debug.html`, `framed.html`, `framed.doc.html`, `load.doc.html`, `multidocs.html` | Upstream developer/integrator test harnesses, excluded from production installs by upstream; not EdenDocs product UI |
| `l10n/*.json` translation catalogs | Outside the include-globs by design: upstream translations may legitimately carry the product name; reviewed as a v1 exception. Revisit if EdenDocs ships its own catalogs |

## 5. MPL / legal preservation

- `COPYING`, `THIRDPARTYLICENSES`, and `CODA-THIRDPARTYLICENSES.html` are
  **byte-identical to their committed upstream state** — mechanically asserted
  on every push by verify-branding.sh gate 6
  (`git diff --quiet HEAD -- COPYING THIRDPARTYLICENSES CODA-THIRDPARTYLICENSES.html`),
  plus a positive `Mozilla Public` content check on `COPYING`.
- Per-file MPL headers in upstream files are never edited or stripped
  (taxonomy class 5); the isolated patches in §3 change only content strings,
  never license headers.
- New Eden-owned files (`eden-branding/*`, `scripts/eden/*`) follow the
  `README.FILENOTICES.md` convention: an MPL-2.0 header with an
  AO Cyber Systems copyright line.
- MPL §3.2 source-availability: the complete corresponding source is public at
  **https://github.com/AO-Cyber-Systems/EdenDocs** — linked from the welcome
  page footer (`eden-branding/welcome/welcome.html`) and this document.

## 6. No-phone-home (BRAND-06)

Four configure flags are **never passed** to `./configure` in
`scripts/eden/build.sh` (the HARD OMIT LIST):

- `--with-welcome-url`
- `--with-feedback-url`
- `--with-infobar-url`
- `--with-support-public-key`

`FEEDBACK_URL`/`INFOBAR_URL` default empty and `--with-support-public-key`
unset actively removes the support-key path — omission is sufficient
(research §7). Critically, `--with-welcome-url` also flips the configure
branch that generates `CONFIG_INTERFACE_FRAGMENT`: passing it would
**silently delete** `brandProductName`/`brandProductURL`/`logoURL` from the
generated coolwsd.xml (research Pitfall 1). Therefore:

- verify-branding.sh gate 7 asserts each forbidden flag is absent from
  `scripts/eden/build.sh` (negative `grep -- "--<flag>"`), and
- the `<brandProductName` **canary** in the generated `coolwsd.xml` proves the
  welcome-url dependency held — one assertion proves both halves.

These mechanical gates run in CI on every push; BRAND-06 is enforced, not
observed.

## 7. Divergence allowlist candidates (for Objective 5)

Upstream-file divergences carried by `eden-main`, each an isolated
single-purpose commit (UPST-02 harvest list):

| File | Commit | Purpose |
|------|--------|---------|
| `browser/html/cool.html.m4` | `3aa5befe6c5` (02-01) | Editor `<title>EdenDocs</title>` |
| `browser/admin/admintemplate.html` | `d27976178bb` (02-01) | Admin `<title>EdenDocs - Admin console</title>` |
| `browser/src/control/backstage/Sidebar.tsx` | `f13d4afc092` (02-02) | Backstage header "EdenDocs" |
| `browser/admin/adminSettings.html` | `b2759f21f4a` (02-02) | Trademark-free admin Version-tab headers |
| `browser/admin/adminIntegratorSettings.html.m4` | `6f9e184b4ec` (02-05) | Integrator-settings `<title>EdenDocs - Settings</title>` (gate-5 sweep catch) |

**REMARK — build-time-only working-tree divergence:** repo-root `favicon.ico`
is overwritten at build time by `scripts/eden/build.sh`
(`cp eden-branding/favicon.ico ./favicon.ico`) because coolwsd's favicon
route has no config override. This tracked upstream file is intentionally
dirty in the working tree after any build and must **never be committed**
(`git checkout -- favicon.ico` to restore); it is NOT an eden-main content
divergence.

---
*Verified mechanically by `scripts/eden/verify-branding.sh` (CI step
"Verify EdenDocs branding + trademark/MPL gates" in
`.github/workflows/build.yml`). Green-run evidence: recorded in §3/§8 after
the first green run of this gate.*

## 8. Green-run evidence

*(final green run pending — blocked by an exogenous upstream engine-tarball
republish at 2026-07-08T18:02:24Z that SIGSEGVs kit startup; see
02-05-SUMMARY.md. Resume: rerun run 28964794754 after the next upstream
republish of `engine-main-assets.tar.gz`.)*

Partial evidence already banked:

- Run [28964158981](https://github.com/AO-Cyber-Systems/EdenDocs/actions/runs/28964158981):
  gates 1-4 executed PASS against the real Linux dist; gate 5 correctly
  detected exactly ONE unexplained survivor
  (`<title>Collabora Online - Settings</title>`) with self-diagnosing output —
  fixed by isolated patch `6f9e184b4ec`. All 22 exception lines and the
  sweep-scope excludes were exact on first contact with the real dist tree.
- `BRANDING SMOKE PASSED` (runtime half): runs 28964158981 and 28960901457.
- Final `BRANDING VERIFICATION PASSED (BRAND-01..06)` line: pending green run.
