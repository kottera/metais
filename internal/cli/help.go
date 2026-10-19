// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/kottera/metais/internal/cli/args"
	"github.com/kottera/metais/internal/finding"
)

// helpWidth is the column help is wrapped at (ADR-008 D11 item 7).
const helpWidth = 80

// description is the tool's one-line description, at the head of the
// overview.
const description = "metais reads, validates and renders METAIS packages, and helps author them."

// topic is one help topic (ADR-008 D15 item 4).
type topic struct {
	Name  string
	Title string
	text  func(t []*Command) string
}

var topics = []topic{
	{"inputs", "Supplying a package's inputs, and which source wins.", func([]*Command) string { return inputsTopic }},
	{"globs", "The patterns that select a package's entries.", func([]*Command) string { return globsTopic }},
	{"json", "The JSON document every command prints with --json.", jsonTopic},
	{"exit-codes", "What each exit status means.", func([]*Command) string { return exitCodesTopic() }},
}

func lookupTopic(name string) (topic, bool) {
	i := slices.IndexFunc(topics, func(t topic) bool { return t.Name == name })
	if i < 0 {
		return topic{}, false
	}
	return topics[i], true
}

const inputsTopic = "A package declares its inputs, each with a schema, and a caller supplies " +
	"them from three sources, applied in this order, each to the result of the one " +
	"before:\n\n" +
	"1. The data directory, given with --data: one file per input, named after the " +
	"input with the extension of its form (vendor.json, criteria.csv), or one " +
	"directory per collection input. Each gives its input's whole value. A file that " +
	"names no declared input is a finding, and so is a second file for one input.\n\n" +
	"2. Each --inputs-json, in command-line order: a JSON object whose members are " +
	"input names, each replacing that input's whole value; values are never merged. " +
	"--inputs-json - reads the object from standard input, once.\n\n" +
	"3. Each path set, in command-line order: --set /input/pointer=text sets one value " +
	"at a JSON Pointer whose first token is an input's name, coercing the text by the " +
	"schema; --set-json /input/pointer=json sets it to a JSON value, which is how null, " +
	"an array or an object is set. Each replaces only the value at its pointer.\n\n" +
	"The order is fixed by kind, not by position: every --inputs-json applies before " +
	"every path set, wherever they are written, and among one kind the later wins. To " +
	"change one nested value and keep the rest, use a path set; --inputs-json replaces " +
	"the whole input.\n\n" +
	"Defaults from the schemas are applied after all three, and then the inputs are " +
	"validated, every finding at once."

const globsTopic = "A glob selects a package's entries by path. It is matched against the " +
	"whole entry path, segment by segment, with / as the separator, and is " +
	"case-sensitive.\n\n" +
	"*  matches any run of characters within one segment.\n" +
	"?  matches one character other than /.\n" +
	"[...] and [^...]  match one character in, or not in, a class; lo-hi is a range.\n" +
	"\\  makes the next character literal.\n" +
	"**  standing alone as a segment matches zero or more whole segments; ** anywhere " +
	"else is a usage error.\n\n" +
	"There are no braces. *, ?, [...] and ** never match a segment that begins with " +
	"a dot; a pattern segment that itself begins with a dot matches one. So **/*.md " +
	"does not reach into .compiled/, and .compiled/** does. A pattern with no wildcard " +
	"that names a directory selects every entry beneath it."

