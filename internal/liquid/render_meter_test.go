// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package liquid

import (
	"bytes"
	"testing"

	"metais.org/metais/internal/liquid/ast"
	"metais.org/metais/internal/liquid/value"
	"metais.org/metais/internal/testenv"
)

// TestMeterCounts: a meter counts each kind of work, refuses the first
// charge that would exceed its kind's limit, naming the limit as ADR-009
// does, does not raise the total past it, and refuses every later charge
// of the kind (the Meter contract).
func TestMeterCounts(t *testing.T) {
	testenv.Home(t)
	m := NewMeter(MeterLimits{RenderWork: 10, OutputSize: 1, AssignBytes: 1, LoopIterations: 1})
	if err := m.Charge(WorkUnits, 6); err != nil {
		t.Fatal(err)
	}
	if err := m.Charge(WorkUnits, 4); err != nil {
		t.Fatalf("a charge reaching the limit exactly: %v", err)
	}
	for _, n := range []int64{1, 0} {
		err := m.Charge(WorkUnits, n)
		if err == nil || err.Code != CodeLimit || err.Args["limit"] != "render-work" {
			t.Errorf("Charge(%d) past the limit = %v, want render-work's CodeLimit", n, err)
		}
	}
	if got := m.Used(WorkUnits); got != 10 {
		t.Errorf("Used = %d after a refusal, want 10", got)
	}
	for w, name := range map[Work]string{WorkOutputBytes: "output-size", WorkHeldBytes: "assign-bytes", WorkIterations: "loop-iterations"} {
		if err := m.Charge(w, 2); err == nil || err.Args["limit"] != name {
			t.Errorf("Charge(%d, 2) = %v, want %s's CodeLimit", w, err, name)
		}
		if got := m.Used(w); got != 0 {
			t.Errorf("Used(%d) = %d after a refused charge, want 0", w, got)
		}
	}
	// WorkPartials has no limit of its own, and zero means no bound.
	if err := m.Charge(WorkPartials, 1<<40); err != nil {
		t.Errorf("WorkPartials refused: %v", err)
	}
	unbounded := NewMeter(MeterLimits{})
	if err := unbounded.Charge(WorkUnits, 1<<62); err != nil {
		t.Errorf("a zero limit refused: %v", err)
	}
	if err := m.Charge(WorkUnits, -1); err == nil || err.Code != CodeInternal {
		t.Errorf("a negative charge = %v, want CodeInternal", err)
	}
}

// renderCounted renders src with a fresh meter of testLimits and returns
// it.
func renderCounted(t *testing.T, src string, opts ...option) Meter {
	t.Helper()
	m := NewMeter(testLimits)
	if _, err := renderText(t, src, append(opts, meter(m))...); err != nil {
		t.Fatalf("Render(%q): %v", src, err)
	}
	return m
}

