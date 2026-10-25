// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"archive/tar"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"text/tabwriter"
	"time"

	"metais.org/metais/internal/tools/ux/scenario"
)

// scenariosDir is the scenario set, relative to the top of the checkout.
const scenariosDir = "internal/tools/ux/scenarios"

// errQuota stops a set on a quota pause, to be resumed.
var errQuota = errors.New("quota pause")

// config is one invocation of the runner.
type config struct {
	root       string // the top of the checkout, where every command runs
	out        string
	scenarios  []string
	models     []string
	harness    harness
	withSkill  bool
	pace       time.Duration
	quotaWait  time.Duration
	timeout    time.Duration
	allowDirty bool
	// host and transcripts are what the scorecard publishes as the host
	// and as where the transcripts are kept.
	host        string
	transcripts string
	environ     []string
	stdout      io.Writer
	// credential is -credential-file's token, which turns on the runner's
	// own credential route (ADR-009 A12), and upstream is where the route
	// sends it; "" is upstreamURL.
	credential string
	upstream   string

	now   func() time.Time
	sleep func(time.Duration)
	// start runs one command to its end, its output to stdout and stderr,
	// and returns its exit status; nil means exec.
	start func(ctx context.Context, inv invocation, stdout, stderr io.Writer) (int, error)
	// command runs a command in root, and returns its standard output;
	// nil means exec.
	command func(name string, env []string, args ...string) ([]byte, error)
}

// setState is what run prepared, shared by every run of the set.
type setState struct {
	rec      setRecord
	bin      string // the directory holding the metais under test
	fixtures string // the directory holding each scenario's packed fixtures
}

