// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package csv

import (
	"errors"
	"strings"
	"unicode/utf8"
)

// The refusals of Write. Each is a defect in its caller: the csv filter
// checks its delimiter argument and builds a table of one width before it
// writes (ADR-004 D10 item 3).
var (
	ErrDelimiter  = errors.New("csv: the delimiter is not one character other than '\"', CR or LF")
	ErrNoFields   = errors.New("csv: a record has no fields")
	ErrFieldCount = errors.New("csv: records differ in their number of fields")
)

// Write returns records as RFC 4180 text separated by delim, the bytes of
// ADR-004 D10 item 3. The csv filter passes its header as the first record
// when it writes one.
//
// A field is quoted, with each '"' doubled, when it contains delim, '"', CR
// or LF, and is written bare otherwise. A record of one empty field is
// written "", because an empty line is no record (ADR-003 D11 item 1).
// Every record, the last too, ends with LF.
//
// delim is one character other than '"', CR and LF; every record has the
// same number of fields, at least one.
func Write(records [][]string, delim rune) ([]byte, error) {
	cells := make([][]Cell, len(records))
	for i, rec := range records {
		cells[i] = make([]Cell, len(rec))
		for j, text := range rec {
			cells[i][j] = Cell{Text: text, Quoted: len(rec) == 1 && text == ""}
		}
	}
	return WriteCells(cells, delim)
}

// Cell is one field WriteCells writes: its text, and whether it is quoted
// whatever its text holds.
type Cell struct {
	Text   string
	Quoted bool
}

// WriteCells returns records as Write does, except that a cell marked
// Quoted is quoted too. A file that is read back must tell the empty
// string, a quoted empty field, from an absent value, an unquoted one
// (ADR-003 D11 item 5), which the sample generator's CSV writes so.
func WriteCells(records [][]Cell, delim rune) ([]byte, error) {
	if delim == '"' || delim == '\r' || delim == '\n' || !utf8.ValidRune(delim) {
		return nil, ErrDelimiter
	}
	var b strings.Builder
	for _, rec := range records {
		switch {
		case len(rec) == 0:
			return nil, ErrNoFields
		case len(rec) != len(records[0]):
			return nil, ErrFieldCount
		}
		for i, c := range rec {
			if i > 0 {
				b.WriteRune(delim)
			}
			if !c.Quoted && !strings.ContainsRune(c.Text, delim) && !strings.ContainsAny(c.Text, "\"\r\n") {
				b.WriteString(c.Text)
				continue
			}
			b.WriteByte('"')
			b.WriteString(strings.ReplaceAll(c.Text, `"`, `""`))
			b.WriteByte('"')
		}
		b.WriteByte('\n')
	}
	return []byte(b.String()), nil
}
