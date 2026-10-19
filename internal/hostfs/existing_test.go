// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package hostfs

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// existingOut makes parent/out holding fs, and returns its path.
func existingOut(t *testing.T, fs map[string]string) (parent, out string) {
	t.Helper()
	parent = dir(t)
	out = filepath.Join(parent, "out")
	if err := os.Mkdir(out, 0o755); err != nil {
		t.Fatal(err)
	}
	files(t, out, fs)
	return parent, out
}

func openExisting(t *testing.T, out string) *Existing {
	t.Helper()
	x, err := OpenExisting(out, WriteOptions{})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { x.Close() })
	return x
}

// stage checks paths, which must all be admitted, and stages them.
func stage(t *testing.T, x *Existing, paths ...string) *Output {
	t.Helper()
	checks := x.Check(paths)
	for _, c := range checks {
		if c.Err != nil {
			t.Fatalf("Check(%q): %v", c.Path, c.Err)
		}
	}
	o, err := x.Stage(checks)
	if err != nil {
		t.Fatalf("Stage: %v", err)
	}
	t.Cleanup(func() { o.Abort() })
	return o
}

func write(t *testing.T, o *Output, name, body string) {
	t.Helper()
	if err := o.Create(name, false, strings.NewReader(body)); err != nil {
		t.Fatalf("Create(%q): %v", name, err)
	}
}

// existingTree is the output directory the write tests start from.
var existingTree = map[string]string{
	"keep.txt":  "K",
	"r.txt":     "old r",
	"a/old.txt": "A",
	"a/r.txt":   "old a/r",
}

// existingPlan is what they write: two replaces, a create beside an
// existing file, and two files beneath a directory that does not exist.
var existingPlan = []string{"a/n.txt", "a/r.txt", "b/c/n.txt", "b/d.txt", "r.txt"}

// W5-0.3, W5-0.4: the checks give each planned path its status and unit
// and write nothing; the staged files sit beneath the staging directory
// and nothing else changes until Commit, which puts each in place and
// removes the staging directory.
func TestExistingWrite(t *testing.T) {
	parent, out := existingOut(t, existingTree)
	x := openExisting(t, out)
	before := tree(t, parent)
	checks := x.Check(existingPlan)
	type got struct {
		status  Status
		unit    string
		unitDir bool
	}
	want := []got{
		{Create, "a/n.txt", false},
		{Replace, "a/r.txt", false},
		{Create, "b", true},
		{Create, "b", true},
		{Replace, "r.txt", false},
	}
	for i, c := range checks {
		if c.Err != nil || c.Path != existingPlan[i] || (got{c.Status, c.unit, c.unitDir}) != want[i] {
			t.Errorf("Check %d = %+v, want %s %+v", i, c, existingPlan[i], want[i])
		}
	}
	if after := tree(t, parent); !slices.Equal(after, before) {
		t.Fatalf("Check wrote: %q, was %q", after, before)
	}
	o, err := x.Stage(checks)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range existingPlan {
		write(t, o, name, "<"+name+">")
	}
	staged := slices.DeleteFunc(tree(t, parent), func(p string) bool { return slices.Contains(before, p) })
	wantStaged := []string{
		"out/.metais-staging/", "out/.metais-staging/a/", "out/.metais-staging/a/n.txt", "out/.metais-staging/a/r.txt",
		"out/.metais-staging/b/", "out/.metais-staging/b/c/", "out/.metais-staging/b/c/n.txt", "out/.metais-staging/b/d.txt",
		"out/.metais-staging/r.txt",
	}
	if !slices.Equal(staged, wantStaged) {
		t.Fatalf("prepared %q, want %q", staged, wantStaged)
	}
	if err := o.Commit(); err != nil {
		t.Fatal(err)
	}
	wantTree := []string{"out/", "out/a/", "out/a/n.txt", "out/a/old.txt", "out/a/r.txt", "out/b/", "out/b/c/", "out/b/c/n.txt", "out/b/d.txt", "out/keep.txt", "out/r.txt"}
	if got := tree(t, parent); !slices.Equal(got, wantTree) {
		t.Fatalf("after Commit: %q, want %q", got, wantTree)
	}
	for _, name := range existingPlan {
		if got := readHost(t, filepath.Join(out, filepath.FromSlash(name))); got != "<"+name+">" {
			t.Errorf("%s holds %q", name, got)
		}
	}
	for name, body := range map[string]string{"keep.txt": "K", "a/old.txt": "A"} {
		if got := readHost(t, filepath.Join(out, filepath.FromSlash(name))); got != body {
			t.Errorf("%s, which the write did not plan, holds %q", name, got)
		}
	}
}

