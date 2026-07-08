---
kind: app
github_repo: AO-Cyber-Systems/EdenDocs
org_project: PVT_kwDODwqLrc4BRsOP
org: AO-Cyber-Systems
---

# EdenDocs

## What This Is

EdenDocs is the document-editing suite of the Eden Productivity Suite — a fork of Collabora Online (LibreOffice-technology collaborative office suite) rebranded and integrated into the Eden platform. It gives Eden customers real-time collaborative Docs/Sheets/Slides editing in the browser, self-hosted and privacy-first.

Not to be confused with `~/dev/eden-docs`, which is the Eden platform documentation directory.

## Core Value

Eden customers can collaboratively edit documents in the browser on infrastructure they control — privacy-first, with no third-party document cloud.

## Requirements

### Validated

<!-- Inherited from upstream Collabora Online — shipped and working. -->

- ✓ Real-time collaborative editing of text documents, spreadsheets, and presentations (LibreOffice core) — existing
- ✓ WOPI host protocol integration for pluggable file-storage backends — existing
- ✓ Document conversion API (`/cool/convert-to`) — existing
- ✓ Admin console with live metrics — existing
- ✓ Mobile/responsive browser UI — existing
- ✓ Upstream docker image + Helm chart packaging — existing

### Active

- [ ] EdenDocs rebrand of the web UI (product name, logo, colors, about/help surfaces)
- [ ] AOID authentication integration (OIDC)
- [ ] From-source build pipeline (coolwsd + LibreOffice core LOKit)
- [ ] EdenDocs-branded container images
- [ ] Upstream tracking workflow (documented merge cadence from online.mirror into eden-main)

### Out of Scope

- Forking LibreOffice core itself — consume Collabora's prebuilt core/LOKit packages; core-level changes are upstream's domain
- Native iOS/Android apps — browser-first for v1; upstream mobile shells untouched
- New editing features beyond upstream — v1 is rebrand + integration + build, not editor feature development
- Contributing changes upstream via GitHub PRs — upstream development happens on Gerrit (gerrit.collaboraoffice.com); the GitHub mirror is read-only

## Context

- **Fork lineage:** Fork of `CollaboraOnline/online.mirror` (read-only GitHub mirror of the full source). Collabora moved active development to Gerrit at gerrit.collaboraoffice.com; the `CollaboraOnline/online` GitHub repo now hosts only issue tracking, the Helm chart, and docker build assets. An initial fork of that wrong repo was created and deleted on 2026-07-07.
- **Branch model:** `eden-main` is the default branch (all EdenDocs work); `upstream` remote tracks online.mirror for periodic merges. Same pattern as the ao-terminal fork (ao-main).
- **Upstream architecture:** `coolwsd` C++ WebSocket server, per-document sandboxed kit processes embedding LibreOfficeKit, `browser/` TypeScript frontend (leaflet-derived canvas UI), WOPI protocol between coolwsd and storage hosts. Build system is autotools (configure.ac); frontend built via npm inside the autotools flow.
- **Local reference runtime:** docker container `edendocs-code` (upstream `collabora/code` image, coolwsd 26.04.2.1) on 127.0.0.1:9980 with SSL disabled — used as behavioral reference while the from-source build comes up.
- **Eden ecosystem:** AOID is the identity provider (OIDC); Eden-Biz is the business platform and system of record; EdenDocs is the first product of the Eden Productivity Suite (Mail, Calendar, Drive, Docs, Sheets, Slides, Meet) referenced in the aocyber-marketing DataRoom and eden-circle planning.
- **License:** MPL-2.0 (file-level copyleft). Fork is public; preserve license headers and notices; source-availability obligations apply to distributed modifications.

## Constraints

- **Ports**: Never use port 8080 (permanently occupied on the dev machine). Local web verification uses 8091; coolwsd runtime uses its native 9980.
- **Upstream sync**: The mirror is read-only and moves fast. Rebrand and integration changes must be maintainable across upstream merges — prefer theming/config layers and additive files over scattered edits to upstream sources.
- **License**: MPL-2.0 — keep modified-file source availability, preserve notices, retain CODA third-party license attributions.
- **Brand**: EdenDocs ships under Eden product branding. RESOLVED 2026-07-08: Eden brand tokens = the Eden UI design system (`eden-ui-flutter` `EdenColors`), whose primary brand ramp defaults to **Eden gold `#D4A853`** (50–950 shades) — consistent with the AO Cyber master-brand gold. Logo = the official AO gold-gradient emblem (`https://aocyber.ai/images/ao-icon.svg`, gradient `#fce88d → #efb32c`) paired with an "EdenDocs" wordmark; never a text placeholder or recolored stand-in. Use darker ramp shades (600/700) where gold-on-light contrast is insufficient.
- **Build**: From-source coolwsd builds require LibreOffice core (LOKit) and POCO. Linux container builds are the deliverable; macOS is dev-convenience only.

## Key Decisions

| Decision | Rationale | Outcome |
|----------|-----------|---------|
| Fork `online.mirror`, not `CollaboraOnline/online` | The GitHub `online` repo no longer carries source (dev moved to Gerrit); the mirror is the full source | ✓ Good |
| `eden-main` default branch with `upstream` remote | Clean upstream merges; matches ao-terminal fork pattern | — Pending |
| Product name "EdenDocs" | Distinct from the `eden-docs` documentation folder; fits Eden product naming | ✓ Good |
| Rebrand via theming/config layer where possible | Survive fast-moving upstream merges with minimal conflicts | — Pending |
| Consume prebuilt LibreOffice core, don't fork it | Core is a separate multi-hour build and upstream's domain; EdenDocs owns wsd + browser layers | — Pending |
| **Branch-tracking rule (UPST-01, recorded 2026-07-08 before Objective 2):** `eden-main` tracks `upstream/main`, weekly-minimum merge cadence | `eden-main` already tracks `main`; `main` (and `co-25.04`) are the only branches publishing the `engine-*-assets.tar.gz` prebuilt tarball that Objective 1's build pipeline depends on — `co-26.04` has no tarball. Revisit only if Collabora publishes distro-branch engine assets | — Pending |
| **Eden brand tokens for EdenDocs (recorded 2026-07-08 before Objective 2):** Eden UI gold `#D4A853` ramp (from `eden-ui-flutter` `EdenColors.gold`, 50–950) + official AO emblem + "EdenDocs" wordmark | Pulled from the real Eden design system, not invented: `EdenColors.primary` "defaults to gold" per the canonical token file; matches AO master-brand gold `#d4a853`. Branding is config/CSS-driven (additive `eden-branding/`), so a future Eden-specific palette swap is cheap | — Pending |

---
*Last updated: 2026-07-08 — branch-tracking + brand-token decisions recorded ahead of Objective 2*
