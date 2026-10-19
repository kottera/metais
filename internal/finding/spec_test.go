// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package finding

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/santhosh-tekuri/jsonschema/v6"

	"github.com/kottera/metais/internal/specdata"
)

const (
	registrySchemaID = "https://metais.org/schemas/1.0/finding-registry.schema.json"
	findingSchemaID  = "https://metais.org/schemas/1.0/finding.schema.json"
)

// noLoader refuses every URL: the schemas are given to the compiler, and
// nothing is fetched (ADR-011 D7 item 7).
type noLoader struct{}

func (noLoader) Load(url string) (any, error) { return nil, errors.New("not loaded: " + url) }

// compile compiles the published schema with id, embedded as b, checking
// it against the 2020-12 meta-schema, and returns the schema at loc.
func compile(t *testing.T, id string, b []byte, loc string) *jsonschema.Schema {
	t.Helper()
	doc, err := jsonschema.UnmarshalJSON(bytes.NewReader(b))
	if err != nil {
		t.Fatalf("%s: %v", id, err)
	}
	c := jsonschema.NewCompiler()
	c.UseLoader(noLoader{})
	if err := c.AddResource(id, doc); err != nil {
		t.Fatalf("%s: %v", id, err)
	}
	s, err := c.Compile(loc)
	if err != nil {
		t.Fatalf("%s does not compile: %v", loc, err)
	}
	return s
}

func validate(t *testing.T, s *jsonschema.Schema, what string, b []byte) {
	t.Helper()
	inst, err := jsonschema.UnmarshalJSON(bytes.NewReader(b))
	if err != nil {
		t.Fatalf("%s is not JSON: %v", what, err)
	}
	if err := s.Validate(inst); err != nil {
		t.Errorf("%s does not validate: %v\n%s", what, err, b)
	}
}

// TestPublishedSchemas checks the two schemas are published as ADR-011 D7
// item 4 says: 2020-12, the $id their path gives, valid against the
// meta-schema, two-space indented, ending in one LF.
func TestPublishedSchemas(t *testing.T) {
	for id, b := range map[string][]byte{registrySchemaID: specdata.FindingRegistrySchema, findingSchemaID: specdata.FindingSchema} {
		compile(t, id, b, id)
		var head struct {
			Schema string `json:"$schema"`
			ID     string `json:"$id"`
		}
		if err := json.Unmarshal(b, &head); err != nil {
			t.Fatal(err)
		}
		if head.Schema != "https://json-schema.org/draft/2020-12/schema" || head.ID != id {
			t.Errorf("%s: $schema %q, $id %q", id, head.Schema, head.ID)
		}
		if !bytes.HasSuffix(b, []byte("}\n")) || bytes.HasSuffix(b, []byte("\n\n")) || bytes.Contains(b, []byte("\r")) || bytes.Contains(b, []byte("\t")) {
			t.Errorf("%s does not end in one LF, or holds a CR or a tab", id)
		}
		for i, line := range strings.Split(string(b), "\n") {
			if n := len(line) - len(strings.TrimLeft(line, " ")); n%2 != 0 {
				t.Errorf("%s line %d is indented %d spaces", id, i+1, n)
			}
		}
		if bytes.Contains(b, []byte(`"$ref": "http`)) {
			t.Errorf("%s has an external $ref", id)
		}
	}
}

func TestRegistryValidates(t *testing.T) {
	s := compile(t, registrySchemaID, specdata.FindingRegistrySchema, registrySchemaID)
	validate(t, s, "spec/findings/registry.json", specdata.FindingRegistry)
	// The schema refuses what the registry must not hold.
	for _, bad := range []string{
		`{}`,
		`{"codes":[{"code":"nofamily.x"}]}`,
		`{"codes":[],"extra":1}`,
		strings.Replace(string(specdata.FindingRegistry), `"location": "none"`, `"location": "end"`, 1),
		strings.Replace(string(specdata.FindingRegistry), `"code": "limit.invalid"`, `"code": "limit"`, 1),
		strings.Replace(string(specdata.FindingRegistry), `"type": "string",`, `"type": "string", "items": "string",`, 1),
	} {
		inst, err := jsonschema.UnmarshalJSON(strings.NewReader(bad))
		if err != nil {
			t.Fatal(err)
		}
		if s.Validate(inst) == nil {
			t.Errorf("the registry schema accepts %.80s", bad)
		}
	}
}

