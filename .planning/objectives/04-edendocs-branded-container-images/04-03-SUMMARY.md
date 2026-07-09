---
objective: 04-edendocs-branded-container-images
job: "03"
subsystem: infra
tags: [nginx, reverse-proxy, websocket, proxy_read_timeout, coolwsd, docker, network-namespace, smoke-test, negative-control, dist-02]

# Dependency graph
requires: []   # Wave 1 independent — validates ANY coolwsd; no dependency on 04-01/04-02 files
provides:
  - "scripts/eden/proxy-smoke-test.sh — DIST-02 reverse-proxy half: stands up coolwsd + a throwaway nginx:alpine sharing coolwsd's network namespace, opens a WebSocket document session through the proxy on host port 8091, holds it idle HOLD_SECS=80s (>60s), asserts still-alive AND structurally asserts the loaded proxy_read_timeout 36000s; runs a short-timeout negative control asserted to die"
  - "scripts/eden/nginx-test-server.conf — server{} wrapper on 8091 that includes the in-tree etc/nginx/coolwsd.conf VERBATIM via /etc/nginx/snippets/coolwsd.conf; declares no /cool/ directives of its own"
  - "scripts/eden/nginx-test-server-negative.conf — negative control (proxy_read_timeout 5s) proving the harness discriminates the 60s-default failure mode"
affects: [04-04-docker-publish-cosign]   # 04-04 re-runs this harness against the EdenDocs image in CI as the DIST-02 proxy proof

# Tech tracking
tech-stack:
  added: []   # no repo dependency added — the WS client (python3 websockets) is installed on-demand inside the harness
  patterns:
    - "Shared-network-namespace reuse (docker run --network container:<coolwsd>): co-locate nginx in coolwsd's namespace so the in-tree coolwsd.conf's proxy_pass http://localhost:9980 resolves with ZERO edits — the enforcement mechanism for verbatim reuse. Host port is published on the FIRST (coolwsd) container (-p 8091:8091); nginx binds 8091 inside the shared namespace."
    - "Bare-location snippet mounted OUTSIDE conf.d: coolwsd.conf is a location-only snippet, so it mounts read-only at /etc/nginx/snippets/coolwsd.conf (NOT conf.d, where nginx:alpine's http-scope `include conf.d/*.conf` would auto-load a bare location and abort). Only the server{} wrapper goes into conf.d (as server.conf); /dev/null masks nginx:alpine's default.conf to free 8091."
    - "Two-legged positive proof: (a) LIVE — the WS is still open at t=80s (>60s); (b) STRUCTURAL — `nginx -T` inside the proxy shows proxy_read_timeout 36000s loaded on /cool/. Asserting BOTH keeps a keepalive-masked live pass backed by the structural check."
    - "Negative control as a discrimination proof: the same idle session through a short-timeout (5s) config must die before 80s — without it a green happy path cannot distinguish 'the 36000s override works' from 'nothing tested the timeout' (Pitfall 5)."

key-files:
  created:
    - "scripts/eden/proxy-smoke-test.sh"
    - "scripts/eden/nginx-test-server.conf"
    - "scripts/eden/nginx-test-server-negative.conf"
  modified: []

key-decisions:
  - "coolwsd.conf reused BYTE-FOR-BYTE by co-locating nginx in coolwsd's network namespace (--network container:edendocs-proxy-coolwsd) rather than editing its proxy_pass localhost:9980 to a container name — editing would violate verbatim reuse. The harness self-asserts `git diff --quiet HEAD -- etc/nginx/coolwsd.conf` before use and exits FATAL if the file is dirty."
  - "Mount coolwsd.conf read-only at /etc/nginx/snippets/coolwsd.conf (NOT conf.d) — the fixed-defect mount path. The server{} wrapper (which DOES belong in conf.d, mounted as server.conf) includes the snippet by absolute path; /dev/null is mounted over conf.d/default.conf to prevent nginx:alpine's stock server colliding on 8091."
  - "HOLD_SECS default 80s (>60s): a sub-60s test proves nothing about nginx's 60s default proxy_read_timeout (Pitfall 5). Positive expects ALIVE at 80s; negative expects DEAD before 80s."
  - "Negative control uses proxy_read_timeout 5s (far shorter than coolwsd's ~generous keepalive/idle cadence) so the idle WS is killed well inside the window; primary discriminator is live death, with the positive leg's `nginx -T` 36000s structural assert as the complementary discriminator. If coolwsd server->client pings mask the live death in CI, error_recovery adds an nginx -T assert of the negative 5s timeout — the actual discriminator that fires is recorded when the live run executes in 04-04 CI."
  - "WS client is python3 `websockets` installed on-demand inside the harness (no repo dependency added); the client keepalive is a client->server ws.ping() which does NOT reset the server->client proxy_read_timeout under test. Client stays otherwise idle so the read-timeout is the thing being tested."
  - "Every port reference is 8091 (host web) or 9980 (coolwsd native); the project-banned web port appears in none of the three files (comments reworded to avoid even the literal token, satisfying the harness's own `! grep -rn` gate)."

