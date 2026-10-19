// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package finding

import (
	"slices"
	"strings"
)

// CodeInputValidationPartial is the info finding that an array input was
// validated without the items that did not parse (ADR-006 D7 item 4).
const CodeInputValidationPartial Code = "input.validation.partial"

// The families a JSON Schema failure is reported in (ADR-006 D6 item 1):
// the same keyword codes under each prefix, each about its own subject.
var schemaFamilies = []struct {
	prefix  string
	subject SubjectKind
}{
	{"input", SubjectInput},
	{"manifest", SubjectManifest},
	{"output", SubjectOutput},
}

// schemaKeyword is one keyword code of ADR-006 D6 item 4, without its
// family prefix.
type schemaKeyword struct {
	suffix  string
	params  []ParamSpec
	message string
	hint    string
	span    SpanKind
}

func always(name string, t ParamType, meaning string) ParamSpec {
	return ParamSpec{Name: name, Type: t, Meaning: meaning, Always: true}
}

func optional(name string, t ParamType, meaning string) ParamSpec {
	return ParamSpec{Name: name, Type: t, Meaning: meaning}
}

const (
	meaningProperty   = "The member's name, as the supplier wrote it or as the schema declares it."
	meaningSuggestion = "The declared member name closest to this one, by ADR-006 D9 item 7's rule; present when one is close enough."
	meaningLimit      = "The keyword's value in the schema."
	meaningCount      = "How many there are."
	meaningIndex      = "The item's 0-based index in the array."
	meaningValue      = "The value, absent when the schema at the value's location is writeOnly."
)

