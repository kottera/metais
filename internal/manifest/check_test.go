// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package manifest

import (
	"bytes"
	"encoding/json"
	"slices"
	"strings"
	"testing"

	"metais.org/metais/internal/finding"
)

// TestLicenseGrammar is ADR-002 D4 item 11: SPDX 2.3.1 Annex D's grammar,
// with any idstring as an identifier, upper-case operators, and no
// DocumentRef-.
func TestLicenseGrammar(t *testing.T) {
	accepted := []string{
		"MIT",
		"Apache-2.0",
		"GPL-2.0+",
		"GPL-2.0-or-later",
		"LicenseRef-my.own-1",
		"MIT OR Apache-2.0",
		"MIT AND Apache-2.0 OR BSD-3-Clause",
		"(MIT OR Apache-2.0) AND BSD-2-Clause",
		"((MIT))",
		"GPL-2.0 WITH Classpath-exception-2.0",
		"GPL-2.0+ WITH Classpath-exception-2.0 OR MIT",
		"(GPL-2.0 WITH Classpath-exception-2.0) AND MIT",
		"  MIT   OR  Apache-2.0 ",
		"(MIT)AND(Apache-2.0)",
		"NotOnTheList-1.0",
	}
	for _, s := range accepted {
		if err := checkLicense(s); err != nil {
			t.Errorf("%q: refused (%s at %d)", s, err.reason, err.pos)
		}
	}
	refused := []struct {
		text   string
		reason string
		pos    int
	}{
		{"", reasonSyntax, 1},
		{"   ", reasonSyntax, 4},
		{"MIT and Apache-2.0", reasonOperatorCase, 5},
		{"MIT Or Apache-2.0", reasonOperatorCase, 5},
		{"GPL-2.0 with Classpath-exception-2.0", reasonOperatorCase, 9},
		{"DocumentRef-spdx-tool-1.2:LicenseRef-MIT-Style-2", reasonDocumentRef, 1},
		{"MIT OR DocumentRef-a:LicenseRef-b", reasonDocumentRef, 8},
		{"LicenseRef-mine+", reasonLicenseRefAdd, 1},
		{"(MIT OR Apache-2.0", reasonParentheses, 1},
		{"MIT OR Apache-2.0)", reasonParentheses, 18},
		{")MIT(", reasonParentheses, 1},
		{"()", reasonParentheses, 2},
		{"MIT OR", reasonSyntax, 7},
		{"OR MIT", reasonSyntax, 1},
		{"MIT Apache-2.0", reasonSyntax, 5},
		{"MIT AND AND Apache-2.0", reasonSyntax, 9},
		{"MIT WITH", reasonSyntax, 9},
		{"MIT WITH (X)", reasonSyntax, 10},
		{"MIT WITH X+", reasonSyntax, 10},
		{"(MIT) WITH X", reasonSyntax, 7},
		{"MIT/Apache", reasonSyntax, 1},
		{"MIT,", reasonSyntax, 1},
		{"MIT++", reasonSyntax, 1},
		{"+", reasonSyntax, 1},
		{"MIT\tOR Apache-2.0", reasonSyntax, 1},
		{"Licença", reasonSyntax, 1},
		{"É OR MIT AND", reasonSyntax, 1},
		{"MIT AND é", reasonSyntax, 9},
	}
	for _, c := range refused {
		err := checkLicense(c.text)
		if err == nil || err.reason != c.reason || err.pos != c.pos {
			t.Errorf("%q: %+v, want %s at %d", c.text, err, c.reason, c.pos)
		}
	}
}

// TestLicenseDepth: a deeply parenthesized license within the schema's
// 512 characters is parsed, not refused for its depth.
func TestLicenseDepth(t *testing.T) {
	s := strings.Repeat("(", 250) + "MIT" + strings.Repeat(")", 250)
	if err := checkLicense(s); err != nil {
		t.Fatalf("refused: %+v", err)
	}
	if err := checkLicense(s[1:]); err == nil || err.reason != reasonParentheses {
		t.Fatalf("an unbalanced one: %+v", err)
	}
}

// indented writes m as indented JSON, so each member is on its own line
// and a finding's line says which one it is about.
func indented(t testing.TB, m tree) []byte {
	t.Helper()
	b := encode(t, m)
	var out bytes.Buffer
	if err := json.Indent(&out, b, "", "  "); err != nil {
		t.Fatal(err)
	}
	return out.Bytes()
}

// lineOf returns the 1-based line of the first line of text holding s.
func lineOf(t testing.TB, text []byte, s string) int {
	t.Helper()
	for i, l := range strings.Split(string(text), "\n") {
		if strings.Contains(l, s) {
			return i + 1
		}
	}
	t.Fatalf("no line holds %q", s)
	return 0
}

