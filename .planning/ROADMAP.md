# Roadmap: EdenDocs

## Overview

EdenDocs turns the `online.mirror` fork into a shippable Eden product in five
dependency-ordered objectives. A working from-source build comes first,
because nothing else — visual branding, an end-to-end OIDC round trip, or a
container image — can be verified without it. Once the build is stable, the
web-UI rebrand and the AOID/OIDC integration proceed in parallel: they touch
disjoint parts of the tree (config + `eden-branding/` vs. a standalone WOPI
host service) and share only the Objective 1 build as a dependency. Branded
container images come next, since they need both the build output and the
finished branding assets. The upstream-tracking workflow is written and
finalized last (once Objective 2 has proven the config/additive-file rebrand
seams hold with zero upstream edits), but the *rule* it will formalize — the
branch-tracking decision and merge cadence — must be adopted informally
before Objective 2 begins, per UPST-01 and research pitfall 2. See the note
under Objective 5 below.

**Environment constraint (all objectives):** never use port 8080 for any
build, serve, or verification step. Local web checks use 8091; coolwsd's
native runtime port is 9980.

## Objectives

**Objective Numbering:**
- Integer objectives (1, 2, 3): Planned milestone work
- Decimal objectives (2.1, 2.2): Urgent insertions (marked with INSERTED)

Decimal objectives appear between their surrounding integers in numeric order.

- [x] **Objective 1: From-Source Build Pipeline** - Build coolwsd + browser/dist from source on Linux via the prebuilt engine tarball, verified in CI on every push (completed 2026-07-08)
- [ ] **Objective 2: EdenDocs Rebrand of the Web UI** - Editor and admin console show EdenDocs branding end-to-end; Collabora trademarks stripped, MPL attribution preserved
- [ ] **Objective 3: AOID Authentication Integration (OIDC)** - A reference WOPI host authenticates users against AOID and surfaces their identity inside the editor
- [ ] **Objective 4: EdenDocs-Branded Container Images** - Signed, branded container images built on the from-source pattern, verified through a real reverse proxy
- [ ] **Objective 5: Upstream Tracking Workflow** - A written, owned merge cadence and divergence allowlist keep eden-main mergeable against the fast-moving mirror

## Objective Details

### Objective 1: From-Source Build Pipeline
**Goal**: A developer can build `coolwsd` + `browser/dist` from EdenDocs source on Linux using Collabora's prebuilt engine/LOKit tarball (no from-scratch `engine/` compile), and every push is verified by CI on GitHub-hosted runners.
**Depends on**: Nothing (first objective)
**Requirements**: BUILD-01, BUILD-02, BUILD-03, BUILD-04
**Success Criteria** (what must be TRUE):
  1. A developer can run a documented build on Linux that produces a working `coolwsd` binary and `browser/dist` using the `ENGINE_ASSETS` prebuilt tarball fast path, with no from-scratch `engine/` compile
  2. The build's reliance on in-tree POCO auto-discovery (`--with-lo-builddir`, no `--with-poco-*` flags) against the prebuilt tarball is validated early, with a documented fallback if auto-discovery fails
  3. Every push to `eden-main` triggers a GitHub Actions run on `ubuntu-latest` that completes a full configure + build using Node 20 LTS and ccache, modeled on the in-tree `codeql-analysis.yml` invocation
  4. The built `coolwsd` serves `/hosting/discovery` and `/hosting/capabilities` correctly on its native port 9980 (never port 8080)
**Plans**: 3 TRDs (3 waves, sequential)

Jobs:
- [x] 01-01: Build scripts + early POCO/tarball validation (01-01-build-scripts-TRD.md) — wave 1
- [x] 01-02: Push-triggered CI workflow green on eden-main (01-02-ci-workflow-TRD.md) — wave 2
- [x] 01-03: Developer build documentation BUILDING-EdenDocs.md (01-03-build-doc-TRD.md) — wave 3

