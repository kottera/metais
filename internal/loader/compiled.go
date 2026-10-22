// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package loader

import (
	"fmt"
	"slices"

	"metais.org/metais/internal/finding"
	"metais.org/metais/internal/jsonx"
	"metais.org/metais/internal/liquid/value"
	"metais.org/metais/internal/markdown"
	"metais.org/metais/internal/schema"
)

// compile is ADR-003 D22 item 1 at open: every schema file under inputs/
// compiled, whether or not an input names it (D3 item 1), then the
// combined schema bundled from them and the manifest (D7) and compiled
// too, within combined-schema-size.
func (o *opening) compile(p *Package) []finding.Finding {
	files := map[string][]byte{}
	for _, n := range o.names {
		if schema.IsSchemaFile(n) {
			files[n] = o.c.bytes(n)
		}
	}
	values := map[string]value.Value{}
	c, fs := schema.NewCompiler(newMemTree(files), o.lim, func(entry string, src []byte) (value.Value, []finding.Finding) {
		v, fs := decodeSchema(entry, src, o.lim)
		if len(fs) == 0 {
			values[entry] = v
		}
		return v, fs
	})
	if len(fs) > 0 {
		return fs
	}
	m := p.Manifest
	decl := schema.Declarations{Constraints: m.InputConstraints}
	for _, in := range m.Inputs {
		si := schema.Input{Name: in.Name, Title: in.Title, Schema: in.Schema, Required: in.Required}
		if in.Description != "" {
			d := in.Description
			si.Description = &d
		}
		decl.Inputs = append(decl.Inputs, si)
	}
	paths := make([]string, 0, len(values))
	for n := range values {
		paths = append(paths, n)
	}
	slices.Sort(paths)
	sf := make([]schema.File, 0, len(paths))
	for _, n := range paths {
		sf = append(sf, schema.File{Path: n, Root: values[n]})
	}
	doc, err := schema.Bundle(decl, sf)
	if err != nil {
		// Every rule Bundle refuses is reported before it runs: the
		// manifest's paths, and the compiler's $schema and $id checks.
		panic(fmt.Sprintf("loader: %v", err))
	}
	combined, fs := c.Combined(doc)
	if len(fs) > 0 {
		return fs
	}
	p.Schemas, p.Combined, p.CombinedValue = c, combined, doc
	return nil
}

// derive is ADR-001 D7 item 2: the compiled files, each a deterministic
// function of the package's other entries. A packed package's are compared
// byte for byte with what it carries, and every one that differs refuses
// it (D8 step 7); a source directory's are kept, for its package digest.
// The canonical manifest was compared before the version gate, so only the
// combined schema and the references index are compared here.
func (o *opening) derive(p *Package) []finding.Finding {
	p.References = o.references()
	refs := encodeReferences(p.References)
	schemaBytes := jsonx.Canonical(p.CombinedValue)
	if o.form == Directory {
		o.derived = []File{
			{compiledManifest, jsonx.Canonical(o.doc.root)},
			{compiledReferences, refs},
			{compiledSchema, schemaBytes},
		}
		return nil
	}
	var out []finding.Finding
	for _, f := range []File{{compiledReferences, refs}, {compiledSchema, schemaBytes}} {
		if !o.sameBytes(f.Name, f.Content) {
			out = append(out, entryFinding(CodeCompiledMismatch, f.Name, nil))
		}
	}
	return out
}

// Reference is one entry of the references index (ADR-008 D17 item 1).
type Reference struct {
	// Path is the entry path, beneath references/.
	Path string
	// Size is the entry's length in bytes.
	Size int64
	// Text reports whether the entry is valid UTF-8 with no NUL byte.
	Text bool
	// Lines is a text entry's line count by ADR-008 D5's rule, and zero
	// for a binary entry, which has none.
	Lines int64
	// Title is the plain text of the first level-1 heading of a text
	// entry whose name ends in .md or .markdown, and nil for every other
	// entry and for one with no level-1 heading.
	Title *string
}

// references is the references index of the package's entries: one
// Reference per entry beneath references/, in byte order.
func (o *opening) references() []Reference {
	var out []Reference
	for _, n := range o.under(referencesDir) {
		s := o.c.refs[n]
		r := Reference{Path: n, Size: s.size, Text: s.text()}
		if r.Text {
			r.Lines = s.lines()
			if isMarkdown(n) {
				r.Title = referenceTitle(o.c.bytes(n))
			}
		}
		out = append(out, r)
	}
	return out
}

// referenceTitle is the plain text of the first level-1 heading of src, a
// Markdown entry's bytes, and nil when it has none. The open needs only
// the title, so it keeps no other heading (ADR-009 Amendment A6 item 2);
// metais outline reads the whole outline.
func referenceTitle(src []byte) *string {
	if t, ok := markdown.Title(src); ok {
		return &t
	}
	return nil
}

// encodeReferences is .compiled/references.json for refs, in ADR-002 D9's
// canonical form (ADR-008 D17 item 1).
func encodeReferences(refs []Reference) []byte {
	items := make([]value.Value, 0, len(refs))
	for _, r := range refs {
		members := []value.Member{
			{Name: "path", Value: mustString(r.Path)},
			{Name: "size", Value: mustInt(r.Size)},
			{Name: "text", Value: value.Bool(r.Text)},
		}
		if r.Text {
			members = append(members, value.Member{Name: "lines", Value: mustInt(r.Lines)})
		}
		var title value.Value = value.Null{}
		if r.Title != nil {
			title = mustString(*r.Title)
		}
		members = append(members, value.Member{Name: "title", Value: title})
		obj, err := value.NewObject(members...)
		if err != nil {
			panic(err)
		}
		items = append(items, obj)
	}
	arr, err := value.NewArray(items...)
	if err != nil {
		panic(err)
	}
	root, err := value.NewObject(value.Member{Name: "references", Value: arr})
	if err != nil {
		panic(err)
	}
	return jsonx.Canonical(root)
}

func mustString(s string) value.String {
	v, err := value.NewString(s)
	if err != nil {
		panic(err)
	}
	return v
}

func mustInt(n int64) value.Number {
	v, err := value.ParseNumber(fmt.Sprint(n))
	if err != nil {
		panic(err)
	}
	return v
}
