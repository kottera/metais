// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"bytes"
	"context"
	"slices"
	"strings"
	"testing"

	"github.com/kottera/metais/internal/cli/args"
	"github.com/kottera/metais/internal/finding"
	"github.com/kottera/metais/internal/hostenv"
)

// lineCase is one command line, the status it exits with, and the codes
// of its findings in report order.
type lineCase struct {
	args   []string
	status int
	codes  []string
	json   bool // whether a document is printed
}

// lineCases are ADR-008 D16 item 7's argument-syntax table and D13's exit
// mapping over this wave's commands: interspersed flags, --, the = and
// space forms, short aliases, refused grouping, abbreviation and
// repetition, and --json honoured on a line that does not parse.
var lineCases = []lineCase{
	// Dispatch and the global flags, before and after the command.
	{args: []string{"version"}, status: 0},
	{args: []string{"version", "--json"}, status: 0, json: true},
	{args: []string{"--json", "version"}, status: 0, json: true},
	{args: []string{"--color", "never", "version", "--json"}, status: 0, json: true},
	{args: []string{"--color=never", "limits"}, status: 0},
	{args: []string{"limits", "--color", "always"}, status: 0},
	{args: []string{"--version"}, status: 0},
	{args: []string{"--json", "--version"}, status: 0, json: true},
	{args: []string{"--version", "--json"}, status: 0, json: true},
	{args: []string{"help"}, status: 0},
	{args: []string{"help", "--json"}, status: 0, json: true},
	{args: []string{"help", "limits", "--json"}, status: 0, json: true},
	{args: []string{"--json", "help", "globs"}, status: 0, json: true},
	{args: []string{"--help"}, status: 0},
	{args: []string{"-h"}, status: 0},
	{args: []string{"--help", "--json"}, status: 0, json: true},
	{args: []string{"version", "--help"}, status: 0},
	{args: []string{"version", "-h", "--json"}, status: 0, json: true},
	{args: []string{"-h", "limits"}, status: 0},

	// -- ends the flags: what follows is an operand.
	{args: []string{"help", "--", "version"}, status: 0},
	{args: []string{"help", "--json", "--", "json"}, status: 0, json: true},
	{args: []string{"version", "--", "--json"}, status: 3, codes: []string{"usage.unexpected_operand"}},

	// No command, and an unknown command, with and without --json.
	{args: nil, status: 3, codes: []string{"usage.missing_command"}},
	{args: []string{"--json"}, status: 3, codes: []string{"usage.missing_command"}, json: true},
	{args: []string{"--color", "never"}, status: 3, codes: []string{"usage.missing_command"}},
	{args: []string{"bogus"}, status: 3, codes: []string{"usage.unknown_command"}},
	{args: []string{"bogus", "--json"}, status: 3, codes: []string{"usage.unknown_command"}, json: true},
	{args: []string{"--json", "bogus", "--bogus"}, status: 3, codes: []string{"usage.unknown_command"}, json: true},
	{args: []string{"vers"}, status: 3, codes: []string{"usage.unknown_command"}},
	{args: []string{"Version"}, status: 3, codes: []string{"usage.unknown_command"}},
	{args: []string{"--", "version"}, status: 3, codes: []string{"usage.unknown_command"}},
	{args: []string{"-"}, status: 3, codes: []string{"usage.unknown_command"}},
	{args: []string{"--version", "limits"}, status: 3, codes: []string{"usage.unexpected_operand"}},

	// A command's required operand, and standard input as a package.
	{args: []string{"inspect", "--json"}, status: 3, codes: []string{"usage.missing_operand"}, json: true},
	{args: []string{"inspect", "-", "--json"}, status: 3, codes: []string{"usage.package_from_stdin"}, json: true},
	{args: []string{"inspect", "--json", "--limit", "entries=99999999"}, status: 3, codes: []string{"usage.missing_operand"}, json: true},

	// An unknown flag, with and without --json, before and after the command.
	{args: []string{"version", "--bogus"}, status: 3, codes: []string{"usage.unknown_flag"}},
	{args: []string{"version", "--bogus", "--json"}, status: 3, codes: []string{"usage.unknown_flag"}, json: true},
	{args: []string{"--bogus", "version", "--json"}, status: 3, codes: []string{"usage.unknown_flag"}, json: true},
	{args: []string{"--json", "--bogus"}, status: 3, codes: []string{"usage.unknown_flag"}, json: true},
	{args: []string{"-r", "version"}, status: 3, codes: []string{"usage.unknown_flag"}},
	{args: []string{"version", "-x", "--json"}, status: 3, codes: []string{"usage.unknown_flag"}, json: true},
	{args: []string{"limits", "-json"}, status: 3, codes: []string{"usage.unknown_flag"}},
	{args: []string{"--version", "--bogus"}, status: 3, codes: []string{"usage.unknown_flag"}},
	{args: []string{"version", "--version"}, status: 3, codes: []string{"usage.unknown_flag"}},

	// Refused abbreviation and grouping.
	{args: []string{"version", "--js"}, status: 3, codes: []string{"usage.unknown_flag"}},
	{args: []string{"version", "--hel", "--json"}, status: 3, codes: []string{"usage.unknown_flag"}, json: true},
	{args: []string{"--vers"}, status: 3, codes: []string{"usage.unknown_flag"}},
	{args: []string{"help", "-hx", "--json"}, status: 3, codes: []string{"usage.unknown_flag"}, json: true},
	{args: []string{"-hh"}, status: 3, codes: []string{"usage.unknown_flag"}},

	// Refused repetition, within and across the command's name.
	{args: []string{"version", "--json", "--json"}, status: 3, codes: []string{"usage.repeated_flag"}, json: true},
	{args: []string{"--json", "version", "--json"}, status: 3, codes: []string{"usage.repeated_flag"}, json: true},
	{args: []string{"--json", "--json", "limits"}, status: 3, codes: []string{"usage.repeated_flag"}, json: true},
	{args: []string{"version", "-h", "--help"}, status: 3, codes: []string{"usage.repeated_flag"}},
	{args: []string{"--color", "never", "help", "--color=never"}, status: 3, codes: []string{"usage.repeated_flag"}},
	{args: []string{"--version", "--version"}, status: 3, codes: []string{"usage.repeated_flag"}},

	// Values: the = and space forms, a missing value, a boolean given one.
	{args: []string{"version", "--color"}, status: 3, codes: []string{"usage.bad_value"}},
	{args: []string{"version", "--color", "purple", "--json"}, status: 3, codes: []string{"usage.bad_value"}, json: true},
	{args: []string{"version", "--color=", "--json"}, status: 3, codes: []string{"usage.bad_value"}, json: true},
	{args: []string{"version", "--color", "--json"}, status: 3, codes: []string{"usage.bad_value"}, json: true},
	{args: []string{"--color", "purple"}, status: 3, codes: []string{"usage.bad_value"}},
	{args: []string{"version", "--json=true"}, status: 3, codes: []string{"usage.bad_value"}},
	{args: []string{"version", "--json=", "--json"}, status: 3, codes: []string{"usage.bad_value"}, json: true},
	{args: []string{"--help=yes"}, status: 3, codes: []string{"usage.bad_value"}},

	// Operands.
	{args: []string{"version", "extra"}, status: 3, codes: []string{"usage.unexpected_operand"}},
	{args: []string{"limits", "a", "--json", "b"}, status: 3, codes: []string{"usage.unexpected_operand", "usage.unexpected_operand"}, json: true},
	{args: []string{"help", "version", "limits"}, status: 3, codes: []string{"usage.unexpected_operand"}},
	{args: []string{"help", "exit-codes", "limits", "--json"}, status: 3, codes: []string{"usage.unexpected_operand"}, json: true},
	{args: []string{"help", "refusals", "limits", "version"}, status: 3, codes: []string{"usage.unexpected_operand"}},
	{args: []string{"help", "refusals", "nope"}, status: 3, codes: []string{"usage.unknown_topic"}},
	{args: []string{"help", "refusals", "exit-codes", "--json"}, status: 3, codes: []string{"usage.unknown_topic"}, json: true},
	{args: []string{"help", "refusals", "render", "--json"}, status: 0, json: true},
	{args: []string{"help", "nope"}, status: 3, codes: []string{"usage.unknown_topic"}},
	{args: []string{"help", "nope", "--json"}, status: 3, codes: []string{"usage.unknown_topic"}, json: true},
	{args: []string{"help", "Version", "--json"}, status: 3, codes: []string{"usage.unknown_topic"}, json: true},

	// Several refusals at once, each reported, and help never runs on a
	// line that does not parse.
	{args: []string{"--bogus", "version", "--js", "--json", "--color", "x"}, status: 3, codes: []string{"usage.bad_value", "usage.unknown_flag", "usage.unknown_flag"}, json: true},
	{args: []string{"version", "--help", "--bogus"}, status: 3, codes: []string{"usage.unknown_flag"}},
}

