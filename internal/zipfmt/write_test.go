// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package zipfmt

import (
	"archive/zip"
	"bytes"
	"encoding/binary"
	"hash/crc32"
	"io"
	"strings"
	"testing"

	"github.com/kottera/metais/internal/limits"
	"github.com/kottera/metais/internal/pkgtree/pkgtreetest"
)

// sample is a package-shaped set of entries in ADR-001 D1 item 7's order:
// text that deflates, bytes that do not, an empty entry, and an entry the
// caller asks to store.
func sample() []Entry {
	noise := make([]byte, 3000)
	x := uint32(1)
	for i := range noise {
		x ^= x << 13
		x ^= x >> 17
		x ^= x << 5
		noise[i] = byte(x)
	}
	return []Entry{
		{Name: ".compiled/digests.json", Content: []byte(`{"algorithm":"sha256"}`), Method: Auto},
		{Name: "assets/empty.txt", Content: nil, Method: Auto},
		{Name: "assets/noise.bin", Content: noise, Method: Auto},
		{Name: "metais.yaml", Content: []byte(strings.Repeat("metais: \"1.0\"\nname: example\n", 40)), Method: Auto},
		{Name: "templates/report.md.liquid", Content: []byte(strings.Repeat("# {{ title }}\n", 30)), Method: Store},
	}
}

func write(t testing.TB, entries []Entry) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := Write(&buf, entries); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

// TestWriteMatchesCreateRaw compares the writer's bytes with what
// archive/zip's CreateRaw writes for the same headers and data (ADR-010
// D11 item 2): the profile is a zip any reader opens, and Write writes it
// directly.
func TestWriteMatchesCreateRaw(t *testing.T) {
	entries := sample()
	got := write(t, entries)

	var want bytes.Buffer
	zw := zip.NewWriter(&want)
	for _, e := range entries {
		h, data, err := encode(e)
		if err != nil {
			t.Fatal(err)
		}
		w, err := zw.CreateRaw(&zip.FileHeader{
			Name: e.Name, CreatorVersion: versionMadeBy, ReaderVersion: versionNeeded(h.method),
			Flags: flagUTF8, Method: h.method, ModifiedTime: dosTime, ModifiedDate: dosDate,
			CRC32: h.crc, CompressedSize64: uint64(h.csize), UncompressedSize64: uint64(h.usize),
			ExternalAttrs: externalAttrs,
		})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write(data); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want.Bytes()) {
		t.Errorf("Write wrote %d bytes that differ from CreateRaw's %d", len(got), want.Len())
	}
}

// TestWriteFields reads back every field D1 fixes, from the bytes, and the
// method rule of D4 item 2.
func TestWriteFields(t *testing.T) {
	entries := sample()
	b := write(t, entries)
	a, found, err := Open(bytes.NewReader(b), int64(len(b)), limits.Defaults())
	if err != nil || len(found) > 0 {
		t.Fatalf("Open of Write's output: %v %v", err, codes(found))
	}
	wantMethod := map[string]uint16{
		".compiled/digests.json":     methodStored, // 22 bytes do not deflate shorter
		"assets/empty.txt":           methodStored, // an empty entry is stored
		"assets/noise.bin":           methodStored, // noise does not deflate shorter
		"metais.yaml":                methodDeflated,
		"templates/report.md.liquid": methodStored, // asked to store
	}
	for i, e := range a.entries {
		if e.method != wantMethod[e.name] {
			t.Errorf("%s: method %d, want %d", e.name, e.method, wantMethod[e.name])
		}
		if e.versNeeded != versionNeeded(e.method) || e.flags != flagUTF8 || e.time != 0 || e.date != 0x21 || e.extraLen != 0 {
			t.Errorf("%s: header %+v breaks D1 item 3", e.name, e)
		}
		if e.crc != crc32.ChecksumIEEE(entries[i].Content) || int(e.usize) != len(entries[i].Content) {
			t.Errorf("%s: CRC or size is not the content's", e.name)
		}
	}
	// The end record is the last 22 bytes, with no comment.
	if !bytes.Equal(b[len(b)-22:len(b)-18], []byte("PK\x05\x06")) || !bytes.Equal(b[len(b)-2:], []byte{0, 0}) {
		t.Error("the end record is not the file's last 22 bytes with no comment")
	}
}

