// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package finding

import (
	"encoding/json"
	"strings"
	"testing"
)

// wcag is ADR-006 D10 item 2's and D11 item 1's example finding, with the
// condition D11's `when` line shows.
func wcag() Finding {
	loc := Location{Origin: OriginDataDir, File: "wcag_criteria.csv", Line: 7, Column: 41}.WithPointer("/5/remarks")
	return New(codeInputMinLength,
		Subject{Kind: SubjectInput, Name: "wcag_criteria", Title: "WCAG 2.2 criteria"},
		&loc,
		Params{"length": Int(4), "limit": Int(10)},
		WithSchema(Schema{Entry: "inputs/wcag_criteria.schema.json", Pointer: "/items/properties/remarks/minLength", Keyword: "minLength", KeywordLocation: "/items/properties/remarks/minLength"}),
		WithField(Field{Title: "Remarks", Description: "How the product meets, partly meets or fails the criterion, with the evidence."}),
		WithVia(Via{Keyword: "then", SchemaPointer: "/items/then", Condition: &Condition{SchemaPointer: "/items/if", Title: "Final report"}}),
	)
}

func TestJSONEncoding(t *testing.T) {
	withTestCodes(t)
	got := string(AppendJSON(nil, wcag()))
	want := `[{"code":"input.fixture_min_length","severity":"error",` +
		`"message":"Remarks must be at least 10 characters long; it has 4.",` +
		`"hint":"Write at least 10 characters.",` +
		`"subject":{"kind":"input","name":"wcag_criteria","title":"WCAG 2.2 criteria"},` +
		`"location":{"origin":"data_dir","file":"wcag_criteria.csv","line":7,"column":41,"pointer":"/5/remarks"},` +
		`"schema":{"entry":"inputs/wcag_criteria.schema.json","pointer":"/items/properties/remarks/minLength","keyword":"minLength","keyword_location":"/items/properties/remarks/minLength"},` +
		`"field":{"title":"Remarks","description":"How the product meets, partly meets or fails the criterion, with the evidence."},` +
		`"params":{"limit":10,"length":4},` +
		`"via":[{"keyword":"then","schema_pointer":"/items/then","condition":{"schema_pointer":"/items/if","title":"Final report"}}]}]`
	if got != want {
		t.Errorf("JSON\n got  %s\n want %s", got, want)
	}
	if !json.Valid([]byte(got)) {
		t.Error("the encoding is not valid JSON")
	}
}

func TestJSONEncodingMembers(t *testing.T) {
	withTestCodes(t)
	branch := New(codeInputType, input("v"), dataDir("v.json", 2, 3, "/kind"), Params{"expected": List(String("string")), "actual": String("number")})
	loc := Location{Origin: OriginInputsJSON, Argument: 1}.WithPointer("")
	f := New(codeInputType, input("v"), &loc, Params{"expected": List(String("object"), String("null")), "actual": String("array")},
		WithAlternatives(Alternative{Branch: 0, Title: "A card", Findings: []Finding{branch, branch}}, Alternative{Branch: 1, Omitted: 2}),
		WithVia(Via{Keyword: "oneOf", SchemaPointer: "/oneOf", Branch: 0}),
		WithRelated(Related{Location: Location{Origin: OriginDataDir, File: "v.json", Line: 1, Column: 1}.WithPointer("/0"), Note: "first occurrence"}),
		WithField(Field{}),
	)
	got := string(AppendJSON(nil, f))
	want := `[{"code":"input.fixture_type","severity":"error","message":"the input ` + "`v`" + ` must be object, null; it is array.","hint":"Write a value of type object, null.",` +
		`"subject":{"kind":"input","name":"v","title":"V"},` +
		`"location":{"origin":"inputs_json","argument":1,"pointer":""},` +
		`"params":{"expected":["object","null"],"actual":"array"},` +
		`"via":[{"keyword":"oneOf","schema_pointer":"/oneOf","branch":0}],` +
		`"alternatives":[{"branch":0,"title":"A card","findings":[` + strings.Trim(string(AppendJSON(nil, branch)), "[]") + `],"omitted":0},{"branch":1,"findings":[],"omitted":2}],` +
		`"related":[{"origin":"data_dir","file":"v.json","line":1,"column":1,"pointer":"/0","note":"first occurrence"}]}]`
	if got != want {
		t.Errorf("JSON\n got  %s\n want %s", got, want)
	}
	if !json.Valid([]byte(got)) {
		t.Error("the encoding is not valid JSON")
	}
}

