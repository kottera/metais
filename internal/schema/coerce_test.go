// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package schema

import (
	"errors"
	"testing"

	"github.com/kottera/metais/internal/finding"
	"github.com/kottera/metais/internal/inputs/csv"
	"github.com/kottera/metais/internal/limits"
	"github.com/kottera/metais/internal/liquid/value"
)

// coerced describes a coerced value for comparison: its kind and, for a
// number, its literal, so a re-rounded or re-spelt number fails.
func coerced(v value.Value) string {
	switch x := v.(type) {
	case value.Bool:
		if x {
			return "boolean true"
		}
		return "boolean false"
	case value.Number:
		return "number " + x.Literal()
	case value.String:
		return "string " + `"` + x.Text() + `"`
	case nil:
		return "absent"
	}
	return v.Kind().String()
}

type coerceCase struct {
	text  string
	types TypeSet
	want  string
}

func checkCoerce(t *testing.T, cases []coerceCase) {
	t.Helper()
	for _, c := range cases {
		v, err := Coerce(c.text, c.types)
		if err != nil {
			t.Errorf("Coerce(%q, %s): %v", c.text, c.types, err)
			continue
		}
		if got := coerced(v); got != c.want {
			t.Errorf("Coerce(%q, %s) = %s, want %s", c.text, c.types, got, c.want)
		}
	}
}

// TestCoerceBoolean is D14 item 2's first row: boolean in T and one of
// YAML's six core forms.
func TestCoerceBoolean(t *testing.T) {
	checkCoerce(t, []coerceCase{
		{"true", TypeBoolean, "boolean true"},
		{"True", TypeBoolean, "boolean true"},
		{"TRUE", TypeBoolean, "boolean true"},
		{"false", TypeBoolean, "boolean false"},
		{"False", TypeBoolean, "boolean false"},
		{"FALSE", TypeBoolean, "boolean false"},
		// Not a core form.
		{"tRUE", TypeBoolean, `string "tRUE"`},
		{"yes", TypeBoolean, `string "yes"`},
		{"on", TypeBoolean, `string "on"`},
		{"1", TypeBoolean, `string "1"`},
		{" true", TypeBoolean, `string " true"`},
		{"", TypeBoolean, `string ""`},
		// The row comes first.
		{"true", TypeBoolean | TypeInteger | TypeNumber | TypeString, "boolean true"},
		// boolean not in T.
		{"true", TypeString, `string "true"`},
		{"false", TypeInteger | TypeNumber, `string "false"`},
		{"true", TypeNull | TypeObject | TypeArray, `string "true"`},
	})
}

// TestCoerceInteger is the second row: integer in T and text matching
// -?(0|[1-9][0-9]*).
func TestCoerceInteger(t *testing.T) {
	checkCoerce(t, []coerceCase{
		{"0", TypeInteger, "number 0"},
		{"-0", TypeInteger, "number -0"},
		{"42", TypeInteger, "number 42"},
		{"-17", TypeInteger, "number -17"},
		{"9007199254740991", TypeInteger, "number 9007199254740991"},
		// D14 item 3's examples, and the rest of what the pattern refuses.
		{" 42", TypeInteger, `string " 42"`},
		{"007", TypeInteger, `string "007"`},
		{"1,5", TypeInteger, `string "1,5"`},
		{"yes", TypeInteger, `string "yes"`},
		{"42 ", TypeInteger, `string "42 "`},
		{"+1", TypeInteger, `string "+1"`},
		{"-", TypeInteger, `string "-"`},
		{"--1", TypeInteger, `string "--1"`},
		{"1.0", TypeInteger, `string "1.0"`},
		{"1e3", TypeInteger, `string "1e3"`},
		{"0x10", TypeInteger, `string "0x10"`},
		{"", TypeInteger, `string ""`},
		{"١٢", TypeInteger, `string "١٢"`},
		// After the boolean row: "1" is no boolean form.
		{"1", TypeBoolean | TypeInteger, "number 1"},
		// integer not in T.
		{"42", TypeString, `string "42"`},
		{"42", TypeBoolean, `string "42"`},
	})
}

