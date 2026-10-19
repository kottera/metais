// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package inputs

import (
	"strings"
	"testing"

	"github.com/kottera/metais/internal/limits"
	"github.com/kottera/metais/internal/testenv"
)

// supplyPkg declares vendor, an object with typed members, one of them
// reached through a $ref, and list, an array, neither required.
var supplyPkg = pkg{
	inputs: []map[string]any{
		slot("vendor", "inputs/vendor.json", "required", false),
		slot("list", "inputs/list.json", "required", false),
	},
	schemas: map[string]string{
		"inputs/vendor.json": `{"type": "object", "properties": {
			"name": {"type": "string"}, "count": {"type": "integer"}, "flag": {"type": "boolean"},
			"ratio": {"$ref": "#/$defs/ratio"}, "tags": {"type": "array", "items": {"type": "string"}},
			"address": {"type": "object", "properties": {"city": {"type": "string"}, "zip": {"type": "string"}}}},
			"$defs": {"ratio": {"type": ["number", "null"]}}}`,
		"inputs/list.json": `{"type": "array", "items": {"type": "object", "properties": {"n": {"type": "integer"}}}}`,
	},
}

// vals is each input's value by name.
func vals(res *Result) map[string]string {
	out := map[string]string{}
	for _, r := range res.Inputs {
		out[r.Name] = canonical(r.Value)
	}
	return out
}

// TestInputsJSON: "--inputs-json supplies one JSON object whose members
// are input names … It follows D9's rules. A non-object, or a member naming
// no declared input, is a finding" (ADR-003 D17 item 1). A value from it
// carries the line and column within the argument's text (ADR-006 D5 item
// 5), and its argument is the ordinal among every --inputs-json.
func TestInputsJSON(t *testing.T) {
	testenv.Home(t)
	l := layer(t, supplyPkg)
	res := run(t, l, limits.Defaults(), args{flags: []string{
		`--inputs-json={"vendor": {"name": "Acme", "count": 1}}`,
		"--inputs-json=[1]",
		"--inputs-json={\n  \"vendr\": {}, \"list\": [{\"n\": \"x\"}]}",
	}})
	wantFindings(t, res.Findings,
		`input.json_arg_not_object inputs_json:2:1:1 {"actual":"array"}`,
		`input.json_arg_unknown vendr inputs_json:3:2:3`,
		`input.type list inputs_json:3:2:31 #/0/n {"expected":["integer"],"actual":"string"}`,
	)
	if v := vals(res); v["vendor"] != `{"count":1,"name":"Acme"}` {
		t.Errorf("vendor = %s", v["vendor"])
	}
	// A finding inside a member is about that member's input, at its
	// pointer within the input; one outside every member is about the
	// argument, and its inputs are not known, so no constraint and no
	// missing input is judged.
	res = run(t, l, limits.Defaults(), args{flags: []string{
		`--inputs-json={"vendor": {"name": "A", "name": "B"}}`,
		`--inputs-json={"list": [1,]}`,
		`--inputs-json={"vendor": {}} x`,
		`--inputs-json={"vendr": {"a": 1, "a": 2}}`,
	}})
	wantFindings(t, res.Findings,
		`json.duplicate_name vendor inputs_json:1:1:26 #/name {"name":"name"}`,
		`input.json_arg_unknown vendr inputs_json:4:1:2`,
		`input.json_arg_invalid inputs_json:4:1:20 {"problem":"json.duplicate_name"}`,
		`json.syntax list inputs_json:2:1:13 #/1 {"expected":"value","found":"unexpected character"}`,
		`input.json_arg_invalid inputs_json:3:1:16 {"problem":"json.trailing_content"}`,
	)
	for _, name := range []string{"vendor", "list"} {
		if r := record(t, res, name); !r.Unparsed {
			t.Errorf("%s: an argument that did not parse inside it left it parsed: %s", name, canonical(r.Value))
		}
	}
}

