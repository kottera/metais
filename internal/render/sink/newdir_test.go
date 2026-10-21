// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package sink

import (
	"io"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/kottera/metais/internal/fault"
	"github.com/kottera/metais/internal/finding"
	"github.com/kottera/metais/internal/limits"
	"github.com/kottera/metais/internal/loader"
	"github.com/kottera/metais/internal/pkgtree"
	"github.com/kottera/metais/internal/render"
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

// renderTo renders the package at dir, with no inputs, under faults, to
// s, through wrap's tree when wrap is not nil, as the render command
// does.
func renderTo(t *testing.T, dir string, faults fault.Set, wrap func(pkgtree.Tree) pkgtree.Tree, s render.Sink) render.Result {
	t.Helper()
	pkg, fs := loader.Open(dir, loader.Options{Limits: limits.Defaults()}, 0)
	if pkg == nil {
		t.Fatalf("the package does not open: %v", fs)
	}
	t.Cleanup(func() { pkg.Close() })
	if wrap != nil {
		pkg.Tree = wrap(pkg.Tree)
	}
	lim := limits.Defaults()
	ts := template.NewTemplates(pkg.Tree, lim, pkg.Manifest.Inputs)
	b, err := template.NewBindings(template.Package{
		Tree: pkg.Tree, Manifest: pkg.Manifest, ManifestValue: pkg.ManifestValue,
		Digest: pkg.Digest, Combined: pkg.CombinedValue,
	}, nil, "", lim)
	if err != nil {
		t.Fatal(err)
	}
	return render.Run(render.Options{Package: pkg, Templates: ts, Bindings: b, Limits: lim, Seams: faults,
		Fails: func(fs []finding.Finding) bool { return len(fs) > 0 }}, s)
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

// beforeEmit is s, with check run on the files before s emits them.
type beforeEmit struct {
	Sink
	check func([]*render.File)
}

func (b beforeEmit) Emit(files []*render.File) Emission {
	b.check(files)
	return b.Sink.Emit(files)
}

// TestAssetBytesStreamed is ADR-005 D13 item 4 and D5: an asset's bytes
// are not read while the render plans, checks and renders — only its size
// is counted against the output limits — and are read once, from the
// package, as the new-directory sink writes them.
func TestAssetBytesStreamed(t *testing.T) {
	root := testenv.Dir(t)
	pkgDir := writeSourcePackage(t, root, "  - id: big\n    asset: big.bin\n  - id: static\n    asset: static\n",
		map[string]string{"assets/big.bin": strings.Repeat("x", 1<<16), "assets/static/a.css": "a", "assets/static/b/c.js": "c"})
	var c *openCounter
	wrap := func(tree pkgtree.Tree) pkgtree.Tree {
		c = &openCounter{Tree: tree, opened: map[string]int{}}
		return c
	}
	s := beforeEmit{Sink: Directory(filepath.Join(root, "out"), false, limits.Defaults(), 0), check: func([]*render.File) {
		for _, name := range slices.Sorted(maps.Keys(c.opened)) {
			if strings.HasPrefix(name, "assets/") {
				t.Errorf("planning and rendering opened %s %d times", name, c.opened[name])
			}
		}
	}}
	defer s.Close()
	res := renderTo(t, pkgDir, 0, wrap, s)
	if !res.Emitted || len(res.Findings) != 0 || len(res.Files) != 3 {
		t.Fatalf("render of %d files, emitted %v, findings %v", len(res.Files), res.Emitted, res.Findings)
	}
	for _, name := range []string{"assets/big.bin", "assets/static/a.css", "assets/static/b/c.js"} {
		if c.opened[name] != 1 {
			t.Errorf("the sink opened %s %d times, want once", name, c.opened[name])
		}
	}
	if got, err := os.ReadFile(filepath.Join(root, "out", "static", "b", "c.js")); err != nil || string(got) != "c" {
		t.Errorf("static/b/c.js is %q, %v", got, err)
	}
}

// A render whose write fails part way, after the staging directory exists
// and a file is written beneath it, aborts: neither the staging directory,
// nor the output directory, nor an ancestor the render created is left
// (ADR-005 D14 item 1.5). The second planned path runs through the first,
// a file, which no plan that passed the path checks holds, so the plan is
// made with SkipConfine, which skips the portable-path predicate, and the
// create fails beneath the staging directory.
func TestRenderCommitAbortsAFailedWrite(t *testing.T) {
	testenv.Home(t)
	root := testenv.Dir(t)
	pkgDir := writeSourcePackage(t, root, "  - id: a\n    template: a.liquid\n    path: a.txt\n"+
		"  - id: b\n    template: b.liquid\n    path: a.txt/b.txt\n",
		map[string]string{"templates/a.liquid": "A", "templates/b.liquid": "B"})
	parent := filepath.Join(root, "parent")
	if err := os.Mkdir(parent, 0o755); err != nil {
		t.Fatal(err)
	}
	s := Directory(filepath.Join(parent, "x", "out"), false, limits.Defaults(), 0)
	res := renderTo(t, pkgDir, fault.SkipConfine, nil, s)
	if res.Emitted || len(res.Findings) != 1 {
		t.Fatalf("the render emitted %v, findings %v", res.Emitted, res.Findings)
	}
	if f := res.Findings[0]; f.Code() != finding.CodeDestinationNothingWritten {
		t.Errorf("finding %s, want %s", f.Code(), finding.CodeDestinationNothingWritten)
	}
	entries, err := os.ReadDir(parent)
	if err != nil {
		t.Fatal(err)
	}
	var left []string
	for _, e := range entries {
		left = append(left, e.Name())
	}
	if len(left) != 0 {
		t.Errorf("the failed render left %q", left)
	}
}
