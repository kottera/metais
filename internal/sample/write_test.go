// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package sample

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/kottera/metais/internal/inputs/csv"
	"github.com/kottera/metais/internal/liquid/value"
	"github.com/kottera/metais/internal/testenv"
)

// files writes v as in, failing t on an error.
func files(t testing.TB, p *pkg, in Input, v value.Value) []File {
	t.Helper()
	fs, err := p.gen(Options{}).Files(in, v)
	if err != nil {
		t.Fatal(err)
	}
	return fs
}

// TestFiles is ADR-007 D9 item 21: each form's bytes, extension and path.
func TestFiles(t *testing.T) {
	testenv.Home(t)
	p := compile(t, map[string]string{
		entry: `{"type":"object","properties":{"b":{"type":"integer"},"a":{"type":"string"}}}`,
		"inputs/rows.json": `{"type":"array","items":{"type":"object","properties":{
			"zeta":{"type":"string"},"n":{"type":"integer"},"x":{"type":"number"},"ok":{"type":"boolean"},"alpha":{"type":["string","integer"]}}}}`,
		"inputs/doc.json":  `{"type":"object","properties":{"title":{"type":"string"},"body":{"type":"string"}}}`,
		"inputs/text.json": `{"type":"string"}`,
		"inputs/list.json": `{"type":"array","items":{"type":"object"}}`,
	})
	obj := parse(t, `{"b":1,"a":"x y"}`)
	rows := parse(t, `[{"zeta":"a,b","n":12,"x":0.000001,"ok":true,"alpha":"text"},{"zeta":"","n":-3,"alpha":7},{"zeta":"say \"hi\"\nthere","x":1e15}]`)
	for _, tc := range []struct {
		name string
		in   Input
		v    value.Value
		want map[string]string
	}{
		{"json", Input{Name: "data", Schema: entry, Forms: []Form{FormJSON, FormYAML}}, obj,
			map[string]string{"data.json": "{\n  \"a\": \"x y\",\n  \"b\": 1\n}\n"}},
		{"yaml", Input{Name: "data", Schema: entry, Forms: []Form{FormYAML}}, obj,
			map[string]string{"data.yaml": "a: x y\nb: 1\n"}},
		{"csv", Input{Name: "rows", Schema: "inputs/rows.json", Forms: []Form{FormCSV}}, rows,
			map[string]string{"rows.csv": "zeta,n,x,ok,alpha\r\n\"a,b\",12,0.000001,true,text\r\n\"\",-3,,,7\r\n\"say \"\"hi\"\"\r\nthere\",,1000000000000000,,\r\n"}},
		{"csv with another delimiter", Input{Name: "rows", Schema: "inputs/rows.json", Forms: []Form{FormCSV}, Delimiter: csv.Semicolon}, parse(t, `[{"zeta":"a,b;c","n":1}]`),
			map[string]string{"rows.csv": "zeta;n;x;ok;alpha\r\n\"a,b;c\";1;;;\r\n"}},
		{"text", Input{Name: "summary", Schema: "inputs/text.json", Forms: []Form{FormText}}, parse(t, `"one\ntwo\n"`),
			map[string]string{"summary.txt": "one\ntwo\n\n"}},
		{"markdown", Input{Name: "doc", Schema: "inputs/doc.json", Forms: []Form{FormMarkdown}}, parse(t, `{"title":"T: one","body":"# Head\n\nText."}`),
			map[string]string{"doc.md": "---\ntitle: \"T: one\"\n---\n# Head\n\nText.\n"}},
		{"markdown with no frontmatter", Input{Name: "doc", Schema: "inputs/doc.json", Forms: []Form{FormMarkdown}}, parse(t, `{"body":"Text."}`),
			map[string]string{"doc.md": "Text.\n"}},
		{"a body that opens like frontmatter", Input{Name: "doc", Schema: "inputs/doc.json", Forms: []Form{FormMarkdown}}, parse(t, `{"body":"---\nx"}`),
			map[string]string{"doc.md": "---\n---\n---\nx\n"}},
		{"a collection", Input{Name: "list", Schema: "inputs/list.json", Forms: []Form{FormJSON}, Collection: true}, parse(t, `[{},{"k":1}]`),
			map[string]string{"list/": "", "list/001.json": "{}\n", "list/002.json": "{\n  \"k\": 1\n}\n"}},
		{"an empty collection", Input{Name: "list", Schema: "inputs/list.json", Forms: []Form{FormYAML}, Collection: true}, parse(t, `[]`),
			map[string]string{"list/": ""}},
	} {
		got := map[string]string{}
		for _, f := range files(t, p, tc.in, tc.v) {
			got[f.Path] = string(f.Data)
		}
		if fmt.Sprint(got) != fmt.Sprint(tc.want) {
			t.Errorf("%s: got %q\nwant %q", tc.name, got, tc.want)
		}
	}
	var items []value.Value
	for range 1000 {
		items = append(items, value.Null{})
	}
	many, _ := value.NewArray(items...)
	fs := files(t, p, Input{Name: "list", Schema: "inputs/list.json", Forms: []Form{FormJSON}, Collection: true}, many)
	if fs[1].Path != "list/0001.json" || fs[1000].Path != "list/1000.json" {
		t.Errorf("1000 items are named %s … %s", fs[1].Path, fs[1000].Path)
	}
}

