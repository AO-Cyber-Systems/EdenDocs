# Objective 4: EdenDocs-Branded Container Images - Research

**Researched:** 2026-07-09
**Domain:** Docker multi-stage builds, LibreOffice/Collabora packaging, cosign/Sigstore keyless signing, GHCR publishing, WebSocket reverse-proxy behavior
**Confidence:** HIGH

<phase_requirements>
## Objective Requirements

| ID | Description | Research Support |
|----|-------------|-------------------|
| DIST-01 | EdenDocs-branded container image built on the `docker/from-source/{Ubuntu,Debian}` pattern (prebuilt engine + EdenDocs-built wsd/browser + eden-branding assets) — not `from-packages` | Architecture Patterns §"docker/from-source anatomy" + §"Recommended: docker/eden/Dockerfile"; Code Examples §1-3 |
| DIST-02 | Image preserves the `verify-caps` capability-check stage; a document session held open >60s through a real reverse proxy (not just direct 127.0.0.1:9980) | Common Pitfalls #1 (verify-caps doesn't exist in from-source today — must be added); Architecture Patterns §"Reverse-proxy test design"; Code Examples §4-5 |
| DIST-03 | Images signed with cosign, published to the AO-Cyber-Systems registry, carrying EdenDocs LABEL metadata | Standard Stack §"CI/CD publishing"; Code Examples §6-7; State of the Art (keyless cosign supersedes prior STACK.md guidance) |
</phase_requirements>

## Summary

EdenDocs already has everything it needs to satisfy DIST-01/02/03 without inventing new mechanisms — the work is almost entirely **assembly and CI wiring**, not new engineering. The upstream `docker/from-source/{Ubuntu,Debian}` Dockerfiles are single-stage, generic (`COPY /instdir /` + `setcap` + `adduser` + `ENTRYPOINT`), and contain **zero EdenDocs-specific content already baked into the image at Docker-build time** — all branding lives in the `instdir/` tree assembled *before* `docker build` runs (Objective 1/2's build pipeline already produces it). This means the upstream Dockerfiles can, in principle, be used byte-for-byte unmodified. The catch is `verify-caps`: that stage exists only in `docker/from-packages/Dockerfile` today (a multi-stage, `COPY --from=builder` build where BuildKit's xattr-preservation of `security.capability` genuinely is at risk). The from-source Dockerfiles are single-stage — `setcap` runs directly against the final image's own filesystem, so there is no `COPY --from` capability-loss risk to guard against, and no `verify-caps` stage exists to "still pass."

**Primary recommendation:** Add one new, additive file — `docker/eden/Dockerfile` — that ports the from-packages `verify-caps` PATTERN onto the from-source CONTENT model: a 3-stage build (`assemble` → `verify-caps` → final runtime) using the same `ubuntu:24.04` base and the same `COPY /instdir /` + `setcap` recipe as upstream's `docker/from-source/Ubuntu`, but split across stages so `COPY --from=assemble` genuinely exercises (and proves) xattr preservation, literally satisfying DIST-02's "verify-caps stage still passes" language rather than reinterpreting it away. This is a new file, not an edit to any upstream file — zero divergence-allowlist entries needed. LABELs are applied via `docker/metadata-action` + `build-push-action`'s `labels:` input (CI-level, not baked into the Dockerfile text), keeping the Dockerfile itself free of hardcoded version strings. The reverse-proxy smoke test needs no new nginx config — `etc/nginx/coolwsd.conf` (already `make install`-shipped) has the correct `36000s` `proxy_read_timeout` and `Upgrade`/`Connection` headers on a prefix-matched (`^~`) `/cool/` location, ready to drop into a throwaway `nginx:alpine` container for the >60s WS-session test. Signing follows the AO-Cyber-Systems org's **uniform keyless cosign** convention (Sigstore/GitHub OIDC, `id-token: write`, `cosign sign --yes`) — confirmed across every release workflow in `~/dev` (eden-biz, aodex-go, aodex-flutter, aocore, aocore-admin, aoinference) with zero key-based exceptions, which **supersedes** the project's own `.planning/research/STACK.md` guidance to "generate its own cosign keypair." CI resource risk is low: compilation happens on the host runner exactly as Objective 1 already does (native `make`), and `docker build` only copies a pre-built `instdir/` into a base image — no C++ compilation occurs inside the Dockerfile, so the BuildKit-cache-mount/ccache-in-Docker complexity that would otherwise apply to this objective is moot.

The one live, unresolved risk outside EdenDocs' control is the upstream `engine-main-assets.tar.gz` rolling tarball's volatility: it was republished three times in the ~36 hours preceding this research (one of the three versions SIGSEGVs kit startup), so a container image built straight off `fetch-engine-assets.sh`'s live rolling fetch is not reproducible and can silently ship a broken kit. Recommend pinning/caching a known-good engine snapshot for release builds specifically (see Open Questions #1 and Common Pitfalls #2).

## Standard Stack

### Core

| Tool | Version | Purpose | Why Standard |
|------|---------|---------|---------------|
| Docker BuildKit | via `docker/setup-buildx-action@v3` | Multi-stage image build, `COPY --from` xattr preservation | Already required by `docker/README` (`DOCKER_BUILDKIT=1`) for the existing `from-packages` build; org-standard across all `~/dev` release workflows |
| `docker/build-push-action` | v6 (v5 acceptable, matches aodex) | Build + push to GHCR, label/annotation injection, provenance/SBOM attestation | Used uniformly in eden-biz, aodex-go, aocore, aoinference release workflows |
| `docker/metadata-action` | v5 | Generates OCI-compliant tags + `org.opencontainers.image.*` labels from git ref/sha | Standard companion to build-push-action across the org; avoids hand-writing LABEL strings |
| `sigstore/cosign-installer` | v3 | Installs `cosign` CLI in CI | Used identically in aodex-go, aodex-flutter, aocore, aocore-admin, aoinference — zero variance |
| `cosign` (keyless/Sigstore) | latest via installer | Signs published image digests using GitHub Actions OIDC identity, no stored private key | **Org-uniform convention** — every checked release workflow uses `cosign sign --yes "${tag}@${DIGEST}"` with `id-token: write` permission and no `COSIGN_KEY`/`COSIGN_PASSWORD` secret anywhere in `~/dev` |
| `anchore/sbom-action/download-syft` | v0 | Installs Syft for CycloneDX SBOM generation | Paired with `cosign attest --type cyclonedx` in aodex-go/aocore pattern |
| `aquasecurity/trivy-action` | latest (matches aodex-go) | Blocking HIGH/CRITICAL vulnerability gate before push | Two-phase build pattern: build `:scan` tag with `load: true` → Trivy `exit-code: '1'` → only then build+push multi-arch |

