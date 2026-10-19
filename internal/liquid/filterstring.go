// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package liquid

import (
	"encoding/base64"
	"regexp"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/kottera/metais/internal/liquid/value"
)

// The string filters. Each works on its input's text, and its string
// arguments' (ADR-004 D4 item 4), so a number, a boolean or an array is
// read as its text and an object is CodeObjectOutput. Lengths and
// positions count characters (Unicode scalar values), as Ruby's do.

func stringFilters() []*stdFilter {
	none := signature{}
	one := signature{min: 1, max: 1}
	replace := signature{min: 1, max: 2}
	counted := signature{max: 2, literal: countArg(0)}
	return []*stdFilter{
		{name: "append", sig: one, fn: func(c *call, in value.Value) (value.Value, *Error) {
			a, _ := c.arg(0)
			return concatText(c, in, a)
		}},
		{name: "prepend", sig: one, fn: func(c *call, in value.Value) (value.Value, *Error) {
			a, _ := c.arg(0)
			return concatText(c, a, in)
		}},
		{name: "upcase", sig: none, fn: caseMap(unicode.ToUpper)},
		{name: "downcase", sig: none, fn: caseMap(unicode.ToLower)},
		{name: "capitalize", sig: none, fn: capitalize},
		{name: "escape", sig: none, fn: escape, safe: true},
		{name: "h", sig: none, fn: escape, safe: true},
		{name: "escape_once", sig: none, fn: escapeOnce, safe: true},
		{name: "strip", sig: none, fn: trim(true, true)},
		{name: "lstrip", sig: none, fn: trim(true, false)},
		{name: "rstrip", sig: none, fn: trim(false, true)},
		{name: "squish", sig: none, fn: squish},
		{name: "strip_html", sig: none, fn: stripHTML},
		{name: "strip_newlines", sig: none, fn: newlines("")},
		{name: "newline_to_br", sig: none, fn: newlines("<br />\n")},
		{name: "remove", sig: one, fn: replaceFilter(replaceAll, false)},
		{name: "remove_first", sig: one, fn: replaceFilter(replaceFirst, false)},
		{name: "remove_last", sig: one, fn: replaceFilter(replaceLast, false)},
		{name: "replace", sig: replace, fn: replaceFilter(replaceAll, true)},
		{name: "replace_first", sig: replace, fn: replaceFilter(replaceFirst, true)},
		{name: "replace_last", sig: signature{min: 2, max: 2}, fn: replaceFilter(replaceLast, true)},
		{name: "split", sig: one, fn: split},
		{name: "truncate", sig: counted, fn: truncate},
		{name: "truncatewords", sig: counted, fn: truncateWords},
		{name: "url_encode", sig: none, fn: urlEncode},
		{name: "url_decode", sig: none, fn: urlDecode},
		{name: "base64_encode", sig: none, fn: base64Encode(base64.StdEncoding)},
		{name: "base64_url_safe_encode", sig: none, fn: base64Encode(base64.URLEncoding)},
		{name: "base64_decode", sig: none, fn: base64Decode(false)},
		{name: "base64_url_safe_decode", sig: none, fn: base64Decode(true)},
	}
}

// countArg refuses, at parse, a literal at positional argument n (0-based)
// that toInteger refuses. The arguments after it take any value.
func countArg(n int) func(string, int, string, value.Value) *Error {
	return func(filter string, arg int, keyword string, v value.Value) *Error {
		if keyword != "" || arg != n+1 {
			return nil
		}
		_, err := toInteger(filter, strconv.Itoa(arg), v)
		return err
	}
}

// str returns v's text as a string filter reads its input or an argument.
func (c *call) str(v value.Value) (string, *Error) {
	if v == nil {
		return "", nil
	}
	return c.text(v)
}

// built charges a string of n bytes, then returns the one make builds.
func (c *call) built(n int, make func() string) (value.Value, *Error) {
	if err := c.build(int64(n)); err != nil {
		return nil, err
	}
	return newString(make()), nil
}

// concatText is append and prepend: a's text followed by b's.
func concatText(c *call, a, b value.Value) (value.Value, *Error) {
	x, err := c.str(a)
	if err != nil {
		return nil, err
	}
	y, err := c.str(b)
	if err != nil {
		return nil, err
	}
	return c.built(len(x)+len(y), func() string { return x + y })
}

