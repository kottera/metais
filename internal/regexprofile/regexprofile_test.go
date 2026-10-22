// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package regexprofile

import (
	"errors"
	"strings"
	"testing"

	"metais.org/metais/internal/limits"
)

func mustCompile(t *testing.T, pattern string) *Regexp {
	t.Helper()
	re, err := Compile(pattern, limits.Defaults(), nil)
	if err != nil {
		t.Fatalf("Compile(%q): %v", pattern, err)
	}
	return re
}

// TestTranslation holds each row of ADR-009 D10 item 6's table to what
// ECMA-262 says the construct matches: each subject is matched, or not,
// as a u-mode RegExp's test would.
func TestTranslation(t *testing.T) {
	cases := []struct {
		pattern string
		match   []string
		nomatch []string
	}{
		// Literals, unanchored.
		{"abc", []string{"abc", "xabcx"}, []string{"ab", "ABC"}},
		{`a\.b`, []string{"a.b"}, []string{"axb"}},
		{`\/\$\^\*\+\?\(\)\[\]\{\}\|\\`, []string{`/$^*+?()[]{}|\`}, []string{"x"}},
		{"é", []string{"café"}, []string{"cafe"}},
		{"😀", []string{"x😀"}, []string{"x"}},
		// \uXXXX, a surrogate pair, and \u{…}.
		{`é`, []string{"é"}, []string{"e"}},
		{`😀`, []string{"😀"}, []string{"x"}},
		{`\u{1F600}`, []string{"😀"}, []string{"x"}},
		{`\x41`, []string{"A"}, []string{"a"}},
		// Control escapes, \cX and \0.
		{`\t\n\v\f\r`, []string{"\t\n\v\f\r"}, []string{"tnvfr"}},
		{`\cJ`, []string{"\n"}, []string{"J"}},
		{`\ca`, []string{"\x01"}, []string{"a"}},
		{`a\0b`, []string{"a\x00b"}, []string{"a0b"}},
		// "." excludes exactly ECMA-262's line terminators.
		{"^.$", []string{"a", "\v", "\u0085", "😀"}, []string{"\n", "\r", "\u2028", "\u2029", ""}},
		// \s and \S are ECMA-262's whitespace and line terminators.
		{`^\s$`, []string{" ", "\t", "\n", "\v", "\f", "\r", "\u00A0", "\u1680", "\u2000", "\u200A", "\u2028", "\u2029", "\u202F", "\u205F", "\u3000", "\uFEFF"},
			[]string{"\u0085", "\u200B", "\u180E", "a", ""}},
		{`^\S$`, []string{"a", "\u0085", "\u200B"}, []string{" ", "\u00A0", "\uFEFF", "\u3000"}},
		// \d, \w, \b and \B are ASCII.
		{`^\d$`, []string{"0", "9"}, []string{"٣", "a"}},
		{`^\D$`, []string{"a", "٣"}, []string{"5"}},
		{`^\w$`, []string{"a", "Z", "0", "_"}, []string{"é", "-"}},
		{`^\W$`, []string{"é", "-"}, []string{"a", "_"}},
		{`\bfoo\b`, []string{"a foo b", "foo"}, []string{"afoo", "foo_"}},
		{`\Boo`, []string{"foo"}, []string{"oo", " oo"}},
		// [] matches nothing; [^] matches any code point.
		{`[]`, nil, []string{"", "a", "\n"}},
		{`^[^]$`, []string{"a", "\n", "\u2028", "😀"}, []string{"", "ab"}},
		// \p{…} by general category and by script.
		{`^\p{Lu}$`, []string{"A", "É"}, []string{"a"}},
		{`^\p{gc=Lu}$`, []string{"A"}, []string{"a"}},
		{`^\p{General_Category=Nd}$`, []string{"5", "٣"}, []string{"a"}},
		{`^\p{Script=Greek}$`, []string{"α"}, []string{"a"}},
		{`^\p{sc=Latin}$`, []string{"a"}, []string{"α"}},
		{`^\P{L}$`, []string{"1"}, []string{"a"}},
		// Classes, ranges, negation and escapes inside them.
		{`^[a-c]$`, []string{"a", "b", "c"}, []string{"d"}},
		{`^[^a-c]$`, []string{"d", "\n"}, []string{"a"}},
		{`^[-a]$`, []string{"-", "a"}, []string{"b"}},
		{`^[a-]$`, []string{"-", "a"}, []string{"b"}},
		{`^[\d\s]$`, []string{"1", " ", "\u3000"}, []string{"a"}},
		{`^[\S]$`, []string{"a"}, []string{" "}},
		{`^[^\S]$`, []string{" ", "\uFEFF"}, []string{"a"}},
		{`^[\b]$`, []string{"\b"}, []string{"b"}},
		{`^[\-]$`, []string{"-"}, []string{"a"}},
		{`^[\p{Lu}\d]$`, []string{"A", "1"}, []string{"a"}},
		{`^[^\p{Lu}]$`, []string{"a"}, []string{"A"}},
		{`^[\u{1F600}-\u{1F64F}]$`, []string{"😀"}, []string{"a"}},
		{`^[.]$`, []string{"."}, []string{"a"}},
		// Quantifiers, their lazy forms, and bounds up to 1000.
		{"^a*$", []string{"", "aaa"}, []string{"b"}},
		{"^a+?$", []string{"a", "aa"}, []string{""}},
		{"^a??b$", []string{"b", "ab"}, []string{"aab"}},
		{"^a{2}$", []string{"aa"}, []string{"a", "aaa"}},
		{"^a{2,}$", []string{"aa", "aaaa"}, []string{"a"}},
		{"^a{2,3}?$", []string{"aa", "aaa"}, []string{"aaaa"}},
		{"^a{1000}$", []string{strings.Repeat("a", 1000)}, []string{strings.Repeat("a", 999)}},
		// Groups, named groups and alternation.
		{"^(ab)+$", []string{"abab"}, []string{"aba"}},
		{"^(?:a|b)c$", []string{"ac", "bc"}, []string{"c"}},
		{"^(?<year>[0-9]{4})-(?<mo>[0-9]{2})$", []string{"2026-10"}, []string{"26-10"}},
		{"^(?<ünï>x)$", []string{"x"}, []string{"y"}},
		{"cat|dog", []string{"hotdog"}, []string{"cow"}},
		{"^$", []string{""}, []string{"a"}},
		// ^ and $ match only at the ends of the subject: there is no m flag.
		{"^b", []string{"bc"}, []string{"a\nb"}},
		{"a$", []string{"ba"}, []string{"a\nb"}},
	}
	for _, c := range cases {
		re := mustCompile(t, c.pattern)
		for _, s := range c.match {
			if !re.MatchString(s) {
				t.Errorf("%q (translated %q) does not match %q", c.pattern, re.Translated(), s)
			}
		}
		for _, s := range c.nomatch {
			if re.MatchString(s) {
				t.Errorf("%q (translated %q) matches %q", c.pattern, re.Translated(), s)
			}
		}
		if re.String() != c.pattern {
			t.Errorf("String() = %q, want the pattern as written, %q", re.String(), c.pattern)
		}
	}
}

// TestRefusals is a negative test per construct ADR-009 D10 item 6
// refuses, and per kind of pattern that is not u-mode syntax: each is
// refused with the construct named and where it begins.
func TestRefusals(t *testing.T) {
	cases := []struct {
		pattern string
		want    Construct
		offset  int
	}{
		{"a(?=b)", Lookahead, 1},
		{"a(?!b)", Lookahead, 1},
		{"(?<=a)b", Lookbehind, 0},
		{"(?<!a)b", Lookbehind, 0},
		{`(a)\1`, Backreference, 3},
		{`(a)(b)(c)(d)(e)(f)(g)(h)(i)(j)\10`, Backreference, 30},
		{`(?<n>a)\k<n>`, Backreference, 7},
		{`a{1001}`, RepeatBound, 1},
		{`a{0,1001}`, RepeatBound, 1},
		{`a{1001,}`, RepeatBound, 1},
		{`a{99999999999999999999}`, RepeatBound, 1},
		{`(a{2}){600}`, RepeatBound, 0},
		{`\uD800`, LoneSurrogate, 0},
		{`x\uDC00`, LoneSurrogate, 1},
		{`\uD800A`, LoneSurrogate, 0},
		{`\u{D800}`, LoneSurrogate, 0},
		{`[\uDFFF]`, LoneSurrogate, 1},
		{`\p{Letter}`, UnicodeProperty, 0},
		{`\p{Uppercase_Letter}`, UnicodeProperty, 0},
		{`\p{Alphabetic}`, UnicodeProperty, 0},
		{`\p{Script=Grek}`, UnicodeProperty, 0},
		{`\p{Script_Extensions=Greek}`, UnicodeProperty, 0},
		{`\p{Greek}`, UnicodeProperty, 0},
		{`[\P{Any}]`, UnicodeProperty, 1},
		{`(?i)abc`, RE2Syntax, 0},
		{`(?i:abc)`, RE2Syntax, 0},
		{`(?s).`, RE2Syntax, 0},
		{`(?-m:a)`, RE2Syntax, 0},
		{`\Aabc`, RE2Syntax, 0},
		{`abc\z`, RE2Syntax, 3},
		{`abc\Z`, RE2Syntax, 3},
		{`\Qa.b\E`, RE2Syntax, 0},
		{`(?P<n>a)`, RE2Syntax, 0},
		{`\pL`, RE2Syntax, 0},
		{`\C`, RE2Syntax, 0},
		{`\a`, RE2Syntax, 0},
		// Not u-mode syntax.
		{"(", Syntax, 0},
		{"a)", Syntax, 1},
		{"[a", Syntax, 0},
		{"*a", Syntax, 0},
		{"a**", Syntax, 2},
		{"a{2}{3}", Syntax, 4},
		{"{", Syntax, 0},
		{"a{", Syntax, 1},
		{"a{,3}", Syntax, 1},
		{"a{3,2}", Syntax, 1},
		{"}", Syntax, 0},
		{"]", Syntax, 0},
		{"^*", Syntax, 1},
		{`\b+`, Syntax, 2},
		{`\e`, Syntax, 0},
		{`\-`, Syntax, 0},
		{`\c1`, Syntax, 0},
		{`\x4`, Syntax, 0},
		{`\u12`, Syntax, 0},
		{`\u{}`, Syntax, 0},
		{`\u{110000}`, Syntax, 0},
		{`\00`, Syntax, 0},
		{`[\d-z]`, Syntax, 3},
		{`[z-a]`, Syntax, 2},
		{`[\B]`, Syntax, 1},
		{`[\1]`, Syntax, 1},
		{`(?<a>x)(?<a>y)`, Syntax, 7},
		{`(?<>x)`, Syntax, 0},
		{`(?<1a>x)`, Syntax, 0},
		{`(?#comment)`, Syntax, 0},
		{`\p{Lu`, Syntax, 0},
		{`\p{L u}`, Syntax, 0},
		{"\xff", Syntax, 0},
		{`\`, Syntax, 0},
	}
	for _, c := range cases {
		_, err := Compile(c.pattern, limits.Defaults(), nil)
		var ue *UnsupportedError
		if !errors.As(err, &ue) {
			t.Errorf("Compile(%q) = %v, want an UnsupportedError", c.pattern, err)
			continue
		}
		if ue.Construct != c.want || ue.Offset != c.offset {
			t.Errorf("Compile(%q) refused %s at %d, want %s at %d", c.pattern, ue.Construct, ue.Offset, c.want, c.offset)
		}
		if strings.Contains(ue.Error(), c.pattern) && c.pattern != "" && len(c.pattern) > 1 {
			t.Errorf("the refusal of %q repeats the pattern: %q", c.pattern, ue.Error())
		}
	}
}

func lowered(t *testing.T, arg string) limits.Set {
	t.Helper()
	lim := limits.Defaults()
	if f, ok := lim.Lower(arg); !ok {
		t.Fatalf("Lower(%q): %v", arg, f.Message())
	}
	return lim
}

// TestPatternLength refuses a pattern longer than pattern-length before
// reading it, and accepts one at the bound.
func TestPatternLength(t *testing.T) {
	lim := lowered(t, "pattern-length=8")
	if _, err := Compile("abcdefgh", lim, nil); err != nil {
		t.Errorf("a pattern at the bound: %v", err)
	}
	// Nine bytes, the ninth an unclosed group that a reading would refuse
	// as syntax: the limit is checked first.
	_, err := Compile("abcdefgh(", lim, nil)
	var le *LimitError
	if !errors.As(err, &le) || le.Limit != limits.PatternLength || !le.HasObserved || le.Observed != 9 {
		t.Errorf("a pattern one byte over: %v, want pattern-length observed 9", err)
	}
	if _, err := Compile("["+strings.Repeat("a", 4094)+"]", limits.Defaults(), nil); err != nil {
		t.Errorf("a pattern at the default bound: %v", err)
	}
	if _, err := Compile(strings.Repeat("a", 4097), limits.Defaults(), nil); !errors.As(err, &le) {
		t.Errorf("a pattern over the default bound: %v", err)
	}
}

// TestRegexProgram refuses a pattern whose compiled program is larger
// than regex-program, counting the program's instructions.
func TestRegexProgram(t *testing.T) {
	re := mustCompile(t, "a{1000}")
	n := re.Instructions()
	if n <= 1000 {
		t.Fatalf("a{1000} compiles to %d instructions, want more than 1000", n)
	}
	var le *LimitError
	if _, err := Compile("a{1000}", lowered(t, "regex-program="+itoa(n-1)), nil); !errors.As(err, &le) ||
		le.Limit != limits.RegexProgram || le.Observed != int64(n) {
		t.Errorf("one instruction over: %v, want regex-program observed %d", err, n)
	}
	if _, err := Compile("a{1000}", lowered(t, "regex-program="+itoa(n)), nil); err != nil {
		t.Errorf("at the bound: %v", err)
	}
}

func itoa(n int) string {
	var b []byte
	for {
		b = append([]byte{byte('0' + n%10)}, b...)
		if n /= 10; n == 0 {
			return string(b)
		}
	}
}

// TestMeter charges each match the program's size times the subject's
// length, and refuses, before it runs, a match whose charge would exceed
// what remains (ADR-009 D10 item 7).
func TestMeter(t *testing.T) {
	re0 := mustCompile(t, "b")
	cost := int64(re0.Instructions()) * 4
	m := NewMeter(2 * cost)
	re, err := Compile("b", limits.Defaults(), m)
	if err != nil {
		t.Fatal(err)
	}
	if !re.MatchString("abcd") || m.Spent() != cost {
		t.Fatalf("first match: spent %d, want %d", m.Spent(), cost)
	}
	if !re.MatchString("abcd") || m.Exhausted() {
		t.Fatalf("second match, exactly the budget: exhausted %v", m.Exhausted())
	}
	if re.MatchString("b") || !m.Exhausted() {
		t.Errorf("a match beyond the budget ran, or did not mark the meter exhausted")
	}
	if m.Spent() != 2*cost {
		t.Errorf("a refused match was charged: spent %d, want %d", m.Spent(), 2*cost)
	}
}

// TestEveryErrorIsTyped holds Compile to its contract: whatever it is
// given, a refusal is an UnsupportedError or a LimitError.
func TestEveryErrorIsTyped(t *testing.T) {
	for _, p := range []string{"", "(((((", "[[[]]]", `\u{FFFFFFFFFF}`, "(?<a", `[\p{`, "a{1,", "(?", "(?<", `\k`, `[\c]`} {
		if _, err := Compile(p, limits.Defaults(), nil); err != nil && !typed(err) {
			t.Errorf("Compile(%q): an untyped error %v", p, err)
		}
	}
}

func typed(err error) bool {
	var ue *UnsupportedError
	var le *LimitError
	return errors.As(err, &ue) || errors.As(err, &le)
}
