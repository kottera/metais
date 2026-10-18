// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

// Package hostenv is the tool's one reader of its host process's
// environment, and its one judge of whether a stream is a terminal
// (ADR-008 D12, ADR-009 D14 item 1).
package hostenv

import (
	"os"
	"time"

	"golang.org/x/term"
)

// Display is what the host may say about how the tool shows its own human
// output. It never reaches a template, an output, a finding or a JSON
// document, and its values are never printed (ADR-008 D12 item 6).
type Display struct {
	// NoColor is true when NO_COLOR is set to a non-empty value.
	NoColor bool
}

// Isolate cuts the process off from its host and returns the one display
// setting read before the cut. cmd/metais calls it before anything else.
//
// In this order, it reads NO_COLOR, the only environment variable the tool
// reads; clears the process environment, so a later os.Getenv anywhere in
// the process, a dependency's included, returns ""; and sets time.Local to
// UTC, so nothing formats in the host's zone or reads its zone database.
// These are defence in depth: no code path asks for a host fact, and
// Isolate makes a missed one find nothing.
func Isolate() Display {
	d := Display{NoColor: os.Getenv("NO_COLOR") != ""}
	os.Clearenv()
	time.Local = time.UTC
	return d
}

// IsTerminal reports whether stream is a terminal. A stream that is not
// backed by a file descriptor is never a terminal.
func IsTerminal(stream any) bool {
	f, ok := stream.(interface{ Fd() uintptr })
	if !ok {
		return false
	}
	return term.IsTerminal(int(f.Fd()))
}
