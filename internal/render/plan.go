// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package render

import (
	"cmp"
	pathpkg "path"
	"slices"
	"strings"

	"github.com/kottera/metais/internal/fault"
	"github.com/kottera/metais/internal/finding"
	"github.com/kottera/metais/internal/limits"
	"github.com/kottera/metais/internal/loader"
	"github.com/kottera/metais/internal/manifest"
	"github.com/kottera/metais/internal/template"
	"github.com/kottera/metais/internal/zipfmt"
)

// joinIn is the path p within the tree of the container at in.
func joinIn(in, p string) string {
	if in == "" {
		return p
	}
	return in + "/" + p
}

// PlanOutputs is ADR-005 D13's phases 1 and 2 for the package's
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
func PlanOutputs(pkg *loader.Package, lim limits.Set, faults fault.Set, r *template.Renderer) ([]*File, []finding.Finding) {
	p := &planner{pkg: pkg, lim: lim, faults: faults, r: r, counted: true,
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

// planner is one render's phases 1 and 2.
type planner struct {
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
func (p *planner) at(ptr string) finding.Location { return manifestAt(p.pkg, ptr) }

// beyond reports whether the plan already holds more files than outputs.
func (p *planner) beyond() bool { return int64(len(p.all)) > p.lim.Value(limits.Outputs) }

// tree plans decls, the declarations of one tree: the output directory's,
// or the entries of the container at in, which depth containers enclose,
// in the enclosing container's evaluation fan. owner is the id an entry
// with none of its own is named by. The output directory's paths are
// held to D12 together here, and a container's by container.
func (p *planner) tree(decls []manifest.Output, owner, in string, depth int, fan *template.Fan) []*File {
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

// declaration plans the files of the declaration d, whose expressions are
// x: one per evaluation whose when holds, in the order of the for_each
// array, or in the enclosing container's evaluation fan, each with its
// path rendered, or the default path when d gives none (ADR-005 D3, D4),
// and each container's entries; an asset's or a mirror's path names the
// destination of the files it expands into (D5, D6). Before each evaluation but the first, a
// plan already beyond outputs stops the fan-out. The first error of an
// expression is the declaration's finding, and it plans no file.
func (p *planner) declaration(d manifest.Output, x *template.Exprs, owner, in string, depth int, fan *template.Fan) []*File {
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
			pf := &File{decl: d, fan: fan, ptr: d.Pointer + "/path", literal: x.Literal, owner: owner, in: in, template: d.Source}
			if x.Path == nil {
				pf.path, pf.ptr, pf.literal = d.Source, d.Pointer+"/"+string(d.Form), true
				if d.Form == manifest.OutputTemplate {
					pf.path = strings.TrimSuffix(d.Source, ".liquid")
				}
			} else {
				pf.path, f = p.r.Path(x, fan)
			}
			if f == nil {
				switch d.Form {
				case manifest.OutputAsset:
					expanded := p.asset(pf)
					p.all, files = append(p.all, expanded...), append(files, expanded...)
				case manifest.OutputMirror:
					expanded := p.mirror(pf)
					p.all, files = append(p.all, expanded...), append(files, expanded...)
				default:
					p.all = append(p.all, pf)
					if d.Form == manifest.OutputContainer {
						pf.entries = p.container(pf, depth+1)
					}
					files = append(files, pf)
				}
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

// asset is the files of the asset declaration's evaluation at, whose path
// names their destination (ADR-005 D5): the file assets/<path>, copied to
// the path; or every file beneath the directory assets/<path>, "."
// naming assets/ itself, each at its path relative to that directory
// beneath the destination, "." being the root of the tree. Each file's
// size is the entry's, and its bytes are not read. The loader has refused
// an asset that names no entry.
func (p *planner) asset(at *File) []*File {
	src := at.decl.Source
	file := func(entry, path string) *File {
		info, err := p.pkg.Tree.Stat(entry)
		if err != nil {
			panic("render: an entry the plan names has no Stat: " + entry)
		}
		pf := *at
		pf.path, pf.template, pf.asset, pf.tree, pf.size = path, "", entry, p.pkg.Tree, info.Size
		return &pf
	}
	if _, err := p.pkg.Tree.Stat(template.AssetDir + src); err == nil {
		return []*File{file(template.AssetDir+src, at.path)}
	}
	var out []*File
	for _, e := range p.entries {
		if rel, ok := strings.CutPrefix(e, TreePrefix(template.AssetDir, src)); ok {
			out = append(out, file(e, Beneath(at.path, rel)))
		}
	}
	return out
}

// mirror is the files of the mirror's evaluation at, whose path names
// their destination (ADR-005 D6): every template beneath templates/<dir>,
// "." naming templates/ itself, in byte order, but one with a name
// beginning with _ at any depth beneath the mirror's root and one a
// template field names anywhere in the manifest, each rendered to its path
// relative to the mirror's root, with a final .liquid removed, beneath the
// destination. A template named exactly .liquid would be written with an
// empty name, and is refused; a mirror that yields no file is a warning.
func (p *planner) mirror(at *File) []*File {
	d := at.decl
	var out []*File
	refused := false
	subj := finding.Subject{Kind: finding.SubjectOutput, Name: at.owner}
	loc := p.at(d.Pointer + "/mirror")
	for _, e := range p.entries {
		rel, ok := strings.CutPrefix(e, TreePrefix(template.Dir, d.Source))
		name := strings.TrimPrefix(e, template.Dir)
		if !ok || p.explicit[name] || slices.ContainsFunc(strings.Split(rel, "/"), func(seg string) bool { return strings.HasPrefix(seg, "_") }) {
			continue
		}
		if pathpkg.Base(rel) == ".liquid" {
			refused = true
			p.fs = append(p.fs, finding.New(CodeMirrorEmptyName, subj, &loc, finding.Params{"template": finding.String(name)}))
			continue
		}
		pf := *at
		pf.path, pf.template = Beneath(at.path, strings.TrimSuffix(rel, ".liquid")), name
		out = append(out, &pf)
	}
	if len(out) == 0 && !refused {
		p.fs = append(p.fs, finding.New(CodeMirrorEmpty, subj, &loc, finding.Params{"directory": finding.String(d.Source)}))
	}
	return out
}

// The codes of an output declaration a render reports (ADR-005's
// interfaces name the family output.declaration).
const (
	CodeMirrorEmptyName finding.Code = "output.declaration.empty_name"
	CodeMirrorEmpty     finding.Code = "output.declaration.empty_mirror"
)

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

// ExplicitTemplates is every template a template output names, at any
// depth of a container: no mirror renders one (ADR-005 D6 item 3).
func ExplicitTemplates(decls []manifest.Output) map[string]bool {
	out := map[string]bool{}
	var walk func([]manifest.Output)
	walk = func(ds []manifest.Output) {
		for _, d := range ds {
			switch d.Form {
			case manifest.OutputTemplate:
				out[d.Source] = true
			case manifest.OutputContainer:
				walk(d.Container.Entries)
			}
		}
	}
	walk(decls)
	return out
}

// container plans the entries of the container pf, the depth-th
// container on its chain, in pf's evaluation: one tree, held to D12
// within it and to its profile (D8 items 3 and 8, D9), in the container's
// order (D8 item 6).
func (p *planner) container(pf *File, depth int) []*File {
	c := pf.decl.Container
	in := joinIn(pf.in, pf.path)
	var entries []*File
	if m, ok := c.ToolMimetype(); ok {
		// D9 items 2 to 4: the profile's own mimetype entry, which takes
		// its place in the tree like any other.
		tool := &File{decl: pf.decl, path: manifest.MimetypeEntry, ptr: pf.decl.Pointer + "/container/profile",
			literal: true, fan: pf.fan, owner: pf.owner, in: in, bytes: []byte(m), tool: true}
		p.all = append(p.all, tool)
		entries = append(entries, tool)
	}
	for _, e := range p.tree(c.Entries, pf.owner, in, depth, pf.fan) {
		subj := finding.Subject{Kind: finding.SubjectOutput, Name: e.owner}
		params := pathParams(e)
		params["profile"] = finding.String(c.Profile)
		switch c.CheckEntry(e.path) {
		case manifest.EntryReserved:
			p.fs = append(p.fs, finding.New(finding.CodeContainerReservedEntry, subj, ptrTo(p.at(e.ptr)), params))
			continue
		case manifest.EntryNonASCII:
			p.fs = append(p.fs, finding.New(finding.CodeContainerNonASCII, subj, ptrTo(p.at(e.ptr)), params))
		case manifest.EntrySpace:
			p.fs = append(p.fs, finding.New(finding.CodeContainerSpace, subj, ptrTo(p.at(e.ptr)), params))
		}
		entries = append(entries, e)
	}
	p.checkPaths(entries)
	subj := finding.Subject{Kind: finding.SubjectOutput, Name: pf.owner}
	if req := c.RequiredEntry(); req != "" && !slices.ContainsFunc(entries, func(e *File) bool { return e.path == req }) {
		params := finding.Params{"entry": finding.String(req), "profile": finding.String(c.Profile)}
		if i := pf.Item(); i != nil {
			params["item"] = finding.Int(int64(*i))
		}
		p.fs = append(p.fs, finding.New(finding.CodeContainerMissingEntry, subj, ptrTo(p.at(pf.decl.Pointer+"/container/profile")), params))
	}
	if len(entries) > zipfmt.MaxEntries {
		p.fs = append(p.fs, finding.New(finding.CodeContainerTooManyEntries, subj,
			ptrTo(p.at(pf.decl.Pointer+"/container/entries")), finding.Params{"count": finding.Int(int64(len(entries)))}))
	}
	return containerOrder(c, entries)
}

// ptrTo is a pointer to a copy of l.
func ptrTo(l finding.Location) *finding.Location { return &l }

// containerOrder is ADR-005 D8 item 6's order of a container's entries:
// the profile's first entries, the profile's own mimetype before the
// author's [Content_Types].xml; then the files of every first: true
// declaration, in declaration order, each declaration's in byte order of
// their paths; then every other file in byte order. A file appears once,
// at its earliest place.
func containerOrder(c *manifest.Container, entries []*File) []*File {
	sorted := slices.SortedFunc(slices.Values(entries), func(a, b *File) int { return strings.Compare(a.path, b.path) })
	out := make([]*File, 0, len(entries))
	placed := map[*File]bool{}
	place := func(keep func(e *File) bool) {
		for _, e := range sorted {
			if !placed[e] && keep(e) {
				placed[e] = true
				out = append(out, e)
			}
		}
	}
	place(func(e *File) bool { return e.tool })
	if first := c.FirstEntry(); first != "" {
		place(func(e *File) bool { return e.path == first })
	}
	for _, d := range c.Entries {
		if d.First {
			place(func(e *File) bool { return !e.tool && e.decl.Pointer == d.Pointer })
		}
	}
	place(func(*File) bool { return true })
	return out
}

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
