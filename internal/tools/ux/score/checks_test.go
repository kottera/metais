// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package score

import "testing"

// TestAnswerLineNames: a line of a C4 answer names a field by every word
// of its name, or by its title's number beside a word of that title, and
// by nothing less: a bare number, a range that stops short, or some of
// the name's words are not a field named. A word of the line may be a
// word of the name's end, as phone is of telephone.
func TestAnswerLineNames(t *testing.T) {
	contactName := []string{"primary_contact_name", "1.7 Primary contact person name"}
	secondEmail := []string{"second_contact_email", "1.11 Second contact person email"}
	detected := []string{"detected_at", "2.2 Date and time of detection of the ICT-related incident"}
	code := []string{"submitting_entity_code", "1.3 Identification code of the entity submitting the report"}
	phone := []string{"primary_contact_telephone", "1.9 Primary contact person telephone"}
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
		// A word of five letters or more that ends a word of the name is
		// that word, plural or not (W3-10).
		{"**Primary and second contact:** name, email and phone for each.", phone, true},
		{"Primary contact: phones must be international", phone, true},
		// Some of the name's words are not the field.
		{"Primary contact: email", contactName, false},
		{"Whom to contact", secondEmail, false},
		{"Primary and second contact: name and email for each", phone, false},
		{"Phones must be in international format", phone, false},
		// A shorter end, or a word that is not an end, is not the word.
		{"Primary contact: name, email and tel", phone, false},
		{"Primary contact: name, email and one fax", phone, false},
		{"Primary contact: name, email and telephones", phone, true},
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

// TestScriptRunEndsItsCall: the harness reports one status for a Bash
// call, its last command's, so a test run of the script that ends a call
// the harness reports as an error has failed, whatever its output lists.
// Only a run that other commands follow is judged by render's listing
// (W3-1). The probe is c5-sonnet-renders-before-an-error with its line-35
// call reduced to the script's run, its output and error kept.
func TestScriptRunEndsItsCall(t *testing.T) {
	f := fixtureNamed(t, "c5-sonnet-renders-before-an-error")
	for _, c := range []struct {
		name string
		edit func([]string) []string
		pass bool
	}{
		{"as recorded", func(l []string) []string { return l }, true},
		{"the run ends its call", func(l []string) []string {
			l[35] = withCommand(t, l[35], "./render-report.sh ../ex/inputs.json ../ex/reports")
			return l
		}, false},
	} {
		o, err := Score(probe(t, f, c.edit))
		if err != nil {
			t.Fatal(err)
		}
		if res := ruleResult(t, o, RuleOutcome); res.Pass != c.pass {
			t.Errorf("%s: outcome %+v, want pass %v", c.name, res, c.pass)
		}
	}
}

// TestFieldWordEndsAtFiveLetters: a word of the line is read as the end
// of a field's word only from five letters, so a shorter word that ends
// it does not name the field (W3-10).
func TestFieldWordEndsAtFiveLetters(t *testing.T) {
	phone := []string{"primary_contact_telephone", "1.9 Primary contact person telephone"}
	for _, c := range []struct {
		line string
		want bool
	}{
		{"Primary contact: phone", true},
		{"Primary contact: hone", false},
		{"Primary contact: one", false},
		{"Primary contact: ne", false},
	} {
		if got := answerLines(c.line)[0].names(phone); got != c.want {
			t.Errorf("%q names %s: %v, want %v", c.line, phone[0], got, c.want)
		}
	}
}
