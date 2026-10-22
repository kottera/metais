// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

//go:build linux || darwin

package author

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"syscall"
	"testing"

	"metais.org/metais/internal/hostfs"
	"metais.org/metais/internal/limits"
	"metais.org/metais/internal/pkgtree"
	"metais.org/metais/internal/pkgtree/pkgtreetest"
	"metais.org/metais/internal/testenv"
)

// openFiles is the open-file soft limit W4-5.3's test unpacks under: far
// below the entries limit, and below darwin's default of 256. No test in
// this package is parallel, so nothing else runs under it.
const openFiles = 128

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

// fullTree is a package of as many entries as the entries limit admits,
// .compiled/'s four among them, a thousand files to a directory.
func fullTree(t *testing.T) (pkgtree.Tree, int) {
	t.Helper()
	files := map[string][]byte{}
	for _, f := range []string{".compiled/digests.json", ".compiled/manifest.json", ".compiled/references.json", ".compiled/schema.json", "metais.yaml"} {
		files[f] = []byte("x\n")
	}
	n := int(limits.Get(limits.Entries).Default) - len(files)
	for i := range n {
		files[fmt.Sprintf("references/d%02d/f%05d.txt", i/1000, i)] = []byte(fmt.Sprint(i))
	}
	tr, err := pkgtreetest.New(files)
	if err != nil {
		t.Fatal(err)
	}
	return tr, n + 1
}

// countFiles is the number of regular files beneath dir.
func countFiles(t *testing.T, dir string) int {
	t.Helper()
	n := 0
	err := filepath.WalkDir(dir, func(_ string, d fs.DirEntry, err error) error {
		if err == nil && d.Type().IsRegular() {
			n++
		}
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	return n
}

// TestUnpackWritesTheEntriesLimitUnderTheOpenFileLimit is W4-5.3's first
// constraint: an unpack of a package of as many entries as ADR-009 D3's
// entries limit admits succeeds, staged and in place, with the open-file
// soft limit far below that number. No limit is lowered to fit (R1-015).
// So does a staged unpack that fails at its last entry, which removes
// everything it wrote under the same limit.
func TestUnpackWritesTheEntriesLimitUnderTheOpenFileLimit(t *testing.T) {
	tr, n := fullTree(t)
	t.Run("staged", func(t *testing.T) {
		to := filepath.Join(testenv.Dir(t), "src")
		lowerOpenFiles(t, openFiles)
		files, err := Unpack(tr, to, 0)
		if err != nil || len(files) != n {
			t.Fatalf("Unpack: %v, after %d of %d files", err, len(files), n)
		}
		if got := countFiles(t, to); got != n {
			t.Errorf("the target holds %d files, want %d", got, n)
		}
	})
	t.Run("in place", func(t *testing.T) {
		to := testenv.Dir(t)
		lowerOpenFiles(t, openFiles)
		files, err := Unpack(tr, to, 0)
		if err != nil || len(files) != n {
			t.Fatalf("Unpack: %v, after %d of %d files", err, len(files), n)
		}
		if got := countFiles(t, to); got != n {
			t.Errorf("the target holds %d files, want %d", got, n)
		}
	})
	t.Run("staged failure", func(t *testing.T) {
		root := testenv.Dir(t)
		last := UnpackedEntries(tr)[n-1]
		lowerOpenFiles(t, openFiles)
		_, err := Unpack(failingTree{tr, last}, filepath.Join(root, "src"), 0)
		var we *WriteError
		if !errors.As(err, &we) || we.Reason != hostfs.IOError {
			t.Fatalf("Unpack: %v", err)
		}
		if got := tree(t, root); len(got) != 0 {
			t.Errorf("after the failure: %d entries left, the first %q", len(got), got[0])
		}
	})
}

// unlock gives the owner read, write and search on every directory
// beneath root, so a test can look at what a write under a restrictive
// umask made and its temporary directory can be removed.
func unlock(t *testing.T, root string) {
	t.Helper()
	filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err == nil && d.IsDir() {
			info, _ := d.Info()
			os.Chmod(p, info.Mode().Perm()|0o700)
		}
		return nil
	})
}

