// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package liquid

import (
	"errors"
	"fmt"
	"strings"

	"metais.org/metais/internal/liquid/ast"
	"metais.org/metais/internal/liquid/value"
)

// The expression lexer splits markup into Shopify's strict2 tokens:
//
//	identifier  [A-Za-z_][A-Za-z0-9_-]*\??
//	string      '…' or "…", with no escapes (ADR-004 D1 item 8)
//	number      -?[0-9]+(\.[0-9]+)?
//	comparison  == != <> <= < >= >, and contains followed by whitespace
//	punctuation | . .. : , [ ] ( ) =
//
// separated by optional whitespace. Any other character is a syntax error.

type etKind uint8

const (
	etEnd etKind = iota
	etIdent
	etString
	etNumber
	etCompare
	etPipe
	etDot
	etDotDot
	etColon
	etComma
	etLBracket
	etRBracket
	etLParen
	etRParen
	etAssign
	etBad
)

// etok is one token of markup. start and end are offsets in the source.
type etok struct {
	kind       etKind
	text       string
	start, end int
}

// lexMarkup tokenizes s, which begins at offset base of the source. A
// character no token begins with ends the list with an etBad token.
func lexMarkup(s string, base int) []etok {
	var ts []etok
	i := 0
	emit := func(k etKind, n int) {
		ts = append(ts, etok{kind: k, text: s[i : i+n], start: base + i, end: base + i + n})
		i += n
	}
	for {
		for i < len(s) && isSpace(s[i]) {
			i++
		}
		if i >= len(s) {
			ts = append(ts, etok{kind: etEnd, start: base + i, end: base + i})
			return ts
		}
		c := s[i]
		switch {
		case isIdentStart(c):
			n := 1
			for i+n < len(s) && (isWordByte(s[i+n]) || s[i+n] == '-') {
				n++
			}
			if i+n < len(s) && s[i+n] == '?' {
				n++
			}
			if s[i:i+n] == "contains" && i+n < len(s) && isSpace(s[i+n]) {
				emit(etCompare, n)
			} else {
				emit(etIdent, n)
			}
		case c == '\'' || c == '"':
			j := strings.IndexByte(s[i+1:], c)
			if j < 0 {
				emit(etBad, len(s)-i)
				return append(ts, etok{kind: etEnd, start: base + len(s), end: base + len(s)})
			}
			emit(etString, j+2)
		case isDigit(c) || c == '-' && i+1 < len(s) && isDigit(s[i+1]):
			n := 1
			for i+n < len(s) && isDigit(s[i+n]) {
				n++
			}
			if i+n+1 < len(s) && s[i+n] == '.' && isDigit(s[i+n+1]) {
				n++
				for i+n < len(s) && isDigit(s[i+n]) {
					n++
				}
			}
			emit(etNumber, n)
		case strings.HasPrefix(s[i:], ".."):
			emit(etDotDot, 2)
		case c == '.':
			emit(etDot, 1)
		case c == '|':
			emit(etPipe, 1)
		case c == ':':
			emit(etColon, 1)
		case c == ',':
			emit(etComma, 1)
		case c == '[':
			emit(etLBracket, 1)
		case c == ']':
			emit(etRBracket, 1)
		case c == '(':
			emit(etLParen, 1)
		case c == ')':
			emit(etRParen, 1)
		case strings.HasPrefix(s[i:], "=="), strings.HasPrefix(s[i:], "!="), strings.HasPrefix(s[i:], "<>"),
			strings.HasPrefix(s[i:], "<="), strings.HasPrefix(s[i:], ">="):
			emit(etCompare, 2)
		case c == '<' || c == '>':
			emit(etCompare, 1)
		case c == '=':
			emit(etAssign, 1)
		default:
			emit(etBad, 1)
			return append(ts, etok{kind: etEnd, start: base + i, end: base + i})
		}
	}
}

func isIdentStart(c byte) bool { return c == '_' || 'a' <= c && c <= 'z' || 'A' <= c && c <= 'Z' }
func isDigit(c byte) bool      { return '0' <= c && c <= '9' }

// isIdentifier reports whether s is an identifier with no trailing "?":
// the names a Dialect defines, and the names a template binds.
func isIdentifier(s string) bool {
	if s == "" || !isIdentStart(s[0]) {
		return false
	}
	for i := 1; i < len(s); i++ {
		if !isWordByte(s[i]) && s[i] != '-' {
			return false
		}
	}
	return true
}

