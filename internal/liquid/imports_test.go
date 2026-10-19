// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package liquid

import (
	"go/build"
	"slices"
	"strings"
	"testing"

	"github.com/kottera/metais/internal/testenv"
)

// liquidPrefix is internal/liquid/..., whose packages may import each
// other and nothing else outside the standard library.
const liquidPrefix = "github.com/kottera/metais/internal/liquid/"

// foreignImports returns every import of the non-test Go files in dir, on
// every platform, that is neither a standard-library package nor a
// package of internal/liquid/.... A standard path's first element has no
// dot; every module path's does.
func foreignImports(dir string) ([]string, error) {
	ctxt := build.Default
	ctxt.UseAllFiles = true // every file, whatever its build constraints
	p, err := ctxt.ImportDir(dir, 0)
	if err != nil {
		return nil, err
	}
	var bad []string
	for _, path := range p.Imports {
		first, _, _ := strings.Cut(path, "/")
		if strings.Contains(first, ".") && !strings.HasPrefix(path, liquidPrefix) {
			bad = append(bad, path)
		}
	}
	return bad, nil
}

// TestStandardLibraryOnly pins ADR-010 D2 item 2's rule for the front end:
// internal/liquid and internal/liquid/ast import only the standard library
// and each other's packages, so no other internal package and no module.
// The layering table holds the whole of internal/liquid/... to the same
// rule over the build graph.
func TestStandardLibraryOnly(t *testing.T) {
	testenv.Home(t)
	for _, dir := range []string{".", "ast"} {
		bad, err := foreignImports(dir)
		if err != nil {
			t.Fatal(err)
		}
		if len(bad) != 0 {
			t.Errorf("internal/liquid/%s imports %q; it may import only the standard library", dir, bad)
		}
	}
}

// TestStandardLibraryOnlyControl runs the check on a fixture that imports
// an internal package and a third-party module, so the check is shown to
// flag what it exists to flag.
func TestStandardLibraryOnlyControl(t *testing.T) {
	testenv.Home(t)
	bad, err := foreignImports("value/testdata/leak")
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"github.com/kottera/metais/internal/testenv", "golang.org/x/term"}
	if !slices.Equal(bad, want) {
		t.Errorf("control fixture flagged %q, want %q", bad, want)
	}
}