// TestCheckFindings is each finding Check reports (ADR-002 D10 item 6):
// its code, class, subject, pointer, line, params and, for a duplicate,
// the first occurrence it relates.
func TestCheckFindings(t *testing.T) {
	isolate(t)
	m := base()
	set(t, m, "", "license", "MIT and Apache-2.0")
	set(t, m, "inputs/1", "name", "product")
	set(t, m, "outputs/4", "id", "report")
	set(t, m, "outputs/1/container/entries/0", "id", "logo")
	set(t, m, "outputs/5/container/entries/0", "for_each", "item.chapters")
	set(t, m, "outputs/5/container/entries/0", "path", "{{ item }}.xhtml")
	text := indented(t, m)
	root, pos := parse(t, text)
	fs := Check(root, Source{File: ManifestFile, Positions: pos})

	type want struct {
		code    finding.Code
		ptr     string
		line    string // text on the finding's line
		params  map[string]string
		related string // the related first occurrence's pointer, if any
		relLine string
	}
	wants := []want{
		{CodeLicenseExpression, "/license", `"license": "MIT and`, map[string]string{"reason": reasonOperatorCase}, "", ""},
		{CodeInputDuplicateName, "/inputs/1/name", `"name": "product",`, map[string]string{"name": "product"}, "/inputs/0/name", `"name": "product",`},
		{CodeOutputDuplicateID, "/outputs/4/id", `"id": "report"`, map[string]string{"id": "report"}, "/outputs/0/id", `"id": "report"`},
		{CodeOutputDuplicateID, "/outputs/1/container/entries/1/id", `"id": "logo"`, map[string]string{"id": "logo"}, "/outputs/1/container/entries/0/id", `"id": "logo"`},
		{CodeOutputNestedEach, "/outputs/5/container/entries/0/for_each", `"for_each": "item.chapters"`, nil, "", ""},
	}
	if len(fs) != len(wants) {
		t.Fatalf("Check reported %v", codes(fs))
	}
	for i, w := range wants {
		f := fs[i]
		if f.Code() != w.code || f.Class() != finding.ClassContent || f.Subject().Kind != finding.SubjectManifest {
			t.Errorf("finding %d: %s %s %s, want %s content manifest", i, f.Code(), f.Class(), f.Subject().Kind, w.code)
			continue
		}
		loc, _ := f.Location()
		if loc.Pointer != w.ptr || loc.File != ManifestFile || loc.Origin != finding.OriginPackage {
			t.Errorf("%s: location %+v, want %s in %s", w.code, loc, w.ptr, ManifestFile)
		}
		// Duplicates name the same text twice, so the line is found from
		// the occurrence's own place: the last line holding it for a
		// later duplicate, the first for its first occurrence.
		wantLine := lineOf(t, text, w.line)
		if w.related != "" {
			wantLine = lastLineOf(t, text, w.line, wantLine)
		}
		if loc.Line != wantLine {
			t.Errorf("%s: line %d, want %d", w.code, loc.Line, wantLine)
		}
		for k, v := range w.params {
			if got, _ := f.Param(k); func() bool { s, ok := got.Str(); return !ok || s != v }() {
				t.Errorf("%s: %s = %v, want %q", w.code, k, got, v)
			}
		}
		rel := f.Related()
		if w.related == "" {
			if len(rel) != 0 {
				t.Errorf("%s: related %v", w.code, rel)
			}
			continue
		}
		if len(rel) != 1 || rel[0].Note != noteFirst || rel[0].Location.Pointer != w.related || rel[0].Location.Line != lineOf(t, text, w.relLine) {
			t.Errorf("%s: related %+v, want %s on line %d", w.code, rel, w.related, lineOf(t, text, w.relLine))
		}
	}
	if p, _ := fs[0].Param("position"); func() bool { n, ok := p.Int(); return !ok || n != 5 }() {
		t.Errorf("license position = %v, want 5", p)
	}
}

// lastLineOf returns the line of the occurrence of s after line after.
func lastLineOf(t testing.TB, text []byte, s string, after int) int {
	t.Helper()
	lines := strings.Split(string(text), "\n")
	for i := after; i < len(lines); i++ {
		if strings.Contains(lines[i], s) {
			return i + 1
		}
	}
	t.Fatalf("no second line holds %q", s)
	return 0
}

// TestCheckWithoutPositions: a finding has a pointer and no line when the
// reader kept no positions.
func TestCheckWithoutPositions(t *testing.T) {
	isolate(t)
	m := base()
	set(t, m, "", "license", "MIT OR")
	root, _ := parse(t, encode(t, m))
	fs := Check(root, Source{File: ManifestFile})
	if len(fs) != 1 {
		t.Fatalf("Check reported %v", codes(fs))
	}
	if loc, _ := fs[0].Location(); loc.Line != 0 || loc.Pointer != "/license" {
		t.Errorf("location %+v", loc)
	}
}

// TestCheckShapes: Check passes over values of the wrong shape, which the
// schema reports, and judges the rest of the manifest.
func TestCheckShapes(t *testing.T) {
	isolate(t)
	m := base()
	set(t, m, "", "license", 42)
	set(t, m, "", "inputs", []any{"x", tree{"name": 1}, tree{"name": "a"}, tree{"name": "a"}})
	set(t, m, "", "outputs", []any{7, tree{"id": []any{}}, tree{"id": "a", "container": "zip"},
		tree{"id": "b", "for_each": "x", "container": tree{"entries": "no"}}, tree{"id": "a"}})
	root, _ := parse(t, encode(t, m))
	got := codes(Check(root, Source{File: ManifestFile}))
	if want := []finding.Code{CodeInputDuplicateName, CodeOutputDuplicateID}; !slices.Equal(got, want) {
		t.Fatalf("Check reported %v, want %v", got, want)
	}
}
