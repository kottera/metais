// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

// Package score is the UX runner's scorer (PLAN-2026-10-metais-ux W1-2).
// It reads one run, its record, its stream-json transcript and the
// session's directory as the session left it, and computes R-UX-5's pass
// rules and the scenario's own outcome check, each with the evidence it
// rests on: the command, the turn and line of the transcript, the file.
//
// It is handed everything it reads as an fs.FS, the runner opening the
// run's directory and each fixture's source for it, and refers to no host
// symbol, so it runs in the gate over committed fixtures with none of the
// runner's allowance (ADR-009 Amendment A8 item 4). It starts nothing and
// reaches no network: every rule and check is code over what it is given,
// so the same run is scored the same way every time.
package score

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"path"
	"slices"
	"strings"

	"github.com/kottera/metais/internal/tools/ux/scenario"
	"github.com/kottera/metais/internal/tools/ux/streamjson"
)

// Format is an outcome's format. A change to what an outcome means
// changes it.
const Format = "metais-ux-outcome/1"

// The pass rules of R-UX-5, by the names an outcome reports them under.
const (
	// RuleFirstInspect: the first metais command of a consuming scenario
	// is inspect.
	RuleFirstInspect = "first-metais-is-inspect"
	// RuleNoZip: no zip extraction, by a zip tool or by parsing the
	// archive itself.
	RuleNoZip = "no-zip-extraction"
	// RuleNoUnknown: no unknown-command error from metais.
	RuleNoUnknown = "no-unknown-command"
	// RuleUntouched: R-UX-8, the package left alone by a scenario whose
	// prompt asks for output.
	RuleUntouched = "package-untouched"
	// RuleOutcome: the scenario's own outcome check.
	RuleOutcome = "outcome"
)

// Outcome is a run's score: what the runner writes into its record's
// outcome.
type Outcome struct {
	Format   string `json:"format"`
	Scenario string `json:"scenario"`
	// Check names the scenario's outcome check.
	Check string `json:"check"`
	// Scored is false for a run that is not scored: an invalid run, or one
	// that did not finish. NotScored says why.
	Scored    bool   `json:"scored"`
	NotScored string `json:"not_scored,omitempty"`
	// Pass is true when every rule that applies passes.
	Pass  bool     `json:"pass"`
	Rules []Result `json:"rules"`
}

// Result is one rule's result.
type Result struct {
	Rule string `json:"rule"`
	// Applies is false for a rule the scenario is not held to: the first
	// command in an authoring scenario, R-UX-8 where the prompt asks for
	// no output.
	Applies bool `json:"applies"`
	Pass    bool `json:"pass"`
	// Detail says why, in a sentence.
	Detail   string     `json:"detail"`
	Evidence []Evidence `json:"evidence,omitempty"`
}

// Evidence is what a result rests on: a tool call of the transcript, or a
// file of the run's directory, or both.
type Evidence struct {
	// Turn is the session's turn, Line the transcript's line, both from 1.
	Turn int `json:"turn,omitempty"`
	Line int `json:"line,omitempty"`
	// Tool is the tool called, and Command what it ran or what it named.
	Tool    string `json:"tool,omitempty"`
	Command string `json:"command,omitempty"`
	// File is a path of the run's directory, such as work/out/report.md.
	File string `json:"file,omitempty"`
	// Text is the session's own words the result rests on.
	Text string `json:"text,omitempty"`
}

// Failing names the rules that apply and fail.
func (o *Outcome) Failing() []string {
	var out []string
	for _, r := range o.Rules {
		if r.Applies && !r.Pass {
			name := r.Rule
			if r.Rule == RuleOutcome {
				name += " (" + o.Check + ")"
			}
			out = append(out, name)
		}
	}
	return out
}

// Input is one run, as the scorer is handed it.
type Input struct {
	// Scenario is the scenario the run ran, as the set declares it.
	Scenario *scenario.Scenario
	// Run is the run's directory: record.json, the transcript, and the
	// session's work/ and tmp/ as it left them.
	Run fs.FS
	// Sources are the source directories of the scenario's fixtures, by
	// the file each is packed as.
	Sources map[string]fs.FS
}

