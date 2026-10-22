// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package schema

import (
	"fmt"

	"metais.org/metais/internal/liquid/value"
)

// Coerce returns the value of text coerced to the declared type set t
// (ADR-003 D14 item 2), as a CSV field (D11 item 6) and a text path set
// (D17 item 3) are:
//
//   - `boolean` in t and text one of YAML's core forms `true`, `True`,
//     `TRUE`, `false`, `False`, `FALSE`: that boolean;
//   - otherwise `integer` in t and text matching `-?(0|[1-9][0-9]*)`: that
//     number;
//   - otherwise `number` in t and text matching RFC 8259 §6's number
//     grammar: that number;
//   - otherwise text, as a string.
//
// Coercion never fails on what the text says (D14 item 3): text that fits
// no listed type stays a string, for validation to report against the
// schema. A number keeps text as its literal (ADR-010 D5 item 4), so it is
// never re-rounded on its way to the validator or a template, and a reader
// applies ADR-003 D8 item 5's refusal of an integer beyond 2^53−1 to it as
// to any number it reads. Text that matches the grammar but rounds to an
// infinity is no binary64, so it stays a string. Dates and times stay
// strings (D14 item 4).
//
// The only error is text that is not a sequence of Unicode scalar values,
// which no reader gives.
func Coerce(text string, t TypeSet) (value.Value, error) {
	if t.Has(TypeBoolean) {
		switch text {
		case "true", "True", "TRUE":
			return value.Bool(true), nil
		case "false", "False", "FALSE":
			return value.Bool(false), nil
		}
	}
	// ParseNumber holds RFC 8259's grammar; its only refusals are text
	// outside it and a literal that rounds to an infinity. Every integer
	// text is in the grammar, so the number row adds nothing after a
	// refused integer.
	if t.Has(TypeInteger) && integerText(text) || t.Has(TypeNumber) {
		if n, err := value.ParseNumber(text); err == nil {
			return n, nil
		}
	}
	s, err := value.NewString(text)
	if err != nil {
		return nil, fmt.Errorf("schema: coerce: %w", err)
	}
	return s, nil
}

// CoerceField returns the value of one CSV field against its column's
// declared type set t (ADR-003 D11 items 5 and 6). An unquoted empty field
// is absent, and ok is false; a quoted empty field is the empty string;
// every other field is text, coerced by Coerce.
func CoerceField(text string, quoted bool, t TypeSet) (v value.Value, ok bool, err error) {
	switch {
	case text == "" && !quoted:
		return nil, false, nil
	case text == "":
		return value.String{}, true, nil
	}
	v, err = Coerce(text, t)
	return v, err == nil, err
}

// integerText reports whether s matches `-?(0|[1-9][0-9]*)`.
func integerText(s string) bool {
	if len(s) > 0 && s[0] == '-' {
		s = s[1:]
	}
	if s == "" || (s[0] == '0' && len(s) > 1) {
		return false
	}
	for _, c := range []byte(s) {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}
