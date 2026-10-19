// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package zipfmt

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"slices"
	"strings"
	"testing"

	"github.com/kottera/metais/internal/finding"
	"github.com/kottera/metais/internal/limits"
)

// codes lists the findings' codes, each with its distinguishing parameter
// when it has one, sorted.
func codes(fs []finding.Finding) []string {
	var out []string
	for _, f := range fs {
		s := string(f.Code())
		for _, p := range []string{"reason", "header_field", "limit"} {
			if v, ok := f.Param(p); ok {
				t, _ := v.Str()
				s += ":" + t
			}
		}
		out = append(out, s)
	}
	slices.Sort(out)
	return out
}

func lowered(t testing.TB, args ...string) limits.Set {
	t.Helper()
	lim := limits.Defaults()
	for _, a := range args {
		if f, ok := lim.Lower(a); !ok {
			t.Fatalf("Lower(%s): %s", a, f.Message())
		}
	}
	return lim
}

func open(t testing.TB, b []byte, lim limits.Set) (*Archive, []finding.Finding) {
	t.Helper()
	a, found, err := Open(bytes.NewReader(b), int64(len(b)), lim)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if (a == nil) == (len(found) == 0) {
		t.Fatalf("Open returned an archive %v beside %d findings", a != nil, len(found))
	}
	for _, f := range found {
		if f.Class() != finding.ClassRefused || f.Subject().Kind != finding.SubjectPackage {
			t.Errorf("%s: class %s about a %s, want refused about the package", f.Code(), f.Class(), f.Subject().Kind)
		}
		if v, ok := f.Param("offset"); ok {
			if n, _ := v.Int(); n < 0 || n >= int64(len(b)) {
				t.Errorf("%s: offset %d is outside the file's %d bytes", f.Code(), n, len(b))
			}
		}
	}
	return a, found
}

// two is a two-entry package, one deflated, one stored.
func two() []rawEntry {
	return []rawEntry{
		{name: "a.txt", content: []byte(strings.Repeat("hello, package\n", 20)), deflate: true},
		{name: "b.txt", content: []byte("stored")},
	}
}

func with(es []rawEntry, i int, f func(*rawEntry)) []rawEntry {
	f(&es[i])
	return es
}

// zip64Extra is a zip64 extended information extra field (APPNOTE
// 4.5.3) giving an entry's uncompressed and compressed sizes, both size.
func zip64Extra(size int) []byte {
	b := binary.LittleEndian.AppendUint16(nil, 0x0001)
	b = binary.LittleEndian.AppendUint16(b, 16)
	b = binary.LittleEndian.AppendUint64(b, uint64(size))
	return binary.LittleEndian.AppendUint64(b, uint64(size))
}

// zip64Locator is the 20 bytes of a zip64 end of central directory
// locator.
func zip64Locator() []byte {
	b := binary.LittleEndian.AppendUint32(nil, sigZip64Loc)
	return append(b, make([]byte, 16)...)
}

// TestOpenRefuses is the reader's negative tests, one or more per refusal
// of ADR-001 D8 steps 1 to 5 (W2-4.5): each case breaks the profile in one
// place and expects exactly the findings of the step that sees it.
func TestOpenRefuses(t *testing.T) {
	for _, c := range refusals(t) {
		lim := c.lim
		a, found := open(t, c.zip, lim)
		got := codes(found)
		want := slices.Sorted(slices.Values(c.want))
		if !slices.Equal(got, want) {
			t.Errorf("%s: found %q, want %q", c.name, got, want)
		}
		if len(want) == 0 && a == nil {
			t.Errorf("%s: refused", c.name)
		}
	}
}

// refusal is one hostile case: a zip, the limits it is opened under, and
// the findings it gives.
type refusal struct {
	name string
	zip  []byte
	lim  limits.Set
	want []string
}

