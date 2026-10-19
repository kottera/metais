// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

//go:build metais_canary_cwd

package template

import (
	"os"
	"path/filepath"
)

// The cwd canary (ADR-009 D16 item 2): a partial the package lacks is read
// from the working directory, which is found when the process starts. It
// is a deliberate leak, built only under its tag, which the SC7 harness
// must catch on the working directory's channel.
func init() {
	wd, err := os.Getwd()
	if err != nil {
		return
	}
	canaryPartial = func(name string) ([]byte, bool) {
		b, err := os.ReadFile(filepath.Join(wd, filepath.FromSlash(name)))
		return b, err == nil
	}
}
