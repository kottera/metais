// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package ooxml

import (
	"cmp"
	"slices"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/kottera/metais/internal/finding"
	"github.com/kottera/metais/internal/liquid"
)

// cause is a set of the reasons ADR-007 D4 item 6 names for a run ending
// inside a construct, found in the markup between its pieces.
type cause uint8

const (
	causeSpelling cause = 1 << iota // w:proofErr, or a DrawingML run property err="1"
	causeTracked                    // w:ins, w:del or w:delText
	causeBookmark                   // w:bookmarkStart, w:bookmarkEnd, w:commentRangeStart or w:commentRangeEnd
	causeField                      // w:fldChar or w:instrText
	causeKinds    = 4
)

// The cause parameter's values, in the order the hint prefers them: item
// 6's list, with a change of run properties when nothing else is found.
var causeNames = [...]struct {
	c    cause
	name string
}{
	{causeSpelling, "spelling or grammar marks"},
	{causeTracked, "tracked changes (accept or reject them first)"},
	{causeBookmark, "a bookmark or comment"},
	{causeField, "a field"},
}

// causeFormatting is the cause when the markup holds none of the others.
const causeFormatting = "a change of run properties (formatting)"

// name is the cause parameter for the causes in c.
func (c cause) name() string {
	for _, n := range causeNames {
		if c&n.c != 0 {
			return n.name
		}
	}
	return causeFormatting
}

// frame is one open element.
type frame struct {
	key       qname
	decls     []string // the prefixes its start tag declares
	id        int      // unique within the scan
	text      bool     // a text element (D4 item 2)
	cause     cause    // a text element's own cause: delText or instrText
	container int      // the id of the innermost open paragraph container, or -1
	run       int      // the stack index of the innermost open DrawingML run, or -1
	errMark   bool     // a DrawingML run whose properties carry err="1"
}

type qname struct{ prefix, local string }

// piece is one stretch of a text element's content: the character data,
// or a CDATA section's content, between two pieces of markup inside it.
type piece struct {
	start, end int
	elem       int // the text element's id
	container  int // the innermost paragraph container's id, or -1
	cdata      bool
	cause      cause
}

// causeAt is a piece of markup that names a cause, at its offset.
type causeAt struct {
	off int
	c   cause
}

// scanner scans one template.
type scanner struct {
	t      *template
	file   string
	base   bindings
	ns     map[string][]family // the in-template declarations in scope
	open   map[qname]int       // how many elements of each name are open
	stack  []frame
	nextID int
	pieces []piece
	causes []causeAt
	// counts[k][i] is how many of causes[:i] name the cause 1<<k, so the
	// causes between two offsets are found in logarithmic time however
	// the containers nest.
	counts [causeKinds][]int
}

// located is a finding with the offset it is located at, for ordering.
type located struct {
	off int
	f   finding.Finding
}

// scanTemplate scans t, whose prefixes not bound in its own source resolve
// through base.
func scanTemplate(t *template, base bindings) []finding.Finding {
	s := &scanner{t: t, file: TemplatesDir + t.path, base: base, ns: map[string][]family{}, open: map[qname]int{}}
	z := newTokenizer(t.src, t.cons)
	for ev, ok := z.next(); ok; ev, ok = z.next() {
		switch ev.kind {
		case evStart:
			s.start(ev)
		case evEnd:
			s.end(ev)
		case evText, evCDATA:
			s.text(ev)
		}
	}
	for k := range causeKinds {
		s.counts[k] = make([]int, len(s.causes)+1)
		for i, c := range s.causes {
			s.counts[k][i+1] = s.counts[k][i]
			if c.c&(1<<k) != 0 {
				s.counts[k][i+1]++
			}
		}
	}
	found := append(s.splitConstructs(), s.splitDelimiters()...)
	slices.SortStableFunc(found, func(a, b located) int { return cmp.Compare(a.off, b.off) })
	out := make([]finding.Finding, len(found))
	for i, l := range found {
		out[i] = l.f
	}
	return out
}

