// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package schema

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/santhosh-tekuri/jsonschema/v6"

	"github.com/kottera/metais/internal/finding"
	"github.com/kottera/metais/internal/limits"
	"github.com/kottera/metais/internal/liquid/value"
	"github.com/kottera/metais/internal/pkgtree/pkgtreetest"
	"github.com/kottera/metais/internal/schemafail"
	"github.com/kottera/metais/internal/testenv"
)

// bare compiles schema with the library as it comes, configured by
// nothing of METAIS's: the control that shows a refusal below is
// METAIS's own, made before the library sees the schema.
func bare(t *testing.T, schema string) error {
	t.Helper()
	doc, err := jsonschema.UnmarshalJSON(strings.NewReader(schema))
	if err != nil {
		t.Fatal(err)
	}
	c := jsonschema.NewCompiler()
	c.AssertFormat()
	if err := c.AddResource("https://example.com/s.json", doc); err != nil {
		t.Fatal(err)
	}
	_, err = c.Compile("https://example.com/s.json")
	return err
}

// only returns the one finding of fs, failing unless there is exactly one.
func only(t *testing.T, fs []finding.Finding) finding.Finding {
	t.Helper()
	if len(fs) != 1 {
		t.Fatalf("got %d findings, want 1:\n%s", len(fs), describe(fs))
	}
	return fs[0]
}

func param(t *testing.T, f finding.Finding, name string) string {
	t.Helper()
	v, ok := f.Param(name)
	if !ok {
		t.Fatalf("%s has no %s", f.Code(), name)
	}
	s, ok := v.Str()
	if !ok {
		t.Fatalf("%s's %s is not a string", f.Code(), name)
	}
	return s
}

func lower(t *testing.T, arg string) limits.Set {
	t.Helper()
	lim := limits.Defaults()
	if f, ok := lim.Lower(arg); !ok {
		t.Fatalf("Lower(%q): %s", arg, f.Message())
	}
	return lim
}

