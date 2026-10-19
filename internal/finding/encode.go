// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package finding

import (
	"strconv"
	"strings"
	"unicode/utf8"
)

// neutralised reports whether r is written as a \u escape wherever package
// or input text is printed (ADR-006 D9 item 9): a C0 control, line breaks
// and tabs included, DEL, a C1 control, or a bidirectional control.
func neutralised(r rune) bool {
	return r < 0x20 || (r >= 0x7f && r <= 0x9f) || (r >= 0x202a && r <= 0x202e) || (r >= 0x2066 && r <= 0x2069)
}

const hexDigits = "0123456789abcdef"

func appendEscape(b []byte, r rune) []byte {
	return append(b, '\\', 'u', hexDigits[r>>12&0xf], hexDigits[r>>8&0xf], hexDigits[r>>4&0xf], hexDigits[r&0xf])
}

// appendText appends s with every neutralised character written as a \u
// escape, and every invalid UTF-8 byte as U+FFFD.
func appendText(b []byte, s string) []byte {
	for i := 0; i < len(s); {
		r, n := utf8.DecodeRuneInString(s[i:])
		switch {
		case r == utf8.RuneError && n == 1:
			b = utf8.AppendRune(b, utf8.RuneError)
		case neutralised(r):
			b = appendEscape(b, r)
		default:
			b = append(b, s[i:i+n]...)
		}
		i += n
	}
	return b
}

// appendJSONString appends s as a JSON string (ADR-006 D10 item 1): "
// and \ escaped with a backslash, the neutralised characters as \u
// escapes, and nothing else escaped.
func appendJSONString(b []byte, s string) []byte {
	b = append(b, '"')
	for i := 0; i < len(s); {
		r, n := utf8.DecodeRuneInString(s[i:])
		switch {
		case r == '"' || r == '\\':
			b = append(b, '\\', byte(r))
		case r == utf8.RuneError && n == 1:
			b = utf8.AppendRune(b, utf8.RuneError)
		case neutralised(r):
			b = appendEscape(b, r)
		default:
			b = append(b, s[i:i+n]...)
		}
		i += n
	}
	return append(b, '"')
}

// object writes one JSON object's members in the order they are added,
// omitting none: the caller decides presence.
type object struct {
	b     []byte
	first bool
}

func openObject(b []byte) *object { return &object{b: append(b, '{'), first: true} }

func (o *object) key(k string) {
	if !o.first {
		o.b = append(o.b, ',')
	}
	o.first = false
	o.b = appendJSONString(o.b, k)
	o.b = append(o.b, ':')
}

func (o *object) str(k, v string) { o.key(k); o.b = appendJSONString(o.b, v) }
func (o *object) num(k string, v int) {
	o.key(k)
	o.b = strconv.AppendInt(o.b, int64(v), 10)
}
func (o *object) strIf(k, v string) {
	if v != "" {
		o.str(k, v)
	}
}
func (o *object) numIf(k string, v int) {
	if v != 0 {
		o.num(k, v)
	}
}
func (o *object) close() []byte { return append(o.b, '}') }

// AppendJSON appends fs as a JSON array of ADR-006 D10's finding objects:
// compact, members in D1's order, absent members omitted, params in the
// registry's order.
func AppendJSON(b []byte, fs ...Finding) []byte {
	b = append(b, '[')
	for i, f := range fs {
		if i > 0 {
			b = append(b, ',')
		}
		b = f.appendJSON(b)
	}
	return append(b, ']')
}

