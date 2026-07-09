#!/usr/bin/env bash
# build-release.sh — NON-debug release build → docker/eden/instdir (DIST-01).
# Distinct from scripts/eden/build.sh (the debug dev/CI loop). Produces the
# COPY /instdir / build context for docker/eden/Dockerfile. Binds no network port.
# Prereqs (CI, 04-04): scripts/eden/build-deps.sh (system POCO) then
# scripts/eden/fetch-engine-pinned.sh (pinned engine) must run BEFORE this.
set -euo pipefail
cd "$(dirname "$0")/../.."

if command -v ccache >/dev/null 2>&1 && [ -z "${CC:-}" ]; then
  export CC="ccache gcc" CXX="ccache g++"
fi

./autogen.sh

# NON-debug + FHS prefixes. --with-lo-path is the FUTURE container runtime path
# (Pitfall 4: not existence-checked in non-debug). HARD OMIT LIST unchanged (BRAND-06).
./configure --prefix=/usr --sysconfdir=/etc --localstatedir=/var \
  --enable-silent-rules --disable-tests \
  --with-lokit-path="$PWD/engine/include" \
  --with-lo-path=/opt/collaboraoffice \
  --with-app-branding="$PWD/eden-branding" \
  --with-app-name="EdenDocs" \
  --with-vendor="AO Cyber Systems" \
  --with-help-url="https://aocyber.ai" \
  LDFLAGS="-L/usr/local/lib"

# BRAND-01/06 canary — brandProductName only survives when no welcome URL is set.
grep -q "<brandProductName" coolwsd.xml \
  || { echo "ERROR: brandProductName missing from coolwsd.xml — a phone-home flag leaked (Pitfall 1)"; exit 1; }

# Inject EdenDocs brand keys into the generated (gitignored) coolwsd.xml.
python3 - <<'PYEOF'
import re, pathlib
p = pathlib.Path('coolwsd.xml'); c = p.read_text()
c = re.sub(r'(<brandProductName[^>]*>)[^<]*(</brandProductName>)', r'\1EdenDocs\2', c)
c = re.sub(r'(<brandProductURL[^>]*>)[^<]*(</brandProductURL>)', r'\1https://aocyber.ai\2', c)
c = re.sub(r'(<logoURL[^>]*>)[^<]*(</logoURL>)', r'\1images/eden-logo.svg\2', c)
p.write_text(c)
PYEOF
grep -q ">EdenDocs</brandProductName>" coolwsd.xml || { echo "ERROR: brandProductName injection failed"; exit 1; }

make -j"$(nproc)" build-nocheck
make -j"$(nproc)" -C browser

INSTDIR="$PWD/docker/eden/instdir"
rm -rf "$INSTDIR"
mkdir -p "$INSTDIR"
DESTDIR="$INSTDIR" make install
mkdir -p "$INSTDIR/opt"
cp -a engine/instdir "$INSTDIR/opt/collaboraoffice"

# EdenDocs branding overlays into the DESTDIR tree (file_server_root_path=/usr/share/coolwsd
# in the container; favicon has no config override; welcome must be fully replaced).
cp eden-branding/favicon.ico "$INSTDIR/usr/share/coolwsd/favicon.ico"
rm -rf "$INSTDIR/usr/share/coolwsd/browser/dist/welcome"
cp -R eden-branding/welcome "$INSTDIR/usr/share/coolwsd/browser/dist/welcome"

# Loud context assertions — the Dockerfile's setcap + systemplate depend on these.
test -x "$INSTDIR/usr/bin/coolwsd"          || { echo "ERROR: instdir coolwsd missing"; exit 1; }
test -x "$INSTDIR/usr/bin/coolforkit-caps"  || { echo "ERROR: instdir coolforkit-caps missing"; exit 1; }
test -x "$INSTDIR/usr/bin/coolmount"        || { echo "ERROR: instdir coolmount missing"; exit 1; }
test -x "$INSTDIR/usr/bin/coolwsd-systemplate-setup" || { echo "ERROR: instdir coolwsd-systemplate-setup missing"; exit 1; }
test -d "$INSTDIR/opt/collaboraoffice/program" || { echo "ERROR: engine not staged at opt/collaboraoffice"; exit 1; }
grep -q ">EdenDocs</brandProductName>" "$INSTDIR/etc/coolwsd/coolwsd.xml" \
  || { echo "ERROR: installed coolwsd.xml missing EdenDocs brandProductName"; exit 1; }
if grep -rqi collabora "$INSTDIR/usr/share/coolwsd/browser/dist/welcome/"; then
  echo "ERROR: welcome overlay still references Collabora"; exit 1
fi

echo "build-release.sh complete: docker/eden/instdir staged (coolwsd + engine + EdenDocs branding)."
