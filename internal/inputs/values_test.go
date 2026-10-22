// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package inputs

import (
	"testing"

	"metais.org/metais/internal/finding"
	"metais.org/metais/internal/limits"
	"metais.org/metais/internal/manifest"
	"metais.org/metais/internal/schema"
	"metais.org/metais/internal/testenv"
)

// TestInputValuesCount is ADR-009 Amendment A4 item 1's count for the
// documents that decode to the least (W7-NB3): an empty YAML document
// decodes to null, one value, its root, as a document of only a comment
// does; a Markdown file's object is one value whether its frontmatter is
// absent, empty or only a comment, so each of those counts its object and
// its body.
func TestInputValuesCount(t *testing.T) {
	testenv.Home(t)
	l := layer(t, formsPkg)
	cases := []struct {
		name  string
		input string
		form  manifest.Form
		data  string
		want  int64
	}{
		{"an empty YAML document", "doc", manifest.FormYAML, "", 1},
		{"a YAML document of a comment", "doc", manifest.FormYAML, "# nothing\n", 1},
		{"a YAML document of one member", "doc", manifest.FormYAML, "a: 1\n", 2},
		{"Markdown without frontmatter", "page", manifest.FormMarkdown, "The body.\n", 2},
		{"Markdown with empty frontmatter", "page", manifest.FormMarkdown, "---\n---\nThe body.\n", 2},
		{"Markdown with frontmatter of a comment", "page", manifest.FormMarkdown, "---\n# nothing\n---\nThe body.\n", 2},
		{"Markdown with frontmatter of one field", "page", manifest.FormMarkdown, "---\na: 1\n---\nThe body.\n", 3},
	}
	lim := limits.Defaults()
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var in manifest.Input
			for _, i := range l.inputs {
				if i.Name == c.input {
					in = i
				}
			}
			values := lim.Counter(limits.InputValues)
			loc := finding.Location{Origin: finding.OriginDataDir, File: c.input}
			p := l.reader(in, lim, values).read([]byte(c.data), c.data, c.form, loc, "", []schema.Step{schema.Member(c.input)})
			if len(p.findings) > 0 || p.value == nil {
				t.Fatalf("%q: no value, findings:\n%s", c.data, describe(p.findings))
			}
			if got := values.Count(); got != c.want {
				t.Errorf("%q counts %d values, want %d", c.data, got, c.want)
			}
		})
	}
}
