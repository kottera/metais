// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

//go:build unix

package conformance

import (
	"os"
	"syscall"
)

// rawMaxRSS is ru_maxrss of an exited process, in the platform's unit,
// and whether the platform reports it.
func rawMaxRSS(ps *os.ProcessState) (int64, bool) {
	ru, ok := ps.SysUsage().(*syscall.Rusage)
	if !ok || ru == nil {
		return 0, false
	}
	return int64(ru.Maxrss), true
}

// selfCPU is this process's own user plus system CPU time, in
// nanoseconds; the fake reads it to burn a known amount.
func selfCPU() int64 {
	var ru syscall.Rusage
	if err := syscall.Getrusage(syscall.RUSAGE_SELF, &ru); err != nil {
		return 0
	}
	return ru.Utime.Nano() + ru.Stime.Nano()
}

// mkfifo makes a named pipe at p.
func mkfifo(p string) error { return syscall.Mkfifo(p, 0o644) }
