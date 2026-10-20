// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

// Package testenv gives tests a hermetic host: a temporary HOME and
// temporary directories, so no test reads the developer's configuration or
// leaves anything behind (ADR-010 D14 item 1).
//
// It is built only on the testing package's own helpers, never on an os
// file or environment function, so it holds no host access of its own:
// GitEnv works on the environment its caller passes it.
package testenv

import (
	"os"
	"strings"
	"testing"
)

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

// GitEnv returns environ as the environment of a git command a test runs,
// so that nothing the host configures can change what git does: every
// variable of git's own (GIT_*) the caller set is dropped, no system or
// global configuration or attributes file is read, no excludes file but the
// repository's own is read, and core.precomposeunicode is off.
//
// The last is what a clone needs on darwin, where git writes
// core.precomposeunicode=true into every repository it creates. With it on,
// git converts each non-ASCII name it reads from a directory through
// iconv's UTF-8-MAC converter, and dies where iconv cannot open it.
// Variables given this way outrank every configuration file, the
// repository's own included.
//
// Pass os.Environ(); GitEnv reads no environment of its own.
func GitEnv(environ []string) []string {
	env := make([]string, 0, len(environ)+10)
	for _, kv := range environ {
		if !strings.HasPrefix(kv, "GIT_") {
			env = append(env, kv)
		}
	}
	return append(env,
		"GIT_CONFIG_NOSYSTEM=1",
		"GIT_ATTR_NOSYSTEM=1",
		"GIT_CONFIG_GLOBAL="+os.DevNull,
		"GIT_CONFIG_COUNT=3",
		"GIT_CONFIG_KEY_0=core.precomposeunicode",
		"GIT_CONFIG_VALUE_0=false",
		"GIT_CONFIG_KEY_1=core.excludesFile",
		"GIT_CONFIG_VALUE_1="+os.DevNull,
		"GIT_CONFIG_KEY_2=core.attributesFile",
		"GIT_CONFIG_VALUE_2="+os.DevNull,
	)
}
