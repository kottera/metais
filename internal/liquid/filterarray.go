// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package liquid

import (
	"math"
	"math/bits"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/kottera/metais/internal/liquid/value"
)

// The array filters, and default. Most read their input as Shopify's
// InputIterator does (iterate): an array flattened, an object or any
// other value as one element, nil as none. Those that read an element's
// property read it as property does: a member of an object, a substring
// of a string; an element that is nil or a boolean cannot be indexed, and
// a number refuses the property. An object is iterated in canonical
// order, as a two-element array of its name and value (ADR-004 D4
// item 5).

func arrayFilters() []*stdFilter {
	none := signature{}
	optional := signature{max: 1}
	selecting := signature{min: 1, max: 2}
	return []*stdFilter{
		{name: "size", sig: none, fn: size},
		{name: "first", sig: none, fn: func(c *call, in value.Value) (value.Value, *Error) { return end(c, in, true) }},
		{name: "last", sig: none, fn: func(c *call, in value.Value) (value.Value, *Error) { return end(c, in, false) }},
		{name: "join", sig: optional, fn: join},
		{name: "reverse", sig: none, fn: reverse},
		{name: "concat", sig: signature{min: 1, max: 1, literal: arrayArg}, fn: concat},
		{name: "compact", sig: optional, fn: compact},
		{name: "uniq", sig: optional, fn: uniq},
		{name: "map", sig: signature{min: 1, max: 1}, fn: mapFilter},
		{name: "sort", sig: optional, fn: sortFilter(false)},
		{name: "sort_natural", sig: optional, fn: sortFilter(true)},
		{name: "slice", sig: signature{min: 1, max: 2, literal: sliceArg}, fn: slice},
		{name: "where", sig: selecting, fn: selectFilter(selectWhere)},
		{name: "reject", sig: selecting, fn: selectFilter(selectReject)},
		{name: "has", sig: selecting, fn: selectFilter(selectHas)},
		{name: "find", sig: selecting, fn: selectFilter(selectFind)},
		{name: "find_index", sig: selecting, fn: selectFilter(selectFindIndex)},
		{name: "default", sig: signature{max: 1, keywords: []string{"allow_false"}}, fn: defaultFilter},
	}
}

// size is a string's number of characters, an array's elements or an
// object's members; anything else has size 0.
func size(c *call, in value.Value) (value.Value, *Error) {
	var n int
	switch in := in.(type) {
	case value.String:
		n = utf8.RuneCountInString(in.Text())
	case value.Array:
		n = in.Len()
	case value.Object:
		n = in.Len()
	}
	return tnum{isInt: true, i: int64(n)}.value(c.name)
}

// pair is an object member as iteration gives it: [name, value].
func pair(name string, v value.Value) value.Value {
	return newArray([]value.Value{newString(name), v})
}

// end is first and last: a string's first or last character, an array's
// element, an object's member as [name, value] in canonical order; nil
// when there is none, and for anything else.
func end(c *call, in value.Value, first bool) (value.Value, *Error) {
	switch in := in.(type) {
	case value.String:
		s := in.Text()
		if s == "" {
			return value.Null{}, nil
		}
		if first {
			_, n := utf8.DecodeRuneInString(s)
			return newString(s[:n]), nil
		}
		_, n := utf8.DecodeLastRuneInString(s)
		return newString(s[len(s)-n:]), nil
	case value.Array:
		if in.Len() == 0 {
			return value.Null{}, nil
		}
		if first {
			return in.At(0), nil
		}
		return in.At(in.Len() - 1), nil
	case value.Object:
		names := in.Names()
		if len(names) == 0 {
			return value.Null{}, nil
		}
		name := names[len(names)-1]
		if first {
			name = names[0]
		}
		v, _ := in.Get(name)
		return pair(name, v), nil
	}
	return value.Null{}, nil
}

