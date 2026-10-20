// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package author

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"path"
	"slices"
	"strconv"

	"github.com/kottera/metais/internal/digest"
	"github.com/kottera/metais/internal/fault"
	"github.com/kottera/metais/internal/finding"
	"github.com/kottera/metais/internal/hostfs"
	"github.com/kottera/metais/internal/limits"
	"github.com/kottera/metais/internal/loader"
	"github.com/kottera/metais/internal/manifest"
	"github.com/kottera/metais/internal/pkgtree"
	"github.com/kottera/metais/internal/zipfmt"
)

// LayoutEntries are the top-level entries of ADR-001 D5's layout that
// pack writes from a source directory, in byte order: the manifest and the
// layout trees (ADR-007 D7 item 3). pack refuses an output inside any of
// them (item 5).
var LayoutEntries = []string{"assets", "examples", "inputs", "metais.yaml", "references", "templates"}

// EntryDirs is every directory that holds one of names, entry paths,
// each once, in byte order: for a source directory that opened, every
// directory beneath its layout trees, since an empty one is refused
// (ADR-001 D5 item 6).
func EntryDirs(names []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, n := range names {
		for d := path.Dir(n); d != "." && !seen[d]; d = path.Dir(d) {
			seen[d] = true
			out = append(out, d)
		}
	}
	slices.Sort(out)
	return out
}

// Packed is a package as pack writes it from a source directory (ADR-007
// D7): its entries, README.md, the source's and .compiled/'s, and its
// package digest.
type Packed struct {
	// Entries is every entry's name, in the order written: README.md,
	// then ascending byte order (ADR-001 D1 item 7, A1 item 4).
	Entries []string
	// Digest is the package digest of the entries (ADR-001 D6).
	Digest digest.Sum

	tree    pkgtree.Tree
	sums    map[string]digest.Sum
	bytes   map[string][]byte // README.md and the compiled files, the digest file's included
	written []zipfmt.Written  // where the last Write put each entry, in the order of Entries
}

// PlanPack reads every entry of tree, a source directory that opened,
// once, and returns the package pack writes from it: the README text of
// spec, its declared version, as README.md (ADR-001 A1 item 1); those
// entries; the compiled files the open derived, compiled; and the digest
// file of them all but README.md (ADR-007 D7 item 3, ADR-001 D6 and D7, A1
// item 5). Content is packed byte for byte, never normalised, and nothing
// but the entries' names and content reaches the package (item 4).
//
// faults are the invocation's; PlanPack is fault.PackMapOrder's seam,
// which puts the entries in the order of a Go map built from them.
func PlanPack(tree pkgtree.Tree, compiled []loader.File, spec manifest.Spec, faults fault.Set) (*Packed, error) {
	p := &Packed{tree: tree, sums: map[string]digest.Sum{}, bytes: map[string][]byte{}}
	var list []digest.Entry
	for _, n := range tree.List() {
		sum, err := hashEntry(tree, n, nil)
		if err != nil {
			return nil, err
		}
		p.sums[n] = sum
		list = append(list, digest.Entry{Name: n, Sum: sum})
	}
	for _, f := range compiled {
		p.bytes[f.Name] = f.Content
		list = append(list, digest.Entry{Name: f.Name, Sum: sha256.Sum256(f.Content)})
	}
	p.bytes[digest.File] = digest.Encode(list)
	p.Digest = digest.Package(list)
	if faults.Has(fault.PackMapOrder) {
		// The control's seam (ADR-010 D13 item 2).
		p.bytes[digest.File] = mapOrdered(list, p.Digest)
	}
	for _, e := range list {
		p.Entries = append(p.Entries, e.Name)
	}
	p.Entries = append(p.Entries, digest.File)
	slices.Sort(p.Entries)
	// README.md first, before .compiled/ (ADR-001 A1 item 4).
	p.bytes[zipfmt.ReadmeName] = loader.Readme(spec)
	p.Entries = append([]string{zipfmt.ReadmeName}, p.Entries...)
	return p, nil
}

