// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package inputs

import (
	"strings"
	"testing"

	"github.com/kottera/metais/internal/finding"
	"github.com/kottera/metais/internal/inputs/csv"
	"github.com/kottera/metais/internal/limits"
	"github.com/kottera/metais/internal/sample"
	"github.com/kottera/metais/internal/testenv"
)

// TestSampleDataValidates is where W3-5 meets W3-2: the data the sample
// generator makes and writes as a data directory (ADR-007 D9 item 21) is
// read back through this layer and validated against the inputs' schemas
// with no error finding (item 22, "validated like supplied data"), in
// every form, collections too, under every mode.
func TestSampleDataValidates(t *testing.T) {
	testenv.Home(t)
	p := pkg{
		inputs: []map[string]any{
			slot("vendor", "inputs/vendor.json", "forms", []any{"json"}),
			slot("settings", "inputs/settings.json", "forms", []any{"yaml"}, "required", false),
			slot("lines", "inputs/lines.json", "forms", []any{"csv"}, "csv", map[string]any{"delimiter": ";"}),
			slot("summary", "inputs/summary.json", "forms", []any{"text"}),
			slot("articles", "inputs/articles.json", "forms", []any{"markdown"}, "collection", true),
			slot("events", "inputs/events.json", "forms", []any{"json", "yaml"}, "collection", true),
		},
		schemas: map[string]string{
			"inputs/vendor.json": `{"type":"object","required":["name","id"],"properties":{
				"name":{"type":"string","maxLength":40},"id":{"type":"string","format":"uuid"},
				"email":{"type":"string","format":"email"},"rating":{"type":"number","minimum":0,"maximum":5},
				"tier":{"enum":["gold","silver"]},"since":{"type":"string","format":"date"},
				"code":{"type":"string","pattern":"^[A-Z]{3}-[0-9]{4}$"},"active":{"type":"boolean","default":true},
				"kind":{"enum":["person","company"]}},
				"if":{"properties":{"kind":{"const":"company"}},"required":["kind"]},"then":{"required":["registration"],"properties":{"registration":{"type":"string","minLength":8}}}}`,
			"inputs/settings.json": `{"type":"object","properties":{"retries":{"type":"integer","minimum":1,"maximum":9},"tags":{"type":"array","items":{"type":"string"},"uniqueItems":true},"note":{"type":["string","null"]}}}`,
			"inputs/lines.json":    `{"type":"array","minItems":1,"items":{"type":"object","required":["sku","qty"],"properties":{"sku":{"type":"string","pattern":"^[a-z]{2}[0-9]{2}$"},"qty":{"type":"integer","minimum":1},"price":{"type":"number","multipleOf":0.01,"minimum":0},"gift":{"type":"boolean"},"remark":{"type":"string"}}}}`,
			"inputs/summary.json":  `{"type":"string","minLength":20}`,
			"inputs/articles.json": `{"type":"array","items":{"type":"object","required":["title"],"properties":{"title":{"type":"string"},"draft":{"type":"boolean"},"body":{"type":"string"}}}}`,
			"inputs/events.json":   `{"type":"array","maxItems":5,"items":{"type":"object","required":["at"],"properties":{"at":{"type":"string","format":"date-time"},"what":{"type":"string"}}}}`,
		},
	}
	pk := open(t, p)
	l, err := New(pk)
	if err != nil {
		t.Fatal(err)
	}
	for _, mode := range []sample.Mode{sample.All, sample.None, sample.Mixed} {
		for seed := range uint64(4) {
			g := sample.New(pk.Compiler, nil, limits.Defaults(), sample.Options{Seed: seed, Optional: mode, Items: sample.DefaultItems})
			files := map[string]string{}
			for _, in := range pk.Manifest.Inputs {
				si := sample.Input{Name: in.Name, Schema: in.Schema, Collection: in.Collection, Required: in.Required}
				for _, f := range in.Forms {
					si.Forms = append(si.Forms, sample.Form(f))
				}
				if d, ok := csv.ParseDelimiter(in.Delimiter); ok {
					si.Delimiter = d
				}
				v, present, fs, err := g.Generate(si)
				if err != nil || len(fs) > 0 {
					t.Fatalf("%s, %s, seed %d: %v%s", in.Name, mode, seed, err, describe(fs))
				}
				if !present {
					continue
				}
				written, err := g.Files(si, v)
				if err != nil {
					t.Fatalf("%s, %s, seed %d: %v", in.Name, mode, seed, err)
				}
				for _, f := range written {
					files[f.Path] = string(f.Data)
				}
			}
			res := run(t, l, limits.Defaults(), args{data: dataDir(t, files)})
			var errs []string
			for _, f := range res.Findings {
				if f.Severity() == finding.SeverityError {
					loc, _ := f.Location()
					errs = append(errs, string(f.Code())+" "+loc.File+"#"+loc.Pointer+": "+f.Message())
				}
			}
			if len(errs) > 0 {
				t.Errorf("%s, seed %d: the generated data does not validate:\n%s\nfiles %q", mode, seed, strings.Join(errs, "\n"), files)
			}
		}
	}
}
