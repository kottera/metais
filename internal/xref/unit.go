// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package xref

import (
	"fmt"
	"slices"

	"github.com/kottera/metais/internal/finding"
	"github.com/kottera/metais/internal/liquid/ast"
	"github.com/kottera/metais/internal/liquid/value"
	"github.com/kottera/metais/internal/manifest"
	"github.com/kottera/metais/internal/schema"
)

// maxDeclared is how many declared names template.xref.unknown_field
// lists.
const maxDeclared = 20

// globalNames are the names ADR-004 D7 binds, and Liquid's loop objects.
// A partial that reads one nothing binds there is not reading a
// parameter.
var globalNames = map[string]bool{
	"inputs": true, "package": true, "item": true, "fan_out": true, "output": true, "now": true,
	"sources": true, "forloop": true, "tablerowloop": true,
}

// unit is the analysis of one template, partial or manifest expression in
// one context.
type unit struct {
	a   *analyzer
	ctx *context
	// file is the package entry the text is in: the template's, or the
	// manifest's for a manifest expression, which ptr then locates.
	file string
	ptr  string
	// partial is the call when the unit is a partial rendered by one.
	partial *call

	// locals are the template-scoped bindings, flow-insensitive (ADR-007
	// D3 item 7): each name's value is the union of every value bound to
	// it anywhere, computed to a fixed point. counters are increment's and
	// decrement's names.
	locals   map[string]set
	counters map[string]bool
	// blocks is the stack of block-scoped bindings: a for or tablerow
	// variable, and its loop object.
	blocks []map[string]set

	// reporting is set on the pass that reports; the passes before it
	// compute the locals. changed is set when a pass changes a local.
	reporting bool
	changed   bool
}

func (a *analyzer) unit(c *context, file string) *unit {
	return &unit{a: a, ctx: c, file: file, locals: map[string]set{}, counters: map[string]bool{}}
}

// exprUnit is a unit for the manifest expression at ptr.
func (a *analyzer) exprUnit(c *context, ptr string) *unit {
	u := a.unit(c, manifest.ManifestFile)
	u.ptr = ptr
	return u
}

// run analyses root: the passes that compute its locals to a fixed point,
// then the pass that reports.
func (u *unit) run(root ast.Node) {
	ast.Walk(root, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.Assign:
			u.locals[n.Name] = nil
		case *ast.Capture:
			u.locals[n.Name] = nil
		case *ast.Counter:
			u.counters[n.Name] = true
		}
		return true
	})
	if len(u.locals) > 0 {
		// Each pass that changes a local adds an alternative to it, and a
		// set holds at most maxAlternatives before it becomes unknown, so
		// this bound is never reached; the budget bounds the work besides.
		for range (maxAlternatives + 2) * (len(u.locals) + 1) {
			u.changed = false
			u.node(root)
			if !u.changed || u.a.exhausted {
				break
			}
		}
	}
	u.reporting = true
	u.node(root)
}

// bind adds v to the value of the local name.
func (u *unit) bind(name string, v set) {
	old := u.locals[name]
	nv := union(old, v)
	if nv.key() != old.key() {
		u.locals[name] = nv
		u.changed = true
	}
}

// location is where the text at sp is.
func (u *unit) location(sp ast.Span) finding.Location {
	if u.ptr != "" {
		l := finding.Location{Origin: finding.OriginPackage, File: u.file}.WithPointer(u.ptr)
		if u.a.pkg.Positions != nil {
			if p, ok := u.a.pkg.Positions.Value(u.ptr); ok {
				l = l.At(p)
			}
		}
		return l
	}
	return finding.Location{Origin: finding.OriginPackage, File: u.file, Line: sp.Line, Column: sp.Col}
}

