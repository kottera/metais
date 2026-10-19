// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package manifest

import "strings"

// The container profiles (ADR-005 D9).
const (
	ProfileOOXML = "ooxml"
	ProfileODF   = "odf"
	ProfileEPUB  = "epub"
)

// MimetypeEntry is the entry the odf and epub profiles write themselves
// (ADR-005 D9 items 2 to 4).
const MimetypeEntry = "mimetype"

// epubMimetype is the epub profile's media type (ADR-005 D9 item 3).
const epubMimetype = "application/epub+zip"

// RequiredEntry is the entry path c's profile requires, or "" when c has
// no profile (ADR-005 D9 items 1 to 3).
func (c *Container) RequiredEntry() string {
	switch c.Profile {
	case ProfileOOXML:
		return "[Content_Types].xml"
	case ProfileODF:
		return "META-INF/manifest.xml"
	case ProfileEPUB:
		return "META-INF/container.xml"
	}
	return ""
}

// ToolMimetype is the content of the mimetype entry c's profile writes
// itself, and false when its profile writes none (ADR-005 D9 items 2 and
// 3).
func (c *Container) ToolMimetype() (string, bool) {
	switch c.Profile {
	case ProfileODF:
		return c.Mimetype, true
	case ProfileEPUB:
		return epubMimetype, true
	}
	return "", false
}

// FirstEntry is the entry of the author's that c's profile places first,
// or "" (ADR-005 D9 item 1). A profile's own mimetype entry precedes
// every other.
func (c *Container) FirstEntry() string {
	if c.Profile == ProfileOOXML {
		return "[Content_Types].xml"
	}
	return ""
}

// EntryProblem is a rule of c's profile an entry path breaks.
type EntryProblem int

const (
	// EntryFine is a path the profile accepts.
	EntryFine EntryProblem = iota
	// EntryReserved is the author's own mimetype entry, which the odf and
	// epub profiles write themselves.
	EntryReserved
	// EntryNonASCII is a path holding a character outside ASCII, which
	// ooxml refuses (ECMA-376 Part 2 §7.3.3).
	EntryNonASCII
	// EntrySpace is a path holding U+0020, which EPUB says it should not:
	// a warning, never a refusal.
	EntrySpace
)

// CheckEntry returns the rule of c's profile the entry path p breaks
// (ADR-005 D9). D12's rules are not the profile's, and are not judged
// here.
func (c *Container) CheckEntry(p string) EntryProblem {
	switch c.Profile {
	case ProfileOOXML:
		for i := 0; i < len(p); i++ {
			if p[i] >= 0x80 {
				return EntryNonASCII
			}
		}
	case ProfileODF:
		if p == MimetypeEntry {
			return EntryReserved
		}
	case ProfileEPUB:
		if p == MimetypeEntry {
			return EntryReserved
		}
		if strings.Contains(p, " ") {
			return EntrySpace
		}
	}
	return EntryFine
}
