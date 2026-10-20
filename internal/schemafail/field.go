// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package schemafail

import (
	"slices"
	"strconv"

	"github.com/kottera/metais/internal/finding"
)

// maxRefs bounds how many $ref steps a title is looked for through. A
// package's references are bounded by its limits long before this.
const maxRefs = 64

// schemaObject is a schema object within an entry.
type schemaObject struct{ entry, pointer string }

// fieldAt is field's answer for the chain of failures ending at one: the
// first title and the first description along it, and whether any object
// on it is writeOnly.
type fieldAt struct {
	fd        finding.Field
	writeOnly bool
}

// field is ADR-006 D6 item 9's lookup for a finding of n at inst. It walks
// the evaluation path from n toward the root and takes the first title and,
// independently, the first description of a schema object whose instance
// location is inst. It reports, too, whether any of those objects is
// writeOnly (D9 item 5).
//
// The answer for a failure is its own objects' answer completed by its
// parent's, when the walk goes on to the parent, so each failure's is
// worked out once and kept: a deep tree of failures costs its depth once,
// not once for each of its findings.
func (m *mapper) field(path []*Failure, n *Failure, inst []string) (finding.Field, bool) {
	if !slices.Equal(n.InstancePointer, inst) {
		return finding.Field{}, false
	}
	if m.fields == nil {
		m.fields = map[*Failure]fieldAt{}
	}
	chain := append(slices.Clip(path), n)
	// Walk up to the first failure whose answer is known or that ends the
	// walk, then work the answers out back down.
	top := len(chain) - 1
	for top > 0 {
		if _, ok := m.fields[chain[top]]; ok {
			break
		}
		if m.own(chain[top]).descended || !slices.Equal(chain[top-1].InstancePointer, inst) {
			break
		}
		top--
	}
	for i := top; i < len(chain); i++ {
		f := chain[i]
		if _, ok := m.fields[f]; ok {
			continue
		}
		at := m.own(f).at
		if i > top {
			p := m.fields[chain[i-1]]
			if at.fd.Title == "" {
				at.fd.Title = p.fd.Title
			}
			if at.fd.Description == "" {
				at.fd.Description = p.fd.Description
			}
			at.writeOnly = at.writeOnly || p.writeOnly
		}
		m.fields[f] = at
	}
	at := m.fields[n]
	return at.fd, at.writeOnly
}

// ownField is what one failure's own schema objects give field: the
// answer over the objects at its instance location from the one its
// keyword is written in toward its entry's root, and whether the walk
// descended, so that it ends there.
type ownField struct {
	at        fieldAt
	descended bool
}

// own is f's ownField.
func (m *mapper) own(f *Failure) ownField {
	toks := tokens(f.SchemaPointer)
	if f.Keyword != KeywordFalse && len(toks) > 0 {
		toks = toks[:len(toks)-1] // the object the keyword is written in
	}
	up, descended := upward(toks)
	var o ownField
	o.descended = descended
	for _, p := range up {
		a := m.t.Schemas.Annotations(f.Entry, p)
		if o.at.fd.Title == "" {
			o.at.fd.Title = a.Title
		}
		if o.at.fd.Description == "" {
			o.at.fd.Description = a.Description
		}
		o.at.writeOnly = o.at.writeOnly || a.WriteOnly
	}
	return o
}

// upward returns the schema objects at the instance location of the one
// toks names, from it toward its entry's root: those an evaluation path
// reaches through applicators that keep the instance location. It stops at
// an applicator that descends, reporting so, and at a $defs member, where
// a reference's target begins, or the entry's root, where the evaluation
// path came from a node above.
func upward(toks []string) (ptrs []string, descended bool) {
	ptrs = []string{pointer(toks)}
	ss, rest := steps(toks)
	if rest != len(toks) {
		return ptrs, false
	}
	for k := len(ss) - 1; k >= 0; k-- {
		s := ss[k]
		if s.descends {
			return ptrs, true
		}
		if s.keyword == "$defs" {
			return ptrs, false
		}
		start := 0
		if k > 0 {
			start = ss[k-1].end
		}
		ptrs = append(ptrs, pointer(toks[:start]))
	}
	return ptrs, false
}

// memberField is D6 item 9's lookup for an absent member, name, that n
// requires: the properties/<name> subschema of the object holding n,
// following its $ref chain while it has neither a title nor a description.
func (m *mapper) memberField(n *Failure, name string) finding.Field {
	toks := tokens(n.SchemaPointer)
	holder := toks[:max(len(toks)-1, 0)]
	a := m.annotationsThroughRefs(schemaObject{n.Entry, pointer(append(slices.Clip(holder), "properties", name))},
		func(a Annotations) bool { return a.Title != "" || a.Description != "" })
	return finding.Field{Title: a.Title, Description: a.Description}
}

// branchTitle is branch b of choice's title: D6 item 9's lookup applied to
// the branch, which follows its $ref chain while it has no title.
func (m *mapper) branchTitle(choice *Failure, b int) string {
	o := schemaObject{choice.Entry, pointer(append(tokens(choice.SchemaPointer), strconv.Itoa(b)))}
	return m.annotationsThroughRefs(o, func(a Annotations) bool { return a.Title != "" }).Title
}

// annotationsThroughRefs returns the annotations of o, or of the first
// object along its $ref chain whose annotations satisfy done.
func (m *mapper) annotationsThroughRefs(o schemaObject, done func(Annotations) bool) Annotations {
	seen := []schemaObject{o}
	for range maxRefs {
		a := m.t.Schemas.Annotations(o.entry, o.pointer)
		if done(a) {
			return a
		}
		e, p, ok := m.t.Schemas.Ref(o.entry, o.pointer)
		if !ok {
			return a
		}
		o = schemaObject{e, p}
		if slices.Contains(seen, o) {
			return a
		}
		seen = append(seen, o)
	}
	return Annotations{}
}
