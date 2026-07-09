#!/usr/bin/env bash
# This Source Code Form is subject to the terms of the Mozilla Public
# License, v. 2.0. If a copy of the MPL was not distributed with this
# file, You can obtain one at http://mozilla.org/MPL/2.0/.
#
# EdenDocs WOPI end-to-end proof (AUTH-01/02/04): one deterministic,
# headless command proves the full chain — AOID OIDC login (fake AOID) ->
# launch page mints a WOPI token -> coolwsd opens a real document session ->
# coolwsd calls back CheckFileInfo + GetFile with valid proof signatures ->
# a forced save round-trips through PutFile — and separately proves coolwsd
# trusts the WOPI host ONLY because storage.wopi.alias_groups is configured
# (positive AND negative trust, both behavioral, never config-file
# inspection alone).
#
# Two modes, selected by COOLWSD_MODE (default "auto"):
#   native  -- build/run coolwsd from ./coolwsd (repo root), native port 9980
#   docker  -- spin an ephemeral collabora/code container on host port 9981
#              (the long-running edendocs-code reference container already
#              owns 127.0.0.1:9980 -- never touched by this script)
#
# HARD PROJECT RULE: never use port 8080 for anything. Ports in play here:
# 8091 (wopi-host), 8092 (fake-aoid), 9980 (coolwsd native), 9981 (this
# script's own docker-mode coolwsd container).
#
# WOPI_HOST_PORT overrides the wopi-host listen port (default 8091, the
# TRD-specified/production value CI always uses). This exists SOLELY so a
# local dry run can dodge an unrelated pre-existing local listener on 8091
# on a shared dev machine without changing any committed behavior.
set -euo pipefail
cd "$(dirname "$0")/../.."

WOPI_HOST_PORT="${WOPI_HOST_PORT:-8091}"

MODE="${COOLWSD_MODE:-auto}"
if [ "$MODE" = "auto" ]; then
  if [ -x ./coolwsd ]; then
    MODE="native"
  else
    MODE="docker"
  fi
fi

TMPD=$(mktemp -d)
DATA_DIR=$(mktemp -d)
WOPI_LOG="$TMPD/wopi-host.log"
PIDS=()

cleanup() {
  for pid in "${PIDS[@]:-}"; do
    kill "$pid" >/dev/null 2>&1 || true
  done
  if [ "$MODE" = "docker" ]; then
    docker rm -f edendocs-wopi-e2e >/dev/null 2>&1 || true
  fi
  # Ephemeral native proof_key -- generated below, NEVER committed.
  rm -f ./proof_key ./proof_key.pub
  rm -rf "$TMPD" "$DATA_DIR"
}
trap cleanup EXIT

# On failure, surface every log this script has -- wopi-host's structured
# log (the 03-03 contract) and coolwsd's own log (native file / docker logs)
# -- mirroring scripts/eden/smoke-test.sh's fail() convention.
fail() {
  echo "WOPI E2E FAILED: $1"
  echo "=== tail of wopi-host log ($WOPI_LOG) ==="
  tail -n 200 "$WOPI_LOG" 2>/dev/null || echo "(no wopi-host log)"
  if [ "$MODE" = "native" ]; then
    echo "=== tail of /tmp/coolwsd.log ==="
    tail -n 200 /tmp/coolwsd.log 2>/dev/null || echo "(no /tmp/coolwsd.log)"
  else
    echo "=== docker logs edendocs-wopi-e2e (tail) ==="
    docker logs --tail 200 edendocs-wopi-e2e 2>/dev/null || echo "(no docker logs)"
  fi
  exit 1
}

# wait_for_http polls url until it returns 2xx, writing the body to outfile.
wait_for_http() {
  local url="$1" outfile="$2" tries="${3:-60}"
  for _ in $(seq 1 "$tries"); do
    if curl -fsS "$url" -o "$outfile" 2>/dev/null; then
      return 0
    fi
    sleep 1
  done
  return 1
}

# wait_for_log polls logfile for pattern (plain grep, BRE) up to tries
# seconds -- used for the async wopi-host structured-log assertions (proof
# verification and PutFile in particular can lag the WS frame that triggers
# them by a beat).
wait_for_log() {
  local pattern="$1" logfile="$2" tries="${3:-15}"
  for _ in $(seq 1 "$tries"); do
    grep -q "$pattern" "$logfile" 2>/dev/null && return 0
    sleep 1
  done
  return 1
}

echo "=== WOPI E2E: mode=$MODE ==="

