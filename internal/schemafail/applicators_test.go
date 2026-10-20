// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package schemafail

import (
	"fmt"
	"testing"
)

// paymentSchema is a tagged union by a const on a member, and a nullable
// string.
const paymentSchema = `{
  "properties": {
    "payment": {
      "title": "Payment",
      "anyOf": [
        {"title": "Card", "properties": {"kind": {"const": "card"}, "number": {"minLength": 12}}, "required": ["number"]},
        {"title": "Transfer", "properties": {"kind": {"const": "transfer"}, "iban": {"minLength": 15}}, "required": ["iban"]},
        {"$ref": "#/$defs/cash"}
      ]
    },
    "note": {"oneOf": [{"type": "null"}, {"title": "Text", "type": "string", "minLength": 3}]}
  },
  "if": {"title": "Final report", "properties": {"stage": {"const": "final"}}},
  "then": {"required": ["signature"]},
  "else": {"required": ["draft_of"]},
  "$defs": {"cash": {"title": "Cash", "properties": {"kind": {"const": "cash"}}}}
}`

const anyOfPtr = "/properties/payment/anyOf"

// branchFailure is a failure in branch b of the payment anyOf, at sub
// within the branch, on the payment's member inst.
func branchFailure(b int, sub, kw string, inst []string, data Data, children ...Failure) Failure {
	ptr := fmt.Sprintf("%s/%d%s", anyOfPtr, b, sub)
	return at(kw, ptr, append(path("payment"), inst...), data, children...)
}

func kindConst(b int, want string) Failure {
	return branchFailure(b, "/properties/kind/const", "const", path("kind"), Data{"expected": `"` + want + `"`})
}

// cashConst is branch 2's const failure, reached through its $ref.
func cashConst(withRef bool) Failure {
	f := Failure{Keyword: "const", KeywordLocation: anyOfPtr + "/2/$ref/properties/kind/const", Entry: "vendor.schema.json",
		SchemaPointer: "/$defs/cash/properties/kind/const", InstancePointer: path("payment", "kind"), Data: Data{"expected": `"cash"`}}
	if !withRef {
		return f
	}
	return Failure{Keyword: "$ref", KeywordLocation: anyOfPtr + "/2/$ref", Entry: "vendor.schema.json", SchemaPointer: anyOfPtr + "/2/$ref",
		InstancePointer: path("payment"), Children: []Failure{f}}
}

func payment(children ...Failure) Failure {
	return at("anyOf", anyOfPtr, path("payment"), Data{"branch_count": 3}, children...)
}

// TestChoiceCollapses is ADR-006 D6 item 6.3: a tagged union reports the
// one branch its discriminating member selects, as ordinary findings with
// a via naming the branch and its title.
func TestChoiceCollapses(t *testing.T) {
	d := newDocs(t, map[string]string{"vendor.schema.json": paymentSchema})
	for _, withRef := range []bool{true, false} {
		tree := payment(
			branchFailure(0, "/properties/number/minLength", "minLength", path("number"), Data{"limit": 12, "length": 3}),
			kindConst(1, "transfer"),
			branchFailure(1, "/required", "required", nil, Data{"missing": []string{"iban"}}),
			cashConst(withRef),
		)
		checkBriefs(t, fmt.Sprintf("card, with a node for $ref %v", withRef), mapAll(t, target(d), tree),
			`input.min_length "/payment/number" {"length":3,"limit":12} via=anyOf@/properties/payment/anyOf#0(Card)`)
	}

	// A type failure at the choice's own location rejects a branch: the
	// nullable string reports the string branch's finding.
	note := at("oneOf", "/properties/note/oneOf", path("note"), Data{"branch_count": 2},
		at("type", "/properties/note/oneOf/0/type", path("note"), Data{"want": []string{"null"}, "got": "string"}),
		at("minLength", "/properties/note/oneOf/1/minLength", path("note"), Data{"limit": 3, "length": 2}))
	checkBriefs(t, "nullable", mapAll(t, target(d), note),
		`input.min_length "/note" {"length":2,"limit":3} field=Text| via=oneOf@/properties/note/oneOf#1(Text)`)

	// A type failure at a member does not reject; an enum or a false
	// schema at a member does.
	tree := payment(
		branchFailure(0, "/properties/number/type", "type", path("number"), Data{"want": []string{"string"}, "got": "number"}),
		branchFailure(1, "/properties/kind/enum", "enum", path("kind"), Data{"allowed": []string{`"transfer"`}}),
		branchFailure(2, "/properties/kind", KeywordFalse, path("kind"), nil),
	)
	checkBriefs(t, "type at a member", mapAll(t, target(d), tree),
		`input.type "/payment/number" {"actual":"number","expected":["string"]} via=anyOf@/properties/payment/anyOf#0(Card)`)
}

