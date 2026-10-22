// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package schema

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"metais.org/metais/internal/finding"
	"metais.org/metais/internal/limits"
	"metais.org/metais/internal/liquid/value"
	"metais.org/metais/internal/schemafail"
)

// combinedFiles is a package's schema files: one with an $id, one reached
// by its retrieval URI, a shared definition and the cross-input
// constraints.
var combinedFiles = map[string]string{
	"inputs/vendor.schema.json": `{"$id": "https://example.com/vendor", "type": "object",
		"properties": {"name": {"type": "string", "minLength": 2}, "address": {"$ref": "metais:///inputs/common/address.json"}},
		"required": ["name"]}`,
	"inputs/common/address.json": `{"type": "object", "properties": {"city": {"type": "string"}},
		"required": ["city"], "additionalProperties": false}`,
	"inputs/count.schema.json":       `{"type": "integer", "minimum": 1}`,
	"inputs/constraints.schema.json": `{"if": {"properties": {"count": {"const": 1}}}, "then": {"required": ["vendor"]}}`,
}

// failureKeys describes each failure of fs, leaves only, by where its
// keyword is and where its value is: what a finding is made of, whatever
// path led there.
func failureKeys(fs []schemafail.Failure, prefix []string) []string {
	var out []string
	for i := range fs {
		fs[i].Walk(func(_ []*schemafail.Failure, f *schemafail.Failure) bool {
			if len(f.Children) == 0 {
				inst := strings.Join(append(append([]string{}, prefix...), f.InstancePointer...), "/")
				out = append(out, fmt.Sprintf("%s#%s %s @/%s %v", f.Entry, f.SchemaPointer, f.Keyword, inst, f.Data))
			}
			return true
		})
	}
	slices.Sort(out)
	return out
}

// TestCombinedSchema is the meeting of this item with W2-12's bundling
// (ADR-003 D7). The combined schema Bundle derives compiles under
// NewCompiler's closed configuration; its $defs keys are the schema files
// the compiler holds and its $ids the canonical URIs the compiler
// registered for them; and it validates an inputs object exactly as the
// files do, one by one: the same verdict, and failures of the same
// keywords at the same places, which schemafail.Map turns into the same
// findings.
func TestCombinedSchema(t *testing.T) {
	c, fs := compileFiles(t, limits.Defaults(), combinedFiles)
	if len(fs) > 0 {
		t.Fatal(describe(fs))
	}
	var files []File
	for entry := range combinedFiles {
		files = append(files, File{Path: entry, Root: c.docs[entry]})
	}
	doc, err := Bundle(Declarations{
		Inputs: []Input{
			{Name: "vendor", Title: "Vendor", Schema: "inputs/vendor.schema.json", Required: false},
			{Name: "count", Title: "Count", Schema: "inputs/count.schema.json", Required: true},
		},
		Constraints: "inputs/constraints.schema.json",
	}, files)
	if err != nil {
		t.Fatal(err)
	}

	defs, _ := doc.Get("$defs")
	for key, res := range defs.(value.Object).All() {
		id, _ := res.(value.Object).Get("$id")
		r, ok := c.roots[id.(value.String).Text()]
		if !ok || r != (root{key, ""}) {
			t.Errorf("$defs %q has $id %v, which the compiler registered for %+v", key, id, r)
		}
	}
	if n := defs.(value.Object).Len(); n != len(combinedFiles) {
		t.Errorf("$defs has %d resources, want %d", n, len(combinedFiles))
	}

	combined, fs := c.Combined(doc)
	if len(fs) > 0 {
		t.Fatal(describe(fs))
	}
	inputs := map[string]*Schema{}
	for _, name := range []string{"vendor", "count", "constraints"} {
		inputs[name], _ = c.Schema("inputs/" + name + ".schema.json")
	}
	cases := []string{
		`{"vendor": {"name": "Acme", "address": {"city": "Oslo"}}, "count": 2}`,
		`{"count": 3}`,
		`{"vendor": {"name": "A", "address": {"city": 5, "zip": "x"}}, "count": 0}`,
		`{"vendor": {}, "count": 1.5}`,
		`{"count": 1}`,
	}
	for _, src := range cases {
		obj := mustValue(t, src).(value.Object)
		got, err := combined.Validate(obj)
		if err != nil {
			t.Fatal(err)
		}
		var want []string
		valid := true
		for name, v := range obj.All() {
			fails, err := inputs[name].Validate(v)
			if err != nil {
				t.Fatal(err)
			}
			want = append(want, failureKeys(fails, []string{name})...)
			valid = valid && len(fails) == 0
		}
		cons, err := inputs["constraints"].Validate(obj)
		if err != nil {
			t.Fatal(err)
		}
		want = append(want, failureKeys(cons, nil)...)
		valid = valid && len(cons) == 0
		slices.Sort(want)
		if (len(got) == 0) != valid {
			t.Errorf("%s: the combined schema says valid %v, the files %v", src, len(got) == 0, valid)
		}
		// The combined schema's own keywords (its properties, required and
		// additionalProperties) are what the files do not have; every
		// other failure is a file's.
		var gotKeys []string
		for _, k := range failureKeys(got, nil) {
			if !strings.HasPrefix(k, CombinedEntry+"#") {
				gotKeys = append(gotKeys, k)
			}
		}
		if !slices.Equal(gotKeys, want) {
			t.Errorf("%s:\ncombined\n%s\nfiles\n%s", src, strings.Join(gotKeys, "\n"), strings.Join(want, "\n"))
		}
		findings, err := schemafail.Map(target(c), got...)
		if err != nil {
			t.Errorf("%s: schemafail.Map refuses the combined schema's tree: %v", src, err)
		}
		for _, f := range findings {
			if s, ok := f.Schema(); ok && s.Entry == CombinedEntry && !strings.HasPrefix(s.Pointer, "/properties") && s.Pointer != "/required" {
				t.Errorf("%s: a finding names the combined schema at %s", src, s.Pointer)
			}
		}
	}
	// The input missing from the inputs object is the combined schema's
	// required, which no file has. (The constraints' if holds vacuously
	// with count absent, so their then fails too.)
	got, _ := combined.Validate(mustValue(t, `{}`))
	if keys := failureKeys(got, nil); !slices.Contains(keys, CombinedEntry+"#/required required @/ map[missing:[count]]") {
		t.Errorf("an absent required input: %q", keys)
	}
}

// TestCombinedSchemaSize refuses a combined schema larger than
// combined-schema-size.
func TestCombinedSchemaSize(t *testing.T) {
	c, fs := compileFiles(t, lower(t, "combined-schema-size=10"), map[string]string{"inputs/a.json": `{}`})
	if len(fs) > 0 {
		t.Fatal(describe(fs))
	}
	s, fs := c.Combined(mustValue(t, `{"$id": "metais:///.compiled/schema.json", "type": "object"}`))
	f := only(t, fs)
	if s != nil || f.Code() != finding.Code("limit.package") || param(t, f, "limit") != "combined-schema-size" {
		t.Errorf("got %v, %s", s, describe(fs))
	}
}