// report records a finding about this unit's text at sp, with the related
// locations rel, on the reporting pass and while the budget lasts.
// withOutput adds the output whose context the unit is in, when there is
// one. A finding repeated by another context, or by another output's, is
// recorded once.
func (u *unit) report(code finding.Code, sp ast.Span, p finding.Params, withOutput bool, rel ...finding.Related) {
	if !u.reporting || u.a.exhausted {
		return
	}
	if p == nil {
		p = finding.Params{}
	}
	loc := u.location(sp)
	names := make([]string, 0, len(p))
	for n := range p {
		names = append(names, n)
	}
	slices.Sort(names)
	key := fmt.Sprintf("%s|%s|%+v", code, u.file, loc)
	for _, n := range names {
		key += fmt.Sprintf("|%s=%+v", n, p[n])
	}
	key += fmt.Sprintf("|%+v", rel)
	if withOutput && u.ctx.output != "" {
		p["output"] = finding.String(u.ctx.output)
	}
	f := finding.New(code, finding.Subject{Kind: finding.SubjectTemplate, Name: u.file}, &loc, p, finding.WithRelated(rel...))
	u.a.add(f, key)
}

// node analyses n and everything beneath it.
func (u *unit) node(n ast.Node) {
	switch n := n.(type) {
	case *ast.Block:
		for _, c := range n.Nodes {
			u.node(c)
		}
	case *ast.Output:
		u.expr(n.Value)
	case *ast.Echo:
		u.expr(n.Value)
	case *ast.If:
		u.branches(n.Branches, n.Else)
	case *ast.Unless:
		u.branches(n.Branches, n.Else)
	case *ast.Case:
		u.primary(n.Subject)
		u.block(n.Before)
		for _, w := range n.Clauses {
			for _, v := range w.Values {
				u.primary(v)
			}
			u.block(w.Body)
		}
	case *ast.For:
		coll := u.primary(n.Collection)
		u.primary(n.Limit)
		u.primary(n.Offset)
		loop := u.a.loopRecord(u.enclosingLoop())
		u.blocks = append(u.blocks, map[string]set{n.Var: u.elements(coll), "forloop": of(recordAlt(loop))})
		u.block(n.Body)
		u.blocks = u.blocks[:len(u.blocks)-1]
		u.block(n.Else)
	case *ast.Tablerow:
		coll := u.primary(n.Collection)
		u.primary(n.Cols)
		u.primary(n.Limit)
		u.primary(n.Offset)
		u.blocks = append(u.blocks, map[string]set{n.Var: u.elements(coll), "tablerowloop": of(recordAlt(u.a.tablerowRecord()))})
		u.block(n.Body)
		u.blocks = u.blocks[:len(u.blocks)-1]
	case *ast.Assign:
		u.bind(n.Name, u.expr(n.Value))
	case *ast.Capture:
		u.block(n.Body)
		u.bind(n.Name, str)
	case *ast.Cycle:
		u.primary(n.Group)
		for _, v := range n.Values {
			u.primary(v)
		}
	case *ast.Ifchanged:
		u.block(n.Body)
	case *ast.Liquid:
		u.block(n.Body)
	case *ast.Render:
		u.render(n)
	case *ast.Tag:
		u.tag(n)
	}
}

func (u *unit) block(b *ast.Block) {
	if b != nil {
		u.node(b)
	}
}

func (u *unit) branches(bs []*ast.Branch, els *ast.Block) {
	for _, b := range bs {
		u.cond(b.Cond)
		u.block(b.Body)
	}
	u.block(els)
}

// cond analyses every operand of a condition.
func (u *unit) cond(c *ast.Cond) {
	if c == nil {
		return
	}
	switch c.Kind {
	case ast.CondAnd, ast.CondOr:
		u.cond(c.X)
		u.cond(c.Y)
	default:
		u.primary(c.Left)
		u.primary(c.Right)
	}
}

// enclosingLoop is the forloop a new for loop's parentloop is: the
// innermost one in scope, or nil.
func (u *unit) enclosingLoop() set {
	for i := len(u.blocks) - 1; i >= 0; i-- {
		if v, ok := u.blocks[i]["forloop"]; ok {
			return v
		}
	}
	if u.partial != nil {
		if v, ok := u.partial.params["forloop"]; ok {
			return v
		}
	}
	return null
}

// expr evaluates a value expression: its base, then each filter.
func (u *unit) expr(e *ast.Expr) set {
	if e == nil {
		return unknown
	}
	v := u.primary(e.Base)
	for _, f := range e.Filters {
		v = u.filter(v, f)
	}
	return v
}

