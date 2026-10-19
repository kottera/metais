// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package csv

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/kottera/metais/internal/finding"
	"github.com/kottera/metais/internal/limits"
)

var testSource = Source{
	Input: finding.Subject{Kind: finding.SubjectInput, Name: "rows"},
	At:    finding.Location{Origin: finding.OriginDataDir, File: "rows.csv"},
}

// f is a field in a want table: its text, quotedness, line and column.
func f(text string, quoted bool, line, col int) Field {
	return Field{Text: text, Quoted: quoted, Line: line, Col: col}
}

// got is one finding as the tests compare it: its code, line, column and
// pointer ("-" when it has none).
type got struct {
	code      finding.Code
	line, col int
	ptr       string
}

func (g got) String() string { return fmt.Sprintf("%s@%d:%d%s", g.code, g.line, g.col, g.ptr) }

func summarize(t *testing.T, fs []finding.Finding) []got {
	t.Helper()
	var out []got
	for _, fd := range fs {
		l, ok := fd.Location()
		if !ok {
			t.Fatalf("%s has no location", fd.Code())
		}
		if l.Origin != finding.OriginDataDir || l.File != "rows.csv" {
			t.Errorf("%s names %v %q, not the source", fd.Code(), l.Origin, l.File)
		}
		g := got{fd.Code(), l.Line, l.Column, "-"}
		if l.HasPointer {
			g.ptr = l.Pointer
		}
		out = append(out, g)
	}
	return out
}

func readComma(t *testing.T, src string) (Table, []got) {
	t.Helper()
	tbl, fs := Read([]byte(src), Comma, testSource, limits.Defaults())
	return tbl, summarize(t, fs)
}

// rows returns tbl's header followed by its records' fields.
func rows(tbl Table) [][]Field {
	out := [][]Field{tbl.Header}
	for _, r := range tbl.Records {
		out = append(out, r.Fields)
	}
	return out
}