// W5-0.7: a replace renames over the directory entry and never writes into
// the file: a hard link to the replaced file keeps the old bytes.
func TestExistingReplaceKeepsAHardLink(t *testing.T) {
	parent, out := existingOut(t, map[string]string{"r.txt": "old"})
	link := filepath.Join(parent, "link")
	if err := os.Link(filepath.Join(out, "r.txt"), link); err != nil {
		t.Skipf("cannot make a hard link here: %v", err)
	}
	x := openExisting(t, out)
	o := stage(t, x, "r.txt")
	write(t, o, "r.txt", "new")
	if err := o.Commit(); err != nil {
		t.Fatal(err)
	}
	if got := readHost(t, link); got != "old" {
		t.Errorf("the hard link holds %q, want the old bytes", got)
	}
	if got := readHost(t, filepath.Join(out, "r.txt")); got != "new" {
		t.Errorf("r.txt holds %q", got)
	}
}

// W5-V2.A1: a planned directory, a path that ends in "/", is checked with
// the files, prepared beneath the staging directory, and arrives in the
// output directory only by the commit's rename of its unit. One that
// exists already is Present and left as it is; one with a file, a link or
// a case variant in its way is refused with the others.
func TestExistingPlannedDirectory(t *testing.T) {
	parent, out := existingOut(t, map[string]string{"r.txt": "old", "kept/x.txt": "X"})
	x := openExisting(t, out)
	before := tree(t, parent)
	plan := []string{"empty/", "kept/", "new/deep/", "r.txt"}
	checks := x.Check(plan)
	type got struct {
		status  Status
		unit    string
		unitDir bool
	}
	want := []got{{Create, "empty", true}, {Present, "kept", false}, {Create, "new", true}, {Replace, "r.txt", false}}
	for i, c := range checks {
		if c.Err != nil || c.Path != plan[i] || (got{c.Status, c.unit, c.unitDir}) != want[i] {
			t.Errorf("Check %d = %+v, want %s %+v", i, c, plan[i], want[i])
		}
	}
	o, err := x.Stage(checks)
	if err != nil {
		t.Fatal(err)
	}
	for _, d := range []string{"empty", "new/deep"} {
		if err := o.MkdirAll(d); err != nil {
			t.Fatal(err)
		}
	}
	write(t, o, "r.txt", "new")
	var renamed []string
	x.w.beforeRename = func(from, to string) {
		if to == "empty" || to == "new" {
			if _, err := os.Lstat(filepath.Join(out, to)); err == nil {
				t.Errorf("%s is in the output directory before its rename", to)
			}
			if _, err := os.Lstat(filepath.Join(out, filepath.FromSlash(from))); err != nil {
				t.Errorf("%s is not staged at its rename: %v", to, err)
			}
		}
		renamed = append(renamed, to)
	}
	staged := slices.DeleteFunc(tree(t, parent), func(p string) bool { return slices.Contains(before, p) })
	wantStaged := []string{"out/.metais-staging/", "out/.metais-staging/empty/", "out/.metais-staging/new/", "out/.metais-staging/new/deep/", "out/.metais-staging/r.txt"}
	if !slices.Equal(staged, wantStaged) {
		t.Fatalf("prepared %q, want %q", staged, wantStaged)
	}
	if err := o.Commit(); err != nil {
		t.Fatal(err)
	}
	if want := []string{"empty", "new", "r.txt"}; !slices.Equal(renamed, want) {
		t.Errorf("renamed %q, want %q", renamed, want)
	}
	wantTree := []string{"out/", "out/empty/", "out/kept/", "out/kept/x.txt", "out/new/", "out/new/deep/", "out/r.txt"}
	if got := tree(t, parent); !slices.Equal(got, wantTree) {
		t.Fatalf("after Commit: %q, want %q", got, wantTree)
	}

	t.Run("refused", func(t *testing.T) {
		parent, out := existingOut(t, map[string]string{"file": "F", "Cased/x.txt": "X"})
		symlink(t, out, "Cased", "link")
		before := tree(t, parent)
		x := openExisting(t, out)
		plan := []string{"cased/", "file/", "link/", "ok/"}
		reasons := []Reason{CaseVariant, NotDirectory, Link, ""}
		for i, c := range x.Check(plan) {
			if reasons[i] == "" {
				if c.Err != nil {
					t.Errorf("%s: %v", c.Path, c.Err)
				}
				continue
			}
			wantReason(t, c.Err, reasons[i], out)
		}
		if after := tree(t, parent); !slices.Equal(after, before) {
			t.Fatalf("Check wrote: %q, was %q", after, before)
		}
	})

	t.Run("not made", func(t *testing.T) {
		parent, out := existingOut(t, map[string]string{"keep.txt": "K"})
		before := tree(t, parent)
		o := stage(t, openExisting(t, out), "empty/")
		var ce *CommitError
		if err := o.Commit(); !errors.As(err, &ce) || len(ce.Committed) > 0 {
			t.Fatalf("Commit of a directory not made = %v, want a CommitError with nothing committed", err)
		}
		if after := tree(t, parent); !slices.Equal(after, before) {
			t.Fatalf("after the refused commit: %q, was %q", after, before)
		}
	})
}

