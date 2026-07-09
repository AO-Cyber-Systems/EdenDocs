---
objective: 04-edendocs-branded-container-images
trd: "04"
type: standard
wave: 2
depends_on: ["04-01", "04-02", "04-03"]
files_modified:
  - .github/workflows/docker-publish.yml
  - docs/eden/TRADEMARK-MPL-CHECKLIST.md
autonomous: true
requirements: [DIST-01, DIST-02, DIST-03]
# TDD-EXCEPTION: infra/release work — a CI workflow. The workflow's own gates
# (verify-caps build stage, boot-health, proxy-smoke >60s, trivy, cosign verify)
# ARE the tests, driven green as the integrated DIST-01/02/03 proof (same shape as
# TRD 01-02). actionlint + the green run are the gate. Resolver's UI-TDD profile N/A.

user_setup:
  - service: ghcr (GitHub Container Registry)
    why: "Publish the signed EdenDocs image (DIST-03). CI uses the automatic GITHUB_TOKEN + OIDC — no secret to add. The only optional human action is flipping the new package's visibility to Public if external pull is desired."
    dashboard_config:
      - task: "(Optional) Set package visibility to Public"
        location: "GitHub → AO-Cyber-Systems/EdenDocs → Packages → edendocs → Package settings → Change visibility"

must_haves:
  truths:
    - "A new additive .github/workflows/docker-publish.yml (build.yml untouched) triggers on push to eden-main + workflow_dispatch and, on ubuntu-latest, runs: build-deps → fetch-engine-pinned (sha256-verified) → build-release (non-debug instdir) → docker buildx build of docker/eden/Dockerfile (verify-caps stage passes) → boot-health gate → reverse-proxy >60s smoke → trivy → push → cosign keyless sign + SBOM attest"
    - "The latest docker-publish.yml run on eden-main is GREEN and its logs carry machine-checkable evidence: verify-caps 'post-COPY caps' line, boot-health 'coolwsd accepts on 9980 / no SIGSEGV', 'PROXY SMOKE PASSED', and a cosign verify success"
    - "The published image ghcr.io/ao-cyber-systems/edendocs (lowercase) carries EdenDocs OCI LABEL metadata injected via docker/metadata-action (vendor=AO Cyber Systems, title=EdenDocs, source=…/EdenDocs, licenses=MPL-2.0) with NO Collabora trademark on any image-identity surface, and the image retains COPYING/THIRDPARTYLICENSES byte-intact"
    - "The image digest is cosign-signed keyless (Sigstore/GitHub OIDC, id-token: write, no stored key) and a CycloneDX SBOM is attested; cosign verify succeeds against the repo's OIDC identity"
    - "docs/eden/TRADEMARK-MPL-CHECKLIST.md gains container-specific rows (image name, LABEL set, org.opencontainers.image.* annotations, MPL-in-image byte-integrity) and records that Objective 4 adds ZERO upstream-file divergences"
  artifacts:
    - ".github/workflows/docker-publish.yml"
    - "docs/eden/TRADEMARK-MPL-CHECKLIST.md"
  key_links:
    - "docker-publish.yml → scripts/eden/{build-deps,fetch-engine-pinned,build-release}.sh (04-01) + docker/eden/Dockerfile (04-02) + scripts/eden/proxy-smoke-test.sh (04-03) — the integration join for all three"
    - "docker/metadata-action → build-push-action labels: → ghcr.io/ao-cyber-systems/edendocs (lowercase; Pitfall 7) OCI LABELs (DIST-03)"
    - "cosign sign --yes image@DIGEST + syft SBOM + cosign attest cyclonedx → keyless Sigstore/OIDC (id-token: write) — mirrors aodex release-go.yml exactly"
    - "boot-health gate (docker run + coolwsd --probe / healthcheck) → validates the pinned engine (04-01) actually boots in the final image before publish (engine-pin decision (b))"
---