// TestPrecheckRefusals is a negative test for each refusal ADR-010 D5 item
// 3 makes before the library sees a schema: an unknown keyword, a format
// outside ADR-003 D4's list, a $schema other than 2020-12's, $vocabulary,
// a pattern outside the profile, and the size and nesting limits. For
// each, the bare library compiles the same schema, so the refusal is the
// pre-check's.
func TestPrecheckRefusals(t *testing.T) {
	testenv.Home(t)
	cases := []struct {
		name, schema string
		lim          limits.Set
		code         finding.Code
		ptr          string
		param, value string
	}{
		{"an unknown keyword", `{"type": "object", "requried": ["a"]}`, limits.Defaults(),
			CodeKeywordUnknown, "/requried", "keyword", "requried"},
		{"an unknown keyword in a subschema", `{"properties": {"a": {"maxLenght": 3}}}`, limits.Defaults(),
			CodeKeywordUnknown, "/properties/a/maxLenght", "keyword", "maxLenght"},
		{"an earlier draft's keyword", `{"definitions": {"a": {}}}`, limits.Defaults(),
			CodeKeywordUnknown, "/definitions", "keyword", "definitions"},
		{"an unknown keyword in $defs", `{"$defs": {"a": {"x-ok": 1, "nope": 1}}}`, limits.Defaults(),
			CodeKeywordUnknown, "/$defs/a/nope", "keyword", "nope"},
		{"an IRI format", `{"format": "iri"}`, limits.Defaults(), CodeFormatUnsupported, "/format", "format", "iri"},
		{"the regex format", `{"format": "regex"}`, limits.Defaults(), CodeFormatUnsupported, "/format", "format", "regex"},
		{"an IDN format", `{"format": "idn-hostname"}`, limits.Defaults(), CodeFormatUnsupported, "/format", "format", "idn-hostname"},
		{"a format only the library knows", `{"format": "semver"}`, limits.Defaults(), CodeFormatUnsupported, "/format", "format", "semver"},
		{"an unknown format", `{"items": {"format": "colour"}}`, limits.Defaults(), CodeFormatUnsupported, "/items/format", "format", "colour"},
		{"an earlier draft", `{"$schema": "http://json-schema.org/draft-07/schema#"}`, limits.Defaults(),
			CodeDialect, "/$schema", "reason", "schema"},
		{"2020-12 with an empty fragment", `{"$schema": "https://json-schema.org/draft/2020-12/schema#"}`, limits.Defaults(),
			CodeDialect, "/$schema", "reason", "schema"},
		{"an embedded resource's dialect", `{"$defs": {"a": {"$id": "https://example.com/a", "$schema": "https://json-schema.org/draft/2019-09/schema"}}}`, limits.Defaults(),
			CodeDialect, "/$defs/a/$schema", "reason", "schema"},
		{"$vocabulary", `{"$vocabulary": {"https://json-schema.org/draft/2020-12/vocab/core": true}}`, limits.Defaults(),
			CodeDialect, "/$vocabulary", "reason", "vocabulary"},
		{"RE2's inline flags", `{"pattern": "(?i)abc"}`, limits.Defaults(),
			CodePatternUnsupported, "/pattern", "construct", "re2_syntax"},
		{"RE2's \\z", `{"patternProperties": {"^a\\z": {}}}`, limits.Defaults(),
			CodePatternUnsupported, `/patternProperties/^a\z`, "construct", "re2_syntax"},
		{"schema-size", `{"type": "string"}`, lower(t, "schema-size=10"), finding.Code("limit.package"), "", "limit", "schema-size"},
		{"schema-depth", `{"items": {"items": {"items": {}}}}`, lower(t, "schema-depth=3"),
			finding.Code("limit.package"), "/items/items/items", "limit", "schema-depth"},
		{"pattern-length", `{"pattern": "abcdef"}`, lower(t, "pattern-length=5"), finding.Code("limit.package"), "/pattern", "limit", "pattern-length"},
		{"regex-program", `{"pattern": "a{100}"}`, lower(t, "regex-program=50"), finding.Code("limit.package"), "/pattern", "limit", "regex-program"},
	}
	for _, c := range cases {
		if err := bare(t, c.schema); err != nil {
			t.Errorf("%s: the bare library refuses the schema too (%v), so it shows nothing", c.name, err)
		}
		comp, fs := compileFiles(t, c.lim, map[string]string{"inputs/s.json": c.schema})
		if comp != nil {
			t.Errorf("%s: NewCompiler returned a compiler beside its findings", c.name)
		}
		if len(fs) != 1 {
			t.Errorf("%s: %d findings, want 1:\n%s", c.name, len(fs), describe(fs))
			continue
		}
		f := fs[0]
		l, _ := f.Location()
		if f.Code() != c.code || l.File != "inputs/s.json" || l.Pointer != c.ptr {
			t.Errorf("%s: got %s at %s#%s, want %s at inputs/s.json#%s", c.name, f.Code(), l.File, l.Pointer, c.code, c.ptr)
		}
		if got := param(t, f, c.param); got != c.value {
			t.Errorf("%s: %s is %q, want %q", c.name, c.param, got, c.value)
		}
	}
}

// TestPrecheckAccepts holds the pre-check to what it lets through: x-
// annotations, keywords whose values are data rather than schemas, the
// 2020-12 $schema, and every asserted format.
func TestPrecheckAccepts(t *testing.T) {
	schemas := []string{
		`{"$schema": "https://json-schema.org/draft/2020-12/schema", "x-owner": {"nope": 1}}`,
		`{"enum": [{"requried": 1}], "const": {"maxLenght": 1}, "default": {"x": 1}, "examples": [{"y": 2}]}`,
		`{"properties": {"requried": {"type": "string"}}, "required": ["requried"]}`,
		`{"title": "t", "description": "d", "deprecated": false, "readOnly": true, "writeOnly": false, "$comment": "c"}`,
		`{"contentEncoding": "base64", "contentMediaType": "application/json", "contentSchema": {"type": "object"}}`,
		`{"anyOf": [{"format": "date-time"}, {"format": "date"}, {"format": "time"}, {"format": "duration"},
		  {"format": "email"}, {"format": "hostname"}, {"format": "ipv4"}, {"format": "ipv6"}, {"format": "uri"},
		  {"format": "uri-reference"}, {"format": "uuid"}, {"format": "uri-template"}, {"format": "json-pointer"},
		  {"format": "relative-json-pointer"}]}`,
		`{"pattern": "^\\p{Lu}[a-z]{1,1000}$", "patternProperties": {"^x-": {}}}`,
	}
	for _, s := range schemas {
		if _, fs := compileFiles(t, limits.Defaults(), map[string]string{"inputs/s.json": s}); len(fs) > 0 {
			t.Errorf("%s: %s", s, describe(fs))
		}
	}
}

