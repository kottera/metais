// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package dates

import (
	"errors"
	"strings"
	"testing"
)

// The instants the format tests use, each with the offset it carries.
const (
	// A Sunday morning at an offset of 5:45, with nanoseconds.
	instA = "2026-10-18T09:05:03.123456789+05:45"
	// A Sunday evening in the first ISO week, at UTC.
	instB = "2026-01-04T21:00:00Z"
	// New Year's Day of 2027, in ISO week 53 of 2026.
	instC = "2027-01-01T00:00:00Z"
	// Noon in year 5, at a negative half-hour offset.
	instD = "0005-03-01T12:00:00-00:30"
	// A Monday in ISO week 1 of the next year.
	instE = "2024-12-30T23:59:59.5-09:30"
	// The second before the epoch.
	instF = "1969-12-31T23:59:59Z"
)

// formatCases are expected results, checked against Ruby's strftime for
// every directive Ruby defines the same way. %Z is METAIS's own: Ruby
// writes nothing for an offset with no zone name (ADR-004 D11 item 3).
var formatCases = []struct {
	at, format, want string
}{
	{instA, "%a", "Sun"},
	{instA, "%A", "Sunday"},
	{instA, "%b", "Oct"},
	{instA, "%h", "Oct"},
	{instA, "%B", "October"},
	{instA, "%c", "Sun Oct 18 09:05:03 2026"},
	{instB, "%c", "Sun Jan  4 21:00:00 2026"},
	{instA, "%C", "20"},
	{instD, "%C", "00"},
	{instA, "%d", "18"},
	{instB, "%d", "04"},
	{instA, "%D", "10/18/26"},
	{instA, "%e", "18"},
	{instB, "%e", " 4"},
	{instA, "%F", "2026-10-18"},
	{instD, "%F", "0005-03-01"},
	{instA, "%G", "2026"},
	{instC, "%G", "2026"},
	{instE, "%G", "2025"},
	{instA, "%H", "09"},
	{instB, "%H", "21"},
	{instA, "%I", "09"},
	{instB, "%I", "09"},
	{instC, "%I", "12"},
	{instD, "%I", "12"},
	{instA, "%j", "291"},
	{instB, "%j", "004"},
	{instE, "%j", "365"},
	{instA, "%k", " 9"},
	{instB, "%k", "21"},
	{instA, "%l", " 9"},
	{instC, "%l", "12"},
	{instA, "%L", "123"},
	{instB, "%L", "000"},
	{instE, "%L", "500"},
	{instA, "%m", "10"},
	{instB, "%m", "01"},
	{instA, "%M", "05"},
	{instA, "%n", "\n"},
	{instA, "%N", "123456789"},
	{instE, "%N", "500000000"},
	{instA, "%3N", "123"},
	{instA, "%6N", "123456"},
	{instA, "%9N", "123456789"},
	{instB, "%6N", "000000"},
	{instA, "%p", "AM"},
	{instB, "%p", "PM"},
	{instD, "%p", "PM"},
	{instC, "%p", "AM"},
	{instA, "%P", "am"},
	{instB, "%P", "pm"},
	{instA, "%r", "09:05:03 AM"},
	{instB, "%r", "09:00:00 PM"},
	{instA, "%R", "09:05"},
	{instA, "%s", "1792293603"},
	{instB, "%s", "1767560400"},
	{instF, "%s", "-1"},
	{instA, "%S", "03"},
	{instA, "%t", "\t"},
	{instA, "%T", "09:05:03"},
	{instA, "%u", "7"},
	{instE, "%u", "1"},
	{instA, "%U", "42"},
	{instB, "%U", "01"},
	{instC, "%U", "00"},
	{instE, "%U", "52"},
	{instA, "%V", "42"},
	{instB, "%V", "01"},
	{instC, "%V", "53"},
	{instE, "%V", "01"},
	{instA, "%w", "0"},
	{instC, "%w", "5"},
	{instA, "%W", "41"},
	{instB, "%W", "00"},
	{instE, "%W", "53"},
	{instA, "%x", "10/18/26"},
	{instA, "%X", "09:05:03"},
	{instA, "%y", "26"},
	{instD, "%y", "05"},
	{instA, "%Y", "2026"},
	{instD, "%Y", "0005"},
	{instA, "%z", "+0545"},
	{instB, "%z", "+0000"},
	{instD, "%z", "-0030"},
	{instE, "%z", "-0930"},
	{instA, "%:z", "+05:45"},
	{instD, "%:z", "-00:30"},
	{instA, "%Z", "+05:45"},
	{instB, "%Z", "UTC"},
	{instD, "%Z", "-00:30"},
	{"2026-10-18T09:05:03-00:00", "%Z", "UTC"},
	{instA, "%%", "%"},

	// The flags.
	{instB, "%-d", "4"},
	{instB, "%_d", " 4"},
	{instB, "%0d", "04"},
	{instB, "%-e", "4"},
	{instB, "%0e", "04"},
	{instB, "%_e", " 4"},
	{instB, "%-j", "4"},
	{instB, "%_j", "  4"},
	{instB, "%-m", "1"},
	{instB, "%_m", " 1"},
	{instA, "%-H", "9"},
	{instA, "%_H", " 9"},
	{instA, "%-I", "9"},
	{instA, "%0k", "09"},
	{instA, "%-k", "9"},
	{instA, "%-l", "9"},
	{instA, "%0l", "09"},
	{instA, "%-M", "5"},
	{instA, "%-S", "3"},
	{instB, "%-U", "1"},
	{instB, "%-V", "1"},
	{instB, "%_W", " 0"},
	{instD, "%-y", "5"},
	{instD, "%-Y", "5"},
	{instD, "%_Y", "   5"},
	{instD, "%-C", "0"},
	{instD, "%-G", "5"},
	{instA, "%^a", "SUN"},
	{instA, "%^A", "SUNDAY"},
	{instA, "%^b", "OCT"},
	{instA, "%^h", "OCT"},
	{instA, "%^B", "OCTOBER"},
	{instA, "%^c", "SUN OCT 18 09:05:03 2026"},
	{instA, "%^p", "AM"},
	{instA, "%^P", "AM"},
	{instB, "%^Z", "UTC"},
	{instB, "%^-d", "4"},
	{instB, "%-^a", "SUN"},
	{instB, "%_^e", " 4"},
	// Padding flags change nothing that is not padded.
	{instA, "%-N", "123456789"},
	{instA, "%-L", "123"},
	{instA, "%_3N", "123"},
	{instA, "%-s", "1792293603"},
	{instA, "%-u", "7"},
	{instA, "%-z", "+0545"},
	{instA, "%_:z", "+05:45"},
	{instA, "%-%", "%"},
	{instA, "%-a", "Sun"},
	{instB, "%-D", "01/04/26"},

	// Literal text, and directives beside it.
	{instA, "on %A, %B %-d, %Y at %-I:%M %p (%Z)", "on Sunday, October 18, 2026 at 9:05 AM (+05:45)"},
	{instA, "no directives", "no directives"},
	{instA, "%Y%m%d%H%M%S", "20261018090503"},
	{instA, "100%% at %F", "100% at 2026-10-18"},
	{instA, "日付 %F", "日付 2026-10-18"},
}

