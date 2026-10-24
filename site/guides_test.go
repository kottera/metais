// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package site

import (
	"html"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"

	"metais.org/metais/internal/testenv"
)

// invoiceDir is the example package the guide "Writing a package" writes
// (Ed's M5, W2-OOB5): its files are the ones the guide shows.
const invoiceDir = "../examples/invoice"

// codeBlock matches one code block of a built page.
var codeBlock = regexp.MustCompile(`(?s)<pre><code>(.*?)</code></pre>`)

// codeBlocks returns the text of every code block of page, a path in the
// built site, as a reader copies it: its markup removed and its
// references decoded.
func codeBlocks(t *testing.T, page string) []string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(builtSite(t), filepath.FromSlash(page)))
	if err != nil {
		t.Fatal(err)
	}
	tag := regexp.MustCompile(`<[^>]*>`)
	var out []string
	for _, m := range codeBlock.FindAllSubmatch(b, -1) {
		out = append(out, html.UnescapeString(tag.ReplaceAllString(string(m[1]), "")))
	}
	return out
}

// TestWritingGuideInvoice holds the guide "Writing a package" and
// examples/invoice identical: its schema and its template are the files
// the guide shows, byte for byte but the final newline, and its manifest
// holds the guide's inputs and outputs verbatim. The example package's
// own checks and tests are TestExamplePackages', in internal/cli.
func TestWritingGuideInvoice(t *testing.T) {
	blocks := codeBlocks(t, "guide/writing-a-package/index.html")
	for _, c := range []struct {
		file, start string
		whole       bool
	}{
		{file: "inputs/invoice.schema.json", start: "{\n  \"$schema\"", whole: true},
		{file: "templates/invoice.html.liquid", start: "<h1>Invoice ", whole: true},
		{file: "metais.yaml", start: "inputs:\n"},
	} {
		b, err := os.ReadFile(filepath.Join(invoiceDir, filepath.FromSlash(c.file)))
		if err != nil {
			t.Fatal(err)
		}
		var found []string
		for _, block := range blocks {
			if strings.HasPrefix(block, c.start) {
				found = append(found, block)
			}
		}
		if len(found) != 1 {
			t.Errorf("the guide has %d code blocks starting %q, want the one that shows %s", len(found), c.start, c.file)
			continue
		}
		block := strings.TrimSuffix(found[0], "\n") + "\n"
		switch {
		case c.whole && block != string(b):
			t.Errorf("examples/invoice/%s differs from the guide's block:\n--- the guide\n%s--- the example\n%s", c.file, block, b)
		case !c.whole && !strings.Contains(string(b), block):
			t.Errorf("examples/invoice/%s does not hold the guide's block:\n%s", c.file, block)
		}
	}
}

// guideStep is what a run of a guide's commands does around one command
// line: before, the reader's own work the guide describes ahead of it, such
// as editing a file; fails, when the guide says the command fails; and
// says, what its output must hold.
type guideStep struct {
	before func(t *testing.T, dir string)
	fails  bool
	says   []string
}

// runGuide runs every command line of the code blocks of page, a guide in
// the built site, in the order the guide gives them, in one new directory,
// with the metais built from this checkout first on PATH, and nothing of
// the host's but its system directories. A line is a metais command, or
// ./<script> for a script the guide wrote with a step's before. Each must
// succeed, unless its step says the guide has it fail. Every step must
// name a line of the guide, so a step whose command the guide no longer
// gives is a failure rather than a step never taken. It returns the
// directory.
func runGuide(t *testing.T, page string, setup func(t *testing.T, dir string), steps map[string]guideStep) string {
	t.Helper()
	_, metais := exportedInputs(t)
	bin := testenv.Dir(t)
	if err := os.Symlink(metais, filepath.Join(bin, "metais")); err != nil {
		t.Fatal(err)
	}
	dir := testenv.Dir(t)
	env := []string{
		"PATH=" + strings.Join([]string{bin, "/usr/bin", "/bin"}, string(os.PathListSeparator)),
		"HOME=" + testenv.Home(t),
		"LC_ALL=C",
	}
	setup(t, dir)
	var lines []string
	for _, block := range codeBlocks(t, page) {
		for line := range strings.SplitSeq(block, "\n") {
			if line = strings.TrimSpace(line); commandLine.MatchString(line) || strings.HasPrefix(line, "./") {
				lines = append(lines, line)
			}
		}
	}
	if len(lines) == 0 {
		t.Fatalf("%s holds no command line; the check is blind", page)
	}
	for line := range steps {
		if !slices.Contains(lines, line) {
			t.Errorf("%s does not give the command %q", page, line)
		}
	}
	for _, line := range lines {
		step := steps[line]
		if step.before != nil {
			step.before(t, dir)
		}
		cmd := exec.Command("sh", "-c", line)
		cmd.Dir = dir
		cmd.Env = env
		out, err := cmd.CombinedOutput()
		if _, ok := err.(*exec.ExitError); err != nil && !ok {
			t.Fatal(err)
		}
		switch {
		case step.fails && err == nil:
			t.Fatalf("%s: %s succeeded, and the guide says it fails:\n%s", page, line, out)
		case !step.fails && err != nil:
			t.Fatalf("%s: %s: %v\n%s", page, line, err, out)
		}
		for _, want := range step.says {
			if !strings.Contains(string(out), want) {
				t.Errorf("%s: %s does not say %q, as the guide says it does:\n%s", page, line, want, out)
			}
		}
	}
	return dir
}