// W5-0.2: every refusal is reported per planned path, with the others, in
// the order planned, and nothing is written.
func TestExistingCheckRefusals(t *testing.T) {
	parent, out := existingOut(t, map[string]string{
		"a/x.txt":   "X",
		"file":      "F",
		"Readme.md": "R",
		"d/x":       "X",
		// NFD: "e" and a combining acute accent.
		"cafe\u0301.txt": "C",
	})
	outside := filepath.Join(parent, "outside")
	if err := os.Mkdir(outside, 0o755); err != nil {
		t.Fatal(err)
	}
	symlink(t, out, "a", "inlink")
	symlink(t, out, outside, "outlink")
	symlink(t, out, "a/x.txt", "filelink")
	x := openExisting(t, out)
	before := tree(t, parent)
	cases := []struct {
		path string
		want Reason
	}{
		{"inlink/y", Link},
		{"outlink/y", Link},
		{"filelink", Link},
		{"file/y", NotDirectory},
		{"a/x.txt/y", NotDirectory},
		{"README.md", CaseVariant},
		{"A/new.txt", CaseVariant},
		{"caf\u00e9.txt", CaseVariant},
		{"d", IsDirectory},
		{".metais-staging/x", Exists},
		{".METAIS-STAGING", Exists},
		{"..", BadName},
		{"a/../x", BadName},
		{"", BadName},
		{".", BadName},
	}
	var paths []string
	for _, c := range cases {
		paths = append(paths, c.path)
	}
	paths = append(paths, "a/ok.txt", "Readme.md")
	got := x.Check(paths)
	for i, c := range cases {
		if got[i].Path != c.path {
			t.Fatalf("check %d is of %q, want %q", i, got[i].Path, c.path)
		}
		if got[i].Err == nil {
			t.Errorf("%q: admitted as %s, want %s", c.path, got[i].Status, c.want)
			continue
		}
		wantReason(t, got[i].Err, c.want, out, parent)
	}
	if ok := got[len(cases)]; ok.Err != nil || ok.Status != Create {
		t.Errorf("a/ok.txt: %+v, want create", ok)
	}
	if same := got[len(cases)+1]; same.Err != nil || same.Status != Replace {
		t.Errorf("Readme.md, byte-equal to the existing name: %+v, want replace", same)
	}
	if after := tree(t, parent); !slices.Equal(after, before) {
		t.Fatalf("Check wrote: %q", after)
	}
	if _, err := x.Stage(got); err == nil {
		t.Fatal("Stage admitted refused checks")
	}
	if after := tree(t, parent); !slices.Equal(after, before) {
		t.Fatalf("a refused Stage wrote: %q", after)
	}
}

