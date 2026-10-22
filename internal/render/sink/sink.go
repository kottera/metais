// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

// Package sink holds where a render goes: ADR-005 D13's phases 3 and 5,
// the destination checks and emission (ADR-010 Amendment A7 items 2 and
// 7). Each sink is a file of its own, and the command constructs the one
// it runs; nothing here chooses whether standard output carries bytes,
// which the command decides from its line before it runs (ADR-008 D11
// items 2 and 4).
package sink

import (
	"io"
	pathpkg "path"

	"metais.org/metais/internal/finding"
	"metais.org/metais/internal/hostfs"
	"metais.org/metais/internal/render"
)

// Sink is where a render goes, in ADR-005 D13's phases 3 and 5. render.Run
// calls Check and Emit; the command calls Status for its result records,
// and Close once the command is done with the sink.
type Sink interface {
	// Check is phase 3 for the paths of the top-level files planned,
	// every refusal a finding reported with the others, nothing written
	// (D15 item 8).
	Check(paths []string) []finding.Finding
	// Status is D16 item 2's create or replace for the top-level path,
	// or "" for a sink that writes no directory.
	Status(path string) string
	// Emit is phase 5, for the files selected, in path order, which no
	// sink reorders.
	Emit(files []*render.File) Emission
	// Close releases what Check opened.
	Close()
}

// Emission is how phase 5 ended: with its finding when it failed, or with
// a write to standard output the command runs after its report, as its
// outcome's stream.
type Emission = render.Emission

// prepare writes every file of plan through o, in path order, with every
// directory it needs. A failure aborts o, which removes what it wrote.
func prepare(o *hostfs.Output, plan []*render.File) (finding.Finding, bool) {
	for _, pf := range plan {
		err := o.MkdirAll(pathpkg.Dir(pf.Path()))
		if err == nil {
			r, done := pf.Open()
			err = o.Create(pf.Path(), pf.Executable(), r)
			if derr := done(err); err == nil && derr != nil {
				panic("sink: a measured container does not write: " + derr.Error())
			}
		}
		if err != nil {
			o.Abort()
			return NothingWritten(hostfs.ReasonOf(err)), false
		}
	}
	return finding.Finding{}, true
}

// stickyWriter is w that keeps the first error a write returns.
type stickyWriter struct {
	w   io.Writer
	err error
}

func (sw *stickyWriter) Write(b []byte) (int, error) {
	if sw.err != nil {
		return 0, sw.err
	}
	n, err := sw.w.Write(b)
	sw.err = err
	return n, err
}