// TestPathSetPointers: "A pointer not beginning with /, or whose first
// token names no declared input, is a finding" (ADR-003 D17 item 2); the
// argument is split at its first =; a set's argument is its ordinal among
// every --set and --set-json, in command-line order (ADR-006 D4 item 4).
func TestPathSetPointers(t *testing.T) {
	testenv.Home(t)
	l := layer(t, supplyPkg)
	res := run(t, l, limits.Defaults(), args{flags: []string{
		"--set=/vendor/name",
		"--set-json=vendor/name=1",
		"--set=/vendor/a~2b=1",
		"--set-json=/vendr/name=1",
		"--set=/=1",
		"--set=/vendor/name=a=b",
	}})
	wantFindings(t, res.Findings,
		`input.set_pointer_invalid set:1 {"reason":"no_value"}`,
		`input.set_pointer_invalid set:2 {"reason":"not_a_pointer"}`,
		`input.set_pointer_invalid set:3 {"reason":"malformed"}`,
		`input.set_pointer_invalid set:4 {"reason":"unknown_input"}`,
		`input.set_pointer_invalid set:5 {"reason":"unknown_input"}`,
	)
	if v := vals(res); v["vendor"] != `{"name":"a=b"}` {
		t.Errorf("vendor = %s", v["vendor"])
	}
}

// TestPathSetTraversal: ADR-003 D17 item 4. A missing object member is
// created as an empty object; an index must be an existing one without a
// leading zero; - is accepted only as the last token, and appends;
// traversing into a scalar, an index out of range, and - before the last
// token are findings, and the set is not applied. Arrays are never
// created implicitly.
func TestPathSetTraversal(t *testing.T) {
	testenv.Home(t)
	l := layer(t, supplyPkg)
	res := run(t, l, limits.Defaults(), args{
		data: dataDir(t, map[string]string{"list.json": `[{"n": 1}, {"n": 2}]`}),
		flags: []string{
			"--set=/vendor/address/city=Lyon",
			"--set=/vendor/address/city/x=1",
			"--set=/list/01/n=3",
			"--set=/list/x/n=3",
			"--set=/list/2/n=3",
			"--set=/list/-/n=3",
			"--set-json=/list/-={\"n\": 4}",
			"--set=/list/1/n=5",
			"--set=/vendor/tags/0=a",
		},
	})
	wantFindings(t, res.Findings,
		`input.set_traversal vendor set:2 {"reason":"scalar"}`,
		`input.set_traversal list set:3 {"reason":"index_invalid"}`,
		`input.set_traversal list set:4 {"reason":"index_invalid"}`,
		`input.set_traversal list set:5 {"reason":"index_out_of_range"}`,
		`input.set_traversal list set:6 {"reason":"dash_not_last"}`,
		`input.type vendor set:9 #/tags {"expected":["array"],"actual":"object"}`,
	)
	v := vals(res)
	if v["vendor"] != `{"address":{"city":"Lyon"},"tags":{"0":"a"}}` || v["list"] != `[{"n":1},{"n":5},{"n":4}]` {
		t.Errorf("values %v", v)
	}
}

// TestTextSetCoercion: a text set's value "is coerced by D14 against the
// static subschema at the pointer … A text set with an empty value sets
// the empty string, then coerced"; a JSON set "is placed as it is — the
// only way to set null, an array or an object at a path" (ADR-003 D17
// item 3).
func TestTextSetCoercion(t *testing.T) {
	testenv.Home(t)
	l := layer(t, supplyPkg)
	res := run(t, l, limits.Defaults(), args{flags: []string{
		"--set=/vendor/count=42",
		"--set=/vendor/flag=TRUE",
		"--set=/vendor/name=42",
		"--set=/vendor/ratio=1.5e2",
		"--set=/vendor/address/zip=",
		"--set=/vendor/x=true",
		"--set-json=/list=[{\"n\": null}]",
		"--set=/list/0/n=007",
	}})
	if v := vals(res); v["vendor"] != `{"address":{"zip":""},"count":42,"flag":true,"name":"42","ratio":150,"x":"true"}` || v["list"] != `[{"n":"007"}]` {
		t.Errorf("values %v", v)
	}
	wantFindings(t, res.Findings, `input.type list set:8 #/0/n {"expected":["integer"],"actual":"string"}`)

	res = run(t, l, limits.Defaults(), args{flags: []string{
		"--set=/vendor/count=9007199254740993",
		"--set-json=/vendor/ratio=null",
	}})
	wantFindings(t, res.Findings, `input.number_out_of_range vendor set:1 #/count {"reason":"unsafe_integer"}`)
	if v := vals(res); v["vendor"] != `{"ratio":null}` {
		t.Errorf("vendor = %s", v["vendor"])
	}
	// A --set-json value that does not parse leaves its input unparsed, and
	// no set into it is applied.
	res = run(t, l, limits.Defaults(), args{flags: []string{
		"--set-json=/vendor/tags=[\"a\", ]",
		"--set=/vendor/name=x",
	}})
	wantFindings(t, res.Findings,
		`json.syntax vendor set:1:1:7 #/tags/1 {"expected":"value","found":"unexpected character"}`,
		`input.set_into_unparsed vendor set:2`,
	)
	if r := record(t, res, "vendor"); !r.Unparsed {
		t.Errorf("a --set-json value that did not parse left vendor parsed: %s", canonical(r.Value))
	}
	// ADR-009 D7 item 6: nothing is expanded.
	res = run(t, l, limits.Defaults(), args{flags: []string{"--set=/vendor/name=$HOME ${HOME} %HOME% ~ @x"}})
	if v := vals(res); v["vendor"] != `{"name":"$HOME ${HOME} %HOME% ~ @x"}` {
		t.Errorf("vendor = %s", v["vendor"])
	}
}

