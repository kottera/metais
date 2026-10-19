// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package guarantees

import (
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"
)

// This file reads the specification as ADR-011 D1 to D3 lay it out: the
// Markdown documents under spec/, the blocks they are made of, and the
// statements those blocks open. The D14 tripwires in spec*_test.go are
// built on it.

// specRoot is spec/, from this package's directory, where go test runs.
const specRoot = moduleRoot + "/spec"

// vectorsDir is the vector suite, below spec/ (ADR-011 D9 item 1). Its
// Markdown files are package and expected files, not the document.
const vectorsDir = "conformance/vectors"

// mdDoc is one Markdown document of the specification.
type mdDoc struct {
	path   string   // slash-separated, below spec/
	lines  []string // without line ends
	fenced []bool   // the line is a fence or inside a fenced code block
	fences []fenceBlock
	blocks []mdBlock
}

// fenceBlock is a fenced code block: lines open to close, both fences
// included. An unclosed fence runs to the end of the document.
type fenceBlock struct {
	open, close int // line indexes
	indent      int // the opening fence's indentation
	info        string
}

// body is the block's content as a file would hold it: each line with
// the opening fence's indentation removed, and every line, the last
// included, ending in LF (ADR-011 D4 item 2).
func (f fenceBlock) body(d *mdDoc) string {
	var b strings.Builder
	for _, l := range d.lines[f.open+1 : f.close] {
		n := min(f.indent, len(l)-len(strings.TrimLeft(l, " ")))
		b.WriteString(l[n:])
		b.WriteByte('\n')
	}
	return b.String()
}

// mdBlock is a block a statement may open, or one that bounds another.
type mdBlock struct {
	kind   byte // 'p' paragraph, 'l' list item, 'r' table row, 'h' heading, 'q' block quote
	start  int  // first line
	end    int  // one past the last line of the block's scope
	indent int
	lead   string // the first line's content after the block's marker
}

var (
	fenceOpenRe = regexp.MustCompile("^( *)(`{3,}|~{3,})(.*)$")
	headingRe   = regexp.MustCompile(`^ {0,3}(#{1,6})(?: +(.*?))?\s*$`)
	listItemRe  = regexp.MustCompile(`^( *)(?:[-*+]|[0-9]{1,9}[.)])(?: +(.*))?$`)
	tableRowRe  = regexp.MustCompile(`^ *\|(.*)$`)
	quoteRe     = regexp.MustCompile(`^ *>`)
)

// specDocuments returns the slash-separated path, below root, of every
// Markdown document of the specification: every .md file under root but
// those of the vector suite.
func specDocuments(root string) ([]string, error) {
	var docs []string
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, p)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if d.IsDir() {
			if rel == vectorsDir || (rel != "." && strings.HasPrefix(d.Name(), ".")) {
				return filepath.SkipDir
			}
			return nil
		}
		if d.Type().IsRegular() && strings.HasSuffix(rel, ".md") {
			docs = append(docs, rel)
		}
		return nil
	})
	slices.Sort(docs)
	return docs, err
}

// readDoc reads and scans the document at rel below root.
func readDoc(root, rel string) (*mdDoc, error) {
	b, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
	if err != nil {
		return nil, err
	}
	return scanDoc(rel, string(b)), nil
}

// readDocs reads every document of the specification under root.
func readDocs(root string) ([]*mdDoc, error) {
	names, err := specDocuments(root)
	if err != nil {
		return nil, err
	}
	var docs []*mdDoc
	for _, name := range names {
		d, err := readDoc(root, name)
		if err != nil {
			return nil, err
		}
		docs = append(docs, d)
	}
	return docs, nil
}

func indentOf(l string) int { return len(l) - len(strings.TrimLeft(l, " ")) }

func blank(l string) bool { return strings.TrimSpace(l) == "" }