// TestFilesRefuse: a value its form cannot write so that it reads back
// the same is refused, not written.
func TestFilesRefuse(t *testing.T) {
	testenv.Home(t)
	p := compile(t, map[string]string{
		"inputs/rows.json": `{"type":"array","items":{"type":"object","properties":{"n":{"type":"integer"},"s":{"type":"string"},"b":{"type":["boolean","string"]}}}}`,
		"inputs/one.json":  `{"type":"array","items":{"type":"object","properties":{"s":{"type":"string"}}}}`,
		"inputs/text.json": `{"type":"string"}`,
		"inputs/doc.json":  `{"type":"object"}`,
	})
	rows := Input{Name: "rows", Schema: "inputs/rows.json", Forms: []Form{FormCSV}}
	for _, tc := range []struct {
		name string
		in   Input
		v    string
	}{
		{"a CR in a text file", Input{Name: "t", Schema: "inputs/text.json", Forms: []Form{FormText}}, `"a\rb"`},
		{"a number as a text file", Input{Name: "t", Schema: "inputs/text.json", Forms: []Form{FormText}}, `1`},
		{"a CR in a body", Input{Name: "d", Schema: "inputs/doc.json", Forms: []Form{FormMarkdown}}, `{"body":"a\r"}`},
		{"no body", Input{Name: "d", Schema: "inputs/doc.json", Forms: []Form{FormMarkdown}}, `{"t":"x"}`},
		{"a null field", rows, `[{"n":null}]`},
		{"an array field", rows, `[{"s":[1]}]`},
		{"an object field", rows, `[{"s":{}}]`},
		{"a CR in a field", rows, `[{"s":"a\rb"}]`},
		{"a string an integer column reads as a number", rows, `[{"n":"12"}]`},
		{"a string a boolean column reads as one", rows, `[{"b":"true"}]`},
		{"a fraction in an integer column", rows, `[{"n":1.5}]`},
		{"a large integer", rows, `[{"n":1e17}]`},
		{"a record that is not an object", rows, `[1]`},
		{"an empty one-column record", Input{Name: "one", Schema: "inputs/one.json", Forms: []Form{FormCSV}}, `[{}]`},
		{"a table with no column", Input{Name: "t", Schema: "inputs/text.json", Forms: []Form{FormCSV}}, `[]`},
	} {
		_, err := p.gen(Options{}).Files(tc.in, parse(t, tc.v))
		var ue *UnwritableError
		if !errors.As(err, &ue) {
			t.Errorf("%s: %v", tc.name, err)
		}
	}
}

// TestSuppliedLargeInteger is W3-5F: a supplied integral number beyond
// 2^53−1, which the reader takes when it is spelled with an exponent, is
// written in JSON and YAML with an exponent, the shortest text that
// round-trips, and reads back to the same value with no finding. A CSV
// integer column still has no field for it, and refuses it.
func TestSuppliedLargeInteger(t *testing.T) {
	testenv.Home(t)
	p := compile(t, map[string]string{
		"inputs/doc.json":  `{"type":"object","properties":{"n":{"type":"integer"},"x":{"type":"number"},"s":{"type":"string"}}}`,
		"inputs/rows.json": `{"type":"array","items":{"type":"object","properties":{"n":{"type":"integer"}}}}`,
	})
	for _, tc := range []struct {
		supplied, json string
	}{
		{`{"n":11e17}`, "{\n  \"n\": 1.1e+18\n}\n"},
		{`{"n":-11e17}`, "{\n  \"n\": -1.1e+18\n}\n"},
		{`{"n":9007199254740992e0}`, "{\n  \"n\": 9.007199254740992e+15\n}\n"},
		{`{"n":1e20}`, "{\n  \"n\": 1e+20\n}\n"},
		{`{"n":1e21}`, "{\n  \"n\": 1e+21\n}\n"},
		// The largest and the smallest finite binary64, each integral.
		{`{"n":1.7976931348623157e308,"x":[-1.7976931348623157e308,9007199254740991]}`,
			"{\n  \"n\": 1.7976931348623157e+308,\n  \"x\": [\n    -1.7976931348623157e+308,\n    9007199254740991\n  ]\n}\n"},
	} {
		for _, f := range []Form{FormJSON, FormYAML} {
			in := Input{Name: "d", Schema: "inputs/doc.json", Forms: []Form{f}, Supplied: parse(t, tc.supplied)}
			v, _, fs, err := p.gen(Options{Optional: None}).Generate(in)
			if err != nil || fs != nil || !value.Equal(v, parse(t, tc.supplied)) {
				t.Fatalf("%s %s: %s %v%s", f, tc.supplied, text(v), err, describe(fs))
			}
			written, err := p.gen(Options{}).Files(in, v)
			if err != nil {
				t.Errorf("%s %s: %v", f, tc.supplied, err)
				continue
			}
			if f == FormJSON && string(written[0].Data) != tc.json {
				t.Errorf("json %s: wrote %q, want %q", tc.supplied, written[0].Data, tc.json)
			}
			// readBack fails on any finding.
			if back := readBack(t, p, in, written); !value.Equal(back, v) {
				t.Errorf("%s %s: %q reads back as %s", f, tc.supplied, written[0].Data, text(back))
			}
		}
	}
	rows := Input{Name: "rows", Schema: "inputs/rows.json", Forms: []Form{FormCSV}, Supplied: parse(t, `[{"n":11e17}]`)}
	v, _, _, _ := p.gen(Options{}).Generate(rows)
	_, err := p.gen(Options{}).Files(rows, v)
	var ue *UnwritableError
	if !errors.As(err, &ue) || ue.Pointer != "/0/n" {
		t.Errorf("csv: %v", err)
	}
}

