// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package sink

import (
	"errors"
	"slices"

	"github.com/kottera/metais/internal/fault"
	"github.com/kottera/metais/internal/finding"
	"github.com/kottera/metais/internal/hostfs"
	"github.com/kottera/metais/internal/limits"
	"github.com/kottera/metais/internal/render"
	"github.com/kottera/metais/internal/unitext"
)

// Destination is ADR-005 D13's phase 3 for an --out that exists: the
// directory, opened, and D15 item 4's check of each top-level planned
// path, as Amendment A1 items 4 and 5 extend it, by path. A render into a
// directory that does not exist has none.
type Destination struct {
	x      *hostfs.Existing
	checks []hostfs.PathCheck
	status map[string]hostfs.Status
}

// Close releases the directory, unless a commit already has.
func (d *Destination) Close() {
	if d != nil {
		d.x.Close()
	}
}

// CheckDestination is ADR-005 D13's phase 3, D15, for out and the
// top-level files of plan, every refusal a finding reported with the
// others before anything is written (D15 item 8). An out that does not
// exist is D14 item 1's, and has no destination. An out that is, or lies
// beneath, a file is refused. An existing directory is D14 item 2's, as
// Amendment A1 amends it: one with no entry at all is written into as it
// is (D15 item 3); one that holds any entry, a hidden one included, is
// refused unless overwrite; a leftover staging directory in it refuses the
// render (A1 item 1); and each planned path that passed the path checks
// is checked through the directory (D15 item 4), so a link, a file where
// a directory must be, a case variant, a directory or non-regular target,
// another device, or the staging name refuses it. Nothing in the
// directory is read: only names and Lstat.
func CheckDestination(out string, overwrite bool, plan []*render.File, lim limits.Set, faults fault.Set) (*Destination, []finding.Finding) {
	w, rest, err := hostfs.OpenNearest(out)
	if err != nil {
		if hostfs.ReasonOf(err) == hostfs.NotDirectory {
			return nil, []finding.Finding{finding.New(finding.CodeDestinationNotDirectory, invocation, nil, nil)}
		}
		return nil, []finding.Finding{NothingWritten(hostfs.ReasonOf(err))}
	}
	w.Close()
	if rest != "." {
		return nil, nil
	}
	x, err := hostfs.OpenExisting(out, hostfs.WithFaults(faults))
	if err != nil {
		if hostfs.ReasonOf(err) == hostfs.NotDirectory {
			return nil, []finding.Finding{finding.New(finding.CodeDestinationNotDirectory, invocation, nil, nil)}
		}
		return nil, []finding.Finding{NothingWritten(hostfs.ReasonOf(err))}
	}
	d := &Destination{x: x, status: map[string]hostfs.Status{}}
	var fs []finding.Finding
	if !overwrite {
		entries, err := x.List(MaxListed + 1)
		if err != nil && hostfs.ReasonOf(err) != hostfs.TooLarge {
			d.Close()
			return nil, []finding.Finding{NothingWritten(hostfs.ReasonOf(err))}
		}
		if len(entries) > 0 {
			var names []string
			for _, e := range entries[:min(len(entries), MaxListed)] {
				names = append(names, e.Name)
			}
			slices.Sort(names)
			fs = append(fs, finding.New(finding.CodeDestinationNotEmpty, invocation, nil, finding.Params{
				"entries": stringList(names),
				"more":    finding.Bool(len(entries) > MaxListed),
			}))
		}
	}
	if _, err := x.Lstat(hostfs.StagingDir); err == nil {
		fs = append(fs, StagingExists())
	} else if r := hostfs.ReasonOf(err); r != hostfs.NotFound {
		fs = append(fs, NothingWritten(r))
	}
	// A path that breaks D12 has its own finding, and is not checked here;
	// the output path check's seam (hostfs.CheckPaths) decides which.
	bad := map[string]bool{}
	for _, p := range hostfs.CheckPaths(hostfs.WithFaults(faults), func() []string {
		var bad []string
		for _, pf := range plan {
			if len(unitext.CheckPath(pf.Path(), lim)) > 0 {
				bad = append(bad, pf.Path())
			}
		}
		return bad
	}) {
		bad[p] = true
	}
	var paths []string
	for _, pf := range plan {
		if !bad[pf.Path()] {
			paths = append(paths, pf.Path())
		}
	}
	d.checks = x.Check(paths)
	for _, c := range d.checks {
		if c.Err != nil {
			fs = append(fs, PathRefusal(c.Path, c.Err))
			continue
		}
		d.status[c.Path] = c.Status
	}
	return d, fs
}

