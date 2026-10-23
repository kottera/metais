// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package liquid

import (
	"fmt"
	"maps"
	"slices"
	"strings"
)

// Code is an engine error code. It is the engine's own name for what went
// wrong; internal/template maps each to a code of ADR-006's template
// family, or to ADR-009 D19's limit codes. Codes are never shown to a
// user as they are.
type Code string

// The parse error codes (ADR-004 D16 item 4, D1 and D2).
const (
	// CodeSyntax is text the strict2 grammar refuses, an unknown tag, or
	// U+0000 in a template (ADR-004 D1 items 2, 6 and 7, D2). A parse
	// stops at the first.
	CodeSyntax Code = "syntax"
	// CodeNotUTF8 is a template that is not valid UTF-8 (ADR-004 D1 item
	// 7). The parse stops there.
	CodeNotUTF8 Code = "not_utf8"
	// CodeInclude is the include tag, which the dialect excludes; Args
	// "use" is "render", the tag to use instead (ADR-004 D1 item 5).
	CodeInclude Code = "include"
	// CodeUnknownFilter is a filter the dialect does not define, found
	// whether or not its branch would run (ADR-004 D1 item 6). Args
	// "filter" names it.
	CodeUnknownFilter Code = "unknown_filter"
	// CodeReservedName is a template-scoped binding of a name the dialect
	// reserves (ADR-004 D7 item 8). Args "name" names it.
	CodeReservedName Code = "reserved_name"
	// CodePartialName is a render whose partial name is not a string
	// literal (ADR-004 D8 item 1).
	CodePartialName Code = "partial_name"
	// CodeSafePosition is a LastFilter applied other than as the last
	// filter of an output expression: before another filter, or in an
	// assign or a manifest expression (ADR-004 D9 item 3). Args "filter"
	// names it.
	CodeSafePosition Code = "safe_position"
	// CodeNumberRange is a numeric literal beyond what a template number
	// holds: an integer of magnitude above 2^53−1, or a float beyond
	// binary64 (ADR-004 D4 items 2 and 3).
	CodeNumberRange Code = "number_range"
	// CodeLimit is a bound exceeded. Args "limit" names it, as ADR-009
	// does. At parse it is "template-nesting" or "template-size", breached
	// by the template's own text, and the parse stops there. At render it
	// is "partial-depth", "value-size" or "array-size", or the limit of a
	// Meter's kind of work: "render-work", "output-size", "assign-bytes"
	// or "loop-iterations".
	CodeLimit Code = "limit"
	// CodeInternal is a defect of the engine, recovered from a panic
	// (ADR-010 D4 item 6). Every one is a bug.
	CodeInternal Code = "internal"
)

// The filter error codes (ADR-004 D16 item 1). A filter's Apply reports
// them at render, and its Check at parse when the arguments that decide
// them are literals. Each carries Args "filter", the filter's name.
// CodeNumberRange is reported by a filter too, for a result beyond what a
// template number holds (ADR-004 D4 item 3).
const (
	// CodeFilterInput is a filter's input of a kind the filter refuses.
	// Args "type" names the kind found, never the value.
	CodeFilterInput Code = "filter_input"
	// CodeFilterArgument is a filter applied with an argument count it
	// never takes, a keyword it does not know, or an argument of a kind it
	// refuses. Args "argument" names the argument: its 1-based position,
	// or its keyword.
	CodeFilterArgument Code = "filter_argument"
	// CodeDivisionByZero is divided_by or modulo by zero (ADR-004 D4
	// item 3).
	CodeDivisionByZero Code = "division_by_zero"
	// CodeObjectOutput is the text of an object, which no value has
	// (ADR-004 D4 item 4); Args "use" is "json", the filter to use.
	CodeObjectOutput Code = "object_output"
	// CodeInvalidUTF8 is a decoding filter's result that is not a sequence
	// of Unicode scalar values (ADR-004 D4 item 1).
	CodeInvalidUTF8 Code = "invalid_utf8"
)

