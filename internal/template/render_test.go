// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package template

import (
	"reflect"
	"testing"

	"metais.org/metais/internal/finding"
	"metais.org/metais/internal/limits"
	"metais.org/metais/internal/liquid"
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

// TestChargeChange is ADR-010 Amendment A7 item 8: a change in an output's
// length after it rendered is charged against output-size, from the bytes
// it rendered, and against output-total with every other output's; a
// breach is the output's, at its path, a container's entry's by its
// container's path and its own; a shortening gives back what it removed,
// and a change of nothing is never refused; and once the total is
// breached every later output's first byte is refused too.
func TestChargeChange(t *testing.T) {
	lim := limits.Defaults()
	for _, arg := range []string{"output-size=10", "output-total=15"} {
		if f, ok := lim.Lower(arg); !ok {
			t.Fatalf("lowering %s: %v", arg, f)
		}
	}
	r := NewRenderer(nil, nil, lim, 0)
	rendered := func(n int64) {
		t.Helper()
		r.meter.out = 0
		if e := r.meter.Charge(liquid.WorkOutputBytes, n); e != nil {
			t.Fatalf("rendering %d bytes refused %v", n, e.Args["limit"])
		}
	}
	change := func(o Output, from, to int64, limit, file string) {
		t.Helper()
		f := r.ChargeChange(o, from, to)
		switch {
		case limit == "" && f != nil:
			t.Errorf("%s from %d to %d (total %d): %s, want no finding", o.ID, from, to, r.meter.total, f.Code())
		case limit == "":
		case f == nil:
			t.Errorf("%s from %d to %d (total %d): no finding, want %s", o.ID, from, to, r.meter.total, limit)
		default:
			loc, _ := f.Location()
			if l, _ := f.Param("limit"); f.Code() != finding.CodeLimitExceededOutput || !reflect.DeepEqual(l, finding.String(limit)) || loc.File != file {
				t.Errorf("%s from %d to %d: %s %v at %q, want limit.exceeded.output %s at %q", o.ID, from, to, f.Code(), l, loc.File, limit, file)
			}
		}
	}
	a := Output{ID: "a", Path: "a.txt"}
	rendered(8)
	change(a, 8, 10, "", "")
	change(a, 10, 11, "output-size", "a.txt")
	change(a, 10, 6, "", "")
	if got := r.meter.total; got != 6 {
		t.Errorf("after a shortening the total is %d, want 6", got)
	}
	b := Output{ID: "b", Path: "b.txt", In: "c.zip"}
	rendered(6)
	change(b, 6, 10, "output-total", "c.zip/b.txt")
	r.meter.out = 0
	if e := r.meter.Charge(liquid.WorkOutputBytes, 1); e == nil || e.Args["limit"] != "output-total" {
		t.Errorf("a later output's first byte after the total's breach: %v, want output-total", e)
	}
	// An empty output renders after the breach, and a stage that leaves
	// its length as it is changes nothing to refuse.
	change(Output{ID: "e", Path: "e.txt"}, 0, 0, "", "")
}
