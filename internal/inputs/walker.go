// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package inputs

import (
	"errors"
	"fmt"

	"github.com/kottera/metais/internal/liquid/value"
	"github.com/kottera/metais/internal/schema"
)

// walker is the static walk of ADR-003 D14 item 1 over the combined
// schema, whose root is the inputs object's schema: a path's first step is
// an input's name.
type walker struct {
	w *schema.Walker
}

func newWalker(combined value.Value) (*walker, error) {
	w, err := schema.NewWalker(combined, schema.CombinedURI)
	if err != nil {
		return nil, fmt.Errorf("inputs: the combined schema cannot be walked: %w", err)
	}
	return &walker{w: w}, nil
}

// static is the static subschema at path, and false when none is
// declared there. A `$ref` the walked document does not hold, one to a
// 2020-12 meta-schema, declares nothing the walk can read. Any other
// refusal is a schema the compiler accepted and the walk cannot read,
// which is a defect in the tool.
func (w *walker) static(path []schema.Step) (schema.Static, bool) {
	s, ok, err := w.w.Walk(path)
	if errors.Is(err, schema.ErrUnresolved) {
		return schema.Static{}, false
	}
	if err != nil {
		panic(fmt.Sprintf("inputs: the static walk of a compiled schema: %v", err))
	}
	return s, ok
}

// types is the declared type set at path, empty where none is declared.
func (w *walker) types(path []schema.Step) schema.TypeSet {
	s, ok := w.static(path)
	if !ok {
		return 0
	}
	return s.Types
}

// pointer is the static subschema's pointer at path, which names it
// within the combined schema, or a text no pointer is when none is
// declared there.
func (w *walker) pointer(path []schema.Step) string {
	s, ok := w.static(path)
	if !ok {
		return "\x00"
	}
	return s.Pointer
}

// chain is the `$ref` chain from s (schema.Walker.Chain), or s alone when
// the chain leaves the walked document.
func (w *walker) chain(s schema.Static) []schema.Static {
	c, err := w.w.Chain(s)
	if errors.Is(err, schema.ErrUnresolved) {
		return []schema.Static{s}
	}
	if err != nil {
		panic(fmt.Sprintf("inputs: the $ref chain of a compiled schema: %v", err))
	}
	return c
}
