// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package finding

import (
	"crypto/sha256"
	"sync"
)

// PathKey identifies a Path's tokens: two paths have the same key exactly
// when they have the same tokens, however each was built. It is
// comparable, so a table remembers a path by it.
type PathKey [sha256.Size]byte

// Key is p's PathKey. It costs nothing for p's length.
func (p *Path) Key() PathKey { return p.digest() }

// Parent returns p without its last token: nil for the root and for a
// path of one token.
func (p *Path) Parent() *Path {
	if p == nil {
		return nil
	}
	return p.parent
}

// Last returns p's last token, decoded: "" for the root.
func (p *Path) Last() string {
	if p == nil {
		return ""
	}
	return p.token.text
}

// PathCache makes Paths from the decoded tokens a validator gives, such as
// failures' instance locations, in the order it gives them. It remembers
// the path it made last, and a path that begins with the same tokens
// below the same prefix is made from it: only the tokens after the shared
// ones are read. A validator gives the paths of the values under one
// member name together, each token the same string, so many paths under
// one long member name cost each its depth, never the name's length. The
// zero PathCache is ready to use; it is not safe for concurrent use.
type PathCache struct {
	under  *Path
	tokens []string
	paths  []*Path // paths[i] is the path of tokens[:i+1] below under
}

// Path returns the path of tokens below under: under itself when tokens
// is empty.
func (c *PathCache) Path(under *Path, tokens []string) *Path {
	shared := 0
	if under == c.under {
		for shared < len(tokens) && shared < len(c.tokens) && tokens[shared] == c.tokens[shared] {
			shared++
		}
	}
	c.under, c.tokens, c.paths = under, append(c.tokens[:shared], tokens[shared:]...), c.paths[:shared]
	p := under
	if shared > 0 {
		p = c.paths[shared-1]
	}
	for _, t := range tokens[shared:] {
		p = p.Child(t)
		c.paths = append(c.paths, p)
	}
	return p
}

// PathPositions is a Positions that reads a value by its Path at the cost
// of the path's depth, never its length. A parser's position table
// implements it; ValueAt and KeyAt use it where it is.
type PathPositions interface {
	Positions
	// ValueAt is Value of the pointer p names.
	ValueAt(p *Path) (Pos, bool)
	// KeyAt is Key of the pointer p names.
	KeyAt(p *Path) (Pos, bool)
}

// ValueAt is pos.Value of the pointer p names, read by its Path when pos
// is a PathPositions.
func ValueAt(pos Positions, p *Path) (Pos, bool) {
	if pp, ok := pos.(PathPositions); ok {
		return pp.ValueAt(p)
	}
	return pos.Value(p.String())
}

// KeyAt is pos.Key of the pointer p names, read by its Path when pos is a
// PathPositions.
func KeyAt(pos Positions, p *Path) (Pos, bool) {
	if pp, ok := pos.(PathPositions); ok {
		return pp.KeyAt(p)
	}
	return pos.Key(p.String())
}

// Resolver finds what a Path names in a tree a token at a time, and
// remembers every path it has resolved by its PathKey. A path whose parent
// was resolved before costs one step, for its own token, so resolving many
// paths under one long member name reads the name once. It is safe for
// concurrent use.
type Resolver[N any] struct {
	root N
	// step returns what token names within n, and false when it names
	// nothing.
	step func(n N, token string) (N, bool)

	mu   sync.Mutex
	seen map[PathKey]resolved[N]
}

// resolved is what a path resolved to: the node of its longest prefix
// that names one, and how many tokens that prefix has.
type resolved[N any] struct {
	n     N
	depth int
}

// NewResolver returns a Resolver of the tree at root, which step walks.
func NewResolver[N any](root N, step func(n N, token string) (N, bool)) *Resolver[N] {
	return &Resolver[N]{root: root, step: step}
}

// Resolve returns what the longest prefix of p that names something
// names, and how many of p's tokens that prefix has: p's depth when p
// itself names something.
func (r *Resolver[N]) Resolve(p *Path) (N, int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	// The ancestors of p not yet resolved, p first.
	var todo []*Path
	at := resolved[N]{n: r.root}
	for q := p; q != nil; q = q.parent {
		if s, ok := r.seen[q.id]; ok {
			at = s
			break
		}
		todo = append(todo, q)
	}
	if r.seen == nil {
		r.seen = map[PathKey]resolved[N]{}
	}
	for i := len(todo) - 1; i >= 0; i-- {
		q := todo[i]
		if at.depth == q.depth-1 {
			if n, ok := r.step(at.n, q.token.text); ok {
				at = resolved[N]{n: n, depth: q.depth}
			}
		}
		r.seen[q.id] = at
	}
	return at.n, at.depth
}
