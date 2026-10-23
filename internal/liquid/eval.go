// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package liquid

import (
	"fmt"
	"math"
	"strings"
	"unicode/utf8"

	"metais.org/metais/internal/liquid/ast"
	"metais.org/metais/internal/liquid/value"
)

// expr evaluates a value expression: its primary, then each filter in
// turn. safe reports whether the result is safe from escaping (ADR-004 D9
// item 2): a bare name bound to a safe value, or the result of a filter
// that marks it safe.
func (r *renderer) expr(e *ast.Expr) (value.Value, bool, *Error) {
	v, safe, err := r.primary(e.Base)
	if err != nil {
		return nil, false, err
	}
	for _, f := range e.Filters {
		if v, safe, err = r.filter(f, v); err != nil {
			return nil, false, err
		}
	}
	return v, safe, nil
}

// filterContext is the FilterContext a FilterDef applies through.
type filterContext struct{ m Meter }

func (c filterContext) Meter() Meter { return c.m }

// filter applies one filter through its definition. The renderer charges
// the application's node, nodeCost units, before the filter runs; the
// definition charges the rest of ADR-009 D11 item 2's table — the shallow
// sizes of its input, arguments and result, and any excess — as
// FilterDef's contract says. A string result is held to value-size and
// an array result to array-size.
func (r *renderer) filter(f *ast.Filter, in value.Value) (value.Value, bool, *Error) {
	def, ok := r.dialect.Filter(f.Name)
	if !ok {
		return nil, false, r.fail(f, CodeUnknownFilter, map[string]string{"filter": f.Name},
			fmt.Sprintf("the render's dialect has no filter %q", f.Name))
	}
	args := make([]Arg, len(f.Args))
	for i, a := range f.Args {
		v, _, err := r.primary(a.Value)
		if err != nil {
			return nil, false, err
		}
		args[i] = Arg{Name: a.Name, Value: v}
	}
	if err := r.charge(WorkUnits, nodeCost, f); err != nil {
		return nil, false, err
	}
	out, ferr := def.Apply(filterContext{r.meter}, in, args)
	if ferr != nil {
		return nil, false, r.adopt(ferr, f)
	}
	switch o := out.(type) {
	case nil:
		return nil, false, r.fail(f, CodeInternal, nil, fmt.Sprintf("filter %q returned no value and no error", f.Name))
	case value.String:
		if err := r.checkString(len(o.Text()), f); err != nil {
			return nil, false, err
		}
	case value.Array:
		if err := r.checkArray(int64(o.Len()), f); err != nil {
			return nil, false, err
		}
	}
	sf, isSafe := def.(SafeFilter)
	return out, isSafe && sf.MarksSafe(), nil
}

// primary evaluates a literal, a path or a range. empty and blank, as
// values, are "" (ADR-004 D5 item 2).
func (r *renderer) primary(p ast.Primary) (value.Value, bool, *Error) {
	switch p := p.(type) {
	case *ast.Literal:
		if p.Value == nil {
			if p.Kind == ast.LitEmpty || p.Kind == ast.LitBlank {
				return value.String{}, false, nil
			}
			return nil, false, r.fail(p, CodeInternal, nil, "a literal with no value")
		}
		return p.Value, false, nil
	case *ast.Path:
		return r.path(p)
	case *ast.Range:
		v, err := r.rangeValue(p)
		return v, false, err
	}
	return nil, false, r.fail(p, CodeInternal, nil, fmt.Sprintf("no evaluation of %T", p))
}

// rangeValue makes a range an array of its integers. The range is an
// operator: it costs nodeCost units, one for each bound, and one per
// element of its result (ADR-009 D11 item 2), and its result is held to
// array-size.
func (r *renderer) rangeValue(p *ast.Range) (value.Value, *Error) {
	from, to, err := r.rangeBounds(p, p)
	if err != nil {
		return nil, err
	}
	n := max(to-from+1, 0)
	if err := r.checkArray(n, p); err != nil {
		return nil, err
	}
	if err := r.charge(WorkUnits, saturatingAdd(nodeCost+2, n), p); err != nil {
		return nil, err
	}
	elems := make([]value.Value, n)
	for i := range elems {
		elems[i] = intValue(from + int64(i))
	}
	return newArray(elems), nil
}

