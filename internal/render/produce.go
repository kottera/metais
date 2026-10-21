// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package render

import (
	"crypto/sha256"
	"encoding/hex"

	"github.com/kottera/metais/internal/finding"
	"github.com/kottera/metais/internal/limits"
	"github.com/kottera/metais/internal/loader"
	"github.com/kottera/metais/internal/manifest"
	"github.com/kottera/metais/internal/template"
)

// Producing is what phase 4 needs: the package, its templates, the
// render's renderer and limits, the stages' own state, and what the
// producers share.
type Producing struct {
	pkg     *loader.Package
	ts      *template.Templates
	r       *template.Renderer
	lim     limits.Set
	staging *Staging
	// ready is whether each template a file renders from parsed, with
	// every template it reaches, and needs no raw form that does not
	// exist.
	ready map[string]bool
	// measured are the findings of measuring the top-level containers,
	// reported once every file has been produced.
	measured []finding.Finding
}

// produce is ADR-005 D13's phase 4: every template a template output
// reaches, at any depth of containers, is parsed, each once, and every
// parse finding is reported, whether or not the output plans a file; then
// the templates every other planned file reaches, by its producer; then
// every embed of a raw form that does not exist, whose input and item are
// known, is reported before anything renders (ADR-004 D12 item 3). Then
// each planned file is produced by its producer, depth first in planning
// order: rendered with c's renderer, each to its first error (ADR-004 D16
// item 4), and its stages run, when its templates parsed and need no such
// raw form; copied from an asset, which is not read, its size counted
// against the output limits with the bytes of every other file (ADR-005
// D13 item 4), and the asset tag renders through the plan's placement map
// (D5 item 6); or a container, measured once its entries all are, and
// held to output-size after every file.
func produce(c *Producing, plan []*File) []finding.Finding {
	ts, r := c.ts, c.r
	var fs []finding.Finding
	reported := map[string]bool{}
	ready := map[string]bool{}
	var sources []string // ready's keys, in the order they were reached
	reach := func(source string) {
		if _, seen := ready[source]; seen {
			return
		}
		ready[source] = true
		sources = append(sources, source)
		for _, name := range ts.Reach(source) {
			_, pfs := ts.Parse(name)
			if len(pfs) > 0 {
				ready[source] = false
			}
			if !reported[name] {
				reported[name] = true
				fs = append(fs, pfs...)
			}
		}
	}
	var parse func(decls []manifest.Output)
	parse = func(decls []manifest.Output) {
		for _, d := range decls {
			if d.Container != nil {
				parse(d.Container.Entries)
			}
			if d.Form == manifest.OutputTemplate {
				reach(d.Source)
			}
		}
	}
	parse(c.pkg.Manifest.Outputs)
	// A mirror's templates, and the placement map, from the plan.
	placed := map[string][]template.Placement{}
	walk(plan, func(pf *File) {
		for _, source := range pf.producer.Reach(pf) {
			reach(source)
		}
		if name, at, ok := placement(pf); ok {
			placed[name] = append(placed[name], at)
		}
	})
	r.PlaceAssets(placed)
	unavailable := map[string]bool{}
	for _, source := range sources {
		if !ready[source] {
			continue
		}
		for _, name := range ts.Reach(source) {
			if _, seen := unavailable[name]; !seen {
				ufs := r.Unavailable(name)
				unavailable[name] = len(ufs) > 0
				fs = append(fs, ufs...)
			}
			if unavailable[name] {
				ready[source] = false
			}
		}
	}
	c.ready = ready
	for _, pf := range plan {
		fs = append(fs, pf.producer.Produce(c, pf)...)
	}
	return append(fs, c.measured...)
}

// measureBytes is the size and digest of b.
func measureBytes(b []byte) (int64, string) {
	sum := sha256.Sum256(b)
	return int64(len(b)), "sha256:" + hex.EncodeToString(sum[:])
}
