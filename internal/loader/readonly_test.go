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

const (
	modulePath = "metais.org/metais/"
	hostfsPath = modulePath + "internal/hostfs"
)

// hostReach is every package through which Go code can write to the host,
// or reach a writer by other means: os and its subpackages, ioutil's
// wrappers of them, the system calls under both, and unsafe, cgo, plugin
// and the network, which can each get at the host without naming any of
// them. net's syntax packages only parse strings, and are not among them.
// Only hostfs may import one.
func hostReach(path string) bool {
	switch path {
	case "net/url", "net/netip", "net/mail":
		return false
	case "os", "io/ioutil", "syscall", "unsafe", "C", "plugin", "net", "runtime/cgo":
		return true
	}
	for _, p := range []string{"os/", "syscall/", "net/", "golang.org/x/sys/"} {
		if strings.HasPrefix(path, p) {
			return true
		}
	}
	return false
}

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
// hostfs but its read side, and to importing no package that reaches the
// host, directly or through any of this module's packages it imports:
// every host file the loader reaches, it reaches through hostfs's readers.
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
			if hostReach(path) {
				t.Errorf("%s imports %s", name, path)
			}
		}
		ast.Inspect(f, func(n ast.Node) bool {
			sel, ok := n.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			if id, ok := sel.X.(*ast.Ident); ok && local[id.Name] == hostfsPath && !slices.Contains(hostfsReads, sel.Sel.Name) {
				t.Errorf("%s: names hostfs.%s, which is not of hostfs's read side", fset.Position(sel.Pos()), sel.Sel.Name)
			}
			return true
		})
	}
}

// TestOnlyReadsTransitively: no package of this module the loader imports,
// at any depth, imports a package that reaches the host, but hostfs, whose
// use TestOnlyReads holds to its read side. A writer moved into a helper
// package is as much the loader's as one in its own files.
func TestOnlyReadsTransitively(t *testing.T) {
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	var visit func(pkg, dir, via string)
	visit = func(pkg, dir, via string) {
		if seen[pkg] {
			return
		}
		seen[pkg] = true
		files, err := filepath.Glob(filepath.Join(dir, "*.go"))
		if err != nil {
			t.Fatal(err)
		}
		if len(files) == 0 {
			t.Fatalf("%s: no Go files at %s", pkg, dir)
		}
		fset := token.NewFileSet()
		for _, name := range files {
			if strings.HasSuffix(name, "_test.go") {
				continue
			}
			// Every file, whatever its build constraint: a writer on one
			// platform is a writer.
			f, err := parser.ParseFile(fset, name, nil, parser.ImportsOnly)
			if err != nil {
				t.Fatal(err)
			}
			for _, im := range f.Imports {
				path := strings.Trim(im.Path.Value, `"`)
				switch {
				case hostReach(path):
					t.Errorf("%s (imported through %s) imports %s", pkg, via, path)
				case path == hostfsPath:
				case strings.HasPrefix(path, modulePath):
					visit(path, filepath.Join(root, strings.TrimPrefix(path, modulePath)), via+" > "+path)
				}
			}
		}
	}
	visit(modulePath+"internal/loader", ".", "loader")
	if len(seen) < 2 {
		t.Fatalf("visited only %v", seen)
	}
}