func refusals(t testing.TB) []refusal {
	def := limits.Defaults()
	signature := []byte("PK\x05\x06")
	return []refusal{
		// Step 1.
		{"empty file", nil, def, []string{"package.container.not_zip"}},
		{"shorter than an end record", []byte("PK\x05\x06"), def, []string{"package.container.not_zip"}},
		{"larger than package-size", rawZip{entries: two()}.bytes(), lowered(t, "package-size=100"), []string{"limit.package:package-size"}},

		// Step 2.
		{"not a zip", bytes.Repeat([]byte("not a zip "), 10), def, []string{"package.container.not_zip"}},
		{"archive comment", rawZip{entries: two(), comment: []byte("a comment")}.bytes(), def, []string{"package.container.archive_comment"}},
		{"comment length at the end", rawZip{entries: two(), eocd: func(e *rawEOCD) { e.commentLen = 4 }}.bytes(), def, []string{"package.container.archive_comment"}},
		{"trailing bytes", rawZip{entries: two(), suffix: []byte("trailing")}.bytes(), def, []string{"package.container.trailing_data"}},
		{"comment and trailing bytes", rawZip{entries: two(), comment: []byte("c"), suffix: []byte("trailing")}.bytes(), def, []string{"package.container.trailing_data"}},
		{"disk number", rawZip{entries: two(), eocd: func(e *rawEOCD) { e.disk = 1 }}.bytes(), def, []string{"package.container.field_value:number of this disk"}},
		{"central directory disk", rawZip{entries: two(), eocd: func(e *rawEOCD) { e.cdDisk = 1 }}.bytes(), def, []string{"package.container.field_value:number of the disk with the start of the central directory"}},
		{"entries on disk differ", rawZip{entries: two(), eocd: func(e *rawEOCD) { e.onDisk = 1 }}.bytes(), def, []string{"package.container.field_value:total number of entries in the central directory on this disk"}},
		{"zip64 entry count", rawZip{entries: two(), eocd: func(e *rawEOCD) { e.total, e.onDisk = 0xFFFF, 0xFFFF }}.bytes(), def, []string{"package.container.zip64"}},
		{"zip64 offset", rawZip{entries: two(), eocd: func(e *rawEOCD) { e.cdOff = 0xFFFFFFFF }}.bytes(), def, []string{"package.container.zip64"}},
		{"zip64 locator", rawZip{entries: two(), cdTail: zip64Locator(), eocd: func(e *rawEOCD) { e.cdSize -= 20 }}.bytes(), def, []string{"package.container.zip64"}},
		{"more entries than entries", rawZip{entries: two()}.bytes(), lowered(t, "entries=1"), []string{"limit.package:entries"}},
		{"central directory short of the end record", rawZip{entries: two(), eocd: func(e *rawEOCD) { e.cdSize-- }}.bytes(), def, []string{"package.container.overlap_or_gap"}},
		{"central directory offset", rawZip{entries: two(), eocd: func(e *rawEOCD) { e.cdOff++ }}.bytes(), def, []string{"package.container.overlap_or_gap"}},
		// R1-014: a second end-record signature in the final 65,557 bytes,
		// here inside a stored entry's content.
		{"second end record in an entry", rawZip{entries: []rawEntry{{name: "a.zip", content: append(append([]byte("x"), signature...), make([]byte, 100)...)}}}.bytes(), def, []string{"package.container.second_end_record"}},
		{"second end record after the central directory", rawZip{entries: two(), cdTail: signature}.bytes(), def, []string{"package.container.second_end_record"}},

		// Step 3: the central directory header's fixed fields.
		{"version made by", rawZip{entries: with(two(), 0, func(e *rawEntry) { e.central = func(h *rawHeader) { h.madeBy = 0x0014 } })}.bytes(), def, []string{"package.container.field_value:version made by"}},
		{"symbolic link", rawZip{entries: with(two(), 0, func(e *rawEntry) { e.central = func(h *rawHeader) { h.external = 0xA1FF0000 } })}.bytes(), def, []string{"package.container.field_value:external file attributes"}},
		{"directory mode", rawZip{entries: with(two(), 0, func(e *rawEntry) { e.central = func(h *rawHeader) { h.external = 0x41ED0000 } })}.bytes(), def, []string{"package.container.field_value:external file attributes"}},
		{"setuid", rawZip{entries: with(two(), 0, func(e *rawEntry) { e.central = func(h *rawHeader) { h.external = 0x89A40000 } })}.bytes(), def, []string{"package.container.field_value:external file attributes"}},
		{"disk number start", rawZip{entries: with(two(), 0, func(e *rawEntry) { e.central = func(h *rawHeader) { h.disk = 1 } })}.bytes(), def, []string{"package.container.field_value:disk number start"}},
		{"internal attributes", rawZip{entries: with(two(), 0, func(e *rawEntry) { e.central = func(h *rawHeader) { h.internal = 1 } })}.bytes(), def, []string{"package.container.field_value:internal file attributes"}},
		{"file comment", rawZip{entries: with(two(), 1, func(e *rawEntry) { e.cdComment = []byte("hi") })}.bytes(), def, []string{"package.container.field_value:file comment length"}},
		{"central extra field", rawZip{entries: with(two(), 1, func(e *rawEntry) { e.cdExtra = []byte{0x55, 0x54, 1, 0, 0} })}.bytes(), def, []string{"package.container.extra_field"}},
		// ADR-001 D1 item 2: a zip64 extra field is a zip64 structure, and an
		// extra field, whatever the sizes beside it hold.
		{"zip64 extra field", rawZip{entries: with(two(), 1, func(e *rawEntry) {
			e.cdExtra = zip64Extra(len(e.content))
			e.localExtra = e.cdExtra
		})}.bytes(), def, []string{"package.container.extra_field", "package.container.zip64"}},
		{"zip64 extra field after another", rawZip{entries: with(two(), 1, func(e *rawEntry) {
			e.cdExtra = append([]byte{0x55, 0x54, 1, 0, 0}, zip64Extra(len(e.content))...)
			e.localExtra = e.cdExtra
		})}.bytes(), def, []string{"package.container.extra_field", "package.container.zip64"}},
		{"extra field not of records", rawZip{entries: with(two(), 1, func(e *rawEntry) {
			e.cdExtra = []byte{0x01, 0x00, 0xFF}
		})}.bytes(), def, []string{"package.container.extra_field"}},
		{"time", rawZip{entries: with(two(), 0, func(e *rawEntry) { e.central = func(h *rawHeader) { h.time = 1 } })}.bytes(), def, []string{"package.container.field_value:last mod file time"}},
		{"date", rawZip{entries: with(two(), 0, func(e *rawEntry) { e.central = func(h *rawHeader) { h.date = 0x5821 } })}.bytes(), def, []string{"package.container.field_value:last mod file date"}},
		{"version needed", rawZip{entries: with(two(), 1, func(e *rawEntry) { e.central = func(h *rawHeader) { h.needed = 20 } })}.bytes(), def, []string{"package.container.field_value:version needed to extract"}},
		{"encryption bit 0", rawZip{entries: with(two(), 1, func(e *rawEntry) { e.central = func(h *rawHeader) { h.flags |= 1 } })}.bytes(), def, []string{"package.container.encrypted"}},
		{"strong encryption bit 6", rawZip{entries: with(two(), 1, func(e *rawEntry) { e.central = func(h *rawHeader) { h.flags |= 1 << 6 } })}.bytes(), def, []string{"package.container.encrypted"}},
		{"central directory encryption bit 13", rawZip{entries: with(two(), 1, func(e *rawEntry) { e.central = func(h *rawHeader) { h.flags |= 1 << 13 } })}.bytes(), def, []string{"package.container.encrypted"}},
		{"AES method 99", rawZip{entries: with(two(), 1, func(e *rawEntry) { e.central = func(h *rawHeader) { h.method = 99 } })}.bytes(), def, []string{"package.container.encrypted"}},
		{"method", rawZip{entries: with(two(), 1, func(e *rawEntry) { e.central = func(h *rawHeader) { h.method = 12 } })}.bytes(), def, []string{"package.container.method"}},
		{"data descriptor", rawZip{entries: with(two(), 1, func(e *rawEntry) { e.central = func(h *rawHeader) { h.flags |= 1 << 3 } })}.bytes(), def, []string{"package.container.data_descriptor"}},
		{"no UTF-8 flag", rawZip{entries: with(two(), 1, func(e *rawEntry) { e.central = func(h *rawHeader) { h.flags = 0 } })}.bytes(), def, []string{"package.container.flags"}},
		{"another flag", rawZip{entries: with(two(), 1, func(e *rawEntry) { e.central = func(h *rawHeader) { h.flags |= 1 << 1 } })}.bytes(), def, []string{"package.container.flags"}},
		{"zip64 size", rawZip{entries: with(two(), 1, func(e *rawEntry) { e.central = func(h *rawHeader) { h.usize = 0xFFFFFFFF } })}.bytes(), def, []string{"package.container.zip64"}},
		{"zip64 relative offset", rawZip{entries: with(two(), 1, func(e *rawEntry) { e.central = func(h *rawHeader) { h.offset = 0xFFFFFFFF } })}.bytes(), def, []string{"package.container.zip64"}},
		{"stored sizes differ", rawZip{entries: with(two(), 1, func(e *rawEntry) { e.central = func(h *rawHeader) { h.usize++ } })}.bytes(), def, []string{"package.container.size_mismatch"}},
		{"empty entry deflated", rawZip{entries: []rawEntry{{name: "a", deflate: true}}}.bytes(), def, []string{"package.container.field_value:compression method"}},
		{"central directory holds fewer headers", rawZip{entries: two(), eocd: func(e *rawEOCD) { e.total, e.onDisk = 3, 3 }}.bytes(), def, []string{"package.container.overlap_or_gap"}},
		{"central directory holds more", rawZip{entries: two(), eocd: func(e *rawEOCD) { e.total, e.onDisk = 1, 1 }}.bytes(), def, []string{"package.container.overlap_or_gap"}},
		{"bad central signature", rawZip{entries: two(), cdTail: make([]byte, 46), eocd: func(e *rawEOCD) { e.total, e.onDisk = 3, 3 }}.bytes(), def, []string{"package.container.field_value:central file header signature"}},

		// Step 3: the declared sizes against the limits.
		{"entry-size", rawZip{entries: two()}.bytes(), lowered(t, "entry-size=100"), []string{"limit.package:entry-size"}},
		{"package-uncompressed", rawZip{entries: two()}.bytes(), lowered(t, "package-uncompressed=300"), []string{"limit.package:package-uncompressed"}},
		{"entry-ratio", rawZip{entries: []rawEntry{{name: "zeros", content: make([]byte, 2<<20), deflate: true}}}.bytes(), def, []string{"limit.package:entry-ratio"}},
		{"entry-ratio of no data", rawZip{entries: []rawEntry{{name: "zeros", content: make([]byte, 2<<20), deflate: true, data: []byte{}}}}.bytes(), def, []string{"limit.package:entry-ratio"}},

		// Step 3: the names (ADR-001 D2).
		{"invalid UTF-8", rawZip{entries: []rawEntry{{name: "a\xff"}}}.bytes(), def, []string{"package.name.encoding:invalid_utf8"}},
		{"byte order mark", rawZip{entries: []rawEntry{{name: "\uFEFFa"}}}.bytes(), def, []string{"package.name.encoding:byte_order_mark"}},
		{"not NFC", rawZip{entries: []rawEntry{{name: "cafe\u0301"}}}.bytes(), def, []string{"package.name.not_nfc"}},
		{"empty name", rawZip{entries: []rawEntry{{name: ""}}}.bytes(), def, []string{"package.name.shape:empty"}},
		{"absolute", rawZip{entries: []rawEntry{{name: "/etc/passwd"}}}.bytes(), def, []string{"package.name.shape:leading_slash"}},
		{"traversal", rawZip{entries: []rawEntry{{name: "../evil"}}}.bytes(), def, []string{"package.name.shape:dot_segment"}},
		{"backslash", rawZip{entries: []rawEntry{{name: `a\b`}}}.bytes(), def, []string{"package.name.shape:backslash"}},
		{"drive letter", rawZip{entries: []rawEntry{{name: "C:/a"}}}.bytes(), def, []string{"package.name.shape:colon"}},
		{"directory entry", rawZip{entries: []rawEntry{{name: "a/"}}}.bytes(), def, []string{"package.name.directory_entry"}},
		{"bidirectional override", rawZip{entries: []rawEntry{{name: "a\u202etxt.exe"}}}.bytes(), def, []string{"package.name.character"}},
		{"control", rawZip{entries: []rawEntry{{name: "a\x1bb"}}}.bytes(), def, []string{"package.name.character"}},
		{"reserved device name", rawZip{entries: []rawEntry{{name: "aux.c"}}}.bytes(), def, []string{"package.name.not_portable:reserved_name"}},
		{".git", rawZip{entries: []rawEntry{{name: "templates/.GIT/config"}}}.bytes(), def, []string{"package.name.not_portable:git"}},
		{"segment end", rawZip{entries: []rawEntry{{name: "a./b"}}}.bytes(), def, []string{"package.name.not_portable:segment_end"}},
		{"long segment", rawZip{entries: []rawEntry{{name: strings.Repeat("a", 256)}}}.bytes(), def, []string{"package.name.not_portable:segment_length"}},
		{"path-length", rawZip{entries: []rawEntry{{name: strings.Repeat("a/", 10) + "a"}}}.bytes(), lowered(t, "path-length=20"), []string{"limit.package:path-length"}},
		{"path-depth", rawZip{entries: []rawEntry{{name: strings.Repeat("a/", 10) + "a"}}}.bytes(), lowered(t, "path-depth=10"), []string{"limit.package:path-depth"}},
		{"out of order", rawZip{entries: []rawEntry{{name: "b"}, {name: "a"}}}.bytes(), def, []string{"package.container.entry_order"}},
		{"duplicate", rawZip{entries: []rawEntry{{name: "a"}, {name: "a"}}}.bytes(), def, []string{"package.name.duplicate"}},
		{"case collision", rawZip{entries: []rawEntry{{name: "README.md"}, {name: "Readme.md"}}}.bytes(), def, []string{"package.name.case_collision"}},
		{"full case folding", rawZip{entries: []rawEntry{{name: "STRASSE"}, {name: "straße"}}}.bytes(), def, []string{"package.name.case_collision"}},
		{"directory case collision", rawZip{entries: []rawEntry{{name: "Docs/a"}, {name: "docs/b"}}}.bytes(), def, []string{"package.name.case_collision"}},
		{"file and directory", rawZip{entries: []rawEntry{{name: "a"}, {name: "a/b"}}}.bytes(), def, []string{"package.name.file_directory_conflict"}},

		// Step 4: the walk.
		{"prepended bytes", rawZip{prefix: []byte("MZ stub"), entries: two()}.bytes(), def, []string{"package.container.prepended_data"}},
		{"gap between entries", rawZip{entries: with(two(), 0, func(e *rawEntry) { e.gapAfter = []byte("hidden") })}.bytes(), def, []string{"package.container.overlap_or_gap"}},
		{"gap before the central directory", rawZip{entries: with(two(), 1, func(e *rawEntry) { e.gapAfter = []byte("hidden") })}.bytes(), def, []string{"package.container.overlap_or_gap"}},
		{"overlapping entries", overlapping(), def, []string{"package.container.overlap_or_gap"}},
		{"local version needed", rawZip{entries: with(two(), 1, func(e *rawEntry) { e.local = func(h *rawHeader) { h.needed = 20 } })}.bytes(), def, []string{"package.container.header_mismatch:version needed to extract"}},
		{"local flags", rawZip{entries: with(two(), 1, func(e *rawEntry) { e.local = func(h *rawHeader) { h.flags |= 1 << 3 } })}.bytes(), def, []string{"package.container.header_mismatch:general purpose bit flag"}},
		{"local method", rawZip{entries: with(two(), 1, func(e *rawEntry) { e.local = func(h *rawHeader) { h.method = 8 } })}.bytes(), def, []string{"package.container.header_mismatch:compression method"}},
		{"local time", rawZip{entries: with(two(), 1, func(e *rawEntry) { e.local = func(h *rawHeader) { h.time = 7 } })}.bytes(), def, []string{"package.container.header_mismatch:last mod file time"}},
		{"local date", rawZip{entries: with(two(), 1, func(e *rawEntry) { e.local = func(h *rawHeader) { h.date = 0x22 } })}.bytes(), def, []string{"package.container.header_mismatch:last mod file date"}},
		{"local CRC", rawZip{entries: with(two(), 1, func(e *rawEntry) { e.local = func(h *rawHeader) { h.crc = 0 } })}.bytes(), def, []string{"package.container.header_mismatch:crc-32"}},
		{"local sizes", rawZip{entries: with(two(), 1, func(e *rawEntry) {
			e.local = func(h *rawHeader) { h.csize, h.usize = 0, 0 }
		})}.bytes(), def, []string{"package.container.header_mismatch:compressed size", "package.container.header_mismatch:uncompressed size"}},
		{"local name", rawZip{entries: with(two(), 1, func(e *rawEntry) { e.localName = "c.txt" })}.bytes(), def, []string{"package.container.header_mismatch:file name"}},
		{"local name length", rawZip{entries: with(two(), 1, func(e *rawEntry) { e.localName = "b.txt2" })}.bytes(), def, []string{"package.container.overlap_or_gap", "package.container.header_mismatch:file name length"}},
		{"local extra field", rawZip{entries: with(two(), 1, func(e *rawEntry) { e.localExtra = []byte{0x55, 0x54, 1, 0, 0} })}.bytes(), def, []string{"package.container.overlap_or_gap", "package.container.header_mismatch:extra field length"}},

		// Step 5: the data.
		{"CRC", rawZip{entries: with(two(), 1, func(e *rawEntry) { e.central = func(h *rawHeader) { h.crc++ }; e.local = e.central })}.bytes(), def, []string{"package.container.crc_mismatch"}},
		{"CRC recorded as zero", rawZip{entries: with(two(), 1, func(e *rawEntry) { e.central = func(h *rawHeader) { h.crc = 0 }; e.local = e.central })}.bytes(), def, []string{"package.container.crc_mismatch"}},
		{"deflated longer than declared", rawZip{entries: with(two(), 0, func(e *rawEntry) { e.central = func(h *rawHeader) { h.usize-- }; e.local = e.central })}.bytes(), def, []string{"package.container.size_mismatch"}},
		{"deflated shorter than declared", rawZip{entries: with(two(), 0, func(e *rawEntry) { e.central = func(h *rawHeader) { h.usize++ }; e.local = e.central })}.bytes(), def, []string{"package.container.size_mismatch"}},
		{"invalid stream", rawZip{entries: with(two(), 0, func(e *rawEntry) { e.data = []byte{0xff, 0xff, 0xff, 0xff} })}.bytes(), def, []string{"package.container.deflate_stream:invalid"}},
		{"truncated stream", rawZip{entries: with(two(), 0, func(e *rawEntry) { e.data = deflated(e.content)[:10] })}.bytes(), def, []string{"package.container.deflate_stream:truncated"}},
		{"bytes after the stream", rawZip{entries: with(two(), 0, func(e *rawEntry) { e.data = append(deflated(e.content), "hidden"...) })}.bytes(), def, []string{"package.container.deflate_stream:trailing_bytes"}},
	}
}

