// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package zipfmt

import (
	"compress/flate"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"hash"
	"hash/crc32"
	"io"

	"github.com/kottera/metais/internal/finding"
	"github.com/kottera/metais/internal/limits"
	"github.com/kottera/metais/internal/unitext"
)

// Open reads the packed package r, of size bytes, and applies ADR-001 D8's
// steps 1 to 5 in order, each bounded by lim before the bytes it bounds are
// read:
//
//  1. the file's size is within package-size and D3's ceiling;
//  2. the end record is the file's last 22 bytes and satisfies D1 item 2,
//     its entry count is within entries, and no other end-record signature
//     lies in the file's final 65,557 bytes;
//  3. every central directory header satisfies D1 item 4 and its name
//     ADR-001 D2; the names are in order and pass the uniqueness rule; each
//     entry's declared sizes and ratio, and the sum of the declared sizes,
//     are within entry-size, entry-ratio and package-uncompressed;
//  4. walking from offset 0, every local header is where the previous
//     entry ended and equals its central directory header, and the central
//     directory begins where the last entry ends;
//  5. every entry's data is decoded once, stopped one byte past its
//     declared size, and its length, CRC-32 and SHA-256 established.
//
// A step that finds anything ends the reading there, with every finding
// of that step: later steps would only report its consequences. Open
// returns the Archive only when every step passed, and then no finding.
// It reads every byte of the file from r once and decodes every entry
// once.
//
// An error is a failure of r itself, or a file shorter than size; it is
// not a finding about the package, and the caller reports it.
//
// Steps 6 and 7, the digest file and the compiled files, are the loader's
// (ADR-001 D8), over the Archive's Entries.
func Open(r io.ReaderAt, size int64, lim limits.Set) (*Archive, []finding.Finding, error) {
	return OpenWith(r, size, lim, nil)
}

// OpenWith is Open, handing step 5's decoding of each entry to sink as it
// goes: sink is asked once per entry, in the file's order, for a writer
// that receives the entry's decoded bytes, and gives nil for an entry it
// does not want. So a reader that needs some entries' content — the
// loader's manifest, compiled files and schemas — has it without decoding
// any entry a second time (ADR-010 D15 item 2). What a writer receives is
// not yet verified: it is the package's content only when OpenWith
// returns the Archive, and its digest file has verified it (ADR-001 D8
// step 6).
func OpenWith(r io.ReaderAt, size int64, lim limits.Set, sink func(name string) io.Writer) (*Archive, []finding.Finding, error) {
	o := &opener{s: &source{r: r, size: size}, lim: lim, sink: sink}
	for _, step := range []func() error{o.size, o.end, o.directory, o.walk, o.decode} {
		if err := step(); err != nil {
			return nil, nil, err
		}
		if len(o.found) > 0 {
			return nil, o.found, nil
		}
	}
	a := &Archive{s: o.s, entries: o.entries, index: make(map[string]int, len(o.entries))}
	for i, e := range o.entries {
		a.index[e.name] = i
	}
	a.decodes = o.decodes
	return a, nil, nil
}

// entry is one entry as the central directory declares it and step 5
// establishes it.
type entry struct {
	name       string
	nameRead   bool  // the name was within path-length, and read
	header     int64 // offset of the central directory header
	versNeeded uint16
	flags      uint16
	method     uint16
	time, date uint16
	crc        uint32
	csize      uint32
	usize      uint32
	nameLen    uint16
	extraLen   uint16
	local      int64 // offset of the local header, as declared
	data       int64 // offset of the file data, from the walk
	sum        [sha256.Size]byte
}

type opener struct {
	s       *source
	lim     limits.Set
	found   []finding.Finding
	cdOff   int64
	cdSize  int64
	count   int
	entries []entry
	decodes int
	sink    func(name string) io.Writer
}

func (o *opener) add(f finding.Finding) { o.found = append(o.found, f) }

