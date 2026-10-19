// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package author

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/kottera/metais/internal/hostfs"
	"github.com/kottera/metais/internal/pkgtree"
	"github.com/kottera/metais/internal/pkgtree/pkgtreetest"
	"github.com/kottera/metais/internal/testenv"
)

// packedFiles are a packed package's entries, .compiled/ included, as
// Unpack sees them: it reads a tree the loader verified.
var packedFiles = map[string][]byte{
	".compiled/digests.json":           []byte("{}\n"),
	".compiled/manifest.json":          []byte("{}\n"),
	".compiled/references.json":        []byte("[]\n"),
	".compiled/schema.json":            []byte("{}\n"),
	"assets/logo.txt":                  []byte("logo\n"),
	"examples/basic/inputs/r.json":     []byte("{}\n"),
	"inputs/r.schema.json":             []byte("{\"type\": \"object\"}\n"),
	"metais.yaml":                      []byte("# kept verbatim\nmetais: \"1.0\"\n"),
	"references/notes.md":              []byte("notes\r\n"),
	"templates/report.md.liquid":       []byte("{{ inputs.r }}\n"),
	"templates/site/index.html.liquid": []byte("<p></p>\n"),
}

// unpackedTree is what Unpack leaves of packedFiles.
var unpackedTree = []string{
	"assets/", "assets/logo.txt", "examples/", "examples/basic/", "examples/basic/inputs/",
	"examples/basic/inputs/r.json", "inputs/", "inputs/r.schema.json", "metais.yaml",
	"references/", "references/notes.md", "templates/", "templates/report.md.liquid",
	"templates/site/", "templates/site/index.html.liquid",
}

// unpackedFiles are the files of unpackedTree, in the order Unpack writes
// them.
var unpackedFiles = []string{
	"assets/logo.txt", "examples/basic/inputs/r.json", "inputs/r.schema.json", "metais.yaml",
	"references/notes.md", "templates/report.md.liquid", "templates/site/index.html.liquid",
}

func packedTree(t *testing.T) pkgtree.Tree {
	t.Helper()
	tr, err := pkgtreetest.New(packedFiles)
	if err != nil {
		t.Fatal(err)
	}
	return tr
}

// checkUnpacked fails unless dir holds exactly unpackedTree, each file's
// bytes the package's, files 0644 and directories 0755 less the umask.
func checkUnpacked(t *testing.T, dir string, fileMode, dirMode os.FileMode, besides ...string) {
	t.Helper()
	want := slices.Sorted(slices.Values(append(slices.Clone(unpackedTree), besides...)))
	if got := tree(t, dir); !slices.Equal(got, want) {
		t.Fatalf("unpacked %q, want %q", got, want)
	}
	for _, e := range unpackedTree {
		p := filepath.Join(dir, filepath.FromSlash(strings.TrimSuffix(e, "/")))
		fi, err := os.Lstat(p)
		if err != nil {
			t.Fatal(err)
		}
		mode := fileMode
		if strings.HasSuffix(e, "/") {
			mode = dirMode
		} else if b, _ := os.ReadFile(p); string(b) != string(packedFiles[e]) {
			t.Errorf("%s holds %q, want %q", e, b, packedFiles[e])
		}
		if fi.Mode().Perm() != mode {
			t.Errorf("%s: mode %v, want %v", e, fi.Mode().Perm(), mode)
		}
	}
}

// TestUnpackedEntries is ADR-007 D8 item 4's entry set: every entry but
// .compiled/'s, in byte order.
func TestUnpackedEntries(t *testing.T) {
	if got := UnpackedEntries(packedTree(t)); !slices.Equal(got, unpackedFiles) {
		t.Errorf("UnpackedEntries = %q, want %q", got, unpackedFiles)
	}
}

// TestUnpackIntoANewDirectory is W4-5.1 for a target that does not
// exist: it is created, with its missing parents, holding every entry but
// .compiled/, with ADR-007 D8 item 4's modes, and nothing else is left
// beside it: no staging directory.
func TestUnpackIntoANewDirectory(t *testing.T) {
	fileMode, dirMode := probeModes(t, testenv.Dir(t))
	root := testenv.Dir(t)
	to := filepath.Join(root, "a", "b", "src")
	got, err := Unpack(packedTree(t), to, 0)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(got, unpackedFiles) {
		t.Errorf("Unpack wrote %q, want %q", got, unpackedFiles)
	}
	checkUnpacked(t, to, fileMode, dirMode)
	if got := tree(t, filepath.Join(root, "a", "b")); got[0] != "src/" || slices.ContainsFunc(got, func(e string) bool { return !strings.HasPrefix(e, "src/") }) {
		t.Errorf("beside the target: %q", got)
	}
}

