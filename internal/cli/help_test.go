// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"bytes"
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/kottera/metais/internal/hostenv"
	"github.com/kottera/metais/internal/version"
)

var update = flag.Bool("update", false, "rewrite the help golden files")

// helpGolden names each help output that has a golden file, by the
// command line that prints it.
func helpGolden() map[string][]string {
	g := map[string][]string{
		"overview.txt":      {"help"},
		"help.json":         {"help", "--json"},
		"command-help.json": {"version", "--help", "--json"},
	}
	for _, c := range table {
		g["command-"+c.Name+".txt"] = []string{"help", c.Name}
	}
	for _, tp := range topics {
		g["topic-"+tp.Name+".txt"] = []string{"help", tp.Name}
	}
	return g
}

// TestHelpGolden holds every help text, and the help documents, to golden
// files under testdata/help (ADR-008 D16 item 1). Run with -update to
// rewrite them after a deliberate change, and read the diff.
func TestHelpGolden(t *testing.T) {
	version.Override(t, "1.2.3")
	for file, argv := range helpGolden() {
		r := runTool(t, argv...)
		if r.status != 0 || r.stderr != "" {
			t.Errorf("metais %q: status %d, stderr %q", argv, r.status, r.stderr)
			continue
		}
		path := filepath.Join("testdata", "help", file)
		if *update {
			if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, []byte(r.stdout), 0o644); err != nil {
				t.Fatal(err)
			}
			continue
		}
		want, err := os.ReadFile(path)
		if err != nil {
			t.Errorf("%s: %v (run go test -run TestHelpGolden -args -update)", file, err)
			continue
		}
		if r.stdout != string(want) {
			t.Errorf("metais %q differs from %s:\ngot:\n%s\nwant:\n%s", argv, path, r.stdout, want)
		}
	}
	if *update {
		return
	}
	// Every golden file is one the test writes, so none goes stale.
	names, err := filepath.Glob(filepath.Join("testdata", "help", "*"))
	if err != nil {
		t.Fatal(err)
	}
	for _, n := range names {
		if _, ok := helpGolden()[filepath.Base(n)]; !ok {
			t.Errorf("%s is no help output's golden file", n)
		}
	}
}

// TestHelpCoverage is ADR-008 D16 item 1: every command of the dispatch
// table is in metais help and in help --json; every flag it accepts is in
// its --help and its help --json entry; and every example command line in
// help parses against the declarations.
func TestHelpCoverage(t *testing.T) {
	overview := runTool(t, "help").stdout
	all := runTool(t, "help", "--json")
	var res helpResult
	if err := json.Unmarshal(all.doc.Result, &res); err != nil {
		t.Fatal(err)
	}
	var listed []string
	for _, c := range res.Commands {
		listed = append(listed, c.Name)
	}
	for _, c := range table {
		if !strings.Contains(overview, "\n    "+c.Name+" ") {
			t.Errorf("metais help does not list %s", c.Name)
		}
		i := slices.Index(listed, c.Name)
		if i < 0 {
			t.Errorf("help --json does not list %s", c.Name)
			continue
		}
		entry := res.Commands[i]
		text := runTool(t, c.Name, "--help").stdout
		if again := runTool(t, "help", c.Name).stdout; again != text {
			t.Errorf("metais help %s differs from metais %s --help", c.Name, c.Name)
		}
		for _, f := range c.spec().Flags {
			if !strings.Contains(text, "--"+f.Name) {
				t.Errorf("metais %s --help does not show --%s", c.Name, f.Name)
			}
			if f.Short != 0 && !strings.Contains(text, "-"+string(f.Short)+", --"+f.Name) {
				t.Errorf("metais %s --help does not show -%c", c.Name, f.Short)
			}
			if !slices.ContainsFunc(entry.Flags, func(j flagJSON) bool { return j.Name == "--"+f.Name }) {
				t.Errorf("help --json does not list %s --%s", c.Name, f.Name)
			}
		}
		for _, o := range c.Operands {
			if !strings.Contains(text, o.Name) {
				t.Errorf("metais %s --help does not show %s", c.Name, o.Name)
			}
		}
		for _, rf := range c.Refusals {
			if !strings.Contains(text, string(rf.Code)) || !strings.Contains(text, rf.Next.Run) {
				t.Errorf("metais %s --help does not show the refusal %s and its next step", c.Name, rf.Code)
			}
		}
		for _, e := range c.Examples {
			if !strings.Contains(text, "\n  "+e.Run+"\n") {
				t.Errorf("metais %s --help does not show the example %q", c.Name, e.Run)
			}
		}
		if !strings.Contains(text, c.Format) {
			t.Errorf("metais %s --help does not name its format", c.Name)
		}
	}
	if len(res.Commands) != len(table) {
		t.Errorf("help --json lists %d commands, the table %d", len(res.Commands), len(table))
	}
	for _, f := range leadSpec.Flags {
		if !strings.Contains(overview, "--"+f.Name) {
			t.Errorf("metais help does not show the global flag --%s", f.Name)
		}
	}
	for _, tp := range topics {
		if !strings.Contains(overview, "\n  "+tp.Name+" ") || !slices.ContainsFunc(res.Topics, func(j topicJSON) bool { return j.Name == tp.Name }) {
			t.Errorf("the topic %s is not listed", tp.Name)
		}
	}
	for _, e := range exitCodes {
		if !strings.Contains(overview, e.Meaning[:20]) {
			t.Errorf("metais help does not show exit code %d", e.Code)
		}
	}
	// Every command line help prints parses: examples, next steps, and the
	// lines help's own text suggests.
	var lines []string
	for _, c := range res.Commands {
		for _, e := range c.Examples {
			lines = append(lines, e.Run)
		}
		for _, rf := range c.Refusals {
			lines = append(lines, rf.Next.Run)
		}
	}
	for _, rf := range res.Refusals {
		lines = append(lines, rf.Next.Run)
	}
	for _, l := range lines {
		if err := parsesAsLine(l); err != nil {
			t.Error(err)
		}
	}
	if len(lines) == 0 {
		t.Error("help has no command line to parse")
	}
}