// TestCoerceNumber is the third row: number in T and text matching RFC
// 8259 §6's number grammar.
func TestCoerceNumber(t *testing.T) {
	checkCoerce(t, []coerceCase{
		{"1.5", TypeNumber, "number 1.5"},
		{"-0.0", TypeNumber, "number -0.0"},
		{"12", TypeNumber, "number 12"},
		{"1e3", TypeNumber, "number 1e3"},
		{"1E+3", TypeNumber, "number 1E+3"},
		{"2.5e-7", TypeNumber, "number 2.5e-7"},
		{"1.", TypeNumber, `string "1."`},
		{".5", TypeNumber, `string ".5"`},
		{"01", TypeNumber, `string "01"`},
		{"+1", TypeNumber, `string "+1"`},
		{"1,5", TypeNumber, `string "1,5"`},
		{"NaN", TypeNumber, `string "NaN"`},
		{"Infinity", TypeNumber, `string "Infinity"`},
		{"0x10", TypeNumber, `string "0x10"`},
		{" 1", TypeNumber, `string " 1"`},
		{"", TypeNumber, `string ""`},
		// After the integer row: a non-integer text in an integer-or-number
		// column is a number.
		{"1.5", TypeInteger | TypeNumber, "number 1.5"},
		{"7", TypeInteger | TypeNumber, "number 7"},
		// number not in T.
		{"1.5", TypeInteger, `string "1.5"`},
		{"1.5", TypeString, `string "1.5"`},
	})
}

// TestCoerceString is the last row: whatever no other row takes is the
// text, as a string, whatever T holds; coercion never fails (D14 item 3).
func TestCoerceString(t *testing.T) {
	checkCoerce(t, []coerceCase{
		{"Acme Ltd", TypeString, `string "Acme Ltd"`},
		{"42", 0, `string "42"`},
		{"true", 0, `string "true"`},
		{"null", TypeNull, `string "null"`},
		{"{}", TypeObject, `string "{}"`},
		{"[1]", TypeArray, `string "[1]"`},
		{"", TypeString, `string ""`},
		// Dates and times stay strings (D14 item 4).
		{"2026-10-18", TypeString, `string "2026-10-18"`},
		{"2026-10-18", TypeInteger | TypeNumber, `string "2026-10-18"`},
		{"2026-10-18T09:00:00Z", TypeNumber, `string "2026-10-18T09:00:00Z"`},
		// Text that matches the grammar but is no binary64.
		{"1e400", TypeNumber, `string "1e400"`},
	})
}

// TestCoerceKeepsLiteral pins ADR-010 D5 item 4 false first: a coerced
// number keeps its text as its literal, so it is never re-rounded or
// re-spelt on its way to the validator or a template. An integer beyond
// 2^53−1 is still a number, its literal the text, for the reader's D8
// item 5 rule to find.
func TestCoerceKeepsLiteral(t *testing.T) {
	for _, c := range []struct {
		text  string
		types TypeSet
	}{
		{"1.50", TypeNumber},
		{"1.0", TypeNumber},
		{"1E3", TypeNumber},
		{"0.1000000000000000055511151231257827", TypeNumber},
		{"-0", TypeInteger},
		{"9007199254740993", TypeInteger},
		{"123456789012345678901234567890", TypeInteger | TypeNumber},
	} {
		v, err := Coerce(c.text, c.types)
		if err != nil {
			t.Fatalf("Coerce(%q): %v", c.text, err)
		}
		n, ok := v.(value.Number)
		if !ok || n.Literal() != c.text {
			t.Errorf("Coerce(%q, %s) = %s, want the number written %q", c.text, c.types, coerced(v), c.text)
		}
		if got := canonical(v); c.text == "1.50" && got != "1.5" {
			// Canonical form is RFC 8785's, which the literal does not
			// change: keeping the literal is not re-spelling the value.
			t.Errorf("canonical(%q) = %s", c.text, got)
		}
	}
	v, _ := Coerce("9007199254740993", TypeInteger)
	if n := v.(value.Number); !n.WrittenAsInteger() {
		t.Error("a coerced big integer does not say it was written as an integer")
	} else if _, safe := n.SafeInt(); safe {
		t.Error("9007199254740993 is a safe integer")
	}
}

