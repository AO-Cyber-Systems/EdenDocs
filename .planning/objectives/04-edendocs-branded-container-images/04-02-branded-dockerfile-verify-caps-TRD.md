---
objective: 04-edendocs-branded-container-images
trd: "02"
type: standard
wave: 1
depends_on: []
files_modified:
  - docker/eden/Dockerfile
  - docker/eden/.dockerignore
  - docker/eden/.gitignore
autonomous: true
requirements: [DIST-01, DIST-02]
# TDD-EXCEPTION: infra/packaging work — a Dockerfile. The verify-caps stage IS the
# in-build test (getcap assertions fail the build), and hadolint + structural greps
# are the local gate; the full image build is proven in 04-04's CI. The resolver's
# (app,feature) widget/Maestro TDD profile does not apply to a container image spec.

must_haves:
  truths:
    - "docker/eden/Dockerfile is a NEW additive file (zero edits to docker/from-source/{Ubuntu,Debian}), parameterized ARG base_image=ubuntu:24.04, that borrows upstream from-source's apt list + setcap + adduser + systemplate-setup + ENTRYPOINT verbatim"
    - "The build is 3-stage: assemble (COPY instdir / + setcap) → verify-caps (COPY --from=assemble the two capability-bearing binaries, getcap-assert cap_sys_chroot on coolforkit-caps and cap_sys_admin on coolmount, fail the build on loss) → runtime — and the runtime stage is FORCED to pull the verify-caps proof marker so verify-caps runs on every build"
    - "The final image EXPOSEs 9980 (never 8080), runs as USER 100, carries a HEALTHCHECK using coolwsd --probe, and the ENTRYPOINT drives coolwsd with --use-env-vars + the same --o: overrides as upstream from-source"
    - "The Dockerfile carries NO LABEL directives and NO hardcoded version/commit strings (LABELs are injected at CI level in 04-04), and no Collabora trademark on any image-identity surface"
  artifacts:
    - "docker/eden/Dockerfile"
    - "docker/eden/.dockerignore"
    - "docker/eden/.gitignore"
  key_links:
    - "docker/eden/Dockerfile COPY instdir / → docker/eden/instdir (staged by 04-01's build-release.sh: usr/bin/cool* + opt/collaboraoffice)"
    - "verify-caps stage COPY --from=assemble → getcap assertion (DIST-02: manufactures the from-packages xattr-loss risk surface and proves capabilities survive it, ported onto from-source content)"
    - "runtime ENTRYPOINT coolwsd --o:file_server_root_path=/usr/share/coolwsd → the EdenDocs branding overlays 04-01 staged there (favicon, welcome, branding.css/js)"
---

<objective>
Add the single new file that makes an EdenDocs image a from-source image with a real capability-check gate: `docker/eden/Dockerfile` — a 3-stage build (assemble → verify-caps → runtime) that ports the `docker/from-packages` `verify-caps` PATTERN onto `docker/from-source/Ubuntu`'s CONTENT. It reuses upstream's exact apt package list, `setcap`, `adduser`, `coolwsd-systemplate-setup`, `EXPOSE 9980`, `USER 100`, and `ENTRYPOINT` — split across stages so a `COPY --from` genuinely exercises (and proves) `security.capability` xattr preservation, satisfying DIST-02's "verify-caps stage still passes" literally rather than reinterpreting it away.

Purpose: DIST-01 requires the from-source pattern (not from-packages). from-source's single-stage design has no `verify-caps` stage today (it exists only in from-packages), so DIST-02's capability guarantee must be built new here. `ARG base_image` keeps Debian a trivial v2 flip while only Ubuntu 24.04 ships/tests in v1. LABELs stay out of the Dockerfile (injected by CI in 04-04) so the file needs no per-release edits.

Output: `docker/eden/Dockerfile`, `docker/eden/.dockerignore`, `docker/eden/.gitignore` — committed on eden-main.
</objective>