func TestFormat(t *testing.T) {
	for _, tt := range formatCases {
		t.Run(tt.at+" "+tt.format, func(t *testing.T) {
			at, err := ParseNow(tt.at)
			if err != nil {
				t.Fatal(err)
			}
			got, err := FormatString(at, tt.format)
			if err != nil {
				t.Fatalf("FormatString(%q) = %v", tt.format, err)
			}
			if got != tt.want {
				t.Errorf("FormatString(%q) = %q, want %q", tt.format, got, tt.want)
			}
		})
	}
}

// closedSet is ADR-004 D11 item 3's directives, as the ADR lists them.
var closedSet = strings.Fields("%a %A %b %h %B %c %C %d %D %e %F %G %H %I %j %k " +
	"%l %L %m %M %n %N %3N %6N %9N %p %P %r %R %s %S %t %T %u %U %V %w %W %x %X %y %Y %z %:z %Z %%")

// TestEveryDirectiveIsTested holds formatCases to the whole closed set:
// each directive is the whole format of at least one case.
func TestEveryDirectiveIsTested(t *testing.T) {
	tested := map[string]bool{}
	for _, tt := range formatCases {
		tested[tt.format] = true
	}
	for _, d := range closedSet {
		if !tested[d] {
			t.Errorf("no format case is the directive %s alone", d)
		}
	}
}

