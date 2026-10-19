// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package liquid

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/kottera/metais/internal/liquid/ast"
	"github.com/kottera/metais/internal/liquid/value"
	"github.com/kottera/metais/internal/testenv"
)

// goldenRenderCase is one case of golden_liquid.json as rendering reads
// it: goldenCase with its data and expected result.
type goldenRenderCase struct {
	goldenCase
	Data   json.RawMessage `json:"data"`
	Result *string         `json:"result"`
}

func loadGoldenRender(t *testing.T) []goldenRenderCase {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(goldenDir, "golden_liquid.json"))
	if err != nil {
		t.Fatal(err)
	}
	var doc struct {
		Tests []goldenRenderCase `json:"tests"`
	}
	if err := json.Unmarshal(b, &doc); err != nil {
		t.Fatal(err)
	}
	return doc.Tests
}

// dataValue reads JSON as the data model holds it, each number presented
// to a template as ADR-004 D4 item 2 says: an integral number of magnitude
// at most 2^53−1 is an integer, written as one, and any other a float,
// written with a point or an exponent.
func dataValue(raw []byte) (value.Value, error) {
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	var x any
	if err := dec.Decode(&x); err != nil {
		return nil, err
	}
	return fromJSON(x)
}

func fromJSON(x any) (value.Value, error) {
	switch x := x.(type) {
	case nil:
		return value.Null{}, nil
	case bool:
		return value.Bool(x), nil
	case string:
		return value.NewString(x)
	case json.Number:
		f, err := strconv.ParseFloat(string(x), 64)
		if err != nil {
			return nil, err
		}
		if f == math.Trunc(f) && math.Abs(f) <= value.MaxSafeInteger {
			return value.ParseNumber(strconv.FormatInt(int64(f), 10))
		}
		return value.ParseNumber(floatText(f))
	case []any:
		var elems []value.Value
		for _, e := range x {
			v, err := fromJSON(e)
			if err != nil {
				return nil, err
			}
			elems = append(elems, v)
		}
		return value.NewArray(elems...)
	case map[string]any:
		var ms []value.Member
		for k, e := range x {
			v, err := fromJSON(e)
			if err != nil {
				return nil, err
			}
			ms = append(ms, value.Member{Name: k, Value: v})
		}
		return value.NewObject(ms...)
	}
	return nil, fmt.Errorf("unexpected %T", x)
}

// The codes outputEval reports for what the renderer refuses: strictness
// (ADR-004 D6) and a range's bounds (D2). The renderer owns their names.
const (
	evalUndefined Code = "undefined"
	evalRange     Code = "range"
)

// outputEval evaluates a template of text and output expressions only,
// the shape of most of Golden Liquid's filter cases, with ADR-004 D6's
// strictness: just enough evaluation to run the catalog through its
// FilterDefs against the corpus. It is not the renderer, which is W3-3's.
type outputEval struct {
	data value.Object
	ctx  testContext
}

// supported reports whether the template is text and output expressions
// only, and applies a filter.
func supported(root *ast.Block) bool {
	ok, filtered := true, false
	ast.Walk(root, func(n ast.Node) bool {
		switch n.(type) {
		case *ast.Block, *ast.Text, *ast.Output, *ast.Expr, *ast.Arg, *ast.Literal, *ast.Path, *ast.Segment, *ast.Range:
		case *ast.Filter:
			filtered = true
		default:
			ok = false
		}
		return ok
	})
	return ok && filtered
}

func (e *outputEval) render(root *ast.Block) (string, *Error) {
	var b strings.Builder
	for _, n := range root.Nodes {
		switch n := n.(type) {
		case *ast.Text:
			b.WriteString(n.Trimmed())
		case *ast.Output:
			if n.Value == nil {
				continue
			}
			v, err := e.expr(n.Value)
			if err != nil {
				return "", err
			}
			s, err := text("", v)
			if err != nil {
				return "", err
			}
			b.WriteString(s)
		}
	}
	return b.String(), nil
}

func (e *outputEval) expr(x *ast.Expr) (value.Value, *Error) {
	v, err := e.primary(x.Base)
	if err != nil {
		return nil, err
	}
	for _, f := range x.Filters {
		def, _ := Core().Filter(f.Name)
		var args []Arg
		for _, a := range f.Args {
			av, err := e.primary(a.Value)
			if err != nil {
				return nil, err
			}
			args = append(args, Arg{Name: a.Name, Value: av})
		}
		if v, err = def.Apply(e.ctx, v, args); err != nil {
			return nil, err
		}
	}
	return v, nil
}

