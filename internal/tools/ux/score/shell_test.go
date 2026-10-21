// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package score

import (
	"slices"
	"strings"
	"testing"
)

// TestParseShell: the command lines sessions write are split into their
// simple commands, each named as the shell would run it.
func TestParseShell(t *testing.T) {
	for _, c := range []struct {
		line string
		want []string // each simple command's name and words, | between
	}{
		{"metais inspect report.metais", []string{"metais inspect report.metais"}},
		{"which metais; metais --version 2>&1 | head -5", []string{"which metais", "metais --version", "head -5"}},
		{"cd out && ls -la || echo none", []string{"cd out", "ls -la", "echo none"}},
		{`metais render report.metais --data "my data" --out 'the out'`, []string{"metais render report.metais --data my data --out the out"}},
		{"metais render r.metais --out - --only x.md > report.md 2>/dev/null", []string{"metais render r.metais --out - --only x.md"}},
		{"X=1 FOO=bar metais ls r.metais", []string{"X=1 FOO=bar metais ls r.metais"}},
		{"cat > data/stage.json <<'EOF'\nunzip x.zip\n\"final report\"\nEOF\nmetais validate r.metais --data data", []string{"cat", "metais validate r.metais --data data"}},
		{"cat <<-EOF > a\n\tmetais cat\n\tEOF\nls", []string{"cat", "ls"}},
		{"echo $(metais version) `unzip -l a`", []string{"echo", "metais version", "unzip -l a"}},
		{"bash -c 'unzip -l report.metais'", []string{"bash -c unzip -l report.metais", "unzip -l report.metais"}},
		{"ls # unzip report.metais\nmetais ls r", []string{"ls", "metais ls r"}},
		{"(cd out && metais test .) &> log.txt &", []string{"cd out", "metais test ."}},
		{"for f in a b; do metais validate r --data $f; done", []string{"for f in a b", "do metais validate r --data $f", "done"}},
		{`python3 -c "import zipfile; print(1)"`, []string{"python3 -c import zipfile; print(1)"}},
	} {
		var got []string
		for _, s := range parseShell(c.line) {
			got = append(got, strings.Join(s.words, " "))
		}
		if !slices.Equal(got, c.want) {
			t.Errorf("%q:\n\tgot  %q\n\twant %q", c.line, got, c.want)
		}
	}
	s := parseShell("metais render r.metais --out - > out/report.md < in.json")[0]
	if !slices.Equal(s.outs, []string{"out/report.md"}) || !slices.Equal(s.ins, []string{"in.json"}) {
		t.Errorf("redirections: outs %q, ins %q", s.outs, s.ins)
	}
}

// TestName: a command is named by its word after assignments and
// wrappers; command -v looks one up and runs nothing.
func TestName(t *testing.T) {
	for line, want := range map[string]string{
		"FOO=1 /ux/set/bin/metais inspect r": "metais",
		"./metais --help":                    "metais",
		"env -i metais ls r":                 "metais",
		"sudo unzip r":                       "unzip",
		"command -v metais":                  "command",
		"command metais ls r":                "metais",
		"do metais validate r --data $f":     "metais",
		"then ! unzip -l r":                  "unzip",
	} {
		if got, _ := parseShell(line)[0].name(); got != want {
			t.Errorf("%q is named %q, want %q", line, got, want)
		}
	}
}

// TestExpand: a variable assigned on the line is expanded, so $M runs
// the metais it names; one never assigned is left alone.
func TestExpand(t *testing.T) {
	vars := map[string]string{"M": "/ux/set/bin/metais", "W": "/ux/run/work"}
	s := parseShell(`$M render ${W}/report.metais --data $D`)[0].expand(vars)
	if got := strings.Join(s.words, " "); got != "/ux/set/bin/metais render /ux/run/work/report.metais --data $D" {
		t.Errorf("expanded to %q", got)
	}
	if as, ok := parseShell("M=/x/metais P=r.metais")[0].assignments(); !ok || as["M"] != "/x/metais" || as["P"] != "r.metais" {
		t.Errorf("assignments %v, %v", as, ok)
	}
	if _, ok := parseShell("M=/x/metais ls")[0].assignments(); ok {
		t.Error("a command with assignments is read as assignments alone")
	}
}

// TestListingFlags: unzip -l, -v, -Z and -t, zipinfo, 7z l and tar -t
// only list an archive; plain unzip, -o, -x, -p and -d extract it (Ed's
// ruling, 2026-10-21).
func TestListingFlags(t *testing.T) {
	for line, want := range map[string]bool{
		"unzip -l report.metais":           true,
		"unzip -v report.metais":           true,
		"unzip -Z report.metais":           true,
		"unzip -tq report.metais":          true,
		"7z l report.metais":               true,
		"tar -tf report.metais":            true,
		"unzip report.metais":              false,
		"unzip -o -q report.metais":        false,
		"unzip -p report.metais a.md":      false,
		"unzip report.metais -d out":       false,
		"unzip report.metais -x README.md": false,
		"tar -xf report.metais":            false,
	} {
		s := parseShell(line)[0]
		name, rest := s.name()
		if got := listingFlags(&cmd{name: name, rest: rest}); got != want {
			t.Errorf("%q lists only: %v, want %v", line, got, want)
		}
	}
}
