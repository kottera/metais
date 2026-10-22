// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package schema

import (
	"errors"
	"strings"
	"testing"

	"metais.org/metais/internal/liquid/value"
)

// The schema files of ADR-003 D7 item 1's example.
const (
	vendorSchema      = `{"type": "object", "properties": {"name": {"type": "string"}}, "required": ["name"]}`
	constraintsSchema = `{"allOf": [{"if": {"properties": {"stage": {"const": "final"}}, "required": ["stage"]}, "then": {"required": ["incident"]}}]}`
)

// example is ADR-003 D7 item 1's package: one input, vendor, and
// input_constraints.
func example(t *testing.T) (Declarations, []File) {
	t.Helper()
	return Declarations{
			Inputs: []Input{{
				Name:        "vendor",
				Title:       "Vendor",
				Description: ptr("The organisation that makes the product."),
				Schema:      "inputs/vendor.schema.json",
				Required:    true,
			}},
			Constraints: "inputs/constraints.schema.json",
		}, []File{
			{Path: "inputs/vendor.schema.json", Root: parse(t, vendorSchema)},
			{Path: "inputs/constraints.schema.json", Root: parse(t, constraintsSchema)},
		}
}

func mustBundle(t *testing.T, decl Declarations, files []File) value.Object {
	t.Helper()
	b, err := Bundle(decl, files)
	if err != nil {
		t.Fatalf("Bundle: %v", err)
	}
	return b
}

// get returns the value at the JSON Pointer p in v, failing t when there
// is none.
func get(t *testing.T, v value.Value, p string) value.Value {
	t.Helper()
	got, ok := lookup(v, p)
	if !ok {
		t.Fatalf("no value at %q in %s", p, canonical(v))
	}
	return got
}

// TestBundleCanonicalForm is the meeting with internal/jsonx: the
// canonical JSON of the combined schema is the document ADR-003 D7 item 1
// shows, with the example's elided resources written out, in RFC 8785's
// member order.
func TestBundleCanonicalForm(t *testing.T) {
	decl, files := example(t)
	want := `{` +
		`"$defs":{` +
		`"inputs/constraints.schema.json":{` +
		`"$id":"metais:///inputs/constraints.schema.json",` +
		`"$schema":"https://json-schema.org/draft/2020-12/schema",` +
		`"allOf":[{"if":{"properties":{"stage":{"const":"final"}},"required":["stage"]},"then":{"required":["incident"]}}]},` +
		`"inputs/vendor.schema.json":{` +
		`"$id":"metais:///inputs/vendor.schema.json",` +
		`"$schema":"https://json-schema.org/draft/2020-12/schema",` +
		`"properties":{"name":{"type":"string"}},"required":["name"],"type":"object"}},` +
		`"$id":"metais:///.compiled/schema.json",` +
		`"$schema":"https://json-schema.org/draft/2020-12/schema",` +
		`"additionalProperties":false,` +
		`"allOf":[{"$ref":"metais:///inputs/constraints.schema.json"}],` +
		`"properties":{"vendor":{` +
		`"$ref":"metais:///inputs/vendor.schema.json",` +
		`"description":"The organisation that makes the product.",` +
		`"title":"Vendor"}},` +
		`"required":["vendor"],` +
		`"type":"object"}`
	if got := canonical(mustBundle(t, decl, files)); got != want {
		t.Errorf("canonical combined schema:\n got %s\nwant %s", got, want)
	}
}

// TestBundleProperties holds D7 item 2's properties: one member per input,
// each the slot's title and description beside a $ref to the canonical
// URI of the input's schema, and nothing else. A slot with no description
// has none.
func TestBundleProperties(t *testing.T) {
	decl := Declarations{Inputs: []Input{
		{Name: "vendor", Title: "Vendor", Description: ptr("Who makes it.\nAnd where."), Schema: "inputs/vendor.schema.json", Required: true},
		{Name: "summary", Title: "Summary", Schema: "inputs/summary.schema.json"},
		// Two slots may share one schema (ADR-003 Proposed ruling 3).
		{Name: "buyer", Title: "Buyer", Description: ptr(""), Schema: "inputs/vendor.schema.json"},
	}}
	files := []File{
		{Path: "inputs/vendor.schema.json", Root: parse(t, vendorSchema)},
		{Path: "inputs/summary.schema.json", Root: parse(t, `{"$id": "https://example.org/summary", "type": "string"}`)},
	}
	b := mustBundle(t, decl, files)
	for p, want := range map[string]string{
		"/properties/vendor":  `{"$ref":"metais:///inputs/vendor.schema.json","description":"Who makes it.\nAnd where.","title":"Vendor"}`,
		"/properties/summary": `{"$ref":"https://example.org/summary","title":"Summary"}`,
		"/properties/buyer":   `{"$ref":"metais:///inputs/vendor.schema.json","description":"","title":"Buyer"}`,
	} {
		if got := canonical(get(t, b, p)); got != want {
			t.Errorf("%s = %s, want %s", p, got, want)
		}
	}
	if n := get(t, b, "/properties").(value.Object).Len(); n != 3 {
		t.Errorf("properties has %d members, want 3", n)
	}
	for p, want := range map[string]string{
		"/type":                 `"object"`,
		"/additionalProperties": `false`,
		"/$id":                  `"metais:///.compiled/schema.json"`,
		"/$schema":              `"https://json-schema.org/draft/2020-12/schema"`,
	} {
		if got := canonical(get(t, b, p)); got != want {
			t.Errorf("%s = %s, want %s", p, got, want)
		}
	}
}

