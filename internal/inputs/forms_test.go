// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package inputs

import (
	"strings"
	"testing"

	"github.com/kottera/metais/internal/limits"
	"github.com/kottera/metais/internal/testenv"
)

// formsPkg declares one input per form: doc in JSON or YAML, rows in CSV,
// note in plain text and page in Markdown, none of them required, with
// schemas that accept every value the tests give, so that the findings are
// the readers'.
var formsPkg = pkg{
	inputs: []map[string]any{
		slot("doc", "inputs/any.json", "required", false),
		slot("rows", "inputs/rows.json", "forms", []any{"csv"}, "required", false),
		slot("note", "inputs/note.json", "forms", []any{"text"}, "required", false),
		slot("page", "inputs/any.json", "forms", []any{"markdown"}, "required", false),
		slot("semi", "inputs/rows.json", "forms", []any{"csv"}, "csv", map[string]any{"delimiter": ";"}, "required", false),
	},
	schemas: map[string]string{
		"inputs/any.json":  `{}`,
		"inputs/note.json": `{"type": "string"}`,
		"inputs/rows.json": `{"type": "array", "items": {"type": "object", "properties": {
			"n": {"type": "integer"}, "x": {"type": "number"}, "b": {"type": "boolean"},
			"nb": {"type": ["number", "boolean"]}, "bn": {"type": ["boolean", "number"]},
			"in": {"type": ["integer", "string"]}, "s": {"type": "string"}}}}`,
	},
}

// values reads files as a data directory of formsPkg and returns each
// input's value by name, failing on any finding.
func values(t *testing.T, files map[string]string) map[string]string {
	t.Helper()
	res := run(t, layer(t, formsPkg), limits.Defaults(), args{data: dataDir(t, files)})
	if len(res.Findings) > 0 {
		t.Fatalf("findings:\n%s", describe(res.Findings))
	}
	out := map[string]string{}
	for _, r := range res.Inputs {
		out[r.Name] = canonical(r.Value)
	}
	return out
}

// TestByteOrderMarkRemoved: "A single leading byte order mark (EF BB BF) is
// removed before parsing, in every form" (ADR-003 D8 item 1), and "the raw
// bytes keep" it (D21 item 2).
func TestByteOrderMarkRemoved(t *testing.T) {
	testenv.Home(t)
	files := map[string]string{
		"doc.yaml": "a: 1\n",
		"rows.csv": "s,n\nx,1\n",
		"note.txt": "hello\n",
		"page.md":  "---\ntitle: T\n---\nBody\n",
	}
	plain := values(t, files)
	withBOM := map[string]string{}
	for k, v := range files {
		withBOM[k] = "\uFEFF" + v
	}
	withBOM["doc.json"] = "\uFEFF{\"a\": 1}"
	delete(withBOM, "doc.yaml")
	plain2 := values(t, withBOM)
	for name, v := range plain {
		if plain2[name] != v {
			t.Errorf("%s: with a byte order mark %s, without %s", name, plain2[name], v)
		}
	}
	res := run(t, layer(t, formsPkg), limits.Defaults(), args{data: dataDir(t, withBOM)})
	for _, r := range res.Inputs {
		if r.HasRaw && !strings.HasPrefix(r.Raw, "\uFEFF") {
			t.Errorf("%s: the raw bytes lost the byte order mark: %q", r.Name, r.Raw)
		}
	}
}

// TestLineBreaksNormalized: "For csv, text and markdown, line breaks —
// CRLF, CR and LF — are normalized to LF before parsing, inside quoted CSV
// fields too" (ADR-003 D8 item 2); the raw bytes keep them as written.
func TestLineBreaksNormalized(t *testing.T) {
	testenv.Home(t)
	lf := map[string]string{
		"rows.csv": "s,n\n\"a\nb\",1\nc,2\n",
		"note.txt": "one\ntwo\n",
		"page.md":  "---\ntitle: T\n---\n\nPara\nline\n",
	}
	want := values(t, lf)
	for _, br := range []string{"\r\n", "\r"} {
		files := map[string]string{}
		for k, v := range lf {
			files[k] = strings.ReplaceAll(v, "\n", br)
		}
		got := values(t, files)
		for name, v := range want {
			if got[name] != v {
				t.Errorf("%q: %s is %s, with LF %s", br, name, got[name], v)
			}
		}
		res := run(t, layer(t, formsPkg), limits.Defaults(), args{data: dataDir(t, files)})
		if r := record(t, res, "note"); string(r.Raw) != files["note.txt"] {
			t.Errorf("%q: raw %q", br, r.Raw)
		}
	}
	if want["rows"] != `[{"n":1,"s":"a\nb"},{"n":2,"s":"c"}]` || want["note"] != `"one\ntwo"` || want["page"] != `{"body":"\nPara\nline","title":"T"}` {
		t.Errorf("values %v", want)
	}
}

