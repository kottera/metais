// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package schemafail

import (
	"math/rand/v2"
	"slices"
	"testing"

	"metais.org/metais/internal/finding"
)

// orderCase is a target and the failures validated against it.
type orderCase struct {
	name string
	tg   Target
	fs   []Failure
}

// orderCases are trees whose findings depend on more than one child of a
// node: choices collapsed and listed, conditions, splits, and an array
// validated in part.
func orderCases(t *testing.T) []orderCase {
	pay := newDocs(t, map[string]string{"vendor.schema.json": paymentSchema})
	fields := newDocs(t, map[string]string{"vendor.schema.json": fieldSchema, "common.schema.json": commonSchema})
	arts := newDocs(t, map[string]string{"vendor.schema.json": articlesSchema})
	partial := target(arts)
	partial.Locate = collection
	partial.Partial = &Partial{Kept: []int{0, 2, 4}, Unparsed: 2}
	var many []Failure
	for i := range 20 {
		many = append(many, branchFailure(0, "/properties/m/minLength", "minLength", path("m"+string(rune('a'+i))), Data{"limit": 1, "length": 0}))
	}
	return []orderCase{
		{"collapsed", target(pay), []Failure{payment(
			branchFailure(0, "/properties/number/minLength", "minLength", path("number"), Data{"limit": 12, "length": 3}),
			branchFailure(0, "/required", "required", nil, Data{"missing": []string{"cvc", "expiry", "holder"}}),
			kindConst(1, "transfer"),
			branchFailure(1, "/required", "required", nil, Data{"missing": []string{"iban"}}),
			cashConst(true),
		)}},
		{"listed, one branch over the cap", target(pay), []Failure{payment(append(many, kindConst(1, "transfer"),
			branchFailure(1, "/properties/iban/minLength", "minLength", path("iban"), Data{"limit": 15, "length": 2}),
			branchFailure(2, "/required", "required", nil, Data{"missing": []string{"a", "b"}}))...)}},
		{"conditions", target(pay), []Failure{
			at("then", "/then", nil, nil, at("required", "/then/required", nil, Data{"missing": []string{"signature", "date"}})),
			at("allOf", "/allOf", nil, nil,
				at("minLength", "/allOf/0/minLength", nil, Data{"limit": 1, "length": 0}),
				at("type", "/allOf/1/type", nil, Data{"want": []string{"string"}, "got": "number"}),
				at(KeywordFalse, "/allOf/2/additionalProperties", path("x"), nil),
				at(KeywordFalse, "/allOf/2/additionalProperties", path("y"), nil)),
		}},
		{"fields", target(fields), []Failure{
			ref("/properties/contact/$ref", "/properties/contact/$ref", path("contact"),
				common("required", "/properties/contact/$ref/required", "/$defs/contact/required", path("contact"), Data{"missing": []string{"email", "phone"}}),
				common("type", "/properties/contact/$ref/type", "/$defs/contact/type", path("contact"), Data{"want": []string{"object"}, "got": "string"})),
			at("allOf", "/properties/remarks/allOf", path("remarks"), nil,
				at("minLength", "/properties/remarks/allOf/0/minLength", path("remarks"), Data{"limit": 10, "length": 4})),
		}},
		{"partial", partial, []Failure{
			at("items", "/items", nil, nil,
				at("minLength", "/items/properties/title/minLength", path("1", "title"), Data{"limit": 3, "length": 1}),
				at(KeywordFalse, "/items", path("2"), Data{"prefix_count": 1})),
			at("minItems", "/minItems", nil, Data{"limit": 5, "count": 3}),
			at("uniqueItems", "/uniqueItems", nil, Data{"duplicates": []int{2, 0}}),
			at(KeywordFalse, "/unevaluatedItems", path("1"), nil),
		}},
	}
}

// shuffled returns a copy of fs with the failures, and every node's
// children, in an order r chooses.
func shuffled(fs []Failure, r *rand.Rand) []Failure {
	out := make([]Failure, len(fs))
	for i, f := range fs {
		f.Children = shuffled(f.Children, r)
		out[i] = f
	}
	r.Shuffle(len(out), func(i, j int) { out[i], out[j] = out[j], out[i] })
	return out
}

func reversed(fs []Failure) []Failure {
	out := make([]Failure, len(fs))
	for i, f := range fs {
		f.Children = reversed(f.Children)
		out[len(fs)-1-i] = f
	}
	return out
}

// orderProblems maps each case's failures in their order, reversed, and
// shuffled by fixed seeds, through mapper, and names each case whose
// findings differ.
func orderProblems(t *testing.T, mapper func(Target, ...Failure) ([]finding.Finding, error)) []string {
	var bad []string
	for _, c := range orderCases(t) {
		encode := func(fs []Failure) string {
			got, err := mapper(c.tg, fs...)
			if err != nil {
				t.Fatalf("%s: %v", c.name, err)
			}
			r := finding.NewReport(10000)
			for _, f := range got {
				r.Add(f)
			}
			return string(finding.AppendJSON(nil, r.Findings()...))
		}
		want := encode(c.fs)
		orders := [][]Failure{reversed(c.fs)}
		for seed := range uint64(40) {
			orders = append(orders, shuffled(c.fs, rand.New(rand.NewPCG(seed, 0x5eed))))
		}
		for _, fs := range orders {
			if encode(fs) != want {
				bad = append(bad, c.name)
				break
			}
		}
	}
	return bad
}

// TestChildOrder is the mapping's half of ADR-006 D8 item 7: the findings
// do not depend on the order of a tree's children, nor of the failures
// given, proved by permuting them.
func TestChildOrder(t *testing.T) {
	if bad := orderProblems(t, Map); len(bad) > 0 {
		t.Errorf("the findings depend on the children's order in %v", bad)
	}
}

// TestChildOrderControl is the detector's control: a mapping that reads
// only each node's first child, as one that stops at the first error the
// library lists would, is caught.
func TestChildOrderControl(t *testing.T) {
	var firstOnly func(fs []Failure) []Failure
	firstOnly = func(fs []Failure) []Failure {
		out := slices.Clone(fs)
		for i := range out {
			out[i].Children = firstOnly(out[i].Children[:min(1, len(out[i].Children))])
		}
		return out
	}
	bad := orderProblems(t, func(tg Target, fs ...Failure) ([]finding.Finding, error) { return Map(tg, firstOnly(fs)...) })
	if len(bad) == 0 {
		t.Error("a mapping that reads only the first child is not caught")
	}
}