// W5-0.1: an existing staging directory, a killed write's leftover,
// refuses the write with Exists and nothing is written.
func TestExistingRefusesAStagingLeftover(t *testing.T) {
	parent, out := existingOut(t, map[string]string{".metais-staging/left": "L"})
	x := openExisting(t, out)
	before := tree(t, parent)
	_, err := x.Stage(x.Check([]string{"f"}))
	wantReason(t, err, Exists, out)
	if after := tree(t, parent); !slices.Equal(after, before) {
		t.Fatalf("%q, was %q", after, before)
	}
}

// W5-0.1: a failure while preparing, and Abort, remove the staging
// directory and change nothing else.
func TestExistingPrepareFailureChangesNothing(t *testing.T) {
	parent, out := existingOut(t, existingTree)
	x := openExisting(t, out)
	before := tree(t, parent)
	o := stage(t, x, existingPlan...)
	write(t, o, "b/c/n.txt", "N")
	write(t, o, "r.txt", "R")
	wantReason(t, o.Create("a/r.txt", false, failingReader{}), IOError, out)
	if err := o.Abort(); err != nil {
		t.Fatal(err)
	}
	if after := tree(t, parent); !slices.Equal(after, before) {
		t.Fatalf("after Abort: %q, was %q", after, before)
	}
	for name, body := range existingTree {
		if got := readHost(t, filepath.Join(out, filepath.FromSlash(name))); got != body {
			t.Errorf("%s holds %q", name, got)
		}
	}
}

// A Stage writes its planned paths and no other.
func TestExistingWritesOnlyThePlan(t *testing.T) {
	_, out := existingOut(t, nil)
	o := stage(t, openExisting(t, out), "a/b.txt")
	wantReason(t, o.Create("a/c.txt", false, strings.NewReader("C")), BadName, out)
}

// W5-0.5: a commit that cannot start, because a planned file was not
// written or a check now gives another answer, commits nothing, removes
// the staging directory and changes nothing.
func TestExistingCommitRefusedBeforeTheFirstRename(t *testing.T) {
	for _, tt := range []struct {
		name   string
		before func(t *testing.T, out string, o *Output)
		want   Reason
	}{
		{"a planned file not written", func(t *testing.T, out string, o *Output) {}, NotFound},
		{"a directory appeared", func(t *testing.T, out string, o *Output) {
			write(t, o, "z.txt", "Z")
			if err := os.Mkdir(filepath.Join(out, "b"), 0o755); err != nil {
				t.Fatal(err)
			}
		}, Changed},
		{"a link appeared", func(t *testing.T, out string, o *Output) {
			write(t, o, "z.txt", "Z")
			symlink(t, out, "a", "b")
		}, Link},
	} {
		t.Run(tt.name, func(t *testing.T) {
			parent, out := existingOut(t, map[string]string{"a/x": "X"})
			o := stage(t, openExisting(t, out), "b/n.txt", "z.txt")
			write(t, o, "b/n.txt", "N")
			tt.before(t, out, o)
			before := slices.DeleteFunc(tree(t, parent), func(p string) bool { return strings.HasPrefix(p, "out/.metais-staging/") })
			err := o.Commit()
			var ce *CommitError
			if !errors.As(err, &ce) || len(ce.Committed) != 0 {
				t.Fatalf("Commit = %v, want a *CommitError with nothing committed", err)
			}
			wantReason(t, err, tt.want, out)
			if after := tree(t, parent); !slices.Equal(after, before) {
				t.Fatalf("after the refused Commit: %q, want %q", after, before)
			}
		})
	}
}

