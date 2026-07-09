/*
 * Copyright 2026 AO Cyber Systems
 *
 * SPDX-License-Identifier: MPL-2.0
 *
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at http://mozilla.org/MPL/2.0/.
 */

// Command e2e-probe is a headless coolwsd WebSocket client used exclusively
// by wopi-host/scripts/wopi-e2e.sh (TRD 03-04) to prove — behaviorally, not
// by config-file inspection — that coolwsd trusts (or correctly rejects) a
// given WOPISrc.
//
// It mirrors the exact handshake browser/js/global.js's makeDocAndWopiSrcUrl
// and browser/src/app/socket.ts's _onSocketOpen perform: connect the
// /cool/<enc-docurl>/ws WebSocket, send a "coolclient" hello, then a
// "load url=" command, and inspect the first server frames coolwsd sends
// back.
//
//   - A frame starting "status:" means coolwsd accepted the WOPISrc host and
//     the access_token, and completed CheckFileInfo + GetFile against this
//     WOPI host — POSITIVE trust proof.
//   - A frame starting "error:" (e.g. "error: cmd=internal kind=unauthorized")
//     means coolwsd rejected the WOPISrc host or token before ever calling
//     back — NEGATIVE trust proof, used with -expect-unauthorized.
//
// gorilla/websocket is imported ONLY here — never in a server package
// (wopi-host/go.mod gotcha).
package main

