/*
 * Copyright 2026 AO Cyber Systems
 *
 * SPDX-License-Identifier: MPL-2.0
 *
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at http://mozilla.org/MPL/2.0/.
 */

// Package wopi implements the three WOPI endpoints coolwsd calls back on
// (verified against wsd/wopi/WopiStorage.cpp):
//
//	GET  /wopi/files/{id}           -> CheckFileInfo
//	GET  /wopi/files/{id}/contents  -> GetFile
//	POST /wopi/files/{id}/contents  -> PutFile (X-WOPI-Override: PUT)
//
// Every request is guarded by (1) WOPI access_token lookup against the
// session store and (2) X-WOPI-Proof/X-WOPI-ProofOld verification against
// the proof keys coolwsd publishes in /hosting/discovery — proof-key
// validation is entirely the WOPI host's job, there is no coolwsd-side
// toggle (3-RESEARCH.md Pitfall 3).
package wopi

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/AO-Cyber-Systems/EdenDocs/wopi-host/internal/coolwsd"
	"github.com/AO-Cyber-Systems/EdenDocs/wopi-host/internal/proof"
	"github.com/AO-Cyber-Systems/EdenDocs/wopi-host/internal/session"
	"github.com/AO-Cyber-Systems/EdenDocs/wopi-host/internal/storage"
)

// maxPutFileBytes caps a PutFile request body (defense against unbounded
// uploads; coolwsd saves whole documents, 100 MiB is generous for the
// reference host).
const maxPutFileBytes = 100 << 20

// CheckFileInfo is coolwsd's ACTUAL parsing contract, verified from
// wsd/wopi/WopiStorage.cpp — field names are exact and case-sensitive.
type CheckFileInfo struct {
	BaseFileName     string `json:"BaseFileName"`     // required
	Size             int64  `json:"Size"`             // required
	OwnerId          string `json:"OwnerId"`          // required — WOPI-host storage concept
	UserId           string `json:"UserId"`           // Identity.Subject (AOID sub)
	UserFriendlyName string `json:"UserFriendlyName"` // Identity.Name — never empty (coolwsd LOG_ERRs + falls back to "UnknownUser")
	UserCanWrite     bool   `json:"UserCanWrite"`
	// UserExtraInfo is OMITTED entirely in v1: AOID has no avatar claim
	// anywhere in its OIDC surface, and the browser's LOUtil.setUserImage
	// falls back gracefully to the built-in user.svg when the key is absent
	// (3-RESEARCH.md Pitfall 5). Never emit an empty/placeholder avatar URL.
	UserExtraInfo map[string]any `json:"UserExtraInfo,omitempty"`
	// IsAdminUser is TOP-LEVEL — the nested UserExtraInfo.is_admin shape is
	// deprecated and logged by coolwsd (research anti-pattern). Always false
	// in v1: AOID v1.0 has no cleanly-consumable admin claim for a generic
	// OIDC RP (3-RESEARCH.md Pitfall 1).
	IsAdminUser       bool   `json:"IsAdminUser"`
	PostMessageOrigin string `json:"PostMessageOrigin,omitempty"`
	// SupportsLocks is false as a deliberate v1 scope reduction: WOPI
	// lock-conflict semantics were left untraced (research Open Question 1).
	// coolwsd omits all X-WOPI-Lock headers when this is false. Documented
	// as a known limitation in the 03-05 runbook.
	SupportsLocks bool `json:"SupportsLocks"`
}

type handler struct {
	sessions     *session.Store
	store        *storage.Store
	disco        *coolwsd.Client
	wopiBaseURL  string
	publicURL    string
	requireProof bool
	window       time.Duration

	warnOnce sync.Once
}

// New builds the WOPI protocol handler. wopiBaseURL is the origin coolwsd
// uses to call back (it differs from publicURL in docker mode —
// host.docker.internal); the proof middleware reconstructs the FULL
// absolute request URL coolwsd signed from wopiBaseURL +
// r.URL.RequestURI(), never from r.Host (proxies/containers rewrite Host).
// publicURL becomes CheckFileInfo's PostMessageOrigin. requireProof=false
// disables proof verification (dev only; logged loudly once). window
// bounds accepted X-WOPI-TimeStamp drift (MS-WOPI convention: 20 minutes).
func New(sessions *session.Store, store *storage.Store, disco *coolwsd.Client,
	wopiBaseURL string, publicURL string, requireProof bool, window time.Duration) http.Handler {

	h := &handler{
		sessions:     sessions,
		store:        store,
		disco:        disco,
		wopiBaseURL:  wopiBaseURL,
		publicURL:    publicURL,
		requireProof: requireProof,
		window:       window,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /wopi/files/{id}", h.guard("CheckFileInfo", h.checkFileInfo))
	mux.HandleFunc("GET /wopi/files/{id}/contents", h.guard("GetFile", h.getFile))
	mux.HandleFunc("POST /wopi/files/{id}/contents", h.guard("PutFile", h.putFile))
	return mux
}

// guard is the shared middleware for every WOPI route, in the order the
// TRD specifies: (1) access_token lookup -> 401 + log; (2) proof
// verification -> reject + "proof: REJECTED" log; (3) handler.
func (h *handler) guard(route string, next func(http.ResponseWriter, *http.Request, session.WopiSession, string)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")

		// (1) access_token lookup.
		token := r.URL.Query().Get("access_token")
		sess, ok := h.sessions.LookupWopiToken(token)
		if !ok {
			h.logRoute(route, id, "", 0, http.StatusUnauthorized)
			http.Error(w, "invalid or expired access_token", http.StatusUnauthorized)
			return
		}

		// (2) proof verification.
		if h.requireProof {
			if err := h.verifyProof(r, token); err != nil {
				log.Printf("proof: REJECTED reason=%v", err)
				h.logRoute(route, id, sess.Identity.Subject, 0, http.StatusUnauthorized)
				http.Error(w, "WOPI proof verification failed", http.StatusUnauthorized)
				return
			}
			log.Printf("proof: verified ok")
		} else {
			h.warnOnce.Do(func() {
				log.Printf("WARN wopi: proof verification DISABLED (requireProof=false) — dev mode only, never run production this way")
			})
		}

		// (3) handler.
		next(w, r, sess, id)
	}
}