func (o *opener) breached(id limits.ID, name string, observed int64, has bool) {
	o.add(o.lim.Breached(id, *at(name), observed, has))
}

// size is step 1.
func (o *opener) size() error {
	bound := min(o.lim.Value(limits.PackageSize), maxSize)
	switch {
	case o.s.size > bound:
		o.breached(limits.PackageSize, "", o.s.size, true)
	case o.s.size < eocdLen:
		o.add(finding.New(CodeNotZip, pkg, at(""), nil))
	}
	return nil
}

// end is step 2. It reads the file's final 65,557 bytes, or all of a
// smaller file, and keeps them for the steps after it.
func (o *opener) end() error {
	s := o.s
	n := min(s.size, endSearch)
	tail := make([]byte, n)
	if err := s.readAt(tail, s.size-n); err != nil {
		return err
	}
	s.tail, s.tailStart = tail, s.size-n
	le := binary.LittleEndian
	recAt := int(n) - eocdLen
	rec := tail[recAt:]
	if le.Uint32(rec) != sigEOCD {
		o.misplacedEnd(tail)
		return nil
	}
	eocd := s.size - eocdLen
	for i := 0; i+4 <= len(tail); i++ {
		if i != recAt && le.Uint32(tail[i:]) == sigEOCD {
			o.add(container(CodeSecondEndRecord, "", s.tailStart+int64(i), nil))
			break
		}
	}
	disk, cdDisk := le.Uint16(rec[4:]), le.Uint16(rec[6:])
	onDisk, total := le.Uint16(rec[8:]), le.Uint16(rec[10:])
	cdSize, cdOff := le.Uint32(rec[12:]), le.Uint32(rec[16:])
	comment := le.Uint16(rec[20:])
	if comment != 0 {
		o.add(container(CodeArchiveComment, "", eocd, finding.Params{"length": num(int64(comment))}))
	}
	if disk != 0 {
		o.add(fieldValue("", recordEOCD, "number of this disk", eocd+4, int64(disk), 0))
	}
	if cdDisk != 0 {
		o.add(fieldValue("", recordEOCD, "number of the disk with the start of the central directory", eocd+6, int64(cdDisk), 0))
	}
	zip64 := onDisk == 0xFFFF || total == 0xFFFF || cdSize == 0xFFFFFFFF || cdOff == 0xFFFFFFFF
	if zip64 {
		o.add(container(CodeZip64, "", eocd, nil))
	} else if onDisk != total {
		o.add(fieldValue("", recordEOCD, "total number of entries in the central directory on this disk", eocd+8, int64(onDisk), int64(total)))
	}
	if !zip64 && int64(total) > min(o.lim.Value(limits.Entries), maxEntries) {
		o.breached(limits.Entries, "", int64(total), true)
	}
	if !zip64 && int64(cdOff)+int64(cdSize) != eocd {
		if eocd >= 20 && eocd-20 >= s.tailStart && le.Uint32(tail[recAt-20:]) == sigZip64Loc {
			o.add(container(CodeZip64, "", eocd-20, nil))
		} else {
			o.add(container(CodeOverlapOrGap, "", eocd, finding.Params{"declared": num(int64(cdOff))}))
		}
	}
	o.cdOff, o.cdSize, o.count = int64(cdOff), int64(cdSize), int(total)
	return nil
}

// misplacedEnd reports a file whose last 22 bytes are not an end record:
// one with an archive comment or trailing bytes, when an end-record
// signature lies earlier in tail, and otherwise one that is not a zip.
func (o *opener) misplacedEnd(tail []byte) {
	le := binary.LittleEndian
	for i := len(tail) - eocdLen - 1; i >= 0; i-- {
		if le.Uint32(tail[i:]) != sigEOCD {
			continue
		}
		at := o.s.tailStart + int64(i)
		end := at + eocdLen + int64(le.Uint16(tail[i+20:]))
		switch {
		case end == o.s.size:
			o.add(container(CodeArchiveComment, "", at, finding.Params{"length": num(end - at - eocdLen)}))
		case end < o.s.size:
			o.add(container(CodeTrailingData, "", end, nil))
		default:
			o.add(container(CodeTrailingData, "", at+eocdLen, nil))
		}
		return
	}
	o.add(finding.New(CodeNotZip, pkg, at(""), nil))
}

