// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package liquid

import (
	"encoding/json"
	"fmt"
	"runtime"
	"strings"
	"testing"

	"metais.org/metais/internal/liquid/value"
	"metais.org/metais/internal/testenv"
)

// jv reads JSON as a template value, each number taking its kind from its
// spelling as a literal does (ADR-004 D4 item 2): 5 is an integer, 5.0 a
// float.
func jv(t testing.TB, src string) value.Value {
	t.Helper()
	dec := json.NewDecoder(strings.NewReader(src))
	dec.UseNumber()
	var x any
	if err := dec.Decode(&x); err != nil {
		t.Fatalf("%s: %v", src, err)
	}
	v, err := spelled(x)
	if err != nil {
		t.Fatalf("%s: %v", src, err)
	}
	return v
}

func spelled(x any) (value.Value, error) {
	switch x := x.(type) {
	case json.Number:
		return value.ParseNumber(string(x))
	case []any:
		var elems []value.Value
		for _, e := range x {
			v, err := spelled(e)
			if err != nil {
				return nil, err
			}
			elems = append(elems, v)
		}
		return value.NewArray(elems...)
	case map[string]any:
		var ms []value.Member
		for k, e := range x {
			v, err := spelled(e)
			if err != nil {
				return nil, err
			}
			ms = append(ms, value.Member{Name: k, Value: v})
		}
		return value.NewObject(ms...)
	}
	return fromJSON(x)
}

// same reports whether a and b are the same value, kinds of number
// included.
func same(a, b value.Value) bool {
	return string(appendIdentity(nil, a)) == string(appendIdentity(nil, b))
}

// show writes v as JSON for a failure message, a float with its point.
func show(v value.Value) string {
	switch v := v.(type) {
	case nil:
		return "<none>"
	case value.Number:
		if numberKind(v).isInt {
			return v.Literal()
		}
		return floatText(v.Float64())
	case value.String:
		b, _ := json.Marshal(v.Text())
		return string(b)
	case value.Array:
		var parts []string
		for _, e := range v.All() {
			parts = append(parts, show(e))
		}
		return "[" + strings.Join(parts, ",") + "]"
	case value.Object:
		var parts []string
		for k, e := range v.All() {
			parts = append(parts, fmt.Sprintf("%q:%s", k, show(e)))
		}
		return "{" + strings.Join(parts, ",") + "}"
	case value.Null:
		return "null"
	case value.Bool:
		return fmt.Sprint(bool(v))
	}
	return "?"
}

// fcase is one application: the filter, its input and positional
// arguments as JSON, and either the result as JSON or the refusal's code.
type fcase struct {
	filter string
	in     string
	args   string // a JSON array
	want   string
	code   Code
}

func (c fcase) run(t *testing.T) {
	t.Helper()
	def, ok := Core().Filter(c.filter)
	if !ok {
		t.Fatalf("Core() has no %s", c.filter)
	}
	var args []Arg
	if c.args != "" {
		for _, a := range jv(t, c.args).(value.Array).All() {
			args = append(args, Arg{Value: a})
		}
	}
	got, err := def.Apply(testContext{newTestMeter(nil)}, jv(t, c.in), args)
	label := fmt.Sprintf("%s | %s: %s", c.in, c.filter, c.args)
	switch {
	case c.code != "":
		if err == nil || err.Code != c.code {
			t.Errorf("%s = %s, %v; want %s", label, show(got), err, c.code)
		} else if err.Args["filter"] != c.filter && err.Code != CodeLimit {
			t.Errorf("%s: %v names no filter %s", label, err, c.filter)
		}
	case err != nil:
		t.Errorf("%s: %v; want %s", label, err, c.want)
	case !same(got, jv(t, c.want)):
		t.Errorf("%s = %s; want %s", label, show(got), c.want)
	}
}

