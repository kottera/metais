// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package inputs

import (
	"testing"

	"metais.org/metais/internal/finding"
	"metais.org/metais/internal/limits"
	"metais.org/metais/internal/testenv"
)

// sc8Pkg is shaped as SC8's package: an input in each form, a collection,
// and input constraints over them.
var sc8Pkg = pkg{
	inputs: []map[string]any{
		slot("product", "inputs/product.json"),
		slot("vendor", "inputs/object.json"),
		slot("criteria", "inputs/criteria.json", "forms", []any{"csv"}),
		slot("articles", "inputs/articles.json", "collection", true, "forms", []any{"markdown"}),
		slot("summary", "inputs/summary.json", "forms", []any{"text"}),
		slot("owner", "inputs/object.json"),
		slot("edition", "inputs/edition.json"),
	},
	constraints: "inputs/constraints.json",
	schemas: map[string]string{
		"inputs/product.json": `{"type": "object", "properties": {"name": {"type": "string", "minLength": 2}}}`,
		"inputs/object.json":  `{"type": "object"}`,
		"inputs/criteria.json": `{"type": "array", "items": {"type": "object",
			"properties": {"id": {"type": "string"}, "level": {"enum": ["A", "AA"]}, "remarks": {"type": "string"}}}}`,
		"inputs/articles.json":    `{"type": "array", "items": {"type": "object", "required": ["title"]}}`,
		"inputs/summary.json":     `{"type": "string", "maxLength": 20}`,
		"inputs/edition.json":     `{"type": "string", "enum": ["2.1", "2.2"]}`,
		"inputs/constraints.json": `{"if": {"properties": {"edition": {"const": "2.2"}}}, "then": {"required": ["criteria"]}}`,
	},
}

// TestEveryPlantedError: the inputs layer reports every planted error of
// an SC8-shaped fixture in one call (plan §9's acceptance): parse findings
// across several files, every form's, together with the schema findings
// of what did parse, a collection and a CSV file each validated in part,
// the supply's own refusals, a required input missing, and the info that
// the constraints were not evaluated (ADR-003 D22, ADR-006 D7).
func TestEveryPlantedError(t *testing.T) {
	testenv.Home(t)
	l := layer(t, sc8Pkg)
	res := run(t, l, limits.Defaults(), args{
		data: dataDir(t, map[string]string{
			"product.json":    "{\n  \"name\": \"Widget\",\n}\n",
			"vendor.yaml":     "name: A\nname: B\n",
			"criteria.csv":    "id,level,remarks\n1.1.1,A,\n1.2.1,AA\n1.3.1,AAA,x\n",
			"articles/one.md": "---\ntitle: One\n",
			"articles/two.md": "---\nauthor: X\n---\nText\n",
			"summary.txt":     "A summary well over twenty characters.\n",
			"extra.json":      "{}",
		}),
		flags: []string{
			`--inputs-json={"edition": 2.2}`,
			"--set=/criteria/0/remarks=x",
			"--set=/edition/x=1",
		},
	})
	wantFindings(t, res.Findings,
		`json.syntax product data_dir:product.json:3:1 # {"expected":"member name","found":"unexpected character"}`,
		`yaml.duplicate_key vendor data_dir:vendor.yaml:2:1 #/name`,
		`csv.field_count criteria data_dir:criteria.csv:3:1 #/1 {"expected":3,"found":2}`,
		`input.enum criteria data_dir:criteria.csv:4:7 #/2/level {"allowed":["\"A\"","\"AA\""],"allowed_count":2,"suggestion":"AA"}`,
		`input.validation.partial criteria data_dir:criteria.csv # {"unparsed":1,"suppressed":0}`,
		`markdown.frontmatter_unclosed articles data_dir:articles/one.md:1:1`,
		`input.required articles data_dir:articles/two.md:1:1 #/1/title {"property":"title"}`,
		`input.validation.partial articles data_dir:articles # {"unparsed":1,"suppressed":0}`,
		`input.max_length summary data_dir:summary.txt # {"limit":20,"length":38}`,
		`input.file_unknown extra data_dir:extra.json`,
		`input.type edition inputs_json:1:1:13 # {"expected":["string"],"actual":"number"}`,
		`input.set_into_unparsed criteria set:1`,
		`input.set_traversal edition set:2 {"reason":"scalar"}`,
		`input.missing owner`,
		`input.constraints_not_evaluated`,
	)
	// Every finding, ordered as a report orders them, is one of a kind a
	// supplier fixes in one round: none is lost to the cap or a duplicate.
	r := finding.NewReport(int(limits.Defaults().Value(limits.Findings)))
	for _, f := range res.Findings {
		r.Add(f)
	}
	if s := r.Summary(); s.Listed != len(res.Findings) || s.Omitted != 0 || s.Errors != 12 || s.Info != 3 {
		t.Errorf("summary %+v", s)
	}
}
