/*
 * Copyright 2026 AO Cyber Systems
 *
 * SPDX-License-Identifier: MPL-2.0
 *
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at http://mozilla.org/MPL/2.0/.
 */

package wopi

import (
	"bytes"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/AO-Cyber-Systems/EdenDocs/wopi-host/internal/coolwsd"
	"github.com/AO-Cyber-Systems/EdenDocs/wopi-host/internal/session"
	"github.com/AO-Cyber-Systems/EdenDocs/wopi-host/internal/storage"
)

const ticksToUnixEpoch = 621355968000000000

// signProof mirrors coolwsd's wsd/ProofKey.cpp SignProof byte construction
// — the same signer the proof package's own tests use, applied here to
// shape whole HTTP requests exactly as coolwsd would send them.
func signProof(t *testing.T, priv *rsa.PrivateKey, accessToken, uri string, ticks int64) string {
	t.Helper()
	buf := new(bytes.Buffer)
	write := func(b []byte) {
		if err := binary.Write(buf, binary.BigEndian, int32(len(b))); err != nil {
			t.Fatalf("write length prefix: %v", err)
		}
		buf.Write(b)
	}
	write([]byte(accessToken))
	write([]byte(strings.ToUpper(uri)))
	tb := make([]byte, 8)
	binary.BigEndian.PutUint64(tb, uint64(ticks))
	write(tb)

	h := sha256.Sum256(buf.Bytes())
	sig, err := rsa.SignPKCS1v15(rand.Reader, priv, crypto.SHA256, h[:])
	if err != nil {
		t.Fatalf("sign proof: %v", err)
	}
	return base64.StdEncoding.EncodeToString(sig)
}

func nowTicks() int64 {
	return ticksToUnixEpoch + time.Now().UnixNano()/100
}

// discoveryXMLForKey renders a minimal discovery document whose <proof-key>
// element carries pub's modulus/exponent — the same discovery shape coolwsd
// serves, so the wopi handler under test learns the verification key
// exactly the way production does (via the coolwsd discovery client).
func discoveryXMLForKey(pub *rsa.PublicKey) string {
	mod := base64.StdEncoding.EncodeToString(pub.N.Bytes())
	exp := base64.StdEncoding.EncodeToString(bigEndianExponent(pub.E))
	return fmt.Sprintf(`<wopi-discovery>
    <net-zone name="external-http">
        <app name="writer">
            <action default="true" ext="odt" name="edit" urlsrc="http://127.0.0.1:9980/browser/abc123/cool.html?"/>
        </app>
    </net-zone>
<proof-key exponent=%q modulus=%q oldexponent=%q oldmodulus=%q/></wopi-discovery>`, exp, mod, exp, mod)
}

func bigEndianExponent(e int) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, uint64(e))
	i := 0
	for i < 7 && b[i] == 0 {
		i++
	}
	return b[i:]
}

// env is one fully-wired WOPI handler test environment.
type env struct {
	handler     http.Handler
	sessions    *session.Store
	store       *storage.Store
	signer      *rsa.PrivateKey
	wopiBaseURL string
	now         *atomic.Int64 // unix-nanos injected clock for session TTL
}

func newEnv(t *testing.T) *env {
	t.Helper()

	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate signer key: %v", err)
	}

	discoSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/xml")
		_, _ = w.Write([]byte(discoveryXMLForKey(&priv.PublicKey)))
	}))
	t.Cleanup(discoSrv.Close)
	disco := coolwsd.New(discoSrv.URL, time.Hour)

	now := &atomic.Int64{}
	now.Store(time.Now().UnixNano())
	sessions := session.NewStore(func() time.Time { return time.Unix(0, now.Load()) })

	dataDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dataDir, "hello.odt"), []byte("hello odt bytes"), 0o644); err != nil {
		t.Fatalf("seed data dir: %v", err)
	}
	store, err := storage.New(dataDir)
	if err != nil {
		t.Fatalf("storage.New: %v", err)
	}

	wopiBaseURL := "http://127.0.0.1:8091"
	handler := New(sessions, store, disco, wopiBaseURL, "http://127.0.0.1:8091", true, 20*time.Minute)

	return &env{
		handler:     handler,
		sessions:    sessions,
		store:       store,
		signer:      priv,
		wopiBaseURL: wopiBaseURL,
		now:         now,
	}
}

