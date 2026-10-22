// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"archive/tar"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"metais.org/metais/internal/testenv"
	"metais.org/metais/internal/tools/ux/score"
	"metais.org/metais/internal/tools/ux/scrub"
)

// The tests in this file run whole sets with every command and every
// session faked: no go test starts a nested session or touches the
// network (plan §2, "a UX run is evidence, not a gate").

const fakeVersion = "2.1.299 (Claude Code)"

// fakeClaude is the Claude Code harness with its binary never run.
type fakeClaude struct{ *claudeCode }

func (fakeClaude) version() (string, error) { return fakeVersion, nil }

// fakeSet is a set's configuration over the real checkout and scenario
// set, with git, go, metais and the sessions faked.
type fakeSet struct {
	c        *config
	out      *bytes.Buffer
	sessions []invocation
	sleeps   []time.Duration
	// builds are the arguments of each go build the runner ran.
	builds [][]string
	commit string
	// archive is what git archive writes; nil writes treeTar(t, nil).
	archive []byte
	// quota makes a session of this scenario directory end on a quota
	// pause, as many times as it says.
	quota map[string]int
	// session is what a session writes to its transcript, given its
	// invocation; nil writes a clean session.
	session func(inv invocation) string
}

// fakeRoot is a checkout holding only what the runner reads of one: the
// module line of go.mod, and the scenario set, copied from this package's
// directory.
func fakeRoot(t *testing.T) string {
	t.Helper()
	root := testenv.Dir(t)
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module metais.org/metais\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.CopyFS(filepath.Join(root, scenariosDir), os.DirFS("./scenarios")); err != nil {
		t.Fatal(err)
	}
	// The scorer reads the C scenarios' package from its source.
	if err := os.CopyFS(filepath.Join(root, "examples", "dora-incident-report"), os.DirFS("../../../examples/dora-incident-report")); err != nil {
		t.Fatal(err)
	}
	return root
}

func newFakeSet(t *testing.T, scenarios ...string) *fakeSet {
	t.Helper()
	f := &fakeSet{out: &bytes.Buffer{}, commit: "0123456789abcdef0123456789abcdef01234567", quota: map[string]int{}}
	clock := time.Date(2026, 10, 21, 12, 0, 0, 0, time.UTC)
	f.c = &config{
		root:      fakeRoot(t),
		out:       filepath.Join(testenv.Dir(t), "set"),
		scenarios: scenarios,
		models:    []string{"haiku", "sonnet"},
		harness:   fakeClaude{&claudeCode{hostPath: "/usr/bin:/bin", baseURL: "http://127.0.0.1:9", binary: "/opt/claude/bin/claude"}},
		pace:      30 * time.Second,
		timeout:   time.Minute,
		environ:   []string{"PATH=/usr/bin:/bin"},
		stdout:    f.out,
		now:       func() time.Time { clock = clock.Add(1500 * time.Millisecond); return clock },
		sleep:     func(d time.Duration) { f.sleeps = append(f.sleeps, d) },
		command:   f.command,
		start:     f.start,
	}
	return f
}

// command fakes git, go build, and the metais under test.
func (f *fakeSet) command(name string, env []string, args ...string) ([]byte, error) {
	switch {
	case name == "git" && args[0] == "rev-parse":
		return []byte(f.commit + "\n"), nil
	case name == "git" && args[0] == "status":
		return nil, nil
	case name == "git" && args[0] == "archive":
		if !slices.Equal(args, []string{"archive", "--format=tar", f.commit, "--", "spec"}) {
			return nil, fmt.Errorf("unexpected git %q", args)
		}
		return f.archive, nil
	case name == "go" && args[0] == "build":
		f.builds = append(f.builds, args)
		return nil, os.WriteFile(args[slices.Index(args, "-o")+1], []byte("metais "+f.commit), 0o755)
	case filepath.Base(name) == "metais" && args[0] == "version":
		return []byte("metais 0.0.0-devel\n"), nil
	case filepath.Base(name) == "metais" && args[0] == "pack":
		return nil, os.WriteFile(args[slices.Index(args, "--out")+1], []byte("packed "+args[1]), 0o644)
	case filepath.Base(name) == "metais" && args[0] == "skill":
		dir := filepath.Join(args[slices.Index(args, "--dir")+1], ".claude", "skills", "metais")
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, err
		}
		return nil, os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte("skill"), 0o644)
	}
	return nil, fmt.Errorf("unexpected command %s %q", name, args)
}

