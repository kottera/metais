// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package schema

import (
	"net/url"
	"strconv"
	"strings"

	"github.com/kottera/metais/internal/finding"
	"github.com/kottera/metais/internal/limits"
	"github.com/kottera/metais/internal/liquid/value"
	"github.com/kottera/metais/internal/regexprofile"
)

// Dialect is the one $schema value a schema may declare (ADR-003 D3 item
// 2). Absent, it is assumed.
const Dialect = "https://json-schema.org/draft/2020-12/schema"

// shape is what a keyword's value holds: a value the walk does not enter,
// a subschema, an array of subschemas, or an object of them.
type shape uint8

const (
	shapeValue shape = iota
	shapeSchema
	shapeSchemaArray
	shapeSchemaMap
)

// keywords is every keyword of draft 2020-12's core, applicator,
// unevaluated, validation, format-annotation, content and meta-data
// vocabularies (ADR-003 D3 items 3 and 4). Any other keyword but one
// beginning "x-" is refused (D3 item 5, RULINGS R1-002 and R1-016), and so
// is $vocabulary (D3 item 2), which the walk reports as a dialect.
var keywords = map[string]shape{
	// Core.
	"$schema": shapeValue, "$id": shapeValue, "$ref": shapeValue, "$anchor": shapeValue,
	"$dynamicRef": shapeValue, "$dynamicAnchor": shapeValue, "$comment": shapeValue,
	"$defs": shapeSchemaMap,
	// Applicator.
	"allOf": shapeSchemaArray, "anyOf": shapeSchemaArray, "oneOf": shapeSchemaArray,
	"not": shapeSchema, "if": shapeSchema, "then": shapeSchema, "else": shapeSchema,
	"dependentSchemas": shapeSchemaMap, "prefixItems": shapeSchemaArray, "items": shapeSchema,
	"contains": shapeSchema, "properties": shapeSchemaMap, "patternProperties": shapeSchemaMap,
	"additionalProperties": shapeSchema, "propertyNames": shapeSchema,
	// Unevaluated.
	"unevaluatedItems": shapeSchema, "unevaluatedProperties": shapeSchema,
	// Validation.
	"type": shapeValue, "enum": shapeValue, "const": shapeValue,
	"multipleOf": shapeValue, "maximum": shapeValue, "exclusiveMaximum": shapeValue,
	"minimum": shapeValue, "exclusiveMinimum": shapeValue,
	"maxLength": shapeValue, "minLength": shapeValue, "pattern": shapeValue,
	"maxItems": shapeValue, "minItems": shapeValue, "uniqueItems": shapeValue,
	"maxContains": shapeValue, "minContains": shapeValue,
	"maxProperties": shapeValue, "minProperties": shapeValue,
	"required": shapeValue, "dependentRequired": shapeValue,
	// Format, under ADR-003 D4.
	"format": shapeValue,
	// Content, annotations only.
	"contentEncoding": shapeValue, "contentMediaType": shapeValue, "contentSchema": shapeSchema,
	// Meta-data.
	"title": shapeValue, "description": shapeValue, "default": shapeValue,
	"deprecated": shapeValue, "readOnly": shapeValue, "writeOnly": shapeValue, "examples": shapeValue,
}

// metaSchemas are the 2020-12 meta-schemas a reference may name, which
// the library carries built in (ADR-003 D5 item 3, ADR-009 D9 item 6).
var metaSchemas = map[string]bool{
	Dialect: true,
	"https://json-schema.org/draft/2020-12/meta/core":              true,
	"https://json-schema.org/draft/2020-12/meta/applicator":        true,
	"https://json-schema.org/draft/2020-12/meta/unevaluated":       true,
	"https://json-schema.org/draft/2020-12/meta/validation":        true,
	"https://json-schema.org/draft/2020-12/meta/meta-data":         true,
	"https://json-schema.org/draft/2020-12/meta/format-annotation": true,
	"https://json-schema.org/draft/2020-12/meta/content":           true,
}

// resource is a schema resource of a file: its root, or a subschema that
// declares its own $id.
type resource struct {
	uri string // canonical, absolute, without a fragment
	ptr string // where it is in the file
	v   value.Value
}

// reference is a $ref or $dynamicRef the walk found.
type reference struct {
	ptr  string // of the keyword
	base *url.URL
	text string
}

// walker is the pre-check of one schema file (ADR-010 D5 item 3): what
// the library would let through or never sees, checked before it sees
// the file.
type walker struct {
	entry     string
	lim       limits.Set
	findings  []finding.Finding
	resources []resource
	refs      []reference
	anchors   []resource // each $anchor and $dynamicAnchor, by its URI
	deep      bool       // schema-depth has been reported
}

