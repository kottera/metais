// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package render

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/kottera/metais/internal/finding"
	"github.com/kottera/metais/internal/testenv"
)

// A render whose write fails part way, after the staging directory exists
// and a file is written beneath it, aborts: neither the staging directory,
// nor the output directory, nor an ancestor the render created is left
// (ADR-005 D14 item 1.5). The second planned path runs through the first,
// a file, which no plan that passed the path checks holds, so the create
// fails beneath the staging directory.
func TestRenderCommitAbortsAFailedWrite(t *testing.T) {
	testenv.Home(t)
	parent := testenv.Dir(t)
	plan := []*File{
		{path: "a.txt", bytes: []byte("A")},
		{path: "a.txt/b.txt", bytes: []byte("B")},
	}
	f, ok := Commit(filepath.Join(parent, "x", "out"), nil, plan, 0)
	if ok {
		t.Fatal("the commit succeeded")
	}
	if f.Code() != finding.CodeDestinationNothingWritten {
		t.Errorf("finding %s, want %s", f.Code(), finding.CodeDestinationNothingWritten)
	}
	entries, err := os.ReadDir(parent)
	if err != nil {
		t.Fatal(err)
	}
	var left []string
	for _, e := range entries {
		left = append(left, e.Name())
	}
	if len(left) != 0 {
		t.Errorf("the failed render left %q", left)
	}
}
