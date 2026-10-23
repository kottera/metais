// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package cli

import "strings"

// topicSchemasExample is chapter 13's example of the rules across the
// rows of a table (§13.1, step 5), laid out to fit the help's width.
// TestSchemasTopicExample holds it to the chapter's JSON, compiles it and
// lists its rules.
const topicSchemasExample = `{
  "type": "array",
  "uniqueKeys": ["/criterion"],
  "coverage": {
    "key": "/criterion",
    "values": ["1.1.1", "1.2.1", "1.2.2", "1.2.3", "1.2.4", "1.2.5"]
  },
  "items": {
    "type": "object",
    "required": ["criterion", "conformance"],
    "properties": {
      "criterion": {
        "type": "string",
        "enum": [
          "1.1.1", "1.2.1", "1.2.2", "1.2.3", "1.2.4", "1.2.5", "1.2.6"
        ]
      },
      "conformance": {
        "enum": [
          "Supports", "Partially Supports", "Does Not Support",
          "Not Applicable", "Not Evaluated"
        ]
      }
    }
  }
}
`

// topicSchemasRules are the rules schema --fields lists for
// topicSchemasExample, in its order, a line cut where it ends "…".
var topicSchemasRules = []string{
	"rows must include, at `/criterion`, each of: \"1.1.1\", \"1.2.1\", …",
	"no two rows share `/criterion`",
}

// keywordGroup is one vocabulary of the dialect as help schemas lists it.
type keywordGroup struct {
	Name  string
	Names []string
}

// assertedKeywords and annotationKeywords are every keyword a schema may
// use but one beginning x-, by vocabulary (ADR-003 D3 items 3 and 4,
// Amendment A2). TestSchemasTopicKeywords holds them to schema.Keywords.
var (
	assertedKeywords = []keywordGroup{
		{"Core", []string{"$schema", "$id", "$ref", "$defs", "$anchor", "$dynamicRef", "$dynamicAnchor", "$comment"}},
		{"Applicator", []string{"allOf", "anyOf", "oneOf", "not", "if", "then", "else", "dependentSchemas",
			"properties", "patternProperties", "additionalProperties", "propertyNames",
			"prefixItems", "items", "contains"}},
		{"Unevaluated", []string{"unevaluatedProperties", "unevaluatedItems"}},
		{"Validation", []string{"type", "enum", "const", "multipleOf", "minimum", "exclusiveMinimum",
			"maximum", "exclusiveMaximum", "minLength", "maxLength", "pattern", "minItems", "maxItems",
			"uniqueItems", "minContains", "maxContains", "minProperties", "maxProperties",
			"required", "dependentRequired"}},
		{"Rows", []string{"uniqueKeys", "coverage"}},
		{"Format", []string{"format"}},
	}
	annotationKeywords = []keywordGroup{
		{"Meta-data", []string{"title", "description", "default", "deprecated", "readOnly", "writeOnly", "examples"}},
		{"Content", []string{"contentEncoding", "contentMediaType", "contentSchema"}},
	}
)

// formatDoc is one asserted format as help schemas gives it: what it
// holds, and a string of it. TestSchemasTopicFormats holds the rows to
// schema.Formats and validates each example against its format.
type formatDoc struct {
	Name, Text, Example string
}

var formatDocs = []formatDoc{
	{"date-time", "An RFC 3339 date and time, with its offset", "2026-10-23T09:30:00Z"},
	{"date", "An RFC 3339 full date", "2026-10-23"},
	{"time", "An RFC 3339 time, with its offset", "09:30:00+02:00"},
	{"duration", "An RFC 3339 duration (Appendix A)", "P1DT12H"},
	{"email", "An RFC 5321 mailbox", "ada@example.org"},
	{"hostname", "An RFC 1123 host name", "docs.example.org"},
	{"ipv4", "An IPv4 address in dotted-decimal form", "192.0.2.1"},
	{"ipv6", "An RFC 4291 IPv6 address", "2001:db8::1"},
	{"uri", "An absolute RFC 3986 URI", "https://example.org/a?b=c"},
	{"uri-reference", "An RFC 3986 URI or relative reference", "../a#b"},
	{"uuid", "An RFC 4122 UUID in its string form", "f81d4fae-7dec-11d0-a765-00a0c91e6bf6"},
	{"uri-template", "An RFC 6570 URI template", "https://example.org/{id}"},
	{"json-pointer", "An RFC 6901 JSON Pointer", "/rows/0/name"},
	{"relative-json-pointer", "A relative JSON Pointer", "1/name"},
}

