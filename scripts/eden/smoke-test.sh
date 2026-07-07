#!/usr/bin/env bash
# EdenDocs smoke test (BUILD-04): the built coolwsd must serve
# /hosting/discovery and /hosting/capabilities on its NATIVE port 9980.
# HARD PROJECT RULE: never use port 8080 for anything.
set -euo pipefail
cd "$(dirname "$0")/../.."

# systemplate bootstrap — canonical in-tree tool (Makefile.am:668)
./coolwsd-systemplate-setup ./systemplate "$PWD/engine/instdir"
mkdir -p ./jails ./cache

./coolwsd --disable-ssl \
  --o:sys_template_path="$PWD/systemplate" \
  --o:child_root_path="$PWD/jails" \
  --o:cache_files.path="$PWD/cache" \
  --o:storage.filesystem[@allow]=true \
  --o:admin_console.username=admin --o:admin_console.password=admin &
COOLWSD_PID=$!
trap 'kill "$COOLWSD_PID" 2>/dev/null || true' EXIT

# Readiness: coolwsd's built-in self-contained probe (wsd/COOLWSD.cpp,
# probeRunningServer) — primary signal, retried up to 120s for cold CI starts.
for _ in $(seq 1 120); do
  if ./coolwsd --probe >/dev/null 2>&1; then break; fi
  sleep 1
done

# Literal BUILD-04 checks on the native port 9980:
curl -fsS http://127.0.0.1:9980/hosting/discovery    | grep -q 'wopi-discovery'
curl -fsS http://127.0.0.1:9980/hosting/capabilities | grep -q 'convert-to'
echo "SMOKE TEST PASSED: coolwsd serves /hosting/discovery and /hosting/capabilities on 9980"
