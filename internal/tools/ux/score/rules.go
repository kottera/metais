// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package score

import (
	"fmt"
	"path"
	"regexp"
	"slices"
	"strings"

	"github.com/kottera/metais/internal/tools/ux/streamjson"
)

// contentReaders are the metais commands that read a package's contents.
var contentReaders = []string{"ls", "read", "schema", "outline", "grep", "extract", "unpack"}

// inspectFirst: a consuming session reads nothing of the package's
// contents before metais inspect (Ed's ruling, 2026-10-21). metais
// --version, metais version and metais help before it are no read and
// pass. What fails is extracting the package, or metais ls, read, schema,
// outline, grep, extract or unpack on it, before metais inspect, and a
// session that never runs metais inspect. Reading files extracted from
// the package needs the package extracted first, so it fails as that
// extraction does. A listing of the archive (unzip -l) is a note.
func inspectFirst(r *run) Result {
	var inspect *call
	for _, c := range r.calls {
		if c.sub == "inspect" {
			inspect = c
			break
		}
	}
	before := func(seq int) bool { return inspect == nil || seq < inspect.seq }
	var ev []Evidence
	for _, c := range r.calls {
		if before(c.seq) && slices.Contains(contentReaders, c.sub) {
			e := c.evidence()
			e.Text = "metais " + c.sub + " reads the package's contents"
			ev = append(ev, e)
		}
	}
	for _, z := range r.zipReads() {
		if !z.listing && before(z.cmd.seq) {
			ev = append(ev, z.ev)
		}
	}
	slices.SortStableFunc(ev, func(a, b Evidence) int { return lineOrder(a) - lineOrder(b) })
	switch {
	case inspect == nil && len(ev) > 0:
		return Result{Detail: "the session never runs metais inspect, and " + ev[0].Text, Evidence: ev}
	case inspect == nil:
		return Result{Detail: "the session never runs metais inspect"}
	case len(ev) > 0:
		return Result{Detail: "before metais inspect, " + ev[0].Text, Evidence: append(ev, inspect.evidence())}
	}
	return Result{Pass: true, Detail: "the session reads nothing of the package before metais inspect", Evidence: []Evidence{inspect.evidence()}}
}

func orNone(sub string) string {
	if sub == "" {
		return "with no command"
	}
	return sub
}

// zipTools are the commands that extract a zip archive's entries, and
// zipListers those that only list them (Ed's ruling, 2026-10-21: a
// listing is a note, not a failure).
var (
	zipTools   = []string{"unzip", "funzip", "7z", "7za", "7zr", "bsdtar", "ditto", "jar", "unar"}
	zipListers = []string{"zipinfo", "lsar", "zipdetails", "zipnote"}
)

// zipLibrary matches a program that parses the archive itself, through a
// zip library of its language, and zipLibraryRead one that reads or
// extracts an entry with it, not only lists them.
var (
	zipLibrary     = regexp.MustCompile(`(?i)\bzipfile\b|archive/zip|\bjszip\b|adm-zip|\byauzl\b|\bunzipper\b|java\.util\.zip|ZipArchive|Expand-Archive|System\.IO\.Compression|\bRubyzip\b|Zip::File`)
	zipLibraryRead = regexp.MustCompile(`(?i)\.read\(|\.open\(|extract|-m\s+zipfile\s+-e|\.readFile|getEntry|getInputStream|Expand-Archive|\bunzipSync\b|\.async\(`)
)

// zipRead is one command that reads a zip archive: an extraction, or only
// a listing of its entries.
type zipRead struct {
	cmd     *cmd
	ev      Evidence
	listing bool
}

// listingFlags reports whether an unzip, 7z or tar command line only
// lists the archive: unzip -l, -v, -Z or -t, 7z l, tar -t.
func listingFlags(c *cmd) bool {
	for _, w := range c.rest {
		switch {
		case c.name == "unzip" && strings.HasPrefix(w, "-") && strings.ContainsAny(w, "lvZt"):
			return true
		case strings.HasPrefix(c.name, "7z") && (w == "l" || w == "t"):
			return true
		case (c.name == "tar" || c.name == "bsdtar") && strings.HasPrefix(w, "-") && strings.Contains(w, "t") && !strings.Contains(w, "x"):
			return true
		}
	}
	return false
}