// TestReadRFC4180 is ADR-003 D11 item 1, a case per bullet.
func TestReadRFC4180(t *testing.T) {
	for _, c := range []struct {
		name  string
		src   string
		delim Delimiter
		want  [][]Field
	}{
		{
			name: "records end in LF",
			src:  "a,b\n1,2\n",
			want: [][]Field{{f("a", false, 1, 1), f("b", false, 1, 3)}, {f("1", false, 2, 1), f("2", false, 2, 3)}},
		},
		{
			name: "the last record's LF is optional",
			src:  "a,b\n1,2",
			want: [][]Field{{f("a", false, 1, 1), f("b", false, 1, 3)}, {f("1", false, 2, 1), f("2", false, 2, 3)}},
		},
		{
			name: "a trailing delimiter ends with an empty field",
			src:  "a,b\n1,\n",
			want: [][]Field{{f("a", false, 1, 1), f("b", false, 1, 3)}, {f("1", false, 2, 1), f("", false, 2, 3)}},
		},
		{
			name: "a trailing delimiter at the end of the file ends with an empty field",
			src:  "a,b\n1,",
			want: [][]Field{{f("a", false, 1, 1), f("b", false, 1, 3)}, {f("1", false, 2, 1), f("", false, 2, 3)}},
		},
		{
			name:  "fields are separated by the input's delimiter, semicolon",
			src:   "a;b\n1,5;2\n",
			delim: Semicolon,
			want:  [][]Field{{f("a", false, 1, 1), f("b", false, 1, 3)}, {f("1,5", false, 2, 1), f("2", false, 2, 5)}},
		},
		{
			name:  "fields are separated by the input's delimiter, pipe",
			src:   "a|b\n1,2|3\n",
			delim: Pipe,
			want:  [][]Field{{f("a", false, 1, 1), f("b", false, 1, 3)}, {f("1,2", false, 2, 1), f("3", false, 2, 5)}},
		},
		{
			name:  "fields are separated by the input's delimiter, tab",
			src:   "a\tb\n1 2\t3\n",
			delim: Tab,
			want:  [][]Field{{f("a", false, 1, 1), f("b", false, 1, 3)}, {f("1 2", false, 2, 1), f("3", false, 2, 5)}},
		},
		{
			name: "a field beginning with a quote is quoted",
			src:  "\"a\",b\n\"1\",\"2\"\n",
			want: [][]Field{{f("a", true, 1, 1), f("b", false, 1, 5)}, {f("1", true, 2, 1), f("2", true, 2, 5)}},
		},
		{
			name: "inside a quoted field two quotes are a quote",
			src:  "a\n\"say \"\"hi\"\"\"\n",
			want: [][]Field{{f("a", false, 1, 1)}, {f(`say "hi"`, true, 2, 1)}},
		},
		{
			name: "inside a quoted field the delimiter is data",
			src:  "a,b\n\"1,2\",3\n",
			want: [][]Field{{f("a", false, 1, 1), f("b", false, 1, 3)}, {f("1,2", true, 2, 1), f("3", false, 2, 7)}},
		},
		{
			name: "inside a quoted field LF is data",
			src:  "a,b\n\"1\n2\",3\n4,5\n",
			want: [][]Field{
				{f("a", false, 1, 1), f("b", false, 1, 3)},
				{f("1\n2", true, 2, 1), f("3", false, 3, 4)},
				{f("4", false, 4, 1), f("5", false, 4, 3)},
			},
		},
		{
			name: "after a closing quote the delimiter, LF or end of file follows",
			src:  "a,b\n\"1\",\"2\"\n\"3\",\"4\"",
			want: [][]Field{
				{f("a", false, 1, 1), f("b", false, 1, 3)},
				{f("1", true, 2, 1), f("2", true, 2, 5)},
				{f("3", true, 3, 1), f("4", true, 3, 5)},
			},
		},
		{
			name: "spaces are data, never trimmed",
			src:  " a , b \n 1 ,\" 2 \"\n",
			want: [][]Field{{f(" a ", false, 1, 1), f(" b ", false, 1, 5)}, {f(" 1 ", false, 2, 1), f(" 2 ", true, 2, 5)}},
		},
		{
			name: "an empty line is not a record, wherever it occurs",
			src:  "\n\na,b\n\n1,2\n\n\n3,4\n\n",
			want: [][]Field{
				{f("a", false, 3, 1), f("b", false, 3, 3)},
				{f("1", false, 5, 1), f("2", false, 5, 3)},
				{f("3", false, 8, 1), f("4", false, 8, 3)},
			},
		},
		{
			name: "a single-column empty string is written \"\" and is a record",
			src:  "a\n\"\"\n\nb\n",
			want: [][]Field{{f("a", false, 1, 1)}, {f("", true, 2, 1)}, {f("b", false, 4, 1)}},
		},
		{
			name: "a quoted field may hold empty lines",
			src:  "a\n\"\n\n\"\n",
			want: [][]Field{{f("a", false, 1, 1)}, {f("\n\n", true, 2, 1)}},
		},
		{
			name: "a CR is data, the file having been normalized",
			src:  "a,b\n1\r,2\n",
			want: [][]Field{{f("a", false, 1, 1), f("b", false, 1, 3)}, {f("1\r", false, 2, 1), f("2", false, 2, 4)}},
		},
		{
			name: "columns count code points",
			src:  "é,ß€,c\n\"日本\",x,\"\"\n",
			want: [][]Field{
				{f("é", false, 1, 1), f("ß€", false, 1, 3), f("c", false, 1, 6)},
				{f("日本", true, 2, 1), f("x", false, 2, 6), f("", true, 2, 8)},
			},
		},
		{
			name: "a header alone is a file of no records",
			src:  "a,b\n",
			want: [][]Field{{f("a", false, 1, 1), f("b", false, 1, 3)}},
		},
	} {
		t.Run(c.name, func(t *testing.T) {
			delim := c.delim
			if delim == 0 {
				delim = Comma
			}
			tbl, fs := Read([]byte(c.src), delim, testSource, limits.Defaults())
			if len(fs) != 0 {
				t.Fatalf("findings %v", summarize(t, fs))
			}
			if g := rows(tbl); !slices.EqualFunc(g, c.want, slices.Equal) {
				t.Errorf("read\n\t%+v\nwant\n\t%+v", g, c.want)
			}
		})
	}
}