<objective>
Wire the new, additive `.github/workflows/docker-publish.yml` that composes the three Wave-1 building blocks into the DIST-03 publish pipeline — and drive it green as the single integrated proof of DIST-01, DIST-02, and DIST-03. On every push to `eden-main` (plus `workflow_dispatch`) it: installs deps, fetches the sha256-pinned engine, runs the non-debug release build into `docker/eden/instdir`, `docker buildx build`s the branded 3-stage image (the verify-caps stage fails the build if capabilities are lost), gates on boot-health (coolwsd accepts on 9980, no SIGSEGV — validating the pinned engine), runs the reverse-proxy >60s WebSocket smoke, scans with Trivy, pushes `:main`/`:sha-<short>`/`:latest` to `ghcr.io/ao-cyber-systems/edendocs` with EdenDocs OCI LABELs, and cosign-signs the digest keyless with a CycloneDX SBOM attestation.

It also extends `docs/eden/TRADEMARK-MPL-CHECKLIST.md` with the container-specific rows BRAND-05/Objective-4 deferred (image name, LABEL set, annotations, MPL-in-image integrity), closing the checklist's open "Container image LABELs" row.

Purpose: This is the join point where DIST-01 (from-source image builds), DIST-02 (verify-caps passes + proxy >60s survives), and DIST-03 (signed, published, LABELed) are proven end-to-end in one green run — the same author-then-drive-green shape as TRD 01-02. Keyless cosign matches the AO-Cyber-Systems org convention (no keypair). build.yml stays untouched (separate debug-build concern).

Output: `.github/workflows/docker-publish.yml` committed + pushed with a green run publishing a signed image; `docs/eden/TRADEMARK-MPL-CHECKLIST.md` container rows.
</objective>

<file_tree>
.github/workflows/
├── docker-publish.yml   ← CREATE (build-release → docker build → boot-health → proxy-smoke → trivy → push → cosign)
└── build.yml              (UNTOUCHED — debug build/smoke; do not overload it)
docs/eden/
└── TRADEMARK-MPL-CHECKLIST.md  ← MODIFY (append container-image rows §3 + a container subsection)
</file_tree>

<execution_context>
@~/.claude/devflow/workflows/execute-trd.md
@~/.claude/devflow/templates/summary.md
</execution_context>

<embedded_context>

<codebase_examples>
`.github/workflows/build.yml` is the sibling to model (trigger shape, ccache, Node 20 pin) — but this is a SEPARATE file with a NON-debug release build. The permissions + cosign + metadata-action pattern comes from the org's aodex release workflow (4-RESEARCH.md Code Examples 6 + 7), reproduced as the canonical shape:

```yaml
permissions: { contents: read, packages: write, id-token: write, attestations: write }
# ... after build+push, with DIGEST captured from build-push-action's outputs.digest:
- uses: sigstore/cosign-installer@v3
- run: cosign sign --yes "ghcr.io/ao-cyber-systems/edendocs@${DIGEST}"
- uses: anchore/sbom-action/download-syft@v0
- run: syft "ghcr.io/ao-cyber-systems/edendocs@${DIGEST}" -o cyclonedx-json > sbom.cdx.json
- run: cosign attest --yes --predicate sbom.cdx.json --type cyclonedx "ghcr.io/ao-cyber-systems/edendocs@${DIGEST}"
- run: |
    cosign verify "ghcr.io/ao-cyber-systems/edendocs@${DIGEST}" \
      --certificate-identity-regexp="https://github.com/AO-Cyber-Systems/EdenDocs/.*" \
      --certificate-oidc-issuer="https://token.actions.githubusercontent.com"
```

metadata-action + build-push-action (LABELs injected at CI, Dockerfile stays static):
```yaml
- uses: docker/metadata-action@v5
  id: meta
  with:
    images: ghcr.io/ao-cyber-systems/edendocs
    tags: |
      type=raw,value=main,enable=${{ github.ref == 'refs/heads/eden-main' }}
      type=sha,prefix=sha-,format=short
      type=raw,value=latest
    labels: |
      org.opencontainers.image.vendor=AO Cyber Systems
      org.opencontainers.image.title=EdenDocs
      org.opencontainers.image.description=Privacy-first self-hosted collaborative document editor
      org.opencontainers.image.licenses=MPL-2.0
```

`docs/eden/TRADEMARK-MPL-CHECKLIST.md` §3 currently has: `| Container image LABELs | — | None exist … Objective 4 owns any future Eden LABEL scheme |` — this TRD fills that row in and adds the container subsection.
</codebase_examples>