// TestArithmeticFilters holds the arithmetic filters to ADR-004 D4
// item 3: integers stay integers and floor; a float operand gives the
// exact result on the operands' shortest decimal values, rounded once;
// round rounds half away from zero; integer overflow and division by zero
// are refused. Operands are coerced as Shopify's Utils.to_number does.
func TestArithmeticFilters(t *testing.T) {
	testenv.Home(t)
	for _, c := range []fcase{
		{filter: "plus", in: `0.1`, args: `[0.2]`, want: `0.3`},
		{filter: "plus", in: `1`, args: `[2]`, want: `3`},
		{filter: "plus", in: `1`, args: `[2.0]`, want: `3.0`},
		{filter: "minus", in: `0.3`, args: `[0.1]`, want: `0.2`},
		{filter: "times", in: `3`, args: `[0.1]`, want: `0.3`},
		{filter: "times", in: `-5`, args: `[2]`, want: `-10`},
		{filter: "divided_by", in: `10`, args: `[4]`, want: `2`},
		{filter: "divided_by", in: `10`, args: `[4.0]`, want: `2.5`},
		{filter: "divided_by", in: `-7`, args: `[2]`, want: `-4`},
		{filter: "divided_by", in: `7`, args: `[-2]`, want: `-4`},
		{filter: "divided_by", in: `20`, args: `[7.0]`, want: `2.857142857142857`},
		{filter: "divided_by", in: `1`, args: `[3.0]`, want: `0.3333333333333333`},
		{filter: "modulo", in: `-7`, args: `[2]`, want: `1`},
		{filter: "modulo", in: `7`, args: `[-2]`, want: `-1`},
		{filter: "modulo", in: `-7.5`, args: `[2]`, want: `0.5`},
		{filter: "modulo", in: `10.1`, args: `[7.0]`, want: `3.1`},
		{filter: "divided_by", in: `1`, args: `[0]`, code: CodeDivisionByZero},
		{filter: "divided_by", in: `1.5`, args: `[0.0]`, code: CodeDivisionByZero},
		{filter: "modulo", in: `1`, args: `["x"]`, code: CodeDivisionByZero},
		{filter: "plus", in: `9007199254740991`, args: `[1]`, code: CodeNumberRange},
		{filter: "minus", in: `-9007199254740991`, args: `[1]`, code: CodeNumberRange},
		{filter: "times", in: `4503599627370496`, args: `[2]`, code: CodeNumberRange},
		{filter: "times", in: `94906265`, args: `[94906265]`, want: `9007199136250225`},
		{filter: "times", in: `94906266`, args: `[94906266]`, code: CodeNumberRange},
		{filter: "times", in: `1.0e308`, args: `[10]`, code: CodeNumberRange},
		{filter: "plus", in: `"12abc"`, args: `[1]`, want: `13`},
		{filter: "plus", in: `" 1_000"`, args: `[1]`, want: `1001`},
		{filter: "plus", in: `" -5.5 "`, args: `[1]`, want: `-4.5`},
		{filter: "plus", in: `"5."`, args: `[1]`, want: `6`},
		{filter: "plus", in: `"99999999999999999999"`, args: `[0]`, code: CodeNumberRange},
		{filter: "plus", in: `{"a":1}`, args: `[[1]]`, want: `0`},
		{filter: "plus", in: `true`, args: `[null]`, want: `0`},
		{filter: "abs", in: `-0`, want: `0`},
		{filter: "abs", in: `-5.5`, want: `5.5`},
		{filter: "abs", in: `"-5.1"`, want: `5.1`},
		{filter: "ceil", in: `5.4`, want: `6`},
		{filter: "ceil", in: `-5.4`, want: `-5`},
		{filter: "ceil", in: `1.0e20`, code: CodeNumberRange},
		{filter: "floor", in: `-5.4`, want: `-6`},
		{filter: "floor", in: `"5.9"`, want: `5`},
		{filter: "round", in: `2.5`, want: `3`},
		{filter: "round", in: `-2.5`, want: `-3`},
		{filter: "round", in: `-0.5`, want: `-1`},
		{filter: "round", in: `-0.4`, want: `0`},
		{filter: "round", in: `1.005`, args: `[2]`, want: `1.01`},
		{filter: "round", in: `-1.005`, args: `[2]`, want: `-1.01`},
		{filter: "round", in: `5.666`, args: `[1.9]`, want: `5.7`},
		{filter: "round", in: `5.666`, args: `[0]`, want: `6`},
		{filter: "round", in: `1250.0`, args: `[-2]`, want: `1300`},
		{filter: "round", in: `1250`, args: `[-2]`, want: `1300`},
		{filter: "round", in: `-1249`, args: `[-2]`, want: `-1200`},
		{filter: "round", in: `7`, args: `[2]`, want: `7`},
		{filter: "round", in: `7`, args: `[-9999]`, want: `0`},
		{filter: "round", in: `0.1`, args: `[99999]`, want: `0.1`},
		{filter: "round", in: `1.0e300`, code: CodeNumberRange},
		{filter: "at_least", in: `5`, args: `[8.5]`, want: `8.5`},
		{filter: "at_least", in: `5`, args: `[5.0]`, want: `5`},
		{filter: "at_most", in: `5.0`, args: `[5]`, want: `5.0`},
		{filter: "at_most", in: `"abc"`, args: `[-2]`, want: `-2`},
		{filter: "sum", in: `[1, [2, [3]]]`, want: `6`},
		{filter: "sum", in: `[1, 2.5]`, want: `3.5`},
		{filter: "sum", in: `[0.1, 0.2]`, want: `0.3`},
		{filter: "sum", in: `[]`, want: `0`},
		{filter: "sum", in: `null`, want: `0`},
		{filter: "sum", in: `[9007199254740991, 1, -1]`, want: `9007199254740991`},
		{filter: "sum", in: `[9007199254740991, 1]`, code: CodeNumberRange},
		{filter: "sum", in: `[{"k": 1}, {"k": "2"}, {"x": 3}, null, true]`, args: `["k"]`, want: `3`},
		{filter: "sum", in: `[{"k": 1}, 2]`, args: `["k"]`, code: CodeFilterInput},
		{filter: "sum", in: `[1, 2]`, args: `[null]`, want: `3`},
	} {
		c.run(t)
	}
}

