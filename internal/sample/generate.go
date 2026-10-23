// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package sample

import (
	"cmp"
	"maps"
	"math"
	"slices"
	"strconv"
	"strings"

	"metais.org/metais/internal/finding"
	"metais.org/metais/internal/jsonx"
	"metais.org/metais/internal/liquid/value"
	"metais.org/metais/internal/regexprofile"
)

// The generator's fixed bounds (ADR-007 D9 items 9, 10 and 16): passes
// for conditionals to settle, and regenerations for not and uniqueItems.
const (
	maxPasses   = 4
	maxAttempts = 16
)

// value makes the value at loc from the schemas starts. Each anyOf and
// oneOf in C takes a branch, and when the finished value does not match
// exactly one branch of a oneOf, the next branch in order is tried, each
// once (item 8). A branch from which no value can be made is passed over
// the same way, for the next.
func (s *Generator) value(loc *place, starts []node) (value.Value, bool) {
	if s.stopped() {
		return nil, false
	}
	choose := map[string]int{}
	tried := map[string]int{}
	// next moves the first group of c that has a branch left, and that
	// moves, to its next branch, and reports whether one did.
	next := func(c *conj, moves func(group) bool) bool {
		for _, g := range c.groups {
			if tried[g.key()]+1 >= g.branches || !moves(g) {
				continue
			}
			tried[g.key()]++
			for _, h := range c.groups {
				choose[h.key()] = h.chosen
			}
			choose[g.key()] = (g.chosen + 1) % g.branches
			return true
		}
		return false
	}
	for {
		mark := len(s.findings)
		c := s.effective(loc, starts, choose)
		v, ok := s.located(loc, c)
		if s.stopped() {
			return nil, false
		}
		if !ok {
			if next(c, func(group) bool { return true }) {
				s.findings = s.findings[:mark]
				continue
			}
			return nil, false
		}
		mismatched := func(g group) bool {
			if g.keyword != "oneOf" {
				return false
			}
			matched := 0
			for i := range g.branches {
				if s.valid([]node{g.at.child("oneOf", itoa(i))}, v) {
					matched++
				}
			}
			return matched != 1
		}
		if !next(c, mismatched) {
			return v, true
		}
		s.findings = s.findings[:mark]
	}
}

// located makes the value at loc from C, once the path's recursion is
// counted (item 20): from C itself, then with each conditional branch the
// value selects (item 9), then away from every not (item 10).
func (s *Generator) located(at *place, c *conj) (value.Value, bool) {
	loc := *at
	if len(c.entered) > 0 {
		loc.refs = make(map[node]int, len(at.refs)+len(c.entered))
		for t, n := range at.refs {
			loc.refs[t] = n
		}
		for _, t := range c.entered {
			loc.refs[t]++
			switch {
			case loc.refs[t] >= failRecursion:
				s.unsatisfiable(at, t, ReasonRecursion)
				return nil, false
			case loc.refs[t] >= stopRecursion:
				loc.stop = true
			}
		}
	}
	if c.falseAt != nil {
		s.unsatisfiable(&loc, *c.falseAt, ReasonType)
		return nil, false
	}
	if loc.fixed != nil {
		if _, ok := loc.fixed.(value.Object); !ok || !fillsAround(&loc, c) {
			// A supplied value is taken as given (item 2), an object too
			// where C makes no object to fill it around; validation then
			// reports one that C refuses.
			return loc.fixed, true
		}
	}
	v, ok := s.fromC(&loc, c)
	if !ok || !s.spend(&loc, size(v)) {
		return nil, false
	}
	if _, isObj := v.(value.Object); !isObj {
		// An object settles its conditionals as its properties are made.
		for pass := 0; ; pass++ {
			var failing []node
			for _, n := range s.selected(c, v) {
				if !s.valid([]node{n}, v) {
					failing = append(failing, n)
				}
			}
			if len(failing) == 0 {
				break
			}
			if pass == maxPasses-1 {
				s.unsatisfiable(&loc, c.starts[0], ReasonConditionals)
				return nil, false
			}
			c = s.with(&loc, c, s.selected(c, v))
			if v, ok = s.fromC(&loc, c); !ok {
				return nil, false
			}
		}
	}
	nots := c.nodesAt("not")
	for k := 1; s.matchesAny(nots, v); k++ {
		if k > maxAttempts {
			s.unsatisfiable(&loc, c.starts[0], ReasonNot)
			return nil, false
		}
		again := loc
		again.purpose = attempt(k)
		if v, ok = s.fromC(&again, c); !ok {
			return nil, false
		}
	}
	return v, true
}