// resolve returns the vocabularies prefix is bound to where the scanner
// stands.
func (s *scanner) resolve(prefix string) family {
	if prefix == "xml" || prefix == "xmlns" {
		return 0
	}
	if st := s.ns[prefix]; len(st) > 0 {
		return st[len(st)-1]
	}
	return s.base[prefix]
}

func (s *scanner) start(ev *event) {
	var decls []string
	for _, a := range ev.attrs {
		if p, ok := declared(a); ok {
			s.ns[p] = append(s.ns[p], families[decode(a.value)])
			decls = append(decls, p)
		}
	}
	f := s.resolve(ev.prefix)
	if c := markupCause(f, ev); c != 0 {
		s.causes = append(s.causes, causeAt{ev.start, c})
	}
	parent := frame{container: -1, run: -1}
	if len(s.stack) > 0 {
		parent = s.stack[len(s.stack)-1]
	}
	if f&famA != 0 && ev.local == "rPr" && parent.run >= 0 && errMarked(ev) {
		s.stack[parent.run].errMark = true
	}
	if ev.selfClose {
		s.undeclare(decls)
		return
	}
	fr := frame{key: qname{ev.prefix, ev.local}, decls: decls, id: s.nextID, container: parent.container, run: parent.run}
	s.nextID++
	switch l := ev.local; {
	case f&famW != 0 && (l == "t" || l == "delText" || l == "instrText"), f&(famS|famA) != 0 && l == "t":
		fr.text = true
		switch {
		case f&famW != 0 && l == "delText":
			fr.cause = causeTracked
		case f&famW != 0 && l == "instrText":
			fr.cause = causeField
		}
	case f&(famW|famA) != 0 && l == "p", f&famS != 0 && (l == "si" || l == "is"):
		fr.container = fr.id
	case f&famA != 0 && l == "r":
		fr.run = len(s.stack)
	}
	s.stack = append(s.stack, fr)
	s.open[fr.key]++
}

// end closes the innermost open element of the end tag's name, and every
// element opened inside it. An end tag that matches no open element is
// ignored, as Liquid's branches routinely leave the XML unbalanced.
func (s *scanner) end(ev *event) {
	if c := markupCause(s.resolve(ev.prefix), ev); c != 0 {
		s.causes = append(s.causes, causeAt{ev.start, c})
	}
	key := qname{ev.prefix, ev.local}
	if s.open[key] == 0 {
		return
	}
	for {
		fr := s.stack[len(s.stack)-1]
		s.stack = s.stack[:len(s.stack)-1]
		s.open[fr.key]--
		s.undeclare(fr.decls)
		if fr.key == key {
			return
		}
	}
}

func (s *scanner) undeclare(decls []string) {
	for _, p := range decls {
		st := s.ns[p]
		s.ns[p] = st[:len(st)-1]
	}
}

// text records character data as a piece when the innermost open element
// is a text element.
func (s *scanner) text(ev *event) {
	if len(s.stack) == 0 || ev.textEnd <= ev.textStart {
		return
	}
	fr := s.stack[len(s.stack)-1]
	if !fr.text {
		return
	}
	p := piece{start: ev.textStart, end: ev.textEnd, elem: fr.id, container: fr.container, cdata: ev.kind == evCDATA, cause: fr.cause}
	if fr.run >= 0 && s.stack[fr.run].errMark {
		p.cause |= causeSpelling
	}
	s.pieces = append(s.pieces, p)
}

// markupCause returns the cause a tag names (D4 item 6), resolved to the
// vocabularies f.
func markupCause(f family, ev *event) cause {
	if f&famA != 0 && ev.local == "rPr" && ev.kind == evStart && errMarked(ev) {
		return causeSpelling
	}
	if f&famW == 0 {
		return 0
	}
	switch ev.local {
	case "proofErr":
		return causeSpelling
	case "ins", "del", "delText":
		return causeTracked
	case "bookmarkStart", "bookmarkEnd", "commentRangeStart", "commentRangeEnd":
		return causeBookmark
	case "fldChar", "instrText":
		return causeField
	}
	return 0
}

