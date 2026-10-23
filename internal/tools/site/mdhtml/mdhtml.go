// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

// Package mdhtml converts this repository's own Markdown, the
// specification's chapters and the agent skill, to the HTML fragments the
// site's package renders (ADR-013 D8 item 1). It is input preparation,
// outside the package and outside the binary: METAIS gains no Markdown
// filter, and the format's own reading of Markdown stays
// internal/markdown's.
//
// The conversion is goldmark's CommonMark with its table extension, the
// one module ADR-010 Amendment A10 admits for it. The text converted is
// this repository's, never a package's, so raw HTML passes through, as the
// specification's statement anchors need. Every heading gets an id, made
// the way GitHub makes the anchors the specification's links name.
package mdhtml

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
	"unicode"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/text"
)

// Heading is one heading of a document: its level, the id its element
// carries, and its plain text.
type Heading struct {
	Level int    `json:"level"`
	ID    string `json:"id"`
	Text  string `json:"text"`
}

// Doc is one converted document.
type Doc struct {
	// Title is the plain text of the document's first level-1 heading, or
	// "" when it has none.
	Title string
	// HTML is the document as an HTML fragment: the body's content, with no
	// html, head or body element.
	HTML string
	// Headings are every heading, in document order.
	Headings []Heading
}

// Convert converts src, CommonMark with tables, to a Doc. link maps the
// destination of every link to the one the fragment writes; nil leaves
// each as it is.
func Convert(src []byte, link func(dest string) string) (Doc, error) {
	md := goldmark.New(
		goldmark.WithExtensions(extension.Table),
		goldmark.WithParserOptions(parser.WithAutoHeadingID()),
		goldmark.WithRendererOptions(html.WithUnsafe()),
	)
	ctx := parser.NewContext(parser.WithIDs(&githubIDs{seen: map[string]int{}}))
	root := md.Parser().Parse(text.NewReader(src), parser.WithContext(ctx))
	var doc Doc
	err := ast.Walk(root, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		switch n := n.(type) {
		case *ast.Link:
			if link != nil {
				n.Destination = []byte(link(string(n.Destination)))
			}
		case *ast.Heading:
			id, _ := n.AttributeString("id")
			idText, _ := id.([]byte)
			h := Heading{Level: n.Level, ID: string(idText), Text: plainText(n, src)}
			if h.Level == 1 && doc.Title == "" {
				doc.Title = h.Text
			}
			doc.Headings = append(doc.Headings, h)
		}
		return ast.WalkContinue, nil
	})
	if err != nil {
		return Doc{}, err
	}
	var buf bytes.Buffer
	if err := md.Renderer().Render(&buf, src, root); err != nil {
		return Doc{}, fmt.Errorf("rendering the Markdown: %w", err)
	}
	doc.HTML = buf.String()
	return doc, nil
}

// plainText is the text n's inline content reads as, with every markup
// character removed: its text and code spans, and the text of its links
// and emphasis.
func plainText(n ast.Node, src []byte) string {
	var b strings.Builder
	var walk func(ast.Node)
	walk = func(n ast.Node) {
		for c := n.FirstChild(); c != nil; c = c.NextSibling() {
			switch c := c.(type) {
			case *ast.Text:
				b.Write(c.Segment.Value(src))
				if c.SoftLineBreak() || c.HardLineBreak() {
					b.WriteByte(' ')
				}
			case *ast.String:
				b.Write(c.Value)
			case *ast.RawHTML:
				// An anchor or other inline HTML is markup, not text.
			default:
				walk(c)
			}
		}
	}
	walk(n)
	return b.String()
}

// githubIDs makes heading ids the way GitHub makes a Markdown heading's
// anchor, which the specification's links between chapters name: the
// heading's text lower-cased, each letter, digit, mark, hyphen and
// underscore kept, each space made a hyphen, and every other character
// dropped; a repeated id gets -1, -2 and so on.
type githubIDs struct {
	seen map[string]int
}

func (g *githubIDs) Generate(value []byte, kind ast.NodeKind) []byte {
	var b strings.Builder
	for _, r := range strings.ToLower(string(value)) {
		switch {
		case r == ' ':
			b.WriteByte('-')
		case r == '-' || r == '_' || unicode.IsLetter(r) || unicode.IsDigit(r) || unicode.IsMark(r):
			b.WriteRune(r)
		}
	}
	id := b.String()
	if id == "" {
		id = "heading"
	}
	n, ok := g.seen[id]
	g.seen[id] = n + 1
	if ok {
		id += "-" + strconv.Itoa(n)
		g.seen[id]++
	}
	return []byte(id)
}

func (g *githubIDs) Put(value []byte) {
	g.seen[string(value)]++
}
