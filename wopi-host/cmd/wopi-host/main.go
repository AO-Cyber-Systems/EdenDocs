/*
 * Copyright 2026 AO Cyber Systems
 *
 * SPDX-License-Identifier: MPL-2.0
 *
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at http://mozilla.org/MPL/2.0/.
 */

// Command wopi-host runs the EdenDocs reference WOPI host: an AOID OIDC
// relying party that authenticates users, then serves the WOPI protocol
// surface (CheckFileInfo/GetFile/PutFile) coolwsd needs to open documents on
// their behalf. See .planning/objectives/03-aoid-authentication-integration-oidc
// for the full design.
//
// NOTE: full route wiring (OIDC login, WOPI handlers, launch page) lands in
// TRD 03-04. This wave only proves the module builds, config loads, and the
// process listens with a /healthz endpoint.
package main

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/AO-Cyber-Systems/EdenDocs/wopi-host/internal/config"
)

func main() {
	cfg, err := config.FromEnv()
	if err != nil {
		log.Fatalf("wopi-host: config error: %v", err)
	}

	log.Printf("wopi-host: starting with config: %+v", cfg.Redacted())

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", handleHealthz)

	log.Printf("wopi-host: listening on %s", cfg.ListenAddr)
	if err := http.ListenAndServe(cfg.ListenAddr, mux); err != nil {
		log.Fatalf("wopi-host: server error: %v", err)
	}
}

func handleHealthz(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}
