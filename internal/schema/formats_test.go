// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package schema

import (
	"slices"
	"strings"
	"testing"
)

// TestFormatList pins the asserted formats to ADR-003 D4 item 1's list,
// exactly.
func TestFormatList(t *testing.T) {
	want := []string{
		"date", "date-time", "duration", "email", "hostname", "ipv4", "ipv6", "json-pointer",
		"relative-json-pointer", "time", "uri", "uri-reference", "uri-template", "uuid",
	}
	if got := Formats(); !slices.Equal(got, want) {
		t.Errorf("Formats() = %q, want %q", got, want)
	}
}

// TestFormatsAreMETAIS holds ADR-010 D5 item 1's rule that every asserted
// format is checked by METAIS's validator and never the library's. Each
// validator is replaced in turn, first by one that refuses everything and
// then by one that accepts everything, and a compiled schema must follow
// the replacement: a string the library's built-in would accept is
// refused, and one it would refuse is accepted.
func TestFormatsAreMETAIS(t *testing.T) {
	good := map[string]string{
		"date-time": "2026-10-19T12:00:00Z", "date": "2026-10-19", "time": "12:00:00Z", "duration": "P1D",
		"email": "a@example.com", "hostname": "example.com", "ipv4": "192.0.2.1", "ipv6": "::1",
		"uri": "https://example.com/", "uri-reference": "a/b", "uuid": "123e4567-e89b-12d3-a456-426614174000",
		"uri-template": "/a/{b}", "json-pointer": "/a", "relative-json-pointer": "0/a",
	}
	for _, name := range Formats() {
		real := formats[name]
		for _, replacement := range []struct {
			valid    func(string) bool
			instance string
			want     bool
		}{
			{func(string) bool { return false }, good[name], false},
			{func(string) bool { return true }, "\x00not of any format", true},
		} {
			formats[name] = replacement.valid
			s := mustCompile(t, map[string]string{"inputs/s.json": `{"format": "` + name + `"}`}, "inputs/s.json")
			fails, err := s.Validate(mustValue(t, `"`+strings.ReplaceAll(replacement.instance, "\x00", `\u0000`)+`"`))
			formats[name] = real
			if err != nil {
				t.Fatal(err)
			}
			if (len(fails) == 0) != replacement.want {
				t.Errorf("%s: the compiled schema did not use METAIS's validator for %q", name, replacement.instance)
			}
		}
		if !real(good[name]) {
			t.Errorf("%s: %q is refused", name, good[name])
		}
	}
}

// TestFormatsIgnoreNonStrings holds ADR-003 D4 item 3.
func TestFormatsIgnoreNonStrings(t *testing.T) {
	for _, name := range Formats() {
		s := mustCompile(t, map[string]string{"inputs/s.json": `{"format": "` + name + `"}`}, "inputs/s.json")
		for _, v := range []string{`1`, `1.5`, `true`, `null`, `[]`, `{}`} {
			if fails, err := s.Validate(mustValue(t, v)); err != nil || len(fails) > 0 {
				t.Errorf("%s: %s failed: %v", name, v, err)
			}
		}
	}
}

