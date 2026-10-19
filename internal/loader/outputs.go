// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package loader

import (
	"github.com/kottera/metais/internal/finding"
	"github.com/kottera/metais/internal/limits"
	"github.com/kottera/metais/internal/liquid"
	"github.com/kottera/metais/internal/liquid/ast"
	"github.com/kottera/metais/internal/manifest"
)

// The template family's codes the parse of a manifest expression reports
// (ADR-004 D16 items 1 and 5). Each is class content, about the template,
// which for a manifest expression is metais.yaml.
const (
	CodeTemplateSyntax        finding.Code = "template.syntax"
	CodeTemplateInclude       finding.Code = "template.include"
	CodeTemplateUnknownFilter finding.Code = "template.unknown_filter"
	CodeTemplateNumberRange   finding.Code = "template.number_range"
	CodeTemplateTagContext    finding.Code = "template.tag_context"
	CodeTemplateFilterArg     finding.Code = "template.filter_argument"
	CodeTemplateFilterInput   finding.Code = "template.filter_input"
	CodeTemplateDivision      finding.Code = "template.division_by_zero"
)

// The top-level directories a manifest path is resolved in.
const (
	inputsDir     = "inputs/"
	templatesDir  = "templates/"
	assetsDir     = "assets/"
	referencesDir = "references/"
)

// at is the location of the manifest value at ptr.
func (o *opening) at(ptr string) *finding.Location {
	l := manifestLocation.WithPointer(ptr)
	if pos := o.doc.src.Positions; pos != nil {
		if p, ok := pos.Value(ptr); ok {
			l = l.At(p)
		}
	}
	return &l
}

// paths is ADR-003 D1 item 4 and ADR-005 D1 item 2's rule that a path the
// manifest names is an entry: an input's schema and meaning and the
// input_constraints, which the manifest schema has already placed under
// inputs/; and each output's template, asset, mirror and validate, at
// every depth of a container.
func (o *opening) paths(p *Package) []finding.Finding {
	m := p.Manifest
	var out []finding.Finding
	missing := func(ptr, dir string) {
		out = append(out, finding.New(CodeManifestPathAbsent, manifestSubject, o.at(ptr), finding.Params{"directory": finding.String(dir)}))
	}
	for _, in := range m.Inputs {
		if !o.has(in.Schema) {
			missing(in.Pointer+"/schema", inputsDir)
		}
		if in.Meaning != "" && !o.has(in.Meaning) {
			missing(in.Pointer+"/meaning", inputsDir)
		}
	}
	if m.InputConstraints != "" && !o.has(m.InputConstraints) {
		missing(finding.Pointer("input_constraints"), inputsDir)
	}
	var walk func([]manifest.Output)
	walk = func(outs []manifest.Output) {
		for _, d := range outs {
			switch d.Form {
			case manifest.OutputTemplate:
				if !o.has(templatesDir + d.Source) {
					missing(d.Pointer+"/template", templatesDir)
				}
			case manifest.OutputAsset:
				if !o.hasTree(assetsDir, d.Source) {
					missing(d.Pointer+"/asset", assetsDir)
				}
			case manifest.OutputMirror:
				if !o.hasDir(templatesDir, d.Source) {
					missing(d.Pointer+"/mirror", templatesDir)
				}
			case manifest.OutputContainer:
				walk(d.Container.Entries)
			}
			if d.Validate != "" && !o.has(d.Validate) {
				missing(d.Pointer+"/validate", ".")
			}
		}
	}
	walk(m.Outputs)
	return out
}

// hasTree reports whether rel, beneath dir, names a file or a directory
// holding one; "." names dir itself (ADR-005 D5 item 1).
func (o *opening) hasTree(dir, rel string) bool {
	return o.has(dir+rel) || o.hasDir(dir, rel)
}

// hasDir reports whether rel, beneath dir, is a directory holding a file;
// "." is dir itself.
func (o *opening) hasDir(dir, rel string) bool {
	prefix := dir + rel + "/"
	if rel == "." {
		prefix = dir
	}
	return len(o.under(prefix)) > 0
}

