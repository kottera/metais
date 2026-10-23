// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package schemafail

import (
	"fmt"
	"strings"
	"testing"
	"unicode/utf8"

	"metais.org/metais/internal/finding"
)

// hintSchema is the schema the hint cases are written in: an enum of a
// few short values and one of many, a format with a description, one
// with only a pattern beside it, and one with neither, and a string
// field, one of them writeOnly.
var hintSchema = func() string {
	var many []string
	for i := range 86 {
		many = append(many, fmt.Sprintf(`"%d.%d.%d"`, i/30+1, i/10%3+1, i%10+1))
	}
	return `{
  "properties": {
    "level": {"title": "Level", "enum": ["Supports", "Partially Supports", "Does Not Support", "Not Applicable", "Not Evaluated"]},
    "criterion": {"title": "Criterion", "enum": [` + strings.Join(many, ",") + `]},
    "website": {"title": "Website", "description": "The vendor's web address, beginning https:// or http://.\nAnything else is refused.", "format": "uri", "pattern": "^https?://"},
    "home": {"title": "Home", "format": "uri", "pattern": "^https?://"},
    "email": {"title": "Email", "format": "email"},
    "version": {"title": "Version", "type": "string"},
    "pin": {"title": "PIN", "type": "string", "writeOnly": true}
  }
}`
}()

// TestDeclaredHints is ADR-006 Amendment A4 items 2 to 4: a hint names the
// step its own failure needs. An enum's hint names the near value when
// there is one, every value when the few there are fit, and otherwise how
// many there are and the command that lists them; a format's hint says
// what the field's description, or else the pattern beside it, asks; and
// a type's hint, for a value an unquoted YAML scalar made a number or a
// boolean where a string is wanted, says to quote it, as it was written.
func TestDeclaredHints(t *testing.T) {
	d := newDocs(t, map[string]string{"vendor.schema.json": hintSchema})
	var many []string
	for i := range 86 {
		many = append(many, fmt.Sprintf(`"%d.%d.%d"`, i/30+1, i/10%3+1, i%10+1))
	}
	levels := []string{`"Supports"`, `"Partially Supports"`, `"Does Not Support"`, `"Not Applicable"`, `"Not Evaluated"`}
	written := map[string]string{"/version": "3.10", "/pin": "1234", "/long": strings.Repeat("9", 60) + ".5"}
	tg := target(d)
	tg.Written = func(p *finding.Path) (string, bool) {
		s, ok := written[p.String()]
		return s, ok
	}
	cases := []struct {
		name string
		f    Failure
		want string
	}{
		{"an enum's near value", at("enum", "/properties/level/enum", path("level"), Data{"allowed": levels, "got": "Supported"}),
			`Write "Supports", the allowed value nearest to the one written.`},
		{"a few values, none near", at("enum", "/properties/level/enum", path("level"), Data{"allowed": levels, "got": "Yes"}),
			`Write one of the allowed values: "Supports", "Partially Supports", "Does Not Support", "Not Applicable", "Not Evaluated".`},
		{"a value that is not a string", at("enum", "/properties/level/enum", path("level"), Data{"allowed": levels}),
			`Write one of the allowed values: "Supports", "Partially Supports", "Does Not Support", "Not Applicable", "Not Evaluated".`},
		{"many values, one near", at("enum", "/properties/criterion/enum", path("criterion"), Data{"allowed": many, "got": "1.2.99"}),
			`Write "1.2.9", the allowed value nearest to the one written.`},
		{"many values, none near", at("enum", "/properties/criterion/enum", path("criterion"), Data{"allowed": many, "got": "criterion one"}),
			"Write one of the 86 values the schema allows; `metais schema <package> vendor --fields` lists them all."},
		{"a format with a description", at("format", "/properties/website/format", path("website"), Data{"format": "uri"}),
			"Write a value in the uri format that fits the field's description: The vendor's web address, beginning https:// or http://."},
		{"a format with a pattern beside it", at("format", "/properties/home/format", path("home"), Data{"format": "uri"}),
			"Write a value in the uri format that matches the pattern ^https?://."},
		{"a format alone", at("format", "/properties/email/format", path("email"), Data{"format": "email"}),
			"Write a value in the email format."},
		{"an unquoted YAML number", at("type", "/properties/version/type", path("version"), Data{"want": []string{"string"}, "got": "number"}),
			`Put the value in quotes, as "3.10": YAML reads an unquoted 3.10 as a number.`},
		{"an unquoted YAML number where writeOnly", at("type", "/properties/pin/type", path("pin"), Data{"want": []string{"string"}, "got": "number"}),
			"Put the value in quotes, so that YAML reads it as a string rather than as a number."},
		{"a long unquoted YAML number", at("type", "/properties/long/type", path("long"), Data{"want": []string{"string", "null"}, "got": "number"}),
			"Put the value in quotes, so that YAML reads it as a string rather than as a number."},
		{"a number not from YAML", at("type", "/properties/name/type", path("name"), Data{"want": []string{"string"}, "got": "number"}),
			"Write a value of type string."},
		{"a YAML number where no string is wanted", at("type", "/properties/version/type", path("version"), Data{"want": []string{"object"}, "got": "number"}),
			"Write a value of type object."},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			fs := mapAll(t, tg, c.f)
			if len(fs) != 1 {
				t.Fatalf("%d findings: %v", len(fs), briefs(fs))
			}
			if got := fs[0].Hint(); got != c.want {
				t.Errorf("hint\n  %q\nwant\n  %q", got, c.want)
			}
			if n := utf8.RuneCountInString(fs[0].Hint()); n > 240 {
				t.Errorf("a hint of %d code points", n)
			}
		})
	}
	// The other families have no command that lists a schema's values.
	out := tg
	out.Family, out.Subject = FamilyOutput, finding.Subject{Kind: finding.SubjectOutput, Name: "report"}
	fs := mapAll(t, out, at("enum", "/properties/criterion/enum", path("criterion"), Data{"allowed": many, "got": "criterion one"}))
	if want := "Write one of the 86 values the schema's enum lists."; len(fs) != 1 || fs[0].Hint() != want {
		t.Errorf("an output's many values: %q, want %q", fs[0].Hint(), want)
	}
}
