// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package liquid

import (
	"fmt"
	"runtime/debug"
	"strings"
	"unicode/utf8"

	"github.com/kottera/metais/internal/liquid/ast"
)

// Source is a template's text and the name errors give it: for a package's
// template, its entry path.
type Source struct {
	Name string
	Text []byte
}

// Template is a parsed template. It is immutable, and safe to render any
// number of times.
type Template struct {
	Name string
	Root *ast.Block
}

// DefaultNesting is the nesting bound when ParseLimits names none:
// Shopify's Block::MAX_DEPTH, and ADR-009's default for template-nesting.
const DefaultNesting = 100

// ParseLimits bounds what a template's own text may make the parser do.
// The values are the caller's — internal/template passes ADR-009's
// template-nesting and template-size — and this package knows no other.
type ParseLimits struct {
	// Nesting bounds the block tags open at once (ADR-009 template-nesting).
	// A template opening one more is refused with CodeLimit. The tags that
	// count are those with a body: if, unless, case, for, tablerow,
	// capture, ifchanged and comment, and the liquid tag, whose lines nest
	// inside it. The same bound applies, separately, to the brackets and
	// ranges nested in one expression, so that no text can make the parser
	// recurse without bound. Zero means DefaultNesting.
	Nesting int
	// Size bounds the template's length in bytes (ADR-009 template-size).
	// Zero means no bound.
	Size int
}

// Parse parses s by the dialect's grammar: Shopify's strict2 (ADR-004 D1
// to D3) with d's filters, tags and reserved names.
//
// A syntax error, an unknown tag, U+0000, text that is not UTF-8 and a
// breached limit each stop the parse; it reports that error and nothing
// after it, so a template has at most one of them (ADR-004 D16 item 4).
// Every other error — include, an unknown filter, a reserved name, a
// non-literal partial name, a refusal by a TagDef's or FilterDef's Check
// — is reported and the parse continues, so every one before the first
// stopping error is reported. Errors are in source order.
//
// The template is nil exactly when an error is reported. A panic inside
// the parser is recovered as one CodeInternal error.
func Parse(d *Dialect, s Source, lim ParseLimits) (t *Template, errs []*Error) {
	p := newParser(d, s, lim)
	defer p.recover(func(e *Error) { t, errs = nil, append(p.errs, e) })
	root, fatal := p.parseTemplate()
	if fatal != nil {
		p.errs = append(p.errs, fatal)
	}
	if len(p.errs) > 0 {
		return nil, p.errs
	}
	return &Template{Name: s.Name, Root: root}, nil
}

// ParseValue parses s as a value expression: exactly what may appear
// inside "{{ … }}", a lookup or literal followed by any filters of d
// (ADR-005 D3 item 2). It is what a for_each and a path's expressions
// are. Errors are located within s. An empty expression is a syntax
// error, and so is anything after the expression.
func ParseValue(d *Dialect, s Source, lim ParseLimits) (e *ast.Expr, err *Error) {
	p := newParser(d, s, lim)
	defer p.recover(func(x *Error) { e, err = nil, x })
	if fatal := p.checkText(); fatal != nil {
		return nil, fatal
	}
	src := p.src
	x := p.exprParser(src, 0, -1)
	if x.atEnd() {
		return nil, x.errorf("expected a value")
	}
	e, fatal := x.valueExpr()
	if fatal == nil {
		fatal = x.end()
	}
	if fatal != nil {
		return nil, fatal
	}
	if len(p.errs) > 0 {
		return nil, p.errs[0]
	}
	return e, nil
}