// W5-0.4, W5-0.5: a directory that appears at a unit's name between the
// check and its rename refuses that rename; the commit stops, reports what
// it committed, and removes the staging directory and everything still in
// it. The directory that appeared is not the write's and is left.
func TestExistingCommitFailureReportsWhatWasCommitted(t *testing.T) {
	parent, out := existingOut(t, map[string]string{"a/x": "X"})
	x := openExisting(t, out)
	o := stage(t, x, "a/n.txt", "m/n.txt", "z/y/n.txt")
	for _, name := range []string{"a/n.txt", "m/n.txt", "z/y/n.txt"} {
		write(t, o, name, name)
	}
	x.w.beforeRename = func(from, to string) {
		if to == "z" {
			if err := os.MkdirAll(filepath.Join(out, "z", "theirs"), 0o755); err != nil {
				t.Fatal(err)
			}
		}
	}
	err := o.Commit()
	var ce *CommitError
	if !errors.As(err, &ce) {
		t.Fatalf("Commit = %v, want a *CommitError", err)
	}
	wantReason(t, err, Exists, out)
	if want := []string{"a/n.txt", "m/n.txt"}; !slices.Equal(ce.Committed, want) {
		t.Errorf("committed %q, want %q", ce.Committed, want)
	}
	want := []string{"out/", "out/a/", "out/a/n.txt", "out/a/x", "out/m/", "out/m/n.txt", "out/z/", "out/z/theirs/"}
	if got := tree(t, parent); !slices.Equal(got, want) {
		t.Fatalf("after the failed Commit: %q, want %q", got, want)
	}
}

// W5-0.2: a link planted in place of a directory between the check and the
// rename fails the commit rather than being followed: nothing is written
// where the link points.
func TestExistingCommitRefusesALinkPlantedAtWriteTime(t *testing.T) {
	parent, out := existingOut(t, map[string]string{"a/x": "X", "m/x": "X"})
	outside := filepath.Join(parent, "outside")
	if err := os.Mkdir(outside, 0o755); err != nil {
		t.Fatal(err)
	}
	x := openExisting(t, out)
	o := stage(t, x, "a/n.txt", "m/n.txt")
	write(t, o, "a/n.txt", "A")
	write(t, o, "m/n.txt", "M")
	x.w.beforeRename = func(from, to string) {
		if to == "m/n.txt" {
			if err := os.RemoveAll(filepath.Join(out, "m")); err != nil {
				t.Fatal(err)
			}
			symlink(t, out, outside, "m")
		}
	}
	err := o.Commit()
	var ce *CommitError
	if !errors.As(err, &ce) || !slices.Equal(ce.Committed, []string{"a/n.txt"}) {
		t.Fatalf("Commit = %v (%+v), want a *CommitError having committed a/n.txt", err, ce)
	}
	if got := tree(t, outside); len(got) != 0 {
		t.Fatalf("the commit wrote through the link: %q", got)
	}
	if _, err := os.Lstat(filepath.Join(out, StagingDir)); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("the staging directory is left: %v", err)
	}
}

// W5-0.2: a link planted at a replace target between the check and its
// rename fails the commit with link, by the target's re-check at write
// time, even though it points inside the output directory: neither the
// file it points at nor the link is touched. The replacing rename would
// replace the link's entry without following it, and the root does not
// refuse a link that stays inside. On a posix host the re-check's
// linkLike and fileRefusal refuse a symbolic link alike, so this pins the
// two together; only a windows reparse point that is not a symbolic link
// tells linkLike apart.
func TestExistingCommitRefusesALinkPlantedAtAReplaceTarget(t *testing.T) {
	_, out := existingOut(t, map[string]string{"a/keep.txt": "K", "a/r.txt": "old"})
	x := openExisting(t, out)
	o := stage(t, x, "a/r.txt")
	write(t, o, "a/r.txt", "new")
	x.w.beforeRename = func(from, to string) {
		if to == "a/r.txt" {
			if err := os.Remove(filepath.Join(out, "a", "r.txt")); err != nil {
				t.Fatal(err)
			}
			symlink(t, out, "keep.txt", "a/r.txt")
		}
	}
	err := o.Commit()
	var ce *CommitError
	if !errors.As(err, &ce) || len(ce.Committed) != 0 {
		t.Fatalf("Commit = %v, want a *CommitError with nothing committed", err)
	}
	wantReason(t, err, Link, out)
	if got := readHost(t, filepath.Join(out, "a", "keep.txt")); got != "K" {
		t.Errorf("a/keep.txt holds %q", got)
	}
	if target, err := os.Readlink(filepath.Join(out, "a", "r.txt")); err != nil || target != "keep.txt" {
		t.Errorf("a/r.txt is no longer the planted link: %q, %v", target, err)
	}
}

