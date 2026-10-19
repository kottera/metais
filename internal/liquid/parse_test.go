// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package liquid

import (
	"fmt"
	"strings"
	"testing"

	"github.com/kottera/metais/internal/liquid/ast"
	"github.com/kottera/metais/internal/testenv"
)

// parse parses text with the test dialect and default limits.
func parse(t *testing.T, d *Dialect, text string) (*Template, []*Error) {
	t.Helper()
	return Parse(d, Source{Name: "t.liquid", Text: []byte(text)}, ParseLimits{})
}

// mustParse parses text and fails the test on any error.
func mustParse(t *testing.T, text string) *Template {
	t.Helper()
	tpl, errs := parse(t, goldenDialect(t), text)
	if len(errs) > 0 {
		t.Fatalf("Parse(%q): %v", text, errs)
	}
	return tpl
}

// refused parses text, requires exactly the codes want in order, and
// returns the errors.
func refused(t *testing.T, d *Dialect, text string, want ...Code) []*Error {
	t.Helper()
	tpl, errs := parse(t, d, text)
	var got []Code
	for _, e := range errs {
		got = append(got, e.Code)
	}
	if fmt.Sprint(got) != fmt.Sprint(want) {
		t.Errorf("Parse(%q) gave %v (%v), want %v", text, got, errs, want)
	}
	if tpl != nil {
		t.Errorf("Parse(%q) returned a template beside its errors", text)
	}
	return errs
}

// TestCoreTagsParse holds every tag of ADR-004 D2 to the grammar, in each
// of its forms.
func TestCoreTagsParse(t *testing.T) {
	testenv.Home(t)
	for _, text := range []string{
		`{% if a %}x{% elsif b %}y{% else %}z{% endif %}`,
		`{% unless a %}x{% elsif b %}y{% else %}z{% endunless %}`,
		`{% case a %}{% when 1, 2 or 3 %}x{% when "s" %}y{% else %}z{% endcase %}`,
		`{% for x in xs reversed limit: 2 offset: 1 %}{{ x }}{% break %}{% continue %}{% else %}e{% endfor %}`,
		`{% for x in (1..n) offset: continue %}{{ forloop.index }}{% endfor %}`,
		`{% cycle "a", "b" %}{% cycle "g": "a", "b" %}`,
		`{% tablerow x in xs cols: 2 limit: 4 offset: 1 %}{{ x }}{% endtablerow %}`,
		`{% assign a = b | upcase %}{% capture c %}x{% endcapture %}`,
		`{% increment n %}{% decrement n %}`,
		`{% ifchanged %}{{ x }}{% endifchanged %}`,
		`{% echo a | upcase %}`,
		"{% liquid\n  assign a = 1\n  if a\n    echo a\n  endif\n  # a comment\n%}",
		`{% comment %}{% if %}{% comment %}{% endcomment %}{% raw %}{% endcomment %}{% endraw %}{% endcomment %}`,
		`{% # inline %}{%# tight %}`,
		`{% doc %}{{ not parsed }}{% enddoc %}`,
		`{% raw %}{{ not parsed }}{% if {% endraw %}`,
		`{% render "p.liquid" %}{% render "p.liquid", a: 1 %}{% render "p.liquid" with x as y %}{% render "p.liquid" for xs as x, k: v %}`,
		`{{ a.b[0]["c d"].size | default: "x" | join: ", " }}`,
		`{{ self }}{{ self.x }}`, // self is an ordinary name (ADR-004 D1 item 5)
		`{{ }}{% echo %}`,
	} {
		mustParse(t, text)
	}
}

// TestCoreTagList compares the grammar's tags with ADR-004 D2's table, so
// the two cannot drift apart (ADR-010 D4 item 2).
func TestCoreTagList(t *testing.T) {
	testenv.Home(t)
	adr004 := "# assign break capture case comment continue cycle decrement doc echo for if ifchanged increment liquid raw render tablerow unless"
	if got := strings.Join(CoreTags(), " "); got != adr004 {
		t.Errorf("CoreTags() = %q, want ADR-004 D2's %q", got, adr004)
	}
	for _, name := range CoreTags() {
		_, errs := parse(t, goldenDialect(t), "{% "+name+" %}")
		for _, e := range errs {
			if strings.Contains(e.Detail, "unknown tag") {
				t.Errorf("core tag %q parsed as unknown", name)
			}
		}
	}
}