// join writes the texts of the input's elements, separated by the
// argument's text, a space when it is omitted.
func join(c *call, in value.Value) (value.Value, *Error) {
	items, err := c.iterate(in)
	if err != nil {
		return nil, err
	}
	glue := " "
	if a, ok := c.arg(0); ok {
		if glue, err = c.str(a); err != nil {
			return nil, err
		}
	}
	texts := make([]string, len(items))
	n := 0
	for i, it := range items {
		if texts[i], err = c.str(it); err != nil {
			return nil, err
		}
		n += len(texts[i])
	}
	if len(items) > 1 {
		n += (len(items) - 1) * len(glue)
	}
	return c.built(n, func() string { return strings.Join(texts, glue) })
}

// array charges a result of len(elems) elements, then returns it.
func (c *call) array(elems []value.Value) (value.Value, *Error) {
	if err := c.result(int64(len(elems))); err != nil {
		return nil, err
	}
	return newArray(elems), nil
}

func reverse(c *call, in value.Value) (value.Value, *Error) {
	items, err := c.iterate(in)
	if err != nil {
		return nil, err
	}
	slices.Reverse(items)
	return c.array(items)
}

// arrayArg refuses, at parse, a literal argument: no literal is an array.
func arrayArg(filter string, arg int, _ string, v value.Value) *Error {
	if _, ok := v.(value.Array); ok {
		return nil
	}
	return filterErr(filter, CodeFilterArgument, map[string]string{"argument": strconv.Itoa(arg), "type": kindName(v)},
		"the argument is not an array")
}

// concat is the input's elements followed by the argument's, which must
// be an array; the argument's own nested arrays stay as they are.
func concat(c *call, in value.Value) (value.Value, *Error) {
	a, _ := c.arg(0)
	tail, ok := a.(value.Array)
	if !ok {
		return nil, arrayArg(c.name, 1, "", a)
	}
	items, err := c.iterate(in)
	if err != nil {
		return nil, err
	}
	if err := c.result(int64(len(items) + tail.Len())); err != nil {
		return nil, err
	}
	for _, e := range tail.All() {
		items = append(items, e)
	}
	return newArray(items), nil
}

// propertyArg returns the property a filter's first argument names, and
// false when it is omitted or nil: then the filter reads the elements
// themselves.
func (c *call) propertyArg() (value.Value, bool) {
	p, ok := c.arg(0)
	if !ok {
		return nil, false
	}
	if _, isNull := p.(value.Null); isNull {
		return nil, false
	}
	return p, true
}

// keys returns each item's property prop, or false when an item cannot
// be indexed: then the filter gives nil, as Shopify's do.
func (c *call) keys(items []value.Value, prop value.Value) ([]value.Value, bool, *Error) {
	keys := make([]value.Value, len(items))
	for i, it := range items {
		v, how := property(it, prop)
		switch how {
		case noIndex:
			return nil, false, nil
		case badIndex:
			return nil, false, inputErr(c.name, it, "an element cannot be indexed by the property")
		}
		keys[i] = v
	}
	return keys, true, nil
}

// compact drops the elements that are nil, or with a property those whose
// property is nil.
func compact(c *call, in value.Value) (value.Value, *Error) {
	items, err := c.iterate(in)
	if err != nil {
		return nil, err
	}
	keys := items
	if prop, ok := c.propertyArg(); ok {
		var indexable bool
		if keys, indexable, err = c.keys(items, prop); err != nil || !indexable {
			return value.Null{}, err
		}
	}
	var out []value.Value
	for i, it := range items {
		if _, isNull := keys[i].(value.Null); !isNull {
			out = append(out, it)
		}
	}
	return c.array(out)
}

