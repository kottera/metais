// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package csv

import (
	"errors"
	"testing"
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
