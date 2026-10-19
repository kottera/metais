// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package loader

import (
	"bytes"
	"crypto/sha256"
	"hash"
	"io"
	"strings"
	"unicode/utf8"

	"github.com/kottera/metais/internal/digest"
	"github.com/kottera/metais/internal/manifest"
	"github.com/kottera/metais/internal/pkgtree"
	"github.com/kottera/metais/internal/schema"
)

// contents is what the open keeps of the entries as they are read, the one
// time each is decoded (ADR-010 D15 item 2): the bytes of the entries step
// 7 reads — the manifest, the compiled files, the schema files and the
// Markdown references whose titles the references index gives — and, for
// every reference, what the index says of it.
type contents struct {
	kept map[string]*bytes.Buffer
	refs map[string]*refStats
}

func newContents() *contents {
	return &contents{kept: map[string]*bytes.Buffer{}, refs: map[string]*refStats{}}
}

// sink is the writer an entry's bytes are handed to as it is decoded, or
// nil for an entry nothing of which is kept.
func (c *contents) sink(name string) io.Writer {
	var ws []io.Writer
	if keep(name) {
		b := &bytes.Buffer{}
		c.kept[name] = b
		ws = append(ws, b)
	}
	if strings.HasPrefix(name, referencesDir) {
		s := &refStats{}
		c.refs[name] = s
		ws = append(ws, s)
	}
	switch len(ws) {
	case 0:
		return nil
	case 1:
		return ws[0]
	}
	return io.MultiWriter(ws...)
}

// keep reports whether step 7 reads the entry name's bytes.
func keep(name string) bool {
	return name == manifest.ManifestFile || strings.HasPrefix(name, compiledDir) ||
		schema.IsSchemaFile(name) || (strings.HasPrefix(name, referencesDir) && isMarkdown(name))
}

// bytes returns the kept bytes of the entry name, or nil.
func (c *contents) bytes(name string) []byte {
	if b, ok := c.kept[name]; ok {
		return b.Bytes()
	}
	return nil
}

// refStats is what the references index says of one entry (ADR-008 D17
// item 1), gathered as its bytes pass: its size, whether it is text by
// ADR-008 D5's rule (valid UTF-8 with no NUL byte), and its line count.
type refStats struct {
	size    int64
	lfs     int64
	last    byte
	binary  bool
	pending []byte // the bytes of a UTF-8 sequence a write ended inside
}

func (s *refStats) Write(p []byte) (int, error) {
	n := len(p)
	if n == 0 {
		return 0, nil
	}
	s.size += int64(n)
	s.lfs += int64(bytes.Count(p, []byte{'\n'}))
	s.last = p[n-1]
	if s.binary {
		return n, nil
	}
	if bytes.IndexByte(p, 0) >= 0 {
		s.binary = true
		return n, nil
	}
	buf := p
	if len(s.pending) > 0 {
		buf = append(s.pending, p...)
		s.pending = nil
	}
	// Hold back a sequence the write may have cut, at most three bytes.
	cut := len(buf)
	for i := len(buf) - 1; i >= 0 && i >= len(buf)-3; i-- {
		if utf8.RuneStart(buf[i]) {
			if !utf8.FullRune(buf[i:]) {
				cut = i
			}
			break
		}
	}
	if !utf8.Valid(buf[:cut]) {
		s.binary = true
		return n, nil
	}
	s.pending = append([]byte(nil), buf[cut:]...)
	return n, nil
}

// text reports whether the entry is text, once all of it has passed.
func (s *refStats) text() bool {
	return !s.binary && len(s.pending) == 0
}

// lines is the entry's line count by ADR-008 D5's rule: its LFs, and one
// more when it is not empty and does not end in an LF.
func (s *refStats) lines() int64 {
	if s.size > 0 && s.last != '\n' {
		return s.lfs + 1
	}
	return s.lfs
}

func newHash() hash.Hash { return sha256.New() }

func sha256Sum(b []byte) digest.Sum { return sha256.Sum256(b) }

// memTree is a pkgtree.Tree over entries already read: the schema files the
// compiler reads, which the open has decoded once and does not decode
// again.
type memTree struct {
	names []string
	files map[string][]byte
}

var _ pkgtree.Tree = (*memTree)(nil)

func newMemTree(files map[string][]byte) *memTree {
	t := &memTree{files: files}
	for n := range files {
		t.names = append(t.names, n)
	}
	sortStrings(t.names)
	return t
}

func (t *memTree) List() []string { return append([]string(nil), t.names...) }

func (t *memTree) Stat(path string) (pkgtree.Info, error) {
	b, ok := t.files[path]
	if !ok {
		return pkgtree.Info{}, &pkgtree.PathError{Op: "stat", Path: path, Err: pkgtree.ErrNotExist}
	}
	return pkgtree.Info{Path: path, Size: int64(len(b))}, nil
}

func (t *memTree) Open(path string) (io.ReadCloser, error) {
	b, ok := t.files[path]
	if !ok {
		return nil, &pkgtree.PathError{Op: "open", Path: path, Err: pkgtree.ErrNotExist}
	}
	return io.NopCloser(bytes.NewReader(b)), nil
}
