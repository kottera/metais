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
		for i, text := range rec {
			if i > 0 {
				b.WriteRune(delim)
			}
			if !quoted(text, delim, len(rec)) {
				b.WriteString(text)
				continue
			}
			b.WriteByte('"')
			b.WriteString(strings.ReplaceAll(text, `"`, `""`))
			b.WriteByte('"')
		}
		b.WriteByte('\n')
	}
	return []byte(b.String()), nil
}

// quoted reports whether Write quotes text in a record of width fields.
func quoted(text string, delim rune, width int) bool {
	return strings.ContainsRune(text, delim) || strings.ContainsAny(text, "\"\r\n") || (width == 1 && text == "")
}