// primary evaluates a value without filters.
func (u *unit) primary(p ast.Primary) set {
	switch p := p.(type) {
	case *ast.Literal:
		if p == nil {
			return unknown
		}
		switch p.Kind {
		case ast.LitString:
			return str
		case ast.LitInteger, ast.LitFloat:
			return num
		case ast.LitTrue, ast.LitFalse:
			return boolean
		case ast.LitNil:
			return null
		}
		return unknown
	case *ast.Path:
		if p == nil {
			return unknown
		}
		return u.path(p)
	case *ast.Range:
		if p == nil {
			return unknown
		}
		u.primary(p.From)
		u.primary(p.To)
		return array(num)
	}
	return unknown
}

// path evaluates a variable path: its root (ADR-007 D3 item 3), then each
// segment.
func (u *unit) path(p *ast.Path) set {
	segs := p.Segments
	cur, isInputs := u.root(p)
	if isInputs && len(segs) == 0 {
		u.a.useAll(u.ctx)
	}
	for _, s := range segs {
		if !u.a.spend() {
			return unknown
		}
		cur = u.segment(cur, s)
	}
	return cur
}

// root resolves a path's root name (ADR-007 D3 item 3, ADR-004 D6 item
// 1): a block- or partial-scoped binding, a template-scoped binding, a
// counter, then a global bound in this context. isInputs is set when it
// is the global inputs.
func (u *unit) root(p *ast.Path) (v set, isInputs bool) {
	name := p.Root
	for i := len(u.blocks) - 1; i >= 0; i-- {
		if v, ok := u.blocks[i][name]; ok {
			return v, false
		}
	}
	if u.partial != nil {
		if v, ok := u.partial.params[name]; ok {
			return v, false
		}
	}
	if v, ok := u.locals[name]; ok {
		return v, false
	}
	if u.counters[name] {
		return num, false
	}
	if v, ok := u.global(name); ok {
		return v, name == "inputs"
	}
	switch {
	case u.ctx.unused:
		// Every name nothing binds may be a partial parameter.
	case u.partial != nil && !globalNames[name]:
		u.report(CodeParameterNotPassed, p.Span, finding.Params{"name": finding.String(name)}, true,
			finding.Related{Location: u.partial.site, Note: noteCallSite})
	default:
		params := finding.Params{"name": finding.String(name)}
		if s, ok := nearest(name, u.visible()); ok {
			params["suggestion"] = finding.String(s)
		}
		u.report(CodeUnknownVariable, p.Span, params, true)
	}
	return unknown, false
}

// global returns the global name's value, if this context binds it
// (ADR-004 D7 item 10).
func (u *unit) global(name string) (set, bool) {
	switch name {
	case "inputs":
		return of(recordAlt(u.a.inputsRec)), true
	case "package":
		return of(recordAlt(u.a.packageRec)), true
	case "now":
		return union(str, null), true
	case "sources":
		return of(recordAlt(u.a.sourcesRec)), true
	case "item":
		if u.ctx.fan != nil {
			return u.ctx.fan.item, true
		}
	case "fan_out":
		if u.ctx.fan != nil {
			return of(recordAlt(u.ctx.fan.rec)), true
		}
	case "output":
		if u.ctx.body {
			return of(recordAlt(u.a.outputRec)), true
		}
	}
	return nil, false
}

// visible returns every name bound where the unit is.
func (u *unit) visible() []string {
	var names []string
	for _, b := range u.blocks {
		for n := range b {
			names = append(names, n)
		}
	}
	if u.partial != nil {
		for n := range u.partial.params {
			names = append(names, n)
		}
	}
	for n := range u.locals {
		names = append(names, n)
	}
	for n := range u.counters {
		names = append(names, n)
	}
	for n := range globalNames {
		if _, ok := u.global(n); ok {
			names = append(names, n)
		}
	}
	slices.Sort(names)
	return slices.Compact(names)
}