// caseMap maps each character by Unicode 15.0.0's simple case mapping
// (ADR-004 D1 item 4): one code point to one, no special casing, no
// locale.
func caseMap(m func(rune) rune) func(*call, value.Value) (value.Value, *Error) {
	return func(c *call, in value.Value) (value.Value, *Error) {
		s, err := c.str(in)
		if err != nil {
			return nil, err
		}
		n := 0
		for _, r := range s {
			n += utf8.RuneLen(m(r))
		}
		return c.built(n, func() string { return strings.Map(m, s) })
	}
}

// capitalize writes the first character in its simple titlecase and the
// rest in their simple lowercase, as Ruby's String#capitalize does under
// ADR-004 D1 item 4's mapping.
func capitalize(c *call, in value.Value) (value.Value, *Error) {
	s, err := c.str(in)
	if err != nil {
		return nil, err
	}
	m := func(i int, r rune) rune {
		if i == 0 {
			return unicode.ToTitle(r)
		}
		return unicode.ToLower(r)
	}
	n := 0
	for i, r := range s {
		n += utf8.RuneLen(m(i, r))
	}
	return c.built(n, func() string {
		var b strings.Builder
		b.Grow(n)
		for i, r := range s {
			b.WriteRune(m(i, r))
		}
		return b.String()
	})
}

// htmlEntity is ADR-004 D9 item 1's escape mapping.
func htmlEntity(c byte) string {
	switch c {
	case '&':
		return "&amp;"
	case '<':
		return "&lt;"
	case '>':
		return "&gt;"
	case '"':
		return "&quot;"
	case '\'':
		return "&#39;"
	}
	return ""
}

// EscapeHTML appends s to dst through ADR-004 D9 item 1's escape mapping:
// the Escape of an output whose escape is html.
func EscapeHTML(dst []byte, s string) []byte {
	for i := 0; i < len(s); i++ {
		if e := htmlEntity(s[i]); e != "" {
			dst = append(dst, e...)
		} else {
			dst = append(dst, s[i])
		}
	}
	return dst
}

// escape is ADR-004 D9's escape mapping over the input's text. nil stays
// nil.
func escape(c *call, in value.Value) (value.Value, *Error) {
	if _, ok := in.(value.Null); ok {
		return in, nil
	}
	s, err := c.str(in)
	if err != nil {
		return nil, err
	}
	return escapeWith(c, s, func(int) bool { return true })
}

// escapeOnce is escape that leaves an ampersand alone where it opens a
// character reference already: one or more ASCII letters, or "#" and one
// or more ASCII digits, then ";".
func escapeOnce(c *call, in value.Value) (value.Value, *Error) {
	if _, ok := in.(value.Null); ok {
		return in, nil
	}
	s, err := c.str(in)
	if err != nil {
		return nil, err
	}
	return escapeWith(c, s, func(i int) bool { return !openReference(s[i+1:]) })
}

// escapeWith escapes s, an ampersand at i only when amp(i).
func escapeWith(c *call, s string, amp func(int) bool) (value.Value, *Error) {
	entity := func(i int) string {
		if s[i] == '&' && !amp(i) {
			return ""
		}
		return htmlEntity(s[i])
	}
	n := 0
	for i := 0; i < len(s); i++ {
		if e := entity(i); e != "" {
			n += len(e)
		} else {
			n++
		}
	}
	return c.built(n, func() string {
		var b strings.Builder
		b.Grow(n)
		for i := 0; i < len(s); i++ {
			if e := entity(i); e != "" {
				b.WriteString(e)
			} else {
				b.WriteByte(s[i])
			}
		}
		return b.String()
	})
}

// openReference reports whether s begins with the rest of a character
// reference after its ampersand: letters, or "#" and digits, then ";".
func openReference(s string) bool {
	i := 0
	if strings.HasPrefix(s, "#") {
		i = 1
		for i < len(s) && isDigit(s[i]) {
			i++
		}
	} else {
		for i < len(s) && ('a' <= s[i]|0x20 && s[i]|0x20 <= 'z') {
			i++
		}
	}
	min := 1
	if strings.HasPrefix(s, "#") {
		min = 2
	}
	return i >= min && i < len(s) && s[i] == ';'
}