// TestValueText holds the text a string filter reads to ADR-004 D4 item
// 4: Ruby's Float#to_s with its two notations and their boundaries, an
// integer's digits, an array's elements concatenated, and no text for an
// object.
func TestValueText(t *testing.T) {
	testenv.Home(t)
	for _, c := range []fcase{
		{in: `3.0`, want: `"3.0"`},
		{in: `-0.0`, want: `"-0.0"`},
		{in: `0.0`, want: `"0.0"`},
		{in: `0.0001`, want: `"0.0001"`},
		{in: `0.00001`, want: `"1.0e-05"`},
		{in: `0.000123`, want: `"0.000123"`},
		{in: `123456789012345.0`, want: `"123456789012345.0"`},
		{in: `999999999999999.9`, want: `"999999999999999.9"`},
		{in: `1.0e15`, want: `"1.0e+15"`},
		{in: `1.5e300`, want: `"1.5e+300"`},
		{in: `-2.5e-7`, want: `"-2.5e-07"`},
		{in: `1e2`, want: `"100.0"`},
		{in: `-0`, want: `"0"`},
		{in: `12`, want: `"12"`},
		{in: `true`, want: `"true"`},
		{in: `null`, want: `""`},
		{in: `[1, [2.0, "a"], null, false]`, want: `"12.0afalse"`},
		{in: `{"a": 1}`, code: CodeObjectOutput},
		{in: `[{"a": 1}]`, code: CodeObjectOutput},
	} {
		c.filter, c.args = "append", `[""]`
		c.run(t)
	}
}

// TestStringFilters holds the string filters to Shopify's definitions,
// with ADR-004 D1 item 4's simple case mapping and D9 item 1's escape
// mapping; lengths and offsets count characters.
func TestStringFilters(t *testing.T) {
	testenv.Home(t)
	for _, c := range []fcase{
		{filter: "append", in: `5`, args: `[1.5]`, want: `"51.5"`},
		{filter: "prepend", in: `"b"`, args: `["a"]`, want: `"ab"`},
		{filter: "append", in: `"a"`, args: `[{"x": 1}]`, code: CodeObjectOutput},
		{filter: "upcase", in: `"straße ǆ ı"`, want: `"STRAßE Ǆ I"`},
		{filter: "downcase", in: `"ÀBẞ"`, want: `"àbß"`},
		{filter: "capitalize", in: `"hELLO wORLD"`, want: `"Hello world"`},
		{filter: "capitalize", in: `"ǆemal"`, want: `"ǅemal"`},
		{filter: "capitalize", in: `""`, want: `""`},
		{filter: "escape", in: `"<a href=\"x\">'&'</a>"`, want: `"&lt;a href=&quot;x&quot;&gt;&#39;&amp;&#39;&lt;/a&gt;"`},
		{filter: "h", in: `"<&>"`, want: `"&lt;&amp;&gt;"`},
		{filter: "escape", in: `null`, want: `null`},
		{filter: "escape", in: `1.5`, want: `"1.5"`},
		{filter: "escape_once", in: `"&amp; & &#38; &#x26; &lt &;"`, want: `"&amp; &amp; &#38; &amp;#x26; &amp;lt &amp;;"`},
		{filter: "strip", in: "\" \\u0000\\t a b\\n\\u000b\\f\\r \"", want: `"a b"`},
		{filter: "lstrip", in: `"  a  "`, want: `"a  "`},
		{filter: "rstrip", in: `"  a  "`, want: `"  a"`},
		{filter: "squish", in: "\" a \\t\\n b  c \"", want: `"a b c"`},
		{filter: "squish", in: `null`, want: `null`},
		{filter: "strip_html", in: `"<script x>a</script>b<!-- c\n -->c<style>d</style><p class='q'>e</p>"`, want: `"bce"`},
		{filter: "strip_html", in: `"<script>a"`, want: `"a"`},
		{filter: "strip_html", in: `"a < b"`, want: `"a < b"`},
		{filter: "strip_newlines", in: `"a\r\nb\nc\rd"`, want: `"abc\rd"`},
		{filter: "newline_to_br", in: `"a\r\nb\rc\n"`, want: `"a<br />\nb\rc<br />\n"`},
		{filter: "replace", in: `"aaa"`, args: `["a", "bb"]`, want: `"bbbbbb"`},
		{filter: "replace", in: `"aé"`, args: `["", "-"]`, want: `"-a-é-"`},
		{filter: "replace", in: `"abc"`, args: `["b"]`, want: `"ac"`},
		{filter: "replace_first", in: `"abab"`, args: `["b", "X"]`, want: `"aXab"`},
		{filter: "replace_first", in: `"ab"`, args: `["", "X"]`, want: `"Xab"`},
		{filter: "replace_last", in: `"abcabc"`, args: `["bc", "X"]`, want: `"abcaX"`},
		{filter: "replace_last", in: `"ab"`, args: `["", "X"]`, want: `"abX"`},
		{filter: "replace_last", in: `"ab"`, args: `["z", "X"]`, want: `"ab"`},
		{filter: "remove", in: `"a1b1"`, args: `[1]`, want: `"ab"`},
		{filter: "remove", in: `"ab"`, args: `[""]`, want: `"ab"`},
		{filter: "remove_first", in: `"a1b1"`, args: `[1]`, want: `"ab1"`},
		{filter: "remove_last", in: `"a1b1"`, args: `[1]`, want: `"a1b"`},
		{filter: "split", in: `"a,b,,c,,"`, args: `[","]`, want: `["a","b","","c"]`},
		{filter: "split", in: `",a"`, args: `[","]`, want: `["","a"]`},
		{filter: "split", in: "\"  a \\t b\\n\"", args: `[" "]`, want: `["a","b"]`},
		{filter: "split", in: `"aé"`, args: `[""]`, want: `["a","é"]`},
		{filter: "split", in: `""`, args: `[","]`, want: `[]`},
		{filter: "split", in: `",,"`, args: `[","]`, want: `[]`},
		{filter: "split", in: `12345`, args: `[3]`, want: `["12","45"]`},
		{filter: "split", in: `"a b"`, args: `[null]`, want: `["a"," ","b"]`},
		{filter: "truncate", in: `"日本語テキスト"`, args: `[4]`, want: `"日..."`},
		{filter: "truncate", in: `"abc"`, args: `[2, ""]`, want: `"ab"`},
		{filter: "truncate", in: `"abc"`, args: `[3]`, want: `"abc"`},
		{filter: "truncate", in: `"abcdef"`, args: `[-1]`, want: `"..."`},
		{filter: "truncate", in: `"abcdef"`, args: `[" 2 ", 1]`, want: `"a1"`},
		{filter: "truncate", in: `null`, args: `[1]`, want: `null`},
		{filter: "truncate", in: `"abc"`, args: `[1.0]`, code: CodeFilterArgument},
		{filter: "truncate", in: `"abc"`, args: `["1e2"]`, code: CodeFilterArgument},
		{filter: "truncate", in: `"abc"`, args: `["0x10"]`, code: CodeFilterArgument},
		{filter: "truncate", in: `"abc"`, args: `["99999999999999999999"]`, code: CodeNumberRange},
		{filter: "truncatewords", in: `"  a  b\tc d "`, args: `[2]`, want: `"a b..."`},
		{filter: "truncatewords", in: `"a b c"`, args: `[0]`, want: `"a..."`},
		{filter: "truncatewords", in: `" a  b "`, args: `[2]`, want: `" a  b "`},
		{filter: "truncatewords", in: `"a b c"`, args: `[9007199254740991]`, want: `"a b c"`},
		{filter: "truncatewords", in: `"a b c"`, args: `[1, null]`, want: `"a"`},
		{filter: "url_encode", in: `"a b/é~*"`, want: `"a+b%2F%C3%A9~%2A"`},
		{filter: "url_encode", in: `null`, want: `null`},
		{filter: "url_decode", in: `"%C3%a9+%zz%4"`, want: `"é %zz%4"`},
		{filter: "url_decode", in: `"%FF"`, code: CodeInvalidUTF8},
		{filter: "base64_encode", in: `"é"`, want: `"w6k="`},
		{filter: "base64_encode", in: `"~~~"`, want: `"fn5+"`},
		{filter: "base64_url_safe_encode", in: `"~~~"`, want: `"fn5-"`},
		{filter: "base64_url_safe_encode", in: `"a"`, want: `"YQ=="`},
		{filter: "base64_decode", in: `"fn5+"`, want: `"~~~"`},
		{filter: "base64_decode", in: `"YQ"`, code: CodeFilterInput},
		{filter: "base64_decode", in: `"YQ==\n"`, code: CodeFilterInput},
		{filter: "base64_decode", in: `"fn5-"`, code: CodeFilterInput},
		{filter: "base64_decode", in: `"/w=="`, code: CodeInvalidUTF8},
		{filter: "base64_decode", in: `null`, want: `""`},
		{filter: "base64_url_safe_decode", in: `"fn5-"`, want: `"~~~"`},
		{filter: "base64_url_safe_decode", in: `"fn5+"`, want: `"~~~"`},
		{filter: "base64_url_safe_decode", in: `"YQ"`, want: `"a"`},
		{filter: "base64_url_safe_decode", in: `"Y"`, code: CodeFilterInput},
	} {
		c.run(t)
	}
}

