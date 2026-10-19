// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package jsonx

import (
	"slices"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/kottera/metais/internal/finding"
)

// bom is the UTF-8 byte order mark ADR-003 D8 item 1 removes.
const bom = "\xEF\xBB\xBF"

// lineIndex converts a byte offset in one text to ADR-006 D5 item 3's line
// and column: lines end at LF, CRLF or a lone CR, each one line break;
// columns count code points from the start of the line, an invalid byte
// counting as one; a leading byte order mark is not counted.
type lineIndex struct {
	src    []byte
	starts []int // the offset of each line's first byte, ascending
}

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
	return &lineIndex{src: src, starts: starts}
}

// pos returns the line and column of the byte at off, which is in [0,
// len(src)]: the end of the text has a position too, for a finding about
// what is missing there. An offset inside the byte order mark is the
// first column of the first line.
func (x *lineIndex) pos(off int) finding.Pos {
	off = max(off, x.starts[0])
	line := sort.Search(len(x.starts), func(i int) bool { return x.starts[i] > off }) - 1
	return finding.Pos{Line: line + 1, Column: utf8.RuneCount(x.src[x.starts[line]:off]) + 1}
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
		tok = tokenUnescaper.Replace(tok)
		switch {
		case n.members != nil:
			var ok bool
			if m, ok = n.member(tok); !ok {
				return nil, nil, false
			}
			n = &m.node
		case n.elems != nil:
			i, ok := arrayIndex(tok)
			if !ok || i >= len(n.elems) {
				return nil, nil, false
			}
			n, m = &n.elems[i], nil
		default:
			return nil, nil, false
		}
	}
	return n, m, true
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