# ---------------------------------------------------------------------------
# Build all three wopi-host binaries.
# ---------------------------------------------------------------------------
mkdir -p "$TMPD/bin"
(cd wopi-host && go build -o "$TMPD/bin/" ./cmd/...) \
  || fail "go build ./cmd/... failed"

# ---------------------------------------------------------------------------
# Seed the document data dir.
# ---------------------------------------------------------------------------
cp wopi-host/testdata/hello.odt "$DATA_DIR/hello.odt"

# ---------------------------------------------------------------------------
# Bring up coolwsd (mode-specific) and resolve COOLWSD_HOST_URL -- the
# origin BOTH wopi-host's own discovery client and e2e-probe's WebSocket
# dial use (coolwsd's host-published port either way; wopi-host always runs
# on the host process side, never inside the container).
# ---------------------------------------------------------------------------
if [ "$MODE" = "native" ]; then
  COOLWSD_HOST_URL="http://127.0.0.1:9980"
  WOPI_BASE_URL="http://127.0.0.1:$WOPI_HOST_PORT"
  WOPI_LISTEN_ADDR="127.0.0.1:$WOPI_HOST_PORT"

  # setcap + systemplate bootstrap (mirrors scripts/eden/smoke-test.sh).
  if command -v setcap >/dev/null 2>&1 && command -v sudo >/dev/null 2>&1; then
    sudo setcap cap_fowner,cap_chown,cap_sys_chroot=ep ./coolforkit-caps
    sudo setcap cap_sys_admin=ep ./coolmount
  fi
  ./coolwsd-systemplate-setup ./systemplate "$PWD/engine/instdir"
  mkdir -p ./jails ./cache

  # Ephemeral proof key -- debug builds resolve DEBUG_ABSSRCDIR/proof_key to
  # the repo root (3-RESEARCH.md). NEVER committed (cleanup trap removes it).
  if [ ! -f ./proof_key ]; then
    ssh-keygen -t rsa -b 2048 -N "" -m PEM -f ./proof_key >/dev/null
  fi

  ./coolwsd --disable-ssl \
    --o:sys_template_path="$PWD/systemplate" \
    --o:child_root_path="$PWD/jails" \
    --o:cache_files.path="$PWD/cache" \
    --o:storage.wopi.alias_groups[@mode]=groups \
    --o:storage.wopi.alias_groups.group[0].host=http://127.0.0.1:$WOPI_HOST_PORT \
    --o:storage.wopi.alias_groups.group[0].host[@allow]=true \
    --o:admin_console.username=admin --o:admin_console.password=admin &
  COOLWSD_PID=$!
  PIDS+=("$COOLWSD_PID")

  for _ in $(seq 1 120); do
    ./coolwsd --probe >/dev/null 2>&1 && break
    sleep 1
  done
  ./coolwsd --probe >/dev/null 2>&1 || fail "coolwsd (native) did not become ready on 9980 within 120s"
else
  COOLWSD_HOST_URL="http://127.0.0.1:9981"
  WOPI_BASE_URL="http://host.docker.internal:$WOPI_HOST_PORT"
  WOPI_LISTEN_ADDR="0.0.0.0:$WOPI_HOST_PORT"

  # The long-running edendocs-code reference container already owns
  # 127.0.0.1:9980 -- this is a SEPARATE, ephemeral container on 9981.
  docker run --rm -d --name edendocs-wopi-e2e -p 127.0.0.1:9981:9980 \
    -e aliasgroup1="http://host\\.docker\\.internal:$WOPI_HOST_PORT" \
    -e extra_params='--o:ssl.enable=false' \
    collabora/code:latest >/dev/null \
    || fail "could not start edendocs-wopi-e2e docker container"

  # This fresh container has no proof_key file yet (Pitfall 3: coolwsd only
  # signs/publishes proof keys when {COOLWSD_CONFIGDIR}/proof_key exists) --
  # generate one now, matching the reference container's provenance in
  # 03-03-SUMMARY.md. Retried: the entrypoint may still be finishing its own
  # bootstrap when this first runs.
  PROOF_KEY_OK=0
  for _ in $(seq 1 30); do
    if docker exec -u root edendocs-wopi-e2e coolconfig generate-proof-key >/dev/null 2>&1; then
      PROOF_KEY_OK=1
      break
    fi
    sleep 1
  done
  [ "$PROOF_KEY_OK" -eq 1 ] \
    || fail "could not generate coolwsd proof_key inside edendocs-wopi-e2e container"

  wait_for_http "$COOLWSD_HOST_URL/hosting/capabilities" "$TMPD/capabilities.json" 120 \
    || fail "coolwsd (docker) did not become ready on 9981 within 120s"
  grep -q 'convert-to' "$TMPD/capabilities.json" \
    || fail "coolwsd (docker) capabilities missing convert-to"