// TestQuotedEmptyIsNotUnquotedEmpty is ADR-003 D11 item 5's distinction,
// which encoding/csv cannot make: "" is quoted, an empty field is not.
func TestQuotedEmptyIsNotUnquotedEmpty(t *testing.T) {
	tbl, fs := readComma(t, "a,b,c\n\"\",,\"\"\n")
	if len(fs) != 0 {
		t.Fatalf("findings %v", fs)
	}
	want := []Field{f("", true, 2, 1), f("", false, 2, 4), f("", true, 2, 5)}
	if len(tbl.Records) != 1 || !slices.Equal(tbl.Records[0].Fields, want) {
		t.Fatalf("records %+v, want one of %+v", tbl.Records, want)
	}
	if a, b := tbl.Records[0].Fields[0], tbl.Records[0].Fields[1]; a.Text != b.Text || a.Quoted == b.Quoted {
		t.Errorf("the quoted empty field %+v and the unquoted %+v are not told apart by quotedness alone", a, b)
	}
}

// TestReadRefusals is every structural refusal, each at its line and
// column.
func TestReadRefusals(t *testing.T) {
	for _, c := range []struct {
		name string
		src  string
		want []got
	}{
		{"an empty file has no header", "", []got{{CodeHeaderMissing, 1, 1, "-"}}},
		{"empty lines are no header", "\n\n", []got{{CodeHeaderMissing, 3, 1, "-"}}},
		{"an empty header name", "a,,c\n", []got{{CodeHeaderEmpty, 1, 3, "-"}}},
		{"a quoted empty header name", "a,\"\"\n", []got{{CodeHeaderEmpty, 1, 3, "-"}}},
		{"a single empty header name", "\"\"\n", []got{{CodeHeaderEmpty, 1, 1, "-"}}},
		{"a duplicate header name", "a,b,a,a\n", []got{{CodeHeaderDuplicate, 1, 5, "-"}, {CodeHeaderDuplicate, 1, 7, "-"}}},
		{"header names are compared exactly", "a,A,a \n", nil},
		{"a quoted duplicate header name", "a,\"a\"\n", []got{{CodeHeaderDuplicate, 1, 3, "-"}}},
		{"too few fields", "a,b\n1\n", []got{{CodeFieldCount, 2, 1, "/0"}}},
		{"too many fields", "a,b\n1,2\n1,2,3\n", []got{{CodeFieldCount, 3, 1, "/1"}}},
		{"a bare quote", "a,b\n1,x\"y\n", []got{{CodeBareQuote, 2, 4, "/0/b"}}},
		{"a bare quote past the header's columns", "a\n1,x\"\n", []got{{CodeBareQuote, 2, 4, "/0"}}},
		{"a bare quote in the header", "a,b\"\n1,2\n", []got{{CodeBareQuote, 1, 4, "-"}}},
		{"text after a closing quote", "a,b\n\"1\"x,2\n", []got{{CodeAfterQuote, 2, 4, "/0/a"}}},
		{"a space after a closing quote", "a,b\n1,\"2\" \n", []got{{CodeAfterQuote, 2, 6, "/0/b"}}},
		{"a quote after a closing quote's pair", "a\n\"x\"\"\"y\n", []got{{CodeAfterQuote, 2, 6, "/0/a"}}},
		{"an unterminated quote", "a,b\n1,\"2\n3,4\n", []got{{CodeUnterminatedQuote, 2, 3, "/0/b"}}},
		{"an unterminated quote in the header", "\"a,b\n", []got{{CodeUnterminatedQuote, 1, 1, "-"}}},
		{"an unterminated quote of nothing", "a\n\"", []got{{CodeUnterminatedQuote, 2, 1, "/0/a"}}},
		{"field counts are not checked against a header with a quote finding", "a\"\n1,2\n3\n", []got{{CodeBareQuote, 1, 2, "-"}}},
		{"the column of a refusal counts code points", "é,b\n日本,x\"\n", []got{{CodeBareQuote, 2, 5, "/0/b"}}},
	} {
		t.Run(c.name, func(t *testing.T) {
			_, g := readComma(t, c.src)
			if !slices.Equal(g, c.want) {
				t.Errorf("findings %v, want %v", g, c.want)
			}
		})
	}
}

