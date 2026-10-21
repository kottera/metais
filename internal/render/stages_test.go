// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package render

import (
	"reflect"
	"testing"

	"github.com/kottera/metais/internal/finding"
	"github.com/kottera/metais/internal/limits"
	"github.com/kottera/metais/internal/loader"
	"github.com/kottera/metais/internal/template"
	"github.com/kottera/metais/internal/testenv"
)

// TestStageGrowthCharged is ADR-010 Amendment A7 items 6 and 8 on a
// container's entry, which no vector reaches: the CRLF conversion's growth
// is charged to output-size, a breach is the entry's
// limit.exceeded.output located by its container's path and its own, and
// validation, the stage after it, does not run on it. Under the default
// limits the same entry is converted and then validated, so validation
// applies to it.
func TestStageGrowthCharged(t *testing.T) {
	root := testenv.Dir(t)
	pkgDir := writeSourcePackage(t, root,
		"  - id: bundle\n    path: bundle.zip\n    container:\n      entries:\n"+
			"        - template: data.json.liquid\n          path: data.json\n          newline: crlf\n          validate: references/data.schema.json\n",
		map[string]string{
			"templates/data.json.liquid":  "{\n\"n\": \"x\"\n}\n", // 13 bytes, 16 converted
			"references/data.schema.json": `{"$schema": "https://json-schema.org/draft/2020-12/schema", "properties": {"n": {"type": "integer"}}}`,
		})
	render := func(args ...string) []finding.Finding {
		t.Helper()
		lim := limits.Defaults()
		for _, arg := range args {
			if f, ok := lim.Lower(arg); !ok {
				t.Fatalf("lowering %s: %v", arg, f)
			}
		}
		pkg, fs := loader.Open(pkgDir, loader.Options{Limits: lim}, 0)
		if pkg == nil {
			t.Fatalf("the package does not open: %v", fs)
		}
		defer pkg.Close()
		ts := template.NewTemplates(pkg.Tree, lim, pkg.Manifest.Inputs)
		b, err := template.NewBindings(template.Package{
			Tree: pkg.Tree, Manifest: pkg.Manifest, ManifestValue: pkg.ManifestValue,
			Digest: pkg.Digest, Combined: pkg.CombinedValue,
		}, nil, "", lim)
		if err != nil {
			t.Fatal(err)
		}
		return Run(Options{Package: pkg, Templates: ts, Bindings: b, Limits: lim, Fails: func(fs []finding.Finding) bool { return len(fs) > 0 }}, Emitter{}).Findings
	}

	fs := render()
	if len(fs) == 0 {
		t.Fatal("under the default limits the entry validated, want its validation's findings")
	}
	for _, f := range fs {
		if f.Code() == finding.CodeLimitExceededOutput {
			t.Errorf("under the default limits: %s", f.Code())
		}
	}

	fs = render("output-size=15")
	if len(fs) != 1 {
		t.Fatalf("growing past output-size: %d findings %v, want the limit's alone", len(fs), fs)
	}
	loc, _ := fs[0].Location()
	if l, _ := fs[0].Param("limit"); fs[0].Code() != finding.CodeLimitExceededOutput || !reflect.DeepEqual(l, finding.String("output-size")) || loc.File != "bundle.zip/data.json" {
		t.Errorf("growing past output-size: %s %v at %q, want limit.exceeded.output output-size at bundle.zip/data.json", fs[0].Code(), l, loc.File)
	}
}
