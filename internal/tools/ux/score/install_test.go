// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package score

import (
	"strings"
	"testing"
)

// TestInstallRoutes: golden-install reads each route of ADR-013 D6 from
// the command that installs by it, credits the last install before the
// first metais that runs, and passes only a golden path. Each case is
// edited-i1-install-script with its install call's command replaced:
// the edges of the rule that no model shows on demand before the site is
// live.
func TestInstallRoutes(t *testing.T) {
	f := fixtureNamed(t, "edited-i1-install-script")
	const installLine = 5
	for _, c := range []struct {
		command string
		pass    bool
		detail  string
	}{
		{"curl -fsSL https://metais.org/install.sh | sh", true, "by the install script from metais.org"},
		{"wget -qO- https://metais.org/install.sh | bash", true, "by the install script from metais.org"},
		{`sh -c "$(curl -fsSL https://metais.org/install.sh)"`, true, "by the install script from metais.org"},
		{"curl -fsSLo install.sh https://metais.org/install.sh && sh install.sh", true, "by the install script from metais.org"},
		{"curl -fsSL --output /ux/run/tmp/i.sh https://metais.org/install.sh && less /ux/run/tmp/i.sh && bash /ux/run/tmp/i.sh", true, "by the install script from metais.org"},
		{"curl -fsSLO https://metais.org/install.sh; chmod +x install.sh; ./install.sh", true, "by the install script from metais.org"},
		{"curl -fsSL https://metais.org/install.sh > get.sh && sh get.sh", true, "by the install script from metais.org"},
		{"brew install kottera/tap/metais", true, "by brew install kottera/tap/metais"},
		{"brew tap kottera/tap && brew install metais", true, "by brew install kottera/tap/metais"},
		{"go install metais.org/metais/cmd/metais@latest && curl -fsSL https://metais.org/install.sh | sh", true, "by the install script from metais.org"},
		// Fetching the script without running it installs nothing.
		{"curl -fsSL https://metais.org/install.sh", false, "by no route the scorer sees"},
		{"curl -fsSL -o install.sh https://metais.org/install.sh && cat install.sh", false, "by no route the scorer sees"},
		// brew install metais without the tap is another formula.
		{"brew install metais", false, "by no route the scorer sees"},
		{"curl -fsSL https://metais.org/install.sh | sh && go install metais.org/metais/cmd/metais@latest", false, "by go install metais.org/metais/cmd/metais, not by a golden path"},
		{"gh release download v1.0.0-rc.1 -R kottera/metais -p '*darwin_arm64*' && tar xzf metais_1.0.0-rc.1_darwin_arm64.tar.gz -C ~/.local/bin metais", false, "by the GitHub release's archives, not by a golden path"},
		{"curl -fsSLO https://github.com/kottera/metais/releases/download/v1.0.0-rc.1/metais_1.0.0-rc.1_darwin_arm64.tar.gz", false, "by the GitHub release's archives, not by a golden path"},
		{"git clone https://github.com/kottera/metais && cd metais && go build -o ~/.local/bin/metais ./cmd/metais", false, "by a build from the source, not by a golden path"},
	} {
		in := probe(t, f, func(lines []string) []string {
			if !strings.Contains(lines[installLine], "metais.org/install.sh") {
				t.Fatalf("line %d is not the install: %.200s", installLine, lines[installLine])
			}
			lines[installLine] = withCommand(t, lines[installLine], c.command)
			return lines
		})
		o, err := Score(in)
		if err != nil {
			t.Fatal(err)
		}
		res := ruleResult(t, o, RuleGoldenInstall)
		if res.Pass != c.pass || !strings.Contains(res.Detail, c.detail) {
			t.Errorf("%s: golden-install is %v, %q; want %v, %q", c.command, res.Pass, res.Detail, c.pass, c.detail)
		}
	}
}

// TestInstallNeedsARun: a metais that the shell does not find, in a call
// the harness reports no error for, is no metais that ran.
func TestInstallNeedsARun(t *testing.T) {
	f := fixtureNamed(t, "edited-i1-install-script")
	in := probe(t, f, func(lines []string) []string {
		lines[7] = withCommand(t, lines[7], "metais inspect report.metais || true")
		lines[8] = strings.Replace(lines[8], `DORA major`, `(eval):1: metais: command not found\nDORA major`, 2)
		return lines
	})
	o, err := Score(in)
	if err != nil {
		t.Fatal(err)
	}
	if res := ruleResult(t, o, RuleGoldenInstall); res.Pass || !strings.Contains(res.Detail, "never runs a metais it installed") {
		t.Errorf("a metais the shell did not find: %+v", res)
	}
}
