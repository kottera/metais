// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"encoding/json"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"

	"metais.org/metais/internal/testenv"
)

// TestSchemaFields is metais schema --fields (ADR-008 A7) over the shipped
// examples' nested inputs: one tab-separated line per field, its pointer
// beginning with the input's name; a field's line carries what its $ref
// reaches (the accessibility report's text definitions); a collection's
// items are /*; a field required under a condition is listed again with
// the condition, nested conditions joined (the DORA report's stages); and
// the JSON result carries the same lines, with no schema member.
func TestSchemaFields(t *testing.T) {
	testenv.Home(t)
	acr := filepath.Join(examplePackagesDir, "accessibility-conformance-report")
	dora := filepath.Join(examplePackagesDir, "dora-incident-report")
	site := filepath.Join(examplePackagesDir, "static-site")
	for _, c := range []struct {
		argv []string
		want []string
	}{
		{[]string{"schema", acr, "report", "--fields"}, []string{
			"/report\tobject\trequired\tadditionalProperties=false\tReport details",
			"/report/date\tstring\trequired\tformat=\"date\"\tReport date",
			`/report/product/name` + "\tstring\trequired\t" + `maxLength=200 minLength=1 pattern="^[^\\u0000-\\u001F\\u007F]+$"` + "\tProduct name",
			"/report/vendor/email\tstring\toptional\tformat=\"email\" maxLength=254\tContact email",
		}},
		{[]string{"schema", site, "articles", "--fields"}, []string{
			"/articles\tarray\trequired\tmaxItems=500 minItems=1\tArticles",
			"/articles/*/title\tstring\trequired\tmaxLength=200 minLength=1\tTitle",
			"/articles/*/tags/*\tstring\toptional\tmaxLength=40 minLength=1\t-",
		}},
		{[]string{"schema", acr, "--fields"}, []string{
			"/section508\tany\toptional\tfalse\t-\twhen \"\" does not match " +
				`{"properties":{"report":{"properties":{"edition":{"enum":["section508","international"]}},"required":["edition"]}},"required":["report"]}`,
		}},
		{[]string{"schema", dora, "--fields"}, []string{
			"/stage\tany\trequired\tenum=[\"initial notification\",\"intermediate report\",\"final report\"]\t1.1 Type of submission",
			"/incident/reputational_impact\tany\trequired\t-\t-\twhen \"\" matches " +
				`{"properties":{"stage":{"enum":["intermediate report","final report"]}},"required":["stage"]}` +
				` and /incident matches {"properties":{"classification_criteria":{"contains":{"const":"reputational impact"}}},"required":["classification_criteria"]}`,
		}},
	} {
		r := runTool(t, c.argv...)
		if r.status != 0 {
			t.Fatalf("metais %s exited %d: %s", strings.Join(c.argv, " "), r.status, r.stderr)
		}
		lines := strings.Split(strings.TrimSuffix(r.stdout, "\n"), "\n")
		for _, w := range c.want {
			if !slices.Contains(lines, w) {
				t.Errorf("metais %s lists no line\n%q\nin\n%s", strings.Join(c.argv[2:], " "), w, r.stdout)
			}
		}
		checkFieldListing(t, c.argv, r.stdout)
	}
}

// checkFieldListing holds the field listing text, which argv printed, to
// ADR-008 A7 item 6 and ADR-003 A2 item 10: a field's line has five or
// six columns and a rule's line, its second column rule, three or four;
// and the JSON result's entries, each giving its field's line and its
// rules' lines, give the text exactly.
func checkFieldListing(t *testing.T, argv []string, text string) {
	t.Helper()
	for l := range strings.SplitSeq(strings.TrimSuffix(text, "\n"), "\n") {
		n := strings.Count(l, "\t")
		if cols := strings.Split(l, "\t"); len(cols) > 1 && cols[1] == "rule" {
			if n != 2 && n != 3 {
				t.Errorf("a rule's line of %d columns: %q", n+1, l)
			}
		} else if n != 4 && n != 5 {
			t.Errorf("a line of %d columns: %q", n+1, l)
		}
	}
	var res schemaResult
	result(t, runTool(t, append([]string{"--json"}, argv...)...), &res)
	if res.Schema != nil || res.Fields == nil {
		t.Fatalf("metais %s --json: schema %d bytes, fields %v", strings.Join(argv[2:], " "), len(res.Schema), res.Fields)
	}
	var b strings.Builder
	for _, f := range *res.Fields {
		b.WriteString(fieldLine(f))
	}
	if b.String() != text {
		t.Errorf("the JSON fields give\n%s\nthe text is\n%s", b.String(), text)
	}
}

