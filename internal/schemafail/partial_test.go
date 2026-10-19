// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package schemafail

import (
	"fmt"
	"strings"
	"testing"

	"github.com/kottera/metais/internal/finding"
)

const articlesSchema = `{
  "type": "array",
  "minItems": 5,
  "prefixItems": [{"title": "Cover", "type": "object"}],
  "items": {"properties": {"title": {"title": "Title", "minLength": 3}, "tags": {"prefixItems": [{"type": "string"}]}}},
  "unevaluatedItems": false
}`

// collection locates each item in its own file, as a collection does: the
// item's original index names it.
func collection(ptr string, key bool) finding.Location {
	toks := tokens(ptr)
	if len(toks) == 0 {
		return finding.Location{Origin: finding.OriginDataDir, File: "articles"}
	}
	return finding.Location{Origin: finding.OriginDataDir, File: "articles/" + toks[0] + ".md", Line: len(toks), Column: 1}
}

// TestPartial is ADR-006 D7 item 4: an array validated without its
// unparsed items reports every finding at the item's original index and
// file, suppresses those about the array itself and those through
// prefixItems or unevaluatedItems applied to it, and says so in one
// input.validation.partial finding.
func TestPartial(t *testing.T) {
	d := newDocs(t, map[string]string{"vendor.schema.json": articlesSchema})
	tg := target(d)
	tg.Subject = finding.Subject{Kind: finding.SubjectInput, Name: "articles"}
	tg.Locate = collection
	// Items 1 and 3 of five did not parse: validated items 0, 1 and 2 are
	// the original 0, 2 and 4.
	tg.Partial = &Partial{Kept: []int{0, 2, 4}, Unparsed: 2}
	fs := []Failure{
		at("items", "/items", nil, nil,
			at("minLength", "/items/properties/title/minLength", path("1", "title"), Data{"limit": 3, "length": 1}),
			// prefixItems applied to an item's member, not to the array.
			at("type", "/items/properties/tags/prefixItems/0/type", path("2", "tags", "0"), Data{"want": []string{"string"}, "got": "number"}),
			at(KeywordFalse, "/items", path("2"), Data{"prefix_count": 1}),
		),
		at("uniqueItems", "/uniqueItems", nil, Data{"duplicates": []int{2, 0}}),
		at("anyOf", "/items/anyOf", path("1"), Data{"branch_count": 2},
			at("type", "/items/anyOf/0/type", path("1"), Data{"want": []string{"null"}, "got": "object"}),
			at("required", "/items/anyOf/1/required", path("1"), Data{"missing": []string{"body"}})),

		// Suppressed: about the array itself,
		at("minItems", "/minItems", nil, Data{"limit": 5, "count": 3}),
		at("contains", "/contains", nil, nil),
		// through prefixItems applied to the array,
		at("prefixItems", "/prefixItems", nil, nil,
			at("type", "/prefixItems/0/type", path("0"), Data{"want": []string{"object"}, "got": "string"})),
		// and through unevaluatedItems applied to it, with or without a
		// node for it.
		at(KeywordFalse, "/unevaluatedItems", path("1"), nil),
		at("unevaluatedItems", "/unevaluatedItems", nil, nil, at(KeywordFalse, "/unevaluatedItems", path("2"), nil)),
	}
	got := mapAll(t, tg, fs...)
	checkBriefs(t, "partial", got,
		`input.validation.partial "" {"suppressed":5,"unparsed":2}`,
		`input.required "/2/body" {"property":"body"} via=anyOf@/items/anyOf#1`,
		`input.min_length "/2/title" {"length":1,"limit":3} field=Title|`,
		`input.additional_item "/4" {"index":4,"prefix_count":1}`,
		`input.unique_items "/4" {"duplicate_of":0,"index":4}`,
		`input.type "/4/tags/0" {"actual":"number","expected":["string"]}`,
	)
	var files []string
	for _, f := range got {
		l, _ := f.Location()
		files = append(files, l.File)
		if f.Code() == "input.unique_items" {
			r := f.Related()
			if len(r) != 1 || r[0].Location.Pointer != "/0" || r[0].Location.File != "articles/0.md" || r[0].Note != "first occurrence" {
				t.Errorf("unique_items related %+v", r)
			}
		}
	}
	if g := strings.Join(files, " "); g != "articles articles/2.md articles/2.md articles/4.md articles/4.md articles/4.md" {
		t.Errorf("files %s", g)
	}

	// The control: validated whole, nothing is suppressed or moved.
	tg.Partial = nil
	if n := len(mapAll(t, tg, fs...)); n != 10 {
		t.Errorf("validated whole, %d findings, want 10", n)
	}
}

// TestPartialCountsDistinct holds suppressed to distinct findings: one
// keyword reached by two paths is suppressed once, as it would be
// reported once.
func TestPartialCountsDistinct(t *testing.T) {
	d := newDocs(t, map[string]string{"vendor.schema.json": articlesSchema})
	tg := target(d)
	tg.Partial = &Partial{Kept: []int{0}, Unparsed: 1}
	minItems := func(kl string) Failure {
		f := at("minItems", "/minItems", nil, Data{"limit": 5, "count": 1})
		f.KeywordLocation = kl
		return f
	}
	got := mapAll(t, tg, minItems("/allOf/0/$ref/minItems"), minItems("/allOf/1/$ref/minItems"))
	if len(got) != 1 || fmt.Sprint(briefs(got)) != `[input.validation.partial "" {"suppressed":1,"unparsed":1}]` {
		t.Errorf("%v", briefs(got))
	}
}
