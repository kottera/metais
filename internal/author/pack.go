// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package author

import (
	"crypto/sha256"
	"io"
	"slices"

	"github.com/kottera/metais/internal/digest"
	"github.com/kottera/metais/internal/fault"
	"github.com/kottera/metais/internal/hostfs"
	"github.com/kottera/metais/internal/loader"
	"github.com/kottera/metais/internal/pkgtree"
	"github.com/kottera/metais/internal/zipfmt"
)

// LayoutEntries are the top-level entries of ADR-001 D5's layout that
// pack writes from a source directory, in byte order: the manifest and the
// layout trees (ADR-007 D7 item 3). pack refuses an output inside any of
// them (item 5).
var LayoutEntries = []string{"assets", "examples", "inputs", "metais.yaml", "references", "templates"}

// Packed is a package as pack writes it from a source directory (ADR-007
// D7): its entries, the source's and .compiled/'s, and its package digest.
type Packed struct {
	// Entries is every entry's name, in the order written: ascending
	// byte order (ADR-001 D1 item 7).
	Entries []string
	// Digest is the package digest of the entries (ADR-001 D6).
	Digest digest.Sum

	tree  pkgtree.Tree
	sums  map[string]digest.Sum
	bytes map[string][]byte // the compiled files, the digest file's included
}

// PlanPack reads every entry of tree, a source directory that opened,
// once, and returns the package pack writes from it: those entries, the
// compiled files the open derived, compiled, and the digest file of them
// all (ADR-007 D7 item 3, ADR-001 D6 and D7). Content is packed byte for
// byte, never normalised, and nothing but the entries' names and content
// reaches the package (item 4).
//
// faults are the invocation's; PlanPack is fault.PackMapOrder's seam,
// which puts the entries in the order of a Go map built from them.
func PlanPack(tree pkgtree.Tree, compiled []loader.File, faults fault.Set) (*Packed, error) {
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
	for _, e := range list {
		p.Entries = append(p.Entries, e.Name)
	}
	p.Entries = append(p.Entries, digest.File)
	slices.Sort(p.Entries)
	if faults.Has(fault.PackMapOrder) {
		// The control's seam (ADR-010 D13 item 2).
		set := map[string]bool{}
		for _, e := range p.Entries {
			set[e] = true
		}
		p.Entries = p.Entries[:0]
		for e := range set {
			p.Entries = append(p.Entries, e)
		}
	}
	return p, nil
}

// Write writes the package to w through ADR-001 D4's writer, every entry
// method auto (item 7). A source entry is read again for each pass the
// writer makes over it, and must hold the bytes PlanPack hashed: one that
// changed since fails the write with a *ReadError of reason
// hostfs.Changed, so the digests a package carries are always its
// entries' own. A failure to read one is a *ReadError too; a failure to
// write to w is returned as it is.
func (p *Packed) Write(w io.Writer) error {
	parts := make([]zipfmt.Part, len(p.Entries))
	for i, n := range p.Entries {
		parts[i] = zipfmt.Part{Name: n, Method: zipfmt.Auto}
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
	_, err := zipfmt.WriteParts(w, parts)
	return err
}

// ReadError is a source entry pack could not read, or one that changed
// between two reads of it: the source directory, not the output, failed.
type ReadError struct {
	Reason hostfs.Reason
}

func (e *ReadError) Error() string { return "author: reading a source entry: " + string(e.Reason) }

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