<file_tree>
docker/eden/
├── Dockerfile        ← CREATE (3-stage: assemble → verify-caps → runtime)
├── .dockerignore     ← CREATE (keep the build context small; do not send .git etc.)
├── .gitignore        ← CREATE (ignore the build-time instdir/ staged by 04-01)
├── engine-pin.env      (04-01 — do NOT ignore)
└── instdir/            (build-time only, staged by build-release.sh — gitignored)
docker/from-source/    (UNTOUCHED upstream — Ubuntu, Debian, build.sh, README)
docker/from-packages/  (UNTOUCHED upstream — verify-caps PATTERN source)
</file_tree>

<execution_context>
@~/.claude/devflow/workflows/execute-trd.md
@~/.claude/devflow/templates/summary.md
</execution_context>

<embedded_context>

<codebase_examples>
Borrow the CONTENT block verbatim from `docker/from-source/Ubuntu` (the apt list, setcap, adduser, systemplate-setup, EXPOSE 9980, USER 100, ENTRYPOINT) — reproduced here so you do not need to re-read it:

```dockerfile
FROM ubuntu:24.04
RUN apt-get update && \
    apt-get -y install libpng16-16 fontconfig adduser cpio tzdata \
               findutils nano libcap2-bin openssl openssh-client \
               libxcb-shm0 libxcb-render0 libxrender1 libxext6 \
               fonts-wqy-zenhei fonts-wqy-microhei fonts-droid-fallback \
               fonts-noto-cjk ca-certificates
COPY /instdir /
RUN setcap cap_fowner,cap_chown,cap_sys_chroot=ep /usr/bin/coolforkit-caps && \
    setcap cap_sys_admin=ep /usr/bin/coolmount && \
    adduser --quiet --system --group --home /opt/cool cool && \
    rm -rf /opt/cool && mkdir -p /opt/cool/child-roots /opt/cool/cache && \
    coolwsd-systemplate-setup /opt/cool/systemplate /opt/collaboraoffice >/dev/null 2>&1 && \
    touch /var/log/coolwsd.log && chown cool:cool /var/log/coolwsd.log && \
    chown -R cool:cool /opt/ && chown -R cool:cool /etc/coolwsd && chmod g+w /etc/passwd
EXPOSE 9980
USER 100
ENTRYPOINT ["/usr/bin/coolwsd", "--use-env-vars", \
  "--o:sys_template_path=/opt/cool/systemplate", \
  "--o:child_root_path=/opt/cool/child-roots", \
  "--o:file_server_root_path=/usr/share/coolwsd", \
  "--o:cache_files.path=/opt/cool/cache", \
  "--o:logging.color=false", "--o:stop_on_config_change=true"]
```

Borrow the PATTERN (the verify-caps stage) from `docker/from-packages/Dockerfile` lines 163-171 + the forcing COPY on line 201:

```dockerfile
FROM builder AS verify-caps
COPY --from=builder /rootfs/usr/bin/coolforkit-caps /rootfs/usr/bin/coolmount /probe/
RUN set -eu; \
    fc="$(/sbin/getcap /probe/coolforkit-caps)"; mc="$(/sbin/getcap /probe/coolmount)"; \
    case "$fc" in *cap_sys_chroot*) ;; *) echo "FATAL: cap_sys_chroot lost on COPY --from" >&2; exit 1;; esac; \
    case "$mc" in *cap_sys_admin*) ;; *) echo "FATAL: cap_sys_admin lost on COPY --from" >&2; exit 1;; esac; \
    touch /caps-verified
# ...later, in the runtime stage, to FORCE verify-caps to run every build:
COPY --from=verify-caps /caps-verified /opt/cool/.caps-verified
```
</codebase_examples>