// TestHeaderDuplicateNamesFirst checks a duplicate header name's params
// and its related first occurrence.
func TestHeaderDuplicateNamesFirst(t *testing.T) {
	_, fs := Read([]byte("x,é,é\n"), Comma, testSource, limits.Defaults())
	if len(fs) != 1 {
		t.Fatalf("findings %v", summarize(t, fs))
	}
	fd := fs[0]
	if _, ok := fd.Param("name"); ok {
		t.Error("the finding echoes the header name, a string from the input (ADR-006 D9 item 4)")
	}
	if n, _ := intParam(fd, "ordinal"); n != 3 {
		t.Errorf("ordinal %d", n)
	}
	if n, _ := intParam(fd, "first"); n != 2 {
		t.Errorf("first %d", n)
	}
	if strings.Contains(fd.Message(), "é") || strings.Contains(fd.Hint(), "é") {
		t.Errorf("the message %q echoes the header name", fd.Message())
	}
	rel := fd.Related()
	if len(rel) != 1 || rel[0].Note != noteFirst || rel[0].Location.Line != 1 || rel[0].Location.Column != 3 || rel[0].Location.File != "rows.csv" {
		t.Errorf("related %+v, want the first occurrence at 1:3", rel)
	}
}

func TestFieldCountParams(t *testing.T) {
	_, fs := Read([]byte("a,b,c\n1\n"), Comma, testSource, limits.Defaults())
	if len(fs) != 1 {
		t.Fatalf("findings %v", summarize(t, fs))
	}
	if n, _ := intParam(fs[0], "expected"); n != 3 {
		t.Errorf("expected %d", n)
	}
	if n, _ := intParam(fs[0], "found"); n != 1 {
		t.Errorf("found %d", n)
	}
}

// TestReadContinuesAfterEachFinding is ADR-003 D11 item 9 and ADR-010 D8
// item 2: after every finding but an unterminated quote the reader carries
// on at the next record, so every finding is reported; after an
// unterminated quote it stops.
func TestReadContinuesAfterEachFinding(t *testing.T) {
	src := strings.Join([]string{
		"a,a,",             // 1: a duplicate and an empty name
		"1,x\"y,\"z\"\"",   // 2: a bare quote; the rest of the line is skipped
		"\"1\"2,3,4",       // 3: text after a closing quote
		"1,2",              // 4: too few fields
		"",                 // 5: no record
		"5,6,7",            // 6: read
		"8,9,\"10\"",       // 7: read
		"\"11\nstill 11\"", // 8-9: too few fields, after a multi-line field
		"1,\"2,3",          // 10: an unterminated quote that runs to the end
		"x,y",              // 11: inside the unterminated field
		"1,2,3,4",
	}, "\n")
	tbl, g := readComma(t, src)
	want := []got{
		{CodeHeaderDuplicate, 1, 3, "-"},
		{CodeHeaderEmpty, 1, 5, "-"},
		{CodeBareQuote, 2, 4, "/0/a"},
		{CodeAfterQuote, 3, 4, "/1/a"},
		{CodeFieldCount, 4, 1, "/2"},
		{CodeFieldCount, 8, 1, "/5"},
		{CodeUnterminatedQuote, 10, 3, "/6/a"},
	}
	if !slices.Equal(g, want) {
		t.Errorf("findings\n\t%v\nwant\n\t%v", g, want)
	}
	wantRecords := []Record{
		{Index: 3, Line: 6, Fields: []Field{f("5", false, 6, 1), f("6", false, 6, 3), f("7", false, 6, 5)}},
		{Index: 4, Line: 7, Fields: []Field{f("8", false, 7, 1), f("9", false, 7, 3), f("10", true, 7, 5)}},
	}
	if !slices.EqualFunc(tbl.Records, wantRecords, func(a, b Record) bool {
		return a.Index == b.Index && a.Line == b.Line && slices.Equal(a.Fields, b.Fields)
	}) {
		t.Errorf("records\n\t%+v\nwant\n\t%+v", tbl.Records, wantRecords)
	}
	if len(tbl.Header) != 3 {
		t.Errorf("header %+v: a header with only name findings is kept", tbl.Header)
	}
}

