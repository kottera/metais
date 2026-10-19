// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package liquid

import (
	"bytes"
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

// fuzzGlobals is FuzzLiquidRender's data: a value of every kind, nested.
const fuzzGlobals = `{"a": 1, "b": 2.5, "s": "<x&y>", "e": "", "n": null, "t": true, "f": false,
	"xs": [1, "two", [3, 4], {"k": "v"}, null], "o": {"a": 1, "b": [1, 2], "size": 3, "c": {"d": null}},
	"big": 9007199254740991, "neg": -7}`

// fuzzMeter and fuzzRenderLimits are FuzzLiquidRender's small limits, so
// that every input's render is cheap and every bound is reachable.
var (
	fuzzMeter        = MeterLimits{RenderWork: 1 << 14, OutputSize: 1 << 12, AssignBytes: 1 << 12, LoopIterations: 1 << 10}
	fuzzRenderLimits = RenderLimits{PartialDepth: 3, ValueSize: 1 << 10, ArraySize: 1 << 8}
)

// FuzzLiquidRender is ADR-010 D14 item 6's render target, at small
// limits: whatever template parses, Render, and Eval and EvalCondition
// over the same bytes, recover no internal error — a recovered panic
// fails the target, since every one is a bug (ADR-010 D4 item 6); every
// error and every call site it carries is located within the input; the
// meter never passes a limit and the output never passes output-size;
// and a second render gives the same bytes, the same error and the same
// counts. The template may render itself, as the partial "self", so
// recursion reaches partial-depth; its dialect is Core() and the test
// definitions. Its seed corpus, under testdata/fuzz/FuzzLiquidRender,
// holds a valid Golden Liquid template for each of the corpus's tag
// groups, the renderer's own edges and filter chains under Core(), and
// runs in every go test.
func FuzzLiquidRender(f *testing.F) {
	d := fuzzDialect(f)
	v, err := jsonValue([]byte(fuzzGlobals))
	if err != nil {
		f.Fatal(err)
	}
	globals := Data(v).(value.Object)
	plim := ParseLimits{Nesting: 8, Size: 1 << 10}
	f.Fuzz(func(t *testing.T, src []byte) {
		x := newLineIndex(string(src))
		at := func(what string, source string, off, line, col int) {
			if source != "f" && source != "" {
				t.Fatalf("%s: located in %q, not the input", what, source)
			}
			if off < 0 || off > len(src) {
				t.Fatalf("%s: offset %d, outside %d bytes", what, off, len(src))
			}
			if l, c := x.at(off); l != line || c != col {
				t.Fatalf("%s: offset %d is %d:%d, want %d:%d", what, off, line, col, l, c)
			}
		}
		checkErr := func(what string, e *Error) {
			if e == nil {
				return
			}
			if e.Code == CodeInternal {
				t.Fatalf("%s: internal error: %v", what, e)
			}
			at(what, e.Source, e.Offset, e.Line, e.Col)
			for _, s := range e.Chain {
				at(what+"'s call site", s.Source, s.Offset, s.Line, s.Col)
			}
		}
		render := func() (string, *Error, Meter) {
			tpl, errs := Parse(d, Source{Name: "f", Text: src}, plim)
			if len(errs) > 0 {
				return "", nil, nil
			}
			m := NewMeter(fuzzMeter)
			env := &Env{Dialect: d, Globals: globals, Partials: mapPartials{"self": tpl}, Meter: m, Limits: fuzzRenderLimits}
			var b bytes.Buffer
			err := Render(&b, tpl, env)
			return b.String(), err, m
		}
		out, err, m := render()
		if m == nil {
			return
		}
		checkErr("Render", err)
		if len(out) > int(fuzzMeter.OutputSize) {
			t.Fatalf("Render wrote %d bytes, above output-size %d", len(out), fuzzMeter.OutputSize)
		}
		for w, lim := range map[Work]int64{WorkUnits: fuzzMeter.RenderWork, WorkOutputBytes: fuzzMeter.OutputSize,
			WorkHeldBytes: fuzzMeter.AssignBytes, WorkIterations: fuzzMeter.LoopIterations} {
			if m.Used(w) > lim {
				t.Fatalf("work %d: %d charged, above its limit %d", w, m.Used(w), lim)
			}
		}
		out2, err2, m2 := render()
		if out2 != out || (err == nil) != (err2 == nil) || (err != nil && err.Error() != err2.Error()) ||
			m2.Used(WorkUnits) != m.Used(WorkUnits) || m2.Used(WorkIterations) != m.Used(WorkIterations) {
			t.Fatalf("two renders differ: %q, %v and %q, %v", out, err, out2, err2)
		}
		env := &Env{Dialect: d, Globals: globals, Meter: NewMeter(fuzzMeter), Limits: fuzzRenderLimits}
		if e, perr := ParseValue(d, Source{Name: "f", Text: src}, plim); perr == nil {
			_, err := Eval(e, env)
			checkErr("Eval", err)
		}
		if c, perr := ParseCondition(d, Source{Name: "f", Text: src}, plim); perr == nil {
			_, err := EvalCondition(c, env)
			checkErr("EvalCondition", err)
		}
	})
}
