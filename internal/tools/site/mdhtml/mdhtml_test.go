// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package mdhtml

import (
	"slices"
	"strings"
	"testing"
)

func TestConvert(t *testing.T) {
	src := "# 2. Container\n\n" +
		"<a id=\"mts-pkg-001\"></a>**MTS-PKG-001** A processor MUST read it. See [the manifest](03-manifest.md#31-the-file).\n\n" +
		"## 2.1 The `zip` profile — strictly\n\n" +
		"| Field | Value |\n|---|---|\n| a | b |\n\n" +
		"## 2.1 The `zip` profile — strictly\n\n" +
		"### Ünïcode, *emphasis* and [a link](x.md)\n"
	var seen []string
	doc, err := Convert([]byte(src), func(dest string) string {
		seen = append(seen, dest)
		return "/to/" + dest
	})
	if err != nil {
		t.Fatal(err)
	}
	if doc.Title != "2. Container" {
		t.Errorf("Title = %q, want the first level-1 heading's text", doc.Title)
	}
	want := []Heading{
		{1, "2-container", "2. Container"},
		{2, "21-the-zip-profile--strictly", "2.1 The zip profile — strictly"},
		{2, "21-the-zip-profile--strictly-1", "2.1 The zip profile — strictly"},
		{3, "ünïcode-emphasis-and-a-link", "Ünïcode, emphasis and a link"},
	}
	if !slices.Equal(doc.Headings, want) {
		t.Errorf("Headings = %v\nwant %v", doc.Headings, want)
	}
	if !slices.Equal(seen, []string{"03-manifest.md#31-the-file", "x.md"}) {
		t.Errorf("link saw %q, want every link's destination once, in order", seen)
	}
	for _, frag := range []string{
		`<h1 id="2-container">2. Container</h1>`,
		`<a id="mts-pkg-001"></a><strong>MTS-PKG-001</strong>`,
		`<a href="/to/03-manifest.md#31-the-file">the manifest</a>`,
		`<h2 id="21-the-zip-profile--strictly">2.1 The <code>zip</code> profile — strictly</h2>`,
		`<table>`, `<td>b</td>`,
	} {
		if !strings.Contains(doc.HTML, frag) {
			t.Errorf("the HTML lacks %s:\n%s", frag, doc.HTML)
		}
	}
	for _, frag := range []string{"<html", "<body"} {
		if strings.Contains(doc.HTML, frag) {
			t.Errorf("the HTML is not a fragment: it holds %s", frag)
		}
	}
}

func TestConvertNoTitle(t *testing.T) {
	doc, err := Convert([]byte("## Only a section\n\ntext\n"), nil)
	if err != nil {
		t.Fatal(err)
	}
	if doc.Title != "" || len(doc.Headings) != 1 || doc.Headings[0].ID != "only-a-section" {
		t.Errorf("Convert = %+v, want no title and the one heading", doc)
	}
}

// TestConvertDeterministic converts the same text twice: the ids of one
// conversion never leak into the next.
func TestConvertDeterministic(t *testing.T) {
	src := []byte("# A\n\n# A\n")
	a, err := Convert(src, nil)
	if err != nil {
		t.Fatal(err)
	}
	b, err := Convert(src, nil)
	if err != nil {
		t.Fatal(err)
	}
	if a.HTML != b.HTML || a.HTML != "<h1 id=\"a\">A</h1>\n<h1 id=\"a-1\">A</h1>\n" {
		t.Errorf("two conversions gave %q and %q", a.HTML, b.HTML)
	}
}