// TestInclude pins ADR-004 D1 item 5: include is refused at parse, naming
// render, and the parse goes on to report what follows it.
func TestInclude(t *testing.T) {
	testenv.Home(t)
	d := goldenDialect(t)
	errs := refused(t, d, `ab{% include "p" %}{% include 'q', x: 1 %}{{ x | nope }}`, CodeInclude, CodeInclude, CodeUnknownFilter)
	if len(errs) == 3 {
		if errs[0].Args["use"] != "render" || errs[0].Col != 3 || errs[1].Col != 20 {
			t.Errorf("include errors %v, want use=render at columns 3 and 20", errs)
		}
	}
	// Control: a template without include parses.
	mustParse(t, `ab{% render "p" %}`)
}

// TestRefusals holds each refusal of ADR-004 D1, D2 and D16 to its code.
func TestRefusals(t *testing.T) {
	testenv.Home(t)
	d := goldenDialect(t)
	for _, c := range []struct {
		name, text string
		want       []Code
	}{
		{"unknown tag", `{% nosuchthing %}`, []Code{CodeSyntax}},
		{"unknown filter in a dead branch", `{% if false %}{{ x | nosuch }}{% endif %}`, []Code{CodeUnknownFilter}},
		{"every unknown filter", `{{ x | a1 }}{{ x | a2 }}`, []Code{CodeUnknownFilter, CodeUnknownFilter}},
		{"reversed after a named parameter", `{% for x in xs limit: 2 reversed %}{% endfor %}`, []Code{CodeSyntax}},
		{"a parameter twice", `{% for x in xs limit: 2 limit: 3 %}{% endfor %}`, []Code{CodeSyntax}},
		{"a path that begins with an index", `{{ ["a b"] }}`, []Code{CodeSyntax}},
		{"a computed root name", `{% if [k].x %}{% endif %}`, []Code{CodeSyntax}},
		{"non-literal render name", `{% render name %}`, []Code{CodePartialName}},
		{"render as without with", `{% render "p" as x %}`, []Code{CodeSyntax}},
		{"U+0000", "a\x00b", []Code{CodeSyntax}},
		{"not UTF-8", "a\xffb", []Code{CodeNotUTF8}},
		{"unclosed block", `{% if a %}x`, []Code{CodeSyntax}},
		{"stray end tag", `{% endif %}`, []Code{CodeSyntax}},
		{"mismatched end tag", `{% if a %}{% endfor %}`, []Code{CodeSyntax}},
		{"else after else", `{% if a %}{% else %}{% else %}{% endif %}`, []Code{CodeSyntax}},
		{"elsif after else", `{% if a %}{% else %}{% elsif b %}{% endif %}`, []Code{CodeSyntax}},
		{"else markup", `{% if a %}{% else b %}{% endif %}`, []Code{CodeSyntax}},
		{"break markup", `{% for x in y %}{% break 1 %}{% endfor %}`, []Code{CodeSyntax}},
		{"output not closed", `{{ a }`, []Code{CodeSyntax}},
		{"output with a tag inside", `{{ a {% b %}`, []Code{CodeSyntax}},
		{"tag not closed", `{% if a `, []Code{CodeSyntax}},
		{"tag without a name", `{% %}`, []Code{CodeSyntax}},
		{"string escapes are not escapes", `{{ "a\"b" }}`, []Code{CodeSyntax}},
		{"no arithmetic operator", `{{ 1 + 2 }}`, []Code{CodeSyntax}},
		{"not is no operator", `{% if not a %}{% endif %}`, []Code{CodeSyntax}},
		{"filter in a condition", `{% if a | upcase %}{% endif %}`, []Code{CodeSyntax}},
		{"two values in an output", "{{ a \n b }}", []Code{CodeSyntax}},
		{"dot then digit", `{{ a.0 }}`, []Code{CodeSyntax}},
		{"assign to a literal", `{% assign empty = 1 %}`, []Code{CodeSyntax}},
		{"assign to a non-identifier", `{% assign 123 = 1 %}`, []Code{CodeSyntax}},
		{"nested doc", `{% doc %}{% doc %}{% enddoc %}`, []Code{CodeSyntax}},
		{"doc markup", `{% doc x %}{% enddoc %}`, []Code{CodeSyntax}},
		{"unclosed raw", `{% raw %}x`, []Code{CodeSyntax}},
		{"unclosed comment", `{% comment %}{% comment %}{% endcomment %}`, []Code{CodeSyntax}},
		{"unclosed raw in a comment", `{% comment %}{% raw %}{% endcomment %}`, []Code{CodeSyntax}},
		{"inline comment without # on a line", "{% # a\n b %}", []Code{CodeSyntax}},
		{"liquid closing an outer block", "{% if a %}{% liquid endif %}", []Code{CodeSyntax}},
		{"liquid lines end at line feeds", "{% liquid\rif a\recho a\rendif %}", []Code{CodeSyntax}},
		{"integer literal beyond 2^53-1", `{{ 9007199254740992 }}`, []Code{CodeNumberRange}},
		{"trailing comma in a filter", `{{ a | join: "," , }}`, []Code{CodeSyntax}},
	} {
		t.Run(c.name, func(t *testing.T) { refused(t, d, c.text, c.want...) })
	}
	// Controls: the nearest accepted forms.
	for _, text := range []string{
		`{% for x in xs reversed limit: 2 %}{% endfor %}`,
		`{% if a %}{% elsif b %}{% else %}{% endif %}`,
		`{{ 9007199254740991 }}{{ -9007199254740991 }}`,
		`{% assign empty_ish = 1 %}`,
		"{% # a\n # b %}",
		`{{ "a\" }}`, // the string a\, with no escape in it
	} {
		mustParse(t, text)
	}
}

