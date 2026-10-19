// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package schema

import (
	"errors"
	"slices"
	"strings"
)

// The formats ADR-003 D4 item 1 asserts, each checked by METAIS's own
// validator below and never by the library's built-in one, so that a
// library update cannot change what a format accepts (ADR-010 D5 item 1).
// Any other format is refused before the library sees the schema (D4 item
// 2). Every validator is syntactic: none looks anything up (ADR-009 D9
// item 7).
var formats = map[string]func(string) bool{
	"date-time":             isDateTime,
	"date":                  isDate,
	"time":                  isTime,
	"duration":              isDuration,
	"email":                 isEmail,
	"hostname":              isHostname,
	"ipv4":                  isIPv4,
	"ipv6":                  isIPv6,
	"uri":                   isURI,
	"uri-reference":         isURIReference,
	"uuid":                  isUUID,
	"uri-template":          isURITemplate,
	"json-pointer":          isJSONPointer,
	"relative-json-pointer": isRelativeJSONPointer,
}

// Formats returns the names of the asserted formats, sorted.
func Formats() []string {
	names := make([]string, 0, len(formats))
	for n := range formats {
		names = append(names, n)
	}
	slices.Sort(names)
	return names
}

// errFormat is what a format validator gives the library for a string
// that is not of the format. The adapter never reads it.
var errFormat = errors.New("not of the format")

// formatCheck returns the library's form of the validator: a value that is
// not a string passes (ADR-003 D4 item 3).
func formatCheck(valid func(string) bool) func(any) error {
	return func(v any) error {
		s, ok := v.(string)
		if !ok || valid(s) {
			return nil
		}
		return errFormat
	}
}

