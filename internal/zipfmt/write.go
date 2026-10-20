// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package zipfmt

import (
	"bytes"
	"compress/flate"
	"encoding/binary"
	"errors"
	"hash"
	"hash/crc32"
	"io"
	"strconv"
)

// Method is how Write is asked to store an entry (ADR-001 D4 item 1).
type Method uint8

const (
	// Auto deflates the content and keeps the result only when it is
	// strictly shorter; otherwise the entry is stored (ADR-001 D4 item 2).
	Auto Method = iota + 1
	// Store stores the content as it is.
	Store
)

// Entry is one entry for Write: its name, its content and how to store it.
type Entry struct {
	Name    string
	Content []byte
	Method  Method
}

// Write writes entries, in the order given, as ADR-001 D1's canonical zip:
// every header field is D1's constant or is computed from the content, so
// nothing of the host enters the output, and one build writes the same
// bytes for the same entries everywhere (ADR-001 D4). It writes front to
// back and never seeks, and needs no data descriptor.
//
// Write does not judge names: a package's entries are in D1 item 7's order
// and pass ADR-001 D2, and a container output's are in ADR-005 D8's, and
// both are their callers' to establish. It refuses only what the format
// cannot carry: an empty name, a method that is neither Auto nor Store,
// more than 65,534 entries, and a size or offset beyond 0xFFFFFFFE (D3).
func Write(w io.Writer, entries []Entry) error {
	parts := make([]Part, len(entries))
	for i, e := range entries {
		parts[i] = Part{Name: e.Name, Method: e.Method, Content: func(w io.Writer) error {
			_, err := w.Write(e.Content)
			return err
		}}
	}
	_, err := WriteParts(w, parts)
	return err
}

// Part is one entry for WriteParts: its name, how to store it, and the
// function that writes its content. WriteParts calls Content twice, once
// to measure the entry and once to write it, and refuses the zip when the
// two calls write different bytes.
type Part struct {
	Name    string
	Method  Method
	Content func(w io.Writer) error
}

// Written is what WriteParts chose for one entry, whether Auto deflated
// it, and where it wrote the entry's records.
type Written struct {
	Deflated bool
	// Local and Central are the byte ranges, [start, end), of the entry's
	// local header with its stored bytes, and of its central directory
	// header, in the zip.
	Local, Central [2]int64
}

// MaxEntries is the most entries a zip WriteParts writes may hold: ADR-001
// D3's ceiling, since there is no zip64.
const MaxEntries = maxEntries

// WriteParts is Write over entries whose content is written rather than
// held: an entry is measured — its CRC, its size and, for Auto, its
// deflated size — by writing its content once, and is then written by
// writing it again behind its local header. So no entry, and no zip,
// is ever held whole in memory, and a zip can be an entry of another,
// written as it is written (ADR-005 D13 item 5).
func WriteParts(w io.Writer, parts []Part) ([]Written, error) {
	if len(parts) > maxEntries {
		return nil, errors.New("zipfmt: " + strconv.Itoa(len(parts)) + " entries is more than a zip can hold")
	}
	var (
		cd      []byte
		offset  int64
		written = make([]Written, len(parts))
		d       deflater
	)
	for i, p := range parts {
		h, err := d.measure(p)
		if err != nil {
			return nil, err
		}
		written[i].Deflated = h.method == methodDeflated
		h.offset = uint32(offset)
		local := h.appendLocal(nil)
		written[i].Local[0] = offset
		offset += int64(len(local)) + int64(h.csize)
		written[i].Local[1] = offset
		if offset > maxSize {
			return nil, errors.New("zipfmt: the entries are larger than a zip can hold")
		}
		if _, err := w.Write(local); err != nil {
			return nil, err
		}
		if err := d.emit(w, p, h); err != nil {
			return nil, err
		}
		written[i].Central[0] = int64(len(cd))
		cd = h.appendCentral(cd)
		written[i].Central[1] = int64(len(cd))
	}
	for i := range written {
		written[i].Central[0] += offset
		written[i].Central[1] += offset
	}
	if offset+int64(len(cd))+eocdLen > maxSize {
		return nil, errors.New("zipfmt: the entries are larger than a zip can hold")
	}
	end := make([]byte, eocdLen)
	le := binary.LittleEndian
	le.PutUint32(end[0:], sigEOCD)
	le.PutUint16(end[8:], uint16(len(parts)))
	le.PutUint16(end[10:], uint16(len(parts)))
	le.PutUint32(end[12:], uint32(len(cd)))
	le.PutUint32(end[16:], uint32(offset))
	if _, err := w.Write(cd); err != nil {
		return nil, err
	}
	if _, err := w.Write(end); err != nil {
		return nil, err
	}
	return written, nil
}

