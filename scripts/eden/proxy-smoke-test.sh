#!/usr/bin/env bash
# proxy-smoke-test.sh -- DIST-02: prove a document session survives >60s through a
# REAL reverse proxy (not direct 9980). Reuses etc/nginx/coolwsd.conf VERBATIM.
# Host web port 8091 only -- the project-banned web port is never used. Usage:
#   proxy-smoke-test.sh [IMAGE]   (IMAGE defaults to the reference coolwsd for local runs)
set -euo pipefail
cd "$(dirname "$0")/../.."

IMAGE="${1:-collabora/code:latest}"          # 04-04 passes the EdenDocs :scan image
HOLD_SECS="${HOLD_SECS:-80}"                  # >60s to clear nginx's 60s default
NET_NS_NAME=edendocs-proxy-coolwsd

# Assert we are about to reuse coolwsd.conf UNMODIFIED.
git diff --quiet HEAD -- etc/nginx/coolwsd.conf \
  || { echo "FATAL: etc/nginx/coolwsd.conf is modified -- DIST-02 requires verbatim reuse"; exit 1; }

cleanup() { docker rm -f "$NET_NS_NAME" edendocs-proxy edendocs-proxy-neg 2>/dev/null || true; }
trap cleanup EXIT
cleanup

# 1) coolwsd container: publish 8091 on IT (nginx will bind 8091 in its namespace).
docker run -d --name "$NET_NS_NAME" -p 8091:8091 \
  --cap-add MKNOD --cap-add SYS_CHROOT --cap-add FOWNER --cap-add CHOWN --cap-add SYS_ADMIN \
  -e "extra_params=--o:ssl.enable=false --o:net.idle_timeout_secs=3600" \
  "$IMAGE"
# wait for coolwsd to accept on 9980 inside the namespace
for _ in $(seq 1 120); do
  docker exec "$NET_NS_NAME" /usr/bin/coolwsd --probe >/dev/null 2>&1 && break
  sleep 1
done

run_case() {  # $1=config file  $2=expect (alive|dead)  $3=proxy container name
  local conf="$1" expect="$2" name="$3"
  docker run -d --name "$name" --network "container:$NET_NS_NAME" \
    -v "$PWD/etc/nginx/coolwsd.conf:/etc/nginx/snippets/coolwsd.conf:ro" \
    -v "$PWD/$conf:/etc/nginx/conf.d/server.conf:ro" \
    -v /dev/null:/etc/nginx/conf.d/default.conf:ro \
    nginx:alpine
  sleep 3
  # Positive-config structural leg: prove the 36000s override is actually loaded.
  if [ "$expect" = alive ]; then
    docker exec "$name" nginx -T 2>/dev/null | grep -q 'proxy_read_timeout 36000s' \
      || { echo "FATAL: 36000s proxy_read_timeout not in loaded config (override ignored)"; exit 1; }
  fi
  # Live leg: open a WS through :8091 to /cool/<id>/ws, hold idle, check liveness.
  python3 - "$HOLD_SECS" "$expect" <<'PY'
import sys, time, socket
try:
    from websockets.sync.client import connect
except Exception:
    import subprocess
    subprocess.run([sys.executable, "-m", "pip", "install", "-q", "websockets"])
    from websockets.sync.client import connect
hold = int(sys.argv[1]); expect = sys.argv[2]
url = "ws://127.0.0.1:8091/cool/edendocsproxytest/ws"
alive = True
try:
    ws = connect(url, open_timeout=15)
    t0 = time.time()
    while time.time() - t0 < hold:
        time.sleep(5)
        try:
            ws.ping()             # client->server; read timeout is server->client silence
        except Exception:
            alive = False; break
    try:
        ws.close()
    except Exception:
        pass
except Exception as e:
    alive = False
print(f"case expect={expect} alive_after={hold}s -> {alive}")
if expect == "alive" and not alive: sys.exit(1)
if expect == "dead"  and alive:     sys.exit(1)
PY
  docker rm -f "$name" >/dev/null 2>&1 || true
}

echo "=== POSITIVE: verbatim coolwsd.conf (36000s) -- expect ALIVE at ${HOLD_SECS}s ==="
run_case scripts/eden/nginx-test-server.conf alive edendocs-proxy

echo "=== NEGATIVE CONTROL: short proxy_read_timeout -- expect DEAD before ${HOLD_SECS}s ==="
run_case scripts/eden/nginx-test-server-negative.conf dead edendocs-proxy-neg

echo "PROXY SMOKE PASSED: WS session survived >60s through the verbatim reverse proxy; negative control died (DIST-02)."
