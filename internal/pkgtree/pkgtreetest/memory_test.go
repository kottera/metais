// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package pkgtreetest

import (
	"bytes"
	"errors"
	"io"
	"strconv"
	"strings"
	"testing"

	"metais.org/metais/internal/pkgtree"
)

// samplePackage is a package's worth of entries: every top-level name of
// ADR-001 D5, nesting, a dot-file below the top level, an empty entry, a
// name beyond ASCII and an entry larger than any reader's first buffer.
func samplePackage() map[string][]byte {
	return map[string][]byte{
		".compiled/digests.json":              []byte(`{"entries":{}}`),
		"metais.yaml":                         []byte("metais: \"1.0\"\nname: report\n"),
		"inputs/customer.schema.json":         []byte(`{"type":"object"}`),
		"inputs/customer.md":                  []byte("# Customer\n"),
		"templates/report.md.liquid":          []byte("Hello {{ customer.name }}\n"),
		"templates/.github/workflows/ci.yml":  []byte("on: push\n"),
		"assets/empty.txt":                    {},
		"references/straße/naïve résumé.md":   []byte("Ünïcödé\n"),
		"examples/basic/inputs/customer.json": []byte(`{"name":"Ada"}`),
		"examples/basic/expected/report.md":   []byte("Hello Ada\n"),
		"references/large.bin":                bytes.Repeat([]byte{0, 1, 2, 3, 0xff}, 40000),
	}
}

func TestNewMeetsContract(t *testing.T) {
	files := samplePackage()
	tree, err := New(files)
	if err != nil {
		t.Fatal(err)
	}
	if err := Check(tree, files); err != nil {
		t.Error(err)
	}
}

func TestNewEmptyMeetsContract(t *testing.T) {
	for _, files := range []map[string][]byte{nil, {}} {
		tree, err := New(files)
		if err != nil {
			t.Fatal(err)
		}
		if got := tree.List(); len(got) != 0 {
			t.Errorf("List() = %q, want nothing", got)
		}
		if err := Check(tree, files); err != nil {
			t.Error(err)
		}
	}
}

// TestNewNamesThatOnlyLookAlike builds a tree whose names a lookup by
// prefix, case or trimming would confuse, and checks each is its own entry.
func TestNewNamesThatOnlyLookAlike(t *testing.T) {
	files := map[string][]byte{
		"a":     []byte("1"),
		"a.b":   []byte("2"),
		"a-b/c": []byte("3"),
		"ab/c":  []byte("4"),
		"A/c":   []byte("5"),
		"b c":   []byte("6"),
	}
	tree, err := New(files)
	if err != nil {
		t.Fatal(err)
	}
	if err := Check(tree, files); err != nil {
		t.Error(err)
	}
}

func TestNewRefuses(t *testing.T) {
	tests := []struct {
		name  string
		paths []string
		bad   string // the path the refusal names
		why   string
	}{
		{"empty", []string{""}, "", "is empty"},
		{"invalid UTF-8", []string{"a/\xff"}, "a/\xff", "is not valid UTF-8"},
		{"byte order mark", []string{"\uFEFFa"}, "\uFEFFa", "byte order mark"},
		{"leading slash", []string{"/a"}, "/a", "begins with /"},
		{"trailing slash", []string{"a/"}, "a/", "ends with /"},
		{"lone slash", []string{"/"}, "/", "begins with /"},
		{"empty segment", []string{"a//b"}, "a//b", "empty segment"},
		{"dot segment", []string{"a/./b"}, "a/./b", ". segment"},
		{"leading dot segment", []string{"./a"}, "./a", ". segment"},
		{"dot-dot segment", []string{"a/../b"}, "a/../b", ".. segment"},
		{"lone dot-dot", []string{".."}, "..", ".. segment"},
		{"backslash", []string{`a\b`}, `a\b`, `holds \`},
		{"colon", []string{"C:/a"}, "C:/a", "holds :"},
		{"file and directory", []string{"a/b", "a/b/c"}, "a/b", `directory of "a/b/c"`},
		{"file and distant directory", []string{"a", "a b", "a.c", "a/z"}, "a", `directory of "a/z"`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			files := map[string][]byte{}
			for _, p := range tt.paths {
				files[p] = []byte("x")
			}
			tree, err := New(files)
			if err == nil {
				t.Fatalf("New(%q) built a tree", tt.paths)
			}
			if tree != nil {
				t.Errorf("New(%q) returned a tree beside its error", tt.paths)
			}
			want := "entry path " + strconv.Quote(tt.bad) + " "
			if msg := err.Error(); !strings.Contains(msg, want) || !strings.Contains(msg, tt.why) {
				t.Errorf("New(%q) = %q, want it to name %s and say %q", tt.paths, msg, want, tt.why)
			}
		})
	}
}

// TestNewReportsEveryBadPath checks New reports every refused path at once,
// in path order, rather than stopping at the first.
func TestNewReportsEveryBadPath(t *testing.T) {
	_, err := New(map[string][]byte{
		"z/../a":  nil,
		"good":    nil,
		"/abs":    nil,
		"dir":     nil,
		"dir/sub": nil,
	})
	if err == nil {
		t.Fatal("New built a tree from three bad paths")
	}
	lines := strings.Split(err.Error(), "\n")
	want := []string{`"/abs"`, `"dir"`, `"z/../a"`}
	if len(lines) != len(want) {
		t.Fatalf("New reported %d problems, want %d:\n%s", len(lines), len(want), err)
	}
	for i, w := range want {
		if !strings.Contains(lines[i], w) {
			t.Errorf("problem %d = %q, want it to name %s", i, lines[i], w)
		}
	}
}

// TestNewCopies checks a tree is fixed once built: changing the map or the
// bytes it was built from changes nothing the tree returns.
func TestNewCopies(t *testing.T) {
	content := []byte("original")
	files := map[string][]byte{"a": content}
	tree, err := New(files)
	if err != nil {
		t.Fatal(err)
	}
	copy(content, "spoiled!")
	files["b"] = []byte("added")
	delete(files, "a")

	if err := Check(tree, map[string][]byte{"a": []byte("original")}); err != nil {
		t.Error(err)
	}
}

// TestRefusalIsPathError pins the refusal's form beside what Check asserts:
// the error a caller sees, and that no reader leaks.
func TestRefusalIsPathError(t *testing.T) {
	tree, err := New(map[string][]byte{"inputs/a.json": []byte("{}")})
	if err != nil {
		t.Fatal(err)
	}
	r, err := tree.Open("inputs")
	if r != nil {
		t.Error("Open of a directory prefix returned a reader")
	}
	var pe *pkgtree.PathError
	if !errors.As(err, &pe) || pe.Op != "open" || pe.Path != "inputs" || !errors.Is(err, pkgtree.ErrNotExist) {
		t.Fatalf("Open(%q) = %v, want a *pkgtree.PathError for open of it wrapping ErrNotExist", "inputs", err)
	}
	if got, want := err.Error(), `open "inputs": no such entry`; got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
	if _, err := tree.Stat("INPUTS/A.JSON"); !errors.Is(err, pkgtree.ErrNotExist) {
		t.Errorf("Stat of a case variant = %v, want ErrNotExist", err)
	}
	r, err = tree.Open("inputs/a.json")
	if err != nil {
		t.Fatal(err)
	}
	b, err := io.ReadAll(r)
	if err != nil || string(b) != "{}" {
		t.Errorf("reading inputs/a.json = %q, %v", b, err)
	}
	if err := r.Close(); err != nil {
		t.Errorf("Close() = %v", err)
	}
}
