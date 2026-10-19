// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package finding

import (
	"cmp"
	"crypto/sha256"
	"strings"
)

// Pos is a position in a file or an argument's text (ADR-006 D5 item 3):
// a 1-based line, and a 1-based column counted in Unicode code points.
// The zero Pos is unknown.
type Pos struct{ Line, Column int }

// Known reports whether p names a position.
func (p Pos) Known() bool { return p.Line > 0 }

// Positions is what a parser keeps of one parsed file or argument, so a
// finding about a value can say where it is (ADR-006 D13 item 2). ptr is an
// RFC 6901 JSON Pointer into the parsed value.
type Positions interface {
	// Value is the position of the first character of the value at ptr.
	Value(ptr string) (Pos, bool)
	// Key is the position of the first character of the member name, for
	// an object member.
	Key(ptr string) (Pos, bool)
}

var tokenEscaper = strings.NewReplacer("~", "~0", "/", "~1")

// Pointer returns the RFC 6901 JSON Pointer, in its JSON-string form, made
// of tokens: "" for none, and each token escaped and prefixed with "/".
func Pointer(tokens ...string) string {
	var b strings.Builder
	for _, t := range tokens {
		b.WriteByte('/')
		tokenEscaper.WriteString(&b, t)
	}
	return b.String()
}

var tokenUnescaper = strings.NewReplacer("~1", "/", "~0", "~")

// pointerTokens splits ptr into its decoded reference tokens. RFC 6901
// decodes ~1 before ~0, which a single left-to-right pass of
// tokenUnescaper does, since it never rescans what it wrote.
func pointerTokens(ptr string) []string {
	if ptr == "" {
		return nil
	}
	parts := strings.Split(strings.TrimPrefix(ptr, "/"), "/")
	for i, p := range parts {
		parts[i] = tokenUnescaper.Replace(p)
	}
	return parts
}

// Path is an RFC 6901 JSON Pointer held as its decoded tokens, each node
// sharing its parent. A reader that reports many values under one long
// pointer builds the parent once and each child from it, so building,
// de-duplicating and ordering their findings costs each finding its own
// token, never the whole pointer. The nil *Path is the root, "".
type Path struct {
	parent *Path
	token  string
	depth  int
	id     [sha256.Size]byte // the SHA-256 of the parent's id and the token's digest
}

// Token is one reference token of a Path with its digest, which is read
// from the token once. A reader that names one long token under many
// parents, such as a long column name in every record, makes it once and
// gives it to ChildToken, so each child costs nothing for its length.
type Token struct {
	text   string
	digest [sha256.Size]byte
}

// NewToken returns token as a Token. An invalid UTF-8 byte in token is
// replaced with U+FFFD (ADR-006 D9 item 9).
func NewToken(token string) Token {
	token = validUTF8(token)
	return Token{text: token, digest: sha256.Sum256([]byte(token))}
}

// Child returns the path to the member or element token of the value at
// p, as ChildToken(NewToken(token)) does.
func (p *Path) Child(token string) *Path { return p.ChildToken(NewToken(token)) }

// ChildToken returns the path to the member or element t of the value at
// p.
func (p *Path) ChildToken(t Token) *Path {
	c := &Path{parent: p, token: t.text, depth: p.Depth() + 1}
	h := sha256.New()
	id := p.digest()
	h.Write(id[:])
	h.Write(t.digest[:])
	h.Sum(c.id[:0])
	return c
}

// Depth is how many tokens p has.
func (p *Path) Depth() int {
	if p == nil {
		return 0
	}
	return p.depth
}

// String returns p in its JSON-string form, as Pointer does.
func (p *Path) String() string {
	tokens := make([]string, p.Depth())
	for q := p; q != nil; q = q.parent {
		tokens[q.depth-1] = q.token
	}
	return Pointer(tokens...)
}

// digest identifies p's token sequence: two paths have the same digest
// exactly when they have the same tokens. The root's is all zeros.
func (p *Path) digest() [sha256.Size]byte {
	if p == nil {
		return [sha256.Size]byte{}
	}
	return p.id
}

// parsePath returns the path ptr, an RFC 6901 pointer, names.
func parsePath(ptr string) *Path {
	var p *Path
	for _, t := range pointerTokens(ptr) {
		p = p.Child(t)
	}
	return p
}

// comparePaths compares two paths token by token (ADR-006 D8 item 1.6):
// at the first token from the root at which they differ, else the shorter
// first. Shared or equal prefixes are passed over by their digests, so
// only the tokens that differ are read.
func comparePaths(a, b *Path) int {
	if a == b || a.digest() == b.digest() {
		return 0
	}
	ca, cb := a.chain(), b.chain()
	for i := 0; i < len(ca) && i < len(cb); i++ {
		if ca[i].id != cb[i].id {
			return compareTokens(ca[i].token, cb[i].token)
		}
	}
	return cmp.Compare(len(ca), len(cb))
}

// chain returns p's nodes, the root's child first.
func (p *Path) chain() []*Path {
	c := make([]*Path, p.Depth())
	for q := p; q != nil; q = q.parent {
		c[q.depth-1] = q
	}
	return c
}