// errMarked reports whether a DrawingML run-properties tag marks its run
// as misspelled: an unqualified err attribute that is true.
func errMarked(ev *event) bool {
	for _, a := range ev.attrs {
		if a.prefix == "" && a.local == "err" {
			v := strings.TrimSpace(decode(a.value))
			return v == "1" || v == "true"
		}
	}
	return false
}

// causesIn returns the causes named by markup in [from, to).
func (s *scanner) causesIn(from, to int) cause {
	i := sort.Search(len(s.causes), func(i int) bool { return s.causes[i].off >= from })
	j := sort.Search(len(s.causes), func(i int) bool { return s.causes[i].off >= to })
	var c cause
	for k := range causeKinds {
		if s.counts[k][j] > s.counts[k][i] {
			c |= 1 << k
		}
	}
	return c
}

// splitConstructs reports each construct that overlaps the content of at
// least one text element and is not contained in the content of exactly
// one (D4 item 4). The constructs and the pieces are each in order and
// never overlap one another, so one pass over both finds every overlap.
func (s *scanner) splitConstructs() []located {
	var out []located
	pj := 0
	for _, c := range s.t.cons {
		for pj < len(s.pieces) && s.pieces[pj].end <= c.Span.Start {
			pj++
		}
		k := pj
		for k < len(s.pieces) && s.pieces[k].start < c.Span.End {
			k++
		}
		ov := s.pieces[pj:k]
		if len(ov) == 0 || len(ov) == 1 && ov[0].start <= c.Span.Start && c.Span.End <= ov[0].end {
			continue
		}
		var text strings.Builder
		elems, last := 0, -1
		why := s.causesIn(c.Span.Start, c.Span.End)
		for _, p := range ov {
			s.visible(&text, p, max(p.start, c.Span.Start), min(p.end, c.Span.End))
			if p.elem != last {
				elems, last = elems+1, p.elem
			}
			why |= p.cause
		}
		out = append(out, located{c.Span.Start, s.finding(CodeSplitConstruct, c.Span.Line, c.Span.Col, finding.Params{
			"text":          finding.String(text.String()),
			"text_elements": finding.Int(int64(elems)),
			"cause":         finding.String(why.name()),
		})})
	}
	return out
}

// splitDelimiters reports each pair of consecutive non-empty text element
// contents of one paragraph container where the last character of the
// first and the first character of the second form a delimiter (D4 item
// 5).
func (s *scanner) splitDelimiters() []located {
	var out []located
	last := map[int]int{} // a container's id to its last piece
	pos := newCursor(s.t.src)
	for i, cur := range s.pieces {
		if cur.container < 0 {
			continue
		}
		j, ok := last[cur.container]
		last[cur.container] = i
		if !ok {
			continue
		}
		prev := s.pieces[j]
		d := string([]byte{s.t.src[prev.end-1], s.t.src[cur.start]})
		if d != "{{" && d != "{%" && d != "}}" && d != "%}" {
			continue
		}
		var text strings.Builder
		s.visible(&text, prev, prev.start, prev.end)
		s.visible(&text, cur, cur.start, cur.end)
		elems := 2
		if prev.elem == cur.elem {
			elems = 1
		}
		why := s.causesIn(prev.end, cur.start) | prev.cause | cur.cause
		line, col := pos.at(prev.end - 1)
		out = append(out, located{prev.end - 1, s.finding(CodeSplitDelimiter, line, col, finding.Params{
			"delimiter":     finding.String(d),
			"text":          finding.String(text.String()),
			"text_elements": finding.Int(int64(elems)),
			"cause":         finding.String(why.name()),
		})})
	}
	return out
}

func (s *scanner) finding(code finding.Code, line, col int, p finding.Params) finding.Finding {
	loc := finding.Location{Origin: finding.OriginPackage, File: s.file, Line: line, Column: col}
	return finding.New(code, finding.Subject{Kind: finding.SubjectTemplate, Name: s.file}, &loc, p)
}

// visible writes the visible text of src[from:to], which lies in p: a
// CDATA section's content as it is, and character data with its entity
// and character references decoded.
func (s *scanner) visible(b *strings.Builder, p piece, from, to int) {
	if p.cdata {
		b.WriteString(s.t.src[from:to])
		return
	}
	b.WriteString(decode(s.t.src[from:to]))
}

