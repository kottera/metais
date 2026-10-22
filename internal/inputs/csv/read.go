// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package csv

import (
	"fmt"
	"strconv"
	"strings"

	"metais.org/metais/internal/finding"
	"metais.org/metais/internal/limits"
)

// Read reads src, one CSV file's text, under delim and lim (ADR-003 D11,
// ADR-009 D10 item 3).
//
// src is the file after ADR-003 D8's UTF-8 check, byte order mark removal
// and line-break normalization, so a record ends in LF and a CR is data;
// lines and columns are those of src.
//
// The first record that is not an empty line is the header and every later
// one a data record; an empty line is no record wherever it occurs. Every
// finding src gives is reported in the order met. A bare quote, or text
// after a closing quote, ends its record's reading: the rest of the line is
// skipped and the reader continues at the next record. An unterminated
// quote, and a breach of csv-rows, csv-columns or csv-cell, stop the
// reading. An empty or duplicate header name, and a data record whose
// field count differs from the header's, are reported without ending
// anything. Table holds what was read either way; a file with any finding
// contributes no value (ADR-003 D11 item 9), which is the caller's to
// apply.
//
// Read panics when delim is not one of ADR-003 D11 item 2's set, which is
// a defect in the caller.
func Read(src []byte, delim Delimiter, from Source, lim limits.Set) (Table, []finding.Finding) {
	if !delim.Valid() {
		panic(fmt.Sprintf("csv: delimiter %q is not one of ADR-003 D11 item 2's set", byte(delim)))
	}
	r := &reader{src: src, delim: byte(delim), from: from, lim: lim, line: 1, col: 1}
	r.file()
	return r.table, r.findings
}

// reader is one reading of a CSV file. off is the byte offset of the next
// byte, and line and col are its position.
type reader struct {
	src   []byte
	delim byte
	from  Source
	lim   limits.Set

	off, line, col int
	stopped        bool

	table     Table
	hasHeader bool // the header record was read without a quote finding
	findings  []finding.Finding

	columns     []*finding.Token // each header name's pointer token, once a finding needs it
	recordPath  *finding.Path    // the pointer of the record at recordIndex, once a finding needs it
	recordIndex int
}

// next moves past the byte at off.
func (r *reader) next() {
	switch c := r.src[r.off]; {
	case c == '\n':
		r.line, r.col = r.line+1, 1
	case c&0xC0 != 0x80: // not a UTF-8 continuation byte
		r.col++
	}
	r.off++
}

func (r *reader) eof() bool { return r.off >= len(r.src) }

func (r *reader) pos() finding.Pos { return finding.Pos{Line: r.line, Column: r.col} }

// file reads every record of src.
func (r *reader) file() {
	sawHeader := false
	index := 0
	if !r.from.Values.Add(1) {
		r.limitExceeded(r.from.Values.ID(), r.pos(), nil)
		return
	}
	for !r.stopped {
		for !r.eof() && r.src[r.off] == '\n' {
			r.next()
		}
		if r.eof() {
			break
		}
		if !sawHeader {
			sawHeader = true
			if fields, ok := r.record(-1); ok {
				r.table.Header, r.hasHeader = fields, true
				r.checkHeader()
			}
			continue
		}
		if int64(index) >= r.lim.Value(limits.CSVRows) {
			r.limitExceeded(limits.CSVRows, r.pos(), r.recordPointer(index))
			return
		}
		line := r.line
		start := r.pos()
		fields, ok := r.record(index)
		if ok && r.hasHeader && len(fields) != len(r.table.Header) {
			r.report(CodeFieldCount, start, r.recordPointer(index), finding.Params{
				"expected": finding.Int(int64(len(r.table.Header))),
				"found":    finding.Int(int64(len(fields))),
			})
			ok = false
		}
		if ok {
			if !r.from.Values.Add(values(fields)) {
				r.limitExceeded(r.from.Values.ID(), start, r.recordPointer(index))
				return
			}
			r.table.Records = append(r.table.Records, Record{Index: index, Line: line, Fields: fields})
		}
		index++
	}
	if !sawHeader {
		r.report(CodeHeaderMissing, r.pos(), nil, nil)
	}
}

// values is how many values a data record of fields decodes to: its
// object, and each field but an unquoted empty one, which is absent
// (ADR-003 D11 item 5).
func values(fields []Field) int64 {
	n := int64(1)
	for _, f := range fields {
		if f.Quoted || f.Text != "" {
			n++
		}
	}
	return n
}