// The registry file's form, as encoding/json reads it.
type jsonRegistry struct {
	Codes []struct {
		Code     string
		Severity string
		Class    string
		Subject  string
		Params   []struct {
			Name    string
			Type    string
			Items   string
			Meaning string
			Always  bool
		}
		Message    string
		Hint       string
		Location   string
		Since      string
		Retired    string
		Statements []string
	}
}

func parseName[T ~uint8](t *testing.T, names []string, s string) T {
	t.Helper()
	if s == "" {
		return 0
	}
	i := slices.Index(names, s)
	if i <= 0 {
		t.Fatalf("unknown name %q", s)
	}
	return T(i)
}

// fromRegistryFile reads spec/findings/registry.json into the tool's
// table form.
func fromRegistryFile(t *testing.T, b []byte) []Entry {
	var r jsonRegistry
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&r); err != nil {
		t.Fatalf("registry.json: %v", err)
	}
	var out []Entry
	for _, c := range r.Codes {
		e := Entry{
			Code:       Code(c.Code),
			Severity:   parseName[Severity](t, severityNames[:], c.Severity),
			Class:      parseName[Class](t, classNames[:], c.Class),
			Subject:    parseName[SubjectKind](t, subjectNames[:], c.Subject),
			Message:    c.Message,
			Hint:       c.Hint,
			Span:       parseName[SpanKind](t, spanNames[:], c.Location),
			Since:      c.Since,
			Retired:    c.Retired,
			Statements: c.Statements,
		}
		for _, p := range c.Params {
			e.Params = append(e.Params, ParamSpec{
				Name:    p.Name,
				Type:    parseName[ParamType](t, paramTypeNames[:], p.Type),
				Items:   parseName[ParamType](t, paramTypeNames[:], p.Items),
				Meaning: p.Meaning,
				Always:  p.Always,
			})
		}
		out = append(out, e)
	}
	return out
}

// TestRegistryMatchesSpec is ADR-006 D13 item 4's first test: the tool's
// table is spec/findings/registry.json, entry for entry and member for
// member.
func TestRegistryMatchesSpec(t *testing.T) {
	if d := registryDiff(fromRegistryFile(t, specdata.FindingRegistry), Registry()); d != "" {
		t.Error(d)
	}
}

// registryDiff describes the first difference between the registry file's
// entries and the tool's, or returns "".
func registryDiff(file, table []Entry) string {
	for i := range max(len(file), len(table)) {
		switch {
		case i >= len(file):
			return fmt.Sprintf("the tool registers %s, which registry.json does not", table[i].Code)
		case i >= len(table):
			return fmt.Sprintf("registry.json registers %s, which the tool does not", file[i].Code)
		case !reflect.DeepEqual(file[i], table[i]):
			return fmt.Sprintf("entry %d differs:\n registry.json %+v\n the tool     %+v", i, file[i], table[i])
		}
	}
	return ""
}

