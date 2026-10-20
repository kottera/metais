// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/kottera/metais/internal/testenv"
)

// hostEnviron is the environment this test binary started with, taken
// before any test points HOME at a temporary directory: go env reads the
// host's caches from it.
var hostEnviron = os.Environ()

// goHost is the host's toolchain and its caches (testenv.GoHost), read
// once by the go command of the toolchain running this test. A test binary
// built with -trimpath does not know its GOROOT; then the go on PATH asks.
var goHost = sync.OnceValues(func() (testenv.GoHost, error) {
	exe := "go"
	if root := runtime.GOROOT(); root != "" {
		exe = filepath.Join(root, "bin", "go")
	}
	cmd := exec.Command(exe, testenv.GoHostQuery...)
	cmd.Env = append(slices.Clone(hostEnviron), "GOTOOLCHAIN=local")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return testenv.GoHost{}, fmt.Errorf("go %s: %v\n%s", strings.Join(testenv.GoHostQuery, " "), err, stderr.Bytes())
	}
	return testenv.ParseGoHost(hostEnviron, out)
})

// goCommand is the go command with args, in the whole environment
// testenv.GoEnv gives over a fresh directory: it takes nothing from the
// host but its toolchain and caches, and never reaches the network.
func goCommand(t testing.TB, args ...string) *exec.Cmd {
	t.Helper()
	h, err := goHost()
	if err != nil {
		t.Fatalf("locating the host's toolchain: %v", err)
	}
	cmd := exec.Command(h.Go(), args...)
	cmd.Env = hostlessEnv(t)
	return cmd
}

// hostlessEnv is testenv.GoEnv's environment over a fresh directory, for the
// builder's go commands and git.
func hostlessEnv(t testing.TB) []string {
	t.Helper()
	h, err := goHost()
	if err != nil {
		t.Fatalf("locating the host's toolchain: %v", err)
	}
	return testenv.GoEnv(h, testenv.Dir(t))
}
