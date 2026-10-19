// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package markdown

import (
	"bytes"
)

// rawHTMLTags are the tag names of HTML block start condition 1 (§4.6).
var rawHTMLTags = []string{"pre", "script", "style", "textarea"}

// blockTags are the tag names of HTML block start condition 6 (§4.6).
var blockTags = map[string]bool{}

func init() {
	for _, t := range []string{
		"address", "article", "aside", "base", "basefont", "blockquote", "body",
		"caption", "center", "col", "colgroup", "dd", "details", "dialog",
		"dir", "div", "dl", "dt", "fieldset", "figcaption", "figure",
		"footer", "form", "frame", "frameset",
		"h1", "h2", "h3", "h4", "h5", "h6", "head", "header", "hr",
		"html", "iframe", "legend", "li", "link", "main", "menu", "menuitem",
		"nav", "noframes", "ol", "optgroup", "option", "p", "param",
		"search", "section", "summary", "table", "tbody", "td",
		"tfoot", "th", "thead", "title", "tr", "track", "ul",
	} {
		blockTags[t] = true
	}
}

func isASCIILetter(b byte) bool { return b|0x20 >= 'a' && b|0x20 <= 'z' }
func isASCIIDigit(b byte) bool  { return b >= '0' && b <= '9' }

// hasPrefixFold reports whether s begins with the lower-case ASCII prefix,
// ignoring ASCII case.
func hasPrefixFold(s []byte, prefix string) bool {
	if len(s) < len(prefix) {
		return false
	}
	for i := 0; i < len(prefix); i++ {
		if s[i]|0x20 != prefix[i] && s[i] != prefix[i] {
			return false
		}
	}
	return true
}

// htmlBlockStart returns the start condition, 1 to 7, that the line s
// meets (§4.6), or 0. s begins at the line's first non-space byte.
// Condition 7 cannot interrupt a paragraph, and allow7 says whether it may
// be met.
func htmlBlockStart(s []byte, allow7 bool) int {
	if len(s) < 2 || s[0] != '<' {
		return 0
	}
	for _, t := range rawHTMLTags {
		if hasPrefixFold(s[1:], t) {
			if n := 1 + len(t); n == len(s) || isSpaceOrTab(s[n]) || s[n] == '>' {
				return 1
			}
		}
	}
	switch {
	case bytes.HasPrefix(s, []byte("<!--")):
		return 2
	case s[1] == '?':
		return 3
	case bytes.HasPrefix(s, []byte("<![CDATA[")):
		return 5
	case s[1] == '!' && len(s) > 2 && isASCIILetter(s[2]):
		return 4
	}
	j := 1
	if s[j] == '/' {
		j++
	}
	k := j
	for k < len(s) && (isASCIILetter(s[k]) || isASCIIDigit(s[k])) {
		k++
	}
	if k > j && blockTags[string(bytes.ToLower(s[j:k]))] {
		if k == len(s) || isSpaceOrTab(s[k]) || s[k] == '>' || bytes.HasPrefix(s[k:], []byte("/>")) {
			return 6
		}
	}
	if !allow7 {
		return 0
	}
	var end int
	var ok bool
	if s[1] == '/' {
		end, ok = closingTag(s, 0)
	} else {
		end, ok = openTag(s, 0, nil)
		for _, t := range rawHTMLTags {
			if ok && hasPrefixFold(s[1:], t) && (len(s) == 1+len(t) || !isTagNameByte(s[1+len(t)])) {
				ok = false
			}
		}
	}
	if ok && isBlank(s[end:]) {
		return 7
	}
	return 0
}

// htmlBlockEnds reports whether the line s meets the end condition of an
// HTML block of start condition kind (§4.6). Conditions 6 and 7 end at a
// blank line, which ends the block before it is read.
func htmlBlockEnds(kind int, s []byte) bool {
	switch kind {
	case 1:
		l := bytes.ToLower(s)
		for _, t := range rawHTMLTags {
			if bytes.Contains(l, []byte("</"+t+">")) {
				return true
			}
		}
	case 2:
		return bytes.Contains(s, []byte("-->"))
	case 3:
		return bytes.Contains(s, []byte("?>"))
	case 4:
		return bytes.IndexByte(s, '>') >= 0
	case 5:
		return bytes.Contains(s, []byte("]]>"))
	}
	return false
}

func isTagNameByte(b byte) bool { return isASCIILetter(b) || isASCIIDigit(b) || b == '-' }

func isAttrNameStart(b byte) bool { return isASCIILetter(b) || b == '_' || b == ':' }