// TestRegistryDiffCatchesChanges is the control: a one-member change on
// either side is caught.
func TestRegistryDiffCatchesChanges(t *testing.T) {
	file := fromRegistryFile(t, specdata.FindingRegistry)
	// two is the first entry with two parameters.
	two := slices.IndexFunc(file, func(e Entry) bool { return len(e.Params) >= 2 })
	changes := []func([]Entry) []Entry{
		func(e []Entry) []Entry { e[0].Hint += "."; return e },
		func(e []Entry) []Entry { e[two].Params[1].Always = !e[two].Params[1].Always; return e },
		func(e []Entry) []Entry { e[1].Severity = SeverityWarning; return e },
		func(e []Entry) []Entry { e[0].Statements = append(e[0].Statements, "MTS-SEC-001"); return e },
		func(e []Entry) []Entry { return e[:1] },
		func(e []Entry) []Entry { return append(e, e[0]) },
	}
	for i, change := range changes {
		if registryDiff(file, change(Registry())) == "" {
			t.Errorf("change %d is not caught", i)
		}
	}
	changed := strings.Replace(string(specdata.FindingRegistry), `"since": "1.0"`, `"since": "1.1"`, 1)
	if registryDiff(fromRegistryFile(t, []byte(changed)), Registry()) == "" {
		t.Error("a change to registry.json is not caught")
	}
}

var placeholder = regexp.MustCompile(`\{\{|\}\}|\{([^{}]*)\}`)

// TestRegistryEntries checks every entry is one New can build: codes in
// order and unique, families from the closed list, and templates that name
// only parameters always present and the built-ins the subject has.
func TestRegistryEntries(t *testing.T) {
	codeSyntax := regexp.MustCompile(`^[a-z][a-z0-9_]*(\.[a-z][a-z0-9_]*){1,3}$`)
	for i, e := range registry {
		if i > 0 && registry[i-1].Code >= e.Code {
			t.Errorf("%s is out of order or repeated", e.Code)
		}
		if !codeSyntax.MatchString(string(e.Code)) || len(e.Code) > 64 {
			t.Errorf("%s is not a code", e.Code)
		}
		for _, tmpl := range []string{e.Message, e.Hint} {
			if strings.ContainsAny(tmpl, "\n\r") {
				t.Errorf("%s: a template holds a line break", e.Code)
			}
			for _, m := range placeholder.FindAllStringSubmatch(tmpl, -1) {
				name := m[1]
				switch {
				case m[0] == "{{" || m[0] == "}}", name == "field", name == "package":
				case name == "input":
					if e.Subject != SubjectInput && e.Subject != SubjectSchema {
						t.Errorf("%s: {input} in a finding about a %s", e.Code, e.Subject)
					}
				case name == "output":
					if e.Subject != SubjectOutput {
						t.Errorf("%s: {output} in a finding about a %s", e.Code, e.Subject)
					}
				default:
					if !slices.ContainsFunc(e.Params, func(p ParamSpec) bool { return p.Name == name && p.Always }) {
						t.Errorf("%s: {%s} names no parameter that is always present", e.Code, name)
					}
				}
			}
		}
	}
}

