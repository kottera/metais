// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package render

import (
	pathpkg "path"
	"slices"
	"strings"

	"metais.org/metais/internal/finding"
	"metais.org/metais/internal/manifest"
	"metais.org/metais/internal/template"
)

// mirrorForm is a mirror's (ADR-005 D6).
type mirrorForm struct{}

// DefaultPath is the mirror's directory beneath templates/.
func (mirrorForm) DefaultPath(d manifest.Output) string { return d.Source }

// Expand is the files of the mirror's evaluation at, whose path names
// their destination (ADR-005 D6): every template beneath templates/<dir>,
// "." naming templates/ itself, in byte order, but one with a name
// beginning with _ at any depth beneath the mirror's root and one a
// template field names anywhere in the manifest, each rendered to its path
// relative to the mirror's root, with a final .liquid removed, beneath the
// destination. A template named exactly .liquid would be written with an
// empty name, and is refused; a mirror that yields no file is a warning.
func (mirrorForm) Expand(p *Planner, at *File, _ int) []*File {
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
		pf.producer = producerOf(byRendering)
		out = append(out, &pf)
	}
	if len(out) == 0 && !refused {
		p.fs = append(p.fs, finding.New(CodeMirrorEmpty, subj, &loc, finding.Params{"directory": finding.String(d.Source)}))
	}
	return p.plan(out...)
}

// The codes of an output declaration a render reports (ADR-005's
// interfaces name the family output.declaration).
const (
	CodeMirrorEmptyName finding.Code = "output.declaration.empty_name"
	CodeMirrorEmpty     finding.Code = "output.declaration.empty_mirror"
)

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
