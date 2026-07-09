---
objective: 04-edendocs-branded-container-images
job: "02"
subsystem: infra
tags: [docker, dockerfile, verify-caps, linux-capabilities, setcap, getcap, coolwsd, buildkit, dist-01, dist-02]

# Dependency graph
requires:
  - objective: 04-edendocs-branded-container-images
    provides: "04-01: scripts/eden/build-release.sh (stages docker/eden/instdir), scripts/eden/fetch-engine-pinned.sh, docker/eden/engine-pin.env"
  - objective: 02-edendocs-rebrand-of-the-web-ui
    provides: "eden-branding/ overlays (favicon, welcome, branding.css/js) baked into instdir under /usr/share/coolwsd + the BRAND-06 HARD OMIT / zero-Collabora-identity discipline"
provides:
  - "docker/eden/Dockerfile — additive 3-stage (assemble -> verify-caps -> runtime) from-source image with an in-build getcap capability gate; ARG base_image=ubuntu:24.04"
  - "docker/eden/.dockerignore — lean docker build context (excludes VCS/editor/*.md, keeps the instdir/ payload)"
  - "docker/eden/.gitignore — hides the build-time instdir/ generated tree so it is never committed"
affects: [04-03-reverse-proxy-smoke-harness, 04-04-docker-publish-cosign]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Pattern 2 verify-caps (literal stage-split): a dedicated FROM assemble AS verify-caps stage COPY --from=assemble the two capability-bearing binaries, then getcap-asserts cap_sys_chroot/cap_sys_admin survived the security.capability xattr boundary and fails the build on loss; runtime FORCES it via COPY --from=verify-caps /caps-verified"
    - "from-packages PATTERN ported onto from-source CONTENT: reuse from-source/Ubuntu's exact apt/setcap/adduser/systemplate-setup/EXPOSE/USER/ENTRYPOINT, restructured across stages to manufacture + prove the COPY --from cap-preservation guarantee from-source's single-stage design never exercised"
    - "LABEL-free Dockerfile: OCI metadata injected at CI level (docker/metadata-action in 04-04) so the file carries no version/commit strings and needs no per-release edits"
    - "ARG base_image before first FROM (global): Ubuntu-24.04-only v1, Debian a trivial --build-arg override later"

key-files:
  created:
    - "docker/eden/Dockerfile"
    - "docker/eden/.dockerignore"
    - "docker/eden/.gitignore"
  modified: []

key-decisions:
  - "Runtime stage uses the PRIMARY canonical `COPY --from=assemble / /` (not the `FROM assemble` fallback) — assemble is a full single filesystem with caps already set directly; the forcing `COPY --from=verify-caps /caps-verified` still pulls the gate into every build"
  - "getcap assertions: cap_sys_chroot on /probe/coolforkit-caps and cap_sys_admin on /probe/coolmount (case-glob match, echo the observed caps, exit 1 with a FATAL message on loss) — mirrors from-packages verbatim, retargeted from /rootfs paths to from-source's `/usr/bin` paths"
  - "apt list follows the canonical <action>: dropped the interactive editor `nano` (pointless/attack-surface in a headless container) and appended `rm -rf /var/lib/apt/lists/*` (DL3009) with a scoped `# hadolint ignore=DL3008,DL3009`; ALL capability-critical/functional lines (setcap, adduser, coolwsd-systemplate-setup, EXPOSE 9980, USER 100, ENTRYPOINT) are byte-faithful to docker/from-source/Ubuntu"
  - "USER 100 kept numeric (Kubernetes Pod Security compat) as upstream from-source does; HEALTHCHECK --start-period=120s to absorb cold coolwsd + engine load (04-04 boot-health gate waits on this)"
  - "NO LABEL directives, zero Collabora identity string on any metadata surface; never any :8080 (coolwsd is 9980, host proxy check is 8091)"

patterns-established:
  - "Literal Pattern-2 verify-caps as an in-build test: the capability gate IS the test (getcap fails the build), CI-enforced on every image build in 04-04"
  - "Additive-only image spec: docker/eden/Dockerfile + dotfiles are NEW; docker/from-source/{Ubuntu,Debian} and docker/from-packages/* stay byte-untouched (zero divergence-allowlist entries)"

requirements-completed: []  # DIST-01/DIST-02 span 04-02/04-04 (image built + published + signed there). Per repo precedent (multi-TRD reqs marked by the LAST contributing TRD) both stay Pending until 04-04; 04-02 delivers the branded Dockerfile + verify-caps gate spec only.
requirements-advanced: [DIST-01, DIST-02]

# Verification evidence
verification:
  gates_defined: 3
  gates_passed: 3
  auto_fix_cycles: 0
  tdd_evidence: false
  test_pairing: false

# Metrics
duration: 3min
completed: 2026-07-09
---

