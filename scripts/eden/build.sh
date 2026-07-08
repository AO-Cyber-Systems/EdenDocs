#!/usr/bin/env bash
#
# build.sh — EdenDocs from-source build pipeline: autogen/configure/make.
#
# Configures and builds EdenDocs (coolwsd) against the prebuilt ENGINE_ASSETS
# tarball (scripts/eden/fetch-engine-assets.sh) and the system POCO installed
# by scripts/eden/build-deps.sh. Run build-deps.sh BEFORE this script — POCO
# install must strictly precede configure, or configure hard-errors with
# "header Poco/Net/WebSocket.h not found" (research Pitfall 1).
#
# This script does not bind or reference any network port.
#
set -euo pipefail

cd "$(dirname "$0")/../.."

# ccache wiring (BUILD-03): root configure.ac has NO Linux ccache
# auto-detection (only a Windows/CODA path around lines 2758-2806), so it
# must be wired externally via CC/CXX. Only set if the caller hasn't
# already exported CC/CXX themselves.
if command -v ccache >/dev/null 2>&1 && [ -z "${CC:-}" ]; then
  export CC="ccache gcc" CXX="ccache g++"
fi

./autogen.sh

# Configure, teeing output to a log so the BUILD-02 fallback assertion
# below can inspect it. --enable-debug is required: it arms the
# --with-lo-path versionrc hard-check (configure.ac:1765-1774), matching
# codeql-analysis.yml's recipe and preventing a silently broken build.
#
# LDFLAGS=-L/usr/local/lib: the system-POCO fallback emits bare -lPoco*
# with no -L (configure.ac:2090-2094), relying on the linker's built-in
# search path. Debug builds auto-pick a fast linker (lld/mold/gold,
# configure.ac:623-673), and gold/lld do NOT search /usr/local/lib by
# default (only the bfd ld does) — without this, linking fails with
# "cannot find -lPocoFoundation" et al. (proven in CI run 28907270901).
#
# BRAND-01/02/03/04/06 branding flags (research 2-RESEARCH.md Pitfall 1):
# --with-app-branding wires eden-branding/ into the APP_HAS_BRANDING copy
# recipe in browser/Makefile.am (branding.css/js, images/*.svg overwrites,
# toolbar-bg.svg rename, welcome/ dir). --with-app-name/--with-vendor feed
# the product-name/vendor strings; --with-help-url sets the help link
# target.
#
# HARD OMIT LIST — never pass any of the following to configure, ever:
#   with-welcome-url, with-feedback-url, with-infobar-url,
#   with-support-public-key
# Passing with-welcome-url flips configure.ac's ENABLE_WELCOME_MESSAGE
# branch (configure.ac:1316-1329), which SKIPS generating
# CONFIG_INTERFACE_FRAGMENT entirely — silently deleting brandProductName,
# brandProductURL, and logoURL from the generated coolwsd.xml with no
# error (BRAND-01 dies silently). The canary assertion below is the
# permanent guard against this regressing. Do NOT loosen it.
#
# with-info-url is intentionally NOT passed here: INFO_URL only feeds the
# mobile-app m4 path, not the same key as brandProductURL — passing it
# would do nothing for this web build.
CONFIGURE_LOG="$(mktemp)"
./configure --enable-silent-rules --enable-debug \
  --with-lokit-path="$PWD/engine/include" \
  --with-lo-path="$PWD/engine/instdir" \
  --with-app-branding="$PWD/eden-branding" \
  --with-app-name="EdenDocs" \
  --with-vendor="AO Cyber Systems" \
  --with-help-url="https://aocyber.ai" \
  LDFLAGS="-L/usr/local/lib" 2>&1 | tee "$CONFIGURE_LOG"

