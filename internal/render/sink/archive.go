// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package sink

import (
	"io"

	"github.com/kottera/metais/internal/finding"
	"github.com/kottera/metais/internal/render"
	"github.com/kottera/metais/internal/tarw"
)

// Archive is the sink of ADR-005 D17's archive on standard output: it
// checks nothing, since it writes no directory, and its emission is the
// write of the archive the command runs after its report.
func Archive() Sink { return archive{} }

type archive struct{}

func (archive) Check([]string) []finding.Finding { return nil }
func (archive) Status(string) string             { return "" }
func (archive) Close()                           {}

func (archive) Emit(files []*render.File) Emission {
	return Emission{Stream: func(w io.Writer) { WriteArchive(w, files) }}
}

// WriteArchive writes plan, in path order, to w as ADR-005 D17's archive,
// built from the rendered bytes and never from disk (item 6). A failed
// write to w ends it unreported, as every write to standard output is
// (ADR-006 D12 item 5); any other error is a defect.
func WriteArchive(w io.Writer, plan []*render.File) {
	ew := &stickyWriter{w: w}
	tw := tarw.NewWriter(ew)
	for _, pf := range plan {
		size, _ := pf.Measured()
		r, done := pf.Open()
		err := tw.Add(pf.Path(), pf.Executable(), size, r)
		if derr := done(err); err == nil && derr != nil {
			panic("sink: a measured container does not write: " + derr.Error())
		}
		if err != nil {
			if ew.err != nil {
				return
			}
			panic("sink: " + err.Error())
		}
	}
	if err := tw.Close(); err != nil && ew.err == nil {
		panic("sink: " + err.Error())
	}
}
