// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package xref

import (
	"strings"

	"github.com/kottera/metais/internal/finding"
	"github.com/kottera/metais/internal/liquid"
	"github.com/kottera/metais/internal/liquid/ast"
	"github.com/kottera/metais/internal/manifest"
	"github.com/kottera/metais/internal/schema"
)

// run analyses every output in the manifest's order, then every template
// no output renders, and reports the inputs nothing uses (ADR-007 D3
// items 9 and 11).
func (a *analyzer) run() {
	for _, o := range a.pkg.Manifest.Outputs {
		a.markExplicit(o)
	}
	for _, o := range a.pkg.Manifest.Outputs {
		a.output(o, nil, o.ID)
	}
	named := map[string]bool{}
	for _, p := range a.paths {
		if root := a.templates[p]; root != nil {
			ast.Walk(root, func(n ast.Node) bool {
				if r, ok := n.(*ast.Render); ok {
					named[r.Name] = true
				}
				return true
			})
		}
	}
	unused := &context{unused: true, body: true}
	for _, p := range a.paths {
		if a.rendered[p] || named[p] {
			continue
		}
		file := TemplatesDir + p
		loc := finding.Location{Origin: finding.OriginPackage, File: file}
		a.add(finding.New(CodeUnusedTemplate, finding.Subject{Kind: finding.SubjectTemplate, Name: file}, &loc, nil),
			string(CodeUnusedTemplate)+"|"+file)
		a.analyzeTemplate(p, unused)
	}
	if a.allUsed || a.unsound || a.exhausted {
		return
	}
	for _, decl := range a.pkg.Manifest.Inputs {
		if a.used[decl.Name] {
			continue
		}
		loc := finding.Location{Origin: finding.OriginPackage, File: manifest.ManifestFile}.WithPointer(decl.Pointer)
		if a.pkg.Positions != nil {
			if p, ok := a.pkg.Positions.Value(decl.Pointer); ok {
				loc = loc.At(p)
			}
		}
		a.add(finding.New(CodeUnusedInput, finding.Subject{Kind: finding.SubjectInput, Name: decl.Name, Title: decl.Title}, &loc, nil),
			string(CodeUnusedInput)+"|"+decl.Name)
	}
}

// markExplicit records every template a template output names, at any
// depth: no mirror renders one (ADR-005 D6 item 3).
func (a *analyzer) markExplicit(o manifest.Output) {
	switch o.Form {
	case manifest.OutputTemplate:
		a.explicit[o.Source] = true
	case manifest.OutputContainer:
		if o.Container != nil {
			for _, e := range o.Container.Entries {
				a.markExplicit(e)
			}
		}
	}
}

// output analyses the declaration o, of the top-level output top, inside
// the fan-out outer when an enclosing container has for_each (ADR-007 D3
// item 9, ADR-004 D7 item 10): its for_each without item, then its when,
// its path and its templates with item and fan_out where it fans out.
func (a *analyzer) output(o manifest.Output, outer *fan, top string) {
	fn := outer
	if o.ForEach != "" {
		fn = a.forEach(o, top)
	}
	c := &context{output: top, effective: true, fan: fn}
	if o.When != "" {
		cond, err := liquid.ParseCondition(a.pkg.Dialect, a.source(o.When), a.pkg.ParseLimits)
		if err != nil {
			a.unsound = true
		} else if a.enter() {
			u := a.exprUnit(c, o.Pointer+"/when")
			u.reporting = true
			u.cond(cond)
		}
	}
	if o.Path != "" {
		t, errs := liquid.Parse(a.pkg.Dialect, a.source(o.Path), a.pkg.ParseLimits)
		if len(errs) > 0 {
			a.unsound = true
		} else if a.enter() {
			a.exprUnit(c, o.Pointer+"/path").run(t.Root)
		}
	}
	body := *c
	body.body = true
	switch o.Form {
	case manifest.OutputTemplate:
		a.rendered[o.Source] = true
		a.analyzeTemplate(o.Source, &body)
	case manifest.OutputMirror:
		for _, p := range a.mirrored(o.Source) {
			a.rendered[p] = true
			a.analyzeTemplate(p, &body)
		}
	case manifest.OutputContainer:
		if o.Container != nil {
			for _, e := range o.Container.Entries {
				a.output(e, fn, top)
			}
		}
	}
}

// forEach analyses o's for_each in the context without item, holds its
// value to permitting arrays, and returns the fan-out its elements make.
func (a *analyzer) forEach(o manifest.Output, top string) *fan {
	ptr := o.Pointer + "/for_each"
	e, err := liquid.ParseValue(a.pkg.Dialect, a.source(o.ForEach), a.pkg.ParseLimits)
	if err != nil || !a.enter() {
		if err != nil {
			a.unsound = true
		}
		return a.fan(unknown)
	}
	u := a.exprUnit(&context{output: top, effective: true}, ptr)
	u.reporting = true
	v := u.expr(e)
	if t, ok := a.notArray(v); !ok && !a.exhausted {
		loc := u.location(ast.Span{})
		a.add(finding.New(CodeForEachNotArray, finding.Subject{Kind: finding.SubjectOutput, Name: top}, &loc,
			finding.Params{"types": typeList(t)}), string(CodeForEachNotArray)+"|"+ptr)
	}
	return a.fan(u.elements(v))
}

// fan returns the fan-out whose item is item.
func (a *analyzer) fan(item set) *fan {
	return &fan{item: item, rec: a.fanRecord(item)}
}

// notArray returns the types v may have and false when no alternative of
// v permits an array.
func (a *analyzer) notArray(v set) (schema.TypeSet, bool) {
	var t schema.TypeSet
	for _, x := range v {
		switch x.k {
		case kSchema:
			xt := a.typesOf(x.ptr)
			if xt&schema.TypeArray != 0 {
				return 0, true
			}
			t |= xt
		case kScalar:
			t |= x.sc.types()
		case kRecord:
			t |= schema.TypeObject
		default:
			return 0, true
		}
	}
	return t, len(v) == 0
}

// mirrored returns every template a mirror of dir renders (ADR-005 D6):
// those beneath templates/<dir>, in byte order, but for one with a name
// beginning with _ at any depth beneath dir and one a template output
// names.
func (a *analyzer) mirrored(dir string) []string {
	var out []string
	for _, p := range a.paths {
		rel := p
		if dir != "." {
			r, ok := strings.CutPrefix(p, strings.TrimSuffix(dir, "/")+"/")
			if !ok {
				continue
			}
			rel = r
		}
		partial := false
		for seg := range strings.SplitSeq(rel, "/") {
			if strings.HasPrefix(seg, "_") {
				partial = true
				break
			}
		}
		if !partial && !a.explicit[p] {
			out = append(out, p)
		}
	}
	return out
}

// source is a manifest expression's text, named by the manifest file.
func (a *analyzer) source(text string) liquid.Source {
	return liquid.Source{Name: manifest.ManifestFile, Text: []byte(text)}
}