// segment resolves one segment of a path on cur.
func (u *unit) segment(cur set, s *ast.Segment) set {
	if s.Index == nil {
		return u.property(cur, s.Name, s.Span)
	}
	if lit, ok := s.Index.(*ast.Literal); ok && lit != nil {
		switch lit.Kind {
		case ast.LitString:
			return u.property(cur, lit.Text, s.Span)
		case ast.LitInteger:
			if n, ok := lit.Value.(value.Number); ok {
				if i, ok := n.SafeInt(); ok {
					return u.index(cur, int(i), s.Span)
				}
			}
		}
		// Any other literal is an index no value has.
		return unknown
	}
	u.primary(s.Index)
	return u.dynamic(cur, s.Span)
}

// property resolves the property segment p on cur (ADR-007 D3 item 4):
// an error only when every alternative rejects it.
func (u *unit) property(cur set, p string, sp ast.Span) set {
	var out []set
	var rejs []*rejection
	for _, x := range cur {
		switch x.k {
		case kOpen, kUnknown:
			out = append(out, set{x})
		case kSchema:
			s, rej := u.a.schemaProperty(x.input, x.ptr, p)
			if rej != nil {
				rejs = append(rejs, rej)
			} else {
				out = append(out, s)
			}
		case kRecord:
			if f, ok := x.rec.fields[p]; ok {
				if x.rec.inputs {
					u.a.use(u.ctx, p)
				}
				out = append(out, f)
			} else if p == "size" {
				out = append(out, num)
			} else {
				rejs = append(rejs, &rejection{code: rejUnknownField, declared: x.rec.names})
			}
		case kArray:
			switch p {
			case "size":
				out = append(out, num)
			case "first", "last":
				out = append(out, x.elem)
			default:
				rejs = append(rejs, &rejection{code: rejNotAnObject, types: schema.TypeArray})
			}
		case kScalar:
			switch {
			case x.sc == sString && p == "size":
				out = append(out, num)
			case x.sc == sString && (p == "first" || p == "last"):
				out = append(out, str)
			case (x.sc == sNumber || x.sc == sBool) && (p == "size" || p == "first" || p == "last"):
				out = append(out, unknown)
			default:
				rejs = append(rejs, &rejection{code: rejNotAnObject, types: x.sc.types()})
			}
		}
	}
	if len(out) == 0 && len(rejs) > 0 {
		u.reject(rejs, p, 0, sp)
		return unknown
	}
	return union(out...)
}

// index resolves the index segment [i] on cur (ADR-007 D3 item 6).
func (u *unit) index(cur set, i int, sp ast.Span) set {
	var out []set
	var rejs []*rejection
	for _, x := range cur {
		switch x.k {
		case kOpen, kUnknown:
			out = append(out, set{x})
		case kSchema:
			s, rej := u.a.schemaIndex(x.input, x.ptr, i)
			if rej != nil {
				rejs = append(rejs, rej)
			} else {
				out = append(out, s)
			}
		case kArray:
			out = append(out, x.elem)
		case kRecord:
			rejs = append(rejs, &rejection{code: rejNotAnArray, types: schema.TypeObject})
		case kScalar:
			rejs = append(rejs, &rejection{code: rejNotAnArray, types: x.sc.types()})
		}
	}
	if len(out) == 0 && len(rejs) > 0 {
		u.reject(rejs, "", i, sp)
		return unknown
	}
	return union(out...)
}

// dynamic resolves a computed segment [expr] on cur (ADR-007 D3 item 6).
// An alternative it leaves nothing for is unknown, and the reference is
// template.xref.dynamic_reference; a computed name into inputs always is,
// and uses every input.
func (u *unit) dynamic(cur set, sp ast.Span) set {
	var out []set
	warn := false
	for _, x := range cur {
		var s set
		switch x.k {
		case kOpen, kUnknown:
			s = set{x}
		case kSchema:
			s = u.a.schemaDynamic(x.input, x.ptr)
		case kArray:
			s = x.elem
		case kRecord:
			if x.rec.inputs {
				u.a.useAll(u.ctx)
				warn = true
				s = unknown
				break
			}
			var fs []set
			for _, n := range x.rec.names {
				fs = append(fs, x.rec.fields[n])
			}
			s = union(fs...)
		}
		if len(s) == 0 {
			warn = true
			s = unknown
		}
		out = append(out, s)
	}
	if warn {
		u.report(CodeDynamicReference, sp, nil, true)
	}
	return union(out...)
}

