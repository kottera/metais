// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package liquid

import (
	"bytes"
	"maps"
	"slices"
	"strings"
	"testing"

	"github.com/kottera/metais/internal/liquid/ast"
	"github.com/kottera/metais/internal/liquid/value"
)

// fnFilter is a test-only FilterDef defined by a function. The standard
// catalog is W3-4's; the renderer's tests apply these instead.
type fnFilter struct {
	name  string
	apply func(c FilterContext, in value.Value, args []Arg) (value.Value, *Error)
}

func (f fnFilter) Name() string           { return f.name }
func (fnFilter) Check(*ast.Filter) *Error { return nil }
func (f fnFilter) Apply(c FilterContext, in value.Value, args []Arg) (value.Value, *Error) {
	return f.apply(c, in, args)
}

// safeFilter is an fnFilter whose result is safe from escaping.
type safeFilter struct{ fnFilter }

func (safeFilter) MarksSafe() bool { return true }

// panicTag is a TagDef whose Render panics.
type panicTag struct{}

func (panicTag) Name() string                       { return "boom" }
func (panicTag) Check(*ast.Tag) *Error              { return nil }
func (panicTag) Render(TagContext, *ast.Tag) *Error { panic("a defect") }

// echoTag is a TagDef that writes its argument's text, unescaped, then
// charges one unit of WorkPartials, so a test sees each part of
// TagContext reached.
type echoTag struct{}

func (echoTag) Name() string          { return "say" }
func (echoTag) Check(*ast.Tag) *Error { return nil }
func (echoTag) Render(c TagContext, t *ast.Tag) *Error {
	v, err := c.Eval(t.Args[0].Value)
	if err != nil {
		return err
	}
	s, ok := v.(value.String)
	if !ok {
		return &Error{Code: "tag_input", Detail: "say takes a string"}
	}
	if err := c.Write([]byte(s.Text())); err != nil {
		return err
	}
	return c.Meter().Charge(WorkPartials, 1)
}

// testDialect is the renderer's test dialect: a few filters of its own,
// and two added tags.
//
//   - suffix: its input's text followed by its argument's;
//   - pair: an array of its input twice;
//   - object: an object {"a": its input};
//   - fail: refuses with code "filter_input";
//   - explode: panics;
//   - mark: its input, marked safe;
//   - grow: a string of its argument's length, charging nothing itself.
func testDialect(t testing.TB) *Dialect {
	t.Helper()
	text := func(v value.Value) string {
		if s, ok := v.(value.String); ok {
			return s.Text()
		}
		return scalarText(v)
	}
	d, err := NewDialect(Spec{
		Filters: []FilterDef{
			fnFilter{"suffix", func(_ FilterContext, in value.Value, args []Arg) (value.Value, *Error) {
				return stringValue(text(in) + text(args[0].Value)), nil
			}},
			fnFilter{"pair", func(_ FilterContext, in value.Value, _ []Arg) (value.Value, *Error) {
				return mustArray([]value.Value{in, in}), nil
			}},
			fnFilter{"object", func(_ FilterContext, in value.Value, _ []Arg) (value.Value, *Error) {
				return mustObject([]value.Member{{Name: "a", Value: in}}), nil
			}},
			fnFilter{"fail", func(FilterContext, value.Value, []Arg) (value.Value, *Error) {
				return nil, &Error{Code: "filter_input", Args: map[string]string{"filter": "fail"}}
			}},
			fnFilter{"explode", func(FilterContext, value.Value, []Arg) (value.Value, *Error) {
				panic("a defect")
			}},
			safeFilter{fnFilter{"mark", func(_ FilterContext, in value.Value, _ []Arg) (value.Value, *Error) {
				return in, nil
			}}},
			fnFilter{"grow", func(_ FilterContext, _ value.Value, args []Arg) (value.Value, *Error) {
				n, _ := templateInt(args[0].Value.(value.Number))
				return stringValue(strings.Repeat("x", int(n))), nil
			}},
		},
		Tags: []TagDef{panicTag{}, echoTag{}},
	})
	if err != nil {
		t.Fatal(err)
	}
	return d
}