// TestArrayFilters holds the array filters to Shopify's definitions:
// input flattened, an object read as one element, nil as none; objects
// iterated in canonical order (ADR-004 D4 item 5); sorting stable, by
// value and by code point.
func TestArrayFilters(t *testing.T) {
	testenv.Home(t)
	for _, c := range []fcase{
		{filter: "size", in: `"héllo"`, want: `5`},
		{filter: "size", in: `{"a": 1, "b": 2}`, want: `2`},
		{filter: "size", in: `[1, [2, 3]]`, want: `2`},
		{filter: "size", in: `5`, want: `0`},
		{filter: "first", in: `"héllo"`, want: `"h"`},
		{filter: "last", in: `"héllo"`, want: `"o"`},
		{filter: "first", in: `{"b": 1, "a": 2, "😀": 3, "！": 4}`, want: `["a", 2]`},
		{filter: "last", in: `{"b": 1, "a": 2, "😀": 3, "！": 4}`, want: `["！", 4]`},
		{filter: "first", in: `[]`, want: `null`},
		{filter: "last", in: `""`, want: `null`},
		{filter: "first", in: `5`, want: `null`},
		{filter: "join", in: `[1, 2.0, true, null, "x"]`, args: `[","]`, want: `"1,2.0,true,,x"`},
		{filter: "join", in: `[[1, [2]], 3]`, args: `["-"]`, want: `"1-2-3"`},
		{filter: "join", in: `["a", "b"]`, want: `"a b"`},
		{filter: "join", in: `{"a": 1}`, code: CodeObjectOutput},
		{filter: "reverse", in: `[1, [2, 3]]`, want: `[3, 2, 1]`},
		{filter: "reverse", in: `"ab"`, want: `["ab"]`},
		{filter: "concat", in: `[1, [2]]`, args: `[[[3]]]`, want: `[1, 2, [3]]`},
		{filter: "concat", in: `null`, args: `[[1]]`, want: `[1]`},
		{filter: "concat", in: `[1]`, args: `[{"a": 1}]`, code: CodeFilterArgument},
		{filter: "compact", in: `[1, null, [null, 2]]`, want: `[1, 2]`},
		{filter: "compact", in: `[{"a": 1}, {"a": null}, {"b": 1}, "xa", "y"]`, args: `["a"]`, want: `[{"a": 1}, "xa"]`},
		{filter: "compact", in: `[{"a": 1}, null]`, args: `["a"]`, want: `null`},
		{filter: "uniq", in: `[1, 1.0, 1, "1", 1.0, [], []]`, want: `[1, 1.0, "1"]`},
		{filter: "uniq", in: `[{"a": 1}, {"a": 1.0}, {"a": 1}]`, want: `[{"a": 1}, {"a": 1.0}]`},
		{filter: "uniq", in: `[{"a": 1, "k": 1}, {"a": 2, "k": 1}, {"a": 3}]`, args: `["k"]`, want: `[{"a": 1, "k": 1}, {"a": 3}]`},
		{filter: "uniq", in: `[0.0, -0.0]`, want: `[0.0]`},
		{filter: "map", in: `[{"a": 1}, {"b": 2}, null, "xa"]`, args: `["a"]`, want: `[1, null, null, "a"]`},
		{filter: "map", in: `{"a": 1}`, args: `["a"]`, want: `[1]`},
		{filter: "map", in: `[1]`, args: `["a"]`, code: CodeFilterInput},
		{filter: "map", in: `["a"]`, args: `[1]`, code: CodeFilterInput},
		{filter: "sort", in: `[3, 1.5, 2, null, 10]`, want: `[1.5, 2, 3, 10, null]`},
		{filter: "sort", in: `["b", "B", null, "a", "é", "z"]`, want: `["B", "a", "b", "z", "é", null]`},
		{filter: "sort", in: `[1, "a"]`, code: CodeFilterInput},
		{filter: "sort", in: `[true, false]`, code: CodeFilterInput},
		{filter: "sort", in: `[[1]]`, want: `[1]`},
		{filter: "sort", in: `[{"a": 1}]`, want: `[{"a": 1}]`},
		{filter: "sort", in: `[{"k": 2, "i": 0}, {"k": 1, "i": 1}, {"k": 2, "i": 2}, {"i": 3}]`, args: `["k"]`,
			want: `[{"k": 1, "i": 1}, {"k": 2, "i": 0}, {"k": 2, "i": 2}, {"i": 3}]`},
		{filter: "sort", in: `[{"k": 2}, null]`, args: `["k"]`, want: `null`},
		{filter: "sort", in: `[{"k": 2}, 1]`, args: `["k"]`, code: CodeFilterInput},
		{filter: "sort_natural", in: `["b", "B", "a", "A", null, "_"]`, want: `["_", "a", "A", "b", "B", null]`},
		{filter: "sort_natural", in: `[{"t": 9}, {"t": 1111}, {"t": 87}]`, args: `["t"]`, want: `[{"t": 1111}, {"t": 87}, {"t": 9}]`},
		{filter: "sort_natural", in: `[{"t": {}}, {"t": 1}]`, args: `["t"]`, code: CodeObjectOutput},
		{filter: "slice", in: `[1, 2, 3, 4]`, args: `[-2]`, want: `[3]`},
		{filter: "slice", in: `[1, 2, 3, 4]`, args: `[1, 99]`, want: `[2, 3, 4]`},
		{filter: "slice", in: `[1, 2]`, args: `[0, -1]`, want: `[]`},
		{filter: "slice", in: `[1, 2]`, args: `[-3]`, want: `[]`},
		{filter: "slice", in: `"héllo"`, args: `[1, 3]`, want: `"éll"`},
		{filter: "slice", in: `"héllo"`, args: `["-1", null]`, want: `"o"`},
		{filter: "slice", in: `"abc"`, args: `[3]`, want: `""`},
		{filter: "slice", in: `"abc"`, args: `[4]`, want: `""`},
		{filter: "slice", in: `"abc"`, args: `[1, 9007199254740991]`, want: `"bc"`},
		{filter: "slice", in: `"abc"`, args: `[1.5]`, code: CodeFilterArgument},
		{filter: "slice", in: `{"a": 1}`, args: `[0]`, code: CodeObjectOutput},
		{filter: "where", in: `[{"a": 1}, {"a": 2}, {"a": null}, {"a": false}]`, args: `["a"]`, want: `[{"a": 1}, {"a": 2}]`},
		{filter: "where", in: `[{"a": 1}, {"a": 2}, {"a": 2.0}]`, args: `["a", 2]`, want: `[{"a": 2}, {"a": 2.0}]`},
		{filter: "where", in: `[{"a": [1]}, {"a": [2]}]`, args: `["a", [2]]`, want: `[{"a": [2]}]`},
		{filter: "where", in: `[{"a": "1"}]`, args: `["a", 1]`, want: `[]`},
		{filter: "where", in: `[{"a": 1}, null]`, args: `["a"]`, want: `null`},
		{filter: "where", in: `"foobar"`, args: `["oo"]`, want: `["foobar"]`},
		{filter: "where", in: `[]`, args: `["a"]`, want: `[]`},
		{filter: "where", in: `[1]`, args: `["a"]`, code: CodeFilterInput},
		{filter: "reject", in: `[{"a": 1}, {"a": false}, {}]`, args: `["a"]`, want: `[{"a": false}, {}]`},
		{filter: "reject", in: `[{"a": 1}, {"a": false}]`, args: `["a", false]`, want: `[{"a": 1}]`},
		{filter: "has", in: `[{"a": 1}, null]`, args: `["a", 1]`, want: `true`},
		{filter: "has", in: `[null, {"a": 1}]`, args: `["a", 1]`, want: `null`},
		{filter: "has", in: `["x", 1]`, args: `["x"]`, want: `true`},
		{filter: "has", in: `[{"a": 2}]`, args: `["a", 1]`, want: `false`},
		{filter: "has", in: `[]`, args: `["a"]`, want: `false`},
		{filter: "find", in: `[{"a": 1, "i": 0}, {"a": 2, "i": 1}]`, args: `["a", 2]`, want: `{"a": 2, "i": 1}`},
		{filter: "find", in: `[{"a": 1}]`, args: `["a", 2]`, want: `null`},
		{filter: "find_index", in: `[{"a": 1}, [{"a": 2}]]`, args: `["a", 2]`, want: `1`},
		{filter: "find_index", in: `[{"a": 1}]`, args: `["b"]`, want: `null`},
		{filter: "default", in: `null`, args: `["d"]`, want: `"d"`},
		{filter: "default", in: `false`, args: `["d"]`, want: `"d"`},
		{filter: "default", in: `""`, args: `["d"]`, want: `"d"`},
		{filter: "default", in: `[]`, args: `["d"]`, want: `"d"`},
		{filter: "default", in: `{}`, args: `["d"]`, want: `"d"`},
		{filter: "default", in: `0`, args: `["d"]`, want: `0`},
		{filter: "default", in: `" "`, args: `["d"]`, want: `" "`},
		{filter: "default", in: `null`, want: `""`},
	} {
		c.run(t)
	}
}