// reject reports a segment every alternative rejected: one finding, whose
// code is the alternatives' (ADR-007 D3 items 4.6 and 6). p is the
// property read, or "" for an index, which is i.
func (u *unit) reject(rejs []*rejection, p string, i int, sp ast.Span) {
	var (
		code     = CodeUnknownField
		params   = finding.Params{}
		types    schema.TypeSet
		declared []string
		input    string
		related  []finding.Related
		seen     = map[string]bool{}
	)
	allNotObject, anyOutOfRange := true, false
	for _, r := range rejs {
		types |= r.types
		declared = append(declared, r.declared...)
		allNotObject = allNotObject && r.code == rejNotAnObject
		anyOutOfRange = anyOutOfRange || r.code == rejOutOfRange
		if input == "" {
			input = r.input
		}
		if r.ptr != "" && !seen[r.ptr] {
			seen[r.ptr] = true
			file, ptr := inFile(r.ptr)
			related = append(related, finding.Related{
				Location: finding.Location{Origin: finding.OriginPackage, File: file}.WithPointer(ptr),
				Note:     noteSchema,
			})
		}
	}
	switch {
	case p == "" && anyOutOfRange:
		code = CodeIndexOutOfRange
		params["index"] = finding.Int(int64(i))
	case p == "":
		code = CodeNotAnArray
		params["types"] = typeList(types)
	case allNotObject:
		code = CodeNotAnObject
		params["property"] = finding.String(p)
		params["types"] = typeList(types)
	default:
		slices.Sort(declared)
		declared = slices.Compact(declared)
		params["property"] = finding.String(p)
		params["declared_count"] = finding.Int(int64(len(declared)))
		if s, ok := nearest(p, declared); ok {
			params["suggestion"] = finding.String(s)
		}
		list := make([]finding.Value, 0, min(len(declared), maxDeclared))
		for _, d := range declared[:min(len(declared), maxDeclared)] {
			list = append(list, finding.String(d))
		}
		params["declared"] = finding.List(list...)
	}
	if input != "" {
		params["input"] = finding.String(input)
	}
	u.report(code, sp, params, true, related...)
}

// typeList is a types param.
func typeList(t schema.TypeSet) finding.Value {
	names := typeNames(t)
	vs := make([]finding.Value, 0, len(names))
	for _, n := range names {
		vs = append(vs, finding.String(n))
	}
	return finding.List(vs...)
}

// elements returns the element of each alternative of v, as a for loop
// binds it (ADR-007 D3 item 8) and a filter reads it: [expr] on an array,
// and unknown over an object or a scalar.
func (u *unit) elements(v set) set {
	var out []set
	for _, x := range v {
		switch x.k {
		case kSchema:
			out = append(out, u.a.schemaElements(x.input, x.ptr))
		case kArray:
			out = append(out, x.elem)
		case kOpen:
			out = append(out, open)
		default:
			out = append(out, unknown)
		}
	}
	return union(out...)
}

// elementAt returns what first (i = 0) or last (i = -1) gives of v: an
// array's element, a string's character. It reports nothing: a filter's
// input of the wrong type is the render's to refuse.
func (u *unit) elementAt(v set, i int) set {
	var out []set
	for _, x := range v {
		switch x.k {
		case kSchema:
			t := u.a.typesOf(x.ptr)
			var r set
			if t&schema.TypeArray != 0 {
				if s, rej := u.a.schemaIndex(x.input, x.ptr, i); rej == nil {
					r = union(r, s)
				}
			}
			if t&schema.TypeString != 0 {
				r = union(r, str)
			}
			if len(r) == 0 {
				r = unknown
			}
			out = append(out, r)
		case kArray:
			out = append(out, x.elem)
		case kScalar:
			if x.sc == sString {
				out = append(out, str)
			} else {
				out = append(out, unknown)
			}
		case kOpen:
			out = append(out, open)
		default:
			out = append(out, unknown)
		}
	}
	return union(out...)
}

