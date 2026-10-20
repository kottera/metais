// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/kottera/metais/internal/finding"
	"github.com/kottera/metais/internal/skill"
	"github.com/kottera/metais/internal/testenv"
	"github.com/kottera/metais/internal/version"
)

// skillDoc is a metais.skill/1 result as a test reads it.
type skillDoc struct {
	Chosen  *string
	Targets []struct {
		Path, Because, Action string
		Harnesses             []string
		From                  *string
	}
	Text *string
}

func readSkillDoc(t *testing.T, r ran) skillDoc {
	t.Helper()
	var d skillDoc
	if err := json.Unmarshal(r.doc.Result, &d); err != nil {
		t.Fatalf("metais %q: %v", r.args, err)
	}
	return d
}

// TestSkillWrites runs metais skill over each harness, and holds the
// document to what it wrote (ADR-008 A2 items 3 and 8).
func TestSkillWrites(t *testing.T) {
	version.Override(t, "1.2.3")
	for _, h := range skill.Harnesses {
		dir := testenv.Dir(t)
		r := runTool(t, "skill", "--harness", h, "--dir", dir, "--json")
		if r.status != 0 || !r.doc.OK {
			t.Fatalf("metais skill --harness %s: status %d, %s", h, r.status, r.stdout)
		}
		d := readSkillDoc(t, r)
		want := skill.PathOf(".agents")
		if h == "claude" {
			want = skill.PathOf(".claude")
		}
		if *d.Chosen != "flag" || len(d.Targets) != 1 || d.Targets[0].Path != want || d.Targets[0].Action != "installed" {
			t.Errorf("--harness %s: %s", h, r.doc.Result)
		}
		b, err := os.ReadFile(filepath.Join(dir, want))
		if err != nil || string(b) != string(skill.Text("1.2.3")) {
			t.Errorf("--harness %s: %s does not hold the skill (%v)", h, want, err)
		}
		// No .git: the warning, and success.
		if !slices.Equal(r.codes(), []string{string(finding.CodeUsageNotRepositoryRoot)}) {
			t.Errorf("--harness %s: findings %q", h, r.codes())
		}
		if strings.Contains(r.stdout, dir) {
			t.Errorf("the directory is echoed: %s", r.stdout)
		}
	}
}

// TestSkillRefusals triggers each refusal metais skill declares of its
// own, and holds its exit status and next steps (ADR-008 A2 items 2, 3, 4
// and 9).
func TestSkillRefusals(t *testing.T) {
	version.Override(t, "1.2.3")
	t.Run("no harness settled", func(t *testing.T) {
		for _, dirs := range [][]string{nil, {".claude", ".agents"}} {
			dir := testenv.Dir(t)
			for _, d := range dirs {
				os.Mkdir(filepath.Join(dir, d), 0o755)
			}
			r := runTool(t, "skill", "--dir", dir, "--json")
			if r.status != finding.ExitUsage || !slices.Equal(r.codes(), []string{string(finding.CodeUsageHarnessRequired)}) {
				t.Fatalf("%v: status %d, %q", dirs, r.status, r.codes())
			}
			var runs []string
			for _, n := range r.doc.Next {
				runs = append(runs, n.Run)
			}
			want := []string{
				"metais skill --harness claude --dir <dir>", "metais skill --harness codex --dir <dir>",
				"metais skill --harness gemini --dir <dir>", "metais skill --harness cursor --dir <dir>",
				"metais skill --harness opencode --dir <dir>", "metais skill --harness claude --harness codex --dir <dir>",
			}
			if !slices.Equal(runs, want) {
				t.Errorf("next %q", runs)
			}
			if entries, _ := os.ReadDir(dir); len(entries) != len(dirs) {
				t.Errorf("wrote %v", entries)
			}
		}
	})
	t.Run("an unexpanded ~", func(t *testing.T) {
		for _, argv := range [][]string{{"--dir=~"}, {"--dir=~/x"}, {"--dir", "~"}} {
			r := runTool(t, append([]string{"skill", "--harness", "claude", "--json"}, argv...)...)
			if r.status != finding.ExitUsage || !slices.Equal(r.codes(), []string{string(finding.CodeUsageUnexpandedTilde)}) {
				t.Errorf("%q: status %d, %q", argv, r.status, r.codes())
			}
		}
	})
	t.Run("an edited copy", func(t *testing.T) {
		dir := testenv.Dir(t)
		p := filepath.Join(dir, ".claude", "skills", "metais")
		os.MkdirAll(p, 0o755)
		os.WriteFile(filepath.Join(p, "SKILL.md"), []byte("# ours\n"), 0o644)
		r := runTool(t, "skill", "--dir", dir, "--json")
		if r.status != finding.ExitUsage || !slices.Equal(r.codes(), []string{string(finding.CodeUsageSkillEdited)}) {
			t.Fatalf("status %d, %q", r.status, r.codes())
		}
		if r.doc.Next[0].Run != "metais skill --overwrite --dir <dir>" || r.doc.Next[1].Run != "metais skill --print" {
			t.Errorf("next %v", r.doc.Next)
		}
		if b, _ := os.ReadFile(filepath.Join(p, "SKILL.md")); string(b) != "# ours\n" {
			t.Error("the edited copy was changed")
		}
		r = runTool(t, "skill", "--dir", dir, "--overwrite", "--json")
		if r.status != 0 || readSkillDoc(t, r).Targets[0].Action != "replaced" {
			t.Errorf("--overwrite: status %d, %s", r.status, r.stdout)
		}
	})
	t.Run("a link", func(t *testing.T) {
		dir := testenv.Dir(t)
		if err := os.Symlink(testenv.Dir(t), filepath.Join(dir, ".claude")); err != nil {
			t.Skip(err)
		}
		r := runTool(t, "skill", "--harness", "claude", "--dir", dir, "--json")
		if r.status != finding.ExitDestination || !slices.Equal(r.codes(), []string{string(finding.CodeDestinationLink)}) {
			t.Errorf("status %d, %q", r.status, r.codes())
		}
	})
	t.Run("a directory that is a file", func(t *testing.T) {
		f := filepath.Join(testenv.Dir(t), "f")
		os.WriteFile(f, nil, 0o644)
		r := runTool(t, "skill", "--harness", "claude", "--dir", f, "--json")
		if r.status != finding.ExitDestination || !slices.Equal(r.codes(), []string{string(finding.CodeDestinationNotDirectory)}) {
			t.Errorf("status %d, %q", r.status, r.codes())
		}
	})
	t.Run("--print with another flag", func(t *testing.T) {
		r := runTool(t, "skill", "--print", "--dry-run", "--json")
		if r.status != finding.ExitUsage || !slices.Equal(r.codes(), []string{string(finding.CodeUsageFlagExcluded)}) {
			t.Errorf("status %d, %q", r.status, r.codes())
		}
	})
}