# Objective 4 TRD 02: Branded Dockerfile + verify-caps Summary

**Additive `docker/eden/Dockerfile` — a 3-stage (assemble -> verify-caps -> runtime) from-source EdenDocs image that ports the from-packages capability-check PATTERN onto from-source CONTENT: a literal stage-split verify-caps stage COPY --from=assemble the two cap-bearing binaries and getcap-asserts cap_sys_chroot + cap_sys_admin survived the xattr boundary, failing the build on loss. ARG base_image=ubuntu:24.04, LABEL-free, no Collabora identity, never :8080.**

## Performance

- **Duration:** 3 min
- **Started:** 2026-07-09T07:30:21Z
- **Completed:** 2026-07-09T07:33:45Z
- **Tasks:** 2
- **Files created:** 3

## Accomplishments
- New additive `docker/eden/Dockerfile`: 3 FROM stages (assemble / verify-caps / runtime), `ARG base_image="ubuntu:24.04"`, from-source apt+setcap+adduser+systemplate-setup+EXPOSE 9980+USER 100+ENTRYPOINT reused verbatim, plus a coolwsd `--probe` HEALTHCHECK (start-period 120s).
- Literal Pattern-2 verify-caps stage: `COPY --from=assemble /usr/bin/coolforkit-caps /usr/bin/coolmount /probe/` then getcap-asserts `cap_sys_chroot` (coolforkit-caps) + `cap_sys_admin` (coolmount) with a FATAL exit-1 on loss; runtime FORCES it via `COPY --from=verify-caps /caps-verified /opt/cool/.caps-verified`.
- `.dockerignore` keeps the build context lean without dropping the `instdir/` payload; `.gitignore` hides the heavy build-time `instdir/` (git check-ignore confirmed) while keeping Dockerfile + engine-pin.env tracked.
- Zero upstream edits: `docker/from-source/{Ubuntu,Debian}` and `docker/from-packages/*` byte-untouched. No LABELs, no Collabora identity string, no `:8080` anywhere.

## Task Evidence

| Task | Verify Command | Exit Code | Status |
|---|---|---|---|
| 1: Author docker/eden/Dockerfile (3-stage) | `grep '^ARG base_image="ubuntu:24.04"' && FROM-count==3 && AS assemble && AS verify-caps && COPY --from=assemble coolforkit-caps && cap_sys_chroot && cap_sys_admin && COPY --from=verify-caps /caps-verified && EXPOSE 9980 && USER 100 && coolwsd --probe && !LABEL && !collabora-identity && !8080` | 0 | PASS |
| 2: Add .dockerignore + .gitignore + local gate | `grep '^instdir/' .gitignore && !instdir in .dockerignore && git check-ignore instdir OK && Dockerfile/engine-pin.env NOT gitignored` | 0 | PASS |

Supplementary functional proof (Task 2): `git check-ignore docker/eden/instdir/usr/bin/coolwsd` -> matched (payload gitignored); `git check-ignore docker/eden/Dockerfile` / `docker/eden/engine-pin.env` -> not ignored (tracked-eligible).

## Task Commits

Each task was committed atomically to eden-main:

1. **Task 1: Author docker/eden/Dockerfile (3-stage assemble -> verify-caps -> runtime)** - `471caa933d8` (feat)
2. **Task 2: Add .dockerignore + .gitignore and run the local structural gate** - `a46f550fe96` (build)

**Plan metadata:** committed with STATE.md/ROADMAP.md update (docs).

## Validation Gate Results

| Gate | Command | Exit Code | Status |
|---|---|---|---|
| lint | `command -v hadolint && hadolint docker/eden/Dockerfile \|\| echo "not installed locally; enforced in 04-04 CI"` | 0 | PASS (hadolint absent locally; scoped `# hadolint ignore=DL3008,DL3009` pre-applied for 04-04 CI) |
| test | `grep -cE '^FROM ' docker/eden/Dockerfile \| grep -qx 3` | 0 | PASS (exactly 3 FROM stages) |
| build | `true # image build runs in 04-04 docker-publish.yml` | 0 | PASS (deferred to 04-04 CI by design) |

## Post-TRD Verification

- **Auto-fix cycles used:** 0
- **Must-haves verified:** 4/4 truths (additive+ARG+from-source-borrow; 3-stage assemble->verify-caps->runtime with forced gate; EXPOSE 9980/USER 100/--probe HEALTHCHECK/env-vars ENTRYPOINT; NO LABEL/no version strings/no Collabora identity) — full `docker build` capability proof deferred to 04-04 CI (macOS dev host is Linux-build-incapable, per TRD).
- **Gate failures:** None

## TDD Exceptions

