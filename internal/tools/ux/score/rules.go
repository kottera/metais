// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package score

import (
	"fmt"
	"path"
	"regexp"
	"slices"
	"strings"
)

// firstInspect: the first metais command a consuming session runs is
// metais inspect. Any metais command counts, metais --version and
// metais help among them; a command that only looks metais up, such as
// which metais, runs no metais and does not.
func firstInspect(r *run) Result {
	if len(r.calls) == 0 {
		return Result{Detail: "the session ran no metais command"}
	}
	c := r.calls[0]
	if c.sub == "inspect" {
		return Result{Pass: true, Detail: "the first metais command is metais inspect", Evidence: []Evidence{c.evidence()}}
	}
	return Result{Detail: fmt.Sprintf("the first metais command is metais %s, not metais inspect", orNone(c.sub)), Evidence: []Evidence{c.evidence()}}
}

func orNone(sub string) string {
	if sub == "" {
		return "with no command"
	}
	return sub
}

// zipTools are the commands that read a zip archive's entries. metais
// unpack and metais extract are metais's own reading, not a zip tool's:
// the plan orchestrator's ruling for this wave, from the amended R-UX-8.
var zipTools = []string{"unzip", "zipinfo", "funzip", "7z", "7za", "7zr", "bsdtar", "ditto", "jar", "unar", "lsar", "zipdetails", "zipnote"}

// zipLibrary matches a program that parses the archive itself, through a
// zip library of its language.
var zipLibrary = regexp.MustCompile(`(?i)\bzipfile\b|archive/zip|\bjszip\b|adm-zip|\byauzl\b|\bunzipper\b|java\.util\.zip|ZipArchive|Expand-Archive|System\.IO\.Compression|\bRubyzip\b|Zip::File`)

// zipReads are the session's commands that extract or parse a zip
// archive: a zip tool, tar given a package, a command line that uses a
// zip library, or a program the session wrote with one that it then ran.
func (r *run) zipReads() []Evidence {
	var out []Evidence
	seen := map[int]bool{}
	add := func(c *cmd, why string) {
		if seen[c.use.Line] {
			return
		}
		seen[c.use.Line] = true
		e := c.evidence()
		e.Text = why
		out = append(out, e)
	}
	scripts := map[string]string{}
	for _, w := range r.writes {
		if m := zipLibrary.FindString(w.text); m != "" && w.file != "" {
			scripts[path.Base(w.file)] = m
		}
	}
	for _, c := range r.cmds {
		switch {
		case slices.Contains(zipTools, c.name):
			add(c, c.name+" reads the archive")
		case c.name == "tar" && slices.ContainsFunc(c.rest, func(w string) bool { return strings.HasSuffix(w, ".metais") }):
			add(c, "tar reads the archive")
		default:
			if m := zipLibrary.FindString(c.use.Command()); m != "" {
				add(c, "the command parses the archive with "+m)
				continue
			}
			for _, w := range append([]string{c.name}, c.rest...) {
				if m, ok := scripts[path.Base(w)]; ok {
					add(c, fmt.Sprintf("it runs %s, which the session wrote to parse the archive with %s", path.Base(w), m))
					break
				}
			}
		}
	}
	return out
}

// noZip: the session extracts no zip archive, with a zip tool or by
// parsing it itself (R-UX-5).
func noZip(r *run) Result {
	if ev := r.zipReads(); len(ev) > 0 {
		return Result{Detail: "the session reads the archive itself: " + ev[0].Text, Evidence: ev}
	}
	return Result{Pass: true, Detail: "the session extracts no archive with a zip tool or a zip library"}
}

// unknownCommand is metais's finding code for a command it does not have.
const unknownCommand = "usage.unknown_command"

// noUnknown: no metais command the session ran is refused as unknown.
// The binary's own finding decides, not a list of commands here, so the
// rule follows the binary as it gains commands.
func noUnknown(r *run) Result {
	var ev []Evidence
	for _, c := range r.calls {
		if strings.Contains(c.output(), unknownCommand) {
			e := c.evidence()
			e.Text = "metais " + orNone(c.sub) + " is refused: " + unknownCommand
			if len(ev) == 0 || ev[len(ev)-1].Line != e.Line {
				ev = append(ev, e)
			}
		}
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
	for _, e := range r.zipReads() {
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