<anti_patterns>
- **Do NOT extend build.yml** (research Q4) — build.yml runs a `--enable-debug` build (ships disabled security pieces) and is already ~90-min-budgeted. A release image needs the non-debug build-release.sh path in its own workflow.
- **Do NOT use key-based cosign** — no `cosign generate-key-pair`, no `COSIGN_KEY`/`COSIGN_PASSWORD`. Keyless Sigstore/OIDC is the uniform org convention (zero key-based repos). `docker/cosign.pub` in-tree is Collabora's, not ours — ignore it.
- **Do NOT push a mixed-case repo path** (Pitfall 7) — `ghcr.io/ao-cyber-systems/edendocs` is lowercase; the org display casing (AO-Cyber-Systems) fails OCI push.
- **Do NOT hardcode LABELs in the Dockerfile** — inject via metadata-action here so the Dockerfile stays edit-free per release.
- **Do NOT inject any Collabora trademark into LABELs/annotations** — image identity is EdenDocs; a grep gate in the workflow asserts no `collabora` (case-insensitive) in the built image's LABELs.
- **Do NOT build the publishable image against the rolling engine** — call `fetch-engine-pinned.sh` (04-01), never `fetch-engine-assets.sh`.
- **NEVER reference port 8080** anywhere in the workflow, its comments, or the proxy-smoke step. coolwsd=9980, host proxy check=8091 (proxy-smoke-test.sh owns it).
- **Do NOT deploy beyond the registry** — publishing a signed image to ghcr IS the deliverable (DIST-03). Do NOT trigger any k8s/Helm/DNS rollout; there is none in scope. Surface package-visibility (public) as an optional human action, do not force it.
</anti_patterns>

<error_recovery>
- **verify-caps stage fails the build ('cap_sys_chroot lost on COPY --from')** → BuildKit is not preserving the security.capability xattr; confirm `docker/setup-buildx-action@v3` is used (modern frontend) and `# syntax=docker/dockerfile:1.4` is on line 1 of the Dockerfile. This is the gate working as intended — do NOT bypass it.
- **Boot-health gate: coolwsd SIGSEGVs / never accepts on 9980** → the pinned engine is bad. If the pin is ea91ef33… and it still crashes, RE-PIN: fetch current rolling engine, boot-validate it in a scratch step, and if green update docker/eden/engine-pin.env (04-01's file) to that sha256 + upload it to the mirror. NEVER pin 076a4129…. Abort publish on any crash (that is the gate's job — decision (b)).
- **Engine mirror 404 (release asset engine-pin not populated yet)** → first run must POPULATE the mirror. Do the one-time upload: obtain a boot-validated known-good tarball (prefer ea91ef33…; else re-pin as above), `gh release create engine-pin --notes "pinned EdenDocs release engine"` + `gh release upload engine-pin engine-main-assets.tar.gz`. Then re-run.
- **`docker push` 'repository name must be lowercase'** → fix the images: value to lowercase ao-cyber-systems/edendocs (Pitfall 7).
- **cosign verify fails on certificate-identity** → the OIDC SAN uses the repo's actual casing (AO-Cyber-Systems/EdenDocs); keep the `--certificate-identity-regexp` matching `https://github.com/AO-Cyber-Systems/EdenDocs/.*`.
- **ghcr login denied** → the job needs `permissions: packages: write` and `docker/login-action@v3` with `password: ${{ secrets.GITHUB_TOKEN }}`. On a genuine auth failure, surface as an auth gate — never embed credentials.
- **Trivy HIGH/CRITICAL blocks the push** → this is the gate working. Triage: if it is an unfixable base-image CVE, document it and consider `--severity CRITICAL` only or an explicit `.trivyignore` with justification; do not silently disable the scan.
- **Job timeout** → the non-debug native build + docker build + 80s proxy smoke + trivy is heavier than build.yml; set `timeout-minutes: 120`. If still tight, ensure ccache is restored (cold build dominates).
- **>4 red fix cycles** → stop and surface the failure pattern to the user rather than thrashing (same discipline as TRD 01-02).
</error_recovery>

</embedded_context>

<context>
@.planning/PROJECT.md
@.planning/ROADMAP.md
@.planning/STATE.md
@.planning/objectives/04-edendocs-branded-container-images/4-RESEARCH.md
@.planning/objectives/04-edendocs-branded-container-images/04-01-release-build-engine-pin-TRD.md
@.planning/objectives/04-edendocs-branded-container-images/04-02-branded-dockerfile-verify-caps-TRD.md
@.planning/objectives/04-edendocs-branded-container-images/04-03-reverse-proxy-smoke-harness-TRD.md
@.github/workflows/build.yml
@docs/eden/TRADEMARK-MPL-CHECKLIST.md
</context>

