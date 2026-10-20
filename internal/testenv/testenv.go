// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

// Package testenv gives tests a hermetic host: a temporary HOME and
// temporary directories, so no test reads the developer's configuration or
// leaves anything behind (ADR-010 D14 item 1).
//
// It is built only on the testing package's own helpers, never on an os
// file or environment function, so it holds no host access of its own:
// GitEnv works on the environment its caller passes it, and GoHost on the
// environment and go env output its caller passes it.
package testenv

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
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

// GoHostQuery is the go env invocation whose output ParseGoHost reads. A
// test runs the go command of the toolchain running it with these
// arguments, in the environment the test process started with and
// GOTOOLCHAIN=local, so the answer names the host's own toolchain and
// caches whatever HOME a test has since set.
var GoHostQuery = []string{"env", "-json", "GOROOT", "GOMODCACHE", "GOCACHE", "CC"}

// GoHost is the whole of what a go command a test runs takes from the host:
// the toolchain, the module cache it already holds, its build cache, and
// the C compiler and PATH that a cgo build (the race detector's) finds it
// by.
type GoHost struct {
	GOROOT     string
	GOMODCACHE string
	GOCACHE    string
	CC         string
	PATH       string
}

// ParseGoHost reads a GoHost from goEnv, the output of the go command run
// with GoHostQuery, and from environ, the environment it was run in.
//
// Pass the environment the test process started with, taken before any
// test changed it; ParseGoHost reads no environment of its own.
func ParseGoHost(environ []string, goEnv []byte) (GoHost, error) {
	var h GoHost
	if err := json.Unmarshal(goEnv, &h); err != nil {
		return GoHost{}, fmt.Errorf("reading go env -json: %v", err)
	}
	for _, kv := range environ {
		if k, v, _ := strings.Cut(kv, "="); strings.EqualFold(k, "PATH") {
			h.PATH = v
		}
	}
	switch {
	case h.GOROOT == "":
		return GoHost{}, errors.New("go env reports no GOROOT")
	case h.GOMODCACHE == "":
		return GoHost{}, errors.New("go env reports no GOMODCACHE")
	case h.GOCACHE == "" || h.GOCACHE == "off":
		return GoHost{}, fmt.Errorf("go env reports GOCACHE %q; a build needs a build cache", h.GOCACHE)
	}
	return h, nil
}

// Go is the go command of h's toolchain.
func (h GoHost) Go() string {
	return filepath.Join(h.GOROOT, "bin", "go")
}

// GoEnv returns the whole environment of a go command a test runs, so that
// nothing the host configures can change what it builds or send it to the
// network. Nothing is inherited but h:
//
//   - HOME, the temporary directory and GOPATH are under dir, which the
//     caller makes and removes;
//   - GOENV=off ignores the developer's go configuration file, and
//     GOWORK=off any workspace above the module;
//   - GOPROXY=off, GOSUMDB=off and GOTOOLCHAIN=local keep the command off
//     the network: a module the host's cache does not hold is an error,
//     never a download;
//   - GOMODCACHE is the cache the host's toolchain already holds, read
//     only (-mod=readonly), so no module is written under dir, where it
//     would be read-only and RemoveAll could not delete it;
//   - GOCACHE is the host's build cache, which changes how long a build
//     takes and never what it builds;
//   - -buildvcs=false stamps no VCS information, so neither git nor the
//     tree's state reaches the binary;
//   - CGO_ENABLED=0, which a caller that builds with the race detector
//     overrides by appending CGO_ENABLED=1: the last value of a variable
//     is the one a command sees.
func GoEnv(h GoHost, dir string) []string {
	return []string{
		"HOME=" + dir,
		"USERPROFILE=" + dir,
		"TMPDIR=" + dir,
		"TMP=" + dir,
		"TEMP=" + dir,
		"GOTMPDIR=" + dir,
		"GOPATH=" + filepath.Join(dir, "gopath"),
		"PATH=" + h.PATH,
		"CC=" + h.CC,
		"GOENV=off",
		"GOWORK=off",
		"GOFLAGS=-mod=readonly -buildvcs=false",
		"GOPROXY=off",
		"GOSUMDB=off",
		"GOTOOLCHAIN=local",
		"GOMODCACHE=" + h.GOMODCACHE,
		"GOCACHE=" + h.GOCACHE,
		"CGO_ENABLED=0",
	}
}