// overlapping is two central directory entries naming one local header,
// Fifield's construction.
func overlapping() []byte {
	b := rawZip{entries: []rawEntry{{name: "a", content: []byte("same")}, {name: "b", content: []byte("same")}}}.bytes()
	// Point the second central header's relative offset at the first
	// entry's local header.
	cd := bytes.LastIndex(b, []byte("PK\x01\x02"))
	binary.LittleEndian.PutUint32(b[cd+42:], 0)
	return b
}

// TestOpenAccepts is the control of TestOpenRefuses: the profile at its
// edges opens.
func TestOpenAccepts(t *testing.T) {
	def := limits.Defaults()
	far := append(append([]byte("x"), "PK\x05\x06"...), make([]byte, endSearch)...)
	for name, z := range map[string]rawZip{
		"no entries":      {},
		"one empty entry": {entries: []rawEntry{{name: "a"}}},
		// R1-014's boundary: a signature further than 65,557 bytes from the
		// end is no second end record.
		"signature beyond the search": {entries: []rawEntry{{name: "a.zip", content: far}}},
		// entry-ratio applies only above 1 MiB.
		"small entry at a high ratio": {entries: []rawEntry{{name: "zeros", content: make([]byte, 1<<20), deflate: true}}},
		"unicode names":               {entries: []rawEntry{{name: "caf\u00e9/\u4e2d.txt"}, {name: "\u00e9t\u00e9"}}},
	} {
		if _, found := open(t, z.bytes(), def); len(found) > 0 {
			t.Errorf("%s: refused with %q", name, codes(found))
		}
	}
}

