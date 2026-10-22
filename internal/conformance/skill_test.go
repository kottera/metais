// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package conformance

import (
	"bytes"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"metais.org/metais/internal/testenv"
)

// skillHarnesses are the harnesses metais skill writes for, and the
// directory each one's SKILL.md goes below, beneath --dir (ADR-008 A2):
// .claude/skills/metais for Claude Code, .agents/skills/metais for the
// others. They are written from the decision, not read from the tool, so
// a harness the tool stops writing where the decision says is red here.
var skillHarnesses = []struct{ name, dir string }{
	{"claude", ".claude/skills/metais"},
	{"codex", ".agents/skills/metais"},
	{"gemini", ".agents/skills/metais"},
	{"cursor", ".agents/skills/metais"},
	{"opencode", ".agents/skills/metais"},
}

// skillDirName is the decoy directory metais skill writes beneath, in the
// work directory. Its name holds the sentinel, the sandbox path's, so a
// stream that echoes --dir is caught by check 1 in either path mode, and a
// written path below it, which carries the sandbox path anyway, is not.
// It is the same in both environments, so the relative mode's command
// lines are the same in A and B.
const skillDirName = sentinel + "PATHSKILLDIR"

// skillCase is one command line of ADR-009 A5 item 5: metais skill for a
// harness, or with --print when harness is "", in a form and a path mode.
type skillCase struct {
	harness string
	dir     string // where SKILL.md goes below --dir, or "" with --print
	json    bool
	mode    pathMode
}

func (c skillCase) String() string {
	form, mode := "human", "relative"
	if c.json {
		form = "json"
	}
	if c.mode == absPaths {
		mode = "absolute"
	}
	what := "--print"
	if c.harness != "" {
		what = "--harness " + c.harness
	}
	return fmt.Sprintf("skill %s, %s, %s paths", what, form, mode)
}

// skillCases are every command line ADR-009 A5 item 5 runs: every harness
// in both forms, and --print in both forms, each in both path modes.
func skillCases() []skillCase {
	var cs []skillCase
	for _, mode := range []pathMode{relPaths, absPaths} {
		for _, h := range skillHarnesses {
			for _, asJSON := range []bool{false, true} {
				cs = append(cs, skillCase{harness: h.name, dir: h.dir, json: asJSON, mode: mode})
			}
		}
		for _, asJSON := range []bool{false, true} {
			cs = append(cs, skillCase{json: asJSON, mode: mode})
		}
	}
	return cs
}

// skillRun is what one skill case gave in one environment.
type skillRun struct {
	run     *envRun
	inv     *invocation
	written []byte // the SKILL.md written, or nil
}

// TestSkillHostIsolation is ADR-009 A5 item 5: the SC7 harness (D16 item
// 1) runs the release build's metais skill for every harness, in both
// path modes, with and without --json, and with --print, under
// environments A and B, against a decoy --dir whose name holds the
// sentinel and with environment A's decoys in the home, working and
// temporary directories, SKILL.md decoys among them. Its checks apply as
// to every command: check 1, no sentinel in any stream or in the written
// file; check 2, no real host fact in either; check 3, A and B give the
// same streams, exit and written bytes, B started two seconds after A
// ended; and check 4, nothing changes anywhere in the sandbox but the
// SKILL.md the decision names and the directories it needs, and nothing
// at all with --print. Every run exits 0, and the file written is the
// skill --print prints.
func TestSkillHostIsolation(t *testing.T) {
	if builtTool == "" {
		t.Skip("runs the release build of ./cmd/metais, which -conformance.tool replaces")
	}
	t.Parallel()
	program := releaseBuild(t)
	text := skillText(t, program)
	facts := skillFacts(t, program, text)
	v := &vector{ID: "skill", dir: testenv.Dir(t)} // no files of its own
	base := testenv.Dir(t)
	cases := skillCases()

	a := runSkillEnv(program, base, envA, v, facts, cases)
	time.Sleep(2 * time.Second) // B starts two seconds after A's last process ended (runCaseIn)
	b := runSkillEnv(program, base, envB, v, facts, cases)

	for i, c := range cases {
		ra, rb := a[i], b[i]
		for _, r := range []*skillRun{ra, rb} {
			for _, p := range r.run.problems {
				t.Errorf("%s, environment %s: %s", c, r.run.sb.env, p)
			}
		}
		if ra.run.fatal() || rb.run.fatal() {
			continue
		}
		for _, p := range differences(ra.run, rb.run, checkAB, differingVars(ra.run.sb.vars, rb.run.sb.vars)) {
			t.Errorf("%s: %s", c, p)
		}
		if c.harness == "" {
			if !c.json && !bytes.Equal(ra.inv.stdout, text) {
				t.Errorf("%s: standard output is not the skill --print printed first\n%s", c, firstDifference(text, ra.inv.stdout))
			}
			continue
		}
		if !bytes.Equal(ra.written, text) {
			t.Errorf("%s: the SKILL.md written is not the skill --print prints\n%s", c, firstDifference(text, ra.written))
		}
	}

	// The harness sees a leak: the decoy directory's absolute path in a
	// stream is a sentinel hit, and the plain run's streams were clean.
	for i, c := range cases {
		if c.mode != absPaths || c.harness == "" || a[i].inv == nil {
			continue
		}
		leaked := *a[i].inv
		leaked.stdout = append(slices.Clone(leaked.stdout), filepath.Join(a[i].run.sb.work, skillDirName)...)
		ps := scanInvocation(sentinels, &leaked, false, nil)
		if !slices.ContainsFunc(ps, func(p problem) bool { return p.check == checkSentinel }) {
			t.Errorf("%s: a stream holding the decoy directory's absolute path is not caught by the sentinel scan; it reported %q", c, ps)
		}
		break
	}
}

