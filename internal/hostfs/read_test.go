// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package hostfs

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func mustOpenDir(t *testing.T, path string) *Dir {
	t.Helper()
	d, err := OpenDir(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { d.Close() })
	return d
}

// mustNotOpen makes d fail t if it ever gets as far as opening a file.
func mustNotOpen(t *testing.T, d *Dir) {
	d.beforeOpen = func(name string) {
		t.Errorf("opened %q, which its Lstat mode refuses", name)
	}
}

func TestReadFile(t *testing.T) {
	root := dir(t)
	files(t, root, map[string]string{"a.json": "{}", "c/d.csv": "x,y\n", "five": "12345"})
	d := mustOpenDir(t, root)
	for name, want := range map[string]string{"a.json": "{}", "c/d.csv": "x,y\n"} {
		got, err := d.ReadFile(name, 1<<20)
		if err != nil || string(got) != want {
			t.Errorf("ReadFile(%q) = %q, %v", name, got, err)
		}
	}
	if got, err := d.ReadFile("five", 5); err != nil || string(got) != "12345" {
		t.Errorf("a file of exactly its limit: %q, %v", got, err)
	}
	_, err := d.ReadFile("five", 4)
	wantReason(t, err, TooLarge, root, "five")
}

// The read stops at its limit plus one byte, whatever the source holds: a
// reader that never ends is read exactly that far. The bound is a count
// of bytes, not a time.
func TestReadStopsAtLimitPlusOne(t *testing.T) {
	for _, limit := range []int64{0, 1, 4096, 1 << 20} {
		src := &endless{}
		_, err := readLimited(src, limit)
		wantReason(t, err, TooLarge)
		if src.n != limit+1 {
			t.Errorf("limit %d: read %d bytes, want %d", limit, src.n, limit+1)
		}
	}
}

type endless struct{ n int64 }

func (e *endless) Read(p []byte) (int, error) {
	for i := range p {
		p[i] = 'x'
	}
	e.n += int64(len(p))
	return len(p), nil
}

func TestReadRefusals(t *testing.T) {
	root := dir(t)
	outside := testDirOutside(t)
	files(t, root, map[string]string{"in.json": "{}", "sub/x": "x"})
	files(t, outside, map[string]string{"secret": "s"})
	symlink(t, root, "in.json", "inside-link")
	symlink(t, root, filepath.Join(outside, "secret"), "outside-link")
	symlink(t, root, "sub", "dir-link")
	d := mustOpenDir(t, root)
	mustNotOpen(t, d)
	cases := []struct {
		name string
		want Reason
	}{
		{"inside-link", Link},
		{"outside-link", Link},
		{"dir-link", Link},
		{"dir-link/x", Link},
		{"sub", IsDirectory},
		{"absent", NotFound},
		{"in.json/x", NotFound},
		{"../" + filepath.Base(outside) + "/secret", BadName},
		{"/etc/passwd", BadName},
		{`sub\x`, BadName},
		{"", BadName},
		{"sub//x", BadName},
	}
	for _, c := range cases {
		_, err := d.ReadFile(c.name, 1<<20)
		t.Run(c.name, func(t *testing.T) { wantReason(t, err, c.want, root, outside) })
	}
}

// testDirOutside is a second temporary directory, beside the one under
// test, for what must stay out of reach.
func testDirOutside(t *testing.T) string {
	t.Helper()
	return t.TempDir()
}

func TestReadPermissionDenied(t *testing.T) {
	root := dir(t)
	files(t, root, map[string]string{"locked": "x"})
	p := filepath.Join(root, "locked")
	if err := os.Chmod(p, 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(p, 0o644) })
	if f, err := os.Open(p); err == nil {
		f.Close()
		t.Skip("a mode-0 file is readable here, as it is to root")
	}
	_, err := mustOpenDir(t, root).ReadFile("locked", 10)
	wantReason(t, err, PermissionDenied, root, "locked")
}

// A file swapped for a link between the Lstat that admits it and the open
// is refused: the open follows the link, inside the root, but the handle
// is not the file Lstat described. Without os.SameFile this test reads the
// other file's bytes.
func TestReadSwappedForLinkIsChanged(t *testing.T) {
	root := dir(t)
	files(t, root, map[string]string{"input.json": "{}", "other.json": "OTHER"})
	d := mustOpenDir(t, root)
	d.beforeOpen = func(name string) {
		p := filepath.Join(root, name)
		if err := os.Remove(p); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink("other.json", p); err != nil {
			t.Skipf("cannot make a symbolic link here: %v", err)
		}
	}
	got, err := d.ReadFile("input.json", 1<<20)
	if bytes.Contains(got, []byte("OTHER")) {
		t.Fatal("read the bytes of the file a link swapped in")
	}
	wantReason(t, err, Changed, root)
}

