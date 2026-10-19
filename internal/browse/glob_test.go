// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package browse

import (
	"errors"
	"slices"
	"strings"
	"testing"
)

// TestGlobGrammar is ADR-008 D16 item 8's table for the glob grammar of
// D5: each wildcard, classes and ranges, escapes, ** as a whole segment,
// the dot-segment rule (RULINGS R2-121), a literal directory selecting
// what lies beneath it, and case-sensitivity.
func TestGlobGrammar(t *testing.T) {
	cases := []struct {
		glob string
		path string
		want bool
	}{
		{"*", "metais.yaml", true},
		{"*", "inputs/a.json", false},
		{"inputs/*", "inputs/a.json", true},
		{"inputs/*.json", "inputs/a.yaml", false},
		{"inputs/a*b", "inputs/ab", true},
		{"inputs/a*b", "inputs/axxb", true},
		{"inputs/a*b", "inputs/axxbx", false},
		{"inputs/*a*", "inputs/bab", true},
		{"?.md", "a.md", true},
		{"?.md", "ab.md", false},
		{"?.md", "é.md", true},
		{"[ab].md", "b.md", true},
		{"[ab].md", "c.md", false},
		{"[^ab].md", "c.md", true},
		{"[^ab].md", "a.md", false},
		{"[a-c].md", "b.md", true},
		{"[a-c].md", "d.md", false},
		{"[]a].md", "].md", true},
		{"[a-].md", "-.md", true},
		{`\*.md`, "*.md", true},
		{`\*.md`, "a.md", false},
		{`[\]].md`, "].md", true},
		{"**", "metais.yaml", true},
		{"**", "references/wcag/1.md", true},
		{"**/*.md", "a.md", true},
		{"**/*.md", "references/wcag/1.md", true},
		{"references/**", "references/wcag/1.md", true},
		{"references/**/1.md", "references/1.md", true},
		{"references/**/1.md", "references/a/b/1.md", true},
		{"a/**/b/**/c", "a/x/b/y/z/c", true},
		{"a/**/b/**/c", "a/x/y/c", false},
		{"README.MD", "README.md", false},
		{"*.md", "A.MD", false},
		// The dot rule: no wildcard enters a segment that begins with ".".
		{"*", ".gitattributes", false},
		{"**/*.md", ".compiled/x.md", false},
		{"**", ".compiled/manifest.json", false},
		{"**", "references/.hidden.md", false},
		{"?compiled/**", ".compiled/schema.json", false},
		{"[.]compiled/**", ".compiled/schema.json", false},
		{".compiled/**", ".compiled/schema.json", true},
		{".compiled/*", ".compiled/schema.json", true},
		{".*", ".gitattributes", true},
		{`\.compiled/*`, ".compiled/schema.json", true},
		{"references/.*", "references/.hidden.md", true},
		// A glob with no wildcard: the entry, or a directory and what lies
		// beneath it, its own dot segments included.
		{"references", "references/a/b.md", true},
		{"references/", "references/a/b.md", true},
		{"references", "references.md", false},
		{"references/a", "references/a", true},
		{".compiled", ".compiled/schema.json", true},
		{"references", "references/.hidden.md", true},
		{".", "metais.yaml", true},
		{".", ".compiled/schema.json", true},
		{`refer\ences`, "references/x", true},
	}
	for _, c := range cases {
		g, err := CompileGlob(c.glob)
		if err != nil {
			t.Errorf("CompileGlob(%q): %v", c.glob, err)
			continue
		}
		if got := g.Match(c.path); got != c.want {
			t.Errorf("glob %q against %q = %v, want %v", c.glob, c.path, got, c.want)
		}
	}
}

// TestGlobRefusals is the grammar's refusals: ** that does not stand
// alone as a segment, an unclosed or reversed class, a trailing escape,
// an empty segment, and a glob that is not UTF-8.
func TestGlobRefusals(t *testing.T) {
	for glob, reason := range map[string]string{
		"a**":           GlobDoubleStar,
		"**.md":         GlobDoubleStar,
		"x/a**b/c":      GlobDoubleStar,
		"***":           GlobDoubleStar,
		"[ab":           GlobClass,
		"[a/b]":         GlobClass,
		"[z-a]":         GlobClass,
		"[]":            GlobClass,
		`a\`:            GlobEscape,
		`[a\`:           GlobEscape,
		"":              GlobEmpty,
		"/a":            GlobEmpty,
		"a//b":          GlobEmpty,
		"/":             GlobEmpty,
		"a\xffb":        GlobEncoding,
		"**/x/**/[b-a]": GlobClass,
	} {
		_, err := CompileGlob(glob)
		var ge *GlobError
		if !errors.As(err, &ge) || ge.Reason != reason {
			t.Errorf("CompileGlob(%q) = %v, want the reason %s", glob, err, reason)
		}
		if err != nil && strings.Contains(err.Error(), glob) && glob != "" {
			t.Errorf("CompileGlob(%q)'s error repeats the glob: %v", glob, err)
		}
	}
}

// TestGlobSelect is a selection in path order, and many ** segments over a
// deep path, which the memo keeps from being exponential.
func TestGlobSelect(t *testing.T) {
	entries := []string{".compiled/schema.json", "inputs/a.json", "inputs/b.yaml", "metais.yaml", "references/x.md"}
	g, err := CompileGlob("**/*.json")
	if err != nil {
		t.Fatal(err)
	}
	if got := g.Select(entries); !slices.Equal(got, []string{"inputs/a.json"}) {
		t.Errorf("Select = %q", got)
	}
	deep := strings.Repeat("a/", 31) + "b"
	g, err = CompileGlob(strings.Repeat("**/", 20) + "c")
	if err != nil {
		t.Fatal(err)
	}
	if g.Match(deep) {
		t.Error("a glob ending c matched a path ending b")
	}
}
