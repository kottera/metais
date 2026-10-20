// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package guarantees

import (
	"path"
	"testing"
)

// readerDocs are the documents a reader of the repository starts at, from
// the module root, each with the directories it must link to rather than
// name as a path (W7-NB5).
var readerDocs = map[string][]string{
	"README.md":     {"examples/"},
	"docs/INDEX.md": {"dev/rounds/"},
}

// TestReaderDocLinks holds every link of README.md and docs/INDEX.md to a
// file and an anchor that exist, and requires each to link the directories
// readerDocs names.
func TestReaderDocLinks(t *testing.T) {
	var docs []*mdDoc
	for rel := range readerDocs {
		d, err := readDoc(moduleRoot, rel)
		if err != nil {
			t.Fatal(err)
		}
		docs = append(docs, d)
	}
	report(t, linkProblems(moduleRoot, docs))
	for _, d := range docs {
		for _, dir := range readerDocs[d.path] {
			if !linksTo(d, dir) {
				t.Errorf("%s names %s without linking it", d.path, dir)
			}
		}
	}
}

// linksTo reports whether d has a link, outside its code blocks, whose
// target is dir.
func linksTo(d *mdDoc, dir string) bool {
	for i, l := range d.lines {
		if d.fenced[i] {
			continue
		}
		for _, m := range mdLinkRe.FindAllStringSubmatch(l, -1) {
			if path.Clean(m[2]) == path.Clean(dir) {
				return true
			}
		}
	}
	return false
}

func TestLinksTo(t *testing.T) {
	d := scanDoc("x.md", "See `examples/` and [the rounds](dev/rounds/).\n\n```\n[examples](examples/)\n```\n")
	for dir, want := range map[string]bool{"dev/rounds/": true, "dev/rounds": true, "examples/": false} {
		if got := linksTo(d, dir); got != want {
			t.Errorf("linksTo(%q) = %v, want %v", dir, got, want)
		}
	}
}