// ParseCondition parses s as a condition: exactly what may follow if in
// "{% if … %}" (ADR-005 D3 item 1). It is what a when is. Errors are
// located within s; an empty condition is a syntax error, and so is
// anything after it.
func ParseCondition(d *Dialect, s Source, lim ParseLimits) (c *ast.Cond, err *Error) {
	p := newParser(d, s, lim)
	defer p.recover(func(x *Error) { c, err = nil, x })
	if fatal := p.checkText(); fatal != nil {
		return nil, fatal
	}
	x := p.exprParser(p.src, 0, -1)
	if x.atEnd() {
		return nil, x.errorf("expected a condition")
	}
	c, fatal := x.condition()
	if fatal == nil {
		fatal = x.end()
	}
	if fatal != nil {
		return nil, fatal
	}
	if len(p.errs) > 0 {
		return nil, p.errs[0]
	}
	return c, nil
}

// parser is one parse of one source.
type parser struct {
	d     *Dialect
	name  string
	src   string
	lines *lineIndex
	lim   ParseLimits
	errs  []*Error
	depth int // block tags open, and liquid tags
	// lastText is the text node the previous token made, which a "{{-" or
	// "{%-" trims; nil when the previous token was not text.
	lastText *ast.Text
	// trimNext is set when the previous token ended with "-}}" or "-%}".
	trimNext bool
}

func newParser(d *Dialect, s Source, lim ParseLimits) *parser {
	if lim.Nesting <= 0 {
		lim.Nesting = DefaultNesting
	}
	if d == nil {
		d = &Dialect{}
	}
	src := string(s.Text)
	return &parser{d: d, name: s.Name, src: src, lines: newLineIndex(src), lim: lim}
}

// recover turns a panic of the parse into a CodeInternal error, handed to
// set. It is deferred by each entry point.
func (p *parser) recover(set func(*Error)) {
	if r := recover(); r != nil {
		e := p.errAt(0, CodeInternal, nil, fmt.Sprintf("panic: %v\n%s", r, debug.Stack()))
		set(e)
	}
}

// errAt returns an error located at off.
func (p *parser) errAt(off int, code Code, args map[string]string, detail string) *Error {
	line, col := p.lines.at(off)
	return &Error{Source: p.name, Line: line, Col: col, Offset: off, Code: code, Args: args, Detail: detail}
}

// syntaxAt returns a syntax error located at off.
func (p *parser) syntaxAt(off int, format string, args ...any) *Error {
	return p.errAt(off, CodeSyntax, nil, fmt.Sprintf(format, args...))
}

// span returns the span of src[start:end].
func (p *parser) span(start, end int) ast.Span {
	line, col := p.lines.at(start)
	return ast.Span{Start: start, End: end, Line: line, Col: col}
}

// checkText applies the checks every source passes before it is parsed:
// its size, UTF-8 and U+0000 (ADR-004 D1 item 7).
func (p *parser) checkText() *Error {
	if p.lim.Size > 0 && len(p.src) > p.lim.Size {
		return p.errAt(0, CodeLimit, map[string]string{"limit": "template-size"},
			fmt.Sprintf("%d bytes, above the bound of %d", len(p.src), p.lim.Size))
	}
	if !utf8.ValidString(p.src) {
		off := 0
		for off < len(p.src) {
			r, n := utf8.DecodeRuneInString(p.src[off:])
			if r == utf8.RuneError && n <= 1 {
				break
			}
			off += n
		}
		return p.errAt(off, CodeNotUTF8, nil, "the text is not valid UTF-8")
	}
	if i := strings.IndexByte(p.src, 0); i >= 0 {
		return p.syntaxAt(i, "U+0000 in a template")
	}
	return nil
}

// frame is a block tag being parsed: its node, the body nodes are being
// added to, and what may come next.
type frame struct {
	name  string // the opening tag's name
	start int    // the opening token's offset
	node  ast.Node
	body  *ast.Block
	// sawElse is set once an else has been read; no branch may follow it
	// in if, unless or for.
	sawElse bool
}

// source is where a run of tokens comes from: a template's scanner, or
// the lines of a liquid tag.
type source interface {
	next() tok
}