// scanDoc splits text into fenced code blocks and the blocks around
// them. It reads the Markdown this specification is written in —
// paragraphs, nested lists, pipe tables, block quotes and fences — and no
// more.
func scanDoc(rel, text string) *mdDoc {
	d := &mdDoc{path: rel, lines: strings.Split(strings.TrimSuffix(text, "\n"), "\n")}
	if text == "" {
		d.lines = nil
	}
	d.fenced = make([]bool, len(d.lines))
	for i := 0; i < len(d.lines); i++ {
		m := fenceOpenRe.FindStringSubmatch(d.lines[i])
		if m == nil || (m[2][0] == '`' && strings.Contains(m[3], "`")) {
			continue
		}
		f := fenceBlock{open: i, close: len(d.lines) - 1, indent: len(m[1]), info: strings.TrimSpace(m[3])}
		closeRe := regexp.MustCompile("^ *" + regexp.QuoteMeta(m[2][:1]) + "{" + fmt.Sprint(len(m[2])) + ",} *$")
		for j := i + 1; j < len(d.lines); j++ {
			if closeRe.MatchString(d.lines[j]) {
				f.close = j
				break
			}
		}
		for j := f.open; j <= f.close; j++ {
			d.fenced[j] = true
		}
		d.fences = append(d.fences, f)
		i = f.close
	}
	d.blocks = scanBlocks(d)
	return d
}

// lineKind says which block a line begins when read where no paragraph is
// open — a heading, a quote, a table row, a list item or a paragraph — with
// the line's content after the block's marker, and its indentation.
func lineKind(l string) (byte, string, int) {
	if m := headingRe.FindStringSubmatch(l); m != nil {
		return 'h', m[2], indentOf(l)
	}
	if quoteRe.MatchString(l) {
		return 'q', strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(l), ">")), indentOf(l)
	}
	if m := tableRowRe.FindStringSubmatch(l); m != nil {
		return 'r', strings.TrimLeft(m[1], " "), indentOf(l)
	}
	if m := listItemRe.FindStringSubmatch(l); m != nil {
		return 'l', m[2], len(m[1])
	}
	return 'p', strings.TrimSpace(l), indentOf(l)
}

func scanBlocks(d *mdDoc) []mdBlock {
	var blocks []mdBlock
	n := len(d.lines)
	// fenceIndent is the indentation of the fence a fenced line belongs to.
	fenceIndent := make([]int, n)
	for _, f := range d.fences {
		for j := f.open; j <= f.close; j++ {
			fenceIndent[j] = f.indent
		}
	}
	prevOpen := byte(0) // the kind of block the previous line belongs to; 0 after a blank or a fence
	for i := 0; i < n; i++ {
		l := d.lines[i]
		if d.fenced[i] || blank(l) {
			prevOpen = 0
			continue
		}
		kind, lead, indent := lineKind(l)
		if kind == 'p' && (prevOpen == 'p' || prevOpen == 'l' || prevOpen == 'q') {
			continue // a continuation line
		}
		b := mdBlock{kind: kind, start: i, end: i + 1, indent: indent, lead: lead}
		switch kind {
		case 'p', 'q':
			for b.end < n && !d.fenced[b.end] && !blank(d.lines[b.end]) {
				if k, _, _ := lineKind(d.lines[b.end]); k != 'p' && !(kind == 'q' && k == 'q') {
					break
				}
				b.end++
			}
		case 'l':
		item:
			for b.end < n {
				nl := d.lines[b.end]
				switch {
				case d.fenced[b.end]:
					if fenceIndent[b.end] <= indent {
						break item
					}
				case blank(nl):
					k := b.end + 1
					for k < n && blank(d.lines[k]) {
						k++
					}
					if k == n || indentOf(d.lines[k]) <= indent {
						break item
					}
				default:
					if k, _, _ := lineKind(nl); k != 'p' && indentOf(nl) <= indent {
						break item
					}
				}
				b.end++
			}
			for b.end > i+1 && blank(d.lines[b.end-1]) {
				b.end--
			}
		}
		blocks = append(blocks, b)
		prevOpen = kind
		if kind == 'h' || kind == 'r' {
			prevOpen = 0
		}
	}
	return blocks
}

// statement is one statement of the specification (ADR-011 D3).
type statement struct {
	ID         string
	File       string // the document, below spec/
	Line       int    // 1-based
	Section    string // the number of the innermost numbered heading above it
	Kind       string // requirement, recommendation, permission; empty when it holds no keyword
	Since      string
	Observable bool
	Withdrawn  string // the erratum that withdrew it
	AmendedBy  []string
	Text       string // its scope, whitespace collapsed
	lines      []int  // the line indexes of its scope, fences excluded
}

