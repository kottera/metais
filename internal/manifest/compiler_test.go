// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package manifest

import (
	"testing"

	"github.com/kottera/metais/internal/finding"
	"github.com/kottera/metais/internal/jsonx"
	"github.com/kottera/metais/internal/limits"
	"github.com/kottera/metais/internal/liquid/value"
	"github.com/kottera/metais/internal/pkgtree/pkgtreetest"
	"github.com/kottera/metais/internal/schema"
	"github.com/kottera/metais/internal/specdata"
)

// wrappedEntry is where the manifest schema is placed in the tree the
// closed compiler reads: a schema file under inputs/ (ADR-003 D3 item 1).
const wrappedEntry = "inputs/manifest.schema.json"

// closedSchema compiles the embedded manifest schema with W2-10's closed
// compiler (ADR-010 D5), wrapped in a package tree as its only schema
// file, and fails t on any finding: an unknown keyword, a format outside
// ADR-003 D4, a pattern outside the package profile, a reference that
// leaves the document.
func closedSchema(t testing.TB) *schema.Schema {
	t.Helper()
	tree, err := pkgtreetest.New(map[string][]byte{wrappedEntry: specdata.ManifestSchema10})
	if err != nil {
		t.Fatal(err)
	}
	decode := func(entry string, src []byte) (value.Value, []finding.Finding) {
		doc, fs := jsonx.Parse(src, jsonx.Source{
			Input: finding.Subject{Kind: finding.SubjectSchema, Name: entry},
			At:    finding.Location{Origin: finding.OriginPackage, File: entry},
		}, limits.Defaults())
		return doc.Value, fs
	}
	c, fs := schema.NewCompiler(tree, limits.Defaults(), decode)
	for _, f := range fs {
		l, _ := f.Location()
		t.Errorf("the closed compiler refuses the manifest schema: %s at %s: %s", f.Code(), l.Pointer, f.Message())
	}
	if c == nil {
		t.FailNow()
	}
	s, ok := c.Schema(wrappedEntry)
	if !ok {
		t.Fatalf("the compiler holds no %s", wrappedEntry)
	}
	return s
}

// TestClosedCompilerMeeting is W2-16's meeting point with W2-10: the
// manifest schema compiles under schema.NewCompiler's closed rules, and the
// compiled schema judges every manifest of TestRules as the model's rules
// do, the valid and the invalid alike. Each case has a compiler of its
// own, since the regex-work budget is a command's.
func TestClosedCompilerMeeting(t *testing.T) {
	isolate(t)
	closedSchema(t)
	for _, c := range ruleCases() {
		t.Run(c.name, func(t *testing.T) {
			m := base()
			c.change(t, m)
			root, _ := parse(t, encode(t, m))
			failures, err := closedSchema(t).Validate(root)
			if err != nil {
				t.Fatal(err)
			}
			_, buildErr := Build(root)
			schemaOK := len(failures) == 0
			if want := c.ptr == ok || c.byCheck; schemaOK != want {
				t.Errorf("the closed compiler accepts: %v, want %v (%d failures)", schemaOK, want, len(failures))
			}
			if schemaOK && len(Check(root, Source{})) == 0 && buildErr != nil {
				t.Errorf("the closed compiler and Check accept what Build refuses: %v", buildErr)
			}
			if !schemaOK && buildErr == nil {
				t.Error("Build accepts what the closed compiler refuses")
			}
		})
	}
}
