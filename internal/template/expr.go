// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package template

import (
	"bytes"
	"strconv"
	"strings"

	"github.com/kottera/metais/internal/finding"
	"github.com/kottera/metais/internal/limits"
	"github.com/kottera/metais/internal/liquid"
	"github.com/kottera/metais/internal/liquid/ast"
	"github.com/kottera/metais/internal/liquid/value"
	"github.com/kottera/metais/internal/loader"
	"github.com/kottera/metais/internal/manifest"
)

// CodeForEachType is a for_each whose value is neither an array nor nil
// (ADR-005 D3 item 2), in the output.expression family ADR-005's
// interfaces name. It is class content, about the output.
const CodeForEachType finding.Code = "output.expression.for_each_type"

// Exprs is one declaration's manifest expressions, parsed in the dialect
// (ADR-005 D3 and D4, ADR-004 D16 item 5): its for_each, its when and its
// path, each nil when the declaration omits it.
type Exprs struct {
	// ID names the top-level output the declaration is, or is inside.
	ID      string
	ForEach *ast.Expr
	When    *ast.Cond
	Path    *liquid.Template
	// Literal is set for a path that holds no Liquid markup: the path the
	// manifest writes is the path, the package's own content (ADR-009
	// D19).
	Literal bool
	// at locates a field of the declaration: its manifest pointer, and its
	// line and column in metais.yaml.
	at  func(field string) finding.Location
	ptr string
}

// ParseExprs parses o's for_each as a value expression, its when as a
// condition and its path as a template in the restricted mode of ADR-005
// D4 item 1, under lim, and returns them, or nil and every finding of
// their parse, each located by at at the field that holds it, with its
// column within the expression (ADR-004 D16 item 5). id names the
// top-level output o is, or is inside.
func ParseExprs(o manifest.Output, id string, at func(ptr string) finding.Location, lim limits.Set) (*Exprs, []finding.Finding) {
	x := &Exprs{ID: id, ptr: o.Pointer, at: func(field string) finding.Location { return at(o.Pointer + "/" + field) }}
	pl := loader.ParseLimits(lim)
	src := func(text string) liquid.Source {
		return liquid.Source{Name: manifest.ManifestFile, Text: []byte(text)}
	}
	var fs []finding.Finding
	report := func(field string, e *liquid.Error) {
		fs = append(fs, exprParseFinding(e, x.at(field), lim))
	}
	if o.ForEach != "" {
		e, err := liquid.ParseValue(dialect, src(o.ForEach), pl)
		if err != nil {
			report("for_each", err)
		}
		x.ForEach = e
	}
	if o.When != "" {
		c, err := liquid.ParseCondition(dialect, src(o.When), pl)
		if err != nil {
			report("when", err)
		}
		x.When = c
	}
	if o.Path != "" {
		t, errs := liquid.Parse(dialect, src(o.Path), pl)
		for _, e := range errs {
			report("path", e)
		}
		if t != nil {
			// A path computes text only: render, and every tag that reads
			// beyond the path's own text, is refused in it (ADR-005 D4
			// item 1).
			ast.Walk(t.Root, func(n ast.Node) bool {
				if r, ok := n.(*ast.Render); ok {
					loc := x.at("path")
					fs = append(fs, finding.New(loader.CodeTemplateTagContext, exprSubject, &loc, finding.Params{
						"tag": finding.String("render"), "column": finding.Int(int64(r.Span.Col)),
					}))
				}
				return true
			})
		}
		x.Path = t
		x.Literal = !strings.Contains(o.Path, "{{") && !strings.Contains(o.Path, "{%")
	}
	if len(fs) > 0 {
		return nil, fs
	}
	return x, nil
}

// exprSubject is the template a manifest expression is part of: the
// manifest's own entry (ADR-006 D4 item 2).
var exprSubject = finding.Subject{Kind: finding.SubjectTemplate, Name: manifest.ManifestFile}