// TestEncodingRefused: "invalid UTF-8 is a finding at the line and byte
// where it occurs … Any other encoding, UTF-16 included, is a finding"
// (ADR-003 D8 item 1), in every form, and the file contributes no value.
func TestEncodingRefused(t *testing.T) {
	testenv.Home(t)
	l := layer(t, formsPkg)
	res := run(t, l, limits.Defaults(), args{data: dataDir(t, map[string]string{
		"doc.yaml": "a: 1\nb: \xff\n",
		"rows.csv": "\xff\xfes\x00",
		"note.txt": "ok\r\nnot \xc3\n",
		"page.md":  "\uFEFF---\n\xe2\x82\n",
		"semi.csv": "\xfe\xff\x00s",
	})})
	wantFindings(t, res.Findings,
		`input.encoding_invalid doc data_dir:doc.yaml:2:4 {"reason":"invalid_utf8","offset":8}`,
		`input.encoding_invalid rows data_dir:rows.csv:1:1 {"reason":"utf16","offset":0}`,
		`input.encoding_invalid note data_dir:note.txt:2:5 {"reason":"invalid_utf8","offset":8}`,
		`input.encoding_invalid page data_dir:page.md:2:1 {"reason":"invalid_utf8","offset":7}`,
		`input.encoding_invalid semi data_dir:semi.csv:1:1 {"reason":"utf16","offset":0}`,
	)
	for _, r := range res.Inputs {
		if !r.Unparsed || r.Value != nil {
			t.Errorf("%s: a file that is not UTF-8 gave %s (unparsed %v)", r.Name, canonical(r.Value), r.Unparsed)
		}
	}
}

// TestJSONForm: a JSON file is one RFC 8259 text; trailing content and
// every duplicate member name are findings (ADR-003 D9); a number written
// as an integer beyond 2^53−1 is a finding, and 1 and 1.0 are the same
// value with the same digest (D8 item 5).
func TestJSONForm(t *testing.T) {
	testenv.Home(t)
	l := layer(t, formsPkg)
	res := run(t, l, limits.Defaults(), args{data: dataDir(t, map[string]string{"doc.json": "{\"a\": 1, \"a\": 2, \"a\": 3}"})})
	wantFindings(t, res.Findings,
		`json.duplicate_name doc data_dir:doc.json:1:10 #/a {"name":"a"}`,
		`json.duplicate_name doc data_dir:doc.json:1:18 #/a {"name":"a"}`,
	)
	res = run(t, l, limits.Defaults(), args{data: dataDir(t, map[string]string{"doc.json": "{}\n{}"})})
	wantFindings(t, res.Findings, `json.trailing_content doc data_dir:doc.json:2:1`)
	res = run(t, l, limits.Defaults(), args{data: dataDir(t, map[string]string{"doc.json": "[9007199254740991, 9007199254740992, 1e300]"})})
	wantFindings(t, res.Findings, `input.number_out_of_range doc data_dir:doc.json:1:20 #/1 {"reason":"unsafe_integer"}`)

	one := run(t, l, limits.Defaults(), args{data: dataDir(t, map[string]string{"doc.json": `{"n": 1}`})})
	oneDot := run(t, l, limits.Defaults(), args{data: dataDir(t, map[string]string{"doc.json": `{"n": 1.0}`})})
	if a, b := record(t, one, "doc"), record(t, oneDot, "doc"); a.Digest != b.Digest {
		t.Errorf("1 and 1.0 digest apart: %s, %s", canonical(a.Value), canonical(b.Value))
	}
}

