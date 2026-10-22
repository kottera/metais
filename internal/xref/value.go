// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package xref

import (
	"slices"
	"strconv"
	"strings"

	"metais.org/metais/internal/schema"
)

// maxAlternatives is the most alternatives a set holds; a larger one
// becomes unknown (ADR-007 D3 item 2).
const maxAlternatives = 64

// kind is which abstract value an alternative is (ADR-007 D3 item 2).
type kind uint8

const (
	// kSchema is a node of an input's schema: the input and the node's
	// pointer in the combined schema.
	kSchema kind = iota + 1
	// kOpen is a value below which anything is declared.
	kOpen
	// kUnknown is a value the analysis cannot tell: nothing below it is
	// checked, and nothing below it is an error.
	kUnknown
	// kScalar is a string, number, boolean or nil.
	kScalar
	// kRecord is a fixed set of named fields: inputs, package, fan_out,
	// output, sources and its members, forloop and tablerowloop.
	kRecord
	// kArray is an array whose elements are an element set: what map
	// projects, what concat joins and what a range holds.
	kArray
)

// scalar is the kind of a scalar.
type scalar uint8

const (
	sString scalar = iota + 1
	sNumber
	sBool
	sNil
)

// types returns the JSON types of a scalar of kind s.
func (s scalar) types() schema.TypeSet {
	switch s {
	case sString:
		return schema.TypeString
	case sNumber:
		return schema.TypeNumber
	case sBool:
		return schema.TypeBoolean
	}
	return schema.TypeNull
}

// alt is one alternative of an abstract value.
type alt struct {
	k     kind
	sc    scalar  // kScalar
	input string  // kSchema: the input whose schema the node is in
	ptr   string  // kSchema: the node's pointer in the combined schema
	rec   *record // kRecord
	elem  set     // kArray
	key   string  // the alternative's identity, which sets order and compare by
}

// record is a record's fields. Records are interned by their content, so
// one record is one pointer and its key is fixed.
type record struct {
	key    string
	fields map[string]set
	names  []string // the field names, sorted
	inputs bool     // the inputs namespace, whose fields mark use
}

// set is an abstract value: its alternatives, sorted by key, none
// repeated. The empty set is a value nothing has yet been bound to, which
// the fixed point of locals starts from: nothing below it is checked.
type set []alt

func schemaAlt(input, ptr string) alt {
	return alt{k: kSchema, input: input, ptr: ptr, key: "s" + strconv.Quote(input) + ptr}
}

var (
	openAlt    = alt{k: kOpen, key: "o"}
	unknownAlt = alt{k: kUnknown, key: "u"}
	open       = set{openAlt}
	unknown    = set{unknownAlt}
	str        = scalarSet(sString)
	num        = scalarSet(sNumber)
	boolean    = scalarSet(sBool)
	null       = scalarSet(sNil)
)

func scalarAlt(s scalar) alt {
	return alt{k: kScalar, sc: s, key: "k" + strconv.Itoa(int(s))}
}

func scalarSet(s scalar) set { return set{scalarAlt(s)} }

func recordAlt(r *record) alt { return alt{k: kRecord, rec: r, key: r.key} }

func arrayAlt(elem set) alt {
	return alt{k: kArray, elem: elem, key: "a(" + elem.key() + ")"}
}

// array returns the set holding an array of elem.
func array(elem set) set { return set{arrayAlt(elem)} }

// union returns the set of every alternative of ss, sorted and without
// repetition, unknown when there are more than maxAlternatives.
func union(ss ...set) set {
	n := 0
	for _, s := range ss {
		n += len(s)
	}
	out := make(set, 0, n)
	for _, s := range ss {
		out = append(out, s...)
	}
	slices.SortFunc(out, func(a, b alt) int { return strings.Compare(a.key, b.key) })
	out = slices.CompactFunc(out, func(a, b alt) bool { return a.key == b.key })
	if len(out) > maxAlternatives {
		return unknown
	}
	return out
}

// of returns the set of the alternatives as.
func of(as ...alt) set { return union(set(as)) }

// key is the set's identity: equal sets have equal keys.
func (s set) key() string {
	var b strings.Builder
	b.WriteByte('{')
	for i, a := range s {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(a.key)
	}
	b.WriteByte('}')
	return b.String()
}

// recordKey is the key of a record with these fields.
func recordKey(fields map[string]set, names []string, inputs bool) string {
	var b strings.Builder
	b.WriteString("r")
	if inputs {
		b.WriteString("!")
	}
	b.WriteByte('{')
	for i, n := range names {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(strconv.Quote(n))
		b.WriteByte(':')
		b.WriteString(fields[n].key())
	}
	b.WriteByte('}')
	return b.String()
}
