// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package hostfs

import (
	"errors"
	"os"
	"syscall"

	"golang.org/x/sys/unix"
)

// errNotEmpty is rmdir's refusal of a directory that holds an entry.
const errNotEmpty = syscall.ENOTEMPTY

// platformRenameNoReplace renames from to to, single names in the open
// directory dir, with renameatx_np(RENAME_EXCL), the directory-relative
// form of renamex_np, which refuses an existing target with EEXIST
// (ADR-005 D14 item 1.4). A file system that does not support the flag
// answers ENOTSUP or EINVAL.
func platformRenameNoReplace(dir *os.File, _ string, from, to string) error {
	fd := int(dir.Fd())
	err := unix.RenameatxNp(fd, from, fd, to, unix.RENAME_EXCL)
	if errors.Is(err, unix.ENOTSUP) || errors.Is(err, unix.EINVAL) {
		return errNoReplaceUnsupported
	}
	return err
}
