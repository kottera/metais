// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package liquid

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/kottera/metais/internal/liquid/ast"
	"github.com/kottera/metais/internal/testenv"
)

// everyConstruct uses every node type the parser builds.
const everyConstruct = `a{{ x.y[0]["k"] | f: 1, k: (1..2) }}
{% if a == 1 and b or c %}{% elsif d %}{% else %}{% endif %}
{% unless e %}{% else %}{% endunless %}
{% case f %} {% when 1, 2 %}{% else %}{% endcase %}
{% for i in xs limit: n offset: 2 %}{% break %}{% continue %}{% else %}{% endfor %}
{% tablerow r in rs cols: 2 %}{% endtablerow %}
{% assign g = "s" | f %}{% capture h %}{% endcapture %}{% increment i %}
{% cycle "grp": "a", "b" %}{% echo j %}{% ifchanged %}{% endifchanged %}
{% comment %}c{% endcomment %}{% # n %}{% doc %}d{% enddoc %}{% raw %}r{% endraw %}
{% liquid
  echo k
%}{% render "p" for ps as q, w: 1 %}{% added 1, z: 2 %}`

// allNodeTypes is every concrete node type of package ast.
var allNodeTypes = strings.Fields(`*ast.Arg *ast.Assign *ast.Block *ast.Branch
	*ast.Break *ast.Capture *ast.Case *ast.Comment *ast.Cond *ast.Continue
	*ast.Counter *ast.Cycle *ast.Echo *ast.Expr *ast.Filter *ast.For *ast.If
	*ast.Ifchanged *ast.Liquid *ast.Literal *ast.Output *ast.Path *ast.Range
	*ast.Raw *ast.Render *ast.Segment *ast.Tablerow *ast.Tag *ast.Text
	*ast.Unless *ast.When`)

// constructDialect is the test dialect with a tag named added.
func constructDialect(t testing.TB) *Dialect {
	t.Helper()
	d, err := goldenDialect(t).Extend(Spec{Tags: []TagDef{&recordTag{name: "added"}}, Filters: []FilterDef{stubFilter("f")}})
	if err != nil {
		t.Fatal(err)
	}
	return d
}

// TestWalkReachesEveryNode pins that Walk visits every node type the
// parser builds, so ADR-007's cross-reference sees the whole tree.
func TestWalkReachesEveryNode(t *testing.T) {
	testenv.Home(t)
	tpl, errs := Parse(constructDialect(t), Source{Text: []byte(everyConstruct)}, ParseLimits{})
	if len(errs) > 0 {
		t.Fatal(errs)
	}
	seen := map[string]bool{}
	ast.Walk(tpl.Root, func(n ast.Node) bool {
		seen[fmt.Sprintf("%T", n)] = true
		return true
	})
	var missing []string
	for _, ty := range allNodeTypes {
		if !seen[ty] {
			missing = append(missing, ty)
		}
	}
	if len(missing) > 0 {
		t.Errorf("Walk never visited %v", missing)
	}
	// Control: returning false stops the descent.
	n := 0
	ast.Walk(tpl.Root, func(ast.Node) bool { n++; return false })
	if n != 1 {
		t.Errorf("Walk visited %d nodes after visit returned false, want 1", n)
	}
}

// TestSpans pins that every node carries a span: within its source, with
// its line and column those of its start, nested within its parent's,
// and visited in source order.
func TestSpans(t *testing.T) {
	testenv.Home(t)
	tpl, errs := Parse(constructDialect(t), Source{Text: []byte(everyConstruct)}, ParseLimits{})
	if len(errs) > 0 {
		t.Fatal(errs)
	}
	for _, v := range spanViolations(everyConstruct, tpl.Root) {
		t.Error(v)
	}
	// Spot checks: the text of a few spans.
	text := func(n ast.Node) string { s := n.Pos(); return everyConstruct[s.Start:s.End] }
	var got []string
	ast.Walk(tpl.Root, func(n ast.Node) bool {
		switch n.(type) {
		case *ast.Filter, *ast.Range, *ast.Segment, *ast.Render:
			got = append(got, text(n))
		}
		return true
	})
	want := []string{`.y`, `[0]`, `["k"]`, `| f: 1, k: (1..2)`, `(1..2)`, `| f`, `{% render "p" for ps as q, w: 1 %}`}
	if !slices.Equal(got, want) {
		t.Errorf("spans %q, want %q", got, want)
	}
}

