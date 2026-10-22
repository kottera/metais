// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package schema

import (
	"fmt"

	"metais.org/metais/internal/liquid/value"
)

// Chain returns s and every schema its `$ref` chain reaches, in order:
// s itself, the target of its `$ref`, that target's own `$ref` target, and
// so on until a schema has no `$ref` (ADR-003 D14 item 1). Each is a
// Static whose Types is the type set declared from it onward. It is how a
// reader finds what a location's schema says beyond its own members: D19
// applies the schema object's own `properties` first and then those
// reached through its `$ref`, and takes a member's `default` from the first
// schema in its chain that has one.
//
// A chain that returns to a schema it has passed through is ErrRefCycle;
// a `$ref` to a 2020-12 meta-schema, which the walked document does not
// hold, is ErrUnresolved, as Walk reports it.
func (w *Walker) Chain(s Static) ([]Static, error) {
	cur := node{v: s.Schema, ptr: s.Pointer}
	seen := map[string]bool{}
	var out []Static
	for {
		if seen[cur.ptr] {
			return nil, fmt.Errorf("%w: through %q", ErrRefCycle, cur.ptr)
		}
		seen[cur.ptr] = true
		types, err := w.declared(cur)
		if err != nil {
			return nil, err
		}
		out = append(out, Static{Schema: cur.v, Pointer: cur.ptr, Types: types})
		o, isObj := cur.v.(value.Object)
		if !isObj {
			return out, nil
		}
		next, ok, err := w.ref(o, cur.ptr)
		if err != nil {
			return nil, err
		}
		if !ok {
			return out, nil
		}
		cur = next
	}
}
