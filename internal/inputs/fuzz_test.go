// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package inputs

import (
	"crypto/sha256"
	"testing"

	"github.com/kottera/metais/internal/finding"
	"github.com/kottera/metais/internal/jsonx"
	"github.com/kottera/metais/internal/limits"
	"github.com/kottera/metais/internal/schema"
)

// FuzzForms reads data as a file of each form (ADR-003 D9 to D13): it
// never panics; a file gives a value or a finding, and every finding is
// located in the file; a value is one of the data model's, whose canonical
// form reads back as itself.
func FuzzForms(f *testing.F) {
	l := layer(f, formsPkg)
	lim := limits.Defaults()
	f.Fuzz(func(t *testing.T, data []byte) {
		for _, in := range l.inputs {
			loc := finding.Location{Origin: finding.OriginDataDir, File: in.Name}
			for _, form := range in.Forms {
				p := l.reader(in, lim, lim.Counter(limits.InputValues)).read(data, form, loc, "", []schema.Step{schema.Member(in.Name)})
				if p.value == nil && len(p.findings) == 0 {
					t.Fatalf("%s as %s: no value and no finding", in.Name, form)
				}
				for _, fd := range p.findings {
					if l, ok := fd.Location(); !ok || l.Origin != finding.OriginDataDir || l.File != in.Name {
						t.Fatalf("%s as %s: %s is not located in the file", in.Name, form, fd.Code())
					}
				}
				if p.value == nil {
					continue
				}
				v := p.value.value()
				c := jsonx.Canonical(v)
				back, fs := jsonx.Parse(c, jsonx.Source{Input: argSubject, At: loc}, lim)
				if len(fs) > 0 || string(jsonx.Canonical(back.Value)) != string(c) {
					t.Fatalf("%s as %s: the canonical form %q does not read back", in.Name, form, c)
				}
			}
		}
	})
}

// FuzzSupply gives an --inputs-json argument, a --set and a --set-json to
// one package and validates (ADR-003 D17, D18, D22): it never panics and
// never fails; every input with a value has its digest, and every finding
// is from the command line or a default.
func FuzzSupply(f *testing.F) {
	l := layer(f, supplyPkg)
	lim := limits.Defaults()
	f.Fuzz(func(t *testing.T, inputsJSON []byte, set, setJSON string) {
		s := l.NewSupply(lim)
		s.AddInputsJSON(inputsJSON)
		s.AddSet(set)
		s.AddSetJSON(setJSON)
		res, err := s.Validate()
		if err != nil {
			t.Fatal(err)
		}
		for _, r := range res.Inputs {
			if (r.Value != nil) != r.HasDigest {
				t.Fatalf("%s: value %v, digest %v", r.Name, r.Value != nil, r.HasDigest)
			}
			if r.Value != nil && r.Digest != sha256.Sum256(jsonx.Canonical(r.Value)) {
				t.Fatalf("%s: a digest of another value", r.Name)
			}
		}
		for _, fd := range res.Findings {
			if l, ok := fd.Location(); ok {
				switch l.Origin {
				case finding.OriginInputsJSON, finding.OriginSet, finding.OriginDefault:
				default:
					t.Fatalf("%s from %s", fd.Code(), l.Origin)
				}
			}
		}
	})
}
