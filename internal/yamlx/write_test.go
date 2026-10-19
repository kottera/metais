// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package yamlx

import (
	"bytes"
	"flag"
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/kottera/metais/internal/jsonx"
	"github.com/kottera/metais/internal/liquid/value"
	"github.com/kottera/metais/internal/testenv"
)

var update = flag.Bool("update", false, "rewrite testdata/write's golden files")

// TestWrite is W2-7.6: the `yaml` writer's bytes, rule by rule of ADR-004
// D10 item 2.
func TestWrite(t *testing.T) {
	testenv.Home(t)
	cases := []struct {
		name string
		v    value.Value
		want string
	}{
		{"null", value.Null{}, "null"},
		{"true", value.Bool(true), "true"},
		{"false", value.Bool(false), "false"},
		{"an empty mapping", obj(), "{}"},
		{"an empty sequence", arr(), "[]"},
		{"a mapping", obj("b", num("2"), "a", str("x")), "a: x\nb: 2"},
		{"members in canonical order", obj("\uFF61", num("1"), "\U0001F600", num("2"), "B", num("3"), "a", num("4")), "B: 3\na: 4\n\"\U0001F600\": 2\n\"\uFF61\": 1"},
		{"a nested mapping", obj("a", obj("b", obj("c", num("1")))), "a:\n  b:\n    c: 1"},
		{"a sequence in a mapping", obj("a", arr(num("1"), num("2"))), "a:\n  - 1\n  - 2"},
		{"empty collections in a mapping", obj("a", obj(), "b", arr()), "a: {}\nb: []"},
		{"a mapping in a sequence", arr(obj("a", num("1"), "b", num("2")), num("3")), "- a: 1\n  b: 2\n- 3"},
		{"a sequence in a sequence", arr(arr(num("1"), num("2")), num("3")), "- - 1\n  - 2\n- 3"},
		{"deep nesting", arr(obj("a", arr(obj("b", arr(num("1")))))), "- a:\n    - b:\n        - 1"},
		{"empty collections in a sequence", arr(obj(), arr(), value.Null{}), "- {}\n- []\n- null"},
		// numbers as the json filter writes them
		{"an integer", num("12"), "12"},
		{"an integer written with a fraction", num("3.0"), "3"},
		{"a trailing zero", num("4.50"), "4.5"},
		{"minus zero", num("-0"), "0"},
		{"a negative", num("-1.5"), "-1.5"},
		{"an exponent", num("1E30"), "1e+30"},
		{"1e21", num("1e21"), "1e+21"},
		{"just below 1e21", num("999999999999999900000"), "999999999999999900000"},
		{"a small number", num("0.000001"), "0.000001"},
		{"a smaller number", num("0.0000001"), "1e-7"},
		{"a small mantissa", num("1.5e-7"), "1.5e-7"},
		{"the largest double", num("1.7976931348623157e308"), "1.7976931348623157e+308"},
		{"the smallest denormal", num("5e-324"), "5e-324"},
		{"2^53", num("9007199254740992"), "9007199254740992"},
		{"a long fraction", num("0.1"), "0.1"},
		{"a third", num("0.3333333333333333"), "0.3333333333333333"},
		{"123.456", num("123.456"), "123.456"},
		// plain strings and names
		{"a word", str("hello"), "hello"},
		{"words", str("Status widget"), "Status widget"},
		{"a path", str("/usr/local_1/x.y-z"), "/usr/local_1/x.y-z"},
		{"an underscore", str("_"), "_"},
		{"one letter", str("x"), "x"},
		{"a name", obj("first name", str("Ada")), "first name: Ada"},
		// quoted strings and names
		{"the empty string", str(""), `""`},
		{"a leading digit", str("1.0"), `"1.0"`},
		{"a date", str("2026-10-18"), `"2026-10-18"`},
		{"a leading space", str(" a"), `" a"`},
		{"a trailing space", str("a "), `"a "`},
		{"a trailing hyphen", str("a-"), `"a-"`},
		{"a colon", str("a: b"), `"a: b"`},
		{"a hash", str("a #b"), `"a #b"`},
		{"a leading hyphen", str("- a"), `"- a"`},
		{"a leading dot", str(".inf"), `".inf"`},
		{"a tilde", str("~"), `"~"`},
		{"non-ASCII", str("é"), `"é"`},
		{"true", str("true"), `"true"`},
		{"True", str("True"), `"True"`},
		{"FALSE", str("FALSE"), `"FALSE"`},
		{"yes", str("yes"), `"yes"`},
		{"No", str("No"), `"No"`},
		{"on", str("on"), `"on"`},
		{"OFF", str("OFF"), `"OFF"`},
		{"y", str("y"), `"y"`},
		{"N", str("N"), `"N"`},
		{"Null", str("Null"), `"Null"`},
		{"not a reserved word", str("nulls"), "nulls"},
		{"a quoted name", obj("<<", num("1"), "", num("2")), "\"\": 2\n\"<<\": 1"},
		// escapes
		{"a quote and a backslash", str(`say "a\b"`), `"say \"a\\b\""`},
		{"the short escapes", str("\b\t\n\f\r"), `"\b\t\n\f\r"`},
		{"other C0 controls", str("\x00\x01\x1f"), `"\u0000\u0001\u001f"`},
		{"DEL and C1", str("\x7f\u0080\u0085\u009f\u00a0"), "\"\\u007f\\u0080\\u0085\\u009f\u00a0\""},
		{"LS, PS and BOM", str("\u2028\u2029\ufeff"), `"\u2028\u2029\ufeff"`},
		{"other characters as they are", str("\u00e9\u2027\u202a\U0001F600/"), "\"\u00e9\u2027\u202a\U0001F600/\""},
	}
	for _, tc := range cases {
		if got := string(Write(tc.v)); got != tc.want {
			t.Errorf("%s: got\n%s\nwant\n%s", tc.name, got, tc.want)
		}
	}
}