import (
	"flag"
	"fmt"
	"log"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

func main() {
	coolwsdURLFlag := flag.String("coolwsd-url", "http://127.0.0.1:9980", "coolwsd base URL (http/https; converted to ws/wss)")
	wopiSrc := flag.String("wopisrc", "", "the WOPISrc URL to open (required)")
	token := flag.String("token", "", "WOPI access_token minted by the launch page (required)")
	ttl := flag.String("ttl", "", "WOPI access_token_ttl (Unix ms epoch, from the launch page form)")
	timeout := flag.Duration("timeout", 60*time.Second, "how long to wait for the expected frame before giving up")
	save := flag.Bool("save", false, "after a positive status frame, send a forced save command and keep reading until timeout")
	expectUnauthorized := flag.Bool("expect-unauthorized", false, "expect coolwsd to reject this WOPISrc/token with an unauthorized error frame (negative trust test)")
	flag.Parse()

	if *wopiSrc == "" {
		log.Fatal("e2e-probe: -wopisrc is required")
	}
	if *token == "" {
		log.Fatal("e2e-probe: -token is required")
	}

	wsURL, docurl, err := buildWSURL(*coolwsdURLFlag, *wopiSrc, *token, *ttl)
	if err != nil {
		log.Fatalf("e2e-probe: building WS URL: %v", err)
	}
	fmt.Fprintf(os.Stderr, "e2e-probe: dialing %s\n", wsURL)

	conn, resp, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		status := "n/a"
		if resp != nil {
			status = resp.Status
		}
		log.Fatalf("e2e-probe: WS dial failed (handshake status %s): %v", status, err)
	}
	defer conn.Close()

	nowMs := time.Now().UnixMilli()
	hello := fmt.Sprintf("coolclient 0.1 %d %d", nowMs, nowMs)
	if err := conn.WriteMessage(websocket.TextMessage, []byte(hello)); err != nil {
		log.Fatalf("e2e-probe: sending coolclient hello: %v", err)
	}

	loadCmd := "load url=" + url.QueryEscape(docurl)
	if err := conn.WriteMessage(websocket.TextMessage, []byte(loadCmd)); err != nil {
		log.Fatalf("e2e-probe: sending load command: %v", err)
	}

	deadline := time.Now().Add(*timeout)

	// Phase 1: wait for the first "status:" (accepted) or "error:" (rejected)
	// frame. Any other frame type is diagnostic noise — keep waiting.
	for {
		remaining := time.Until(deadline)
		if remaining <= 0 {
			if *expectUnauthorized {
				log.Fatal("e2e-probe: FAIL — timed out waiting for an unauthorized error frame")
			}
			log.Fatal("e2e-probe: FAIL — timed out waiting for a status: frame")
		}
		if err := conn.SetReadDeadline(time.Now().Add(remaining)); err != nil {
			log.Fatalf("e2e-probe: setting read deadline: %v", err)
		}

		_, data, err := conn.ReadMessage()
		if err != nil {
			if *expectUnauthorized {
				log.Fatalf("e2e-probe: FAIL — connection error before an unauthorized frame arrived: %v", err)
			}
			log.Fatalf("e2e-probe: FAIL — connection error before a status: frame arrived: %v", err)
		}

		frame := string(data)
		preview := truncate(frame, 120)
		fmt.Fprintf(os.Stderr, "e2e-probe: recv: %s\n", preview)

		switch {
		case strings.HasPrefix(frame, "error:"):
			if *expectUnauthorized && strings.Contains(strings.ToLower(frame), "unauthorized") {
				fmt.Fprintf(os.Stderr, "e2e-probe: PASS — got expected unauthorized error frame: %s\n", preview)
				return
			}
			log.Fatalf("e2e-probe: FAIL — received error frame: %s", preview)

		case strings.HasPrefix(frame, "status:"):
			if *expectUnauthorized {
				log.Fatalf("e2e-probe: FAIL — expected unauthorized rejection but coolwsd accepted the session: %s", preview)
			}
			fmt.Fprintf(os.Stderr, "e2e-probe: PASS — got status: frame\n")
			goto accepted
		}
	}

accepted:
	if !*save {
		return
	}

	// Phase 2 (-save only): trigger a forced save, then drain frames purely
	// for diagnostics until the timeout elapses. The actual PutFile
	// confirmation is the SCRIPT's job (grepping the wopi-host log for
	// "wopi: PutFile ... status=200") — a clean disconnect or a plain
	// timeout here are both expected, not failures.
	saveCmd := "save dontTerminateEdit=1 dontSaveIfUnmodified=0"
	if err := conn.WriteMessage(websocket.TextMessage, []byte(saveCmd)); err != nil {
		log.Fatalf("e2e-probe: sending save command: %v", err)
	}
	fmt.Fprintf(os.Stderr, "e2e-probe: sent forced save command; draining frames until timeout for PutFile to land\n")

	for {
		remaining := time.Until(deadline)
		if remaining <= 0 {
			fmt.Fprintf(os.Stderr, "e2e-probe: PASS — save window elapsed\n")
			return
		}
		if err := conn.SetReadDeadline(time.Now().Add(remaining)); err != nil {
			return
		}
		_, data, err := conn.ReadMessage()
		if err != nil {
			// Connection errors/closures after save are non-fatal — the
			// script's log grep is the source of truth for PutFile.
			fmt.Fprintf(os.Stderr, "e2e-probe: connection ended after save (%v) — deferring to script log grep\n", err)
			return
		}
		fmt.Fprintf(os.Stderr, "e2e-probe: recv: %s\n", truncate(string(data), 120))
	}
}

// buildWSURL constructs coolwsd's /cool/<enc-docurl>/ws WebSocket URL,
// mirroring browser/js/global.js's makeDocAndWopiSrcUrl exactly. It returns
// both the WS URL to dial and the plain (unescaped) docurl, since the
// "load url=" command re-escapes docurl itself.
func buildWSURL(coolwsdBase, wopiSrc, token, ttl string) (wsURL string, docurl string, err error) {
	base, err := url.Parse(coolwsdBase)
	if err != nil {
		return "", "", fmt.Errorf("parsing -coolwsd-url %q: %w", coolwsdBase, err)
	}

	wsScheme := "ws"
	if base.Scheme == "https" {
		wsScheme = "wss"
	}

	docurl = wopiSrc + "?access_token=" + token + "&access_token_ttl=" + ttl + "&permission=edit"

	wsURL = wsScheme + "://" + base.Host + "/cool/" + url.QueryEscape(docurl) + "/ws" +
		"?WOPISrc=" + url.QueryEscape(wopiSrc) + "&compat=" + "/ws"

	return wsURL, docurl, nil
}

func truncate(s string, n int) string {
	if len(s) > n {
		return s[:n]
	}
	return s
}
