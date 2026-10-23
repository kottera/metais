// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package template

import (
	"os"
	"strings"
	"testing"

	"metais.org/metais/internal/finding"
	"metais.org/metais/internal/jsonx"
	"metais.org/metais/internal/limits"
	"metais.org/metais/internal/liquid"
	"metais.org/metais/internal/liquid/value"
	"metais.org/metais/internal/testenv"
)

// TestWritingValuesForReaders: chapter 13 §13.9's template, rendered in
// the dialect a package's templates are rendered in over its report input,
// writes exactly the lines the section says it writes. The three are read
// from the chapter itself, so the section and the filters cannot drift
// apart.
func TestWritingValuesForReaders(t *testing.T) {
	testenv.Home(t)
	text, err := os.ReadFile("../../spec/13-authoring.md")
	if err != nil {
		t.Fatal(err)
	}
	_, section, ok := strings.Cut(string(text), "\n## 13.9 Writing values for readers\n")
	if !ok {
		t.Fatal("spec/13-authoring.md has no §13.9")
	}
	report, src, want := fence(t, section, "json"), fence(t, section, "liquid"), fence(t, section, "text")

	doc, fs := jsonx.Parse([]byte(`{"inputs": {"report": `+report+`}}`), jsonx.Source{Input: finding.Subject{Kind: finding.SubjectInput, Name: "report"}}, limits.Defaults())
	if len(fs) > 0 {
		t.Fatalf("§13.9's report: %v", fs)
	}
	tpl, errs := liquid.Parse(Dialect(), liquid.Source{Name: "report.md", Text: []byte(src)}, liquid.ParseLimits{})
	if len(errs) > 0 {
		t.Fatalf("§13.9's template: %v", errs[0])
	}
	env := &liquid.Env{Dialect: Dialect(), Globals: liquid.Data(doc.Value).(value.Object), Meter: liquid.NewMeter(liquid.MeterLimits{})}
	var b strings.Builder
	if err := liquid.Render(&b, tpl, env); err != nil {
		t.Fatalf("§13.9's template: %v", err)
	}
	if got := b.String(); got != want {
		t.Errorf("§13.9's template writes\n%s\nthe section says\n%s", got, want)
	}
	if n := strings.Count(want, "\n"); n != 4 {
		t.Errorf("§13.9 says its template writes %d lines, not 4", n)
	}
}

// fence returns the body of the first block fenced as lang in text, each
// line ending in a line feed.
func fence(t *testing.T, text, lang string) string {
	t.Helper()
	_, rest, ok := strings.Cut(text, "\n```"+lang+"\n")
	if !ok {
		t.Fatalf("§13.9 has no %s block", lang)
	}
	body, _, ok := strings.Cut(rest, "\n```\n")
	if !ok {
		t.Fatalf("§13.9's %s block is not closed", lang)
	}
	return body + "\n"
}
