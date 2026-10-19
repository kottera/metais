// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package schema

import (
	"testing"

	"github.com/kottera/metais/internal/limits"
	"github.com/kottera/metais/internal/testenv"
)

// subFiles is a package whose schemas the tests below read in pieces.
var subFiles = map[string]string{
	"inputs/a.json": `{"$id":"https://example.com/a","type":"object","properties":{
		"n":{"type":"integer","minimum":3},
		"odd name/~x":{"type":"string","maxLength":2},
		"r":{"$ref":"b.json#/$defs/s"},
		"d":{"$dynamicRef":"#node"}},
		"$defs":{"node":{"$dynamicAnchor":"node","type":"boolean"}}}`,
	"inputs/b.json": `{"$id":"https://example.com/b.json","$defs":{"s":{"type":"string","minLength":2}}}`,
}

// TestLookup: Lookup gives a schema file's value at a pointer as
// authored, and nothing where there is none.
func TestLookup(t *testing.T) {
	testenv.Home(t)
	c, fs := compileFiles(t, limits.Defaults(), subFiles)
	if len(fs) > 0 {
		t.Fatal(describe(fs))
	}
	if v, ok := c.Lookup("inputs/a.json", "/properties/n/minimum"); !ok || canonical(v) != "3" {
		t.Errorf("minimum: %v %v", v, ok)
	}
	for _, at := range [][2]string{{"inputs/a.json", "/properties/none"}, {"inputs/c.json", ""}, {"inputs/a.json", "/required/0"}} {
		if v, ok := c.Lookup(at[0], at[1]); ok {
			t.Errorf("%s#%s: %v", at[0], at[1], v)
		}
	}
}

// TestDynamicRef: DynamicRef places a $dynamicRef's target as written,
// as Ref places a $ref's.
func TestDynamicRef(t *testing.T) {
	testenv.Home(t)
	c, fs := compileFiles(t, limits.Defaults(), subFiles)
	if len(fs) > 0 {
		t.Fatal(describe(fs))
	}
	if e, p, ok := c.DynamicRef("inputs/a.json", "/properties/d"); !ok || e != "inputs/a.json" || p != "/$defs/node" {
		t.Errorf("$dynamicRef: %s %s %v", e, p, ok)
	}
	if e, p, ok := c.Ref("inputs/a.json", "/properties/r"); !ok || e != "inputs/b.json" || p != "/$defs/s" {
		t.Errorf("$ref: %s %s %v", e, p, ok)
	}
	if _, _, ok := c.DynamicRef("inputs/a.json", "/properties/r"); ok {
		t.Error("a $ref is a $dynamicRef")
	}
}

// TestSubschema: a subschema compiled alone validates by its own
// keywords, its references resolved where its file resolves them, and a
// pointer whose tokens need escaping reaches its schema.
func TestSubschema(t *testing.T) {
	testenv.Home(t)
	c, fs := compileFiles(t, limits.Defaults(), subFiles)
	if len(fs) > 0 {
		t.Fatal(describe(fs))
	}
	for _, tc := range []struct {
		ptr, value string
		valid      bool
	}{
		{"/properties/n", `5`, true},
		{"/properties/n", `2`, false},
		{"/properties/n", `"x"`, false},
		{"/properties/odd name~1~0x", `"ab"`, true},
		{"/properties/odd name~1~0x", `"abc"`, false},
		{"/properties/r", `"ok"`, true},
		{"/properties/r", `"k"`, false},
		{"/properties/d", `true`, true},
		{"", `{"n":4,"r":"zz"}`, true},
		{"", `{"n":1}`, false},
	} {
		s, err := c.Subschema("inputs/a.json", tc.ptr)
		if err != nil {
			t.Fatalf("%s: %v", tc.ptr, err)
		}
		fails, err := s.Validate(mustValue(t, tc.value))
		if err != nil || (len(fails) == 0) != tc.valid {
			t.Errorf("%s with %s: %d failures, %v; want valid %v", tc.ptr, tc.value, len(fails), err, tc.valid)
		}
		if again, _ := c.Subschema("inputs/a.json", tc.ptr); again != s {
			t.Errorf("%s was compiled twice", tc.ptr)
		}
	}
	if _, err := c.Subschema("inputs/none.json", ""); err == nil {
		t.Error("a file that is not the package's compiled")
	}
	if _, err := c.Subschema("inputs/a.json", "/properties/none"); err == nil {
		t.Error("a pointer to nothing compiled")
	}
}
