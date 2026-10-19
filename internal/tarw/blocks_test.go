// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package tarw

import (
	"bytes"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"testing"
)

// The tests read the archive by the format, block by block, and never
// through archive/tar's reader, so a field archive/tar would write and
// then read back the same wrong way is still seen. They share the
// package's blockSize.

// file is one file handed to the writer under test.
type file struct {
	path       string
	executable bool
	content    string
}

// write archives files with a Writer and returns the bytes.
func write(t testing.TB, files ...file) []byte {
	t.Helper()
	var buf bytes.Buffer
	w := NewWriter(&buf)
	for _, f := range files {
		if err := w.Add(f.path, f.executable, int64(len(f.content)), strings.NewReader(f.content)); err != nil {
			t.Fatalf("Add(%q): %v", f.path, err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	return buf.Bytes()
}

// member is one entry of an archive as the blocks hold it.
type member struct {
	paxHeader []byte   // the typeflag x header block, or nil
	pax       []string // its records, "key=value", in order
	header    []byte   // the entry's own header block
	data      []byte
}

// path is the member's path: its pax path record where it has one, and its
// ustar prefix and name otherwise.
func (m member) path() string {
	for _, r := range m.pax {
		if v, ok := strings.CutPrefix(r, "path="); ok {
			return v
		}
	}
	name, prefix := cstr(m.header[0:100]), cstr(m.header[345:500])
	if prefix == "" {
		return name
	}
	return prefix + "/" + name
}

// members splits an archive into its members, and refuses one that does
// not end in exactly two zero blocks.
func members(b []byte) ([]member, error) {
	if len(b)%blockSize != 0 {
		return nil, fmt.Errorf("the archive is %d bytes, not a whole number of blocks", len(b))
	}
	var ms []member
	var pending *member
	for off := 0; off < len(b); {
		blk := b[off : off+blockSize]
		if isZero(blk) {
			if pending != nil {
				return nil, errors.New("an extended header is followed by a zero block")
			}
			if len(b)-off != 2*blockSize || !isZero(b[off+blockSize:]) {
				return nil, fmt.Errorf("a zero block at %d is not the first of exactly two that end the archive", off)
			}
			return ms, nil
		}
		if err := checksum(blk); err != nil {
			return nil, fmt.Errorf("the header at %d: %v", off, err)
		}
		size, err := octal(blk[124:136])
		if err != nil {
			return nil, fmt.Errorf("the header at %d: size: %v", off, err)
		}
		start := off + blockSize
		end := start + int(size)
		if end > len(b) {
			return nil, fmt.Errorf("the header at %d claims %d bytes past the end", off, size)
		}
		data := b[start:end]
		off = start + (int(size)+blockSize-1)/blockSize*blockSize
		switch blk[156] {
		case 'x':
			if pending != nil {
				return nil, errors.New("two extended headers in a row")
			}
			recs, err := paxRecords(data)
			if err != nil {
				return nil, err
			}
			pending = &member{paxHeader: blk, pax: recs}
		default:
			m := member{header: blk, data: data}
			if pending != nil {
				m.paxHeader, m.pax, pending = pending.paxHeader, pending.pax, nil
			}
			ms = append(ms, m)
		}
	}
	return nil, errors.New("the archive does not end in two zero blocks")
}

// paxRecords parses pax extended header data: "<len> <key>=<value>\n"
// records, each len counting the whole record.
func paxRecords(data []byte) ([]string, error) {
	var recs []string
	for len(data) > 0 {
		sp := bytes.IndexByte(data, ' ')
		if sp < 1 {
			return nil, fmt.Errorf("a pax record has no length: %q", data)
		}
		n, err := strconv.Atoi(string(data[:sp]))
		if err != nil || n <= sp+1 || n > len(data) || data[n-1] != '\n' {
			return nil, fmt.Errorf("a malformed pax record: %q", data)
		}
		recs = append(recs, string(data[sp+1:n-1]))
		data = data[n:]
	}
	return recs, nil
}

func isZero(b []byte) bool {
	for _, c := range b {
		if c != 0 {
			return false
		}
	}
	return true
}

// cstr is a header string field: its bytes up to the first NUL.
func cstr(b []byte) string {
	if i := bytes.IndexByte(b, 0); i >= 0 {
		b = b[:i]
	}
	return string(b)
}

// octal is a header number field: octal digits, NUL- or space-terminated.
func octal(b []byte) (int64, error) {
	s := strings.TrimRight(cstr(b), " ")
	return strconv.ParseInt(s, 8, 64)
}

// sum is a header block's checksum: its bytes summed with the checksum
// field taken as eight spaces.
func sum(blk []byte) int64 {
	var s int64
	for i, c := range blk {
		if i >= 148 && i < 156 {
			c = ' '
		}
		s += int64(c)
	}
	return s
}

func checksum(blk []byte) error {
	got, err := octal(blk[148:156])
	if err != nil {
		return fmt.Errorf("checksum: %v", err)
	}
	if want := sum(blk); got != want {
		return fmt.Errorf("checksum %d, and the block sums to %d", got, want)
	}
	return nil
}

// reseal rewrites a mutated header block's checksum as archive/tar would,
// so a detector sees only the field that was changed.
func reseal(blk []byte) {
	copy(blk[148:156], fmt.Sprintf("%06o\x00 ", sum(blk)))
}
