// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package liquid

import (
	"testing"

	"github.com/kottera/metais/internal/liquid/ast"
	"github.com/kottera/metais/internal/liquid/value"
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

// FuzzFilterApply holds every filter of Core() to FilterDef's contract
// on arbitrary values: whatever its input and arguments, Apply does not
// panic and gives no internal error; it gives a value of the data model
// exactly when it gives no error; it gives the same answer, charging the
// same, every time; and under a small meter it either succeeds or is
// refused by the meter. Each argument is JSON, read as a template value,
// or, when it is not JSON, a string of its bytes; an empty one is
// omitted. Its seed corpus, under testdata/fuzz/FuzzFilterApply, holds
// the edges of the coercions, the texts and the decoders, and runs in
// every go test.
func FuzzFilterApply(f *testing.F) {
	asValue := func(b string) (value.Value, bool) {
		if v, err := dataValue([]byte(b)); err == nil {
			return v, true
		}
		s, err := value.NewString(b)
		return s, err == nil
	}
	f.Fuzz(func(t *testing.T, in, arg1, arg2 string) {
		input, ok := asValue(in)
		if !ok {
			return
		}
		var args []Arg
		for _, a := range []string{arg1, arg2} {
			if a == "" {
				continue
			}
			v, ok := asValue(a)
			if !ok {
				return
			}
			args = append(args, Arg{Value: v})
		}
		for _, name := range Core().FilterNames() {
			def, _ := Core().Filter(name)
			m := newTestMeter(nil)
			got, err := def.Apply(testContext{m}, input, args)
			if err != nil && err.Code == CodeInternal {
				t.Fatalf("%s: internal error: %v", name, err)
			}
			if (got == nil) == (err == nil) {
				t.Fatalf("%s gave %v beside %v", name, got, err)
			}
			again := newTestMeter(nil)
			got2, err2 := def.Apply(testContext{again}, input, args)
			if (err == nil) != (err2 == nil) || (err == nil && !same(got, got2)) ||
				(err != nil && err.Code != err2.Code) || m.Used(WorkUnits) != again.Used(WorkUnits) {
				t.Fatalf("%s is not deterministic: %v, %v then %v, %v", name, got, err, got2, err2)
			}
			small := newTestMeter(map[Work]int64{WorkUnits: 64, WorkHeldBytes: 32})
			if _, err := def.Apply(testContext{small}, input, args); err != nil && err.Code == CodeInternal {
				t.Fatalf("%s under a small meter: %v", name, err)
			}
		}
	})
}