// signatureFromEnd is a one-entry package whose stored content holds an
// end-record signature dist bytes before the end of the file.
func signatureFromEnd(t testing.TB, dist int) []byte {
	t.Helper()
	build := func(pad int) []byte {
		content := append([]byte("PK\x05\x06"), make([]byte, pad)...)
		return rawZip{entries: []rawEntry{{name: "a.zip", content: content}}}.bytes()
	}
	b := build(0)
	pad := dist - (len(b) - bytes.Index(b, []byte("PK\x05\x06")))
	if pad < 0 {
		t.Fatalf("no package holds a signature %d bytes from its end", dist)
	}
	b = build(pad)
	if got := len(b) - bytes.Index(b, []byte("PK\x05\x06")); got != dist {
		t.Fatalf("signature %d bytes from the end, want %d", got, dist)
	}
	return b
}

// only returns the single finding of code in found, failing the test when
// there is not exactly one, and its offset.
func only(t testing.TB, found []finding.Finding, code finding.Code) int64 {
	t.Helper()
	var n []int64
	for _, f := range found {
		if f.Code() == code {
			off, _ := f.Param("offset")
			v, _ := off.Int()
			n = append(n, v)
		}
	}
	if len(n) != 1 {
		t.Fatalf("found %q, want one %s", codes(found), code)
	}
	return n[0]
}

