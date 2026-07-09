---
objective: 04-edendocs-branded-container-images
trd: "01"
type: standard
wave: 1
depends_on: []
files_modified:
  - scripts/eden/build-release.sh
  - scripts/eden/fetch-engine-pinned.sh
  - docker/eden/engine-pin.env
autonomous: true
requirements: [DIST-01]
# TDD-EXCEPTION: infra/packaging work — shell build+fetch scripts. The runnable
# verification gate (shellcheck, sha256 fail-closed check, no-debug/no-phone-home
# greps, and the CI boot-health gate in 04-04) is the test. No unit-testable
# business logic; the resolver's (app,feature) widget/Maestro TDD profile does not apply.

must_haves:
  truths:
    - "scripts/eden/fetch-engine-pinned.sh fetches an org-controlled engine snapshot and HARD-FAILS (exit 1) unless its sha256 equals the recorded pin — it also hard-refuses the known-SIGSEGV digest 076a4129… regardless of what the mirror serves"
    - "docker/eden/engine-pin.env records the pinned sha256 (default: the known-good ea91ef33…) so a signed release image is rebuildable from a fixed input set"
    - "scripts/eden/build-release.sh runs a NON-debug configure (no --enable-debug), sets --with-lo-path=/opt/collaboraoffice (the container runtime path, not a build-host path), applies the EdenDocs branding flags, and never passes any of the four phone-home flags"
    - "build-release.sh stages a complete docker build context at docker/eden/instdir/: coolwsd + coolforkit-caps + coolmount under usr/bin, the pinned engine under opt/collaboraoffice, EdenDocs branding overlays (favicon + welcome), and an EdenDocs-injected etc/coolwsd/coolwsd.xml"
  artifacts:
    - "scripts/eden/fetch-engine-pinned.sh"
    - "scripts/eden/build-release.sh"
    - "docker/eden/engine-pin.env"
  key_links:
    - "fetch-engine-pinned.sh → docker/eden/engine-pin.env (ENGINE_PIN_SHA256 fail-closed verify; MIRROR_ENGINE_URL override for re-pin)"
    - "build-release.sh → docker/eden/instdir/ (the exact COPY /instdir / context docker/eden/Dockerfile in 04-02 consumes — usr/bin/cool* + opt/collaboraoffice)"
    - "build-release.sh --with-lo-path=/opt/collaboraoffice → docker/eden/Dockerfile's engine location (Pitfall 4: non-debug bakes the future runtime path, not the build-host tree path)"
---

<objective>
Produce the two Eden-owned release-build scripts and the engine pin record that turn EdenDocs source + a *pinned, boot-validated* engine snapshot into a reproducible docker build context — the DIST-01 "from-source, not from-packages" foundation.

`scripts/eden/fetch-engine-pinned.sh` is the release-path counterpart to the existing (deliberately unpinned) `fetch-engine-assets.sh`: it fetches an org-controlled immutable mirror and fail-closes on any sha256 drift, so a signed image can never be built against the volatile rolling `for-code-assets` tarball (one version of which SIGSEGVs kit startup — STATE.md). `scripts/eden/build-release.sh` is the NON-debug counterpart to `scripts/eden/build.sh`: same branding flags and HARD OMIT LIST, but no `--enable-debug` and a container-runtime `--with-lo-path`, staging everything into `docker/eden/instdir/`.

Purpose: These are new, additive files (zero upstream edits). They give 04-02's Dockerfile a `COPY /instdir /` context in the exact upstream from-source shape, and give 04-04's CI a pinned, reproducible input set to sign.

Output: `scripts/eden/build-release.sh`, `scripts/eden/fetch-engine-pinned.sh`, `docker/eden/engine-pin.env` — committed on eden-main.
</objective>

<file_tree>
scripts/eden/
├── build-release.sh          ← CREATE (non-debug release build → docker/eden/instdir)
├── fetch-engine-pinned.sh    ← CREATE (pinned, sha256-verified engine fetch)
├── build.sh                    (untouched — debug/CI dev loop)
└── fetch-engine-assets.sh      (untouched — rolling/unpinned dev loop)
docker/eden/
└── engine-pin.env            ← CREATE (recorded pin sha256 + mirror URL)
</file_tree>

<execution_context>
@~/.claude/devflow/workflows/execute-trd.md
@~/.claude/devflow/templates/summary.md
</execution_context>

<embedded_context>

