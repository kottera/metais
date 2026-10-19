// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

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

// TestRenderAssetRecords is ADR-005 D7 item 3 and D16 item 2 in render's
// result (ADR-008 D8 item 9): a file copied from an asset is recorded with
// source asset, no kind, its size and the digest of its bytes, beside a
// template file's record; and the human form lists both in path order.
func TestRenderAssetRecords(t *testing.T) {
	root := testenv.Dir(t)
	logo := "\x89PNG\r\n\x1a\n\x00\xff"
	pkg := writeSourcePackage(t, root, "  - id: logo\n    asset: logo.png\n    executable: true\n  - id: page\n    template: page.html.liquid\n",
		map[string]string{"assets/logo.png": logo, "templates/page.html.liquid": `<img src="{% asset "logo.png" %}">`})
	r := runTool(t, "render", pkg, "--out", filepath.Join(root, "out"), "--json")
	if r.status != 0 {
		t.Fatalf("metais render --json: status %d, %s", r.status, r.stdout)
	}
	res := readRender(t, r)
	if len(res.Files) != 2 {
		t.Fatalf("files %+v", res.Files)
	}
	sum := sha256.Sum256([]byte(logo))
	a, p := res.Files[0], res.Files[1]
	if a.Path != "logo.png" || a.Output != "logo" || a.Source != "asset" || a.Kind != nil || a.Size != int64(len(logo)) ||
		a.Digest != "sha256:"+hex.EncodeToString(sum[:]) || !a.Executable || a.Status != "create" {
		t.Errorf("the asset's record %+v", a)
	}
	if p.Path != "page.html" || p.Source != "template" || p.Kind == nil || *p.Kind != "html" {
		t.Errorf("the template's record %+v", p)
	}
	if !strings.Contains(r.stdout, `"kind":null`) {
		t.Errorf("the asset's kind is not null: %s", r.stdout)
	}
	got, err := os.ReadFile(filepath.Join(root, "out", "logo.png"))
	if err != nil || string(got) != logo {
		t.Errorf("logo.png is %q, %v", got, err)
	}

	r = runTool(t, "render", pkg, "--out", filepath.Join(root, "out2"))
	want := "10\t" + a.Digest + "\tlogo.png\n" + "20\t" + p.Digest + "\tpage.html\n"
	if r.status != 0 || r.stdout != want {
		t.Errorf("metais render: status %d, stdout %q, want %q", r.status, r.stdout, want)
	}
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
	s := &session{}
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
	plan, fs := planOutputs(pkg, lim, s, r)
	fs = append(fs, renderPlan(pkg, plan, ts, r, lim)...)
	if len(fs) != 0 || len(plan) != 3 {
		t.Fatalf("plan %d files, findings %v", len(plan), fs)
	}
	for _, name := range slices.Sorted(maps.Keys(c.opened)) {
		if strings.HasPrefix(name, "assets/") {
			t.Errorf("planning and rendering opened %s %d times", name, c.opened[name])
		}
	}
	if f, ok := commit(filepath.Join(root, "out"), plan, s); !ok {
		t.Fatalf("the commit failed: %v", f.Code())
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