// TestNothingOutsideTheSetCompiles tries every conversion byte, with and
// without each flag, and every width before N, and holds what compiles to
// exactly the closed set.
func TestNothingOutsideTheSetCompiles(t *testing.T) {
	inSet := map[string]bool{}
	for _, d := range closedSet {
		inSet[d] = true
	}
	for c := 0; c < 256; c++ {
		for _, pre := range []string{"", "1", "2", "3", "4", "5", "6", "7", "8", "9", "12", ":", "::", "E", "O", "#"} {
			d := "%" + pre + string(rune(c))
			if c >= 0x80 {
				d = "%" + pre + string([]byte{byte(c)})
			}
			_, err := ParseFormat(d)
			if inSet[d] {
				if err != nil {
					t.Errorf("ParseFormat(%q) = %v, want it compiled", d, err)
				}
				continue
			}
			if !errors.Is(err, ErrFormat) {
				t.Errorf("ParseFormat(%q) compiled; it is outside ADR-004 D11 item 3's set", d)
			}
		}
	}
}

func TestParseFormatRefuses(t *testing.T) {
	tests := []struct {
		format, directive string
	}{
		{"", ""},
		{"%", "%"},
		{"%Y-%", "%"},
		{"%-", "%-"},
		{"%^", "%^"},
		{"%q", "%q"},
		{"%E", "%E"},
		{"%Ey", "%E"},
		{"%Oy", "%O"},
		{"%#a", "%#"},
		{"%g", "%g"},
		{"%v", "%v"},
		{"%+", "%+"},
		{"%Q", "%Q"},
		{"%i", "%i"},
		{"%3", "%3"},
		{"%3L", "%3L"},
		{"%4N", "%4"},
		{"%10N", "%1"},
		{"%12N", "%1"},
		{"%10Y", "%1"},
		{"%:", "%:"},
		{"%:Z", "%:Z"},
		{"%::z", "%::"},
		{"%--d", "%--"},
		{"%-_d", "%-_"},
		{"%_0d", "%_0"},
		{"%0-d", "%0-"},
		{"%^^a", "%^^"},
		{"%%%", "%"},
		{"ok %Y then %Ex", "%E"},
		{"%é", "%é"},
		{"%-日", "%-日"},
		{"%3é", "%3é"},
	}
	for _, tt := range tests {
		t.Run(tt.format, func(t *testing.T) {
			_, err := ParseFormat(tt.format)
			var fe *FormatError
			if !errors.Is(err, ErrFormat) || !errors.As(err, &fe) {
				t.Fatalf("ParseFormat(%q) = %v, want a *FormatError", tt.format, err)
			}
			if fe.Format != tt.format || fe.Directive != tt.directive {
				t.Errorf("FormatError = %+v, want directive %q", fe, tt.directive)
			}
			if _, err := FormatString(Time{}, tt.format); !errors.Is(err, ErrFormat) {
				t.Errorf("FormatString(%q) = %v, want ErrFormat", tt.format, err)
			}
		})
	}
}

// TestFormatsTheOwnOffset pins ADR-004 D11 item 2: one instant written in
// two offsets formats as each offset's wall clock, never as UTC's.
func TestFormatsTheOwnOffset(t *testing.T) {
	east, err := ParseNow("2026-10-18T23:30:00+13:45")
	if err != nil {
		t.Fatal(err)
	}
	west, err := ParseNow("2026-10-18T00:15:00-09:30")
	if err != nil {
		t.Fatal(err)
	}
	if east.Unix() != west.Unix() {
		t.Fatalf("the two texts are different instants: %d and %d", east.Unix(), west.Unix())
	}
	const f = "%F %T %Z %A %j"
	for _, tt := range []struct {
		at   Time
		want string
	}{
		{east, "2026-10-18 23:30:00 +13:45 Sunday 291"},
		{west, "2026-10-18 00:15:00 -09:30 Sunday 291"},
	} {
		if got, _ := FormatString(tt.at, f); got != tt.want {
			t.Errorf("FormatString = %q, want %q", got, tt.want)
		}
	}
}
