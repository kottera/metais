// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package xref

import "testing"

// TestNoSchemas: when the package's schemas did not compile, every input
// is unknown, so nothing read through one is checked; every other
// reference is, and use is still found.
func TestNoSchemas(t *testing.T) {
	fx := one(objectAB, `{{ inputs.metrics.zzz[3].q }}{{ inptus.metrics }}`)
	fx.inputs = append(fx.inputs, in("spare", objectAB))
	p := fx.pkg(t)
	p.NoSchemas = true
	r, err := Analyze(p)
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}
	expect(t, r,
		`template.xref.unknown_variable templates/report.md.liquid:1:33 name="inptus" output="report" suggestion="inputs"`,
		`template.xref.unused_input metais.yaml#/inputs/1`,
	)
}