fi

# ---------------------------------------------------------------------------
# Bring up fake-aoid (8092) FIRST and WAIT for it before launching wopi-host:
# wopi-host performs a single, non-retried OIDC issuer discovery at startup
# and log.Fatal()s if the IdP is unreachable, so ordering here must be
# deterministic (start IdP -> wait ready -> start wopi-host -> wait ready).
# ---------------------------------------------------------------------------
EDENDOCS_FAKE_AOID_ADDR=127.0.0.1:8092 "$TMPD/bin/fake-aoid" >"$TMPD/fake-aoid.log" 2>&1 &
FAKE_AOID_PID=$!
PIDS+=("$FAKE_AOID_PID")

wait_for_http "http://127.0.0.1:8092/.well-known/openid-configuration" "$TMPD/aoid-discovery.json" 15 \
  || fail "fake-aoid did not become ready on 8092 within 15s (startup-ordering gate)"

export EDENDOCS_WOPI_AOID_ISSUER="http://127.0.0.1:8092"
export EDENDOCS_WOPI_OIDC_CLIENT_ID="edendocs-e2e"
export EDENDOCS_WOPI_OIDC_CLIENT_SECRET="e2e-secret"
export EDENDOCS_WOPI_DATA_DIR="$DATA_DIR"
export EDENDOCS_WOPI_LOG_FILE="$WOPI_LOG"
export EDENDOCS_WOPI_REQUIRE_PROOF="true"
export EDENDOCS_WOPI_PUBLIC_URL="http://127.0.0.1:$WOPI_HOST_PORT"
export EDENDOCS_WOPI_COOLWSD_URL="$COOLWSD_HOST_URL"
export EDENDOCS_WOPI_BASE_URL="$WOPI_BASE_URL"
export EDENDOCS_WOPI_LISTEN_ADDR="$WOPI_LISTEN_ADDR"

"$TMPD/bin/wopi-host" >"$TMPD/wopi-host.stdout.log" 2>&1 &
WOPI_HOST_PID=$!
PIDS+=("$WOPI_HOST_PID")

wait_for_http "http://127.0.0.1:$WOPI_HOST_PORT/healthz" "$TMPD/healthz.json" 30 \
  || fail "wopi-host did not become ready on $WOPI_HOST_PORT within 30s"

# ===========================================================================
# Test list 1-10.
# ===========================================================================

echo "--- [1/10] fake-aoid healthy on 8092 ---"
kill -0 "$FAKE_AOID_PID" 2>/dev/null || fail "fake-aoid process died"
grep -q 'authorization_endpoint' "$TMPD/aoid-discovery.json" \
  || fail "fake-aoid discovery document missing authorization_endpoint"

echo "--- [2/10] wopi-host healthy on $WOPI_HOST_PORT ---"
kill -0 "$WOPI_HOST_PID" 2>/dev/null || fail "wopi-host process died"
curl -fsS "http://127.0.0.1:$WOPI_HOST_PORT/healthz" -o "$TMPD/healthz.json" \
  || fail "wopi-host not healthy on $WOPI_HOST_PORT"
grep -q '"status":"ok"' "$TMPD/healthz.json" \
  || fail "wopi-host /healthz did not report ok"

echo "--- [3/10] coolwsd healthy ($MODE) ---"
if [ "$MODE" = "native" ]; then
  ./coolwsd --probe >/dev/null 2>&1 || fail "coolwsd (native) failed --probe readiness check"
else
  curl -fsS "$COOLWSD_HOST_URL/hosting/capabilities" -o "$TMPD/capabilities.json" \
    || fail "coolwsd (docker) not healthy on 9981"
  grep -q 'convert-to' "$TMPD/capabilities.json" \
    || fail "coolwsd (docker) capabilities missing convert-to"
fi

echo "--- [4/10] OIDC round trip -> /open lands with a session ---"
curl -fsS -c "$TMPD/jar" -b "$TMPD/jar" -L "http://127.0.0.1:$WOPI_HOST_PORT/open?file=hello.odt" \
  -o "$TMPD/open.html" || fail "OIDC round trip to /open failed"
grep -q 'access_token' "$TMPD/open.html" \
  || fail "/open page did not render an access_token form field"

