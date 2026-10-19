// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package zipfmt

import (
	"bytes"
	"compress/flate"
	"encoding/binary"
	"hash/crc32"
)

// This file builds zips byte by byte, with every field of every record
// under the test's control, so a test can break one rule of the profile at
// a time. A rawZip with no changes is exactly what Write writes.

// rawHeader is every field of a local or central directory header.
type rawHeader struct {
	madeBy, needed, flags, method, time, date uint16
	crc, csize, usize                         uint32
	nameLen, extraLen, commentLen             uint16
	disk, internal                            uint16
	external, offset                          uint32
}

// rawEntry is one entry. Its headers start from the profile's values for
// its content; local and central change them.
type rawEntry struct {
	name    string
	content []byte
	deflate bool
	data    []byte // the file data, when set; otherwise content, deflated or not

	localName           string // the local header's name, when set
	localExtra, cdExtra []byte
	cdComment           []byte
	local, central      func(*rawHeader)
	gapAfter            []byte // bytes between this entry and the next record
}

// rawEOCD is every field of the end record.
type rawEOCD struct {
	disk, cdDisk, onDisk, total uint16
	cdSize, cdOff               uint32
	commentLen                  uint16
}

type rawZip struct {
	prefix  []byte
	entries []rawEntry
	cdTail  []byte // bytes after the last central directory header
	eocd    func(*rawEOCD)
	comment []byte // after the end record, as its comment
	suffix  []byte // after the comment
}

func deflated(b []byte) []byte {
	var buf bytes.Buffer
	w, _ := flate.NewWriter(&buf, flate.BestCompression)
	w.Write(b)
	w.Close()
	return buf.Bytes()
}

func (z rawZip) bytes() []byte {
	le := binary.LittleEndian
	out := append([]byte{}, z.prefix...)
	var cd []byte
	for _, e := range z.entries {
		data := e.content
		h := rawHeader{madeBy: versionMadeBy, needed: 10, flags: flagUTF8, date: dosDate,
			crc: crc32.ChecksumIEEE(e.content), usize: uint32(len(e.content)), external: externalAttrs}
		if e.deflate {
			data, h.method, h.needed = deflated(e.content), methodDeflated, 20
		}
		if e.data != nil {
			data = e.data
		}
		h.csize = uint32(len(data))
		h.nameLen = uint16(len(e.name))
		h.offset = uint32(len(out))
		lh, ch := h, h
		lname := e.name
		if e.localName != "" {
			lname = e.localName
		}
		lh.nameLen = uint16(len(lname))
		lh.extraLen = uint16(len(e.localExtra))
		ch.extraLen = uint16(len(e.cdExtra))
		ch.commentLen = uint16(len(e.cdComment))
		if e.local != nil {
			e.local(&lh)
		}
		if e.central != nil {
			e.central(&ch)
		}
		out = le.AppendUint32(out, sigLocal)
		for _, v := range []uint16{lh.needed, lh.flags, lh.method, lh.time, lh.date} {
			out = le.AppendUint16(out, v)
		}
		out = le.AppendUint32(out, lh.crc)
		out = le.AppendUint32(out, lh.csize)
		out = le.AppendUint32(out, lh.usize)
		out = le.AppendUint16(out, lh.nameLen)
		out = le.AppendUint16(out, lh.extraLen)
		out = append(out, lname...)
		out = append(out, e.localExtra...)
		out = append(out, data...)
		out = append(out, e.gapAfter...)

		cd = le.AppendUint32(cd, sigCentral)
		for _, v := range []uint16{ch.madeBy, ch.needed, ch.flags, ch.method, ch.time, ch.date} {
			cd = le.AppendUint16(cd, v)
		}
		cd = le.AppendUint32(cd, ch.crc)
		cd = le.AppendUint32(cd, ch.csize)
		cd = le.AppendUint32(cd, ch.usize)
		for _, v := range []uint16{ch.nameLen, ch.extraLen, ch.commentLen, ch.disk, ch.internal} {
			cd = le.AppendUint16(cd, v)
		}
		cd = le.AppendUint32(cd, ch.external)
		cd = le.AppendUint32(cd, ch.offset)
		cd = append(cd, e.name...)
		cd = append(cd, e.cdExtra...)
		cd = append(cd, e.cdComment...)
	}
	cd = append(cd, z.cdTail...)
	end := rawEOCD{onDisk: uint16(len(z.entries)), total: uint16(len(z.entries)),
		cdSize: uint32(len(cd)), cdOff: uint32(len(out)), commentLen: uint16(len(z.comment))}
	if z.eocd != nil {
		z.eocd(&end)
	}
	out = append(out, cd...)
	out = le.AppendUint32(out, sigEOCD)
	for _, v := range []uint16{end.disk, end.cdDisk, end.onDisk, end.total} {
		out = le.AppendUint16(out, v)
	}
	out = le.AppendUint32(out, end.cdSize)
	out = le.AppendUint32(out, end.cdOff)
	out = le.AppendUint16(out, end.commentLen)
	out = append(out, z.comment...)
	return append(out, z.suffix...)
}