func isDigit(c byte) bool { return c >= '0' && c <= '9' }
func isAlpha(c byte) bool { return c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' }
func isHex(c byte) bool   { return isDigit(c) || c >= 'a' && c <= 'f' || c >= 'A' && c <= 'F' }

// digits returns the value of s, which must be all ASCII digits, and
// false otherwise.
func digits(s string) (int, bool) {
	if s == "" {
		return 0, false
	}
	n := 0
	for i := 0; i < len(s); i++ {
		if !isDigit(s[i]) {
			return 0, false
		}
		n = n*10 + int(s[i]-'0')
	}
	return n, true
}

// isDate is RFC 3339 §5.6 full-date: date-fullyear "-" date-month "-"
// date-mday, the day within its month's length in the proleptic Gregorian
// calendar.
func isDate(s string) bool {
	if len(s) != 10 || s[4] != '-' || s[7] != '-' {
		return false
	}
	y, ok1 := digits(s[0:4])
	m, ok2 := digits(s[5:7])
	d, ok3 := digits(s[8:10])
	return ok1 && ok2 && ok3 && m >= 1 && m <= 12 && d >= 1 && d <= daysIn(y, m)
}

func daysIn(y, m int) int {
	switch m {
	case 2:
		if y%4 == 0 && (y%100 != 0 || y%400 == 0) {
			return 29
		}
		return 28
	case 4, 6, 9, 11:
		return 30
	}
	return 31
}

// isTime is RFC 3339 §5.6 full-time: partial-time time-offset. T and Z
// may be lower case (§5.6's note). A leap second, second 60, is valid only
// where the time, moved to UTC by its offset, is 23:59:60 (§5.7).
func isTime(s string) bool {
	if len(s) < 9 || s[2] != ':' || s[5] != ':' {
		return false
	}
	h, ok1 := digits(s[0:2])
	m, ok2 := digits(s[3:5])
	sec, ok3 := digits(s[6:8])
	if !ok1 || !ok2 || !ok3 || h > 23 || m > 59 || sec > 60 {
		return false
	}
	rest := s[8:]
	if rest[0] == '.' {
		i := 1
		for i < len(rest) && isDigit(rest[i]) {
			i++
		}
		if i == 1 {
			return false
		}
		rest = rest[i:]
	}
	var offset int // minutes east of UTC
	switch {
	case rest == "Z" || rest == "z":
	case len(rest) == 6 && (rest[0] == '+' || rest[0] == '-') && rest[3] == ':':
		oh, ok1 := digits(rest[1:3])
		om, ok2 := digits(rest[4:6])
		if !ok1 || !ok2 || oh > 23 || om > 59 {
			return false
		}
		offset = oh*60 + om
		if rest[0] == '-' {
			offset = -offset
		}
	default:
		return false
	}
	if sec == 60 {
		utc := ((h*60+m-offset)%1440 + 1440) % 1440
		return utc == 23*60+59
	}
	return true
}

// isDateTime is RFC 3339 §5.6 date-time: full-date "T" full-time, the T
// in either case.
func isDateTime(s string) bool {
	return len(s) > 11 && (s[10] == 'T' || s[10] == 't') && isDate(s[:10]) && isTime(s[11:])
}

// isDuration is RFC 3339 Appendix A's duration:
//
//	duration = "P" (dur-date / dur-time / dur-week)
//	dur-date = (dur-day / dur-month / dur-year) [dur-time]
//	dur-time = "T" (dur-hour / dur-minute / dur-second)
//
// where each unit is 1*DIGIT and its letter, a year may be followed by a
// month and a month by a day, an hour by a minute and a minute by a
// second, and nothing else may follow.
func isDuration(s string) bool {
	if len(s) < 3 || s[0] != 'P' {
		return false
	}
	s = s[1:]
	if units, ok := durationUnits(s); ok && len(units) == 1 && units[0] == 'W' && s[len(s)-1] == 'W' {
		return true
	}
	date, tm, hasT := strings.Cut(s, "T")
	if !durationPart(date, "YMD", hasT) {
		return false
	}
	if hasT {
		return tm != "" && durationPart(tm, "HMS", false)
	}
	return true
}

// durationUnits splits s into number-letter pairs and returns the letters.
func durationUnits(s string) ([]byte, bool) {
	var units []byte
	for s != "" {
		i := 0
		for i < len(s) && isDigit(s[i]) {
			i++
		}
		if i == 0 || i == len(s) {
			return nil, false
		}
		units = append(units, s[i])
		s = s[i+1:]
	}
	return units, true
}

// durationPart reports whether s is a run of units drawn from order, each
// unit after the one before it with none skipped. An empty s is allowed
// only before a time part.
func durationPart(s, order string, beforeTime bool) bool {
	if s == "" {
		return beforeTime
	}
	units, ok := durationUnits(s)
	if !ok {
		return false
	}
	at := strings.IndexByte(order, units[0])
	if at < 0 {
		return false
	}
	for i, u := range units {
		if at+i >= len(order) || order[at+i] != u {
			return false
		}
	}
	return true
}

// isEmail is RFC 5321 §4.1.2 Mailbox: Local-part "@" (Domain /
// address-literal), the local part a Dot-string or a Quoted-string of at
// most 64 octets, the domain at most 255 (§4.5.3.1). An address literal is
// an IPv4 address or "IPv6:" and an IPv6 address; no General-address-literal
// tag is registered.
func isEmail(s string) bool {
	at := strings.LastIndexByte(s, '@')
	if at < 1 {
		return false
	}
	local, domain := s[:at], s[at+1:]
	if len(local) > 64 || len(domain) > 255 || !(isDotString(local) || isQuotedString(local)) {
		return false
	}
	if strings.HasPrefix(domain, "[") && strings.HasSuffix(domain, "]") {
		lit := domain[1 : len(domain)-1]
		// ABNF's quoted strings are case-insensitive (RFC 5234 §2.3).
		if len(lit) >= 5 && strings.EqualFold(lit[:5], "IPv6:") {
			return isIPv6(lit[5:])
		}
		return isIPv4(lit)
	}
	return isDomain(domain)
}

func isAtext(c byte) bool {
	return isAlpha(c) || isDigit(c) || strings.IndexByte("!#$%&'*+-/=?^_`{|}~", c) >= 0
}

// isDotString is Atom *("." Atom), each Atom 1*atext.
func isDotString(s string) bool {
	for _, atom := range strings.Split(s, ".") {
		if atom == "" {
			return false
		}
		for i := 0; i < len(atom); i++ {
			if !isAtext(atom[i]) {
				return false
			}
		}
	}
	return true
}

// isQuotedString is DQUOTE *QcontentSMTP DQUOTE.
func isQuotedString(s string) bool {
	if len(s) < 2 || s[0] != '"' || s[len(s)-1] != '"' {
		return false
	}
	for i := 1; i < len(s)-1; i++ {
		c := s[i]
		switch {
		case c == '\\':
			i++
			if i >= len(s)-1 || s[i] < 32 || s[i] > 126 {
				return false
			}
		case c == 32 || c == 33 || c >= 35 && c <= 91 || c >= 93 && c <= 126:
		default:
			return false
		}
	}
	return true
}

// isDomain is RFC 5321's Domain: sub-domain *("." sub-domain), each
// Let-dig [Ldh-str].
func isDomain(s string) bool {
	if s == "" {
		return false
	}
	for _, label := range strings.Split(s, ".") {
		if !isLabel(label) {
			return false
		}
	}
	return true
}

// isLabel is a label of letters, digits and hyphens, 1 to 63 octets,
// neither beginning nor ending with a hyphen (RFC 1123 §2.1).
func isLabel(l string) bool {
	if l == "" || len(l) > 63 || l[0] == '-' || l[len(l)-1] == '-' {
		return false
	}
	for i := 0; i < len(l); i++ {
		if !isAlpha(l[i]) && !isDigit(l[i]) && l[i] != '-' {
			return false
		}
	}
	return true
}

// isHostname is RFC 1123 §2.1: labels separated by dots, at most 253
// octets in all. A trailing dot, the root, is not part of a host name.
func isHostname(s string) bool {
	return len(s) <= 253 && isDomain(s)
}

// isIPv4 is the dotted-decimal form of RFC 2673 §3.2: four decimal octets
// of 0 to 255 with no leading zero.
func isIPv4(s string) bool {
	parts := strings.Split(s, ".")
	if len(parts) != 4 {
		return false
	}
	for _, p := range parts {
		n, ok := digits(p)
		if !ok || len(p) > 3 || n > 255 || len(p) > 1 && p[0] == '0' {
			return false
		}
	}
	return true
}

// isIPv6 is RFC 4291 §2.2's text forms: eight groups of one to four hex
// digits, at most one "::" standing for one or more zero groups, and the
// last two groups optionally written as an IPv4 address. No zone is
// allowed.
func isIPv6(s string) bool {
	groups := 0
	head, tail, compressed := strings.Cut(s, "::")
	if compressed && strings.Contains(tail, "::") {
		return false
	}
	count := func(part string, last bool) bool {
		if part == "" {
			return true
		}
		fields := strings.Split(part, ":")
		for i, f := range fields {
			if last && i == len(fields)-1 && strings.Contains(f, ".") {
				if !isIPv4(f) {
					return false
				}
				groups += 2
				continue
			}
			if f == "" || len(f) > 4 {
				return false
			}
			for j := 0; j < len(f); j++ {
				if !isHex(f[j]) {
					return false
				}
			}
			groups++
		}
		return true
	}
	if !compressed {
		return count(s, true) && groups == 8
	}
	return count(head, false) && count(tail, true) && groups <= 7
}

// isUUID is RFC 4122 §3's string form: 8-4-4-4-12 hexadecimal digits,
// of either case.
func isUUID(s string) bool {
	if len(s) != 36 {
		return false
	}
	for i := 0; i < len(s); i++ {
		switch i {
		case 8, 13, 18, 23:
			if s[i] != '-' {
				return false
			}
		default:
			if !isHex(s[i]) {
				return false
			}
		}
	}
	return true
}

// isJSONPointer is RFC 6901 §3: empty, or "/" and a reference token, any
// number of times, with "~" only as "~0" or "~1".
func isJSONPointer(s string) bool {
	if s != "" && s[0] != '/' {
		return false
	}
	for i := 0; i < len(s); i++ {
		if s[i] == '~' && (i+1 == len(s) || s[i+1] != '0' && s[i+1] != '1') {
			return false
		}
	}
	return true
}

// isRelativeJSONPointer is draft-handrews-relative-json-pointer-01 §3, the
// draft 2020-12 cites: a non-negative integer without a leading zero,
// followed by "#" or a JSON Pointer.
func isRelativeJSONPointer(s string) bool {
	i := 0
	for i < len(s) && isDigit(s[i]) {
		i++
	}
	if i == 0 || i > 1 && s[0] == '0' {
		return false
	}
	return s[i:] == "#" || isJSONPointer(s[i:])
}
