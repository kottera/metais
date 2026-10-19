// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package unitext

import (
	"strings"
	"unicode/utf8"
)

// CollisionKind is which of ADR-005 D12 items 6 and 7 two paths break.
type CollisionKind uint8

const (
	// Duplicate: two paths are the same bytes.
	Duplicate CollisionKind = iota + 1
	// CaseCollision: two distinct files, or two distinct directories, are
	// canonical caseless matches (item 6): "Readme.md" beside "README.md",
	// or "Docs/a" beside "docs/b".
	CaseCollision
	// FileDirectoryConflict: a file's whole path is, or caselessly matches,
	// a directory prefix of another path (item 7): "docs" beside
	// "docs/a.md", or "Docs" beside "docs/a.md".
	FileDirectoryConflict
)

// Collision is one member of a tree's collision set that collides with an
// earlier one.
type Collision struct {
	Kind CollisionKind
	// Path is the path that brought Member into the set, and Member is
	// Path itself, when it collides as a file, or the directory prefix of
	// Path that collides.
	Path, Member string
	// Other is the earlier path that brought OtherMember into the set, and
	// OtherMember the member Member collides with.
	Other, OtherMember string
}

// member is one element of the collision set.
type member struct {
	text string
	file bool
	path string
}

// Collisions applies ADR-005 D12 items 6 and 7 to one tree, given as its
// paths in any order. It takes the set of every path and every directory
// prefix of every path ("a/b/c" contributes the directories "a" and "a/b"
// and the file "a/b/c"), and reports each member that collides with an
// earlier one, in the order the paths are given: a member that is the same
// bytes as an earlier directory is that directory, not a collision. Each
// collision is reported against the first member of the set it matches.
//
// A path that is not valid UTF-8 has no caseless key and takes no part;
// CheckPath reports it.
func Collisions(paths []string) []Collision {
	var out []Collision
	byKey := map[string][]member{}
	for _, p := range paths {
		if !utf8.ValidString(p) {
			continue
		}
		for i := 0; i <= len(p); i++ {
			if i < len(p) && p[i] != '/' {
				continue
			}
			m := member{text: p[:i], file: i == len(p), path: p}
			k := CaselessKey(m.text)
			seen := byKey[k]
			if !m.file && containsDir(seen, m.text) {
				continue
			}
			if other, ok := sameParent(seen, m.text); ok {
				out = append(out, Collision{
					Kind:   collisionKind(other, m),
					Path:   p,
					Member: m.text, Other: other.path, OtherMember: other.text,
				})
			}
			byKey[k] = append(seen, m)
		}
	}
	return out
}

// sameParent returns the first member of seen whose parent directory is
// the same bytes as text's. A member that matches text caselessly has a
// parent that matches text's caselessly, since folding and normalization
// never move a "/"; when the two parents are not the same bytes, they are
// the collision, already reported, and this one is its consequence.
func sameParent(seen []member, text string) (member, bool) {
	parent := text[:max(strings.LastIndexByte(text, '/'), 0)]
	for _, s := range seen {
		if s.text[:max(strings.LastIndexByte(s.text, '/'), 0)] == parent {
			return s, true
		}
	}
	return member{}, false
}

// containsDir reports whether seen holds the directory text.
func containsDir(seen []member, text string) bool {
	for _, s := range seen {
		if !s.file && s.text == text {
			return true
		}
	}
	return false
}

// collisionKind says how m collides with first, an earlier member with the
// same caseless key and the same parent. They are not the same directory.
func collisionKind(first, m member) CollisionKind {
	switch {
	case first.file != m.file:
		return FileDirectoryConflict
	case first.text == m.text:
		return Duplicate
	default:
		return CaseCollision
	}
}