patterns-established:
  - "Reverse-proxy >60s WebSocket-survival smoke harness with a negative control — reusable against ANY coolwsd image; 04-04 wires it against the EdenDocs :scan image in CI."
  - "Additive-only: three NEW files under scripts/eden/; etc/nginx/coolwsd.conf and all upstream files byte-untouched."

requirements-completed: []   # DIST-02 spans 04-02/04-03/04-04; per repo precedent the LAST contributing TRD (04-04: image built+published+signed, harness run in CI) marks it complete. Stays Pending until then.
requirements-advanced: [DIST-02]

# Verification evidence
verification:
  gates_defined: 3
  gates_passed: 3
  auto_fix_cycles: 0
  tdd_evidence: false
  test_pairing: false
  live_e2e: deferred-to-04-04-ci   # macOS dev host has no Linux docker engine for the >60s live WS run

# Metrics
duration: 6min
completed: 2026-07-09
---

# Objective 4 TRD 03: Reverse-Proxy Smoke Harness Summary

**Additive DIST-02 reverse-proxy harness — `scripts/eden/proxy-smoke-test.sh` stands up coolwsd + a throwaway `nginx:alpine` sharing coolwsd's network namespace (so the in-tree `etc/nginx/coolwsd.conf` is reused BYTE-FOR-BYTE with its `proxy_pass localhost:9980` intact), opens a WebSocket document session through the proxy on host port 8091, holds it idle 80s (>60s), and asserts it is still alive AND that `proxy_read_timeout 36000s` is loaded via `nginx -T`. A short-timeout (5s) negative control proves the harness discriminates the 60s-default failure mode. The snippet mounts at `/etc/nginx/snippets/` (never conf.d); the project-banned web port appears nowhere; coolwsd.conf is never edited. The live >60s run is deferred to 04-04's CI (macOS dev host has no Linux docker engine).**

## Performance

- **Duration:** ~6 min
- **Completed:** 2026-07-09
- **Tasks:** 2
- **Files created:** 3

## Accomplishments
- New additive `scripts/eden/nginx-test-server.conf`: a minimal `server { listen 8091; include /etc/nginx/snippets/coolwsd.conf; }` wrapper that pulls in the verbatim coolwsd.conf snippet and declares NO `/cool/` directives of its own (the snippet owns the 36000s timeout + Upgrade/Connection headers + `^~ /cool/` prefix match).
- New additive `scripts/eden/nginx-test-server-negative.conf`: the discrimination control with a deliberately short `proxy_read_timeout 5s;` on `/cool/`, so an idle WS is killed fast — proving the harness detects the failure it exists to catch.
- New additive `scripts/eden/proxy-smoke-test.sh`: self-asserts coolwsd.conf byte-integrity, stands up coolwsd (`-p 8091:8091`) + nginx (`--network container:`), runs a POSITIVE case (verbatim config, expect ALIVE at 80s, plus a `nginx -T` structural assert of the loaded 36000s override) and a NEGATIVE case (short-timeout config, expect DEAD before 80s). WS client is `python3 websockets` installed on-demand; no repo dependency added.
- Zero upstream edits: `etc/nginx/coolwsd.conf` byte-unmodified (`git diff --quiet HEAD` clean); all three deliverables live under `scripts/eden/`. No reference to the project-banned web port anywhere.

## Task Evidence

| Task | Verify Command (as run) | Exit Code | Status |
|---|---|---|---|
| 1: Author nginx wrapper + negative control | `grep 'listen 8091;' wrapper && grep 'include /etc/nginx/snippets/coolwsd.conf;' wrapper && ! grep proxy_read_timeout wrapper && grep 'proxy_read_timeout 5s;' negative && grep 'listen 8091;' negative && ! grep 8080 both && (nginx not local -> lint in nginx:alpine/CI)` | 0 | PASS |
| 2: Author proxy-smoke-test.sh | `chmod +x && bash -n && shellcheck -e SC2016,SC1091 && grep 'git diff --quiet HEAD -- etc/nginx/coolwsd.conf' && grep 'proxy_read_timeout 36000s' && grep 'network "container:' && grep 'HOLD_SECS:-80' && (negative control wired 'dead') && ! grep 8080 && grep 8091` | 0* | PASS |

