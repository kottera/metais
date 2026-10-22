// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package inputs

import (
	"slices"
	"strings"
	"testing"

	"metais.org/metais/internal/finding"
	"metais.org/metais/internal/limits"
	"metais.org/metais/internal/testenv"
)

// stagedPkg is PRD §4.2's staged incident report: the stage is an input,
// and which fields of the incident are required follows from it, by the
// input constraints of ADR-003 D6 item 2's example.
var stagedPkg = pkg{
	inputs: []map[string]any{
		slot("stage", "inputs/stage.json"),
		slot("incident", "inputs/incident.json", "required", false),
	},
	constraints: "inputs/constraints.schema.yaml",
	schemas: map[string]string{
		"inputs/stage.json":    `{"type": "string", "enum": ["draft", "final"]}`,
		"inputs/incident.json": `{"type": "object", "properties": {"root_cause": {"type": "string", "title": "Root cause"}, "resolution_date": {"type": "string", "format": "date"}}}`,
		"inputs/constraints.schema.yaml": "allOf:\n" +
			"  - if:   { title: Final report, properties: { stage: { const: final } }, required: [stage] }\n" +
			"    then:\n" +
			"      required: [incident]\n" +
			"      properties:\n" +
			"        incident: { required: [root_cause, resolution_date] }\n",
	},
}

// TestConstraints: the input constraints are evaluated against the object
// holding every input; "a finding at /incident/root_cause is reported
// against input incident at /root_cause" (ADR-003 D6 item 2), and a
// stage-dependent requirement reads "required (when: Final report)"
// through its via (ADR-006 D6 item 7).
func TestConstraints(t *testing.T) {
	testenv.Home(t)
	l := layer(t, stagedPkg)
	res := run(t, l, limits.Defaults(), args{flags: []string{"--set=/stage=draft"}})
	wantFindings(t, res.Findings)

	res = run(t, l, limits.Defaults(), args{flags: []string{"--set=/stage=final"}})
	wantFindings(t, res.Findings, `input.missing incident via:then`)
	if via := res.Findings[0].Via(); via[0].Condition == nil || via[0].Condition.Title != "Final report" || via[0].Condition.SchemaPointer != "/allOf/0/if" {
		t.Errorf("via %+v", via[0])
	}

	res = run(t, l, limits.Defaults(), args{data: dataDir(t, map[string]string{"incident.json": `{"root_cause": "x"}`}), flags: []string{"--set=/stage=final"}})
	wantFindings(t, res.Findings, `input.required incident data_dir:incident.json:1:1 #/resolution_date {"property":"resolution_date"} via:then`)
	f := res.Findings[0]
	if s, _ := f.Schema(); s.Entry != "inputs/constraints.schema.yaml" || s.Pointer != "/allOf/0/then/properties/incident/required" {
		t.Errorf("schema %+v", s)
	}
}

// TestConstraintsNotEvaluated: "When any input is unparsed, the
// input_constraints subschema is not evaluated, and an info finding says
// so" (ADR-003 D22 item 4); the unparsed input "is left out of the
// instance and no required finding is reported for it, since its parse
// findings already are" (item 3).
func TestConstraintsNotEvaluated(t *testing.T) {
	testenv.Home(t)
	l := layer(t, stagedPkg)
	res := run(t, l, limits.Defaults(), args{data: dataDir(t, map[string]string{"stage.json": `"final`})})
	wantFindings(t, res.Findings,
		`json.syntax stage data_dir:stage.json:1:7 # {"expected":"closing quote","found":"end of input"}`,
		`input.constraints_not_evaluated`,
	)
	if f := res.Findings[1]; f.Severity() != finding.SeverityInfo {
		t.Errorf("%s is %s", f.Code(), f.Severity())
	}
	// An --inputs-json argument that does not parse outside any member
	// leaves which inputs it supplies unknown: no input is judged missing,
	// and the constraints are not evaluated.
	res = run(t, l, limits.Defaults(), args{flags: []string{`--inputs-json={"stage": "final"`}})
	wantFindings(t, res.Findings,
		`input.json_arg_invalid inputs_json:1:1:18 {"problem":"json.syntax"}`,
		`input.constraints_not_evaluated`,
	)
}

