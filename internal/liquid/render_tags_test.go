// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package liquid

import (
	"fmt"
	"testing"

	"metais.org/metais/internal/testenv"
)

// The tests below hold each tag of ADR-004 D2 to the text of its row.

// TestRenderIf: if / elsif / else, with and and or grouping from the
// right, the left operand evaluated first, and short-circuited.
func TestRenderIf(t *testing.T) {
	testenv.Home(t)
	const src = `{% if a %}x{% elsif b %}y{% else %}z{% endif %}`
	renders(t, src, "x", data(`{"a": true, "b": true}`))
	renders(t, src, "y", data(`{"a": false, "b": 1}`))
	renders(t, src, "z", data(`{"a": null, "b": false}`))
	// "a or b and c" is a or (b and c); "a and b or c" is a and (b or c).
	renders(t, `{% if true or false and false %}T{% else %}F{% endif %}`, "T")
	renders(t, `{% if false and false or true %}T{% else %}F{% endif %}`, "F")
	// and stops at a falsy left operand, so its right is never evaluated.
	renders(t, `{% if a and a.b %}x{% else %}y{% endif %}`, "y", data(`{"a": null}`))
	renderRefused(t, `{% if a.b %}x{% endif %}`, CodeNilProperty, data(`{"a": null}`))
	// or stops at a truthy one.
	renders(t, `{% if a or nope %}x{% endif %}`, "x", data(`{"a": 0}`))
	renderRefused(t, `{% if false or nope %}x{% endif %}`, CodeUndefinedVariable)
}

// TestRenderUnless: the first branch runs when its condition is falsy;
// each elsif after it as in if.
func TestRenderUnless(t *testing.T) {
	testenv.Home(t)
	const src = `{% unless a %}x{% elsif b %}y{% else %}z{% endunless %}`
	renders(t, src, "x", data(`{"a": false, "b": true}`))
	renders(t, src, "y", data(`{"a": true, "b": true}`))
	renders(t, src, "z", data(`{"a": true, "b": null}`))
}

// TestRenderCase: when takes several values separated by "," or "or";
// every when one of whose values equals the subject renders, once per
// value that does, and an else renders when no earlier when matched.
func TestRenderCase(t *testing.T) {
	testenv.Home(t)
	const src = `{% case n %}{% when 1, 2 %}low{% when 3 or 4 %}mid{% else %}high{% endcase %}`
	renders(t, src, "low", data(`{"n": 2}`))
	renders(t, src, "mid", data(`{"n": 4}`))
	renders(t, src, "high", data(`{"n": 9}`))
	renders(t, `{% case "a" %}{% when "a" %}1{% when "b", "a" %}2{% endcase %}`, "12")
	renders(t, `{% case "a" %}{% when "a", "a" %}x{% endcase %}`, "xx")
	renders(t, `{% case 1 %}{% else %}e{% when 1 %}w{% endcase %}`, "ew")
	renders(t, `{% case 1 %}{% when 1 %}w{% else %}e{% endcase %}`, "w")
	// What lies before the first clause is parsed and never rendered.
	renders(t, `{% case 1 %}before{% when 1 %}w{% endcase %}`, "w")
	// Integers and floats compare by value.
	renders(t, `{% case 1.0 %}{% when 1 %}eq{% endcase %}`, "eq")
	renderRefused(t, `{% case 1 %}{% when nope %}w{% endcase %}`, CodeUndefinedVariable)
}

