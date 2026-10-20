// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package finding

import (
	"encoding/json"
	"fmt"
	"maps"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/kottera/metais/internal/specdata"
)

// TestManifestKeywordCodes is ADR-006 A1 item 4's tripwire. It walks the
// manifest schema the binary embeds, and fails when a keyword the schema
// uses maps to a manifest.* keyword code the registry lacks, and when the
// registry holds a manifest.* keyword code that no keyword of the schema
// maps to, so the manifest family registers exactly the keyword codes a
// manifest can be given.
//
// "Maps to" is D6's mapping, read where the keyword stands. A keyword in a
// place whose failure the mapping never reports maps to no code: the
// subschema of not, the if subschema and the subschema of propertyNames
// report nothing from inside them (D6 items 7 and 10). A oneOf maps to
// manifest.one_of, and to manifest.one_of.ambiguous only when two of its
// branches could both match: the tripwire counts two branches exclusive
// when, through their $refs, one of them allows only an object, and one
// requires a member the other, a closed object, does not allow. Any other
// pair it counts as able to both match, the stricter reading: without a
// type, a value that is not an object matches both.
func TestManifestKeywordCodes(t *testing.T) {
	for _, p := range manifestCodeProblems(specdata.ManifestSchema10, registeredManifestCodes(Registry())) {
		t.Error(p)
	}
}

// TestManifestKeywordCodesControl breaks the schema or the registry once
// per way the tripwire must catch, and requires each break to be caught.
func TestManifestKeywordCodesControl(t *testing.T) {
	codes := registeredManifestCodes(Registry())
	without := func(c string) map[string]bool {
		m := maps.Clone(codes)
		delete(m, c)
		return m
	}
	with := func(c string) map[string]bool {
		m := maps.Clone(codes)
		m[c] = true
		return m
	}
	schema := string(specdata.ManifestSchema10)
	edit := func(old, new string) []byte {
		if !strings.Contains(schema, old) {
			t.Fatalf("the manifest schema holds no %q to edit", old)
		}
		return []byte(strings.Replace(schema, old, new, 1))
	}
	// untyped removes type from the named outputs' branches, which still
	// exclude each other's required members as objects, while a string
	// matches every branch without a type.
	untyped := func(kinds ...string) []byte {
		s := schema
		for _, k := range kinds {
			old := `"type": "object",
      "required": ["id", "` + k + `"`
			if !strings.Contains(s, old) {
				t.Fatalf("the manifest schema holds no %q to edit", old)
			}
			s = strings.Replace(s, old, `"required": ["id", "`+k+`"`, 1)
		}
		return []byte(s)
	}
	for _, c := range []struct {
		name   string
		schema []byte
		codes  map[string]bool
		want   string
	}{
		{"a code the schema can produce, unregistered", specdata.ManifestSchema10, without("manifest.pattern"),
			"the manifest schema uses pattern, at /$defs/"},
		{"a false schema's code, unregistered", specdata.ManifestSchema10, without("manifest.forbidden"),
			"manifest.forbidden, which the registry lacks"},
		{"a keyword the schema does not use, registered", specdata.ManifestSchema10, with("manifest.minimum"),
			"manifest.minimum is registered, and no keyword of the manifest schema maps to it"},
		{"a keyword added to the schema", edit(`"uniqueItems": true`, `"uniqueItems": true, "minProperties": 1`), codes,
			"the manifest schema uses minProperties, at /$defs/"},
		{"an anyOf out from beneath its not", edit(`"not": {
        "anyOf"`, `"then": {
        "anyOf"`), codes, "anyOf, at /properties/name/then/anyOf, which maps to manifest.any_of"},
		{"a oneOf whose branches can both match", edit(`{ "$ref": "#/$defs/asset_output" },`, `{ "$ref": "#/$defs/asset_output" }, { "$ref": "#/$defs/asset_output" },`), codes,
			"oneOf, at /$defs/output/oneOf, which maps to manifest.one_of.ambiguous"},
		{"two closed oneOf branches without their type", untyped("template", "asset"), codes,
			"oneOf, at /$defs/output/oneOf, which maps to manifest.one_of.ambiguous"},
		{"a keyword the tripwire does not know", edit(`"uniqueItems": true`, `"uniqueItems": true, "$dynamicRef": "#x"`), codes,
			"a keyword the tripwire does not know"},
	} {
		t.Run(c.name, func(t *testing.T) {
			ps := manifestCodeProblems(c.schema, c.codes)
			if !slices.ContainsFunc(ps, func(p string) bool { return strings.Contains(p, c.want) }) {
				t.Errorf("not caught: want a problem holding %q, got %q", c.want, ps)
			}
		})
	}
}

