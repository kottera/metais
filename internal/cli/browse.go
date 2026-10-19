// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"encoding/json"
	"errors"
	"slices"
	"strconv"
	"strings"

	"github.com/kottera/metais/internal/browse"
	"github.com/kottera/metais/internal/cli/args"
	"github.com/kottera/metais/internal/finding"
	"github.com/kottera/metais/internal/hostfs"
	"github.com/kottera/metais/internal/jsonx"
	"github.com/kottera/metais/internal/limits"
	"github.com/kottera/metais/internal/loader"
	"github.com/kottera/metais/internal/manifest"
	"github.com/kottera/metais/internal/regexprofile"
	"github.com/kottera/metais/internal/schema"
)

// The steps the reading commands' refusals point to. A placeholder in
// angle brackets is filled with what the refusal knows: the nearest
// directory that exists, an entry's path, a line count.
var (
	nextLsNearest = Next{Run: "metais ls <package> <path>", Why: "Lists the nearest directory above the path that exists."}
	nextLsDir     = Next{Run: "metais ls <package> <path>", Why: "Lists the entries beneath the directory."}
	nextExtract   = Next{Run: "metais extract <package> <path> --to <dir>", Why: "Writes the entry, as stored, to a file beneath a new directory."}
	nextReadLines = Next{Run: "metais read <package> <path> --lines 1-<total>", Why: "Reads the entry's lines, from its first to its last."}
	nextGlobs     = Next{Run: "metais help globs", Why: "Prints the glob grammar."}
	nextPattern   = Next{Run: "metais limits", Why: "Lists the limits a grep pattern is held to: pattern-length, regex-program and regex-work."}
	nextLsAll     = Next{Run: "metais ls <package> -r", Why: "Lists every entry, so a glob can be written against them."}
	nextExtractTo = Next{Run: "metais extract <package> <glob> --to <dir>", Why: "Writes into a directory that does not exist yet; extract never writes into an existing one."}
	nextInspect   = Next{Run: "metais inspect <package>", Why: "Lists the package's inputs, each with how to supply it."}
)

// readingCommand is the declaration every reading command shares: its
// group, role and the refusals of opening a package.
func readingCommand(c *Command, refusals ...Refusal) *Command {
	c.Group, c.Role = "Reading a package", finding.RoleConsumer
	c.Refusals = append(packageRefusals(c.Name), refusals...)
	c.Flags = append(c.Flags, limitFlag, failOnFlag)
	c.Operands = append([]args.Operand{packageOperand}, c.Operands...)
	return c
}

var packageOperand = args.Operand{Name: "<package>", Required: true,
	Description: "A packed package file, or a source directory; - is refused, since a package is read with random access."}

