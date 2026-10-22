// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package liquid

import (
	"errors"
	"strings"
	"testing"

	"metais.org/metais/internal/liquid/ast"
	"metais.org/metais/internal/liquid/value"
	"metais.org/metais/internal/testenv"
)

// recordTag is a TagDef that records each tag it is asked to check, and
// refuses one whose first argument is the literal "no".
type recordTag struct {
	name    string
	checked []*ast.Tag
}

func (r *recordTag) Name() string { return r.name }

func (r *recordTag) Check(t *ast.Tag) *Error {
	r.checked = append(r.checked, t)
	if len(t.Args) > 0 {
		if l, ok := t.Args[0].Value.(*ast.Literal); ok && l.Text == "no" {
			return &Error{Code: "tag_refused", Args: map[string]string{"tag": r.name}}
		}
	}
	return nil
}

func (r *recordTag) Render(TagContext, *ast.Tag) *Error { return nil }

// checkFilter is a FilterDef whose Check refuses any argument.
type checkFilter string

func (f checkFilter) Name() string { return string(f) }
func (f checkFilter) Check(a *ast.Filter) *Error {
	if len(a.Args) > 0 {
		return &Error{Code: "filter_argument", Args: map[string]string{"filter": string(f)}}
	}
	return nil
}
func (checkFilter) Apply(FilterContext, value.Value, []Arg) (value.Value, *Error) { return nil, nil }

// panicFilter panics when checked: a defect the parser must recover.
type panicFilter struct{}

func (panicFilter) Name() string             { return "boom" }
func (panicFilter) Check(*ast.Filter) *Error { panic("a defect") }
func (panicFilter) Apply(FilterContext, value.Value, []Arg) (value.Value, *Error) {
	return nil, nil
}

