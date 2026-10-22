// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"slices"
	"strings"

	"github.com/kottera/metais/internal/cli/args"
	"github.com/kottera/metais/internal/finding"
)

// Command is one command's declaration (ADR-008 D15 item 1). The parser,
// the dispatcher, the help printer and help --json all read it, so a flag
// cannot be parsed unless it is declared, and cannot be declared without
// its help.
type Command struct {
	Name     string
	Group    string // one of groups
	Synopsis string // the usage line, without its leading "metais "
	// Description is one sentence, shown in the overview and at the head
	// of the command's help.
	Description string
	// Detail is the help's body: paragraphs separated by a blank line.
	Detail string
	// Role is what the command is to a package (ADR-008 D13), which
	// decides its exit status; RoleNoPackage marks a command that opens
	// no package.
	Role finding.Role
	// WritesSource marks an authoring command that writes a new source
	// directory and opens no package: init (ADR-008 D9).
	WritesSource bool
	// Operands and Flags are the command's own; every command also takes
	// the global flags.
	Operands []args.Operand
	Flags    []args.Flag
	// Format is the name of the JSON document the command prints.
	Format string
	// Prints says what the command writes in human mode.
	Prints string
	// NextToStdout puts the human Next: block on standard output, after
	// the result, rather than on standard error (ADR-008 D11 item 3).
	NextToStdout bool
	// FindingsAreResult puts the human findings, the result and the Next:
	// block on standard output, unless the command is refused: check's
	// and test's findings are what they report (ADR-008 D11 item 3).
	FindingsAreResult bool
	Refusals          []Refusal
	Examples          []Example

	run func(s *session, p args.Parsed) outcome
}

// OpensPackage reports whether the command opens a package: every
// consumer and authoring command but one that writes a new source
// directory.
func (c *Command) OpensPackage() bool { return c.Role != finding.RoleNoPackage && !c.WritesSource }

// spec is everything the command's line may hold after its name.
func (c *Command) spec() args.Spec {
	return args.Spec{Flags: append(append([]args.Flag{}, c.Flags...), globalFlags...), Operands: c.Operands}
}

// Refusal is a code a command may refuse with, its cause, and the step
// that follows it (ADR-008 D14 item 4).
type Refusal struct {
	Code  finding.Code
	Cause string
	Next  Next
	// Hint, when set, is the hint template of this cause's findings in
	// place of the code's own: for a cause whose author the registry's
	// hint, written for every reader of the code, does not help.
	Hint string
	// match picks out the findings of this cause, where the command
	// declares Code for more than one: a finding of Code takes the next
	// step of the row whose match it satisfies, or else of the row of
	// Code with no match.
	match func(f finding.Finding) bool
	// gist is the cause of a refusal a loop over the registry declares,
	// without the code's own words: text help lists the refusals of one
	// code family that share a gist and a next step on one line (ADR-008
	// D15 item 3, as amended by A1). A hand-written refusal has none, and
	// is listed on its own.
	gist string
}

// familyRefusal is the refusal of code a loop over the registry declares:
// its cause is what with the code's own words, words, in parentheses, and
// its gist is what alone.
func familyRefusal(code finding.Code, what, words string, next Next) Refusal {
	return Refusal{Code: code, Cause: what + " (" + words + ").", Next: next, gist: what + "."}
}

// valueRefused matches usage.bad_value for a value one of flags refuses
// once read, as the command reads it: never a value missing, nor one
// written to a flag that takes none.
func valueRefused(flags ...string) func(finding.Finding) bool {
	return func(f finding.Finding) bool {
		reason, _ := f.Param("reason")
		r, _ := reason.Str()
		return r == "value_not_allowed" && flagIn(f, flags)
	}
}

// flagGiven matches a finding whose flag is one of flags.
func flagGiven(flags ...string) func(finding.Finding) bool {
	return func(f finding.Finding) bool { return flagIn(f, flags) }
}

// flagIn reports whether f's flag parameter is one of flags.
func flagIn(f finding.Finding, flags []string) bool {
	v, _ := f.Param("flag")
	flag, ok := v.Str()
	return ok && slices.Contains(flags, flag)
}

// Example is one command line of a command's help, and what it does.
type Example struct {
	Run string `json:"run"`
	Why string `json:"why"`
}

// Next is one command to run next (ADR-008 D14 item 1).
type Next struct {
	Run string `json:"run"`
	Why string `json:"why"`
}

// groups are the overview's command groups, in the order help shows them
// (ADR-008 D1 item 1).
var groups = []string{"Reading a package", "Supplying and rendering", "Authoring", "The tool"}

