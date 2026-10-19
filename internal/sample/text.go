// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package sample

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/kottera/metais/internal/liquid/value"
)

// Corpus is the placeholder text every generated word comes from (ADR-007
// D9 item 19): the standard "Lorem ipsum" passage, fixed in the binary so
// that placeholder text is recognised as placeholder and replaced.
const Corpus = "Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do " +
	"eiusmod tempor incididunt ut labore et dolore magna aliqua. Ut enim ad " +
	"minim veniam, quis nostrud exercitation ullamco laboris nisi ut aliquip " +
	"ex ea commodo consequat. Duis aute irure dolor in reprehenderit in " +
	"voluptate velit esse cillum dolore eu fugiat nulla pariatur. Excepteur " +
	"sint occaecat cupidatat non proident, sunt in culpa qui officia " +
	"deserunt mollit anim id est laborum."

// words is the corpus split into words, punctuation removed, lowercased.
var words = func() []string {
	var out []string
	for _, w := range strings.Fields(Corpus) {
		out = append(out, strings.ToLower(strings.Trim(w, ",.")))
	}
	return out
}()

// wordStream yields corpus words in order from a start its stream draws,
// wrapping at the end (item 19).
type wordStream struct {
	i int
}

func newWords(st *SplitMix64) *wordStream {
	return &wordStream{i: int(st.Below(uint64(len(words))))}
}

func (w *wordStream) next() string {
	word := words[w.i%len(words)]
	w.i++
	return word
}

// capital is word with its first letter in upper case; every corpus word
// is ASCII.
func capital(word string) string {
	if word == "" {
		return word
	}
	return strings.ToUpper(word[:1]) + word[1:]
}

// length is s's length in code points, which length bounds count (item
// 18).
func length(s string) int64 { return int64(utf8.RuneCountInString(s)) }

// lengths are a string's bounds: min, and max when hasMax.
type lengths struct {
	min, max int64
	hasMax   bool
}

func (l lengths) fits(s string) bool {
	n := length(s)
	return n >= l.min && (!l.hasMax || n <= l.max)
}

// clamp is target within the bounds.
func (l lengths) clamp(target int64) int64 {
	target = max(target, l.min)
	if l.hasMax {
		target = min(target, l.max)
	}
	return target
}

// text makes a string from C (item 18): from its format, else from its
// pattern, else prose where the string is prose, else plain text.
func (s *Generator) text(loc *place, c *conj) (value.Value, bool) {
	var l lengths
	l.min, _ = c.count("minLength", true)
	l.max, l.hasMax = c.count("maxLength", false)
	if l.hasMax && l.min > l.max {
		s.unsatisfiable(loc, s.holder(c, "minLength"), ReasonLength)
		return nil, false
	}
	st := s.stream(loc.ptr, loc.valuePurpose())
	patterns := c.strings("pattern")
	// keep reports whether str meets the length bounds and every pattern,
	// and reads back.
	keep := func(str string) bool {
		if !l.fits(str) {
			return false
		}
		for _, p := range patterns {
			if !s.matches(p, str) {
				return false
			}
		}
		return s.writable(loc, mustString(str))
	}
	if formats := c.strings("format"); len(formats) > 0 {
		str := formatText(formats[0], st)
		if str == "" || !keep(str) || !s.valid(c.starts, mustString(str)) {
			s.unsatisfiable(loc, s.holder(c, "format"), ReasonFormat)
			return nil, false
		}
		return mustString(str), true
	}
	if len(patterns) > 0 {
		for k := range maxPatternAttempts {
			str, ok := s.patternText(patterns[0], st, k, l.min)
			if !ok {
				s.unsatisfiable(loc, s.holder(c, "pattern"), ReasonPatternUnsupported)
				return nil, false
			}
			if keep(str) {
				return mustString(str), true
			}
		}
		s.unsatisfiable(loc, s.holder(c, "pattern"), ReasonPatternUnmet)
		return nil, false
	}
	kind := loc.prose
	for _, mt := range c.strings("contentMediaType") {
		switch mt {
		case "text/markdown":
			kind = proseMarkdown
		case "text/plain":
			if kind == proseNone {
				kind = proseText
			}
		}
	}
	var str string
	if kind == proseNone {
		str = plainText(newWords(st), st, l)
	} else {
		str = proseOf(newWords(st), st, l, kind == proseMarkdown)
	}
	if !keep(str) {
		s.unsatisfiable(loc, s.holder(c, "minLength"), ReasonLength)
		return nil, false
	}
	return mustString(str), true
}

// plainText is corpus words to a target length of 24 code points within
// the bounds (item 18): under 40, words joined by spaces, the first
// capitalised and no punctuation; from 40, sentences.
func plainText(w *wordStream, st *SplitMix64, l lengths) string {
	target := l.clamp(24)
	if target >= 40 {
		var b strings.Builder
		for length(b.String()) < target {
			if b.Len() > 0 {
				b.WriteByte(' ')
			}
			b.WriteString(sentence(w, st))
		}
		return fit(b.String(), w, l)
	}
	var b strings.Builder
	for length(b.String()) < target {
		word := w.next()
		if b.Len() == 0 {
			word = capital(word)
		} else {
			b.WriteByte(' ')
		}
		b.WriteString(word)
	}
	return fit(b.String(), w, l)
}

// sentence is 6 to 12 words, the first capitalised, ending in a full stop.
func sentence(w *wordStream, st *SplitMix64) string {
	n := 6 + int(st.Below(7))
	parts := make([]string, n)
	for i := range parts {
		parts[i] = w.next()
	}
	parts[0] = capital(parts[0])
	return strings.Join(parts, " ") + "."
}

