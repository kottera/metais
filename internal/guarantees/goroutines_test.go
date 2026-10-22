// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package guarantees

import (
	"go/ast"
	"strings"
	"testing"
)

// The render's one goroutine is the assembled producer's, which writes a
// container into a pipe as it is read (ADR-010 A6 item 1); every other
// path of the render is single-threaded (D2 item 3).
const (
	renderPackages = "metais.org/metais/internal/render/..."
	goroutinePkg   = "metais.org/metais/internal/render"
	goroutineFile  = "producer_assembled.go"
)

// renderGoStatements checks that files of renderPackages hold exactly one
// go statement, in goroutinePkg's goroutineFile, and returns what is
// wrong: each go statement elsewhere, each beyond the first there, or the
// absence of the one.
func renderGoStatements(files []*sourceFile) []string {
	var bad []string
	found := 0
	for _, f := range files {
		if !matchPattern(renderPackages, f.pkg) {
			continue
		}
		ast.Inspect(f.file, func(n ast.Node) bool {
			if g, ok := n.(*ast.GoStmt); ok {
				if f.pkg == goroutinePkg && f.name == goroutineFile && found == 0 {
					found++
				} else {
					bad = append(bad, f.at(g)+": a go statement beyond the assembled producer's one")
				}
			}
			return true
		})
	}
	if found == 0 {
		bad = append(bad, goroutinePkg+"/"+goroutineFile+": no go statement, where the assembled producer's one should be")
	}
	return bad
}

// TestRenderOneGoStatement holds internal/render and its subpackages to
// one go statement, in the assembled producer's file (ADR-010 A7 item 11,
// D2 item 2). It reads the release files of every target, so a file a
// build constraint keeps from the host's build cannot hide one.
func TestRenderOneGoStatement(t *testing.T) {
	for _, v := range renderGoStatements(moduleFiles(t, moduleRootAbs(t))) {
		t.Error(v)
	}
}

// TestRenderOneGoStatementControl feeds the check go statements where they
// may and may not be, so a check gone blind, or one that flags the one
// goroutine, fails the build.
func TestRenderOneGoStatementControl(t *testing.T) {
	const one = "package render\n\nfunc open() { go func() {}() }\n"
	parse := func(pkg, name, src string) *sourceFile {
		f, err := parseSource(pkg, name, []byte(src))
		if err != nil {
			t.Fatal(err)
		}
		return f
	}
	for _, c := range []struct {
		name  string
		files []*sourceFile
		want  []string
	}{
		{"the one", []*sourceFile{parse(goroutinePkg, goroutineFile, one)}, nil},
		{"outside render", []*sourceFile{
			parse(goroutinePkg, goroutineFile, one),
			parse("metais.org/metais/internal/cli", "run.go", "package cli\n\nfunc run() { go run() }\n"),
		}, nil},
		{"none", []*sourceFile{parse(goroutinePkg, "plan.go", "package render\n")}, []string{
			"metais.org/metais/internal/render/producer_assembled.go: no go statement, where the assembled producer's one should be",
		}},
		{"a second in the file", []*sourceFile{parse(goroutinePkg, goroutineFile, one+"\nfunc again() {\n\tgo open()\n}\n")}, []string{
			"metais.org/metais/internal/render/producer_assembled.go:6: a go statement beyond the assembled producer's one",
		}},
		{"one in a sink", []*sourceFile{
			parse(goroutinePkg, goroutineFile, one),
			parse("metais.org/metais/internal/render/sink", "archive.go", "package sink\n\nfunc emit() {\n\tgo emit()\n}\n"),
		}, []string{
			"metais.org/metais/internal/render/sink/archive.go:4: a go statement beyond the assembled producer's one",
		}},
		{"one in another file of render", []*sourceFile{
			parse(goroutinePkg, "producer_rendered.go", one),
		}, []string{
			"metais.org/metais/internal/render/producer_rendered.go:3: a go statement beyond the assembled producer's one",
			"metais.org/metais/internal/render/producer_assembled.go: no go statement, where the assembled producer's one should be",
		}},
	} {
		got := renderGoStatements(c.files)
		if strings.Join(got, "\n") != strings.Join(c.want, "\n") {
			t.Errorf("%s: got\n\t%s\nwant\n\t%s", c.name, strings.Join(got, "\n\t"), strings.Join(c.want, "\n\t"))
		}
	}
}
