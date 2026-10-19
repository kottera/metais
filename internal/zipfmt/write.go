// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package zipfmt

import (
	"bytes"
	"compress/flate"
	"encoding/binary"
	"errors"
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
// back and never seeks. Each entry is compressed in memory before its
// local header is written, so no data descriptor is needed.
//
// Write does not judge names: a package's entries are in D1 item 7's order
// and pass ADR-001 D2, and a container output's are in ADR-005 D8's, and
// both are their callers' to establish. It refuses only what the format
// cannot carry: an empty name, a method that is neither Auto nor Store,
// more than 65,534 entries, and a size or offset beyond 0xFFFFFFFE (D3).
func Write(w io.Writer, entries []Entry) error {
	if len(entries) > maxEntries {
		return errors.New("zipfmt: " + strconv.Itoa(len(entries)) + " entries is more than a package can hold")
	}
	var (
		cd     []byte
		offset int64
	)
	for _, e := range entries {
		h, data, err := encode(e)
		if err != nil {
			return err
		}
		h.offset = uint32(offset)
		local := h.appendLocal(nil)
		offset += int64(len(local)) + int64(len(data))
		if offset > maxSize {
			return errors.New("zipfmt: the entries are larger than a package can hold")
		}
		if _, err := w.Write(local); err != nil {
			return err
		}
		if _, err := w.Write(data); err != nil {
			return err
		}
		cd = h.appendCentral(cd)
	}
	if offset+int64(len(cd))+eocdLen > maxSize {
		return errors.New("zipfmt: the entries are larger than a package can hold")
	}
	end := make([]byte, eocdLen)
	le := binary.LittleEndian
	le.PutUint32(end[0:], sigEOCD)
	le.PutUint16(end[8:], uint16(len(entries)))
	le.PutUint16(end[10:], uint16(len(entries)))
	le.PutUint32(end[12:], uint32(len(cd)))
	le.PutUint32(end[16:], uint32(offset))
	if _, err := w.Write(cd); err != nil {
		return err
	}
	_, err := w.Write(end)
	return err
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
