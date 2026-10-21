// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

// Package render is ADR-005 D13's render pipeline: planning, the path
// checks, producing every file's bytes, the destination check and
// emission (ADR-010 Amendment A7 item 2). internal/cli keeps each command
// that runs it: its flags, refusals, help and result records, and the
// choice of where the render goes.
package render

import (
	"io"
	"slices"
	"strings"

	"github.com/kottera/metais/internal/fault"
	"github.com/kottera/metais/internal/finding"
	"github.com/kottera/metais/internal/limits"
	"github.com/kottera/metais/internal/loader"
	"github.com/kottera/metais/internal/template"
)

// Options are one render's: the package, its templates and the bindings
// they see, its limits, and the command's fault set, which the pipeline
// carries to the seams ADR-010 D13 item 2 names.
type Options struct {
	Package   *loader.Package
	Templates *template.Templates
	Bindings  *template.Bindings
	Limits    limits.Set
	Faults    fault.Set
	// Schemas are the output validation schemas, compiled once by a
	// command that renders more than once; when nil, Run compiles them
	// after phase 3 (ADR-005 D11 item 3).
	Schemas *Schemas
	// Fails reports whether findings fail the render: an error, or under
	// --fail-on warning a warning (ADR-006 D2 item 5).
	Fails func([]finding.Finding) bool
	// Only is the path --only selects, when HasOnly (ADR-005 D18).
	Only    string
	HasOnly bool
}

// Emitter is where a render goes, as the command chooses it: Check is
// ADR-005 D13's phase 3 for the top-level files planned, every refusal a
// finding reported with the others before anything is written (D15 item
// 8), or nil for none; Emit is phase 5 for the files selected, in path
// order, or nil for a dry run, which writes nothing (D16).
type Emitter struct {
	Check func(plan []*File) []finding.Finding
	Emit  func(files []*File) Emission
}

// Emission is how phase 5 ended: with its finding when it failed, or with
// a write to standard output the command runs after its report.
type Emission struct {
	Failed *finding.Finding
	Stream func(io.Writer)
}

// Result is a render's: every finding, in the order the phases made them,
// and how far it went.
type Result struct {
	Findings []finding.Finding
	// Produced is whether no finding fails the render; Files are then the
	// top-level files, in path order.
	Produced bool
	Files    []*File
	// Selected is whether --only names a file, or is not given; Written
	// are then the files selected, Files or the one --only names.
	Selected bool
	Written  []*File
	// Emitted is whether the emission ran and did not fail, and Stream
	// its write to standard output, if it has one.
	Emitted bool
	Stream  func(io.Writer)
}

// Run is the render's one entry point, ADR-005 D13's phases in order: it
// plans and checks the paths of every file (phases 1 and 2), runs e's
// phase 3, produces every file (phase 4), and, when no finding fails the
// render, selects what --only names from the whole render, which has run
// and been verified (D18 items 1, 2 and 4), and emits it by e (phase 5).
// It is the only thing that orders the files, sorting the top-level files
// by path once, and the only thing that constructs the renderer, whose
// work the manifest's expressions and the templates share (ADR-009 D11).
func Run(o Options, e Emitter) Result {
	pkg, lim := o.Package, o.Limits
	r := template.NewRenderer(o.Templates, o.Bindings, lim, o.Faults)
	plan, fs := planOutputs(pkg, lim, o.Faults, r)
	if e.Check != nil {
		fs = append(fs, e.Check(plan)...)
	}
	schemas := o.Schemas
	if schemas == nil {
		var sfs []finding.Finding
		schemas, sfs = OutputSchemas(pkg, lim)
		fs = append(fs, sfs...)
	}
	fs = append(fs, produce(&Producing{pkg: pkg, ts: o.Templates, r: r, lim: lim, staging: &Staging{pkg: pkg, lim: lim, schemas: schemas}}, plan)...)
	res := Result{Findings: fs}
	if o.Fails(fs) {
		return res
	}
	slices.SortFunc(plan, func(a, b *File) int { return strings.Compare(a.path, b.path) })
	res.Produced, res.Files = true, plan
	written := plan
	if o.HasOnly {
		i := slices.IndexFunc(plan, func(pf *File) bool { return pf.path == o.Only })
		if i < 0 {
			return res
		}
		written = plan[i : i+1]
	}
	res.Selected, res.Written = true, written
	if e.Emit != nil {
		em := e.Emit(written)
		if em.Failed != nil {
			res.Findings = append(res.Findings, *em.Failed)
			return res
		}
		res.Stream = em.Stream
	}
	res.Emitted = true
	digestAssets(written)
	return res
}

// walk visits every file of tree but a container, depth first in planning
// order, a container's entries in its order (ADR-005 D13 item 6).
func walk(tree []*File, visit func(*File)) {
	for _, pf := range tree {
		if pf.IsContainer() {
			walk(pf.entries, visit)
			continue
		}
		visit(pf)
	}
}
