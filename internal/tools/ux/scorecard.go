// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"metais.org/metais/internal/tools/ux/scenario"
	"metais.org/metais/internal/tools/ux/score"
	"metais.org/metais/internal/tools/ux/scrub"
)

// scored is one run of a scorecard: its record and its outcome, or nil
// when it has no record yet.
type scored struct {
	scenario *scenario.Scenario
	model    string
	rec      *runRecord
	outcome  *score.Outcome
}

// scoreSet scores every recorded run of the set in out, writes each
// outcome into its record, and returns them by scenario and model.
func (c *config) scoreSet(set []*scenario.Scenario) ([]scored, error) {
	var runs []scored
	for _, s := range set {
		sources := map[string]fs.FS{}
		for _, f := range s.Fixtures {
			if !f.Package() {
				continue
			}
			dir := filepath.Join(c.root, "examples", f.Example)
			if f.Source != "" {
				dir = filepath.Join(c.root, scenariosDir, s.Dir, f.Source)
			}
			sources[f.As] = os.DirFS(dir)
		}
		for _, m := range c.models {
			dir := filepath.Join(c.out, "runs", s.Name(), m)
			r, err := readRecord(dir)
			if err != nil {
				return nil, err
			}
			sc := scored{scenario: s, model: m, rec: r}
			if finished(r) {
				if sc.outcome, err = score.Score(score.Input{Scenario: s, Run: os.DirFS(dir), Sources: sources}); err != nil {
					return nil, fmt.Errorf("%s %s: %w", s.Name(), m, err)
				}
				if r.Outcome, err = json.Marshal(sc.outcome); err != nil {
					return nil, err
				}
				if err := writeJSON(filepath.Join(dir, recordFile), r); err != nil {
					return nil, err
				}
			}
			runs = append(runs, sc)
		}
	}
	return runs, nil
}

// gatedModels are the models R-UX-5 gates authoring scenarios on. Haiku's
// authoring results are recorded, not gated.
var gatedAuthoring = []string{"opus", "sonnet"}

// gated reports whether a run gates: R-UX-5's thresholds count it, or,
// for an install scenario, ADR-013 D9's gate does. D9 gates I1, the
// consuming install scenario, on every model, and I2, the authoring one,
// on none: it is evidence of what a model and a search engine know on
// the day.
func gated(s *scenario.Scenario, model string) bool {
	if s.Install {
		return s.Kind == scenario.Consuming
	}
	return s.Kind == scenario.Consuming || slices.Contains(gatedAuthoring, model)
}

