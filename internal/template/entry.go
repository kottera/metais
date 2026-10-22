// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package template

import (
	"regexp"
	"slices"
	"strings"
	"unicode/utf8"

	"metais.org/metais/internal/finding"
	"metais.org/metais/internal/limits"
	"metais.org/metais/internal/liquid"
	"metais.org/metais/internal/loader"
)

// ParseEntry is the one parse of a package's template, which render and
// check both make (ADR-007 D2 step 6, ADR-004 D16): entry's text, held to
// template-size and parsed under template-nesting in the dialect, every
// parse-time finding reported, at most one template.syntax, which names
// what the error is taken to come from (ADR-007 D2 item 4). ooxml says
// whether the template is rendered as a part of an Office Open XML
// container, which names two more causes. It returns the template, or nil
// and its findings.
//
// A render tag naming no template is not this parse's to report: render
// reports it as template.partial_not_found (ADR-004 D8 item 2), and check
// as its cross-reference's template.xref.missing_partial (ADR-007 D3).
func ParseEntry(entry string, text []byte, lim limits.Set, ooxml bool) (*liquid.Template, []finding.Finding) {
	t, fs, _ := parseEntry(entry, text, lim, ooxml)
	var out []finding.Finding
	for _, f := range fs {
		out = append(out, f.f)
	}
	return t, out
}

// located is a finding of a template's parse, and the byte offset in its
// text it is at, which orders a template's findings.
type located struct {
	offset int
	f      finding.Finding
}

// parseEntry is ParseEntry, its findings with their offsets, and the
// offset where the parse stopped: the first error that stops a parse, 0
// when the text was refused before it was parsed, and past the end when
// nothing stopped it.
func parseEntry(entry string, text []byte, lim limits.Set, ooxml bool) (*liquid.Template, []located, int) {
	subj := subject(entry)
	file := finding.Location{Origin: finding.OriginPackage, File: entry}
	if n := int64(len(text)); n > lim.Value(limits.TemplateSize) {
		return nil, []located{{0, lim.Breached(limits.TemplateSize, file, n, true)}}, 0
	}
	if !utf8.Valid(text) {
		off := invalidOffset(text)
		l := at(text, off, file)
		return nil, []located{{off, finding.New(CodeNotUTF8, subj, &l, nil)}}, 0
	}
	t, errs := liquid.Parse(dialect, liquid.Source{Name: entry, Text: text}, ParseLimits(lim))
	var fs []located
	var cons []liquid.Construct
	stop := len(text) + 1
	for _, e := range errs {
		cause := ""
		switch e.Code {
		case liquid.CodeSyntax:
			if cons == nil {
				cons = liquid.Constructs(string(text))
			}
			cause = Cause(text, cons, e.Offset, ooxml)
			stop = min(stop, e.Offset)
		case liquid.CodeNotUTF8, liquid.CodeLimit:
			stop = min(stop, e.Offset)
		}
		fs = append(fs, located{e.Offset, parseFinding(e, lim, cause)})
	}
	if len(fs) > 0 {
		return nil, fs, stop
	}
	return t, nil, stop
}

// ParseLimits are the bounds a package's templates are parsed under:
// ADR-009's template-nesting and template-size, from lim.
func ParseLimits(lim limits.Set) liquid.ParseLimits {
	return liquid.ParseLimits{Nesting: int(lim.Value(limits.TemplateNesting)), Size: int(lim.Value(limits.TemplateSize))}
}

// invalidOffset is the offset of text's first byte that does not begin a
// valid UTF-8 sequence.
func invalidOffset(text []byte) int {
	for i := 0; i < len(text); {
		r, n := utf8.DecodeRune(text[i:])
		if r == utf8.RuneError && n <= 1 {
			return i
		}
		i += n
	}
	return len(text)
}

// at is file located at byte off of text: its 1-based line, and its column
// in code points (ADR-006 D5 item 3), an invalid byte counting as one.
func at(text []byte, off int, file finding.Location) finding.Location {
	line, start := 1, 0
	for i := 0; i < off; i++ {
		if text[i] == '\n' {
			line, start = line+1, i+1
		}
	}
	col := 1
	for i := start; i < off; {
		_, n := utf8.DecodeRune(text[i:])
		i += n
		col++
	}
	file.Line, file.Column = line, col
	return file
}

var markupRE = regexp.MustCompile(`</?[A-Za-z_][A-Za-z0-9_.:-]*[\s/>]`)

// Cause is what the parse error at off in text is taken to come from
// (ADR-007 D2 item 4), from the construct of cons holding it: document
// markup in an Office Open XML template, typographic quotes, an escaped
// character in an Office Open XML template, and otherwise the grammar.
func Cause(text []byte, cons []liquid.Construct, off int, ooxml bool) string {
	i := slices.IndexFunc(cons, func(c liquid.Construct) bool { return c.Span.Start <= off && off < c.Span.End })
	if i < 0 {
		return loader.CauseGrammar
	}
	s := string(text[cons[i].Span.Start:cons[i].Span.End])
	switch {
	case ooxml && markupRE.MatchString(s):
		return loader.CauseMarkup
	case strings.ContainsAny(s, "‘’“”"):
		return loader.CauseQuotes
	case ooxml && escapedOutsideString(s):
		return loader.CauseEscaped
	}
	return loader.CauseGrammar
}

// escapedOutsideString reports whether s holds &lt;, &gt;, &amp;, &quot;
// or &apos; outside a Liquid string literal.
func escapedOutsideString(s string) bool {
	var quote byte
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case quote != 0:
			if c == quote {
				quote = 0
			}
		case c == '"' || c == '\'':
			quote = c
		case c == '&':
			for _, e := range []string{"&lt;", "&gt;", "&amp;", "&quot;", "&apos;"} {
				if strings.HasPrefix(s[i:], e) {
					return true
				}
			}
		}
	}
	return false
}
