#!/usr/bin/env bash
#
# fetch-engine-assets.sh — EdenDocs from-source build pipeline (BUILD-01 fast path).
#
# Downloads the upstream prebuilt "for-code-assets" engine tarball
# (LibreOffice-derived instdir/) and extracts it into engine/, so a
# from-source EdenDocs build does not need to compile LibreOffice itself.
#
# Empirically verified (2026-07-07, objective 01-from-source-build-pipeline,
# BUILD-02): the live engine-main-assets.tar.gz contains ONLY instdir/ —
# zero POCO headers, zero workdir/. This means POCO auto-discovery via
# --with-lo-builddir deterministically fails, and the build MUST fall back
# to a system-installed POCO (see scripts/eden/build-deps.sh and the
# assertion in scripts/eden/build.sh). This script does not attempt to
# fetch or verify POCO — that is build-deps.sh's job.
#
# HARD PROJECT RULE: never use, bind, curl, or reference port 8080 for
# anything. This script does not touch any port; noted here for grep
# auditability.
#
set -euo pipefail

cd "$(dirname "$0")/../.."

# Overridable so a distro/collabora/co-XX.04 branch (or a renamed upstream
# asset) can point elsewhere without editing this script. Defaults to the
# main-branch asset since eden-main tracks upstream main.
ENGINE_ASSETS_URL="${ENGINE_ASSETS_URL:-https://github.com/CollaboraOnline/online/releases/download/for-code-assets/engine-main-assets.tar.gz}"

# Idempotence guard: skip re-fetching if engine/instdir already exists,
# unless the caller explicitly forces a refetch.
if [ -d engine/instdir ] && [ "${FORCE_REFETCH:-0}" != "1" ]; then
  echo "engine/instdir already exists — skipping fetch (set FORCE_REFETCH=1 to force)."
  exit 0
fi

echo "Fetching engine assets from: $ENGINE_ASSETS_URL"
wget --no-verbose "$ENGINE_ASSETS_URL" -O engine-assets.tar.gz

# Log the sha256 for traceability only — we deliberately do NOT pin/verify
# against a hardcoded checksum. "for-code-assets" is a rolling, unversioned
# release tag with no published sidecar hash; upstream rebuilds this asset
# on every main-branch push. Pinning a checksum here would fossilize the
# build against a stale artifact and break every future fetch.
sha256sum engine-assets.tar.gz

mkdir -p engine
tar xzf engine-assets.tar.gz -C engine
rm engine-assets.tar.gz

# Post-extract sanity check: fail loudly (not silently) if upstream ever
# changes the tarball layout. --with-lo-path's --enable-debug check in
# scripts/eden/build.sh depends on this exact file existing.
test -f engine/instdir/program/versionrc

echo "engine/instdir populated successfully (versionrc present)."