// TestRenderFor: limit, offset and reversed, applied as offset, then
// limit, then reversed; ranges of integers; forloop; else; break and
// continue; offset: continue; objects in canonical order.
func TestRenderFor(t *testing.T) {
	testenv.Home(t)
	xs := data(`{"xs": [1, 2, 3, 4]}`)
	renders(t, `{% for x in xs %}{{ x }}{% endfor %}`, "1234", xs)
	renders(t, `{% for x in xs reversed limit: 2 offset: 1 %}{{ x }}{% endfor %}`, "32", xs)
	renders(t, `{% for x in xs limit: "2" %}{{ x }}{% endfor %}`, "12", xs)
	renders(t, `{% for x in xs limit: n %}{{ x }}{% endfor %}`, "1234", data(`{"xs": [1, 2, 3, 4], "n": null}`))
	renders(t, `{% for x in xs offset: -2 limit: 3 %}{{ x }}{% endfor %}`, "1", xs)
	renders(t, `{% for i in (1..3) %}{{ i }}{% endfor %}`, "123")
	renders(t, `{% for i in (3..1) %}{{ i }}{% else %}none{% endfor %}`, "none")
	renders(t, `{% for i in (a..b) %}{{ i }}{% endfor %}`, "-101", data(`{"a": -1, "b": "1"}`))
	renders(t, `{% for i in (2.7..4) %}{{ i }}{% endfor %}`, "234")

	renders(t, `{% for x in xs %}{{ forloop.index }}{{ forloop.index0 }}{{ forloop.rindex }}{{ forloop.rindex0 }}`+
		`{{ forloop.first }}{{ forloop.last }}{{ forloop.length }} {% endfor %}`,
		"1021truefalse2 2110falsetrue2 ", data(`{"xs": ["a", "b"]}`))
	renders(t, `{% for x in xs limit: 1 %}{{ forloop.name }}{% endfor %}`, "x-xs", xs)
	renders(t, `{% for i in (1..2) %}{% for j in (1..2) %}{{ forloop.parentloop.index }}{{ j }} {% endfor %}{% endfor %}`,
		"11 12 21 22 ")
	renders(t, `[{% for i in (1..1) %}{{ forloop.parentloop }}{% endfor %}]`, "[]")

	for _, d := range []string{`{"c": []}`, `{"c": null}`, `{"c": 5}`, `{"c": ""}`, `{"c": false}`} {
		renders(t, `{% for x in c %}x{% else %}none{% endfor %}`, "none", data(d))
	}
	renders(t, `{% for x in c %}[{{ x }}]{% endfor %}`, "[abc]", data(`{"c": "abc"}`))

	// An object is iterated as [name, value] pairs in canonical order: by
	// UTF-16 code units, so U+1F600 sorts before U+E000 (ADR-004 D4 item 5).
	renders(t, `{% for p in o %}{{ p[0] }}={{ p[1] }};{% endfor %}`, "a=4;b=2;\U0001F600=3;\uE000=1;",
		data(`{"o": {"\ue000": 1, "b": 2, "\ud83d\ude00": 3, "a": 4}}`))

	renders(t, `{% for i in (1..5) %}{% if i == 2 %}{% continue %}{% endif %}{% if i == 4 %}{% break %}{% endif %}{{ i }}{% endfor %}`, "13")
	renders(t, `{% for i in (1..5) limit: 2 %}{{ i }}{% endfor %}|{% for i in (1..5) offset: continue %}{{ i }}{% endfor %}`, "12|345")
	// The loop variable is block-scoped and shadows a global only inside.
	renders(t, `{% for item in (1..2) %}{{ item }}{% endfor %}{{ item }}`, "12g", data(`{"item": "g"}`))
	renderRefused(t, `{% for x in (1..2) %}{% endfor %}{{ x }}`, CodeUndefinedVariable)
	// assign inside a loop is template-scoped.
	renders(t, `{% for i in (1..3) %}{% assign last = i %}{% endfor %}{{ last }}`, "3")
}

