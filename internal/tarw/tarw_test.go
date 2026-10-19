// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package tarw

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"slices"
	"strings"
	"testing"
)

var (
	long100    = strings.Repeat("n", 100)
	prefix155  = strings.Repeat("d", 155)
	prefix156  = strings.Repeat("e", 156)
	component1 = "z" + long100
)

// samples is a render's top-level files, in path order, covering every way
// a path is carried: a ustar name, a prefix/name split, and the pax path
// record for a path that is not ASCII and for one too long to split.
var samples = []file{
	{"a.txt", false, "alpha\n"},
	{"bin/run.sh", true, "#!/bin/sh\necho ok\n"},
	{"café/menü.txt", false, "non-ASCII path\n"},
	{prefix155 + "/" + long100, false, ""},
	{component1, true, strings.Repeat("x", 513)},
}

// wantMode is the mode D17 item 3 gives f.
func wantMode(f file) int64 {
	if f.executable {
		return 0o644 | 0o111
	}
	return 0o644
}

// field is one header field ADR-005 D17 item 3 fixes: how to check it, and
// how to break it, so the check is proved red on a header that gets it
// wrong.
type field struct {
	name    string
	check   func(m member, f file) error
	corrupt func(m *member)
}

func numberIs(hdr []byte, lo, hi int, want int64) error {
	got, err := octal(hdr[lo:hi])
	if err != nil {
		return err
	}
	if got != want {
		return fmt.Errorf("%d, want %d", got, want)
	}
	return nil
}

func stringIs(hdr []byte, lo, hi int, want string) error {
	if got := string(hdr[lo:hi]); got != want {
		return fmt.Errorf("%q, want %q", got, want)
	}
	return nil
}

func setNumber(lo, hi int, v int64) func(*member) {
	return func(m *member) {
		copy(m.header[lo:hi], fmt.Sprintf("%0*o\x00", hi-lo-1, v))
	}
}

func setString(lo, hi int, s string) func(*member) {
	return func(m *member) {
		clear(m.header[lo:hi])
		copy(m.header[lo:hi], s)
	}
}

var nul = func(n int) string { return strings.Repeat("\x00", n) }

var fields = []field{
	{"name and prefix",
		func(m member, f file) error {
			if got := m.path(); got != f.path {
				return fmt.Errorf("path %q, want %q", got, f.path)
			}
			return nil
		},
		// Break the path where it is carried: the pax record, or the
		// ustar name.
		func(m *member) {
			if m.pax != nil {
				m.pax = []string{"path=Q"}
				return
			}
			m.header[0] = 'Q'
		}},
	{"typeflag",
		func(m member, _ file) error { return stringIs(m.header, 156, 157, "0") },
		setString(156, 157, "7")},
	{"mode",
		func(m member, f file) error { return numberIs(m.header, 100, 108, wantMode(f)) },
		setNumber(100, 108, 0o600)},
	{"uid",
		func(m member, _ file) error { return numberIs(m.header, 108, 116, 0) },
		setNumber(108, 116, 1000)},
	{"gid",
		func(m member, _ file) error { return numberIs(m.header, 116, 124, 0) },
		setNumber(116, 124, 1000)},
	{"uname",
		func(m member, _ file) error { return stringIs(m.header, 265, 297, nul(32)) },
		setString(265, 297, "root")},
	{"gname",
		func(m member, _ file) error { return stringIs(m.header, 297, 329, nul(32)) },
		setString(297, 329, "wheel")},
	{"mtime",
		func(m member, _ file) error { return numberIs(m.header, 136, 148, 315532800) },
		setNumber(136, 148, 315532801)},
	{"size",
		func(m member, f file) error {
			if string(m.data) != f.content {
				return fmt.Errorf("content %q, want %q", m.data, f.content)
			}
			return numberIs(m.header, 124, 136, int64(len(f.content)))
		},
		setNumber(124, 136, 7)},
	{"devmajor",
		func(m member, _ file) error { return numberIs(m.header, 329, 337, 0) },
		setNumber(329, 337, 8)},
	{"devminor",
		func(m member, _ file) error { return numberIs(m.header, 337, 345, 0) },
		setNumber(337, 345, 1)},
	{"magic",
		func(m member, _ file) error { return stringIs(m.header, 257, 263, "ustar\x00") },
		setString(257, 263, "ustar ")},
	{"version",
		func(m member, _ file) error { return stringIs(m.header, 263, 265, "00") },
		setString(263, 265, " \x00")},
	{"linkname",
		func(m member, _ file) error { return stringIs(m.header, 157, 257, nul(100)) },
		setString(157, 257, "target")},
}