// The reading commands (ADR-008 D5).
func readingDeclarations() []*Command {
	pathRefusals := func(cmd string) []Refusal {
		return []Refusal{
			{finding.CodeUsageEntryNotFound, "The path names no entry and no directory of the package.", nextLsNearest},
			{finding.CodeUsageIsADirectory, "The path names a directory, and metais " + cmd + " reads one entry.", nextLsDir},
		}
	}
	textFact := "An entry is text when it is valid UTF-8 and holds no NUL byte, and binary otherwise. " +
		"A line ends at an LF; an entry has as many lines as LFs, plus one when it is not empty and does " +
		"not end with an LF."
	return []*Command{
		readingCommand(&Command{
			Name: "ls", Synopsis: "ls <package> [<path>] [--recursive | -r] [--limit <name>=<value>]... [--json]",
			Description: "List the entries under a path of a package.",
			Detail: "With no path, lists the root. A path that names a directory lists its children, or with " +
				"--recursive every file and directory beneath it; a path that names a file lists that file. " +
				"Directories are implied by the entries' paths. The listing is in the byte order of the " +
				"paths, a directory's written with its trailing /.\n\n" +
				"A packed package's entries are the zip's, .compiled/ included; a source directory's are the " +
				"files metais pack would write from it, without .compiled/. Paths use / on every platform and " +
				"are matched byte for byte, case-sensitively; the root is . or omitted.",
			Operands: []args.Operand{{Name: "<path>", Description: "An entry or a directory of the package, with or without a trailing /; the root when omitted."}},
			Flags:    []args.Flag{{Name: "recursive", Short: 'r', Description: "List every file and directory beneath the path, not only its children."}},
			Format:   "metais.ls/1",
			Prints:   "One line per entry: its type (file or dir), a file's size in bytes or the number of files beneath a directory, and its path, separated by tabs.",
			Examples: []Example{
				{"metais ls report-1.0.0.metais", "Lists the package's top level."},
				{"metais ls report-1.0.0.metais references/ -r", "Lists everything beneath references/."},
				{"metais ls report-1.0.0.metais --json", "Prints the listing as one metais.ls/1 document."},
			},
			run: runLs,
		}, Refusal{finding.CodeUsageEntryNotFound, "The path names no entry and no directory of the package.", nextLsNearest}),
		readingCommand(&Command{
			Name: "read", Synopsis: "read <package> <path> [--lines <range>] [--number] [--limit <name>=<value>]... [--json]",
			Description: "Print an entry of a package, or a range of its lines.",
			Detail: "Prints the entry's bytes exactly as stored. " + textFact + "\n\n" +
				"--lines takes N (one line), N-M or N- (to the end), 1-based and inclusive, with N at least 1 " +
				"and M at least N. A range that starts beyond the last line is refused; an end beyond it is " +
				"read to the last line. The lines are printed with their own line endings. --number prefixes " +
				"each line with its number and a tab. Both read lines of text, and refuse a binary entry; " +
				"a binary entry is not written to a terminal.\n\n" +
				"With --json, a text entry is the string content, with the lines read and the entry's line " +
				"count; a binary entry is content_base64.",
			Operands: []args.Operand{{Name: "<path>", Required: true, Description: "The entry to read."}},
			Flags: []args.Flag{
				{Name: "lines", Value: "<range>", Description: "Print only the lines N, N-M or N- of a text entry."},
				{Name: "number", Description: "Prefix each line printed with its line number and a tab."},
			},
			Format: "metais.read/1",
			Prints: "The entry's bytes, or the lines selected, unchanged; never coloured.",
			Examples: []Example{
				{"metais read report-1.0.0.metais inputs/product.md", "Prints what the input product means."},
				{"metais read report-1.0.0.metais references/wcag/1.1.1.md --lines 40-72", "Prints one section of a reference, as metais outline numbers it."},
				{"metais read report-1.0.0.metais .compiled/schema.json --json", "Prints the combined schema's entry as one metais.read/1 document."},
			},
			run: runRead,
		}, append(pathRefusals("read"),
			Refusal{finding.CodeUsageNotText, "--lines or --number is given for a binary entry.", nextExtract},
			Refusal{finding.CodeUsageLineOutOfRange, "The range of --lines starts beyond the entry's last line.", nextReadLines},
			Refusal{finding.CodeUsageBinaryToTerminal, "The entry is binary and standard output is a terminal.", nextExtract},
		)...),
		readingCommand(&Command{
			Name: "outline", Synopsis: "outline <package> <path> [--limit <name>=<value>]... [--json]",
			Description: "List the headings of a Markdown entry, each with the lines of its section.",
			Detail: "Reads a text entry as CommonMark 0.31.2 and lists its ATX and setext headings, levels 1 " +
				"to 6, in document order. A leading frontmatter block is skipped, and its lines still count. " +
				"What CommonMark does not make a heading, such as a # line in a code block, is not listed.\n\n" +
				"Each heading is shown with its first line and the last line of its section: the line before " +
				"the next heading of the same or a higher level, or the entry's last line. So metais read " +
				"<package> <path> --lines <line>-<end> reads exactly one section. An entry with no headings " +
				"gives an empty list.",
			Operands: []args.Operand{{Name: "<path>", Required: true, Description: "The text entry to outline."}},
			Format:   "metais.outline/1",
			Prints:   "One line per heading: its line and its section's last line, a tab, two spaces per level below 1, the heading's #s and its text.",
			Examples: []Example{
				{"metais outline report-1.0.0.metais references/wcag/1.1.1.md", "Lists the reference's sections."},
				{"metais outline report-1.0.0.metais inputs/product.md --json", "Prints the headings as one metais.outline/1 document."},
			},
			run: runOutline,
		}, append(pathRefusals("outline"),
			Refusal{finding.CodeUsageNotText, "The entry is binary.", nextExtract},
		)...),
		readingCommand(&Command{
			Name:        "grep",
			Synopsis:    "grep <package> <pattern> [<glob>]... [--context <n> | -C <n>] [--ignore-case | -i] [--max-results <n>] [--limit <name>=<value>]... [--json]",
			Description: "Search a package's entries for a regular expression, with context.",
			Detail: "The pattern is RE2 syntax, as Go's regexp defines it, and is matched against each line " +
				"without its line ending, so ^ and $ anchor the line; a CR before the LF is part of the " +
				"ending. --ignore-case prepends (?i). The pattern is bounded by the pattern-length, " +
				"regex-program and regex-work limits.\n\n" +
				"Globs select the entries searched (metais help globs); with none, every entry is searched " +
				"but those beneath a segment that begins with a dot, so .compiled/ is searched only when a " +
				"glob names it. Entries are searched in path order, lines in order. Binary entries are " +
				"skipped and listed.\n\n" +
				"--max-results bounds the matching lines reported, from 1 to the grep-matches limit; the " +
				"search still runs to the end, and counts every matching line. A line longer than 1024 bytes " +
				"is shown cut, and read --lines gives it whole. No match is not a failure: grep exits 0.",
			Operands: []args.Operand{
				{Name: "<pattern>", Required: true, Description: "A regular expression in RE2 syntax; write -- before one that begins with -."},
				{Name: "<glob>", Repeatable: true, Description: "A glob selecting the entries to search; every entry when none is given."},
			},
			Flags: []args.Flag{
				{Name: "context", Short: 'C', Value: "<n>", Default: "0", Description: "Show n lines before and after each matching line, from 0 to the grep-context limit."},
				{Name: "ignore-case", Short: 'i', Description: "Match letters regardless of case."},
				{Name: "max-results", Value: "<n>", Default: "100", Description: "Report at most n matching lines, from 1 to the grep-matches limit."},
			},
			Format: "metais.grep/1",
			Prints: "Each matching line as path:line:text and each line of context as path-line-text, with -- between groups that are not adjacent; a truncation notice and the binary entries skipped go to standard error.",
			Examples: []Example{
				{"metais grep report-1.0.0.metais 'alt text' references/", "Searches the references."},
				{"metais grep report-1.0.0.metais -i -C 2 '^## ' '**/*.md'", "Lists every level-2 heading of the Markdown entries, with two lines around each."},
				{"metais grep report-1.0.0.metais schema .compiled/** --json", "Searches the compiled files, as one metais.grep/1 document."},
			},
			run: runGrep,
		},
			Refusal{finding.CodeUsageBadGlob, "A glob does not satisfy the glob grammar.", nextGlobs},
			Refusal{finding.CodeUsageBadPattern, "The pattern is not RE2 syntax, or breaches pattern-length, regex-program or regex-work.", nextPattern},
		),
		readingCommand(&Command{
			Name: "schema", Synopsis: "schema <package> [<input>] [--limit <name>=<value>]... [--json]",
			Description: "Print the package's combined input schema, or one input's schema.",
			Detail: "With no input, prints the combined input schema: one JSON Schema 2020-12 document with a " +
				"property per input, which is what the inputs are validated against. With an input, prints " +
				"that input's schema as one self-contained document: the input's title and description beside " +
				"a $ref to its schema, with $defs embedding every schema resource the $ref reaches. A " +
				"collection's schema is the array's, whose items each file must satisfy.\n\n" +
				"The schema is printed in canonical key order, indented by two spaces, so " +
				"metais schema <package> <input> > <input>.json writes a usable schema file.",
			Operands: []args.Operand{{Name: "<input>", Description: "An input's name; the combined schema when omitted."}},
			Format:   "metais.schema/1",
			Prints:   "The schema alone, indented by two spaces; never coloured.",
			Examples: []Example{
				{"metais schema report-1.0.0.metais", "Prints the combined input schema."},
				{"metais schema report-1.0.0.metais product", "Prints the schema of the input product."},
				{"metais schema report-1.0.0.metais product --json", "Prints it as one metais.schema/1 document."},
			},
			run: runSchema,
		}, Refusal{finding.CodeUsageUnknownInput, "The operand names no input of the package; the finding lists them.", nextInspect}),
		readingCommand(&Command{
			Name: "extract", Synopsis: "extract <package> <glob>... --to <dir> [--dry-run] [--limit <name>=<value>]... [--json]",
			Description: "Write the entries the globs select to a new directory.",
			Detail: "Writes every entry any glob selects to <dir>/<entry path>, as stored (metais help globs). " +
				"Every glob must select at least one entry, or nothing is written: a glob that selects nothing " +
				"is a typo to hear about.\n\n" +
				"<dir> must not exist: it is created, with its missing parents, and every file is created " +
				"beneath it, never through a link. An existing <dir> is refused, empty or not, and nothing in " +
				"it is touched. A failed write stops extract, which lists the files it wrote and removes none " +
				"of them. --dry-run lists the files it would write, and writes nothing.",
			Operands: []args.Operand{{Name: "<glob>", Required: true, Repeatable: true, Description: "A glob selecting entries to write."}},
			Flags: []args.Flag{
				{Name: "to", Value: "<dir>", Description: "The directory to write into, which must not exist; required."},
				{Name: "dry-run", Description: "List the files extract would write, and write nothing."},
			},
			Format: "metais.extract/1",
			Prints: "The files written, or that would be, each with its size, relative to <dir>.",
			Examples: []Example{
				{"metais extract report-1.0.0.metais 'examples/basic/**' --to basic", "Writes the example basic into the new directory basic."},
				{"metais extract report-1.0.0.metais references --to refs --dry-run", "Lists the references it would write."},
			},
			run: runExtract,
		},
			Refusal{finding.CodeUsageBadGlob, "A glob does not satisfy the glob grammar.", nextGlobs},
			Refusal{finding.CodeUsageNoMatch, "A glob selects no entry; nothing is written.", nextLsAll},
			Refusal{finding.CodeDestinationNotEmpty, "<dir> exists; extract writes only into a directory it creates.", nextExtractTo},
			Refusal{finding.CodeDestinationNotDirectory, "<dir>, or a directory above it, exists and is not a directory.", nextExtractTo},
			Refusal{finding.CodeDestinationWriteFailed, "A write failed; the files already written are listed and left in place.", nextExtractTo},
		),
	}
}

