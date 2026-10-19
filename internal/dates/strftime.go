// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package dates

import (
	"fmt"
	"strings"
)

// Format is a compiled `date` format: Ruby strftime's language, closed to
// ADR-004 D11 item 3's set. It is METAIS's own formatter, never Go's
// layouts, and it formats a Time in the Time's own offset.
type Format struct {
	items []item
}

// item is one piece of a format: literal text, or a directive.
type item struct {
	lit   string
	conv  byte // 0 for literal text; ':' for %:z
	width int  // %3N, %6N and %9N; 0 otherwise
	pad   byte // 0 for the directive's own padding, '-' for none, else the padding byte
	upper bool // the ^ flag
}

// FormatError describes a format `date` refuses (ADR-004 D11 item 3).
type FormatError struct {
	// Format is the refused format.
	Format string
	// Directive is the first directive outside the closed set, from its
	// "%" to the byte that put it outside; empty for an empty format.
	Directive string
}

func (e *FormatError) Error() string {
	if e.Format == "" {
		return fmt.Sprintf("%v: the format is empty", ErrFormat)
	}
	return fmt.Sprintf("%v: %q has the directive %q", ErrFormat, e.Format, e.Directive)
}

func (e *FormatError) Unwrap() error { return ErrFormat }

// conversions is ADR-004 D11 item 3's closed set of conversion characters.
// N is the one that takes a width, and only 3, 6 or 9; ':' is %:z's.
const conversions = "aAbhBcCdDeFGHIjklLmMnNpPrRsStTuUVwWxXyYzZ%"

// ParseFormat compiles f. A directive is "%", then any of the flags "-"
// (no padding), "_" (space padding), "0" (zero padding) and "^" (upper
// case), each at most once and at most one of the three padding flags,
// then a conversion of the closed set: %3N, %6N and %9N are the only widths,
// and %:z the only colon. Any other directive, or an empty format, is a
// *FormatError.
func ParseFormat(f string) (Format, error) {
	if f == "" {
		return Format{}, &FormatError{}
	}
	var items []item
	lit := 0
	for i := 0; i < len(f); {
		if f[i] != '%' {
			i++
			continue
		}
		if lit < i {
			items = append(items, item{lit: f[lit:i]})
		}
		it, n, ok := directive(f[i:])
		if !ok {
			return Format{}, &FormatError{Format: f, Directive: f[i : i+n]}
		}
		items = append(items, it)
		i += n
		lit = i
	}
	if lit < len(f) {
		items = append(items, item{lit: f[lit:]})
	}
	return Format{items: items}, nil
}

// directive reads the directive at the start of s, which begins with "%".
// It returns the bytes it read: on a refusal, through the one that refused
// it.
func directive(s string) (it item, n int, ok bool) {
	n = 1
	for ; n < len(s); n++ {
		switch c := s[n]; c {
		case '^':
			if it.upper {
				return it, n + 1, false
			}
			it.upper = true
			continue
		case '-', '_', '0':
			if it.pad != 0 {
				return it, n + 1, false
			}
			it.pad = c
			if c == '_' {
				it.pad = ' '
			}
			continue
		}
		break
	}
	if n == len(s) {
		return it, n, false
	}
	switch c := s[n]; {
	case c == '3' || c == '6' || c == '9':
		if n+1 < len(s) && s[n+1] == 'N' {
			it.conv, it.width = 'N', int(c-'0')
			return it, n + 2, true
		}
		return it, min(n+2, len(s)), false
	case c == ':':
		if n+1 < len(s) && s[n+1] == 'z' {
			it.conv = ':'
			return it, n + 2, true
		}
		return it, min(n+2, len(s)), false
	case strings.IndexByte(conversions, c) >= 0:
		it.conv = c
		return it, n + 1, true
	}
	return it, n + 1, false
}

// Composite directives, as ADR-004 D11 item 3 defines them.
var compositeText = map[byte]string{
	'c': "%a %b %e %H:%M:%S %Y",
	'D': "%m/%d/%y",
	'F': "%Y-%m-%d",
	'r': "%I:%M:%S %p",
	'R': "%H:%M",
	'T': "%H:%M:%S",
	'x': "%m/%d/%y",
	'X': "%H:%M:%S",
}

// composites are compositeText compiled.
var composites = func() map[byte]Format {
	m := make(map[byte]Format, len(compositeText))
	for c, text := range compositeText {
		f, err := ParseFormat(text)
		if err != nil {
			panic(err)
		}
		m[c] = f
	}
	return m
}()

var (
	weekdays = [...]string{"Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday"}
	months   = [...]string{"January", "February", "March", "April", "May", "June", "July", "August", "September", "October", "November", "December"}
)

// Apply formats t in its own offset.
func (f Format) Apply(t Time) string {
	var b strings.Builder
	for _, it := range f.items {
		if it.conv == 0 {
			b.WriteString(it.lit)
			continue
		}
		s := it.format(t)
		if it.upper {
			s = strings.ToUpper(s)
		}
		b.WriteString(s)
	}
	return b.String()
}

