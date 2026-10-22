// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package jsonx

import (
	"slices"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"

	"metais.org/metais/internal/finding"
)

// bom is the UTF-8 byte order mark ADR-003 D8 item 1 removes.
const bom = "\xEF\xBB\xBF"

// lineIndex converts a byte offset in one text to ADR-006 D5 item 3's line
// and column: lines end at LF, CRLF or a lone CR, each one line break;
// columns count code points from the start of the line, an invalid byte
// counting as one; a leading byte order mark is not counted.
//
// A column is counted from the nearest checkpoint, not from the start of
// its line, so a finding on a long line costs a bounded count, never the
// line's length.
type lineIndex struct {
	src    []byte
	starts []int // the offset of each line's first byte, ascending
	// runes holds, for valid UTF-8, the code points before each multiple
	// of checkpoint bytes; nil for text that is not, whose columns are
	// counted from the line's start.
	runes []int
}

// checkpoint is how many bytes apart lineIndex counts code points.
const checkpoint = 256

func newLineIndex(src []byte) *lineIndex {
	start := 0
	if len(src) >= len(bom) && string(src[:len(bom)]) == bom {
		start = len(bom)
	}
	starts := []int{start}
	for i := start; i < len(src); i++ {
		switch src[i] {
		case '\n':
			starts = append(starts, i+1)
		case '\r':
			if i+1 < len(src) && src[i+1] == '\n' {
				i++
			}
			starts = append(starts, i+1)
		}
	}
	x := &lineIndex{src: src, starts: starts}
	if utf8.Valid(src) {
		x.runes = make([]int, len(src)/checkpoint+1)
		n := 0
		for i, c := range src {
			if utf8.RuneStart(c) {
				n++
			}
			if (i+1)%checkpoint == 0 {
				x.runes[(i+1)/checkpoint] = n
			}
		}
	}
	return x
}

// runesTo returns how many code points of valid UTF-8 come before off.
// Every byte that starts a code point is one.
func (x *lineIndex) runesTo(off int) int {
	n := x.runes[off/checkpoint]
	for _, c := range x.src[off/checkpoint*checkpoint : off] {
		if utf8.RuneStart(c) {
			n++
		}
	}
	return n
}

// pos returns the line and column of the byte at off, which is in [0,
// len(src)]: the end of the text has a position too, for a finding about
// what is missing there. An offset inside the byte order mark is the
// first column of the first line.
func (x *lineIndex) pos(off int) finding.Pos {
	off = max(off, x.starts[0])
	line := sort.Search(len(x.starts), func(i int) bool { return x.starts[i] > off }) - 1
	if x.runes == nil {
		return finding.Pos{Line: line + 1, Column: utf8.RuneCount(x.src[x.starts[line]:off]) + 1}
	}
	return finding.Pos{Line: line + 1, Column: x.runesTo(off) - x.runesTo(x.starts[line]) + 1}
}

// node is where one value of a document is: the offset of its first byte,
// and, for an array or an object, where each element or member is. It
// mirrors the value tree, so the table costs one node per value however
// long the pointers to them would be.
type node struct {
	off     int
	elems   []node
	members []member // by name, once the object is read
}

// member is one object member's position: its name's opening quote and
// its value. A duplicate name keeps its first occurrence's positions.
type member struct {
	name   string
	keyOff int
	node
}

// sortMembers orders an object's members by name, for lookup.
func (n *node) sortMembers() {
	slices.SortFunc(n.members, func(a, b member) int { return strings.Compare(a.name, b.name) })
}

func (n *node) member(name string) (*member, bool) {
	i, ok := slices.BinarySearchFunc(n.members, name, func(m member, name string) int { return strings.Compare(m.name, name) })
	if !ok {
		return nil, false
	}
	return &n.members[i], true
}

// table is a document's position table (ADR-006 D13 item 2). A nil table,
// or one whose root was never read, holds nothing.
type table struct {
	lines *lineIndex
	root  *node
	paths *finding.Resolver[place] // the places read by Path
}

