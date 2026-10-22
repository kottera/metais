// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package ooxml

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"metais.org/metais/internal/finding"
	"metais.org/metais/internal/testenv"
)

// The main namespaces of ADR-007 D4 item 1.
const (
	nsW       = "http://schemas.openxmlformats.org/wordprocessingml/2006/main"
	nsS       = "http://schemas.openxmlformats.org/spreadsheetml/2006/main"
	nsA       = "http://schemas.openxmlformats.org/drawingml/2006/main"
	nsWStrict = "http://purl.oclc.org/ooxml/wordprocessingml/main"
	nsSStrict = "http://purl.oclc.org/ooxml/spreadsheetml/main"
	nsAStrict = "http://purl.oclc.org/ooxml/drawingml/main"
	nsOther   = "http://example.com/not-office"
)

// word is a WordprocessingML document part whose body is body.
func word(body string) string {
	return `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` + "\n" +
		`<w:document xmlns:w="` + nsW + `"><w:body>` + body + `</w:body></w:document>`
}

// split is a placeholder Word has split across two runs at a spelling
// mark, as ADR-007's Context describes it.
const split = `<w:p><w:r><w:t>{{ inputs.</w:t></w:r><w:proofErr w:type="spellStart"/><w:r><w:t>vendor.name }}</w:t></w:r><w:proofErr w:type="spellEnd"/></w:p>`

// scan scans one template, named document.xml.
func scan(t *testing.T, src string) []finding.Finding {
	t.Helper()
	return Scan([]Template{{Path: "document.xml", Text: []byte(src)}})
}

// codes lists the findings' codes.
func codes(fs []finding.Finding) []finding.Code {
	var out []finding.Code
	for _, f := range fs {
		out = append(out, f.Code())
	}
	return out
}

func str(t *testing.T, f finding.Finding, name string) string {
	t.Helper()
	v, ok := f.Param(name)
	s, isStr := v.Str()
	if !ok || !isStr {
		t.Fatalf("%s has no string parameter %s", f.Code(), name)
	}
	return s
}

func num(t *testing.T, f finding.Finding, name string) int64 {
	t.Helper()
	v, ok := f.Param(name)
	n, isNum := v.Int()
	if !ok || !isNum {
		t.Fatalf("%s has no number parameter %s", f.Code(), name)
	}
	return n
}

// want checks that fs is exactly the codes want.
func want(t *testing.T, fs []finding.Finding, want ...finding.Code) {
	t.Helper()
	if got := codes(fs); !slices.Equal(got, want) {
		t.Errorf("got %q, want %q", got, want)
	}
}

// D4 item 1: a template is scanned when its source declares one of the six
// main namespaces, whatever prefix it binds, and not otherwise.
func TestScannedWhenAMainNamespaceIsDeclared(t *testing.T) {
	testenv.Home(t)
	body := func(ns, prefix string) string {
		q := func(l string) string {
			if prefix == "" {
				return l
			}
			return prefix + ":" + l
		}
		decl := "xmlns"
		if prefix != "" {
			decl += ":" + prefix
		}
		return fmt.Sprintf(`<%s %s="%s"><%s><%s>{{ a.</%s><%s>b }}</%s></%s></%s>`,
			q("root"), decl, ns, q("p"), q("t"), q("t"), q("t"), q("t"), q("p"), q("root"))
	}
	for _, c := range []struct {
		ns, prefix string
	}{
		{nsW, "w"}, {nsWStrict, "w"}, {nsA, "a"}, {nsAStrict, "a"},
		{nsW, "word"}, {nsA, ""}, {nsW, ""},
	} {
		want(t, scan(t, body(c.ns, c.prefix)), CodeSplitConstruct)
	}
	// SpreadsheetML's paragraph containers are si and is, and its text
	// element is t.
	for _, ns := range []string{nsS, nsSStrict} {
		for _, prefix := range []string{"x", ""} {
			src := strings.ReplaceAll(body(ns, prefix), "p>", "si>")
			want(t, scan(t, src), CodeSplitConstruct)
		}
	}
	// Any other namespace, and none, is not scanned.
	want(t, scan(t, body(nsOther, "w")))
	want(t, scan(t, `<w:p><w:t>{{ a.</w:t><w:t>b }}</w:t></w:p>`))
	want(t, scan(t, strings.Replace(word(split), nsW, nsW+"/", 1)))
}

