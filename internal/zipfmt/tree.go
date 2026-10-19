// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package zipfmt

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"io"
	"slices"

	"github.com/kottera/metais/internal/pkgtree"
)

// Archive is a packed package that passed ADR-001 D8's steps 1 to 5: the
// packed pkgtree.Tree. It reads its entries from the io.ReaderAt Open was
// given, which the caller keeps open for as long as it uses the Archive.
type Archive struct {
	s       *source
	entries []entry
	index   map[string]int
	decodes int // entries decoded, at open and since; tests read it
}

var _ pkgtree.Tree = (*Archive)(nil)

// EntryInfo is what step 5 established about one entry: its name, its
// uncompressed size and the SHA-256 of its content, which the loader
// checks against the digest file (ADR-001 D8 step 6).
type EntryInfo struct {
	Name   string
	Size   int64
	SHA256 [sha256.Size]byte
}

// Entries returns every entry, in the file's order, which is ascending
// byte order of the names.
func (a *Archive) Entries() []EntryInfo {
	out := make([]EntryInfo, len(a.entries))
	for i, e := range a.entries {
		out[i] = EntryInfo{Name: e.name, Size: int64(e.usize), SHA256: e.sum}
	}
	return out
}

// Decodes returns how many times the Archive has decoded an entry: once
// per entry at open, and once per reader read since. It is how a reader
// holds itself to ADR-010 D15 item 2's budget.
func (a *Archive) Decodes() int { return a.decodes }

// List returns every entry path, in ascending byte order.
func (a *Archive) List() []string {
	out := make([]string, len(a.entries))
	for i, e := range a.entries {
		out[i] = e.name
	}
	return out
}

// Stat returns the entry at path, looked up exactly.
func (a *Archive) Stat(path string) (pkgtree.Info, error) {
	i, ok := a.index[path]
	if !ok {
		return pkgtree.Info{}, &pkgtree.PathError{Op: "stat", Path: path, Err: pkgtree.ErrNotExist}
	}
	return pkgtree.Info{Path: path, Size: int64(a.entries[i].usize)}, nil
}

// Open returns a reader of the entry at path, looked up exactly. The entry
// is decoded again on the first Read, into memory, within its size, which
// step 3 held within entry-size; when its bytes are not those step 5
// established, Read returns ErrChanged and none of them.
func (a *Archive) Open(path string) (io.ReadCloser, error) {
	i, ok := a.index[path]
	if !ok {
		return nil, &pkgtree.PathError{Op: "open", Path: path, Err: pkgtree.ErrNotExist}
	}
	return &entryReader{a: a, e: &a.entries[i]}, nil
}

// ErrChanged is the error a reader of an entry returns when the package
// file no longer holds the bytes Open verified.
var ErrChanged = errors.New("zipfmt: the package file changed after it was opened")

type entryReader struct {
	a    *Archive
	e    *entry
	r    *bytes.Reader
	err  error
	done bool
}

func (r *entryReader) Read(p []byte) (int, error) {
	if !r.done {
		r.done = true
		r.r, r.err = r.a.read(r.e)
	}
	if r.err != nil {
		return 0, r.err
	}
	return r.r.Read(p)
}

func (r *entryReader) Close() error { return nil }

// read decodes e again and returns its content when it is what step 5
// established.
func (a *Archive) read(e *entry) (*bytes.Reader, error) {
	a.decodes++
	var buf bytes.Buffer
	buf.Grow(int(e.usize))
	h := sha256.New()
	_, problem, err := decodeData(a.s, e, io.MultiWriter(&buf, h))
	if err != nil {
		return nil, err
	}
	if problem != "" || !slices.Equal(h.Sum(nil), e.sum[:]) {
		return nil, ErrChanged
	}
	return bytes.NewReader(buf.Bytes()), nil
}