// fillsAround reports whether C makes an object at loc that a supplied
// object is filled around (item 2): one whose type set allows an object,
// or that declares none, with no const or enum to take whole, outside a
// CSV field.
func fillsAround(loc *place, c *conj) bool {
	if loc.cell || c.has("const", "enum") {
		return false
	}
	types, declared := c.types()
	return !declared || slices.Contains(types, "object")
}

// size is what v costs spend: a string its bytes, any other value one.
// The members and items of an object or array are charged as they are
// made.
func size(v value.Value) int64 {
	if str, ok := v.(value.String); ok {
		return int64(len(str.Text())) + 1
	}
	return 1
}

// matchesAny reports whether v validates against any schema of ns.
func (s *Generator) matchesAny(ns []node, v value.Value) bool {
	for _, n := range ns {
		if s.valid([]node{n}, v) {
			return true
		}
	}
	return false
}

// selected returns the then or else of every if in C that v selects
// (item 9), and the dependentSchemas whose trigger v holds.
func (s *Generator) selected(c *conj, v value.Value) []node {
	var out []node
	for i, o := range c.objs {
		n := c.nodes[i]
		if _, ok := o.Get("if"); ok {
			branch := "else"
			if s.valid([]node{n.child("if")}, v) {
				branch = "then"
			}
			if _, ok := o.Get(branch); ok {
				out = append(out, n.child(branch))
			}
		}
		obj, isObj := v.(value.Object)
		if ds, ok := o.Get("dependentSchemas"); ok && isObj {
			if dso, ok := ds.(value.Object); ok {
				for trigger := range dso.All() {
					if _, present := obj.Get(trigger); present {
						out = append(out, n.child("dependentSchemas", trigger))
					}
				}
			}
		}
	}
	return out
}

// fromC makes a value from C: the author's own, when C has one that
// validates (item 12), and otherwise one generated by type (item 14).
func (s *Generator) fromC(loc *place, c *conj) (value.Value, bool) {
	if v, ok := s.candidate(loc, c); ok {
		return v, true
	}
	if s.err != nil {
		return nil, false
	}
	if _, has := c.consts(); has {
		s.unsatisfiable(loc, s.holder(c, "const"), ReasonConst)
		return nil, false
	}
	if _, has := c.enum(); has {
		s.unsatisfiable(loc, s.holder(c, "enum"), ReasonEnum)
		return nil, false
	}
	t, ok := s.chooseType(loc, c)
	if !ok {
		s.unsatisfiable(loc, s.holder(c, "type"), ReasonType)
		return nil, false
	}
	switch t {
	case "null":
		return value.Null{}, true
	case "boolean":
		// true when the value stream's first draw is even.
		return value.Bool(s.stream(loc.ptr, loc.valuePurpose()).Next()%2 == 0), true
	case "object":
		return s.object(loc, c)
	case "array":
		return s.array(loc, c)
	case "integer", "number":
		return s.number(loc, c, t == "integer")
	}
	return s.text(loc, c)
}

// holder is the first schema of C that has keyword, or C's first schema
// when none does: the subschema a finding about the keyword names.
func (s *Generator) holder(c *conj, keyword string) node {
	for i, o := range c.objs {
		if _, ok := o.Get(keyword); ok {
			return c.nodes[i]
		}
	}
	if len(c.starts) > 0 {
		return c.starts[0]
	}
	return node{s.input.Schema, ""}
}

