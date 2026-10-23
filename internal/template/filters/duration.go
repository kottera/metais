// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package filters

import (
	"strconv"
	"strings"
	"unicode/utf8"

	"metais.org/metais/internal/liquid"
	"metais.org/metais/internal/liquid/value"
)

// durationFilter is ADR-004 Amendment A2 decision item 4 (MTS-FLT-021,
// MTS-FLT-022): a length of time written by a format of %d, %H, %M and
// %S, whose directives mirror date's. It reads whole seconds, an ISO 8601
// duration of W, D, H, M and S, or, with fields:, a string of fields
// separated by ":"; nil is nil. An input it does not read is
// CodeDurationInput with its reason, and a format outside the set
// CodeDurationFormat, a literal one when the template is parsed. Its
// result is charged as built, as date's is (A2 item 7): the format's
// length, which bounds the work, was charged as its shallow size.
var durationFilter = &filter{
	name: "duration",
	min:  1,
	max:  1,
	positional: func(name, arg string, v value.Value) *liquid.Error {
		_, err := durationFormat(name, v)
		return err
	},
	keywords: map[string]func(string, value.Value) *liquid.Error{
		"fields": func(name string, v value.Value) *liquid.Error {
			if _, ok := durationFields(v); !ok {
				return argErr(name, "fields", `fields is one of "d:h:m:s", "d:h:m", "h:m:s", "h:m" and "m:s"`)
			}
			return nil
		},
	},
	apply: func(c *call, in value.Value) (value.Value, *liquid.Error) {
		f, err := durationFormat(c.name, c.pos[0])
		if err != nil {
			return nil, err
		}
		var fields []int64
		if v, ok := c.kw["fields"]; ok {
			fields, _ = durationFields(v)
		}
		if _, ok := in.(value.Null); ok {
			if err := c.units(liquid.Shallow(in)); err != nil {
				return nil, err
			}
			return value.Null{}, nil
		}
		secs, reason := durationSeconds(in, fields)
		if reason != "" {
			return nil, liquid.FilterError(c.name, liquid.CodeDurationInput,
				map[string]string{"type": liquid.KindName(in), "reason": reason},
				"duration does not read this "+liquid.KindName(in)+" ("+reason+")")
		}
		s := f.write(secs)
		return c.text(int64(len(s)), func() string { return s })
	},
}

// maxDuration is the longest duration MTS-FLT-021 reads, 2^53−1 seconds.
const maxDuration = value.MaxSafeInteger

// The seconds in each unit a duration reads or writes (MTS-FLT-021).
const (
	secondsWeek   = 604800
	secondsDay    = 86400
	secondsHour   = 3600
	secondsMinute = 60
)

// The reasons of CodeDurationInput (MTS-FLT-021).
const (
	reasonKind     = "kind"
	reasonCalendar = "calendar"
	reasonRange    = "range"
	reasonForm     = "form"
)

// durationFields is fields:'s value, one of MTS-FLT-022's five, as the
// seconds in each field it names, in order.
func durationFields(v value.Value) ([]int64, bool) {
	s, ok := v.(value.String)
	if !ok {
		return nil, false
	}
	switch s.Text() {
	case "d:h:m:s":
		return []int64{secondsDay, secondsHour, secondsMinute, 1}, true
	case "d:h:m":
		return []int64{secondsDay, secondsHour, secondsMinute}, true
	case "h:m:s":
		return []int64{secondsHour, secondsMinute, 1}, true
	case "h:m":
		return []int64{secondsHour, secondsMinute}, true
	case "m:s":
		return []int64{secondsMinute, 1}, true
	}
	return nil, false
}

// durationSeconds reads in as MTS-FLT-021 says: whole seconds or an ISO
// 8601 duration without fields, a string of those fields with them. reason
// is empty when in is read, and otherwise why it is refused.
func durationSeconds(in value.Value, fields []int64) (secs int64, reason string) {
	switch v := in.(type) {
	case value.Number:
		// A number written as an integer beyond 2^53−1 is a float to
		// the template (ADR-004 D4 item 2), so it is refused as one.
		i, ok := liquid.Integer(v)
		switch {
		case !ok || fields != nil:
			return 0, reasonKind
		case i < 0:
			return 0, reasonRange
		}
		return i, ""
	case value.String:
		s := v.Text()
		var ok bool
		if fields != nil {
			secs, ok, reason = fieldsSeconds(s, fields)
		} else {
			secs, ok, reason = isoSeconds(s)
		}
		switch {
		case ok:
			return secs, ""
		case reason != "":
			return 0, reason
		case calendar(s):
			return 0, reasonCalendar
		}
		return 0, reasonForm
	}
	return 0, reasonKind
}

