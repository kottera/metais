// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package jsonx

import (
	"github.com/kottera/metais/internal/finding"
	"github.com/kottera/metais/internal/limits"
)

// The codes Parse reports (ADR-003 Interfaces, ADR-009 D19). Each is class
// content, about an input.
const (
	CodeSyntax           finding.Code = "json.syntax"
	CodeTrailingContent  finding.Code = "json.trailing_content"
	CodeDuplicateName    finding.Code = "json.duplicate_name"
	CodeLoneSurrogate    finding.Code = "json.lone_surrogate"
	CodeNoncharacter     finding.Code = "json.noncharacter"
	CodeEncodingInvalid  finding.Code = "input.encoding_invalid"
	CodeNumberOutOfRange finding.Code = "input.number_out_of_range"
	CodeLimitExceeded    finding.Code = "limit.exceeded"
)

// The words a json.syntax finding's expected and found params are made of.
// They describe the text by its class and never echo it (ADR-006 D9 item
// 4).
const (
	wantValue        = "value"
	wantName         = "member name"
	wantColon        = "':'"
	wantCommaOrClose = "',' or '%c'"
	wantDigit        = "digit"
	wantHex          = "four hexadecimal digits"
	wantStringEnd    = "closing quote"

	foundEnd       = "end of input"
	foundControl   = "control character"
	foundEscape    = "invalid escape"
	foundCharacter = "unexpected character"
	foundLeadZero  = "leading zero"
)

// The reasons of input.encoding_invalid and input.number_out_of_range.
const (
	reasonInvalidUTF8   = "invalid_utf8"
	reasonUTF16         = "utf16"
	reasonOverflow      = "overflow"
	reasonUnsafeInteger = "unsafe_integer"
)

// noteFirst is the related note of a duplicate name's first occurrence
// (ADR-006 D5 item 9).
const noteFirst = "first occurrence"

// at returns the location of the byte at off, pointing at ptr when hasPtr.
func (p *parser) at(off int, ptr string, hasPtr bool) *finding.Location {
	l := p.from.At.At(p.lines.pos(off))
	l.Pointer, l.HasPointer = "", false
	if hasPtr {
		l = l.WithPointer(ptr)
	}
	return &l
}

func (p *parser) report(code finding.Code, loc *finding.Location, params finding.Params, opts ...finding.Option) {
	if params == nil {
		params = finding.Params{}
	}
	p.findings = append(p.findings, finding.New(code, p.from.Input, loc, params, opts...))
}

// syntaxError reports json.syntax at off, inside the value at ptr, and
// stops the reading.
func (p *parser) syntaxError(off int, ptr, expected, found string) {
	p.report(CodeSyntax, p.at(off, ptr, true), finding.Params{
		"expected": finding.String(expected),
		"found":    finding.String(found),
	})
	p.stopped = true
}

// limitExceeded reports limit.exceeded for id, at off and the value at ptr,
// with the observed value when it is known (ADR-009 D2 item 5).
func (p *parser) limitExceeded(id limits.ID, off int, ptr string, hasPtr bool, observed int64) {
	params := finding.Params{
		"limit":   finding.String(limits.Get(id).Name),
		"bound":   finding.Int(p.lim.Value(id)),
		"lowered": finding.Bool(p.lim.Lowered(id)),
	}
	if observed >= 0 && observed <= 1<<53 {
		params["observed"] = finding.Int(observed)
	}
	p.report(CodeLimitExceeded, p.at(off, ptr, hasPtr), params)
}