// rangeBounds evaluates a range's bounds as Shopify does (ADR-004 D2).
// When both are literals, each is read as Ruby's to_i reads it: a float
// is truncated, a string read as String#to_i, nil is 0. Otherwise each is
// evaluated: an integer is itself, nil is 0, a string is read as
// String#to_i, and anything else — a float among them — is refused.
func (r *renderer) rangeBounds(p *ast.Range, at ast.Node) (from, to int64, err *Error) {
	lf, fromLit := p.From.(*ast.Literal)
	lt, toLit := p.To.(*ast.Literal)
	if fromLit && toLit {
		if from, err = r.literalToI(lf, at); err != nil {
			return 0, 0, err
		}
		to, err = r.literalToI(lt, at)
		return from, to, err
	}
	if from, err = r.rangeBound(p.From, at); err != nil {
		return 0, 0, err
	}
	to, err = r.rangeBound(p.To, at)
	return from, to, err
}

// literalToI reads a literal range bound as Ruby's to_i.
func (r *renderer) literalToI(l *ast.Literal, at ast.Node) (int64, *Error) {
	switch v := l.Value.(type) {
	case value.Number:
		if i, ok := templateInt(v); ok {
			return i, nil
		}
		f := v.Float64()
		if f > value.MaxSafeInteger || f < -value.MaxSafeInteger {
			return 0, r.fail(at, CodeNumberRange, nil, "a range bound beyond ±(2^53-1)")
		}
		return int64(f), nil
	case value.String:
		return r.stringToI(v, at)
	case value.Null:
		return 0, nil
	case nil:
		return 0, r.rangeErr(value.String{}, at)
	}
	return 0, r.rangeErr(l.Value, at)
}

// rangeBound evaluates a range bound that is not read at parse.
func (r *renderer) rangeBound(p ast.Primary, at ast.Node) (int64, *Error) {
	v, _, err := r.primary(p)
	if err != nil {
		return 0, err
	}
	switch v := v.(type) {
	case value.Number:
		if i, ok := templateInt(v); ok {
			return i, nil
		}
	case value.Null:
		return 0, nil
	case value.String:
		return r.stringToI(v, at)
	}
	return 0, r.rangeErr(v, at)
}

func (r *renderer) rangeErr(v value.Value, at ast.Node) *Error {
	return r.fail(at, CodeArgument, map[string]string{"argument": "range", "kind": kindName(v)},
		"a range's bounds are integers")
}

// lookup resolves a top-level name (ADR-004 D6 item 1): a block- or
// partial-scoped binding, innermost first, then a template-scoped
// binding, a counter, and a global.
func (r *renderer) lookup(name string) (binding, bool) {
	sc := r.fr.scopes
	for i := len(sc) - 1; i >= 0; i-- {
		if b, ok := sc[i][name]; ok {
			return b, true
		}
	}
	if b, ok := r.fr.vars[name]; ok {
		return b, true
	}
	if c, ok := r.fr.counters[name]; ok {
		return binding{v: intValue(c)}, true
	}
	if v, ok := r.env.Globals.Get(name); ok {
		return binding{v: v}, true
	}
	return binding{}, false
}

// path evaluates a variable lookup, strictly (ADR-004 D6).
func (r *renderer) path(p *ast.Path) (value.Value, bool, *Error) {
	segs := p.Segments
	name := p.Root
	b, ok := r.lookup(name)
	if !ok {
		return nil, false, r.fail(p, CodeUndefinedVariable, map[string]string{"name": name},
			fmt.Sprintf("%q is not bound", name))
	}
	v := b.v
	for i, s := range segs {
		var key value.Value
		if s.Index == nil {
			key = newString(s.Name)
		} else {
			k, _, err := r.primary(s.Index)
			if err != nil {
				return nil, false, err
			}
			key = k
		}
		var err *Error
		if v, err = r.property(v, key, s.Index == nil, p, i+1); err != nil {
			return nil, false, err
		}
	}
	return v, b.safe && len(segs) == 0, nil
}

