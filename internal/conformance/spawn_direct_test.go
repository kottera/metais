// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

//go:build linux || !unix

package conformance

// forkStarts is whether Go starts a process here by fork(2). On linux it
// starts one by clone(2) with CLONE_VFORK and CLONE_VM, which shares the
// address space rather than marking it copy-on-write, and on Windows by
// CreateProcess, so the runner starts each process itself.
const forkStarts = false
