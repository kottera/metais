// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package inputs

import (
	"errors"
	"io"
	"strings"

	"github.com/kottera/metais/internal/pkgtree"
)

// ReadPackageDir reads the directory dir of the package tree t as a data
// directory (ADR-003 D16): an example's inputs/, which test reads in place,
// inside a packed package as in a source directory (ADR-007 D5 item 2.1,
// D6). dir is an entry path ending "/". It reads as ReadDataDir does, every
// finding located with origin example at the package entry it is about,
// such as examples/basic/inputs/vendor.yaml (ADR-006 D4 item 4). A tree
// holds only regular files and the directories above them, so no entry is
// a link or a special file. A dir the tree holds nothing beneath is an
// example that supplies nothing, and gives the supply nothing.
func (s *Supply) ReadPackageDir(t pkgtree.Tree, dir string) {
	if s.dataDir {
		panic("inputs: a supply's data directory is read once")
	}
	s.dataDir = true
	s.locPrefix = dir
	var names []string
	for _, n := range t.List() {
		if rest, ok := strings.CutPrefix(n, dir); ok {
			names = append(names, rest)
		}
	}
	src := dirSource{
		list: func(name string, max int) ([]dirEntry, string) {
			prefix := ""
			if name != "." {
				prefix = name + "/"
			}
			var out []dirEntry
			for _, n := range names {
				rest, ok := strings.CutPrefix(n, prefix)
				if !ok {
					continue
				}
				child, _, isDir := strings.Cut(rest, "/")
				if isDir {
					if len(out) > 0 && out[len(out)-1].name == child {
						continue
					}
					out = append(out, dirEntry{name: child, kind: entryDir})
				} else {
					info, err := t.Stat(dir + n)
					if err != nil {
						return nil, readIOError
					}
					out = append(out, dirEntry{name: child, kind: entryFile, size: info.Size})
				}
				if len(out) > max {
					return nil, readTooLarge
				}
			}
			return out, ""
		},
		read: func(name string, limit int64) ([]byte, string) {
			r, err := t.Open(dir + name)
			if err != nil {
				return nil, readIOError
			}
			defer r.Close()
			b, err := io.ReadAll(io.LimitReader(r, limit+1))
			switch {
			case err != nil && !errors.Is(err, io.EOF):
				return nil, readIOError
			case int64(len(b)) > limit:
				return nil, readTooLarge
			}
			return b, ""
		},
	}
	// The top level is bounded by the package's entries limit, which the
	// tree was opened under; a collection's listing is bounded as a data
	// directory's is.
	top, _ := src.list(".", len(names))
	s.readEntries(src, top)
}
