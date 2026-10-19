// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package conformance

import (
	"archive/tar"
	"bytes"
	"compress/flate"
	"encoding/binary"
	"errors"
	"fmt"
	"hash/crc32"
	"io"
	"slices"
	"time"
)

// maxInflate bounds what the runner inflates from one zip entry or tar
// member, so a tool that writes a bomb fails a step rather than the run.
const maxInflate = 256 << 20

// The compression methods ADR-001 D1 allows.
const (
	methodStore   = 0
	methodDeflate = 8
)

// The runner reads and writes zips with its own code over compress/flate:
// it links no part of the implementation, and archive/zip is zipfmt's
// alone (layers.json).

// zipEntry is one entry of a zip, as a container comparison sees it.
type zipEntry struct {
	name    string
	method  uint16
	content []byte
}

// isZip reports whether b starts like a zip's first local header.
func isZip(b []byte) bool {
	return len(b) >= 4 && binary.LittleEndian.Uint32(b) == 0x04034b50
}

// zipHeader is one central directory header.
type zipHeader struct {
	name                                       string
	creator, reader, flags, method, time, date uint16
	crc, csize, usize                          uint32
	extra, comment                             int
	disk, internal                             uint16
	external, offset                           uint32
	at                                         int64 // where the header is
}

// zipArchive is a zip's central directory and end record.
type zipArchive struct {
	headers     []zipHeader
	end         int64 // where the end record is
	disk, cdDsk uint16
	count       uint16
	total       uint16
	cdSize      int64
	cdStart     int64
	comment     int
}

// parseZip reads b's end record and central directory.
func parseZip(b []byte) (*zipArchive, error) {
	le := binary.LittleEndian
	end := int64(-1)
	for i := int64(len(b)) - 22; i >= 0 && i >= int64(len(b))-22-0xFFFF; i-- {
		if le.Uint32(b[i:]) == 0x06054b50 {
			end = i
			break
		}
	}
	if end < 0 {
		return nil, errors.New("no end of central directory record")
	}
	e := b[end:]
	z := &zipArchive{end: end, disk: le.Uint16(e[4:]), cdDsk: le.Uint16(e[6:]), count: le.Uint16(e[8:]), total: le.Uint16(e[10:]),
		cdSize: int64(le.Uint32(e[12:])), cdStart: int64(le.Uint32(e[16:])), comment: int(le.Uint16(e[20:]))}
	if z.cdStart+z.cdSize > end {
		return nil, errors.New("the central directory runs past the end record")
	}
	for p := z.cdStart; p < z.cdStart+z.cdSize; {
		if p+46 > z.cdStart+z.cdSize || le.Uint32(b[p:]) != 0x02014b50 {
			return nil, fmt.Errorf("no central directory header at %d", p)
		}
		h := b[p : p+46]
		nameLen, extra, comment := int64(le.Uint16(h[28:])), int(le.Uint16(h[30:])), int(le.Uint16(h[32:]))
		if p+46+nameLen > z.cdStart+z.cdSize {
			return nil, fmt.Errorf("the central directory header at %d runs past the directory", p)
		}
		z.headers = append(z.headers, zipHeader{
			name: string(b[p+46 : p+46+nameLen]), creator: le.Uint16(h[4:]), reader: le.Uint16(h[6:]), flags: le.Uint16(h[8:]),
			method: le.Uint16(h[10:]), time: le.Uint16(h[12:]), date: le.Uint16(h[14:]), crc: le.Uint32(h[16:]),
			csize: le.Uint32(h[20:]), usize: le.Uint32(h[24:]), extra: extra, comment: comment,
			disk: le.Uint16(h[34:]), internal: le.Uint16(h[36:]), external: le.Uint32(h[38:]), offset: le.Uint32(h[42:]), at: p,
		})
		p += 46 + nameLen + int64(extra) + int64(comment)
	}
	return z, nil
}