// The global flags, which every command takes (ADR-008 D4).
var globalFlags = []args.Flag{
	{Name: "json", Description: "Print the result as one JSON document on standard output."},
	{Name: "color", Value: "<when>", Values: []string{"auto", "always", "never"}, Default: "auto",
		Description: "Colour human output: auto colours a stream that is a terminal when NO_COLOR is unset or empty; always and never override NO_COLOR."},
	{Name: "help", Short: 'h', Description: "Print help to standard output and exit 0; nothing else runs."},
}

// versionFlag is the one flag that is written only before a command:
// metais --version is metais version (ADR-008 D1 item 4).
var versionFlag = args.Flag{Name: "version", Description: "Run metais version."}

// leadSpec is what may come before the command's name: the global flags
// and --version, and nothing else (ADR-008 D3 item 3).
var leadSpec = args.Spec{Flags: append(append([]args.Flag{}, globalFlags...), versionFlag)}

// usageFormat is the format of a document from a line that names no
// command (ADR-008 D10 item 1).
const usageFormat = "metais.usage/1"

// The steps a refusal points to.
var (
	nextHelp = Next{Run: "metais help", Why: "Lists every command, the global flags and the help topics."}
)

func nextCommandHelp(name string) Next {
	return Next{Run: "metais " + name + " --help", Why: "Prints the operands and flags metais " + name + " takes."}
}

// lineRefusals are the refusals of any command's line (ADR-008 D3).
func lineRefusals(name string) []Refusal {
	n := nextCommandHelp(name)
	return []Refusal{
		{Code: finding.CodeUsageBadValue, Cause: "A flag's value is missing, or is not one the flag accepts, or a flag that takes no value is given one.", Next: n},
		{Code: finding.CodeUsageRepeatedFlag, Cause: "A flag that may be given once is given again.", Next: n},
		{Code: finding.CodeUsageUnexpectedOperand, Cause: "An operand the command does not take.", Next: n},
		{Code: finding.CodeUsageUnknownFlag, Cause: "An argument that is not a flag the command accepts, an abbreviated flag, or one-letter aliases written together.", Next: n},
	}
}

// toolRefusals are the refusals of a line before it names a command.
var toolRefusals = []Refusal{
	{Code: finding.CodeUsageBadValue, Cause: "A global flag's value is missing or is not one it accepts, or a flag that takes no value is given one.", Next: nextHelp},
	{Code: finding.CodeUsageMissingCommand, Cause: "No command is named.", Next: nextHelp},
	{Code: finding.CodeUsageRepeatedFlag, Cause: "A global flag is given twice.", Next: nextHelp},
	{Code: finding.CodeUsageUnknownCommand, Cause: "The command's name is not one metais has.", Next: nextHelp},
	{Code: finding.CodeUsageUnknownFlag, Cause: "A flag before the command's name that is not a global flag.", Next: nextHelp},
}

// path is one golden path (ADR-008 D15 item 2), shown once every command
// it names is declared.
type path struct {
	Title    string   `json:"title"`
	Steps    string   `json:"steps"`
	requires []string // the commands Steps names
}

var paths = []path{
	{Title: "Consuming a package", Steps: "inspect → read → schema → validate → render; " +
		"preview from generated data with render --sample, or start from data generated by sample",
		requires: []string{"inspect", "read", "schema", "validate", "render", "sample"}},
	{Title: "Authoring a package", Steps: "init → edit (preview with render --sample) → check → test → pack",
		requires: []string{"init", "render", "check", "test", "pack"}},
}

// exitCode is one row of ADR-008 D13's table.
type exitCode struct {
	Code    int    `json:"code"`
	Meaning string `json:"meaning"`
}

var exitCodes = []exitCode{
	{finding.ExitOK, "Success, warnings included; help, version and limits."},
	{finding.ExitFindings, "An error in content: invalid inputs, a failing check or test, an output refused by its schema or its path, a package's own defect under an authoring command, or a warning under --fail-on warning."},
	{finding.ExitInternal, "A defect in metais itself (internal.error, or internal.memory when it stops a command holding more memory than it allows itself), or the Go runtime's status after a crash."},
	{finding.ExitUsage, "The command line is wrong: an unknown command or flag, a missing or extra operand, a malformed or repeated flag, or no command at all."},
	{finding.ExitRefused, "The package is refused: not found, unreadable, out of specification, a digest mismatch, an unsupported specification version or a package limit; or a package's own defect under a consuming command."},
	{finding.ExitDestination, "The destination is refused: not empty without --overwrite, a link or special file in the way, or a failed write."},
}