// parseTemplate parses the whole template into its root block.
func (p *parser) parseTemplate() (*ast.Block, *Error) {
	if e := p.checkText(); e != nil {
		return nil, e
	}
	root := &ast.Block{Span: p.span(0, len(p.src))}
	if e := p.parseTokens(&scanner{src: p.src}, root, false); e != nil {
		return nil, e
	}
	return root, nil
}

// parseTokens parses every token of ts into root, until ts ends. Every
// block opened must be closed before the end: a liquid tag's blocks close
// inside it. inLiquid is set for a liquid tag's lines.
func (p *parser) parseTokens(ts source, root *ast.Block, inLiquid bool) *Error {
	var stack []*frame
	top := func() *frame {
		if len(stack) == 0 {
			return nil
		}
		return stack[len(stack)-1]
	}
	add := func(n ast.Node) {
		b := root
		if f := top(); f != nil {
			b = f.body
		}
		b.Nodes = append(b.Nodes, n)
	}
	for {
		t := ts.next()
		switch t.kind {
		case tokEOF:
			if f := top(); f != nil {
				return p.syntaxAt(f.start, "%q was never closed", f.name)
			}
			return nil
		case tokText:
			txt := &ast.Text{Span: p.span(t.start, t.end), Text: p.src[t.start:t.end], TrimLeft: p.trimNext}
			add(txt)
			p.lastText, p.trimNext = txt, false
			continue
		}
		if t.bad != "" {
			return p.syntaxAt(t.start, "%s", t.bad)
		}
		if t.trimL && p.lastText != nil {
			p.lastText.TrimRight = true
		}
		p.lastText, p.trimNext = nil, t.trimR
		if t.kind == tokOutput {
			o, e := p.output(t)
			if e != nil {
				return e
			}
			add(o)
			continue
		}
		f := top()
		switch name := t.name; {
		case f != nil && name == "end"+f.name:
			if e := p.noMarkup(t); e != nil {
				return e
			}
			p.closeFrame(f, t)
			stack = stack[:len(stack)-1]
			p.depth--
		case f != nil && isBranchTag(f.name, name):
			if e := p.branch(f, t); e != nil {
				return e
			}
		case isBlockTag(name):
			if p.depth >= p.lim.Nesting {
				return p.errAt(t.start, CodeLimit, map[string]string{"limit": "template-nesting"},
					fmt.Sprintf("more than %d block tags open at once", p.lim.Nesting))
			}
			if name == "comment" {
				c, e := p.comment(ts, t, inLiquid)
				if e != nil {
					return e
				}
				add(c)
				continue
			}
			nf, e := p.open(t)
			if e != nil {
				return e
			}
			add(nf.node)
			stack = append(stack, nf)
			p.depth++
		default:
			n, e := p.inlineTag(ts, t, inLiquid)
			if e != nil {
				return e
			}
			if n != nil {
				add(n)
			}
		}
	}
}

// isBlockTag reports whether name opens a block with a body the parser
// parses: one of the tags that count towards nesting.
func isBlockTag(name string) bool {
	switch name {
	case "if", "unless", "case", "for", "tablerow", "capture", "ifchanged", "comment":
		return true
	}
	return false
}

// isBranchTag reports whether name starts a new branch of the block block.
func isBranchTag(block, name string) bool {
	switch block {
	case "if", "unless":
		return name == "elsif" || name == "else"
	case "case":
		return name == "when" || name == "else"
	case "for":
		return name == "else"
	}
	return false
}

// noMarkup refuses markup on a tag that takes none. Shopify ignores it;
// METAIS refuses what Shopify ignores (ADR-004 D1 item 2).
func (p *parser) noMarkup(t tok) *Error {
	if strings.TrimLeft(t.markup, " \t\n\v\f\r") != "" {
		return p.syntaxAt(t.start, "%q takes no markup", t.name)
	}
	return nil
}

// tokSpan is the span of a whole token.
func (p *parser) tokSpan(t tok) ast.Span { return p.span(t.start, t.end) }

// markupParser returns an expression parser over t's markup, its errors
// located at the token (ADR-004 D16 item 2).
func (p *parser) markupParser(t tok) *exprParser {
	return p.exprParser(t.markup, t.markupAt, t.start)
}

