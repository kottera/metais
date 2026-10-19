// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package xref

import (
	"strings"
	"testing"

	"github.com/kottera/metais/internal/finding"
	"github.com/kottera/metais/internal/manifest"
)

// TestBudgetConstants pins ADR-007 D3 item 13's bounds.
func TestBudgetConstants(t *testing.T) {
	if MaxContexts != 10_000 || MaxSegments != 1_000_000 {
		t.Errorf("bounds %d and %d, want 10 000 and 1 000 000", MaxContexts, MaxSegments)
	}
}

// budgetFinding returns r's template.xref.budget_exhausted, failing t
// unless there is exactly one, a warning naming bound.
func budgetFinding(t *testing.T, r Result, bound string) {
	t.Helper()
	n := 0
	for _, f := range r.Findings {
		if f.Code() != CodeBudgetExhausted {
			continue
		}
		n++
		if f.Severity() != finding.SeverityWarning {
			t.Errorf("budget_exhausted is a %s, want a warning", f.Severity())
		}
		if b, _ := f.Param("bound"); func() string { s, _ := b.Str(); return s }() != bound {
			t.Errorf("budget_exhausted names %v, want %s", b, bound)
		}
	}
	if n != 1 {
		t.Errorf("%d budget_exhausted findings, want 1: %q", n, describeAll(t, r))
	}
}

// TestBudgetContexts reaches the bound of 10 000 template-context
// analyses: a mirror of 10 001 templates. The analysis stops there with a
// warning, refuses nothing, checks nothing after it, and calls no input
// unused, since the references it did not check might use one.
func TestBudgetContexts(t *testing.T) {
	ts := map[string]string{}
	for i := range MaxContexts + 1 {
		ts["t/"+pad(i)+".liquid"] = `{{ inputs.closed.nope }}`
	}
	fx := fixture{
		inputs:    []fxInput{in("closed", `{"type": "object"}`), in("idle", `{"type": "object"}`)},
		templates: ts,
		outputs:   []manifest.Output{mirror("all", ".")},
	}
	r := fx.run(t)
	if r.Contexts != MaxContexts {
		t.Errorf("contexts = %d, want %d", r.Contexts, MaxContexts)
	}
	budgetFinding(t, r, "contexts")
	fields := 0
	for _, f := range r.Findings {
		switch f.Code() {
		case CodeUnknownField:
			fields++
		case CodeUnusedInput:
			t.Errorf("an input is called unused after the budget ran out")
		}
	}
	if fields != MaxContexts {
		t.Errorf("%d unknown_field findings, want one per template analysed, %d", fields, MaxContexts)
	}
}

// TestBudgetSegments reaches the bound of 1 000 000 resolved segments:
// one path of more segments than that, into an open schema, then a
// reference that would be an error. The analysis stops at the bound with
// a warning, and checks nothing after it.
func TestBudgetSegments(t *testing.T) {
	text := "{{ inputs.open" + strings.Repeat(".a", MaxSegments) + " }}{{ inputs.closed.nope }}"
	fx := fixture{
		inputs:    []fxInput{in("open", `true`), in("closed", `{"type": "object"}`)},
		templates: map[string]string{"big.liquid": text},
		outputs:   []manifest.Output{tpl("big", "big.liquid")},
	}
	r := fx.run(t)
	if r.Segments != MaxSegments {
		t.Errorf("segments = %d, want %d", r.Segments, MaxSegments)
	}
	budgetFinding(t, r, "segments")
	if got := codes(r); len(got) != 1 {
		t.Errorf("findings %q, want the budget's alone", got)
	}
	// Exactly MaxSegments, .open and .closed.nope among them, is checked whole.
	fx.templates["big.liquid"] = "{{ inputs.open" + strings.Repeat(".a", MaxSegments-3) + " }}{{ inputs.closed.nope }}"
	r = fx.run(t)
	if r.Segments != MaxSegments {
		t.Errorf("segments = %d, want %d", r.Segments, MaxSegments)
	}
	if got := codes(r); !equalCodes(got, []string{unknownField}) {
		t.Errorf("findings %q, want the error the last segment finds", got)
	}
}

// pad is i in five digits, so the templates sort as they are numbered.
func pad(i int) string {
	s := itoa(i)
	return strings.Repeat("0", 5-len(s)) + s
}