// directory is step 3: every central directory header, its name, the
// order and uniqueness of the names, and the declared sizes.
func (o *opener) directory() error {
	le := binary.LittleEndian
	r := newRegion(o.s, o.cdOff, o.cdOff+o.cdSize)
	pathLength := o.lim.Value(limits.PathLength)
	var (
		names []string
		total int64
		fixed [centralLen]byte
	)
	for range o.count {
		off := o.cdOff + r.consumed
		if r.remaining() < centralLen {
			o.add(container(CodeOverlapOrGap, "", off, nil))
			return nil
		}
		if err := r.readFull(fixed[:]); err != nil {
			return err
		}
		h := fixed[:]
		if le.Uint32(h) != sigCentral {
			o.add(fieldValue("", recordCentral, "central file header signature", off, int64(le.Uint32(h)), sigCentral))
			return nil
		}
		e := entry{
			header:     off,
			versNeeded: le.Uint16(h[6:]),
			flags:      le.Uint16(h[8:]),
			method:     le.Uint16(h[10:]),
			time:       le.Uint16(h[12:]),
			date:       le.Uint16(h[14:]),
			crc:        le.Uint32(h[16:]),
			csize:      le.Uint32(h[20:]),
			usize:      le.Uint32(h[24:]),
			nameLen:    le.Uint16(h[28:]),
			extraLen:   le.Uint16(h[30:]),
			local:      int64(le.Uint32(h[42:])),
		}
		commentLen := le.Uint16(h[32:])
		if r.remaining() < int64(e.nameLen)+int64(e.extraLen)+int64(commentLen) {
			o.add(container(CodeOverlapOrGap, "", off, nil))
			return nil
		}
		if int64(e.nameLen) <= pathLength {
			name := make([]byte, e.nameLen)
			if err := r.readFull(name); err != nil {
				return err
			}
			e.name, e.nameRead = string(name), true
		} else if err := r.skip(int64(e.nameLen)); err != nil {
			return err
		}
		if err := r.skip(int64(e.extraLen) + int64(commentLen)); err != nil {
			return err
		}
		o.header(&e, h, commentLen)
		if e.nameRead {
			o.found = append(o.found, nameFindings(e.name, off, unitext.CheckPath(e.name, o.lim), o.lim)...)
			if len(names) > 0 && e.name < names[len(names)-1] {
				o.add(container(CodeEntryOrder, e.name, off, nil))
			}
			names = append(names, e.name)
		} else {
			o.breached(limits.PathLength, "", int64(e.nameLen), true)
		}
		if e.usize != 0xFFFFFFFF {
			total += int64(e.usize)
		}
		o.entries = append(o.entries, e)
	}
	if r.remaining() > 0 {
		o.add(container(CodeOverlapOrGap, "", o.cdOff+r.consumed, nil))
	}
	o.collisions()
	if total > o.lim.Value(limits.PackageUncompressed) {
		o.breached(limits.PackageUncompressed, "", total, true)
	}
	return nil
}