<anti_patterns>
- **Do NOT edit `docker/from-source/Ubuntu` (or Debian) to add stages** — the research recommends a NEW `docker/eden/Dockerfile` precisely to keep upstream files byte-untouched (zero divergence-allowlist entries). Objective 4 adds ZERO upstream-file divergences.
- **Do NOT put LABEL directives in the Dockerfile** — they are injected at CI level via docker/metadata-action in 04-04, so the Dockerfile carries no version/commit strings and needs no per-release edits. from-packages hardcodes Collabora LABELs; we do not.
- **Do NOT reintroduce any Collabora trademark on an image-identity surface** (no `author="Collabora…"`, no Collabora in comments that become metadata). Image identity is EdenDocs (checklist rows land in 04-04).
- **NEVER EXPOSE or reference 8080** — coolwsd is 9980; the reverse-proxy host check (04-03/04-04) is 8091.
- **Do not compile inside the Dockerfile** — this is a from-source *assembly* image (COPY a pre-built instdir), not a compiling image. No C++ toolchain, no `make` in any stage (Pitfall 8's ccache-in-Docker complexity is moot by design).
- **A "preserve verify-caps" task with no create step is a no-op** (Pitfall 1): from-source has no verify-caps to preserve — you are CREATING it here.
</anti_patterns>

<error_recovery>
- **`COPY --from=assemble / /` in the runtime stage errors or bloats absurdly** → fall back to `FROM assemble` for the final stage (single filesystem, caps already set directly on it) and keep only `COPY --from=verify-caps /caps-verified /opt/cool/.caps-verified` to force the gate. This still manufactures + asserts the COPY --from xattr risk in verify-caps and is more faithful to from-source's single-filesystem setcap semantics. Note the choice in the SUMMARY.
- **hadolint DL3008 (pin apt versions) / DL3009 (delete apt lists)** → these mirror upstream's own unpinned from-source style; add a targeted `# hadolint ignore=DL3008,DL3009` on the RUN, or `rm -rf /var/lib/apt/lists/*` for DL3009. Do not diverge from the upstream package set to satisfy a lint nit.
- **`setcap` fails "Operation not supported"** at build time → the assemble stage's base must have `libcap2-bin` (it does, in the apt list) and BuildKit must be enabled (DOCKER_BUILDKIT=1 / buildx, set in 04-04). This is an environment fix in the CI job, not a Dockerfile bug.
- **getcap not found in verify-caps** → it is provided by `libcap2-bin`; since verify-caps is `FROM assemble`, the package is already present. Use `/sbin/getcap` (absolute) as from-packages does.
- **Local `docker build` fails because docker/eden/instdir does not exist** → expected on a dev machine; instdir is staged by 04-01's build-release.sh in CI. Local verification here is hadolint + structural greps only (see verify).
</error_recovery>

</embedded_context>

<context>
@.planning/PROJECT.md
@.planning/objectives/04-edendocs-branded-container-images/4-RESEARCH.md
@docker/from-source/Ubuntu
@docker/from-packages/Dockerfile
</context>

<research_context>
From 4-RESEARCH.md (HIGH confidence) — Open Questions 2 + 3 resolved:
- Q2 → Use Pattern 2 (literal stage-split verify-caps). from-source's single-stage setcap has no inherent COPY --from xattr-loss risk, but restructuring into stages to exercise COPY --from and assert capability survival is a strictly stronger, CI-enforced guarantee and satisfies DIST-02's wording directly.
- Q3 → Ubuntu 24.04 ONLY for v1; parameterize `ARG base_image` so Debian is a trivial later flip. Ubuntu 24.04 matches the ubuntu-latest runner userland, minimizing engine/runtime ABI mismatch.
- LABELs applied via docker/metadata-action + build-push-action `labels:` (CI-level) — the Dockerfile stays static.
- HEALTHCHECK `coolwsd --probe` needs no curl/shell and reads the image's own config to pick port/scheme — recommended on the runtime stage.
- The full Pattern 2 Dockerfile is in 4-RESEARCH.md §"Pattern 2" — use it as the canonical shape.
</research_context>

<gotchas>
- `# syntax=docker/dockerfile:1.4` on line 1 enables the modern frontend (needed for reliable xattr preservation on COPY --from and for --mount if ever added).
- Put `ARG base_image="ubuntu:24.04"` BEFORE the first `FROM` and reference `${base_image}` in each `FROM` that needs it — an ARG before FROM is global and must be re-declared inside a stage only if used in RUN.
- `USER 100` (numeric) matches upstream from-source for Kubernetes Pod Security compatibility — keep numeric, not the `cool` name.
- The runtime stage's HEALTHCHECK `--start-period=120s` matters: cold coolwsd + engine load can exceed 60s; too-short a start period flaps the healthcheck (04-04's boot-health gate waits on this).
- .dockerignore must NOT exclude `instdir/` (that IS the build context); it should exclude VCS/planning cruft so the context stays lean. .gitignore is the opposite concern — it MUST ignore the build-time `instdir/` so it is never committed.
</gotchas>

