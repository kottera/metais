// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"bytes"
	"errors"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/kottera/metais/internal/testenv"
)

// TestBinary builds the tool and runs it as a subprocess, so the shell's
// behaviour is proved through main as well as through cli.Run: it reads
// nothing, writes nothing, and exits with status 2.
func TestBinary(t *testing.T) {
	if testing.Short() {
		t.Skip("builds the binary")
	}
	bin := filepath.Join(testenv.Dir(t), "metais")
	build := exec.Command("go", "build", "-trimpath", "-o", bin, ".")
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("go build: %v\n%s", err, out)
	}
	for _, args := range [][]string{nil, {"version"}, {"help"}} {
		cmd := exec.Command(bin, args...)
		cmd.Dir = testenv.Dir(t)
		cmd.Env = []string{"HOME=" + testenv.Dir(t), "NO_COLOR=1"}
		var stdout, stderr bytes.Buffer
		cmd.Stdin, cmd.Stdout, cmd.Stderr = bytes.NewReader([]byte("ignored")), &stdout, &stderr
		err := cmd.Run()
		var exit *exec.ExitError
		if !errors.As(err, &exit) || exit.ExitCode() != 2 {
			t.Errorf("metais %q: %v, want exit status 2", args, err)
		}
		if stdout.Len() != 0 || stderr.Len() != 0 {
			t.Errorf("metais %q wrote stdout %q, stderr %q, want nothing", args, stdout.Bytes(), stderr.Bytes())
		}
	}
}