// TestReservedName pins ADR-004 D7 item 8: a template-scoped binding of a
// reserved name is refused; a block- or partial-scoped one is not.
func TestReservedName(t *testing.T) {
	testenv.Home(t)
	d, err := goldenDialect(t).Extend(Spec{Reserved: []string{"inputs", "forloop"}})
	if err != nil {
		t.Fatal(err)
	}
	for _, text := range []string{
		`{% assign inputs = 1 %}`,
		`{% capture forloop %}{% endcapture %}`,
		`{% increment inputs %}`,
		`{% decrement inputs %}`,
	} {
		errs := refused(t, d, text, CodeReservedName)
		if len(errs) == 1 && errs[0].Args["name"] == "" {
			t.Errorf("%q: the error names no name", text)
		}
	}
	for _, text := range []string{
		`{% for inputs in xs %}{% endfor %}`,
		`{% tablerow inputs in xs %}{% endtablerow %}`,
		`{% render "p" for xs as inputs, forloop: 1 %}`,
		`{% assign inputs_x = 1 %}`,
	} {
		if _, errs := parse(t, d, text); len(errs) > 0 {
			t.Errorf("Parse(%q): %v", text, errs)
		}
	}
	// Control: without the reservation, the same assign parses.
	if _, errs := parse(t, goldenDialect(t), `{% assign inputs = 1 %}`); len(errs) > 0 {
		t.Errorf("an unreserved name is refused: %v", errs)
	}
}

