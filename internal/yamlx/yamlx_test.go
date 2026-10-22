// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package yamlx

import (
	"reflect"
	"strings"
	"testing"

	"metais.org/metais/internal/finding"
	"metais.org/metais/internal/limits"
	"metais.org/metais/internal/liquid/value"
	"metais.org/metais/internal/testenv"
)

// TestResolution pins every row of ADR-002 D2 item 9's table, in both
// families: what a plain scalar resolves to, or the form it is refused
// as. The library's Tag would make 0777 the integer 511 and 2026-01-02 a
// timestamp; the profile makes the one refused and the other a string.
func TestResolution(t *testing.T) {
	cases := []struct {
		scalar string
		want   value.Value // nil when refused
		form   string
	}{
		// empty and null
		{"", value.Null{}, ""},
		{"null", value.Null{}, ""},
		// booleans
		{"true", value.Bool(true), ""},
		{"false", value.Bool(false), ""},
		// the RFC 8259 number grammar
		{"0", num("0"), ""},
		{"-0", num("-0"), ""},
		{"12", num("12"), ""},
		{"-7", num("-7"), ""},
		{"1.0", num("1.0"), ""},
		{"1.10", num("1.10"), ""},
		{"1e5", num("1e5"), ""},
		{"1E-5", num("1E-5"), ""},
		{"-1.5e+3", num("-1.5e+3"), ""},
		// other spellings of null and the booleans
		{"~", nil, formNull},
		{"Null", nil, formNull},
		{"NULL", nil, formNull},
		{"True", nil, formBoolean},
		{"TRUE", nil, formBoolean},
		{"False", nil, formBoolean},
		{"FALSE", nil, formBoolean},
		// other Core Schema int and float forms
		{"+1", nil, formNumber},
		{"012", nil, formNumber},
		{"0777", nil, formNumber},
		{"-012", nil, formNumber},
		{"00", nil, formNumber},
		{"0o17", nil, formNumber},
		{"0x1F", nil, formNumber},
		{"0xff", nil, formNumber},
		{".5", nil, formNumber},
		{"-.5", nil, formNumber},
		{"1.", nil, formNumber},
		{"+1.5e3", nil, formNumber},
		{"01.5", nil, formNumber},
		{".inf", nil, formNumber},
		{"-.Inf", nil, formNumber},
		{"+.INF", nil, formNumber},
		{".nan", nil, formNumber},
		{".NaN", nil, formNumber},
		{".NAN", nil, formNumber},
		// anything else is a string
		{"no", str("no"), ""},
		{"NO", str("NO"), ""},
		{"yes", str("yes"), ""},
		{"on", str("on"), ""},
		{"off", str("off"), ""},
		{"y", str("y"), ""},
		{"2026-01-02", str("2026-01-02"), ""},
		{"2026-10-18", str("2026-10-18"), ""},
		{"2026-10-18T12:30:00Z", str("2026-10-18T12:30:00Z"), ""},
		{"12:30", str("12:30"), ""},
		{"1_000", str("1_000"), ""},
		{"1.0.0", str("1.0.0"), ""},
		{"0b101", str("0b101"), ""},
		{"0X1F", str("0X1F"), ""},
		{"0o8", str("0o8"), ""},
		{"1e", str("1e"), ""},
		{"inf", str("inf"), ""},
		{".Nan", str(".Nan"), ""},
		{"nULL", str("nULL"), ""},
		{"tRUE", str("tRUE"), ""},
		{"Status widget", str("Status widget"), ""},
	}
	for _, tc := range cases {
		for _, src := range []Source{manifestSrc, inputSrc} {
			doc, fs := parse(t, src, "v: "+tc.scalar+"\n")
			if tc.want == nil {
				if len(fs) != 1 || fs[0].Code() != codeOf(src, ruleScalarRefused) {
					t.Errorf("%s %q: %s, want %s", src.Subject.Kind, tc.scalar, describe(fs), codeOf(src, ruleScalarRefused))
					continue
				}
				if form, _ := fs[0].Param("form"); !reflect.DeepEqual(form, finding.String(tc.form)) {
					t.Errorf("%s %q: form %v, want %s", src.Subject.Kind, tc.scalar, form, tc.form)
				}
				continue
			}
			if doc.Value == nil {
				t.Errorf("%s %q: refused: %s", src.Subject.Kind, tc.scalar, describe(fs))
				continue
			}
			got, _ := doc.Value.(value.Object).Get("v")
			if !value.Equal(got, tc.want) || got.Kind() != tc.want.Kind() {
				t.Errorf("%s %q: %#v, want %#v", src.Subject.Kind, tc.scalar, got, tc.want)
			}
			if n, ok := got.(value.Number); ok && n.Literal() != tc.scalar {
				t.Errorf("%s %q: literal %q", src.Subject.Kind, tc.scalar, n.Literal())
			}
		}
	}
}

