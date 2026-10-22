// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"archive/tar"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"testing"

	"metais.org/metais/internal/fault"
	"metais.org/metais/internal/finding"
	"metais.org/metais/internal/hostenv"
	"metais.org/metais/internal/tarw"
	"metais.org/metais/internal/testenv"
)

// streamDecls are the outputs of the stream tests' package: a fan-out of
// three files, one with a path no ustar header holds, and an executable
// script.
const streamDecls = `  - id: page
    template: t.txt.liquid
    for_each: inputs.pages
    path: "pages/{{ item }}.txt"
  - id: run
    template: t.txt.liquid
    path: run.sh
    executable: true
`

// streamFile is one file the stream tests' package renders.
type streamFile struct {
	path       string
	item       *int
	output     string
	kind       string
	executable bool
}

// streamPackage is the stream tests' package and its data directory, and
// the files it renders, in path order. Each file's body is its own path
// and a line feed.
func streamPackage(t *testing.T) (pkg, data string, files []streamFile) {
	t.Helper()
	pkg, data = declarationPackage(t, streamDecls)
	long := strings.Repeat("ü", 60)
	if err := os.WriteFile(filepath.Join(data, "pages.json"), []byte(`["b", "a", "`+long+`"]`), 0o644); err != nil {
		t.Fatal(err)
	}
	idx := func(i int) *int { return &i }
	return pkg, data, []streamFile{
		{path: "pages/a.txt", item: idx(1), output: "page", kind: "text"},
		{path: "pages/b.txt", item: idx(0), output: "page", kind: "text"},
		{path: "pages/" + long + ".txt", item: idx(2), output: "page", kind: "text"},
		{path: "run.sh", output: "run", kind: "hash", executable: true},
	}
}

