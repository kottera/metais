// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

// Package cli breaks only-importers twice and only-referrers once.
package cli

import (
	"os"

	"golang.org/x/sys/unix"
	"golang.org/x/term"
)

// Terminal is term.IsTerminal.
var Terminal = term.IsTerminal

// Run writes to the standard output it was not given.
func Run() {
	_, _ = os.Stdout.Write(nil)
	_ = unix.Sync()
}