// open parses the opening tag of a block and returns its frame.
func (p *parser) open(t tok) (*frame, *Error) {
	f := &frame{name: t.name, start: t.start}
	body := func() *ast.Block {
		return &ast.Block{Span: p.span(t.end, t.end)}
	}
	x := p.markupParser(t)
	switch t.name {
	case "if", "unless":
		c, e := x.wholeCondition()
		if e != nil {
			return nil, e
		}
		b := &ast.Branch{Span: p.tokSpan(t), Cond: c, Body: body()}
		if t.name == "if" {
			f.node = &ast.If{Span: p.tokSpan(t), Branches: []*ast.Branch{b}}
		} else {
			f.node = &ast.Unless{Span: p.tokSpan(t), Branches: []*ast.Branch{b}}
		}
		f.body = b.Body
	case "case":
		if x.atEnd() {
			return nil, x.errorf("case needs a value")
		}
		v, e := x.primary()
		if e == nil {
			e = x.end()
		}
		if e != nil {
			return nil, e
		}
		n := &ast.Case{Span: p.tokSpan(t), Subject: v, Before: body()}
		f.node, f.body = n, n.Before
	case "for", "tablerow":
		n, e := p.loop(t, x)
		if e != nil {
			return nil, e
		}
		f.node = n
		switch n := n.(type) {
		case *ast.For:
			n.Body = body()
			f.body = n.Body
		case *ast.Tablerow:
			n.Body = body()
			f.body = n.Body
		}
	case "capture":
		name, at, e := x.bindingName()
		if e == nil {
			e = x.end()
		}
		if e != nil {
			return nil, e
		}
		n := &ast.Capture{Span: p.tokSpan(t), Name: name, NameSpan: at, Body: body()}
		p.checkReserved(t, name)
		f.node, f.body = n, n.Body
	case "ifchanged":
		if e := p.noMarkup(t); e != nil {
			return nil, e
		}
		n := &ast.Ifchanged{Span: p.tokSpan(t), Body: body()}
		f.node, f.body = n, n.Body
	}
	return f, nil
}

// branch parses an elsif, else or when of the block f.
func (p *parser) branch(f *frame, t tok) *Error {
	if f.sawElse && f.name != "case" {
		return p.syntaxAt(t.start, "%q after else in %q", t.name, f.name)
	}
	f.body.Span.End = t.start
	body := &ast.Block{Span: p.span(t.end, t.end)}
	if t.name == "else" {
		if e := p.noMarkup(t); e != nil {
			return e
		}
		f.sawElse = true
	}
	switch n := f.node.(type) {
	case *ast.If:
		if t.name == "else" {
			n.Else = body
		} else {
			c, e := p.markupParser(t).wholeCondition()
			if e != nil {
				return e
			}
			n.Branches = append(n.Branches, &ast.Branch{Span: p.tokSpan(t), Cond: c, Body: body})
		}
	case *ast.Unless:
		if t.name == "else" {
			n.Else = body
		} else {
			c, e := p.markupParser(t).wholeCondition()
			if e != nil {
				return e
			}
			n.Branches = append(n.Branches, &ast.Branch{Span: p.tokSpan(t), Cond: c, Body: body})
		}
	case *ast.For:
		n.Else = body
	case *ast.Case:
		w := &ast.When{Span: p.tokSpan(t), Body: body, Else: t.name == "else"}
		if t.name == "when" {
			vs, e := p.markupParser(t).whenValues()
			if e != nil {
				return e
			}
			w.Values = vs
		}
		n.Clauses = append(n.Clauses, w)
	}
	f.body = body
	return nil
}