func (f Finding) appendJSON(b []byte) []byte {
	o := openObject(b)
	o.str("code", string(f.entry.Code))
	o.str("severity", f.entry.Severity.String())
	o.str("message", f.Message())
	o.str("hint", f.Hint())
	o.key("subject")
	s := openObject(o.b)
	s.str("kind", f.subject.Kind.String())
	s.strIf("name", f.subject.Name)
	s.strIf("title", f.subject.Title)
	o.b = s.close()
	if f.location != nil {
		o.key("location")
		o.b = appendLocation(o.b, *f.location, "")
	}
	if f.schema != nil {
		o.key("schema")
		s := openObject(o.b)
		s.str("entry", f.schema.Entry)
		s.str("pointer", f.schema.Pointer)
		s.str("keyword", f.schema.Keyword)
		s.str("keyword_location", f.schema.KeywordLocation)
		o.b = s.close()
	}
	if f.field != nil {
		o.key("field")
		s := openObject(o.b)
		s.strIf("title", f.field.Title)
		s.strIf("description", f.field.Description)
		o.b = s.close()
	}
	o.key("params")
	p := openObject(o.b)
	for _, spec := range f.entry.Params {
		if v, ok := f.params[spec.Name]; ok {
			p.key(spec.Name)
			p.b = v.appendJSON(p.b, appendJSONString)
		}
	}
	o.b = p.close()
	if f.via != nil {
		o.key("via")
		o.b = append(o.b, '[')
		for i, v := range f.via {
			if i > 0 {
				o.b = append(o.b, ',')
			}
			s := openObject(o.b)
			s.str("keyword", v.Keyword)
			s.str("schema_pointer", v.SchemaPointer)
			if v.Keyword == "anyOf" || v.Keyword == "oneOf" {
				s.num("branch", v.Branch)
			}
			s.strIf("title", v.Title)
			if v.Condition != nil {
				s.key("condition")
				c := openObject(s.b)
				c.str("schema_pointer", v.Condition.SchemaPointer)
				c.strIf("title", v.Condition.Title)
				s.b = c.close()
			}
			o.b = s.close()
		}
		o.b = append(o.b, ']')
	}
	if f.alternatives != nil {
		o.key("alternatives")
		o.b = append(o.b, '[')
		for i, a := range f.alternatives {
			if i > 0 {
				o.b = append(o.b, ',')
			}
			s := openObject(o.b)
			s.num("branch", a.Branch)
			s.strIf("title", a.Title)
			s.key("findings")
			s.b = AppendJSON(s.b, a.Findings...)
			s.num("omitted", a.Omitted)
			o.b = s.close()
		}
		o.b = append(o.b, ']')
	}
	if f.related != nil {
		o.key("related")
		o.b = append(o.b, '[')
		for i, r := range f.related {
			if i > 0 {
				o.b = append(o.b, ',')
			}
			o.b = appendLocation(o.b, r.Location, r.Note)
		}
		o.b = append(o.b, ']')
	}
	return o.close()
}

// appendLocation appends l's members in D10 item 1's order, and note when
// it is a related location's.
func appendLocation(b []byte, l Location, note string) []byte {
	o := openObject(b)
	o.str("origin", l.Origin.String())
	o.strIf("file", l.File)
	o.numIf("argument", l.Argument)
	o.numIf("line", l.Line)
	o.numIf("column", l.Column)
	if l.HasPointer {
		o.str("pointer", l.pointer())
	}
	o.strIf("note", note)
	return o.close()
}

// AppendJSON appends s as ADR-006 D10 item 3's summary object.
func (s Summary) AppendJSON(b []byte) []byte {
	o := openObject(b)
	o.num("errors", s.Errors)
	o.num("warnings", s.Warnings)
	o.num("info", s.Info)
	o.num("listed", s.Listed)
	o.num("omitted", s.Omitted)
	o.key("by_code")
	c := openObject(o.b)
	for _, cc := range s.ByCode {
		c.num(string(cc.Code), cc.Count)
	}
	o.b = c.close()
	return o.close()
}

// The SGR sequences that colour a severity word (ADR-006 D11 item 4,
// ADR-008 D12 item 2).
var severityColour = [...]string{SeverityError: "\x1b[31m", SeverityWarning: "\x1b[33m", SeverityInfo: "\x1b[36m"}

const sgrReset = "\x1b[0m"

// maxDescription is how many code points of a field's description the
// human report prints (ADR-006 D11 item 1).
const maxDescription = 160

// AppendHuman appends fs as ADR-006 D11's human report, without the
// summary line. colour is whether to colour the severity word, which the
// caller decides by ADR-008's colour policy. Every text from the package
// or the inputs is neutralised (D9 item 9).
func AppendHuman(b []byte, colour bool, fs ...Finding) []byte {
	for _, f := range fs {
		b = f.appendHuman(b, colour, "")
	}
	return b
}