func (s statement) anchor() string { return strings.ToLower(s.ID) }

func (s statement) active() bool { return s.Withdrawn == "" }

// area is the statement's area code.
func (s statement) area() string { return s.ID[4:7] }

// statementLess orders identifiers by area, then by number.
func statementLess(a, b string) int {
	if c := strings.Compare(a[:8], b[:8]); c != 0 {
		return c
	}
	if len(a) != len(b) {
		return len(a) - len(b)
	}
	return strings.Compare(a, b)
}

var (
	// headCandidateRe is a block lead that sets out to be a statement
	// head; statementHeadRe is ADR-011 D3 item 1's statement-head.
	headCandidateRe = regexp.MustCompile(`(?i)^(?:<a\s[^>]*id\s*=\s*"?mts-|\*\*mts-)`)
	tableIDLeadRe   = regexp.MustCompile(`^MTS-[A-Z]{3}-[0-9]`)
	statementHeadRe = regexp.MustCompile(`^<a id="([^"]*)"></a>(\*\*)?(MTS-([A-Z]{3})-([0-9]+))(\*\*)?(?: \*\(([^)]*)\)\*)? `)
	anchorRe        = regexp.MustCompile(`(?i)<a\s[^>]*\bid\s*=\s*"?mts-`)
	versionRe       = regexp.MustCompile(`^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$`)
	erratumRe       = regexp.MustCompile(`^E-(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)-[0-9]{3,}$`)
	numberedRe      = regexp.MustCompile(`^((?:[0-9]+|[A-Z])(?:\.[0-9]+)*)\.? `)
	keywordRe       = regexp.MustCompile(`\b(?:MUST NOT|MUST|SHOULD NOT|SHOULD|MAY|SHALL NOT|SHALL|NOT RECOMMENDED|RECOMMENDED|REQUIRED|OPTIONAL)\b`)
	shouldRe        = regexp.MustCompile(`\bSHOULD\b`)
	itemMarkerRe    = regexp.MustCompile(`^(?:[-*+]|[0-9]{1,9}[.)])$`)
	synonymRe       = regexp.MustCompile(`\b(?:SHALL NOT|SHALL|NOT RECOMMENDED|RECOMMENDED|REQUIRED|OPTIONAL)\b`)
)

// bcp14 is RFC 8174 §2's boilerplate, which ADR-011 D2 item 1 places once,
// in 01-introduction.md: the one text where the synonyms appear, and the
// one place a key word stands outside a statement.
const bcp14 = `The key words "MUST", "MUST NOT", "REQUIRED", "SHALL", "SHALL NOT", "SHOULD", "SHOULD NOT", "RECOMMENDED", "NOT RECOMMENDED", "MAY", and "OPTIONAL" in this document are to be interpreted as described in BCP 14 [RFC2119] [RFC8174] when, and only when, they appear in all capitals, as shown here.`

// bcp14Home is the document that carries bcp14.
const bcp14Home = "01-introduction.md"

func collapse(s string) string { return strings.Join(strings.Fields(s), " ") }

// scope is lines start to end of d, fences excluded, collapsed.
func (d *mdDoc) scope(start, end int) (string, []int) {
	var parts []string
	var lines []int
	for j := start; j < end; j++ {
		if !d.fenced[j] {
			parts = append(parts, d.lines[j])
			lines = append(lines, j)
		}
	}
	return collapse(strings.Join(parts, "\n")), lines
}

// boilerplate returns the line indexes of the bcp14 paragraph in d, which
// only bcp14Home may hold.
func (d *mdDoc) boilerplate() [][]int {
	var found [][]int
	for _, b := range d.blocks {
		if b.kind != 'p' {
			continue
		}
		if text, lines := d.scope(b.start, b.end); text == bcp14 {
			found = append(found, lines)
		}
	}
	return found
}