// TestDefaultAllowFalse pins default's one keyword.
func TestDefaultAllowFalse(t *testing.T) {
	testenv.Home(t)
	def, _ := Core().Filter("default")
	for _, c := range []struct {
		in, allow string
		want      string
	}{
		{`false`, `true`, `false`},
		{`false`, `1`, `false`},
		{`false`, `false`, `"d"`},
		{`false`, `null`, `"d"`},
		{`null`, `true`, `"d"`},
		{`""`, `true`, `"d"`},
	} {
		got, err := def.Apply(testContext{newTestMeter(nil)}, jv(t, c.in),
			[]Arg{{Name: "allow_false", Value: jv(t, c.allow)}, {Value: jv(t, `"d"`)}})
		if err != nil || !same(got, jv(t, c.want)) {
			t.Errorf("%s | default: allow_false: %s, \"d\" = %s, %v; want %s", c.in, c.allow, show(got), err, c.want)
		}
	}
}

// sample is an application of every filter of Core() that succeeds.
var sample = map[string]struct{ in, args string }{
	"abs": {`-1`, ``}, "append": {`"a"`, `["b"]`}, "at_least": {`1`, `[2]`}, "at_most": {`1`, `[2]`},
	"base64_decode": {`"YQ=="`, ``}, "base64_encode": {`"a"`, ``}, "base64_url_safe_decode": {`"YQ"`, ``},
	"base64_url_safe_encode": {`"a"`, ``}, "capitalize": {`"ab"`, ``}, "ceil": {`1.5`, ``},
	"compact": {`[1, null]`, ``}, "concat": {`[1]`, `[[2]]`}, "default": {`null`, `[1]`},
	"divided_by": {`7`, `[2]`}, "downcase": {`"AB"`, ``}, "escape": {`"<"`, ``}, "escape_once": {`"<&lt;"`, ``},
	"find": {`[{"a": 1}]`, `["a"]`}, "find_index": {`[{"a": 1}]`, `["a"]`}, "first": {`[1, 2]`, ``},
	"floor": {`1.5`, ``}, "h": {`">"`, ``}, "has": {`[{"a": 1}]`, `["a"]`}, "join": {`[1, 2]`, `["-"]`},
	"last": {`[1, 2]`, ``}, "lstrip": {`" a "`, ``}, "map": {`[{"a": 1}]`, `["a"]`}, "minus": {`1`, `[2]`},
	"modulo": {`7`, `[2]`}, "newline_to_br": {`"a\nb"`, ``}, "plus": {`1`, `[2]`}, "prepend": {`"a"`, `["b"]`},
	"reject": {`[{"a": 1}, {}]`, `["a"]`}, "remove": {`"aba"`, `["a"]`}, "remove_first": {`"aba"`, `["a"]`},
	"remove_last": {`"aba"`, `["a"]`}, "replace": {`"aba"`, `["a", "c"]`}, "replace_first": {`"aba"`, `["a", "c"]`},
	"replace_last": {`"aba"`, `["a", "c"]`}, "reverse": {`[1, 2]`, ``}, "round": {`1.25`, `[1]`},
	"rstrip": {`" a "`, ``}, "size": {`"ab"`, ``}, "slice": {`"abc"`, `[1]`}, "sort": {`[2, 1]`, ``},
	"sort_natural": {`["b", "A"]`, ``}, "split": {`"a,b"`, `[","]`}, "squish": {`" a  b "`, ``},
	"strip": {`" a "`, ``}, "strip_html": {`"<b>a</b>"`, ``}, "strip_newlines": {`"a\nb"`, ``},
	"sum": {`[1, 2]`, ``}, "times": {`2`, `[3]`}, "truncate": {`"abcdef"`, `[4]`}, "truncatewords": {`"a b c"`, `[1]`},
	"uniq": {`[1, 1]`, ``}, "upcase": {`"ab"`, ``}, "url_decode": {`"a%20b"`, ``}, "url_encode": {`"a b"`, ``},
	"where": {`[{"a": 1}, {}]`, `["a"]`},
}

