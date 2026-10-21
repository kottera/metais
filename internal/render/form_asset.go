// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package render

import (
	"strings"

	"github.com/kottera/metais/internal/manifest"
	"github.com/kottera/metais/internal/template"
)

// assetForm is an asset declaration's (ADR-005 D5).
type assetForm struct{}

// DefaultPath is the asset's own path beneath assets/.
func (assetForm) DefaultPath(d manifest.Output) string { return d.Source }

// Expand is the files of the asset declaration's evaluation at, whose path
// names their destination (ADR-005 D5): the file assets/<path>, copied to
// the path; or every file beneath the directory assets/<path>, "."
// naming assets/ itself, each at its path relative to that directory
// beneath the destination, "." being the root of the tree. Each file's
// size is the entry's, and its bytes are not read. The loader has refused
// an asset that names no entry. Each is copied from the package.
func (assetForm) Expand(p *Planner, at *File, _ int) []*File {
	src := at.decl.Source
	file := func(entry, path string) *File {
		info, err := p.pkg.Tree.Stat(entry)
		if err != nil {
			panic("render: an entry the plan names has no Stat: " + entry)
		}
		pf := *at
		pf.path, pf.asset, pf.tree, pf.size = path, entry, p.pkg.Tree, info.Size
		pf.producer = producerOf(byCopying)
		return &pf
	}
	if _, err := p.pkg.Tree.Stat(template.AssetDir + src); err == nil {
		return p.plan(file(template.AssetDir+src, at.path))
	}
	var out []*File
	for _, e := range p.entries {
		if rel, ok := strings.CutPrefix(e, TreePrefix(template.AssetDir, src)); ok {
			out = append(out, file(e, Beneath(at.path, rel)))
		}
	}
	return p.plan(out...)
}
