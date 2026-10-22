// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package render

import (
	"bytes"
	"io"

	"metais.org/metais/internal/finding"
	"metais.org/metais/internal/template"
	"metais.org/metais/internal/zipfmt"
)

// rendered is the producer of a template output's file and a mirror's
// (ADR-005 D4, D6): its bytes are its template's, rendered, then
// transformed by its stages.
type rendered struct{}

// Record is a template's, which has a kind (D7).
func (rendered) Record() (string, bool) { return "template", true }

// Reach is the template f renders from.
func (rendered) Reach(f *File) []string { return []string{f.template} }

// Produce renders f, when its template is ready, to its first error
// (ADR-004 D16 item 4), runs its stages, and measures its bytes.
func (rendered) Produce(c *Producing, f *File) []finding.Finding {
	fs := renderFile(c, f)
	f.size, f.digest = measureBytes(f.bytes)
	return fs
}

// renderFile renders f and runs its stages, or marks it failed.
func renderFile(c *Producing, f *File) []finding.Finding {
	if !c.ready[f.template] {
		f.failed = true
		return nil
	}
	out, fd := c.r.Render(template.Output{ID: f.owner, Template: f.template, Path: f.path, Fan: f.fan,
		Kind: kindOf(f.path, f.decl.Kind), In: f.in, HTML: f.decl.Escape == "html"})
	if fd != nil {
		f.failed = true
		return []finding.Finding{*fd}
	}
	f.bytes = out
	return runStages(c.staging, f)
}

func (rendered) Open(f *File) (io.Reader, func(error) error) {
	return bytes.NewReader(f.bytes), func(error) error { return nil }
}

func (rendered) Method(f *File) zipfmt.Method { return declaredMethod(f) }
