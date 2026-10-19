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
	"syscall"
	"testing"

	"github.com/kottera/metais/internal/limits"
)

// openFiles is the open-file soft limit the W4-0.1 test runs under: far
// below every entry count it writes, and below darwin's default of 256.
const openFiles = 128

// lowerOpenFiles sets the process's open-file soft limit to n for the
// rest of t, and back afterwards. No test in this package is parallel, so
// nothing else runs under it.
func lowerOpenFiles(t *testing.T, n uint64) {
	t.Helper()
	var old syscall.Rlimit
	if err := syscall.Getrlimit(syscall.RLIMIT_NOFILE, &old); err != nil {
		t.Fatal(err)
	}
	lim := old
	lim.Cur = n
	if err := syscall.Setrlimit(syscall.RLIMIT_NOFILE, &lim); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := syscall.Setrlimit(syscall.RLIMIT_NOFILE, &old); err != nil {
			t.Fatal(err)
		}
	})
}

// manyNames are n file names, a thousand to a directory.
func manyNames(n int64) []string {
	names := make([]string, n)
	for i := range names {
		names[i] = fmt.Sprintf("d%03d/f%05d", i/1000, i)
	}
	return names
}

// W4-0.1: each shape writes as many entries as ADR-009 D3's defaults admit
// it with the open-file soft limit far below that number. StageNew is what
// render writes a new directory with, up to the outputs limit; unpack's and
// extract's entries are a package's, up to the entries limit. No limit is
// lowered to fit (R1-015).
//
// The control shows the limit is real: a Writer that pins every entry,
// which is what each of these writes was before, is refused under it.
func TestShapesWriteTheirLimitUnderTheOpenFileLimit(t *testing.T) {
	counts := map[string]int64{
		"StageNew":     limits.Get(limits.Outputs).Default,
		"WriteInPlace": limits.Get(limits.Entries).Default,
		"CreateNew":    limits.Get(limits.Entries).Default,
	}
	t.Run("control", func(t *testing.T) {
		root := dir(t)
		lowerOpenFiles(t, openFiles)
		w, rest, err := OpenNearest(filepath.Join(root, "out"))
		if err != nil {
			t.Fatal(err)
		}
		defer w.Close()
		if err := w.Mkdir(rest); err != nil {
			t.Fatal(err)
		}
		s, err := w.Sub(rest)
		if err != nil {
			t.Fatal(err)
		}
		for i := range 2 * openFiles {
			if err := s.Create(fmt.Sprintf("f%d", i), false, strings.NewReader("x")); err != nil {
				return
			}
		}
		t.Fatalf("a Writer that pins every entry wrote %d under a limit of %d open files: the limit is not in force", 2*openFiles, openFiles)
	})
	for _, s := range shapes {
		t.Run(s.name, func(t *testing.T) {
			parent := dir(t)
			names := manyNames(counts[s.name])
			lowerOpenFiles(t, openFiles)
			o := s.begin(t, parent, WriteOptions{})
			last := ""
			for _, name := range names {
				if d := dirOf(name); d != last {
					if err := o.MkdirAll(d); err != nil {
						t.Fatalf("MkdirAll(%q): %v", d, err)
					}
					last = d
				}
				if err := o.Create(name, false, strings.NewReader(name)); err != nil {
					t.Fatalf("Create(%q), after %d files: %v", name, len(o.Written()), err)
				}
			}
			if err := o.Commit(); err != nil {
				t.Fatal(err)
			}
			if n := countFiles(t, filepath.Join(parent, "out")); n != len(names) {
				t.Fatalf("the output holds %d files; want %d", n, len(names))
			}
		})
	}
	// A failed StageNew of the outputs limit removes all of it under the
	// same limit. It writes the entries as render's prepare does, with a
	// MkdirAll of each file's directory before the file, so all but the
	// first file of a directory take MkdirAll's path for a directory that
	// exists, under the limit, as often as render does.
	t.Run("StageNew abort", func(t *testing.T) {
		parent := dir(t)
		names := manyNames(counts["StageNew"])
		lowerOpenFiles(t, openFiles)
		o, err := StageNew(filepath.Join(parent, "a", "out"), WriteOptions{})
		if err != nil {
			t.Fatal(err)
		}
		for _, name := range names {
			if err := o.MkdirAll(dirOf(name)); err != nil {
				t.Fatalf("MkdirAll(%q), after %d files: %v", dirOf(name), len(o.Written()), err)
			}
			if err := o.Create(name, false, strings.NewReader("x")); err != nil {
				t.Fatal(err)
			}
		}
		if err := o.Abort(); err != nil {
			t.Fatal(err)
		}
		if got := tree(t, parent); len(got) != 0 {
			t.Fatalf("after Abort the parent holds %d names", len(got))
		}
	})
}