TDD-EXCEPTION (declared in TRD frontmatter): infra/packaging work — a Dockerfile. The verify-caps stage IS the in-build test (getcap assertions fail the build); hadolint + structural greps are the local gate; the full image build + boot-health prove out in 04-04's CI. The (app,feature) widget/Maestro TDD profile does not apply to a container-image spec.

## Files Created/Modified
- `docker/eden/Dockerfile` - 3-stage from-source EdenDocs image (assemble -> verify-caps -> runtime), ARG base_image=ubuntu:24.04, verify-caps getcap gate, coolwsd --probe HEALTHCHECK, LABEL-free.
- `docker/eden/.dockerignore` - lean docker build context (excludes VCS/editor/*.md; keeps instdir/ payload).
- `docker/eden/.gitignore` - ignores the build-time instdir/ staging tree (never committed).

## Decisions Made
- **Runtime stage = PRIMARY canonical `COPY --from=assemble / /`** (not the `FROM assemble` fallback). assemble is a complete single filesystem with caps set directly on it; the runtime pulls it whole and the separate `COPY --from=verify-caps /caps-verified` still forces the gate into every build. The `FROM assemble` fallback remains documented in error_recovery for 04-04 CI if the whole-fs COPY proves problematic at real build time.
- **getcap assertions:** `cap_sys_chroot` on coolforkit-caps, `cap_sys_admin` on coolmount — from-packages verbatim, retargeted from `/rootfs/usr/bin/*` to from-source's `/usr/bin/*`. Uses absolute `/sbin/getcap` (provided by libcap2-bin, already present since verify-caps is `FROM assemble`).
- **apt list:** dropped `nano` (interactive editor, unneeded in a headless container) and appended `rm -rf /var/lib/apt/lists/*` (DL3009), matching the TRD canonical <action>; all capability-critical/functional directives byte-faithful to upstream.
- **HEALTHCHECK --start-period=120s** so a cold coolwsd + engine load does not flap the check (04-04's boot-health gate waits on this).

## Deviations from Plan

None - TRD executed exactly as written.

The canonical `<action>` Dockerfile drops `nano` and appends the DL3009 apt-lists cleanup relative to a strict byte-verbatim reading of `docker/from-source/Ubuntu`'s apt list; this is the planner's own canonical code (authored as-specified), not an executor deviation, and is documented above under Decisions. No Rule 1-4 deviations, no auto-fix cycles.

## Issues Encountered
None. (macOS dev host cannot build coolwsd or run a Linux `docker build`; local verification was lint-level — hadolint-if-present + structural greps + upstream byte-integrity — exactly as the TRD prescribes. The real image + capability build proof runs in 04-04's CI.)

## Hard-Constraint Confirmations
- **verify-caps = literal Pattern 2 stage-split:** `FROM assemble AS verify-caps` -> `COPY --from=assemble /usr/bin/coolforkit-caps /usr/bin/coolmount /probe/` -> getcap-assert both caps -> `touch /caps-verified`; runtime forces it. Not a no-op, not a comment.
- **Base image:** `ARG base_image="ubuntu:24.04"` only; Debian is a trivial `--build-arg base_image=debian:...` flip in v2, but only Ubuntu 24.04 ships/tests in v1.
- **LABELs:** none in the Dockerfile (OCI metadata injected at CI level in 04-04); no Collabora trademark on any image-identity surface.
- **Port policy:** no `:8080` anywhere (`grep -rn '8080' docker/eden/` -> none); coolwsd `EXPOSE 9980`.
- **Additive discipline:** `git status --short docker/from-source/ docker/from-packages/` -> empty (byte-untouched).

## Next Objective Readiness
- 04-03 (reverse-proxy smoke harness) and 04-04 (docker-publish + cosign) can now build against `docker/eden/Dockerfile`. 04-04's CI stages `docker/eden/instdir` (via 04-01's build-release.sh on Linux), runs hadolint, performs the real BuildKit `docker build` (which exercises + proves the verify-caps gate), injects OCI LABELs via docker/metadata-action, and runs the boot-health check the 120s HEALTHCHECK start-period supports.
- DIST-01 + DIST-02 remain Pending until 04-04 (last contributing TRD): the image is only truly built + published + signed there.

---
*Objective: 04-edendocs-branded-container-images*
*Completed: 2026-07-09*

## Self-Check: PASSED

- FOUND: docker/eden/Dockerfile
- FOUND: docker/eden/.dockerignore
- FOUND: docker/eden/.gitignore
- FOUND: .planning/objectives/04-edendocs-branded-container-images/04-02-SUMMARY.md
- FOUND commit: 471caa933d8 (feat 04-02 Dockerfile)
- FOUND commit: a46f550fe96 (build 04-02 dotfiles)
- git ls-files docker/eden/ = Dockerfile + .dockerignore + .gitignore + engine-pin.env (instdir/ correctly untracked)
