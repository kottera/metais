// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

//go:build unix

package schema

import (
	"syscall"
	"time"
)

// selfCPU is this process's own user plus system CPU time, and whether the
// platform reports it.
func selfCPU() (time.Duration, bool) {
	var ru syscall.Rusage
	if err := syscall.Getrusage(syscall.RUSAGE_SELF, &ru); err != nil {
		return 0, false
	}
	return time.Duration(ru.Utime.Nano() + ru.Stime.Nano()), true
}
