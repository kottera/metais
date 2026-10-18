// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package guarantees

import (
	"fmt"
	"go/ast"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// faultSeams is ADR-010 D13 item 2's table: each named fault and the one
// package, its seam, that consults it.
var faultSeams = map[string]string{
	"SkipConfine":  "internal/hostfs",
	"SkipDigest":   "internal/pkgtree",
	"UnpackToTemp": "internal/browse",
	"MapOrder":     "internal/liquid/value",
	"FirstFinding": "internal/finding",
}

const (
	faultPackage   = "internal/fault"
	versionPackage = "internal/version"
)

// faultViolations applies ADR-010 D13 item 2's rule to non-test files: no
// file outside internal/fault names a fault constant, except a seam naming
// its own fault, and internal/cli never assigns the options' fault field.
//
// It is syntactic: every name selected from internal/fault other than the
// type Set is a fault constant, so a fault added to the package is covered
// before anyone lists it here.
func faultViolations(files []*sourceFile, module string) []string {
	var bad []string
	for _, f := range files {
		pkg := relPackage(module, f.pkg)
		if pkg == faultPackage {
			continue
		}
		for _, ref := range f.references() {
			if ref.path != module+"/"+faultPackage || ref.name == "Set" {
				continue
			}
			if seam, ok := faultSeams[ref.name]; ok && seam == pkg {
				continue
			}
			bad = append(bad, fmt.Sprintf("%s: names fault.%s outside internal/fault and its seam (ADR-010 D13 item 2)", f.at(ref.sel), ref.name))
		}
		if pkg == "internal/cli" || strings.HasPrefix(pkg, "internal/cli/") {
			bad = append(bad, faultAssignments(f)...)
		}
	}
	return bad
}

// faultAssignments finds every way f can set the options' Faults field: an
// assignment or increment of a selector .Faults, a composite literal keyed
// Faults or an unkeyed Options literal, and taking the field's address.
func faultAssignments(f *sourceFile) []string {
	var bad []string
	isFaults := func(e ast.Expr) bool {
		s, ok := ast.Unparen(e).(*ast.SelectorExpr)
		return ok && s.Sel.Name == "Faults"
	}
	flag := func(n ast.Node, how string) {
		bad = append(bad, fmt.Sprintf("%s: %s; internal/cli never sets the fault field (ADR-010 D13 item 2)", f.at(n), how))
	}
	ast.Inspect(f.file, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.AssignStmt:
			for _, lhs := range n.Lhs {
				if isFaults(lhs) {
					flag(n, "assigns Faults")
				}
			}
		case *ast.IncDecStmt:
			if isFaults(n.X) {
				flag(n, "changes Faults")
			}
		case *ast.UnaryExpr:
			if n.Op == token.AND && isFaults(n.X) {
				flag(n, "takes the address of Faults")
			}
		case *ast.CompositeLit:
			for _, elt := range n.Elts {
				if kv, ok := elt.(*ast.KeyValueExpr); ok {
					if id, ok := kv.Key.(*ast.Ident); ok && id.Name == "Faults" {
						flag(kv, "sets Faults in a composite literal")
					}
				}
			}
			if len(n.Elts) > 0 && typeName(n.Type) == "Options" {
				if _, keyed := n.Elts[0].(*ast.KeyValueExpr); !keyed {
					flag(n, "builds Options without field names")
				}
			}
		}
		return true
	})
	return bad
}

// typeName is the name of a composite literal's type, without its package.
func typeName(e ast.Expr) string {
	switch e := e.(type) {
	case *ast.Ident:
		return e.Name
	case *ast.SelectorExpr:
		return e.Sel.Name
	}
	return ""
}

