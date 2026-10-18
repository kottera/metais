// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package finding

import (
	"fmt"
	"slices"
	"strings"
)

// Entry is one code's registry entry (ADR-006 D3 item 3).
type Entry struct {
	Code       Code
	Severity   Severity
	Class      Class
	Subject    SubjectKind
	Params     []ParamSpec
	Message    string // template
	Hint       string // template
	Span       SpanKind
	Since      string
	Statements []string
	Retired    string // the minor that retired the code; empty while it is emitted
}

// ParamSpec declares one parameter of a code.
type ParamSpec struct {
	Name    string
	Type    ParamType
	Items   ParamType // an array's element type; zero when its elements may be any scalar
	Meaning string
	Always  bool
}

// The codes this package and internal/limits emit.
const (
	CodeReportTruncated Code = "report.truncated"
	CodeLimitInvalid    Code = "limit.invalid"
)

// registry is the tool's table of codes, in code order. It is the Go form
// of spec/findings/registry.json, which is the normative text; a test fails
// on any difference between the two (ADR-006 D13 item 4).
var registry = []Entry{
	{
		Code:     CodeLimitInvalid,
		Severity: SeverityError,
		Class:    ClassUsage,
		Subject:  SubjectInvocation,
		Params: []ParamSpec{
			{Name: "reason", Type: TypeString, Meaning: "Why the argument is refused: unknown_name (no limit has that name), malformed_value (the value is not a non-negative whole number with a unit the limit allows), or above_default (the value is higher than the limit's default, and a limit is never raised).", Always: true},
			{Name: "name", Type: TypeString, Meaning: "The limit's name, present when the argument names a limit."},
			{Name: "default", Type: TypeNumber, Meaning: "The limit's default, present when the argument names a limit."},
			{Name: "unit", Type: TypeString, Meaning: "The limit's unit, present when the argument names a limit: bytes, count or ratio."},
			{Name: "value", Type: TypeNumber, Meaning: "The value asked for, in the limit's unit, present when the reason is above_default and the value is at most 2^53."},
		},
		Message:    "A --limit argument is refused ({reason}); a limit can only be lowered, to a whole number no higher than its default.",
		Hint:       "Run `metais limits` to see every limit's name, unit and default, and give --limit NAME=VALUE with VALUE at most that default.",
		Span:       SpanNone,
		Since:      "1.0",
		Statements: []string{},
	},
	{
		Code:     CodeReportTruncated,
		Severity: SeverityInfo,
		Class:    ClassContent,
		Subject:  SubjectReport,
		Params: []ParamSpec{
			{Name: "listed", Type: TypeNumber, Meaning: "How many findings the report lists before this one.", Always: true},
			{Name: "omitted", Type: TypeNumber, Meaning: "How many more findings were found and not listed.", Always: true},
		},
		Message:    "The report lists the first {listed} findings; {omitted} more were found and are counted in the summary but not listed.",
		Hint:       "Fix the findings listed and run the command again, or read the summary's totals by code.",
		Span:       SpanNone,
		Since:      "1.0",
		Statements: []string{},
	},
}

// Registry returns a copy of the tool's table of codes, in code order.
func Registry() []Entry {
	out := make([]Entry, len(registry))
	for i, e := range registry {
		e.Params = slices.Clone(e.Params)
		e.Statements = slices.Clone(e.Statements)
		out[i] = e
	}
	return out
}

func lookup(code Code) (*Entry, bool) {
	i, ok := slices.BinarySearchFunc(registry, code, func(e Entry, c Code) int {
		return strings.Compare(string(e.Code), string(c))
	})
	if !ok {
		return nil, false
	}
	return &registry[i], true
}

// Lookup returns code's registry entry, and false when code is not
// registered.
func Lookup(code Code) (Entry, bool) {
	e, ok := lookup(code)
	if !ok {
		return Entry{}, false
	}
	c := *e
	c.Params, c.Statements = slices.Clone(e.Params), slices.Clone(e.Statements)
	return c, true
}

// checkParams reports the first way p does not fit e's declared
// parameters.
func (e *Entry) checkParams(p Params) error {
	names := make([]string, 0, len(p))
	for name := range p {
		names = append(names, name)
	}
	slices.Sort(names)
	for _, name := range names {
		v := p[name]
		i := slices.IndexFunc(e.Params, func(s ParamSpec) bool { return s.Name == name })
		if i < 0 {
			return fmt.Errorf("parameter %q is not declared", name)
		}
		s := e.Params[i]
		if v.typ != s.Type {
			return fmt.Errorf("parameter %q is a %s, not a %s", name, v.typ, s.Type)
		}
		if s.Type == TypeArray && s.Items != 0 {
			for _, el := range v.list {
				if el.typ != s.Items {
					return fmt.Errorf("parameter %q holds a %s, not a %s", name, el.typ, s.Items)
				}
			}
		}
	}
	for _, s := range e.Params {
		if _, ok := p[s.Name]; s.Always && !ok {
			return fmt.Errorf("parameter %q is always present and was not given", s.Name)
		}
	}
	return nil
}
