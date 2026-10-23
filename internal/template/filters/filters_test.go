// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package filters

import (
	"strings"
	"testing"
	"unicode"

	"metais.org/metais/internal/finding"
	"metais.org/metais/internal/jsonx"
	"metais.org/metais/internal/limits"
	"metais.org/metais/internal/liquid"
	"metais.org/metais/internal/liquid/value"
	"metais.org/metais/internal/testenv"
	"metais.org/metais/internal/unitext"
)

// data reads src, JSON, as a render's globals.
func data(t *testing.T, src string) value.Object {
	t.Helper()
	if src == "" {
		return value.Object{}
	}
	doc, fs := jsonx.Parse([]byte(src), jsonx.Source{Input: finding.Subject{Kind: finding.SubjectInput, Name: "d"}}, limits.Defaults())
	if len(fs) > 0 {
		t.Fatalf("data %s: %v", src, fs)
	}
	return liquid.Data(doc.Value).(value.Object)
}

// render parses src and renders it with the globals of data, returning
// the text, or the first error and whether it was found at parse.
func render(t *testing.T, src, globals string) (out string, err *liquid.Error, atParse bool) {
	t.Helper()
	d := dialect
	tpl, errs := liquid.Parse(d, liquid.Source{Name: "t", Text: []byte(src)}, liquid.ParseLimits{})
	if len(errs) > 0 {
		return "", errs[0], true
	}
	var b strings.Builder
	e := liquid.Render(&b, tpl, &liquid.Env{Dialect: d, Globals: data(t, globals), Meter: liquid.NewMeter(liquid.MeterLimits{})})
	return b.String(), e, false
}

// filterCase is a template, its data, and what it must render or the
// error it must give.
type filterCase struct {
	name, src, data string
	want            string
	code            liquid.Code
	args            map[string]string // the error's Args beside filter, when set
	atParse         bool
}

func runCases(t *testing.T, cases []filterCase) {
	t.Helper()
	for _, c := range cases {
		out, err, atParse := render(t, c.src, c.data)
		switch {
		case c.code == "" && err != nil:
			t.Errorf("%s: %s: %v", c.name, c.src, err)
		case c.code == "" && out != c.want:
			t.Errorf("%s: %s\ngot  %q\nwant %q", c.name, c.src, out, c.want)
		case c.code != "" && (err == nil || err.Code != c.code):
			t.Errorf("%s: %s: got %q, %v; want %s", c.name, c.src, out, err, c.code)
		case c.code != "" && atParse != c.atParse:
			t.Errorf("%s: %s: found at parse %v, want %v: %v", c.name, c.src, atParse, c.atParse, err)
		case c.code != "":
			for k, v := range c.args {
				if err.Args[k] != v {
					t.Errorf("%s: %s: Args[%q] = %q, want %q (%v)", c.name, c.src, k, err.Args[k], v, err)
				}
			}
		}
	}
}

// TestJSON is ADR-004 D10 item 1.
func TestJSON(t *testing.T) {
	testenv.Home(t)
	const d = `{"v": {"b": [1, 2.50, "x\u0001\"", null, true], "a": {}, "é": 1e21, "c": []}}`
	runCases(t, []filterCase{
		{name: "compact, canonical order", src: `{{ v | json }}`, data: d, want: `{"a":{},"b":[1,2.5,"x\u0001\"",null,true],"c":[],"é":1e+21}`},
		{name: "indent", src: `{{ v.b | json: indent: 2 }}`, data: d, want: "[\n  1,\n  2.5,\n  \"x\\u0001\\\"\",\n  null,\n  true\n]"},
		{name: "indent, nested and empty", src: `{{ v | json: indent: 1 }}`, data: `{"v": {"a": {"b": []}}}`, want: "{\n \"a\": {\n  \"b\": []\n }\n}"},
		{name: "nil", src: `{{ nil | json }}`, want: "null"},
		{name: "a float literal", src: `{{ 3.0 | json }}`, want: "3"},
		{name: "a string", src: `{{ "a<b" | json }}`, want: `"a<b"`},
		{name: "indent 0", src: `{{ 1 | json: indent: 0 }}`, code: liquid.CodeFilterArgument, args: map[string]string{"argument": "indent"}, atParse: true},
		{name: "indent 9", src: `{{ 1 | json: indent: 9 }}`, code: liquid.CodeFilterArgument, atParse: true},
		{name: "indent a float", src: `{{ 1 | json: indent: 2.0 }}`, code: liquid.CodeFilterArgument, atParse: true},
		{name: "indent a string at render", src: `{{ 1 | json: indent: s }}`, data: `{"s": "2"}`, code: liquid.CodeFilterArgument, args: map[string]string{"argument": "indent"}},
		{name: "a positional argument", src: `{{ 1 | json: 2 }}`, code: liquid.CodeFilterArgument, args: map[string]string{"argument": "1"}, atParse: true},
		{name: "an unknown keyword", src: `{{ 1 | json: pretty: true }}`, code: liquid.CodeFilterArgument, args: map[string]string{"argument": "pretty"}, atParse: true},
	})
}

