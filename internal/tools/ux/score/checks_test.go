// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package score

import "testing"

// TestAnswerLineNames: a line of a C4 answer names a field by every word
// of its name, or by its title's number beside a word of that title, and
// by nothing less: a bare number, a range that stops short, or some of
// the name's words are not a field named.
func TestAnswerLineNames(t *testing.T) {
	contactName := []string{"primary_contact_name", "1.7 Primary contact person name"}
	secondEmail := []string{"second_contact_email", "1.11 Second contact person email"}
	detected := []string{"detected_at", "2.2 Date and time of detection of the ICT-related incident"}
	code := []string{"submitting_entity_code", "1.3 Identification code of the entity submitting the report"}
	for _, c := range []struct {
		line  string
		field []string
		want  bool
	}{
		{"Primary and second contact: name, email and telephone", contactName, true},
		{"Primary and second contact: name, email and telephone", secondEmail, true},
		{"- detection and classification timestamps", detected, true},
		{"**2.2** When the incident was detected, in UTC", detected, true},
		{"**1.3** Its identification code (its LEI)", code, true},
		{"**1.10–1.12** Second contact's name, email and telephone", secondEmail, true},
		{"**1.7-1.9** Primary contact's name, email and telephone", contactName, true},
		// Some of the name's words are not the field.
		{"Primary contact: email", contactName, false},
		{"Whom to contact", secondEmail, false},
		// A number with no word of its title is not the field.
		{"Fields 1.2 to 1.15", secondEmail, false},
		{"**1.3**", code, false},
		// A range that stops short of the number, or of another section.
		{"**1.7–1.9** Primary contact's details", secondEmail, false},
		{"**2.1–2.4** entity details and dates", code, false},
		// 1.1 is not read inside 1.12.
		{"**1.12** the code", []string{"x_y", "1.1 Something code"}, false},
	} {
		if got := answerLines(c.line)[0].names(c.field); got != c.want {
			t.Errorf("%q names %s: %v, want %v", c.line, c.field[0], got, c.want)
		}
	}
}