// withUmask sets the process umask to mask for the rest of t. No test in
// this package is parallel.
func withUmask(t *testing.T, mask int) {
	t.Helper()
	old := syscall.Umask(mask)
	t.Cleanup(func() { syscall.Umask(old) })
}

// unlock gives the owner read, write and search on every directory
// beneath root, top down, so the test can look at what a write under a
// restrictive umask made, and its temporary directory can be removed.
func unlock(t *testing.T, root string) {
	t.Helper()
	var walk func(p string)
	walk = func(p string) {
		info, err := os.Lstat(p)
		if err != nil || !info.IsDir() {
			return
		}
		if err := os.Chmod(p, info.Mode().Perm()|0o700); err != nil {
			t.Fatal(err)
		}
		entries, err := os.ReadDir(p)
		if err != nil {
			t.Fatal(err)
		}
		for _, e := range entries {
			walk(filepath.Join(p, e.Name()))
		}
	}
	walk(root)
}

// lstatModes records the permission bits of each name beneath root that
// Lstat reaches without reading a directory: the names are given.
func lstatModes(t *testing.T, root string, names []string) map[string]fs.FileMode {
	t.Helper()
	got := map[string]fs.FileMode{}
	for _, name := range names {
		info, err := os.Lstat(filepath.Join(root, filepath.FromSlash(name)))
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		got[name] = info.Mode().Perm()
	}
	return got
}

// W4-0.3: under a umask that withholds the owner's read, each shape writes,
// fails and cleans up as it does under any other, and what it leaves has
// its modes less that umask (ADR-005 D14 item 4). os.Root reaches a name
// by opening the directories above it for reading, so without its grant a
// write would be refused below its first directory.
func TestShapesUnderAUmaskWithoutOwnerRead(t *testing.T) {
	const mask = 0o477 // the owner keeps write and search, and nothing else is granted
	names := []string{"a", "a/b", "a/b/c.txt", "a/d.txt", "e.txt"}
	for _, s := range shapes {
		t.Run(s.name+" commit", func(t *testing.T) {
			parent := dir(t)
			t.Cleanup(func() { unlock(t, parent) })
			withUmask(t, mask)
			o := s.begin(t, parent, WriteOptions{})
			for _, name := range written {
				put(t, o, name, "<"+name+">")
			}
			if err := o.Commit(); err != nil {
				t.Fatal(err)
			}
			check := names
			if !s.exists {
				check = append([]string{"."}, names...)
			}
			got := lstatModes(t, filepath.Join(parent, "out"), check)
			for name, mode := range got {
				want := fs.FileMode(fileMode &^ mask)
				if !strings.HasSuffix(name, ".txt") {
					want = dirMode &^ mask
				}
				if mode != want {
					t.Errorf("%s: mode %v, want %v", name, mode, want)
				}
			}
			unlock(t, parent)
			for _, name := range written {
				p := filepath.Join(parent, "out", filepath.FromSlash(name))
				if err := os.Chmod(p, 0o600); err != nil {
					t.Fatal(err)
				}
				if got := readHost(t, p); got != "<"+name+">" {
					t.Errorf("%s holds %q", name, got)
				}
			}
		})
		t.Run(s.name+" failure", func(t *testing.T) {
			parent := dir(t)
			t.Cleanup(func() { unlock(t, parent) })
			files(t, parent, map[string]string{"theirs": "T"})
			withUmask(t, mask)
			o := s.begin(t, parent, WriteOptions{})
			for _, name := range written {
				put(t, o, name, "W")
			}
			wantReason(t, o.Create("a/b/half", false, failingReader{}), IOError, parent)
			if err := o.Abort(); err != nil {
				t.Fatal(err)
			}
			if !s.keeps {
				if got := tree(t, parent); !slices.Equal(got, []string{"theirs"}) {
					t.Fatalf("after Abort: %q, want only theirs", got)
				}
				return
			}
			if _, err := os.Lstat(filepath.Join(parent, "out", "a", "b", "half")); !os.IsNotExist(err) {
				t.Errorf("the failed file is still there: %v", err)
			}
			if got := lstatModes(t, filepath.Join(parent, "out"), []string{"a", "a/b"}); got["a"] != dirMode&^mask || got["a/b"] != dirMode&^mask {
				t.Errorf("after Abort the directories' modes are %v, want %v", got, fs.FileMode(dirMode&^mask))
			}
			unlock(t, parent)
			want := []string{"out/", "out/a/", "out/a/b/", "out/a/b/c.txt", "out/a/d.txt", "out/e.txt", "theirs"}
			if got := tree(t, parent); !slices.Equal(got, want) {
				t.Fatalf("after Abort: %q, want %q", got, want)
			}
		})
	}
}
