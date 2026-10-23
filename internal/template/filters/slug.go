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
			n := 0
			slug(v.Text(), func(byte) { n++ })
			if n == 0 {
				return nil, liquid.FilterError(c.name, liquid.CodeSlugEmpty, nil, "no letter or digit of the text has an ASCII form")
			}
			return c.text(int64(n), func() string {
				b := make([]byte, 0, n)
				slug(v.Text(), func(c byte) { b = append(b, c) })
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
func slug(s string, emit func(byte)) {
	started, dash := false, false
	put := func(r rune) {
		if 'A' <= r && r <= 'Z' {
			r += 'a' - 'A'
		}
		if ('a' <= r && r <= 'z') || ('0' <= r && r <= '9') {
			if dash && started {
				emit('-')
			}
			started, dash = true, false
			emit(byte(r))
			return
		}
		dash = true
	}
	letter := func(d rune) {
		if unicode.Is(unicode.Mn, d) {
			return
		}
		if ascii, ok := slugLetters[d]; ok {
			for _, a := range ascii {
				put(a)
			}
			return
		}
		put(d)
	}
	for _, r := range s {
		if r < utf8.RuneSelf {
			put(r)
			continue
		}
		for _, d := range unitext.NFKD(string(r)) {
			letter(d)
		}
	}
}
