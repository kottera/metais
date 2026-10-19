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

// platformRenameNoReplace renames from, a single name in the open
// directory fromDir, to to, a single name in the open directory toDir,
// with renameatx_np(RENAME_EXCL), the directory-relative form of
// renamex_np, which refuses an existing target with EEXIST (ADR-005 D14
// item 1.4). A file system that does not support the flag answers ENOTSUP
// or EINVAL.
func platformRenameNoReplace(fromDir *os.File, _ string, from string, toDir *os.File, _ string, to string) error {
	err := unix.RenameatxNp(int(fromDir.Fd()), from, int(toDir.Fd()), to, unix.RENAME_EXCL)
	if errors.Is(err, unix.ENOTSUP) || errors.Is(err, unix.EINVAL) {
		return errNoReplaceUnsupported
	}
	return err
}

// platformUnlink removes name, a single name in the open directory dir,
// with unlinkat and no flag, which refuses a directory: it never removes
// one.
func platformUnlink(dir *os.File, _ string, name string) error {
	return unix.Unlinkat(int(dir.Fd()), name, 0)
}

// platformRenameReplace renames from, a single name in the open directory
// fromDir, over to, a single name in the open directory toDir, with
// renameat, which replaces the directory entry and never writes into the
// file it replaces (ADR-005 D15 item 6).
func platformRenameReplace(fromDir *os.File, _ string, from string, toDir *os.File, _ string, to string) error {
	return unix.Renameat(int(fromDir.Fd()), from, int(toDir.Fd()), to)
}
