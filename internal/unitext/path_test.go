// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package unitext

import (
	"slices"
	"strings"
	"testing"

	"metais.org/metais/internal/limits"
)

func TestIsNFC(t *testing.T) {
	for _, c := range []struct {
		s    string
		want bool
	}{
		{"", true},
		{"readme.md", true},
		{"caf\u00e9", true},
		{"cafe\u0301", false},
		{"\u212b", false}, // ANGSTROM SIGN decomposes to U+00C5
		{"\xff", false},
	} {
		if got := IsNFC(c.s); got != c.want {
			t.Errorf("IsNFC(%+q) = %v, want %v", c.s, got, c.want)
		}
	}
}

// TestNFC: a host name is normalized to NFC (ADR-003 D15 item 5), so the
// decomposed and precomposed spellings of one name are the same name.
func TestNFC(t *testing.T) {
	for _, c := range []struct{ s, want string }{
		{"", ""},
		{"readme.md", "readme.md"},
		{"cafe\u0301.md", "caf\u00e9.md"},
		{"caf\u00e9.md", "caf\u00e9.md"},
		{"\u212b", "\u00c5"},
	} {
		if got := NFC(c.s); got != c.want {
			t.Errorf("NFC(%+q) = %+q, want %+q", c.s, got, c.want)
		}
		if !IsNFC(NFC(c.s)) {
			t.Errorf("NFC(%+q) is not NFC", c.s)
		}
	}
}

func TestCaselessKey(t *testing.T) {
	for _, c := range []struct {
		a, b  string
		match bool
	}{
		{"Readme.md", "README.md", true},
		{"straße.md", "STRASSE.md", true}, // full folding: ß folds to ss
		{"caf\u00e9", "CAFE\u0301", true}, // canonical: NFD both sides
		{"\u212b", "\u00e5", true},        // ANGSTROM SIGN and å
		{"ǅ", "Ǆ", true},
		{"ﬃ", "FFI", true}, // status F
		{"a", "b", false},
		{"i", "ı", false}, // no Turkic tailoring
		{"İ", "i", false}, // İ folds to i + U+0307 under full folding
	} {
		if got := CaselessKey(c.a) == CaselessKey(c.b); got != c.match {
			t.Errorf("%+q and %+q match = %v, want %v", c.a, c.b, got, c.match)
		}
	}
}

func TestIsPathChar(t *testing.T) {
	for _, c := range []struct {
		r    rune
		want bool
	}{
		{'a', true}, {'Z', true}, {'0', true}, {'-', true}, {'_', true}, {'~', true},
		{' ', true}, {'é', true}, {'\u0301', true}, {'中', true}, {'€', true}, {'\U0001F600', true},
		{'\u0000', false}, {'\t', false}, {'\n', false}, {'\u007f', false}, {'\u0085', false},
		{'\u00a0', false}, {'\u2028', false}, {'\u2029', false}, {'\u3000', false},
		{'\u200b', false}, {'\u200d', false}, {'\u202e', false}, {'\u2066', false}, {'\u00ad', false}, {'\ufeff', false},
		{'\ue000', false}, {'\U000F0000', false}, // private use
		{'\ufdd0', false}, {'\uffff', false}, {'\U0001FFFE', false}, // noncharacters
		{'\u0378', false}, // unassigned in 15.0.0
	} {
		if got := IsPathChar(c.r); got != c.want {
			t.Errorf("IsPathChar(%U) = %v, want %v", c.r, got, c.want)
		}
	}
}

// problems formats CheckPath's answer for comparison.
func problems(p string, lim limits.Set) []string {
	var out []string
	for _, pr := range CheckPath(p, lim) {
		s := kindName[pr.Kind]
		if pr.Reason != "" {
			s += ":" + pr.Reason
		}
		if pr.Kind == Character {
			s += ":" + string(pr.CodePoint)
		}
		out = append(out, s)
	}
	return out
}

var kindName = map[Kind]string{
	Encoding: "encoding", NotNFC: "not_nfc", Shape: "shape", Character: "character",
	NotPortable: "not_portable", Length: "length", Depth: "depth",
}

