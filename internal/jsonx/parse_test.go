// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package jsonx

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"metais.org/metais/internal/finding"
	"metais.org/metais/internal/limits"
	"metais.org/metais/internal/liquid/value"
)

// vendor is the source every test reads from: vendor.json in the data
// directory, the input vendor.
var vendor = Source{
	Input: finding.Subject{Kind: finding.SubjectInput, Name: "vendor"},
	At:    finding.Location{Origin: finding.OriginDataDir, File: "vendor.json"},
}

func parse(t *testing.T, src string) (Doc, []finding.Finding) {
	t.Helper()
	return Parse([]byte(src), vendor, limits.Defaults())
}

// mustParse parses src, failing t on any finding.
func mustParse(t *testing.T, src string) Doc {
	t.Helper()
	d, fs := parse(t, src)
	if len(fs) > 0 {
		t.Fatalf("Parse(%q): %s", src, describe(fs))
	}
	if d.Value == nil {
		t.Fatalf("Parse(%q): no findings and no value", src)
	}
	return d
}

// want is one expected finding: its code, line, column and pointer
// (hasPtr false for none).
type want struct {
	code   finding.Code
	line   int
	column int
	ptr    string
	hasPtr bool
}

func at(code finding.Code, line, column int, ptr string) want {
	return want{code, line, column, ptr, true}
}

func atNoPtr(code finding.Code, line, column int) want {
	return want{code: code, line: line, column: column}
}

func got(fs []finding.Finding) []want {
	var ws []want
	for _, f := range fs {
		l, ok := f.Location()
		if !ok {
			ws = append(ws, want{code: f.Code()})
			continue
		}
		ws = append(ws, want{f.Code(), l.Line, l.Column, l.Pointer, l.HasPointer})
	}
	return ws
}

func describe(fs []finding.Finding) string {
	var b strings.Builder
	for _, f := range fs {
		l, _ := f.Location()
		fmt.Fprintf(&b, "\n\t%s at %d:%d %q (%v): %s", f.Code(), l.Line, l.Column, l.Pointer, l.HasPointer, f.Message())
	}
	return b.String()
}

func checkFindings(t *testing.T, src string, fs []finding.Finding, ws ...want) {
	t.Helper()
	if g := got(fs); !slices.Equal(g, ws) {
		t.Errorf("Parse(%q):\ngot  %+v\nwant %+v%s", src, g, ws, describe(fs))
	}
	for _, f := range fs {
		l, ok := f.Location()
		if !ok || l.Line < 1 || l.Column < 1 || l.Origin != finding.OriginDataDir || l.File != "vendor.json" {
			t.Errorf("Parse(%q): %s is not positioned in vendor.json: %+v", src, f.Code(), l)
		}
		if f.Subject() != vendor.Input {
			t.Errorf("Parse(%q): %s is about %+v, not the input", src, f.Code(), f.Subject())
		}
	}
}

func param(t *testing.T, f finding.Finding, name string) string {
	t.Helper()
	v, ok := f.Param(name)
	if !ok {
		t.Fatalf("%s has no param %s", f.Code(), name)
	}
	if s, ok := v.Str(); ok {
		return s
	}
	if n, ok := v.Int(); ok {
		return fmt.Sprint(n)
	}
	t.Fatalf("%s param %s is a %s", f.Code(), name, v.Type())
	return ""
}

