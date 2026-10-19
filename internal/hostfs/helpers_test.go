// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package hostfs

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/kottera/metais/internal/testenv"
)

// dir gives t a temporary HOME and returns a fresh, empty directory.
func dir(t *testing.T) string {
	t.Helper()
	testenv.Home(t)
	return testenv.Dir(t)
}

// files writes each name, "/"-separated and relative to root, with its
// bytes, creating the directories above it.
func files(t *testing.T, root string, fs map[string]string) {
	t.Helper()
	for name, body := range fs {
		p := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

// symlink makes a symbolic link at name, beneath root, to target.
func symlink(t *testing.T, root, target, name string) {
	t.Helper()
	p := filepath.Join(root, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, p); err != nil {
		t.Skipf("cannot make a symbolic link here: %v", err)
	}
}

// wantReason fails t unless err is an *Error with reason want, and carries
// none of the strings in hidden: every path and name the test gave.
func wantReason(t *testing.T, err error, want Reason, hidden ...string) {
	t.Helper()
	if err == nil {
		t.Fatalf("succeeded, want %s", want)
	}
	var e *Error
	if !errors.As(err, &e) {
		t.Fatalf("returned %T %q, want an *Error", err, err)
	}
	if e.Reason != want {
		t.Fatalf("returned %q, want reason %s", err, want)
	}
	noPath(t, err, hidden...)
}

// noPath fails t if err's text holds any of hidden.
func noPath(t *testing.T, err error, hidden ...string) {
	t.Helper()
	for _, h := range hidden {
		if h != "" && strings.Contains(err.Error(), h) {
			t.Fatalf("error %q holds %q, a path the caller gave", err, h)
		}
	}
}

// tree lists every name beneath root, "/"-separated, each directory with a
// trailing "/", in byte order. It reads nothing.
func tree(t *testing.T, root string) []string {
	t.Helper()
	var out []string
	err := filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(root, p)
		if rel == "." {
			return nil
		}
		rel = filepath.ToSlash(rel)
		if d.IsDir() {
			rel += "/"
		}
		out = append(out, rel)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	slices.Sort(out)
	return out
}

// countFiles counts the entries beneath root that are not directories, as
// tree does, without building or sorting their names: the count is all a
// test of many entries asks of the tree it wrote.
func countFiles(t *testing.T, root string) int {
	t.Helper()
	n := 0
	err := filepath.WalkDir(root, func(_ string, d os.DirEntry, err error) error {
		if err == nil && !d.IsDir() {
			n++
		}
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	return n
}