// record is what the scorer reads of a run's record.json.
type record struct {
	Format   string `json:"format"`
	Scenario struct {
		ID            string `json:"id"`
		Kind          string `json:"kind"`
		AsksForOutput bool   `json:"asks_for_output"`
	} `json:"scenario"`
	Status     string `json:"status"`
	Valid      bool   `json:"valid"`
	Invocation struct {
		Dir string `json:"dir"`
	} `json:"invocation"`
	Fixtures   []fileEntry `json:"fixtures"`
	Files      []fileEntry `json:"files"`
	Transcript string      `json:"transcript"`
	Checks     []struct {
		Name   string `json:"name"`
		Pass   bool   `json:"pass"`
		Detail string `json:"detail"`
	} `json:"checks"`
}

type fileEntry struct {
	Path   string `json:"path"`
	Size   int64  `json:"size"`
	SHA256 string `json:"sha256"`
	Change string `json:"change"`
}

// rule is one pass rule: whether a run is held to it, and its result.
type rule struct {
	name    string
	applies func(r *run) bool
	check   func(r *run) Result
}

// rules are R-UX-5's, in the order an outcome reports them. The outcome
// check is last.
var rules = []rule{
	{RuleFirstInspect, func(r *run) bool { return r.scenario.Kind == scenario.Consuming }, firstInspect},
	{RuleNoZip, always, noZip},
	{RuleNoUnknown, always, noUnknown},
	{RuleUntouched, func(r *run) bool { return r.scenario.Kind == scenario.Consuming && r.scenario.AsksForOutput }, untouched},
	{RuleOutcome, always, outcome},
}

func always(*run) bool { return true }

// Score scores one run. A run the runner recorded invalid, or one that
// did not run to its end, is not scored: its outcome says why, and no
// rule is computed. An error is a run the scorer cannot read at all.
func Score(in Input) (*Outcome, error) {
	r, err := load(in)
	if err != nil {
		return nil, err
	}
	o := &Outcome{Format: Format, Scenario: in.Scenario.Name(), Check: r.checkName}
	switch {
	case !r.rec.Valid:
		o.NotScored = "the run is invalid: " + strings.Join(r.failedChecks(), "; ")
		return o, nil
	case r.rec.Status != "complete":
		o.NotScored = "the run did not complete: its status is " + r.rec.Status
		return o, nil
	}
	o.Scored, o.Pass = true, true
	for _, ru := range rules {
		res := Result{Rule: ru.name}
		if ru.applies(r) {
			res = ru.check(r)
			res.Rule, res.Applies = ru.name, true
		} else {
			res.Pass, res.Detail = true, "does not apply to "+r.scenario.Name()
		}
		if res.Applies && !res.Pass {
			o.Pass = false
		}
		o.Rules = append(o.Rules, res)
	}
	return o, nil
}

// run is one run, read, with the facts every rule draws on.
type run struct {
	scenario  *scenario.Scenario
	checkName string
	rec       record
	t         *streamjson.Transcript
	fsys      fs.FS
	// runDir is the run's directory as the session saw it: the parent of
	// the session's work directory.
	runDir  string
	sources map[string]*source
	// cmds are every simple command of every Bash call, in order, and
	// calls those that ran metais.
	cmds  []*cmd
	calls []*call
	// writes are the session's Write and Edit tool calls.
	writes []*write
}

func load(in Input) (*run, error) {
	if in.Scenario == nil || in.Run == nil {
		return nil, errors.New("score: no scenario or no run")
	}
	r := &run{scenario: in.Scenario, fsys: in.Run}
	b, err := fs.ReadFile(in.Run, "record.json")
	if err != nil {
		return nil, fmt.Errorf("score: %w", err)
	}
	if err := json.Unmarshal(b, &r.rec); err != nil {
		return nil, fmt.Errorf("score: record.json: %w", err)
	}
	if r.rec.Format != "metais-ux-run/1" {
		return nil, fmt.Errorf("score: record.json: format %q", r.rec.Format)
	}
	if r.rec.Scenario.ID != in.Scenario.ID {
		return nil, fmt.Errorf("score: the run is of scenario %s, not %s", r.rec.Scenario.ID, in.Scenario.ID)
	}
	if r.checkName, err = CheckName(in.Scenario); err != nil {
		return nil, err
	}
	name := r.rec.Transcript
	if name == "" {
		name = "transcript.jsonl"
	}
	tb, err := fs.ReadFile(in.Run, name)
	if err != nil && r.rec.Status != "not-run" {
		return nil, fmt.Errorf("score: %w", err)
	}
	r.t = streamjson.Read(tb)
	r.runDir = path.Dir(r.rec.Invocation.Dir)
	r.sources = map[string]*source{}
	for as, src := range in.Sources {
		s, err := readSource(src)
		if err != nil {
			return nil, fmt.Errorf("score: the source of %s: %w", as, err)
		}
		r.sources[as] = s
	}
	r.read()
	return r, nil
}