// closeFrame records where f's node and its last body end.
func (p *parser) closeFrame(f *frame, t tok) {
	f.body.Span.End = t.start
	switch n := f.node.(type) {
	case *ast.If:
		n.Span.End = t.end
	case *ast.Unless:
		n.Span.End = t.end
	case *ast.Case:
		n.Span.End = t.end
	case *ast.For:
		n.Span.End = t.end
	case *ast.Tablerow:
		n.Span.End = t.end
	case *ast.Capture:
		n.Span.End = t.end
	case *ast.Ifchanged:
		n.Span.End = t.end
	}
}

// checkReserved reports a template-scoped binding of a reserved name
// (ADR-004 D7 item 8); the parse continues.
func (p *parser) checkReserved(t tok, name string) {
	if p.d.Reserved(name) {
		p.errs = append(p.errs, p.errAt(t.start, CodeReservedName, map[string]string{"name": name},
			fmt.Sprintf("%q may not be bound by %s", name, t.name)))
	}
}

// output parses "{{ … }}".
func (p *parser) output(t tok) (*ast.Output, *Error) {
	o := &ast.Output{Span: p.tokSpan(t)}
	x := p.markupParser(t)
	if x.atEnd() {
		return o, nil
	}
	e, err := x.valueExpr()
	if err == nil {
		err = x.end()
	}
	if err != nil {
		return nil, err
	}
	o.Value = e
	return o, nil
}

// inlineTag parses a tag that is not a block, or reports a tag that may
// not appear here. It returns a nil node for a tag that is refused but
// does not stop the parse.
func (p *parser) inlineTag(ts source, t tok, inLiquid bool) (ast.Node, *Error) {
	x := p.markupParser(t)
	switch t.name {
	case "assign":
		return p.assign(t, x)
	case "echo":
		n := &ast.Echo{Span: p.tokSpan(t)}
		if x.atEnd() {
			return n, nil
		}
		v, e := x.valueExpr()
		if e == nil {
			e = x.end()
		}
		if e != nil {
			return nil, e
		}
		n.Value = v
		return n, nil
	case "increment", "decrement":
		name, at, e := x.bindingName()
		if e == nil {
			e = x.end()
		}
		if e != nil {
			return nil, e
		}
		p.checkReserved(t, name)
		return &ast.Counter{Span: p.tokSpan(t), Name: name, NameSpan: at, Decrement: t.name == "decrement"}, nil
	case "cycle":
		return x.cycle(p.tokSpan(t))
	case "break", "continue":
		if e := p.noMarkup(t); e != nil {
			return nil, e
		}
		if t.name == "break" {
			return &ast.Break{Span: p.tokSpan(t)}, nil
		}
		return &ast.Continue{Span: p.tokSpan(t)}, nil
	case "#":
		return p.inlineComment(t)
	case "raw":
		if inLiquid {
			return nil, p.syntaxAt(t.start, "raw cannot appear in a liquid tag")
		}
		return p.raw(ts, t)
	case "doc":
		if inLiquid {
			return nil, p.syntaxAt(t.start, "doc cannot appear in a liquid tag")
		}
		return p.doc(ts, t)
	case "liquid":
		return p.liquid(t)
	case "render":
		return x.render(p.tokSpan(t))
	case "include":
		p.errs = append(p.errs, p.errAt(t.start, CodeInclude, map[string]string{"use": "render"},
			"include is not part of the dialect; use render"))
		return nil, nil
	}
	if def, ok := p.d.Tag(t.name); ok {
		return p.addedTag(t, x, def)
	}
	switch t.name {
	case "elsif", "else", "when", "endif", "endunless", "endcase", "endfor", "endtablerow",
		"endcapture", "endifchanged", "endcomment", "endraw", "enddoc":
		return nil, p.syntaxAt(t.start, "%q is not inside a block it belongs to", t.name)
	}
	return nil, p.syntaxAt(t.start, "unknown tag %q", t.name)
}

