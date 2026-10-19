// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

//go:build linux || darwin

package hostfs

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/kottera/metais/internal/limits"
)

// W5-0.9: a write into an existing directory of as many files as ADR-009
// D3's outputs limit admits holds a bounded number of handles, and is not
// refused by an open-file soft limit far below that number. Half the
// directories exist, and a tenth of the files in them are replaced. No
// limit is lowered to fit (R1-015). An aborted write removes what it
// prepared under the same limit; it is run with a thousand files, eight
// times the limit, since removal opens nothing and the commit has proved
// the prepare at the full count.
func TestExistingWritesItsLimitUnderTheOpenFileLimit(t *testing.T) {
	for _, commit := range []bool{true, false} {
		n := limits.Get(limits.Outputs).Default
		if !commit {
			n = 1000
		}
		names := manyNames(n)
		seed := map[string]string{}
		for i, name := range names {
			if d := i / 1000; d%2 == 0 && i%10 == 0 {
				seed[name] = "old"
			}
		}
		t.Run(fmt.Sprint("commit=", commit), func(t *testing.T) {
			parent, out := existingOut(t, seed)
			var before []string
			if !commit {
				before = tree(t, parent)
			}
			lowerOpenFiles(t, openFiles)
			x, err := OpenExisting(out, WriteOptions{})
			if err != nil {
				t.Fatal(err)
			}
			defer x.Close()
			checks := x.Check(names)
			replaced := 0
			for _, c := range checks {
				if c.Err != nil {
					t.Fatalf("%s: %v", c.Path, c.Err)
				}
				if c.Status == Replace {
					replaced++
				}
			}
			if replaced != len(seed) {
				t.Fatalf("%d replaces, want %d", replaced, len(seed))
			}
			o, err := x.Stage(checks)
			if err != nil {
				t.Fatal(err)
			}
			for _, name := range names {
				if err := o.Create(name, false, strings.NewReader(name)); err != nil {
					t.Fatalf("Create(%q), after %d files: %v", name, len(o.Written()), err)
				}
			}
			if !commit {
				if err := o.Abort(); err != nil {
					t.Fatal(err)
				}
				if after := tree(t, parent); !slices.Equal(after, before) {
					t.Fatalf("after Abort the tree holds %d names, was %d", len(after), len(before))
				}
				return
			}
			if err := o.Commit(); err != nil {
				t.Fatal(err)
			}
			for i, name := range names {
				if i%997 != 0 {
					continue
				}
				if got := readHost(t, filepath.Join(out, filepath.FromSlash(name))); got != name {
					t.Fatalf("%s holds %q", name, got)
				}
			}
			if files := countFiles(t, parent); files != len(names) {
				t.Fatalf("the output holds %d files, want %d", files, len(names))
			}
		})
	}
}

// Under a umask that withholds the owner's read, a write into an existing
// directory prepares, commits and cleans up as under any other, and what
// it commits has its modes less that umask (ADR-005 D14 item 4): a
// directory committed whole gets its own mode back once it is in place.
func TestExistingUnderAUmaskWithoutOwnerRead(t *testing.T) {
	const mask = 0o477
	parent, out := existingOut(t, map[string]string{"e/keep": "K", "r.txt": "old"})
	t.Cleanup(func() { unlock(t, parent) })
	withUmask(t, mask)
	x := openExisting(t, out)
	o := stage(t, x, "a/b/c.txt", "e/d.txt", "r.txt")
	for _, name := range []string{"a/b/c.txt", "e/d.txt", "r.txt"} {
		write(t, o, name, "<"+name+">")
	}
	if err := o.Commit(); err != nil {
		t.Fatal(err)
	}
	got := lstatModes(t, out, []string{"a", "a/b", "a/b/c.txt", "e", "e/d.txt", "r.txt"})
	want := map[string]fs.FileMode{
		"a": dirMode &^ mask, "a/b": dirMode &^ mask, "a/b/c.txt": fileMode &^ mask,
		"e": 0o755, "e/d.txt": fileMode &^ mask, "r.txt": fileMode &^ mask,
	}
	for name, mode := range want {
		if got[name] != mode {
			t.Errorf("%s: mode %v, want %v", name, got[name], mode)
		}
	}
	unlock(t, parent)
	if _, err := os.Lstat(filepath.Join(out, StagingDir)); !os.IsNotExist(err) {
		t.Errorf("the staging directory is left: %v", err)
	}
	for _, name := range []string{"a/b/c.txt", "e/d.txt", "r.txt"} {
		p := filepath.Join(out, filepath.FromSlash(name))
		if err := os.Chmod(p, 0o600); err != nil {
			t.Fatal(err)
		}
		if got := readHost(t, p); got != "<"+name+">" {
			t.Errorf("%s holds %q", name, got)
		}
	}
}

// W5-0.2: a FIFO at a planned target is refused, unopened, as a file that
// is not regular.
func TestExistingRefusesAFIFOTarget(t *testing.T) {
	_, out := existingOut(t, nil)
	mkfifo(t, out, "a/pipe")
	got := openExisting(t, out).Check([]string{"a/pipe"})
	wantReason(t, got[0].Err, NotRegularFile, out)
}

// W5-0.11: the device comparison on a real mount, where the host has one
// to hand without privilege: /dev is its own file system on darwin and on
// most linux hosts. Checked against "/" as the output directory, a planned
// file in /dev is refused with io_error, and nothing is written, because a
// check writes nothing.
func TestExistingRefusesARealMountPoint(t *testing.T) {
	root, err := os.Lstat("/")
	if err != nil {
		t.Skip(err)
	}
	dev, err := os.Lstat("/dev")
	if err != nil {
		t.Skip(err)
	}
	a, _ := platformDevice(root)
	b, _ := platformDevice(dev)
	if a == b {
		t.Skip("/dev is on the root file system here")
	}
	x, err := OpenExisting("/", WriteOptions{})
	if err != nil {
		t.Skipf("cannot open / as a root here: %v", err)
	}
	defer x.Close()
	got := x.Check([]string{"dev/metais-w5-0-absent"})
	if ReasonOf(got[0].Err) == PermissionDenied {
		t.Skip("this sandbox does not let / or /dev be examined; TestExistingRefusesAnotherDevice holds the comparison")
	}
	wantReason(t, got[0].Err, IOError, "/dev")
}
