// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package guarantees

import (
	"fmt"
	"slices"
	"strings"
	"testing"
)

// encodingCSVImporters returns each package of the module whose release
// files import encoding/csv. A test may import it; nothing that reads an
// input may (ADR-010 D3 item 4, D8 item 1).
func encodingCSVImporters(pkgs []*listedPackage) []string {
	var bad []string
	for _, p := range pkgs {
		if p.Standard || !p.inModule(modulePath) {
			continue
		}
		if slices.Contains(p.Imports, "encoding/csv") {
			bad = append(bad, fmt.Sprintf("%s imports encoding/csv outside its tests; CSV is read by internal/inputs/csv (ADR-010 D8 item 1)", p.ImportPath))
		}
	}
	return bad
}

// TestNoEncodingCSV is ADR-010 D8 item 1 over every target of the gate.
func TestNoEncodingCSV(t *testing.T) {
	root := moduleRootAbs(t)
	for _, tg := range targets {
		for _, v := range encodingCSVImporters(cachedList(t, root, tg, "./...")) {
			t.Errorf("%s: %s", tg, v)
		}
	}
}

// TestNoEncodingCSVControl: a release import of encoding/csv is caught,
// and a test's import, the standard library's own and another module's
// are not.
func TestNoEncodingCSVControl(t *testing.T) {
	main := &listedModule{Path: modulePath, Main: true}
	pkgs := []*listedPackage{
		{ImportPath: modulePath + "/internal/inputs/csv", Module: main, Imports: []string{"encoding/csv", "strings"}},
		{ImportPath: modulePath + "/internal/conformance", Module: main, Imports: []string{"strings"}, TestImports: []string{"encoding/csv"}, XTestImports: []string{"encoding/csv"}},
		{ImportPath: "encoding/csv", Standard: true, Imports: []string{"bufio"}},
		{ImportPath: "example.com/other", Module: &listedModule{Path: "example.com/other"}, Imports: []string{"encoding/csv"}},
	}
	got := encodingCSVImporters(pkgs)
	if len(got) != 1 || !strings.HasPrefix(got[0], modulePath+"/internal/inputs/csv ") {
		t.Errorf("the encoding/csv check reported %q, want internal/inputs/csv alone", got)
	}
}