// candidate returns the author's own value for loc, by item 12's order of
// preference — const, an entry of examples, default, an enum member —
// each kept only when it validates against C, can be written so that it
// reads back (writable), and, in a uniqueItems array, equals no earlier
// item, nor, at a uniqueKeys key of one pointer, an earlier item's key.
// An object or array is taken whole from examples or default only when C
// declares no properties or items for it (item 13).
func (s *Generator) candidate(loc *place, c *conj) (value.Value, bool) {
	if loc.fixed != nil {
		return nil, false
	}
	keep := func(v value.Value) bool {
		return s.writable(loc, v) &&
			!slices.ContainsFunc(loc.earlier, func(e value.Value) bool { return value.Equal(e, v) }) &&
			!loc.avoids(v) &&
			s.valid(c.starts, v)
	}
	whole := func(v value.Value) bool {
		switch v.(type) {
		case value.Object:
			return !c.has("properties", "patternProperties")
		case value.Array:
			return !c.has("items", "prefixItems")
		}
		return true
	}
	// cycle tries the members of list from (seed + k) mod count on, moved
	// on by a draw of the attempt's stream when a uniqueKeys key repeated,
	// so that each column of a composite key moves apart from the others.
	cycle := func(list []value.Value, filter func(value.Value) bool) (value.Value, bool) {
		n := uint64(len(list))
		if n == 0 {
			return nil, false
		}
		start := (s.opts.Seed%n + uint64(loc.k)%n) % n
		if loc.attempt != "" {
			start = (start + s.stream(loc.ptr, loc.attempt).Below(n)) % n
		}
		for i := range n {
			v := list[(start+i)%n]
			if filter(v) && keep(v) {
				return v, true
			}
		}
		return nil, false
	}
	if consts, has := c.consts(); has {
		if len(consts) == 1 && keep(consts[0]) {
			return consts[0], true
		}
		return nil, false
	}
	if ex, ok := c.first("examples"); ok {
		if a, ok := ex.(value.Array); ok {
			var list []value.Value
			for _, e := range a.All() {
				list = append(list, e)
			}
			if v, ok := cycle(list, whole); ok {
				return v, true
			}
		}
	}
	if d, ok := c.first("default"); ok && whole(d) && keep(d) {
		return d, true
	}
	if members, has := c.enum(); has {
		return cycle(members, func(value.Value) bool { return true })
	}
	return nil, false
}

// writable reports whether v can be written in the input's form so that
// its reader reads back the same value. No reader holds a noncharacter in
// a string (ADR-003 D8 item 4), and a form that normalizes line breaks
// holds no CR. An integral number of magnitude above 2^53−1 has no CSV
// field an integer column reads back as it, since ADR-003 D8 item 5
// refuses the integer literal, so the generator keeps none in any form.
func (s *Generator) writable(loc *place, v value.Value) bool {
	switch v := v.(type) {
	case value.Null:
		return !loc.cell
	case value.Number:
		return !(v.Integral() && math.Abs(v.Float64()) > value.MaxSafeInteger)
	case value.String:
		for _, r := range v.Text() {
			if r == '\r' && loc.noCR || 0xFDD0 <= r && r <= 0xFDEF || r&0xFFFE == 0xFFFE {
				return false
			}
		}
	case value.Array:
		if loc.cell {
			return false
		}
		sub := &place{noCR: loc.noCR}
		for _, e := range v.All() {
			if !s.writable(sub, e) {
				return false
			}
		}
	case value.Object:
		if loc.cell {
			return false
		}
		sub := &place{noCR: loc.noCR}
		for name, e := range v.All() {
			if !s.writable(sub, mustString(name)) || !s.writable(sub, e) {
				return false
			}
		}
	}
	return true
}

// chooseType chooses the type of the value at loc (item 14): from C's
// type set, or the type inferred when no schema of C declares one, the
// first in the order written, except that the optional mode decides null:
// All never chooses it, None always does, and Mixed chooses it when the
// type stream's first draw is odd. A CSV field holds no null, array or
// object.
func (s *Generator) chooseType(loc *place, c *conj) (string, bool) {
	types, declared := c.types()
	if !declared {
		types = []string{c.inferred()}
	}
	if loc.cell {
		types = slices.DeleteFunc(slices.Clone(types), func(t string) bool { return t == "null" || t == "array" || t == "object" })
	}
	if _, ok := loc.fixed.(value.Object); ok {
		// A supplied object is filled around (item 2); located takes one
		// C makes no object for as given.
		return "object", true
	}
	if len(types) == 0 {
		return "", false
	}
	nonNull := slices.DeleteFunc(slices.Clone(types), func(t string) bool { return t == "null" })
	if len(nonNull) == len(types) {
		return types[0], true
	}
	if len(nonNull) == 0 {
		return "null", true
	}
	switch s.opts.Optional {
	case None:
		return "null", true
	case Mixed:
		if s.stream(loc.ptr, purposeType).Next()%2 == 1 {
			return "null", true
		}
	}
	return nonNull[0], true
}

