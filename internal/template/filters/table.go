// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package filters

import (
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/kottera/metais/internal/liquid"
	"github.com/kottera/metais/internal/liquid/value"
)

// csv and markdown_table write an array of objects as a table (ADR-004 D10
// items 3 and 4). Both read it the same way, through table: the columns
// are the positional arguments, or, without any, the union of the rows'
// member names in canonical order; each cell is the text of its value
// (D4 item 4), nil being empty; and headers: replaces the header text of
// the named columns.

// csvFilter is ADR-004 D10 item 3: RFC 4180 records, each ending with a
// line feed, with a header record unless header: false, and a field
// quoted, its quotes doubled, when it holds the delimiter, a quote, a
// carriage return or a line feed.
var csvFilter = &filter{
	name:       "csv",
	max:        -1,
	positional: stringArg,
	keywords:   keywordsOf("delimiter", "header", "headers"),
	shapeCheck: namedHeaders,
	apply: func(c *call, in value.Value) (value.Value, *liquid.Error) {
		t, err := c.table(in)
		if err != nil {
			return nil, err
		}
		delim := ","
		if d, ok := c.kw["delimiter"]; ok {
			delim = d.(value.String).Text()
		}
		header := true
		if h, ok := c.kw["header"]; ok {
			header = bool(h.(value.Bool))
		}
		var records [][]string
		if header {
			records = append(records, t.headers)
		}
		records = append(records, t.rows...)
		if len(t.columns) == 0 {
			records = nil
		}
		field := func(s string) (string, bool) {
			quote := strings.Contains(s, delim) || strings.ContainsAny(s, "\"\r\n")
			return s, quote
		}
		var n int64
		for _, r := range records {
			for i, s := range r {
				if i > 0 {
					n += int64(len(delim))
				}
				n += int64(len(s))
				if _, quote := field(s); quote {
					n += 2 + int64(strings.Count(s, `"`))
				}
			}
			n++
		}
		return c.text(n, func() string {
			var b strings.Builder
			for _, r := range records {
				for i, s := range r {
					if i > 0 {
						b.WriteString(delim)
					}
					if _, quote := field(s); quote {
						b.WriteByte('"')
						b.WriteString(strings.ReplaceAll(s, `"`, `""`))
						b.WriteByte('"')
					} else {
						b.WriteString(s)
					}
				}
				b.WriteByte('\n')
			}
			return b.String()
		})
	},
}

// markdownTableFilter is ADR-004 D10 item 4: a GitHub Flavored Markdown
// table, `| h | h |`, then `| --- | --- |`, then a line per row, each line
// ending with a line feed; in every cell, | is \| and each line break is
// <br>. With no columns it is the empty string.
var markdownTableFilter = &filter{
	name:       "markdown_table",
	max:        -1,
	positional: stringArg,
	keywords:   keywordsOf("headers"),
	shapeCheck: namedHeaders,
	apply: func(c *call, in value.Value) (value.Value, *liquid.Error) {
		t, err := c.table(in)
		if err != nil {
			return nil, err
		}
		if len(t.columns) == 0 {
			return c.text(0, func() string { return "" })
		}
		delims := make([]string, len(t.columns))
		for i := range delims {
			delims[i] = "---"
		}
		lines := append([][]string{t.headers, delims}, t.rows...)
		for _, l := range lines {
			for i, s := range l {
				l[i] = mdCell(s)
			}
		}
		var n int64
		for _, l := range lines {
			// "| " before the first cell, " | " between cells, and " |"
			// and the line feed after the last.
			n += 2 + 3*int64(len(l)-1) + 3
			for _, s := range l {
				n += int64(len(s))
			}
		}
		return c.text(n, func() string {
			var b strings.Builder
			for _, l := range lines {
				b.WriteString("| ")
				b.WriteString(strings.Join(l, " | "))
				b.WriteString(" |\n")
			}
			return b.String()
		})
	},
}

// mdCell escapes a markdown_table cell: each | is \|, and each line break,
// CR LF, LF or CR, is <br>.
func mdCell(s string) string {
	if !strings.ContainsAny(s, "|\r\n") {
		return s
	}
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		switch c := s[i]; c {
		case '|':
			b.WriteString(`\|`)
		case '\r':
			if i+1 < len(s) && s[i+1] == '\n' {
				i++
			}
			b.WriteString("<br>")
		case '\n':
			b.WriteString("<br>")
		default:
			b.WriteByte(c)
		}
	}
	return b.String()
}

