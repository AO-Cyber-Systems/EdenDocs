# Requirements: EdenDocs

**Defined:** 2026-07-07
**Core Value:** Eden customers can collaboratively edit documents in the browser on infrastructure they control — privacy-first, with no third-party document cloud.

## v1 Requirements

Requirements for initial release. Each maps to roadmap objectives.

### Build Pipeline

- [ ] **BUILD-01**: Developer can build coolwsd + browser/dist from EdenDocs source on Linux using the prebuilt engine assets fast path (`ENGINE_ASSETS` tarball, no from-scratch `engine/` compile)
- [ ] **BUILD-02**: Build validates early that the prebuilt engine tarball satisfies in-tree POCO auto-discovery (`--with-lo-builddir`, no `--with-poco-*` flags); documented fallback if it doesn't
- [ ] **BUILD-03**: CI builds eden-main on GitHub-hosted ubuntu runners on every push, modeled on the in-tree `codeql-analysis.yml` configure invocation (Node 20 LTS, ccache)
- [ ] **BUILD-04**: Built coolwsd passes a smoke test — serves `/hosting/discovery` and `/hosting/capabilities` (never on port 8080; local checks use coolwsd's native 9980)

### Rebrand

- [ ] **BRAND-01**: Editor UI shows EdenDocs product name and logo via `coolwsd.xml` (`brandProductName`/`brandProductURL`/`logoURL`) plus an additive `eden-branding/` directory wired with `--with-app-branding` (branding.css, branding.js, logo SVGs, toolbar-bg-logo)
- [ ] **BRAND-02**: Window/tab titles and About dialog show EdenDocs (the two isolated `<title>` patches in `cool.html.m4` and `admintemplate.html`, kept as their own commits; `--with-app-name="EdenDocs"` / `--with-vendor="AO Cyber Systems"` compiled fallbacks)
- [ ] **BRAND-03**: Admin console carries EdenDocs branding via the shared `%BRANDING_JS%` hook
- [ ] **BRAND-04**: Favicon replaced and color palette overridden via a branding-dir CSS file (upstream `color-palette.css` never edited in place), using approved Eden brand tokens
- [ ] **BRAND-05**: Collabora trademarks removed from all user-facing surfaces (UI strings, help_url, container LABEL metadata) while MPL headers, COPYING, and THIRDPARTYLICENSES attribution are preserved — verified against a written trademark/MPL checklist
- [ ] **BRAND-06**: No-phone-home defaults verified: `INFOBAR_URL`/`FEEDBACK_URL`/`INFO_URL` unset, `--with-support-public-key` never passed, `--with-welcome-url` never passed

### Authentication (AOID)

- [ ] **AUTH-01**: A reference WOPI host service (new deployable, outside this repo's wsd/browser tree) authenticates users against AOID via OIDC Authorization Code flow
- [ ] **AUTH-02**: The WOPI host implements CheckFileInfo/GetFile/PutFile, mints short-lived WOPI access_tokens, and serves a launch page embedding the editor iframe
- [ ] **AUTH-03**: AOID identity is visible in the editor UI (UserFriendlyName, avatar, IsAdminUser passed through CheckFileInfo)
- [ ] **AUTH-04**: coolwsd trusts the WOPI host via `storage.wopi.alias_groups` (`mode="groups"`) with proof-key validation enabled — no OIDC code added to `wsd/`

### Container Images

- [ ] **DIST-01**: EdenDocs-branded container image built on the `docker/from-source/{Ubuntu,Debian}` pattern (prebuilt engine + EdenDocs-built wsd/browser + eden-branding assets) — not `from-packages`
- [ ] **DIST-02**: Image preserves the `verify-caps` capability-check stage and passes a deployment smoke test: a document session held open >60s through a real reverse proxy (not just direct 127.0.0.1:9980)
- [ ] **DIST-03**: Images are signed (cosign) and published to the AO-Cyber-Systems registry with EdenDocs LABEL metadata

### Upstream Tracking

- [ ] **UPST-01**: A written, owned merge cadence (weekly minimum) with an explicit branch-tracking decision (upstream/main vs a distro/collabora/co-XX.04 branch), started as a rule before rebrand work begins
- [ ] **UPST-02**: A documented divergence allowlist (the few files permitted to differ from upstream: `<title>` patches, etc.) checked at every upstream merge

## v2 Requirements

Deferred to future release. Tracked but not in current roadmap.

### Differentiators

- **TENANT-01**: Per-tenant runtime reskin via `css_variables`/`theme` iframe params exposed to Eden-Biz (mechanism exists upstream; zero-rebuild per-customer theming)
- **BRAND-07**: Outbound User-Agent patch (`getAgentString()` in net/Socket.cpp) so AOCore/Eden logs show "EdenDocs" instead of "COOLWSD"
- **ADMIN-01**: Admin console PAM bridge, then native OIDC login for the admin console
- **PWA-01**: PWA manifest and icon set (upstream has none)
- **PKG-01**: RPM/DEB packages with EdenDocs vendor strings (containers-only for v1)

## Out of Scope

Explicitly excluded. Documented to prevent scope creep.

| Feature | Reason |
|---------|--------|
| Forking `engine/` (LibreOffice core) | Consume prebuilt core; core is upstream's domain, 73k files, multi-hour builds |
| Native iOS/Android apps | Browser-first v1; upstream mobile shells untouched |
| New editor features beyond upstream | v1 is rebrand + integration + build, not editor development |
| Reusing Collabora's proprietary `online-branding` artwork | Private repo, not licensed to us; mechanism is open, artwork is not |
| Upstream contributions via GitHub PRs | Upstream development happens on Gerrit; mirror is read-only |
| Editing WOPI protocol surface for branding | Breaks integrator compatibility; anti-feature per research |

## Traceability

Which objectives cover which requirements. Updated during roadmap creation.

| Requirement | Objective | Status |
|-------------|-----------|--------|
| BUILD-01 | Objective 1 - From-Source Build Pipeline | Pending |
| BUILD-02 | Objective 1 - From-Source Build Pipeline | Pending |
| BUILD-03 | Objective 1 - From-Source Build Pipeline | Pending |
| BUILD-04 | Objective 1 - From-Source Build Pipeline | Pending |
| BRAND-01 | Objective 2 - EdenDocs Rebrand of the Web UI | Pending |
| BRAND-02 | Objective 2 - EdenDocs Rebrand of the Web UI | Pending |
| BRAND-03 | Objective 2 - EdenDocs Rebrand of the Web UI | Pending |
| BRAND-04 | Objective 2 - EdenDocs Rebrand of the Web UI | Pending |
| BRAND-05 | Objective 2 - EdenDocs Rebrand of the Web UI | Pending |
| BRAND-06 | Objective 2 - EdenDocs Rebrand of the Web UI | Pending |
| AUTH-01 | Objective 3 - AOID Authentication Integration (OIDC) | Pending |
| AUTH-02 | Objective 3 - AOID Authentication Integration (OIDC) | Pending |
| AUTH-03 | Objective 3 - AOID Authentication Integration (OIDC) | Pending |
| AUTH-04 | Objective 3 - AOID Authentication Integration (OIDC) | Pending |
| DIST-01 | Objective 4 - EdenDocs-Branded Container Images | Pending |
| DIST-02 | Objective 4 - EdenDocs-Branded Container Images | Pending |
| DIST-03 | Objective 4 - EdenDocs-Branded Container Images | Pending |
| UPST-01 | Objective 5 - Upstream Tracking Workflow | Pending |
| UPST-02 | Objective 5 - Upstream Tracking Workflow | Pending |

**Coverage:**
- v1 requirements: 19 total
- Mapped to objectives: 19
- Unmapped: 0 ✓

---
*Requirements defined: 2026-07-07*
*Last updated: 2026-07-07 after roadmap creation (5 objectives, full coverage)*
