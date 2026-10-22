// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package xref

import (
	"bytes"
	"slices"
	"testing"

	"metais.org/metais/internal/finding"
	"metais.org/metais/internal/liquid"
	"metais.org/metais/internal/liquid/value"
)

// shorthandSchema declares an array, a string, an object with no member
// named size, first or last, and an object with all three.
const shorthandSchema = `{
	"type": "object",
	"properties": {
		"xs": {"type": "array", "items": {"type": "string"}},
		"s": {"type": "string"},
		"o": {"type": "object", "properties": {"a": {"type": "integer"}}, "additionalProperties": false},
		"m": {"type": "object", "properties": {"size": {"type": "integer"}, "first": {"type": "integer"}, "last": {"type": "integer"}}}
	}
}`

// shorthandData is a value of shorthandSchema, bound as render binds it.
const shorthandData = `{
	"inputs": {"metrics": {"xs": ["a", "b"], "s": "abc", "o": {"a": 1}, "m": {"size": 7, "first": 8, "last": 9}}},
	"package": {"name": "fixture", "version": "1.0.0", "title": "Fixture"}
}`

// TestShorthandAgreement: for size, first and last written as a
// bracketed string on an array, a string and an object — from a schema,
// a local or a record — the cross-reference reports an error exactly
// where render refuses, and names the finding ADR-007 D3 item 4.6 gives:
// Liquid's command names are read only after a "." (ADR-004 D4 item 7),
// so a bracketed one is a name like any other. After a ".", they read as
// D3 item 4.5 reads them, which leaves an object's first and last
// undeclared where render gives its first and last pairs.
func TestShorthandAgreement(t *testing.T) {
	globals := liquid.Data(jsonValue(t, "data.json", shorthandData)).(value.Object)
	for _, base := range []struct {
		name, prelude, path string
		// bracketed is the finding a bracketed shorthand gives, "" for none.
		bracketed string
	}{
		{"a schema's array", "", "inputs.metrics.xs", notAnObject},
		{"a schema's string", "", "inputs.metrics.s", notAnObject},
		{"a schema's object", "", "inputs.metrics.o", unknownField},
		{"a schema's object with the members", "", "inputs.metrics.m", ""},
		{"a local array", `{% assign v = (1..3) %}`, "v", notAnObject},
		{"a local string", `{% assign v = "abc" %}`, "v", notAnObject},
		{"a record", "", "package", unknownField},
	} {
		for _, name := range []string{"size", "first", "last"} {
			for _, seg := range []string{"." + name, `["` + name + `"]`} {
				text := base.prelude + "{{ " + base.path + seg + " }}"
				bracketed := seg[0] == '['
				var want []string
				switch {
				case bracketed && base.bracketed != "":
					want = []string{base.bracketed}
				case !bracketed && base.bracketed == unknownField && name != "size":
					want = []string{unknownField}
				}
				d := dialect(t)
				tpl, errs := liquid.Parse(d, liquid.Source{Name: "t.liquid", Text: []byte(text)}, liquid.ParseLimits{})
				if len(errs) > 0 {
					t.Fatalf("parse %s: %v", text, errs)
				}
				var b bytes.Buffer
				rerr := liquid.Render(&b, tpl, &liquid.Env{Dialect: d, Globals: globals, Meter: liquid.NewMeter(liquid.MeterLimits{
					RenderWork: 1 << 20, OutputSize: 1 << 20, AssignBytes: 1 << 20, LoopIterations: 1 << 10,
				})})
				var errCodes []string
				for _, f := range one(shorthandSchema, text).run(t).Findings {
					if f.Severity() == finding.SeverityError {
						errCodes = append(errCodes, string(f.Code()))
					}
				}
				if refused := rerr != nil; (bracketed || want == nil) && refused != (len(errCodes) > 0) {
					t.Errorf("%s: %s: render refused %v (%v), check errors %v", base.name, text, refused, rerr, errCodes)
				}
				if !slices.Equal(errCodes, want) {
					t.Errorf("%s: %s: check errors %v, want %v", base.name, text, errCodes, want)
				}
			}
		}
	}
}
