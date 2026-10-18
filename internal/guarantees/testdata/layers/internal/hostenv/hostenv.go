// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

// Package hostenv keeps only-importers' exception: it may import x/term.
package hostenv

import "golang.org/x/term"

// IsTerminal reports whether fd is a terminal.
func IsTerminal(fd int) bool { return term.IsTerminal(fd) }
