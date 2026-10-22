// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package guarantees

import (
	"bytes"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"metais.org/metais/internal/testenv"
)

// licenceHeader is the first two lines of every Go file (AGENTS.md,
// Licensing; ADR-010 D1 item 5).
const licenceHeader = "// Copyright 2026 Kottera\n// SPDX-License-Identifier: Apache-2.0\n"

// moduleRoot is the module's root, from this package's directory, where go
// test runs.
const moduleRoot = "../.."

// missingHeader returns the slash-separated path, relative to root, of
// every Go file below root that does not begin with licenceHeader. It
// visits every directory, testdata included, except those whose names
// begin with a dot, which hold no module source (.git, editor and
// worktree directories).
func missingHeader(root string) ([]string, error) {
	var bad []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if path != root && strings.HasPrefix(d.Name(), ".") {
				return filepath.SkipDir
			}
			return nil
		}
		if !d.Type().IsRegular() || !strings.HasSuffix(d.Name(), ".go") {
			return nil
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if !bytes.HasPrefix(b, []byte(licenceHeader)) {
			rel, err := filepath.Rel(root, path)
			if err != nil {
				return err
			}
			bad = append(bad, filepath.ToSlash(rel))
		}
		return nil
	})
	return bad, err
}

func TestLicenceHeaders(t *testing.T) {
	if _, err := os.Stat(filepath.Join(moduleRoot, "go.mod")); err != nil {
		t.Fatalf("module root %s: %v", moduleRoot, err)
	}
	bad, err := missingHeader(moduleRoot)
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range bad {
		t.Errorf("%s does not begin with the licence header of AGENTS.md:\n%s", path, licenceHeader)
	}
}

// TestLicenceHeadersControl runs the check over a tree with files that lack
// the header, so a check gone blind fails the build.
func TestLicenceHeadersControl(t *testing.T) {
	root := testenv.Dir(t)
	files := map[string]string{
		"good.go":                    licenceHeader + "\npackage p\n",
		"none.go":                    "package p\n",
		"late.go":                    "// Package p.\n" + licenceHeader + "package p\n",
		"year.go":                    strings.Replace(licenceHeader, "2026", "2025", 1) + "package p\n",
		"crlf.go":                    strings.ReplaceAll(licenceHeader, "\n", "\r\n") + "package p\n",
		"half.go":                    "// Copyright 2026 Kottera\npackage p\n",
		"sub/testdata/fixture.go":    "package main\n",
		"sub/ok.go":                  licenceHeader + "package sub\n",
		"notgo.txt":                  "no header\n",
		".hidden/skipped.go":         "package hidden\n",
		"sub/.worktree/x/skipped.go": "package x\n",
	}
	for name, body := range files {
		p := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	got, err := missingHeader(root)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"crlf.go", "half.go", "late.go", "none.go", "sub/testdata/fixture.go", "year.go"}
	slices.Sort(got)
	if !slices.Equal(got, want) {
		t.Errorf("missingHeader flagged %q, want %q", got, want)
	}
}
