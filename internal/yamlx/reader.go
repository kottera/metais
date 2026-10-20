// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package yamlx

import (
	"bytes"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/kottera/metais/internal/finding"
	"github.com/kottera/metais/internal/limits"
)

// reader is one Parse: the document's bytes, its lines, and what has been
// found in it so far.
type reader struct {
	src      Source
	manifest bool
	data     []byte
	lim      limits.Set
	// lines is the byte offset at which each line of data begins. A line
	// ends at LF, CR LF or a lone CR (ADR-006 D5 item 3).
	lines     []int
	findings  []finding.Finding
	positions *positions
	// cur caches the last offsetOf, so walking a line's nodes in order
	// costs the line's length once rather than once per node.
	cur struct{ line, col, off int }
	// steps counts the characters offsetOf has walked, which a test holds
	// to the length of the text.
	steps int
	// counted is set once src.Values has passed its bound: the value
	// that passed it is reported, and nothing after it is decoded.
	counted bool
}

func newReader(src Source, data []byte, lim limits.Set) *reader {
	r := &reader{
		src:       src,
		manifest:  src.Subject.Kind == finding.SubjectManifest,
		data:      data,
		lim:       lim,
		lines:     []int{0},
		positions: newPositions(),
	}
	return r
}

// index builds r.lines. It reads every byte, so parse calls it only once
// the size limit has passed.
func (r *reader) index() {
	data := r.data
	for i := 0; i < len(data); i++ {
		switch data[i] {
		case '\r':
			if i+1 < len(data) && data[i+1] == '\n' {
				i++
			}
			r.lines = append(r.lines, i+1)
		case '\n':
			r.lines = append(r.lines, i+1)
		}
	}
}

// at is the finding location of pos in the document, with no pointer.
// An unknown pos gives a location with no line.
func (r *reader) at(pos finding.Pos) finding.Location {
	l := r.src.At
	l.Line, l.Column, l.Pointer, l.HasPointer = 0, 0, "", false
	if pos.Known() {
		l = l.At(r.inFile(pos))
	}
	return l
}

// inFile is pos, a position in the document, as a position in its file.
func (r *reader) inFile(pos finding.Pos) finding.Pos {
	if r.src.FirstLine > 1 {
		pos.Line += r.src.FirstLine - 1
	}
	return pos
}

// add records a finding of rule at loc.
func (r *reader) add(ru rule, loc finding.Location, p finding.Params, opts ...finding.Option) {
	code := codes[ru][1]
	if r.manifest {
		code = codes[ru][0]
	}
	r.findings = append(r.findings, finding.New(code, r.src.Subject, &loc, p, opts...))
}

// limit is the finding for a breach of id at loc, observed when
// hasObserved is set: limit.package for the manifest, which is the
// package's own content, and limit.exceeded for an input (ADR-009 D19).
func (r *reader) limit(id limits.ID, loc finding.Location, observed int64, hasObserved bool) finding.Finding {
	if r.manifest {
		return r.lim.Breached(id, loc, observed, hasObserved)
	}
	return r.lim.Exceeded(id, r.src.Subject, loc, observed, hasObserved)
}

// posAt is the position of the byte at off.
func (r *reader) posAt(off int) finding.Pos {
	i, found := slices.BinarySearch(r.lines, off)
	if !found {
		i--
	}
	return finding.Pos{Line: i + 1, Column: utf8.RuneCount(r.data[r.lines[i]:off]) + 1}
}

// offsetOf is the byte offset of line and col, a position the library
// reports in the document, or -1 when the document has no such position:
// a column is a character of its line, or just past its last.
func (r *reader) offsetOf(line, col int) int {
	if line < 1 || line > len(r.lines) || col < 1 {
		return -1
	}
	if r.cur.line != line || r.cur.col > col {
		r.cur.line, r.cur.col, r.cur.off = line, 1, r.lines[line-1]
	}
	end := len(r.data)
	if line < len(r.lines) {
		end = r.lines[line]
	}
	for end > r.lines[line-1] && (r.data[end-1] == '\n' || r.data[end-1] == '\r') {
		end--
	}
	for r.cur.col < col {
		if r.cur.off >= end {
			return -1
		}
		_, n := utf8.DecodeRune(r.data[r.cur.off:end])
		r.cur.off += n
		r.cur.col++
		r.steps++
	}
	return r.cur.off
}

