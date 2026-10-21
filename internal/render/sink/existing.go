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

// existingDir is the sink of an --out that is an existing directory, x,
// opened (ADR-005 D14 item 2, as Amendment A1 amends it), written through
// hostfs.Existing, the only way to write into a directory that exists.
// Once phase 3 has run it holds D15 item 4's check of each top-level
// planned path, as Amendment A1 items 4 and 5 extend it, and each path's
// status.
type existingDir struct {
	x         *hostfs.Existing
	overwrite bool
	lim       limits.Set
	faults    fault.Set
	checks    []hostfs.PathCheck
	status    map[string]hostfs.Status
}

// Check is ADR-005 D13's phase 3, D15, for the directory and the
// top-level paths planned. One with no entry at all is written into as it
// is (D15 item 3); one that holds any entry, a hidden one included, is
// refused unless overwrite; a leftover staging directory in it refuses the
// render (A1 item 1); and each planned path that passed the path checks
// is checked through the directory (D15 item 4), so a link, a file where
// a directory must be, a case variant, a directory or non-regular target,
// another device, or the staging name refuses it. Nothing in the
// directory is read: only names and Lstat.
func (d *existingDir) Check(paths []string) []finding.Finding {
	var fs []finding.Finding
	if !d.overwrite {
		entries, err := d.x.List(MaxListed + 1)
		if err != nil && hostfs.ReasonOf(err) != hostfs.TooLarge {
			return []finding.Finding{NothingWritten(hostfs.ReasonOf(err))}
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
	if _, err := d.x.Lstat(hostfs.StagingDir); err == nil {
		fs = append(fs, StagingExists())
	} else if r := hostfs.ReasonOf(err); r != hostfs.NotFound {
		fs = append(fs, NothingWritten(r))
	}
	// A path that breaks D12 has its own finding, and is not checked here;
	// the output path check's seam (hostfs.CheckPaths) decides which.
	bad := map[string]bool{}
	for _, p := range hostfs.CheckPaths(hostfs.WithFaults(d.faults), func() []string {
		var bad []string
		for _, p := range paths {
			if len(unitext.CheckPath(p, d.lim)) > 0 {
				bad = append(bad, p)
			}
		}
		return bad
	}) {
		bad[p] = true
	}
	var good []string
	for _, p := range paths {
		if !bad[p] {
			good = append(good, p)
		}
	}
	d.checks = d.x.Check(good)
	for _, c := range d.checks {
		if c.Err != nil {
			fs = append(fs, PathRefusal(c.Path, c.Err))
			continue
		}
		d.status[c.Path] = c.Status
	}
	return fs
}

// Status is D16 item 2's create or replace for the top-level path p,
// which phase 3 checked, or "" for one it did not.
func (d *existingDir) Status(p string) string { return string(d.status[p]) }

// Close releases the directory, unless a commit already has.
func (d *existingDir) Close() { d.x.Close() }

// Emit writes files into the directory under ADR-005 D14 item 2, as
// Amendment A1 amends it: every file is prepared beneath the one staging
// directory inside it (item 1), which a leftover refuses with
// output.destination.staging_exists, and is committed by unit (item 2),
// every check made again first. A failure before the first unit is renamed
// leaves the directory as it was, and is output.destination.nothing_written;
// one after is output.destination.commit_failed, listing the paths already
// committed (item 3).
func (d *existingDir) Emit(files []*render.File) Emission {
	defer d.Close()
	// --only writes the one file selected, by the same rules (ADR-005 D18
	// item 3); every planned path was checked all the same, so a selected
	// render is refused exactly when the whole one is (item 2).
	write := map[string]bool{}
	for _, pf := range files {
		write[pf.Path()] = true
	}
	var checks []hostfs.PathCheck
	for _, c := range d.checks {
		if write[c.Path] {
			checks = append(checks, c)
		}
	}
	o, err := d.x.Stage(checks)
	if err != nil {
		f := StageFailed(err)
		return Emission{Failed: &f}
	}
	if f, ok := prepare(o, files); !ok {
		return Emission{Failed: &f}
	}
	err = o.Commit()
	if err == nil {
		return Emission{}
	}
	var ce *hostfs.CommitError
	if !errors.As(err, &ce) || len(ce.Committed) == 0 {
		f := NothingWritten(hostfs.ReasonOf(err))
		return Emission{Failed: &f}
	}
	f := finding.New(finding.CodeDestinationCommitFailed, invocation, nil, finding.Params{
		"reason":    finding.String(WriteReason(hostfs.ReasonOf(err))),
		"committed": stringList(ce.Committed),
	})
	return Emission{Failed: &f}
}