// expressions parses every output declaration's path as a Liquid
// template, its when as a condition and its for_each as a value expression
// (ADR-005 D3 and D4, ADR-004 D16 item 5), in the dialect's grammar. A path
// computes text only, so render in a path is template.tag_context (ADR-005
// D4 item 1). Each finding is at the field's manifest pointer and line,
// its column within the expression given as a parameter.
//
// The dialect is Dialect(), the one a package's templates are parsed in.
func (o *opening) expressions(p *Package) []finding.Finding {
	d := Dialect()
	lim := ParseLimits(o.lim)
	var out []finding.Finding
	report := func(ptr string, e *liquid.Error) {
		out = append(out, o.templateFinding(ptr, e))
	}
	var walk func([]manifest.Output)
	walk = func(outs []manifest.Output) {
		for _, decl := range outs {
			if decl.Path != "" {
				ptr := decl.Pointer + "/path"
				t, errs := liquid.Parse(d, liquid.Source{Name: manifest.ManifestFile, Text: []byte(decl.Path)}, lim)
				for _, e := range errs {
					report(ptr, e)
				}
				if t != nil {
					ast.Walk(t.Root, func(n ast.Node) bool {
						if r, ok := n.(*ast.Render); ok {
							out = append(out, finding.New(CodeTemplateTagContext, templateSubject, o.at(ptr), finding.Params{
								"tag": finding.String("render"), "column": finding.Int(int64(r.Span.Col)),
							}))
						}
						return true
					})
				}
			}
			if decl.When != "" {
				if _, e := liquid.ParseCondition(d, liquid.Source{Name: manifest.ManifestFile, Text: []byte(decl.When)}, lim); e != nil {
					report(decl.Pointer+"/when", e)
				}
			}
			if decl.ForEach != "" {
				if _, e := liquid.ParseValue(d, liquid.Source{Name: manifest.ManifestFile, Text: []byte(decl.ForEach)}, lim); e != nil {
					report(decl.Pointer+"/for_each", e)
				}
			}
			if decl.Container != nil {
				walk(decl.Container.Entries)
			}
		}
	}
	walk(p.Manifest.Outputs)
	return out
}

// templateSubject is the template a manifest expression is part of: the
// manifest's own entry (ADR-006 D4 item 2).
var templateSubject = finding.Subject{Kind: finding.SubjectTemplate, Name: manifest.ManifestFile}

// templateFinding maps the engine's error e in the expression at ptr to its
// finding (ADR-004 D16 item 1): its code, and its column within the
// expression.
func (o *opening) templateFinding(ptr string, e *liquid.Error) finding.Finding {
	column := finding.Int(int64(max(e.Col, 1)))
	switch e.Code {
	case liquid.CodeInclude:
		return finding.New(CodeTemplateInclude, templateSubject, o.at(ptr), finding.Params{"column": column})
	case liquid.CodeUnknownFilter:
		return finding.New(CodeTemplateUnknownFilter, templateSubject, o.at(ptr), finding.Params{"filter": finding.String(e.Args["filter"]), "column": column})
	case liquid.CodeFilterArgument:
		return finding.New(CodeTemplateFilterArg, templateSubject, o.at(ptr), finding.Params{"filter": finding.String(e.Args["filter"]), "argument": finding.String(e.Args["argument"]), "column": column})
	case liquid.CodeFilterInput:
		return finding.New(CodeTemplateFilterInput, templateSubject, o.at(ptr), finding.Params{"filter": finding.String(e.Args["filter"]), "type": finding.String(e.Args["type"]), "column": column})
	case liquid.CodeDivisionByZero:
		return finding.New(CodeTemplateDivision, templateSubject, o.at(ptr), finding.Params{"filter": finding.String(e.Args["filter"]), "column": column})
	case liquid.CodeNumberRange:
		return finding.New(CodeTemplateNumberRange, templateSubject, o.at(ptr), finding.Params{"column": column})
	case liquid.CodePartialName:
		return finding.New(CodeTemplateTagContext, templateSubject, o.at(ptr), finding.Params{"tag": finding.String("render"), "column": column})
	case liquid.CodeLimit:
		id := limits.TemplateNesting
		if e.Args["limit"] == limits.Get(limits.TemplateSize).Name {
			id = limits.TemplateSize
		}
		return o.lim.Breached(id, *o.at(ptr), 0, false)
	case liquid.CodeInternal:
		panic("loader: the Liquid parser met a defect: " + e.Detail)
	}
	// CodeSyntax, and CodeNotUTF8 and CodeReservedName, which a manifest
	// string under Core(), which reserves no name, cannot give.
	return finding.New(CodeTemplateSyntax, templateSubject, o.at(ptr), finding.Params{"column": column})
}

// isMarkdown reports whether the entry name ends in .md or .markdown,
// ASCII case-insensitively (ADR-008 D17 item 1).
func isMarkdown(name string) bool {
	return hasSuffixFold(name, ".md") || hasSuffixFold(name, ".markdown")
}

// hasSuffixFold reports whether s ends in suffix, an ASCII lower-case
// string, comparing ASCII letters without regard to case.
func hasSuffixFold(s, suffix string) bool {
	if len(s) < len(suffix) {
		return false
	}
	tail := s[len(s)-len(suffix):]
	for i := 0; i < len(suffix); i++ {
		c := tail[i]
		if 'A' <= c && c <= 'Z' {
			c += 'a' - 'A'
		}
		if c != suffix[i] {
			return false
		}
	}
	return true
}
