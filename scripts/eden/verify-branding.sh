#!/usr/bin/env bash
# -*- Mode: sh; indent-tabs-mode: nil; sh-basic-offset: 2 -*-
#
# verify-branding.sh — EdenDocs mechanical rebrand gate (BRAND-01..06).
#
# Copyright 2026 AO Cyber Systems
#
# This Source Code Form is subject to the terms of the Mozilla Public
# License, v. 2.0. If a copy of the MPL was not distributed with this
# file, You can obtain one at http://mozilla.org/MPL/2.0/.
#
# STATIC checks only — sweeps the BUILT browser/dist tree, asserts MPL/legal
# files untouched, and asserts the no-phone-home flag set. Runtime checks
# live in scripts/eden/smoke-test.sh. This script never binds, serves,
# curls, or references any network port.
#
# The BRAND-05 sweep (gate 5) matches PER-OCCURRENCE snippets, not whole
# lines: esbuild --minify packs bundle.js densely, so line-based filtering
# would let one accepted string silently mask an undiscovered leak sharing
# the same physical line. Every hit gets a ±40-char snippet, and snippets
# are filtered against scripts/eden/trademark-exceptions.txt — whose every
# line MUST have a justifying row in docs/eden/TRADEMARK-MPL-CHECKLIST.md.
#
# Sweep scope mirrors upstream's own production ship-set: the excludes below
# are exactly what browser/Makefile.am's non-debug install-data-hook EXCLUDES
# (the dist/src debug source mirror and the developer test-harness pages).
# See the checklist's "Sweep scope" section.
set -euo pipefail

cd "$(dirname "$0")/../.."

EXCEPTIONS=scripts/eden/trademark-exceptions.txt
CHECKLIST=docs/eden/TRADEMARK-MPL-CHECKLIST.md

# --- Gate 0: exceptions-file hygiene -------------------------------------
# An EMPTY line in a grep -f pattern file matches EVERYTHING — with -v that
# silently disables the entire sweep. Comments ('#') would be matched as
# literals by grep -F and never fire — ban both, loudly.
test -f "$EXCEPTIONS" || { echo "ERROR: $EXCEPTIONS missing"; exit 1; }
test -f "$CHECKLIST" || { echo "ERROR: $CHECKLIST missing (every exception needs a checklist row)"; exit 1; }
if command grep -q '^$' "$EXCEPTIONS"; then
  echo "ERROR: $EXCEPTIONS contains an empty line — this would disable the whole sweep (grep -vf matches everything)"
  exit 1
fi
if command grep -q '^#' "$EXCEPTIONS"; then
  echo "ERROR: $EXCEPTIONS contains a '#' comment line — grep -F treats every line as a literal pattern"
  exit 1
fi

# --- Gate 1: BRAND-01 dist branding artifacts -----------------------------
test -d browser/dist || { echo "ERROR: browser/dist missing — run the build first (BRAND-01)"; exit 1; }
test -f browser/dist/branding.css || { echo "ERROR: browser/dist/branding.css missing (BRAND-01)"; exit 1; }
test -f browser/dist/branding.js || { echo "ERROR: browser/dist/branding.js missing (BRAND-01)"; exit 1; }
test -f browser/dist/images/toolbar-bg.svg || { echo "ERROR: browser/dist/images/toolbar-bg.svg missing (BRAND-01)"; exit 1; }
command grep -qi 'efb32c' browser/dist/images/collabora-office-white.svg \
  || { echo "ERROR: browser/dist/images/collabora-office-white.svg is not the AO gold emblem (BRAND-01 collision overwrite failed)"; exit 1; }

# --- Gate 2: BRAND-02 titles + vendor in dist ------------------------------
command grep -q '<title>EdenDocs</title>' browser/dist/cool.html \
  || { echo "ERROR: dist cool.html title is not EdenDocs (BRAND-02)"; exit 1; }
ADMIN_TEMPLATE=$(find browser/dist -name 'admintemplate.html' | head -n 1)
test -n "$ADMIN_TEMPLATE" \
  || { echo "ERROR: admintemplate.html not found under browser/dist (BRAND-02)"; exit 1; }
command grep -q '<title>EdenDocs - Admin console</title>' "$ADMIN_TEMPLATE" \
  || { echo "ERROR: $ADMIN_TEMPLATE title is not 'EdenDocs - Admin console' (BRAND-02)"; exit 1; }
command grep -q 'AO Cyber Systems' browser/dist/cool.html \
  || { echo "ERROR: vendor 'AO Cyber Systems' missing from dist cool.html (BRAND-02 --with-vendor)"; exit 1; }

