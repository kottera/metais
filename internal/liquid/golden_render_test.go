// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package liquid

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"os"
	"slices"
	"testing"

	"github.com/kottera/metais/internal/liquid/ast"
	"github.com/kottera/metais/internal/liquid/value"
	"github.com/kottera/metais/internal/testenv"
)

// goldenRenderCase is one case of golden_liquid.json as rendering reads
// it: goldenCase, with its data and its expected result.
type goldenRenderCase struct {
	goldenCase
	Data   json.RawMessage `json:"data"`
	Result *string         `json:"result"`
}

func loadGoldenRender(t *testing.T) []goldenRenderCase {
	t.Helper()
	var doc struct {
		Tests []goldenRenderCase `json:"tests"`
	}
	readJSON(t, goldenDir+"/golden_liquid.json", &doc)
	return doc.Tests
}

// jsonValue reads a JSON document as a value of the data model, its
// numbers kept as written. Empty input, and JSON's null, are null.
func jsonValue(raw []byte) (value.Value, error) {
	if len(bytes.TrimSpace(raw)) == 0 {
		return value.Null{}, nil
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	v, err := decodeValue(dec)
	if err != nil {
		return nil, err
	}
	if _, err := dec.Token(); err != io.EOF {
		return nil, fmt.Errorf("data after the value")
	}
	return v, nil
}

func decodeValue(dec *json.Decoder) (value.Value, error) {
	tok, err := dec.Token()
	if err != nil {
		return nil, err
	}
	switch tok := tok.(type) {
	case nil:
		return value.Null{}, nil
	case bool:
		return value.Bool(tok), nil
	case json.Number:
		return value.ParseNumber(string(tok))
	case string:
		return value.NewString(tok)
	case json.Delim:
		switch tok {
		case '[':
			var elems []value.Value
			for dec.More() {
				e, err := decodeValue(dec)
				if err != nil {
					return nil, err
				}
				elems = append(elems, e)
			}
			if _, err := dec.Token(); err != nil {
				return nil, err
			}
			return value.NewArray(elems...)
		case '{':
			var ms []value.Member
			for dec.More() {
				k, err := dec.Token()
				if err != nil {
					return nil, err
				}
				v, err := decodeValue(dec)
				if err != nil {
					return nil, err
				}
				ms = append(ms, value.Member{Name: k.(string), Value: v})
			}
			if _, err := dec.Token(); err != nil {
				return nil, err
			}
			return value.NewObject(ms...)
		}
	}
	return nil, fmt.Errorf("unexpected token %v", tok)
}

// globalsOf is a case's data as a render's globals: an object, made by
// Data. Null data binds nothing.
func globalsOf(t *testing.T, raw []byte) value.Object {
	t.Helper()
	v, err := jsonValue(raw)
	if err != nil {
		t.Fatalf("data: %v", err)
	}
	switch v := Data(v).(type) {
	case value.Object:
		return v
	case value.Null:
		return value.Object{}
	default:
		t.Fatalf("data is a %s, not an object", kindName(v))
		return value.Object{}
	}
}

// mapPartials is a Partials over parsed templates, by name.
type mapPartials map[string]*Template

func (m mapPartials) Partial(name string) (*Template, *Error) {
	if t, ok := m[name]; ok {
		return t, nil
	}
	return nil, &Error{Code: "partial_not_found", Args: map[string]string{"name": name}}
}

// goldenLimits are ADR-009's defaults, which the corpus never nears.
var goldenLimits = MeterLimits{RenderWork: 1 << 32, OutputSize: 256 << 20, AssignBytes: 512 << 20, LoopIterations: 10_000_000}

// renderGolden parses and renders a case under d. parsed is false when
// the template or a partial does not parse.
func renderGolden(t *testing.T, d *Dialect, c goldenRenderCase) (out string, parsed bool, err *Error) {
	t.Helper()
	tpl, errs := Parse(d, Source{Name: "template", Text: []byte(c.Template)}, ParseLimits{})
	if len(errs) > 0 {
		return "", false, errs[0]
	}
	partials := mapPartials{}
	for _, name := range slices.Sorted(maps.Keys(c.Templates)) {
		p, errs := Parse(d, Source{Name: name, Text: []byte(c.Templates[name])}, ParseLimits{})
		if len(errs) > 0 {
			return "", false, errs[0]
		}
		partials[name] = p
	}
	env := &Env{Dialect: d, Globals: globalsOf(t, c.Data), Partials: partials, Meter: NewMeter(goldenLimits),
		Limits: RenderLimits{ValueSize: 64 << 20, ArraySize: 1_000_000}}
	var b bytes.Buffer
	err = Render(&b, tpl, env)
	return b.String(), true, err
}

// appliesFilter reports whether a case's template, or a partial it
// carries, applies a filter.
func appliesFilter(d *Dialect, c goldenRenderCase) bool {
	texts := []string{c.Template}
	for _, name := range slices.Sorted(maps.Keys(c.Templates)) {
		texts = append(texts, c.Templates[name])
	}
	for _, s := range texts {
		tpl, errs := Parse(d, Source{Text: []byte(s)}, ParseLimits{})
		if len(errs) > 0 {
			continue
		}
		found := false
		ast.Walk(tpl.Root, func(n ast.Node) bool {
			if _, ok := n.(*ast.Filter); ok {
				found = true
			}
			return !found
		})
		if found {
			return true
		}
	}
	return false
}

// TestGoldenLiquidRender is ADR-010 D4 item 7's breadth check, at render,
// over every valid case of Golden Liquid whose template and partials
// apply no filter: each renders its expected result exactly, or is listed
// in exclusions.txt with a dialect line naming the ADR-004 item it
// follows instead, and then must not render that result, so a stale line
// is found. A listed case that the parse refuses is the parse test's;
// every other listed case must be one this run renders, so no line is
// left unchecked. No case may recover an internal error.
func TestGoldenLiquidRender(t *testing.T) {
	testenv.Home(t)
	excluded := readList(t, "exclusions.txt")
	d := goldenDialect(t)
	var run, passed, skipped, filtered, invalid int
	for _, c := range loadGoldenRender(t) {
		if c.Invalid {
			invalid++
			continue
		}
		if appliesFilter(d, c) {
			filtered++
			if x, ok := excluded[c.Name]; ok {
				if _, parsed, _ := renderGolden(t, d, c); parsed {
					t.Errorf("exclusions.txt:%d: %q parses and applies a filter, so no run checks its line", x.line, c.Name)
				}
			}
			continue
		}
		out, parsed, err := renderGolden(t, d, c)
		if err != nil && err.Code == CodeInternal {
			t.Errorf("%q: internal error: %v", c.Name, err)
			continue
		}
		x, isExcluded := excluded[c.Name]
		if !parsed {
			if !isExcluded {
				t.Errorf("%q does not parse and is not excluded: %v", c.Name, err)
			}
			continue
		}
		run++
		ok := err == nil && c.Result != nil && out == *c.Result
		if isExcluded {
			if ok {
				t.Errorf("exclusions.txt:%d: %q renders its result; remove the line", x.line, c.Name)
			}
			skipped++
			continue
		}
		if !ok {
			want := "<none>"
			if c.Result != nil {
				want = fmt.Sprintf("%q", *c.Result)
			}
			if err != nil {
				t.Errorf("%q: %v\ntemplate: %q\nwant: %s", c.Name, err, c.Template, want)
			} else {
				t.Errorf("%q: got %q, want %s\ntemplate: %q", c.Name, out, want, c.Template)
			}
			continue
		}
		passed++
	}
	t.Logf("valid cases without filters: %d rendered, %d pass, %d excluded; %d apply a filter, %d invalid, not run here",
		run, passed, skipped, filtered, invalid)
	if run == 0 {
		t.Fatal("no case rendered")
	}
}

// readJSON reads a JSON file into v.
func readJSON(t *testing.T, file string, v any) {
	t.Helper()
	b, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(b, v); err != nil {
		t.Fatal(err)
	}
}
