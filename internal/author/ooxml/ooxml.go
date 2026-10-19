// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

// Package ooxml is check's scan for a Liquid construct split across Office
// Open XML text runs (ADR-007 D4). A word processor ends a run inside a
// placeholder the author typed — a spelling mark, a tracked change, a
// bookmark or a change of formatting does it — and the Liquid engine,
// which reads the raw bytes, then sees markup inside an expression, or an
// opener that never forms.
//
// Scan reads templates as they are, with no manifest (D4 item 1), so it
// runs before a container is wired up. A template is scanned when its
// source declares a WordprocessingML, SpreadsheetML or DrawingML main
// namespace, or when it is rendered, directly or transitively, from one
// that is; a partial inherits the prefix bindings declared on the root
// element of each template that renders it. The scan resolves every
// element's name through the namespaces in scope, never by its prefix
// (D4 item 2).
//
// The scan pairs two tokenizations of one source: liquid.Constructs, which
// ignores XML, gives the byte range of every Liquid construct, and this
// package's lenient XML tokenizer gives the byte range of every text
// element's content (D4 item 3). A construct that overlaps text content
// and does not lie inside one text element's content is
// template.ooxml.split_construct (item 4); a delimiter broken between two
// text elements of one paragraph container is
// template.ooxml.split_delimiter (item 5). Both are errors (item 7).
//
// The scan of one template is linear in its length.
package ooxml

import (
	"slices"

	"github.com/kottera/metais/internal/finding"
	"github.com/kottera/metais/internal/liquid"
)

// The scan's codes (ADR-007 D4 items 4 and 5). Each is an error of class
// content about a template.
const (
	CodeSplitConstruct finding.Code = "template.ooxml.split_construct"
	CodeSplitDelimiter finding.Code = "template.ooxml.split_delimiter"
)

// TemplatesDir is the directory every template lies under (ADR-001 D5).
const TemplatesDir = "templates/"

// Template is one template of a package.
type Template struct {
	// Path is the template's path relative to templates/, which is the
	// name a render tag gives it.
	Path string
	// Text is the template's source.
	Text []byte
}

// family is a set of the Office Open XML vocabularies a namespace name
// belongs to. A prefix a partial inherits from several renderers may be
// bound to more than one.
type family uint8

const (
	famW family = 1 << iota // WordprocessingML
	famS                    // SpreadsheetML
	famA                    // DrawingML
)

// families maps the main namespaces of ADR-007 D4 item 1, transitional and
// strict, to their vocabulary.
var families = map[string]family{
	"http://schemas.openxmlformats.org/wordprocessingml/2006/main": famW,
	"http://schemas.openxmlformats.org/spreadsheetml/2006/main":    famS,
	"http://schemas.openxmlformats.org/drawingml/2006/main":        famA,
	"http://purl.oclc.org/ooxml/wordprocessingml/main":             famW,
	"http://purl.oclc.org/ooxml/spreadsheetml/main":                famS,
	"http://purl.oclc.org/ooxml/drawingml/main":                    famA,
}

// bindings maps a namespace prefix, "" for the default namespace, to the
// vocabularies it is bound to. A prefix bound to another namespace maps to
// zero.
type bindings map[string]family

// template is what the scan knows of one template before scanning it.
type template struct {
	path    string
	src     string
	cons    []liquid.Construct
	ooxml   bool                // its source declares one of the main namespaces
	root    bindings            // the prefixes its root element declares
	free    map[string]struct{} // the prefixes it resolves through what it inherits
	renders []string            // the partials its render tags name
}

// Scan scans templates for constructs split across text runs, and returns
// the findings in the order of templates, each template's in source order.
// Templates are named by Path; a render tag naming no template is the
// cross-reference's finding, not this one's.
func Scan(templates []Template) []finding.Finding {
	ts := make([]*template, len(templates))
	byPath := make(map[string]int, len(templates))
	for i, t := range templates {
		ts[i] = prepare(t)
		if _, dup := byPath[t.Path]; !dup {
			byPath[t.Path] = i
		}
	}
	inherited, scanned := inherit(ts, byPath)
	var out []finding.Finding
	for i, t := range ts {
		if scanned[i] {
			out = append(out, scanTemplate(t, inherited[i])...)
		}
	}
	return out
}

// Scanned reports, for each of templates, whether Scan scans it: whether
// it declares a main namespace, or is rendered, directly or transitively,
// from one that does (D4 item 1). It is how a parse error's hint knows a
// template is Office Open XML (ADR-007 D2 item 4).
func Scanned(templates []Template) []bool {
	ts := make([]*template, len(templates))
	byPath := make(map[string]int, len(templates))
	for i, t := range templates {
		ts[i] = prepare(t)
		if _, dup := byPath[t.Path]; !dup {
			byPath[t.Path] = i
		}
	}
	_, scanned := inherit(ts, byPath)
	return scanned
}