// TestChoiceAlternatives is ADR-006 D6 item 6.4: when no branch, or more
// than one, survives, one any_of or one_of finding lists every branch in
// index order, each with its findings, sorted and capped at 16.
func TestChoiceAlternatives(t *testing.T) {
	d := newDocs(t, map[string]string{"vendor.schema.json": paymentSchema})
	all := payment(kindConst(0, "card"), kindConst(1, "transfer"), cashConst(true))
	checkBriefs(t, "every branch rejected", mapAll(t, target(d), all),
		`input.any_of "/payment" {"branch_count":3} field=Payment| alts=[`+
			`0(Card): input.const "/payment/kind" {"expected":"\"card\""} | `+
			`1(Transfer): input.const "/payment/kind" {"expected":"\"transfer\""} | `+
			`2(Cash): input.const "/payment/kind" {"expected":"\"cash\""}]`)

	var many []Failure
	for i := range 18 {
		many = append(many, branchFailure(0, fmt.Sprintf("/properties/m%02d/minLength", i), "minLength", path(fmt.Sprintf("m%02d", i)), Data{"limit": 1, "length": 0}))
	}
	two := payment(append(many,
		branchFailure(1, "/required", "required", nil, Data{"missing": []string{"iban"}}),
		kindConst(2, "cash"))...)
	got := briefs(mapAll(t, target(d), two))
	want := `input.any_of "/payment" {"branch_count":3} field=Payment| alts=[0(Card)+2: `
	for i := range 16 {
		if i > 0 {
			want += "; "
		}
		want += fmt.Sprintf(`input.min_length "/payment/m%02d" {"length":0,"limit":1}`, i)
	}
	want += ` | 1(Transfer): input.required "/payment/iban" {"property":"iban"} | 2(Cash): input.const "/payment/kind" {"expected":"\"cash\""}]`
	if len(got) != 1 || got[0] != want {
		t.Errorf("two branches surviving:\n got  %v\n want %s", got, want)
	}

	// The one survivor reports nothing: every branch is listed rather
	// than the failure lost.
	empty := payment(branchFailure(0, "/if", "if", nil, nil), kindConst(1, "transfer"), cashConst(false))
	checkBriefs(t, "a survivor with no finding", mapAll(t, target(d), empty),
		`input.any_of "/payment" {"branch_count":3} field=Payment| alts=[0(Card):  | `+
			`1(Transfer): input.const "/payment/kind" {"expected":"\"transfer\""} | `+
			`2(Cash): input.const "/payment/kind" {"expected":"\"cash\""}]`)

	oneOf := at("oneOf", "/properties/note/oneOf", path("note"), Data{"branch_count": 2},
		at("type", "/properties/note/oneOf/0/type", path("note"), Data{"want": []string{"null"}, "got": "number"}),
		at("type", "/properties/note/oneOf/1/type", path("note"), Data{"want": []string{"string"}, "got": "number"}))
	checkBriefs(t, "oneOf", mapAll(t, target(d), oneOf),
		`input.one_of "/note" {"branch_count":2} alts=[`+
			`0: input.type "/note" {"actual":"number","expected":["null"]} | `+
			`1(Text): input.type "/note" {"actual":"number","expected":["string"]} field=Text|]`)
}

// TestTransparent is ADR-006 D6 item 5: $ref, allOf, properties and the
// other applicators give no finding of their own, and their failures are
// reported as if they were not there, the evaluation path kept.
func TestTransparent(t *testing.T) {
	d := newDocs(t, map[string]string{"vendor.schema.json": paymentSchema})
	leaf := Failure{Keyword: "minLength", KeywordLocation: "/allOf/0/$ref/properties/a/items/minLength", Entry: "vendor.schema.json",
		SchemaPointer: "/$defs/x/properties/a/items/minLength", InstancePointer: path("a", "0"), Data: Data{"limit": 2, "length": 1}}
	wrap := func(kw, kl, sp string, inst []string, c Failure) Failure {
		return Failure{Keyword: kw, KeywordLocation: kl, Entry: "vendor.schema.json", SchemaPointer: sp, InstancePointer: inst, Children: []Failure{c}}
	}
	tree := wrap("allOf", "/allOf", "/allOf", nil,
		wrap("$ref", "/allOf/0/$ref", "/allOf/0/$ref", nil,
			wrap("properties", "/allOf/0/$ref/properties", "/$defs/x/properties", nil,
				wrap("items", "/allOf/0/$ref/properties/a/items", "/$defs/x/properties/a/items", path("a"), leaf))))
	for _, f := range []Failure{tree, leaf} {
		fs := mapAll(t, target(d), f)
		checkBriefs(t, "transparent", fs, `input.min_length "/a/0" {"length":1,"limit":2}`)
		if s, _ := fs[0].Schema(); s.KeywordLocation != leaf.KeywordLocation || s.Pointer != leaf.SchemaPointer {
			t.Errorf("schema %+v", s)
		}
	}
	for _, kw := range []string{"$dynamicRef", "patternProperties", "additionalProperties", "unevaluatedProperties", "prefixItems", "unevaluatedItems", "dependentSchemas"} {
		checkBriefs(t, kw, mapAll(t, target(d), wrap(kw, "/"+kw, "/"+kw, nil, leaf)), `input.min_length "/a/0" {"length":1,"limit":2}`)
	}
}