// W5-0.2: a link planted in place of a directory the commit opens, between
// the check and the rename, fails the commit with link, by that
// directory's own re-check at write time, even though it points at a
// directory inside the output directory: nothing is written through it.
// Without the re-check the directory is refused as not a directory, and
// the root does not refuse a link that stays inside.
func TestExistingCommitRefusesALinkPlantedAtACommitDirectory(t *testing.T) {
	_, out := existingOut(t, map[string]string{"a/x": "X", "m/x": "X"})
	x := openExisting(t, out)
	o := stage(t, x, "m/n.txt")
	write(t, o, "m/n.txt", "M")
	x.w.beforeRename = func(from, to string) {
		if to == "m/n.txt" {
			if err := os.RemoveAll(filepath.Join(out, "m")); err != nil {
				t.Fatal(err)
			}
			symlink(t, out, "a", "m")
		}
	}
	err := o.Commit()
	var ce *CommitError
	if !errors.As(err, &ce) || len(ce.Committed) != 0 {
		t.Fatalf("Commit = %v, want a *CommitError with nothing committed", err)
	}
	wantReason(t, err, Link, out)
	if want, got := []string{"x"}, tree(t, filepath.Join(out, "a")); !slices.Equal(got, want) {
		t.Fatalf("the commit wrote through the link: a holds %q, want %q", got, want)
	}
}

// W5-0.6: beneath the staging directory the write removes only names it
// created: a name planted there is left, and so then is the staging
// directory, whether the write is aborted or committed; a committed write
// reports it.
func TestExistingRemovesOnlyItsOwn(t *testing.T) {
	t.Run("abort", func(t *testing.T) {
		_, out := existingOut(t, map[string]string{"keep": "K"})
		o := stage(t, openExisting(t, out), "a/n.txt")
		write(t, o, "a/n.txt", "N")
		files(t, out, map[string]string{".metais-staging/a/theirs": "T"})
		o.Abort()
		want := []string{".metais-staging/", ".metais-staging/a/", ".metais-staging/a/theirs", "keep"}
		if got := tree(t, out); !slices.Equal(got, want) {
			t.Fatalf("after Abort: %q, want %q", got, want)
		}
	})
	t.Run("commit", func(t *testing.T) {
		_, out := existingOut(t, map[string]string{"a/keep": "K"})
		o := stage(t, openExisting(t, out), "a/n.txt")
		write(t, o, "a/n.txt", "N")
		files(t, out, map[string]string{".metais-staging/theirs": "T"})
		err := o.Commit()
		var ce *CommitError
		if !errors.As(err, &ce) || !slices.Equal(ce.Committed, []string{"a/n.txt"}) {
			t.Fatalf("Commit = %v, want a *CommitError having committed a/n.txt", err)
		}
		want := []string{".metais-staging/", ".metais-staging/theirs", "a/", "a/keep", "a/n.txt"}
		if got := tree(t, out); !slices.Equal(got, want) {
			t.Fatalf("after Commit: %q, want %q", got, want)
		}
	})
}

