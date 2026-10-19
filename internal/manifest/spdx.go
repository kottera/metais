// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package manifest

import "strings"

// spdxToken is one token of a license expression: an operator, a
// parenthesis, or a word, with the 1-based character position it begins
// at.
type spdxToken struct {
	text string
	pos  int
}

// spdxError is why a license is not an SPDX license expression, and the
// 1-based character position where that shows.
type spdxError struct {
	reason string
	pos    int
}

// checkLicense judges text by SPDX 2.3.1 Annex D's grammar, as ADR-002 D4
// item 11 narrows it: any idstring stands where a license or exception
// identifier does, the operators AND, OR and WITH are upper case, and the
// DocumentRef- form is refused, since it names a document outside the
// package. WITH binds tighter than AND, and AND than OR. Tokens are
// separated by spaces, which are otherwise ignored.
//
// Identifiers are not checked against the SPDX License List: the list
// grows, and checking it would make validity depend on the build.
func checkLicense(text string) *spdxError {
	toks, err := spdxTokens(text)
	if err != nil {
		return err
	}
	p := spdxParser{toks: toks, end: len([]rune(text)) + 1}
	if err := p.expression(); err != nil {
		return err
	}
	if p.i < len(p.toks) {
		t := p.toks[p.i]
		if t.text == ")" {
			return &spdxError{reasonParentheses, t.pos}
		}
		return &spdxError{reasonSyntax, t.pos}
	}
	return nil
}

// spdxTokens splits text into tokens, judging each word on its own.
func spdxTokens(text string) ([]spdxToken, *spdxError) {
	var toks []spdxToken
	var word []rune
	start := 0
	flush := func() *spdxError {
		if len(word) == 0 {
			return nil
		}
		w := string(word)
		word = word[:0]
		if err := spdxWord(w, start); err != nil {
			return err
		}
		toks = append(toks, spdxToken{w, start})
		return nil
	}
	pos := 0
	for _, r := range text {
		pos++
		switch r {
		case ' ', '(', ')':
			if err := flush(); err != nil {
				return nil, err
			}
			if r != ' ' {
				toks = append(toks, spdxToken{string(r), pos})
			}
		default:
			if len(word) == 0 {
				start = pos
			}
			word = append(word, r)
		}
	}
	if err := flush(); err != nil {
		return nil, err
	}
	return toks, nil
}

// spdxWord judges one word beginning at pos: an operator, or a license
// identifier with an optional + suffix, or a LicenseRef-.
func spdxWord(w string, pos int) *spdxError {
	switch w {
	case "AND", "OR", "WITH":
		return nil
	}
	switch strings.ToUpper(w) {
	case "AND", "OR", "WITH":
		return &spdxError{reasonOperatorCase, pos}
	}
	if strings.HasPrefix(w, "DocumentRef-") {
		return &spdxError{reasonDocumentRef, pos}
	}
	id, plus := strings.CutSuffix(w, "+")
	if plus && strings.HasPrefix(id, "LicenseRef-") {
		return &spdxError{reasonLicenseRefAdd, pos}
	}
	if !idstring(id) {
		return &spdxError{reasonSyntax, pos}
	}
	return nil
}

// idstring reports whether s is Annex D's idstring: one or more ASCII
// letters, digits, hyphens and full stops.
func idstring(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '-' || c == '.') {
			return false
		}
	}
	return true
}

// spdxParser parses the tokens by recursive descent. Each level is entered
// once per parenthesis, so its depth is bounded by the expression's 512
// characters.
type spdxParser struct {
	toks []spdxToken
	i    int
	end  int // the position just past the text, where a missing token is
}

func (p *spdxParser) peek() (spdxToken, bool) {
	if p.i < len(p.toks) {
		return p.toks[p.i], true
	}
	return spdxToken{pos: p.end}, false
}

// expression is and-expression *("OR" and-expression).
func (p *spdxParser) expression() *spdxError {
	return p.binary("OR", p.and)
}

// and is term *("AND" term).
func (p *spdxParser) and() *spdxError {
	return p.binary("AND", p.term)
}

func (p *spdxParser) binary(op string, operand func() *spdxError) *spdxError {
	if err := operand(); err != nil {
		return err
	}
	for {
		t, ok := p.peek()
		if !ok || t.text != op {
			return nil
		}
		p.i++
		if err := operand(); err != nil {
			return err
		}
	}
}

// term is "(" expression ")", or a simple expression with an optional
// "WITH" exception.
func (p *spdxParser) term() *spdxError {
	t, ok := p.peek()
	switch {
	case !ok:
		return &spdxError{reasonSyntax, t.pos}
	case t.text == "(":
		p.i++
		if err := p.expression(); err != nil {
			return err
		}
		c, ok := p.peek()
		if !ok {
			return &spdxError{reasonParentheses, t.pos}
		}
		if c.text != ")" {
			return &spdxError{reasonSyntax, c.pos}
		}
		p.i++
		return nil
	case t.text == ")":
		return &spdxError{reasonParentheses, t.pos}
	case isOperator(t.text):
		return &spdxError{reasonSyntax, t.pos}
	}
	p.i++
	w, ok := p.peek()
	if !ok || w.text != "WITH" {
		return nil
	}
	p.i++
	e, ok := p.peek()
	if !ok || isOperator(e.text) || e.text == "(" || e.text == ")" || strings.HasSuffix(e.text, "+") {
		return &spdxError{reasonSyntax, e.pos}
	}
	p.i++
	return nil
}

func isOperator(s string) bool { return s == "AND" || s == "OR" || s == "WITH" }
