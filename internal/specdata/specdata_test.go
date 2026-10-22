// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package specdata

import (
	"bytes"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/kottera/metais/internal/testenv"
)

// mismatches names every file of names whose copy in copies is not byte
// for byte the file below specRoot.
func mismatches(specRoot string, copies map[string][]byte, names []string) []string {
	var bad []string
	for _, name := range names {
		want, err := os.ReadFile(filepath.Join(specRoot, filepath.FromSlash(name)))
		if err != nil {
			bad = append(bad, fmt.Sprintf("%s: %v", name, err))
			continue
		}
		got, ok := copies[name]
		switch {
		case !ok:
			bad = append(bad, name+": not embedded")
		case !bytes.Equal(got, want):
			bad = append(bad, name+": the embedded copy differs from spec/; run go generate ./internal/specdata")
		}
	}
	return bad
}

// TestSpecDataMatchesSpec fails on any difference between an embedded
// copy and its spec file (ADR-010 D2 item 4, ADR-011 D7 item 6).
func TestSpecDataMatchesSpec(t *testing.T) {
	for _, m := range mismatches(specRoot, files, specFiles) {
		t.Error(m)
	}
	embedded := slices.Sorted(maps.Keys(files))
	if listed := slices.Sorted(slices.Values(specFiles)); !slices.Equal(embedded, listed) {
		t.Errorf("embedded %q, but specFiles lists %q", embedded, listed)
	}
}

// TestSpecDataMatchesSpecCatchesOneByte is TestSpecDataMatchesSpec's
// control: a copy one byte away from its spec file, anywhere in it, is
// caught.
func TestSpecDataMatchesSpecCatchesOneByte(t *testing.T) {
	for _, name := range specFiles {
		orig := files[name]
		for _, at := range []int{0, len(orig) / 2, len(orig) - 1} {
			changed := bytes.Clone(orig)
			changed[at] ^= 1
			copies := maps.Clone(files)
			copies[name] = changed
			if got := mismatches(specRoot, copies, specFiles); len(got) != 1 {
				t.Errorf("%s with byte %d changed: %d mismatches reported, want 1", name, at, len(got))
			}
		}
		copies := maps.Clone(files)
		copies[name] = append(bytes.Clone(orig), '\n')
		if got := mismatches(specRoot, copies, specFiles); len(got) != 1 {
			t.Errorf("%s with a byte added: %d mismatches reported, want 1", name, len(got))
		}
	}
	// And a spec file changed under an unchanged copy.
	root := testenv.Dir(t)
	for _, name := range specFiles {
		b := bytes.Clone(files[name])
		if name == specFiles[0] {
			b[len(b)/3] ^= 0x20
		}
		p := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, b, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if got := mismatches(root, files, specFiles); len(got) != 1 {
		t.Errorf("a changed spec file: %d mismatches reported, want 1", len(got))
	}
}

// readmeBlock is the fenced block of chapter 2 that follows the line
// introducing the README text of version, the bytes MTS-PKG-071 fixes,
// each line with its LF.
func readmeBlock(chapter []byte, version string) ([]byte, error) {
	intro := []byte("The **README text** of specification " + version + " is")
	i := bytes.Index(chapter, intro)
	if i < 0 {
		return nil, fmt.Errorf("chapter 2 does not introduce the README text of %s", version)
	}
	rest := chapter[i:]
	open := []byte("\n```text\n")
	j := bytes.Index(rest, open)
	if j < 0 {
		return nil, fmt.Errorf("the README text of %s has no ```text block", version)
	}
	body := rest[j+len(open):]
	k := bytes.Index(body, []byte("\n```\n"))
	if k < 0 {
		return nil, fmt.Errorf("the README text of %s's block is not closed", version)
	}
	return body[:k+1], nil
}

// TestReadmeMatchesChapter holds the README text the tool writes and
// checks (ADR-001 A1) to the text chapter 2 states normatively: one
// source, spec/readme/1.0/README.md, embedded as Readme10, and the same
// bytes in the chapter. TestSpecDataMatchesSpec holds the embedded copy to
// the file.
func TestReadmeMatchesChapter(t *testing.T) {
	chapter, err := os.ReadFile(filepath.Join(specRoot, "02-container.md"))
	if err != nil {
		t.Fatal(err)
	}
	block, err := readmeBlock(chapter, "1.0")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(block, Readme10) {
		t.Errorf("chapter 2's README text of 1.0 is\n%q\nbut spec/readme/1.0/README.md is\n%q", block, Readme10)
	}
	// A1 item 1: UTF-8, LF line endings, ending in one line feed.
	if bytes.ContainsRune(Readme10, '\r') || !bytes.HasSuffix(Readme10, []byte("\n")) || bytes.HasSuffix(Readme10, []byte("\n\n")) {
		t.Errorf("the README text is not LF-terminated lines ending in one line feed: %q", Readme10)
	}
	if !bytes.HasPrefix(Readme10, []byte("# This is a METAIS package\n")) {
		t.Errorf("the README text does not begin with the line a detector keys on (A1 item 8): %q", Readme10)
	}
}

// TestReadmeReadsInPlace is ADR-001 A3: the README tells a reader that
// the file is read in place with the metais tool and not unzipped, to
// start with metais inspect, and that metais read reads a file inside
// it. It names those two commands and no other, the two the reference
// tool keeps for every package of the version (A3 item 3).
func TestReadmeReadsInPlace(t *testing.T) {
	text := string(Readme10)
	for _, want := range []string{"do not unzip it: the metais tool reads\nit in place", "`metais inspect`", "`metais read`"} {
		if !strings.Contains(text, want) {
			t.Errorf("the README text does not say %q:\n%s", want, text)
		}
	}
	if strings.Index(text, "`metais inspect`") > strings.Index(text, "`metais read`") {
		t.Errorf("the README text names read before inspect:\n%s", text)
	}
	var named []string
	for _, m := range regexp.MustCompile("`metais ([a-z-]+)").FindAllStringSubmatch(text, -1) {
		named = append(named, m[1])
	}
	if !slices.Equal(named, []string{"inspect", "read"}) {
		t.Errorf("the README text names the commands %v, want inspect and read alone", named)
	}
}

// TestReadmeMatchesChapterCatchesOneByte is TestReadmeMatchesChapter's
// control: a chapter one byte away from the file is caught.
func TestReadmeMatchesChapterCatchesOneByte(t *testing.T) {
	chapter, err := os.ReadFile(filepath.Join(specRoot, "02-container.md"))
	if err != nil {
		t.Fatal(err)
	}
	block, err := readmeBlock(chapter, "1.0")
	if err != nil {
		t.Fatal(err)
	}
	at := bytes.Index(chapter, block) + len(block)/2
	changed := bytes.Clone(chapter)
	changed[at] ^= 1
	got, err := readmeBlock(changed, "1.0")
	if err == nil && bytes.Equal(got, Readme10) {
		t.Error("a chapter one byte away from the README text was not caught")
	}
}
