// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package schema

import (
	"fmt"
	"net/url"
	"strings"
	"testing"

	"github.com/kottera/metais/internal/finding"
	"github.com/kottera/metais/internal/limits"
	"github.com/kottera/metais/internal/liquid/value"
)

// intParam is f's integer parameter name.
func intParam(t *testing.T, f finding.Finding, name string) int64 {
	t.Helper()
	v, ok := f.Param(name)
	if !ok {
		t.Fatalf("%s has no %s", f.Code(), name)
	}
	n, ok := v.Int()
	if !ok {
		t.Fatalf("%s's %s is not an integer", f.Code(), name)
	}
	return n
}

// wantBreach checks that fs is one limit.package finding of limit at
// entry#ptr, observing observed.
func wantBreach(t *testing.T, fs []finding.Finding, limit, entry, ptr string, observed int64) {
	t.Helper()
	if len(fs) != 1 {
		t.Fatalf("%d findings, want one %s:\n%s", len(fs), limit, describe(fs))
	}
	f := fs[0]
	l, _ := f.Location()
	if f.Code() != finding.CodeLimitPackage || l.File != entry || !l.HasPointer || l.Pointer != ptr {
		t.Fatalf("got %s at %s#%s, want limit.package at %s#%s", f.Code(), l.File, l.Pointer, entry, ptr)
	}
	if got := param(t, f, "limit"); got != limit {
		t.Errorf("limit is %q, want %q", got, limit)
	}
	if got := intParam(t, f, "observed"); got != observed {
		t.Errorf("observed is %d, want %d", got, observed)
	}
}

// boundsFiles are two schema files of five subschemas together, in the
// order the pre-check walks them: inputs/a.json's root, /properties/x (a
// boolean), /properties/y, then inputs/b.json's root and /items (a
// boolean). enum and const hold data, not schemas, and count for nothing.
var boundsFiles = map[string]string{
	"inputs/a.json": `{"properties": {"x": true, "y": {"enum": [{}, {}], "const": {}}}}`,
	"inputs/b.json": `{"items": false}`,
}

// TestSchemaSubschemas is ADR-009 Amendment A1 item 1: the subschemas of
// every schema file together, booleans included, are bounded by
// schema-subschemas, and the first one past the bound is refused with
// limit.package at its file and pointer, observing the package's total.
func TestSchemaSubschemas(t *testing.T) {
	if _, fs := compileFiles(t, lower(t, "schema-subschemas=5"), boundsFiles); len(fs) > 0 {
		t.Fatalf("at the bound:\n%s", describe(fs))
	}
	for _, c := range []struct {
		bound      string
		entry, ptr string
	}{
		{"4", "inputs/b.json", "/items"},
		{"3", "inputs/b.json", ""},
		{"2", "inputs/a.json", "/properties/y"},
		{"1", "inputs/a.json", "/properties/x"},
		{"0", "inputs/a.json", ""},
	} {
		t.Run(c.bound, func(t *testing.T) {
			comp, fs := compileFiles(t, lower(t, "schema-subschemas="+c.bound), boundsFiles)
			if comp != nil {
				t.Error("NewCompiler returned a compiler beside its findings")
			}
			wantBreach(t, fs, "schema-subschemas", c.entry, c.ptr, 5)
		})
	}
}

// TestSchemaSubschemasBesideOtherFindings holds the count to the pre-check's
// rule that every finding is reported at once: a file past the bound is
// still checked, and its other refusals reported beside the count's.
func TestSchemaSubschemasBesideOtherFindings(t *testing.T) {
	_, fs := compileFiles(t, lower(t, "schema-subschemas=1"), map[string]string{
		"inputs/s.json": `{"items": {"x-a": 1, "format": "nonsense"}}`,
	})
	if got, want := strings.Join(codes(fs), " "), "limit.package schema.format_unsupported"; got != want {
		t.Errorf("codes %q, want %q:\n%s", got, want, describe(fs))
	}
}

