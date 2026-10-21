// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package render

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"io"
	pathpkg "path"
	"strings"

	"github.com/kottera/metais/internal/manifest"
	"github.com/kottera/metais/internal/pkgtree"
	"github.com/kottera/metais/internal/template"
)

// File is one file a render plans (ADR-005 D13 phase 1): its
// declaration, its path within its tree, where in the manifest the path
// comes from, the evaluation of a fan-out it is, what it is written from,
// and its bytes once rendered. A file copied from an asset holds no
// bytes: they are streamed from the package as they are written (D13 item
// 4).
type File struct {
	decl manifest.Output
	path string
	ptr  string
	// template is the template a template output or a mirror renders the
	// file from, beneath templates/.
	template string
	// asset is the entry an asset declaration copies the file from, and
	// tree the package's tree it is read from.
	asset string
	tree  pkgtree.Tree
	// literal is set for a path the manifest writes, or defaults, rather
	// than one a path template rendered.
	literal bool
	fan     *template.Fan
	bytes   []byte
	// owner is the id findings about the file name: its declaration's,
	// or, for a container's entry that has none, the nearest enclosing
	// declaration's.
	owner string
	// in is the path of the container the file is an entry of, through
	// every enclosing container, joined by "/", or "" at the top level.
	in string
	// entries are a container's files, in the container's order (ADR-005
	// D8 item 6).
	entries []*File
	// tool marks the mimetype entry a profile writes itself (ADR-005 D9
	// item 4).
	tool bool
	// failed marks a file that did not render, and a container one of
	// whose entries did not.
	failed bool
	// size and digest are a container's zip's, measured in phase 4, or an
	// asset's entry's, its digest taken as it is first read; deflated is
	// whether its container deflates an entry.
	size     int64
	digest   string
	deflated bool
}

// Path is f's path within its tree: the output directory's, or its
// container's.
func (pf *File) Path() string { return pf.path }

// Output is the id of f's declaration, or "" for one with none and for a
// profile's mimetype, which no declaration writes (ADR-005 D16 item 2).
func (pf *File) Output() string {
	if pf.tool {
		return ""
	}
	return pf.decl.ID
}

// Executable reports whether f's declaration says executable: true.
func (pf *File) Executable() bool { return pf.decl.Executable }

// Entries are a container's files, in the container's order.
func (pf *File) Entries() []*File { return pf.entries }

// Deflated reports whether f's container deflates it, once measured.
func (pf *File) Deflated() bool { return pf.deflated }

// Stored reports whether f, as a container's entry, is always stored: its
// declaration says stored: true, or it is a profile's mimetype (ADR-005 D8
// item 7).
func (pf *File) Stored() bool { return pf.tool || pf.decl.Stored }

// Item is the 0-based index of the fan-out evaluation pf is, or nil.
func (pf *File) Item() *int {
	if pf.fan == nil {
		return nil
	}
	i := pf.fan.Index()
	return &i
}

// IsContainer reports whether pf is a container, whose bytes are its
// entries' zip.
func (pf *File) IsContainer() bool { return pf.decl.Form == manifest.OutputContainer && !pf.tool }

// Source is the record's source of pf (ADR-005 D16 item 2): template for
// a file a template output or a mirror renders, asset, container, or
// profile for the mimetype entry a profile writes.
func (pf *File) Source() string {
	switch {
	case pf.tool:
		return "profile"
	case pf.asset != "":
		return "asset"
	case pf.IsContainer():
		return "container"
	}
	return "template"
}

// Kind is the kind of pf (ADR-005 D7), or nil for an asset, a container
// and a profile's mimetype, which have none (D7 item 3).
func (pf *File) Kind() *string {
	if pf.asset != "" || pf.IsContainer() || pf.tool {
		return nil
	}
	k := kindOf(pf.path, pf.decl.Kind)
	return &k
}

// Measured is pf's size and digest: a container's zip's, an asset's
// entry's, or its bytes'.
func (pf *File) Measured() (int64, string) {
	if pf.IsContainer() || pf.asset != "" {
		return pf.size, pf.digest
	}
	sum := sha256.Sum256(pf.bytes)
	return int64(len(pf.bytes)), "sha256:" + hex.EncodeToString(sum[:])
}

// Open is the reader of pf's bytes as they are written: an asset's are
// streamed from the package, its digest taken on the first whole read; a
// container's zip is written into a pipe as it is read; and done reports
// the error of its writing, once the reader is finished with.
func (pf *File) Open() (r io.Reader, done func(read error) error) {
	if pf.asset != "" {
		rc, err := pf.tree.Open(pf.asset)
		if err != nil {
			panic("render: opening " + pf.asset + ", which the plan found: " + err.Error())
		}
		h := sha256.New()
		return io.TeeReader(rc, h), func(read error) error {
			rc.Close()
			if read == nil && pf.digest == "" {
				pf.digest = "sha256:" + hex.EncodeToString(h.Sum(nil))
			}
			return nil
		}
	}
	if !pf.IsContainer() {
		return bytes.NewReader(pf.bytes), func(error) error { return nil }
	}
	pr, pw := io.Pipe()
	errc := make(chan error, 1)
	go func() {
		_, err := writeContainer(pw, pf)
		pw.CloseWithError(err)
		errc <- err
	}()
	return pr, func(read error) error {
		// A reader that stopped early unblocks the writer.
		pr.CloseWithError(read)
		if err := <-errc; err != nil && read == nil {
			return err
		}
		return nil
	}
}

// kindOf is a rendered file's kind (ADR-005 D7): the declaration's kind
// when it gives one, otherwise from the path's final extension, ASCII
// case-insensitively, then its whole file name.
func kindOf(p, declared string) string {
	if declared != "" {
		return declared
	}
	name := pathpkg.Base(p)
	if i := strings.LastIndexByte(name, '.'); i >= 0 {
		if k, ok := kindExtensions[strings.ToLower(name[i+1:])]; ok {
			return k
		}
	}
	if k, ok := kindNames[name]; ok {
		return k
	}
	return "text"
}

// kindExtensions and kindNames are ADR-005 D7 item 1's table.
var (
	kindExtensions = func() map[string]string {
		m := map[string]string{}
		for kind, exts := range map[string]string{
			"html":     "html htm",
			"xml":      "xml xhtml svg rels opf ncx xsd xsl xslt plist",
			"markdown": "md markdown",
			"json":     "json geojson webmanifest",
			"yaml":     "yaml yml",
			"csv":      "csv",
			"css":      "css scss less",
			"hash":     "sh bash zsh py rb pl r toml ps1 properties tf",
			"slash":    "js mjs cjs ts tsx jsx go c h cc cpp hpp java kt kts swift rs cs scala dart jsonc proto groovy gradle",
			"sql":      "sql",
		} {
			for _, e := range strings.Fields(exts) {
				m[e] = kind
			}
		}
		return m
	}()
	kindNames = map[string]string{
		"Makefile": "hash", "Dockerfile": "hash", "Containerfile": "hash", ".gitignore": "hash",
		".gitattributes": "hash", ".dockerignore": "hash", ".env": "hash",
	}
)
