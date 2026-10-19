// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

// Package yamlx reads YAML under METAIS's profile (ADR-002 D2, ADR-003
// D10, ADR-010 D6) and writes the `yaml` filter's text (ADR-004 D10 item
// 2).
//
// go.yaml.in/yaml/v3 turns the bytes into a node tree with positions;
// everything the profile accepts or refuses is decided here, over the
// bytes and over that tree. The library's own type resolution never
// reaches the data model: a plain scalar is resolved by ADR-002 D2 item
// 9's table from its text and style, and the node's Tag is never read for
// a type. One implementation serves the manifest, YAML inputs and Markdown
// frontmatter; the caller's Source names which, and so which family of
// finding codes reports a refusal.
package yamlx

import (
	"bytes"
	"errors"
	"io"
	"strings"

	"go.yaml.in/yaml/v3"

	"github.com/kottera/metais/internal/finding"
	"github.com/kottera/metais/internal/limits"
	"github.com/kottera/metais/internal/liquid/value"
)

// Source says what a YAML document is, which decides the codes and the
// subject of its findings and the size limit that bounds it.
type Source struct {
	// Subject is SubjectManifest for metais.yaml, whose findings are the
	// manifest.yaml family, and an input's subject, with its name, for a
	// YAML input or a Markdown input's frontmatter, whose findings are the
	// yaml family.
	Subject finding.Subject
	// At is where the document is: its origin, with its file or its
	// argument ordinal. Its Line, Column and Pointer are ignored.
	At finding.Location
	// Size is the limit on the document's length in bytes: ManifestSize,
	// InputSize or FrontmatterSize.
	Size limits.ID
	// FirstLine is the line of the file the document's first byte is on,
	// so a frontmatter's positions count from the first line of its file
	// (ADR-006 D5 item 6). Zero is line 1.
	FirstLine int
}

// Doc is a parsed document: its value in the data model and the position
// of every value and member name in it, as jsonx.Doc is for JSON.
type Doc struct {
	// Value is the document's value. It is nil when Parse reported a
	// finding.
	Value value.Value
	pos   *positions
}

// Positions is the document's position table (ADR-006 D13 item 2).
func (d Doc) Positions() finding.Positions { return d.pos }

// positions is a document's position table: for each value, by its JSON
// Pointer, the line and the column in code points of its first
// character, and for each object member the same of its name. It is held
// as a tree of the document's places, so reading a value costs its own
// token, never its whole pointer.
type positions struct {
	root *place
}

// place is one value of the document: where it and, for an object member,
// its name are, and the places within it by token.
type place struct {
	parent   *place
	token    string
	value    finding.Pos
	hasValue bool
	key      finding.Pos
	hasKey   bool
	children map[string]*place
	path     *finding.Path // built when a finding first needs it
	built    bool
}

// child returns the place of the member or element token of the value at
// p. Every occurrence of a duplicate name is one place.
func (p *place) child(token string) *place {
	if c, ok := p.children[token]; ok {
		return c
	}
	if p.children == nil {
		p.children = map[string]*place{}
	}
	c := &place{parent: p, token: token}
	p.children[token] = c
	return c
}

// pointer is p's JSON Pointer, built once, from its parent's.
func (p *place) pointer() *finding.Path {
	if p.parent == nil {
		return nil
	}
	if !p.built {
		p.path, p.built = p.parent.pointer().Child(p.token), true
	}
	return p.path
}

// find returns the place ptr names, an RFC 6901 pointer in its
// JSON-string form.
func (p *positions) find(ptr string) (*place, bool) {
	if p == nil {
		return nil, false
	}
	pl := p.root
	if ptr == "" {
		return pl, true
	}
	if ptr[0] != '/' {
		return nil, false
	}
	for _, tok := range strings.Split(ptr[1:], "/") {
		if !escaped(tok) {
			return nil, false
		}
		if pl = pl.children[tokenUnescaper.Replace(tok)]; pl == nil {
			return nil, false
		}
	}
	return pl, true
}

var tokenUnescaper = strings.NewReplacer("~1", "/", "~0", "~")

// escaped reports whether tok, a token of a pointer in its JSON-string
// form, has ~ only in ~0 and ~1.
func escaped(tok string) bool {
	for i := 0; i < len(tok); i++ {
		if tok[i] == '~' && (i+1 == len(tok) || (tok[i+1] != '0' && tok[i+1] != '1')) {
			return false
		}
	}
	return true
}

// Value is the position of the value at ptr.
func (p *positions) Value(ptr string) (finding.Pos, bool) {
	if pl, ok := p.find(ptr); ok && pl.hasValue {
		return pl.value, true
	}
	return finding.Pos{}, false
}

// Key is the position of the member name of the object member at ptr.
func (p *positions) Key(ptr string) (finding.Pos, bool) {
	if pl, ok := p.find(ptr); ok && pl.hasKey {
		return pl.key, true
	}
	return finding.Pos{}, false
}

// Parse reads src, one YAML document, under the profile. It returns the
// document, or a Doc with no value and every finding: the profile's refusals, each with
// its line and column; a syntax error, which is one finding and stops the
// reading (ADR-002 D2 item 11); or a limit, which stops it before the work
// it bounds.
//
// The size limit is checked before anything is parsed, and the
// value-depth and number-length limits on the node tree (ADR-009 D10 item
// 1). A document holding no node, empty or comments only, is null (ADR-003
// D10 item 4); the manifest's root must be a mapping (ADR-002 D2 item 3).
//
// A byte order mark is refused here. For an input, ADR-003 D8 item 1
// removes a single leading one before the form's parser runs; that is the
// caller's.
func Parse(src []byte, from Source, lim limits.Set) (Doc, []finding.Finding) {
	r := newReader(from, src, lim)
	v, fs := r.parse()
	if fs != nil {
		return Doc{}, fs
	}
	return Doc{Value: v, pos: r.positions}, nil
}

// parse is the value of the document, or the findings that refuse it.
func (r *reader) parse() (value.Value, []finding.Finding) {
	if size := int64(len(r.data)); size > r.lim.Value(r.src.Size) {
		return nil, []finding.Finding{r.limit(r.src.Size, r.at(finding.Pos{}), size, true)}
	}
	r.index()
	text, ok := r.scan()
	if !ok {
		return nil, r.findings
	}
	dec := yaml.NewDecoder(bytes.NewReader(text))
	var root yaml.Node
	switch err := dec.Decode(&root); {
	case errors.Is(err, io.EOF):
		if r.manifest {
			r.add(ruleRootNotMapping, r.at(finding.Pos{}).WithPointer(""), nil)
			return nil, r.findings
		}
		return value.Null{}, nil
	case err != nil:
		r.syntax(err)
		return nil, r.findings
	}
	var next yaml.Node
	switch err := dec.Decode(&next); {
	case err == nil:
		r.add(ruleMultiDocument, r.at(r.pos(&next)), nil)
	case !errors.Is(err, io.EOF):
		r.syntax(err)
		return nil, r.findings
	}
	if len(root.Content) != 1 {
		// A decoded document node holds its one root; anything else is the
		// library breaking its contract, which no finding can describe.
		panic("yamlx: a document node without one root")
	}
	top := root.Content[0]
	if r.manifest && top.Kind != yaml.MappingNode && top.Kind != yaml.AliasNode {
		r.add(ruleRootNotMapping, r.at(r.pos(top)).WithPointer(""), nil)
	}
	v := r.node(top, r.positions.root, 1)
	if len(r.findings) > 0 {
		return nil, r.findings
	}
	return v, nil
}