// TestSchemaFieldsLongEnum is ADR-008 A16: a field whose enum is longer
// than the listing prints either lists every value or names the command
// that does. The DORA report's root causes are a long enum, longer than a
// constraint's 1024 bytes; its line ends with the mark naming the input's
// whole schema, and that command prints every value.
func TestSchemaFieldsLongEnum(t *testing.T) {
	testenv.Home(t)
	dora := filepath.Join(examplePackagesDir, "dora-incident-report")
	r := runTool(t, "schema", dora, "incident", "--fields")
	var line string
	for l := range strings.SplitSeq(r.stdout, "\n") {
		if strings.HasPrefix(l, "/incident/root_cause_details/*\t") {
			line = l
		}
	}
	whole := runTool(t, "schema", "--json", dora, "incident")
	var res struct {
		Schema struct {
			Defs map[string]struct {
				Properties map[string]struct {
					Items struct {
						Enum []string `json:"enum"`
					} `json:"items"`
				} `json:"properties"`
			} `json:"$defs"`
		} `json:"schema"`
	}
	result(t, whole, &res)
	values := res.Schema.Defs["inputs/incident.schema.json"].Properties["root_cause_details"].Items.Enum
	if len(values) < 2 || len(line) == 0 {
		t.Fatalf("no root causes: the listing's line %q, the schema's values %v", line, values)
	}
	var cut string
	for _, v := range values {
		if !strings.Contains(line, strconv.Quote(v)) {
			cut = v
		}
	}
	if cut == "" {
		return
	}
	const mark = " bytes; metais schema <package> incident prints it whole]"
	if !strings.Contains(line, mark+"\t") {
		t.Fatalf("the listing does not show %q, and names no command that does: %q", cut, line[max(0, len(line)-120):])
	}
	text := runTool(t, "schema", dora, "incident").stdout
	for _, v := range values {
		if !strings.Contains(text, strconv.Quote(v)) {
			t.Errorf("metais schema <package> incident does not print %q", v)
		}
	}
	var listed schemaResult
	result(t, runTool(t, "--json", "schema", dora, "incident", "--fields"), &listed)
	for _, f := range *listed.Fields {
		if f.Pointer == "/incident/root_cause_details/*" {
			if c := f.Constraints[0]; !strings.HasSuffix(c, mark) {
				t.Errorf("the JSON constraint names no command: %q", c[max(0, len(c)-120):])
			}
		}
	}
	var combined json.RawMessage
	result(t, runTool(t, "--json", "schema", dora, "--fields"), &combined)
	if !strings.Contains(string(combined), " bytes; metais schema <package> prints it whole]") {
		t.Errorf("the combined listing's mark does not name the combined schema")
	}
}

// TestSchemaFieldsNext: --fields names the whole schema next, with its
// descriptions, and then validate.
func TestSchemaFieldsNext(t *testing.T) {
	testenv.Home(t)
	r := runTool(t, "schema", "--json", filepath.Join(examplePackagesDir, "static-site"), "site", "--fields")
	if len(r.doc.Next) != 2 || r.doc.Next[0].Run != "metais schema <package> site" || r.doc.Next[1].Run != "metais validate <package> --data <dir>" {
		t.Errorf("next: %v", r.doc.Next)
	}
}