<tasks>

<task type="auto">
  <name>Task 1: Author docker/eden/Dockerfile (3-stage assemble → verify-caps → runtime)</name>
  <files>docker/eden/Dockerfile</files>
  <action>
Create `docker/eden/Dockerfile` following 4-RESEARCH.md Pattern 2 (canonical), reproduced with the exact structure to author:

```dockerfile
# syntax=docker/dockerfile:1.4
#
# This Source Code Form is subject to the terms of the Mozilla Public
# License, v. 2.0. If a copy of the MPL was not distributed with this
# file, You can obtain one at http://mozilla.org/MPL/2.0/.
#
# EdenDocs branded container image — from-source pattern (DIST-01), with a
# verify-caps capability-check stage ported from docker/from-packages (DIST-02).
# NO LABELs here: OCI labels/annotations are injected at CI level
# (docker/metadata-action) in .github/workflows/docker-publish.yml.
ARG base_image="ubuntu:24.04"

# ---- Stage 1: assemble — upstream from-source content, single filesystem ----
FROM ${base_image} AS assemble
RUN apt-get update && \
    apt-get -y install libpng16-16 fontconfig adduser cpio tzdata \
      findutils libcap2-bin openssl openssh-client \
      libxcb-shm0 libxcb-render0 libxrender1 libxext6 \
      fonts-wqy-zenhei fonts-wqy-microhei fonts-droid-fallback \
      fonts-noto-cjk ca-certificates && \
    rm -rf /var/lib/apt/lists/*
# The build context root is docker/eden/ which contains instdir/ (staged by
# scripts/eden/build-release.sh: usr/bin/cool* + opt/collaboraoffice + branding).
COPY instdir /
RUN setcap cap_fowner,cap_chown,cap_sys_chroot=ep /usr/bin/coolforkit-caps && \
    setcap cap_sys_admin=ep /usr/bin/coolmount && \
    adduser --quiet --system --group --home /opt/cool cool && \
    rm -rf /opt/cool && mkdir -p /opt/cool/child-roots /opt/cool/cache && \
    coolwsd-systemplate-setup /opt/cool/systemplate /opt/collaboraoffice >/dev/null 2>&1 && \
    touch /var/log/coolwsd.log && chown cool:cool /var/log/coolwsd.log && \
    chown -R cool:cool /opt/ && chown -R cool:cool /etc/coolwsd && chmod g+w /etc/passwd

# ---- Stage 2: verify-caps — exercise COPY --from and assert caps survived ----
FROM assemble AS verify-caps
COPY --from=assemble /usr/bin/coolforkit-caps /usr/bin/coolmount /probe/
RUN set -eu; \
    fc="$(/sbin/getcap /probe/coolforkit-caps)"; mc="$(/sbin/getcap /probe/coolmount)"; \
    echo "post-COPY caps -- coolforkit-caps: ${fc:-<none>} | coolmount: ${mc:-<none>}"; \
    case "$fc" in *cap_sys_chroot*) ;; *) echo "FATAL: cap_sys_chroot lost on COPY --from" >&2; exit 1;; esac; \
    case "$mc" in *cap_sys_admin*) ;; *) echo "FATAL: cap_sys_admin lost on COPY --from" >&2; exit 1;; esac; \
    touch /caps-verified

# ---- Stage 3: runtime ----
FROM ${base_image}
COPY --from=assemble / /
# Force the verify-caps gate into every build: this pulls in stage 2, which
# fails the build if the security.capability xattr did not survive COPY --from.
COPY --from=verify-caps /caps-verified /opt/cool/.caps-verified
EXPOSE 9980
USER 100
HEALTHCHECK --interval=30s --timeout=10s --start-period=120s --retries=3 \
    CMD ["/usr/bin/coolwsd", "--probe"]
ENTRYPOINT ["/usr/bin/coolwsd", "--use-env-vars", \
    "--o:sys_template_path=/opt/cool/systemplate", \
    "--o:child_root_path=/opt/cool/child-roots", \
    "--o:file_server_root_path=/usr/share/coolwsd", \
    "--o:cache_files.path=/opt/cool/cache", \
    "--o:logging.color=false", "--o:stop_on_config_change=true"]
```

