// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

// Package filters holds METAIS's own filters, json, yaml, csv,
// markdown_table and slug (ADR-004 D10), safe and json_script (D9, D10
// item 5), and registers date with them, which internal/dates defines
// (ADR-004 D11). They are what a package's
// templates and manifest expressions are parsed and rendered with beyond
// internal/liquid's standard catalog: internal/loader adds them to Core()
// through Extend, so one dialect parses a template and a path template
// alike (ADR-004 D16 item 5).
//
// Each filter refuses an input or argument of a kind it does not take
// with CodeFilterInput or CodeFilterArgument, a literal argument when the
// template is parsed, and reads nothing but its input, its arguments and
// the meter (ADR-004 D7 item 1). Each charges ADR-009 D11 item 2's table
// to the meter before the work it charges for: the shallow sizes of its
// input and arguments first; then, for a filter that writes a value's
// text, the input's deep size; and the result's byte length, as work units
// and held bytes, before the result is returned. json, yaml, csv and
// markdown_table count that length before they write a byte of it, since
// their text can be many times their input's size; slug's is no larger
// than a few bytes for each of its input's.
package filters

import (
	"fmt"
	"slices"
	"strconv"

	"metais.org/metais/internal/dates"
	"metais.org/metais/internal/jsonx"
	"metais.org/metais/internal/liquid"
	"metais.org/metais/internal/liquid/ast"
	"metais.org/metais/internal/liquid/value"
	"metais.org/metais/internal/yamlx"
)

// All returns METAIS's filters and date, the filters a dialect adds to
// Core().
func All() []liquid.FilterDef {
	return []liquid.FilterDef{
		jsonFilter, yamlFilter, csvFilter, markdownTableFilter, slugFilter,
		safeFilter{}, jsonScriptFilter{},
		dates.Filter(),
	}
}

// filter is one of METAIS's filters. Its applications take from min to
// max positional arguments (max -1 for any number), each checked by
// positional when it is a literal, and the keywords it knows, each at
// most once and checked by keyword when it is a literal; shapeCheck, when
// set, refuses a combination of them. apply is its semantics, given the
// arguments, after the shallow sizes are charged.
type filter struct {
	name       string
	min, max   int
	positional func(name, arg string, v value.Value) *liquid.Error
	keywords   map[string]func(name string, v value.Value) *liquid.Error
	shapeCheck func(name string, pos int, keywords []string) *liquid.Error
	apply      func(c *call, in value.Value) (value.Value, *liquid.Error)
}

func (f *filter) Name() string { return f.name }

// Check refuses an application that is wrong whatever the data: an
// argument count it never takes, a keyword it does not know or given
// twice, and a literal argument of a kind it refuses.
func (f *filter) Check(app *ast.Filter) *liquid.Error {
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
		if err := f.check(pos, a.Name, v); err != nil {
			return err
		}
	}
	return nil
}

// check refuses argument v, positional argument pos when keyword is empty.
func (f *filter) check(pos int, keyword string, v value.Value) *liquid.Error {
	if keyword != "" {
		return f.keywords[keyword](f.name, v)
	}
	if f.positional == nil {
		return nil
	}
	return f.positional(f.name, strconv.Itoa(pos), v)
}

// shape refuses pos positional arguments and keywords the filter does
// not take.
func (f *filter) shape(pos int, keywords []string) *liquid.Error {
	seen := map[string]bool{}
	for _, k := range keywords {
		if _, ok := f.keywords[k]; !ok {
			return argErr(f.name, k, fmt.Sprintf("%s takes no keyword %q", f.name, k))
		}
		if seen[k] {
			return argErr(f.name, k, fmt.Sprintf("keyword %q given twice", k))
		}
		seen[k] = true
	}
	switch {
	case pos < f.min:
		return argErr(f.name, strconv.Itoa(pos+1), fmt.Sprintf("%s takes at least %d arguments, not %d", f.name, f.min, pos))
	case f.max == 0 && pos > 0:
		return argErr(f.name, strconv.Itoa(pos), fmt.Sprintf("%s takes no positional argument, not %d", f.name, pos))
	case f.max >= 0 && pos > f.max:
		return argErr(f.name, strconv.Itoa(pos), fmt.Sprintf("%s takes at most %d arguments, not %d", f.name, f.max, pos))
	}
	if f.shapeCheck != nil {
		return f.shapeCheck(f.name, pos, keywords)
	}
	return nil
}

// Apply applies the filter to in with args: it refuses an application
// Check would refuse, charges the shallow sizes of the input and of every
// argument, checks every argument, and then applies the filter, which
// charges the rest of its work before doing it.
func (f *filter) Apply(c liquid.FilterContext, in value.Value, args []liquid.Arg) (value.Value, *liquid.Error) {
	cl := &call{name: f.name, m: c.Meter(), kw: map[string]value.Value{}}
	var keywords []string
	work := liquid.Shallow(in)
	for _, a := range args {
		work += liquid.Shallow(a.Value)
		if a.Name == "" {
			cl.pos = append(cl.pos, a.Value)
			continue
		}
		keywords = append(keywords, a.Name)
		cl.kw[a.Name] = a.Value
	}
	if err := f.shape(len(cl.pos), keywords); err != nil {
		return nil, err
	}
	if err := cl.units(work); err != nil {
		return nil, err
	}
	pos := 0
	for _, a := range args {
		if a.Name == "" {
			pos++
		}
		if err := f.check(pos, a.Name, a.Value); err != nil {
			return nil, err
		}
	}
	return f.apply(cl, in)
}

