// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package value

import (
	"go/build"
	"strings"
	"testing"

	"github.com/kottera/metais/internal/testenv"
)

// nonStandardImports returns every import of the non-test Go files in dir,
// on every platform, that is not a standard-library package. A standard
// path's first element has no dot; every module path's does, this
// module's included.
func nonStandardImports(dir string) ([]string, error) {
	ctxt := build.Default
	ctxt.UseAllFiles = true // every file, whatever its build constraints
	p, err := ctxt.ImportDir(dir, 0)
	if err != nil {
		return nil, err
	}
	var bad []string
	for _, path := range p.Imports {
		first, _, _ := strings.Cut(path, "/")
		if strings.Contains(first, ".") {
			bad = append(bad, path)
		}
	}
	return bad, nil
}

// TestStandardLibraryOnly pins ADR-010 D2 item 2's rule for
// internal/liquid/...: this package imports nothing outside the standard
// library, and so no other internal package (acceptance criterion 5).
func TestStandardLibraryOnly(t *testing.T) {
	testenv.Home(t)
	bad, err := nonStandardImports(".")
	if err != nil {
		t.Fatal(err)
	}
	if len(bad) != 0 {
		t.Errorf("internal/liquid/value imports %q; it may import only the standard library", bad)
	}
}

// TestStandardLibraryOnlyControl runs the check on a fixture that imports
// an internal package and a third-party module, so the check is shown to
// flag what it exists to flag.
func TestStandardLibraryOnlyControl(t *testing.T) {
	testenv.Home(t)
	bad, err := nonStandardImports("testdata/leak")
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"github.com/kottera/metais/internal/testenv", "golang.org/x/term"}
	if strings.Join(bad, " ") != strings.Join(want, " ") {
		t.Errorf("control fixture flagged %q, want %q", bad, want)
	}
}