func escapeToken(t string) string {
	return strings.NewReplacer("~", "~0", "/", "~1").Replace(t)
}

// walk checks the subschema v at ptr, depth levels below the file's root
// schema, whose base URI is base.
func (w *walker) walk(v value.Value, ptr string, base *url.URL, depth int) {
	o, ok := v.(value.Object)
	if !ok {
		// A boolean schema has nothing to check, and a value that is not a
		// schema is the meta-schema's to refuse.
		return
	}
	if depth > int(w.lim.Value(limits.SchemaDepth)) {
		if !w.deep {
			w.deep = true
			w.findings = append(w.findings, w.lim.Breached(limits.SchemaDepth, *at(w.entry, ptr), int64(depth), true))
		}
		return
	}
	if id, ok := o.Get("$id"); ok {
		if s, ok := id.(value.String); ok {
			// An $id with a fragment is the meta-schema's to refuse; it
			// declares no resource.
			if u, err := url.Parse(s.Text()); err == nil && u.Fragment == "" {
				abs := base.ResolveReference(u)
				abs.Fragment, abs.RawFragment = "", ""
				base = abs
				w.resources = append(w.resources, resource{uri: abs.String(), ptr: ptr, v: v})
			}
		}
	}
	for name, member := range o.All() {
		at := ptr + "/" + escapeToken(name)
		sh, known := keywords[name]
		switch {
		case name == "$vocabulary":
			w.findings = append(w.findings, report(CodeDialect, w.entry, at, finding.Params{"reason": str(dialectVocabulary)}))
			continue
		case !known:
			if !strings.HasPrefix(name, "x-") {
				w.findings = append(w.findings, report(CodeKeywordUnknown, w.entry, at, finding.Params{"keyword": str(name)}))
			}
			continue
		}
		s, isString := member.(value.String)
		switch name {
		case "$schema":
			if isString && s.Text() != Dialect {
				w.findings = append(w.findings, report(CodeDialect, w.entry, at, finding.Params{"reason": str(dialectSchema)}))
			}
		case "format":
			if _, ok := formats[s.Text()]; isString && !ok {
				w.findings = append(w.findings, report(CodeFormatUnsupported, w.entry, at, finding.Params{"format": str(s.Text())}))
			}
		case "pattern":
			if isString {
				w.pattern(s.Text(), at)
			}
		case "patternProperties":
			if po, ok := member.(value.Object); ok {
				for key := range po.All() {
					w.pattern(key, at+"/"+escapeToken(key))
				}
			}
		case "$anchor", "$dynamicAnchor":
			if isString {
				w.anchors = append(w.anchors, resource{uri: base.String() + "#" + s.Text(), ptr: ptr})
			}
		case "$ref", "$dynamicRef":
			if isString {
				w.refs = append(w.refs, reference{ptr: at, base: base, text: s.Text()})
			}
		}
		switch sh {
		case shapeSchema:
			w.walk(member, at, base, depth+1)
		case shapeSchemaArray:
			if a, ok := member.(value.Array); ok {
				for i, sub := range a.All() {
					w.walk(sub, at+"/"+strconv.Itoa(i), base, depth+1)
				}
			}
		case shapeSchemaMap:
			if m, ok := member.(value.Object); ok {
				for key, sub := range m.All() {
					w.walk(sub, at+"/"+escapeToken(key), base, depth+1)
				}
			}
		}
	}
}

// pattern compiles a pattern or a patternProperties name by the profile,
// which refuses what ADR-009 D10 item 6 refuses and checks pattern-length
// and regex-program.
func (w *walker) pattern(p, at string) {
	_, err := regexprofile.Compile(p, w.lim, nil)
	if err == nil {
		return
	}
	if f, ok := regexFinding(err, w.entry, at, w.lim); ok {
		w.findings = append(w.findings, f)
		return
	}
	// The profile's contract: any other error is a defect in the tool.
	panic(err)
}

// climbs reports whether ref's path, merged with base's directory by RFC
// 3986 §5.2.3, would go above the root before dot-segment removal clamps
// it (ADR-003 D5 item 5).
func climbs(base, ref string) bool {
	segs := strings.Split(ref, "/")
	depth := 0
	if !strings.HasPrefix(ref, "/") {
		if i := strings.LastIndexByte(base, '/'); i > 0 {
			depth = strings.Count(base[:i], "/")
		}
	} else {
		segs = segs[1:]
	}
	for _, s := range segs {
		switch s {
		case "..":
			if depth--; depth < 0 {
				return true
			}
		case ".", "":
		default:
			depth++
		}
	}
	return false
}