// TestNesting pins the template-nesting bound (ADR-009, ADR-004 D8 item
// 6): as many block tags open as the bound allows parse, and one more is
// refused with CodeLimit, at the tag that opens it.
func TestNesting(t *testing.T) {
	testenv.Home(t)
	d := goldenDialect(t)
	nest := func(n int, open, close string) string {
		return strings.Repeat(open, n) + strings.Repeat(close, n)
	}
	for _, c := range []struct {
		name        string
		open, close string
	}{
		{"if", "{% if a %}", "{% endif %}"},
		{"for", "{% for x in y %}", "{% endfor %}"},
		{"capture", "{% capture c %}", "{% endcapture %}"},
		{"unless", "{% unless a %}", "{% endunless %}"},
	} {
		t.Run(c.name, func(t *testing.T) {
			for _, lim := range []int{0, 3} {
				bound := lim
				if bound == 0 {
					bound = DefaultNesting
				}
				src := Source{Name: "n", Text: []byte(nest(bound, c.open, c.close))}
				if _, errs := Parse(d, src, ParseLimits{Nesting: lim}); len(errs) > 0 {
					t.Errorf("%d open at once, bound %d: %v", bound, bound, errs)
				}
				src.Text = []byte(nest(bound+1, c.open, c.close))
				_, errs := Parse(d, src, ParseLimits{Nesting: lim})
				if len(errs) != 1 || errs[0].Code != CodeLimit || errs[0].Args["limit"] != "template-nesting" {
					t.Fatalf("%d open at once, bound %d: %v, want one template-nesting limit", bound+1, bound, errs)
				}
				if want := bound*len(c.open) + 1; errs[0].Col != want {
					t.Errorf("located at column %d, want %d, the tag beyond the bound", errs[0].Col, want)
				}
			}
		})
	}
	// Liquid tags nest too, and so do brackets and ranges in one
	// expression, each bounded.
	deep := "{% liquid " + strings.Repeat("liquid ", 5) + "echo 1 %}"
	if _, errs := Parse(d, Source{Text: []byte(deep)}, ParseLimits{Nesting: 3}); len(errs) != 1 || errs[0].Code != CodeLimit {
		t.Errorf("nested liquid tags: %v, want a limit", errs)
	}
	brackets := "{{ a" + strings.Repeat("[a", 4) + strings.Repeat("]", 4) + " }}"
	if _, errs := Parse(d, Source{Text: []byte(brackets)}, ParseLimits{Nesting: 4}); len(errs) > 0 {
		t.Errorf("4 brackets, bound 4: %v", errs)
	}
	brackets = "{{ a" + strings.Repeat("[a", 5) + strings.Repeat("]", 5) + " }}"
	if _, errs := Parse(d, Source{Text: []byte(brackets)}, ParseLimits{Nesting: 4}); len(errs) != 1 || errs[0].Code != CodeLimit {
		t.Errorf("5 brackets, bound 4: %v, want a limit", errs)
	}
	ranges := "{{ " + strings.Repeat("(", 5) + "1" + strings.Repeat("..1)", 5) + " }}"
	if _, errs := Parse(d, Source{Text: []byte(ranges)}, ParseLimits{Nesting: 4}); len(errs) != 1 || errs[0].Code != CodeLimit {
		t.Errorf("5 ranges, bound 4: %v, want a limit", errs)
	}
}

// TestTemplateSize pins ParseLimits.Size: a template one byte over is
// refused before anything else is read.
func TestTemplateSize(t *testing.T) {
	testenv.Home(t)
	d := goldenDialect(t)
	text := []byte("{{ a }}xyz")
	if _, errs := Parse(d, Source{Text: text}, ParseLimits{Size: len(text)}); len(errs) > 0 {
		t.Errorf("at the bound: %v", errs)
	}
	_, errs := Parse(d, Source{Text: append(text, '\xff')}, ParseLimits{Size: len(text)})
	if len(errs) != 1 || errs[0].Code != CodeLimit || errs[0].Args["limit"] != "template-size" {
		t.Errorf("one byte over: %v, want one template-size limit", errs)
	}
}

// TestOneSyntaxError pins ADR-004 D16 item 4: the parse stops at its
// first syntax error, having reported every other error before it.
func TestOneSyntaxError(t *testing.T) {
	testenv.Home(t)
	errs := refused(t, goldenDialect(t), `{{ a | nope }}{% bad %}{{ b | nope2 }}{% worse %}`, CodeUnknownFilter, CodeSyntax)
	if len(errs) == 2 && errs[1].Col != 15 {
		t.Errorf("the syntax error is at column %d, want 15", errs[1].Col)
	}
}