// TestRenderForRefusals: a limit or offset that is not an integer is
// refused at render (render-time.txt's loop modifier lines), and so is a
// range bound that is not one.
func TestRenderForRefusals(t *testing.T) {
	testenv.Home(t)
	for _, c := range []struct {
		src, arg, kind string
	}{
		{`{% for i in (1..4) limit: 'foo' %}{% endfor %}`, "limit", "string"},
		{`{% for i in (1..4) offset: 'foo' %}{% endfor %}`, "offset", "string"},
		{`{% for i in (1..4) limit: xs %}{% endfor %}`, "limit", "array"},
		{`{% for i in (1..4) offset: xs %}{% endfor %}`, "offset", "array"},
		{`{% for i in (1..4) limit: 1.5 %}{% endfor %}`, "limit", "float"},
		{`{% for i in (1..4) limit: true %}{% endfor %}`, "limit", "boolean"},
	} {
		e := renderRefused(t, c.src, CodeArgument, data(`{"xs": [1, 2]}`))
		if e.Args["tag"] != "for" || e.Args["argument"] != c.arg || e.Args["kind"] != c.kind {
			t.Errorf("%s: Args %v, want tag for, argument %s, kind %s", c.src, e.Args, c.arg, c.kind)
		}
		if e.Line != 1 || e.Col != 1 {
			t.Errorf("%s: at %d:%d, want the for tag at 1:1", c.src, e.Line, e.Col)
		}
	}
	for _, src := range []string{`{% for i in (x..5) %}{% endfor %}`, `{{ (x..5) }}`, `{% for i in (1..xs) %}{% endfor %}`} {
		e := renderRefused(t, src, CodeArgument, data(`{"x": 2.3, "xs": [1]}`))
		if e.Args["argument"] != "range" {
			t.Errorf("%s: Args %v, want argument range", src, e.Args)
		}
	}
}

// TestRenderCycle: inside or outside a loop; an optional group keys the
// position.
func TestRenderCycle(t *testing.T) {
	testenv.Home(t)
	renders(t, `{% for i in (1..4) %}{% cycle "a", "b", "c" %}{% endfor %}`, "abca")
	renders(t, `{% cycle "x", "y" %}{% cycle "x", "y" %}{% cycle "x", "y" %}`, "xyx")
	renders(t, `{% cycle "g": "a", "b" %}{% cycle "g": "c", "d" %}{% cycle "h": "a", "b" %}`, "ada")
	renders(t, `{% cycle "1", "2" %}{% cycle "1", "2", "3" %}`, "11")
	renders(t, `{% cycle v, "r" %}{% cycle v, "r" %}`, "qq", data(`{"v": "q"}`))
	renders(t, `{% cycle "g": "1", "2", "3" %}{% cycle "g": "1", "2" %}{% cycle "g": "1" %}`, "121")
	renders(t, `{% cycle "g": "1", "2" %}{% cycle "g": "1", "2", "3" %}[{% cycle "g": "1" %}]`, "12[]")
	renders(t, `{% cycle g: "a", "b" %}{% assign g = "other" %}{% cycle g: "a", "b" %}`, "aa", data(`{"g": "one"}`))
	renderRefused(t, `{% cycle o %}`, CodeObjectOutput, data(`{"o": {"a": 1}}`))
}

// TestRenderTablerow: the markup exactly as ADR-004's Context gives it,
// cols, limit, offset and tablerowloop.
func TestRenderTablerow(t *testing.T) {
	testenv.Home(t)
	renders(t, `{% tablerow i in (1..3) cols: 2 %}{{ i }}{% endtablerow %}`,
		"<tr class=\"row1\">\n<td class=\"col1\">1</td><td class=\"col2\">2</td></tr>\n<tr class=\"row2\"><td class=\"col1\">3</td></tr>\n")
	renders(t, `{% tablerow i in (1..2) %}{{ i }}{% endtablerow %}`,
		"<tr class=\"row1\">\n<td class=\"col1\">1</td><td class=\"col2\">2</td></tr>\n")
	renders(t, `{% tablerow i in xs %}{{ i }}{% endtablerow %}`, "<tr class=\"row1\">\n</tr>\n", data(`{"xs": []}`))
	renders(t, `{% tablerow i in (1..3) cols: 2 %}{{ tablerowloop.col }}{{ tablerowloop.row }}{{ tablerowloop.col_first }}{% endtablerow %}`,
		"<tr class=\"row1\">\n<td class=\"col1\">11true</td><td class=\"col2\">21false</td></tr>\n<tr class=\"row2\"><td class=\"col1\">12true</td></tr>\n")
	renders(t, `{% tablerow i in (1..5) offset: 1 limit: 2 %}{{ i }}{% endtablerow %}`,
		"<tr class=\"row1\">\n<td class=\"col1\">2</td><td class=\"col2\">3</td></tr>\n")
	renders(t, `{% tablerow i in (1..4) cols: 2.6 %}{{ i }}{% endtablerow %}`,
		"<tr class=\"row1\">\n<td class=\"col1\">1</td><td class=\"col2\">2</td></tr>\n<tr class=\"row2\"><td class=\"col1\">3</td><td class=\"col2\">4</td></tr>\n")
	renders(t, `{% tablerow i in (1..3) %}{{ i }}{% break %}{% endtablerow %}`, "<tr class=\"row1\">\n<td class=\"col1\">1</td></tr>\n")
	renders(t, `[{% tablerow i in n %}x{% endtablerow %}]`, "[]", data(`{"n": null}`))
	// tablerow keeps its whitespace: the blank-body rule is not its.
	renders(t, `{% tablerow i in (1..1) %} {% endtablerow %}`, "<tr class=\"row1\">\n<td class=\"col1\"> </td></tr>\n")
	renderRefused(t, `{% tablerow i in (1..2) cols: xs %}{% endtablerow %}`, CodeArgument, data(`{"xs": [1]}`))
	renderRefused(t, `{% tablerow i in (1..2) cols: 0 %}{% endtablerow %}`, CodeArgument)
}

