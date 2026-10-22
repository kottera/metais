// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package score

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/kottera/metais/internal/tools/ux/scenario"
	"github.com/kottera/metais/internal/tools/ux/scrub"
)

// The scorer is proved against recorded runs: each directory of
// testdata/runs is one run the runner recorded, put through -scrub, with
// fixture.json saying what it shows, which run it was recorded as, and
// what every rule must make of it. No go test starts a session: these
// are the recordings (plan §2, "a UX run is evidence, not a gate").

const fixturesDir = "testdata/runs"

// fixture is a recorded run's fixture.json.
type fixture struct {
	// Shows is the behaviour the fixture is named for.
	Shows string `json:"shows"`
	// Recorded names the real run: the set it was recorded in, the run,
	// the day, the commit and the model the session reported.
	Recorded struct {
		Set    string `json:"set"`
		Run    string `json:"run"`
		Date   string `json:"date"`
		Commit string `json:"commit"`
		Model  string `json:"model"`
	} `json:"recorded"`
	// Edited says what was changed in the recording to show the behaviour,
	// when a model would not show it on demand; "" for a run as recorded.
	Edited string `json:"edited,omitempty"`
	// Expect is every rule's result: pass, fail, or n/a for a rule the
	// scenario is not held to.
	Expect map[string]string `json:"expect"`
	// Reasons are, for each rule expected to fail, words its detail must
	// hold: the reason the fixture fails it for.
	Reasons map[string]string `json:"reasons"`
}

type loaded struct {
	name string
	fx   fixture
	in   Input
}

// repoRoot is the top of the checkout, from this package's directory.
const repoRoot = "../../../.."

func loadFixtures(t *testing.T) []loaded {
	t.Helper()
	set, err := scenario.Load(os.DirFS("../scenarios"))
	if err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(fixturesDir)
	if err != nil {
		t.Fatal(err)
	}
	var out []loaded
	for _, e := range entries {
		dir := filepath.Join(fixturesDir, e.Name())
		var fx fixture
		b, err := os.ReadFile(filepath.Join(dir, "fixture.json"))
		if err != nil {
			t.Fatal(err)
		}
		dec := json.NewDecoder(strings.NewReader(string(b)))
		dec.DisallowUnknownFields()
		if err := dec.Decode(&fx); err != nil {
			t.Fatalf("%s/fixture.json: %v", e.Name(), err)
		}
		var rec record
		rb, err := os.ReadFile(filepath.Join(dir, "record.json"))
		if err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(rb, &rec); err != nil {
			t.Fatal(err)
		}
		i := slices.IndexFunc(set, func(s *scenario.Scenario) bool { return s.ID == rec.Scenario.ID })
		if i < 0 {
			t.Fatalf("%s: no scenario %s", e.Name(), rec.Scenario.ID)
		}
		s := set[i]
		out = append(out, loaded{e.Name(), fx, Input{Scenario: s, Run: os.DirFS(dir), Sources: sources(s)}})
	}
	return out
}

// sources are a scenario's fixtures' source directories, as the runner
// hands them to the scorer.
func sources(s *scenario.Scenario) map[string]fs.FS {
	m := map[string]fs.FS{}
	for _, f := range s.Fixtures {
		if !f.Package() {
			continue
		}
		dir := filepath.Join(repoRoot, "examples", f.Example)
		if f.Source != "" {
			dir = filepath.Join("..", "scenarios", s.Dir, f.Source)
		}
		m[f.As] = os.DirFS(dir)
	}
	return m
}

