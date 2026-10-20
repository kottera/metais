// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package jsonx

import (
	"crypto/sha256"
	"errors"
	"math"
	"strconv"
	"strings"
	"testing"

	"github.com/gowebpki/jcs"

	"github.com/kottera/metais/internal/liquid/value"
)

// The examples of RFC 8785, transcribed from the RFC: §3.2.2's
// serialization of primitive data types, and §3.2.3's sorting of
// properties. gowebpki/jcs carries the same two texts as
// testdata/{input,output}/values.json and weird.json.
const (
	rfcValuesIn = `{
  "numbers": [333333333.33333329, 1E30, 4.50,
              2e-3, 0.000000000000000000000000001],
  "string": "\u20ac$\u000F\u000aA'\u0042\u0022\u005c\\\"\/",
  "literals": [null, true, false]
}`
	rfcValuesOut = `{"literals":[null,true,false],"numbers":[333333333.3333333,1e+30,4.5,0.002,1e-27],"string":"€$\u000f\nA'B\"\\\\\"/"}`

	rfcSortingIn = `{
  "\u20ac": "Euro Sign",
  "\r": "Carriage Return",
  "\ufb33": "Hebrew Letter Dalet With Dagesh",
  "1": "One",
  "\ud83d\ude00": "Emoji: Grinning Face",
  "\u0080": "Control",
  "\u00f6": "Latin Small Letter O With Diaeresis"
}`
	rfcSortingOut = "{\"\\r\":\"Carriage Return\",\"1\":\"One\",\"\u0080\":\"Control\",\"\u00f6\":\"Latin Small Letter O With Diaeresis\",\"\u20ac\":\"Euro Sign\",\"\U0001f600\":\"Emoji: Grinning Face\",\"\ufb33\":\"Hebrew Letter Dalet With Dagesh\"}"
)

// rfcNumbers is RFC 8785 Appendix B's table of IEEE 754 doubles and their
// serializations, less its NaN and Infinity rows, which no JSON value
// holds.
var rfcNumbers = []struct {
	bits uint64
	want string
}{
	{0x0000000000000000, "0"},
	{0x8000000000000000, "0"},
	{0x0000000000000001, "5e-324"},
	{0x8000000000000001, "-5e-324"},
	{0x7fefffffffffffff, "1.7976931348623157e+308"},
	{0xffefffffffffffff, "-1.7976931348623157e+308"},
	{0x4340000000000000, "9007199254740992"},
	{0xc340000000000000, "-9007199254740992"},
	{0x4430000000000000, "295147905179352830000"},
	{0x44b52d02c7e14af5, "9.999999999999997e+22"},
	{0x44b52d02c7e14af6, "1e+23"},
	{0x44b52d02c7e14af7, "1.0000000000000001e+23"},
	{0x444b1ae4d6e2ef4e, "999999999999999700000"},
	{0x444b1ae4d6e2ef4f, "999999999999999900000"},
	{0x444b1ae4d6e2ef50, "1e+21"},
	{0x3eb0c6f7a0b5ed8c, "9.999999999999997e-7"},
	{0x3eb0c6f7a0b5ed8d, "0.000001"},
	{0x41b3de4355555553, "333333333.3333332"},
	{0x41b3de4355555554, "333333333.33333325"},
	{0x41b3de4355555555, "333333333.3333333"},
	{0x41b3de4355555556, "333333333.3333334"},
	{0x41b3de4355555557, "333333333.33333343"},
	{0xbecbf647612f3696, "-0.0000033333333333333333"},
	{0x43143ff3c1cb0959, "1424953923781206.2"},
}

// TestCanonicalRFC8785 is W2-6.3: Canonical reproduces RFC 8785's
// examples byte for byte, and so does Encode without an indent.
func TestCanonicalRFC8785(t *testing.T) {
	for _, c := range []struct{ name, in, out string }{
		{"§3.2.2", rfcValuesIn, rfcValuesOut},
		{"§3.2.3", rfcSortingIn, rfcSortingOut},
	} {
		v := mustParse(t, c.in).Value
		if got := string(Canonical(v)); got != c.out {
			t.Errorf("%s: Canonical\ngot  %s\nwant %s", c.name, got, c.out)
		}
		got, err := Encode(v, 0)
		if err != nil || string(got) != c.out {
			t.Errorf("%s: Encode\ngot  %s (%v)\nwant %s", c.name, got, err, c.out)
		}
	}
	for _, c := range rfcNumbers {
		f := math.Float64frombits(c.bits)
		n, err := value.NumberFromFloat(f)
		if err != nil {
			t.Fatal(err)
		}
		if got := string(Canonical(n)); got != c.want {
			t.Errorf("%016x: Canonical %s, want %s", c.bits, got, c.want)
		}
		if got := FormatNumber(f); got != c.want {
			t.Errorf("%016x: FormatNumber %s, want %s", c.bits, got, c.want)
		}
	}
}

