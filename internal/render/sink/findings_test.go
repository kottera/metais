// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package sink

import (
	"testing"

	"metais.org/metais/internal/finding"
	"metais.org/metais/internal/hostfs"
)

// TestPathRefusal is the finding of each reason ADR-005 D15 item 4's
// checks refuse a planned path with, as Amendment A1 items 4 and 5 extend
// them. Another device is io_error and names nothing of the device (A1
// item 4): no vector can mount a file system, so this test, with
// internal/hostfs's own device tests, is its proof.
func TestPathRefusal(t *testing.T) {
	for _, c := range []struct {
		reason hostfs.Reason
		code   finding.Code
		path   bool   // the finding names the path
		why    string // its reason parameter, or ""
	}{
		{hostfs.Link, finding.CodeDestinationLink, true, ""},
		{hostfs.NotDirectory, finding.CodeDestinationNotDirectory, true, ""},
		{hostfs.CaseVariant, finding.CodeDestinationCaseVariant, true, ""},
		{hostfs.IsDirectory, finding.CodeDestinationNotRegularFile, true, "is_directory"},
		{hostfs.NotRegularFile, finding.CodeDestinationNotRegularFile, true, "not_regular_file"},
		{hostfs.IOError, finding.CodeDestinationNothingWritten, false, "io_error"},
		{hostfs.Exists, finding.CodeDestinationNothingWritten, false, "exists"},
		{hostfs.PermissionDenied, finding.CodeDestinationNothingWritten, false, "permission_denied"},
	} {
		f := PathRefusal("a/b.txt", &hostfs.Error{Op: "check", Reason: c.reason})
		p, hasPath := f.Param("path")
		ps, _ := p.Str()
		w, _ := f.Param("reason")
		why, _ := w.Str()
		if f.Code() != c.code || hasPath != c.path || hasPath && ps != "a/b.txt" || why != c.why {
			t.Errorf("%s: %s, path %q (%v), reason %q; want %s, path %v, reason %q", c.reason, f.Code(), ps, hasPath, why, c.code, c.path, c.why)
		}
	}
}