// skillText is the skill the program prints with --print, in a plain
// sandbox.
func skillText(t *testing.T, program string) []byte {
	t.Helper()
	sb, err := plainSandbox(testenv.Dir(t))
	if err != nil {
		t.Fatal(err)
	}
	p, err := runProc(program, []string{"skill", "--print"}, sb.work, sb.environ(), nil, defaultWall)
	if err != nil {
		t.Fatal(err)
	}
	if p.exit != 0 || len(p.stdout) == 0 {
		t.Fatalf("skill --print exited %d with %d bytes\n%s", p.exit, len(p.stdout), p.stderr)
	}
	return p.stdout
}

// skillFacts are check 2's host facts for the program, as hostRunner
// gathers them, with the skill's own text beside the tool's help and the
// registry: a host or user name that is a word of the skill is the tool's
// own text, which a scan cannot tell from a leak.
func skillFacts(t *testing.T, program string, text []byte) *hostFacts {
	t.Helper()
	vocab, err := toolVocabulary(program, testenv.Dir(t))
	if err != nil {
		t.Fatal(err)
	}
	reg, err := os.ReadFile(filepath.Join(defaultSuite(), "..", "findings", "registry.json"))
	if err != nil {
		t.Fatal(err)
	}
	return newHostFacts(append(vocab, reg, text)...)
}

// runSkillEnv runs the cases in environment e, as many at once as
// -parallel allows (the runner's own bound, main_test.go), each in a fresh
// sandbox of its own below base, and returns their runs in the cases'
// order.
func runSkillEnv(program, base string, e env, v *vector, facts *hostFacts, cases []skillCase) []*skillRun {
	out := make([]*skillRun, len(cases))
	slots := make(chan struct{}, testParallel())
	var wg sync.WaitGroup
	for i, c := range cases {
		wg.Go(func() {
			slots <- struct{}{}
			defer func() { <-slots }()
			out[i] = runSkillCase(program, base, e, v, facts, c)
		})
	}
	wg.Wait()
	return out
}

