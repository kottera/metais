// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package conformance

import (
	"bytes"
	"strings"
	"testing"
)

// TestScanPacked pins what the scan of a package pack wrote sees (ADR-009
// D16 item 1): a sentinel in a deflated entry, which its raw bytes do not
// show; one in a zip inside an entry; and a package behind a prefix, which
// the scan of any other written file, inflating only what begins as a zip,
// passes over. A clean package scans clean.
func TestScanPacked(t *testing.T) {
	leak := sentinel + "HOME"
	clean := []zipEntry{
		{name: "metais.yaml", method: methodDeflate, content: []byte(padding + "name: fake\n")},
		{name: "mimetype", method: methodStore, content: []byte("application/x-fake")},
	}
	planted := append(clean[:len(clean):len(clean)], zipEntry{name: "references/leak.txt", method: methodDeflate,
		content: []byte(padding + leak + padding)})
	nested := append(clean[:len(clean):len(clean)], zipEntry{name: "assets/inner.zip", method: methodDeflate,
		content: writeZip([]zipEntry{{name: "deep.txt", method: methodDeflate, content: []byte(padding + leak + padding)}}, false)})
	pkg := writeZip(planted, false)
	if bytes.Contains(asciiUpper(pkg), []byte(sentinel)) {
		t.Fatal("the planted package shows the sentinel in its raw bytes, so this proves nothing of its entries")
	}

	for _, c := range []struct {
		name string
		b    []byte
		want string // a prefix of the hit, or "" for none
	}{
		{"clean", writeZip(clean, false), ""},
		{"a deflated entry", pkg, "p: entry references/leak.txt holds the sentinel from HOME"},
		{"a zip inside an entry", writeZip(nested, false), "p: entry assets/inner.zip: entry deep.txt holds the sentinel from HOME"},
		{"behind a prefix", append([]byte(packStub), pkg...), "p does not begin as a package"},
		{"cut short", pkg[:len(pkg)-30], "p reads as a zip that cannot be opened"},
	} {
		t.Run(c.name, func(t *testing.T) {
			hit := scanPacked(sentinels, "p", c.b)
			if (c.want == "") != (hit == "") || !strings.HasPrefix(hit, c.want) {
				t.Errorf("the scan reports %q, want %q", hit, c.want)
			}
		})
	}
	if hit := scanFor(sentinels, "p", append([]byte(packStub), pkg...), zipDepth); hit != "" {
		t.Errorf("the scan of a written file now sees behind a prefix (%q), so scanPacked's own check proves nothing", hit)
	}
}

// TestPackedFiles holds the pack scan to what a pack step wrote: every
// file it created or changed in the work directory, and nothing a step of
// any other operation wrote.
func TestPackedFiles(t *testing.T) {
	before := map[string]node{
		"work/out/old.metais":  {kind: "file", digest: "1"},
		"work/out/same.metais": {kind: "file", digest: "2"},
		"work/package":         {kind: "dir"},
	}
	after := map[string]node{
		"work/out/old.metais":  {kind: "file", digest: "3"},
		"work/out/same.metais": {kind: "file", digest: "2"},
		"work/out/new.metais":  {kind: "file", digest: "4"},
		"work/out/dir":         {kind: "dir"},
		"work/package":         {kind: "dir"},
		"home/stray":           {kind: "file", digest: "5"},
	}
	got := packedFiles(&step{Op: "pack"}, before, after)
	if want := []string{"out/new.metais", "out/old.metais"}; strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("a pack step wrote %q, want %q", got, want)
	}
	if got := packedFiles(&step{Op: "render"}, before, after); got != nil {
		t.Errorf("a render step packed %q", got)
	}
	trees := map[string]*tree{"out": {files: map[string]treeFile{"new.metais": {}, "old.metais": {}, "notes.txt": {}}}}
	if kept := withoutFiles(trees, got); len(kept["out"].files) != 1 || len(trees["out"].files) != 3 {
		t.Errorf("the trees without the packages hold %d files, and the trees %d", len(kept["out"].files), len(trees["out"].files))
	}
}