// modelName is a model as a run's directory may be named after it.
var modelName = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9.\[\]-]*$`)

// run prepares the set and runs every scenario × model not yet recorded.
func (c *config) run() error {
	if c.start == nil {
		c.start = execStart
	}
	if c.command == nil {
		c.command = func(name string, env []string, args ...string) ([]byte, error) {
			return execCommand(c.root, name, env, args...)
		}
	}
	if err := c.refuse(); err != nil {
		return err
	}
	if c.credential != "" {
		up := c.upstream
		if up == "" {
			up = upstreamURL
		}
		rt, err := startRoute(c.credential, up)
		if err != nil {
			return err
		}
		defer rt.close()
		c.harness.route(rt.url())
	}
	set, err := c.loadSet()
	if err != nil {
		return err
	}
	st, err := c.prepare(set)
	if err != nil {
		return err
	}
	ran := false
	for _, s := range set {
		for _, m := range c.models {
			dir := filepath.Join(c.out, "runs", s.Name(), m)
			r, err := readRecord(dir)
			if err != nil {
				return err
			}
			if finished(r) {
				continue
			}
			for {
				if ran {
					c.sleep(c.pace)
				}
				ran = true
				r, err = c.runOne(st, s, m, dir)
				if err != nil {
					return err
				}
				fmt.Fprintf(c.stdout, "%s %s: %s%s\n", s.Name(), m, r.Status, map[bool]string{false: ", invalid"}[r.Valid])
				if r.Status != statusQuota {
					break
				}
				if c.quotaWait <= 0 {
					c.summary(set)
					return errQuota
				}
				fmt.Fprintf(c.stdout, "quota pause: waiting %s before running it again\n", c.quotaWait)
				c.sleep(c.quotaWait)
			}
		}
	}
	if err := c.summary(set); err != nil {
		return err
	}
	fmt.Fprintln(c.stdout)
	return c.scoreAndPrint(set)
}

// loadSet reads the scenario set and selects -scenarios from it.
func (c *config) loadSet() ([]*scenario.Scenario, error) {
	set, err := scenario.Load(os.DirFS(filepath.Join(c.root, scenariosDir)))
	if err != nil {
		return nil, err
	}
	set, err = selectScenarios(set, c.scenarios)
	if err != nil {
		return nil, err
	}
	if c.withSkill {
		for _, s := range set {
			if s.Install {
				return nil, fmt.Errorf("%s starts with no metais and no skill (ADR-013 D9): run it without -with-skill", s.Name())
			}
		}
	}
	return set, nil
}

// printScorecard scores the runs already recorded and prints the
// scorecard, running nothing.
func (c *config) printScorecard() error {
	set, err := c.loadSet()
	if err != nil {
		return err
	}
	return c.scoreAndPrint(set)
}

func (c *config) scoreAndPrint(set []*scenario.Scenario) error {
	runs, err := c.scoreSet(set)
	if err != nil {
		return err
	}
	return c.scorecard(c.stdout, set, runs)
}

// refuse checks the invocation: root is the top of a metais checkout, the output directory is absolute and outside any repository, and
// the models are names.
func (c *config) refuse() error {
	gomod, err := os.ReadFile(filepath.Join(c.root, "go.mod"))
	if err != nil || !bytes.HasPrefix(gomod, []byte("module metais.org/metais\n")) {
		return errors.New("run from the top of a metais checkout")
	}
	if ch := checkOutsideRepository(c.out); !ch.Pass {
		return fmt.Errorf("-out: %s", ch.Detail)
	}
	if len(c.models) == 0 {
		return errors.New("no model")
	}
	for _, m := range c.models {
		if !modelName.MatchString(m) {
			return fmt.Errorf("model %q is not a model's name", m)
		}
	}
	return nil
}

// selectScenarios is the scenarios ids names, in the set's order, or,
// when ids is empty, every scenario but the install scenarios. An install
// scenario runs only when named: it needs the live site and a host whose
// sessions can install metais (ADR-013 D9).
func selectScenarios(set []*scenario.Scenario, ids []string) ([]*scenario.Scenario, error) {
	if len(ids) == 0 {
		return slices.DeleteFunc(slices.Clone(set), func(s *scenario.Scenario) bool { return s.Install }), nil
	}
	var out []*scenario.Scenario
	for _, s := range set {
		if slices.Contains(ids, s.ID) {
			out = append(out, s)
		}
	}
	for _, id := range ids {
		if !slices.ContainsFunc(out, func(s *scenario.Scenario) bool { return s.ID == id }) {
			return nil, fmt.Errorf("no scenario %s", id)
		}
	}
	return out, nil
}

// prepare builds metais from the commit under test, packs every fixture
// with it, and writes set.json, or, when the set resumes, requires the
// one it holds to describe the same set.
func (c *config) prepare(set []*scenario.Scenario) (*setState, error) {
	st := &setState{bin: filepath.Join(c.out, "bin"), fixtures: filepath.Join(c.out, "fixtures")}
	r := &st.rec
	r.Format = setFormat
	r.WithSkill = c.withSkill
	var err error
	if r.Harness.Version, err = c.harness.version(); err != nil {
		return nil, err
	}
	r.Harness.Name = c.harness.name()

	head, err := c.command("git", gitEnv(c.environ), "rev-parse", "HEAD")
	if err != nil {
		return nil, err
	}
	r.Commit = strings.TrimSpace(string(head))
	status, err := c.command("git", gitEnv(c.environ), "status", "--porcelain", "--untracked-files=no")
	if err != nil {
		return nil, err
	}
	if r.Dirty = len(bytes.TrimSpace(status)) > 0; r.Dirty && !c.allowDirty {
		return nil, fmt.Errorf("the checkout has uncommitted changes, so no commit is under test:\n%s", status)
	}

	if err := os.MkdirAll(st.bin, 0o755); err != nil {
		return nil, err
	}
	metais := filepath.Join(st.bin, "metais")
	// -buildvcs=false: git in the sandbox fails on an iconv locale error
	// when the go command stamps the build (R-UX-7). The commit is
	// recorded beside the binary instead.
	if _, err := c.command("go", nil, "build", "-trimpath", "-buildvcs=false", "-o", metais, "./cmd/metais"); err != nil {
		return nil, err
	}
	b, err := os.ReadFile(metais)
	if err != nil {
		return nil, err
	}
	r.Metais.SHA256 = digest(b)
	v, err := c.command(metais, nil, "version")
	if err != nil {
		return nil, err
	}
	r.Metais.Version = strings.TrimSpace(string(v))

	if err := os.RemoveAll(st.fixtures); err != nil {
		return nil, err
	}
	for _, s := range set {
		dir := filepath.Join(st.fixtures, s.Name())
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, err
		}
		for _, f := range s.Fixtures {
			if !f.Package() {
				sum, err := c.unpackTree(r.Commit, f.Tree, filepath.Join(dir, f.As))
				if err != nil {
					return nil, err
				}
				r.Fixtures = append(r.Fixtures, fixtureRecord{s.Name(), f.As + "/", sum})
				continue
			}
			// src is relative to root, where pack runs.
			src := filepath.Join(scenariosDir, s.Dir, f.Source)
			if f.Example != "" {
				src = filepath.Join("examples", f.Example)
			}
			dst := filepath.Join(dir, f.As)
			if _, err := c.command(metais, nil, "pack", src, "--out", dst); err != nil {
				return nil, err
			}
			b, err := os.ReadFile(dst)
			if err != nil {
				return nil, err
			}
			r.Fixtures = append(r.Fixtures, fixtureRecord{s.Name(), f.As, digest(b)})
		}
	}

	var prev setRecord
	switch err := readJSON(filepath.Join(c.out, setFile), &prev); {
	case errors.Is(err, os.ErrNotExist):
		r.Started = c.now().UTC().Format(time.RFC3339)
		return st, writeJSON(filepath.Join(c.out, setFile), r)
	case err != nil:
		return nil, err
	}
	if why := sameSet(prev, *r); why != "" {
		return nil, fmt.Errorf("%s holds a set of another %s; resume it with what it measured, or give a new -out", c.out, why)
	}
	r.Started = prev.Started
	return st, nil
}

// sameSet is "" when a resumed set measures what prev did, or what
// differs. The fixtures compared are those both name: a resume may run a
// scenario the first invocation did not.
func sameSet(prev, cur setRecord) string {
	switch {
	case prev.Format != cur.Format:
		return "format"
	case prev.Commit != cur.Commit || prev.Dirty != cur.Dirty:
		return "commit"
	case prev.Metais != cur.Metais:
		return "metais binary"
	case prev.Harness != cur.Harness:
		return "harness"
	case prev.WithSkill != cur.WithSkill:
		return "skill mode"
	}
	for _, f := range cur.Fixtures {
		for _, p := range prev.Fixtures {
			if p.Scenario == f.Scenario && p.Path == f.Path && p.SHA256 != f.SHA256 {
				return "fixture " + f.Scenario + "/" + f.Path
			}
		}
	}
	return ""
}

// unpackTree writes the tracked files of the directory tree of commit
// into dst, as git archive gives them, and returns the tree's digest: the
// SHA-256 of its files' paths and digests, one line each, in order. A
// tree holding anything but directories and regular files is refused.
func (c *config) unpackTree(commit, tree, dst string) (string, error) {
	b, err := c.command("git", gitEnv(c.environ), "archive", "--format=tar", commit, "--", tree)
	if err != nil {
		return "", err
	}
	var lines []string
	tr := tar.NewReader(bytes.NewReader(b))
	for {
		h, err := tr.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return "", fmt.Errorf("git archive of %s: %w", tree, err)
		}
		switch h.Typeflag {
		case tar.TypeXGlobalHeader, tar.TypeDir:
			continue
		case tar.TypeReg:
		default:
			return "", fmt.Errorf("git archive of %s: %s is not a regular file", tree, h.Name)
		}
		rel, ok := strings.CutPrefix(h.Name, tree+"/")
		if !ok || !filepath.IsLocal(rel) {
			return "", fmt.Errorf("git archive of %s: %s is outside it", tree, h.Name)
		}
		data, err := io.ReadAll(tr)
		if err != nil {
			return "", err
		}
		out := filepath.Join(dst, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
			return "", err
		}
		if err := os.WriteFile(out, data, 0o644); err != nil {
			return "", err
		}
		lines = append(lines, rel+" "+digest(data)+"\n")
	}
	if len(lines) == 0 {
		return "", fmt.Errorf("git archive of %s: the commit holds no file beneath it", tree)
	}
	slices.Sort(lines)
	return digest([]byte(strings.Join(lines, ""))), nil
}

func digest(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

// runOne runs scenario s on model m in dir, afresh, and writes its record.
func (c *config) runOne(st *setState, s *scenario.Scenario, m, dir string) (*runRecord, error) {
	if err := os.RemoveAll(dir); err != nil {
		return nil, err
	}
	ses := session{
		work:    filepath.Join(dir, workDir),
		home:    filepath.Join(dir, "home"),
		tmp:     filepath.Join(dir, "tmp"),
		bin:     st.bin,
		model:   m,
		prompt:  s.Prompt,
		skill:   c.withSkill,
		install: s.Install,
	}
	for _, d := range []string{ses.work, ses.home, ses.tmp} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			return nil, err
		}
	}
	r := &runRecord{Format: runFormat, ModelRequested: m, Harness: st.rec.Harness, WithSkill: c.withSkill,
		Commit: st.rec.Commit, MetaisSHA256: st.rec.Metais.SHA256,
		Transcript: transcriptFile, Stderr: stderrFile, Work: workDir}
	r.Scenario.ID, r.Scenario.Name, r.Scenario.Kind = s.ID, s.Name(), string(s.Kind)
	r.Scenario.AsksForOutput, r.Scenario.Prompt = s.AsksForOutput, s.Prompt

	for _, f := range s.Fixtures {
		if !f.Package() {
			tree, err := copyTree(filepath.Join(st.fixtures, s.Name(), f.As), filepath.Join(ses.work, f.As))
			if err != nil {
				return nil, err
			}
			for _, e := range tree {
				e.Path = f.As + "/" + e.Path
				r.Fixtures = append(r.Fixtures, e)
			}
			continue
		}
		b, err := os.ReadFile(filepath.Join(st.fixtures, s.Name(), f.As))
		if err != nil {
			return nil, err
		}
		if err := os.WriteFile(filepath.Join(ses.work, f.As), b, 0o644); err != nil {
			return nil, err
		}
		r.Fixtures = append(r.Fixtures, fileEntry{Path: f.As, Size: int64(len(b)), SHA256: digest(b)})
	}
	if c.withSkill {
		skill, err := c.installSkill(st, ses.work)
		if err != nil {
			return nil, err
		}
		r.Fixtures = append(r.Fixtures, skill...)
	}
	slices.SortFunc(r.Fixtures, func(a, b fileEntry) int { return strings.Compare(a.Path, b.Path) })

	inv := c.harness.invocation(ses)
	r.Invocation.Path, r.Invocation.Args, r.Invocation.Env, r.Invocation.Dir = inv.path, inv.args, inv.env, inv.dir
	onPath := checkMetaisFirstOnPath(inv, st.bin, st.rec.Metais.SHA256)
	if s.Install {
		onPath = checkMetaisNotOnPath(inv)
	}
	r.Checks = slices.Concat([]check{
		checkOutsideRepository(ses.work),
		checkFreshDirectory(ses.work, r.Fixtures),
		onPath,
	}, c.harness.checks(inv))
	if !valid(r.Checks) {
		r.Status = statusNotRun
		r.Files = changes(r.Fixtures, r.Fixtures)
		return r, writeJSON(filepath.Join(dir, recordFile), r)
	}

	// The set's directory outside the run's, as the session found it.
	before, err := outsideRun(c.out, dir)
	if err != nil {
		return nil, err
	}
	out, err := os.Create(filepath.Join(dir, transcriptFile))
	if err != nil {
		return nil, err
	}
	var stderr bytes.Buffer
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	started := c.now()
	code, runErr := c.start(ctx, inv, out, &stderr)
	ended := c.now()
	timedOut := ctx.Err() != nil
	cancel()
	if err := out.Close(); err != nil {
		return nil, err
	}
	if err := os.WriteFile(filepath.Join(dir, stderrFile), stderr.Bytes(), 0o644); err != nil {
		return nil, err
	}
	r.Started, r.Ended = started.UTC().Format(time.RFC3339Nano), ended.UTC().Format(time.RFC3339Nano)
	r.WallMS = ended.Sub(started).Milliseconds()
	r.ExitCode = code

	b, err := os.ReadFile(filepath.Join(dir, transcriptFile))
	if err != nil {
		return nil, err
	}
	t := c.harness.read(b)
	r.ModelReported, r.ModelsUsed = t.Init.Model, t.Result.Models
	r.Turns, r.CostUSD, r.DurationMS = t.Result.Turns, t.Result.CostUSD, t.Result.DurationMS
	r.IsError, r.Subtype, r.ToolUses = t.Result.IsError, t.Result.Subtype, len(t.ToolUses)
	r.Checks = append(r.Checks, c.harness.after(ses, st.rec.Harness.Version, t)...)
	afterSet, err := outsideRun(c.out, dir)
	if err != nil {
		return nil, err
	}
	r.Checks = append(r.Checks, checkWritesInsideRun(dir, ses.work, before, afterSet, t))
	if c.credential != "" {
		r.Checks = append(r.Checks, credentialNotSeen(dir, c.credential))
	}
	r.Valid = valid(r.Checks)

	switch {
	case timedOut:
		r.Status = statusTimeout
	case quotaPause(t, stderr.Bytes()):
		r.Status = statusQuota
		r.Valid = false
	case !t.Result.Found:
		r.Status = statusHarnessError
		r.Valid = false
		if runErr != nil {
			r.Checks = append(r.Checks, fail("harness-ran", "%v", runErr))
		}
	default:
		r.Status = statusComplete
	}
	after, err := listFiles(ses.work)
	if err != nil {
		return nil, err
	}
	r.Files = changes(r.Fixtures, after)
	if err := writeJSON(filepath.Join(dir, recordFile), r); err != nil {
		return nil, err
	}
	// A run that holds the credential stops the set: the file it names is
	// removed before anything is scrubbed or published.
	if i := slices.IndexFunc(r.Checks, func(ch check) bool { return ch.Name == checkCredentialNotSeen }); i >= 0 && !r.Checks[i].Pass {
		return nil, fmt.Errorf("%s %s: %s: %s, beneath %s; remove it before scrubbing or publishing the set", s.Name(), m, checkCredentialNotSeen, r.Checks[i].Detail, dir)
	}
	return r, nil
}

// copyTree copies every file of the directory src into dst, and lists
// them.
func copyTree(src, dst string) ([]fileEntry, error) {
	files, err := listFiles(src)
	if err != nil {
		return nil, err
	}
	for _, f := range files {
		b, err := os.ReadFile(filepath.Join(src, filepath.FromSlash(f.Path)))
		if err != nil {
			return nil, err
		}
		out := filepath.Join(dst, filepath.FromSlash(f.Path))
		if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
			return nil, err
		}
		if err := os.WriteFile(out, b, 0o644); err != nil {
			return nil, err
		}
	}
	return files, nil
}

// installSkill writes metais's skill for the harness beneath work, with
// the metais under test, before the session starts, and lists what it
// wrote.
func (c *config) installSkill(st *setState, work string) ([]fileEntry, error) {
	if _, err := c.command(filepath.Join(st.bin, "metais"), nil, "skill", "--harness", "claude", "--dir", work); err != nil {
		return nil, err
	}
	files, err := listFiles(work)
	if err != nil {
		return nil, err
	}
	var skill []fileEntry
	for _, f := range files {
		if strings.HasPrefix(f.Path, ".claude/skills/metais/") {
			skill = append(skill, f)
		}
	}
	if len(skill) == 0 {
		return nil, errors.New("metais skill wrote nothing under .claude/skills/metais")
	}
	return skill, nil
}

// summary prints every run of the set recorded so far.
func (c *config) summary(set []*scenario.Scenario) error {
	w := tabwriter.NewWriter(c.stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "scenario\tmodel\treported\tstatus\tvalid\tturns\ttime\tcost")
	var runs, ok, invalid, pending int
	var cost float64
	for _, s := range set {
		for _, m := range c.models {
			r, err := readRecord(filepath.Join(c.out, "runs", s.Name(), m))
			if err != nil {
				return err
			}
			if !finished(r) {
				pending++
				fmt.Fprintf(w, "%s\t%s\t\tto run\t\t\t\t\n", s.Name(), m)
				continue
			}
			runs++
			if r.Valid {
				ok++
			} else {
				invalid++
			}
			cost += r.CostUSD
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%d\t%s\t$%.4f\n", s.Name(), m, r.ModelReported, r.Status,
				map[bool]string{true: "yes", false: "no"}[r.Valid], r.Turns,
				(time.Duration(r.WallMS) * time.Millisecond).Round(time.Second), r.CostUSD)
		}
	}
	if err := w.Flush(); err != nil {
		return err
	}
	fmt.Fprintf(c.stdout, "%d runs recorded: %d valid, %d invalid; %d to run; $%.4f in all\n", runs, ok, invalid, pending, cost)
	return nil
}

// execStart runs inv to its end. A context that ends kills it.
func execStart(ctx context.Context, inv invocation, stdout, stderr io.Writer) (int, error) {
	cmd := exec.CommandContext(ctx, inv.path, inv.args...)
	cmd.Dir, cmd.Env = inv.dir, inv.env
	cmd.Stdout, cmd.Stderr = stdout, stderr
	cmd.WaitDelay = 10 * time.Second
	err := cmd.Run()
	if cmd.ProcessState != nil {
		return cmd.ProcessState.ExitCode(), err
	}
	return -1, err
}

// execCommand runs name in dir, with env (nil: the runner's own), and
// names it and its standard error when it fails.
func execCommand(dir, name string, env []string, args ...string) ([]byte, error) {
	cmd := exec.Command(name, args...)
	cmd.Dir, cmd.Env = dir, env
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("%s: %w\n%s", strings.Join(cmd.Args, " "), err, strings.TrimSpace(stderr.String()))
	}
	return stdout.Bytes(), nil
}

// gitEnv is environ with no GIT_* variable of the host's, and with
// core.precomposeunicode off, which git on darwin needs where iconv cannot
// open its UTF-8-MAC converter.
func gitEnv(environ []string) []string {
	env := slices.DeleteFunc(slices.Clone(environ), func(kv string) bool {
		return strings.HasPrefix(kv, "GIT_")
	})
	return append(env,
		"GIT_CONFIG_COUNT=1",
		"GIT_CONFIG_KEY_0=core.precomposeunicode",
		"GIT_CONFIG_VALUE_0=false",
	)
}