// filter applies f to in by its analysis signature (ADR-007 D3 item 10).
// Every argument is analysed as an expression; a property-name argument
// is resolved as .p on the input's element and checked like a segment.
func (u *unit) filter(in set, f *ast.Filter) set {
	var positional []set
	for _, arg := range f.Args {
		v := u.primary(arg.Value)
		if arg.Name == "" {
			positional = append(positional, v)
		}
	}
	sig, ok := signatures[f.Name]
	if !ok {
		return unknown
	}
	var projected set
	if sig.property && len(f.Args) > 0 && f.Args[0].Name == "" {
		if lit, ok := f.Args[0].Value.(*ast.Literal); ok && lit != nil && lit.Kind == ast.LitString {
			if !u.a.spend() {
				return unknown
			}
			projected = u.property(u.elements(in), lit.Text, lit.Span)
		}
	}
	switch sig.kind {
	case sigPreserve:
		return in
	case sigDefault:
		if len(positional) > 0 {
			return union(in, positional[0])
		}
		return in
	case sigConcat:
		elem := u.elements(in)
		if len(positional) > 0 {
			elem = union(elem, u.elements(positional[0]))
		}
		return array(elem)
	case sigFirst:
		return u.elementAt(in, 0)
	case sigLast:
		return u.elementAt(in, -1)
	case sigProject:
		if projected == nil {
			return array(unknown)
		}
		return array(projected)
	case sigScalar:
		return scalarSet(sig.scalar)
	}
	return unknown
}

// render analyses a render tag (ADR-007 D3 item 8): its values in the
// caller's scope, then, on the reporting pass, the partial in a fresh
// scope holding the globals, its parameters and its with or for value.
func (u *unit) render(n *ast.Render) {
	params := map[string]set{}
	switch n.Mode {
	case ast.RenderWith:
		params[n.Alias] = u.primary(n.Value)
	case ast.RenderFor:
		params[n.Alias] = u.elements(u.primary(n.Value))
		params["forloop"] = of(recordAlt(u.a.loopRecord(null)))
	}
	for _, p := range n.Params {
		params[p.Name] = u.primary(p.Value)
	}
	if !u.reporting || u.a.exhausted {
		return
	}
	root, exists := u.a.templates[n.Name]
	switch {
	case !exists:
		u.report(CodeMissingPartial, n.Span, finding.Params{"partial": finding.String(n.Name)}, true)
		return
	case root == nil:
		if u.ctx.effective {
			u.a.unsound = true
		}
		return
	}
	u.a.analyzePartial(n.Name, root, u.ctx, &call{params: params, site: u.location(n.Span)})
}

// tag analyses a tag the dialect adds (ADR-007 D3 item 12): embed names an
// input and asset an asset, each by literal; provenance takes keywords,
// which are not references. Any other tag's arguments are expressions.
func (u *unit) tag(n *ast.Tag) {
	args := n.Args
	var first *ast.Literal
	if len(args) > 0 && args[0].Name == "" {
		if lit, ok := args[0].Value.(*ast.Literal); ok && lit != nil && lit.Kind == ast.LitString {
			first = lit
		}
	}
	switch n.Name {
	case "provenance":
		return
	case "embed":
		if first != nil {
			args = args[1:]
			if _, ok := u.a.inputs[first.Text]; ok {
				u.a.use(u.ctx, first.Text)
			} else {
				params := finding.Params{"name": finding.String(first.Text)}
				if s, ok := nearest(first.Text, u.a.inputNames); ok {
					params["suggestion"] = finding.String(s)
				}
				u.report(CodeUnknownInput, n.Span, params, false)
			}
		}
	case "asset":
		if first != nil {
			args = args[1:]
			if !u.a.assets[first.Text] {
				u.report(CodeMissingAsset, n.Span, finding.Params{"asset": finding.String(first.Text)}, false)
			}
		}
	}
	for _, arg := range args {
		u.primary(arg.Value)
	}
}