// assign parses "{% assign name = value %}".
func (p *parser) assign(t tok, x *exprParser) (ast.Node, *Error) {
	name, at, e := x.bindingName()
	if e != nil {
		return nil, e
	}
	if !x.consume(etAssign) {
		return nil, x.errorf("assign needs \"=\" after its name")
	}
	if x.atEnd() {
		return nil, x.errorf("assign needs a value")
	}
	v, e := x.valueExpr()
	if e == nil {
		e = x.end()
	}
	if e != nil {
		return nil, e
	}
	p.checkReserved(t, name)
	return &ast.Assign{Span: p.tokSpan(t), Name: name, NameSpan: at, Value: v}, nil
}

// loop parses the markup of for and tablerow: "var in collection" and its
// modifiers.
func (p *parser) loop(t tok, x *exprParser) (ast.Node, *Error) {
	name, at, e := x.loopName()
	if e != nil {
		return nil, e
	}
	if !x.consumeWord("in") {
		return nil, x.errorf("%s needs \"in\" after its variable", t.name)
	}
	if x.atEnd() {
		return nil, x.errorf("%s needs a collection", t.name)
	}
	coll, e := x.primary()
	if e != nil {
		return nil, e
	}
	loopName := name + "-" + p.exprString(coll)
	if t.name == "for" {
		n := &ast.For{Span: p.tokSpan(t), Var: name, VarSpan: at, Collection: coll, Name: loopName}
		n.Reversed = x.consumeWord("reversed")
		seen := map[string]bool{}
		for !x.atEnd() {
			if x.consume(etComma) && x.atEnd() {
				break // Shopify accepts a trailing comma here
			}
			attr, ok := x.word()
			if !ok || (attr != "limit" && attr != "offset") {
				return nil, x.errorf("for takes limit: and offset:, and reversed only before them")
			}
			if seen[attr] {
				return nil, x.errorf("%s: given twice", attr)
			}
			seen[attr] = true
			x.next()
			if !x.consume(etColon) {
				return nil, x.errorf("%s needs \":\"", attr)
			}
			if attr == "offset" && x.peekWord() == "continue" && x.peekAt(1).kind != etDot && x.peekAt(1).kind != etLBracket {
				x.next()
				n.OffsetContinue = true
				continue
			}
			v, e := x.primary()
			if e != nil {
				return nil, e
			}
			if attr == "limit" {
				n.Limit = v
			} else {
				n.Offset = v
			}
		}
		return n, nil
	}
	n := &ast.Tablerow{Span: p.tokSpan(t), Var: name, VarSpan: at, Collection: coll, Name: loopName}
	seen := map[string]bool{}
	for !x.atEnd() {
		x.consume(etComma)
		attr, ok := x.word()
		if !ok || (attr != "cols" && attr != "limit" && attr != "offset") {
			return nil, x.errorf("tablerow takes cols:, limit: and offset:")
		}
		if seen[attr] {
			return nil, x.errorf("%s: given twice", attr)
		}
		seen[attr] = true
		x.next()
		if !x.consume(etColon) {
			return nil, x.errorf("%s needs \":\"", attr)
		}
		v, e := x.primary()
		if e != nil {
			return nil, e
		}
		switch attr {
		case "cols":
			n.Cols = v
		case "limit":
			n.Limit = v
		default:
			n.Offset = v
		}
	}
	return n, nil
}

// inlineComment parses "{% # … %}": every line of it must begin with "#"
// (ADR-004 D2), so that a later dialect can let a comment end at its line.
func (p *parser) inlineComment(t tok) (ast.Node, *Error) {
	lines := strings.Split(t.markup, "\n")
	for _, l := range lines[1:] {
		l = strings.TrimLeft(l, " \t\v\f\r")
		if l != "" && l[0] != '#' {
			return nil, p.syntaxAt(t.start, "every line of a # comment begins with #")
		}
	}
	return &ast.Comment{Span: p.tokSpan(t), Kind: ast.CommentInline, Text: t.markup}, nil
}

