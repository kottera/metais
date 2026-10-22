// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package liquid

import (
	"strings"
	"testing"

	"metais.org/metais/internal/liquid/ast"
	"metais.org/metais/internal/testenv"
)

// TestParseValue pins ADR-005 D3 item 2: a value expression is exactly
// what may appear inside "{{ … }}".
func TestParseValue(t *testing.T) {
	testenv.Home(t)
	d := goldenDialect(t)
	e, err := ParseValue(d, Source{Name: "/outputs/0/for_each", Text: []byte(`inputs.articles | where: "draft", false | sort: "id"`)}, ParseLimits{})
	if err != nil {
		t.Fatal(err)
	}
	p, ok := e.Base.(*ast.Path)
	if !ok || p.Root != "inputs" || len(p.Segments) != 1 || p.Segments[0].Name != "articles" {
		t.Errorf("base %#v, want inputs.articles", e.Base)
	}
	if len(e.Filters) != 2 || e.Filters[0].Name != "where" || len(e.Filters[0].Args) != 2 || e.Filters[1].Name != "sort" {
		t.Errorf("filters %+v", e.Filters)
	}
	for _, text := range []string{`1`, `"s"`, `(1..3)`, `a["b"][c]`, ` x `} {
		if _, err := ParseValue(d, Source{Text: []byte(text)}, ParseLimits{}); err != nil {
			t.Errorf("ParseValue(%q): %v", text, err)
		}
	}
}

// TestParseValueRefusals holds ParseValue's refusals to their codes and
// positions, which are within the expression (ADR-005 D3 item 8).
func TestParseValueRefusals(t *testing.T) {
	testenv.Home(t)
	d := goldenDialect(t)
	for _, c := range []struct {
		text string
		code Code
		col  int
	}{
		{``, CodeSyntax, 1},
		{`   `, CodeSyntax, 4},
		{`a b`, CodeSyntax, 3},
		{`a | nope`, CodeUnknownFilter, 5},
		{`a == b`, CodeSyntax, 3},
		{`{{ a }}`, CodeSyntax, 1},
		{`a }}`, CodeSyntax, 3},
		{`a |`, CodeSyntax, 4},
		{`a | join:`, CodeSyntax, 10},
		{"a\x00", CodeSyntax, 2},
		{"a\xff", CodeNotUTF8, 2},
		{`a` + strings.Repeat("[a", 101) + strings.Repeat("]", 101), CodeLimit, 202},
		{`99999999999999999999`, CodeNumberRange, 1},
	} {
		e, err := ParseValue(d, Source{Name: "v", Text: []byte(c.text)}, ParseLimits{})
		if err == nil || e != nil {
			t.Errorf("ParseValue(%q) = %v, %v; want %s", c.text, e, err, c.code)
			continue
		}
		if err.Code != c.code || err.Line != 1 || err.Col != c.col || err.Source != "v" {
			t.Errorf("ParseValue(%q): %v, want %s at v:1:%d", c.text, err, c.code, c.col)
		}
	}
}

// TestParseCondition pins ADR-005 D3 item 1: a condition is exactly what
// may follow if.
func TestParseCondition(t *testing.T) {
	testenv.Home(t)
	d := goldenDialect(t)
	for _, text := range []string{`a`, `inputs.x.note`, `a == "b"`, `a contains "x" and b`, `x != empty`, `(1..2) == r`} {
		if _, err := ParseCondition(d, Source{Text: []byte(text)}, ParseLimits{}); err != nil {
			t.Errorf("ParseCondition(%q): %v", text, err)
		}
	}
	for _, c := range []struct {
		text string
		col  int
	}{
		{``, 1},
		{`a and`, 6},
		{`a ==`, 5},
		{`a | upcase`, 3},
		{`a b`, 3},
		{`not a`, 5},
		{`a = b`, 3},
		{`(a and b)`, 4},
	} {
		cond, err := ParseCondition(d, Source{Name: "w", Text: []byte(c.text)}, ParseLimits{})
		if err == nil || cond != nil {
			t.Errorf("ParseCondition(%q) = %v, %v; want a syntax error", c.text, cond, err)
			continue
		}
		if err.Code != CodeSyntax || err.Col != c.col {
			t.Errorf("ParseCondition(%q): %v, want syntax at column %d", c.text, err, c.col)
		}
	}
}