// D4 item 1 and the Context: detection resolves namespaces, not prefixes.
// A w: prefix bound to another namespace is not WordprocessingML, and an
// x: prefix bound to SpreadsheetML is.
func TestDetectionResolvesNamespacesNotPrefixes(t *testing.T) {
	testenv.Home(t)
	src := `<w:root xmlns:w="` + nsOther + `" xmlns:o="` + nsW + `">` +
		`<w:p><w:t>{{ a.</w:t><w:t>b }}</w:t></w:p></w:root>`
	want(t, scan(t, src))
	src = `<sst xmlns="` + nsS + `"><si><r><t>{{ a.</t></r><r><rPr><b/></rPr><t>b }}</t></r></si></sst>`
	want(t, scan(t, src), CodeSplitConstruct)
	// PresentationML declares xmlns:a on individual elements.
	src = `<p:sld xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main"><p:txBody>` +
		`<a:p xmlns:a="` + nsA + `"><a:r><a:t>{{ a.</a:t></a:r><a:r><a:t>b }}</a:t></a:r></a:p>` +
		`<a:p><a:r><a:t>{{ c.</a:t></a:r><a:r><a:t>d }}</a:t></a:r></a:p></p:txBody></p:sld>`
	fs := scan(t, src)
	want(t, fs, CodeSplitConstruct)
	// The second a:p is outside the declaration's scope.
	if len(fs) == 1 && str(t, fs[0], "text") != "{{ a.b }}" {
		t.Errorf("found %q, want the construct inside the declaration's scope", str(t, fs[0], "text"))
	}
}

// fragment is a partial with no namespace declaration of its own, as a
// table row or a list item usually is.
const fragment = `<w:p><w:r><w:t>{{ row.</w:t></w:r><w:r><w:rPr><w:b/></w:rPr><w:t>name }}</w:t></w:r></w:p>`

// D4 item 1: every template rendered from a scanned one, directly or
// transitively, is scanned, with the prefix bindings of the root element
// of the template that renders it.
func TestPartialsInheritTheRootBindings(t *testing.T) {
	testenv.Home(t)
	scanAll := func(ts ...Template) map[string][]finding.Code {
		got := map[string][]finding.Code{}
		for _, f := range Scan(ts) {
			got[f.Subject().Name] = append(got[f.Subject().Name], f.Code())
		}
		return got
	}
	root := word(`{% for row in inputs.rows %}{% render 'row.xml', row: row %}{% endfor %}`)
	got := scanAll(Template{"document.xml", []byte(root)}, Template{"row.xml", []byte(fragment)})
	if !slices.Equal(got["templates/row.xml"], []finding.Code{CodeSplitConstruct}) || len(got) != 1 {
		t.Errorf("a partial rendered from a document part: got %v", got)
	}
	// Transitively, through a partial that declares nothing either, and
	// through a liquid tag's render line.
	got = scanAll(
		Template{"document.xml", []byte(root)},
		Template{"row.xml", []byte(`<w:p>{% liquid` + "\n  render \"cell.xml\"\n" + `%}</w:p>`)},
		Template{"cell.xml", []byte(fragment)},
	)
	if !slices.Equal(got["templates/cell.xml"], []finding.Code{CodeSplitConstruct}) || len(got) != 1 {
		t.Errorf("a partial rendered from a partial: got %v", got)
	}
	// A fragment no scanned template renders is not scanned.
	got = scanAll(Template{"document.xml", []byte(word(""))}, Template{"row.xml", []byte(fragment)})
	if len(got) != 0 {
		t.Errorf("a partial nothing renders: got %v", got)
	}
	// Only the root element's declarations are inherited.
	inner := `<o:root xmlns:o="` + nsOther + `"><w:body xmlns:w="` + nsW + `">{% render 'row.xml' %}</w:body></o:root>`
	got = scanAll(Template{"document.xml", []byte(inner)}, Template{"row.xml", []byte(fragment)})
	if len(got) != 0 {
		t.Errorf("a declaration below the root element: got %v", got)
	}
	// The partial's own declaration of the prefix is the one in scope.
	own := `<w:p xmlns:w="` + nsOther + `"><w:r><w:t>{{ row.</w:t></w:r><w:r><w:t>name }}</w:t></w:r></w:p>`
	got = scanAll(Template{"document.xml", []byte(root)}, Template{"row.xml", []byte(own)})
	if len(got) != 0 {
		t.Errorf("a partial rebinding the prefix: got %v", got)
	}
	// A partial's root rebinding the prefix stops the inheritance for the
	// partials it renders too: they inherit its binding, not the one above
	// it.
	mid := `<w:p xmlns:w="` + nsOther + `">{% render 'cell.xml' %}</w:p>`
	got = scanAll(Template{"document.xml", []byte(root)}, Template{"row.xml", []byte(mid)}, Template{"cell.xml", []byte(fragment)})
	if len(got) != 0 {
		t.Errorf("a partial rendered from a partial rebinding the prefix: got %v", got)
	}
	// A partial rendered from two templates binding its prefix
	// differently is scanned under both.
	sheet := `<x:worksheet xmlns:x="` + nsS + `" xmlns:w="` + nsOther + `">{% render 'row.xml' %}</x:worksheet>`
	got = scanAll(Template{"sheet.xml", []byte(sheet)}, Template{"document.xml", []byte(root)}, Template{"row.xml", []byte(fragment)})
	if !slices.Equal(got["templates/row.xml"], []finding.Code{CodeSplitConstruct}) || len(got) != 1 {
		t.Errorf("a partial rendered under two bindings: got %v", got)
	}
}