<codebase_examples>
`scripts/eden/build.sh` is the debug/CI-loop sibling — copy its structure (ccache wiring, brandProductName canary, python brand-key injection, loud artifact assertions) but change three things for release: drop `--enable-debug`, use FHS prefixes + `--with-lo-path=/opt/collaboraoffice`, and stage into `docker/eden/instdir` via `DESTDIR make install` instead of a dev-tree build. Its branding-flag block (verbatim, keep):

```bash
./configure --enable-silent-rules \
  --with-lokit-path="$PWD/engine/include" \
  --with-lo-path="$PWD/engine/instdir" \        # DEBUG-ONLY real path — REPLACE for release
  --with-app-branding="$PWD/eden-branding" \
  --with-app-name="EdenDocs" \
  --with-vendor="AO Cyber Systems" \
  --with-help-url="https://aocyber.ai" \
  LDFLAGS="-L/usr/local/lib"
# HARD OMIT LIST — never pass: --with-welcome-url --with-feedback-url
#   --with-infobar-url --with-support-public-key   (BRAND-06 / Pitfall 1)
```

`scripts/eden/fetch-engine-assets.sh` shows the fetch/extract shape and the `engine/instdir/program/versionrc` sanity check — reuse the extract + sanity logic, ADD hard sha256 verification (the dev-loop script deliberately does not verify; the release path must).

The upstream from-source instdir contract (`docker/from-source/build.sh` lines 104-115 + `docker/from-source/Ubuntu` line 22 `COPY /instdir /`): the context root contains `instdir/` with `usr/bin/cool*` and `opt/collaboraoffice/` (engine). build-release.sh must reproduce exactly this shape at `docker/eden/instdir`.
</codebase_examples>

<anti_patterns>
- **NEVER pin `076a41297c2ed659638617ed27cb6d4cff01f1558f36fb0e0b1abd6f4a14868e`** — this rolling engine version SIGSEGVs kit startup (forkCOKit/lokit_main), coolwsd never accepts on 9980 (STATE.md, CI run 28964794754). fetch-engine-pinned.sh must hard-refuse it even if the mirror is mis-populated with it.
- **No `--enable-debug` in the release path** (Pitfall 3): `configure.ac` documents debug mode as disabling "some security pieces". `--enable-debug` exists in build.sh only to arm the build-time `--with-lo-path` existence check for CI — wrong for what ships.
- **No `--with-lo-path="$PWD/engine/instdir"` in the release path** (Pitfall 4): in non-debug mode `--with-lo-path` is just a compiled-in string for the *future* runtime location and does NOT need to exist at configure time. Bake `/opt/collaboraoffice` (where the container will place the engine), never a build-host tree path.
- **No editing `scripts/eden/fetch-engine-assets.sh` or `scripts/eden/build.sh`** — they serve the fast dev/CI loop; the release path is NEW additive files so both loops stay independent.
- **Never fetch the live rolling `for-code-assets` URL for a publishable image** — that is exactly the unpinned volatility this TRD exists to eliminate (research Pitfall 2). Release builds fetch the org mirror + verify sha256.
- **NEVER reference port 8080** anywhere in these scripts or comments — banned project-wide. These scripts bind no port; keep it that way.
</anti_patterns>

<error_recovery>
- **The known-good ea91ef33… tarball is no longer downloadable** (the rolling `for-code-assets` tag overwrote it upstream) → the mirror must be populated with a boot-validated snapshot. This is the RE-PIN path, executed in 04-04's CI (where a real Linux build+boot happens): fetch the current rolling engine, build+boot-health it; if it accepts on 9980 with no SIGSEGV, record ITS sha256 in engine-pin.env and upload that exact tarball to the mirror. Prefer ea91ef33… if obtainable; otherwise any freshly boot-validated snapshot — never 076a4129….
- **`configure` fails "header Poco/Net/WebSocket.h not found"** → `scripts/eden/build-deps.sh` (system POCO) must run before build-release.sh, same ordering as build.sh. This is a CI step-order fix (04-04), not a script bug.
- **`make install` DESTDIR path wrong / instdir empty** → confirm `--prefix=/usr --sysconfdir=/etc --localstatedir=/var` and `DESTDIR="$PWD/docker/eden/instdir"`; the installed tree must be `docker/eden/instdir/usr/bin/coolwsd` etc. (FHS under DESTDIR).
- **`brandProductName` missing from generated coolwsd.xml** → a welcome/feedback/infobar URL leaked into configure (Pitfall 1). Grep the configure invocation for the HARD OMIT flags; remove and re-run. The canary in this script is the permanent guard — do NOT loosen it.
</error_recovery>