func isAttrNameByte(b byte) bool {
	return isASCIILetter(b) || isASCIIDigit(b) || b == '_' || b == '.' || b == ':' || b == '-'
}

// htmlMemo remembers, for one heading's text, the offsets from which a
// raw HTML construct's terminator or a quoted attribute value's closing
// quote is known to be absent, so that no later scan reads that far again.
type htmlMemo struct {
	none map[string]int // terminator → the offset from which it does not occur
}

func newHTMLMemo() *htmlMemo { return &htmlMemo{none: map[string]int{}} }

// find returns the offset of the first term at or after i, or -1.
func (m *htmlMemo) find(s []byte, i int, term string) int {
	if m != nil {
		if from, ok := m.none[term]; ok && i >= from {
			return -1
		}
	}
	k := bytes.Index(s[i:], []byte(term))
	if k < 0 {
		if m != nil {
			m.none[term] = i
		}
		return -1
	}
	return i + k
}

// tagName reads a tag name at s[i]: an ASCII letter, then letters, digits
// and hyphens.
func tagName(s []byte, i int) (int, bool) {
	if i >= len(s) || !isASCIILetter(s[i]) {
		return 0, false
	}
	i++
	for i < len(s) && isTagNameByte(s[i]) {
		i++
	}
	return i, true
}

// openTag reads an open tag at s[i], which is '<' (§6.6).
func openTag(s []byte, i int, memo *htmlMemo) (int, bool) {
	j, ok := tagName(s, i+1)
	if !ok {
		return 0, false
	}
	for {
		k := skipLinkSpace(s, j)
		if k == j || k >= len(s) || !isAttrNameStart(s[k]) {
			j = k
			break
		}
		j = k + 1
		for j < len(s) && isAttrNameByte(s[j]) {
			j++
		}
		m := skipLinkSpace(s, j)
		if m < len(s) && s[m] == '=' {
			v := skipLinkSpace(s, m+1)
			end, ok := attrValue(s, v, memo)
			if !ok {
				return 0, false
			}
			j = end
		}
	}
	if j < len(s) && s[j] == '/' {
		j++
	}
	if j < len(s) && s[j] == '>' {
		return j + 1, true
	}
	return 0, false
}

// attrValue reads an attribute value at s[i] (§6.6).
func attrValue(s []byte, i int, memo *htmlMemo) (int, bool) {
	if i >= len(s) {
		return 0, false
	}
	switch q := s[i]; q {
	case '"', '\'':
		k := memo.find(s, i+1, string(q))
		if k < 0 {
			return 0, false
		}
		return k + 1, true
	}
	j := i
	for j < len(s) && bytes.IndexByte([]byte(" \t\n\"'=<>`"), s[j]) < 0 {
		j++
	}
	return j, j > i
}

// closingTag reads a closing tag at s[i], which is '<' (§6.6).
func closingTag(s []byte, i int) (int, bool) {
	if i+1 >= len(s) || s[i+1] != '/' {
		return 0, false
	}
	j, ok := tagName(s, i+2)
	if !ok {
		return 0, false
	}
	j = skipLinkSpace(s, j)
	if j < len(s) && s[j] == '>' {
		return j + 1, true
	}
	return 0, false
}

// rawHTML reads an inline HTML tag at s[i], which is '<' (§6.6): an open
// tag, a closing tag, a comment, a processing instruction, a declaration
// or a CDATA section.
func rawHTML(s []byte, i int, memo *htmlMemo) (int, bool) {
	if i+1 >= len(s) {
		return 0, false
	}
	switch s[i+1] {
	case '/':
		return closingTag(s, i)
	case '?':
		if k := memo.find(s, i+2, "?>"); k >= 0 {
			return k + 2, true
		}
		return 0, false
	case '!':
		rest := s[i:]
		switch {
		case bytes.HasPrefix(rest, []byte("<!-->")):
			return i + 5, true
		case bytes.HasPrefix(rest, []byte("<!--->")):
			return i + 6, true
		case bytes.HasPrefix(rest, []byte("<!--")):
			if k := memo.find(s, i+4, "-->"); k >= 0 {
				return k + 3, true
			}
		case bytes.HasPrefix(rest, []byte("<![CDATA[")):
			if k := memo.find(s, i+9, "]]>"); k >= 0 {
				return k + 3, true
			}
		case len(rest) > 2 && isASCIILetter(rest[2]):
			if k := memo.find(s, i+3, ">"); k >= 0 {
				return k + 1, true
			}
		}
		return 0, false
	}
	return openTag(s, i, memo)
}
