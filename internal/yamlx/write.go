// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package yamlx

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/kottera/metais/internal/jsonx"
	"github.com/kottera/metais/internal/liquid/value"
)

// Write is v as the `yaml` filter writes it (ADR-004 D10 item 2): a YAML
// 1.2 block document with no `---` and no trailing line feed, members in
// canonical order, and no anchor, alias, tag, block scalar or flow
// collection but `{}` and `[]`, so a value has one text. A number is
// written by jsonx.FormatInputNumber: as the `json` filter writes it,
// except where that text is one the reader refuses. Every byte is METAIS
// code (ADR-010 D3 item 2); no library serializes it. What Write writes,
// Parse reads back as the same value, wherever the profile it reads under
// holds that value at all.
func Write(v value.Value) []byte {
	if scalar, ok := inline(v); ok {
		return []byte(scalar)
	}
	var b strings.Builder
	for i, line := range block(v) {
		if i > 0 {
			b.WriteByte('\n')
		}
		b.WriteString(line)
	}
	return []byte(b.String())
}

// inline is v's text when v is written on the line that names it: a
// scalar, or an empty object or array.
func inline(v value.Value) (string, bool) {
	switch v := v.(type) {
	case value.Null:
		return "null", true
	case value.Bool:
		return strconv.FormatBool(bool(v)), true
	case value.Number:
		return jsonx.FormatInputNumber(v.Float64()), true
	case value.String:
		return quote(v.Text()), true
	case value.Array:
		if v.Len() == 0 {
			return "[]", true
		}
	case value.Object:
		if v.Len() == 0 {
			return "{}", true
		}
	}
	return "", false
}

// block is the lines of v, a non-empty object or array, at no
// indentation.
func block(v value.Value) []string {
	var lines []string
	switch v := v.(type) {
	case value.Object:
		for name, m := range v.All() {
			key := quote(name) + ":"
			if s, ok := inline(m); ok {
				lines = append(lines, key+" "+s)
				continue
			}
			lines = append(lines, key)
			for _, l := range block(m) {
				lines = append(lines, "  "+l)
			}
		}
	case value.Array:
		for _, e := range v.All() {
			if s, ok := inline(e); ok {
				lines = append(lines, "- "+s)
				continue
			}
			for i, l := range block(e) {
				if i == 0 {
					lines = append(lines, "- "+l)
				} else {
					lines = append(lines, "  "+l)
				}
			}
		}
	}
	return lines
}

// plainText is the form a string or name is written plain in; any other
// is double-quoted.
var plainText = regexp.MustCompile(`^[A-Za-z_/]([A-Za-z0-9_./ -]*[A-Za-z0-9_./])?$`)

// quote is s as a plain scalar when ADR-004 D10 item 2 allows it, and
// double-quoted otherwise: `"`, `\` and U+0000 to U+001F escaped as the
// `json` filter escapes them, and U+007F to U+009F, U+2028, U+2029,
// U+FEFF, U+FFFE and U+FFFF as \uxxxx in lowercase hexadecimal. U+FFFE
// and U+FFFF may not stand raw in a YAML stream (YAML 1.2.2 §5.1), and the
// reader refuses them there.
func quote(s string) string {
	if plainText.MatchString(s) {
		switch strings.ToLower(s) {
		case "true", "false", "yes", "no", "on", "off", "y", "n", "null":
		default:
			return s
		}
	}
	var b strings.Builder
	b.WriteByte('"')
	for _, c := range s {
		switch {
		case c == '"':
			b.WriteString(`\"`)
		case c == '\\':
			b.WriteString(`\\`)
		case c == '\b':
			b.WriteString(`\b`)
		case c == '\t':
			b.WriteString(`\t`)
		case c == '\n':
			b.WriteString(`\n`)
		case c == '\f':
			b.WriteString(`\f`)
		case c == '\r':
			b.WriteString(`\r`)
		case c < 0x20, 0x7F <= c && c <= 0x9F, c == 0x2028, c == 0x2029, c == 0xFEFF, c == 0xFFFE, c == 0xFFFF:
			const hex = "0123456789abcdef"
			b.WriteString(`\u`)
			for shift := 12; shift >= 0; shift -= 4 {
				b.WriteByte(hex[c>>shift&0xF])
			}
		default:
			b.WriteRune(c)
		}
	}
	b.WriteByte('"')
	return b.String()
}
