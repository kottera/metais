// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"errors"
	"slices"
	"strings"

	"github.com/kottera/metais/internal/cli/args"
	"github.com/kottera/metais/internal/finding"
	"github.com/kottera/metais/internal/hostfs"
	"github.com/kottera/metais/internal/skill"
	"github.com/kottera/metais/internal/version"
)

// skillResult is metais.skill/1's result (ADR-008 A2 item 8): what was
// done to each target, or with --print the skill's text.
type skillResult struct {
	Skill   string        `json:"skill"`
	Version string        `json:"version"`
	Chosen  *string       `json:"chosen"`
	Targets []skillTarget `json:"targets"`
	Text    *string       `json:"text"`
}

type skillTarget struct {
	Path      string   `json:"path"`
	Harnesses []string `json:"harnesses"`
	Because   string   `json:"because"`
	Action    string   `json:"action"`
	From      *string  `json:"from"`
}

// The hints of the destination refusals metais skill shares with render,
// whose own hints speak of a render.
const (
	skillLinkHint     = "Remove the link, or write the skill with `metais skill --print` to where the link points. The other targets were decided on their own."
	skillNotDirHint   = "Remove what stands where the skill's directory goes, or name another directory with --dir."
	skillNotRegHint   = "Remove what stands at the skill's path; metais skill replaces only a regular file."
	skillWriteHint    = "Fix the cause, and run metais skill again."
	skillOpenNotFound = "Name an existing directory with --dir; metais skill creates only the skill's own directories beneath it."
)

// skillDeclaration is metais skill's declaration (ADR-008 A2).
func skillDeclaration() *Command {
	return &Command{
		Name: "skill", Group: "The tool",
		Synopsis:    "skill [--harness <name>]... [--dir <dir>] [--overwrite] [--dry-run] [--print] [--json]",
		Description: "Write metais's agent skill, which teaches a coding agent the golden paths, where the agent's harness reads skills.",
		Detail: "Writes SKILL.md, the skill a coding agent's harness loads before its first command, to " +
			".claude/skills/metais/ for Claude Code, or to .agents/skills/metais/ for Codex, Gemini CLI, " +
			"Cursor and OpenCode, beneath the directory --dir names: the current directory when it is " +
			"omitted. The skill's text is part of metais and versioned with it.\n\n" +
			"Without --harness, the harness is the one the directory shows: .claude/ or .agents/. When it " +
			"holds neither, or both, metais skill writes nothing and asks for --harness. Both directories " +
			"are written only when --harness names a harness of each.\n\n" +
			"A harness finds a project's skills from the directory it starts in up to the repository's top " +
			"level, so run metais skill there; a directory with no .git entry gets a warning. For every " +
			"project of your own, write --dir ~, with a space: a shell does not expand the ~ of --dir=~.\n\n" +
			"Each SKILL.md ends with a line naming the version that wrote it and the digest of the text " +
			"above it. Running metais skill again replaces a copy metais wrote that nobody edited, when its " +
			"version or text differs, and leaves an edited copy as it is unless --overwrite is given. " +
			"--dry-run reports what each target needs, so it says whether the skill is current.",
		Role: finding.RoleNoPackage,
		Flags: []args.Flag{
			{Name: "harness", Value: "<name>", Values: skill.Harnesses, Repeatable: true,
				Description: "The harness to write the skill for: claude writes .claude/skills, and codex, gemini, cursor and opencode write .agents/skills. Detected from the directory when omitted."},
			{Name: "dir", Value: "<dir>",
				Description: "The existing directory to write beneath, the current directory when omitted: a repository's top level, or ~ for your own scope, written --dir ~ with a space."},
			{Name: "overwrite", Description: "Replace a skill someone edited, or one metais did not write."},
			{Name: "dry-run", Description: "Report what each target needs, and write nothing."},
			{Name: "print", Description: "Print the skill to standard output, and write nothing; it takes no other flag but --json."},
		},
		Format: "metais.skill/1",
		Prints: "Each SKILL.md written, or that would be, with what was done to it and the harnesses that read it; with --print, the skill itself.",
		Refusals: []Refusal{
			{Code: finding.CodeUsageHarnessRequired, Cause: "No --harness is given, and the directory holds neither or both of .claude/ and .agents/.", Next: nextSkillHarness[0]},
			{Code: finding.CodeUsageUnexpandedTilde, Cause: "--dir's value begins with a literal ~, as --dir=~ gives it.", Next: nextSkillHome},
			{Code: finding.CodeUsageSkillEdited, Cause: "A SKILL.md metais did not write, or one edited since, stands at a target.", Next: nextSkillOverwrite},
			{Code: finding.CodeUsageFlagExcluded, Cause: "--print is given with another of metais skill's flags.", Next: nextCommandHelp("skill")},
			{Code: finding.CodeDestinationLink, Cause: "A symbolic link stands at, or above, a target beneath the directory.", Next: nextSkillPrint, Hint: skillLinkHint},
			{Code: finding.CodeDestinationNotDirectory, Cause: "The directory, or a directory a target needs, is not a directory.", Next: nextSkillAgain, Hint: skillNotDirHint},
			{Code: finding.CodeDestinationNotRegularFile, Cause: "A directory or special file stands at a target's SKILL.md.", Next: nextSkillAgain, Hint: skillNotRegHint},
			{Code: finding.CodeDestinationNothingWritten, Cause: "The directory cannot be opened, or a write failed before any target was written.", Next: nextSkillAgain, Hint: skillWriteHint},
			{Code: finding.CodeDestinationWriteFailed, Cause: "A write failed after another target was written, which is left in place.", Next: nextSkillAgain, Hint: skillWriteHint},
		},
		Examples: []Example{
			{"metais skill --harness claude", "Writes .claude/skills/metais/SKILL.md beneath the current directory, for Claude Code."},
			{"metais skill", "Writes the skill for the harness the current directory shows, .claude/ or .agents/."},
			{"metais skill --harness claude --harness codex", "Writes it for Claude Code and for the harnesses that read .agents/skills."},
			{"metais skill --harness claude --dir ~", "Writes it to ~/.claude/skills, for every project of yours."},
			{"metais skill --dry-run", "Reports whether the skill is current, and writes nothing."},
			{"metais skill --print", "Prints the skill, for a harness that reads skills from somewhere else."},
		},
		run: runSkill,
	}
}

