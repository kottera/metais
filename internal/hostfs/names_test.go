// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package hostfs

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"syscall"
	"testing"
)

// The names are pinned to their bytes: ".metais-staging-" or ".metais-tmp-"
// and the first 16 hex digits of the SHA-256 of the name, which a test
// derives independently (printf %s out | sha256sum). A name from a random
// source or the process would not match, and would not match itself on a
// second call.
func TestFixedNames(t *testing.T) {
	cases := []struct {
		got, want string
	}{
		{StagingName("out"), ".metais-staging-762069bc07a6e1b5"},
		{StagingName("site"), ".metais-staging-fbae041b02c41ed0"},
		{TempName("README.md"), ".metais-tmp-b335630551682c19"},
		{TempName("out"), ".metais-tmp-762069bc07a6e1b5"},
	}
	for _, c := range cases {
		if c.got != c.want {
			t.Errorf("got %q, want %q", c.got, c.want)
		}
	}
	for range 3 {
		if StagingName("out") != cases[0].want || TempName("README.md") != cases[2].want {
			t.Fatal("a fixed name changed between calls")
		}
	}
}

func TestOSName(t *testing.T) {
	good := []string{".", "a", "a/b", ".metais-tmp-x", "a b/c", "é"}
	for _, n := range good {
		if _, ok := osName(n); !ok {
			t.Errorf("osName(%q) refused", n)
		}
	}
	bad := []string{"", "/a", "a/", "a//b", "./a", "a/.", "..", "../a", "a/../b", `a\b`, "c:a", "a:b", "a\x00b"}
	for _, n := range bad {
		if _, ok := osName(n); ok {
			t.Errorf("osName(%q) admitted it", n)
		}
	}
}

// Every operating-system error maps to one of ADR-006 D9 item 8's
// reasons, or to Exists for an exclusive operation, and the error that
// results carries no path: not the one inside the *PathError, not any.
func TestOSErrorsMapToClosedReasons(t *testing.T) {
	const secret = "/home/someone/.ssh/id_ed25519"
	pe := func(err error) error { return &fs.PathError{Op: "open", Path: secret, Err: err} }
	cases := []struct {
		err  error
		want Reason
	}{
		{pe(syscall.ENOENT), NotFound},
		{pe(fs.ErrNotExist), NotFound},
		{pe(syscall.ENOTDIR), NotFound},
		{pe(syscall.EACCES), PermissionDenied},
		{pe(syscall.EPERM), PermissionDenied},
		{pe(fs.ErrPermission), PermissionDenied},
		{pe(syscall.EISDIR), IsDirectory},
		{pe(syscall.ELOOP), NotRegularFile},
		{pe(syscall.EIO), IOError},
		{pe(syscall.ENOSPC), IOError},
		{pe(errors.New("something else " + secret)), IOError},
		{&os.LinkError{Op: "rename", Old: secret, New: secret, Err: syscall.EEXIST}, Exists},
		{fmt.Errorf("wrapped: %w", pe(syscall.EACCES)), PermissionDenied},
	}
	for _, c := range cases {
		err := mapErr("open", c.err)
		var e *Error
		if !errors.As(err, &e) {
			t.Fatalf("mapErr(%v) = %T", c.err, err)
		}
		if e.Reason != c.want {
			t.Errorf("mapErr(%v) reason %s, want %s", c.err, e.Reason, c.want)
		}
		if !e.Reason.IsOS() && e.Reason != Exists {
			t.Errorf("mapErr(%v) reason %s is outside ADR-006 D9 item 8's set", c.err, e.Reason)
		}
		noPath(t, err, secret, "/home", "id_ed25519")
		if errors.Unwrap(err) != nil {
			t.Errorf("mapErr(%v) keeps the OS error, whose text carries the path", c.err)
		}
	}
	if mapErr("open", nil) != nil {
		t.Error("mapErr(nil) is not nil")
	}
}

func TestReasonsOS(t *testing.T) {
	for _, r := range []Reason{NotFound, PermissionDenied, IsDirectory, NotRegularFile, IOError} {
		if !r.IsOS() {
			t.Errorf("%s is not an OS reason", r)
		}
	}
	for _, r := range []Reason{Link, NotDirectory, Exists, Changed, TooLarge, BadName} {
		if r.IsOS() {
			t.Errorf("%s is an OS reason", r)
		}
	}
	if !errors.Is(&Error{Op: "open", Reason: NotFound}, fs.ErrNotExist) ||
		!errors.Is(&Error{Op: "create", Reason: Exists}, fs.ErrExist) ||
		!errors.Is(&Error{Op: "open", Reason: PermissionDenied}, fs.ErrPermission) ||
		errors.Is(&Error{Op: "open", Reason: Link}, fs.ErrNotExist) {
		t.Error("Error.Is does not match the fs errors its reasons mean")
	}
	if ReasonOf(errors.New("x")) != IOError || ReasonOf(fmt.Errorf("w: %w", &Error{Reason: Link})) != Link {
		t.Error("ReasonOf")
	}
}