// TestSchemaLocationLength is ADR-009 Amendment A1 item 2 at its boundary:
// a subschema's location is the longer of the canonical URI of its nearest
// enclosing $id and its file's retrieval URI, then "#", then its pointer
// as RFC 6901 escapes it. At its own length it passes; one byte lower, the
// subschema is refused with limit.package at its file and pointer,
// observing that length.
func TestSchemaLocationLength(t *testing.T) {
	const retrieval = "metais:///inputs/s.json"
	long := "https://example.com/schemas/a-longer-identifier-than-the-retrieval-uri.json"
	nested := "https://example.com/schemas/nested/an-identifier-longer-than-its-parents-is-used.json"
	cases := []struct {
		name, schema, ptr string
		location          string // the location of the longest subschema at ptr
	}{
		{"the retrieval URI, at the root", `{}`, "", retrieval + "#"},
		{"the retrieval URI", `{"properties": {"abc": {}}}`, "/properties/abc", retrieval + "#/properties/abc"},
		{"a boolean", `{"properties": {"abcdef": true}}`, "/properties/abcdef", retrieval + "#/properties/abcdef"},
		{"RFC 6901's escapes", `{"properties": {"a/b~c": {}}}`, "/properties/a~1b~0c", retrieval + "#/properties/a~1b~0c"},
		{"a longer root $id", `{"$id": "` + long + `", "items": {}}`, "/items", long + "#/items"},
		{"a shorter root $id", `{"$id": "urn:x", "items": {}}`, "/items", retrieval + "#/items"},
		{"a relative $id", `{"$id": "sub/deeper/than/the/file.json", "items": {}}`, "/items",
			"metais:///inputs/sub/deeper/than/the/file.json#/items"},
		{"a nested $id, by its file's pointer", `{"$defs": {"d": {"$id": "` + nested + `", "items": {}}}}`,
			"/$defs/d/items", nested + "#/$defs/d/items"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			n := len(c.location)
			files := map[string]string{"inputs/s.json": c.schema}
			if _, fs := compileFiles(t, lower(t, fmt.Sprintf("schema-location-length=%d", n)), files); len(fs) > 0 {
				t.Fatalf("at %d bytes:\n%s", n, describe(fs))
			}
			_, fs := compileFiles(t, lower(t, fmt.Sprintf("schema-location-length=%d", n-1)), files)
			wantBreach(t, fs, "schema-location-length", "inputs/s.json", c.ptr, int64(n))
		})
	}
}

