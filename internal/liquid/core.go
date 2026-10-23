// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package liquid

import (
	"fmt"
	"maps"
	"slices"
	"strconv"

	"metais.org/metais/internal/liquid/ast"
	"metais.org/metais/internal/liquid/value"
)

// Core returns the dialect of ADR-004's core set: every core tag of D2,
// which the grammar holds, and every standard filter of Shopify's
// standardfilters.rb at 5.14.0 that ADR-004's Context lists, with escape's
// alias h, but date. date is internal/dates's, and internal/template adds
// it through Extend (plan §8), so Core narrows ADR-010 D4 item 2's list by
// that one filter. No other filter is in Core, and it reserves no name:
// the globals ADR-004 D7 item 8 reserves are internal/template's too.
func Core() *Dialect { return core }

// core is built once; a Dialect is immutable.
var core = func() *Dialect {
	d, err := NewDialect(Spec{Filters: standardFilters()})
	if err != nil {
		panic("liquid: the core dialect: " + err.Error())
	}
	return d
}()

// FilterNames returns the names of the dialect's filters, sorted.
func (d *Dialect) FilterNames() []string { return slices.Sorted(maps.Keys(d.filters)) }

// TagNames returns the names of the tags the dialect adds to the core
// tags, sorted. CoreTags names the grammar's own.
func (d *Dialect) TagNames() []string { return slices.Sorted(maps.Keys(d.tags)) }

// stdFilter is a filter of the standard catalog. Its signature is what
// Check holds an application to, and fn its semantics: ADR-004's where it
// pins one, and Shopify's at 5.14.0 otherwise (D1 items 1 and 4).
//
// The charges are ADR-009 D11 item 2's, which make render-work defined
// rather than measured. The renderer charges the 16 units of the
// application's node. Apply charges the rest, as work units:
//
//   - the shallow size of the input and of each argument, before anything
//     else is done;
//   - the shallow size of the result. A filter that builds a string
//     charges its byte length before building it; one whose result is no
//     larger than what it was charged for already charges the result when
//     it is made. A built string is held to value-size by the renderer and
//     is no held bytes: assign-bytes counts only what assign and capture
//     store (ADR-009 Amendment A10 item 4);
//   - where a filter's true cost exceeds those shallow charges, the excess,
//     before the work: the elements of every nested array it flattens; the
//     deep size of an array whose text it takes; the deep size of values it
//     compares for equality or keys by; and for sort and sort_natural,
//     n·⌈log2 n⌉ comparisons of n elements, with each string key's bytes
//     charged ⌈log2 n⌉ times.
//
// safe marks the result safe from escaping (ADR-004 D9 item 2): escape,
// its alias h, and escape_once.
type stdFilter struct {
	name string
	sig  signature
	fn   func(c *call, in value.Value) (value.Value, *Error)
	safe bool
}

// signature is the shape of a filter's applications: from min to max
// positional arguments, and the keywords it knows, each at most once.
// literal, when set, checks each argument that is a literal, at parse.
type signature struct {
	min, max int
	keywords []string
	literal  func(filter string, arg int, keyword string, v value.Value) *Error
}

func (f *stdFilter) Name() string { return f.name }

func (f *stdFilter) MarksSafe() bool { return f.safe }

// Check refuses an application that is wrong whatever the data: an
// argument count the filter never takes, a keyword it does not know or
// given twice, and a literal argument the filter refuses.
func (f *stdFilter) Check(app *ast.Filter) *Error {
	var pos int
	var keywords []string
	for _, a := range app.Args {
		if a.Name == "" {
			pos++
		} else {
			keywords = append(keywords, a.Name)
		}
	}
	if err := f.shape(pos, keywords); err != nil {
		return err
	}
	if f.sig.literal == nil {
		return nil
	}
	pos = 0
	for _, a := range app.Args {
		if a.Name == "" {
			pos++
		}
		lit, ok := a.Value.(*ast.Literal)
		if !ok {
			continue
		}
		v := lit.Value
		if lit.Kind == ast.LitEmpty || lit.Kind == ast.LitBlank {
			// As a value outside a comparison, each is "" (ADR-004 D5
			// item 2).
			v = value.String{}
		}
		if v == nil {
			continue
		}
		if err := f.sig.literal(f.name, pos, a.Name, v); err != nil {
			return err
		}
	}
	return nil
}