func (f Finding) appendHuman(b []byte, colour bool, indent string) []byte {
	b = append(b, indent...)
	b = appendText(b, f.where())
	b = append(b, ": "...)
	if colour {
		b = append(b, severityColour[f.entry.Severity]...)
	}
	b = append(b, f.entry.Severity.String()...)
	if colour {
		b = append(b, sgrReset...)
	}
	b = append(b, ": "...)
	b = appendText(b, f.Message())
	b = append(b, " ["...)
	b = append(b, f.entry.Code...)
	b = append(b, "]\n"...)
	line := func(label, text string) {
		b = append(b, indent...)
		b = append(b, "  "...)
		b = append(b, label...)
		b = append(b, ": "...)
		b = appendText(b, text)
		b = append(b, '\n')
	}
	if l := f.location; l != nil && l.HasPointer {
		at := l.pointer()
		if f.subject.Kind == SubjectInput {
			at = Pointer(f.subject.Name) + at
		} else if at == "" {
			at = `""`
		}
		if f.subject.Name != "" {
			about := f.subject.Kind.String() + " " + f.subject.Name
			if f.subject.Title != "" {
				about += ": " + f.subject.Title
			}
			at += " (" + about + ")"
		}
		b = append(b, indent...)
		b = append(b, "  at "...)
		b = appendText(b, at)
		b = append(b, '\n')
	}
	if fd := f.field; fd != nil {
		text := fd.Title
		if d := cutDescription(fd.Description); d != "" {
			if text != "" {
				text += " — "
			}
			text += d
		}
		line("field", text)
	}
	if s := f.schema; s != nil {
		at := s.Pointer
		if tokens := pointerTokens(at); len(tokens) > 0 && tokens[len(tokens)-1] == s.Keyword {
			at = Pointer(tokens[:len(tokens)-1]...)
		}
		rule := s.Keyword
		if at != "" {
			rule += " at " + at
		}
		line("rule", rule+" in "+s.Entry)
	}
	for _, v := range f.via {
		switch {
		case v.Condition != nil:
			text := v.Condition.Title
			if text == "" {
				text = "the condition at " + v.Condition.SchemaPointer
			}
			line("when", text)
		default:
			text := strconv.Itoa(v.Branch + 1)
			if v.Title != "" {
				text += ": " + v.Title
			}
			line("option", text)
		}
	}
	line("hint", f.Hint())
	for _, a := range f.alternatives {
		b = append(b, indent...)
		b = append(b, "  option "...)
		b = strconv.AppendInt(b, int64(a.Branch+1), 10)
		if a.Title != "" {
			b = append(b, ": "...)
			b = appendText(b, a.Title)
		}
		b = append(b, '\n')
		for _, af := range a.Findings {
			b = af.appendHuman(b, colour, indent+"    ")
		}
		if a.Omitted > 0 {
			b = append(b, indent...)
			b = append(b, "    "...)
			b = strconv.AppendInt(b, int64(a.Omitted), 10)
			b = append(b, " more in this option\n"...)
		}
	}
	return b
}

// where is the first line's place (ADR-006 D11 item 1), in its order of
// preference: file:line:column, file:line, file, the argument, the input,
// the package, or metais.
func (f Finding) where() string {
	if l := f.location; l != nil {
		switch {
		case l.File != "" && l.Line > 0 && l.Column > 0:
			return l.File + ":" + strconv.Itoa(l.Line) + ":" + strconv.Itoa(l.Column)
		case l.File != "" && l.Line > 0:
			return l.File + ":" + strconv.Itoa(l.Line)
		case l.File != "":
			return l.File
		case l.Origin == OriginSet && l.Argument > 0:
			return "--set #" + strconv.Itoa(l.Argument)
		case l.Origin == OriginInputsJSON && l.Argument > 0:
			return "--inputs-json #" + strconv.Itoa(l.Argument)
		}
	}
	switch f.subject.Kind {
	case SubjectInvocation, SubjectReport:
		return "metais"
	case SubjectInput, SubjectSchema:
		if f.subject.Name != "" {
			return "input " + f.subject.Name
		}
	}
	return "package"
}

// cutDescription is a description as the field line prints it: up to its
// first line break, and at most maxDescription code points, the cut marked
// with an ellipsis.
func cutDescription(d string) string {
	cut := false
	if i := strings.IndexAny(d, "\r\n"); i >= 0 {
		d, cut = d[:i], true
	}
	if utf8.RuneCountInString(d) > maxDescription {
		n := 0
		for i := range d {
			if n == maxDescription {
				d, cut = d[:i], true
				break
			}
			n++
		}
	}
	if cut && d != "" {
		d += "…"
	}
	return d
}

// AppendHuman appends s as the human report's last line (ADR-006 D11 item
// 3): the counts by severity and, when findings were omitted, how many.
func (s Summary) AppendHuman(b []byte) []byte {
	b = strconv.AppendInt(b, int64(s.Errors), 10)
	b = append(b, plural(s.Errors, " error", " errors")...)
	b = append(b, ", "...)
	b = strconv.AppendInt(b, int64(s.Warnings), 10)
	b = append(b, plural(s.Warnings, " warning", " warnings")...)
	b = append(b, ", "...)
	b = strconv.AppendInt(b, int64(s.Info), 10)
	b = append(b, " info"...)
	if s.Omitted > 0 {
		b = append(b, " ("...)
		b = strconv.AppendInt(b, int64(s.Listed), 10)
		b = append(b, " listed, "...)
		b = strconv.AppendInt(b, int64(s.Omitted), 10)
		b = append(b, " more not shown; use --json for the totals by code)"...)
	}
	return append(b, '\n')
}

func plural(n int, one, many string) string {
	if n == 1 {
		return one
	}
	return many
}

// AppendText appends s as human output writes text from a package, an
// input or a host's directory (ADR-006 D9 item 9): every C0, DEL, C1 and
// bidirectional control as a \u escape, and every invalid UTF-8 byte as
// U+FFFD.
func AppendText(b []byte, s string) []byte { return appendText(b, s) }
