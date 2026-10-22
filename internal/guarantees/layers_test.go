// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package guarantees

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/token"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"metais.org/metais/internal/testenv"
)

// layerRule is one row of layers.json, the table of ADR-010 D2 item 2's
// layering rules. Its rule field says which of the kinds below it is, and
// only that kind's fields are set.
type layerRule struct {
	Rule   string `json:"rule"`
	Reason string `json:"reason"`

	// standard-library-only: no package matching Packages depends on a
	// package outside the standard library, except one matching Packages.
	// only-referrers: only Packages refer to Symbols, in non-test files.
	Packages json.RawMessage `json:"packages,omitempty"`

	// single-entry-point: Package's non-test files refer to Imports only
	// inside its top-level function Function; or, with Through instead,
	// only to call Imports's function Through, or inside that call's
	// arguments but not inside a function literal among them, so
	// everything Package takes from Imports is for that one call.
	// test-only: no package but Package imports Package in its non-test
	// files; tests may.
	// never-depends: Package does not depend, directly or through another
	// package, on a package matching Imports.
	Package  string `json:"package,omitempty"`
	Function string `json:"function,omitempty"`
	Through  string `json:"through,omitempty"`

	// only-importers: only Importers import a package matching Imports,
	// tests included. single-entry-point and never-depends use Imports
	// too.
	Imports   string   `json:"imports,omitempty"`
	Importers []string `json:"importers,omitempty"`

	Symbols []string `json:"symbols,omitempty"`
}

// readLayers reads and checks the table at path.
func readLayers(path string) ([]layerRule, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var table struct {
		Rules []layerRule `json:"rules"`
	}
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&table); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	if len(table.Rules) == 0 {
		return nil, fmt.Errorf("%s: no rules", path)
	}
	for i, r := range table.Rules {
		if strings.TrimSpace(r.Reason) == "" {
			return nil, fmt.Errorf("%s: rule %d has no reason", path, i)
		}
		ok := false
		switch r.Rule {
		case "standard-library-only":
			var p string
			ok = json.Unmarshal(r.Packages, &p) == nil && p != "" && r.Package == "" && r.Imports == "" && r.Importers == nil && r.Symbols == nil
		case "single-entry-point":
			ok = r.Package != "" && r.Imports != "" && (r.Function == "") != (r.Through == "") && r.Packages == nil && r.Importers == nil && r.Symbols == nil
		case "test-only":
			ok = r.Package != "" && r.Packages == nil && r.Imports == "" && r.Function == "" && r.Through == "" && r.Importers == nil && r.Symbols == nil
		case "never-depends":
			ok = r.Package != "" && r.Imports != "" && r.Function == "" && r.Through == "" && r.Packages == nil && r.Importers == nil && r.Symbols == nil
		case "only-importers":
			ok = r.Imports != "" && len(r.Importers) > 0 && r.Packages == nil && r.Package == "" && r.Through == "" && r.Symbols == nil
		case "only-referrers":
			var ps []string
			ok = len(r.Symbols) > 0 && json.Unmarshal(r.Packages, &ps) == nil && len(ps) > 0 && r.Package == "" && r.Imports == "" && r.Through == "" && r.Importers == nil
		}
		if !ok {
			return nil, fmt.Errorf("%s: rule %d is not a well-formed rule of a known kind: %+v", path, i, r)
		}
	}
	return table.Rules, nil
}

// matchPattern reports whether path matches pattern, which is an import
// path or one ending /... for the path and everything below it.
func matchPattern(pattern, path string) bool {
	if base, ok := strings.CutSuffix(pattern, "/..."); ok {
		return path == base || strings.HasPrefix(path, base+"/")
	}
	return path == pattern
}