// TestQuotedAndBlockScalarsAreStrings: quoting or a block scalar makes
// any text a string, the forms the table refuses included.
func TestQuotedAndBlockScalarsAreStrings(t *testing.T) {
	doc, fs := parse(t, inputSrc, "a: '012'\nb: \"0777\"\nc: \"~\"\nd: 'true'\ne: |\n  .inf\nf: >-\n  null\n  x\ng: \"\"\n")
	if fs != nil {
		t.Fatal(describe(fs))
	}
	want := obj("a", str("012"), "b", str("0777"), "c", str("~"), "d", str("true"), "e", str(".inf\n"), "f", str("null x"), "g", str(""))
	if !value.Equal(doc.Value, want) {
		t.Errorf("%#v", doc.Value)
	}
}

// TestKeysAreText pins ADR-002 D2 item 7: a key is its scalar's text,
// with no type resolution, plain or quoted.
func TestKeysAreText(t *testing.T) {
	doc, fs := parse(t, manifestSrc, "on: 1\n012: 2\ntrue: 3\n~: 4\nnull: 5\n\"\": 6\n'0x1F': 7\n2026-01-02: 8\n")
	if fs != nil {
		t.Fatal(describe(fs))
	}
	want := obj("on", num("1"), "012", num("2"), "true", num("3"), "~", num("4"), "null", num("5"), "", num("6"), "0x1F", num("7"), "2026-01-02", num("8"))
	if !value.Equal(doc.Value, want) {
		t.Errorf("%#v", doc.Value)
	}
}

// codeOf is rule's code in src's family.
func codeOf(src Source, ru rule) finding.Code {
	if src.Subject.Kind == finding.SubjectManifest {
		return codes[ru][0]
	}
	return codes[ru][1]
}

