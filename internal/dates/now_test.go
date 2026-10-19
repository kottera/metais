// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package dates

import (
	"errors"
	"testing"
)

func TestParseNow(t *testing.T) {
	tests := []struct {
		in         string
		normalized string
		offset     int
	}{
		{"2026-10-18T09:30:00Z", "2026-10-18T09:30:00Z", 0},
		{"2026-10-18t09:30:00z", "2026-10-18T09:30:00Z", 0},
		{"2026-10-18T09:30:00+00:00", "2026-10-18T09:30:00Z", 0},
		{"2026-10-18T09:30:00-00:00", "2026-10-18T09:30:00Z", 0},
		{"2026-10-18T09:30:00+02:00", "2026-10-18T07:30:00Z", 7200},
		{"2026-10-18T09:30:00-05:30", "2026-10-18T15:00:00Z", -19800},
		{"2026-10-18T09:30:00+12:45", "2026-10-17T20:45:00Z", 45900},
		{"2026-10-18T09:30:00+23:59", "2026-10-17T09:31:00Z", 86340},
		{"2026-10-18T09:30:00-23:59", "2026-10-19T09:29:00Z", -86340},
		{"2026-10-18T09:30:00.25+02:00", "2026-10-18T07:30:00.25Z", 7200},
		{"2026-10-18T09:30:00.250000Z", "2026-10-18T09:30:00.25Z", 0},
		{"2026-10-18T09:30:00.000Z", "2026-10-18T09:30:00Z", 0},
		{"2026-10-18T09:30:00.1234567890123Z", "2026-10-18T09:30:00.1234567890123Z", 0},
		{"2024-02-29T23:59:59Z", "2024-02-29T23:59:59Z", 0},
		{"0000-01-01T00:00:00Z", "0000-01-01T00:00:00Z", 0},
		{"0000-01-01T00:00:00-01:00", "0000-01-01T01:00:00Z", -3600},
		{"9999-12-31T23:59:59Z", "9999-12-31T23:59:59Z", 0},
		{"9999-12-31T23:59:59+01:00", "9999-12-31T22:59:59Z", 3600},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			got, err := ParseNow(tt.in)
			if err != nil {
				t.Fatalf("ParseNow(%q) = %v", tt.in, err)
			}
			if s := got.Normalized(); s != tt.normalized {
				t.Errorf("Normalized() = %q, want %q", s, tt.normalized)
			}
			if off := got.Offset(); off != tt.offset {
				t.Errorf("Offset() = %d, want %d", off, tt.offset)
			}
		})
	}
}

// TestParseNowRefuses holds --now to an RFC 3339 date-time with an offset:
// every other form, including those time.Parse alone lets through, is
// refused.
func TestParseNowRefuses(t *testing.T) {
	for _, in := range []string{
		"",
		"now",
		"today",
		"2026-10-18",                      // a full-date has no time and no offset
		"2026-10-18T09:30:00",             // no offset
		"2026-10-18T09:30Z",               // no seconds
		"2026-10-18 09:30:00Z",            // a space for T
		"2026-10-18T9:30:00Z",             // a one-digit hour, which time.Parse's slow path takes
		"2026-10-18T09:30:00,25Z",         // a comma for the point, which time.Parse takes
		"2026-10-18T09:30:00.Z",           // a point with no digits
		"2026-10-18T09:30:00+0200",        // no colon in the offset
		"2026-10-18T09:30:00+02",          // no offset minutes
		"2026-10-18T09:30:00+24:00",       // offset hour 24
		"2026-10-18T09:30:00+02:60",       // offset minute 60
		"2026-10-18T09:30:00UTC",          // a zone name
		"2026-10-18T09:30:00Z ",           // trailing text
		" 2026-10-18T09:30:00Z",           // leading text
		"2026-10-18T09:30:00ZZ",           // two Zs
		"2026-10-18T09:30:60Z",            // second 60 (ADR-004 D11 item 1)
		"2016-12-31T23:59:60Z",            // a real leap second, still refused
		"2026-10-18T24:00:00Z",            // hour 24
		"2026-10-18T09:60:00Z",            // minute 60
		"2026-02-29T09:30:00Z",            // not a leap year
		"2026-13-01T09:30:00Z",            // month 13
		"2026-00-01T09:30:00Z",            // month 0
		"2026-10-00T09:30:00Z",            // day 0
		"2026-10-32T09:30:00Z",            // day 32
		"12026-10-18T09:30:00Z",           // five-digit year
		"+2026-10-18T09:30:00Z",           // signed year
		"2026-1-18T09:30:00Z",             // one-digit month
		"２０２６-10-18T09:30:00Z",            // non-ASCII digits
		"1792315800",                      // Unix seconds are a `date` input, not --now
		"0000-01-01T00:00:00+00:01",       // UTC year -1
		"9999-12-31T23:59:59-00:01",       // UTC year 10000
		"2026-10-18T09:30:00.25+02:00x",   // trailing text after an offset
		"2026-10-18T09:30:00.2.5+02:00",   // two points
		"2026-10-18T09:30:00+02:00:00",    // offset seconds
		"2026-10-18T09:30:00\x00Z",        // a NUL
		"Sun, 18 Oct 2026 09:30:00 +0000", // RFC 1123
	} {
		t.Run(in, func(t *testing.T) {
			got, err := ParseNow(in)
			if !errors.Is(err, ErrNow) {
				t.Fatalf("ParseNow(%q) = %v, %v; want an error wrapping ErrNow", in, got.Normalized(), err)
			}
		})
	}
}