// spanViolations checks every span under root against src.
func spanViolations(src string, root ast.Node) []string {
	x := newLineIndex(src)
	var bad []string
	var lastStart int
	ast.Walk(root, func(n ast.Node) bool {
		s := n.Pos()
		if s.Start < 0 || s.Start > s.End || s.End > len(src) {
			bad = append(bad, fmt.Sprintf("%T span [%d,%d) outside %d bytes", n, s.Start, s.End, len(src)))
			return false
		}
		if l, c := x.at(s.Start); l != s.Line || c != s.Col {
			bad = append(bad, fmt.Sprintf("%T at %d is %d:%d, want %d:%d", n, s.Start, s.Line, s.Col, l, c))
		}
		if s.Start < lastStart {
			bad = append(bad, fmt.Sprintf("%T at %d visited after %d", n, s.Start, lastStart))
		}
		lastStart = s.Start
		return true
	})
	return bad
}

// TestBinders pins that the binding constructs are marked: each says what
// it binds and in which scope (ADR-004 D7 item 8), and nothing else
// claims to bind.
func TestBinders(t *testing.T) {
	testenv.Home(t)
	tpl, errs := Parse(constructDialect(t), Source{Text: []byte(everyConstruct)}, ParseLimits{})
	if len(errs) > 0 {
		t.Fatal(errs)
	}
	var got []string
	ast.Walk(tpl.Root, func(n ast.Node) bool {
		if b, ok := n.(ast.Binder); ok {
			for _, bd := range b.Bindings() {
				s := bd.Pos
				got = append(got, fmt.Sprintf("%T %s %s %v %q", n, bd.Name, bd.Scope, bd.Counter, everyConstruct[s.Start:s.End]))
			}
		}
		return true
	})
	want := []string{
		`*ast.For i block false "i"`,
		`*ast.Tablerow r block false "r"`,
		`*ast.Assign g template false "g"`,
		`*ast.Capture h template false "h"`,
		`*ast.Counter i template true "i"`,
		`*ast.Render q partial false "q"`,
		`*ast.Render w partial false "w: 1"`,
	}
	if !slices.Equal(got, want) {
		t.Errorf("bindings\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
	// A render with a value and no alias binds the partial's base name.
	tpl = mustParse(t, `{% render "cards/card.liquid" with c %}`)
	r := tpl.Root.Nodes[0].(*ast.Render)
	if b := r.Bindings(); len(b) != 1 || b[0].Name != "card.liquid" {
		t.Errorf("render with, no alias, binds %v", b)
	}
	// A plain render binds nothing.
	tpl = mustParse(t, `{% render "p" %}`)
	if b := tpl.Root.Nodes[0].(*ast.Render).Bindings(); len(b) != 0 {
		t.Errorf("plain render binds %v", b)
	}
}

// TestSpanViolationsControl shows the span check, which TestSpans and
// FuzzLiquidParse rely on, flagging the spans it exists to flag.
func TestSpanViolationsControl(t *testing.T) {
	testenv.Home(t)
	src := "ab\ncd"
	for name, n := range map[string]ast.Node{
		"past the end": &ast.Text{Span: ast.Span{Start: 0, End: 6, Line: 1, Col: 1}},
		"inverted":     &ast.Text{Span: ast.Span{Start: 2, End: 1, Line: 1, Col: 3}},
		"wrong line":   &ast.Text{Span: ast.Span{Start: 3, End: 4, Line: 1, Col: 4}},
		"wrong column": &ast.Text{Span: ast.Span{Start: 4, End: 5, Line: 2, Col: 1}},
		"out of order": &ast.Block{Span: ast.Span{Start: 0, End: 5, Line: 1, Col: 1}, Nodes: []ast.Node{&ast.Text{Span: ast.Span{Start: 3, End: 4, Line: 2, Col: 1}}, &ast.Text{Span: ast.Span{Start: 0, End: 1, Line: 1, Col: 1}}}},
	} {
		if len(spanViolations(src, n)) == 0 {
			t.Errorf("%s: not flagged", name)
		}
	}
	ok := &ast.Text{Span: ast.Span{Start: 3, End: 5, Line: 2, Col: 1}}
	if v := spanViolations(src, ok); len(v) > 0 {
		t.Errorf("a good span flagged: %v", v)
	}
}