// TestRefusals is W2-7.1: every construct ADR-002 D2 and ADR-010 D6 item
// 2 refuse, in both families, each with its position and, where the
// finding is about a value or member, its pointer.
func TestRefusals(t *testing.T) {
	cases := []struct {
		name, text string
		rule       rule
		want       string // describe's line, with the code left out
	}{
		{"a second document", "a: 1\n---\nb: 2\n", ruleMultiDocument, "2:1"},
		{"an empty second document", "a: 1\n---\n", ruleMultiDocument, "2:1"},
		{"a second document after an end marker", "a: 1\n...\n---\nb: 2\n", ruleMultiDocument, "3:1"},
		{"a %YAML directive", "%YAML 1.2\n---\na: 1\n", ruleDirective, "1:1"},
		{"a %YAML 1.1 directive", "%YAML 1.1\n---\na: 1\n", ruleDirective, "1:1"},
		{"a %TAG directive after a comment", "# c\n%TAG ! tag:x,2000:\n---\na: 1\n", ruleDirective, "2:1"},
		{"an alias", "a: &x 1\nb: *x\n", ruleAnchor, "1:4 \"/a\"\nalias 2:4 \"/b\""},
		{"an alias as a key", "a: &x k\n*x : 2\n", ruleAnchor, "1:4 \"/a\"\nalias 2:1 \"\""},
		{"an anchor on the root", "--- &x\na: 1\n", ruleAnchor, `1:5 ""`},
		{"an anchor on the first key", "&x a: 1\n", ruleAnchor, `1:1 ""`},
		{"an anchor", "a: &x 1\n", ruleAnchor, `1:4 "/a"`},
		{"an anchor on a mapping", "a: &x\n  b: 1\n", ruleAnchor, `1:4 "/a"`},
		{"an anchor on a later key", "a: 1\n&x b: 2\n", ruleAnchor, `2:1 ""`},
		{"a !!str tag", "a: !!str 1\n", ruleTag, `1:4 "/a"`},
		{"the non-specific tag", "a: ! x\n", ruleTag, `1:4 "/a"`},
		{"a local tag", "a: !foo y\n", ruleTag, `1:4 "/a"`},
		{"a verbatim tag", "a: !<tag:yaml.org,2002:str> y\n", ruleTag, `1:4 "/a"`},
		{"a tag on a sequence", "a: !!seq\n  - 1\n", ruleTag, `1:4 "/a"`},
		{"a tag on the root", "--- !!map\na: 1\n", ruleTag, `1:5 ""`},
		{"a tag in a flow sequence", "a: [1, ! 2]\n", ruleTag, `1:8 "/a/1"`},
		{"a tag on the first key", "!!str a: 1\n", ruleTag, `1:1 ""`},
		{"the non-specific tag on the first key", "! a: 1\n", ruleTag, `1:1 ""`},
		{"a tag on a later key", "a: 1\n!t b: 2\n", ruleTag, `2:1 ""`},
		{"a tag after an anchor", "a: &x !t 1\n", ruleAnchor, "1:4 \"/a\"\ntag 1:4 \"/a\""},
		{"a tag on a line after its anchor", "a: &x # c\n  ! 1\n", ruleAnchor, "1:4 \"/a\"\ntag 1:4 \"/a\""},
		{"a merge key", "<<: {a: 1}\n", ruleMergeKey, `1:1 ""`},
		{"a merge key in a nested mapping", "a:\n  <<: {b: 1}\n", ruleMergeKey, `2:3 "/a"`},
		{"a complex key", "? [a]\n: 1\n", ruleKeyNotScalar, `1:3 ""`},
		{"a mapping key", "? {a: 1}\n: 1\n", ruleKeyNotScalar, `1:3 ""`},
		{"a flow sequence key", "{[a]: 1}\n", ruleKeyNotScalar, `1:2 ""`},
		{"an empty key", "? \n: 1\n", ruleKeyEmpty, `1:2 ""`},
		{"a duplicate key", "a: 1\nb: 2\na: 3\n", ruleDuplicateKey, `3:1 "/a"`},
		{"a duplicate of a quoted key", "a: 1\n\"a\": 2\n", ruleDuplicateKey, `2:1 "/a"`},
		{"a duplicate in a nested mapping", "b:\n  'x y': 3\n  x y: 4\n", ruleDuplicateKey, `3:3 "/b/x y"`},
		{"a duplicate in a flow mapping", "{a: 1, a: 2}\n", ruleDuplicateKey, `1:8 "/a"`},
		{"a byte order mark", "\ufeffa: 1\n", ruleBOM, "1:1"},
		{"a C0 control", "a: 1\nb: x\x01\n", ruleCharacter, "2:5"},
		{"DEL", "a: \x7f\n", ruleCharacter, "1:4"},
		{"a C1 control", "a: \u0080\n", ruleCharacter, "1:4"},
		{"U+FFFE raw", "a: é\uFFFE\n", ruleCharacter, "1:5"},
		{"NEL raw", "a: x\u0085y\n", ruleCharacter, "1:5"},
		{"LS raw", "a: \"x\u2028y\"\n", ruleCharacter, "1:6"},
		{"PS raw", "a: x\u2029\n", ruleCharacter, "1:5"},
		{"invalid UTF-8", "a: 1\nb: é\xff\n", ruleEncoding, "2:5"},
		{"UTF-16", "\xfe\xff\x00a\x00:\x00 \x001\n", ruleEncoding, "1:1"},
		{"a plain ~", "a: ~\n", ruleScalarRefused, `1:4 "/a"`},
		{"a leading zero", "a:\n  - 0777\n", ruleScalarRefused, `2:5 "/a/0"`},
		{"an overflowing number", "a: 1e400\n", ruleNumberRange, `1:4 "/a"`},
		{"an integer beyond 2^53-1", "a: [9007199254740992]\n", ruleNumberRange, `1:5 "/a/0"`},
		{"a negative integer beyond 2^53-1", "a: -9007199254740993\n", ruleNumberRange, `1:4 "/a"`},
		{"a syntax error", "a: 1\nb: c: d\n", ruleSyntax, "2:0"},
		{"a syntax error on line 1", "a: b: c\n", ruleSyntax, "1:0"},
		{"an unclosed flow mapping", "x: 1\ny: 2\nz: {a: 1\n", ruleSyntax, "3:0"},
		{"an unclosed quote", "a: 1\n\nb: \"x\n", ruleSyntax, "3:0"},
		{"a tab as indentation", "a:\n\tb: 1\n", ruleSyntax, "2:0"},
		{"an empty flow key", "{: 1}\n", ruleSyntax, "1:0"},
	}
	for _, tc := range cases {
		for _, src := range []Source{manifestSrc, inputSrc} {
			_, fs := parse(t, src, tc.text)
			want := string(codeOf(src, tc.rule)) + " " + tc.want + "\n"
			for name, ru := range map[string]rule{"alias": ruleAlias, "tag": ruleTag} {
				want = strings.ReplaceAll(want, "\n"+name+" ", "\n"+string(codeOf(src, ru))+" ")
			}
			if got := describe(fs); got != want {
				t.Errorf("%s, %s: got\n%swant\n%s", tc.name, src.Subject.Kind, got, want)
			}
		}
	}
}