// layerViolations checks rules against one target's listing of the module
// and the parsed non-test files of its packages.
func layerViolations(rules []layerRule, pkgs []*listedPackage, files []*sourceFile) []string {
	byPath := map[string]*listedPackage{}
	for _, p := range pkgs {
		byPath[p.ImportPath] = p
	}
	var bad []string
	for _, r := range rules {
		switch r.Rule {
		case "standard-library-only":
			var pattern string
			_ = json.Unmarshal(r.Packages, &pattern)
			for _, p := range pkgs {
				if p.Standard || !matchPattern(pattern, p.ImportPath) {
					continue
				}
				for _, dep := range p.Deps {
					if d := byPath[dep]; d != nil && !d.Standard && !matchPattern(pattern, dep) {
						bad = append(bad, fmt.Sprintf("%s depends on %s, outside the standard library (%s)", p.ImportPath, dep, r.Reason))
					}
				}
			}
		case "only-importers":
			for _, p := range pkgs {
				if p.Standard || p.Module == nil || !p.Module.Main || slices.Contains(r.Importers, p.ImportPath) {
					continue
				}
				for _, imp := range slices.Concat(p.Imports, p.TestImports, p.XTestImports) {
					if matchPattern(r.Imports, imp) {
						bad = append(bad, fmt.Sprintf("%s imports %s, which only %s may (%s)", p.ImportPath, imp, strings.Join(r.Importers, ", "), r.Reason))
					}
				}
			}
		case "never-depends":
			if p := byPath[r.Package]; p != nil {
				for _, dep := range p.Deps {
					if matchPattern(r.Imports, dep) {
						bad = append(bad, fmt.Sprintf("%s depends on %s (%s)", p.ImportPath, dep, r.Reason))
					}
				}
			}
		case "test-only":
			for _, p := range pkgs {
				if p.Standard || p.Module == nil || !p.Module.Main || p.ImportPath == r.Package {
					continue
				}
				if slices.Contains(p.Imports, r.Package) {
					bad = append(bad, fmt.Sprintf("%s imports %s outside its tests (%s)", p.ImportPath, r.Package, r.Reason))
				}
			}
		case "only-referrers":
			var allowed []string
			_ = json.Unmarshal(r.Packages, &allowed)
			for _, f := range files {
				if slices.Contains(allowed, f.pkg) {
					continue
				}
				for _, ref := range f.references() {
					if slices.Contains(r.Symbols, ref.symbol()) {
						bad = append(bad, fmt.Sprintf("%s: refers to %s, which only %s may (%s)", f.at(ref.sel), ref.symbol(), strings.Join(allowed, ", "), r.Reason))
					}
				}
			}
		case "single-entry-point":
			for _, f := range files {
				if f.pkg != r.Package {
					continue
				}
				if r.Through != "" {
					bad = append(bad, throughViolations(r, f)...)
					continue
				}
				for _, ref := range f.references() {
					if ref.path == r.Imports && ref.fn != r.Function {
						where := "outside any function"
						if ref.fn != "" {
							where = "in " + ref.fn
						}
						bad = append(bad, fmt.Sprintf("%s: refers to %s %s, not in %s (%s)", f.at(ref.sel), ref.symbol(), where, r.Function, r.Reason))
					}
				}
			}
		}
	}
	return bad
}

// throughViolations is every reference f makes to r.Imports that is not
// a call of r.Through or inside such a call's arguments, outside any
// function literal there: a literal is code of its own, which the call
// may run, so a reference inside one is not for the call.
func throughViolations(r layerRule, f *sourceFile) []string {
	refs := f.references()
	type span struct{ pos, end token.Pos }
	var args, lits []span
	for _, ref := range refs {
		if ref.path != r.Imports || ref.name != r.Through || ref.call == nil {
			continue
		}
		for _, a := range ref.call.Args {
			args = append(args, span{a.Pos(), a.End()})
			ast.Inspect(a, func(n ast.Node) bool {
				if l, ok := n.(*ast.FuncLit); ok {
					lits = append(lits, span{l.Pos(), l.End()})
				}
				return true
			})
		}
	}
	within := func(spans []span, p token.Pos) bool {
		return slices.ContainsFunc(spans, func(s span) bool { return s.pos <= p && p < s.end })
	}
	var bad []string
	for _, ref := range refs {
		if ref.path != r.Imports {
			continue
		}
		if ref.name == r.Through && ref.call != nil {
			continue
		}
		if p := ref.sel.Pos(); within(args, p) && !within(lits, p) {
			continue
		}
		bad = append(bad, fmt.Sprintf("%s: refers to %s, not for a call of %s.%s (%s)", f.at(ref.sel), ref.symbol(), r.Imports, r.Through, r.Reason))
	}
	return bad
}

// moduleNonTestFiles parses the release files of the module's packages in
// one target's listing.
func moduleNonTestFiles(t testing.TB, pkgs []*listedPackage) []*sourceFile {
	t.Helper()
	var files []*sourceFile
	for _, p := range pkgs {
		if p.Standard || p.Module == nil || !p.Module.Main {
			continue
		}
		for _, name := range p.releaseFiles() {
			f, err := parseFile(p.ImportPath, p.Dir, name)
			if err != nil {
				t.Fatal(err)
			}
			files = append(files, f)
		}
	}
	return files
}

// TestLayers checks ADR-010 D2 item 2's layering rules, as layers.json
// states them, against go list -deps -json ./... for every target.
func TestLayers(t *testing.T) {
	rules, err := readLayers("layers.json")
	if err != nil {
		t.Fatal(err)
	}
	root := moduleRootAbs(t)
	for _, tg := range targets {
		pkgs := cachedList(t, root, tg, "./...")
		for _, v := range layerViolations(rules, pkgs, moduleNonTestFiles(t, pkgs)) {
			t.Errorf("%s: %s", tg, v)
		}
	}
}