// TestHeaderWithQuoteFinding: a header whose reading met a quote finding
// is no header, and the records after it are still read.
func TestHeaderWithQuoteFinding(t *testing.T) {
	tbl, g := readComma(t, "a,\"b\"c\n1,2\n3,x\"\n")
	want := []got{{CodeAfterQuote, 1, 6, "-"}, {CodeBareQuote, 3, 4, "/1"}}
	if !slices.Equal(g, want) {
		t.Errorf("findings %v, want %v", g, want)
	}
	if tbl.Header != nil || len(tbl.Records) != 1 || tbl.Records[0].Index != 0 {
		t.Errorf("table %+v, want no header and record 0", tbl)
	}
}

func intParam(fd finding.Finding, name string) (int64, bool) {
	v, ok := fd.Param(name)
	if !ok {
		return 0, false
	}
	return v.Int()
}

func lowered(t *testing.T, args ...string) limits.Set {
	t.Helper()
	lim := limits.Defaults()
	for _, a := range args {
		if fd, ok := lim.Lower(a); !ok {
			t.Fatalf("lowering %s: %s", a, fd.Message())
		}
	}
	return lim
}

// TestLimits is ADR-009 D10 item 3: csv-rows, csv-columns and csv-cell are
// checked as each record is read; at the bound a file is read, past it the
// reading stops with limit.exceeded.
func TestLimits(t *testing.T) {
	for _, c := range []struct {
		name  string
		limit string
		src   string
		want  []got // nil: read with no finding
	}{
		{"csv-rows at the bound", "csv-rows=2", "a\n1\n2\n", nil},
		{"csv-rows past the bound", "csv-rows=2", "a\n1\n2\n\n3\n4\"\n", []got{{CodeLimitExceeded, 5, 1, "/2"}}},
		{"csv-rows counts records with findings", "csv-rows=1", "a\n1\"\n2\n", []got{{CodeBareQuote, 2, 2, "/0/a"}, {CodeLimitExceeded, 3, 1, "/1"}}},
		{"csv-rows does not count the header", "csv-rows=0", "a,b\n", nil},
		{"csv-columns at the bound", "csv-columns=2", "a,b\n1,2\n", nil},
		{"csv-columns past the bound in the header", "csv-columns=2", "a,b,c\n1\"\n", []got{{CodeLimitExceeded, 1, 5, "-"}}},
		{"csv-columns past the bound in a record", "csv-columns=2", "a,b\n1,2,\n3\"\n", []got{{CodeLimitExceeded, 2, 5, "/0"}}},
		{"csv-cell at the bound", "csv-cell=3", "abc,\"d\"\"e\"\n\"x\ny\",zzz\n", nil},
		{"csv-cell past the bound, unquoted", "csv-cell=3", "a,b\n1,abcd\n\"\n", []got{{CodeLimitExceeded, 2, 3, "/0/b"}}},
		{"csv-cell past the bound, quoted", "csv-cell=3", "a,b\n\"ab\"\"c\",1\n", []got{{CodeLimitExceeded, 2, 1, "/0/a"}}},
		{"csv-cell counts the text, not the quotes", "csv-cell=2", "\"\"\"\"\"\"\n", nil},
		{"csv-cell in the header", "csv-cell=1", "ab\n", []got{{CodeLimitExceeded, 1, 1, "-"}}},
	} {
		t.Run(c.name, func(t *testing.T) {
			lim := lowered(t, c.limit)
			_, fs := Read([]byte(c.src), Comma, testSource, lim)
			g := summarize(t, fs)
			if !slices.Equal(g, c.want) {
				t.Fatalf("findings %v, want %v", g, c.want)
			}
			for _, fd := range fs {
				if fd.Code() != CodeLimitExceeded {
					continue
				}
				name, bound, _ := strings.Cut(c.limit, "=")
				if v, _ := fd.Param("limit"); v.Type() != finding.TypeString {
					t.Errorf("limit %v", v)
				} else if s, _ := v.Str(); s != name {
					t.Errorf("limit %s, want %s", s, name)
				}
				if n, _ := intParam(fd, "bound"); strconv.FormatInt(n, 10) != bound {
					t.Errorf("bound %d, want %s", n, bound)
				}
				if !strings.Contains(fd.Message(), "the "+name+" limit of "+bound) {
					t.Errorf("message %q", fd.Message())
				}
				if _, ok := fd.Param("observed"); ok {
					t.Error("a reading that stopped at the bound reports an observed value")
				}
			}
		})
	}
}