// property is one step of a path: base's property or element key. upto
// is how many of p's segments the step completes, for the error's path.
// dot is set for a ".name" step, the only one that reads the size, first
// and last shorthands, as Shopify's command flags are set only for one: a
// bracketed "first" is an index like any other.
func (r *renderer) property(base, key value.Value, dot bool, p *ast.Path, upto int) (value.Value, *Error) {
	undefined := func() *Error {
		return r.fail(p, CodeUndefinedProperty, map[string]string{"path": pathText(p, upto)},
			fmt.Sprintf("a %s has no such property", kindName(base)))
	}
	ks, isString := key.(value.String)
	switch b := base.(type) {
	case value.Null:
		return nil, r.fail(p, CodeNilProperty, map[string]string{"path": pathText(p, upto)},
			"a property of nil; guard it with if")
	case value.Object:
		if !isString {
			return nil, undefined()
		}
		if v, ok := b.Get(ks.Text()); ok {
			return v, nil
		}
		if !dot {
			return nil, undefined()
		}
		if v, ok, err := r.shorthand(b, ks.Text(), p); ok || err != nil {
			return v, err
		}
		return nil, undefined()
	case value.Array:
		if n, ok := key.(value.Number); ok {
			i, ok := templateInt(n)
			if !ok {
				return nil, undefined()
			}
			if i < 0 {
				i += int64(b.Len())
			}
			if i < 0 || i >= int64(b.Len()) {
				return value.Null{}, nil
			}
			return b.At(int(i)), nil
		}
		if isString && dot {
			if v, ok, err := r.shorthand(b, ks.Text(), p); ok || err != nil {
				return v, err
			}
		}
		return nil, undefined()
	case value.String:
		if isString && dot {
			if v, ok, err := r.shorthand(b, ks.Text(), p); ok || err != nil {
				return v, err
			}
		}
	}
	return nil, undefined()
}

// shorthand is a string's, array's or object's size, first or last
// (ADR-004 D4 item 7); an object's own member of that name is looked up
// before it. first and last of an empty value are nil; of an object, they
// are its first and last [name, value] pairs in canonical order. A
// string's size counts its characters, and costs a unit per byte it
// reads; every other shorthand reads one place.
func (r *renderer) shorthand(v value.Value, name string, at ast.Node) (value.Value, bool, *Error) {
	switch name {
	case "size", "first", "last":
	default:
		return nil, false, nil
	}
	switch v := v.(type) {
	case value.String:
		s := v.Text()
		switch name {
		case "size":
			if err := r.charge(WorkUnits, int64(len(s)), at); err != nil {
				return nil, false, err
			}
			return intValue(int64(utf8.RuneCountInString(s))), true, nil
		case "first":
			if s == "" {
				return value.Null{}, true, nil
			}
			_, n := utf8.DecodeRuneInString(s)
			return newString(s[:n]), true, nil
		default:
			if s == "" {
				return value.Null{}, true, nil
			}
			_, n := utf8.DecodeLastRuneInString(s)
			return newString(s[len(s)-n:]), true, nil
		}
	case value.Array:
		switch {
		case name == "size":
			return intValue(int64(v.Len())), true, nil
		case v.Len() == 0:
			return value.Null{}, true, nil
		case name == "first":
			return v.At(0), true, nil
		default:
			return v.At(v.Len() - 1), true, nil
		}
	case value.Object:
		if name == "size" {
			return intValue(int64(v.Len())), true, nil
		}
		if v.Len() == 0 {
			return value.Null{}, true, nil
		}
		i := 0
		if name == "last" {
			i = v.Len() - 1
		}
		m := r.member(v, i)
		return newArray([]value.Value{newString(m.Name), m.Value}), true, nil
	}
	return nil, false, nil
}

// pathText writes p's root and its first upto segments as the template
// writes them, for an error's Args: the expression, never a value.
func pathText(p *ast.Path, upto int) string {
	var b strings.Builder
	b.WriteString(p.Root)
	for _, s := range p.Segments[:upto] {
		if s.Index == nil {
			b.WriteString("." + s.Name)
		} else {
			b.WriteString("[" + primaryText(s.Index) + "]")
		}
	}
	return b.String()
}

func primaryText(p ast.Primary) string {
	switch p := p.(type) {
	case *ast.Literal:
		if p.Kind != ast.LitString {
			return p.Text
		}
		if strings.Contains(p.Text, `"`) {
			return "'" + p.Text + "'"
		}
		return `"` + p.Text + `"`
	case *ast.Path:
		return pathText(p, len(p.Segments))
	case *ast.Range:
		return "(" + primaryText(p.From) + ".." + primaryText(p.To) + ")"
	}
	return ""
}

