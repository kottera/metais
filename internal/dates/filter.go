// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package dates

import (
	"errors"
	"fmt"
	"strconv"

	"metais.org/metais/internal/liquid"
	"metais.org/metais/internal/liquid/ast"
	"metais.org/metais/internal/liquid/value"
)

// FilterName is the date filter's name.
const FilterName = "date"

// Filter returns the date filter of ADR-004 D11: {{ x | date: "%Y-%m-%d" }}
// formats x, an RFC 3339 full-date or date-time or an integer of Unix
// seconds, in the offset it carries, with the closed strftime set
// ParseFormat compiles. nil gives nil. internal/template's dialect
// registers it through Extend (plan §8); internal/liquid's Core() does not
// hold it.
//
// It takes exactly one positional argument, a string, and no keyword; a
// literal format is compiled when the template is parsed, so a format
// outside the set is found there. Every refusal is a *liquid.Error:
// CodeFilterArgument for an argument count or kind it never takes,
// CodeDateFormat for a format, and CodeDateInput for an input.
//
// Its work is charged as ADR-009 D11 item 2's table says: the shallow
// sizes of the input and of the format before anything else, then the
// result's byte length, as work units and held bytes, once the result is
// made. Formatting is linear in the format's length, which the first
// charge has already counted, so no work goes uncharged.
func Filter() liquid.FilterDef { return dateFilter{} }

type dateFilter struct{}

func (dateFilter) Name() string { return FilterName }

func (dateFilter) Check(f *ast.Filter) *liquid.Error {
	var pos []*ast.Arg
	for _, a := range f.Args {
		if a.Name != "" {
			return argErr(a.Name, fmt.Sprintf("date takes no keyword %q", a.Name))
		}
		pos = append(pos, a)
	}
	if err := arity(len(pos)); err != nil {
		return err
	}
	lit, ok := pos[0].Value.(*ast.Literal)
	if !ok {
		return nil
	}
	v := lit.Value
	if lit.Kind == ast.LitEmpty || lit.Kind == ast.LitBlank {
		// As a value outside a comparison, each is "" (ADR-004 D5 item 2).
		v = value.String{}
	}
	if v == nil {
		return nil
	}
	_, err := format(v)
	return err
}

func (dateFilter) Apply(c liquid.FilterContext, in value.Value, args []liquid.Arg) (value.Value, *liquid.Error) {
	m := c.Meter()
	work := liquid.Shallow(in)
	for _, a := range args {
		work += liquid.Shallow(a.Value)
	}
	if err := m.Charge(liquid.WorkUnits, work); err != nil {
		return nil, err
	}
	for _, a := range args {
		if a.Name != "" {
			return nil, argErr(a.Name, fmt.Sprintf("date takes no keyword %q", a.Name))
		}
	}
	if err := arity(len(args)); err != nil {
		return nil, err
	}
	f, err := format(args[0].Value)
	if err != nil {
		return nil, err
	}
	t, ok, ierr := input(in)
	if ierr != nil {
		return nil, ierr
	}
	if !ok {
		if err := m.Charge(liquid.WorkUnits, 1); err != nil {
			return nil, err
		}
		return value.Null{}, nil
	}
	s := f.Apply(t)
	n := int64(len(s))
	if err := m.Charge(liquid.WorkUnits, n); err != nil {
		return nil, err
	}
	if err := m.Charge(liquid.WorkHeldBytes, n); err != nil {
		return nil, err
	}
	out, verr := value.NewString(s)
	if verr != nil {
		return nil, &liquid.Error{Code: liquid.CodeInternal, Detail: verr.Error()}
	}
	return out, nil
}

// arity refuses n positional arguments unless n is 1.
func arity(n int) *liquid.Error {
	switch {
	case n == 0:
		return argErr("1", "date takes one argument, the format, not 0")
	case n > 1:
		return argErr(strconv.Itoa(n), fmt.Sprintf("date takes one argument, the format, not %d", n))
	}
	return nil
}

// format compiles the format argument v: a string of the closed set
// (ADR-004 D11 item 3).
func format(v value.Value) (Format, *liquid.Error) {
	s, ok := v.(value.String)
	if !ok {
		return Format{}, argErr("1", "the format is a "+liquid.KindName(v)+", not a string")
	}
	f, err := ParseFormat(s.Text())
	if err != nil {
		var fe *FormatError
		args := map[string]string{}
		if errors.As(err, &fe) && fe.Directive != "" {
			args["directive"] = fe.Directive
		}
		return Format{}, liquid.FilterError(FilterName, liquid.CodeDateFormat, args, err.Error())
	}
	return f, nil
}

// input is the instant v denotes, as FromValue reads it, with ADR-004 D4
// item 2's kinds: a number is seconds only when the template sees it as
// an integer, so the literal 0.0, a float, is refused. ok is false for
// nil.
func input(v value.Value) (t Time, ok bool, err *liquid.Error) {
	if n, isNum := v.(value.Number); isNum {
		if _, isInt := liquid.Integer(n); !isInt {
			return Time{}, false, inputErr(v, false)
		}
	}
	if s, isStr := v.(value.String); isStr && canaryNow != nil && (s.Text() == "now" || s.Text() == "today") {
		return canaryNow(), true, nil
	}
	t, ok, ferr := FromValue(v)
	if ferr != nil {
		var ie *InputError
		return Time{}, false, inputErr(v, errors.As(ferr, &ie) && ie.NamesNow)
	}
	if ok && canaryZone != nil {
		t = canaryZone(t)
	}
	return t, ok, nil
}

// inputErr is CodeDateInput for v. The detail names v's kind, never its
// value (ADR-004 D16 item 3).
func inputErr(v value.Value, namesNow bool) *liquid.Error {
	args := map[string]string{"type": liquid.KindName(v)}
	detail := "date takes an RFC 3339 date or date-time, or an integer of seconds, not this " + liquid.KindName(v)
	if namesNow {
		args["now"] = "true"
		detail = "date never reads the clock: the render time is the now global"
	}
	return liquid.FilterError(FilterName, liquid.CodeDateInput, args, detail)
}

func argErr(arg, detail string) *liquid.Error {
	return liquid.FilterError(FilterName, liquid.CodeFilterArgument, map[string]string{"argument": arg}, detail)
}

// The canary seams (ADR-009 D16 item 2). Only a canary build sets them,
// in canary_tz.go and canary_clock.go: canaryZone moves every instant into
// the host's zone before it is formatted, and canaryNow reads "now" and
// "today" from the clock. The tool's own build never does either.
var (
	canaryZone func(Time) Time
	canaryNow  func() Time
)