// TestRenderAssignCapture: both template-scoped (ADR-004 D7 item 8).
func TestRenderAssignCapture(t *testing.T) {
	testenv.Home(t)
	renders(t, `{% assign a = "x" | suffix: "y" %}{{ a }}`, "xy")
	renders(t, `{% capture c %}a{{ 1 }}b{% endcapture %}[{{ c }}]`, "[a1b]")
	renders(t, `{% if true %}{% capture c %}z{% endcapture %}{% endif %}{{ c }}`, "z")
	renders(t, `{% assign g = 2 %}{{ g }}`, "2", data(`{"g": 1}`))
}

// TestRenderCounters: increment writes and then adds one, decrement
// subtracts one and then writes; counters start at zero in their own
// namespace, which a template-scoped binding of the name shadows.
func TestRenderCounters(t *testing.T) {
	testenv.Home(t)
	renders(t, `{% increment c %}{% increment c %}{% decrement d %}{% decrement d %}{{ c }}`, "01-1-22")
	renders(t, `{% increment c %}{% decrement c %}`, "00")
	renders(t, `{% assign c = 10 %}{% increment c %}{{ c }}`, "010")
}

// TestRenderIfchanged writes its body only when it differs from the last
// one written.
func TestRenderIfchanged(t *testing.T) {
	testenv.Home(t)
	renders(t, `{% for x in xs %}{% ifchanged %}{{ x }}{% endifchanged %}{% endfor %}`, "aba",
		data(`{"xs": ["a", "a", "b", "b", "a"]}`))
}

// TestRenderEchoLiquid: echo is an output expression; liquid is one tag
// per line, echo writing.
func TestRenderEchoLiquid(t *testing.T) {
	testenv.Home(t)
	renders(t, `{% echo "a" | suffix: "b" %}`, "ab")
	renders(t, `[{% echo %}]`, "[]")
	renders(t, "{% liquid\nassign x = 2\nif x == 2\n  echo 'yes'\nendif\n%}", "yes")
	e := renderRefused(t, "{% liquid\nassign x = 2\n  echo nope\n%}", CodeUndefinedVariable)
	if e.Line != 3 || e.Col != 3 {
		t.Errorf("an error inside liquid is at %d:%d, want its line's first non-blank character, 3:3", e.Line, e.Col)
	}
}

// TestRenderCommentRaw: comment, # and doc render nothing; raw writes its
// body verbatim.
func TestRenderCommentRaw(t *testing.T) {
	testenv.Home(t)
	renders(t, `a{% comment %}x{{ y }}{% endcomment %}b{% # note %}c{% doc %}d{% enddoc %}e`, "abce")
	renders(t, `{% raw %}{{ x }}{% if %}{% endraw %}`, "{{ x }}{% if %}")
}