### Objective 2: EdenDocs Rebrand of the Web UI
**Goal**: Every user-facing surface of the editor and admin console — name, logo, colors, favicon, About dialog, help links — shows EdenDocs branding, delivered through config keys and an additive `eden-branding/` directory rather than scattered source edits, with Collabora trademarks removed and MPL/legal attribution preserved.
**Depends on**: Objective 1 (needs a working build to visually verify branding)
**Requirements**: BRAND-01, BRAND-02, BRAND-03, BRAND-04, BRAND-05, BRAND-06
**Success Criteria** (what must be TRUE):
  1. Opening the editor shows the EdenDocs product name and logo, driven by `coolwsd.xml`'s `brandProductName`/`brandProductURL`/`logoURL` plus the additive `eden-branding/` directory (`branding.css`, `branding.js`, logo SVGs, `toolbar-bg-logo`) wired via `--with-app-branding` — no upstream source files edited to achieve it
  2. The browser tab title and About dialog read "EdenDocs" (via the two isolated `<title>` patches in `cool.html.m4`/`admintemplate.html`, kept as their own commits, plus compiled `--with-app-name="EdenDocs"`/`--with-vendor="AO Cyber Systems"` fallbacks)
  3. The admin console shows the same EdenDocs branding as the editor via the shared `%BRANDING_JS%` hook, with the favicon replaced and the color palette overridden via a branding-dir CSS file (upstream `color-palette.css` never edited in place)
  4. No Collabora trademarks (names, logos, `help_url`, container `LABEL` metadata) remain on any user-facing surface, verified against a written trademark/MPL checklist confirming MPL headers, `COPYING`, and `THIRDPARTYLICENSES` attribution are still intact
  5. `INFOBAR_URL`/`FEEDBACK_URL`/`INFO_URL` are confirmed unset and `--with-support-public-key`/`--with-welcome-url` are confirmed never passed to the build (no phone-home, no silently-dropped branding keys)
**Plans**: 5 TRDs (3 waves: 01+02+03 parallel, then 04, then 05)

Jobs:
- [x] 02-01: Isolated upstream `<title>` patches, editor + admin console (02-01-upstream-title-patches-TRD.md) — wave 1
- [x] 02-02: Backstage header + admin Version-tab trademark patches (02-02-upstream-trademark-patches-TRD.md) — wave 1
- [x] 02-03: Additive eden-branding/ assets: real AO emblem, gold palette CSS, branding.js, favicon, welcome page (02-03-eden-branding-assets-TRD.md) — wave 1
- [x] 02-04: Build wiring: configure flags, coolwsd.xml brand keys, favicon/welcome overlays, branding smoke checks, CI green (02-04-build-wiring-TRD.md) — wave 2
- [ ] 02-05: Trademark/MPL checklist + verify-branding.sh mechanical CI gate (02-05-trademark-checklist-verification-TRD.md) — wave 3

> **Prerequisite (not a formal objective dependency):** Per UPST-01 and research
> pitfall 2, the upstream branch-tracking decision and merge-cadence rule
> (which branch `eden-main` tracks, how often it merges) must be **decided and
> recorded before this objective's work begins** — even though Objective 5
> below formally writes and finalizes that documentation last. Record the
> decision as a Key Decision in PROJECT.md (or a STATE.md note) before
> starting Objective 2 execution. See Objective 5's note for the full
> rationale.