// TestConstraintsOverTheWhole: a rule of the input constraints that fails
// at the root of the inputs object names no one input; it is reported
// about the supplied inputs together, with the keyword's schema location.
func TestConstraintsOverTheWhole(t *testing.T) {
	testenv.Home(t)
	l := layer(t, pkg{
		inputs: []map[string]any{
			slot("a", "inputs/any.json", "required", false),
			slot("b", "inputs/any.json", "required", false),
		},
		constraints: "inputs/c.json",
		schemas: map[string]string{
			"inputs/any.json": `{}`,
			"inputs/c.json":   `{"not": {"required": ["a", "b"]}, "anyOf": [{"required": ["a"]}, {"required": ["b"]}]}`,
		},
	})
	res := run(t, l, limits.Defaults(), args{flags: []string{"--set=/a=1", "--set=/b=2"}})
	wantFindings(t, res.Findings, `input.constraints_failed {"keyword":"not"}`)
	res = run(t, l, limits.Defaults(), args{})
	wantFindings(t, res.Findings, `input.constraints_failed {"keyword":"anyOf"}`)
	if s, ok := res.Findings[0].Schema(); !ok || s.Entry != "inputs/c.json" || s.Pointer != "/anyOf" {
		t.Errorf("schema %+v", s)
	}
}

// TestMissingInputs: "an absent required input is a finding at the input's
// root" (ADR-003 D6 item 1); "A command with no supply flag validates
// nothing supplied: every required input is reported missing, in one
// pass" (ADR-008 D7 item 3).
func TestMissingInputs(t *testing.T) {
	testenv.Home(t)
	l := layer(t, pkg{
		inputs: []map[string]any{
			slot("a", "inputs/any.json"), slot("b", "inputs/any.json"), slot("c", "inputs/any.json", "required", false),
		},
		schemas: map[string]string{"inputs/any.json": `{}`},
	})
	res := run(t, l, limits.Defaults(), args{})
	wantFindings(t, res.Findings, `input.missing a`, `input.missing b`)
	if f := res.Findings[0]; f.Subject().Title != "A" {
		t.Errorf("subject %+v", f.Subject())
	}
}

// TestPartialCollection: "When some items of a collection (a file that
// does not parse) … fail to parse, the remaining items are validated
// against the input's full schema as an array with the failed items
// removed, and every finding is reported at the item's original index and
// file. Findings whose instance location is the array itself, and findings
// whose evaluation path passes through prefixItems or unevaluatedItems
// applied to the array, are suppressed … One input.validation.partial
// finding (info) says so" (ADR-006 D7 item 4).
func TestPartialCollection(t *testing.T) {
	testenv.Home(t)
	l := layer(t, pkg{
		inputs: []map[string]any{slot("items", "inputs/items.json", "collection", true)},
		schemas: map[string]string{"inputs/items.json": `{"type": "array", "minItems": 9,
			"prefixItems": [{"type": "object", "required": ["first"]}],
			"items": {"type": "object", "properties": {"n": {"type": "integer"}}}}`},
	})
	res := run(t, l, limits.Defaults(), args{data: dataDir(t, map[string]string{
		"items/a.json": `{"n": 1}`,
		"items/b.json": `{"n": `,
		"items/c.json": "{\n  \"n\": \"x\"\n}",
	})})
	wantFindings(t, res.Findings,
		`json.syntax items data_dir:items/b.json:1:7 #/1/n {"expected":"value","found":"end of input"}`,
		`input.type items data_dir:items/c.json:2:8 #/2/n {"expected":["integer"],"actual":"string"}`,
		`input.validation.partial items data_dir:items # {"unparsed":1,"suppressed":2}`,
	)
}