// TestFamilyRules pins the rules that differ by family: the manifest's
// root is a mapping (ADR-002 D2 item 3) and its numbers are held to
// ADR-002 D4 item 3; an input's root may be any value, its numbers are
// held to ADR-003 D8 item 5, and its strings hold no noncharacter (D8
// item 4).
func TestFamilyRules(t *testing.T) {
	cases := []struct {
		name, text, manifest, input string
	}{
		{"a sequence root", "- a\n", "manifest.yaml.root_not_mapping 1:1 \"\"\n", ""},
		{"a scalar root", "x\n", "manifest.yaml.root_not_mapping 1:1 \"\"\n", ""},
		{"an empty document", "", "manifest.yaml.root_not_mapping 0:0 \"\"\n", ""},
		{"comments only", "# a\n# b\n", "manifest.yaml.root_not_mapping 0:0 \"\"\n", ""},
		{"an empty document with a start marker", "---\n", "manifest.yaml.root_not_mapping 2:1 \"\"\n", ""},
		{"an integral value beyond 2^53-1 written with an exponent", "a: 1e300\n", "manifest.number_out_of_range 1:4 \"/a\"\n", ""},
		{"an integral value beyond 2^53-1 written with a fraction", "a: 9007199254740992.0\n", "manifest.number_out_of_range 1:4 \"/a\"\n", ""},
		{"an escaped noncharacter", "a: \"\\uFFFE\"\n", "", "yaml.character 1:4 \"/a\"\n"},
		{"a raw noncharacter", "a: x\uFDD0\n", "", "yaml.character 1:4 \"/a\"\n"},
		{"a noncharacter in a key", "\"\\U0010FFFF\": 1\n", "", `yaml.character 1:1 "/\U0010ffff"` + "\n"},
	}
	for _, tc := range cases {
		for _, src := range []Source{manifestSrc, inputSrc} {
			_, fs := parse(t, src, tc.text)
			want := tc.input
			if src.Subject.Kind == finding.SubjectManifest {
				want = tc.manifest
			}
			if got := describe(fs); got != want {
				t.Errorf("%s, %s: got\n%swant\n%s", tc.name, src.Subject.Kind, got, want)
			}
		}
	}
	doc, _ := parse(t, inputSrc, "")
	if doc.Value == nil || !value.Equal(doc.Value, value.Null{}) {
		t.Errorf("an empty input is %#v, want null (ADR-003 D10 item 4)", doc)
	}
	_, fs := parse(t, inputSrc, "a: \"\\uFFFE\"\n")
	if r, _ := fs[0].Param("reason"); !reflect.DeepEqual(r, finding.String(reasonNoncharacter)) {
		t.Errorf("reason %v", r)
	}
	if c, _ := fs[0].Param("codepoint"); !reflect.DeepEqual(c, finding.String("U+FFFE")) {
		t.Errorf("codepoint %v", c)
	}
}

// TestDuplicateKeyNamesTheFirst: each duplicate is reported at its own
// line, with the first occurrence related (ADR-002 D2 item 8).
func TestDuplicateKeyNamesTheFirst(t *testing.T) {
	_, fs := parse(t, manifestSrc, "a: 1\nb: 2\na: 3\n'a': 4\n")
	if got, want := describe(fs), "manifest.yaml.duplicate_key 3:1 \"/a\"\nmanifest.yaml.duplicate_key 4:1 \"/a\"\n"; got != want {
		t.Fatalf("got\n%swant\n%s", got, want)
	}
	for _, f := range fs {
		rel := f.Related()
		if len(rel) != 1 || rel[0].Note != noteFirst || rel[0].Location.Line != 1 || rel[0].Location.Column != 1 || rel[0].Location.Pointer != "/a" {
			t.Errorf("related %+v", rel)
		}
	}
}