// TestPrecedence: ADR-003 D18. The data directory, then every
// --inputs-json in command-line order, each member replacing its input's
// whole value, then every path set in command-line order, replacing only
// the value at its pointer; "every --inputs-json applies before every
// path set, wherever they appear on the line; among one kind, the later
// wins". Item 3's example.
func TestPrecedence(t *testing.T) {
	testenv.Home(t)
	l := layer(t, supplyPkg)
	dir := dataDir(t, map[string]string{"vendor.json": `{"name": "Acme", "address": {"city": "Lyon", "zip": "69001"}}`})
	for _, c := range []struct {
		flags []string
		want  string
	}{
		{[]string{"--set=/vendor/address/city=Paris"}, `{"address":{"city":"Paris","zip":"69001"},"name":"Acme"}`},
		{[]string{`--inputs-json={"vendor": {"address": {"city": "Paris"}}}`}, `{"address":{"city":"Paris"}}`},
		{[]string{"--set=/vendor/name=Set", `--inputs-json={"vendor": {"name": "Json"}}`}, `{"name":"Set"}`},
		{[]string{`--inputs-json={"vendor": {"name": "A"}}`, `--inputs-json={"vendor": {"name": "B"}}`}, `{"name":"B"}`},
		{[]string{"--set=/vendor/name=A", "--set-json=/vendor/name=\"B\"", "--set=/vendor/name=C"}, `{"address":{"city":"Lyon","zip":"69001"},"name":"C"}`},
	} {
		res := run(t, l, limits.Defaults(), args{data: dir, flags: c.flags})
		if v := vals(res)["vendor"]; v != c.want || len(res.Findings) > 0 {
			t.Errorf("%q: vendor = %s, want %s\n%s", c.flags, v, c.want, describe(res.Findings))
		}
	}
	// "The origin of a value is the source that won the precedence merge,
	// recorded per value" (ADR-006 D4 item 5).
	res := run(t, l, limits.Defaults(), args{data: dir, flags: []string{"--set=/vendor/name=1", "--set-json=/vendor/address/zip=5"}})
	wantFindings(t, res.Findings, `input.type vendor set:2 #/address/zip {"expected":["string"],"actual":"number"}`)
	res = run(t, l, limits.Defaults(), args{data: dir, flags: []string{"--set-json=/vendor/address={\"zip\": 5}"}})
	wantFindings(t, res.Findings, `input.type vendor set:1 #/address/zip {"expected":["string"],"actual":"number"}`)
}

// TestSetIntoUnparsed: "A path set into an input whose file did not parse
// is a finding and is not applied. A file that did not parse stays a
// finding even when --inputs-json replaces its input" (ADR-003 D18 item
// 4).
func TestSetIntoUnparsed(t *testing.T) {
	testenv.Home(t)
	l := layer(t, supplyPkg)
	dir := dataDir(t, map[string]string{"vendor.json": `{"name": `})
	res := run(t, l, limits.Defaults(), args{data: dir, flags: []string{"--set=/vendor/name=A"}})
	wantFindings(t, res.Findings,
		`json.syntax vendor data_dir:vendor.json:1:10 #/name {"expected":"value","found":"end of input"}`,
		`input.set_into_unparsed vendor set:1`,
	)
	if r := record(t, res, "vendor"); !r.Unparsed || r.Value != nil {
		t.Errorf("vendor %+v", r)
	}
	res = run(t, l, limits.Defaults(), args{data: dir, flags: []string{`--inputs-json={"vendor": {"count": "x"}}`, "--set=/vendor/name=A"}})
	wantFindings(t, res.Findings,
		`json.syntax vendor data_dir:vendor.json:1:10 #/name {"expected":"value","found":"end of input"}`,
		`input.set_into_unparsed vendor set:1`,
		`input.type vendor inputs_json:1:1:22 #/count {"expected":["integer"],"actual":"string"}`,
	)
}

