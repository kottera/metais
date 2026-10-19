// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package unitext

import (
	"fmt"
	"testing"
	"unicode"

	"golang.org/x/text/cases"
	"golang.org/x/text/unicode/norm"
)

// versionProblem says which of the toolchain's Unicode tables is not at the
// version the specification pins, or returns nil when all are.
func versionProblem(unicodeVersion, normVersion, casesVersion string) error {
	for _, v := range []struct{ name, got string }{
		{"unicode.Version", unicodeVersion},
		{"norm.Version", normVersion},
		{"cases.UnicodeVersion", casesVersion},
	} {
		if v.got != Version {
			return fmt.Errorf("%s is %s, and specification 1.0 pins Unicode %s: a toolchain that moves the tables "+
				"moves the set of accepted names, and is taken only with ADR-010 D1 item 4's change", v.name, v.got, Version)
		}
	}
	return nil
}

// TestUnicodeVersion is ADR-010 D1 item 3's tripwire: it fails when the
// compiling toolchain's Unicode tables are not 15.0.0.
func TestUnicodeVersion(t *testing.T) {
	if err := versionProblem(unicode.Version, norm.Version, cases.UnicodeVersion); err != nil {
		t.Error(err)
	}
}

// TestUnicodeVersionControl holds the tripwire red on a changed value of
// each table's version.
func TestUnicodeVersionControl(t *testing.T) {
	for _, c := range [][3]string{
		{"17.0.0", Version, Version},
		{Version, "17.0.0", Version},
		{Version, Version, "17.0.0"},
		{"15.1.0", "15.1.0", "15.1.0"},
	} {
		if versionProblem(c[0], c[1], c[2]) == nil {
			t.Errorf("the tripwire passed unicode %s, norm %s, cases %s", c[0], c[1], c[2])
		}
	}
}
