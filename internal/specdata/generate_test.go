// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package specdata

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/kottera/metais/internal/testenv"
)

// specFiles names the spec files the binary embeds, by their path below
// spec/, slash-separated. Each is copied to the same path below this
// package's directory.
var specFiles = []string{
	"findings/registry.json",
	"schemas/1.0/finding-registry.schema.json",
	"schemas/1.0/finding.schema.json",
	"schemas/1.0/manifest.schema.json",
}

// specRoot is spec/, from this package's directory, where go test runs.
const specRoot = "../../spec"

var generate = flag.Bool("specdata.generate", false, "copy specFiles from spec/ into this package")

// TestGenerate is the copier go generate runs. Without -specdata.generate
// it does nothing.
func TestGenerate(t *testing.T) {
	if !*generate {
		t.Skip("run by go generate ./internal/specdata")
	}
	if err := copySpecFiles(specRoot, ".", specFiles); err != nil {
		t.Fatal(err)
	}
}

// copySpecFiles copies each of files from below src to the same path below
// dst, byte for byte, creating directories as needed. A copy that already
// matches is left untouched.
func copySpecFiles(src, dst string, files []string) error {
	for _, name := range files {
		if !filepath.IsLocal(filepath.FromSlash(name)) {
			return fmt.Errorf("spec file %q is not a local path", name)
		}
		b, err := os.ReadFile(filepath.Join(src, filepath.FromSlash(name)))
		if err != nil {
			return err
		}
		out := filepath.Join(dst, filepath.FromSlash(name))
		if old, err := os.ReadFile(out); err == nil && bytes.Equal(old, b) {
			continue
		}
		if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(out, b, 0o644); err != nil {
			return err
		}
	}
	return nil
}

func TestCopySpecFiles(t *testing.T) {
	src, dst := testenv.Dir(t), testenv.Dir(t)
	files := map[string][]byte{
		"findings/registry.json":     []byte("{\"codes\": []}\n"),
		"schemas/1.0/a.schema.json":  {0, 1, 2, '\r', '\n', 0xff},
		"top.json":                   {},
		"schemas/1.0/not-copied.txt": []byte("left behind"),
	}
	for name, b := range files {
		p := filepath.Join(src, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, b, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	list := []string{"findings/registry.json", "schemas/1.0/a.schema.json", "top.json"}
	// Twice: the second run finds every copy current.
	for range 2 {
		if err := copySpecFiles(src, dst, list); err != nil {
			t.Fatal(err)
		}
	}
	for _, name := range list {
		got, err := os.ReadFile(filepath.Join(dst, filepath.FromSlash(name)))
		if err != nil {
			t.Errorf("%s: %v", name, err)
			continue
		}
		if !bytes.Equal(got, files[name]) {
			t.Errorf("%s: copy is %q, want %q", name, got, files[name])
		}
	}
	if _, err := os.Stat(filepath.Join(dst, "schemas", "1.0", "not-copied.txt")); !os.IsNotExist(err) {
		t.Errorf("a file not in the list was copied: %v", err)
	}

	// A stale copy is overwritten.
	stale := filepath.Join(dst, "top.json")
	if err := os.WriteFile(stale, []byte("stale"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := copySpecFiles(src, dst, list); err != nil {
		t.Fatal(err)
	}
	if got, _ := os.ReadFile(stale); len(got) != 0 {
		t.Errorf("stale copy not refreshed: %q", got)
	}
}

func TestCopySpecFilesRefuses(t *testing.T) {
	src, dst := testenv.Dir(t), testenv.Dir(t)
	for _, name := range []string{"../escape.json", "/abs.json", ""} {
		if err := copySpecFiles(src, dst, []string{name}); err == nil {
			t.Errorf("copySpecFiles(%q) succeeded, want a refusal", name)
		}
	}
	if err := copySpecFiles(src, dst, []string{"missing.json"}); err == nil {
		t.Errorf("copySpecFiles of a missing spec file succeeded")
	}
}

// TestSpecFilesExist keeps the list honest: every listed spec file exists.
func TestSpecFilesExist(t *testing.T) {
	for _, name := range specFiles {
		if _, err := os.Stat(filepath.Join(specRoot, filepath.FromSlash(name))); err != nil {
			t.Errorf("spec file %s: %v", name, err)
		}
	}
}
