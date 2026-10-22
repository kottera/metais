// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"bytes"
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"metais.org/metais/internal/author"
	"metais.org/metais/internal/finding"
	"metais.org/metais/internal/limits"
	"metais.org/metais/internal/loader"
	"metais.org/metais/internal/testenv"
	"metais.org/metais/internal/zipfmt"
)

// The container the tests of the comparison by entries render: box.zip
// holds a.txt, which deflates and whose method the encoder chooses; b.txt,
// declared stored: true; and inner.zip, a container holding c.txt.
const boxOutputs = `  - id: box
    path: box.zip
    container:
      entries:
        - template: a.txt.liquid
        - template: b.txt.liquid
          stored: true
        - id: inner
          path: inner.zip
          container:
            entries:
              - template: c.txt.liquid
`

var (
	boxA = strings.Repeat("a line of a that deflates\n", 20)
	boxB = strings.Repeat("a line of b that deflates\n", 20)
	boxC = strings.Repeat("a line of c that deflates\n", 20)
)

// boxPackage writes the package and returns its source directory, with an
// example basic and no expected/.
func boxPackage(t *testing.T) string {
	t.Helper()
	pkg := writePackage(t, testenv.Dir(t), boxOutputs, map[string]string{"a.txt.liquid": boxA, "b.txt.liquid": boxB, "c.txt.liquid": boxC})
	if err := os.MkdirAll(filepath.Join(pkg, "examples/basic"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pkg, "examples/basic/example.yaml"), []byte("title: Basic\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return pkg
}

// zipBytes is the zip of es, in their order.
func zipBytes(t *testing.T, es ...zipfmt.Entry) []byte {
	t.Helper()
	var b bytes.Buffer
	if err := zipfmt.Write(&b, es); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

// box is an expected box.zip: each entry as named, every method m unless
// an entry's own says otherwise.
type boxEntry struct {
	name    string
	content []byte
	method  zipfmt.Method
}

func boxZip(t *testing.T, es ...boxEntry) []byte {
	var zs []zipfmt.Entry
	for _, e := range es {
		zs = append(zs, zipfmt.Entry{Name: e.name, Content: e.content, Method: e.method})
	}
	return zipBytes(t, zs...)
}

// storedBox is box.zip with every entry stored, inner.zip's included: no
// byte of it is the render's but its entries are.
func storedBox(t *testing.T) []byte {
	inner := boxZip(t, boxEntry{"c.txt", []byte(boxC), zipfmt.Store})
	return boxZip(t, boxEntry{"a.txt", []byte(boxA), zipfmt.Store}, boxEntry{"b.txt", []byte(boxB), zipfmt.Store}, boxEntry{"inner.zip", inner, zipfmt.Store})
}

func writeExpectedBox(t *testing.T, pkg string, b []byte) {
	t.Helper()
	p := filepath.Join(pkg, "examples/basic/expected/box.zip")
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, b, 0o644); err != nil {
		t.Fatal(err)
	}
}

// TestTestContainerByEntries is W6-4.1: a container output is compared by
// its entries, so an expected container whose bytes are none of the
// render's, every entry stored, inner.zip's too, passes; and its zip is
// what --update records, which compares equal.
func TestTestContainerByEntries(t *testing.T) {
	pkg := boxPackage(t)
	if r := runTool(t, "test", pkg, "--update", "--json"); r.status != finding.ExitOK {
		t.Fatalf("--update: status %d, codes %q", r.status, r.codes())
	}
	recorded, err := os.ReadFile(filepath.Join(pkg, "examples/basic/expected/box.zip"))
	if err != nil {
		t.Fatal(err)
	}
	stored := storedBox(t)
	if bytes.Equal(recorded, stored) {
		t.Fatal("the render stores every entry: the comparison by bytes would pass as well")
	}
	for name, b := range map[string][]byte{"recorded": recorded, "stored": stored} {
		writeExpectedBox(t, pkg, b)
		r := runTool(t, "test", pkg, "--json")
		if r.status != finding.ExitOK || testRun(t, r).Examples[0].Status != "pass" {
			t.Errorf("the %s container: status %d, findings %+v", name, r.status, r.doc.Findings)
		}
		// --update writes nothing: the container's entries are the render's.
		u := runTool(t, "test", pkg, "--update", "--dry-run", "--json")
		if res := testRun(t, u); len(res.Update.Written) != 0 {
			t.Errorf("the %s container: --update would write %q", name, res.Update.Written)
		}
	}
}

// containerFinding is the code, location and parameters of a finding,
// without a text difference's diff.
type containerFinding struct {
	code, file string
	origin     string
	params     map[string]any
}

// located is a document's findings with their locations.
type located struct {
	Findings []struct {
		Code     string
		Params   map[string]any
		Location *struct{ Origin, File string }
	}
}

func locatedFindings(t *testing.T, r ran) located {
	t.Helper()
	var doc located
	if err := json.Unmarshal([]byte(r.stdout), &doc); err != nil {
		t.Fatal(err)
	}
	return doc
}

func containerFindings(t *testing.T, r ran) []containerFinding {
	var out []containerFinding
	for _, f := range locatedFindings(t, r).Findings {
		ps := map[string]any{}
		for k, v := range f.Params {
			switch k {
			case "diff", "diff_truncated":
			default:
				ps[k] = v
			}
		}
		cf := containerFinding{code: f.Code, params: ps}
		if f.Location != nil {
			cf.file, cf.origin = f.Location.File, f.Location.Origin
		}
		out = append(out, cf)
	}
	return out
}

// TestTestContainerDifferences is W6-4.2 and W6-4.3: each difference
// between an expected container and the render's, with its code, its
// location at <output>!<entry>, its parameters, the result's record and
// the human line.
func TestTestContainerDifferences(t *testing.T) {
	const file = "examples/basic/expected/box.zip"
	inner := boxZip(t, boxEntry{"c.txt", []byte(boxC), zipfmt.Store})
	a := boxEntry{"a.txt", []byte(boxA), zipfmt.Auto}
	b := boxEntry{"b.txt", []byte(boxB), zipfmt.Store}
	in := boxEntry{"inner.zip", inner, zipfmt.Auto}
	pc := func(kv ...any) map[string]any {
		m := map[string]any{"path": "box.zip"}
		for i := 0; i < len(kv); i += 2 {
			m[kv[i].(string)] = kv[i+1]
		}
		return m
	}
	tests := []struct {
		name     string
		expected []byte
		limit    string
		want     []containerFinding
		diffs    []testDifference
		human    string
	}{
		{
			name:     "missing",
			expected: boxZip(t, a, b, in, boxEntry{"z.txt", []byte("z\n"), zipfmt.Auto}),
			want:     []containerFinding{{"example.entry_missing", file + "!z.txt", "package", pc("entry", "z.txt")}},
			diffs:    []testDifference{{Path: "box.zip!z.txt", Kind: "missing"}},
			human:    "  missing     box.zip!z.txt\n",
		},
		{
			name:     "unexpected",
			expected: boxZip(t, b, in),
			want:     []containerFinding{{"example.entry_unexpected", "box.zip!a.txt", "output", pc("entry", "a.txt")}},
			diffs:    []testDifference{{Path: "box.zip!a.txt", Kind: "unexpected"}},
			human:    "  unexpected  box.zip!a.txt\n",
		},
		{
			name:     "order",
			expected: boxZip(t, b, a, in),
			want:     []containerFinding{{"example.entry_order", file + "!b.txt", "package", pc("index", 0.0, "expected", "b.txt", "rendered", "a.txt")}},
			diffs:    []testDifference{{Path: "box.zip", Kind: "differs"}},
			human:    "  differs     box.zip (entry order: at 0 expected b.txt, rendered a.txt)\n",
		},
		{
			// The order of the entries both hold, past a missing one.
			name:     "order after a missing entry",
			expected: boxZip(t, boxEntry{"0.txt", nil, zipfmt.Auto}, in, a, b),
			want: []containerFinding{
				{"example.entry_missing", file + "!0.txt", "package", pc("entry", "0.txt")},
				{"example.entry_order", file + "!inner.zip", "package", pc("index", 0.0, "expected", "inner.zip", "rendered", "a.txt")},
			},
			diffs: []testDifference{{Path: "box.zip!0.txt", Kind: "missing"}, {Path: "box.zip", Kind: "differs"}},
		},
		{
			name:     "method",
			expected: boxZip(t, a, boxEntry{"b.txt", []byte(boxB), zipfmt.Auto}, in),
			want: []containerFinding{{"example.entry_method", file + "!b.txt", "package",
				pc("entry", "b.txt", "expected_method", "deflated", "rendered_method", "stored")}},
			diffs: []testDifference{{Path: "box.zip!b.txt", Kind: "differs"}},
			human: "  differs     box.zip!b.txt (method: expected deflated, rendered stored)\n",
		},
		{
			name:     "content",
			expected: boxZip(t, boxEntry{"a.txt", []byte(boxA + "more\n"), zipfmt.Auto}, b, in),
			want:     []containerFinding{{"example.output_differs", file + "!a.txt", "package", pc("entry", "a.txt", "binary", false, "cause", "none")}},
			diffs:    []testDifference{{Path: "box.zip!a.txt", Kind: "differs", Line: new(21)}},
			human:    "  differs     box.zip!a.txt (line 21, none)\n",
		},
		{
			name: "content of a container entry",
			expected: boxZip(t, a, b, boxEntry{"inner.zip", boxZip(t, boxEntry{"c.txt", []byte("c\r\n"), zipfmt.Auto},
				boxEntry{"d.txt", nil, zipfmt.Auto}), zipfmt.Auto}),
			// Findings are reported in order of location.
			want: []containerFinding{
				{"example.output_differs", file + "!inner.zip!c.txt", "package", pc("entry", "inner.zip!c.txt", "binary", false, "cause", "none")},
				{"example.entry_missing", file + "!inner.zip!d.txt", "package", pc("entry", "inner.zip!d.txt")},
			},
			diffs: []testDifference{{Path: "box.zip!inner.zip!d.txt", Kind: "missing"}, {Path: "box.zip!inner.zip!c.txt", Kind: "differs", Line: new(1)}},
		},
		{
			name:     "not a zip",
			expected: []byte("not a zip\n"),
			want:     []containerFinding{{"example.expected_unreadable", file, "package", pc("reason", "package.container.not_zip")}},
			diffs:    []testDifference{{Path: "box.zip", Kind: "differs"}},
			human:    "  differs     box.zip (not a readable zip: package.container.not_zip)\n",
		},
		{
			// The bounds are the package reader's: an entry of 2 MiB that
			// deflates beyond entry-ratio, as --limit lowers it.
			name:     "beyond a bound",
			expected: boxZip(t, boxEntry{"a.txt", make([]byte, 2<<20), zipfmt.Auto}, b, in),
			limit:    "entry-ratio=100",
			want:     []containerFinding{{"example.expected_unreadable", file, "package", pc("reason", "limit.package")}},
			diffs:    []testDifference{{Path: "box.zip", Kind: "differs"}},
		},
		{
			// The profile is the package reader's: a timestamp breaks it.
			name:     "outside the profile",
			expected: withTime(t, storedBox(t)),
			want:     []containerFinding{{"example.expected_unreadable", file, "package", pc("reason", "package.container.field_value")}},
			diffs:    []testDifference{{Path: "box.zip", Kind: "differs"}},
		},
		{
			name:     "a container entry that does not read",
			expected: boxZip(t, a, b, boxEntry{"inner.zip", []byte("PK"), zipfmt.Auto}),
			want:     []containerFinding{{"example.expected_unreadable", file + "!inner.zip", "package", pc("entry", "inner.zip", "reason", "package.container.not_zip")}},
			diffs:    []testDifference{{Path: "box.zip!inner.zip", Kind: "differs"}},
		},
	}
	pkg := boxPackage(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			writeExpectedBox(t, pkg, tt.expected)
			argv := []string{"test", pkg}
			if tt.limit != "" {
				argv = append(argv, "--limit", tt.limit)
			}
			r := runTool(t, append(argv, "--json")...)
			if r.status != finding.ExitFindings {
				t.Errorf("status %d", r.status)
			}
			if got := containerFindings(t, r); !equalFindings(got, tt.want) {
				t.Errorf("findings\n%+v\nwant\n%+v", got, tt.want)
			}
			res := testRun(t, r)
			if got := res.Examples[0]; got.Status != "fail" || !equalDiffs(got.Differences, tt.diffs) {
				t.Errorf("result %+v, want differences %+v", got, tt.diffs)
			}
			if tt.human != "" {
				h := runTool(t, argv...)
				if !strings.Contains(h.stdout, "basic: fail\n"+tt.human) {
					t.Errorf("human output\n%s\nwant\n%s", h.stdout, tt.human)
				}
			}
		})
	}
}

