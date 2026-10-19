// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package schema

import (
	"testing"

	"github.com/kottera/metais/internal/limits"
	"github.com/kottera/metais/internal/pkgtree/pkgtreetest"
)

// TestOutputCompiler is ADR-005 D11 item 3's $ref resolution for an output
// schema: ADR-003 D5 with inputs/ read as the top-level directory holding
// the validate file, whose files are loaded only as its references reach
// them, so a file there that is no schema is never read as one.
func TestOutputCompiler(t *testing.T) {
	base := map[string]string{
		"references/out.json":     `{"type": "object", "properties": {"a": {"$ref": "parts/a.json"}}}`,
		"references/parts/a.json": `{"$ref": "b.yaml#/$defs/n"}`,
		"references/parts/b.yaml": `{"$defs": {"n": {"type": "integer"}}}`,
		"references/notes.json":   `{"requried": true}`,
		"inputs/in.schema.json":   `{"type": "string"}`,
	}
	for _, tc := range []struct {
		name  string
		scope string
		roots []string
		extra map[string]string
		want  string
	}{
		{name: "reached by path", scope: "references/", roots: []string{"references/out.json"}},
		{name: "escapes the scope", scope: "references/", roots: []string{"references/x.json"},
			extra: map[string]string{"references/x.json": `{"$ref": "../inputs/in.schema.json"}`},
			want:  "schema.ref_escapes@references/x.json#/$ref"},
		{name: "into the compiled form", scope: "", roots: []string{"x.json"},
			extra: map[string]string{"x.json": `{"$ref": ".compiled/schema.json"}`, ".compiled/schema.json": `{}`},
			want:  "schema.ref_escapes@x.json#/$ref"},
		{name: "resolves to nothing", scope: "references/", roots: []string{"references/x.json"},
			extra: map[string]string{"references/x.json": `{"$ref": "missing.json"}`},
			want:  "schema.ref_unresolved@references/x.json#/$ref"},
		{name: "a root that is no schema", scope: "references/", roots: []string{"references/notes.json"},
			want: "schema.keyword_unknown@references/notes.json#/requried"},
		{name: "the package root", scope: "", roots: []string{"x.json"},
			extra: map[string]string{"x.json": `{"$ref": "references/out.json"}`}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := map[string][]byte{}
			for k, v := range base {
				m[k] = []byte(v)
			}
			for k, v := range tc.extra {
				m[k] = []byte(v)
			}
			tree, err := pkgtreetest.New(m)
			if err != nil {
				t.Fatal(err)
			}
			c, fs := NewOutputCompiler(tree, tc.scope, tc.roots, limits.Defaults(), decodeJSON, nil)
			if got := describe(fs); got != tc.want {
				t.Fatalf("findings:\n%s\nwant:\n%s", got, tc.want)
			}
			if tc.want != "" {
				return
			}
			s, ok := c.Schema(tc.roots[0])
			if !ok {
				t.Fatalf("no schema %s", tc.roots[0])
			}
			if _, ok := c.Schema("references/notes.json"); ok {
				t.Error("a file no reference reaches was compiled")
			}
			if fails, err := s.Validate(mustValue(t, `{"a": 1}`)); err != nil || len(fails) > 0 {
				t.Errorf("valid instance: %v %v", fails, err)
			}
			if tc.name == "reached by path" {
				if fails, _ := s.Validate(mustValue(t, `{"a": "x"}`)); len(fails) == 0 {
					t.Error("a value the referenced schema refuses passed")
				}
			}
		})
	}
}
