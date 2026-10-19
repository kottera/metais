// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package browse

import (
	"strings"
	"unicode/utf8"
)

// GlobError is a glob that does not satisfy the METAIS glob grammar
// (ADR-008 D5). Reason is a fixed word, so a refusal names what is wrong
// and never repeats the glob.
type GlobError struct {
	Reason string
}

func (e *GlobError) Error() string { return "browse: the glob is refused (" + e.Reason + ")" }

// The reasons a glob is refused.
const (
	// GlobEmpty is an empty glob, or one with an empty segment: a leading
	// or doubled /.
	GlobEmpty = "empty_segment"
	// GlobDoubleStar is ** anywhere but standing alone as a segment.
	GlobDoubleStar = "double_star"
	// GlobClass is a character class with no closing ], or with a range
	// whose ends are reversed.
	GlobClass = "class"
	// GlobEscape is a \ with nothing after it.
	GlobEscape = "escape"
	// GlobEncoding is a glob that is not valid UTF-8.
	GlobEncoding = "encoding"
)

// Glob is a compiled METAIS glob (ADR-008 D5): matched against the whole
// entry path, segment by segment, with / as the separator.
//
//   - * matches any run of characters within one segment;
//   - ? matches one character other than /;
//   - [...] and [^...] are character classes with lo-hi ranges;
//   - \ makes the next character literal;
//   - ** standing alone as a segment matches zero or more whole segments.
//
// There are no braces, and matching is case-sensitive. *, ?, [...] and **
// never match a segment that begins with "."; a pattern segment that itself
// begins with a literal "." matches one. A glob with no wildcard that names
// a directory selects every entry beneath it.
type Glob struct {
	segs    []segment
	literal string // the path a glob with no wildcard names; "" otherwise
	wild    bool
}

// segment is one segment of a glob: ** alone, or a run of atoms.
type segment struct {
	any   bool // ** alone
	atoms []atom
	dot   bool // begins with a literal "."
}

type atomKind uint8

const (
	atomLit atomKind = iota
	atomStar
	atomOne
	atomClass
)

// atom is one character of a segment's pattern.
type atom struct {
	kind   atomKind
	r      rune // atomLit's character
	negate bool
	ranges [][2]rune // atomClass's ranges, inclusive
}

// CompileGlob reads s by the METAIS glob grammar. A trailing / is allowed
// and ignored, since a directory may be written with or without one
// (ADR-008 D2 item 5); "." names the root.
func CompileGlob(s string) (*Glob, error) {
	if !utf8.ValidString(s) {
		return nil, &GlobError{GlobEncoding}
	}
	if s == "." || s == "./" {
		return &Glob{literal: "."}, nil
	}
	s = strings.TrimSuffix(s, "/")
	if s == "" {
		return nil, &GlobError{GlobEmpty}
	}
	g := &Glob{}
	var lit []string
	for _, part := range splitGlob(s) {
		if part == "" {
			return nil, &GlobError{GlobEmpty}
		}
		if part == "**" {
			g.segs = append(g.segs, segment{any: true})
			g.wild = true
			continue
		}
		seg, text, wild, err := compileSegment(part)
		if err != nil {
			return nil, err
		}
		g.wild = g.wild || wild
		g.segs = append(g.segs, seg)
		lit = append(lit, text)
	}
	if !g.wild {
		g.literal = strings.Join(lit, "/")
	}
	return g, nil
}

// splitGlob splits s at each / that is not escaped and not inside a
// character class. A class never matches /, so a / inside one is a
// separator, which then leaves the class unclosed.
func splitGlob(s string) []string {
	var out []string
	start := 0
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '\\':
			i++
		case '/':
			out = append(out, s[start:i])
			start = i + 1
		}
	}
	return append(out, s[start:])
}

// compileSegment reads one segment other than a lone **. It returns the
// segment, its text with escapes removed, and whether it holds a wildcard.
func compileSegment(p string) (segment, string, bool, error) {
	var seg segment
	var text strings.Builder
	wild := false
	for i := 0; i < len(p); {
		r, n := utf8.DecodeRuneInString(p[i:])
		switch r {
		case '\\':
			if i+n >= len(p) {
				return seg, "", false, &GlobError{GlobEscape}
			}
			lr, ln := utf8.DecodeRuneInString(p[i+n:])
			seg.atoms = append(seg.atoms, atom{kind: atomLit, r: lr})
			text.WriteRune(lr)
			i += n + ln
		case '*':
			if i+1 < len(p) && p[i+1] == '*' {
				return seg, "", false, &GlobError{GlobDoubleStar}
			}
			seg.atoms = append(seg.atoms, atom{kind: atomStar})
			wild = true
			i += n
		case '?':
			seg.atoms = append(seg.atoms, atom{kind: atomOne})
			wild = true
			i += n
		case '[':
			a, end, err := compileClass(p, i+1)
			if err != nil {
				return seg, "", false, err
			}
			seg.atoms = append(seg.atoms, a)
			wild = true
			i = end
		default:
			seg.atoms = append(seg.atoms, atom{kind: atomLit, r: r})
			text.WriteRune(r)
			i += n
		}
	}
	seg.dot = len(seg.atoms) > 0 && seg.atoms[0].kind == atomLit && seg.atoms[0].r == '.'
	return seg, text.String(), wild, nil
}

