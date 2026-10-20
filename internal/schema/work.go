// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package schema

import (
	"slices"
	"strings"

	"github.com/santhosh-tekuri/jsonschema/v6"

	"github.com/kottera/metais/internal/limits"
	"github.com/kottera/metais/internal/liquid/value"
	"github.com/kottera/metais/internal/regexprofile"
)

// work is schema-work's count of one validation (ADR-009 D3): the units a
// validation of an instance against a compiled schema costs, counted
// before the library evaluates anything, so that a validation past the
// bound is refused rather than run. The library has no hook in its
// evaluation, and its own order depends on the outcomes of what it has
// evaluated so far, so the count is of every subschema that could apply,
// whatever any evaluation's outcome:
//
//   - each subschema applied to an instance location is one unit, and one
//     more for each member of an object or item of an array it is applied
//     to, since the library visits each of them at every evaluation;
//   - at the instance location itself apply $ref, $dynamicRef resolved by
//     the dynamic scope as the library resolves it, every subschema of
//     allOf, anyOf and oneOf, not, if, then unless if is false, else
//     unless if is true, and the dependentSchemas of each member the
//     object has;
//   - to a member apply its properties subschema, every patternProperties
//     subschema whose pattern matches its name, additionalProperties when
//     neither does, unevaluatedProperties when none of those three
//     applies, and propertyNames to its name, with a dynamic scope of its
//     own, as the library gives it;
//   - to an item apply prefixItems by its index, items beyond them,
//     contains, and unevaluatedItems when neither prefixItems nor items
//     applies.
//
// Members are visited in the instance's order, so that a pattern's
// charge to regex-work, made here as the library will make it again,
// meets that limit at the same point on every run.
type work struct {
	c    *Compiler
	left int64
}

// scope is the dynamic scope: the resources a validation has entered, the
// innermost first.
type scope struct {
	resource string
	outer    *scope
}

// patternProperty is one patternProperties member compiled.
type patternProperty struct {
	re  jsonschema.Regexp
	sub *jsonschema.Schema
}

// charge counts what validating v against s costs and charges it to the
// command's schema-work. It returns a *regexprofile.LimitError naming
// schema-work when the count passes what remains of the bound, and one
// naming regex-work when the command's matching budget is spent.
func (c *Compiler) charge(s *jsonschema.Schema, v value.Value) error {
	if c.meter.Exhausted() {
		return &regexprofile.LimitError{Limit: limits.RegexWork}
	}
	w := &work{c: c, left: c.lim.Value(limits.SchemaWork) - c.meter.SchemaWork()}
	start := w.left
	ok := w.eval(s, v, nil)
	c.meter.ChargeSchemaWork(start - w.left)
	switch {
	case !ok:
		return &regexprofile.LimitError{Limit: limits.SchemaWork}
	case c.meter.Exhausted():
		return &regexprofile.LimitError{Limit: limits.RegexWork}
	}
	return nil
}

// cost is the units of one subschema applied to v: one, and one for each
// member or item v has.
func cost(v value.Value) int64 {
	switch v := v.(type) {
	case value.Object:
		return 1 + int64(v.Len())
	case value.Array:
		return 1 + int64(v.Len())
	}
	return 1
}

// spend takes n units, and reports whether they were there to take.
func (w *work) spend(n int64) bool {
	w.left -= n
	return w.left >= 0
}

// eval counts s applied to v within sc, and reports whether the count is
// still within the bound.
func (w *work) eval(s *jsonschema.Schema, v value.Value, sc *scope) bool {
	if !w.spend(cost(v)) {
		return false
	}
	if s.Bool != nil {
		return true
	}
	if r := w.c.resource(s); sc == nil || sc.resource != r {
		sc = &scope{resource: r, outer: sc}
	}
	here := func(subs ...*jsonschema.Schema) bool {
		for _, t := range subs {
			if t != nil && !w.eval(t, v, sc) {
				return false
			}
		}
		return true
	}
	if !here(s.Ref) || !here(s.AllOf...) || !here(s.AnyOf...) || !here(s.OneOf...) || !here(s.Not, s.If, s.Then, s.Else) {
		return false
	}
	if d := s.DynamicRef; d != nil && !here(w.dynamic(d, sc)) {
		return false
	}
	switch v := v.(type) {
	case value.Object:
		return w.members(s, v, sc)
	case value.Array:
		return w.items(s, v, sc)
	}
	return true
}

