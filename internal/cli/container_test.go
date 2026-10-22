// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"metais.org/metais/internal/finding"
	"metais.org/metais/internal/testenv"
)

// manifestHead is a manifest up to its outputs, which a fixture appends.
const manifestHead = `metais: "1.0"
name: example
version: 1.0.0
title: Example
description: An example package.
guide: Supply the inputs and render.
inputs: []
outputs:
`

// writePackage writes a source directory below root holding the manifest
// manifestHead+outputs and each of templates, by its path beneath
// templates/.
func writePackage(t *testing.T, root, outputs string, templates map[string]string) string {
	t.Helper()
	dir := filepath.Join(root, "pkg")
	files := map[string]string{"metais.yaml": manifestHead + outputs}
	for name, text := range templates {
		files[filepath.Join("templates", filepath.FromSlash(name))] = text
	}
	for name, text := range files {
		p := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

// TestRenderFailOnWarning is ADR-006 D2 item 5 on render, through the
// epub profile's warning (ADR-005 D9 item 3): the render warns and
// writes; under --fail-on warning the warning fails it, and nothing is
// written.
func TestRenderFailOnWarning(t *testing.T) {
	root := testenv.Dir(t)
	pkg := writePackage(t, root, `  - id: book
    path: book.epub
    container:
      profile: epub
      entries:
        - template: a.txt.liquid
          path: META-INF/container.xml
        - template: a.txt.liquid
          path: a page.xhtml
`, map[string]string{"a.txt.liquid": "a\n"})
	out := filepath.Join(root, "out")
	r := runTool(t, "render", pkg, "--out", out, "--json")
	if r.status != 0 || !slices.Equal(r.codes(), []string{string(finding.CodeContainerSpace)}) {
		t.Fatalf("render: status %d, findings %q", r.status, r.codes())
	}
	if _, err := os.Stat(filepath.Join(out, "book.epub")); err != nil {
		t.Fatalf("render with a warning wrote no container: %v", err)
	}
	failed := filepath.Join(root, "failed")
	r = runTool(t, "render", pkg, "--out", failed, "--json", "--fail-on", "warning")
	if r.status != finding.ExitFindings || r.doc.OK || !slices.Equal(r.codes(), []string{string(finding.CodeContainerSpace)}) {
		t.Errorf("render --fail-on warning: status %d, ok %t, findings %q", r.status, r.doc.OK, r.codes())
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if e.Name() != "pkg" && e.Name() != "out" {
			t.Errorf("render --fail-on warning left %s", e.Name())
		}
	}
}

// TestContainerEntryExecutable is ADR-005 D2 item 6: executable on a
// container's entry is refused, since a zip records no file mode, and
// nothing is written.
func TestContainerEntryExecutable(t *testing.T) {
	root := testenv.Dir(t)
	pkg := writePackage(t, root, `  - id: box
    path: box.zip
    container:
      entries:
        - template: a.txt.liquid
          executable: true
`, map[string]string{"a.txt.liquid": "a\n"})
	out := filepath.Join(root, "out")
	r := runTool(t, "render", pkg, "--out", out, "--json")
	if r.status == 0 || !slices.Contains(r.codes(), "manifest.one_of") {
		t.Errorf("render of an executable entry: status %d, findings %q", r.status, r.codes())
	}
	if _, err := os.Stat(out); err == nil {
		t.Error("a refused render wrote its output directory")
	}
}