// start fakes a session: a transcript of a clean session in the
// invocation's directory, which writes one file there.
func (f *fakeSet) start(_ context.Context, inv invocation, stdout, _ io.Writer) (int, error) {
	f.sessions = append(f.sessions, inv)
	run := filepath.Base(filepath.Dir(filepath.Dir(inv.dir)))
	if f.quota[run] > 0 {
		f.quota[run]--
		_, err := io.WriteString(stdout, `{"type":"result","is_error":true,"api_error_status":429,"result":"usage limit"}`+"\n")
		return 1, err
	}
	skills := `["debug"]`
	if _, err := os.Stat(filepath.Join(inv.dir, ".claude", "skills", "metais", "SKILL.md")); err == nil {
		skills = `["debug","metais"]`
	}
	t := initLine(session{work: inv.dir}, "2.1.299", "[]", skills) + "\n" + resultLine + "\n"
	if f.session != nil {
		t = f.session(inv)
	}
	if err := os.WriteFile(filepath.Join(inv.dir, "out.md"), []byte("rendered"), 0o644); err != nil {
		return 1, err
	}
	_, err := io.WriteString(stdout, t)
	return 0, err
}

func (f *fakeSet) record(t *testing.T, scenario, model string) *runRecord {
	t.Helper()
	r, err := readRecord(filepath.Join(f.c.out, "runs", scenario, model))
	if err != nil || r == nil {
		t.Fatalf("%s %s: record %v, %v", scenario, model, r, err)
	}
	return r
}

// allChecks are the checks every run that started makes.
var allChecks = []string{
	checkOutsideRepo, checkFresh, checkMetaisOnPath,
	checkCredentialName, checkAutoMemory, checkProjectSettings, checkStrictMCPName, checkNoPersistence,
	checkReported, checkStrictMCPName, checkSkill, checkNoPersistence, checkNoMemory,
	checkInsideRun,
}

func names(checks []check) []string {
	var n []string
	for _, c := range checks {
		n = append(n, c.Name)
	}
	return n
}

