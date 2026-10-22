// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"

	"metais.org/metais/internal/skill"
	"metais.org/metais/internal/testenv"
)

// The wording ADR-008 A8 gives what an example is (item 3), and A10 the
// person a generated-data route serves and what generated data never
// does (item 1), and that an example is no preview (item 2).
const (
	exampleIsWords = "An example is the author's illustration of the inputs"
	notYourData    = "not your data"
	notAPreview    = "not a preview of what the package produces, which is a render from generated data"
	supplierWords  = "a person who asked for an example, sample data or a preview of what the package produces"
	neverFacts     = "Generated data never stands in for the facts a real document needs: without them, ask the person for them."
)

// stepAt is the index in next of the step whose command line is run, or -1.
func stepAt(next []Next, run string) int {
	return slices.IndexFunc(next, func(n Next) bool { return n.Run == run })
}

// TestInspectNextOnTheFly is ADR-008 A8 items 1 and 2 as A10 amends
// them: inspect's next steps offer render --sample and sample after the
// real document's path, each only to a person who asked for an example,
// sample data or a preview, each saying that generated data never stands
// in for a real document's facts; no step names a shipped example; and
// the last is the route to editing the package — unpack for a packed
// package, check for a source directory, which is already the tree unpack
// writes.
func TestInspectNextOnTheFly(t *testing.T) {
	testenv.Home(t)
	for _, c := range []struct {
		pkg, edit, notEdit string
	}{
		{"pkg/opens-packed/package.metais", "metais unpack <package> --out <dir>", "metais check <package>"},
		{"pkg/opens-source/package", "metais check <package>", "metais unpack <package> --out <dir>"},
	} {
		r := runTool(t, "inspect", "--json", filepath.Join(vectorsDir, c.pkg))
		if !r.doc.OK {
			t.Fatalf("metais inspect %s: %s", c.pkg, r.stdout)
		}
		next := r.doc.Next
		real := stepAt(next, "metais render <package> --data <dir> --out <dir>")
		preview := stepAt(next, "metais render <package> --sample --out <dir>")
		start := stepAt(next, "metais sample <package> --out <dir>")
		if real < 0 || preview < 0 || start < 0 {
			t.Fatalf("%s: next %v lacks render --data, render --sample or sample", c.pkg, next)
		}
		if preview < real || start < real {
			t.Errorf("%s: render --sample (%d) and sample (%d) come before render --data (%d)", c.pkg, preview, start, real)
		}
		if why := next[preview].Why; !strings.HasPrefix(why, "Only for a person who asked for an example or a preview of what the package produces:") {
			t.Errorf("%s: render --sample's step says %q", c.pkg, why)
		}
		if why := next[start].Why; !strings.HasPrefix(why, "Only for a person who asked for sample data or data to start from:") {
			t.Errorf("%s: sample's step says %q", c.pkg, why)
		}
		for _, i := range []int{preview, start} {
			if why := next[i].Why; !strings.HasSuffix(why, neverFacts) {
				t.Errorf("%s: %q does not say generated data never stands in for a real document's facts", c.pkg, why)
			}
		}
		for _, n := range next {
			if strings.Contains(n.Run, "examples") || strings.Contains(n.Why, "expected output") {
				t.Errorf("%s: the step %q — %q routes to a shipped example", c.pkg, n.Run, n.Why)
			}
		}
		if next[len(next)-1].Run != c.edit || stepAt(next, c.notEdit) >= 0 {
			t.Errorf("%s: next %v, want %q last and no %q", c.pkg, next, c.edit, c.notEdit)
		}
		if i := stepAt(next, "metais check <package>"); i >= 0 && next[i].Why != checkStepWords {
			t.Errorf("%s: check's step says %q, want %q", c.pkg, next[i].Why, checkStepWords)
		}
	}
}

// checkStepWords is inspect's check step for a source directory, which
// says what MTS-AUT-003 does: every defect, or only the refusal of a
// source check cannot read (ADR-008 A12 item 1, R-W3-4).
const checkStepWords = "The source directory is the package's editable source: edit its files, then check them. check reports every defect without data, or only the refusal of a source it cannot read."