// TestOpenEndSearchBoundary pins ADR-001 D8 step 2's window, R1-014's
// final 65,557 bytes, at both its edges and within the end record itself.
func TestOpenEndSearchBoundary(t *testing.T) {
	def := limits.Defaults()

	// The window's first byte is the file's size less 65,557.
	b := signatureFromEnd(t, 65_557)
	_, found := open(t, b, def)
	if off := only(t, found, CodeSecondEndRecord); off != int64(len(b)-65_557) {
		t.Errorf("second end record at offset %d, want %d", off, len(b)-65_557)
	}

	// One byte further from the end lies outside it.
	if _, found := open(t, signatureFromEnd(t, 65_558), def); len(found) > 0 {
		t.Errorf("a signature 65,558 bytes from the end: refused with %q", codes(found))
	}

	// A maximum-length comment puts the end record at the window's first
	// byte, where it is found and its comment's length reported.
	b = rawZip{entries: two(), comment: bytes.Repeat([]byte("c"), 0xFFFF)}.bytes()
	_, found = open(t, b, def)
	if off := only(t, found, CodeArchiveComment); off != int64(len(b)-65_557) {
		t.Errorf("archive comment at offset %d, want %d", off, len(b)-65_557)
	}
	length, _ := found[0].Param("length")
	if n, _ := length.Int(); n != 0xFFFF {
		t.Errorf("archive comment length %d, want 65535", n)
	}

	// A signature in the end record's own fields, after its signature.
	for _, field := range []int{4, 8, 12, 16} {
		b := rawZip{entries: two()}.bytes()
		copy(b[len(b)-eocdLen+field:], "PK\x05\x06")
		_, found := open(t, b, def)
		if off := only(t, found, CodeSecondEndRecord); off != int64(len(b)-eocdLen+field) {
			t.Errorf("field %d: second end record at offset %d, want %d", field, off, len(b)-eocdLen+field)
		}
	}
}