// jsonTopic describes the envelope and names every format the table's
// commands print.
func jsonTopic(t []*Command) string {
	var formats []string
	for _, c := range t {
		formats = append(formats, c.Format+" ("+c.Name+")")
	}
	formats = append(formats, usageFormat+" (a line that names no command)")
	return "With --json, a run prints exactly one JSON document on standard output: " +
		"compact UTF-8 with no byte order mark, one line ending in one LF, HTML " +
		"escaping off, and standard error empty. Its members are always present, in " +
		"this order:\n\n" +
		"format: the document's name and major version, metais.<command>/<major>.\n" +
		"ok: true when the command completed and no finding is an error.\n" +
		"findings: the findings, in the specification's order; [] when there are none. " +
		"A usage error is a finding too.\n" +
		"summary: the exact counts of errors, warnings and info, how many findings are " +
		"listed and omitted, and the count of each code.\n" +
		"result: the command's result, or null when it was refused before producing one.\n" +
		"next: the commands to run next, each {\"run\", \"why\"}; [] when there is " +
		"nothing to suggest, and never empty after a refusal.\n\n" +
		"The exit status is the same with and without --json, and --json is honoured " +
		"even on a command line that does not parse. Within a major version a format " +
		"only gains members and enumeration values: ignore a member you do not know, " +
		"and read an unknown value as other.\n\n" +
		"The formats: " + strings.Join(formats, ", ") + "."
}

func exitCodesTopic() string {
	var b strings.Builder
	b.WriteString("metais exits with the same status with and without --json:\n\n")
	for _, e := range exitCodes {
		b.WriteString(strconv.Itoa(e.Code) + "  " + e.Meaning + "\n")
	}
	b.WriteString("\nWhen several apply, the first in the order 2, 3, 4, 1, 5 is the status: " +
		"fixing a destination alone would not help a run whose inputs are invalid.")
	return b.String()
}

// overview is the outcome of metais help and metais --help.
func (s *session) overview() outcome {
	res := helpJSON(s.table, s.table, topics)
	res.Refusals = refusalsJSON(toolRefusals)
	return outcome{result: res, human: overviewText(s.table)}
}

// commandHelp is the outcome of metais help <command> and metais
// <command> --help.
func (s *session) commandHelp(c *Command) outcome {
	return outcome{result: helpJSON(s.table, []*Command{c}, nil), human: commandText(c)}
}

// topicHelp is the outcome of metais help <topic>.
func (s *session) topicHelp(tp topic) outcome {
	return outcome{result: helpJSON(s.table, nil, []topic{tp}), human: topicText(s.table, tp)}
}

// The help document, metais.help/1's result (ADR-008 D15 item 5).
type helpResult struct {
	Description string        `json:"description"`
	Usage       string        `json:"usage"`
	Paths       []path        `json:"paths"`
	Commands    []commandJSON `json:"commands"`
	GlobalFlags []flagJSON    `json:"global_flags"`
	Refusals    []refusalJSON `json:"refusals"`
	Topics      []topicJSON   `json:"topics"`
	ExitCodes   []exitCode    `json:"exit_codes"`
}

type commandJSON struct {
	Name         string        `json:"name"`
	Group        string        `json:"group"`
	Synopsis     string        `json:"synopsis"`
	Description  string        `json:"description"`
	Detail       string        `json:"detail"`
	OpensPackage bool          `json:"opens_package"`
	Role         string        `json:"role"`
	Operands     []operandJSON `json:"operands"`
	Flags        []flagJSON    `json:"flags"`
	Format       string        `json:"format"`
	Prints       string        `json:"prints"`
	Refusals     []refusalJSON `json:"refusals"`
	Examples     []Example     `json:"examples"`
}

type operandJSON struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Required    bool   `json:"required"`
	Repeatable  bool   `json:"repeatable"`
}

type flagJSON struct {
	Name        string   `json:"name"`
	Short       *string  `json:"short"`
	Value       *string  `json:"value"`
	Values      []string `json:"values"`
	Default     *string  `json:"default"`
	Repeatable  bool     `json:"repeatable"`
	Description string   `json:"description"`
}

type refusalJSON struct {
	Code  string `json:"code"`
	Cause string `json:"cause"`
	Next  Next   `json:"next"`
}

type topicJSON struct {
	Name  string `json:"name"`
	Title string `json:"title"`
	Text  string `json:"text"`
}