// statements parses every statement d opens, and reports each statement
// head that breaks ADR-011 D3 item 1's grammar and each statement anchor
// that opens no block.
func (d *mdDoc) statements() ([]statement, []string) {
	var stmts []statement
	var problems []string
	heads := map[int]bool{} // lines whose lead was read as a head
	section := ""
	for _, b := range d.blocks {
		where := fmt.Sprintf("%s:%d", d.path, b.start+1)
		if b.kind == 'h' {
			if m := numberedRe.FindStringSubmatch(b.lead + " "); m != nil {
				section = m[1]
			}
			continue
		}
		lead := b.lead
		inTable := b.kind == 'r'
		if !(b.kind == 'p' || b.kind == 'l' || inTable) {
			continue
		}
		if !headCandidateRe.MatchString(lead) && !(inTable && tableIDLeadRe.MatchString(lead)) {
			continue
		}
		heads[b.start] = true
		m := statementHeadRe.FindStringSubmatch(lead)
		if m == nil {
			problems = append(problems, fmt.Sprintf("%s: the statement head %.60q does not match ADR-011 D3 item 1's grammar", where, lead))
			continue
		}
		s := statement{ID: m[3], File: d.path, Line: b.start + 1, Section: section, Since: "1.0", Observable: true}
		bad := func(format string, args ...any) {
			problems = append(problems, fmt.Sprintf("%s: %s ", where, s.ID)+fmt.Sprintf(format, args...))
		}
		ok := true
		if num := m[5]; len(num) < 3 || (len(num) > 3 && num[0] == '0') {
			bad("is not numbered with at least three digits, zero-padded only to three")
			ok = false
		}
		if bold := m[2] != "" || m[6] != ""; bold == inTable || m[2] != m[6] {
			if inTable {
				bad("is bold inside a table cell; a table row's identifier is bare")
			} else {
				bad("is not bold; outside a table cell the identifier is **bold**")
			}
			ok = false
		}
		if m[1] != s.anchor() {
			bad("has the anchor %q, not its identifier in lower case", m[1])
			ok = false
		}
		if m[7] != "" || strings.Contains(m[0], "*()*") {
			seen := map[string]bool{}
			for _, a := range strings.Split(m[7], "; ") {
				kind, val, _ := strings.Cut(a, " ")
				switch {
				case kind == "since" && versionRe.MatchString(val):
					s.Since = val
				case a == "not observable":
					s.Observable = false
				case strings.HasPrefix(a, "amended by ") && erratumRe.MatchString(strings.TrimPrefix(a, "amended by ")):
					e := strings.TrimPrefix(a, "amended by ")
					if slices.Contains(s.AmendedBy, e) {
						bad("names amended by %s twice", e)
						ok = false
					}
					s.AmendedBy = append(s.AmendedBy, e)
					continue
				case strings.HasPrefix(a, "withdrawn by ") && erratumRe.MatchString(strings.TrimPrefix(a, "withdrawn by ")):
					s.Withdrawn = strings.TrimPrefix(a, "withdrawn by ")
					kind = "withdrawn"
				default:
					bad("has the attribute %q, which ADR-011 D3 item 1 does not define", a)
					ok = false
					continue
				}
				if seen[kind] {
					bad("gives the attribute %q twice", kind)
					ok = false
				}
				seen[kind] = true
			}
		}
		if !ok {
			continue
		}
		s.Text, s.lines = d.scope(b.start, b.end)
		if inTable {
			s.Text, s.lines = collapse(d.lines[b.start]), []int{b.start}
		}
		s.Kind = keywordKind(s.Text)
		stmts = append(stmts, s)
	}
	for i, l := range d.lines {
		if d.fenced[i] {
			continue
		}
		locs := anchorRe.FindAllStringIndex(l, -1)
		for k, loc := range locs {
			if k == 0 && heads[i] && strings.TrimSpace(l[:loc[0]]) == leadPrefix(l, loc[0]) {
				continue
			}
			problems = append(problems, fmt.Sprintf("%s:%d: a statement anchor that does not begin a paragraph, a list item or a table row", d.path, i+1))
		}
	}
	return stmts, problems
}

// leadPrefix is the block marker before a head that starts at at: the
// list marker or the table's opening pipe, or nothing.
func leadPrefix(l string, at int) string {
	p := strings.TrimSpace(l[:at])
	if p == "|" || itemMarkerRe.MatchString(p) {
		return p
	}
	return ""
}