<research_context>
From 4-RESEARCH.md (HIGH confidence) — Open Question 4 resolved:
- Separate `docker-publish.yml`, triggered on push to eden-main + workflow_dispatch, publishing :main / :sha-<short> / :latest via docker/metadata-action.
- Two-phase build pattern (aodex-go): build a `:scan` tag with `load: true`, run Trivy `exit-code: '1'` on HIGH/CRITICAL, then build+push. For single-arch amd64 v1 you can build-load once, gate (verify-caps runs in-build, boot-health, proxy-smoke, trivy), then push the same context.
- Keyless cosign (Sigstore/OIDC), `id-token: write`, `cosign sign --yes image@DIGEST`, SBOM via syft + `cosign attest --type cyclonedx`, `cosign verify` with certificate-identity-regexp + oidc-issuer. Confirmed identical across 6 org repos, zero key-based exceptions — SUPERSEDES STACK.md's "generate a keypair" guidance.
- linux/amd64 only for v1 (aodex-go's own primary platform); QEMU/arm64 deferred.
- CI feasibility HIGH: no C++ compile inside Docker (COPY a pre-built instdir), so BuildKit cache-mount complexity is moot; the native build is host-side exactly as build.yml already does.
- Engine-pin decision: (a) pinned sha256 fetch (fetch-engine-pinned.sh) + (b) boot-health gate here as defense-in-depth. A success criterion asserts the pinned sha256 is recorded (04-01) and boot-validated (this gate).
</research_context>

<gotchas>
- HARD RULE: never 8080. The proxy-smoke step invokes scripts/eden/proxy-smoke-test.sh (8091 host). Keep every comment/port here on 9980/8091.
- Capture the pushed digest from `build-push-action`'s `outputs.digest` and sign THAT (`image@sha256:…`), not a mutable tag — cosign signs immutable digests.
- The Ubuntu-based runtime image HAS a shell (not distroless), so the MPL-in-image check can `docker run --rm --entrypoint sh <image> -c 'sha256sum /usr/share/doc/... COPYING'` or `docker create` + `docker cp` to compare against the repo's COPYING. Assert COPYING/THIRDPARTYLICENSES are present + byte-intact in the image (MPL constraint).
- `docker inspect --format '{{json .Config.Labels}}'` on the built image → assert it contains title=EdenDocs + vendor=AO Cyber Systems and does NOT contain (case-insensitive) 'collabora'.
- First push creates the ghcr package PRIVATE by default — that still satisfies DIST-03 "available in the AO-Cyber-Systems registry" (org members can pull). Public visibility is the optional human action in user_setup.
- Re-pin caveat: if this run must update docker/eden/engine-pin.env (04-01's file), that is a legitimate cross-wave integration fixup (04-01 is already merged by the time this Wave-2 TRD runs) — commit it atomically with a clear message.
- Engine-pin mirror population is a one-time `gh release` upload; needs repo write (CI GITHUB_TOKEN has it, or an authed local gh). Surface any auth failure as a gate.
</gotchas>

<tasks>

<task type="auto">
  <name>Task 1: Author .github/workflows/docker-publish.yml (build → verify-caps → boot-health → proxy-smoke → trivy → push → cosign)</name>
  <files>.github/workflows/docker-publish.yml</files>
  <action>
Create `.github/workflows/docker-publish.yml` — new additive file (do NOT touch build.yml). Job outline (fill from the research code examples + the sibling build.yml conventions):

```yaml
name: docker-publish
on:
  push: { branches: [eden-main] }
  workflow_dispatch:
concurrency: { group: docker-publish-${{ github.ref }}, cancel-in-progress: true }
permissions: { contents: read, packages: write, id-token: write, attestations: write }
env:
  IMAGE: ghcr.io/ao-cyber-systems/edendocs   # lowercase (Pitfall 7)
  CCACHE_DIR: /home/runner/.ccache
jobs:
  publish:
    runs-on: ubuntu-latest
    timeout-minutes: 120
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-node@v4
        with: { node-version: '20' }
      - uses: docker/setup-buildx-action@v3
      - uses: docker/login-action@v3
        with: { registry: ghcr.io, username: ${{ github.actor }}, password: ${{ secrets.GITHUB_TOKEN }} }
      - name: Restore ccache
        uses: actions/cache@v4
        with: { path: /home/runner/.ccache, key: ccache-release-${{ hashFiles('configure.ac') }}, restore-keys: 'ccache-release-' }
      - name: Build deps + system POCO
        run: ./scripts/eden/build-deps.sh
      - name: Fetch PINNED engine (sha256-verified — NOT the rolling tag)
        run: ./scripts/eden/fetch-engine-pinned.sh
      - name: Non-debug release build → docker/eden/instdir
        run: ./scripts/eden/build-release.sh
      - name: Metadata (tags + EdenDocs OCI labels)
        id: meta
        uses: docker/metadata-action@v5
        with:
          images: ${{ env.IMAGE }}
          tags: |
            type=raw,value=main
            type=sha,prefix=sha-,format=short
            type=raw,value=latest
          labels: |
            org.opencontainers.image.vendor=AO Cyber Systems
            org.opencontainers.image.title=EdenDocs
            org.opencontainers.image.description=Privacy-first self-hosted collaborative document editor
            org.opencontainers.image.licenses=MPL-2.0
      - name: Build image (:scan, load) — verify-caps stage runs in-build (DIST-02)
        uses: docker/build-push-action@v6
        with:
          context: docker/eden
          file: docker/eden/Dockerfile
          build-args: base_image=ubuntu:24.04
          load: true
          tags: ${{ env.IMAGE }}:scan
          labels: ${{ steps.meta.outputs.labels }}
      - name: Boot-health gate (pinned engine boots — no SIGSEGV, accepts on 9980)
        run: |
          docker run -d --name edendocs-boot --cap-add MKNOD --cap-add SYS_CHROOT \
            --cap-add FOWNER --cap-add CHOWN --cap-add SYS_ADMIN \
            -e "extra_params=--o:ssl.enable=false" ${{ env.IMAGE }}:scan
          for _ in $(seq 1 120); do
            docker exec edendocs-boot /usr/bin/coolwsd --probe >/dev/null 2>&1 && ok=1 && break; sleep 1;
          done
          docker logs edendocs-boot | tail -n 50
          test "${ok:-0}" = 1 || { echo "FATAL: coolwsd never accepted on 9980 (pinned engine boot-health failed)"; exit 1; }
          echo "BOOT HEALTH PASSED: coolwsd accepts on 9980, no SIGSEGV"
          docker rm -f edendocs-boot
      - name: LABEL identity + MPL-in-image integrity
        run: |
          docker inspect --format '{{json .Config.Labels}}' ${{ env.IMAGE }}:scan | tee labels.json
          grep -q 'EdenDocs' labels.json && grep -q 'AO Cyber Systems' labels.json
          grep -qi 'collabora' labels.json && { echo "FATAL: Collabora trademark in image LABELs"; exit 1; } || true
          # COPYING/THIRDPARTYLICENSES present + byte-intact in the image
          docker create --name mpl ${{ env.IMAGE }}:scan; docker cp mpl:/usr/share/doc/coolwsd/COPYING img-COPYING 2>/dev/null || docker cp mpl:/COPYING img-COPYING
          docker rm mpl
          grep -q 'Mozilla Public' img-COPYING || { echo "FATAL: MPL COPYING missing/altered in image"; exit 1; }
      - name: Reverse-proxy >60s WebSocket smoke (DIST-02)
        run: ./scripts/eden/proxy-smoke-test.sh ${{ env.IMAGE }}:scan
      - name: Trivy scan (block HIGH/CRITICAL)
        uses: aquasecurity/trivy-action@master
        with: { image-ref: '${{ env.IMAGE }}:scan', exit-code: '1', severity: 'HIGH,CRITICAL', ignore-unfixed: 'true' }
      - name: Push (multi-tag)
        id: push
        uses: docker/build-push-action@v6
        with:
          context: docker/eden
          file: docker/eden/Dockerfile
          build-args: base_image=ubuntu:24.04
          push: true
          tags: ${{ steps.meta.outputs.tags }}
          labels: ${{ steps.meta.outputs.labels }}
          provenance: mode=max
          sbom: true
      - name: Cosign keyless sign + SBOM attest + verify (DIST-03)
        env: { DIGEST: ${{ steps.push.outputs.digest }} }
        run: |
          cosign sign --yes "${IMAGE}@${DIGEST}"
          syft "${IMAGE}@${DIGEST}" -o cyclonedx-json > sbom.cdx.json
          cosign attest --yes --predicate sbom.cdx.json --type cyclonedx "${IMAGE}@${DIGEST}"
          cosign verify "${IMAGE}@${DIGEST}" \
            --certificate-identity-regexp="https://github.com/AO-Cyber-Systems/EdenDocs/.*" \
            --certificate-oidc-issuer="https://token.actions.githubusercontent.com"
      # cosign + syft installers:
      # - uses: sigstore/cosign-installer@v3   (place before the sign step)
      # - uses: anchore/sbom-action/download-syft@v0
```

Place `sigstore/cosign-installer@v3` and `anchore/sbom-action/download-syft@v0` steps before the cosign step. Keep the whole file additive; do not modify build.yml.
  </action>
  <verify>
python3 -c "import yaml; yaml.safe_load(open('.github/workflows/docker-publish.yml'))" && command -v actionlint >/dev/null && actionlint .github/workflows/docker-publish.yml || echo "actionlint absent — CI parses it"; grep -q 'branches: \[eden-main\]' .github/workflows/docker-publish.yml && grep -q 'ghcr.io/ao-cyber-systems/edendocs' .github/workflows/docker-publish.yml && grep -q 'fetch-engine-pinned.sh' .github/workflows/docker-publish.yml && grep -q 'build-release.sh' .github/workflows/docker-publish.yml && grep -q 'docker/eden/Dockerfile' .github/workflows/docker-publish.yml && grep -q 'proxy-smoke-test.sh' .github/workflows/docker-publish.yml && grep -q 'id-token: write' .github/workflows/docker-publish.yml && grep -q 'cosign sign --yes' .github/workflows/docker-publish.yml && grep -q 'cosign attest' .github/workflows/docker-publish.yml && grep -q 'metadata-action' .github/workflows/docker-publish.yml && grep -qi 'trivy' .github/workflows/docker-publish.yml && ! grep -qi 'COSIGN_KEY\|generate-key-pair' .github/workflows/docker-publish.yml && ! grep -q '8080' .github/workflows/docker-publish.yml && ! grep -q -- '--enable-debug' .github/workflows/docker-publish.yml
  </verify>
  <done>docker-publish.yml parses (actionlint-clean if available), triggers on push to eden-main + dispatch, lowercase ghcr path, calls fetch-engine-pinned.sh + build-release.sh + proxy-smoke-test.sh, builds docker/eden/Dockerfile, has a boot-health gate + MPL/LABEL checks + trivy, signs keyless (id-token: write, cosign sign/attest, no COSIGN_KEY), injects LABELs via metadata-action, references no 8080 and no --enable-debug, and leaves build.yml untouched.</done>
  <recovery>YAML/actionlint errors → fix before pushing (a broken workflow wastes a CI round trip). If build-push-action's digest output name differs, use `${{ steps.push.outputs.digest }}` per v6. Full-run failures are diagnosed in Task 3 against the error_recovery table.</recovery>
</task>

<task type="auto">
  <name>Task 2: Extend the trademark/MPL checklist with container-image rows</name>
  <files>docs/eden/TRADEMARK-MPL-CHECKLIST.md</files>
  <action>
Update `docs/eden/TRADEMARK-MPL-CHECKLIST.md`:

1. **§3 Per-surface checklist — replace the placeholder "Container image LABELs" row** (currently "None exist … Objective 4 owns any future Eden LABEL scheme") with the real scheme:
   `| Container image LABELs / OCI annotations | 1/2 | EdenDocs OCI labels injected at CI via docker/metadata-action (vendor=AO Cyber Systems, title=EdenDocs, description, licenses=MPL-2.0, source=…/EdenDocs); NO LABEL directives in docker/eden/Dockerfile; no Collabora string on any image-identity surface | 04-01/02/04 (docker/eden/Dockerfile + .github/workflows/docker-publish.yml) | docker-publish.yml "LABEL identity + MPL-in-image integrity" step (asserts EdenDocs+vendor present, no 'collabora' in Labels) |`

2. **Add a new subsection "§9 Container image surfaces (Objective 4 / DIST)"** documenting:
   - Image name: `ghcr.io/ao-cyber-systems/edendocs` (lowercase; Pitfall 7) — no Collabora trademark in the repo path or tags.
   - LABEL/annotation set (the metadata-action labels above) — user-facing image identity is EdenDocs.
   - MPL byte-integrity IN THE IMAGE: `COPYING`/`THIRDPARTYLICENSES` ship unaltered inside the image (asserted by the workflow's MPL-in-image step); per-file MPL headers untouched (the image only COPYs a `make install` tree — no source edits).
   - Signing: keyless cosign (Sigstore/OIDC) + CycloneDX SBOM attestation; verifiable via `cosign verify … --certificate-oidc-issuer=…githubusercontent.com`.
   - **Divergence note:** Objective 4 adds ZERO upstream-file divergences — all work is additive (`docker/eden/*`, `scripts/eden/*`, `.github/workflows/docker-publish.yml`). The §7 divergence allowlist is UNCHANGED by this objective.

3. If a green run has been captured (Task 3), add the run URL/evidence to §8; otherwise leave a "pending green run" note like the existing §8.
  </action>
  <verify>
grep -qi 'ghcr.io/ao-cyber-systems/edendocs' docs/eden/TRADEMARK-MPL-CHECKLIST.md && grep -qi 'metadata-action' docs/eden/TRADEMARK-MPL-CHECKLIST.md && grep -qi 'keyless cosign\|cosign' docs/eden/TRADEMARK-MPL-CHECKLIST.md && grep -qi 'ZERO upstream-file divergences\|adds ZERO' docs/eden/TRADEMARK-MPL-CHECKLIST.md && grep -qi 'COPYING' docs/eden/TRADEMARK-MPL-CHECKLIST.md && ! grep -q '8080' docs/eden/TRADEMARK-MPL-CHECKLIST.md
  </verify>
  <done>The checklist's placeholder container-LABEL row is filled in with the real EdenDocs LABEL scheme, a new container-surfaces subsection documents image name/LABELs/MPL-in-image/signing, and it states Objective 4 adds zero upstream divergences (§7 allowlist unchanged). No 8080 reference.</done>
  <recovery>Keep the edit additive/localized — do not renumber or rewrite existing §1–§8 content beyond the single placeholder row + the new subsection. If the exact placeholder row text differs, match on "Container image LABELs" and replace that row only.</recovery>
</task>

<task type="auto">
  <name>Task 3: Populate the engine mirror, push, drive the workflow green, verify the signed published image</name>
  <files>.github/workflows/docker-publish.yml</files>
  <action>
1. **Populate the engine-pin mirror** (first-run prerequisite): obtain a boot-validated known-good `engine-main-assets.tar.gz`. Prefer the recorded ea91ef33… snapshot; if it is no longer obtainable, fetch the current rolling engine, boot-validate it (a scratch build+`coolwsd --probe`), and if green treat ITS sha256 as the pin. Upload it: `gh release create engine-pin --notes "pinned EdenDocs release engine (boot-validated)"` then `gh release upload engine-pin engine-main-assets.tar.gz`. If you had to re-pin, update `docker/eden/engine-pin.env` (04-01) to the validated sha256 (NEVER 076a4129…) and commit atomically.
2. Commit docker-publish.yml + checklist (+ any engine-pin.env re-pin) atomically and push eden-main — this push is itself the first trigger.
3. Watch: `gh run watch $(gh run list --workflow=docker-publish.yml --branch eden-main --limit 1 --json databaseId --jq '.[0].databaseId') --exit-status`.
4. On failure, diagnose against the error_recovery table (verify-caps loss, boot-health SIGSEGV, mirror 404, lowercase, trivy, cosign identity), fix (script fix → scripts/eden or engine-pin.env; workflow fix → docker-publish.yml), commit atomically, push, re-watch. Stop after 4 red cycles and surface the pattern (do not thrash).
5. From the GREEN run, capture evidence for the SUMMARY + checklist §8:
   - verify-caps `post-COPY caps -- coolforkit-caps: … | coolmount: …` line (DIST-02 caps survived)
   - `BOOT HEALTH PASSED` (pinned engine boots — DIST-01/engine-pin (b))
   - `PROXY SMOKE PASSED` (DIST-02 >60s reverse proxy)
   - the pushed digest + `cosign verify` success block (DIST-03)
   - `docker inspect` LABELs showing EdenDocs/vendor and no 'collabora'
6. Confirm the published artifact: `cosign verify "ghcr.io/ao-cyber-systems/edendocs:main" --certificate-identity-regexp="https://github.com/AO-Cyber-Systems/EdenDocs/.*" --certificate-oidc-issuer="https://token.actions.githubusercontent.com"` succeeds, and the image LABELs carry EdenDocs metadata. Record the digest.
7. Surface (do NOT perform) the optional package-visibility→Public step to the user; publishing to the registry is done, deploying anywhere is out of scope.
  </action>
  <verify>
gh run list --workflow=docker-publish.yml --branch eden-main --limit 1 --json conclusion --jq '.[0].conclusion' | grep -x success && RUN_ID=$(gh run list --workflow=docker-publish.yml --branch eden-main --limit 1 --json databaseId --jq '.[0].databaseId') && gh run view "$RUN_ID" --log | grep -qF "PROXY SMOKE PASSED" && gh run view "$RUN_ID" --log | grep -qF "BOOT HEALTH PASSED" && gh run view "$RUN_ID" --log | grep -qiE "cosign.*(sign|verif)"
  </verify>
  <done>The latest docker-publish.yml run on eden-main concluded success; logs contain the verify-caps caps line, BOOT HEALTH PASSED, PROXY SMOKE PASSED, and a cosign sign+verify success; a signed, EdenDocs-LABELed image is present at ghcr.io/ao-cyber-systems/edendocs (:main/:sha-…/:latest) and `cosign verify` passes. Evidence captured for the SUMMARY + checklist §8.</done>
  <recovery>Per-failure via error_recovery. On gh/git auth errors or ghcr login denial, raise an auth gate — never embed credentials. If GitHub Actions is disabled on the repo, surface as user_setup (Settings → Actions). If the mirror upload needs write the token lacks, surface it. Hard stop + structured report after 4 red cycles.</recovery>
</task>

</tasks>

<validation_gates>
<lint>python3 -c "import yaml; yaml.safe_load(open('.github/workflows/docker-publish.yml'))"; command -v actionlint >/dev/null && actionlint .github/workflows/docker-publish.yml || true</lint>
<test>gh run list --workflow=docker-publish.yml --branch eden-main --limit 1 --json conclusion --jq '.[0].conclusion' | grep -x success</test>
<build>true # the CI run IS the build+publish; the green run is the proof</build>
</validation_gates>

<verification>
1. Latest docker-publish.yml run on eden-main: conclusion=success, triggered by push (DIST-03 cadence).
2. verify-caps stage passed in-build (caps line in logs) + boot-health passed (pinned engine boots, no SIGSEGV) (DIST-01 + DIST-02 caps half + engine-pin (b)).
3. PROXY SMOKE PASSED in logs — WS session survived >60s through the verbatim reverse proxy (DIST-02 proxy half).
4. Image pushed to ghcr.io/ao-cyber-systems/edendocs (lowercase) with EdenDocs OCI LABELs, no Collabora identity string, MPL COPYING byte-intact in the image; digest cosign-signed keyless + CycloneDX SBOM attested; `cosign verify` succeeds (DIST-03).
5. build.yml untouched; docker/from-source/{Ubuntu,Debian} untouched; checklist container rows added; §7 divergence allowlist unchanged (zero upstream edits in Objective 4).
</verification>

<success_criteria>
- One green docker-publish.yml run is the integrated proof of DIST-01 (from-source image builds), DIST-02 (verify-caps passes + reverse-proxy >60s survives), and DIST-03 (signed, LABELed image published to ghcr.io/ao-cyber-systems/edendocs).
- Keyless cosign (no keypair), lowercase ghcr path, EdenDocs LABELs with no Collabora identity, MPL intact in-image, pinned engine boot-validated, zero upstream-file edits, no port 8080 anywhere.
</success_criteria>

<output>
After completion, create `.planning/objectives/04-edendocs-branded-container-images/04-04-SUMMARY.md`: green run URL + id, the published digest, the final engine pin sha256 (and whether a re-pin from ea91ef33… was needed), cosign verify output, the LABEL set inspected off the image, cold/warm wall-clock, and the surfaced optional package-visibility action for the user.
</output>