// D4 item 2: the text elements are WordprocessingML t, delText and
// instrText, SpreadsheetML t and DrawingML t; content in any other
// element is markup.
func TestTextElements(t *testing.T) {
	testenv.Home(t)
	for _, l := range []string{"t", "delText", "instrText"} {
		src := word(fmt.Sprintf(`<w:p><w:r><w:%s>{{ a.</w:%s></w:r><w:r><w:%s>b }}</w:%s></w:r></w:p>`, l, l, l, l))
		want(t, scan(t, src), CodeSplitConstruct)
	}
	// In SpreadsheetML and DrawingML, only t.
	for _, c := range []struct{ ns, l string }{{nsS, "t"}, {nsA, "t"}, {nsS, "delText"}, {nsA, "instrText"}, {nsS, "text"}} {
		src := fmt.Sprintf(`<x:root xmlns:x="%s"><x:p><x:si><x:%s>{{ a.</x:%s><x:%s>b }}</x:%s></x:si></x:p></x:root>`, c.ns, c.l, c.l, c.l, c.l)
		var w []finding.Code
		if c.l == "t" {
			w = []finding.Code{CodeSplitConstruct}
		}
		want(t, scan(t, src), w...)
	}
	// Character data in an element that is not a text element is markup:
	// a construct there overlaps no text content.
	want(t, scan(t, word(`<w:p><w:r><w:rPr>{{ a.</w:rPr><w:rPr>b }}</w:rPr></w:r></w:p>`)))
}

// D4 item 2: the paragraph containers are WordprocessingML p, DrawingML p,
// and SpreadsheetML si and is; a delimiter is only matched within one.
func TestParagraphContainers(t *testing.T) {
	testenv.Home(t)
	for _, c := range []struct {
		ns, container string
		found         bool
	}{
		{nsW, "p", true}, {nsA, "p", true}, {nsS, "si", true}, {nsS, "is", true},
		{nsW, "si", false}, {nsS, "p", false}, {nsA, "r", false},
	} {
		src := fmt.Sprintf(`<x:root xmlns:x="%s"><x:%s><x:r><x:t>{</x:t></x:r><x:r><x:t>{ a }}</x:t></x:r></x:%s></x:root>`, c.ns, c.container, c.container)
		var w []finding.Code
		if c.found {
			w = []finding.Code{CodeSplitDelimiter}
		}
		t.Run(c.ns+" "+c.container, func(t *testing.T) { want(t, scan(t, src), w...) })
	}
	// Two paragraphs are two containers.
	want(t, scan(t, word(`<w:p><w:r><w:t>{</w:t></w:r></w:p><w:p><w:r><w:t>{ a }}</w:t></w:r></w:p>`)))
}

// D4 item 3: a Liquid construct inside a start tag is opaque attribute
// text, so a ">" inside it does not end the tag.
func TestConstructsInStartTagsAreOpaque(t *testing.T) {
	testenv.Home(t)
	src := word(`<w:p {% if a > b %}w:rsidR="00A1"{% endif %}><w:r><w:t {% if a > b %}xml:space="preserve"{% endif %}>{{ x }}</w:t></w:r></w:p>`)
	want(t, scan(t, src))
	// A construct inside an attribute value, holding the value's quote.
	src = word(`<w:p><w:r><w:t w:val="{{ "a>" }}">{{ x }}</w:t></w:r></w:p>`)
	want(t, scan(t, src))
	// The same tag with the construct outside the tag is read as markup
	// ending at the ">", so the text element's content begins inside it.
	src = word(`<w:p><w:r><w:t>{% if a > b %}{{ x</w:t></w:r><w:r><w:t> }}{% endif %}</w:t></w:r></w:p>`)
	want(t, scan(t, src), CodeSplitConstruct)
}

