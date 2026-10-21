// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package sink

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/kottera/metais/internal/finding"
	"github.com/kottera/metais/internal/limits"
	"github.com/kottera/metais/internal/pkgtree"
	"github.com/kottera/metais/internal/render"
	"github.com/kottera/metais/internal/zipfmt"
)

// The comparison's codes (ADR-007 D5 item 3), and its caps on a diff
// (item 4).
const (
	codeOutputMissing      finding.Code = "example.output_missing"
	codeOutputUnexpected   finding.Code = "example.output_unexpected"
	codeOutputDiffers      finding.Code = "example.output_differs"
	codeExpectedUnreadable finding.Code = "example.expected_unreadable"
	codeEntryMissing       finding.Code = "example.entry_missing"
	codeEntryUnexpected    finding.Code = "example.entry_unexpected"
	codeEntryOrder         finding.Code = "example.entry_order"
	codeEntryMethod        finding.Code = "example.entry_method"
	MaxDiffJSON                         = 16 << 10 // a diff's cap in a finding (ADR-007 D5 item 4)
	MaxDiffHuman                        = 40       // and in human output, in lines
	DiffTruncatedNotice                 = "... (the difference is truncated)\n"
)

// ComparisonCodes are the codes of a difference between a render and
// expected/, which a real --update makes nothing.
var ComparisonCodes = []finding.Code{codeOutputMissing, codeOutputUnexpected, codeOutputDiffers,
	codeExpectedUnreadable, codeEntryMissing, codeEntryUnexpected, codeEntryOrder, codeEntryMethod}

// Comparison is test's sink (ADR-007 D5 items 3 and 4): it holds an
// example's render in memory, every file's bytes as a render writes them,
// a template's, an asset's from the package, or a container's zip, and
// compares them with the example's expected/, when it has one. It writes
// nothing, and checks nothing in phase 3. Once Emit has run, its fields
// are the render and the comparison's account of it.
type Comparison struct {
	// tree is the package, dir its example's expected/, and expected and
	// hasExpected what Compare was given.
	tree        pkgtree.Tree
	lim         limits.Set
	subj        finding.Subject
	dir         string
	expected    []string
	hasExpected bool
	// containers holds the plan of every container output the render
	// wrote, by path, which is compared by its entries.
	containers map[string]*render.File

	// Rendered holds the bytes of every file the render wrote, by path,
	// and Executable whether each is executable.
	Rendered   map[string][]byte
	Executable map[string]bool
	// Findings are the comparison's, Differences each path it found
	// different, and Human the human report of each, in order.
	Findings    []finding.Finding
	Differences []Difference
	Human       []string
	// Rewrite holds the paths of the rendered files the comparison found
	// different from expected/'s or absent from it: those --update
	// writes.
	Rewrite []string
}

// Difference is one path the comparison found different: kind is
// missing, unexpected or differs, and line is a text difference's first
// differing line in the expected file.
type Difference struct {
	Path string
	Kind string
	Line *int
}

// Compare is the sink of the example called name, which test renders
// into memory: dir is its expected/ in tree, examples/<name>/expected/,
// expected the paths of its files, relative to it, in byte order, and
// hasExpected whether it exists. Without it, the render is held and not
// compared.
func Compare(tree pkgtree.Tree, lim limits.Set, name, dir string, expected []string, hasExpected bool) *Comparison {
	return &Comparison{tree: tree, lim: lim, subj: finding.Subject{Kind: finding.SubjectExample, Name: name},
		dir: dir, expected: expected, hasExpected: hasExpected}
}

func (*Comparison) Check([]string) []finding.Finding { return nil }
func (*Comparison) Status(string) string             { return "" }
func (*Comparison) Close()                           {}