// compileClass reads a character class whose body begins at p[i], after
// its [, and returns it with the index after its ]. A ] first in the body,
// after any ^, is literal.
func compileClass(p string, i int) (atom, int, error) {
	a := atom{kind: atomClass}
	if i < len(p) && p[i] == '^' {
		a.negate = true
		i++
	}
	first := true
	for i < len(p) {
		if p[i] == ']' && !first {
			return a, i + 1, nil
		}
		first = false
		lo, n, err := classChar(p, i)
		if err != nil {
			return a, 0, err
		}
		i += n
		hi := lo
		if i+1 < len(p) && p[i] == '-' && p[i+1] != ']' {
			var m int
			hi, m, err = classChar(p, i+1)
			if err != nil {
				return a, 0, err
			}
			i += 1 + m
			if hi < lo {
				return a, 0, &GlobError{GlobClass}
			}
		}
		a.ranges = append(a.ranges, [2]rune{lo, hi})
	}
	return a, 0, &GlobError{GlobClass}
}

// classChar reads one character of a class at p[i], an escaped one
// included, and returns it with its length.
func classChar(p string, i int) (rune, int, error) {
	if p[i] == '\\' {
		if i+1 >= len(p) {
			return 0, 0, &GlobError{GlobEscape}
		}
		r, n := utf8.DecodeRuneInString(p[i+1:])
		return r, 1 + n, nil
	}
	r, n := utf8.DecodeRuneInString(p[i:])
	return r, n, nil
}

// Literal returns the path a glob with no wildcard names, "." for the
// root, and false for a glob with a wildcard.
func (g *Glob) Literal() (string, bool) {
	return g.literal, !g.wild
}

// Match reports whether the entry path p matches g as a whole. A glob with
// no wildcard matches the path it names and, when that is a directory,
// every path beneath it.
func (g *Glob) Match(p string) bool {
	if !g.wild {
		return g.literal == "." || p == g.literal || strings.HasPrefix(p, g.literal+"/")
	}
	segs := strings.Split(p, "/")
	// memo[i][j] records that pattern segments from i and path segments
	// from j do not match, so ** never makes the walk exponential.
	memo := make(map[[2]int]bool)
	var walk func(i, j int) bool
	walk = func(i, j int) bool {
		for i < len(g.segs) {
			s := g.segs[i]
			if s.any {
				key := [2]int{i, j}
				if memo[key] {
					return false
				}
				// ** takes zero segments, or one more that does not begin
				// with ".".
				if walk(i+1, j) {
					return true
				}
				for k := j; k < len(segs) && !strings.HasPrefix(segs[k], "."); k++ {
					if walk(i+1, k+1) {
						return true
					}
				}
				memo[key] = true
				return false
			}
			if j >= len(segs) || !s.match(segs[j]) {
				return false
			}
			i, j = i+1, j+1
		}
		return j == len(segs)
	}
	return walk(0, 0)
}

// match reports whether name, one path segment, matches s.
func (s segment) match(name string) bool {
	if strings.HasPrefix(name, ".") && !s.dot {
		return false
	}
	return matchAtoms(s.atoms, name)
}

// matchAtoms matches atoms against the whole of name, with the one
// backtracking point of the last * seen, which keeps it linear in their
// lengths' product.
func matchAtoms(atoms []atom, name string) bool {
	ai, ni := 0, 0
	starA, starN := -1, 0
	for ni < len(name) {
		r, n := utf8.DecodeRuneInString(name[ni:])
		if ai < len(atoms) {
			a := atoms[ai]
			switch {
			case a.kind == atomStar:
				starA, starN = ai, ni
				ai++
				continue
			case a.matches(r):
				ai++
				ni += n
				continue
			}
		}
		if starA < 0 {
			return false
		}
		_, sn := utf8.DecodeRuneInString(name[starN:])
		starN += sn
		ai, ni = starA+1, starN
	}
	for ai < len(atoms) && atoms[ai].kind == atomStar {
		ai++
	}
	return ai == len(atoms)
}

// matches reports whether a, which is not *, matches the character r.
func (a atom) matches(r rune) bool {
	switch a.kind {
	case atomLit:
		return a.r == r
	case atomOne:
		return r != '/'
	case atomClass:
		if r == '/' {
			return false
		}
		in := false
		for _, rg := range a.ranges {
			if r >= rg[0] && r <= rg[1] {
				in = true
				break
			}
		}
		return in != a.negate
	}
	return false
}

// Select returns the entries of entries, which are in ascending byte
// order, that g matches, in that order.
func (g *Glob) Select(entries []string) []string {
	var out []string
	for _, e := range entries {
		if g.Match(e) {
			out = append(out, e)
		}
	}
	return out
}