// raw parses "{% raw %}…{% endraw %}". Its body is every byte up to the
// endraw tag, found as Shopify finds it: the first token that ends with a
// tag named endraw, wherever in the token that tag begins.
func (p *parser) raw(ts source, t tok) (ast.Node, *Error) {
	if e := p.noMarkup(t); e != nil {
		return nil, e
	}
	start := t.end
	for {
		u := ts.next()
		if u.kind == tokEOF {
			return nil, p.syntaxAt(t.start, "\"raw\" was never closed")
		}
		if at, trimR, ok := endsWithTag(p.src[u.start:u.end], "endraw"); ok {
			bodyEnd := u.start + at
			n := &ast.Raw{Span: p.span(t.start, u.end), Text: p.src[start:bodyEnd], TextSpan: p.span(start, bodyEnd)}
			p.lastText, p.trimNext = nil, trimR
			return n, nil
		}
	}
}

// doc parses "{% doc %}…{% enddoc %}", which takes no markup and does not
// nest.
func (p *parser) doc(ts source, t tok) (ast.Node, *Error) {
	if e := p.noMarkup(t); e != nil {
		return nil, e
	}
	start := t.end
	for {
		u := ts.next()
		if u.kind == tokEOF {
			return nil, p.syntaxAt(t.start, "\"doc\" was never closed")
		}
		s := p.src[u.start:u.end]
		if _, _, ok := endsWithTag(s, "doc"); ok {
			return nil, p.syntaxAt(u.start, "doc does not nest")
		}
		if at, trimR, ok := endsWithTag(s, "enddoc"); ok {
			bodyEnd := u.start + at
			p.lastText, p.trimNext = nil, trimR
			return &ast.Comment{Span: p.span(t.start, u.end), Kind: ast.CommentDoc, Text: p.src[start:bodyEnd]}, nil
		}
	}
}

// endsWithTag reports whether the token text s ends with a tag named name
// — "{%", an optional "-", whitespace, the name, and anything up to "%}"
// — and where in s that tag begins, and whether it ends with "-%}". It is
// how Shopify finds the end of a raw or doc body: the last "{%" of the
// token is the tag's start.
func endsWithTag(s, name string) (at int, trimR, ok bool) {
	if !strings.HasSuffix(s, "%}") {
		return 0, false, false
	}
	at = strings.LastIndex(s[:len(s)-2], "{%")
	if at < 0 {
		return 0, false, false
	}
	t := tagToken(s, at, len(s))
	if t.bad != "" || t.name != name {
		return 0, false, false
	}
	return at, t.trimR, true
}

// comment parses "{% comment %}…{% endcomment %}". Its body is skipped
// token by token: a comment inside it nests, a raw inside it is skipped to
// its endraw, and nothing else in it is parsed (ADR-004 D2). In a liquid
// tag, its body is the lines up to the matching endcomment. The comment
// tag's own markup is commentary too, and is skipped with it.
func (p *parser) comment(ts source, t tok, inLiquid bool) (ast.Node, *Error) {
	depth := 1
	start := t.end
	for {
		u := ts.next()
		if u.kind == tokEOF {
			return nil, p.syntaxAt(t.start, "\"comment\" was never closed")
		}
		if u.kind != tokTag || (u.bad != "" && !inLiquid) {
			continue
		}
		switch u.name {
		case "raw":
			if inLiquid {
				continue
			}
			if e := p.skipRaw(ts, u); e != nil {
				return nil, e
			}
		case "comment":
			depth++
		case "endcomment":
			depth--
			if depth == 0 {
				p.lastText, p.trimNext = nil, u.trimR
				return &ast.Comment{Span: p.span(t.start, u.end), Kind: ast.CommentBlock, Text: p.src[start:u.start]}, nil
			}
		}
	}
}

// skipRaw skips a raw's body inside a comment.
func (p *parser) skipRaw(ts source, t tok) *Error {
	for {
		u := ts.next()
		if u.kind == tokEOF {
			return p.syntaxAt(t.start, "\"raw\" was never closed")
		}
		if _, _, ok := endsWithTag(p.src[u.start:u.end], "endraw"); ok {
			return nil
		}
	}
}