func equalFindings(a, b []containerFinding) bool {
	return slices.EqualFunc(a, b, func(x, y containerFinding) bool {
		if x.code != y.code || x.file != y.file || x.origin != y.origin || len(x.params) != len(y.params) {
			return false
		}
		for k, v := range y.params {
			if x.params[k] != v {
				return false
			}
		}
		return true
	})
}

func equalDiffs(a, b []testDifference) bool {
	return slices.EqualFunc(a, b, func(x, y testDifference) bool {
		return x.Path == y.Path && x.Kind == y.Kind && (x.Line == nil) == (y.Line == nil) && (x.Line == nil || *x.Line == *y.Line)
	})
}

// withTime is the zip b with every local and central header's time set
// to 00:00:02, which the profile refuses.
func withTime(t *testing.T, b []byte) []byte {
	t.Helper()
	out := slices.Clone(b)
	for i := 0; i+4 <= len(out); i++ {
		switch string(out[i : i+4]) {
		case "PK\x03\x04":
			out[i+10] = 1
		case "PK\x01\x02":
			out[i+12] = 1
		}
	}
	return out
}

// TestTestContainerUpdate: --update rewrites a container any of whose
// entries differs, whole, and the example then passes.
func TestTestContainerUpdate(t *testing.T) {
	pkg := boxPackage(t)
	writeExpectedBox(t, pkg, boxZip(t, boxEntry{"b.txt", []byte(boxB), zipfmt.Store}))
	r := runTool(t, "test", pkg, "--update", "--json")
	if r.status != finding.ExitOK || len(r.doc.Findings) != 0 {
		t.Fatalf("--update: status %d, codes %q", r.status, r.codes())
	}
	if res := testRun(t, r); !slices.Equal(res.Update.Written, []string{"examples/basic/expected/box.zip"}) || res.Examples[0].Status != "pass" {
		t.Errorf("--update: %+v", res)
	}
	if r := runTool(t, "test", pkg, "--json"); r.status != finding.ExitOK {
		t.Errorf("after --update: status %d, codes %q", r.status, r.codes())
	}
}