// TestYAML is ADR-004 D10 item 2.
func TestYAML(t *testing.T) {
	testenv.Home(t)
	runCases(t, []filterCase{
		{name: "a document", src: `{{ v | yaml }}`, data: `{"v": {"b": [1, {"x": "yes"}, []], "a": "plain text", "c": null}}`,
			want: "a: plain text\nb:\n  - 1\n  - x: \"yes\"\n  - []\nc: null"},
		{name: "a scalar", src: `{{ "a: b" | yaml }}`, want: `"a: b"`},
		{name: "an argument", src: `{{ 1 | yaml: 2 }}`, code: liquid.CodeFilterArgument, atParse: true},
	})
}

const rows = `{"rows": [{"name": "Ann", "n": 1, "note": "a, \"b\"\nc"}, {"name": "Bo|b", "n": 2.5, "note": null}]}`

// TestCSV is ADR-004 D10 item 3.
func TestCSV(t *testing.T) {
	testenv.Home(t)
	runCases(t, []filterCase{
		{name: "the union of names", src: `{{ rows | csv }}`, data: rows, want: "n,name,note\n1,Ann,\"a, \"\"b\"\"\nc\"\n2.5,Bo|b,\n"},
		{name: "named columns", src: `{{ rows | csv: "name", "n" }}`, data: rows, want: "name,n\nAnn,1\nBo|b,2.5\n"},
		{name: "headers", src: `{{ rows | csv: "name", "n", headers: h }}`, data: `{"h": ["Name", "N, count"], "rows": [{"name": "A", "n": 1}]}`, want: "Name,\"N, count\"\nA,1\n"},
		{name: "no header", src: `{{ rows | csv: "n", header: false }}`, data: rows, want: "1\n2.5\n"},
		{name: "a delimiter", src: `{{ rows | csv: "name", "n", delimiter: ";" }}`, data: `{"rows": [{"name": "a;b", "n": true}]}`, want: "name;n\n\"a;b\";true\n"},
		{name: "a union with a missing member", src: `{{ rows | csv }}`, data: `{"rows": [{"a": 1}, {"b": 2}]}`, want: "a,b\n1,\n,2\n"},
		{name: "an empty array with columns", src: `{{ rows | csv: "a" }}`, data: `{"rows": []}`, want: "a\n"},
		{name: "an empty array without columns", src: `{{ rows | csv }}`, data: `{"rows": []}`, want: ""},
		{name: "a missing named column", src: `{{ rows | csv: "name", "age" }}`, data: rows, code: liquid.CodeUndefinedProperty, args: map[string]string{"path": "age"}},
		{name: "an array cell", src: `{{ rows | csv }}`, data: `{"rows": [{"a": [1]}]}`, code: liquid.CodeFilterInput, args: map[string]string{"type": "array"}},
		{name: "an object cell", src: `{{ rows | csv }}`, data: `{"rows": [{"a": {}}]}`, code: liquid.CodeFilterInput, args: map[string]string{"type": "object"}},
		{name: "not an array", src: `{{ "x" | csv }}`, code: liquid.CodeFilterInput, args: map[string]string{"type": "string"}},
		{name: "nil", src: `{{ nil | csv }}`, code: liquid.CodeFilterInput, args: map[string]string{"type": "null"}},
		{name: "an element not an object", src: `{{ rows | csv }}`, data: `{"rows": [{}, 3]}`, code: liquid.CodeFilterInput, args: map[string]string{"type": "integer"}},
		{name: "a column not a string", src: `{{ rows | csv: "a", 2 }}`, data: rows, code: liquid.CodeFilterArgument, args: map[string]string{"argument": "2"}, atParse: true},
		{name: "headers short", src: `{{ rows | csv: "name", "n", headers: h }}`, data: `{"h": ["N"], "rows": []}`, code: liquid.CodeFilterArgument, args: map[string]string{"argument": "headers"}},
		{name: "headers without named columns", src: `{{ rows | csv: headers: h }}`, data: `{"h": [], "rows": []}`, code: liquid.CodeFilterArgument, args: map[string]string{"argument": "headers"}, atParse: true},
		{name: "headers not strings", src: `{{ rows | csv: "a", headers: h }}`, data: `{"h": [1], "rows": []}`, code: liquid.CodeFilterArgument, args: map[string]string{"argument": "headers"}},
		{name: "headers a literal", src: `{{ rows | csv: "a", headers: "A" }}`, data: rows, code: liquid.CodeFilterArgument, atParse: true},
		{name: "header not a boolean", src: `{{ rows | csv: header: "no" }}`, data: rows, code: liquid.CodeFilterArgument, args: map[string]string{"argument": "header"}, atParse: true},
		{name: "a quote delimiter", src: `{{ rows | csv: delimiter: '"' }}`, data: rows, code: liquid.CodeFilterArgument, args: map[string]string{"argument": "delimiter"}, atParse: true},
		{name: "a two-character delimiter", src: `{{ rows | csv: delimiter: ";;" }}`, data: rows, code: liquid.CodeFilterArgument, atParse: true},
		{name: "a one-character delimiter of two bytes", src: `{{ rows | csv: "a", delimiter: "§" }}`, data: `{"rows": [{"a": "x§y"}]}`, want: "a\n\"x§y\"\n"},
	})
}