// TestBundleRequiredInManifestOrder holds D7 item 2's required: the inputs
// with required true, in manifest order, which is neither name order nor
// the canonical order properties is held in.
func TestBundleRequiredInManifestOrder(t *testing.T) {
	in := func(name string, req bool) Input {
		return Input{Name: name, Title: name, Schema: "inputs/s.json", Required: req}
	}
	files := []File{{Path: "inputs/s.json", Root: value.Bool(true)}}
	for _, c := range []struct {
		name   string
		inputs []Input
		want   string
	}{
		{"manifest order", []Input{in("zeta", true), in("alpha", true), in("mid", false), in("beta", true)}, `["zeta","alpha","beta"]`},
		{"reversed", []Input{in("beta", true), in("mid", false), in("alpha", true), in("zeta", true)}, `["beta","alpha","zeta"]`},
		{"none required", []Input{in("alpha", false)}, `[]`},
		{"no inputs", nil, `[]`},
	} {
		b := mustBundle(t, Declarations{Inputs: c.inputs}, files)
		if got := canonical(get(t, b, "/required")); got != c.want {
			t.Errorf("%s: required = %s, want %s", c.name, got, c.want)
		}
	}
}

// TestBundleAllOfOnlyWithConstraints holds D7 item 2's allOf: present, a
// $ref to input_constraints' canonical URI, exactly when the manifest
// declares input_constraints.
func TestBundleAllOfOnlyWithConstraints(t *testing.T) {
	decl, files := example(t)
	if got := canonical(get(t, mustBundle(t, decl, files), "/allOf")); got != `[{"$ref":"metais:///inputs/constraints.schema.json"}]` {
		t.Errorf("allOf = %s", got)
	}
	decl.Constraints = ""
	b := mustBundle(t, decl, files)
	if v, ok := b.Get("allOf"); ok {
		t.Errorf("allOf = %s with no input_constraints, want none", canonical(v))
	}
	// The constraints file is still a schema file under inputs/.
	get(t, b, "/$defs/inputs~1constraints.schema.json")

	files[1].Root = parse(t, `{"$id": "urn:example:constraints", "allOf": []}`)
	decl.Constraints = "inputs/constraints.schema.json"
	if got := canonical(get(t, mustBundle(t, decl, files), "/allOf")); got != `[{"$ref":"urn:example:constraints"}]` {
		t.Errorf("allOf by $id = %s", got)
	}
}

