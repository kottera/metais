// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/kottera/metais/internal/testenv"
)

// parseCodes are the codes a template's parse reports (ADR-004 D16 item
// 1), and limit.package for its template-size and template-nesting.
// template.embed_unknown_input is not among them: an embed naming no
// declared input is check's cross-reference's template.xref.unknown_input
// (ADR-007 D3 item 12).
var parseCodes = []string{
	"template.syntax", "template.not_utf8", "template.include", "template.unknown_filter",
	"template.reserved_name", "template.partial_name", "template.number_range",
	"template.filter_argument", "template.filter_input", "template.division_by_zero", "limit.package",
	"template.embed_index", "template.provenance_argument", "template.provenance_style",
}

// parseFindings returns the findings of a command's JSON document that a
// template's parse reports, each as its JSON text, by the template's
// entry.
func parseFindings(t *testing.T, r ran) map[string][]string {
	t.Helper()
	var doc struct {
		Findings []json.RawMessage `json:"findings"`
	}
	if err := json.Unmarshal([]byte(r.stdout), &doc); err != nil {
		t.Fatal(err)
	}
	out := map[string][]string{}
	for _, raw := range doc.Findings {
		var f struct {
			Code     string `json:"code"`
			Location struct {
				File string `json:"file"`
			} `json:"location"`
			Params struct {
				OutputPath *string `json:"output_path"`
			} `json:"params"`
		}
		if err := json.Unmarshal(raw, &f); err != nil {
			t.Fatal(err)
		}
		// A finding naming the output being rendered was met rendering it.
		if slices.Contains(parseCodes, f.Code) && strings.HasPrefix(f.Location.File, "templates/") && f.Params.OutputPath == nil {
			// The message and hint are the same template's text; what
			// is compared is every member.
			out[f.Location.File] = append(out[f.Location.File], string(raw))
		}
	}
	return out
}

// TestCheckAndRenderParseAlike is W4-3.12, the meeting of check (W4-4)
// and render (W4-3): both parse a template with the one parse,
// internal/template's ParseEntry (ADR-007 D2 step 6), so for every
// template render parses, check reports the very findings render does.
// The packages are the suite's that hold templates that do not parse.
func TestCheckAndRenderParseAlike(t *testing.T) {
	testenv.Home(t)
	compared := 0
	for _, v := range []string{"tpl/parse-refusals", "aut/check-parse-hints", "aut/check-templates", "tag/embed-refusals", "tag/provenance-refusals"} {
		pkg := filepath.Join(vectorsDir, filepath.FromSlash(v), "package")
		argv := []string{"render", "--json", pkg, "--out", filepath.Join(testenv.Dir(t), "out")}
		// A render parses only once its inputs are supplied.
		if data := filepath.Join(vectorsDir, filepath.FromSlash(v), "data"); isDir(data) {
			argv = append(argv, "--data", data)
		}
		render := parseFindings(t, runTool(t, argv...))
		check := parseFindings(t, runTool(t, "check", "--json", pkg))
		for file, fs := range render {
			compared++
			if !slices.Equal(fs, check[file]) {
				t.Errorf("%s: %s parses differently:\n render %s\n check  %s", v, file, fs, check[file])
			}
		}
	}
	if compared == 0 {
		t.Fatal("render reported no template's parse findings")
	}
}

// isDir reports whether p is a directory.
func isDir(p string) bool {
	info, err := os.Stat(p)
	return err == nil && info.IsDir()
}
