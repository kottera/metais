// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package render

import (
	"slices"
	"strings"

	"metais.org/metais/internal/finding"
	"metais.org/metais/internal/manifest"
	"metais.org/metais/internal/zipfmt"
)

// containerForm is a container's (ADR-005 D8): each evaluation is one
// file, the zip of its entries, assembled from them.
type containerForm struct{}

// DefaultPath is the container's declared path.
func (containerForm) DefaultPath(d manifest.Output) string { return d.Source }

// Expand is the container at, planned before its entries, which are
// planned, inside depth+1 containers, as a tree of their own.
func (containerForm) Expand(p *Planner, at *File, depth int) []*File {
	at.producer = producerOf(byAssembling)
	p.plan(at)
	at.entries = p.container(at, depth+1)
	return []*File{at}
}

// container plans the entries of the container pf, the depth-th
// container on its chain, in pf's evaluation: one tree, held to D12
// within it and to its profile (D8 items 3 and 8, D9), in the container's
// order (D8 item 6).
func (p *Planner) container(pf *File, depth int) []*File {
	c := pf.decl.Container
	in := joinIn(pf.in, pf.path)
	var entries []*File
	var mimetype *File
	if _, ok := c.ToolMimetype(); ok {
		// D9 items 2 to 4: the profile's own mimetype entry, which takes
		// its place in the tree like any other. No declaration writes it
		// (D16 item 2): it keeps the container's, for its profile and
		// where a finding about it points, without its id.
		decl := pf.decl
		decl.ID = ""
		mimetype = &File{decl: decl, path: manifest.MimetypeEntry, ptr: pf.decl.Pointer + "/container/profile",
			literal: true, fan: pf.fan, owner: pf.owner, in: in, producer: producerOf(byLiteral)}
		entries = append(entries, p.plan(mimetype)...)
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
	return containerOrder(c, mimetype, entries)
}

// containerOrder is ADR-005 D8 item 6's order of a container's entries:
// the profile's first entries, the profile's own mimetype before the
// author's [Content_Types].xml; then the files of every first: true
// declaration, in declaration order, each declaration's in byte order of
// their paths; then every other file in byte order. A file appears once,
// at its earliest place. mimetype is the profile's own, or nil.
func containerOrder(c *manifest.Container, mimetype *File, entries []*File) []*File {
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
	place(func(e *File) bool { return e == mimetype })
	if first := c.FirstEntry(); first != "" {
		place(func(e *File) bool { return e.path == first })
	}
	for _, d := range c.Entries {
		if d.First {
			place(func(e *File) bool { return e.decl.Pointer == d.Pointer })
		}
	}
	place(func(*File) bool { return true })
	return out
}