// scorecard prints the scorecard of the set's runs, as markdown, the form
// a scorecard is published in (plan §2).
func (c *config) scorecard(w io.Writer, set []*scenario.Scenario, runs []scored) error {
	var st setRecord
	if err := readJSON(filepath.Join(c.out, setFile), &st); err != nil {
		return err
	}
	p := func(format string, args ...any) { fmt.Fprintf(w, format, args...) }
	p("# METAIS UX scorecard\n\n")
	commit := st.Commit
	if st.Dirty {
		commit += " (dirty: the checkout had uncommitted changes)"
	}
	host := c.host
	if host == "" {
		host = "not stated (give -host)"
	}
	transcripts := c.transcripts
	if transcripts == "" {
		transcripts = "the set's runs/ directory, each run's transcript.jsonl (not stated: give -transcripts to say where they are kept)"
	}
	skill := "not installed (-with-skill off)"
	if st.WithSkill {
		skill = "installed with metais skill before each session (-with-skill)"
	}
	p("| | |\n|---|---|\n")
	p("| Commit | `%s` |\n", commit)
	p("| metais | %s, sha256 `%s` |\n", oneLine(st.Metais.Version), st.Metais.SHA256)
	p("| Harness | %s %s |\n", st.Harness.Name, st.Harness.Version)
	p("| Models | %s |\n", modelIDs(c.models, runs))
	p("| Host | %s |\n", host)
	p("| Transcripts | %s |\n", transcripts)
	p("| Skill | %s |\n", skill)
	p("| Set started | %s |\n", st.Started)
	p("| Scorer | %s |\n\n", score.Format)

	sections := []struct {
		heading string
		in      func(s *scenario.Scenario) bool
	}{
		{"## Consuming scenarios\n\nGated on every model (R-UX-5).\n\n",
			func(s *scenario.Scenario) bool { return !s.Install && s.Kind == scenario.Consuming }},
		{fmt.Sprintf("## Authoring scenarios\n\nGated on %s; Haiku's results are recorded, not gated (R-UX-5).\n\n", strings.Join(gatedAuthoring, " and ")),
			func(s *scenario.Scenario) bool { return !s.Install && s.Kind == scenario.Authoring }},
		{"## Install scenarios\n\nEach session starts with no metais on PATH (ADR-013 D9). I1 gates on every model. I2 is evidence, not a gate: it records whether a session finds METAIS and installs it, and its results are recorded.\n\n",
			func(s *scenario.Scenario) bool { return s.Install }},
	}
	for _, sec := range sections {
		if !slices.ContainsFunc(set, sec.in) {
			continue
		}
		p("%s", sec.heading)
		p("| Scenario | %s |\n|---|%s\n", strings.Join(c.models, " | "), strings.Repeat("---|", len(c.models)))
		for _, s := range set {
			if !sec.in(s) {
				continue
			}
			p("| %s |", s.Name())
			for _, r := range runs {
				if r.scenario == s {
					p(" %s |", cell(r))
				}
			}
			p("\n")
		}
		p("\n")
	}

	// R-UX-5's thresholds count the runs of every scenario but the install
	// scenarios, and D9's gate the runs of the install scenario it gates.
	type tally struct{ pass, total, invalid, pending int }
	var ux, d9 tally
	for _, r := range runs {
		t := &ux
		if r.scenario.Install {
			if !gated(r.scenario, r.model) {
				continue
			}
			t = &d9
		}
		switch {
		case r.outcome == nil:
			t.pending++
		case !r.outcome.Scored:
			t.invalid++
		case gated(r.scenario, r.model):
			t.total++
			if r.outcome.Pass {
				t.pass++
			}
		}
	}
	holds := func(t tally) bool { return t.pass == t.total && t.invalid == 0 && t.pending == 0 }
	p("## Thresholds\n\n")
	if slices.ContainsFunc(set, func(s *scenario.Scenario) bool { return !s.Install }) {
		p("%d of %d gated runs pass. %d runs are invalid and not scored; %d are not yet run. ", ux.pass, ux.total, ux.invalid, ux.pending)
		p("R-UX-5's thresholds %s.\n\n", map[bool]string{true: "hold", false: "do not hold"}[holds(ux)])
	}
	if slices.ContainsFunc(set, func(s *scenario.Scenario) bool { return s.Install }) {
		p("Install scenarios: %d of %d gated runs pass. %d runs are invalid and not scored; %d are not yet run. ", d9.pass, d9.total, d9.invalid, d9.pending)
		p("ADR-013 D9's gate %s. I2's runs are recorded, not gated.\n\n", map[bool]string{true: "holds", false: "does not hold"}[holds(d9) && d9.total > 0])
	}

	p("## Why each run failed\n\n")
	failed := false
	for _, r := range runs {
		if r.outcome == nil || r.outcome.Pass && r.outcome.Scored {
			continue
		}
		failed = true
		// A scorecard is published: the run's evidence names no host path
		// and no credential.
		sc := scrub.New(filepath.Join(c.out, "runs", r.scenario.Name(), r.model), c.out)
		p := func(format string, args ...any) { w.Write(sc.Bytes(fmt.Appendf(nil, format, args...))) }
		if !r.outcome.Scored {
			p("- **%s %s** — invalid, not scored: %s\n", r.scenario.Name(), r.model, r.outcome.NotScored)
			continue
		}
		p("- **%s %s**%s\n", r.scenario.Name(), r.model, map[bool]string{true: "", false: " (recorded, not gated)"}[gated(r.scenario, r.model)])
		for _, res := range r.outcome.Rules {
			if !res.Applies || res.Pass {
				continue
			}
			name := res.Rule
			if res.Rule == score.RuleOutcome {
				name += " (" + r.outcome.Check + ")"
			}
			p("  - `%s`: %s", name, res.Detail)
			if len(res.Evidence) > 0 {
				p(" — %s", evidence(res.Evidence[0]))
			}
			p("\n")
		}
	}
	if !failed {
		p("Every run passed.\n")
	}
	p("\n")

	p("## Notes\n\n")
	noted := false
	for _, r := range runs {
		if r.outcome == nil || !r.outcome.Scored {
			continue
		}
		sc := scrub.New(filepath.Join(c.out, "runs", r.scenario.Name(), r.model), c.out)
		for _, res := range r.outcome.Rules {
			for _, n := range res.Notes {
				noted = true
				w.Write(sc.Bytes(fmt.Appendf(nil, "- **%s %s** `%s`: %s — %s\n", r.scenario.Name(), r.model, res.Rule, n.Text, evidence(n))))
			}
		}
	}
	if !noted {
		p("None.\n")
	}
	p("\n")

	p("## Recorded verdicts\n\n")
	p("C0's, C1's and I1's checks read the session's words by phrase. A run may carry a verdict, from a judge model or a person, which then decides the check. Before a scorecard counts towards R-UX-5's thresholds or ADR-013 D9's gate, every C0, C1 and I1 run carries one.\n\n")
	var judged, verdicts int
	var needs, disagree []string
	for _, r := range runs {
		if r.outcome == nil || !r.outcome.Judged {
			continue
		}
		judged++
		name := r.scenario.Name() + " " + r.model
		switch {
		case r.outcome.NeedsVerdict:
			needs = append(needs, name)
		case r.outcome.Disagree:
			verdicts++
			v := r.outcome.Verdict
			disagree = append(disagree, fmt.Sprintf("- **%s**: the phrase check says %s (%s); %s judged it %s: %s", name,
				map[bool]string{true: "pass", false: "fail"}[r.outcome.Phrase.Pass], r.outcome.Phrase.Detail, v.Judge, v.Verdict, v.Rationale))
		default:
			verdicts++
		}
	}
	p("%d of %d judged runs carry a verdict.", verdicts, judged)
	if len(needs) > 0 {
		p(" Without one: %s.", strings.Join(needs, ", "))
	}
	p("\n\n")
	if len(disagree) > 0 {
		p("Where the phrase check and the verdict disagree:\n\n%s\n\n", strings.Join(disagree, "\n"))
	} else {
		p("The phrase check and the verdict disagree on no run.\n\n")
	}

	p("## What the outcome checks cannot see\n\n")
	p("Every rule and check is decided by code over the transcript and the run's directory, with no model call. Where the table asks for a judgement, the check is narrower than the judgement, and says so here.\n\n")
	for _, s := range set {
		name, err := score.CheckName(s)
		if err != nil {
			return err
		}
		p("- **%s** (`%s`): %s\n", s.Name(), name, score.CheckLimits(name))
	}
	for _, name := range score.RuleNames() {
		if l := score.CheckLimits(name); l != "" {
			p("- **The pass rule `%s`**: %s\n", name, l)
		}
	}
	p("\n")

	p("## Prompts reworded under R-UX-9\n\n")
	reworded := false
	for _, s := range set {
		for _, rw := range s.Rewordings {
			reworded = true
			p("- **%s**, %s: was %q, is %q. The ambiguity: %s. The transcript that showed it: %s.\n", s.Name(), rw.Date, rw.Was, s.Prompt, rw.Ambiguity, rw.Transcript)
		}
	}
	for _, r := range runs {
		if r.rec != nil && r.rec.Scenario.Prompt != r.scenario.Prompt {
			reworded = true
			p("- **%s %s** ran the prompt %q, which is not the set's %q.\n", r.scenario.Name(), r.model, r.rec.Scenario.Prompt, r.scenario.Prompt)
		}
	}
	if !reworded {
		p("None: every run ran its scenario's prompt as the set holds it.\n")
	}
	return nil
}

