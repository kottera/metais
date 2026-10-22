// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package schemafail

import (
	"fmt"
	"strings"
	"testing"

	"metais.org/metais/internal/finding"
)

// sentinel is every string value of the instance the echo cases validate
// (ADR-006 *Conformance*, "No echo").
const sentinel = "SENTINEL-7f3a"

const echoSchema = `{
  "properties": {
    "code": {"title": "Code", "pattern": "^[A-Z]{2}$"},
    "email": {"title": "Email", "format": "email"},
    "kind": {"enum": ["SENTINEL-7f3b", "other"]},
    "card": {"anyOf": [{"title": "Visa", "pattern": "^4"}, {"title": "Email", "format": "email"}]}
  }
}`

// echoCase is a failure on a sentinel value, and the message the library
// studied writes for it, which quotes the value (ADR-006 D9 item 4).
type echoCase struct {
	f       Failure
	message string
}

func echoCases() []echoCase {
	return []echoCase{
		{at("pattern", "/properties/code/pattern", path("code"), Data{"pattern": "^[A-Z]{2}$"}),
			fmt.Sprintf("'%s' does not match pattern '^[A-Z]{2}$'", sentinel)},
		{at("format", "/properties/email/format", path("email"), Data{"format": "email"}),
			fmt.Sprintf("'%s' is not valid email: missing @", sentinel)},
		{at("enum", "/properties/kind/enum", path("kind"), Data{"allowed": []string{`"SENTINEL-7f3b"`, `"other"`}, "got": sentinel}),
			fmt.Sprintf("value must be one of 'SENTINEL-7f3b', 'other'; got '%s'", sentinel)},
		{at("anyOf", "/properties/card/anyOf", path("card"), Data{"branch_count": 2},
			at("pattern", "/properties/card/anyOf/0/pattern", path("card"), Data{"pattern": "^4"}),
			at("format", "/properties/card/anyOf/1/format", path("card"), Data{"format": "email"})),
			fmt.Sprintf("'%s' does not match pattern '^4'", sentinel)},
	}
}

// echoes names each finding whose JSON encoding holds the sentinel: in its
// message, hint, params, field, via, alternatives or related notes.
func echoes(fs []finding.Finding) []string {
	var bad []string
	for _, f := range fs {
		if j := string(finding.AppendJSON(nil, f)); strings.Contains(j, sentinel) {
			bad = append(bad, j)
		}
	}
	return bad
}

// messageMapping is the mapping ADR-006 D13 item 5 pins the echo test
// false against: one that reports with the library's messages. A finding
// has no free text, so it carries the message in the parameter that names
// the rule broken.
func messageMapping(tg Target, cases []echoCase) []finding.Finding {
	var out []finding.Finding
	for _, c := range cases {
		code := finding.Code(tg.Family.String() + "." + map[string]string{"pattern": "pattern", "format": "format", "enum": "const", "anyOf": "pattern"}[c.f.Keyword])
		param := map[string]string{"pattern": "pattern", "format": "format", "enum": "expected", "anyOf": "pattern"}[c.f.Keyword]
		ptr := c.f.Instance()
		at, _ := finding.ParsePath(ptr)
		loc := tg.Locate(at, false).WithPointer(ptr)
		out = append(out, finding.New(code, tg.Subject, &loc, finding.Params{param: finding.String(c.message)}))
	}
	return out
}

// TestNoEcho is ADR-006 D13 item 5's echo test: no finding of a failure
// on a string value repeats it, though the library's own message for each
// does. Pattern and format values, and the enum's allowed values, are the
// package's and are reported (D9 item 6); the suggestion names an allowed
// value, never the supplied one (D9 item 7).
func TestNoEcho(t *testing.T) {
	d := newDocs(t, map[string]string{"vendor.schema.json": echoSchema})
	var fs []Failure
	for _, c := range echoCases() {
		fs = append(fs, c.f)
	}
	got := mapAll(t, target(d), fs...)
	if len(got) != 4 {
		t.Fatalf("%d findings, want 4: %v", len(got), briefs(got))
	}
	for _, e := range echoes(got) {
		t.Errorf("a finding echoes the value: %s", e)
	}
	checkBriefs(t, "the package's values reported", got,
		`input.any_of "/card" {"branch_count":2} alts=[0(Visa): input.pattern "/card" {"pattern":"^4"} field=Visa| | 1(Email): input.format "/card" {"format":"email"} field=Email|]`,
		`input.pattern "/code" {"pattern":"^[A-Z]{2}$"} field=Code|`,
		`input.format "/email" {"format":"email"} field=Email|`,
		`input.enum "/kind" {"allowed":["\"SENTINEL-7f3b\"","\"other\""],"allowed_count":2,"suggestion":"SENTINEL-7f3b"}`,
	)
}

// TestNoEchoControl is the echo test's control, kept in every run: the
// detector catches the mapping that uses the library's messages, one
// finding for each case.
func TestNoEchoControl(t *testing.T) {
	d := newDocs(t, map[string]string{"vendor.schema.json": echoSchema})
	cases := echoCases()
	if n := len(echoes(messageMapping(target(d), cases))); n != len(cases) {
		t.Errorf("the detector caught %d of the %d findings that use the library's messages", n, len(cases))
	}
}
