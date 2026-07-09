---
objective: 04-edendocs-branded-container-images
trd: "03"
type: standard
wave: 1
depends_on: []
files_modified:
  - scripts/eden/proxy-smoke-test.sh
  - scripts/eden/nginx-test-server.conf
  - scripts/eden/nginx-test-server-negative.conf
autonomous: true
requirements: [DIST-02]
# TDD-EXCEPTION: infra/verification harness — a bash + nginx smoke test. The harness
# IS the test (a >60s WebSocket-survival assertion with a negative control); nginx -t
# + shellcheck are the local gate; the against-EdenDocs-image run is in 04-04's CI.
# The resolver's (app,feature) widget/Maestro profile does not apply.

must_haves:
  truths:
    - "scripts/eden/proxy-smoke-test.sh stands up the coolwsd container + a throwaway nginx:alpine reverse proxy that reuses the in-tree etc/nginx/coolwsd.conf VERBATIM (byte-unmodified), opens a WebSocket document-session through the proxy on host port 8091 (never 8080, never direct 9980), holds it idle, and asserts the session is still alive at t≈80s (>60s)"
    - "The harness proves the 36000s override is actually in effect (not silently ignored): it dumps the proxy's effective config (nginx -T) and asserts proxy_read_timeout 36000s is loaded on the /cool/ location"
    - "A negative control proves the harness can DETECT the failure it exists to catch: the same session through a proxy config with a deliberately short proxy_read_timeout dies before t=80s, and the harness asserts that death"
    - "coolwsd.conf is reused unmodified by co-locating nginx in the coolwsd container's network namespace (docker run --network container:...) so the config's proxy_pass http://localhost:9980 resolves without any edit"
  artifacts:
    - "scripts/eden/proxy-smoke-test.sh"
    - "scripts/eden/nginx-test-server.conf"
    - "scripts/eden/nginx-test-server-negative.conf"
  key_links:
    - "proxy-smoke-test.sh → etc/nginx/coolwsd.conf (reused VERBATIM — the 36000s proxy_read_timeout + Upgrade/Connection headers + ^~ /cool/ prefix match; research Don't-Hand-Roll)"
    - "nginx:alpine --network container:edendocs → coolwsd 9980 in the SAME namespace (keeps coolwsd.conf's localhost:9980 valid with zero edits); host publishes 8091 on the coolwsd container"
    - "negative control (short proxy_read_timeout) → asserted session death <80s (proves discrimination — Pitfall 5)"
---

<objective>
Build the reverse-proxy smoke harness that operationalizes DIST-02's real-world half: a document session held open through a *real* reverse proxy must survive longer than 60 seconds, not merely on direct `127.0.0.1:9980`. This catches the exact failure mode Pitfall 5/6 warn about — nginx's 60s default `proxy_read_timeout` silently killing idle WebSocket editing sessions when a proxy config omits the override.

The harness reuses the in-tree `etc/nginx/coolwsd.conf` byte-for-byte (it already has `proxy_read_timeout 36000s`, the `Upgrade`/`Connection` headers, and the `location ^~ /cool/` prefix match), stands up a throwaway `nginx:alpine` in front of coolwsd, opens a WebSocket session through it on host port 8091, holds it idle, and asserts liveness past 60s. A negative control (short-timeout config) proves the harness genuinely discriminates the failure case.

Purpose: This is authored independently (no dependency on 04-01/04-02 files) and can be validated locally against ANY coolwsd — the proxy-timeout behavior is image-independent — then wired against the EdenDocs image in 04-04's CI as the DIST-02 proxy proof.

Output: `scripts/eden/proxy-smoke-test.sh`, `scripts/eden/nginx-test-server.conf`, `scripts/eden/nginx-test-server-negative.conf` — committed on eden-main.
</objective>

<file_tree>
scripts/eden/
├── proxy-smoke-test.sh              ← CREATE (WS >60s survival harness + negative control)
├── nginx-test-server.conf           ← CREATE (server{} on 8091, includes coolwsd.conf verbatim)
├── nginx-test-server-negative.conf  ← CREATE (negative control: short proxy_read_timeout)
└── smoke-test.sh                      (untouched — direct-9980 HTTP smoke, BUILD-04)
etc/nginx/coolwsd.conf                 (UNTOUCHED upstream — mounted VERBATIM by the harness)
</file_tree>

<execution_context>
@~/.claude/devflow/workflows/execute-trd.md
@~/.claude/devflow/templates/summary.md
</execution_context>

<embedded_context>

<codebase_examples>
The in-tree `etc/nginx/coolwsd.conf` (reuse VERBATIM — mount it, never edit it):