func TestLines(t *testing.T) {
	for _, c := range lineCases {
		r := runTool(t, c.args...)
		if r.status != c.status {
			t.Errorf("metais %q exited %d, want %d\nstdout %q\nstderr %q", c.args, r.status, c.status, r.stdout, r.stderr)
		}
		if (r.doc != nil) != c.json {
			t.Errorf("metais %q printed a document: %v, want %v", c.args, r.doc != nil, c.json)
		}
		if r.doc != nil {
			if got := r.codes(); !slices.Equal(got, c.codes) {
				t.Errorf("metais %q found %q, want %q", c.args, got, c.codes)
			}
			if r.doc.OK != (c.status == 0) {
				t.Errorf("metais %q: ok is %v with status %d", c.args, r.doc.OK, c.status)
			}
			continue
		}
		// Human mode: a refusal writes only to standard error, naming each
		// code; a success writes its result to standard output and nothing
		// to standard error.
		if c.status == 0 {
			if r.stdout == "" || r.stderr != "" {
				t.Errorf("metais %q succeeded with stdout %q, stderr %q", c.args, r.stdout, r.stderr)
			}
			continue
		}
		if r.stdout != "" {
			t.Errorf("metais %q was refused and wrote standard output %q", c.args, r.stdout)
		}
		for _, code := range c.codes {
			if !strings.Contains(r.stderr, "["+code+"]") {
				t.Errorf("metais %q: standard error does not name %s:\n%s", c.args, code, r.stderr)
			}
		}
		if !strings.Contains(r.stderr, "\nNext:\n  metais ") {
			t.Errorf("metais %q: standard error has no Next: block:\n%s", c.args, r.stderr)
		}
	}
}