// Emit holds files in memory, and compares them with expected/ when the
// example has one.
func (c *Comparison) Emit(files []*render.File) Emission {
	c.Rendered, c.Executable, c.containers = map[string][]byte{}, map[string]bool{}, map[string]*render.File{}
	for _, pf := range files {
		if pf.IsContainer() {
			c.containers[pf.Path()] = pf
		}
		r, done := pf.Open()
		b, err := io.ReadAll(r)
		if derr := done(err); err != nil || derr != nil {
			panic("sink: a rendered file does not read: " + errors.Join(err, derr).Error())
		}
		c.Rendered[pf.Path()], c.Executable[pf.Path()] = b, pf.Executable()
	}
	if c.hasExpected {
		c.compare()
	}
	return Emission{}
}

// readEntry reads the package entry name of size bytes, which the
// package's open admitted under entry-size.
func readEntry(tree pkgtree.Tree, name string, size int64) ([]byte, error) {
	r, err := tree.Open(name)
	if err != nil {
		return nil, err
	}
	defer r.Close()
	b := make([]byte, size)
	_, err = io.ReadFull(r, b)
	return b, err
}

// compare compares the render with expected/ (ADR-007 D5 items 3 and
// 4), by path in byte order, directories ignored: a container output by
// its entries (compareContainer), and every other file byte for byte.
func (c *Comparison) compare() {
	subj := c.subj
	paths := slices.Clone(c.expected)
	for p := range c.Rendered {
		if !slices.Contains(c.expected, p) {
			paths = append(paths, p)
		}
	}
	slices.Sort(paths)
	for _, p := range paths {
		entry := c.dir + p
		got, written := c.Rendered[p]
		if !slices.Contains(c.expected, p) {
			l := finding.Location{Origin: finding.OriginOutput, File: p}
			c.Findings = append(c.Findings, finding.New(codeOutputUnexpected, subj, &l, finding.Params{"path": finding.String(p)}))
			c.Differences = append(c.Differences, Difference{Path: p, Kind: "unexpected"})
			c.Human = append(c.Human, "  unexpected  "+string(finding.AppendText(nil, p))+"\n")
			c.Rewrite = append(c.Rewrite, p)
			continue
		}
		l := finding.Location{Origin: finding.OriginPackage, File: entry}
		if !written {
			c.Findings = append(c.Findings, finding.New(codeOutputMissing, subj, &l, finding.Params{"path": finding.String(p)}))
			c.Differences = append(c.Differences, Difference{Path: p, Kind: "missing"})
			c.Human = append(c.Human, "  missing     "+string(finding.AppendText(nil, p))+"\n")
			continue
		}
		info, err := c.tree.Stat(entry)
		if err != nil {
			panic("sink: " + entry + " is listed and not found")
		}
		want, err := readEntry(c.tree, entry, info.Size)
		if err != nil {
			panic("sink: reading " + entry + ": " + err.Error())
		}
		before := len(c.Differences)
		if pf := c.containers[p]; pf != nil {
			c.compareContainer(want, pf, containerAt{subj: subj, path: p, file: entry})
		} else if !bytes.Equal(want, got) {
			f, d, human := describeDifference(want, got, p, "", subj, l)
			c.Findings = append(c.Findings, f)
			c.Differences = append(c.Differences, d)
			c.Human = append(c.Human, human)
		}
		if len(c.Differences) > before {
			c.Rewrite = append(c.Rewrite, p)
		}
	}
}

// containerAt is where a container compareContainer compares is: the
// example's subject, the output's path, the expected file it is read
// from, and, for a container that is an entry of another, its entry path
// through every enclosing container, joined by "!".
type containerAt struct {
	subj  finding.Subject
	path  string
	file  string
	entry string
}

// within is the location of a's entry name: the output's path, or the
// expected file's, then each entry path, joined by "!".
func (a containerAt) within(base, name string) string {
	if a.entry != "" {
		base += "!" + a.entry
	}
	if name != "" {
		base += "!" + name
	}
	return base
}

// params is the parameters every finding about a's entry name carries.
func (a containerAt) params(name string) finding.Params {
	ps := finding.Params{"path": finding.String(a.path)}
	if n := a.within("", name); n != "" {
		ps["entry"] = finding.String(n[1:])
	}
	return ps
}

