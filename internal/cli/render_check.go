// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"slices"

	"github.com/kottera/metais/internal/fault"
	"github.com/kottera/metais/internal/finding"
	"github.com/kottera/metais/internal/hostfs"
	"github.com/kottera/metais/internal/limits"
	"github.com/kottera/metais/internal/unitext"
)

// checkPaths applies the portable-path predicate to the paths of one
// tree (ADR-005 D12): each path alone, and every path of the tree
// together.
func (p *planner) checkPaths(tree []*planned) {
	at := func(pf *planned) *finding.Location { return ptrTo(p.at(pf.ptr)) }
	p.fs = append(p.fs, hostfs.CheckPaths(hostfs.WithFaults(p.s.inv.Options.Faults), func() []finding.Finding {
		return checkPaths(tree, p.lim, at)
	})...)
}

// checkPaths applies the portable-path predicate to the paths of plan
// (ADR-005 D12): each path alone, and every path of the tree together,
// each finding located by at.
func checkPaths(plan []*planned, lim limits.Set, at func(*planned) *finding.Location) []finding.Finding {
	var fs []finding.Finding
	byPath := map[string]*planned{}
	paths := make([]string, len(plan))
	for i, pf := range plan {
		paths[i] = pf.path
		if _, seen := byPath[pf.path]; !seen {
			byPath[pf.path] = pf
		}
		for _, pr := range unitext.CheckPath(pf.path, lim) {
			fs = append(fs, pathFinding(pf, at(pf), pr, lim))
		}
	}
	repeats := map[string]int{}
	for _, c := range unitext.Collisions(paths) {
		pf, other := byPath[c.Path], byPath[c.Other]
		if c.Kind == unitext.Duplicate {
			// The n-th repeat of a path is its (n+1)-th declaration.
			repeats[c.Path]++
			pf = nthOf(plan, c.Path, repeats[c.Path])
		}
		fs = append(fs, collisionFinding(pf, at(pf), c, at(other)))
	}
	return fs
}

// nthOf is the planned file of plan at p after n others at p.
func nthOf(plan []*planned, p string, n int) *planned {
	for _, pf := range plan {
		if pf.path == p {
			if n == 0 {
				return pf
			}
			n--
		}
	}
	return nil
}

// The output path codes (ADR-005 D12; ADR-005's interfaces name the
// family output.path).
const (
	codePathEncoding     finding.Code = "output.path.encoding"
	codePathNotNFC       finding.Code = "output.path.not_nfc"
	codePathShape        finding.Code = "output.path.shape"
	codePathCharacter    finding.Code = "output.path.character"
	codePathNotPortable  finding.Code = "output.path.not_portable"
	codePathDuplicate    finding.Code = "output.path.duplicate"
	codePathCaseCollide  finding.Code = "output.path.case_collision"
	codePathFileDirClash finding.Code = "output.path.file_directory_conflict"
)

// pathFinding is the finding of pr, one rule of ADR-005 D12 items 1 to 5
// the path of pf breaks, at loc: the output path codes, and limit.package
// for path-length and path-depth, since a manifest's path is the
// package's own content (ADR-009 D19).
func pathFinding(pf *planned, loc *finding.Location, pr unitext.Problem, lim limits.Set) finding.Finding {
	subj := finding.Subject{Kind: finding.SubjectOutput, Name: pf.owner}
	p := pathParams(pf)
	var code finding.Code
	switch pr.Kind {
	case unitext.Encoding:
		code, p["reason"] = codePathEncoding, finding.String(pr.Reason)
	case unitext.NotNFC:
		code = codePathNotNFC
	case unitext.Shape:
		code, p["reason"] = codePathShape, finding.String(pr.Reason)
	case unitext.Character:
		code, p["code_point"] = codePathCharacter, finding.Int(int64(pr.CodePoint))
	case unitext.NotPortable:
		code, p["reason"] = codePathNotPortable, finding.String(pr.Reason)
	case unitext.Length, unitext.Depth:
		id := limits.PathLength
		if pr.Kind == unitext.Depth {
			id = limits.PathDepth
		}
		if !pf.literal {
			// A path template's path is the render's (ADR-009 D19).
			return lim.ExceededOutput(id, subj, *loc, pr.Observed, true)
		}
		return lim.Breached(id, *loc, pr.Observed, true)
	default:
		panic("cli: an unknown path problem")
	}
	return finding.New(code, subj, loc, p)
}

// pathParams is the parameters every output path finding about pf has:
// the path as rendered, and the index of the fan-out evaluation that
// rendered it (ADR-005 D4 item 2).
func pathParams(pf *planned) finding.Params {
	p := finding.Params{"path": finding.String(pf.path)}
	if i := pf.item(); i != nil {
		p["item"] = finding.Int(int64(*i))
	}
	return p
}

