// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package csv

import (
	"errors"
	"testing"

	"github.com/kottera/metais/internal/limits"
)

// TestWriteGolden pins the csv filter's bytes (ADR-004 D10 item 3): a
// field is quoted, with '"' doubled, when it holds the delimiter, '"', CR
// or LF, and is bare otherwise; every record ends with LF.
func TestWriteGolden(t *testing.T) {
	for _, c := range []struct {
		name    string
		records [][]string
		delim   rune
		want    string
	}{
		{"bare fields", [][]string{{"a", "b"}, {"1", "2"}}, ',', "a,b\n1,2\n"},
		{"no records", nil, ',', ""},
		{"the delimiter is quoted", [][]string{{"a,b", "c"}}, ',', "\"a,b\",c\n"},
		{"another delimiter is data", [][]string{{"a,b", "c;d"}}, ';', "a,b;\"c;d\"\n"},
		{"a quote is quoted and doubled", [][]string{{`say "hi"`, `"`}}, ',', "\"say \"\"hi\"\"\",\"\"\"\"\n"},
		{"LF is quoted", [][]string{{"1\n2", "x"}}, ',', "\"1\n2\",x\n"},
		{"CR is quoted", [][]string{{"1\r2", "x"}}, ',', "\"1\r2\",x\n"},
		{"CR LF is quoted, not converted", [][]string{{"1\r\n2"}}, ',', "\"1\r\n2\"\n"},
		{"spaces are bare", [][]string{{" a ", "b "}}, ',', " a ,b \n"},
		{"an empty field is bare", [][]string{{"a", "", "c"}, {"", "", ""}}, ',', "a,,c\n,,\n"},
		{"a single empty field is quoted", [][]string{{"a"}, {""}, {"b"}}, ',', "a\n\"\"\nb\n"},
		{"a tab delimiter", [][]string{{"a\tb", "c"}}, '\t', "\"a\tb\"\tc\n"},
		{"a pipe delimiter", [][]string{{"a|b", "c"}}, '|', "\"a|b\"|c\n"},
		{"a delimiter of more than one byte", [][]string{{"a·b", "c", "d"}}, '·', "\"a·b\"·c·d\n"},
		{"any other character is bare", [][]string{{"é", "'x'", "\x00", "a\\b", "#"}}, ',', "é,'x',\x00,a\\b,#\n"},
	} {
		t.Run(c.name, func(t *testing.T) {
			b, err := Write(c.records, c.delim)
			if err != nil {
				t.Fatal(err)
			}
			if string(b) != c.want {
				t.Errorf("wrote %q, want %q", b, c.want)
			}
		})
	}
}

func TestWriteRefusals(t *testing.T) {
	for _, c := range []struct {
		name    string
		records [][]string
		delim   rune
		want    error
	}{
		{"a quote delimiter", [][]string{{"a"}}, '"', ErrDelimiter},
		{"a CR delimiter", [][]string{{"a"}}, '\r', ErrDelimiter},
		{"an LF delimiter", [][]string{{"a"}}, '\n', ErrDelimiter},
		{"a delimiter that is no character", [][]string{{"a"}}, 0xD800, ErrDelimiter},
		{"a record of no fields", [][]string{{"a"}, {}}, ',', ErrNoFields},
		{"records of two widths", [][]string{{"a", "b"}, {"1"}}, ',', ErrFieldCount},
	} {
		t.Run(c.name, func(t *testing.T) {
			if b, err := Write(c.records, c.delim); !errors.Is(err, c.want) || b != nil {
				t.Errorf("Write = %q, %v; want %v", b, err, c.want)
			}
		})
	}
}

// TestWriteCells: a cell marked Quoted is quoted whatever it holds, so the
// empty string is told from an absent value in a record of any width
// (ADR-003 D11 item 5), and it reads back so; every other cell is written
// as Write writes it.
func TestWriteCells(t *testing.T) {
	b, err := WriteCells([][]Cell{
		{{Text: "a"}, {Text: "b"}, {Text: "c"}},
		{{Text: "", Quoted: true}, {Text: ""}, {Text: "x", Quoted: true}},
		{{Text: "p,q"}, {Text: `"`}, {Text: "1\n2"}},
	}, ',')
	if err != nil {
		t.Fatal(err)
	}
	if want := "a,b,c\n\"\",,\"x\"\n\"p,q\",\"\"\"\",\"1\n2\"\n"; string(b) != want {
		t.Errorf("wrote %q, want %q", b, want)
	}
	tab, fs := Read(b, Comma, Source{}, limits.Defaults())
	if len(fs) > 0 || len(tab.Records) != 2 {
		t.Fatalf("read back %+v, %v", tab, fs)
	}
	got := tab.Records[0].Fields
	if got[0].Text != "" || !got[0].Quoted || got[1].Text != "" || got[1].Quoted {
		t.Errorf("read back %+v", got)
	}
	if _, err := WriteCells([][]Cell{{{Text: "a"}}}, '"'); !errors.Is(err, ErrDelimiter) {
		t.Errorf("a quote delimiter: %v", err)
	}
}