// TestMarkdownTable is ADR-004 D10 item 4.
func TestMarkdownTable(t *testing.T) {
	testenv.Home(t)
	runCases(t, []filterCase{
		{name: "the union of names", src: `{{ rows | markdown_table }}`, data: rows,
			want: "| n | name | note |\n| --- | --- | --- |\n| 1 | Ann | a, \"b\"<br>c |\n| 2.5 | Bo\\|b |  |\n"},
		{name: "headers and line breaks", src: `{{ rows | markdown_table: "a", headers: h }}`, data: `{"h": ["A|B"], "rows": [{"a": "1\r\n2\r3\n4 *x*"}]}`,
			want: "| A\\|B |\n| --- |\n| 1<br>2<br>3<br>4 *x* |\n"},
		{name: "an empty array with columns", src: `{{ rows | markdown_table: "a", "b" }}`, data: `{"rows": []}`, want: "| a | b |\n| --- | --- |\n"},
		{name: "no columns", src: `{{ rows | markdown_table }}`, data: `{"rows": [{}]}`, want: ""},
		{name: "a missing named column", src: `{{ rows | markdown_table: "x" }}`, data: rows, code: liquid.CodeUndefinedProperty, args: map[string]string{"path": "x"}},
		{name: "csv's keywords", src: `{{ rows | markdown_table: header: false }}`, data: rows, code: liquid.CodeFilterArgument, args: map[string]string{"argument": "header"}, atParse: true},
		{name: "not an array", src: `{{ 1 | markdown_table }}`, code: liquid.CodeFilterInput, args: map[string]string{"type": "integer"}},
	})
}

// TestSlugLettersDoNotDecompose is ADR-004 Amendment A1's premise: no
// letter of the table decomposes under NFKD, so reading the table after
// NFKD gives each of them the spelling it had when the table was read
// first.
func TestSlugLettersDoNotDecompose(t *testing.T) {
	for r := range slugLetters {
		if d := unitext.NFKD(string(r)); d != string(r) {
			t.Errorf("U+%04X decomposes to %+q", r, d)
		}
	}
}

// TestSlug is ADR-004 D10 item 6.
func TestSlug(t *testing.T) {
	testenv.Home(t)
	runCases(t, []filterCase{
		{name: "the specification's example", src: `{{ "Café Déjà-vu" | slug }}`, want: "cafe-deja-vu"},
		{name: "the letters with no decomposition", src: `{{ "Straße ẞ Æsir æ Œuvre œ Øre ø Đ đ Ð ð Þ þ Łódź ł ı Ħ ħ Ŧ ŧ Ŋ ŋ ĸ" | slug }}`,
			want: "strasse-ss-aesir-ae-oeuvre-oe-ore-o-d-d-d-d-th-th-lodz-l-i-h-h-t-t-ng-ng-k"},
		{name: "letters that reach the table through decomposition", src: `{{ "Ǿrsted ǿ Ǽsir ǽ Ǣ ǣ" | slug }}`,
			want: "orsted-o-aesir-ae-ae-ae"},
		{name: "compatibility forms", src: `{{ "ﬁle ① ½ Ｆｕｌｌ" | slug }}`, want: "file-1-1-2-full"},
		{name: "runs and ends", src: `{{ "  --Hello__World!!2026--  " | slug }}`, want: "hello-world-2026"},
		{name: "a letter with no ASCII form among others", src: `{{ "Ελλάδα 2026" | slug }}`, want: "2026"},
		{name: "nil", src: `{{ nil | slug | json }}`, want: "null"},
		{name: "nothing ASCII", src: `{{ "日本語" | slug }}`, code: liquid.CodeSlugEmpty},
		{name: "empty", src: `{{ "" | slug }}`, code: liquid.CodeSlugEmpty},
		{name: "a number", src: `{{ 42 | slug }}`, code: liquid.CodeFilterInput, args: map[string]string{"type": "integer"}},
		{name: "an argument", src: `{{ "a" | slug: "-" }}`, code: liquid.CodeFilterArgument, atParse: true},
	})
}

