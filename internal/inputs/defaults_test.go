// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package inputs

import (
	"testing"

	"github.com/kottera/metais/internal/limits"
	"github.com/kottera/metais/internal/testenv"
)

// defaultsPkg holds a default for every case of ADR-003 D19: an optional
// member, a required one, one under $ref, one on a $ref's target, one under
// allOf, one inside a default object, array items, a CSV row's column, a
// nested member whose parent has none, and an absent optional input's.
var defaultsPkg = pkg{
	inputs: []map[string]any{
		slot("cfg", "inputs/cfg.json", "required", false),
		slot("rows", "inputs/rows.json", "forms", []any{"csv"}, "required", false),
		slot("opt", "inputs/opt.json", "required", false),
		slot("req", "inputs/opt.json"),
	},
	schemas: map[string]string{
		"inputs/cfg.json": `{"type": "object",
			"required": ["must"],
			"properties": {
				"plain": {"type": "string", "default": "p"},
				"must": {"type": "string", "default": "never"},
				"keep": {"type": ["string", "null"], "default": "k"},
				"viaref": {"$ref": "#/$defs/withdefault"},
				"own": {"$ref": "#/$defs/withdefault", "default": "own"},
				"box": {"type": "object", "default": {}, "properties": {"inner": {"default": 1}}},
				"parent": {"type": "object", "properties": {"child": {"default": 2}}},
				"list": {"type": "array", "items": {"type": "object", "properties": {"d": {"default": 3}}}}
			},
			"allOf": [{"properties": {"cond": {"default": "no"}}}],
			"$ref": "#/$defs/more",
			"$defs": {
				"withdefault": {"type": "string", "default": "target"},
				"more": {"properties": {"plain": {"default": "second"}, "fromref": {"default": "r"}}}
			}}`,
		"inputs/rows.json": `{"type": "array", "items": {"type": "object", "properties": {
			"s": {"type": "string"}, "remarks": {"type": "string", "default": "none"}}}}`,
		"inputs/opt.json": `{"type": "object", "default": {"from": "default"}}`,
	},
}

// TestDefaults: ADR-003 D19 item 1's walk, and items 2 to 4.
func TestDefaults(t *testing.T) {
	testenv.Home(t)
	l := layer(t, defaultsPkg)
	res := run(t, l, limits.Defaults(), args{data: dataDir(t, map[string]string{
		"cfg.json": `{"must": "given", "keep": null, "list": [{}, {"d": 0}]}`,
		"rows.csv": "s,remarks\na,\nb,\"\"\nc,yes\n",
		"req.json": `{}`,
	})})
	if len(res.Findings) > 0 {
		t.Fatalf("findings:\n%s", describe(res.Findings))
	}
	v := vals(res)
	// plain takes the schema object's own default before its $ref's; must
	// is required and takes none; null is present and kept; viaref takes
	// its $ref target's, own its own; box takes {} and then its inner
	// member's; parent is absent and is not created for its child; each
	// list item takes d where it is absent; cond, under allOf, takes none;
	// fromref, declared under the root's $ref, takes its own.
	if want := `{"box":{"inner":1},"fromref":"r","keep":null,"list":[{"d":3},{"d":0}],"must":"given","own":"own","plain":"p","viaref":"target"}`; v["cfg"] != want {
		t.Errorf("cfg\n got %s\nwant %s", v["cfg"], want)
	}
	// A blank CSV cell is absent and takes its column's default; "" is
	// present.
	if want := `[{"remarks":"none","s":"a"},{"remarks":"","s":"b"},{"remarks":"yes","s":"c"}]`; v["rows"] != want {
		t.Errorf("rows %s", v["rows"])
	}
	// An absent optional input takes its root schema's default; a present
	// one keeps its value.
	if v["opt"] != `{"from":"default"}` || v["req"] != `{}` {
		t.Errorf("opt %s, req %s", v["opt"], v["req"])
	}
	if r := record(t, res, "opt"); r.Source.Kind != SourceDefault || len(r.From) != 1 || r.From[0].Kind != FromDefault || !r.HasDigest {
		t.Errorf("opt's record %+v", r)
	}
	if r := record(t, res, "cfg"); len(r.From) != 2 || r.From[1].Kind != FromDefault {
		t.Errorf("cfg's From %+v", r.From)
	}
}

// TestDefaultsAbsence: an absent required input takes no default and is
// reported missing; "a default never replaces" null (ADR-003 D19 item 3,
// D20 item 2); "nothing is ever filled with "", [] or {}" (D20 item 3).
func TestDefaultsAbsence(t *testing.T) {
	testenv.Home(t)
	l := layer(t, defaultsPkg)
	res := run(t, l, limits.Defaults(), args{flags: []string{"--set-json=/opt=null"}})
	wantFindings(t, res.Findings,
		`input.missing req`,
		`input.type opt set:1 # {"expected":["object"],"actual":"null"}`,
	)
	v := vals(res)
	if v["req"] != "absent" || v["opt"] != "null" || v["cfg"] != "absent" || v["rows"] != "absent" {
		t.Errorf("values %v", v)
	}
}

// TestDefaultValidated: "A default is validated with the rest of the
// instance" (ADR-003 D19 item 5); a finding about a value a default
// supplied has origin default, and no file or line (ADR-006 D4 item 4,
// D5 item 5).
func TestDefaultValidated(t *testing.T) {
	testenv.Home(t)
	l := layer(t, pkg{
		inputs:  []map[string]any{slot("v", "inputs/v.json")},
		schemas: map[string]string{"inputs/v.json": `{"type": "object", "properties": {"n": {"type": "integer", "default": "bad"}}}`},
	})
	res := run(t, l, limits.Defaults(), args{data: dataDir(t, map[string]string{"v.yaml": "{}\n"})})
	wantFindings(t, res.Findings, `input.type v default #/n {"expected":["integer"],"actual":"string"}`)
}
