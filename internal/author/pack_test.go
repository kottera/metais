// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package author

import (
	"bytes"
	"errors"
	"io"
	"slices"
	"strings"
	"testing"

	"github.com/kottera/metais/internal/fault"
	"github.com/kottera/metais/internal/hostfs"
	"github.com/kottera/metais/internal/limits"
	"github.com/kottera/metais/internal/loader"
	"github.com/kottera/metais/internal/pkgtree"
	"github.com/kottera/metais/internal/pkgtree/pkgtreetest"
	"github.com/kottera/metais/internal/zipfmt"
)

// packFiles is a source directory's entries: a manifest, a schema, a
// template, and an entry that sorts between the compiled files and them.
func packFiles() map[string][]byte {
	return map[string][]byte{
		"metais.yaml": []byte(`metais: "1.0"
name: packed
version: 1.0.0
title: Packed
description: A package PlanPack packs.
guide: Supply a report.
inputs:
  - name: report
    title: Report
    schema: inputs/report.schema.json
outputs:
  - id: report
    template: report.md.liquid
`),
		"inputs/report.schema.json":  []byte(`{"type": "object", "properties": {"title": {"type": "string"}}}` + "\n"),
		"templates/report.md.liquid": []byte("# {{ inputs.report.title }}\r\n"),
		"assets/.hidden":             nil,
	}
}

// planned opens files as a source directory and plans its pack under
// faults.
func planned(t *testing.T, files map[string][]byte, tree pkgtree.Tree, faults fault.Set) (*loader.Package, *Packed) {
	t.Helper()
	pkg, fs := loader.OpenFiles(files, loader.Options{Limits: limits.Defaults()})
	if pkg == nil {
		t.Fatalf("the source does not open: %v", fs)
	}
	if tree == nil {
		tree = pkg.Tree
	}
	p, err := PlanPack(tree, pkg.Compiled, faults)
	if err != nil {
		t.Fatal(err)
	}
	return pkg, p
}

// TestPlanPack is ADR-007 D7 items 3 and 4 and ADR-001 D4 item 7: the
// entries are the source's and the four compiled files, in byte order,
// .compiled/digests.json first; the package digest is the one the open
// computed for the source directory; and the package written opens, with
// that digest and every source entry's bytes as they were.
func TestPlanPack(t *testing.T) {
	files := packFiles()
	pkg, p := planned(t, files, nil, 0)
	want := []string{".compiled/digests.json", ".compiled/manifest.json", ".compiled/references.json", ".compiled/schema.json",
		"assets/.hidden", "inputs/report.schema.json", "metais.yaml", "templates/report.md.liquid"}
	if !slices.Equal(p.Entries, want) {
		t.Errorf("entries %q, want %q", p.Entries, want)
	}
	if p.Digest != pkg.Digest {
		t.Errorf("the package digest is not the source directory's")
	}
	var b bytes.Buffer
	if err := p.Write(&b); err != nil {
		t.Fatal(err)
	}
	a, fs, err := zipfmt.Open(bytes.NewReader(b.Bytes()), int64(b.Len()), limits.Defaults())
	if err != nil || len(fs) > 0 {
		t.Fatalf("the package does not read: %v %v", err, fs)
	}
	if !slices.Equal(a.List(), want) {
		t.Errorf("the zip holds %q", a.List())
	}
	for n, content := range files {
		r, err := a.Open(n)
		if err != nil {
			t.Fatal(err)
		}
		got, _ := io.ReadAll(r)
		r.Close()
		if !bytes.Equal(got, content) {
			t.Errorf("%s holds %q, want %q", n, got, content)
		}
	}
	var again bytes.Buffer
	if err := p.Write(&again); err != nil || !bytes.Equal(again.Bytes(), b.Bytes()) {
		t.Errorf("a second write of one plan gives other bytes: %v", err)
	}
}

// TestPackMapOrder is the control fault.PackMapOrder is for (ADR-010 D13
// item 2): under it the entries are a Go map's order, which differs
// between plans, so the determinism detector that holds pack's bytes
// equal has a failure to catch.
func TestPackMapOrder(t *testing.T) {
	files := packFiles()
	_, want := planned(t, files, nil, 0)
	for range 20 {
		if _, p := planned(t, files, nil, fault.PackMapOrder); !slices.Equal(p.Entries, want.Entries) {
			return
		}
	}
	t.Error("with PackMapOrder, twenty plans all wrote the entries in byte order")
}

// changing is a tree whose entry name reads differently each time it is
// opened.
type changing struct {
	pkgtree.Tree
	name  string
	reads int
}

func (c *changing) Open(path string) (io.ReadCloser, error) {
	if path != c.name {
		return c.Tree.Open(path)
	}
	c.reads++
	return io.NopCloser(strings.NewReader(strings.Repeat("x", c.reads))), nil
}

// failing is a tree whose entry name cannot be read.
type failing struct {
	pkgtree.Tree
	name string
}

func (f failing) Open(path string) (io.ReadCloser, error) {
	if path == f.name {
		return nil, &hostfs.Error{Op: "open", Reason: hostfs.PermissionDenied}
	}
	return f.Tree.Open(path)
}

// TestPackSourceChanged is ADR-007 D7 item 4's guarantee that a package's
// digests are its entries' own: a source entry whose bytes differ from
// those PlanPack hashed fails the write with a *ReadError of reason
// changed, and one that cannot be read fails the plan with its reason.
func TestPackSourceChanged(t *testing.T) {
	files := packFiles()
	mem, err := pkgtreetest.New(files)
	if err != nil {
		t.Fatal(err)
	}
	_, p := planned(t, files, &changing{Tree: mem, name: "metais.yaml"}, 0)
	var re *ReadError
	if err := p.Write(io.Discard); !errors.As(err, &re) || re.Reason != hostfs.Changed {
		t.Errorf("a changed entry: %v", err)
	}
	pkg, _ := loader.OpenFiles(files, loader.Options{Limits: limits.Defaults()})
	_, err = PlanPack(failing{Tree: mem, name: "templates/report.md.liquid"}, pkg.Compiled, 0)
	if !errors.As(err, &re) || re.Reason != hostfs.PermissionDenied {
		t.Errorf("an unreadable entry: %v", err)
	}
}
