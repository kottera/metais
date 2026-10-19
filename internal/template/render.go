// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package template

import (
	"bytes"

	"github.com/kottera/metais/internal/fault"
	"github.com/kottera/metais/internal/finding"
	"github.com/kottera/metais/internal/limits"
	"github.com/kottera/metais/internal/liquid"
)

// Output is one file a render writes from a template: the declaration's
// id, the template's name beneath templates/, the file's path relative to
// the root of its tree, the evaluation of a fan-out it is, or nil, its
// kind (ADR-005 D7), from which the provenance tag chooses its style
// (ADR-004 D14 item 5). For a container's entry, In is the path of the
// container, itself within its own tree's containers, joined by "/", and
// Path is the entry's path within the container (ADR-004 D7 item 6).
// In is also the tree the file is written in, as a Placement names it:
// "" for the output directory.
// HTML is whether its escape is html (ADR-005 D2 item 4).
type Output struct {
	ID       string
	Template string
	Path     string
	Fan      *Fan
	Kind     string
	In       string
	HTML     bool
}

// file is the path a finding about o names: its path, or for a
// container's entry the container's path and its own (ADR-005 D11 item 4).
func (o Output) file() string {
	if o.In == "" {
		return o.Path
	}
	return o.In + "/" + o.Path
}

// Renderer renders one command's outputs. Their work is counted together,
// against render-work, loop-iterations, assign-bytes and output-total, and
// each output's bytes against output-size (ADR-009 D3, D11).
type Renderer struct {
	ts        *Templates
	b         *Bindings
	lim       limits.Set
	meter     *meter
	unordered bool
	// dialect is the dialect a template renders in: Dialect's, its tags
	// rendering through this renderer. out is the output being rendered.
	dialect *liquid.Dialect
	out     Output
	// assets is where the render writes each asset, by its name beneath
	// assets/ (ADR-005 D5 item 6).
	assets map[string][]Placement
}

// NewRenderer returns the renderer of one command, over the package's
// templates ts and the globals b, under lim. faults is the invocation's
// test-only fault set (ADR-010 D13 item 2), which the tool builds zero; a
// render consults only fault.MapOrder.
func NewRenderer(ts *Templates, b *Bindings, lim limits.Set, faults fault.Set) *Renderer {
	r := &Renderer{
		ts: ts, b: b, lim: lim,
		meter: &meter{
			cmd: liquid.NewMeter(liquid.MeterLimits{
				RenderWork:     lim.Value(limits.RenderWork),
				AssignBytes:    lim.Value(limits.AssignBytes),
				LoopIterations: lim.Value(limits.LoopIterations),
			}),
			outLimit:   lim.Value(limits.OutputSize),
			totalLimit: lim.Value(limits.OutputTotal),
		},
		// The MapOrder seam (ADR-010 D13 item 2, Amendment A2).
		unordered: faults.Has(fault.MapOrder),
	}
	r.dialect = newDialect(r)
	return r
}

// PlaceAssets gives the render the asset placement map (ADR-005 D5 item
// 6): every file the render writes from an asset, by the asset's name
// beneath assets/. An asset tag's path mode resolves through it (ADR-004
// D13 item 2); a render that writes no asset gives none.
func (r *Renderer) PlaceAssets(m map[string][]Placement) { r.assets = m }

// ChargeAsset counts the size of a file the render copies from an asset,
// the output o, against output-size and output-total with the bytes of
// every other output (ADR-005 D13 item 4, ADR-009 D3), and returns the
// finding of the limit it breaches, if any. The asset's bytes are not
// read.
func (r *Renderer) ChargeAsset(o Output, size int64) *finding.Finding {
	r.meter.out = 0
	if e := r.meter.Charge(liquid.WorkOutputBytes, size); e != nil {
		l, _ := limits.Lookup(e.Args["limit"])
		f := r.lim.ExceededOutput(l.ID, finding.Subject{Kind: finding.SubjectOutput, Name: o.ID},
			finding.Location{Origin: finding.OriginOutput, File: o.Path}, 0, false)
		return &f
	}
	return nil
}

// Render renders o's template, every template it reaches having parsed
// without a finding, and returns its bytes, or nil and the finding of its
// first error (ADR-004 D16 items 1, 2 and 4). Nothing of an output with an
// error is returned.
func (r *Renderer) Render(o Output) ([]byte, *finding.Finding) {
	t, fs := r.ts.Parse(o.Template)
	if t == nil {
		panic("template: render of " + o.Template + ", whose parse failed: " + string(fs[0].Code()))
	}
	if o.Kind == "" {
		panic("template: render of " + o.Path + ", which has no kind")
	}
	r.meter.out, r.out = 0, o
	env := &liquid.Env{
		Dialect:  r.dialect,
		Globals:  r.b.scope(o.Fan, &o.Path),
		Partials: partials{r.ts},
		Meter:    r.meter,
		Limits: liquid.RenderLimits{
			PartialDepth: int(r.lim.Value(limits.PartialDepth)),
			ValueSize:    r.lim.Value(limits.ValueSize),
			ArraySize:    r.lim.Value(limits.ArraySize),
		},
		UnorderedObjects: r.unordered,
	}
	if o.HTML {
		// Every value an expression writes is escaped unless safe, in the
		// template and every partial it renders (ADR-004 D8 item 5, D9).
		env.Escape = liquid.EscapeHTML
	}
	var buf bytes.Buffer
	if e := liquid.Render(&buf, t, env); e != nil {
		f := r.renderFinding(o, e)
		return nil, &f
	}
	return buf.Bytes(), nil
}

// meter is a render's Meter: the command's work, counted by the engine's
// own meter, and the bytes of the output being rendered and of every
// output, which it counts itself (ADR-009 D3's output-size and
// output-total).
type meter struct {
	cmd                  liquid.Meter
	out, total           int64
	outLimit, totalLimit int64
	totalOver            bool
}

func (m *meter) Charge(w liquid.Work, n int64) *liquid.Error {
	if w != liquid.WorkOutputBytes {
		return m.cmd.Charge(w, n)
	}
	switch {
	case m.totalOver:
		return limitError(limits.OutputTotal)
	case n > m.outLimit-m.out:
		return limitError(limits.OutputSize)
	case n > m.totalLimit-m.total:
		// Every later output's first byte is refused too.
		m.totalOver = true
		return limitError(limits.OutputTotal)
	}
	m.out += n
	m.total += n
	return nil
}

func (m *meter) Used(w liquid.Work) int64 {
	if w == liquid.WorkOutputBytes {
		return m.total
	}
	return m.cmd.Used(w)
}

// limitError is the engine's error for a breach of id.
func limitError(id limits.ID) *liquid.Error {
	return &liquid.Error{Code: liquid.CodeLimit, Args: map[string]string{"limit": limits.Get(id).Name}}
}
