// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package liquid

import (
	"math"
	"testing"

	"github.com/kottera/metais/internal/liquid/value"
	"github.com/kottera/metais/internal/testenv"
)

// TestTruthiness: only nil and false are falsy; 0, "", [] and {} are
// truthy (ADR-004 D5 item 1).
func TestTruthiness(t *testing.T) {
	testenv.Home(t)
	for v, want := range map[string]string{
		`0`: "T", `""`: "T", `[]`: "T", `{}`: "T", `true`: "T", `0.0`: "T", `"false"`: "T",
		`null`: "F", `false`: "F",
	} {
		renders(t, `{% if v %}T{% else %}F{% endif %}{% unless v %}F{% else %}T{% endunless %}`, want+want, data(`{"v": `+v+`}`))
	}
}

// TestEmptyBlank: x == empty for an empty string, array or object;
// x == blank also for nil, false and a string of only whitespace; != and
// <> negate; as values, each is "" (ADR-004 D5 item 2).
func TestEmptyBlank(t *testing.T) {
	testenv.Home(t)
	for _, c := range []struct {
		v            string
		empty, blank bool
	}{
		{`""`, true, true},
		{`[]`, true, true},
		{`{}`, true, true},
		{`null`, false, true},
		{`false`, false, true},
		{`" \t\n\u000b\f\r"`, false, true},
		{`"a"`, false, false},
		{`0`, false, false},
		{`[0]`, false, false},
		{`true`, false, false},
	} {
		yes := func(b bool) string {
			if b {
				return "T"
			}
			return "F"
		}
		want := yes(c.empty) + yes(c.empty) + yes(!c.empty) + yes(!c.empty) + yes(c.blank) + yes(c.blank) + yes(!c.blank)
		renders(t, `{% if v == empty %}T{% else %}F{% endif %}{% if empty == v %}T{% else %}F{% endif %}`+
			`{% if v != empty %}T{% else %}F{% endif %}{% if v <> empty %}T{% else %}F{% endif %}`+
			`{% if v == blank %}T{% else %}F{% endif %}{% if blank == v %}T{% else %}F{% endif %}`+
			`{% if v != blank %}T{% else %}F{% endif %}`, want, data(`{"v": `+c.v+`}`))
	}
	renders(t, `[{{ empty }}][{{ blank }}]{% if empty %}T{% endif %}`, "[][]T")
	// Neither keyword is a value either can test, itself included.
	renders(t, `{% if empty == empty %}E{% endif %}{% if blank == blank %}B{% endif %}{% if empty == blank %}F{% endif %}{% if blank != blank %}N{% endif %}`, "N")
}

// TestComparison: Shopify's operators; integers and floats by value,
// strings by code point, nil and booleans ordering false, and the
// refusals of ADR-004 D4 item 6.
func TestComparison(t *testing.T) {
	testenv.Home(t)
	d := data(`{"xs": [1, 2], "ys": [1, 2], "o": {"a": 1}, "n": null, "s": "abc"}`)
	for cond, want := range map[string]bool{
		`1 == 1.0`: true, `1 != 1.0`: false, `1 <> 2`: true, `1 < 1.5`: true, `2 >= 2.0`: true, `-1 > -2`: true,
		`"a" < "b"`: true, `"B" < "a"`: true, `"é" > "z"`: true, `"ab" <= "ab"`: true,
		`n < 1`: false, `1 > n`: false, `n <= n`: false, `n == nil`: true, `n == false`: false,
		`"1" == 1`: false, `true == true`: true, `xs == ys`: true, `xs == o`: false, `o == o`: true,
		`true < 1`: false, `1 < true`: false, `5 > true`: false, `"a" >= false`: false, `empty < 1`: false, `1 >= blank`: false,
		`s contains "b"`: true, `s contains "z"`: false, `xs contains 2`: true, `xs contains 2.0`: true,
		`xs contains "2"`: false, `o contains "a"`: true, `o contains "b"`: false, `o contains 1`: false,
		`"a1" contains 1`: true, `"true" contains true`: true, `s contains n`: false, `n contains "a"`: false,
		`3 contains 3`: false, `"false" contains false`: false,
	} {
		w := "F"
		if want {
			w = "T"
		}
		renders(t, `{% if `+cond+` %}T{% else %}F{% endif %}`, w, d)
	}
	for _, c := range []struct{ cond, op, left, right string }{
		{`"2" > 1`, ">", "string", "integer"},
		{`1 < "2"`, "<", "integer", "string"},
		{`xs < 2`, "<", "array", "integer"},
		{`1 <= o`, "<=", "integer", "object"},
		{`s contains xs`, "contains", "string", "array"},
	} {
		e := renderRefused(t, `x{% if `+c.cond+` %}{% endif %}`, CodeComparison, d)
		if e.Args["op"] != c.op || e.Args["left"] != c.left || e.Args["right"] != c.right || e.Col != 2 {
			t.Errorf("%s: %v; want op %s, %s and %s, at the if tag", c.cond, e, c.op, c.left, c.right)
		}
	}
}