// TestRenderPartials: render's forms, its isolation, the globals visible
// in a partial, and the call chain of an error inside one (ADR-004 D8,
// D16 item 2).
func TestRenderPartials(t *testing.T) {
	testenv.Home(t)
	ps := partials(map[string]string{
		"p":        "{{ a }}-{{ b }}",
		"dir/card": "{{ card }}",
		"n":        "{{ n }}",
		"loop":     "{{ forloop.index }}{{ n }}{{ forloop.length }} ",
		"q":        "{{ a }}",
		"inc":      "{% increment c %}",
		"set":      "{% assign a = 2 %}",
		"brk":      "a{% break %}b",
		"g":        "{{ g }}",
		"i":        "{{ i }}",
		"plain":    "P",
	})
	renders(t, `{% render "plain" %}`, "P", ps)
	renders(t, `{% render "p", a: 1, b: "x" %}`, "1-x", ps)
	renders(t, `{% render "p" a: 1 b: "x" %}`, "1-x", ps)
	renders(t, `{% render "dir/card" with 5 %}`, "5", ps)
	renders(t, `{% render "n" with "v" as n %}`, "v", ps)
	renders(t, `{% render "loop" for xs as n %}`, "1a2 2b2 ", ps, data(`{"xs": ["a", "b"]}`))
	renders(t, `{% render "loop" for (1..2) as n %}`, "112 222 ", ps)
	renders(t, `{% render "n" for "s" as n %}`, "s", ps)
	renders(t, `[{% render "n" with nil as n %}]`, "[]", ps)
	// A partial sees the globals, and its parameter shadows one inside it.
	renders(t, `{% render "g" %}`, "G", ps, data(`{"g": "G"}`))
	renders(t, `{% render "g", g: "param" %}{{ g }}`, "paramG", ps, data(`{"g": "G"}`))
	// Isolation: no caller binding is visible, and nothing the partial
	// binds reaches the caller.
	e := renderRefused(t, `{% assign a = 1 %}{% render "q" %}`, CodeUndefinedVariable, ps)
	if e.Source != "q" || len(e.Chain) != 1 || e.Chain[0] != (Site{Source: "t.liquid", Line: 1, Col: 19, Offset: 18}) {
		t.Errorf("error inside a partial: %v, chain %v; want in q, called from t.liquid:1:19", e, e.Chain)
	}
	renderRefused(t, `{% for i in (1..2) %}{% render "i" %}{% endfor %}`, CodeUndefinedVariable, ps)
	renders(t, `{% increment c %}{% render "inc" %}{% increment c %}`, "001", ps)
	renders(t, `{% assign a = 1 %}{% render "set" %}{{ a }}`, "1", ps)
	renders(t, `{% for i in (1..2) %}{% render "brk" %}{% endfor %}`, "aa", ps)

	e = renderRefused(t, `x{% render "missing" %}`, "partial_not_found", ps)
	if e.Source != "t.liquid" || e.Col != 2 || len(e.Chain) != 0 {
		t.Errorf("a partial Partials does not find: %v; want it at the render tag, t.liquid:1:2", e)
	}
	chain := partials(map[string]string{"a": "{% render 'b' %}", "b": "x{% render 'c' %}", "c": "{{ nope }}"})
	e = renderRefused(t, `{% render "a" %}`, CodeUndefinedVariable, chain)
	want := []Site{{"t.liquid", 1, 1, 0}, {"a", 1, 1, 0}, {"b", 1, 2, 1}}
	if e.Source != "c" || fmt.Sprint(e.Chain) != fmt.Sprint(want) {
		t.Errorf("nested partial error: %v, chain %v; want in c, chain %v", e, e.Chain, want)
	}
	renderRefused(t, `{% render "plain" %}`, CodeInternal)
}