// TestEmittedCodes pins the registry to the codes this module emits: the
// report's own, internal/limits', internal/cli's, internal/zipfmt's
// container and entry-name refusals, internal/inputs/csv's refusals of a
// CSV input, internal/markdown's refusal of an unclosed frontmatter block,
// internal/manifest's version gate and manifest rules,
// internal/schemafail's mapping of JSON Schema failures in each of its
// three families (ADR-006 D6 item 1), internal/schema's refusals of a
// schema file, internal/xref's cross-reference (ADR-007 D3),
// internal/inputs' refusals of a supplied input, internal/author/ooxml's
// split runs (ADR-007 D4), and internal/sample's unsatisfiable field
// (ADR-007 D9), and check's steps (ADR-007 D2). A code a later change emits is registered by that change.
func TestEmittedCodes(t *testing.T) {
	var got []Code
	for _, e := range registry {
		got = append(got, e.Code)
	}
	want := []Code{
		"csv.after_quote", "csv.bare_quote", "csv.field_count", "csv.header_duplicate",
		"csv.header_empty", "csv.header_missing", "csv.unterminated_quote",
		CodeInternalError, CodeLimitExceeded, CodeLimitInvalid, CodeLimitPackage, CodeReportTruncated,
		"input.encoding_invalid", "input.number_out_of_range",
		"json.duplicate_name", "json.lone_surrogate", "json.noncharacter", "json.syntax", "json.trailing_content",
		"markdown.frontmatter_unclosed",
		"schema.dialect", "schema.format_unsupported", "schema.id_duplicate", "schema.invalid",
		"schema.keyword_unknown", "schema.pattern_unsupported", "schema.ref_by_path",
		"schema.ref_escapes", "schema.ref_unresolved", "schema.sample_unsatisfiable",
		CodeUsageBadValue, CodeUsageConflictingFlags, CodeUsageDataDirUnreadable, CodeUsageMissingCommand, CodeUsageRepeatedFlag, CodeUsageUnexpectedOperand,
		CodeUsageUnknownCommand, CodeUsageUnknownFlag, CodeUsageUnknownTopic,
		CodeUsageNameInvalid, CodeUsageNameRequired,
		CodeDestinationNotDirectory, CodeDestinationNotEmpty, CodeDestinationWriteFailed,
	}
	for _, c := range []string{
		"archive_comment", "crc_mismatch", "data_descriptor", "deflate_stream", "encrypted", "entry_order",
		"extra_field", "field_value", "flags", "header_mismatch", "method", "not_zip", "overlap_or_gap",
		"prepended_data", "second_end_record", "size_mismatch", "trailing_data", "zip64",
	} {
		want = append(want, Code("package.container."+c))
	}
	for _, c := range []string{
		"case_collision", "character", "directory_entry", "duplicate", "encoding", "file_directory_conflict",
		"not_nfc", "not_portable", "shape",
	} {
		want = append(want, Code("package.name."+c))
	}
	for _, c := range []string{"malformed", "missing", "not_string", "unsupported_major", "unsupported_minor"} {
		want = append(want, Code("package.version."+c))
	}
	want = append(want, "manifest.input.duplicate_name", "manifest.license.expression",
		"manifest.output.duplicate_id", "manifest.output.nested_for_each")
	for _, family := range []string{"input", "manifest", "output"} {
		for _, c := range []string{
			"additional_item", "additional_property", "any_of", "const", "contains", "dependent_required", "enum",
			"exclusive_maximum", "exclusive_minimum", "forbidden", "format", "max_contains", "max_items", "max_length",
			"max_properties", "maximum", "min_contains", "min_items", "min_length", "min_properties", "minimum",
			"multiple_of", "not", "one_of", "one_of.ambiguous", "pattern", "property_name", "required", "type",
			"unevaluated_item", "unevaluated_property", "unique_items",
		} {
			want = append(want, Code(family+"."+c))
		}
	}
	want = append(want, CodeInputValidationPartial)
	for _, c := range []string{
		"alias", "anchor", "bom", "character", "directive", "duplicate_key", "key_empty", "key_not_scalar",
		"merge_key", "multi_document", "scalar_refused", "syntax", "tag",
	} {
		want = append(want, Code("manifest.yaml."+c), Code("yaml."+c))
	}
	want = append(want, "manifest.number_out_of_range", "manifest.yaml.encoding", "manifest.yaml.root_not_mapping")
	want = append(want, "template.ooxml.split_construct", "template.ooxml.split_delimiter")
	for _, c := range []string{
		"budget_exhausted", "dynamic_reference", "for_each_not_array", "index_out_of_range", "missing_asset",
		"missing_partial", "not_an_array", "not_an_object", "partial_parameter_not_passed", "unknown_field",
		"unknown_input", "unknown_variable", "unused_input", "unused_template",
	} {
		want = append(want, Code("template.xref."+c))
	}
	// internal/inputs, the inputs layer (W3-2).
	want = append(want, "collection.as_file", "collection.directory_name", "collection.item_name",
		"collection.stem_duplicate", "collection.subdirectory",
		"input.constraints_failed", "input.constraints_not_evaluated", "input.extension_not_accepted",
		"input.file_duplicate", "input.file_unknown", "input.json_arg_invalid", "input.json_arg_not_object",
		"input.json_arg_unknown", "input.link", "input.missing", "input.noncharacter", "input.not_readable",
		"input.not_regular_file", "input.set_into_unparsed", "input.set_pointer_invalid", "input.set_traversal",
		"markdown.frontmatter_body_field", "markdown.frontmatter_not_mapping")
	// The package loader's and inspect's (W3-1).
	want = append(want, "package.not_found", "package.not_readable",
		"package.layout.unknown_top_level", "package.layout.missing_manifest", "package.layout.compiled_in_source",
		"package.layout.empty_directory", "package.layout.non_regular_file",
		"package.compiled.missing", "package.compiled.unknown_file", "package.compiled.mismatch",
		"package.digest.not_canonical", "package.digest.entries_mismatch", "package.digest.mismatch", "package.digest.package_mismatch",
		"manifest.file.wrong_name", "manifest.path_not_found", "schema.syntax",
		"template.syntax", "template.include", "template.unknown_filter", "template.number_range", "template.tag_context",
		"template.filter_argument", "template.filter_input", "template.division_by_zero",
		CodeUsageMissingOperand, CodeUsagePackageFromStdin)
	// unpack's (W4-5).
	want = append(want, CodeUsageNotPackedPackage)
	// internal/template and render (W4-3).
	for _, c := range []string{
		"comparison", "invalid_utf8", "nil_property", "object_output", "partial_name",
		"partial_not_found", "reserved_name", "tag_argument", "undefined_property", "undefined_variable",
	} {
		want = append(want, Code("template."+c))
	}
	// Escaping (W5-2).
	want = append(want, "template.safe_position")
	// The METAIS filters and date (W5-1).
	want = append(want, "template.date_format", "template.date_input", "template.slug_empty")
	for _, c := range []string{
		"case_collision", "character", "duplicate", "encoding", "file_directory_conflict", "not_nfc",
		"not_portable", "shape",
	} {
		want = append(want, Code("output.path."+c))
	}
	want = append(want, CodeLimitExceededOutput, CodeLimitExceededTemplate, CodeUsageMissingFlag,
		"input.raw_unavailable")
	for _, c := range []string{"syntax", "trailing_content", "duplicate_name", "lone_surrogate", "noncharacter", "encoding"} {
		want = append(want, Code("manifest.json."+c))
	}
	// check's (W4-4).
	want = append(want, "report.step_skipped", "package.layout.os_litter", "package.layout.not_packed",
		"manifest.guide_link_missing", "schema.invalid_default", "schema.invalid_example", "schema.unsatisfiable",
		"schema.unusable_enum_member", "schema.required_not_declared", "template.not_utf8", "template.crlf",
		"example.layout", "manifest.input.form_shape", "manifest.input.collection_shape", "output.declaration.validate_kind")
	// The embed and provenance tags (W5-3).
	want = append(want, "template.embed_index", "template.embed_unknown_input", "template.provenance_argument",
		"template.provenance_style")
	// Containers' (W5-6).
	want = append(want, CodeContainerMissingEntry, CodeContainerNonASCII, CodeContainerReservedEntry, CodeContainerSpace,
		CodeContainerTooManyEntries)
	// Output validation's (W5-7).
	want = append(want, "output.parse")
	// The reading commands' (W4-2).
	want = append(want, CodeUsageBadGlob, CodeUsageBadPattern, CodeUsageBinaryToTerminal, CodeUsageEntryNotFound,
		CodeUsageIsADirectory, CodeUsageLineOutOfRange, CodeUsageNoMatch, CodeUsageNotText, CodeUsageUnknownInput)
	// render's fan-out (W5-4).
	want = append(want, "output.expression.for_each_type")
	// Assets, the mirror and the asset tag (W5-5).
	want = append(want, "output.declaration.empty_mirror", "output.declaration.empty_name", "template.asset_ambiguous",
		"template.asset_not_found", "template.asset_not_text", "template.asset_not_written", "template.asset_tree")
	slices.Sort(want)
	if !slices.Equal(got, want) {
		t.Errorf("registered %q, want %q", got, want)
	}
}

