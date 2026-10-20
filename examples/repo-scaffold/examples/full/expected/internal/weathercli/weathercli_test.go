// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package weathercli

import (
	"strings"
	"testing"
)

func TestRun(t *testing.T) {
	var out strings.Builder
	if err := Run(nil, &out); err != nil {
		t.Fatal(err)
	}
	if got, want := out.String(), Description+"\n"; got != want {
		t.Errorf("Run wrote %q, want %q", got, want)
	}
}

func TestRunRefusesAnArgument(t *testing.T) {
	var out strings.Builder
	if err := Run([]string{"extra"}, &out); err == nil {
		t.Error("Run accepted an argument")
	}
}