// PathRefusal is the finding of err, the refusal of ADR-005 D15 item 4's
// checks of the planned path p, as Amendment A1 items 4 and 5 extend them:
// a link anywhere on the path (ADR-009 D8 item 3), a file where a directory
// must be, a case variant, and a directory or non-regular target, each
// with its own code naming p; another device, the staging name, and an
// error that stopped the check, output.destination.write_failed.
func PathRefusal(p string, err error) finding.Finding {
	params := finding.Params{"path": finding.String(p)}
	switch r := hostfs.ReasonOf(err); r {
	case hostfs.Link:
		return finding.New(finding.CodeDestinationLink, invocation, nil, params)
	case hostfs.NotDirectory:
		return finding.New(finding.CodeDestinationNotDirectory, invocation, nil, params)
	case hostfs.CaseVariant:
		return finding.New(finding.CodeDestinationCaseVariant, invocation, nil, params)
	case hostfs.IsDirectory, hostfs.NotRegularFile:
		params["reason"] = finding.String(string(r))
		return finding.New(finding.CodeDestinationNotRegularFile, invocation, nil, params)
	default:
		return NothingWritten(r)
	}
}

// MaxListed is how many entries a not_empty finding lists.
const MaxListed = 10

// NothingWritten is output.destination.nothing_written for r: a write that
// failed and left the destination as it was.
func NothingWritten(r hostfs.Reason) finding.Finding {
	return finding.New(finding.CodeDestinationNothingWritten, invocation, nil, finding.Params{
		"reason": finding.String(WriteReason(r)),
	})
}

// StagingExists is output.destination.staging_exists: a .metais-staging
// left in an existing destination, which refuses the write before anything
// is written (ADR-005 A1 item 1).
func StagingExists() finding.Finding {
	return finding.New(finding.CodeDestinationStagingExists, invocation, nil, nil)
}

// StageFailed is the finding of err from hostfs.Existing.Stage, which
// writes nothing: the staging leftover, or a write that failed.
func StageFailed(err error) finding.Finding {
	if errors.Is(err, hostfs.ErrStagingExists) {
		return StagingExists()
	}
	return NothingWritten(hostfs.ReasonOf(err))
}

// WriteReason is r as output.destination.write_failed's reason: one of
// ADR-006 D9 item 8's, or exists for a name that appeared where init
// writes, which it never overwrites.
func WriteReason(r hostfs.Reason) string {
	switch {
	case r.IsOS():
		return string(r)
	case r == hostfs.Exists, r == hostfs.NotDirectory, r == hostfs.Link:
		return "exists"
	}
	return string(hostfs.IOError)
}

// stringList is ss as a finding's list of strings.
func stringList(ss []string) finding.Value {
	vs := make([]finding.Value, len(ss))
	for i, s := range ss {
		vs[i] = finding.String(s)
	}
	return finding.List(vs...)
}

// invocation is the subject of every finding about the destination the
// command line names.
var invocation = finding.Subject{Kind: finding.SubjectInvocation}

// Status is D16 item 2's create or replace for the top-level path p,
// which phase 3 checked, or "" for one it did not.
func (d *Destination) Status(p string) string { return string(d.status[p]) }