// scan applies the profile's rules on the bytes (ADR-002 D2 items 1 and
// 2): UTF-8, no byte order mark, no character YAML forbids, and no
// directive. It returns the text the library parses, which is data with
// each directive line blanked, and false when a refusal leaves nothing
// the library can be trusted to parse.
func (r *reader) scan() ([]byte, bool) {
	if !utf8.Valid(r.data) {
		off, reason := 0, reasonInvalidUTF8
		if bytes.HasPrefix(r.data, []byte{0xFE, 0xFF}) || bytes.HasPrefix(r.data, []byte{0xFF, 0xFE}) {
			reason = reasonUTF16
		}
		for off < len(r.data) {
			c, n := utf8.DecodeRune(r.data[off:])
			if c == utf8.RuneError && n == 1 {
				break
			}
			off += n
		}
		r.add(ruleEncoding, r.at(r.posAt(off)), finding.Params{"reason": finding.String(reason), "offset": finding.Int(int64(off))})
		return nil, false
	}
	ok := true
	if bytes.HasPrefix(r.data, []byte("\ufeff")) {
		r.add(ruleBOM, r.at(finding.Pos{Line: 1, Column: 1}), nil)
		ok = false
	}
	seen := map[rune]bool{}
	for off, c := range string(r.data) {
		reason := forbidden(c)
		if reason == "" || seen[c] {
			continue
		}
		seen[c] = true
		r.add(ruleCharacter, r.at(r.posAt(off)), finding.Params{"codepoint": finding.String(codepoint(c)), "reason": finding.String(reason)})
		ok = false
	}
	if !ok {
		return nil, false
	}
	return r.directives(), true
}

// forbidden is why the stream may not hold c, or "" when it may. YAML
// 1.2.2 §5.1 allows tab, LF, CR and the printable characters. NEL, LS and
// PS are printable text to YAML 1.2, but go.yaml.in/yaml/v3 reads each as
// a line break, which changes both a value and every position after it,
// so the profile refuses them raw; a double-quoted escape writes them.
func forbidden(c rune) string {
	switch {
	case c == 0x85 || c == 0x2028 || c == 0x2029:
		return reasonLineSeparator
	case c == '\t' || c == '\n' || c == '\r',
		0x20 <= c && c <= 0x7E,
		0xA0 <= c && c <= 0xD7FF,
		0xE000 <= c && c <= 0xFFFD,
		0x10000 <= c && c <= 0x10FFFF:
		return ""
	}
	return reasonForbidden
}

// codepoint is c written U+XXXX.
func codepoint(c rune) string { return fmt.Sprintf("U+%04X", c) }

// directives refuses each directive (ADR-002 D2 item 2) and returns data
// with every directive line blanked, so the library neither applies a
// %TAG nor refuses a %YAML 1.2, and every position stays where it was. A
// directive is a line beginning with % before the first document's
// content; only blank and comment lines may come between them.
func (r *reader) directives() []byte {
	text, cloned := r.data, false
	for i, start := range r.lines {
		end := len(r.data)
		if i+1 < len(r.lines) {
			end = r.lines[i+1]
		}
		line := r.data[start:end]
		if len(line) > 0 && line[0] == '%' {
			r.add(ruleDirective, r.at(finding.Pos{Line: i + 1, Column: 1}), nil)
			if !cloned {
				text, cloned = slices.Clone(r.data), true
			}
			for j := start; j < end && text[j] != '\n' && text[j] != '\r'; j++ {
				text[j] = ' '
			}
			continue
		}
		if t := strings.TrimLeft(string(line), " \t"); t != "" && t[0] != '\n' && t[0] != '\r' && t[0] != '#' {
			break
		}
	}
	return text
}

// The problems go.yaml.in/yaml/v3's parser, as opposed to its scanner,
// reports (parserc.go). The library's error names the line of the
// problem's context mark, or its problem mark, and only when that mark is
// not on the first line; a scanner error names it 1-based and a parser
// error 0-based.
var parserProblems = []string{
	"did not find expected <document start>",
	"did not find expected <stream-start>",
	"found duplicate %TAG directive",
	"found duplicate %YAML directive",
	"found incompatible YAML document",
	"did not find expected ',' or ']'",
	"did not find expected ',' or '}'",
	"did not find expected '-' indicator",
	"did not find expected key",
	"did not find expected node content",
	"found undefined tag handle",
}

// syntax records the one finding for err, the library's refusal of the
// document's syntax (ADR-002 D2 item 11). Its line is the one the
// library names; the library gives no column, so the finding has none.
// The library's message is read for its line only and never reaches the
// finding (ADR-006 D9 item 4). Nesting beyond the library's own depth is
// past value-depth too, and is reported as that limit.
func (r *reader) syntax(err error) {
	msg, _ := strings.CutPrefix(err.Error(), "yaml: ")
	line := 1
	if rest, ok := strings.CutPrefix(msg, "line "); ok {
		digits, problem, _ := strings.Cut(rest, ": ")
		n, convErr := strconv.Atoi(digits)
		if convErr != nil {
			n = 0
		}
		line, msg = n, problem
		if slices.ContainsFunc(parserProblems, func(p string) bool { return strings.HasPrefix(msg, p) }) {
			line++
		}
	}
	pos := finding.Pos{Line: line}
	if line < 1 || line > len(r.lines) {
		pos = finding.Pos{}
	}
	if strings.HasPrefix(msg, "exceeded max depth of ") {
		r.findings = append(r.findings, r.limit(limits.ValueDepth, r.at(pos), 0, false))
		return
	}
	r.add(ruleSyntax, r.at(pos), nil)
}
