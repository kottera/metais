// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

// Command fakecc is a C compiler that, like a wrapper a host configures as
// CC, needs a variable of its own to run: it refuses unless
// CC_TOOLCHAIN_TARGET_FAKE is set, and otherwise records that it ran and
// runs the real compiler named in the file beside it, <executable>.real.
// TestGoCommandKeepsHostCToolchain builds a cgo program with it.
package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func main() {
	if os.Getenv("CC_TOOLCHAIN_TARGET_FAKE") != "metais-test" {
		fmt.Fprintln(os.Stderr, "fakecc: refused: CC_TOOLCHAIN_TARGET_FAKE is not set")
		os.Exit(2)
	}
	exe, err := os.Executable()
	if err != nil {
		fail(err)
	}
	real, err := os.ReadFile(exe + ".real")
	if err != nil {
		fail(err)
	}
	if err := os.WriteFile(exe+".ran", nil, 0o600); err != nil {
		fail(err)
	}
	cmd := exec.Command(strings.TrimSpace(string(real)), os.Args[1:]...)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		var exit *exec.ExitError
		if errors.As(err, &exit) {
			os.Exit(exit.ExitCode())
		}
		fail(err)
	}
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "fakecc:", err)
	os.Exit(2)
}
