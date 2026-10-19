// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

// Package regexprofile is METAIS's pattern profile (ADR-009 D10 item 6,
// ADR-010 D10 item 2): a schema's pattern and patternProperties are
// ECMA-262 patterns in u mode, read here by METAIS's own tokenizer,
// translated to RE2 and matched by Go's regexp, whose matching is linear in
// the subject.
//
// A construct the profile refuses is never approximated: lookahead,
// lookbehind, backreferences, repeat bounds above 1000, lone surrogates,
// \p{…} forms RE2 cannot express, and RE2-only syntax ECMA-262 does not
// define are each refused with an *UnsupportedError naming the construct,
// and so is a pattern that is not ECMA-262 u-mode syntax at all. The rest
// is translated by ADR-009's table, so that every class means what
// ECMA-262 says it means.
package regexprofile

import (
	"errors"
	"fmt"
	"regexp"
	"regexp/syntax"

	"github.com/kottera/metais/internal/limits"
)

// Construct names what a refused pattern holds. Its values are fixed
// words, so a refusal names the construct and never repeats the pattern.
type Construct string

// The constructs the profile refuses (ADR-009 D10 item 6).
const (
	// Lookahead is (?=…) or (?!…).
	Lookahead Construct = "lookahead"
	// Lookbehind is (?<=…) or (?<!…).
	Lookbehind Construct = "lookbehind"
	// Backreference is a numbered (\1) or named (\k<name>) backreference.
	Backreference Construct = "backreference"
	// RepeatBound is a {n}, {n,} or {n,m} bound above 1000, or nested
	// bounds whose product is: RE2 refuses both.
	RepeatBound Construct = "repeat_bound"
	// LoneSurrogate is a \u escape of a surrogate code point that is not
	// one half of a pair.
	LoneSurrogate Construct = "lone_surrogate"
	// UnicodeProperty is a \p{…} or \P{…} that names no general category
	// or script in a form RE2 accepts.
	UnicodeProperty Construct = "unicode_property"
	// RE2Syntax is syntax RE2 defines and ECMA-262 does not: inline flags
	// such as (?i), \A, \z, \Q…\E, (?P<…>) and the like.
	RE2Syntax Construct = "re2_syntax"
	// Syntax is a pattern that is not ECMA-262 u-mode syntax.
	Syntax Construct = "syntax"
)

// MaxRepeat is the largest repeat bound the profile accepts.
const MaxRepeat = 1000

// UnsupportedError is a pattern the profile refuses.
type UnsupportedError struct {
	// Construct is what the pattern holds.
	Construct Construct
	// Offset is the byte offset in the pattern where the construct
	// begins.
	Offset int
}

func (e *UnsupportedError) Error() string {
	return fmt.Sprintf("regexprofile: %s at offset %d", e.Construct, e.Offset)
}

// LimitError is a pattern that breaches a limit: pattern-length, measured
// before it is read, or regex-program, measured on what it compiles to.
type LimitError struct {
	// Limit is the limit breached.
	Limit limits.ID
	// Observed is the value found, when HasObserved is set; a program RE2
	// refused as too large or too deeply nested has none.
	Observed    int64
	HasObserved bool
}

func (e *LimitError) Error() string {
	return fmt.Sprintf("regexprofile: breaches %s", limits.Get(e.Limit).Name)
}

// Regexp is a compiled pattern. It is what the JSON Schema validator
// matches with, and its translated tree is what the sample generator walks
// (ADR-007 D9 item 18).
type Regexp struct {
	pattern    string
	translated string
	tree       *syntax.Regexp
	insts      int
	re         *regexp.Regexp
	meter      *Meter
}

// Compile reads pattern by the profile and compiles it under lim's
// pattern-length and regex-program. Each match charges m (ADR-009 D10
// item 7); a nil m charges nothing.
//
// The error is an *UnsupportedError or a *LimitError; any other error is a
// defect in the translation, never in the pattern.
func Compile(pattern string, lim limits.Set, m *Meter) (*Regexp, error) {
	if n := int64(len(pattern)); n > lim.Value(limits.PatternLength) {
		return nil, &LimitError{Limit: limits.PatternLength, Observed: n, HasObserved: true}
	}
	translated, err := translate(pattern)
	if err != nil {
		return nil, err
	}
	tree, err := syntax.Parse(translated, syntax.Perl)
	if err != nil {
		var se *syntax.Error
		if errors.As(err, &se) {
			switch se.Code {
			case syntax.ErrInvalidRepeatSize:
				// Nested bounds, each within 1000, whose product RE2
				// refuses: a repeat bound above 1000 all the same.
				return nil, &UnsupportedError{Construct: RepeatBound}
			case syntax.ErrLarge, syntax.ErrNestingDepth:
				// No pattern within pattern-length's default reaches
				// either; a program RE2 refuses as too large or too deep
				// is a regex-program breach whose size is unmeasured.
				return nil, &LimitError{Limit: limits.RegexProgram}
			}
		}
		return nil, fmt.Errorf("regexprofile: the translation of a pattern does not parse: %w", err)
	}
	prog, err := syntax.Compile(tree.Simplify())
	if err != nil {
		return nil, fmt.Errorf("regexprofile: the translation of a pattern does not compile: %w", err)
	}
	if n := int64(len(prog.Inst)); n > lim.Value(limits.RegexProgram) {
		return nil, &LimitError{Limit: limits.RegexProgram, Observed: n, HasObserved: true}
	}
	re, err := regexp.Compile(translated)
	if err != nil {
		return nil, fmt.Errorf("regexprofile: the translation of a pattern does not compile: %w", err)
	}
	return &Regexp{pattern: pattern, translated: translated, tree: tree, insts: len(prog.Inst), re: re, meter: m}, nil
}

// String returns the pattern as it was written.
func (r *Regexp) String() string { return r.pattern }

// Translated returns the RE2 pattern the profile translated it to.
func (r *Regexp) Translated() string { return r.translated }

// Syntax returns the parsed tree of the translation. The tree is shared:
// the caller does not change it.
func (r *Regexp) Syntax() *syntax.Regexp { return r.tree }

// Instructions returns the size of the compiled program, which
// regex-program bounds.
func (r *Regexp) Instructions() int { return r.insts }

// MatchString reports whether s holds a match anywhere: schema patterns
// are unanchored. It first charges the meter the program's size times s's
// length; when the charge would exceed what remains, it refuses the match,
// marks the meter exhausted and returns false.
func (r *Regexp) MatchString(s string) bool {
	if r.meter != nil && !r.meter.charge(int64(r.insts), int64(len(s))) {
		return false
	}
	return r.re.MatchString(s)
}

// Meter is regex-work: the matching budget of one command (ADR-009 D10
// item 7). It is not safe for concurrent use; nothing that matches starts
// a goroutine.
type Meter struct {
	remaining int64
	spent     int64
	exhausted bool
}

// NewMeter returns a meter holding budget units.
func NewMeter(budget int64) *Meter { return &Meter{remaining: budget} }

// Exhausted reports whether a match was refused because its charge would
// have exceeded what remained.
func (m *Meter) Exhausted() bool { return m.exhausted }

// Spent returns the units charged so far.
func (m *Meter) Spent() int64 { return m.spent }

func (m *Meter) charge(insts, length int64) bool {
	if m.exhausted {
		return false
	}
	if length > 0 && insts > m.remaining/length {
		m.exhausted = true
		return false
	}
	m.remaining -= insts * length
	m.spent += insts * length
	return true
}
