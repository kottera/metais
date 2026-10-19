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

// field is ADR-006 D6 item 9's lookup for a finding of n at inst. It walks
// the evaluation path from n toward the root and takes the first title and,
// independently, the first description of a schema object whose instance
// location is inst. It reports, too, whether any of those objects is
// writeOnly (D9 item 5).
func (m *mapper) field(path []*Failure, n *Failure, inst []string) (finding.Field, bool) {
	var objs []schemaObject
	chain := append(slices.Clip(path), n)
	for i := len(chain) - 1; i >= 0; i-- {
		f := chain[i]
		if !slices.Equal(f.InstancePointer, inst) {
			break
		}
		toks := tokens(f.SchemaPointer)
		if f.Keyword != KeywordFalse && len(toks) > 0 {
			toks = toks[:len(toks)-1] // the object the keyword is written in
		}
		up, descended := upward(toks)
		for _, p := range up {
			if o := (schemaObject{f.Entry, p}); !slices.Contains(objs, o) {
				objs = append(objs, o)
			}
		}
		if descended {
			break
		}
	}
	var fd finding.Field
	writeOnly := false
	for _, o := range objs {
		a := m.t.Schemas.Annotations(o.entry, o.pointer)
		if fd.Title == "" {
			fd.Title = a.Title
		}
		if fd.Description == "" {
			fd.Description = a.Description
		}
		writeOnly = writeOnly || a.WriteOnly
	}
	return fd, writeOnly
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
