// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package schemafail

import (
	"fmt"
	"strconv"
	"strings"
	"testing"

	"metais.org/metais/internal/finding"
)

// keywordSchema is the schema the keyword cases are written in.
const keywordSchema = `{
  "properties": {
    "name": {"type": "string"},
    "nickname": {"type": "string"},
    "secret": false
  },
  "additionalProperties": false
}`

// TestKeywordCodes is ADR-006 D6 items 3 and 4: one finding per failed
// assertion, named by its keyword, with its params; required,
// dependentRequired, additionalProperties: false and items: false split
// one per member or item.
func TestKeywordCodes(t *testing.T) {
	d := newDocs(t, map[string]string{"vendor.schema.json": keywordSchema})
	var many []string
	for i := range 25 {
		many = append(many, fmt.Sprintf(`"v%02d"`, i))
	}
	var quoted []string
	for _, m := range many[:20] {
		quoted = append(quoted, strconv.Quote(m))
	}
	cases := []struct {
		name string
		f    Failure
		want []string
	}{
		{"type", at("type", "/type", nil, Data{"want": []string{"object", "null"}, "got": "array"}),
			[]string{`input.type "" {"actual":"array","expected":["object","null"]}`}},
		{"enum", at("enum", "/properties/name/enum", path("name"), Data{"allowed": []string{`"draft"`, `"final"`, `3`, `null`}, "got": "Draft "}),
			[]string{`input.enum "/name" {"allowed":["\"draft\"","\"final\"","3","null"],"allowed_count":4,"suggestion":"draft"}`}},
		{"enum, the first 20 of 25", at("enum", "/enum", nil, Data{"allowed": many}),
			[]string{`input.enum "" {"allowed":[` + strings.Join(quoted, ",") + `],"allowed_count":25}`}},
		{"const", at("const", "/const", nil, Data{"expected": `{"kind":"card"}`}),
			[]string{`input.const "" {"expected":"{\"kind\":\"card\"}"}`}},
		{"required, one per member", at("required", "/required", nil, Data{"missing": []string{"name", "a/b~c"}}),
			[]string{`input.required "/a~1b~0c" {"property":"a/b~c"}`, `input.required "/name" {"property":"name"}`}},
		{"dependentRequired", at("dependentRequired", "/dependentRequired", nil, Data{"missing": []string{"name"}, "because": "nickname"}),
			[]string{`input.dependent_required "/name" {"because":"nickname","property":"name"}`}},
		{"additionalProperties: false, one per member, with a suggestion", at("additionalProperties", "/additionalProperties", nil, nil,
			at(KeywordFalse, "/additionalProperties", path("nme"), nil),
			at(KeywordFalse, "/additionalProperties", path("zzz"), nil)),
			[]string{`input.additional_property "/nme" {"property":"nme","suggestion":"name"}`, `input.additional_property "/zzz" {"property":"zzz"}`}},
		{"unevaluatedProperties: false", at(KeywordFalse, "/unevaluatedProperties", path("Name"), nil),
			[]string{`input.unevaluated_property "/Name" {"property":"Name","suggestion":"name"}`}},
		{"propertyNames, its subschema unread", at("propertyNames", "/propertyNames", nil, Data{"property": "Bad Name"},
			at("pattern", "/propertyNames/pattern", nil, Data{"pattern": "^[a-z]+$"})),
			[]string{`input.property_name "/Bad Name" {"property":"Bad Name"}`}},
		{"minProperties", at("minProperties", "/minProperties", nil, Data{"limit": 2, "count": 1}),
			[]string{`input.min_properties "" {"count":1,"limit":2}`}},
		{"maxProperties", at("maxProperties", "/maxProperties", nil, Data{"limit": 2, "count": 3}),
			[]string{`input.max_properties "" {"count":3,"limit":2}`}},
		{"minLength", at("minLength", "/properties/name/minLength", path("name"), Data{"limit": Number("10"), "length": 4}),
			[]string{`input.min_length "/name" {"length":4,"limit":10}`}},
		{"maxLength", at("maxLength", "/properties/name/maxLength", path("name"), Data{"limit": 3, "length": 4}),
			[]string{`input.max_length "/name" {"length":4,"limit":3}`}},
		{"pattern", at("pattern", "/properties/name/pattern", path("name"), Data{"pattern": "^[A-Z]{2}$"}),
			[]string{`input.pattern "/name" {"pattern":"^[A-Z]{2}$"}`}},
		{"format", at("format", "/properties/name/format", path("name"), Data{"format": "email"}),
			[]string{`input.format "/name" {"format":"email"}`}},
		{"minimum", at("minimum", "/minimum", nil, Data{"limit": Number("1.50"), "value": Number("1e0")}),
			[]string{`input.minimum "" {"limit":1.5,"value":1}`}},
		{"maximum", at("maximum", "/maximum", nil, Data{"limit": 10, "value": Number("10.5")}),
			[]string{`input.maximum "" {"limit":10,"value":10.5}`}},
		{"exclusiveMinimum", at("exclusiveMinimum", "/exclusiveMinimum", nil, Data{"limit": 0, "value": 0}),
			[]string{`input.exclusive_minimum "" {"limit":0,"value":0}`}},
		{"exclusiveMaximum", at("exclusiveMaximum", "/exclusiveMaximum", nil, Data{"limit": Number("1E21"), "value": Number("2e21")}),
			[]string{`input.exclusive_maximum "" {"limit":1e+21,"value":2e+21}`}},
		{"multipleOf", at("multipleOf", "/multipleOf", nil, Data{"divisor": Number("0.01"), "value": Number("0.015")}),
			[]string{`input.multiple_of "" {"divisor":0.01,"value":0.015}`}},
		{"minItems", at("minItems", "/minItems", nil, Data{"limit": 1, "count": 0}),
			[]string{`input.min_items "" {"count":0,"limit":1}`}},
		{"maxItems", at("maxItems", "/maxItems", nil, Data{"limit": 1, "count": 2}),
			[]string{`input.max_items "" {"count":2,"limit":1}`}},
		{"uniqueItems, at the later item", at("uniqueItems", "/uniqueItems", nil, Data{"duplicates": []int{4, 1}}),
			[]string{`input.unique_items "/4" {"duplicate_of":1,"index":4}`}},
		{"contains", at("contains", "/contains", nil, nil),
			[]string{`input.contains "" {}`}},
		{"contains, its subschema transparent", at("contains", "/contains", nil, nil,
			at("type", "/contains/type", path("0"), Data{"want": []string{"string"}, "got": "number"})),
			[]string{`input.contains "" {}`, `input.type "/0" {"actual":"number","expected":["string"]}`}},
		{"minContains", at("minContains", "/minContains", nil, Data{"limit": 2, "count": 1}),
			[]string{`input.min_contains "" {"count":1,"limit":2}`}},
		{"maxContains", at("maxContains", "/maxContains", nil, Data{"limit": 2, "count": 3}),
			[]string{`input.max_contains "" {"count":3,"limit":2}`}},
		{"items: false after prefixItems, one per item", at("items", "/items", nil, nil,
			at(KeywordFalse, "/items", path("2"), Data{"prefix_count": 2}),
			at(KeywordFalse, "/items", path("3"), Data{"prefix_count": 2})),
			[]string{`input.additional_item "/2" {"index":2,"prefix_count":2}`, `input.additional_item "/3" {"index":3,"prefix_count":2}`}},
		{"unevaluatedItems: false", at(KeywordFalse, "/unevaluatedItems", path("1"), nil),
			[]string{`input.unevaluated_item "/1" {"index":1}`}},
		{"a false schema under properties", at(KeywordFalse, "/properties/secret", path("secret"), nil),
			[]string{`input.forbidden "/secret" {"property":"secret"}`}},
		{"a false schema elsewhere", at(KeywordFalse, "/allOf/1", nil, nil),
			[]string{`input.forbidden "" {}`}},
		{"a false root schema", at(KeywordFalse, "", nil, nil),
			[]string{`input.forbidden "" {}`}},
		{"a false schema under a member named items", at(KeywordFalse, "/properties/items", path("items"), nil),
			[]string{`input.forbidden "/items" {"property":"items"}`}},
		{"oneOf with two branches matching", at("oneOf", "/oneOf", nil, Data{"matched": []int{0, 2}, "branch_count": 3}),
			[]string{`input.one_of.ambiguous "" {"branches":[0,2]}`}},
		{"the content vocabulary is not asserted", at("allOf", "/allOf", nil, nil,
			at("contentEncoding", "/allOf/0/contentEncoding", nil, nil),
			at("contentMediaType", "/allOf/0/contentMediaType", nil, nil),
			at("contentSchema", "/allOf/0/contentSchema", nil, nil, at("type", "/allOf/0/contentSchema/type", nil, Data{"want": []string{"object"}, "got": "string"}))),
			nil},
	}
	for _, c := range cases {
		checkBriefs(t, c.name, mapAll(t, target(d), c.f), c.want...)
	}
}