// fill returns ns with each placeholder of with replaced by its value.
func fill(ns []Next, with map[string]string) []Next {
	out := slices.Clone(ns)
	for i := range out {
		for k, v := range with {
			out[i].Run = strings.ReplaceAll(out[i].Run, k, v)
		}
	}
	return out
}

// refuseFilled is the refusal f, its next steps' placeholders filled.
func (s *session) refuseFilled(f finding.Finding, with map[string]string) outcome {
	o := s.refuseWith(f)
	o.next = fill(o.next, with)
	return o
}

// entryPath is an entry path as a next step writes it: the root as ".",
// and a directory with its trailing /.
func dirArg(d string) string {
	if d == "." {
		return "."
	}
	return shellQuote(d)
}

// lookupRefusal is the refusal of the path operand i, which browse.Lookup
// or browse.List refused with err.
func (s *session) lookupRefusal(p args.Parsed, i int, err error) outcome {
	var nf *browse.NotFoundError
	if errors.As(err, &nf) {
		f := finding.New(finding.CodeUsageEntryNotFound, invocation, nil, finding.Params{"argument": s.operandArg(p, i)})
		return s.refuseFilled(f, map[string]string{"<path>": dirArg(nf.Nearest)})
	}
	if errors.Is(err, browse.ErrIsDirectory) {
		f := finding.New(finding.CodeUsageIsADirectory, invocation, nil, finding.Params{
			"argument": s.operandArg(p, i), "command": finding.String(s.line.cmd.Name),
		})
		return s.refuseFilled(f, map[string]string{"<path>": dirArg(browse.Clean(p.Operands[i]) + "/")})
	}
	panic("cli: a lookup refused with " + err.Error())
}

