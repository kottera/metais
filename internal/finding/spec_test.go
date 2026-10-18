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
	changes := []func([]Entry) []Entry{
		func(e []Entry) []Entry { e[0].Hint += "."; return e },
		func(e []Entry) []Entry { e[0].Params[1].Always = !e[0].Params[1].Always; return e },
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
// report's own and internal/limits'. A code a later change emits is
// registered by that change.
func TestEmittedCodes(t *testing.T) {
	var got []Code
	for _, e := range registry {
		got = append(got, e.Code)
	}
	if want := []Code{CodeLimitInvalid, CodeReportTruncated}; !slices.Equal(got, want) {
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