If `COPY --from=assemble / /` proves problematic at build time in 04-04, the documented fallback is `FROM assemble` for stage 3 (keep the verify-caps forcing COPY) — see error_recovery.
  </action>
  <verify>
command -v hadolint >/dev/null && hadolint docker/eden/Dockerfile || echo "hadolint not installed — skipping (CI in 04-04 runs it)"; grep -q '^ARG base_image="ubuntu:24.04"' docker/eden/Dockerfile && test "$(grep -cE '^FROM ' docker/eden/Dockerfile)" -eq 3 && grep -q 'AS assemble' docker/eden/Dockerfile && grep -q 'AS verify-caps' docker/eden/Dockerfile && grep -q 'COPY --from=assemble /usr/bin/coolforkit-caps' docker/eden/Dockerfile && grep -q 'cap_sys_chroot' docker/eden/Dockerfile && grep -q 'cap_sys_admin' docker/eden/Dockerfile && grep -q 'COPY --from=verify-caps /caps-verified' docker/eden/Dockerfile && grep -q 'EXPOSE 9980' docker/eden/Dockerfile && grep -q 'USER 100' docker/eden/Dockerfile && grep -q 'coolwsd", "--probe"' docker/eden/Dockerfile && ! grep -qi '^LABEL' docker/eden/Dockerfile && ! grep -qi 'collabora' <(grep -iE '^(LABEL|#.*author|ARG author)' docker/eden/Dockerfile) && ! grep -q '8080' docker/eden/Dockerfile
  </verify>
  <done>docker/eden/Dockerfile exists, is hadolint-clean (or clean modulo justified DL3008/DL3009 ignores), has exactly 3 FROM stages with ARG base_image=ubuntu:24.04, a verify-caps stage that COPY --from=assemble the two cap binaries and asserts cap_sys_chroot + cap_sys_admin, a runtime stage that forces the verify-caps marker + EXPOSE 9980 + USER 100 + coolwsd --probe HEALTHCHECK + the from-source ENTRYPOINT, carries no LABEL directives and no Collabora trademark on identity surfaces, and never references 8080.</done>
  <recovery>hadolint failures on DL3008/DL3009 → add a scoped `# hadolint ignore=` matching upstream's unpinned style (do not diverge the package set). If the 3-FROM count is off because base_image reuse confused the grep, count stages manually. Full build failures are diagnosed in 04-04's CI, not here.</recovery>
</task>

<task type="auto">
  <name>Task 2: Add .dockerignore + .gitignore and run the local structural/hadolint gate</name>
  <files>docker/eden/.dockerignore, docker/eden/.gitignore</files>
  <action>
Create `docker/eden/.dockerignore` — keep the build context lean; the context root is `docker/eden/`, so only `instdir/`, `Dockerfile`, and `engine-pin.env` should ever be needed by the build. Exclude everything else that could accidentally sit in the dir:

```
# Build context is docker/eden/. instdir/ is the payload; keep it. Exclude cruft
# so `docker build docker/eden/` never uploads planning/VCS/editor files.
**/.git
**/.gitignore
**/.DS_Store
*.md
```
(Do NOT list `instdir` here — it is the payload the build COPYs.)