// TestInspectNextNamesFields is ADR-008 A7 item 7 (scorecard candidate
// 6): inspect's next step for an input's schema runs the field listing,
// and says that it lists the input's fields.
func TestInspectNextNamesFields(t *testing.T) {
	testenv.Home(t)
	r := runTool(t, "inspect", "--json", filepath.Join(examplePackagesDir, "dora-incident-report"))
	for _, in := range []string{"stage", "entity", "incident"} {
		i := slices.IndexFunc(r.doc.Next, func(n Next) bool { return n.Run == "metais schema <package> "+in+" --fields" })
		if i < 0 {
			t.Errorf("inspect's next steps run no field listing of %s: %v", in, r.doc.Next)
			continue
		}
		if why := r.doc.Next[i].Why; !strings.HasPrefix(why, "Lists the fields of the input "+in+",") {
			t.Errorf("the step's why: %q", why)
		}
	}
}

// rulesPackage is a source directory whose input sections nests a table in
// each item, keyed on two columns, and whose input criteria must cover a
// list of values longer than a line's constraints may print; plain has no
// rule across its rows.
func rulesPackage(t *testing.T) (dir string, values []string) {
	for i := 0; len(strings.Join(values, ", ")) <= 4096; i++ {
		values = append(values, strconv.Quote(strconv.Itoa(i/100+1)+"."+strconv.Itoa(i/10%10+1)+"."+strconv.Itoa(i%10+1)))
	}
	return writeTree(t, map[string]string{
		"metais.yaml": `metais: "1.0"
name: org.example.rules
version: 1.0.0
title: Rules
description: A package whose tables carry rules.
guide: Supply the tables.
inputs:
  - name: sections
    title: Sections
    schema: inputs/sections.schema.json
  - name: criteria
    title: Criteria
    schema: inputs/criteria.schema.json
  - name: plain
    title: Plain
    schema: inputs/plain.schema.json
outputs:
  - id: report
    template: report.md.liquid
`,
		"inputs/sections.schema.json": `{"type": "array", "items": {"type": "object", "properties": {
  "rows": {"title": "Rows", "type": "array", "uniqueKeys": ["/standard", "/clause"]}}}}`,
		"inputs/criteria.schema.json": `{"type": "array", "coverage": {"key": "/criterion", "values": [` + strings.Join(values, ",") + `]}}`,
		"inputs/plain.schema.json":    `{"type": "array", "uniqueItems": true, "items": {"type": "string"}}`,
		"templates/report.md.liquid":  "report\n",
	}), values
}

