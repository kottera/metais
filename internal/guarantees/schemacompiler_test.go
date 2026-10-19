// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package guarantees

import (
	"fmt"
	"slices"
	"strings"
	"testing"
)

// jsonschemaPath is the JSON Schema library ADR-010 D5 names.
const jsonschemaPath = "github.com/santhosh-tekuri/jsonschema/v6"

// compilerHome is the one function allowed to construct the library's
// compiler: schema.NewCompiler, which configures it closed (ADR-010 D5
// item 1).
const compilerHome = modulePath + "/internal/schema"

// compilerViolations flags every reference in files to
// jsonschema.NewCompiler but a call in schema.NewCompiler, and returns
// the number of references it allowed.
func compilerViolations(files []*sourceFile) (bad []string, allowed int) {
	for _, f := range files {
		for _, r := range f.references() {
			if r.path != jsonschemaPath || r.name != "NewCompiler" {
				continue
			}
			if f.pkg == compilerHome && r.fn == "NewCompiler" && r.call != nil {
				allowed++
				continue
			}
			bad = append(bad, fmt.Sprintf("%s: refers to jsonschema.NewCompiler outside schema.NewCompiler (ADR-010 D5 item 1)", f.at(r.sel)))
		}
	}
	return bad, allowed
}

// TestOneSchemaCompiler holds ADR-010 D5 item 1: in the module's non-test
// code, schema.NewCompiler is the only call site of jsonschema.NewCompiler,
// so every schema the tool compiles is compiled closed. Tests may use the
// library directly; the rule reads release files only.
func TestOneSchemaCompiler(t *testing.T) {
	bad, allowed := compilerViolations(moduleFiles(t, moduleRootAbs(t)))
	for _, v := range bad {
		t.Error(v)
	}
	if allowed != 1 {
		t.Errorf("found %d calls of jsonschema.NewCompiler in schema.NewCompiler, want exactly 1", allowed)
	}
}

// TestOneSchemaCompilerControl runs the rule over sources that break it,
// one way each, and over the one that keeps it, so a rule gone blind
// fails here.
func TestOneSchemaCompilerControl(t *testing.T) {
	const imp = `import "github.com/santhosh-tekuri/jsonschema/v6"` + "\n"
	cases := []struct {
		name, pkg, src string
		bad, allowed   int
	}{
		{"the home", "internal/schema", "package schema\n" + imp +
			"func NewCompiler() { jsonschema.NewCompiler() }", 0, 1},
		{"another package", "internal/other", "package other\n" + imp +
			"func NewCompiler() { jsonschema.NewCompiler() }", 1, 0},
		{"another function of the home", "internal/schema", "package schema\n" + imp +
			"func other() { jsonschema.NewCompiler() }", 1, 0},
		{"a method of the home", "internal/schema", "package schema\n" + imp +
			"type C struct{}\nfunc (C) NewCompiler() { jsonschema.NewCompiler() }", 1, 0},
		{"an aliased import", "internal/other", "package other\n" +
			`import js "github.com/santhosh-tekuri/jsonschema/v6"` + "\nfunc f() { js.NewCompiler() }", 1, 0},
		{"the function as a value", "internal/schema", "package schema\n" + imp +
			"var mk = jsonschema.NewCompiler", 1, 0},
		{"the function as a value in the home", "internal/schema", "package schema\n" + imp +
			"func NewCompiler() { f := jsonschema.NewCompiler; f() }", 1, 0},
		{"another symbol of the library", "internal/other", "package other\n" + imp +
			"var d = jsonschema.Draft2020", 0, 0},
	}
	for _, c := range cases {
		f := parseCase(t, ruleCase{name: c.name, pkg: c.pkg, src: c.src})
		bad, allowed := compilerViolations([]*sourceFile{f})
		if len(bad) != c.bad || allowed != c.allowed {
			t.Errorf("%s: flagged %q and allowed %d, want %d flagged and %d allowed", c.name, bad, allowed, c.bad, c.allowed)
		}
		if c.bad > 0 && !slices.ContainsFunc(bad, func(v string) bool { return strings.Contains(v, "ADR-010 D5 item 1") }) {
			t.Errorf("%s: the violation does not name its rule: %q", c.name, bad)
		}
	}
}