// TestWriteDeterministic writes the same entries twice, and with content
// slices that are copies, and expects the same bytes: nothing but the
// entries reaches the output (ADR-001 D4 items 4 and 6).
func TestWriteDeterministic(t *testing.T) {
	a := write(t, sample())
	copied := sample()
	for i := range copied {
		copied[i].Content = bytes.Clone(copied[i].Content)
	}
	if b := write(t, copied); !bytes.Equal(a, b) {
		t.Error("two writes of the same entries differ")
	}
}

// TestWriteRefuses holds Write to what the format cannot carry.
func TestWriteRefuses(t *testing.T) {
	for name, entries := range map[string][]Entry{
		"empty name":  {{Name: "", Method: Auto}},
		"no method":   {{Name: "a"}},
		"long name":   {{Name: strings.Repeat("a", 0x10000), Method: Store}},
		"65535 names": make([]Entry, maxEntries+1),
	} {
		if err := Write(io.Discard, entries); err == nil {
			t.Errorf("%s: Write succeeded", name)
		}
	}
}

// TestWriteRawBuilder pins the test's raw builder to the writer, so a
// rawZip with no changes is the profile and each hostile case breaks only
// what it changes.
func TestWriteRawBuilder(t *testing.T) {
	entries := sample()
	var raw rawZip
	for _, e := range entries {
		h, _, err := encode(e)
		if err != nil {
			t.Fatal(err)
		}
		raw.entries = append(raw.entries, rawEntry{name: e.Name, content: e.Content, deflate: h.method == methodDeflated})
	}
	if !bytes.Equal(raw.bytes(), write(t, entries)) {
		t.Error("the raw builder's unchanged zip is not the writer's")
	}
}

// TestPackedTreeContract holds the packed tree to pkgtree.Tree's contract
// (W2-4.7).
func TestPackedTreeContract(t *testing.T) {
	entries := sample()
	want := map[string][]byte{}
	for _, e := range entries {
		want[e.Name] = e.Content
	}
	b := write(t, entries)
	a, found, err := Open(bytes.NewReader(b), int64(len(b)), limits.Defaults())
	if err != nil || len(found) > 0 {
		t.Fatalf("Open: %v %v", err, codes(found))
	}
	if err := pkgtreetest.Check(a, want); err != nil {
		t.Error(err)
	}
	// The control: the tree with one entry's bytes not the wanted ones is
	// caught.
	want["metais.yaml"] = []byte("other")
	if pkgtreetest.Check(a, want) == nil {
		t.Error("Check passed the packed tree against content it does not hold")
	}
}

// TestLimitsWithinCeilings holds ADR-009's defaults below ADR-001 D3's
// ceilings, so no package within the limits needs zip64 (D3 item 2).
func TestLimitsWithinCeilings(t *testing.T) {
	lim := limits.Defaults()
	for _, c := range []struct {
		id      limits.ID
		ceiling int64
	}{
		{limits.PackageSize, maxSize},
		{limits.PackageUncompressed, maxSize},
		{limits.EntrySize, maxSize},
		{limits.Entries, maxEntries},
	} {
		if v := lim.Value(c.id); v > c.ceiling {
			t.Errorf("%s defaults to %d, above the ceiling %d", limits.Get(c.id).Name, v, c.ceiling)
		}
	}
}

