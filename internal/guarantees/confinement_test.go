// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package guarantees

import (
	"path"
	"slices"
	"testing"

	"github.com/kottera/metais/internal/fault"
	"github.com/kottera/metais/internal/testenv"
)

// confine renders st under faults in <tmp>/parent and returns every path
// below <tmp> that changed outside the trees the step declares (ADR-010
// D13 item 4): for a refusal, every change.
func confine(t *testing.T, st renderStep, faults fault.Set) []string {
	t.Helper()
	base := testenv.Dir(t)
	work := st.prepare(t, base)
	if work == "" {
		return nil
	}
	before := renderSnapshot(t, base)
	st.run(t, work, faults)
	var bad []string
	for _, p := range changed(before, renderSnapshot(t, base)) {
		if !slices.ContainsFunc(st.trees, func(tr expectedTree) bool { return under(p, path.Join("parent", tr.Path)) }) {
			bad = append(bad, p)
		}
	}
	return bad
}

// TestConfinementDetector is ADR-010 D13 item 4's confinement guarantee
// (R12 item 1, SC6): every render of the suite runs in <tmp>/parent, and
// nothing below <tmp> changes but the trees the render declares. Its
// control, in the same run, is the SkipConfine fault over the suite's
// hostile render members, whose output paths climb out of the output
// directory: with the output path check made to return success, the
// detector must see a write outside, so a detector gone blind fails here.
func TestConfinementDetector(t *testing.T) {
	testenv.Home(t)
	var hostile []renderStep
	for _, st := range renderSteps(t) {
		if bad := confine(t, st, 0); len(bad) > 0 {
			t.Errorf("%s changed %q, outside its output", st.id, bad)
		}
		if slices.Contains(st.tags, "security") {
			hostile = append(hostile, st)
		}
	}
	if len(hostile) == 0 {
		t.Fatal("the suite holds no hostile render")
	}
	escaped := false
	for _, st := range hostile {
		escaped = escaped || len(confine(t, st, fault.SkipConfine)) > 0
	}
	if !escaped {
		t.Error("with SkipConfine, no hostile render wrote outside its output: the detector cannot tell confinement from its absence")
	}
}