```nginx
location ^~ /browser { proxy_pass http://localhost:9980; proxy_set_header Host $host; }
location ^~ /hosting/discovery { proxy_pass http://localhost:9980; proxy_set_header Host $host; }
location ^~ /hosting/capabilities { proxy_pass http://localhost:9980; proxy_set_header Host $host; }
location ^~ /cool/ {
    proxy_pass http://localhost:9980;
    proxy_http_version 1.1;
    proxy_set_header Upgrade $http_upgrade;
    proxy_set_header Connection "Upgrade";
    proxy_set_header Host $host;
    proxy_read_timeout 36000s;     # <-- the override DIST-02 proves is honored
}
location ^~ /lool/ { proxy_pass http://localhost:9980; proxy_set_header Host $host; }
```

`scripts/eden/smoke-test.sh` shows the repo's fail()/readiness/port conventions (native 9980, 8091 for host web checks, NEVER 8080) — mirror its style. Note it is a DIRECT-9980 test; this harness is deliberately the reverse-proxy counterpart.
</codebase_examples>

<anti_patterns>
- **Do NOT hand-write a new nginx proxy config for /cool/** (Pitfall 6 / research Don't-Hand-Roll) — reuse `etc/nginx/coolwsd.conf` verbatim. It already solved the WS-header + `^~` prefix-match + 36000s-timeout gotchas. The only NEW nginx file is the thin `server { listen 8091; include coolwsd.conf; }` wrapper.
- **Do NOT edit coolwsd.conf's `proxy_pass http://localhost:9980`** to a container name — instead co-locate nginx in coolwsd's network namespace (`--network container:<coolwsd>`) so `localhost:9980` stays correct. Editing it would violate "reuse verbatim".
- **Do NOT test against direct 127.0.0.1:9980** — that is BUILD-04's existing smoke test and proves nothing about proxy behavior. This test MUST go through nginx on 8091.
- **NEVER bind, publish, or reference port 8080** — the nginx wrapper listens on 8091; the coolwsd container publishes 8091:8091 to the host; coolwsd stays on 9980. 8080 is banned even for throwaway test containers (Pitfall 9).
- **A sub-60s test proves nothing** (Pitfall 5) — the positive assertion must be at t>60s (use ~80s for margin). A harness that finishes in under 60s cannot satisfy DIST-02 regardless of pass/fail.
- **Do not skip the negative control** — without it, a passing happy-path run cannot distinguish "the 36000s override works" from "nothing was actually testing the timeout".
</anti_patterns>

<error_recovery>
- **The negative control does NOT die by t=80s** → coolwsd is emitting upstream WebSocket keepalives/pings that reset nginx's `proxy_read_timeout` timer. Fix by making the negative config's timeout SHORTER than coolwsd's keepalive cadence (e.g. `proxy_read_timeout 5s`), and/or additionally assert on the loaded config (`nginx -T` shows the negative timeout) so discrimination is proven structurally even if the ping cadence masks the live death. Record which discriminator fired.
- **WebSocket never upgrades (HTTP 400/404 through the proxy)** → confirm the wrapper `include`s coolwsd.conf inside a `server { listen 8091; ... }` block, that the `^~ /cool/` location is present in the loaded config (`nginx -T`), and that the WS URL path begins with `/cool/`. A 404 on /cool/ with coolwsd reachable directly is the Pitfall 6 misroute signal.
- **`--network container:` cannot publish 8091** → ports must be published on the FIRST (coolwsd) container: `docker run -d --name edendocs -p 8091:8091 <image>`, THEN `docker run --network container:edendocs nginx:alpine`. nginx binds 8091 inside the shared namespace; the host reaches it via the coolwsd container's published 8091.
- **No WebSocket client available** → prefer python3 + `websockets` (`pip install --quiet websockets`, present/installable on ubuntu-latest); `websocat` is an acceptable fallback. Keep the client choice self-contained in the script (install-on-demand), do not add a repo dependency.
- **coolwsd closes the session before 80s for its own reasons (idle_timeout)** → coolwsd's `net.idle_timeout_secs` default is generous (~3600s); if a stripped config shortens it, pass `--o:net.idle_timeout_secs=3600` when starting the coolwsd container, or keep the session marginally active with a single benign frame well inside the window. The target is proving PROXY survival, so keep the client truly idle for the read-timeout to be the thing under test.
</error_recovery>

</embedded_context>

<context>
@.planning/PROJECT.md
@.planning/objectives/04-edendocs-branded-container-images/4-RESEARCH.md
@etc/nginx/coolwsd.conf
@scripts/eden/smoke-test.sh
</context>

<research_context>
From 4-RESEARCH.md (HIGH confidence) — Pattern 3 (Reverse-proxy >60s WebSocket session test):
- nginx's default `proxy_read_timeout` is 60s; any config omitting the override silently kills idle WS at 60s. coolwsd's reference config overrides to 36000s specifically because idle editing (reading, not typing) routinely exceeds a minute.
- Design: (1) coolwsd container on 9980, no host port for 9980; (2) nginx:alpine reusing coolwsd.conf verbatim + a minimal server{listen 8091} wrapper; (3) open the session through :8091 (not :9980), hold the WS idle, assert alive at 75-90s; (4) negative control: a config omitting/shortening the override must die ~60s, proving the harness detects the failure mode.
- Reuse coolwsd.conf verbatim (Don't-Hand-Roll) — it has the correct 36000s timeout, Upgrade/Connection headers, and ^~ /cool/ prefix routing already.
</research_context>

<gotchas>
- HARD RULE: never 8080. nginx wrapper listens 8091; coolwsd container publishes 8091:8091; coolwsd native 9980. Every reference in these three files must honor this — the repo's :8080 discipline covers throwaway test containers too.
- The verbatim reuse is enforced by MOUNTING coolwsd.conf read-only into the nginx container at `/etc/nginx/snippets/coolwsd.conf` (`-v "$PWD/etc/nginx/coolwsd.conf:/etc/nginx/snippets/coolwsd.conf:ro"`) — NOT into conf.d (nginx would auto-load the bare snippet at http scope and abort). The file on disk is never edited, and the harness can assert byte-integrity with `git diff --quiet HEAD -- etc/nginx/coolwsd.conf`.
- The positive proof has two independent legs: (a) live — the WS is still open at t≈80s; (b) structural — `nginx -T` in the proxy shows `proxy_read_timeout 36000s` on /cool/. Assert BOTH so a keepalive-masked live pass is still backed by the structural check.
- This harness is Wave 1 (independent): validate locally against the reference container `edendocs-code`/upstream `collabora/code` on 9980 (PROJECT.md) — you do NOT need the 04-02 EdenDocs image to author or lint it. 04-04 re-runs it against the EdenDocs image.
</gotchas>

<tasks>

<task type="auto">
  <name>Task 1: Author the nginx wrapper configs (verbatim include + negative control)</name>
  <files>scripts/eden/nginx-test-server.conf, scripts/eden/nginx-test-server-negative.conf</files>
  <action>
Create `scripts/eden/nginx-test-server.conf` — a minimal server block that listens on 8091 and INCLUDEs the verbatim coolwsd.conf (mounted at build/run time into the same conf.d):

```nginx
# EdenDocs reverse-proxy smoke wrapper (DIST-02). Listens on 8091 (NEVER 8080).
# Includes etc/nginx/coolwsd.conf VERBATIM (mounted read-only) — do not duplicate
# any /cool/ proxy directives here; the included file owns them (36000s timeout,
# Upgrade/Connection headers, ^~ /cool/ prefix match).
server {
    listen 8091;
    server_name localhost;
    include /etc/nginx/snippets/coolwsd.conf;
}
```

IMPORTANT — mount path: `coolwsd.conf` is a bare `location`-only snippet (no `server{}`/`http{}` wrapper). nginx:alpine's stock `nginx.conf` already does `include /etc/nginx/conf.d/*.conf;` INSIDE `http{}`, so a snippet placed in `conf.d` would be auto-loaded at http scope and nginx would abort with `"location" directive is not allowed here`. It MUST be mounted OUTSIDE `conf.d` — use `/etc/nginx/snippets/coolwsd.conf` (which is exactly where `Makefile.am`'s `nginxconfig_DATA` installs it in-tree). Only this wrapper (a real `server{}` block) goes in `conf.d`; it pulls in the snippet by absolute path.

Create `scripts/eden/nginx-test-server-negative.conf` — the NEGATIVE control. It must route /cool/ with a deliberately SHORT proxy_read_timeout so an idle WS dies fast, proving the harness detects the failure. This one intentionally does NOT include the verbatim file (it is the "broken proxy" control):

```nginx
# NEGATIVE CONTROL (DIST-02 discrimination proof). Same routing shape as
# coolwsd.conf BUT with a short proxy_read_timeout, so an idle WebSocket is
# killed quickly — proving proxy-smoke-test.sh can detect the 60s-default
# failure mode. NEVER 8080.
server {
    listen 8091;
    server_name localhost;
    location ^~ /cool/ {
        proxy_pass http://localhost:9980;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "Upgrade";
        proxy_set_header Host $host;
        proxy_read_timeout 5s;   # deliberately short — the whole point of the control
    }
    location ^~ /browser { proxy_pass http://localhost:9980; proxy_set_header Host $host; }
}
```
  </action>
  <verify>
grep -q 'listen 8091;' scripts/eden/nginx-test-server.conf && grep -q 'include /etc/nginx/snippets/coolwsd.conf;' scripts/eden/nginx-test-server.conf && ! grep -q 'proxy_read_timeout' scripts/eden/nginx-test-server.conf && grep -q 'proxy_read_timeout 5s;' scripts/eden/nginx-test-server-negative.conf && grep -q 'listen 8091;' scripts/eden/nginx-test-server-negative.conf && ! grep -rn '8080' scripts/eden/nginx-test-server.conf scripts/eden/nginx-test-server-negative.conf && { command -v nginx >/dev/null && nginx -t -c /dev/stdin <<<'events{} http{ server{ listen 8091; } }' 2>/dev/null || echo "nginx not local — config lint runs inside nginx:alpine in the harness/CI"; }
  </verify>
  <done>The wrapper listens on 8091 and includes coolwsd.conf verbatim while declaring no /cool/ directives of its own; the negative control uses proxy_read_timeout 5s; neither file references 8080.</done>
  <recovery>If nginx rejects the `include` path, confirm the harness mounts coolwsd.conf to exactly /etc/nginx/snippets/coolwsd.conf (matching the include) — NOT into conf.d, where nginx:alpine's http-scope `include conf.d/*.conf` would auto-load the bare `location` snippet and abort with `"location" directive is not allowed here`. Only the server{} wrapper belongs in conf.d. If nginx:alpine's default conf.d/default.conf collides on 8091, the harness must remove/override it (mount /dev/null over /etc/nginx/conf.d/default.conf) — handled in Task 2.</recovery>
</task>

<task type="auto">
  <name>Task 2: Author proxy-smoke-test.sh (WS >60s survival + nginx -T structural assert + negative control)</name>
  <files>scripts/eden/proxy-smoke-test.sh</files>
  <action>
Create `scripts/eden/proxy-smoke-test.sh`. It takes the coolwsd IMAGE as `$1` (default to a reference image so local runs work), stands up coolwsd + nginx sharing a network namespace, runs the positive proof (verbatim config, alive at ~80s) and the negative control (short-timeout config, dead before ~80s). Structure:

```bash
#!/usr/bin/env bash
# proxy-smoke-test.sh — DIST-02: prove a document session survives >60s through a
# REAL reverse proxy (not direct 9980). Reuses etc/nginx/coolwsd.conf VERBATIM.
# Host web port 8091 only — 8080 is banned project-wide. Usage:
#   proxy-smoke-test.sh [IMAGE]   (IMAGE defaults to the reference coolwsd for local runs)
set -euo pipefail
cd "$(dirname "$0")/../.."

IMAGE="${1:-collabora/code:latest}"          # 04-04 passes the EdenDocs :scan image
HOLD_SECS="${HOLD_SECS:-80}"                  # >60s to clear nginx's 60s default
NET_NS_NAME=edendocs-proxy-coolwsd

# Assert we are about to reuse coolwsd.conf UNMODIFIED.
git diff --quiet HEAD -- etc/nginx/coolwsd.conf \
  || { echo "FATAL: etc/nginx/coolwsd.conf is modified — DIST-02 requires verbatim reuse"; exit 1; }

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
  docker exec "$NET_NS_NAME" /usr/bin/coolwsd --probe >/dev/null 2>&1 && break; sleep 1;
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
    import subprocess; subprocess.run([sys.executable,"-m","pip","install","-q","websockets"]); 
    from websockets.sync.client import connect
hold=int(sys.argv[1]); expect=sys.argv[2]
url="ws://127.0.0.1:8091/cool/edendocsproxytest/ws"
alive=True
try:
    ws=connect(url, open_timeout=15)
    t0=time.time()
    while time.time()-t0 < hold:
        time.sleep(5)
        try: ws.ping()            # client->server; read timeout is server->client silence
        except Exception: alive=False; break
    try: ws.close()
    except Exception: pass
except Exception as e:
    alive=False
print(f"case expect={expect} alive_after={hold}s -> {alive}")
if expect=="alive" and not alive: sys.exit(1)
if expect=="dead"  and alive:     sys.exit(1)
PY
  docker rm -f "$name" >/dev/null 2>&1 || true
}

echo "=== POSITIVE: verbatim coolwsd.conf (36000s) — expect ALIVE at ${HOLD_SECS}s ==="
run_case scripts/eden/nginx-test-server.conf alive edendocs-proxy

echo "=== NEGATIVE CONTROL: short proxy_read_timeout — expect DEAD before ${HOLD_SECS}s ==="
run_case scripts/eden/nginx-test-server-negative.conf dead edendocs-proxy-neg

echo "PROXY SMOKE PASSED: WS session survived >60s through the verbatim reverse proxy; negative control died (DIST-02)."
```

Notes: the WS `ping()` is client→server keepalive (does NOT reset the server→client `proxy_read_timeout`, which is what is under test). If coolwsd's own server→client pings mask the negative control, apply the error_recovery discriminator. Make the script executable.
  </action>
  <verify>
chmod +x scripts/eden/proxy-smoke-test.sh && bash -n scripts/eden/proxy-smoke-test.sh && command -v shellcheck >/dev/null && shellcheck -e SC2016,SC1091 scripts/eden/proxy-smoke-test.sh || true; grep -q 'git diff --quiet HEAD -- etc/nginx/coolwsd.conf' scripts/eden/proxy-smoke-test.sh && grep -q 'proxy_read_timeout 36000s' scripts/eden/proxy-smoke-test.sh && grep -q 'network "container:' scripts/eden/proxy-smoke-test.sh && grep -q 'HOLD_SECS:-80' scripts/eden/proxy-smoke-test.sh && grep -q 'expect=dead' <(grep -o 'negative.conf dead' scripts/eden/proxy-smoke-test.sh || echo 'negative.conf dead') && ! grep -rn '8080' scripts/eden/proxy-smoke-test.sh && grep -q '8091' scripts/eden/proxy-smoke-test.sh
  </verify>
  <done>proxy-smoke-test.sh parses clean, asserts coolwsd.conf is byte-unmodified before use, co-locates nginx via --network container:, opens a WS through :8091, holds ≥80s, structurally asserts the loaded 36000s override, runs a negative control expected to die, references 8091 (not 8080). Live end-to-end proof against the EdenDocs image is 04-04's CI job.</done>
  <recovery>Per error_recovery: negative control not dying → shorten its timeout / rely on the nginx -T structural leg. WS 404 → confirm /cool/ path + include wiring. Locally, run it against `collabora/code` on a machine with docker to sanity-check the flow before 04-04 wires the EdenDocs image; on macOS without a Linux docker engine, defer the live run to CI and keep the local gate at bash -n + shellcheck.</recovery>
</task>

</tasks>

<validation_gates>
<lint>bash -n scripts/eden/proxy-smoke-test.sh; command -v shellcheck >/dev/null && shellcheck -e SC2016,SC1091 scripts/eden/proxy-smoke-test.sh || true</lint>
<test># Live >60s WS run needs a Linux docker engine (04-04 CI, or local against collabora/code). Local gate:
git diff --quiet HEAD -- etc/nginx/coolwsd.conf && grep -q 'proxy_read_timeout 36000s' scripts/eden/proxy-smoke-test.sh</test>
<build>true # no build artifact; the harness is the deliverable</build>
</validation_gates>

<verification>
1. `etc/nginx/coolwsd.conf` is byte-unmodified (git diff clean) and reused verbatim via read-only mount (DIST-02 verbatim reuse).
2. The harness opens a WS through nginx on 8091 (not direct 9980), holds ≥80s, and asserts liveness AND that `proxy_read_timeout 36000s` is in the loaded proxy config.
3. The negative control (short timeout) is asserted to die before 80s — proving the harness discriminates the 60s-default failure mode (Pitfall 5).
4. No file references port 8080; coolwsd=9980, host proxy check=8091.
5. `nginx-test-server.conf` declares no /cool/ directives of its own (it includes the verbatim file).
</verification>

<success_criteria>
- A committed, self-contained harness that proves a coolwsd document session survives >60s through the in-tree nginx reverse-proxy config used verbatim, with a negative control proving the test detects the failure — the DIST-02 reverse-proxy half, wired against the EdenDocs image by 04-04.
- coolwsd.conf reused byte-for-byte; every port reference is 8091/9980, never 8080.
</success_criteria>

<output>
After completion, create `.planning/objectives/04-edendocs-branded-container-images/04-03-SUMMARY.md`: the shared-namespace approach (why it keeps coolwsd.conf verbatim), the exact positive/negative assertions and HOLD_SECS, which discriminator the negative control relied on (live death vs nginx -T structural), and whether a local run against a reference coolwsd was possible or deferred to CI.
</output>
