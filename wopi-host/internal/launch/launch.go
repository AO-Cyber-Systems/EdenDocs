/*
 * Copyright 2026 AO Cyber Systems
 *
 * SPDX-License-Identifier: MPL-2.0
 *
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at http://mozilla.org/MPL/2.0/.
 */

// Package launch serves the browser-facing document list and the launch
// page that embeds the editor iframe. The launch page mints a short-lived
// WOPI access_token and hands it to coolwsd's cool.html via a hidden
// auto-submitting POST form — NEVER a GET query string (3-RESEARCH.md
// Pitfall 6: a query-string token leaks into browser history, proxy access
// logs, and cross-origin Referer headers). coolwsd's FileServer.cpp
// UserRequestVars reads the same field names from either method; POST is
// the deliberate convention here.
//
// Identity extraction is an INJECTED dependency (identityFrom): this
// package must not import the OIDC relying-party package (TRD 03-02's),
// keeping wave-2 TRDs 03-02 and 03-03 file-disjoint. TRD 03-04 wires that
// package's request-context identity helper in.
package launch

import (
	"errors"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/AO-Cyber-Systems/EdenDocs/wopi-host/internal/coolwsd"
	"github.com/AO-Cyber-Systems/EdenDocs/wopi-host/internal/session"
	"github.com/AO-Cyber-Systems/EdenDocs/wopi-host/internal/storage"
)

// Config carries the launch handler's two wiring knobs.
type Config struct {
	// WopiBaseURL is the origin coolwsd uses to call back into this host —
	// it becomes the WOPISrc the editor iframe is pointed at. It differs
	// from the browser-facing public URL in docker mode
	// (host.docker.internal vs 127.0.0.1).
	WopiBaseURL string
	// TokenTTL is the WOPI access_token lifetime minted per launch.
	TokenTTL time.Duration
}

type handler struct {
	sessions     *session.Store
	store        *storage.Store
	disco        *coolwsd.Client
	cfg          Config
	identityFrom func(*http.Request) (session.Identity, bool)
}

// New builds the launch handler. identityFrom resolves the authenticated
// browser identity from a request (TRD 03-04 injects the OIDC RP package's
// context helper; tests inject a stub) — returning false means
// unauthenticated and triggers a redirect to /login?return=<original URL>.
func New(sessions *session.Store, store *storage.Store, disco *coolwsd.Client,
	cfg Config, identityFrom func(*http.Request) (session.Identity, bool)) http.Handler {

	h := &handler{
		sessions:     sessions,
		store:        store,
		disco:        disco,
		cfg:          cfg,
		identityFrom: identityFrom,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", h.index)
	mux.HandleFunc("GET /open", h.open)
	return mux
}

func (h *handler) requireLogin(w http.ResponseWriter, r *http.Request) (session.Identity, bool) {
	id, ok := h.identityFrom(r)
	if !ok {
		http.Redirect(w, r, "/login?return="+url.QueryEscape(r.URL.RequestURI()), http.StatusFound)
		return session.Identity{}, false
	}
	return id, true
}

var indexTmpl = template.Must(template.New("index").Parse(`<!DOCTYPE html>
<html lang="en">
<head><meta charset="utf-8"><title>EdenDocs</title></head>
<body>
<h1>EdenDocs — Documents</h1>
<p>Signed in as {{.Name}}</p>
<ul>
{{range .Files}}  <li><a href="/open?file={{.ID}}">{{.ID}}</a> ({{.Size}} bytes)</li>
{{end}}</ul>
</body>
</html>
`))

func (h *handler) index(w http.ResponseWriter, r *http.Request) {
	id, ok := h.requireLogin(w, r)
	if !ok {
		return
	}

	files, err := h.store.List()
	if err != nil {
		log.Printf("launch: list documents: %v", err)
		http.Error(w, "failed to list documents", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := indexTmpl.Execute(w, struct {
		Name  string
		Files []storage.FileInfo
	}{Name: id.Name, Files: files}); err != nil {
		log.Printf("launch: render index: %v", err)
	}
}

// launchTmpl renders the editor embed: a hidden POST form carrying
// access_token + access_token_ttl into the iframe. The token appears ONLY
// as a hidden form input value — never in any href/src/GET URL (Pitfall 6;
// regression-guarded by TestOpen_TokenNeverInGetURL).
var launchTmpl = template.Must(template.New("launch").Parse(`<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<title>{{.FileID}} — EdenDocs</title>
<style>
  html, body { margin: 0; padding: 0; height: 100%; }
  iframe { border: none; width: 100%; height: 100%; }
</style>
</head>
<body>
<form method="post" action="{{.Action}}" target="editor">
  <input type="hidden" name="access_token" value="{{.AccessToken}}">
  <input type="hidden" name="access_token_ttl" value="{{.AccessTokenTTL}}">
</form>
<iframe name="editor" title="EdenDocs editor" allow="clipboard-read *; clipboard-write *"></iframe>
<script>document.forms[0].submit()</script>
</body>
</html>
`))

func (h *handler) open(w http.ResponseWriter, r *http.Request) {
	id, ok := h.requireLogin(w, r)
	if !ok {
		return
	}

	fileID := r.URL.Query().Get("file")
	if fileID == "" {
		http.Error(w, "missing required ?file= parameter", http.StatusBadRequest)
		return
	}

	if _, err := h.store.Stat(fileID); err != nil {
		if errors.Is(err, storage.ErrNotFound) || errors.Is(err, storage.ErrBadFileID) {
			http.Error(w, "document not found", http.StatusNotFound)
			return
		}
		log.Printf("launch: stat %q: %v", fileID, err)
		http.Error(w, "failed to stat document", http.StatusInternalServerError)
		return
	}

	ext := strings.TrimPrefix(path.Ext(fileID), ".")
	urlsrc, err := h.disco.URLSrc(r.Context(), ext)
	if err != nil {
		if errors.Is(err, coolwsd.ErrUnknownExtension) {
			http.Error(w, fmt.Sprintf("no editor action for .%s files", ext), http.StatusUnsupportedMediaType)
			return
		}
		log.Printf("launch: discovery urlsrc for %q: %v", ext, err)
		http.Error(w, "editor discovery unavailable", http.StatusBadGateway)
		return
	}

	// Mint the short-lived WOPI token at launch-render time. wopiSrc is the
	// CheckFileInfo URL coolwsd will call back on; access_token_ttl is the
	// session expiry in Unix MILLISECONDS (MS-WOPI ms-epoch convention).
	token, sess := h.sessions.MintWopiToken(id, fileID, h.cfg.TokenTTL)
	wopiSrc := h.cfg.WopiBaseURL + "/wopi/files/" + fileID
	action := urlsrc + "WOPISrc=" + url.QueryEscape(wopiSrc)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	err = launchTmpl.Execute(w, struct {
		FileID         string
		Action         string
		AccessToken    string
		AccessTokenTTL int64
	}{
		FileID:         fileID,
		Action:         action,
		AccessToken:    token,
		AccessTokenTTL: sess.ExpiresAt.UnixMilli(),
	})
	if err != nil {
		log.Printf("launch: render launch page: %v", err)
	}
}
