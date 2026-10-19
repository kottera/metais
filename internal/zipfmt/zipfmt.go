// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

// Package zipfmt is the package container (ADR-001 D1 to D4): a strict
// reader of the one zip profile a package may use, which applies ADR-001
// D8's verification steps 1 to 5 before it returns anything, and the
// deterministic writer that produces that profile.
//
// The reader is METAIS's own. archive/zip is never used for reading: it
// checks little of the profile and its path check follows GODEBUG
// (ADR-001 Context, ADR-010 D11 item 1). It reads through an io.ReaderAt,
// every byte of the file once, with compress/flate over a byte-counting
// io.ByteReader, so the decoder never reads ahead and a stream that does
// not end exactly at its entry's compressed size is seen. Every bound is
// one of internal/limits', checked before the bytes it bounds are read.
//
// The package reaches no host file: the caller opens the file and hands
// over its reader and size.
package zipfmt

// The profile's constants (ADR-001 D1).
const (
	sigLocal    = 0x04034b50 // local file header signature
	sigCentral  = 0x02014b50 // central file header signature
	sigEOCD     = 0x06054b50 // end of central directory signature
	sigZip64Loc = 0x07064b50 // zip64 end of central directory locator signature

	localLen   = 30 // a local file header without its name and extra field
	centralLen = 46 // a central directory header without its name, extra field and comment
	eocdLen    = 22 // the end of central directory record, with no comment

	versionMadeBy  = 0x0314     // Unix, specification 2.0
	flagUTF8       = 0x0800     // bit 11, and every other bit clear
	dosTime        = 0x0000     // 00:00:00
	dosDate        = 0x0021     // 1980-01-01
	externalAttrs  = 0x81A40000 // a regular file, mode 0644
	methodStored   = 0
	methodDeflated = 8

	// maxEntries and maxSize are ADR-001 D3's ceilings, which no limit
	// can raise.
	maxEntries = 65_534
	maxSize    = 0xFFFFFFFE

	// endSearch is the largest span an end record and its comment can
	// occupy, in which no second end-record signature may lie (ADR-001 D8
	// step 2, ADR-009 D5 item 2).
	endSearch = 65_557

	// ratioFloor is the uncompressed size above which entry-ratio applies
	// (ADR-009 D3, RULINGS R2-147).
	ratioFloor = 1 << 20
)