// uniq keeps the first of the elements that are the same, or with a
// property whose property values are the same. Values are the same when
// they are equal and of the same kind, so 1 and 1.0 differ, as Ruby's
// eql? has them.
func uniq(c *call, in value.Value) (value.Value, *Error) {
	items, err := c.iterate(in)
	if err != nil {
		return nil, err
	}
	keys := items
	if prop, ok := c.propertyArg(); ok {
		var indexable bool
		if keys, indexable, err = c.keys(items, prop); err != nil || !indexable {
			return value.Null{}, err
		}
	}
	var work int64
	for _, k := range keys {
		work += deepSize(k)
	}
	if err := c.units(work); err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	var out []value.Value
	for i, it := range items {
		var b strings.Builder
		identity(&b, keys[i])
		if k := b.String(); !seen[k] {
			seen[k] = true
			out = append(out, it)
		}
	}
	return c.array(out)
}

// identity writes a text that two values share exactly when they are
// equal and of the same kinds throughout.
func identity(b *strings.Builder, v value.Value) {
	switch v := v.(type) {
	case value.Null:
		b.WriteByte('z')
	case value.Bool:
		if v {
			b.WriteByte('t')
		} else {
			b.WriteByte('f')
		}
	case value.Number:
		n := numberKind(v)
		if n.isInt {
			b.WriteString("i" + strconv.FormatInt(n.i, 10) + ";")
		} else {
			b.WriteString("d" + strconv.FormatFloat(n.f+0, 'g', -1, 64) + ";")
		}
	case value.String:
		b.WriteString("s" + strconv.Itoa(len(v.Text())) + ":" + v.Text())
	case value.Array:
		b.WriteString("a" + strconv.Itoa(v.Len()) + ":")
		for _, e := range v.All() {
			identity(b, e)
		}
	case value.Object:
		b.WriteString("o" + strconv.Itoa(v.Len()) + ":")
		for name, e := range v.All() {
			identity(b, newString(name))
			identity(b, e)
		}
	}
}

// mapFilter is each element's property: nil for an element that cannot
// be indexed.
func mapFilter(c *call, in value.Value) (value.Value, *Error) {
	items, err := c.iterate(in)
	if err != nil {
		return nil, err
	}
	prop, _ := c.arg(0)
	out := make([]value.Value, len(items))
	for i, it := range items {
		v, how := property(it, prop)
		switch how {
		case noIndex:
			v = value.Null{}
		case badIndex:
			return nil, inputErr(c.name, it, "an element cannot be indexed by the property")
		}
		out[i] = v
	}
	return c.array(out)
}

// sortFilter is sort and sort_natural. Both are stable, and put nil keys
// last. sort orders numbers by value and strings by code point (ADR-004
// D4 item 6), and refuses keys that mix them or hold anything else with
// CodeFilterInput. sort_natural orders the keys' texts by code point
// with ASCII letters folded to lower case, as Ruby's casecmp does.
func sortFilter(natural bool) func(*call, value.Value) (value.Value, *Error) {
	return func(c *call, in value.Value) (value.Value, *Error) {
		items, err := c.iterate(in)
		if err != nil {
			return nil, err
		}
		keys := items
		if prop, ok := c.propertyArg(); ok {
			var indexable bool
			if keys, indexable, err = c.keys(items, prop); err != nil || !indexable {
				return value.Null{}, err
			}
		}
		n := len(items)
		var cmp func(a, b int) int
		if natural {
			cmp, err = naturalOrder(c, keys)
		} else {
			cmp, err = valueOrder(c, keys)
		}
		if err != nil {
			return nil, err
		}
		order := make([]int, n)
		for i := range order {
			order[i] = i
		}
		slices.SortStableFunc(order, func(a, b int) int {
			ka, kb := keys[a], keys[b]
			_, na := ka.(value.Null)
			_, nb := kb.(value.Null)
			switch {
			case na && nb:
				return 0
			case na:
				return 1
			case nb:
				return -1
			}
			return cmp(a, b)
		})
		out := make([]value.Value, n)
		for i, j := range order {
			out[i] = items[j]
		}
		return c.array(out)
	}
}