// exprParseFinding is the finding of the parse error e of a manifest
// expression located at at: check's, and template.reserved_name, which
// only the dialect of a render reserves.
func exprParseFinding(e *liquid.Error, at finding.Location, lim limits.Set) finding.Finding {
	if e.Code == liquid.CodeReservedName {
		return finding.New(CodeReservedName, exprSubject, &at, finding.Params{
			"name": finding.String(e.Args["name"]), "column": finding.Int(int64(max(e.Col, 1))),
		})
	}
	return loader.TemplateFinding(e, exprSubject, at, lim, loader.CauseGrammar)
}

// Fan is one evaluation of a for_each declaration: the element bound to
// item, and its position in the for_each array, from which fan_out is
// made (ADR-004 D7 items 4 and 5). A nil *Fan is the one evaluation of a
// declaration without for_each, where neither is bound.
type Fan struct {
	elems []value.Value
	i     int
}

// Index is the evaluation's 0-based position in the for_each array.
func (f *Fan) Index() int { return f.i }

// bind adds item and fan_out to ms: the element, and its position counted
// in the whole for_each array, before any when is applied (ADR-004 D7
// item 5), with its neighbours, nil at either end.
func (f *Fan) bind(ms []value.Member) []value.Member {
	n := len(f.elems)
	neighbour := func(i int) value.Value {
		if i < 0 || i >= n {
			return value.Null{}
		}
		return f.elems[i]
	}
	fanOut := obj(
		value.Member{Name: "index", Value: integer(f.i + 1)},
		value.Member{Name: "index0", Value: integer(f.i)},
		value.Member{Name: "rindex", Value: integer(n - f.i)},
		value.Member{Name: "rindex0", Value: integer(n - f.i - 1)},
		value.Member{Name: "first", Value: value.Bool(f.i == 0)},
		value.Member{Name: "last", Value: value.Bool(f.i == n-1)},
		value.Member{Name: "length", Value: integer(n)},
		value.Member{Name: "previous", Value: neighbour(f.i - 1)},
		value.Member{Name: "next", Value: neighbour(f.i + 1)},
	)
	return append(ms, value.Member{Name: "item", Value: f.elems[f.i]}, value.Member{Name: "fan_out", Value: fanOut})
}

// Fans evaluates x's for_each, with neither item nor fan_out bound
// (ADR-004 D7 item 10), and returns one Fan per element of its array, in
// array order; none for nil (ADR-005 D3 item 2). A declaration without
// for_each has one evaluation, the nil Fan. A value of any other type is
// output.expression.for_each_type; an error of the expression is its
// finding. Each element is one iteration of loop-iterations (ADR-009 D3),
// charged before the first.
func (r *Renderer) Fans(x *Exprs) ([]*Fan, *finding.Finding) {
	if x.ForEach == nil {
		return []*Fan{nil}, nil
	}
	v, e := liquid.Eval(x.ForEach, r.exprEnv(nil, nil))
	if e != nil {
		f := r.exprFinding(x, "for_each", e)
		return nil, &f
	}
	switch v := v.(type) {
	case value.Null:
		return nil, nil
	case value.Array:
		if e := r.meter.Charge(liquid.WorkIterations, int64(v.Len())); e != nil {
			f := r.exprFinding(x, "for_each", e)
			return nil, &f
		}
		elems := make([]value.Value, 0, v.Len())
		for _, el := range v.All() {
			elems = append(elems, el)
		}
		fans := make([]*Fan, len(elems))
		for i := range fans {
			fans[i] = &Fan{elems: elems, i: i}
		}
		return fans, nil
	default:
		loc := x.at("for_each")
		f := finding.New(CodeForEachType, finding.Subject{Kind: finding.SubjectOutput, Name: x.ID}, &loc,
			finding.Params{"type": finding.String(v.Kind().String())})
		return nil, &f
	}
}