// keywordKind is a statement's kind by its key words (ADR-011 D3 item 6).
func keywordKind(text string) string {
	kws := keywordRe.FindAllString(text, -1)
	switch {
	case slices.ContainsFunc(kws, func(k string) bool { return strings.HasPrefix(k, "MUST") }):
		return "requirement"
	case slices.ContainsFunc(kws, func(k string) bool { return strings.HasPrefix(k, "SHOULD") }):
		return "recommendation"
	case slices.Contains(kws, "MAY"):
		return "permission"
	}
	return ""
}

// specStatements is every statement of the documents, in identifier
// order, with every problem parsing them.
func specStatements(docs []*mdDoc) ([]statement, []string) {
	var all []statement
	var problems []string
	for _, d := range docs {
		s, p := d.statements()
		all = append(all, s...)
		problems = append(problems, p...)
	}
	slices.SortStableFunc(all, func(a, b statement) int { return statementLess(a.ID, b.ID) })
	return all, problems
}

// areaRegistry reads §1.8's table of areas from bcp14Home: each area's
// owning document.
func areaRegistry(docs []*mdDoc) (map[string]string, []string) {
	rowRe := regexp.MustCompile("^\\| *`([A-Z]{3})` *\\| *\\[[^\\]]*\\]\\(([^)#]+)\\) *\\|")
	areas := map[string]string{}
	var problems []string
	for _, d := range docs {
		if d.path != bcp14Home {
			continue
		}
		for i, l := range d.lines {
			m := rowRe.FindStringSubmatch(l)
			if m == nil || d.fenced[i] {
				continue
			}
			if _, dup := areas[m[1]]; dup {
				problems = append(problems, fmt.Sprintf("%s:%d: the area %s is registered twice", d.path, i+1, m[1]))
			}
			areas[m[1]] = path.Clean(path.Join(path.Dir(d.path), m[2]))
		}
	}
	if len(areas) == 0 {
		problems = append(problems, bcp14Home+": no area registry (ADR-011 D1 item 3)")
	}
	return areas, problems
}

// statusAreas reads a chapter's status line, "**Normative.** Areas: OUT."
// (ADR-011 D1 item 4), and returns the areas it names.
func (d *mdDoc) statusAreas() ([]string, bool) {
	re := regexp.MustCompile(`^\*\*(?:Normative|Informative)\.\*\* (?:Areas?: ([A-Z]{3}(?:, [A-Z]{3})*)\.|No statement area\.)$`)
	for _, b := range d.blocks {
		if b.kind == 'p' {
			if m := re.FindStringSubmatch(d.lines[b.start]); m != nil {
				if m[1] == "" {
					return nil, true
				}
				return strings.Split(m[1], ", "), true
			}
		}
	}
	return nil, false
}

// wellFormedProblems is TestStatementsWellFormed's check: every head
// matches the grammar with its anchor, no identifier is used twice, every
// area is registered and owned by the document it appears in, and every
// chapter's status line names the areas the registry gives it.
func wellFormedProblems(docs []*mdDoc) []string {
	stmts, problems := specStatements(docs)
	areas, p := areaRegistry(docs)
	problems = append(problems, p...)
	seen := map[string]statement{}
	for _, s := range stmts {
		if prev, dup := seen[s.ID]; dup {
			problems = append(problems, fmt.Sprintf("%s:%d: %s is already used at %s:%d", s.File, s.Line, s.ID, prev.File, prev.Line))
		}
		seen[s.ID] = s
		owner, ok := areas[s.area()]
		switch {
		case !ok:
			problems = append(problems, fmt.Sprintf("%s:%d: %s is in the area %s, which is not in the registry", s.File, s.Line, s.ID, s.area()))
		case owner != s.File:
			problems = append(problems, fmt.Sprintf("%s:%d: %s is in the area %s, which %s owns", s.File, s.Line, s.ID, s.area(), owner))
		}
		if s.Kind == "" {
			problems = append(problems, fmt.Sprintf("%s:%d: %s holds no key word, so it has no kind (ADR-011 D3 item 6)", s.File, s.Line, s.ID))
		}
	}
	chapterRe := regexp.MustCompile(`^(?:[0-9]{2}|[A-Z])-[a-z0-9-]+\.md$`)
	for _, d := range docs {
		var want []string
		for a, owner := range areas {
			if owner == d.path {
				want = append(want, a)
			}
		}
		if len(want) == 0 && !chapterRe.MatchString(d.path) {
			continue
		}
		slices.Sort(want)
		got, ok := d.statusAreas()
		if !ok {
			problems = append(problems, d.path+": no status line naming the chapter's areas (ADR-011 D1 item 4)")
			continue
		}
		slices.Sort(got)
		if !slices.Equal(got, want) {
			problems = append(problems, fmt.Sprintf("%s: the status line names the areas %q, and the registry gives it %q", d.path, got, want))
		}
	}
	return problems
}