// trim removes Ruby's whitespace (rubySpace) from the start, the end or
// both.
func trim(left, right bool) func(*call, value.Value) (value.Value, *Error) {
	return func(c *call, in value.Value) (value.Value, *Error) {
		s, err := c.str(in)
		if err != nil {
			return nil, err
		}
		i, j := 0, len(s)
		for left && i < j && rubySpace(s[i]) {
			i++
		}
		for right && j > i && rubySpace(s[j-1]) {
			j--
		}
		return newString(s[i:j]), nil
	}
}

// squish strips the text, then writes each run of whitespace in it as one
// space. nil stays nil.
func squish(c *call, in value.Value) (value.Value, *Error) {
	if _, ok := in.(value.Null); ok {
		return in, nil
	}
	s, err := c.str(in)
	if err != nil {
		return nil, err
	}
	s = rubyStrip(s)
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if asciiSpace(s[i]) {
			if i > 0 && asciiSpace(s[i-1]) {
				continue
			}
			b.WriteByte(' ')
			continue
		}
		b.WriteByte(s[i])
	}
	if err := c.build(int64(b.Len())); err != nil {
		return nil, err
	}
	return newString(b.String()), nil
}

// Shopify's strip_html removes script and style elements and comments,
// each from its opening to the nearest close, and then every tag, from a
// "<" to the nearest ">". RE2 matches each in time linear in the text.
var (
	htmlBlocks = regexp.MustCompile(`(?s)<script.*?</script>|<!--.*?-->|<style.*?</style>`)
	htmlTags   = regexp.MustCompile(`(?s)<.*?>`)
)

func stripHTML(c *call, in value.Value) (value.Value, *Error) {
	s, err := c.str(in)
	if err != nil {
		return nil, err
	}
	s = htmlTags.ReplaceAllLiteralString(htmlBlocks.ReplaceAllLiteralString(s, ""), "")
	if err := c.build(int64(len(s))); err != nil {
		return nil, err
	}
	return newString(s), nil
}

// newlines replaces each line feed, and the carriage return before one,
// with with: strip_newlines and newline_to_br. A lone carriage return
// stays.
func newlines(with string) func(*call, value.Value) (value.Value, *Error) {
	return func(c *call, in value.Value) (value.Value, *Error) {
		s, err := c.str(in)
		if err != nil {
			return nil, err
		}
		lf := strings.Count(s, "\n")
		crlf := strings.Count(s, "\r\n")
		return c.built(len(s)-lf-crlf+lf*len(with), func() string {
			return strings.ReplaceAll(strings.ReplaceAll(s, "\r\n", "\n"), "\n", with)
		})
	}
}

// Where replaceFilter replaces: every occurrence, the first or the last.
const (
	replaceAll = iota
	replaceFirst
	replaceLast
)

// replaceFilter is replace, replace_first and replace_last, and, without
// a replacement, the remove filters. The pattern is the first argument's
// text, the replacement the second's, empty when omitted. An empty pattern
// matches before every character and at the end, as Ruby's gsub's does;
// its first match is at the start and its last at the end.
func replaceFilter(which int, replacing bool) func(*call, value.Value) (value.Value, *Error) {
	return func(c *call, in value.Value) (value.Value, *Error) {
		s, err := c.str(in)
		if err != nil {
			return nil, err
		}
		p, _ := c.arg(0)
		pat, err := c.str(p)
		if err != nil {
			return nil, err
		}
		var with string
		if r, ok := c.arg(1); ok && replacing {
			if with, err = c.str(r); err != nil {
				return nil, err
			}
		}
		switch which {
		case replaceAll:
			n := strings.Count(s, pat)
			if pat == "" {
				n = utf8.RuneCountInString(s) + 1
			}
			return c.built(len(s)+n*(len(with)-len(pat)), func() string {
				return strings.ReplaceAll(s, pat, with)
			})
		case replaceFirst:
			i := strings.Index(s, pat)
			if i < 0 {
				return newString(s), nil
			}
			return c.built(len(s)+len(with)-len(pat), func() string { return s[:i] + with + s[i+len(pat):] })
		}
		i := strings.LastIndex(s, pat)
		if i < 0 {
			return newString(s), nil
		}
		return c.built(len(s)+len(with)-len(pat), func() string { return s[:i] + with + s[i+len(pat):] })
	}
}