// call is one application of a filter, as its apply sees it.
type call struct {
	name string
	m    liquid.Meter
	pos  []value.Value
	kw   map[string]value.Value
}

// units charges n work units.
func (c *call) units(n int64) *liquid.Error {
	if n <= 0 {
		return nil
	}
	return c.m.Charge(liquid.WorkUnits, n)
}

// text charges a string result of n bytes before it is built — n work
// units and n held bytes — then builds it with build and returns it.
func (c *call) text(n int64, build func() string) (value.Value, *liquid.Error) {
	if err := c.units(n); err != nil {
		return nil, err
	}
	if n > 0 {
		if err := c.m.Charge(liquid.WorkHeldBytes, n); err != nil {
			return nil, err
		}
	}
	s := build()
	if int64(len(s)) != n {
		return nil, &liquid.Error{Code: liquid.CodeInternal, Detail: fmt.Sprintf("%s counted %d bytes and wrote %d", c.name, n, len(s))}
	}
	v, err := value.NewString(s)
	if err != nil {
		return nil, &liquid.Error{Code: liquid.CodeInternal, Detail: err.Error()}
	}
	return v, nil
}

// beyondShallow charges v's deep size beyond its shallow size, which the
// application has charged already: the work of visiting every value
// beneath v.
func (c *call) beyondShallow(v value.Value) *liquid.Error {
	return c.units(liquid.DeepSize(v) - liquid.Shallow(v))
}

func argErr(filter, arg, detail string) *liquid.Error {
	return liquid.FilterError(filter, liquid.CodeFilterArgument, map[string]string{"argument": arg}, detail)
}

func inputErr(filter string, v value.Value, detail string) *liquid.Error {
	return liquid.FilterError(filter, liquid.CodeFilterInput, map[string]string{"type": liquid.KindName(v)}, detail)
}

// jsonFilter is ADR-004 D10 item 1: the value as RFC 8785 JSON, or
// pretty-printed with indent: n, from 1 to 8. Every value has a JSON text,
// so it refuses no input.
var jsonFilter = &filter{
	name: "json",
	keywords: map[string]func(string, value.Value) *liquid.Error{
		"indent": func(name string, v value.Value) *liquid.Error {
			if _, ok := indent(v); !ok {
				return argErr(name, "indent", "indent is an integer from 1 to 8, not this "+liquid.KindName(v))
			}
			return nil
		},
	},
	apply: func(c *call, in value.Value) (value.Value, *liquid.Error) {
		n := 0
		if v, ok := c.kw["indent"]; ok {
			n, _ = indent(v)
		}
		if err := c.beyondShallow(in); err != nil {
			return nil, err
		}
		return c.text(jsonx.EncodedLen(in, n), func() string {
			b, err := jsonx.Encode(in, n)
			if err != nil {
				panic(err) // indent is checked to be from 1 to 8
			}
			return string(b)
		})
	},
}

// indent is json's indent: an integer from 1 to jsonx.MaxIndent.
func indent(v value.Value) (int, bool) {
	n, ok := v.(value.Number)
	if !ok {
		return 0, false
	}
	i, ok := liquid.Integer(n)
	if !ok || i < 1 || i > jsonx.MaxIndent {
		return 0, false
	}
	return int(i), true
}

// yamlFilter is ADR-004 D10 item 2: the value as a YAML 1.2 block
// document, as yamlx.Write writes it. Every value has one, so it refuses
// no input.
var yamlFilter = &filter{
	name: "yaml",
	apply: func(c *call, in value.Value) (value.Value, *liquid.Error) {
		if err := c.beyondShallow(in); err != nil {
			return nil, err
		}
		return c.text(yamlx.WrittenLen(in), func() string { return string(yamlx.Write(in)) })
	},
}

// stringArg refuses a positional argument that is not a string.
func stringArg(name, arg string, v value.Value) *liquid.Error {
	if _, ok := v.(value.String); !ok {
		return argErr(name, arg, "a column is named by a string, not this "+liquid.KindName(v))
	}
	return nil
}

// keywordsOf returns a filter's keyword checks, by name.
func keywordsOf(names ...string) map[string]func(string, value.Value) *liquid.Error {
	m := map[string]func(string, value.Value) *liquid.Error{}
	for _, n := range names {
		m[n] = tableKeywords[n]
	}
	return m
}

// sortedUnique returns names sorted in canonical order (RFC 8785's, by
// UTF-16 code units), each once.
func sortedUnique(names []string) []string {
	slices.SortFunc(names, value.CompareNames)
	return slices.Compact(names)
}
