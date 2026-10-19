// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package xref

import (
	"fmt"
	"slices"
	"strings"

	"github.com/kottera/metais/internal/finding"
	"github.com/kottera/metais/internal/liquid/ast"
	"github.com/kottera/metais/internal/regexprofile"
	"github.com/kottera/metais/internal/schema"
)

// input is one declared input as the analysis knows it.
type input struct {
	index int    // its position in the manifest's inputs
	root  string // its schema file's root, in the combined schema
}

// analyzer is one analysis of one package.
type analyzer struct {
	pkg        *Package
	walker     *schema.Walker
	inputs     map[string]input
	inputNames []string // in manifest order
	templates  map[string]*ast.Block
	paths      []string // every template's path, sorted
	assets     map[string]bool

	// explicit is every template a template output names, which no mirror
	// renders (ADR-005 D6 item 3); rendered is every template an output
	// renders.
	explicit map[string]bool
	rendered map[string]bool

	appl     map[string]*applicable
	types    map[string]schema.TypeSet
	files    map[string]map[string]bool
	patterns map[string]*regexprofile.Regexp
	records  map[string]*record

	inputsRec, packageRec, sourcesRec, outputRec *record

	findings []finding.Finding
	seen     map[string]bool

	// used is every input a reference in an effective template or a
	// manifest expression passes through; allUsed is set when one uses
	// inputs whole or by a computed name. unsound is set when some
	// effective template or expression could not be read, so that no input
	// can be called unused.
	used    map[string]bool
	allUsed bool
	unsound bool

	contexts, segments int
	exhausted          bool

	memo  map[string]bool // every template-context analysis made
	stack map[string]bool // the partial analyses in progress
}

func newAnalyzer(p *Package) (*analyzer, error) {
	var w *schema.Walker
	if !p.NoSchemas {
		var err error
		if w, err = schema.NewWalker(p.Schema, schema.CombinedURI); err != nil {
			return nil, fmt.Errorf("%w: %v", ErrPackage, err)
		}
	}
	a := &analyzer{
		pkg:       p,
		walker:    w,
		inputs:    map[string]input{},
		templates: map[string]*ast.Block{},
		assets:    map[string]bool{},
		explicit:  map[string]bool{},
		rendered:  map[string]bool{},
		appl:      map[string]*applicable{},
		types:     map[string]schema.TypeSet{},
		files:     map[string]map[string]bool{},
		patterns:  map[string]*regexprofile.Regexp{},
		records:   map[string]*record{},
		seen:      map[string]bool{},
		used:      map[string]bool{},
		memo:      map[string]bool{},
		stack:     map[string]bool{},
	}
	for i, in := range p.Manifest.Inputs {
		if w == nil {
			a.inputs[in.Name] = input{index: i}
			a.inputNames = append(a.inputNames, in.Name)
			continue
		}
		root, ok, err := w.Reference("/properties/"+escape(in.Name), "$ref")
		if err != nil || !ok {
			return nil, fmt.Errorf("%w: the combined schema has no schema for the input %q", ErrPackage, in.Name)
		}
		a.inputs[in.Name] = input{index: i, root: root}
		a.inputNames = append(a.inputNames, in.Name)
	}
	for _, t := range p.Templates {
		a.templates[t.Path] = t.Root
		a.paths = append(a.paths, t.Path)
	}
	slices.Sort(a.paths)
	a.paths = slices.Compact(a.paths)
	for _, s := range p.Assets {
		a.assets[s] = true
	}
	a.globals()
	return a, nil
}

// record returns the record of fields, interned.
func (a *analyzer) record(fields map[string]set, inputs bool) *record {
	names := make([]string, 0, len(fields))
	for n := range fields {
		names = append(names, n)
	}
	slices.Sort(names)
	key := recordKey(fields, names, inputs)
	if r, ok := a.records[key]; ok {
		return r
	}
	r := &record{key: key, fields: fields, names: names, inputs: inputs}
	a.records[key] = r
	return r
}

// globals builds the records of the globals every context binds (ADR-004
// D7): inputs, package, sources and output. now is a string or nil.
func (a *analyzer) globals() {
	in := map[string]set{}
	src := map[string]set{}
	for _, decl := range a.pkg.Manifest.Inputs {
		if a.walker == nil {
			in[decl.Name] = of(unknownAlt)
		} else {
			in[decl.Name] = of(schemaAlt(decl.Name, a.inputs[decl.Name].root))
		}
		if decl.Collection {
			item := a.record(map[string]set{"name": str, "stem": str, "form": str}, false)
			src[decl.Name] = array(union(of(recordAlt(item)), null))
		} else {
			origin := a.record(map[string]set{"origin": union(str, null)}, false)
			src[decl.Name] = union(of(recordAlt(origin)), null)
		}
	}
	a.inputsRec = a.record(in, true)
	a.sourcesRec = a.record(src, false)
	optional := union(str, null)
	author := a.record(map[string]set{"name": str, "email": optional, "url": optional}, false)
	a.packageRec = a.record(map[string]set{
		"metais": str, "name": str, "version": str, "title": str, "description": str, "guide": str,
		"authors": union(array(of(recordAlt(author))), null), "license": optional, "homepage": optional,
		"digest": str,
	}, false)
	a.outputRec = a.record(map[string]set{"path": str, "root": str}, false)
}

