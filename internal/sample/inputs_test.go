// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package sample

import (
	"testing"

	"metais.org/metais/internal/liquid/value"
)

// constraintsEntry is the input constraints file of an Inputs test.
const constraintsEntry = "inputs/constraints.schema.json"

// editionFiles is a report whose optional edition decides whether the
// optional table is supplied, as the accessibility conformance report's
// does: required by the b edition, and refused by every other.
var editionFiles = map[string]string{
	"inputs/report.schema.json": `{"type":"object","properties":{"edition":{"enum":["a","b"]},"name":{"type":"string"}}}`,
	"inputs/table.schema.json":  `{"type":"string"}`,
	constraintsEntry: `{"allOf":[{
		"if":{"required":["report"],"properties":{"report":{"required":["edition"],"properties":{"edition":{"const":"b"}}}}},
		"then":{"required":["table"]},
		"else":{"properties":{"table":false}}}]}`,
}

var editionInputs = []Input{
	{Name: "report", Schema: "inputs/report.schema.json", Forms: []Form{FormYAML}, Required: true},
	{Name: "table", Schema: "inputs/table.schema.json", Forms: []Form{FormText}},
}

// inputsObject is the inputs object gs makes of ins.
func inputsObject(t *testing.T, ins []Input, gs []Generated) value.Object {
	t.Helper()
	m := map[string]value.Value{}
	for i, g := range gs {
		if g.Err != nil || len(g.Findings) > 0 {
			t.Fatalf("input %s: %v %s", ins[i].Name, g.Err, describe(g.Findings))
		}
		if g.Present {
			m[ins[i].Name] = g.Value
		}
	}
	return objectOf(m)
}

// TestInputsConstraints is W3-NB1: the generator settles the input
// constraints' conditionals over the inputs object as it settles an
// object's (ADR-007 D9 items 9 and 15), so an optional input is present
// exactly when the other inputs' values make the constraints require it,
// in every optional mode and at every seed, and mixed reaches both
// branches. Deciding each optional input by its include stream alone, as
// Generate does, supplies the table beside an edition that refuses it.
func TestInputsConstraints(t *testing.T) {
	p := compile(t, editionFiles)
	constraints, ok := p.c.Schema(constraintsEntry)
	if !ok {
		t.Fatal("the constraints do not compile")
	}
	branches := map[bool]bool{}
	for _, mode := range []Mode{All, None, Mixed} {
		for seed := range uint64(32) {
			gs := p.gen(Options{Seed: seed, Optional: mode, Items: 1}).Inputs(editionInputs, constraintsEntry)
			obj := inputsObject(t, editionInputs, gs)
			if fails, err := constraints.Validate(obj); err != nil || len(fails) > 0 {
				t.Errorf("%s seed %d: the inputs %s fail the constraints: %v %v", mode, seed, text(obj), fails, err)
			}
			if mode == Mixed {
				branches[gs[1].Present] = true
			}
		}
	}
	if !branches[true] || !branches[false] {
		t.Errorf("mixed supplies the table %v over 32 seeds, want both with and without it", branches)
	}
}

// TestInputsWithoutConstraints holds Inputs with no constraints file to
// Generate's value and presence for every input.
func TestInputsWithoutConstraints(t *testing.T) {
	p := compile(t, editionFiles)
	for seed := range uint64(8) {
		opts := Options{Seed: seed, Optional: Mixed, Items: 1}
		gs := p.gen(opts).Inputs(editionInputs, "")
		for i, in := range editionInputs {
			v, present, fs, err := p.gen(opts).Generate(in)
			if g := gs[i]; g.Present != present || text(g.Value) != text(v) || len(g.Findings) != len(fs) || g.Err != err {
				t.Errorf("seed %d input %s: Inputs gives %v %s, Generate %v %s", seed, in.Name, g.Present, text(g.Value), present, text(v))
			}
		}
	}
}

