// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package liquidspec

import (
	"bytes"
	"io"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/kottera/metais/internal/pkgtree"
	"github.com/kottera/metais/internal/template"
)

// tree is a compile request's package: its file system as the package's
// templates, and the template it compiles. It holds nothing but templates/
// entries, and reads nothing from the host.
type tree struct {
	paths []string
	files map[string][]byte
}

// newTree returns the package of a compile request, and the name of its
// template beneath templates/. Each file of fs is the template of exactly
// its name (MTS-TPL-041); a name that cannot be an entry path (ADR-001
// D2), or that is a directory prefix of another, is no template, so a
// render tag naming it finds none (MTS-TPL-042). The compiled template is
// named "template", with as many leading "_" as keep it from naming a
// file of fs or a directory of one.
func newTree(fs map[string]string, src string) (pkgtree.Tree, string) {
	name := "template"
	for taken(fs, name) {
		name = "_" + name
	}
	files := map[string][]byte{template.Dir + name: []byte(src)}
	for n, text := range fs {
		if entryShaped(n) {
			files[template.Dir+n] = []byte(text)
		}
	}
	paths := make([]string, 0, len(files))
	for p := range files {
		paths = append(paths, p)
	}
	slices.Sort(paths)
	// In byte order a directory prefix "a" is followed by its "a/…"
	// entries, and only by entries that sort between them, which share
	// its prefix too, so it is enough to look ahead while the prefix holds.
	kept := paths[:0:0]
	for i, p := range paths {
		prefix := false
		for _, q := range paths[i+1:] {
			if !strings.HasPrefix(q, p) {
				break
			}
			if q[len(p)] == '/' {
				prefix = true
				break
			}
		}
		if prefix {
			delete(files, p)
			continue
		}
		kept = append(kept, p)
	}
	return &tree{paths: kept, files: files}, name
}

// taken reports whether name is a file of fs or a directory of one.
func taken(fs map[string]string, name string) bool {
	if _, ok := fs[name]; ok {
		return true
	}
	for n := range fs {
		if strings.HasPrefix(n, name+"/") {
			return true
		}
	}
	return false
}

// entryShaped reports whether a name beneath templates/ has ADR-001 D2's
// shape: non-empty valid UTF-8, segments separated by "/", none empty,
// "." or "..", and no "\" or ":".
func entryShaped(name string) bool {
	if name == "" || !utf8.ValidString(name) || strings.ContainsAny(name, `\:`) {
		return false
	}
	return !slices.ContainsFunc(strings.Split(name, "/"), func(s string) bool { return s == "" || s == "." || s == ".." })
}

func (t *tree) List() []string { return slices.Clone(t.paths) }

func (t *tree) Stat(path string) (pkgtree.Info, error) {
	b, ok := t.files[path]
	if !ok {
		return pkgtree.Info{}, &pkgtree.PathError{Op: "stat", Path: path, Err: pkgtree.ErrNotExist}
	}
	return pkgtree.Info{Path: path, Size: int64(len(b))}, nil
}

func (t *tree) Open(path string) (io.ReadCloser, error) {
	b, ok := t.files[path]
	if !ok {
		return nil, &pkgtree.PathError{Op: "open", Path: path, Err: pkgtree.ErrNotExist}
	}
	return io.NopCloser(bytes.NewReader(b)), nil
}