func sampleArgs(t testing.TB, src string) []Arg {
	var args []Arg
	if src != "" {
		for _, a := range jv(t, src).(value.Array).All() {
			args = append(args, Arg{Value: a})
		}
	}
	return args
}

// TestEveryFilterCharges holds every filter of Core() to FilterDef's
// charging contract (ADR-009 D11 item 2): it charges at least the shallow
// sizes of its input, its arguments and its result; it refuses as soon as
// the meter does, with the meter's error, before its work, so a meter
// that refuses the first charge gets no result; and the same application
// gives the same result and charges the same, every time.
func TestEveryFilterCharges(t *testing.T) {
	testenv.Home(t)
	for _, name := range Core().FilterNames() {
		s, ok := sample[name]
		if !ok {
			t.Errorf("no sample application of %s", name)
			continue
		}
		def, _ := Core().Filter(name)
		in, args := jv(t, s.in), sampleArgs(t, s.args)
		m := newTestMeter(nil)
		got, err := def.Apply(testContext{m}, in, args)
		if err != nil {
			t.Errorf("%s: %v", name, err)
			continue
		}
		floor := shallow(in) + shallow(got)
		for _, a := range args {
			floor += shallow(a.Value)
		}
		if m.Used(WorkUnits) < floor {
			t.Errorf("%s charged %d units, below the shallow sizes' %d", name, m.Used(WorkUnits), floor)
		}
		again := newTestMeter(nil)
		if got2, err := def.Apply(testContext{again}, in, args); err != nil || !same(got, got2) ||
			again.Used(WorkUnits) != m.Used(WorkUnits) || again.Used(WorkHeldBytes) != m.Used(WorkHeldBytes) {
			t.Errorf("%s gave %s then %s, %v, charging %d then %d", name, show(got), show(got2), err,
				m.Used(WorkUnits), again.Used(WorkUnits))
		}
		refusing := newTestMeter(map[Work]int64{WorkUnits: 1})
		if got, err := def.Apply(testContext{refusing}, in, args); err == nil || err.Code != CodeLimit || got != nil {
			t.Errorf("%s under a refusing meter gave %s, %v; want the meter's refusal", name, show(got), err)
		}
	}
}

