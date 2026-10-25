// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

// Package unitext is the tool's Unicode: the NFC check, the canonical
// caseless match with full case folding, and the general categories a
// portable path may use (ADR-010 D11 item 7), all at the Unicode version
// specification 1.0 pins. It also holds the one portable-path predicate of
// ADR-005 D12, which ADR-001 D2 applies to entry names (RULINGS R1-023):
// CheckPath and Collisions are the only implementation of that rule in the
// module.
//
// The tables are the toolchain's: the standard library's unicode package
// and golang.org/x/text's, which the compiling toolchain selects (ADR-010 D1
// item 4). version_test.go fails when either moves, so a toolchain upgrade
// cannot widen the accepted names silently.
package unitext

import (
	"unicode"
	"unicode/utf8"

	"golang.org/x/text/cases"
	"golang.org/x/text/unicode/norm"
)

// Version is the Unicode version specification 1.0 pins (ADR-001 D2 item 3,
// ADR-005 D12 item 2).
const Version = "15.0.0"

// IsNFC reports whether s is valid UTF-8 in Unicode Normalization Form C.
// A string that is not is refused, never normalized (ADR-005 D12 item 1).
func IsNFC(s string) bool {
	return utf8.ValidString(s) && norm.NFC.IsNormalString(s)
}

// CaselessKey returns the key under which two strings are canonical
// caseless matches (Unicode D145): NFD(toCasefold(NFD(s))), with full case
// folding, statuses C and F, and no tailoring. Two strings match exactly
// when their keys are equal. s must be valid UTF-8.
func CaselessKey(s string) string {
	return norm.NFD.String(cases.Fold().String(norm.NFD.String(s)))
}

// Fold returns s under full Unicode case folding, statuses C and F, with
// no normalization: the fold CommonMark matches link labels by (ADR-010
// D9, A2).
func Fold(s string) string { return cases.Fold().String(s) }

// IsPathChar reports whether r is U+0020 or a code point of the general
// categories L, M, N, P or S. Every category table holds only assigned code
// points, so an unassigned code point or a noncharacter (Cn), a control
// (Cc), a format character (Cf), a private-use character (Co) and every
// separator but U+0020 (Z) are not path characters (ADR-005 D12 item 2).
func IsPathChar(r rune) bool {
	return r == ' ' || unicode.In(r, unicode.L, unicode.M, unicode.N, unicode.P, unicode.S)
}

// NFC returns s in Unicode Normalization Form C. It is for a name that
// arrives from the host, such as a collection item's file name, which
// ADR-003 D15 item 5 normalizes because one name reaches the tool as
// different bytes from different file systems. A name inside a package is
// refused when it is not NFC, never normalized (ADR-005 D12 item 1). s must
// be valid UTF-8.
func NFC(s string) string { return norm.NFC.String(s) }

// NFKD returns s in Unicode Normalization Form KD, the decomposition the
// slug filter folds a string to ASCII through (ADR-004 D10 item 6). s must
// be valid UTF-8.
func NFKD(s string) string { return norm.NFKD.String(s) }

// A Decomposer appends NFKD forms, as NFKD gives them, and allocates
// nothing when the buffer it appends to has room: the slug filter
// decomposes one code point at a time into one buffer. Its zero value is
// ready; one Decomposer is not for concurrent use.
type Decomposer struct{ it norm.Iter }

// Decomposes reports whether the code point b encodes, valid UTF-8, has a
// decomposition in NFKD's tables. A Hangul syllable, which NFKD
// decomposes by rule rather than by table, has none; every other code
// point with none is its own NFKD form.
func Decomposes(b []byte) bool { return norm.NFKD.Properties(b).Decomposition() != nil }

// AppendNFKD appends the NFKD form of src to dst. src must be valid UTF-8.
func (d *Decomposer) AppendNFKD(dst, src []byte) []byte {
	d.it.Init(norm.NFKD, src)
	for !d.it.Done() {
		dst = append(dst, d.it.Next()...)
	}
	return dst
}
