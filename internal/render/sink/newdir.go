// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package sink

import (
	"metais.org/metais/internal/fault"
	"metais.org/metais/internal/finding"
	"metais.org/metais/internal/hostfs"
	"metais.org/metais/internal/render"
)

// newDir is the sink of an --out that does not exist (ADR-005 D14 item
// 1): phase 3 finds nothing there to check, every file is created, and
// the render is written beneath a staging directory beside out, renamed
// to out. A failure removes what the render created and reports
// output.destination.nothing_written.
type newDir struct {
	out    string
	faults fault.Set
}

func (newDir) Check([]string) []finding.Finding { return nil }
func (newDir) Status(string) string             { return string(hostfs.Create) }
func (newDir) Close()                           {}

func (d newDir) Emit(files []*render.File) Emission {
	o, err := hostfs.StageNew(d.out, hostfs.WithFaults(d.faults))
	if err != nil {
		f := openFailed(err)
		return Emission{Failed: &f}
	}
	if f, ok := prepare(o, files); !ok {
		return Emission{Failed: &f}
	}
	if err := o.Commit(); err != nil {
		f := NothingWritten(hostfs.ReasonOf(err))
		return Emission{Failed: &f}
	}
	return Emission{}
}