// verifyProof checks X-WOPI-Proof/X-WOPI-ProofOld/X-WOPI-TimeStamp against
// the proof keys published in coolwsd's discovery document, over the FULL
// absolute URL as coolwsd signed it.
func (h *handler) verifyProof(r *http.Request, token string) error {
	d, err := h.disco.Get(r.Context())
	if err != nil {
		return fmt.Errorf("discovery fetch for proof keys: %w", err)
	}
	if d.ProofKey == nil && d.OldProofKey == nil {
		return errors.New("discovery published no proof-key (is coolwsd's proof_key file present?)")
	}

	fullURL := h.wopiBaseURL + r.URL.RequestURI()
	return proof.VerifyEither(
		d.ProofKey, d.OldProofKey,
		token, fullURL,
		r.Header.Get("X-WOPI-TimeStamp"),
		r.Header.Get("X-WOPI-Proof"),
		r.Header.Get("X-WOPI-ProofOld"),
		h.window,
	)
}

// logRoute emits the structured per-request log line — the exact format is
// a CONTRACT for TRD 03-04's e2e log greps; do not reword.
func (h *handler) logRoute(route, id, sub string, bytes int64, status int) {
	switch route {
	case "CheckFileInfo":
		log.Printf("wopi: CheckFileInfo file=%s user=%s status=%d", id, sub, status)
	case "GetFile":
		log.Printf("wopi: GetFile file=%s status=%d", id, status)
	case "PutFile":
		log.Printf("wopi: PutFile file=%s bytes=%d status=%d", id, bytes, status)
	}
}

// storageStatus maps storage's typed errors to WOPI HTTP codes: unknown
// and malformed (traversal) fileIDs are both 404 — the storage guard
// surfaces before any disk access outside the data dir can happen.
func storageStatus(err error) int {
	switch {
	case errors.Is(err, storage.ErrNotFound), errors.Is(err, storage.ErrBadFileID):
		return http.StatusNotFound
	default:
		return http.StatusInternalServerError
	}
}

func (h *handler) checkFileInfo(w http.ResponseWriter, r *http.Request, sess session.WopiSession, id string) {
	info, err := h.store.Stat(id)
	if err != nil {
		status := storageStatus(err)
		h.logRoute("CheckFileInfo", id, sess.Identity.Subject, 0, status)
		http.Error(w, "file not found", status)
		return
	}

	// coolwsd LOG_ERRs and substitutes "UnknownUser" on an empty
	// UserFriendlyName — never send empty (TRD contract).
	name := sess.Identity.Name
	if name == "" {
		name = sess.Identity.Email
	}
	if name == "" {
		name = "EdenDocs User"
	}

	resp := CheckFileInfo{
		BaseFileName:      info.ID,
		Size:              info.Size,
		OwnerId:           "edendocs",
		UserId:            sess.Identity.Subject,
		UserFriendlyName:  name,
		UserCanWrite:      true,
		IsAdminUser:       sess.Identity.IsAdmin, // always false in v1 (Pitfall 1)
		PostMessageOrigin: h.publicURL,
		SupportsLocks:     false,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		h.logRoute("CheckFileInfo", id, sess.Identity.Subject, 0, http.StatusInternalServerError)
		return
	}
	h.logRoute("CheckFileInfo", id, sess.Identity.Subject, 0, http.StatusOK)
}

func (h *handler) getFile(w http.ResponseWriter, r *http.Request, sess session.WopiSession, id string) {
	data, _, err := h.store.Read(id)
	if err != nil {
		status := storageStatus(err)
		h.logRoute("GetFile", id, sess.Identity.Subject, 0, status)
		http.Error(w, "file not found", status)
		return
	}

	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Length", strconv.Itoa(len(data)))
	if _, err := w.Write(data); err != nil {
		h.logRoute("GetFile", id, sess.Identity.Subject, 0, http.StatusInternalServerError)
		return
	}
	h.logRoute("GetFile", id, sess.Identity.Subject, 0, http.StatusOK)
}

func (h *handler) putFile(w http.ResponseWriter, r *http.Request, sess session.WopiSession, id string) {
	body := http.MaxBytesReader(w, r.Body, maxPutFileBytes)
	defer body.Close()

	info, err := h.store.Write(id, body)
	if err != nil {
		status := storageStatus(err)
		h.logRoute("PutFile", id, sess.Identity.Subject, 0, status)
		// coolwsd treats any non-200 PutFile as a save failure and surfaces
		// it in the editor UI.
		http.Error(w, "save failed", status)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("{}"))
	h.logRoute("PutFile", id, sess.Identity.Subject, info.Size, http.StatusOK)
}