// TestInspectPreviewIsARender is ADR-008 A10 item 2 over everything
// inspect prints in text, of a package with examples that have expected
// outputs: every line that says "preview" says it is a render from
// generated data, and no line offers an example's files as one.
func TestInspectPreviewIsARender(t *testing.T) {
	testenv.Home(t)
	r := runTool(t, "inspect", filepath.Join(vectorsDir, "pkg/opens-source/package"))
	if !strings.Contains(r.stdout, "(with expected outputs)") {
		t.Fatalf("the package shows no example with expected outputs:\n%s", r.stdout)
	}
	for _, line := range strings.Split(r.stdout, "\n") {
		if strings.Contains(line, "preview") && !strings.Contains(line, "generated data") {
			t.Errorf("a preview that is not a render from generated data: %q", line)
		}
		if strings.Contains(line, "examples/") {
			t.Errorf("inspect routes to a shipped example: %q", line)
		}
	}
}

// TestInspectExamplesSay is ADR-008 A8 item 3: inspect's Examples section
// says what an example is, and nothing inspect prints claims that an
// example's facts are fictional, which the tool cannot know.
func TestInspectExamplesSay(t *testing.T) {
	testenv.Home(t)
	r := runTool(t, "inspect", filepath.Join(vectorsDir, "pkg/opens-source/package"))
	_, examples, ok := strings.Cut(r.stdout, "\nExamples\n")
	if !ok {
		t.Fatalf("no Examples section:\n%s", r.stdout)
	}
	first, _, _ := strings.Cut(examples, "\n")
	if !strings.HasPrefix(first, "  "+exampleIsWords) || !strings.Contains(first, notYourData) || !strings.Contains(first, notAPreview) {
		t.Errorf("the Examples section begins %q", first)
	}
	if strings.Contains(strings.ToLower(r.stdout), "fictional") {
		t.Errorf("inspect calls something fictional:\n%s", r.stdout)
	}
}

// TestSupplierHelp is ADR-008 A8 items 3 to 5 as A10 item 1 amends them:
// render's and sample's help say they serve a person who asked for an
// example, sample data or a preview, and that generated data never stands
// in for a real document's facts; sample's, which names examples, says what one is; sample is
// listed under Supplying and rendering; and no command's help calls an
// example fictional or sends a supplier to copy one.
func TestSupplierHelp(t *testing.T) {
	for _, name := range []string{"render", "sample"} {
		text := strings.Join(strings.Fields(string(commandText(lookup(table, name)))), " ")
		for _, want := range []string{supplierWords, neverFacts} {
			if !strings.Contains(text, want) {
				t.Errorf("metais help %s does not say %q", name, want)
			}
		}
	}
	if text := strings.Join(strings.Fields(string(commandText(lookup(table, "sample")))), " "); !strings.Contains(text, exampleIsWords) {
		t.Errorf("metais help sample does not say what an example is")
	}
	if g := lookup(table, "sample").Group; g != "Supplying and rendering" {
		t.Errorf("sample is listed under %q", g)
	}
	for _, c := range table {
		text := strings.ToLower(string(commandText(c)))
		if strings.Contains(text, "fictional") || strings.Contains(text, "author's preview") {
			t.Errorf("metais help %s calls an example fictional, or sample data the author's alone", c.Name)
		}
	}
}

