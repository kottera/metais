// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package template

import (
	"io"
	"slices"
	"strings"

	"github.com/kottera/metais/internal/finding"
	"github.com/kottera/metais/internal/limits"
	"github.com/kottera/metais/internal/liquid"
	"github.com/kottera/metais/internal/liquid/ast"
	"github.com/kottera/metais/internal/manifest"
	"github.com/kottera/metais/internal/pkgtree"
)

// Dir is the directory of a package's templates (ADR-001 D5). A template's
// name, as a manifest declaration and a render tag write it, is its path
// beneath Dir; its entry is Dir followed by the name.
const Dir = "templates/"

// Templates is a package's templates, each parsed at most once, read in
// place from the package's tree (ADR-004 D8 item 7, ADR-009 D9): nothing
// of a template is read from anywhere else.
type Templates struct {
	tree   pkgtree.Tree
	lim    limits.Set
	inputs []manifest.Input
	parsed map[string]*parsed
}

// parsed is one template's parse: the template, or the findings that
// refused it.
type parsed struct {
	t        *liquid.Template
	findings []finding.Finding
}

// NewTemplates returns the templates of tree, parsed under lim's
// template-size and template-nesting, their embed tags checked against
// the inputs the manifest declares (ADR-004 D12 item 1).
func NewTemplates(tree pkgtree.Tree, lim limits.Set, inputs []manifest.Input) *Templates {
	return &Templates{tree: tree, lim: lim, inputs: inputs, parsed: map[string]*parsed{}}
}

// Has reports whether name is a template of the package.
func (ts *Templates) Has(name string) bool {
	_, err := ts.tree.Stat(Dir + name)
	return err == nil || canaryHas(name)
}

// canaryPartial, when set, is where a template the package lacks is read
// from instead. Only a canary build sets it (ADR-009 D16 item 2): the cwd
// canary falls back to the working directory, which the SC7 harness must
// catch. The tool's own build never does, and reads a template only from
// the package's tree.
var canaryPartial func(name string) ([]byte, bool)

func canaryRead(name string) ([]byte, bool) {
	if canaryPartial == nil {
		return nil, false
	}
	return canaryPartial(name)
}

func canaryHas(name string) bool {
	if canaryPartial == nil {
		return false
	}
	_, ok := canaryPartial(name)
	return ok
}

// Parse parses the template name, a path beneath templates/ that is an
// entry, once, and returns it, or nil and every finding of its parse
// (ADR-004 D16 items 1 and 4): at most one template.syntax, since a parse
// stops there, and every other parse-time finding before it. Every render
// tag must name a partial the package has (D8 item 2): one that does not is
// template.partial_not_found, found here, with no data. Every embed tag of
// a template that parsed must name a declared input, with index: exactly
// when it is a collection (D12 item 1): template.embed_unknown_input and
// template.embed_index, found here too. The findings are the same each
// time it is asked.
func (ts *Templates) Parse(name string) (*liquid.Template, []finding.Finding) {
	if p, ok := ts.parsed[name]; ok {
		return p.t, p.findings
	}
	p := ts.parse(name)
	ts.parsed[name] = p
	return p.t, p.findings
}

func (ts *Templates) parse(name string) *parsed {
	entry := Dir + name
	file := finding.Location{Origin: finding.OriginPackage, File: entry}
	var text []byte
	if info, err := ts.tree.Stat(entry); err == nil {
		if text, err = readAll(ts.tree, entry, info.Size); err != nil {
			panic("template: reading " + entry + ": " + err.Error())
		}
	} else if b, ok := canaryRead(name); ok {
		text = b
	} else {
		panic("template: " + entry + " is not an entry of the package")
	}
	t, all, stop := parseEntry(entry, text, ts.lim, false)
	for _, r := range renderTags(t, text, stop) {
		if ts.partialExists(r.name) {
			continue
		}
		loc := file
		loc.Line, loc.Column = r.span.Line, r.span.Col
		all = append(all, located{r.span.Start, finding.New(CodePartialNotFound, subject(entry), &loc, finding.Params{
			"partial": finding.String(r.name), "column": finding.Int(int64(r.span.Col)),
		})})
	}
	if t != nil {
		all = append(all, embedChecks(entry, t, ts.inputs, true)...)
	}
	if len(all) == 0 {
		return &parsed{t: t}
	}
	slices.SortStableFunc(all, func(a, b located) int { return a.offset - b.offset })
	var fs []finding.Finding
	for _, l := range all {
		fs = append(fs, l.f)
	}
	return &parsed{findings: fs}
}