// TestStrictNames: what the data holds, a declared nil among it, is
// readable; anything else is an error, never nil (ADR-004 D6).
func TestStrictNames(t *testing.T) {
	testenv.Home(t)
	d := data(`{"a": {"b": null, "c": 1}, "xs": [1, 2, 3], "s": "héllo", "n": 1, "t": true,
		"o": {"size": "S", "first": "F"}, "p": {"x": 1, "y": 2}, "k": "c", "a b": "spaced"}`)
	e := renderRefused(t, `x{{ nope }}`, CodeUndefinedVariable, d)
	if e.Args["name"] != "nope" || e.Source != "t.liquid" || e.Line != 1 || e.Col != 2 || e.Offset != 1 {
		t.Errorf("undefined variable: %v; want name nope at t.liquid:1:2", e)
	}
	for src, path := range map[string]string{
		`{{ a.nope }}`:      "a.nope",
		`{{ a["z"] }}`:      `a["z"]`,
		`{{ a[1] }}`:        "a[1]",
		`{{ xs[1.5] }}`:     "xs[1.5]",
		`{{ xs["a"] }}`:     `xs["a"]`,
		`{{ xs.nope }}`:     "xs.nope",
		`{{ s.nope }}`:      "s.nope",
		`{{ s[0] }}`:        "s[0]",
		`{{ n.size }}`:      "n.size",
		`{{ t.first }}`:     "t.first",
		`{{ a[k].nope }}`:   "a[k].nope",
		`{{ xs["first"] }}`: `xs["first"]`,
		`{{ s["size"] }}`:   `s["size"]`,
		`{{ p["last"] }}`:   `p["last"]`,
	} {
		e := renderRefused(t, src, CodeUndefinedProperty, d)
		if e.Args["path"] != path {
			t.Errorf("%s: path %q, want %q", src, e.Args["path"], path)
		}
	}
	e = renderRefused(t, `{{ a.b.c }}`, CodeNilProperty, d)
	if e.Args["path"] != "a.b.c" {
		t.Errorf("nil property: path %q, want a.b.c", e.Args["path"])
	}
	renderRefused(t, `{{ a.b | suffix: "" | suffix: a.b.c }}`, CodeNilProperty, d)
	renders(t, `[{{ a.b }}]{% if a.b and a.b.c %}x{% endif %}{{ a[k] }}{{ o["first"] }}`, "[]1F", d)
	renders(t, `{{ xs[0] }}{{ xs[-1] }}[{{ xs[5] }}][{{ xs[-9] }}]{{ xs[n] }}`, "13[][]2", d)
	renders(t, `{{ s.size }}{{ s.first }}{{ s.last }}{{ xs.size }}{{ xs.first }}{{ xs.last }}`, "5ho313", d)
	renders(t, `{{ o.size }}{{ o.first }}{{ p.size }}{{ p.first }}{{ p.last }}`, "SF2x1y2", d)
	renders(t, `[{{ e.first }}][{{ e.last }}][{{ es.first }}]{{ es.size }}`, "[][][]0", data(`{"e": [], "es": ""}`))
	// The order of lookup (ADR-004 D6 item 1): block, template, counter,
	// global.
	renders(t, `{{ x }}{% increment x %}{{ x }}{% assign x = "tpl" %}{{ x }}{% for x in (7..7) %}{{ x }}{% endfor %}{{ x }}`,
		"global01tpl7tpl", data(`{"x": "global"}`))
}

