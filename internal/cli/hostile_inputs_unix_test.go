// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

//go:build unix

package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/kottera/metais/internal/finding"
	"github.com/kottera/metais/internal/testenv"
)

// hostileText is what a hostile supplier writes into a name or a value:
// a newline, a terminal escape sequence that clears the screen, one that
// sets the window title, a bell, and a bidirectional override and isolate.
const hostileText = "a\nb\x1b[2J\x1b]0;pwn\x07\u202eevil\u2066"

// hostileNeutralised is hostileText as human output prints it (ADR-009
// D15 item 3).
const hostileNeutralised = `a\u000ab\u001b[2J\u001b]0;pwn\u0007\u202eevil\u2066`

// controlFree reports whether s holds none of hostileText's controls.
func controlFree(s string) bool { return !strings.ContainsAny(s, "\x1b\x07\u202e\u2066") }

// echoPackage is a source directory whose collection's items and whose
// object input's member names are echoed by the findings about them.
func echoPackage(t *testing.T) string {
	return writeTree(t, map[string]string{
		"metais.yaml": `metais: "1.0"
name: org.example.echo
version: 1.0.0
title: Echo
description: A package whose findings echo what was supplied.
guide: Supply the inputs.
inputs:
  - name: items
    title: Items
    schema: inputs/items.schema.json
    collection: true
    required: false
  - name: closed
    title: Closed
    schema: inputs/closed.schema.json
    required: false
outputs:
  - id: report
    template: report.md.liquid
`,
		"inputs/items.schema.json":   `{"type": "array", "items": {"type": "object", "required": ["title"]}}`,
		"inputs/closed.schema.json":  `{"type": "object", "additionalProperties": false}`,
		"templates/report.md.liquid": "report\n",
	})
}

// TestHostileNamesAndValuesNeutralised is ADR-009 D17 item 9's hostile
// names and echoed values (D15 item 3; RULINGS R2-145): a collection item
// whose file name holds a newline, an escape sequence and a bidirectional
// override, and a member name holding them, are each echoed by a finding.
// Human output, of validate and of render, writes every such character as
// a \u escape; JSON carries the exact text, JSON-escaped.
func TestHostileNamesAndValuesNeutralised(t *testing.T) {
	testenv.Home(t)
	pkg := echoPackage(t)
	data := testenv.Dir(t)
	if err := os.MkdirAll(filepath.Join(data, "items"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(data, "items", hostileText+".json"), []byte(`{}`), 0o644); err != nil {
		t.Fatal(err)
	}
	member, err := json.Marshal(map[string]int{hostileText: 1})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(data, "closed.json"), member, 0o644); err != nil {
		t.Fatal(err)
	}
	for _, cmd := range [][]string{
		{"validate", pkg, "--data", data},
		{"render", pkg, "--data", data, "--out", filepath.Join(testenv.Dir(t), "out")},
	} {
		h := runTool(t, cmd...)
		if h.status != finding.ExitFindings {
			t.Errorf("%s: exit %d", cmd[0], h.status)
		}
		for _, want := range []string{
			"closed.json:1:2: error: The member `" + hostileNeutralised + "` is not one the schema allows. [input.additional_property]\n" +
				"  at /closed/" + hostileNeutralised + " (input closed: Closed)\n",
			"items/" + hostileNeutralised + ".json:1:1: error: ",
		} {
			// validate's findings are its result, on standard output;
			// render's go to standard error (ADR-008 D11 item 3).
			if !strings.Contains(h.stdout+h.stderr, want) {
				t.Errorf("%s: human output lacks %q:\n%s%s", cmd[0], want, h.stdout, h.stderr)
			}
		}
		if !controlFree(h.stdout) || !controlFree(h.stderr) {
			t.Errorf("%s: a control character reached the terminal:\n%q\n%q", cmd[0], h.stdout, h.stderr)
		}
		j := runTool(t, append([]string{cmd[0], "--json"}, cmd[1:]...)...)
		var doc struct {
			Findings []struct {
				Code     string
				Location struct{ File, Pointer string }
				Params   map[string]any
			}
		}
		if err := json.Unmarshal([]byte(j.stdout), &doc); err != nil {
			t.Fatalf("%s --json: %v", cmd[0], err)
		}
		var got []string
		for _, f := range doc.Findings {
			got = append(got, f.Code+" "+f.Location.File+" "+f.Location.Pointer)
			if p := f.Params["property"]; f.Code == "input.additional_property" && p != hostileText {
				t.Errorf("%s --json: the property is %q, want the exact text", cmd[0], p)
			}
		}
		want := []string{
			"input.additional_property closed.json /" + hostileText,
			"input.required items/" + hostileText + ".json /0/title",
		}
		if !slices.Equal(got, want) {
			t.Errorf("%s --json: findings\n%q\nwant\n%q", cmd[0], got, want)
		}
		if !strings.Contains(j.stdout, `a\u000ab\u001b[2J\u001b]0;pwn\u0007\u202eevil\u2066`) {
			t.Errorf("%s --json does not carry the text JSON-escaped: %s", cmd[0], j.stdout)
		}
	}
}
