// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

//go:build metais_canary_home

package template

import (
	"os"
	"path/filepath"
)

// The home canary (ADR-009 D16 item 2): an asset the package lacks is read
// from the home directory, which is found when the process starts, before
// hostenv.Isolate clears the environment. It is a deliberate leak, built
// only under its tag, which the SC7 harness must catch on HOME's channel.
func init() {
	home := os.Getenv("HOME")
	if home == "" {
		return
	}
	canaryAsset = func(name string) ([]byte, bool) {
		b, err := os.ReadFile(filepath.Join(home, filepath.FromSlash(name)))
		return b, err == nil
	}
}