// sortCharge charges sorting n keys whose strings hold strBytes bytes:
// n·⌈log2 n⌉ comparisons, and each byte ⌈log2 n⌉ times.
func (c *call) sortCharge(n int, strBytes int64) *Error {
	if n < 2 {
		return nil
	}
	lg := int64(bits.Len(uint(n - 1)))
	return c.units(int64(n)*lg + strBytes*lg)
}

// valueOrder returns sort's comparison of keys, refusing keys it cannot
// order.
func valueOrder(c *call, keys []value.Value) (func(a, b int) int, *Error) {
	var nums, strs, others int
	var strBytes int64
	var other value.Value
	for _, k := range keys {
		switch k := k.(type) {
		case value.Null:
		case value.Number:
			nums++
		case value.String:
			strs++
			strBytes += int64(len(k.Text()))
		default:
			others++
			other = k
		}
	}
	if nums+strs+others > 1 && (others > 0 || (nums > 0 && strs > 0)) {
		bad := other
		if bad == nil {
			bad = value.String{}
		}
		return nil, inputErr(c.name, bad, "the values cannot be ordered against each other")
	}
	if err := c.sortCharge(len(keys), strBytes); err != nil {
		return nil, err
	}
	if strs > 0 {
		return func(a, b int) int {
			return strings.Compare(keys[a].(value.String).Text(), keys[b].(value.String).Text())
		}, nil
	}
	nk := make([]tnum, len(keys))
	for i, k := range keys {
		if n, ok := k.(value.Number); ok {
			nk[i] = numberKind(n)
		}
	}
	return func(a, b int) int { return nk[a].cmp(nk[b]) }, nil
}

// naturalOrder returns sort_natural's comparison of keys: of their texts,
// with ASCII letters folded to lower case.
func naturalOrder(c *call, keys []value.Value) (func(a, b int) int, *Error) {
	texts := make([]string, len(keys))
	var strBytes int64
	for i, k := range keys {
		if _, ok := k.(value.Null); ok {
			continue
		}
		s, err := c.str(k)
		if err != nil {
			return nil, err
		}
		texts[i] = s
		strBytes += int64(len(s))
	}
	if err := c.sortCharge(len(keys), strBytes); err != nil {
		return nil, err
	}
	return func(a, b int) int { return foldCompare(texts[a], texts[b]) }, nil
}

// foldCompare compares a and b byte by byte with A to Z read as a to z.
func foldCompare(a, b string) int {
	fold := func(c byte) byte {
		if 'A' <= c && c <= 'Z' {
			return c + 'a' - 'A'
		}
		return c
	}
	for i := 0; i < len(a) && i < len(b); i++ {
		if x, y := fold(a[i]), fold(b[i]); x != y {
			if x < y {
				return -1
			}
			return 1
		}
	}
	switch {
	case len(a) < len(b):
		return -1
	case len(a) > len(b):
		return 1
	}
	return 0
}

// sliceArg refuses, at parse, a literal offset or length that toInteger
// refuses. A length of nil or false is omitted.
func sliceArg(filter string, arg int, keyword string, v value.Value) *Error {
	if keyword != "" {
		return nil
	}
	if arg == 2 && !truthy(v) {
		return nil
	}
	_, err := toInteger(filter, strconv.Itoa(arg), v)
	return err
}