// TestSchemaFieldsRules is ADR-003 A2 item 10 in metais schema --fields,
// as ruling R-W2-a places it: directly beneath the line of an array whose
// schema carries uniqueKeys or coverage, one line per rule in words, a
// composite key's pointers together, on an array nested in an item, one
// reached through $ref and one under a condition, with the condition;
// every coverage value listed where the line's constraints are cut at
// 4096 bytes; the keyword still among the constraints; and the JSON
// result's rules, with the keyword, the pointers and coverage's values.
// An input with no rule has no rule's line, and empty rules.
func TestSchemaFieldsRules(t *testing.T) {
	testenv.Home(t)
	rules, values := rulesPackage(t)
	covered := filepath.Join(vectorsDir, "dat/coverage/package")
	keyed := filepath.Join(vectorsDir, "dat/unique-keys/package")
	for _, c := range []struct {
		argv []string
		want []string // consecutive lines
	}{
		{[]string{"schema", covered, "example", "--fields"}, []string{
			"/example\tarray\trequired\tcoverage={\"key\":\"/criterion\",\"values\":[\"1.1.1\",\"1.2.1\",\"1.3.1\"]} uniqueKeys=[\"/criterion\"]\tExample",
			"/example\trule\trows must include, at `/criterion`, each of: \"1.1.1\", \"1.2.1\", \"1.3.1\"",
			"/example\trule\tno two rows share `/criterion`",
			"/example/*\tobject\toptional\t-\t-",
		}},
		{[]string{"schema", keyed, "composite", "--fields"}, []string{
			"/composite\tarray\trequired\tuniqueKeys=[\"/table\",\"/criterion\"]\tComposite",
			"/composite\trule\tno two rows share `/table` and `/criterion`",
		}},
		{[]string{"schema", covered, "referenced", "--fields"}, []string{
			"/referenced\tarray\trequired\tcoverage={\"key\":\"/id\",\"values\":[1,2]}\tReferenced",
			"/referenced\trule\trows must include, at `/id`, each of: 1, 2",
		}},
		{[]string{"schema", keyed, "conditional", "--fields"}, []string{
			"/conditional\tany\toptional\tuniqueKeys=[\"/id\"]\t-\twhen /conditional matches {\"minItems\":2}",
			"/conditional\trule\tno two rows share `/id`\twhen /conditional matches {\"minItems\":2}",
		}},
		{[]string{"schema", rules, "sections", "--fields"}, []string{
			"/sections/*/rows\tarray\toptional\tuniqueKeys=[\"/standard\",\"/clause\"]\tRows",
			"/sections/*/rows\trule\tno two rows share `/standard` and `/clause`",
		}},
		{[]string{"schema", rules, "criteria", "--fields"}, []string{
			"/criteria\trule\trows must include, at `/criterion`, each of: " + strings.Join(values, ", "),
		}},
		{[]string{"schema", rules, "--fields"}, []string{
			"/sections/*/rows\trule\tno two rows share `/standard` and `/clause`",
		}},
	} {
		r := runTool(t, c.argv...)
		if r.status != 0 {
			t.Fatalf("metais %s exited %d: %s", strings.Join(c.argv, " "), r.status, r.stderr)
		}
		if !strings.Contains("\n"+r.stdout, "\n"+strings.Join(c.want, "\n")+"\n") {
			t.Errorf("metais %s lists no lines\n%s\nin\n%s", strings.Join(c.argv[2:], " "), strings.Join(c.want, "\n"), r.stdout)
		}
		checkFieldListing(t, c.argv, r.stdout)
	}

	r := runTool(t, "schema", rules, "criteria", "--fields")
	if first, _, _ := strings.Cut(r.stdout, "\n"); !strings.HasSuffix(first, "[…+"+strconv.Itoa(len(`{"key":"/criterion","values":[`+strings.Join(values, ",")+`]}`)-1024)+" bytes; metais schema <package> criteria prints it whole]\tCriteria") {
		t.Errorf("the coverage constraint is not cut at 1024 bytes: %q", first[max(0, len(first)-120):])
	}
	var res schemaResult
	result(t, runTool(t, "--json", "schema", rules, "criteria", "--fields"), &res)
	if rs := (*res.Fields)[0].Rules; len(rs) != 1 || rs[0].Keyword != "coverage" || !slices.Equal(rs[0].Pointers, []string{"/criterion"}) || len(rs[0].Values) != len(values) {
		t.Fatalf("the criteria's rules: %+v", rs)
	}
	for i, v := range (*res.Fields)[0].Rules[0].Values {
		if string(v) != values[i] {
			t.Errorf("value %d is %s, want %s", i, v, values[i])
		}
	}
	var raw json.RawMessage
	result(t, runTool(t, "--json", "schema", rules, "sections", "--fields"), &raw)
	if want := `"rules":[{"keyword":"uniqueKeys","pointers":["/standard","/clause"]}]`; !strings.Contains(string(raw), want) {
		t.Errorf("the sections' result holds no %s: %s", want, raw)
	}

	plain := runTool(t, "schema", rules, "plain", "--fields")
	if strings.Contains(plain.stdout, "\trule\t") {
		t.Errorf("an input with no rule lists one:\n%s", plain.stdout)
	}
	result(t, runTool(t, "--json", "schema", rules, "plain", "--fields"), &res)
	for _, f := range *res.Fields {
		if f.Rules == nil || len(f.Rules) != 0 {
			t.Errorf("%s's rules are %v, want []", f.Pointer, f.Rules)
		}
	}
	checkFieldListing(t, []string{"schema", rules, "plain", "--fields"}, plain.stdout)
}