// cell is a run's entry in a scorecard's table.
func cell(r scored) string {
	switch {
	case r.outcome == nil:
		return "not run"
	case !r.outcome.Scored:
		return "invalid"
	case r.outcome.Pass:
		if !gated(r.scenario, r.model) {
			return "pass (recorded)"
		}
		return "pass"
	}
	s := "fail: " + strings.Join(r.outcome.Failing(), ", ")
	if !gated(r.scenario, r.model) {
		s += " (recorded)"
	}
	return s
}

// modelIDs names each model as requested and the model IDs its sessions
// reported.
func modelIDs(models []string, runs []scored) string {
	var parts []string
	for _, m := range models {
		var ids []string
		for _, r := range runs {
			if r.model == m && r.rec != nil && r.rec.ModelReported != "" && !slices.Contains(ids, r.rec.ModelReported) {
				ids = append(ids, r.rec.ModelReported)
			}
		}
		if len(ids) == 0 {
			ids = []string{"no session reported"}
		}
		parts = append(parts, fmt.Sprintf("%s: %s", m, strings.Join(ids, ", ")))
	}
	return strings.Join(parts, "; ")
}

func evidence(e score.Evidence) string {
	var parts []string
	if e.Turn > 0 {
		parts = append(parts, fmt.Sprintf("turn %d", e.Turn))
	}
	if e.Line > 0 {
		parts = append(parts, fmt.Sprintf("line %d", e.Line))
	}
	if e.Command != "" {
		parts = append(parts, "`"+strings.ReplaceAll(e.Command, "`", "'")+"`")
	}
	if e.File != "" {
		parts = append(parts, e.File)
	}
	if e.Text != "" && e.Command == "" {
		parts = append(parts, fmt.Sprintf("%q", e.Text))
	}
	return strings.Join(parts, ", ")
}

func oneLine(s string) string {
	s, _, _ = strings.Cut(s, "\n")
	return s
}
