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
// Route map:
//
//	GET  /healthz               liveness probe (no auth)
//	GET  /login                 starts the AOID OIDC Authorization Code+PKCE flow
//	GET  /callback               AOID redirect target; mints the browser session
//	GET  /logout                 clears the browser session
//	GET  /                       document list (requires browser login)
//	GET  /open                   launch page, mints a WOPI access_token (requires login)
//	GET  /wopi/files/{id}         CheckFileInfo (requires a valid WOPI access_token)
//	GET  /wopi/files/{id}/contents  GetFile
//	POST /wopi/files/{id}/contents  PutFile
package main

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/AO-Cyber-Systems/EdenDocs/wopi-host/internal/config"
	"github.com/AO-Cyber-Systems/EdenDocs/wopi-host/internal/coolwsd"
	"github.com/AO-Cyber-Systems/EdenDocs/wopi-host/internal/launch"
	"github.com/AO-Cyber-Systems/EdenDocs/wopi-host/internal/oidcauth"
	"github.com/AO-Cyber-Systems/EdenDocs/wopi-host/internal/session"
	"github.com/AO-Cyber-Systems/EdenDocs/wopi-host/internal/storage"
	"github.com/AO-Cyber-Systems/EdenDocs/wopi-host/internal/wopi"
)

// discoveryCacheTTL is the production default for the coolwsd discovery
// client (internal/coolwsd.New) — discovery only changes on a coolwsd
// upgrade/restart.
const discoveryCacheTTL = 1 * time.Hour

// proofWindow bounds accepted X-WOPI-TimeStamp drift — MS-WOPI convention
// (03-03's launch.Config/wopi.New wiring note).
const proofWindow = 20 * time.Minute

// discoveryStartupTimeout bounds ONLY the OIDC issuer discovery round trip
// performed once at startup by oidcauth.NewAuthenticator.
const discoveryStartupTimeout = 30 * time.Second

func main() {
	cfg, err := config.FromEnv()
	if err != nil {
		log.Fatalf("wopi-host: config error: %v", err)
	}

	configureLogging()

	log.Printf("wopi-host: starting with config: %+v", cfg.Redacted())

	sessions := session.NewStore(time.Now)

	store, err := storage.New(cfg.DataDir)
	if err != nil {
		log.Fatalf("wopi-host: storage init: %v", err)
	}

	disco := coolwsd.New(cfg.CoolwsdURL, discoveryCacheTTL)

	authCtx, cancel := context.WithTimeout(context.Background(), discoveryStartupTimeout)
	authr, err := oidcauth.NewAuthenticator(authCtx, cfg, sessions)
	cancel()
	if err != nil {
		log.Fatalf("wopi-host: oidcauth init: %v", err)
	}

	wopiHandler := wopi.New(sessions, store, disco, cfg.WopiBaseURL, cfg.PublicURL, cfg.RequireProof, proofWindow)

	// identityFrom bridges oidcauth's context-injection helper into launch's
	// decoupled identityFrom seam (03-03 deliberately kept internal/launch
	// free of any oidcauth import; this is where the wave-2 packages meet).
	identityFrom := func(r *http.Request) (session.Identity, bool) {
		return oidcauth.FromContext(r.Context())
	}
	launchHandler := launch.New(sessions, store, disco, launch.Config{
		WopiBaseURL: cfg.WopiBaseURL,
		TokenTTL:    cfg.WopiTokenTTL,
	}, identityFrom)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", handleHealthz)
	mux.HandleFunc("GET /login", authr.LoginHandler)
	mux.HandleFunc("GET /callback", authr.CallbackHandler)
	mux.HandleFunc("GET /logout", authr.LogoutHandler)
	// /wopi/ routes authenticate via the WOPI access_token query param
	// (internal/wopi's own guard) — never the browser cookie, so they are
	// deliberately NOT wrapped in authr.RequireLogin.
	mux.Handle("/wopi/", wopiHandler)
	// The document list + launch page are browser-facing and cookie-gated:
	// RequireLogin injects the identity into the request context that
	// identityFrom (above) reads back out.
	mux.Handle("/", authr.RequireLogin(launchHandler))

	srv := &http.Server{
		Addr:    cfg.ListenAddr,
		Handler: mux,
	}

	go func() {
		log.Printf("wopi-host: listening on %s", cfg.ListenAddr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("wopi-host: server error: %v", err)
		}
	}()

	waitForShutdown(srv)
}

// configureLogging always writes to stdout, and additionally to
// EDENDOCS_WOPI_LOG_FILE (append mode) when set — the e2e script
// (wopi-host/scripts/wopi-e2e.sh) greps that file for the structured
// contract lines documented in 03-03-SUMMARY.md, mirroring coolwsd's own
// /tmp/coolwsd.log convention.
func configureLogging() {
	out := io.Writer(os.Stdout)

	logFile := os.Getenv("EDENDOCS_WOPI_LOG_FILE")
	if logFile == "" {
		log.SetOutput(out)
		return
	}

	f, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		log.Fatalf("wopi-host: opening EDENDOCS_WOPI_LOG_FILE %q: %v", logFile, err)
	}
	log.SetOutput(io.MultiWriter(out, f))
}

// waitForShutdown blocks until SIGINT/SIGTERM, then gives in-flight
// requests up to 10s to drain via http.Server.Shutdown.
func waitForShutdown(srv *http.Server) {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	log.Printf("wopi-host: shutting down")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("wopi-host: shutdown error: %v", err)
	}
}

func handleHealthz(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}