func (e *outputEval) primary(p ast.Primary) (value.Value, *Error) {
	switch p := p.(type) {
	case *ast.Literal:
		if p.Value == nil {
			return value.String{}, nil
		}
		return p.Value, nil
	case *ast.Range:
		from, err := e.bound(p.From)
		if err != nil {
			return nil, err
		}
		to, err := e.bound(p.To)
		if err != nil {
			return nil, err
		}
		var elems []value.Value
		for i := from; i <= to; i++ {
			n, _ := value.ParseNumber(strconv.FormatInt(i, 10))
			elems = append(elems, n)
		}
		return newArray(elems), nil
	case *ast.Path:
		var cur value.Value = e.data
		if p.Root != "" {
			v, ok := e.data.Get(p.Root)
			if !ok {
				return nil, &Error{Code: evalUndefined, Detail: "undefined variable " + p.Root}
			}
			cur = v
		}
		for _, s := range p.Segments {
			var key value.Value = newString(s.Name)
			if s.Index != nil {
				k, err := e.primary(s.Index)
				if err != nil {
					return nil, err
				}
				key = k
			}
			next, err := segment(cur, key, s.Index == nil)
			if err != nil {
				return nil, err
			}
			cur = next
		}
		return cur, nil
	}
	return nil, &Error{Code: CodeInternal, Detail: "unexpected primary"}
}

func (e *outputEval) bound(p ast.Primary) (int64, *Error) {
	v, err := e.primary(p)
	if err != nil {
		return 0, err
	}
	if n, ok := v.(value.Number); ok && numberKind(n).isInt {
		return numberKind(n).i, nil
	}
	return 0, &Error{Code: evalRange, Detail: "a range bound is not an integer"}
}

// segment is one step of a path, with ADR-004 D6's strictness and D4
// item 7's shorthands.
func segment(cur, key value.Value, dotted bool) (value.Value, *Error) {
	k, isName := key.(value.String)
	switch c := cur.(type) {
	case value.Object:
		if isName {
			if v, ok := c.Get(k.Text()); ok {
				return v, nil
			}
		}
	case value.Array:
		if n, ok := key.(value.Number); ok && numberKind(n).isInt {
			i := numberKind(n).i
			if i < 0 {
				i += int64(c.Len())
			}
			if i < 0 || i >= int64(c.Len()) {
				return value.Null{}, nil
			}
			return c.At(int(i)), nil
		}
	case value.Null:
		return nil, &Error{Code: evalUndefined, Detail: "a property of nil"}
	}
	if isName && dotted {
		switch k.Text() {
		case "size":
			return size(&call{name: "size", m: newTestMeter(nil)}, cur)
		case "first", "last":
			if _, ok := cur.(value.String); !ok {
				return end(&call{m: newTestMeter(nil)}, cur, k.Text() == "first")
			}
		}
	}
	return nil, &Error{Code: evalUndefined, Detail: "undefined property"}
}