// TestParsesAsLineControl holds the line check to failing what it must.
func TestParsesAsLineControl(t *testing.T) {
	for _, l := range []string{"metais", "metais versio", "metais version --jsn", "metais help nope", "metais version extra", "grep x", "metais inspect <nothing>", "metais inspect 'unclosed", "metais nope <package> x", ""} {
		if parsesAsLine(l) == nil {
			t.Errorf("%q parses", l)
		}
	}
	for _, l := range []string{"metais help", "metais --help", "metais version --json", "metais help exit-codes", "metais limits --help", "metais inspect <package>", "metais inspect '<package>' --guide"} {
		if err := parsesAsLine(l); err != nil {
			t.Error(err)
		}
	}
}

// TestHelpForm holds every help text to ADR-008 D11 item 7 and D15 item 6:
// wrapped at 80 columns, no trailing space, never coloured, and the same
// whatever the terminal or --color.
func TestHelpForm(t *testing.T) {
	saved := isTerminal
	t.Cleanup(func() { isTerminal = saved })
	for _, argv := range helpGolden() {
		if slices.Contains(argv, "--json") {
			continue
		}
		isTerminal = func(any) bool { return false }
		plain := runTool(t, argv...)
		for i, line := range strings.Split(plain.stdout, "\n") {
			if n := utf8.RuneCountInString(line); n > helpWidth {
				t.Errorf("metais %q line %d is %d columns:\n%s", argv, i+1, n, line)
			}
			if strings.HasSuffix(line, " ") {
				t.Errorf("metais %q line %d ends in a space", argv, i+1)
			}
		}
		if !strings.HasSuffix(plain.stdout, "\n") || strings.HasSuffix(plain.stdout, "\n\n") {
			t.Errorf("metais %q does not end in one LF", argv)
		}
		isTerminal = func(any) bool { return true }
		for _, colour := range []string{"always", "auto", "never"} {
			r := runWith(t, hostenv.Display{}, append(slices.Clone(argv), "--color", colour)...)
			if r.stdout != plain.stdout || bytes.ContainsRune([]byte(r.stdout), 0x1b) {
				t.Errorf("metais %q --color %s on a terminal differs or is coloured", argv, colour)
			}
		}
	}
}

func TestWrap(t *testing.T) {
	long := strings.Repeat("x", 90)
	for _, c := range []struct {
		text, first, rest, want string
	}{
		{"a b c", "", "", "a b c\n"},
		{"", "  ", "  ", "  \n"},
		{strings.Repeat("word ", 20), "", "", strings.TrimSpace(strings.Repeat("word ", 16)) + "\n" + strings.TrimSpace(strings.Repeat("word ", 4)) + "\n"},
		{"a " + long + " b", "> ", "  ", "> a\n  " + long + "\n  b\n"},
		{"é é", "", "", "é é\n"},
	} {
		if got := string(wrap(nil, c.text, c.first, c.rest)); got != c.want {
			t.Errorf("wrap(%q) = %q, want %q", c.text, got, c.want)
		}
	}
}

// TestGoldenPaths shows each golden path once every command it names is
// declared, and not before (ADR-008 D15 item 2).
func TestGoldenPaths(t *testing.T) {
	// Every command of the consuming path is declared since render
	// (W4-3); the authoring path's test and pack are not yet.
	if got := declaredPaths(table); len(got) != 1 || got[0].Title != "Consuming a package" {
		t.Errorf("the table shows the paths %v, want the consuming path alone", got)
	}
	var fake []*Command
	for _, name := range []string{"inspect", "read", "schema", "validate", "render"} {
		fake = append(fake, &Command{Name: name, Group: "Reading a package", Description: "D."})
	}
	got := declaredPaths(fake)
	if len(got) != 1 || got[0].Title != "Consuming a package" {
		t.Errorf("declaredPaths = %v, want the consuming path alone", got)
	}
	if !strings.Contains(string(overviewText(fake)), "Golden paths:\n  Consuming a package: inspect → read") {
		t.Errorf("the overview does not show the path:\n%s", overviewText(fake))
	}
}

// TestTopicJSONNamesEveryFormat is ADR-008 D15 item 4: the json topic
// names every format the commands print.
func TestTopicJSONNamesEveryFormat(t *testing.T) {
	text := runTool(t, "help", "json").stdout
	for _, c := range table {
		if !strings.Contains(text, c.Format) {
			t.Errorf("help json does not name %s", c.Format)
		}
	}
	if !strings.Contains(text, usageFormat) {
		t.Errorf("help json does not name %s", usageFormat)
	}
}