// zipReads are the session's commands that extract, parse or list a zip
// archive: a zip tool, tar given a package, a command line that uses a
// zip library, or a program the session wrote with one that it then ran,
// unless what it reads is an archive the session rendered (ownRender).
// In a consuming scenario other than C0, metais unpack and metais extract
// are extractions too (Ed's ruling, 2026-10-21). In C0 they are the edit
// reading the amended R-UX-8 names as coherent (the plan orchestrator's
// ruling), and in an authoring scenario they are how a package is edited.
func (r *run) zipReads() []zipRead {
	var out []zipRead
	seen := map[*cmd]bool{}
	// libraryCalls are the Bash calls a zip library was read from: it is
	// read from a call's whole command line, so once for the call, not
	// once for each of its commands.
	libraryCalls := map[*streamjson.ToolUse]bool{}
	add := func(c *cmd, why string, listing bool) {
		if seen[c] {
			return
		}
		seen[c] = true
		e := c.evidence()
		e.Text = why
		out = append(out, zipRead{c, e, listing})
	}
	scripts := map[string]string{}
	reads := map[string]bool{}
	texts := map[string]string{}
	for _, w := range r.writes {
		if m := zipLibrary.FindString(w.text); m != "" && w.file != "" {
			scripts[path.Base(w.file)] = m
			texts[path.Base(w.file)] = w.text
			reads[path.Base(w.file)] = zipLibraryRead.MatchString(w.text)
		}
	}
	metaisExtracts := r.scenario.Kind == "consuming" && r.scenario.ID != "C0"
	for _, c := range r.cmds {
		switch {
		case c.name == "metais":
			if sub, _ := subcommand(c.rest); metaisExtracts && (sub == "unpack" || sub == "extract") {
				add(c, "metais "+sub+" extracts the package", false)
			}
		case (slices.Contains(zipListers, c.name) || slices.Contains(zipTools, c.name) || c.name == "tar") && r.ownRender(c.rest, c.cwd, c.seq):
			// An archive the session rendered, read back: not the package.
		case slices.Contains(zipListers, c.name):
			add(c, c.name+" lists the archive", true)
		case slices.Contains(zipTools, c.name):
			if listingFlags(c) {
				add(c, c.name+" lists the archive", true)
			} else {
				add(c, c.name+" extracts the archive", false)
			}
		case c.name == "tar" && slices.ContainsFunc(c.rest, func(w string) bool { return strings.HasSuffix(w, ".metais") }):
			if listingFlags(c) {
				add(c, "tar lists the archive", true)
			} else {
				add(c, "tar extracts the archive", false)
			}
		default:
			if m := zipLibrary.FindString(c.use.Command()); m != "" {
				if libraryCalls[c.use] || r.ownRender(pathWord.FindAllString(c.use.Command(), -1), c.cwd, c.seq) {
					continue
				}
				libraryCalls[c.use] = true
				if zipLibraryRead.MatchString(c.use.Command()) {
					add(c, "the command parses the archive with "+m, false)
				} else {
					add(c, "the command lists the archive with "+m, true)
				}
				continue
			}
			for _, w := range append([]string{c.name}, c.rest...) {
				if m, ok := scripts[path.Base(w)]; ok {
					if r.ownRender(pathWord.FindAllString(texts[path.Base(w)], -1), c.cwd, c.seq) {
						break
					}
					if reads[path.Base(w)] {
						add(c, fmt.Sprintf("it runs %s, which the session wrote to parse the archive with %s", path.Base(w), m), false)
					} else {
						add(c, fmt.Sprintf("it runs %s, which the session wrote to list the archive with %s", path.Base(w), m), true)
					}
					break
				}
			}
		}
	}
	return out
}

// pathWord is a word of a command line or a program's text that may name
// a file: what lies between spaces, quotes, brackets and commas.
var pathWord = regexp.MustCompile(`[^\s'"()\[\]{},;=]+`)

// ownRender reports whether words, read in cwd at seq, name a file that
// a metais render of the session wrote before it, and name neither the
// package nor any other file the session started with: an archive the
// session rendered and reads back, such as a .docx, which is not the
// package (W3-7). A file is one a render wrote when it lies in the
// render's output directory or is the file its output was redirected
// to, and the run's directory holds it as added, with bytes that are no
// fixture's.
func (r *run) ownRender(words []string, cwd string, seq int) bool {
	fixtures := map[string]bool{}
	for _, f := range r.rec.Fixtures {
		fixtures[f.SHA256] = true
	}
	added := map[string]bool{}
	for _, f := range r.rec.Files {
		if f.Change == "added" && !fixtures[f.SHA256] {
			added[path.Join("work", f.Path)] = true
		}
	}
	rendered := map[string]bool{}
	for _, rd := range r.renders() {
		if rd.seq >= seq {
			continue
		}
		for p := range added {
			if rd.out != "" && strings.HasPrefix(p, rd.out+"/") || slices.Contains(rd.files, p) {
				rendered[p] = true
			}
		}
	}
	own := false
	for _, w := range words {
		if strings.HasSuffix(w, ".metais") {
			return false
		}
		p := r.resolve(w, cwd)
		if slices.ContainsFunc(r.fixtureNames(), func(f string) bool { return p == path.Join("work", f) }) {
			return false
		}
		own = own || rendered[p]
	}
	return own
}

