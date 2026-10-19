// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

//go:build unix

package hostfs

import "syscall"

// openFlags are added to every open for reading. O_NONBLOCK keeps an open
// from waiting on a FIFO put where a regular file was examined; the
// handle's Stat then refuses it. A read of a regular file ignores it.
const openFlags = syscall.O_NONBLOCK