// calendar reports whether s is an ISO 8601 duration whose length depends
// on the calendar: it begins "P" and holds "Y", or "M" before any "T".
func calendar(s string) bool {
	if !strings.HasPrefix(s, "P") {
		return false
	}
	date, _, _ := strings.Cut(s, "T")
	return strings.Contains(s, "Y") || strings.Contains(date, "M")
}

// adder sums a duration's parts in seconds, noting when the sum would
// pass maxDuration.
type adder struct {
	secs int64
	over bool
}

// add adds digits, a run of ASCII digits, of a unit of unit seconds.
func (a *adder) add(digits string, unit int64) {
	n := int64(0)
	for i := 0; i < len(digits); i++ {
		n = n*10 + int64(digits[i]-'0')
		if n > maxDuration {
			a.over = true
			return
		}
	}
	if n > (maxDuration-a.secs)/unit {
		a.over = true
		return
	}
	a.secs += n * unit
}

// fieldsSeconds reads s as fields:'s fields, each unit's seconds in
// fields: one run of ASCII digits for each, separated by ":", each after
// the first within its unit's range. ok is false and reason empty for a
// string of another form; reason is range for a field beyond its range or
// a sum beyond 2^53−1 seconds.
func fieldsSeconds(s string, fields []int64) (secs int64, ok bool, reason string) {
	parts := strings.Split(s, ":")
	if len(parts) != len(fields) {
		return 0, false, ""
	}
	for _, p := range parts {
		if !allDigits(p) {
			return 0, false, ""
		}
	}
	var a adder
	for i, p := range parts {
		if i > 0 {
			bound := 59
			if fields[i] == secondsHour {
				bound = 23
			}
			if n, err := strconv.Atoi(p); err != nil || n > bound {
				return 0, false, reasonRange
			}
		}
		a.add(p, fields[i])
	}
	if a.over {
		return 0, false, reasonRange
	}
	return a.secs, true, ""
}

// isoSeconds reads s as an ISO 8601 duration matching chapter 8's ABNF:
//
//	duration = "P" ( 1*DIGIT "W" / 1*DIGIT "D" [ "T" time ] / "T" time )
//	time     = 1*DIGIT "H" [ 1*DIGIT "M" ] [ 1*DIGIT "S" ]
//	         / 1*DIGIT "M" [ 1*DIGIT "S" ]
//	         / 1*DIGIT "S"
//
// ok is false and reason empty for a string that does not match; reason is
// range for one that does and is longer than 2^53−1 seconds.
func isoSeconds(s string) (secs int64, ok bool, reason string) {
	rest, found := strings.CutPrefix(s, "P")
	if !found {
		return 0, false, ""
	}
	var a adder
	// number takes a run of digits and the designator after it.
	number := func() (string, byte, bool) {
		i := 0
		for i < len(rest) && '0' <= rest[i] && rest[i] <= '9' {
			i++
		}
		if i == 0 || i == len(rest) {
			return "", 0, false
		}
		n, d := rest[:i], rest[i]
		rest = rest[i+1:]
		return n, d, true
	}
	if !strings.HasPrefix(rest, "T") {
		n, d, ok := number()
		switch {
		case !ok:
			return 0, false, ""
		case d == 'W' && rest == "":
			a.add(n, secondsWeek)
		case d == 'D':
			a.add(n, secondsDay)
		default:
			return 0, false, ""
		}
	}
	if t, found := strings.CutPrefix(rest, "T"); found {
		rest = t
		// The time's designators in order, each at most once, at least one.
		units := map[byte]int64{'H': secondsHour, 'M': secondsMinute, 'S': 1}
		order := "HMS"
		if rest == "" {
			return 0, false, ""
		}
		for rest != "" {
			n, d, ok := number()
			k := strings.IndexByte(order, d)
			if !ok || k < 0 {
				return 0, false, ""
			}
			order = order[k+1:]
			a.add(n, units[d])
		}
	}
	if rest != "" {
		return 0, false, ""
	}
	if a.over {
		return 0, false, reasonRange
	}
	return a.secs, true, ""
}

