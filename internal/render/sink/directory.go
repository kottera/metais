// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package sink

import (
	"github.com/kottera/metais/internal/fault"
	"github.com/kottera/metais/internal/finding"
	"github.com/kottera/metais/internal/hostfs"
	"github.com/kottera/metais/internal/limits"
	"github.com/kottera/metais/internal/render"
)

// Directory is the sink of the directory --out names, out, which phase 3
// finds to be one of two sinks (ADR-005 D14): a directory that does not
// exist, the new-directory sink, or one that does, the existing-directory
// sink, opened, which overwrite lets hold entries. An out that is, or lies
// beneath, a file is refused.
func Directory(out string, overwrite bool, lim limits.Set, faults fault.Set) Sink {
	return &directory{out: out, overwrite: overwrite, lim: lim, faults: faults}
}

// directory is --out's directory, and in s the sink its phase 3 found it
// to be, or nil before phase 3 or when phase 3 refused it.
type directory struct {
	out       string
	overwrite bool
	lim       limits.Set
	faults    fault.Set
	s         Sink
}

func (d *directory) Check(paths []string) []finding.Finding {
	w, rest, err := hostfs.OpenNearest(d.out)
	if err != nil {
		return []finding.Finding{openFailed(err)}
	}
	w.Close()
	if rest != "." {
		d.s = newDir{out: d.out, faults: d.faults}
		return d.s.Check(paths)
	}
	x, err := hostfs.OpenExisting(d.out, hostfs.WithFaults(d.faults))
	if err != nil {
		return []finding.Finding{openFailed(err)}
	}
	d.s = &existingDir{x: x, overwrite: d.overwrite, lim: d.lim, faults: d.faults, status: map[string]hostfs.Status{}}
	return d.s.Check(paths)
}

func (d *directory) Status(path string) string { return d.s.Status(path) }

func (d *directory) Emit(files []*render.File) Emission { return d.s.Emit(files) }

func (d *directory) Close() {
	if d.s != nil {
		d.s.Close()
	}
}

// openFailed is the finding of err, which opening out returned: out is,
// or lies beneath, a file, or the open failed.
func openFailed(err error) finding.Finding {
	if hostfs.ReasonOf(err) == hostfs.NotDirectory {
		return finding.New(finding.CodeDestinationNotDirectory, invocation, nil, nil)
	}
	return NothingWritten(hostfs.ReasonOf(err))
}
