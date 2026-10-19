// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package loader

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/kottera/metais/internal/finding"
	"github.com/kottera/metais/internal/jsonx"
	"github.com/kottera/metais/internal/limits"
	"github.com/kottera/metais/internal/liquid/value"
	"github.com/kottera/metais/internal/manifest"
	"github.com/kottera/metais/internal/schema"
	"github.com/kottera/metais/internal/schemafail"
	"github.com/kottera/metais/internal/specdata"
	"github.com/kottera/metais/internal/yamlx"
)

// manifestLocation is the location in metais.yaml.
var manifestLocation = finding.Location{Origin: finding.OriginPackage, File: manifest.ManifestFile}

// locate is ADR-002 D11 items 1 and 2: the one manifest, metais.yaml, at
// the top level, and no other spelling of it (D1 items 2 and 3), within
// manifest-size before it is read.
func (o *opening) locate(*Package) []finding.Finding {
	var out []finding.Finding
	names := o.unknown
	if o.form == Packed {
		names = o.topNames()
	}
	for _, n := range names {
		if wrongManifestName(n) {
			out = append(out, finding.New(CodeManifestWrongName, manifestSubject, ptrTo(entryLocation(n)), nil))
		}
	}
	if !o.has(manifest.ManifestFile) {
		out = append(out, finding.New(CodeMissingManifest, pkgSubject, nil, nil))
	}
	if len(out) > 0 {
		return out
	}
	return o.manifestSize(manifest.ManifestFile)
}

func ptrTo(l finding.Location) *finding.Location { return &l }

// manifestDoc is the manifest as parsed.
type manifestDoc struct {
	root value.Value
	src  manifest.Source
}

// parse is ADR-002 D11 item 3: metais.yaml under the JSON profile when its
// first byte other than U+0020, U+0009, U+000A and U+000D is "{", and
// otherwise under the YAML profile (D1 item 1).
func (o *opening) parse(p *Package) []finding.Finding {
	src := o.c.bytes(manifest.ManifestFile)
	var root value.Value
	var pos finding.Positions
	if isJSON(src) {
		doc, fs := jsonx.Parse(src, jsonx.Source{Input: manifestSubject, At: manifestLocation}, o.lim)
		if len(fs) > 0 {
			return fs
		}
		root, pos = doc.Value, doc.Positions()
	} else {
		doc, fs := yamlx.Parse(src, yamlx.Source{Subject: manifestSubject, At: manifestLocation, Size: limits.ManifestSize}, o.lim)
		if len(fs) > 0 {
			return fs
		}
		root, pos = doc.Value, doc.Positions()
	}
	p.ManifestValue = root
	o.doc = manifestDoc{root: root, src: manifest.Source{File: manifest.ManifestFile, Positions: pos}}
	return nil
}

// isJSON reports whether src's first byte other than U+0020, U+0009,
// U+000A and U+000D is "{" (ADR-002 D1 item 1).
func isJSON(src []byte) bool {
	t := bytes.TrimLeft(src, " \t\n\r")
	return len(t) > 0 && t[0] == '{'
}

// compareManifest is ADR-002 D9 item 4, for a packed package: the source
// manifest's canonical form is byte-equal to .compiled/manifest.json, which
// manifest-size bounds too (ADR-009 D3). It runs after the digests and
// before the version gate (ADR-002 D11 item 8).
func (o *opening) compareManifest(*Package) []finding.Finding {
	if o.form != Packed {
		return nil
	}
	if fs := o.manifestSize(compiledManifest); len(fs) > 0 {
		return fs
	}
	if !o.sameBytes(compiledManifest, jsonx.Canonical(o.doc.root)) {
		return []finding.Finding{entryFinding(CodeCompiledMismatch, compiledManifest, nil)}
	}
	return nil
}

// gate is ADR-002 D11 item 4.
func (o *opening) gate(*Package) []finding.Finding {
	_, fs := manifest.Gate(o.doc.root, o.doc.src)
	return fs
}

// layout is ADR-001 D5 item 4 for a packed package, under the version the
// gate admitted: every top-level name is one of the layout's, and every
// file of .compiled/ one the version defines. A source directory's layout
// was checked as it was walked.
func (o *opening) layout(*Package) []finding.Finding {
	if o.form != Packed {
		return nil
	}
	var out []finding.Finding
	for _, n := range o.topNames() {
		if !contains(topLevel, n) && !wrongManifestName(n) {
			out = append(out, entryFinding(CodeUnknownTopLevel, n, nil))
		}
	}
	for _, n := range o.under(compiledDir) {
		if !contains(compiledFiles, n) {
			out = append(out, entryFinding(CodeCompiledUnknown, n, nil))
		}
	}
	return out
}

func contains(s []string, v string) bool {
	for _, x := range s {
		if x == v {
			return true
		}
	}
	return false
}

