// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"bytes"
	"errors"
	"flag"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"metais.org/metais/internal/testenv"
	"metais.org/metais/internal/tools/release/buildline"
)

var update = flag.Bool("update", false, "rewrite the formula's golden file")

// The golden formula and the SHA256SUMS it is generated from, a release
// candidate's, whose hashes are of no real archive.
var (
	goldenSums    = filepath.Join("testdata", "formula", "SHA256SUMS")
	goldenFormula = filepath.Join("testdata", "formula", "metais.rb")
)

const goldenVersion = "1.0.0-rc.1"

// TestFormulaGolden holds the formula to its golden file (ADR-010 A8 item
// 3). Run with -update to rewrite it after a deliberate change, and read
// the diff: it is what the tap is next pushed. The tap's CI runs brew
// audit and brew test on what this writes (ADR-013 D7 item 5).
func TestFormulaGolden(t *testing.T) {
	got, err := formula(goldenVersion, []byte(readFile(t, goldenSums)))
	if err != nil {
		t.Fatal(err)
	}
	if *update {
		if err := os.WriteFile(goldenFormula, got, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want := readFile(t, goldenFormula)
	if string(got) != want {
		t.Errorf("the formula differs from %s (run go test -run TestFormulaGolden -args -update, and read the diff):\n%s", goldenFormula, got)
	}
}

// TestFormulaIsTheRelease reads the golden formula as Homebrew would: each
// shipped target's archive in its own block, at the GitHub release's URL,
// with its SHA256SUMS line's hash, and nothing else fetched. It checks the
// golden file itself, so an -update cannot write a formula that drifts
// from the release (ADR-013 D7 item 5).
func TestFormulaIsTheRelease(t *testing.T) {
	rb := readFile(t, goldenFormula)
	sums := map[string]string{}
	for line := range strings.Lines(readFile(t, goldenSums)) {
		sum, name, _ := strings.Cut(strings.TrimSuffix(line, "\n"), "  ")
		sums[name] = sum
	}
	block := regexp.MustCompile(`(?m)^  (on_\w+) do\n((?:    .*\n|\n)*?)  end\n`)
	arch := regexp.MustCompile(`(?m)^    (on_\w+) do\n      url "([^"]*)"\n      sha256 "([0-9a-f]{64})"\n    end\n`)
	found := map[buildline.Target]bool{}
	for _, b := range block.FindAllStringSubmatch(rb, -1) {
		for _, a := range arch.FindAllStringSubmatch(b[2], -1) {
			var tg buildline.Target
			for _, s := range buildline.Shipped {
				if brewOS[s.OS] == b[1] && brewArch[s.Arch] == a[1] {
					tg = s
				}
			}
			if tg == (buildline.Target{}) {
				t.Errorf("%s %s selects no shipped target", b[1], a[1])
				continue
			}
			name := archiveName(goldenVersion, tg)
			if want := "https://github.com/kottera/metais/releases/download/v" + goldenVersion + "/" + name; a[2] != want {
				t.Errorf("%s's url is %q, want %q", tg, a[2], want)
			}
			if a[3] != sums[name] {
				t.Errorf("%s's sha256 is %s, and SHA256SUMS has %s", tg, a[3], sums[name])
			}
			if found[tg] {
				t.Errorf("%s is selected twice", tg)
			}
			found[tg] = true
		}
	}
	if len(found) != len(buildline.Shipped) {
		t.Errorf("the formula selects %d targets, want the %d shipped", len(found), len(buildline.Shipped))
	}
	if n := strings.Count(rb, "url "); n != len(buildline.Shipped) {
		t.Errorf("the formula fetches %d URLs, want one per shipped target", n)
	}
	for _, want := range []string{
		"class Metais < Formula\n",
		`  homepage "https://metais.org"` + "\n",
		`  license "Apache-2.0"` + "\n",
		`    bin.install "metais"` + "\n",
		// ADR-013 D6 item 9: the caveat suggests metais skill.
		"        metais skill\n",
		// The test block runs the installed binary.
		`    assert_match "metais #{version}", shell_output("#{bin}/metais version")` + "\n",
	} {
		if !strings.Contains(rb, want) {
			t.Errorf("the formula does not hold %q", want)
		}
	}
	// Homebrew reads the version from the URL; a version stanza beside it
	// is one brew audit refuses as redundant.
	if strings.Contains(rb, "\n  version ") {
		t.Errorf("the formula states its version")
	}
}

func TestFormulaRefusesSums(t *testing.T) {
	good := readFile(t, goldenSums)
	lines := strings.SplitAfter(good, "\n")[:len(buildline.Shipped)]
	cases := []struct {
		name, sums, want string
	}{
		{"empty", "", "does not end with a newline"},
		{"no final newline", strings.TrimSuffix(good, "\n"), "does not end with a newline"},
		{"a line missing", strings.Join(lines[1:], ""), "has 3 lines"},
		{"a line more", good + lines[0], "has 5 lines"},
		{"a blank line", good + "\n", "has 5 lines"},
		{"out of order", lines[1] + lines[0] + strings.Join(lines[2:], ""), "line 1 names \"metais_1.0.0-rc.1_darwin_arm64.tar.gz\""},
		{"a duplicate", lines[0] + lines[0] + strings.Join(lines[2:], ""), "line 2 names \"metais_1.0.0-rc.1_darwin_amd64.tar.gz\""},
		{"another version", strings.ReplaceAll(good, "1.0.0-rc.1", "1.0.0-rc.2"), "line 1 names \"metais_1.0.0-rc.2_darwin_amd64.tar.gz\""},
		{"an unshipped target", strings.Replace(good, "linux_arm64", "windows_arm64", 1), "line 4 names"},
		{"one space", strings.Replace(good, "  ", " ", 1), "line 1, "},
		{"binary mode", strings.Replace(good, "  ", " *", 1), "line 1, "},
		{"a short hash", good[1:], "line 1, "},
		{"an uppercase hash", strings.ToUpper(good[:64]) + good[64:], "line 1, "},
		{"CRLF", strings.ReplaceAll(good, "\n", "\r\n"), "line 1 names"},
	}
	for _, c := range cases {
		_, err := formula(goldenVersion, []byte(c.sums))
		if !errors.Is(err, errSums) || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: formula returned %v, want a refusal saying %q", c.name, err, c.want)
		}
	}
	for _, v := range []string{"", "v1.0.0-rc.1", "1.0", "1.0.0+build"} {
		if _, err := formula(v, []byte(good)); !errors.Is(err, errVersion) {
			t.Errorf("formula(%q) returned %v, want %v", v, err, errVersion)
		}
	}
}

func TestFormulaCLI(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := cli([]string{"formula", "-version", goldenVersion, "-sums", goldenSums}, &stdout, &stderr, nil); code != 0 {
		t.Fatalf("formula exited %d: %s", code, stderr.String())
	}
	if stdout.String() != readFile(t, goldenFormula) {
		t.Errorf("formula wrote\n%s\nnot %s", stdout.String(), goldenFormula)
	}
	if stderr.Len() > 0 {
		t.Errorf("formula wrote %q to stderr", stderr.String())
	}
	for _, c := range []struct {
		args []string
		code int
	}{
		{[]string{"formula"}, 2},
		{[]string{"formula", "-version", goldenVersion}, 2},
		{[]string{"formula", "-sums", goldenSums}, 2},
		{[]string{"formula", "-version", goldenVersion, "-sums", goldenSums, "extra"}, 2},
		{[]string{"formula", "-unknown"}, 2},
		{[]string{"formula", "-version", goldenVersion, "-sums", filepath.Join(testenv.Dir(t), "absent")}, 1},
		{[]string{"formula", "-version", "1.0.0", "-sums", goldenSums}, 1},
	} {
		var stdout, stderr bytes.Buffer
		if code := cli(c.args, &stdout, &stderr, nil); code != c.code {
			t.Errorf("cli(%q) exited %d, want %d", c.args, code, c.code)
		}
		if stdout.Len() > 0 {
			t.Errorf("cli(%q) wrote %q to stdout", c.args, stdout.String())
		}
		if stderr.Len() == 0 {
			t.Errorf("cli(%q) wrote nothing to stderr", c.args)
		}
	}
}
