// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package conformance

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"slices"
	"time"
)

// maxInflate bounds what the runner inflates from one zip entry or tar
// member, so a tool that writes a bomb fails a step rather than the run.
const maxInflate = 256 << 20

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

// readZip reads b's entries in central-directory order, inflated.
func readZip(b []byte) ([]zipEntry, error) {
	zr, err := zip.NewReader(bytes.NewReader(b), int64(len(b)))
	if err != nil && !errors.Is(err, zip.ErrInsecurePath) {
		return nil, err
	}
	var out []zipEntry
	for _, f := range zr.File {
		if f.UncompressedSize64 > maxInflate {
			return nil, fmt.Errorf("entry %q declares %d bytes, above the runner's %d", f.Name, f.UncompressedSize64, maxInflate)
		}
		rc, err := f.Open()
		if err != nil {
			return nil, fmt.Errorf("entry %q: %w", f.Name, err)
		}
		content, err := io.ReadAll(io.LimitReader(rc, maxInflate+1))
		rc.Close()
		if err != nil {
			return nil, fmt.Errorf("entry %q: %w", f.Name, err)
		}
		out = append(out, zipEntry{name: f.Name, method: f.Method, content: content})
	}
	return out, nil
}

// zipFixedFields lists every header field of b that differs from the value
// ADR-001 D1 fixes, local and central, with the structure around them: no
// prefix, no data descriptor, local headers in order and agreeing with the
// central directory, and no archive comment.
func zipFixedFields(b []byte) []string {
	zr, err := zip.NewReader(bytes.NewReader(b), int64(len(b)))
	if err != nil && !errors.Is(err, zip.ErrInsecurePath) {
		return []string{err.Error()}
	}
	var bad []string
	add := func(name, format string, args ...any) {
		bad = append(bad, fmt.Sprintf("entry %q: ", name)+fmt.Sprintf(format, args...))
	}
	if zr.Comment != "" {
		bad = append(bad, "the archive has a comment")
	}
	off := int64(0)
	for _, f := range zr.File {
		wantVersion := uint16(10)
		if f.Method == zip.Deflate {
			wantVersion = 20
		}
		if f.Method != zip.Store && f.Method != zip.Deflate {
			add(f.Name, "compression method %d, not 0 or 8", f.Method)
		}
		if f.ReaderVersion != wantVersion {
			add(f.Name, "version needed to extract %d, not %d", f.ReaderVersion, wantVersion)
		}
		if f.Flags != 0x0800 {
			add(f.Name, "general purpose flags %#04x, not 0x0800", f.Flags)
		}
		if f.ModifiedTime != 0 || f.ModifiedDate != 0x0021 {
			add(f.Name, "modification time %#04x and date %#04x, not 0x0000 and 0x0021", f.ModifiedTime, f.ModifiedDate)
		}
		if len(f.Extra) != 0 {
			add(f.Name, "an extra field of %d bytes in the central directory", len(f.Extra))
		}
		if f.CreatorVersion != 0x0314 {
			add(f.Name, "version made by %#04x, not 0x0314", f.CreatorVersion)
		}
		if f.Comment != "" {
			add(f.Name, "a file comment")
		}
		if f.ExternalAttrs != 0x81A40000 {
			add(f.Name, "external attributes %#08x, not 0x81A40000", f.ExternalAttrs)
		}
		if f.Method == zip.Store && f.CompressedSize64 != f.UncompressedSize64 {
			add(f.Name, "stored with compressed size %d and uncompressed size %d", f.CompressedSize64, f.UncompressedSize64)
		}
		// The local header, which archive/zip reads past: it must start
		// where the previous entry's data ended and repeat the central
		// directory's values with no extra field.
		if off+30 > int64(len(b)) {
			add(f.Name, "no local header at offset %d", off)
			return bad
		}
		h := b[off : off+30]
		le := binary.LittleEndian
		nameLen, extraLen := int64(le.Uint16(h[26:])), int64(le.Uint16(h[28:]))
		switch {
		case le.Uint32(h) != 0x04034b50:
			add(f.Name, "no local header at offset %d", off)
			return bad
		case le.Uint16(h[4:]) != f.ReaderVersion, le.Uint16(h[6:]) != f.Flags, le.Uint16(h[8:]) != f.Method,
			le.Uint16(h[10:]) != f.ModifiedTime, le.Uint16(h[12:]) != f.ModifiedDate, le.Uint32(h[14:]) != f.CRC32,
			int64(le.Uint32(h[18:])) != int64(f.CompressedSize64), int64(le.Uint32(h[22:])) != int64(f.UncompressedSize64):
			add(f.Name, "the local header disagrees with the central directory")
		case extraLen != 0:
			add(f.Name, "an extra field of %d bytes in the local header", extraLen)
		}
		if off+30+nameLen > int64(len(b)) || string(b[off+30:off+30+nameLen]) != f.Name {
			add(f.Name, "the local header's name disagrees with the central directory")
		}
		data, err := f.DataOffset()
		if err != nil || data != off+30+nameLen+extraLen {
			add(f.Name, "the file data does not follow its local header")
		}
		off = off + 30 + nameLen + extraLen + int64(f.CompressedSize64)
	}
	return bad
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
