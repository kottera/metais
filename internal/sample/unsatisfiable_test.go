// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package sample

import (
	"slices"
	"strings"
	"testing"

	"github.com/kottera/metais/internal/finding"
)

// TestUnsatisfiable is W3-5.2: each reason of schema.sample_unsatisfiable
// (ADR-007 D9 items 11 and 23), from a schema no generated value can
// satisfy, named at the subschema that cannot be satisfied.
func TestUnsatisfiable(t *testing.T) {
	for _, tc := range []struct {
		reason, schema string
		pointer        string // the subschema the finding names
	}{
		{ReasonType, `{"type":"string","allOf":[{"type":"integer"}]}`, ""},
		{ReasonType, `false`, ""},
		{ReasonType, `{"type":"object","required":["a"],"properties":{"a":false}}`, ""},
		{ReasonRange, `{"type":"integer","minimum":10,"maximum":5}`, ""},
		{ReasonRange, `{"type":"number","exclusiveMinimum":1,"exclusiveMaximum":1}`, ""},
		{ReasonRange, `{"type":"integer","exclusiveMinimum":1,"exclusiveMaximum":2}`, ""},
		{ReasonRange, `{"type":"integer","minimum":1e17}`, ""},
		{ReasonLength, `{"type":"string","minLength":5,"maxLength":2}`, ""},
		{ReasonLength, `{"type":"array","minItems":3,"maxItems":1}`, ""},
		{ReasonLength, `{"type":"array","prefixItems":[{}],"items":false,"minItems":2}`, ""},
		{ReasonLength, `{"type":"object","minProperties":2,"additionalProperties":false,"properties":{"a":{}}}`, ""},
		{ReasonLength, `{"type":"object","maxProperties":1,"required":["a","b"]}`, ""},
		{ReasonEnum, `{"enum":["a","b"],"allOf":[{"enum":["c"]}]}`, ""},
		{ReasonEnum, `{"type":"string","enum":[1,2]}`, ""},
		{ReasonConst, `{"const":1,"allOf":[{"const":2}]}`, ""},
		{ReasonConst, `{"type":"string","const":"abc","maxLength":2}`, ""},
		{ReasonConst, `{"const":1.2345678901234567e19}`, ""},
		{ReasonMultipleOf, `{"type":"integer","multipleOf":7,"minimum":1,"maximum":6}`, ""},
		{ReasonMultipleOf, `{"type":"number","multipleOf":0.3,"minimum":0.1,"maximum":0.2}`, ""},
		{ReasonPatternUnsupported, `{"type":"string","pattern":"\\bword\\b"}`, ""},
		{ReasonPatternUnmet, `{"type":"string","pattern":"^[a-z]{5}$","maxLength":3}`, ""},
		{ReasonPatternUnmet, `{"type":"string","pattern":"^a","allOf":[{"pattern":"^b"}]}`, ""},
		{ReasonFormat, `{"type":"string","format":"date","maxLength":5}`, ""},
		{ReasonFormat, `{"type":"string","format":"email","pattern":"^[0-9]+$"}`, ""},
		{ReasonUniqueItems, `{"type":"array","uniqueItems":true,"minItems":3,"items":{"enum":["a","b"]}}`, ""},
		{ReasonNot, `{"type":"integer","not":{}}`, ""},
		{ReasonNot, `{"type":"integer","minimum":1,"maximum":1,"not":{"const":1}}`, ""},
		{ReasonConditionals, `{"type":"object","required":["a"],"properties":{"a":{"type":"integer"}},
			"if":{"properties":{"a":{"minimum":50}}},"then":{"properties":{"a":{"maximum":10}}},"else":{"properties":{"a":{"minimum":60}}}}`, ""},
		// A scalar is made again with each branch it fails added to C, so
		// branches that contradict each other are a contradiction.
		{ReasonRange, `{"type":"integer","if":{"minimum":0},"then":{"maximum":-1},"else":{"minimum":0}}`, "/else"},
		{ReasonRecursion, `{"$ref":"#/$defs/n","$defs":{"n":{"type":"object","required":["next"],"properties":{"next":{"$ref":"#/$defs/n"}}}}}`, "/$defs/n"},
	} {
		_, fs := one(t, tc.schema, Options{Items: 3})
		got := reasons(t, fs)
		if !slices.Contains(got, tc.reason) {
			t.Errorf("%s: reasons %v, want %s%s", tc.schema, got, tc.reason, describe(fs))
			continue
		}
		f := fs[slices.Index(got, tc.reason)]
		loc, _ := f.Location()
		if loc.File != entry || !loc.HasPointer || !strings.HasPrefix(loc.Pointer, tc.pointer) {
			t.Errorf("%s: located at %s#%s", tc.schema, loc.File, loc.Pointer)
		}
	}
}

// TestUnsatisfiableFinding is item 23's shape: an error about the input,
// located at the subschema in its schema file with the line and column
// the file's position table gives, naming the instance location and the
// reason, with the hint to add an examples value.
func TestUnsatisfiableFinding(t *testing.T) {
	_, fs := one(t, `{
  "type": "object",
  "properties": {
    "count": {"type": "integer", "minimum": 10, "maximum": 1}
  }
}`, Options{})
	if len(fs) != 1 {
		t.Fatalf("findings:%s", describe(fs))
	}
	f := fs[0]
	if f.Code() != "schema.sample_unsatisfiable" || f.Severity() != finding.SeverityError || f.Class() != finding.ClassContent {
		t.Errorf("%s %v %v", f.Code(), f.Severity(), f.Class())
	}
	if s := f.Subject(); s.Kind != finding.SubjectInput || s.Name != "data" {
		t.Errorf("subject %+v", s)
	}
	loc, _ := f.Location()
	if loc != (finding.Location{Origin: finding.OriginPackage, File: entry, Line: 4, Column: 14, Pointer: "/properties/count", HasPointer: true}) {
		t.Errorf("location %+v", loc)
	}
	inst, _ := f.Param("instance")
	if s, _ := inst.Str(); s != "/count" {
		t.Errorf("instance %q", s)
	}
	if !strings.Contains(f.Hint(), `"examples"`) || !strings.Contains(f.Message(), "data") {
		t.Errorf("message %q, hint %q", f.Message(), f.Hint())
	}
}

// TestEveryUnsatisfiableField is item 23's last sentence: every
// unsatisfiable field is reported in one run.
func TestEveryUnsatisfiableField(t *testing.T) {
	_, fs := one(t, `{"type":"object","properties":{
		"a":{"type":"integer","minimum":3,"maximum":1},
		"b":{"type":"string","pattern":"\\b"},
		"ok":{"type":"string"},
		"list":{"type":"array","items":{"type":"string","minLength":4,"maxLength":1}}}}`, Options{Items: 2})
	var got []string
	for _, f := range fs {
		inst, _ := f.Param("instance")
		s, _ := inst.Str()
		got = append(got, s)
	}
	slices.Sort(got)
	if want := []string{"/a", "/b", "/list/0", "/list/1"}; !slices.Equal(got, want) {
		t.Errorf("instances %v, want %v%s", got, want, describe(fs))
	}
}