echo "--- [5/10] minted token is an opaque uuid, not a JWT (Pitfall 2 guard) ---"
ACCESS_TOKEN=$(grep -o 'name="access_token" value="[^"]*"' "$TMPD/open.html" | head -1 | sed -E 's/.*value="([^"]*)".*/\1/')
ACCESS_TOKEN_TTL=$(grep -o 'name="access_token_ttl" value="[^"]*"' "$TMPD/open.html" | head -1 | sed -E 's/.*value="([^"]*)".*/\1/')
[ -n "$ACCESS_TOKEN" ] || fail "could not extract access_token from /open page"
[ -n "$ACCESS_TOKEN_TTL" ] || fail "could not extract access_token_ttl from /open page"
if ! [[ "$ACCESS_TOKEN" =~ ^[0-9a-f-]{36}$ ]]; then
  fail "access_token is not an opaque uuid (looks like a JWT?): $ACCESS_TOKEN"
fi

FORM_ACTION_RAW=$(grep -o 'action="[^"]*"' "$TMPD/open.html" | head -1 | sed -E 's/^action="(.*)"$/\1/')
FORM_ACTION=$(printf '%s' "$FORM_ACTION_RAW" | sed 's/&amp;/\&/g')
[ -n "$FORM_ACTION" ] || fail "could not extract launch form action from /open page"
WOPISRC_ENC=$(printf '%s' "$FORM_ACTION" | grep -o 'WOPISrc=[^&]*' | sed 's/^WOPISrc=//')
[ -n "$WOPISRC_ENC" ] || fail "could not extract WOPISrc from launch form action"
WOPISRC=$(python3 -c "import sys, urllib.parse; print(urllib.parse.unquote(sys.argv[1]))" "$WOPISRC_ENC")
[ -n "$WOPISRC" ] || fail "WOPISrc URL-decoded to empty"

echo "--- [6-8/10] POSITIVE trust: coolclient load -> CheckFileInfo/GetFile/proof -> forced PutFile ---"
"$TMPD/bin/e2e-probe" \
  -coolwsd-url "$COOLWSD_HOST_URL" \
  -wopisrc "$WOPISRC" \
  -token "$ACCESS_TOKEN" \
  -ttl "$ACCESS_TOKEN_TTL" \
  -timeout 45s \
  -save \
  || fail "e2e-probe positive-trust run did not observe an accepted status: frame"

wait_for_log 'wopi: CheckFileInfo.*status=200' "$WOPI_LOG" 15 \
  || fail "wopi-host log missing 'wopi: CheckFileInfo ... status=200'"
wait_for_log 'wopi: GetFile.*status=200' "$WOPI_LOG" 15 \
  || fail "wopi-host log missing 'wopi: GetFile ... status=200'"
wait_for_log 'proof: verified ok' "$WOPI_LOG" 15 \
  || fail "wopi-host log missing 'proof: verified ok' -- proof-key validation was not exercised"

if ! wait_for_log 'wopi: PutFile.*status=200' "$WOPI_LOG" 15; then
  echo "PutFile not yet observed -- retrying the forced save once (error_recovery ladder)"
  sleep 5
  "$TMPD/bin/e2e-probe" \
    -coolwsd-url "$COOLWSD_HOST_URL" \
    -wopisrc "$WOPISRC" \
    -token "$ACCESS_TOKEN" \
    -ttl "$ACCESS_TOKEN_TTL" \
    -timeout 30s \
    -save \
    || fail "e2e-probe retry positive-trust run failed"
  wait_for_log 'wopi: PutFile.*status=200' "$WOPI_LOG" 20 \
    || fail "wopi-host log still missing 'wopi: PutFile ... status=200' after retry"
fi

echo "--- [9/10] NEGATIVE trust: untrusted WOPISrc host rejected ---"
"$TMPD/bin/e2e-probe" \
  -coolwsd-url "$COOLWSD_HOST_URL" \
  -wopisrc "http://untrusted.invalid/wopi/files/x" \
  -token "$ACCESS_TOKEN" \
  -ttl "$ACCESS_TOKEN_TTL" \
  -timeout 30s \
  -expect-unauthorized \
  || fail "e2e-probe negative-trust run did not observe the expected unauthorized rejection"
if grep -q 'untrusted.invalid' "$WOPI_LOG" 2>/dev/null; then
  fail "wopi-host log shows a callback referencing the untrusted host -- alias_groups trust boundary violated"
fi

echo "--- [10/10] zero-upstream-diff gate (wsd/, browser/) ---"
if [ -n "$(git status --porcelain wsd/ browser/)" ]; then
  fail "wsd/ or browser/ has uncommitted changes -- AUTH-04 requires zero coolwsd-tree edits"
fi

echo "WOPI E2E PASSED: OIDC -> launch -> CheckFileInfo/GetFile/PutFile -> proof + alias_groups trust (mode=$MODE)"