### Supporting

| Tool | Version | Purpose | When to Use |
|------|---------|---------|-------------|
| `nginx:alpine` (or `nginx:1.27-alpine`) | latest stable alpine tag | Lightest containerized reverse proxy for the >60s WS-session smoke test | Reverse-proxy verification step only — never a production recommendation in this repo |
| `docker/setup-qemu-action@v3` | v3 | Multi-arch emulation (optional — only if `linux/arm64` is added later) | Not required for v1 — aodex-go's own primary published platform is `linux/amd64` only |
| GitHub Actions `actions/cache@v4` | existing (already wired) | ccache persistence for the *native* build step (unchanged from Objective 1) | Applies to `scripts/eden/build.sh`/a new release-build variant, not to `docker build` itself |

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| New additive `docker/eden/Dockerfile` (3-stage, ports verify-caps onto from-source content) | Reuse `docker/from-source/Ubuntu` 100% verbatim, apply LABELs only via CI `--label`/metadata-action | Verbatim reuse satisfies DIST-01's letter and needs zero new Dockerfile, but there is then no `verify-caps` stage anywhere in the built image — DIST-02's literal "verify-caps stage still passes" criterion cannot be satisfied. Rejected as primary because it fails a stated success criterion; documented as a valid fallback if the objective owner reinterprets DIST-02 as "N/A, single-stage setcap has no COPY --from risk" (see Open Questions #2). |
| Keyless cosign (Sigstore/GHA OIDC) | Key-based cosign (`cosign generate-key-pair`, `COSIGN_KEY`/`COSIGN_PASSWORD` secrets) | Key-based requires secret rotation/storage burden and is what `docker/cosign.pub` in this repo suggests Collabora itself may use for *their* releases — but that key is Collabora's own and not reusable. Zero AO-Cyber-Systems repos use key-based signing; adopting it here would introduce an org-wide inconsistency for no benefit. |
| ubuntu:24.04 base for the new Dockerfile | debian:stable (matches `docker/from-source/Debian`) | Ubuntu 24.04 matches `ubuntu-latest` GHA runner's own userland (glibc/library versions), minimizing ABI-mismatch risk between the engine tarball (built by upstream CI, likely also Ubuntu-based) and the runtime container. Debian remains a documented v2 option; the new Dockerfile's `FROM` can be parameterized via `ARG base_image` for near-zero cost if both are wanted later. |
| Separate `docker-publish.yml` workflow (own checkout, own native build) | Extend existing `build.yml` with docker/cosign/publish steps appended | `build.yml` already runs a debug-mode build (`--enable-debug`, per `configure.ac`'s own comment "disable some security pieces") intended for fast CI verification, not for shipping. A release image needs a **non-debug** configure invocation. Bolting docker+publish onto the existing debug-build job would either (a) ship a debug binary in the published image (wrong) or (b) require a second full native build inside the same job (pushes an already ~90min-budgeted job further, and the two build modes have different `--with-lo-path` semantics — see Common Pitfalls #4). A separate workflow with its own non-debug build step is cleaner and matches how aodex/aocore separate "CI verify" concerns from "release" concerns. |

**Installation (CI, no local install needed):**
```yaml
# .github/workflows steps, not a package manager install
- uses: docker/setup-buildx-action@v3
- uses: docker/login-action@v3
  with: { registry: ghcr.io, username: ${{ github.actor }}, password: ${{ secrets.GITHUB_TOKEN }} }
- uses: sigstore/cosign-installer@v3
- uses: anchore/sbom-action/download-syft@v0
```

## Architecture Patterns

### docker/from-source anatomy map (what exists today, verified by direct read)

| Component | File | Status | EdenDocs adaptation needed? |
|-----------|------|--------|------------------------------|
| Host build driver | `docker/from-source/build.sh` | Upstream-only; clones a *separate* `online` checkout, applies proprietary `online-branding` (license-restricted, not usable) | **Not reusable as-is.** EdenDocs needs its own release-build script (new file, e.g. `scripts/eden/build-release.sh`) — see Code Example §1. |
| Container Dockerfile (Ubuntu) | `docker/from-source/Ubuntu` | Generic, zero EdenDocs-specific content, zero LABELs, single-stage | **Reusable as build-context input pattern** — the new `docker/eden/Dockerfile` borrows this exact `FROM ubuntu:24.04` / apt-package list / `setcap` / `adduser` / `ENTRYPOINT` block verbatim, just split into stages for `verify-caps`. Upstream file itself is never edited. |
| Container Dockerfile (Debian) | `docker/from-source/Debian` | Same shape as Ubuntu variant | Deferred to v2 per Alternatives Considered above. |
| `verify-caps` stage | Exists ONLY in `docker/from-packages/Dockerfile` (lines ~163-171) | Guards `COPY --from=builder` xattr loss, a risk specific to *that* multi-stage design | **Must be added new** for from-source — see Common Pitfalls #1 and Code Example §2. |
| LABEL metadata | `docker/from-packages/Dockerfile` only (`author`, `description`, `release.notes`, `version`, `commit.history.core`) | from-source Dockerfiles confirmed clean (0 LABEL directives) | Apply via CI (`docker/metadata-action` + `build-push-action` `labels:` input), not hardcoded into any Dockerfile — see Code Example §6. |
| `HEALTHCHECK` | `docker/from-packages/Dockerfile` only (`coolwsd --probe`) | from-source has none | Recommended addition to the new Dockerfile's final stage — `coolwsd --probe` needs no curl/shell, works in any base. |
| Runtime config generation | `coolwsd.xml.in` → `make install` → `${sysconfdir}/coolwsd/coolwsd.xml` (`COOLWSD_CONFIGDIR = ${sysconfdir}/${PACKAGE}` = `/etc/coolwsd` when `--sysconfdir=/etc`) | Driven at container *runtime* via `--use-env-vars` + `--o:` flag overrides in `ENTRYPOINT`, not baked at image-build time | No change needed — same mechanism the existing Dockerfiles already use. |
| nginx/Apache reverse-proxy snippets | `etc/nginx/coolwsd.conf`, `etc/apache2/coolwsd.conf` — installed via `Makefile.am`'s `nginxconfig_DATA`/`apache2config_DATA` to `${sysconfdir}/nginx/snippets/` and `${sysconfdir}/apache2/...` respectively | Already correct (`36000s` timeout, WS upgrade headers, prefix-match routing) | **Zero hand-rolling needed** — reuse verbatim for the reverse-proxy smoke test (Code Example §4). |
| `entrypoint`/start | Inline `ENTRYPOINT` array in the Dockerfile itself (no separate `docker-entrypoint.sh`) | `--use-env-vars` lets `coolwsd` read config overrides from env vars at container start; self-signed cert gen and arbitrary-UID `/etc/passwd` handling done inside `coolwsd` itself | No adaptation needed; EdenDocs' `--with-app-*` branding flags are compiled into the binary at build time (Objective 1/2), not passed at container runtime. |

### Recommended: `docker/eden/Dockerfile` (new, additive file)

```
docker/
├── eden/                      # NEW — additive, zero edits to upstream docker/ tree
│   └── Dockerfile             # 3-stage: assemble -> verify-caps -> runtime
├── from-source/                # untouched upstream (Ubuntu, Debian, build.sh, README)
├── from-packages/               # untouched upstream
├── cosign.pub                   # untouched (Collabora's own key — not used by EdenDocs CI)
└── cool-seccomp-profile.json     # untouched (optional runtime --security-opt reference)
scripts/
└── eden/
    ├── build-release.sh          # NEW — non-debug configure+install variant of build.sh
    └── assemble-instdir.sh       # NEW — DESTDIR install + engine copy + branding overlay into instdir/
.github/
└── workflows/
    └── docker-publish.yml        # NEW — separate workflow: build-release -> docker build -> reverse-proxy smoke -> trivy -> push -> cosign sign+attest
```

### Pattern 1: Assemble `instdir/` for the container build context (release, non-debug)

**What:** A production configure/install pass distinct from `scripts/eden/build.sh`'s CI/debug pass — no `--enable-debug`, standard FHS prefixes, `DESTDIR`-staged install, engine copied alongside.
**When to use:** Only for the container-publish workflow; `scripts/eden/build.sh` remains unchanged for the fast dev/CI loop.
**Why non-debug matters:** `configure.ac`'s own `AC_DEFINE([ENABLE_DEBUG],...)` comment states debug mode "disable[s] some security pieces." `scripts/eden/build.sh` intentionally uses `--enable-debug` to arm the `--with-lo-path` hard-existence-check at configure time (a correctness assertion for CI) — appropriate for verifying the build works, wrong for what ships to customers.
**Key semantic difference:** in debug mode `--with-lo-path` must be a real, already-existing path at configure time; in production/non-debug mode it is just the future runtime path string baked into the binary and does not need to exist yet (`configure.ac:1738-1786`). This lets the release script point `--with-lo-path=/opt/collaboraoffice` (the path the *container* will use) even though that path doesn't exist on the build host.

```bash
# Source: derived from docker/from-source/build.sh's own configure invocation,
# adapted to EdenDocs' existing engine-assets fast path (scripts/eden/fetch-engine-assets.sh)
# and branding flags (scripts/eden/build.sh).
./autogen.sh
./configure --prefix=/usr --sysconfdir=/etc --localstatedir=/var \
  --enable-silent-rules --disable-tests \
  --with-lokit-path="$PWD/engine/include" \
  --with-lo-path=/opt/collaboraoffice \
  --with-app-branding="$PWD/eden-branding" \
  --with-app-name="EdenDocs" \
  --with-vendor="AO Cyber Systems" \
  --with-help-url="https://aocyber.ai" \
  LDFLAGS="-L/usr/local/lib"
  # HARD OMIT LIST unchanged: never pass --with-welcome-url, --with-feedback-url,
  # --with-infobar-url, --with-support-public-key (BRAND-06)

make -j"$(nproc)" build-nocheck
make -j"$(nproc)" -C browser

INSTDIR="$PWD/docker/eden/instdir"
rm -rf "$INSTDIR"
DESTDIR="$INSTDIR" make install
mkdir -p "$INSTDIR/opt"
cp -a engine/instdir "$INSTDIR/opt/collaboraoffice"

# BRAND-04 overlays, same as scripts/eden/build.sh but into the DESTDIR tree:
cp eden-branding/favicon.ico "$INSTDIR/usr/share/coolwsd/favicon.ico" 2>/dev/null || true
```

### Pattern 2: `verify-caps`-equivalent stage ported onto from-source content

**What:** Split the single-stage from-source recipe into `assemble` (COPY instdir + setcap) → `verify-caps` (COPY --from=assemble the two capability-bearing binaries, assert with `getcap`) → final runtime stage (`COPY --from=assemble` everything + `COPY --from=verify-caps` the proof marker).
**When to use:** This is the DIST-02-satisfying design. Even though single-stage from-source has no *inherent* xattr-loss risk (setcap runs directly on the final filesystem, not copied in from elsewhere), restructuring into stages to literally exercise `COPY --from` and prove capability survival gives a strictly stronger, CI-enforced guarantee — and directly satisfies the stated success criterion's wording rather than arguing it doesn't apply.

```dockerfile
# Source: pattern ported from docker/from-packages/Dockerfile lines 154-171
# (verify-caps stage) applied to docker/from-source/Ubuntu's content (lines 1-56).
# syntax=docker/dockerfile:1.4
ARG base_image="ubuntu:24.04"

FROM ${base_image} AS assemble
RUN apt-get update && \
    apt-get -y install libpng16-16 fontconfig adduser cpio tzdata \
      findutils libcap2-bin openssl openssh-client \
      libxcb-shm0 libxcb-render0 libxrender1 libxext6 \
      fonts-wqy-zenhei fonts-wqy-microhei fonts-droid-fallback \
      fonts-noto-cjk ca-certificates && \
    rm -rf /var/lib/apt/lists/*
COPY instdir /
RUN setcap cap_fowner,cap_chown,cap_sys_chroot=ep /usr/bin/coolforkit-caps && \
    setcap cap_sys_admin=ep /usr/bin/coolmount && \
    adduser --quiet --system --group --home /opt/cool cool && \
    rm -rf /opt/cool && \
    mkdir -p /opt/cool/child-roots /opt/cool/cache && \
    coolwsd-systemplate-setup /opt/cool/systemplate /opt/collaboraoffice >/dev/null 2>&1 && \
    touch /var/log/coolwsd.log && chown cool:cool /var/log/coolwsd.log && \
    chown -R cool:cool /opt/ /etc/coolwsd && chmod g+w /etc/passwd

# --- verify-caps: exercise COPY --from and assert capabilities survived it ---
FROM assemble AS verify-caps
RUN apt-get update && apt-get -y install libcap2-bin
COPY --from=assemble /usr/bin/coolforkit-caps /usr/bin/coolmount /probe/
RUN set -eu; \
    fc="$(getcap /probe/coolforkit-caps)"; mc="$(getcap /probe/coolmount)"; \
    echo "post-COPY caps -- coolforkit-caps: ${fc:-<none>} | coolmount: ${mc:-<none>}"; \
    case "$fc" in *cap_sys_chroot*) ;; *) echo "FATAL: cap_sys_chroot lost on COPY --from"; exit 1;; esac; \
    case "$mc" in *cap_sys_admin*) ;; *) echo "FATAL: cap_sys_admin lost on COPY --from"; exit 1;; esac; \
    touch /caps-verified

FROM ${base_image}
COPY --from=assemble / /
COPY --from=verify-caps /caps-verified /opt/cool/.caps-verified
EXPOSE 9980
USER 100
HEALTHCHECK --interval=30s --timeout=10s --start-period=120s --retries=3 \
    CMD ["/usr/bin/coolwsd", "--probe"]
ENTRYPOINT ["/usr/bin/coolwsd", \
            "--use-env-vars", \
            "--o:sys_template_path=/opt/cool/systemplate", \
            "--o:child_root_path=/opt/cool/child-roots", \
            "--o:file_server_root_path=/usr/share/coolwsd", \
            "--o:cache_files.path=/opt/cool/cache", \
            "--o:logging.color=false", \
            "--o:stop_on_config_change=true"]
```
(LABELs deliberately omitted from this file — applied at CI level, see Code Example §6, so the Dockerfile carries no hardcoded version/commit strings and needs no edits on every release.)

### Pattern 3: Reverse-proxy >60s WebSocket session test

**What:** Prove a document session survives longer than 60s through a *real* reverse proxy — not the direct `127.0.0.1:9980` access `scripts/eden/smoke-test.sh` already does.
**Why 60s matters:** nginx's own `proxy_read_timeout` **default is 60s**. Any proxy config that omits an explicit override will silently kill idle WebSocket connections at 60s — this is exactly the failure mode DIST-02 is testing for. The project's own `.planning/research/PITFALLS.md` Pitfall 10 already documents this class of failure (WS upgrade headers, regex-vs-prefix routing, 60s default timeout) — this test operationalizes that pitfall into a mechanical CI check.
**Design:**
1. Start the EdenDocs container (built above) on an internal Docker network, native port 9980, no host port published.
2. Start a throwaway `nginx:alpine` container on the same network, mounting `etc/nginx/coolwsd.conf` (verbatim, unmodified — already has `proxy_read_timeout 36000s;` and `proxy_http_version 1.1; proxy_set_header Upgrade $http_upgrade; proxy_set_header Connection "Upgrade";` on a `location ^~ /cool/` prefix match) into `/etc/nginx/conf.d/` alongside a minimal `server { listen 8091; ... include coolwsd.conf; }` wrapper (port 8091 for the local check per project's hard port rule — **never 8080**).
3. From the test runner, open a document session through the nginx container's `:8091` (not the coolwsd container's `:9980` directly), hold the WebSocket open with periodic no-op traffic suppressed (true idle), and assert the connection is still alive at `>60s` wall-clock (e.g., 75-90s to give margin above nginx's *default* 60s, proving the `36000s` override is actually in effect and not silently ignored).
4. A **negative control** is valuable and cheap: run the same test against an nginx config that deliberately omits the `proxy_read_timeout` override (relying on the 60s default) and assert the session **does** die around 60s — this proves the test harness itself can detect the failure mode it exists to catch, not just that the "happy path" config works.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|--------------|-----|
| nginx reverse-proxy config for the WS smoke test | A bespoke nginx.conf snippet | `etc/nginx/coolwsd.conf` (already in-tree, `make install`-shipped via `Makefile.am`'s `nginxconfig_DATA`) | Already has the correct `36000s` timeout, `Upgrade`/`Connection` headers, and prefix-match (`^~`) routing for the compact `/cool/ws?WOPISrc=` URL pattern — upstream already solved the exact gotchas Pitfall 10 warns about. |
| OCI label / annotation generation | Hand-written `LABEL` lines with manually maintained version/commit strings | `docker/metadata-action@v5` output piped into `build-push-action`'s `labels:` input | Keeps the Dockerfile static (no edits needed per release) and produces spec-correct `org.opencontainers.image.*` keys automatically from git ref/sha/tags. |
| Cosign signing infrastructure | A generated keypair + secret storage/rotation process | Keyless Sigstore signing via `sigstore/cosign-installer` + GitHub OIDC (`id-token: write`) | Org-uniform across every AO-Cyber-Systems release workflow checked; zero secret to leak, zero rotation burden, verifiable via `cosign verify --certificate-oidc-issuer=https://token.actions.githubusercontent.com`. |
| SBOM generation | A custom dependency-scanning script | Syft (`anchore/sbom-action/download-syft@v0`) + `cosign attest --type cyclonedx` | Standard org pattern (aodex-go, aocore); produces a signed, attached SBOM attestation with no bespoke tooling. |
| Capability-preservation verification | A manual `docker inspect`/`getcap` check run ad hoc after build | The `verify-caps` build stage itself (fails the build, not just a post-hoc warning) | This is literally what upstream's own `from-packages/Dockerfile` already does — DIST-02 asks for the same guarantee applied to from-source. |

**Key insight:** Every piece of DIST-01/02/03 has an existing, in-tree or org-wide precedent. The objective's actual work is *composition* (a new Dockerfile that recombines two existing upstream patterns) and *CI wiring* (a new workflow that mirrors the org's already-proven aodex-go release recipe) — not new invention.

## Common Pitfalls

### Pitfall 1: `verify-caps` does not exist in `docker/from-source/{Ubuntu,Debian}` today
**What goes wrong:** A plan that assumes "the verify-caps stage" already exists in the from-source path (as DIST-02's wording implies) will find nothing to preserve — it must be built new.
**Why it happens:** `verify-caps` is a `docker/from-packages/Dockerfile`-only construct, added specifically because that Dockerfile's multi-stage `COPY --from=builder` design has genuine BuildKit-xattr-loss exposure. from-source's single-stage design (`setcap` runs directly on the final image, no cross-stage copy of the capability-bearing binaries) does not share that exposure.
**How to avoid:** Add the stage explicitly in the new `docker/eden/Dockerfile` (Code Example/Pattern 2 above) rather than assuming it is inherited. Flag this explicitly to the planner as scope, not a preservation task.
**Warning signs:** A task titled "preserve verify-caps in the from-source build" with no corresponding "create" step will silently produce a no-op.

### Pitfall 2: The rolling `engine-main-assets.tar.gz` tarball is not reproducible — do not build release images against the live fetch
**What goes wrong:** `scripts/eden/fetch-engine-assets.sh` deliberately does not pin/verify a checksum (by design, for the fast-moving dev/CI loop) and always fetches the current `for-code-assets` release-tag asset. Direct verification during this research (2026-07-09) found the asset has been republished **three times in ~36 hours**: a known-good build (`ea91ef33ba510eb1379568c58afe9ddc54aa05985f4ff963ab8e0a8cdfa6c59f`), a known-bad build that SIGSEGVs kit startup on `forkCOKit`/`lokit_main` (`076a41297c2ed659638617ed27cb6d4cff01f1558f36fb0e0b1abd6f4a14868e`, still failing as of CI run 28991414597 at 2026-07-09T03:34:55Z), and a brand-new, not-yet-validated third version (`baba4367ec744f51fa95177e8237acd5ced8611bba2bc3015f0eafa5ee373666`) discovered live during this research.
**Why it happens:** Upstream's `for-code-assets` tag is an explicitly rolling, unversioned release — there is no immutable per-commit engine artifact published upstream today.
**How to avoid:** For the *release/publish* workflow specifically (not the existing dev/CI `build.yml`), pin to a **known-good** engine snapshot rather than the live tag: either (a) cache/mirror a verified-good `engine-main-assets.tar.gz` copy into an AO-org-controlled location (e.g., an AO-Cyber-Systems release asset or GHCR OCI artifact) and reference that by fixed URL + checksum, or (b) at minimum, add a post-fetch smoke boot (start `coolwsd`, confirm the kit process does not SIGSEGV) as a hard gate *before* the fetched engine is used for a container build that will be published — never let a SIGSEGVing engine reach a signed, published image. See Open Questions #1 for the exact mechanism to decide.
**Warning signs:** A published image whose kit process crashes immediately on first document open — this is exactly what run 28964794754 hit as a CI smoke failure and would be far worse discovered post-publish.

### Pitfall 3: Debug-mode build flags leak "disabled security pieces" into a shipped image
**What goes wrong:** Reusing `scripts/eden/build.sh` verbatim (with `--enable-debug`) for the container release build ships a binary `configure.ac` itself documents as having "some security pieces" disabled.
**Why it happens:** `--enable-debug` was added to `build.sh` specifically to arm a *build-time correctness assertion* (the `--with-lo-path` hard-existence check) for the CI dev loop — a legitimate reason for CI, but an accidental side effect if the same script is reused for what ships to customers.
**How to avoid:** Use a distinct release-build script/invocation (Pattern 1 above) with no `--enable-debug`, standard FHS `--prefix=/usr --sysconfdir=/etc --localstatedir=/var`, and `--with-lo-path=/opt/collaboraoffice` (a future runtime path string, valid in non-debug mode without existing at configure time).
**Warning signs:** Grep the release workflow for `--enable-debug` — its presence anywhere in the container-image build path is the signal to fix.

### Pitfall 4: `--with-lo-path` semantics differ between debug and non-debug configure
**What goes wrong:** Copy-pasting `scripts/eden/build.sh`'s `--with-lo-path="$PWD/engine/instdir"` (a real, existing dev-tree path, required because `--enable-debug` arms a hard existence check) into a non-debug release script produces a binary baked with a build-host-specific path rather than the container's actual runtime path (`/opt/collaboraoffice`).
**Why it happens:** `configure.ac:1738-1786` treats `--with-lo-path` as a hard-checked real path only in debug/developer mode; in production/non-debug mode it is just a compiled-in string for the future runtime location.
**How to avoid:** In the release script, explicitly set `--with-lo-path=/opt/collaboraoffice` to match where the Dockerfile's `docker/eden/Dockerfile` `COPY instdir /` will actually place the engine (`instdir/opt/collaboraoffice/...`), not the build host's working-tree path.
**Warning signs:** `coolwsd` inside the container fails to find LOKit at runtime, or logs point at a build-host path like `/home/runner/work/...` instead of `/opt/collaboraoffice`.

### Pitfall 5: nginx's 60s default `proxy_read_timeout` silently masks a missing override
**What goes wrong:** A reverse-proxy config that forwards WS upgrade headers correctly but omits an explicit `proxy_read_timeout` override will still pass a *short* smoke test (anything under 60s "works") while silently failing DIST-02's actual >60s bar.
**Why it happens:** nginx's documented default for `proxy_read_timeout` is 60s; Collabora's own reference config (`etc/nginx/coolwsd.conf`) explicitly overrides it to `36000s` on the `/cool/` block specifically because idle WS sessions during real editing (user reading, not typing) routinely exceed a minute.
**How to avoid:** Design the smoke test to run meaningfully past 60s (Pattern 3 above recommends 75-90s) and include the negative-control run (default-timeout config, expected to fail at ~60s) so the test provably discriminates the two cases rather than just checking "did it work at all."
**Warning signs:** A reverse-proxy smoke test that completes in under 60s wall-clock cannot actually prove this criterion regardless of its pass/fail result.

### Pitfall 6: nginx `location` regex vs prefix-match routing drops the compact WOPI WS URL
**What goes wrong:** Collabora's compact WS URL pattern (`/cool/<encoded-wopisrc>/ws?...`) needs a **prefix match** (`location ^~ /cool/`); a narrower regex-style location block placed earlier in nginx's matching order (nginx evaluates `location ^~` prefix matches with highest priority among prefix matches, but a `location ~` regex block elsewhere can still intercept unexpected paths if the config is hand-written incorrectly) can silently 404 or misroute the session.
**Why it happens:** This is exactly the failure mode documented in the project's own `.planning/research/PITFALLS.md` Pitfall 10.
**How to avoid:** Reuse `etc/nginx/coolwsd.conf` verbatim (already uses `location ^~ /cool/`) rather than hand-writing a new proxy config for the smoke test container.
**Warning signs:** WS upgrade succeeds but document content never loads, or requests to `/cool/...` 404 despite coolwsd being reachable directly.

### Pitfall 7: GHCR image/org names must be lowercase
**What goes wrong:** Pushing to `ghcr.io/AO-Cyber-Systems/edendocs` (mixed case, matching the GitHub org's actual display casing) fails — GHCR (like all OCI registries) requires lowercase repository paths.
**Why it happens:** OCI distribution spec repository name grammar is lowercase-only; this has bitten other AO-Cyber-Systems repos before (confirmed convention in `eden-biz`'s release workflow: `IMAGE_OWNER: ao-cyber-systems`, all lowercase).
**How to avoid:** Use `ghcr.io/ao-cyber-systems/edendocs` (lowercase) as the published image path, matching the org's own established convention.
**Warning signs:** `docker push` / `build-push-action` fails with a "repository name must be lowercase" error.

### Pitfall 8: BuildKit GHA cache (`type=gha`) does not export `RUN --mount=type=cache` ccache mounts
**What goes wrong:** If a future change moves compilation *inside* the Dockerfile (not the current from-source design, but worth flagging since `docker/from-packages` and hypothetical future variants could tempt this), a `RUN --mount=type=cache,target=/ccache` mount will NOT be persisted by `cache-from/cache-to: type=gha` alone — that only caches Dockerfile *layers*, not BuildKit cache-mount contents.
**Why it happens:** GHA cache backend and BuildKit's cache-mount feature are two separate caching mechanisms; `type=gha` does not automatically bridge them.
**How to avoid:** Not applicable to the recommended design (compilation stays on the host runner, as today) — but if this changes, use `reproducible-containers/buildkit-cache-dance` or a persistent self-hosted builder instead of assuming `type=gha` alone covers it.
**Warning signs:** ccache hit rate inside a Docker build stays near 0% across CI runs despite `type=gha` caching being configured.

### Pitfall 9: The port-8080 ban applies to every layer of this objective, including throwaway test containers
**What goes wrong:** It is easy to default an nginx test container or a `docker-compose` port mapping to `8080` out of habit (it's nginx's/most reverse-proxy tutorials' conventional example port).
**Why it happens:** `8080` is the de facto default in almost every public nginx/reverse-proxy tutorial and example.
**How to avoid:** Any local-facing port in this objective's Dockerfiles, compose files, smoke-test scripts, or docs must be `8091` (matching the rest of this repo's convention — `wopi-host` on 8091, `fake-aoid` on 8092) — never `8080`. coolwsd's own native port (`9980`) is unaffected and unchanged.
**Warning signs:** The repo's existing mechanical grep gate for `:8080` (referenced in the task constraints) will catch this at CI time, but catching it in review is cheaper.

## Code Examples

### 1. Release build script skeleton (new file: `scripts/eden/build-release.sh`)
See Pattern 1 above for the full configure/install sequence.

### 2. `docker/eden/Dockerfile` (new file)
See Pattern 2 above for the complete 3-stage Dockerfile.

### 3. Assembling the docker build context
```bash
# Source: derived from docker/from-source/build.sh's own COPY convention
# ("COPY /instdir /" expects a context root containing an instdir/ directory).
mkdir -p docker/eden/instdir
DESTDIR="$PWD/docker/eden/instdir" make install
cp -a engine/instdir docker/eden/instdir/opt/collaboraoffice
docker buildx build -f docker/eden/Dockerfile docker/eden/ \
  --build-arg base_image=ubuntu:24.04 \
  -t ghcr.io/ao-cyber-systems/edendocs:scan --load
```

### 4. Reverse-proxy config for the smoke test (reused verbatim, zero edits)
```
# Source: /Users/justin/dev/EdenDocs/etc/nginx/coolwsd.conf (already in-tree,
# installed via Makefile.am's nginxconfig_DATA — do not hand-write a new one)
location ^~ /browser { proxy_pass http://localhost:9980; proxy_set_header Host $host; }
location ^~ /hosting/discovery { proxy_pass http://localhost:9980; proxy_set_header Host $host; }
location ^~ /hosting/capabilities { proxy_pass http://localhost:9980; proxy_set_header Host $host; }
location ^~ /cool/ {
    proxy_pass http://localhost:9980;
    proxy_http_version 1.1;
    proxy_set_header Upgrade $http_upgrade;
    proxy_set_header Connection "Upgrade";
    proxy_set_header Host $host;
    proxy_read_timeout 36000s;
}
location ^~ /lool/ { proxy_pass http://localhost:9980; proxy_set_header Host $host; }
```

### 5. Reverse-proxy smoke test harness sketch
```bash
# Port 8091 only — 8080 is banned project-wide.
docker network create edendocs-proxy-test
docker run -d --name edendocs --network edendocs-proxy-test \
  ghcr.io/ao-cyber-systems/edendocs:scan
docker run -d --name proxy --network edendocs-proxy-test -p 8091:8091 \
  -v "$PWD/etc/nginx/coolwsd.conf:/etc/nginx/conf.d/coolwsd.conf:ro" \
  -v "$PWD/scripts/eden/nginx-test-server.conf:/etc/nginx/conf.d/server.conf:ro" \
  nginx:alpine
# ... open a document session against http://127.0.0.1:8091/browser/dist/cool.html,
# hold the WS session idle, assert alive at t=75-90s.
```

### 6. LABEL/annotation injection at CI level (no Dockerfile edits per release)
```yaml
# Source: pattern confirmed identical across eden-biz, aodex-go, aocore release workflows
- uses: docker/metadata-action@v5
  id: meta
  with:
    images: ghcr.io/ao-cyber-systems/edendocs
    labels: |
      org.opencontainers.image.vendor=AO Cyber Systems
      org.opencontainers.image.title=EdenDocs
      org.opencontainers.image.description=Privacy-first self-hosted collaborative document editor
- uses: docker/build-push-action@v6
  with:
    context: docker/eden
    file: docker/eden/Dockerfile
    push: true
    tags: ${{ steps.meta.outputs.tags }}
    labels: ${{ steps.meta.outputs.labels }}
    provenance: mode=max
    sbom: true
```

### 7. Keyless cosign sign + attest (mirrors `aodex/.github/workflows/release-go.yml` exactly)
```yaml
# Source: /Users/justin/dev/aodex/.github/workflows/release-go.yml — confirmed
# identical (keyless) pattern also in release-flutter.yml, aocore, aocore-admin,
# aoinference. No AO-Cyber-Systems repo uses key-based cosign.
permissions:
  contents: read
  packages: write
  id-token: write
  attestations: write
steps:
  - uses: sigstore/cosign-installer@v3
  - name: Sign image
    run: cosign sign --yes "ghcr.io/ao-cyber-systems/edendocs@${DIGEST}"
  - uses: anchore/sbom-action/download-syft@v0
  - run: syft "ghcr.io/ao-cyber-systems/edendocs@${DIGEST}" -o cyclonedx-json > sbom.cdx.json
  - run: cosign attest --yes --predicate sbom.cdx.json --type cyclonedx "ghcr.io/ao-cyber-systems/edendocs@${DIGEST}"
  - name: Verify (summary)
    run: |
      cosign verify "ghcr.io/ao-cyber-systems/edendocs@${DIGEST}" \
        --certificate-identity-regexp="https://github.com/AO-Cyber-Systems/EdenDocs/.*" \
        --certificate-oidc-issuer="https://token.actions.githubusercontent.com"
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|-------------------|---------------|--------|
| `.planning/research/STACK.md`'s prior guidance: "EdenDocs should generate its own cosign keypair" | Keyless cosign via Sigstore/GitHub Actions OIDC | Superseded by this research (2026-07-09) — confirmed org-wide convention across 6 checked repos with zero key-based exceptions | **Correction to prior project research.** The planner should follow this RESEARCH.md's recommendation, not STACK.md's, for the signing mechanism. |
| Assumption that `docker/from-source` already has a `verify-caps` stage (implied by DIST-02's phrasing) | Confirmed: `verify-caps` exists only in `docker/from-packages/Dockerfile`; from-source has none | Verified by direct file read during this research | The planner must scope "add verify-caps to a new from-source-pattern Dockerfile" as new work, not "preserve existing" work. |
| `engine-main-assets.tar.gz` treated as a stable, always-fetchable dependency (Objective 1's framing: "pinning would fossilize the build") | Confirmed empirically volatile — 3 republishes in ~36h, one SIGSEGV-causing — a live risk for anything meant to be *published and signed* | Discovered via live `gh api`/direct-download verification during this research (2026-07-09); root cause (rolling, unversioned upstream tag) is a known, longstanding upstream characteristic, not new | Objective 1's "don't pin" rationale is correct for the fast-moving dev/CI loop, but the *release/publish* path (this objective) has a different reproducibility bar and should pin/cache a known-good snapshot — see Open Questions #1. |

**Deprecated/outdated:** None identified — the from-source/from-packages Docker patterns, `coolwsd.xml.in` config-generation mechanism, and `--use-env-vars`/`--o:` runtime override pattern are all current upstream mechanisms with no announced replacement.

## Open Questions

1. **How should the release workflow pin/cache a known-good engine snapshot?**
   - What we know: The live rolling tarball is volatile (3 versions in 36h, one broken); `scripts/eden/fetch-engine-assets.sh` deliberately does not pin, by design, for the dev/CI loop. A third, brand-new tarball version (`baba4367...`) was observed live during this research but its own boot-health was not validated within research scope (an in-archive path lookup for its version metadata did not resolve on the first attempt and was not retried).
   - What's unclear: Whether AO-Cyber-Systems wants to (a) mirror a verified-good tarball into an org-controlled location (e.g., a GHCR OCI artifact or release asset) and pin the release workflow's fetch to that fixed URL+checksum, or (b) keep fetching the live rolling tag for releases too but add a hard pre-publish boot-health gate (start coolwsd, confirm no SIGSEGV, abort publish if it crashes) as the safety net instead of pinning.
   - Recommendation: Favor (a) for reproducibility (a published, signed image should be rebuildable from a fixed input set) with (b) as a defense-in-depth gate regardless. This should be a planner-level decision with an explicit success criterion (e.g., "the pinned engine snapshot's sha256 is recorded in the release workflow / a CHECKSUMS file").

2. **Does DIST-02's "verify-caps stage still passes" require the literal stage-splitting design (Pattern 2), or would a post-build `getcap` assertion outside the Dockerfile satisfy the intent?**
   - What we know: The from-source single-stage design has no actual `COPY --from` xattr-loss exposure to guard against (setcap runs directly on the final filesystem). A stage-split design (Pattern 2) manufactures the same risk surface as `from-packages` purely to prove resilience to it, and literally satisfies the wording.
   - What's unclear: Whether the objective owner intended DIST-02 to mean "the specific from-packages mechanism, ported" or more loosely "prove capabilities are correctly set in the shipped image, by whatever mechanism."
   - Recommendation: Proceed with Pattern 2 (the literal interpretation) since it costs little extra complexity and unambiguously satisfies the stated criterion; flag this open question to the planner so it can confirm or override before task-level work begins.

3. **Should the new Dockerfile support both Ubuntu and Debian bases for v1, or Ubuntu-only?**
   - What we know: DIST-01's phrasing is `docker/from-source/{Ubuntu,Debian}` (brace notation suggesting "the pattern used by either/both"), not a hard requirement for both to ship. Ubuntu 24.04 matches the `ubuntu-latest` GHA runner's own userland most closely.
   - What's unclear: Whether v1 needs both published, or whether Ubuntu-only satisfies the requirement with Debian deferred.
   - Recommendation: Ship Ubuntu 24.04 only for v1 (the `ARG base_image` parameterization in Pattern 2 makes adding Debian trivial later); confirm with the objective owner if both are actually required at v1.

4. **CI job topology: one workflow or two?**
   - What we know: `build.yml` is already a 90-minute-budgeted job covering native build + smoke test + branding gate + full AUTH e2e. Adding a full second native (non-debug) build + docker build + reverse-proxy smoke + Trivy scan + cosign sign to the same job risks timeout and conflates debug/release build modes.
   - What's unclear: Exact trigger strategy for the new workflow (every push to `eden-main`, tags only, or `workflow_dispatch`) — DIST-03 only requires "published images ... available," not a specific cadence.
   - Recommendation: New `docker-publish.yml`, triggered on push to `eden-main` (matching `build.yml`'s trigger, for now, since EdenDocs has no semver tag/release cadence yet — that may be Objective 5 or a future v2 concern) plus `workflow_dispatch`, publishing `:main`, `:sha-<short>`, and `:latest` tags via `docker/metadata-action`.

## Sources

### Primary (HIGH confidence — direct file reads from this repo)
- `/Users/justin/dev/EdenDocs/docker/from-source/Ubuntu`, `docker/from-source/Debian`, `docker/from-source/README.md`, `docker/from-source/build.sh` — full content read
- `/Users/justin/dev/EdenDocs/docker/from-packages/Dockerfile`, `docker/from-packages/scripts/assemble-rootfs.sh` — full content read (verify-caps stage source)
- `/Users/justin/dev/EdenDocs/docker/README`, `docker/cosign.pub`, `docker/cool-seccomp-profile.json` — read
- `/Users/justin/dev/EdenDocs/scripts/eden/build.sh`, `fetch-engine-assets.sh`, `build-deps.sh`, `smoke-test.sh`, `verify-branding.sh` — full content read
- `/Users/justin/dev/EdenDocs/.github/workflows/build.yml` — full content read
- `/Users/justin/dev/EdenDocs/configure.ac` — `ENABLE_DEBUG`/`--enable-developer` logic (search + targeted read), `--with-lo-path` dual semantics (lines 1738-1786), `COOLWSD_CONFIGDIR`/`COOLWSD_DATADIR` definitions (lines 2336-2341)
- `/Users/justin/dev/EdenDocs/Makefile.am` — `coolwsdconfig_DATA`, `nginxconfig_DATA`, `apache2config_DATA`, `dist_bin_SCRIPTS` install rules
- `/Users/justin/dev/EdenDocs/coolwsd.xml.in` — `server_name`, `net.proxy_prefix`, `ssl.termination` config keys
- `/Users/justin/dev/EdenDocs/etc/nginx/coolwsd.conf`, `etc/apache2/coolwsd.conf` — full content read
- `/Users/justin/dev/EdenDocs/.planning/REQUIREMENTS.md`, `.planning/ROADMAP.md` — DIST-01/02/03 and Objective 4 section, verbatim
- `/Users/justin/dev/EdenDocs/.planning/research/PITFALLS.md` (Pitfall 10: reverse-proxy misconfig; Pitfall 11: version skew) — pre-existing project research, directly corroborating
- `/Users/justin/dev/EdenDocs/.planning/objectives/02-edendocs-rebrand-web-ui/02-05-SUMMARY.md` — historical engine-tarball SIGSEGV incident record (known-good/known-bad sha256, run IDs)
- `gh api repos/CollaboraOnline/online/releases/tags/for-code-assets` (live, 2026-07-09) — direct API query confirming current asset state
- Direct download + `shasum -a 256` of the current `engine-main-assets.tar.gz` (2026-07-09) — confirmed a third, previously-unseen sha256 (`baba4367ec744f51fa95177e8237acd5ced8611bba2bc3015f0eafa5ee373666`), distinct from both prior known-good/known-bad values
- `/Users/justin/dev/eden-biz/.github/workflows/release.yml` — GHCR/registry naming convention (`ghcr.io`, lowercase `ao-cyber-systems`)
- `/Users/justin/dev/aodex/.github/workflows/release-go.yml`, `release-flutter.yml` — full cosign+GHCR+SBOM+Trivy workflow pattern, read directly
- `grep -r "COSIGN_KEY\|COSIGN_PASSWORD\|cosign generate-key-pair" ~/dev` (2026-07-09) — zero matches across the org, confirming uniform keyless convention

### Secondary (MEDIUM confidence)
- nginx `proxy_read_timeout` 60s default and WebSocket proxying requirements — corroborated by both the project's own pre-existing `.planning/research/PITFALLS.md` Pitfall 10 and general nginx documentation knowledge; direct fetch of `sdk.collaboraonline.com`'s official proxy-settings page was blocked by an Anubis anti-bot challenge and could not be used as a primary source for this research pass
- GitHub-hosted `ubuntu-latest` runner specs (4 vCPU / 16 GiB RAM / ~14GB guaranteed free disk on public repos) — from general current knowledge of GitHub Actions runner specifications, not independently re-verified via a live API call in this research pass

### Tertiary (LOW confidence)
- None flagged — all findings above are grounded in either direct repo reads, live API/download verification, or cross-repo pattern confirmation.

## Metadata

**Confidence breakdown:**
- Standard stack (cosign/GHCR/Trivy/SBOM tooling): HIGH — confirmed via direct read of 3+ working org release workflows with zero variance
- Architecture (Dockerfile design, verify-caps porting, instdir assembly): HIGH — derived directly from reading both existing upstream Dockerfiles byte-for-byte
- Engine-tarball volatility/pinning need: HIGH (the volatility itself, empirically proven via live download+hash) / MEDIUM (the exact recommended pinning mechanism — left as Open Question #1 for planner decision)
- Reverse-proxy pitfalls and config: HIGH for the WS-header/prefix-routing/timeout mechanics (in-tree config + corroborating project PITFALLS.md); MEDIUM for the primary-source citation chain (official SDK docs page was unreachable this pass)
- CI resource feasibility: HIGH — compilation confirmed to happen host-side only, not inside Docker, based on direct read of both from-source Dockerfiles (no compiler invocations present)

**Research date:** 2026-07-09
**Valid until:** 7 days for the engine-tarball-pin recommendation specifically (fast-moving, actively volatile upstream asset — re-verify sha256/boot-health immediately before executing Open Question #1's decision); 30 days for all other findings (stable upstream Docker/cosign/GHCR mechanisms)
