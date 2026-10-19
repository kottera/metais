// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package hostfs

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/kottera/metais/internal/fault"
)

// shape is one of the write shapes, opened on the output directory out,
// beneath parent: StageNew and CreateNew on an out that does not exist,
// WriteInPlace on one that does and is empty.
type shape struct {
	name string
	open func(out string, opt WriteOptions) (*Output, error)

	// exists is whether the shape writes into an existing directory.
	exists bool

	// keeps is whether a failed write keeps what it wrote.
	keeps bool
}

var shapes = []shape{
	{"StageNew", StageNew, false, false},
	{"WriteInPlace", WriteInPlace, true, true},
	{"CreateNew", CreateNew, false, true},
}

// begin opens s on parent/out, creating out first when s writes into an
// existing directory.
func (s shape) begin(t *testing.T, parent string, opt WriteOptions) *Output {
	t.Helper()
	out := filepath.Join(parent, "out")
	if s.exists {
		// Chmod too, so the caller's directory is the caller's whatever
		// the umask.
		if err := os.Mkdir(out, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.Chmod(out, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	o, err := s.open(out, opt)
	if err != nil {
		t.Fatalf("%s: %v", s.name, err)
	}
	t.Cleanup(func() { o.Abort() })
	return o
}

func put(t *testing.T, o *Output, name, body string) {
	t.Helper()
	if err := o.MkdirAll(dirOf(name)); err != nil {
		t.Fatalf("MkdirAll(%q): %v", dirOf(name), err)
	}
	if err := o.Create(name, false, strings.NewReader(body)); err != nil {
		t.Fatalf("Create(%q): %v", name, err)
	}
}

func dirOf(name string) string {
	d, _ := split(name)
	return d
}

var written = []string{"a/b/c.txt", "a/d.txt", "e.txt"}

// Each shape writes its entries, and after Commit the output directory
// holds them and nothing else is left beside it.
func TestShapesWrite(t *testing.T) {
	for _, s := range shapes {
		t.Run(s.name, func(t *testing.T) {
			parent := dir(t)
			o := s.begin(t, parent, WriteOptions{})
			for _, name := range written {
				put(t, o, name, "<"+name+">")
			}
			if s.name == "StageNew" {
				if _, err := os.Lstat(filepath.Join(parent, "out")); !errors.Is(err, os.ErrNotExist) {
					t.Fatalf("before Commit the output directory is there: %v", err)
				}
			}
			if err := o.Commit(); err != nil {
				t.Fatal(err)
			}
			want := []string{"out/", "out/a/", "out/a/b/", "out/a/b/c.txt", "out/a/d.txt", "out/e.txt"}
			if got := tree(t, parent); !slices.Equal(got, want) {
				t.Fatalf("after Commit: %q, want %q", got, want)
			}
			for _, name := range written {
				if got := readHost(t, filepath.Join(parent, "out", filepath.FromSlash(name))); got != "<"+name+">" {
					t.Errorf("%s holds %q", name, got)
				}
			}
			if got := o.Written(); !slices.Equal(got, written) {
				t.Errorf("Written = %q, want %q", got, written)
			}
		})
	}
}

// StageNew and CreateNew create the output directory's missing ancestors.
func TestNewShapesCreateAncestors(t *testing.T) {
	for _, open := range []func(string, WriteOptions) (*Output, error){StageNew, CreateNew} {
		parent := dir(t)
		o, err := open(filepath.Join(parent, "x", "y", "out"), WriteOptions{})
		if err != nil {
			t.Fatal(err)
		}
		put(t, o, "f", "F")
		if err := o.Commit(); err != nil {
			t.Fatal(err)
		}
		want := []string{"x/", "x/y/", "x/y/out/", "x/y/out/f"}
		if got := tree(t, parent); !slices.Equal(got, want) {
			t.Errorf("%q, want %q", got, want)
		}
	}
}

// StageNew and CreateNew refuse an output directory that exists, empty or
// not, and a file there; WriteInPlace refuses one that does not.
func TestShapesRefuseTheWrongTarget(t *testing.T) {
	parent := dir(t)
	files(t, parent, map[string]string{"full/x": "X", "file": "F"})
	if err := os.Mkdir(filepath.Join(parent, "empty"), 0o755); err != nil {
		t.Fatal(err)
	}
	before := tree(t, parent)
	for _, open := range []func(string, WriteOptions) (*Output, error){StageNew, CreateNew} {
		for _, name := range []string{"empty", "full"} {
			_, err := open(filepath.Join(parent, name), WriteOptions{})
			wantReason(t, err, Exists, parent)
		}
		_, err := open(filepath.Join(parent, "file"), WriteOptions{})
		wantReason(t, err, NotDirectory, parent)
		_, err = open(filepath.Join(parent, "file", "out"), WriteOptions{})
		wantReason(t, err, NotDirectory, parent)
	}
	_, err := WriteInPlace(filepath.Join(parent, "absent"), WriteOptions{})
	wantReason(t, err, NotFound, parent)
	_, err = WriteInPlace(filepath.Join(parent, "file"), WriteOptions{})
	wantReason(t, err, NotDirectory, parent)
	if got := tree(t, parent); !slices.Equal(got, before) {
		t.Errorf("a refused open changed the parent: %q, want %q", got, before)
	}
}

// A killed write's leftover staging directory, or a concurrent write's,
// refuses StageNew, and is left as it was.
func TestStageNewRefusesAStagingLeftover(t *testing.T) {
	parent := dir(t)
	files(t, parent, map[string]string{"a/" + StagingName("out") + "/left": "L"})
	before := tree(t, parent)
	_, err := StageNew(filepath.Join(parent, "a", "out"), WriteOptions{})
	wantReason(t, err, Exists, parent)
	if got := tree(t, parent); !slices.Equal(got, before) {
		t.Errorf("after the refusal: %q, want %q", got, before)
	}
}

// ADR-005 D14 item 1.5: a StageNew that fails before its rename removes
// the staging directory, what it wrote beneath it and the ancestors it
// created, and leaves the parent as it was.
func TestStageNewAbortLeavesNothing(t *testing.T) {
	parent := dir(t)
	files(t, parent, map[string]string{"theirs": "T", "x/keep": "K"})
	before := tree(t, parent)
	o, err := StageNew(filepath.Join(parent, "x", "new", "deeper", "out"), WriteOptions{})
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range written {
		put(t, o, name, "W")
	}
	wantReason(t, o.Create("half", false, failingReader{}), IOError, parent)
	if err := o.Abort(); err != nil {
		t.Fatal(err)
	}
	if got := tree(t, parent); !slices.Equal(got, before) {
		t.Errorf("after Abort: %q, want %q", got, before)
	}
}

// ADR-005 D14 item 1.4: a target that appears after every check, just
// before the rename, refuses the commit; the racing directory is left as
// it was, and the write removes only its own.
func TestStageNewCommitRefusesARacingTarget(t *testing.T) {
	parent := dir(t)
	o, err := StageNew(filepath.Join(parent, "a", "out"), WriteOptions{})
	if err != nil {
		t.Fatal(err)
	}
	put(t, o, "index.html", "NEW")
	o.parent.beforeRename = func(from, to string) {
		files(t, parent, map[string]string{"a/out/theirs": "THEIRS"})
	}
	wantReason(t, o.Commit(), Exists, parent)
	want := []string{"a/", "a/out/", "a/out/theirs"}
	if got := tree(t, parent); !slices.Equal(got, want) {
		t.Fatalf("after the refused commit: %q, want %q", got, want)
	}
}

// The in-place shapes remove nothing on failure (ADR-007 D8 item 3): every
// file written stays, Written reports it, and only the file whose bytes
// failed part way is gone.
func TestInPlaceShapesKeepWhatTheyWrote(t *testing.T) {
	for _, s := range shapes {
		if !s.keeps {
			continue
		}
		t.Run(s.name, func(t *testing.T) {
			parent := dir(t)
			o := s.begin(t, parent, WriteOptions{})
			for _, name := range written {
				put(t, o, name, "W")
			}
			wantReason(t, o.Create("a/half", false, failingReader{}), IOError, parent)
			if err := o.Abort(); err != nil {
				t.Fatal(err)
			}
			want := []string{"out/", "out/a/", "out/a/b/", "out/a/b/c.txt", "out/a/d.txt", "out/e.txt"}
			if got := tree(t, parent); !slices.Equal(got, want) {
				t.Fatalf("after Abort: %q, want %q", got, want)
			}
			if got := o.Written(); !slices.Equal(got, written) {
				t.Errorf("Written = %q, want %q", got, written)
			}
		})
	}
}

// swapReader, part way through a file's bytes, puts a foreign file in the
// place of the file being written, and fails.
type swapReader struct {
	t    *testing.T
	path string
}

func (r swapReader) Read(p []byte) (int, error) {
	if err := os.Remove(r.path); err != nil {
		r.t.Fatal(err)
	}
	if err := os.WriteFile(r.path, []byte("THEIRS"), 0o644); err != nil {
		r.t.Fatal(err)
	}
	return 0, os.ErrDeadlineExceeded
}

// W4-0.2: a foreign file put in the place of a file whose bytes fail part
// way is not removed, in any shape: the removal is proved by the handle
// that wrote the file, and the newcomer is not what it holds.
func TestShapesLeaveAForeignFileInPlaceOfAFailedOne(t *testing.T) {
	for _, s := range shapes {
		t.Run(s.name, func(t *testing.T) {
			parent := dir(t)
			o := s.begin(t, parent, WriteOptions{})
			at := filepath.Join(o.w.root.Name(), "f")
			wantReason(t, o.Create("f", false, swapReader{t, at}), IOError, parent)
			if got := readHost(t, at); got != "THEIRS" {
				t.Errorf("the foreign file now holds %q", got)
			}
		})
	}
}

// W4-0.2 for StageNew: a newcomer in the place of the staging directory or
// of an ancestor the write created is not removed by Abort, nor is
// anything in it; what the write created is still removed from where it
// was moved to, because the root is a handle on the write's own staging
// directory.
func TestStageNewAbortLeavesNewcomersInItsPlace(t *testing.T) {
	for _, moved := range []string{"a/b/" + StagingName("out"), "a/b", "a"} {
		t.Run(moved, func(t *testing.T) {
			parent := dir(t)
			o, err := StageNew(filepath.Join(parent, "a", "b", "out"), WriteOptions{})
			if err != nil {
				t.Fatal(err)
			}
			put(t, o, "d/f", "MINE")
			// Someone else moves the write's directory aside and puts
			// their own at its name.
			from := filepath.Join(parent, filepath.FromSlash(moved))
			if err := os.Rename(from, filepath.Join(parent, "aside")); err != nil {
				t.Fatal(err)
			}
			files(t, parent, map[string]string{moved + "/theirs": "THEIRS"})
			newcomer := tree(t, from)
			if err := o.Abort(); err != nil {
				t.Fatal(err)
			}
			if got := tree(t, from); !slices.Equal(got, newcomer) {
				t.Errorf("the newcomer now holds %q, want %q", got, newcomer)
			}
			if got := readHost(t, filepath.Join(from, "theirs")); got != "THEIRS" {
				t.Errorf("the newcomer's file now holds %q", got)
			}
			// Nothing the write created is left beneath the staging
			// directory, wherever it went.
			for _, p := range tree(t, filepath.Join(parent, "aside")) {
				if strings.HasSuffix(p, "d/f") || strings.HasSuffix(p, "d/") {
					t.Errorf("Abort left %s, which the write created", p)
				}
			}
		})
	}
}

// W4-0.2 for StageNew: an entry put beneath the staging directory at a name
// the write did not create is not removed, and so neither is the staging
// directory that holds it.
func TestStageNewAbortLeavesAForeignNameBeneathTheStaging(t *testing.T) {
	parent := dir(t)
	o, err := StageNew(filepath.Join(parent, "out"), WriteOptions{})
	if err != nil {
		t.Fatal(err)
	}
	put(t, o, "d/f", "MINE")
	staging := filepath.Join(parent, StagingName("out"))
	files(t, staging, map[string]string{"d/theirs": "T", "top": "T"})
	if err := o.Abort(); err != nil {
		t.Fatal(err)
	}
	want := []string{StagingName("out") + "/", StagingName("out") + "/d/", StagingName("out") + "/d/theirs", StagingName("out") + "/top"}
	if got := tree(t, parent); !slices.Equal(got, want) {
		t.Errorf("after Abort: %q, want %q", got, want)
	}
}

// W4-0.4: every shape confines every name beneath the output directory,
// writes through no link, even one that points inside, and creates
// exclusively. The control shows each escape is real: a writer that joins
// the name onto the directory's path lands outside it.
func TestShapesAreConfined(t *testing.T) {
	for _, s := range shapes {
		t.Run(s.name, func(t *testing.T) {
			parent := dir(t)
			outside := testDirOutside(t)
			files(t, outside, map[string]string{"authorized_keys": "ORIGINAL"})
			o := s.begin(t, parent, WriteOptions{})
			at := o.w.root.Name()
			symlink(t, at, outside, "ssh")
			files(t, at, map[string]string{"inner/keep": "KEEP", "old": "OLD"})
			symlink(t, at, "inner", "inlink")
			symlink(t, at, "old", "oldlink")
			rel, err := filepath.Rel(at, outside)
			if err != nil {
				t.Fatal(err)
			}
			up := filepath.ToSlash(rel) + "/new"

			refusals := []struct {
				name string
				want Reason
			}{
				{"ssh/authorized_keys", Link},
				{"ssh/new", Link},
				{"ssh/sub/new", Link},
				{"inlink/new", Link},
				{up, BadName},
				{"/etc/new", BadName},
				{"a/./b", BadName},
				{`a\b`, BadName},
				{"c:new", BadName},
				{"old", Exists},
				{"oldlink", Exists},
				{"inner", Exists},
				{"old/x", NotDirectory},
			}
			for _, r := range refusals {
				wantReason(t, o.Create(r.name, false, strings.NewReader("PWNED")), r.want, parent, outside)
			}
			for _, r := range []struct {
				name string
				want Reason
			}{{"ssh/new", Link}, {"inlink/d", Link}, {up, BadName}, {"old/d", NotDirectory}, {"oldlink", Link}} {
				wantReason(t, o.MkdirAll(r.name), r.want, parent, outside)
			}
			if got := tree(t, outside); !slices.Equal(got, []string{"authorized_keys"}) {
				t.Errorf("outside the output directory now holds %q", got)
			}
			if got := readHost(t, filepath.Join(outside, "authorized_keys")); got != "ORIGINAL" {
				t.Errorf("a file outside now holds %q", got)
			}
			if got := tree(t, filepath.Join(at, "inner")); !slices.Equal(got, []string{"keep"}) {
				t.Errorf("the linked directory now holds %q", got)
			}
			if got := readHost(t, filepath.Join(at, "old")); got != "OLD" {
				t.Errorf("an existing file now holds %q", got)
			}
		})
	}
}

// W4-0.5: under fault.SkipConfine the output path check returns success,
// and a write that climbs out of the output directory, or goes through a
// link out of it, lands outside; without the fault the same writes are
// refused and nothing outside changes. This is the seam the confinement
// detector's control drives.
func TestSkipConfineLetsAWriteEscape(t *testing.T) {
	for _, s := range shapes {
		t.Run(s.name, func(t *testing.T) {
			for _, faults := range []fault.Set{0, fault.SkipConfine} {
				parent := dir(t)
				outside := testDirOutside(t)
				o := s.begin(t, parent, WriteOptions{Faults: faults})
				at := o.w.root.Name()
				symlink(t, at, outside, "ssh")
				rel, err := filepath.Rel(at, outside)
				if err != nil {
					t.Fatal(err)
				}
				up := filepath.ToSlash(rel) + "/climbed"
				errUp := o.Create(up, false, strings.NewReader("PWNED"))
				errLink := o.Create("ssh/linked", false, strings.NewReader("PWNED"))
				errDir := o.MkdirAll(filepath.ToSlash(rel) + "/made/deeper")
				got := tree(t, outside)
				if faults == 0 {
					wantReason(t, errUp, BadName, parent, outside)
					wantReason(t, errLink, Link, parent, outside)
					wantReason(t, errDir, BadName, parent, outside)
					if len(got) != 0 {
						t.Errorf("without the fault, outside now holds %q", got)
					}
					continue
				}
				if errUp != nil || errLink != nil || errDir != nil {
					t.Fatalf("under SkipConfine: %v, %v, %v", errUp, errLink, errDir)
				}
				want := []string{"climbed", "linked", "made/", "made/deeper/"}
				if !slices.Equal(got, want) {
					t.Errorf("under SkipConfine, outside holds %q, want %q", got, want)
				}
			}
		})
	}
}

// An ended Output writes nothing more, and a second end is harmless.
func TestOutputEndsOnce(t *testing.T) {
	for _, s := range shapes {
		parent := dir(t)
		o := s.begin(t, parent, WriteOptions{})
		if err := o.Commit(); err != nil {
			t.Fatal(err)
		}
		if err := o.Create("late", false, strings.NewReader("x")); err == nil {
			t.Errorf("%s: Create after Commit succeeded", s.name)
		}
		if err := o.MkdirAll("late"); err == nil {
			t.Errorf("%s: MkdirAll after Commit succeeded", s.name)
		}
		if err := o.Commit(); err == nil {
			t.Errorf("%s: a second Commit succeeded", s.name)
		}
		if err := o.Abort(); err != nil {
			t.Errorf("%s: Abort after Commit: %v", s.name, err)
		}
		if got := tree(t, parent); !slices.Equal(got, []string{"out/"}) {
			t.Errorf("%s: %q", s.name, got)
		}
	}
}