// shape refuses pos positional arguments and keywords the signature does
// not admit.
func (f *stdFilter) shape(pos int, keywords []string) *Error {
	seen := map[string]bool{}
	for _, k := range keywords {
		if !slices.Contains(f.sig.keywords, k) {
			return filterErr(f.name, CodeFilterArgument, map[string]string{"argument": k},
				fmt.Sprintf("%s takes no keyword %q", f.name, k))
		}
		if seen[k] {
			return filterErr(f.name, CodeFilterArgument, map[string]string{"argument": k},
				fmt.Sprintf("keyword %q given twice", k))
		}
		seen[k] = true
	}
	if pos < f.sig.min || pos > f.sig.max {
		arg := strconv.Itoa(pos)
		if pos < f.sig.min {
			arg = strconv.Itoa(pos + 1)
		}
		return filterErr(f.name, CodeFilterArgument, map[string]string{"argument": arg},
			fmt.Sprintf("%s takes %s, not %d", f.name, arity(f.sig.min, f.sig.max), pos))
	}
	return nil
}

func arity(min, max int) string {
	switch {
	case min == max && max == 0:
		return "no argument"
	case min == max && max == 1:
		return "one argument"
	case min == max:
		return fmt.Sprintf("%d arguments", max)
	case min == 0 && max == 1:
		return "at most one argument"
	case min == 0:
		return fmt.Sprintf("at most %d arguments", max)
	}
	return fmt.Sprintf("%d to %d arguments", min, max)
}

// Apply applies the filter to in with args, charging its work to c's
// meter as the stdFilter's documentation says. An application its Check
// would refuse is refused here too, for a caller that did not parse it.
func (f *stdFilter) Apply(c FilterContext, in value.Value, args []Arg) (value.Value, *Error) {
	cl := &call{name: f.name, m: c.Meter()}
	var keywords []string
	work := shallow(in)
	for _, a := range args {
		work += shallow(a.Value)
		if a.Name == "" {
			cl.pos = append(cl.pos, a.Value)
			continue
		}
		keywords = append(keywords, a.Name)
		if cl.kw == nil {
			cl.kw = map[string]value.Value{}
		}
		cl.kw[a.Name] = a.Value
	}
	if err := f.shape(len(cl.pos), keywords); err != nil {
		return nil, err
	}
	if err := cl.units(work); err != nil {
		return nil, err
	}
	out, err := f.fn(cl, in)
	if err != nil {
		return nil, err
	}
	if !cl.charged {
		if err := cl.units(shallow(out)); err != nil {
			return nil, err
		}
	}
	return out, nil
}

// call is one application of a standard filter, as its fn sees it.
type call struct {
	name    string
	m       Meter
	pos     []value.Value
	kw      map[string]value.Value
	charged bool
}

// arg returns positional argument i (0-based), and whether it was given.
func (c *call) arg(i int) (value.Value, bool) {
	if i < len(c.pos) {
		return c.pos[i], true
	}
	return nil, false
}

// units charges n work units.
func (c *call) units(n int64) *Error {
	if n <= 0 {
		return nil
	}
	return c.m.Charge(WorkUnits, n)
}

// build charges a string of n bytes the filter is about to build, as its
// result: n work units.
func (c *call) build(n int64) *Error {
	c.charged = true
	return c.units(n)
}

// result charges a result of shallow size n the filter is about to make.
func (c *call) result(n int64) *Error {
	c.charged = true
	return c.units(n)
}

// text returns v's text, charging the work beyond v's shallow size.
func (c *call) text(v value.Value) (string, *Error) {
	if err := c.units(textWork(v)); err != nil {
		return "", err
	}
	return text(c.name, v)
}

// iterate returns Shopify's iteration of v, charging the flattening.
func (c *call) iterate(v value.Value) ([]value.Value, *Error) {
	if err := c.units(iterWork(v)); err != nil {
		return nil, err
	}
	return iterate(v), nil
}

// number returns v coerced to a number.
func (c *call) number(v value.Value) (tnum, *Error) { return toNumber(c.name, v) }

// standardFilters is the catalog, every filter of ADR-004's Context but
// date.
func standardFilters() []FilterDef {
	var fs []FilterDef
	for _, group := range [][]*stdFilter{mathFilters(), stringFilters(), arrayFilters()} {
		for _, f := range group {
			fs = append(fs, f)
		}
	}
	return fs
}