// D4 item 3: a construct's range includes its delimiters and
// whitespace-control markers, and comes from the lexer, so a construct
// that does not parse, or is never closed, is still a construct.
func TestConstructRangesFromTheLexer(t *testing.T) {
	testenv.Home(t)
	want(t, scan(t, word(`<w:p><w:r><w:t>{{- a -}}</w:t></w:r></w:p>`)))
	fs := scan(t, word(`<w:p><w:r><w:t>{{- a.</w:t></w:r><w:r><w:t>b -}}</w:t></w:r></w:p>`))
	want(t, fs, CodeSplitConstruct)
	if len(fs) == 1 && str(t, fs[0], "text") != "{{- a.b -}}" {
		t.Errorf("text %q", str(t, fs[0], "text"))
	}
	// A tag never closed runs to the end of the source.
	want(t, scan(t, word(`<w:p><w:r><w:t>{% if a </w:t></w:r><w:r><w:t>b</w:t></w:r></w:p>`)), CodeSplitConstruct)
	// A raw body is text, not a construct.
	want(t, scan(t, word(`<w:p><w:r><w:t>{% raw %}{{ a.</w:t></w:r><w:r><w:t>b }}{% endraw %}</w:t></w:r></w:p>`)))
}

// D4 item 4: a construct overlapping the content of at least one text
// element and not contained in exactly one is split_construct.
func TestSplitConstruct(t *testing.T) {
	testenv.Home(t)
	want(t, scan(t, word(split)), CodeSplitConstruct)
	// Contained in one text element: no finding.
	want(t, scan(t, word(`<w:p><w:r><w:t xml:space="preserve">Dear {{ inputs.vendor.name }}, {% if a %}hi{% endif %}</w:t></w:r></w:p>`)))
	// A construct lying wholly in markup — a for between table rows —
	// overlaps no text content and is correct.
	want(t, scan(t, word(`<w:tbl>{% for row in inputs.rows %}<w:tr><w:tc><w:p><w:r><w:t>{{ row.name }}</w:t></w:r></w:p></w:tc></w:tr>{% endfor %}</w:tbl>`)))
	// Half in one text element and half in markup.
	fs := scan(t, word(`<w:p><w:r><w:t>{{ a</w:t></w:r>}}</w:p>`))
	want(t, fs, CodeSplitConstruct)
	if len(fs) == 1 && num(t, fs[0], "text_elements") != 1 {
		t.Errorf("text_elements %d, want 1", num(t, fs[0], "text_elements"))
	}
	// Two constructs in two text elements are each contained.
	want(t, scan(t, word(`<w:p><w:r><w:t>{{ a }}</w:t></w:r><w:r><w:t>{{ b }}</w:t></w:r></w:p>`)))
}

// D4 item 5: a delimiter broken between the last character of one text
// element's content and the first of the next non-empty one in the same
// paragraph container is split_delimiter.
func TestSplitDelimiter(t *testing.T) {
	testenv.Home(t)
	for _, c := range []struct{ a, b, d string }{
		{"{", "{ a }}", "{{"},
		{"{", "% if a %}x{% endif %}", "{%"},
		{"{{ a }", "}", "}}"},
		{"{% if a %", "}x{% endif %}", "%}"},
	} {
		src := word(`<w:p><w:r><w:t>` + c.a + `</w:t></w:r><w:r><w:t></w:t></w:r><w:r><w:t>` + c.b + `</w:t></w:r></w:p>`)
		fs := scan(t, src)
		var got []finding.Finding
		for _, f := range fs {
			if f.Code() == CodeSplitDelimiter {
				got = append(got, f)
			}
		}
		if len(got) != 1 {
			t.Errorf("%q + %q: got %q", c.a, c.b, codes(fs))
			continue
		}
		if d := str(t, got[0], "delimiter"); d != c.d {
			t.Errorf("%q + %q: delimiter %q, want %q", c.a, c.b, d, c.d)
		}
		if n := num(t, got[0], "text_elements"); n != 2 {
			t.Errorf("%q + %q: text_elements %d, want 2", c.a, c.b, n)
		}
		if s := str(t, got[0], "text"); s != c.a+c.b {
			t.Errorf("%q + %q: text %q", c.a, c.b, s)
		}
	}
	// No delimiter: none.
	want(t, scan(t, word(`<w:p><w:r><w:t>{</w:t></w:r><w:r><w:t> { a }}</w:t></w:r></w:p>`)))
	want(t, scan(t, word(`<w:p><w:r><w:t>a}</w:t></w:r><w:r><w:t>{b</w:t></w:r></w:p>`)))
}