// mapOrdered is the digest file of list under fault.PackMapOrder: its
// entries' members in the order of a Go map built from them, not the
// canonical order of digest.Encode (ADR-002 D9). The zip stays one a
// reader's steps 1 to 5 accept, and its bytes change from one pack to the
// next.
func mapOrdered(list []digest.Entry, pkg digest.Sum) []byte {
	m := map[string]digest.Sum{}
	for _, e := range list {
		m[e.Name] = e.Sum
	}
	b := []byte(`{"algorithm":"` + digest.Algorithm + `","entries":{`)
	first := true
	for n, sum := range m {
		if !first {
			b = append(b, ',')
		}
		first = false
		b = strconv.AppendQuote(b, n)
		b = append(b, ':', '"')
		b = hex.AppendEncode(b, sum[:])
		b = append(b, '"')
	}
	b = append(b, `},"package":"`...)
	b = hex.AppendEncode(b, pkg[:])
	return append(b, `"}`...)
}

// Write writes the package to w through ADR-001 D4's writer: README.md
// stored (ADR-001 A1 item 1), every other entry method auto (D4 item 7). A source entry is read again for each pass the
// writer makes over it, and must hold the bytes PlanPack hashed: one that
// changed since fails the write with a *ReadError of reason
// hostfs.Changed, so the digests a package carries are always its
// entries' own. A failure to read one is a *ReadError too; a failure to
// write to w is returned as it is.
func (p *Packed) Write(w io.Writer) error {
	parts := make([]zipfmt.Part, len(p.Entries))
	for i, n := range p.Entries {
		parts[i] = zipfmt.Part{Name: n, Method: zipfmt.Auto}
		if n == zipfmt.ReadmeName {
			parts[i].Method = zipfmt.Store
		}
		if b, ok := p.bytes[n]; ok {
			parts[i].Content = func(w io.Writer) error {
				_, err := w.Write(b)
				return err
			}
			continue
		}
		parts[i].Content = func(w io.Writer) error {
			sum, err := hashEntry(p.tree, n, w)
			switch {
			case err != nil:
				return err
			case sum != p.sums[n]:
				return &ReadError{Reason: hostfs.Changed}
			}
			return nil
		}
	}
	written, err := zipfmt.WriteParts(w, parts)
	p.written = written
	return err
}

// EntryAt is the entry whose records, as the last Write wrote them, hold
// the byte at offset: its local header and stored bytes, or its central
// directory header. A reader that refuses a second end record cannot know
// which entry holds the signature; pack can, and that entry is the one an
// author changes, since packing the same source again writes the same
// bytes (R1-014).
func (p *Packed) EntryAt(offset int64) (string, bool) {
	for i, w := range p.written {
		if offset >= w.Local[0] && offset < w.Local[1] || offset >= w.Central[0] && offset < w.Central[1] {
			return p.Entries[i], true
		}
	}
	return "", false
}

// ReadError is a source entry pack could not read, or one that changed
// between two reads of it: the source directory, not the output, failed.
type ReadError struct {
	Reason hostfs.Reason
}

func (e *ReadError) Error() string { return "author: reading a source entry: " + string(e.Reason) }

// VerifyPacked reads the package at r, size bytes, as every reader reads
// one, ADR-001 D8's steps 1 to 5, under lim, and returns what it finds: a
// package whose bytes breach a limit only they can show, such as
// entry-ratio or package-size, which check cannot judge without the
// encoder, is one pack refuses rather than writes (ADR-007 D2 step 3,
// MTS-PKG-038). An error is r's.
func VerifyPacked(r io.ReaderAt, size int64, lim limits.Set) ([]finding.Finding, error) {
	_, fs, err := zipfmt.Open(r, size, lim)
	return fs, err
}

// hashEntry reads the entry n of tree, writing its bytes to w when w is
// not nil, and returns their SHA-256. A failure to read is a *ReadError;
// a failure to write to w is returned as it is.
func hashEntry(tree pkgtree.Tree, n string, w io.Writer) (digest.Sum, error) {
	r, err := tree.Open(n)
	if err != nil {
		return digest.Sum{}, &ReadError{Reason: hostfs.ReasonOf(err)}
	}
	defer r.Close()
	h := sha256.New()
	dst := io.Writer(h)
	if w != nil {
		dst = io.MultiWriter(h, w)
	}
	buf := make([]byte, 32<<10)
	for {
		k, err := r.Read(buf)
		if k > 0 {
			if _, werr := dst.Write(buf[:k]); werr != nil {
				return digest.Sum{}, werr
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return digest.Sum{}, &ReadError{Reason: hostfs.ReasonOf(err)}
		}
	}
	var sum digest.Sum
	copy(sum[:], h.Sum(nil))
	return sum, nil
}