// registeredManifestCodes are the manifest.* keyword codes of reg.
func registeredManifestCodes(reg []Entry) map[string]bool {
	out := map[string]bool{}
	for _, e := range reg {
		if s, ok := strings.CutPrefix(string(e.Code), "manifest."); ok && isKeywordSuffix(s) {
			out[string(e.Code)] = true
		}
	}
	return out
}

// isKeywordSuffix reports whether s is a keyword code's suffix (D6 item
// 4).
func isKeywordSuffix(s string) bool {
	return slices.ContainsFunc(schemaKeywords, func(k schemaKeyword) bool { return k.suffix == s })
}

// The keywords that are annotations or identifiers, and the content
// vocabulary, which ADR-003 D3 item 4 never asserts: none maps to a code.
var silentKeywords = []string{
	"$anchor", "$comment", "$defs", "$id", "$schema", "contentEncoding", "contentMediaType", "contentSchema",
	"default", "deprecated", "description", "examples", "readOnly", "title", "writeOnly",
}

// assertionSuffixes are the assertion keywords and the suffix of the code
// each maps to (D6 item 4).
var assertionSuffixes = map[string]string{
	"type": "type", "enum": "enum", "const": "const", "required": "required",
	"dependentRequired": "dependent_required", "minLength": "min_length", "maxLength": "max_length",
	"pattern": "pattern", "format": "format", "minimum": "minimum", "maximum": "maximum",
	"exclusiveMinimum": "exclusive_minimum", "exclusiveMaximum": "exclusive_maximum",
	"multipleOf": "multiple_of", "minItems": "min_items", "maxItems": "max_items",
	"uniqueItems": "unique_items", "minProperties": "min_properties", "maxProperties": "max_properties",
	"minContains": "min_contains", "maxContains": "max_contains",
}

// manifestCodeProblems walks schema from its root, following its $refs,
// and lists every keyword whose code codes lacks, then every code of codes
// no keyword maps to.
func manifestCodeProblems(schema []byte, codes map[string]bool) []string {
	var root any
	if err := json.Unmarshal(schema, &root); err != nil {
		return []string{"the manifest schema: " + err.Error()}
	}
	w := &codeWalk{root: root, seen: map[string]bool{}, needed: map[string]string{}}
	w.walk("", root, "")
	problems := w.problems
	for _, c := range slices.Sorted(maps.Keys(w.needed)) {
		if !codes[c] {
			problems = append(problems, fmt.Sprintf("%s, which the registry lacks (ADR-006 A1 item 4)", w.needed[c]))
		}
	}
	for _, c := range slices.Sorted(maps.Keys(codes)) {
		if _, ok := w.needed[c]; !ok {
			problems = append(problems, fmt.Sprintf("%s is registered, and no keyword of the manifest schema maps to it (ADR-006 A1 item 1)", c))
		}
	}
	return problems
}

// codeWalk is one walk of a schema: the codes its keywords map to, each
// with where it was first needed.
type codeWalk struct {
	root     any
	seen     map[string]bool
	needed   map[string]string
	problems []string
}

func (w *codeWalk) need(kw, ptr, suffix string) {
	code := "manifest." + suffix
	if _, ok := w.needed[code]; !ok {
		w.needed[code] = fmt.Sprintf("the manifest schema uses %s, at %s, which maps to %s", kw, ptr, code)
	}
}

// walk walks the schema s at ptr, which an applicator holding it as kw
// reached: a false schema is named by that applicator (D6 item 4).
func (w *codeWalk) walk(ptr string, s any, kw string) {
	switch s := s.(type) {
	case bool:
		if s {
			return
		}
		switch kw {
		case "additionalProperties":
			w.need("additionalProperties: false", ptr, "additional_property")
		case "unevaluatedProperties":
			w.need("unevaluatedProperties: false", ptr, "unevaluated_property")
		case "items":
			w.need("items: false", ptr, "additional_item")
		case "unevaluatedItems":
			w.need("unevaluatedItems: false", ptr, "unevaluated_item")
		default:
			w.need("a false schema", ptr, "forbidden")
		}
		return
	case map[string]any:
		if w.seen[ptr] {
			return
		}
		w.seen[ptr] = true
		for _, k := range slices.Sorted(maps.Keys(s)) {
			w.keyword(ptr, s, k)
		}
	default:
		w.problems = append(w.problems, fmt.Sprintf("%s is not a schema", ptr))
	}
}

