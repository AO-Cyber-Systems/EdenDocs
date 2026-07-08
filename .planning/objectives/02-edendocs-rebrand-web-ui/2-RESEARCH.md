# Objective 2: EdenDocs Rebrand of the Web UI - Research

**Researched:** 2026-07-08
**Domain:** Collabora Online (autotools + m4-templated HTML + runtime `%TOKEN%` substitution) rebrand
**Confidence:** HIGH — every finding below is a direct file:line citation from this checkout (`/Users/justin/dev/EdenDocs`, branch `eden-main`), not training-data recall. No Context7/WebSearch was needed or used: this objective is pure source-code archaeology of the current tree.

<phase_requirements>
## Objective Requirements

| ID | Description | Research Support |
|----|-------------|-------------------|
| BRAND-01 | Editor UI shows EdenDocs product name/logo via `coolwsd.xml` (`brandProductName`/`brandProductURL`/`logoURL`) + additive `eden-branding/` wired with `--with-app-branding` | §1 Branding wiring, §2 coolwsd.xml keys, §"Exact configure-flag set", §"eden-branding/ layout" — mechanism fully traced end-to-end, zero upstream edits needed for this part |
| BRAND-02 | Window/tab titles + About dialog show EdenDocs (2 isolated `<title>` patches + compiled `--with-app-name`/`--with-vendor` fallbacks) | §3 Title/About surfaces — exact line numbers for both `<title>` tags; About dialog already config-driven; **scope-expansion finding**: a 3rd hardcoded string (Backstage header) and a 4th surface (admin Version tab) also require edits — see §6 and Pitfall 5 |
| BRAND-03 | Admin console carries branding via shared `%BRANDING_JS%` hook | §4 Admin console branding — traced the exact reason `%BRANDING_JS%` is *required* (not optional) for the admin console: `brandProductName` has no config-substitution path into `admintemplate.html`, only `branding.js` can supply it |
| BRAND-04 | Favicon replaced + color palette overridden via branding-dir CSS (upstream `color-palette.css` never edited) | §5 Color palette, §6 Favicon finding — favicon has **no config override**, documented as a build-script overlay decision, not a config key |
| BRAND-05 | Collabora trademarks removed from all user-facing surfaces, verified against a written checklist; MPL/COPYING/THIRDPARTYLICENSES preserved | §6 Trademark surface inventory — full taxonomy table + verification checklist |
| BRAND-06 | No-phone-home defaults verified (`INFOBAR_URL`/`FEEDBACK_URL`/`INFO_URL` unset; `--with-support-public-key`/`--with-welcome-url` never passed) | §7 Phone-home audit points — **critical dependency discovered**: `--with-welcome-url` must stay unset or BRAND-01's coolwsd.xml keys don't exist at all |

</phase_requirements>

## Summary

EdenDocs' rebrand surface splits into three genuinely different mechanisms, and conflating them is the main risk for planning:

1. **Config-driven** (`coolwsd.xml` → C++ runtime `%TOKEN%` substitution in `wsd/FileServer.cpp`) — the editor's product name, product URL and logo already flow through this path with zero source edits required. This is what BRAND-01 targets.
2. **Build-time m4 literal substitution** (`browser/Makefile.am`'s `m4 -PE -D...` invocations against `cool.html.m4` / `adminIntegratorSettings.html.m4`) — `VENDOR`, `INFO_URL`, `APP_NAME` flow through here from `configure.ac`'s `AC_SUBST` variables. Compiled-in fallbacks only; not runtime-configurable.
3. **Hardcoded strings** requiring an actual source-code edit — confirmed to be more numerous than the objective brief's two named `<title>` patches. Research found a third (Backstage panel header, `Control Sidebar.tsx`) and a fourth surface (admin console Version tab) that the brief did not name. Both are flagged prominently below as scope-expansion findings for the planner.

The additive `--with-app-branding=<dir>` mechanism (`configure.ac:222-226,2386-2397` → `browser/Makefile.am:1014-1050`) is confirmed to work exactly as BRAND-01 describes: it copies `branding*` files and overwrites same-named files under `browser/dist/images/*.svg` **after** all other build artifacts are already in place, with zero upstream file edits. This is the single most load-bearing verified fact in this research: EdenDocs can silently replace the built-in Collabora logo SVG (`collabora-office-white.svg`) just by shipping a same-named file in `eden-branding/images/`, no source edit at all.

The single highest-value discovery is a **hard dependency**: `--with-welcome-url` must remain unset not only for BRAND-06 (no phone-home) but because `configure.ac:1270-1339`'s conditional logic means the `CONFIG_INTERFACE_FRAGMENT` (which literally contains the `brandProductName`/`brandProductURL`/`logoURL` XML key definitions BRAND-01 needs) is **only generated in the `else` branch** — i.e., when `--with-welcome-url` is absent. If a future contributor ever passes `--with-welcome-url`, BRAND-01's coolwsd.xml keys silently disappear. This must be a permanent, loud assertion in the build script (mirroring Objective 1's "silent set -e deaths are banned" pattern), not just a one-time flag choice.

Two further verified pitfalls with real user-facing impact: (a) the dark-theme toggle (`?darkTheme=true`) JS-injects `color-palette-dark.css` into `<head>` *after* our branding CSS is already there, so a naive `branding.css` override of `--color-primary` will be silently reverted to Collabora blue in dark mode unless the override uses `!important`; (b) `browser/welcome/welcome.html` (fully Collabora-branded marketing splash) is unconditionally compiled into `browser/dist/welcome/` for all non-macOS builds — the `--with-app-branding` welcome-folder overlay is gated `if ENABLE_MACOS` only, so the default Linux/web build ships the untouched Collabora welcome page as a directly fetchable static asset even though `welcome.enable` defaults false.

**Primary recommendation:** Wire `--with-app-branding=$PWD/eden-branding --with-app-name="EdenDocs" --with-vendor="AO Cyber Systems"` into `scripts/eden/build.sh`'s existing `./configure` call (confirmed compatible with Objective 1's exact invocation, no other flag interacts). Build `eden-branding/` as a self-contained additive directory. Treat the three source-edit patches (both `<title>` tags + the newly discovered Backstage header string) as three separate isolated commits, each independently revertable across upstream merges, per the project's own mergeability rule.

## 1. Branding Wiring (`--with-app-branding`)

- `configure.ac:222-226` — `AC_ARG_WITH([app-branding], ...)`. Help text says branding ends up "in the app bundle as branding" — this wording is slightly misleading for the web/Linux target (see below); the iOS-specific consumer (`configure.ac:2418-2428`, copies into `ios/Mobile/Branding`) is not relevant to this objective's scope.
- `configure.ac:2386-2397` — sets `APP_BRANDING_DIR` (absolute path passed to `--with-app-branding`) and `APP_HAS_BRANDING` (`"true"` iff `-d "$with_app_branding"` at configure time), exposed to automake via `AM_CONDITIONAL([APP_HAS_BRANDING], ...)`.
- `browser/Makefile.am:9` — `DIST_FOLDER ?= $(builddir)/dist`, i.e. `browser/dist`.
- `browser/Makefile.am:938-940` — `all-local: build-cool`; `make -C browser` (Objective 1's existing invocation, no target given) runs the default `all` target, which triggers `all-local`, which triggers `build-cool`. **No new build-script wiring beyond the configure flag is required** — Objective 1's `make -j"$(nproc)" -C browser` call already exercises this path.
- `browser/Makefile.am:1014-1050` (`build-cool:` recipe body, runs **after** all its file prerequisites — i.e. after `cool.html`, `bundle.css`, `bundle.js`, `color-palette.css`, `color-palette-dark.css`, `adminIntegratorSettings`, and the default `browser/images/*` copies are already built):
  ```makefile
  if APP_HAS_BRANDING
      @if test -d "$(APP_BRANDING_DIR)" ; then cp -a "$(APP_BRANDING_DIR)"/branding* $(DIST_FOLDER)/ ; fi
      @if test -d "$(APP_BRANDING_DIR)" ; then cp -a "$(APP_BRANDING_DIR)"/images/*.svg $(DIST_FOLDER)/images/ ; fi
      @if test -d "$(APP_BRANDING_DIR)" ; then cp -a "$(APP_BRANDING_DIR)/images/toolbar-bg-logo.svg" $(DIST_FOLDER)/images/toolbar-bg.svg ; fi
  if ENABLE_MACOS
      @if test -d "$(APP_BRANDING_DIR)" ; then cp -a "$(APP_BRANDING_DIR)/welcome" "$(DIST_FOLDER)/" ; fi
  endif
  ```
  Verified semantics:
  - `branding*` glob → copies straight into `$(DIST_FOLDER)/` (**not** a `branding/` subdirectory, despite the configure.ac help text) — so `eden-branding/branding.css` and `eden-branding/branding.js` land at `browser/dist/branding.css` / `browser/dist/branding.js`, exactly matching the URLs the `%BRANDING_CSS%`/`%BRANDING_JS%` runtime hooks construct (see §2/§4).
  - `images/*.svg` glob → copies into `$(DIST_FOLDER)/images/`, **same filename overwrites same filename** — this is the mechanism for silently replacing `collabora-office-white.svg` (used by `backstage.css`, see §6) with zero upstream edits.
  - `images/toolbar-bg-logo.svg` is explicitly **renamed** to `toolbar-bg.svg` on copy — the source file in `eden-branding/` must be named `toolbar-bg-logo.svg` even though the destination is `toolbar-bg.svg`.
  - The `welcome/` overlay is gated `if ENABLE_MACOS` — **does not apply to the default web/Linux build**. See Pitfall 4.
  - This recipe body executes unconditionally every time `build-cool` runs (it's not a file-mtime-gated rule), so there is no staleness risk from repeated `make -C browser` calls once `--with-app-branding` is configured — editing `eden-branding/*` and re-running `make -C browser` always re-copies. Flipping `APP_HAS_BRANDING` on/off requires re-running `./configure` (it's an `AM_CONDITIONAL` baked in at configure time), which Objective 1's build script already does fresh on every invocation (`./autogen.sh && ./configure ...`), so this is a non-issue for CI.
- **Alternative considered, not prescribed**: `--with-icon-theme=<path>` (`configure.ac:293-295,1218-1225` → `browser/Makefile.am:129-132,1241-1247`) is a second, purpose-built image-override mechanism using real Make pattern-rule prerequisites (`$(DIST_FOLDER)/images/%: $(CUSTOM_ICONS_DIRECTORY)/%` defined *before* the default `$(DIST_FOLDER)/images/%: $(srcdir)/images/%` rule, so GNU Make prefers it when the custom file exists). Functionally equivalent to the `images/*.svg` glob in `--with-app-branding` for this use case. **Not needed**: BRAND-01 explicitly names `--with-app-branding`'s own image-glob mechanism, which was independently verified to work; introducing a second flag would only add complexity.

## 2. `coolwsd.xml` Keys

- `configure.ac:1270-1339` (read in full) — **critical branch dependency**: if `with_welcome_url` is set, `ENABLE_WELCOME_MESSAGE=1` and `CONFIG_INTERFACE_FRAGMENT` is **never assigned**. Only in the `else` branch (unset, our target state) does configure.ac generate:
  ```
  CONFIG_INTERFACE_FRAGMENT="<brandProductName ...></brandProductName>
  <brandProductURL ... default...>https://www.collaboraonline.com</brandProductURL>
  <logoURL ...></logoURL>"
  ```
  i.e. **BRAND-06 (`--with-welcome-url` never passed) is a hard prerequisite for BRAND-01's coolwsd.xml keys existing at all**, not a parallel/independent requirement. This must be documented as a permanent constraint comment in the build script, the same way Objective 1 documented its POCO-fallback assertion.
- `coolwsd.xml.in:276` — `@WELCOME_CONFIG_FRAGMENT@` placeholder (before `<user_interface>`).
- `coolwsd.xml.in:278-285` — `<user_interface>` block; line 283 is `@CONFIG_INTERFACE_FRAGMENT@`, where the three brand keys land when generated.
- `coolwsd.xml.in:413` — `<help_url ... default="@HELP_URL@">@HELP_URL@</help_url>`.
- `configure.ac:2461-2472` — `HELP_URL` default (when neither `--with-help-url` nor `--without-help-url` given) is `"https://help.collaboraoffice.com/help.html?"`. **This is a live Collabora domain reference that ships by default and is in-scope for BRAND-05** (see §6).
- `configure.ac:1270-1339` — `FEEDBACK_URL`/`INFOBAR_URL` both default empty, only set via `--with-feedback-url`/`--with-infobar-url` (never to be passed, per BRAND-06).
- Runtime consumption confirmed in `wsd/FileServer.cpp:1560-1585` — the same `#if ENABLE_WELCOME_MESSAGE ... #else` split exists at runtime: only the `#else` branch (our target) populates `%PRODUCT_BRANDING_NAME%`/`%PRODUCT_BRANDING_URL%`/`%LOGO_URL%` from `ConfigUtil::getConfigValue<std::string>(config, "user_interface.brandProductName", "")` etc.; the `#if` branch hardcodes them empty regardless of config.

## 3. Title/About Surfaces

- **Patch 1 (editor tab title):** `browser/html/cool.html.m4:42` — `<title>Online Editor</title>`, literal hardcoded string, no token. Isolated single-line edit.
- **Patch 2 (admin console tab title):** `browser/admin/admintemplate.html:12` — `<title>Collabora Online - Admin console</title>`, literal hardcoded string. Isolated single-line edit.
  - Important nuance verified: this static title is only what's visible until JS runs. `browser/admin/adminBody.html:2-3,11` (and `adminClusterBody.html:2-3,39`) already contain `if (typeof brandProductName !== 'undefined') {l10nstrings.strProductName = brandProductName}` followed by `document.title = l10nstrings.strProductName + ' - ' + l10nstrings.strAdminConsole;` — so once `window.brandProductName` is set (see §4), the admin console's title is overwritten to "EdenDocs - Admin Console" a moment after load. **The static `<title>` patch is a no-flash-of-wrong-title / no-JS fallback improvement, not the sole title mechanism** — but it's still required for BRAND-02 (there is a real flash of "Collabora Online" otherwise, and it's the only value used if JS fails to run).
- **About dialog** — `browser/src/control/Control.AboutDialog.ts` (read in full, 503 lines) is **already fully config-driven**, no source edit needed for the product name itself:
  ```ts
  declare var brandProductName: any;
  productName = typeof brandProductName === 'string' && brandProductName.length > 0
      ? brandProductName
      : 'Collabora Online Development Edition (unbranded)';
  ```
  and `Copyright © ${window.copyrightYear}, ${window.vendor}.` — both `vendor`/`copyrightYear` come from `data-vendor='VENDOR'` / `data-copyright-year='_YEAR_'` on `cool.html.m4`'s hidden `#initial-variables` input (m4-substituted from `configure.ac`'s `VENDOR` AC_SUBST var), read by `browser/js/global.js:473-474,517-518`.
  - **Open question, not blocking**: `Control.AboutDialog.ts` also hardcodes a Gerrit git-hash hyperlink: `` `https://gerrit.collaboraoffice.com/plugins/gitiles/online/+log/${info.coolwsdHash}` ``. This is upstream-provenance attribution (arguably legitimate/desirable to keep, and the same commit likely doesn't exist in Collabora's Gerrit post-fork anyway so the link may 404 either way). Recommend: leave as-is for v1 (not named in BRAND-05's explicit surface list — "UI strings, help_url, container LABEL metadata"), revisit only if the user wants full attribution-link removal.
- Global consumers of `brandProductName`/`logoURL` confirmed via grep (all already config-driven, no edits needed once the config keys/branding.js are populated): `browser/js/global.js:317-327` (reads hidden-input values into `window.brandProductName`/`brandProductURL`/`logoURL`), `browser/src/main.js:340` (`document.title = fileName + ' - ' + window.brandProductName`), `browser/src/control/Control.Menubar.ts:2688-2714` (`_createFileIcon`: uses `window.logoURL` as a CSS `backgroundImage`, falls back to a LibreOffice app-type icon class — not a Collabora trademark, safe to leave as fallback), `browser/src/control/Toolbar.js:743-751` (`_doOpenHelpFile`: help dialog title uses `brandProductName` with the same "(unbranded)" fallback pattern), `browser/src/control/Control.LokDialog.js:304-305` (replaces literal `'Collabora Office'` substring inside LOKit dialog titles with `brandProductName` — already config-aware), `browser/src/layer/marker/ProgressOverlay.js:33-38`, `browser/src/map/handler/Map.WOPI.js:215`, `browser/src/map/Clipboard.js:1501` (all share the same `'Collabora Online Development Edition (unbranded)'` fallback string — harmless once `brandProductName` is populated, since the fallback is never reached).

## 4. Admin Console Branding (`%BRANDING_JS%`)

- `wsd/FileServer.cpp:2523-2622` (`preprocessAdminFile()`, read in full) — the admin console's own request handler:
  - Loads `admintemplate.html` (line ~2555-2557).
  - `admintemplate.html:61-63`:
    ```html
    <body>
        <!--%BRANDING_JS%-->
        <!--%BODY%-->
        <!--%FOOTER%-->
    ```
    `<!--%BRANDING_JS%-->` sits **before** `<!--%BODY%-->` in document order, so the `<script src=".../branding.js">` tag it becomes executes (script tags without `defer`/`async` run synchronously as parsed) **before** the inline script inserted at `<!--%BODY%-->` (i.e. `adminBody.html`'s content) runs.
  - `wsd/FileServer.cpp:2579-2582` constructs `brandJS = <script src="<responseRoot>/browser/<HASH>/branding.js"></script>` (constant `BRANDING = "branding"`, `wsd/FileServer.cpp:1279`).
  - `wsd/FileServer.cpp:2585-2605` — if support-key support is compiled in (`ENABLE_SUPPORT_KEY`, not our build) and the key is invalid/expired, `brandJS` is overridden to a `branding-unsupported.js`/`.css` variant instead. Irrelevant since Objective 2 never passes `--with-support-public-key` (BRAND-06).
- **Critical finding, not stated in the objective brief**: `grep -rn "brandProductName" browser/admin/` shows the ONLY producer/consumer pair is:
  ```
  browser/admin/adminBody.html:2:    if (typeof brandProductName !== 'undefined') {l10nstrings.strProductName = brandProductName}
  browser/admin/adminClusterBody.html:2:  if (typeof brandProductName !== 'undefined') { l10nstrings.strProductName = brandProductName }
  ```
  There is **no** `%PRODUCT_BRANDING_NAME%` substitution wired into `preprocessAdminFile()` at all (confirmed by re-reading its full body — it only replaces `JWT_TOKEN`, `BODY`, `MAIN_CONTENT`, `ROUTE_TOKEN`, `BRANDING_JS`, `FOOTER`, `VERSION`, `SERVICE_ROOT`). This means **`brandProductName` is a bare global that is never defined anywhere in the admin console bundle by default** — the `typeof brandProductName !== 'undefined'` check is written defensively for exactly this reason: **`eden-branding/branding.js` is the only thing that can ever satisfy it**, since its `<script>` tag is guaranteed (by template order, above) to execute before `adminBody.html`'s inline script.
  - **This makes the required content of `eden-branding/branding.js` concrete and mandatory, not optional**: it must contain `window.brandProductName = "EdenDocs";` (a plain global assignment) for BRAND-03 to have any effect on the admin console header/title text at all. Without it, the admin console silently keeps the default `'Collabora Online Development Edition (unbranded)'` string from `browser/admin/admin.strings.js:14`, even with `coolwsd.xml`'s `brandProductName` correctly set to "EdenDocs" (that key only reaches the *editor*, via the completely separate global.js/hidden-input path in §3 — the two surfaces do NOT share the config-key path, only the `window.brandProductName` JS-global name).
  - Also verified clean via grep: `browser/admin/adminSettings.html:120-155` has two **hardcoded** headers unrelated to `strProductName` — `<h5><b>Collabora Online</b></h5>` (line 137) and `<h5><b>Collabora Office Engine</b></h5>` (line 142) in the admin "Version" info tab. These do **not** update via `brandProductName` (they're static markup, not `l10nstrings`-templated) — a genuine 4th hardcoded-string surface, see §6.

## 5. Color Palette

- `browser/css/color-palette.css` (light) and `browser/css/color-palette-dark.css` (dark) both define the same `:root { --color-* }` custom-property set; both are listed as `build-cool` prerequisites (`browser/Makefile.am:1032-1033`) and bundled into `bundle.css` at build time.
- Full accent/primary token set (identical property names, different values per file):
  | Property | Light value | Dark value | Note |
  |---|---|---|---|
  | `--color-primary` | `#0b87e7` | `#0b87e7` | border-color; primary Collabora blue |
  | `--color-primary-text` | `#fff` | `#fff` | text color used **on** `--color-primary-lighter` background |
  | `--color-primary-dark` | `#0063b1` | `#0063b1` | |
  | `--color-primary-darker` | `#004b86` | `#004b86` | |
  | `--color-primary-lighter` | `#83beec` | `#83beec` | background-color |
  | `--color-btn-primary-hover-bg` | `#e6f3fd` | `#e6f3fd` | |
  | `--color-overlay` | `#1c5fa814` | `#1c5fa814` | alpha-blue overlay tied to old primary hue |
  | `--color-hyperlink` | `#000080` | `#1d99f3` | differs light/dark |
- Recommended Eden gold mapping (from PROJECT.md's locked `EdenColors.gold` ramp): `--color-primary → #D4A853` (500), `--color-primary-dark → #A67A38` (700), `--color-primary-darker → #856131` (800), `--color-primary-lighter → #F4D5AA` (200, background use), `--color-btn-primary-hover-bg → #FAECD5` (100). **Contrast flag**: `--color-primary-text` is documented as "text color when primary-lighter is background" and is currently white (`#fff`) — white-on-`#F4D5AA` (light gold) fails WCAG contrast; this needs to flip to a dark value (e.g. gold-950 `#3D2A14` or plain near-black) wherever `--color-primary-lighter` is used as a background with `--color-primary-text` as foreground. Flagged as a planning decision, not a research gap.
- `--color-overlay` (currently blue-tinted alpha) should also be re-derived from the gold hex if any UI overlay effect is expected to read as "branded" rather than a stray blue tint.
- **CSS cascade verified**: `browser/html/cool.html.m4:94-102` places `<!--%BRANDING_CSS%-->` **after** the `bundle.css`/per-file CSS list in document order — i.e. after `color-palette.css` is already loaded — so plain (non-`!important`) custom-property overrides in `eden-branding/branding.css` win over the light palette by normal cascade order alone. **However** (Pitfall, see below): the dark-theme toggle script (`cool.html.m4:116-128`) does `document.head.appendChild(link)` for `color-palette-dark.css` **at runtime**, when `?darkTheme=true` is present — since `appendChild` places it at the end of `<head>`, it lands **after** our branding `<link>` (which was already in the static HTML, earlier in document order), and its `--color-primary` etc. re-assert the Collabora blue, silently undoing the light-mode override. **`eden-branding/branding.css` must declare its `--color-*` overrides with `!important`** to survive this, since custom properties do respect `!important` in CSS cascade resolution.
- `favicon.ico` (`wsd/ClientRequestDispatcher.cpp:1755-1780`, `handleFaviconRequest`) has **no config override whatsoever** — it's a hardcoded filesystem lookup: application-path `favicon.ico` first, else `COOLWSD::FileServerRoot + "/favicon.ico"`. Combined with `coolwsd.xml.in:54`'s `file_server_root_path` default (`"browser/../"`, i.e. repo root in dev builds) and `wsd/COOLWSD.cpp:3935-3945`, the effective served file is `/Users/justin/dev/EdenDocs/favicon.ico` (1150 bytes, exists). **This is not a `--with-app-branding` consumer at all** — the branding-dir copy mechanism only touches `browser/dist/`, never the repo root. BRAND-04's favicon replacement therefore requires either (a) an upstream binary-file overwrite in place (a genuine merge-conflict hotspot — binary diffs can't be trivially re-merged) or (b) a post-build overlay step added to our own `scripts/eden/build.sh` (fully additive, recommended) that copies `eden-branding/favicon.ico` over the built/staged `favicon.ico` after `make` runs, before packaging/smoke-test. Recommend (b).

## 6. Trademark Surface Inventory

### Newly discovered scope-expansion findings (beyond the 2 named `<title>` patches)

1. **`browser/src/control/backstage/Sidebar.tsx:69-76`** — inside `namespace BackstageTemplates`, the `header()` function:
   ```tsx
   export function header(props: HeaderProps): HTMLElement {
       return (
           <div class="backstage-header">
               <span class="backstage-header-title">Collabora Office</span>
               ...
   ```
   Confirmed (via `Control.BackstageView.ts` grep, lines ~35,54,70,88,128-196) this is the **always-shown** "File" panel (Info/Save/SaveAs/Print/Share/Templates tabs) — not gated behind any build flag, not mobile/WASM-only. This is a real, prominent, always-visible hardcoded trademark string requiring its own isolated source edit — a genuine third patch beyond BRAND-02's named two.
2. **`browser/admin/adminSettings.html:137,142`** — `<h5><b>Collabora Online</b></h5>` / `<h5><b>Collabora Office Engine</b></h5>` in the admin console's "Version" info tab. Static markup, not `l10nstrings`-driven, does not respond to `brandProductName`. Requires a source edit (recommend "EdenDocs (coolwsd)" / "LibreOffice Core Engine" — technically accurate without the Collabora name).
3. **`browser/welcome/welcome.html`** (+ `welcome.css/js/svg`, `slide*.png`) — extensively Collabora-branded (`<title>Collabora Online Welcome</title>`, "Collabora Online Development Edition", links to collaboraonline.com/collaboraoffice.org). Confirmed via `browser/Makefile.am:141-142,1176` this compiles **unconditionally** into `browser/dist/welcome/` for every build (it's an unconditional `cool.html` prerequisite), while `--with-app-branding`'s `welcome/` overlay only applies `if ENABLE_MACOS` (`browser/Makefile.am:1045-1047`). For the default web/Linux build this file is a **directly fetchable static asset with full Collabora branding**, independent of the `welcome.enable` config flag (which only suppresses *auto-display*, not existence). Not named in the objective brief. Recommend: extend `scripts/eden/build.sh` with an unconditional post-`make -C browser` overlay step copying `eden-branding/welcome/*` onto `browser/dist/welcome/*` — additive, no upstream edit, no `ENABLE_MACOS` dependency.
4. **`browser/images/collabora-office-white.svg`** — actual Collabora logo file, referenced only by `browser/css/backstage.css:56-58,81-88` (`.backstage-header-title::before { background-image: url('images/collabora-office-white.svg'); }`). Fixed for free by `--with-app-branding`'s `images/*.svg` same-filename-overwrite glob (§1) — no source edit needed, just ship `eden-branding/images/collabora-office-white.svg` containing the Eden emblem.
5. **`browser/images/coda-collab-collaborative-editing-{dark,light}.svg`** — used only in `browser/src/control/Permission.js:378,410`, gated behind `--with-wasm-fallback` (not passed by Objective 1's build; confirmed dormant). Lower priority; note for completeness, no action required unless wasm-fallback is enabled later.

### Verified clean (no action needed)

- `browser/html/cool-help.html` — grep for "collabora" (case-insensitive) returns 4 hits, all false positives from the substring "Collabora**tive**"/"collaborat**ively**" — zero actual brand mentions. Help dialog title itself already uses `brandProductName` (`Toolbar.js:743-751`).
- `net/Socket.cpp:1997-2010` — `getAgentString()`/`getServerString()` return `"COOLWSD HTTP Agent " COOLWSD_VERSION` / `"COOLWSD HTTP Server " COOLWSD_VERSION` — internal protocol identifier `COOLWSD`, not "Collabora". Matches REQUIREMENTS.md's deferred `BRAND-07` scope exactly; `security.server_signature` defaults `false` so the server string isn't even sent by default. Out of scope for this objective, correctly deferred.
- `discovery.xml` — grepped for "collabora": zero matches.
- `docker/from-source/Ubuntu`, `docker/from-source/Debian` (the files Objective 1's pipeline actually builds/DIST-01 will package) — zero `LABEL` directives; only non-runtime comments mention "Collabora Online". Nothing to strip here. (`docker/from-packages/Dockerfile`, not used by this project's pipeline, has `LABEL author="Collabora Productivity Ltd."` etc. — useful only as a naming template for a later objective's own LABEL scheme, not an in-scope edit target.)

### Accepted exception (documented, not fixed in v1)

Scattered third-party technical-documentation hyperlinks across ~9 files, all pointing at `sdk.collaboraonline.com`, `help.collaboraoffice.com`, `forum.collaboraonline.com`, `collaboraonline.com`, or `gerrit.collaboraoffice.com`: `browser/src/errormessages.js:40`, `browser/src/canvas/sections/FormulaErrorHelpSection.ts:81`, `browser/src/slideshow/PresenterConsole.js:937-938`, `browser/src/control/Control.Menubar.ts:2574`, `browser/src/control/Control.ServerAuditDialog.ts` (12 occurrences), `browser/html/framed.doc.html:119`, `browser/admin/src/AdminSocketSettings.js:93`, `browser/admin/src/AdminSocketServerAudit.js` (4 occurrences). **Recommendation: treat as an accepted trademark-checklist exception for v1** — these are genuine third-party technical reference links (EdenDocs doesn't fork the wsd/browser protocol, so the docs remain technically accurate), scattered across 9 upstream files (high edit/merge-conflict cost against the additive-first mergeability rule), and no Eden-equivalent documentation exists yet to redirect to. Document explicitly in the checklist as "reviewed, accepted, revisit when Eden publishes admin/integrator docs" rather than silently ignoring.
Also default `HELP_URL` (`configure.ac:2461-2472`, `https://help.collaboraoffice.com/help.html?`, wired to `coolwsd.xml`'s `help_url` key) is explicitly named in BRAND-05's own text ("help_url") — **this one is must-fix, not an accepted exception**: pass `--with-help-url=<eden-help-url>` if EdenDocs has help content, else `--without-help-url` to disable the help menu link entirely rather than pointing at a live Collabora domain.

### String taxonomy (for the written trademark/MPL checklist)

| Class | Definition | Examples | Action |
|---|---|---|---|
| **User-facing branding — MUST change** | Visible product identity: names, logos, titles | `cool.html.m4:42` `<title>`, `admintemplate.html:12` `<title>`, `Sidebar.tsx:69-76` "Collabora Office", `adminSettings.html:137,142` version headers, `collabora-office-white.svg`, `favicon.ico`, `welcome.html` | Fix via patches/config/branding-dir per §1-§5 |
| **Default config values pointing at live Collabora infra — MUST change** | `help_url`, `INFO_URL` default, `HELP_URL` default | `coolwsd.xml.in:413`, `configure.ac:2461-2472` | Override via configure flags |
| **Internal protocol identifiers — MUST NOT change** | Non-user-visible identifiers whose value matters for wire compatibility / upstream mergeability | `COOLWSD` in `Socket.cpp` Agent/Server strings, internal C++ class/namespace names (`COOLWSD`, `wsd/`, `cool.html`/`coolwsd.xml` filenames themselves) | Leave untouched — changing risks upstream-merge conflicts and protocol confusion, provides zero user-facing branding value |
| **Third-party technical documentation — accepted exception** | Genuine reference links to still-accurate third-party docs, scattered, high merge cost | sdk.collaboraonline.com / forum.collaboraonline.com links (9 files) | Document as reviewed exception; revisit v2 |
| **Legal attribution — MUST preserve verbatim** | MPL notices, license files | `COPYING` (3 lines, pure MPL-2.0 notice), `THIRDPARTYLICENSES` (456 lines), `CODA-THIRDPARTYLICENSES.html` (211 lines), `README.FILENOTICES.md`'s required per-file MPL header convention | Never edit/delete; new `eden-branding/*` files get an analogous MPL-2.0 header with an AO Cyber Systems copyright line, per `README.FILENOTICES.md`'s documented convention |
| **Judgment call — flag, don't auto-fix** | Upstream-provenance links whose removal isn't clearly required or beneficial | Gerrit git-hash hyperlink in `Control.AboutDialog.ts` and `AdminSocketSettings.js:93` | Leave for v1; not named in BRAND-05's explicit surface list |

## 7. Phone-Home Audit Points

- `configure.ac:1270-1339` — `FEEDBACK_URL`/`INFOBAR_URL` both default empty (`AC_DEFINE_UNQUOTED` only fires a value if `--with-feedback-url`/`--with-infobar-url` passed). `config.h.in:79,100` confirm these are `#undef` by default (i.e., become empty-string C macros, never populated). **Confirmed: simply never passing these flags is sufficient** — no additional "unset" step needed, current `scripts/eden/build.sh` invocation already omits them.
- `configure.ac:419-421,2065-2078` — `--with-support-public-key`: if unset, `ENABLE_SUPPORT_KEY=0` and `common/support-public-key.hpp` is actively removed if present. Confirmed never passed in `scripts/eden/build.sh`.
- `configure.ac:307-310,1270-1339` — `--with-welcome-url`: if unset, `ENABLE_WELCOME_MESSAGE=0` (compiled false) **and** (critically, see §2) this is the same branch that generates BRAND-01's `CONFIG_INTERFACE_FRAGMENT`. Confirmed never passed.
- `wsd/FileServer.cpp:1630-1631` — `%FEEDBACK_URL%`/`%WELCOME_URL%` substituted from the corresponding C macros; both empty by default, matching the above.
- Objective 1's `scripts/eden/build.sh` current exact configure invocation (verified by direct read):
  ```bash
  ./configure --enable-silent-rules --enable-debug \
    --with-lokit-path="$PWD/engine/include" \
    --with-lo-path="$PWD/engine/instdir" \
    LDFLAGS="-L/usr/local/lib" 2>&1 | tee "$CONFIGURE_LOG"
  ```
  None of Objective 1's existing flags interact with any Objective 2 flag (`--enable-debug`/`--with-lokit-path`/`--with-lo-path`/`LDFLAGS` are engine/build-tooling concerns, entirely orthogonal to branding/vendor/welcome/help-url namespace). New flags append cleanly.

## 8. Build/CI Integration + Verification

- Confirmed compatible: Objective 1's `make -j"$(nproc)" -C browser` (no explicit target = default `all`) already triggers `all-local: build-cool`, which is where `APP_HAS_BRANDING`'s copy logic lives (§1). **No change to the `make` invocations is needed, only the `./configure` line.**
- `ccache` (Objective 1's `CC="ccache gcc" CXX="ccache g++"`) only affects C/C++ object compilation; branding lives entirely in the `browser/` npm/m4/Make pipeline and is untouched by ccache — **not a staleness risk** for this objective, contrary to the initial assumption in the task brief. The real staleness risks are the two documented above (dark-theme CSS override order, welcome.html overlay gap), not ccache.
- `make build-nocheck` (`all-am`, Objective 1's documented finding) still does **not** recurse into `browser/` — branding verification must always run after the existing explicit `make -C browser` step, exactly as Objective 1 already sequences it.
- `.m4` regeneration: `browser/Makefile.am:1174-1197`'s `$(DIST_FOLDER)/cool.html:` rule lists `$(srcdir)/html/cool.html.m4` as an explicit prerequisite — editing the `<title>` line is correctly detected by Make's mtime check and triggers a rebuild on the next `make -C browser`, no manual cache-busting needed.

### Verification-command candidates (BRAND-01 through BRAND-06)

All checks run against native port **9980** (never 8080), reusing `scripts/eden/smoke-test.sh`'s existing curl+grep pattern.

| Req | Command | Expects |
|---|---|---|
| BRAND-01 | `curl -fsS http://127.0.0.1:9980/hosting/discovery \| grep -q ...` (coolwsd.xml key presence isn't discovery-visible; instead) `curl -fsS "http://127.0.0.1:9980/browser/dist/cool.html" \| grep -q 'branding.css'` **and** grep the running config: `grep -A1 'brandProductName' coolwsd.xml \| grep -q EdenDocs` | `branding.css`/`branding.js` referenced in served HTML; `brandProductName` = EdenDocs in the effective config |
| BRAND-01 (files) | `test -f browser/dist/branding.css && test -f browser/dist/branding.js && test -f browser/dist/images/toolbar-bg.svg` | additive files landed in `browser/dist/` |
| BRAND-02 | `grep -q '<title>EdenDocs' browser/dist/cool.html` and `curl -fsS http://127.0.0.1:9980/browser/dist/admin/admintemplate.html \| grep -q '<title>EdenDocs'` (adjust path to actual served admin route) | both static titles patched |
| BRAND-02 (About) | Headless/manual: load editor, open About dialog, assert DOM text contains "EdenDocs" and not "Collabora" | product name + vendor/copyright correct |
| BRAND-03 | `curl -fsS <admin-console-url> \| grep -o '<title>[^<]*</title>'` → expect `EdenDocs - Admin console`; also `grep -q 'window.brandProductName = "EdenDocs"' browser/dist/branding.js` | admin console title/header driven correctly |
| BRAND-04 | `curl -fsS http://127.0.0.1:9980/favicon.ico \| cmp - eden-branding/favicon.ico` (byte-identical); `grep -q -- "--color-primary: #D4A853" browser/dist/branding.css` (or computed-style check via headless browser for the `!important` override) | favicon overlay + gold palette active, including dark-mode (`?darkTheme=true`) |
| BRAND-05 | Static grep sweep: `grep -rn -i "collabora" browser/dist/ --include=*.html --include=*.js --include=*.css \| grep -v -f accepted-exceptions.txt` → expect zero unexplained hits; `grep -c "MPL-2.0" COPYING THIRDPARTYLICENSES` unchanged from pre-rebrand baseline | trademark sweep passes against the checklist; legal files untouched (diff against git baseline) |
| BRAND-06 | `grep -q "FEEDBACK_URL=$" "$CONFIGURE_LOG"` (or equivalent: assert configure log shows empty/default), `! grep -q -- "--with-support-public-key" scripts/eden/build.sh`, `! grep -q -- "--with-welcome-url" scripts/eden/build.sh`, `grep -q "<brandProductName" coolwsd.xml` (proves the welcome-url-off dependency chain held) | all four phone-home flags confirmed absent; brand keys confirmed present as proof the dependency didn't silently break |

## Proposed `eden-branding/` Directory Layout

```
eden-branding/
├── branding.css          # !important overrides for --color-primary* (+ -text, -dark, -darker,
│                          #   -lighter), --color-btn-primary-hover-bg, --color-overlay.
│                          #   MUST use !important (see Pitfall 2 — dark-theme injection order).
│                          #   MPL-2.0 header per README.FILENOTICES.md convention.
├── branding.js            # window.brandProductName = "EdenDocs";  <- MANDATORY for BRAND-03
│                          #   (admin console has no other path to this value, see §4).
│                          #   Optionally: window.brandProductURL, logo onclick handler
│                          #   (cool.html.m4:349 comment invites this: "logo onclick handler").
│                          #   MPL-2.0 header.
├── images/
│   ├── collabora-office-white.svg   # MUST be this exact filename — overwrites the Backstage
│                                     #   logo via browser/Makefile.am's images/*.svg glob (§1, §6.4)
│   └── toolbar-bg-logo.svg          # source name; copied+renamed to images/toolbar-bg.svg (§1)
├── welcome/                # NOT auto-wired by --with-app-branding on Linux (ENABLE_MACOS-gated,
│                          #   §6.3) — must be overlaid by an explicit scripts/eden/build.sh step,
│                          #   not relied upon via configure alone.
│   ├── welcome.html
│   ├── welcome.css
│   ├── welcome.svg (Eden/AO emblem)
│   └── (slide*.png as needed, or a trimmed single-slide version)
└── favicon.ico            # NOT copied by --with-app-branding at all (§5) — overlaid by
                           #   scripts/eden/build.sh directly onto the served favicon.ico path
                           #   (repo root in dev builds; installed path in packaged builds).
```

Logo source: download the real AO gold-gradient emblem from `https://aocyber.ai/images/ao-icon.svg` (per PROJECT.md's locked decision) and derive `collabora-office-white.svg`'s replacement and `welcome.svg` from it — never a text placeholder.

## Exact Configure-Flag Set

```bash
--with-app-branding="$PWD/eden-branding" \
--with-app-name="EdenDocs" \
--with-vendor="AO Cyber Systems" \
--without-help-url            # or --with-help-url="<eden-help-url>" once Eden help content exists
```

**Confirmed omit list (BRAND-06, verified never passed anywhere in the repo):** `--with-welcome-url`, `--with-feedback-url`, `--with-infobar-url`, `--with-support-public-key`. Recommend a loud, permanent comment (matching Objective 1's `build.sh` documentation style) at the configure call site explaining the `--with-welcome-url` ↔ `brandProductName`-key dependency from §2, so a future contributor doesn't "helpfully" enable the welcome URL and silently break BRAND-01.

`--with-info-url` was considered but left unset intentionally — `INFO_URL` defaults to `https://www.collaboraoffice.org/` and only feeds `VENDOR`'s sibling m4-substituted token in `cool.html.m4`'s mobile-app path (`MOBILEAPPNAME`/`INFO_URL` block, lines 60-78) which is not exercised for the web build (`MOBILEAPP` is false); it's also **not** the same as the coolwsd.xml `brandProductURL` key (which comes from `CONFIG_INTERFACE_FRAGMENT`, §2, and defaults to `https://www.collaboraonline.com` if left empty in coolwsd.xml — this default DOES need overriding by setting `brandProductURL` explicitly in coolwsd.xml, since `--with-info-url` doesn't reach it). **Action item for planning**: coolwsd.xml's `brandProductURL` default value needs its own explicit override (e.g. to `https://aocyber.ai` or an EdenDocs marketing URL) since leaving it empty falls back to the Collabora default per `configure.ac:1270-1339`'s literal fragment text.

## Common Pitfalls

### Pitfall 1: `--with-welcome-url` silently deletes BRAND-01's coolwsd.xml keys
**What goes wrong:** Someone passes `--with-welcome-url` (e.g. to re-enable a welcome splash later) and `brandProductName`/`brandProductURL`/`logoURL` vanish from the generated `coolwsd.xml`.
**Why it happens:** `configure.ac:1270-1339`'s `CONFIG_INTERFACE_FRAGMENT` is only generated in the `else` (no-welcome-url) branch.
**How to avoid:** Loud comment at the configure call site; a build-script assertion (`grep -q "<brandProductName" coolwsd.xml`) mirroring Objective 1's POCO-fallback assertion pattern.
**Warning signs:** Editor silently reverts to "Collabora Online Development Edition (unbranded)" with no error.

### Pitfall 2: Dark-theme toggle reverts branding.css's color overrides
**What goes wrong:** `?darkTheme=true` visually shows Collabora blue again instead of Eden gold.
**Why it happens:** `cool.html.m4:116-128`'s inline script `document.head.appendChild()`s `color-palette-dark.css` at runtime, landing after our static branding `<link>` in cascade order.
**How to avoid:** Declare all `--color-primary*` overrides in `eden-branding/branding.css` with `!important`.
**Warning signs:** Light mode looks correctly gold-branded; dark mode reverts to blue.

### Pitfall 3: Favicon and welcome.html are NOT covered by `--with-app-branding`
**What goes wrong:** Assuming the branding-dir mechanism handles everything named in BRAND-04, the favicon and the always-compiled `welcome.html` ship unbranded.
**Why it happens:** `handleFaviconRequest` (`ClientRequestDispatcher.cpp:1755-1780`) is a hardcoded filesystem lookup with no branding-dir integration; `welcome/` overlay is `if ENABLE_MACOS`-gated only (`browser/Makefile.am:1045-1047`).
**How to avoid:** Add explicit post-`make -C browser` overlay steps to `scripts/eden/build.sh` for both, unconditionally (not gated to macOS).
**Warning signs:** `curl http://127.0.0.1:9980/favicon.ico` still returns the Collabora icon; `browser/dist/welcome/welcome.html` still says "Collabora Online Welcome".

### Pitfall 4: Objective brief undercounts hardcoded-string patches
**What goes wrong:** Treating BRAND-02's "two isolated `<title>` patches" as the complete list of required source edits.
**Why it happens:** `Sidebar.tsx:69-76`'s "Collabora Office" backstage header and `adminSettings.html:137,142`'s two version-tab headers are hardcoded, always-visible, and not named in the brief.
**How to avoid:** Plan for 3-4 isolated single-purpose source-edit commits, not 2, each independently revertable across upstream merges (consistent with the project's stated mergeability philosophy).
**Warning signs:** Trademark sweep (BRAND-05 grep) still finds "Collabora" in `browser/dist/bundle.js` (compiled Sidebar.tsx) or `browser/dist/admin/adminSettings.html` after the two named patches are done.

### Pitfall 5: `browser/dist` image-glob overwrite depends on identical filenames
**What goes wrong:** Shipping the Eden logo under a different filename in `eden-branding/images/` does nothing — the old Collabora SVG remains referenced by `backstage.css`'s literal `url('images/collabora-office-white.svg')`.
**How to avoid:** The replacement file in `eden-branding/images/` **must** be named exactly `collabora-office-white.svg` (matching the CSS reference) — this is an intentional filename collision, not a bug, and should be commented as such in the branding dir so a future maintainer doesn't "helpfully" rename it to something more honest-sounding.

## Implications for Planning (suggested task-breakdown seams)

1. **`eden-branding/` directory + configure-flag wiring** — create the additive directory (branding.css with `!important` gold overrides, branding.js with `window.brandProductName`, images/collabora-office-white.svg + toolbar-bg-logo.svg sourced from the real AO emblem), add the 4 configure flags to `scripts/eden/build.sh`, add the loud `--with-welcome-url` dependency comment + assertion. Delivers BRAND-01 core mechanism + half of BRAND-04 (palette) + BRAND-03's actual product-name propagation.
2. **coolwsd.xml explicit `brandProductURL` override** — set it explicitly (don't rely on the empty-string default, which falls back to `https://www.collaboraonline.com` per §2/"Exact configure-flag set"). Small, bundle with task 1 or its own one-liner.
3. **Isolated title patches** — `cool.html.m4:42` and `admintemplate.html:12`, each its own commit (BRAND-02).
4. **Backstage header patch** — `Sidebar.tsx:69-76` "Collabora Office" → "EdenDocs", its own commit (newly discovered, fold into BRAND-02 or BRAND-05 per planner's judgment — recommend BRAND-05 since it's a trademark-removal action, not a title-surface action).
5. **Admin Version-tab patch** — `adminSettings.html:137,142`, its own commit (BRAND-05).
6. **Favicon + welcome.html build-script overlay** — extend `scripts/eden/build.sh` with two unconditional post-`make -C browser` copy steps (favicon to served path, `eden-branding/welcome/*` onto `browser/dist/welcome/*`). No upstream file edits. Delivers rest of BRAND-04.
7. **`--with-help-url`/`--without-help-url` decision + wiring** — small, part of task 1's flag set (BRAND-05's `help_url` clause).
8. **Trademark/MPL checklist authoring + verification pass** — write the actual checklist document (using the taxonomy table in §6 as its backbone), run the BRAND-01..06 verification commands from §8 against a built tree, confirm `COPYING`/`THIRDPARTYLICENSES`/`CODA-THIRDPARTYLICENSES.html` are byte-identical to the pre-rebrand baseline (`git diff --stat` on those three paths should be empty). This is the task that actually proves BRAND-05.
9. **No-phone-home verification** — cheap, can be folded into task 8's verification pass rather than being its own task: grep `scripts/eden/build.sh` for the 4 forbidden flags (already true today, just needs a permanent negative-assertion test) + confirm `coolwsd.xml`'s brand keys are present as the canary for Pitfall 1.

Sequencing note: tasks 3-5 (source edits) are the highest upstream-merge-conflict-risk items and should land as small, easily-rebasable commits early, before task 1's larger additive work, so any merge-conflict resolution needed after a weekly upstream sync only ever touches single-line diffs.

## Open Questions

1. **Gerrit git-hash provenance link** (`Control.AboutDialog.ts`, `AdminSocketSettings.js:93`)
   - What we know: hardcoded `https://gerrit.collaboraoffice.com/plugins/gitiles/online/+log/<hash>` links, likely to 404 for any eden-main-only commit.
   - What's unclear: whether this counts as a "Collabora trademark" under BRAND-05's UI-strings clause, or as acceptable/desirable upstream-provenance attribution.
   - Recommendation: leave as-is for v1 (not named in BRAND-05's explicit surface list); revisit if the user wants full attribution-link removal.
2. **SDK/forum/community documentation links** (9 files, listed in §6)
   - What we know: genuine, still-technically-accurate third-party docs; high edit/merge cost if touched.
   - What's unclear: whether product/legal wants zero Collabora-domain references anywhere, even in "learn more" links.
   - Recommendation: accepted exception for v1, explicitly documented in the checklist, not silently ignored.
3. **`--with-help-url` value** — no Eden-hosted help content is known to exist yet.
   - Recommendation: `--without-help-url` (disables the help-menu link cleanly) until/unless Eden publishes help content, then switch to `--with-help-url`.
4. **Backstage/admin hardcoded-string patches — fold into BRAND-02 or BRAND-05?**
   - What we know: both are genuine, newly-discovered required edits.
   - Recommendation left to planner: this research recommends filing them under BRAND-05 (trademark removal) since they're branding-hygiene fixes rather than title-surface fixes, but either mapping is defensible.

## Sources

### Primary (HIGH confidence — direct reads of this checkout)
- `configure.ac` (multiple ranges: 222-300, 419-421, 1112-1339, 2065-2078, 2386-2428, 2461-2472)
- `config.h.in` (grep for APP_NAME/FEEDBACK_URL/INFOBAR_URL/WELCOME_URL)
- `coolwsd.xml.in` (lines 54, 58, 276-285, 403, 413)
- `wsd/FileServer.cpp` (lines 1279-1280, 1560-1631, 2428-2521, 2523-2622, 2624-2695)
- `wsd/ClientRequestDispatcher.cpp` (lines 1755-1780)
- `wsd/COOLWSD.cpp` (lines 3935-3945)
- `net/Socket.cpp` (lines 1997-2010)
- `browser/Makefile.am` (lines 9, 123-142, 248-249, 673-681, 938-940, 1014-1247, 1328-1334, 1428-1434)
- `browser/html/cool.html.m4` (lines 1-350, full read across ranges)
- `browser/admin/admintemplate.html` (full read, 86 lines)
- `browser/admin/adminBody.html`, `adminClusterBody.html`, `adminSettings.html:120-155`, `admin.strings.js:14`
- `browser/js/global.js` (lines 317-327, 473-474, 516-518)
- `browser/src/control/Control.AboutDialog.ts` (full read, 503 lines)
- `browser/src/control/backstage/Sidebar.tsx` (full read, 145 lines)
- `browser/src/control/Control.Menubar.ts` (lines 2670-2715, 2574)
- `browser/src/control/Toolbar.js` (lines 740-770)
- `browser/src/app/LOUtil.ts` (lines 985-1013)
- `browser/css/color-palette.css`, `color-palette-dark.css` (full custom-property grep)
- `browser/css/backstage.css` (lines 56-58, 81-88)
- `browser/welcome/welcome.html` (grep for Collabora strings)
- `browser/html/cool-help.html` (grep verification — clean)
- `docker/from-source/Ubuntu`, `docker/from-source/Debian`, `docker/from-packages/Dockerfile` (lines 181-188)
- `COPYING`, `README.FILENOTICES.md` (full reads), `THIRDPARTYLICENSES`/`CODA-THIRDPARTYLICENSES.html` (line counts only)
- `.planning/PROJECT.md`, `.planning/REQUIREMENTS.md` (BRAND-01..06 verbatim), `.planning/objectives/01-from-source-build-pipeline/01-01-SUMMARY.md`, `01-02-SUMMARY.md`
- `scripts/eden/build.sh`, `scripts/eden/smoke-test.sh` (full reads)

### Secondary / Tertiary
None used — this objective required zero external library research (no Context7/WebSearch calls made); every finding is grounded in this specific checkout's source code.

## Metadata

**Confidence breakdown:**
- Branding wiring / coolwsd.xml keys / title-about surfaces: HIGH — every claim traced to an exact file:line in this checkout, cross-verified against the actual Makefile recipe bodies and C++ substitution code, not documentation.
- Admin console `%BRANDING_JS%` mechanism: HIGH — traced the full producer/consumer chain, including the non-obvious "brandProductName is a bare global with no other producer" finding.
- Trademark inventory: HIGH for the systematically-grepped surfaces (UI strings, images, help_url, Dockerfiles); MEDIUM for completeness claims (a grep sweep can miss dynamically-constructed strings) — recommend the BRAND-05 verification task re-run a fresh grep sweep against the actual built `browser/dist/` output, not just source, before declaring the checklist satisfied.
- Color palette recommendation: MEDIUM on the exact hex mapping (a design decision, not a research fact) — HIGH on the underlying mechanism (cascade order, `!important` requirement, dark-theme pitfall).

**Research date:** 2026-07-08
**Valid until:** Until the next `upstream/main` merge touches any of the cited files (weekly-minimum cadence per PROJECT.md) — re-verify file:line citations after each merge, especially `configure.ac`, `browser/Makefile.am`, and `cool.html.m4`, which are the highest-churn files in this research.