// The steps metais skill points to. The directory is <dir> (ADR-008 D14
// item 2); skillNext adds it, and the --harness flags given, to a step.
var (
	nextSkillHarness = func() []Next {
		var out []Next
		for _, h := range skill.Harnesses {
			out = append(out, Next{Run: "metais skill --harness " + h, Why: "Writes the skill for " + harnessTitle(h) + "."})
		}
		return append(out, Next{Run: "metais skill --harness claude --harness codex",
			Why: "Writes the skill to both .claude/skills and .agents/skills."})
	}()
	nextSkillHome      = Next{Run: "metais skill --dir ~", Why: "Writes the skill beneath your home directory, which the shell expands when ~ stands alone."}
	nextSkillOverwrite = Next{Run: "metais skill --overwrite", Why: "Replaces the copy found with this version's skill."}
	nextSkillPrint     = Next{Run: "metais skill --print", Why: "Prints the skill, to compare or to write where you choose."}
	nextSkillAgain     = Next{Run: "metais skill", Why: "Writes the skill again once the cause is fixed."}
	nextSkillHelp      = Next{Run: "metais help", Why: "Lists the commands the skill teaches; a harness loads the skill at its next session."}
	nextSkillApply     = Next{Run: "metais skill", Why: "Writes what --dry-run reported."}
)

// harnessTitle is the product name of harness h.
func harnessTitle(h string) string {
	return map[string]string{"claude": "Claude Code", "codex": "Codex", "gemini": "Gemini CLI", "cursor": "Cursor", "opencode": "OpenCode"}[h]
}