// TestFormatGrammars checks each validator against its grammar, on the
// cases ADR-003 D4 item 1 says validators split on among others.
func TestFormatGrammars(t *testing.T) {
	cases := map[string]struct{ valid, invalid []string }{
		"date-time": {
			[]string{"2026-10-19T12:34:56Z", "2026-10-19t12:34:56z", "2026-10-19T12:34:56.789+05:30",
				"1998-12-31T23:59:60Z", "1998-12-31T15:59:60-08:00", "2024-02-29T00:00:00Z"},
			[]string{"2026-10-19 12:34:56Z", "2026-10-19T12:34:56", "1998-12-31T23:58:60Z", "2023-02-29T00:00:00Z",
				"2026-13-01T00:00:00Z", "2026-10-19T24:00:00Z", "2026-10-19T12:34:56.Z", "2026-10-19T12:34:56+0530",
				"2026-10-19T12:34:56+24:00", "２026-10-19T12:34:56Z"},
		},
		"date": {
			[]string{"2026-10-19", "2000-02-29", "0000-01-01"},
			[]string{"2026-10-32", "1900-02-29", "2026-1-19", "20261019", "2026-10-19T00:00:00Z"},
		},
		"time": {
			[]string{"00:00:00Z", "23:59:60Z", "12:00:00.5-01:00", "08:30:06+00:20"},
			[]string{"12:00:00", "12:00Z", "23:59:60+01:00", "25:00:00Z", "12:60:00Z"},
		},
		"duration": {
			[]string{"P1D", "P1Y2M3D", "PT1H", "PT1H2M3S", "P1W", "P1YT1S", "P0D", "PT36H"},
			[]string{"P", "PT", "P1", "1D", "P1D2Y", "P1Y1D", "PT1H1S", "P1W1D", "P1DT", "P1.5D", "p1d"},
		},
		"email": {
			[]string{"a@example.com", "a.b+c@example.com", `"a b"@example.com`, `"a\"b"@example.com`,
				"a@[192.0.2.1]", "a@[IPv6:2001:db8::1]", "a@[ipv6:::1]", "a@localhost", "!#$%&'*+-/=?^_`{|}~@x"},
			[]string{"a", "@example.com", "a@", ".a@example.com", "a.@example.com", "a..b@example.com",
				"a@-example.com", "a@example..com", "a@[192.0.2.256]", "a@[IPv6:::g]", "a@[tag:x]",
				"a b@example.com", "a@example.com.", strings.Repeat("a", 65) + "@x", "ü@example.com"},
		},
		"hostname": {
			[]string{"example.com", "a", "1example.com", "xn--bcher-kva.example", strings.Repeat("a", 63) + ".com"},
			[]string{"example.com.", ".", "", "-a.com", "a-.com", "a..b", "a_b.com", strings.Repeat("a", 64) + ".com",
				strings.Repeat("a.", 127) + "aa", "bücher.example"},
		},
		"ipv4": {
			[]string{"0.0.0.0", "192.0.2.1", "255.255.255.255"},
			[]string{"256.0.0.1", "1.2.3", "1.2.3.4.5", "01.2.3.4", "1.2.3.04", "1.2.3.a", "١.2.3.4", " 1.2.3.4"},
		},
		"ipv6": {
			[]string{"::", "::1", "2001:db8::1", "2001:0db8:0000:0000:0000:0000:0000:0001", "::ffff:192.0.2.1",
				"1:2:3:4:5:6:7::", "1::8", "FFFF::"},
			[]string{"", ":", ":::", "1::2::3", "12345::", "1:2:3:4:5:6:7:8:9", "1:2:3:4:5:6:7", "::1%eth0",
				"::ffff:192.0.2.256", "1.2.3.4::", "g::1", " ::1", "1:2:3:4:5:6:7:8::"},
		},
		"uri": {
			[]string{"https://example.com/a?b#c", "urn:isbn:0451450523", "mailto:a@example.com", "file:///etc",
				"http://[::1]:80/", "http://[v1.x]/", "a:", "http://u:p@h:8080/p%20q"},
			[]string{"//example.com", "/a/b", "a/b", "1http://x", "http://a b", "http://x/%zz", "http://[::1/",
				"http://x:8a/", "http://x/ü", ":"},
		},
		"uri-reference": {
			[]string{"https://example.com", "/a/b", "a/b", "", "#frag", "?q", "//host/p", "./a:b"},
			[]string{"a:b c", "a/%g0", "\\a", "#a#b", "a b"},
		},
		"uuid": {
			[]string{"123e4567-e89b-12d3-a456-426614174000", "123E4567-E89B-12D3-A456-426614174000",
				"00000000-0000-0000-0000-000000000000"},
			[]string{"123e4567e89b12d3a456426614174000", "{123e4567-e89b-12d3-a456-426614174000}",
				"123e4567-e89b-12d3-a456-42661417400g", "123e4567-e89b-12d3-a456_426614174000"},
		},
		"uri-template": {
			[]string{"", "/a", "/a/{b}", "{+b}", "{#b,c}", "{/a*}", "{a:3}", "{a.b}", "{%20}", "/ü/{x}"},
			[]string{"{", "{}", "{a", "a}", "{a b}", "{a:0}", "{a:10000}", "{=a}", "{a.}", "{.a..b}", "/a b", "'", "%zz"},
		},
		"json-pointer": {
			[]string{"", "/", "/a", "/a~0b/c~1d", "/0/-"},
			[]string{"a", "/a~", "/a~2", "#/a"},
		},
		"relative-json-pointer": {
			[]string{"0", "1/a", "0#", "120/a~1b", "2/"},
			[]string{"", "#", "-1/a", "+1/a", "01/a", "0##", "a/b", "1#/a"},
		},
	}
	for name, c := range cases {
		valid := formats[name]
		for _, s := range c.valid {
			if !valid(s) {
				t.Errorf("%s: %q is refused", name, s)
			}
		}
		for _, s := range c.invalid {
			if valid(s) {
				t.Errorf("%s: %q is accepted", name, s)
			}
		}
	}
	for _, name := range Formats() {
		if _, ok := cases[name]; !ok {
			t.Errorf("%s has no grammar cases", name)
		}
	}
}