// TestEveryViolationInOnePass is ADR-002 D2 item 11: a syntactically
// valid document's violations are all reported, in one pass.
func TestEveryViolationInOnePass(t *testing.T) {
	text := "%TAG ! tag:x,2000:\n---\na: &x 1\nb: *x\nc: !t 2\n<<: {}\n? [k]\n: 3\nd: 4\nd: ~\ne: [012, .inf]\n---\nf: 5\n"
	_, fs := parse(t, inputSrc, text)
	want := strings.Join([]string{
		"yaml.directive 1:1",
		"yaml.multi_document 12:1",
		`yaml.anchor 3:4 "/a"`,
		`yaml.alias 4:4 "/b"`,
		`yaml.tag 5:4 "/c"`,
		`yaml.merge_key 6:1 ""`,
		`yaml.key_not_scalar 7:3 ""`,
		`yaml.duplicate_key 10:1 "/d"`,
		`yaml.scalar_refused 10:4 "/d"`,
		`yaml.scalar_refused 11:5 "/e/0"`,
		`yaml.scalar_refused 11:10 "/e/1"`,
	}, "\n") + "\n"
	if got := describe(fs); got != want {
		t.Errorf("got\n%swant\n%s", got, want)
	}
}

// TestNoEcho: no finding carries the library's message or the text of
// the document (ADR-006 D9 item 4).
func TestNoEcho(t *testing.T) {
	for _, text := range []string{"secret: s3cr3t: x\n", "s3cr3t: [\n", "a: !s3cr3t x\n", "a: s3cr3t\na: s3cr3t\n", "s3cr3t: 0x5ec\n"} {
		for _, src := range []Source{manifestSrc, inputSrc} {
			_, fs := parse(t, src, text)
			for _, f := range fs {
				for _, s := range []string{f.Message(), f.Hint()} {
					if strings.Contains(s, "s3cr3t") || strings.Contains(s, "mapping values") || strings.Contains(s, "yaml:") {
						t.Errorf("%q: %s echoes: %s", text, f.Code(), s)
					}
				}
			}
		}
	}
}

// TestPathPositions: a value and a member name read by their Path are
// where Value and Key say they are, read once or again, in any order.
func TestPathPositions(t *testing.T) {
	doc, fs := parse(t, inputSrc, "éé: ü\nliste:\n  - ä\n  - {ö: 1, \"ß\": [x], \"a/~\": 2}\n")
	if fs != nil {
		t.Fatal(describe(fs))
	}
	pos := doc.Positions().(finding.PathPositions)
	ptrs := []string{"/liste/1/ß/0", "", "/éé", "/liste", "/liste/0", "/liste/1", "/liste/1/ö", "/liste/1/ß", "/liste/1/a~1~0", "/nope", "/liste/2", "/éé/x"}
	for range 2 {
		for _, ptr := range ptrs {
			p, ok := finding.ParsePath(ptr)
			if !ok {
				t.Fatalf("%q is not a pointer", ptr)
			}
			wantV, wantOK := pos.Value(ptr)
			if v, ok := pos.ValueAt(p); v != wantV || ok != wantOK {
				t.Errorf("value %q at %+v (%v), want %+v (%v)", ptr, v, ok, wantV, wantOK)
			}
			wantK, wantOK := pos.Key(ptr)
			if k, ok := pos.KeyAt(p); k != wantK || ok != wantOK {
				t.Errorf("key %q at %+v (%v), want %+v (%v)", ptr, k, ok, wantK, wantOK)
			}
		}
	}
}

