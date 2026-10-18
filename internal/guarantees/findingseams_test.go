// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package guarantees

import (
	"slices"
	"testing"
)

// findingSeams are the packages that build every finding, its encodings
// and the limits a finding reports.
var findingSeams = []string{"internal/finding", "internal/limits", "internal/schemafail", "internal/specdata"}

// TestFindingSeamsHoldNoHostFact is where the finding model meets
// TestNoHostFacts: its packages are scanned, hold no allowance, and
// break no rule. A host fact there would reach every finding, which
// ADR-006 D9 item 6 and ADR-009 D15 forbid.
func TestFindingSeamsHoldNoHostFact(t *testing.T) {
	for _, a := range moduleAllowList {
		if slices.Contains(findingSeams, a.pkg) {
			t.Errorf("%s has an allowance for %q", a.pkg, a.symbols)
		}
	}
	var files []*sourceFile
	for _, f := range moduleFiles(t, moduleRootAbs(t)) {
		if slices.Contains(findingSeams, relPackage(modulePath, f.pkg)) {
			files = append(files, f)
		}
	}
	for _, pkg := range findingSeams {
		if !slices.ContainsFunc(files, func(f *sourceFile) bool { return relPackage(modulePath, f.pkg) == pkg }) {
			t.Errorf("the symbol rule scans no file of %s", pkg)
		}
	}
	for _, v := range symbolViolations(files, modulePath) {
		t.Error(v)
	}
	for _, v := range directiveViolations(files) {
		t.Error(v)
	}
}