// TestSkillSendsNoSupplierToAnExample is ADR-008 A8 item 6 for the skill
// (R-UX-2), as A10 item 3 amends it: its fictional-data rule and section
// name inspect before render --sample and sample, the rule counts a
// request to see what the package produces as asking for a preview, the
// skill never shows an example's expected outputs as one, and no command
// line of the skill copies or reads an example's files.
func TestSkillSendsNoSupplierToAnExample(t *testing.T) {
	src := string(skill.Source())
	for _, m := range skillCommand.FindAllStringSubmatch(src, -1) {
		if strings.Contains(m[1], "examples/") {
			t.Errorf("the skill's %q routes through a shipped example", m[1])
		}
	}
	if regexp.MustCompile(`(?i)shipped example|example's inputs`).MatchString(src) {
		t.Errorf("the skill sends a session to an example's inputs")
	}
	flat := strings.Join(strings.Fields(src), " ")
	_, rule, ok := strings.Cut(flat, "**Fictional data only when the person asks for it**")
	if !ok {
		t.Fatal("the skill has no fictional-data rule")
	}
	rule, _, _ = strings.Cut(rule, "- **Follow")
	_, section, ok := strings.Cut(flat, "**Fictional data or a preview, when asked for**")
	if !ok {
		t.Fatal("the skill has no fictional-data section")
	}
	section, _, _ = strings.Cut(section, "## ")
	for _, c := range []struct {
		name, text, inspect string
	}{
		{"rule", rule, "`metais inspect <package>`"},
		{"section", section, "`metais inspect report.metais`"},
	} {
		at := strings.Index(c.text, c.inspect)
		preview := strings.Index(c.text, "--sample")
		start := strings.Index(c.text, "` or `metais sample")
		if c.name == "section" {
			start = strings.Index(c.text, "`metais sample report.metais --out")
		}
		if at < 0 || preview < at || start < at {
			t.Errorf("the fictional-data %s names inspect at %d, render --sample at %d and sample at %d", c.name, at, preview, start)
		}
	}
	if !strings.Contains(rule, `"show me what it produces"`) {
		t.Errorf("the fictional-data rule does not count a request to see what the package produces: %s", rule)
	}
	for _, want := range []string{"never show an example's expected outputs as what the package produces", "not a preview"} {
		if !strings.Contains(rule, want) {
			t.Errorf("the fictional-data rule does not say %q", want)
		}
	}
	if !strings.Contains(section, "That render is the preview: show it, and never an example's `expected/` outputs.") {
		t.Errorf("the fictional-data section does not make the render the preview: %s", section)
	}
}

// exampleNotFactsWords is the wording ADR-008 A11 item 1 gives an
// example's inputs: never a real document's data, whose facts are asked
// for.
const exampleNotFactsWords = "An example's inputs are the author's illustration, never a real document's data: render a real document only from the facts the person supplies, and without them ask the person for them, never taking them from an example."

// TestExampleInputsAreNeverFacts is ADR-008 A11 item 1 (R-UX-2): where
// inspect shows a package's examples, and in the help of every command
// that reaches an example's entries — ls, read, extract and grep, and
// unpack, which writes them (A12 item 2) — the tool says an example's
// inputs are never a real document's data and that its facts are asked
// for; and extract's example that writes an example says so too.
func TestExampleInputsAreNeverFacts(t *testing.T) {
	testenv.Home(t)
	r := runTool(t, "inspect", filepath.Join(vectorsDir, "pkg/opens-source/package"))
	_, examples, ok := strings.Cut(r.stdout, "\nExamples\n")
	if !ok {
		t.Fatalf("no Examples section:\n%s", r.stdout)
	}
	lines := strings.Split(examples, "\n")
	if len(lines) < 2 || lines[1] != "  "+exampleNotFactsWords {
		t.Errorf("the Examples section's second line is not what an example's inputs are:\n%s", examples)
	}
	for _, name := range []string{"inspect", "extract", "read", "ls", "grep", "unpack"} {
		text := strings.Join(strings.Fields(string(commandText(lookup(table, name)))), " ")
		if !strings.Contains(text, exampleNotFactsWords) {
			t.Errorf("metais help %s does not say an example's inputs are never a real document's data", name)
		}
	}
	for _, ex := range lookup(table, "extract").Examples {
		if strings.Contains(ex.Run, "examples/") && !strings.Contains(ex.Why, "never a real document's data") {
			t.Errorf("extract's example %q says %q", ex.Run, ex.Why)
		}
	}
}