// declared returns the names C's properties declare, in declaration order
// (item 15): the order of first appearance walking C, and within one
// schema the order its file writes them in, by the file's position table.
// The names C requires and declares no property for follow, in the order
// written.
func (s *Generator) declared(c *conj) []string {
	var out []string
	seen := map[string]bool{}
	add := func(name string) {
		if !seen[name] {
			seen[name] = true
			out = append(out, name)
		}
	}
	for i, o := range c.objs {
		ps, ok := o.Get("properties")
		if !ok {
			continue
		}
		po, ok := ps.(value.Object)
		if !ok {
			continue
		}
		names := po.Names()
		var t finding.Positions
		if s.pos != nil {
			t = s.pos(c.nodes[i].entry)
		}
		where := func(name string) (finding.Pos, bool) {
			if t == nil {
				return finding.Pos{}, false
			}
			return t.Key(c.nodes[i].child("properties", name).ptr)
		}
		slices.SortStableFunc(names, func(a, b string) int {
			pa, oka := where(a)
			pb, okb := where(b)
			switch {
			case oka && okb:
				return cmp.Or(cmp.Compare(pa.Line, pb.Line), cmp.Compare(pa.Column, pb.Column))
			case oka:
				return -1
			case okb:
				return 1
			}
			return 0
		})
		for _, n := range names {
			add(n)
		}
	}
	c.each("required", func(_ node, v value.Value) {
		if a, ok := v.(value.Array); ok {
			for _, e := range a.All() {
				if n, ok := e.(value.String); ok {
					add(n.Text())
				}
			}
		}
	})
	return out
}

// propStarts returns the schemas C gives the member name: in each schema
// of C, its properties entry for the name and every patternProperties
// entry whose pattern the name matches, or its additionalProperties when
// neither applies.
func (s *Generator) propStarts(c *conj, name string) []node {
	var out []node
	for i, o := range c.objs {
		n := c.nodes[i]
		matched := false
		if ps, ok := o.Get("properties"); ok {
			if po, ok := ps.(value.Object); ok {
				if _, ok := po.Get(name); ok {
					out = append(out, n.child("properties", name))
					matched = true
				}
			}
		}
		if pp, ok := o.Get("patternProperties"); ok {
			if po, ok := pp.(value.Object); ok {
				for pat := range po.All() {
					if s.matches(pat, name) {
						out = append(out, n.child("patternProperties", pat))
						matched = true
					}
				}
			}
		}
		if !matched {
			if _, ok := o.Get("additionalProperties"); ok {
				out = append(out, n.child("additionalProperties"))
			}
		}
	}
	return out
}

// matches reports whether name holds a match of pattern, a pattern the
// compiler has accepted.
func (s *Generator) matches(pattern, name string) bool {
	re, err := s.pattern(pattern)
	if err != nil {
		return false
	}
	return re.MatchString(name)
}

// pattern is pattern compiled by the package profile, once.
func (s *Generator) pattern(pattern string) (*regexprofile.Regexp, error) {
	if re, ok := s.patterns[pattern]; ok {
		return re, nil
	}
	re, err := regexprofile.Compile(pattern, s.lim, nil)
	if err != nil {
		return nil, err
	}
	if s.patterns == nil {
		s.patterns = map[string]*regexprofile.Regexp{}
	}
	s.patterns[pattern] = re
	return re, nil
}

// forbidden reports whether any schema of starts is false.
func (s *Generator) forbidden(starts []node) bool {
	for _, n := range starts {
		if v, ok := s.c.Lookup(n.entry, n.ptr); ok && v == value.Bool(false) {
			return true
		}
	}
	return false
}

// required returns the names C requires of obj: its required lists, the
// dependentRequired of every trigger obj holds, and a Markdown document's
// body.
func (s *Generator) required(loc *place, c *conj, obj map[string]value.Value) map[string]bool {
	out := map[string]bool{}
	names := func(v value.Value) {
		if a, ok := v.(value.Array); ok {
			for _, e := range a.All() {
				if n, ok := e.(value.String); ok {
					out[n.Text()] = true
				}
			}
		}
	}
	c.each("required", func(_ node, v value.Value) { names(v) })
	c.each("dependentRequired", func(_ node, v value.Value) {
		if o, ok := v.(value.Object); ok {
			for trigger, deps := range o.All() {
				if _, present := obj[trigger]; present {
					names(deps)
				}
			}
		}
	})
	if loc.markdown {
		out["body"] = true
	}
	return out
}

// key is a stable text for a list of schemas, which says whether a member
// must be made again.
func key(ns []node) string {
	var b strings.Builder
	for _, n := range ns {
		b.WriteString(n.entry)
		b.WriteByte('#')
		b.WriteString(n.ptr)
		b.WriteByte(0)
	}
	return b.String()
}

