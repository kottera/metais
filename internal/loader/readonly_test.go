// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package loader

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// hostfsReads is everything of internal/hostfs the loader may name: the
// read side alone (W3-1.15). hostfs's write side, its staging and its
// renames are a render's, and the loader only reads.
var hostfsReads = []string{
	"OpenFile", "OpenSource", "SourceOptions", "SourceError", "ReasonOf", "Reason",
	"IsDirectory", "NotFound", "ProblemNonRegularFile", "ProblemEmptyDirectory",
	"ProblemCompiledInSource", "ProblemMissingManifest", "ProblemNameShape",
	"ProblemLimit", "ProblemUnreadable",
}

// TestOnlyReads holds the loader's release files to naming nothing of
// hostfs but its read side, and nothing of os at all: every host file the
// loader reaches, it reaches through hostfs's readers.
func TestOnlyReads(t *testing.T) {
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	fset := token.NewFileSet()
	for _, name := range files {
		if strings.HasSuffix(name, "_test.go") {
			continue
		}
		f, err := parser.ParseFile(fset, name, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		local := map[string]string{}
		for _, im := range f.Imports {
			path := strings.Trim(im.Path.Value, `"`)
			n := filepath.Base(path)
			if im.Name != nil {
				n = im.Name.Name
			}
			local[n] = path
			if path == "os" {
				t.Errorf("%s imports os", name)
			}
		}
		ast.Inspect(f, func(n ast.Node) bool {
			sel, ok := n.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			if id, ok := sel.X.(*ast.Ident); ok && local[id.Name] == "github.com/kottera/metais/internal/hostfs" && !slices.Contains(hostfsReads, sel.Sel.Name) {
				t.Errorf("%s: names hostfs.%s, which is not of hostfs's read side", fset.Position(sel.Pos()), sel.Sel.Name)
			}
			return true
		})
	}
}