// TestEncodingValidates holds the JSON encoding to the published finding
// schema, over every member the model can give.
func TestEncodingValidates(t *testing.T) {
	withTestCodes(t)
	finding := compile(t, findingSchemaID, specdata.FindingSchema, findingSchemaID)
	summary := compile(t, findingSchemaID, specdata.FindingSchema, findingSchemaID+"#/$defs/summary")

	branch := New(codeInputType, input("v"), dataDir("v.json", 2, 3, "/kind"), Params{"expected": List(String("string")), "actual": String("number")})
	argLoc := Location{Origin: OriginInputsJSON, Argument: 1}.WithPointer("")
	r := NewReport(4)
	all := append(orderedFindings(), wcag(),
		New(codeInputType, input("v"), &argLoc, Params{"expected": List(String("object")), "actual": String("array")},
			WithAlternatives(Alternative{Branch: 0, Title: "A card", Findings: []Finding{branch}}, Alternative{Branch: 1, Omitted: 2}),
			WithVia(Via{Keyword: "anyOf", SchemaPointer: "/anyOf", Branch: 1, Title: "B"}),
			WithRelated(Related{Location: Location{Origin: OriginDataDir, File: "v.json", Line: 1, Column: 1}.WithPointer("/0"), Note: "first occurrence"})),
		New(CodeLimitInvalid, Subject{Kind: SubjectInvocation}, nil, Params{"reason": String("above_default"), "name": String("findings"), "default": Int(100000), "unit": String("count"), "value": Int(200000)}),
	)
	for _, f := range all {
		r.Add(f)
		validate(t, finding, string(f.Code()), f.appendJSON(nil))
	}
	for _, f := range r.Findings() {
		validate(t, finding, string(f.Code()), f.appendJSON(nil))
	}
	validate(t, summary, "the summary", r.Summary().AppendJSON(nil))

	// The schema refuses what a finding must not hold.
	for _, bad := range []string{
		`{"code":"input.x","severity":"error","message":"m","hint":"h","subject":{"kind":"input"}}`,
		`{"code":"input.x","severity":"fatal","message":"m","hint":"h","subject":{"kind":"input"},"params":{}}`,
		`{"code":"input.x","severity":"error","message":"m","hint":"h","subject":{"kind":"input","name":""},"params":{}}`,
		`{"code":"input.x","severity":"error","message":"m","hint":"h","subject":{"kind":"input"},"params":{},"location":{"origin":"set","line":0}}`,
		`{"code":"input.x","severity":"error","message":"m","hint":"h","subject":{"kind":"input"},"params":{},"location":{"origin":"set","pointer":"a"}}`,
		`{"code":"input.x","severity":"error","message":"m","hint":"h","subject":{"kind":"input"},"params":{},"location":{"origin":"set","note":"n"}}`,
		`{"code":"input.x","severity":"error","message":"m","hint":"h","subject":{"kind":"input"},"params":{"a":{"b":1}}}`,
		`{"code":"input.x","severity":"error","message":"m","hint":"h","subject":{"kind":"input"},"params":{},"via":[{"keyword":"then","schema_pointer":"/then"}]}`,
		`{"code":"input.x","severity":"error","message":"m","hint":"h","subject":{"kind":"input"},"params":{},"extra":1}`,
	} {
		inst, err := jsonschema.UnmarshalJSON(strings.NewReader(bad))
		if err != nil {
			t.Fatal(err)
		}
		if finding.Validate(inst) == nil {
			t.Errorf("the finding schema accepts %s", bad)
		}
	}
}