// compare holds an outcome to a fixture's expectations, and names every
// rule it disagrees on.
func compare(fx fixture, o *Outcome) error {
	if !o.Scored {
		return fmt.Errorf("not scored: %s", o.NotScored)
	}
	var errs []string
	for _, r := range o.Rules {
		got := map[bool]string{true: "pass", false: "fail"}[r.Pass]
		if !r.Applies {
			got = "n/a"
		}
		want, ok := fx.Expect[r.Rule]
		switch {
		case !ok:
			errs = append(errs, fmt.Sprintf("%s: no expectation; it is %s: %s", r.Rule, got, r.Detail))
		case got != want:
			errs = append(errs, fmt.Sprintf("%s is %s, want %s: %s", r.Rule, got, want, r.Detail))
		case got == "fail" && (fx.Reasons[r.Rule] == "" || !strings.Contains(r.Detail, fx.Reasons[r.Rule])):
			errs = append(errs, fmt.Sprintf("%s fails, but not for %q: %s", r.Rule, fx.Reasons[r.Rule], r.Detail))
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("%s", strings.Join(errs, "\n\t"))
	}
	return nil
}

// TestFixtures: the scorer makes of every recorded run what its
// fixture.json says, every rule's result, and each failure for the
// reason the fixture is named for.
func TestFixtures(t *testing.T) {
	for _, f := range loadFixtures(t) {
		o, err := Score(f.in)
		if err != nil {
			t.Errorf("%s: %v", f.name, err)
			continue
		}
		if err := compare(f.fx, o); err != nil {
			t.Errorf("%s (%s):\n\t%v", f.name, f.fx.Shows, err)
		}
	}
}

// TestFixturesSeenRed: every failure a fixture expects is the rule's own.
// With that one rule disabled, made to pass whatever the run did, the
// fixture no longer scores as it says, so TestFixtures goes red: no
// failing fixture passes TestFixtures on another rule's account.
func TestFixturesSeenRed(t *testing.T) {
	saved := slices.Clone(rules)
	t.Cleanup(func() { rules = saved })
	for _, f := range loadFixtures(t) {
		for i, ru := range saved {
			if f.fx.Expect[ru.name] != "fail" {
				continue
			}
			rules = slices.Clone(saved)
			rules[i].check = func(*run) Result { return Result{Pass: true, Detail: "disabled"} }
			o, err := Score(f.in)
			if err != nil {
				t.Fatal(err)
			}
			err = compare(f.fx, o)
			if err == nil || !strings.Contains(err.Error(), ru.name+" is pass, want fail") {
				t.Errorf("%s: with %s disabled, TestFixtures would not go red on it: %v", f.name, ru.name, err)
			}
		}
	}
	rules = saved
}

// TestFixturesCoverEveryRule: every rule, and every scenario's outcome
// check, has a recorded run that passes it and one that fails it.
func TestFixturesCoverEveryRule(t *testing.T) {
	seen := map[string]map[string][]string{}
	note := func(key, result, name string) {
		if seen[key] == nil {
			seen[key] = map[string][]string{}
		}
		seen[key][result] = append(seen[key][result], name)
	}
	for _, f := range loadFixtures(t) {
		for rule, result := range f.fx.Expect {
			if rule == RuleOutcome {
				name, err := CheckName(f.in.Scenario)
				if err != nil {
					t.Fatal(err)
				}
				rule = "outcome:" + name
			}
			note(rule, result, f.name)
		}
	}
	keys := []string{RuleInspectFirst, RuleNoZip, RuleNoUnknown, RuleUntouched}
	for _, n := range CheckNames() {
		keys = append(keys, "outcome:"+n)
	}
	for _, k := range keys {
		for _, result := range []string{"pass", "fail"} {
			if len(seen[k][result]) == 0 {
				t.Errorf("no fixture %ss %s", strings.TrimSuffix(result, "s"), k)
			}
		}
	}
}

// TestFixturesAreRecorded: every fixture names the real run it was
// recorded as, and an edited one says what was edited.
func TestFixturesAreRecorded(t *testing.T) {
	for _, f := range loadFixtures(t) {
		r := f.fx.Recorded
		if f.fx.Shows == "" || r.Set == "" || r.Run == "" || r.Date == "" || len(r.Commit) != 40 || r.Model == "" {
			t.Errorf("%s: fixture.json does not say what it shows and which run it was recorded as: %+v", f.name, f.fx)
		}
		var rec struct {
			Commit        string `json:"commit"`
			ModelReported string `json:"model_reported"`
		}
		b, _ := fs.ReadFile(f.in.Run, "record.json")
		if json.Unmarshal(b, &rec) != nil || rec.Commit != r.Commit || rec.ModelReported != r.Model {
			t.Errorf("%s: fixture.json names commit %s and model %s, the record %s and %s", f.name, r.Commit, r.Model, rec.Commit, rec.ModelReported)
		}
		if strings.HasPrefix(f.name, "edited-") != (f.fx.Edited != "") {
			t.Errorf("%s: an edited fixture's name begins edited-, and only it says what was edited", f.name)
		}
	}
}

// TestFixturesAreScrubbed: no committed file of a fixture holds a
// credential, the placeholder token among them, or a host path outside
// the run's own directory.
func TestFixturesAreScrubbed(t *testing.T) {
	err := filepath.WalkDir(fixturesDir, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		for _, f := range scrub.Check(b) {
			t.Errorf("%s holds %s", p, f)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

// TestEveryScenarioNamesACheck: each scenario of the set names one of the
// scorer's outcome checks, and each check is some scenario's.
func TestEveryScenarioNamesACheck(t *testing.T) {
	set, err := scenario.Load(os.DirFS("../scenarios"))
	if err != nil {
		t.Fatal(err)
	}
	used := map[string]bool{}
	for _, s := range set {
		name, err := CheckName(s)
		if err != nil {
			t.Error(err)
		}
		used[name] = true
		if CheckLimits(name) == "" {
			t.Errorf("%s: check %s states no limits", s.Name(), name)
		}
	}
	for _, n := range CheckNames() {
		if !used[n] {
			t.Errorf("check %s is no scenario's", n)
		}
	}
}

// TestNotScored: a run the runner recorded invalid, or one that did not
// complete, is not scored, and says why.
func TestNotScored(t *testing.T) {
	set, err := scenario.Load(os.DirFS("../scenarios"))
	if err != nil {
		t.Fatal(err)
	}
	s := set[slices.IndexFunc(set, func(s *scenario.Scenario) bool { return s.ID == "C1" })]
	for _, c := range []struct{ rec, want string }{
		{`{"format":"metais-ux-run/1","scenario":{"id":"C1"},"status":"complete","valid":false,"checks":[{"name":"no-memory-read","pass":false,"detail":"tool call 2 (Read) names a memory file"}]}`, "the run is invalid: no-memory-read: tool call 2"},
		{`{"format":"metais-ux-run/1","scenario":{"id":"C1"},"status":"timeout","valid":true}`, "its status is timeout"},
	} {
		run := fstest.MapFS{"record.json": {Data: []byte(c.rec)}, "transcript.jsonl": {Data: nil}}
		o, err := Score(Input{Scenario: s, Run: run, Sources: sources(s)})
		if err != nil {
			t.Fatal(err)
		}
		if o.Scored || o.Pass || len(o.Rules) != 0 || !strings.Contains(o.NotScored, c.want) {
			t.Errorf("outcome %+v, want not scored for %q", o, c.want)
		}
	}
	// A record of another scenario is refused, not scored as this one.
	run := fstest.MapFS{"record.json": {Data: []byte(`{"format":"metais-ux-run/1","scenario":{"id":"C2"},"status":"complete","valid":true}`)}}
	if _, err := Score(Input{Scenario: s, Run: run}); err == nil {
		t.Error("a C2 record was scored as C1")
	}
}

// withVerdict is a fixture's run with a verdict written into its record.
func withVerdict(t *testing.T, f loaded, v string) Input {
	t.Helper()
	m := fstest.MapFS{}
	err := fs.WalkDir(f.in.Run, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		b, err := fs.ReadFile(f.in.Run, p)
		m[p] = &fstest.MapFile{Data: b}
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	var rec map[string]any
	if err := json.Unmarshal(m["record.json"].Data, &rec); err != nil {
		t.Fatal(err)
	}
	if v != "" {
		rec["verdict"] = map[string]any{"judge": "claude-opus-5-5", "verdict": v, "rationale": "the session says it took the edit reading"}
	}
	b, _ := json.Marshal(rec)
	m["record.json"] = &fstest.MapFile{Data: b}
	in := f.in
	in.Run = m
	return in
}

// TestVerdict: a judged check (C0's and C1's) follows a recorded verdict
// when the run carries one, keeps the phrase check's result beside it,
// and says when the two disagree; without one it follows the phrase check
// and says a verdict is needed. A check that reads no words ignores one.
func TestVerdict(t *testing.T) {
	byName := map[string]loaded{}
	for _, f := range loadFixtures(t) {
		byName[f.name] = f
	}
	outcome := func(o *Outcome) Result {
		return o.Rules[slices.IndexFunc(o.Rules, func(r Result) bool { return r.Rule == RuleOutcome })]
	}
	for _, c := range []struct {
		fixture, verdict      string
		pass, disagree, needs bool
	}{
		{"edited-c0-silent-edit-reading", "", false, false, true},
		{"edited-c0-silent-edit-reading", "pass", true, true, false},
		{"edited-c0-silent-edit-reading", "fail", false, false, false},
		{"edited-c0-stated-edit-reading", "fail", false, true, false},
		{"c1-sonnet-asks", "pass", true, false, false},
		{"c1-sonnet-asks", "maybe", true, false, true},
	} {
		o, err := Score(withVerdict(t, byName[c.fixture], c.verdict))
		if err != nil {
			t.Fatal(err)
		}
		res := outcome(o)
		if !o.Judged || res.Pass != c.pass || o.Disagree != c.disagree || o.NeedsVerdict != c.needs || (o.Verdict != nil) == c.needs {
			t.Errorf("%s with verdict %q: outcome %+v, judged %v, disagree %v, needs %v", c.fixture, c.verdict, res, o.Judged, o.Disagree, o.NeedsVerdict)
		}
		if o.Verdict != nil && (o.Phrase == nil || !strings.Contains(res.Detail, "judged "+c.verdict+" by claude-opus-5-5")) {
			t.Errorf("%s: the phrase check's result is not kept beside the verdict: %+v", c.fixture, res)
		}
	}
	o, err := Score(withVerdict(t, byName["c2-sonnet-new-data"], "fail"))
	if err != nil {
		t.Fatal(err)
	}
	if o.Judged || o.Verdict != nil || !outcome(o).Pass {
		t.Errorf("a verdict decided C2's check, which reads no words: %+v", o)
	}
}

// TestListingIsANote: a listing of the archive (unzip -l) passes both
// rules that read for extraction, and is noted.
func TestListingIsANote(t *testing.T) {
	for _, f := range loadFixtures(t) {
		if f.name != "c4-haiku-unzip" {
			continue
		}
		o, err := Score(f.in)
		if err != nil {
			t.Fatal(err)
		}
		res := o.Rules[slices.IndexFunc(o.Rules, func(r Result) bool { return r.Rule == RuleNoZip })]
		if !res.Pass || len(res.Notes) == 0 || !strings.Contains(res.Notes[0].Text, "unzip lists the archive") {
			t.Errorf("unzip -l: %+v", res)
		}
		return
	}
	t.Fatal("no fixture c4-haiku-unzip")
}
