// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package schema

import (
	"strings"
	"testing"

	"github.com/kottera/metais/internal/finding"
	"github.com/kottera/metais/internal/limits"
	"github.com/kottera/metais/internal/schemafail"
)

// target is a validation of the input vendor, for schemafail.Map.
func target(c *Compiler) schemafail.Target {
	return schemafail.Target{
		Family:  schemafail.FamilyInput,
		Subject: finding.Subject{Kind: finding.SubjectInput, Name: "vendor"},
		Schemas: c,
		Locate: func(string, bool) finding.Location {
			return finding.Location{Origin: finding.OriginDataDir, File: "vendor.json"}
		},
	}
}

// echoSentinel is every string value of the instance TestNoEchoEndToEnd
// validates.
const echoSentinel = "SENTINEL-51c2"

// TestNoEchoEndToEnd is the meeting of this item with W2-11's mapping
// (ADR-006 D13 item 5, RULINGS R1-021): failures of pattern and format,
// and of enum and an anyOf over them, produced by NewCompiler and this
// adapter and mapped by schemafail.Map, give findings in which no string
// value of the instance appears, though the library's own message for the
// same validation quotes it. The schema's pattern and format names are the
// package's, and are reported (ADR-006 D9 item 6).
func TestNoEchoEndToEnd(t *testing.T) {
	files := map[string]string{"inputs/vendor.schema.json": `{"properties": {
		"code": {"title": "Code", "pattern": "^[A-Z]{2}$"},
		"email": {"title": "Email", "format": "email"},
		"kind": {"enum": ["other", "more"]},
		"card": {"anyOf": [{"title": "Visa", "pattern": "^4"}, {"title": "Email", "format": "email"}]}}}`}
	c, fs := compileFiles(t, limits.Defaults(), files)
	if len(fs) > 0 {
		t.Fatal(describe(fs))
	}
	s, _ := c.Schema("inputs/vendor.schema.json")
	inst := mustValue(t, `{"code": "`+echoSentinel+`", "email": "`+echoSentinel+`", "kind": "`+echoSentinel+`", "card": "`+echoSentinel+`"}`)

	// The control: the library's message quotes the value, so a mapping
	// that used it would echo.
	if err := s.lib.Validate(toLibrary(inst)); err == nil || !strings.Contains(err.Error(), echoSentinel) {
		t.Fatalf("control: the library's message does not quote the value: %v", err)
	}

	fails, err := s.Validate(inst)
	if err != nil {
		t.Fatal(err)
	}
	got, err := schemafail.Map(target(c), fails...)
	if err != nil {
		t.Fatal(err)
	}
	var codes []string
	for _, f := range got {
		codes = append(codes, string(f.Code()))
		if j := string(finding.AppendJSON(nil, f)); strings.Contains(j, echoSentinel) {
			t.Errorf("a finding echoes the value: %s", j)
		}
	}
	if want := "input.any_of input.enum input.format input.pattern"; strings.Join(sorted(codes), " ") != want {
		t.Errorf("codes %q, want %s", codes, want)
	}
	for _, f := range got {
		j := string(finding.AppendJSON(nil, f))
		switch f.Code() {
		case "input.pattern":
			if !strings.Contains(j, `"^[A-Z]{2}$"`) {
				t.Errorf("input.pattern does not name the package's pattern: %s", j)
			}
		case "input.format":
			if !strings.Contains(j, `"email"`) {
				t.Errorf("input.format does not name the format: %s", j)
			}
		}
	}
}

func sorted(s []string) []string {
	out := append([]string{}, s...)
	for i := range out {
		for j := i + 1; j < len(out); j++ {
			if out[j] < out[i] {
				out[i], out[j] = out[j], out[i]
			}
		}
	}
	return out
}
