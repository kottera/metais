// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

//go:build unix

package hostfs

import (
	"io/fs"
	"syscall"
)

// platformDevice is the device of the file info describes, from its Lstat,
// and true; false when the host's Lstat does not say. It decides ADR-005
// A1 item 4's refusal and is never reported (ADR-009 D12).
func platformDevice(info fs.FileInfo) (uint64, bool) {
	st, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return 0, false
	}
	return uint64(st.Dev), true
}

// reparse reports whether info is a reparse point that its mode does not
// call a link. A unix host has none: a symbolic link is its only link.
func reparse(fs.FileInfo) bool { return false }