// TestExitIsTheSameWithJSON is ADR-008 D10 item 6: every line exits with
// the same status with and without --json.
func TestExitIsTheSameWithJSON(t *testing.T) {
	for _, c := range lineCases {
		if slices.Contains(c.args, "--") || args.RequestsJSON(c.args) {
			continue
		}
		human := runTool(t, c.args...)
		withJSON := runTool(t, append(slices.Clone(c.args), "--json")...)
		if withJSON.doc == nil {
			t.Errorf("metais %q --json printed no document", c.args)
			continue
		}
		// --json added may itself be refused, as a repetition, beside
		// the line's own refusal; the status stays the usage status.
		if human.status != withJSON.status {
			t.Errorf("metais %q exits %d, and %d with --json", c.args, human.status, withJSON.status)
		}
	}
}

// TestFindingsEchoNoArgument holds every usage finding to ADR-006 D9 item
// 4: a refused argument is named by its position, never by its text.
func TestFindingsEchoNoArgument(t *testing.T) {
	const s = "SENTINELc0ffee"
	for _, argv := range [][]string{
		{s, "--json"}, {"--" + s, "--json"}, {"version", "--" + s, "--json"}, {"version", "--json=" + s, "--json"},
		{"version", "--color", s, "--json"}, {"version", s, "--json"}, {"help", s, "--json"}, {"-" + s, "--json"},
		{s}, {"help", s}, {"version", "--color=" + s},
	} {
		r := runTool(t, argv...)
		if r.status != finding.ExitUsage {
			t.Errorf("metais %q exited %d", argv, r.status)
		}
		if strings.Contains(r.stdout+r.stderr, "SENTINEL") {
			t.Errorf("metais %q echoed its argument:\n%s%s", argv, r.stdout, r.stderr)
		}
	}
}