// testLimits are generous bounds for every kind of work.
var testLimits = MeterLimits{RenderWork: 1 << 32, OutputSize: 1 << 24, AssignBytes: 1 << 24, LoopIterations: 1 << 20}

// option changes a test render's Env.
type option func(t testing.TB, e *Env)

// data binds the JSON object j as the globals, made by Data.
func data(j string) option {
	return func(t testing.TB, e *Env) {
		t.Helper()
		v, err := jsonValue([]byte(j))
		if err != nil {
			t.Fatalf("data %s: %v", j, err)
		}
		e.Globals = Data(v).(value.Object)
	}
}

// partials parses each named source as a partial.
func partials(srcs map[string]string) option {
	return func(t testing.TB, e *Env) {
		t.Helper()
		m := mapPartials{}
		for _, name := range slices.Sorted(maps.Keys(srcs)) {
			tpl, errs := Parse(e.Dialect, Source{Name: name, Text: []byte(srcs[name])}, ParseLimits{})
			if len(errs) > 0 {
				t.Fatalf("partial %s: %v", name, errs)
			}
			m[name] = tpl
		}
		e.Partials = m
	}
}

// meter renders with m.
func meter(m Meter) option { return func(_ testing.TB, e *Env) { e.Meter = m } }

// limits renders with l.
func limits(l RenderLimits) option { return func(_ testing.TB, e *Env) { e.Limits = l } }

// escapeHTML renders with ADR-004 D9's mapping as the Escape.
func escapeHTML(_ testing.TB, e *Env) {
	e.Escape = func(dst []byte, s string) []byte {
		for i := 0; i < len(s); i++ {
			switch c := s[i]; c {
			case '&':
				dst = append(dst, "&amp;"...)
			case '<':
				dst = append(dst, "&lt;"...)
			case '>':
				dst = append(dst, "&gt;"...)
			case '"':
				dst = append(dst, "&quot;"...)
			case '\'':
				dst = append(dst, "&#39;"...)
			default:
				dst = append(dst, c)
			}
		}
		return dst
	}
}

// testEnv is a render's Env under the test dialect and opts.
func testEnv(t testing.TB, opts ...option) *Env {
	t.Helper()
	e := &Env{Dialect: testDialect(t), Meter: NewMeter(testLimits)}
	for _, o := range opts {
		o(t, e)
	}
	return e
}

// renderText parses src as "t.liquid" and renders it.
func renderText(t testing.TB, src string, opts ...option) (string, *Error) {
	t.Helper()
	env := testEnv(t, opts...)
	tpl, errs := Parse(env.Dialect, Source{Name: "t.liquid", Text: []byte(src)}, ParseLimits{})
	if len(errs) > 0 {
		t.Fatalf("Parse(%q): %v", src, errs)
	}
	var b bytes.Buffer
	err := Render(&b, tpl, env)
	return b.String(), err
}

// renders requires src to render want.
func renders(t *testing.T, src, want string, opts ...option) {
	t.Helper()
	got, err := renderText(t, src, opts...)
	if err != nil {
		t.Errorf("Render(%q): %v", src, err)
		return
	}
	if got != want {
		t.Errorf("Render(%q) = %q, want %q", src, got, want)
	}
}

// renderRefused requires src's render to fail with code, and returns the
// error.
func renderRefused(t *testing.T, src string, code Code, opts ...option) *Error {
	t.Helper()
	_, err := renderText(t, src, opts...)
	if err == nil {
		t.Errorf("Render(%q) succeeded, want %s", src, code)
		return &Error{}
	}
	if err.Code != code {
		t.Errorf("Render(%q) = %v, want %s", src, err, code)
	}
	return err
}