// header checks the fixed fields of one central directory header, h, and
// its declared sizes (ADR-001 D1 items 3 to 6, D3; ADR-009 D5 item 4).
func (o *opener) header(e *entry, h []byte, commentLen uint16) {
	le := binary.LittleEndian
	name, off := e.name, e.header
	if v := le.Uint16(h[4:]); v != versionMadeBy {
		o.add(fieldValue(name, recordCentral, "version made by", off+4, int64(v), versionMadeBy))
	}
	o.fields(e, recordCentral, off, h[6:])
	if e.extraLen != 0 {
		o.add(container(CodeExtraField, name, off, finding.Params{"length": num(int64(e.extraLen))}))
	}
	if commentLen != 0 {
		o.add(fieldValue(name, recordCentral, "file comment length", off+32, int64(commentLen), 0))
	}
	if v := le.Uint16(h[34:]); v != 0 {
		o.add(fieldValue(name, recordCentral, "disk number start", off+34, int64(v), 0))
	}
	if v := le.Uint16(h[36:]); v != 0 {
		o.add(fieldValue(name, recordCentral, "internal file attributes", off+36, int64(v), 0))
	}
	if v := le.Uint32(h[38:]); v != externalAttrs {
		o.add(fieldValue(name, recordCentral, "external file attributes", off+38, int64(v), externalAttrs))
	}
	if e.local == 0xFFFFFFFF {
		o.add(container(CodeZip64, name, off+42, nil))
	}
	if e.csize == 0xFFFFFFFF || e.usize == 0xFFFFFFFF {
		o.add(container(CodeZip64, name, off+20, nil))
		return
	}
	if e.method == methodStored && e.csize != e.usize {
		o.add(container(CodeSizeMismatch, name, off+20, finding.Params{"declared": num(int64(e.usize))}))
	}
	if int64(e.usize) > o.lim.Value(limits.EntrySize) {
		o.breached(limits.EntrySize, name, int64(e.usize), true)
	}
	if e.usize > ratioFloor {
		ratio := o.lim.Value(limits.EntryRatio)
		switch {
		case e.csize == 0:
			o.breached(limits.EntryRatio, name, 0, false)
		case int64(e.usize) > ratio*int64(e.csize):
			o.breached(limits.EntryRatio, name, int64(e.usize)/int64(e.csize), true)
		}
	}
}

// fields checks the fields a local header and a central directory header
// share that are constants of the profile or follow from the method:
// version needed to extract, the flags, the method and the time and date,
// which h holds from its first byte, at offset off+6 of the record.
func (o *opener) fields(e *entry, record string, off int64, h []byte) {
	le := binary.LittleEndian
	name := e.name
	vers, flags, method := le.Uint16(h[0:]), le.Uint16(h[2:]), le.Uint16(h[4:])
	encrypted := flags&(1<<0|1<<6|1<<13) != 0 || method == 99
	switch {
	case encrypted:
		o.add(container(CodeEncrypted, name, off+8, nil))
	case method != methodStored && method != methodDeflated:
		o.add(container(CodeMethod, name, off+10, finding.Params{"method": num(int64(method))}))
	}
	if flags&(1<<3) != 0 {
		o.add(container(CodeDataDescriptor, name, off+8, nil))
	}
	if rest := flags &^ (1<<0 | 1<<3 | 1<<6 | 1<<13); rest != flagUTF8 {
		o.add(container(CodeFlags, name, off+8, finding.Params{"value": num(int64(flags))}))
	}
	if method == methodStored || method == methodDeflated {
		if want := versionNeeded(method); vers != want {
			o.add(fieldValue(name, record, "version needed to extract", off+6, int64(vers), int64(want)))
		}
		if method == methodDeflated && e.usize == 0 {
			o.add(fieldValue(name, record, "compression method", off+10, methodDeflated, methodStored))
		}
	}
	if v := le.Uint16(h[6:]); v != dosTime {
		o.add(fieldValue(name, record, "last mod file time", off+12, int64(v), dosTime))
	}
	if v := le.Uint16(h[8:]); v != dosDate {
		o.add(fieldValue(name, record, "last mod file date", off+14, int64(v), dosDate))
	}
}