// object makes an object from C (item 15), settling its conditionals and
// dependencies (item 9): its properties are made, the if of every schema
// of C is evaluated against it, and the branches selected and the
// dependencies its members trigger are added to C, whose members newly
// required or constrained are made again, until nothing changes, at most
// maxPasses times.
func (s *Generator) object(loc *place, base *conj) (value.Value, bool) {
	fixed, _ := loc.fixed.(value.Object)
	members := map[string]value.Value{}
	made := map[string]string{} // the key each member was made, or failed to be made, from
	var extra []node
	var lastRequired map[string]bool
	ok := true
	for pass := 0; pass < maxPasses; pass++ {
		c := base
		if len(extra) > 0 {
			c = s.with(loc, base, extra)
		}
		required := s.required(loc, c, members)
		next := map[string]value.Value{}
		var optional, omitted []string
		ok = true
		gen := func(name string, starts []node) {
			child := s.memberPlace(loc, name)
			k := key(starts)
			prev, tried := made[name]
			if v, had := members[name]; had && tried && prev == k {
				next[name] = v
				return
			}
			if _, had := members[name]; !had && tried && prev == k {
				// It failed from these schemas before, and was reported.
				ok = false
				return
			}
			made[name] = k
			if v, good := s.value(child, starts); good {
				next[name] = v
			} else {
				ok = false
			}
		}
		order := s.declared(c)
		for _, name := range slices.Sorted(maps.Keys(required)) {
			if !slices.Contains(order, name) {
				order = append(order, name)
			}
		}
		for _, name := range order {
			starts := s.propStarts(c, name)
			req := required[name]
			if fv, has := fixedMember(fixed, name); has {
				child := s.memberPlace(loc, name)
				child.fixed = fv
				if v, good := s.value(child, starts); good {
					next[name] = v
				} else {
					ok = false
				}
				continue
			}
			if s.forbidden(starts) {
				if req {
					s.unsatisfiable(s.memberPlace(loc, name), s.holder(c, "required"), ReasonType)
					ok = false
				}
				continue
			}
			if !req && (loc.stop || !s.include(s.memberPlace(loc, name).ptr)) {
				omitted = append(omitted, name)
				continue
			}
			gen(name, starts)
			if !req {
				optional = append(optional, name)
			}
		}
		for name, fv := range fixed.All() {
			if _, has := next[name]; !has {
				next[name] = fv
			}
		}
		if !s.sized(loc, c, next, gen, &optional, omitted) {
			ok = false
		}
		members = next
		obj := objectOf(members)
		sel := s.selected(c, obj)
		reqNow := s.required(loc, c, members)
		if sameNodes(sel, extra) && sameSet(reqNow, required) && (lastRequired == nil || sameSet(reqNow, lastRequired)) {
			return obj, ok
		}
		extra, lastRequired = sel, reqNow
	}
	s.unsatisfiable(loc, base.starts[0], ReasonConditionals)
	return nil, false
}

// sized brings next within C's minProperties and maxProperties (item 15).
// Below the minimum it adds the omitted optional properties in order,
// then names generated from a patternProperties pattern, then, where an
// additionalProperties schema other than false admits them, key1, key2, …
// skipping taken names; every added name satisfies propertyNames. Above
// the maximum it drops optional properties from the end.
func (s *Generator) sized(loc *place, c *conj, next map[string]value.Value, gen func(string, []node), optional *[]string, omitted []string) bool {
	minP, hasMin := c.count("minProperties", true)
	maxP, hasMax := c.count("maxProperties", false)
	if hasMin && hasMax && minP > maxP {
		s.unsatisfiable(loc, s.holder(c, "minProperties"), ReasonLength)
		return false
	}
	nameOK := func(name string) bool {
		if _, taken := next[name]; taken {
			return false
		}
		return s.valid(c.nodesAt("propertyNames"), mustString(name))
	}
	add := func(name string) {
		if nameOK(name) && !s.forbidden(s.propStarts(c, name)) {
			gen(name, s.propStarts(c, name))
			if _, made := next[name]; made {
				*optional = append(*optional, name)
			}
		}
	}
	if hasMin && !s.affords(loc, minP-int64(len(next))) {
		return false
	}
	if hasMin {
		for _, name := range omitted {
			if int64(len(next)) >= minP {
				break
			}
			add(name)
		}
		names := s.stream(loc.ptr, purposeName)
		for i, o := range c.objs {
			pp, ok := o.Get("patternProperties")
			if !ok {
				continue
			}
			po, ok := pp.(value.Object)
			if !ok {
				continue
			}
			_ = i
			for pat := range po.All() {
				for range maxAttempts {
					if int64(len(next)) >= minP {
						break
					}
					if name, ok := s.patternText(pat, names, 0, -1); ok {
						add(name)
					}
				}
			}
		}
		open := false
		c.each("additionalProperties", func(_ node, v value.Value) { open = open || v != value.Bool(false) })
		for i := 1; open && int64(len(next)) < minP && i <= int(minP)+maxAttempts; i++ {
			add("key" + itoa(i))
		}
		if int64(len(next)) < minP {
			s.unsatisfiable(loc, s.holder(c, "minProperties"), ReasonLength)
			return false
		}
	}
	if hasMax {
		for int64(len(next)) > maxP && len(*optional) > 0 {
			last := (*optional)[len(*optional)-1]
			*optional = (*optional)[:len(*optional)-1]
			delete(next, last)
		}
		if int64(len(next)) > maxP {
			s.unsatisfiable(loc, s.holder(c, "maxProperties"), ReasonLength)
			return false
		}
	}
	return true
}