// TestShopifyReadings: what the gem does where the grammar reads two
// ways or a collection is not a list. The one "-" of "{{-}}" trims both
// sides, and a string is a loop's one item whatever its offset and limit
// (Utils.slice_collection).
func TestShopifyReadings(t *testing.T) {
	testenv.Home(t)
	d := data(`{"s": "abc"}`)
	renders(t, "foo {{-}} bar|a {{- -}} b", "foobar|ab", d)
	renders(t, `{% for c in s limit: 0 %}{{ c }}{% else %}none{% endfor %}|{% for c in s offset: 5 %}{{ c }}{% endfor %}`, "abc|abc", d)
	renders(t, `{% tablerow c in s limit: 0 %}{{ c }}{% endtablerow %}`, "<tr class=\"row1\">\n<td class=\"col1\">abc</td></tr>\n", d)
}

// TestObjectOutput: an object written as text is refused, naming json;
// an array writes its elements' texts concatenated (ADR-004 D4 item 4).
func TestObjectOutput(t *testing.T) {
	testenv.Home(t)
	for _, src := range []string{`{{ o }}`, `{% echo o %}`, `{{ xs }}`, `{{ 1 | object }}`} {
		e := renderRefused(t, src, CodeObjectOutput, data(`{"o": {"a": 1}, "xs": [1, {"a": 1}]}`))
		if e.Args["use"] != "json" {
			t.Errorf("%s: Args %v, want use json", src, e.Args)
		}
	}
	renders(t, `{{ xs }}|{{ "a" | pair }}`, "1a2.5true|aa", data(`{"xs": [1, ["a", 2.5], null, true]}`))
}

// TestNumberText: an integer's digits; a float as Ruby's Float#to_s, at
// the boundaries 1e-4 and 1e15; data numbers by value, literals by
// spelling (ADR-004 D4 items 2 and 4).
func TestNumberText(t *testing.T) {
	testenv.Home(t)
	for f, want := range map[float64]string{
		3.0: "3.0", 0.0001: "0.0001", 0.00001: "1.0e-05", 0.00012: "0.00012", 1e14: "100000000000000.0",
		1e15: "1.0e+15", 123456789012345.0: "123456789012345.0", 999999999999999.9: "999999999999999.9",
		0: "0.0", math.Copysign(0, -1): "-0.0", 1.5e300: "1.5e+300", -2.5: "-2.5", 0.1: "0.1",
		1.25e-7: "1.25e-07", -1e100: "-1.0e+100", 5e-324: "5.0e-324", 2.5e15: "2.5e+15",
	} {
		if got := floatText(f); got != want {
			t.Errorf("floatText(%v) = %q, want %q", f, got, want)
		}
	}
	renders(t, `{{ 4.0 }} {{ 4 }} {{ -3 }} {{ 0.5 }} {{ 007 }}`, "4.0 4 -3 0.5 7")
	renders(t, `{{ a }} {{ b }} {{ c }} {{ d }} {{ e }} {{ f }} {{ g }}`,
		"1 2.5 1.0e+300 9.007199254740992e+15 9007199254740991 0 100000",
		data(`{"a": 1.0, "b": 2.5, "c": 1e300, "d": 9007199254740993, "e": 9007199254740991, "f": -0.0, "g": 1e5}`))
	for _, c := range []struct {
		lit  string
		kind string
	}{{"1.0", "integer"}, {"1e2", "integer"}, {"2.5", "float"}, {"9007199254740992", "float"}, {"-0", "integer"}} {
		n, err := value.ParseNumber(c.lit)
		if err != nil {
			t.Fatal(err)
		}
		if got := kindName(Data(n)); got != c.kind {
			t.Errorf("Data(%s) is a template %s, want %s", c.lit, got, c.kind)
		}
	}
}