// TestFindingArguments pins each refusal's argument parameter to the
// 1-based position of the argument it refuses, counted over the whole
// line.
func TestFindingArguments(t *testing.T) {
	for _, c := range []struct {
		args []string
		want []float64
	}{
		{[]string{"bogus", "--json"}, []float64{1}},
		{[]string{"--json", "--color", "never", "bogus"}, []float64{4}},
		{[]string{"--json", "version", "--bogus"}, []float64{3}},
		{[]string{"--json", "version", "--json"}, []float64{3}},
		{[]string{"--color", "never", "version", "--json", "--color", "x"}, []float64{5}},
		{[]string{"version", "--color", "--json"}, []float64{2}},
		{[]string{"--json", "help", "a", "b"}, []float64{4}},
		{[]string{"--json", "help", "--", "nope"}, []float64{4}},
		{[]string{"--json", "--version", "x"}, []float64{3}},
	} {
		r := runTool(t, c.args...)
		var got []float64
		for _, f := range r.doc.Findings {
			got = append(got, f.Params["argument"].(float64))
		}
		if !slices.Equal(got, c.want) {
			t.Errorf("metais %q: arguments %v, want %v", c.args, got, c.want)
		}
	}
}

// TestUsageCodesTriggered is ADR-008 D16 item 2 for the codes this item
// registers: each is triggered by a line of lineCases, and every run
// through runTool asserts its next steps are non-empty and parse.
func TestUsageCodesTriggered(t *testing.T) {
	seen := map[string]bool{}
	for _, c := range lineCases {
		if !c.json {
			continue
		}
		r := runTool(t, c.args...)
		for _, code := range r.codes() {
			seen[code] = true
			if len(r.doc.Next) == 0 {
				t.Errorf("metais %q refused with %s and no next step", c.args, code)
			}
		}
	}
	for _, code := range []finding.Code{
		finding.CodeUsageBadValue, finding.CodeUsageMissingCommand, finding.CodeUsageRepeatedFlag,
		finding.CodeUsageUnexpectedOperand, finding.CodeUsageUnknownCommand, finding.CodeUsageUnknownFlag,
		finding.CodeUsageUnknownTopic, finding.CodeUsageMissingOperand, finding.CodeUsagePackageFromStdin,
	} {
		e, ok := finding.Lookup(code)
		if !ok || e.Class != finding.ClassUsage || len(e.Statements) != 0 {
			t.Errorf("%s is not registered as a usage code naming no statement", code)
		}
		if !seen[string(code)] {
			t.Errorf("no --json line of lineCases triggers %s", code)
		}
	}
}