// tableKeywords are the keyword checks csv and markdown_table share.
var tableKeywords = map[string]func(string, value.Value) *liquid.Error{
	// headers: an array of strings. That it has one per named column is
	// checked with the columns, at render.
	"headers": func(name string, v value.Value) *liquid.Error {
		a, ok := v.(value.Array)
		if !ok {
			return argErr(name, "headers", "headers is an array of strings, not this "+liquid.KindName(v))
		}
		for _, e := range a.All() {
			if _, ok := e.(value.String); !ok {
				return argErr(name, "headers", "headers holds a "+liquid.KindName(e)+", not a string")
			}
		}
		return nil
	},
	// header: a boolean.
	"header": func(name string, v value.Value) *liquid.Error {
		if _, ok := v.(value.Bool); !ok {
			return argErr(name, "header", "header is true or false, not this "+liquid.KindName(v))
		}
		return nil
	},
	// delimiter: one character, neither ", CR nor LF.
	"delimiter": func(name string, v value.Value) *liquid.Error {
		s, ok := v.(value.String)
		if !ok || utf8.RuneCountInString(s.Text()) != 1 || strings.ContainsAny(s.Text(), "\"\r\n") {
			return argErr(name, "delimiter", "the delimiter is one character other than a quote, CR or LF")
		}
		return nil
	},
}

// table is an array of objects read as a table's text.
type table struct {
	columns []string
	headers []string   // one per column
	rows    [][]string // one text per column
}

// table reads in, which must be an array of objects, as a table. A named
// column a row lacks is CodeUndefinedProperty, its path the column's name;
// a cell that is an array or an object is CodeFilterInput. It charges the
// input's deep size beyond its shallow size before reading it and, for the
// union of the rows' names, the comparisons of sorting them, as sort
// charges them.
func (c *call) table(in value.Value) (table, *liquid.Error) {
	a, ok := in.(value.Array)
	if !ok {
		return table{}, inputErr(c.name, in, c.name+" takes an array of objects")
	}
	var rows []value.Object
	for _, e := range a.All() {
		o, ok := e.(value.Object)
		if !ok {
			return table{}, inputErr(c.name, e, c.name+" takes an array of objects; an element is not one")
		}
		rows = append(rows, o)
	}
	if err := c.beyondShallow(in); err != nil {
		return table{}, err
	}
	var t table
	named := len(c.pos) > 0
	if named {
		for _, p := range c.pos {
			t.columns = append(t.columns, p.(value.String).Text())
		}
	} else {
		var names []string
		var nameBytes int64
		for _, r := range rows {
			for name := range r.All() {
				names = append(names, name)
				nameBytes += int64(len(name))
			}
		}
		if err := c.sortCharge(len(names), nameBytes); err != nil {
			return table{}, err
		}
		t.columns = sortedUnique(names)
	}
	t.headers = t.columns
	if h, ok := c.kw["headers"]; ok {
		ha := h.(value.Array)
		if ha.Len() != len(t.columns) {
			return table{}, argErr(c.name, "headers", "headers names one header for each named column")
		}
		t.headers = nil
		for _, e := range ha.All() {
			t.headers = append(t.headers, e.(value.String).Text())
		}
	}
	for _, r := range rows {
		cells := make([]string, len(t.columns))
		for i, col := range t.columns {
			v, ok := r.Get(col)
			if !ok {
				if named {
					return table{}, liquid.FilterError(c.name, liquid.CodeUndefinedProperty, map[string]string{"path": col},
						"a row has no member the column names")
				}
				continue
			}
			s, ok := liquid.ScalarText(v)
			if !ok {
				return table{}, inputErr(c.name, v, "a cell is an "+liquid.KindName(v)+", which has no text in a table")
			}
			cells[i] = s
		}
		t.rows = append(t.rows, cells)
	}
	return t, nil
}

// sortCharge charges sorting n keys whose strings hold strBytes bytes, as
// the standard catalog's sort does.
func (c *call) sortCharge(n int, strBytes int64) *liquid.Error {
	return c.units(liquid.SortWork(n, strBytes))
}

// namedHeaders refuses, whatever the data, a headers: given with no
// column named, since headers name the named columns.
func namedHeaders(name string, pos int, keywords []string) *liquid.Error {
	if pos == 0 && slices.Contains(keywords, "headers") {
		return argErr(name, "headers", "headers names one header for each named column, and no column is named")
	}
	return nil
}