func (r *run) failedChecks() []string {
	var out []string
	for _, c := range r.rec.Checks {
		if !c.Pass {
			out = append(out, c.Name+": "+c.Detail)
		}
	}
	if len(out) == 0 {
		out = append(out, "status "+r.rec.Status)
	}
	return out
}

// CheckName is the outcome check a scenario's declaration names: its
// check, a JSON string naming one of the scorer's checks.
func CheckName(s *scenario.Scenario) (string, error) {
	var name string
	if len(s.Check) == 0 || json.Unmarshal(s.Check, &name) != nil {
		return "", fmt.Errorf("score: %s declares no check: its check must name one of %s", s.Name(), strings.Join(CheckNames(), ", "))
	}
	if _, ok := checks[name]; !ok {
		return "", fmt.Errorf("score: %s names check %q, which is none of %s", s.Name(), name, strings.Join(CheckNames(), ", "))
	}
	return name, nil
}

// CheckNames are the outcome checks a scenario may name.
func CheckNames() []string {
	var names []string
	for n := range checks {
		names = append(names, n)
	}
	slices.Sort(names)
	return names
}

// cmd is one simple command of a Bash call, with the directory it ran in.
type cmd struct {
	use  *streamjson.ToolUse
	s    simple
	name string
	rest []string
	// cwd is the directory it ran in, relative to the run's directory, or
	// "" when that is outside it.
	cwd string
}

// call is a command that ran metais: its subcommand and the words after.
type call struct {
	*cmd
	sub  string
	args []string
}

// write is a Write or Edit tool call: the file it wrote, relative to the
// run's directory ("" outside it), and the text it wrote.
type write struct {
	use  *streamjson.ToolUse
	file string
	text string
}

func (c *cmd) ok() bool { return c.use.Result != nil && !c.use.Result.IsError }

func (c *cmd) output() string {
	if c.use.Result == nil {
		return ""
	}
	return c.use.Result.Text
}

func (c *cmd) evidence() Evidence {
	return Evidence{Turn: c.use.Turn, Line: c.use.Line, Tool: c.use.Name, Command: clip(c.use.Command())}
}

func useEvidence(u *streamjson.ToolUse, file string) Evidence {
	e := Evidence{Turn: u.Turn, Line: u.Line, Tool: u.Name, File: file}
	if c := u.Command(); c != "" {
		e.Command = clip(c)
	} else if p := u.Field("file_path"); p != "" {
		e.Command = p
	}
	return e
}

// clip shortens a command for an evidence line.
func clip(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		s = s[:i] + " …"
	}
	if len(s) > 240 {
		s = s[:240] + " …"
	}
	return s
}

// read draws the facts from the transcript: each command and where it ran,
// each metais call, each write.
func (r *run) read() {
	cwd := "work"
	for i := range r.t.ToolUses {
		u := &r.t.ToolUses[i]
		switch u.Name {
		case "Bash":
			for _, s := range parseShell(u.Command()) {
				name, rest := s.name()
				c := &cmd{use: u, s: s, name: name, rest: rest, cwd: cwd}
				r.cmds = append(r.cmds, c)
				switch name {
				case "cd":
					to := "~"
					if len(rest) > 0 {
						to = rest[0]
					}
					cwd = r.resolve(to, cwd)
				case "metais":
					sub, args := subcommand(rest)
					r.calls = append(r.calls, &call{cmd: c, sub: sub, args: args})
				}
			}
			// The harness keeps the shell's directory between calls only
			// within the session's own.
			if cwd != "work" && !strings.HasPrefix(cwd, "work/") {
				cwd = "work"
			}
		case "Write", "Edit", "MultiEdit", "NotebookEdit":
			w := &write{use: u, file: r.resolve(u.Field("file_path"), "work")}
			w.text = u.Field("content") + u.Field("new_string")
			if u.Name == "MultiEdit" {
				var in struct {
					Edits []struct {
						New string `json:"new_string"`
					} `json:"edits"`
				}
				_ = json.Unmarshal(u.Input, &in)
				for _, e := range in.Edits {
					w.text += e.New
				}
			}
			r.writes = append(r.writes, w)
		}
	}
}