// A file swapped for another regular file in the same window is refused
// too: same kind, different file.
func TestReadSwappedForFileIsChanged(t *testing.T) {
	root := dir(t)
	files(t, root, map[string]string{"input.json": "{}"})
	d := mustOpenDir(t, root)
	d.beforeOpen = func(name string) {
		p := filepath.Join(root, name)
		if err := os.Rename(p, p+".old"); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte("[]"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	_, err := d.ReadFile("input.json", 1<<20)
	wantReason(t, err, Changed, root)
}

func TestList(t *testing.T) {
	root := dir(t)
	files(t, root, map[string]string{"b": "12", "a/x": "", "c": ""})
	symlink(t, root, "b", "l")
	d := mustOpenDir(t, root)
	got, err := d.List(".", 10)
	if err != nil {
		t.Fatal(err)
	}
	want := []Entry{{"a", KindDir, 0}, {"b", KindFile, 2}, {"c", KindFile, 0}, {"l", KindLink, 0}}
	if len(got) != len(want) {
		t.Fatalf("List = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("List[%d] = %v, want %v", i, got[i], want[i])
		}
	}
	if _, err := d.List(".", 3); err == nil {
		t.Error("a directory of 4 entries listed under a bound of 3")
	} else {
		wantReason(t, err, TooLarge, root)
	}
	_, err = d.List("l", 10)
	wantReason(t, err, Link, root)
	_, err = d.List("b", 10)
	wantReason(t, err, NotDirectory, root)
}

func TestOpenDirRefusesAFile(t *testing.T) {
	root := dir(t)
	files(t, root, map[string]string{"f": ""})
	_, err := OpenDir(filepath.Join(root, "f"))
	wantReason(t, err, NotDirectory, root)
	_, err = OpenDir(filepath.Join(root, "absent"))
	wantReason(t, err, NotFound, root)
}

// The caller's own directory may be a link: it named it (ADR-009 D7 item
// 4).
func TestOpenDirFollowsTheCallersLink(t *testing.T) {
	root := dir(t)
	files(t, root, map[string]string{"real/a": "A"})
	symlink(t, root, "real", "named")
	got, err := mustOpenDir(t, filepath.Join(root, "named")).ReadFile("a", 10)
	if err != nil || string(got) != "A" {
		t.Fatalf("ReadFile through the caller's link = %q, %v", got, err)
	}
}

func TestOpenFile(t *testing.T) {
	root := dir(t)
	files(t, root, map[string]string{"p.metais": "0123456789", "d/x": ""})
	symlink(t, root, "p.metais", "named.metais")
	for _, name := range []string{"p.metais", "named.metais"} {
		f, err := OpenFile(filepath.Join(root, name))
		if err != nil {
			t.Fatal(err)
		}
		if f.Size() != 10 {
			t.Errorf("Size = %d", f.Size())
		}
		buf := make([]byte, 4)
		if n, err := f.ReadAt(buf, 3); n != 4 || err != nil || string(buf) != "3456" {
			t.Errorf("ReadAt(3) = %d %q %v", n, buf, err)
		}
		if n, err := f.ReadAt(buf, 8); n != 2 || err != io.EOF || string(buf[:n]) != "89" {
			t.Errorf("ReadAt(8) = %d %q %v", n, buf[:n], err)
		}
		if _, err := f.ReadAt(buf, 10); err != io.EOF {
			t.Errorf("ReadAt(10) = %v", err)
		}
		f.Close()
	}
	_, err := OpenFile(filepath.Join(root, "d"))
	wantReason(t, err, IsDirectory, root)
	_, err = OpenFile(filepath.Join(root, "absent"))
	wantReason(t, err, NotFound, root)
}

// Bytes appended to a package file after it was opened are not read: the
// file is the size it had.
func TestOpenFileKeepsItsSize(t *testing.T) {
	root := dir(t)
	files(t, root, map[string]string{"p": "abc"})
	f, err := OpenFile(filepath.Join(root, "p"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := os.WriteFile(filepath.Join(root, "p"), []byte("abcdef"), 0o644); err != nil {
		t.Fatal(err)
	}
	buf := make([]byte, 8)
	if n, err := f.ReadAt(buf, 0); n != 3 || err != io.EOF {
		t.Errorf("ReadAt = %d %q %v, want the 3 bytes it had", n, buf[:n], err)
	}
	if err := os.WriteFile(filepath.Join(root, "p"), []byte("a"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err = f.ReadAt(buf[:3], 0)
	wantReason(t, err, Changed, root)
}