// wantArchive is ADR-005 D17's archive of files, each holding its path
// and a line feed.
func wantArchive(t *testing.T, files []streamFile) []byte {
	t.Helper()
	var b bytes.Buffer
	w := tarw.NewWriter(&b)
	for _, f := range files {
		body := f.path + "\n"
		if err := w.Add(f.path, f.executable, int64(len(body)), strings.NewReader(body)); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

// digestOf is ADR-005 D16 item 2's digest of b.
func digestOf(b string) string {
	sum := sha256.Sum256([]byte(b))
	return "sha256:" + hex.EncodeToString(sum[:])
}

// checkListing compares a render's files with files, each with status.
func checkListing(t *testing.T, got []renderFile, files []streamFile, status string) {
	t.Helper()
	var want []renderFile
	for _, f := range files {
		kind := f.kind
		want = append(want, renderFile{
			Path: f.path, Output: f.output, Item: f.item, Source: "template", Kind: &kind,
			Size: int64(len(f.path) + 1), Digest: digestOf(f.path + "\n"), Executable: f.executable, Status: status,
		})
	}
	if len(got) != len(want) {
		t.Fatalf("files %+v, want %+v", got, want)
	}
	for i := range want {
		g, w := got[i], want[i]
		if g.Path != w.Path || g.Output != w.Output || (g.Item == nil) != (w.Item == nil) || g.Item != nil && *g.Item != *w.Item ||
			g.Source != w.Source || g.Kind == nil || *g.Kind != *w.Kind || g.Size != w.Size || g.Digest != w.Digest || g.Executable != w.Executable || g.Status != w.Status {
			t.Errorf("file %d: %+v, want %+v", i, g, w)
		}
	}
}

// hostSnapshot is every name below the package's root, the home directory
// and the temporary directory, which the test gives the process.
type hostSnapshot struct{ roots []string }

// isolateHost gives the test a fresh home and temporary directory, and
// returns what a snapshot of the host covers beside root. The test must
// not be parallel.
func isolateHost(t *testing.T, root string) hostSnapshot {
	t.Helper()
	home := testenv.Home(t)
	tmp := testenv.Dir(t)
	t.Setenv("TMPDIR", tmp)
	return hostSnapshot{roots: []string{root, home, tmp}}
}

func (h hostSnapshot) take(t *testing.T) [][]string {
	t.Helper()
	var out [][]string
	for _, r := range h.roots {
		out = append(out, snapshotNames(t, r))
	}
	return out
}

// TestRenderDryRun is ADR-005 D16: a dry run into a directory renders
// everything, lists every top-level file in path order with D16 item 2's
// record and a status, headed would write in human form, and writes
// nothing anywhere: no output, no staging directory, no temporary file
// and no directory, checked on the host rather than in the result. A fan-
// out's files carry their item index (the meeting with W5-4).
func TestRenderDryRun(t *testing.T) {
	pkg, data, files := streamPackage(t)
	root := filepath.Dir(filepath.Dir(pkg))
	host := isolateHost(t, root)
	out := filepath.Join(filepath.Dir(pkg), "a", "out")
	before := host.take(t)
	r := runTool(t, "render", pkg, "--data", data, "--out", out, "--dry-run", "--json")
	if r.status != 0 || !r.doc.OK || len(r.doc.Findings) != 0 {
		t.Fatalf("status %d, %s", r.status, r.stdout)
	}
	res := readRender(t, r)
	if res.Out != "directory" || !res.DryRun || res.Only != nil {
		t.Errorf("result %+v", res)
	}
	checkListing(t, res.Files, files, "create")
	if after := host.take(t); !slices.EqualFunc(before, after, slices.Equal) {
		t.Errorf("the dry run wrote to the host: %q, then %q", before, after)
	}

	h := runTool(t, "render", pkg, "--data", data, "--out", out, "--dry-run")
	want := "would write\n"
	for _, f := range res.Files {
		want += strconv.FormatInt(f.Size, 10) + "\t" + f.Digest + "\t" + string(finding.AppendText(nil, f.Path)) + "\n"
	}
	if h.status != 0 || h.stdout != want || h.stderr != "" {
		t.Errorf("human dry run: status %d, stdout %q, want %q; stderr %q", h.status, h.stdout, want, h.stderr)
	}
	if after := host.take(t); !slices.EqualFunc(before, after, slices.Equal) {
		t.Errorf("the human dry run wrote to the host: %q, then %q", before, after)
	}

	// With --out -, the listing is what the stream would carry, with no
	// status, and the document is on standard output, since no bytes are.
	s := runTool(t, "render", pkg, "--data", data, "--out", "-", "--dry-run", "--json")
	if s.status != 0 || s.stderr != "" {
		t.Fatalf("--out - --dry-run: status %d, %q", s.status, s.stderr)
	}
	res = readRender(t, s)
	if res.Out != "stdout" || !res.DryRun {
		t.Errorf("result %+v", res)
	}
	checkListing(t, res.Files, files, "")
	if after := host.take(t); !slices.EqualFunc(before, after, slices.Equal) {
		t.Errorf("the dry run to standard output wrote to the host: %q, then %q", before, after)
	}
}

// TestRenderDryRunRefusedAsRender is ADR-005 D16 item 3 and ADR-008 D8
// item 2: --out stays required, and a dry run is refused exactly when the
// render is, with the same findings and exit status, writing nothing.
func TestRenderDryRunRefusedAsRender(t *testing.T) {
	pkg, data, _ := streamPackage(t)
	root := filepath.Dir(filepath.Dir(pkg))
	full := filepath.Join(root, "full")
	if err := os.MkdirAll(filepath.Join(full, "x"), 0o755); err != nil {
		t.Fatal(err)
	}
	broken, bdata := declarationPackage(t, `  - id: bad
    template: t.txt.liquid
    path: "{{ inputs.pages | nope }}.txt"
`)
	for _, c := range []struct {
		name string
		args []string
	}{
		{"no --out", []string{"render", pkg, "--data", data}},
		{"a destination that holds entries", []string{"render", pkg, "--data", data, "--out", full}},
		{"a refused path template", []string{"render", broken, "--data", bdata, "--out", filepath.Join(root, "o")}},
		{"a refused path template to standard output", []string{"render", broken, "--data", bdata, "--out", "-"}},
		{"no inputs", []string{"render", pkg, "--out", filepath.Join(root, "o")}},
	} {
		t.Run(c.name, func(t *testing.T) {
			before := snapshotNames(t, root)
			real := runTool(t, append(c.args, "--json")...)
			dry := runTool(t, append(c.args, "--dry-run", "--json")...)
			if real.status == 0 || dry.status != real.status || !slices.Equal(dry.codes(), real.codes()) {
				t.Errorf("render: status %d, codes %q; dry run: status %d, codes %q", real.status, real.codes(), dry.status, dry.codes())
			}
			if after := snapshotNames(t, root); !slices.Equal(before, after) {
				t.Errorf("changed the tree: %q, then %q", before, after)
			}
		})
	}
}

// TestRenderArchive is ADR-005 D17: --out - writes every top-level file
// as one pax interchange archive in path order, with D17 item 3's header
// fields and two terminating zero blocks, read back here by archive/tar;
// the document goes to standard error, with out stdout and no status.
func TestRenderArchive(t *testing.T) {
	pkg, data, files := streamPackage(t)
	r := runTool(t, "render", pkg, "--data", data, "--out", "-", "--json")
	if r.status != 0 || !r.doc.OK {
		t.Fatalf("status %d, %s", r.status, r.stderr)
	}
	res := readRender(t, r)
	if res.Out != "stdout" || res.DryRun || res.Only != nil {
		t.Errorf("result %+v", res)
	}
	checkListing(t, res.Files, files, "")
	b := []byte(r.stdout)
	if len(b) < 1024 || len(b)%512 != 0 || !bytes.Equal(b[len(b)-1024:], make([]byte, 1024)) {
		t.Fatalf("the archive of %d bytes does not end in two zero blocks", len(b))
	}
	tr := tar.NewReader(bytes.NewReader(b))
	for i := 0; ; i++ {
		h, err := tr.Next()
		if errors.Is(err, io.EOF) {
			if i != len(files) {
				t.Errorf("%d entries, want %d", i, len(files))
			}
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		if i >= len(files) {
			t.Fatalf("entry %q is extra", h.Name)
		}
		f := files[i]
		mode := int64(0o644)
		if f.executable {
			mode = 0o755
		}
		body, _ := io.ReadAll(tr)
		if h.Name != f.path || h.Typeflag != tar.TypeReg || h.Mode != mode || h.Uid != 0 || h.Gid != 0 ||
			h.Uname != "" || h.Gname != "" || h.ModTime.Unix() != 315532800 || h.Devmajor != 0 || h.Devminor != 0 ||
			string(body) != f.path+"\n" || h.Size != int64(len(body)) {
			t.Errorf("entry %d: %+v, %q", i, h, body)
		}
		if len(f.path) <= 100 && h.Format != tar.FormatUSTAR || len(f.path) > 100 && h.Format != tar.FormatPAX {
			t.Errorf("entry %q is in format %v", f.path, h.Format)
		}
	}
	h := runTool(t, "render", pkg, "--data", data, "--out", "-")
	if h.status != 0 || h.stdout != r.stdout || h.stderr != "" {
		t.Errorf("human: status %d, %d bytes, stderr %q", h.status, len(h.stdout), h.stderr)
	}
}

// TestRenderOnly is ADR-005 D18: --only selects one top-level file by the
// path the dry run lists. With --out - its bytes are written raw; with
// --out <dir> it is the one file written; a path no file has is
// usage.unknown_output, listing the paths; and the whole render runs, so
// a render refused for another file is refused with --only too.
func TestRenderOnly(t *testing.T) {
	pkg, data, files := streamPackage(t)
	root := filepath.Dir(filepath.Dir(pkg))
	sel := files[3]

	r := runTool(t, "render", pkg, "--data", data, "--out", "-", "--only", sel.path, "--json")
	if r.status != 0 || r.stdout != sel.path+"\n" {
		t.Fatalf("--out - --only: status %d, stdout %q, %s", r.status, r.stdout, r.stderr)
	}
	res := readRender(t, r)
	if res.Out != "stdout" || res.Only == nil || *res.Only != sel.path {
		t.Errorf("result %+v", res)
	}
	checkListing(t, res.Files, files[3:], "")

	out := filepath.Join(root, "out")
	r = runTool(t, "render", pkg, "--data", data, "--out", out, "--only="+sel.path, "--json")
	if r.status != 0 {
		t.Fatalf("--out <dir> --only: status %d, %s", r.status, r.stdout)
	}
	checkListing(t, readRender(t, r).Files, files[3:], "create")
	if got := snapshotNames(t, out); !slices.Equal(got, []string{".", "run.sh"}) {
		t.Errorf("wrote %q", got)
	}
	if fi, err := os.Stat(filepath.Join(out, "run.sh")); err != nil || runtime.GOOS != "windows" && fi.Mode()&0o100 == 0 {
		t.Errorf("run.sh: %v, %v", fi, err)
	}

	// A dry run with --only lists the file selected alone.
	r = runTool(t, "render", pkg, "--data", data, "--out", filepath.Join(root, "dry"), "--only", files[0].path, "--dry-run", "--json")
	if r.status != 0 {
		t.Fatalf("--dry-run --only: status %d, %s", r.status, r.stdout)
	}
	checkListing(t, readRender(t, r).Files, files[:1], "create")

	before := snapshotNames(t, root)
	for _, args := range [][]string{
		{"render", pkg, "--data", data, "--out", "-", "--only", "pages/A.txt", "--json"},
		{"render", pkg, "--data", data, "--out", filepath.Join(root, "o"), "--only", "page", "--json"},
	} {
		r := runTool(t, args...)
		if r.status != finding.ExitUsage || !slices.Equal(r.codes(), []string{string(finding.CodeUsageUnknownOutput)}) {
			t.Fatalf("metais %q: status %d, codes %q", args, r.status, r.codes())
		}
		p := r.doc.Findings[0].Params
		var paths []string
		for _, v := range p["paths"].([]any) {
			paths = append(paths, v.(string))
		}
		var want []string
		for _, f := range files {
			want = append(want, f.path)
		}
		if !slices.Equal(paths, want) || p["path_count"] != float64(len(files)) || p["argument"] != float64(slices.Index(args, "--only")+2) {
			t.Errorf("metais %q: params %v", args, p)
		}
		if args[4] == "-" && r.stdout != "" {
			t.Errorf("metais %q wrote %q to standard output", args, r.stdout)
		}
	}
	if after := snapshotNames(t, root); !slices.Equal(before, after) {
		t.Errorf("a refused --only changed the tree: %q, then %q", before, after)
	}

	// The whole render runs: a file --only does not select still refuses it.
	broken, bdata := declarationPackage(t, `  - id: good
    template: t.txt.liquid
    path: good.txt
  - id: bad
    template: t.txt.liquid
    path: "{{ inputs.pages | nope }}.txt"
`)
	full := runTool(t, "render", broken, "--data", bdata, "--out", "-", "--json")
	for _, out := range []string{"-", filepath.Join(root, "b")} {
		r := runTool(t, "render", broken, "--data", bdata, "--out", out, "--only", "good.txt", "--json")
		if r.status == 0 || r.status != full.status || !slices.Equal(r.codes(), full.codes()) || r.stdout != "" && out == "-" {
			t.Errorf("--out %s --only good.txt of a refused render: status %d, codes %q; the render's %d, %q", out, r.status, r.codes(), full.status, full.codes())
		}
	}
}

// TestRenderArchiveToTerminal is ADR-008 D8 item 4: an archive is not
// written to a terminal, refused with usage.archive_to_terminal and its
// next step; --only's raw bytes, and a dry run, may go to one. The
// terminal is isTerminal's seam: a pseudo-terminal is not offered to
// every host this suite runs on (plan §5).
func TestRenderArchiveToTerminal(t *testing.T) {
	pkg, data, files := streamPackage(t)
	saved := isTerminal
	t.Cleanup(func() { isTerminal = saved })
	isTerminal = func(any) bool { return true }
	for _, json := range []bool{true, false} {
		args := []string{"render", pkg, "--data", data, "--out", "-"}
		if json {
			args = append(args, "--json")
		}
		r := runTool(t, args...)
		if r.status != finding.ExitUsage || r.stdout != "" || !strings.Contains(r.stderr, "usage.archive_to_terminal") {
			t.Errorf("metais %q: status %d, stdout %q, stderr %q", args, r.status, r.stdout, r.stderr)
		}
		if json && (r.doc.Findings[0].Params["argument"] != float64(6) || len(r.doc.Next) != 1 || !strings.Contains(r.doc.Next[0].Run, "--only")) {
			t.Errorf("metais %q: %+v", args, r.doc)
		}
	}
	r := runTool(t, "render", pkg, "--data", data, "--out", "-", "--only", files[0].path)
	if r.status != 0 || r.stdout != files[0].path+"\n" {
		t.Errorf("--only to a terminal: status %d, %q, %q", r.status, r.stdout, r.stderr)
	}
	r = runTool(t, "render", pkg, "--data", data, "--out", "-", "--dry-run")
	if r.status != 0 || !strings.HasPrefix(r.stdout, "would write\n") {
		t.Errorf("--dry-run to a terminal: status %d, %q, %q", r.status, r.stdout, r.stderr)
	}
}

// purityProblems is the standard-output purity detector of ADR-008 D16
// item 5, run with faults injected: render --out -'s standard output is
// the archive byte for byte, with --json and without, --only's is the
// file's bytes, and a refused render's is empty. It returns each case
// whose standard output is anything else.
func purityProblems(t *testing.T, faults fault.Set) []string {
	t.Helper()
	pkg, data, files := streamPackage(t)
	broken, bdata := declarationPackage(t, `  - id: bad
    template: t.txt.liquid
    path: "{{ inputs.pages | nope }}.txt"
`)
	archive := string(wantArchive(t, files))
	warns, wdata, warnArchive := warningPackage(t)
	var bad []string
	for _, c := range []struct {
		name   string
		args   []string
		stdout string
	}{
		{"the archive, with --json", []string{"render", pkg, "--data", data, "--out", "-", "--json"}, archive},
		{"the archive", []string{"render", pkg, "--data", data, "--out", "-"}, archive},
		{"one file, with --json", []string{"render", pkg, "--data", data, "--out", "-", "--only", "run.sh", "--json"}, "run.sh\n"},
		{"the archive with warnings present, with --json", []string{"render", warns, "--data", wdata, "--out", "-", "--json"}, warnArchive},
		{"the archive with warnings present", []string{"render", warns, "--data", wdata, "--out", "-"}, warnArchive},
		{"a refused render, with --json", []string{"render", broken, "--data", bdata, "--out", "-", "--json"}, ""},
		{"a refused render", []string{"render", broken, "--data", bdata, "--out", "-"}, ""},
		{"a refused --only, with --json", []string{"render", pkg, "--data", data, "--out", "-", "--only", "x", "--json"}, ""},
		{"a refused line, with --json", []string{"render", pkg, "--out", "-", "--now", "today", "--json"}, ""},
	} {
		r := runIn(t, hostenv.Display{NoColor: true}, failingReader{t}, faults, c.args...)
		if r.stdout != c.stdout {
			bad = append(bad, c.name+": standard output is not the bytes alone:\n"+firstBytes(r.stdout))
		}
	}
	return bad
}

// firstBytes is the start of s, quoted.
func firstBytes(s string) string {
	return strings.ReplaceAll(strings.TrimRight(s[:min(len(s), 64)], "\x00"), "\x00", `\0`)
}

// TestStdoutPurity is ADR-008 D16 item 5's standard-output purity: every
// case of the detector is clean on the tool, and every one is caught on a
// build that writes a notice to standard output (fault.StdoutNotice).
// "With warnings present" is a render that warns output.container.space
// (W5-6).
func TestStdoutPurity(t *testing.T) {
	for _, p := range purityProblems(t, 0) {
		t.Error(p)
	}
	if n := len(purityProblems(t, fault.StdoutNotice)); n != 9 {
		t.Errorf("with StdoutNotice, the detector caught %d of 9 cases", n)
	}
}

// TestRenderContainerStreams is where the dry run, the archive and --only
// meet containers (W5-6): a container is listed with its entries, its
// size and digest those of its zip; it is one archive entry holding the
// zip a directory render writes; and --only selects it whole, its zip
// written raw.
func TestRenderContainerStreams(t *testing.T) {
	pkg, data := declarationPackage(t, `  - id: box
    path: box.zip
    container:
      entries:
        - id: page
          template: t.txt.liquid
          for_each: inputs.pages
          path: "p/{{ item }}.txt"
`)
	root := filepath.Dir(filepath.Dir(pkg))
	out := filepath.Join(root, "out")
	if r := runTool(t, "render", pkg, "--data", data, "--out", out, "--json"); r.status != 0 {
		t.Fatalf("render: status %d, %s", r.status, r.stdout)
	}
	zip, err := os.ReadFile(filepath.Join(out, "box.zip"))
	if err != nil {
		t.Fatal(err)
	}

	r := runTool(t, "render", pkg, "--data", data, "--out", filepath.Join(root, "dry"), "--dry-run", "--json")
	files := readRender(t, r).Files
	if r.status != 0 || len(files) != 1 {
		t.Fatalf("dry run: status %d, %+v", r.status, files)
	}
	box := files[0]
	if box.Path != "box.zip" || box.Source != "container" || box.Kind != nil || box.Size != int64(len(zip)) ||
		box.Digest != digestOf(string(zip)) || box.Status != "create" || len(box.Entries) != 3 {
		t.Errorf("the container's record: %+v", box)
	}
	for i, e := range box.Entries {
		p := "p/" + []string{"a", "b", "c"}[i] + ".txt"
		if e.Path != p || e.Output == nil || *e.Output != "page" || e.Item == nil || *e.Item != i ||
			e.Size != int64(len(p)+1) || e.Digest != digestOf(p+"\n") {
			t.Errorf("entry %d: %+v", i, e)
		}
	}

	r = runTool(t, "render", pkg, "--data", data, "--out", "-", "--json")
	tr := tar.NewReader(strings.NewReader(r.stdout))
	h, err := tr.Next()
	if r.status != 0 || err != nil || h.Name != "box.zip" || h.Size != int64(len(zip)) {
		t.Fatalf("archive: status %d, %v, %+v", r.status, err, h)
	}
	if b, _ := io.ReadAll(tr); !bytes.Equal(b, zip) {
		t.Error("the archive's container entry is not the zip a directory render writes")
	}
	if _, err := tr.Next(); !errors.Is(err, io.EOF) {
		t.Errorf("the archive holds more than the container: %v", err)
	}

	r = runTool(t, "render", pkg, "--data", data, "--out", "-", "--only", "box.zip", "--json")
	if r.status != 0 || r.stdout != string(zip) {
		t.Errorf("--only box.zip: status %d, %d bytes, want the zip's %d", r.status, len(r.stdout), len(zip))
	}
}

// warningPackage is a package whose render warns, output.container.space
// for an EPUB entry whose path has a space, its data directory, and the
// archive its render writes: built here from the files a render into a
// directory writes, after checking that the render warns.
func warningPackage(t *testing.T) (pkg, data, archive string) {
	t.Helper()
	pkg, data = declarationPackage(t, `  - id: book
    path: book.epub
    container:
      profile: epub
      entries:
        - template: t.txt.liquid
          path: META-INF/container.xml
        - template: t.txt.liquid
          path: OEBPS/first page.xhtml
  - id: note
    template: t.txt.liquid
    path: note.txt
`)
	out := filepath.Join(filepath.Dir(pkg), "warned")
	r := runTool(t, "render", pkg, "--data", data, "--out", out, "--json")
	if r.status != 0 || !slices.Equal(r.codes(), []string{"output.container.space"}) {
		t.Fatalf("the warning package: status %d, codes %q", r.status, r.codes())
	}
	var b bytes.Buffer
	w := tarw.NewWriter(&b)
	for _, name := range []string{"book.epub", "note.txt"} {
		body, err := os.ReadFile(filepath.Join(out, name))
		if err != nil {
			t.Fatal(err)
		}
		if err := w.Add(name, false, int64(len(body)), bytes.NewReader(body)); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	return pkg, data, b.String()
}

// TestRenderAssetStreams is where the dry run, the archive and --only
// meet assets (W5-5): an asset is listed with source asset, no kind and
// the digest of the package's bytes, which a dry run reads for it; it is
// an archive entry holding those bytes, in a container too, read from a
// package still open as the stream is written; and --only writes it raw.
func TestRenderAssetStreams(t *testing.T) {
	pkg, data := declarationPackage(t, `  - id: logo
    asset: logo.svg
  - id: box
    path: box.zip
    container:
      entries:
        - asset: logo.svg
`)
	logo := "<svg/>\n"
	if err := os.MkdirAll(filepath.Join(pkg, "assets"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pkg, "assets", "logo.svg"), []byte(logo), 0o644); err != nil {
		t.Fatal(err)
	}
	root := filepath.Dir(filepath.Dir(pkg))

	r := runTool(t, "render", pkg, "--data", data, "--out", filepath.Join(root, "dry"), "--dry-run", "--json")
	files := readRender(t, r).Files
	if r.status != 0 || len(files) != 2 {
		t.Fatalf("dry run: status %d, %+v", r.status, files)
	}
	if f := files[1]; f.Path != "logo.svg" || f.Source != "asset" || f.Kind != nil || f.Size != int64(len(logo)) || f.Digest != digestOf(logo) {
		t.Errorf("the asset's record: %+v", f)
	}

	r = runTool(t, "render", pkg, "--data", data, "--out", "-", "--json")
	if r.status != 0 {
		t.Fatalf("archive: status %d, %s", r.status, r.stderr)
	}
	tr := tar.NewReader(strings.NewReader(r.stdout))
	for _, name := range []string{"box.zip", "logo.svg"} {
		h, err := tr.Next()
		if err != nil || h.Name != name {
			t.Fatalf("archive entry %q: %v, %+v", name, err, h)
		}
		b, _ := io.ReadAll(tr)
		if name == "logo.svg" && string(b) != logo || name == "box.zip" && !bytes.Contains(b, []byte(logo)) {
			t.Errorf("archive entry %q holds %q", name, b)
		}
	}
	if f := readRender(t, r).Files[1]; f.Digest != digestOf(logo) {
		t.Errorf("the archive's record of the asset: %+v", f)
	}

	r = runTool(t, "render", pkg, "--data", data, "--out", "-", "--only", "logo.svg", "--json")
	if r.status != 0 || r.stdout != logo {
		t.Errorf("--only logo.svg: status %d, %q", r.status, r.stdout)
	}
}
