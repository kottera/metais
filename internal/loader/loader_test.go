// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package loader

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/kottera/metais/internal/digest"
	"github.com/kottera/metais/internal/finding"
	"github.com/kottera/metais/internal/jsonx"
	"github.com/kottera/metais/internal/limits"
	"github.com/kottera/metais/internal/testenv"
	"github.com/kottera/metais/internal/zipfmt"
)

// vectorsDir is the conformance suite's vectors, from this package.
const vectorsDir = "../../spec/conformance/vectors"

var defaults = Options{Limits: limits.Defaults()}

// counting is an io.ReaderAt that counts how many times each byte of b is
// read.
type counting struct {
	b     []byte
	reads []int
}

func (c *counting) ReadAt(p []byte, off int64) (int, error) {
	n, err := bytes.NewReader(c.b).ReadAt(p, off)
	for i := range n {
		c.reads[off+int64(i)]++
	}
	return n, err
}

// TestOpenReadsEveryByteOnce is ADR-010 D15 item 2's budget on open
// (W3-1.6), over the whole open: steps 1 to 7, the manifest, every schema
// and the three compiled files re-derived, read every byte of the file
// once and decode every entry once.
func TestOpenReadsEveryByteOnce(t *testing.T) {
	for _, v := range []string{"pkg/opens-packed", "dat/combined-schema-packed", "man/canonical-form-packed"} {
		b, err := os.ReadFile(filepath.Join(vectorsDir, v, "package.metais"))
		if err != nil {
			t.Fatal(err)
		}
		c := &counting{b: b, reads: make([]int, len(b))}
		p, fs := openPacked(c, int64(len(b)), defaults, 0)
		if p == nil {
			t.Fatalf("%s: %v", v, codes(fs))
		}
		for i, n := range c.reads {
			if n != 1 {
				t.Errorf("%s: byte %d of %d was read %d times", v, i, len(b), n)
				break
			}
		}
		a := p.Tree.(*zipfmt.Archive)
		if n := len(a.List()); a.Decodes() != n {
			t.Errorf("%s: %d entries decoded %d times", v, n, a.Decodes())
		}
	}
}

// TestOpenReadsEveryByteOnceControl holds the budget's decode counter to
// seeing a second read: reading one entry after open decodes it again.
// (Its bytes come from the end-record search's tail, which the reader
// holds, so the byte counter does not see it in a package this small.)
func TestOpenReadsEveryByteOnceControl(t *testing.T) {
	b, err := os.ReadFile(filepath.Join(vectorsDir, "pkg/opens-packed", "package.metais"))
	if err != nil {
		t.Fatal(err)
	}
	c := &counting{b: b, reads: make([]int, len(b))}
	p, _ := openPacked(c, int64(len(b)), defaults, 0)
	r, err := p.Tree.Open("metais.yaml")
	if err != nil {
		t.Fatal(err)
	}
	r.Read(make([]byte, 1))
	if a := p.Tree.(*zipfmt.Archive); a.Decodes() != len(a.List())+1 {
		t.Error("a read after open is not seen by the counters")
	}
}

func codes(fs []finding.Finding) []string {
	var out []string
	for _, f := range fs {
		out = append(out, string(f.Code()))
	}
	return out
}

// minimal is a minimal valid package's files (ADR-001 Conformance).
func minimal() map[string][]byte {
	return map[string][]byte{
		"metais.yaml": []byte("metais: \"1.0\"\nname: minimal\nversion: 1.0.0\ntitle: Minimal\ndescription: The smallest package.\n" +
			"guide: Supply a report.\ninputs:\n  - name: report\n    title: Report\n    schema: inputs/report.schema.json\n" +
			"outputs:\n  - id: report\n    template: report.md.liquid\n"),
		"inputs/report.schema.json":  []byte(`{"type": "object"}`),
		"templates/report.md.liquid": []byte("Report\n"),
	}
}