// exprParser parses one run of markup.
type exprParser struct {
	p     *parser
	toks  []etok
	i     int
	at    int // where errors are located; -1 for at the failing token
	depth int // brackets and ranges open
}

// exprParser returns a parser of markup s, which begins at offset base.
// Its errors are located at at, or, when at is -1, at the token that
// failed.
func (p *parser) exprParser(s string, base, at int) *exprParser {
	return &exprParser{p: p, toks: lexMarkup(s, base), at: at}
}

func (x *exprParser) peek() etok        { return x.toks[x.i] }
func (x *exprParser) peekAt(n int) etok { return x.toks[min(x.i+n, len(x.toks)-1)] }
func (x *exprParser) atEnd() bool       { return x.peek().kind == etEnd }

func (x *exprParser) next() etok {
	t := x.toks[x.i]
	if t.kind != etEnd {
		x.i++
	}
	return t
}

// consume takes the next token if it is of kind k.
func (x *exprParser) consume(k etKind) bool {
	if x.peek().kind == k {
		x.next()
		return true
	}
	return false
}

// peekWord returns the next token's text if it is an identifier.
func (x *exprParser) peekWord() string {
	if t := x.peek(); t.kind == etIdent {
		return t.text
	}
	return ""
}

// word returns the next token's text if it is an identifier, without
// taking it.
func (x *exprParser) word() (string, bool) {
	w := x.peekWord()
	return w, w != ""
}

// consumeWord takes the next token if it is the identifier w.
func (x *exprParser) consumeWord(w string) bool {
	if x.peekWord() == w {
		x.next()
		return true
	}
	return false
}

// errorf returns a syntax error at the next token.
func (x *exprParser) errorf(format string, args ...any) *Error {
	return x.errorAt(x.peek(), CodeSyntax, nil, fmt.Sprintf(format, args...))
}

// errorAt returns an error about t.
func (x *exprParser) errorAt(t etok, code Code, args map[string]string, detail string) *Error {
	off := t.start
	if x.at >= 0 {
		off = x.at
	}
	return x.p.errAt(off, code, args, detail)
}

// report records an error that does not stop the parse.
func (x *exprParser) report(t etok, code Code, args map[string]string, detail string) {
	x.p.errs = append(x.p.errs, x.errorAt(t, code, args, detail))
}

// end refuses anything left in the markup.
func (x *exprParser) end() *Error {
	switch t := x.peek(); t.kind {
	case etEnd:
		return nil
	case etBad:
		return x.errorf("unexpected character %q", t.text)
	default:
		return x.errorf("unexpected %q", t.text)
	}
}

// span returns the span from token a's start to token b's end.
func (x *exprParser) span(a, b etok) ast.Span { return x.p.span(a.start, b.end) }

// last is the token most recently taken.
func (x *exprParser) last() etok { return x.toks[max(x.i-1, 0)] }

// enter counts one more bracket or range open, refusing one beyond the
// nesting bound.
func (x *exprParser) enter() *Error {
	if x.depth >= x.p.lim.Nesting {
		return x.errorAt(x.peek(), CodeLimit, map[string]string{"limit": "template-nesting"},
			fmt.Sprintf("more than %d brackets and ranges nested in one expression", x.p.lim.Nesting))
	}
	x.depth++
	return nil
}

// bindingName takes the name a tag binds: an identifier, which is none of
// the literal keywords. Only a loop variable may end with "?", as in
// Shopify.
func (x *exprParser) bindingName() (string, ast.Span, *Error) { return x.name(false) }

// loopName takes a for or tablerow variable.
func (x *exprParser) loopName() (string, ast.Span, *Error) { return x.name(true) }

func (x *exprParser) name(question bool) (string, ast.Span, *Error) {
	t := x.peek()
	bare := t.text
	if question {
		bare = strings.TrimSuffix(bare, "?")
	}
	if t.kind != etIdent || !isIdentifier(bare) {
		return "", ast.Span{}, x.errorf("expected a variable name")
	}
	if _, ok := keywordLiterals[t.text]; ok {
		return "", ast.Span{}, x.errorf("%q is a literal, not a name", t.text)
	}
	x.next()
	return t.text, x.span(t, t), nil
}