// D4 item 6: the finding is located at the construct's first character,
// and gives the construct's visible text, with entities decoded and
// markup removed, and the number of text elements it spans.
func TestFindingDetail(t *testing.T) {
	testenv.Home(t)
	src := word("\n<w:p><w:r><w:t>Né {{ a | append: &quot;&lt;</w:t></w:r><w:r><w:t>&#x3e;&quot;</w:t></w:r><w:r><w:t><![CDATA[ & ]]>| b }}</w:t></w:r></w:p>")
	fs := scan(t, src)
	want(t, fs, CodeSplitConstruct)
	if len(fs) != 1 {
		return
	}
	f := fs[0]
	if s := str(t, f, "text"); s != `{{ a | append: "<>" & | b }}` {
		t.Errorf("text %q", s)
	}
	if n := num(t, f, "text_elements"); n != 3 {
		t.Errorf("text_elements %d, want 3", n)
	}
	loc, ok := f.Location()
	if !ok || loc.File != "templates/document.xml" || loc.Origin != finding.OriginPackage || loc.Line != 3 || loc.Column != 19 {
		t.Errorf("location %+v", loc)
	}
	if s := f.Subject(); s.Kind != finding.SubjectTemplate || s.Name != "templates/document.xml" {
		t.Errorf("subject %+v", s)
	}
	// split_delimiter is located at the delimiter's first character.
	fs = scan(t, word("<w:p><w:r><w:t>ab\r\nc{</w:t></w:r><w:r><w:t>{ x }}</w:t></w:r></w:p>"))
	want(t, fs, CodeSplitDelimiter)
	if len(fs) == 1 {
		if loc, _ := fs[0].Location(); loc.Line != 3 || loc.Column != 2 {
			t.Errorf("delimiter location %d:%d, want 3:2", loc.Line, loc.Column)
		}
	}
}

// A construct's text is read with every reference XML 1.0 predefines,
// and character references in decimal and in hexadecimal of either case,
// decoded: the visible text of ADR-007 D4 item 6.
func TestReferencesDecoded(t *testing.T) {
	testenv.Home(t)
	fs := scan(t, word("<w:p><w:r><w:t>{{ a | append: &quot;&lt;&gt;&amp;&apos;</w:t></w:r><w:r><w:t>&#62;&#x3E;&#x3e;&quot; | b }}</w:t></w:r></w:p>"))
	want(t, fs, CodeSplitConstruct)
	if len(fs) == 1 {
		if s := str(t, fs[0], "text"); s != `{{ a | append: "<>&'>>>" | b }}` {
			t.Errorf("text %q", s)
		}
	}
}