# --- Gate 3: BRAND-03 admin-console product name ---------------------------
command grep -q 'window.brandProductName' browser/dist/branding.js \
  || { echo "ERROR: window.brandProductName missing from dist branding.js (BRAND-03)"; exit 1; }
command grep -qE "window\.brandProductName = ['\"]EdenDocs['\"]" browser/dist/branding.js \
  || { echo "ERROR: window.brandProductName is not 'EdenDocs' in dist branding.js (BRAND-03)"; exit 1; }

# --- Gate 4: BRAND-04 palette + welcome ------------------------------------
command grep -qF -- '--color-primary: #D4A853 !important' browser/dist/branding.css \
  || { echo "ERROR: gold --color-primary override missing from dist branding.css (BRAND-04)"; exit 1; }
test -d browser/dist/welcome \
  || { echo "ERROR: browser/dist/welcome missing — welcome replacement did not run (BRAND-04)"; exit 1; }
if command grep -rqi collabora browser/dist/welcome/; then
  echo "ERROR: browser/dist/welcome/ still references Collabora (BRAND-04 full replacement failed)"
  exit 1
fi

# --- Gate 5: BRAND-05 trademark sweep (the core gate) -----------------------
# Per-occurrence ±40-char snippets, filtered against the justified
# exceptions. Scope: production ship-set (see header comment + checklist).
SWEEP_SCOPE=(
  --include='*.html' --include='*.js' --include='*.css' --include='*.svg'
  --exclude-dir='src'
  --exclude='debug.html' --exclude='framed.doc.html' --exclude='framed.html'
  --exclude='load.doc.html' --exclude='multidocs.html'
)
SURVIVORS=$(command grep -rhoi '.\{0,40\}collabora.\{0,40\}' browser/dist \
  "${SWEEP_SCOPE[@]}" \
  | command grep -viFf "$EXCEPTIONS" || true)
if [ -n "$SURVIVORS" ]; then
  echo "ERROR: unexplained Collabora trademark occurrences in browser/dist:"
  echo "$SURVIVORS" | sort -u
  echo "--- files containing 'collabora' (for diagnosis) ---"
  command grep -rli 'collabora' browser/dist "${SWEEP_SCOPE[@]}" || true
  echo "Fix the surface or add a JUSTIFIED exception (checklist row required)."
  exit 1
fi

# --- Gate 6: BRAND-05 legal/MPL preservation --------------------------------
test -f COPYING || { echo "ERROR: COPYING missing (BRAND-05 legal)"; exit 1; }
test -f THIRDPARTYLICENSES || { echo "ERROR: THIRDPARTYLICENSES missing (BRAND-05 legal)"; exit 1; }
test -f CODA-THIRDPARTYLICENSES.html || { echo "ERROR: CODA-THIRDPARTYLICENSES.html missing (BRAND-05 legal)"; exit 1; }
git diff --quiet HEAD -- COPYING THIRDPARTYLICENSES CODA-THIRDPARTYLICENSES.html \
  || { echo "ERROR: legal files differ from committed HEAD state — MPL attribution must stay byte-identical (BRAND-05)"; exit 1; }
command grep -q 'Mozilla Public' COPYING \
  || { echo "ERROR: COPYING no longer carries the Mozilla Public License notice (BRAND-05)"; exit 1; }

# --- Gate 7: BRAND-06 no-phone-home ------------------------------------------
# build.sh's comments deliberately name these flags WITHOUT leading dashes;
# this script may contain the literal flags (it greps build.sh, not itself).
test -f scripts/eden/build.sh || { echo "ERROR: scripts/eden/build.sh missing (BRAND-06)"; exit 1; }
for FLAG in '--with-welcome-url' '--with-feedback-url' '--with-infobar-url' '--with-support-public-key'; do
  if command grep -q -- "$FLAG" scripts/eden/build.sh; then
    echo "ERROR: forbidden phone-home flag $FLAG present in scripts/eden/build.sh (BRAND-06 HARD OMIT LIST)"
    exit 1
  fi
done
if [ -f coolwsd.xml ]; then
  command grep -q '<brandProductName' coolwsd.xml \
    || { echo "ERROR: brandProductName canary missing from generated coolwsd.xml — a welcome URL was enabled and silently deleted the brand keys (see 2-RESEARCH.md Pitfall 1). Fix the flag, do NOT loosen this gate."; exit 1; }
else
  echo "WARNING: coolwsd.xml not present (pre-build local run) — skipping brandProductName canary (BRAND-06); CI always runs this post-build"
fi

echo "BRANDING VERIFICATION PASSED (BRAND-01..06)"