// versionViolations applies ADR-010 D12 item 3's rule: only test files call
// version.Override. A non-test file that refers to it at all, outside
// internal/version or inside it by its bare name, fails.
func versionViolations(files []*sourceFile, module string) []string {
	var bad []string
	for _, f := range files {
		if relPackage(module, f.pkg) == versionPackage {
			ast.Inspect(f.file, func(n ast.Node) bool {
				if d, ok := n.(*ast.FuncDecl); ok && d.Recv == nil && d.Name.Name == "Override" {
					ast.Inspect(d.Body, func(n ast.Node) bool {
						if id, ok := n.(*ast.Ident); ok && id.Name == "Override" {
							bad = append(bad, fmt.Sprintf("%s: refers to Override in a non-test file (ADR-010 D12 item 3)", f.at(id)))
						}
						return true
					})
					return false
				}
				if id, ok := n.(*ast.Ident); ok && id.Name == "Override" {
					bad = append(bad, fmt.Sprintf("%s: refers to Override in a non-test file (ADR-010 D12 item 3)", f.at(id)))
				}
				return true
			})
			continue
		}
		for _, ref := range f.references() {
			if ref.path == module+"/"+versionPackage && ref.name == "Override" {
				bad = append(bad, fmt.Sprintf("%s: refers to version.Override in a non-test file (ADR-010 D12 item 3)", f.at(ref.sel)))
			}
		}
	}
	return bad
}

// TestFaultsOnlyAtSeams applies the fault rule to the module.
func TestFaultsOnlyAtSeams(t *testing.T) {
	for _, v := range faultViolations(moduleFiles(t, moduleRootAbs(t)), modulePath) {
		t.Error(v)
	}
}

// TestVersionOverrideOnlyInTests applies the version rule to the module.
func TestVersionOverrideOnlyInTests(t *testing.T) {
	for _, v := range versionViolations(moduleFiles(t, moduleRootAbs(t)), modulePath) {
		t.Error(v)
	}
}

func TestFaultRule(t *testing.T) {
	cases := []ruleCase{
		{"the Set type anywhere", "internal/cli", "", `package cli
import "github.com/kottera/metais/internal/fault"
type Options struct{ Faults fault.Set }`, nil},
		{"a fault outside its seam", "internal/render", "", `package render
import "github.com/kottera/metais/internal/fault"
func f(s fault.Set) bool { return s.Has(fault.SkipConfine) }`, []string{"names fault.SkipConfine"}},
		{"a fault at its seam", "internal/hostfs", "", `package hostfs
import "github.com/kottera/metais/internal/fault"
func f(s fault.Set) bool { return s.Has(fault.SkipConfine) }`, nil},
		{"another seam's fault", "internal/hostfs", "", `package hostfs
import "github.com/kottera/metais/internal/fault"
func f(s fault.Set) bool { return s.Has(fault.SkipDigest) }`, []string{"names fault.SkipDigest"}},
		{"a fault the table does not name", "internal/finding", "", `package finding
import "github.com/kottera/metais/internal/fault"
var all = fault.All`, []string{"names fault.All"}},
		{"an aliased fault import", "internal/render", "", `package render
import f "github.com/kottera/metais/internal/fault"
var x = f.MapOrder`, []string{"names fault.MapOrder"}},
		{"internal/fault itself", "internal/fault", "", `package fault
const SkipConfine Set = 1`, nil},
		{"cli builds zero options", "internal/cli", "", `package cli
func f() Options { return Options{Display: d} }`, nil},
		{"cli assigns the field", "internal/cli", "", `package cli
func f(o *Options) { o.Faults = 1 }`, []string{"assigns Faults"}},
		{"cli ors into the field", "internal/cli", "", `package cli
func f(inv Invocation) { inv.Options.Faults |= 1 }`, []string{"assigns Faults"}},
		{"cli increments the field", "internal/cli", "", `package cli
func f(o *Options) { o.Faults++ }`, []string{"changes Faults"}},
		{"cli takes the field's address", "internal/cli", "", `package cli
func f(o *Options) { set(&o.Faults) }`, []string{"takes the address of Faults"}},
		{"cli keys the field", "internal/cli", "", `package cli
var o = Options{Faults: 1}`, []string{"sets Faults in a composite literal"}},
		{"cli builds options by position", "internal/cli", "", `package cli
var o = Options{d, 1}`, []string{"builds Options without field names"}},
		{"cli's subpackage assigns the field", "internal/cli/args", "", `package args
import "github.com/kottera/metais/internal/cli"
var o = cli.Options{Faults: 1}`, []string{"sets Faults"}},
		{"cli reads the field", "internal/cli", "", `package cli
func f(o Options) bool { return o.Faults != 0 }`, nil},
	}
	checkRule(t, cases, func(f *sourceFile) []string {
		return faultViolations([]*sourceFile{f}, modulePath)
	})
}