// compareContainer compares want, a container's zip from expected/, with
// pf, the container the render wrote, by entries (ADR-007 D5 item 3, R9
// item 5): want is read with the package reader's bounds, and one that does
// not read is examplc.expected_unreadable; then the entry names, an entry
// expected and not written being example.entry_missing and one written
// and not expected example.entry_unexpected; the order of the names both
// hold, example.entry_order at the first index where it differs; and, for
// each entry both hold, in want's order, its uncompressed content, by
// example.output_differs, or a container entry's own entries, and the
// method of an entry whose declaration says stored: true, or a profile's
// mimetype, by example.entry_method. The method of any other entry is the
// encoder's choice (ADR-005 D8 item 7) and is not compared.
func (c *Comparison) compareContainer(want []byte, pf *render.File, at containerAt) {
	decoded := map[string]*bytes.Buffer{}
	a, rfs, err := zipfmt.OpenContainer(bytes.NewReader(want), int64(len(want)), c.lim, func(name string) io.Writer {
		b := &bytes.Buffer{}
		decoded[name] = b
		return b
	})
	if err != nil {
		panic("sink: reading a container from memory: " + err.Error())
	}
	here := at.within(at.path, "")
	if len(rfs) > 0 {
		l := finding.Location{Origin: finding.OriginPackage, File: at.within(at.file, "")}
		ps := at.params("")
		ps["reason"] = finding.String(string(rfs[0].Code()))
		c.Findings = append(c.Findings, finding.New(codeExpectedUnreadable, at.subj, &l, ps))
		c.Differences = append(c.Differences, Difference{Path: here, Kind: "differs"})
		c.Human = append(c.Human, "  differs     "+string(finding.AppendText(nil, here))+" (not a readable zip: "+string(rfs[0].Code())+")\n")
		return
	}
	expected := a.Entries()
	rendered := map[string]*render.File{}
	var wantNames, gotNames []string
	deflated := map[string]bool{}
	for _, x := range expected {
		wantNames, deflated[x.Name] = append(wantNames, x.Name), x.Deflated
	}
	for _, x := range pf.Entries() {
		gotNames, rendered[x.Path()] = append(gotNames, x.Path()), x
	}
	expectedAt := func(name string) *finding.Location {
		return &finding.Location{Origin: finding.OriginPackage, File: at.within(at.file, name)}
	}
	add := func(f finding.Finding, d Difference, human string) {
		c.Findings = append(c.Findings, f)
		c.Differences = append(c.Differences, d)
		c.Human = append(c.Human, human)
	}
	var shared []string
	for _, n := range wantNames {
		if rendered[n] == nil {
			p := at.within(at.path, n)
			add(finding.New(codeEntryMissing, at.subj, expectedAt(n), at.params(n)),
				Difference{Path: p, Kind: "missing"}, "  missing     "+string(finding.AppendText(nil, p))+"\n")
			continue
		}
		shared = append(shared, n)
	}
	var sharedGot []string
	for _, n := range gotNames {
		if _, ok := deflated[n]; !ok {
			p := at.within(at.path, n)
			l := finding.Location{Origin: finding.OriginOutput, File: p}
			add(finding.New(codeEntryUnexpected, at.subj, &l, at.params(n)),
				Difference{Path: p, Kind: "unexpected"}, "  unexpected  "+string(finding.AppendText(nil, p))+"\n")
			continue
		}
		sharedGot = append(sharedGot, n)
	}
	for i := range shared {
		if shared[i] == sharedGot[i] {
			continue
		}
		ps := at.params("")
		ps["index"] = finding.Int(int64(i))
		ps["expected"] = finding.String(shared[i])
		ps["rendered"] = finding.String(sharedGot[i])
		add(finding.New(codeEntryOrder, at.subj, expectedAt(shared[i]), ps),
			Difference{Path: here, Kind: "differs"},
			"  differs     "+string(finding.AppendText(nil, here))+" (entry order: at "+strconv.Itoa(i)+
				" expected "+string(finding.AppendText(nil, shared[i]))+", rendered "+string(finding.AppendText(nil, sharedGot[i]))+")\n")
		break
	}
	for _, n := range shared {
		x := rendered[n]
		w := decoded[n].Bytes()
		if x.IsContainer() {
			inner := at
			inner.entry = at.within("", n)[1:]
			c.compareContainer(w, x, inner)
		} else {
			r, done := x.Open()
			g, err := io.ReadAll(r)
			if derr := done(err); err != nil || derr != nil {
				panic("sink: a rendered entry does not read: " + errors.Join(err, derr).Error())
			}
			if !bytes.Equal(w, g) {
				add(describeDifference(w, g, at.path, at.within("", n)[1:], at.subj, *expectedAt(n)))
			}
		}
		if x.Stored() && deflated[n] {
			p := at.within(at.path, n)
			ps := at.params(n)
			ps["expected_method"], ps["rendered_method"] = finding.String("deflated"), finding.String("stored")
			add(finding.New(codeEntryMethod, at.subj, expectedAt(n), ps),
				Difference{Path: p, Kind: "differs"}, "  differs     "+string(finding.AppendText(nil, p))+" (method: expected deflated, rendered stored)\n")
		}
	}
}