// TestTestContainerPacked is W6-4.4 (ADR-007 D6): test on a packed
// package reads each expected container in place and reports what it
// reports for the source directory, a pass and each difference alike.
func TestTestContainerPacked(t *testing.T) {
	pkg := boxPackage(t)
	for _, expected := range [][]byte{storedBox(t), boxZip(t, boxEntry{"b.txt", []byte(boxB), zipfmt.Auto}, boxEntry{"a.txt", []byte("other\n"), zipfmt.Auto})} {
		writeExpectedBox(t, pkg, expected)
		packed := filepath.Join(testenv.Dir(t), "box.metais")
		if err := os.WriteFile(packed, packDir(t, pkg), 0o644); err != nil {
			t.Fatal(err)
		}
		src, zip := runTool(t, "test", pkg, "--json"), runTool(t, "test", packed, "--json")
		if src.status != zip.status || !slices.Equal(src.codes(), zip.codes()) || !bytes.Equal(src.doc.Result, zip.doc.Result) {
			t.Errorf("source: %d %q %s\npacked: %d %q %s", src.status, src.codes(), src.doc.Result, zip.status, zip.codes(), zip.doc.Result)
		}
		if s, z := containerFindings(t, src), containerFindings(t, zip); !equalFindings(s, z) || len(s) == 0 && !bytes.Equal(expected, storedBox(t)) {
			t.Errorf("source findings %+v\npacked %+v", s, z)
		}
	}
}

