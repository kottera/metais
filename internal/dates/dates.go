// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

// Package dates holds the tool's only reading of time: the RFC 3339
// render time `--now` supplies (ADR-008 D7 item 7, ADR-004 D7 item 7), and
// the inputs, format language and arithmetic of the `date` filter (ADR-004
// D11). It is registered as a filter by internal/template through Extend,
// so internal/liquid stays free of it (ADR-010 D2 item 2).
//
// Every value is formatted in the offset it carries. Nothing here reads the
// clock, the host's time zone or a zone database (ADR-010 D11 item 5):
// text is parsed with time.Parse, integers become instants with time.Unix,
// and each instant is held in a fixed offset of its own, so time.Local
// never reaches a result.
package dates

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"metais.org/metais/internal/liquid/value"
)

// Time is an instant together with the offset it was written in, which is
// the offset it is formatted in. Its fraction of a second is the digits
// that were supplied, so no precision is lost to time.Time's nanoseconds.
//
// The zero Time is time.Time's zero instant, 0001-01-01T00:00:00Z.
type Time struct {
	t    time.Time // in a fixed zone of the value's own offset
	frac string    // the supplied fraction's digits, trailing zeros removed
}

// The range of instants a Time holds: every one whose UTC fields, and whose
// fields in its own offset, are within RFC 3339's four-digit years. An
// instant outside it has no RFC 3339 form, so it is refused rather than
// written some other way.
var (
	minUTC = mustParse("0000-01-01T00:00:00Z")
	maxUTC = mustParse("9999-12-31T23:59:59Z")
)

// MinUnix and MaxUnix are the least and greatest integer `date` accepts as
// seconds since 1970-01-01T00:00:00Z: the first second of year 0000 and the
// last of year 9999.
const (
	MinUnix = -62167219200
	MaxUnix = 253402300799
)

func mustParse(s string) time.Time {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		panic(err)
	}
	return t.UTC()
}

// Errors. Each returned error wraps exactly one of them.
var (
	// ErrNow refuses a --now value that is not an RFC 3339 date-time with
	// an offset (ADR-008 D7 item 7), or whose instant in UTC is outside
	// years 0000 to 9999, where D7 item 7's normalized form cannot be
	// written.
	ErrNow = errors.New("not an RFC 3339 date-time with an offset")
	// ErrInput refuses a `date` input ADR-004 D11 item 1 does not accept.
	// The error is an *InputError.
	ErrInput = errors.New("not a date input")
	// ErrFormat refuses a format outside ADR-004 D11 item 3's closed set,
	// or an empty one. The error is a *FormatError.
	ErrFormat = errors.New("not a date format")
)

// ParseNow parses the text --now supplies: an RFC 3339 date-time, its
// offset required. `T` and `Z` may be written in either case (RFC 3339
// §5.6). Every other form, every out-of-range field (second 60 included)
// and every instant whose UTC year is outside 0000 to 9999 is refused with
// an error wrapping ErrNow.
func ParseNow(s string) (Time, error) {
	t, ok := parseDateTime(s)
	if !ok {
		return Time{}, fmt.Errorf("%w: %q", ErrNow, s)
	}
	return t, nil
}

// Normalized returns the text ADR-004 D7 item 7 binds to `now`: the
// instant in UTC as YYYY-MM-DDTHH:MM:SS[.F]Z, where F is the supplied
// fraction with trailing zeros removed and the point is omitted when
// nothing remains.
func (t Time) Normalized() string {
	u := t.utc()
	var b strings.Builder
	year, month, day := u.Date()
	hour, minute, second := u.Clock()
	pad(&b, int64(year), 4, '0')
	b.WriteByte('-')
	pad(&b, int64(month), 2, '0')
	b.WriteByte('-')
	pad(&b, int64(day), 2, '0')
	b.WriteByte('T')
	pad(&b, int64(hour), 2, '0')
	b.WriteByte(':')
	pad(&b, int64(minute), 2, '0')
	b.WriteByte(':')
	pad(&b, int64(second), 2, '0')
	if t.frac != "" {
		b.WriteByte('.')
		b.WriteString(t.frac)
	}
	b.WriteByte('Z')
	return b.String()
}

// Offset returns the value's own offset from UTC, in seconds east.
func (t Time) Offset() int {
	_, off := t.t.Zone()
	return off
}

// Unix returns the instant as seconds since 1970-01-01T00:00:00Z, rounded
// toward negative infinity.
func (t Time) Unix() int64 { return t.t.Unix() }

func (t Time) utc() time.Time { return t.t.UTC() }

// InputError describes a value `date` refuses (ADR-004 D11 item 1).
type InputError struct {
	// Kind is the refused value's kind.
	Kind value.Kind
	// Text is the refused string, when Kind is KindString.
	Text string
	// NamesNow is true for the strings "now" and "today", for which the
	// finding names the `now` global instead.
	NamesNow bool
}

func (e *InputError) Error() string {
	if e.Kind == value.KindString {
		return fmt.Sprintf("%v: the string %q", ErrInput, e.Text)
	}
	return fmt.Sprintf("%v: a value of kind %v", ErrInput, e.Kind)
}