// TestUnknownFlagReasons pins the reason and flag parameters of the
// abbreviation and grouping refusals.
func TestUnknownFlagReasons(t *testing.T) {
	for _, c := range []struct {
		args          []string
		reason, flag  string
		command       string
		commandAbsent bool
	}{
		{[]string{"version", "--js", "--json"}, "abbreviated", "--json", "version", false},
		{[]string{"--vers", "--json"}, "abbreviated", "--version", "", true},
		{[]string{"help", "-hx", "--json"}, "grouped", "--help", "help", false},
		{[]string{"version", "--nope", "--json"}, "unknown", "", "version", false},
	} {
		r := runTool(t, c.args...)
		p := r.doc.Findings[0].Params
		if p["reason"] != c.reason || (c.flag != "" && p["flag"] != c.flag) || (c.flag == "" && p["flag"] != nil) {
			t.Errorf("metais %q: params %v", c.args, p)
		}
		if _, has := p["command"]; has == c.commandAbsent || (!c.commandAbsent && p["command"] != c.command) {
			t.Errorf("metais %q: command param %v", c.args, p["command"])
		}
	}
	r := runTool(t, "version", "--color", "nope", "--json")
	if got := r.doc.Findings[0].Params["allowed"]; len(got.([]any)) != 3 {
		t.Errorf("bad_value lists allowed values %v", got)
	}
}

// TestNoCommandPrintsOverview is ADR-008 D1 item 4: metais alone writes
// the overview to standard error and exits with the usage status; metais
// --help writes it to standard output and exits 0.
func TestNoCommandPrintsOverview(t *testing.T) {
	overview := string(overviewText(table))
	r := runTool(t)
	if r.status != finding.ExitUsage || r.stdout != "" || !strings.Contains(r.stderr, overview) {
		t.Errorf("metais: status %d, stdout %q, stderr without the overview:\n%s", r.status, r.stdout, r.stderr)
	}
	for _, flag := range []string{"--help", "-h"} {
		r := runTool(t, flag)
		if r.status != 0 || r.stdout != overview || r.stderr != "" {
			t.Errorf("metais %s: status %d, stderr %q, stdout is not the overview", flag, r.status, r.stderr)
		}
	}
}

// TestUsageErrorPrintsUsageLine is ADR-008 D15 item 6: a usage error on a
// command prints its usage line and the way to its help.
func TestUsageErrorPrintsUsageLine(t *testing.T) {
	for _, c := range table {
		r := runTool(t, c.Name, "--bogus")
		if !strings.Contains(r.stderr, "Usage:\n  metais "+c.Synopsis+"\n") || !strings.Contains(r.stderr, "metais "+c.Name+" --help") {
			t.Errorf("metais %s --bogus:\n%s", c.Name, r.stderr)
		}
	}
}

// TestVersionFlag is ADR-008 D1 item 4: metais --version is metais version.
func TestVersionFlag(t *testing.T) {
	for _, mode := range [][]string{nil, {"--json"}} {
		a := runTool(t, append([]string{"--version"}, mode...)...)
		b := runTool(t, append([]string{"version"}, mode...)...)
		if a.status != 0 || a.stdout != b.stdout || a.stderr != b.stderr {
			t.Errorf("metais --version %q differs from metais version: %q, %q", mode, a.stdout, b.stdout)
		}
	}
}

