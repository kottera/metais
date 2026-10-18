// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

//go:build !windows

package hostenv

import "golang.org/x/term"

// isTerminal reports whether fd is a terminal, by golang.org/x/term.
func isTerminal(fd uintptr) bool {
	return term.IsTerminal(int(fd))
}
