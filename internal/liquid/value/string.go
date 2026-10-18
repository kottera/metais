// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package value

import (
	"errors"
	"fmt"
	"unicode/utf8"
)

// String is a sequence of Unicode scalar values (ADR-003 D8 item 4,
// ADR-002 D4 item 1), held as UTF-8. Every String is valid UTF-8, which in
// Go's definition excludes the surrogate code points, so a String can hold
// neither an unpaired surrogate nor a byte sequence that is no character.
//
// The I-JSON refusal of a noncharacter (RFC 7493 §2.1) is a reader's
// finding, reported at the line and column where the reader met it; a
// noncharacter is a scalar value, and this type does not refuse it.
//
// The zero String is the empty string.
type String struct {
	s string
}

// ErrNotScalarValues is returned for text that is not a sequence of
// Unicode scalar values: invalid UTF-8, including an encoded surrogate.
var ErrNotScalarValues = errors.New("not a sequence of Unicode scalar values")

// NewString returns s as a String, refusing it with ErrNotScalarValues
// unless it is valid UTF-8.
func NewString(s string) (String, error) {
	if err := scalarValues(s); err != nil {
		return String{}, err
	}
	return String{s: s}, nil
}

// Text returns the string's UTF-8 text.
func (s String) Text() string { return s.s }

// scalarValues refuses s unless it is valid UTF-8, naming the byte offset
// of the first byte that is not.
func scalarValues(s string) error {
	for i, r := range s {
		if r == utf8.RuneError {
			if _, size := utf8.DecodeRuneInString(s[i:]); size == 1 {
				return fmt.Errorf("%w: invalid UTF-8 at byte %d", ErrNotScalarValues, i)
			}
		}
	}
	return nil
}