// memberPlace is the place of the member name of the object at loc.
func (s *Generator) memberPlace(loc *place, name string) *place {
	child := loc.child(name)
	child.cell = loc.record
	switch {
	case loc.markdown && name == "body":
		child.prose = proseMarkdown
	case loc.record:
	case name == "body" && loc.item:
		child.prose = proseText
	}
	return child
}

// fixedMember returns the supplied value of the member name, when one was
// supplied.
func fixedMember(fixed value.Object, name string) (value.Value, bool) {
	if fixed.Len() == 0 {
		return nil, false
	}
	return fixed.Get(name)
}

// objectOf is the object holding members.
func objectOf(members map[string]value.Value) value.Object {
	ms := make([]value.Member, 0, len(members))
	for n, v := range members {
		ms = append(ms, value.Member{Name: n, Value: v})
	}
	o, err := value.NewObject(ms...)
	if err != nil {
		// A map's keys are unique, and every value is the model's.
		panic(err)
	}
	return o
}

// mustString is s as a String; every name the generator holds is valid
// UTF-8.
func mustString(s string) value.String {
	v, err := value.NewString(s)
	if err != nil {
		panic(err)
	}
	return v
}

func sameNodes(a, b []node) bool { return key(a) == key(b) }

func sameSet(a, b map[string]bool) bool {
	if len(a) != len(b) {
		return false
	}
	for k := range a {
		if !b[k] {
			return false
		}
	}
	return true
}

// itemStarts returns the schemas C gives the array element at i: in each
// schema of C, its prefixItems entry when it has one for i, and its items
// otherwise.
func (s *Generator) itemStarts(c *conj, i int) []node {
	var out []node
	for j, o := range c.objs {
		n := c.nodes[j]
		if p, ok := o.Get("prefixItems"); ok {
			if a, ok := p.(value.Array); ok && i < a.Len() {
				out = append(out, n.child("prefixItems", itoa(i)))
				continue
			}
		}
		if _, ok := o.Get("items"); ok {
			out = append(out, n.child("items"))
		}
	}
	return out
}

