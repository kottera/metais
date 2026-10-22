// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package schema

import (
	"fmt"
	"strings"
	"testing"

	"metais.org/metais/internal/testenv"
)

// benchSchema is a collection's schema, as a package declares one: an
// array of items, each an object of every kind of member, reached by a
// $ref, with required, enum, bounds, a format and uniqueItems, and no
// pattern, so that no iteration spends the command's regex-work.
const benchSchema = `{
  "type": "array",
  "items": {"$ref": "#/$defs/item"},
  "$defs": {
    "item": {
      "type": "object",
      "required": ["id", "name", "status", "amount", "when"],
      "additionalProperties": false,
      "properties": {
        "id": {"type": "integer", "minimum": 0},
        "name": {"type": "string", "minLength": 1, "maxLength": 200},
        "status": {"enum": ["open", "held", "closed"]},
        "amount": {"type": "number", "exclusiveMinimum": -1, "multipleOf": 0.01},
        "when": {"type": "string", "format": "date"},
        "tags": {"type": "array", "items": {"type": "string"}, "uniqueItems": true, "maxItems": 8},
        "where": {"$ref": "#/$defs/where"}
      }
    },
    "where": {
      "type": "object",
      "required": ["file"],
      "properties": {"file": {"type": "string"}, "line": {"type": "integer", "minimum": 1}}
    }
  }
}`

// benchCollection is a collection of n items valid against benchSchema.
func benchCollection(n int) string {
	var s strings.Builder
	s.WriteString("[")
	for i := range n {
		if i > 0 {
			s.WriteString(",")
		}
		fmt.Fprintf(&s, `{"id": %d, "name": "Item %d", "status": %q, "amount": %d.%02d, "when": "2026-%02d-%02d", "tags": ["t%d", "u%d"], "where": {"file": "data/items/%05d.json", "line": %d}}`,
			i, i, []string{"open", "held", "closed"}[i%3], i%5000, i%100, i%12+1, i%28+1, i%13, i%17, i, i%40+1)
	}
	s.WriteString("]")
	return s.String()
}

// BenchmarkValidate is ADR-010 D15 item 1's schema validation of a
// 10,000-item collection: the validation alone, the schema compiled and
// the collection decoded beforehand.
func BenchmarkValidate(b *testing.B) {
	testenv.Home(b)
	v := mustValue(b, benchCollection(10_000))
	b.ReportAllocs()
	for b.Loop() {
		// Each validation is a command of its own, with its own
		// schema-work; the compile is not timed.
		b.StopTimer()
		s := mustCompile(b, map[string]string{"inputs/items.schema.json": benchSchema}, "inputs/items.schema.json")
		b.StartTimer()
		if fails, err := s.Validate(v); err != nil || len(fails) > 0 {
			b.Fatalf("the collection is not valid: %v %v", err, fails)
		}
	}
}

// TestBenchCollection holds the benchmark to validating what it means
// to, in every go test: the collection is valid, and an item that breaks
// the schema is caught, so the work timed is a validation that looks.
func TestBenchCollection(t *testing.T) {
	testenv.Home(t)
	s := mustCompile(t, map[string]string{"inputs/items.schema.json": benchSchema}, "inputs/items.schema.json")
	good := benchCollection(10_000)
	if fails, err := s.Validate(mustValue(t, good)); err != nil || len(fails) > 0 {
		t.Fatalf("the collection is not valid: %v %v", err, fails)
	}
	// A second command, with schema-work of its own.
	s = mustCompile(t, map[string]string{"inputs/items.schema.json": benchSchema}, "inputs/items.schema.json")
	bad := strings.Replace(good, `"status": "held"`, `"status": "gone"`, 1)
	if fails, err := s.Validate(mustValue(t, bad)); err != nil || len(fails) == 0 {
		t.Errorf("an item outside the enum is not caught: %v", err)
	}
}
