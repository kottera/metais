// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package template

import (
	"github.com/kottera/metais/internal/finding"
	"github.com/kottera/metais/internal/limits"
	"github.com/kottera/metais/internal/liquid"
)

// The template family's codes this package reports (ADR-004 D16 item 1).
// Each is class content, about the template it is located in.
const (
	CodeSyntax            finding.Code = "template.syntax"
	CodeNotUTF8           finding.Code = "template.not_utf8"
	CodeInclude           finding.Code = "template.include"
	CodeUnknownFilter     finding.Code = "template.unknown_filter"
	CodeReservedName      finding.Code = "template.reserved_name"
	CodePartialName       finding.Code = "template.partial_name"
	CodePartialNotFound   finding.Code = "template.partial_not_found"
	CodeNumberRange       finding.Code = "template.number_range"
	CodeUndefinedVariable finding.Code = "template.undefined_variable"
	CodeUndefinedProperty finding.Code = "template.undefined_property"
	CodeNilProperty       finding.Code = "template.nil_property"
	CodeObjectOutput      finding.Code = "template.object_output"
	CodeComparison        finding.Code = "template.comparison"
	CodeFilterInput       finding.Code = "template.filter_input"
	CodeFilterArgument    finding.Code = "template.filter_argument"
	CodeTagArgument       finding.Code = "template.tag_argument"
	CodeDivisionByZero    finding.Code = "template.division_by_zero"
	CodeInvalidUTF8       finding.Code = "template.invalid_utf8"
)

// ParseCodes are the codes ParseEntry reports, beside limit.package: the
// codes of a template's parse (ADR-004 D16 item 1), which every command
// that parses a template declares.
func ParseCodes() []finding.Code {
	return []finding.Code{CodeSyntax, CodeNotUTF8, CodeInclude, CodeUnknownFilter, CodeReservedName, CodePartialName,
		CodeNumberRange, CodeFilterArgument, CodeFilterInput, CodeDivisionByZero}
}

// subject is the template whose entry is entry.
func subject(entry string) finding.Subject {
	return finding.Subject{Kind: finding.SubjectTemplate, Name: entry}
}

// errorAt is the location of e: its template entry, line and column.
func errorAt(e *liquid.Error) finding.Location {
	return finding.Location{Origin: finding.OriginPackage, File: e.Source, Line: max(e.Line, 1), Column: max(e.Col, 1)}
}

// parseFinding maps an error of a template's parse to its finding
// (ADR-004 D16 item 1): a code of the template family, or limit.package
// for template-nesting or template-size, which the template's own text
// breaches (ADR-009 D19). cause is a syntax error's cause (ADR-007 D2 item
// 4). The engine's own text is never passed on (D16 item 3).
func parseFinding(e *liquid.Error, lim limits.Set, cause string) finding.Finding {
	loc := errorAt(e)
	column := finding.Int(int64(max(e.Col, 1)))
	p := finding.Params{"column": column}
	var code finding.Code
	switch e.Code {
	case liquid.CodeSyntax:
		code, p["cause"] = CodeSyntax, finding.String(cause)
	case liquid.CodeNotUTF8:
		return finding.New(CodeNotUTF8, subject(e.Source), &loc, nil)
	case liquid.CodeInclude:
		code = CodeInclude
	case liquid.CodeUnknownFilter:
		code, p["filter"] = CodeUnknownFilter, finding.String(e.Args["filter"])
	case liquid.CodeReservedName:
		code, p["name"] = CodeReservedName, finding.String(e.Args["name"])
	case liquid.CodePartialName:
		code = CodePartialName
	case liquid.CodeNumberRange:
		code = CodeNumberRange
	case liquid.CodeFilterArgument:
		code, p["filter"], p["argument"] = CodeFilterArgument, finding.String(e.Args["filter"]), finding.String(e.Args["argument"])
	case liquid.CodeFilterInput:
		code, p["filter"], p["type"] = CodeFilterInput, finding.String(e.Args["filter"]), finding.String(e.Args["type"])
	case liquid.CodeDivisionByZero:
		code, p["filter"] = CodeDivisionByZero, finding.String(e.Args["filter"])
	case liquid.CodeLimit:
		id := limits.TemplateNesting
		if e.Args["limit"] == limits.Get(limits.TemplateSize).Name {
			id = limits.TemplateSize
		}
		return lim.Breached(id, loc, 0, false)
	default:
		panic("template: the Liquid parser met a defect: " + e.Error())
	}
	return finding.New(code, subject(e.Source), &loc, p)
}

