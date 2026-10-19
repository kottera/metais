// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package inputs

import (
	"crypto/sha256"
	"slices"

	"github.com/kottera/metais/internal/jsonx"
	"github.com/kottera/metais/internal/liquid/value"
	"github.com/kottera/metais/internal/manifest"
)

// Record is what a render is given for one declared input (ADR-003 D21).
type Record struct {
	// Name is the input's name.
	Name string
	// Value is the validated value after defaults, bound to
	// inputs.<name>; nil when the input is absent (D21 item 1).
	Value value.Value
	// Unparsed is set when the input's file or argument did not parse, so
	// it was left out of the instance (D22 item 3). Its Value is nil.
	Unparsed bool
	// Raw is the original bytes of the one data-directory file the whole
	// value came from, exactly as read, byte order mark and line breaks
	// included, when no --inputs-json member or path set touched it; nil
	// otherwise, and always for a collection, whose items have their own
	// (D21 item 2). HasRaw tells an empty file's raw form from none.
	Raw    []byte
	HasRaw bool
	// Source is where the input came from (D21 item 3).
	Source Source
	// From is every source the value came from, in the order applied:
	// what validate's result lists (ADR-008 D7 item 5).
	From []From
	// Digest is SHA-256 over the canonical JSON serialization of Value
	// (RFC 8785, ADR-002 D9), computed after defaults; HasDigest is false
	// when the input is absent (D21 item 4).
	Digest    [sha256.Size]byte
	HasDigest bool
	// Items is a collection's items, one per element of Value, in order;
	// nil for an input that is not a collection or is absent.
	Items []Item
}

// Item is what a render is given for one item of a collection (ADR-003
// D21 items 2 to 4).
type Item struct {
	// FromFile is set when the item is a data-directory file; Name, Stem
	// and Form are its, and empty for an item supplied on the command
	// line (D21 item 3).
	FromFile bool
	Name     string
	Stem     string
	Form     manifest.Form
	// Raw is the item file's bytes, when no path set touched the item.
	Raw    []byte
	HasRaw bool
	// Digest is SHA-256 over the item's canonical JSON serialization.
	Digest [sha256.Size]byte
}

// SourceKind is where an input came from, as a whole (ADR-003 D21 item 3).
type SourceKind uint8

const (
	// SourceNone is an absent input's.
	SourceNone SourceKind = iota
	// SourceFile is a value whose root is a data-directory file's, or a
	// collection's directory: File names it.
	SourceFile
	// SourceCommandLine is a value an --inputs-json member, or a path set
	// at the input's root or into an input no file supplied, gave.
	SourceCommandLine
	// SourceDefault is an absent optional input's schema default.
	SourceDefault
)

// Source is where an input came from (ADR-003 D21 item 3).
type Source struct {
	Kind SourceKind
	// File is the data-directory path, relative to the data directory,
	// of a SourceFile.
	File string
}

// FromKind is one kind of source a value came from (ADR-008 D7 item 5).
type FromKind uint8

const (
	FromFile FromKind = iota + 1
	FromInputsJSON
	FromSet
	FromSetJSON
	FromDefault
)

// From is one source a value came from: a data-directory File, the
// Argument'th --inputs-json, the Argument'th path set, or a default.
type From struct {
	Kind     FromKind
	File     string
	Argument int
}

// record is the input in's record, from the assembled object root;
// instance is the inputs object validated, whose members are the values.
func (s *Supply) record(in manifest.Input, root *node, instance value.Object, st *state, leftOut, defaulted bool) Record {
	r := Record{Name: in.Name, Unparsed: leftOut}
	n, ok := root.obj[in.Name]
	if !ok {
		return r
	}
	r.Value, _ = instance.Get(in.Name)
	r.Digest, r.HasDigest = sha256.Sum256(jsonx.Canonical(r.Value)), true
	r.From = slices.Clone(st.from)
	switch {
	case st.fromFile:
		r.Source = Source{Kind: SourceFile, File: s.files[in.Name].file}
	case len(st.from) == 0:
		r.Source = Source{Kind: SourceDefault}
	default:
		r.Source = Source{Kind: SourceCommandLine}
	}
	if defaulted {
		r.From = append(r.From, From{Kind: FromDefault})
	}
	fi := s.files[in.Name]
	if st.fromFile && !st.touched && !in.Collection && fi != nil {
		r.Raw, r.HasRaw = fi.raw, true
	}
	if arr, isArr := r.Value.(value.Array); in.Collection && isArr {
		r.Items = make([]Item, len(n.arr))
		for i, e := range n.arr {
			it := Item{Digest: sha256.Sum256(jsonx.Canonical(arr.At(i)))}
			if e.item != nil && st.fromFile {
				it.FromFile, it.Name, it.Stem, it.Form = true, e.item.name, e.item.stem, e.item.form
				if !slices.Contains(st.touchedItems, e.item.index) {
					it.Raw, it.HasRaw = e.item.raw, true
				}
			}
			r.Items[i] = it
		}
	}
	return r
}