// split splits the input's text at each occurrence of the argument's
// text, as Ruby's String#split does: a single space splits at runs of
// whitespace (asciiSpace) and ignores leading whitespace, an empty
// pattern splits into characters, and empty fields at the end are
// dropped. Empty text gives the empty array.
func split(c *call, in value.Value) (value.Value, *Error) {
	s, err := c.str(in)
	if err != nil {
		return nil, err
	}
	p, _ := c.arg(0)
	pat, err := c.str(p)
	if err != nil {
		return nil, err
	}
	var parts []string
	switch pat {
	case " ":
		parts = fields(s, -1)
	case "":
		parts = make([]string, 0, utf8.RuneCountInString(s))
		for i, r := range s {
			parts = append(parts, s[i:i+utf8.RuneLen(r)])
		}
	default:
		parts = strings.Split(s, pat)
	}
	for len(parts) > 0 && parts[len(parts)-1] == "" {
		parts = parts[:len(parts)-1]
	}
	if err := c.result(int64(len(parts))); err != nil {
		return nil, err
	}
	out := make([]value.Value, len(parts))
	for i, p := range parts {
		out[i] = newString(p)
	}
	return newArray(out), nil
}

// fields is Ruby's awk-style split of s with limit n: leading whitespace
// ignored, fields separated by runs of whitespace, and with n > 0 at most
// n fields, the last holding the rest of s after the whitespace before
// it. n < 0 has no limit, and then trailing whitespace makes no field.
func fields(s string, n int) []string {
	var out []string
	i := 0
	for {
		for i < len(s) && asciiSpace(s[i]) {
			i++
		}
		if i == len(s) {
			return out
		}
		if n > 0 && len(out) == n-1 {
			return append(out, s[i:])
		}
		j := i
		for j < len(s) && !asciiSpace(s[j]) {
			j++
		}
		out = append(out, s[i:j])
		i = j
	}
}

// count returns positional argument i coerced by toInteger, or def when it
// is omitted.
func (c *call) count(i int, def int64) (int64, *Error) {
	a, ok := c.arg(i)
	if !ok {
		return def, nil
	}
	return toInteger(c.name, strconv.Itoa(i+1), a)
}

// ellipsis returns positional argument i's text, or "..." when it is
// omitted.
func (c *call) ellipsis(i int) (string, *Error) {
	a, ok := c.arg(i)
	if !ok {
		return "...", nil
	}
	return c.str(a)
}

// truncate shortens the text to the first argument's number of
// characters, 50 when omitted, the second argument's text ("..." when
// omitted) ending it within that count. Text no longer than the count is
// unchanged. nil stays nil.
func truncate(c *call, in value.Value) (value.Value, *Error) {
	if _, ok := in.(value.Null); ok {
		return in, nil
	}
	s, err := c.str(in)
	if err != nil {
		return nil, err
	}
	n, err := c.count(0, 50)
	if err != nil {
		return nil, err
	}
	end, err := c.ellipsis(1)
	if err != nil {
		return nil, err
	}
	if int64(utf8.RuneCountInString(s)) <= n {
		return newString(s), nil
	}
	keep := max(n-int64(utf8.RuneCountInString(end)), 0)
	cut := len(s)
	for i := range s {
		if keep == 0 {
			cut = i
			break
		}
		keep--
	}
	return c.built(cut+len(end), func() string { return s[:cut] + end })
}

// truncateWords keeps the first argument's number of words, 15 when
// omitted and 1 when it is less than 1, joined by single spaces and
// followed by the second argument's text ("..." when omitted). Text of no
// more words is unchanged. nil stays nil.
func truncateWords(c *call, in value.Value) (value.Value, *Error) {
	if _, ok := in.(value.Null); ok {
		return in, nil
	}
	s, err := c.str(in)
	if err != nil {
		return nil, err
	}
	n, err := c.count(0, 15)
	if err != nil {
		return nil, err
	}
	end, err := c.ellipsis(1)
	if err != nil {
		return nil, err
	}
	n = max(n, 1)
	limit := -1
	if n < int64(len(s)) {
		limit = int(n) + 1
	}
	words := fields(s, limit)
	if int64(len(words)) <= n {
		return newString(s), nil
	}
	words = words[:n]
	size := len(end) + len(words) - 1
	for _, w := range words {
		size += len(w)
	}
	return c.built(size, func() string { return strings.Join(words, " ") + end })
}

