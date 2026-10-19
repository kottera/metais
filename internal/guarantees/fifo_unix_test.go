// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

//go:build unix

package guarantees

import "syscall"

// mkfifo makes a named pipe at p, as a vector's setup asks (ADR-011 D9
// item 4).
func mkfifo(p string) error { return syscall.Mkfifo(p, 0o644) }
