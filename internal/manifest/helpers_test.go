// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package manifest

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/santhosh-tekuri/jsonschema/v6"

	"metais.org/metais/internal/finding"
	"metais.org/metais/internal/jsonx"
	"metais.org/metais/internal/limits"
	"metais.org/metais/internal/liquid/value"
	"metais.org/metais/internal/specdata"
	"metais.org/metais/internal/testenv"
)

// schemaID is the 1.0 manifest schema's $id (ADR-002 D10 item 2).
const schemaID = "https://metais.org/schemas/1.0/manifest.schema.json"

// noFetch refuses every load, so compiling the schema never reaches the
// network: the schema is self-contained (ADR-002 D10 item 3).
type noFetch struct{}

func (noFetch) Load(url string) (any, error) { return nil, fmt.Errorf("no fetch: %s", url) }

var (
	schemaOnce sync.Once
	schemaDoc  *jsonschema.Schema
	schemaErr  error
)

// manifestSchema compiles the embedded manifest schema, once, with the
// JSON Schema library directly: these tests are where ADR-002 D10's schema
// is held to the model, and the library is never called outside them.
func manifestSchema(t testing.TB) *jsonschema.Schema {
	t.Helper()
	schemaOnce.Do(func() {
		doc, err := jsonschema.UnmarshalJSON(bytes.NewReader(specdata.ManifestSchema10))
		if err != nil {
			schemaErr = err
			return
		}
		c := jsonschema.NewCompiler()
		c.UseLoader(noFetch{})
		c.DefaultDraft(jsonschema.Draft2020)
		if err := c.AddResource(schemaID, doc); err != nil {
			schemaErr = err
			return
		}
		schemaDoc, schemaErr = c.Compile(schemaID)
	})
	if schemaErr != nil {
		t.Fatalf("compiling the manifest schema: %v", schemaErr)
	}
	return schemaDoc
}

// schemaAccepts reports whether the manifest schema accepts the JSON text
// b, and why not.
func schemaAccepts(t testing.TB, b []byte) (bool, string) {
	t.Helper()
	inst, err := jsonschema.UnmarshalJSON(bytes.NewReader(b))
	if err != nil {
		t.Fatalf("instance: %v", err)
	}
	if err := manifestSchema(t).Validate(inst); err != nil {
		return false, err.Error()
	}
	return true, ""
}

// parse returns the value tree of the JSON text b, and its positions.
func parse(t testing.TB, b []byte) (value.Value, finding.Positions) {
	t.Helper()
	doc, fs := jsonx.Parse(b, jsonx.Source{
		Input: finding.Subject{Kind: finding.SubjectInput, Name: "manifest", Title: "Manifest"},
		At:    finding.Location{Origin: finding.OriginDataDir, File: "metais.json"},
	}, limits.Defaults())
	if len(fs) > 0 {
		t.Fatalf("parsing %s: %v", b, fs[0].Message())
	}
	return doc.Value, doc.Positions()
}

// tree is a manifest written as Go values, which encoding/json writes as
// the JSON text a test parses; json.RawMessage writes a literal as it is.
type tree = map[string]any

// base returns a valid manifest that uses every member and every form, so
// that each mutation of it breaks exactly the rule it names.
func base() tree {
	return tree{
		"metais":      "1.0",
		"name":        "org.example.vpat-acr",
		"version":     "2.1.0-rc.1+20261018",
		"title":       "Accessibility Conformance Report",
		"description": "An ACR in Markdown and DOCX.\n\tIndented.",
		"guide":       "# Guide\n\nGather the product first.",
		"authors": []any{
			tree{"name": "Example Org", "email": "acr@example.org", "url": "https://example.org/acr"},
			tree{"name": "A. Person"},
		},
		"license":           "(Apache-2.0 OR MIT) AND GPL-2.0-or-later WITH Classpath-exception-2.0",
		"homepage":          "https://example.org:8443/packages/acr?tab=readme",
		"input_constraints": "inputs/constraints.schema.yaml",
		"inputs": []any{
			tree{
				"name": "product", "title": "Product", "description": "The product the report covers.",
				"schema": "inputs/product.schema.json", "meaning": "inputs/product.md",
			},
			tree{
				"name": "criteria", "title": "WCAG criteria", "schema": "inputs/criteria.schema.yaml",
				"forms": []any{"csv", "json"}, "csv": tree{"delimiter": "\t"},
				"collection": false, "required": false,
			},
		},
		"outputs": []any{
			tree{"id": "report", "template": "report.md.liquid"},
			tree{
				"id": "acr_docx", "path": "{{ inputs.product.slug }}-acr.docx", "when": "inputs.product",
				"container": tree{
					"format": "zip", "profile": "ooxml",
					"entries": []any{
						tree{"mirror": "docx", "path": ".", "escape": "html", "newline": "crlf", "first": true},
						tree{"id": "logo", "asset": "vendor-logo.png", "path": "word/media/image1.png", "stored": true},
						tree{"template": "part.json.liquid", "kind": "json", "validate": "references/part.schema.json"},
						tree{"container": tree{"profile": "odf", "mimetype": "application/vnd.oasis.opendocument.text",
							"entries": []any{tree{"asset": "content.xml"}}}, "path": "inner.odt"},
					},
				},
			},
			tree{
				"id": "article", "template": "site/_article.html.liquid",
				"for_each": `inputs.articles | where: "draft", false`, "path": "articles/{{ item.slug }}.html",
				"escape": "html", "kind": "html", "executable": false,
			},
			tree{"id": "site", "mirror": "site", "path": ".", "escape": "html"},
			tree{"id": "static", "asset": "static", "executable": true},
			tree{"id": "books", "for_each": "inputs.books", "path": "{{ item.slug }}.epub",
				"container": tree{"profile": "epub", "entries": []any{tree{"template": "book.xhtml.liquid"}}}},
		},
	}
}

// encode writes m as JSON text.
func encode(t testing.TB, m any) []byte {
	t.Helper()
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(m); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

// at returns the value at the slash-separated path in m, where a number
// indexes an array.
func at(t testing.TB, m any, path string) any {
	t.Helper()
	cur := m
	if path == "" {
		return cur
	}
	for _, tok := range strings.Split(path, "/") {
		switch c := cur.(type) {
		case tree:
			cur = c[tok]
		case []any:
			var i int
			if _, err := fmt.Sscan(tok, &i); err != nil || i >= len(c) {
				t.Fatalf("no element %s in %s", tok, path)
			}
			cur = c[i]
		default:
			t.Fatalf("cannot descend into %T at %s", cur, path)
		}
	}
	return cur
}

// set sets member key of the object at path in m to v, or removes it when
// v is del.
func set(t testing.TB, m tree, path, key string, v any) {
	t.Helper()
	o, ok := at(t, m, path).(tree)
	if !ok {
		t.Fatalf("%s is not an object", path)
	}
	if v == del {
		delete(o, key)
		return
	}
	o[key] = v
}

// del removes a member in set.
var del = &struct{ name string }{"del"}

// isolate gives a test a fresh HOME (internal/testenv).
func isolate(t *testing.T) { testenv.Home(t) }

// codes returns the codes of fs.
func codes(fs []finding.Finding) []finding.Code {
	var out []finding.Code
	for _, f := range fs {
		out = append(out, f.Code())
	}
	return out
}

// pointer returns the pointer of f's location.
func pointer(t testing.TB, f finding.Finding) string {
	t.Helper()
	l, ok := f.Location()
	if !ok || !l.HasPointer {
		t.Fatalf("%s has no pointer", f.Code())
	}
	return l.Pointer
}
