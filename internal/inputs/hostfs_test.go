// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package inputs

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"slices"
	"strconv"
	"strings"
	"testing"
)

// readSide is every name of internal/hostfs the inputs layer may use: the
// read side's directory and the reasons and kinds it answers with
// (ADR-009 D7). Nothing of the write side, OpenWriter, OpenNearest,
// Writer, StagingName or TempName, is among them.
var readSide = []string{
	"OpenDir", "ReasonOf",
	"TooLarge", "Link", "NotRegularFile", "NotDirectory", "IsDirectory", "NotFound", "PermissionDenied", "IOError",
	"KindFile", "KindDir", "KindLink",
}

// TestReadsHostOnlyThroughReadSide: the data directory is read through
// hostfs's read side only (W3-2.8); internal/guarantees' layering rule
// holds every reference to hostfs to Supply.ReadDataDir, and this holds
// what that function refers to to the read side.
func TestReadsHostOnlyThroughReadSide(t *testing.T) {
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	fset := token.NewFileSet()
	var used []string
	for _, e := range entries {
		name := e.Name()
		if !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		f, err := parser.ParseFile(fset, name, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		local := ""
		for _, imp := range f.Imports {
			if p, _ := strconv.Unquote(imp.Path.Value); p == "metais.org/metais/internal/hostfs" {
				local = "hostfs"
				if imp.Name != nil {
					local = imp.Name.Name
				}
			}
		}
		if local == "" {
			continue
		}
		ast.Inspect(f, func(n ast.Node) bool {
			if sel, ok := n.(*ast.SelectorExpr); ok {
				if id, ok := sel.X.(*ast.Ident); ok && id.Name == local {
					used = append(used, sel.Sel.Name)
					if !slices.Contains(readSide, sel.Sel.Name) {
						t.Errorf("%s: refers to hostfs.%s, which is not of its read side", fset.Position(sel.Pos()), sel.Sel.Name)
					}
				}
			}
			return true
		})
	}
	if !slices.Contains(used, "OpenDir") {
		t.Errorf("the layer does not open its data directory with hostfs.OpenDir: %q", used)
	}
}