// W5-0.2, W5-0.11: A1 item 4's device case, with no privileged mount: the
// device the checks compare is replaced so that one directory is on
// another device. Each path whose deepest existing directory is on it is
// refused with io_error, and nothing reports the device.
func TestExistingRefusesAnotherDevice(t *testing.T) {
	parent, out := existingOut(t, map[string]string{"mnt/sub/x": "X", "mnt/r": "R", "a/x": "X"})
	x := openExisting(t, out)
	x.device = func(info fs.FileInfo) (uint64, bool) {
		if info.Name() == "mnt" || info.Name() == "sub" {
			return 7, true
		}
		return 1, true
	}
	before := tree(t, parent)
	got := x.Check([]string{"a/n", "mnt", "mnt/n", "mnt/r", "mnt/sub/new/n", "n"})
	for i, want := range []Reason{"", IsDirectory, IOError, IOError, IOError, ""} {
		if want == "" {
			if got[i].Err != nil {
				t.Errorf("%s: %v, want admitted", got[i].Path, got[i].Err)
			}
			continue
		}
		wantReason(t, got[i].Err, want, out, "7")
	}
	if after := tree(t, parent); !slices.Equal(after, before) {
		t.Fatalf("Check wrote: %q", after)
	}
}

// W5-0.10: RemoveFile removes a regular file beneath the directory named,
// and refuses everything else, reaching nothing outside.
func TestRemoveFile(t *testing.T) {
	parent := dir(t)
	exp := filepath.Join(parent, "expected")
	files(t, parent, map[string]string{"outside/x": "O"})
	files(t, exp, map[string]string{"a/b/stale.txt": "S", "keep.txt": "K", "d/x": "X"})
	symlink(t, exp, filepath.Join(parent, "outside"), "out")
	symlink(t, exp, "a", "in")
	symlink(t, exp, "keep.txt", "flink")
	before := tree(t, parent)
	for _, tt := range []struct {
		name string
		want Reason
	}{
		{"out/x", Link},
		{"in/b/stale.txt", Link},
		{"flink", Link},
		{"keep.txt/x", NotDirectory},
		{"d", IsDirectory},
		{"missing", NotFound},
		{"a/missing/x", NotFound},
		{"../outside/x", BadName},
		{".", BadName},
	} {
		wantReason(t, RemoveFile(exp, tt.name), tt.want, parent)
	}
	if after := tree(t, parent); !slices.Equal(after, before) {
		t.Fatalf("a refused removal changed %q to %q", before, after)
	}
	if err := RemoveFile(exp, "a/b/stale.txt"); err != nil {
		t.Fatal(err)
	}
	want := slices.DeleteFunc(before, func(p string) bool { return p == "expected/a/b/stale.txt" })
	if after := tree(t, parent); !slices.Equal(after, want) {
		t.Fatalf("after RemoveFile: %q, want %q", after, want)
	}
}

// RemoveEmptyDir removes an empty directory beneath its directory, leaves
// one that holds an entry, and refuses a link or a file at any segment,
// changing nothing.
func TestRemoveEmptyDir(t *testing.T) {
	parent := dir(t)
	exp := filepath.Join(parent, "expected")
	files(t, parent, map[string]string{"outside/empty/.keep": "O"})
	files(t, exp, map[string]string{"full/x": "X", "file": "F"})
	for _, d := range []string{"a/b", "outside-empty"} {
		if err := os.MkdirAll(filepath.Join(exp, filepath.FromSlash(d)), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	symlink(t, exp, filepath.Join(parent, "outside"), "out")
	symlink(t, exp, "a", "in")
	before := tree(t, parent)
	for _, tt := range []struct {
		name string
		want Reason
	}{
		{"out/empty", Link},
		{"in/b", Link},
		{"in", Link},
		{"file", NotDirectory},
		{"file/x", NotDirectory},
		{"missing", NotFound},
		{"../outside/empty", BadName},
		{".", BadName},
	} {
		wantReason(t, RemoveEmptyDir(exp, tt.name), tt.want, parent)
	}
	if err := RemoveEmptyDir(exp, "full"); err != nil {
		t.Fatalf("a directory holding an entry: %v, want it left and no error", err)
	}
	if after := tree(t, parent); !slices.Equal(after, before) {
		t.Fatalf("a refused or declined removal changed %q to %q", before, after)
	}
	for _, d := range []string{"a/b", "a"} {
		if err := RemoveEmptyDir(exp, d); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := os.Lstat(filepath.Join(exp, "a")); !os.IsNotExist(err) {
		t.Fatalf("a/ after its removal: %v", err)
	}
}
