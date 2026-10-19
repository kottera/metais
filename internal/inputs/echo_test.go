// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package inputs

import (
	"strings"
	"testing"

	"github.com/kottera/metais/internal/finding"
	"github.com/kottera/metais/internal/limits"
	"github.com/kottera/metais/internal/testenv"
)

// secret is a string value a supplier might not want repeated.
const secret = "s3cr3t-Value"

// TestNoInputValueEchoed: "No message, hint, param or related note
// contains a string value, or a part of one, taken from a supplied input"
// (ADR-006 D9 item 4). The secret is planted, as a value, in every form
// and every supply flag, each time where a finding is reported about it,
// and appears in neither encoding of any finding.
func TestNoInputValueEchoed(t *testing.T) {
	testenv.Home(t)
	l := layer(t, pkg{
		inputs: []map[string]any{
			slot("doc", "inputs/doc.json"),
			slot("rows", "inputs/rows.json", "forms", []any{"csv"}),
			slot("note", "inputs/note.json", "forms", []any{"text"}),
			slot("page", "inputs/page.json", "forms", []any{"markdown"}),
			slot("set", "inputs/doc.json"),
			slot("arg", "inputs/doc.json"),
		},
		schemas: map[string]string{
			"inputs/doc.json":  `{"type": "object", "properties": {"v": {"enum": ["a", "b"]}, "n": {"type": "integer"}}}`,
			"inputs/rows.json": `{"type": "array", "items": {"type": "object", "properties": {"v": {"type": "boolean"}, "w": {"maxLength": 1}}}}`,
			"inputs/note.json": `{"type": "string", "pattern": "^x$"}`,
			"inputs/page.json": `{"type": "object", "properties": {"body": {"const": "y"}, "t": {"format": "date"}}}`,
		},
	})
	res := run(t, l, limits.Defaults(), args{
		data: dataDir(t, map[string]string{
			"doc.json": `{"v": "` + secret + `", "n": "` + secret + `"}`,
			"rows.csv": "v,w\n" + secret + "," + secret + "\n",
			"note.txt": secret + "\n",
			"page.md":  "---\nt: " + secret + "\n---\n" + secret + "\n",
		}),
		flags: []string{
			"--set=/set/v=" + secret,
			"--set-json=/set/n=\"" + secret + "\"",
			`--inputs-json={"arg": {"v": "` + secret + `"}}`,
		},
	})
	if len(res.Findings) < 9 {
		t.Fatalf("only %d findings:\n%s", len(res.Findings), describe(res.Findings))
	}
	for _, f := range res.Findings {
		for _, text := range []string{string(finding.AppendJSON(nil, f)), string(finding.AppendHuman(nil, false, f))} {
			if strings.Contains(strings.ToLower(text), strings.ToLower(secret)) || strings.Contains(text, "s3cr3t") {
				t.Errorf("%s repeats the input's value: %s", f.Code(), text)
			}
		}
	}
}
