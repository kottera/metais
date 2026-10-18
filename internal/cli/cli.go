// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

// Package cli is the metais command line: its command table and the
// dispatch from an invocation to a command (ADR-008).
//
// The whole tool runs in-process through Run, so a test drives a command
// exactly as cmd/metais does, with its own streams and, where it proves a
// guarantee, its own faults (ADR-010 D13 item 3).
package cli

import (
	"context"
	"io"

	"github.com/kottera/metais/internal/fault"
	"github.com/kottera/metais/internal/hostenv"
)

// Options is what an invocation carries besides its arguments and streams.
type Options struct {
	// Display is the host's display setting, read by hostenv.Isolate.
	Display hostenv.Display
	// Faults are the faults a test injects. Main leaves it zero, and
	// nothing reachable from the command line sets it.
	Faults fault.Set
}

// Invocation is one run of the tool.
type Invocation struct {
	Args    []string
	Stdin   io.Reader
	Stdout  io.Writer
	Stderr  io.Writer
	Options Options
}

// command runs one command of the table and returns the exit status.
type command func(ctx context.Context, inv Invocation) int

// commands is the command table, keyed by the command's name.
var commands = map[string]command{}

// exitInternal is the exit status of a defect in the tool (ADR-006 D12).
const exitInternal = 2

// Main runs the tool for cmd/metais: args are the arguments after the
// program name, and d is what hostenv.Isolate returned. It builds the
// invocation with a zero fault set and returns the exit status.
func Main(args []string, stdin io.Reader, stdout, stderr io.Writer, d hostenv.Display) int {
	return Run(context.Background(), Invocation{
		Args:    args,
		Stdin:   stdin,
		Stdout:  stdout,
		Stderr:  stderr,
		Options: Options{Display: d},
	})
}

// Run dispatches inv to the command its first argument names and returns
// that command's exit status.
//
// An invocation that names no command in the table reads nothing, writes
// nothing and exits with status 2: the table is empty until the commands
// are written, so every invocation reaches a part of the tool that does not
// exist yet, which is the tool's defect, not the caller's.
func Run(ctx context.Context, inv Invocation) int {
	if len(inv.Args) > 0 {
		if cmd, ok := commands[inv.Args[0]]; ok {
			return cmd(ctx, inv)
		}
	}
	return exitInternal
}