// TestEveryFindingAtOnce reports every refusal of every file, not the
// first.
func TestEveryFindingAtOnce(t *testing.T) {
	_, fs := compileFiles(t, limits.Defaults(), map[string]string{
		"inputs/a.json": `{"nope": 1, "format": "iri"}`,
		"inputs/b.json": `{"$ref": "https://example.com/x.json", "pattern": "(?=a)"}`,
	})
	want := "schema.format_unsupported@inputs/a.json#/format\nschema.keyword_unknown@inputs/a.json#/nope\n" +
		"schema.pattern_unsupported@inputs/b.json#/pattern\nschema.ref_unresolved@inputs/b.json#/$ref"
	if got := describe(fs); got != want {
		t.Errorf("got\n%s\nwant\n%s", got, want)
	}
}

// TestLoaderRefusesEveryURL holds the loader to ADR-010 D5 item 2: it
// loads nothing, whatever the URL.
func TestLoaderRefusesEveryURL(t *testing.T) {
	for _, u := range []string{
		"file:///etc/passwd", "https://example.com/x.json", "http://localhost:1234/x.json",
		"metais:///inputs/a.json", "https://json-schema.org/draft/2020-12/schema", "", "relative.json",
	} {
		doc, err := refuseLoader{}.Load(u)
		if doc != nil || !errors.Is(err, ErrRefused) {
			t.Errorf("Load(%q) = %v, %v; want a refusal", u, doc, err)
		}
	}
}