func TestParseValues(t *testing.T) {
	d := mustParse(t, ` {"b": [1, 2.50, -0, 1E3, true, false, null], "a": {"x": "yé😂\n\/"}, "": ""} `)
	o, ok := d.Value.(value.Object)
	if !ok {
		t.Fatalf("root is %T", d.Value)
	}
	if names := o.Names(); !slices.Equal(names, []string{"", "a", "b"}) {
		t.Errorf("names %q, want canonical order", names)
	}
	a, _ := o.Get("a")
	x, _ := a.(value.Object).Get("x")
	if s := x.(value.String).Text(); s != "yé😂\n/" {
		t.Errorf("a.x = %q", s)
	}
	b, _ := o.Get("b")
	arr := b.(value.Array)
	if arr.Len() != 7 {
		t.Fatalf("b has %d elements", arr.Len())
	}
	for i, k := range []value.Kind{value.KindNumber, value.KindNumber, value.KindNumber, value.KindNumber, value.KindBool, value.KindBool, value.KindNull} {
		if arr.At(i).Kind() != k {
			t.Errorf("b[%d] is %s, want %s", i, arr.At(i).Kind(), k)
		}
	}
	for _, src := range []string{`0`, `-1.5e-3`, `"s"`, `true`, `false`, `null`, `[]`, `{}`, "\t\r\n[ ]\n", `[[[]]]`} {
		mustParse(t, src)
	}
}

// TestNumberLiteralKept is W2-6.2: every number keeps the text it was
// written as.
func TestNumberLiteralKept(t *testing.T) {
	lits := []string{
		"0", "-0", "1", "1.0", "2.50", "1E3", "1e+3", "1e-3", "-12.5E-07",
		"9007199254740991", "-9007199254740991", "9007199254740992.0", "1e300",
		"0.000000000000000000000000001", "333333333.33333329", "1e-400",
	}
	src := "[" + strings.Join(lits, ",") + "]"
	arr := mustParse(t, src).Value.(value.Array)
	for i, lit := range lits {
		n := arr.At(i).(value.Number)
		if n.Literal() != lit {
			t.Errorf("element %d: literal %q, want %q", i, n.Literal(), lit)
		}
	}
	obj := mustParse(t, `{"n": 4.50}`).Value.(value.Object)
	n, _ := obj.Get("n")
	if n.(value.Number).Literal() != "4.50" {
		t.Errorf("member n: literal %q, want 4.50", n.(value.Number).Literal())
	}
}

