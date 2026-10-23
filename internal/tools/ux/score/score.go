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

	"metais.org/metais/internal/tools/ux/scenario"
	"metais.org/metais/internal/tools/ux/streamjson"
)

// Format is an outcome's format. A change to what an outcome means
// changes it.
const Format = "metais-ux-outcome/1"

// The pass rules of R-UX-5, and the install rule of ADR-013 D9, by the
// names an outcome reports them under.
const (
	// RuleGoldenInstall: an install scenario that gates (I1) installs
	// metais by a golden path of ADR-013 D6 before it runs it.
	RuleGoldenInstall = "golden-install"
	// RuleInspectFirst: a consuming session reads nothing of the package's
	// contents before metais inspect.
	RuleInspectFirst = "inspect-first"
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
	// For a check that reads the session's words (Judged), Verdict is the
	// recorded verdict the outcome rule follows, Phrase the phrase check's
	// own result beside it, and Disagree whether the two differ. A judged
	// check with no verdict follows the phrase check, and NeedsVerdict
	// says so.
	Judged       bool     `json:"judged,omitempty"`
	Verdict      *Verdict `json:"verdict,omitempty"`
	Phrase       *Result  `json:"phrase,omitempty"`
	Disagree     bool     `json:"disagree,omitempty"`
	NeedsVerdict bool     `json:"needs_verdict,omitempty"`
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
	// Notes are what the scorecard records beside a result without
	// failing it, such as a listing of the archive.
	Notes []Evidence `json:"notes,omitempty"`
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

// Verdict is a judgement of a run recorded in its record, from a judge
// model or a person, for an outcome check that reads the session's words
// (Ed's ruling, 2026-10-21). It is optional: the runner never writes one,
// and Wave 3 records one for every C0 and C1 run before its scorecard
// counts towards R-UX-5's thresholds.
type Verdict struct {
	// Judge names who judged it: a model ID, or a person.
	Judge string `json:"judge"`
	// Verdict is pass or fail.
	Verdict string `json:"verdict"`
	// Rationale says why, in a sentence or two.
	Rationale string `json:"rationale"`
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
	Verdict    *Verdict    `json:"verdict"`
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

// rule is one pass rule: whether a run is held to it, its result, and
// what it cannot see, which a scorecard states, or "".
type rule struct {
	name    string
	applies func(r *run) bool
	check   func(r *run) Result
	limits  string
}

// ownRenderLimits is what no-zip-extraction and package-untouched cannot
// see of an archive the session rendered (W3-7).
const ownRenderLimits = "The archive the rule measures is the package, or any other file the session started with, never the session's own rendered output: a zip tool or zip library that reads a file a metais render of the session wrote before it, such as unzip -p out/report.docx, extracts nothing of the package and is not counted. A file is one a render wrote when it lies in the render's output directory, or is the file its output was redirected to, and the run's directory holds it at the end as added, with bytes that are no fixture's. A command that also names a .metais file or a file the session started with is counted. A rendered file the session deleted before the run ended is not seen as rendered, so reading it is counted; one it overwrote by other means with bytes no fixture has is still read as rendered."

// rules are R-UX-5's, after D9's install rule, in the order an outcome
// reports them. The outcome check is last.
var rules = []rule{
	{RuleGoldenInstall, func(r *run) bool { return r.scenario.Install && r.scenario.Kind == scenario.Consuming }, goldenInstall, installLimits},
	{RuleInspectFirst, func(r *run) bool { return r.scenario.Kind == scenario.Consuming }, inspectFirst, ""},
	{RuleNoZip, always, noZip, ownRenderLimits},
	{RuleNoUnknown, always, noUnknown, ""},
	{RuleUntouched, func(r *run) bool { return r.scenario.Kind == scenario.Consuming && r.scenario.AsksForOutput }, untouched, ownRenderLimits},
	{RuleOutcome, always, outcome, ""},
}

// RuleNames are the pass rules, in the order an outcome reports them.
func RuleNames() []string {
	var names []string
	for _, ru := range rules {
		names = append(names, ru.name)
	}
	return names
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
			if ru.name == RuleOutcome && checks[r.checkName].judged {
				res = o.judge(r.rec.Verdict, res)
			}
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

// judge applies a recorded verdict to a judged check's phrase result.
func (o *Outcome) judge(v *Verdict, phrase Result) Result {
	o.Judged = true
	if v == nil || v.Verdict != "pass" && v.Verdict != "fail" || v.Judge == "" {
		o.NeedsVerdict = true
		return phrase
	}
	o.Verdict, o.Phrase = v, &phrase
	pass := v.Verdict == "pass"
	o.Disagree = pass != phrase.Pass
	res := Result{Rule: phrase.Rule, Applies: true, Pass: pass, Evidence: phrase.Evidence, Notes: phrase.Notes,
		Detail: fmt.Sprintf("judged %s by %s: %s (the phrase check: %s)", v.Verdict, v.Judge, v.Rationale, phrase.Detail)}
	return res
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
	// seq is its place among the session's commands and writes.
	seq int
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
	seq  int
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
			// A variable lives as long as the call's shell.
			vars := map[string]string{}
			start := cwd
			for _, s := range unroll(parseShell(u.Command())) {
				// $PWD is the directory the command runs in.
				if cwd != "" && path.IsAbs(r.runDir) {
					vars["PWD"] = path.Join(r.runDir, cwd)
				} else {
					delete(vars, "PWD")
				}
				if as, ok := s.assignments(); ok {
					// A value made by a command substitution is not known: the
					// parser ends the assignment where the substitution begins.
					for k, v := range as {
						if v != "" {
							vars[k] = simple{words: []string{v}}.expand(vars).words[0]
						}
					}
					continue
				}
				s = s.expand(vars)
				name, rest := s.name()
				c := &cmd{use: u, s: s, name: name, rest: rest, cwd: cwd, seq: len(r.cmds) + len(r.writes)}
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
			// The harness records the shell's directory for the next call
			// with pwd -P after the call's command line and &&, so a call
			// that exits other than 0 leaves the directory where it found
			// it, whatever it changed to within the call.
			if exitedNonZero(u) {
				cwd = start
			}
			// The harness keeps the shell's directory between calls only
			// within the session's own.
			if cwd != "work" && !strings.HasPrefix(cwd, "work/") {
				cwd = "work"
			}
		case "Write", "Edit", "MultiEdit", "NotebookEdit":
			w := &write{use: u, file: r.resolve(u.Field("file_path"), "work"), seq: len(r.cmds) + len(r.writes)}
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

// exitedNonZero reports whether a tool call's command exited other than
// 0, as its result shows: the harness reports it as an error, or reads its
// status as something else, such as grep's "No matches found".
func exitedNonZero(u *streamjson.ToolUse) bool {
	return u.Result != nil && (u.Result.IsError || u.Result.Interpreted != "")
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
