// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package hostenv

import "syscall"

// isTerminal reports whether fd is a console. It asks GetConsoleMode, as
// golang.org/x/term does on windows, through the standard library: x/term
// would link golang.org/x/sys/windows, which imports net, and ADR-009 D16
// item 3 keeps net out of the binary on every target.
func isTerminal(fd uintptr) bool {
	var mode uint32
	return syscall.GetConsoleMode(syscall.Handle(fd), &mode) == nil
}
