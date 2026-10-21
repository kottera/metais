// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package render

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"slices"
	"testing"

	"github.com/kottera/metais/internal/finding"
	"github.com/kottera/metais/internal/limits"
	"github.com/kottera/metais/internal/loader"
	"github.com/kottera/metais/internal/template"
	"github.com/kottera/metais/internal/testenv"
	"github.com/kottera/metais/internal/zipfmt"
)

// TestAssembledContainer drives the assembled producer through Run, under
// -race in the gate: an epub container holding the profile's own
// mimetype, a rendered entry, a copied asset and a nested container, read
// through the pipe of the render's one goroutine (ADR-010 A6 item 1).
// What is read is the zip measured in phase 4, its entries in the
// container's order (ADR-005 D8 item 6), the nested container written in
// place; and a reader that stops early ends the goroutine without an
// error (A6 item 1's early close).
func TestAssembledContainer(t *testing.T) {
	root := testenv.Dir(t)
	dir := writeSourcePackage(t, root, "  - id: book\n    path: book.epub\n    container:\n      profile: epub\n      entries:\n"+
		"        - template: container.xml.liquid\n          path: META-INF/container.xml\n"+
		"        - asset: a.css\n"+
		"        - path: inner.zip\n          container:\n            entries:\n"+
		"              - template: e.txt.liquid\n",
		map[string]string{"templates/container.xml.liquid": "<container/>\n", "templates/e.txt.liquid": "entry\n", "assets/a.css": "a {}\n"})
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
	var read []byte
	res := Run(Options{Package: pkg, Templates: ts, Bindings: b, Limits: lim, Fails: func(fs []finding.Finding) bool { return len(fs) > 0 }},
		emitTo(func(files []*File) Emission {
			r, done := files[0].Open()
			read, err = io.ReadAll(r)
			if derr := done(err); err != nil || derr != nil {
				t.Fatalf("reading the container: %v", errors.Join(err, derr))
			}
			r, done = files[0].Open()
			if _, err := io.ReadFull(r, make([]byte, 8)); err != nil {
				t.Fatalf("reading the container's first bytes: %v", err)
			}
			if err := done(errors.New("stopped")); err != nil {
				t.Errorf("a reader that stopped early: %v", err)
			}
			return Emission{}
		}))
	if !res.Emitted || len(res.Findings) != 0 || len(res.Files) != 1 {
		t.Fatalf("render of %d files, emitted %v, findings %v", len(res.Files), res.Emitted, res.Findings)
	}
	book := res.Files[0]
	sum := sha256.Sum256(read)
	if size, digest := book.Measured(); size != int64(len(read)) || digest != "sha256:"+hex.EncodeToString(sum[:]) {
		t.Errorf("measured %d bytes, %s; read %d bytes, sha256:%x", size, digest, len(read), sum)
	}
	a, zfs, err := zipfmt.OpenContainer(bytes.NewReader(read), int64(len(read)), lim, nil)
	if err != nil || len(zfs) != 0 {
		t.Fatalf("the container is no zip: %v, %v", err, zfs[0].Code())
	}
	want := []string{"mimetype", "META-INF/container.xml", "a.css", "inner.zip"}
	if got := a.List(); !slices.Equal(got, want) {
		t.Errorf("entries %q, want %q", got, want)
	}
	var sources []string
	for _, e := range book.Entries() {
		sources = append(sources, e.Source())
	}
	if want := []string{"profile", "template", "asset", "container"}; !slices.Equal(sources, want) {
		t.Errorf("entries' sources %q, want %q", sources, want)
	}
	if !book.Entries()[0].Stored() || book.Entries()[0].Output() != "" || book.Entries()[0].Kind() != nil {
		t.Errorf("the profile's mimetype is not stored, or has an output or a kind")
	}
	for name, want := range map[string]string{"mimetype": "application/epub+zip", "a.css": "a {}\n"} {
		rc, err := a.Open(name)
		if err != nil {
			t.Fatal(err)
		}
		got, err := io.ReadAll(rc)
		rc.Close()
		if err != nil || string(got) != want {
			t.Errorf("%s is %q, %v; want %q", name, got, err, want)
		}
	}
}