Create `docker/eden/.gitignore` — the build-time `instdir/` is a heavy generated artifact (staged by build-release.sh); it must never be committed:

```
# Build-time staging tree produced by scripts/eden/build-release.sh — never commit.
instdir/
```
(Do NOT ignore `Dockerfile`, `.dockerignore`, or `engine-pin.env` — those are tracked.)

Then run the local gate: hadolint if present, and confirm the ignore files behave (git does not see instdir/, git DOES see Dockerfile + engine-pin.env).
  </action>
  <verify>
grep -q '^instdir/' docker/eden/.gitignore && ! grep -qx 'instdir' docker/eden/.dockerignore && ! grep -qx 'instdir/' docker/eden/.dockerignore && git check-ignore -q docker/eden/instdir 2>/dev/null; test $? -le 1 && git check-ignore docker/eden/Dockerfile >/dev/null 2>&1 && { echo "ERROR: Dockerfile is gitignored"; exit 1; } || true; git check-ignore docker/eden/engine-pin.env >/dev/null 2>&1 && { echo "ERROR: engine-pin.env is gitignored"; exit 1; } || true; command -v hadolint >/dev/null && hadolint docker/eden/Dockerfile || echo "hadolint absent — 04-04 CI enforces it"
  </verify>
  <done>docker/eden/.gitignore ignores instdir/ (git check-ignore confirms) but NOT Dockerfile/engine-pin.env; docker/eden/.dockerignore excludes VCS/planning cruft without excluding the instdir payload; hadolint (if installed) is clean.</done>
  <recovery>If git check-ignore shows Dockerfile or engine-pin.env as ignored, the .gitignore pattern is too broad — narrow it to exactly `instdir/`. If .dockerignore accidentally lists instdir, remove that line (it would empty the build context).</recovery>
</task>

</tasks>

<validation_gates>
<lint>command -v hadolint >/dev/null && hadolint docker/eden/Dockerfile || echo "hadolint not installed locally; enforced in 04-04 CI"</lint>
<test># Full `docker build` proven in 04-04 CI (needs docker/eden/instdir staged on Linux). Local = structural greps + hadolint.
grep -cE '^FROM ' docker/eden/Dockerfile | grep -qx 3</test>
<build>true # image build runs in 04-04's docker-publish.yml, not per-task</build>
</validation_gates>

<verification>
1. `docker/eden/Dockerfile` is additive (docker/from-source/Ubuntu + Debian byte-untouched); `git status docker/from-source/` shows no changes (DIST-01: from-source pattern, not from-packages, zero upstream edits).
2. 3 stages: assemble (COPY instdir / + setcap), verify-caps (COPY --from=assemble + getcap assert cap_sys_chroot/cap_sys_admin + touch marker), runtime (forces the marker, EXPOSE 9980, USER 100, --probe HEALTHCHECK, from-source ENTRYPOINT) (DIST-02: verify-caps).
3. ARG base_image=ubuntu:24.04 (Debian deferrable via override); no LABELs in the file; no port 8080; no Collabora identity string.
4. .gitignore hides the build-time instdir/; .dockerignore keeps the context lean without dropping the payload.
</verification>

<success_criteria>
- A new, additive `docker/eden/Dockerfile` that builds an EdenDocs image from the from-source pattern (not from-packages) and carries a working verify-caps capability gate — the DIST-01 image spec + the DIST-02 verify-caps half, proven end-to-end by 04-04's CI build.
- Zero upstream-file edits; Ubuntu-24.04-only v1 with a trivial Debian flip via ARG.
</success_criteria>

<output>
After completion, create `.planning/objectives/04-edendocs-branded-container-images/04-02-SUMMARY.md`: the stage design (and whether the runtime stage used `COPY --from=assemble / /` or the `FROM assemble` fallback), the exact getcap assertions, confirmation docker/from-source/{Ubuntu,Debian} are byte-untouched, and any hadolint ignores added with justification.
</output>
