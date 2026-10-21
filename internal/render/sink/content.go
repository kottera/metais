// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package sink

import (
	"io"

	"github.com/kottera/metais/internal/finding"
	"github.com/kottera/metais/internal/render"
)

// Content is the sink of one file's bytes on standard output, which
// --only selects with --out - (ADR-005 D18 item 1): it checks nothing,
// and its emission is the write of the one file selected.
func Content() Sink { return content{} }

type content struct{}

func (content) Check([]string) []finding.Finding { return nil }
func (content) Status(string) string             { return "" }
func (content) Close()                           {}

func (content) Emit(files []*render.File) Emission {
	return Emission{Stream: func(w io.Writer) { WriteContent(w, files[0]) }}
}

// WriteContent writes pf's bytes alone to w (ADR-005 D18 item 1): an
// asset's as the package holds them, and a container's zip as it is
// assembled. A failed write ends it unreported.
func WriteContent(w io.Writer, pf *render.File) {
	ew := &stickyWriter{w: w}
	r, done := pf.Open()
	_, err := io.Copy(ew, r)
	if derr := done(err); err == nil && derr != nil {
		panic("sink: a measured container does not write: " + derr.Error())
	}
	if err != nil && ew.err == nil {
		panic("sink: " + err.Error())
	}
}