// TestRenderCharges holds the renderer to ADR-009 D11 item 2's table:
// each node evaluated costs 16 units, a filter application among them,
// whose definition charges the rest; an operator adds the shallow size of
// its operands and result;
// an equality of two arrays costs their deep sizes; writing an array, and
// a string's size, cost what they read. The counts are exact,
// and the meter reads no clock: the same template costs the same on
// every run.
func TestRenderCharges(t *testing.T) {
	testenv.Home(t)
	for _, c := range []struct {
		src   string
		units int64
	}{
		{`hello`, 0},
		{`{{ x }}`, 16},
		// A filter application's node is the renderer's; the test filter
		// charges nothing of its own.
		{`{{ "ab" | suffix: "cd" }}`, 16 + 16},
		{`{% if 1 == 1 %}{% endif %}`, 16 + (16 + 1 + 1 + 1)},
		{`{% if true and false %}{% endif %}`, 16 + (16 + 1 + 1 + 1)},
		{`{% if true and false or true %}{% endif %}`, 16 + 19 + 19},
		// deep([1, [2, 3]]) = 2 + 1 + (2 + 1 + 1) = 7.
		{`{% if xs == ys %}{% endif %}`, 16 + (16 + 1 + 7 + 7)},
		{`{% if xs contains 1 %}{% endif %}`, 16 + (16 + 1 + 2 + 1)},
		{`{% assign r = (1..5) %}`, 16 + (16 + 2 + 5)},
		{`{% for i in (1..3) %}{{ i }}{% endfor %}`, 16 + 3*16},
		{`{% comment %}x{% endcomment %}{% raw %}y{% endraw %}`, 32},
		// Writing an array costs a unit per element; a string's size, a
		// unit per byte.
		{`{{ xs }}`, 16 + 2 + 2},
		{`{{ x.size }}{{ xs.size }}{{ x.first }}`, 16 + 3 + 16 + 16},
	} {
		d := data(`{"x": "abc", "xs": [1, [2, 3]], "ys": [1, [2, 3]]}`)
		m := renderCounted(t, c.src, d)
		if got := m.Used(WorkUnits); got != c.units {
			t.Errorf("%s: %d units, want %d", c.src, got, c.units)
		}
		if again := renderCounted(t, c.src, d).Used(WorkUnits); again != c.units {
			t.Errorf("%s: %d units on a second run, want %d", c.src, again, c.units)
		}
	}
	// Through Core(), a filter application is charged once: its node by
	// the renderer, and append's input, argument and result (2 + 2 + 4)
	// by the catalog.
	if m := renderCounted(t, `{{ "ab" | append: "cd" }}`, withDialect(Core())); m.Used(WorkUnits) != 16+16+8 {
		t.Errorf("append through the renderer: %d units, want %d", m.Used(WorkUnits), 16+16+8)
	}
	m := renderCounted(t, `ab{{ x }}{% capture c %}de{% endcapture %}{% assign s = "fgh" %}{% assign n = 1 %}`, data(`{"x": "abc"}`))
	if out, held := m.Used(WorkOutputBytes), m.Used(WorkHeldBytes); out != 5 || held != 5 {
		t.Errorf("%d output bytes and %d held, want 5 and 5", out, held)
	}
	for src, want := range map[string]int64{
		`{% for i in (1..10) limit: 3 %}{% endfor %}`:                10,
		`{% for x in xs limit: 2 %}{% endfor %}`:                     2,
		`{% tablerow x in xs %}{% endtablerow %}`:                    3,
		`{% for x in xs %}{% break %}{% endfor %}`:                   3,
		`{% render "p" for xs as x %}{% render "p" %}`:               3,
		`{% for x in xs %}{% for y in xs %}{% endfor %}{% endfor %}`: 12,
	} {
		m := renderCounted(t, src, data(`{"xs": [1, 2, 3]}`), partials(map[string]string{"p": ""}))
		if got := m.Used(WorkIterations); got != want {
			t.Errorf("%s: %d iterations, want %d", src, got, want)
		}
	}
	m = renderCounted(t, `{% render "p" %}{% render "p" for xs as x %}`, data(`{"xs": [1, 2]}`), partials(map[string]string{"p": `{% render "q" %}`, "q": ""}))
	if got := m.Used(WorkPartials); got != 6 {
		t.Errorf("%d partials rendered, want 6", got)
	}
}

// TestRenderLimits: each bound refuses at the node that breaches it.
func TestRenderLimits(t *testing.T) {
	testenv.Home(t)
	refusedAt := func(src, limit string, col int, opts ...option) {
		t.Helper()
		e := renderRefused(t, src, CodeLimit, opts...)
		if e.Args["limit"] != limit || e.Col != col {
			t.Errorf("%s: %v; want %s at column %d", src, e, limit, col)
		}
	}
	// 16 units an output: the third is refused.
	refusedAt(`{{ 1 }}{{ 2 }}{{ 3 }}`, "render-work", 15, meter(NewMeter(MeterLimits{RenderWork: 40})))
	refusedAt(`abcd`, "output-size", 1, meter(NewMeter(MeterLimits{OutputSize: 3})))
	refusedAt(`{% assign s = "abc" %}`, "assign-bytes", 1, meter(NewMeter(MeterLimits{AssignBytes: 2})))
	refusedAt(`x{% capture c %}abcd{% endcapture %}`, "value-size", 2, limits(RenderLimits{ValueSize: 3}))
	refusedAt(`{{ 1 | grow: 4 }}`, "value-size", 1, limits(RenderLimits{ValueSize: 3}))
	refusedAt(`{% assign r = (1..4) %}`, "array-size", 1, limits(RenderLimits{ArraySize: 3}))
	refusedAt(`{{ "a" | pair }}`, "array-size", 1, limits(RenderLimits{ArraySize: 1}))
	renders(t, `{{ 1 | grow: 3 }}{% assign r = (1..3) %}`, "xxx", limits(RenderLimits{ValueSize: 3, ArraySize: 3}))

	// A range whose span exceeds what remains of loop-iterations is
	// refused before its first iteration (ADR-009 D11 item 1).
	m := NewMeter(MeterLimits{LoopIterations: 100})
	out, err := renderText(t, `{% for i in (1..1000000000) %}x{% endfor %}`, meter(m))
	if err == nil || err.Args["limit"] != "loop-iterations" || out != "" {
		t.Errorf("a range beyond loop-iterations: %q, %v; want nothing written and loop-iterations refused", out, err)
	}
	renders(t, `{% for i in (1..100) %}{% endfor %}ok`, "ok", meter(NewMeter(MeterLimits{LoopIterations: 100})))
	refusedAt(`{% for i in (1..1000000000) limit: 1 %}{% endfor %}`, "loop-iterations", 1, meter(NewMeter(MeterLimits{LoopIterations: 100})))
}

