// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package filters

import (
	"unicode"
	"unicode/utf8"

	"metais.org/metais/internal/liquid"
	"metais.org/metais/internal/liquid/value"
	"metais.org/metais/internal/unitext"
)

// slugFilter is ADR-004 D10 item 6: a string as a URL-safe slug, in
// [a-z0-9]+(-[a-z0-9]+)*. nil is nil; any other input but a string is
// CodeFilterInput; an empty result is CodeSlugEmpty, since no slug can be
// made of the text and the author must supply one as data.
var slugFilter = &filter{
	name: "slug",
	apply: func(c *call, in value.Value) (value.Value, *liquid.Error) {
		switch v := in.(type) {
		case value.Null:
			if err := c.units(1); err != nil {
				return nil, err
			}
			return value.Null{}, nil
		case value.String:
			var sp slugSpeller
			n := 0
			slug(v.Text(), &sp, func(byte) { n++ })
			if n == 0 {
				return nil, liquid.FilterError(c.name, liquid.CodeSlugEmpty, nil, "no letter or digit of the text has an ASCII form")
			}
			return c.text(int64(n), func() string {
				b := make([]byte, 0, n)
				slug(v.Text(), &sp, func(c byte) { b = append(b, c) })
				return string(b)
			})
		}
		return nil, inputErr(c.name, in, "slug takes a string")
	},
}

// slugLetters is step 3 of ADR-004 D10 item 6 (Amendment A1): the Latin
// letters that have no decomposition, each with its ASCII spelling. It is
// read after NFKD, so a letter that decomposes to one of them, as U+01FE
// does to U+00D8 and a combining acute, is spelled as its base is.
var slugLetters = map[rune]string{
	'ß': "ss", 'ẞ': "ss",
	'æ': "ae", 'Æ': "ae",
	'œ': "oe", 'Œ': "oe",
	'ø': "o", 'Ø': "o",
	'đ': "d", 'Đ': "d", 'ð': "d", 'Ð': "d",
	'þ': "th", 'Þ': "th",
	'ł': "l", 'Ł': "l",
	'ı': "i",
	'ħ': "h", 'Ħ': "h",
	'ŧ': "t", 'Ŧ': "t",
	'ŋ': "ng", 'Ŋ': "ng",
	'ĸ': "k",
}

// slug applies ADR-004 D10 item 6's steps (Amendment A1) to s and hands
// each byte of the slug to emit, in order, building nothing itself: NFKD,
// every Mn removed, the table of letters, ASCII A–Z lower-cased, each
// maximal run of anything but a–z and 0–9 replaced by one "-", and a
// leading or trailing "-" removed. It decomposes one code point at a time,
// so it holds no copy of s however far NFKD expands it. That gives the
// slug of the whole string's NFKD: normalizing a string decomposes each
// code point and then reorders only runs of combining marks, which are
// never a–z, 0–9 or a letter of the table and so end as part of a run of
// "-" wherever they stand.
func slug(s string, sp *slugSpeller, emit func(byte)) {
	started, dash := false, false
	put := func(b byte) {
		if 'a' <= b && b <= 'z' || '0' <= b && b <= '9' {
			if dash && started {
				emit('-')
			}
			started, dash = true, false
			emit(b)
			return
		}
		dash = true
	}
	for _, r := range s {
		if r < utf8.RuneSelf {
			put(asciiSpelling(byte(r)))
			continue
		}
		for _, b := range []byte(sp.spelling(r)) {
			put(b)
		}
	}
}

// A slugSpeller gives a code point's spelling: the a–z, 0–9 and "-" that
// its NFKD, its Mn removed and the table of letters applied, puts in the
// slug, a "-" standing for anything else. A code point with no
// decomposition is spelled as itself, with no NFKD: a Hangul syllable,
// which NFKD decomposes by rule, is "-" as its jamo are. One with a
// decomposition is decomposed the first time it is met and remembered, so
// NFKD's expansion, up to 18 code points from one, is paid once a call
// for each code point, however often the text repeats it.
type slugSpeller struct {
	d     unitext.Decomposer
	one   [utf8.UTFMax]byte
	nfkd  []byte
	known map[rune]string
}

func (sp *slugSpeller) spelling(r rune) string {
	b := utf8.AppendRune(sp.one[:0], r)
	if !unitext.Decomposes(b) {
		return letterSpelling(r)
	}
	if s, ok := sp.known[r]; ok {
		return s
	}
	sp.nfkd = sp.d.AppendNFKD(sp.nfkd[:0], b)
	var out []byte
	for _, d := range string(sp.nfkd) {
		out = append(out, letterSpelling(d)...)
	}
	if sp.known == nil {
		sp.known = map[rune]string{}
	}
	s := string(out)
	sp.known[r] = s
	return s
}

// letterSpelling is one code point of an NFKD form spelled: nothing for
// an Mn, the table's spelling for a letter of it, an ASCII letter or digit
// lower-cased, and "-" for anything else.
func letterSpelling(d rune) string {
	if d < utf8.RuneSelf {
		return asciiSpellings[d]
	}
	if unicode.Is(unicode.Mn, d) {
		return ""
	}
	if ascii, ok := slugLetters[d]; ok {
		return ascii
	}
	return "-"
}

// asciiSpellings spells each ASCII byte as letterSpelling does.
var asciiSpellings = func() (t [utf8.RuneSelf]string) {
	for b := range t {
		t[b] = string(asciiSpelling(byte(b)))
	}
	return t
}()

// asciiSpelling is an ASCII byte spelled: a–z and 0–9 as themselves, A–Z
// lower-cased, and "-" for anything else.
func asciiSpelling(b byte) byte {
	switch {
	case 'A' <= b && b <= 'Z':
		return b + 'a' - 'A'
	case 'a' <= b && b <= 'z', '0' <= b && b <= '9':
		return b
	}
	return '-'
}
