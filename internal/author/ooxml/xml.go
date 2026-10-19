// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package ooxml

import (
	"strings"

	"github.com/kottera/metais/internal/liquid"
)

// The lenient XML tokenizer of ADR-007 D4 item 3. It reads a template's
// source as XML that may have Liquid in it, and never refuses: what is not
// markup is character data, and a construct that runs to the end of the
// source ends there.
//
// It knows the template's Liquid constructs, and treats any construct that
// begins inside a start tag as opaque attribute text, so a tag such as
// <w:p {% if x %}w:rsidR="1"{% endif %}> is one start tag whatever the
// construct holds. A construct that begins in character data is not
// skipped: that is where a split construct holds the markup that split it.

type evKind uint8

const (
	evStart  evKind = iota + 1 // a start tag, or an empty-element tag when selfClose
	evEnd                      // an end tag
	evText                     // character data
	evCDATA                    // a CDATA section, whose content is character data
	evMarkup                   // a comment, a processing instruction or a declaration
)

// attr is one attribute of a start tag: its name split at the first colon,
// and its value as written, without its quotes and not decoded.
type attr struct {
	prefix, local, value string
}

// event is one token. start and end are the token's bytes; for character
// data and a CDATA section, [textStart, textEnd) is the content. A tag's
// name is split at its first colon into prefix and local.
type event struct {
	kind               evKind
	start, end         int
	textStart, textEnd int
	prefix, local      string
	selfClose          bool
	attrs              []attr
}

// tokenizer yields a source's events in order. Its work is linear in the
// source's length: every byte is examined a bounded number of times.
type tokenizer struct {
	src  string
	off  int
	cons []liquid.Construct
	ci   int // the first construct that may begin at or after off
	ev   event
}

func newTokenizer(src string, cons []liquid.Construct) *tokenizer {
	return &tokenizer{src: src, cons: cons}
}

// next returns the next event, and false at the end of the source. The
// event, and its attrs, are valid until the next call.
func (z *tokenizer) next() (*event, bool) {
	src, i := z.src, z.off
	if i >= len(src) {
		return nil, false
	}
	z.ev = event{attrs: z.ev.attrs[:0]}
	ev := &z.ev
	if src[i] == '<' {
		if z.markup(i) {
			return ev, true
		}
	}
	// Character data runs to the next "<" that begins markup.
	j := i + 1
	for {
		k := strings.IndexByte(src[j:], '<')
		if k < 0 {
			j = len(src)
			break
		}
		j += k
		if isMarkupStart(src, j) {
			break
		}
		j++
	}
	*ev = event{kind: evText, start: i, end: j, textStart: i, textEnd: j, attrs: ev.attrs}
	z.off = j
	return ev, true
}

// isMarkupStart reports whether the "<" at i begins markup: a start or end
// tag, a comment, a CDATA section, a processing instruction or a
// declaration. Any other "<" is character data, as a lenient reader of a
// template must take it.
func isMarkupStart(src string, i int) bool {
	if i+1 >= len(src) {
		return false
	}
	switch c := src[i+1]; {
	case c == '!' || c == '?':
		return true
	case c == '/':
		return i+2 < len(src) && isNameStart(src[i+2])
	default:
		return isNameStart(c)
	}
}

// markup reads the markup at i into z.ev, when "<" at i begins any.
func (z *tokenizer) markup(i int) bool {
	src, ev := z.src, &z.ev
	if !isMarkupStart(src, i) {
		return false
	}
	switch {
	case strings.HasPrefix(src[i:], "<!--"):
		ev.kind, ev.start, ev.end = evMarkup, i, endAfter(src, i+4, "-->")
	case strings.HasPrefix(src[i:], "<![CDATA["):
		// An unclosed section's content runs to the end of the source.
		end := endAfter(src, i+9, "]]>")
		ev.kind, ev.start, ev.end = evCDATA, i, end
		ev.textStart, ev.textEnd = i+9, end
		if strings.HasSuffix(src[:end], "]]>") && end-3 >= i+9 {
			ev.textEnd = end - 3
		}
	case src[i+1] == '?':
		ev.kind, ev.start, ev.end = evMarkup, i, endAfter(src, i+2, "?>")
	case src[i+1] == '!':
		ev.kind, ev.start, ev.end = evMarkup, i, endAfter(src, i+2, ">")
	case src[i+1] == '/':
		n := nameEnd(src, i+2)
		ev.kind, ev.start, ev.end = evEnd, i, endAfter(src, n, ">")
		ev.prefix, ev.local = splitName(src[i+2 : n])
	default:
		z.startTag(i)
	}
	z.off = ev.end
	return true
}