// runSkill is metais skill (ADR-008 A2, ADR-009 A5).
func runSkill(s *session, p args.Parsed) outcome {
	v := version.String()
	if p.Has("print") {
		for _, other := range []string{"harness", "dir", "overwrite", "dry-run"} {
			if _, ok := p.Arg(other); ok {
				return s.refuseWith(s.excluded(p, other, "print"))
			}
		}
		text := string(skill.Text(v))
		return outcome{result: &skillResult{Skill: skill.Name, Version: v, Targets: []skillTarget{}, Text: &text}, human: []byte(text)}
	}
	dir, dirGiven := p.Value("dir")
	if !dirGiven {
		dir = "."
	}
	if dir == "~" || strings.HasPrefix(dir, "~/") {
		i, _ := p.Arg("dir")
		return s.refuseWith(finding.New(finding.CodeUsageUnexpandedTilde, invocation, nil,
			finding.Params{"argument": finding.Int(int64(s.line.at[i] + 1))}))
	}
	var harnesses []string
	for _, h := range p.Values("harness") {
		if !slices.Contains(harnesses, h) {
			harnesses = append(harnesses, h)
		}
	}
	step := func(n Next, extra string) Next {
		for _, h := range harnesses {
			n.Run += " --harness " + h
		}
		if dirGiven {
			n.Run += " --dir <dir>"
		}
		n.Run += extra
		return n
	}
	res, err := skill.Install(dir, skill.Options{Harnesses: harnesses, Overwrite: p.Has("overwrite"), DryRun: p.Has("dry-run"), Version: v})
	var need *skill.NeedHarnessError
	switch {
	case errors.As(err, &need):
		o := s.refuseWith(finding.New(finding.CodeUsageHarnessRequired, invocation, nil, finding.Params{"found": finding.String(need.Found)}))
		o.next = nil
		for _, n := range nextSkillHarness {
			if dirGiven {
				n.Run += " --dir <dir>"
			}
			o.next = append(o.next, n)
		}
		return o
	case err != nil:
		return s.skillOpenFailed(err, step(nextSkillAgain, ""))
	}
	var fs []finding.Finding
	var next []Next
	addNext := func(n Next) {
		if !slices.Contains(next, n) {
			next = append(next, n)
		}
	}
	var written []string
	result := &skillResult{Skill: skill.Name, Version: v, Chosen: &res.Chosen, Targets: []skillTarget{}}
	for _, t := range res.Targets {
		if t.Err != nil {
			fs = append(fs, skillTargetFinding(t, written))
			addNext(step(declaredCause(s.refusals(), fs[len(fs)-1]).Next, ""))
			continue
		}
		if t.Action == skill.Edited {
			fs = append(fs, finding.New(finding.CodeUsageSkillEdited, invocation, nil, finding.Params{"path": finding.String(t.Path)}))
			addNext(step(nextSkillOverwrite, ""))
			addNext(nextSkillPrint)
		}
		if t.Action == skill.Installed || t.Action == skill.Updated || t.Action == skill.Replaced {
			written = append(written, t.Path)
		}
		st := skillTarget{Path: t.Path, Harnesses: t.Harnesses, Because: t.Because, Action: string(t.Action)}
		if t.From != "" {
			from := t.From
			st.From = &from
		}
		result.Targets = append(result.Targets, st)
	}
	if !res.Repository {
		fs = append(fs, finding.New(finding.CodeUsageNotRepositoryRoot, invocation, nil, nil))
	}
	if len(next) == 0 {
		if slices.ContainsFunc(result.Targets, func(t skillTarget) bool { return strings.HasPrefix(t.Action, "would_") }) {
			addNext(step(nextSkillApply, ""))
		} else {
			addNext(nextSkillHelp)
		}
	}
	return outcome{findings: fs, result: result, human: skillHuman(p.Has("dry-run"), v, result), next: next}
}

// skillTargetFinding is the finding of t's refusal or failed write;
// written are the targets written before it.
func skillTargetFinding(t skill.Target, written []string) finding.Finding {
	path := finding.String(t.Path)
	switch r := hostfs.ReasonOf(t.Err); r {
	case hostfs.Link:
		return finding.New(finding.CodeDestinationLink, invocation, nil, finding.Params{"path": path}).WithHint(skillLinkHint)
	case hostfs.NotDirectory:
		return finding.New(finding.CodeDestinationNotDirectory, invocation, nil, finding.Params{"path": path}).WithHint(skillNotDirHint)
	case hostfs.IsDirectory, hostfs.NotRegularFile:
		return finding.New(finding.CodeDestinationNotRegularFile, invocation, nil, finding.Params{"path": path, "reason": finding.String(string(r))}).WithHint(skillNotRegHint)
	default:
		if len(written) == 0 {
			return nothingWritten(r).WithHint(skillWriteHint)
		}
		return finding.New(finding.CodeDestinationWriteFailed, invocation, nil, finding.Params{
			"reason":  finding.String(writeReason(r)),
			"written": stringList(written),
		}).WithHint(skillWriteHint)
	}
}

// skillOpenFailed is the outcome of a directory metais skill cannot open.
func (s *session) skillOpenFailed(err error, again Next) outcome {
	var f finding.Finding
	switch r := hostfs.ReasonOf(err); r {
	case hostfs.NotDirectory:
		f = finding.New(finding.CodeDestinationNotDirectory, invocation, nil, nil).WithHint(skillNotDirHint)
	case hostfs.NotFound:
		f = nothingWritten(r).WithHint(skillOpenNotFound)
	default:
		f = nothingWritten(r).WithHint(skillWriteHint)
	}
	return outcome{findings: []finding.Finding{f}, next: []Next{again}}
}

// skillHuman is metais skill's human result: each target, what was done
// to it, and the harnesses that read it.
func skillHuman(dryRun bool, v string, r *skillResult) []byte {
	if len(r.Targets) == 0 {
		return nil
	}
	head := "The metais skill, from metais " + v + ":\n"
	if dryRun {
		head = "The metais skill, from metais " + v + "; --dry-run wrote nothing:\n"
	}
	rows := make([][]string, 0, len(r.Targets))
	for _, t := range r.Targets {
		action := strings.ReplaceAll(t.Action, "_", " ")
		if t.From != nil && *t.From != v {
			action += " from " + *t.From
		}
		rows = append(rows, []string{"  " + t.Path, action, "for " + strings.Join(t.Harnesses, ", ") + ", because " + becauseText(t.Because)})
	}
	b := append([]byte(head), columns(rows)...)
	return append(b, "\nA harness loads the skill when its next session starts.\n"...)
}

// becauseText is a target's because as human output says it.
func becauseText(because string) string {
	if because == "flag" {
		return "--harness names it"
	}
	return because
}
