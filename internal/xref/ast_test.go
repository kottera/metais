// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package xref

import (
	"fmt"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"

	"metais.org/metais/internal/liquid/ast"
	"metais.org/metais/internal/manifest"
)

// everyConstruct is a template with a variable path in every place the
// dialect's grammar admits one, each rooted at a name nothing binds:
// v1, v2 and so on.
const everyConstruct = `{{ v1 | append: v2 | json: indent: v3 }}
{% if v4 == v5 and v6 contains "x" or v7 %}{% elsif v8 %}{% else %}{% endif %}
{% unless v9 %}{% elsif v10 %}{% endunless %}
{% case v11 %}{% when v12, v13 %}{% else %}{% endcase %}
{% for x in v14 limit: v15 offset: v16 %}{% else %}{% endfor %}
{% for i in (v17..v18) %}{% endfor %}
{% tablerow x in v19 cols: v20 limit: v21 offset: v22 %}{% endtablerow %}
{% assign a = v23 %}{% capture c %}{{ v24 }}{% endcapture %}
{% cycle v25, v26 %}{% cycle v27: "a", "b" %}
{% echo v28 %}
{% ifchanged %}{{ v29 }}{% endifchanged %}
{% liquid
  echo v30
  assign b = v31 %}
{% render "p.liquid", k: v32 %}{% render "p.liquid" with v33 %}{% render "p.liquid" for v34 %}
{{ inputs.m[v35] }}{{ inputs.m[v36.x] }}
{% embed "m", index: v37 %}{% asset "a.css", as: v38 %}
{% provenance package version %}`

// TestReadsTheTree holds the analysis to ADR-007 D3 item 1: it reads the
// engine's own tree, every variable path the tree holds and no other.
// Each path ast.Walk finds rooted at a vN is reported once, at the
// path's own position; provenance's keywords, which the tree holds as
// paths, are not references.
func TestReadsTheTree(t *testing.T) {
	fx := fixture{
		inputs:    []fxInput{in("m", `{"type": "object", "additionalProperties": true}`)},
		templates: map[string]string{"main.liquid": everyConstruct, "p.liquid": ``},
		assets:    []string{"a.css"},
		outputs:   []manifest.Output{tpl("main", "main.liquid")},
	}
	p := fx.pkg(t)
	var root *ast.Block
	for _, tm := range p.Templates {
		if tm.Path == "main.liquid" {
			root = tm.Root
		}
	}
	var want []string
	ast.Walk(root, func(n ast.Node) bool {
		if path, ok := n.(*ast.Path); ok && strings.HasPrefix(path.Root, "v") && strings.Trim(path.Root[1:], "0123456789") == "" {
			want = append(want, fmt.Sprintf("%s %d:%d", path.Root, path.Span.Line, path.Span.Col))
		}
		return true
	})
	if len(want) != 38 {
		t.Fatalf("the tree holds %d paths rooted at vN, want 38: %q", len(want), want)
	}
	r, err := Analyze(p)
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, f := range r.Findings {
		if f.Code() != CodeUnknownVariable {
			t.Errorf("unexpected %s", describe(t, f))
			continue
		}
		name, _ := f.Param("name")
		n, _ := name.Str()
		loc, _ := f.Location()
		got = append(got, fmt.Sprintf("%s %d:%d", n, loc.Line, loc.Column))
	}
	slices.Sort(got)
	slices.Sort(want)
	if !slices.Equal(got, want) {
		t.Errorf("reported\n\t%q\nthe tree's paths\n\t%q", got, want)
	}
}

// TestNoParserOfItsOwn holds the package to ADR-007 D3 item 1 and
// ADR-010 D4: templates arrive as internal/liquid/ast trees, and the
// manifest's expressions are parsed by internal/liquid's ParseValue,
// ParseCondition and Parse, which are the only parsers its release files
// call. They import nothing that scans text.
func TestNoParserOfItsOwn(t *testing.T) {
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	allowed := map[string]bool{
		"errors": true, "fmt": true, "maps": true, "net/url": true, "slices": true, "strconv": true,
		"strings": true, "unicode/utf8": true,
		"metais.org/metais/internal/finding":      true,
		"metais.org/metais/internal/limits":       true,
		"metais.org/metais/internal/liquid":       true,
		"metais.org/metais/internal/liquid/ast":   true,
		"metais.org/metais/internal/liquid/value": true,
		"metais.org/metais/internal/manifest":     true,
		"metais.org/metais/internal/regexprofile": true,
		"metais.org/metais/internal/schema":       true,
	}
	fset := token.NewFileSet()
	calls := map[string]bool{}
	for _, name := range files {
		if strings.HasSuffix(name, "_test.go") {
			continue
		}
		src, err := os.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		f, err := parser.ParseFile(fset, name, src, 0)
		if err != nil {
			t.Fatal(err)
		}
		for _, imp := range f.Imports {
			path, _ := strconv.Unquote(imp.Path.Value)
			if !allowed[path] {
				t.Errorf("%s imports %s", name, path)
			}
		}
		for _, fn := range []string{"liquid.Parse(", "liquid.ParseValue(", "liquid.ParseCondition("} {
			if strings.Contains(string(src), fn) {
				calls[fn] = true
			}
		}
	}
	if len(calls) != 3 {
		t.Errorf("the manifest's expressions are parsed by %v, want liquid's three parsers", calls)
	}
}