*Task 2 note: the TRD `<verify>` block's one `expect=dead` sub-check is an unsatisfiable predicate in the gate itself (see Deviations) — it greps for the literal `expect=dead` inside a process substitution whose content is always the string `negative.conf dead`, so it can never match regardless of script content. Its clear INTENT (the negative control's `run_case` is wired with the `dead` expectation) is satisfied and was proven with the corrected predicate `grep -q 'nginx-test-server-negative.conf dead'` (matched line 83). All other sub-checks passed as written. coolwsd.conf byte-integrity and the shellcheck/bash -n lint additionally passed.

## Task Commits

Each task was committed atomically to eden-main:

1. **Task 1: Author the nginx wrapper configs (verbatim include + negative control)** — `fc2a47a2165` (feat)
2. **Task 2: Author proxy-smoke-test.sh (WS >60s survival + nginx -T structural assert + negative control)** — `f8918903de4` (test)

**Plan metadata:** committed with STATE.md/ROADMAP.md update (docs).

## Validation Gate Results

| Gate | Command | Exit Code | Status |
|---|---|---|---|
| lint | `bash -n scripts/eden/proxy-smoke-test.sh; shellcheck -e SC2016,SC1091 scripts/eden/proxy-smoke-test.sh` | 0 | PASS (shellcheck present locally — clean) |
| test | `git diff --quiet HEAD -- etc/nginx/coolwsd.conf && grep -q 'proxy_read_timeout 36000s' scripts/eden/proxy-smoke-test.sh` | 0 | PASS |
| build | `true # no build artifact; the harness is the deliverable` | 0 | PASS |

Supplementary local proof: embedded Python WS-leg syntax-checked with `py_compile` (OK); comprehensive cross-file sweep — zero `8080`, coolwsd.conf mounted to `/etc/nginx/snippets/` (never conf.d), only the wrapper mounted to `conf.d/server.conf`, wrapper has no `/cool/` location directive, negative control `proxy_read_timeout 5s`, `--network container:` co-location present, `git diff --quiet HEAD` self-assert present.

## Post-TRD Verification

- **Auto-fix cycles used:** 0 (no code auto-fixes; two cosmetic comment rewords + one gate-intent correction — see Deviations)
- **Must-haves verified:** 4/4 truths structurally/lint-verified (verbatim-reuse harness on 8091 with byte-integrity self-assert; two-legged positive proof incl. nginx -T 36000s structural leg; negative control asserted to die; shared-namespace co-location keeping proxy_pass localhost:9980 valid). The LIVE >60s end-to-end WS run is deferred to 04-04 CI (macOS dev host is Linux-docker-incapable, per TRD hard constraint).
- **Gate failures:** None (the single unsatisfiable `expect=dead` sub-check is a defect in the TRD gate, not the deliverable — intent proven; documented under Deviations).

## Negative-Control Discriminator

- **Design:** primary discriminator is **live death** — the negative control's `proxy_read_timeout 5s` is far shorter than coolwsd's generous keepalive/idle cadence, so the idle WS is torn down well before t=80s while the positive session (36000s) stays open.
- **Complementary structural leg:** the POSITIVE case additionally asserts `nginx -T` shows `proxy_read_timeout 36000s` loaded on `/cool/`, so a keepalive-masked live pass is still backed structurally.
- **Which discriminator actually fires** (live death vs an added `nginx -T` assert of the negative 5s timeout, per error_recovery if coolwsd pings mask the death) is recorded when the live run executes in **04-04's CI** — locally the run is deferred (no Linux docker engine).

## TDD Exceptions

TDD-EXCEPTION (declared in TRD frontmatter): infra/verification harness — a bash + nginx smoke test. The harness IS the test (a >60s WebSocket-survival assertion with a negative control); `bash -n` + `shellcheck` + `nginx -t` (inside nginx:alpine) are the local/CI gate, and the against-EdenDocs-image run is 04-04's CI. The resolver's (app,feature) widget/Maestro profile does not apply.