// TestGoldenLiquidFilters runs the catalog through its FilterDefs against
// every case of Golden Liquid's filters group that outputEval can
// evaluate: a template of text and output expressions only that applies a
// filter, and does not apply date (goldenDateSelector). Each case is parsed under Core() and rendered
// with its data:
//
//   - a valid case renders its result;
//   - an invalid case is refused, at parse or at render;
//   - a case that cannot do either is listed in filter-exclusions.txt,
//     with a dialect line naming the ADR-004 item that pins the behaviour
//     Shopify's differs from, and must still fail, so a stale line is
//     found.
//
// The run reports what it ran, passed, excluded and selected out. The
// render run over the whole corpus is the renderer's.
func TestGoldenLiquidFilters(t *testing.T) {
	testenv.Home(t)
	excluded := readList(t, "filter-exclusions.txt")
	appliesDate, err := goldenDateSelector(Core())
	if err != nil {
		t.Fatal(err)
	}
	cases := loadGoldenRender(t)
	names := map[string]bool{}
	for _, c := range cases {
		names[c.Name] = true
	}
	for _, l := range excluded {
		if l.reason != "dialect" || !strings.HasPrefix(l.note, "ADR-004 ") {
			t.Errorf("filter-exclusions.txt:%d: a line is a dialect line naming its ADR-004 item", l.line)
		}
		if !names[l.name] {
			t.Errorf("filter-exclusions.txt:%d: no case %q", l.line, l.name)
		}
	}
	ran := map[string]bool{}
	var passed, skipped, selectedOut int
	for _, c := range cases {
		if !strings.HasPrefix(c.Name, "filters, ") || len(c.Templates) > 0 {
			continue
		}
		tpl, _ := Parse(Core(), Source{Name: "template", Text: []byte(c.Template)}, ParseLimits{})
		if tpl != nil && !supported(tpl.Root) {
			continue
		}
		if appliesDate(c.goldenCase) {
			selectedOut++
			continue
		}
		if tpl == nil {
			// Refused at parse: TestGoldenLiquidParse holds these.
			continue
		}
		ran[c.Name] = true
		data := value.Object{}
		if len(c.Data) > 0 {
			v, err := dataValue(c.Data)
			if err != nil {
				t.Fatalf("%q: data: %v", c.Name, err)
			}
			data = v.(value.Object)
		}
		ev := &outputEval{data: data, ctx: testContext{newTestMeter(nil)}}
		got, rerr := ev.render(tpl.Root)
		if rerr != nil && rerr.Code == CodeInternal {
			t.Errorf("%q: internal error: %v", c.Name, rerr)
		}
		var ok bool
		var why string
		switch {
		case c.Invalid:
			ok = rerr != nil
			why = "is invalid and renders " + strconv.Quote(got)
		case rerr != nil:
			why = "is refused: " + rerr.Error()
		case c.Result == nil:
			why = "has no result"
		default:
			ok = got == *c.Result
			why = fmt.Sprintf("renders %q, want %q", got, *c.Result)
		}
		if x, isExcluded := excluded[c.Name]; isExcluded {
			if ok {
				t.Errorf("filter-exclusions.txt:%d: %q passes; remove the line", x.line, c.Name)
			}
			skipped++
			continue
		}
		if !ok {
			t.Errorf("%q %s\n%s", c.Name, why, c.Template)
			continue
		}
		passed++
	}
	for _, l := range excluded {
		if names[l.name] && !ran[l.name] {
			t.Errorf("filter-exclusions.txt:%d: %q is not a case this run evaluates", l.line, l.name)
		}
	}
	t.Logf("%d filter cases ran: %d pass, %d excluded; %d selected out for applying date", len(ran), passed, skipped, selectedOut)
	if len(ran) < 300 {
		t.Errorf("only %d cases ran; the selection is broken", len(ran))
	}
}

// TestGoldenDateSelector pins the selector's two halves: it takes out
// exactly the cases that apply date, among them the two render-time.txt
// keeps for Wave 5; and it fails the moment date is registered in the
// dialect a run uses, so the cases come back in scope.
func TestGoldenDateSelector(t *testing.T) {
	testenv.Home(t)
	appliesDate, err := goldenDateSelector(Core())
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, c := range loadGolden(t) {
		if appliesDate(c) {
			out = append(out, c.Name)
			if !strings.Contains(c.Template, "date") {
				t.Errorf("%q is selected out and its template names no date", c.Name)
			}
		}
	}
	for _, n := range []string{"filters, date, missing argument", "filters, date, too many arguments"} {
		if !slices.Contains(out, n) {
			t.Errorf("%q is not selected out", n)
		}
	}
	t.Logf("%d cases apply date: %q", len(out), out)
	if appliesDate(goldenCase{Template: `{{ "date" }}{% assign date = 1 %}{{ date | plus: 1 }}`}) {
		t.Errorf("a template that names date without applying it is selected out")
	}
	if !appliesDate(goldenCase{Template: `x`, Templates: map[string]string{"p": `{{ x | date: "%Y" }}`}}) {
		t.Errorf("a partial that applies date is not selected out")
	}

	withDate, err := Core().Extend(Spec{Filters: []FilterDef{stubFilter("date")}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := goldenDateSelector(withDate); err == nil {
		t.Errorf("the selector accepts a dialect that registers date")
	}
}
