// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package sample

import (
	"math/big"
	"slices"
	"strconv"

	"metais.org/metais/internal/finding"
	"metais.org/metais/internal/liquid/value"
)

// node is one schema in a package: a schema file and the JSON Pointer of
// the schema within it.
type node struct {
	entry, ptr string
}

// child is the node of the schema at keyword, and at each token after it,
// below n.
func (n node) child(keyword string, tokens ...string) node {
	return node{n.entry, n.ptr + finding.Pointer(append([]string{keyword}, tokens...)...)}
}

// conj is the conjunction C at an instance location (ADR-007 D9 item 7):
// the schemas the location starts from and, recursively, their $ref and
// $dynamicRef targets and allOf members, with a branch of each anyOf and
// oneOf (item 8) and each conditional's selected branch (item 9).
type conj struct {
	// base are the schemas the location started from, and selected the
	// conditional branches added to them.
	base []node
	// starts are the schemas the location is validated against: the ones
	// it started from and every branch added to it.
	starts []node
	// nodes is every schema object of C, in walk order: each schema, then
	// its $ref target, its $dynamicRef target and its allOf members, in
	// order.
	nodes []node
	objs  []value.Object
	// falseAt is a false schema in C, when there is one: no value
	// satisfies C.
	falseAt *node
	// entered is every $ref and $dynamicRef target walked into, in walk
	// order, for the recursion bound (item 20).
	entered []node
	// groups is each anyOf and oneOf of C, in walk order, with the branch
	// chosen for it.
	groups []group
}

// group is one anyOf or oneOf in C and the branch chosen of it.
type group struct {
	at       node // the schema holding the keyword
	keyword  string
	branches int
	chosen   int
}

func (g group) key() string { return g.at.entry + "#" + g.at.ptr + "/" + g.keyword }

// collect adds n and what it brings into C: its $ref and $dynamicRef
// targets and its allOf members, depth first in that order. seen stops a
// schema from being walked twice at one location.
func (s *Generator) collect(c *conj, n node, seen map[node]bool) {
	if seen[n] {
		return
	}
	seen[n] = true
	v, ok := s.c.Lookup(n.entry, n.ptr)
	if !ok {
		return
	}
	switch v := v.(type) {
	case value.Bool:
		if !v && c.falseAt == nil {
			c.falseAt = &n
		}
		return
	case value.Object:
		c.nodes = append(c.nodes, n)
		c.objs = append(c.objs, v)
		if e, p, ok := s.c.Ref(n.entry, n.ptr); ok {
			t := node{e, p}
			c.entered = append(c.entered, t)
			s.collect(c, t, seen)
		}
		if e, p, ok := s.c.DynamicRef(n.entry, n.ptr); ok {
			t := node{e, p}
			c.entered = append(c.entered, t)
			s.collect(c, t, seen)
		}
		if all, ok := v.Get("allOf"); ok {
			if a, ok := all.(value.Array); ok {
				for i := range a.Len() {
					s.collect(c, n.child("allOf", strconv.Itoa(i)), seen)
				}
			}
		}
	}
}

// effective builds C at loc from starts and extra, the branches already
// selected for it. Each anyOf and oneOf C holds takes a branch: the one
// choose names for it when it names one, and otherwise one drawn from the
// location's branch stream, in walk order (item 8). The branch is added to
// C, which may bring further groups.
func (s *Generator) effective(loc *place, starts []node, choose map[string]int) *conj {
	c := &conj{base: starts}
	seen := map[node]bool{}
	for _, n := range starts {
		c.starts = append(c.starts, n)
		s.collect(c, n, seen)
	}
	var branchStream *SplitMix64
	done := map[string]bool{}
	for {
		added := false
		for i := 0; i < len(c.objs); i++ {
			for _, kw := range []string{"anyOf", "oneOf"} {
				bv, ok := c.objs[i].Get(kw)
				if !ok {
					continue
				}
				arr, ok := bv.(value.Array)
				if !ok || arr.Len() == 0 {
					continue
				}
				gr := group{at: c.nodes[i], keyword: kw, branches: arr.Len()}
				if done[gr.key()] {
					continue
				}
				done[gr.key()] = true
				if k, ok := choose[gr.key()]; ok {
					gr.chosen = k
				} else {
					if branchStream == nil {
						branchStream = s.stream(loc.ptr, purposeBranch)
					}
					gr.chosen = int(branchStream.Below(uint64(arr.Len())))
				}
				c.groups = append(c.groups, gr)
				b := gr.at.child(kw, strconv.Itoa(gr.chosen))
				c.starts = append(c.starts, b)
				s.collect(c, b, seen)
				added = true
			}
		}
		if !added {
			return c
		}
	}
}