func allDigits(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}

// durFormat is a compiled duration format (MTS-FLT-022).
type durFormat struct {
	items []durItem
	units []int64 // the seconds of each unit the format names, largest first
}

// durItem is one piece of a format: literal text, or a directive.
type durItem struct {
	lit  string
	unit int64 // the directive's unit in seconds; 0 for literal text
	pad  byte  // 0 for the directive's own padding, else its flag: '-', '_' or '0'
}

// durationUnits are the conversions of MTS-FLT-022's set, with their
// units in seconds.
var durationUnits = map[byte]int64{'d': secondsDay, 'H': secondsHour, 'M': secondsMinute, 'S': 1}

// durationFormat compiles the format argument v: a string of MTS-FLT-022's
// set. A directive is "%", then optionally one of the flags "-", "_" and
// "0", then d, H, M or S, each named at most once; or "%%". Any other
// directive, or an empty format, is CodeDurationFormat, its "directive"
// from its "%" to the character that put it outside the set.
func durationFormat(name string, v value.Value) (durFormat, *liquid.Error) {
	s, ok := v.(value.String)
	if !ok {
		return durFormat{}, argErr(name, "1", "the format is a "+liquid.KindName(v)+", not a string")
	}
	f := s.Text()
	if f == "" {
		return durFormat{}, liquid.FilterError(name, liquid.CodeDurationFormat, nil, "the format is empty")
	}
	var out durFormat
	named := map[int64]bool{}
	lit := 0
	refuse := func(i, n int) (durFormat, *liquid.Error) {
		// A directive refused at a multi-byte character is named through
		// the whole character, so the name is valid UTF-8.
		if c := f[i+n-1]; c >= utf8.RuneSelf {
			_, size := utf8.DecodeRuneInString(f[i+n-1:])
			n += size - 1
		}
		d := f[i : i+n]
		return durFormat{}, liquid.FilterError(name, liquid.CodeDurationFormat, map[string]string{"directive": d},
			"the format has the directive "+strconv.Quote(d)+", outside the set")
	}
	for i := 0; i < len(f); {
		if f[i] != '%' {
			i++
			continue
		}
		if lit < i {
			out.items = append(out.items, durItem{lit: f[lit:i]})
		}
		j := i + 1
		if j < len(f) && f[j] == '%' {
			out.items = append(out.items, durItem{lit: "%"})
			i, lit = j+1, j+1
			continue
		}
		var pad byte
		if j < len(f) && (f[j] == '-' || f[j] == '_' || f[j] == '0') {
			pad = f[j]
			j++
		}
		if j >= len(f) {
			return refuse(i, j-i)
		}
		unit, ok := durationUnits[f[j]]
		if !ok || named[unit] {
			return refuse(i, j+1-i)
		}
		named[unit] = true
		out.items = append(out.items, durItem{unit: unit, pad: pad})
		i, lit = j+1, j+1
	}
	if lit < len(f) {
		out.items = append(out.items, durItem{lit: f[lit:]})
	}
	for _, u := range []int64{secondsDay, secondsHour, secondsMinute, 1} {
		if named[u] {
			out.units = append(out.units, u)
		}
	}
	return out, nil
}

// write writes secs by the format: the largest unit named takes the whole
// count of it, each smaller one the whole count of it in what the larger
// leave, and what lies below the smallest is dropped (MTS-FLT-022).
func (f durFormat) write(secs int64) string {
	counts := map[int64]int64{}
	for _, u := range f.units {
		counts[u] = secs / u
		secs %= u
	}
	var b strings.Builder
	for _, it := range f.items {
		if it.unit == 0 {
			b.WriteString(it.lit)
			continue
		}
		n := strconv.FormatInt(counts[it.unit], 10)
		pad := it.pad
		if pad == 0 {
			pad = '0'
			if it.unit == secondsDay {
				pad = '-'
			}
		}
		switch {
		case pad == '-' || len(n) >= 2:
		case pad == '_':
			n = " " + n
		default:
			n = "0" + n
		}
		b.WriteString(n)
	}
	return b.String()
}