// TestHeaderFields is W2-13.2: each field D17 item 3 fixes is written at
// its fixed value on every entry, whether its path is carried by ustar or
// by pax, and each field's check is red on a header that breaks it.
func TestHeaderFields(t *testing.T) {
	ms, err := members(write(t, samples...))
	if err != nil {
		t.Fatal(err)
	}
	if len(ms) != len(samples) {
		t.Fatalf("%d members, want %d", len(ms), len(samples))
	}
	for _, fd := range fields {
		t.Run(fd.name, func(t *testing.T) {
			for i, m := range ms {
				if err := fd.check(m, samples[i]); err != nil {
					t.Errorf("%q: %s: %v", samples[i].path, fd.name, err)
				}
				broken := m
				broken.header = slices.Clone(m.header)
				fd.corrupt(&broken)
				reseal(broken.header)
				if fd.check(broken, samples[i]) == nil {
					t.Errorf("%q: the %s check passes a header that breaks it", samples[i].path, fd.name)
				}
			}
		})
	}
}

// TestNoOtherMember holds D17 item 2: one regular-file entry per file and
// nothing else; no directory entry, no link and no global extended header,
// so every header block is typeflag 0 or the x that precedes one.
func TestNoOtherMember(t *testing.T) {
	b := write(t, samples...)
	for off := 0; off+blockSize <= len(b)-2*blockSize; {
		blk := b[off : off+blockSize]
		if tf := blk[156]; tf != '0' && tf != 'x' {
			t.Errorf("a header at %d has typeflag %q", off, tf)
		}
		size, err := octal(blk[124:136])
		if err != nil {
			t.Fatal(err)
		}
		off += blockSize + int(size+blockSize-1)/blockSize*blockSize
	}
}

// TestPathOrder holds D17 item 2's order: the entries are in the order
// given, which is byte order of the path.
func TestPathOrder(t *testing.T) {
	ms, err := members(write(t, samples...))
	if err != nil {
		t.Fatal(err)
	}
	for i, m := range ms {
		if m.path() != samples[i].path {
			t.Errorf("member %d is %q, want %q", i, m.path(), samples[i].path)
		}
	}
}

// TestEnd is D17 item 4: two 512-octet zero blocks end the archive and
// nothing follows them, with or without entries.
func TestEnd(t *testing.T) {
	if b := write(t); !bytes.Equal(b, make([]byte, 2*blockSize)) {
		t.Errorf("an empty archive is %d bytes, want exactly two zero blocks", len(b))
	}
	b := write(t, samples[:2]...)
	// a.txt and bin/run.sh: a header and one data block each.
	if want := 4*blockSize + 2*blockSize; len(b) != want {
		t.Errorf("the archive is %d bytes, want %d", len(b), want)
	}
	if !isZero(b[len(b)-2*blockSize:]) {
		t.Error("the archive does not end in two zero blocks")
	}
	// The reader the other tests use refuses padding after the end, and a
	// missing end.
	if _, err := members(append(slices.Clone(b), make([]byte, blockSize)...)); err == nil {
		t.Error("members accepts a third zero block")
	}
	if _, err := members(b[:len(b)-blockSize]); err == nil {
		t.Error("members accepts one zero block")
	}
}

// TestPaxOnlyWhereRequired is W2-13.3: a path that fits ustar, as a name
// or as a prefix/name split at a "/", has no extended header; a path that
// does not is preceded by one carrying exactly one record, path, in UTF-8
// (RULINGS R2-072).
func TestPaxOnlyWhereRequired(t *testing.T) {
	tests := []struct {
		path    string
		wantPax bool
	}{
		{"a.txt", false},
		{"docs/guide/index.html", false},
		{long100, false},
		{prefix155 + "/" + long100, false},
		{"dir/" + long100, false},
		{"a b/c-d_e.f~g", false},
		{component1, true},
		{prefix156 + "/x", true},
		{"dir/" + component1, true},
		{"café", true},
		{"日本/語.md", true},
		{"emoji/🙂.txt", true},
	}
	for _, tt := range tests {
		b := write(t, file{tt.path, false, "c"})
		ms, err := members(b)
		if err != nil {
			t.Fatalf("%q: %v", tt.path, err)
		}
		if len(ms) != 1 {
			t.Fatalf("%q: %d members", tt.path, len(ms))
		}
		m := ms[0]
		if got := m.paxHeader != nil; got != tt.wantPax {
			t.Errorf("%q: extended header %v, want %v", tt.path, got, tt.wantPax)
			continue
		}
		if !tt.wantPax {
			continue
		}
		if want := []string{"path=" + tt.path}; !slices.Equal(m.pax, want) {
			t.Errorf("%q: pax records %q, want %q", tt.path, m.pax, want)
		}
		// The extended header block's own fixed fields.
		h := m.paxHeader
		for _, c := range []struct {
			name string
			err  error
		}{
			{"mode", numberIs(h, 100, 108, 0)},
			{"uid", numberIs(h, 108, 116, 0)},
			{"gid", numberIs(h, 116, 124, 0)},
			{"mtime", numberIs(h, 136, 148, 0)},
			{"uname", stringIs(h, 265, 297, nul(32))},
			{"gname", stringIs(h, 297, 329, nul(32))},
			{"magic", stringIs(h, 257, 263, "ustar\x00")},
			{"version", stringIs(h, 263, 265, "00")},
		} {
			if c.err != nil {
				t.Errorf("%q: the extended header's %s: %v", tt.path, c.name, c.err)
			}
		}
	}
}

