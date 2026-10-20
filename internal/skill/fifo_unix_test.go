// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

//go:build unix

package skill

import (
	"syscall"
	"testing"
)

func mkfifo(t *testing.T, p string) {
	t.Helper()
	if err := syscall.Mkfifo(p, 0o644); err != nil {
		t.Fatal(err)
	}
}
