// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

// Package schemafail is the library-neutral failure tree a JSON Schema
// validation produces (ADR-006 D6 item 2, D13 item 3). An adapter builds it
// from the validator's own error tree, and ADR-006 D6's rules map it to
// findings, so the validator can change without changing a finding and the
// mapping is tested on hand-built trees.
package schemafail

import (
	"fmt"
	"maps"
	"slices"
	"strings"
)

// Failure is one node of the tree: a keyword that failed, or an applicator
// whose subschemas did.
type Failure struct {
	// Keyword is the keyword's name, such as minLength, required or $ref.
	Keyword string
	// KeywordLocation is the evaluation path from the root schema through
	// every $ref, in JSON Schema's keywordLocation form.
	KeywordLocation string
	// Entry is the package entry the keyword is written in, empty for a
	// keyword of a built-in 2020-12 meta-schema that a schema refers to.
	Entry string
	// SchemaPointer is the keyword's RFC 6901 pointer within Entry.
	SchemaPointer string
	// InstancePointer is the failing value's location in the instance,
	// as decoded reference tokens; empty at the root.
	InstancePointer []string
	// Data is the keyword's structured fields: for required the members
	// missing, for type the types wanted and the one found.
	Data Data
	// Children are the failures beneath an applicator, in the order the
	// adapter gives them; the mapping sorts what it reports.
	Children []Failure
}

// Data holds a keyword's structured fields by name. A value is a string, a
// bool, an int, a Number, or a slice of one of those, so the tree carries
// nothing of the library that produced it.
type Data map[string]any

// Number is a JSON number as its literal text, which keeps its exact
// value (ADR-010 D5 item 4).
type Number string

var tokenEscaper = strings.NewReplacer("~", "~0", "/", "~1")

// Instance is the failure's instance location as an RFC 6901 JSON
// Pointer.
func (f Failure) Instance() string {
	var b strings.Builder
	for _, t := range f.InstancePointer {
		b.WriteByte('/')
		tokenEscaper.WriteString(&b, t)
	}
	return b.String()
}

// Walk calls fn for f and each failure beneath it, depth first, parents
// before children and children in order. path holds the ancestors of the
// failure fn is given, outermost first. When fn returns false, the
// failure's children are skipped.
func (f *Failure) Walk(fn func(path []*Failure, f *Failure) bool) {
	f.walk(nil, fn)
}

func (f *Failure) walk(path []*Failure, fn func([]*Failure, *Failure) bool) {
	if !fn(path, f) {
		return
	}
	path = append(path, f)
	for i := range f.Children {
		f.Children[i].walk(slices.Clip(path), fn)
	}
}

// Check reports the first node of the tree that an adapter must not
// produce: one with no keyword, a schema pointer that is not a JSON
// Pointer, or a Data value of a type the tree does not carry.
func (f Failure) Check() error {
	var err error
	f.Walk(func(path []*Failure, n *Failure) bool {
		if err != nil {
			return false
		}
		where := func() string {
			var kl []string
			for _, p := range path {
				kl = append(kl, p.Keyword)
			}
			return strings.Join(append(kl, n.Keyword), " > ")
		}
		switch {
		case n.Keyword == "":
			err = fmt.Errorf("schemafail: a failure under %q has no keyword", where())
		case !validPointer(n.SchemaPointer):
			err = fmt.Errorf("schemafail: %s: schema pointer %q is not a JSON Pointer", where(), n.SchemaPointer)
		}
		for _, k := range slices.Sorted(maps.Keys(n.Data)) {
			if err == nil && !admitted(n.Data[k]) {
				err = fmt.Errorf("schemafail: %s: data %q is a %T", where(), k, n.Data[k])
			}
		}
		return err == nil
	})
	return err
}

// validPointer reports whether p is an RFC 6901 pointer: empty, or each
// token prefixed by / with ~ only in ~0 and ~1.
func validPointer(p string) bool {
	if p != "" && p[0] != '/' {
		return false
	}
	for i := 0; i < len(p); i++ {
		if p[i] == '~' && (i+1 == len(p) || (p[i+1] != '0' && p[i+1] != '1')) {
			return false
		}
	}
	return true
}

func admitted(v any) bool {
	switch v.(type) {
	case string, bool, int, Number, []string, []bool, []int, []Number:
		return true
	}
	return false
}