// D4 item 6: the hint names the cause from the elements found between the
// pieces, and then says what to do.
func TestCauseHints(t *testing.T) {
	testenv.Home(t)
	r := func(t string) string { return `<w:r><w:t>` + t + `</w:t></w:r>` }
	const a, b = "{{ inputs.", "vendor }}"
	for _, c := range []struct {
		name, src, cause string
	}{
		{"proofErr", word(`<w:p>` + r(a) + `<w:proofErr w:type="spellStart"/>` + r(b) + `</w:p>`), "spelling or grammar marks"},
		{"err=1", `<a:p xmlns:a="` + nsA + `"><a:r><a:rPr lang="en-US"/><a:t>` + a + `</a:t></a:r><a:r><a:rPr lang="en-US" err="1"/><a:t>` + b + `</a:t></a:r></a:p>`, "spelling or grammar marks"},
		{"err=1 first", `<a:p xmlns:a="` + nsA + `"><a:r><a:rPr err="1"/><a:t>` + a + `</a:t></a:r><a:r><a:rPr/><a:t>` + b + `</a:t></a:r></a:p>`, "spelling or grammar marks"},
		{"ins", word(`<w:p>` + r(a) + `<w:ins w:id="1" w:author="A">` + r(b) + `</w:ins></w:p>`), "tracked changes (accept or reject them first)"},
		{"del", word(`<w:p>` + r(a) + `<w:del w:id="1"><w:r><w:delText>x</w:delText></w:r></w:del>` + r(b) + `</w:p>`), "tracked changes (accept or reject them first)"},
		{"delText", word(`<w:p><w:del w:id="1"><w:r><w:delText>` + a + `</w:delText></w:r></w:del>` + r(b) + `</w:p>`), "tracked changes (accept or reject them first)"},
		{"bookmark", word(`<w:p>` + r(a) + `<w:bookmarkStart w:id="0" w:name="x"/>` + r(b) + `<w:bookmarkEnd w:id="0"/></w:p>`), "a bookmark or comment"},
		{"comment", word(`<w:p>` + r(a) + `<w:commentRangeStart w:id="0"/>` + r(b) + `</w:p>`), "a bookmark or comment"},
		{"commentEnd", word(`<w:p>` + r(a) + `<w:commentRangeEnd w:id="0"/>` + r(b) + `</w:p>`), "a bookmark or comment"},
		{"fldChar", word(`<w:p>` + r(a) + `<w:r><w:fldChar w:fldCharType="begin"/></w:r>` + r(b) + `</w:p>`), "a field"},
		{"instrText", word(`<w:p>` + r(a) + `<w:r><w:instrText>` + b + `</w:instrText></w:r></w:p>`), "a field"},
		{"formatting", word(`<w:p>` + r(a) + `<w:r><w:rPr><w:b/></w:rPr><w:t>` + b + `</w:t></w:r></w:p>`), "a change of run properties (formatting)"},
		{"first in order", word(`<w:p>` + r(a) + `<w:bookmarkStart w:id="0" w:name="x"/><w:proofErr w:type="spellStart"/>` + r(b) + `</w:p>`), "spelling or grammar marks"},
	} {
		t.Run(c.name, func(t *testing.T) {
			fs := scan(t, c.src)
			want(t, fs, CodeSplitConstruct)
			if len(fs) != 1 {
				return
			}
			if got := str(t, fs[0], "cause"); got != c.cause {
				t.Errorf("cause %q, want %q", got, c.cause)
			}
			h := fs[0].Hint()
			if !strings.Contains(h, c.cause) || !strings.Contains(h, "type it again in one go") || !strings.Contains(h, "edit the XML") {
				t.Errorf("hint %q", h)
			}
		})
	}
	// The same causes for a split delimiter.
	fs := scan(t, word(`<w:p><w:r><w:t>{</w:t></w:r><w:ins w:id="1"><w:r><w:t>{ a }}</w:t></w:r></w:ins></w:p>`))
	want(t, fs, CodeSplitDelimiter)
	if len(fs) == 1 && str(t, fs[0], "cause") != "tracked changes (accept or reject them first)" {
		t.Errorf("delimiter cause %q", str(t, fs[0], "cause"))
	}
}

// D4 item 7: both findings are errors.
func TestSeverityIsError(t *testing.T) {
	testenv.Home(t)
	for _, f := range scan(t, word(split+`<w:p><w:r><w:t>{</w:t></w:r><w:r><w:t>{ a }}</w:t></w:r></w:p>`)) {
		if f.Severity() != finding.SeverityError || f.Class() != finding.ClassContent {
			t.Errorf("%s: %s %s", f.Code(), f.Severity(), f.Class())
		}
	}
}

// The tokenizer is lenient: unbalanced markup, which Liquid's branches
// routinely leave, and markup of every other kind, never stop the scan.
func TestLenientMarkup(t *testing.T) {
	testenv.Home(t)
	// A start tag in each branch, one end tag.
	src := word(`<w:p>{% if a %}<w:r><w:t>{% else %}<w:r><w:t xml:space="preserve">{% endif %}{{ x }}</w:t></w:r></w:p>` + split)
	want(t, scan(t, src), CodeSplitConstruct)
	// An end tag nothing opened, a comment, a processing instruction and
	// a lone "<".
	src = word(`</w:x><!-- c --><?pi ?><w:p><w:r><w:t>a < b {{ x }}</w:t></w:r></w:p>` + split)
	want(t, scan(t, src), CodeSplitConstruct)
	// A comment inside a text element breaks its content.
	want(t, scan(t, word(`<w:p><w:r><w:t>{{ a<!-- x -->b }}</w:t></w:r></w:p>`)), CodeSplitConstruct)
	// Unterminated markup ends at the end of the source.
	for _, tail := range []string{"<!-- x", "<![CDATA[ x", "<w:t a=\"x", "<?pi", "<w:t"} {
		want(t, scan(t, word(split)+tail), CodeSplitConstruct)
	}
}