# BUILD-02 permanent assertion: this notice is SUCCESS for ENGINE_ASSETS
# builds, not a warning to fix. The engine-main-assets.tar.gz ships no
# workdir/ POCO (empirically verified 2026-07-07), so configure's
# --with-lo-builddir auto-discovery is EXPECTED to fail and fall back to
# the system POCO installed by scripts/eden/build-deps.sh. If this grep
# ever fails, do NOT silently loosen it — inspect the configure log,
# re-verify the tarball contents, and update BUILD-02 documentation.
grep -q "POCO not found in the engine workdir, falling back to system POCO" "$CONFIGURE_LOG"

# BRAND-01/BRAND-06 canary (research 2-RESEARCH.md Pitfall 1): brandProductName
# only survives configure when no welcome URL is set. If this ever fails,
# someone enabled a welcome URL — fix that, do NOT loosen this assertion.
grep -q "<brandProductName" coolwsd.xml \
  || { echo "ERROR: brandProductName key missing from generated coolwsd.xml — welcome-url dependency broken (see 2-RESEARCH.md Pitfall 1)"; exit 1; }

# Inject the EdenDocs values into the GENERATED (gitignored) coolwsd.xml.
# coolwsd.xml.in / configure.ac are never touched — only configure's OUTPUT
# is post-processed here.
python3 - <<'PYEOF'
import re, pathlib
p = pathlib.Path('coolwsd.xml')
c = p.read_text()
c = re.sub(r'(<brandProductName[^>]*>)[^<]*(</brandProductName>)', r'\1EdenDocs\2', c)
c = re.sub(r'(<brandProductURL[^>]*>)[^<]*(</brandProductURL>)', r'\1https://aocyber.ai\2', c)
c = re.sub(r'(<logoURL[^>]*>)[^<]*(</logoURL>)', r'\1images/eden-logo.svg\2', c)
p.write_text(c)
PYEOF

grep -q ">EdenDocs</brandProductName>" coolwsd.xml || { echo "ERROR: brandProductName injection failed"; exit 1; }
grep -q "aocyber.ai</brandProductURL>" coolwsd.xml || { echo "ERROR: brandProductURL injection failed"; exit 1; }

make -j"$(nproc)" build-nocheck

# build-nocheck maps to automake's `all-am` (Makefile.am:854), which builds
# only the top-level directory and its all-local hooks — it does NOT recurse
# into SUBDIRS, so browser/ (and therefore browser/dist) is never built by
# it. codeql-analysis.yml never notices because CodeQL only needs the C++
# compile. Build the browser bundle explicitly; its node_modules rule runs
# `npm ci --offline` against the vendored node_shrinkpack/ (no registry
# access, browser/Makefile.am:1214).
make -j"$(nproc)" -C browser

# Makefile.am's $(SYSTEM_STAMP) rule (lines 662-670) runs CLEANUP_COMMAND:
# './coolwsd --cleanup ... || rm -f ./coolwsd'. If coolwsd exits nonzero
# there, the '|| rm -f' DELETES the binary while make still exits 0 —
# leaving a successful-looking build with no coolwsd. Relink it
# ('make coolwsd' does not depend on system_stamp, so the cleanup rule
# does not re-fire) and re-run the exact cleanup invocation in the
# foreground to surface its output and exit code for diagnosis. The smoke
# test remains the authoritative proof the binary actually runs.
if [ ! -x ./coolwsd ]; then
  echo "WARNING: ./coolwsd missing after make — CLEANUP_COMMAND deleted it. Relinking..."
  make coolwsd
  echo "--- cleanup diagnosis: re-running the CLEANUP_COMMAND coolwsd invocation ---"
  set +e
  ./coolwsd --disable-cool-user-checking --cleanup --o:logging.level=trace
  echo "--- cleanup diagnosis exit code: $? ---"
  set -e
fi

# BUILD-01 output assertions — loud on failure so a missing artifact is
# never a silent set -e death (cost us CI runs 28907904535 + 28908705828).
test -x ./coolwsd || { echo "ERROR: ./coolwsd missing after build"; exit 1; }
test -d ./browser/dist || { echo "ERROR: ./browser/dist missing after build"; exit 1; }

echo "build.sh complete: ./coolwsd and ./browser/dist present."
