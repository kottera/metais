// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package manifest

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/kottera/metais/internal/testenv"
)

const modulePath = "github.com/kottera/metais"

// validators are the packages that validate against a JSON Schema: the
// library ADR-010 D5 admits, and the module's schema layer, where the
// closed compiler lives.
var validators = []string{"github.com/santhosh-tekuri/jsonschema", modulePath + "/internal/schema"}

// nonTestImports returns the imports of the non-test Go files in dir.
func nonTestImports(t testing.TB, dir string) []string {
	t.Helper()
	files, err := filepath.Glob(filepath.Join(dir, "*.go"))
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, f := range files {
		if strings.HasSuffix(f, "_test.go") {
			continue
		}
		src, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		af, err := parser.ParseFile(token.NewFileSet(), f, src, parser.ImportsOnly)
		if err != nil {
			t.Fatal(err)
		}
		for _, im := range af.Imports {
			p, _ := strconv.Unquote(im.Path.Value)
			out = append(out, p)
		}
	}
	return out
}

// reachesValidator reports the chain of module packages by which the
// non-test files of the package at importPath, in dir, import a validator.
func reachesValidator(t testing.TB, root, importPath string, seen map[string]bool) []string {
	t.Helper()
	if seen[importPath] {
		return nil
	}
	seen[importPath] = true
	dir := filepath.Join(root, filepath.FromSlash(strings.TrimPrefix(importPath, modulePath)))
	for _, im := range nonTestImports(t, dir) {
		if slices.ContainsFunc(validators, func(v string) bool { return im == v || strings.HasPrefix(im, v+"/") }) {
			return []string{importPath, im}
		}
		if strings.HasPrefix(im, modulePath+"/") {
			if chain := reachesValidator(t, root, im, seen); chain != nil {
				return append([]string{importPath}, chain...)
			}
		}
	}
	return nil
}

// TestNoValidator is W2-16.5: the library does not call the JSON Schema
// validator in non-test code. No non-test file of this package, nor of a
// module package it imports, imports one; the tests do, to hold the model
// to the schema.
func TestNoValidator(t *testing.T) {
	root := filepath.Join("..", "..")
	if chain := reachesValidator(t, root, modulePath+"/internal/manifest", map[string]bool{}); chain != nil {
		t.Fatalf("internal/manifest reaches a validator: %s", strings.Join(chain, " imports "))
	}
	// The control: the detector finds a validator two module packages
	// away, through a non-test file, and not through a test file.
	tmp := testenv.Dir(t)
	write := func(rel, src string) {
		p := filepath.Join(tmp, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("internal/a/a.go", "package a\nimport _ \""+modulePath+"/internal/b\"\n")
	write("internal/b/b.go", "package b\nimport _ \"github.com/santhosh-tekuri/jsonschema/v6\"\n")
	write("internal/c/c.go", "package c\n")
	write("internal/c/c_test.go", "package c\nimport _ \"github.com/santhosh-tekuri/jsonschema/v6\"\n")
	if chain := reachesValidator(t, tmp, modulePath+"/internal/a", map[string]bool{}); len(chain) != 3 {
		t.Errorf("the control: the chain through internal/b was not found: %v", chain)
	}
	if chain := reachesValidator(t, tmp, modulePath+"/internal/c", map[string]bool{}); chain != nil {
		t.Errorf("the control: a test file's import was counted: %v", chain)
	}
}
