// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package render

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"slices"

	"metais.org/metais/internal/finding"
	"metais.org/metais/internal/limits"
	"metais.org/metais/internal/zipfmt"
)

// assembled is the producer of a container (ADR-005 D8): its bytes are
// the zip of its entries, measured into a hash in phase 4 and written as
// it is read in phase 5 (D13 item 5), never held whole.
type assembled struct{}

// Record is a container's: it has no kind (D7 item 3).
func (assembled) Record() (string, bool) { return "container", false }

// Reach is none: its entries reach their own.
func (assembled) Reach(*File) []string { return nil }

// Produce produces f's entries, each by its producer, in the container's
// order, then measures f when every one of them produced. A top-level
// container is held to output-size, its finding reported once every file
// has been produced.
func (assembled) Produce(c *Producing, f *File) []finding.Finding {
	var fs []finding.Finding
	for _, e := range f.entries {
		fs = append(fs, e.producer.Produce(c, e)...)
	}
	if !measure(f) || f.in != "" {
		return fs
	}
	if f.size > c.lim.Value(limits.OutputSize) {
		loc := finding.Location{Origin: finding.OriginOutput, File: f.path}
		c.measured = append(c.measured, c.lim.ExceededOutput(limits.OutputSize, finding.Subject{Kind: finding.SubjectOutput, Name: f.owner}, loc, f.size, true))
	}
	return fs
}

// Open is the reader of f's zip, written into a pipe as it is read by
// the render's one goroutine (ADR-010 A6 item 1), and done reports the
// error of its writing.
func (assembled) Open(f *File) (io.Reader, func(error) error) {
	pr, pw := io.Pipe()
	errc := make(chan error, 1)
	go func() {
		_, err := writeContainer(pw, f)
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

func (assembled) Method(f *File) zipfmt.Method { return declaredMethod(f) }

// measure measures the container pf, whose entries have been produced,
// when every one of them produced, every container among them measured:
// the size and digest of the zip it writes, and which of its entries it
// deflates. It writes the zip to nothing but a hash, so no container is
// held whole in memory.
func measure(pf *File) bool {
	if len(pf.entries) > zipfmt.MaxEntries {
		// Refused in planning (D8 item 8); a zip cannot hold them.
		pf.failed = true
		return false
	}
	if slices.ContainsFunc(pf.entries, func(e *File) bool { return e.failed }) {
		pf.failed = true
		return false
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
// producer's, a container entry's zip itself written in place as it is
// written, and each is written by its producer's method (D8 item 7).
func writeContainer(w io.Writer, pf *File) ([]zipfmt.Written, error) {
	parts := make([]zipfmt.Part, len(pf.entries))
	for i, e := range pf.entries {
		parts[i] = zipfmt.Part{Name: e.path, Method: e.producer.Method(e), Content: func(w io.Writer) error {
			if _, nested := e.producer.(assembled); nested {
				_, err := writeContainer(w, e)
				return err
			}
			r, done := e.producer.Open(e)
			_, err := io.Copy(w, r)
			return done(err)
		}}
	}
	return zipfmt.WriteParts(w, parts)
}