// TestFixedPaxName is W2-13.3's fixed names: the extended header is named
// PaxHeaders.0/<file> in the file's own directory, ASCII and at most 100
// bytes, never from a process id or a random source (ADR-005 PR16,
// RULINGS R2-077), so two archives of the same files are the same bytes.
func TestFixedPaxName(t *testing.T) {
	tests := []struct{ path, want string }{
		{"café/menü.txt", "caf/PaxHeaders.0/men.txt"},
		{"日本/語.md", "/PaxHeaders.0/.md"},
		{component1, ("PaxHeaders.0/" + component1)[:100]},
		{"dir/" + component1, ("dir/PaxHeaders.0/" + component1)[:100]},
	}
	for _, tt := range tests {
		b := write(t, file{tt.path, false, "c"})
		ms, err := members(b)
		if err != nil {
			t.Fatal(err)
		}
		if got := cstr(ms[0].paxHeader[0:100]); got != tt.want {
			t.Errorf("%q: extended header named %q, want %q", tt.path, got, tt.want)
		}
		if again := write(t, file{tt.path, false, "c"}); !bytes.Equal(b, again) {
			t.Errorf("%q: two archives of the same file differ", tt.path)
		}
	}
}

// failingReader fails every read with err.
type failingReader struct{ err error }

func (r failingReader) Read([]byte) (int, error) { return 0, r.err }

// TestAddRefuses is every refusal of Add: each leaves the Writer refusing
// further calls, and writes no entry of its own.
func TestAddRefuses(t *testing.T) {
	readErr := errors.New("read failed")
	tests := []struct {
		name  string
		first string // a path added before, or ""
		path  string
		size  int64
		r     io.Reader
		want  string
	}{
		{"an empty path", "", "", 1, strings.NewReader("c"), "empty path"},
		{"a NUL", "", "a\x00b", 1, strings.NewReader("c"), "NUL"},
		{"a trailing slash", "", "dir/", 1, strings.NewReader("c"), `ends in "/"`},
		{"a repeated path", "b", "b", 1, strings.NewReader("c"), "byte order"},
		{"a path before the previous", "b", "a", 1, strings.NewReader("c"), "byte order"},
		{"upper case after lower", "a", "B", 1, strings.NewReader("c"), "byte order"},
		{"a negative size", "", "a", -1, strings.NewReader(""), "does not fit"},
		{"a size beyond ustar", "", "a", maxSize + 1, strings.NewReader(""), "does not fit"},
		{"short content", "", "a", 2, strings.NewReader("c"), "shorter than its size"},
		{"long content", "", "a", 1, strings.NewReader("cc"), "longer than its size"},
		{"a reader that fails", "", "a", 1, failingReader{readErr}, "read failed"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			w := NewWriter(&buf)
			if tt.first != "" {
				if err := w.Add(tt.first, false, 0, strings.NewReader("")); err != nil {
					t.Fatal(err)
				}
			}
			err := w.Add(tt.path, false, tt.size, tt.r)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("Add: %v, want an error containing %q", err, tt.want)
			}
			if again := w.Add("zz", false, 0, strings.NewReader("")); again == nil {
				t.Error("Add after a refusal succeeds")
			}
			if w.Close() == nil {
				t.Error("Close after a refusal succeeds")
			}
		})
	}
}

// TestAddAccepts is the control beside TestAddRefuses: the largest size
// that fits is admitted to the header check, an empty file is written, and
// byte order, not case or length, decides what may follow.
func TestAddAccepts(t *testing.T) {
	ms, err := members(write(t,
		file{"B", false, ""},
		file{"a", false, "1"},
		file{"a/b", false, "2"},
		file{"a0", false, "3"},
		file{"aa", false, "4"},
	))
	if err != nil {
		t.Fatal(err)
	}
	if len(ms) != 5 {
		t.Fatalf("%d members, want 5", len(ms))
	}
	// maxSize is ustar's own ceiling: archive/tar writes it with no pax
	// size record. Only the header is written; the content is cut short.
	var buf bytes.Buffer
	w := NewWriter(&buf)
	err = w.Add("big", false, maxSize, failingReader{io.ErrUnexpectedEOF})
	if !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Fatalf("Add of the largest size: %v, want the reader's error", err)
	}
	hdr := buf.Bytes()[:blockSize]
	if hdr[156] != '0' {
		t.Errorf("the largest size is written with typeflag %q, want 0 with no extended header", hdr[156])
	}
	if err := numberIs(hdr, 124, 136, maxSize); err != nil {
		t.Errorf("the largest size: %v", err)
	}
}

// TestAddAfterClose: a closed archive takes no further entry and is not
// closed twice.
func TestAddAfterClose(t *testing.T) {
	var buf bytes.Buffer
	w := NewWriter(&buf)
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	if w.Add("a", false, 0, strings.NewReader("")) == nil {
		t.Error("Add after Close succeeds")
	}
	if w.Close() == nil {
		t.Error("a second Close succeeds")
	}
	if buf.Len() != 2*blockSize {
		t.Errorf("%d bytes written, want the two end blocks", buf.Len())
	}
}
