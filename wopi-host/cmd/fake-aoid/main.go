/*
 * Copyright 2026 AO Cyber Systems
 *
 * SPDX-License-Identifier: MPL-2.0
 *
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at http://mozilla.org/MPL/2.0/.
 */

// Command fake-aoid serves a standalone, AOID-shaped OIDC issuer for local
// development and TRD 03-04's e2e script (wopi-host/scripts/wopi-e2e.sh).
// It is the exact same handler the in-process oidctest package mounts on
// httptest.Server in wopi-host's unit tests — production oidcauth code
// paths are identical for fake and real AOID.
package main

import (
	"log"
	"net"
	"os"
	"strings"

	"net/http"

	"github.com/AO-Cyber-Systems/EdenDocs/wopi-host/internal/oidctest"
)

// defaultAddr is 127.0.0.1:8092 — never 8080 (banned project-wide), never
// 8091 (the WOPI host itself), never 9980/9981 (coolwsd).
const defaultAddr = "127.0.0.1:8092"

func main() {
	addr := os.Getenv("EDENDOCS_FAKE_AOID_ADDR")
	if addr == "" {
		addr = defaultAddr
	}
	if strings.Contains(addr, ":8080") {
		log.Fatal("fake-aoid: port 8080 is banned project-wide; use 8092")
	}

	ln, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("fake-aoid: listen %s: %v", addr, err)
	}

	idp := oidctest.New()
	idp.SetIssuer("http://" + ln.Addr().String())

	log.Printf("fake-aoid: serving AOID-shaped OIDC issuer at %s", idp.Issuer())
	log.Printf("fake-aoid: canned identity sub=%s name=%q email=%s", oidctest.DefaultIdentity().Subject, oidctest.DefaultIdentity().Name, oidctest.DefaultIdentity().Email)

	srv := &http.Server{Handler: idp.Handler()}
	if err := srv.Serve(ln); err != nil && err != http.ErrServerClosed {
		log.Fatalf("fake-aoid: serve: %v", err)
	}
}
