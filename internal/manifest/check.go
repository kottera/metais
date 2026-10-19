// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package manifest

import (
	"github.com/kottera/metais/internal/finding"
	"github.com/kottera/metais/internal/liquid/value"
)

// Check returns the findings of the manifest rules for root that the
// manifest schema does not express (ADR-002 D10 item 6), after the version
// gate:
//
//   - Gate's refusal, alone: when the gate refuses, no other rule is
//     judged (D7 item 8, D11 step 4);
//   - manifest.license.expression, a license that is not an SPDX license
//     expression (D4 item 11): parenthesized expressions are not a regular
//     language;
//   - manifest.input.duplicate_name, two inputs with one name (ADR-003 D1
//     item 3);
//   - manifest.output.duplicate_id, two top-level declarations, or two
//     entries of one container, with one id (ADR-005 D2 item 1);
//   - manifest.output.nested_for_each, a for_each beneath a container that
//     has one (ADR-005 D3 item 6).
//
// Each rule reads only what it judges and passes over a value of the wrong
// shape, which the schema reports, so Check runs in the same pass as the
// schema's validation (D11 step 6) and reports every finding at once.
func Check(root value.Value, src Source) []finding.Finding {
	if _, refused := Gate(root, src); len(refused) > 0 {
		return refused
	}
	c := checker{src: src}
	obj, _ := root.(value.Object)
	if v, ok := obj.Get("license"); ok {
		if s, ok := v.(value.String); ok {
			if err := checkLicense(s.Text()); err != nil {
				c.add(CodeLicenseExpression, finding.Pointer("license"), finding.Params{
					"reason":   finding.String(err.reason),
					"position": finding.Int(int64(err.pos)),
				})
			}
		}
	}
	if v, ok := obj.Get("inputs"); ok {
		if a, ok := v.(value.Array); ok {
			c.unique(a, finding.Pointer("inputs"), "name", CodeInputDuplicateName)
		}
	}
	if v, ok := obj.Get("outputs"); ok {
		if a, ok := v.(value.Array); ok {
			c.outputs(a, finding.Pointer("outputs"), false)
		}
	}
	return c.findings
}

type checker struct {
	src      Source
	findings []finding.Finding
}

func (c *checker) add(code finding.Code, ptr string, p finding.Params, opts ...finding.Option) {
	c.findings = append(c.findings, c.src.finding(code, finding.SubjectManifest, ptr, p, opts...))
}

// unique reports each object of a whose string member is the same as an
// earlier object's, at the later one's member, relating the first.
func (c *checker) unique(a value.Array, ptr, member string, code finding.Code) {
	first := map[string]string{}
	for i, v := range a.All() {
		o, ok := v.(value.Object)
		if !ok {
			continue
		}
		s, ok := stringMember(o, member)
		if !ok {
			continue
		}
		at := index(ptr, i) + finding.Pointer(member)
		if earlier, dup := first[s]; dup {
			param := "name"
			if code == CodeOutputDuplicateID {
				param = "id"
			}
			c.add(code, at, finding.Params{param: finding.String(s)},
				finding.WithRelated(finding.Related{Location: *c.src.location(earlier), Note: noteFirst}))
			continue
		}
		first[s] = at
	}
}

// outputs judges the declarations of a, at ptr: top-level ones, or the
// entries of a container, which inForEach says has a for_each itself or
// beneath one.
func (c *checker) outputs(a value.Array, ptr string, inForEach bool) {
	c.unique(a, ptr, "id", CodeOutputDuplicateID)
	for i, v := range a.All() {
		o, ok := v.(value.Object)
		if !ok {
			continue
		}
		at := index(ptr, i)
		_, each := o.Get("for_each")
		if each && inForEach {
			c.add(CodeOutputNestedEach, at+finding.Pointer("for_each"), finding.Params{})
		}
		cv, ok := o.Get("container")
		if !ok {
			continue
		}
		co, ok := cv.(value.Object)
		if !ok {
			continue
		}
		if ev, ok := co.Get("entries"); ok {
			if entries, ok := ev.(value.Array); ok {
				c.outputs(entries, at+finding.Pointer("container", "entries"), inForEach || each)
			}
		}
	}
}

func stringMember(o value.Object, name string) (string, bool) {
	v, ok := o.Get(name)
	if !ok {
		return "", false
	}
	s, ok := v.(value.String)
	return s.Text(), ok
}