// schemasTopic is metais help schemas (ADR-003 A2 item 10): the dialect
// an author writes, and the rules across a table's rows with chapter
// 13's example first.
func schemasTopic(h *helpText, _ []*Command) {
	h.line("Example:")
	h.blank()
	h.para("A table whose rows each answer one criterion, from the specification's "+
		"chapter 13. uniqueKeys refuses a second row for one criterion, and coverage "+
		"reports each Level A or AA criterion no row answers; a row for 1.2.6, a Level AAA "+
		"criterion, is allowed and not required.", "")
	h.blank()
	for l := range strings.SplitSeq(strings.TrimSuffix(topicSchemasExample, "\n"), "\n") {
		h.line("  " + l)
	}
	h.blank()
	h.para("metais schema <package> <input> --fields lists its two rules in words, "+
		"directly beneath the array's line:", "")
	h.blank()
	for _, l := range topicSchemasRules {
		h.line("  " + l)
	}

	h.heading("The dialect:")
	h.para("Every .json, .yaml and .yml file under inputs/ is a schema of JSON Schema "+
		"2020-12, compiled when the package is opened whether or not an input names it. "+
		"$schema, where a schema gives it, is https://json-schema.org/draft/2020-12/schema; "+
		"any other, and $vocabulary, is refused. A schema uses only these keywords:", "")
	h.blank()
	h.line("Asserted:")
	for _, g := range assertedKeywords {
		h.item("  "+g.Name, 15, strings.Join(g.Names, " "))
	}
	h.line("Annotations, never asserted:")
	for _, g := range annotationKeywords {
		h.item("  "+g.Name, 15, strings.Join(g.Names, " "))
	}
	h.blank()
	h.para("default fills a value nobody supplied (metais help inputs). Any other keyword "+
		"is refused as schema.keyword_unknown, so a misspelt requried or maxLenght is "+
		"found, not ignored. The one exception is a keyword beginning x-: it is the "+
		"author's own annotation, carried unchanged and never interpreted, so it may hold "+
		"a note but never a rule.", "")
	h.blank()
	h.para("A $ref resolves only inside the package: to a schema file under inputs/ by "+
		"its path from the referring file, or by its root's $id when it has one, and then "+
		"only by that $id; to an anchor or a JSON Pointer within one; or to a 2020-12 "+
		"meta-schema. Nothing is ever fetched.", "")
	h.blank()
	h.para("pattern and the names of patternProperties are ECMA-262 regular expressions, "+
		"unanchored: write ^ and $ to match a whole string. A lookahead, a lookbehind, a "+
		"backreference and a repeat bound above 1000 are refused.", "")

	h.heading("Formats:")
	h.para("format names one of these, and is asserted: a string that is not of its "+
		"format is a finding, and a value that is not a string passes.", "")
	h.blank()
	for _, f := range formatDocs {
		h.item("  "+f.Name, 25, f.Text+": "+f.Example)
	}
	h.blank()
	h.para("Any other format, idn-email, idn-hostname, iri, iri-reference and regex "+
		"among them, is refused as schema.format_unsupported: write a pattern instead, or "+
		"say what you expect in the input's meaning document.", "")

	h.heading("Rules across the rows of a table:")
	h.para("JSON Schema judges each row of a table alone. Two keywords of this "+
		"specification, written in the schema of the array, state a rule across its "+
		"rows:", "")
	h.blank()
	h.item("  uniqueKeys", 14, "The columns that identify a row, as 1 to 8 JSON "+
		"Pointers into one row. A later row with the same values in all of them is "+
		"refused, at that row. Several pointers make one key together: "+
		"[\"/standard\", \"/clause\"] allows one clause in two standards.")
	h.item("  coverage", 14, "An object of key, one such pointer, and values, the "+
		"values that column must hold. Each value no row holds is reported. List what "+
		"is required, not everything that may appear.")
	h.blank()
	h.para("Values are compared as JSON values, so 1.0 equals 1. A row with no value "+
		"in a key column takes no part in either rule, so make the column required when "+
		"every row needs one. A rule between the fields of one row needs neither "+
		"keyword: write if and then in the item schema. A rule between inputs belongs in "+
		"the input constraints (input_constraints in metais help manifest).", "")
	h.blank()
	h.para("Both keywords are this specification's, so a JSON Schema tool that knows "+
		"only 2020-12 ignores them: say each rule in the input's meaning document too, so "+
		"that a supplier gathers complete data before validating it. metais check "+
		"refuses a coverage value the item schema refuses, or more values than maxItems "+
		"allows, as schema.unsatisfiable, and warns of a key pointer no row declares, as "+
		"schema.key_not_declared.", "")
	h.blank()
	h.para("metais check [<dir>] reports every finding of every schema at once. metais "+
		"schema <package> <input> --fields lists an input's fields and rules, one line "+
		"each. The specification's chapter 5 states the dialect, and chapter 13 how to "+
		"write a package's schemas.", "")
}