// TestSlugStepByStep holds slug, which decomposes one code point at a
// time, to ADR-004 D10 item 6's steps over the whole string: the table,
// then NFKD of the whole, whose canonical reordering moves combining
// marks across code points.
func TestSlugStepByStep(t *testing.T) {
	testenv.Home(t)
	whole := func(s string) string {
		for from, to := range slugLetters {
			s = strings.ReplaceAll(s, string(from), to)
		}
		var b strings.Builder
		dash := false
		for _, r := range unitext.NFKD(s) {
			if unicode.Is(unicode.Mn, r) {
				continue
			}
			if 'A' <= r && r <= 'Z' {
				r += 'a' - 'A'
			}
			if ('a' <= r && r <= 'z') || ('0' <= r && r <= '9') {
				if dash && b.Len() > 0 {
					b.WriteByte('-')
				}
				dash = false
				b.WriteRune(r)
				continue
			}
			dash = true
		}
		return b.String()
	}
	for _, s := range []string{
		"Café Déjà-vu", "q\u0307\u0323x", "a\u0328\u0301b", "Å\u0316\u0300z", "ǅemal ǈ", "한국어 abc", "ﷺ 1 ﷺ",
		"x\u0903y", "ⅷ ¼ ㎒ ﬁ", "Straße Øre ı", "-", "", "\u0301\u0301a", "a\u20dd-\u20ddb", "Ｆｕｌｌ\u3000Width",
	} {
		var b []byte
		slug(s, func(c byte) { b = append(b, c) })
		if got, want := string(b), whole(s); got != want {
			t.Errorf("slug(%q) = %q, the steps over the whole string give %q", s, got, want)
		}
	}
}

// TestDate is ADR-004 D11 through the dialect: the filter of
// internal/dates, registered here.
func TestDate(t *testing.T) {
	testenv.Home(t)
	runCases(t, []filterCase{
		{name: "a full-date", src: `{{ "2026-10-18" | date: "%Y-%m-%d %H:%M %Z" }}`, want: "2026-10-18 00:00 UTC"},
		{name: "a date-time in its own offset", src: `{{ "2026-10-18T09:30:00.25+13:45" | date: "%c %z %:z %Z %L" }}`, want: "Sun Oct 18 09:30:00 2026 +1345 +13:45 +13:45 250"},
		{name: "Unix seconds", src: `{{ 0 | date: "%H:%M %z" }}`, want: "00:00 +0000"},
		{name: "flags", src: `{{ "2026-01-05T07:08:09Z" | date: "%-d %_m %0e %^a %^B" }}`, want: "5  1 05 MON JANUARY"},
		{name: "nil", src: `{{ nil | date: "%Y" | json }}`, want: "null"},
		{name: "now", src: `{{ "now" | date: "%s" }}`, code: liquid.CodeDateInput, args: map[string]string{"type": "string", "now": "true"}},
		{name: "today", src: `{{ "today" | date: "%s" }}`, code: liquid.CodeDateInput, args: map[string]string{"now": "true"}},
		{name: "other text", src: `{{ "March 14, 2016" | date: "%s" }}`, code: liquid.CodeDateInput, args: map[string]string{"type": "string", "now": ""}},
		{name: "a float", src: `{{ 0.0 | date: "%s" }}`, code: liquid.CodeDateInput, args: map[string]string{"type": "float"}},
		{name: "an array", src: `{{ a | date: "%s" }}`, data: `{"a": []}`, code: liquid.CodeDateInput, args: map[string]string{"type": "array"}},
		{name: "second 60", src: `{{ "2016-12-31T23:59:60Z" | date: "%s" }}`, code: liquid.CodeDateInput},
		{name: "an unknown directive", src: `{{ 0 | date: "%Y %Q" }}`, code: liquid.CodeDateFormat, args: map[string]string{"directive": "%Q"}, atParse: true},
		{name: "an empty format", src: `{{ 0 | date: "" }}`, code: liquid.CodeDateFormat, atParse: true},
		{name: "an unknown directive at render", src: `{{ 0 | date: f }}`, data: `{"f": "%E"}`, code: liquid.CodeDateFormat, args: map[string]string{"directive": "%E"}},
		{name: "a format not a string", src: `{{ 0 | date: 5 }}`, code: liquid.CodeFilterArgument, args: map[string]string{"argument": "1"}, atParse: true},
		{name: "no format", src: `{{ 0 | date }}`, code: liquid.CodeFilterArgument, args: map[string]string{"argument": "1"}, atParse: true},
		{name: "two formats", src: `{{ 0 | date: "%Y", "%m" }}`, code: liquid.CodeFilterArgument, args: map[string]string{"argument": "2"}, atParse: true},
		{name: "a keyword", src: `{{ 0 | date: "%Y", tz: "UTC" }}`, code: liquid.CodeFilterArgument, args: map[string]string{"argument": "tz"}, atParse: true},
	})
}

