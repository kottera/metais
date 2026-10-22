// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package filters

import (
	"bytes"
	"testing"

	"metais.org/metais/internal/liquid"
	"metais.org/metais/internal/liquid/value"
	"metais.org/metais/internal/testenv"
)

// dialect is the core catalog with every METAIS filter.
var dialect = func() *liquid.Dialect {
	d, err := liquid.Core().Extend(liquid.Spec{Filters: All()})
	if err != nil {
		panic(err)
	}
	return d
}()

// renderWith renders src in the dialect with s bound to the string str and
// o to an object holding it, escaped as html when html is set, and returns
// its text, the error that stopped it, and the meter it charged.
func renderWith(t *testing.T, src, str string, html bool) (string, *liquid.Error, liquid.Meter) {
	t.Helper()
	tpl, errs := liquid.Parse(dialect, liquid.Source{Name: "t", Text: []byte(src)}, liquid.ParseLimits{})
	if len(errs) > 0 {
		return "", errs[0], nil
	}
	s, err := value.NewString(str)
	if err != nil {
		t.Fatal(err)
	}
	n, _ := value.NumberFromFloat(1)
	o, err := value.NewObject(value.Member{Name: "z", Value: s}, value.Member{Name: "a", Value: n})
	if err != nil {
		t.Fatal(err)
	}
	g, err := value.NewObject(value.Member{Name: "s", Value: s}, value.Member{Name: "o", Value: o})
	if err != nil {
		t.Fatal(err)
	}
	m := liquid.NewMeter(liquid.MeterLimits{})
	env := &liquid.Env{Dialect: dialect, Globals: g, Meter: m}
	if html {
		env.Escape = liquid.EscapeHTML
	}
	var b bytes.Buffer
	e := liquid.Render(&b, tpl, env)
	return b.String(), e, m
}

// TestSafe: safe writes its input as it is in an output whose escape is
// html, and does nothing in one whose escape is none; it takes no argument
// (ADR-004 D9 item 3).
func TestSafe(t *testing.T) {
	testenv.Home(t)
	for _, html := range []bool{true, false} {
		if got, e, _ := renderWith(t, `{{ s | safe }}`, `<i>&</i>`, html); e != nil || got != `<i>&</i>` {
			t.Errorf("html %v: %q, %v", html, got, e)
		}
	}
	if _, e, _ := renderWith(t, `{{ s | safe: 1 }}`, "", false); e == nil || e.Code != liquid.CodeFilterArgument || e.Args["argument"] != "1" {
		t.Errorf("safe with an argument: %v", e)
	}
	if _, e, _ := renderWith(t, `{{ s | safe | upcase }}`, "", false); e == nil || e.Code != liquid.CodeSafePosition {
		t.Errorf("safe before a filter: %v", e)
	}
}

// TestJSONScript: json_script is the canonical form with "<", ">", "&",
// U+2028 and U+2029 written as \u escapes; its result is safe; and it
// charges the input's deep size, then its result's bytes as units and as
// held bytes (ADR-004 D10 item 5, ADR-009 D11 item 2).
func TestJSONScript(t *testing.T) {
	testenv.Home(t)
	in := "</script><!-- & \u2028\u2029 \"é\""
	const want = `{"a":1,"z":"\u003c/script\u003e\u003c!-- \u0026 \u2028\u2029 \"é\""}`
	for _, html := range []bool{true, false} {
		got, e, m := renderWith(t, `{{ o | json_script }}`, in, html)
		if e != nil || got != want {
			t.Errorf("html %v: %s, %v; want %s", html, got, e, want)
			continue
		}
		// The output node, the path, the filter's node, then the deep
		// size of o (its two members, a string and a number) and the
		// result twice over: as units and as held bytes.
		deep := int64(2 + len(in) + 1)
		if got := m.Used(liquid.WorkHeldBytes); got != int64(len(want)) {
			t.Errorf("held bytes %d, want %d", got, len(want))
		}
		if got := m.Used(liquid.WorkUnits); got < deep+int64(len(want)) {
			t.Errorf("units %d, want at least %d", got, deep+int64(len(want)))
		}
	}
	if got, e, _ := renderWith(t, `{{ s | json_script | upcase }}`, "<", true); e != nil || got != `&quot;\U003C&quot;` {
		t.Errorf("json_script then upcase, escaped: %q, %v", got, e)
	}
	if _, e, _ := renderWith(t, `{{ s | json_script: indent: 2 }}`, "", false); e == nil || e.Code != liquid.CodeFilterArgument || e.Args["argument"] != "indent" {
		t.Errorf("json_script with an argument: %v", e)
	}
}
