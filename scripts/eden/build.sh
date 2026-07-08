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
CONFIGURE_LOG="$(mktemp)"
./configure --enable-silent-rules --enable-debug \
  --with-lokit-path="$PWD/engine/include" \
  --with-lo-path="$PWD/engine/instdir" \
  LDFLAGS="-L/usr/local/lib" 2>&1 | tee "$CONFIGURE_LOG"

# BUILD-02 permanent assertion: this notice is SUCCESS for ENGINE_ASSETS
# builds, not a warning to fix. The engine-main-assets.tar.gz ships no
# workdir/ POCO (empirically verified 2026-07-07), so configure's
# --with-lo-builddir auto-discovery is EXPECTED to fail and fall back to
# the system POCO installed by scripts/eden/build-deps.sh. If this grep
# ever fails, do NOT silently loosen it — inspect the configure log,
# re-verify the tarball contents, and update BUILD-02 documentation.
grep -q "POCO not found in the engine workdir, falling back to system POCO" "$CONFIGURE_LOG"

make -j"$(nproc)" build-nocheck

# BUILD-01 output assertions.
test -x ./coolwsd
test -d ./browser/dist

echo "build.sh complete: ./coolwsd and ./browser/dist present."
