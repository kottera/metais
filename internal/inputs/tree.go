// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package inputs

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/kottera/metais/internal/finding"
	"github.com/kottera/metais/internal/liquid/value"
)

// node is one value of the inputs object while it is assembled (ADR-003
// D18) and given its defaults (D19): a value the steps can change in
// place, which records, where a source put it, which source that was
// (ADR-006 D4 item 5) and how to find a position in it.
type node struct {
	scalar value.Value      // null, boolean, number or string; nil for an object or an array
	obj    map[string]*node // an object's members, when isObj
	arr    []*node          // an array's elements, when isArr
	isObj  bool
	isArr  bool
	// at is where the value came from, set where a source placed it: a
	// file's root, an --inputs-json member, a path set's value and every
	// object it created, a default. A value below it is located through
	// it.
	at locator
	// index is the element's index in its collection's order before any
	// item was left out (ADR-006 D7 item 4); elsewhere its position.
	index int
	// item is the data-directory file an element of a collection came
	// from, and nil for an element supplied any other way.
	item *item
}

// locator finds the position of a value below the node it is recorded
// at: rel is the pointer's tokens from that node, and key asks for the
// member name's position rather than the value's (ADR-006 D5 item 6). The
// location it returns has no pointer.
type locator interface {
	locate(rel []string, key bool) finding.Location
}

// fromValue is v as a node, located by at.
func fromValue(v value.Value, at locator) *node {
	n := build(v)
	n.at = at
	return n
}

func build(v value.Value) *node {
	switch x := v.(type) {
	case value.Object:
		n := &node{isObj: true, obj: make(map[string]*node, x.Len())}
		for name, m := range x.All() {
			n.obj[name] = build(m)
		}
		return n
	case value.Array:
		n := &node{isArr: true, arr: make([]*node, 0, x.Len())}
		for i, e := range x.All() {
			c := build(e)
			c.index = i
			n.arr = append(n.arr, c)
		}
		return n
	}
	return &node{scalar: v}
}

func newObject(at locator) *node { return &node{isObj: true, obj: map[string]*node{}, at: at} }

// value is the node as a value of the data model.
func (n *node) value() value.Value {
	switch {
	case n.isObj:
		members := make([]value.Member, 0, len(n.obj))
		for name, m := range n.obj {
			members = append(members, value.Member{Name: name, Value: m.value()})
		}
		o, err := value.NewObject(members...)
		if err != nil {
			panic(fmt.Sprintf("inputs: an assembled object is refused: %v", err))
		}
		return o
	case n.isArr:
		elems := make([]value.Value, len(n.arr))
		for i, e := range n.arr {
			elems[i] = e.value()
		}
		a, err := value.NewArray(elems...)
		if err != nil {
			panic(fmt.Sprintf("inputs: an assembled array is refused: %v", err))
		}
		return a
	}
	return n.scalar
}

// element returns the element of the array n whose index is i: its
// position, or, in an array some of whose items were left out, the
// element that had index i before.
func (n *node) element(i int) (*node, bool) {
	if i >= 0 && i < len(n.arr) && n.arr[i].index == i {
		return n.arr[i], true
	}
	j, ok := slices.BinarySearchFunc(n.arr, i, func(e *node, i int) int { return e.index - i })
	if !ok {
		return nil, false
	}
	return n.arr[j], true
}

// locate returns where the value at ptr, a pointer below n, came from: the
// deepest node at or above it that records a source, asked for the rest
// of the pointer. A pointer that names an absent value (ADR-006 D5 item 2)
// is located through the nearest node above it.
func (n *node) locate(ptr string, key bool) finding.Location {
	toks := pointerTokens(ptr)
	at, rel := n.at, toks
	cur := n
	for i, tok := range toks {
		var next *node
		ok := false
		switch {
		case cur.isObj:
			next, ok = cur.obj[tok]
		case cur.isArr:
			if idx, isIdx := arrayIndex(tok); isIdx {
				next, ok = cur.element(idx)
			}
		}
		if !ok {
			break
		}
		cur = next
		if cur.at != nil {
			at, rel = cur.at, toks[i+1:]
		}
	}
	if at == nil {
		panic("inputs: a value with no source")
	}
	return at.locate(rel, key)
}

// depth is how deeply arrays and objects nest in n: zero for a scalar.
func (n *node) depth() int {
	d := 0
	for _, c := range n.children() {
		d = max(d, c.depth())
	}
	if n.isObj || n.isArr {
		d++
	}
	return d
}

func (n *node) children() []*node {
	if n.isArr {
		return n.arr
	}
	out := make([]*node, 0, len(n.obj))
	for _, c := range n.obj {
		out = append(out, c)
	}
	return out
}

// pointerTokens splits an RFC 6901 pointer into its decoded tokens.
func pointerTokens(ptr string) []string {
	if ptr == "" {
		return nil
	}
	parts := strings.Split(ptr[1:], "/")
	for i, p := range parts {
		parts[i] = strings.ReplaceAll(strings.ReplaceAll(p, "~1", "/"), "~0", "~")
	}
	return parts
}

// arrayIndex reads an RFC 6901 array index: 0, or digits with no leading
// zero.
func arrayIndex(tok string) (int, bool) {
	if tok == "" || len(tok) > 1 && tok[0] == '0' || len(tok) > 9 {
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

// docLocator locates a value in a parsed JSON or YAML document by its
// position table: the document at its origin, the value's pointer in it
// being prefix and then the pointer below the node.
type docLocator struct {
	at     finding.Location
	pos    finding.Positions
	prefix string
}

func (d docLocator) locate(rel []string, key bool) finding.Location {
	ptr := d.prefix + finding.Pointer(rel...)
	get := d.pos.Value
	if key && len(rel) > 0 {
		get = d.pos.Key
	}
	if p, ok := get(ptr); ok {
		return d.at.At(p)
	}
	return d.at
}

// plainLocator locates every value below it at one place with no line: a
// text file, which is one value (ADR-006 D5 item 5), a path set and a
// default.
type plainLocator struct{ at finding.Location }

func (p plainLocator) locate([]string, bool) finding.Location { return p.at }

// csvLocator locates a value in a CSV file's rows: a row at its first
// field, a member at its field.
type csvLocator struct {
	at    finding.Location
	rows  map[int]finding.Pos            // by the row's index
	cells map[int]map[string]finding.Pos // by the row's index, then the column
}

func (c csvLocator) locate(rel []string, _ bool) finding.Location {
	if len(rel) == 0 {
		return c.at
	}
	row, ok := arrayIndex(rel[0])
	if !ok {
		return c.at
	}
	if len(rel) >= 2 {
		if p, ok := c.cells[row][rel[1]]; ok {
			return c.at.At(p)
		}
	}
	if p, ok := c.rows[row]; ok {
		return c.at.At(p)
	}
	return c.at
}

// markdownLocator locates a value in a Markdown file: the body at its
// first line (ADR-006 D5 item 6), a frontmatter field by the
// frontmatter's position table, whose lines count from the file's first.
type markdownLocator struct {
	at       finding.Location
	front    finding.Positions // nil without frontmatter
	bodyLine int
}

func (m markdownLocator) locate(rel []string, key bool) finding.Location {
	switch {
	case len(rel) == 0:
		return m.at.At(finding.Pos{Line: 1, Column: 1})
	case rel[0] == bodyField && len(rel) == 1:
		if key {
			return m.at
		}
		return m.at.At(finding.Pos{Line: m.bodyLine, Column: 1})
	case m.front != nil:
		return docLocator{at: m.at, pos: m.front}.locate(rel, key)
	}
	return m.at
}