</embedded_context>

<context>
@.planning/PROJECT.md
@.planning/STATE.md
@.planning/objectives/04-edendocs-branded-container-images/4-RESEARCH.md
@scripts/eden/build.sh
@scripts/eden/fetch-engine-assets.sh
@docker/from-source/build.sh
</context>

<research_context>
From 4-RESEARCH.md (HIGH confidence) — Open Question 1 resolved as BOTH (a)+(b):
- (a) Pin a known-good `engine-main-assets.tar.gz` by recorded sha256 fetched from an org-controlled immutable mirror, so the signed image is rebuildable from a fixed input set. Known-good digest: `ea91ef33ba510eb1379568c58afe9ddc54aa05985f4ff963ab8e0a8cdfa6c59f` (smoke-green same day). Known-BAD: `076a4129…` (SIGSEGV — never pin).
- (b) A hard pre-publish boot-health gate (start coolwsd from the built image, confirm it accepts on 9980 / no SIGSEGV, abort publish on crash) — implemented in 04-04's CI as defense-in-depth even with the pin.
- Pattern 1 (release build): non-debug configure/install into `docker/eden/instdir`, engine copied to `opt/collaboraoffice`, branding overlaid into the DESTDIR tree. `--with-lo-path` semantics differ in non-debug (compiled-in future path, no existence check).
- Research "Valid until": 7 days for the engine-pin recommendation specifically (volatile upstream asset). Re-verify the digest/boot-health at execution time; the mirror + fail-closed sha256 is what makes it durable.
</research_context>

<gotchas>
- The pin's *durability* comes from the org mirror, not the recorded digest alone: recording ea91ef33… but fetching the rolling URL would fail sha256 verification (the rolling URL now serves a different tarball). fetch-engine-pinned.sh MUST fetch `MIRROR_ENGINE_URL`, not the upstream rolling tag.
- macOS dev machines cannot build coolwsd (Linux-only, PROJECT.md). These scripts' full build + the pin's boot-health are proven in 04-04's ubuntu-latest CI, not locally. Local verification here is shellcheck + static/no-port/no-debug/no-phone-home greps + sha256-logic inspection only.
- `--disable-tests` keeps the release configure fast and avoids pulling test-only deps into the image context.
</gotchas>

<tasks>

<task type="auto">
  <name>Task 1: Record the engine pin + author the fail-closed pinned fetch script</name>
  <files>docker/eden/engine-pin.env, scripts/eden/fetch-engine-pinned.sh</files>
  <action>
Create `docker/eden/engine-pin.env` (a CHECKSUMS-style record, sourced by the fetch script):

```sh
# EdenDocs release engine pin (DIST-01 — Open Q1 decision (a)+(b)).
# The pinned, boot-validated engine-main-assets.tar.gz snapshot for PUBLISHED images.
# Rebuildable-from-a-fixed-input-set guarantee: a signed release image is always
# built from exactly this tarball, fetched from the org mirror and sha256-verified.
#
# KNOWN-GOOD (default): ea91ef33… (smoke-green same day — STATE.md).
# KNOWN-BAD (NEVER pin): 076a41297c2ed659638617ed27cb6d4cff01f1558f36fb0e0b1abd6f4a14868e
#   SIGSEGVs kit startup (forkCOKit/lokit_main); coolwsd never accepts on 9980.
ENGINE_PIN_SHA256=ea91ef33ba510eb1379568c58afe9ddc54aa05985f4ff963ab8e0a8cdfa6c59f

# Org-controlled immutable mirror of the pinned tarball (populated once in 04-04:
# `gh release create engine-pin` + `gh release upload engine-pin engine-main-assets.tar.gz`).
# A GitHub Release asset gives a stable https URL that CI pulls with only GITHUB_TOKEN.
# Override with MIRROR_ENGINE_URL for a re-pin.
ENGINE_PIN_MIRROR_URL=https://github.com/AO-Cyber-Systems/EdenDocs/releases/download/engine-pin/engine-main-assets.tar.gz
```

Create `scripts/eden/fetch-engine-pinned.sh` (new, additive; release-path counterpart to fetch-engine-assets.sh):