// slice is the part of an array, or of the input's text, that starts at
// the first argument's offset and holds the second argument's number of
// elements or characters, 1 when it is omitted, nil or false. A negative
// offset counts from the end. An offset before the start or past the
// end, or a negative length, gives the empty array or string.
func slice(c *call, in value.Value) (value.Value, *Error) {
	a, _ := c.arg(0)
	off, err := toInteger(c.name, "1", a)
	if err != nil {
		return nil, err
	}
	n := int64(1)
	if l, ok := c.arg(1); ok && truthy(l) {
		if n, err = toInteger(c.name, "2", l); err != nil {
			return nil, err
		}
	}
	span := func(size int) (int, int, bool) {
		start := off
		if start < 0 {
			start += int64(size)
		}
		if start < 0 || start > int64(size) || n < 0 {
			return 0, 0, false
		}
		return int(start), int(min(start+n, int64(size))), true
	}
	if arr, ok := in.(value.Array); ok {
		i, j, ok := span(arr.Len())
		var out []value.Value
		for k := i; ok && k < j; k++ {
			out = append(out, arr.At(k))
		}
		return c.array(out)
	}
	s, err := c.str(in)
	if err != nil {
		return nil, err
	}
	i, j, ok := span(utf8.RuneCountInString(s))
	if !ok {
		return value.String{}, nil
	}
	from, to, k := len(s), len(s), 0
	for b := range s {
		if k == i {
			from = b
		}
		if k == j {
			to = b
			break
		}
		k++
	}
	return newString(s[from:to]), nil
}

// What selectFilter does with the elements that match.
const (
	selectWhere = iota
	selectReject
	selectHas
	selectFind
	selectFindIndex
)

// selectFilter is where, reject, has, find and find_index. An element
// matches when its property, the first argument, equals the second
// argument, or, when that is omitted or nil, when the property is truthy.
// where keeps the matches and reject the rest; has is whether one
// matches; find is the first match and find_index its index, nil when
// none does. An element that cannot be indexed makes the result nil, as
// it does in Shopify's, unless a match ends the search before it.
func selectFilter(mode int) func(*call, value.Value) (value.Value, *Error) {
	return func(c *call, in value.Value) (value.Value, *Error) {
		items, err := c.iterate(in)
		if err != nil {
			return nil, err
		}
		if len(items) == 0 {
			switch mode {
			case selectHas:
				return value.Bool(false), nil
			case selectFind, selectFindIndex:
				return value.Null{}, nil
			}
			return c.array(nil)
		}
		prop, _ := c.arg(0)
		target, equality := c.arg(1)
		if _, isNull := target.(value.Null); isNull {
			equality = false
		}
		if k := target; equality && (k.Kind() == value.KindArray || k.Kind() == value.KindObject) {
			// Comparing an element with an array or object costs at most
			// the target's deep size.
			work := min(deepSize(target), math.MaxInt64/int64(len(items)))
			if err := c.units(work * int64(len(items))); err != nil {
				return nil, err
			}
		}
		var out []value.Value
		for i, it := range items {
			v, how := property(it, prop)
			switch how {
			case noIndex:
				return value.Null{}, nil
			case badIndex:
				return nil, inputErr(c.name, it, "an element cannot be indexed by the property")
			}
			match := truthy(v)
			if equality {
				match = value.Equal(v, target)
			}
			switch {
			case mode == selectHas && match:
				return value.Bool(true), nil
			case mode == selectFind && match:
				return it, nil
			case mode == selectFindIndex && match:
				return tnum{isInt: true, i: int64(i)}.value(c.name)
			case mode == selectWhere && match, mode == selectReject && !match:
				out = append(out, it)
			}
		}
		switch mode {
		case selectHas:
			return value.Bool(false), nil
		case selectFind, selectFindIndex:
			return value.Null{}, nil
		}
		return c.array(out)
	}
}

// defaultFilter is the argument ("" when omitted) when the input is nil,
// false, or an empty string, array or object, and the input otherwise.
// With allow_false truthy, false is kept.
func defaultFilter(c *call, in value.Value) (value.Value, *Error) {
	use := !truthy(in)
	if af, ok := c.kw["allow_false"]; ok && truthy(af) {
		_, use = in.(value.Null)
	}
	switch in := in.(type) {
	case value.String:
		use = in.Text() == ""
	case value.Array:
		use = in.Len() == 0
	case value.Object:
		use = in.Len() == 0
	}
	if !use {
		return in, nil
	}
	if d, ok := c.arg(0); ok {
		return d, nil
	}
	return value.String{}, nil
}
