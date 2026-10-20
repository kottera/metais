// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package loader

import (
	"io"
	"slices"

	"github.com/kottera/metais/internal/manifest"
	"github.com/kottera/metais/internal/pkgtree"
	"github.com/kottera/metais/internal/specdata"
	"github.com/kottera/metais/internal/zipfmt"
)

// Readme is the README text of the specification version spec (ADR-001
// A1 item 3): the bytes every packed package declaring it begins with, as
// README.md. Specification 1.0 defines the first text; a minor that
// defines none has the text of the latest minor before it that does.
func Readme(spec manifest.Spec) []byte {
	return specdata.Readme10
}

// contentTree is a packed package's entries as every command reads them:
// the Archive's without README.md, which is the specification's text and
// no part of the package's content (ADR-001 A1 item 5), so that a packed
// package reads as its source directory does (ADR-008 D16 item 4).
// Step 7 judges README.md from the entries and their kept bytes, not
// through the tree.
type contentTree struct{ *zipfmt.Archive }

func (t contentTree) List() []string {
	return slices.DeleteFunc(t.Archive.List(), func(n string) bool { return n == zipfmt.ReadmeName })
}

func (t contentTree) Stat(path string) (pkgtree.Info, error) {
	if path == zipfmt.ReadmeName {
		return pkgtree.Info{}, &pkgtree.PathError{Op: "stat", Path: path, Err: pkgtree.ErrNotExist}
	}
	return t.Archive.Stat(path)
}

func (t contentTree) Open(path string) (io.ReadCloser, error) {
	if path == zipfmt.ReadmeName {
		return nil, &pkgtree.PathError{Op: "open", Path: path, Err: pkgtree.ErrNotExist}
	}
	return t.Archive.Open(path)
}
