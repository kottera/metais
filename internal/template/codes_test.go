// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package template

import (
	"testing"

	"metais.org/metais/internal/limits"
)

// TestUnknownFilterAlternative is MTS-FLT-023 in a template's parse:
// template.unknown_filter gives the alternative amount for exactly the
// three money names, and none for a name one edit away from money or
// amount, nor for one far from both. flt/money-near-names refuses the
// near-names too, but a vector's expected params match as a subset
// (ADR-011 D10 item 2), so it cannot see an alternative that should be
// absent; this test is what does.
func TestUnknownFilterAlternative(t *testing.T) {
	for name, want := range map[string]string{
		"money":                  "amount",
		"money_with_currency":    "amount",
		"money_without_currency": "amount",
		"moneys":                 "",
		"mony":                   "",
		"Money":                  "",
		"amout":                  "",
		"amounts":                "",
		"money_with_currencies":  "",
		"frobnicate":             "",
	} {
		_, fs := ParseEntry("templates/t.md", []byte("{{ 1 | "+name+" }}"), limits.Defaults(), false)
		if len(fs) != 1 || fs[0].Code() != CodeUnknownFilter {
			t.Errorf("%s: the parse gives %v, want one %s", name, fs, CodeUnknownFilter)
			continue
		}
		f := fs[0]
		fv, _ := f.Param("filter")
		if filter, _ := fv.Str(); filter != name {
			t.Errorf("%s: the finding names the filter %q", name, filter)
		}
		v, ok := f.Param("alternative")
		got, _ := v.Str()
		if ok != (want != "") || got != want {
			t.Errorf("%s: alternative %q (present %v), want %q (present %v)", name, got, ok, want, want != "")
		}
	}
}