// TestUnpackIntoAnEmptyDirectory is W4-5.1 for an existing directory: one
// that is empty, or holds only quiet entries (ADR-007 D1 item 3), is
// written in place, and its quiet entries are left as they were.
func TestUnpackIntoAnEmptyDirectory(t *testing.T) {
	fileMode, dirMode := probeModes(t, testenv.Dir(t))
	t.Run("empty", func(t *testing.T) {
		to := testenv.Dir(t)
		if _, err := Unpack(packedTree(t), to, 0); err != nil {
			t.Fatal(err)
		}
		checkUnpacked(t, to, fileMode, dirMode)
	})
	t.Run("quiet", func(t *testing.T) {
		to := testenv.Dir(t)
		if err := os.MkdirAll(filepath.Join(to, ".git"), 0o755); err != nil {
			t.Fatal(err)
		}
		for _, f := range []string{".git/HEAD", ".gitattributes", "old-1.0.0.metais"} {
			if err := os.WriteFile(filepath.Join(to, f), []byte("theirs\n"), 0o644); err != nil {
				t.Fatal(err)
			}
		}
		if _, err := Unpack(packedTree(t), to, 0); err != nil {
			t.Fatal(err)
		}
		checkUnpacked(t, to, fileMode, dirMode, ".git/", ".git/HEAD", ".gitattributes", "old-1.0.0.metais")
		if b, _ := os.ReadFile(filepath.Join(to, ".git", "HEAD")); string(b) != "theirs\n" {
			t.Errorf(".git/HEAD holds %q", b)
		}
	})
}

