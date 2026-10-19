// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package guarantees

import (
	"io/fs"
	"syscall"
)

// accessTime is the access time Lstat reports for info, in nanoseconds.
func accessTime(info fs.FileInfo) (int64, bool) {
	d, ok := info.Sys().(*syscall.Win32FileAttributeData)
	if !ok {
		return 0, false
	}
	return d.LastAccessTime.Nanoseconds(), true
}