// guideBlock is the one code block of page that starts with start.
func guideBlock(t *testing.T, page, start string) string {
	t.Helper()
	var found []string
	for _, block := range codeBlocks(t, page) {
		if strings.HasPrefix(block, start) {
			found = append(found, block)
		}
	}
	if len(found) != 1 {
		t.Fatalf("%s has %d code blocks starting %q, want one", page, len(found), start)
	}
	return strings.TrimSuffix(found[0], "\n") + "\n"
}

// edit replaces old in the file at name below dir with new, once, and
// fails when old is not there.
func edit(t *testing.T, dir, name, old, new string) {
	t.Helper()
	p := filepath.Join(dir, filepath.FromSlash(name))
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), old) {
		t.Fatalf("%s does not hold %q, which the guide has the reader change", name, old)
	}
	writeFile(t, p, strings.Replace(string(b), old, new, 1), 0o644)
}

// The guides' tests run the guides' command lines as a reader runs them,
// each once, in order, in one directory, so each step reads what the
// steps before it wrote. What drifts in a guide is a command's arguments:
// the package's file name, the data directory, the output directory and
// the flags. A run of one representative per command would miss a drift
// in any other line of the same command, such as a render whose --data
// names a directory no step made, and so would a run of each line alone,
// which cannot see that a line names what an earlier line wrote.

// TestUsingGuideCommands runs "Using a package" against
// examples/status-widget, packed, from the data the guide writes into
// data/status.json and with render-status.sh as the guide writes it. Its
// sample --out data starts the data of a reader who has none yet, as both
// sections that give it say, so its data directory is removed first.
func TestUsingGuideCommands(t *testing.T) {
	const page = "guide/using-a-package/index.html"
	data := guideBlock(t, page, "{\n  \"service\"")
	script := guideBlock(t, page, "#!/bin/sh\n")
	runGuide(t, page, func(t *testing.T, dir string) {
		_, metais := exportedInputs(t)
		src, err := filepath.Abs("../examples/status-widget")
		if err != nil {
			t.Fatal(err)
		}
		if out, err := exec.Command(metais, "pack", src, "--out", dir).CombinedOutput(); err != nil {
			t.Fatalf("metais pack examples/status-widget: %v\n%s", err, out)
		}
		if _, err := os.Stat(filepath.Join(dir, "status-widget-1.0.0.metais")); err != nil {
			t.Fatal(err)
		}
		writeFile(t, filepath.Join(dir, "data", "status.json"), data, 0o644)
	}, map[string]guideStep{
		"metais sample status-widget-1.0.0.metais --out data": {before: func(t *testing.T, dir string) {
			if err := os.RemoveAll(filepath.Join(dir, "data")); err != nil {
				t.Fatal(err)
			}
		}},
		"./render-status.sh": {before: func(t *testing.T, dir string) {
			writeFile(t, filepath.Join(dir, "render-status.sh"), script, 0o755)
		}},
	})
}