// collisions applies ADR-001 D2 item 4 to the names step 3 read.
func (o *opener) collisions() {
	var names []string
	header := map[string]int64{}
	for _, e := range o.entries {
		if e.nameRead {
			if _, ok := header[e.name]; !ok {
				header[e.name] = e.header
			}
			names = append(names, e.name)
		}
	}
	// The header of each occurrence, in order, for duplicates.
	seen := map[string]int{}
	offsets := map[string][]int64{}
	for _, e := range o.entries {
		if e.nameRead {
			offsets[e.name] = append(offsets[e.name], e.header)
		}
	}
	for _, c := range unitext.Collisions(names) {
		off := header[c.Path]
		if c.Kind == unitext.Duplicate {
			seen[c.Path]++
			off = offsets[c.Path][seen[c.Path]]
		}
		o.add(collisionFinding(c, off))
	}
}

// walk is step 4: from offset 0, each local header where the previous
// entry ended, equal to its central directory header, and the central
// directory where the last entry ends.
func (o *opener) walk() error {
	le := binary.LittleEndian
	var next int64
	var fixed [localLen]byte
	walkedAstray := false
	for i := range o.entries {
		e := &o.entries[i]
		misplaced := e.local != next
		walkedAstray = walkedAstray || misplaced
		if misplaced {
			if i == 0 {
				o.add(container(CodePrependedData, e.name, 0, finding.Params{"declared": num(e.local)}))
				return nil
			}
			// The walk goes on from where the header is declared, so each
			// misplaced entry is reported once, and not the entries after it.
			o.add(container(CodeOverlapOrGap, e.name, next, finding.Params{"declared": num(e.local)}))
		}
		pos := e.local
		dataEnd := pos + localLen + int64(e.nameLen) + int64(e.csize)
		if dataEnd > o.cdOff {
			o.add(container(CodeOverlapOrGap, e.name, min(next, o.cdOff), nil))
			return nil
		}
		e.data = pos + localLen + int64(e.nameLen)
		next = dataEnd
		if misplaced {
			continue
		}
		if err := o.s.readAt(fixed[:], pos); err != nil {
			return err
		}
		h := fixed[:]
		if le.Uint32(h) != sigLocal {
			o.add(container(CodeOverlapOrGap, e.name, pos, nil))
			continue
		}
		o.compareLocal(e, pos, h)
		if le.Uint16(h[26:]) == e.nameLen && e.nameRead {
			name := make([]byte, e.nameLen)
			if err := o.s.readAt(name, pos+localLen); err != nil {
				return err
			}
			if string(name) != e.name {
				o.add(container(CodeHeaderMismatch, e.name, pos+localLen, finding.Params{"header_field": finding.String("file name")}))
			}
		}
	}
	// After a misplaced entry the end of the walk is its consequence.
	if next != o.cdOff && !walkedAstray {
		if len(o.entries) == 0 {
			o.add(container(CodePrependedData, "", 0, finding.Params{"declared": num(o.cdOff)}))
		} else {
			o.add(container(CodeOverlapOrGap, "", next, finding.Params{"declared": num(o.cdOff)}))
		}
	}
	return nil
}

// compareLocal compares the local header h at off with e, its central
// directory header, field for field (ADR-001 D1 item 4, ADR-009 D5 item 5).
func (o *opener) compareLocal(e *entry, off int64, h []byte) {
	le := binary.LittleEndian
	for _, f := range []struct {
		field     string
		at        int64
		got, want uint32
	}{
		{"version needed to extract", 4, uint32(le.Uint16(h[4:])), uint32(e.versNeeded)},
		{"general purpose bit flag", 6, uint32(le.Uint16(h[6:])), uint32(e.flags)},
		{"compression method", 8, uint32(le.Uint16(h[8:])), uint32(e.method)},
		{"last mod file time", 10, uint32(le.Uint16(h[10:])), uint32(e.time)},
		{"last mod file date", 12, uint32(le.Uint16(h[12:])), uint32(e.date)},
		{"crc-32", 14, le.Uint32(h[14:]), e.crc},
		{"compressed size", 18, le.Uint32(h[18:]), e.csize},
		{"uncompressed size", 22, le.Uint32(h[22:]), e.usize},
		{"file name length", 26, uint32(le.Uint16(h[26:])), uint32(e.nameLen)},
		{"extra field length", 28, uint32(le.Uint16(h[28:])), uint32(e.extraLen)},
	} {
		if f.got != f.want {
			o.add(container(CodeHeaderMismatch, e.name, off+f.at, finding.Params{"header_field": finding.String(f.field)}))
		}
	}
}