// renderFinding maps the first error of rendering the output o to its
// finding (ADR-004 D16 items 1 and 2): a code of the template family, with
// the output's path, and the render tags that led to a partial as related
// locations; or a limit's breach, limit.exceeded.template for the work of
// the template (ADR-009 D11) and limit.exceeded.output for the bytes of
// the output.
func (r *Renderer) renderFinding(o Output, e *liquid.Error) finding.Finding {
	loc := errorAt(e)
	column := finding.Int(int64(max(e.Col, 1)))
	p := finding.Params{"column": column, "output_path": finding.String(o.Path)}
	var code finding.Code
	switch e.Code {
	case liquid.CodeUndefinedVariable:
		code, p["name"] = CodeUndefinedVariable, finding.String(e.Args["name"])
	case liquid.CodeUndefinedProperty:
		code, p["path"] = CodeUndefinedProperty, finding.String(e.Args["path"])
	case liquid.CodeNilProperty:
		code, p["path"] = CodeNilProperty, finding.String(e.Args["path"])
	case liquid.CodeObjectOutput:
		code = CodeObjectOutput
	case liquid.CodeComparison:
		code = CodeComparison
		p["operator"], p["left"], p["right"] = finding.String(e.Args["op"]), finding.String(e.Args["left"]), finding.String(e.Args["right"])
	case liquid.CodeFilterInput:
		code, p["filter"], p["type"] = CodeFilterInput, finding.String(e.Args["filter"]), finding.String(e.Args["type"])
	case liquid.CodeFilterArgument:
		code, p["filter"], p["argument"] = CodeFilterArgument, finding.String(e.Args["filter"]), finding.String(e.Args["argument"])
	case liquid.CodeArgument:
		code = CodeTagArgument
		p["argument"], p["type"] = finding.String(e.Args["argument"]), finding.String(e.Args["kind"])
		if tag, ok := e.Args["tag"]; ok {
			p["tag"] = finding.String(tag)
		}
	case liquid.CodeDivisionByZero:
		code, p["filter"] = CodeDivisionByZero, finding.String(e.Args["filter"])
	case liquid.CodeNumberRange:
		code = CodeNumberRange
		if f, ok := e.Args["filter"]; ok {
			p["filter"] = finding.String(f)
		}
	case liquid.CodeInvalidUTF8:
		code, p["filter"] = CodeInvalidUTF8, finding.String(e.Args["filter"])
	case liquid.CodeLimit:
		return r.limitFinding(o, e, loc)
	default:
		panic("template: the Liquid renderer met a defect: " + e.Error())
	}
	return finding.New(code, subject(e.Source), &loc, p, finding.WithRelated(chain(e)...))
}

// noteRenderedFrom is the note of each render tag on the way to a partial.
const noteRenderedFrom = "rendered from"

// chain is the render tags that led to a render error in a partial,
// outermost first, as related locations (ADR-004 D16 item 2).
func chain(e *liquid.Error) []finding.Related {
	var out []finding.Related
	for _, s := range e.Chain {
		out = append(out, finding.Related{
			Location: finding.Location{Origin: finding.OriginPackage, File: s.Source, Line: s.Line, Column: s.Col},
			Note:     noteRenderedFrom,
		})
	}
	return out
}

// limitFinding is a breach at render of the limit e names (ADR-009 D19):
// the bytes of one output (output-size) or of every output (output-total)
// are the output's, at the output's path; every other limit bounds the
// work of the template e is located in.
func (r *Renderer) limitFinding(o Output, e *liquid.Error, loc finding.Location) finding.Finding {
	l, ok := limits.Lookup(e.Args["limit"])
	if !ok {
		panic("template: the renderer breached an unknown limit " + e.Args["limit"])
	}
	if l.ID == limits.OutputSize || l.ID == limits.OutputTotal {
		out := finding.Location{Origin: finding.OriginOutput, File: o.Path}
		return r.lim.ExceededOutput(l.ID, finding.Subject{Kind: finding.SubjectOutput, Name: o.ID}, out, 0, false)
	}
	return r.lim.ExceededTemplate(l.ID, subject(e.Source), loc, o.Path, finding.WithRelated(chain(e)...))
}