// TestInputsSupplied holds a supplied edition as the fixed point the
// constraints are settled around (ADR-007 D9 item 2): b brings the table
// in under none, and a keeps it out under all.
func TestInputsSupplied(t *testing.T) {
	p := compile(t, editionFiles)
	for _, tc := range []struct {
		mode    Mode
		edition string
		table   bool
	}{{None, "b", true}, {All, "a", false}, {Mixed, "b", true}, {Mixed, "a", false}} {
		ins := []Input{editionInputs[0], editionInputs[1]}
		ins[0].Supplied = parse(t, `{"edition":"`+tc.edition+`"}`)
		gs := p.gen(Options{Optional: tc.mode, Items: 1}).Inputs(ins, constraintsEntry)
		inputsObject(t, ins, gs)
		if gs[1].Present != tc.table {
			t.Errorf("%s, edition %s: the table is present %v, want %v", tc.mode, tc.edition, gs[1].Present, tc.table)
		}
	}
}

// TestInputsConstrainValue holds the schemas a selected branch of the
// constraints gives an input to its value: under none, a b edition's
// branch requiring the report's name makes the report again with it, as
// the DORA incident report's stage requires the fields of a later stage.
func TestInputsConstrainValue(t *testing.T) {
	files := map[string]string{}
	for k, v := range editionFiles {
		files[k] = v
	}
	files[constraintsEntry] = `{"if":{"properties":{"report":{"required":["edition"],"properties":{"edition":{"const":"b"}}}}},
		"then":{"properties":{"report":{"required":["name"]}}}}`
	p := compile(t, files)
	constraints, _ := p.c.Schema(constraintsEntry)
	for _, edition := range []string{"a", "b"} {
		ins := []Input{editionInputs[0], editionInputs[1]}
		ins[0].Supplied = parse(t, `{"edition":"`+edition+`"}`)
		gs := p.gen(Options{Optional: None, Items: 1}).Inputs(ins, constraintsEntry)
		obj := inputsObject(t, ins, gs)
		if fails, err := constraints.Validate(obj); err != nil || len(fails) > 0 {
			t.Errorf("edition %s: the inputs %s fail the constraints: %v %v", edition, text(obj), fails, err)
		}
		_, named := gs[0].Value.(value.Object).Get("name")
		if named != (edition == "b") {
			t.Errorf("edition %s: the report %s, named %v", edition, text(gs[0].Value), named)
		}
	}
}

// TestInputsUnsatisfiable is ADR-007 D9 item 23 over the input
// constraints: an input they require and give the false schema is
// unsatisfiable, and conditionals that do not settle are too, each
// reported about the input and nothing written for it.
func TestInputsUnsatisfiable(t *testing.T) {
	for _, tc := range []struct {
		name, constraints, input, reason string
	}{
		{"required and refused", `{"required":["table"],"properties":{"table":false}}`, "table", ReasonType},
		{"never settles", `{"if":{"required":["table"]},"then":{"properties":{"table":false}},"else":{"required":["table"]}}`, "table", ReasonConditionals},
	} {
		t.Run(tc.name, func(t *testing.T) {
			files := map[string]string{}
			for k, v := range editionFiles {
				files[k] = v
			}
			files[constraintsEntry] = tc.constraints
			p := compile(t, files)
			gs := p.gen(Options{Items: 1}).Inputs(editionInputs, constraintsEntry)
			for i, g := range gs {
				if editionInputs[i].Name != tc.input {
					if len(g.Findings) > 0 {
						t.Errorf("input %s:%s", editionInputs[i].Name, describe(g.Findings))
					}
					continue
				}
				if got := reasons(t, g.Findings); !g.Present || len(got) != 1 || got[0] != tc.reason {
					t.Errorf("input %s: reasons %v, want %s%s", tc.input, got, tc.reason, describe(g.Findings))
				}
				if f := g.Findings; len(f) > 0 && f[0].Subject().Name != tc.input {
					t.Errorf("the finding is about %+v", f[0].Subject())
				}
			}
		})
	}
}
