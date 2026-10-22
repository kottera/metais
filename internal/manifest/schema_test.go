// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package manifest

import (
	"encoding/json"
	"regexp"
	"slices"
	"strings"
	"testing"

	"metais.org/metais/internal/specdata"
)

// schemaTree is the embedded manifest schema decoded as plain JSON.
func schemaTree(t testing.TB) map[string]any {
	t.Helper()
	var s map[string]any
	if err := json.Unmarshal(specdata.ManifestSchema10, &s); err != nil {
		t.Fatal(err)
	}
	return s
}

// lookup returns the value at the slash-separated path below s.
func lookup(t testing.TB, s any, path string) any {
	t.Helper()
	cur := s
	for _, tok := range strings.Split(strings.TrimPrefix(path, "/"), "/") {
		switch c := cur.(type) {
		case map[string]any:
			v, ok := c[tok]
			if !ok {
				t.Fatalf("the schema has nothing at %s", path)
			}
			cur = v
		case []any:
			var i int
			for _, r := range tok {
				i = i*10 + int(r-'0')
			}
			cur = c[i]
		default:
			t.Fatalf("the schema has nothing at %s", path)
		}
	}
	return cur
}

// TestSchemaPatterns holds each pattern of the schema to the expression
// Build judges the same field with, byte for byte, so the two cannot
// drift (ADR-002 D10 item 7).
func TestSchemaPatterns(t *testing.T) {
	s := schemaTree(t)
	for path, want := range map[string]string{
		"/properties/name/pattern":                     namePattern,
		"/properties/name/not/anyOf/0/pattern":         nameDevicePattern,
		"/properties/name/not/anyOf/1/pattern":         nameLongSegmentPattern,
		"/properties/version/pattern":                  semverPattern,
		"/properties/input_constraints/pattern":        schemaPathPattern,
		"/$defs/line/pattern":                          linePattern,
		"/$defs/prose/pattern":                         prosePattern,
		"/$defs/http_uri/pattern":                      httpURIPattern,
		"/$defs/author/properties/email/pattern":       emailPattern,
		"/$defs/input/properties/name/pattern":         identifierPattern,
		"/$defs/input/properties/schema/pattern":       schemaPathPattern,
		"/$defs/input/properties/meaning/pattern":      meaningPathPattern,
		"/$defs/container/properties/mimetype/pattern": mimetypePattern,
		"/$defs/id/pattern":                            identifierPattern,
		"/$defs/template/pattern":                      relativePathPattern,
		"/$defs/asset/pattern":                         relativePathPattern,
		"/$defs/mirror/pattern":                        relativePathPattern,
		"/$defs/validate/pattern":                      relativePathPattern,
		"/$defs/validate/not/pattern":                  compiledPattern,
	} {
		if got := lookup(t, s, path); got != want {
			t.Errorf("%s = %q, want %q", path, got, want)
		}
	}
	// The value sets Build judges by are the schema's.
	for path, want := range map[string][]string{
		"/$defs/input/properties/name/not/enum":                 reservedInputNames,
		"/$defs/input/properties/forms/items/enum":              {"json", "yaml", "csv", "text", "markdown"},
		"/$defs/input/properties/csv/properties/delimiter/enum": {",", ";", "|", "\t"},
		"/$defs/kind/enum":                                      kinds,
		"/$defs/escape/enum":                                    {"none", "html"},
		"/$defs/newline/enum":                                   {"as-written", "crlf"},
		"/$defs/container/properties/profile/enum":              {"ooxml", "odf", "epub"},
		"/required": rootRequired,
	} {
		var got []string
		for _, v := range lookup(t, s, path).([]any) {
			got = append(got, v.(string))
		}
		if !slices.Equal(got, want) {
			t.Errorf("%s = %q, want %q", path, got, want)
		}
	}
	var props []string
	for name := range lookup(t, s, "/properties").(map[string]any) {
		props = append(props, name)
	}
	slices.Sort(props)
	want := slices.Sorted(slices.Values(append(slices.Clone(rootRequired), rootOptional...)))
	if !slices.Equal(props, want) {
		t.Errorf("the root's properties %q, want %q", props, want)
	}
}

// TestSchemaIdentity is ADR-002 D10 items 1 to 3: a 2020-12 document, its
// permanent $id, metais pinned to "1.0", and no external $ref.
func TestSchemaIdentity(t *testing.T) {
	s := schemaTree(t)
	if s["$schema"] != "https://json-schema.org/draft/2020-12/schema" || s["$id"] != schemaID {
		t.Errorf("$schema %v, $id %v", s["$schema"], s["$id"])
	}
	if got := lookup(t, s, "/properties/metais/const"); got != "1.0" {
		t.Errorf("metais const %v", got)
	}
	walkSchemas(t, s, "", func(ptr string, sch map[string]any) {
		ref, ok := sch["$ref"].(string)
		if !ok {
			return
		}
		name, local := strings.CutPrefix(ref, "#/$defs/")
		if !local {
			t.Errorf("%s: the $ref %q is not to one of the document's own $defs", ptr, ref)
			return
		}
		if _, ok := s["$defs"].(map[string]any)[name]; !ok {
			t.Errorf("%s: the $ref %q names no $defs member", ptr, ref)
		}
	})
	manifestSchema(t)
}

