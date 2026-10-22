// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package schemafail

import (
	"strings"

	"metais.org/metais/internal/finding"
)

// argKind is what follows a keyword that holds subschemas, in a schema
// pointer or an evaluation path.
type argKind uint8

const (
	argSchema  argKind = iota + 1 // the subschema itself: items, not, $ref's target
	argNamed                      // a member name, then the subschema: properties
	argIndexed                    // an index, then the subschema: allOf
)

type applicator struct {
	arg argKind
	// descends is whether the subschema applies at a location beneath the
	// instance the keyword applies at.
	descends bool
}

// applicators are the keywords of 2020-12 whose values hold subschemas,
// with $defs, which holds the targets of references. In an evaluation path
// $ref and $dynamicRef are followed by the path within their target.
var applicators = map[string]applicator{
	"$ref":                  {argSchema, false},
	"$dynamicRef":           {argSchema, false},
	"$defs":                 {argNamed, false},
	"allOf":                 {argIndexed, false},
	"anyOf":                 {argIndexed, false},
	"oneOf":                 {argIndexed, false},
	"not":                   {argSchema, false},
	"if":                    {argSchema, false},
	"then":                  {argSchema, false},
	"else":                  {argSchema, false},
	"dependentSchemas":      {argNamed, false},
	"properties":            {argNamed, true},
	"patternProperties":     {argNamed, true},
	"additionalProperties":  {argSchema, true},
	"unevaluatedProperties": {argSchema, true},
	"propertyNames":         {argSchema, true},
	"prefixItems":           {argIndexed, true},
	"items":                 {argSchema, true},
	"unevaluatedItems":      {argSchema, true},
	"contains":              {argSchema, true},
	"contentSchema":         {argSchema, true},
}

// step is one applicator a path passes through: tokens[:end] is the
// subschema it reaches.
type step struct {
	keyword  string
	end      int
	descends bool
}

// steps parses tokens, a path from a schema object, into the applicators
// it passes through. rest is the index of the first token after the last
// subschema reached: len(tokens) when tokens name a subschema, otherwise
// the keyword that follows it.
func steps(tokens []string) (ss []step, rest int) {
	i := 0
	for i < len(tokens) {
		a, ok := applicators[tokens[i]]
		if !ok {
			return ss, i
		}
		end := i + 1
		if a.arg != argSchema {
			end = i + 2
		}
		if end > len(tokens) {
			return ss, i
		}
		ss = append(ss, step{tokens[i], end, a.descends})
		i = end
	}
	return ss, i
}

// tokens splits ptr, a JSON Pointer, into its decoded reference tokens.
func tokens(ptr string) []string {
	if ptr == "" {
		return nil
	}
	parts := strings.Split(ptr[1:], "/")
	for i, p := range parts {
		parts[i] = strings.ReplaceAll(strings.ReplaceAll(p, "~1", "/"), "~0", "~")
	}
	return parts
}

// pointer is the JSON Pointer made of toks.
func pointer(toks []string) string { return finding.Pointer(toks...) }
