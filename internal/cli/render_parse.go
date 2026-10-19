// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"strings"

	"github.com/kottera/metais/internal/finding"
	"github.com/kottera/metais/internal/limits"
	"github.com/kottera/metais/internal/loader"
	"github.com/kottera/metais/internal/manifest"
	"github.com/kottera/metais/internal/schema"
	"github.com/kottera/metais/internal/template"
	"github.com/kottera/metais/internal/zipfmt"
)

// renderPlan is ADR-005 D13's phase 4: every template a template output
// reaches, at any depth of containers, is parsed, each once, and every
// parse finding is reported, whether or not the output plans a file; then
// every embed of a raw form that does not exist, whose input and item are
// known, is reported before anything renders (ADR-004 D12 item 3); then
// every planned file whose templates parsed and need no such raw form is
// rendered with r, each to its first error (ADR-004 D16 item 4). The bytes
// of each file that rendered are held in its planned file; a file copied
// from an asset is not read, its size counted against the output limits
// with the bytes of every other file (ADR-005 D13 item 4), and the asset
// tag renders through the plan's placement map (D5 item 6). Then each
// container whose entries all rendered is measured, and held to
// output-size.
func renderPlan(pkg *loader.Package, plan []*planned, ts *template.Templates, r *template.Renderer, lim limits.Set, schemas map[string]*schema.Compiler) []finding.Finding {
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
	parse(pkg.Manifest.Outputs)
	// A mirror's templates, and the placement map, from the plan.
	placed := map[string][]template.Placement{}
	var walk func(tree []*planned)
	walk = func(tree []*planned) {
		for _, pf := range tree {
			switch {
			case pf.isContainer():
				walk(pf.entries)
			case pf.asset != "":
				name := strings.TrimPrefix(pf.asset, template.AssetDir)
				placed[name] = append(placed[name], template.Placement{Tree: pf.in, Path: pf.path})
			case pf.decl.Form == manifest.OutputMirror:
				if _, seen := ready[pf.template]; seen {
					continue
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
		}
	}
	walk(plan)
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
	var render func(tree []*planned)
	render = func(tree []*planned) {
		for _, pf := range tree {
			switch {
			case pf.isContainer():
				render(pf.entries)
				continue
			case pf.tool:
				continue
			case pf.asset != "":
				o := template.Output{ID: pf.owner, Path: pf.path, In: pf.in}
				if f := r.ChargeAsset(o, pf.size); f != nil {
					fs = append(fs, *f)
					pf.failed = true
				}
				continue
			case !ready[pf.template]:
				pf.failed = true
				continue
			}
			out, f := r.Render(template.Output{ID: pf.owner, Template: pf.template, Path: pf.path, Fan: pf.fan, Kind: kindOf(pf.path, pf.decl.Kind), In: pf.in, HTML: pf.decl.Escape == "html"})
			if f != nil {
				fs = append(fs, *f)
				pf.failed = true
				continue
			}
			pf.bytes = out
			fs = append(fs, transform(pkg, pf, schemas, lim)...)
		}
	}
	render(plan)
	for _, pf := range plan {
		if !pf.isContainer() || !measure(pf) {
			continue
		}
		if pf.size > lim.Value(limits.OutputSize) {
			loc := finding.Location{Origin: finding.OriginOutput, File: pf.path}
			fs = append(fs, lim.ExceededOutput(limits.OutputSize, finding.Subject{Kind: finding.SubjectOutput, Name: pf.owner}, loc, pf.size, true))
		}
	}
	return fs
}

// measure measures the container pf, and every container among its
// entries, when every entry rendered: the size and digest of the zip it
// writes, and which of its entries it deflates. It writes the zip to
// nothing but a hash, so no container is held whole in memory.
func measure(pf *planned) bool {
	if len(pf.entries) > zipfmt.MaxEntries {
		// Refused in planning (D8 item 8); a zip cannot hold them.
		pf.failed = true
		return false
	}
	for _, e := range pf.entries {
		if e.failed || e.isContainer() && !measure(e) {
			pf.failed = true
			return false
		}
	}
	h := sha256.New()
	n := &byteCount{}
	written, err := writeContainer(io.MultiWriter(h, n), pf)
	if err != nil {
		panic("cli: a planned container does not write: " + err.Error())
	}
	for i, w := range written {
		pf.entries[i].deflated = w.Deflated
	}
	pf.size, pf.digest = n.n, "sha256:"+hex.EncodeToString(h.Sum(nil))
	return true
}

// byteCount counts the bytes written to it.
type byteCount struct{ n int64 }

func (c *byteCount) Write(b []byte) (int, error) {
	c.n += int64(len(b))
	return len(b), nil
}