// data returns the file data of h, which follows its local header.
func (h zipHeader) data(b []byte) ([]byte, int64, error) {
	le := binary.LittleEndian
	off := int64(h.offset)
	if off+30 > int64(len(b)) || le.Uint32(b[off:]) != 0x04034b50 {
		return nil, 0, fmt.Errorf("no local header at %d", off)
	}
	start := off + 30 + int64(le.Uint16(b[off+26:])) + int64(le.Uint16(b[off+28:]))
	if start+int64(h.csize) > int64(len(b)) {
		return nil, 0, errors.New("the file data runs past the archive")
	}
	return b[start : start+int64(h.csize)], start, nil
}

// inflate returns h's content, bounded by maxInflate and checked against
// its size and CRC-32.
func (h zipHeader) inflate(b []byte) ([]byte, error) {
	if h.usize > maxInflate {
		return nil, fmt.Errorf("it declares %d bytes, above the runner's %d", h.usize, maxInflate)
	}
	data, _, err := h.data(b)
	if err != nil {
		return nil, err
	}
	var content []byte
	switch h.method {
	case methodStore:
		content = bytes.Clone(data)
	case methodDeflate:
		fr := flate.NewReader(bytes.NewReader(data))
		content, err = io.ReadAll(io.LimitReader(fr, int64(h.usize)+1))
		fr.Close()
		if err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("compression method %d", h.method)
	}
	if len(content) != int(h.usize) || crc32.ChecksumIEEE(content) != h.crc {
		return nil, errors.New("the content does not match its recorded size and CRC-32")
	}
	return content, nil
}

// readZip reads b's entries in central-directory order, inflated.
func readZip(b []byte) ([]zipEntry, error) {
	z, err := parseZip(b)
	if err != nil {
		return nil, err
	}
	var out []zipEntry
	for _, h := range z.headers {
		content, err := h.inflate(b)
		if err != nil {
			return nil, fmt.Errorf("entry %q: %w", h.name, err)
		}
		out = append(out, zipEntry{name: h.name, method: h.method, content: content})
	}
	return out, nil
}

// zipFixedFields lists every field of b that differs from the value
// ADR-001 D1 fixes: no prefix and nothing between the parts; local
// headers in order, agreeing with the central directory, with no extra
// field; the central directory's fixed values; and an end record that is
// the last 22 bytes, on disk 0, counting every entry, with no comment.
func zipFixedFields(b []byte) []string {
	z, err := parseZip(b)
	if err != nil {
		return []string{err.Error()}
	}
	le := binary.LittleEndian
	var bad []string
	add := func(name, format string, args ...any) {
		bad = append(bad, fmt.Sprintf("entry %q: ", name)+fmt.Sprintf(format, args...))
	}
	off := int64(0)
	for _, h := range z.headers {
		wantVersion := uint16(10)
		if h.method == methodDeflate {
			wantVersion = 20
		}
		if h.method != methodStore && h.method != methodDeflate {
			add(h.name, "compression method %d, not 0 or 8", h.method)
		}
		if h.reader != wantVersion {
			add(h.name, "version needed to extract %d, not %d", h.reader, wantVersion)
		}
		if h.flags != 0x0800 {
			add(h.name, "general purpose flags %#04x, not 0x0800", h.flags)
		}
		if h.time != 0 || h.date != 0x0021 {
			add(h.name, "modification time %#04x and date %#04x, not 0x0000 and 0x0021", h.time, h.date)
		}
		if h.extra != 0 || h.comment != 0 {
			add(h.name, "an extra field or a comment in the central directory")
		}
		if h.creator != 0x0314 {
			add(h.name, "version made by %#04x, not 0x0314", h.creator)
		}
		if h.disk != 0 || h.internal != 0 {
			add(h.name, "disk number %d and internal attributes %#04x, not 0 and 0", h.disk, h.internal)
		}
		if h.external != 0x81A40000 {
			add(h.name, "external attributes %#08x, not 0x81A40000", h.external)
		}
		if h.method == methodStore && h.csize != h.usize {
			add(h.name, "stored with compressed size %d and uncompressed size %d", h.csize, h.usize)
		}
		if int64(h.offset) != off {
			add(h.name, "its local header is at %d, not where the previous entry ends, %d", h.offset, off)
			return bad
		}
		_, start, err := h.data(b)
		if err != nil {
			add(h.name, "%v", err)
			return bad
		}
		l := b[off : off+30]
		switch {
		case le.Uint16(l[4:]) != h.reader, le.Uint16(l[6:]) != h.flags, le.Uint16(l[8:]) != h.method,
			le.Uint16(l[10:]) != h.time, le.Uint16(l[12:]) != h.date, le.Uint32(l[14:]) != h.crc,
			le.Uint32(l[18:]) != h.csize, le.Uint32(l[22:]) != h.usize,
			string(b[off+30:off+30+int64(le.Uint16(l[26:]))]) != h.name:
			add(h.name, "the local header disagrees with the central directory")
		case le.Uint16(l[28:]) != 0:
			add(h.name, "an extra field of %d bytes in the local header", le.Uint16(l[28:]))
		}
		off = start + int64(h.csize)
	}
	switch {
	case z.cdStart != off:
		bad = append(bad, fmt.Sprintf("the central directory starts at %d, not where the entries end, %d", z.cdStart, off))
	case z.cdStart+z.cdSize != z.end:
		bad = append(bad, "bytes lie between the central directory and the end record")
	}
	if z.end != int64(len(b))-22 || z.comment != 0 {
		bad = append(bad, "the end record is not the last 22 bytes: the archive has a comment or trailing bytes")
	}
	if z.disk != 0 || z.cdDsk != 0 || int(z.count) != len(z.headers) || int(z.total) != len(z.headers) {
		bad = append(bad, "the end record's disk numbers or entry counts are not 0 and the number of entries")
	}
	return bad
}