// TestFilterCharges pins the charge of a few applications exactly, so a
// change to the table is a change a test sees.
func TestFilterCharges(t *testing.T) {
	testenv.Home(t)
	for _, c := range []struct {
		filter, in, args string
		units, held      int64
	}{
		// 2 + 3 for input and argument; the built "abcde": 5 units, and no
		// held bytes, which only assign and capture charge.
		{"append", `"ab"`, `["cde"]`, 10, 0},
		// 2 for the input, 2 for the built result.
		{"upcase", `"ab"`, ``, 4, 0},
		// 3 for the input, 1 for the integer result.
		{"size", `[1, 2, 3]`, ``, 4, 0},
		// 3 for the input; 3 comparisons ⌈log2 3⌉ = 2 deep, so 6; 3 for the result.
		{"sort", `[3, 1, 2]`, ``, 12, 0},
		// 3 for the input; 3 comparisons 2 deep, and the keys' 3 bytes 2
		// deep, so 12; 3 for the result.
		{"sort", `["c", "a", "b"]`, ``, 18, 0},
		// 2 + 1; 2 flattened from the nested array; the built "1,2,3": 5 units.
		{"join", `[[1, 2], 3]`, `[","]`, 10, 0},
		// 1 + 1 for the operands, 1 for the result.
		{"plus", `1`, `[2]`, 3, 0},
		// 3 for the input; the substring result: 1.
		{"strip", `" a "`, ``, 4, 0},
	} {
		def, _ := Core().Filter(c.filter)
		m := newTestMeter(nil)
		if _, err := def.Apply(testContext{m}, jv(t, c.in), sampleArgs(t, c.args)); err != nil {
			t.Fatalf("%s: %v", c.filter, err)
		}
		if m.Used(WorkUnits) != c.units || m.Used(WorkHeldBytes) != c.held {
			t.Errorf("%s | %s: %s charged %d units, %d held bytes; want %d, %d", c.in, c.filter, c.args,
				m.Used(WorkUnits), m.Used(WorkHeldBytes), c.units, c.held)
		}
	}
}