// TestCoerceRefusesInvalidText: text that is not a sequence of scalar
// values is the one error, and no reader gives one.
func TestCoerceRefusesInvalidText(t *testing.T) {
	for _, types := range []TypeSet{0, TypeString, TypeInteger | TypeNumber | TypeBoolean} {
		if _, err := Coerce("a\xffb", types); !errors.Is(err, value.ErrNotScalarValues) {
			t.Errorf("Coerce(invalid UTF-8, %s): err = %v", types, err)
		}
		if _, ok, err := CoerceField("a\xffb", true, types); ok || !errors.Is(err, value.ErrNotScalarValues) {
			t.Errorf("CoerceField(invalid UTF-8, %s): ok %v, err %v", types, ok, err)
		}
	}
}

// TestCoerceField holds ADR-003 D11 item 5 beside D14's table: an unquoted
// empty field is absent and a quoted one the empty string, whatever the
// column's type set; every other field, quoted or not, is coerced.
func TestCoerceField(t *testing.T) {
	for _, types := range []TypeSet{0, TypeString, TypeInteger, TypeNumber, TypeBoolean, TypeInteger | TypeNumber | TypeBoolean | TypeString} {
		for _, c := range []struct {
			text   string
			quoted bool
			want   string
		}{
			{"", false, "absent"},
			{"", true, `string ""`},
		} {
			v, ok, err := CoerceField(c.text, c.quoted, types)
			if err != nil || ok != (c.want != "absent") || coerced(v) != c.want {
				t.Errorf("CoerceField(%q, quoted %v, %s) = %s, %v, %v; want %s", c.text, c.quoted, types, coerced(v), ok, err, c.want)
			}
		}
	}
	for _, c := range []struct {
		text   string
		quoted bool
		types  TypeSet
		want   string
	}{
		{"42", false, TypeInteger, "number 42"},
		{"42", true, TypeInteger, "number 42"},
		{"TRUE", true, TypeBoolean, "boolean true"},
		{" ", false, TypeInteger, `string " "`},
		{"007", true, TypeInteger, `string "007"`},
	} {
		v, ok, err := CoerceField(c.text, c.quoted, c.types)
		if err != nil || !ok || coerced(v) != c.want {
			t.Errorf("CoerceField(%q, quoted %v, %s) = %s, %v, %v; want %s", c.text, c.quoted, c.types, coerced(v), ok, err, c.want)
		}
	}
}

// TestCoerceCSVFields is the meeting with internal/inputs/csv: fields read
// by its reader reach CoerceField with their quotedness, so a quoted empty
// field and an unquoted one are distinct values, and each coerces against
// the column's declared type set as D14's table and D11 item 5 state.
func TestCoerceCSVFields(t *testing.T) {
	src := "id,remarks,count,done\n" +
		"1,,\"7\",TRUE\n" +
		"2,\"\",,no\n" +
		"\"3\",\"note, quoted\",007,\"\"\n"
	from := csv.Source{
		Input: finding.Subject{Kind: finding.SubjectInput, Name: "criteria"},
		At:    finding.Location{Origin: finding.OriginDataDir, File: "criteria.csv"},
	}
	table, fs := csv.Read([]byte(src), csv.Comma, from, limits.Defaults())
	if len(fs) > 0 {
		t.Fatalf("csv.Read: %v", fs)
	}
	types := []TypeSet{TypeInteger, TypeString | TypeNull, TypeInteger, TypeBoolean}
	want := [][]string{
		{"number 1", "absent", "number 7", "boolean true"},
		{"number 2", `string ""`, "absent", `string "no"`},
		{"number 3", `string "note, quoted"`, `string "007"`, `string ""`},
	}
	if len(table.Records) != len(want) {
		t.Fatalf("read %d records, want %d", len(table.Records), len(want))
	}
	for r, rec := range table.Records {
		for i, f := range rec.Fields {
			v, _, err := CoerceField(f.Text, f.Quoted, types[i])
			if err != nil {
				t.Fatalf("record %d field %d: %v", r, i, err)
			}
			if got := coerced(v); got != want[r][i] {
				t.Errorf("record %d, %s: %s (text %q, quoted %v), want %s", r, table.Header[i].Text, got, f.Text, f.Quoted, want[r][i])
			}
		}
	}
	// The two empty fields of the remarks column reached coercion as
	// different values.
	unquoted, quoted := table.Records[0].Fields[1], table.Records[1].Fields[1]
	if unquoted.Text != "" || quoted.Text != "" || unquoted.Quoted || !quoted.Quoted {
		t.Errorf("remarks fields = %+v and %+v, want an unquoted and a quoted empty field", unquoted, quoted)
	}
}
