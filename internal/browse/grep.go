// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package browse

import (
	"errors"
	"unicode/utf8"

	"github.com/kottera/metais/internal/pkgtree"
	"github.com/kottera/metais/internal/regexprofile"
)

// MaxDisplayed is the most bytes of a line grep displays (ADR-008 D5, grep
// item 6): a longer line is cut at the last UTF-8 boundary within it.
const MaxDisplayed = 1024

// GrepOptions is what Grep takes beside the tree, the entries and the
// pattern.
type GrepOptions struct {
	Options
	// Context is the number of lines shown before and after each matching
	// line reported.
	Context int
	// MaxResults is the number of matching lines reported; the search
	// still runs to the end and counts every one.
	MaxResults int
}

// Shown is a line as grep displays it: its number, and its text without
// its line ending, cut to MaxDisplayed bytes with Omitted the bytes cut.
type Shown struct {
	Line    int
	Text    string
	Omitted int
}

// Match is one matching line reported, with its context.
type Match struct {
	Path string
	Shown
	Before, After []Shown
}

// GrepResult is a search's result (ADR-008 D5, grep item 7).
type GrepResult struct {
	// Matches are the matching lines reported, at most MaxResults, in
	// entry path order and line order.
	Matches []Match
	// MatchingLines is every matching line found, reported or not; a line
	// with several matches counts once.
	MatchingLines int
	// Truncated is whether matching lines were found beyond Matches.
	Truncated bool
	// Skipped are the binary entries among those searched, in path order.
	Skipped []string
}

// ErrRegexWork is a search that the regex-work meter refused before it
// finished (ADR-009 D10 item 7): no result is given for it.
var ErrRegexWork = errors.New("browse: the search exceeds regex-work")

// Grep searches the entries of t named by entries, in their order, which
// is path order, matching re against each line of each text entry
// separately, without its line ending (ADR-008 D5, grep). Binary entries
// are skipped and listed. re charges m for every line it is matched
// against, and an exhausted m ends the search with ErrRegexWork.
func Grep(t pkgtree.Tree, entries []string, re *regexprofile.Regexp, m *regexprofile.Meter, opt GrepOptions) (GrepResult, error) {
	res := GrepResult{Matches: []Match{}, Skipped: []string{}}
	for _, e := range entries {
		b, err := Content(t, e, opt.Options)
		if err != nil {
			return GrepResult{}, err
		}
		if !IsText(b) {
			res.Skipped = append(res.Skipped, e)
			continue
		}
		lines := Lines(b)
		for i, l := range lines {
			matched := re.MatchString(string(l.Text))
			if m != nil && m.Exhausted() {
				return GrepResult{}, ErrRegexWork
			}
			if !matched {
				continue
			}
			res.MatchingLines++
			if len(res.Matches) == opt.MaxResults {
				res.Truncated = true
				continue
			}
			mt := Match{Path: e, Shown: show(l), Before: []Shown{}, After: []Shown{}}
			for _, c := range lines[max(0, i-opt.Context):i] {
				mt.Before = append(mt.Before, show(c))
			}
			for _, c := range lines[i+1 : min(len(lines), i+1+opt.Context)] {
				mt.After = append(mt.After, show(c))
			}
			res.Matches = append(res.Matches, mt)
		}
	}
	return res, nil
}

// show is l as grep displays it.
func show(l Line) Shown {
	text := l.Text
	omitted := 0
	if len(text) > MaxDisplayed {
		cut := MaxDisplayed
		for cut > 0 && !utf8.RuneStart(text[cut]) {
			cut--
		}
		omitted = len(text) - cut
		text = text[:cut]
	}
	return Shown{Line: l.Number, Text: string(text), Omitted: omitted}
}