// lsResult is metais.ls/1's result (ADR-008 D5, ls item 2).
type lsResult struct {
	Entries []lsEntry `json:"entries"`
}

type lsEntry struct {
	Path  string `json:"path"`
	Type  string `json:"type"`
	Size  *int64 `json:"size,omitempty"`
	Files *int   `json:"files,omitempty"`
}

// runLs is metais ls.
func runLs(s *session, p args.Parsed) outcome {
	pkg, refused := s.open(p)
	if pkg == nil {
		return refused
	}
	defer pkg.Close()
	path := "."
	if len(p.Operands) > 1 {
		path = p.Operands[1]
	}
	es, err := browse.List(pkg.Tree, path, p.Has("recursive"))
	if err != nil {
		return s.lookupRefusal(p, 1, err)
	}
	res := &lsResult{Entries: make([]lsEntry, 0, len(es))}
	var b strings.Builder
	for _, e := range es {
		le := lsEntry{Path: e.Path, Type: "file"}
		n := strconv.FormatInt(e.Size, 10)
		if e.Dir {
			files := e.Files
			le.Type, le.Files, n = "dir", &files, strconv.Itoa(files)
		} else {
			size := e.Size
			le.Size = &size
		}
		res.Entries = append(res.Entries, le)
		b.WriteString(le.Type + "\t" + n + "\t" + browse.Neutralise(e.Path) + "\n")
	}
	return outcome{result: res, human: []byte(b.String())}
}

// readResult is metais.read/1's result (ADR-008 D5, read item 5).
type readResult struct {
	Path          string     `json:"path"`
	Size          int64      `json:"size"`
	Binary        bool       `json:"binary"`
	Content       *string    `json:"content,omitempty"`
	Lines         *readLines `json:"lines,omitempty"`
	ContentBase64 *[]byte    `json:"content_base64,omitempty"`
}