// chargeLog is a testMeter that keeps every charge it is asked for, in
// order, refused or not.
type chargeLog struct {
	*testMeter
	charges []charge
}

type charge struct {
	w Work
	n int64
}

func (m *chargeLog) Charge(w Work, n int64) *Error {
	m.charges = append(m.charges, charge{w, n})
	return m.testMeter.Charge(w, n)
}

func (m *chargeLog) Meter() Meter { return m }

// TestFilterChargesOperandsFirst pins the order of the stdFilter's
// charges: before a filter does any of its work, Apply charges the
// shallow sizes of its input and of its arguments, so a filter given
// operands larger than the meter allows is refused having done nothing.
// Every filter of the catalog is applied to its sample, and its first
// charge must be that one.
func TestFilterChargesOperandsFirst(t *testing.T) {
	testenv.Home(t)
	for _, name := range Core().FilterNames() {
		s, ok := sample[name]
		if !ok {
			t.Errorf("%s has no sample", name)
			continue
		}
		def, _ := Core().Filter(name)
		in, args := jv(t, s.in), sampleArgs(t, s.args)
		want := shallow(in)
		for _, a := range args {
			want += shallow(a.Value)
		}
		m := &chargeLog{testMeter: newTestMeter(nil)}
		if _, err := def.Apply(m, in, args); err != nil {
			t.Errorf("%s: %v", name, err)
			continue
		}
		if len(m.charges) == 0 || m.charges[0] != (charge{WorkUnits, want}) {
			t.Errorf("%s | %s: %s charged %v first; want %d units for its operands", s.in, name, s.args, m.charges, want)
		}
	}
}

// TestGrowthRefusedBeforeWork pins the charge-first property where it
// matters: a filter whose result would be far larger than its operands
// charges the result's size, and is refused, before it builds it. The
// meter holds the operands' charges and 16 KiB of units more; each
// application here would build at least four times that, and the replace
// with an empty pattern gibibytes. Each is refused by render-work, the
// operands charged, having allocated next to nothing: a built string
// charges no held bytes (ADR-009 Amendment A10 item 4), so its units are
// what refuse it.
func TestGrowthRefusedBeforeWork(t *testing.T) {
	testenv.Home(t)
	big := strings.Repeat("x", 1<<16)
	for _, c := range []struct {
		filter string
		in     value.Value
		args   []Arg
	}{
		{"replace", newString(big), []Arg{{Value: value.String{}}, {Value: newString(big)}}},
		{"replace", newString(big), []Arg{{Value: newString("x")}, {Value: newString(big)}}},
		{"append", newString(big + big), []Arg{{Value: newString(big + big)}}},
		{"url_encode", newString(strings.Repeat("é", 1<<15)), nil},
		{"join", newArray(func() []value.Value {
			vs := make([]value.Value, 1<<10)
			for i := range vs {
				vs[i] = newString(big)
			}
			return vs
		}()), []Arg{{Value: newString(big)}}},
	} {
		def, _ := Core().Filter(c.filter)
		operands := shallow(c.in)
		for _, a := range c.args {
			operands += shallow(a.Value)
		}
		m := newTestMeter(map[Work]int64{WorkUnits: operands + 1<<14})
		var before, after runtime.MemStats
		runtime.GC()
		runtime.ReadMemStats(&before)
		got, err := def.Apply(testContext{m}, c.in, c.args)
		runtime.ReadMemStats(&after)
		if err == nil || err.Code != CodeLimit || got != nil {
			t.Errorf("%s: %v; want refused by the meter", c.filter, err)
		}
		if used := m.Used(WorkUnits); used < operands {
			t.Errorf("%s: refused having charged %d units, before its operands' %d", c.filter, used, operands)
		}
		if grew := after.TotalAlloc - before.TotalAlloc; grew > 4<<20 {
			t.Errorf("%s allocated %d bytes before it was refused", c.filter, grew)
		}
	}
}

// TestFilterResultsAreValues holds every result a filter gives to the
// data model: Apply never hands back a nil Value beside a nil error.
func TestFilterResultsAreValues(t *testing.T) {
	testenv.Home(t)
	for _, name := range Core().FilterNames() {
		def, _ := Core().Filter(name)
		for _, in := range []string{`null`, `""`, `[]`, `{}`, `0`, `-1.5`, `true`, `"a b"`, `[null, [1], {"a": "b"}]`} {
			got, err := def.Apply(testContext{newTestMeter(nil)}, jv(t, in), sampleArgs(t, sample[name].args))
			if err == nil && got == nil {
				t.Errorf("%s | %s gave no value", in, name)
			}
			if err != nil && err.Code == CodeInternal {
				t.Errorf("%s | %s: %v", in, name, err)
			}
		}
	}
}
