// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"cmp"
	pathpkg "path"
	"slices"
	"strings"

	"github.com/kottera/metais/internal/finding"
	"github.com/kottera/metais/internal/limits"
	"github.com/kottera/metais/internal/loader"
	"github.com/kottera/metais/internal/manifest"
	"github.com/kottera/metais/internal/pkgtree"
	"github.com/kottera/metais/internal/template"
	"github.com/kottera/metais/internal/zipfmt"
)

// planned is one file a render plans (ADR-005 D13 phase 1): its
// declaration, its path within its tree, where in the manifest the path
// comes from, the evaluation of a fan-out it is, what it is written from,
// and its bytes once rendered. A file copied from an asset holds no
// bytes: they are streamed from the package as they are written (D13 item
// 4).
type planned struct {
	decl manifest.Output
	path string
	ptr  string
	// template is the template a template output or a mirror renders the
	// file from, beneath templates/.
	template string
	// asset is the entry an asset declaration copies the file from, and
	// tree the package's tree it is read from.
	asset string
	tree  pkgtree.Tree
	// literal is set for a path the manifest writes, or defaults, rather
	// than one a path template rendered.
	literal bool
	fan     *template.Fan
	bytes   []byte
	// owner is the id findings about the file name: its declaration's,
	// or, for a container's entry that has none, the nearest enclosing
	// declaration's.
	owner string
	// in is the path of the container the file is an entry of, through
	// every enclosing container, joined by "/", or "" at the top level.
	in string
	// entries are a container's files, in the container's order (ADR-005
	// D8 item 6).
	entries []*planned
	// tool marks the mimetype entry a profile writes itself (ADR-005 D9
	// item 4).
	tool bool
	// failed marks a file that did not render, and a container one of
	// whose entries did not.
	failed bool
	// size and digest are a container's zip's, measured in phase 4, or an
	// asset's entry's, its digest taken as it is first read; deflated is
	// whether its container deflates an entry.
	size     int64
	digest   string
	deflated bool
}

// item is the 0-based index of the fan-out evaluation pf is, or nil.
func (pf *planned) item() *int {
	if pf.fan == nil {
		return nil
	}
	i := pf.fan.Index()
	return &i
}

// isContainer reports whether pf is a container, whose bytes are its
// entries' zip.
func (pf *planned) isContainer() bool { return pf.decl.Form == manifest.OutputContainer && !pf.tool }

// source is the record's source of pf (ADR-005 D16 item 2): template for
// a file a template output or a mirror renders, asset, container, or
// profile for the mimetype entry a profile writes.
func (pf *planned) source() string {
	switch {
	case pf.tool:
		return "profile"
	case pf.asset != "":
		return "asset"
	case pf.isContainer():
		return "container"
	}
	return "template"
}