// TestLayersControl runs the same table over testdata/layers, a module
// that breaks every rule once and keeps each rule's exception, so a rule
// gone blind, or one that flags what it allows, fails the build.
func TestLayersControl(t *testing.T) {
	rules, err := readLayers("layers.json")
	if err != nil {
		t.Fatal(err)
	}
	dir, err := filepath.Abs(filepath.Join("testdata", "layers"))
	if err != nil {
		t.Fatal(err)
	}
	// x/sys/unix builds for unix only, so the control reads one target.
	pkgs := goList(t, dir, target{"linux", "amd64"}, "./...")
	got := layerViolations(rules, pkgs, moduleNonTestFiles(t, pkgs))
	for i, v := range got {
		got[i], _, _ = strings.Cut(v, " (ADR-")
	}
	slices.Sort(got)
	want := []string{
		"metais.org/metais/cmd/metais depends on metais.org/metais/internal/tools/release/archive",
		"metais.org/metais/cmd/metais depends on metais.org/metais/internal/tools/ux/scenario",
		"metais.org/metais/internal/author/ooxml depends on metais.org/metais/internal/manifest",
		"metais.org/metais/internal/browse/browse.go:15: refers to metais.org/metais/internal/hostfs.Write in List, not in Extract",
		"metais.org/metais/internal/browse/browse.go:23: refers to metais.org/metais/internal/hostfs.Write in T.Extract, not in Extract",
		"metais.org/metais/internal/browse/browse.go:25: refers to metais.org/metais/internal/hostfs.Write outside any function, not in Extract",
		"metais.org/metais/internal/cli imports golang.org/x/sys/unix, which only metais.org/metais/internal/hostfs may",
		"metais.org/metais/internal/cli imports golang.org/x/term, which only metais.org/metais/internal/hostenv may",
		"metais.org/metais/internal/cli/cli.go:19: refers to os.Stdout, which only metais.org/metais/cmd/metais, metais.org/metais/internal/tools/release, metais.org/metais/internal/tools/ux may",
		"metais.org/metais/internal/liquid depends on metais.org/metais/internal/finding, outside the standard library",
		"metais.org/metais/internal/liquid/value depends on golang.org/x/sys/unix, outside the standard library",
		"metais.org/metais/internal/liquid/value depends on golang.org/x/term, outside the standard library",
		"metais.org/metais/internal/liquid/value imports golang.org/x/term, which only metais.org/metais/internal/hostenv may",
		"metais.org/metais/internal/render imports archive/zip, which only metais.org/metais/internal/zipfmt may",
		"metais.org/metais/internal/render imports go.yaml.in/yaml/v3, which only metais.org/metais/internal/yamlx may",
		"metais.org/metais/internal/render imports golang.org/x/text/cases, which only metais.org/metais/internal/unitext may",
		"metais.org/metais/internal/render imports metais.org/metais/internal/pkgtree/pkgtreetest outside its tests",
		"metais.org/metais/internal/render imports metais.org/metais/internal/xref, which only metais.org/metais/internal/author may",
		"metais.org/metais/internal/render/paths.go:18: refers to metais.org/metais/internal/hostfs.Write, not for a call of metais.org/metais/internal/hostfs.CheckPaths",
		"metais.org/metais/internal/render/paths.go:24: refers to metais.org/metais/internal/hostfs.Write, not for a call of metais.org/metais/internal/hostfs.CheckPaths",
		"metais.org/metais/internal/render/paths.go:27: refers to metais.org/metais/internal/hostfs.CheckPaths, not for a call of metais.org/metais/internal/hostfs.CheckPaths",
		"metais.org/metais/internal/tools/release/archive/archive.go:13: refers to os.Stdout, which only metais.org/metais/cmd/metais, metais.org/metais/internal/tools/release, metais.org/metais/internal/tools/ux may",
	}
	if !slices.Equal(got, want) {
		t.Errorf("the layering rules over testdata/layers reported\n\t%s\nwant\n\t%s", strings.Join(got, "\n\t"), strings.Join(want, "\n\t"))
	}
}

func TestReadLayersRefusesMalformedRules(t *testing.T) {
	for name, table := range map[string]string{
		"no rules":               `{"rules": []}`,
		"unknown kind":           `{"rules": [{"rule": "no-cycles", "reason": "r"}]}`,
		"unknown field":          `{"rules": [{"rule": "only-importers", "imports": "a", "importers": ["b"], "reason": "r", "except": "c"}]}`,
		"no reason":              `{"rules": [{"rule": "only-importers", "imports": "a", "importers": ["b"]}]}`,
		"missing field":          `{"rules": [{"rule": "single-entry-point", "package": "a", "imports": "b", "reason": "r"}]}`,
		"mixed fields":           `{"rules": [{"rule": "only-importers", "imports": "a", "importers": ["b"], "symbols": ["os.Args"], "reason": "r"}]}`,
		"wrong shape":            `{"rules": [{"rule": "standard-library-only", "packages": ["a"], "reason": "r"}]}`,
		"test-only with imports": `{"rules": [{"rule": "test-only", "package": "a", "imports": "b", "reason": "r"}]}`,
		"never-depends unnamed":  `{"rules": [{"rule": "never-depends", "imports": "b", "reason": "r"}]}`,
		"function and through":   `{"rules": [{"rule": "single-entry-point", "package": "a", "imports": "b", "function": "F", "through": "G", "reason": "r"}]}`,
		"through elsewhere":      `{"rules": [{"rule": "never-depends", "package": "a", "imports": "b", "through": "G", "reason": "r"}]}`,
	} {
		path := filepath.Join(testenv.Dir(t), "layers.json")
		if err := os.WriteFile(path, []byte(table), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := readLayers(path); err == nil {
			t.Errorf("%s: readLayers accepted %s", name, table)
		}
	}
}
