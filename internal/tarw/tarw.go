// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

// Package tarw writes the standard-output archive (ADR-005 D17): one POSIX
// pax interchange format archive holding every top-level file of a render,
// one regular-file entry per file, in byte order of the path.
//
// Every header field is D17 item 3's constant or is taken from the file's
// path, executable bit and size, so nothing of the host, the clock, the
// user or the process enters the archive. A file's header is ustar where
// its path fits one, as a name or as a prefix and a name split at a "/",
// and is otherwise preceded by a typeflag x extended header carrying one
// path record (RULINGS R2-072). That extended header's own name is
// <dir>/PaxHeaders.0/<file>, taken from no process id and no random source
// (ADR-005 PR16, RULINGS R2-077).
//
// The bytes are exactly those archive/tar writes when it is given the same
// headers with no format forced, which ADR-005 D17 item 5 states: the
// golden files under testdata/ hold them (ADR-010 D14 item 5), and a test
// compares this package with archive/tar over many paths. The package
// encodes them itself rather than linking archive/tar, because
// archive/tar imports os/user, which ADR-010 D11 item 10 keeps out of the
// binary.
//
// The package reaches no host file: the caller hands over the stream, and
// each file's content as a reader of a stated size, so an archive is built
// from the render's plan and never from files on disk (D17 item 6).
package tarw

import (
	"errors"
	"io"
	pathpkg "path"
	"strconv"
	"strings"
)

// The header fields D17 item 3 fixes. uname, gname, devmajor and devminor
// are left at their zero values, which archive/tar writes as an empty name
// and 0.
const (
	modeFile       = 0o644
	modeExecutable = 0o755

	// mtime is 1980-01-01T00:00:00Z as Unix seconds, the instant ADR-001
	// fixes for zip entries, so every archive the tool writes carries one
	// timestamp. modTime is the same instant as archive/tar takes it.
	mtime = 315532800

	// maxSize is the largest size a ustar header's eleven octal digits
	// hold. A larger file would need a pax size record, and D17 item 3's
	// extended header carries only path.
	maxSize = 0o77777777777

	blockSize = 512
	// nameSize and prefixSize are the ustar name and prefix fields'.
	nameSize   = 100
	prefixSize = 155
)

// The offsets of a ustar header's fields (POSIX.1-2017, pax, "ustar
// Interchange Format").
const (
	offName     = 0
	offMode     = 100
	offUID      = 108
	offGID      = 116
	offSize     = 124
	offMtime    = 136
	offChksum   = 148
	offTypeflag = 156
	offMagic    = 257
	offVersion  = 263
	offDevmajor = 329
	offDevminor = 337
	offPrefix   = 345
)

// Writer writes one standard-output archive. Add each file in byte order of
// its path, then Close.
type Writer struct {
	w    io.Writer
	last string
	n    int
	err  error
}

// NewWriter returns a Writer that writes an archive to w.
func NewWriter(w io.Writer) *Writer {
	return &Writer{w: w}
}

// Add writes one regular-file entry (typeflag 0) for the file at path, with
// mode 0755 when executable and 0644 otherwise, whose content is exactly
// size bytes read from r.
//
// Add does not judge paths: a render's top-level paths pass ADR-005 D12's
// portable-path predicate, and establishing that is its caller's. It
// refuses only what would break the archive's profile: an empty path, a
// path that holds NUL or ends in "/", a path not strictly after the
// previous one in byte order (ADR-005 D13 item 6), a size that is negative
// or does not fit a ustar header, and content that is not exactly size
// bytes long. After any error the Writer refuses every further call.
func (w *Writer) Add(path string, executable bool, size int64, r io.Reader) error {
	if w.err != nil {
		return w.err
	}
	w.err = w.add(path, executable, size, r)
	return w.err
}

func (w *Writer) add(path string, executable bool, size int64, r io.Reader) error {
	switch {
	case path == "":
		return errors.New("tarw: a file has an empty path")
	case strings.IndexByte(path, 0) >= 0:
		return errors.New("tarw: a path holds NUL")
	case strings.HasSuffix(path, "/"):
		return errors.New("tarw: a file's path ends in \"/\"")
	case w.n > 0 && path <= w.last:
		return errors.New("tarw: a path is not after the previous one in byte order")
	case size < 0 || size > maxSize:
		return errors.New("tarw: a file of " + strconv.FormatInt(size, 10) + " bytes does not fit a ustar header")
	}
	if err := w.writeHeaders(path, executable, size); err != nil {
		return err
	}
	if _, err := io.CopyN(w.w, r, size); err != nil {
		if err == io.EOF {
			return errors.New("tarw: a file's content is shorter than its size")
		}
		return err
	}
	var extra [1]byte
	if n, _ := io.ReadFull(r, extra[:]); n > 0 {
		return errors.New("tarw: a file's content is longer than its size")
	}
	if err := w.pad(size); err != nil {
		return err
	}
	w.last, w.n = path, w.n+1
	return nil
}

