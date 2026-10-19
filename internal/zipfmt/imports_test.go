// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package zipfmt

import (
	"go/parser"
	"go/token"
	"os"
	"strconv"
	"strings"
	"testing"
)

// releaseImports returns, for each release file of the package in dir,
// the paths it imports.
func releaseImports(t *testing.T, dir string) map[string][]string {
	t.Helper()
	ents, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	out := map[string][]string{}
	for _, e := range ents {
		name := e.Name()
		if !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		f, err := parser.ParseFile(token.NewFileSet(), dir+"/"+name, nil, parser.ImportsOnly)
		if err != nil {
			t.Fatal(err)
		}
		for _, imp := range f.Imports {
			p, _ := strconv.Unquote(imp.Path.Value)
			out[name] = append(out[name], p)
		}
	}
	return out
}

// TestNoArchiveZip holds zipfmt's release files to ADR-010 D11 item 1:
// archive/zip is not used for reading (W2-4.10). layers.json lets only
// zipfmt import archive/zip, and only its tests do, to compare the writer
// with CreateRaw.
func TestNoArchiveZip(t *testing.T) {
	files := releaseImports(t, ".")
	if len(files) == 0 {
		t.Fatal("no release files read")
	}
	for name, imps := range files {
		for _, p := range imps {
			if p == "archive/zip" {
				t.Errorf("%s imports archive/zip", name)
			}
		}
	}
	// The control: the test files do import it, and are seen to.
	f, err := parser.ParseFile(token.NewFileSet(), "write_test.go", nil, parser.ImportsOnly)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, imp := range f.Imports {
		found = found || imp.Path.Value == `"archive/zip"`
	}
	if !found {
		t.Error("the control: write_test.go's import of archive/zip is not seen")
	}
}
