// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package schema

import (
	"testing"

	"github.com/kottera/metais/internal/finding"
	"github.com/kottera/metais/internal/jsonx"
	"github.com/kottera/metais/internal/limits"
	"github.com/kottera/metais/internal/liquid/value"
	"github.com/kottera/metais/internal/pkgtree/pkgtreetest"
)

// TestWithJSONReader compiles through internal/jsonx, the reader the
// loader gives NewCompiler for a JSON schema file, and validates an
// instance it read: a number keeps its literal from the reader's text to
// the library (ADR-010 D5 item 4), and a reader's finding is reported
// with no compiler.
func TestWithJSONReader(t *testing.T) {
	lim := limits.Defaults()
	read := func(src string) (value.Value, []finding.Finding) {
		d, fs := jsonx.Parse([]byte(src), jsonx.Source{
			Input: finding.Subject{Kind: finding.SubjectInput, Name: "x"},
			At:    finding.Location{Origin: finding.OriginPackage, File: "inputs/s.json"},
		}, lim)
		return d.Value, fs
	}
	decode := func(_ string, src []byte) (value.Value, []finding.Finding) { return read(string(src)) }
	tree, err := pkgtreetest.New(map[string][]byte{"inputs/s.json": []byte(`{"maximum": 0.1}`)})
	if err != nil {
		t.Fatal(err)
	}
	c, fs := NewCompiler(tree, lim, decode)
	if len(fs) > 0 {
		t.Fatal(describe(fs))
	}
	s, _ := c.Schema("inputs/s.json")
	for inst, valid := range map[string]bool{`0.1`: true, `0.10000000000000000001`: false, `1e-1`: true} {
		v, fs := read(inst)
		if len(fs) > 0 {
			t.Fatal(describe(fs))
		}
		fails, err := s.Validate(v)
		if err != nil || (len(fails) == 0) != valid {
			t.Errorf("%s: %d failures, %v; want valid %v", inst, len(fails), err, valid)
		}
	}
	bad, err := pkgtreetest.New(map[string][]byte{"inputs/s.json": []byte(`{"a": 1, "a": 2}`)})
	if err != nil {
		t.Fatal(err)
	}
	if c, fs := NewCompiler(bad, lim, decode); c != nil || len(fs) != 1 || fs[0].Code() != "json.duplicate_name" {
		t.Errorf("a duplicate name: %v, %s", c, describe(fs))
	}
}