// TestWriteGolden pins a whole document's bytes against
// testdata/write/document.yaml.
func TestWriteGolden(t *testing.T) {
	testenv.Home(t)
	in, err := os.ReadFile(filepath.Join("testdata", "write", "document.in.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	doc, fs := parse(t, inputSrc, string(in))
	if fs != nil {
		t.Fatal(describe(fs))
	}
	got := Write(doc.Value)
	golden := filepath.Join("testdata", "write", "document.yaml")
	if *update {
		if err := os.WriteFile(golden, got, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(golden)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Errorf("got\n%s\nwant\n%s", got, want)
	}
	again, fs := parse(t, inputSrc, string(got))
	if fs != nil || !value.Equal(again.Value, doc.Value) {
		t.Errorf("the written document does not read back as the value: %s", describe(fs))
	}
}

// TestWriteReadsBack: what the writer writes, the profile reads as the
// same value.
func TestWriteReadsBack(t *testing.T) {
	testenv.Home(t)
	for _, v := range []value.Value{
		obj("a", arr(obj(), arr(), value.Null{}, value.Bool(false), num("1e-7"), num("1e300")), "b", str("multi\nline\ttext \"q\" \\"), "c c", obj("x", arr(arr(arr(str("deep")))))),
		arr(str("true"), str("012"), str("~"), str("- x"), str("a: b"), str("#"), str("\u2028"), str("\u0085")),
		str("\U0001F600 \u00e9"),
		num("-0.5"),
	} {
		text := Write(v)
		doc, fs := parse(t, inputSrc, string(text))
		if fs != nil {
			t.Errorf("%s: %s", text, describe(fs))
			continue
		}
		if !value.Equal(doc.Value, v) {
			t.Errorf("%s: read back as %#v", text, doc.Value)
		}
	}
}

// TestNumbersAsJSONWrites is where the yaml writer meets the json one
// (ADR-004 D10 item 2, "numbers as in json"): every number, over the
// boundaries of ECMAScript's notations, is written by both alike.
func TestNumbersAsJSONWrites(t *testing.T) {
	for _, tc := range []struct {
		f    float64
		want string
	}{
		{1e20, "100000000000000000000"},
		{1e21, "1e+21"},
		{123e18, "123000000000000000000"},
		{1.5e21, "1.5e+21"},
		{1e-6, "0.000001"},
		{1.25e-6, "0.00000125"},
		{1e-7, "1e-7"},
		{-1e-7, "-1e-7"},
		{math.MaxFloat64, "1.7976931348623157e+308"},
		{math.SmallestNonzeroFloat64, "5e-324"},
		{math.Nextafter(0.3, 1), "0.30000000000000004"},
		{math.Copysign(0, -1), "0"},
		{100, "100"},
		{1.5, "1.5"},
	} {
		n, err := value.NumberFromFloat(tc.f)
		if err != nil {
			t.Fatal(err)
		}
		j, err := jsonx.Encode(n, 0)
		if err != nil {
			t.Fatal(err)
		}
		if got := string(Write(n)); got != tc.want || got != string(j) {
			t.Errorf("%v: yaml %s, json %s, want %s", tc.f, got, j, tc.want)
		}
	}
}