// writeZip writes entries as a zip with every field at ADR-001 D1's fixed
// value; badHeader sets a non-fixed modification time on the first entry.
func writeZip(entries []zipEntry, badHeader bool) []byte {
	le := binary.LittleEndian
	var out, dir []byte
	for i, e := range entries {
		data, version := e.content, uint16(10)
		if e.method == methodDeflate {
			var c bytes.Buffer
			fw, _ := flate.NewWriter(&c, flate.BestCompression)
			fw.Write(e.content)
			fw.Close()
			data, version = c.Bytes(), 20
		}
		mtime := uint16(0)
		if badHeader && i == 0 {
			mtime = 1
		}
		crc, off := crc32.ChecksumIEEE(e.content), len(out)
		out = le.AppendUint32(out, 0x04034b50)
		for _, v := range []uint16{version, 0x0800, e.method, mtime, 0x0021} {
			out = le.AppendUint16(out, v)
		}
		out = le.AppendUint32(le.AppendUint32(le.AppendUint32(out, crc), uint32(len(data))), uint32(len(e.content)))
		out = le.AppendUint16(le.AppendUint16(out, uint16(len(e.name))), 0)
		out = append(append(out, e.name...), data...)

		dir = le.AppendUint32(dir, 0x02014b50)
		for _, v := range []uint16{0x0314, version, 0x0800, e.method, mtime, 0x0021} {
			dir = le.AppendUint16(dir, v)
		}
		dir = le.AppendUint32(le.AppendUint32(le.AppendUint32(dir, crc), uint32(len(data))), uint32(len(e.content)))
		for _, v := range []uint16{uint16(len(e.name)), 0, 0, 0, 0} {
			dir = le.AppendUint16(dir, v)
		}
		dir = le.AppendUint32(le.AppendUint32(dir, 0x81A40000), uint32(off))
		dir = append(dir, e.name...)
	}
	start := len(out)
	out = append(out, dir...)
	out = le.AppendUint32(out, 0x06054b50)
	for _, v := range []uint16{0, 0, uint16(len(entries)), uint16(len(entries))} {
		out = le.AppendUint16(out, v)
	}
	out = le.AppendUint32(le.AppendUint32(out, uint32(len(dir))), uint32(start))
	return le.AppendUint16(out, 0)
}

