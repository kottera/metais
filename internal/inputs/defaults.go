// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package inputs

import (
	"slices"

	"metais.org/metais/internal/finding"
	"metais.org/metais/internal/liquid/value"
	"metais.org/metais/internal/schema"
)

// defaultAt is where a value a default supplied came from: a `default`,
// with no file and no line (ADR-006 D4 item 4, D5 item 5).
var defaultAt = plainLocator{finding.Location{Origin: finding.OriginDefault}}

// applyDefaults applies the schemas' defaults to the assembled inputs
// object at root (ADR-003 D19), by a walk over the instance from the
// combined schema, whose root is the inputs object's: an absent optional
// input takes its root schema's default as an absent optional member does.
// leftOut names the inputs left out of the instance because they did not
// parse, which are not absent and take no default (D22 item 3). It
// returns the inputs a default was applied in.
func (l *Layer) applyDefaults(root *node, leftOut map[string]bool) map[string]bool {
	d := &defaulter{w: l.walker, leftOut: leftOut, applied: map[string]bool{}}
	d.walk(root, nil)
	return d.applied
}

type defaulter struct {
	w       *walker
	leftOut map[string]bool
	applied map[string]bool
}

// walk applies the defaults at n, whose instance location is path, and
// then beneath it.
func (d *defaulter) walk(n *node, path []schema.Step) {
	switch {
	case n.isObj:
		if s, ok := d.w.static(path); ok {
			d.members(n, path, s)
		}
		for _, name := range sortedNames(n) {
			d.walk(n.obj[name], appendStep(path, schema.Member(name)))
		}
	case n.isArr:
		// D19 item 1: each existing element, with its static subschema.
		// An element keeps the index it had before any item was left out,
		// so prefixItems answers for the item it names.
		for _, e := range n.arr {
			d.walk(e, appendStep(path, schema.Index(e.index)))
		}
	}
}

// members supplies, at the object n whose static schema is s, each absent
// member a `properties` of s's chain declares with a default and does not
// list in the same schema object's `required`: s's own first, then those
// reached through its `$ref` (D19 item 1). A member already present, null
// included, is left alone (D19 item 3).
func (d *defaulter) members(n *node, path []schema.Step, s schema.Static) {
	for _, c := range d.w.chain(s) {
		o, ok := c.Schema.(value.Object)
		if !ok {
			continue
		}
		pv, ok := o.Get("properties")
		props, isObj := pv.(value.Object)
		if !ok || !isObj {
			continue
		}
		required := names(o, "required")
		for _, name := range props.Names() {
			if _, present := n.obj[name]; present || slices.Contains(required, name) {
				continue
			}
			if len(path) == 0 && d.leftOut[name] {
				continue
			}
			sub, _ := props.Get(name)
			def, ok := d.defaultOf(schema.Static{Schema: sub, Pointer: c.Pointer + "/properties/" + finding.Pointer(name)[1:]})
			if !ok {
				continue
			}
			n.obj[name] = fromValue(def, defaultAt)
			input := name
			if len(path) > 0 {
				input = path[0].String()
			}
			d.applied[input] = true
		}
	}
}

// defaultOf is the `default` of the member subschema s: on s itself, else
// on the first schema of its `$ref` chain that has one (D19 item 1).
func (d *defaulter) defaultOf(s schema.Static) (value.Value, bool) {
	for _, c := range d.w.chain(s) {
		if o, ok := c.Schema.(value.Object); ok {
			if v, ok := o.Get("default"); ok {
				return v, true
			}
		}
	}
	return nil, false
}

// names is the strings of o's member key, an array of member names.
func names(o value.Object, key string) []string {
	v, ok := o.Get(key)
	if !ok {
		return nil
	}
	a, ok := v.(value.Array)
	if !ok {
		return nil
	}
	var out []string
	for _, e := range a.All() {
		if s, ok := e.(value.String); ok {
			out = append(out, s.Text())
		}
	}
	return out
}

// sortedNames is n's member names in byte order, so the walk visits them
// in an order that owes nothing to a map's.
func sortedNames(n *node) []string {
	out := make([]string, 0, len(n.obj))
	for name := range n.obj {
		out = append(out, name)
	}
	slices.Sort(out)
	return out
}

func appendStep(path []schema.Step, s schema.Step) []schema.Step {
	return append(path[:len(path):len(path)], s)
}
