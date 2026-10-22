// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package guarantees

import (
	"fmt"
	"go/ast"
	"go/token"
	"testing"
)

// unorderedPackage is the one package whose release files may set the
// engine's Env.UnorderedObjects: internal/template, the MapOrder seam
// (ADR-010 Amendment A2 item 4).
const unorderedPackage = "internal/template"

// unorderedViolations applies Amendment A2 item 4's rule to non-test
// files: outside internal/template, nothing sets UnorderedObjects — no
// assignment to the field, no composite literal keyed by it, no address
// taken of it. Like the fault rule it is syntactic: any selector or key of
// that name is the option, since nothing else in the module has the name.
func unorderedViolations(files []*sourceFile, module string) []string {
	var bad []string
	for _, f := range files {
		if relPackage(module, f.pkg) == unorderedPackage {
			continue
		}
		isField := func(e ast.Expr) bool {
			s, ok := ast.Unparen(e).(*ast.SelectorExpr)
			return ok && s.Sel.Name == "UnorderedObjects"
		}
		flag := func(n ast.Node, how string) {
			bad = append(bad, fmt.Sprintf("%s: %s; only internal/template sets the engine's UnorderedObjects (ADR-010 Amendment A2 item 4)", f.at(n), how))
		}
		ast.Inspect(f.file, func(n ast.Node) bool {
			switch n := n.(type) {
			case *ast.AssignStmt:
				for _, lhs := range n.Lhs {
					if isField(lhs) {
						flag(n, "assigns UnorderedObjects")
					}
				}
			case *ast.UnaryExpr:
				if n.Op == token.AND && isField(n.X) {
					flag(n, "takes the address of UnorderedObjects")
				}
			case *ast.KeyValueExpr:
				if id, ok := n.Key.(*ast.Ident); ok && id.Name == "UnorderedObjects" {
					flag(n, "sets UnorderedObjects in a composite literal")
				}
			}
			return true
		})
	}
	return bad
}

// TestUnorderedObjectsOnlyInTemplate applies the rule to the module.
func TestUnorderedObjectsOnlyInTemplate(t *testing.T) {
	for _, v := range unorderedViolations(moduleFiles(t, moduleRootAbs(t)), modulePath) {
		t.Error(v)
	}
}

func TestUnorderedObjectsRule(t *testing.T) {
	cases := []ruleCase{
		{"internal/template sets it", "internal/template", "", `package template
import "metais.org/metais/internal/liquid"
var e = liquid.Env{UnorderedObjects: true}`, nil},
		{"the engine reads it", "internal/liquid", "", `package liquid
type Env struct{ UnorderedObjects bool }
func f(e *Env) bool { return e.UnorderedObjects }`, nil},
		{"cli keys it", "internal/cli", "", `package cli
import "metais.org/metais/internal/liquid"
var e = liquid.Env{UnorderedObjects: true}`, []string{"sets UnorderedObjects in a composite literal"}},
		{"another package assigns it", "internal/author", "", `package author
import "metais.org/metais/internal/liquid"
func f(e *liquid.Env) { e.UnorderedObjects = true }`, []string{"assigns UnorderedObjects"}},
		{"the engine assigns it", "internal/liquid", "", `package liquid
type Env struct{ UnorderedObjects bool }
func f(e *Env) { e.UnorderedObjects = true }`, []string{"assigns UnorderedObjects"}},
		{"its address taken", "internal/cli", "", `package cli
import "metais.org/metais/internal/liquid"
func f(e *liquid.Env) *bool { return &e.UnorderedObjects }`, []string{"takes the address of UnorderedObjects"}},
	}
	checkRule(t, cases, func(f *sourceFile) []string {
		return unorderedViolations([]*sourceFile{f}, modulePath)
	})
}