type readLines struct {
	First int `json:"first"`
	Last  int `json:"last"`
	Total int `json:"total"`
}

// entry reads the entry the path operand i names, or returns the refusal.
func (s *session) entry(pkg *loader.Package, p args.Parsed, i int) (string, []byte, *outcome) {
	info, err := browse.Lookup(pkg.Tree, p.Operands[i])
	if err != nil {
		o := s.lookupRefusal(p, i, err)
		return "", nil, &o
	}
	b, err := browse.Content(pkg.Tree, info.Path, s.inv.Options.Faults)
	if err != nil {
		// The package was verified whole when it opened, so an entry
		// that cannot be read again is a file that changed beneath the
		// tool, which nothing it reports would describe.
		panic("cli: reading a verified entry: " + err.Error())
	}
	return info.Path, b, nil
}

// notText is the refusal of a binary entry, named by the path operand i,
// read as lines.
func (s *session) notText(p args.Parsed, i int, path string) outcome {
	f := finding.New(finding.CodeUsageNotText, invocation, nil, finding.Params{
		"argument": s.operandArg(p, i), "command": finding.String(s.line.cmd.Name),
	})
	return s.refuseFilled(f, map[string]string{"<path>": shellQuote(path)})
}

// runRead is metais read.
func runRead(s *session, p args.Parsed) outcome {
	var rng *browse.Range
	if v, ok := p.Value("lines"); ok {
		r, ok := browse.ParseRange(v)
		if !ok {
			return s.refuseWith(finding.New(finding.CodeUsageBadValue, invocation, nil, finding.Params{
				"argument": s.flagArg(p, "lines"), "flag": finding.String("--lines"), "reason": finding.String("value_not_allowed"),
			}))
		}
		rng = &r
	}
	pkg, refused := s.open(p)
	if pkg == nil {
		return refused
	}
	defer pkg.Close()
	path, b, refusal := s.entry(pkg, p, 1)
	if refusal != nil {
		return *refusal
	}
	res := &readResult{Path: path, Size: int64(len(b))}
	if !browse.IsText(b) {
		if rng != nil || p.Has("number") {
			return s.notText(p, 1, path)
		}
		if !s.line.json && isTerminal(s.inv.Stdout) {
			f := finding.New(finding.CodeUsageBinaryToTerminal, invocation, nil, finding.Params{"argument": s.operandArg(p, 1)})
			return s.refuseFilled(f, map[string]string{"<path>": shellQuote(path)})
		}
		res.Binary, res.ContentBase64 = true, &b
		return outcome{result: res, human: b}
	}
	sel, err := browse.SelectLines(b, rng)
	if err != nil {
		f := finding.New(finding.CodeUsageLineOutOfRange, invocation, nil, finding.Params{
			"argument": s.flagArg(p, "lines"), "total": finding.Int(int64(sel.Total)),
		})
		o := s.refuseFilled(f, map[string]string{"<path>": shellQuote(path), "<total>": strconv.Itoa(sel.Total)})
		if sel.Total == 0 {
			o.next = []Next{{Run: "metais read <package> " + shellQuote(path), Why: "Reads the entry, which is empty and has no lines."}}
		}
		return o
	}
	content := string(sel.Bytes)
	res.Content, res.Lines = &content, &readLines{sel.First, sel.Last, sel.Total}
	human := sel.Bytes
	if p.Has("number") {
		var nb []byte
		for _, l := range sel.Lines {
			nb = append(append(strconv.AppendInt(nb, int64(l.Number), 10), '\t'), l.Bytes...)
		}
		human = nb
	}
	return outcome{result: res, human: human}
}

// outlineResult is metais.outline/1's result (ADR-008 D5, outline item 2).
type outlineResult struct {
	Path     string           `json:"path"`
	Headings []outlineHeading `json:"headings"`
}

type outlineHeading struct {
	Level   int    `json:"level"`
	Text    string `json:"text"`
	Line    int    `json:"line"`
	EndLine int    `json:"end_line"`
}

// runOutline is metais outline.
func runOutline(s *session, p args.Parsed) outcome {
	pkg, refused := s.open(p)
	if pkg == nil {
		return refused
	}
	defer pkg.Close()
	path, b, refusal := s.entry(pkg, p, 1)
	if refusal != nil {
		return *refusal
	}
	if !browse.IsText(b) {
		return s.notText(p, 1, path)
	}
	res := &outlineResult{Path: path, Headings: []outlineHeading{}}
	var h strings.Builder
	for _, hd := range browse.Outline(b) {
		res.Headings = append(res.Headings, outlineHeading{hd.Level, hd.Text, hd.Line, hd.EndLine})
		h.WriteString(strconv.Itoa(hd.Line) + "-" + strconv.Itoa(hd.EndLine) + "\t" +
			strings.Repeat("  ", hd.Level-1) + strings.Repeat("#", hd.Level) + " " + browse.Neutralise(hd.Text) + "\n")
	}
	o := outcome{result: res, human: []byte(h.String())}
	if len(res.Headings) == 0 {
		o.notice = []byte("The entry has no headings.\n")
	}
	return o
}

