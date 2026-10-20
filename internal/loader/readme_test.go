// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package loader

import (
	"bytes"
	"crypto/sha256"
	"slices"
	"strings"
	"testing"

	"github.com/kottera/metais/internal/digest"
	"github.com/kottera/metais/internal/finding"
	"github.com/kottera/metais/internal/specdata"
	"github.com/kottera/metais/internal/zipfmt"
)

// readmeCase is how a packed package carries its README.md: its content,
// its method, and whether it is the first entry or in byte order among
// the others. A nil content leaves it out.
type readmeCase struct {
	content []byte
	method  zipfmt.Method
	first   bool
}

// packedWith is files packed as pack packs them, the compiled files the
// open derives and the digest file of every entry but README.md, with
// README.md as r says.
func packedWith(t *testing.T, files map[string][]byte, r readmeCase) []byte {
	t.Helper()
	p, fs := OpenFiles(files, defaults)
	if p == nil {
		t.Fatalf("the sources do not open: %v", codes(fs))
	}
	var es []zipfmt.Entry
	var list []digest.Entry
	for n, b := range files {
		es = append(es, zipfmt.Entry{Name: n, Content: b, Method: zipfmt.Auto})
		list = append(list, digest.Entry{Name: n, Sum: sha256.Sum256(b)})
	}
	for _, f := range p.Compiled {
		es = append(es, zipfmt.Entry{Name: f.Name, Content: f.Content, Method: zipfmt.Auto})
		list = append(list, digest.Entry{Name: f.Name, Sum: sha256.Sum256(f.Content)})
	}
	es = append(es, zipfmt.Entry{Name: digest.File, Content: digest.Encode(list), Method: zipfmt.Auto})
	if r.content != nil && !r.first {
		es = append(es, zipfmt.Entry{Name: zipfmt.ReadmeName, Content: r.content, Method: r.method})
	}
	slices.SortFunc(es, func(a, b zipfmt.Entry) int { return strings.Compare(a.Name, b.Name) })
	if r.content != nil && r.first {
		es = append([]zipfmt.Entry{{Name: zipfmt.ReadmeName, Content: r.content, Method: r.method}}, es...)
	}
	var b bytes.Buffer
	if err := zipfmt.Write(&b, es); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

// readmeReasons is the reason of each package.layout.readme finding of
// fs, and every other code of fs as it is.
func readmeReasons(fs []finding.Finding) []string {
	var out []string
	for _, f := range fs {
		if f.Code() != CodeReadme {
			out = append(out, string(f.Code()))
			continue
		}
		v, _ := f.Param("reason")
		s, _ := v.Str()
		out = append(out, s)
	}
	return out
}

// TestOpenChecksReadme is ADR-001 A1 item 6 (MTS-PKG-071, MTS-PKG-072):
// a packed package opens only when its first entry is README.md, stored,
// holding the README text of its declared version; a package without
// one, with one compressed, with one not first, and with one whose text
// is altered are each refused with package.layout.readme, every reason
// that applies reported at once. README.md is no part of the package
// digest (item 5), so each malformed package carries correct digests.
func TestOpenChecksReadme(t *testing.T) {
	text := specdata.Readme10
	altered := bytes.Replace(text, []byte("METAIS"), []byte("METAlS"), 1)
	cases := []struct {
		name string
		r    readmeCase
		want []string // nil: the package opens
	}{
		{"specification's", readmeCase{text, zipfmt.Store, true}, nil},
		{"missing", readmeCase{nil, zipfmt.Store, true}, []string{"missing"}},
		{"compressed", readmeCase{text, zipfmt.Auto, true}, []string{"compressed"}},
		{"not first", readmeCase{text, zipfmt.Store, false}, []string{"not_first"}},
		{"altered", readmeCase{altered, zipfmt.Store, true}, []string{"altered"}},
		{"CRLF", readmeCase{bytes.ReplaceAll(text, []byte("\n"), []byte("\r\n")), zipfmt.Store, true}, []string{"altered"}},
		{"not first, compressed and altered", readmeCase{altered, zipfmt.Auto, false}, []string{"not_first", "compressed", "altered"}},
	}
	files := minimal()
	src, _ := OpenFiles(files, defaults)
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			b := packedWith(t, files, c.r)
			p, fs := openPacked(bytes.NewReader(b), int64(len(b)), defaults, 0)
			if got := readmeReasons(fs); !slices.Equal(got, c.want) {
				t.Fatalf("findings %q, want %q", got, c.want)
			}
			if c.want != nil {
				return
			}
			if p.Digest != src.Digest {
				t.Errorf("the packed package's digest is not its source directory's")
			}
			// README.md is no part of the content every command reads
			// (A1 item 5).
			if names := p.Tree.List(); !slices.IsSorted(names) || slices.Contains(names, zipfmt.ReadmeName) {
				t.Errorf("the tree lists %q", names)
			}
			if _, err := p.Tree.Open(zipfmt.ReadmeName); err == nil {
				t.Errorf("the tree opens README.md")
			}
		})
	}
}

// TestCheckChecksReadme is the same check in check's reading of a packed
// package, which refuses what open refuses.
func TestCheckChecksReadme(t *testing.T) {
	files := minimal()
	b := packedWith(t, files, readmeCase{specdata.Readme10, zipfmt.Auto, true})
	c, fs := checkPacked(bytes.NewReader(b), int64(len(b)), defaults, 0)
	if c != nil || !slices.Equal(readmeReasons(fs), []string{"compressed"}) {
		t.Errorf("check found %q", readmeReasons(fs))
	}
	b = packedWith(t, files, readmeCase{specdata.Readme10, zipfmt.Store, true})
	if c, fs := checkPacked(bytes.NewReader(b), int64(len(b)), defaults, 0); c == nil {
		t.Errorf("check refused the specification's README: %v", codes(fs))
	}
}

// TestOpenFilesIgnoresReadme is A1 item 7 for a source given in memory:
// its own top-level README.md is ignored silently, as a source
// directory's is, and the package and its digest are those without it.
func TestOpenFilesIgnoresReadme(t *testing.T) {
	files := minimal()
	want, _ := OpenFiles(files, defaults)
	files[zipfmt.ReadmeName] = []byte("# The author's own README\n")
	p, fs := OpenFiles(files, defaults)
	if p == nil {
		t.Fatalf("refused: %v", codes(fs))
	}
	if p.Digest != want.Digest || slices.Contains(p.Tree.List(), zipfmt.ReadmeName) {
		t.Errorf("the author's README.md is part of the package: %q", p.Tree.List())
	}
}