func (e *InputError) Unwrap() error { return ErrInput }

// FromValue returns the instant a `date` input denotes (ADR-004 D11 item
// 1). A string is an RFC 3339 full-date, which is midnight UTC, or a
// date-time with its offset, `T` and `Z` in either case and second 60
// refused. A number ADR-004 D4 item 2 presents as an integer is seconds
// since 1970-01-01T00:00:00Z, in UTC, between MinUnix and MaxUnix. nil
// gives ok false and no error: the filter's result is nil. Every other
// value, "now" and "today" among them, is an *InputError.
func FromValue(v value.Value) (t Time, ok bool, err error) {
	switch v := v.(type) {
	case value.Null:
		return Time{}, false, nil
	case value.String:
		s := v.Text()
		if t, ok := parseFullDate(s); ok {
			return t, true, nil
		}
		if t, ok := parseDateTime(s); ok {
			return t, true, nil
		}
		return Time{}, false, &InputError{Kind: value.KindString, Text: s, NamesNow: s == "now" || s == "today"}
	case value.Number:
		if sec, ok := v.SafeInt(); ok {
			if t, ok := FromUnix(sec); ok {
				return t, true, nil
			}
		}
		return Time{}, false, &InputError{Kind: value.KindNumber}
	}
	return Time{}, false, &InputError{Kind: v.Kind()}
}

// FromUnix returns the instant sec seconds after 1970-01-01T00:00:00Z, in
// UTC. ok is false outside MinUnix to MaxUnix.
func FromUnix(sec int64) (Time, bool) {
	if sec < MinUnix || sec > MaxUnix {
		return Time{}, false
	}
	return Time{t: time.Unix(sec, 0).In(time.UTC)}, true
}

// parseFullDate parses an RFC 3339 full-date as midnight UTC.
func parseFullDate(s string) (Time, bool) {
	if !fullDate(s) {
		return Time{}, false
	}
	return parseDateTime(s + "T00:00:00Z")
}

// parseDateTime parses an RFC 3339 date-time. The grammar is checked here,
// byte by byte, because time.Parse is looser than RFC 3339 when its fast
// path declines: it takes a one-digit hour and a comma before the fraction.
// time.Parse then checks the ranges of every field, second 60 included.
func parseDateTime(s string) (Time, bool) {
	frac, ok := dateTime(s)
	if !ok {
		return Time{}, false
	}
	// time.Parse matches the layout's T and Z exactly; RFC 3339 §5.6 lets
	// either be lower case.
	canon := []byte(s)
	canon[10] = 'T'
	if c := canon[len(canon)-1]; c == 'z' {
		canon[len(canon)-1] = 'Z'
	}
	p, err := time.Parse(time.RFC3339, string(canon))
	if err != nil {
		return Time{}, false
	}
	// Parse gives a time whose offset matches the host's zone that zone's
	// location; the value keeps the offset alone.
	_, off := p.Zone()
	p = p.In(time.FixedZone("", off))
	if u := p.UTC(); u.Before(minUTC) || u.After(maxUTC) {
		return Time{}, false
	}
	return Time{t: p, frac: strings.TrimRight(frac, "0")}, true
}

// fullDate reports whether s is RFC 3339's full-date: four, two and two
// digits separated by "-".
func fullDate(s string) bool {
	return len(s) == 10 && digitsAt(s, 0, 4) && s[4] == '-' && digitsAt(s, 5, 2) && s[7] == '-' && digitsAt(s, 8, 2)
}

// dateTime reports whether s matches RFC 3339 §5.6's date-time,
//
//	full-date "T" HH ":" MM ":" SS [ "." 1*DIGIT ] ( "Z" / ( "+" / "-" ) HH ":" MM )
//
// with "T" and "Z" in either case, and an offset's hour at most 23 and its
// minute at most 59 (time-numoffset). It returns the fraction's digits.
func dateTime(s string) (frac string, ok bool) {
	if len(s) < 20 || !fullDate(s[:10]) || (s[10] != 'T' && s[10] != 't') ||
		!digitsAt(s, 11, 2) || s[13] != ':' || !digitsAt(s, 14, 2) || s[16] != ':' || !digitsAt(s, 17, 2) {
		return "", false
	}
	rest := s[19:]
	if rest[0] == '.' {
		n := 1
		for n < len(rest) && isDigit(rest[n]) {
			n++
		}
		if n == 1 {
			return "", false
		}
		frac, rest = rest[1:n], rest[n:]
	}
	switch {
	case rest == "Z" || rest == "z":
		return frac, true
	case len(rest) == 6 && (rest[0] == '+' || rest[0] == '-') && digitsAt(rest, 1, 2) && rest[3] == ':' && digitsAt(rest, 4, 2):
		return frac, rest[1:3] <= "23" && rest[4:6] <= "59"
	}
	return "", false
}

func digitsAt(s string, i, n int) bool {
	for j := i; j < i+n; j++ {
		if !isDigit(s[j]) {
			return false
		}
	}
	return true
}

func isDigit(c byte) bool { return '0' <= c && c <= '9' }