// TestPartialDepth holds the render depth to partial-depth (ADR-004 D8
// item 6): n nested render calls render, and one more is refused with
// the limit named and the chain of calls.
func TestPartialDepth(t *testing.T) {
	testenv.Home(t)
	// chainOf is n partials, each rendering the next, the last writing
	// "end": rendering the first makes n render calls active at once.
	chainOf := func(n int) option {
		srcs := map[string]string{}
		for i := 1; i < n; i++ {
			srcs[fmt.Sprintf("p%d", i)] = fmt.Sprintf("{%% render 'p%d' %%}", i+1)
		}
		srcs[fmt.Sprintf("p%d", n)] = "end"
		return partials(srcs)
	}
	renders(t, `{% render "p1" %}`, "end", chainOf(3), limits(RenderLimits{PartialDepth: 3}))
	e := renderRefused(t, `{% render "p1" %}`, CodeLimit, chainOf(4), limits(RenderLimits{PartialDepth: 3}))
	if e.Args["limit"] != "partial-depth" || e.Source != "p3" || len(e.Chain) != 3 {
		t.Errorf("one call beyond partial-depth: %v, chain %v; want partial-depth at p3 with three calls", e, e.Chain)
	}
	// With no PartialDepth, ADR-009's default of 32 holds.
	renders(t, `{% render "p1" %}`, "end", chainOf(DefaultPartialDepth))
	renderRefused(t, `{% render "p1" %}`, CodeLimit, chainOf(DefaultPartialDepth+1))
	// Recursion is allowed, and stopped at the bound.
	e = renderRefused(t, `{% render "self" %}`, CodeLimit, partials(map[string]string{"self": "x{% render 'self' %}"}))
	if len(e.Chain) != DefaultPartialDepth {
		t.Errorf("self-recursion stopped with %d calls in its chain, want %d", len(e.Chain), DefaultPartialDepth)
	}
}

// TestRenderWhitespace: each control marker removes the whitespace beside
// it; nothing else is trimmed; a byte-order mark and CR LF are written
// as the template has them (ADR-004 D1 item 7, D3).
func TestRenderWhitespace(t *testing.T) {
	testenv.Home(t)
	renders(t, `a {{- "b" -}} c`, "abc")
	renders(t, `a {%- if true -%} b {%- endif -%} c`, "abc")
	renders(t, "a \t\n\v\f\r{{- 'b' -}} \t\n\v\f\rc", "abc")
	renders(t, "a\n{% assign x = 1 %}\nb", "a\n\nb")
	renders(t, "\ufeffa\r\n{{ 1 }}\r\n", "\ufeffa\r\n1\r\n")
}

// TestRenderBlankBody: an if, unless, case or for all of whose bodies
// hold only whitespace and blank tags writes none of their whitespace
// (ADR-004 D3 item 3); tablerow keeps it.
func TestRenderBlankBody(t *testing.T) {
	testenv.Home(t)
	for _, c := range []struct{ src, want string }{
		{"a{% if true %}  \n  {% endif %}b", "ab"},
		{"a{% if true %} {% assign x = 1 %} {% else %} {% endif %}b", "ab"},
		{"a{% if true %} {{ 1 }} {% endif %}b", "a 1 b"},
		{"a{% if true %} {% else %}x{% endif %}b", "a b"},
		{"a{% unless false %} {% endunless %}b", "ab"},
		{"a{% case 1 %}{% when 1 %} {% endcase %}b", "ab"},
		{"a{% case 1 %}{% when 1 %} {{ 1 }} {% endcase %}b", "a 1 b"},
		{"a{% for i in (1..2) %} {% endfor %}b", "ab"},
		{"a{% for i in (1..0) %} {% else %} {% endfor %}b", "ab"},
		{"a{% if true %} {% if true %} {% endif %} {% endif %}b", "ab"},
		{"a{% if true %} {% capture c %}x{% endcapture %} {% endif %}b{{ c }}", "abx"},
		{"a{% if true %} {% raw %}r{% endraw %} {% endif %}b", "a r b"},
		{"a{% if true %} {% raw %}{% endraw %} {% endif %}b", "ab"},
		{"a{% if true %} {% comment %}c{% endcomment %} {% # c %} {% endif %}b", "ab"},
		{"a{% if true %} {% doc %}{% enddoc %} {% endif %}b", "ab"},
		{"a{% if true %} {% doc %}d{% enddoc %} {% endif %}b", "a  b"},
		{"a{% for i in (1..1) %} {% break %}{% endfor %}b", "a b"},
		// A blank ifchanged makes its if blank, and keeps its own text.
		{"a{% if true %} {% ifchanged %} {% endifchanged %} {% endif %}b", "a b"},
		{"a{% if true %} {% tablerow i in (1..0) %} {% endtablerow %} {% endif %}b", "a<tr class=\"row1\">\n</tr>\nb"},
	} {
		renders(t, c.src, c.want)
	}
}

