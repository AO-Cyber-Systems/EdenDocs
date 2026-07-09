/*
 * Copyright 2026 AO Cyber Systems
 *
 * SPDX-License-Identifier: MPL-2.0
 *
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at http://mozilla.org/MPL/2.0/.
 */

package storage

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteThenRead(t *testing.T) {
	t.Run("write then read round-trips exact bytes", func(t *testing.T) {
		store, err := New(t.TempDir())
		if err != nil {
			t.Fatalf("New: %v", err)
		}

		want := []byte("Hello, EdenDocs. This is a hand-typed literal fixture body, not generated content.")
		if _, err := store.Write("note.txt", bytes.NewReader(want)); err != nil {
			t.Fatalf("Write: %v", err)
		}

		got, _, err := store.Read("note.txt")
		if err != nil {
			t.Fatalf("Read: %v", err)
		}
		if !bytes.Equal(got, want) {
			t.Errorf("round-trip mismatch: got %q, want %q", got, want)
		}
	})
}

func TestStatReportsSizeAndModTime(t *testing.T) {
	t.Run("stat reports correct size and non-zero mod time for a seeded file", func(t *testing.T) {
		store, err := New(t.TempDir())
		if err != nil {
			t.Fatalf("New: %v", err)
		}

		content := []byte("twenty-two byte body!")
		if _, err := store.Write("seeded.txt", bytes.NewReader(content)); err != nil {
			t.Fatalf("Write: %v", err)
		}

		info, err := store.Stat("seeded.txt")
		if err != nil {
			t.Fatalf("Stat: %v", err)
		}
		if info.Size != int64(len(content)) {
			t.Errorf("Size mismatch: got %d, want %d", info.Size, len(content))
		}
		if info.ModTime.IsZero() {
			t.Error("ModTime is zero, want a non-zero mod time")
		}
		if info.ID != "seeded.txt" {
			t.Errorf("ID mismatch: got %q, want %q", info.ID, "seeded.txt")
		}
	})
}

func TestListReturnsSeededFiles(t *testing.T) {
	t.Run("list returns seeded files including a copied hello.odt fixture", func(t *testing.T) {
		dataDir := t.TempDir()
		store, err := New(dataDir)
		if err != nil {
			t.Fatalf("New: %v", err)
		}

		if _, err := store.Write("a.txt", bytes.NewReader([]byte("aaa"))); err != nil {
			t.Fatalf("Write a.txt: %v", err)
		}
		if _, err := store.Write("b.txt", bytes.NewReader([]byte("bbbbb"))); err != nil {
			t.Fatalf("Write b.txt: %v", err)
		}

		fixture, err := os.ReadFile(filepath.Join("..", "..", "testdata", "hello.odt"))
		if err != nil {
			t.Fatalf("reading testdata/hello.odt: %v", err)
		}
		if _, err := store.Write("hello.odt", bytes.NewReader(fixture)); err != nil {
			t.Fatalf("Write hello.odt: %v", err)
		}

		infos, err := store.List()
		if err != nil {
			t.Fatalf("List: %v", err)
		}

		got := make(map[string]bool)
		for _, info := range infos {
			got[info.ID] = true
		}
		for _, want := range []string{"a.txt", "b.txt", "hello.odt"} {
			if !got[want] {
				t.Errorf("List missing expected file %q; got %v", want, infos)
			}
		}
		if len(infos) != 3 {
			t.Errorf("List returned %d entries, want 3: %v", len(infos), infos)
		}
	})
}

func TestFileIDTraversalRejected(t *testing.T) {
	badIDs := []string{
		"../escape.txt",
		"..\\escape.txt",
		"a/b.txt",
		"a\\b.txt",
		"../../etc/passwd",
		"..",
		"",
	}

	for _, id := range badIDs {
		id := id
		t.Run("rejects bad fileID "+id, func(t *testing.T) {
			store, err := New(t.TempDir())
			if err != nil {
				t.Fatalf("New: %v", err)
			}

			if _, err := store.Read(id); !errors.Is(err, ErrBadFileID) {
				t.Errorf("Read(%q): got err %v, want ErrBadFileID", id, err)
			}
			if _, err := store.Write(id, bytes.NewReader([]byte("x"))); !errors.Is(err, ErrBadFileID) {
				t.Errorf("Write(%q): got err %v, want ErrBadFileID", id, err)
			}
			if _, err := store.Stat(id); !errors.Is(err, ErrBadFileID) {
				t.Errorf("Stat(%q): got err %v, want ErrBadFileID", id, err)
			}
		})
	}
}

func TestWriteIsAtomic(t *testing.T) {
	t.Run("write is atomic: no temp-file residue and no truncated destination", func(t *testing.T) {
		dataDir := t.TempDir()
		store, err := New(dataDir)
		if err != nil {
			t.Fatalf("New: %v", err)
		}

		original := []byte("original committed content, fully written")
		if _, err := store.Write("doc.txt", bytes.NewReader(original)); err != nil {
			t.Fatalf("Write: %v", err)
		}

		replacement := []byte("replacement content, also fully written, longer than the original")
		if _, err := store.Write("doc.txt", bytes.NewReader(replacement)); err != nil {
			t.Fatalf("Write: %v", err)
		}

		got, _, err := store.Read("doc.txt")
		if err != nil {
			t.Fatalf("Read: %v", err)
		}
		if !bytes.Equal(got, replacement) {
			t.Errorf("final content mismatch: got %q, want %q", got, replacement)
		}

		entries, err := os.ReadDir(dataDir)
		if err != nil {
			t.Fatalf("ReadDir: %v", err)
		}
		for _, e := range entries {
			if strings.HasSuffix(e.Name(), ".tmp") {
				t.Errorf("temp-file residue left behind: %s", e.Name())
			}
		}
	})
}

func TestReadNonexistentFileID(t *testing.T) {
	t.Run("read of a nonexistent fileID returns a typed not-found error", func(t *testing.T) {
		store, err := New(t.TempDir())
		if err != nil {
			t.Fatalf("New: %v", err)
		}

		if _, _, err := store.Read("does-not-exist.txt"); !errors.Is(err, ErrNotFound) {
			t.Errorf("Read: got err %v, want ErrNotFound", err)
		}
		if _, err := store.Stat("does-not-exist.txt"); !errors.Is(err, ErrNotFound) {
			t.Errorf("Stat: got err %v, want ErrNotFound", err)
		}
	})
}