// skillSection is the flattened text of the skill from the bold heading
// head to the next bold heading or section, or "" when it has none.
func skillSection(flat, head string) string {
	_, s, ok := strings.Cut(flat, head)
	if !ok {
		return ""
	}
	if i := strings.Index(s, " **"); i >= 0 {
		s = s[:i]
	}
	s, _, _ = strings.Cut(s, "## ")
	return s
}

// TestSkillRoutesWhatAPackageNeeds is ADR-008 A11 item 2: the skill has a
// route for a person who asks what a package needs, and it leads from
// inspect to every input's fields, not to inspect alone.
func TestSkillRoutesWhatAPackageNeeds(t *testing.T) {
	flat := strings.Join(strings.Fields(string(skill.Source())), " ")
	s := skillSection(flat, "**What a package needs, when asked**")
	if s == "" {
		t.Fatal("the skill has no route for what a package needs")
	}
	inspect := strings.Index(s, "`metais inspect report.metais`")
	fields := strings.Index(s, "`metais schema report.metais --fields`")
	if inspect < 0 || fields < inspect {
		t.Errorf("the route names inspect at %d and every input's fields at %d: %s", inspect, fields, s)
	}
	if !strings.Contains(s, "not from `inspect` alone") {
		t.Errorf("the route does not say inspect alone is not the answer: %s", s)
	}
}

// TestSkillTestsAScriptOnGeneratedData is ADR-008 A11 item 3: the skill
// lets a session test a script or tool the person asked for on data
// metais sample writes inside its working directory, and keeps generated
// data from standing in for a real document's facts.
func TestSkillTestsAScriptOnGeneratedData(t *testing.T) {
	flat := strings.Join(strings.Fields(string(skill.Source())), " ")
	_, rule, _ := strings.Cut(flat, "**Fictional data only when the person asks for it**")
	rule, _, _ = strings.Cut(rule, "- **Follow")
	if !strings.Contains(rule, "or to test a script or tool the person asked for") {
		t.Errorf("the fictional-data rule makes no room for testing a script: %s", rule)
	}
	s := skillSection(flat, "**Testing a script or tool the person asked for**")
	if s == "" {
		t.Fatal("the skill has no route for testing a script")
	}
	for _, want := range []string{
		"`metais sample report.metais --out <data dir>`",
		"inside your working directory",
		"never the person's document",
		"without them, ask for them",
	} {
		if !strings.Contains(s, want) {
			t.Errorf("the script-testing route does not say %q: %s", want, s)
		}
	}
}

// TestSkillCheckSaysTheRefusal is MTS-AUT-003 as W3-11 applied it to the
// help: the skill's check step does not claim every defect of a package
// whose read is refused.
func TestSkillCheckSaysTheRefusal(t *testing.T) {
	flat := strings.Join(strings.Fields(string(skill.Source())), " ")
	_, step, ok := strings.Cut(flat, "`metais check <dir>` — ")
	if !ok {
		t.Fatal("the skill has no check step")
	}
	step, _, _ = strings.Cut(step, ". ")
	if !strings.Contains(step, "only the refusal of a package it cannot read") {
		t.Errorf("the skill's check step says %q", step)
	}
}

// TestSkillNamesEachFieldAsListed is W3-14's refinement of ADR-008 A11
// item 2: the route for what a package needs tells the session to name
// each required field in its answer as `schema --fields` lists it, by its
// name or by its title with its number, so a paraphrase that drops the word
// telling two fields apart is not the answer.
func TestSkillNamesEachFieldAsListed(t *testing.T) {
	flat := strings.Join(strings.Fields(string(skill.Source())), " ")
	s := skillSection(flat, "**What a package needs, when asked**")
	fields := strings.Index(s, "`metais schema report.metais --fields`")
	if fields < 0 {
		t.Fatalf("the route does not list every input's fields: %s", s)
	}
	answer := s[fields:]
	for _, want := range []string{
		"name every required field",
		"as the listing names it",
		"by its name",
		"by its title, with its number",
		"never by a paraphrase",
	} {
		if !strings.Contains(answer, want) {
			t.Errorf("the route's answer does not say %q: %s", want, answer)
		}
	}
}
