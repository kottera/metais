// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package vectorgen

// layout is where archive.write placed each record of a zip: what a
// case's expected values are derived from (values_test.go), never from a
// reading of the bytes.
type layout struct {
	entries []placed
	cd      int  // the first central directory header
	cdTail  int  // the bytes after the last central directory header
	end     int  // the end record
	eocd    eocd // the end record's fields, as written
	suffix  int  // the bytes after the end record's comment
	size    int
}

// placed is one entry's records: the offsets of its local header, its
// file data and its central directory header, and both headers' fields
// as written.
type placed struct {
	name                 string
	local, data, central int
	lh, ch               header
}

// The offsets of the fields of a local header, a central directory header
// and the end record that a finding names (APPNOTE 4.3.7, 4.3.12, 4.3.16).
const (
	lfNeeded = 4
	lfFlags  = 6
	lfMethod = 8
	lfCRC    = 14
	lfCSize  = 18
	lfUSize  = 22
	lfName   = 30

	cfMadeBy     = 4
	cfNeeded     = 6
	cfFlags      = 8
	cfMethod     = 10
	cfTime       = 12
	cfDate       = 14
	cfCSize      = 20
	cfUSize      = 24
	cfCommentLen = 32
	cfDisk       = 34
	cfInternal   = 36
	cfExternal   = 38
	cfOffset     = 42

	efDisk   = 4
	efCDDisk = 6
	efOnDisk = 8
	efCDOff  = 16
)

// entry is the first entry of l named name.
func (l layout) entry(name string) placed { return l.nth(name, 0) }

// nth is the entry of l named name that comes after i others of that name.
func (l layout) nth(name string, i int) placed {
	for _, e := range l.entries {
		if e.name == name {
			if i == 0 {
				return e
			}
			i--
		}
	}
	panic("vectorgen: the layout places no entry " + name)
}