func (e *env) mintToken(t *testing.T, fileID string) string {
	t.Helper()
	tok, _ := e.sessions.MintWopiToken(session.Identity{
		Subject: "sub-uuid-1234",
		Name:    "Ada Lovelace",
		Email:   "ada@example.test",
	}, fileID, 15*time.Minute)
	return tok
}

// signedRequest builds an HTTP request shaped exactly as coolwsd would
// send it: access_token in the query string, X-WOPI-TimeStamp in .NET
// ticks, X-WOPI-Proof/X-WOPI-ProofOld signatures over the FULL absolute
// URL (wopiBaseURL + RequestURI, uppercased) per ProofKey.cpp.
func (e *env) signedRequest(t *testing.T, method, pathAndQuery string, body []byte) *http.Request {
	t.Helper()
	req := httptest.NewRequest(method, pathAndQuery, bytes.NewReader(body))

	u, err := req.URL.Parse(pathAndQuery)
	if err != nil {
		t.Fatalf("parse request URI: %v", err)
	}
	token := u.Query().Get("access_token")

	fullURL := e.wopiBaseURL + req.URL.RequestURI()
	ticks := nowTicks()
	proofB64 := signProof(t, e.signer, token, fullURL, ticks)
	req.Header.Set("X-WOPI-TimeStamp", strconv.FormatInt(ticks, 10))
	req.Header.Set("X-WOPI-Proof", proofB64)
	req.Header.Set("X-WOPI-ProofOld", proofB64)
	return req
}

// --- test-list 1: CheckFileInfo golden contract ---

func TestCheckFileInfo_GoldenContract(t *testing.T) {
	e := newEnv(t)
	tok := e.mintToken(t, "hello.odt")

	req := e.signedRequest(t, http.MethodGet, "/wopi/files/hello.odt?access_token="+tok, nil)
	rec := httptest.NewRecorder()
	e.handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("CheckFileInfo status = %d, want 200; body: %s", rec.Code, rec.Body.String())
	}

	raw := rec.Body.String()
	if strings.Contains(raw, `"UserExtraInfo"`) {
		t.Fatalf("CheckFileInfo must OMIT UserExtraInfo entirely (Pitfall 5); body: %s", raw)
	}

	var got map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal CheckFileInfo: %v", err)
	}

	for _, key := range []string{"BaseFileName", "Size", "OwnerId", "UserId", "UserFriendlyName", "UserCanWrite", "IsAdminUser", "PostMessageOrigin"} {
		if _, ok := got[key]; !ok {
			t.Errorf("CheckFileInfo missing required key %q", key)
		}
	}

	if got["BaseFileName"] != "hello.odt" {
		t.Errorf("BaseFileName = %v, want hello.odt", got["BaseFileName"])
	}
	if got["Size"] != float64(len("hello odt bytes")) {
		t.Errorf("Size = %v, want %d", got["Size"], len("hello odt bytes"))
	}
	if got["OwnerId"] != "edendocs" {
		t.Errorf("OwnerId = %v, want edendocs", got["OwnerId"])
	}
	if got["UserId"] != "sub-uuid-1234" {
		t.Errorf("UserId = %v, want the identity's AOID sub", got["UserId"])
	}
	if got["UserFriendlyName"] != "Ada Lovelace" {
		t.Errorf("UserFriendlyName = %v, want the identity name", got["UserFriendlyName"])
	}
	if got["UserCanWrite"] != true {
		t.Errorf("UserCanWrite = %v, want true", got["UserCanWrite"])
	}
	if got["IsAdminUser"] != false {
		t.Errorf("IsAdminUser = %v, want top-level false", got["IsAdminUser"])
	}
	if got["PostMessageOrigin"] != "http://127.0.0.1:8091" {
		t.Errorf("PostMessageOrigin = %v, want the public URL", got["PostMessageOrigin"])
	}
}