// renderTag is a render tag's literal partial name, and where the tag is.
type renderTag struct {
	name string
	span ast.Span
}

// renderTags returns every render tag of a template whose partial is named
// by a literal (ADR-004 D8 item 1): from its syntax tree when it parsed,
// and otherwise from its constructs before stop, where the parse stopped,
// which the parser read, so that a missing partial is reported with the
// template's other parse findings in one pass (D16 item 4).
func renderTags(t *liquid.Template, text []byte, stop int) []renderTag {
	var out []renderTag
	if t != nil {
		ast.Walk(t.Root, func(n ast.Node) bool {
			if r, ok := n.(*ast.Render); ok {
				out = append(out, renderTag{r.Name, r.Span})
			}
			return true
		})
		return out
	}
	for _, c := range liquid.Constructs(string(text[:min(stop, len(text))])) {
		if c.Tag != "render" || c.Span.Start >= stop {
			continue
		}
		if name, ok := literalName(c.Markup); ok {
			out = append(out, renderTag{name, c.Span})
		}
	}
	return out
}

// literalName is the string literal a render tag's markup begins with:
// its text between quotes, which carries no escapes (ADR-004 D1 item 8).
func literalName(markup string) (string, bool) {
	m := strings.TrimLeft(markup, " \t\n\v\f\r")
	if m == "" || (m[0] != '"' && m[0] != '\'') {
		return "", false
	}
	end := strings.IndexByte(m[1:], m[0])
	if end < 0 {
		return "", false
	}
	return m[1 : 1+end], true
}

// partialExists reports whether a render tag's name names a template: a
// name confined beneath templates/ — no leading "/", no "\", no "." or
// ".." segment, no empty segment — matched byte for byte to an entry
// (ADR-004 D8 items 1 and 2).
func (ts *Templates) partialExists(name string) bool {
	if name == "" || strings.ContainsRune(name, '\\') {
		return false
	}
	if slices.ContainsFunc(strings.Split(name, "/"), func(s string) bool { return s == "" || s == "." || s == ".." }) {
		return false
	}
	return ts.Has(name)
}

// readAll reads the entry name of size bytes from tree.
func readAll(tree pkgtree.Tree, name string, size int64) ([]byte, error) {
	r, err := tree.Open(name)
	if err != nil {
		return nil, err
	}
	defer r.Close()
	b := make([]byte, 0, size)
	for {
		if len(b) == cap(b) {
			b = slices.Grow(b, 1)
		}
		n, err := r.Read(b[len(b):cap(b)])
		b = b[:len(b)+n]
		if err == io.EOF {
			return b, nil
		}
		if err != nil {
			return nil, err
		}
	}
}

// partials is the engine's Partials over ts: a render tag's partial is the
// template its literal names, parsed once (ADR-004 D8). Every name was
// found to exist when the calling template was parsed, and every
// template a render reaches parsed without a finding before it began.
type partials struct{ ts *Templates }

// Partials is the engine's Partials over ts, for a render of a template
// that Parse returned without a finding, every template its Reach names
// having parsed without one too.
func (ts *Templates) Partials() liquid.Partials { return partials{ts} }

func (p partials) Partial(name string) (*liquid.Template, *liquid.Error) {
	if !p.ts.partialExists(name) {
		return nil, &liquid.Error{Code: liquid.CodeInternal, Detail: "a render tag names a partial its template's parse did not find: " + name}
	}
	t, fs := p.ts.Parse(name)
	if t == nil {
		return nil, &liquid.Error{Code: liquid.CodeInternal, Detail: "a render reached a partial whose parse failed: " + name + ", " + string(fs[0].Code())}
	}
	return t, nil
}

// Reach returns the templates a render of name reaches: name, and every
// partial its render tags name, transitively, each once, in the order
// first met. A template whose parse failed reaches nothing further.
func (ts *Templates) Reach(name string) []string {
	seen := map[string]bool{name: true}
	out := []string{name}
	for i := 0; i < len(out); i++ {
		t, _ := ts.Parse(out[i])
		if t == nil {
			continue
		}
		ast.Walk(t.Root, func(n ast.Node) bool {
			if r, ok := n.(*ast.Render); ok && !seen[r.Name] {
				seen[r.Name] = true
				out = append(out, r.Name)
			}
			return true
		})
	}
	return out
}
