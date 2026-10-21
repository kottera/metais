// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package render

import (
	"cmp"

	"github.com/kottera/metais/internal/fault"
	"github.com/kottera/metais/internal/finding"
	"github.com/kottera/metais/internal/limits"
	"github.com/kottera/metais/internal/loader"
	"github.com/kottera/metais/internal/manifest"
	"github.com/kottera/metais/internal/template"
)

// joinIn is the path p within the tree of the container at in.
func joinIn(in, p string) string {
	if in == "" {
		return p
	}
	return in + "/" + p
}

// planOutputs is ADR-005 D13's phases 1 and 2 for the package's
// declarations. Each declaration's for_each gives its evaluations, each
// evaluation's when decides whether it writes a file, and its path is
// rendered, or defaulted (D3, D4); a declaration whose expressions fail
// contributes their finding and no file, and planning goes on with the
// next. A container is planned as a tree of its own, its entries in each
// of its evaluations, to container-depth, and held to its profile (D8,
// D9). Every path is held to the portable-path predicate, each alone and
// every path of its tree together (D12), and the count of planned files,
// every evaluation's and every container entry's, to the outputs limit
// (ADR-009 D3).
func planOutputs(pkg *loader.Package, lim limits.Set, faults fault.Set, r *template.Renderer) ([]*File, []finding.Finding) {
	p := &Planner{pkg: pkg, lim: lim, faults: faults, r: r, counted: true,
		entries: pkg.Tree.List(), explicit: ExplicitTemplates(pkg.Manifest.Outputs)}
	plan := p.tree(pkg.Manifest.Outputs, "", "", 0, nil)
	if n, bound := int64(len(p.all)), lim.Value(limits.Outputs); n > bound {
		over := p.all[bound]
		if over.fan != nil {
			// The files the data fans out to take the plan beyond the
			// bound: the render's, not the package's (ADR-009 D19).
			subj := finding.Subject{Kind: finding.SubjectOutput, Name: over.owner}
			p.fs = append(p.fs, lim.ExceededOutput(limits.Outputs, subj, p.at(over.decl.Pointer), n, p.counted))
		} else {
			p.fs = append(p.fs, lim.Breached(limits.Outputs, p.at(over.decl.Pointer), n, p.counted))
		}
	}
	return plan, p.fs
}

// Planner is one render's phases 1 and 2. It evaluates every
// declaration's when and for_each, renders its paths and holds the
// outputs limit; each declaration's form expands its evaluations into
// files (ADR-010 A7 item 4).
type Planner struct {
	pkg    *loader.Package
	lim    limits.Set
	faults fault.Set
	r      *template.Renderer
	fs     []finding.Finding
	all    []*File // every planned file, at every depth, in planning order
	// counted is cleared when a fan-out is left unevaluated because the
	// plan is already beyond outputs.
	counted bool
	// entries are the package's, and explicit the templates a template
	// output names, which no mirror renders (ADR-005 D6 item 3).
	entries  []string
	explicit map[string]bool
}

// at is the location of the manifest's value at ptr.
func (p *Planner) at(ptr string) finding.Location { return manifestAt(p.pkg, ptr) }

// beyond reports whether the plan already holds more files than outputs.
func (p *Planner) beyond() bool { return int64(len(p.all)) > p.lim.Value(limits.Outputs) }

// tree plans decls, the declarations of one tree: the output directory's,
// or the entries of the container at in, which depth containers enclose,
// in the enclosing container's evaluation fan. owner is the id an entry
// with none of its own is named by. The output directory's paths are
// held to D12 together here, and a container's by container.
func (p *Planner) tree(decls []manifest.Output, owner, in string, depth int, fan *template.Fan) []*File {
	var tree []*File
	for _, d := range decls {
		id := cmp.Or(d.ID, owner)
		x, xfs := template.ParseExprs(d, id, p.at, p.lim)
		if len(xfs) > 0 {
			p.fs = append(p.fs, xfs...)
			continue
		}
		if x.ForEach != nil && p.beyond() {
			p.counted = false
			continue
		}
		if d.Form == manifest.OutputContainer && int64(depth) >= p.lim.Value(limits.ContainerDepth) {
			// D8 item 4: a container nested deeper than container-depth
			// is the package's own breach (ADR-009 D19), and plans
			// nothing.
			loc := p.at(d.Pointer + "/container")
			p.fs = append(p.fs, p.lim.Breached(limits.ContainerDepth, loc, int64(depth)+1, true))
			continue
		}
		tree = append(tree, p.declaration(d, x, id, in, depth, fan)...)
	}
	if in == "" {
		p.checkPaths(tree)
	}
	return tree
}

// plan records files as planned, in planning order, where the outputs
// limit counts them, and returns them.
func (p *Planner) plan(files ...*File) []*File {
	p.all = append(p.all, files...)
	return files
}

// declaration plans the files of the declaration d, whose expressions are
// x: one per evaluation whose when holds, in the order of the for_each
// array, or in the enclosing container's evaluation fan, each with its
// path rendered, or its form's default path when d gives none (ADR-005
// D3, D4), and each expanded by d's form. Before each evaluation but the
// first, a plan already beyond outputs stops the fan-out. The first error
// of an expression is the declaration's finding, and it plans no file.
func (p *Planner) declaration(d manifest.Output, x *template.Exprs, owner, in string, depth int, fan *template.Fan) []*File {
	form := formOf(d.Form)
	fans := []*template.Fan{fan}
	if x.ForEach != nil {
		var f *finding.Finding
		if fans, f = p.r.Fans(x); f != nil {
			p.fs = append(p.fs, *f)
			return nil
		}
	}
	mark, marks := len(p.all), len(p.fs)
	var files []*File
	for i, fan := range fans {
		if i > 0 && p.beyond() {
			p.counted = false
			break
		}
		ok, f := p.r.When(x, fan)
		if f == nil && ok {
			pf := &File{decl: d, fan: fan, ptr: d.Pointer + "/path", literal: x.Literal, owner: owner, in: in}
			if x.Path == nil {
				pf.path, pf.ptr, pf.literal = form.DefaultPath(d), d.Pointer+"/"+string(d.Form), true
			} else {
				pf.path, f = p.r.Path(x, fan)
			}
			if f == nil {
				files = append(files, form.Expand(p, pf, depth)...)
			}
		}
		if f != nil {
			// The declaration plans nothing: not the files of its earlier
			// evaluations, nor what their containers found.
			p.all, p.fs = p.all[:mark], append(p.fs[:marks], *f)
			return nil
		}
	}
	return files
}

// TreePrefix is the prefix of every entry beneath the directory rel of the
// layout directory dir, "." naming dir itself.
func TreePrefix(dir, rel string) string {
	if rel == "." {
		return dir
	}
	return dir + rel + "/"
}

// Beneath is the path rel beneath the destination directory dest, "."
// being the root of the tree. The paths are joined as written, never
// cleaned: a path is exactly what the manifest gives (ADR-005 D4 item 2).
func Beneath(dest, rel string) string {
	if dest == "." {
		return rel
	}
	return dest + "/" + rel
}

// ptrTo is a pointer to a copy of l.
func ptrTo(l finding.Location) *finding.Location { return &l }

// manifestAt is the location of the manifest value at ptr, with its line
// and column when the manifest has them.
func manifestAt(pkg *loader.Package, ptr string) finding.Location {
	l := finding.Location{Origin: finding.OriginPackage, File: manifest.ManifestFile}.WithPointer(ptr)
	if pkg.Positions != nil {
		if pos, ok := pkg.Positions.Value(ptr); ok {
			l = l.At(pos)
		}
	}
	return l
}
