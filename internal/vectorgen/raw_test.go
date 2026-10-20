// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package vectorgen

import (
	"bytes"
	"compress/flate"
	"encoding/binary"
	"hash/crc32"
)

// This file writes zips field by field, every field of every record under
// the case's control, so a hostile package can break one rule of ADR-001
// D1's profile at a time. An archive with no changes is exactly the profile
// zipfmt.Write writes.

// The profile's constants (ADR-001 D1).
const (
	sigLocal      = 0x04034b50
	sigCentral    = 0x02014b50
	sigEOCD       = 0x06054b50
	versionMadeBy = 0x0314
	flagUTF8      = 0x0800
	dosDate       = 0x0021
	externalAttrs = 0x81A40000
	methodDeflate = 8
)

// header is every field of a local or central directory header.
type header struct {
	madeBy, needed, flags, method, time, date uint16
	crc, csize, usize                         uint32
	nameLen, extraLen, commentLen             uint16
	disk, internal                            uint16
	external, offset                          uint32
}

// zentry is one entry. Its headers start from the profile's values for its
// content; local and central change them.
type zentry struct {
	name    string
	content []byte
	deflate bool
	data    []byte // the file data, when set; otherwise content, deflated or not

	localName           string // the local header's name, when set
	localExtra, cdExtra []byte
	cdComment           []byte
	local, central      func(*header)
	gapAfter            []byte // bytes between this entry and the next record
}

// eocd is every field of the end record.
type eocd struct {
	disk, cdDisk, onDisk, total uint16
	cdSize, cdOff               uint32
	commentLen                  uint16
}

type archive struct {
	prefix  []byte
	entries []zentry
	cdTail  []byte // bytes after the last central directory header
	eocd    func(*eocd)
	comment []byte // after the end record, as its comment
	suffix  []byte // after the comment
}

// deflated is b as one raw DEFLATE stream at level 9 (ADR-001 D4 item 3).
func deflated(b []byte) []byte {
	var buf bytes.Buffer
	w, _ := flate.NewWriter(&buf, flate.BestCompression)
	w.Write(b)
	w.Close()
	return buf.Bytes()
}

func (z archive) bytes() []byte {
	b, _ := z.write()
	return b
}

// built is z's bytes and where they placed each record.
func (z archive) built() built {
	b, l := z.write()
	return built{bytes: b, at: l}
}

// write is z's bytes, and the layout of every record it placed.
func (z archive) write() ([]byte, layout) {
	le := binary.LittleEndian
	out := append([]byte{}, z.prefix...)
	var cd []byte
	var l layout
	for _, e := range z.entries {
		data := e.content
		h := header{madeBy: versionMadeBy, needed: 10, flags: flagUTF8, date: dosDate,
			crc: crc32.ChecksumIEEE(e.content), usize: uint32(len(e.content)), external: externalAttrs}
		if e.deflate {
			data, h.method, h.needed = deflated(e.content), methodDeflate, 20
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
		p := placed{name: e.name, local: len(out), central: len(cd), lh: lh, ch: ch}
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
		p.data = len(out)
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
		l.entries = append(l.entries, p)
	}
	l.cdTail = len(cd)
	cd = append(cd, z.cdTail...)
	end := eocd{onDisk: uint16(len(z.entries)), total: uint16(len(z.entries)),
		cdSize: uint32(len(cd)), cdOff: uint32(len(out)), commentLen: uint16(len(z.comment))}
	if z.eocd != nil {
		z.eocd(&end)
	}
	l.cd = len(out)
	for i := range l.entries {
		l.entries[i].central += l.cd
	}
	l.cdTail += l.cd
	out = append(out, cd...)
	l.end, l.eocd = len(out), end
	out = le.AppendUint32(out, sigEOCD)
	for _, v := range []uint16{end.disk, end.cdDisk, end.onDisk, end.total} {
		out = le.AppendUint16(out, v)
	}
	out = le.AppendUint32(out, end.cdSize)
	out = le.AppendUint32(out, end.cdOff)
	out = le.AppendUint16(out, end.commentLen)
	out = append(out, z.comment...)
	l.suffix = len(out)
	out = append(out, z.suffix...)
	l.size = len(out)
	return out, l
}