// keywordLiterals are the identifiers that, standing alone, are literals.
var keywordLiterals = map[string]ast.LiteralKind{
	"nil": ast.LitNil, "null": ast.LitNil, "true": ast.LitTrue, "false": ast.LitFalse,
	"empty": ast.LitEmpty, "blank": ast.LitBlank,
}

// primary parses a literal, a path or a range.
func (x *exprParser) primary() (ast.Primary, *Error) {
	t := x.peek()
	switch t.kind {
	case etString:
		x.next()
		text := t.text[1 : len(t.text)-1]
		v, err := value.NewString(text)
		if err != nil {
			return nil, x.errorAt(t, CodeSyntax, nil, "string literal is not Unicode text")
		}
		return &ast.Literal{Span: x.span(t, t), Kind: ast.LitString, Text: text, Value: v}, nil
	case etNumber:
		x.next()
		return x.number(t), nil
	case etIdent:
		if k, ok := keywordLiterals[t.text]; ok {
			if n := x.peekAt(1).kind; n != etDot && n != etLBracket {
				x.next()
				return &ast.Literal{Span: x.span(t, t), Kind: k, Text: t.text, Value: keywordValue(k)}, nil
			}
		}
		x.next()
		return x.segments(&ast.Path{Span: x.span(t, t), Root: t.text})
	case etLBracket:
		// strict2 refuses a path that begins with an index, which would
		// look a variable up by a computed name (ADR-004 D1 items 2 and 5).
		return nil, x.errorf("a path begins with a name, not %q", t.text)
	case etLParen:
		if e := x.enter(); e != nil {
			return nil, e
		}
		x.next()
		from, e := x.primary()
		if e != nil {
			return nil, e
		}
		if !x.consume(etDotDot) {
			return nil, x.errorf("expected \"..\" in a range")
		}
		to, e := x.primary()
		if e != nil {
			return nil, e
		}
		if !x.consume(etRParen) {
			return nil, x.errorf("expected \")\" closing a range")
		}
		x.depth--
		return &ast.Range{Span: x.span(t, x.last()), From: from, To: to}, nil
	case etBad:
		return nil, x.errorf("unexpected character %q", t.text)
	case etEnd:
		return nil, x.errorf("expected a value")
	}
	return nil, x.errorf("expected a value, not %q", t.text)
}

// keywordValue is the data-model value of a keyword literal; nil for
// empty and blank, which have none.
func keywordValue(k ast.LiteralKind) value.Value {
	switch k {
	case ast.LitNil:
		return value.Null{}
	case ast.LitTrue:
		return value.Bool(true)
	case ast.LitFalse:
		return value.Bool(false)
	}
	return nil
}

// number builds a numeric literal. Its kind is its spelling's: a fraction
// makes it a float (ADR-004 D4 item 2). Its value is the literal with
// leading zeros removed, as Ruby reads one. An integer of magnitude above
// 2^53−1, or a float beyond binary64, is reported as CodeNumberRange and
// the parse continues.
func (x *exprParser) number(t etok) *ast.Literal {
	lit := &ast.Literal{Span: x.span(t, t), Kind: ast.LitInteger, Text: t.text}
	s := t.text
	sign := ""
	if s[0] == '-' {
		sign, s = "-", s[1:]
	}
	intPart, frac, isFloat := strings.Cut(s, ".")
	intPart = strings.TrimLeft(intPart, "0")
	if intPart == "" {
		intPart = "0"
	}
	norm := sign + intPart
	if isFloat {
		lit.Kind = ast.LitFloat
		norm += "." + frac
	}
	n, err := value.ParseNumber(norm)
	switch {
	case errors.Is(err, value.ErrNumberRange):
		x.report(t, CodeNumberRange, nil, "the literal is beyond binary64")
		return lit
	case err != nil:
		x.report(t, CodeSyntax, nil, err.Error())
		return lit
	}
	if !isFloat {
		if _, ok := n.SafeInt(); !ok {
			x.report(t, CodeNumberRange, nil, "an integer literal of magnitude above 2^53-1")
			return lit
		}
	}
	lit.Value = n
	return lit
}