// liquid parses "{% liquid … %}": its markup is a tag per line, parsed as
// a block whose blocks close inside it.
func (p *parser) liquid(t tok) (ast.Node, *Error) {
	if p.depth >= p.lim.Nesting {
		return nil, p.errAt(t.start, CodeLimit, map[string]string{"limit": "template-nesting"},
			fmt.Sprintf("more than %d block tags open at once", p.lim.Nesting))
	}
	p.depth++
	defer func() { p.depth-- }()
	body := &ast.Block{Span: p.span(t.markupAt, t.markupAt+len(t.markup))}
	p.lastText, p.trimNext = nil, false
	if e := p.parseTokens(&lineSource{text: t.markup, at: t.markupAt}, body, true); e != nil {
		return nil, e
	}
	p.lastText, p.trimNext = nil, t.trimR
	return &ast.Liquid{Span: p.tokSpan(t), Body: body}, nil
}

// lineSource yields a liquid tag's lines as tags. A line holding only
// whitespace is skipped. Each line's token starts at its first non-blank
// character, which is where its errors are located (ADR-004 D16 item 2).
// Lines are split at line feeds only; a carriage return is whitespace
// within its line.
type lineSource struct {
	text string
	at   int // text's offset in the source
	off  int
}

func (l *lineSource) next() tok {
	for l.off <= len(l.text) {
		end := strings.IndexByte(l.text[l.off:], '\n')
		if end < 0 {
			end = len(l.text)
		} else {
			end += l.off
		}
		line, lineAt := l.text[l.off:end], l.at+l.off
		l.off = end + 1
		i := 0
		for i < len(line) && isSpace(line[i]) {
			i++
		}
		if i == len(line) {
			continue
		}
		t := tok{kind: tokTag, start: lineAt + i, end: lineAt + len(line)}
		n := i
		if line[n] == '#' {
			n++
		} else {
			for n < len(line) && isWordByte(line[n]) {
				n++
			}
		}
		if n == i {
			t.bad = "a line of a liquid tag begins with a tag name"
			return t
		}
		t.name = line[i:n]
		if t.name != "#" {
			for n < len(line) && isSpace(line[n]) {
				n++
			}
		}
		t.markup, t.markupAt = line[n:], lineAt+n
		return t
	}
	return tok{kind: tokEOF, start: l.at + len(l.text), end: l.at + len(l.text)}
}

// addedTag parses a tag of the dialect's: its markup as an argument list,
// then its definition's static check.
func (p *parser) addedTag(t tok, x *exprParser, def TagDef) (ast.Node, *Error) {
	args, e := x.argList(false)
	if e != nil {
		return nil, e
	}
	n := &ast.Tag{Span: p.tokSpan(t), Name: t.name, Args: args}
	if r := def.Check(n); r != nil {
		p.errs = append(p.errs, p.locate(r, t.start))
	}
	return n, nil
}

// locate fills in a definition's refusal with the source and the position
// at off.
func (p *parser) locate(r *Error, off int) *Error {
	e := *r
	line, col := p.lines.at(off)
	e.Source, e.Line, e.Col, e.Offset = p.name, line, col, off
	return &e
}

// exprString writes a primary as Shopify's parser rebuilds an
// expression's text, which is what forloop.name joins to the variable
// name: each literal as written, and the rest rebuilt without whitespace.
func (p *parser) exprString(v ast.Primary) string {
	switch v := v.(type) {
	case *ast.Literal:
		return p.src[v.Span.Start:v.Span.End]
	case *ast.Range:
		return "(" + p.exprString(v.From) + ".." + p.exprString(v.To) + ")"
	case *ast.Path:
		var b strings.Builder
		b.WriteString(v.Root)
		for _, s := range v.Segments {
			if s.Index == nil {
				b.WriteString("." + s.Name)
			} else {
				b.WriteString("[" + p.exprString(s.Index) + "]")
			}
		}
		return b.String()
	}
	return ""
}