// TestUnpackUnderAUmaskWithoutOwnerRead is W4-5.3's third constraint: an
// unpack under a umask that withholds the owner's read succeeds, staged
// and in place, and leaves the modes ADR-007 D8 item 4 states less that
// umask. Nothing depends on the umask granting the owner read.
func TestUnpackUnderAUmaskWithoutOwnerRead(t *testing.T) {
	const mask = 0o477
	for _, c := range []struct {
		name string
		to   func(root string) string
	}{
		{"staged", func(root string) string { return filepath.Join(root, "a", "src") }},
		{"in place", func(root string) string { return root }},
	} {
		t.Run(c.name, func(t *testing.T) {
			root := testenv.Dir(t)
			t.Cleanup(func() { unlock(t, root) })
			to := c.to(root)
			old := syscall.Umask(mask)
			files, err := Unpack(packedTree(t), to, 0)
			syscall.Umask(old)
			if err != nil || !slices.Equal(files, unpackedFiles) {
				t.Fatalf("Unpack: %v, %q", err, files)
			}
			var got []string
			filepath.WalkDir(to, func(p string, d fs.DirEntry, err error) error {
				if err != nil || p == to && c.name == "in place" {
					return nil
				}
				info, _ := os.Lstat(p)
				rel, _ := filepath.Rel(to, p)
				want := fs.FileMode(0o644 &^ mask)
				if d.IsDir() {
					want = 0o755 &^ mask
				}
				if info.Mode().Perm() != want {
					t.Errorf("%s: mode %v, want %v", rel, info.Mode().Perm(), want)
				}
				got = append(got, rel)
				return nil
			})
			unlock(t, root)
			for _, f := range unpackedFiles {
				p := filepath.Join(to, filepath.FromSlash(f))
				os.Chmod(p, 0o600)
				if b, _ := os.ReadFile(p); string(b) != string(packedFiles[f]) {
					t.Errorf("%s holds %q", f, b)
				}
			}
			if len(got) == 0 {
				t.Error("nothing was walked")
			}
		})
	}
}

// writable reports whether this process can create a file in a directory
// of mode 0555, as root can.
func writable(t *testing.T) bool {
	t.Helper()
	d := testenv.Dir(t)
	if err := os.Chmod(d, 0o555); err != nil {
		t.Fatal(err)
	}
	defer os.Chmod(d, 0o755)
	err := os.WriteFile(filepath.Join(d, "probe"), nil, 0o644)
	return err == nil
}

// TestUnpackIntoAnUnwritableDirectory is output.destination.nothing_written's
// permission_denied: an unpack into an empty directory this process may
// not write, and a staged unpack beneath one, each stop at their first
// write with nothing written, and nothing is left, which the error says.
func TestUnpackIntoAnUnwritableDirectory(t *testing.T) {
	if writable(t) {
		t.Skip("a directory of mode 0555 is writable to this process, as to root")
	}
	for _, c := range []struct {
		name string
		to   func(locked string) string
	}{
		{"in place", func(locked string) string { return locked }},
		{"staged", func(locked string) string { return filepath.Join(locked, "src") }},
	} {
		t.Run(c.name, func(t *testing.T) {
			locked := testenv.Dir(t)
			if err := os.Chmod(locked, 0o555); err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { os.Chmod(locked, 0o755) })
			files, err := Unpack(packedTree(t), c.to(locked), 0)
			var we *WriteError
			if !errors.As(err, &we) || we.Reason != hostfs.PermissionDenied || len(we.Written) != 0 || we.Left || len(files) != 0 {
				t.Fatalf("Unpack: %v, %#v, %q", err, we, files)
			}
			if got := tree(t, locked); len(got) != 0 {
				t.Errorf("left %q", got)
			}
			if fi, _ := os.Stat(locked); fi.Mode().Perm() != 0o555 {
				t.Errorf("the directory's mode is %v, want it left 0555", fi.Mode().Perm())
			}
		})
	}
}