// counter counts the bytes written to it, and hashes them when crc is set.
type counter struct {
	n   int64
	crc hash.Hash32
}

func (c *counter) Write(b []byte) (int, error) {
	c.n += int64(len(b))
	if c.n > maxSize {
		return 0, errors.New("zipfmt: an entry is larger than a zip can hold")
	}
	if c.crc != nil {
		c.crc.Write(b)
	}
	return len(b), nil
}

// deflater is the one compressor of a zip's entries, reset for each:
// level 9's state is large, and a zip may hold tens of thousands of
// entries. A zip written as a part of another has its own.
type deflater struct{ fw *flate.Writer }

// to returns the deflater reset to write to w, which writes what a new
// one would.
func (d *deflater) to(w io.Writer) (*flate.Writer, error) {
	if d.fw == nil {
		var err error
		d.fw, err = flate.NewWriter(w, flate.BestCompression)
		return d.fw, err
	}
	d.fw.Reset(w)
	return d.fw, nil
}

// measure writes p's content once and returns its header, its offset
// unset: its CRC and size, and the method and compressed size Auto
// chooses (ADR-001 D4 item 2).
func (d *deflater) measure(p Part) (header, error) {
	switch {
	case p.Name == "":
		return header{}, errors.New("zipfmt: an entry has an empty name")
	case len(p.Name) > 0xFFFF:
		return header{}, errors.New("zipfmt: an entry's name is longer than 65,535 bytes")
	case p.Method != Auto && p.Method != Store:
		return header{}, errors.New("zipfmt: an entry has no method")
	}
	raw := &counter{crc: crc32.NewIEEE()}
	var (
		packed counter
		fw     *flate.Writer
		dst    io.Writer = raw
	)
	if p.Method == Auto {
		var err error
		if fw, err = d.to(&packed); err != nil {
			return header{}, err
		}
		dst = io.MultiWriter(raw, fw)
	}
	if err := p.Content(dst); err != nil {
		return header{}, err
	}
	h := header{name: p.Name, method: methodStored, crc: raw.crc.Sum32(), csize: uint32(raw.n), usize: uint32(raw.n)}
	if fw != nil && raw.n > 0 {
		if err := fw.Close(); err != nil {
			return header{}, err
		}
		if packed.n < raw.n {
			h.method, h.csize = methodDeflated, uint32(packed.n)
		}
	}
	return h, nil
}

// emit writes p's content again, as h's file data, and refuses content
// that is not what measure measured.
func (d *deflater) emit(w io.Writer, p Part, h header) error {
	raw := &counter{crc: crc32.NewIEEE()}
	out := &countingWriter{w: w}
	var (
		fw  *flate.Writer
		dst io.Writer
	)
	if h.method == methodDeflated {
		var err error
		if fw, err = d.to(out); err != nil {
			return err
		}
		dst = io.MultiWriter(raw, fw)
	} else {
		dst = io.MultiWriter(raw, out)
	}
	if err := p.Content(dst); err != nil {
		return err
	}
	if fw != nil {
		if err := fw.Close(); err != nil {
			return err
		}
	}
	if raw.crc.Sum32() != h.crc || uint32(raw.n) != h.usize || uint32(out.n) != h.csize {
		return errors.New("zipfmt: the content of " + strconv.Quote(p.Name) + " changed between its measure and its write")
	}
	return nil
}