// TestPositions is W2-7.3: every value and every member name keeps its
// line, and its column in code points, whatever the line endings.
func TestPositions(t *testing.T) {
	for _, eol := range []string{"\n", "\r\n", "\r"} {
		text := strings.ReplaceAll("éé: ü\nliste:\n  - ä\n  - {ö: 1, \"ß\": [x]}\n  # comment\nlast: |\n  text\n", "\n", eol)
		doc, fs := parse(t, inputSrc, text)
		if fs != nil {
			t.Fatalf("%q: %s", eol, describe(fs))
		}
		values := map[string]finding.Pos{
			"":             {Line: 1, Column: 1},
			"/éé":          {Line: 1, Column: 5},
			"/liste":       {Line: 3, Column: 3},
			"/liste/0":     {Line: 3, Column: 5},
			"/liste/1":     {Line: 4, Column: 5},
			"/liste/1/ö":   {Line: 4, Column: 9},
			"/liste/1/ß":   {Line: 4, Column: 17},
			"/liste/1/ß/0": {Line: 4, Column: 18},
			"/last":        {Line: 6, Column: 7},
		}
		keys := map[string]finding.Pos{
			"/éé":        {Line: 1, Column: 1},
			"/liste":     {Line: 2, Column: 1},
			"/liste/1/ö": {Line: 4, Column: 6},
			"/liste/1/ß": {Line: 4, Column: 12},
			"/last":      {Line: 6, Column: 1},
		}
		for ptr, want := range values {
			if got, ok := doc.Positions().Value(ptr); !ok || got != want {
				t.Errorf("%q: value %q at %v, want %v", eol, ptr, got, want)
			}
		}
		for ptr, want := range keys {
			if got, ok := doc.Positions().Key(ptr); !ok || got != want {
				t.Errorf("%q: key %q at %v, want %v", eol, ptr, got, want)
			}
		}
		if all, allKeys := doc.pos.entries(); len(all) != len(values) || len(allKeys) != len(keys) {
			t.Errorf("%q: %d values and %d keys, want %d and %d", eol, len(all), len(allKeys), len(values), len(keys))
		}
		if _, ok := doc.Positions().Key("/liste/0"); ok {
			t.Errorf("an element has a key position")
		}
	}
}

// TestFirstLine: a frontmatter's positions count from its file's first
// line (ADR-006 D5 item 6), its findings' too.
func TestFirstLine(t *testing.T) {
	src := inputSrc
	src.FirstLine = 2
	doc, fs := parse(t, src, "a: 1\nb: [x]\n")
	if fs != nil {
		t.Fatal(describe(fs))
	}
	if p, _ := doc.Positions().Value("/b/0"); p != (finding.Pos{Line: 3, Column: 5}) {
		t.Errorf("/b/0 at %v", p)
	}
	if p, _ := doc.Positions().Key("/a"); p != (finding.Pos{Line: 2, Column: 1}) {
		t.Errorf("key /a at %v", p)
	}
	_, fs = parse(t, src, "a: 1\nb: ~\n")
	if got, want := describe(fs), "yaml.scalar_refused 3:4 \"/b\"\n"; got != want {
		t.Errorf("got %s want %s", got, want)
	}
	_, fs = parse(t, src, "a: 1\nb: c: d\n")
	if got, want := describe(fs), "yaml.syntax 3:0\n"; got != want {
		t.Errorf("got %s want %s", got, want)
	}
}

// TestFindingsCarryTheSource: a finding names the Source's subject,
// origin and file.
func TestFindingsCarryTheSource(t *testing.T) {
	src := inputSrc
	src.Subject.Title = "Vendor"
	_, fs := parse(t, src, "a: ~\n")
	l, _ := fs[0].Location()
	if s := fs[0].Subject(); s != src.Subject || l.Origin != finding.OriginDataDir || l.File != "vendor.yaml" {
		t.Errorf("subject %+v, location %+v", s, l)
	}
	_, fs = parse(t, manifestSrc, "a: ~\n")
	l, _ = fs[0].Location()
	if s := fs[0].Subject(); s.Kind != finding.SubjectManifest || l.Origin != finding.OriginPackage || l.File != "metais.yaml" {
		t.Errorf("subject %+v, location %+v", s, l)
	}
}

// nested is depth arrays, one inside the next, in flow style.
func nested(depth int) string {
	return "a: " + strings.Repeat("[", depth) + strings.Repeat("]", depth) + "\n"
}