// kind is the kind of pf (ADR-005 D7), or nil for an asset, a container
// and a profile's mimetype, which have none (D7 item 3).
func (pf *planned) kind() *string {
	if pf.asset != "" || pf.isContainer() || pf.tool {
		return nil
	}
	k := kindOf(pf.path, pf.decl.Kind)
	return &k
}

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
func planOutputs(pkg *loader.Package, lim limits.Set, s *session, r *template.Renderer) ([]*planned, []finding.Finding) {
	p := &planner{pkg: pkg, lim: lim, s: s, r: r, counted: true,
		entries: pkg.Tree.List(), explicit: explicitTemplates(pkg.Manifest.Outputs)}
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
	pkg *loader.Package
	lim limits.Set
	s   *session
	r   *template.Renderer
	fs  []finding.Finding
	all []*planned // every planned file, at every depth, in planning order
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
func (p *planner) tree(decls []manifest.Output, owner, in string, depth int, fan *template.Fan) []*planned {
	var tree []*planned
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
func (p *planner) declaration(d manifest.Output, x *template.Exprs, owner, in string, depth int, fan *template.Fan) []*planned {
	fans := []*template.Fan{fan}
	if x.ForEach != nil {
		var f *finding.Finding
		if fans, f = p.r.Fans(x); f != nil {
			p.fs = append(p.fs, *f)
			return nil
		}
	}
	mark, marks := len(p.all), len(p.fs)
	var files []*planned
	for i, fan := range fans {
		if i > 0 && p.beyond() {
			p.counted = false
			break
		}
		ok, f := p.r.When(x, fan)
		if f == nil && ok {
			pf := &planned{decl: d, fan: fan, ptr: d.Pointer + "/path", literal: x.Literal, owner: owner, in: in, template: d.Source}
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
func (p *planner) asset(at *planned) []*planned {
	src := at.decl.Source
	file := func(entry, path string) *planned {
		info, err := p.pkg.Tree.Stat(entry)
		if err != nil {
			panic("cli: an entry the plan names has no Stat: " + entry)
		}
		pf := *at
		pf.path, pf.template, pf.asset, pf.tree, pf.size = path, "", entry, p.pkg.Tree, info.Size
		return &pf
	}
	if _, err := p.pkg.Tree.Stat(template.AssetDir + src); err == nil {
		return []*planned{file(template.AssetDir+src, at.path)}
	}
	var out []*planned
	for _, e := range p.entries {
		if rel, ok := strings.CutPrefix(e, treePrefix(template.AssetDir, src)); ok {
			out = append(out, file(e, beneath(at.path, rel)))
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
func (p *planner) mirror(at *planned) []*planned {
	d := at.decl
	var out []*planned
	refused := false
	subj := finding.Subject{Kind: finding.SubjectOutput, Name: at.owner}
	loc := p.at(d.Pointer + "/mirror")
	for _, e := range p.entries {
		rel, ok := strings.CutPrefix(e, treePrefix(template.Dir, d.Source))
		name := strings.TrimPrefix(e, template.Dir)
		if !ok || p.explicit[name] || slices.ContainsFunc(strings.Split(rel, "/"), func(seg string) bool { return strings.HasPrefix(seg, "_") }) {
			continue
		}
		if pathpkg.Base(rel) == ".liquid" {
			refused = true
			p.fs = append(p.fs, finding.New(codeMirrorEmptyName, subj, &loc, finding.Params{"template": finding.String(name)}))
			continue
		}
		pf := *at
		pf.path, pf.template = beneath(at.path, strings.TrimSuffix(rel, ".liquid")), name
		out = append(out, &pf)
	}
	if len(out) == 0 && !refused {
		p.fs = append(p.fs, finding.New(codeMirrorEmpty, subj, &loc, finding.Params{"directory": finding.String(d.Source)}))
	}
	return out
}

// The codes of an output declaration a render reports (ADR-005's
// interfaces name the family output.declaration).
const (
	codeMirrorEmptyName finding.Code = "output.declaration.empty_name"
	codeMirrorEmpty     finding.Code = "output.declaration.empty_mirror"
)

// treePrefix is the prefix of every entry beneath the directory rel of the
// layout directory dir, "." naming dir itself.
func treePrefix(dir, rel string) string {
	if rel == "." {
		return dir
	}
	return dir + rel + "/"
}

// beneath is the path rel beneath the destination directory dest, "."
// being the root of the tree. The paths are joined as written, never
// cleaned: a path is exactly what the manifest gives (ADR-005 D4 item 2).
func beneath(dest, rel string) string {
	if dest == "." {
		return rel
	}
	return dest + "/" + rel
}

// explicitTemplates is every template a template output names, at any
// depth of a container: no mirror renders one (ADR-005 D6 item 3).
func explicitTemplates(decls []manifest.Output) map[string]bool {
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
func (p *planner) container(pf *planned, depth int) []*planned {
	c := pf.decl.Container
	in := joinIn(pf.in, pf.path)
	var entries []*planned
	if m, ok := c.ToolMimetype(); ok {
		// D9 items 2 to 4: the profile's own mimetype entry, which takes
		// its place in the tree like any other.
		tool := &planned{decl: pf.decl, path: manifest.MimetypeEntry, ptr: pf.decl.Pointer + "/container/profile",
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
	if req := c.RequiredEntry(); req != "" && !slices.ContainsFunc(entries, func(e *planned) bool { return e.path == req }) {
		params := finding.Params{"entry": finding.String(req), "profile": finding.String(c.Profile)}
		if i := pf.item(); i != nil {
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
func containerOrder(c *manifest.Container, entries []*planned) []*planned {
	sorted := slices.SortedFunc(slices.Values(entries), func(a, b *planned) int { return strings.Compare(a.path, b.path) })
	out := make([]*planned, 0, len(entries))
	placed := map[*planned]bool{}
	place := func(keep func(e *planned) bool) {
		for _, e := range sorted {
			if !placed[e] && keep(e) {
				placed[e] = true
				out = append(out, e)
			}
		}
	}
	place(func(e *planned) bool { return e.tool })
	if first := c.FirstEntry(); first != "" {
		place(func(e *planned) bool { return e.path == first })
	}
	for _, d := range c.Entries {
		if d.First {
			place(func(e *planned) bool { return !e.tool && e.decl.Pointer == d.Pointer })
		}
	}
	place(func(*planned) bool { return true })
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

// kindOf is a rendered file's kind (ADR-005 D7): the declaration's kind
// when it gives one, otherwise from the path's final extension, ASCII
// case-insensitively, then its whole file name.
func kindOf(p, declared string) string {
	if declared != "" {
		return declared
	}
	name := pathpkg.Base(p)
	if i := strings.LastIndexByte(name, '.'); i >= 0 {
		if k, ok := kindExtensions[strings.ToLower(name[i+1:])]; ok {
			return k
		}
	}
	if k, ok := kindNames[name]; ok {
		return k
	}
	return "text"
}

// kindExtensions and kindNames are ADR-005 D7 item 1's table.
var (
	kindExtensions = func() map[string]string {
		m := map[string]string{}
		for kind, exts := range map[string]string{
			"html":     "html htm",
			"xml":      "xml xhtml svg rels opf ncx xsd xsl xslt plist",
			"markdown": "md markdown",
			"json":     "json geojson webmanifest",
			"yaml":     "yaml yml",
			"csv":      "csv",
			"css":      "css scss less",
			"hash":     "sh bash zsh py rb pl r toml ps1 properties tf",
			"slash":    "js mjs cjs ts tsx jsx go c h cc cpp hpp java kt kts swift rs cs scala dart jsonc proto groovy gradle",
			"sql":      "sql",
		} {
			for _, e := range strings.Fields(exts) {
				m[e] = kind
			}
		}
		return m
	}()
	kindNames = map[string]string{
		"Makefile": "hash", "Dockerfile": "hash", "Containerfile": "hash", ".gitignore": "hash",
		".gitattributes": "hash", ".dockerignore": "hash", ".env": "hash",
	}
)