// --- test-list 2: GetFile byte-identical ---

func TestGetFile_ByteIdentical(t *testing.T) {
	e := newEnv(t)
	tok := e.mintToken(t, "hello.odt")

	req := e.signedRequest(t, http.MethodGet, "/wopi/files/hello.odt/contents?access_token="+tok, nil)
	rec := httptest.NewRecorder()
	e.handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("GetFile status = %d, want 200; body: %s", rec.Code, rec.Body.String())
	}
	if rec.Body.String() != "hello odt bytes" {
		t.Fatalf("GetFile body = %q, want the stored bytes", rec.Body.String())
	}
	if cl := rec.Header().Get("Content-Length"); cl != strconv.Itoa(len("hello odt bytes")) {
		t.Fatalf("Content-Length = %q, want %d", cl, len("hello odt bytes"))
	}
}

// --- test-list 3: PutFile persists ---

func TestPutFile_Persists(t *testing.T) {
	e := newEnv(t)
	tok := e.mintToken(t, "hello.odt")
	newBody := []byte("updated document bytes")

	req := e.signedRequest(t, http.MethodPost, "/wopi/files/hello.odt/contents?access_token="+tok, newBody)
	req.Header.Set("X-WOPI-Override", "PUT")
	rec := httptest.NewRecorder()
	e.handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("PutFile status = %d, want 200; body: %s", rec.Code, rec.Body.String())
	}

	data, _, err := e.store.Read("hello.odt")
	if err != nil {
		t.Fatalf("read back after PutFile: %v", err)
	}
	if !bytes.Equal(data, newBody) {
		t.Fatalf("stored bytes = %q, want %q", data, newBody)
	}
}

// --- test-list 4: unknown/expired token -> 401 on all three routes ---

func TestUnknownAndExpiredToken_401(t *testing.T) {
	e := newEnv(t)

	routes := []struct {
		method, path string
	}{
		{http.MethodGet, "/wopi/files/hello.odt"},
		{http.MethodGet, "/wopi/files/hello.odt/contents"},
		{http.MethodPost, "/wopi/files/hello.odt/contents"},
	}

	for _, r := range routes {
		req := e.signedRequest(t, r.method, r.path+"?access_token=totally-unknown", nil)
		rec := httptest.NewRecorder()
		e.handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("%s %s with unknown token: status = %d, want 401", r.method, r.path, rec.Code)
		}
	}

	// Expired: mint, then advance the injected clock past the TTL.
	tok := e.mintToken(t, "hello.odt")
	e.now.Add(int64(16 * time.Minute)) // TTL was 15m
	for _, r := range routes {
		req := e.signedRequest(t, r.method, r.path+"?access_token="+tok, nil)
		rec := httptest.NewRecorder()
		e.handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("%s %s with expired token: status = %d, want 401", r.method, r.path, rec.Code)
		}
	}
}

// --- test-list 5: missing proof headers rejected ---

func TestMissingProofHeaders_Rejected(t *testing.T) {
	e := newEnv(t)
	tok := e.mintToken(t, "hello.odt")

	req := httptest.NewRequest(http.MethodGet, "/wopi/files/hello.odt?access_token="+tok, nil)
	rec := httptest.NewRecorder()
	e.handler.ServeHTTP(rec, req)

	if rec.Code == http.StatusOK {
		t.Fatalf("request without proof headers must be rejected, got 200")
	}
	if rec.Code != http.StatusUnauthorized && rec.Code != http.StatusInternalServerError {
		t.Fatalf("missing proof rejection status = %d, want 401/500-class", rec.Code)
	}
}

