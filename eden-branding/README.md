<!--
  Copyright the EdenDocs contributors / AO Cyber Systems.

  This Source Code Form is subject to the terms of the Mozilla Public
  License, v. 2.0. If a copy of the MPL was not distributed with this
  file, You can obtain one at http://mozilla.org/MPL/2.0/.
-->

# eden-branding/

Self-contained, **fully additive** EdenDocs brand overlay. Nothing here edits an
upstream file — every asset is consumed by a build-time copy or overlay wired up
in TRD 02-04 (`scripts/eden/build.sh`) and the upstream `--with-app-branding`
Makefile recipe. This directory is the single source of truth for the EdenDocs
mark, palette, and product-name globals.

## What each file is, and which mechanism consumes it

| File | Consumed by | Effect |
|---|---|---|
| `branding.css` | `--with-app-branding` — `branding*` glob → `browser/dist/` root | Overrides the primary palette to Eden gold (see token table). |
| `branding.js` | `--with-app-branding` — `branding*` glob → `browser/dist/` root; injected via the `%BRANDING_JS%` template tag | Defines `window.brandProductName` / `window.brandProductURL` globals. |
| `images/eden-emblem.svg` | source asset for the other SVGs + favicon; also overlaid for the welcome page | The official AO emblem, verbatim. |
| `images/collabora-office-white.svg` | `--with-app-branding` — `images/*.svg` glob → `browser/dist/images/` with **same-filename overwrite** | Replaces the Backstage logo referenced by `backstage.css` (`.backstage-header-title::before`). |
| `images/toolbar-bg-logo.svg` | `--with-app-branding` — `cp` **renamed** to `images/toolbar-bg.svg` unconditionally | Required hook file. The Makefile recipe **FAILS if this source is absent**, even though EdenDocs does not use a distinct toolbar background. |
| `images/eden-logo.svg` | Eden-owned lockup (emblem + "EdenDocs" wordmark) for docs/marketing surfaces | Not consumed by `--with-app-branding`; referenced by EdenDocs overlays. |
| `favicon.ico` | **NOT** covered by `--with-app-branding` on Linux → overlaid by `scripts/eden/build.sh` (TRD 02-04) | Multi-size (16/32/48) browser-tab icon derived from the emblem. |
| `welcome/welcome.html` + `welcome.css` + `eden-emblem.svg` | **NOT** covered by `--with-app-branding` on Linux → overlaid by `scripts/eden/build.sh` (TRD 02-04) | Trademark-free replacement welcome page, self-contained (HTML + CSS + SVG only, no JS, no slide images). |

## ⚠️ DO NOT RENAME `images/collabora-office-white.svg`

**The filename collision IS the zero-edit overwrite mechanism.** The upstream
Backstage stylesheet hard-codes `url('images/collabora-office-white.svg')` in
`browser/css/backstage.css`. We never touch that stylesheet; instead we ship a
file with the **same name** so the `images/*.svg` copy glob overwrites the
upstream Backstage glyph with the AO emblem. **Renaming this file silently
restores the Collabora logo in the Backstage view** — the CSS still points at the
old name, and our replacement never lands. Keep the name exactly as-is.

## Emblem provenance

`images/eden-emblem.svg` is the official AO emblem, downloaded verbatim from
<https://aocyber.ai/images/ao-icon.svg> (same file as the aocyber.ai favicon).
Its gold gradient (`#fce88d → #efb32c`) is preserved exactly — **never recolor
it, never substitute a placeholder or a redrawn mark.** All derived assets embed
this file (the logo lockup nests it as a `<svg>` so the class-based gradient
definitions stay intact rather than extracting paths).

## Brand token overrides (`branding.css`)

Every custom property carries `!important`. This is **required**, not cosmetic:
the `?darkTheme=true` runtime script `appendChild()`s `color-palette-dark.css`
into `<head>` **after** our link element, so any non-`!important` override is
silently reverted to Collabora blue (2-RESEARCH.md Pitfall 2).

| Property | Value | Ramp | Note |
|---|---|---|---|
| `--color-primary` | `#D4A853` | gold-500 | Primary accent. |
| `--color-primary-dark` | `#A67A38` | gold-700 | |
| `--color-primary-darker` | `#856131` | gold-800 | |
| `--color-primary-lighter` | `#F4D5AA` | gold-200 | Background use. |
| `--color-primary-text` | `#3D2A14` | gold-950 | **Flipped dark.** Upstream white is the fg used on the light-gold `-lighter` background and fails WCAG there — so foreground goes to gold-950. |
| `--color-btn-primary-hover-bg` | `#FAECD5` | gold-100 | |
| `--color-overlay` | `#D4A85314` | gold @ 8% α | Alpha-hex. |

`--color-hyperlink` is **intentionally left at the upstream value** — it is a
standard link affordance, not a trademark surface, and recoloring it would hurt
link legibility without any branding benefit.

## Port rule (dev-machine hard rule)

**Never use port 8080** for anything on this machine — it is permanently
occupied by another app. For any local web verification of these assets use port
**8091**. `coolwsd`'s own native port is **9980**. This TRD (02-03) needs no
server; the constraint applies to any later manual smoke check.