// keyword walks the keyword k of the schema object s at ptr.
func (w *codeWalk) keyword(ptr string, s map[string]any, k string) {
	at := ptr + "/" + escapeToken(k)
	v := s[k]
	if suffix, ok := assertionSuffixes[k]; ok {
		w.need(k, at, suffix)
		return
	}
	switch {
	case slices.Contains(silentKeywords, k):
	case k == "$ref":
		ref, _ := v.(string)
		target, ok := resolveLocal(w.root, ref)
		if !ok {
			w.problems = append(w.problems, fmt.Sprintf("%s: $ref %q names no schema in the manifest schema", at, ref))
			return
		}
		w.walk(strings.TrimPrefix(ref, "#"), target, k)
	case k == "properties" || k == "patternProperties" || k == "dependentSchemas":
		m, _ := v.(map[string]any)
		for _, name := range slices.Sorted(maps.Keys(m)) {
			w.walk(at+"/"+escapeToken(name), m[name], k)
		}
	case k == "additionalProperties" || k == "unevaluatedProperties" || k == "items" || k == "unevaluatedItems":
		w.walk(at, v, k)
	case k == "allOf" || k == "prefixItems":
		l, _ := v.([]any)
		for i, b := range l {
			w.walk(at+"/"+strconv.Itoa(i), b, k)
		}
	case k == "contains":
		w.need(k, at, "contains")
		w.walk(at, v, k)
	case k == "anyOf" || k == "oneOf":
		l, _ := v.([]any)
		w.need(k, at, map[string]string{"anyOf": "any_of", "oneOf": "one_of"}[k])
		if k == "oneOf" && !w.exclusive(l) {
			w.need(k, at, "one_of.ambiguous")
		}
		for i, b := range l {
			w.walk(at+"/"+strconv.Itoa(i), b, k)
		}
	case k == "not":
		// D6 item 10: nothing inside the subschema is reported.
		w.need(k, at, "not")
	case k == "propertyNames":
		// D6 item 10, as schemafail maps it: one finding, nothing inside.
		w.need(k, at, "property_name")
	case k == "if":
		// D6 item 7: the condition reports nothing.
	case k == "then" || k == "else":
		w.walk(at, v, k)
	default:
		w.problems = append(w.problems, fmt.Sprintf("%s: %s is a keyword the tripwire does not know", at, k))
	}
}

// exclusive reports whether no value can match two of branches: for each
// pair, one allows only an object, and one requires a member the other
// does not allow.
func (w *codeWalk) exclusive(branches []any) bool {
	objs := make([]map[string]any, len(branches))
	for i, b := range branches {
		objs[i] = w.deref(b)
	}
	for i := range objs {
		for j := i + 1; j < len(objs); j++ {
			if !objectOnly(objs[i]) && !objectOnly(objs[j]) ||
				!excludes(objs[i], objs[j]) && !excludes(objs[j], objs[i]) {
				return false
			}
		}
	}
	return true
}

// deref follows s's $ref chain to the schema object it ends at, or nil.
func (w *codeWalk) deref(s any) map[string]any {
	for range 32 {
		m, ok := s.(map[string]any)
		if !ok {
			return nil
		}
		ref, ok := m["$ref"].(string)
		if !ok {
			return m
		}
		if s, ok = resolveLocal(w.root, ref); !ok {
			return nil
		}
	}
	return nil
}

// objectOnly reports whether s allows only an object: its type is
// "object", or a list of that type alone.
func objectOnly(s map[string]any) bool {
	switch t := s["type"].(type) {
	case string:
		return t == "object"
	case []any:
		return len(t) > 0 && !slices.ContainsFunc(t, func(v any) bool { return v != "object" })
	}
	return false
}

// excludes reports whether a requires a member that b, a closed object,
// does not allow: b has additionalProperties false, no patternProperties,
// and the member is not among its properties, or is false there.
func excludes(a, b map[string]any) bool {
	if a == nil || b == nil || b["additionalProperties"] != false || b["patternProperties"] != nil {
		return false
	}
	props, _ := b["properties"].(map[string]any)
	req, _ := a["required"].([]any)
	for _, r := range req {
		name, _ := r.(string)
		if p, ok := props[name]; !ok || p == false {
			return true
		}
	}
	return false
}

// resolveLocal resolves a $ref of the form #/a/b within root.
func resolveLocal(root any, ref string) (any, bool) {
	p, ok := strings.CutPrefix(ref, "#")
	if !ok {
		return nil, false
	}
	cur := root
	if p == "" {
		return cur, true
	}
	if !strings.HasPrefix(p, "/") {
		return nil, false
	}
	for _, tok := range strings.Split(p[1:], "/") {
		tok = strings.ReplaceAll(strings.ReplaceAll(tok, "~1", "/"), "~0", "~")
		switch c := cur.(type) {
		case map[string]any:
			if cur, ok = c[tok]; !ok {
				return nil, false
			}
		case []any:
			i, err := strconv.Atoi(tok)
			if err != nil || i < 0 || i >= len(c) {
				return nil, false
			}
			cur = c[i]
		default:
			return nil, false
		}
	}
	return cur, true
}

// escapeToken escapes a member name as an RFC 6901 token.
func escapeToken(s string) string {
	return strings.ReplaceAll(strings.ReplaceAll(s, "~", "~0"), "/", "~1")
}