// operand is one side of a comparison: a value, or the empty or blank
// literal, which are not values (ADR-004 D5 item 2).
type operand struct {
	v       value.Value
	special ast.LiteralKind // LitEmpty, LitBlank or 0
}

func (r *renderer) operand(p ast.Primary) (operand, *Error) {
	if l, ok := p.(*ast.Literal); ok && (l.Kind == ast.LitEmpty || l.Kind == ast.LitBlank) {
		return operand{v: value.String{}, special: l.Kind}, nil
	}
	v, _, err := r.primary(p)
	return operand{v: v}, err
}

// kind names the operand's kind, for an error's Args.
func (o operand) kind() string {
	switch o.special {
	case ast.LitEmpty:
		return "empty"
	case ast.LitBlank:
		return "blank"
	}
	return kindName(o.v)
}

// operatorCost is what a comparison, and or or costs beyond its operands:
// nodeCost and the result's shallow size (ADR-009 D11 item 2).
const operatorCost = nodeCost + 1

// cond evaluates a condition (ADR-004 D2, D4 item 6, D5).
func (r *renderer) cond(c *ast.Cond) (bool, *Error) {
	switch c.Kind {
	case ast.CondValue:
		o, err := r.operand(c.Left)
		if err != nil {
			return false, err
		}
		return truthy(o.v), nil
	case ast.CondCompare:
		l, err := r.operand(c.Left)
		if err != nil {
			return false, err
		}
		rt, err := r.operand(c.Right)
		if err != nil {
			return false, err
		}
		return r.compare(c.Op, l, rt, c)
	case ast.CondAnd, ast.CondOr:
		// and and or are operators on two conditions, whose results are
		// booleans of shallow size 1. The left is evaluated first; and
		// stops at a falsy one and or at a truthy one.
		if err := r.charge(WorkUnits, operatorCost+2, c); err != nil {
			return false, err
		}
		x, err := r.cond(c.X)
		if err != nil {
			return false, err
		}
		if x == (c.Kind == ast.CondOr) {
			return x, nil
		}
		return r.cond(c.Y)
	}
	return false, r.fail(c, CodeInternal, nil, fmt.Sprintf("condition of kind %d", c.Kind))
}

// compare applies a comparison operator (ADR-004 D4 item 6). It costs
// operatorCost units and the shallow size of each operand, or, for an
// equality or contains between two arrays or objects, their deep sizes
// (ADR-009 D11 item 2); and an array's contains of a string compares the
// string with each element, so it costs the string's bytes once for each
// element too. All of it is charged before the comparison is made.
func (r *renderer) compare(op ast.CompareOp, l, rt operand, at ast.Node) (bool, *Error) {
	var cost int64
	if (op != ast.OpLt && op != ast.OpGt && op != ast.OpLe && op != ast.OpGe) && composite(l.v) && composite(rt.v) {
		cost = saturatingAdd(deepSize(l.v), deepSize(rt.v))
	} else {
		cost = saturatingAdd(shallow(l.v), shallow(rt.v))
	}
	if a, ok := l.v.(value.Array); ok && op == ast.OpContains && l.special == 0 {
		if s, ok := rt.v.(value.String); ok && a.Len() > 0 {
			cost = saturatingAdd(cost, min(int64(len(s.Text())), math.MaxInt64/int64(a.Len()))*int64(a.Len()))
		}
	}
	if err := r.charge(WorkUnits, saturatingAdd(operatorCost, cost), at); err != nil {
		return false, err
	}
	switch op {
	case ast.OpEq:
		return equal(l, rt), nil
	case ast.OpNe, ast.OpLtGt:
		return !equal(l, rt), nil
	case ast.OpContains:
		return r.contains(l, rt, at)
	case ast.OpLt, ast.OpGt, ast.OpLe, ast.OpGe:
		return r.order(op, l, rt, at)
	}
	return false, r.fail(at, CodeInternal, nil, fmt.Sprintf("operator %q", op))
}