// with returns C with extra, further selected branches, added: a new C
// over the schemas it started from and extra, keeping the branches it
// chose.
func (s *Generator) with(loc *place, c *conj, extra []node) *conj {
	choose := map[string]int{}
	for _, g := range c.groups {
		choose[g.key()] = g.chosen
	}
	return s.effective(loc, slices.Concat(c.base, extra), choose)
}

// each calls f with every schema object of C that has keyword, and the
// keyword's value there.
func (c *conj) each(keyword string, f func(n node, v value.Value)) {
	for i, o := range c.objs {
		if v, ok := o.Get(keyword); ok {
			f(c.nodes[i], v)
		}
	}
}

// has reports whether any schema of C has one of keywords.
func (c *conj) has(keywords ...string) bool {
	for _, o := range c.objs {
		for _, k := range keywords {
			if _, ok := o.Get(k); ok {
				return true
			}
		}
	}
	return false
}

// typeNames is the type keyword's value as a list of names.
func typeNames(v value.Value) []string {
	switch v := v.(type) {
	case value.String:
		return []string{v.Text()}
	case value.Array:
		var out []string
		for _, e := range v.All() {
			if s, ok := e.(value.String); ok {
				out = append(out, s.Text())
			}
		}
		return out
	}
	return nil
}

// allows reports whether a type keyword naming set admits the type t:
// integer is admitted by number too.
func allows(set []string, t string) bool {
	return slices.Contains(set, t) || t == "integer" && slices.Contains(set, "number")
}

// types returns C's type set (item 7, intersected across C) in the order
// the first schema to declare one writes it, and whether any schema
// declares one. A set number meets integer as integer.
func (c *conj) types() ([]string, bool) {
	var sets [][]string
	c.each("type", func(_ node, v value.Value) { sets = append(sets, typeNames(v)) })
	if len(sets) == 0 {
		return nil, false
	}
	universe := slices.Clone(sets[0])
	if i := slices.Index(universe, "number"); i >= 0 && !slices.Contains(universe, "integer") {
		universe = slices.Insert(universe, i+1, "integer")
	}
	var out []string
	for _, t := range universe {
		ok := true
		for _, set := range sets {
			ok = ok && allows(set, t)
		}
		if ok {
			out = append(out, t)
		}
	}
	// integer, inserted after number, is redundant beside it.
	if slices.Contains(out, "number") && !slices.Contains(sets[0], "integer") {
		out = slices.DeleteFunc(out, func(t string) bool { return t == "integer" })
	}
	return out, true
}

// inferred is the type a schema with no type keyword is taken to have
// (item 14).
func (c *conj) inferred() string {
	switch {
	case c.has("properties", "required", "additionalProperties", "patternProperties", "minProperties", "maxProperties", "propertyNames", "dependentRequired", "dependentSchemas"):
		return "object"
	case c.has("items", "prefixItems", "contains", "minItems", "maxItems", "uniqueItems"):
		return "array"
	case c.has("minLength", "maxLength", "pattern", "format"):
		return "string"
	case c.has("minimum", "maximum", "exclusiveMinimum", "exclusiveMaximum", "multipleOf"):
		return "number"
	}
	return "string"
}

// consts returns the distinct const values of C, and whether any schema
// has one.
func (c *conj) consts() ([]value.Value, bool) {
	var out []value.Value
	found := false
	c.each("const", func(_ node, v value.Value) {
		found = true
		if !slices.ContainsFunc(out, func(o value.Value) bool { return value.Equal(o, v) }) {
			out = append(out, v)
		}
	})
	return out, found
}