// TestWritingGuideCommands runs "Writing a package", from metais init to
// the second pack, doing each edit the guide asks of the reader: the
// manifest's inputs and outputs, the schema and the template as the guide
// shows them, with the scaffold's files renamed to match, and then the due
// date. Its description and guide, which the guide asks for and does not
// show, are examples/invoice's. The test of the unpacked package fails, as
// the guide says it does, until the example has a due date.
func TestWritingGuideCommands(t *testing.T) {
	const page = "guide/writing-a-package/index.html"
	manifest := guideBlock(t, page, "inputs:\n")
	schema := guideBlock(t, page, "{\n  \"$schema\"")
	template := guideBlock(t, page, "<h1>Invoice ")
	example, err := os.ReadFile(filepath.Join(invoiceDir, "metais.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	head, _, ok := strings.Cut(string(example), "\ninputs:\n")
	if !ok {
		t.Fatalf("examples/invoice/metais.yaml has no inputs")
	}
	dir := runGuide(t, page, func(*testing.T, string) {}, map[string]guideStep{
		"metais render invoice --sample --out preview": {before: func(t *testing.T, dir string) {
			pkg := filepath.Join(dir, "invoice")
			writeFile(t, filepath.Join(pkg, "metais.yaml"), head+"\n"+manifest, 0o644)
			writeFile(t, filepath.Join(pkg, "inputs", "invoice.schema.json"), schema, 0o644)
			writeFile(t, filepath.Join(pkg, "templates", "invoice.html.liquid"), template, 0o644)
			if err := os.Rename(filepath.Join(pkg, "inputs", "data.md"), filepath.Join(pkg, "inputs", "invoice.md")); err != nil {
				t.Fatal(err)
			}
			for _, old := range []string{"inputs/data.schema.json", "templates/page.html.liquid"} {
				if err := os.Remove(filepath.Join(pkg, filepath.FromSlash(old))); err != nil {
					t.Fatal(err)
				}
			}
		}},
		"metais test invoice-src": {fails: true, says: []string{"at /invoice/due", "[input.required]"}, before: func(t *testing.T, dir string) {
			src := filepath.Join(dir, "invoice-src")
			edit(t, src, "inputs/invoice.schema.json", `"required": ["number", "date", "customer", "lines"]`, `"required": ["number", "date", "customer", "lines", "due"]`)
			edit(t, src, "inputs/invoice.schema.json", `"properties": {`+"\n", `"properties": {`+"\n"+`    "due": {"type": "string", "format": "date", "title": "Due date"},`+"\n")
			edit(t, src, "templates/invoice.html.liquid", "</p>\n", "</p>\n<p>Due {{ inputs.invoice.due }}</p>\n")
			edit(t, src, "metais.yaml", `version: "0.1.0"`, `version: "0.2.0"`)
		}},
		"metais test invoice-src --update": {before: func(t *testing.T, dir string) {
			edit(t, filepath.Join(dir, "invoice-src"), "examples/basic/inputs/invoice.json", "{", `{"due": "2026-11-30",`)
		}},
	})
	for _, p := range []string{"invoice-0.1.0.metais", "invoice-0.2.0.metais"} {
		if _, err := os.Stat(filepath.Join(dir, p)); err != nil {
			t.Errorf("the guide's packs did not write %s: %v", p, err)
		}
	}
}

// TestGettingStartedRedirect requires /getting-started/, the page the
// guides replaced, to redirect to "Using a package" (ADR-013 D1 item 5,
// Ed's M7 in W2-OOB5). TestInternalLinks holds the target to a page the
// site serves.
func TestGettingStartedRedirect(t *testing.T) {
	b, err := os.ReadFile(filepath.Join(builtSite(t), "getting-started", "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	m := refreshURL.FindAllSubmatch(b, -1)
	if len(m) != 1 || string(m[0][1]) != "/guide/using-a-package/" {
		t.Errorf("getting-started/index.html redirects to %q, want once to /guide/using-a-package/", m)
	}
}

// teraWord matches tera as a word, in any case.
var teraWord = regexp.MustCompile(`(?i)\btera\b`)

// TestTeraOnlyWhoMakes requires the built site to name tera on no page but
// the landing page, and there only in "Who makes METAIS" (ADR-013 D2 item
// 1, the brief's vendor-neutrality line). TestVendorNeutral holds the help
// and the skill the reference pages come from.
func TestTeraOnlyWhoMakes(t *testing.T) {
	site := tree(t, builtSite(t))
	for p, b := range site {
		if !strings.HasSuffix(p, ".html") {
			continue
		}
		text := string(b)
		if p == "index.html" {
			who := strings.Index(text, `<h2 id="who-makes-metais">`)
			if who < 0 || !teraWord.MatchString(text[who:]) {
				t.Errorf("index.html has no \"Who makes METAIS\" naming tera")
				continue
			}
			text = text[:who]
		}
		if loc := teraWord.FindStringIndex(text); loc != nil {
			t.Errorf("%s names tera outside the landing page's \"Who makes METAIS\": %q", p, text[max(0, loc[0]-40):min(len(text), loc[1]+40)])
		}
	}
}
