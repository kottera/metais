// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package schema

import (
	"fmt"
	"slices"

	"metais.org/metais/internal/liquid/value"
)

// The Walker also answers what ADR-007 D3's cross-reference reads of a
// document without walking an instance location: the schema at a pointer,
// where each of its references leads, and where a dynamic anchor is
// declared. Each answer is static, as Walk's are: nothing is evaluated.

// Lookup returns the value at the JSON Pointer ptr in the walked document,
// and false when there is none.
func (w *Walker) Lookup(ptr string) (value.Value, bool) {
	return lookup(w.doc, ptr)
}

// Reference returns the pointer of the schema that the reference keyword
// kw, "$ref" or "$dynamicRef", of the schema at ptr names, resolved
// against ptr's base URI (ADR-003 D5 item 3). A `$dynamicRef` resolves to
// its initial target, as a `$ref` with the same text would: the dynamic
// scope is not consulted. ok is false when the schema at ptr has no such
// keyword. A reference that names nothing in the document is ErrUnresolved.
func (w *Walker) Reference(ptr, kw string) (target string, ok bool, err error) {
	if kw != "$ref" && kw != "$dynamicRef" {
		return "", false, fmt.Errorf("%w: %q is not a reference keyword", ErrMalformed, kw)
	}
	v, found := lookup(w.doc, ptr)
	if !found {
		return "", false, fmt.Errorf("%w: no schema at %q", ErrUnresolved, ptr)
	}
	o, isObj := v.(value.Object)
	if !isObj {
		return "", false, nil
	}
	n, ok, err := w.reference(o, ptr, kw)
	if err != nil || !ok {
		return "", false, err
	}
	return n.ptr, true, nil
}

// DynamicAnchors returns the pointer of every schema in the document whose
// `$dynamicAnchor` is name, sorted.
func (w *Walker) DynamicAnchors(name string) []string {
	return sortedPointers(w.dynamic[name])
}

// References returns the pointer of every schema in the document that
// holds a `$ref` or a `$dynamicRef`, sorted.
func (w *Walker) References() []string {
	return sortedPointers(w.refSites)
}

// sortedPointers returns a sorted copy of ptrs, nil when it is empty.
func sortedPointers(ptrs []string) []string {
	if len(ptrs) == 0 {
		return nil
	}
	s := slices.Clone(ptrs)
	slices.Sort(s)
	return s
}
