// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/kottera/metais/internal/skill"
	"github.com/kottera/metais/internal/testenv"
)

// The wording ADR-008 A8 gives what an example is (item 3), and the
// supplier a generated-data route serves (item 4).
const (
	exampleIsWords = "An example is the author's illustration of the inputs"
	notYourData    = "not your data"
	supplierWords  = "a supplier who wants an example, a preview or data to start from"
)

// stepAt is the index in next of the step whose command line is run, or -1.
func stepAt(next []Next, run string) int {
	return slices.IndexFunc(next, func(n Next) bool { return n.Run == run })
}

// TestInspectNextOnTheFly is ADR-008 A8 items 1 and 2: inspect's next
// steps offer render --sample and sample, both before the step that lists
// a shipped example, and a route to editing the package — unpack for a
// packed package, check for a source directory, which is already the tree
// unpack writes. Neither step says that generated data is a document's
// facts.
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
		preview := stepAt(next, "metais render <package> --sample --out <dir>")
		start := stepAt(next, "metais sample <package> --out <dir>")
		example := slices.IndexFunc(next, func(n Next) bool { return strings.HasPrefix(n.Run, "metais ls <package> examples/") })
		if preview < 0 || start < 0 || example < 0 {
			t.Fatalf("%s: next %v lacks render --sample, sample or the example listing", c.pkg, next)
		}
		if preview > example || start > example {
			t.Errorf("%s: render --sample (%d) and sample (%d) come after the example listing (%d)", c.pkg, preview, start, example)
		}
		if why := next[preview].Why; !strings.HasPrefix(why, "Renders an example from generated data") {
			t.Errorf("%s: render --sample's step says %q", c.pkg, why)
		}
		if why := next[start].Why; !strings.Contains(why, "data to start from") {
			t.Errorf("%s: sample's step says %q", c.pkg, why)
		}
		for _, i := range []int{preview, start} {
			if why := next[i].Why; !strings.Contains(why, "never a real document's facts") {
				t.Errorf("%s: %q does not say generated data is no real document's facts", c.pkg, why)
			}
		}
		if why := next[example].Why; !strings.Contains(why, exampleIsWords) || !strings.Contains(why, notYourData) {
			t.Errorf("%s: the example listing's step says %q", c.pkg, why)
		}
		if stepAt(next, c.edit) < 0 || stepAt(next, c.notEdit) >= 0 {
			t.Errorf("%s: next %v, want %q and no %q", c.pkg, next, c.edit, c.notEdit)
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
	if !strings.HasPrefix(first, "  "+exampleIsWords) || !strings.Contains(first, notYourData) {
		t.Errorf("the Examples section begins %q", first)
	}
	if strings.Contains(strings.ToLower(r.stdout), "fictional") {
		t.Errorf("inspect calls something fictional:\n%s", r.stdout)
	}
}

// TestSupplierHelp is ADR-008 A8 items 3 to 5: render's and sample's help
// say they serve a supplier who wants an example, a preview or data to
// start from; sample's, which names examples, says what one is; sample is
// listed under Supplying and rendering; and no command's help calls an
// example fictional or sends a supplier to copy one.
func TestSupplierHelp(t *testing.T) {
	for _, name := range []string{"render", "sample"} {
		text := string(commandText(lookup(table, name)))
		if !strings.Contains(strings.Join(strings.Fields(text), " "), supplierWords) {
			t.Errorf("metais help %s does not name %q", name, supplierWords)
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
// (R-UX-2): its fictional-data rule and section name render --sample and
// sample, and no command line of the skill copies or reads an example's
// files.
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
	_, section, ok := strings.Cut(src, "**Fictional data, when asked for:**")
	if !ok {
		t.Fatal("the skill has no fictional-data section")
	}
	section, _, _ = strings.Cut(section, "\n## ")
	for _, want := range []string{"metais render report.metais --sample --out", "metais sample report.metais --out"} {
		if !strings.Contains(strings.Join(strings.Fields(section), " "), want) {
			t.Errorf("the fictional-data section names no %q", want)
		}
	}
}
