// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package zipfmt

import (
	"errors"
	"io"
)

// source is the package file as the reader sees it: an io.ReaderAt of a
// known size, whose final bytes are held in memory once step 2 has read
// them. Every read of the tail is served from memory, so reading the end
// record, then the central directory and the entries, reads each byte of
// the file from r once.
type source struct {
	r         io.ReaderAt
	size      int64
	tail      []byte
	tailStart int64
}

// errShort is a ReaderAt that ended inside the size it was opened with:
// the file changed, or the size was wrong. Neither is the package's
// finding to report.
var errShort = errors.New("zipfmt: the package file is shorter than its size")

// readAt fills p from the file at off. The caller has checked that the
// range lies inside the file.
func (s *source) readAt(p []byte, off int64) error {
	if off < 0 || off+int64(len(p)) > s.size {
		return errShort
	}
	if s.tail != nil && off+int64(len(p)) > s.tailStart {
		cut := max(s.tailStart-off, 0)
		copy(p[cut:], s.tail[off+cut-s.tailStart:])
		p = p[:cut]
	}
	if len(p) == 0 {
		return nil
	}
	n, err := s.r.ReadAt(p, off)
	if n == len(p) {
		return nil
	}
	if err == nil || err == io.EOF {
		return errShort
	}
	return err
}

// region reads the bytes of [start, end) of the file front to back, in
// chunks, as an io.ByteReader and io.Reader: the decoder of one entry's
// data reads through it, and it reads no byte outside its range. consumed
// counts the bytes it has handed out.
type region struct {
	s         *source
	next, end int64
	buf       []byte
	pos       int
	consumed  int64
	err       error
}

const chunk = 32 << 10

func newRegion(s *source, start, end int64) *region {
	return &region{s: s, next: start, end: end}
}

func (r *region) fill() bool {
	if r.err != nil {
		return false
	}
	if r.next >= r.end {
		r.err = io.EOF
		return false
	}
	n := min(int64(chunk), r.end-r.next)
	if int64(cap(r.buf)) < n {
		r.buf = make([]byte, n)
	}
	r.buf = r.buf[:n]
	if err := r.s.readAt(r.buf, r.next); err != nil {
		r.err = err
		return false
	}
	r.next += n
	r.pos = 0
	return true
}

func (r *region) ReadByte() (byte, error) {
	if r.pos >= len(r.buf) && !r.fill() {
		return 0, r.err
	}
	c := r.buf[r.pos]
	r.pos++
	r.consumed++
	return c, nil
}

func (r *region) Read(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	if r.pos >= len(r.buf) && !r.fill() {
		return 0, r.err
	}
	n := copy(p, r.buf[r.pos:])
	r.pos += n
	r.consumed += int64(n)
	return n, nil
}

// readFull reads exactly len(p) bytes, or reports io.ErrUnexpectedEOF when
// the region ends first.
func (r *region) readFull(p []byte) error {
	_, err := io.ReadFull(r, p)
	if err == io.EOF {
		return io.ErrUnexpectedEOF
	}
	return err
}

// skip discards n bytes without handing them to anyone.
func (r *region) skip(n int64) error {
	_, err := io.CopyN(io.Discard, r, n)
	if err == io.EOF {
		return io.ErrUnexpectedEOF
	}
	return err
}

// remaining is how many bytes of the region are not yet handed out.
func (r *region) remaining() int64 {
	return r.end - r.next + int64(len(r.buf)-r.pos)
}