## Files Created/Modified
- `scripts/eden/proxy-smoke-test.sh` — WS >60s survival harness: coolwsd + nginx:alpine shared-namespace stand-up, byte-integrity self-assert, positive (alive@80s + nginx -T 36000s) and negative (dead<80s) cases; port 8091 host / 9980 native only.
- `scripts/eden/nginx-test-server.conf` — server{} on 8091 that includes coolwsd.conf verbatim via /etc/nginx/snippets/; no /cool/ directives of its own.
- `scripts/eden/nginx-test-server-negative.conf` — negative control with proxy_read_timeout 5s.

## Decisions Made
- **Verbatim reuse via shared namespace, not config edit.** `--network container:edendocs-proxy-coolwsd` keeps coolwsd.conf's `proxy_pass http://localhost:9980` valid with zero edits; the harness FATAL-exits if `git diff --quiet HEAD -- etc/nginx/coolwsd.conf` is dirty.
- **snippets/ mount, not conf.d.** coolwsd.conf is a bare location snippet -> `/etc/nginx/snippets/coolwsd.conf:ro`; only the server{} wrapper -> `conf.d/server.conf:ro`; `/dev/null` masks nginx:alpine's `conf.d/default.conf` so 8091 is free.
- **Two-legged positive proof** (live alive@80s + structural `nginx -T` 36000s) so a keepalive-masked live pass is still structurally backed.
- **Negative control `proxy_read_timeout 5s`** for fast, unambiguous idle-WS death; live death is the primary discriminator, structural nginx -T assert the complement.

## Deviations from Plan

Three minor, non-architectural deviations — none touched the harness's behavior or any upstream file:

1. **[Rule 1 — Gate defect] Unsatisfiable `expect=dead` sub-check in the Task 2 `<verify>` block.** The TRD gate contains `grep -q 'expect=dead' <(grep -o 'negative.conf dead' ... || echo 'negative.conf dead')`. The process substitution always yields the string `negative.conf dead`, which can never contain `expect=dead` — so this one predicate fails for ANY script content (a self-contradictory gate, not a deliverable fault). I ran the gate as written (per hard constraint), isolated the defect, and verified its clear INTENT ("the negative control's run_case carries the `dead` expectation") with the corrected predicate `grep -q 'nginx-test-server-negative.conf dead'` (matched line 83). The planning TRD was NOT edited (additive-only + planning doc). Every other sub-check passed verbatim.

2. **[Rule 1 — Constraint reconciliation] Reworded the "NEVER 8080" comments to remove the literal token.** The TRD's sample comment text for the two nginx configs literally contained "8080" (in "NEVER 8080" warnings), which trips the same files' `! grep -rn '8080'` gate AND the objective's stricter success criterion "No file references port 8080". Reworded to "the project-banned web port is forbidden" so the files contain zero `8080` tokens while preserving the warning's intent. Purely cosmetic; no behavior change.

3. Minor formatting-only cleanup of the embedded Python heredoc (split a compound `import subprocess; subprocess.run(...);` line onto separate lines and stripped trailing whitespace) — semantically identical, `py_compile`-clean, shellcheck-clean. Not a functional change.

No Rule 4 (architectural) deviations. No auto-fix cycles on the harness logic itself.

## Issues Encountered
None blocking. macOS dev host cannot run the live >60s Linux-docker WS test; per the TRD hard constraint the local gate is lint-level (`bash -n` + `shellcheck` + structural greps + coolwsd.conf byte-integrity) and the real end-to-end proof (coolwsd + nginx:alpine, WS alive@80s, negative control dies) runs in **04-04's CI** against the EdenDocs image.

## Hard-Constraint Confirmations
- **snippets/ mount path (NOT conf.d):** `-v "$PWD/etc/nginx/coolwsd.conf:/etc/nginx/snippets/coolwsd.conf:ro"`; only the wrapper goes to `conf.d/server.conf`. Confirmed by grep + false-positive re-check (the only `/cool/` in the wrapper is a comment; no location directive).
- **>60s hold + 36000s structural assert + negative control:** `HOLD_SECS:-80` (>60), `nginx -T | grep -q 'proxy_read_timeout 36000s'` on the positive case, negative control `proxy_read_timeout 5s` expected DEAD before 80s.
- **Port discipline:** zero `8080` across all three files; host web = 8091, coolwsd native = 9980.
- **Verbatim reuse + additive:** `etc/nginx/coolwsd.conf` byte-unmodified (`git diff --quiet HEAD` clean); the two task commits touch only `scripts/eden/` files.
- **Live run:** possible only on a Linux docker engine -> deferred to 04-04 CI; local verification was lint/structural, exactly as the TRD prescribes.

## Self-Check: PASSED
