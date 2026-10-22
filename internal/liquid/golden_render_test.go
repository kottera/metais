// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package liquid

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"slices"
	"strings"
	"testing"

	"metais.org/metais/internal/liquid/value"
	"metais.org/metais/internal/testenv"
)

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

// TestGoldenLiquidRender is ADR-010 D4 item 7's breadth check, at render,
// and the meeting of the renderer with the standard catalog: every case
// of Golden Liquid runs under goldenCore(), Core() with date, filters
// included.
//
//   - a valid case renders its expected result exactly;
//   - an invalid case that render-time.txt lists parses, and is refused
//     at render with the code its line names;
//   - every other invalid case is refused at parse, among them the arity
//     cases that left render-time.txt because the catalog's Check refuses
//     them there;
//   - a case that does none of these is listed in exclusions.txt, or in
//     the catalog's filter-exclusions.txt, with a dialect line naming the
//     ADR-004 item it follows instead, and must still fail, so a stale
//     line is found.
//
// No case may recover an internal error. The run reports what it ran,
// passed and excluded, each line.
func TestGoldenLiquidRender(t *testing.T) {
	testenv.Home(t)
	excluded := readList(t, "exclusions.txt")
	filterExcluded := readList(t, "filter-exclusions.txt")
	renderTime := readList(t, "render-time.txt")
	d := goldenCore(t)
	var run, passed, rendered, refusedAtRender, refusedAtParse, byCheck int
	var lines []string
	for _, c := range loadGoldenRender(t) {
		run++
		out, parsed, err := renderGolden(t, d, c)
		if err != nil && err.Code == CodeInternal {
			t.Errorf("%q: internal error: %v", c.Name, err)
			continue
		}
		_, deferred := renderTime[c.Name]
		var ok bool
		var why string
		switch {
		case !c.Invalid && !parsed:
			why = fmt.Sprintf("is valid and refused at parse: %v", err)
		case !c.Invalid && err != nil:
			why = fmt.Sprintf("is valid and refused: %v", err)
		case !c.Invalid:
			ok = c.Result != nil && out == *c.Result
			why = fmt.Sprintf("renders %q, want %s", out, quoted(c.Result))
		case deferred && !parsed:
			why = fmt.Sprintf("is in render-time.txt and refused at parse: %v", err)
		case deferred && err == nil:
			why = fmt.Sprintf("is in render-time.txt and renders %q", out)
		case deferred:
			want := renderTimeCode(renderTime[c.Name].reason)
			ok = err.Code == want
			why = fmt.Sprintf("is refused with %v; its render-time.txt line names %s", err, want)
		default:
			ok = !parsed
			why = fmt.Sprintf("is invalid and parses, rendering %q, %v", out, err)
		}
		x, isExcluded := excluded[c.Name]
		file := "exclusions.txt"
		if !isExcluded {
			x, isExcluded = filterExcluded[c.Name]
			file = "filter-exclusions.txt"
		}
		if isExcluded {
			if ok {
				t.Errorf("%s:%d: %q passes; remove the line", file, x.line, c.Name)
			}
			lines = append(lines, fmt.Sprintf("%s:%d\t%s\t%s\t%s", file, x.line, c.Name, x.reason, x.note))
			continue
		}
		if !ok {
			t.Errorf("%q %s\ntemplate: %q", c.Name, why, c.Template)
			continue
		}
		passed++
		switch {
		case !c.Invalid:
			rendered++
		case deferred:
			refusedAtRender++
		default:
			refusedAtParse++
			if err.Code == CodeFilterArgument || err.Code == CodeDivisionByZero {
				byCheck++
			}
		}
	}
	t.Logf("%d cases run: %d pass (%d valid render their result, %d of render-time.txt refused at render, "+
		"%d invalid refused at parse, %d of them by the catalog's Check), %d excluded",
		run, passed, rendered, refusedAtRender, refusedAtParse, byCheck, len(lines))
	for _, l := range lines {
		t.Logf("excluded: %s", l)
	}
	if run != len(loadGoldenRender(t)) || run < 1000 {
		t.Errorf("%d cases run of %d", run, len(loadGoldenRender(t)))
	}
}

// renderTimeCode is the code a render-time.txt line's reason names.
func renderTimeCode(reason string) Code {
	for _, c := range []struct {
		says string
		code Code
	}{
		{"filter_input", CodeFilterInput},
		{"filter_argument", CodeFilterArgument},
		{"an undefined variable", CodeUndefinedVariable},
		{"a comparison", CodeComparison},
		{"a loop modifier", CodeArgument},
		{"a range's bounds", CodeArgument},
		{"division by zero", CodeDivisionByZero},
	} {
		if strings.Contains(reason, c.says) {
			return c.code
		}
	}
	return "unnamed"
}

// quoted writes an expected result, or that there is none.
func quoted(s *string) string {
	if s == nil {
		return "<none>"
	}
	return fmt.Sprintf("%q", *s)
}