// equal is ==: numbers by value, other values as the data model's
// equality, values of different kinds never; empty and blank as ADR-004
// D5 item 2 defines them, so that neither equals a keyword, itself
// included, as Shopify's equal_variables finds no value to test.
func equal(l, r operand) bool {
	switch {
	case l.special != 0 && r.special != 0:
		return false
	case l.special != 0:
		return matchesSpecial(l.special, r.v)
	case r.special != 0:
		return matchesSpecial(r.special, l.v)
	}
	return valuesEqual(l.v, r.v)
}

// valuesEqual is the equality == applies to two values.
func valuesEqual(a, b value.Value) bool {
	if a.Kind() != b.Kind() {
		return false
	}
	return value.Equal(a, b)
}

// matchesSpecial reports whether v == empty, or v == blank: empty holds
// for an empty string, array or object; blank for those, nil, false, and
// a string of only whitespace (ADR-004 D5 item 2).
func matchesSpecial(k ast.LiteralKind, v value.Value) bool {
	switch v := v.(type) {
	case value.String:
		if k == ast.LitEmpty {
			return v.Text() == ""
		}
		s := v.Text()
		for i := 0; i < len(s); i++ {
			if !ast.IsSpace(s[i]) {
				return false
			}
		}
		return true
	case value.Array:
		return v.Len() == 0
	case value.Object:
		return v.Len() == 0
	case value.Null:
		return k == ast.LitBlank
	case value.Bool:
		return k == ast.LitBlank && !bool(v)
	}
	return false
}

// order is <, >, <= and >=: numbers by value, strings by code point. With
// nil, empty or blank on either side it is false; with an array or an
// object on either side, or between a string and a number, it is refused.
// A boolean on the left is false, as Shopify's is; on the right of a
// number or string it is refused.
func (r *renderer) order(op ast.CompareOp, l, rt operand, at ast.Node) (bool, *Error) {
	refuse := func() (bool, *Error) {
		return false, r.fail(at, CodeComparison, map[string]string{"op": string(op), "left": l.kind(), "right": rt.kind()},
			fmt.Sprintf("%s %s %s does not compare", l.kind(), op, rt.kind()))
	}
	if l.special != 0 || rt.special != 0 {
		return false, nil
	}
	_, ln := l.v.(value.Null)
	_, rn := rt.v.(value.Null)
	if ln || rn {
		return false, nil
	}
	if composite(l.v) || composite(rt.v) {
		return refuse()
	}
	// A boolean has no order: Shopify's comparison is false when either
	// side does not respond to the operator.
	if _, ok := rt.v.(value.Bool); ok {
		return false, nil
	}
	var c int
	switch a := l.v.(type) {
	case value.Bool:
		return false, nil
	case value.Number:
		b, ok := rt.v.(value.Number)
		if !ok {
			return refuse()
		}
		x, y := a.Float64(), b.Float64()
		switch {
		case x < y:
			c = -1
		case x > y:
			c = 1
		}
	case value.String:
		b, ok := rt.v.(value.String)
		if !ok {
			return refuse()
		}
		c = strings.Compare(a.Text(), b.Text())
	default:
		return refuse()
	}
	switch op {
	case ast.OpLt:
		return c < 0, nil
	case ast.OpGt:
		return c > 0, nil
	case ast.OpLe:
		return c <= 0, nil
	}
	return c >= 0, nil
}

// contains tests a substring, an array's membership or an object's member
// name (ADR-004 D4 item 6). With nil or false on either side, or a left
// side that is a number, true, empty or blank, it is false, as Shopify's
// is. A string searched for a number or true is searched for its text;
// for an array or object it is refused.
func (r *renderer) contains(l, rt operand, at ast.Node) (bool, *Error) {
	if l.special != 0 || !truthy(rt.v) {
		return false, nil
	}
	switch a := l.v.(type) {
	case value.String:
		if composite(rt.v) {
			return false, r.fail(at, CodeComparison, map[string]string{"op": "contains", "left": l.kind(), "right": rt.kind()},
				fmt.Sprintf("a string does not contain an %s", rt.kind()))
		}
		return strings.Contains(a.Text(), scalarText(rt.v)), nil
	case value.Array:
		for _, e := range a.All() {
			if valuesEqual(e, rt.v) {
				return true, nil
			}
		}
		return false, nil
	case value.Object:
		if s, ok := rt.v.(value.String); ok {
			_, has := a.Get(s.Text())
			return has, nil
		}
	}
	return false, nil
}