// collisionFinding is the finding of c, ADR-005 D12 item 6 or 7, about the
// planned file pf at loc, which collides with the one at other.
func collisionFinding(pf *planned, loc *finding.Location, c unitext.Collision, other *finding.Location) finding.Finding {
	subj := finding.Subject{Kind: finding.SubjectOutput, Name: pf.owner}
	p := pathParams(pf)
	switch c.Kind {
	case unitext.Duplicate:
		return finding.New(codePathDuplicate, subj, loc, p,
			finding.WithRelated(finding.Related{Location: *other, Note: "first occurrence"}))
	case unitext.CaseCollision:
		p["member"], p["other"] = finding.String(c.Member), finding.String(c.OtherMember)
		return finding.New(codePathCaseCollide, subj, loc, p,
			finding.WithRelated(finding.Related{Location: *other, Note: "collides with"}))
	}
	p["member"], p["other"] = finding.String(c.Member), finding.String(c.OtherMember)
	return finding.New(codePathFileDirClash, subj, loc, p,
		finding.WithRelated(finding.Related{Location: *other, Note: "collides with"}))
}

// destination is ADR-005 D13's phase 3 for an --out that exists: the
// directory, opened, and D15 item 4's check of each top-level planned
// path, as Amendment A1 items 4 and 5 extend it, by path. A render into a
// directory that does not exist has none.
type destination struct {
	x      *hostfs.Existing
	checks []hostfs.PathCheck
	status map[string]hostfs.Status
}

// close releases the directory, unless a commit already has.
func (d *destination) close() {
	if d != nil {
		d.x.Close()
	}
}

// checkDestination is ADR-005 D13's phase 3, D15, for out and the
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
func checkDestination(out string, overwrite bool, plan []*planned, lim limits.Set, faults fault.Set) (*destination, []finding.Finding) {
	w, rest, err := hostfs.OpenNearest(out)
	if err != nil {
		if hostfs.ReasonOf(err) == hostfs.NotDirectory {
			return nil, []finding.Finding{finding.New(finding.CodeDestinationNotDirectory, invocation, nil, nil)}
		}
		return nil, []finding.Finding{writeFailed(hostfs.ReasonOf(err))}
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
		return nil, []finding.Finding{writeFailed(hostfs.ReasonOf(err))}
	}
	d := &destination{x: x, status: map[string]hostfs.Status{}}
	var fs []finding.Finding
	if !overwrite {
		entries, err := x.List(maxListed + 1)
		if err != nil && hostfs.ReasonOf(err) != hostfs.TooLarge {
			d.close()
			return nil, []finding.Finding{writeFailed(hostfs.ReasonOf(err))}
		}
		if len(entries) > 0 {
			var names []string
			for _, e := range entries[:min(len(entries), maxListed)] {
				names = append(names, e.Name)
			}
			slices.Sort(names)
			fs = append(fs, finding.New(finding.CodeDestinationNotEmpty, invocation, nil, finding.Params{
				"entries": stringList(names),
				"more":    finding.Bool(len(entries) > maxListed),
			}))
		}
	}
	if _, err := x.Lstat(hostfs.StagingDir); err == nil {
		fs = append(fs, writeFailed(hostfs.Exists))
	} else if r := hostfs.ReasonOf(err); r != hostfs.NotFound {
		fs = append(fs, writeFailed(r))
	}
	// A path that breaks D12 has its own finding, and is not checked here;
	// the output path check's seam (hostfs.CheckPaths) decides which.
	bad := map[string]bool{}
	for _, p := range hostfs.CheckPaths(hostfs.WithFaults(faults), func() []string {
		var bad []string
		for _, pf := range plan {
			if len(unitext.CheckPath(pf.path, lim)) > 0 {
				bad = append(bad, pf.path)
			}
		}
		return bad
	}) {
		bad[p] = true
	}
	var paths []string
	for _, pf := range plan {
		if !bad[pf.path] {
			paths = append(paths, pf.path)
		}
	}
	d.checks = x.Check(paths)
	for _, c := range d.checks {
		if c.Err != nil {
			fs = append(fs, pathRefusal(c.Path, c.Err))
			continue
		}
		d.status[c.Path] = c.Status
	}
	return d, fs
}

// pathRefusal is the finding of err, the refusal of ADR-005 D15 item 4's
// checks of the planned path p, as Amendment A1 items 4 and 5 extend them:
// a link anywhere on the path (ADR-009 D8 item 3), a file where a directory
// must be, a case variant, and a directory or non-regular target, each
// with its own code naming p; another device, the staging name, and an
// error that stopped the check, output.destination.write_failed.
func pathRefusal(p string, err error) finding.Finding {
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
		return writeFailed(r)
	}
}

// maxListed is how many entries a not_empty finding lists.
const maxListed = 10

// writeFailed is output.destination.write_failed for r, with nothing left
// written.
func writeFailed(r hostfs.Reason) finding.Finding {
	return finding.New(finding.CodeDestinationWriteFailed, invocation, nil, finding.Params{
		"reason":  finding.String(writeReason(r)),
		"written": stringList(nil),
	})
}