```bash
#!/usr/bin/env bash
# fetch-engine-pinned.sh — RELEASE path: fetch a PINNED, sha256-verified engine.
# Unlike fetch-engine-assets.sh (rolling/unpinned — dev/CI loop), this fetches an
# org-controlled immutable mirror and HARD-FAILS on any sha256 drift, so a signed
# image is never built against the volatile rolling for-code-assets tarball.
# This script binds no network port.
set -euo pipefail
cd "$(dirname "$0")/../.."

# shellcheck source=docker/eden/engine-pin.env
source docker/eden/engine-pin.env
KNOWN_BAD=076a41297c2ed659638617ed27cb6d4cff01f1558f36fb0e0b1abd6f4a14868e
MIRROR_URL="${MIRROR_ENGINE_URL:-$ENGINE_PIN_MIRROR_URL}"

# Fail-closed guard: refuse the known-SIGSEGV digest even if the pin file or the
# mirror is mis-populated with it.
[ "$ENGINE_PIN_SHA256" != "$KNOWN_BAD" ] \
  || { echo "FATAL: pin equals the known-SIGSEGV engine ($KNOWN_BAD) — refusing"; exit 1; }

if [ -d engine/instdir ] && [ "${FORCE_REFETCH:-0}" != "1" ]; then
  echo "engine/instdir already exists — skipping fetch (FORCE_REFETCH=1 to force)."
  exit 0
fi

echo "Fetching PINNED engine from mirror: $MIRROR_URL"
wget --no-verbose "$MIRROR_URL" -O engine-assets.tar.gz

GOT=$(sha256sum engine-assets.tar.gz | awk '{print $1}')
if [ "$GOT" != "$ENGINE_PIN_SHA256" ]; then
  echo "FATAL: fetched engine sha256 mismatch"
  echo "  expected (pin): $ENGINE_PIN_SHA256"
  echo "  got            : $GOT"
  echo "Refusing to build a publishable image against an unpinned engine."
  exit 1
fi
[ "$GOT" != "$KNOWN_BAD" ] || { echo "FATAL: fetched the known-SIGSEGV engine — refusing"; exit 1; }

mkdir -p engine
tar xzf engine-assets.tar.gz -C engine
rm engine-assets.tar.gz
test -f engine/instdir/program/versionrc

echo "Pinned engine verified (sha256=$ENGINE_PIN_SHA256) and extracted to engine/instdir."
```

`chmod +x scripts/eden/fetch-engine-pinned.sh`.
  </action>
  <verify>
chmod +x scripts/eden/fetch-engine-pinned.sh && bash -n scripts/eden/fetch-engine-pinned.sh && command -v shellcheck >/dev/null && shellcheck -x scripts/eden/fetch-engine-pinned.sh || true; grep -q 'ea91ef33ba510eb1379568c58afe9ddc54aa05985f4ff963ab8e0a8cdfa6c59f' docker/eden/engine-pin.env && ! grep -q '^ENGINE_PIN_SHA256=076a4129' docker/eden/engine-pin.env && grep -q 'ENGINE_PIN_MIRROR_URL' docker/eden/engine-pin.env && grep -q 'sha256sum' scripts/eden/fetch-engine-pinned.sh && grep -q 'FATAL' scripts/eden/fetch-engine-pinned.sh && grep -q '076a41297c2ed659638617ed27cb6d4cff01f1558f36fb0e0b1abd6f4a14868e' scripts/eden/fetch-engine-pinned.sh && ! grep -rn '8080' scripts/eden/fetch-engine-pinned.sh docker/eden/engine-pin.env
  </verify>
  <done>engine-pin.env records ea91ef33… (not 076a4129…) plus a mirror URL; fetch-engine-pinned.sh parses (bash -n clean, shellcheck-clean if available), sources the pin, hard-fails on sha256 mismatch, explicitly refuses the known-bad digest, and references no port 8080.</done>
  <recovery>If shellcheck flags a real issue (unquoted expansion, missing source pragma), fix it. If the mirror URL scheme needs to change (OCI artifact via oras instead of a release asset), update both engine-pin.env and the fetch mechanism together — keep the sha256 fail-closed verify intact regardless of transport.</recovery>
</task>

<task type="auto">
  <name>Task 2: Author the non-debug release build script that stages docker/eden/instdir</name>
  <files>scripts/eden/build-release.sh</files>
  <action>
Create `scripts/eden/build-release.sh` (new, additive; non-debug counterpart to build.sh). Model it on build.sh but with the release deltas. Key requirements — reference gotchas/anti_patterns above:

```bash
#!/usr/bin/env bash
# build-release.sh — NON-debug release build → docker/eden/instdir (DIST-01).
# Distinct from scripts/eden/build.sh (--enable-debug, dev/CI loop). Produces the
# COPY /instdir / build context for docker/eden/Dockerfile. Binds no network port.
# Prereqs (CI, 04-04): scripts/eden/build-deps.sh (system POCO) then
# scripts/eden/fetch-engine-pinned.sh (pinned engine) must run BEFORE this.
set -euo pipefail
cd "$(dirname "$0")/../.."

if command -v ccache >/dev/null 2>&1 && [ -z "${CC:-}" ]; then
  export CC="ccache gcc" CXX="ccache g++"
fi

./autogen.sh

# NON-debug + FHS prefixes. --with-lo-path is the FUTURE container runtime path
# (Pitfall 4: not existence-checked in non-debug). HARD OMIT LIST unchanged (BRAND-06).
./configure --prefix=/usr --sysconfdir=/etc --localstatedir=/var \
  --enable-silent-rules --disable-tests \
  --with-lokit-path="$PWD/engine/include" \
  --with-lo-path=/opt/collaboraoffice \
  --with-app-branding="$PWD/eden-branding" \
  --with-app-name="EdenDocs" \
  --with-vendor="AO Cyber Systems" \
  --with-help-url="https://aocyber.ai" \
  LDFLAGS="-L/usr/local/lib"

# BRAND-01/06 canary — brandProductName only survives when no welcome URL is set.
grep -q "<brandProductName" coolwsd.xml \
  || { echo "ERROR: brandProductName missing from coolwsd.xml — a phone-home flag leaked (Pitfall 1)"; exit 1; }

# Inject EdenDocs brand keys into the generated (gitignored) coolwsd.xml.
python3 - <<'PYEOF'
import re, pathlib
p = pathlib.Path('coolwsd.xml'); c = p.read_text()
c = re.sub(r'(<brandProductName[^>]*>)[^<]*(</brandProductName>)', r'\1EdenDocs\2', c)
c = re.sub(r'(<brandProductURL[^>]*>)[^<]*(</brandProductURL>)', r'\1https://aocyber.ai\2', c)
c = re.sub(r'(<logoURL[^>]*>)[^<]*(</logoURL>)', r'\1images/eden-logo.svg\2', c)
p.write_text(c)
PYEOF
grep -q ">EdenDocs</brandProductName>" coolwsd.xml || { echo "ERROR: brandProductName injection failed"; exit 1; }

make -j"$(nproc)" build-nocheck
make -j"$(nproc)" -C browser

INSTDIR="$PWD/docker/eden/instdir"
rm -rf "$INSTDIR"
mkdir -p "$INSTDIR"
DESTDIR="$INSTDIR" make install
mkdir -p "$INSTDIR/opt"
cp -a engine/instdir "$INSTDIR/opt/collaboraoffice"

# EdenDocs branding overlays into the DESTDIR tree (file_server_root_path=/usr/share/coolwsd
# in the container; favicon has no config override; welcome must be fully replaced).
cp eden-branding/favicon.ico "$INSTDIR/usr/share/coolwsd/favicon.ico"
rm -rf "$INSTDIR/usr/share/coolwsd/browser/dist/welcome"
cp -R eden-branding/welcome "$INSTDIR/usr/share/coolwsd/browser/dist/welcome"

# Loud context assertions — the Dockerfile's setcap + systemplate depend on these.
test -x "$INSTDIR/usr/bin/coolwsd"          || { echo "ERROR: instdir coolwsd missing"; exit 1; }
test -x "$INSTDIR/usr/bin/coolforkit-caps"  || { echo "ERROR: instdir coolforkit-caps missing"; exit 1; }
test -x "$INSTDIR/usr/bin/coolmount"        || { echo "ERROR: instdir coolmount missing"; exit 1; }
test -x "$INSTDIR/usr/bin/coolwsd-systemplate-setup" || { echo "ERROR: instdir coolwsd-systemplate-setup missing"; exit 1; }
test -d "$INSTDIR/opt/collaboraoffice/program" || { echo "ERROR: engine not staged at opt/collaboraoffice"; exit 1; }
grep -q ">EdenDocs</brandProductName>" "$INSTDIR/etc/coolwsd/coolwsd.xml" \
  || { echo "ERROR: installed coolwsd.xml missing EdenDocs brandProductName"; exit 1; }
if grep -rqi collabora "$INSTDIR/usr/share/coolwsd/browser/dist/welcome/"; then
  echo "ERROR: welcome overlay still references Collabora"; exit 1
fi

echo "build-release.sh complete: docker/eden/instdir staged (coolwsd + engine + EdenDocs branding)."
```