// roundTrip is one case of the round-trip property: a package's schema
// files and an input of it.
type roundTrip struct {
	name  string
	files map[string]string
	in    Input
}

// roundTrips is the property test's corpus: every form, collections of
// each, and values that press on what the writers must escape or spell.
func roundTrips() []roundTrip {
	obj := `{"type":"object","required":["id"],"properties":{
		"id":{"type":"string","format":"uuid"},
		"title":{"type":"string","maxLength":30},
		"quote":{"type":"string","pattern":"^[\"\\\\#:{}\\[\\]&*!|>'%@,-]{4}$"},
		"controls":{"type":"string","pattern":"^[\\u0000-\\u0008\\u000b-\\u001f]{3}$"},
		"line":{"type":"string","pattern":"^a\\nb\\u2028c\\u0085d$"},
		"word":{"enum":["yes","no","null","true","~","012","1e3",""]},
		"n":{"type":"integer","minimum":9007199254740000},
		"neg":{"type":"integer","maximum":-9007199254740000},
		"x":{"type":"number","multipleOf":0.0000001,"minimum":0,"maximum":0.000002},
		"big":{"type":"number","minimum":1e15},
		"flag":{"type":"boolean"},
		"none":{"type":["null","string"]},
		"when":{"type":"string","format":"date-time"},
		"tags":{"type":"array","items":{"type":"string","pattern":"^[a-z]{1,3}( [a-z]{1,3})?$"},"uniqueItems":true},
		"nested":{"type":"object","properties":{"deep":{"type":"array","prefixItems":[{"type":"integer"},{"type":"object","properties":{"k":{"type":"string","examples":["- x","a: b","é"]}}}]}}},
		"open":{"type":"object","minProperties":2,"additionalProperties":{"type":"number"}}}}`
	rows := `{"type":"array","items":{"type":"object","required":["name"],"properties":{
		"name":{"type":"string","pattern":"^[a-z,;|\"\\t ]{2,6}$"},
		"empty":{"const":""},
		"count":{"type":"integer","minimum":-5,"maximum":5},
		"ratio":{"type":"number","exclusiveMinimum":0,"exclusiveMaximum":1},
		"ok":{"type":"boolean"},
		"note":{"type":["string","null"],"maxLength":40},
		"lines":{"type":"string","pattern":"^x\\ny$"},
		"either":{"type":["integer","string"]},
		"punct":{"enum":["a,b","say \"x\"","p|q","t\tu","s;t"," lead","trail ","x\ny"]}}}}`
	doc := `{"type":"object","required":["title"],"properties":{"title":{"type":"string"},"tags":{"type":"array","items":{"type":"string"}},"draft":{"type":"boolean"},"body":{"type":"string"}}}`
	var out []roundTrip
	for _, f := range []Form{FormJSON, FormYAML} {
		out = append(out,
			roundTrip{"object as " + string(f), map[string]string{"inputs/o.json": obj}, Input{Name: "o", Schema: "inputs/o.json", Forms: []Form{f}, Required: true}},
			roundTrip{"collection as " + string(f), map[string]string{"inputs/o.json": `{"type":"array","items":` + obj + `}`}, Input{Name: "o", Schema: "inputs/o.json", Forms: []Form{f}, Collection: true, Required: true}},
			roundTrip{"scalar as " + string(f), map[string]string{"inputs/s.json": `{"anyOf":[{"type":"string","pattern":"^\\u00a0[ \\t]$"},{"type":"number","minimum":-1e-7,"maximum":1e-7,"multipleOf":1e-8},{"type":"null"},{"type":"boolean"}]}`}, Input{Name: "s", Schema: "inputs/s.json", Forms: []Form{f}, Required: true}},
		)
	}
	for _, d := range []csv.Delimiter{csv.Comma, csv.Semicolon, csv.Pipe, csv.Tab} {
		out = append(out,
			roundTrip{fmt.Sprintf("csv %q", d), map[string]string{"inputs/r.json": rows}, Input{Name: "r", Schema: "inputs/r.json", Forms: []Form{FormCSV}, Delimiter: d, Required: true}},
			roundTrip{fmt.Sprintf("csv collection %q", d), map[string]string{"inputs/r.json": `{"type":"array","items":` + rows + `}`}, Input{Name: "r", Schema: "inputs/r.json", Forms: []Form{FormCSV}, Delimiter: d, Collection: true, Required: true}},
		)
	}
	out = append(out,
		roundTrip{"text", map[string]string{"inputs/t.json": `{"type":"string"}`}, Input{Name: "t", Schema: "inputs/t.json", Forms: []Form{FormText}, Required: true}},
		roundTrip{"text ending in LF", map[string]string{"inputs/t.json": `{"type":"string","pattern":"^a\\n\\n$"}`}, Input{Name: "t", Schema: "inputs/t.json", Forms: []Form{FormText}, Required: true}},
		roundTrip{"text collection", map[string]string{"inputs/t.json": `{"type":"array","items":{"type":"string","maxLength":50}}`}, Input{Name: "t", Schema: "inputs/t.json", Forms: []Form{FormText, FormJSON}, Collection: true, Required: true}},
		roundTrip{"markdown", map[string]string{"inputs/m.json": doc}, Input{Name: "m", Schema: "inputs/m.json", Forms: []Form{FormMarkdown}, Required: true}},
		roundTrip{"markdown body only", map[string]string{"inputs/m.json": `{"type":"object","properties":{"body":{"type":"string","pattern":"^---\\nx$"}}}`}, Input{Name: "m", Schema: "inputs/m.json", Forms: []Form{FormMarkdown}, Required: true}},
		roundTrip{"markdown collection", map[string]string{"inputs/m.json": `{"type":"array","items":` + doc + `}`}, Input{Name: "m", Schema: "inputs/m.json", Forms: []Form{FormMarkdown, FormYAML}, Collection: true, Required: true}},
	)
	return out
}

