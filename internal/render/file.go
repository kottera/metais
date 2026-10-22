// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package render

import (
	"io"
	pathpkg "path"
	"strings"

	"metais.org/metais/internal/manifest"
	"metais.org/metais/internal/pkgtree"
	"metais.org/metais/internal/template"
	"metais.org/metais/internal/zipfmt"
)

// File is one file a render plans (ADR-005 D13 phase 1): its
// declaration, its path within its tree, where in the manifest the path
// comes from, the evaluation of a fan-out it is, the producer its form
// set, and its bytes once produced (ADR-010 A7 item 3). Only its producer
// sets its bytes; a file copied from an asset holds none: they are
// streamed from the package as they are written (D13 item 4).
type File struct {
	decl manifest.Output
	path string
	ptr  string
	// producer is where the file's bytes come from (ADR-010 A7 item 5).
	producer Producer
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
	// failed marks a file that did not render, and a container one of
	// whose entries did not.
	failed bool
	// size and digest are the file's, once its producer has measured it:
	// an asset's digest is taken as its entry is first read; deflated is
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
func (pf *File) Output() string { return pf.decl.ID }

// Executable reports whether f's declaration says executable: true.
func (pf *File) Executable() bool { return pf.decl.Executable }

// Entries are a container's files, in the container's order.
func (pf *File) Entries() []*File { return pf.entries }

// Deflated reports whether f's container deflates it, once measured.
func (pf *File) Deflated() bool { return pf.deflated }

// Stored reports whether f, as a container's entry, is always stored: its
// declaration says stored: true, or it is a profile's mimetype (ADR-005 D8
// item 7).
func (pf *File) Stored() bool { return pf.producer.Method(pf) == zipfmt.Store }

// Item is the 0-based index of the fan-out evaluation pf is, or nil.
func (pf *File) Item() *int {
	if pf.fan == nil {
		return nil
	}
	i := pf.fan.Index()
	return &i
}

// IsContainer reports whether pf is a container, whose bytes its producer
// assembles from its entries.
func (pf *File) IsContainer() bool {
	_, ok := pf.producer.(assembled)
	return ok
}

// Source is the record's source of pf (ADR-005 D16 item 2): template for
// a file a template output or a mirror renders, asset, container, or
// profile for the mimetype entry a profile writes.
func (pf *File) Source() string {
	source, _ := pf.producer.Record()
	return source
}

// Kind is the kind of pf (ADR-005 D7), or nil for an asset, a container
// and a profile's mimetype, which have none (D7 item 3).
func (pf *File) Kind() *string {
	if _, hasKind := pf.producer.Record(); !hasKind {
		return nil
	}
	k := kindOf(pf.path, pf.decl.Kind)
	return &k
}

// Measured is pf's size and digest, as its producer measured them.
func (pf *File) Measured() (int64, string) { return pf.size, pf.digest }

// Open is the reader of pf's bytes as they are written, from its
// producer, and done reports the error of their writing, once the reader
// is finished with.
func (pf *File) Open() (r io.Reader, done func(read error) error) { return pf.producer.Open(pf) }

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