// TestSkillPrintAndDryRun holds --print to the text metais skill writes,
// and --dry-run to writing nothing.
func TestSkillPrintAndDryRun(t *testing.T) {
	version.Override(t, "1.2.3")
	r := runTool(t, "skill", "--print", "--json")
	if d := readSkillDoc(t, r); d.Text == nil || *d.Text != string(skill.Text("1.2.3")) || d.Chosen != nil || len(d.Targets) != 0 {
		t.Errorf("--print --json: %s", r.doc.Result)
	}
	dir := testenv.Dir(t)
	os.Mkdir(filepath.Join(dir, ".git"), 0o755)
	r = runTool(t, "skill", "--harness", "codex", "--dir", dir, "--dry-run", "--json")
	if d := readSkillDoc(t, r); r.status != 0 || len(r.codes()) != 0 || d.Targets[0].Action != "would_install" {
		t.Errorf("--dry-run: status %d, %s", r.status, r.stdout)
	}
	if r.doc.Next[0].Run != "metais skill --harness codex --dir <dir>" {
		t.Errorf("next %v", r.doc.Next)
	}
	if entries, _ := os.ReadDir(dir); len(entries) != 1 {
		t.Errorf("--dry-run wrote %v", entries)
	}
}

// skillCommand is a metais command line in the skill's text: a code span
// that begins "metais ", which may wrap.
var skillCommand = regexp.MustCompile("`(metais [^`]+)`")

// TestSkillCommandsParse is ADR-008 A2 item 7's drift test: every metais
// command line the skill names parses under its command's declaration, so
// a renamed command or flag fails here until the skill says the new name.
func TestSkillCommandsParse(t *testing.T) {
	sub := strings.NewReplacer("<entry>", "guide.md", "<input>", "status", "<new dir>", "out",
		"<preview dir>", "preview", "<n>", "1", "<command>", "render", "<pattern>", "x", "<file>", "status.json")
	seen := map[string]bool{}
	for _, m := range skillCommand.FindAllStringSubmatch(string(skill.Source()), -1) {
		line := strings.Join(strings.Fields(m[1]), " ")
		line, _, _ = strings.Cut(line, " > ")
		if err := parsesAsLine(sub.Replace(line)); err != nil {
			t.Errorf("the skill's %q: %v", m[1], err)
		}
		seen[strings.Fields(line)[1]] = true
	}
	// The skill teaches both golden paths whole.
	for _, p := range paths {
		for _, c := range p.requires {
			if !seen[c] {
				t.Errorf("the skill names no metais %s line, a step of %s", c, p.Title)
			}
		}
	}
}