// schemaKeywords are ADR-006 D6 item 4's codes, in the order of its table.
// The content vocabulary has none: ADR-003 D3 item 4 never asserts it.
var schemaKeywords = []schemaKeyword{
	{"type", []ParamSpec{
		{Name: "expected", Type: TypeArray, Items: TypeString, Meaning: "The type names the schema allows.", Always: true},
		always("actual", TypeString, "The value's JSON type: null, boolean, object, array, number or string."),
	}, "{field} must be of type {expected}; it is of type {actual}.", "Write a value of type {expected}.", SpanValue},
	{"enum", []ParamSpec{
		{Name: "allowed", Type: TypeArray, Items: TypeString, Meaning: "The first 20 values the schema allows, in its order, each as its canonical JSON text.", Always: true},
		always("allowed_count", TypeNumber, "How many values the schema allows."),
		optional("suggestion", TypeString, "The allowed string closest to the value, by ADR-006 D9 item 7's rule; present when one is close enough and the value is a string."),
	}, "{field} must be one of the {allowed_count} values the schema allows.", "Write one of the allowed values: {allowed}.", SpanValue},
	{"const", []ParamSpec{
		always("expected", TypeString, "The value the schema requires, as its canonical JSON text."),
	}, "{field} must be {expected}.", "Write {expected}.", SpanValue},
	{"required", []ParamSpec{
		always("property", TypeString, "The missing member's name."),
	}, "{field} is required.", "Add the member `{property}`.", SpanParent},
	{"dependent_required", []ParamSpec{
		always("property", TypeString, "The missing member's name."),
		always("because", TypeString, "The present member that requires it."),
	}, "{field} is required when `{because}` is present.", "Add the member `{property}`, or remove `{because}`.", SpanParent},
	{"additional_property", []ParamSpec{
		always("property", TypeString, meaningProperty),
		optional("suggestion", TypeString, meaningSuggestion),
	}, "The member `{property}` is not one the schema allows.", "Remove `{property}`, or rename it to a member the schema declares.", SpanKey},
	{"unevaluated_property", []ParamSpec{
		always("property", TypeString, meaningProperty),
		optional("suggestion", TypeString, meaningSuggestion),
	}, "The member `{property}` is not one the schema allows.", "Remove `{property}`, or rename it to a member the schema declares.", SpanKey},
	{"property_name", []ParamSpec{
		always("property", TypeString, meaningProperty),
	}, "The member name `{property}` breaks the schema's rule for member names.", "Rename `{property}` to a name the schema's propertyNames allows.", SpanKey},
	{"min_properties", []ParamSpec{
		always("limit", TypeNumber, meaningLimit), always("count", TypeNumber, "How many members the object has."),
	}, "{field} must have at least {limit} members; it has {count}.", "Add members until there are at least {limit}.", SpanValue},
	{"max_properties", []ParamSpec{
		always("limit", TypeNumber, meaningLimit), always("count", TypeNumber, "How many members the object has."),
	}, "{field} must have at most {limit} members; it has {count}.", "Remove members until there are at most {limit}.", SpanValue},
	{"min_length", []ParamSpec{
		always("limit", TypeNumber, meaningLimit), optional("length", TypeNumber, "The string's length in code points, absent when the schema at the value's location is writeOnly."),
	}, "{field} must be at least {limit} characters long.", "Write at least {limit} characters.", SpanValue},
	{"max_length", []ParamSpec{
		always("limit", TypeNumber, meaningLimit), optional("length", TypeNumber, "The string's length in code points, absent when the schema at the value's location is writeOnly."),
	}, "{field} must be at most {limit} characters long.", "Write at most {limit} characters.", SpanValue},
	{"pattern", []ParamSpec{
		always("pattern", TypeString, "The regular expression the schema requires the string to match."),
	}, "{field} does not match the pattern the schema requires.", "Write a value that matches the pattern {pattern}.", SpanValue},
	{"format", []ParamSpec{
		always("format", TypeString, "The format the schema requires."),
	}, "{field} is not in the {format} format.", "Write a value in the {format} format.", SpanValue},
	{"minimum", []ParamSpec{
		always("limit", TypeNumber, meaningLimit), optional("value", TypeNumber, meaningValue),
	}, "{field} must be at least {limit}.", "Write a number of at least {limit}.", SpanValue},
	{"maximum", []ParamSpec{
		always("limit", TypeNumber, meaningLimit), optional("value", TypeNumber, meaningValue),
	}, "{field} must be at most {limit}.", "Write a number of at most {limit}.", SpanValue},
	{"exclusive_minimum", []ParamSpec{
		always("limit", TypeNumber, meaningLimit), optional("value", TypeNumber, meaningValue),
	}, "{field} must be greater than {limit}.", "Write a number greater than {limit}.", SpanValue},
	{"exclusive_maximum", []ParamSpec{
		always("limit", TypeNumber, meaningLimit), optional("value", TypeNumber, meaningValue),
	}, "{field} must be less than {limit}.", "Write a number less than {limit}.", SpanValue},
	{"multiple_of", []ParamSpec{
		always("divisor", TypeNumber, meaningLimit), optional("value", TypeNumber, meaningValue),
	}, "{field} must be a multiple of {divisor}.", "Write a multiple of {divisor}.", SpanValue},
	{"min_items", []ParamSpec{
		always("limit", TypeNumber, meaningLimit), always("count", TypeNumber, "How many items the array has."),
	}, "{field} must have at least {limit} items; it has {count}.", "Add items until there are at least {limit}.", SpanValue},
	{"max_items", []ParamSpec{
		always("limit", TypeNumber, meaningLimit), always("count", TypeNumber, "How many items the array has."),
	}, "{field} must have at most {limit} items; it has {count}.", "Remove items until there are at most {limit}.", SpanValue},
	{"unique_items", []ParamSpec{
		always("index", TypeNumber, "The later item's 0-based index in the array."),
		always("duplicate_of", TypeNumber, "The 0-based index of the earlier item it repeats."),
	}, "{field} repeats item {duplicate_of} of the array, whose items must be unique.", "Remove the repeated item, or change it.", SpanValue},
	{"contains", nil, "{field} has no item the schema's contains accepts.", "Add an item that the contains subschema accepts.", SpanValue},
	{"min_contains", []ParamSpec{
		always("limit", TypeNumber, meaningLimit), always("count", TypeNumber, "How many items the contains subschema accepts."),
	}, "{field} must have at least {limit} items the schema's contains accepts; it has {count}.", "Add items the contains subschema accepts until there are at least {limit}.", SpanValue},
	{"max_contains", []ParamSpec{
		always("limit", TypeNumber, meaningLimit), always("count", TypeNumber, "How many items the contains subschema accepts."),
	}, "{field} must have at most {limit} items the schema's contains accepts; it has {count}.", "Remove items the contains subschema accepts until there are at most {limit}.", SpanValue},
	{"additional_item", []ParamSpec{
		always("index", TypeNumber, meaningIndex),
		always("prefix_count", TypeNumber, "How many items the schema's prefixItems describes."),
	}, "{field} is beyond the {prefix_count} items the schema allows.", "Remove the items after the first {prefix_count}.", SpanValue},
	{"unevaluated_item", []ParamSpec{
		always("index", TypeNumber, meaningIndex),
	}, "{field} is an item the schema does not allow.", "Remove the item.", SpanValue},
	{"forbidden", []ParamSpec{
		optional("property", TypeString, "The member's name, when the schema forbids it under properties."),
	}, "{field} is not allowed here.", "Remove the value.", SpanValue},
	{"not", nil, "{field} matches a schema it must not match.", "Change the value so that the not subschema no longer accepts it.", SpanValue},
	{"any_of", []ParamSpec{
		always("branch_count", TypeNumber, "How many alternatives the schema gives."),
	}, "{field} matches none of the {branch_count} alternatives the schema allows.", "Change the value to match one alternative; each alternative's findings are listed under it.", SpanValue},
	{"one_of", []ParamSpec{
		always("branch_count", TypeNumber, "How many alternatives the schema gives."),
	}, "{field} matches none of the {branch_count} alternatives the schema allows exactly one of.", "Change the value to match one alternative; each alternative's findings are listed under it.", SpanValue},
	{"one_of.ambiguous", []ParamSpec{
		{Name: "branches", Type: TypeArray, Items: TypeNumber, Meaning: "The 0-based indexes of two alternatives the value matches.", Always: true},
	}, "{field} matches alternatives {branches} where the schema allows exactly one.", "Change the value so that only one alternative accepts it.", SpanValue},
}