// validate is ADR-002 D11 items 5 and 6: the manifest against the declared
// minor's schema (D10), and the rules the schema cannot express (D10 item
// 6), in one pass; then the model is built.
func (o *opening) validate(p *Package) []finding.Finding {
	c := manifestSchemaCompiler()
	s, _ := c.Schema(manifestSchemaEntry)
	failures, err := s.Validate(o.doc.root)
	if err != nil {
		panic(fmt.Sprintf("loader: validating the manifest: %v", err))
	}
	for i := range failures {
		renameEntry(&failures[i])
	}
	fs, err := schemafail.Map(schemafail.Target{
		Family:  schemafail.FamilyManifest,
		Subject: manifestSubject,
		Schemas: manifestSchemas{c},
		Locate: func(ptr *finding.Path, key bool) finding.Location {
			l := manifestLocation
			if pos := o.doc.src.Positions; pos != nil {
				at, ok := finding.ValueAt(pos, ptr)
				if key {
					at, ok = finding.KeyAt(pos, ptr)
				}
				if ok {
					l = l.At(at)
				}
			}
			return l
		},
	}, failures...)
	if err != nil {
		panic(fmt.Sprintf("loader: mapping the manifest's failures: %v", err))
	}
	fs = append(fs, manifest.Check(o.doc.root, o.doc.src)...)
	if len(fs) > 0 {
		return fs
	}
	m, err := manifest.Build(o.doc.root)
	if err != nil {
		// The schema and Check accepted the tree, so Build does too
		// (manifest.Build's contract): this is a defect in the tool.
		panic(fmt.Sprintf("loader: %v", err))
	}
	p.Manifest = m
	return nil
}

// The manifest schema is compiled by the one closed compiler (ADR-010 D5
// item 1) as the only schema file of a tree of its own, at
// manifestSchemaEntry; its findings name it by its published identifier
// (ADR-006 D6 item 8).
const (
	manifestSchemaEntry = "inputs/manifest.schema.json"
	manifestSchemaID    = "https://metais.org/schemas/1.0/manifest.schema.json"
)

// manifestSchemaCompiler compiles the embedded manifest schema of 1.0, the
// one minor this tool implements, at the specification's default limits:
// it is the tool's own schema, not the package's.
func manifestSchemaCompiler() *schema.Compiler {
	lim := limits.Defaults()
	c, fs := schema.NewCompiler(newMemTree(map[string][]byte{manifestSchemaEntry: specdata.ManifestSchema10}), lim, decoder(lim))
	if len(fs) > 0 {
		panic(fmt.Sprintf("loader: the manifest schema does not compile: %s", fs[0].Code()))
	}
	return c
}

// renameEntry gives every node of f the manifest schema's published
// identifier as its entry.
func renameEntry(f *schemafail.Failure) {
	if f.Entry == manifestSchemaEntry {
		f.Entry = manifestSchemaID
	}
	for i := range f.Children {
		renameEntry(&f.Children[i])
	}
}

// manifestSchemas answers schemafail's questions about the manifest schema
// by its published identifier.
type manifestSchemas struct{ c *schema.Compiler }

func (m manifestSchemas) entry(e string) string {
	if e == manifestSchemaID {
		return manifestSchemaEntry
	}
	return e
}

func (m manifestSchemas) Annotations(entry, pointer string) schemafail.Annotations {
	return m.c.Annotations(m.entry(entry), pointer)
}

func (m manifestSchemas) Ref(entry, pointer string) (string, string, bool) {
	e, p, ok := m.c.Ref(m.entry(entry), pointer)
	if e == manifestSchemaEntry {
		e = manifestSchemaID
	}
	return e, p, ok
}

func (m manifestSchemas) Properties(entry, pointer string) []string {
	return m.c.Properties(m.entry(entry), pointer)
}

// decodeSchema reads a schema file into the data model by the rules of its
// form, from its extension (ADR-003 D3 item 1, D9, D10), after ADR-003 D8
// item 1's removal of a single leading byte order mark. What a form's
// reader refuses is schema.syntax, naming the rule, and a limit it reaches
// is limit.package: a schema file is the package's own content.
func decoder(lim limits.Set) schema.Decode {
	return func(entry string, src []byte) (value.Value, []finding.Finding) {
		return decodeSchema(entry, src, lim)
	}
}

func decodeSchema(entry string, src []byte, lim limits.Set) (value.Value, []finding.Finding) {
	subject := finding.Subject{Kind: finding.SubjectInput, Name: entry}
	at := entryLocation(entry)
	var v value.Value
	var fs []finding.Finding
	form := "json"
	if strings.HasSuffix(entry, ".json") {
		doc, f := jsonx.Parse(src, jsonx.Source{Input: subject, At: at}, lim)
		v, fs = doc.Value, f
	} else {
		form = "yaml"
		doc, f := yamlx.Parse(bytes.TrimPrefix(src, []byte("\uFEFF")), yamlx.Source{Subject: subject, At: at, Size: limits.SchemaSize}, lim)
		v, fs = doc.Value, f
	}
	if len(fs) == 0 {
		return v, nil
	}
	out := make([]finding.Finding, 0, len(fs))
	for _, f := range fs {
		// At keeps the pointer as the reader's path: writing it out would
		// cost every finding the pointer's length (ADR-009 D10).
		loc, _ := f.At()
		if f.Code() == finding.CodeLimitExceeded {
			p := finding.Params{}
			for _, n := range []string{"limit", "bound", "lowered", "observed"} {
				if v, ok := f.Param(n); ok {
					p[n] = v
				}
			}
			out = append(out, finding.New(finding.CodeLimitPackage, pkgSubject, &loc, p))
			continue
		}
		code := string(f.Code())
		out = append(out, finding.New(CodeSchemaSyntax, pkgSubject, &loc, finding.Params{
			"form": finding.String(form),
			"rule": finding.String(code[strings.LastIndexByte(code, '.')+1:]),
		}))
	}
	return nil, out
}