// TestNeutralisation is ADR-006 D9 item 9: control and bidirectional
// characters in package text are \u escapes in both reports, and an
// invalid UTF-8 byte is U+FFFD.
func TestNeutralisation(t *testing.T) {
	withTestCodes(t)
	loc := Location{Origin: OriginDataDir, File: "a\x1b[2Jb\xff.csv", Line: 1, Column: 1}.WithPointer("/x")
	f := New(codeInputMinLength, Subject{Kind: SubjectInput, Name: "n", Title: "T\u202e\t"}, &loc,
		Params{"limit": Int(1), "length": Int(0)},
		WithField(Field{Title: "Q\"\\", Description: "D\u001b\u0085\u2066\u007f"}))
	j := string(AppendJSON(nil, f))
	for _, want := range []string{`"file":"a\u001b[2Jb` + "\ufffd" + `.csv"`, `"title":"T\u202e\u0009"`, `"title":"Q\"\\"`, `"description":"D\u001b\u0085\u2066\u007f"`} {
		if !strings.Contains(j, want) {
			t.Errorf("JSON lacks %s:\n%s", want, j)
		}
	}
	h := string(AppendHuman(nil, false, f))
	for _, want := range []string{"a\\u001b[2Jb\ufffd.csv:1:1: error:", "(input n: T\\u202e\\u0009)", `field: Q"\ — D\u001b\u0085\u2066\u007f`} {
		if !strings.Contains(h, want) {
			t.Errorf("human report lacks %q:\n%s", want, h)
		}
	}
	for _, s := range []string{j, h} {
		for _, r := range s {
			if r != '\n' && neutralised(r) {
				t.Errorf("an encoding holds %U raw", r)
			}
		}
	}
}

func TestHumanRendering(t *testing.T) {
	withTestCodes(t)
	got := string(AppendHuman(nil, false, wcag()))
	want := "wcag_criteria.csv:7:41: error: Remarks must be at least 10 characters long; it has 4. [input.fixture_min_length]\n" +
		"  at /wcag_criteria/5/remarks (input wcag_criteria: WCAG 2.2 criteria)\n" +
		"  field: Remarks — How the product meets, partly meets or fails the criterion, with the evidence.\n" +
		"  rule: minLength at /items/properties/remarks in inputs/wcag_criteria.schema.json\n" +
		"  when: Final report\n" +
		"  hint: Write at least 10 characters.\n"
	if got != want {
		t.Errorf("human\n got\n%s want\n%s", got, want)
	}
	coloured := string(AppendHuman(nil, true, wcag()))
	if !strings.Contains(coloured, ": \x1b[31merror\x1b[0m: ") || strings.Count(coloured, "\x1b[") != 2 {
		t.Errorf("colour is not on the severity word alone:\n%q", coloured)
	}
}