// schemaEntries are the registry entries of ADR-006 D6 item 4's keyword
// codes in each family, and D7 item 4's partial-validation finding.
// schemaStatements are the statements each keyword code a vector produces
// reports, by code; a code it does not name reports none yet.
var schemaStatements = map[Code][]string{
	"manifest.additional_property": {"MTS-MAN-011", "MTS-MAN-019"},
	"manifest.enum":                {"MTS-MAN-019"},
	"manifest.max_length":          {"MTS-MAN-012", "MTS-MAN-014", "MTS-MAN-019"},
	"manifest.min_items":           {"MTS-MAN-019"},
	"manifest.not":                 {"MTS-MAN-014", "MTS-MAN-019"},
	"manifest.pattern":             {"MTS-MAN-012", "MTS-MAN-014", "MTS-MAN-019", "MTS-VER-008"},
	"manifest.required":            {"MTS-MAN-011", "MTS-MAN-019"},
	"manifest.type":                {"MTS-MAN-011", "MTS-MAN-019", "MTS-VER-008"},
}

func schemaEntries() []Entry {
	var out []Entry
	for _, fam := range schemaFamilies {
		for _, k := range schemaKeywords {
			out = append(out, Entry{
				Code:       Code(fam.prefix + "." + k.suffix),
				Severity:   SeverityError,
				Class:      ClassContent,
				Subject:    fam.subject,
				Params:     slices.Clone(k.params),
				Message:    k.message,
				Hint:       k.hint,
				Span:       k.span,
				Since:      "1.0",
				Statements: append([]string{}, schemaStatements[Code(fam.prefix+"."+k.suffix)]...),
			})
		}
	}
	return append(out, Entry{
		Code:     CodeInputValidationPartial,
		Severity: SeverityInfo,
		Class:    ClassContent,
		Subject:  SubjectInput,
		Params: []ParamSpec{
			always("unparsed", TypeNumber, "How many items of the array did not parse, and were left out of its validation."),
			always("suppressed", TypeNumber, "How many findings were not reported because removing the items changes what they measure: those about the array itself, and those through prefixItems or unevaluatedItems applied to it."),
		},
		Message:    "The input {input} was validated without the {unparsed} items that did not parse, and {suppressed} findings about the array as a whole were left out.",
		Hint:       "Fix the items that did not parse and validate again, which may report more.",
		Span:       SpanValue,
		Since:      "1.0",
		Statements: []string{},
	})
}

// withSchemaEntries returns base with schemaEntries added, in code order.
func withSchemaEntries(base []Entry) []Entry {
	all := append(base, schemaEntries()...)
	slices.SortFunc(all, func(a, b Entry) int { return strings.Compare(string(a.Code), string(b.Code)) })
	return all
}