// TestCheckPath holds the predicate to ADR-005 D12 items 1 to 5 and to
// ADR-001 D2, a path at a time: every accepted path the rules allow, and
// every refusal with the rule it breaks.
func TestCheckPath(t *testing.T) {
	lim := limits.Defaults()
	accepted := []string{
		"a", "metais.yaml", "templates/report.md.liquid", "assets/.well-known/security.txt",
		"templates/.github/workflows/ci.yml", "a b/c d.txt", "caf\u00e9/na\u00efve.md", "中文/文件.txt",
		"€uro.txt", "~tilde", "a.b.c", ".hidden", "..x", "x..y", "conf.txt", "console", "com10", "lpt",
		"nul_", "auxiliary.md", "com", "git", ".gitignore", ".github", "git~2", "COM\u2074.txt",
		"emoji\U0001F600.txt", strings.Repeat("a", 255), strings.Repeat("a/", 31) + "a",
		strings.Repeat("é", 127) + "a",
	}
	for _, p := range accepted {
		if got := problems(p, lim); got != nil {
			t.Errorf("CheckPath(%+q) = %q, want nothing", p, got)
		}
	}
	refused := []struct {
		p    string
		want []string
	}{
		{"", []string{"shape:empty"}},
		{"\xff", []string{"encoding:invalid_utf8"}},
		{"a/\xc0\x80", []string{"encoding:invalid_utf8"}},
		{"\uFEFFa", []string{"encoding:byte_order_mark"}},
		{"cafe\u0301", []string{"not_nfc"}},
		{"\u212b.txt", []string{"not_nfc"}},
		{"/a", []string{"shape:leading_slash"}},
		{"a/", []string{"shape:trailing_slash"}},
		{"/", []string{"shape:leading_slash", "shape:trailing_slash", "shape:empty_segment"}},
		{"a//b", []string{"shape:empty_segment"}},
		{"./a", []string{"shape:dot_segment"}},
		{"a/../b", []string{"shape:dot_segment"}},
		{"..", []string{"shape:dot_segment"}},
		{`a\b`, []string{"shape:backslash"}},
		{"C:/x", []string{"shape:colon"}},
		{"a:stream", []string{"shape:colon"}},
		{"a<b", []string{"character:<"}},
		{"a>b", []string{"character:>"}},
		{`a"b`, []string{`character:"`}},
		{"a|b", []string{"character:|"}},
		{"a?b", []string{"character:?"}},
		{"a*b", []string{"character:*"}},
		{"a\x00b", []string{"character:\x00"}},
		{"a\tb", []string{"character:\t"}},
		{"a\nb", []string{"character:\n"}},
		{"a\x7fb", []string{"character:\x7f"}},
		{"a\u00a0b", []string{"character:\u00a0"}},
		{"a\u202eb", []string{"character:\u202e"}},
		{"a\u200db", []string{"character:\u200d"}},
		{"a\ue000b", []string{"character:\ue000"}},
		{"a\u0378b", []string{"character:\u0378"}},
		{"a\ufffcb", []string{"character:\ufffc"}},
		{"a\ufffdb", []string{"character:\ufffd"}},
		{"a\ufff0b", []string{"character:\ufff0"}},
		{"a ", []string{"not_portable:segment_end"}},
		{"a./b", []string{"not_portable:segment_end"}},
		{"CON", []string{"not_portable:reserved_name"}},
		{"con.md", []string{"not_portable:reserved_name"}},
		{"Con .md", []string{"not_portable:reserved_name"}},
		{"aux.c", []string{"not_portable:reserved_name"}},
		{"x/NUL.tar.gz", []string{"not_portable:reserved_name"}},
		{"prn", []string{"not_portable:reserved_name"}},
		{"COM0", []string{"not_portable:reserved_name"}},
		{"lpt9.txt", []string{"not_portable:reserved_name"}},
		{"com\u00b9.txt", []string{"not_portable:reserved_name"}},
		{"LPT\u00b3", []string{"not_portable:reserved_name"}},
		{".git", []string{"not_portable:git"}},
		{".GIT/config", []string{"not_portable:git"}},
		{"a/.Git/hooks/pre-commit", []string{"not_portable:git"}},
		{"GIT~1/config", []string{"not_portable:git"}},
		{".g\u200cit", []string{"character:\u200c", "not_portable:git"}},
		{".git.", []string{"not_portable:segment_end"}},
		{strings.Repeat("a", 256), []string{"not_portable:segment_length"}},
		{strings.Repeat("é", 128), []string{"not_portable:segment_length"}},
		{strings.Repeat("abcdefg/", 64) + "x", []string{"length", "depth"}},
		{strings.Repeat("a/", 32) + "a", []string{"depth"}},
		{strings.Repeat(strings.Repeat("a", 200)+"/", 2) + strings.Repeat("a", 111), []string{"length"}},
		{"con/a<b/.git/\u00ad", []string{"not_portable:reserved_name", "character:<", "not_portable:git"}},
	}
	for _, c := range refused {
		got := problems(c.p, lim)
		slices.Sort(got)
		want := slices.Sorted(slices.Values(c.want))
		// Each kind is reported once: of two bad characters, the first.
		if !slices.Equal(got, want) {
			t.Errorf("CheckPath(%+q) = %q, want %q", c.p, got, want)
		}
	}
}

// TestCheckPathBounds holds the predicate to a lowered path-length and
// path-depth: the bounds come from internal/limits, not from constants.
func TestCheckPathBounds(t *testing.T) {
	lim := limits.Defaults()
	for _, arg := range []string{"path-length=8", "path-depth=2"} {
		if f, ok := lim.Lower(arg); !ok {
			t.Fatalf("Lower(%s): %v", arg, f.Message())
		}
	}
	for _, c := range []struct {
		p    string
		want []string
	}{
		{"abcdefgh", nil},
		{"abcdefghi", []string{"length"}},
		{"a/b", nil},
		{"a/b/c", []string{"depth"}},
	} {
		if got := problems(c.p, lim); !slices.Equal(got, c.want) {
			t.Errorf("CheckPath(%q) at lowered limits = %q, want %q", c.p, got, c.want)
		}
	}
	got := CheckPath("abcdefghij", lim)
	if len(got) != 1 || got[0].Observed != 10 {
		t.Errorf("Length problem = %+v, want one with Observed 10", got)
	}
}