// startTag reads the start tag at i, whose name begins at i+1.
func (z *tokenizer) startTag(i int) {
	src, ev := z.src, &z.ev
	n := nameEnd(src, i+1)
	ev.kind, ev.start = evStart, i
	ev.prefix, ev.local = splitName(src[i+1 : n])
	j := n
	for j < len(src) {
		if e := z.construct(j); e > j {
			j = e
			continue
		}
		switch c := src[j]; {
		case c == '>':
			ev.end = j + 1
			return
		case c == '/' && j+1 < len(src) && src[j+1] == '>':
			ev.end, ev.selfClose = j+2, true
			return
		case isSpace(c) || !isNameStart(c):
			j++
		default:
			j = z.attribute(j)
		}
	}
	ev.end = len(src)
}

// attribute reads the attribute whose name begins at j, and returns where
// the tokenizer goes on. An attribute with no value is recorded with an
// empty one; a quoted value runs to its closing quote, skipping the
// constructs that begin inside it.
func (z *tokenizer) attribute(j int) int {
	src := z.src
	n := nameEnd(src, j)
	a := attr{}
	a.prefix, a.local = splitName(src[j:n])
	k := n
	for k < len(src) && isSpace(src[k]) {
		k++
	}
	if k >= len(src) || src[k] != '=' {
		z.ev.attrs = append(z.ev.attrs, a)
		return n
	}
	k++
	for k < len(src) && isSpace(src[k]) {
		k++
	}
	if k < len(src) && (src[k] == '"' || src[k] == '\'') {
		q, v := src[k], k+1
		e := v
		for e < len(src) && src[e] != q {
			if c := z.construct(e); c > e {
				e = c
				continue
			}
			e++
		}
		a.value = src[v:min(e, len(src))]
		z.ev.attrs = append(z.ev.attrs, a)
		return min(e+1, len(src))
	}
	e := k
	for e < len(src) && !isSpace(src[e]) && src[e] != '>' {
		e++
	}
	a.value = src[k:e]
	z.ev.attrs = append(z.ev.attrs, a)
	return e
}

// construct returns the end of the construct that begins at j, or j when
// none does. The tokenizer's offsets only grow, so the constructs are
// passed over once.
func (z *tokenizer) construct(j int) int {
	for z.ci < len(z.cons) && z.cons[z.ci].Span.Start < j {
		z.ci++
	}
	if z.ci < len(z.cons) && z.cons[z.ci].Span.Start == j {
		return z.cons[z.ci].Span.End
	}
	return j
}

// endAfter returns the offset just past the first close at or after i, or
// the end of src when there is none.
func endAfter(src string, i int, close string) int {
	if i > len(src) {
		return len(src)
	}
	k := strings.Index(src[i:], close)
	if k < 0 {
		return len(src)
	}
	return i + k + len(close)
}

// nameEnd returns the end of the name that begins at i.
func nameEnd(src string, i int) int {
	for i < len(src) && isNameByte(src[i]) {
		i++
	}
	return i
}

// splitName splits a qualified name at its first colon.
func splitName(name string) (prefix, local string) {
	if p, l, ok := strings.Cut(name, ":"); ok {
		return p, l
	}
	return "", name
}

// isNameStart reports whether b may begin an XML name: an ASCII letter,
// "_", ":", or any byte of a non-ASCII character, which the lenient reader
// accepts without checking XML's ranges.
func isNameStart(b byte) bool {
	return b == '_' || b == ':' || 'a' <= b && b <= 'z' || 'A' <= b && b <= 'Z' || b >= 0x80
}

// isNameByte reports whether b may continue an XML name.
func isNameByte(b byte) bool {
	return isNameStart(b) || b == '-' || b == '.' || '0' <= b && b <= '9'
}

// isSpace reports whether b is XML white space.
func isSpace(b byte) bool { return b == ' ' || b == '\t' || b == '\n' || b == '\r' }