### Objective 3: AOID Authentication Integration (OIDC)
**Goal**: Users authenticate against AOID before opening a document, via a new reference WOPI host service (outside this repo's `wsd/`/`browser/` tree), and their AOID identity is visible inside the editor — with no OIDC code added to coolwsd itself.
**Depends on**: Objective 1 (needs a running coolwsd to test the end-to-end round trip); independent of Objective 2, can run in parallel
**Requirements**: AUTH-01, AUTH-02, AUTH-03, AUTH-04
**Success Criteria** (what must be TRUE):
  1. A user visiting the reference WOPI host's launch page is taken through AOID's OIDC Authorization Code flow and returned to an embedded editor iframe
  2. The WOPI host correctly implements CheckFileInfo/GetFile/PutFile and mints short-lived WOPI `access_token`s that coolwsd accepts for document sessions
  3. The editor UI displays the authenticated user's name and avatar (and admin flag where applicable), sourced from CheckFileInfo's `UserFriendlyName`/avatar/`IsAdminUser` fields
  4. coolwsd trusts the reference WOPI host only because it is explicitly registered in `storage.wopi.alias_groups` (`mode="groups"`) with proof-key validation enabled, and no OIDC/OAuth code exists anywhere in `wsd/`
**Plans**: TBD

Jobs:
- [ ] 03-01: TBD (planned by /devflow:plan-objective)

> **Non-linear dependency:** Objective 3 depends on Objective 1 (not
> Objective 2) because the WOPI host is a separate deployable that shares no
> code with the branding work. Objectives 2 and 3 can execute in parallel via
> `/devflow:workstreams`. Objective 4 below is a join point that waits for
> both.

### Objective 4: EdenDocs-Branded Container Images
**Goal**: Customers can deploy a signed, EdenDocs-branded container image, built on the from-source pattern, that survives real-world reverse-proxy conditions rather than only direct-connect access.
**Depends on**: Objective 1, Objective 2 (needs build output and finished branding assets); independent of Objective 3 — the WOPI host is a separate deployable and is not baked into this image
**Requirements**: DIST-01, DIST-02, DIST-03
**Success Criteria** (what must be TRUE):
  1. A container image builds successfully from the `docker/from-source/{Ubuntu,Debian}` pattern, embedding the prebuilt engine, EdenDocs-built `wsd`/`browser`, and `eden-branding` assets — not the `from-packages` repackage path
  2. The image's `verify-caps` capability-check build stage still passes, and a document session held open through a real reverse proxy survives longer than 60 seconds (not just direct `127.0.0.1:9980` access)
  3. Published images are signed with cosign and available in the AO-Cyber-Systems registry, carrying EdenDocs `LABEL` metadata
**Plans**: TBD

Jobs:
- [ ] 04-01: TBD (planned by /devflow:plan-objective)

### Objective 5: Upstream Tracking Workflow
**Goal**: EdenDocs has a written, owned discipline for staying mergeable against the fast-moving `online.mirror` indefinitely — a fixed merge cadence, an explicit branch-tracking decision, and a divergence allowlist checked at every merge.
**Depends on**: Objective 1 (technical baseline); the underlying *rule* (branch-tracking decision + cadence) must be decided before Objective 2 begins — see note below
**Requirements**: UPST-01, UPST-02
**Success Criteria** (what must be TRUE):
  1. A written document states the merge cadence (weekly minimum) and the explicit branch-tracking decision (`upstream/main` vs. a `distro/collabora/co-XX.04` branch)
  2. That branch-tracking decision is adopted and recorded (e.g., as a Key Decision in PROJECT.md) before Objective 2's rebrand commits land — not merely documented after the fact
  3. A divergence allowlist exists enumerating every file permitted to differ from upstream (the `<title>` patches, `eden-branding/` wiring points, etc.) and is checked at every upstream merge

> **Why this objective is sequenced last but its rule is not:** The
> *documented* workflow is deliberately finalized last, once Objective 2 has
> proven the config/additive-file rebrand seams hold with zero upstream-file
> edits — the doc should codify seams actually proven to work, not
> theoretical ones (research: Implications for Roadmap). But UPST-01 itself
> requires the underlying rule to exist "as a rule before rebrand work
> begins" (research pitfall 2: upstream drift becomes unmergeable within
> weeks if this is deferred). Resolution: decide and record the
> branch-tracking rule informally before Objective 2 starts; execute this
> objective's full job list (formal write-up + validated divergence
> allowlist) in its numbered position, informed by what Objective 2 actually
> touched.
**Plans**: TBD

Jobs:
- [ ] 05-01: TBD (planned by /devflow:plan-objective)

## Progress

**Execution Order:**
Objectives execute in numeric order: 1 → 2 → 3 → 4 → 5 (2 and 3 may run in
parallel via `/devflow:workstreams`; the Objective 5 branch-tracking *rule*
must be decided before Objective 2 begins, per the note above).

| Objective | Jobs Complete | Status | Completed |
|-------|----------------|--------|-----------|
| 1. From-Source Build Pipeline | 3/3 | Complete    | 2026-07-08 |
| 2. EdenDocs Rebrand of the Web UI | 0/TBD | Not started | - |
| 3. AOID Authentication Integration (OIDC) | 0/TBD | Not started | - |
| 4. EdenDocs-Branded Container Images | 0/TBD | Not started | - |
| 5. Upstream Tracking Workflow | 0/TBD | Not started | - |

---
*Roadmap created: 2026-07-07*
*Depth: standard*
