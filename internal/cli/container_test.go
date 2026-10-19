// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"testing"
	"time"

	"github.com/kottera/metais/internal/finding"
	"github.com/kottera/metais/internal/limits"
	"github.com/kottera/metais/internal/loader"
	"github.com/kottera/metais/internal/template"
	"github.com/kottera/metais/internal/testenv"
	"github.com/kottera/metais/internal/zipfmt"
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

// W5-6.2, MTS-TRN-007: a container is planned with at most
// zipfmt.MaxEntries entries, 65,534, since a zip without zip64 holds no
// more: exactly that many are accepted, and one more is refused with
// output.container.too_many_entries. Both are planned only, never
// rendered, so the boundary is proved without rendering 65,534 templates;
// trn/container-too-many-entries holds the refusal end to end. The
// planning is bounded work, held to a CPU time.
func TestContainerEntryBoundary(t *testing.T) {
	const cpuBound = 20 * time.Second // about half a second, two under the race detector
	for _, n := range []int{zipfmt.MaxEntries, zipfmt.MaxEntries + 1} {
		t.Run(strconv.Itoa(n), func(t *testing.T) {
			root := testenv.Dir(t)
			dir := writePackage(t, root, "  - id: box\n    path: box.zip\n    container:\n      entries:\n"+
				"        - template: e.txt.liquid\n          for_each: (1.."+strconv.Itoa(n)+")\n          path: \"{{ item }}\"\n",
				map[string]string{"e.txt.liquid": ""})
			pkg, fs := loader.Open(dir, loader.Options{Limits: limits.Defaults()}, 0)
			if pkg == nil {
				t.Fatalf("the package does not open: %v", fs)
			}
			defer pkg.Close()
			lim := limits.Defaults()
			ts := template.NewTemplates(pkg.Tree, lim, pkg.Manifest.Inputs)
			b, err := template.NewBindings(template.Package{
				Tree: pkg.Tree, Manifest: pkg.Manifest, ManifestValue: pkg.ManifestValue,
				Digest: pkg.Digest, Combined: pkg.CombinedValue,
			}, nil, "", lim)
			if err != nil {
				t.Fatal(err)
			}
			r := template.NewRenderer(ts, b, lim, 0)
			before, measured := selfCPU()
			plan, fs := planOutputs(pkg, lim, &session{}, r)
			if after, _ := selfCPU(); measured {
				t.Logf("planning %d entries took %v of CPU", n, after-before)
				if after-before > cpuBound {
					t.Errorf("planning %d entries took %v of CPU, more than %v", n, after-before, cpuBound)
				}
			}
			if len(plan) != 1 || len(plan[0].entries) != n {
				t.Fatalf("planned %d outputs, want one container of %d entries", len(plan), n)
			}
			codes := make([]finding.Code, len(fs))
			for i, f := range fs {
				codes[i] = f.Code()
			}
			var want []finding.Code
			if n > zipfmt.MaxEntries {
				want = []finding.Code{finding.CodeContainerTooManyEntries}
			}
			if !slices.Equal(codes, want) {
				t.Fatalf("planning %d entries: findings %v, want %v", n, codes, want)
			}
		})
	}
}