// TestErrorOrder pins that errors come in source order, a block never
// closed being located at its opening tag.
func TestErrorOrder(t *testing.T) {
	testenv.Home(t)
	errs := refused(t, goldenDialect(t), `{{ a | n1 }}{% if a %}{{ b | n2 }}`, CodeUnknownFilter, CodeSyntax, CodeUnknownFilter)
	for i := 1; i < len(errs); i++ {
		if errs[i-1].Offset > errs[i].Offset {
			t.Errorf("error %d at %d after error %d at %d", i, errs[i].Offset, i-1, errs[i-1].Offset)
		}
	}
}

// TestErrorLocation pins ADR-004 D16 item 2: an error names the line and
// column of the "{{" or "{%" opening its token, the column in code
// points, and inside a liquid tag the inner line's first non-blank
// character.
func TestErrorLocation(t *testing.T) {
	testenv.Home(t)
	d := goldenDialect(t)
	for _, c := range []struct {
		text      string
		line, col int
	}{
		{"ab\n{{ a | nope }}", 2, 1},
		{"é€😀{% bad %}", 1, 4},
		{"a\r\nb\rc{% bad %}", 3, 2},
		{"{% if a == %}", 1, 1},
		{"x\n{% liquid\n  assign a = 1\n    echo a | nope\n%}", 4, 5},
		{"{%- liquid\nbad\n-%}", 2, 1},
	} {
		_, errs := parse(t, d, c.text)
		if len(errs) == 0 {
			t.Errorf("%q parses", c.text)
			continue
		}
		if e := errs[0]; e.Line != c.line || e.Col != c.col || e.Source != "t.liquid" {
			t.Errorf("%q: error at %s:%d:%d, want t.liquid:%d:%d", c.text, e.Source, e.Line, e.Col, c.line, c.col)
		}
	}
}

// TestWhitespaceControl pins ADR-004 D3 item 1's markers as the text nodes
// record them: "-" trims the text on its side, and nothing else does.
func TestWhitespaceControl(t *testing.T) {
	testenv.Home(t)
	tpl := mustParse(t, " a \n{{- x -}}\n b {%- if y -%} c {% endif %} d {%raw-%} r {%-endraw%} e ")
	var texts []string
	ast.Walk(tpl.Root, func(n ast.Node) bool {
		if tx, ok := n.(*ast.Text); ok {
			texts = append(texts, fmt.Sprintf("%q:%v:%v=%q", tx.Text, tx.TrimLeft, tx.TrimRight, tx.Trimmed()))
		}
		return true
	})
	want := []string{
		`" a \n":false:true=" a"`,
		`"\n b ":true:true="b"`,
		`" c ":true:false="c "`,
		`" d ":false:false=" d "`,
		`" e ":false:false=" e "`,
	}
	if strings.Join(texts, "|") != strings.Join(want, "|") {
		t.Errorf("text nodes\n%s\nwant\n%s", strings.Join(texts, "\n"), strings.Join(want, "\n"))
	}
	var raw *ast.Raw
	ast.Walk(tpl.Root, func(n ast.Node) bool {
		if r, ok := n.(*ast.Raw); ok {
			raw = r
		}
		return true
	})
	if raw == nil || raw.Text != " r " {
		t.Errorf("raw body %+v, want \" r \" verbatim", raw)
	}
	// Bytes are kept as written: a BOM and CR LF reach the text unchanged
	// (ADR-004 D1 item 7).
	tpl = mustParse(t, "\uFEFFa\r\n{{ x }}\r\n")
	if got := tpl.Root.Nodes[0].(*ast.Text).Text; got != "\uFEFFa\r\n" {
		t.Errorf("first text %q, want the BOM and CR LF kept", got)
	}
}