// helpJSON is the help document for cmds and tps, read from the table t.
func helpJSON(t, cmds []*Command, tps []topic) *helpResult {
	r := &helpResult{
		Description: description,
		Usage:       "metais <command> [operands] [flags]",
		Paths:       declaredPaths(t),
		Commands:    []commandJSON{},
		GlobalFlags: flagsJSON(leadSpec.Flags),
		Refusals:    []refusalJSON{},
		Topics:      []topicJSON{},
		ExitCodes:   exitCodes,
	}
	for _, c := range cmds {
		r.Commands = append(r.Commands, commandJSON{
			Name:         c.Name,
			Group:        c.Group,
			Synopsis:     "metais " + c.Synopsis,
			Description:  c.Description,
			Detail:       c.Detail,
			OpensPackage: c.OpensPackage(),
			Role:         roleName(c.Role),
			Operands:     operandsJSON(c.Operands),
			Flags:        flagsJSON(c.spec().Flags),
			Format:       c.Format,
			Prints:       c.Prints,
			Refusals:     refusalsJSON(c.Refusals),
			Examples:     append([]Example{}, c.Examples...),
		})
	}
	for _, tp := range tps {
		r.Topics = append(r.Topics, topicJSON{tp.Name, tp.Title, tp.text(t)})
	}
	return r
}

func roleName(r finding.Role) string {
	switch r {
	case finding.RoleConsumer:
		return "consumer"
	case finding.RoleAuthoring:
		return "authoring"
	}
	return "none"
}

func operandsJSON(ops []args.Operand) []operandJSON {
	out := []operandJSON{}
	for _, o := range ops {
		out = append(out, operandJSON{o.Name, o.Description, o.Required, o.Repeatable})
	}
	return out
}

func flagsJSON(fs []args.Flag) []flagJSON {
	opt := func(s string) *string {
		if s == "" {
			return nil
		}
		return &s
	}
	out := []flagJSON{}
	for _, f := range fs {
		var short string
		if f.Short != 0 {
			short = "-" + string(f.Short)
		}
		out = append(out, flagJSON{
			Name: "--" + f.Name, Short: opt(short), Value: opt(f.Value), Values: slices.Clone(f.Values),
			Default: opt(f.Default), Repeatable: f.Repeatable, Description: f.Description,
		})
	}
	return out
}

func refusalsJSON(rs []Refusal) []refusalJSON {
	out := []refusalJSON{}
	for _, r := range rs {
		out = append(out, refusalJSON{string(r.Code), r.Cause, r.Next})
	}
	return out
}

// declaredPaths is the golden paths every command of which t declares.
func declaredPaths(t []*Command) []path {
	out := []path{}
	for _, p := range paths {
		if !slices.ContainsFunc(p.requires, func(name string) bool { return lookup(t, name) == nil }) {
			out = append(out, p)
		}
	}
	return out
}

// overviewText is metais help (ADR-008 D15 item 2).
func overviewText(t []*Command) []byte {
	var b helpText
	b.para(description, "")
	b.heading("Usage:")
	b.line("  metais <command> [operands] [flags]")
	if ps := declaredPaths(t); len(ps) > 0 {
		b.heading("Golden paths:")
		for _, p := range ps {
			b.para(p.Title+": "+p.Steps, "  ")
		}
	}
	b.heading("Commands:")
	width := 0
	for _, c := range t {
		width = max(width, len(c.Name))
	}
	for _, g := range groups {
		first := true
		for _, c := range t {
			if c.Group != g {
				continue
			}
			if first {
				b.line("  " + g + ":")
				first = false
			}
			b.item("    "+c.Name, width+6, c.Description)
		}
	}
	b.heading("Global flags:")
	b.flags(leadSpec.Flags)
	b.heading("Topics (metais help <topic>):")
	width = 0
	for _, tp := range topics {
		width = max(width, len(tp.Name))
	}
	for _, tp := range topics {
		b.item("  "+tp.Name, width+4, tp.Title)
	}
	b.heading("Exit codes:")
	for _, e := range exitCodes {
		b.item("  "+strconv.Itoa(e.Code), 5, e.Meaning)
	}
	b.blank()
	b.para("Run 'metais help <command>' or 'metais <command> --help' for a command's operands, flags, refusals and examples.", "")
	return b.bytes()
}