// TestUnpackRefusesANonEmptyDirectory is R2-105 and R2-136: a target
// holding anything but quiet entries is refused, listing up to ten of
// them in byte order, and nothing is written. .compiled is not quiet.
func TestUnpackRefusesANonEmptyDirectory(t *testing.T) {
	for _, c := range []struct {
		name  string
		files []string
		want  []string
		more  bool
	}{
		{"one file", []string{"notes.txt", ".git"}, []string{"notes.txt"}, false},
		{"a compiled directory", []string{".compiled/x"}, []string{".compiled"}, false},
		{"a manifest", []string{"metais.yaml"}, []string{"metais.yaml"}, false},
		{"eleven", []string{"k", "j", "i", "h", "g", "f", "e", "d", "c", "b", "a"},
			[]string{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j"}, true},
	} {
		t.Run(c.name, func(t *testing.T) {
			to := testenv.Dir(t)
			for _, f := range c.files {
				p := filepath.Join(to, filepath.FromSlash(f))
				if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(p, []byte("theirs\n"), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			before := tree(t, to)
			files, err := Unpack(packedTree(t), to, 0)
			var ne *NotEmptyError
			if !errors.As(err, &ne) || !slices.Equal(ne.Entries, c.want) || ne.More != c.more || files != nil {
				t.Fatalf("Unpack: %v, %#v, %q; want the entries %q, more %v", err, ne, files, c.want, c.more)
			}
			if got := tree(t, to); !slices.Equal(got, before) {
				t.Errorf("after the refusal: %q, want %q", got, before)
			}
		})
	}
}

// TestUnpackRefusesANonDirectory is W4-5.1's last refusal: a target that
// is a file, or lies beneath one, is not a directory, and nothing is
// written.
func TestUnpackRefusesANonDirectory(t *testing.T) {
	root := testenv.Dir(t)
	file := filepath.Join(root, "file")
	if err := os.WriteFile(file, []byte("theirs\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, to := range []string{file, filepath.Join(file, "src")} {
		_, err := Unpack(packedTree(t), to, 0)
		var we *WriteError
		if !errors.As(err, &we) || we.Op != "open" || we.Reason != hostfs.NotDirectory || len(we.Written) != 0 {
			t.Errorf("Unpack(%q): %v, %#v", to, err, we)
		}
	}
	if got := tree(t, root); !slices.Equal(got, []string{"file"}) {
		t.Errorf("after the refusals: %q", got)
	}
}

// failingTree is a tree whose entry fail cannot be read past its first
// byte: a package that failed under the reader after it opened.
type failingTree struct {
	pkgtree.Tree
	fail string
}

func (f failingTree) Open(p string) (io.ReadCloser, error) {
	r, err := f.Tree.Open(p)
	if err != nil || p != f.fail {
		return r, err
	}
	return struct {
		io.Reader
		io.Closer
	}{io.MultiReader(io.LimitReader(r, 1), errReader{}), r}, nil
}

type errReader struct{}

func (errReader) Read([]byte) (int, error) { return 0, errors.New("the package changed") }

// TestUnpackFailureInANewDirectoryLeavesNothing is W4-5.1 and W4-5.3: a
// staged unpack that fails part way removes what it wrote, and only that,
// and reports no file written. A foreign file beside the target stays.
func TestUnpackFailureInANewDirectoryLeavesNothing(t *testing.T) {
	root := testenv.Dir(t)
	if err := os.WriteFile(filepath.Join(root, "theirs"), []byte("T"), 0o644); err != nil {
		t.Fatal(err)
	}
	tr := failingTree{packedTree(t), "references/notes.md"}
	files, err := Unpack(tr, filepath.Join(root, "a", "src"), 0)
	var we *WriteError
	if !errors.As(err, &we) || we.Reason != hostfs.IOError || len(we.Written) != 0 || files != nil {
		t.Fatalf("Unpack: %v, %#v, %q", err, we, files)
	}
	if got := tree(t, root); !slices.Equal(got, []string{"theirs"}) {
		t.Errorf("after the failure: %q, want only theirs", got)
	}
}

// TestUnpackFailureInPlaceReportsWhatItWrote is W4-5.1: an unpack into an
// existing empty directory that fails part way removes nothing it wrote
// but the file that failed, and reports the rest in the order written.
func TestUnpackFailureInPlaceReportsWhatItWrote(t *testing.T) {
	to := testenv.Dir(t)
	tr := failingTree{packedTree(t), "metais.yaml"}
	files, err := Unpack(tr, to, 0)
	written := unpackedFiles[:3]
	var we *WriteError
	if !errors.As(err, &we) || we.Reason != hostfs.IOError || !slices.Equal(we.Written, written) || !slices.Equal(files, written) {
		t.Fatalf("Unpack: %v, %#v, %q; want %q written", err, we, files, written)
	}
	want := []string{"assets/", "assets/logo.txt", "examples/", "examples/basic/", "examples/basic/inputs/",
		"examples/basic/inputs/r.json", "inputs/", "inputs/r.schema.json"}
	if got := tree(t, to); !slices.Equal(got, want) {
		t.Errorf("after the failure: %q, want %q", got, want)
	}
}

// TestUnpackRefusesANameThatAppears is a target written in place, in which
// a name the unpack needs appears after the emptiness check: it is
// refused with exists, never replaced.
func TestUnpackRefusesANameThatAppears(t *testing.T) {
	to := testenv.Dir(t)
	beforeCreate = func(p string) {
		if p == "metais.yaml" {
			os.WriteFile(filepath.Join(to, "metais.yaml"), []byte("theirs\n"), 0o644)
		}
	}
	t.Cleanup(func() { beforeCreate = nil })
	_, err := Unpack(packedTree(t), to, 0)
	var we *WriteError
	if !errors.As(err, &we) || we.Reason != hostfs.Exists || !slices.Equal(we.Written, unpackedFiles[:3]) {
		t.Fatalf("Unpack: %v, %#v", err, we)
	}
	if b, _ := os.ReadFile(filepath.Join(to, "metais.yaml")); string(b) != "theirs\n" {
		t.Errorf("metais.yaml holds %q", b)
	}
}

// TestUnpackRefusesAStagingLeftover is a target that does not exist
// beside the staging directory a killed unpack of it left: the unpack is
// refused with exists, writes nothing, and leaves the leftover alone.
func TestUnpackRefusesAStagingLeftover(t *testing.T) {
	root := testenv.Dir(t)
	leftover := filepath.Join(root, hostfs.StagingName("src"))
	if err := os.Mkdir(leftover, 0o755); err != nil {
		t.Fatal(err)
	}
	files, err := Unpack(packedTree(t), filepath.Join(root, "src"), 0)
	var we *WriteError
	if !errors.As(err, &we) || we.Reason != hostfs.Exists || len(we.Written) != 0 || files != nil {
		t.Fatalf("Unpack: %v, %#v, %q", err, we, files)
	}
	if got := tree(t, root); !slices.Equal(got, []string{hostfs.StagingName("src") + "/"}) {
		t.Errorf("after the refusal: %q", got)
	}
}

// TestUnpackDryRun is W5-12.3 in this package: for every target Unpack
// writes into or refuses, UnpackDryRun, run on a second copy of the same
// host, returns the files Unpack wrote, or refuses as Unpack refused, and
// changes nothing on its copy.
func TestUnpackDryRun(t *testing.T) {
	write := func(t *testing.T, root string, files ...string) {
		t.Helper()
		for _, f := range files {
			p := filepath.Join(root, filepath.FromSlash(f))
			if strings.HasSuffix(f, "/") {
				if err := os.MkdirAll(p, 0o755); err != nil {
					t.Fatal(err)
				}
				continue
			}
			if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(p, []byte("theirs\n"), 0o644); err != nil {
				t.Fatal(err)
			}
		}
	}
	for _, c := range []struct {
		name  string
		host  []string // beneath the root, "/"-separated; a trailing "/" is a directory
		to    string   // the target, beneath the root
		write bool     // whether Unpack writes rather than refuses
	}{
		{"a new directory", nil, "src", true},
		{"beneath missing parents", nil, "a/b/src", true},
		{"an empty directory", []string{"src/"}, "src", true},
		{"quiet entries", []string{".git/HEAD", ".gitattributes", "old-1.0.0.metais"}, ".", true},
		{"one entry", []string{"src/notes.txt", "src/.git"}, "src", false},
		{"a compiled directory", []string{"src/.compiled/x"}, "src", false},
		{"eleven entries", []string{"src/k", "src/j", "src/i", "src/h", "src/g", "src/f", "src/e", "src/d", "src/c", "src/b", "src/a"}, "src", false},
		{"a file", []string{"file"}, "file", false},
		{"beneath a file", []string{"file"}, "file/src", false},
		{"a staging leftover", []string{hostfs.StagingName("src") + "/"}, "src", false},
		{"a staging leftover beneath a missing parent", []string{"a/" + hostfs.StagingName("src") + "/"}, "a/src", false},
	} {
		t.Run(c.name, func(t *testing.T) {
			dry, wet := testenv.Dir(t), testenv.Dir(t)
			write(t, dry, c.host...)
			write(t, wet, c.host...)
			before := tree(t, dry)
			got, gerr := UnpackDryRun(packedTree(t), filepath.Join(dry, filepath.FromSlash(c.to)))
			want, werr := Unpack(packedTree(t), filepath.Join(wet, filepath.FromSlash(c.to)), 0)
			if after := tree(t, dry); !slices.Equal(after, before) {
				t.Errorf("the dry run changed the host: %q, was %q", after, before)
			}
			if (werr == nil) != c.write {
				t.Fatalf("Unpack: %v, %q", werr, want)
			}
			if !slices.Equal(got, want) {
				t.Errorf("the dry run lists %q, and Unpack wrote %q", got, want)
			}
			var gne, wne *NotEmptyError
			var gwe, wwe *WriteError
			switch {
			case werr == nil:
				if gerr != nil {
					t.Errorf("the dry run refused with %v, and Unpack wrote", gerr)
				}
			case errors.As(werr, &wne):
				if !errors.As(gerr, &gne) || !slices.Equal(gne.Entries, wne.Entries) || gne.More != wne.More {
					t.Errorf("the dry run refused with %#v, and Unpack with %#v", gerr, wne)
				}
			case errors.As(werr, &wwe):
				// Op is not compared: it names the step that failed, and
				// the dry run takes none.
				if !errors.As(gerr, &gwe) || gwe.Reason != wwe.Reason || len(gwe.Written) != 0 || len(wwe.Written) != 0 {
					t.Errorf("the dry run refused with %#v, and Unpack with %#v", gerr, wwe)
				}
				if wwe.Reason == hostfs.NotDirectory && (gwe.Op != "open" || wwe.Op != "open") {
					t.Errorf("not a directory at ops %q and %q, want open", gwe.Op, wwe.Op)
				}
			default:
				t.Fatalf("Unpack: %v", werr)
			}
		})
	}
}
