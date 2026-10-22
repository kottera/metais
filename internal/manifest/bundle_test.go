// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package manifest

import (
	"bytes"
	"testing"

	"metais.org/metais/internal/jsonx"
	"metais.org/metais/internal/schema"
)

// declarations is the loader's step from a model to bundling: the inputs
// as internal/schema reads them, with ADR-003 D1's defaults the model has
// already applied.
func declarations(m *Manifest) schema.Declarations {
	d := schema.Declarations{Constraints: m.InputConstraints}
	for _, in := range m.Inputs {
		si := schema.Input{Name: in.Name, Title: in.Title, Schema: in.Schema, Required: in.Required}
		if in.Description != "" {
			desc := in.Description
			si.Description = &desc
		}
		d.Inputs = append(d.Inputs, si)
	}
	return d
}

// TestBundleMeeting is W2-16's meeting point with W2-12: the inputs of a
// manifest the model builds feed W2-12's bundling, and give the combined
// schema of ADR-003 D7: a property per input in manifest order with the
// slot's title and description beside a $ref to its schema's canonical
// URI, required listing the required inputs (an omitted required is true),
// allOf naming input_constraints, and every schema file in $defs.
func TestBundleMeeting(t *testing.T) {
	isolate(t)
	m := base()
	set(t, m, "", "inputs", []any{
		tree{"name": "vendor", "title": "Vendor", "description": "The organisation that makes the product.",
			"schema": "inputs/vendor.schema.json"},
		tree{"name": "notes", "title": "Notes", "schema": "inputs/notes.schema.yaml", "required": false},
		tree{"name": "criteria", "title": "Criteria", "schema": "inputs/criteria.schema.json", "required": true},
	})
	root, _ := parse(t, encode(t, m))
	model, err := Build(root)
	if err != nil {
		t.Fatal(err)
	}
	file := func(path, text string) schema.File {
		v, _ := parse(t, []byte(text))
		return schema.File{Path: path, Root: v}
	}
	files := []schema.File{
		file("inputs/vendor.schema.json", `{"type": "object"}`),
		file("inputs/notes.schema.yaml", `{"type": "string"}`),
		file("inputs/criteria.schema.json", `{"$id": "https://example.org/criteria.json", "type": "array"}`),
		file("inputs/constraints.schema.yaml", `{"required": ["vendor"]}`),
	}
	got, err := schema.Bundle(declarations(model), files)
	if err != nil {
		t.Fatal(err)
	}
	want, _ := parse(t, []byte(`{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"$id": "metais:///.compiled/schema.json",
		"type": "object",
		"properties": {
			"vendor": {"title": "Vendor", "description": "The organisation that makes the product.",
				"$ref": "metais:///inputs/vendor.schema.json"},
			"notes": {"title": "Notes", "$ref": "metais:///inputs/notes.schema.yaml"},
			"criteria": {"title": "Criteria", "$ref": "https://example.org/criteria.json"}
		},
		"required": ["vendor", "criteria"],
		"additionalProperties": false,
		"allOf": [{"$ref": "metais:///inputs/constraints.schema.yaml"}],
		"$defs": {
			"inputs/vendor.schema.json": {"$schema": "https://json-schema.org/draft/2020-12/schema",
				"$id": "metais:///inputs/vendor.schema.json", "type": "object"},
			"inputs/notes.schema.yaml": {"$schema": "https://json-schema.org/draft/2020-12/schema",
				"$id": "metais:///inputs/notes.schema.yaml", "type": "string"},
			"inputs/criteria.schema.json": {"$schema": "https://json-schema.org/draft/2020-12/schema",
				"$id": "https://example.org/criteria.json", "type": "array"},
			"inputs/constraints.schema.yaml": {"$schema": "https://json-schema.org/draft/2020-12/schema",
				"$id": "metais:///inputs/constraints.schema.yaml", "required": ["vendor"]}
		}
	}`))
	if g, w := jsonx.Canonical(got), jsonx.Canonical(want); !bytes.Equal(g, w) {
		t.Fatalf("the combined schema is\n%s\nwant\n%s", g, w)
	}
}