// TestYAMLForm: input YAML is the manifest's profile (ADR-003 D10): `key:`
// is null, not absent; a plain 2026-10-18, yes and on are strings; an
// empty file is null; ~, .inf and an alias are refused.
func TestYAMLForm(t *testing.T) {
	testenv.Home(t)
	got := values(t, map[string]string{"doc.yaml": "a:\nd: 2026-10-18\ny: yes\no: on\nn: null\n"})
	if got["doc"] != `{"a":null,"d":"2026-10-18","n":null,"o":"on","y":"yes"}` {
		t.Errorf("doc = %s", got["doc"])
	}
	if got := values(t, map[string]string{"doc.yaml": "# only a comment\n"}); got["doc"] != "null" {
		t.Errorf("a comment-only file = %s", got["doc"])
	}
	l := layer(t, formsPkg)
	res := run(t, l, limits.Defaults(), args{data: dataDir(t, map[string]string{"doc.yaml": "a: ~\nb: .inf\nc: &x 1\n"})})
	if c := codes(res.Findings); strings.Join(c, " ") != "yaml.anchor yaml.scalar_refused yaml.scalar_refused" {
		t.Errorf("codes %v:\n%s", c, describe(res.Findings))
	}
}

// TestCSVForm: ADR-003 D11. The header is required; an empty line is no
// record; spaces are data; an unquoted empty field is absent and "" is the
// empty string (R1-024); each field is coerced to its column's declared
// type set by D14 item 2's rows, whatever order the set is written in; a
// column the schema does not declare yields strings.
func TestCSVForm(t *testing.T) {
	testenv.Home(t)
	l := layer(t, formsPkg)
	res := run(t, l, limits.Defaults(), args{data: dataDir(t, map[string]string{
		"rows.csv": "n,x,b,nb,bn,in,s,u\n" +
			"\n" +
			"1,1.5,true,TRUE,False,007, a ,9\n" +
			",,,,,,\"\",\n" +
			"\n" +
			"-0,1e3,yes,1,1,12,x,\"\"\n" +
			"1.0,-,True,true,1.5, 42,\"q\"\"q\",z\n",
		"semi.csv": "s;n\na,b;1\n",
	})})
	want := `[` +
		`{"b":true,"bn":false,"in":"007","n":1,"nb":true,"s":" a ","u":"9","x":1.5},` +
		`{"s":""},` +
		`{"b":"yes","bn":1,"in":12,"n":0,"nb":1,"s":"x","u":"","x":1000},` +
		`{"b":true,"bn":1.5,"in":" 42","n":"1.0","nb":true,"s":"q\"q","u":"z","x":"-"}]`
	got := map[string]string{"rows": canonical(record(t, res, "rows").Value), "semi": canonical(record(t, res, "semi").Value)}
	if got["rows"] != want {
		t.Errorf("rows\n got %s\nwant %s", got["rows"], want)
	}
	// Text that fits no listed type stays a string, and validation
	// reports it (D14 item 3), at the field (ADR-006 D5 item 1).
	want2 := []string{
		`input.type rows data_dir:rows.csv:6:8 #/2/b {"expected":["boolean"],"actual":"string"}`,
		`input.type rows data_dir:rows.csv:7:1 #/3/n {"expected":["integer"],"actual":"string"}`,
		`input.type rows data_dir:rows.csv:7:5 #/3/x {"expected":["number"],"actual":"string"}`,
	}
	wantFindings(t, res.Findings, want2...)
	if got["semi"] != `[{"n":1,"s":"a,b"}]` {
		t.Errorf("semi = %s", got["semi"])
	}

	res = run(t, l, limits.Defaults(), args{data: dataDir(t, map[string]string{"rows.csv": "\n\n"})})
	wantFindings(t, res.Findings, `csv.header_missing rows data_dir:rows.csv:3:1`)
	// A number written as an integer beyond 2^53−1 is refused in a field
	// as in any form (D8 item 5); the record is left out alone.
	res = run(t, l, limits.Defaults(), args{data: dataDir(t, map[string]string{"rows.csv": "n,s\n9007199254740993,a\n2,b\n"})})
	if c := codes(res.Findings); strings.Join(c, " ") != "input.number_out_of_range input.validation.partial" {
		t.Errorf("codes %v:\n%s", c, describe(res.Findings))
	}
}