// loopRecord returns forloop's record (ADR-004 D2), whose parentloop is
// parent.
func (a *analyzer) loopRecord(parent set) *record {
	return a.record(map[string]set{
		"index": num, "index0": num, "rindex": num, "rindex0": num, "length": num,
		"first": boolean, "last": boolean, "name": str, "parentloop": parent,
	}, false)
}

// tablerowRecord returns tablerowloop's record, with Shopify's fields.
func (a *analyzer) tablerowRecord() *record {
	return a.record(map[string]set{
		"index": num, "index0": num, "rindex": num, "rindex0": num, "length": num,
		"first": boolean, "last": boolean, "col": num, "col0": num, "col_first": boolean,
		"col_last": boolean, "row": num,
	}, false)
}

// fanRecord returns fan_out's record (ADR-004 D7 item 5) for the item
// value item.
func (a *analyzer) fanRecord(item set) *record {
	neighbour := union(item, null)
	return a.record(map[string]set{
		"index": num, "index0": num, "rindex": num, "rindex0": num, "length": num,
		"first": boolean, "last": boolean, "previous": neighbour, "next": neighbour,
	}, false)
}

// context is what a template or an expression is analysed in: which
// globals are bound, and on whose behalf (ADR-004 D7 item 10, ADR-007 D3
// item 9).
type context struct {
	// output is the id of the top-level output the analysis is for; ""
	// for a template no output renders.
	output string
	// effective is set for a context an output renders: its references
	// count as use (ADR-007 D3 item 11).
	effective bool
	// unused is set for a template no output renders and no render names:
	// a name nothing binds may be a partial parameter, and is unknown.
	unused bool
	// fan is set in a fan-out: item's value, and fan_out's record.
	fan *fan
	// body is set in a template body, where output is bound.
	body bool
}

type fan struct {
	item set
	rec  *record
}

// key is the context's identity for memoising: the output is not part of
// it, so a template two outputs render in one context is analysed once.
func (c *context) key() string {
	var b strings.Builder
	fmt.Fprintf(&b, "%t,%t,%t", c.effective, c.unused, c.body)
	if c.fan != nil {
		b.WriteString(",item=")
		b.WriteString(c.fan.item.key())
	}
	return b.String()
}

// enter counts one template-context analysis, and reports whether the
// budget allows it.
func (a *analyzer) enter() bool {
	if a.exhausted {
		return false
	}
	if a.contexts >= MaxContexts {
		a.exhaust("contexts", MaxContexts)
		return false
	}
	a.contexts++
	return true
}

// spend counts one resolved segment, and reports whether the budget
// allows it.
func (a *analyzer) spend() bool {
	if a.exhausted {
		return false
	}
	if a.segments >= MaxSegments {
		a.exhaust("segments", MaxSegments)
		return false
	}
	a.segments++
	return true
}

func (a *analyzer) exhaust(bound string, limit int) {
	a.exhausted = true
	a.add(finding.New(CodeBudgetExhausted, finding.Subject{Kind: finding.SubjectPackage}, nil, finding.Params{
		"bound": finding.String(bound),
		"limit": finding.Int(int64(limit)),
	}), string(CodeBudgetExhausted))
}

// add records f, unless a finding with the same key was recorded.
func (a *analyzer) add(f finding.Finding, key string) {
	if a.seen[key] {
		return
	}
	a.seen[key] = true
	a.findings = append(a.findings, f)
}

// use records that an effective reference passed through inputs.<name>.
func (a *analyzer) use(c *context, name string) {
	if c.effective {
		a.used[name] = true
	}
}

// useAll records that an effective reference used inputs whole, or
// selected an input by a computed name.
func (a *analyzer) useAll(c *context) {
	if c.effective {
		a.allUsed = true
	}
}

// analyzeTemplate analyses the template at path in c, once per context.
func (a *analyzer) analyzeTemplate(path string, c *context) {
	root, exists := a.templates[path]
	switch {
	case !exists:
		// The manifest names no such template: the loader's finding.
		return
	case root == nil:
		if c.effective {
			a.unsound = true
		}
		return
	}
	key := "t|" + path + "|" + c.key()
	if a.memo[key] {
		return
	}
	a.memo[key] = true
	if !a.enter() {
		return
	}
	u := a.unit(c, TemplatesDir+path)
	u.run(root)
}

// call is a render tag's call of a partial: what it binds there, and
// where it is.
type call struct {
	params map[string]set
	site   finding.Location
}

// analyzePartial analyses the partial at path rendered by cl in c, once
// per context and bound values, and never again while it is already
// being analysed with them (ADR-007 D3 item 8).
func (a *analyzer) analyzePartial(path string, root *ast.Block, c *context, cl *call) {
	names := make([]string, 0, len(cl.params))
	for n := range cl.params {
		names = append(names, n)
	}
	slices.Sort(names)
	var b strings.Builder
	b.WriteString("p|" + path + "|" + c.key())
	for _, n := range names {
		fmt.Fprintf(&b, "|%q=%s", n, cl.params[n].key())
	}
	key := b.String()
	if a.memo[key] || a.stack[key] {
		return
	}
	a.memo[key] = true
	if !a.enter() {
		return
	}
	a.stack[key] = true
	defer delete(a.stack, key)
	u := a.unit(c, TemplatesDir+path)
	u.partial = cl
	u.run(root)
}
