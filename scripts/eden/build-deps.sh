#!/usr/bin/env bash
#
# build-deps.sh — EdenDocs from-source build pipeline: system dependencies.
#
# Installs the apt packages and the source-built POCO 1.12.5p2 required to
# configure/build EdenDocs (coolwsd) against the prebuilt ENGINE_ASSETS
# tarball fetched by scripts/eden/fetch-engine-assets.sh.
#
# WHY source-built POCO is mandatory (not optional, not apt):
#   - The engine-main-assets.tar.gz ships ONLY instdir/ — zero POCO headers,
#     zero workdir/ (empirically verified 2026-07-07, BUILD-02). configure's
#     --with-lo-builddir POCO auto-discovery therefore deterministically
#     fails and a system POCO >= 1.12.0 is required.
#   - Ubuntu's apt package `libpoco-dev` ships POCO 1.11.0, which is BELOW
#     configure.ac's hard floor of >= 1.12.0 (configure.ac:2231-2239). It
#     fails at configure time, wasting a full CI run.
#   - `--with-poco-includes` / `--with-poco-libs` are accepted by
#     configure.ac (365-391) but silently ignored (warning only) — they
#     pin nothing and cannot substitute for a real system install.
#   - This is the exact recipe proven in .github/workflows/codeql-analysis.yml
#     (the only in-tree CI recipe that builds against ENGINE_ASSETS today).
#
# This script does not bind or reference any network port.
#
set -euo pipefail

export DEBIAN_FRONTEND=noninteractive

sudo apt-get update

# Verbatim package list from .github/workflows/codeql-analysis.yml, plus:
#   - ccache (BUILD-03: root configure.ac has no Linux ccache auto-detect;
#     wired externally via CC/CXX in scripts/eden/build.sh)
#   - autoconf automake libtool pkg-config: defensive inclusion. Believed
#     preinstalled on ubuntu-latest (MEDIUM confidence per research Open
#     Question 2) — a no-op when already present, cheap insurance when not.
sudo apt-get install -y \
  libunwind-dev build-essential git libcap-dev python3-polib npm \
  libpng-dev python3-lxml libpam-dev libzstd-dev libssl-dev libcppunit-dev \
  wget cmake ccache autoconf automake libtool pkg-config

# Source-built POCO 1.12.5p2 — idempotent guard so re-running this script
# (e.g. in a warm CI cache or a re-provisioned dev box) does not rebuild
# POCO every time. The guard checks BOTH Net/WebSocket.h and
# Zip/Decompress.h: a POCO installed by the older recipe (which omitted
# Zip) must be rebuilt, not skipped.
if [ ! -f /usr/local/include/Poco/Net/WebSocket.h ] || [ ! -f /usr/local/include/Poco/Zip/Decompress.h ]; then
  POCO_BUILD_DIR="$(mktemp -d)"
  (
    cd "$POCO_BUILD_DIR"
    wget --no-verbose https://pocoproject.org/releases/poco-1.12.5p2/poco-1.12.5p2-all.tar.gz
    tar xf poco-1.12.5p2-all.tar.gz
    cd poco-1.12.5p2-all
    # Recipe from .github/workflows/codeql-analysis.yml (lines 56-61) with
    # ONE divergence: Zip is NOT omitted. Upstream main's wsd/Unzip.cpp
    # includes Poco/Zip/Decompress.h, so omitting Zip fails the build at
    # wsd/Unzip.o (proven in CI run 28906749089, 2026-07-07) — the codeql
    # omit list is stale for this tree. All other omitted modules were
    # cross-checked against the compiled dirs (wsd/ common/ kit/ net/
    # tools/) and none of their headers are included.
    ./configure --static --no-tests --no-samples --no-sharedlibs \
      --cflags="-fPIC" \
      --omit=Data,Data/SQLite,Data/ODBC,Data/MySQL,MongoDB,PDF,CppParser,PageCompiler,Redis,Encodings,ActiveRecord,Prometheus,JWT
    make -j"$(nproc)"
    sudo make install
  )
  rm -rf "$POCO_BUILD_DIR"
else
  echo "System POCO already present (Net/WebSocket.h + Zip/Decompress.h) — skipping build."
fi

echo "build-deps.sh complete: apt packages + system POCO 1.12.5p2 installed."
