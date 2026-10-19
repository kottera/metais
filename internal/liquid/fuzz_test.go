// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package liquid

import (
	"testing"

	"github.com/kottera/metais/internal/liquid/ast"
)

// FuzzLiquidParse is ADR-010 D14 item 6's target: whatever the bytes,
// Parse, ParseValue and ParseCondition do not panic and recover no
// internal error; a template comes back exactly when no error does; and
// every position they give — each error's, and each node's span — is
// within the input. It runs at the default limits and at small ones. Its
// seed corpus, under testdata/fuzz/FuzzLiquidParse, holds a valid and an
// invalid Golden Liquid template for each tag of the corpus, and the
// lexer's and the bounds' edges, and runs in every go test.
func FuzzLiquidParse(f *testing.F) {
	d := constructDialect(f)
	f.Fuzz(func(t *testing.T, src []byte) {
		x := newLineIndex(string(src))
		checkErr := func(what string, e *Error) {
			if e.Code == CodeInternal {
				t.Fatalf("%s: internal error: %v", what, e)
			}
			if e.Offset < 0 || e.Offset > len(src) {
				t.Fatalf("%s: error at offset %d, outside %d bytes", what, e.Offset, len(src))
			}
			if l, c := x.at(e.Offset); l != e.Line || c != e.Col {
				t.Fatalf("%s: error at %d is %d:%d, want %d:%d", what, e.Offset, e.Line, e.Col, l, c)
			}
		}
		checkTree := func(what string, n ast.Node) {
			for _, v := range spanViolations(string(src), n) {
				t.Fatalf("%s: %s", what, v)
			}
		}
		for _, lim := range []ParseLimits{{}, {Nesting: 3, Size: 512}} {
			s := Source{Name: "f", Text: src}
			tpl, errs := Parse(d, s, lim)
			if (tpl == nil) != (len(errs) > 0) {
				t.Fatalf("Parse gave %v beside %d errors", tpl, len(errs))
			}
			for _, e := range errs {
				checkErr("Parse", e)
			}
			if tpl != nil {
				checkTree("Parse", tpl.Root)
			}
			if e, err := ParseValue(d, s, lim); err != nil {
				checkErr("ParseValue", err)
			} else {
				checkTree("ParseValue", e)
			}
			if c, err := ParseCondition(d, s, lim); err != nil {
				checkErr("ParseCondition", err)
			} else {
				checkTree("ParseCondition", c)
			}
		}
	})
}
