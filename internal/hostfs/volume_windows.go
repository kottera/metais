// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package hostfs

import (
	"io/fs"
	"syscall"
)

// platformDevice answers false: Lstat on windows does not say which volume
// a file is on. ADR-005 A1 item 4's case, a volume mounted inside the
// output directory, is a mount point, which is a reparse point, and every
// reparse point is refused as a link before a device would be compared.
func platformDevice(fs.FileInfo) (uint64, bool) { return 0, false }

// reparse reports whether info is a reparse point: a junction, a volume
// mount point or any other, which a link is refused for wherever a segment
// of an output path is checked (ADR-005 D15 item 4). Lstat reports a
// junction or mount point as irregular rather than as a link.
func reparse(info fs.FileInfo) bool {
	d, ok := info.Sys().(*syscall.Win32FileAttributeData)
	return ok && d.FileAttributes&syscall.FILE_ATTRIBUTE_REPARSE_POINT != 0
}