// members counts the subschemas s applies to each member of o.
func (w *work) members(s *jsonschema.Schema, o value.Object, sc *scope) bool {
	patterns := w.c.patterns(s)
	for name, m := range o.All() {
		if t, ok := s.DependentSchemas[name]; ok && !w.eval(t, o, sc) {
			return false
		}
		evaluated := false
		if t, ok := s.Properties[name]; ok {
			evaluated = true
			if !w.eval(t, m, sc) {
				return false
			}
		}
		for _, p := range patterns {
			if !p.re.MatchString(name) {
				continue
			}
			evaluated = true
			if !w.eval(p.sub, m, sc) {
				return false
			}
		}
		if !evaluated {
			switch a := s.AdditionalProperties.(type) {
			case *jsonschema.Schema:
				evaluated = true
				if !w.eval(a, m, sc) {
					return false
				}
			case bool:
				// A boolean subschema, held as the boolean itself.
				evaluated = true
				if !w.spend(cost(m)) {
					return false
				}
			}
		}
		if t := s.UnevaluatedProperties; t != nil && !evaluated && !w.eval(t, m, sc) {
			return false
		}
		if t := s.PropertyNames; t != nil {
			n, err := value.NewString(name)
			if err != nil {
				// An object's member names are scalar values.
				panic(err)
			}
			if !w.eval(t, n, nil) {
				return false
			}
		}
	}
	return true
}

// items counts the subschemas s applies to each item of a.
func (w *work) items(s *jsonschema.Schema, a value.Array, sc *scope) bool {
	for i, item := range a.All() {
		evaluated := true
		switch {
		case i < len(s.PrefixItems):
			if !w.eval(s.PrefixItems[i], item, sc) {
				return false
			}
		case s.Items2020 != nil:
			if !w.eval(s.Items2020, item, sc) {
				return false
			}
		default:
			evaluated = false
		}
		if t := s.Contains; t != nil && !w.eval(t, item, sc) {
			return false
		}
		if t := s.UnevaluatedItems; t != nil && !evaluated && !w.eval(t, item, sc) {
			return false
		}
	}
	return true
}

// dynamic resolves d as the library does: to its initial target, unless
// that declares the $dynamicAnchor d names, in which case to the
// outermost resource in sc that declares one of that name.
func (w *work) dynamic(d *jsonschema.DynamicRef, sc *scope) *jsonschema.Schema {
	t := d.Ref
	if d.Anchor == "" || t.DynamicAnchor != d.Anchor {
		return t
	}
	for ; sc != nil; sc = sc.outer {
		if a := w.c.dynamicAnchor(sc.resource, d.Anchor); a != nil {
			t = a
		}
	}
	return t
}

// resource returns the key of the resource s is in: its schema file and
// the pointer of the nearest $id at or above it, or, for a meta-schema,
// its URI, each meta-schema being one resource.
func (c *Compiler) resource(s *jsonschema.Schema) string {
	if r, ok := c.resourceOf[s]; ok {
		return r
	}
	r, _, _ := strings.Cut(s.Location, "#")
	if entry, ptr, ok := c.locate(s.Location); ok {
		r = entry + "#" + c.resourcePointer(entry, ptr)
	}
	c.resourceOf[s] = r
	return r
}

// resourcePointer returns the pointer, in entry, of the resource holding
// the subschema at ptr: the longest of the file's resources' pointers that
// is ptr or one of its ancestors.
func (c *Compiler) resourcePointer(entry, ptr string) string {
	best := ""
	for _, p := range c.resources[entry] {
		if len(p) > len(best) && (ptr == p || strings.HasPrefix(ptr, p+"/")) {
			best = p
		}
	}
	return best
}

// dynamicAnchor returns the schema declaring the $dynamicAnchor name in
// the resource keyed r, or nil.
func (c *Compiler) dynamicAnchor(r, name string) *jsonschema.Schema {
	entry, rptr, inPackage := strings.Cut(r, "#")
	if !inPackage {
		// A meta-schema declares its dynamic anchor at its root.
		root, err := c.lib.Compile(r)
		if err != nil || root.DynamicAnchor != name {
			return nil
		}
		return root
	}
	for _, d := range c.dynamic[entry] {
		if d.name != name || c.resourcePointer(entry, d.ptr) != rptr {
			continue
		}
		sub, err := c.Subschema(entry, d.ptr)
		if err != nil {
			panic(err)
		}
		return sub.lib
	}
	return nil
}

// patterns returns s's patternProperties in the order of their patterns,
// so that their charges to regex-work are made in one order on every run.
func (c *Compiler) patterns(s *jsonschema.Schema) []patternProperty {
	if len(s.PatternProperties) == 0 {
		return nil
	}
	if ps, ok := c.patternsOf[s]; ok {
		return ps
	}
	ps := make([]patternProperty, 0, len(s.PatternProperties))
	for re, sub := range s.PatternProperties {
		ps = append(ps, patternProperty{re, sub})
	}
	slices.SortFunc(ps, func(a, b patternProperty) int { return strings.Compare(a.re.String(), b.re.String()) })
	c.patternsOf[s] = ps
	return ps
}