// packDir is the packed package of the source directory dir, as pack
// writes it.
func packDir(t *testing.T, dir string) []byte {
	t.Helper()
	files := map[string][]byte{}
	err := filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		b, err := os.ReadFile(p)
		rel, _ := filepath.Rel(dir, p)
		files[filepath.ToSlash(rel)] = b
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	return packFiles(t, files).Bytes()
}

// packFiles is the package pack writes from files, a source directory's
// entries: the plan of internal/author, which pack writes (ADR-007 D7).
func packFiles(t testing.TB, files map[string][]byte) *bytes.Buffer {
	t.Helper()
	p, fs := loader.OpenFiles(files, loader.Options{Limits: limits.Defaults()})
	if p == nil {
		t.Fatalf("the source directory does not open: %v", fs)
	}
	plan, err := author.PlanPack(p.Tree, p.Compiled, p.Manifest.Metais, 0)
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	if err := plan.Write(&buf); err != nil {
		t.Fatal(err)
	}
	return &buf
}

// TestTestContainerStoredZipEntry: a container whose entry is a container
// declared stored: true holds that zip's end record among its bytes, and
// what --update records of it still reads and compares equal.
func TestTestContainerStoredZipEntry(t *testing.T) {
	pkg := writePackage(t, testenv.Dir(t), strings.Replace(boxOutputs, "          path: inner.zip\n", "          path: inner.zip\n          stored: true\n", 1),
		map[string]string{"a.txt.liquid": boxA, "b.txt.liquid": boxB, "c.txt.liquid": boxC})
	if err := os.MkdirAll(filepath.Join(pkg, "examples/basic"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pkg, "examples/basic/example.yaml"), []byte("title: Basic\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if r := runTool(t, "test", pkg, "--update", "--json"); r.status != finding.ExitOK {
		t.Fatalf("--update: status %d, codes %q", r.status, r.codes())
	}
	b, err := os.ReadFile(filepath.Join(pkg, "examples/basic/expected/box.zip"))
	if err != nil || !bytes.Contains(b[:len(b)-22], []byte("PK\x05\x06")) {
		t.Fatalf("the recorded container holds no stored zip: %v", err)
	}
	if r := runTool(t, "test", pkg, "--json"); r.status != finding.ExitOK {
		t.Errorf("test: status %d, codes %q", r.status, r.codes())
	}
}