// decode replaces XML's five predefined entity references and its
// character references in s with the characters they name. A reference
// that is neither, or that names no character XML allows, is left as it
// is written.
func decode(s string) string {
	if strings.IndexByte(s, '&') < 0 {
		return s
	}
	var b strings.Builder
	for {
		i := strings.IndexByte(s, '&')
		if i < 0 {
			b.WriteString(s)
			return b.String()
		}
		b.WriteString(s[:i])
		s = s[i:]
		// A reference is at most "&#x10FFFF;" long, so the search for its
		// ";" is bounded.
		j := strings.IndexByte(s[:min(len(s), 12)], ';')
		if r, ok := reference(s[1:max(j, 1)]); j > 0 && ok {
			b.WriteString(r)
			s = s[j+1:]
			continue
		}
		b.WriteByte('&')
		s = s[1:]
	}
}

// reference returns the text the reference named name stands for.
func reference(name string) (string, bool) {
	switch name {
	case "lt":
		return "<", true
	case "gt":
		return ">", true
	case "amp":
		return "&", true
	case "quot":
		return "\"", true
	case "apos":
		return "'", true
	}
	num, ok := strings.CutPrefix(name, "#")
	if !ok || num == "" {
		return "", false
	}
	base := 10
	if h, ok := strings.CutPrefix(num, "x"); ok {
		num, base = h, 16
	}
	if num == "" || num[0] == '+' || num[0] == '-' {
		return "", false
	}
	n, err := strconv.ParseUint(num, base, 32)
	if err != nil || !xmlChar(rune(n)) {
		return "", false
	}
	return string(rune(n)), true
}

// xmlChar reports whether r is a character XML 1.0 allows.
func xmlChar(r rune) bool {
	switch {
	case r == 0x9 || r == 0xA || r == 0xD:
		return true
	case 0x20 <= r && r <= 0xD7FF, 0xE000 <= r && r <= 0xFFFD, 0x10000 <= r && r <= utf8.MaxRune:
		return true
	}
	return false
}

// renders returns the partials a construct's render tags name: a render
// tag's, or those of a liquid tag's render lines. A partial named by
// anything but a string literal names nothing the scan can follow; the
// parser refuses it.
func renders(c liquid.Construct) []string {
	switch c.Tag {
	case "render":
		if n, ok := partialName(c.Markup); ok {
			return []string{n}
		}
	case "liquid":
		var out []string
		for line := range strings.SplitSeq(c.Markup, "\n") {
			line = strings.TrimLeft(line, " \t\r\v\f")
			rest, ok := strings.CutPrefix(line, "render")
			if !ok || rest == "" || isNameByte(rest[0]) {
				continue
			}
			if n, ok := partialName(strings.TrimLeft(rest, " \t\r\v\f")); ok {
				out = append(out, n)
			}
		}
		return out
	}
	return nil
}

// partialName returns the string literal markup begins with.
func partialName(markup string) (string, bool) {
	if markup == "" || markup[0] != '\'' && markup[0] != '"' {
		return "", false
	}
	end := strings.IndexByte(markup[1:], markup[0])
	if end < 0 {
		return "", false
	}
	return markup[1 : 1+end], true
}

// cursor turns offsets, asked for in increasing order, into lines and
// columns as ADR-006 D5 item 3 counts them: a line ends at LF, CR LF or a
// lone CR, and a column counts code points from the start of its line.
// Every byte is passed over once, however many offsets are asked for.
type cursor struct {
	src            string
	off, line, col int
}

func newCursor(src string) *cursor { return &cursor{src: src, line: 1, col: 1} }

func (c *cursor) at(off int) (line, col int) {
	for ; c.off < off; c.off++ {
		switch b := c.src[c.off]; {
		case b == '\n':
			c.line, c.col = c.line+1, 1
		case b == '\r':
			if c.off+1 >= len(c.src) || c.src[c.off+1] != '\n' {
				c.line, c.col = c.line+1, 1
			}
		case b&0xC0 != 0x80:
			c.col++
		}
	}
	return c.line, c.col
}