// TestBundleDefs holds D7 item 3: every schema file under inputs/ is in
// $defs, keyed by its package path, whether or not an input names it, with
// $id its canonical URI (D5 item 2) and $schema set. A YAML schema file is
// embedded as the JSON value its reader gives.
func TestBundleDefs(t *testing.T) {
	decl := Declarations{Inputs: []Input{
		{Name: "vendor", Title: "Vendor", Schema: "inputs/vendor.schema.json", Required: true},
		{Name: "summary", Title: "Summary", Schema: "inputs/sub dir/summary.schema.yaml", Required: true},
		{Name: "report", Title: "Report", Schema: "inputs/report.schema.yml", Required: true},
	}}
	files := []File{
		{Path: "inputs/vendor.schema.json", Root: parse(t, vendorSchema)},
		// What D10's reader gives for "type: string\nmaxLength: 200\n".
		{Path: "inputs/sub dir/summary.schema.yaml", Root: parse(t, `{"type": "string", "maxLength": 200}`)},
		{Path: "inputs/report.schema.yml", Root: parse(t, `{"$id": "report/v2", "$schema": "https://json-schema.org/draft/2020-12/schema", "type": "object"}`)},
		{Path: "inputs/common.schema.json", Root: parse(t, `{"$id": "https://example.org/common.json", "$defs": {"id": {"type": "string"}}}`)},
		{Path: "inputs/never-named.json", Root: parse(t, `{"type": "number"}`)},
	}
	b := mustBundle(t, decl, files)
	defs := get(t, b, "/$defs").(value.Object)
	want := map[string]struct{ id, rest string }{
		"inputs/vendor.schema.json":          {"metais:///inputs/vendor.schema.json", `{"properties":{"name":{"type":"string"}},"required":["name"],"type":"object"}`},
		"inputs/sub dir/summary.schema.yaml": {"metais:///inputs/sub%20dir/summary.schema.yaml", `{"maxLength":200,"type":"string"}`},
		"inputs/report.schema.yml":           {"metais:///inputs/report/v2", `{"type":"object"}`},
		"inputs/common.schema.json":          {"https://example.org/common.json", `{"$defs":{"id":{"type":"string"}}}`},
		"inputs/never-named.json":            {"metais:///inputs/never-named.json", `{"type":"number"}`},
	}
	if defs.Len() != len(want) {
		t.Errorf("$defs has %v, want the %d files", defs.Names(), len(want))
	}
	for path, w := range want {
		res, ok := defs.Get(path)
		if !ok {
			t.Errorf("$defs has no %q", path)
			continue
		}
		o := res.(value.Object)
		if got := canonical(get(t, o, "/$id")); got != `"`+w.id+`"` {
			t.Errorf("%s: $id = %s, want %q", path, got, w.id)
		}
		if got := canonical(get(t, o, "/$schema")); got != `"`+Dialect+`"` {
			t.Errorf("%s: $schema = %s", path, got)
		}
		if got := canonical(without(t, o, "$id", "$schema")); got != w.rest {
			t.Errorf("%s: other members = %s, want %s", path, got, w.rest)
		}
	}
	if got := canonical(get(t, b, "/properties/report/$ref")); got != `"metais:///inputs/report/v2"` {
		t.Errorf("report's $ref = %s, want its $id resolved", got)
	}
}

// TestBundleBooleanSchemaFiles: a boolean schema file has no members to
// hold an $id, so it is embedded as the object schema that means the same.
func TestBundleBooleanSchemaFiles(t *testing.T) {
	files := []File{
		{Path: "inputs/any.json", Root: value.Bool(true)},
		{Path: "inputs/none.json", Root: value.Bool(false)},
	}
	defs := get(t, mustBundle(t, Declarations{}, files), "/$defs")
	for p, want := range map[string]string{
		"/inputs~1any.json":  `{"$id":"metais:///inputs/any.json","$schema":"https://json-schema.org/draft/2020-12/schema"}`,
		"/inputs~1none.json": `{"$id":"metais:///inputs/none.json","$schema":"https://json-schema.org/draft/2020-12/schema","not":{}}`,
	} {
		if got := canonical(get(t, defs, p)); got != want {
			t.Errorf("%s = %s, want %s", p, got, want)
		}
	}
}

// TestBundleSubstitutesNoRef pins D7 item 4 false first: every member of a
// schema file but $id and $schema is embedded unchanged, so no $ref,
// $dynamicRef or relative reference is replaced by its target or
// rewritten. A recursive schema, which substitution cannot express,
// bundles.
func TestBundleSubstitutesNoRef(t *testing.T) {
	tree := `{
		"$id": "tree.json",
		"$dynamicAnchor": "node",
		"type": "object",
		"properties": {
			"children": {"type": "array", "items": {"$dynamicRef": "#node"}},
			"parent": {"$ref": "#"},
			"label": {"$ref": "common.json#/$defs/label"},
			"meta": {"$ref": "#/$defs/meta"},
			"kind": {"$ref": "#kind"}
		},
		"$defs": {
			"meta": {"$ref": "https://example.org/meta.json"},
			"kind": {"$anchor": "kind", "enum": ["a", "b"]}
		}
	}`
	common := `{"$id": "common.json", "$defs": {"label": {"type": "string", "$ref": "#/$defs/text"}, "text": {"minLength": 1}}}`
	files := []File{
		{Path: "inputs/tree.json", Root: parse(t, tree)},
		{Path: "inputs/common.json", Root: parse(t, common)},
		{Path: "inputs/alias.json", Root: parse(t, `{"$ref": "tree.json", "description": "Another name for a tree."}`)},
	}
	decl := Declarations{Inputs: []Input{{Name: "tree", Title: "Tree", Schema: "inputs/tree.json", Required: true}}}
	b := mustBundle(t, decl, files)
	for _, f := range files {
		res := get(t, b, "/$defs/"+escapeToken(f.Path)).(value.Object)
		if !value.Equal(without(t, res, "$id", "$schema"), without(t, f.Root.(value.Object), "$id", "$schema")) {
			t.Errorf("%s changed in bundling:\n got %s\nfrom %s", f.Path, canonical(res), canonical(f.Root))
		}
	}
	// Each relative $id resolved against its file's retrieval URI, so each
	// relative reference resolves as it did in its own file.
	if got := canonical(get(t, b, "/$defs/inputs~1tree.json/$id")); got != `"metais:///inputs/tree.json"` {
		t.Errorf("tree's $id = %s", got)
	}
	if got := canonical(get(t, b, "/properties/tree/$ref")); got != `"metais:///inputs/tree.json"` {
		t.Errorf("tree's property $ref = %s", got)
	}
}

