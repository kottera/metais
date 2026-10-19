// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package template

import (
	"testing"

	"github.com/kottera/metais/internal/limits"
	"github.com/kottera/metais/internal/liquid"
)

// TestMeterOutputBytes is the meter's own count of output bytes (ADR-009
// D3): output-size bounds each output, counted afresh for each, and
// output-total every output together, and once the total is breached every
// later output's first byte is refused too.
func TestMeterOutputBytes(t *testing.T) {
	m := &meter{cmd: liquid.NewMeter(liquid.MeterLimits{}), outLimit: 4, totalLimit: 6}
	charge := func(n int64, want string) {
		t.Helper()
		e := m.Charge(liquid.WorkOutputBytes, n)
		got := ""
		if e != nil {
			got = e.Args["limit"]
		}
		if got != want {
			t.Errorf("charging %d (out %d, total %d) refused %q, want %q", n, m.out, m.total, got, want)
		}
	}
	charge(4, "")
	charge(1, "output-size")
	m.out = 0
	charge(2, "")
	charge(1, "output-total")
	m.out = 0
	charge(0, "output-total")
	if m.Used(liquid.WorkOutputBytes) != 6 {
		t.Errorf("Used = %d, want 6", m.Used(liquid.WorkOutputBytes))
	}
	if e := m.Charge(liquid.WorkUnits, 100); e != nil {
		t.Errorf("render-work with no limit refused: %v", e)
	}
}

// TestRenderTagsOfABrokenTemplate is renderTags over a template whose parse
// failed: the literal names of render tags before the error that stopped
// the parse, read from its constructs, and none after it.
func TestRenderTagsOfABrokenTemplate(t *testing.T) {
	src := []byte(`{% include "x" %}{% render 'a.liquid' %}{% render "b" with x %}{% render name %}{% if %}{% render "after" %}`)
	tpl, _, stop := parseEntry("t", src, limits.Defaults(), false)
	if tpl != nil {
		t.Fatal("the template parsed")
	}
	var got []string
	for _, r := range renderTags(nil, src, stop) {
		got = append(got, r.name)
	}
	if want := []string{"a.liquid", "b"}; len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Errorf("renderTags = %q, want %q", got, want)
	}
}
