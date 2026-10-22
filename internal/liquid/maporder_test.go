// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package liquid

import (
	"fmt"
	goast "go/ast"
	"go/build"
	"go/importer"
	goparser "go/parser"
	"go/token"
	"go/types"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"metais.org/metais/internal/testenv"
)

// liquidPath is internal/liquid's import path.
const liquidPath = "metais.org/metais/internal/liquid"

// typeChecker type-checks a directory of internal/liquid/... from source,
// with the standard library type-checked from GOROOT's source, so the
// check needs neither the go command nor export data.
type typeChecker struct {
	fset *token.FileSet
	std  types.Importer
	pkgs map[string]*types.Package
}

func newTypeChecker() *typeChecker {
	fset := token.NewFileSet()
	return &typeChecker{fset: fset, std: importer.ForCompiler(fset, "source", nil), pkgs: map[string]*types.Package{}}
}

func (c *typeChecker) Import(path string) (*types.Package, error) {
	if path == liquidPath || strings.HasPrefix(path, liquidPrefix) {
		if p, ok := c.pkgs[path]; ok {
			return p, nil
		}
		p, _, _, err := c.check(filepath.FromSlash("."+strings.TrimPrefix(path, liquidPath)), path)
		return p, err
	}
	return c.std.Import(path)
}

// check type-checks the non-test Go files of dir as the package path.
func (c *typeChecker) check(dir, path string) (*types.Package, *types.Info, []*goast.File, error) {
	bp, err := build.Default.ImportDir(dir, 0)
	if err != nil {
		return nil, nil, nil, err
	}
	var files []*goast.File
	for _, name := range bp.GoFiles {
		f, err := goparser.ParseFile(c.fset, filepath.Join(dir, name), nil, 0)
		if err != nil {
			return nil, nil, nil, err
		}
		files = append(files, f)
	}
	info := &types.Info{Types: map[goast.Expr]types.TypeAndValue{}, Uses: map[*goast.Ident]types.Object{}}
	conf := types.Config{Importer: c}
	p, err := conf.Check(path, c.fset, files, info)
	if err != nil {
		return nil, nil, nil, err
	}
	c.pkgs[path] = p
	return p, info, files, nil
}

// allowedMapIteration is the one function of the engine that iterates a Go
// map: the order Env.UnorderedObjects asks for (ADR-010 Amendment A2).
const allowedMapIteration = "unorderedMembers"

// mapIterations returns where the non-test Go files of dir iterate a Go
// map: a range over a map, or a call of maps.Keys, maps.Values or
// maps.All whose result is not handed straight to slices.Sorted or
// slices.SortedFunc, which make the order the map's iteration had
// irrelevant. A function named allowed is not looked in.
func mapIterations(c *typeChecker, dir, path, allowed string) ([]string, error) {
	_, info, files, err := c.check(dir, path)
	if err != nil {
		return nil, err
	}
	var found []string
	for _, f := range files {
		sorted := map[goast.Expr]bool{}
		goast.Inspect(f, func(n goast.Node) bool {
			switch n := n.(type) {
			case *goast.FuncDecl:
				return n.Recv != nil || n.Name.Name != allowed
			case *goast.RangeStmt:
				if _, ok := info.Types[n.X].Type.Underlying().(*types.Map); ok {
					found = append(found, fmt.Sprintf("%s: range over a map", c.fset.Position(n.Pos())))
				}
			case *goast.CallExpr:
				switch pkgFunc(info, n) {
				case "slices.Sorted", "slices.SortedFunc":
					sorted[n.Args[0]] = true
				case "maps.Keys", "maps.Values", "maps.All":
					if !sorted[n] {
						found = append(found, fmt.Sprintf("%s: %s iterates a map", c.fset.Position(n.Pos()), pkgFunc(info, n)))
					}
				}
			}
			return true
		})
	}
	return found, nil
}

// pkgFunc names the package-level function a call calls, as "pkg.Name",
// or "" for any other call.
func pkgFunc(info *types.Info, call *goast.CallExpr) string {
	fun := call.Fun
	if ix, ok := fun.(*goast.IndexExpr); ok {
		fun = ix.X
	}
	sel, ok := fun.(*goast.SelectorExpr)
	if !ok {
		return ""
	}
	fn, ok := info.Uses[sel.Sel].(*types.Func)
	if !ok || fn.Pkg() == nil {
		return ""
	}
	return fn.Pkg().Path() + "." + fn.Name()
}

// TestNoMapIteration pins ADR-010 D4 item 2 as Amendment A2 states it:
// no Go map is iterated by the engine. Objects iterate their ordered
// members, so the same template and data render the same bytes on every
// run (ADR-004 D4 item 5); the one exception A2 admits is the order
// Env.UnorderedObjects asks for, in allowedMapIteration.
func TestNoMapIteration(t *testing.T) {
	testenv.Home(t)
	c := newTypeChecker()
	for _, dir := range []string{".", "ast", "value"} {
		path := liquidPath
		if dir != "." {
			path += "/" + dir
		}
		found, err := mapIterations(c, dir, path, allowedMapIteration)
		if err != nil {
			t.Fatal(err)
		}
		for _, f := range found {
			t.Errorf("%s", f)
		}
	}
}

// TestNoMapIterationControl runs the check on a fixture that iterates
// maps in each way it refuses, so the check is shown to see them.
func TestNoMapIterationControl(t *testing.T) {
	testenv.Home(t)
	found, err := mapIterations(newTypeChecker(), filepath.Join("testdata", "maporder"), "leak", allowedMapIteration)
	if err != nil {
		t.Fatal(err)
	}
	var kinds []string
	for _, f := range found {
		_, k, _ := strings.Cut(f, ": ")
		kinds = append(kinds, k)
	}
	want := []string{"range over a map", "range over a map", "maps.Keys iterates a map", "range over a map"}
	if !slices.Equal(kinds, want) {
		t.Errorf("the control fixture gave %q, want %q", found, want)
	}
}

// TestUnorderedObjects is Env.UnorderedObjects (ADR-010 Amendment A2 item
// 2): without it, a for over an object visits its members in canonical
// order on every render; with it, the order follows a Go map built from
// them, which twenty renders of a 20-member object do not all repeat, and
// each render still visits every member once.
func TestUnorderedObjects(t *testing.T) {
	var j, want strings.Builder
	j.WriteString("{\"o\": {")
	for i := 20; i >= 1; i-- {
		fmt.Fprintf(&j, "%q: %d", fmt.Sprintf("m%02d", i), i)
		if i > 1 {
			j.WriteString(", ")
		}
	}
	j.WriteString("}}")
	for i := 1; i <= 20; i++ {
		fmt.Fprintf(&want, "m%02d;", i)
	}
	const src = "{% for m in o %}{{ m[0] }};{% endfor %}"
	for range 20 {
		renders(t, src, want.String(), data(j.String()))
	}
	unordered := func(_ testing.TB, e *Env) { e.UnorderedObjects = true }
	differs := false
	for range 20 {
		got, err := renderText(t, src, data(j.String()), unordered)
		if err != nil {
			t.Fatal(err)
		}
		parts := strings.Split(strings.TrimSuffix(got, ";"), ";")
		slices.Sort(parts)
		if strings.Join(parts, ";")+";" != want.String() {
			t.Fatalf("an unordered render visits %q", got)
		}
		differs = differs || got != want.String()
	}
	if !differs {
		t.Error("twenty unordered renders all visited canonical order")
	}
}
