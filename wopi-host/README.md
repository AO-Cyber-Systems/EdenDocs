<!--
Copyright 2026 AO Cyber Systems

SPDX-License-Identifier: MPL-2.0

This Source Code Form is subject to the terms of the Mozilla Public
License, v. 2.0. If a copy of the MPL was not distributed with this
file, You can obtain one at http://mozilla.org/MPL/2.0/.
-->

# wopi-host

EdenDocs' reference WOPI host: authenticates users against AOID (OIDC
Authorization Code + PKCE) and fronts `coolwsd` as a WOPI host
(CheckFileInfo/GetFile/PutFile). A standalone Go module, isolated from the
rest of this repo's build.

**Full runbook (architecture, env reference, AOID client registration,
production coolwsd trust config, v1 limitations):**
[`docs/eden/WOPI-HOST.md`](../docs/eden/WOPI-HOST.md).

## Private dependency setup (one-time)

This module depends on the private `eden-platform-go` module. Before the
first build:

```bash
git config --global url."https://<YOUR_GITHUB_TOKEN>@github.com/AO-Cyber-Systems/".insteadOf "https://github.com/aocybersystems/"
go env -w GOPRIVATE=github.com/aocybersystems/*
```

## Build

```bash
cd wopi-host
go build ./cmd/...
```

Produces `wopi-host`, `fake-aoid` (dev/CI-only fake OIDC IdP), and
`e2e-probe` (headless coolwsd WebSocket test client).

## Run (against fake-aoid, port 8091)

```bash
EDENDOCS_FAKE_AOID_ADDR=127.0.0.1:8092 ./fake-aoid &

EDENDOCS_WOPI_AOID_ISSUER=http://127.0.0.1:8092 \
EDENDOCS_WOPI_OIDC_CLIENT_ID=edendocs-e2e \
EDENDOCS_WOPI_OIDC_CLIENT_SECRET=e2e-secret \
EDENDOCS_WOPI_COOLWSD_URL=http://127.0.0.1:9980 \
./wopi-host
```

coolwsd must already be running (native port 9980) and configured to trust
this host via `storage.wopi.alias_groups` — see the runbook §5. Then visit
`http://127.0.0.1:8091/open?file=hello.odt`.

For a real AOID instance instead of fake-aoid, register the WOPI host as an
AOID OAuth client first — runbook §4 — then set
`EDENDOCS_WOPI_OIDC_CLIENT_ID`/`EDENDOCS_WOPI_OIDC_CLIENT_SECRET`/
`EDENDOCS_WOPI_AOID_ISSUER` to the real values.

## Test

```bash
cd wopi-host
go vet ./...
go test -race ./... -count=1
```

## End-to-end (headless, fake-aoid, both coolwsd modes)

```bash
# native mode (builds/runs ./coolwsd at the repo root, port 9980)
COOLWSD_MODE=native ./scripts/wopi-e2e.sh

# docker mode (ephemeral collabora/code container, host port 9981)
COOLWSD_MODE=docker ./scripts/wopi-e2e.sh
```

One deterministic script proves OIDC login -> WOPI launch -> CheckFileInfo/
GetFile/PutFile -> proof-key + `alias_groups` trust (positive AND
negative) end to end. See runbook §2 for details.

## Ports

8091 (wopi-host), 8092 (fake-aoid), 9980 (coolwsd native), 9981 (coolwsd
docker-mode e2e container). Port 8080 is permanently banned project-wide —
never use it here.
