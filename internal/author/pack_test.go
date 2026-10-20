// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package author

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"slices"
	"strings"
	"testing"

	"github.com/kottera/metais/internal/digest"
	"github.com/kottera/metais/internal/fault"
	"github.com/kottera/metais/internal/hostfs"
	"github.com/kottera/metais/internal/limits"
	"github.com/kottera/metais/internal/loader"
	"github.com/kottera/metais/internal/pkgtree"
	"github.com/kottera/metais/internal/pkgtree/pkgtreetest"
	"github.com/kottera/metais/internal/specdata"
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
	p, err := PlanPack(tree, pkg.Compiled, pkg.Manifest.Metais, faults)
	if err != nil {
		t.Fatal(err)
	}
	return pkg, p
}

// TestPlanPack is ADR-007 D7 items 3 and 4 and ADR-001 D4 item 7: the
// entries are README.md, then the source's and the four compiled files,
// in byte order, .compiled/digests.json first among them (ADR-001 A1
// item 4); the package digest is the one the open
// computed for the source directory; and the package written opens, with
// that digest and every source entry's bytes as they were.
func TestPlanPack(t *testing.T) {
	files := packFiles()
	pkg, p := planned(t, files, nil, 0)
	want := []string{"README.md", ".compiled/digests.json", ".compiled/manifest.json", ".compiled/references.json", ".compiled/schema.json",
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
	if listed := slices.Sorted(slices.Values(want)); !slices.Equal(a.List(), listed) {
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

// TestEntryAt is the entry pack names for a byte of the package it last
// wrote (R1-014's advice): the entry whose local record or central
// directory header holds the byte, so the signature of a second end
// record, in an entry's stored bytes or in its header's CRC or sizes, is
// traced to the entry an author changes; the end record is no entry's,
// and before a write nothing is known.
func TestEntryAt(t *testing.T) {
	files := packFiles()
	files["assets/blob.bin"] = []byte("xx\x01\x7fPK\x05\x06\xfe\x80\x00")
	_, p := planned(t, files, nil, 0)
	if n, ok := p.EntryAt(0); ok {
		t.Errorf("before a write, offset 0 is %q's", n)
	}
	var b bytes.Buffer
	if err := p.Write(&b); err != nil {
		t.Fatal(err)
	}
	pkg := b.Bytes()
	sig := bytes.Index(pkg, []byte("PK\x05\x06"))
	if n, ok := p.EntryAt(int64(sig)); !ok || n != "assets/blob.bin" {
		t.Errorf("the signature at %d is %q's (%v), want assets/blob.bin's", sig, n, ok)
	}
	last := p.Entries[len(p.Entries)-1]
	if n, ok := p.EntryAt(int64(len(pkg) - 22 - 1)); !ok || n != last {
		t.Errorf("the central directory's last byte is %q's (%v), want %q's", n, ok, last)
	}
	if n, ok := p.EntryAt(int64(len(pkg) - 22)); ok {
		t.Errorf("the end record is %q's", n)
	}
}

// TestPackMapOrder is the control fault.PackMapOrder is for (ADR-010 D13
// item 2): under it the digest file's members are a Go map's order, which
// differs between plans, so the determinism detector that holds pack's
// bytes equal has a failure to catch; the entries stay in byte order.
func TestPackMapOrder(t *testing.T) {
	files := packFiles()
	_, want := planned(t, files, nil, 0)
	for range 20 {
		_, p := planned(t, files, nil, fault.PackMapOrder)
		if !slices.Equal(p.Entries, want.Entries) {
			t.Fatalf("the entries moved: %q", p.Entries)
		}
		if !bytes.Equal(p.bytes[digest.File], want.bytes[digest.File]) {
			return
		}
	}
	t.Error("with PackMapOrder, twenty plans all wrote the canonical digest file")
}

// TestEntryDirs is every directory that holds an entry, each once.
func TestEntryDirs(t *testing.T) {
	got := EntryDirs([]string{"a/b/c.txt", "a/b/d.txt", "a/e.txt", "metais.yaml", "t/x/y/z"})
	if want := []string{"a", "a/b", "t", "t/x", "t/x/y"}; !slices.Equal(got, want) {
		t.Errorf("EntryDirs = %q, want %q", got, want)
	}
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
	_, err = PlanPack(failing{Tree: mem, name: "templates/report.md.liquid"}, pkg.Compiled, pkg.Manifest.Metais, 0)
	if !errors.As(err, &re) || re.Reason != hostfs.PermissionDenied {
		t.Errorf("an unreadable entry: %v", err)
	}
}

// readmeFirstStored is ADR-001 A1 items 1 and 8 over a packed package's
// bytes: its first local header names README.md, stored, with no extra
// field, so the README text is the file's bytes from offset 39.
func readmeFirstStored(b, text []byte) error {
	le := binary.LittleEndian
	switch {
	case len(b) < 39+len(text) || le.Uint32(b) != 0x04034b50:
		return errors.New("no local header at offset 0")
	case string(b[30:39]) != "README.md" || le.Uint16(b[26:]) != 9:
		return fmt.Errorf("the first entry is %q", b[30:min(len(b), 30+int(le.Uint16(b[26:])))])
	case le.Uint16(b[8:]) != 0:
		return fmt.Errorf("README.md has method %d, not stored", le.Uint16(b[8:]))
	case le.Uint16(b[28:]) != 0:
		return errors.New("README.md has an extra field")
	case !bytes.Equal(b[39:39+len(text)], text):
		return fmt.Errorf("the bytes at offset 39 are %q", b[39:39+len(text)])
	}
	return nil
}

// TestPackWritesReadmeFirst is ADR-001 A1 items 1, 4 and 5 (MTS-PKG-035,
// MTS-PKG-036, MTS-PKG-071): pack writes the README text of the declared
// version as README.md, the first entry and stored, before .compiled/;
// the digest file does not list it, and the package digest is the source
// directory's.
func TestPackWritesReadmeFirst(t *testing.T) {
	files := packFiles()
	pkg, p := planned(t, files, nil, 0)
	if len(p.Entries) < 2 || p.Entries[0] != "README.md" || p.Entries[1] != digest.File {
		t.Fatalf("entries %q, want README.md then %s first", p.Entries, digest.File)
	}
	var b bytes.Buffer
	if err := p.Write(&b); err != nil {
		t.Fatal(err)
	}
	if err := readmeFirstStored(b.Bytes(), specdata.Readme10); err != nil {
		t.Error(err)
	}
	if bytes.Contains(p.bytes[digest.File], []byte(`"README.md"`)) {
		t.Errorf("the digest file lists README.md: %s", p.bytes[digest.File])
	}
	if p.Digest != pkg.Digest {
		t.Errorf("the package digest is not the source directory's")
	}
}

// TestReadmeFirstStoredControl holds readmeFirstStored to failing on a
// writer that deflates README.md as it deflates other entries, and on one
// that writes it in byte order rather than first.
func TestReadmeFirstStoredControl(t *testing.T) {
	text := specdata.Readme10
	rest := zipfmt.Entry{Name: digest.File, Content: []byte("{}"), Method: zipfmt.Auto}
	for name, es := range map[string][]zipfmt.Entry{
		"deflated":  {{Name: "README.md", Content: text, Method: zipfmt.Auto}, rest},
		"not first": {rest, {Name: "README.md", Content: text, Method: zipfmt.Store}},
	} {
		var b bytes.Buffer
		if err := zipfmt.Write(&b, es); err != nil {
			t.Fatal(err)
		}
		if readmeFirstStored(b.Bytes(), text) == nil {
			t.Errorf("%s: a package with README.md %s passed", name, name)
		}
	}
}