// compareZips compares two zips by entries (ADR-011 D10 item 3): names,
// order and content, the methods of the entries listed, and the actual
// zip's fixed header fields. It returns the first difference, or "".
func compareZips(want, got []byte, methods []string, all bool) string {
	we, err := readZip(want)
	if err != nil {
		return "the expected container does not read as a zip: " + err.Error()
	}
	ge, err := readZip(got)
	if err != nil {
		return "the container does not read as a zip: " + err.Error()
	}
	for i := 0; i < max(len(we), len(ge)); i++ {
		switch {
		case i >= len(ge):
			return fmt.Sprintf("entry %d: expected %q, but the container ends", i+1, we[i].name)
		case i >= len(we):
			return fmt.Sprintf("entry %d: %q is extra", i+1, ge[i].name)
		case we[i].name != ge[i].name:
			return fmt.Sprintf("entry %d: expected %q, got %q", i+1, we[i].name, ge[i].name)
		case !bytes.Equal(we[i].content, ge[i].content):
			return fmt.Sprintf("entry %q: other content\n%s", we[i].name, firstDifference(we[i].content, ge[i].content))
		case (all || slices.Contains(methods, we[i].name)) && we[i].method != ge[i].method:
			return fmt.Sprintf("entry %q: compression method %d, expected %d", we[i].name, ge[i].method, we[i].method)
		}
	}
	if bad := zipFixedFields(got); len(bad) > 0 {
		return "a header field differs from its fixed value: " + bad[0]
	}
	return ""
}

// tarMember is one member of a tar stream, as ADR-005 D17 item 5 compares
// it.
type tarMember struct {
	name     string
	typeflag byte
	mode     int64
	uid, gid int
	uname    string
	gname    string
	mtime    time.Time
	size     int64
	content  []byte
}

// header is every compared field of m but its content.
func (m tarMember) header() string {
	return fmt.Sprintf("{name %q, typeflag %q, mode %#o, uid %d, gid %d, uname %q, gname %q, mtime %s, size %d}",
		m.name, m.typeflag, m.mode, m.uid, m.gid, m.uname, m.gname, m.mtime.Format(time.RFC3339Nano), m.size)
}

// readTar reads every member of b.
func readTar(b []byte) ([]tarMember, error) {
	tr := tar.NewReader(bytes.NewReader(b))
	var out []tarMember
	for {
		h, err := tr.Next()
		if err == io.EOF {
			return out, nil
		}
		if err != nil {
			return nil, err
		}
		content, err := io.ReadAll(io.LimitReader(tr, maxInflate+1))
		if err != nil {
			return nil, err
		}
		out = append(out, tarMember{
			name: h.Name, typeflag: h.Typeflag, mode: h.Mode, uid: h.Uid, gid: h.Gid,
			uname: h.Uname, gname: h.Gname, mtime: h.ModTime.UTC(), size: h.Size, content: content,
		})
	}
}

// compareTars compares two archive streams by entries (ADR-011 D10 item
// 5): order, path, typeflag, mode, uid, gid, uname, gname, mtime, size and
// content. It returns the first difference, or "".
func compareTars(want, got []byte) string {
	wm, err := readTar(want)
	if err != nil {
		return "the expected stream does not read as a tar: " + err.Error()
	}
	gm, err := readTar(got)
	if err != nil {
		return "the stream does not read as a tar: " + err.Error()
	}
	for i := 0; i < max(len(wm), len(gm)); i++ {
		switch {
		case i >= len(gm):
			return fmt.Sprintf("member %d: expected %q, but the stream ends", i+1, wm[i].name)
		case i >= len(wm):
			return fmt.Sprintf("member %d: %q is extra", i+1, gm[i].name)
		}
		w, g := wm[i], gm[i]
		if !bytes.Equal(w.content, g.content) {
			return fmt.Sprintf("member %q: other content\n%s", w.name, firstDifference(w.content, g.content))
		}
		if d, g := w.header(), g.header(); d != g {
			return fmt.Sprintf("member %d: expected %s, got %s", i+1, d, g)
		}
	}
	return ""
}
