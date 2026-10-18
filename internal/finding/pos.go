// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package finding

import "strings"

// Pos is a position in a file or an argument's text (ADR-006 D5 item 3):
// a 1-based line, and a 1-based column counted in Unicode code points.
// The zero Pos is unknown.
type Pos struct{ Line, Column int }

// Known reports whether p names a position.
func (p Pos) Known() bool { return p.Line > 0 }

// Positions is what a parser keeps of one parsed file or argument, so a
// finding about a value can say where it is (ADR-006 D13 item 2). ptr is an
// RFC 6901 JSON Pointer into the parsed value.
type Positions interface {
	// Value is the position of the first character of the value at ptr.
	Value(ptr string) (Pos, bool)
	// Key is the position of the first character of the member name, for
	// an object member.
	Key(ptr string) (Pos, bool)
}

var tokenEscaper = strings.NewReplacer("~", "~0", "/", "~1")

// Pointer returns the RFC 6901 JSON Pointer, in its JSON-string form, made
// of tokens: "" for none, and each token escaped and prefixed with "/".
func Pointer(tokens ...string) string {
	var b strings.Builder
	for _, t := range tokens {
		b.WriteByte('/')
		tokenEscaper.WriteString(&b, t)
	}
	return b.String()
}

var tokenUnescaper = strings.NewReplacer("~1", "/", "~0", "~")

// pointerTokens splits ptr into its decoded reference tokens. RFC 6901
// decodes ~1 before ~0, which a single left-to-right pass of
// tokenUnescaper does, since it never rescans what it wrote.
func pointerTokens(ptr string) []string {
	if ptr == "" {
		return nil
	}
	parts := strings.Split(strings.TrimPrefix(ptr, "/"), "/")
	for i, p := range parts {
		parts[i] = tokenUnescaper.Replace(p)
	}
	return parts
}
