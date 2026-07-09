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
