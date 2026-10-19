// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package csv

import (
	"github.com/kottera/metais/internal/finding"
	"github.com/kottera/metais/internal/limits"
)

// The codes Read reports (ADR-003 Interfaces, ADR-009 D19). Each is class
// content, about an input.
const (
	CodeHeaderMissing     finding.Code = "csv.header_missing"
	CodeHeaderEmpty       finding.Code = "csv.header_empty"
	CodeHeaderDuplicate   finding.Code = "csv.header_duplicate"
	CodeFieldCount        finding.Code = "csv.field_count"
	CodeBareQuote         finding.Code = "csv.bare_quote"
	CodeAfterQuote        finding.Code = "csv.after_quote"
	CodeUnterminatedQuote finding.Code = "csv.unterminated_quote"
	CodeLimitExceeded     finding.Code = "limit.exceeded"
)

// noteFirst is the related note of a duplicate header name's first
// occurrence (ADR-006 D5 item 9).
const noteFirst = "first occurrence"

// report records the finding for code at p, pointing at *ptr when ptr is
// not nil.
func (r *reader) report(code finding.Code, p finding.Pos, ptr *string, params finding.Params, opts ...finding.Option) {
	l := r.from.At.At(p)
	l.Pointer, l.HasPointer = "", false
	if ptr != nil {
		l = l.WithPointer(*ptr)
	}
	if params == nil {
		params = finding.Params{}
	}
	r.findings = append(r.findings, finding.New(code, r.from.Input, &l, params, opts...))
}

// limitExceeded reports limit.exceeded for id at p and stops the reading.
// The reading stops at the bound, so the finding carries no observed value
// (ADR-009 D2 item 5).
func (r *reader) limitExceeded(id limits.ID, p finding.Pos, ptr *string) {
	r.report(CodeLimitExceeded, p, ptr, finding.Params{
		"limit":   finding.String(limits.Get(id).Name),
		"bound":   finding.Int(r.lim.Value(id)),
		"lowered": finding.Bool(r.lim.Lowered(id)),
	})
	r.stopped = true
}
