// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package liquid

import (
	"slices"
	"testing"
)

// goldenFilters are the filters every Golden Liquid run adds to Core():
// date, which internal/dates defines and internal/template registers
// (plan §8). This package cannot import internal/dates, which imports it,
// so golden_date_test.go, a file of the external test package linked into
// the same test binary, registers it through RegisterGoldenFilter before
// any test runs.
var goldenFilters []FilterDef

// RegisterGoldenFilter adds f to the filters every Golden Liquid run
// adds to Core(). It is for golden_date_test.go alone.
func RegisterGoldenFilter(f FilterDef) { goldenFilters = append(goldenFilters, f) }

// goldenCore is the dialect every Golden Liquid run renders in: Core()
// with date. A run without date would refuse its cases as unknown, so a
// binary in which date was not registered fails here rather than passing
// with the cases unrun.
func goldenCore(t testing.TB) *Dialect {
	t.Helper()
	d, err := Core().Extend(Spec{Filters: goldenFilters})
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(d.FilterNames(), "date") {
		t.Fatal("date is not registered for the Golden Liquid runs: golden_date_test.go is not in the test binary")
	}
	return d
}
