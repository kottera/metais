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
	// CodeNumberRange is a numeric literal beyond what a template number
	// holds: an integer of magnitude above 2^53−1, or a float beyond
	// binary64 (ADR-004 D4 items 2 and 3).
	CodeNumberRange Code = "number_range"
	// CodeLimit is a bound of ParseLimits exceeded by the template's own
	// text. Args "limit" names it, as ADR-009 does: "template-nesting" or
	// "template-size". The parse stops there.
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

// Error is an error the engine reports.
//
// Line and Col are 1-based, Col counted in code points (ADR-006 D5 item
// 3), and Offset is the byte offset in the source they name. For a
// template, they name the "{{" or "{%" that opens the failing token, or,
// inside a liquid tag, the first non-blank character of the failing line
// (ADR-004 D16 item 2). For ParseValue and ParseCondition, they name the
// failing token within the expression.
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
}

// Error describes the error for a developer.
func (e *Error) Error() string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s:%d:%d: %s", e.Source, e.Line, e.Col, e.Code)
	for _, k := range slices.Sorted(maps.Keys(e.Args)) {
		fmt.Fprintf(&b, " %s=%q", k, e.Args[k])
	}
	if e.Detail != "" {
		b.WriteString(": ")
		b.WriteString(e.Detail)
	}
	return b.String()
}
