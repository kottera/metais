// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package guarantees

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

// The meeting of the document tripwires with the conformance runner (W2-2,
// W2-3): both read vector.json, the runner against
// spec/schemas/conformance/1/vector.schema.json and these tripwires by the
// members ADR-011 D9 item 3 defines.

// vectorSchema compiles the published vector schema.
func vectorSchema(t *testing.T) *jsonschema.Schema {
	t.Helper()
	const rel = "conformance/1/vector.schema.json"
	b, err := os.ReadFile(filepath.Join(specRoot, schemasDir, filepath.FromSlash(rel)))
	if err != nil {
		t.Fatal(err)
	}
	doc, err := jsonschema.UnmarshalJSON(bytes.NewReader(b))
	if err != nil {
		t.Fatal(err)
	}
	c := jsonschema.NewCompiler()
	c.UseLoader(noFetch{})
	if err := c.AddResource(schemaBase+rel, doc); err != nil {
		t.Fatal(err)
	}
	s, err := c.Compile(schemaBase + rel)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

// TestVectorReaderMeetsTheSchema validates the fixture vector these
// tripwires read against the runner's vector schema, and reads it with
// readVectors: the members the tripwires use (statements, since, and each
// step's expected findings file) are those the schema defines, with the
// values it validated. Renaming any of them makes the schema refuse the
// vector, so the reader cannot drift from it unnoticed.
func TestVectorReaderMeetsTheSchema(t *testing.T) {
	const name = vectorsDir + "/out/relative-path/vector.json"
	raw := fixtureFiles[name]
	s := vectorSchema(t)
	validate := func(text string) error {
		inst, err := jsonschema.UnmarshalJSON(strings.NewReader(text))
		if err != nil {
			t.Fatal(err)
		}
		return s.Validate(inst)
	}
	if err := validate(raw); err != nil {
		t.Fatalf("the fixture vector fails the vector schema: %v", err)
	}

	var generic struct {
		ID         string   `json:"id"`
		Statements []string `json:"statements"`
		Since      string   `json:"since"`
		Cases      []struct {
			Steps []struct {
				Expect map[string]any `json:"expect"`
			} `json:"steps"`
		} `json:"cases"`
	}
	if err := json.Unmarshal([]byte(raw), &generic); err != nil {
		t.Fatal(err)
	}
	vs, err := readVectors(newSpecFixture(t).root)
	if err != nil {
		t.Fatal(err)
	}
	i := slices.IndexFunc(vs, func(v vector) bool { return v.dir == "out/relative-path" })
	if i < 0 {
		t.Fatal("readVectors did not read the fixture vector")
	}
	v := vs[i]
	if v.ID != generic.ID || !slices.Equal(v.Statements, generic.Statements) || v.Since != generic.Since {
		t.Errorf("readVectors read id %q, statements %q and since %q; the schema validated %q, %q and %q",
			v.ID, v.Statements, v.Since, generic.ID, generic.Statements, generic.Since)
	}
	var read, validated []string
	for ci, c := range v.Cases {
		for si, st := range c.Steps {
			read = append(read, st.Expect.Findings)
			f, _ := generic.Cases[ci].Steps[si].Expect["findings"].(string)
			validated = append(validated, f)
		}
	}
	if !slices.Equal(read, validated) || !slices.Contains(read, "expected/findings.json") {
		t.Errorf("readVectors read the findings files %q; the schema validated %q", read, validated)
	}

	for _, member := range []string{`"statements"`, `"since"`, `"findings"`} {
		renamed := strings.Replace(raw, member, strings.TrimSuffix(member, `"`)+`_x"`, 1)
		if validate(renamed) == nil {
			t.Errorf("the vector schema accepts a vector with %s renamed", member)
		}
	}
}

// TestConformanceDocumentsChecked shows the runner's published files are
// among those the document tripwires check, and pass:
// spec/conformance/README.md under TestKeywordsOnlyInStatements, and every
// schema of spec/schemas/conformance/1/ under TestSchemasPublished.
func TestConformanceDocumentsChecked(t *testing.T) {
	var readme *mdDoc
	for _, d := range mustDocs(t, specRoot) {
		if d.path == "conformance/README.md" {
			readme = d
		}
	}
	if readme == nil {
		t.Fatal("TestKeywordsOnlyInStatements does not read spec/conformance/README.md")
	}
	report(t, keywordProblems([]*mdDoc{readme}))

	published, err := treeFiles(filepath.Join(specRoot, schemasDir))
	if err != nil {
		t.Fatal(err)
	}
	for _, n := range []string{"vector", "findings", "driver-request", "driver-response", "record"} {
		rel := "conformance/1/" + n + ".schema.json"
		b, ok := published[rel]
		if !ok {
			t.Errorf("TestSchemasPublished does not read spec/%s/%s", schemasDir, rel)
			continue
		}
		report(t, schemaFileProblems(rel, b))
	}
}