// segments parses the ".name" and "[expr]" steps after a path's root.
func (x *exprParser) segments(path *ast.Path) (ast.Primary, *Error) {
	for {
		t := x.peek()
		switch t.kind {
		case etDot:
			x.next()
			n := x.peek()
			if n.kind != etIdent {
				return nil, x.errorf("expected a name after \".\"")
			}
			x.next()
			path.Segments = append(path.Segments, &ast.Segment{Span: x.span(t, n), Name: n.text})
		case etLBracket:
			if e := x.enter(); e != nil {
				return nil, e
			}
			x.next()
			idx, e := x.primary()
			if e != nil {
				return nil, e
			}
			if !x.consume(etRBracket) {
				return nil, x.errorf("expected \"]\"")
			}
			x.depth--
			path.Segments = append(path.Segments, &ast.Segment{Span: x.span(t, x.last()), Index: idx})
		default:
			path.Span.End = x.last().end
			return path, nil
		}
	}
}

// valueExpr parses a primary and its filters. output says whether it is
// an output expression, "{{ }}" or echo's, the only place a LastFilter may
// be applied, and there only last.
func (x *exprParser) valueExpr(output bool) (*ast.Expr, *Error) {
	first := x.peek()
	base, e := x.primary()
	if e != nil {
		return nil, e
	}
	ex := &ast.Expr{Span: x.span(first, x.last()), Base: base}
	for x.peek().kind == etPipe {
		pipe := x.next()
		nt := x.peek()
		if nt.kind != etIdent {
			return nil, x.errorf("expected a filter name after \"|\"")
		}
		x.next()
		f := &ast.Filter{Name: nt.text}
		if x.consume(etColon) {
			for {
				a, e := x.arg()
				if e != nil {
					return nil, e
				}
				f.Args = append(f.Args, a)
				if !x.consume(etComma) {
					break
				}
			}
		}
		f.Span = x.span(pipe, x.last())
		if def, ok := x.p.d.Filter(f.Name); !ok {
			x.report(nt, CodeUnknownFilter, map[string]string{"filter": f.Name},
				fmt.Sprintf("the dialect has no filter %q", f.Name))
		} else if r := def.Check(f); r != nil {
			off := nt.start
			if x.at >= 0 {
				off = x.at
			}
			x.p.errs = append(x.p.errs, x.p.locate(r, off))
		} else if l, ok := def.(LastFilter); ok && l.OnlyLast() && (!output || x.peek().kind == etPipe) {
			x.report(nt, CodeSafePosition, map[string]string{"filter": f.Name},
				fmt.Sprintf("%s may only be the last filter of an output expression", f.Name))
		}
		ex.Filters = append(ex.Filters, f)
	}
	ex.Span.End = x.last().end
	return ex, nil
}

// arg parses one argument: "name: value" or a value.
func (x *exprParser) arg() (*ast.Arg, *Error) {
	t := x.peek()
	if t.kind == etIdent && x.peekAt(1).kind == etColon {
		x.next()
		x.next()
		v, e := x.primary()
		if e != nil {
			return nil, e
		}
		return &ast.Arg{Span: x.span(t, x.last()), Name: t.text, Value: v}, nil
	}
	v, e := x.primary()
	if e != nil {
		return nil, e
	}
	return &ast.Arg{Span: x.span(t, x.last()), Value: v}, nil
}

// wholeCondition parses a tag's markup as one condition.
func (x *exprParser) wholeCondition() (*ast.Cond, *Error) {
	if x.atEnd() {
		return nil, x.errorf("expected a condition")
	}
	c, e := x.condition()
	if e == nil {
		e = x.end()
	}
	return c, e
}

// condition parses comparisons joined by and and or, grouped from the
// right as Shopify groups them (ADR-004 D2).
func (x *exprParser) condition() (*ast.Cond, *Error) {
	var conds []*ast.Cond
	var ops []ast.CondKind
	for {
		c, e := x.comparison()
		if e != nil {
			return nil, e
		}
		conds = append(conds, c)
		switch x.peekWord() {
		case "and":
			ops = append(ops, ast.CondAnd)
		case "or":
			ops = append(ops, ast.CondOr)
		default:
			c := conds[len(conds)-1]
			for i := len(ops) - 1; i >= 0; i-- {
				l := conds[i]
				c = &ast.Cond{Span: x.p.span(l.Span.Start, c.Span.End), Kind: ops[i], X: l, Y: c}
			}
			return c, nil
		}
		x.next()
	}
}

