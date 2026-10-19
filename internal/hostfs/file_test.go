// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package hostfs

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"slices"
	"testing"
)

// writeOut opens a FileOut at path, naming name, writes body through it
// and returns the bytes it wrote.
func writeOut(t *testing.T, path, name, body string) (*FileOut, int64, error) {
	t.Helper()
	o, err := OpenFileOut(path, name)
	if err != nil {
		return nil, 0, err
	}
	defer o.Close()
	n, err := o.Write(func(w io.Writer) error {
		_, err := io.WriteString(w, body)
		return err
	})
	return o, n, err
}

// TestFileOutWrites is ADR-007 D7 item 5's write: a path naming a
// directory gets the file named name inside it; any other path is the
// file; an existing regular file is renamed over, never written into; and
// nothing is left beside it.
func TestFileOutWrites(t *testing.T) {
	root := dir(t)
	o, n, err := writeOut(t, root, "p.metais", "first")
	if err != nil || n != 5 || !o.InDirectory() || o.Name() != "p.metais" || o.Replaces() {
		t.Fatalf("into a directory: %v, %d, %+v", err, n, o)
	}
	target := filepath.Join(root, "p.metais")
	held, err := os.Open(target)
	if err != nil {
		t.Fatal(err)
	}
	defer held.Close()
	before, _ := held.Stat()
	o, _, err = writeOut(t, target, "ignored.metais", "second")
	if err != nil || o.InDirectory() || !o.Replaces() {
		t.Fatalf("over the file: %v, %+v", err, o)
	}
	after, _ := os.Lstat(target)
	b, _ := os.ReadFile(target)
	old, _ := io.ReadAll(held)
	if os.SameFile(before, after) || string(b) != "second" || string(old) != "first" {
		t.Errorf("the replace wrote into the file: now %q, the old handle reads %q", b, old)
	}
	if got := tree(t, root); !slices.Equal(got, []string{"p.metais"}) {
		t.Errorf("the directory holds %q", got)
	}
}

// TestFileOutRefuses is what OpenFileOut refuses, each with a reason and
// no path: a missing directory, a file above, a path ending in a separator
// that names no directory, a directory or a temporary file's leftover at
// the name, and a name hostfs will not resolve.
func TestFileOutRefuses(t *testing.T) {
	root := dir(t)
	files(t, root, map[string]string{"afile": "x", "out/" + TempName("p.metais"): "left"})
	if err := os.MkdirAll(filepath.Join(root, "dirs", "p.metais"), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct {
		path string
		want Reason
	}{
		{filepath.Join(root, "missing", "p.metais"), NotFound},
		{filepath.Join(root, "afile", "p.metais"), NotDirectory},
		{filepath.Join(root, "nodir") + string(filepath.Separator), NotFound},
		{filepath.Join(root, "dirs"), IsDirectory},
		{filepath.Join(root, "out"), Exists},
		{filepath.Join(root, "a:b"), BadName},
	} {
		_, err := OpenFileOut(c.path, "p.metais")
		wantReason(t, err, c.want, root)
	}
	if got := tree(t, filepath.Join(root, "out")); !slices.Equal(got, []string{TempName("p.metais")}) {
		t.Errorf("out holds %q", got)
	}
}

// TestFileOutWriteFails is a write that fails: the writer's own error is
// returned as it is, the temporary file is removed, and the target keeps
// its bytes.
func TestFileOutWriteFails(t *testing.T) {
	root := dir(t)
	files(t, root, map[string]string{"p.metais": "old"})
	o, err := OpenFileOut(root, "p.metais")
	if err != nil {
		t.Fatal(err)
	}
	defer o.Close()
	boom := errors.New("boom")
	if _, err := o.Write(func(w io.Writer) error {
		io.WriteString(w, "part")
		return boom
	}); err != boom {
		t.Errorf("Write returned %v, want the writer's own error", err)
	}
	if b, _ := os.ReadFile(filepath.Join(root, "p.metais")); string(b) != "old" {
		t.Errorf("the target holds %q", b)
	}
	if got := tree(t, root); !slices.Equal(got, []string{"p.metais"}) {
		t.Errorf("the directory holds %q", got)
	}
}

// TestFileOutWithin is ADR-007 D7 item 5's refusal of an output inside
// the source directory's layout trees, or at its manifest: found by
// identity, at any depth, and not for a file beside them.
func TestFileOutWithin(t *testing.T) {
	root := dir(t)
	src := filepath.Join(root, "src")
	files(t, src, map[string]string{"metais.yaml": "m", "templates/deep/x.liquid": "t", "assets/a": "a"})
	names := []string{"assets", "examples", "inputs", "metais.yaml", "references", "templates"}
	for _, c := range []struct {
		path, want string
	}{
		{filepath.Join(src, "templates", "deep"), "templates"},
		{filepath.Join(src, "assets", "p.metais"), "assets"},
		{filepath.Join(src, "metais.yaml"), "metais.yaml"},
		{src, ""},
		{root, ""},
	} {
		o, err := OpenFileOut(c.path, "p.metais")
		if err != nil {
			t.Fatal(err)
		}
		got, err := o.Within(src, names)
		o.Close()
		if err != nil || got != c.want {
			t.Errorf("%s: Within = %q, %v; want %q", c.path, got, err, c.want)
		}
	}
}