func TestRunSet(t *testing.T) {
	f := newFakeSet(t, "C1", "A2")
	if err := f.c.run(); err != nil {
		t.Fatalf("run: %v\n%s", err, f.out)
	}
	if len(f.sessions) != 4 {
		t.Fatalf("%d sessions, want 4 (two scenarios × two models)", len(f.sessions))
	}
	if !slices.Equal(f.sleeps, []time.Duration{30 * time.Second, 30 * time.Second, 30 * time.Second}) {
		t.Errorf("paced %v, want the pace between each two of four runs", f.sleeps)
	}
	r := f.record(t, "C1-render-real", "haiku")
	if !r.Valid || r.Status != statusComplete {
		t.Errorf("status %s, valid %v: %+v", r.Status, r.Valid, r.Checks)
	}
	if got := names(r.Checks); !slices.Equal(got, allChecks) {
		t.Errorf("checks %q, want %q", got, allChecks)
	}
	if r.Format != runFormat || r.Scenario.ID != "C1" || !r.Scenario.AsksForOutput || r.Scenario.Prompt != "render this file: report.metais" ||
		r.ModelRequested != "haiku" || r.ModelReported != "claude-haiku-5-5" || !slices.Equal(r.ModelsUsed, []string{"claude-haiku-5-5"}) ||
		r.Harness != (harnessRecord{claudeCodeName, fakeVersion}) || r.Commit != f.commit || r.MetaisSHA256 != digest([]byte("metais "+f.commit)) ||
		r.Turns != 3 || r.CostUSD != 0.01 || r.DurationMS != 1200 || r.WallMS != 1500 {
		t.Errorf("record %+v", r)
	}
	// The scorer wrote its outcome into the record: the fake session ran
	// no metais, so it fails.
	var o score.Outcome
	if err := json.Unmarshal(r.Outcome, &o); err != nil || o.Format != score.Format || !o.Scored || o.Pass ||
		!slices.Contains(o.Failing(), score.RuleInspectFirst) {
		t.Errorf("outcome %s: %v", r.Outcome, err)
	}
	files := []string{"out.md added", "report.metais unchanged"}
	var got []string
	for _, f := range r.Files {
		got = append(got, f.Path+" "+f.Change)
	}
	if !slices.Equal(got, files) {
		t.Errorf("files %q, want %q", got, files)
	}
	dir := filepath.Join(f.c.out, "runs", "C1-render-real", "haiku")
	for _, p := range []string{transcriptFile, stderrFile, filepath.Join(workDir, "report.metais")} {
		if _, err := os.Stat(filepath.Join(dir, p)); err != nil {
			t.Error(err)
		}
	}
	// A2's fixture is packed from the source beside its scenario.
	b, err := os.ReadFile(filepath.Join(f.c.out, "runs", "A2-edit-package", "sonnet", workDir, "invoice.metais"))
	if err != nil || string(b) != "packed "+filepath.Join(scenariosDir, "A2-edit-package", "invoice") {
		t.Errorf("A2's fixture: %q, %v", b, err)
	}
	// Each session has its own directory, HOME and TMPDIR, and the
	// placeholder credential.
	seen := map[string]bool{}
	for _, inv := range f.sessions {
		home, _ := lookup(inv.env, "HOME")
		tmp, _ := lookup(inv.env, "TMPDIR")
		ctmp, _ := lookup(inv.env, "CLAUDE_CODE_TMPDIR")
		tok, _ := lookup(inv.env, "ANTHROPIC_AUTH_TOKEN")
		if seen[inv.dir] || seen[home] || seen[tmp] || tmp != ctmp || tok != placeholderToken || !strings.HasPrefix(tmp, filepath.Dir(inv.dir)) {
			t.Errorf("session %s: HOME %s, TMPDIR %s, CLAUDE_CODE_TMPDIR %s, token %s", inv.dir, home, tmp, ctmp, tok)
		}
		seen[inv.dir], seen[home], seen[tmp] = true, true, true
	}
	if !strings.Contains(f.out.String(), "4 runs recorded: 4 valid, 0 invalid; 0 to run") {
		t.Errorf("summary:\n%s", f.out)
	}
	// Then the scorecard.
	for _, want := range []string{
		"# METAIS UX scorecard",
		"| Commit | `" + f.commit + "` |",
		"| Harness | claude-code " + fakeVersion + " |",
		"| Models | haiku: claude-haiku-5-5; sonnet: claude-haiku-5-5 |",
		"| Host | not stated (give -host) |",
		"| Skill | not installed (-with-skill off) |",
		"| C1-render-real | fail: inspect-first, outcome (asks-not-invents) |",
		"| A2-edit-package | fail: outcome (schema-and-template) (recorded) | fail: outcome (schema-and-template) |",
		"0 of 3 gated runs pass.",
		"## Prompts reworded under R-UX-9\n\nNone",
		"## What the outcome checks cannot see",
		"- **The pass rule `no-zip-extraction`**: The archive the rule measures is the package",
		"- **The pass rule `package-untouched`**: The archive the rule measures is the package",
		"## Recorded verdicts",
		"0 of 2 judged runs carry a verdict. Without one: C1-render-real haiku, C1-render-real sonnet.",
		"The phrase check and the verdict disagree on no run.",
		"## Notes\n\nNone.",
	} {
		if !strings.Contains(f.out.String(), want) {
			t.Errorf("the scorecard lacks %q:\n%s", want, f.out)
		}
	}

	// A recorded verdict decides C1's check, survives the runner's
	// rewriting of the record, and its disagreement with the phrase check
	// is listed.
	dir1 := filepath.Join(f.c.out, "runs", "C1-render-real", "haiku")
	rec := f.record(t, "C1-render-real", "haiku")
	rec.Verdict = json.RawMessage(`{"judge":"Ed","verdict":"pass","rationale":"it asked for the facts"}`)
	if err := writeJSON(filepath.Join(dir1, recordFile), rec); err != nil {
		t.Fatal(err)
	}
	f.out.Reset()
	if err := f.c.printScorecard(); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"1 of 2 judged runs carry a verdict. Without one: C1-render-real sonnet.",
		"- **C1-render-real haiku**: the phrase check says fail (the session renders nothing, and does not ask for the facts); Ed judged it pass: it asked for the facts",
	} {
		if !strings.Contains(f.out.String(), want) {
			t.Errorf("the scorecard lacks %q:\n%s", want, f.out)
		}
	}
	var kept bytes.Buffer
	if r := f.record(t, "C1-render-real", "haiku"); json.Compact(&kept, r.Verdict) != nil || kept.String() != string(rec.Verdict) {
		t.Errorf("the verdict was not kept: %s", kept.String())
	}

	// Run again, the set has nothing left to run.
	f.sessions, f.sleeps = nil, nil
	if err := f.c.run(); err != nil || len(f.sessions) != 0 {
		t.Errorf("a finished set ran again: %v, %d sessions", err, len(f.sessions))
	}
}