// TestWritePartsStreams is ADR-005 D13 item 5 for the writer: a part's
// content is written through to the zip as it is produced, never held
// whole, stored or deflated, and the zip is the one Write writes for the
// same bytes. A zip written as a part of another is that zip's bytes.
func TestWritePartsStreams(t *testing.T) {
	const chunk, chunks = 8 << 10, 512 // 4 MiB of each
	text := []byte(strings.Repeat("a line of text that deflates well\n", chunk/34+1))[:chunk]
	content := func(w io.Writer) error {
		for range chunks {
			if _, err := w.Write(text); err != nil {
				return err
			}
		}
		return nil
	}
	for _, m := range []Method{Store, Auto} {
		var out bytes.Buffer
		w := &largestWrite{w: &out}
		if _, err := WriteParts(w, []Part{{Name: "big.txt", Method: m, Content: content}}); err != nil {
			t.Fatal(err)
		}
		if w.max > 2*chunk {
			t.Errorf("method %d: a write of %d bytes reached the zip: the entry was held, not streamed", m, w.max)
		}
		want := write(t, []Entry{{Name: "big.txt", Content: bytes.Repeat(text, chunks), Method: m}})
		if !bytes.Equal(out.Bytes(), want) {
			t.Errorf("method %d: WriteParts wrote other bytes than Write", m)
		}
	}
	inner := write(t, sample())
	var outer bytes.Buffer
	written, err := WriteParts(&outer, []Part{{Name: "inner.zip", Method: Auto, Content: func(w io.Writer) error {
		parts := make([]Part, 0, len(sample()))
		for _, e := range sample() {
			parts = append(parts, Part{Name: e.Name, Method: e.Method, Content: func(w io.Writer) error {
				_, err := w.Write(e.Content)
				return err
			}})
		}
		_, err := WriteParts(w, parts)
		return err
	}}})
	if err != nil {
		t.Fatal(err)
	}
	if want := write(t, []Entry{{Name: "inner.zip", Content: inner, Method: Auto}}); !bytes.Equal(outer.Bytes(), want) || len(written) != 1 {
		t.Error("a zip written as a part is not the zip's bytes")
	}
}

// TestWritePartsRefusesChangedContent: content that writes other bytes
// the second time is refused, never written with a wrong header.
func TestWritePartsRefusesChangedContent(t *testing.T) {
	calls := 0
	_, err := WriteParts(io.Discard, []Part{{Name: "a", Method: Store, Content: func(w io.Writer) error {
		calls++
		_, err := w.Write([]byte(strings.Repeat("x", calls)))
		return err
	}}})
	if err == nil {
		t.Error("content that changed between its measure and its write was written")
	}
}

// largestWrite records the largest single write passed through it.
type largestWrite struct {
	w   io.Writer
	max int
}

func (l *largestWrite) Write(b []byte) (int, error) {
	l.max = max(l.max, len(b))
	return l.w.Write(b)
}

// TestWritePartsRanges: WriteParts reports where it wrote each entry's
// records, which a writer reads to name the entry that holds a byte of
// the zip. The local records tile the zip from its start, each beginning
// with a local header's signature and ending with the entry's stored
// bytes; the central headers tile the central directory, each beginning
// with a central header's signature, up to the end record.
func TestWritePartsRanges(t *testing.T) {
	entries := sample()
	parts := make([]Part, len(entries))
	for i, e := range entries {
		parts[i] = Part{Name: e.Name, Method: e.Method, Content: func(w io.Writer) error {
			_, err := w.Write(e.Content)
			return err
		}}
	}
	var buf bytes.Buffer
	written, err := WriteParts(&buf, parts)
	if err != nil {
		t.Fatal(err)
	}
	b := buf.Bytes()
	le := binary.LittleEndian
	var local, central int64 = 0, written[0].Central[0]
	for i, w := range written {
		h, data, err := encode(entries[i])
		if err != nil {
			t.Fatal(err)
		}
		switch {
		case w.Local[0] != local || le.Uint32(b[local:]) != sigLocal:
			t.Errorf("%s: local record at %d, want %d", entries[i].Name, w.Local[0], local)
		case w.Central[0] != central || le.Uint32(b[central:]) != sigCentral:
			t.Errorf("%s: central header at %d, want %d", entries[i].Name, w.Central[0], central)
		case !bytes.Equal(b[w.Local[1]-int64(h.csize):w.Local[1]], data):
			t.Errorf("%s: the local record does not end with the stored bytes", entries[i].Name)
		}
		local, central = w.Local[1], w.Central[1]
	}
	if local != written[0].Central[0] || central != int64(len(b))-eocdLen || le.Uint32(b[central:]) != sigEOCD {
		t.Errorf("the records end at %d and %d, the zip at %d", local, central, len(b))
	}
}
