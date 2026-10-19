// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package hostfs

import (
	"os"
	"path/filepath"
	"syscall"
)

// errNotEmpty is RemoveDirectory's refusal of a directory that holds an
// entry, ERROR_DIR_NOT_EMPTY.
const errNotEmpty = syscall.Errno(145)

// platformRenameNoReplace renames from, a single name in the directory
// fromPath, to to, a single name in the directory toPath, with MoveFileW, which takes no flags and fails on any
// existing target, file or directory: the refusal ADR-005 D14 item 1.4
// asks of MoveFileExW without MOVEFILE_REPLACE_EXISTING (ADR-010 D11 item
// 3, Amendment A1). It is the standard library's syscall.MoveFile, so the
// windows build links neither golang.org/x/sys nor golang.org/x/term:
// golang.org/x/sys/windows imports net, which ADR-009 D16 item 3 keeps out
// of the binary on every target.
//
// MoveFileW names files by path, not by a directory handle, so each
// directory's path is the root's own path joined with the directory
// beneath it, which the caller has examined with Lstat through the root;
// from and to are single names checked by the caller.
func platformRenameNoReplace(_ *os.File, fromPath string, from string, _ *os.File, toPath string, to string) error {
	src, err := syscall.UTF16PtrFromString(filepath.Join(fromPath, from))
	if err != nil {
		return err
	}
	dst, err := syscall.UTF16PtrFromString(filepath.Join(toPath, to))
	if err != nil {
		return err
	}
	err = syscall.MoveFile(src, dst)
	if err == syscall.ERROR_ALREADY_EXISTS || err == syscall.ERROR_FILE_EXISTS {
		return os.ErrExist
	}
	return err
}

// platformUnlink removes name in the directory dirPath with DeleteFileW,
// which refuses a directory: it never removes one. It names the file by
// path, as platformRenameNoReplace does.
func platformUnlink(_ *os.File, dirPath string, name string) error {
	p, err := syscall.UTF16PtrFromString(filepath.Join(dirPath, name))
	if err != nil {
		return err
	}
	return syscall.DeleteFile(p)
}
