// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package schemafail

import (
	"strings"
	"testing"
)

// tree is a hand-built tree of the shape ADR-006 D6 maps: a $ref to an
// object whose required and minLength fail, beneath an allOf.
func tree() Failure {
	return Failure{
		Keyword: "allOf", KeywordLocation: "/allOf", Entry: "inputs/vendor.schema.json", SchemaPointer: "/allOf",
		Children: []Failure{
			{
				Keyword: "$ref", KeywordLocation: "/allOf/0/$ref", Entry: "inputs/vendor.schema.json", SchemaPointer: "/allOf/0/$ref",
				Children: []Failure{
					{Keyword: "required", KeywordLocation: "/allOf/0/$ref/required", Entry: "inputs/common.schema.json", SchemaPointer: "/required",
						Data: Data{"missing": []string{"name", "a/b~c"}}},
					{Keyword: "minLength", KeywordLocation: "/allOf/0/$ref/properties/remarks/minLength", Entry: "inputs/common.schema.json", SchemaPointer: "/properties/remarks/minLength",
						InstancePointer: []string{"remarks"}, Data: Data{"limit": Number("10"), "length": 4}},
				},
			},
			{Keyword: "type", KeywordLocation: "/allOf/1/type", Entry: "inputs/vendor.schema.json", SchemaPointer: "/allOf/1/type",
				InstancePointer: []string{"a/b~c", "0"}, Data: Data{"want": []string{"object"}, "got": "array"}},
		},
	}
}

func TestInstance(t *testing.T) {
	f := tree()
	if got := f.Instance(); got != "" {
		t.Errorf("root instance %q", got)
	}
	if got := f.Children[1].Instance(); got != "/a~1b~0c/0" {
		t.Errorf("instance %q, want /a~1b~0c/0", got)
	}
}

func TestWalk(t *testing.T) {
	f := tree()
	var seen []string
	f.Walk(func(path []*Failure, n *Failure) bool {
		var p []string
		for _, a := range path {
			p = append(p, a.Keyword)
		}
		seen = append(seen, strings.Join(append(p, n.Keyword), ">"))
		return true
	})
	want := []string{"allOf", "allOf>$ref", "allOf>$ref>required", "allOf>$ref>minLength", "allOf>type"}
	if strings.Join(seen, " ") != strings.Join(want, " ") {
		t.Errorf("walked %q, want %q", seen, want)
	}
	seen = nil
	f.Walk(func(path []*Failure, n *Failure) bool {
		seen = append(seen, n.Keyword)
		return n.Keyword != "$ref"
	})
	if got := strings.Join(seen, " "); got != "allOf $ref type" {
		t.Errorf("skipping $ref's children walked %q", got)
	}
}

func TestCheck(t *testing.T) {
	if err := tree().Check(); err != nil {
		t.Fatalf("a well-formed tree: %v", err)
	}
	bad := []func(*Failure){
		func(f *Failure) { f.Children[0].Children[1].Keyword = "" },
		func(f *Failure) { f.Children[1].SchemaPointer = "allOf/1" },
		func(f *Failure) { f.Children[1].Data["got"] = 1.5 },
		func(f *Failure) { f.Data = Data{"x": map[string]any{}} },
		func(f *Failure) { f.Children[0].Children[0].Data["missing"] = []any{"name"} },
	}
	for i, change := range bad {
		f := tree()
		change(&f)
		if err := f.Check(); err == nil {
			t.Errorf("change %d is not refused", i)
		}
	}
}