// TestSupplyArgumentLimits: input-size bounds each --inputs-json, --set and
// --set-json value, and inputs-total every byte; value-depth bounds a set's
// pointer and what it places (ADR-009 D3).
func TestSupplyArgumentLimits(t *testing.T) {
	testenv.Home(t)
	l := layer(t, supplyPkg)
	res := run(t, l, lowered(t, "input-size=8"), args{flags: []string{
		"--set=/vendor/name=123456789",
		`--inputs-json={"list": []}`,
	}})
	wantFindings(t, res.Findings,
		`limit.exceeded vendor set:1 {"limit":"input-size","bound":8,"lowered":true,"observed":9}`,
		`input.json_arg_invalid inputs_json:1:1:1 {"problem":"limit.exceeded","limit":"input-size"}`,
	)
	res = run(t, l, lowered(t, "inputs-total=12"), args{flags: []string{`--inputs-json={"list": []}`, "--set=/vendor/name=x"}})
	wantFindings(t, res.Findings, `limit.exceeded vendor set:1 {"limit":"inputs-total","bound":12,"lowered":true}`)
	res = run(t, l, lowered(t, "value-depth=3"), args{flags: []string{
		"--set=/vendor/a/b/c/d=x",
		"--set-json=/vendor/a=[[[1]]]",
		"--set-json=/vendor/b=[[1]]",
	}})
	wantFindings(t, res.Findings,
		`limit.exceeded vendor set:1 {"limit":"value-depth","bound":3,"lowered":true,"observed":4}`,
		`limit.exceeded vendor set:2 {"limit":"value-depth","bound":3,"lowered":true,"observed":4}`,
	)
	if !strings.Contains(vals(res)["vendor"], `"b":[[1]]`) {
		t.Errorf("vendor = %s", vals(res)["vendor"])
	}
}

// TestNumberLengthAfterCoercion: number-length bounds "one numeric
// literal" (ADR-009 D3) wherever it is read, so a CSV field or a --set text
// that coerces to a number is held to it as a JSON or YAML literal is: a
// million-digit literal would cost arbitrary-precision arithmetic in
// multipleOf and comparisons.
func TestNumberLengthAfterCoercion(t *testing.T) {
	testenv.Home(t)
	l := layer(t, pkg{
		inputs: []map[string]any{
			slot("rows", "inputs/rows.json", "forms", []any{"csv"}),
			slot("v", "inputs/v.json"),
		},
		schemas: map[string]string{
			"inputs/rows.json": `{"type": "array", "items": {"type": "object", "properties": {"n": {"type": "number", "multipleOf": 3}}}}`,
			"inputs/v.json":    `{"type": "object", "properties": {"n": {"type": "number"}}}`,
		},
	})
	res := run(t, l, lowered(t, "number-length=4"), args{
		data:  dataDir(t, map[string]string{"rows.csv": "n\n12345\n6\n"}),
		flags: []string{"--set=/v/n=1.5e10", "--set=/v/m=1"},
	})
	wantFindings(t, res.Findings,
		`limit.exceeded rows data_dir:rows.csv:2:1 #/0/n {"limit":"number-length","bound":4,"lowered":true,"observed":5}`,
		`input.validation.partial rows data_dir:rows.csv # {"unparsed":1,"suppressed":0}`,
		`limit.exceeded v set:1 #/n {"limit":"number-length","bound":4,"lowered":true,"observed":6}`,
	)
}

// TestFailedInputsJSONHidesItsMembers: an --inputs-json argument that does
// not parse gives no value, so the inputs its other members would have
// supplied are not judged missing, and the constraints are not evaluated
// (ADR-003 D22 items 3 and 4).
func TestFailedInputsJSONHidesItsMembers(t *testing.T) {
	testenv.Home(t)
	l := layer(t, pkg{
		inputs:      []map[string]any{slot("a", "inputs/any.json"), slot("b", "inputs/any.json")},
		constraints: "inputs/any.json",
		schemas:     map[string]string{"inputs/any.json": `{}`},
	})
	res := run(t, l, limits.Defaults(), args{flags: []string{`--inputs-json={"a": {"x": }, "b": 1}`}})
	wantFindings(t, res.Findings,
		`json.syntax a inputs_json:1:1:13 #/x {"expected":"value","found":"unexpected character"}`,
		`input.constraints_not_evaluated`,
	)
}