// writeHeaders writes the file's header, preceded by an extended header
// carrying its path when no ustar header holds the path.
func (w *Writer) writeHeaders(path string, executable bool, size int64) error {
	mode := int64(modeFile)
	if executable {
		mode = modeExecutable
	}
	var h [blockSize]byte
	if prefix, name, ok := ustarPath(path); ok {
		putString(h[offName:offName+nameSize], name)
		putString(h[offPrefix:offPrefix+prefixSize], prefix)
	} else {
		if err := w.writeExtended(path); err != nil {
			return err
		}
		// The name field carries what of the path is ASCII, cut to fit,
		// as archive/tar writes it; the extended header's record is the
		// path a reader takes.
		putString(h[offName:offName+nameSize], toASCII(path))
	}
	putOctal(h[offMode:offMode+8], mode)
	putOctal(h[offUID:offUID+8], 0)
	putOctal(h[offGID:offGID+8], 0)
	putOctal(h[offSize:offSize+12], size)
	putOctal(h[offMtime:offMtime+12], mtime)
	h[offTypeflag] = '0'
	putOctal(h[offDevmajor:offDevmajor+8], 0)
	putOctal(h[offDevminor:offDevminor+8], 0)
	return w.writeBlock(&h)
}

// writeExtended writes a typeflag x extended header carrying one record,
// path, and its data, padded to a block.
func (w *Writer) writeExtended(path string) error {
	data := paxRecord("path", path)
	dir, file := pathpkg.Split(path)
	name := toASCII(pathpkg.Join(dir, "PaxHeaders.0", file))
	if len(name) > nameSize {
		name = name[:nameSize]
	}
	name = strings.TrimRight(name, "/")
	var h [blockSize]byte
	putString(h[offName:offName+nameSize], name)
	putOctal(h[offMode:offMode+8], 0)
	putOctal(h[offUID:offUID+8], 0)
	putOctal(h[offGID:offGID+8], 0)
	putOctal(h[offSize:offSize+12], int64(len(data)))
	putOctal(h[offMtime:offMtime+12], 0)
	h[offTypeflag] = 'x'
	if err := w.writeBlock(&h); err != nil {
		return err
	}
	if _, err := io.WriteString(w.w, data); err != nil {
		return err
	}
	return w.pad(int64(len(data)))
}

// writeBlock sets the header's magic, version and checksum, and writes it.
func (w *Writer) writeBlock(h *[blockSize]byte) error {
	copy(h[offMagic:], "ustar\x00")
	copy(h[offVersion:], "00")
	var sum int64
	for i, c := range h {
		if offChksum <= i && i < offChksum+8 {
			c = ' '
		}
		sum += int64(c)
	}
	putOctal(h[offChksum:offChksum+7], sum)
	h[offChksum+7] = ' '
	_, err := w.w.Write(h[:])
	return err
}

// pad writes the zeros that fill the last block of n bytes of data.
func (w *Writer) pad(n int64) error {
	if r := n % blockSize; r != 0 {
		_, err := w.w.Write(make([]byte, blockSize-r))
		return err
	}
	return nil
}

// ustarPath is path as a ustar header holds it: the name alone when the
// path is ASCII and fits the name field, or a prefix and a name split at
// the last "/" that fits both fields. ok is false when neither fits.
func ustarPath(path string) (prefix, name string, ok bool) {
	if !isASCII(path) {
		return "", "", false
	}
	if len(path) <= nameSize {
		return "", path, true
	}
	i := strings.LastIndex(path[:min(len(path), prefixSize+1)], "/")
	if i <= 0 || len(path)-i-1 > nameSize || len(path)-i-1 == 0 {
		return "", "", false
	}
	return path[:i], path[i+1:], true
}

// isASCII reports whether s holds only ASCII and no NUL.
func isASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] >= 0x80 || s[i] == 0 {
			return false
		}
	}
	return true
}

// toASCII is s with every code point that is not ASCII, and every NUL,
// dropped.
func toASCII(s string) string {
	if isASCII(s) {
		return s
	}
	b := make([]byte, 0, len(s))
	for _, c := range s {
		if c < 0x80 && c != 0 {
			b = append(b, byte(c))
		}
	}
	return string(b)
}

// putString copies s into the field b, cut to fit, followed by a NUL when
// it is shorter. A cut that leaves "/" at the field's end ends at the
// slashes before it instead, as archive/tar writes it, so no reader takes
// the name for a directory.
func putString(b []byte, s string) {
	copy(b, s)
	if len(s) < len(b) {
		b[len(s)] = 0
	}
	if len(s) > len(b) && b[len(b)-1] == '/' {
		b[len(strings.TrimRight(s[:len(b)-1], "/"))] = 0
	}
}

// putOctal writes x into the field b as octal digits, zero-filled to all
// but the field's last byte, which is NUL. Every value given fits.
func putOctal(b []byte, x int64) {
	s := strconv.FormatInt(x, 8)
	if n := len(b) - len(s) - 1; n > 0 {
		s = strings.Repeat("0", n) + s
	}
	putString(b, s)
}

// paxRecord is one extended header record, "<length> <key>=<value>\n",
// whose length counts its own digits.
func paxRecord(key, value string) string {
	size := len(key) + len(value) + 3
	size += len(strconv.Itoa(size))
	rec := strconv.Itoa(size) + " " + key + "=" + value + "\n"
	if len(rec) != size {
		rec = strconv.Itoa(len(rec)) + " " + key + "=" + value + "\n"
	}
	return rec
}

// Close ends the archive with two 512-octet zero blocks and no further
// padding (D17 item 4). It does not close the underlying writer.
func (w *Writer) Close() error {
	if w.err != nil {
		return w.err
	}
	w.err = errors.New("tarw: the archive is closed")
	_, err := w.w.Write(make([]byte, 2*blockSize))
	return err
}
