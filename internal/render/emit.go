// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package render

import (
	"errors"
	"io"
	pathpkg "path"

	"github.com/kottera/metais/internal/fault"
	"github.com/kottera/metais/internal/finding"
	"github.com/kottera/metais/internal/hostfs"
	"github.com/kottera/metais/internal/tarw"
)

// WriteArchive writes plan, in path order, to w as ADR-005 D17's archive,
// built from the rendered bytes and never from disk (item 6). A failed
// write to w ends it unreported, as every write to standard output is
// (ADR-006 D12 item 5); any other error is a defect.
func WriteArchive(w io.Writer, plan []*File) {
	ew := &stickyWriter{w: w}
	tw := tarw.NewWriter(ew)
	for _, pf := range plan {
		size, _ := pf.Measured()
		r, done := pf.Open()
		err := tw.Add(pf.path, pf.decl.Executable, size, r)
		if derr := done(err); err == nil && derr != nil {
			panic("render: a measured container does not write: " + derr.Error())
		}
		if err != nil {
			if ew.err != nil {
				return
			}
			panic("render: " + err.Error())
		}
	}
	if err := tw.Close(); err != nil && ew.err == nil {
		panic("render: " + err.Error())
	}
}

// WriteContent writes pf's bytes alone to w (ADR-005 D18 item 1): an
// asset's as the package holds them, and a container's zip as it is
// assembled. A failed write ends it unreported.
func WriteContent(w io.Writer, pf *File) {
	ew := &stickyWriter{w: w}
	r, done := pf.Open()
	_, err := io.Copy(ew, r)
	if derr := done(err); err == nil && derr != nil {
		panic("render: a measured container does not write: " + derr.Error())
	}
	if err != nil && ew.err == nil {
		panic("render: " + err.Error())
	}
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

// Commit writes plan, in path order, to out (ADR-005 D14): into dest,
// the existing directory phase 3 checked, by commitExisting; otherwise
// beneath a staging directory beside out, renamed to out (item 1). A
// failure removes what the render created and reports
// output.destination.nothing_written.
func Commit(out string, dest *Destination, plan []*File, faults fault.Set) (finding.Finding, bool) {
	if dest != nil {
		return commitExisting(dest, plan)
	}
	o, err := hostfs.StageNew(out, hostfs.WithFaults(faults))
	if err != nil {
		if hostfs.ReasonOf(err) == hostfs.NotDirectory {
			return finding.New(finding.CodeDestinationNotDirectory, invocation, nil, nil), false
		}
		return NothingWritten(hostfs.ReasonOf(err)), false
	}
	if f, ok := prepare(o, plan); !ok {
		return f, false
	}
	if err := o.Commit(); err != nil {
		return NothingWritten(hostfs.ReasonOf(err)), false
	}
	return finding.Finding{}, true
}

// commitExisting writes plan into dest under ADR-005 D14 item 2, as
// Amendment A1 amends it: every file is prepared beneath the one staging
// directory inside it (item 1), which a leftover refuses with
// output.destination.staging_exists, and is committed by unit (item 2),
// every check made again first. A failure before the first unit is renamed
// leaves the directory as it was, and is output.destination.nothing_written;
// one after is
// output.destination.commit_failed, listing the paths already committed
// (item 3).
func commitExisting(dest *Destination, plan []*File) (finding.Finding, bool) {
	defer dest.Close()
	// --only writes the one file selected, by the same rules (ADR-005 D18
	// item 3); every planned path was checked all the same, so a selected
	// render is refused exactly when the whole one is (item 2).
	write := map[string]bool{}
	for _, pf := range plan {
		write[pf.path] = true
	}
	var checks []hostfs.PathCheck
	for _, c := range dest.checks {
		if write[c.Path] {
			checks = append(checks, c)
		}
	}
	o, err := dest.x.Stage(checks)
	if err != nil {
		return StageFailed(err), false
	}
	if f, ok := prepare(o, plan); !ok {
		return f, false
	}
	err = o.Commit()
	if err == nil {
		return finding.Finding{}, true
	}
	var ce *hostfs.CommitError
	if !errors.As(err, &ce) || len(ce.Committed) == 0 {
		return NothingWritten(hostfs.ReasonOf(err)), false
	}
	return finding.New(finding.CodeDestinationCommitFailed, invocation, nil, finding.Params{
		"reason":    finding.String(WriteReason(hostfs.ReasonOf(err))),
		"committed": stringList(ce.Committed),
	}), false
}

// prepare writes every file of plan through o, in path order, with every
// directory it needs. A failure aborts o, which removes what it wrote.
func prepare(o *hostfs.Output, plan []*File) (finding.Finding, bool) {
	for _, pf := range plan {
		err := o.MkdirAll(pathpkg.Dir(pf.path))
		if err == nil {
			r, done := pf.Open()
			err = o.Create(pf.path, pf.decl.Executable, r)
			if derr := done(err); err == nil && derr != nil {
				panic("render: a measured container does not write: " + derr.Error())
			}
		}
		if err != nil {
			o.Abort()
			return NothingWritten(hostfs.ReasonOf(err)), false
		}
	}
	return finding.Finding{}, true
}