// FormatString compiles format and applies it to t.
func FormatString(t Time, format string) (string, error) {
	f, err := ParseFormat(format)
	if err != nil {
		return "", err
	}
	return f.Apply(t), nil
}

// format writes one directive, before the ^ flag.
func (it item) format(t Time) string {
	if f, ok := composites[it.conv]; ok {
		return f.Apply(t)
	}
	lt := t.t
	year, month, day := lt.Date()
	hour, minute, second := lt.Clock()
	yday := lt.YearDay() - 1
	wday := int(lt.Weekday())
	switch it.conv {
	case 'a':
		return weekdays[wday][:3]
	case 'A':
		return weekdays[wday]
	case 'b', 'h':
		return months[month-1][:3]
	case 'B':
		return months[month-1]
	case 'p':
		if hour < 12 {
			return "AM"
		}
		return "PM"
	case 'P':
		if hour < 12 {
			return "am"
		}
		return "pm"
	case 'n':
		return "\n"
	case 't':
		return "\t"
	case '%':
		return "%"
	case 'z':
		return offset(t.Offset(), "")
	case ':':
		return offset(t.Offset(), ":")
	case 'Z':
		if t.Offset() == 0 {
			return "UTC"
		}
		return offset(t.Offset(), ":")
	case 'L':
		return fraction(t.frac, 3)
	case 'N':
		w := it.width
		if w == 0 {
			w = 9
		}
		return fraction(t.frac, w)
	case 's':
		return number(t.Unix(), 1, '0', 0)
	case 'u':
		return number(int64((wday+6)%7+1), 1, '0', 0)
	case 'w':
		return number(int64(wday), 1, '0', 0)
	}
	var v int64
	width, padding := 2, byte('0')
	switch it.conv {
	case 'C':
		v = int64(floorDiv(year, 100))
	case 'd':
		v = int64(day)
	case 'e':
		v, padding = int64(day), ' '
	case 'G':
		y, _ := lt.ISOWeek()
		v, width = int64(y), 4
	case 'H':
		v = int64(hour)
	case 'I':
		v = int64(hour12(hour))
	case 'j':
		v, width = int64(yday+1), 3
	case 'k':
		v, padding = int64(hour), ' '
	case 'l':
		v, padding = int64(hour12(hour)), ' '
	case 'm':
		v = int64(month)
	case 'M':
		v = int64(minute)
	case 'S':
		v = int64(second)
	case 'U':
		v = int64((yday + 7 - wday) / 7)
	case 'V':
		_, w := lt.ISOWeek()
		v = int64(w)
	case 'W':
		v = int64((yday + 7 - (wday+6)%7) / 7)
	case 'y':
		v = int64(year - floorDiv(year, 100)*100)
	case 'Y':
		v, width = int64(year), 4
	default:
		panic(fmt.Sprintf("dates: conversion %q compiled but not formatted", it.conv))
	}
	return number(v, width, padding, it.pad)
}

func hour12(h int) int {
	if h%12 == 0 {
		return 12
	}
	return h % 12
}

func floorDiv(a, b int) int {
	q := a / b
	if a%b != 0 && a < 0 {
		q--
	}
	return q
}

// number writes v at width with padding, unless flag overrides it: '-'
// writes no padding, and ' ' or '0' pads with that byte instead. A minus
// sign comes before the padding and is not counted in the width.
func number(v int64, width int, padding, flag byte) string {
	switch flag {
	case '-':
		width = 1
	case ' ', '0':
		padding = flag
	}
	var b strings.Builder
	neg := v < 0
	if neg {
		v = -v
		if padding == '0' {
			b.WriteByte('-')
		}
	}
	digits := fmt.Sprint(v)
	if pad := width - len(digits); pad > 0 {
		b.WriteString(strings.Repeat(string(padding), pad))
	}
	if neg && padding != '0' {
		b.WriteByte('-')
	}
	b.WriteString(digits)
	return b.String()
}

// pad writes v at width with padding, for Normalized.
func pad(b *strings.Builder, v int64, width int, padding byte) {
	b.WriteString(number(v, width, padding, 0))
}

// offset writes off seconds east of UTC as +hhmm, or +hh:mm with sep ":".
// Offsets are whole minutes, as RFC 3339 writes them.
func offset(off int, sep string) string {
	sign := byte('+')
	if off < 0 {
		sign, off = '-', -off
	}
	m := off / 60
	return string(sign) + number(int64(m/60), 2, '0', 0) + sep + number(int64(m%60), 2, '0', 0)
}

// fraction writes the first n digits of the supplied fraction, right-padded
// with zeros: Ruby's %N truncates, it never rounds.
func fraction(frac string, n int) string {
	if len(frac) >= n {
		return frac[:n]
	}
	return frac + strings.Repeat("0", n-len(frac))
}