// TestParseRefusals is W2-6.1: each refusal of ADR-003 D8 and D9 and
// ADR-010 D7 item 1, with its code and position.
func TestParseRefusals(t *testing.T) {
	cases := []struct {
		name string
		src  string
		want []want
	}{
		// ADR-003 D8 item 1: not UTF-8, located by line and byte.
		{"invalid UTF-8", "{\"a\": \"\xff\"}", []want{atNoPtr(CodeEncodingInvalid, 1, 8)}},
		{"invalid UTF-8 on a later line", "[\n1,\n\"é\xc3\"]", []want{atNoPtr(CodeEncodingInvalid, 3, 3)}},
		{"an encoded surrogate", "\"\xed\xa0\x80\"", []want{atNoPtr(CodeEncodingInvalid, 1, 2)}},
		{"a truncated sequence at the end", "\"\xe2\x82", []want{atNoPtr(CodeEncodingInvalid, 1, 2)}},
		{"a UTF-16 LE byte order mark", "\xff\xfe[\x00]\x00", []want{atNoPtr(CodeEncodingInvalid, 1, 1)}},
		{"a UTF-16 BE byte order mark", "\xfe\xff\x00[\x00]", []want{atNoPtr(CodeEncodingInvalid, 1, 1)}},
		{"a second byte order mark", "\xef\xbb\xbf\xef\xbb\xbf{}", []want{at(CodeSyntax, 1, 1, "")}},

		// ADR-003 D8 item 4: an escaped unpaired surrogate or a
		// noncharacter.
		{"a lone high surrogate", `["\ud800"]`, []want{at(CodeLoneSurrogate, 1, 2, "/0")}},
		{"a lone low surrogate", `{"a": "x\udc00y"}`, []want{at(CodeLoneSurrogate, 1, 7, "/a")}},
		{"a high surrogate before a non-surrogate", `"\ud800A"`, []want{at(CodeLoneSurrogate, 1, 1, "")}},
		{"two high surrogates", `"\ud800\ud800"`, []want{at(CodeLoneSurrogate, 1, 1, "")}},
		{"a reversed pair", `"\udc00\ud800"`, []want{at(CodeLoneSurrogate, 1, 1, "")}},
		{"a lone surrogate in a name", `{"\ud800": 1}`, []want{at(CodeLoneSurrogate, 1, 2, "")}},
		{"an escaped noncharacter", `"\ufdd0"`, []want{at(CodeNoncharacter, 1, 1, "")}},
		{"an escaped U+FFFF", `["\uffff"]`, []want{at(CodeNoncharacter, 1, 2, "/0")}},
		{"an escaped supplementary noncharacter", `"\ud83f\udffe"`, []want{at(CodeNoncharacter, 1, 1, "")}},
		{"a raw noncharacter", "{\"k\": \"\xef\xb7\x90\"}", []want{at(CodeNoncharacter, 1, 7, "/k")}},
		{"a raw U+10FFFF", "\"\xf4\x8f\xbf\xbf\"", []want{at(CodeNoncharacter, 1, 1, "")}},
		{"a noncharacter in a name", "{\"\xef\xbf\xbe\": 1}", []want{at(CodeNoncharacter, 1, 2, "")}},

		// ADR-003 D9 item 2: every duplicate name, compared after escapes.
		{"a duplicate name", `{"a": 1, "b": 2, "a": 3}`, []want{at(CodeDuplicateName, 1, 18, "/a")}},
		{"every duplicate", "{\"a\": 1,\n \"a\": 2,\n \"a\": 3}", []want{at(CodeDuplicateName, 2, 2, "/a"), at(CodeDuplicateName, 3, 2, "/a")}},
		{"a duplicate by escape", `{"a": 1, "\u0061": 2}`, []want{at(CodeDuplicateName, 1, 10, "/a")}},
		{"a nested duplicate", `[{"x": {"~/": 1, "~/": 2}}]`, []want{at(CodeDuplicateName, 1, 18, "/0/x/~0~1")}},
		{"duplicates in two objects", `[{"a":1,"a":1},{"a":1,"a":1}]`, []want{at(CodeDuplicateName, 1, 9, "/0/a"), at(CodeDuplicateName, 1, 23, "/1/a")}},

		// ADR-003 D8 item 5: numbers the data model cannot hold.
		{"an overflow", `[1e400]`, []want{at(CodeNumberOutOfRange, 1, 2, "/0")}},
		{"a negative overflow", `{"n": -1.5E309}`, []want{at(CodeNumberOutOfRange, 1, 7, "/n")}},
		{"an integer above 2^53-1", `9007199254740992`, []want{at(CodeNumberOutOfRange, 1, 1, "")}},
		{"an integer below -(2^53-1)", `[-9007199254740993]`, []want{at(CodeNumberOutOfRange, 1, 2, "/0")}},

		// ADR-003 D9 item 1: one RFC 8259 text.
		{"empty", ``, []want{at(CodeSyntax, 1, 1, "")}},
		{"whitespace only", " \n ", []want{at(CodeSyntax, 2, 2, "")}},
		{"a byte order mark only", "\xef\xbb\xbf", []want{at(CodeSyntax, 1, 1, "")}},
		{"an unclosed object", `{`, []want{at(CodeSyntax, 1, 2, "")}},
		{"an unclosed array", `[1`, []want{at(CodeSyntax, 1, 3, "")}},
		{"a trailing comma in an array", `[1,]`, []want{at(CodeSyntax, 1, 4, "/1")}},
		{"a trailing comma in an object", `{"a":1,}`, []want{at(CodeSyntax, 1, 8, "")}},
		{"a missing colon", `{"a" 1}`, []want{at(CodeSyntax, 1, 6, "/a")}},
		{"a missing comma", `[1 2]`, []want{at(CodeSyntax, 1, 4, "")}},
		{"a single-quoted name", `{'a': 1}`, []want{at(CodeSyntax, 1, 2, "")}},
		{"a bare name", `{a: 1}`, []want{at(CodeSyntax, 1, 2, "")}},
		{"a non-string name", `{1: 1}`, []want{at(CodeSyntax, 1, 2, "")}},
		{"a leading zero", `[01]`, []want{at(CodeSyntax, 1, 2, "/0")}},
		{"a leading plus", `+1`, []want{at(CodeSyntax, 1, 1, "")}},
		{"a bare point", `.5`, []want{at(CodeSyntax, 1, 1, "")}},
		{"no fraction digit", `1.`, []want{at(CodeSyntax, 1, 3, "")}},
		{"no exponent digit", `1e+`, []want{at(CodeSyntax, 1, 4, "")}},
		{"a bare minus", `-`, []want{at(CodeSyntax, 1, 2, "")}},
		{"a hexadecimal number", `0x10`, []want{atNoPtr(CodeTrailingContent, 1, 2)}},
		{"NaN", `NaN`, []want{at(CodeSyntax, 1, 1, "")}},
		{"Infinity", `[-Infinity]`, []want{at(CodeSyntax, 1, 3, "/0")}},
		{"a misspelt literal", `tru`, []want{at(CodeSyntax, 1, 4, "")}},
		{"a capitalised literal", `True`, []want{at(CodeSyntax, 1, 1, "")}},
		{"an unterminated string", `"abc`, []want{at(CodeSyntax, 1, 5, "")}},
		{"a raw line feed in a string", "\"a\nb\"", []want{at(CodeSyntax, 1, 3, "")}},
		{"a raw tab in a string", "[\"a\tb\"]", []want{at(CodeSyntax, 1, 4, "/0")}},
		{"an unknown escape", `"\x41"`, []want{at(CodeSyntax, 1, 2, "")}},
		{"a short unicode escape", `"\u12"`, []want{at(CodeSyntax, 1, 6, "")}},
		{"a non-hex unicode escape", `"\u12g4"`, []want{at(CodeSyntax, 1, 6, "")}},
		{"an escape at the end", `"\`, []want{at(CodeSyntax, 1, 3, "")}},
		{"a comment", `/* c */ 1`, []want{at(CodeSyntax, 1, 1, "")}},
		{"a line comment after a value", "[1 // c\n]", []want{at(CodeSyntax, 1, 4, "")}},
		{"a control character outside a string", "[\x01]", []want{at(CodeSyntax, 1, 2, "/0")}},
		{"a form feed as whitespace", "[\f1]", []want{at(CodeSyntax, 1, 2, "/0")}},
		{"a no-break space as whitespace", "[ 1]", []want{at(CodeSyntax, 1, 2, "/0")}},
		{"an error deep inside", "{\"a\": [{\"b\": [1, }]}]}", []want{at(CodeSyntax, 1, 18, "/a/0/b/1")}},
		{"trailing content", `{} x`, []want{atNoPtr(CodeTrailingContent, 1, 4)}},
		{"a second value", "1\n2", []want{atNoPtr(CodeTrailingContent, 2, 1)}},
		{"an extra bracket", `[]]`, []want{atNoPtr(CodeTrailingContent, 1, 3)}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			d, fs := parse(t, c.src)
			checkFindings(t, c.src, fs, c.want...)
			if d.Value != nil {
				t.Errorf("Parse(%q) gave a value beside its findings", c.src)
			}
		})
	}
}

// TestFindingParams pins the params each code states.
func TestFindingParams(t *testing.T) {
	_, fs := parse(t, "[\"a\",\n\"\xc0\"]")
	if param(t, fs[0], "reason") != reasonInvalidUTF8 || param(t, fs[0], "offset") != "7" {
		t.Errorf("invalid UTF-8: reason %s offset %s", param(t, fs[0], "reason"), param(t, fs[0], "offset"))
	}
	_, fs = parse(t, "\xff\xfe")
	if param(t, fs[0], "reason") != reasonUTF16 {
		t.Errorf("UTF-16: reason %s", param(t, fs[0], "reason"))
	}
	_, fs = parse(t, `[1e999, 12345678901234567890]`)
	if param(t, fs[0], "reason") != reasonOverflow || param(t, fs[1], "reason") != reasonUnsafeInteger {
		t.Errorf("numbers: reasons %s, %s", param(t, fs[0], "reason"), param(t, fs[1], "reason"))
	}
	_, fs = parse(t, `{"name": 1, "name": 2}`)
	if param(t, fs[0], "name") != "name" {
		t.Errorf("duplicate: name %s", param(t, fs[0], "name"))
	}
	rel := fs[0].Related()
	if len(rel) != 1 || rel[0].Note != noteFirst || rel[0].Location.Line != 1 || rel[0].Location.Column != 2 || rel[0].Location.Pointer != "/name" {
		t.Errorf("duplicate: related %+v, want the first occurrence at 1:2", rel)
	}
	_, fs = parse(t, `[1,`)
	if param(t, fs[0], "expected") != wantValue || param(t, fs[0], "found") != foundEnd {
		t.Errorf("syntax: expected %s found %s", param(t, fs[0], "expected"), param(t, fs[0], "found"))
	}
	_, fs = parse(t, `{"a":1 "b":2}`)
	if param(t, fs[0], "expected") != "',' or '}'" || param(t, fs[0], "found") != foundCharacter {
		t.Errorf("syntax: expected %s found %s", param(t, fs[0], "expected"), param(t, fs[0], "found"))
	}
}

// TestFindingsEchoNoValue holds the findings to ADR-006 D9 item 4: no
// message, hint or param repeats a string value from the input.
func TestFindingsEchoNoValue(t *testing.T) {
	const secret = "hunter2secret"
	srcs := []string{
		`{"a": "` + secret + `", "a": "` + secret + `"}`,
		`["` + secret + `\ud800"]`,
		`["` + secret + `\ufdd0"]`,
		`["` + secret + `\q"]`,
		`["` + secret + `"] ` + secret,
		`["` + secret + "\xff\"]",
		`["` + secret,
	}
	for _, src := range srcs {
		_, fs := parse(t, src)
		if len(fs) == 0 {
			t.Fatalf("Parse(%q) reported nothing", src)
		}
		for _, f := range fs {
			text := f.Message() + f.Hint()
			for _, name := range []string{"reason", "expected", "found", "name", "offset"} {
				if v, ok := f.Param(name); ok {
					s, _ := v.Str()
					text += s
				}
			}
			if strings.Contains(text, "hunter") {
				t.Errorf("Parse(%q): %s echoes the value: %s", src, f.Code(), text)
			}
		}
	}
}

// TestEveryFindingInOnePass: the findings the reading can get past are all
// reported together.
func TestEveryFindingInOnePass(t *testing.T) {
	src := `{"a": "\ud800", "b": 1e400, "a": 2, "c": ["\ufdd0", 9007199254740993], "d": "\udc00"}`
	_, fs := parse(t, src)
	checkFindings(t, src, fs,
		at(CodeLoneSurrogate, 1, 7, "/a"),
		at(CodeNumberOutOfRange, 1, 22, "/b"),
		at(CodeDuplicateName, 1, 29, "/a"),
		at(CodeNoncharacter, 1, 43, "/c/0"),
		at(CodeNumberOutOfRange, 1, 53, "/c/1"),
		at(CodeLoneSurrogate, 1, 77, "/d"),
	)
	// A syntax error after them stops the reading, and is reported too.
	src = `["\ud800", 1e400, ]`
	_, fs = parse(t, src)
	checkFindings(t, src, fs, at(CodeLoneSurrogate, 1, 2, "/0"), at(CodeNumberOutOfRange, 1, 12, "/1"), at(CodeSyntax, 1, 19, "/2"))
}

// lowered returns the defaults with each "name=value" of args applied.
func lowered(t *testing.T, args ...string) limits.Set {
	t.Helper()
	s := limits.Defaults()
	for _, a := range args {
		if f, ok := s.Lower(a); !ok {
			t.Fatalf("Lower(%q): %s", a, f.Message())
		}
	}
	return s
}

func limitParams(t *testing.T, f finding.Finding) (name, bound, observed string, lowered bool) {
	t.Helper()
	l, ok := f.Param("lowered")
	if !ok || l.Type() != finding.TypeBoolean {
		t.Fatalf("%s has no boolean lowered", f.Code())
	}
	observed = "-"
	if _, ok := f.Param("observed"); ok {
		observed = param(t, f, "observed")
	}
	low, _ := l.Bool()
	return param(t, f, "limit"), param(t, f, "bound"), observed, low
}

// TestLimits: ADR-009's value-depth, number-length and input-size, at
// their defaults and lowered.
func TestLimits(t *testing.T) {
	depth := int(limits.Get(limits.ValueDepth).Default)
	ok := strings.Repeat("[", depth) + strings.Repeat("]", depth)
	mustParse(t, ok)
	mustParse(t, strings.Repeat(`{"a":`, depth-1)+"[]"+strings.Repeat("}", depth-1))

	deep := strings.Repeat("[", depth+1) + strings.Repeat("]", depth+1)
	_, fs := parse(t, deep)
	ptr := strings.Repeat("/0", depth)
	checkFindings(t, deep, fs, at(CodeLimitExceeded, 1, depth+1, ptr))
	if name, bound, observed, low := limitParams(t, fs[0]); name != "value-depth" || bound != "64" || observed != "-" || low {
		t.Errorf("value-depth: %s %s %s %v", name, bound, observed, low)
	}

	// A breach far below the bound of a hostile depth costs only the
	// bound: the reading stops at the first level too deep.
	hostile := []byte(strings.Repeat("[", 1<<20))
	p := &parser{src: hostile, from: vendor, lim: limits.Defaults(), lines: newLineIndex(hostile)}
	p.document()
	checkFindings(t, "[…", p.findings, at(CodeLimitExceeded, 1, depth+1, ptr))
	if p.i != depth {
		t.Errorf("the reading of %d open brackets stopped after %d bytes, want %d", len(hostile), p.i, depth)
	}

	lim := lowered(t, "value-depth=2")
	_, fs = Parse([]byte(`{"a": [[1]]}`), vendor, lim)
	checkFindings(t, "lowered", fs, at(CodeLimitExceeded, 1, 8, "/a/0"))
	if name, bound, _, low := limitParams(t, fs[0]); name != "value-depth" || bound != "2" || !low {
		t.Errorf("lowered value-depth: %s %s %v", name, bound, low)
	}

	maxLen := int(limits.Get(limits.NumberLength).Default)
	fits := "0." + strings.Repeat("0", maxLen-3) + "1"
	mustParse(t, "["+fits+"]")
	long := "0." + strings.Repeat("0", maxLen-2) + "1"
	src := `[` + long + `, "\ud800"]`
	_, fs = parse(t, src)
	checkFindings(t, src, fs, at(CodeLimitExceeded, 1, 2, "/0"), at(CodeLoneSurrogate, 1, maxLen+5, "/1"))
	if name, bound, observed, _ := limitParams(t, fs[0]); name != "number-length" || bound != "1024" || observed != "1025" {
		t.Errorf("number-length: %s %s %s", name, bound, observed)
	}

	lim = lowered(t, "input-size=4")
	if _, fs := Parse([]byte(`[12]`), vendor, lim); len(fs) != 0 {
		t.Errorf("4 bytes under input-size=4: %s", describe(fs))
	}
	_, fs = Parse([]byte(`[123]`), vendor, lim)
	checkFindings(t, "input-size", fs, atNoPtr(CodeLimitExceeded, 1, 1))
	if name, bound, observed, low := limitParams(t, fs[0]); name != "input-size" || bound != "4" || observed != "5" || !low {
		t.Errorf("input-size: %s %s %s %v", name, bound, observed, low)
	}
}
