// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package sink

import (
	"errors"

	"metais.org/metais/internal/finding"
	"metais.org/metais/internal/hostfs"
)

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