// urlEncode is Ruby's CGI.escape: ASCII letters, digits and "_.-~" stay,
// a space becomes "+", and every other byte "%XX" in upper-case hex. nil
// stays nil.
func urlEncode(c *call, in value.Value) (value.Value, *Error) {
	if _, ok := in.(value.Null); ok {
		return in, nil
	}
	s, err := c.str(in)
	if err != nil {
		return nil, err
	}
	keep := func(b byte) bool {
		return 'a' <= b|0x20 && b|0x20 <= 'z' || isDigit(b) || strings.IndexByte("_.-~ ", b) >= 0
	}
	n := 0
	for i := 0; i < len(s); i++ {
		if keep(s[i]) {
			n++
		} else {
			n += 3
		}
	}
	return c.built(n, func() string {
		const hex = "0123456789ABCDEF"
		b := make([]byte, 0, n)
		for i := 0; i < len(s); i++ {
			switch x := s[i]; {
			case x == ' ':
				b = append(b, '+')
			case keep(x):
				b = append(b, x)
			default:
				b = append(b, '%', hex[x>>4], hex[x&15])
			}
		}
		return string(b)
	})
}

// urlDecode is Ruby's CGI.unescape: "+" becomes a space and "%XX" the
// byte it names; a "%" not followed by two hex digits stays. A result that
// is not UTF-8 is CodeInvalidUTF8. nil stays nil.
func urlDecode(c *call, in value.Value) (value.Value, *Error) {
	if _, ok := in.(value.Null); ok {
		return in, nil
	}
	s, err := c.str(in)
	if err != nil {
		return nil, err
	}
	b := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		switch {
		case s[i] == '+':
			b = append(b, ' ')
		case s[i] == '%' && i+2 < len(s) && isHex(s[i+1]) && isHex(s[i+2]):
			b = append(b, unhex(s[i+1])<<4|unhex(s[i+2]))
			i += 2
		default:
			b = append(b, s[i])
		}
	}
	return decoded(c, b)
}

func isHex(b byte) bool { return isDigit(b) || 'a' <= b|0x20 && b|0x20 <= 'f' }

func unhex(b byte) byte {
	if isDigit(b) {
		return b - '0'
	}
	return b | 0x20 - 'a' + 10
}

// decoded returns decoded bytes as a string, refusing bytes that are not
// UTF-8.
func decoded(c *call, b []byte) (value.Value, *Error) {
	if !utf8.Valid(b) {
		return nil, filterErr(c.name, CodeInvalidUTF8, nil, "the decoded bytes are not UTF-8")
	}
	if err := c.build(int64(len(b))); err != nil {
		return nil, err
	}
	return newString(string(b)), nil
}

// base64Encode encodes the text's UTF-8 bytes, padded, in enc's alphabet:
// RFC 4648's base64, or its URL-safe alphabet.
func base64Encode(enc *base64.Encoding) func(*call, value.Value) (value.Value, *Error) {
	return func(c *call, in value.Value) (value.Value, *Error) {
		s, err := c.str(in)
		if err != nil {
			return nil, err
		}
		return c.built(enc.EncodedLen(len(s)), func() string { return enc.EncodeToString([]byte(s)) })
	}
}

// base64Decode decodes the text as Ruby's strict_decode64 does: RFC 4648
// base64, padded, with nothing else, not even a line break. urlSafe is
// urlsafe_decode64, which reads "-" and "_" as "+" and "/" and adds the
// padding the text omits. Text that does not decode is CodeFilterInput,
// and bytes that are not UTF-8 are CodeInvalidUTF8.
func base64Decode(urlSafe bool) func(*call, value.Value) (value.Value, *Error) {
	return func(c *call, in value.Value) (value.Value, *Error) {
		s, err := c.str(in)
		if err != nil {
			return nil, err
		}
		if urlSafe {
			if !strings.HasSuffix(s, "=") && len(s)%4 != 0 {
				s += strings.Repeat("=", 4-len(s)%4)
			}
			s = strings.NewReplacer("-", "+", "_", "/").Replace(s)
		}
		if strings.ContainsAny(s, "\r\n") {
			return nil, inputErr(c.name, in, "the text is not base64")
		}
		b, derr := base64.StdEncoding.Strict().DecodeString(s)
		if derr != nil {
			return nil, inputErr(c.name, in, "the text is not base64")
		}
		return decoded(c, b)
	}
}