// TestOpenReportsEveryFindingOfAStep holds the reader to ADR-001 D8 item 2:
// within a step every independent finding is reported, and a later step
// does not run once one has refused.
func TestOpenReportsEveryFindingOfAStep(t *testing.T) {
	z := rawZip{entries: []rawEntry{
		{name: "CON", content: []byte("x"), central: func(h *rawHeader) { h.time = 1 }},
		{name: "a\x00", content: []byte("y"), central: func(h *rawHeader) { h.crc = 0 }},
		{name: "a/", content: []byte("z")},
		{name: "b", content: []byte("w"), central: func(h *rawHeader) { h.external = 0 }},
	}}
	_, found := open(t, z.bytes(), limits.Defaults())
	want := []string{
		"package.container.field_value:external file attributes", "package.container.field_value:last mod file time",
		"package.name.character", "package.name.directory_entry", "package.name.not_portable:reserved_name",
	}
	if got := codes(found); !slices.Equal(got, want) {
		t.Errorf("found %q, want %q: step 3's findings, and not step 5's CRC", got, want)
	}

	// Two local headers that each differ from their central headers.
	z = rawZip{entries: []rawEntry{
		{name: "a", content: []byte("x"), local: func(h *rawHeader) { h.time = 1 }},
		{name: "b", content: []byte("y"), local: func(h *rawHeader) { h.date = 0 }},
	}}
	_, found = open(t, z.bytes(), limits.Defaults())
	if got := codes(found); len(got) != 2 {
		t.Errorf("two mismatched local headers gave %q, want two header_mismatch findings", got)
	}
}

