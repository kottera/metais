// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

// Package markdown is METAIS's Markdown (ADR-010 D9): the frontmatter split
// of ADR-003 D13 item 2, and the heading outline of ADR-008 D5's outline.
//
// SplitFrontmatter is METAIS code, and no frontmatter library is used: the
// rule is the specification's own, and the common libraries add TOML and
// JSON forms it does not have. It does not parse the frontmatter's YAML,
// which is internal/yamlx's, composed with this by the inputs layer.
//
// Outline reads CommonMark 0.31.2 with METAIS's own linear block and
// inline passes (ADR-010 D9, A2), after the frontmatter has been split off,
// and keeps only the headings: no other inline content is read, and no
// Markdown is rendered (ADR-010 D9 item 3).
package markdown

import "github.com/kottera/metais/internal/finding"

// Source is the file SplitFrontmatter reads, so that its findings can name
// it: the input it is part of, and where its text came from.
// SplitFrontmatter fills in each finding's line and column.
type Source struct {
	// Input is the subject every finding names: an input (ADR-006 D4).
	Input finding.Subject
	// At is the origin of the text, with its file. Its Line, Column and
	// Pointer are ignored.
	At finding.Location
}

// Heading is one heading of an outline (ADR-008 D5, outline item 2).
type Heading struct {
	// Level is 1 to 6.
	Level int
	// Text is the heading's inline content as plain text: its text and
	// code-span contents concatenated, markup removed, entities and
	// backslash escapes decoded, and each line break, as a multi-line
	// setext heading has, a single space.
	Text string
	// Line is the 1-based line of the file on which the heading begins,
	// counted by ADR-008 D5's rule: a line ends at an LF.
	Line int
}