// grepResult is metais.grep/1's result (ADR-008 D5, grep item 7).
type grepResult struct {
	Pattern       string      `json:"pattern"`
	Globs         []string    `json:"globs"`
	Matches       []grepMatch `json:"matches"`
	MatchingLines int         `json:"matching_lines"`
	Truncated     bool        `json:"truncated"`
	Skipped       []string    `json:"skipped"`
}

type grepMatch struct {
	Path          string     `json:"path"`
	Line          int        `json:"line"`
	Text          string     `json:"text"`
	TextTruncated bool       `json:"text_truncated"`
	Before        []grepLine `json:"before"`
	After         []grepLine `json:"after"`
}

type grepLine struct {
	Line          int    `json:"line"`
	Text          string `json:"text"`
	TextTruncated bool   `json:"text_truncated"`
}

func grepLines(ss []browse.Shown) []grepLine {
	out := make([]grepLine, len(ss))
	for i, l := range ss {
		out[i] = grepLine{l.Line, l.Text, l.Omitted > 0}
	}
	return out
}

// count reads the value of the flag called name, a decimal integer from
// lo to hi, or def when it is not given; ok is false when it is refused.
func (s *session) count(p args.Parsed, name string, def, lo, hi int64) (int64, *outcome) {
	v, given := p.Value(name)
	if !given {
		return def, nil
	}
	n, err := strconv.ParseInt(v, 10, 64)
	if err != nil || strings.TrimLeft(v, "0123456789") != "" || n < lo || n > hi {
		o := s.refuseWith(finding.New(finding.CodeUsageBadValue, invocation, nil, finding.Params{
			"argument": s.flagArg(p, name), "flag": finding.String("--" + name), "reason": finding.String("value_not_allowed"),
		}))
		return 0, &o
	}
	return n, nil
}

// globs compiles the glob operands from first on, or returns the refusal
// of the first that does not satisfy the grammar.
func (s *session) globs(p args.Parsed, first int) ([]*browse.Glob, *outcome) {
	var out []*browse.Glob
	for i := first; i < len(p.Operands); i++ {
		g, err := browse.CompileGlob(p.Operands[i])
		var ge *browse.GlobError
		if errors.As(err, &ge) {
			o := s.refuseWith(finding.New(finding.CodeUsageBadGlob, invocation, nil, finding.Params{
				"argument": s.operandArg(p, i), "reason": finding.String(ge.Reason),
			}))
			return nil, &o
		}
		out = append(out, g)
	}
	return out, nil
}

// runGrep is metais grep.
func runGrep(s *session, p args.Parsed) outcome {
	lim, fs := s.limits(p)
	if len(fs) > 0 {
		return outcome{findings: fs, next: s.nextSteps(fs)}
	}
	context, refused := s.count(p, "context", 0, 0, lim.Value(limits.GrepContext))
	if refused != nil {
		return *refused
	}
	maxResults, refused := s.count(p, "max-results", 100, 1, lim.Value(limits.GrepMatches))
	if refused != nil {
		return *refused
	}
	globs, refused := s.globs(p, 2)
	if refused != nil {
		return *refused
	}
	meter := regexprofile.NewMeter(lim.Value(limits.RegexWork))
	re, err := regexprofile.CompileRE2(p.Operands[1], p.Has("ignore-case"), lim, meter)
	if err != nil {
		return s.badPattern(p, err, lim)
	}
	pkg, out := s.open(p)
	if pkg == nil {
		return out
	}
	defer pkg.Close()
	if len(globs) == 0 {
		all, _ := browse.CompileGlob("**")
		globs = append(globs, all)
	}
	var entries []string
	for _, e := range pkg.Tree.List() {
		if slices.ContainsFunc(globs, func(g *browse.Glob) bool { return g.Match(e) }) {
			entries = append(entries, e)
		}
	}
	found, err := browse.Grep(pkg.Tree, entries, re, meter,
		browse.GrepOptions{Context: int(context), MaxResults: int(maxResults)}, s.inv.Options.Faults)
	if err != nil {
		return s.badPattern(p, err, lim)
	}
	res := &grepResult{Pattern: p.Operands[1], Globs: slices.Clone(p.Operands[2:]), Matches: []grepMatch{},
		MatchingLines: found.MatchingLines, Truncated: found.Truncated, Skipped: found.Skipped}
	if res.Globs == nil {
		res.Globs = []string{}
	}
	for _, m := range found.Matches {
		res.Matches = append(res.Matches, grepMatch{m.Path, m.Line, m.Text, m.Omitted > 0, grepLines(m.Before), grepLines(m.After)})
	}
	o := outcome{result: res, human: grepHuman(found)}
	if found.Truncated {
		o.notice = append(o.notice, ("Showing the first " + strconv.Itoa(len(found.Matches)) + " of " +
			strconv.Itoa(found.MatchingLines) + " matching lines; " + strconv.Itoa(found.MatchingLines-len(found.Matches)) +
			" more were found and not shown (--max-results).\n")...)
	}
	if len(found.Skipped) > 0 {
		o.notice = append(o.notice, "Binary entries skipped:\n"...)
		for _, e := range found.Skipped {
			o.notice = append(o.notice, "  "+browse.Neutralise(e)+"\n"...)
		}
	}
	return o
}

