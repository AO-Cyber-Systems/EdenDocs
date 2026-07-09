/*
 * Copyright 2026 AO Cyber Systems
 *
 * SPDX-License-Identifier: MPL-2.0
 *
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at http://mozilla.org/MPL/2.0/.
 */

// Package config loads the wopi-host reference deployment's configuration
// from environment variables. Every setting has an EDENDOCS_WOPI_-prefixed
// env var and, where sensible, a safe local-dev default.
package config

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds the fully resolved wopi-host runtime configuration.
type Config struct {
	// ListenAddr is the address the HTTP server binds. Defaults to
	// 127.0.0.1:8091. Port 8080 is permanently banned project-wide — see
	// validate().
	ListenAddr string

	// PublicURL is the origin the BROWSER uses to reach this host (also the
	// PostMessageOrigin value for the editor iframe).
	PublicURL string

	// WopiBaseURL is the origin COOLWSD uses to call back into this host.
	// Differs from PublicURL in docker mode (e.g.
	// http://host.docker.internal:8091), where coolwsd runs in a container
	// and can't reach 127.0.0.1 on the host.
	WopiBaseURL string

	// AOIDIssuer is the AOID OIDC issuer URL. Required starting TRD 03-02;
	// no default — an empty value here is valid at this wave (main.go only
	// serves /healthz) but the OIDC relying party will refuse to start
	// without it.
	AOIDIssuer string

	// ClientID / ClientSecret are this WOPI host's registered AOID OIDC
	// client credentials.
	ClientID     string
	ClientSecret string

	// StateHMACKey signs the OIDC `state`/session cookies. If unset, 32
	// random bytes are generated at startup (logged as a WARN) — sessions
	// then don't survive a process restart, which is acceptable reference-
	// host behavior but never appropriate for a real deployment.
	StateHMACKey []byte

	// CoolwsdURL is the origin of the coolwsd instance this host fronts.
	CoolwsdURL string

	// DataDir is the filesystem root the storage package reads/writes
	// documents from.
	DataDir string

	// WopiTokenTTL is the lifetime of a minted WOPI access_token (see
	// internal/session). Default 15 minutes.
	WopiTokenTTL time.Duration

	// RequireProof controls whether the WOPI protocol surface (TRD 03-03)
	// enforces coolwsd's X-WOPI-Proof signature verification. Default true.
	RequireProof bool
}

// bannedPort is the project-wide banned port substring. Never use it for
// any bind/serve/curl/reference — see the repo's HARD RULES.
const bannedPort = ":8080"

// FromEnv resolves a Config from the process environment, applying defaults
// documented on each Config field.
func FromEnv() (Config, error) {
	cfg := Config{
		ListenAddr:   getEnv("EDENDOCS_WOPI_LISTEN_ADDR", "127.0.0.1:8091"),
		AOIDIssuer:   os.Getenv("EDENDOCS_WOPI_AOID_ISSUER"),
		ClientID:     os.Getenv("EDENDOCS_WOPI_OIDC_CLIENT_ID"),
		ClientSecret: os.Getenv("EDENDOCS_WOPI_OIDC_CLIENT_SECRET"),
		CoolwsdURL:   getEnv("EDENDOCS_WOPI_COOLWSD_URL", "http://127.0.0.1:9980"),
		DataDir:      getEnv("EDENDOCS_WOPI_DATA_DIR", "./data"),
		RequireProof: true,
	}

	// PublicURL's default is derived from the resolved ListenAddr (not a
	// hardcoded literal) so a custom EDENDOCS_WOPI_LISTEN_ADDR is reflected
	// in the default too.
	cfg.PublicURL = getEnv("EDENDOCS_WOPI_PUBLIC_URL", "http://"+cfg.ListenAddr)
	cfg.WopiBaseURL = getEnv("EDENDOCS_WOPI_BASE_URL", cfg.PublicURL)

	ttl, err := parseDuration("EDENDOCS_WOPI_TOKEN_TTL", 15*time.Minute)
	if err != nil {
		return Config{}, err
	}
	cfg.WopiTokenTTL = ttl

	if v := os.Getenv("EDENDOCS_WOPI_REQUIRE_PROOF"); v != "" {
		b, err := strconv.ParseBool(v)
		if err != nil {
			return Config{}, fmt.Errorf("config: invalid EDENDOCS_WOPI_REQUIRE_PROOF %q: %w", v, err)
		}
		cfg.RequireProof = b
	}

	key, err := resolveStateKey()
	if err != nil {
		return Config{}, err
	}
	cfg.StateHMACKey = key

	if err := cfg.validate(); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

// validate enforces invariants that must hold across every source of
// configuration (env, defaults, or future config-file support).
func (c Config) validate() error {
	if strings.Contains(c.ListenAddr, bannedPort) {
		return errors.New("config: port 8080 is banned project-wide; use 8091")
	}
	return nil
}

// Redacted returns a copy of c with secret fields replaced by a fixed
// placeholder, safe to log at startup.
func (c Config) Redacted() Config {
	redacted := c
	if redacted.ClientSecret != "" {
		redacted.ClientSecret = "[REDACTED]"
	}
	if len(redacted.StateHMACKey) > 0 {
		redacted.StateHMACKey = []byte("[REDACTED]")
	}
	return redacted
}

// String implements fmt.Stringer with secrets ALWAYS redacted, so any
// %v/%+v print of a Config — including an accidental one — never leaks
// ClientSecret or StateHMACKey.
func (c Config) String() string {
	r := c.Redacted()
	return fmt.Sprintf(
		"{ListenAddr:%s PublicURL:%s WopiBaseURL:%s AOIDIssuer:%s ClientID:%s ClientSecret:%s StateHMACKey:%s CoolwsdURL:%s DataDir:%s WopiTokenTTL:%s RequireProof:%t}",
		r.ListenAddr, r.PublicURL, r.WopiBaseURL, r.AOIDIssuer, r.ClientID,
		r.ClientSecret, string(r.StateHMACKey), r.CoolwsdURL, r.DataDir,
		r.WopiTokenTTL, r.RequireProof,
	)
}

func resolveStateKey() ([]byte, error) {
	if v := os.Getenv("EDENDOCS_WOPI_STATE_KEY"); v != "" {
		key, err := base64.StdEncoding.DecodeString(v)
		if err != nil {
			return nil, fmt.Errorf("config: invalid EDENDOCS_WOPI_STATE_KEY (must be base64): %w", err)
		}
		return key, nil
	}

	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return nil, fmt.Errorf("config: generating random state key: %w", err)
	}
	log.Println("WARN: EDENDOCS_WOPI_STATE_KEY not set; generated an ephemeral key — sessions will not survive a process restart")
	return key, nil
}

func getEnv(name, def string) string {
	if v := os.Getenv(name); v != "" {
		return v
	}
	return def
}

func parseDuration(name string, def time.Duration) (time.Duration, error) {
	v := os.Getenv(name)
	if v == "" {
		return def, nil
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return 0, fmt.Errorf("config: invalid %s %q: %w", name, v, err)
	}
	return d, nil
}