// record reads one record, the header when index is negative, and reports
// whether it was read without a finding. After a quote finding it skips to
// the next record.
func (r *reader) record(index int) ([]Field, bool) {
	var fields []Field
	for {
		if int64(len(fields)) >= r.lim.Value(limits.CSVColumns) {
			r.limitExceeded(limits.CSVColumns, r.pos(), r.fieldPointer(index, len(fields)))
			return nil, false
		}
		f, ok := r.field(index, len(fields))
		if r.stopped {
			return nil, false
		}
		if !ok {
			for !r.eof() && r.src[r.off] != '\n' {
				r.next()
			}
			if !r.eof() {
				r.next()
			}
			return nil, false
		}
		fields = append(fields, f)
		if r.eof() {
			return fields, true
		}
		c := r.src[r.off]
		r.next()
		if c == '\n' {
			return fields, true
		}
		// c is the delimiter, so another field follows, empty at the end
		// of the line or of the file.
	}
}

// field reads the field at off, the column'th of its record, and leaves
// off at the delimiter, LF or end of file that ends it. It reports false
// after a quote finding, with off at the offending quote or character.
func (r *reader) field(index, column int) (Field, bool) {
	f := Field{Line: r.line, Col: r.col}
	bound := r.lim.Value(limits.CSVCell)
	start := r.pos()
	if r.eof() || r.src[r.off] != '"' {
		from := r.off
		for !r.eof() {
			c := r.src[r.off]
			if c == r.delim || c == '\n' {
				break
			}
			if c == '"' {
				r.report(CodeBareQuote, r.pos(), r.fieldPointer(index, column), nil)
				return f, false
			}
			if int64(r.off-from) >= bound {
				r.limitExceeded(limits.CSVCell, start, r.fieldPointer(index, column))
				return f, false
			}
			r.next()
		}
		f.Text = string(r.src[from:r.off])
		return f, true
	}
	f.Quoted = true
	r.next()
	var b strings.Builder
	for {
		if r.eof() {
			r.report(CodeUnterminatedQuote, start, r.fieldPointer(index, column), nil)
			r.stopped = true
			return f, false
		}
		c := r.src[r.off]
		if c == '"' {
			r.next()
			if r.eof() || r.src[r.off] != '"' {
				break
			}
		}
		if int64(b.Len()) >= bound {
			r.limitExceeded(limits.CSVCell, start, r.fieldPointer(index, column))
			return f, false
		}
		b.WriteByte(c)
		r.next()
	}
	if !r.eof() && r.src[r.off] != r.delim && r.src[r.off] != '\n' {
		r.report(CodeAfterQuote, r.pos(), r.fieldPointer(index, column), nil)
		return f, false
	}
	f.Text = b.String()
	return f, true
}

// checkHeader reports each empty header name, and each name that repeats
// an earlier one, related to its first occurrence (ADR-003 D11 item 3).
func (r *reader) checkHeader() {
	first := map[string]int{}
	for i, h := range r.table.Header {
		at := finding.Pos{Line: h.Line, Column: h.Col}
		if h.Text == "" {
			r.report(CodeHeaderEmpty, at, nil, finding.Params{"ordinal": finding.Int(int64(i + 1))})
			continue
		}
		j, seen := first[h.Text]
		if !seen {
			first[h.Text] = i
			continue
		}
		prev := r.table.Header[j]
		r.report(CodeHeaderDuplicate, at, nil, finding.Params{
			"ordinal": finding.Int(int64(i + 1)),
			"first":   finding.Int(int64(j + 1)),
		}, finding.WithRelated(finding.Related{
			Location: r.from.At.At(finding.Pos{Line: prev.Line, Column: prev.Col}),
			Note:     noteFirst,
		}))
	}
}

// recordPointer is the JSON Pointer of the index'th data record in the
// file's value, and nil for the header. The last record's is kept, for
// the next finding about it.
func (r *reader) recordPointer(index int) *finding.Path {
	if index < 0 {
		return nil
	}
	if r.recordPath == nil || r.recordIndex != index {
		r.recordPath, r.recordIndex = (*finding.Path)(nil).Child(strconv.Itoa(index)), index
	}
	return r.recordPath
}

// fieldPointer is the JSON Pointer of the column'th field of the index'th
// data record: the member its header name names, or the record when the
// header names no column there. Each header name's token is made once, so
// a finding in every record under a long name costs none of its length.
func (r *reader) fieldPointer(index, column int) *finding.Path {
	if index < 0 {
		return nil
	}
	if !r.hasHeader || column >= len(r.table.Header) {
		return r.recordPointer(index)
	}
	if r.columns == nil {
		r.columns = make([]*finding.Token, len(r.table.Header))
	}
	if r.columns[column] == nil {
		t := finding.NewToken(r.table.Header[column].Text)
		r.columns[column] = &t
	}
	return r.recordPointer(index).ChildToken(*r.columns[column])
}