// TestDeclarationsOpeningSet is W2-1.9 as this wave leaves it: every
// declaration marks whether its command opens a package: inspect (W3-1),
// validate (W4-1), check (W4-4), test (W5-11), unpack (W4-5), render
// (W4-3) and sample (W5-10) do. init
// is an authoring command (ADR-008 D13) that opens no package: it writes a
// new source directory.
func TestDeclarationsOpeningSet(t *testing.T) {
	for _, c := range table {
		if c.Role < finding.RoleConsumer || c.Role > finding.RoleNoPackage {
			t.Errorf("%s declares no role", c.Name)
		}
	}
	consumers := []string{"inspect", "ls", "read", "outline", "grep", "schema", "extract", "validate", "render"}
	authoring := []string{"check", "sample", "test", "unpack"}
	if got, want := OpeningCommands(), slices.Concat(consumers, authoring); !slices.Equal(got, want) {
		t.Errorf("OpeningCommands() = %q, want %q", got, want)
	}
	// The marking is read from the role, both ways, and from WritesSource.
	if !(&Command{Role: finding.RoleConsumer}).OpensPackage() || !(&Command{Role: finding.RoleAuthoring}).OpensPackage() {
		t.Error("a consumer or authoring command is not marked as opening a package")
	}
	if (&Command{Role: finding.RoleAuthoring, WritesSource: true}).OpensPackage() {
		t.Error("an authoring command that writes a new source directory is marked as opening a package")
	}
	if c := lookup(table, "init"); c == nil || c.Role != finding.RoleAuthoring || !c.WritesSource {
		t.Error("init is not declared as an authoring command that writes a source directory")
	}
	r := runTool(t, "help", "--json")
	var res helpResult
	if err := jsonUnmarshal(r.doc.Result, &res); err != nil {
		t.Fatal(err)
	}
	for _, c := range res.Commands {
		want := map[string]string{"init": "authoring"}[c.Name]
		if slices.Contains(consumers, c.Name) {
			want = "consumer"
		}
		if slices.Contains(authoring, c.Name) {
			want = "authoring"
		}
		if want == "" {
			want = "none"
		}
		if c.OpensPackage != (slices.Contains(consumers, c.Name) || slices.Contains(authoring, c.Name)) || c.Role != want {
			t.Errorf("help --json marks %s as opening a package: %v, with the role %s", c.Name, c.OpensPackage, c.Role)
		}
	}
}

// TestMissingOperand is usage.missing_operand, which inspect's required
// operand makes reachable: the line is refused naming the command and the
// operand's placeholder, and --help needs no operand.
func TestMissingOperand(t *testing.T) {
	r := runTool(t, "inspect", "--json")
	if r.status != finding.ExitUsage || len(r.doc.Findings) != 1 {
		t.Fatalf("metais inspect --json: status %d, %s", r.status, r.stdout)
	}
	f := r.doc.Findings[0]
	if f.Code != string(finding.CodeUsageMissingOperand) || f.Params["command"] != "inspect" || f.Params["operand"] != "<package>" {
		t.Errorf("metais inspect --json: %+v", f)
	}
	if h := runTool(t, "inspect", "--help"); h.status != 0 || !strings.HasPrefix(h.stdout, "metais inspect") {
		t.Errorf("metais inspect --help: status %d, %q", h.status, h.stdout)
	}
}

// TestDeclarations holds the table to the rules its readers rely on.
func TestDeclarations(t *testing.T) {
	seen := map[string]bool{}
	for _, c := range table {
		if seen[c.Name] {
			t.Errorf("%s is declared twice", c.Name)
		}
		seen[c.Name] = true
		if !slices.Contains(groups, c.Group) {
			t.Errorf("%s is in no group", c.Name)
		}
		if c.Format != "metais."+c.Name+"/1" {
			t.Errorf("%s prints %s", c.Name, c.Format)
		}
		if !strings.HasPrefix(c.Synopsis, c.Name) || c.Description == "" || c.Prints == "" || len(c.Examples) == 0 || c.run == nil {
			t.Errorf("%s's declaration is incomplete", c.Name)
		}
		for _, f := range c.spec().Flags {
			if f.Description == "" {
				t.Errorf("%s --%s has no description", c.Name, f.Name)
			}
		}
		for _, r := range c.Refusals {
			if _, ok := finding.Lookup(r.Code); !ok {
				t.Errorf("%s declares %s, which is not registered", c.Name, r.Code)
			}
		}
	}
	for _, r := range toolRefusals {
		if _, ok := finding.Lookup(r.Code); !ok {
			t.Errorf("the tool declares %s, which is not registered", r.Code)
		}
	}
	// Each alias is one of ADR-008 D3 item 2's closed list.
	for _, f := range allFlags() {
		if f.Short != 0 && !strings.ContainsRune("hrCi", rune(f.Short)) {
			t.Errorf("--%s has the alias -%c, which ADR-008 D3 item 2 does not list", f.Name, f.Short)
		}
	}
}

