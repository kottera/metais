// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package liquid

import (
	"slices"
	"testing"

	"github.com/kottera/metais/internal/testenv"
)

// TestConstructs: every construct the lexer finds, with its delimiters and
// whitespace control, its line and column, and a tag's name and markup;
// a token that is not well formed is a construct too.
func TestConstructs(t *testing.T) {
	testenv.Home(t)
	src := "a {{- x | f -}} b\n{%- if y -%}é{% endif %}{{ z }"
	got := Constructs(src)
	type c struct {
		text      string
		line, col int
		tag, mark string
	}
	var have []c
	for _, k := range got {
		have = append(have, c{src[k.Span.Start:k.Span.End], k.Span.Line, k.Span.Col, k.Tag, k.Markup})
	}
	want := []c{
		{"{{- x | f -}}", 1, 3, "", ""},
		{"{%- if y -%}", 2, 1, "if", "y "},
		{"{% endif %}", 2, 14, "endif", ""},
		{"{{ z }", 2, 25, "", ""},
	}
	if !slices.Equal(have, want) {
		t.Errorf("got %+v\nwant %+v", have, want)
	}
}

// TestConstructsSkipTextBodies: a raw's body is text, and a comment's and
// a doc's bodies are skipped, so none holds a construct; the tags the
// grammar reads to find each body's end are constructs.
func TestConstructsSkipTextBodies(t *testing.T) {
	testenv.Home(t)
	for src, want := range map[string][]string{
		"{% raw %}{{ a }}{% if %}{% endraw %}{{ b }}":                                      {"{% raw %}", "{% endraw %}", "{{ b }}"},
		"{% raw %}{{ a {% endraw %}":                                                       {"{% raw %}", "{% endraw %}"},
		"{% doc %}{{ a }}{% enddoc %}{{ b }}":                                              {"{% doc %}", "{% enddoc %}", "{{ b }}"},
		"{% comment %}{{ a }}{% if %}{% comment %}{% endcomment %}{% endcomment %}{{ b }}": {"{% comment %}", "{% comment %}", "{% endcomment %}", "{% endcomment %}", "{{ b }}"},
		"{% comment %}{% raw %}{% endcomment %}{% endraw %}{% endcomment %}":               {"{% comment %}", "{% raw %}", "{% endraw %}", "{% endcomment %}"},
		"{% raw %}{{ a }}":         {"{% raw %}"},
		"{% comment %}{{ a }}":     {"{% comment %}"},
		"{% liquid\nrender 'x' %}": {"{% liquid\nrender 'x' %}"},
	} {
		var got []string
		for _, c := range Constructs(src) {
			got = append(got, src[c.Span.Start:c.Span.End])
		}
		if !slices.Equal(got, want) {
			t.Errorf("%q: got %q, want %q", src, got, want)
		}
	}
}
