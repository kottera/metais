// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// declarationPackage is a source directory whose one input, pages, is
// supplied in its data directory, and whose outputs are decls, written
// beneath outputs:. It returns the package and its data directory.
func declarationPackage(t *testing.T, decls string) (pkg, data string) {
	t.Helper()
	dir := writeTree(t, map[string]string{
		"pkg/metais.yaml": `metais: "1.0"
name: example
version: 1.0.0
title: Example
description: An example package.
guide: Supply the pages and render.
inputs:
  - name: pages
    title: Pages
    schema: inputs/pages.schema.json
outputs:
` + decls,
		"pkg/inputs/pages.schema.json": `{"type": "array", "items": {"type": "string"}}`,
		"pkg/templates/t.txt.liquid":   "{{ output.path }}\n",
		"data/pages.json":              `["a", "b", "c"]`,
	})
	return filepath.Join(dir, "pkg"), filepath.Join(dir, "data")
}

// TestNoDeclarationStops renders one fixture per declaration Wave 4's
// render ended with internal.error for, and Wave 5 implements: each
// renders, and none meets internal.error. Each item of Wave 5 that removes
// a stop adds its rows.
func TestNoDeclarationStops(t *testing.T) {
	for _, c := range []struct {
		name  string
		decls string
		files []string
		extra map[string]string // more files of the package
	}{
		{"when", `  - id: w
    template: t.txt.liquid
    when: inputs.pages.size > 1
`, []string{"t.txt"}, nil},
		{"for_each", `  - id: f
    template: t.txt.liquid
    for_each: inputs.pages
    path: "p/{{ item }}.txt"
`, []string{"p/a.txt", "p/b.txt", "p/c.txt"}, nil},
		{"Liquid path", `  - id: l
    template: t.txt.liquid
    path: "{{ inputs.pages | join: '-' }}.txt"
`, []string{"a-b-c.txt"}, nil},
		{"container", `  - id: box
    path: box.epub
    container:
      profile: epub
      entries:
        - template: t.txt.liquid
          path: META-INF/container.xml
        - path: inner.zip
          container:
            entries:
              - template: t.txt.liquid
`, []string{"box.epub"}, nil},
		{"a file asset", `  - id: logo
    asset: logo.svg
`, []string{"logo.svg"}, map[string]string{"assets/logo.svg": "<svg/>"}},
		{"a directory asset", `  - id: static
    asset: static
    path: .
`, []string{"a.css", "js/b.js"}, map[string]string{"assets/static/a.css": "a", "assets/static/js/b.js": "b"}},
		{"a mirror", `  - id: site
    mirror: site
`, []string{"site/index.html"}, map[string]string{"templates/site/index.html.liquid": "{{ output.path }}"}},
	} {
		t.Run(c.name, func(t *testing.T) {
			pkg, data := declarationPackage(t, c.decls)
			for name, body := range c.extra {
				p := filepath.Join(pkg, filepath.FromSlash(name))
				if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			out := filepath.Join(filepath.Dir(pkg), "out")
			r := runTool(t, "render", pkg, "--data", data, "--out", out, "--json")
			if r.status != 0 || slices.Contains(r.codes(), "internal.error") {
				t.Fatalf("status %d, codes %q", r.status, r.codes())
			}
			var got []string
			for _, f := range readRender(t, r).Files {
				got = append(got, f.Path)
			}
			if !slices.Equal(got, c.files) {
				t.Errorf("files %q, want %q", got, c.files)
			}
		})
	}
}

// TestRenderFanOut is a fan-out's result: each file names its output and
// its evaluation's item index, counted before when, in path order; a file
// of a declaration without for_each has none; and each file's body sees
// its own path.
func TestRenderFanOut(t *testing.T) {
	pkg, data := declarationPackage(t, `  - id: page
    template: t.txt.liquid
    for_each: inputs.pages | reverse
    when: item != "b"
    path: "{{ fan_out.index0 }}-{{ item }}.txt"
  - id: index
    template: t.txt.liquid
    path: index.txt
`)
	out := filepath.Join(filepath.Dir(pkg), "out")
	r := runTool(t, "render", pkg, "--data", data, "--out", out, "--json")
	if r.status != 0 {
		t.Fatalf("status %d, %s", r.status, r.stdout)
	}
	type file struct {
		path, output string
		item         int
	}
	var got []file
	for _, f := range readRender(t, r).Files {
		i := -1
		if f.Item != nil {
			i = *f.Item
		}
		got = append(got, file{f.Path, f.Output, i})
		b, err := os.ReadFile(filepath.Join(out, filepath.FromSlash(f.Path)))
		if err != nil || string(b) != f.Path+"\n" {
			t.Errorf("%s holds %q, %v", f.Path, b, err)
		}
	}
	want := []file{{"0-c.txt", "page", 0}, {"2-a.txt", "page", 2}, {"index.txt", "index", -1}}
	if !slices.Equal(got, want) {
		t.Errorf("files %v, want %v", got, want)
	}
}

// TestRenderFanOutPathsFromData is the headline guarantee of paths from
// data (ADR-009 D8 item 2, ADR-005 D12): a path a fan-out renders from the
// inputs is held to the portable-path predicate, so a supplier's traversal,
// absolute path or .git segment is refused, naming its item, and nothing
// is written anywhere.
func TestRenderFanOutPathsFromData(t *testing.T) {
	pkg, data := declarationPackage(t, `  - id: each
    template: t.txt.liquid
    for_each: inputs.pages
    path: "{{ item }}"
`)
	if err := os.WriteFile(filepath.Join(data, "pages.json"),
		[]byte(`["../../escape.txt", "/abs.txt", ".git/hooks/pre-commit", "fine.txt"]`), 0o644); err != nil {
		t.Fatal(err)
	}
	root := filepath.Dir(filepath.Dir(pkg))
	before := snapshotNames(t, root)
	r := runTool(t, "render", pkg, "--data", data, "--out", filepath.Join(filepath.Dir(pkg), "out"), "--json")
	if r.status != 1 {
		t.Fatalf("status %d, %s", r.status, r.stdout)
	}
	var items []string
	for _, f := range r.doc.Findings {
		if !strings.HasPrefix(f.Code, "output.path.") {
			t.Errorf("finding %s", f.Code)
		}
		items = append(items, f.Params["path"].(string))
		if f.Params["item"] == nil {
			t.Errorf("%s names no item", f.Code)
		}
	}
	slices.Sort(items)
	if want := []string{"../../escape.txt", ".git/hooks/pre-commit", "/abs.txt"}; !slices.Equal(items, want) {
		t.Errorf("refused %q, want %q", items, want)
	}
	if after := snapshotNames(t, root); !slices.Equal(before, after) {
		t.Errorf("the refused render changed the tree: %q, then %q", before, after)
	}
}
