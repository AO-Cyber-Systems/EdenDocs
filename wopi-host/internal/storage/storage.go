/*
 * Copyright 2026 AO Cyber Systems
 *
 * SPDX-License-Identifier: MPL-2.0
 *
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at http://mozilla.org/MPL/2.0/.
 */

// Package storage is a filesystem-backed document store for the reference
// WOPI host: documents live as plain files directly under a configurable
// data directory, one file per fileID.
//
// A fileID is always treated as an exact base filename inside DataDir —
// never a nested path. This is a deliberate, load-bearing simplification:
// it makes path-traversal rejection trivial to reason about (reject any
// fileID containing a path separator or ".."), at the cost of a flat
// namespace. Sufficient for a reference deployment; documented as a
// production-scale swap-in point (e.g. S3, a real DB-backed store with a
// richer key scheme) rather than built here.
package storage

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ErrNotFound is returned when a fileID has no corresponding file. The WOPI
// protocol layer (TRD 03-03) maps this to HTTP 404.
var ErrNotFound = errors.New("storage: file not found")

// ErrBadFileID is returned when a fileID fails sanitization (path
// traversal, path separators, or empty). The WOPI protocol layer maps this
// to HTTP 400/404 as appropriate.
var ErrBadFileID = errors.New("storage: invalid file id")

// FileInfo describes a stored document.
type FileInfo struct {
	ID      string
	Size    int64
	ModTime time.Time
}

// Store is a filesystem-backed document store rooted at a single data
// directory. The zero value is not usable — construct with New.
type Store struct {
	dataDir string
}

// New constructs a Store rooted at dataDir, creating the directory
// (including parents) if it does not already exist.
func New(dataDir string) (*Store, error) {
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return nil, fmt.Errorf("storage: creating data dir %q: %w", dataDir, err)
	}
	return &Store{dataDir: dataDir}, nil
}

// List returns FileInfo for every document currently stored. Temp files
// left over from an interrupted Write (see the *.tmp convention below) are
// excluded.
func (s *Store) List() ([]FileInfo, error) {
	entries, err := os.ReadDir(s.dataDir)
	if err != nil {
		return nil, fmt.Errorf("storage: listing %q: %w", s.dataDir, err)
	}

	var infos []FileInfo
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasSuffix(name, ".tmp") {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			return nil, fmt.Errorf("storage: stat during list %q: %w", name, err)
		}
		infos = append(infos, FileInfo{
			ID:      name,
			Size:    info.Size(),
			ModTime: info.ModTime(),
		})
	}
	return infos, nil
}

// Stat returns FileInfo for fileID without reading its contents.
func (s *Store) Stat(fileID string) (FileInfo, error) {
	path, err := s.resolve(fileID)
	if err != nil {
		return FileInfo{}, err
	}

	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return FileInfo{}, fmt.Errorf("%w: %s", ErrNotFound, fileID)
		}
		return FileInfo{}, fmt.Errorf("storage: stat %q: %w", fileID, err)
	}

	return FileInfo{ID: fileID, Size: info.Size(), ModTime: info.ModTime()}, nil
}

// Read returns the full contents of fileID plus its FileInfo.
func (s *Store) Read(fileID string) ([]byte, FileInfo, error) {
	path, err := s.resolve(fileID)
	if err != nil {
		return nil, FileInfo{}, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, FileInfo{}, fmt.Errorf("%w: %s", ErrNotFound, fileID)
		}
		return nil, FileInfo{}, fmt.Errorf("storage: read %q: %w", fileID, err)
	}

	info, err := os.Stat(path)
	if err != nil {
		return nil, FileInfo{}, fmt.Errorf("storage: stat after read %q: %w", fileID, err)
	}

	return data, FileInfo{ID: fileID, Size: info.Size(), ModTime: info.ModTime()}, nil
}

// Write persists r's contents under fileID atomically: it writes to a
// sibling temp file inside DataDir (same filesystem, so os.Rename is
// guaranteed atomic) and renames it into place only after the write fully
// succeeds. A failed or partial write never leaves a truncated destination
// file, and the temp file is cleaned up on any error path.
func (s *Store) Write(fileID string, r io.Reader) (FileInfo, error) {
	path, err := s.resolve(fileID)
	if err != nil {
		return FileInfo{}, err
	}

	tmp, err := os.CreateTemp(s.dataDir, fileID+".*.tmp")
	if err != nil {
		return FileInfo{}, fmt.Errorf("storage: creating temp file for %q: %w", fileID, err)
	}
	tmpPath := tmp.Name()
	// Best-effort cleanup; if the rename below succeeds this is a no-op
	// (the file no longer exists under tmpPath).
	defer os.Remove(tmpPath)

	if _, err := io.Copy(tmp, r); err != nil {
		tmp.Close()
		return FileInfo{}, fmt.Errorf("storage: writing temp file for %q: %w", fileID, err)
	}
	if err := tmp.Close(); err != nil {
		return FileInfo{}, fmt.Errorf("storage: closing temp file for %q: %w", fileID, err)
	}

	if err := os.Rename(tmpPath, path); err != nil {
		return FileInfo{}, fmt.Errorf("storage: committing write for %q: %w", fileID, err)
	}

	info, err := os.Stat(path)
	if err != nil {
		return FileInfo{}, fmt.Errorf("storage: stat after write %q: %w", fileID, err)
	}

	return FileInfo{ID: fileID, Size: info.Size(), ModTime: info.ModTime()}, nil
}

// resolve sanitizes fileID and returns the absolute path it maps to inside
// DataDir. A fileID must be a non-empty exact base filename: no path
// separators (/ or \), and no ".." segment.
func (s *Store) resolve(fileID string) (string, error) {
	if fileID == "" {
		return "", fmt.Errorf("%w: empty file id", ErrBadFileID)
	}
	if strings.ContainsAny(fileID, "/\\") {
		return "", fmt.Errorf("%w: %s", ErrBadFileID, fileID)
	}
	if fileID == "." || fileID == ".." || strings.Contains(fileID, "..") {
		return "", fmt.Errorf("%w: %s", ErrBadFileID, fileID)
	}
	if fileID != filepath.Base(fileID) {
		return "", fmt.Errorf("%w: %s", ErrBadFileID, fileID)
	}

	return filepath.Join(s.dataDir, fileID), nil
}