// commandText is a command's help (ADR-008 D15 item 3).
func commandText(c *Command) []byte {
	var b helpText
	b.para("metais "+c.Name+" — "+c.Description, "")
	b.heading("Usage:")
	b.line("  metais " + c.Synopsis)
	if c.Detail != "" {
		b.blank()
		b.para(c.Detail, "")
	}
	if len(c.Operands) > 0 {
		b.heading("Operands:")
		for _, o := range c.Operands {
			b.line("  " + o.Name)
			text := o.Description
			if o.Required {
				text += " Required."
			}
			b.para(text, "      ")
		}
	}
	b.heading("Flags:")
	b.flags(c.spec().Flags)
	b.heading("Prints:")
	b.para(c.Prints+" With --json, one "+c.Format+" document.", "  ")
	b.heading("Refusals:")
	for _, r := range c.Refusals {
		b.line("  " + string(r.Code))
		b.para(r.Cause, "      ")
		b.para("Next: "+r.Next.Run, "      ")
	}
	b.heading("Examples:")
	for _, e := range c.Examples {
		b.line("  " + e.Run)
		b.para(e.Why, "      ")
	}
	return b.bytes()
}

// topicText is metais help <topic>.
func topicText(t []*Command, tp topic) []byte {
	var b helpText
	b.para("metais help "+tp.Name+" — "+tp.Title, "")
	b.blank()
	b.para(tp.text(t), "")
	return b.bytes()
}

// helpText builds help, wrapped at helpWidth columns.
type helpText struct{ b []byte }

func (h *helpText) bytes() []byte { return h.b }
func (h *helpText) blank()        { h.b = append(h.b, '\n') }
func (h *helpText) line(s string) { h.b = append(append(h.b, s...), '\n') }

func (h *helpText) heading(s string) {
	h.blank()
	h.line(s)
}

// para writes text wrapped at helpWidth, every line indented by indent.
// A blank line in text separates paragraphs, and a line break in it is
// kept.
func (h *helpText) para(text, indent string) {
	for i, p := range strings.Split(text, "\n") {
		if i > 0 && p == "" {
			h.blank()
			continue
		}
		h.b = wrap(h.b, p, indent, indent)
	}
}

// item writes a name in a column width wide, then text wrapped beside it.
func (h *helpText) item(name string, width int, text string) {
	if utf8.RuneCountInString(name) >= width {
		h.line(name)
		h.b = wrap(h.b, text, strings.Repeat(" ", width), strings.Repeat(" ", width))
		return
	}
	h.b = wrap(h.b, text, name+strings.Repeat(" ", width-utf8.RuneCountInString(name)), strings.Repeat(" ", width))
}

// flags writes each flag's spellings, then its description, values and
// default beneath it.
func (h *helpText) flags(fs []args.Flag) {
	for _, f := range fs {
		name := "--" + f.Name
		if f.Short != 0 {
			name = "-" + string(f.Short) + ", " + name
		}
		if !f.Boolean() {
			name += " " + f.Value
		}
		h.line("  " + name)
		text := f.Description
		if f.Values != nil {
			text += " Values: " + strings.Join(f.Values, ", ") + "."
		}
		if f.Default != "" {
			text += " Default: " + f.Default + "."
		}
		if f.Repeatable {
			text += " May be given more than once."
		}
		h.para(text, "      ")
	}
}

// wrap appends text as words filling lines of at most helpWidth columns,
// the first line beginning with first and the rest with rest. A word
// longer than a line has a line to itself.
func wrap(b []byte, text, first, rest string) []byte {
	line, n, empty := first, utf8.RuneCountInString(first), true
	for _, w := range strings.Fields(text) {
		wn := utf8.RuneCountInString(w)
		if !empty && n+1+wn > helpWidth {
			b = append(append(b, line...), '\n')
			line, n, empty = rest, utf8.RuneCountInString(rest), true
		}
		if !empty {
			line += " "
			n++
		}
		line += w
		n += wn
		empty = false
	}
	return append(append(b, line...), '\n')
}