// noZip: the session extracts no archive, with a zip tool or by parsing
// it itself (R-UX-5). A listing passes, and is noted. An archive is the
// package or another file the session started with, never one it
// rendered (ownRender).
func noZip(r *run) Result {
	var ev, notes []Evidence
	for _, z := range r.zipReads() {
		if z.listing {
			notes = append(notes, z.ev)
		} else {
			ev = append(ev, z.ev)
		}
	}
	if len(ev) > 0 {
		return Result{Detail: "the session extracts the archive itself: " + ev[0].Text, Evidence: ev, Notes: notes}
	}
	return Result{Pass: true, Detail: "the session extracts no archive with a zip tool or a zip library", Notes: notes}
}

// unknownCommand is metais's finding code for a command it does not have.
const unknownCommand = "usage.unknown_command"

// noUnknown: no metais command the session ran is refused as unknown.
// The binary's own finding decides, not a list of commands here, so the
// rule follows the binary as it gains commands.
func noUnknown(r *run) Result {
	var ev []Evidence
	for i, c := range r.calls {
		if !strings.Contains(c.output(), unknownCommand) || i > 0 && r.calls[i-1].use == c.use {
			continue
		}
		// One Bash call may run several metais commands, and its output is
		// theirs together: each is named, as the scorer cannot tell which
		// one metais refused.
		var subs []string
		for _, o := range r.calls[i:] {
			if o.use == c.use {
				subs = append(subs, "metais "+orNone(o.sub))
			}
		}
		e := c.evidence()
		if len(subs) == 1 {
			e.Text = subs[0] + " is refused: " + unknownCommand
		} else {
			e.Text = "one of " + strings.Join(subs, ", ") + " is refused: " + unknownCommand
		}
		ev = append(ev, e)
	}
	if len(ev) > 0 {
		return Result{Detail: "metais refuses a command the session guessed: " + ev[0].Text, Evidence: ev}
	}
	return Result{Pass: true, Detail: fmt.Sprintf("none of the %d metais commands is refused as unknown", len(r.calls))}
}

// untouched: R-UX-8. In a consuming scenario whose prompt asks for output,
// the session leaves the package alone: it does not extract it (unzip,
// unpack, extract, or parsing it), change it, or repack it.
func untouched(r *run) Result {
	var ev []Evidence
	for _, c := range r.calls {
		switch c.sub {
		case "unpack", "extract":
			e := c.evidence()
			e.Text = "metais " + c.sub + " extracts the package"
			ev = append(ev, e)
		case "pack":
			e := c.evidence()
			e.Text = "metais pack repacks a package"
			ev = append(ev, e)
		}
	}
	for _, z := range r.zipReads() {
		if z.listing || z.cmd.name == "metais" {
			continue
		}
		e := z.ev
		e.Text = "the package is extracted: " + e.Text
		ev = append(ev, e)
	}
	fixtures := r.fixtureNames()
	for _, f := range r.rec.Files {
		if slices.Contains(fixtures, f.Path) && (f.Change == "modified" || f.Change == "deleted") {
			ev = append(ev, Evidence{File: path.Join("work", f.Path), Text: "the package is " + f.Change})
		}
	}
	for _, w := range r.writes {
		for _, f := range fixtures {
			if w.file == path.Join("work", f) {
				e := useEvidence(w.use, w.file)
				e.Text = "the session writes the package"
				ev = append(ev, e)
			}
		}
	}
	slices.SortStableFunc(ev, func(a, b Evidence) int { return lineOrder(a) - lineOrder(b) })
	if len(ev) > 0 {
		return Result{Detail: "the prompt asks for output, and " + ev[0].Text + " (R-UX-8)", Evidence: ev}
	}
	return Result{Pass: true, Detail: "the package is not extracted, changed or repacked"}
}

// lineOrder orders evidence by its line, a file's last.
func lineOrder(e Evidence) int {
	if e.Line == 0 {
		return 1 << 30
	}
	return e.Line
}