// TestTextForm: "The value is the whole file as one string … with one
// final LF removed … An empty file is the empty string — present, not
// absent" (ADR-003 D12).
func TestTextForm(t *testing.T) {
	testenv.Home(t)
	for src, v := range map[string]string{
		"":          `""`,
		"\n":        `""`,
		"a":         `"a"`,
		"a\n":       `"a"`,
		"a\n\n":     `"a\n"`,
		"a\r\n\r\n": `"a\n"`,
		" a \t\n":   `" a \t"`,
	} {
		if got := values(t, map[string]string{"note.txt": src}); got["note"] != v {
			t.Errorf("%q = %s, want %s", src, got["note"], v)
		}
	}
	// One string: a finding about it has a file and no line (ADR-006 D5
	// item 5).
	l := layer(t, pkg{
		inputs:  []map[string]any{slot("note", "inputs/note.json", "forms", []any{"text"})},
		schemas: map[string]string{"inputs/note.json": `{"type": "string", "maxLength": 2}`},
	})
	res := run(t, l, limits.Defaults(), args{data: dataDir(t, map[string]string{"note.txt": "four\n"})})
	wantFindings(t, res.Findings, `input.max_length note data_dir:note.txt # {"limit":2,"length":4}`)
}

// TestMarkdownForm: ADR-003 D13. The value is the frontmatter's fields
// plus body; frontmatter is present only when the first line is exactly
// ---; body keeps a blank line after the frontmatter and loses one final
// LF.
func TestMarkdownForm(t *testing.T) {
	testenv.Home(t)
	for src, v := range map[string]string{
		"---\ntitle: A\ntags: [x]\n---\n\nText\n": `{"body":"\nText","tags":["x"],"title":"A"}`,
		"---\n---\nText":        `{"body":"Text"}`,
		"Text\n---\n":           `{"body":"Text\n---"}`,
		"--- \ntitle: A\n---\n": `{"body":"--- \ntitle: A\n---"}`,
		"***\n# H\n":            `{"body":"***\n# H"}`,
		"":                      `{"body":""}`,
	} {
		if got := values(t, map[string]string{"page.md": src}); got["page"] != v {
			t.Errorf("%q = %s, want %s", src, got["page"], v)
		}
	}
	l := layer(t, formsPkg)
	for src, w := range map[string]string{
		"---\ntitle: A\n":            `markdown.frontmatter_unclosed page data_dir:page.md:1:1`,
		"---\n- a\n- b\n---\nText\n": `markdown.frontmatter_not_mapping page data_dir:page.md:2:1 {"actual":"array"}`,
		"---\nx: 1\nbody: b\n---\n":  `markdown.frontmatter_body_field page data_dir:page.md:3:1 #/body`,
		"---\ntitle: [\n---\nText\n": `yaml.syntax page data_dir:page.md:3:0`,
	} {
		res := run(t, l, limits.Defaults(), args{data: dataDir(t, map[string]string{"page.md": src})})
		if got := describe(res.Findings); got != w {
			t.Errorf("%q:\n%s\nwant %s", src, got, w)
		}
	}
	// A frontmatter field's position counts from the file's first line,
	// and the body's is its first line (ADR-006 D5 item 6).
	l = layer(t, pkg{
		inputs: []map[string]any{slot("page", "inputs/page.json", "forms", []any{"markdown"})},
		schemas: map[string]string{"inputs/page.json": `{"type": "object", "properties": {
			"title": {"type": "string"}, "body": {"maxLength": 1}}}`},
	})
	res := run(t, l, limits.Defaults(), args{data: dataDir(t, map[string]string{"page.md": "---\n\ntitle: 5\n---\nab\n"})})
	wantFindings(t, res.Findings,
		`input.type page data_dir:page.md:3:8 #/title {"expected":["string"],"actual":"number"}`,
		`input.max_length page data_dir:page.md:5:1 #/body {"limit":1,"length":2}`,
	)
}

// TestNoncharacterRefused: "an escaped unpaired surrogate or a
// noncharacter is a finding" (ADR-003 D8 item 4), in a text, CSV or
// Markdown file as in JSON and YAML.
func TestNoncharacterRefused(t *testing.T) {
	testenv.Home(t)
	l := layer(t, formsPkg)
	res := run(t, l, limits.Defaults(), args{data: dataDir(t, map[string]string{
		"note.txt": "a\nb\uFFFE",
		"rows.csv": "s\n\U0010FFFF\n",
		"page.md":  "---\nt: x\n---\n\n\uFDD0",
		"doc.json": `"\uFFFF"`,
	})})
	wantFindings(t, res.Findings,
		`input.noncharacter note data_dir:note.txt:2:2 {"codepoint":"U+FFFE"}`,
		`input.noncharacter rows data_dir:rows.csv:2:1 {"codepoint":"U+10FFFF"}`,
		`input.noncharacter page data_dir:page.md:5:1 {"codepoint":"U+FDD0"}`,
		`json.noncharacter doc data_dir:doc.json:1:1 #`,
	)
}