// keywordProblems is TestKeywordsOnlyInStatements' check: no key word in
// capitals outside a statement, no synonym anywhere, and the boilerplate
// once, in its home and nowhere else.
func keywordProblems(docs []*mdDoc) []string {
	var problems []string
	for _, d := range docs {
		stmts, _ := d.statements()
		inside := map[int]bool{}
		for _, s := range stmts {
			for _, l := range s.lines {
				inside[l] = true
			}
		}
		exempt := map[int]bool{}
		bp := d.boilerplate()
		switch {
		case d.path == bcp14Home && len(bp) != 1:
			problems = append(problems, fmt.Sprintf("%s: holds RFC 8174's boilerplate %d times, not once (ADR-011 D2 item 1)", d.path, len(bp)))
		case d.path != bcp14Home && len(bp) > 0:
			problems = append(problems, fmt.Sprintf("%s: holds RFC 8174's boilerplate, which only %s carries", d.path, bcp14Home))
		}
		if d.path == bcp14Home && len(bp) > 0 {
			for _, l := range bp[0] {
				exempt[l] = true
			}
		}
		for i, l := range d.lines {
			if exempt[i] {
				continue
			}
			for _, kw := range synonymRe.FindAllString(l, -1) {
				problems = append(problems, fmt.Sprintf("%s:%d: %q is a synonym ADR-011 D2 item 2 excludes", d.path, i+1, kw))
			}
			if inside[i] {
				continue
			}
			for _, kw := range keywordRe.FindAllString(l, -1) {
				if !synonymRe.MatchString(kw) {
					problems = append(problems, fmt.Sprintf("%s:%d: the key word %q stands outside a statement (ADR-011 D2 item 4)", d.path, i+1, kw))
				}
			}
		}
	}
	return problems
}

// recommendationProblems is TestRecommendationsNotObservable's check.
func recommendationProblems(stmts []statement) []string {
	var problems []string
	for _, s := range stmts {
		if s.Observable && shouldRe.MatchString(s.Text) {
			problems = append(problems, fmt.Sprintf("%s:%d: %s says SHOULD and is not marked not observable (ADR-011 D3 item 5)", s.File, s.Line, s.ID))
		}
	}
	return problems
}

// mustDocs reads the specification's documents for a test.
func mustDocs(t *testing.T, root string) []*mdDoc {
	t.Helper()
	docs, err := readDocs(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(docs) == 0 {
		t.Fatalf("no document under %s", root)
	}
	return docs
}

func report(t *testing.T, problems []string) {
	t.Helper()
	for _, p := range problems {
		t.Error(p)
	}
}

// TestStatementsWellFormed is ADR-011 D14's tripwire over spec/.
func TestStatementsWellFormed(t *testing.T) {
	report(t, wellFormedProblems(mustDocs(t, specRoot)))
}

// TestKeywordsOnlyInStatements is ADR-011 D14's tripwire over spec/.
func TestKeywordsOnlyInStatements(t *testing.T) {
	report(t, keywordProblems(mustDocs(t, specRoot)))
}

// TestRecommendationsNotObservable is ADR-011 D14's tripwire over spec/.
func TestRecommendationsNotObservable(t *testing.T) {
	stmts, _ := specStatements(mustDocs(t, specRoot))
	report(t, recommendationProblems(stmts))
}
