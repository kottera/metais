// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package finding

import (
	"strings"
	"testing"
)

// TestPathParts: a path's parent, last token and key are those of the
// pointer it names, however it was built.
func TestPathParts(t *testing.T) {
	var root *Path
	p := root.Child("a/b").Child("~")
	if q, _ := ParsePath("/a~1b/~0"); q.Key() != p.Key() || q.String() != "/a~1b/~0" {
		t.Errorf("the parsed path is %q, key equal %v", q.String(), q.Key() == p.Key())
	}
	if p.Last() != "~" || p.Parent().Last() != "a/b" || p.Parent().Parent() != nil {
		t.Errorf("parts of %q: last %q, parent's last %q", p.String(), p.Last(), p.Parent().Last())
	}
	if root.Parent() != nil || root.Last() != "" || root.Key() != (PathKey{}) {
		t.Errorf("the root has a parent, a token or a key")
	}
	if root.Child("a").Key() == root.Child("b").Key() || root.Child("a").Key() == p.Key() {
		t.Errorf("different paths share a key")
	}
}

// TestTokenCachePaths: a path the cache makes is the one Child makes, from
// a string given once or many times, and under a prefix.
func TestTokenCachePaths(t *testing.T) {
	var c TokenCache
	long := strings.Repeat("~/é", 100)
	under, _ := ParsePath("/base")
	for _, toks := range [][]string{nil, {""}, {long, "x"}, {long, "y"}, {"bad\xff", long}} {
		var want *Path
		for _, tok := range append([]string{"base"}, toks...) {
			want = want.Child(tok)
		}
		for range 2 {
			got := c.Path(under, toks)
			if got.Key() != want.Key() || got.String() != want.String() || got.Depth() != want.Depth() {
				t.Errorf("%q: made %q, want %q", toks, got.String(), want.String())
			}
		}
	}
	if got := c.Path(nil, nil); got != nil {
		t.Errorf("no tokens made %q, want the root", got.String())
	}
}

// TestResolver: a path resolves to what its longest prefix that names
// something names, and a token is stepped through once however many
// paths below it are resolved.
func TestResolver(t *testing.T) {
	tree := map[string]map[string]bool{"long": {"a": true, "b": true}}
	steps := map[string]int{}
	type node struct {
		name string
		kids map[string]bool
	}
	r := NewResolver(node{}, func(n node, tok string) (node, bool) {
		steps[tok]++
		switch {
		case n.name == "":
			kids, ok := tree[tok]
			return node{tok, kids}, ok
		case n.kids[tok]:
			return node{name: n.name + "/" + tok}, true
		}
		return node{}, false
	})
	var root *Path
	for _, tt := range []struct {
		path  *Path
		name  string
		depth int
	}{
		{root, "", 0},
		{root.Child("long").Child("a"), "long/a", 2},
		{root.Child("long").Child("b"), "long/b", 2},
		{root.Child("long").Child("c"), "long", 1},
		{root.Child("long").Child("c").Child("d"), "long", 1},
		{root.Child("nope").Child("a"), "", 0},
		{root.Child("long").Child("a"), "long/a", 2},
	} {
		n, depth := r.Resolve(tt.path)
		if n.name != tt.name || depth != tt.depth {
			t.Errorf("%q resolves to %q at %d, want %q at %d", tt.path.String(), n.name, depth, tt.name, tt.depth)
		}
	}
	if steps["long"] != 1 || steps["a"] != 1 || steps["c"] != 1 || steps["d"] != 0 {
		t.Errorf("the steps taken by token: %v", steps)
	}
}

// pointerPositions is a Positions that reads only pointers in their
// JSON-string form.
type pointerPositions map[string]Pos

func (p pointerPositions) Value(ptr string) (Pos, bool) { v, ok := p[ptr]; return v, ok }
func (p pointerPositions) Key(ptr string) (Pos, bool)   { v, ok := p["key "+ptr]; return v, ok }

// TestValueAtPointerPositions: a Positions that reads no Path is asked for
// the pointer the path names.
func TestValueAtPointerPositions(t *testing.T) {
	pos := pointerPositions{"/a~1b": {Line: 2, Column: 3}, "key /a~1b": {Line: 2, Column: 1}}
	p := (*Path)(nil).Child("a/b")
	if v, ok := ValueAt(pos, p); !ok || v != (Pos{Line: 2, Column: 3}) {
		t.Errorf("value at %v, %v", v, ok)
	}
	if v, ok := KeyAt(pos, p); !ok || v != (Pos{Line: 2, Column: 1}) {
		t.Errorf("key at %v, %v", v, ok)
	}
	if _, ok := ValueAt(pos, p.Child("x")); ok {
		t.Errorf("an absent value is found")
	}
}
