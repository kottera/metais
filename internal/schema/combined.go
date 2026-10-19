// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package schema

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/kottera/metais/internal/finding"
	"github.com/kottera/metais/internal/jsonx"
	"github.com/kottera/metais/internal/limits"
	"github.com/kottera/metais/internal/liquid/value"
)

// CombinedEntry is the package entry the combined schema is published as
// (ADR-001 D7, ADR-003 D7 item 1).
const CombinedEntry = ".compiled/schema.json"

// Combined compiles doc, the combined schema Bundle derives from the
// package's schema files and manifest, which validation evaluates exactly
// (ADR-003 D7 item 5, D22). It refuses a document larger, in its canonical
// form, than combined-schema-size (ADR-009 D3), with limit.package at
// CombinedEntry, and one of more subschemas than schema-subschemas
// (Amendment A1 item 1), at CombinedEntry and the first subschema past it.
//
// The library compiles it as its own root, under CombinedURI, so each
// reference in it resolves to the resource it embeds. A failure inside an
// embedded resource is placed in the schema file the resource is, by its
// $defs key (ADR-006 D6 item 8 names the published entry), so the combined
// schema's failures name the keywords the files' own do.
func (c *Compiler) Combined(doc value.Value) (*Schema, []finding.Finding) {
	if n := int64(len(jsonx.Canonical(doc))); n > c.lim.Value(limits.CombinedSchemaSize) {
		loc := finding.Location{Origin: finding.OriginPackage, File: CombinedEntry}
		return nil, []finding.Finding{c.lim.Breached(limits.CombinedSchemaSize, loc, n, true)}
	}
	if _, ok := c.docs[CombinedEntry]; ok {
		panic("schema: the combined schema is compiled twice")
	}
	base, _ := url.Parse(CombinedURI)
	// The combined schema's subschemas are counted as a whole, apart from
	// the files'; its locations, longer by /$defs/ and a package path, are
	// bounded by the files' and path-length (ADR-009 Amendment A1 items 1
	// and 2).
	w := &prechecker{entry: CombinedEntry, lim: c.lim, count: &subschemas{}}
	w.walk(doc, "", base, 1)
	if f, ok := w.count.breached(c.lim); ok {
		return nil, []finding.Finding{f}
	}
	if len(w.findings) > 0 {
		// Bundle embeds files the pre-check has passed; a refusal here is
		// a defect in the tool.
		panic(fmt.Sprintf("schema: the combined schema fails the pre-check: %v", w.findings[0].Code()))
	}
	c.docs[CombinedEntry] = doc
	for _, a := range w.anchors {
		c.anchors[a.uri] = root{CombinedEntry, a.ptr}
	}
	c.recordRefs(CombinedEntry, w.refs)
	if err := c.lib.AddResource(CombinedURI, toLibrary(doc)); err != nil {
		panic(err)
	}
	c.roots[CombinedURI] = root{CombinedEntry, ""}
	s, err := c.lib.Compile(CombinedURI)
	if err != nil {
		fs, ierr := c.compileFindings(CombinedEntry, err)
		if ierr != nil {
			panic(ierr)
		}
		return nil, fs
	}
	return &Schema{c: c, lib: s}, nil
}

// embedded places a pointer into the combined schema in the schema file
// whose resource it is in, by the resource's $defs key.
func (c *Compiler) embedded(entry, ptr string) (string, string) {
	if entry != CombinedEntry || !strings.HasPrefix(ptr, "/$defs/") {
		return entry, ptr
	}
	toks := pointerTokens(ptr)
	if len(toks) < 2 {
		return entry, ptr
	}
	if _, ok := c.schemas[toks[1]]; !ok {
		return entry, ptr
	}
	rest := ""
	for _, t := range toks[2:] {
		rest += "/" + escapeToken(t)
	}
	return toks[1], rest
}