func TestHumanWhere(t *testing.T) {
	withTestCodes(t)
	at := func(l Location) *Location { return &l }
	tests := []struct {
		f    Finding
		want string
	}{
		{New(codeInputMinLength, input("a"), at(Location{Origin: OriginDataDir, File: "a.csv", Line: 3}.WithPointer("/0")), Params{"limit": Int(1), "length": Int(0)}), "a.csv:3: "},
		{New(codeInputMinLength, input("a"), at(Location{Origin: OriginDataDir, File: "a.txt"}.WithPointer("")), Params{"limit": Int(1), "length": Int(0)}), "a.txt: "},
		{New(codeInputMinLength, input("a"), at(Location{Origin: OriginSet, Argument: 2}.WithPointer("/x")), Params{"limit": Int(1), "length": Int(0)}), "--set #2: "},
		{New(codeInputMinLength, input("a"), at(Location{Origin: OriginInputsJSON, Argument: 1, Line: 1, Column: 4}.WithPointer("/x")), Params{"limit": Int(1), "length": Int(0)}), "--inputs-json #1: "},
		{New(codeInputMinLength, input("a"), at(Location{Origin: OriginDefault}.WithPointer("/x")), Params{"limit": Int(1), "length": Int(0)}), "input a: "},
		{New(codeTemplateParse, Subject{Kind: SubjectTemplate, Name: "t"}, at(Location{Origin: OriginPackage}), nil), "package: "},
		{New(codeUsageFlag, Subject{Kind: SubjectInvocation}, nil, nil), "metais: "},
	}
	for _, tt := range tests {
		if got := string(AppendHuman(nil, false, tt.f)); !strings.HasPrefix(got, tt.want) {
			t.Errorf("got %q, want it to begin %q", got, tt.want)
		}
	}
}

func TestHumanAlternatives(t *testing.T) {
	withTestCodes(t)
	branch := New(codeInputType, input("v"), dataDir("v.json", 2, 3, "/kind"), Params{"expected": List(String("string")), "actual": String("number")})
	loc := Location{Origin: OriginDataDir, File: "v.json", Line: 1, Column: 1}.WithPointer("")
	f := New(codeInputType, input("v"), &loc, Params{"expected": List(String("object")), "actual": String("array")},
		WithAlternatives(Alternative{Branch: 0, Title: "A card", Findings: []Finding{branch}}, Alternative{Branch: 1, Omitted: 3}))
	got := string(AppendHuman(nil, false, f))
	want := "v.json:1:1: error: the input `v` must be object; it is array. [input.fixture_type]\n" +
		"  at /v (input v: V)\n" +
		"  hint: Write a value of type object.\n" +
		"  option 1: A card\n" +
		"    v.json:2:3: error: `/kind` must be string; it is number. [input.fixture_type]\n" +
		"      at /v/kind (input v: V)\n" +
		"      hint: Write a value of type string.\n" +
		"  option 2\n" +
		"    3 more in this option\n"
	if got != want {
		t.Errorf("human\n got\n%s want\n%s", got, want)
	}
}

func TestCutDescription(t *testing.T) {
	long := strings.Repeat("é", 200)
	tests := []struct{ in, want string }{
		{"short", "short"},
		{"one\ntwo", "one…"},
		{"one\r\ntwo", "one…"},
		{"\nafter a break", ""},
		{long, strings.Repeat("é", 160) + "…"},
		{strings.Repeat("a", 160), strings.Repeat("a", 160)},
	}
	for _, tt := range tests {
		if got := cutDescription(tt.in); got != tt.want {
			t.Errorf("cutDescription(%.20q) = %.20q…, want %.20q…", tt.in, got, tt.want)
		}
	}
}

func TestSummaryHuman(t *testing.T) {
	tests := []struct {
		s    Summary
		want string
	}{
		{Summary{}, "0 errors, 0 warnings, 0 info\n"},
		{Summary{Errors: 2, Warnings: 1}, "2 errors, 1 warning, 0 info\n"},
		{Summary{Errors: 1, Warnings: 2, Info: 1, Listed: 100000, Omitted: 4120}, "1 error, 2 warnings, 1 info (100000 listed, 4120 more not shown; use --json for the totals by code)\n"},
	}
	for _, tt := range tests {
		if got := string(tt.s.AppendHuman(nil)); got != tt.want {
			t.Errorf("got %q, want %q", got, tt.want)
		}
	}
}
