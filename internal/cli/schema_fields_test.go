// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"path/filepath"
	"slices"
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
		for _, l := range lines {
			if n := strings.Count(l, "\t"); n != 4 && n != 5 {
				t.Errorf("a line of %d columns: %q", n+1, l)
			}
		}
		var res schemaResult
		result(t, runTool(t, append([]string{"--json"}, c.argv...)...), &res)
		if res.Schema != nil || res.Fields == nil || len(*res.Fields) != len(lines) {
			t.Fatalf("metais %s --json: schema %d bytes, fields %v", strings.Join(c.argv[2:], " "), len(res.Schema), res.Fields)
		}
		for i, f := range *res.Fields {
			if got := fieldLine(f); got != lines[i]+"\n" {
				t.Errorf("JSON field %d is %q, the text line %q", i, got, lines[i])
			}
		}
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