// prepare reads what the scan needs of t before propagating bindings: its
// constructs, whether it declares a main namespace, the declarations on
// its root element, the prefixes it uses where it declares none, and the
// partials it renders.
func prepare(t Template) *template {
	p := &template{path: t.Path, src: string(t.Text)}
	p.cons = liquid.Constructs(p.src)
	s := newScanner(p, nil)
	s.free = map[string]struct{}{}
	s.walk()
	p.ooxml, p.root, p.free = s.ooxml, s.root, s.free
	for _, c := range p.cons {
		p.renders = append(p.renders, renders(c)...)
	}
	return p
}

// declared reports whether a declares a namespace, and the prefix it
// binds: "" for xmlns, and p for xmlns:p.
func declared(a attr) (string, bool) {
	switch {
	case a.prefix == "" && a.local == "xmlns":
		return "", true
	case a.prefix == "xmlns" && a.local != "":
		return a.local, true
	}
	return "", false
}

// inherit finds which templates are scanned and the bindings each
// inherits (ADR-007 D4 item 1): a template that declares a main namespace
// is scanned, and so is every template rendered from a scanned one, which
// inherits the bindings of its renderer's root element. A renderer's root
// bindings are those it inherits with its own root element's declarations
// over them, so a root element declaring a prefix stops what it would
// inherit for it. A partial rendered from several renderers inherits all
// of their bindings at once, so a construct split in any one context is
// found.
//
// Only a prefix some scanned template resolves through what it inherits
// is propagated, by one walk of the render graph per prefix, which visits
// each template at most once per vocabulary. What is kept is only what a
// template resolves, so the memory is bounded by the templates' size,
// whatever the graph.
func inherit(ts []*template, byPath map[string]int) ([]bindings, []bool) {
	children := make([][]int, len(ts))
	for i, t := range ts {
		seen := map[int]struct{}{}
		for _, name := range t.renders {
			if c, ok := byPath[name]; ok {
				if _, dup := seen[c]; !dup {
					seen[c] = struct{}{}
					children[i] = append(children[i], c)
				}
			}
		}
	}
	scanned := make([]bool, len(ts))
	var stack []int
	for i, t := range ts {
		if t.ooxml {
			scanned[i] = true
			stack = append(stack, i)
		}
	}
	for len(stack) > 0 {
		i := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		for _, c := range children[i] {
			if !scanned[c] {
				scanned[c] = true
				stack = append(stack, c)
			}
		}
	}
	// Each prefix a scanned template resolves through its inheritance, in
	// order, with the scanned roots that declare it.
	type source struct {
		t int
		f family
	}
	sources := map[string][]source{}
	var prefixes []string
	for i, t := range ts {
		if !scanned[i] {
			continue
		}
		for p := range t.free {
			if _, ok := sources[p]; !ok {
				sources[p] = []source{}
				prefixes = append(prefixes, p)
			}
		}
	}
	for i, t := range ts {
		if !scanned[i] {
			continue
		}
		for p, f := range t.root {
			if _, ok := sources[p]; ok && f != 0 {
				sources[p] = append(sources[p], source{i, f})
			}
		}
	}
	slices.Sort(prefixes)
	inherited := make([]bindings, len(ts))
	reached := make([]family, len(ts)) // the vocabularies this prefix's walk has brought to each template
	var touched []int
	type step struct {
		t int
		f family
	}
	var walk []step
	for _, p := range prefixes {
		for _, src := range sources[p] {
			for _, c := range children[src.t] {
				walk = append(walk, step{c, src.f})
			}
		}
		for len(walk) > 0 {
			st := walk[len(walk)-1]
			walk = walk[:len(walk)-1]
			add := st.f &^ reached[st.t]
			if add == 0 {
				continue
			}
			if reached[st.t] == 0 {
				touched = append(touched, st.t)
			}
			reached[st.t] |= add
			if _, ok := ts[st.t].free[p]; ok {
				if inherited[st.t] == nil {
					inherited[st.t] = bindings{}
				}
				inherited[st.t][p] |= add
			}
			if _, shadowed := ts[st.t].root[p]; !shadowed {
				for _, c := range children[st.t] {
					walk = append(walk, step{c, add})
				}
			}
		}
		for _, i := range touched {
			reached[i] = 0
		}
		touched = touched[:0]
	}
	return inherited, scanned
}
