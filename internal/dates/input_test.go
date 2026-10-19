// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package dates

import (
	"errors"
	"testing"

	"github.com/kottera/metais/internal/liquid/value"
)

func str(t *testing.T, s string) value.Value {
	t.Helper()
	v, err := value.NewString(s)
	if err != nil {
		t.Fatal(err)
	}
	return v
}

func num(t *testing.T, lit string) value.Value {
	t.Helper()
	n, err := value.ParseNumber(lit)
	if err != nil {
		t.Fatal(err)
	}
	return n
}

// iso writes t back as RFC 3339 in its own offset, through this package's
// own formatter, so a test sees both the instant and the offset kept.
const iso = "%Y-%m-%dT%H:%M:%S.%N%:z"

func TestFromValue(t *testing.T) {
	tests := []struct {
		name string
		in   value.Value
		want string // iso of the result
		unix int64
	}{
		{"full-date", str(t, "2026-10-18"), "2026-10-18T00:00:00.000000000+00:00", 1792281600},
		{"full-date, leap day", str(t, "2024-02-29"), "2024-02-29T00:00:00.000000000+00:00", 1709164800},
		{"full-date, first", str(t, "0000-01-01"), "0000-01-01T00:00:00.000000000+00:00", MinUnix},
		{"full-date, last", str(t, "9999-12-31"), "9999-12-31T00:00:00.000000000+00:00", MaxUnix - 86399},
		{"date-time Z", str(t, "2026-10-18T09:30:00Z"), "2026-10-18T09:30:00.000000000+00:00", 1792315800},
		{"date-time lower t and z", str(t, "2026-10-18t09:30:00z"), "2026-10-18T09:30:00.000000000+00:00", 1792315800},
		{"date-time with a fraction and an offset", str(t, "2026-10-18T09:30:00.25+02:00"), "2026-10-18T09:30:00.250000000+02:00", 1792308600},
		{"date-time, negative half-hour offset", str(t, "2026-10-18T09:30:00-03:30"), "2026-10-18T09:30:00.000000000-03:30", 1792328400},
		{"date-time, Chatham's offset", str(t, "2026-10-18T09:30:00+13:45"), "2026-10-18T09:30:00.000000000+13:45", 1792266300},
		{"date-time, nanoseconds and beyond", str(t, "2026-10-18T09:30:00.123456789123Z"), "2026-10-18T09:30:00.123456789+00:00", 1792315800},
		{"integer zero", num(t, "0"), "1970-01-01T00:00:00.000000000+00:00", 0},
		{"integer", num(t, "1792315800"), "2026-10-18T09:30:00.000000000+00:00", 1792315800},
		{"negative integer", num(t, "-1"), "1969-12-31T23:59:59.000000000+00:00", -1},
		{"integer written 1e3", num(t, "1e3"), "1970-01-01T00:16:40.000000000+00:00", 1000},
		{"integer written 1.0", num(t, "1.0"), "1970-01-01T00:00:01.000000000+00:00", 1},
		{"least integer", num(t, "-62167219200"), "0000-01-01T00:00:00.000000000+00:00", MinUnix},
		{"greatest integer", num(t, "253402300799"), "9999-12-31T23:59:59.000000000+00:00", MaxUnix},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok, err := FromValue(tt.in)
			if err != nil || !ok {
				t.Fatalf("FromValue = %v, %v", ok, err)
			}
			if s, err := FormatString(got, iso); err != nil || s != tt.want {
				t.Errorf("formatted = %q, %v; want %q", s, err, tt.want)
			}
			if u := got.Unix(); u != tt.unix {
				t.Errorf("Unix() = %d, want %d", u, tt.unix)
			}
		})
	}
}

func TestFromValueNil(t *testing.T) {
	got, ok, err := FromValue(value.Null{})
	if ok || err != nil {
		t.Fatalf("FromValue(nil) = %v, %v, %v; want not ok and no error", got, ok, err)
	}
}

