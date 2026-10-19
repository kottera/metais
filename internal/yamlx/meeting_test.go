// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package yamlx

import (
	"slices"
	"strconv"
	"testing"

	"github.com/kottera/metais/internal/finding"
	"github.com/kottera/metais/internal/jsonx"
	"github.com/kottera/metais/internal/limits"
	"github.com/kottera/metais/internal/liquid/value"
	"github.com/kottera/metais/internal/testenv"
)

// TestSameTreeAsJSON is where yamlx meets jsonx (ADR-010 D7 item 1): the
// same data, written in JSON and in the YAML profile, gives the same value
// tree, numbers keeping their literal text, objects and arrays nested
// alike, and members in the same order.
func TestSameTreeAsJSON(t *testing.T) {
	testenv.Home(t)
	const json = `{"z": 1.50, "a": [1, -0, 2.5e-3, 12345678901234567890.0, {"y": null, "b": [true, false, "s", []]}],
	 "é": {"k": "v", "e": {}}, "😀": 1E2, "｡": -7, "": 0}`
	if sameText(arr(num("1.50")), arr(num("1.5")), "") == "" {
		t.Fatal("sameText does not see a literal's text")
	}
	for name, yaml := range map[string]string{
		"block": "z: 1.50\na:\n  - 1\n  - -0\n  - 2.5e-3\n  - 12345678901234567890.0\n  - y: null\n    b: [true, false, s, []]\n" +
			"é:\n  k: v\n  e: {}\n\"\U0001F600\": 1E2\n｡: -7\n\"\": 0\n",
		"the JSON text itself": json,
	} {
		jdoc, fs := jsonx.Parse([]byte(json), jsonx.Source{Input: inputSrc.Subject, At: inputSrc.At}, limits.Defaults())
		if fs != nil {
			t.Fatalf("jsonx: %v", fs)
		}
		ydoc, fs := parse(t, inputSrc, yaml)
		if fs != nil {
			t.Fatalf("%s: %s", name, describe(fs))
		}
		if !value.Equal(jdoc.Value, ydoc.Value) {
			t.Errorf("%s: json %#v\nyaml %#v", name, jdoc.Value, ydoc.Value)
		}
		if err := sameText(jdoc.Value, ydoc.Value, ""); err != "" {
			t.Errorf("%s: %s", name, err)
		}
	}
}

// sameText reports where a and b, equal values, differ in a number's
// literal text or an object's member order.
func sameText(a, b value.Value, ptr string) string {
	switch a := a.(type) {
	case value.Number:
		if a.Literal() != b.(value.Number).Literal() {
			return ptr + ": literal " + a.Literal() + " and " + b.(value.Number).Literal()
		}
	case value.Array:
		for i, e := range a.All() {
			if d := sameText(e, b.(value.Array).At(i), ptr+finding.Pointer(strconv.Itoa(i))); d != "" {
				return d
			}
		}
	case value.Object:
		bo := b.(value.Object)
		if !slices.Equal(a.Names(), bo.Names()) {
			return ptr + ": member order"
		}
		for n, m := range a.All() {
			bm, _ := bo.Get(n)
			if d := sameText(m, bm, ptr+finding.Pointer(n)); d != "" {
				return d
			}
		}
	}
	return ""
}