// TestConditions is ADR-006 D6 item 7: if gives no finding; a failure
// under then or else is reported with a via naming the condition and its
// title, and vias nest outermost first.
func TestConditions(t *testing.T) {
	d := newDocs(t, map[string]string{"vendor.schema.json": paymentSchema})
	then := at("then", "/then", nil, nil, at("required", "/then/required", nil, Data{"missing": []string{"signature"}}))
	els := at("else", "/else", nil, nil, at("required", "/else/required", nil, Data{"missing": []string{"draft_of"}}))
	cond := at("if", "/if", nil, nil, kindConst(0, "final"))
	checkBriefs(t, "then", mapAll(t, target(d), then, cond),
		`input.required "/signature" {"property":"signature"} via=then@/then</if:Final report>`)
	checkBriefs(t, "else", mapAll(t, target(d), els),
		`input.required "/draft_of" {"property":"draft_of"} via=else@/else</if:Final report>`)

	// A condition inside a branch the choice collapses to.
	nested := payment(
		branchFailure(0, "/then", "then", nil, nil, branchFailure(0, "/then/required", "required", nil, Data{"missing": []string{"cvc"}})),
		kindConst(1, "transfer"), cashConst(true))
	checkBriefs(t, "nested", mapAll(t, target(d), nested),
		`input.required "/payment/cvc" {"property":"cvc"} via=anyOf@/properties/payment/anyOf#0(Card),then@/properties/payment/anyOf/0/then</properties/payment/anyOf/0/if:>`)
}

// TestBooleanContexts is ADR-006 D6 item 10: nothing is reported from
// inside not's subschema, the if subschema, or the branches of a oneOf
// more than one of which matched; and the collapse rule does not look
// inside them for a discriminating failure.
func TestBooleanContexts(t *testing.T) {
	d := newDocs(t, map[string]string{"vendor.schema.json": paymentSchema})
	typeAt := func(ptr string) Failure {
		return at("type", ptr, path("payment"), Data{"want": []string{"object"}, "got": "string"})
	}
	cases := []struct {
		name string
		f    Failure
		want []string
	}{
		{"not", at("not", "/properties/payment/not", path("payment"), nil, typeAt("/properties/payment/not/type")),
			[]string{`input.not "/payment" {} field=Payment|`}},
		{"if", at("if", "/if", nil, nil, typeAt("/if/type")), nil},
		{"a oneOf two of whose branches matched", at("oneOf", "/properties/payment/oneOf", path("payment"),
			Data{"branch_count": 2, "matched": []int{0, 1}}, typeAt("/properties/payment/oneOf/0/type")),
			[]string{`input.one_of.ambiguous "/payment" {"branches":[0,1]} field=Payment|`}},
		{"propertyNames", at("propertyNames", "/propertyNames", nil, Data{"property": "x"}, typeAt("/propertyNames/type")),
			[]string{`input.property_name "/x" {"property":"x"}`}},
		{"a discriminating failure inside not, if or a nested choice", payment(
			branchFailure(0, "/not", "not", nil, nil, branchFailure(0, "/not/type", "type", nil, Data{"want": []string{"object"}, "got": "string"})),
			branchFailure(0, "/if", "if", nil, nil, kindConst(0, "x")),
			branchFailure(0, "/anyOf", "anyOf", nil, Data{"branch_count": 1}, branchFailure(0, "/anyOf/0/const", "const", nil, Data{"expected": `"x"`})),
			kindConst(1, "transfer"), cashConst(false)),
			[]string{
				`input.any_of "/payment" {"branch_count":1} field=Card| via=anyOf@/properties/payment/anyOf#0(Card) alts=[0: input.const "/payment" {"expected":"\"x\""} field=Card|]`,
				`input.not "/payment" {} field=Card| via=anyOf@/properties/payment/anyOf#0(Card)`,
			}},
	}
	for _, c := range cases {
		checkBriefs(t, c.name, mapAll(t, target(d), c.f), c.want...)
	}
}

// TestChoiceListedAtMost is MTS-FND-020 in the mapping: a choice is built
// listing at most the target's Listed, itself and its alternatives'
// findings together, so a report's findings limit bounds what one finding
// holds; the findings past it are counted in their branch's omitted.
func TestChoiceListedAtMost(t *testing.T) {
	d := newDocs(t, map[string]string{"vendor.schema.json": paymentSchema})
	all := payment(kindConst(0, "card"), kindConst(1, "transfer"), cashConst(true))
	tg := target(d)
	tg.Listed = 2
	checkBriefs(t, "listed at most 2", mapAll(t, tg, all),
		`input.any_of "/payment" {"branch_count":3} field=Payment| alts=[`+
			`0(Card): input.const "/payment/kind" {"expected":"\"card\""} | `+
			`1(Transfer)+1:  | `+
			`2(Cash)+1: ]`)
}
