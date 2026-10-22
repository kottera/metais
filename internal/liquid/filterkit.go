// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package liquid

import (
	"math/bits"

	"metais.org/metais/internal/liquid/value"
)

// This file is what a FilterDef defined outside this package shares with
// the standard catalog, so that ADR-004's kinds and texts and ADR-009 D11
// item 2's sizes have one definition: METAIS's own filters and date
// (ADR-004 D10, D11), which a dialect adds to Core() through Extend.

// SortWork is the work of sorting n keys whose strings hold strBytes
// bytes (ADR-009 D11 item 2): n·⌈log2 n⌉ comparisons, and each byte
// ⌈log2 n⌉ times; nothing for fewer than two keys.
func SortWork(n int, strBytes int64) int64 {
	if n < 2 {
		return 0
	}
	lg := int64(bits.Len(uint(n - 1)))
	return int64(n)*lg + strBytes*lg
}

// KindName names v's kind as a finding names it: nil, boolean, integer,
// float, string, array or object, a number being an integer or a float as
// ADR-004 D4 item 2 says.
func KindName(v value.Value) string { return kindName(v) }

// Integer returns n's value when the template sees it as an integer
// (ADR-004 D4 item 2): written as one, of magnitude at most 2^53−1.
func Integer(n value.Number) (int64, bool) {
	t := numberKind(n)
	return t.i, t.isInt
}

// ScalarText returns the text of a nil, a boolean, a number or a string
// (ADR-004 D4 item 4), and false for an array or an object.
func ScalarText(v value.Value) (string, bool) {
	switch v.(type) {
	case value.Array, value.Object:
		return "", false
	}
	s, err := text("", v)
	return s, err == nil
}

// FilterError returns an error of the filter named filter, with code and
// args, its Args carrying the filter's name beside args.
func FilterError(filter string, code Code, args map[string]string, detail string) *Error {
	return filterErr(filter, code, args, detail)
}