// writeSource writes files as a source directory.
func writeSource(t *testing.T, files map[string][]byte) string {
	dir := testenv.Dir(t)
	for n, b := range files {
		p := filepath.Join(dir, filepath.FromSlash(n))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, b, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

// TestSourceDigest is ADR-001 D6 item 3: a source directory's package
// digest is the digest of its files and the compiled files derived from
// them, computed in memory, and OpenFiles over the same files gives the
// same package.
func TestSourceDigest(t *testing.T) {
	testenv.Home(t)
	files := minimal()
	p, fs := Open(writeSource(t, files), defaults, 0)
	if p == nil {
		t.Fatal(codes(fs))
	}
	defer p.Close()
	var entries []digest.Entry
	for n, b := range files {
		entries = append(entries, digest.Entry{Name: n, Sum: sha256Sum(b)})
	}
	var names []string
	for _, f := range p.Compiled {
		names = append(names, f.Name)
		entries = append(entries, digest.Entry{Name: f.Name, Sum: sha256Sum(f.Content)})
	}
	if !slices.Equal(names, []string{".compiled/manifest.json", ".compiled/references.json", ".compiled/schema.json"}) {
		t.Errorf("compiled files %q", names)
	}
	if p.Digest != digest.Package(entries) {
		t.Errorf("the source directory's digest is not its package's")
	}
	q, fs := OpenFiles(files, defaults)
	if q == nil || q.Digest != p.Digest {
		t.Errorf("OpenFiles: %v", codes(fs))
	}
	if string(p.Compiled[1].Content) != `{"references":[]}` {
		t.Errorf("an empty references index is %s", p.Compiled[1].Content)
	}
}

// TestSourceNames holds a source directory's names to ADR-001 D2 by the
// rule a packed package's are held to, without the offsets a directory
// has not got: the predicate is the one portable-path rule.
func TestSourceNames(t *testing.T) {
	testenv.Home(t)
	for name, c := range map[string]struct {
		extra map[string][]byte
		want  string
	}{
		"not portable": {map[string][]byte{"assets/aux.txt": []byte("a")}, "package.name.not_portable"},
		"segment end":  {map[string][]byte{"assets/name.": []byte("a")}, "package.name.not_portable"},
		"character":    {map[string][]byte{"assets/a\u202eb": []byte("a")}, "package.name.character"},
		"not NFC":      {map[string][]byte{"assets/café": []byte("a")}, "package.name.not_nfc"},
	} {
		files := minimal()
		for n, b := range c.extra {
			files[n] = b
		}
		p, fs := Open(writeSource(t, files), defaults, 0)
		if p != nil || !slices.Equal(codes(fs), []string{c.want}) {
			t.Errorf("%s: %v", name, codes(fs))
			continue
		}
		if _, has := fs[0].Param("offset"); has {
			t.Errorf("%s: a source directory's name finding has an offset", name)
		}
	}
	// OpenFiles holds names to the same rule, case collisions included.
	files := minimal()
	files["assets/STRASSE"], files["assets/stra\u00dfe"] = []byte("a"), []byte("b")
	if _, fs := OpenFiles(files, defaults); !slices.Equal(codes(fs), []string{"package.name.case_collision"}) {
		t.Errorf("OpenFiles with a case collision: %v", codes(fs))
	}
}

// TestSourceLayout is ADR-001 D5 items 5 and 6 for a source directory.
func TestSourceLayout(t *testing.T) {
	testenv.Home(t)
	files := minimal()
	files["README.md"] = []byte("not part of the package\n")
	files[".git/config"] = []byte("ignored\n")
	files["old-1.0.0.metais"] = []byte("ignored\n")
	p, fs := Open(writeSource(t, files), defaults, 0)
	if p == nil {
		t.Fatalf("unknown and ignored top-level names refuse: %v", codes(fs))
	}
	if slices.ContainsFunc(p.Tree.List(), func(n string) bool { return !strings.Contains(n, "/") && n != "metais.yaml" }) {
		t.Errorf("a top-level name outside the layout is an entry: %q", p.Tree.List())
	}
	p.Close()
	q, _ := OpenFiles(minimal(), defaults)
	if p.Digest != q.Digest {
		t.Error("names that are no part of the package change its digest")
	}
	files = minimal()
	delete(files, "metais.yaml")
	files["metais.yml"] = []byte("metais: \"1.0\"\n")
	if _, fs := Open(writeSource(t, files), defaults, 0); !slices.Equal(codes(fs), []string{"package.layout.missing_manifest"}) {
		t.Errorf("no metais.yaml: %v", codes(fs))
	}
}

// TestRefusalWritesNothing holds the loader to only reading: a source
// directory and the temporary home are the same before and after a
// refused open (ADR-011 D10 item 4's property, in process).
func TestRefusalWritesNothing(t *testing.T) {
	home := testenv.Home(t)
	files := minimal()
	files["inputs/report.schema.json"] = []byte(`{"type": 5}`)
	dir := writeSource(t, files)
	before := snapshot(t, dir) + snapshot(t, home)
	if p, _ := Open(dir, defaults, 0); p != nil {
		t.Fatal("the package opened")
	}
	if after := snapshot(t, dir) + snapshot(t, home); after != before {
		t.Errorf("a refused open changed the file system:\n%s\n%s", before, after)
	}
}

func snapshot(t *testing.T, root string) string {
	var b strings.Builder
	filepath.Walk(root, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		c, _ := os.ReadFile(p)
		fmt.Fprintf(&b, "%s %v %d %x\n", p, info.Mode(), info.Size(), sha256Sum(c))
		return nil
	})
	return b.String()
}

// TestReferencesIndex is ADR-008 D17 item 1 on the cases its rule turns
// on: a binary entry, a text entry's lines with and without a final LF,
// an empty entry, and a Markdown title after frontmatter, with none.
func TestReferencesIndex(t *testing.T) {
	files := minimal()
	files["references/a.bin"] = []byte("a\x00b")
	files["references/b.txt"] = []byte("one\ntwo")
	files["references/c.txt"] = []byte("one\ntwo\n")
	files["references/d.txt"] = []byte{}
	files["references/e.MD"] = []byte("---\ntitle: x\n---\n## Not one\n\n# The *title* `here`\n")
	files["references/f.markdown"] = []byte("no heading\n")
	files["references/g.txt"] = []byte("caf\xc3")
	p, fs := OpenFiles(files, defaults)
	if p == nil {
		t.Fatal(codes(fs))
	}
	want := `{"references":[` +
		`{"path":"references/a.bin","size":3,"text":false,"title":null},` +
		`{"lines":2,"path":"references/b.txt","size":7,"text":true,"title":null},` +
		`{"lines":2,"path":"references/c.txt","size":8,"text":true,"title":null},` +
		`{"lines":0,"path":"references/d.txt","size":0,"text":true,"title":null},` +
		`{"lines":6,"path":"references/e.MD","size":50,"text":true,"title":"The title here"},` +
		`{"lines":1,"path":"references/f.markdown","size":11,"text":true,"title":null},` +
		`{"path":"references/g.txt","size":4,"text":false,"title":null}]}`
	if got := string(p.Compiled[1].Content); got != want {
		t.Errorf("the references index is\n%s\nwant\n%s", got, want)
	}
}

// TestRefStatsAcrossWrites holds the streamed UTF-8 check to a character a
// write cuts.
func TestRefStatsAcrossWrites(t *testing.T) {
	for _, chunks := range [][]string{{"caf", "é"}, {"caf\xc3", "\xa9"}, {"\xf0\x9f", "\x98", "\x80!"}} {
		s := &refStats{}
		for _, c := range chunks {
			s.Write([]byte(c))
		}
		if !s.text() {
			t.Errorf("%q is not text", chunks)
		}
	}
	s := &refStats{}
	s.Write([]byte("caf\xc3"))
	if s.text() {
		t.Error("a cut character at the end is text")
	}
}

// TestCompiledFilesReadable is W3-1.14 over the suite, under MTS-MAN-023:
// every compiled file the open derives from a source directory of the
// conformance suite is its own canonical form, and is text the tool's own
// JSON reader accepts unless it holds the canonical text of an integral
// number beyond ±(2^53−1), each of which, and nothing else, the reader
// refuses as unsafe_integer.
func TestCompiledFilesReadable(t *testing.T) {
	testenv.Home(t)
	dirs, _ := filepath.Glob(filepath.Join(vectorsDir, "*", "*", "package"))
	opened, large := 0, 0
	for _, d := range dirs {
		p, _ := Open(d, defaults, 0)
		if p == nil {
			continue
		}
		opened++
		for _, f := range p.Compiled {
			unsafe := unsafeIntegers(t, f.Content)
			doc, fs := jsonx.Parse(f.Content, jsonx.Source{Input: finding.Subject{Kind: finding.SubjectInput, Name: "compiled"}}, limits.Defaults())
			if unsafe > 0 {
				large++
				if len(fs) != unsafe {
					t.Errorf("%s: %s holds %d integral numbers beyond 2^53-1, and the reader reports %v", d, f.Name, unsafe, codes(fs))
				}
				for _, x := range fs {
					r, _ := x.Param("reason")
					if reason, _ := r.Str(); x.Code() != jsonx.CodeNumberOutOfRange || reason != "unsafe_integer" {
						t.Errorf("%s: %s is refused with %s, not only for its large integers", d, f.Name, x.Code())
					}
				}
				continue
			}
			if len(fs) > 0 {
				t.Errorf("%s: %s is not readable: %v", d, f.Name, codes(fs))
				continue
			}
			if !bytes.Equal(jsonx.Canonical(doc.Value), f.Content) {
				t.Errorf("%s: %s is not its own canonical form", d, f.Name)
			}
		}
		p.Close()
	}
	if opened == 0 {
		t.Fatal("no source directory of the suite opened")
	}
	if large == 0 {
		t.Fatal("no compiled file of the suite holds an integral number beyond 2^53-1, so MTS-MAN-023's exception is not exercised")
	}
}

// unsafeIntegers counts the numbers in the JSON text b written without a
// fraction or exponent whose magnitude is above 2^53−1, read by
// encoding/json rather than by the reader under test.
func unsafeIntegers(t *testing.T, b []byte) int {
	t.Helper()
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.UseNumber()
	var v any
	if err := dec.Decode(&v); err != nil {
		t.Fatalf("a compiled file is not JSON: %v", err)
	}
	limit := new(big.Int).SetUint64(1<<53 - 1)
	n := 0
	var walk func(any)
	walk = func(v any) {
		switch v := v.(type) {
		case json.Number:
			i, ok := new(big.Int).SetString(string(v), 10)
			if ok && i.CmpAbs(limit) > 0 {
				n++
			}
		case []any:
			for _, e := range v {
				walk(e)
			}
		case map[string]any:
			for _, e := range v {
				walk(e)
			}
		}
	}
	walk(v)
	return n
}
