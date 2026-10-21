// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package render

import (
	"crypto/sha256"
	"encoding/hex"
	"io"

	"github.com/kottera/metais/internal/zipfmt"
)

// measure measures the container pf, and every container among its
// entries, when every entry rendered: the size and digest of the zip it
// writes, and which of its entries it deflates. It writes the zip to
// nothing but a hash, so no container is held whole in memory.
func measure(pf *File) bool {
	if len(pf.entries) > zipfmt.MaxEntries {
		// Refused in planning (D8 item 8); a zip cannot hold them.
		pf.failed = true
		return false
	}
	for _, e := range pf.entries {
		if e.failed || e.IsContainer() && !measure(e) {
			pf.failed = true
			return false
		}
	}
	h := sha256.New()
	n := &byteCount{}
	written, err := writeContainer(io.MultiWriter(h, n), pf)
	if err != nil {
		panic("render: a planned container does not write: " + err.Error())
	}
	for i, w := range written {
		pf.entries[i].deflated = w.Deflated
	}
	pf.size, pf.digest = n.n, "sha256:"+hex.EncodeToString(h.Sum(nil))
	return true
}

// byteCount counts the bytes written to it.
type byteCount struct{ n int64 }

func (c *byteCount) Write(b []byte) (int, error) {
	c.n += int64(len(b))
	return len(b), nil
}

// writeContainer writes the zip of the container pf to w, assembling it as
// it is written (ADR-005 D8, D13 item 5): each entry's content is its
// rendered bytes, or the zip of a container entry, itself written as it
// is written. stored: true and a profile's mimetype are stored; every
// other entry is ADR-001 D4's auto (D8 item 7).
func writeContainer(w io.Writer, pf *File) ([]zipfmt.Written, error) {
	parts := make([]zipfmt.Part, len(pf.entries))
	for i, e := range pf.entries {
		method := zipfmt.Auto
		if e.tool || e.decl.Stored {
			method = zipfmt.Store
		}
		parts[i] = zipfmt.Part{Name: e.path, Method: method, Content: func(w io.Writer) error {
			if e.IsContainer() {
				_, err := writeContainer(w, e)
				return err
			}
			r, done := e.Open()
			_, err := io.Copy(w, r)
			return done(err)
		}}
	}
	return zipfmt.WriteParts(w, parts)
}