// badPattern is the refusal of grep's pattern for err, from
// regexprofile.CompileRE2 or browse.Grep.
func (s *session) badPattern(p args.Parsed, err error, lim limits.Set) outcome {
	params := finding.Params{"argument": s.operandArg(p, 1), "reason": finding.String("syntax")}
	var le *regexprofile.LimitError
	var id limits.ID
	switch {
	case errors.As(err, &le):
		id = le.Limit
	case errors.Is(err, browse.ErrRegexWork):
		id = limits.RegexWork
	case errors.Is(err, regexprofile.ErrRE2Syntax):
	default:
		panic("cli: a grep refused with " + err.Error())
	}
	if id != 0 || errors.Is(err, browse.ErrRegexWork) {
		name := limits.Get(id).Name
		params["reason"] = finding.String(strings.ReplaceAll(name, "-", "_"))
		params["limit"] = finding.String(name)
		params["bound"] = finding.Int(lim.Value(id))
	}
	return s.refuseWith(finding.New(finding.CodeUsageBadPattern, invocation, nil, params))
}

// grepHuman is grep's human result (ADR-008 D5, grep item 8), in GNU
// grep's form: path:line:text for a matching line reported, path-line-text
// for a line of context, and -- between groups that are not adjacent;
// groups that touch or overlap are merged.
func grepHuman(r browse.GrepResult) []byte {
	type shown struct {
		path  string
		line  browse.Shown
		match bool
	}
	var rows []shown
	at := map[string]int{} // path and line → index in rows
	key := func(path string, line int) string { return path + "\x00" + strconv.Itoa(line) }
	add := func(path string, l browse.Shown, match bool) {
		if i, ok := at[key(path, l.Line)]; ok {
			rows[i].match = rows[i].match || match
			return
		}
		at[key(path, l.Line)] = len(rows)
		rows = append(rows, shown{path, l, match})
	}
	for _, m := range r.Matches {
		for _, l := range m.Before {
			add(m.Path, l, false)
		}
		add(m.Path, m.Shown, true)
		for _, l := range m.After {
			add(m.Path, l, false)
		}
	}
	slices.SortStableFunc(rows, func(a, b shown) int {
		if c := strings.Compare(a.path, b.path); c != 0 {
			return c
		}
		return a.line.Line - b.line.Line
	})
	var b strings.Builder
	for i, row := range rows {
		if i > 0 && (rows[i-1].path != row.path || rows[i-1].line.Line+1 != row.line.Line) {
			b.WriteString("--\n")
		}
		sep := "-"
		if row.match {
			sep = ":"
		}
		b.WriteString(browse.Neutralise(row.path) + sep + strconv.Itoa(row.line.Line) + sep + browse.Neutralise(row.line.Text))
		if row.line.Omitted > 0 {
			b.WriteString("[…+" + strconv.Itoa(row.line.Omitted) + " bytes]")
		}
		b.WriteString("\n")
	}
	return []byte(b.String())
}

// schemaResult is metais.schema/1's result (ADR-008 D5, schema item 3):
// the schema's data, in canonical key order.
type schemaResult struct {
	Input  *string         `json:"input"`
	Schema json.RawMessage `json:"schema"`
}

