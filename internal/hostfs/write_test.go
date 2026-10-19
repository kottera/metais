// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package hostfs

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func openWriter(t *testing.T, path string) *Writer {
	t.Helper()
	w, err := OpenWriter(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { w.Close() })
	return w
}

func create(t *testing.T, w *Writer, name, body string) {
	t.Helper()
	if err := w.Create(name, false, strings.NewReader(body)); err != nil {
		t.Fatalf("Create(%q): %v", name, err)
	}
}

func readHost(t *testing.T, p string) string {
	t.Helper()
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// No write outside the root succeeds. The control shows each attack is
// real: a writer that joins the name onto the directory's path lands
// outside it. The detector is hostfs refusing the same names, and the
// directory outside is untouched.
func TestWritesAreConfined(t *testing.T) {
	root := dir(t)
	outside := testDirOutside(t)
	files(t, outside, map[string]string{"authorized_keys": "ORIGINAL"})
	symlink(t, root, outside, "ssh")
	attacks := []string{
		"ssh/authorized_keys",
		"ssh/new",
		"../" + filepath.Base(outside) + "/new",
		"ssh/sub/new",
	}

	// Control: string joining escapes through every attack but the one
	// whose parent does not exist.
	escaped := 0
	for _, a := range attacks {
		p := filepath.Join(root, filepath.FromSlash(a))
		if !strings.HasPrefix(filepath.Clean(p), filepath.Clean(root)+string(filepath.Separator)) || resolvesOutside(t, root, p) {
			escaped++
		}
	}
	if escaped < 3 {
		t.Fatalf("control: only %d of the attacks reach outside by string joining; the test proves nothing", escaped)
	}

	w := openWriter(t, root)
	for _, a := range attacks {
		if err := w.Create(a, false, strings.NewReader("PWNED")); err == nil {
			t.Errorf("Create(%q) succeeded", a)
		} else {
			noPath(t, err, root, outside)
		}
		if err := w.MkdirAll(a); err == nil {
			t.Errorf("MkdirAll(%q) succeeded", a)
		}
		if err := w.Mkdir(a); err == nil {
			t.Errorf("Mkdir(%q) succeeded", a)
		}
	}
	if _, err := w.Sub("ssh"); err == nil {
		t.Error("Sub through a link succeeded")
	}
	create(t, w, "staged", "x")
	for _, a := range attacks {
		if err := w.RenameReplace("staged", a); err == nil {
			t.Errorf("RenameReplace onto %q succeeded", a)
		}
		if err := w.RenameNoReplace("staged", a); err == nil {
			t.Errorf("RenameNoReplace onto %q succeeded", a)
		}
	}
	if got := tree(t, outside); !slices.Equal(got, []string{"authorized_keys"}) {
		t.Errorf("outside the root now holds %q", got)
	}
	if got := readHost(t, filepath.Join(outside, "authorized_keys")); got != "ORIGINAL" {
		t.Errorf("a file outside the root now holds %q", got)
	}
}

// resolvesOutside reports whether p, beneath root by its spelling, reaches
// outside root once links are followed.
func resolvesOutside(t *testing.T, root, p string) bool {
	t.Helper()
	real, err := filepath.EvalSymlinks(filepath.Dir(p))
	if err != nil {
		return false
	}
	rr, _ := filepath.EvalSymlinks(root)
	return !strings.HasPrefix(real+string(filepath.Separator), rr+string(filepath.Separator))
}

// A link inside the root is not written through either, even one whose
// target stays inside (ADR-009 D8 item 3).
func TestWritesRefuseLinksInside(t *testing.T) {
	root := dir(t)
	files(t, root, map[string]string{"real/keep": "KEEP", "target": "T"})
	symlink(t, root, "real", "dir")
	symlink(t, root, "target", "file")
	w := openWriter(t, root)
	err := w.Create("dir/new", false, strings.NewReader("x"))
	wantReason(t, err, Link, root)
	err = w.MkdirAll("dir/a/b")
	wantReason(t, err, Link, root)
	// An exclusive create does not follow a link at the end of the name.
	err = w.Create("file", false, strings.NewReader("x"))
	wantReason(t, err, Exists, root)
	create(t, w, "staged", "x")
	err = w.RenameReplace("staged", "file")
	wantReason(t, err, Link, root)
	if got := readHost(t, filepath.Join(root, "target")); got != "T" {
		t.Errorf("the link's target now holds %q", got)
	}
	if got := tree(t, filepath.Join(root, "real")); !slices.Equal(got, []string{"keep"}) {
		t.Errorf("the linked directory now holds %q", got)
	}
}

func TestCreateIsExclusive(t *testing.T) {
	root := dir(t)
	files(t, root, map[string]string{"old": "OLD", "d/x": ""})
	w := openWriter(t, root)
	wantReason(t, w.Create("old", false, strings.NewReader("new")), Exists, root)
	wantReason(t, w.Create("d", false, strings.NewReader("new")), Exists, root)
	wantReason(t, w.Mkdir("d"), Exists, root)
	wantReason(t, w.Mkdir("old"), Exists, root)
	wantReason(t, w.Create("old/x", false, strings.NewReader("")), NotDirectory, root)
	wantReason(t, w.Create("absent/x", false, strings.NewReader("")), NotFound, root)
	if got := readHost(t, filepath.Join(root, "old")); got != "OLD" {
		t.Errorf("an existing file now holds %q", got)
	}
}

func TestCreateModes(t *testing.T) {
	root := dir(t)
	w := openWriter(t, root)
	create(t, w, "plain", "x")
	if err := w.Create("tool", true, strings.NewReader("#!")); err != nil {
		t.Fatal(err)
	}
	if err := w.Mkdir("d"); err != nil {
		t.Fatal(err)
	}
	for name, want := range map[string]os.FileMode{"plain": 0o644, "tool": 0o755, "d": 0o755} {
		info, err := os.Lstat(filepath.Join(root, name))
		if err != nil {
			t.Fatal(err)
		}
		// The umask may only take bits away.
		if got := info.Mode().Perm(); got&^want != 0 || got&0o700 != want&0o700 {
			t.Errorf("%s: mode %v, want %v less the umask", name, got, want)
		}
	}
}

// A file whose bytes fail part way is not left behind.
func TestCreateFailureLeavesNothing(t *testing.T) {
	root := dir(t)
	w := openWriter(t, root)
	err := w.Create("half", false, failingReader{})
	wantReason(t, err, IOError, root)
	if got := tree(t, root); len(got) != 0 {
		t.Errorf("a failed create left %q", got)
	}
}

type failingReader struct{}

func (failingReader) Read(p []byte) (int, error) {
	copy(p, "part")
	return 4, os.ErrDeadlineExceeded
}

// The no-replace rename refuses an existing target, file or directory,
// and leaves both as they were. On this host it is the platform's own call
// (renameat2, renamex_np or MoveFileW).
func TestRenameNoReplaceRefusesAnExistingTarget(t *testing.T) {
	root := dir(t)
	files(t, root, map[string]string{"from": "NEW", "file": "OLD", "dir/x": "X"})
	w := openWriter(t, root)
	for _, target := range []string{"file", "dir"} {
		wantReason(t, w.RenameNoReplace("from", target), Exists, root)
	}
	if readHost(t, filepath.Join(root, "file")) != "OLD" || readHost(t, filepath.Join(root, "from")) != "NEW" {
		t.Error("a refused rename changed a file")
	}
	if err := w.RenameNoReplace("from", "to"); err != nil {
		t.Fatal(err)
	}
	if got := readHost(t, filepath.Join(root, "to")); got != "NEW" {
		t.Errorf("renamed file holds %q", got)
	}
}

// The refusal is the rename's own, not a check before it: a target put in
// place after every check, just before the call, is still refused.
func TestRenameNoReplaceRefusesARacingTarget(t *testing.T) {
	root := dir(t)
	w := openWriter(t, root)
	if err := w.Mkdir(StagingName("out")); err != nil {
		t.Fatal(err)
	}
	s, err := w.Sub(StagingName("out"))
	if err != nil {
		t.Fatal(err)
	}
	create(t, s, "index.html", "NEW")
	w.beforeRename = func(from, to string) {
		if err := os.Mkdir(filepath.Join(root, to), 0o755); err != nil {
			t.Fatal(err)
		}
		files(t, root, map[string]string{to + "/theirs": "THEIRS"})
	}
	wantReason(t, w.RenameNoReplace(StagingName("out"), "out"), Exists, root)
	if got := readHost(t, filepath.Join(root, "out", "theirs")); got != "THEIRS" {
		t.Errorf("the racing directory now holds %q", got)
	}
}

func TestRenameReplace(t *testing.T) {
	root := dir(t)
	files(t, root, map[string]string{"a/page.html": "OLD", "a/d/x": ""})
	w := openWriter(t, root)
	tmp := "a/" + TempName("page.html")
	create(t, w, tmp, "NEW")
	if err := w.RenameReplace(tmp, "a/page.html"); err != nil {
		t.Fatal(err)
	}
	if got := readHost(t, filepath.Join(root, "a", "page.html")); got != "NEW" {
		t.Errorf("replaced file holds %q", got)
	}
	create(t, w, tmp, "NEW")
	wantReason(t, w.RenameReplace(tmp, "a/d"), IsDirectory, root)
	wantReason(t, w.RenameReplace(tmp, "page.html"), BadName, root) // another directory
	wantReason(t, w.RenameReplace("a/absent", "a/x"), NotFound, root)
}

// ADR-005 D14 item 1, end to end: the output directory and two missing
// ancestors are absent; the render creates the ancestors, stages beside
// the output, and commits with the no-replace rename. On success nothing
// is removed.
func TestStagedRender(t *testing.T) {
	parent := dir(t)
	files(t, parent, map[string]string{"theirs": "T"})
	out := filepath.Join(parent, "a", "b", "site")

	var render func(meanwhile func()) error
	render = func(meanwhile func()) error {
		w, rest, err := OpenNearest(out)
		if err != nil {
			t.Fatal(err)
		}
		defer w.Close()
		if rest != "a/b/site" {
			t.Fatalf("OpenNearest rest %q", rest)
		}
		if meanwhile != nil {
			meanwhile()
		}
		dirName, base := split(rest)
		if err := w.MkdirAll(dirName); err != nil {
			return err
		}
		staging := join(dirName, StagingName(base))
		if err := w.Mkdir(staging); err != nil {
			return err
		}
		s, err := w.Sub(staging)
		if err != nil {
			return err
		}
		if err := s.MkdirAll("css"); err != nil {
			return err
		}
		create(t, s, "css/site.css", "a{}")
		create(t, s, "index.html", "<p>")
		if err := w.RenameNoReplace(staging, rest); err != nil {
			if rerr := w.RemoveCreated(); rerr != nil {
				t.Fatal(rerr)
			}
			return err
		}
		return nil
	}

	// A render that found the output absent, while another render of the
	// same output runs to its end: the second to commit is refused by
	// the rename, and removes only its own staging directory.
	var inner error
	err := render(func() { inner = render(nil) })
	if inner != nil {
		t.Fatal(inner)
	}
	wantReason(t, err, Exists, parent, out)
	want := []string{"a/", "a/b/", "a/b/site/", "a/b/site/css/", "a/b/site/css/site.css", "a/b/site/index.html", "theirs"}
	if got := tree(t, parent); !slices.Equal(got, want) {
		t.Fatalf("after both renders: %q, want %q", got, want)
	}
}

// ADR-005 D14 item 1.5: when the commit is refused, removing what the
// render created takes its staging directory and the ancestors it made,
// and nothing that was there before.
func TestRemoveCreatedRestoresTheParent(t *testing.T) {
	parent := dir(t)
	files(t, parent, map[string]string{"theirs": "T", "x/keep": "K"})
	before := tree(t, parent)
	w, rest, err := OpenNearest(filepath.Join(parent, "x", "new", "deeper", "out"))
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	if rest != "new/deeper/out" {
		t.Fatalf("rest %q, want new/deeper/out", rest)
	}
	if err := w.MkdirAll("new/deeper"); err != nil {
		t.Fatal(err)
	}
	staging := "new/deeper/" + StagingName("out")
	if err := w.Mkdir(staging); err != nil {
		t.Fatal(err)
	}
	s, err := w.Sub(staging)
	if err != nil {
		t.Fatal(err)
	}
	create(t, s, "a", "A")
	if err := s.MkdirAll("b/c"); err != nil {
		t.Fatal(err)
	}
	create(t, s, "b/c/d", "D")
	if err := w.RemoveCreated(); err != nil {
		t.Fatal(err)
	}
	if got := tree(t, parent); !slices.Equal(got, before) {
		t.Errorf("after removing the render's creations: %q, want %q", got, before)
	}
}

// Removal touches only what the render created. A name that existed
// before is never removed, a created name that now holds something else
// is left alone, a created directory that is not empty stays, and what the
// render committed stays.
func TestRemoveCreatedRemovesOnlyItsOwn(t *testing.T) {
	root := dir(t)
	files(t, root, map[string]string{"old/keep": "K", "before": "B"})
	w := openWriter(t, root)
	if err := w.MkdirAll("old/sub/deeper"); err != nil {
		t.Fatal(err)
	}
	create(t, w, "old/sub/deeper/mine", "M")
	create(t, w, "replaced", "MINE")
	if err := w.Mkdir("shared"); err != nil {
		t.Fatal(err)
	}
	create(t, w, "committed.tmp", "C")
	if err := w.RenameNoReplace("committed.tmp", "committed"); err != nil {
		t.Fatal(err)
	}
	// Someone else's work, after the render's.
	if err := os.Remove(filepath.Join(root, "replaced")); err != nil {
		t.Fatal(err)
	}
	files(t, root, map[string]string{"replaced": "THEIRS", "shared/theirs": "T"})

	if err := w.RemoveCreated(); err != nil {
		t.Fatal(err)
	}
	want := []string{"before", "committed", "old/", "old/keep", "replaced", "shared/", "shared/theirs"}
	if got := tree(t, root); !slices.Equal(got, want) {
		t.Fatalf("after RemoveCreated: %q, want %q", got, want)
	}
	if got := readHost(t, filepath.Join(root, "replaced")); got != "THEIRS" {
		t.Errorf("a name the render no longer owns now holds %q", got)
	}
}

func TestOpenNearest(t *testing.T) {
	parent := dir(t)
	files(t, parent, map[string]string{"f": ""})
	w, rest, err := OpenNearest(parent)
	if err != nil || rest != "." {
		t.Fatalf("an existing directory: %q, %v", rest, err)
	}
	w.Close()
	_, _, err = OpenNearest(filepath.Join(parent, "f", "out"))
	wantReason(t, err, NotDirectory, parent)
}

func TestWriterListAndLstatReadNoContent(t *testing.T) {
	root := dir(t)
	files(t, root, map[string]string{"a": "abc", "d/x": ""})
	w := openWriter(t, root)
	got, err := w.List(".", 10)
	if err != nil || len(got) != 2 || got[0] != (Entry{"a", KindFile, 3}) || got[1].Kind != KindDir {
		t.Fatalf("List = %v, %v", got, err)
	}
	if e, err := w.Lstat("d/x"); err != nil || e.Kind != KindFile {
		t.Errorf("Lstat = %v, %v", e, err)
	}
}
