// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package render

import (
	"strings"

	"github.com/kottera/metais/internal/finding"
	"github.com/kottera/metais/internal/limits"
	"github.com/kottera/metais/internal/loader"
	"github.com/kottera/metais/internal/manifest"
	"github.com/kottera/metais/internal/template"
)

// Producing is what phase 4 needs: the package, its templates, the
// render's renderer and limits, and the stages' own state.
type Producing struct {
	pkg     *loader.Package
	ts      *template.Templates
	r       *template.Renderer
	lim     limits.Set
	staging *Staging
}

// produce is ADR-005 D13's phase 4: every template a template output
// reaches, at any depth of containers, is parsed, each once, and every
// parse finding is reported, whether or not the output plans a file; then
// every embed of a raw form that does not exist, whose input and item are
// known, is reported before anything renders (ADR-004 D12 item 3); then
// every planned file whose templates parsed and need no such raw form is
// rendered with c's renderer, each to its first error (ADR-004 D16 item
// 4), and its stages run. The bytes of each file that rendered are held in
// its planned file; a file copied from an asset is not read, its size
// counted against the output limits with the bytes of every other file
// (ADR-005 D13 item 4), and the asset tag renders through the plan's
// placement map (D5 item 6). Then each container whose entries all
// rendered is measured, and held to output-size.
func produce(c *Producing, plan []*File) []finding.Finding {
	ts, r, lim := c.ts, c.r, c.lim
	var fs []finding.Finding
	reported := map[string]bool{}
	ready := map[string]bool{}
	var sources []string // ready's keys, in manifest order
	var parse func(decls []manifest.Output)
	parse = func(decls []manifest.Output) {
		for _, d := range decls {
			if d.Container != nil {
				parse(d.Container.Entries)
			}
			if _, seen := ready[d.Source]; seen || d.Form != manifest.OutputTemplate {
				continue
			}
			ready[d.Source] = true
			sources = append(sources, d.Source)
			for _, name := range ts.Reach(d.Source) {
				_, pfs := ts.Parse(name)
				if len(pfs) > 0 {
					ready[d.Source] = false
				}
				if !reported[name] {
					reported[name] = true
					fs = append(fs, pfs...)
				}
			}
		}
	}
	parse(c.pkg.Manifest.Outputs)
	// A mirror's templates, and the placement map, from the plan.
	placed := map[string][]template.Placement{}
	walk(plan, func(pf *File) {
		switch {
		case pf.asset != "":
			name := strings.TrimPrefix(pf.asset, template.AssetDir)
			placed[name] = append(placed[name], template.Placement{Tree: pf.in, Path: pf.path})
		case pf.decl.Form == manifest.OutputMirror:
			if _, seen := ready[pf.template]; seen {
				return
			}
			ready[pf.template] = true
			sources = append(sources, pf.template)
			for _, name := range ts.Reach(pf.template) {
				_, pfs := ts.Parse(name)
				if len(pfs) > 0 {
					ready[pf.template] = false
				}
				if !reported[name] {
					reported[name] = true
					fs = append(fs, pfs...)
				}
			}
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
	walk(plan, func(pf *File) {
		switch {
		case pf.tool:
			return
		case pf.asset != "":
			o := template.Output{ID: pf.owner, Path: pf.path, In: pf.in}
			if f := r.ChargeAsset(o, pf.size); f != nil {
				fs = append(fs, *f)
				pf.failed = true
			}
			return
		case !ready[pf.template]:
			pf.failed = true
			return
		}
		out, f := r.Render(template.Output{ID: pf.owner, Template: pf.template, Path: pf.path, Fan: pf.fan, Kind: kindOf(pf.path, pf.decl.Kind), In: pf.in, HTML: pf.decl.Escape == "html"})
		if f != nil {
			fs = append(fs, *f)
			pf.failed = true
			return
		}
		pf.bytes = out
		fs = append(fs, runStages(c.staging, pf)...)
	})
	for _, pf := range plan {
		if !pf.IsContainer() || !measure(pf) {
			continue
		}
		if pf.size > lim.Value(limits.OutputSize) {
			loc := finding.Location{Origin: finding.OriginOutput, File: pf.path}
			fs = append(fs, lim.ExceededOutput(limits.OutputSize, finding.Subject{Kind: finding.SubjectOutput, Name: pf.owner}, loc, pf.size, true))
		}
	}
	return fs
}
