// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package inputs

import (
	"slices"
	"testing"

	"metais.org/metais/internal/limits"
	"metais.org/metais/internal/testenv"
)

// TestDuplicatesBySchemaLocation: "Of two findings equal in code, subject,
// origin, file, argument, pointer, schema entry, schema pointer and
// parameters, it MUST report one" (MTS-FND-003). Two keywords at two
// locations that fail alike are two findings: a table spelled with one
// minContains branch per criterion reports every criterion it misses, not
// one, and a member two required keywords both require is reported once
// for each. One keyword reached twice through a shared $ref ends at one
// authored location (MTS-FND-005), so it is one finding. Two of each are
// the smallest size that can fail the key.
func TestDuplicatesBySchemaLocation(t *testing.T) {
	testenv.Home(t)
	l := layer(t, pkg{
		inputs: []map[string]any{slot("table", "inputs/table.json"), slot("form", "inputs/form.json")},
		schemas: map[string]string{
			"inputs/table.json": `{"type": "array", "allOf": [
				{"contains": {"const": "1.1.1"}, "minContains": 1},
				{"contains": {"const": "1.2.1"}, "minContains": 1}]}`,
			"inputs/form.json": `{"type": "object", "$defs": {"name": {"required": ["name"]}},
				"allOf": [{"required": ["name"]}, {"required": ["name"]}],
				"properties": {"shared": {"allOf": [{"$ref": "#/$defs/name"}, {"$ref": "#/$defs/name"}]}}}`,
		},
	})
	res := run(t, l, limits.Defaults(), args{flags: []string{`--set-json=/table=[]`, `--set-json=/form={"shared": {}}`}})
	wantFindings(t, res.Findings,
		`input.min_contains table set:1 # {"limit":1,"count":0}`,
		`input.min_contains table set:1 # {"limit":1,"count":0}`,
		`input.required form set:2 #/name {"property":"name"}`,
		`input.required form set:2 #/name {"property":"name"}`,
		`input.required form set:2 #/shared/name {"property":"name"}`,
	)
	var ptrs []string
	for _, f := range res.Findings {
		s, _ := f.Schema()
		ptrs = append(ptrs, s.Entry+"#"+s.Pointer)
	}
	want := []string{
		"inputs/form.json#/$defs/name/required", "inputs/form.json#/allOf/0/required", "inputs/form.json#/allOf/1/required",
		"inputs/table.json#/allOf/0/minContains", "inputs/table.json#/allOf/1/minContains",
	}
	slices.Sort(ptrs)
	if !slices.Equal(ptrs, want) {
		t.Errorf("schema locations %v, want %v", ptrs, want)
	}
}