// describeDifference is the finding, the result's record and the human
// lines of the difference between want, the expected file at l, and got,
// the render's file at p, or its entry entry when entry is not empty
// (ADR-007 D5 items 3 and 4).
func describeDifference(want, got []byte, p, entry string, subj finding.Subject, l finding.Location) (finding.Finding, Difference, string) {
	cause, crlf := differenceCause(want, got)
	params := finding.Params{"path": finding.String(p), "cause": finding.String(cause)}
	if crlf != "" {
		params["crlf"] = finding.String(crlf)
	}
	if entry != "" {
		params["entry"] = finding.String(entry)
		p += "!" + entry
	}
	d := Difference{Path: p, Kind: "differs"}
	human := "  differs     " + string(finding.AppendText(nil, p))
	if isText(want) && isText(got) {
		pos := firstDifference(want, got)
		l = l.At(pos)
		line := pos.Line
		d.Line = &line
		diff, cut := unifiedDiff(want, got, "expected/"+p, "rendered/"+p, MaxDiffJSON)
		params["binary"] = finding.Bool(false)
		params["diff"] = finding.String(capBytes(diff, MaxDiffJSON))
		params["diff_truncated"] = finding.Bool(cut || len(diff) > MaxDiffJSON)
		human += " (line " + strconv.Itoa(line) + ", " + cause + ")\n" + indent(capLines(diff, MaxDiffHuman), "    ")
	} else {
		params["binary"] = finding.Bool(true)
		off := 0
		for off < len(want) && off < len(got) && want[off] == got[off] {
			off++
		}
		params["offset"] = finding.Int(int64(off))
		params["expected_size"] = finding.Int(int64(len(want)))
		params["rendered_size"] = finding.Int(int64(len(got)))
		params["expected_digest"] = finding.String(sha256Text(want))
		params["rendered_digest"] = finding.String(sha256Text(got))
		human += " (binary, first difference at byte " + strconv.Itoa(off) + ", " + cause + ")\n" +
			"    expected " + strconv.Itoa(len(want)) + " bytes " + sha256Text(want) + "\n" +
			"    rendered " + strconv.Itoa(len(got)) + " bytes " + sha256Text(got) + "\n"
	}
	return finding.New(codeOutputDiffers, subj, &l, params), d, human
}

// isText reports whether b is text in ADR-007 D5 item 4's sense: valid
// UTF-8 holding no NUL byte.
func isText(b []byte) bool { return utf8.Valid(b) && bytes.IndexByte(b, 0) < 0 }

func sha256Text(b []byte) string {
	sum := sha256.Sum256(b)
	return "sha256:" + hex.EncodeToString(sum[:])
}