// TestConditionGrouping pins ADR-004 D2: and and or group from the right.
func TestConditionGrouping(t *testing.T) {
	testenv.Home(t)
	for text, want := range map[string]string{
		"a or b and c":       "(a or (b and c))",
		"a and b or c":       "(a and (b or c))",
		"a == 1 or b and c":  "((a == 1) or (b and c))",
		"a contains 'x'":     "(a contains 'x')",
		"a <> b and c >= 2":  "((a <> b) and (c >= 2))",
		"a and b and c or d": "(a and (b and (c or d)))",
	} {
		c, err := ParseCondition(goldenDialect(t), Source{Text: []byte(text)}, ParseLimits{})
		if err != nil {
			t.Errorf("ParseCondition(%q): %v", text, err)
			continue
		}
		if got := condString(text, c); got != want {
			t.Errorf("ParseCondition(%q) = %s, want %s", text, got, want)
		}
	}
}

// condString writes a condition tree with every grouping explicit.
func condString(src string, c *ast.Cond) string {
	text := func(p ast.Primary) string { s := p.Pos(); return src[s.Start:s.End] }
	switch c.Kind {
	case ast.CondAnd:
		return "(" + condString(src, c.X) + " and " + condString(src, c.Y) + ")"
	case ast.CondOr:
		return "(" + condString(src, c.X) + " or " + condString(src, c.Y) + ")"
	case ast.CondCompare:
		return "(" + text(c.Left) + " " + string(c.Op) + " " + text(c.Right) + ")"
	}
	return text(c.Left)
}

// TestLiterals pins ADR-004 D4 item 2 and D1 item 8: a numeric literal's
// kind is its spelling's, and a string literal carries no escapes.
func TestLiterals(t *testing.T) {
	testenv.Home(t)
	for text, want := range map[string]string{
		`4`:      "integer 4",
		`-4`:     "integer -4",
		`007`:    "integer 7",
		`4.0`:    "float 4.0",
		`-0.50`:  "float -0.50",
		`"a\nb"`: `string a\nb`,
		`'it"s'`: `string it"s`,
		`nil`:    "nil",
		`null`:   "nil",
		`true`:   "true",
		`empty`:  "empty",
		`blank`:  "blank",
	} {
		e, err := ParseValue(goldenDialect(t), Source{Text: []byte(text)}, ParseLimits{})
		if err != nil {
			t.Errorf("ParseValue(%q): %v", text, err)
			continue
		}
		lit, ok := e.Base.(*ast.Literal)
		if !ok {
			t.Errorf("ParseValue(%q) = %T, want a literal", text, e.Base)
			continue
		}
		var got string
		switch lit.Kind {
		case ast.LitInteger, ast.LitFloat:
			kind := map[ast.LiteralKind]string{ast.LitInteger: "integer", ast.LitFloat: "float"}[lit.Kind]
			got = fmt.Sprintf("%s %v", kind, lit.Value)
			if n, ok := lit.Value.(interface{ Literal() string }); ok {
				got = kind + " " + n.Literal()
			}
		case ast.LitString:
			got = "string " + lit.Text
		case ast.LitNil:
			got = "nil"
		case ast.LitTrue:
			got = "true"
		case ast.LitEmpty:
			got = "empty"
		case ast.LitBlank:
			got = "blank"
		}
		if got != want {
			t.Errorf("ParseValue(%q) = %s, want %s", text, got, want)
		}
	}
	// A keyword followed by a segment is a path, not a literal.
	e, err := ParseValue(goldenDialect(t), Source{Text: []byte("nil.x")}, ParseLimits{})
	if err != nil {
		t.Fatal(err)
	}
	if p, ok := e.Base.(*ast.Path); !ok || p.Root != "nil" || len(p.Segments) != 1 {
		t.Errorf("nil.x = %#v, want a path rooted at nil", e.Base)
	}
}

// TestForloopName pins Shopify's forloop.name, the variable and the
// collection as its parser rebuilds it.
func TestForloopName(t *testing.T) {
	testenv.Home(t)
	tpl := mustParse(t, `{% for x in a . b[ "c" ] %}{% endfor %}{% for i in ( 1 .. n ) %}{% endfor %}`)
	var names []string
	for _, n := range tpl.Root.Nodes {
		names = append(names, n.(*ast.For).Name)
	}
	if got := strings.Join(names, " "); got != `x-a.b["c"] i-(1..n)` {
		t.Errorf("loop names %q", got)
	}
}
