// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package finding

import (
	"cmp"
	"crypto/sha256"
	"encoding/binary"
	"slices"
	"strings"
)

// Compare orders two findings by ADR-006 D8 item 1's key, member by
// member, every string by its UTF-8 bytes. An absent member sorts before a
// present one. Severity is not in the key (D8 item 2). Two findings that
// are not duplicates (D8 item 3) can still tie on that key when only their
// subject's title differs, so the title is compared last, which makes the
// order total over every finding a report keeps.
func Compare(a, b Finding) int {
	if c := comparePrefix(a, b); c != 0 {
		return c
	}
	sa, sb := a.schemaOrZero(), b.schemaOrZero()
	return cmp.Or(
		cmpBool(a.schema != nil, b.schema != nil),
		strings.Compare(sa.Entry, sb.Entry),
		strings.Compare(sa.Pointer, sb.Pointer),
		strings.Compare(a.paramsKey, b.paramsKey),
		strings.Compare(a.subject.Title, b.subject.Title),
	)
}

// comparePrefix compares the members of the key before schema: subject,
// location and code. Two duplicates (D8 item 3) are always equal here,
// since the report refuses duplicates at different lines or columns; they
// can differ only in the members after it.
func comparePrefix(a, b Finding) int {
	la, lb := a.locationOrZero(), b.locationOrZero()
	if c := cmp.Or(
		cmp.Compare(a.subject.Kind, b.subject.Kind),
		strings.Compare(a.subject.Name, b.subject.Name),
		cmpBool(a.location != nil, b.location != nil),
		cmp.Compare(la.Origin, lb.Origin),
		strings.Compare(la.File, lb.File),
		cmp.Compare(la.Argument, lb.Argument),
		cmp.Compare(la.Line, lb.Line),
		cmp.Compare(la.Column, lb.Column),
		cmpBool(la.HasPointer, lb.HasPointer),
	); c != 0 {
		return c
	}
	// The pointer, which can be as long as the input, is compared only
	// between findings at the same place: cmp.Or is given every result,
	// so it is not one of them.
	return cmp.Or(comparePaths(la.path, lb.path), strings.Compare(string(a.entry.Code), string(b.entry.Code)))
}

// cmpBool orders false before true: an absent member before a present one.
func cmpBool(a, b bool) int {
	switch {
	case a == b:
		return 0
	case !a:
		return -1
	}
	return 1
}

// compareTokens compares two tokens of JSON Pointers, which comparePaths
// compares token by token (D8 item 1.6). Two tokens that are both array
// indexes (0, or a decimal without a leading zero) compare as numbers, so
// /9 sorts before /10; an index sorts before any other token, which keeps
// the order total; other tokens compare by their decoded UTF-8 bytes.
// Whether a token indexes an array or names a member is not known from
// the pointer alone, so a member named 10 is ordered as an index would be.
func compareTokens(a, b string) int {
	ia, ib := isIndex(a), isIndex(b)
	switch {
	case ia && ib:
		return cmp.Or(cmp.Compare(len(a), len(b)), strings.Compare(a, b))
	case ia:
		return -1
	case ib:
		return 1
	}
	return strings.Compare(a, b)
}

func isIndex(t string) bool {
	if t == "" || (t[0] == '0' && len(t) > 1) {
		return false
	}
	for i := 0; i < len(t); i++ {
		if t[i] < '0' || t[i] > '9' {
			return false
		}
	}
	return true
}

func (f Finding) locationOrZero() Location {
	if f.location == nil {
		return Location{}
	}
	return *f.location
}

func (f Finding) schemaOrZero() Schema {
	if f.schema == nil {
		return Schema{}
	}
	return *f.schema
}

// keywordLocation is the member that picks which of two duplicates is
// kept (D8 item 3); a finding with no schema has none, and sorts first.
func (f Finding) keywordLocation() string {
	if f.schema == nil {
		return ""
	}
	return f.schema.KeywordLocation
}

// canonicalParams is f.paramsKey, the RFC 8785 canonical form of its
// params, which New computes once.
func (f Finding) canonicalParams() string { return f.paramsKey }

// canonicalForm is the RFC 8785 canonical form of params:
// members sorted by name, no whitespace, numbers in canonical form and
// strings escaped as RFC 8785 §3.2.2.2 says. Parameter names are lowercase
// ASCII, so their byte order is RFC 8785's UTF-16 order.
func canonicalForm(params Params) string {
	names := make([]string, 0, len(params))
	for n := range params {
		names = append(names, n)
	}
	slices.Sort(names)
	b := []byte{'{'}
	for i, n := range names {
		if i > 0 {
			b = append(b, ',')
		}
		b = appendCanonicalString(b, n)
		b = append(b, ':')
		b = params[n].appendJSON(b, appendCanonicalString)
	}
	return string(append(b, '}'))
}

// appendCanonicalString appends s as RFC 8785 writes a string: the
// two-character escapes for " \ and the five control characters that have
// one, \u00xx for every other control character below U+0020, and every
// other character as it is.
func appendCanonicalString(b []byte, s string) []byte {
	b = append(b, '"')
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch c {
		case '"', '\\':
			b = append(b, '\\', c)
		case '\b':
			b = append(b, '\\', 'b')
		case '\f':
			b = append(b, '\\', 'f')
		case '\n':
			b = append(b, '\\', 'n')
		case '\r':
			b = append(b, '\\', 'r')
		case '\t':
			b = append(b, '\\', 't')
		default:
			if c < 0x20 {
				b = append(b, '\\', 'u', '0', '0', hexDigits[c>>4], hexDigits[c&0xf])
			} else {
				b = append(b, c)
			}
		}
	}
	return append(b, '"')
}

// duplicateKey is the SHA-256 digest of the members that make two
// findings duplicates (D8 item 3): code, subject, location's origin, file,
// argument and pointer, and params. Each member is written with its
// length, so no two keys share an encoding; the pointer is written as its
// path's digest, which is as unique as its tokens and costs nothing to
// write however long the pointer is.
func (f Finding) duplicateKey() [sha256.Size]byte {
	h := sha256.New()
	var n [8]byte
	str := func(s string) {
		binary.BigEndian.PutUint64(n[:], uint64(len(s)))
		h.Write(n[:])
		h.Write([]byte(s))
	}
	num := func(v int) {
		binary.BigEndian.PutUint64(n[:], uint64(v))
		h.Write(n[:])
	}
	str(string(f.entry.Code))
	num(int(f.subject.Kind))
	str(f.subject.Name)
	str(f.subject.Title)
	if l := f.location; l != nil {
		num(1)
		num(int(l.Origin))
		str(l.File)
		num(l.Argument)
		if l.HasPointer {
			num(1)
			d := l.path.digest()
			h.Write(d[:])
		} else {
			num(0)
		}
	} else {
		num(0)
	}
	str(f.paramsKey)
	var d [sha256.Size]byte
	h.Sum(d[:0])
	return d
}
