// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package regexprofile

import (
	"regexp/syntax"
	"testing"

	"github.com/kottera/metais/internal/limits"
)

// FuzzRegexProfile (ADR-010 D14 item 6) holds the profile to its contract
// on any pattern: it never panics; a refusal is an UnsupportedError or a
// LimitError, never an error of the translation; and a pattern it accepts
// has a translation RE2 parses, a program within regex-program, and a
// matcher that runs.
func FuzzRegexProfile(f *testing.F) {
	for _, s := range []string{"abc", `^\d{3}-\d{4}$`, "(?=a)", `😀`, `[^\s\p{Lu}]`} {
		f.Add(s, "subject")
	}
	lim := limits.Defaults()
	f.Fuzz(func(t *testing.T, pattern, subject string) {
		re, err := Compile(pattern, lim, NewMeter(lim.Value(limits.RegexWork)))
		if err != nil {
			if !typed(err) {
				t.Fatalf("Compile(%q): an untyped error %v", pattern, err)
			}
			return
		}
		if _, err := syntax.Parse(re.Translated(), syntax.Perl); err != nil {
			t.Fatalf("Compile(%q) accepted a translation RE2 does not parse: %v", pattern, err)
		}
		if int64(re.Instructions()) > lim.Value(limits.RegexProgram) {
			t.Fatalf("Compile(%q) accepted a program of %d instructions", pattern, re.Instructions())
		}
		re.MatchString(subject)
	})
}
