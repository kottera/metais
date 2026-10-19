// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/kottera/metais/internal/cli/args"
	"github.com/kottera/metais/internal/finding"
	"github.com/kottera/metais/internal/hostenv"
	"github.com/kottera/metais/internal/testenv"
)

// writeTree writes files, by slash-separated path, under a new directory
// and returns it.
func writeTree(t testing.TB, files map[string]string) string {
	t.Helper()
	dir := testenv.Dir(t)
	for name, body := range files {
		p := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

// supplyPackage is a source directory with a required object input, an
// optional one with a default, and an optional collection.
func supplyPackage(t testing.TB) string {
	return writeTree(t, map[string]string{
		"metais.yaml": `metais: "1.0"
name: org.example.supply
version: 1.0.0
title: Supply
description: A package for supplying inputs.
guide: Supply a vendor.
inputs:
  - name: vendor
    title: Vendor
    schema: inputs/vendor.schema.json
  - name: options
    title: Options
    schema: inputs/options.schema.json
    required: false
  - name: notes
    title: Notes
    schema: inputs/notes.schema.json
    collection: true
    forms: [markdown]
    required: false
outputs:
  - id: report
    template: report.md.liquid
`,
		"inputs/vendor.schema.json": `{"type": "object", "required": ["name"],
  "properties": {"name": {"title": "Name", "type": "string", "description": "The vendor's name.\u001b[2J\u202e"},
    "size": {"type": "integer", "maximum": 100}, "tags": {"type": "array"}, "nested": {"$ref": "#/$defs/node"}},
  "$defs": {"node": {"type": "object", "additionalProperties": {"$ref": "#/$defs/node"}}}}`,
		"inputs/options.schema.json": `{"type": "object", "default": {"mode": "full"}, "properties": {"mode": {"type": "string"}, "depth": {"type": "integer", "default": 2}}}`,
		"inputs/notes.schema.json":   `{"type": "array", "items": {"type": "object", "required": ["title"]}}`,
		"templates/report.md.liquid": "{{ inputs.vendor.name }}\n",
	})
}

// validateDoc runs validate with --json and returns its run and result.
func validateDoc(t *testing.T, stdin string, argv ...string) (ran, validateResult) {
	t.Helper()
	var in = strings.NewReader(stdin)
	r := runIn(t, hostenv.Display{NoColor: true}, in, 0, append([]string{"validate", "--json"}, argv...)...)
	var res validateResult
	if string(r.doc.Result) != "null" {
		if err := json.Unmarshal(r.doc.Result, &res); err != nil {
			t.Fatalf("%s: %v", r.stdout, err)
		}
	}
	return r, res
}

// TestValidateSupplyOrder is ADR-008 D7 and ADR-003 D18 through validate:
// the data directory, then every --inputs-json, then the path sets in
// command-line order, each over what came before; the result says where
// each value came from, and --resolved gives the data model with the
// defaults applied, absent inputs absent, in canonical key order.
func TestValidateSupplyOrder(t *testing.T) {
	testenv.Home(t)
	pkg := supplyPackage(t)
	data := writeTree(t, map[string]string{"vendor.json": `{"name": "From file", "size": 3}`})
	r, res := validateDoc(t, `{"options": {"mode": "lite"}}`, pkg, "--data", data,
		"--set-json", "/vendor/tags=[1]",
		"--inputs-json", "-",
		"--set", "/vendor/size=5",
		"--set-json", "/vendor/size=6",
		"--resolved")
	if r.status != 0 || !r.doc.OK || len(r.doc.Findings) != 0 {
		t.Fatalf("exit %d: %s", r.status, r.stdout)
	}
	got, _ := json.Marshal(res.Inputs)
	want := `[{"name":"vendor","supplied":true,"from":[{"source":"data_dir","file":"vendor.json"},` +
		`{"source":"set_json","argument":1},{"source":"set","argument":2},{"source":"set_json","argument":3}]},` +
		`{"name":"options","supplied":true,"from":[{"source":"inputs_json","argument":1},{"source":"default"}]},` +
		`{"name":"notes","supplied":false,"from":[]}]`
	if string(got) != want {
		t.Errorf("inputs = %s\nwant     %s", got, want)
	}
	if res.Data == nil || string(*res.Data) != `{"options":{"depth":2,"mode":"lite"},"vendor":{"name":"From file","size":6,"tags":[1]}}` {
		t.Errorf("data = %s", string(*res.Data))
	}
	if len(r.doc.Next) != 0 {
		t.Errorf("next = %v: validate's step after success is render, which this tool does not declare yet", r.doc.Next)
	}
	// An --inputs-json member replaces the file's whole value, which
	// then gives none of it; an optional input absent is given its
	// default. Without --resolved, data is absent.
	_, res = validateDoc(t, "", pkg, "--data", data, "--inputs-json", `{"vendor": {"name": "Inline"}}`)
	got, _ = json.Marshal(res.Inputs[:2])
	if string(got) != `[{"name":"vendor","supplied":true,"from":[{"source":"inputs_json","argument":1}]},{"name":"options","supplied":false,"from":[{"source":"default"}]}]` {
		t.Errorf("inputs = %s", got)
	}
	if res.Data != nil {
		t.Errorf("data without --resolved: %s", *res.Data)
	}
}

// TestValidateNothingSupplied is ADR-008 D7 item 3: no supply flag
// validates nothing supplied, which reports every required input missing
// as a result, exit 1, and points at validating again.
func TestValidateNothingSupplied(t *testing.T) {
	testenv.Home(t)
	r, res := validateDoc(t, "", supplyPackage(t))
	if r.status != finding.ExitFindings || r.doc.OK || !slices.Equal(r.codes(), []string{"input.missing"}) {
		t.Errorf("exit %d, codes %q", r.status, r.codes())
	}
	if len(res.Inputs) != 3 || res.Inputs[0].Supplied {
		t.Errorf("result = %+v", res)
	}
	if len(r.doc.Next) == 0 || !strings.HasPrefix(r.doc.Next[0].Run, "metais validate <package>") {
		t.Errorf("next = %v", r.doc.Next)
	}
}

// TestValidateHuman is ADR-008 D11 item 3 and ADR-006 D11 for validate:
// the findings are the result, on standard output with the summary line
// and the Next: block; the at line prints the inputs pointer --set takes
// (R2-093); package text is neutralised (R2-145); --resolved is indented
// JSON after the findings.
func TestValidateHuman(t *testing.T) {
	testenv.Home(t)
	pkg := supplyPackage(t)
	data := writeTree(t, map[string]string{"vendor.json": `{"size": 500}`})
	r := runTool(t, "validate", pkg, "--data", data, "--resolved")
	if r.status != finding.ExitFindings || r.stderr != "" {
		t.Fatalf("exit %d, stderr %q", r.status, r.stderr)
	}
	for _, want := range []string{
		"vendor.json:1:1: error: Name is required. [input.required]\n  at /vendor/name (input vendor: Vendor)\n  field: Name — The vendor's name.\\u001b[2J\\u202e\n",
		"vendor.json:1:10: error: `/size` must be at most 100. [input.maximum]\n  at /vendor/size (input vendor: Vendor)\n",
		"2 errors, 0 warnings, 0 info\n{\n  \"options\": {\n",
		"\nNext:\n  metais validate <package> --data <data dir>",
	} {
		if !strings.Contains(r.stdout, want) {
			t.Errorf("standard output lacks %q:\n%s", want, r.stdout)
		}
	}
	if strings.ContainsAny(r.stdout, "\x1b\u202e") {
		t.Error("a control character reached standard output")
	}
	// A clean run prints the summary alone, which is the report.
	ok := runTool(t, "validate", pkg, "--set", "/vendor/name=Acme")
	if ok.status != 0 || ok.stdout != "0 errors, 0 warnings, 0 info\n" || ok.stderr != "" {
		t.Errorf("a clean run: exit %d, %q, %q", ok.status, ok.stdout, ok.stderr)
	}
}

// TestValidateRefusals is ADR-008 D16 item 2 for validate's own refusals:
// each is triggered, refused before any input is read, with a null result
// and its declared next step (runTool holds the declaration).
func TestValidateRefusals(t *testing.T) {
	testenv.Home(t)
	pkg := supplyPackage(t)
	file := writeTree(t, map[string]string{"f": "x"})
	for _, c := range []struct {
		argv   []string
		code   string
		status int
		params map[string]any
	}{
		{[]string{"-"}, "usage.package_from_stdin", 3, nil},
		{[]string{}, "usage.missing_operand", 3, nil},
		{[]string{pkg, "--data", filepath.Join(file, "absent")}, "usage.data_dir_unreadable", 3, map[string]any{"argument": 4.0, "reason": "not_found"}},
		{[]string{pkg, "--data", filepath.Join(file, "f")}, "usage.data_dir_unreadable", 3, map[string]any{"argument": 4.0, "reason": "not_regular_file"}},
		{[]string{pkg, "--inputs-json", "-", "--set", "/vendor/name=a", "--inputs-json", "-"}, "usage.conflicting_flags", 3, map[string]any{"argument": 8.0, "flag": "--inputs-json"}},
		{[]string{pkg, "--limit", "findings=x"}, "limit.invalid", 3, nil},
		{[]string{pkg, "--fail-on", "never"}, "usage.bad_value", 3, nil},
		{[]string{filepath.Join(file, "absent")}, "package.not_found", 4, nil},
	} {
		r, _ := validateDoc(t, "", c.argv...)
		if r.status != c.status || !slices.Equal(r.codes(), []string{c.code}) || string(r.doc.Result) != "null" {
			t.Errorf("validate %q: exit %d, codes %q, result %s", c.argv, r.status, r.codes(), r.doc.Result)
			continue
		}
		for k, v := range c.params {
			if r.doc.Findings[0].Params[k] != v {
				t.Errorf("validate %q: %s = %v, want %v", c.argv, k, r.doc.Findings[0].Params[k], v)
			}
		}
	}
}

// TestValidateStandardInputBound is ADR-008 D7 item 2: --inputs-json -
// reads standard input within input-size, and more is refused as an
// argument over the limit.
func TestValidateStandardInputBound(t *testing.T) {
	testenv.Home(t)
	pkg := supplyPackage(t)
	r, _ := validateDoc(t, `{"vendor": {"name": "`+strings.Repeat("a", 600)+`"}}`, pkg, "--inputs-json", "-", "--limit", "input-size=500")
	if !slices.Equal(r.codes(), []string{"input.json_arg_invalid"}) || r.doc.Findings[0].Params["limit"] != "input-size" {
		t.Errorf("codes %q, %v", r.codes(), r.doc.Findings)
	}
}

// TestValidateCap is ADR-006 D8 items 5 and 6 through validate: with the
// findings limit lowered to 3, ten errors list three, then
// report.truncated with the rest, and the summary counts every one.
func TestValidateCap(t *testing.T) {
	testenv.Home(t)
	pkg := writeTree(t, map[string]string{
		"metais.yaml": "metais: \"1.0\"\nname: org.example.cap\nversion: 1.0.0\ntitle: Cap\ndescription: Ten errors.\n" +
			"guide: Supply them.\ninputs:\n  - name: rows\n    title: Rows\n    schema: inputs/rows.schema.json\n" +
			"outputs:\n  - id: report\n    template: report.md.liquid\n",
		"inputs/rows.schema.json":    `{"type": "array", "items": {"type": "integer"}}`,
		"templates/report.md.liquid": "x\n",
	})
	r, _ := validateDoc(t, "", pkg, "--set-json", `/rows=["a","b","c","d","e","f","g","h","i","j"]`, "--limit", "findings=3")
	if got := r.codes(); !slices.Equal(got, []string{"input.type", "input.type", "input.type", "report.truncated"}) {
		t.Fatalf("codes %q", got)
	}
	if p := r.doc.Findings[3].Params; p["listed"] != 3.0 || p["omitted"] != 7.0 {
		t.Errorf("report.truncated params %v", p)
	}
	var sum struct{ Errors, Info, Listed, Omitted int }
	if err := json.Unmarshal(r.doc.Summary, &sum); err != nil || sum.Errors != 10 || sum.Info != 1 || sum.Listed != 3 || sum.Omitted != 7 {
		t.Errorf("summary %s", r.doc.Summary)
	}
	h := runTool(t, "validate", pkg, "--set-json", `/rows=["a","b","c","d","e","f","g","h","i","j"]`, "--limit", "findings=3")
	if !strings.Contains(h.stdout, "10 errors, 0 warnings, 1 info (3 listed, 7 more not shown; use --json for the totals by code)\n") {
		t.Errorf("human summary:\n%s", h.stdout)
	}
}

// TestValidateMessagesBounded is W4-1.6: no message validate prints
// exceeds ADR-006 D9 item 1's 240 code points, a {field} on a pointer
// longer than that included, and the finding's location keeps the whole
// pointer.
func TestValidateMessagesBounded(t *testing.T) {
	testenv.Home(t)
	pkg := supplyPackage(t)
	nested := strings.Repeat(`{"level-of-nesting": `, 30) + `"x"` + strings.Repeat("}", 30)
	r := runTool(t, "validate", "--json", pkg, "--set", "/vendor/name=a", "--set-json", "/vendor/nested="+nested)
	var doc struct {
		Findings []struct {
			Code     string
			Message  string
			Hint     string
			Location struct{ Pointer string }
		}
	}
	if err := json.Unmarshal([]byte(r.stdout), &doc); err != nil {
		t.Fatal(err)
	}
	if len(doc.Findings) == 0 {
		t.Fatalf("no finding: %s", r.stdout)
	}
	long := false
	for _, f := range doc.Findings {
		if n := utf8.RuneCountInString(f.Message); n > 240 {
			t.Errorf("%s: a message of %d code points", f.Code, n)
		}
		if n := utf8.RuneCountInString(f.Hint); n > 240 {
			t.Errorf("%s: a hint of %d code points", f.Code, n)
		}
		long = long || len(f.Location.Pointer) > 240
	}
	if !long {
		t.Errorf("no finding carries a pointer longer than the bound: %s", r.stdout)
	}
}

// TestFailOn is R2-081 and R2-134: --fail-on is declared once, on every
// command that reports findings, and nowhere else; under warning, a
// warning fails the command with exit status 1 and ok false, and no
// finding's severity changes.
func TestFailOn(t *testing.T) {
	testenv.Home(t)
	for _, c := range table {
		reports := c.Role != finding.RoleNoPackage || c.WritesSource
		if got := slices.ContainsFunc(c.Flags, func(f args.Flag) bool { return f.Name == failOnFlag.Name }); got != reports {
			t.Errorf("%s declares --fail-on: %v, want %v", c.Name, got, reports)
		}
	}
	// A command whose run reports one warning: the tool's mechanism, as
	// every command meets it.
	warns := &Command{Name: "warns", Group: "The tool", Synopsis: "warns", Role: finding.RoleAuthoring,
		Flags: []args.Flag{failOnFlag}, Format: "metais.warns/1",
		run: func(*session, args.Parsed) outcome {
			loc := finding.Location{Origin: finding.OriginPackage, File: "metais.yaml"}.WithPointer("/inputs/0")
			f := finding.New("template.xref.unused_input", finding.Subject{Kind: finding.SubjectInput, Name: "a"}, &loc, finding.Params{})
			return outcome{findings: []finding.Finding{f}, result: &struct{}{}}
		}}
	t2 := []*Command{warns}
	for _, c := range []struct {
		argv   []string
		status int
		ok     bool
	}{
		{[]string{"warns", "--json"}, 0, true},
		{[]string{"warns", "--json", "--fail-on", "error"}, 0, true},
		{[]string{"warns", "--json", "--fail-on", "warning"}, finding.ExitFindings, false},
		{[]string{"warns", "--fail-on=warning"}, finding.ExitFindings, false},
	} {
		var out, errs strings.Builder
		status := run(context.Background(), Invocation{Args: c.argv, Stdin: strings.NewReader(""), Stdout: &out, Stderr: &errs}, t2)
		if status != c.status {
			t.Errorf("%q: exit %d, want %d", c.argv, status, c.status)
		}
		if strings.Contains(strings.Join(c.argv, " "), "--json") {
			var doc struct {
				OK       bool
				Findings []struct{ Severity string }
			}
			if err := json.Unmarshal([]byte(out.String()), &doc); err != nil {
				t.Fatal(err)
			}
			if doc.OK != c.ok || len(doc.Findings) != 1 || doc.Findings[0].Severity != "warning" {
				t.Errorf("%q: %s", c.argv, out.String())
			}
		}
	}
	// --fail-on warning changes nothing without a warning.
	r := runTool(t, "validate", "--json", supplyPackage(t), "--set", "/vendor/name=a", "--fail-on", "warning")
	if r.status != 0 || !r.doc.OK {
		t.Errorf("validate --fail-on warning with no warning: exit %d", r.status)
	}
	if r := runTool(t, "inspect", "--fail-on", "warning", "--json", filepath.Join(vectorsDir, "pkg/opens-source/package")); r.status != 0 {
		t.Errorf("inspect --fail-on warning: exit %d", r.status)
	}
}

// TestValidateLinkedDataDirectory is ADR-009 D7 item 4 through validate:
// the data directory is the caller's own name, and is followed when it is
// a link, while a link inside it is not.
func TestValidateLinkedDataDirectory(t *testing.T) {
	testenv.Home(t)
	pkg := supplyPackage(t)
	real := writeTree(t, map[string]string{"vendor.json": `{"name": "Acme"}`})
	linked := filepath.Join(testenv.Dir(t), "linked")
	if err := os.Symlink(real, linked); err != nil {
		t.Skipf("symbolic links: %v", err)
	}
	if r, _ := validateDoc(t, "", pkg, "--data", linked); r.status != 0 || len(r.doc.Findings) != 0 {
		t.Errorf("a linked data directory: exit %d, codes %q", r.status, r.codes())
	}
	if err := os.Symlink(filepath.Join(real, "vendor.json"), filepath.Join(real, "options.json")); err != nil {
		t.Fatal(err)
	}
	if r, _ := validateDoc(t, "", pkg, "--data", linked); !slices.Equal(r.codes(), []string{"input.link"}) {
		t.Errorf("a link inside it: codes %q", r.codes())
	}
}

// TestValidateEchoesNoInput is ADR-006 D9 item 4 through validate, over
// the conformance suite's no-echo vector, whose input strings are each one
// word: no message, hint or parameter, in JSON or in human form, repeats
// it. The vector compares the parameters; this test reads the wording,
// which a vector never compares.
func TestValidateEchoesNoInput(t *testing.T) {
	testenv.Home(t)
	dir := filepath.Join(vectorsDir, "fnd/no-echo")
	for _, argv := range [][]string{
		{"validate", "--json", filepath.Join(dir, "package"), "--data", filepath.Join(dir, "data/sentinels")},
		{"validate", filepath.Join(dir, "package"), "--data", filepath.Join(dir, "data/sentinels")},
	} {
		r := runTool(t, argv...)
		if r.status != finding.ExitFindings {
			t.Fatalf("%q: exit %d", argv, r.status)
		}
		if strings.Contains(r.stdout+r.stderr, "CONFIDENTIALWORD") || strings.Contains(r.stdout, "123456") {
			t.Errorf("%q echoes an input value:\n%s", argv, r.stdout)
		}
	}
}

// TestValidateEscapesPackageText is ADR-006 D9 item 9 and D10 item 1
// through validate, over the neutralisation vector: a field's title and
// description holding ESC, a line break and bidirectional controls are
// written as \u escapes in the JSON report and in the human one, and
// neither stream holds the characters themselves.
func TestValidateEscapesPackageText(t *testing.T) {
	testenv.Home(t)
	dir := filepath.Join(vectorsDir, "fnd/neutralised-text")
	j := runTool(t, "validate", "--json", filepath.Join(dir, "package"), "--set-json", `/v={"name": 1}`)
	h := runTool(t, "validate", filepath.Join(dir, "package"), "--set-json", `/v={"name": 1}`)
	if !strings.Contains(j.stdout, `"field":{"title":"Name\u001b[31m","description":"Line one\u000aline two \u202eevil\u2066"}`) {
		t.Errorf("the JSON report:\n%s", j.stdout)
	}
	if !strings.Contains(h.stdout, `field: Name\u001b[31m — Line one…`) {
		t.Errorf("the human report:\n%s", h.stdout)
	}
	for _, out := range []string{j.stdout, h.stdout} {
		if strings.ContainsAny(out, "\x1b‮⁦") {
			t.Errorf("a control character reached standard output: %q", out)
		}
	}
}