// panicking is a table whose one command panics with a sentinel, for the
// internal-error path.
func panicking() []*Command {
	return []*Command{{
		Name: "boom", Group: "The tool", Synopsis: "boom", Description: "Panics.", Role: finding.RoleNoPackage,
		Format: "metais.version/1", Prints: "Nothing.",
		Refusals: lineRefusals("boom"),
		run:      func(*session, args.Parsed) outcome { panic("SENTINEL panic value /home/someone") },
	}}
}

// TestInternalError is ADR-006 D12 item 4: a panic in a command is
// internal.error with status 2, carrying the tool's version and the phase
// and nothing of the panic, on any stream.
func TestInternalError(t *testing.T) {
	for _, argv := range [][]string{{"boom"}, {"boom", "--json"}} {
		var stdout, stderr bytes.Buffer
		status := run(context.Background(), Invocation{Args: argv, Stdin: failingReader{t}, Stdout: &stdout, Stderr: &stderr}, panicking())
		if status != finding.ExitInternal {
			t.Errorf("metais %q exited %d, want 2", argv, status)
		}
		out := stdout.String() + stderr.String()
		if strings.Contains(out, "SENTINEL") || strings.Contains(out, "/home/") || strings.Contains(out, "goroutine") {
			t.Errorf("metais %q printed the panic:\n%s", argv, out)
		}
		if !strings.Contains(out, "internal.error") {
			t.Errorf("metais %q reported no internal.error:\n%s", argv, out)
		}
		if len(argv) == 2 {
			if stderr.Len() != 0 {
				t.Errorf("metais %q --json wrote standard error", argv)
			}
			if err := checkDocument(stdout.Bytes()); err != nil {
				t.Error(err)
			}
			if !strings.Contains(stdout.String(), `"phase":"run"`) || !strings.Contains(stdout.String(), `"tool_version":`) {
				t.Errorf("internal.error lacks its params: %s", stdout.String())
			}
		}
	}
}

// TestDispatch proves Run hands the parsed line to the command its name
// names, and that Main builds options with the display and no fault.
func TestDispatch(t *testing.T) {
	var got args.Parsed
	probe := &Command{
		Name: "probe", Group: "The tool", Synopsis: "probe", Description: "Probes.", Role: finding.RoleNoPackage,
		Format: "metais.version/1", Prints: "Nothing.", Refusals: lineRefusals("probe"),
		Operands: []args.Operand{{Name: "<a>", Repeatable: true}},
		run: func(s *session, p args.Parsed) outcome {
			got = p
			if s.inv.Options != (Options{Display: hostenv.Display{NoColor: true}}) {
				t.Errorf("the command received options %+v", s.inv.Options)
			}
			return outcome{human: []byte("probed\n")}
		},
	}
	var stdout, stderr bytes.Buffer
	inv := Invocation{Args: []string{"probe", "a", "--", "-b"}, Stdin: failingReader{t}, Stdout: &stdout, Stderr: &stderr, Options: Options{Display: hostenv.Display{NoColor: true}}}
	if status := run(context.Background(), inv, []*Command{probe}); status != 0 || stdout.String() != "probed\n" {
		t.Errorf("run = %d, stdout %q", status, stdout.String())
	}
	if !slices.Equal(got.Operands, []string{"a", "-b"}) {
		t.Errorf("the command received operands %q", got.Operands)
	}
	stdout.Reset()
	if status := Main([]string{"version"}, failingReader{t}, &stdout, &stderr, hostenv.Display{NoColor: true}); status != 0 || !strings.HasPrefix(stdout.String(), "metais ") {
		t.Errorf("Main(version) = %d, %q", status, stdout.String())
	}
}
