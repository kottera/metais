// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

// Package jsonx is METAIS's JSON (ADR-010 D7): its own strict RFC 8259
// reader, the canonical form of ADR-002 D9, and the writer the json filter
// of ADR-004 D10 item 1 writes with.
//
// Parse reads one JSON text into the data model's value tree, refusing
// everything ADR-003 D8 and D9 refuse and reporting each refusal as a
// positioned finding: invalid UTF-8, by line and byte; an escaped unpaired
// surrogate or a noncharacter; every duplicate member name; a number the
// data model cannot hold; text that is not one JSON value; and ADR-009's
// input-size, value-depth and number-length limits. Every number keeps its
// literal text, and every value and member name its byte offset, from
// which a finding's line and column are computed (ADR-006 D5 item 4, D13
// item 2). encoding/json reads no input: it hides every one of these.
//
// Canonical is RFC 8785's serialization, through gowebpki/jcs (ADR-002
// D9, ADR-010 D7 item 2). Encode is the json filter's writer, whose bytes
// are METAIS code (ADR-010 D3 item 2): compact, it is byte-identical to
// Canonical; with an indent it pretty-prints as ADR-004 D10 item 1 says.
package jsonx

import (
	"github.com/kottera/metais/internal/finding"
	"github.com/kottera/metais/internal/limits"
	"github.com/kottera/metais/internal/liquid/value"
)

// Source is the document Parse reads, so that its findings can name it: the
// input it is part of, and where its text came from. Parse fills in each
// finding's line, column and pointer.
type Source struct {
	// Input is the subject every finding names: an input (ADR-006 D4),
	// or, for metais.yaml read under ADR-002 D3's JSON profile, the
	// manifest, whose findings are the manifest.json family and whose
	// limits are limit.package, about the package.
	Input finding.Subject
	// At is the origin of the text, with its file or its argument ordinal.
	// Its Line, Column and Pointer are ignored.
	At finding.Location
	// Output says the text is a rendered output, parsed to be validated by
	// its schema (ADR-005 D11 item 3), whose length output-size bounds
	// rather than input-size.
	Output bool
	// Size is the limit on the length of a text that is the package's own
	// content: ManifestSize for metais.yaml under the JSON profile,
	// SchemaSize for a schema file, EntrySize for the digest file. A data
	// limit never bounds the package (ADR-009 D3). Its zero value,
	// PackageSize, bounds a package file and never one document, and
	// leaves the text an input's, bounded by input-size, or an output's.
	Size limits.ID
}

// Doc is one parsed JSON text: its value and where each part of it is.
type Doc struct {
	// Value is the text's value. It is nil when Parse reported a finding.
	Value value.Value
	pos   *table
}

// Positions returns the document's position table (ADR-006 D13 item 2):
// the line and column of every value and of every member name, by JSON
// Pointer.
func (d Doc) Positions() finding.Positions { return d.pos }

// ValueOffset returns the byte offset in the text Parse was given of the
// first byte of the value at ptr.
func (d Doc) ValueOffset(ptr string) (int, bool) { return d.pos.valueOffset(ptr) }

// KeyOffset returns the byte offset of the opening quote of the member
// name, for the object member at ptr.
func (d Doc) KeyOffset(ptr string) (int, bool) { return d.pos.keyOffset(ptr) }

// Parse reads src as one JSON input file or argument (ADR-003 D8 and D9)
// under lim. A single leading UTF-8 byte order mark is removed first, and
// is neither part of the value nor counted in a column (ADR-003 D8 item 1,
// ADR-006 D5 item 3); offsets stay offsets into src.
//
// Every finding the text gives is reported, in the order met, with what
// the reader could not get past last: invalid UTF-8, a syntax error, text
// after the value and a breach of value-depth stop the reading; a
// duplicate name, an unpaired surrogate, a noncharacter, a number out of
// range and a breach of number-length do not. Doc.Value is set only when
// there are no findings; the position table holds what was read either
// way.
func Parse(src []byte, from Source, lim limits.Set) (Doc, []finding.Finding) {
	p := &parser{src: src, from: from, lim: lim, lines: newLineIndex(src)}
	v, root := p.document()
	d := Doc{pos: newTable(p.lines, root)}
	if len(p.findings) == 0 {
		d.Value = v
	}
	return d, p.findings
}