// TestOpenFindingLocations holds the findings to ADR-001's location rule:
// the entry name, and the field's APPNOTE name and offset.
func TestOpenFindingLocations(t *testing.T) {
	b := rawZip{entries: with(two(), 1, func(e *rawEntry) { e.central = func(h *rawHeader) { h.time = 9 } })}.bytes()
	_, found := open(t, b, limits.Defaults())
	if len(found) != 1 {
		t.Fatalf("found %q", codes(found))
	}
	f := found[0]
	loc, _ := f.Location()
	field, _ := f.Param("header_field")
	off, _ := f.Param("offset")
	name, _ := field.Str()
	n, _ := off.Int()
	if loc.Origin != finding.OriginPackage || loc.File != "b.txt" || name != "last mod file time" ||
		binary.LittleEndian.Uint16(b[n:]) != 9 {
		t.Errorf("finding at %+v, field %q, offset %d", loc, name, n)
	}
}

// TestOpenRelatesCollisions names the earlier entry of a collision.
func TestOpenRelatesCollisions(t *testing.T) {
	_, found := open(t, rawZip{entries: []rawEntry{{name: "Docs/a"}, {name: "docs/b"}}}.bytes(), limits.Defaults())
	if len(found) != 1 {
		t.Fatalf("found %q", codes(found))
	}
	rel := found[0].Related()
	loc, _ := found[0].Location()
	member, _ := found[0].Param("member")
	m, _ := member.Str()
	if loc.File != "docs/b" || m != "docs" || len(rel) != 1 || rel[0].Location.File != "Docs/a" {
		t.Errorf("collision at %q member %q related %+v", loc.File, m, rel)
	}
}

// failing is a ReaderAt that fails at and after one offset.
type failing struct {
	b  []byte
	at int64
}

func (f failing) ReadAt(p []byte, off int64) (int, error) {
	if off+int64(len(p)) > f.at {
		return 0, errors.New("disk on fire")
	}
	return bytes.NewReader(f.b).ReadAt(p, off)
}

// TestOpenReaderErrors keeps a failure of the file apart from the
// package's findings.
func TestOpenReaderErrors(t *testing.T) {
	b := rawZip{entries: two()}.bytes()
	for _, at := range []int64{0, 10, int64(len(b)) - 10} {
		a, found, err := Open(failing{b, at}, int64(len(b)), limits.Defaults())
		if err == nil || a != nil || found != nil {
			t.Errorf("a reader failing at %d: archive %v, findings %q, error %v", at, a != nil, codes(found), err)
		}
	}
	// A size larger than the file is a short read.
	if _, _, err := Open(bytes.NewReader(b), int64(len(b))+5, limits.Defaults()); err == nil {
		t.Error("a size beyond the file gave no error")
	}
}