// TestDepth is W2-7.4: value-depth is enforced on the node tree, at its
// bound and below the library's own 10,000, in flow and block style; a
// document past the library's 10,000 is reported as the limit too.
func TestDepth(t *testing.T) {
	if bound := limits.Get(limits.ValueDepth).Default; bound >= 10_000 {
		t.Fatalf("value-depth's default %d is not below the library's 10,000", bound)
	}
	// The root mapping is depth 1, so nested(63) is 64 deep.
	for _, src := range []Source{manifestSrc, inputSrc} {
		if _, fs := parse(t, src, nested(63)); fs != nil {
			t.Errorf("%s: 64 deep: %s", src.Subject.Kind, describe(fs))
		}
		_, fs := parse(t, src, nested(64))
		want := map[finding.SubjectKind]finding.Code{finding.SubjectManifest: finding.CodeLimitPackage, finding.SubjectInput: finding.CodeLimitExceeded}[src.Subject.Kind]
		if len(fs) != 1 || fs[0].Code() != want {
			t.Fatalf("%s: 65 deep: %s", src.Subject.Kind, describe(fs))
		}
		l, _ := fs[0].Location()
		if l.Line != 1 || l.Column != 67 || l.Pointer != "/a"+strings.Repeat("/0", 63) {
			t.Errorf("%s: 65 deep at %+v", src.Subject.Kind, l)
		}
		if v, _ := fs[0].Param("limit"); !reflect.DeepEqual(v, finding.String("value-depth")) {
			t.Errorf("limit %v", v)
		}
		if _, ok := fs[0].Param("observed"); ok {
			t.Error("the reading stopped at the bound, and observed is given")
		}
		_, fs = parse(t, src, strings.Repeat("[", 10_001)+strings.Repeat("]", 10_001))
		if len(fs) != 1 || fs[0].Code() != want {
			t.Errorf("%s: 10,001 deep: %s", src.Subject.Kind, describe(fs))
		}
	}
	lim := lowered(t, "value-depth=2")
	block := "a:\n  b:\n    - 1\n"
	_, fs := parseLim(t, inputSrc, block, lim)
	if got, want := describe(fs), "limit.exceeded 3:5 \"/a/b\"\n"; got != want {
		t.Errorf("block, depth 2: got %swant %s", got, want)
	}
	if b, _ := fs[0].Param("lowered"); !reflect.DeepEqual(b, finding.Bool(true)) {
		t.Errorf("lowered %v", b)
	}
	if _, fs := parseLim(t, inputSrc, "a:\n  b: 1\n", lim); fs != nil {
		t.Errorf("depth 2: %s", describe(fs))
	}
	if _, fs := parseLim(t, inputSrc, "[[1]]", lim); fs != nil {
		t.Errorf("an input's root array, depth 2: %s", describe(fs))
	}
}

// TestSizeBeforeParsing: the size limit is checked before the document
// is parsed, so a document over it is refused with the limit alone, its
// observed size given, whatever else is wrong with it.
func TestSizeBeforeParsing(t *testing.T) {
	text := "a: [\n" + strings.Repeat("#", 100)
	for _, tc := range []struct {
		src  Source
		arg  string
		code finding.Code
	}{
		{manifestSrc, "manifest-size=100", finding.CodeLimitPackage},
		{inputSrc, "input-size=100", finding.CodeLimitExceeded},
	} {
		lim := lowered(t, tc.arg)
		_, fs := parseLim(t, tc.src, text, lim)
		if len(fs) != 1 || fs[0].Code() != tc.code {
			t.Fatalf("%s: %s", tc.arg, describe(fs))
		}
		if v, _ := fs[0].Param("observed"); !reflect.DeepEqual(v, finding.Int(int64(len(text)))) {
			t.Errorf("observed %v", v)
		}
		if l, _ := fs[0].Location(); l.Line != 0 || l.File != tc.src.At.File {
			t.Errorf("location %+v", l)
		}
		if _, fs := parseLim(t, tc.src, text[:100], lim); len(fs) != 1 || fs[0].Code() != codeOf(tc.src, ruleSyntax) {
			t.Errorf("%s, at the bound: %s", tc.arg, describe(fs))
		}
	}
	src := inputSrc
	src.Size = limits.FrontmatterSize
	if _, fs := parseLim(t, src, text, lowered(t, "frontmatter-size=100")); len(fs) != 1 || fs[0].Code() != finding.CodeLimitExceeded {
		t.Errorf("frontmatter-size: %s", describe(fs))
	}
}

