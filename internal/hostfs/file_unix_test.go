// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

//go:build unix

package hostfs

import (
	"path/filepath"
	"testing"
)

// TestFileOutNotRegular is ADR-007 D7 item 5's refusal of a target that
// exists and is not a regular file: a FIFO and a link, which is neither
// followed nor replaced, whether the caller names it or a directory it is
// in.
func TestFileOutNotRegular(t *testing.T) {
	root := dir(t)
	mkfifo(t, root, "fifo/p.metais")
	files(t, root, map[string]string{"target.metais": "x"})
	symlink(t, root, "../target.metais", "link/p.metais")
	for _, c := range []struct {
		path string
		want Reason
	}{
		{filepath.Join(root, "fifo"), NotRegularFile},
		{filepath.Join(root, "fifo", "p.metais"), NotRegularFile},
		{filepath.Join(root, "link"), Link},
		{filepath.Join(root, "link", "p.metais"), Link},
	} {
		_, err := OpenFileOut(c.path, "p.metais")
		wantReason(t, err, c.want, root)
	}
}

// TestFileOutWithinLink is the layout-tree refusal through a link: an
// output directory reached through a symbolic link to a layout tree is
// that tree.
func TestFileOutWithinLink(t *testing.T) {
	root := dir(t)
	src := filepath.Join(root, "src")
	files(t, src, map[string]string{"metais.yaml": "m", "references/r.md": "r"})
	symlink(t, root, "src/references", "refs")
	o, err := OpenFileOut(filepath.Join(root, "refs"), "p.metais")
	if err != nil {
		t.Fatal(err)
	}
	defer o.Close()
	h, err := Hold(src, []string{"metais.yaml", "references"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if got, err := o.Within(h); err != nil || got != "references" {
		t.Errorf("Within = %q, %v; want references", got, err)
	}
}