// TestSchemaMember is ADR-006 D6 item 8: the finding names the keyword's
// entry, its pointer within the entry, its name and the evaluation path,
// and a false schema is named by the applicator that reached it.
func TestSchemaMember(t *testing.T) {
	d := newDocs(t, map[string]string{"vendor.schema.json": keywordSchema})
	tree := Failure{Keyword: "$ref", KeywordLocation: "/properties/vendor/$ref", Entry: "vendor.schema.json", SchemaPointer: "/properties/vendor/$ref",
		InstancePointer: path("vendor"), Children: []Failure{
			{Keyword: "minLength", KeywordLocation: "/properties/vendor/$ref/minLength", Entry: "common.schema.json", SchemaPointer: "/$defs/name/minLength",
				InstancePointer: path("vendor"), Data: Data{"limit": 1, "length": 0}},
			{Keyword: KeywordFalse, KeywordLocation: "/properties/vendor/$ref/additionalProperties", Entry: "common.schema.json", SchemaPointer: "/$defs/name/additionalProperties",
				InstancePointer: path("vendor", "x")},
		}}
	var got []string
	for _, f := range mapAll(t, target(d), tree) {
		s, _ := f.Schema()
		got = append(got, fmt.Sprintf("%s %s %s %s", s.Entry, s.Pointer, s.Keyword, s.KeywordLocation))
	}
	want := []string{
		"common.schema.json /$defs/name/minLength minLength /properties/vendor/$ref/minLength",
		"common.schema.json /$defs/name/additionalProperties additionalProperties /properties/vendor/$ref/additionalProperties",
	}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Errorf("schema members:\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
}

// TestSpans is ADR-006 D5 items 2 and 6 as the mapping applies them: a
// missing member is pointed at, with its parent's line; an extra member's
// line and column are its name's; every other finding's are its value's.
func TestSpans(t *testing.T) {
	d := newDocs(t, map[string]string{"vendor.schema.json": keywordSchema})
	tree := at("properties", "/properties", nil, nil,
		at("required", "/properties/a/required", path("a"), Data{"missing": []string{"name"}}),
		at(KeywordFalse, "/properties/a/additionalProperties", path("a", "x"), nil),
		at("minLength", "/properties/b/minLength", path("b"), Data{"limit": 2, "length": 1}),
	)
	var got []string
	for _, f := range mapAll(t, target(d), tree) {
		l, _ := f.Location()
		got = append(got, fmt.Sprintf("%s %s %d:%d", f.Code(), l.Pointer, l.Line, l.Column))
	}
	// locate puts a value at line depth+1, column 1, and a name at column 3.
	want := "input.required /a/name 2:1, input.min_length /b 2:1, input.additional_property /a/x 3:3"
	if g := strings.Join(got, ", "); g != want {
		t.Errorf("got %s\nwant %s", g, want)
	}
}

// TestFamilies is ADR-006 D6 item 1: the same mapping for an output and
// the manifest, under their prefixes and about their subjects.
func TestFamilies(t *testing.T) {
	d := newDocs(t, map[string]string{"vendor.schema.json": keywordSchema})
	tree := at("required", "/required", nil, Data{"missing": []string{"name"}})
	for _, c := range []struct {
		family Family
		subj   finding.Subject
		want   string
	}{
		{FamilyInput, vendor, "input.required"},
		{FamilyOutput, finding.Subject{Kind: finding.SubjectOutput, Name: "report"}, "output.required"},
		{FamilyManifest, finding.Subject{Kind: finding.SubjectManifest}, "manifest.required"},
	} {
		tg := target(d)
		tg.Family, tg.Subject = c.family, c.subj
		fs := mapAll(t, tg, tree)
		if len(fs) != 1 || fs[0].Code() != finding.Code(c.want) || fs[0].Subject() != c.subj {
			t.Errorf("%s: %v", c.family, briefs(fs))
		}
	}
}

// TestMapRefusals holds Map to refusing a tree no adapter may give, and a
// target it cannot report for.
func TestMapRefusals(t *testing.T) {
	d := newDocs(t, map[string]string{"vendor.schema.json": keywordSchema})
	for _, c := range []struct {
		name string
		tg   func(Target) Target
		f    Failure
		want string
	}{
		{"an unknown keyword", nil, at("requried", "/requried", nil, nil), "a keyword the mapping does not know"},
		{"a missing field", nil, at("minLength", "/minLength", nil, Data{"limit": 1}), "length is not a number"},
		{"a field of the wrong type", nil, at("required", "/required", nil, Data{"missing": []int{1}}), "missing is not a list of strings"},
		{"a number that is not one", nil, at("minimum", "/minimum", nil, Data{"limit": Number("1."), "value": 1}), "not a JSON number"},
		{"a branch child outside the branches", nil, at("anyOf", "/anyOf", nil, Data{"branch_count": 1},
			at("type", "/anyOf/1/type", nil, Data{"want": []string{"string"}, "got": "null"})), "in no branch of 1"},
		{"an enum's got that is not a string", nil, at("enum", "/enum", nil, Data{"allowed": []string{"1"}, "got": 1}), "got is a int"},
		{"duplicates that are not two", nil, at("uniqueItems", "/uniqueItems", nil, Data{"duplicates": []int{1}}), "not two indexes"},
		{"a uniqueKeys with no keys", nil, at("uniqueKeys", "/uniqueKeys", nil, Data{"duplicates": []int{0, 1}}), "keys is not a list of strings"},
		{"a coverage with no key", nil, at("coverage", "/coverage", nil, Data{"missing": "1", "keys": []string{}}), "names no key pointer"},
		{"a refused item with no index", nil, at(KeywordFalse, "/items", path("x"), Data{"prefix_count": 0}), "no index"},
		{"a tree Check refuses", nil, Failure{Keyword: ""}, "no keyword"},
		{"no Locate", func(tg Target) Target { tg.Locate = nil; return tg }, at("not", "/not", nil, nil), "incomplete target"},
		{"no family", func(tg Target) Target { tg.Family = 0; return tg }, at("not", "/not", nil, nil), "incomplete target"},
		{"an output in part", func(tg Target) Target {
			tg.Family, tg.Subject, tg.Partial = FamilyOutput, finding.Subject{Kind: finding.SubjectOutput, Name: "o"}, &Partial{}
			return tg
		}, at("not", "/not", nil, nil), "only an input"},
		{"an index beyond the items validated", func(tg Target) Target { tg.Partial = &Partial{Kept: []int{0}, Unparsed: 1}; return tg },
			at("type", "/items/type", path("3"), Data{"want": []string{"string"}, "got": "null"}), "beyond the 1 items"},
	} {
		tg := target(d)
		if c.tg != nil {
			tg = c.tg(tg)
		}
		_, err := Map(tg, c.f)
		if err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: %v, want an error holding %q", c.name, err, c.want)
		}
	}
}
