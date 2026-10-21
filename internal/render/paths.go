// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package render

import (
	"github.com/kottera/metais/internal/finding"
	"github.com/kottera/metais/internal/hostfs"
	"github.com/kottera/metais/internal/limits"
	templates "github.com/kottera/metais/internal/template"
	"github.com/kottera/metais/internal/unitext"
)

// checkPaths applies the portable-path predicate to the paths of one
// tree (ADR-005 D12): each path alone, and every path of the tree
// together.
func (p *planner) checkPaths(tree []*File) {
	at := func(pf *File) *finding.Location { return ptrTo(p.at(pf.ptr)) }
	p.fs = append(p.fs, hostfs.CheckPaths(hostfs.WithFaults(p.faults), func() []finding.Finding {
		return checkPaths(tree, p.lim, at)
	})...)
}

// checkPaths applies the portable-path predicate to the paths of plan
// (ADR-005 D12): each path alone, and every path of the tree together,
// each finding located by at.
func checkPaths(plan []*File, lim limits.Set, at func(*File) *finding.Location) []finding.Finding {
	var fs []finding.Finding
	byPath := map[string]*File{}
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
func nthOf(plan []*File, p string, n int) *File {
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

// pathFinding is the finding of pr, one rule of ADR-005 D12 items 1 to 5
// the path of pf breaks, at loc (template.PathFinding).
func pathFinding(pf *File, loc *finding.Location, pr unitext.Problem, lim limits.Set) finding.Finding {
	subj := finding.Subject{Kind: finding.SubjectOutput, Name: pf.owner}
	return templates.PathFinding(pr, subj, *loc, pathParams(pf), pf.literal, lim)
}

// pathParams is the parameters every output path finding about pf has:
// the path as rendered, and the index of the fan-out evaluation that
// rendered it (ADR-005 D4 item 2).
func pathParams(pf *File) finding.Params {
	p := finding.Params{"path": finding.String(pf.path)}
	if i := pf.Item(); i != nil {
		p["item"] = finding.Int(int64(*i))
	}
	return p
}

// collisionFinding is the finding of c, ADR-005 D12 item 6 or 7, about the
// planned file pf at loc, which collides with the one at other
// (template.CollisionFinding).
func collisionFinding(pf *File, loc *finding.Location, c unitext.Collision, other *finding.Location) finding.Finding {
	subj := finding.Subject{Kind: finding.SubjectOutput, Name: pf.owner}
	return templates.CollisionFinding(c, subj, *loc, pathParams(pf), *other)
}