// TestFromValueRefuses holds `date` to ADR-004 D11 item 1's forms: every
// other value is ErrInput, and only "now" and "today" name `now`.
func TestFromValueRefuses(t *testing.T) {
	arr, err := value.NewArray(str(t, "2026-10-18"))
	if err != nil {
		t.Fatal(err)
	}
	obj, err := value.NewObject()
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name     string
		in       value.Value
		kind     value.Kind
		namesNow bool
	}{
		{"now", str(t, "now"), value.KindString, true},
		{"today", str(t, "today"), value.KindString, true},
		{"Now", str(t, "Now"), value.KindString, false},
		{"empty string", str(t, ""), value.KindString, false},
		{"Unix seconds as text", str(t, "1792315800"), value.KindString, false},
		{"date-time with no offset", str(t, "2026-10-18T09:30:00"), value.KindString, false},
		{"date-time with a space", str(t, "2026-10-18 09:30:00Z"), value.KindString, false},
		{"second 60", str(t, "2026-10-18T09:30:60Z"), value.KindString, false},
		{"a leap second", str(t, "2016-12-31T23:59:60Z"), value.KindString, false},
		{"one-digit hour", str(t, "2026-10-18T9:30:00Z"), value.KindString, false},
		{"comma fraction", str(t, "2026-10-18T09:30:00,5Z"), value.KindString, false},
		{"full-date out of range", str(t, "2026-02-30"), value.KindString, false},
		{"full-date with a one-digit day", str(t, "2026-10-8"), value.KindString, false},
		{"full-date in lower case", str(t, "2026-10-18z"), value.KindString, false},
		{"full-date with an offset", str(t, "2026-10-18+02:00"), value.KindString, false},
		{"year and month", str(t, "2026-10"), value.KindString, false},
		{"week date", str(t, "2026-W42-7"), value.KindString, false},
		{"ordinal date", str(t, "2026-291"), value.KindString, false},
		{"RFC 2822", str(t, "Sun, 18 Oct 2026 09:30:00 +0000"), value.KindString, false},
		{"UTC year -1", str(t, "0000-01-01T00:00:00+00:01"), value.KindString, false},
		{"fraction", num(t, "1.5"), value.KindNumber, false},
		{"beyond 2^53-1", num(t, "9007199254740992"), value.KindNumber, false},
		{"before year 0", num(t, "-62167219201"), value.KindNumber, false},
		{"after year 9999", num(t, "253402300800"), value.KindNumber, false},
		{"true", value.Bool(true), value.KindBool, false},
		{"false", value.Bool(false), value.KindBool, false},
		{"array", arr, value.KindArray, false},
		{"object", obj, value.KindObject, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok, err := FromValue(tt.in)
			var ie *InputError
			if ok || !errors.Is(err, ErrInput) || !errors.As(err, &ie) {
				t.Fatalf("FromValue = %v, %v, %v; want an *InputError", got, ok, err)
			}
			if ie.Kind != tt.kind || ie.NamesNow != tt.namesNow {
				t.Errorf("InputError = %+v, want kind %v and NamesNow %v", ie, tt.kind, tt.namesNow)
			}
		})
	}
}

func TestFromUnixBounds(t *testing.T) {
	if MinUnix != minUTC.Unix() || MaxUnix != maxUTC.Unix() {
		t.Fatalf("MinUnix, MaxUnix = %d, %d; the range's ends are %d and %d", MinUnix, MaxUnix, minUTC.Unix(), maxUTC.Unix())
	}
	for _, sec := range []int64{MinUnix - 1, MaxUnix + 1, -1 << 63, 1<<63 - 1} {
		if got, ok := FromUnix(sec); ok {
			t.Errorf("FromUnix(%d) = %v, want refused", sec, got.Normalized())
		}
	}
	if got, ok := FromUnix(MinUnix); !ok || got.Normalized() != "0000-01-01T00:00:00Z" {
		t.Errorf("FromUnix(MinUnix) = %q, %v", got.Normalized(), ok)
	}
	if got, ok := FromUnix(MaxUnix); !ok || got.Normalized() != "9999-12-31T23:59:59Z" {
		t.Errorf("FromUnix(MaxUnix) = %q, %v", got.Normalized(), ok)
	}
}