// comparison parses "a", or "a op b".
func (x *exprParser) comparison() (*ast.Cond, *Error) {
	first := x.peek()
	a, e := x.primary()
	if e != nil {
		return nil, e
	}
	t := x.peek()
	if t.kind != etCompare {
		return &ast.Cond{Span: x.span(first, x.last()), Kind: ast.CondValue, Left: a}, nil
	}
	x.next()
	b, e := x.primary()
	if e != nil {
		return nil, e
	}
	return &ast.Cond{Span: x.span(first, x.last()), Kind: ast.CondCompare, Left: a, Op: ast.CompareOp(t.text), Right: b}, nil
}

// whenValues parses a when's values, separated by "," or "or".
func (x *exprParser) whenValues() ([]ast.Primary, *Error) {
	if x.atEnd() {
		return nil, x.errorf("when needs a value")
	}
	var vs []ast.Primary
	for {
		v, e := x.primary()
		if e != nil {
			return nil, e
		}
		vs = append(vs, v)
		if !x.consume(etComma) && !x.consumeWord("or") {
			return vs, x.end()
		}
	}
}

// cycle parses "[group:] value, value, …".
func (x *exprParser) cycle(at ast.Span) (ast.Node, *Error) {
	if x.atEnd() {
		return nil, x.errorf("cycle needs a value")
	}
	n := &ast.Cycle{Span: at}
	v, e := x.primary()
	if e != nil {
		return nil, e
	}
	if x.consume(etColon) {
		n.Group = v
		if v, e = x.primary(); e != nil {
			return nil, e
		}
	}
	n.Values = append(n.Values, v)
	for x.consume(etComma) {
		v, e := x.primary()
		if e != nil {
			return nil, e
		}
		n.Values = append(n.Values, v)
	}
	return n, x.end()
}

// render parses render's markup: a partial name, then "with value" or
// "for collection", then "as alias", then name: value parameters, commas
// between them optional (ADR-004 D8 item 3).
func (x *exprParser) render(at ast.Span) (ast.Node, *Error) {
	if x.atEnd() {
		return nil, x.errorf("render needs a partial name")
	}
	first := x.peek()
	name, e := x.primary()
	if e != nil {
		return nil, e
	}
	n := &ast.Render{Span: at, NameSpan: name.Pos(), Mode: ast.RenderPlain}
	if lit, ok := name.(*ast.Literal); ok && lit.Kind == ast.LitString {
		n.Name = lit.Text
	} else {
		x.report(first, CodePartialName, nil, "render's partial name is a string literal")
	}
	switch {
	case x.consumeWord("with"):
		n.Mode = ast.RenderWith
	case x.consumeWord("for"):
		n.Mode = ast.RenderFor
	}
	if n.Mode != ast.RenderPlain {
		if n.Value, e = x.primary(); e != nil {
			return nil, e
		}
		n.Alias = n.Name[strings.LastIndexByte(n.Name, '/')+1:]
		n.AliasSpan = n.NameSpan
	}
	if x.peekWord() == "as" && x.peekAt(1).kind == etIdent {
		x.next()
		alias, sp, e := x.bindingName()
		if e != nil {
			return nil, e
		}
		if n.Mode == ast.RenderPlain {
			return nil, x.errorAt(x.last(), CodeSyntax, nil, "render's as needs with or for")
		}
		n.Alias, n.AliasSpan = alias, sp
	}
	x.consume(etComma)
	params, e := x.argList(true)
	if e != nil {
		return nil, e
	}
	n.Params = params
	return n, nil
}

// argList parses arguments to the end of the markup, separated by
// optional commas. Positional arguments come before keyword arguments,
// and namedOnly refuses any positional one. A keyword is given once.
func (x *exprParser) argList(namedOnly bool) ([]*ast.Arg, *Error) {
	var args []*ast.Arg
	named := false
	seen := map[string]bool{}
	for !x.atEnd() {
		t := x.peek()
		a, e := x.arg()
		if e != nil {
			return nil, e
		}
		switch {
		case a.Name == "" && (named || namedOnly):
			return nil, x.errorAt(t, CodeSyntax, nil, "expected name: value")
		case a.Name != "" && seen[a.Name]:
			return nil, x.errorAt(t, CodeSyntax, nil, fmt.Sprintf("%s: given twice", a.Name))
		case a.Name != "":
			named, seen[a.Name] = true, true
		}
		args = append(args, a)
		if x.consume(etComma) && x.atEnd() {
			return nil, x.errorf("expected an argument after \",\"")
		}
	}
	return args, nil
}