// declarations returns the command table, in the order help lists each
// group's commands (ADR-008 D1 item 1).
func declarations() []*Command {
	return slices.Concat([]*Command{inspectDeclaration()}, readingDeclarations(), []*Command{
		validateDeclaration(),
		renderDeclaration(),
		{
			Name: "init", Group: "Authoring", Synopsis: "init [<dir>] [--name <name>] [--minimal] [--html] [--dry-run] [--json]",
			Description: "Write a new package's source directory: a greeting to learn from, or with --minimal one input and one template to replace.",
			Detail: "Without --minimal, writes eight files into <dir>: metais.yaml, one input with its " +
				"schema and meaning document, one template, one example with its inputs and expected " +
				"output, and a .gitattributes that keeps git from converting line endings. The " +
				"scaffold is a small package, a greeting, to learn from and edit.\n\n" +
				"With --minimal, writes five: .gitattributes, metais.yaml, one input, data, with its " +
				"schema and meaning document, and one template, each a placeholder to replace. It has " +
				"no example, so test has nothing to compare until you add one.\n\n" +
				"With --html, the output is an HTML page in place of Markdown: the output declares " +
				"escape: html, so every value the template writes with {{ }} is HTML-escaped, and the " +
				"template writes its data into a <script> element with json_script. Either scaffold " +
				"checks, renders with render --sample and packs as written.\n\n" +
				"The package's name is --name, or else the last element of <dir> as written. When " +
				"<dir> is omitted or ends in . or .., --name is required: metais never looks up the " +
				"working directory's name. The name must satisfy the package name rule.\n\n" +
				"<dir> is created, with its missing parents, when it does not exist. An existing " +
				"<dir> must be empty, apart from names beginning with . (such as .git) other than " +
				".compiled and .gitattributes, and files ending .metais; init never overwrites, and " +
				"refuses anything else, listing up to ten of the entries it found. A failed write " +
				"stops init, which lists the files it wrote and removes none of them.",
			Role:         finding.RoleAuthoring,
			WritesSource: true,
			Operands: []args.Operand{{Name: "<dir>",
				Description: "The directory to write into, created when absent; the current directory when omitted."}},
			Flags: []args.Flag{
				{Name: "name", Value: "<name>", Description: "The package's name. It is required when <dir> is omitted or its last element is . or ..; otherwise it defaults to that element."},
				{Name: "minimal", Description: "Write the layout, one input and one template to replace, without the greeting and its example."},
				{Name: "html", Description: "Make the output an HTML page that declares escape: html, and whose template writes its data into a <script> with json_script."},
				{Name: "dry-run", Description: "Report the files init would write, and write nothing."},
				failOnFlag,
			},
			Format: "metais.init/1",
			Prints: "The files written, relative to <dir>, in the order written, then what to edit.",
			Refusals: []Refusal{
				{Code: finding.CodeUsageNameRequired, Cause: "<dir> is omitted or ends in . or .., and --name is not given.", Next: nextInitName},
				{Code: finding.CodeUsageNameInvalid, Cause: "The name, from --name or from <dir>, does not satisfy the package name rule.", Next: nextInitValidName},
				{Code: finding.CodeDestinationNotEmpty, Cause: "<dir> holds an entry other than the quiet ones.", Next: nextInitEmpty},
				{Code: finding.CodeDestinationNotDirectory, Cause: "<dir>, or a directory above it, exists and is not a directory.", Next: nextInitEmpty},
				{Code: finding.CodeDestinationWriteFailed, Cause: "A write failed after init had created something; the files already written are listed and left in place.", Next: nextInitAgain},
				{Code: finding.CodeDestinationNothingWritten, Cause: "A write failed before init had created anything; nothing was written.", Next: nextInitAgain},
			},
			Examples: []Example{
				{"metais init acme-report --minimal", "Writes a package called acme-report into the directory acme-report: the layout, one input and one template, to replace with your own."},
				{"metais init acme-report", "Writes the greeting scaffold instead, with an example, to learn from and edit."},
				{"metais init acme-report --minimal --html", "Writes the minimal package with an HTML output that escapes for HTML and writes its data into a <script> with json_script."},
				{"metais init --name acme-report --minimal", "Writes it into the current directory, which must be empty."},
				{"metais init acme-report --dry-run --json", "Lists the files it would write, as one metais.init/1 document."},
			},
			run: runInit,
		},
		checkDeclaration(),
		sampleDeclaration(),
		testDeclaration(),
		packDeclaration(),
		unpackDeclaration(),
		{
			Name: "version", Group: "The tool", Synopsis: "version [--json]",
			Description: "Print the tool's version and the specification versions it implements.",
			Detail: "The tool's version is a semantic version. Its version is independent of the " +
				"specification's: a release may implement a new minor version of the specification " +
				"without a new major version of the tool. Every specification version the tool " +
				"implements is listed, ascending, so a caller tests membership without knowing which " +
				"minor versions imply which.\n\n" +
				"metais --version, before any command, is this command.",
			Role:   finding.RoleNoPackage,
			Format: "metais.version/1",
			Prints: "Two lines: metais and the tool's version, then the specification versions it implements.",
			Examples: []Example{
				{"metais version", "Prints the tool's version and the specification versions it implements."},
				{"metais version --json", "Prints the same as one metais.version/1 document."},
			},
			run: runVersion,
		},
		{
			Name: "limits", Group: "The tool", Synopsis: "limits [--json]",
			Description: "List every resource limit, with its default, unit and class.",
			Detail: "Every size, count and ratio metais reads, computes or writes is bounded, and " +
				"every bound has a name and a default. A package within every package limit is " +
				"portable: every conforming tool accepts it at the defaults. A limit may be lowered " +
				"for one run, and is never raised.\n\n" +
				"A default in bytes is shown in the largest of KiB, MiB and GiB (powers of 1024) " +
				"that it is a whole number of; --json gives every default as a number in the " +
				"limit's unit.",
			Role:   finding.RoleNoPackage,
			Format: "metais.limits/1",
			Prints: "A table of every limit: its name, default, unit (bytes, count or ratio) and class (package, data or work), in the specification's order.",
			Examples: []Example{
				{"metais limits", "Prints the table of limits."},
				{"metais limits --json", "Prints it as one metais.limits/1 document."},
			},
			run: runLimits,
		},
		skillDeclaration(),
		{
			Name: "help", Group: "The tool", Synopsis: "help [<command> | <topic> [<command>]] [--json]",
			Description: "Print the overview, a command's help, or a topic.",
			Detail: "With no operand, prints the overview: the commands by group, the global flags, " +
				"the topics and the exit codes. With a command's name, prints that command's help, " +
				"as metais <command> --help does. With a topic, prints the topic; the topic refusals " +
				"takes a command's name after it, and lists every code that command may refuse with.\n\n" +
				"With --json, prints the help as one metais.help/1 document: every command's full " +
				"declaration and every topic for the overview, or the one command or topic asked for.",
			Role: finding.RoleNoPackage,
			Operands: []args.Operand{
				{Name: "<command> | <topic>", Description: "A command's name, or one of the topics " + topicNames() + "."},
				{Name: "<command>", Description: "After the topic refusals, the command whose codes are listed; no other topic takes one."},
			},
			Format:   "metais.help/1",
			Prints:   "The help asked for, wrapped at 80 columns, never coloured.",
			Refusals: []Refusal{{Code: finding.CodeUsageUnknownTopic, Cause: "The operand names no command and no topic, or the operand after refusals names no command.", Next: nextHelp}},
			Examples: []Example{
				{"metais help", "Prints the overview."},
				{"metais help templates", "Prints the template language, every filter with an example, and escaping."},
				{"metais help manifest", "Prints every field of metais.yaml with an example."},
				{"metais help refusals render", "Lists every code metais render may refuse with, by its next step."},
				{"metais help --json", "Prints every command's declaration and every topic as one document."},
			},
			run: runHelp,
		},
	})
}

// table is the command table every invocation is dispatched through.
var table = declarations()

// init gives every command the refusals of its line, and lists each
// command's refusals in code order.
func init() {
	for _, c := range table {
		switch c.Name {
		case "render":
			renderRefusals(c)
		case "sample":
			sampleRefusals(c)
		}
		c.Refusals = append(lineRefusals(c.Name), c.Refusals...)
		slices.SortStableFunc(c.Refusals, func(a, b Refusal) int { return strings.Compare(string(a.Code), string(b.Code)) })
	}
}

// lookup returns the command called name in t, or nil.
func lookup(t []*Command, name string) *Command {
	for _, c := range t {
		if c.Name == name {
			return c
		}
	}
	return nil
}

// OpeningCommands returns the names of the commands whose declarations
// mark them as opening a package, in table order: the set the conformance
// suite's hostile vectors are run under.
func OpeningCommands() []string {
	var out []string
	for _, c := range table {
		if c.OpensPackage() {
			out = append(out, c.Name)
		}
	}
	return out
}
