// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

// Package testenv gives tests a hermetic host: a temporary HOME and
// temporary directories, so no test reads the developer's configuration or
// leaves anything behind (ADR-010 D14 item 1).
//
// It is built only on the testing package's own helpers, never on an os
// file or environment function, so it holds no host access of its own.
package testenv

import "testing"

// Home gives t a fresh, empty home directory and returns its path. HOME,
// and USERPROFILE, which is where Windows looks for a home directory, both
// name it until t ends.
//
// Home sets the environment, so a test that calls it must not call
// t.Parallel.
func Home(t testing.TB) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("USERPROFILE", dir)
	return dir
}

// Dir returns a fresh, empty directory that is removed when t ends. Every
// call returns a different directory.
func Dir(t testing.TB) string {
	t.Helper()
	return t.TempDir()
}
