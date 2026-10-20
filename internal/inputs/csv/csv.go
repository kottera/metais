// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

// Package csv is METAIS's CSV (ADR-010 D8): its own RFC 4180 reader of
// ADR-003 D11, and the writer the csv filter of ADR-004 D10 item 3 writes
// with.
//
// Read takes a file's text after ADR-003 D8's UTF-8 check, byte order mark
// removal and line-break normalization, and returns its header and its data
// records. Each field carries its text, whether it was quoted, and its line
// and column, so a quoted empty field is told apart from an unquoted empty
// one (ADR-003 D11 item 5). Every structural finding the reader can reach is
// reported: an unterminated quote, and a breach of csv-rows, csv-columns or
// csv-cell, stop the reading; every other finding lets it continue at the
// next record. encoding/csv reads no input: it cannot tell "" from an empty
// field, and after an unterminated quote it consumes the rest of the file.
//
// Write is the csv filter's writer, whose bytes are METAIS code (ADR-010 D3
// item 2).
package csv

import (
	"github.com/kottera/metais/internal/finding"
	"github.com/kottera/metais/internal/limits"
)

// Delimiter is the byte that separates the fields of a record: one of
// ADR-003 D11 item 2's set.
type Delimiter byte

// ADR-003 D11 item 2's delimiters.
const (
	Comma     Delimiter = ','
	Semicolon Delimiter = ';'
	Pipe      Delimiter = '|'
	Tab       Delimiter = '\t'
)

// ParseDelimiter returns the delimiter a manifest's csv.delimiter names,
// and false when s is not one of ADR-003 D11 item 2's set.
func ParseDelimiter(s string) (Delimiter, bool) {
	if len(s) != 1 {
		return 0, false
	}
	d := Delimiter(s[0])
	return d, d.Valid()
}

// Valid reports whether d is one of ADR-003 D11 item 2's set.
func (d Delimiter) Valid() bool {
	switch d {
	case Comma, Semicolon, Pipe, Tab:
		return true
	}
	return false
}

// Source is the file Read reads, so that its findings can name it: the
// input it is part of, and where its text came from. Read fills in each
// finding's line, column and pointer.
type Source struct {
	// Input is the subject every finding names: an input (ADR-006 D4).
	Input finding.Subject
	// At is the origin of the text, with its file. Its Line, Column and
	// Pointer are ignored.
	At finding.Location
	// Values, when set, counts the values the file decodes to against
	// input-values, which a command's inputs share (ADR-009 Amendment A4
	// item 1): its array, and each data record's object and each field it
	// holds, an unquoted empty field being absent. The reading stops at
	// the record that passes the bound.
	Values *limits.Counter
}

// Field is one field of a record (ADR-010 D8 item 2).
type Field struct {
	// Text is the field's text: a quoted field's without its enclosing
	// quotes and with each "" read as one quote.
	Text string
	// Quoted is whether the field began with a quote. A quoted empty field
	// is the empty string and an unquoted one is absent (ADR-003 D11 item
	// 5).
	Quoted bool
	// Line and Col are the 1-based line and column, in code points, of the
	// field's first character: its opening quote when it is quoted.
	Line, Col int
}

// Record is one data record.
type Record struct {
	// Index is the record's 0-based ordinal among the file's data records,
	// counting those Read reported a finding in, so that it is the index of
	// the row in the file's value (ADR-003 D11 item 8).
	Index int
	// Line is the 1-based line the record begins on.
	Line   int
	Fields []Field
}

// Table is what Read read of one CSV file.
type Table struct {
	// Header is the header record's fields. It is nil when the file has no
	// header record, or when the header record holds a quote finding.
	Header []Field
	// Records is every data record read without a finding, in file order.
	Records []Record
}