// TestNumberLength: number-length bounds a numeric literal.
func TestNumberLength(t *testing.T) {
	lim := lowered(t, "number-length=4")
	if _, fs := parseLim(t, inputSrc, "a: -1.5\n", lim); fs != nil {
		t.Errorf("four bytes: %s", describe(fs))
	}
	_, fs := parseLim(t, inputSrc, "a: [-1.50]\n", lim)
	if got, want := describe(fs), "limit.exceeded 1:5 \"/a/0\"\n"; got != want {
		t.Fatalf("got %swant %s", got, want)
	}
	if v, _ := fs[0].Param("observed"); !reflect.DeepEqual(v, finding.Int(5)) {
		t.Errorf("observed %v", v)
	}
	// A string is not a numeric literal.
	if _, fs := parseLim(t, inputSrc, "a: 1.0.0.0\n", lim); fs != nil {
		t.Errorf("a string: %s", describe(fs))
	}
}

// TestValueTree: the data model a document gives, objects in canonical
// order, numbers keeping their literal text.
func TestValueTree(t *testing.T) {
	doc, fs := parse(t, manifestSrc, "z: 1.50\na:\n  - x\n  - {b: null, a: [true, 1e2]}\n\"\U0001F600\": 1\n\uFF61: 2\n")
	if fs != nil {
		t.Fatal(describe(fs))
	}
	want := obj("z", num("1.50"), "a", arr(str("x"), obj("b", value.Null{}, "a", arr(value.Bool(true), num("1e2")))), "\U0001F600", num("1"), "\uFF61", num("2"))
	if !value.Equal(doc.Value, want) {
		t.Errorf("%#v", doc.Value)
	}
	if got := doc.Value.(value.Object).Names(); !reflect.DeepEqual(got, []string{"a", "z", "\U0001F600", "\uFF61"}) {
		t.Errorf("member order %q", got)
	}
}

// TestPositionWorkIsLinear: finding the text at each node's position walks
// each line once, however many nodes, duplicates and properties share it,
// so the work is bounded by the text's length, counted rather than timed.
func TestPositionWorkIsLinear(t *testing.T) {
	testenv.Home(t)
	const n = 20_000
	for name, text := range map[string]string{
		"a long flow sequence":        "a: [" + strings.Repeat("é, ", n) + "x]\n",
		"duplicates on one line":      "{" + strings.Repeat("ä: 1, ", n) + "b: 2}\n",
		"tags on one line":            "[" + strings.Repeat("! x, ", n) + "y]\n",
		"anchors on one line":         "[" + strings.Repeat("&a x, ", n) + "y]\n",
		"a long block mapping":        strings.Repeat("ö: [1, {b: 2}]\n", n),
		"nested flow on one line":     "a: [" + strings.Repeat("[1, 2], ", n) + "3]\n",
		"long keys on one line":       "{" + strings.Repeat(strings.Repeat("k", 40)+"ü: 1, ", n) + "b: 2}\n",
		"empty values at the end":     "a:",
		"refused scalars on one line": "[" + strings.Repeat("~, ", n) + "y]\n",
	} {
		r := newReader(inputSrc, []byte(text), limits.Defaults())
		r.parse()
		if r.steps == 0 && name != "empty values at the end" {
			t.Errorf("%s: no step counted", name)
		}
		if r.steps > len(text) {
			t.Errorf("%s: %d steps over %d bytes", name, r.steps, len(text))
		}
	}
}

// TestReasons pins the reasons jsonx gives the same codes:
// input.encoding_invalid's and input.number_out_of_range's.
func TestReasons(t *testing.T) {
	for _, tc := range []struct {
		text, param, want string
	}{
		{"a: \xff\n", "reason", reasonInvalidUTF8},
		{"\xff\xfea\x00", "reason", reasonUTF16},
		{"\xfe\xff\x00a", "reason", reasonUTF16},
		{"a: 1e400\n", "reason", reasonOverflow},
		{"a: -9007199254740992\n", "reason", reasonUnsafeInteger},
	} {
		for _, src := range []Source{manifestSrc, inputSrc} {
			_, fs := parse(t, src, tc.text)
			if len(fs) != 1 {
				t.Fatalf("%q: %s", tc.text, describe(fs))
			}
			if v, _ := fs[0].Param(tc.param); !reflect.DeepEqual(v, finding.String(tc.want)) {
				t.Errorf("%q, %s: %s %v, want %s", tc.text, src.Subject.Kind, tc.param, v, tc.want)
			}
		}
	}
}