// runSkillCase runs case c in a fresh sandbox of environment e prepared
// as the SC7 harness prepares one for c's path mode, and makes checks 1,
// 2 and 4 of its run, the exit and, with --json, the document's ok.
func runSkillCase(program, base string, e env, v *vector, facts *hostFacts, c skillCase) *skillRun {
	r := &skillRun{run: &envRun{}}
	fail := func(check string, harness bool, msg string) {
		r.run.problems = append(r.run.problems, problem{check: check, harness: harness, msg: msg})
	}
	sb, err := newSandbox(base, e)
	if err != nil {
		fail(checkHarness, true, err.Error())
		return r
	}
	r.run.sb = sb
	if err := sb.harness(c.mode); err != nil {
		fail(checkHarness, true, err.Error())
		return r
	}
	if err := os.MkdirAll(filepath.Join(sb.work, skillDirName), 0o755); err != nil {
		fail(checkHarness, true, err.Error())
		return r
	}
	if e == envA {
		if err := plantSkillDecoys(sb, v); err != nil {
			fail(checkHarness, true, err.Error())
			return r
		}
	}
	args := []string{"skill"}
	if c.json {
		args = append(args, "--json")
	}
	if c.harness == "" {
		args = append(args, "--print")
	} else {
		args = append(args, "--harness", c.harness, "--dir", sb.path(skillDirName))
	}
	before, err := sb.snapshot()
	if err != nil {
		fail(checkHarness, true, err.Error())
		return r
	}
	p, err := runProc(program, args, sb.cwd, sb.environ(), nil, defaultWall)
	if err != nil {
		fail(checkHarness, true, err.Error())
		return r
	}
	r.inv = &invocation{label: c.String(), args: args, stdout: p.stdout, stderr: p.stderr, proc: p}
	if p.killed {
		fail(checkWall, false, "the wall clock ran out")
		return r
	}
	if h := crashed(p.stderr); h != "" {
		r.run.problems = append(r.run.problems, crashProblem(h))
		return r
	}
	if p.exit != 0 {
		fail(checkExit, false, fmt.Sprintf("exit %d, want 0\n%s", p.exit, p.stderr))
	}
	if c.json {
		doc, err := parseEnvelope(p.stdout)
		switch {
		case err != nil:
			fail(checkHarness, true, "the tool's JSON document: "+err.Error())
		case !doc.ok:
			fail(checkOK, false, "ok is false")
		}
	}
	after, err := sb.snapshot()
	if err != nil {
		fail(checkHarness, true, err.Error())
		return r
	}
	// Check 4: exactly the SKILL.md and the directories it needs are
	// created, below the decoy directory, and nothing else changes.
	var want []string
	if c.harness != "" {
		rel := path.Join("work", skillDirName)
		for _, seg := range strings.Split(path.Join(c.dir, "SKILL.md"), "/") {
			rel = path.Join(rel, seg)
			want = append(want, rel+" was created")
		}
		slices.Sort(want)
	}
	if got := changesOutside(before, after, nil, nil); !slices.Equal(got, want) {
		fail(checkSnapshot, false, fmt.Sprintf("the sandbox's changes are %q, want %q", got, want))
	}
	sr := stepRun{invs: []*invocation{r.inv}, trees: map[string]*tree{}}
	if c.harness != "" {
		dir := filepath.Join(sb.work, skillDirName)
		t, err := readTree(dir)
		if err != nil {
			fail(checkHarness, true, err.Error())
			return r
		}
		sr.trees[skillDirName] = t
		r.written = t.files[path.Join(c.dir, "SKILL.md")].content
	}
	r.run.steps = []stepRun{sr}
	// Checks 1 and 2: the streams, and every file written anywhere in the
	// sandbox, by its path and its content.
	finders := []finder{facts.finder(v, base)}
	if e == envA {
		finders = append(finders, sentinels)
	}
	for _, f := range finders {
		r.run.problems = append(r.run.problems, scanInvocation(f, r.inv, false, nil)...)
		r.run.problems = append(r.run.problems, scanWritten(f, sb, before, after, nil)...)
	}
	return r
}

// plantSkillDecoys writes environment A's decoys for metais skill: the
// SC7 harness's decoys in the working, home and temporary directories
// (plantDecoys, plantHarnessDecoys), and in each of them a SKILL.md of
// sentinel content at every place a harness reads one.
func plantSkillDecoys(sb *sandbox, v *vector) error {
	c := &vcase{}
	if err := sb.plantDecoys(v, c); err != nil {
		return err
	}
	if err := sb.plantHarnessDecoys(v, c); err != nil {
		return err
	}
	for _, d := range []struct{ dir, token string }{
		{sb.cwd, sentinel + "CWD"},
		{sb.home, sentinel + "HOME"},
		{sb.tmp, sentinel + "TMPDECOY"},
	} {
		for _, name := range []string{".claude/skills/metais/SKILL.md", ".agents/skills/metais/SKILL.md"} {
			p := filepath.Join(d.dir, filepath.FromSlash(name))
			if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
				return err
			}
			if err := os.WriteFile(p, []byte(d.token+"\n"), 0o644); err != nil {
				return err
			}
		}
	}
	return nil
}
