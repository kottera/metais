// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package liquid

import "github.com/kottera/metais/internal/liquid/ast"

// Construct is one Liquid construct of a template's source, as the lexer
// finds it: an output "{{ … }}" or a tag "{% … %}", delimiters and
// whitespace-control markers included. Span is the whole token. Tag is
// the tag's name, and Markup what follows it, for a tag the lexer could
// name; both are empty for an output.
type Construct struct {
	Span   ast.Span
	Tag    string
	Markup string
}

// Constructs returns every Liquid construct of src, in source order, as
// the analysis tokenizer of ADR-007 D3 item 1 finds them: by the lexer
// alone, which knows nothing of what the text between them is, so a
// template that does not parse still has its constructs. A token that is
// not well formed is a construct too, running as far as the lexer runs
// it.
//
// The bytes the grammar reads as text are not constructs: the body of a
// raw, which is text, and the bodies of a comment and a doc, which are
// skipped. Their closing tags are constructs, and so are a comment's
// nested comment, endcomment and raw tags, which the grammar reads to
// find where the body ends. The constructs never overlap.
//
// The work is linear in the length of src.
func Constructs(src string) []Construct {
	c := &constructs{s: scanner{src: src}, x: newLineIndex(src)}
	for {
		t := c.s.next()
		if t.kind == tokEOF {
			return c.out
		}
		if t.kind == tokText {
			continue
		}
		c.add(t)
		if t.kind != tokTag || t.bad != "" {
			continue
		}
		switch t.name {
		case "raw":
			c.skipTo("endraw")
		case "doc":
			c.skipTo("enddoc")
		case "comment":
			c.comment()
		}
	}
}

// constructs collects a template's constructs.
type constructs struct {
	s   scanner
	x   *lineIndex
	out []Construct
}

func (c *constructs) add(t tok) {
	line, col := c.x.at(t.start)
	k := Construct{Span: ast.Span{Start: t.start, End: t.end, Line: line, Col: col}}
	if t.kind == tokTag && t.bad == "" {
		k.Tag, k.Markup = t.name, t.markup
	}
	c.out = append(c.out, k)
}

// skipTo consumes tokens up to the first that ends with a tag named end,
// wherever in the token that tag begins, as the parser finds the end of a
// raw or doc body, and adds that tag. It reports whether it found one.
func (c *constructs) skipTo(end string) bool {
	for {
		u := c.s.next()
		if u.kind == tokEOF {
			return false
		}
		if at, _, ok := endsWithTag(c.s.src[u.start:u.end], end); ok {
			c.add(tagToken(c.s.src, u.start+at, u.end))
			return true
		}
	}
}

// comment consumes a comment's body after its opening tag, adding the
// tags the parser reads in it, up to and including the endcomment that
// closes it (parser.comment).
func (c *constructs) comment() {
	depth := 1
	for {
		u := c.s.next()
		if u.kind == tokEOF {
			return
		}
		if u.kind != tokTag || u.bad != "" {
			continue
		}
		switch u.name {
		case "raw":
			c.add(u)
			if !c.skipTo("endraw") {
				return
			}
		case "comment":
			c.add(u)
			depth++
		case "endcomment":
			c.add(u)
			if depth--; depth == 0 {
				return
			}
		}
	}
}