// TestDialectRefusals pins what a Spec may not add.
func TestDialectRefusals(t *testing.T) {
	testenv.Home(t)
	for name, s := range map[string]Spec{
		"a core tag":               {Tags: []TagDef{&recordTag{name: "if"}}},
		"include":                  {Tags: []TagDef{&recordTag{name: "include"}}},
		"an end tag":               {Tags: []TagDef{&recordTag{name: "endembed"}}},
		"a tag twice":              {Tags: []TagDef{&recordTag{name: "embed"}, &recordTag{name: "embed"}}},
		"a tag that is no name":    {Tags: []TagDef{&recordTag{name: "1x"}}},
		"a filter twice":           {Filters: []FilterDef{stubFilter("f"), stubFilter("f")}},
		"a filter that is no name": {Filters: []FilterDef{stubFilter("a b")}},
		"a name reserved twice":    {Reserved: []string{"x", "x"}},
		"a reserved non-name":      {Reserved: []string{"x?"}},
	} {
		if _, err := NewDialect(s); !errors.Is(err, ErrDialect) {
			t.Errorf("%s: NewDialect = %v, want ErrDialect", name, err)
		}
	}
	// Extend refuses what the dialect already has, and leaves it as it was.
	d, err := NewDialect(Spec{Filters: []FilterDef{stubFilter("f")}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := d.Extend(Spec{Filters: []FilterDef{stubFilter("f")}}); !errors.Is(err, ErrDialect) {
		t.Errorf("Extend with a filter the dialect has = %v, want ErrDialect", err)
	}
	e, err := d.Extend(Spec{Filters: []FilterDef{stubFilter("g")}})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := d.Filter("g"); ok {
		t.Error("Extend changed the dialect it extended")
	}
	if _, ok := e.Filter("f"); !ok {
		t.Error("Extend lost the dialect's own filter")
	}
}

// TestBareDialect pins that the bare grammar has no filter: every filter
// is a dialect's, so an unknown one is refused (ADR-004 D1 item 6).
func TestBareDialect(t *testing.T) {
	testenv.Home(t)
	d, err := NewDialect(Spec{})
	if err != nil {
		t.Fatal(err)
	}
	refused(t, d, `{{ x | upcase }}`, CodeUnknownFilter)
	refused(t, d, `{% embed "x" %}`, CodeSyntax)
	if _, errs := parse(t, d, `{% if x %}{{ x }}{% endif %}`); len(errs) > 0 {
		t.Errorf("the bare grammar refuses a core tag: %v", errs)
	}
	if _, errs := Parse(nil, Source{Text: []byte("{{ x }}")}, ParseLimits{}); len(errs) > 0 {
		t.Errorf("a nil dialect is not the bare grammar: %v", errs)
	}
}

// TestAddedTag pins the TagDef contract at parse: the markup parses as an
// argument list, Check sees each occurrence, and its refusal is reported
// at the tag, the parse going on.
func TestAddedTag(t *testing.T) {
	testenv.Home(t)
	tag := &recordTag{name: "embed"}
	d, err := NewDialect(Spec{Tags: []TagDef{tag}})
	if err != nil {
		t.Fatal(err)
	}
	tpl, errs := parse(t, d, `{% embed "a", index: 2 %}{% embed "b" index: n %}`)
	if len(errs) > 0 {
		t.Fatal(errs)
	}
	if len(tag.checked) != 2 {
		t.Fatalf("Check saw %d tags, want 2", len(tag.checked))
	}
	n := tpl.Root.Nodes[0].(*ast.Tag)
	if n.Name != "embed" || len(n.Args) != 2 || n.Args[0].Name != "" || n.Args[1].Name != "index" {
		t.Errorf("tag %+v, want embed with one positional and one keyword argument", n)
	}
	errs = refused(t, d, `x{% embed "no" %}{% embed "yes" %}{% bad %}`, "tag_refused", CodeSyntax)
	if len(errs) > 0 && (errs[0].Col != 2 || errs[0].Source != "t.liquid" || errs[0].Args["tag"] != "embed") {
		t.Errorf("refusal %v, want it at t.liquid:1:2 with its args", errs[0])
	}
	for _, text := range []string{`{% embed k: 1, "late" %}`, `{% embed k: 1, k: 2 %}`, `{% embed "a", %}`, `{% embed "a" | f %}`} {
		refused(t, d, text, CodeSyntax)
	}
}

// TestFilterCheck pins the FilterDef contract at parse: Check sees each
// application, and its refusal is reported at the filter, the parse
// going on.
func TestFilterCheck(t *testing.T) {
	testenv.Home(t)
	d, err := NewDialect(Spec{Filters: []FilterDef{checkFilter("upcase")}})
	if err != nil {
		t.Fatal(err)
	}
	if _, errs := parse(t, d, `{{ x | upcase }}`); len(errs) > 0 {
		t.Errorf("an accepted application is refused: %v", errs)
	}
	errs := refused(t, d, `{{ x | upcase: 1 }}{{ y | upcase: 2 }}`, "filter_argument", "filter_argument")
	if len(errs) == 2 && (errs[0].Col != 1 || errs[1].Col != 20) {
		t.Errorf("refusals at columns %d and %d, want 1 and 20", errs[0].Col, errs[1].Col)
	}
}

// TestPanicRecovered pins ADR-010 D4 item 6: a panic inside the parse is
// one CodeInternal error, never a panic out of Parse, ParseValue or
// ParseCondition.
func TestPanicRecovered(t *testing.T) {
	testenv.Home(t)
	d, err := NewDialect(Spec{Filters: []FilterDef{panicFilter{}}})
	if err != nil {
		t.Fatal(err)
	}
	tpl, errs := parse(t, d, `{{ x | boom }}`)
	if tpl != nil || len(errs) != 1 || errs[0].Code != CodeInternal || !strings.Contains(errs[0].Detail, "a defect") {
		t.Errorf("Parse = %v, %v; want one internal error", tpl, errs)
	}
	if e, err := ParseValue(d, Source{Text: []byte("x | boom")}, ParseLimits{}); e != nil || err == nil || err.Code != CodeInternal {
		t.Errorf("ParseValue = %v, %v; want an internal error", e, err)
	}
	if c, err := ParseCondition(d, Source{Text: []byte("x")}, ParseLimits{}); c == nil || err != nil {
		t.Errorf("ParseCondition = %v, %v; want the condition", c, err)
	}
}