// When evaluates x's when in the evaluation fan, with item and fan_out
// bound in a fan-out (ADR-005 D3 items 4 and 5), and reports whether it
// holds; a declaration without when always holds.
func (r *Renderer) When(x *Exprs, fan *Fan) (bool, *finding.Finding) {
	if x.When == nil {
		return true, nil
	}
	ok, e := liquid.EvalCondition(x.When, r.exprEnv(fan, nil))
	if e != nil {
		f := r.exprFinding(x, "when", e)
		return false, &f
	}
	return ok, nil
}

// Path renders x's path in the evaluation fan, unescaped, and returns the
// rendered string, which is the path exactly (ADR-005 D4 items 1 and 2). Its
// bytes are held to path-length as they are written: a path template that
// writes more is refused there, with limit.exceeded.output.
func (r *Renderer) Path(x *Exprs, fan *Fan) (string, *finding.Finding) {
	pm := &pathMeter{m: r.meter, limit: r.lim.Value(limits.PathLength)}
	var buf bytes.Buffer
	if e := liquid.Render(&buf, x.Path, r.exprEnv(fan, pm)); e != nil {
		if e.Code == liquid.CodeLimit && e.Args["limit"] == limits.Get(limits.PathLength).Name {
			loc := x.at("path")
			f := r.lim.ExceededOutput(limits.PathLength, finding.Subject{Kind: finding.SubjectOutput, Name: x.ID}, loc, 0, false)
			return "", &f
		}
		f := r.exprFinding(x, "path", e)
		return "", &f
	}
	return buf.String(), nil
}

// exprEnv is the environment of a manifest expression in the evaluation
// fan: the globals of ADR-004 D7 item 10's table, with no output, no
// partials and no escaping, under the render's meter, or m when given.
func (r *Renderer) exprEnv(fan *Fan, m liquid.Meter) *liquid.Env {
	if m == nil {
		m = r.meter
	}
	return &liquid.Env{
		Dialect: dialect,
		Globals: r.b.scope(fan, nil),
		Meter:   m,
		Limits: liquid.RenderLimits{
			PartialDepth: int(r.lim.Value(limits.PartialDepth)),
			ValueSize:    r.lim.Value(limits.ValueSize),
			ArraySize:    r.lim.Value(limits.ArraySize),
		},
		UnorderedObjects: r.unordered,
	}
}

// exprFinding maps the error e of evaluating x's field to its finding: the
// template family's code a template body's error would have, or a limit's
// breach, located at the field, its column within the expression
// (ADR-004 D16 item 5).
func (r *Renderer) exprFinding(x *Exprs, field string, e *liquid.Error) finding.Finding {
	loc := x.at(field)
	if e.Code == liquid.CodeLimit {
		l, ok := limits.Lookup(e.Args["limit"])
		if !ok {
			panic("template: an expression breached an unknown limit " + e.Args["limit"])
		}
		return r.lim.ExceededTemplate(l.ID, exprSubject, loc, "")
	}
	return evalFinding(e, exprSubject, loc, nil)
}

// pathMeter is the meter of a path's render: the render's work, and the
// path's own bytes, held to path-length rather than counted as an
// output's.
type pathMeter struct {
	m        *meter
	n, limit int64
}

func (p *pathMeter) Charge(w liquid.Work, n int64) *liquid.Error {
	if w != liquid.WorkOutputBytes {
		return p.m.cmd.Charge(w, n)
	}
	if n > p.limit-p.n {
		return limitError(limits.PathLength)
	}
	p.n += n
	return nil
}

func (p *pathMeter) Used(w liquid.Work) int64 {
	if w == liquid.WorkOutputBytes {
		return p.n
	}
	return p.m.cmd.Used(w)
}

// integer is n as a value of the data model.
func integer(n int) value.Value {
	v, err := value.ParseNumber(strconv.Itoa(n))
	if err != nil {
		panic("template: " + err.Error())
	}
	return liquid.Data(v)
}