// without returns o less the members names.
func without(t *testing.T, o value.Object, names ...string) value.Object {
	t.Helper()
	var ms []value.Member
	for n, v := range o.All() {
		if !strings.Contains("\x00"+strings.Join(names, "\x00")+"\x00", "\x00"+n+"\x00") {
			ms = append(ms, value.Member{Name: n, Value: v})
		}
	}
	r, err := value.NewObject(ms...)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func TestRetrievalURI(t *testing.T) {
	for path, want := range map[string]string{
		"inputs/vendor.schema.json": "metais:///inputs/vendor.schema.json",
		"inputs/a b.json":           "metais:///inputs/a%20b.json",
		"inputs/é.json":             "metais:///inputs/%C3%A9.json",
		"inputs/a#b?.json":          "metais:///inputs/a%23b%3F.json",
		"inputs/a%b.json":           "metais:///inputs/a%25b.json",
	} {
		if got := RetrievalURI(path); got != want {
			t.Errorf("RetrievalURI(%q) = %q, want %q", path, got, want)
		}
	}
}

// TestBundleRefusals: every rule another layer reports as a finding before
// it bundles is refused here too, rather than giving a combined schema
// that names something it does not hold.
func TestBundleRefusals(t *testing.T) {
	vendor := Input{Name: "vendor", Title: "Vendor", Schema: "inputs/vendor.schema.json", Required: true}
	ok := File{Path: "inputs/vendor.schema.json", Root: value.Bool(true)}
	for _, c := range []struct {
		name  string
		decl  Declarations
		files []File
		want  string
	}{
		{"file outside inputs/", Declarations{}, []File{{Path: "schemas/x.json", Root: value.Bool(true)}}, "not under inputs/"},
		{"file given twice", Declarations{}, []File{ok, ok}, "given twice"},
		{"input names no file", Declarations{Inputs: []Input{vendor}}, nil, "names schema inputs/vendor.schema.json"},
		{"input names a path outside inputs/", Declarations{Inputs: []Input{{Name: "v", Title: "V", Schema: "x.json"}}}, []File{ok}, "names schema x.json"},
		{"constraints name no file", Declarations{Constraints: "inputs/c.json"}, []File{ok}, "input_constraints names inputs/c.json"},
		{"one canonical URI, by $id and retrieval URI", Declarations{}, []File{ok, {Path: "inputs/b.json", Root: parse(t, `{"$id": "vendor.schema.json"}`)}}, "one canonical URI"},
		{"one canonical URI, two $ids", Declarations{}, []File{
			{Path: "inputs/a.json", Root: parse(t, `{"$id": "https://example.org/s"}`)},
			{Path: "inputs/b.json", Root: parse(t, `{"$id": "https://example.org/s#"}`)},
		}, "one canonical URI"},
		{"$id with a fragment", Declarations{}, []File{{Path: "inputs/a.json", Root: parse(t, `{"$id": "a.json#frag"}`)}}, "non-empty fragment"},
		{"$id not a string", Declarations{}, []File{{Path: "inputs/a.json", Root: parse(t, `{"$id": 1}`)}}, "not a string"},
		{"$id not a URI reference", Declarations{}, []File{{Path: "inputs/a.json", Root: parse(t, `{"$id": "%zz"}`)}}, "not a URI reference"},
		{"another $schema", Declarations{}, []File{{Path: "inputs/a.json", Root: parse(t, `{"$schema": "http://json-schema.org/draft-07/schema#"}`)}}, "$schema is not"},
		{"$schema not a string", Declarations{}, []File{{Path: "inputs/a.json", Root: parse(t, `{"$schema": true}`)}}, "$schema is not"},
		{"root not a schema", Declarations{}, []File{{Path: "inputs/a.json", Root: parse(t, `"string"`)}}, "not a string"},
		{"root missing", Declarations{}, []File{{Path: "inputs/a.json"}}, "not a missing value"},
		{"two inputs, one name", Declarations{Inputs: []Input{vendor, vendor}}, []File{ok}, "duplicate member name"},
		{"title not scalar values", Declarations{Inputs: []Input{{Name: "v", Title: "\xff", Schema: ok.Path}}}, []File{ok}, "title"},
		{"description not scalar values", Declarations{Inputs: []Input{{Name: "v", Title: "V", Description: ptr("\xff"), Schema: ok.Path}}}, []File{ok}, "description"},
	} {
		_, err := Bundle(c.decl, c.files)
		if !errors.Is(err, ErrBundle) || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: err = %v, want ErrBundle containing %q", c.name, err, c.want)
		}
	}
}