// TestEval: a value expression evaluated with the globals bound, its
// filters applied, its errors located at the failing node with no
// Source (ADR-010 D4 item 3).
func TestEval(t *testing.T) {
	testenv.Home(t)
	env := testEnv(t, data(`{"a": {"b": "x"}, "n": 2}`))
	parse := func(s string) *ast.Expr {
		t.Helper()
		e, err := ParseValue(env.Dialect, Source{Text: []byte(s)}, ParseLimits{})
		if err != nil {
			t.Fatal(err)
		}
		return e
	}
	v, err := Eval(parse(`a.b | suffix: "!"`), env)
	if err != nil || !value.Equal(v, newString("x!")) {
		t.Errorf("Eval = %v, %v; want x!", v, err)
	}
	v, err = Eval(parse(`(1..n)`), env)
	if err != nil || v.(value.Array).Len() != 2 {
		t.Errorf("Eval of a range = %v, %v; want [1, 2]", v, err)
	}
	_, err = Eval(parse(`n | suffix: a.nope`), env)
	if err == nil || err.Code != CodeUndefinedProperty || err.Source != "" || err.Offset != 12 || err.Col != 13 {
		t.Errorf("Eval's error = %v; want undefined_property at the path, offset 12", err)
	}
	if _, err := Eval(parse(`1`), &Env{}); err == nil || err.Code != CodeInternal {
		t.Errorf("Eval with no meter = %v, want CodeInternal", err)
	}
}

// TestEvalCondition: a condition evaluated with the globals bound.
func TestEvalCondition(t *testing.T) {
	testenv.Home(t)
	env := testEnv(t, data(`{"a": 1, "b": null}`))
	for src, want := range map[string]bool{`a == 1`: true, `a == 1 and b`: false, `b or a > 0`: true, `b`: false} {
		c, perr := ParseCondition(env.Dialect, Source{Text: []byte(src)}, ParseLimits{})
		if perr != nil {
			t.Fatal(perr)
		}
		got, err := EvalCondition(c, env)
		if err != nil || got != want {
			t.Errorf("EvalCondition(%s) = %v, %v; want %v", src, got, err, want)
		}
	}
	c, _ := ParseCondition(env.Dialect, Source{Text: []byte(`b and b.c`)}, ParseLimits{})
	if _, err := EvalCondition(c, env); err != nil {
		t.Errorf("and short-circuits a nil property: %v", err)
	}
	c, _ = ParseCondition(env.Dialect, Source{Text: []byte(`a < "2"`)}, ParseLimits{})
	if _, err := EvalCondition(c, env); err == nil || err.Code != CodeComparison || err.Offset != 0 {
		t.Errorf("EvalCondition's error = %v; want comparison at the comparison, offset 0", err)
	}
}

// TestRenderPanicRecovered: a panic inside a render, an Eval or an
// EvalCondition — a definition's defect, or the engine's — is one
// CodeInternal error, never a Go panic (ADR-010 D4 item 6).
func TestRenderPanicRecovered(t *testing.T) {
	testenv.Home(t)
	renderRefused(t, `{{ 1 | explode }}`, CodeInternal)
	renderRefused(t, `{% boom %}`, CodeInternal)
	env := testEnv(t)
	e, perr := ParseValue(env.Dialect, Source{Text: []byte(`1 | explode`)}, ParseLimits{})
	if perr != nil {
		t.Fatal(perr)
	}
	if _, err := Eval(e, env); err == nil || err.Code != CodeInternal {
		t.Errorf("Eval of a panicking filter = %v, want CodeInternal", err)
	}
	if _, err := EvalCondition(nil, env); err == nil || err.Code != CodeInternal {
		t.Errorf("EvalCondition(nil) = %v, want CodeInternal", err)
	}
	if err := Render(&bytes.Buffer{}, nil, env); err == nil || err.Code != CodeInternal {
		t.Errorf("Render(nil) = %v, want CodeInternal", err)
	}
}