// --- test-list 6: invalid signature (different key) rejected ---

func TestInvalidProofSignature_Rejected(t *testing.T) {
	e := newEnv(t)
	tok := e.mintToken(t, "hello.odt")

	otherKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate other key: %v", err)
	}

	pathAndQuery := "/wopi/files/hello.odt?access_token=" + tok
	req := httptest.NewRequest(http.MethodGet, pathAndQuery, nil)
	fullURL := e.wopiBaseURL + req.URL.RequestURI()
	ticks := nowTicks()
	wrongProof := signProof(t, otherKey, tok, fullURL, ticks)
	req.Header.Set("X-WOPI-TimeStamp", strconv.FormatInt(ticks, 10))
	req.Header.Set("X-WOPI-Proof", wrongProof)
	req.Header.Set("X-WOPI-ProofOld", wrongProof)

	rec := httptest.NewRecorder()
	e.handler.ServeHTTP(rec, req)
	if rec.Code == http.StatusOK {
		t.Fatalf("request signed with a different key must be rejected, got 200")
	}
}

// --- test-list 7: stale timestamp rejected ---

func TestStaleTimestamp_Rejected(t *testing.T) {
	e := newEnv(t)
	tok := e.mintToken(t, "hello.odt")

	pathAndQuery := "/wopi/files/hello.odt?access_token=" + tok
	req := httptest.NewRequest(http.MethodGet, pathAndQuery, nil)
	fullURL := e.wopiBaseURL + req.URL.RequestURI()
	staleTicks := nowTicks() - int64(30*time.Minute/(100*time.Nanosecond)) // outside 20m window
	proofB64 := signProof(t, e.signer, tok, fullURL, staleTicks)
	req.Header.Set("X-WOPI-TimeStamp", strconv.FormatInt(staleTicks, 10))
	req.Header.Set("X-WOPI-Proof", proofB64)
	req.Header.Set("X-WOPI-ProofOld", proofB64)

	rec := httptest.NewRecorder()
	e.handler.ServeHTTP(rec, req)
	if rec.Code == http.StatusOK {
		t.Fatalf("request with stale X-WOPI-TimeStamp must be rejected, got 200")
	}
}

// --- test-list 8: valid ProofOld + garbage Proof accepted (rotation) ---

func TestProofOldFallback_Accepted(t *testing.T) {
	e := newEnv(t)
	tok := e.mintToken(t, "hello.odt")

	pathAndQuery := "/wopi/files/hello.odt?access_token=" + tok
	req := httptest.NewRequest(http.MethodGet, pathAndQuery, nil)
	fullURL := e.wopiBaseURL + req.URL.RequestURI()
	ticks := nowTicks()
	validProof := signProof(t, e.signer, tok, fullURL, ticks)
	req.Header.Set("X-WOPI-TimeStamp", strconv.FormatInt(ticks, 10))
	req.Header.Set("X-WOPI-Proof", base64.StdEncoding.EncodeToString([]byte("garbage-not-a-signature")))
	req.Header.Set("X-WOPI-ProofOld", validProof)

	rec := httptest.NewRecorder()
	e.handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("valid X-WOPI-ProofOld with garbage X-WOPI-Proof must be accepted (rotation tolerance), got %d: %s", rec.Code, rec.Body.String())
	}
}

// --- test-list 9: path-traversal fileID -> 404 ---

func TestPathTraversalFileID_404(t *testing.T) {
	e := newEnv(t)
	tok := e.mintToken(t, "..%2F..%2Fetc%2Fpasswd")

	req := e.signedRequest(t, http.MethodGet, "/wopi/files/..%2F..%2Fetc%2Fpasswd?access_token="+tok, nil)
	rec := httptest.NewRecorder()
	e.handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("path-traversal fileID status = %d, want 404", rec.Code)
	}
}