// TestRoundTrip is W3-5.4: a file the tool writes is readable by the
// tool. Every value the generator makes, written in its input's form by
// Files, reads back through Wave 2's reader of that form to the same
// value, and validates against its schema, under every mode and many
// seeds.
func TestRoundTrip(t *testing.T) {
	testenv.Home(t)
	for _, rt := range roundTrips() {
		p := compile(t, rt.files)
		sch, _ := p.c.Schema(rt.in.Schema)
		for _, mode := range []Mode{All, None, Mixed} {
			for seed := range uint64(6) {
				opts := Options{Seed: seed, Optional: mode, Items: int(seed % 4)}
				v, present, fs, err := p.gen(opts).Generate(rt.in)
				if err != nil || len(fs) > 0 || !present {
					t.Fatalf("%s, %s, seed %d: %v%s", rt.name, mode, seed, err, describe(fs))
				}
				if fails, err := sch.Validate(v); err != nil || len(fails) > 0 {
					t.Errorf("%s, %s, seed %d: %s does not validate: %v", rt.name, mode, seed, text(v), fails)
					continue
				}
				written := files(t, p, rt.in, v)
				back := readBack(t, p, rt.in, written)
				if !value.Equal(back, v) {
					t.Errorf("%s, %s, seed %d: %s\nreads back as %s\nfrom %q", rt.name, mode, seed, text(v), text(back), written)
				}
			}
		}
	}
}

// TestRoundTripPressed checks the corpus presses on what it is meant to:
// strings the writers must escape or quote, and numbers they must spell
// with care, are among the values made.
func TestRoundTripPressed(t *testing.T) {
	testenv.Home(t)
	rt := roundTrips()[0]
	p := compile(t, rt.files)
	v, _, _, _ := p.gen(Options{}).Generate(rt.in)
	for _, name := range []string{"quote", "controls", "line", "n", "neg", "x", "big", "open"} {
		if member(v, name) == nil {
			t.Errorf("%s was not made", name)
		}
	}
	if n := member(v, "n").(value.Number); n.Float64() < 9007199254740000 {
		t.Errorf("n is %s", n.Literal())
	}
	if s := member(v, "line").(value.String).Text(); !strings.Contains(s, "\u2028") || !strings.Contains(s, "\u0085") {
		t.Errorf("line is %q", s)
	}
}