func TestVersionRule(t *testing.T) {
	cases := []ruleCase{
		{"a call of version.String", "internal/cli", "", `package cli
import "github.com/kottera/metais/internal/version"
var v = version.String()`, nil},
		{"a call of version.Override", "internal/cli", "", `package cli
import "github.com/kottera/metais/internal/version"
func f() { version.Override("1.2.3") }`, []string{"refers to version.Override"}},
		{"version.Override as a value", "internal/render", "", `package render
import v "github.com/kottera/metais/internal/version"
var set = v.Override`, []string{"refers to version.Override"}},
		{"Override's own declaration", "internal/version", "", `package version
var override string
func Override(v string) func() { old := override; override = v; return func() { override = old } }`, nil},
		{"Override called inside its package", "internal/version", "", `package version
func Override(string) {}
func init() { Override("1.2.3") }`, []string{"refers to Override"}},
		{"Override calling itself", "internal/version", "", `package version
func Override(v string) { if v == "" { Override("0.0.0") } }`, []string{"refers to Override"}},
	}
	checkRule(t, cases, func(f *sourceFile) []string {
		return versionViolations([]*sourceFile{f}, modulePath)
	})
}

// TestVersionRuleMeetsVersionPackage is where the version rule meets the
// real internal/version: the package declares the Override the rule names,
// so a rename cannot leave the rule looking for a name nothing has, and a
// call added to the package's own file or to a real caller is flagged.
func TestVersionRuleMeetsVersionPackage(t *testing.T) {
	root := moduleRootAbs(t)
	var versionFile, caller *sourceFile
	for _, f := range moduleFiles(t, root) {
		switch relPackage(modulePath, f.pkg) + "/" + f.name {
		case versionPackage + "/version.go":
			versionFile = f
		case "internal/cli/cli.go":
			caller = f
		}
	}
	if versionFile == nil || caller == nil {
		t.Fatal("internal/version/version.go or internal/cli/cli.go is not among the module's release files")
	}
	declared := false
	for _, d := range versionFile.file.Decls {
		if fn, ok := d.(*ast.FuncDecl); ok && fn.Recv == nil && fn.Name.Name == "Override" {
			declared = true
		}
	}
	if !declared {
		t.Fatal("internal/version declares no top-level func Override, which the version rule names")
	}

	src := func(f *sourceFile) []byte {
		b, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(relPackage(modulePath, f.pkg)), f.name))
		if err != nil {
			t.Fatal(err)
		}
		return b
	}
	inPackage, err := parseSource(versionFile.pkg, versionFile.name,
		append(src(versionFile), "\nfunc init() { Override(nil, \"9.9.9\") }\n"...))
	if err != nil {
		t.Fatal(err)
	}
	callerSrc := strings.Replace(string(src(caller)), "import (", "import (\n\t\"github.com/kottera/metais/internal/version\"", 1)
	fromCaller, err := parseSource(caller.pkg, caller.name, []byte(callerSrc+"\nvar _ = version.Override\n"))
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range []*sourceFile{inPackage, fromCaller} {
		if got := versionViolations([]*sourceFile{f}, modulePath); len(got) != 1 {
			t.Errorf("%s/%s with a call of Override added: flagged %q, want one violation", f.pkg, f.name, got)
		}
	}
}