// TestRunSetResumesAfterAQuotaPause: a quota pause stops the set with the
// runs before it kept, and the next invocation runs only what is left.
func TestRunSetResumesAfterAQuotaPause(t *testing.T) {
	f := newFakeSet(t, "C1", "C4")
	f.quota["C4-what-it-needs"] = 1
	if err := f.c.run(); err != errQuota {
		t.Fatalf("run = %v, want the quota pause", err)
	}
	if r := f.record(t, "C4-what-it-needs", "haiku"); r.Status != statusQuota || r.Valid {
		t.Errorf("the paused run: %s, valid %v", r.Status, r.Valid)
	}
	if !strings.Contains(f.out.String(), "2 runs recorded") || !strings.Contains(f.out.String(), "2 to run") {
		t.Errorf("summary:\n%s", f.out)
	}
	f.sessions = nil
	if err := f.c.run(); err != nil {
		t.Fatalf("resume: %v", err)
	}
	var ran []string
	for _, inv := range f.sessions {
		ran = append(ran, strings.TrimPrefix(filepath.Dir(inv.dir), filepath.Join(f.c.out, "runs")+string(filepath.Separator)))
	}
	want := []string{filepath.Join("C4-what-it-needs", "haiku"), filepath.Join("C4-what-it-needs", "sonnet")}
	if !slices.Equal(ran, want) {
		t.Errorf("the resume ran %q, want %q", ran, want)
	}
	if r := f.record(t, "C4-what-it-needs", "haiku"); r.Status != statusComplete || !r.Valid {
		t.Errorf("the resumed run: %s, valid %v", r.Status, r.Valid)
	}
}

// TestRunSetWaitsOutAQuotaPause: with -quota-wait, the runner waits and
// runs the paused run again in the same invocation.
func TestRunSetWaitsOutAQuotaPause(t *testing.T) {
	f := newFakeSet(t, "C4")
	f.c.models = []string{"haiku"}
	f.c.quotaWait = time.Hour
	f.quota["C4-what-it-needs"] = 2
	if err := f.c.run(); err != nil {
		t.Fatal(err)
	}
	if len(f.sessions) != 3 || !slices.Contains(f.sleeps, time.Hour) {
		t.Errorf("%d sessions, sleeps %v", len(f.sessions), f.sleeps)
	}
	if r := f.record(t, "C4-what-it-needs", "haiku"); r.Status != statusComplete {
		t.Errorf("status %s", r.Status)
	}
}

func TestRunSetRefusesToResumeAnotherSet(t *testing.T) {
	f := newFakeSet(t, "C4")
	f.c.models = []string{"haiku"}
	if err := f.c.run(); err != nil {
		t.Fatal(err)
	}
	f.commit = "fedcba9876543210fedcba9876543210fedcba98"
	if err := f.c.run(); err == nil || !strings.Contains(err.Error(), "another commit") {
		t.Errorf("another commit: %v", err)
	}
	f.c.withSkill = true
	f.commit = "0123456789abcdef0123456789abcdef01234567"
	if err := f.c.run(); err == nil || !strings.Contains(err.Error(), "another skill mode") {
		t.Errorf("another skill mode: %v", err)
	}
}