// array makes an array from C (item 16). Its length is --items, or the
// prefixItems length when C has one, clamped to minItems and maxItems,
// where maxItems is lowered to the prefixItems length when items is false;
// past the recursion bound it is minItems. With coverage, the first items
// each hold one member of values at the key, in values' order, and the
// length is at least their count (ADR-003 Amendment A2 item 17). The
// first minContains items are made from contains as well, and a later one
// that would push the count past maxContains is made again. A duplicate in
// a uniqueItems array, and an item whose key repeats under uniqueKeys, is
// made again from attempt streams, and dropped when they are spent, while
// the array stays at its minimum length.
func (s *Generator) array(loc *place, c *conj) (value.Value, bool) {
	prefix, capItems := 0, int64(-1)
	for _, o := range c.objs {
		plen := 0
		if p, ok := o.Get("prefixItems"); ok {
			if a, ok := p.(value.Array); ok {
				plen = a.Len()
			}
		}
		prefix = max(prefix, plen)
		if it, ok := o.Get("items"); ok && it == value.Bool(false) {
			if capItems < 0 || int64(plen) < capItems {
				capItems = int64(plen)
			}
		}
	}
	minI, _ := c.count("minItems", true)
	maxI, hasMax := c.count("maxItems", false)
	if capItems >= 0 && (!hasMax || capItems < maxI) {
		maxI, hasMax = capItems, true
	}
	if hasMax && minI > maxI {
		s.unsatisfiable(loc, s.holder(c, "minItems"), ReasonLength)
		return nil, false
	}
	covered := c.covered()
	if hasMax && int64(len(covered)) > maxI {
		s.unsatisfiable(loc, s.holder(c, "coverage"), ReasonCoverage)
		return nil, false
	}
	target := int64(s.opts.Items)
	if prefix > 0 {
		target = int64(prefix)
	}
	if loc.stop {
		target = minI
	}
	n := max(target, minI)
	if hasMax {
		n = min(n, maxI)
	}
	n = max(n, int64(len(covered)))
	if !s.affords(loc, n) {
		return nil, false
	}
	contains := c.nodesAt("contains")
	minC, maxC, hasMaxC := int64(0), int64(0), false
	if len(contains) > 0 {
		minC = 1
		if m, ok := c.count("minContains", true); ok {
			minC = m
		}
		maxC, hasMaxC = c.count("maxContains", false)
	}
	unique := false
	c.each("uniqueItems", func(_ node, v value.Value) { unique = unique || v == value.Bool(true) })
	keys := c.uniqueKeys()
	repeats := func(v value.Value) bool {
		return slices.ContainsFunc(keys, func(k *keyed) bool { return k.holds(v) })
	}
	var items []value.Value
	ok := true
	counted := int64(0)
	dropped := ReasonUniqueItems
	for i := int64(0); i < n; i++ {
		starts := s.itemStarts(c, int(i))
		if i < minC {
			starts = append(starts, contains...)
		}
		// place is the item's place, regenerated under attempt k when k
		// is above 0: holding its coverage member, and steered off the
		// values earlier items hold at a key of one pointer.
		place := func(k int) *place {
			p := s.itemPlace(loc, int(i))
			if i < int64(len(covered)) {
				p.fixed = covered[i].item()
			}
			if unique {
				p.earlier = items
			}
			for _, k := range keys {
				if len(k.key) == 1 {
					p.avoid = append(p.avoid, avoided{k.key[0], k.held})
				}
			}
			if k > 0 {
				p.purpose = attempt(k)
				if len(keys) > 0 {
					p.attempt = p.purpose
				}
			}
			return p
		}
		mark := len(s.findings)
		v, good := s.value(place(0), starts)
		if !good && (unique || len(keys) > 0) && !s.stopped() {
			// No value but one equal to an earlier item, or holding an
			// earlier item's key: the item is a duplicate, not
			// unsatisfiable.
			s.findings = s.findings[:mark]
			free := s.itemPlace(loc, int(i))
			if i < int64(len(covered)) {
				free.fixed = covered[i].item()
			}
			v, good = s.value(free, starts)
		}
		if !good {
			ok = false
			continue
		}
		clash := func(v value.Value) bool {
			if unique && slices.ContainsFunc(items, func(e value.Value) bool { return value.Equal(e, v) }) {
				return true
			}
			if repeats(v) {
				return true
			}
			return i >= minC && hasMaxC && counted >= maxC && s.matchesAny(contains, v)
		}
		for k := 1; clash(v) && k <= maxAttempts && !s.stopped(); k++ {
			mark := len(s.findings)
			if w, good := s.value(place(k), starts); good {
				v = w
			} else {
				s.findings = s.findings[:mark]
			}
		}
		if clash(v) {
			if repeats(v) {
				dropped = ReasonUniqueKeys
			}
			if i < int64(len(covered)) {
				// An item holding a coverage member is never dropped.
				s.unsatisfiable(loc, s.holder(c, "uniqueKeys"), ReasonUniqueKeys)
				ok = false
			}
			// Dropped: the array is shortened, and below minItems it is
			// unsatisfiable.
			continue
		}
		if len(contains) > 0 && (i < minC || s.matchesAny(contains, v)) {
			counted++
		}
		items = append(items, v)
		for _, k := range keys {
			k.add(v)
		}
	}
	if ok && int64(len(items)) < minI {
		keyword := "uniqueItems"
		if dropped == ReasonUniqueKeys {
			keyword = "uniqueKeys"
		}
		s.unsatisfiable(loc, s.holder(c, keyword), dropped)
		return nil, false
	}
	a, err := value.NewArray(items...)
	if err != nil {
		panic(err)
	}
	return a, ok
}

// cover is one member of a coverage's values, and the key it is held at.
type cover struct {
	key    []string
	member value.Value
}