// Keywords ADR-003 D3 admits (items 3 and 4), which ADR-010 D5's closed
// compiler accepts. The schema uses no format at all (ADR-003 D4).
var admittedKeywords = []string{
	"$schema", "$id", "$ref", "$defs", "$anchor", "$dynamicRef", "$dynamicAnchor", "$comment",
	"allOf", "anyOf", "oneOf", "not", "if", "then", "else", "dependentSchemas",
	"prefixItems", "items", "contains", "properties", "patternProperties", "additionalProperties", "propertyNames",
	"unevaluatedItems", "unevaluatedProperties",
	"type", "enum", "const", "multipleOf", "maximum", "exclusiveMaximum", "minimum", "exclusiveMinimum",
	"maxLength", "minLength", "pattern", "maxItems", "minItems", "uniqueItems", "maxContains", "minContains",
	"maxProperties", "minProperties", "required", "dependentRequired",
	"title", "description", "default", "deprecated", "readOnly", "writeOnly", "examples",
}

// walkSchemas calls f on every schema object in s, by its JSON Pointer.
func walkSchemas(t testing.TB, s any, ptr string, f func(string, map[string]any)) {
	sch, ok := s.(map[string]any)
	if !ok {
		if b, isBool := s.(bool); !isBool {
			t.Errorf("%s is not a schema", ptr)
		} else if b {
			t.Errorf("%s is the schema true, which closes nothing", ptr)
		}
		return
	}
	f(ptr, sch)
	for k, v := range sch {
		at := ptr + "/" + k
		switch k {
		case "properties", "patternProperties", "$defs", "dependentSchemas":
			for name, sub := range v.(map[string]any) {
				walkSchemas(t, sub, at+"/"+name, f)
			}
		case "items", "not", "if", "then", "else", "contains", "additionalProperties", "propertyNames",
			"unevaluatedItems", "unevaluatedProperties":
			walkSchemas(t, v, at, f)
		case "allOf", "anyOf", "oneOf", "prefixItems":
			for i, sub := range v.([]any) {
				walkSchemas(t, sub, at+"/"+string(rune('0'+i)), f)
			}
		}
	}
}

// TestSchemaKeywords: the schema is one ADR-010 D5's closed compiler
// accepts, with no keyword outside ADR-003 D3's set and no format, and
// patterns without lookaround that Go's regexp reads (ADR-002 D10 item 7).
func TestSchemaKeywords(t *testing.T) {
	lookaround := regexp.MustCompile(`\(\?`)
	walkSchemas(t, schemaTree(t), "", func(ptr string, sch map[string]any) {
		for k, v := range sch {
			if !slices.Contains(admittedKeywords, k) {
				t.Errorf("%s: the keyword %q is not one ADR-003 D3 admits", ptr, k)
			}
			if k == "pattern" {
				p := v.(string)
				if lookaround.MatchString(p) {
					t.Errorf("%s: the pattern %q has a group construct", ptr, p)
				}
				if _, err := regexp.Compile(p); err != nil {
					t.Errorf("%s: %v", ptr, err)
				}
			}
		}
		if _, ok := sch["format"]; ok {
			t.Errorf("%s: a format", ptr)
		}
	})
}

// TestSchemaFieldsTitled is ADR-002 D10 item 4: every property carries a
// title and a description, its own or those of the $defs member it is a
// bare reference to, so a finding about a field can name it. A false
// subschema forbids its property and names nothing.
func TestSchemaFieldsTitled(t *testing.T) {
	s := schemaTree(t)
	defs := s["$defs"].(map[string]any)
	titled := func(m map[string]any) bool {
		title, _ := m["title"].(string)
		desc, _ := m["description"].(string)
		return title != "" && desc != ""
	}
	walkSchemas(t, s, "", func(ptr string, sch map[string]any) {
		props, ok := sch["properties"].(map[string]any)
		if !ok {
			return
		}
		for name, v := range props {
			at := ptr + "/properties/" + name
			if v == false {
				continue
			}
			p := v.(map[string]any)
			if titled(p) {
				continue
			}
			ref, _ := p["$ref"].(string)
			target, _ := defs[strings.TrimPrefix(ref, "#/$defs/")].(map[string]any)
			if len(p) != 1 || target == nil || !titled(target) {
				t.Errorf("%s has no title and description of its own or of a $defs member", at)
			}
		}
	})
	// Every object a declaration or an author is, too, so a finding about
	// a whole one names what it is.
	for _, name := range []string{"author", "input", "output", "entry", "container",
		"template_output", "asset_output", "mirror_output", "container_output",
		"template_entry", "asset_entry", "mirror_entry", "container_entry"} {
		if !titled(defs[name].(map[string]any)) {
			t.Errorf("$defs/%s has no title and description", name)
		}
	}
}

// TestSchemaClosed is ADR-002 D10 item 1 and D4 item 4: every object the
// schema defines refuses a member it does not define.
func TestSchemaClosed(t *testing.T) {
	walkSchemas(t, schemaTree(t), "", func(ptr string, sch map[string]any) {
		if _, ok := sch["properties"]; !ok || strings.Contains(ptr, "/dependentSchemas/") ||
			strings.HasSuffix(ptr, "/if") || strings.HasSuffix(ptr, "/else") {
			// A conditional's properties constrain an object its own
			// schema already closes.
			return
		}
		if sch["additionalProperties"] != false {
			t.Errorf("%s defines properties and is not closed", ptr)
		}
	})
}