// countingWriter writes to w and counts what it wrote.
type countingWriter struct {
	w io.Writer
	n int64
}

func (c *countingWriter) Write(b []byte) (int, error) {
	n, err := c.w.Write(b)
	c.n += int64(n)
	return n, err
}

// encode returns e's header, its offset unset, and its file data.
func encode(e Entry) (header, []byte, error) {
	switch {
	case e.Name == "":
		return header{}, nil, errors.New("zipfmt: an entry has an empty name")
	case len(e.Name) > 0xFFFF:
		return header{}, nil, errors.New("zipfmt: an entry's name is longer than 65,535 bytes")
	case int64(len(e.Content)) > maxSize:
		return header{}, nil, errors.New("zipfmt: an entry is larger than a package can hold")
	case e.Method != Auto && e.Method != Store:
		return header{}, nil, errors.New("zipfmt: an entry has no method")
	}
	h := header{
		name:   e.Name,
		method: methodStored,
		crc:    crc32.ChecksumIEEE(e.Content),
		csize:  uint32(len(e.Content)),
		usize:  uint32(len(e.Content)),
	}
	data := e.Content
	if e.Method == Auto && len(e.Content) > 0 {
		var buf bytes.Buffer
		fw, err := flate.NewWriter(&buf, flate.BestCompression)
		if err != nil {
			return header{}, nil, err
		}
		if _, err := fw.Write(e.Content); err != nil {
			return header{}, nil, err
		}
		if err := fw.Close(); err != nil {
			return header{}, nil, err
		}
		if buf.Len() < len(e.Content) {
			h.method, h.csize, data = methodDeflated, uint32(buf.Len()), buf.Bytes()
		}
	}
	return h, data, nil
}

// header is the fields of one entry that are not ADR-001 D1's constants.
type header struct {
	name   string
	method uint16
	crc    uint32
	csize  uint32
	usize  uint32
	offset uint32
}

// versionNeeded is D1 item 3's version needed to extract for the method.
func versionNeeded(method uint16) uint16 {
	if method == methodDeflated {
		return 20
	}
	return 10
}

// appendLocal appends h's local file header (ADR-001 D1 item 3).
func (h header) appendLocal(b []byte) []byte {
	le := binary.LittleEndian
	b = le.AppendUint32(b, sigLocal)
	b = le.AppendUint16(b, versionNeeded(h.method))
	b = le.AppendUint16(b, flagUTF8)
	b = le.AppendUint16(b, h.method)
	b = le.AppendUint16(b, dosTime)
	b = le.AppendUint16(b, dosDate)
	b = le.AppendUint32(b, h.crc)
	b = le.AppendUint32(b, h.csize)
	b = le.AppendUint32(b, h.usize)
	b = le.AppendUint16(b, uint16(len(h.name)))
	b = le.AppendUint16(b, 0)
	return append(b, h.name...)
}

// appendCentral appends h's central directory header (ADR-001 D1 item 4).
func (h header) appendCentral(b []byte) []byte {
	le := binary.LittleEndian
	b = le.AppendUint32(b, sigCentral)
	b = le.AppendUint16(b, versionMadeBy)
	b = le.AppendUint16(b, versionNeeded(h.method))
	b = le.AppendUint16(b, flagUTF8)
	b = le.AppendUint16(b, h.method)
	b = le.AppendUint16(b, dosTime)
	b = le.AppendUint16(b, dosDate)
	b = le.AppendUint32(b, h.crc)
	b = le.AppendUint32(b, h.csize)
	b = le.AppendUint32(b, h.usize)
	b = le.AppendUint16(b, uint16(len(h.name)))
	b = le.AppendUint16(b, 0) // extra field length
	b = le.AppendUint16(b, 0) // file comment length
	b = le.AppendUint16(b, 0) // disk number start
	b = le.AppendUint16(b, 0) // internal file attributes
	b = le.AppendUint32(b, externalAttrs)
	b = le.AppendUint32(b, h.offset)
	return append(b, h.name...)
}
