// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

// Package hostfs is the one package that touches the host's files (ADR-009
// D7, D8 and D14 item 3; ADR-010 D11 item 3). Every directory a caller
// names is opened as an os.Root, and every later operation is relative to
// it, so no path a package or an input produced reaches anything outside
// that directory, and no write path is ever built by joining strings onto a
// host path.
//
// Reading opens regular files only. A link, FIFO, socket or device is
// refused from its Lstat mode without being opened; after opening, the
// handle's own Stat must still be a regular file and the same file; and a
// read stops at its limit plus one byte (ADR-009 D7 item 3). A source
// directory is read the same way, as a pkgtree.Tree (OpenSource).
//
// Writing creates every file and directory exclusively, records what it
// created, and removes only those records (ADR-005 D14 item 5). Staging and
// temporary names are fixed, from a SHA-256 of a name (ADR-005 D14), and
// the rename that commits a staged directory or a new file refuses an
// existing target.
//
// No error this package returns carries a path, or the operating system's
// text: an OS error is mapped to one of ADR-006 D9 item 8's reasons, which
// read the same on every platform (ADR-009 D15 item 2). The caller, which
// knows which name it asked about and in what role, says where.
package hostfs

import (
	"errors"
	"io/fs"
	"syscall"
)

// Reason is why an operation failed or was refused.
//
// The first five are ADR-006 D9 item 8's closed set, the params.reason of
// a finding that reports a file the tool could not read or write; every
// operating-system error maps to exactly one of them. The rest are this
// package's own refusals, each of which its caller reports under its own
// code: a link in a data directory is input.link, beneath an output
// directory output.destination.link.
type Reason string

const (
	NotFound         Reason = "not_found"
	PermissionDenied Reason = "permission_denied"
	IsDirectory      Reason = "is_directory"
	NotRegularFile   Reason = "not_regular_file"
	IOError          Reason = "io_error"

	// Link is a symbolic link, a Windows junction or another reparse
	// point where a file or directory was wanted. No link is followed or
	// written through, even one that points inside (ADR-009 D7 item 2, D8
	// item 3).
	Link Reason = "link"

	// NotDirectory is a name that must be a directory and is not.
	NotDirectory Reason = "not_directory"

	// Exists is a target that already exists where an exclusive create or
	// a no-replace rename needs it absent.
	Exists Reason = "exists"

	// Changed is a file that is no longer the file that was examined: the
	// handle opened is not the one Lstat described, or its size moved
	// while it was read (ADR-009 D7 item 3).
	Changed Reason = "changed"

	// TooLarge is a read that reached its limit plus one byte, or a
	// listing that reached its bound plus one name.
	TooLarge Reason = "too_large"

	// BadName is a name this package will not resolve: empty, absolute,
	// holding a "\", ":" or NUL, or a "", "." or ".." segment. The names a
	// caller passes are relative and "/"-separated, and none of them ever
	// leaves the directory it is resolved in.
	BadName Reason = "bad_name"
)

// osReasons are ADR-006 D9 item 8's reasons, in its order.
var osReasons = []Reason{NotFound, PermissionDenied, IsDirectory, NotRegularFile, IOError}

// IsOS reports whether r is one of ADR-006 D9 item 8's reasons.
func (r Reason) IsOS() bool {
	for _, o := range osReasons {
		if r == o {
			return true
		}
	}
	return false
}

// Error is every error this package returns. It names the operation and
// the reason and nothing else: no path, and none of the operating system's
// text, which differs by platform and can carry a path the caller did not
// give (ADR-006 D9 item 8, ADR-009 D15).
type Error struct {
	// Op is the operation: "open", "read", "list", "stat", "create",
	// "mkdir", "rename", "remove" or "close".
	Op string

	// Reason is why it failed.
	Reason Reason
}

func (e *Error) Error() string {
	return "hostfs: " + e.Op + ": " + string(e.Reason)
}

// Is lets errors.Is match fs.ErrNotExist, fs.ErrExist and fs.ErrPermission
// against the reasons that mean them.
func (e *Error) Is(target error) bool {
	switch target {
	case fs.ErrNotExist:
		return e.Reason == NotFound
	case fs.ErrExist:
		return e.Reason == Exists
	case fs.ErrPermission:
		return e.Reason == PermissionDenied
	}
	return false
}

// ReasonOf returns the reason err carries when it is an *Error, and
// IOError for any other non-nil error.
func ReasonOf(err error) Reason {
	var e *Error
	if errors.As(err, &e) {
		return e.Reason
	}
	return IOError
}

func fail(op string, r Reason) error {
	return &Error{Op: op, Reason: r}
}

// mapErr rewrites an operating-system error from op into an *Error, keeping
// only its reason. It returns nil for nil.
func mapErr(op string, err error) error {
	if err == nil {
		return nil
	}
	var e *Error
	if errors.As(err, &e) {
		return &Error{Op: op, Reason: e.Reason}
	}
	return &Error{Op: op, Reason: osReason(err)}
}

// osReason maps an operating-system error to ADR-006 D9 item 8's closed
// set, or to Exists for an exclusive create or a no-replace rename that
// met an existing target.
func osReason(err error) Reason {
	switch {
	case errors.Is(err, fs.ErrExist):
		return Exists
	case errors.Is(err, fs.ErrNotExist), errors.Is(err, syscall.ENOTDIR):
		// A missing component and a component that is a file both say
		// the same thing: nothing has that name.
		return NotFound
	case errors.Is(err, fs.ErrPermission):
		return PermissionDenied
	case errors.Is(err, syscall.EISDIR):
		return IsDirectory
	case errors.Is(err, syscall.ELOOP):
		// The final component is a link that an open refused to follow.
		return NotRegularFile
	}
	return IOError
}