// TestSchemaLocationLengthPerFile holds schema-location-length to one
// finding per file, at its first location over the bound in the walk's
// order, and to a check of every file.
func TestSchemaLocationLengthPerFile(t *testing.T) {
	_, fs := compileFiles(t, lower(t, "schema-location-length=30"), map[string]string{
		"inputs/a.json": `{"properties": {"long-1": {}, "long-2": {"items": {}}}}`,
		"inputs/b.json": `{"items": {"items": {}}}`,
	})
	var got []string
	for _, f := range fs {
		l, _ := f.Location()
		got = append(got, fmt.Sprintf("%s %s#%s", param(t, f, "limit"), l.File, l.Pointer))
	}
	want := []string{
		"schema-location-length inputs/a.json#/properties/long-1",
		"schema-location-length inputs/b.json#/items/items",
	}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Errorf("got\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
}

// TestCombinedSubschemas is Amendment A1 item 1's combined schema: counted
// as a whole, apart from the files, and refused with limit.package at
// CombinedEntry and its first subschema past the bound, as
// combined-schema-size is, while its longer locations are not checked.
func TestCombinedSubschemas(t *testing.T) {
	files := map[string]string{"inputs/s.json": `{"items": true}`}
	bundle := func(c *Compiler) value.Value {
		doc, err := Bundle(Declarations{Inputs: []Input{{Name: "s", Title: "S", Schema: "inputs/s.json"}}},
			[]File{{Path: "inputs/s.json", Root: c.docs["inputs/s.json"]}})
		if err != nil {
			t.Fatal(err)
		}
		return doc
	}
	c, fs := compileFiles(t, limits.Defaults(), files)
	if len(fs) > 0 {
		t.Fatal(describe(fs))
	}
	doc := bundle(c)
	total := (&prechecker{lim: limits.Defaults(), count: &subschemas{}})
	base, _ := url.Parse(CombinedURI)
	total.walk(doc, "", base, 1)
	n := total.count.n
	if n <= 2 {
		t.Fatalf("the combined schema has %d subschemas, which shows nothing beside the file's 2", n)
	}

	// Its locations are each longer than the file's own, and pass.
	location := fmt.Sprintf("schema-location-length=%d", len("metais:///inputs/s.json#/items"))
	at := func(bound string) []finding.Finding {
		lim := lower(t, "schema-subschemas="+bound)
		if f, ok := lim.Lower(location); !ok {
			t.Fatal(f.Message())
		}
		c, fs := compileFiles(t, lim, files)
		if len(fs) > 0 {
			t.Fatalf("the file alone, at %s:\n%s", bound, describe(fs))
		}
		_, fs = c.Combined(bundle(c))
		return fs
	}
	if fs := at(fmt.Sprint(n)); len(fs) > 0 {
		t.Fatalf("at the bound:\n%s", describe(fs))
	}
	fs = at(fmt.Sprint(n - 1))
	if len(fs) != 1 {
		t.Fatalf("%d findings past the bound:\n%s", len(fs), describe(fs))
	}
	l, _ := fs[0].Location()
	if l.File != CombinedEntry || !l.HasPointer {
		t.Errorf("refused at %s, pointer %v, want %s and a pointer", l.File, l.HasPointer, CombinedEntry)
	}
	wantBreach(t, fs, "schema-subschemas", CombinedEntry, l.Pointer, n)
}

// fhirShaped is a schema of FHIR R4's shape (ADR-009 Amendment A1 item 4):
// one file of definitions, each an object of properties, most of them a
// $ref to another definition and some an array of them, about 15,500
// subschemas in all, under a root $id, and its longest pointer about 105
// bytes long.
func fhirShaped() string {
	const defs, props = 199, 51
	name := func(i int) string {
		// "ExplanationOfBenefit_Item_Detail_SubDetail"'s length, and more.
		return fmt.Sprintf("Resource%03d_Component_Detail_SubDetail_Element_Backbone_Part", i)
	}
	var b strings.Builder
	b.WriteString(`{"$id": "http://hl7.org/fhir/json-schema/4.0", "$schema": "https://json-schema.org/draft/2020-12/schema", "$defs": {`)
	for i := range defs {
		if i > 0 {
			b.WriteString(",")
		}
		fmt.Fprintf(&b, `%q: {"description": "A FHIR-shaped definition.", "type": "object", "properties": {`, name(i))
		for j := range props {
			if j > 0 {
				b.WriteString(",")
			}
			ref := fmt.Sprintf(`{"$ref": "#/$defs/%s"}`, name((i+j+1)%defs))
			if j%2 == 0 {
				fmt.Fprintf(&b, `"modifierExtension%02d": {"description": "Extensions.", "type": "array", "items": %s}`, j, ref)
			} else {
				fmt.Fprintf(&b, `"element%02d": %s`, j, ref)
			}
		}
		b.WriteString(`}, "additionalProperties": false}`)
	}
	b.WriteString(`}}`)
	return b.String()
}

// glossary is an ordinary field glossary: a few thousand fields of ordinary
// names, each a typed, described property.
func glossary() string {
	var b strings.Builder
	b.WriteString(`{"type": "object", "properties": {`)
	for i := range 3_000 {
		if i > 0 {
			b.WriteString(",")
		}
		fmt.Fprintf(&b, `"applicant_postal_address_line_%04d": {"title": "Address line", "description": "One line of the applicant's postal address.", "type": "string", "maxLength": 200}`, i)
	}
	b.WriteString(`}}`)
	return b.String()
}

// TestSchemaBoundsAdmitRealSchemas is Amendment A1 item 4's claim: neither
// limit refuses a schema of FHIR R4's shape, nor a field glossary of
// ordinary names, at the defaults. The FHIR-shaped schema is checked to
// have FHIR's measures, so the proof is of the shape it names.
func TestSchemaBoundsAdmitRealSchemas(t *testing.T) {
	fhir := fhirShaped()
	for _, c := range []struct {
		name, src            string
		minCount, maxCount   int64
		minPointer, maxPoint int
	}{
		{"FHIR R4's shape", fhir, 15_000, 16_000, 100, 110},
		{"a field glossary", glossary(), 3_000, 3_002, 0, 512},
	} {
		t.Run(c.name, func(t *testing.T) {
			w := &prechecker{lim: limits.Defaults(), count: &subschemas{}}
			doc := mustValue(t, c.src)
			longest := 0
			var measure func(v value.Value, ptr string)
			measure = func(v value.Value, ptr string) {
				longest = max(longest, len(ptr))
				o, ok := v.(value.Object)
				if !ok {
					return
				}
				for k, m := range o.All() {
					switch keywords[k] {
					case shapeSchema:
						measure(m, ptr+"/"+escapeToken(k))
					case shapeSchemaMap:
						for kk, mm := range m.(value.Object).All() {
							measure(mm, ptr+"/"+escapeToken(k)+"/"+escapeToken(kk))
						}
					}
				}
			}
			measure(doc, "")
			base, _ := url.Parse(RetrievalURI("inputs/s.json"))
			w.walk(doc, "", base, 1)
			if n := w.count.n; n < c.minCount || n > c.maxCount {
				t.Fatalf("%d subschemas, want %d to %d", n, c.minCount, c.maxCount)
			}
			if longest < c.minPointer || longest > c.maxPoint {
				t.Fatalf("longest pointer %d bytes, want %d to %d", longest, c.minPointer, c.maxPoint)
			}
			if _, fs := compileFiles(t, limits.Defaults(), map[string]string{"inputs/s.json": c.src}); len(fs) > 0 {
				t.Fatalf("refused at the defaults:\n%s", describe(fs))
			}
		})
	}
}