// TestRenderAddedTag: a dialect's tag renders through TagContext — its
// arguments evaluated in scope, its bytes written unescaped, its work
// charged to the render's meter — and its refusal is located at it.
func TestRenderAddedTag(t *testing.T) {
	testenv.Home(t)
	m := NewMeter(testLimits)
	renders(t, `{% for s in xs %}{% say s %}{% endfor %}`, "a<", data(`{"xs": ["a", "<"]}`), meter(m), escapeHTML)
	if got := m.Used(WorkPartials); got != 2 {
		t.Errorf("the tag charged %d to the meter, want 2", got)
	}
	e := renderRefused(t, `x{% say 1 %}`, "tag_input")
	if e.Source != "t.liquid" || e.Col != 2 {
		t.Errorf("a tag's refusal: %v; want it at the tag, t.liquid:1:2", e)
	}
	renderRefused(t, `{% say nope %}`, CodeUndefinedVariable)
}

// TestRenderEscape: with an Escape, the text of every value an output, an
// echo or a cycle writes is escaped unless safe; the template's text, raw
// and tablerow's markup are not (ADR-004 D9).
func TestRenderEscape(t *testing.T) {
	testenv.Home(t)
	d := data(`{"s": "<a&'\">"}`)
	const esc = "&lt;a&amp;&#39;&quot;&gt;"
	renders(t, `<b>{{ s }}</b>{% echo s %}{% cycle s %}{% raw %}<{% endraw %}`, "<b>"+esc+"</b>"+esc+esc+"<", d, escapeHTML)
	renders(t, `{{ s | mark }}|{{ s | mark | suffix: "" }}`, "<a&'\">|"+esc, d, escapeHTML)
	renders(t, `{% capture c %}{{ s }}{% endcapture %}{{ c }}{% assign e = c %}{{ e }}`, esc+esc, d, escapeHTML)
	renders(t, `{{ c.size }}`, "1", data(`{"c": "<"}`), escapeHTML)
	renders(t, `{{ s }}`, "<a&'\">", d)
	// The core catalog's escape, its alias h and escape_once mark their
	// result safe, so it is not escaped twice; a later filter drops it.
	core := withDialect(Core())
	renders(t, `{{ s | escape }}|{{ s | h }}|{{ "&amp;<" | escape_once }}|{{ s | escape | append: "" }}`,
		esc+"|"+esc+"|&amp;&lt;|"+"&amp;lt;a&amp;amp;&amp;#39;&amp;quot;&amp;gt;", d, core, escapeHTML)
	renders(t, `{{ s | escape }}`, esc, d, core)
	renders(t, `{% tablerow x in xs %}{{ x }}{% endtablerow %}`, "<tr class=\"row1\">\n<td class=\"col1\">&lt;</td></tr>\n",
		data(`{"xs": ["<"]}`), escapeHTML)
}

// TestEscapeHTML: EscapeHTML is ADR-004 D9 item 1's mapping, each of its
// five characters and nothing else, appended to what dst holds.
func TestEscapeHTML(t *testing.T) {
	testenv.Home(t)
	got := string(EscapeHTML([]byte("x:"), "a&b<c>d\"e'f é "))
	if want := "x:a&amp;b&lt;c&gt;d&quot;e&#39;f é "; got != want {
		t.Errorf("EscapeHTML = %q, want %q", got, want)
	}
}