// The codes of the filters a dialect adds to Core(): METAIS's own and
// date (ADR-004 D10, D11), which internal/template and internal/dates
// define. The engine never reports them itself; each carries Args
// "filter" like the codes above.
const (
	// CodeDateInput is a date input ADR-004 D11 item 1 does not accept.
	// Args "type" names the kind found; "now" is "true" for the strings
	// "now" and "today", for which the finding names the now global.
	CodeDateInput Code = "date_input"
	// CodeDateFormat is a date format outside ADR-004 D11 item 3's closed
	// set, or an empty one. Args "directive" is the first directive
	// outside the set, as the format writes it, and absent for an empty
	// format.
	CodeDateFormat Code = "date_format"
	// CodeSlugEmpty is a slug whose result is empty (ADR-004 D10 item 6).
	CodeSlugEmpty Code = "slug_empty"
	// CodeDurationInput is a duration input MTS-FLT-021 does not read
	// (ADR-004 Amendment A2 decision item 4). Args "type" names the kind
	// found and "reason" why it is refused: kind, calendar, range or form.
	CodeDurationInput Code = "duration_input"
	// CodeDurationFormat is a duration format outside MTS-FLT-022's set,
	// or an empty one. Args "directive" is the first directive outside
	// the set, as the format writes it, and absent for an empty format.
	CodeDurationFormat Code = "duration_format"
)

// The render error codes (ADR-004 D4 to D6, D16 item 1). A render stops at
// its first error.
const (
	// CodeUndefinedVariable is a top-level name bound by nothing in scope
	// (ADR-004 D6 item 1). Args "name" names it.
	CodeUndefinedVariable Code = "undefined_variable"
	// CodeUndefinedProperty is a property of an object that neither
	// declares nor holds it, a property a string, number, boolean or array
	// does not have, or an index of the wrong type (ADR-004 D6 items 4, 6
	// and 7). Args "path" is the expression up to and including the
	// failing step, as the template writes it.
	CodeUndefinedProperty Code = "undefined_property"
	// CodeNilProperty is a property of nil (ADR-004 D6 item 5). Args
	// "path" is the expression up to and including the failing step.
	CodeNilProperty Code = "nil_property"
	// CodeComparison is an ordering comparison, or a contains, between
	// values Shopify does not order or search: a string with a number, or
	// an array or object (ADR-004 D4 item 6). Args "op" is the operator,
	// "left" and "right" the kinds of its operands.
	CodeComparison Code = "comparison"
	// CodeArgument is an argument of a core tag of the wrong type: a for
	// loop's limit or offset that is not an integer, a tablerow's cols,
	// limit or offset that has no integer value, or a range bound that is
	// not one (ADR-004 D2). Args "tag" names the tag ("for" or
	// "tablerow"; absent for a range bound), "argument" the argument
	// ("limit", "offset", "cols", or "range" for a bound), and "kind" the
	// kind of value it was given.
	CodeArgument Code = "argument"
	// CodeWrite is the writer refusing the render's output. Detail holds
	// the writer's error. It is no finding of the template's.
	CodeWrite Code = "write"
)

// Site is one render call: where in which template it is written.
type Site struct {
	Source    string
	Line, Col int
	Offset    int
}

// Error is an error the engine reports.
//
// Line and Col are 1-based, Col counted in code points (ADR-006 D5 item
// 3), and Offset is the byte offset in the source they name. For a
// template, they name the "{{" or "{%" that opens the failing token, or,
// inside a liquid tag, the first non-blank character of the failing line
// (ADR-004 D16 item 2). For ParseValue and ParseCondition, they name the
// failing token within the expression. For Render, they name the token of
// the statement that failed, in the template or partial Source names; for
// Eval and EvalCondition, the node of the expression that failed, and
// Source is empty.
//
// Args carries the error's parameters, each named by the code's
// documentation. Detail is the engine's own description, for a developer
// reading a test failure. Neither ever holds a value from a template's
// data, and neither is shown to a user: internal/template builds the
// finding from Code and Args (ADR-004 D16 item 3).
type Error struct {
	Source    string
	Line, Col int
	Offset    int
	Code      Code
	Args      map[string]string
	Detail    string
	// Chain is the render calls that led to a render error inside a
	// partial, outermost first: the output's template's render tag, then
	// each partial's, up to the one that rendered Source (ADR-004 D16
	// item 2). It is empty for an error in the template rendered.
	Chain []Site
}

// Error describes the error for a developer.
func (e *Error) Error() string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s:%d:%d: %s", e.Source, e.Line, e.Col, e.Code)
	for _, k := range slices.Sorted(maps.Keys(e.Args)) {
		fmt.Fprintf(&b, " %s=%q", k, e.Args[k])
	}
	for _, c := range e.Chain {
		fmt.Fprintf(&b, " (rendered from %s:%d:%d)", c.Source, c.Line, c.Col)
	}
	if e.Detail != "" {
		b.WriteString(": ")
		b.WriteString(e.Detail)
	}
	return b.String()
}