// TestCanonicalUsesTheLiteral: Canonical serializes a parsed number from
// its value, whatever literal wrote it, and the literal survives.
func TestCanonicalUsesTheLiteral(t *testing.T) {
	v := mustParse(t, `[1.0, 1E3, -0, 0.10e1, 1e-400, 123456789012345678901234567890.0]`).Value
	const want = `[1,1000,0,1,0,1.2345678901234568e+29]`
	if got := string(Canonical(v)); got != want {
		t.Errorf("Canonical %s, want %s", got, want)
	}
	if lit := v.(value.Array).At(1).(value.Number).Literal(); lit != "1E3" {
		t.Errorf("the literal became %q", lit)
	}
}

// TestFormatNumberAgreesWithJCS is a differential: METAIS's own
// ECMAScript number text, which the json filter writes, against
// gowebpki/jcs's, over doubles spread across every exponent.
func TestFormatNumberAgreesWithJCS(t *testing.T) {
	var bits uint64 = 1
	for i := 0; i < 200_000; i++ {
		bits = bits*6364136223846793005 + 1442695040888963407
		f := math.Float64frombits(bits)
		if math.IsInf(f, 0) || math.IsNaN(f) {
			continue
		}
		want, err := jcs.NumberToJSON(f)
		if err != nil {
			t.Fatalf("%v: %v", f, err)
		}
		got := FormatNumber(f)
		if got != want {
			t.Fatalf("%016x: FormatNumber %s, jcs %s", bits, got, want)
		}
		if back, err := strconv.ParseFloat(got, 64); err != nil || back != f {
			t.Fatalf("%016x: %s does not round-trip: %v %v", bits, got, back, err)
		}
	}
	for _, c := range []struct {
		f    float64
		want string
	}{
		{3.0, "3"}, {-1.5, "-1.5"}, {1e21, "1e+21"}, {1e20, "100000000000000000000"},
		{1e-6, "0.000001"}, {1e-7, "1e-7"}, {123e-20, "1.23e-18"}, {0.1, "0.1"}, {math.Copysign(0, -1), "0"},
	} {
		if got := FormatNumber(c.f); got != c.want {
			t.Errorf("FormatNumber(%v) = %s, want %s", c.f, got, c.want)
		}
	}
}

// TestEncodeGolden is W2-6.5: the json filter's bytes, with and without
// an indent (ADR-004 D10 item 1).
func TestEncodeGolden(t *testing.T) {
	v := mustParse(t, `{"b": [1, {"c": [], "d": "\u0001\t\"é"}, 2.50], "a": {}, "e": null, "f": [true]}`).Value
	cases := []struct {
		indent int
		want   string
	}{
		{0, `{"a":{},"b":[1,{"c":[],"d":"\u0001\t\"é"},2.5],"e":null,"f":[true]}`},
		{2, `{
  "a": {},
  "b": [
    1,
    {
      "c": [],
      "d": "\u0001\t\"é"
    },
    2.5
  ],
  "e": null,
  "f": [
    true
  ]
}`},
		{1, "{\n \"a\": {},\n \"b\": [\n  1,\n  {\n   \"c\": [],\n   \"d\": \"\\u0001\\t\\\"é\"\n  },\n  2.5\n ],\n \"e\": null,\n \"f\": [\n  true\n ]\n}"},
	}
	for _, c := range cases {
		got, err := Encode(v, c.indent)
		if err != nil || string(got) != c.want {
			t.Errorf("Encode(indent %d)\ngot  %s (%v)\nwant %s", c.indent, got, err, c.want)
		}
	}
	deep := mustParse(t, `[[["x"]]]`).Value
	got, _ := Encode(deep, 8)
	const want8 = "[\n        [\n                [\n                        \"x\"\n                ]\n        ]\n]"
	if string(got) != want8 {
		t.Errorf("Encode(indent 8)\ngot  %q\nwant %q", got, want8)
	}
	for _, s := range []string{`"s"`, `12`, `null`, `[]`, `{}`} {
		got, _ := Encode(mustParse(t, s).Value, 4)
		if string(got) != s {
			t.Errorf("Encode(%s, 4) = %s", s, got)
		}
	}
}

// TestEncodeStringEscapes pins RFC 8785 §3.2.2.2's escapes: ", \ and
// U+0000 to U+001F only, with the five short forms and lowercase hex, and
// every other code point as itself.
func TestEncodeStringEscapes(t *testing.T) {
	var in []byte
	for c := 0; c < 0x20; c++ {
		in = append(in, byte(c))
	}
	in = append(in, "\"\\/\u007f\u0080\u2028\u2029\ufeff<>&😀"...)
	s, err := value.NewString(string(in))
	if err != nil {
		t.Fatal(err)
	}
	const want = `"\u0000\u0001\u0002\u0003\u0004\u0005\u0006\u0007\b\t\n\u000b\f\r\u000e\u000f` +
		`\u0010\u0011\u0012\u0013\u0014\u0015\u0016\u0017\u0018\u0019\u001a\u001b\u001c\u001d\u001e\u001f` +
		"\\\"\\\\/\u007f\u0080\u2028\u2029\ufeff<>&😀\""
	got, _ := Encode(s, 0)
	if string(got) != want {
		t.Errorf("Encode\ngot  %q\nwant %q", got, want)
	}
	if c := string(Canonical(s)); c != want {
		t.Errorf("Canonical\ngot  %q\nwant %q", c, want)
	}
}

