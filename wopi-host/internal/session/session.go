/*
 * Copyright 2026 AO Cyber Systems
 *
 * SPDX-License-Identifier: MPL-2.0
 *
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at http://mozilla.org/MPL/2.0/.
 */

// Package session holds two independent in-memory session stores:
//
//   - the browser LOGIN session (cookie-carried session id -> Identity),
//     created after a successful AOID OIDC round trip;
//   - the WOPI access_token store (opaque token -> WopiSession), minted at
//     launch-page-render time and consumed by coolwsd's CheckFileInfo/
//     GetFile/PutFile callbacks.
//
// These are deliberately two unrelated credential spaces (3-RESEARCH.md
// Pitfall 2): a WOPI access_token is a fresh, random, WOPI-host-minted
// uuid.NewString() that carries zero AOID claim material — coolwsd only
// ever echoes it back, it never decodes it. Do not "simplify" this into a
// single token space; that would leak AOID session state into a credential
// that also transits browser query strings and coolwsd's WOPI callbacks.
//
// Production-scale note: both stores here are a single-process in-memory
// map guarded by sync.RWMutex, adequate for a reference deployment (a
// single wopi-host instance, short token TTLs). A production deployment
// running multiple wopi-host replicas would swap this for Redis or another
// shared store — that swap is out of scope for this objective.
package session

import (
	"sync"
	"time"

	"github.com/google/uuid"
)

// Identity is the authenticated user's identity, as resolved from an AOID
// OIDC round trip (id_token claims + fallback chain). It flows into WOPI
// CheckFileInfo's UserId/UserFriendlyName/UserExtraInfo fields (TRD 03-03).
type Identity struct {
	// Subject is the AOID sub claim (account UUID) -> CheckFileInfo UserId.
	Subject string
	// Name is the display name after AOID's fallback chain -> CheckFileInfo
	// UserFriendlyName.
	Name string
	// Email is the AOID email claim.
	Email string
	// IsAdmin is always false in v1 — see 3-RESEARCH.md Pitfall 1 (AOID v1.0
	// has no clean, generally-usable admin/role claim for an OIDC RP).
	IsAdmin bool
}

// WopiSession binds a minted WOPI access_token to the identity and file it
// authorizes access to, plus its expiry.
type WopiSession struct {
	Identity Identity
	FileID   string
	// ExpiresAt is the token's absolute expiry. TRD 03-03 computes the WOPI
	// access_token_ttl form field as ExpiresAt.UnixMilli() (MS-WOPI
	// convention: an absolute Unix-milliseconds timestamp, not a duration).
	ExpiresAt time.Time
}

// webSessionEntry pairs a stored Identity with its own expiry.
type webSessionEntry struct {
	identity  Identity
	expiresAt time.Time
}

// Store is a single-process, in-memory holder of both session spaces
// described in the package doc. The zero value is not usable — construct
// with NewStore.
type Store struct {
	now func() time.Time

	mu           sync.RWMutex
	webSessions  map[string]webSessionEntry
	wopiSessions map[string]WopiSession
}

// NewStore constructs an empty Store. now is injected (rather than calling
// time.Now directly) so tests can control expiry deterministically without
// sleeping — pass time.Now in production code.
func NewStore(now func() time.Time) *Store {
	return &Store{
		now:          now,
		webSessions:  make(map[string]webSessionEntry),
		wopiSessions: make(map[string]WopiSession),
	}
}

// CreateWebSession stores id under a fresh random session id, valid for
// ttl, and returns that session id (intended for a Set-Cookie value).
func (s *Store) CreateWebSession(id Identity, ttl time.Duration) string {
	sid := uuid.NewString()

	s.mu.Lock()
	defer s.mu.Unlock()
	s.webSessions[sid] = webSessionEntry{
		identity:  id,
		expiresAt: s.now().Add(ttl),
	}
	return sid
}

// LookupWebSession resolves sid to its Identity, provided the session
// exists and has not expired. An expired entry is lazily deleted.
func (s *Store) LookupWebSession(sid string) (Identity, bool) {
	s.mu.RLock()
	entry, ok := s.webSessions[sid]
	s.mu.RUnlock()
	if !ok {
		return Identity{}, false
	}

	if s.now().After(entry.expiresAt) {
		s.mu.Lock()
		delete(s.webSessions, sid)
		s.mu.Unlock()
		return Identity{}, false
	}

	return entry.identity, true
}

// DeleteWebSession removes sid unconditionally (used on logout).
func (s *Store) DeleteWebSession(sid string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.webSessions, sid)
}

// MintWopiToken mints a fresh opaque WOPI access_token bound to id and
// fileID, valid for ttl, and returns both the token string and the
// WopiSession record it resolves to. The token is a bare uuid.NewString():
// it carries no identity material and is not decodable as a JWT (Pitfall 2)
// — coolwsd only ever echoes it back on WOPI callbacks.
func (s *Store) MintWopiToken(id Identity, fileID string, ttl time.Duration) (string, WopiSession) {
	token := uuid.NewString()
	sess := WopiSession{
		Identity:  id,
		FileID:    fileID,
		ExpiresAt: s.now().Add(ttl),
	}

	s.mu.Lock()
	s.wopiSessions[token] = sess
	s.mu.Unlock()

	return token, sess
}

// LookupWopiToken resolves token to its WopiSession, provided it exists and
// has not expired. An expired entry is lazily deleted.
func (s *Store) LookupWopiToken(token string) (WopiSession, bool) {
	if token == "" {
		return WopiSession{}, false
	}

	s.mu.RLock()
	sess, ok := s.wopiSessions[token]
	s.mu.RUnlock()
	if !ok {
		return WopiSession{}, false
	}

	if s.now().After(sess.ExpiresAt) {
		s.mu.Lock()
		delete(s.wopiSessions, token)
		s.mu.Unlock()
		return WopiSession{}, false
	}

	return sess, true
}
