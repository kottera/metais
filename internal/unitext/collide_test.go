// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package unitext

import (
	"errors"
	"fmt"
	"slices"
	"testing"

	"github.com/kottera/metais/internal/limits"
)

func collisionText(cs []Collision) []string {
	names := map[CollisionKind]string{Duplicate: "duplicate", CaseCollision: "case", FileDirectoryConflict: "file_dir"}
	var out []string
	for _, c := range cs {
		out = append(out, fmt.Sprintf("%s %s(%s) %s(%s)", names[c.Kind], c.Path, c.Member, c.Other, c.OtherMember))
	}
	return out
}

// TestCollisions holds Collisions to ADR-005 D12 items 6 and 7 and ADR-001
// D2 item 4.
func TestCollisions(t *testing.T) {
	for _, c := range []struct {
		paths []string
		want  []string
	}{
		{[]string{"a", "b", "c/d", "c/e", "c/f/g"}, nil},
		{[]string{"docs/a", "docs/b", "docs/c/d"}, nil},
		{[]string{"a", "a"}, []string{"duplicate a(a) a(a)"}},
		{[]string{"Readme.md", "README.md"}, []string{"case README.md(README.md) Readme.md(Readme.md)"}},
		{[]string{"straße.md", "STRASSE.md"}, []string{"case STRASSE.md(STRASSE.md) straße.md(straße.md)"}},
		{[]string{"Docs/a", "docs/b"}, []string{"case docs/b(docs) Docs/a(Docs)"}},
		// One collision per member, however many paths bring it.
		{[]string{"Docs/a", "docs/b", "docs/c", "docs/d"}, []string{"case docs/b(docs) Docs/a(Docs)"}},
		{[]string{"a/b", "a/b/c"}, []string{"file_dir a/b/c(a/b) a/b(a/b)"}},
		{[]string{"a/b/c", "a/b"}, []string{"file_dir a/b(a/b) a/b/c(a/b)"}},
		{[]string{"Docs", "docs/a.md"}, []string{"file_dir docs/a.md(docs) Docs(Docs)"}},
		{[]string{"x/café", "x/CAFÉ"}, []string{"case x/CAFÉ(x/CAFÉ) x/café(x/café)"}},
		{[]string{"A/b", "a/B"}, []string{"case a/B(a) A/b(A)"}},
		{[]string{"\xff", "\xff"}, nil},
		// A deeper collision under colliding directories is theirs.
		{[]string{"Dir/x/f", "dir/X/F", "dir/x/g", "dir/x/G"}, []string{"case dir/X/F(dir) Dir/x/f(Dir)", "case dir/x/g(dir/x) dir/X/F(dir/X)", "case dir/x/G(dir/x/G) dir/x/g(dir/x/g)"}},
	} {
		if got := collisionText(Collisions(c.paths)); !slices.Equal(got, c.want) {
			t.Errorf("Collisions(%q) = %q, want %q", c.paths, got, c.want)
		}
	}
}

// TestCollisionsBounded holds the collision set to one member per path
// segment: a tree of n paths of depth d does work linear in n×d, and many
// paths sharing a colliding directory report it once.
func TestCollisionsBounded(t *testing.T) {
	var paths []string
	for i := range 2000 {
		paths = append(paths, fmt.Sprintf("Dir/sub/%d", i), fmt.Sprintf("dir/sub/%d", i))
	}
	got := Collisions(paths)
	if len(got) != 1 || got[0].Member != "dir" {
		t.Errorf("Collisions over 4000 paths in two colliding directories reported %d collisions, want the one directory", len(got))
	}
}

// TestCheckTree composes the predicate over a tree: nil for a tree that
// passes, and every path's problems and every collision at once otherwise.
func TestCheckTree(t *testing.T) {
	lim := limits.Defaults()
	if err := CheckTree([]string{"metais.yaml", "templates/a.md", "templates/b.md"}, lim); err != nil {
		t.Errorf("a portable tree: %v", err)
	}
	err := CheckTree([]string{"CON", "a<b", "x/A", "x/a", "ok"}, lim)
	var te *TreeError
	if !errors.As(err, &te) {
		t.Fatalf("CheckTree = %v, want a *TreeError", err)
	}
	if len(te.Paths) != 2 || te.Paths[0].Path != "CON" || te.Paths[1].Path != "a<b" || len(te.Collisions) != 1 {
		t.Errorf("CheckTree found %+v", te)
	}
}
