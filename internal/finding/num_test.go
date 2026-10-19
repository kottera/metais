// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package finding

import "testing"

// TestNum holds Num to RFC 8785's number form, the ECMAScript one, over
// each of its branches, and to its refusals.
func TestNum(t *testing.T) {
	for lit, want := range map[string]string{
		"0": "0", "-0": "0", "0.0": "0", "10": "10", "1.50": "1.5", "-2.25": "-2.25", "1e2": "100",
		"1E21": "1e+21", "123456789012345678901": "123456789012345680000", "0.000001": "0.000001",
		"1e-7": "1e-7", "1.5e-7": "1.5e-7", "-1.25e30": "-1.25e+30", "0.1": "0.1", "9007199254740993": "9007199254740992",
		"5e-324": "5e-324", "1.7976931348623157e308": "1.7976931348623157e+308",
	} {
		v, err := Num(lit)
		if err != nil {
			t.Errorf("Num(%s): %v", lit, err)
			continue
		}
		if v.Type() != TypeNumber || v.render() != want {
			t.Errorf("Num(%s) = %s %q, want %q", lit, v.Type(), v.render(), want)
		}
	}
	for _, bad := range []string{"", "01", "+1", "1.", ".5", "1e", "0x10", "NaN", "Infinity", "1e400", " 1"} {
		if _, err := Num(bad); err == nil {
			t.Errorf("Num(%q) is accepted", bad)
		}
	}
}