// fit cuts text at the last word boundary that fits the maximum, mid-word
// only when one word is too long, and extends it by words to reach the
// minimum (item 18).
func fit(text string, w *wordStream, l lengths) string {
	if l.hasMax && length(text) > l.max {
		text = cut(text, l.max)
	}
	for length(text) < l.min {
		if text != "" {
			text += " "
		}
		text += w.next()
	}
	if l.hasMax && length(text) > l.max {
		text = cutRunes(text, l.max)
	}
	return text
}

// cut is text cut at its last word boundary within n code points, or at n
// code points when its first word alone is longer.
func cut(text string, n int64) string {
	head := cutRunes(text, n)
	if len(head) < len(text) && text[len(head)] != ' ' {
		if i := strings.LastIndexAny(head, " \n"); i > 0 {
			head = head[:i]
		}
	}
	return strings.TrimRight(head, " \n")
}

// cutRunes is the first n code points of text.
func cutRunes(text string, n int64) string {
	i := int64(0)
	for off := range text {
		if i == n {
			return text[:off]
		}
		i++
	}
	return text
}

// proseOf is prose for a prose field (item 18): a target of 400 code
// points within the bounds, as paragraphs of three to five sentences
// separated by a blank line. Markdown adds a level-one heading of three
// title-cased words before them and a three-item bullet list after, when
// the bounds leave room.
func proseOf(w *wordStream, st *SplitMix64, l lengths, markdown bool) string {
	target := l.clamp(400)
	var heading, list string
	if markdown {
		heading = "# " + capital(w.next()) + " " + capital(w.next()) + " " + capital(w.next())
		var items []string
		for range 3 {
			items = append(items, "- "+capital(w.next())+" "+w.next()+" "+w.next())
		}
		list = strings.Join(items, "\n")
	}
	var paras []string
	total := int64(0)
	for total < target {
		n := 3 + int(st.Below(3))
		ss := make([]string, n)
		for i := range ss {
			ss[i] = sentence(w, st)
		}
		p := strings.Join(ss, " ")
		paras = append(paras, p)
		total += length(p) + 2
	}
	body := strings.Join(paras, "\n\n")
	if markdown {
		whole := heading + "\n\n" + body + "\n\n" + list
		if !l.hasMax || length(whole) <= l.max {
			return fit(whole, w, l)
		}
	}
	return fit(body, w, l)
}

// formatText is a value of the format name (item 18), from fixed,
// reserved or placeholder sources, never from the host; "" for a format
// ADR-003 D4 does not assert, which the compiler has already refused.
func formatText(name string, st *SplitMix64) string {
	w := newWords(st)
	word := func() string { return w.next() }
	switch name {
	case "date":
		return date(st)
	case "date-time":
		return date(st) + "T" + clock(st) + "Z"
	case "time":
		return clock(st) + "Z"
	case "duration":
		d, h := 1+st.Below(30), 1+st.Below(23)
		switch st.Below(3) {
		case 0:
			return fmt.Sprintf("P%dD", d)
		case 1:
			return fmt.Sprintf("PT%dH", h)
		}
		return fmt.Sprintf("P%dDT%dH", d, h)
	case "email":
		return word() + "." + word() + "@example.com"
	case "hostname":
		return word() + ".example.com"
	case "ipv4":
		blocks := []string{"192.0.2", "198.51.100", "203.0.113"}
		return fmt.Sprintf("%s.%d", blocks[st.Below(3)], 1+st.Below(254))
	case "ipv6":
		var b strings.Builder
		b.WriteString("2001:db8")
		for range 6 {
			fmt.Fprintf(&b, ":%x", st.Below(1<<16))
		}
		return b.String()
	case "uri":
		return "https://example.com/" + word() + "/" + word()
	case "uri-reference":
		return "/" + word() + "/" + word()
	case "uri-template":
		return "https://example.com/{" + word() + "}"
	case "uuid":
		var b [16]byte
		hi, lo := st.Next(), st.Next()
		for i := range 8 {
			b[i] = byte(hi >> (56 - 8*i))
			b[8+i] = byte(lo >> (56 - 8*i))
		}
		b[6] = b[6]&0x0f | 0x40 // version 4
		b[8] = b[8]&0x3f | 0x80 // the RFC 9562 variant
		return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
	case "json-pointer":
		return "/" + word() + "/" + fmt.Sprint(st.Below(10))
	case "relative-json-pointer":
		return "0/" + word()
	}
	return ""
}

// date is a date uniform in 2020-01-01 … 2029-12-31, written
// YYYY-MM-DD: the decade's 3653 days, counted through from its first.
func date(st *SplitMix64) string {
	day := int(st.Below(3653))
	for y := 2020; ; y++ {
		for m := 1; m <= 12; m++ {
			n := monthDays[m-1]
			if m == 2 && y%4 == 0 {
				// Every year of the decade divisible by 4 is a leap year.
				n++
			}
			if day < n {
				return fmt.Sprintf("%04d-%02d-%02d", y, m, day+1)
			}
			day -= n
		}
	}
}

var monthDays = [12]int{31, 28, 31, 30, 31, 30, 31, 31, 30, 31, 30, 31}

// clock is a time of day, HH:MM:SS.
func clock(st *SplitMix64) string {
	sec := st.Below(86400)
	return fmt.Sprintf("%02d:%02d:%02d", sec/3600, sec/60%60, sec%60)
}
