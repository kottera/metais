// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package site

import (
	"html"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"metais.org/metais/internal/testenv"
)

// landingBadData is the broken invoice the landing page's "Check" step
// shows marked up: no customer, a date not in the date format, a price that
// is not a whole number of cents, and a quantity of 0.
const landingBadData = "testdata/landing/bad"

// TestLandingDemos holds the landing page's demos to what metais prints. The
// page shows the invoice example's real data and output, typeset; this
// renders the page's text and requires every line of these to be in it:
// the good and the bad data, what metais validate prints for each, the
// rendered invoice's text, the identity metais inspect gives the packed
// package, and what metais skill prints after its first line (the page
// shows the version 1.0.0 there, not the build's). A change to metais's
// output, the example or the page that leaves the page showing what metais
// does not print fails here.
func TestLandingDemos(t *testing.T) {
	page := landingText(t)
	_, metais := exportedInputs(t)
	dir := testenv.Dir(t)
	run := func(want int, args ...string) string {
		t.Helper()
		cmd := exec.Command(metais, args...)
		cmd.Dir = dir
		cmd.Env = []string{"HOME=" + testenv.Home(t), "LC_ALL=C", "NO_COLOR=1", "PATH=/usr/bin:/bin"}
		out, err := cmd.CombinedOutput()
		code := 0
		if exit, ok := err.(*exec.ExitError); ok {
			code = exit.ExitCode()
		} else if err != nil {
			t.Fatal(err)
		}
		if code != want {
			t.Fatalf("metais %s exited %d, want %d:\n%s", strings.Join(args, " "), code, want, out)
		}
		return string(out)
	}

	src, err := filepath.Abs(invoiceDir)
	if err != nil {
		t.Fatal(err)
	}
	run(0, "pack", src, "--out", dir)
	const pkg = "invoice-0.1.0.metais"
	for name, from := range map[string]string{
		"good": filepath.Join(invoiceDir, "examples", "basic", "inputs", "invoice.json"),
		"bad":  filepath.Join(landingBadData, "invoice.json"),
	} {
		b, err := os.ReadFile(from)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.MkdirAll(filepath.Join(dir, name), 0o755); err != nil {
			t.Fatal(err)
		}
		writeFile(t, filepath.Join(dir, name, "invoice.json"), string(b), 0o644)
		requireLines(t, page, "the "+name+" data", string(b))
	}

	requireLines(t, page, "metais validate --data good", run(0, "validate", pkg, "--data", "good"))
	bad := run(1, "validate", pkg, "--data", "bad")
	if n := strings.Count(bad, ": error: "); n != 4 {
		t.Errorf("metais validate --data bad reports %d errors, not the 4 the page marks:\n%s", n, bad)
	}
	requireLines(t, page, "metais validate --data bad", bad)

	run(0, "render", pkg, "--data", "good", "--out", "out")
	rendered, err := os.ReadFile(filepath.Join(dir, "out", "invoice.html"))
	if err != nil {
		t.Fatal(err)
	}
	cell := regexp.MustCompile(`<(?:h1|p|td)>([^<]*)</`)
	cells := cell.FindAllStringSubmatch(string(rendered), -1)
	if len(cells) < 10 {
		t.Fatalf("the rendered invoice holds %d headings, paragraphs and cells; the check is blind:\n%s", len(cells), rendered)
	}
	for _, m := range cells {
		requireLines(t, page, "the rendered invoice", html.UnescapeString(m[1]))
	}

	inspect := strings.SplitN(run(0, "inspect", pkg), "\n", 4)
	requireLines(t, page, "metais inspect's identity", strings.Join(inspect[:3], "\n"))

	if err := os.Mkdir(filepath.Join(dir, "skilldir"), 0o755); err != nil {
		t.Fatal(err)
	}
	skill := strings.SplitN(run(0, "skill", "--harness", "claude", "--dir", "skilldir"), "\n", 2)
	if !strings.HasPrefix(skill[0], "The metais skill, from metais ") || !strings.Contains(page, "The metais skill, from metais 1.0.0:") {
		t.Errorf("metais skill's first line is %q, and the page should show it with the version 1.0.0", skill[0])
	}
	first, _, _ := strings.Cut(skill[1], "metais: warning:")
	requireLines(t, page, "metais skill", first)
}

// landingText is the built landing page's text: its markup removed, with
// the line numbers and proof marks the page sets beside and inside lines of
// data, its entities unescaped, and every run of white space one space.
func landingText(t *testing.T) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(builtSite(t), "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	s := regexp.MustCompile(`(?s)<(style|script)[^>]*>.*?</(style|script)>`).ReplaceAllString(string(b), "")
	s = regexp.MustCompile(`<span class="(?:ln|num)">[^<]*</span>`).ReplaceAllString(s, "")
	s = regexp.MustCompile(`<[^>]*>`).ReplaceAllString(s, "")
	return strings.Join(strings.Fields(html.UnescapeString(s)), " ")
}

// requireLines requires every line of out that is not blank to be in page,
// its white space collapsed as landingText collapses the page's.
func requireLines(t *testing.T, page, what, out string) {
	t.Helper()
	n := 0
	for line := range strings.SplitSeq(out, "\n") {
		line = strings.Join(strings.Fields(line), " ")
		if line == "" {
			continue
		}
		n++
		if !strings.Contains(page, line) {
			t.Errorf("the landing page does not show this line of %s:\n%s", what, line)
		}
	}
	if n == 0 {
		t.Errorf("%s printed nothing; the check is blind", what)
	}
}