// differenceCause is the first of ADR-007 D5 item 4's hints that applies
// to want and got, and for line endings the side that holds CR LF.
func differenceCause(want, got []byte) (cause, crlf string) {
	lf := func(b []byte) []byte { return bytes.ReplaceAll(b, []byte("\r\n"), []byte("\n")) }
	if bytes.Equal(lf(want), lf(got)) {
		w, g := bytes.Contains(want, []byte("\r\n")), bytes.Contains(got, []byte("\r\n"))
		switch {
		case w && g:
			return "line_endings", "both"
		case w:
			return "line_endings", "expected"
		}
		return "line_endings", "rendered"
	}
	if bytes.Equal(append(slices.Clip(want), '\n'), got) || bytes.Equal(want, append(slices.Clip(got), '\n')) {
		return "final_newline", ""
	}
	bom := []byte("\uFEFF")
	if bytes.Equal(bytes.TrimPrefix(want, bom), bytes.TrimPrefix(got, bom)) {
		return "bom", ""
	}
	if bytes.Equal(trimLineEnds(want), trimLineEnds(got)) {
		return "trailing_whitespace", ""
	}
	return "none", ""
}

// trimLineEnds is b with the spaces and tabs ending each line removed.
func trimLineEnds(b []byte) []byte {
	lines := bytes.SplitAfter(b, []byte("\n"))
	var out []byte
	for _, l := range lines {
		body, nl := l, []byte(nil)
		if bytes.HasSuffix(body, []byte("\n")) {
			body, nl = body[:len(body)-1], []byte("\n")
			if bytes.HasSuffix(body, []byte("\r")) {
				body, nl = body[:len(body)-1], []byte("\r\n")
			}
		}
		out = append(append(out, bytes.TrimRight(body, " \t")...), nl...)
	}
	return out
}

// firstDifference is the position in want of the first code point at
// which want and got differ, its line and column as ADR-006 counts them:
// columns in code points from 1, a line ended by LF, CR LF or a lone CR.
// When one is a prefix of the other it is the position just past the
// shorter's end in want.
func firstDifference(want, got []byte) finding.Pos {
	i := 0
	for i < len(want) && i < len(got) && want[i] == got[i] {
		i++
	}
	// Back to the start of the code point the difference is in.
	for i > 0 && i < len(want) && !utf8.RuneStart(want[i]) {
		i--
	}
	pos := finding.Pos{Line: 1, Column: 1}
	for j := 0; j < i; {
		switch c := want[j]; {
		case c == '\r' && j+1 < len(want) && want[j+1] == '\n':
			if j+1 == i {
				// The difference is inside a CR LF: at the CR.
				return pos
			}
			pos.Line, pos.Column = pos.Line+1, 1
			j += 2
		case c == '\r' || c == '\n':
			pos.Line, pos.Column = pos.Line+1, 1
			j++
		default:
			_, n := utf8.DecodeRune(want[j:])
			pos.Column++
			j += n
		}
	}
	return pos
}

// capBytes is s cut at max bytes, at a code point's start, with the
// truncation notice when it was cut.
func capBytes(s string, max int) string {
	if len(s) <= max {
		return s
	}
	cut := max - len(DiffTruncatedNotice)
	for cut > 0 && !utf8.RuneStart(s[cut]) {
		cut--
	}
	return s[:cut] + DiffTruncatedNotice
}

// capLines is s cut to max lines, with the truncation notice when it was
// cut.
func capLines(s string, max int) string {
	n := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			n++
			if n == max && i+1 < len(s) {
				return s[:i+1] + DiffTruncatedNotice
			}
		}
	}
	return s
}

// indent prefixes every line of s with pre.
func indent(s, pre string) string {
	if s == "" {
		return s
	}
	lines := strings.SplitAfter(s, "\n")
	var b strings.Builder
	for _, l := range lines {
		if l != "" {
			b.WriteString(pre + l)
		}
	}
	return b.String()
}