// TestFileRefNotRead pins the loader's property false first: the bare
// library, with its default FileLoader, reads a file:// reference from
// the host; NewCompiler refuses the same schema with a finding naming the
// reference, and never opens the file.
func TestFileRefNotRead(t *testing.T) {
	dir := testenv.Dir(t)
	host := filepath.Join(dir, "host.json")
	if err := os.WriteFile(host, []byte(`{"type": "string"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	ref := "file://" + filepath.ToSlash(host)
	if !strings.HasPrefix(ref, "file:///") {
		ref = "file:///" + strings.TrimPrefix(ref, "file://")
	}
	schema := `{"$ref": "` + ref + `"}`
	if err := bare(t, schema); err != nil {
		t.Fatalf("control: the bare library did not load the host file: %v", err)
	}
	_, fs := compileFiles(t, limits.Defaults(), map[string]string{"inputs/s.json": schema})
	f := only(t, fs)
	if f.Code() != CodeRefUnresolved || param(t, f, "reference") != ref {
		t.Errorf("got %s naming %q, want %s naming %q", f.Code(), param(t, f, "reference"), CodeRefUnresolved, ref)
	}
	if strings.Contains(f.Message(), "string") {
		t.Errorf("the finding holds the host file's content: %s", f.Message())
	}
}

// TestReferences holds ADR-003 D5 items 3 to 6: what a reference may name,
// and the finding for each it may not, naming the reference.
func TestReferences(t *testing.T) {
	files := map[string]string{
		"inputs/b.json":       `{"$id": "https://example.com/b.json", "type": "string"}`,
		"inputs/sub/d.json":   `{"type": "integer", "$defs": {"e": {"$id": "https://example.com/e", "type": "boolean"}}}`,
		"inputs/sp ace.json":  `{"type": "null"}`,
		"templates/t.json":    `{}`,
		"inputs/readme.md":    `not a schema`,
		"inputs/data.txt":     `not a schema either`,
		"outputs/other.json":  `{"nope": 1}`,
		"inputs/x/notes.yaml": `{"type": "array"}`,
	}
	cases := []struct {
		ref  string
		code finding.Code // "" for a reference that resolves
		id   string
	}{
		{"https://example.com/b.json", "", ""},
		{"https://example.com/b.json#", "", ""},
		{"sub/d.json#/$defs/e", "", ""},
		{"sub/d.json", "", ""},
		{"./sub/../sub/d.json", "", ""},
		{"metais:///inputs/sub/d.json", "", ""},
		{"/inputs/sub/d.json", "", ""},
		{"sp%20ace.json", "", ""},
		{"https://example.com/e", "", ""},
		{"x/notes.yaml", "", ""},
		{"#/$defs/local", "", ""},
		{"https://json-schema.org/draft/2020-12/schema", "", ""},
		{"https://json-schema.org/draft/2020-12/meta/validation", "", ""},
		{"b.json", CodeRefByPath, "https://example.com/b.json"},
		{"metais:///inputs/b.json", CodeRefByPath, "https://example.com/b.json"},
		{"missing.json", CodeRefUnresolved, ""},
		{"https://example.com/missing.json", CodeRefUnresolved, ""},
		{"file:///etc/passwd", CodeRefUnresolved, ""},
		{"http://localhost:1234/draft2020-12/integer.json", CodeRefUnresolved, ""},
		{"https://json-schema.org/draft-07/schema", CodeRefUnresolved, ""},
		{"readme.md", CodeRefUnresolved, ""},
		{"../templates/t.json", CodeRefEscapes, ""},
		{"metais:///templates/t.json", CodeRefEscapes, ""},
		{"../outputs/other.json", CodeRefEscapes, ""},
		{"../../inputs/sub/d.json", CodeRefEscapes, ""},
		{"sub/../../../inputs/b.json", CodeRefEscapes, ""},
	}
	for _, c := range cases {
		fs := map[string]string{"inputs/a.json": `{"$ref": "` + c.ref + `", "$defs": {"local": {}}}`}
		for k, v := range files {
			fs[k] = v
		}
		comp, got := compileFiles(t, limits.Defaults(), fs)
		if c.code == "" {
			if len(got) > 0 || comp == nil {
				t.Errorf("%s: %s", c.ref, describe(got))
			}
			continue
		}
		if len(got) != 1 {
			t.Errorf("%s: %d findings, want 1:\n%s", c.ref, len(got), describe(got))
			continue
		}
		f := got[0]
		l, _ := f.Location()
		if f.Code() != c.code || l.File != "inputs/a.json" || l.Pointer != "/$ref" || param(t, f, "reference") != c.ref {
			t.Errorf("%s: got %s at %s#%s naming %q, want %s at inputs/a.json#/$ref naming it", c.ref, f.Code(), l.File, l.Pointer, param(t, f, "reference"), c.code)
		}
		if c.id != "" && param(t, f, "id") != c.id {
			t.Errorf("%s: id %q, want %q", c.ref, param(t, f, "id"), c.id)
		}
	}
}

// TestUnresolvedFragments refuses a reference to an anchor or a pointer a
// resource does not have, and an $id or anchor the library cannot parse.
func TestUnresolvedFragments(t *testing.T) {
	cases := []struct {
		schema string
		code   finding.Code
	}{
		{`{"$ref": "#nope"}`, CodeRefUnresolved},
		{`{"$ref": "#/$defs/nope"}`, CodeRefUnresolved},
		{`{"$ref": "metais:///inputs/s.json#/$defs/nope"}`, CodeRefUnresolved},
		{`{"$defs": {"a": {"$id": "#foo"}}}`, CodeInvalid},
		{`{"$id": "https://example.com/a#foo"}`, CodeInvalid},
		{`{"$anchor": "1bad"}`, CodeInvalid},
		{`{"$defs": {"a": {"$anchor": "x"}, "b": {"$anchor": "x"}}}`, CodeIDDuplicate},
	}
	for _, c := range cases {
		_, fs := compileFiles(t, limits.Defaults(), map[string]string{"inputs/s.json": c.schema})
		if len(fs) == 0 || codes(fs)[0] != string(c.code) || len(codes(fs)) != 1 {
			t.Errorf("%s: got\n%s\nwant %s", c.schema, describe(fs), c.code)
		}
		for _, f := range fs {
			if l, _ := f.Location(); l.File != "inputs/s.json" {
				t.Errorf("%s: located in %q", c.schema, l.File)
			}
		}
	}
}

// TestCrossFileReferences validates through references between files: by
// canonical $id, by the retrieval URI of a file with none, and to an
// embedded resource of another file.
func TestCrossFileReferences(t *testing.T) {
	s := mustCompile(t, map[string]string{
		"inputs/a.json": `{"properties": {
			"b": {"$ref": "https://example.com/b.json"},
			"d": {"$ref": "sub/d.json"},
			"e": {"$ref": "https://example.com/e"}}}`,
		"inputs/b.json":     `{"$id": "https://example.com/b.json", "type": "string", "minLength": 2}`,
		"inputs/sub/d.json": `{"type": "integer", "$defs": {"e": {"$id": "https://example.com/e", "type": "boolean"}}}`,
	}, "inputs/a.json")
	fails, err := s.Validate(mustValue(t, `{"b": "x", "d": 1.5, "e": 1}`))
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, f := range fails {
		f.Walk(func(_ []*schemafail.Failure, n *schemafail.Failure) bool {
			got = append(got, n.Keyword+" "+n.KeywordLocation+" "+n.Entry+"#"+n.SchemaPointer+" @"+n.Instance())
			return true
		})
	}
	// The library reports a property's failures directly, with no node
	// for properties, which ADR-006 D6 item 5 makes transparent anyway.
	want := []string{
		"$ref /properties/b/$ref inputs/a.json#/properties/b/$ref @/b",
		"minLength /properties/b/$ref/minLength inputs/b.json#/minLength @/b",
		"$ref /properties/d/$ref inputs/a.json#/properties/d/$ref @/d",
		"type /properties/d/$ref/type inputs/sub/d.json#/type @/d",
		"$ref /properties/e/$ref inputs/a.json#/properties/e/$ref @/e",
		"type /properties/e/$ref/type inputs/sub/d.json#/$defs/e/type @/e",
	}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Errorf("got\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
}

// TestIDDuplicate holds ADR-003 D5 item 6: two resources with one URI.
func TestIDDuplicate(t *testing.T) {
	cases := []struct {
		name  string
		files map[string]string
		at    string
		id    string
		other string
	}{
		{"two files with one $id", map[string]string{
			"inputs/a.json": `{"$id": "https://example.com/x"}`,
			"inputs/b.json": `{"$id": "https://example.com/x"}`,
		}, "inputs/b.json#/$id", "https://example.com/x", "inputs/a.json"},
		{"an $id that is another file's retrieval URI", map[string]string{
			"inputs/a.json": `{"$id": "metais:///inputs/b.json"}`,
			"inputs/b.json": `{}`,
		}, "inputs/a.json#/$id", "metais:///inputs/b.json", "inputs/b.json"},
		{"an embedded $id that another file declares", map[string]string{
			"inputs/a.json": `{"$id": "https://example.com/x"}`,
			"inputs/b.json": `{"$defs": {"y": {"$id": "https://example.com/x"}}}`,
		}, "inputs/b.json#/$defs/y/$id", "https://example.com/x", "inputs/a.json"},
		{"a meta-schema's URI", map[string]string{
			"inputs/a.json": `{"$id": "https://json-schema.org/draft/2020-12/schema"}`,
		}, "inputs/a.json#/$id", "https://json-schema.org/draft/2020-12/schema", ""},
	}
	for _, c := range cases {
		_, fs := compileFiles(t, limits.Defaults(), c.files)
		if len(fs) != 1 {
			t.Errorf("%s: %d findings, want 1:\n%s", c.name, len(fs), describe(fs))
			continue
		}
		f := fs[0]
		l, _ := f.Location()
		if f.Code() != CodeIDDuplicate || l.File+"#"+l.Pointer != c.at || param(t, f, "id") != c.id {
			t.Errorf("%s: got %s", c.name, describe(fs))
		}
		other, ok := f.Param("other")
		if s, _ := other.Str(); ok != (c.other != "") || s != c.other {
			t.Errorf("%s: other %q, want %q", c.name, s, c.other)
		}
	}
	// An $id equal to the file's own retrieval URI is the file's one name.
	if _, fs := compileFiles(t, limits.Defaults(), map[string]string{"inputs/a.json": `{"$id": "metais:///inputs/a.json"}`}); len(fs) > 0 {
		t.Errorf("an $id naming the file itself: %s", describe(fs))
	}
}

// TestMetaSchemaFailures reports a schema that is not a valid 2020-12
// schema with schema.invalid at the failing value, naming the
// meta-schema's keyword.
func TestMetaSchemaFailures(t *testing.T) {
	cases := []struct{ schema, ptr, keyword string }{
		{`{"minLength": -1}`, "/minLength", "minimum"},
		{`{"type": "strnig"}`, "/type", "anyOf"},
		{`{"properties": {"a": {"required": "a"}}}`, "/properties/a/required", "type"},
		{`{"items": 3}`, "/items", "type"},
	}
	for _, c := range cases {
		_, fs := compileFiles(t, limits.Defaults(), map[string]string{"inputs/s.json": c.schema})
		if len(fs) == 0 {
			t.Errorf("%s: compiled", c.schema)
			continue
		}
		var found bool
		for _, f := range fs {
			l, _ := f.Location()
			if f.Code() == CodeInvalid && l.File == "inputs/s.json" && l.Pointer == c.ptr {
				found = true
				if k := param(t, f, "keyword"); c.keyword != "" && k != c.keyword && len(fs) == 1 {
					t.Errorf("%s: keyword %q, want %q", c.schema, k, c.keyword)
				}
			}
			if f.Code() != CodeInvalid {
				t.Errorf("%s: %s", c.schema, describe(fs))
			}
		}
		if !found {
			t.Errorf("%s: no schema.invalid at %s:\n%s", c.schema, c.ptr, describe(fs))
		}
	}
}

// TestOnlySchemaFiles compiles the JSON and YAML files under inputs/ and
// nothing else.
func TestOnlySchemaFiles(t *testing.T) {
	for entry, want := range map[string]bool{
		"inputs/a.json": true, "inputs/a.yaml": true, "inputs/a.yml": true, "inputs/x/y/a.json": true,
		"inputs/a.md": false, "inputs/a.JSON": false, "templates/a.json": false, "a.json": false,
		"examples/basic/inputs/a.json": false, "inputs": false,
	} {
		if IsSchemaFile(entry) != want {
			t.Errorf("IsSchemaFile(%q) = %v", entry, !want)
		}
	}
	if got := RetrievalURI("inputs/a b/ü.json"); got != "metais:///inputs/a%20b/%C3%BC.json" {
		t.Errorf("RetrievalURI = %q", got)
	}
}

// TestDecodeFindingsReported passes the reader's findings through, and
// compiles nothing.
func TestDecodeFindingsReported(t *testing.T) {
	tree, err := pkgtreetest.New(map[string][]byte{"inputs/a.json": []byte("{")})
	if err != nil {
		t.Fatal(err)
	}
	want := finding.New("limit.package", finding.Subject{Kind: finding.SubjectPackage},
		&finding.Location{Origin: finding.OriginPackage, File: "inputs/a.json"},
		finding.Params{"limit": finding.String("value-depth"), "bound": finding.Int(64), "lowered": finding.Bool(false)})
	c, fs := NewCompiler(tree, limits.Defaults(), func(entry string, src []byte) (value.Value, []finding.Finding) {
		if entry != "inputs/a.json" || !bytes.Equal(src, []byte("{")) {
			t.Errorf("decode was given %s, %q", entry, src)
		}
		return nil, []finding.Finding{want}
	})
	if c != nil || len(fs) != 1 || fs[0].Code() != want.Code() {
		t.Errorf("got %v, %s", c, describe(fs))
	}
}

// TestSchemaYAMLSize is ADR-009 Amendment A6 item 5, pinned false first: a
// YAML schema file longer than schema-yaml-size is refused with
// limit.package, observing its length, before it is read; one of exactly
// schema-yaml-size compiles, and a JSON file is not bounded by it.
func TestSchemaYAMLSize(t *testing.T) {
	const doc = `{"type": "string"}`
	n := strconv.Itoa(len(doc))
	less := strconv.Itoa(len(doc) - 1)
	for _, c := range []struct {
		entry, lim string
		refused    bool
	}{
		{"inputs/s.yaml", "schema-yaml-size=" + n, false},
		{"inputs/s.yml", "schema-yaml-size=" + n, false},
		{"inputs/s.yaml", "schema-yaml-size=" + less, true},
		{"inputs/s.yml", "schema-yaml-size=" + less, true},
		{"inputs/s.json", "schema-yaml-size=" + less, false},
	} {
		_, fs := compileFiles(t, lower(t, c.lim), map[string]string{c.entry: doc})
		switch {
		case !c.refused && len(fs) > 0:
			t.Errorf("%s at %s: %s", c.entry, c.lim, describe(fs))
		case c.refused && (len(fs) != 1 || fs[0].Code() != finding.Code("limit.package") || param(t, fs[0], "limit") != "schema-yaml-size"):
			t.Errorf("%s at %s: %s, want limit.package naming schema-yaml-size", c.entry, c.lim, describe(fs))
		case c.refused:
			o, _ := fs[0].Param("observed")
			if v, ok := o.Int(); !ok || v != int64(len(doc)) {
				t.Errorf("%s at %s: observed %d, %v", c.entry, c.lim, v, ok)
			}
		}
	}
}