func TestRunSetWithSkill(t *testing.T) {
	f := newFakeSet(t, "C4")
	f.c.models, f.c.withSkill = []string{"haiku"}, true
	if err := f.c.run(); err != nil {
		t.Fatal(err)
	}
	r := f.record(t, "C4-what-it-needs", "haiku")
	if !r.Valid || !r.WithSkill || !slices.ContainsFunc(r.Fixtures, func(e fileEntry) bool { return e.Path == ".claude/skills/metais/SKILL.md" }) {
		t.Errorf("valid %v, with skill %v, fixtures %v, checks %+v", r.Valid, r.WithSkill, r.Fixtures, r.Checks)
	}
}

// treeTar is a tar of the tree spec as git archive writes one: its pax
// global header naming the commit, its directories, then its files, and
// each extra entry after them.
func treeTar(t *testing.T, extra ...*tar.Header) []byte {
	t.Helper()
	var b bytes.Buffer
	w := tar.NewWriter(&b)
	entries := []struct {
		h    *tar.Header
		data string
	}{
		{&tar.Header{Typeflag: tar.TypeXGlobalHeader, Name: "pax_global_header", PAXRecords: map[string]string{"comment": "0123456789abcdef0123456789abcdef01234567"}}, ""},
		{&tar.Header{Typeflag: tar.TypeDir, Name: "spec/", Mode: 0o755}, ""},
		{&tar.Header{Typeflag: tar.TypeReg, Name: "spec/B-worked-package.md", Mode: 0o644}, "# The worked package\n"},
		{&tar.Header{Typeflag: tar.TypeDir, Name: "spec/schemas/", Mode: 0o755}, ""},
		{&tar.Header{Typeflag: tar.TypeReg, Name: "spec/schemas/manifest.schema.json", Mode: 0o644}, "{}\n"},
	}
	for _, h := range extra {
		entries = append(entries, struct {
			h    *tar.Header
			data string
		}{h, ""})
	}
	for _, e := range entries {
		e.h.Size = int64(len(e.data))
		if err := w.WriteHeader(e.h); err != nil {
			t.Fatal(err)
		}
		if _, err := io.WriteString(w, e.data); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

// TestRunSetCopiesATree: A4's session starts with the commit's spec/ as
// git archive gives it, every file of it a fixture the fresh-directory
// check holds the session's directory to; set.json records the tree's
// digest; and a scrubbed copy leaves out the files the session left
// unchanged, keeping what it wrote.
func TestRunSetCopiesATree(t *testing.T) {
	f := newFakeSet(t, "A4")
	f.c.models = []string{"opus"}
	f.archive = treeTar(t)
	if err := f.c.run(); err != nil {
		t.Fatalf("run: %v\n%s", err, f.out)
	}
	r := f.record(t, "A4-spec-only-package", "opus")
	if !r.Valid || r.Status != statusComplete {
		t.Errorf("status %s, valid %v: %+v", r.Status, r.Valid, r.Checks)
	}
	var fixtures []string
	for _, e := range r.Fixtures {
		fixtures = append(fixtures, e.Path)
	}
	if want := []string{"spec/B-worked-package.md", "spec/schemas/manifest.schema.json"}; !slices.Equal(fixtures, want) {
		t.Errorf("fixtures %q, want %q", fixtures, want)
	}
	work := filepath.Join(f.c.out, "runs", "A4-spec-only-package", "opus", workDir)
	if b, err := os.ReadFile(filepath.Join(work, "spec", "B-worked-package.md")); err != nil || string(b) != "# The worked package\n" {
		t.Errorf("the session's spec/: %q, %v", b, err)
	}
	var st setRecord
	if err := readJSON(filepath.Join(f.c.out, setFile), &st); err != nil {
		t.Fatal(err)
	}
	sum := digest([]byte("B-worked-package.md " + digest([]byte("# The worked package\n")) + "\n" +
		"schemas/manifest.schema.json " + digest([]byte("{}\n")) + "\n"))
	if !slices.Equal(st.Fixtures, []fixtureRecord{{"A4-spec-only-package", "spec/", sum}}) {
		t.Errorf("set.json's fixtures %+v, want the tree spec/ at %s", st.Fixtures, sum)
	}

	// A tree of another commit is another set.
	f.archive = treeTar(t, &tar.Header{Typeflag: tar.TypeReg, Name: "spec/new.md", Mode: 0o644})
	if err := f.c.run(); err == nil || !strings.Contains(err.Error(), "fixture A4-spec-only-package/spec/") {
		t.Errorf("a changed tree resumed: %v", err)
	}

	dest := filepath.Join(testenv.Dir(t), "scrubbed")
	if err := f.c.scrubSet(dest); err != nil {
		t.Fatal(err)
	}
	run := filepath.Join(dest, "runs", "A4-spec-only-package", "opus", workDir)
	if _, err := os.Stat(filepath.Join(run, "spec")); err == nil {
		t.Error("the scrubbed copy holds the spec/ the session left unchanged")
	}
	if _, err := os.Stat(filepath.Join(run, "out.md")); err != nil {
		t.Errorf("the scrubbed copy lacks the file the session wrote: %v", err)
	}
}

// TestRunSetRefusesATreeOfOtherEntries: a tree whose archive holds a
// link, or a path outside the tree, is refused before any session runs.
func TestRunSetRefusesATreeOfOtherEntries(t *testing.T) {
	for _, c := range []struct {
		name string
		h    *tar.Header
		want string
	}{
		{"a link", &tar.Header{Typeflag: tar.TypeSymlink, Name: "spec/l", Linkname: "../../etc"}, "not a regular file"},
		{"a path outside", &tar.Header{Typeflag: tar.TypeReg, Name: "spec/../x", Mode: 0o644}, "outside it"},
		{"another tree", &tar.Header{Typeflag: tar.TypeReg, Name: "docs/x", Mode: 0o644}, "outside it"},
	} {
		f := newFakeSet(t, "A4")
		f.c.models = []string{"opus"}
		f.archive = treeTar(t, c.h)
		if err := f.c.run(); err == nil || !strings.Contains(err.Error(), c.want) || len(f.sessions) != 0 {
			t.Errorf("%s: run = %v, %d sessions, want an error containing %q", c.name, err, len(f.sessions), c.want)
		}
	}
	f := newFakeSet(t, "A4")
	f.c.models = []string{"opus"}
	var empty bytes.Buffer
	if err := tar.NewWriter(&empty).Close(); err != nil {
		t.Fatal(err)
	}
	f.archive = empty.Bytes()
	if err := f.c.run(); err == nil || !strings.Contains(err.Error(), "holds no file") {
		t.Errorf("an empty tree: %v", err)
	}
}

// TestRunSetBuildsMetaisUnstamped: the runner builds the metais under
// test with -trimpath and -buildvcs=false. git in the sandbox fails on an
// iconv locale error when the go command stamps the build (R-UX-7), so
// the build must not ask it to.
func TestRunSetBuildsMetaisUnstamped(t *testing.T) {
	f := newFakeSet(t, "C4")
	f.c.models = []string{"haiku"}
	if err := f.c.run(); err != nil {
		t.Fatal(err)
	}
	if len(f.builds) != 1 {
		t.Fatalf("go build ran %d times, want once: %q", len(f.builds), f.builds)
	}
	for _, flag := range []string{"-trimpath", "-buildvcs=false"} {
		if !slices.Contains(f.builds[0], flag) {
			t.Errorf("go %q builds metais without %s", f.builds[0], flag)
		}
	}
}

// TestPlaceholderIsScrubbed: the token the runner gives a session as its
// credential is one the scrub removes, so a fixture recorded with it can
// never hold it.
func TestPlaceholderIsScrubbed(t *testing.T) {
	c := &claudeCode{hostPath: "/usr/bin:/bin", baseURL: "http://127.0.0.1:9", binary: "/opt/claude/bin/claude"}
	inv := c.invocation(session{work: "/run/work", home: "/run/home", tmp: "/run/tmp", bin: "/set/bin", model: "haiku", prompt: "p"})
	tok, ok := lookup(inv.env, "ANTHROPIC_AUTH_TOKEN")
	if !ok || tok == "" {
		t.Fatalf("the session's environment holds no ANTHROPIC_AUTH_TOKEN: %q", inv.env)
	}
	if got := string(scrub.New("/run", "/set").Bytes([]byte(tok))); strings.Contains(got, tok) || len(scrub.Check([]byte(tok))) == 0 {
		t.Errorf("the scrub keeps the runner's placeholder %q: %q", tok, got)
	}
}

// TestRunSetRecordsAnInvalidRun: a run whose session shows what a check
// forbids is recorded, and recorded invalid.
func TestRunSetRecordsAnInvalidRun(t *testing.T) {
	f := newFakeSet(t, "C4")
	f.c.models = []string{"haiku"}
	f.session = func(inv invocation) string {
		return initLine(session{work: inv.dir}, "2.1.299", "[]", "[]") + "\n" +
			toolLine("Read", `{"file_path":"/home/x/.claude/CLAUDE.md"}`) + "\n" + resultLine + "\n"
	}
	if err := f.c.run(); err != nil {
		t.Fatal(err)
	}
	r := f.record(t, "C4-what-it-needs", "haiku")
	if r.Valid || r.Status != statusComplete {
		t.Errorf("status %s, valid %v", r.Status, r.Valid)
	}
	if !strings.Contains(f.out.String(), "1 runs recorded: 0 valid, 1 invalid") {
		t.Errorf("summary:\n%s", f.out)
	}
}

// TestRunSetRefusesAWriteOutsideTheRun: a session that writes into the
// set's directory outside its own run's, as a Haiku A2 session did with
// <set>/invoice-src on 2026-10-21, or names a file outside the run's
// directory in a Write call, is recorded invalid; one that writes only in
// its own directory is not.
func TestRunSetRefusesAWriteOutsideTheRun(t *testing.T) {
	clean := func(inv invocation) string {
		return initLine(session{work: inv.dir}, "2.1.299", "[]", "[]") + "\n" + resultLine + "\n"
	}
	for _, c := range []struct {
		name    string
		session func(inv invocation) string
		want    string
	}{
		{"its own directory", func(inv invocation) string {
			if err := os.WriteFile(filepath.Join(filepath.Dir(inv.dir), "tmp", "x.json"), []byte("{}"), 0o644); err != nil {
				t.Fatal(err)
			}
			return clean(inv)
		}, ""},
		{"the set's directory", func(inv invocation) string {
			set := filepath.Dir(filepath.Dir(filepath.Dir(filepath.Dir(inv.dir))))
			if err := os.MkdirAll(filepath.Join(set, "invoice-src"), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(set, "invoice-src", "metais.yaml"), []byte("x"), 0o644); err != nil {
				t.Fatal(err)
			}
			return clean(inv)
		}, "the set's invoice-src/metais.yaml is added outside the run's directory"},
		{"a Write elsewhere", func(inv invocation) string {
			return initLine(session{work: inv.dir}, "2.1.299", "[]", "[]") + "\n" +
				toolLine("Write", `{"file_path":"../../../../../elsewhere/a.json","content":"{}"}`) + "\n" + resultLine + "\n"
		}, "tool call 1 (Write) writes"},
	} {
		f := newFakeSet(t, "C4")
		f.c.models = []string{"haiku"}
		f.session = c.session
		if err := f.c.run(); err != nil {
			t.Fatal(err)
		}
		r := f.record(t, "C4-what-it-needs", "haiku")
		i := slices.IndexFunc(r.Checks, func(ch check) bool { return ch.Name == checkInsideRun })
		switch {
		case i < 0:
			t.Errorf("%s: no %s check", c.name, checkInsideRun)
		case c.want == "" && (!r.Checks[i].Pass || !r.Valid):
			t.Errorf("%s: %+v, valid %v", c.name, r.Checks[i], r.Valid)
		case c.want != "" && (r.Checks[i].Pass || r.Valid || !strings.Contains(r.Checks[i].Detail, c.want)):
			t.Errorf("%s: %+v, valid %v, want a failure naming %q", c.name, r.Checks[i], r.Valid, c.want)
		}
	}
}

// TestRunSetDoesNotStartAnInvalidSession: a run that fails a check before
// it starts is recorded as not run, and no session starts.
func TestRunSetDoesNotStartAnInvalidSession(t *testing.T) {
	f := newFakeSet(t, "C4")
	f.c.models = []string{"haiku"}
	// A metais of the host's first on the session's PATH.
	other := testenv.Dir(t)
	if err := os.WriteFile(filepath.Join(other, "metais"), []byte("the host's metais"), 0o755); err != nil {
		t.Fatal(err)
	}
	f.c.harness = prependPath{f.c.harness.(fakeClaude), other}
	if err := f.c.run(); err != nil {
		t.Fatal(err)
	}
	r := f.record(t, "C4-what-it-needs", "haiku")
	if r.Status != statusNotRun || r.Valid || len(f.sessions) != 0 {
		t.Errorf("status %s, valid %v, %d sessions", r.Status, r.Valid, len(f.sessions))
	}
	i := slices.IndexFunc(r.Checks, func(c check) bool { return c.Name == checkMetaisOnPath })
	if i < 0 || r.Checks[i].Pass {
		t.Errorf("checks %+v", r.Checks)
	}
}

// prependPath is a harness whose sessions' PATH starts with dir, before
// the metais under test.
type prependPath struct {
	fakeClaude
	dir string
}

func (p prependPath) invocation(s session) invocation {
	inv := p.fakeClaude.invocation(s)
	path, _ := lookup(inv.env, "PATH")
	inv.env = append(inv.env, "PATH="+p.dir+string(filepath.ListSeparator)+path)
	return inv
}

func TestRunRefuses(t *testing.T) {
	for _, c := range []struct {
		name string
		edit func(c *config)
		want string
	}{
		{"a relative -out", func(c *config) { c.out = "set" }, "not an absolute path"},
		{"not the checkout", func(c *config) { c.root = testenv.Dir(t) }, "top of a metais checkout"},
		{"no model", func(c *config) { c.models = nil }, "no model"},
		{"a model that is a path", func(c *config) { c.models = []string{"../haiku"} }, "not a model's name"},
		{"an unknown scenario", func(c *config) { c.scenarios = []string{"C9"} }, "no scenario C9"},
	} {
		f := newFakeSet(t)
		c.edit(f.c)
		if err := f.c.run(); err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: run = %v, want %q", c.name, err, c.want)
		}
	}
	// -out inside a repository, by an absolute path.
	dir := testenv.Dir(t)
	if err := os.Mkdir(filepath.Join(dir, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	f := newFakeSet(t)
	f.c.out = filepath.Join(dir, "set")
	if err := f.c.run(); err == nil || !strings.Contains(err.Error(), "inside the repository") {
		t.Errorf("-out in a repository: %v", err)
	}
	// A dirty checkout.
	f = newFakeSet(t)
	cmd := f.c.command
	f.c.command = func(name string, env []string, args ...string) ([]byte, error) {
		if name == "git" && args[0] == "status" {
			return []byte(" M cmd/metais/main.go\n"), nil
		}
		return cmd(name, env, args...)
	}
	if err := f.c.run(); err == nil || !strings.Contains(err.Error(), "uncommitted changes") {
		t.Errorf("a dirty checkout: %v", err)
	}
	f.c.allowDirty = true
	f.c.scenarios, f.c.models = []string{"C4"}, []string{"haiku"}
	if err := f.c.run(); err != nil {
		t.Errorf("a dirty checkout allowed: %v", err)
	}
}

func TestHarnessByName(t *testing.T) {
	if _, err := harnessByName("codex", []string{"PATH=/bin"}); err == nil {
		t.Error("an unimplemented harness was accepted")
	}
	h, err := harnessByName(claudeCodeName, []string{"PATH=/bin"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := h.version(); err == nil || !strings.Contains(err.Error(), "ANTHROPIC_BASE_URL") {
		t.Errorf("no proxy: version = %v", err)
	}
}