`chmod +x scripts/eden/build-release.sh`. Do NOT overlay favicon at repo root (that is build.sh's dev-tree behavior); in the container the served path is `/usr/share/coolwsd/favicon.ico`.
  </action>
  <verify>
chmod +x scripts/eden/build-release.sh && bash -n scripts/eden/build-release.sh && command -v shellcheck >/dev/null && shellcheck -x scripts/eden/build-release.sh || true; ! grep -q -- '--enable-debug' scripts/eden/build-release.sh && grep -q -- '--with-lo-path=/opt/collaboraoffice' scripts/eden/build-release.sh && grep -q 'DESTDIR="\$INSTDIR" make install' scripts/eden/build-release.sh && grep -q 'docker/eden/instdir' scripts/eden/build-release.sh && grep -q '<brandProductName' scripts/eden/build-release.sh && for f in --with-welcome-url --with-feedback-url --with-infobar-url --with-support-public-key; do ! grep -q -- "$f" scripts/eden/build-release.sh || { echo "FORBIDDEN FLAG $f"; exit 1; }; done && ! grep -rn '8080' scripts/eden/build-release.sh
  </verify>
  <done>build-release.sh parses clean, contains NO --enable-debug, sets --with-lo-path=/opt/collaboraoffice, stages via DESTDIR make install into docker/eden/instdir, carries the brandProductName canary + injection, passes none of the four phone-home flags, and references no port 8080. (Full Linux build + boot proven in 04-04 CI.)</done>
  <recovery>bash -n / shellcheck failure → fix syntax. If the installed coolwsd.xml lands at a different path than etc/coolwsd/coolwsd.xml, adjust the assertion to the actual COOLWSD_CONFIGDIR (--sysconfdir=/etc ⇒ /etc/coolwsd). If `make install` needs a target that build-nocheck skipped (e.g. browser dist install), add the explicit install step rather than hand-copying dist.</recovery>
</task>

</tasks>

<validation_gates>
<lint>bash -n scripts/eden/build-release.sh scripts/eden/fetch-engine-pinned.sh; command -v shellcheck >/dev/null && shellcheck -x scripts/eden/build-release.sh scripts/eden/fetch-engine-pinned.sh || true</lint>
<test># Full build + pin boot-health is proven in 04-04's ubuntu-latest CI (macOS dev cannot build coolwsd). Local gate = lint + static greps below.
grep -q 'ea91ef33' docker/eden/engine-pin.env && ! grep -q -- '--enable-debug' scripts/eden/build-release.sh</test>
<build>true # heavy native build runs in CI (04-04), not per-task</build>
</validation_gates>

<verification>
1. `docker/eden/engine-pin.env` records ENGINE_PIN_SHA256=ea91ef33… (NOT 076a4129…) and a mirror URL (DIST-01 reproducibility: "pinned sha256 is recorded").
2. `scripts/eden/fetch-engine-pinned.sh` fail-closes on sha256 mismatch and hard-refuses the known-SIGSEGV digest.
3. `scripts/eden/build-release.sh` is non-debug, uses --with-lo-path=/opt/collaboraoffice, stages docker/eden/instdir with cool* binaries + engine + EdenDocs branding, passes no phone-home flag.
4. Neither script references port 8080; both bind no port.
5. Zero upstream files edited — three new additive files only.
</verification>

<success_criteria>
- A reproducible, pinned engine input (recorded sha256, fail-closed fetch) plus a non-debug release build that stages the exact upstream from-source `instdir/` shape for 04-02's Dockerfile — the DIST-01 build foundation, with the pin's actual boot-health proven by 04-04's CI gate.
- The recorded pin is a known-good digest (ea91ef33…), never the SIGSEGV digest (076a4129…).
</success_criteria>

<output>
After completion, create `.planning/objectives/04-edendocs-branded-container-images/04-01-SUMMARY.md`: the recorded pin sha256 + mirror URL, the exact non-debug configure line, the staged instdir layout (key paths), and a note that boot-health of the pin is validated in 04-04's CI gate (with the re-pin fallback if ea91ef33… proves unobtainable at execution time).
</output>
