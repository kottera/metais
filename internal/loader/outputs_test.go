// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package loader

import (
	"testing"

	"metais.org/metais/internal/finding"
	"metais.org/metais/internal/limits"
	"metais.org/metais/internal/liquid"
)

// TestUnknownFilterAlternative is MTS-FLT-023 in a manifest expression:
// template.unknown_filter gives the alternative amount for exactly the
// three money names, and none for a name near them by spelling.
func TestUnknownFilterAlternative(t *testing.T) {
	subj := templateSubject
	for name, want := range map[string]string{
		"money":                  "amount",
		"money_with_currency":    "amount",
		"money_without_currency": "amount",
		"moneys":                 "",
		"Money":                  "",
		"amounts":                "",
	} {
		e := &liquid.Error{Code: liquid.CodeUnknownFilter, Col: 1, Args: map[string]string{"filter": name}}
		f := TemplateFinding(e, subj, finding.Location{}, limits.Defaults(), "")
		v, ok := f.Param("alternative")
		got, _ := v.Str()
		if ok != (want != "") || got != want {
			t.Errorf("%s: alternative %q (present %v), want %q", name, got, ok, want)
		}
	}
}