// item is the fixed point of the item that holds the member: an object
// holding it at the key, which the item's other fields are generated
// around (item 2).
func (cv cover) item() value.Value {
	v := cv.member
	for i := len(cv.key) - 1; i >= 0; i-- {
		o, err := value.NewObject(value.Member{Name: cv.key[i], Value: v})
		if err != nil {
			// A pointer's tokens are valid UTF-8, and one member is unique.
			panic(err)
		}
		v = o
	}
	return v
}

// covered returns the members of every coverage of C, each with its key,
// in walk order and each coverage's values in order, a member a coverage
// before it already holds at the same key left out (ADR-003 Amendment A2
// item 17).
func (c *conj) covered() []cover {
	var out []cover
	seen := map[string]bool{}
	c.each("coverage", func(_ node, v value.Value) {
		o, ok := v.(value.Object)
		if !ok {
			return
		}
		kv, _ := o.Get("key")
		ks, _ := kv.(value.String)
		key := pointerTokens(ks.Text())
		vs, _ := o.Get("values")
		values, _ := vs.(value.Array)
		for _, m := range values.All() {
			// A key pointer's text ends where the member's digest, of a
			// fixed length, begins.
			if d := digest(m); !seen[ks.Text()+d] {
				seen[ks.Text()+d] = true
				out = append(out, cover{key, m})
			}
		}
	})
	return out
}

// keyed is one uniqueKeys of C: its key pointers' tokens, and the keys
// the items made so far hold under it, each by its encoding.
type keyed struct {
	key  [][]string
	held map[string]bool
}

// uniqueKeys returns every uniqueKeys of C, in walk order, none of whose
// keys is yet held.
func (c *conj) uniqueKeys() []*keyed {
	var out []*keyed
	c.each("uniqueKeys", func(_ node, v value.Value) {
		a, ok := v.(value.Array)
		if !ok {
			return
		}
		k := &keyed{held: map[string]bool{}}
		for _, p := range a.All() {
			if ps, ok := p.(value.String); ok {
				k.key = append(k.key, pointerTokens(ps.Text()))
			}
		}
		out = append(out, k)
	})
	return out
}

// of returns v's key under k, encoded so that two keys are encoded alike
// exactly when MTS-DAT-018 has them equal: pointer by pointer, by the
// equality of MTS-DAT-013. ok is false when a pointer resolves to nothing
// within v, which then has no key.
func (k *keyed) of(v value.Value) (string, bool) {
	var b strings.Builder
	for _, toks := range k.key {
		x, ok := at(v, toks)
		if !ok {
			return "", false
		}
		b.WriteString(digest(x))
	}
	return b.String(), true
}

// holds reports whether v's key under k is one an earlier item holds.
func (k *keyed) holds(v value.Value) bool {
	e, ok := k.of(v)
	return ok && k.held[e]
}

// add records v's key under k as held, when v has one.
func (k *keyed) add(v value.Value) {
	if e, ok := k.of(v); ok {
		k.held[e] = true
	}
}

// digest is the SHA-256 of v's canonical JSON, as text: two values have
// the same digest exactly when they are equal, a number by its value and
// an object whatever its members' order.
func digest(v value.Value) string {
	d := jsonx.CanonicalDigest(v)
	return string(d[:])
}

// pointerTokens is a key pointer's tokens; the schema compiler has held
// it to RFC 6901.
func pointerTokens(p string) []string {
	path, ok := finding.ParsePath(p)
	if !ok {
		return nil
	}
	return path.Tokens()
}

// at returns the value the tokens name within v, as RFC 6901 evaluates a
// pointer, and false when they name nothing.
func at(v value.Value, toks []string) (value.Value, bool) {
	for _, t := range toks {
		switch c := v.(type) {
		case value.Object:
			m, ok := c.Get(t)
			if !ok {
				return nil, false
			}
			v = m
		case value.Array:
			i, err := strconv.Atoi(t)
			if err != nil || i < 0 || i >= c.Len() || itoa(i) != t {
				return nil, false
			}
			v = c.At(i)
		default:
			return nil, false
		}
	}
	return v, true
}

// itemPlace is the place of the element at i of the array at loc.
func (s *Generator) itemPlace(loc *place, i int) *place {
	child := loc.child(itoa(i))
	child.k = i
	switch {
	case loc.collection:
		child.item = true
		s.input.fileAt(child)
	case loc.table:
		child.record = true
	}
	return child
}