// enum returns the members of the first enum of C that every other enum
// of C holds too, in its order, and whether any schema has one.
func (c *conj) enum() ([]value.Value, bool) {
	var sets []value.Array
	c.each("enum", func(_ node, v value.Value) {
		if a, ok := v.(value.Array); ok {
			sets = append(sets, a)
		}
	})
	if len(sets) == 0 {
		return nil, false
	}
	var out []value.Value
	for _, m := range sets[0].All() {
		in := true
		for _, other := range sets[1:] {
			found := false
			for _, o := range other.All() {
				found = found || value.Equal(m, o)
			}
			in = in && found
		}
		if in && !slices.ContainsFunc(out, func(o value.Value) bool { return value.Equal(o, m) }) {
			out = append(out, m)
		}
	}
	return out, true
}

// first returns the value of keyword in the first schema of C that has
// it.
func (c *conj) first(keyword string) (value.Value, bool) {
	for _, o := range c.objs {
		if v, ok := o.Get(keyword); ok {
			return v, true
		}
	}
	return nil, false
}

// rat is the exact rational a number's literal writes.
func rat(n value.Number) *big.Rat {
	r, ok := new(big.Rat).SetString(n.Literal())
	if !ok {
		// Every literal of the data model is an RFC 8259 number, which
		// big.Rat reads.
		panic("sample: a number literal big.Rat does not read: " + n.Literal())
	}
	return r
}

// bound is one side of a numeric range: a value, and whether it is
// excluded.
type bound struct {
	v         *big.Rat
	exclusive bool
}

// numBounds returns the tightest lower and upper bounds C sets (item 7),
// nil for a side it leaves open.
func (c *conj) numBounds() (lo, hi *bound) {
	tighter := func(cur *bound, v *big.Rat, excl bool, lower bool) *bound {
		if cur == nil {
			return &bound{v, excl}
		}
		switch cmp := v.Cmp(cur.v); {
		case lower && cmp > 0, !lower && cmp < 0:
			return &bound{v, excl}
		case cmp == 0 && excl:
			return &bound{v, true}
		}
		return cur
	}
	for _, kw := range []struct {
		name        string
		excl, lower bool
	}{{"minimum", false, true}, {"exclusiveMinimum", true, true}, {"maximum", false, false}, {"exclusiveMaximum", true, false}} {
		c.each(kw.name, func(_ node, v value.Value) {
			n, ok := v.(value.Number)
			if !ok {
				return
			}
			if kw.lower {
				lo = tighter(lo, rat(n), kw.excl, true)
			} else {
				hi = tighter(hi, rat(n), kw.excl, false)
			}
		})
	}
	return lo, hi
}

// multipleOf returns the least common multiple of C's multipleOf values,
// as an exact rational (item 7), or nil when C has none.
func (c *conj) multipleOf() *big.Rat {
	var m *big.Rat
	c.each("multipleOf", func(_ node, v value.Value) {
		n, ok := v.(value.Number)
		if !ok {
			return
		}
		r := rat(n)
		if r.Sign() <= 0 {
			return
		}
		if m == nil {
			m = r
			return
		}
		// lcm(a/b, c/d) = lcm(a, c) / gcd(b, d), both in lowest terms.
		a, b, cc, d := m.Num(), m.Denom(), r.Num(), r.Denom()
		g := new(big.Int).GCD(nil, nil, a, cc)
		l := new(big.Int).Mul(a, cc)
		l.Quo(l, g)
		m = new(big.Rat).SetFrac(l, new(big.Int).GCD(nil, nil, b, d))
	})
	return m
}

// count returns the tightest bound C sets with keyword, a non-negative
// integer: the largest for a minimum, the smallest for a maximum. ok is
// false when C sets none.
func (c *conj) count(keyword string, lower bool) (n int64, ok bool) {
	c.each(keyword, func(_ node, v value.Value) {
		num, isNum := v.(value.Number)
		if !isNum {
			return
		}
		f := num.Float64()
		if f > 1<<62 {
			f = 1 << 62
		}
		x := int64(f)
		if !ok || lower && x > n || !lower && x < n {
			n, ok = x, true
		}
	})
	return n, ok
}

// strings returns the string values of keyword across C, in walk order.
func (c *conj) strings(keyword string) []string {
	var out []string
	c.each(keyword, func(_ node, v value.Value) {
		if s, ok := v.(value.String); ok {
			out = append(out, s.Text())
		}
	})
	return out
}

// nodesAt returns the node of keyword in every schema of C that has it.
func (c *conj) nodesAt(keyword string) []node {
	var out []node
	c.each(keyword, func(n node, _ value.Value) { out = append(out, n.child(keyword)) })
	return out
}