// subcommand is a metais command line's command, after the global flags,
// and the words after it. --version and --help alone are version and help.
func subcommand(words []string) (string, []string) {
	for i := 0; i < len(words); i++ {
		w := words[i]
		switch {
		case w == "--color":
			i++
		case w == "--version":
			return "version", words[i+1:]
		case w == "-h" || w == "--help":
			if i == len(words)-1 {
				return "help", nil
			}
		case strings.HasPrefix(w, "-"):
		default:
			return w, words[i+1:]
		}
	}
	return "", nil
}

// resolve is p, a path as a command named it in directory cwd, relative
// to the run's directory, or "" when it is outside it.
func (r *run) resolve(p, cwd string) string {
	switch {
	case p == "":
		return ""
	case p == "~" || strings.HasPrefix(p, "~/"):
		p = path.Join("home", strings.TrimPrefix(strings.TrimPrefix(p, "~"), "/"))
	case strings.HasPrefix(p, "$HOME"):
		p = path.Join("home", strings.TrimPrefix(p, "$HOME"))
	case path.IsAbs(p):
		if r.runDir == "" || r.runDir == "." {
			return ""
		}
		if p == r.runDir {
			return "."
		}
		rel, ok := strings.CutPrefix(path.Clean(p), r.runDir+"/")
		if !ok {
			return ""
		}
		p = rel
	default:
		if cwd == "" {
			return ""
		}
		p = path.Join(cwd, p)
	}
	p = path.Clean(p)
	if p == ".." || strings.HasPrefix(p, "../") {
		return ""
	}
	return p
}

// flag is the value of a command's flag: --name value or --name=value,
// every one given.
func flagValues(words []string, name string) []string {
	var out []string
	for i, w := range words {
		if w == name && i+1 < len(words) {
			out = append(out, words[i+1])
		} else if v, ok := strings.CutPrefix(w, name+"="); ok {
			out = append(out, v)
		}
	}
	return out
}

func hasFlag(words []string, name string) bool {
	return slices.Contains(words, name) || slices.ContainsFunc(words, func(w string) bool { return strings.HasPrefix(w, name+"=") })
}

// texts are the session's own words: each text part it wrote, and its
// final result.
func (r *run) texts() []streamjson.Text {
	out := slices.Clone(r.t.Texts)
	if r.t.Result.Found && r.t.Result.Text != "" {
		out = append(out, streamjson.Text{Line: r.t.Result.Line, Text: r.t.Result.Text})
	}
	return out
}

// final is the session's last word to the user: its result's text.
func (r *run) final() streamjson.Text {
	return streamjson.Text{Line: r.t.Result.Line, Text: r.t.Result.Text}
}

// file reads a file of the run's directory.
func (r *run) file(p string) ([]byte, bool) {
	if p == "" {
		return nil, false
	}
	b, err := fs.ReadFile(r.fsys, p)
	return b, err == nil
}

// filesUnder lists the regular files of the run's directory beneath dir.
func (r *run) filesUnder(dir string) []string {
	var out []string
	_ = fs.WalkDir(r.fsys, dir, func(p string, d fs.DirEntry, err error) error {
		if err == nil && d.Type().IsRegular() {
			out = append(out, p)
		}
		return nil
	})
	return out
}

// fixtureNames are the files the session's directory started with, as
// the record lists them.
func (r *run) fixtureNames() []string {
	var out []string
	for _, f := range r.rec.Fixtures {
		out = append(out, f.Path)
	}
	return out
}
