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
	"strconv"
	"strings"
	"testing"
	"unicode"
	"unicode/utf8"

	"metais.org/metais/internal/finding"
	"metais.org/metais/internal/hostenv"
	"metais.org/metais/internal/skill"
	"metais.org/metais/internal/version"
)

var update = flag.Bool("update", false, "rewrite the help golden files")

// helpGolden names each help output that has a golden file, by the
// command line that prints it.
func helpGolden() map[string][]string {
	g := map[string][]string{
		"overview.txt":      {"help"},
		"help.json":         {"help", "--json"},
		"command-help.json": {"version", "--help", "--json"},
		// The agent skill is help too, for a harness: metais skill --print
		// prints what metais skill writes (ADR-008 A2 item 7).
		"skill-print.md": {"skill", "--print"},
	}
	for _, c := range table {
		g["command-"+c.Name+".txt"] = []string{"help", c.Name}
	}
	for _, tp := range topics {
		g["topic-"+tp.Name+".txt"] = []string{"help", tp.Name}
	}
	for _, c := range table {
		g["topic-refusals-"+c.Name+".txt"] = []string{"help", "refusals", c.Name}
	}
	return g
}

// goldenForm is out as its golden file holds it: a JSON document indented
// so that git merges it line by line, any other text as it is.
func goldenForm(t *testing.T, file, out string) []byte {
	t.Helper()
	if filepath.Ext(file) != ".json" {
		return []byte(out)
	}
	var b bytes.Buffer
	if err := json.Indent(&b, []byte(strings.TrimSuffix(out, "\n")), "", "  "); err != nil {
		t.Fatalf("%s: %v", file, err)
	}
	return append(b.Bytes(), '\n')
}