func TestEncodeRefusesIndent(t *testing.T) {
	for _, n := range []int{-1, 9, 100} {
		if _, err := Encode(value.Null{}, n); !errors.Is(err, ErrIndent) {
			t.Errorf("Encode(indent %d): %v, want ErrIndent", n, err)
		}
	}
}

// TestEncodeInput: the writer of a JSON input file writes what Encode
// writes, except that an integral number beyond 2^53−1 and below 1e21 is
// spelled with an exponent, and every number it writes reads back as
// itself with no finding. Encode's text for the same value is unchanged.
func TestEncodeInput(t *testing.T) {
	v := mustParse(t, `{"a":[11e17,-11e17,9007199254740991,-9007199254740991,9007199254740992e0,1e20,1e21,2.5,1e-7,0],`+
		`"m":[1.7976931348623157e308,-1.7976931348623157e308]}`).Value
	const want = `{"a":[1.1e+18,-1.1e+18,9007199254740991,-9007199254740991,9.007199254740992e+15,1e+20,1e+21,2.5,1e-7,0],` +
		`"m":[1.7976931348623157e+308,-1.7976931348623157e+308]}`
	got, err := EncodeInput(v, 0)
	if err != nil || string(got) != want {
		t.Fatalf("EncodeInput\ngot  %s (%v)\nwant %s", got, err, want)
	}
	if back := mustParse(t, string(got)).Value; !value.Equal(back, v) {
		t.Errorf("%s reads back as %v", got, back)
	}
	const filter = `{"a":[1100000000000000000,-1100000000000000000,9007199254740991,-9007199254740991,9007199254740992,100000000000000000000,1e+21,2.5,1e-7,0],` +
		`"m":[1.7976931348623157e+308,-1.7976931348623157e+308]}`
	if got, _ := Encode(v, 0); string(got) != filter {
		t.Errorf("Encode\ngot  %s\nwant %s", got, filter)
	}
	pretty, _ := EncodeInput(mustParse(t, `[11e17]`).Value, 2)
	if string(pretty) != "[\n  1.1e+18\n]" {
		t.Errorf("EncodeInput(indent 2) = %q", pretty)
	}
	if _, err := EncodeInput(value.Null{}, 9); !errors.Is(err, ErrIndent) {
		t.Errorf("EncodeInput(indent 9): %v, want ErrIndent", err)
	}
}

// TestEncodedLen: EncodedLen is the length Encode writes, at every indent,
// for every escape, number form, nesting and empty container.
func TestEncodedLen(t *testing.T) {
	var docs []value.Value
	for _, src := range []string{
		`null`, `true`, `false`, `0`, `-0.5`, `1e21`, `1e-7`, `""`, `"\u0000\u001f\b\t\n\f\r\"\\/é😀"`,
		`[]`, `{}`, `[[]]`, `{"": {}}`, `[1, [2, [3, []]], {"a": [true, null]}]`,
		`{"b": [1, {"c": [], "d": "\u0001\t\"é"}, 2.50], "a": {}, "e": null, "f": [true], "\n": "x"}`,
	} {
		docs = append(docs, mustParse(t, src).Value)
	}
	for _, v := range docs {
		for indent := 0; indent <= MaxIndent; indent++ {
			b, err := Encode(v, indent)
			if err != nil {
				t.Fatal(err)
			}
			if n := EncodedLen(v, indent); n != int64(len(b)) {
				t.Errorf("EncodedLen(%s, %d) = %d, Encode writes %d bytes", Canonical(v), indent, n, len(b))
			}
		}
	}
}

// TestCanonicalDigest holds CanonicalDigest to the digest of Canonical's
// text, for values whose text passes the size at which the digest writes
// a piece at a time, and for small ones.
func TestCanonicalDigest(t *testing.T) {
	long := strings.Repeat("a\"\\\u0001é", spillAt/3)
	elems := make([]value.Value, 0, spillAt)
	for i := range spillAt {
		n, err := value.ParseNumber(strconv.Itoa(i) + ".50")
		if err != nil {
			t.Fatal(err)
		}
		elems = append(elems, n)
	}
	arr, err := value.NewArray(elems...)
	if err != nil {
		t.Fatal(err)
	}
	s, err := value.NewString(long)
	if err != nil {
		t.Fatal(err)
	}
	obj, err := value.NewObject(value.Member{Name: "z", Value: s}, value.Member{Name: "a", Value: arr}, value.Member{Name: "é", Value: value.Null{}})
	if err != nil {
		t.Fatal(err)
	}
	for _, v := range []value.Value{value.Null{}, value.Bool(true), s, arr, obj} {
		if got, want := CanonicalDigest(v), sha256.Sum256(Canonical(v)); got != want {
			t.Errorf("CanonicalDigest of a %s is not the digest of its canonical text", v.Kind())
		}
	}
}
