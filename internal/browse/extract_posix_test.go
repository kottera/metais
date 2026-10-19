// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

//go:build !windows

package browse

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/kottera/metais/internal/hostfs"
	"github.com/kottera/metais/internal/testenv"
)

// TestExtractRefusesALink is ADR-005 D15 item 4 for extract: a symbolic
// link at a file's path, or above it, is refused, even one that points
// inside the destination, and is neither followed nor replaced.
func TestExtractRefusesALink(t *testing.T) {
	tr := tree(t, extractTree)
	to := testenv.Dir(t)
	seed(t, to, "elsewhere/", "target.txt")
	if err := os.Symlink("elsewhere", filepath.Join(to, "inputs")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("target.txt", filepath.Join(to, "metais.yaml")); err != nil {
		t.Fatal(err)
	}
	before := hostTree(t, to)
	_, err := Extract(tr, extractAll, to, ExtractOptions{Overwrite: true}, 0)
	var ce *CheckError
	if !errors.As(err, &ce) {
		t.Fatalf("Extract: %#v", err)
	}
	var got []string
	for _, c := range ce.Refused {
		got = append(got, c.Path+" "+string(hostfs.ReasonOf(c.Err)))
	}
	if want := []string{"inputs/a.json link", "inputs/x/b.json link", "metais.yaml link"}; !slices.Equal(got, want) {
		t.Errorf("refused %q, want %q", got, want)
	}
	if after := hostTree(t, to); !slices.Equal(after, before) {
		t.Errorf("the refusal wrote: %q, was %q", after, before)
	}
}

// TestExtractCommitFailsPartWay is ADR-005 A1 item 3 for extract: a commit
// that stops after moving a file into place reports what it committed, in
// path order, and leaves every other file as it was, with no staging
// directory. The second unit's directory refuses the rename: it cannot be
// written.
func TestExtractCommitFailsPartWay(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("the superuser writes into a directory without write permission")
	}
	tr := tree(t, extractTree)
	to := testenv.Dir(t)
	seed(t, to, "references/")
	locked := filepath.Join(to, "references")
	if err := os.Chmod(locked, 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(locked, 0o755) })
	_, err := Extract(tr, extractAll, to, ExtractOptions{Overwrite: true}, 0)
	var ce *CommitError
	if !errors.As(err, &ce) || ce.Reason != string(hostfs.PermissionDenied) {
		t.Fatalf("Extract: %#v", err)
	}
	if want := []string{"inputs/a.json", "inputs/x/b.json", "metais.yaml"}; !slices.Equal(ce.Committed, want) {
		t.Errorf("committed %q, want %q", ce.Committed, want)
	}
	want := []string{"inputs/", "inputs/a.json={}", "inputs/x/", "inputs/x/b.json={}!", "metais.yaml=manifest", "references/"}
	if got := hostTree(t, to); !slices.Equal(got, want) {
		t.Errorf("after the failure: %q, want %q", got, want)
	}
}