// TestArchiveDetectsChange holds the packed tree to its contract: bytes
// beneath it that are no longer those it verified give an error from
// Read, never different content.
func TestArchiveDetectsChange(t *testing.T) {
	big := bytes.Repeat([]byte("0123456789abcdef"), 8000) // beyond the tail
	b := rawZip{entries: []rawEntry{{name: "a", content: big}, {name: "b", content: []byte("tail")}}}.bytes()
	file := bytes.Clone(b)
	a, found := open(t, file, limits.Defaults())
	if len(found) > 0 {
		t.Fatalf("found %q", codes(found))
	}
	file[100] ^= 1
	r, err := a.Open("a")
	if err != nil {
		t.Fatal(err)
	}
	if n, err := r.Read(make([]byte, 10)); n != 0 || !errors.Is(err, ErrChanged) {
		t.Errorf("Read of a changed entry gave %d bytes and %v, want none and ErrChanged", n, err)
	}
	file[100] ^= 1
	r, _ = a.Open("a")
	if got, err := io.ReadAll(r); err != nil || !bytes.Equal(got, big) {
		t.Errorf("Read of the restored entry: %v", err)
	}
}

// TestOpenContainer is ADR-005 D8 item 8: a container's zip is read under
// the package profile and every bound, but in the order its entries are
// written, which Open refuses as package.container.entry_order; each
// entry says whether it is deflated.
func TestOpenContainer(t *testing.T) {
	text := []byte(strings.Repeat("a part that deflates\n", 40))
	b := write(t, []Entry{
		{Name: "mimetype", Content: []byte("application/epub+zip"), Method: Store},
		{Name: "META-INF/container.xml", Content: text, Method: Auto},
		{Name: "a.txt", Content: nil, Method: Auto},
	})
	if _, fs, err := Open(bytes.NewReader(b), int64(len(b)), limits.Defaults()); err != nil || !slices.ContainsFunc(fs, func(f finding.Finding) bool { return f.Code() == CodeEntryOrder }) {
		t.Fatalf("Open of an unsorted zip: %v, %v", fs, err)
	}
	got := map[string][]byte{}
	a, fs, err := OpenContainer(bytes.NewReader(b), int64(len(b)), limits.Defaults(), func(name string) io.Writer {
		var buf bytes.Buffer
		got[name] = nil
		return writerFunc(func(p []byte) (int, error) {
			buf.Write(p)
			got[name] = buf.Bytes()
			return len(p), nil
		})
	})
	if err != nil || len(fs) > 0 {
		t.Fatalf("OpenContainer: %v, %v", fs, err)
	}
	var names []string
	var deflated []bool
	for _, e := range a.Entries() {
		names, deflated = append(names, e.Name), append(deflated, e.Deflated)
	}
	if want := []string{"mimetype", "META-INF/container.xml", "a.txt"}; !slices.Equal(names, want) || !slices.Equal(a.List(), want) {
		t.Errorf("entries %q, want %q in the file's order", names, want)
	}
	if !slices.Equal(deflated, []bool{false, true, false}) {
		t.Errorf("deflated %v", deflated)
	}
	if !bytes.Equal(got["META-INF/container.xml"], text) {
		t.Errorf("the sink received %q", got["META-INF/container.xml"])
	}
	// An entry that is a stored zip holds an end-record signature, which
	// Open refuses and OpenContainer does not.
	nested := write(t, []Entry{{Name: "inner.zip", Content: b, Method: Store}, {Name: "a.txt", Content: nil, Method: Auto}})
	if _, fs, _ := Open(bytes.NewReader(nested), int64(len(nested)), limits.Defaults()); len(fs) == 0 || fs[0].Code() != CodeSecondEndRecord {
		t.Errorf("Open of a zip holding a stored zip: %v", fs)
	}
	if _, fs, err := OpenContainer(bytes.NewReader(nested), int64(len(nested)), limits.Defaults(), nil); err != nil || len(fs) > 0 {
		t.Errorf("OpenContainer of a zip holding a stored zip: %v, %v", fs, err)
	}
	// Every other rule holds: a bound, and a refusal of the profile.
	low := limits.Defaults()
	if _, ok := low.Lower("entries=2"); !ok {
		t.Fatal("entries=2 is refused")
	}
	if _, fs, _ := OpenContainer(bytes.NewReader(b), int64(len(b)), low, nil); len(fs) == 0 || fs[0].Code() != finding.CodeLimitPackage {
		t.Errorf("OpenContainer within entries=2: %v", fs)
	}
	if _, fs, _ := OpenContainer(bytes.NewReader(b[1:]), int64(len(b)-1), limits.Defaults(), nil); len(fs) == 0 {
		t.Error("OpenContainer opened a zip with a byte cut from its start")
	}
}

type writerFunc func([]byte) (int, error)

func (f writerFunc) Write(p []byte) (int, error) { return f(p) }
