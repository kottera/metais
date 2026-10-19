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
// id, the template's name beneath templates/, and the file's path relative
// to the root of its tree.
type Output struct {
	ID       string
	Template string
	Path     string
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
}

// NewRenderer returns the renderer of one command, over the package's
// templates ts and the globals b, under lim. faults is the invocation's
// test-only fault set (ADR-010 D13 item 2), which the tool builds zero; a
// render consults only fault.MapOrder.
func NewRenderer(ts *Templates, b *Bindings, lim limits.Set, faults fault.Set) *Renderer {
	return &Renderer{
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
	r.meter.out = 0
	env := &liquid.Env{
		Dialect:  dialect,
		Globals:  r.b.globals(o.Path),
		Partials: partials{r.ts},
		Meter:    r.meter,
		Limits: liquid.RenderLimits{
			PartialDepth: int(r.lim.Value(limits.PartialDepth)),
			ValueSize:    r.lim.Value(limits.ValueSize),
			ArraySize:    r.lim.Value(limits.ArraySize),
		},
		UnorderedObjects: r.unordered,
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
