// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package hostfs

import (
	"bytes"
	"errors"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"metais.org/metais/internal/limits"
	"metais.org/metais/internal/pkgtree/pkgtreetest"
	"metais.org/metais/internal/unitext"
	"metais.org/metais/internal/zipfmt"
)

// The meeting of W2-4 and W2-5 (PLAN-2026-10-metais-v1 §8): the
// source-directory tree takes ADR-001 D2's name rule through
// SourceOptions.Names, and the one portable-path predicate, unitext's, is
// that rule; and the source tree and the packed tree meet one contract.

// predicateOptions are the defaults with the predicate as the name rule,
// as the loader composes them.
func predicateOptions() SourceOptions {
	lim := limits.Defaults()
	return SourceOptions{Limits: lim, Names: func(paths []string) error { return unitext.CheckTree(paths, lim) }}
}

// TestSourceNamesThroughPredicate: a source directory holding a name ADR-001
// D2 refuses is refused, with the predicate's own answer inside the
// *SourceError, and the directory without it opens.
func TestSourceNamesThroughPredicate(t *testing.T) {
	for name, bad := range map[string]string{
		"a reserved device name":    "templates/CON.md",
		"a .git segment":            "assets/.git/config",
		"a bidirectional control":   "references/a\u202etxt.md",
		"not NFC":                   "templates/café.md",
		"a segment ending in a dot": "assets/a./b",
	} {
		root := dir(t)
		files(t, root, packageFiles)
		if s, err := OpenSource(root, predicateOptions()); err != nil {
			t.Fatalf("%s: the clean directory: %v", name, err)
		} else {
			s.Close()
		}
		files(t, root, map[string]string{bad: "x"})
		got := tree(t, root)
		if !slices.Contains(got, bad) {
			t.Logf("%s: this file system did not keep the name %+q as written; not run", name, bad)
			continue
		}
		se := refused(t, root, predicateOptions())
		var te *unitext.TreeError
		if !errors.As(se, &te) {
			t.Fatalf("%s: the refusal does not carry the predicate's answer: %v", name, se)
		}
		if len(te.Paths) != 1 || te.Paths[0].Path != bad || len(se.Problems) != 0 {
			t.Errorf("%s: refused with %+v and %+v, want the predicate's refusal of %+q alone", name, te.Paths, se.Problems, bad)
		}
	}
}

// TestSourceCollisionsThroughPredicate: two names that collide caselessly
// are refused as a collision. "straße" and "STRASSE" match only under full
// case folding, so a file system that folds case simply holds both; a
// case-only pair needs a case-sensitive one.
func TestSourceCollisionsThroughPredicate(t *testing.T) {
	ran := 0
	for _, pair := range [][2]string{
		{"templates/straße.md", "templates/STRASSE.md"},
		{"templates/Readme.md", "templates/README.md"},
	} {
		root := dir(t)
		files(t, root, packageFiles)
		files(t, root, map[string]string{pair[0]: "a"})
		if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(pair[1])), []byte("b"), 0o644); err != nil {
			t.Fatal(err)
		}
		if got := tree(t, root); !slices.Contains(got, pair[0]) || !slices.Contains(got, pair[1]) {
			t.Logf("this file system holds one name for %q; not run", pair)
			continue
		}
		ran++
		se := refused(t, root, predicateOptions())
		var te *unitext.TreeError
		if !errors.As(se, &te) || len(te.Collisions) != 1 || te.Collisions[0].Kind != unitext.CaseCollision {
			t.Errorf("%q: refused with %+v, want one case collision", pair, se)
		}
	}
	if ran == 0 {
		t.Skip("no pair could be held by this file system")
	}
}

// TestBothTreesMeetTheContract runs pkgtreetest.Check, unchanged, over the
// source tree of a package's files and over the packed tree of the same
// files: neither item extended Check, and both trees pass it.
func TestBothTreesMeetTheContract(t *testing.T) {
	root := dir(t)
	files(t, root, packageFiles)
	want := map[string][]byte{}
	for p, b := range packageFiles {
		want[p] = []byte(b)
	}
	s := openSource(t, root, predicateOptions())
	if err := pkgtreetest.Check(s, want); err != nil {
		t.Errorf("the source tree: %v", err)
	}

	var entries []zipfmt.Entry
	for _, p := range slices.Sorted(maps.Keys(want)) {
		entries = append(entries, zipfmt.Entry{Name: p, Content: want[p], Method: zipfmt.Auto})
	}
	var buf bytes.Buffer
	if err := zipfmt.Write(&buf, entries); err != nil {
		t.Fatal(err)
	}
	a, found, err := zipfmt.Open(bytes.NewReader(buf.Bytes()), int64(buf.Len()), limits.Defaults())
	if err != nil || len(found) > 0 {
		t.Fatalf("zipfmt.Open: %v, %d findings", err, len(found))
	}
	if err := pkgtreetest.Check(a, want); err != nil {
		t.Errorf("the packed tree: %v", err)
	}
	if !slices.Equal(s.List(), a.List()) {
		t.Errorf("the trees list %q and %q", s.List(), a.List())
	}
}
