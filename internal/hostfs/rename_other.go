// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

//go:build unix && !linux && !darwin

package hostfs

import (
	"os"
	"syscall"
)

// errNotEmpty is rmdir's refusal of a directory that holds an entry.
const errNotEmpty = syscall.ENOTEMPTY

// platformRenameNoReplace has no call that refuses an existing target on
// this platform, which is none of ADR-005 D14's, so it answers as a file
// system without the flag does, and the caller checks with Lstat and
// renames.
func platformRenameNoReplace(*os.File, string, string, string) error {
	return errNoReplaceUnsupported
}
