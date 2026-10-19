// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package markdown

import (
	"html"
	"sort"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

// The inline pass reads a heading's raw content by CommonMark 0.31.2 §6
// and gives its plain text (ADR-008 D5, outline item 2; ADR-010 D9 item 2):
// the literal characters of its text, of its code spans and of its
// autolinks, with backslash escapes and entity references decoded, the
// markup of emphasis, links and images removed, raw HTML dropped, and each
// line ending a single space.
//
// The content is read once from left to right. What would otherwise be
// read again is bounded or remembered: a code span's closing backtick
// string is found through an index of the backtick strings, emphasis is
// matched with the openers_bottom bounds of §6.2's algorithm, a link
// label holds at most 999 characters, a destination's parentheses nest at
// most 32 deep, and a title, an attribute value or an HTML construct whose
// terminator is absent from some offset is not looked for again past it.

// piece is a run of the output: text, or a delimiter run's remaining
// characters.
type piece struct {
	text string
	ch   byte // '*' or '_' for a delimiter run
	n    int  // a delimiter run's remaining length
}

// delim is a delimiter run on the delimiter stack (§6.2). The stack is a
// list linked through prev and next, in source order, so a delimiter's
// index in the inline's slice also orders it.
type delim struct {
	piece             int
	ch                byte
	length            int // the run's length as written
	canOpen, canClose bool
	prev, next        int // -1 at either end
	gone              bool
}

// opener is a '[' or '![' on the bracket stack (§6.3).
type opener struct {
	piece  int
	start  int // the offset of the link text
	image  bool
	bottom int // the last delimiter when it was pushed, -1 for none
}

type inline struct {
	s    []byte
	refs map[string]bool

	pieces []piece
	buf    []byte // text not yet a piece

	ds         []delim
	head, tail int

	brackets []opener
	// inactiveBelow: the link openers below this height are inactive, as
	// a link cannot contain another (§6.3).
	inactiveBelow int

	ticks tickIndex
	// bracketAt is the offset of every unescaped bracket, in order, so
	// that whether a link's text holds one, and so is no label, is known
	// without reading the text again for each ']'.
	bracketAt []int
	titles    *titleMemo
	tags      *htmlMemo
}

// flatten returns the plain text of a heading's raw content s, whose link
// references resolve against the document's definitions refs.
func flatten(s []byte, refs map[string]bool) string {
	in := &inline{s: s, refs: refs, head: -1, tail: -1, ticks: newTickIndex(s), bracketAt: bracketOffsets(s), titles: newTitleMemo(), tags: newHTMLMemo()}
	in.read()
	in.flush()
	in.emphasis(-1)
	var b strings.Builder
	for _, p := range in.pieces {
		if p.ch != 0 {
			b.WriteString(strings.Repeat(string(p.ch), p.n))
		} else {
			b.WriteString(p.text)
		}
	}
	// U+0000 is replaced by U+FFFD wherever it is (§2.3), a code span
	// included.
	return strings.ReplaceAll(b.String(), "\x00", "\uFFFD")
}

func (in *inline) flush() {
	if len(in.buf) > 0 {
		in.pieces = append(in.pieces, piece{text: string(in.buf)})
		in.buf = in.buf[:0]
	}
}

func (in *inline) read() {
	s := in.s
	for i := 0; i < len(s); {
		switch c := s[i]; c {
		case '\n':
			// A line ending, soft or hard, is a space; the next line's
			// leading spaces and tabs were stripped.
			in.buf = append(in.buf, ' ')
			i = skipSpaceTab(s, i+1)
		case ' ':
			// The spaces at the end of a line are not kept (§6.7, §6.8).
			j := i
			for j < len(s) && s[j] == ' ' {
				j++
			}
			if j == len(s) || s[j] != '\n' {
				in.buf = append(in.buf, s[i:j]...)
			}
			i = j
		case '\\':
			switch {
			case i+1 < len(s) && s[i+1] == '\n':
				in.buf = append(in.buf, ' ')
				i = skipSpaceTab(s, i+2)
			case i+1 < len(s) && isASCIIPunct(s[i+1]):
				in.buf = append(in.buf, s[i+1])
				i += 2
			default:
				in.buf = append(in.buf, '\\')
				i++
			}
		case '`':
			i = in.codeSpan(i)
		case '*', '_':
			i = in.delimiterRun(i)
		case '[':
			in.push(i+1, false, "[")
			i++
		case '!':
			if i+1 < len(s) && s[i+1] == '[' {
				in.push(i+2, true, "![")
				i += 2
			} else {
				in.buf = append(in.buf, '!')
				i++
			}
		case ']':
			i = in.closeBracket(i)
		case '&':
			if text, end, ok := entity(s, i); ok {
				in.buf = append(in.buf, text...)
				i = end
			} else {
				in.buf = append(in.buf, '&')
				i++
			}
		case '<':
			if end, ok := autolink(s, i); ok {
				in.buf = append(in.buf, s[i+1:end-1]...)
				i = end
			} else if end, ok := rawHTML(s, i, in.tags); ok {
				i = end
			} else {
				in.buf = append(in.buf, '<')
				i++
			}
		default:
			in.buf = append(in.buf, c)
			i++
		}
	}
}

// codeSpan reads a backtick string at s[i] (§6.1): a code span when a
// backtick string of the same length follows, and literal backticks
// otherwise.
func (in *inline) codeSpan(i int) int {
	s := in.s
	n := 0
	for i+n < len(s) && s[i+n] == '`' {
		n++
	}
	k := in.ticks.find(n, i+n)
	if k < 0 {
		in.buf = append(in.buf, s[i:i+n]...)
		return i + n
	}
	content := []byte(strings.ReplaceAll(string(s[i+n:k]), "\n", " "))
	if len(content) > 0 && content[0] == ' ' && content[len(content)-1] == ' ' && strings.Trim(string(content), " ") != "" {
		content = content[1 : len(content)-1]
	}
	in.flush()
	in.pieces = append(in.pieces, piece{text: string(content)})
	return k + n
}

// delimiterRun reads a run of '*' or '_' at s[i] and pushes it onto the
// delimiter stack with what it can do (§6.2).
func (in *inline) delimiterRun(i int) int {
	s := in.s
	ch := s[i]
	n := 0
	for i+n < len(s) && s[i+n] == ch {
		n++
	}
	before, after := ' ', ' ' // the start and the end of the text count as whitespace
	if i > 0 {
		before, _ = utf8.DecodeLastRune(s[:i])
	}
	if i+n < len(s) {
		after, _ = utf8.DecodeRune(s[i+n:])
	}
	left := !isUnicodeSpace(after) && (!isPunct(after) || isUnicodeSpace(before) || isPunct(before))
	right := !isUnicodeSpace(before) && (!isPunct(before) || isUnicodeSpace(after) || isPunct(after))
	d := delim{ch: ch, length: n, prev: in.tail, next: -1}
	if ch == '*' {
		d.canOpen, d.canClose = left, right
	} else {
		d.canOpen = left && (!right || isPunct(before))
		d.canClose = right && (!left || isPunct(after))
	}
	in.flush()
	d.piece = len(in.pieces)
	in.pieces = append(in.pieces, piece{ch: ch, n: n})
	in.ds = append(in.ds, d)
	idx := len(in.ds) - 1
	if in.tail >= 0 {
		in.ds[in.tail].next = idx
	} else {
		in.head = idx
	}
	in.tail = idx
	return i + n
}

// isUnicodeSpace is §2.1's Unicode whitespace character.
func isUnicodeSpace(r rune) bool {
	return r == ' ' || r == '\t' || r == '\n' || r == '\f' || r == '\r' || unicode.Is(unicode.Zs, r)
}

// isPunct is §2.1's Unicode punctuation character: general category P or S.
func isPunct(r rune) bool {
	if r < utf8.RuneSelf {
		return isASCIIPunct(byte(r))
	}
	return unicode.In(r, unicode.P, unicode.S)
}

// push puts a link or image opener on the bracket stack.
func (in *inline) push(start int, image bool, text string) {
	in.flush()
	in.brackets = append(in.brackets, opener{piece: len(in.pieces), start: start, image: image, bottom: in.tail})
	in.pieces = append(in.pieces, piece{text: text})
}

// pop removes the top opener.
func (in *inline) pop() {
	in.brackets = in.brackets[:len(in.brackets)-1]
	if in.inactiveBelow > len(in.brackets) {
		in.inactiveBelow = len(in.brackets)
	}
}

// closeBracket reads ']' at s[i] (§6.3, and the "look for link or image"
// step of the appendix): it closes the top opener into a link or an image
// when an inline link or a reference that is defined follows, and is
// literal otherwise.
func (in *inline) closeBracket(i int) int {
	if len(in.brackets) == 0 {
		in.buf = append(in.buf, ']')
		return i + 1
	}
	top := len(in.brackets) - 1
	o := in.brackets[top]
	if !o.image && top < in.inactiveBelow {
		in.pop()
		in.buf = append(in.buf, ']')
		return i + 1
	}
	end, ok := in.inlineLink(i + 1)
	if !ok {
		end, ok = in.reference(o, i)
	}
	if !ok {
		in.pop()
		in.buf = append(in.buf, ']')
		return i + 1
	}
	in.flush()
	in.pieces[o.piece].text = ""
	in.emphasis(o.bottom)
	in.pop()
	if !o.image {
		in.inactiveBelow = len(in.brackets)
	}
	return end
}

// inlineLink reads the "(destination title)" of an inline link at s[i]
// (§6.3), all of it optional but the parentheses.
func (in *inline) inlineLink(i int) (int, bool) {
	s := in.s
	if i >= len(s) || s[i] != '(' {
		return 0, false
	}
	j := skipLinkSpace(s, i+1)
	if j < len(s) && s[j] != ')' {
		dend, ok := scanDestination(s, j)
		if !ok {
			return 0, false
		}
		j = dend
		if t := skipLinkSpace(s, j); t > j && t < len(s) && s[t] != ')' {
			tend, ok := scanTitle(s, t, in.titles)
			if !ok {
				return 0, false
			}
			j = tend
		}
		j = skipLinkSpace(s, j)
	}
	if j < len(s) && s[j] == ')' {
		return j + 1, true
	}
	return 0, false
}

// reference reads what follows the ']' at s[i] of the link text after o
// as a full, collapsed or shortcut reference (§6.3), and reports whether
// its label is defined.
func (in *inline) reference(o opener, i int) (int, bool) {
	s := in.s
	text := s[o.start:i]
	j := i + 1
	switch {
	case j+1 < len(s) && s[j] == '[' && s[j+1] == ']':
		return j + 2, in.textIsLabel(o.start, i) && in.refs[normalizeLabel(text)]
	case j < len(s) && s[j] == '[':
		if lend, label, ok := scanLabel(s, j); ok {
			return lend, in.refs[normalizeLabel(label)]
		}
	}
	return j, in.textIsLabel(o.start, i) && in.refs[normalizeLabel(text)]
}

// textIsLabel reports whether s[start:end], a link's text, is a link label
// (§4.7), as the label of a shortcut or collapsed reference. The first
// unescaped bracket at or after start must be the ']' at end; a text that
// passes holds no bracket, so no two texts it reads overlap.
func (in *inline) textIsLabel(start, end int) bool {
	k := sort.SearchInts(in.bracketAt, start)
	if k == len(in.bracketAt) || in.bracketAt[k] != end {
		return false
	}
	return labelText(in.s[start:end])
}

// bracketOffsets returns the offset of every bracket in s that a backslash
// does not escape, in order.
func bracketOffsets(s []byte) []int {
	var at []int
	for j := 0; j < len(s); j++ {
		switch {
		case s[j] == '\\' && j+1 < len(s) && isASCIIPunct(s[j+1]):
			j++
		case s[j] == '[' || s[j] == ']':
			at = append(at, j)
		}
	}
	return at
}

// emphasis is §6.2's "process emphasis" over the delimiters after bottom,
// which it then removes. openers_bottom is kept by delimiter index, which
// orders the delimiters as the text does, so a bound stays where it was
// set whatever is removed after.
func (in *inline) emphasis(bottom int) {
	var floor [2][2][3]int // by character, by whether the closer can open, by length mod 3
	for a := range floor {
		for b := range floor[a] {
			for c := range floor[a][b] {
				floor[a][b][c] = bottom
			}
		}
	}
	cur := in.head
	if bottom >= 0 {
		cur = in.ds[bottom].next
		if in.ds[bottom].gone {
			cur = in.firstAfter(bottom)
		}
	}
	for cur >= 0 {
		d := &in.ds[cur]
		if !d.canClose {
			cur = d.next
			continue
		}
		ci, oi, m := chIndex(d.ch), boolIndex(d.canOpen), d.length%3
		found := -1
		for o := d.prev; o > floor[ci][oi][m] && o > bottom; o = in.ds[o].prev {
			od := &in.ds[o]
			if od.ch != d.ch || !od.canOpen {
				continue
			}
			if (od.canClose || d.canOpen) && (od.length+d.length)%3 == 0 && !(od.length%3 == 0 && d.length%3 == 0) {
				continue
			}
			found = o
			break
		}
		if found < 0 {
			floor[ci][oi][m] = d.prev
			next := d.next
			if !d.canOpen {
				in.unlink(cur)
			}
			cur = next
			continue
		}
		od := &in.ds[found]
		use := 1
		if in.pieces[od.piece].n >= 2 && in.pieces[d.piece].n >= 2 {
			use = 2
		}
		in.pieces[od.piece].n -= use
		in.pieces[d.piece].n -= use
		// The delimiters between the two are literal now.
		for x := od.next; x != cur; {
			next := in.ds[x].next
			in.unlink(x)
			x = next
		}
		if in.pieces[od.piece].n == 0 {
			in.unlink(found)
		}
		if in.pieces[d.piece].n == 0 {
			next := d.next
			in.unlink(cur)
			cur = next
		}
	}
	for in.tail > bottom {
		in.unlink(in.tail)
	}
}

// firstAfter is the first delimiter on the stack after index i.
func (in *inline) firstAfter(i int) int {
	x := in.tail
	first := -1
	for x > i {
		first = x
		x = in.ds[x].prev
	}
	return first
}

// unlink removes delimiter x from the stack.
func (in *inline) unlink(x int) {
	d := &in.ds[x]
	if d.prev >= 0 {
		in.ds[d.prev].next = d.next
	} else {
		in.head = d.next
	}
	if d.next >= 0 {
		in.ds[d.next].prev = d.prev
	} else {
		in.tail = d.prev
	}
	d.gone = true
}

func chIndex(ch byte) int {
	if ch == '*' {
		return 0
	}
	return 1
}

func boolIndex(b bool) int {
	if b {
		return 1
	}
	return 0
}

// tickIndex finds, for a code span's opening backtick string, the next
// backtick string of the same length, by keeping every backtick string's
// offset by length and a cursor into each that only moves forward, as the
// reading does.
type tickIndex struct {
	at   map[int][]int
	next map[int]int
}

func newTickIndex(s []byte) tickIndex {
	t := tickIndex{at: map[int][]int{}, next: map[int]int{}}
	for i := 0; i < len(s); {
		if s[i] != '`' {
			i++
			continue
		}
		j := i
		for j < len(s) && s[j] == '`' {
			j++
		}
		t.at[j-i] = append(t.at[j-i], i)
		i = j
	}
	return t
}

// find returns the offset of the first backtick string of length n that
// begins at or after from, or -1.
func (t tickIndex) find(n, from int) int {
	list := t.at[n]
	k := t.next[n]
	for k < len(list) && list[k] < from {
		k++
	}
	t.next[n] = k
	if k < len(list) {
		return list[k]
	}
	return -1
}

// entity reads an entity or numeric character reference at s[i], which is
// '&' (§2.5), and returns the characters it stands for.
func entity(s []byte, i int) (string, int, bool) {
	j := i + 1
	if j < len(s) && s[j] == '#' {
		j++
		hex := j < len(s) && (s[j] == 'x' || s[j] == 'X')
		if hex {
			j++
		}
		k := j
		for k < len(s) && (isASCIIDigit(s[k]) || hex && isHexLetter(s[k])) {
			k++
		}
		if k == j || k-j > 7 || hex && k-j > 6 || k >= len(s) || s[k] != ';' {
			return "", 0, false
		}
		base := 10
		if hex {
			base = 16
		}
		cp, err := strconv.ParseUint(string(s[j:k]), base, 32)
		r := rune(cp)
		if err != nil || cp == 0 || !utf8.ValidRune(r) {
			r = utf8.RuneError
		}
		return string(r), k + 1, true
	}
	k := j
	for k < len(s) && k-j < 32 && (isASCIILetter(s[k]) || isASCIIDigit(s[k])) {
		k++
	}
	if k == j || !isASCIILetter(s[j]) || k >= len(s) || s[k] != ';' {
		return "", 0, false
	}
	ref := string(s[i : k+1])
	text := html.UnescapeString(ref)
	// html.UnescapeString also reads a legacy name without its ';' as a
	// prefix, which leaves the rest of the reference, ';' and all.
	if text == ref || (strings.HasSuffix(text, ";") && ref != "&semi;") {
		return "", 0, false
	}
	return text, k + 1, true
}

func isHexLetter(b byte) bool { return b|0x20 >= 'a' && b|0x20 <= 'f' }

// autolink reads an autolink at s[i], which is '<' (§6.5): an absolute URI
// or an email address between angle brackets.
func autolink(s []byte, i int) (int, bool) {
	j := i + 1
	// A scheme: a letter, then 1 to 31 letters, digits, '+', '.' or '-'.
	k := j
	for k < len(s) && k-j < 32 && (isASCIILetter(s[k]) || k > j && (isASCIIDigit(s[k]) || s[k] == '+' || s[k] == '.' || s[k] == '-')) {
		k++
	}
	if k-j >= 2 && k < len(s) && s[k] == ':' {
		for m := k + 1; m < len(s); m++ {
			switch c := s[m]; {
			case c == '>':
				return m + 1, true
			case c <= ' ' || c == '<' || c == 0x7f:
				return 0, false
			}
		}
		return 0, false
	}
	// An email address.
	m := j
	for m < len(s) && (isASCIILetter(s[m]) || isASCIIDigit(s[m]) || strings.IndexByte(".!#$%&'*+/=?^_`{|}~-", s[m]) >= 0) {
		m++
	}
	if m == j || m >= len(s) || s[m] != '@' {
		return 0, false
	}
	m++
	for {
		// A label: 1 to 63 letters, digits and hyphens, not beginning or
		// ending with a hyphen.
		l := m
		for m < len(s) && m-l < 63 && (isASCIILetter(s[m]) || isASCIIDigit(s[m]) || s[m] == '-') {
			m++
		}
		if m == l || s[l] == '-' || s[m-1] == '-' {
			return 0, false
		}
		if m < len(s) && s[m] == '.' {
			m++
			continue
		}
		if m < len(s) && s[m] == '>' {
			return m + 1, true
		}
		return 0, false
	}
}