// TestPartialCSV: a CSV input with a record ADR-003 can reject alone, such
// as a wrong field count, is validated in part (ADR-006 D7 item 4), every
// finding at the row's original index and its line.
func TestPartialCSV(t *testing.T) {
	testenv.Home(t)
	l := layer(t, pkg{
		inputs: []map[string]any{slot("rows", "inputs/rows.json", "forms", []any{"csv"})},
		schemas: map[string]string{"inputs/rows.json": `{"type": "array", "maxItems": 1,
			"items": {"type": "object", "properties": {"n": {"type": "integer"}}, "required": ["n", "s"]}}`},
	})
	res := run(t, l, limits.Defaults(), args{data: dataDir(t, map[string]string{"rows.csv": "n,s\n1,a\n2\nx,b\n,c\n"})})
	wantFindings(t, res.Findings,
		`csv.field_count rows data_dir:rows.csv:3:1 #/1 {"expected":2,"found":1}`,
		`input.type rows data_dir:rows.csv:4:1 #/2/n {"expected":["integer"],"actual":"string"}`,
		`input.required rows data_dir:rows.csv:5:1 #/3/n {"property":"n"}`,
		`input.validation.partial rows data_dir:rows.csv # {"unparsed":1,"suppressed":1}`,
	)
	// A file whose header is refused contributes no value (ADR-003 D11
	// item 9).
	res = run(t, l, limits.Defaults(), args{data: dataDir(t, map[string]string{"rows.csv": "n,n\n1,2\n"})})
	if c := codes(res.Findings); strings.Join(c, " ") != "csv.header_duplicate" || !record(t, res, "rows").Unparsed {
		t.Errorf("codes %v", c)
	}
}

// TestRegexWorkSpent: once the command's regex-work is spent, no input's
// validation has an answer, and each input present says so with
// limit.exceeded (ADR-009 D3, D10 item 7).
func TestRegexWorkSpent(t *testing.T) {
	testenv.Home(t)
	lim := lowered(t, "regex-work=10")
	l := layer(t, pkg{
		inputs:  []map[string]any{slot("a", "inputs/p.json"), slot("b", "inputs/p.json")},
		schemas: map[string]string{"inputs/p.json": `{"type": "string", "pattern": "^a+$"}`},
		lim:     &lim,
	})
	res := run(t, l, lim, args{flags: []string{"--set=/a=" + strings.Repeat("a", 64), "--set=/b=b"}})
	wantFindings(t, res.Findings,
		`limit.exceeded a set:1 # {"limit":"regex-work","bound":10,"lowered":true}`,
		`limit.exceeded b set:2 # {"limit":"regex-work","bound":10,"lowered":true}`,
	)
}

// TestConstraintsNotEvaluatedWhenOthersFail: the input constraints are not
// evaluated while any input is unparsed, even when the inputs they read
// are all present and would fail them (ADR-003 D22 item 4). Here stage
// and incident are supplied and break the Final report rule, and a third
// input the rule never names did not parse: no constraint finding is
// reported, and the info finding says the constraints were skipped.
func TestConstraintsNotEvaluatedWhenOthersFail(t *testing.T) {
	testenv.Home(t)
	p := stagedPkg
	p.inputs = append(slices.Clone(stagedPkg.inputs), slot("other", "inputs/other.json", "required", false))
	p.schemas = map[string]string{"inputs/other.json": `{}`}
	for k, v := range stagedPkg.schemas {
		p.schemas[k] = v
	}
	l := layer(t, p)
	files := map[string]string{"incident.json": `{"root_cause": "x"}`}
	a := args{data: dataDir(t, files), flags: []string{"--set=/stage=final"}}
	res := run(t, l, limits.Defaults(), a)
	wantFindings(t, res.Findings, `input.required incident data_dir:incident.json:1:1 #/resolution_date {"property":"resolution_date"} via:then`)

	files["other.json"] = `{`
	a.data = dataDir(t, files)
	res = run(t, l, limits.Defaults(), a)
	wantFindings(t, res.Findings,
		`json.syntax other data_dir:other.json:1:2 # {"expected":"member name","found":"end of input"}`,
		`input.constraints_not_evaluated`,
	)
}
