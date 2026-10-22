// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"bytes"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"metais.org/metais/internal/testenv"
)

// mainShape is the whole of main, one statement a line, as ADR-010 D2's
// layout table and W1-1.2 state it: Isolate runs before anything else, and
// the arguments and streams go to cli.Main and nowhere else.
var mainShape = []string{
	"d := hostenv.Isolate()",
	"os.Exit(cli.Main(os.Args[1:], os.Stdin, os.Stdout, os.Stderr, d))",
}

// TestMainShape reads the package's release files and holds them to
// mainShape. Nothing else runs before Isolate: the package declares
// nothing but its imports and main, so it has no init function and no
// package variable whose initializer would run first.
func TestMainShape(t *testing.T) {
	if err := checkMainShape("."); err != "" {
		t.Error(err)
	}
}

// checkMainShape returns what in the package at dir departs from
// mainShape, or "" when nothing does.
func checkMainShape(dir string) string {
	names, err := filepath.Glob(filepath.Join(dir, "*.go"))
	if err != nil {
		return err.Error()
	}
	names = slices.DeleteFunc(names, func(n string) bool { return strings.HasSuffix(n, "_test.go") })
	if len(names) != 1 || filepath.Base(names[0]) != "main.go" {
		return "the package's release files are " + strings.Join(names, ", ") + ", want main.go alone"
	}
	src, err := os.ReadFile(names[0])
	if err != nil {
		return err.Error()
	}
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, names[0], src, parser.SkipObjectResolution)
	if err != nil {
		return err.Error()
	}
	var body []ast.Stmt
	for _, d := range f.Decls {
		switch d := d.(type) {
		case *ast.GenDecl:
			if d.Tok == token.IMPORT {
				continue
			}
		case *ast.FuncDecl:
			if d.Recv == nil && d.Name.Name == "main" && body == nil {
				body = d.Body.List
				continue
			}
		}
		return fset.Position(d.Pos()).String() + ": declares something beside its imports and main"
	}
	var got []string
	for _, s := range body {
		var b bytes.Buffer
		if err := format.Node(&b, fset, s); err != nil {
			return err.Error()
		}
		got = append(got, b.String())
	}
	if !slices.Equal(got, mainShape) {
		return "main is\n\t" + strings.Join(got, "\n\t") + "\nwant\n\t" + strings.Join(mainShape, "\n\t")
	}
	return ""
}

// TestMainShapeControl holds checkMainShape to finding what it exists to
// find: main.go with a statement before Isolate, with its statements
// reordered, with one after the call, with an init function or a package
// variable, and a second release file.
func TestMainShapeControl(t *testing.T) {
	const head = "package main\n\nimport (\n\t\"os\"\n\n\t\"metais.org/metais/internal/cli\"\n\t\"metais.org/metais/internal/hostenv\"\n)\n\n"
	const body = "\td := hostenv.Isolate()\n\tos.Exit(cli.Main(os.Args[1:], os.Stdin, os.Stdout, os.Stderr, d))\n"
	for name, files := range map[string]map[string]string{
		"args cut before Isolate": {"main.go": head + "func main() {\n\tos.Args = os.Args[:1]\n" + body + "}\n"},
		"reordered":               {"main.go": head + "func main() {\n\tos.Exit(cli.Main(os.Args[1:], os.Stdin, os.Stdout, os.Stderr, d))\n\td := hostenv.Isolate()\n}\n"},
		"a statement after":       {"main.go": head + "func main() {\n" + body + "\tos.Exit(0)\n}\n"},
		"other arguments":         {"main.go": head + "func main() {\n\td := hostenv.Isolate()\n\tos.Exit(cli.Main(os.Args, os.Stdin, os.Stdout, os.Stderr, d))\n}\n"},
		"an init function":        {"main.go": head + "func init() { os.Args = nil }\n\nfunc main() {\n" + body + "}\n"},
		"a package variable":      {"main.go": head + "var args = os.Args\n\nfunc main() {\n" + body + "}\n"},
		"a second file":           {"main.go": head + "func main() {\n" + body + "}\n", "init.go": "package main\n"},
	} {
		dir := testenv.Dir(t)
		for file, src := range files {
			if err := os.WriteFile(filepath.Join(dir, file), []byte(src), 0o644); err != nil {
				t.Fatal(err)
			}
		}
		if checkMainShape(dir) == "" {
			t.Errorf("%s: checkMainShape found nothing", name)
		}
	}
	dir := testenv.Dir(t)
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte(head+"func main() {\n"+body+"}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := checkMainShape(dir); err != "" {
		t.Errorf("the shape itself: %s", err)
	}
}