// meter is a FilterContext over a fresh meter with limits l.
type meter struct{ m liquid.Meter }

func (c meter) Meter() liquid.Meter { return c.m }

// TestCharges is ADR-009 D11 item 2's table for this package's filters,
// counted on the meter: the shallow sizes of the input and the arguments,
// the deep size beyond the input's shallow size for a filter that writes
// a value's text, the comparisons of sorting a table's names, and the
// result's length as work units and held bytes.
func TestCharges(t *testing.T) {
	testenv.Home(t)
	d := dialect
	g := data(t, `{"o": {"a": [1, 2]}, "rows": [{"b": "x", "a": 1}, {"a": 22}], "s": "Ab c", "n": 0}`)
	get := func(name string) value.Value { v, _ := g.Get(name); return v }
	str := func(s string) value.Value { v, _ := value.NewString(s); return v }
	cases := []struct {
		filter      string
		in          value.Value
		args        []liquid.Arg
		units, held int64
	}{
		// 1 + 4 beyond + 11 for {"a":[1,2]}
		{"json", get("o"), nil, 1 + 4 + 11, 11},
		// 1 + 1 for indent + 4 beyond + 21 for "{\n \"a\": [\n  1,\n  2\n ]\n}"
		{"json", get("o"), []liquid.Arg{{Name: "indent", Value: value.Number{}}}, 0, 0},
		// 1 + 4 beyond + 14 for "a:\n  - 1\n  - 2"
		{"yaml", get("o"), nil, 1 + 4 + 14, 14},
		// 2 + 6 beyond (the rows, 2 and 1, and their values, 2 and 1) +
		// sorting 3 names of 3 bytes, 3·2 + 3·2, + 12 for
		// "a,b\n1,x\n22,\n"
		{"csv", get("rows"), nil, 2 + 6 + 12 + 12, 12},
		// 4 + 10 for "ab-c": the result is no longer than the input
		{"slug", get("s"), nil, 4 + 4, 4},
		// 1 + 2 + 5 for "00:00"
		{"date", get("n"), []liquid.Arg{{Value: str("%H:%M")}}, 1 + 5 + 5, 5},
	}
	for i, c := range cases {
		if c.filter == "json" && c.args != nil {
			// indent 1, a number whose shallow size is 1
			n, _ := value.ParseNumber("1")
			c.args[0].Value = n
			c.units, c.held = 1+1+4+23, 23
		}
		f, _ := d.Filter(c.filter)
		m := liquid.NewMeter(liquid.MeterLimits{})
		out, err := f.Apply(meter{m}, c.in, c.args)
		if err != nil {
			t.Errorf("%d %s: %v", i, c.filter, err)
			continue
		}
		if got := m.Used(liquid.WorkUnits); got != c.units {
			t.Errorf("%d %s → %q: %d units, want %d", i, c.filter, out.(value.String).Text(), got, c.units)
		}
		if got := m.Used(liquid.WorkHeldBytes); got != c.held {
			t.Errorf("%d %s → %q: %d held bytes, want %d", i, c.filter, out.(value.String).Text(), got, c.held)
		}
		// One unit fewer is refused, at the last charge, before the
		// result is built.
		m = liquid.NewMeter(liquid.MeterLimits{RenderWork: c.units - 1})
		if _, err := f.Apply(meter{m}, c.in, c.args); err == nil || err.Code != liquid.CodeLimit || err.Args["limit"] != "render-work" {
			t.Errorf("%d %s with render-work %d: %v, want the limit", i, c.filter, c.units-1, err)
		}
	}
}