// runSchema is metais schema.
func runSchema(s *session, p args.Parsed) outcome {
	pkg, refused := s.open(p)
	if pkg == nil {
		return refused
	}
	defer pkg.Close()
	doc := pkg.CombinedValue
	res := &schemaResult{}
	if len(p.Operands) > 1 {
		name := p.Operands[1]
		i := slices.IndexFunc(pkg.Manifest.Inputs, func(in manifest.Input) bool { return in.Name == name })
		if i < 0 {
			names := make([]string, 0, len(pkg.Manifest.Inputs))
			for _, in := range pkg.Manifest.Inputs {
				names = append(names, in.Name)
			}
			o := s.refuseWith(finding.New(finding.CodeUsageUnknownInput, invocation, nil, finding.Params{
				"argument": s.operandArg(p, 1), "inputs": stringList(names),
			}))
			o.human = listing("The package's inputs:", names, false)
			return o
		}
		in := pkg.Manifest.Inputs[i]
		res.Input = &in.Name
		doc = schema.InputDocument(doc, in.Name)
	}
	res.Schema = json.RawMessage(jsonx.Canonical(doc))
	human, err := jsonx.Encode(doc, 2)
	if err != nil {
		panic("cli: encoding a schema: " + err.Error())
	}
	next := slices.DeleteFunc([]Next{
		{Run: "metais validate <package> --data <dir>", Why: "Validates the inputs you gathered in a data directory against this schema."},
	}, func(n Next) bool { return lookup(s.table, "validate") == nil })
	return outcome{result: res, human: append(human, '\n'), next: next}
}

// extractResult is metais.extract/1's result (ADR-008 D5, extract item 6).
// The destination is not echoed (D11 item 5).
type extractResult struct {
	DryRun bool          `json:"dry_run"`
	Files  []extractFile `json:"files"`
}

type extractFile struct {
	Path string `json:"path"`
	Size int64  `json:"size"`
}

// runExtract is metais extract.
func runExtract(s *session, p args.Parsed) outcome {
	to, ok := p.Value("to")
	if !ok {
		return s.refuseWith(finding.New(finding.CodeUsageMissingOperand, invocation, nil, finding.Params{
			"command": finding.String("extract"), "operand": finding.String("--to <dir>"),
		}))
	}
	globs, refused := s.globs(p, 1)
	if refused != nil {
		return *refused
	}
	pkg, out := s.open(p)
	if pkg == nil {
		return out
	}
	defer pkg.Close()
	all := pkg.Tree.List()
	var entries []string
	var fs []finding.Finding
	for i, g := range globs {
		sel := g.Select(all)
		if len(sel) == 0 {
			fs = append(fs, finding.New(finding.CodeUsageNoMatch, invocation, nil, finding.Params{"argument": s.operandArg(p, i+1)}))
		}
		entries = append(entries, sel...)
	}
	if len(fs) > 0 {
		return outcome{findings: fs, next: s.nextSteps(fs)}
	}
	slices.Sort(entries)
	entries = slices.Compact(entries)
	dry := p.Has("dry-run")
	files, err := browse.Extract(pkg.Tree, entries, to, browse.ExtractOptions{DryRun: dry}, s.inv.Options.Faults)
	if err != nil {
		return s.extractFailed(err)
	}
	res := &extractResult{DryRun: dry, Files: make([]extractFile, len(files))}
	var names []string
	for i, f := range files {
		res.Files[i] = extractFile{f.Path, f.Size}
		names = append(names, f.Path)
	}
	head := "Wrote " + plural(len(files), "file", "files") + ":"
	if dry {
		head = "Would write " + plural(len(files), "file", "files") + "; --dry-run wrote nothing:"
	}
	return outcome{result: res, human: listing(head, names, false)[1:]}
}

// extractFailed is the outcome of err from browse.Extract.
func (s *session) extractFailed(err error) outcome {
	var de *browse.DestinationError
	var we *browse.WriteError
	switch {
	case errors.As(err, &de) && de.NotDirectory:
		return s.refuseWith(finding.New(finding.CodeDestinationNotDirectory, invocation, nil, nil))
	case errors.As(err, &de):
		o := s.refuseWith(finding.New(finding.CodeDestinationNotEmpty, invocation, nil, finding.Params{
			"entries": stringList(de.Entries), "more": finding.Bool(de.More),
		}))
		if len(de.Entries) > 0 || de.More {
			o.human = listing("Found in the directory:", de.Entries, de.More)
		}
		return o
	case errors.As(err, &we):
		written := we.Written
		if written == nil {
			written = []string{}
		}
		o := s.refuseWith(finding.New(finding.CodeDestinationWriteFailed, invocation, nil, finding.Params{
			"reason":  finding.String(writeReason(hostfs.Reason(we.Reason))),
			"written": stringList(written),
		}))
		if len(written) > 0 {
			o.human = listing("Written before the failure, and left in place:", written, false)
		}
		return o
	}
	panic("cli: extract failed with " + err.Error())
}
