// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package render

import (
	"io"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/kottera/metais/internal/finding"
	"github.com/kottera/metais/internal/limits"
	"github.com/kottera/metais/internal/loader"
	"github.com/kottera/metais/internal/pkgtree"
	"github.com/kottera/metais/internal/template"
	"github.com/kottera/metais/internal/testenv"
)

// writeSourcePackage writes a source directory below root, its manifest's
// outputs section being outputs and its files those of files, and returns
// its path.
func writeSourcePackage(t *testing.T, root, outputs string, files map[string]string) string {
	t.Helper()
	pkg := filepath.Join(root, "pkg")
	files["metais.yaml"] = "metais: \"1.0\"\nname: example\nversion: 1.0.0\ntitle: Example\n" +
		"description: An example package.\nguide: Render it.\ninputs: []\noutputs:\n" + outputs
	for name, text := range files {
		p := filepath.Join(pkg, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return pkg
}

// openCounter is a package tree that counts the opens of each entry.
type openCounter struct {
	pkgtree.Tree
	opened map[string]int
}

func (c *openCounter) Open(name string) (io.ReadCloser, error) {
	c.opened[name]++
	return c.Tree.Open(name)
}

// TestAssetBytesStreamed is ADR-005 D13 item 4 and D5: an asset's bytes
// are not read while the render plans, checks and renders — only its size
// is counted against the output limits — and are read once, from the
// package, as the commit writes them.
func TestAssetBytesStreamed(t *testing.T) {
	root := testenv.Dir(t)
	pkgDir := writeSourcePackage(t, root, "  - id: big\n    asset: big.bin\n  - id: static\n    asset: static\n",
		map[string]string{"assets/big.bin": strings.Repeat("x", 1<<16), "assets/static/a.css": "a", "assets/static/b/c.js": "c"})
	pkg, fs := loader.Open(pkgDir, loader.Options{Limits: limits.Defaults()}, 0)
	if pkg == nil {
		t.Fatalf("the package does not open: %v", fs)
	}
	defer pkg.Close()
	c := &openCounter{Tree: pkg.Tree, opened: map[string]int{}}
	pkg.Tree = c
	lim := limits.Defaults()
	ts := template.NewTemplates(pkg.Tree, lim, pkg.Manifest.Inputs)
	b, err := template.NewBindings(template.Package{
		Tree: pkg.Tree, Manifest: pkg.Manifest, ManifestValue: pkg.ManifestValue,
		Digest: pkg.Digest, Combined: pkg.CombinedValue,
	}, nil, "", lim)
	if err != nil {
		t.Fatal(err)
	}
	res := Run(Options{Package: pkg, Templates: ts, Bindings: b, Limits: lim, Fails: func(fs []finding.Finding) bool { return len(fs) > 0 }},
		Emitter{Emit: func(files []*File) Emission {
			for _, name := range slices.Sorted(maps.Keys(c.opened)) {
				if strings.HasPrefix(name, "assets/") {
					t.Errorf("planning and rendering opened %s %d times", name, c.opened[name])
				}
			}
			if f, ok := Commit(filepath.Join(root, "out"), nil, files, 0); !ok {
				return Emission{Failed: &f}
			}
			return Emission{}
		}})
	if !res.Emitted || len(res.Findings) != 0 || len(res.Files) != 3 {
		t.Fatalf("render of %d files, emitted %v, findings %v", len(res.Files), res.Emitted, res.Findings)
	}
	for _, name := range []string{"assets/big.bin", "assets/static/a.css", "assets/static/b/c.js"} {
		if c.opened[name] != 1 {
			t.Errorf("the commit opened %s %d times, want once", name, c.opened[name])
		}
	}
	if got, err := os.ReadFile(filepath.Join(root, "out", "static", "b", "c.js")); err != nil || string(got) != "c" {
		t.Errorf("static/b/c.js is %q, %v", got, err)
	}
}