// place is a value of the document a Path names: its node, and the
// member it is when the path names an object member.
type place struct {
	n *node
	m *member
}

var _ finding.PathPositions = (*table)(nil)

func newTable(lines *lineIndex, root *node) *table {
	t := &table{lines: lines, root: root}
	t.paths = finding.NewResolver(place{n: root}, func(at place, tok string) (place, bool) {
		n, m, ok := at.n.child(tok)
		return place{n, m}, ok
	})
	return t
}

// find returns the node at ptr, and the member it is when ptr names an
// object member.
func (t *table) find(ptr string) (*node, *member, bool) {
	if t == nil || t.root == nil {
		return nil, nil, false
	}
	if ptr == "" {
		return t.root, nil, true
	}
	if ptr[0] != '/' {
		return nil, nil, false
	}
	n := t.root
	var m *member
	for _, tok := range strings.Split(ptr[1:], "/") {
		var ok bool
		if n, m, ok = n.child(tokenUnescaper.Replace(tok)); !ok {
			return nil, nil, false
		}
	}
	return n, m, true
}

// child returns the node of the member or element tok, a decoded token,
// of n, and the member it is when n is an object.
func (n *node) child(tok string) (*node, *member, bool) {
	switch {
	case n.members != nil:
		m, ok := n.member(tok)
		if !ok {
			return nil, nil, false
		}
		return &m.node, m, true
	case n.elems != nil:
		i, ok := arrayIndex(tok)
		if !ok || i >= len(n.elems) {
			return nil, nil, false
		}
		return &n.elems[i], nil, true
	}
	return nil, nil, false
}

// findPath returns the node at p, as find does: its cost is p's depth,
// and a token of p is read only the first time a path through it is.
func (t *table) findPath(p *finding.Path) (*node, *member, bool) {
	if t == nil || t.root == nil {
		return nil, nil, false
	}
	at, depth := t.paths.Resolve(p)
	if depth != p.Depth() {
		return nil, nil, false
	}
	return at.n, at.m, true
}

var tokenUnescaper = strings.NewReplacer("~1", "/", "~0", "~")

// arrayIndex reads an RFC 6901 array index: 0, or digits without a leading
// zero.
func arrayIndex(tok string) (int, bool) {
	if tok == "" || (len(tok) > 1 && tok[0] == '0') {
		return 0, false
	}
	for i := 0; i < len(tok); i++ {
		if tok[i] < '0' || tok[i] > '9' {
			return 0, false
		}
	}
	i, err := strconv.Atoi(tok)
	return i, err == nil
}

func (t *table) valueOffset(ptr string) (int, bool) {
	n, _, ok := t.find(ptr)
	if !ok {
		return 0, false
	}
	return n.off, true
}

func (t *table) keyOffset(ptr string) (int, bool) {
	_, m, ok := t.find(ptr)
	if !ok || m == nil {
		return 0, false
	}
	return m.keyOff, true
}

// Value is the position of the first character of the value at ptr.
func (t *table) Value(ptr string) (finding.Pos, bool) {
	off, ok := t.valueOffset(ptr)
	if !ok {
		return finding.Pos{}, false
	}
	return t.lines.pos(off), true
}

// Key is the position of the opening quote of the member name, for the
// object member at ptr.
func (t *table) Key(ptr string) (finding.Pos, bool) {
	off, ok := t.keyOffset(ptr)
	if !ok {
		return finding.Pos{}, false
	}
	return t.lines.pos(off), true
}

// ValueAt is Value of the pointer p names.
func (t *table) ValueAt(p *finding.Path) (finding.Pos, bool) {
	n, _, ok := t.findPath(p)
	if !ok {
		return finding.Pos{}, false
	}
	return t.lines.pos(n.off), true
}

// KeyAt is Key of the pointer p names.
func (t *table) KeyAt(p *finding.Path) (finding.Pos, bool) {
	_, m, ok := t.findPath(p)
	if !ok || m == nil {
		return finding.Pos{}, false
	}
	return t.lines.pos(m.keyOff), true
}