// decode is step 5: every entry's data decoded once, counted against its
// declared size and stopped one byte past it, its CRC-32 checked and its
// SHA-256 computed (ADR-001 D1 item 5, ADR-009 D5 item 7).
func (o *opener) decode() error {
	var total int64
	for i := range o.entries {
		e := &o.entries[i]
		var h hash.Hash = sha256.New()
		if o.sink != nil {
			if w := o.sink(e.name); w != nil {
				h = teeHash{h, w}
			}
		}
		n, err := o.decodeEntry(e, h, func(name string, off int64, code finding.Code, p finding.Params) {
			o.add(container(code, name, off, p))
		})
		if err != nil {
			return err
		}
		total += n
	}
	if total > o.lim.Value(limits.PackageUncompressed) {
		o.breached(limits.PackageUncompressed, "", total, true)
	}
	return nil
}

// teeHash is a hash that also writes what it hashes to w.
type teeHash struct {
	hash.Hash
	w io.Writer
}

func (t teeHash) Write(p []byte) (int, error) {
	t.w.Write(p)
	return t.Hash.Write(p)
}

// decodeEntry decodes e's data into h, reporting through report, and
// returns how many bytes it decoded.
func (o *opener) decodeEntry(e *entry, h hash.Hash, report func(string, int64, finding.Code, finding.Params)) (int64, error) {
	o.decodes++
	n, problem, err := decodeData(o.s, e, h)
	if err != nil {
		return n, err
	}
	switch problem {
	case "":
		copy(e.sum[:], h.Sum(nil))
	case "size":
		report(e.name, e.data, CodeSizeMismatch, finding.Params{"declared": num(int64(e.usize))})
	case "crc":
		report(e.name, e.data, CodeCRCMismatch, nil)
	default:
		report(e.name, e.data, CodeDeflateStream, finding.Params{"reason": finding.String(problem)})
	}
	return n, nil
}

// decodeData decodes e's data from s into w, and returns how many bytes it
// decoded and what is wrong with them: "" when nothing is, "size" or
// "crc", or a deflate_stream reason. It reads no byte outside e's data and
// decodes at most one byte past its declared size.
func decodeData(s *source, e *entry, w io.Writer) (int64, string, error) {
	r := newRegion(s, e.data, e.data+int64(e.csize))
	var src io.Reader = r
	var fr io.ReadCloser
	if e.method == methodDeflated {
		fr = flate.NewReader(r)
		src = fr
	}
	crc := crc32.NewIEEE()
	n, err := io.Copy(io.MultiWriter(w, crc), io.LimitReader(src, int64(e.usize)+1))
	if err != nil {
		if r.err != nil && r.err != io.EOF {
			return n, "", r.err
		}
		var corrupt flate.CorruptInputError
		switch {
		case errors.Is(err, io.ErrUnexpectedEOF):
			return n, streamTruncated, nil
		case errors.As(err, &corrupt):
			return n, streamInvalid, nil
		}
		return n, streamInvalid, nil
	}
	if n != int64(e.usize) {
		return n, "size", nil
	}
	if fr != nil {
		// The stream must end here, exactly at the end of the data.
		var one [1]byte
		switch m, err := fr.Read(one[:]); {
		case m > 0:
			return n, "size", nil
		case err == io.ErrUnexpectedEOF:
			return n, streamTruncated, nil
		case err != io.EOF:
			if r.err != nil && r.err != io.EOF {
				return n, "", r.err
			}
			return n, streamInvalid, nil
		}
		if r.consumed != int64(e.csize) {
			return n, streamTrailing, nil
		}
	}
	if crc.Sum32() != e.crc {
		return n, "crc", nil
	}
	return n, "", nil
}