// printedForm is what the tool prints for the golden file want holds: a
// JSON golden compacted onto one line ending in one LF, any other as it
// is. A JSON golden not in goldenForm's indentation fails, so its form
// cannot drift either.
func printedForm(t *testing.T, file string, want []byte) string {
	t.Helper()
	if filepath.Ext(file) != ".json" {
		return string(want)
	}
	var b bytes.Buffer
	if err := json.Compact(&b, want); err != nil {
		t.Fatalf("%s: %v", file, err)
	}
	printed := b.String() + "\n"
	if !bytes.Equal(goldenForm(t, file, printed), want) {
		t.Errorf("%s is not indented as -update writes it", file)
	}
	return printed
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
		// What the tool prints is what its golden file gives back, byte for
		// byte, so the comparison below holds every byte it prints.
		if back := printedForm(t, file, goldenForm(t, file, r.stdout)); back != r.stdout {
			t.Errorf("metais %q does not survive its golden file's form:\ngot:\n%s\nwant:\n%s", argv, back, r.stdout)
		}
		path := filepath.Join("testdata", "help", file)
		if *update {
			if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, goldenForm(t, file, r.stdout), 0o644); err != nil {
				t.Fatal(err)
			}
			continue
		}
		want, err := os.ReadFile(path)
		if err != nil {
			t.Errorf("%s: %v (run go test -run TestHelpGolden -args -update)", file, err)
			continue
		}
		if printed := printedForm(t, file, want); r.stdout != printed {
			t.Errorf("metais %q differs from %s:\ngot:\n%s\nwant:\n%s", argv, path, r.stdout, printed)
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
		// Every refusal is in help refusals <command> beneath its next
		// step; one with a hand-written cause is in --help with its next
		// step, and one a loop over the registry declares is in exactly
		// one of --help's family lines (ADR-008 D15 items 3 and 4, as
		// amended by A1).
		listed := nextOfCodes(runTool(t, "help", "refusals", c.Name).stdout)
		generated := 0
		for _, rf := range c.Refusals {
			if !slices.Contains(listed[rf.Code], rf.Next.Run) {
				t.Errorf("metais help refusals %s does not list the refusal %s beneath its next step", c.Name, rf.Code)
			}
			if rf.gist != "" {
				generated++
			} else if !strings.Contains(text, "\n  "+string(rf.Code)+"\n") || !strings.Contains(text, rf.Next.Run) {
				t.Errorf("metais %s --help does not show the refusal %s and its next step", c.Name, rf.Code)
			}
		}
		inFamilies := 0
		for _, f := range families(c.Refusals) {
			inFamilies += len(f.codes)
			if !strings.Contains(text, "\n  "+f.pattern()+" ("+strconv.Itoa(len(f.codes))+")") {
				t.Errorf("metais %s --help does not show the family %s", c.Name, f.pattern())
			}
			for _, code := range f.codes {
				if !matchesFamily(f.pattern(), code) {
					t.Errorf("metais %s: the family %s does not match its code %s", c.Name, f.pattern(), code)
				}
			}
		}
		if inFamilies != generated {
			t.Errorf("metais %s --help's families hold %d codes, the declaration %d from the registry", c.Name, inFamilies, generated)
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

// nextOfCodes reads metais help refusals <command>: the next steps each
// code is listed beneath.
func nextOfCodes(text string) map[finding.Code][]string {
	out := map[finding.Code][]string{}
	run := ""
	for _, line := range strings.Split(text, "\n") {
		switch {
		case strings.HasPrefix(line, "Next: "):
			run = strings.TrimPrefix(line, "Next: ")
		case strings.HasPrefix(line, "  ") && !strings.HasPrefix(line, "   ") && run != "":
			code := finding.Code(strings.TrimPrefix(line, "  "))
			out[code] = append(out[code], run)
		}
	}
	return out
}

// matchesFamily reports whether code is one the family pattern p, as
// family.pattern writes it, names.
func matchesFamily(p string, code finding.Code) bool {
	for _, alt := range strings.Split(p, " and ") {
		if prefix, ok := strings.CutSuffix(alt, ".*"); ok && strings.HasPrefix(string(code), prefix+".") || alt == string(code) {
			return true
		}
	}
	return false
}

// TestHelpListsNoGeneratedRefusalOnItsOwn holds every command's text help
// to listing on its own line only a refusal whose cause is hand-written: a
// cause that carries its code's own words in parentheses is one a loop over
// the registry built, which belongs on a family line (ADR-008 D15 item 3,
// as amended by A1, items 2 and 3).
func TestHelpListsNoGeneratedRefusalOnItsOwn(t *testing.T) {
	for _, c := range table {
		text := runTool(t, c.Name, "--help").stdout
		_, section, ok := strings.Cut(text, "\nRefusals:\n")
		if !ok {
			continue
		}
		section, _, _ = strings.Cut(section, "\n\n")
		own := map[finding.Code]string{}
		var code finding.Code
		for _, line := range strings.Split(section, "\n") {
			switch {
			case strings.HasPrefix(line, "      Next: "):
			case strings.HasPrefix(line, "      "):
				own[code] += " " + strings.TrimSpace(line)
			case strings.HasPrefix(line, "  "):
				code = finding.Code(strings.TrimPrefix(line, "  "))
				own[code] = ""
			}
		}
		if len(own) == 0 {
			t.Errorf("metais %s --help: no refusal is read on its own line", c.Name)
		}
		for _, r := range c.Refusals {
			if r.gist != "" && !strings.ContainsFunc(r.gist[:1], unicode.IsUpper) {
				t.Errorf("metais %s: the refusal %s's family cause %q names no subject", c.Name, r.Code, r.gist)
			}
		}
		for code, cause := range own {
			_, rest, _ := strings.Cut(string(code), ".")
			for _, words := range []string{
				strings.ReplaceAll(rest, "_", " "),
				strings.ReplaceAll(strings.ReplaceAll(rest, "_", " "), ".", ": "),
			} {
				if strings.Contains(cause, "("+words+")") {
					t.Errorf("metais %s --help lists %s on its own line with a generated cause:%s", c.Name, code, cause)
					break
				}
			}
		}
	}
}

// TestFamilyPattern holds a family's pattern to its codes' longest common
// prefix of whole segments.
func TestFamilyPattern(t *testing.T) {
	for _, c := range []struct {
		codes []finding.Code
		want  string
	}{
		{[]finding.Code{"json.syntax"}, "json.syntax"},
		{[]finding.Code{"json.bom", "json.syntax"}, "json.*"},
		{[]finding.Code{"output.path.a", "output.path.b"}, "output.path.*"},
		{[]finding.Code{"output.path.a", "output.pathx.b"}, "output.*"},
		{[]finding.Code{"limit.exceeded", "limit.exceeded.a"}, "limit.exceeded and limit.exceeded.*"},
	} {
		f := family{codes: c.codes}
		if got := f.pattern(); got != c.want {
			t.Errorf("pattern of %v = %q, want %q", c.codes, got, c.want)
		}
		for _, code := range c.codes {
			if !matchesFamily(f.pattern(), code) {
				t.Errorf("%q does not match %s", f.pattern(), code)
			}
		}
	}
	if matchesFamily("json.*", "jsonx.a") || matchesFamily("json.*", "json") || matchesFamily("json.bom", "json.syntax") {
		t.Error("matchesFamily matches a code outside its pattern")
	}
}

// TestFamiliesGroupByNextStep holds families to its grouping key: two
// refusals of one code family and one gist that name different next steps
// are two family lines, each with its own next step, because one line names
// one next step (ADR-008 D15 item 3, as amended by A1).
func TestFamiliesGroupByNextStep(t *testing.T) {
	again := Next{Run: "metais check <package>", Why: "again"}
	other := Next{Run: "metais inspect <package>", Why: "other"}
	got := families([]Refusal{
		{Code: "json.bom", Cause: "bom", Next: again, gist: "A file."},
		{Code: "json.syntax", Cause: "syntax", Next: other, gist: "A file."},
		{Code: "json.depth", Cause: "depth", Next: again, gist: "A file."},
		{Code: "yaml.syntax", Cause: "yaml", Next: again, gist: "A file."},
		{Code: "json.hand", Cause: "hand", Next: again},
	})
	want := []family{
		{codes: []finding.Code{"json.bom", "json.depth"}, gist: "A file.", next: again, cause: "bom"},
		{codes: []finding.Code{"json.syntax"}, gist: "A file.", next: other, cause: "syntax"},
		{codes: []finding.Code{"yaml.syntax"}, gist: "A file.", next: again, cause: "yaml"},
	}
	if !slices.EqualFunc(got, want, func(a, b family) bool {
		return slices.Equal(a.codes, b.codes) && a.gist == b.gist && a.next == b.next && a.cause == b.cause
	}) {
		t.Errorf("families = %+v, want %+v", got, want)
	}
}

// TestCauseArticle holds every command's refusal causes to an indefinite
// article that agrees with the severity after it.
func TestCauseArticle(t *testing.T) {
	for _, c := range []struct{ word, want string }{
		{"error", "an error"}, {"warning", "a warning"}, {"info", "an info"},
	} {
		if got := withArticle(c.word); got != c.want {
			t.Errorf("withArticle(%q) = %q, want %q", c.word, got, c.want)
		}
	}
	for _, c := range table {
		for _, r := range c.Refusals {
			for _, wrong := range []string{"a error", "a info", "an warning"} {
				for _, text := range []string{r.Cause, r.gist} {
					if strings.Contains(text, " "+wrong+".") {
						t.Errorf("metais %s: the refusal %s's cause %q reads %q", c.Name, r.Code, text, wrong)
					}
				}
			}
		}
	}
}

// TestParsesAsLineControl holds the line check to failing what it must.
func TestParsesAsLineControl(t *testing.T) {
	for _, l := range []string{"metais", "metais versio", "metais version --jsn", "metais help nope", "metais help limits render", "metais help refusals nope", "metais version extra", "grep x", "metais inspect <nothing>", "metais inspect 'unclosed", "metais nope <package> x", ""} {
		if parsesAsLine(l) == nil {
			t.Errorf("%q parses", l)
		}
	}
	for _, l := range []string{"metais help", "metais --help", "metais version --json", "metais help exit-codes", "metais help refusals render", "metais limits --help", "metais inspect <package>", "metais inspect '<package>' --guide"} {
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
		// The skill is read by a harness, not wrapped for a terminal: its
		// frontmatter and stamp are one line each (ADR-008 A2 items 5, 7).
		if slices.Contains(argv, "--json") || argv[0] == "skill" {
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
	// (W4-3), and of the authoring path since pack (W6-1).
	if got := declaredPaths(table); len(got) != 2 || got[0].Title != "Consuming a package" || got[1].Title != "Authoring a package" {
		t.Errorf("the table shows the paths %v, want the consuming path and the authoring path", got)
	}
	var fake []*Command
	for _, name := range []string{"inspect", "read", "schema", "validate", "render", "sample"} {
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

// TestConsumingPathOnTheFly is ADR-008 A5 item 3: the consuming path
// offers a supplier a preview from generated data, render --sample, and
// data to start from, sample, and no route through a shipped example
// (R-UX-2).
func TestConsumingPathOnTheFly(t *testing.T) {
	ps := declaredPaths(table)
	i := slices.IndexFunc(ps, func(p path) bool { return p.Title == "Consuming a package" })
	if i < 0 {
		t.Fatal("the table shows no consuming path")
	}
	steps := ps[i].Steps
	if !strings.HasPrefix(steps, "inspect → read → schema → validate → render") {
		t.Errorf("the consuming path is %q, want it to begin inspect → read → schema → validate → render", steps)
	}
	for _, want := range []string{"render --sample", "sample"} {
		if !strings.Contains(strings.TrimPrefix(steps, "inspect → read → schema → validate → render"), want) {
			t.Errorf("the consuming path %q offers no %s", steps, want)
		}
	}
	if !slices.Contains(ps[i].requires, "sample") {
		t.Errorf("the consuming path names sample but requires %v", ps[i].requires)
	}
	if strings.Contains(steps, "example") || strings.Contains(steps, "extract") {
		t.Errorf("the consuming path %q routes through a shipped example", steps)
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

// TestHelpExamplesFirst is ADR-008 Amendment A3: a command's help shows
// its examples straight after its usage, before its detail, operands,
// flags, prints and refusals, by both of the routes that print it.
func TestHelpExamplesFirst(t *testing.T) {
	for _, c := range table {
		if len(c.Examples) == 0 {
			t.Errorf("metais %s declares no example", c.Name)
			continue
		}
		for _, argv := range [][]string{{c.Name, "--help"}, {"help", c.Name}} {
			text := runTool(t, argv...).stdout
			usage := strings.Index(text, "\nUsage:\n")
			examples := strings.Index(text, "\nExamples:\n")
			if usage < 0 || examples < usage {
				t.Errorf("metais %q: no Examples: heading after Usage:\n%s", argv, text)
				continue
			}
			// The heading after Usage: is Examples:, and every example
			// is listed beneath it before the next heading.
			if next := strings.Index(text[usage+1:], "\n\n"); next < 0 || usage+2+next != examples {
				t.Errorf("metais %q: Examples: does not follow the usage lines:\n%s", argv, text)
			}
			end := len(text)
			for _, h := range []string{"Operands:", "Flags:", "Prints:", "Refusals:"} {
				if i := strings.Index(text, "\n"+h+"\n"); i >= 0 {
					end = min(end, i)
					if i < examples {
						t.Errorf("metais %q: %s comes before Examples:", argv, h)
					}
				}
			}
			if end < examples {
				continue
			}
			if c.Detail != "" {
				first, _, _ := strings.Cut(c.Detail, " ")
				if i := strings.Index(text[examples:], "\n\n"+first); i < 0 || examples+i > end {
					t.Errorf("metais %q: the detail does not follow the examples", argv)
				}
			}
			for _, e := range c.Examples {
				if !strings.Contains(text[examples:end], "  "+e.Run+"\n") {
					t.Errorf("metais %q: example %q is not listed under Examples:", argv, e.Run)
				}
			}
		}
	}
}

// TestAuthoringTopicsReachable is ADR-008 A6 item 4: the templates and
// manifest topics are where a session looks for them — the overview's
// topics, help --json, help's own operand text and examples, and the
// skill wherever it concerns an author.
func TestAuthoringTopicsReachable(t *testing.T) {
	overview := runTool(t, "help").stdout
	var res helpResult
	if err := json.Unmarshal(runTool(t, "help", "--json").doc.Result, &res); err != nil {
		t.Fatal(err)
	}
	own := runTool(t, "help", "help").stdout
	operand := lookup(table, "help").Operands[0].Description
	for _, name := range []string{"templates", "manifest"} {
		if _, ok := lookupTopic(name); !ok {
			t.Errorf("there is no topic %s", name)
			continue
		}
		if !strings.Contains(overview, "\n  "+name+" ") {
			t.Errorf("the overview does not list the topic %s", name)
		}
		if !slices.ContainsFunc(res.Topics, func(j topicJSON) bool { return j.Name == name && j.Text != "" }) {
			t.Errorf("help --json does not carry the topic %s", name)
		}
		if !strings.Contains(own, "\n  metais help "+name+"\n") {
			t.Errorf("metais help help has no example of metais help %s", name)
		}
		if !strings.Contains(string(skill.Source()), "`metais help "+name+"`") {
			t.Errorf("the skill does not name metais help %s", name)
		}
	}
	// help's operand names every topic, so none is found only by listing.
	for _, tp := range topics {
		if !strings.Contains(operand, " "+tp.Name) {
			t.Errorf("help's operand does not name the topic %s: %q", tp.Name, operand)
		}
	}
}