// TestCellLimitStopsAtTheBound is ADR-009 D2 item 2: a field past
// csv-cell is read no further than the bound and one byte, whatever its
// length.
func TestCellLimitStopsAtTheBound(t *testing.T) {
	for _, quote := range []string{"", `"`} {
		src := []byte("a\n" + quote + strings.Repeat("x", 1<<20))
		r := &reader{src: src, delim: ',', from: testSource, lim: lowered(t, "csv-cell=8"), line: 1, col: 1}
		r.file()
		if len(r.findings) != 1 || r.findings[0].Code() != CodeLimitExceeded {
			t.Fatalf("quote %q: findings %v", quote, summarize(t, r.findings))
		}
		if read := r.off - len("a\n"+quote); read > 9 {
			t.Errorf("quote %q: read %d bytes of the field past a bound of 8", quote, read)
		}
	}
}

// TestDefaultLimitsApply: Read with the defaults refuses a record of
// csv-columns' default plus one fields, which a reader with no bound
// would read.
func TestDefaultLimitsApply(t *testing.T) {
	n := int(limits.Defaults().Value(limits.CSVColumns))
	_, g := readComma(t, strings.Repeat("a,", n)+"\n")
	if want := []got{{CodeLimitExceeded, 1, 2*n + 1, "-"}}; !slices.Equal(g, want) {
		t.Errorf("findings %v, want %v", g, want)
	}
	_, g = readComma(t, strings.Repeat("a,", n-1)+"a\n")
	if !slices.Contains(g, got{CodeHeaderDuplicate, 1, 3, "-"}) || slices.ContainsFunc(g, func(x got) bool { return x.code == CodeLimitExceeded }) {
		t.Errorf("at the default, findings %v", g[:min(len(g), 3)])
	}
}

func TestDelimiters(t *testing.T) {
	for s, want := range map[string]Delimiter{",": Comma, ";": Semicolon, "|": Pipe, "\t": Tab} {
		if d, ok := ParseDelimiter(s); !ok || d != want {
			t.Errorf("ParseDelimiter(%q) = %q, %v", s, byte(d), ok)
		}
	}
	for _, s := range []string{"", ":", " ", `"`, "\n", ",,", "\\t", "·"} {
		if d, ok := ParseDelimiter(s); ok {
			t.Errorf("ParseDelimiter(%q) = %q, true", s, byte(d))
		}
	}
	defer func() {
		if recover() == nil {
			t.Error("Read with ':' did not panic")
		}
	}()
	Read([]byte("a:b\n"), Delimiter(':'), testSource, limits.Defaults())
}
