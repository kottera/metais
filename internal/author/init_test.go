// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package author

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"metais.org/metais/internal/hostfs"
	"metais.org/metais/internal/testenv"
)

// tree returns every entry beneath dir, "/"-separated, directories with a
// trailing /, in byte order.
func tree(t *testing.T, dir string) []string {
	t.Helper()
	var out []string
	err := filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil || p == dir {
			return err
		}
		rel, _ := filepath.Rel(dir, p)
		rel = filepath.ToSlash(rel)
		if d.IsDir() {
			rel += "/"
		}
		out = append(out, rel)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	slices.Sort(out)
	return out
}

// scaffoldTree is what Init leaves in an empty directory.
var scaffoldTree = []string{
	".gitattributes", "examples/", "examples/basic/", "examples/basic/example.yaml",
	"examples/basic/expected/", "examples/basic/expected/greeting.md",
	"examples/basic/inputs/", "examples/basic/inputs/greeting.json",
	"inputs/", "inputs/greeting.md", "inputs/greeting.schema.json",
	"metais.yaml", "templates/", "templates/greeting.md.liquid",
}

// probeModes returns the modes a file created 0644 and a directory created
// 0755 get in dir, less the process umask, whatever this platform makes of
// them.
func probeModes(t *testing.T, dir string) (file, directory fs.FileMode) {
	t.Helper()
	f := filepath.Join(dir, "probe-file")
	if err := os.WriteFile(f, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	d := filepath.Join(dir, "probe-dir")
	if err := os.Mkdir(d, 0o755); err != nil {
		t.Fatal(err)
	}
	fi, _ := os.Stat(f)
	di, _ := os.Stat(d)
	return fi.Mode().Perm(), di.Mode().Perm()
}

// TestInitWrites is W3-8.1 and W3-8.3 on a directory that does not exist:
// Init creates it with its parents and writes exactly the scaffold, each
// file's bytes the golden file's, files 0644 and directories 0755 less the
// umask, and returns the paths in the order written.
func TestInitWrites(t *testing.T) {
	root := testenv.Dir(t)
	fileMode, dirMode := probeModes(t, testenv.Dir(t))
	dir := filepath.Join(root, "a", "b", goldenName)
	got, err := Init(dir, goldenName, Form{}, false)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(got, scaffoldOrder) {
		t.Errorf("Init returned %q, want %q", got, scaffoldOrder)
	}
	if tr := tree(t, dir); !slices.Equal(tr, scaffoldTree) {
		t.Errorf("Init left %q, want %q", tr, scaffoldTree)
	}
	for _, f := range readGolden(t, goldenForms[0].dir, scaffoldOrder) {
		p := filepath.Join(dir, filepath.FromSlash(f.Path))
		b, err := os.ReadFile(p)
		if err != nil || string(b) != string(f.Data) {
			t.Errorf("%s is not the golden file (%v)", f.Path, err)
		}
		if fi, err := os.Stat(p); err != nil || fi.Mode().Perm() != fileMode {
			t.Errorf("%s has mode %v, want %v", f.Path, fi.Mode().Perm(), fileMode)
		}
	}
	for _, d := range []string{"a", "a/b", "a/b/" + goldenName} {
		if fi, err := os.Stat(filepath.Join(root, filepath.FromSlash(d))); err != nil || fi.Mode().Perm() != dirMode {
			t.Errorf("%s has mode %v, want %v", d, fi.Mode().Perm(), dirMode)
		}
	}
	for _, e := range scaffoldTree {
		if d, ok := strings.CutSuffix(e, "/"); ok {
			if fi, err := os.Stat(filepath.Join(dir, filepath.FromSlash(d))); err != nil || fi.Mode().Perm() != dirMode {
				t.Errorf("%s has mode %v, want %v", d, fi.Mode().Perm(), dirMode)
			}
		}
	}
}

// TestInitQuiet is W3-8.3: a directory holding only quiet entries counts
// as empty, so git init followed by metais init works, and the quiet
// entries are left as they were.
func TestInitQuiet(t *testing.T) {
	dir := testenv.Dir(t)
	quiet := []string{".DS_Store", ".git/", ".git/HEAD", ".gitignore", "old-0.1.0.metais"}
	plant(t, dir, quiet...)
	if _, err := Init(dir, goldenName, Form{}, false); err != nil {
		t.Fatal(err)
	}
	want := slices.Sorted(slices.Values(append(slices.Clone(scaffoldTree), quiet...)))
	if tr := tree(t, dir); !slices.Equal(tr, want) {
		t.Errorf("Init left %q, want %q", tr, want)
	}
}

// plant makes each entry beneath dir: a directory when it ends in /, else
// a file holding its own name.
func plant(t *testing.T, dir string, entries ...string) {
	t.Helper()
	for _, e := range entries {
		p := filepath.Join(dir, filepath.FromSlash(strings.TrimSuffix(e, "/")))
		var err error
		if strings.HasSuffix(e, "/") {
			err = os.MkdirAll(p, 0o755)
		} else {
			err = os.WriteFile(p, []byte(e), 0o644)
		}
		if err != nil {
			t.Fatal(err)
		}
	}
}

// TestInitNotEmpty is W3-8.3's refusal: anything but a quiet entry, and a
// quiet entry the scaffold would write, makes the directory not empty;
// Init lists up to ten of them in byte order and writes nothing, with
// --dry-run or without.
func TestInitNotEmpty(t *testing.T) {
	for _, c := range []struct {
		name    string
		planted []string
		listed  []string
		more    bool
	}{
		{"a file", []string{"README.md", ".git/"}, []string{"README.md"}, false},
		{"compiled", []string{".compiled/"}, []string{".compiled"}, false},
		{"a .metais directory", []string{"x.metais/"}, []string{"x.metais"}, false},
		{"a .gitattributes of its own", []string{".gitattributes", ".gitignore"}, []string{".gitattributes"}, false},
		{"a scaffold name", []string{"metais.yaml"}, []string{"metais.yaml"}, false},
		{"eleven", []string{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j", "k", ".git/"},
			[]string{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j"}, true},
		{"exactly ten", []string{"j", "i", "h", "g", "f", "e", "d", "c", "b", "a"},
			[]string{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j"}, false},
	} {
		for _, dry := range []bool{false, true} {
			dir := testenv.Dir(t)
			plant(t, dir, c.planted...)
			before := tree(t, dir)
			files, err := Init(dir, goldenName, Form{}, dry)
			var ne *NotEmptyError
			if !errors.As(err, &ne) {
				t.Errorf("%s (dry run %v): Init = %q, %v; want a refusal", c.name, dry, files, err)
				continue
			}
			if !slices.Equal(ne.Entries, c.listed) || ne.More != c.more || files != nil {
				t.Errorf("%s: listed %q, more %v; want %q, %v", c.name, ne.Entries, ne.More, c.listed, c.more)
			}
			if after := tree(t, dir); !slices.Equal(before, after) {
				t.Errorf("%s: the refusal wrote %q", c.name, after)
			}
		}
	}
}

// TestInitDryRun is W3-8.4: --dry-run reports what Init would write and
// writes nothing, into an absent directory or an empty one.
func TestInitDryRun(t *testing.T) {
	root := testenv.Dir(t)
	absent := filepath.Join(root, "a", "b")
	got, err := Init(absent, goldenName, Form{}, true)
	if err != nil || !slices.Equal(got, scaffoldOrder) {
		t.Errorf("Init dry run = %q, %v", got, err)
	}
	if tr := tree(t, root); len(tr) != 0 {
		t.Errorf("a dry run wrote %q", tr)
	}
	got, err = Init(root, goldenName, Form{}, true)
	if err != nil || !slices.Equal(got, scaffoldOrder) {
		t.Errorf("Init dry run = %q, %v", got, err)
	}
	if tr := tree(t, root); len(tr) != 0 {
		t.Errorf("a dry run wrote %q", tr)
	}
}

// TestInitNotDirectory is a destination that is a file, or lies beneath
// one: refused, with nothing written.
func TestInitNotDirectory(t *testing.T) {
	root := testenv.Dir(t)
	plant(t, root, "file")
	for _, dir := range []string{"file", "file/below"} {
		_, err := Init(filepath.Join(root, filepath.FromSlash(dir)), goldenName, Form{}, false)
		var we *WriteError
		if !errors.As(err, &we) || we.Op != "open" || we.Reason != hostfs.NotDirectory || we.Written != nil {
			t.Errorf("Init into %s = %v, want a refusal of a non-directory", dir, err)
		}
	}
	if tr := tree(t, root); !slices.Equal(tr, []string{"file"}) {
		t.Errorf("the refusal wrote %q", tr)
	}
}

// TestInitFailedWriteLeavesNothing is a write into an existing empty
// directory whose first create fails: Init has created nothing, and its
// error says so, which the command line reports as nothing written.
func TestInitFailedWriteLeavesNothing(t *testing.T) {
	dir := testenv.Dir(t)
	beforeCreate = func(p string) {
		if p == scaffoldOrder[0] {
			plant(t, dir, p)
		}
	}
	t.Cleanup(func() { beforeCreate = nil })
	_, err := Init(dir, goldenName, Form{}, false)
	var we *WriteError
	if !errors.As(err, &we) || we.Reason != hostfs.Exists || len(we.Written) != 0 || we.Left {
		t.Fatalf("Init = %#v, want a write error that left nothing", err)
	}
}

// TestInitFailedWrite is W3-8.5 (ADR-007 D1 item 6): a write that fails
// part way stops Init, which reports the files already written, in order,
// and removes nothing, neither the files nor the directories it made.
// The write is made to fail by putting a file where Init is about to
// create one, which Init never overwrites.
func TestInitFailedWrite(t *testing.T) {
	for i, stop := range scaffoldOrder {
		dir := filepath.Join(testenv.Dir(t), "pkg")
		beforeCreate = func(p string) {
			if p == stop {
				plant(t, dir, stop)
			}
		}
		written, err := Init(dir, goldenName, Form{}, false)
		beforeCreate = nil
		var we *WriteError
		if !errors.As(err, &we) || we.Reason != hostfs.Exists || !we.Left {
			// Init created dir, so the failure leaves something, a
			// file or not.
			t.Fatalf("a failed create of %s: Init = %#v, want a write error that left what it made", stop, err)
		}
		if want := scaffoldOrder[:i]; !slices.Equal(we.Written, want) || !slices.Equal(written, want) {
			t.Errorf("a failed create of %s reports %q written, want %q", stop, we.Written, want)
		}
		for _, f := range readGolden(t, goldenForms[0].dir, scaffoldOrder)[:i] {
			if b, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(f.Path))); err != nil || string(b) != string(f.Data) {
				t.Errorf("after a failed create of %s, %s was removed or changed", stop, f.Path)
			}
		}
		if b, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(stop))); err != nil || string(b) != stop {
			t.Errorf("Init replaced the file in its way at %s", stop)
		}
		for _, d := range []string{"inputs", "templates", "examples/basic/inputs"} {
			if j := slices.IndexFunc(scaffoldOrder, func(p string) bool { return strings.HasPrefix(p, d+"/") }); j < i {
				if _, err := os.Stat(filepath.Join(dir, filepath.FromSlash(d))); err != nil {
					t.Errorf("after a failed create of %s, the directory %s was removed", stop, d)
				}
			}
		}
	}
}

// TestNameFromDir is W3-8.2: the name is the last element of the directory
// as written, and there is none to take when it is omitted or ends in .
// or ..; nothing is looked up.
func TestNameFromDir(t *testing.T) {
	for dir, want := range map[string]string{
		"acme-report":              "acme-report",
		"a/b/acme-report":          "acme-report",
		"acme-report/":             "acme-report",
		"../acme-report":           "acme-report",
		"Acme Report":              "Acme Report",
		"":                         "",
		".":                        "",
		"..":                       "",
		"./":                       "",
		"a/.":                      "",
		"a/..":                     "",
		"a/../":                    "",
		string(filepath.Separator): "",
	} {
		got, ok := NameFromDir(dir)
		if got != want || ok != (want != "") {
			t.Errorf("NameFromDir(%q) = %q, %v; want %q", dir, got, ok, want)
		}
	}
}

// TestSuggestName is W3-8.2's suggested name: always one the rule accepts,
// and none when nothing of the name survives.
func TestSuggestName(t *testing.T) {
	for name, want := range map[string]string{
		"Acme Report":           "acme-report",
		"acme_report":           "acme-report",
		"--acme--report--":      "acme-report",
		"Org.Example..VPAT ACR": "org.example.vpat-acr",
		"con":                   "con-package",
		"COM1.reports":          "com1-package.reports",
		"café":                  "caf",
		strings.Repeat("a", 70): strings.Repeat("a", 63),
		"!!!":                   "",
		"":                      "",
		"...":                   "",
	} {
		got, ok := SuggestName(name)
		if got != want && ok || ok != (want != "") {
			t.Errorf("SuggestName(%q) = %q, %v; want %q", name, got, ok, want)
		}
		if ok && !ValidName(got) {
			t.Errorf("SuggestName(%q) = %q, which the rule refuses", name, got)
		}
	}
	long := strings.Repeat(strings.Repeat("a", 60)+".", 4)
	if got, ok := SuggestName(long); !ok || len(got) > 128 || !ValidName(got) {
		t.Errorf("SuggestName of a long name = %q, %v", got, ok)
	}
}

// TestIsQuiet is ADR-007 D7 item 3's quiet list.
func TestIsQuiet(t *testing.T) {
	for _, c := range []struct {
		e    hostfs.Entry
		want bool
	}{
		{hostfs.Entry{Name: ".git", Kind: hostfs.KindDir}, true},
		{hostfs.Entry{Name: ".gitignore", Kind: hostfs.KindFile}, true},
		{hostfs.Entry{Name: ".DS_Store", Kind: hostfs.KindFile}, true},
		{hostfs.Entry{Name: ".compiled", Kind: hostfs.KindDir}, false},
		{hostfs.Entry{Name: "x-1.0.0.metais", Kind: hostfs.KindFile}, true},
		{hostfs.Entry{Name: "x.metais", Kind: hostfs.KindDir}, false},
		{hostfs.Entry{Name: "README.md", Kind: hostfs.KindFile}, false},
	} {
		if got := IsQuiet(c.e); got != c.want {
			t.Errorf("IsQuiet(%+v) = %v", c.e, got)
		}
	}
}
