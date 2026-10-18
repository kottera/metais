// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"bytes"
	"context"
	"testing"

	"github.com/kottera/metais/internal/hostenv"
)

func TestCommandTableEmpty(t *testing.T) {
	if len(commands) != 0 {
		t.Errorf("command table holds %d commands, want none until Wave 2", len(commands))
	}
}

// failingReader fails the test if the shell reads standard input.
type failingReader struct{ t *testing.T }

func (r failingReader) Read([]byte) (int, error) {
	r.t.Error("standard input was read")
	return 0, nil
}

func TestShell(t *testing.T) {
	for _, args := range [][]string{nil, {}, {"version"}, {"help"}, {"render", "pkg.metais"}, {"--help"}, {""}} {
		t.Run("Run", func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			got := Run(context.Background(), Invocation{Args: args, Stdin: failingReader{t}, Stdout: &stdout, Stderr: &stderr})
			if got != exitInternal {
				t.Errorf("Run(%q) = %d, want %d", args, got, exitInternal)
			}
			if stdout.Len() != 0 || stderr.Len() != 0 {
				t.Errorf("Run(%q) wrote stdout %q, stderr %q, want nothing", args, stdout.Bytes(), stderr.Bytes())
			}
		})
		t.Run("Main", func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			got := Main(args, failingReader{t}, &stdout, &stderr, hostenv.Display{NoColor: true})
			if got != exitInternal {
				t.Errorf("Main(%q) = %d, want %d", args, got, exitInternal)
			}
			if stdout.Len() != 0 || stderr.Len() != 0 {
				t.Errorf("Main(%q) wrote stdout %q, stderr %q, want nothing", args, stdout.Bytes(), stderr.Bytes())
			}
		})
	}
}

// TestDispatch proves Run hands an invocation to the command its first
// argument names, unchanged, which is the seam Wave 2's commands fill.
func TestDispatch(t *testing.T) {
	var got Invocation
	commands["probe"] = func(_ context.Context, inv Invocation) int {
		got = inv
		return 7
	}
	t.Cleanup(func() { delete(commands, "probe") })

	var stdout, stderr bytes.Buffer
	stdin := bytes.NewReader(nil)
	inv := Invocation{Args: []string{"probe", "a"}, Stdin: stdin, Stdout: &stdout, Stderr: &stderr, Options: Options{Display: hostenv.Display{NoColor: true}}}
	if status := Run(context.Background(), inv); status != 7 {
		t.Errorf("Run = %d, want the command's status 7", status)
	}
	if len(got.Args) != 2 || got.Args[1] != "a" || got.Stdin != stdin || got.Stdout != &stdout || got.Stderr != &stderr || got.Options != inv.Options {
		t.Errorf("command received %+v, want %+v", got, inv)
	}

	got = Invocation{}
	if status := Main([]string{"probe"}, stdin, &stdout, &stderr, hostenv.Display{NoColor: true}); status != 7 {
		t.Errorf("Main = %d, want the command's status 7", status)
	}
	if want := (Options{Display: hostenv.Display{NoColor: true}}); got.Options != want {
		t.Errorf("Main built options %+v, want %+v: the display passed through and no fault", got.Options, want)
	}
}
