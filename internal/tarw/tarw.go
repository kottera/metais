// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

// Package tarw writes the standard-output archive (ADR-005 D17): one POSIX
// pax interchange format archive holding every top-level file of a render,
// one regular-file entry per file, in byte order of the path.
//
// Every header field is D17 item 3's constant or is taken from the file's
// path, executable bit and size, so nothing of the host, the clock, the
// user or the process enters the archive. The headers are handed to
// archive/tar with no format forced, which writes a ustar header where the
// path fits one and precedes it with a typeflag x extended header carrying
// one path record where it does not (RULINGS R2-072). That extended
// header's own name is archive/tar's fixed PaxHeaders.0/<file>, taken from
// no process id and no random source (ADR-005 PR16, RULINGS R2-077). The
// golden files under testdata/ hold the exact bytes for each case (ADR-010
// D14 item 5).
//
// archive/tar's GODEBUG setting, tarinsecurepath, governs reading only, so
// no GODEBUG setting changes the bytes written (ADR-010 D11 item 4).
//
// The package reaches no host file: the caller hands over the stream, and
// each file's content as a reader of a stated size, so an archive is built
// from the render's plan and never from files on disk (D17 item 6).
package tarw

import (
	"archive/tar"
	"errors"
	"io"
	"strconv"
	"strings"
	"time"
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
)

var modTime = time.Date(1980, time.January, 1, 0, 0, 0, 0, time.UTC)

// Writer writes one standard-output archive. Add each file in byte order of
// its path, then Close.
type Writer struct {
	tw   *tar.Writer
	last string
	n    int
	err  error
}

// NewWriter returns a Writer that writes an archive to w.
func NewWriter(w io.Writer) *Writer {
	return &Writer{tw: tar.NewWriter(w)}
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
	if err := w.tw.WriteHeader(header(path, executable, size)); err != nil {
		return err
	}
	if _, err := io.CopyN(w.tw, r, size); err != nil {
		if err == io.EOF {
			return errors.New("tarw: a file's content is shorter than its size")
		}
		return err
	}
	var extra [1]byte
	if n, _ := io.ReadFull(r, extra[:]); n > 0 {
		return errors.New("tarw: a file's content is longer than its size")
	}
	w.last, w.n = path, w.n+1
	return nil
}

// header is the archive/tar header D17 item 3 gives the file. Its format is
// left unknown, so archive/tar chooses ustar or pax by what the path needs.
func header(path string, executable bool, size int64) *tar.Header {
	mode := int64(modeFile)
	if executable {
		mode = modeExecutable
	}
	return &tar.Header{
		Typeflag: tar.TypeReg,
		Name:     path,
		Mode:     mode,
		Size:     size,
		ModTime:  modTime,
	}
}

// Close ends the archive with two 512-octet zero blocks and no further
// padding (D17 item 4). It does not close the underlying writer.
func (w *Writer) Close() error {
	if w.err != nil {
		return w.err
	}
	w.err = errors.New("tarw: the archive is closed")
	return w.tw.Close()
}
